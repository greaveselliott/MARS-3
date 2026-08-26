/*
FactoryDocSync:
docs:
- docs/features/F-001-doctrine-foundation.md
- docs/design-docs/mars-provenance.md
- docs/code-documentation-map.md
*/

package doctrine

import (
	"bufio"
	"errors"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const factoryDocSyncMarker = "FactoryDocSync"

var documentationPathPattern = regexp.MustCompile(`docs/[A-Za-z0-9][A-Za-z0-9._/-]*\.md`)

type docSyncRule struct {
	Prefix       string
	RequiredDocs []string
}

type docSyncConfig struct {
	Marker           string
	SourceExtensions []string
	Rules            []docSyncRule
	Excluded         []string
}

// AuditDocSync verifies structural documentation ownership. Passing this audit
// means links exist and required prefixes are covered; it is not a claim of
// semantic correctness.
func AuditDocSync(repo string) ([]Finding, error) {
	root, err := repositoryRoot(repo)
	if err != nil {
		return nil, err
	}
	config, configPath, err := loadDocSyncConfig(root)
	if err != nil {
		return []Finding{{Path: configPath, Code: "docsync.config", Message: err.Error()}}, nil
	}
	var findings []Finding
	for _, rule := range config.Rules {
		for _, document := range rule.RequiredDocs {
			if !repoFileExists(root, document) {
				addFinding(&findings, configPath, "docsync.config_target_missing", "prefix %q requires missing document %s", rule.Prefix, document)
			}
		}
	}
	paths, err := walkRepository(root)
	if err != nil {
		return nil, err
	}
	for _, path := range paths {
		if !isDocSyncSource(path, config.SourceExtensions) || hasPathPrefix(path, config.Excluded) {
			continue
		}
		matchingRules := matchingDocSyncRules(path, config.Rules)
		mapped := len(matchingRules) > 0
		if !mapped {
			addFinding(&findings, path, "docsync.unmapped_source", "configured source file is outside the canonical prefix map")
			continue
		}
		requiredDocs := requiredDocsForRules(matchingRules)
		data, err := readAuditedFile(root, path)
		if err != nil {
			addFinding(&findings, path, "docsync.read", "%v", err)
			continue
		}
		docs, markerCount, validMarkerCount := parseDocSyncMarkers(path, data, config.Marker)
		if markerCount == 0 {
			addFinding(&findings, path, "docsync.marker_missing", "%s metadata is required", config.Marker)
			continue
		}
		if markerCount != 1 {
			addFinding(&findings, path, "docsync.marker_cardinality", "exactly one %s marker is required", config.Marker)
		}
		if validMarkerCount != 1 {
			addFinding(&findings, path, "docsync.marker_malformed", "%s metadata must use a structured docs field", config.Marker)
		}
		if markerIndex := strings.Index(string(data), config.Marker); markerIndex < 0 || markerIndex > 4096 {
			addFinding(&findings, path, "docsync.marker_placement", "%s metadata must appear in the file header", config.Marker)
		}
		if len(docs) == 0 {
			addFinding(&findings, path, "docsync.docs_missing", "%s must name at least one documentation file", config.Marker)
		}
		for _, required := range requiredDocs {
			if !containsString(docs, required) {
				addFinding(&findings, path, "docsync.prefix_requirement", "matching prefix requirements include %s", required)
			}
		}
		for _, document := range docs {
			if !safeRelativePath(document) || !strings.HasPrefix(document, "docs/") {
				addFinding(&findings, path, "docsync.unsafe_target", "documentation target %q is unsafe", document)
				continue
			}
			if !repoFileExists(root, document) {
				addFinding(&findings, path, "docsync.target_missing", "documentation target %s does not exist", document)
			}
		}
	}
	sortFindings(findings)
	return findings, nil
}

func loadDocSyncConfig(root string) (docSyncConfig, string, error) {
	const yamlPath = ".harness/docsync.yaml"
	if data, err := readRepoFile(root, yamlPath); err == nil {
		config, parseErr := parseDocSyncYAML(data)
		return config, yamlPath, parseErr
	}
	const markdownPath = "docs/code-documentation-map.md"
	if data, err := readRepoFile(root, markdownPath); err == nil {
		config, parseErr := parseDocSyncMarkdown(data)
		return config, markdownPath, parseErr
	}
	return docSyncConfig{}, yamlPath, errors.New("canonical DocSync config or prefix map is required")
}

func parseDocSyncYAML(data []byte) (docSyncConfig, error) {
	config := docSyncConfig{Marker: factoryDocSyncMarker}
	values := yamlScalars(data)
	if version := scalar(values, "schemaVersion", "schema_version"); version != "1" {
		return config, errors.New("DocSync schemaVersion must be 1")
	}
	if marker := scalar(values, "marker"); marker != "" {
		config.Marker = marker
	}
	if config.Marker != factoryDocSyncMarker {
		return config, errors.New("DocSync marker must be FactoryDocSync")
	}

	section := ""
	var current *docSyncRule
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), " \t\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !strings.HasPrefix(trimmed, "-") && strings.HasSuffix(trimmed, ":") {
			key := normalizeKey(strings.TrimSuffix(trimmed, ":"))
			switch key {
			case "sourceextensions":
				section = "extensions"
				current = nil
			case "prefixes", "sourceprefixes", "prefixrequirements", "mappings":
				section = "rules"
				current = nil
			case "excluded", "excludedprefixes", "exclusions":
				section = "excluded"
				current = nil
			case "docs", "requireddocs":
				if current != nil {
					section = "rule-docs"
				}
			}
			continue
		}
		if strings.HasPrefix(trimmed, "-") {
			item := strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
			if section == "extensions" {
				config.SourceExtensions = append(config.SourceExtensions, trimYAMLScalar(item))
				continue
			}
			if section == "excluded" {
				config.Excluded = append(config.Excluded, trimYAMLScalar(item))
				continue
			}
			if section == "rule-docs" && current != nil && !strings.Contains(item, ":") {
				current.RequiredDocs = append(current.RequiredDocs, trimYAMLScalar(item))
				continue
			}
			fields := strings.SplitN(item, ":", 2)
			if len(fields) == 2 && normalizeKey(fields[0]) == "prefix" {
				config.Rules = append(config.Rules, docSyncRule{Prefix: trimYAMLScalar(fields[1])})
				current = &config.Rules[len(config.Rules)-1]
				section = "rules"
			}
			continue
		}
		if current == nil {
			continue
		}
		fields := strings.SplitN(trimmed, ":", 2)
		if len(fields) != 2 {
			continue
		}
		switch normalizeKey(fields[0]) {
		case "prefix":
			current.Prefix = trimYAMLScalar(fields[1])
		case "docs", "requireddocs":
			inline := strings.TrimSpace(fields[1])
			if inline == "" {
				section = "rule-docs"
			} else {
				current.RequiredDocs = append(current.RequiredDocs, parseInlineYAMLList(inline)...)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return config, err
	}
	return normalizeDocSyncConfig(config)
}

func parseInlineYAMLList(value string) []string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(strings.TrimSuffix(value, "]"), "[")
	var result []string
	for _, item := range strings.Split(value, ",") {
		item = trimYAMLScalar(item)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

func parseDocSyncMarkdown(data []byte) (docSyncConfig, error) {
	config := docSyncConfig{
		Marker: factoryDocSyncMarker,
		SourceExtensions: []string{
			".go", ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs", ".jsonc", ".yaml", ".yml", ".html", ".css", ".scss", ".sql",
		},
	}
	lines := strings.Split(string(data), "\n")
	for index := 0; index+2 < len(lines); index++ {
		headers := splitMarkdownRow(lines[index])
		separators := splitMarkdownRow(lines[index+1])
		if len(headers) != len(separators) || len(headers) < 2 {
			continue
		}
		columns := make(map[string]int)
		for column, header := range headers {
			columns[normalizeKey(header)] = column
		}
		prefixColumn, hasPrefix := firstTableColumn(columns, "sourceprefix", "prefix")
		docsColumn, hasDocs := firstTableColumn(columns, "requireddocs", "documentation", "docs")
		if !hasPrefix || !hasDocs {
			continue
		}
		for rowIndex := index + 2; rowIndex < len(lines); rowIndex++ {
			row := splitMarkdownRow(lines[rowIndex])
			if len(row) != len(headers) {
				break
			}
			prefix := firstCodeSpan(row[prefixColumn])
			docs := documentationPathPattern.FindAllString(row[docsColumn], -1)
			if prefix != "" {
				config.Rules = append(config.Rules, docSyncRule{Prefix: prefix, RequiredDocs: docs})
			}
		}
		break
	}
	return normalizeDocSyncConfig(config)
}

func normalizeDocSyncConfig(config docSyncConfig) (docSyncConfig, error) {
	if len(config.SourceExtensions) == 0 {
		return config, errors.New("DocSync source_extensions must contain at least one extension")
	}
	for _, extension := range config.SourceExtensions {
		if !strings.HasPrefix(extension, ".") || strings.ContainsAny(extension, "/\\") {
			return config, errors.New("DocSync source_extensions contains an invalid extension")
		}
	}
	config.SourceExtensions = uniqueSorted(config.SourceExtensions)
	if len(config.Rules) == 0 {
		return config, errors.New("DocSync prefix map must contain at least one rule")
	}
	seen := make(map[string]bool)
	for index := range config.Rules {
		rule := &config.Rules[index]
		rule.Prefix = cleanPublicPath(rule.Prefix)
		if !strings.HasSuffix(rule.Prefix, "/") {
			rule.Prefix += "/"
		}
		if !safeRelativePrefix(rule.Prefix) {
			return config, errors.New("DocSync prefix map contains an unsafe prefix")
		}
		if seen[rule.Prefix] {
			return config, errors.New("DocSync prefix map contains a duplicate prefix")
		}
		seen[rule.Prefix] = true
		rule.RequiredDocs = uniqueSorted(rule.RequiredDocs)
		if len(rule.RequiredDocs) == 0 {
			return config, errors.New("every DocSync prefix must require at least one document")
		}
		for _, document := range rule.RequiredDocs {
			if !safeRelativePath(document) || !strings.HasPrefix(document, "docs/") {
				return config, errors.New("DocSync prefix map contains an unsafe documentation path")
			}
		}
	}
	for index := range config.Excluded {
		config.Excluded[index] = cleanPublicPath(config.Excluded[index])
		if !strings.HasSuffix(config.Excluded[index], "/") {
			config.Excluded[index] += "/"
		}
	}
	sort.Slice(config.Rules, func(i, j int) bool { return config.Rules[i].Prefix < config.Rules[j].Prefix })
	config.Excluded = uniqueSorted(config.Excluded)
	return config, nil
}

func safeRelativePrefix(prefix string) bool {
	return prefix != "/" && !filepath.IsAbs(prefix) && !strings.Contains(prefix, "..") && !strings.Contains(prefix, "\\")
}

func firstCodeSpan(value string) string {
	start := strings.IndexByte(value, '`')
	if start < 0 {
		return strings.TrimSpace(value)
	}
	end := strings.IndexByte(value[start+1:], '`')
	if end < 0 {
		return ""
	}
	return value[start+1 : start+1+end]
}

func isDocSyncSource(path string, extensions []string) bool {
	extension := strings.ToLower(filepath.Ext(path))
	for _, configured := range extensions {
		if extension == strings.ToLower(configured) {
			return true
		}
	}
	return false
}

func hasPathPrefix(path string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func matchingDocSyncRules(path string, rules []docSyncRule) []docSyncRule {
	var matches []docSyncRule
	for _, rule := range rules {
		if strings.HasPrefix(path, rule.Prefix) {
			matches = append(matches, rule)
		}
	}
	return matches
}

func requiredDocsForRules(rules []docSyncRule) []string {
	var required []string
	for _, rule := range rules {
		required = append(required, rule.RequiredDocs...)
	}
	return uniqueSorted(required)
}

func parseDocSyncMarkers(path string, data []byte, marker string) ([]string, int, int) {
	comments := commentBodies(path, data)
	set := make(map[string]bool)
	count := 0
	valid := 0
	for _, comment := range comments {
		if !strings.Contains(comment, marker) {
			continue
		}
		count++
		documents, ok := parseStructuredDocSyncComment(comment, marker)
		if !ok {
			continue
		}
		valid++
		for _, document := range documents {
			set[document] = true
		}
	}
	result := make([]string, 0, len(set))
	for document := range set {
		result = append(result, document)
	}
	sort.Strings(result)
	return result, count, valid
}

func parseStructuredDocSyncComment(comment, marker string) ([]string, bool) {
	index := strings.Index(comment, marker)
	if index < 0 {
		return nil, false
	}
	tail := strings.TrimSpace(comment[index+len(marker):])
	if !strings.HasPrefix(tail, ":") {
		return nil, false
	}
	tail = strings.TrimSpace(strings.TrimPrefix(tail, ":"))
	if strings.HasPrefix(tail, "{") {
		var compact struct {
			Docs []string `json:"docs"`
		}
		if err := decodeStrictJSON([]byte(tail), &compact); err != nil || len(compact.Docs) == 0 {
			return nil, false
		}
		return uniqueSorted(compact.Docs), true
	}
	lines := strings.Split(tail, "\n")
	seenDocs := false
	var documents []string
	for _, line := range lines {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "*"))
		if line == "docs:" {
			if seenDocs {
				return nil, false
			}
			seenDocs = true
			continue
		}
		if !seenDocs || line == "" {
			continue
		}
		if !strings.HasPrefix(line, "-") {
			return nil, false
		}
		document := strings.TrimSpace(strings.TrimPrefix(line, "-"))
		if !documentationPathPattern.MatchString(document) || documentationPathPattern.FindString(document) != document {
			return nil, false
		}
		documents = append(documents, document)
	}
	return uniqueSorted(documents), seenDocs && len(documents) > 0
}

func commentBodies(path string, data []byte) []string {
	extension := strings.ToLower(filepath.Ext(path))
	switch extension {
	case ".yaml", ".yml":
		return yamlCommentBodies(string(data))
	case ".html", ".htm":
		return delimitedComments(string(data), "<!--", "-->")
	case ".sql":
		return sqlCommentBodies(string(data))
	default:
		return cStyleCommentBodies(string(data))
	}
}

func sqlCommentBodies(content string) []string {
	var comments []string
	var group []string
	flush := func() {
		if len(group) > 0 {
			comments = append(comments, strings.Join(group, "\n"))
			group = nil
		}
	}
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") {
			group = append(group, strings.TrimSpace(strings.TrimPrefix(trimmed, "--")))
		} else {
			flush()
		}
	}
	flush()
	comments = append(comments, delimitedComments(content, "/*", "*/")...)
	return comments
}

func yamlCommentBodies(content string) []string {
	var comments []string
	var group []string
	flush := func() {
		if len(group) > 0 {
			comments = append(comments, strings.Join(group, "\n"))
			group = nil
		}
	}
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			group = append(group, strings.TrimSpace(strings.TrimPrefix(trimmed, "#")))
		} else {
			flush()
		}
	}
	flush()
	return comments
}

func delimitedComments(content, opening, closing string) []string {
	var comments []string
	for {
		start := strings.Index(content, opening)
		if start < 0 {
			return comments
		}
		content = content[start+len(opening):]
		end := strings.Index(content, closing)
		if end < 0 {
			return comments
		}
		comments = append(comments, content[:end])
		content = content[end+len(closing):]
	}
}

func cStyleCommentBodies(content string) []string {
	var comments []string
	for index := 0; index < len(content); {
		switch content[index] {
		case '"', '\'', '`':
			quote := content[index]
			index++
			for index < len(content) {
				if content[index] == '\\' && quote != '`' {
					index += 2
					continue
				}
				if content[index] == quote {
					index++
					break
				}
				index++
			}
		case '/':
			if index+1 >= len(content) {
				index++
				continue
			}
			if content[index+1] == '/' {
				var group []string
				for {
					start := index + 2
					end := strings.IndexByte(content[start:], '\n')
					if end < 0 {
						group = append(group, content[start:])
						comments = append(comments, strings.Join(group, "\n"))
						return comments
					}
					group = append(group, content[start:start+end])
					index = start + end + 1
					next := index
					for next < len(content) && (content[next] == ' ' || content[next] == '\t') {
						next++
					}
					if next+1 >= len(content) || content[next] != '/' || content[next+1] != '/' {
						break
					}
					index = next
				}
				comments = append(comments, strings.Join(group, "\n"))
				continue
			}
			if content[index+1] == '*' {
				start := index + 2
				end := strings.Index(content[start:], "*/")
				if end < 0 {
					return comments
				}
				comments = append(comments, content[start:start+end])
				index = start + end + 2
				continue
			}
			index++
		default:
			index++
		}
	}
	return comments
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func uniqueSorted(values []string) []string {
	seen := make(map[string]bool)
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			seen[value] = true
		}
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
