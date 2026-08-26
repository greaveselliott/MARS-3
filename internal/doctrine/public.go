/*
FactoryDocSync:
docs:
- docs/features/F-001-doctrine-foundation.md
- docs/design-docs/ADR-004-pr-first-publication.md
- docs/design-docs/mars-provenance.md
- docs/code-documentation-map.md
*/

package doctrine

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"net/mail"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

var (
	highConfidenceSecrets = []*regexp.Regexp{
		regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9]{30,}\b`),
		regexp.MustCompile(`\bgithub_pat_[A-Za-z0-9_]{40,}\b`),
		regexp.MustCompile(`\bsk-[A-Za-z0-9_-]{24,}\b`),
		regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`),
		regexp.MustCompile(`(?i)\bbearer\s+[A-Za-z0-9._~+/=-]{24,}`),
		regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH |DSA )?PRIVATE KEY-----`),
	}
	emailPattern       = regexp.MustCompile(`[A-Za-z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`)
	remoteImagePattern = regexp.MustCompile(`!\[[^\]]*\]\(\s*https?://`)
	privateIPv4Pattern = regexp.MustCompile(`\b(?:10(?:\.\d{1,3}){3}|192\.168(?:\.\d{1,3}){2}|172\.(?:1[6-9]|2\d|3[01])(?:\.\d{1,3}){2})\b`)
	hostFieldPattern   = regexp.MustCompile(`(?i)\b(?:host(?:name)?|machine[_-]?id|trace[_-]?backend(?:[_-]?url)?)\s*[:=]\s*["']?([A-Za-z0-9._:/-]+)`)
	unsafeMarkup       = regexp.MustCompile(`(?i)(<\s*(?:script|iframe|object|embed|foreignObject)\b|\bon(?:error|load|click)\s*=|javascript\s*:|data\s*:\s*text/html)`)
	containerReference = regexp.MustCompile(`(?:docker://|ghcr\.io/|docker\.io/)[A-Za-z0-9._/@:-]+`)
	containerDigest    = regexp.MustCompile(`@sha256:[0-9a-f]{64}$`)
	dockerCommand      = regexp.MustCompile(`\bdocker\s+([A-Za-z][A-Za-z0-9-]*)\b`)
	secretWord         = regexp.MustCompile(`(?i)\bsecrets\b`)
	githubTokenWord    = regexp.MustCompile(`(?i)\bgithub\s*(?:\.\s*token|\[\s*["']token["']\s*\])`)
)

var allowedWorkflowActions = map[string]bool{
	"actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683": true,
	"actions/setup-go@d35c59abb061a4a6fb18e82ac0862c26744d6ab5": true,
}

var allowedWorkflowExpressions = map[string]int{
	"github.workflow": 1,
	"github.event.pull_request.number || github.ref": 1,
}

const allowedWorkflowContainer = "docker.io/zricethezav/gitleaks@sha256:75bdb2b2f4db213cde0b8295f13a88d6b333091bbfbf3012a4e083d00d31caba"

const (
	canonicalFoundationWorkflowPath   = ".github/workflows/foundation-quality.yml"
	canonicalFoundationWorkflowSHA256 = "b087a9bacc60f895aa00d58c34bd4b3791500762330addee84691ddc7dda2c62"
)

var forbiddenBasenames = map[string]bool{
	".ds_store":           true,
	"cookies":             true,
	"cookies.sqlite":      true,
	"credentials":         true,
	"credentials.json":    true,
	"id_rsa":              true,
	"id_ed25519":          true,
	"known_hosts":         true,
	"login data":          true,
	"web data":            true,
	"local state":         true,
	"keychain-export.txt": true,
}

var forbiddenExtensions = map[string]bool{
	".cast":            true,
	".cer":             true,
	".crt":             true,
	".db":              true,
	".kdbx":            true,
	".key":             true,
	".mobileprovision": true,
	".p12":             true,
	".pem":             true,
	".pfx":             true,
	".sqlite":          true,
	".sqlite3":         true,
}

// CheckPublic enforces the public-from-first-commit content boundary. It
// reports repository-relative findings and never emits matched secret bytes.
func CheckPublic(repo string) ([]Finding, error) {
	root, err := repositoryRoot(repo)
	if err != nil {
		return nil, err
	}
	var findings []Finding
	checkWave1PlanningGrant(root, &findings)
	checkRequiredPublicMetadata(root, &findings)
	checkSymlinks(root, &findings)
	declarations := loadGeneratedDeclarations(root, &findings)
	paths, err := walkPublicRepository(root)
	if err != nil {
		return nil, err
	}
	for _, path := range paths {
		checkPublicPath(path, &findings)
		checkGovernedPublicScope(path, &findings)
		data, err := readAuditedFile(root, path)
		if err != nil {
			addFinding(&findings, path, "public.read", "%v", err)
			continue
		}
		checkPublicContent(root, path, data, &findings)
		if isGeneratedPath(path) && path != ".harness/generated/generated-files.json" {
			if _, ok := declarations[path]; !ok {
				addFinding(&findings, path, "public.generated_undeclared", "generated file needs a source and reproducible command declaration")
			}
		}
	}
	for path := range declarations {
		if !repoFileExists(root, path) {
			addFinding(&findings, ".harness/generated/generated-files.json", "public.generated_missing", "declaration names missing file %s", path)
		}
	}
	sortFindings(findings)
	return findings, nil
}

func checkGovernedPublicScope(path string, findings *[]Finding) {
	if !safeRelativePath(path) || cleanPublicPath(path) != path {
		addFinding(findings, path, "public.scope", "path is not a canonical repository-relative governed path")
		return
	}
	for _, prefix := range []string{
		".github/",
		".harness/",
		"api/authority/",
		"charts/",
		"cmd/mars3/",
		"cmd/mars3-authority/",
		"cmd/mars3-platform/",
		"database/authority/",
		"database/platform/",
		"deploy/authority/",
		"deploy/platform/",
		"docs/",
		"internal/authority/",
		"internal/doctrine/",
		"internal/platform/",
	} {
		if strings.HasPrefix(path, prefix) {
			return
		}
	}
	for _, allowed := range []string{
		".gitattributes", ".gitignore", "AGENTS.md", "CODE_OF_CONDUCT.md", "CONTRIBUTING.md", "LICENSE", "Makefile", "NOTICE", "README.md", "SECURITY.md", "THIRD_PARTY_NOTICES", "go.mod", "go.sum",
	} {
		if path == allowed {
			return
		}
	}
	addFinding(findings, path, "public.scope", "path is outside the governed public foundation and Wave-1 source roots")
}

func checkRequiredPublicMetadata(root string, findings *[]Finding) {
	required := []string{
		"LICENSE", "NOTICE", "THIRD_PARTY_NOTICES", "README.md", "SECURITY.md", "CONTRIBUTING.md", "CODE_OF_CONDUCT.md", "AGENTS.md",
		".github/pull_request_template.md", ".github/dependabot.yml", ".github/ISSUE_TEMPLATE/config.yml", ".github/ISSUE_TEMPLATE/bug-report.yml",
		".github/ISSUE_TEMPLATE/foundation-finding.yml", ".github/workflows/foundation-quality.yml",
		".harness/genesis.yaml", ".harness/genesis.yaml.sig", ".harness/manifest.yaml", ".harness/docsync.yaml",
		".harness/claims/H-001.yaml", ".harness/claims/H-001.yaml.sig",
		".harness/generated/genesis-effect-chain.json", ".harness/generated/generated-files.json", ".harness/generated/mars/source-manifest.json",
		"docs/goals/active.md", "docs/product-decisions/PD-001-public-first.md", "docs/product-decisions/PD-002-git-beads-authority.md",
		"docs/product-decisions/PD-003-provider-neutral.md", "docs/product-specs/foundation.md", "docs/features/F-001-doctrine-foundation.md",
		"docs/exec-plans/active/current-operating-plan.md", "docs/design-docs/ADR-002-trace-spine.md", "docs/design-docs/ADR-003-rule-of-two.md",
	}
	for _, path := range required {
		if !repoFileExists(root, path) {
			addFinding(findings, path, "public.metadata_missing", "required public license or provenance metadata is missing")
		}
	}
	if data, err := readRepoFile(root, "LICENSE"); err == nil && !bytes.Contains(data, []byte("Apache License")) {
		addFinding(findings, "LICENSE", "public.license", "LICENSE must contain the Apache License")
	}
	for _, path := range []string{"NOTICE", "THIRD_PARTY_NOTICES", "SECURITY.md", "CONTRIBUTING.md"} {
		if data, err := readRepoFile(root, path); err == nil && len(bytes.TrimSpace(data)) < 40 {
			addFinding(findings, path, "public.metadata_empty", "required public governance metadata is incomplete")
		}
	}
}

func checkSymlinks(root string, findings *[]Finding) {
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		relative = cleanPublicPath(relative)
		if entry.IsDir() && filepath.Base(relative) == ".git" {
			return filepath.SkipDir
		}
		if entry.Type()&os.ModeSymlink != 0 {
			addFinding(findings, relative, "public.symlink", "symbolic links are prohibited in the public foundation")
		}
		return nil
	})
}

func checkPublicPath(path string, findings *[]Finding) {
	lower := strings.ToLower(path)
	base := strings.ToLower(filepath.Base(path))
	extension := strings.ToLower(filepath.Ext(path))
	if strings.HasPrefix(base, ".env") || forbiddenBasenames[base] || forbiddenExtensions[extension] {
		addFinding(findings, path, "public.forbidden_file", "file type or name is prohibited")
	}
	for _, segment := range strings.Split(lower, "/") {
		switch segment {
		case ".beads", ".dolt", "browser-profile", "browser-profiles", "keychain", "object-store", "postgres-data", "temporal-data", "kubernetes-secrets":
			addFinding(findings, path, "public.forbidden_state", "local authority, credential, or runtime state is prohibited")
			return
		}
	}
}

func checkPublicContent(root, path string, data []byte, findings *[]Finding) {
	if !utf8.Valid(data) || bytes.IndexByte(data, 0) >= 0 {
		addFinding(findings, path, "public.binary", "binary and unscannable content is prohibited in governed public source")
		return
	}
	content := string(data)
	for _, pattern := range highConfidenceSecrets {
		if pattern.MatchString(content) {
			addFinding(findings, path, "public.secret", "high-confidence secret material detected; matched bytes are redacted")
			break
		}
	}
	if containsDeveloperPath(content) {
		addFinding(findings, path, "public.developer_path", "absolute developer or machine-specific path detected")
	}
	checkPublicIdentity(path, content, findings)
	checkProhibitedRawFields(path, content, findings)
	checkMachineMetadata(path, content, findings)
	if isFixturePath(path) {
		checkSyntheticFixture(path, content, findings)
	}
	if isMarkupPath(path) && unsafeMarkup.MatchString(content) {
		addFinding(findings, path, "public.unsafe_markup", "unsafe executable markup detected")
	}
	if isFixturePath(path) && strings.EqualFold(filepath.Ext(path), ".md") {
		if remoteImagePattern.MatchString(content) {
			addFinding(findings, path, "public.remote_image_fixture", "remote images are prohibited in Markdown fixtures")
		}
		if hasExecutableFence(content) {
			addFinding(findings, path, "public.executable_markdown", "executable shell fences are prohibited in Markdown fixtures")
		}
	}
	workflowPath := strings.ToLower(filepath.ToSlash(path))
	if strings.HasPrefix(workflowPath, ".github/workflows/") && (strings.HasSuffix(workflowPath, ".yaml") || strings.HasSuffix(workflowPath, ".yml")) {
		checkWorkflow(path, content, findings)
	}
	if strings.EqualFold(filepath.Ext(path), ".md") {
		if info, err := os.Stat(filepath.Join(root, filepath.FromSlash(path))); err == nil && info.Mode().Perm()&0o111 != 0 {
			addFinding(findings, path, "public.executable_markdown", "Markdown files must not be executable")
		}
	}
}

func checkMachineMetadata(path, content string, findings *[]Finding) {
	if privateIPv4Pattern.MatchString(content) {
		addFinding(findings, path, "public.private_host", "repository text contains private network metadata")
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json", ".jsonc", ".md", ".toml", ".txt", ".yaml", ".yml":
	default:
		return
	}
	for _, match := range hostFieldPattern.FindAllStringSubmatch(content, -1) {
		if !syntheticHostValue(match[1]) {
			addFinding(findings, path, "public.host_metadata", "repository text contains identifying host or backend metadata")
			return
		}
	}
}

func checkPublicIdentity(path, content string, findings *[]Finding) {
	for _, candidate := range emailPattern.FindAllString(content, -1) {
		address, err := mail.ParseAddress(candidate)
		if err != nil {
			continue
		}
		parts := strings.SplitN(address.Address, "@", 2)
		if len(parts) != 2 || !isReservedDomain(strings.ToLower(parts[1])) {
			addFinding(findings, path, "public.identity", "repository text contains a non-synthetic email identity")
			return
		}
	}
}

func checkProhibitedRawFields(path, content string, findings *[]Finding) {
	lower := strings.ToLower(content)
	for _, field := range []string{"raw_prompt", "rawprompt", "raw_completion", "rawcompletion", "chain_of_thought", "chainofthought", "tool_payload", "toolpayload", "provider_session_state", "providersessionstate", "cookie", "session_token"} {
		fieldPattern := regexp.MustCompile(`(?m)["']?` + regexp.QuoteMeta(field) + `["']?\s*[:=]`)
		if fieldPattern.MatchString(lower) {
			addFinding(findings, path, "public.raw_payload", "repository text contains a prohibited raw payload field")
			return
		}
	}
}

func containsDeveloperPath(content string) bool {
	boundary := `(?:^|[\s"'` + "`" + `(=])`
	unixHome := regexp.MustCompile(boundary + `/` + `(?:Users|home)` + `/[A-Za-z0-9._-]+(?:/|\b)`)
	macTemp := regexp.MustCompile(boundary + `/` + `(?:private/)?var/folders/[A-Za-z0-9._/-]+`)
	windowsHome := regexp.MustCompile(`(?i)` + boundary + `[A-Z]:\\` + `Users\\[A-Za-z0-9._-]+\\`)
	return unixHome.MatchString(content) || macTemp.MatchString(content) || windowsHome.MatchString(content)
}

func isFixturePath(path string) bool {
	lower := "/" + strings.ToLower(path) + "/"
	for _, marker := range []string{"/fixture/", "/fixtures/", "/testdata/", "/samples/"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func isEvidencePath(path string) bool {
	lower := "/" + strings.ToLower(path) + "/"
	base := strings.ToLower(filepath.Base(path))
	return strings.Contains(lower, "/evidence/") || strings.Contains(lower, "/traces/") || strings.Contains(lower, "/reports/") || strings.HasPrefix(base, "evidence.") || strings.Contains(base, "-evidence.")
}

func checkSyntheticFixture(path, content string, findings *[]Finding) {
	for _, candidate := range emailPattern.FindAllString(content, -1) {
		address, err := mail.ParseAddress(candidate)
		if err != nil {
			continue
		}
		domain := strings.ToLower(strings.SplitN(address.Address, "@", 2)[1])
		if !isReservedDomain(domain) {
			addFinding(findings, path, "public.fixture_identity", "fixture email must use an RFC-reserved synthetic domain")
			break
		}
	}
	if privateIPv4Pattern.MatchString(content) {
		addFinding(findings, path, "public.fixture_host", "fixture contains private network metadata")
	}
	for _, match := range hostFieldPattern.FindAllStringSubmatch(content, -1) {
		value := strings.Trim(match[1], "\"'")
		if !syntheticHostValue(value) {
			addFinding(findings, path, "public.fixture_host", "fixture host metadata must use a reserved synthetic value")
			break
		}
	}
}

func isReservedDomain(domain string) bool {
	return domain == "example.com" || domain == "example.net" || domain == "example.org" || domain == "localhost" || strings.HasSuffix(domain, ".test") || strings.HasSuffix(domain, ".invalid") || strings.HasSuffix(domain, ".example")
}

func syntheticHostValue(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.TrimPrefix(strings.TrimPrefix(value, "https://"), "http://")
	host := strings.SplitN(value, "/", 2)[0]
	host = strings.SplitN(host, ":", 2)[0]
	return host == "localhost" || host == "127.0.0.1" || host == "::1" || host == "example.com" || strings.HasSuffix(host, ".test") || strings.HasSuffix(host, ".invalid") || strings.HasSuffix(host, ".example")
}

func checkEvidenceContent(path, content string, findings *[]Finding) {
	for _, match := range hostFieldPattern.FindAllStringSubmatch(content, -1) {
		if !syntheticHostValue(match[1]) {
			addFinding(findings, path, "public.host_metadata", "evidence contains identifying host or backend metadata")
			break
		}
	}
}

func isMarkupPath(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".html", ".htm", ".svg":
		return true
	default:
		return false
	}
}

func hasExecutableFence(content string) bool {
	fence := regexp.MustCompile("(?im)^```(?:ba)?sh|^```zsh|^```powershell|^```cmd(?:$|\\s)")
	return fence.MatchString(content)
}

func checkWorkflow(path, content string, findings *[]Finding) {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	digest := fmt.Sprintf("%x", sha256.Sum256([]byte(normalized)))
	if filepath.ToSlash(path) != canonicalFoundationWorkflowPath || digest != canonicalFoundationWorkflowSHA256 {
		addFinding(findings, path, "public.workflow_contract", "workflow must match the immutable H-001 foundation contract")
	}
	records, syntaxMessages := parseCanonicalWorkflow(content)
	for _, message := range syntaxMessages {
		addFinding(findings, path, "public.workflow_yaml", "%s", message)
	}
	for _, message := range checkWorkflowEvents(records) {
		addFinding(findings, path, "public.workflow_event", "%s", message)
	}
	for _, message := range checkWorkflowPermissions(content) {
		addFinding(findings, path, "public.workflow_permissions", "%s", message)
	}
	for _, message := range checkWorkflowSecrets(records, content) {
		addFinding(findings, path, "public.workflow_secret", "%s", message)
	}
	for _, message := range checkWorkflowActions(records) {
		addFinding(findings, path, "public.workflow_action", "%s", message)
	}
	for _, message := range checkWorkflowContainers(records, content) {
		addFinding(findings, path, "public.workflow_container", "%s", message)
	}
	for _, reference := range containerReference.FindAllString(content, -1) {
		if reference != allowedWorkflowContainer || !containerDigest.MatchString(reference) {
			addFinding(findings, path, "public.unpinned_container", "workflow container is not on the immutable H-001 allowlist")
		}
	}
}

type workflowYAMLRecord struct {
	Line      int
	Indent    int
	Key       string
	Value     string
	List      bool
	Ancestors []string
}

type workflowYAMLStackEntry struct {
	Indent int
	Key    string
}

// parseCanonicalWorkflow recognizes only the deliberately small YAML surface
// used by the H-001 foundation workflow. Unsupported YAML is rejected rather
// than interpreted: authority-bearing syntax must have one obvious spelling.
func parseCanonicalWorkflow(content string) ([]workflowYAMLRecord, []string) {
	var records []workflowYAMLRecord
	var messages []string
	if strings.HasPrefix(content, "\ufeff") {
		messages = append(messages, "UTF-8 BOM is prohibited")
	}

	lines := strings.Split(content, "\n")
	stack := []workflowYAMLStackEntry{}
	blockScalarIndent := -1
	for index, line := range lines {
		raw := strings.TrimSuffix(line, "\r")
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		indent, structural, validIndent := workflowYAMLLine(raw)
		if !validIndent {
			messages = append(messages, fmt.Sprintf("line %d uses a tab in YAML indentation", index+1))
			continue
		}
		if blockScalarIndent >= 0 {
			if indent > blockScalarIndent {
				continue
			}
			blockScalarIndent = -1
		}
		if structural == "" {
			continue
		}
		if structural == "---" || structural == "..." || strings.HasPrefix(structural, "%") {
			messages = append(messages, fmt.Sprintf("line %d uses a YAML directive or document marker", index+1))
			continue
		}
		if workflowYAMLExplicitKey(structural) {
			messages = append(messages, fmt.Sprintf("line %d uses an explicit YAML mapping key", index+1))
			continue
		}
		if workflowYAMLHasAnchorOrAlias(structural) {
			messages = append(messages, fmt.Sprintf("line %d uses a YAML anchor or alias", index+1))
			continue
		}
		if workflowYAMLHasTag(structural) {
			messages = append(messages, fmt.Sprintf("line %d uses a YAML tag", index+1))
			continue
		}

		mappingLine := structural
		list := false
		if strings.HasPrefix(mappingLine, "-") {
			if len(mappingLine) == 1 || (mappingLine[1] != ' ' && mappingLine[1] != '\t') {
				messages = append(messages, fmt.Sprintf("line %d uses unsupported sequence syntax", index+1))
				continue
			}
			list = true
			mappingLine = strings.TrimSpace(mappingLine[1:])
		}

		colon := workflowYAMLColon(mappingLine)
		if colon < 0 {
			messages = append(messages, fmt.Sprintf("line %d is not a canonical mapping", index+1))
			continue
		}
		rawKey := strings.TrimSpace(mappingLine[:colon])
		if len(rawKey) >= 1 && (rawKey[0] == '\'' || rawKey[0] == '"') {
			messages = append(messages, fmt.Sprintf("line %d uses a quoted or escaped mapping key", index+1))
			continue
		}
		key, value, mapping := workflowYAMLMapping(mappingLine)
		if !mapping || !workflowCanonicalKey(key) {
			messages = append(messages, fmt.Sprintf("line %d uses a non-canonical mapping key", index+1))
			continue
		}
		if workflowYAMLHasFlowMap(mappingLine) {
			messages = append(messages, fmt.Sprintf("line %d uses a flow-style mapping", index+1))
			continue
		}
		if workflowYAMLHasFlowSequence(mappingLine) && !(key == "branches" && value == "[main]") {
			messages = append(messages, fmt.Sprintf("line %d uses an unsupported flow-style sequence", index+1))
			continue
		}

		for len(stack) > 0 && stack[len(stack)-1].Indent >= indent {
			stack = stack[:len(stack)-1]
		}
		ancestors := make([]string, len(stack))
		for stackIndex, entry := range stack {
			ancestors[stackIndex] = entry.Key
		}
		records = append(records, workflowYAMLRecord{
			Line:      index + 1,
			Indent:    indent,
			Key:       key,
			Value:     value,
			List:      list,
			Ancestors: ancestors,
		})
		stack = append(stack, workflowYAMLStackEntry{Indent: indent, Key: key})
		if workflowYAMLBlockScalar(value) {
			blockScalarIndent = indent
		}
	}
	return records, messages
}

func workflowCanonicalKey(key string) bool {
	if key == "" {
		return false
	}
	for _, character := range key {
		if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9') || character == '_' || character == '-' || character == '.' {
			continue
		}
		return false
	}
	return true
}

func workflowYAMLHasTag(line string) bool {
	return workflowYAMLHasIndicator(line, '!')
}

func workflowYAMLHasIndicator(line string, indicator byte) bool {
	var quote byte
	for index := 0; index < len(line); index++ {
		character := line[index]
		if quote != 0 {
			if character == quote && (index == 0 || line[index-1] != '\\') {
				quote = 0
			}
			continue
		}
		if character == '\'' || character == '"' {
			quote = character
			continue
		}
		if character != indicator {
			continue
		}
		if index > 0 && !workflowYAMLIndicatorBoundary(line[index-1]) {
			continue
		}
		if index+1 < len(line) && !workflowYAMLIndicatorTerminator(line[index+1]) {
			return true
		}
	}
	return false
}

func workflowYAMLHasFlowMap(line string) bool {
	return workflowYAMLHasFlowCharacters(line, '{', '}')
}

func workflowYAMLHasFlowSequence(line string) bool {
	return workflowYAMLHasFlowCharacters(line, '[', ']')
}

func workflowYAMLHasFlowCharacters(line string, open, close byte) bool {
	var quote byte
	for index := 0; index < len(line); index++ {
		character := line[index]
		if quote != 0 {
			if character == quote && (index == 0 || line[index-1] != '\\') {
				quote = 0
			}
			continue
		}
		if character == '\'' || character == '"' {
			quote = character
			continue
		}
		if strings.HasPrefix(line[index:], "${{") {
			end := strings.Index(line[index+3:], "}}")
			if end >= 0 {
				index += end + 4
				continue
			}
		}
		if character == open || character == close {
			return true
		}
	}
	return false
}

func checkWorkflowEvents(records []workflowYAMLRecord) []string {
	var messages []string
	var onIndexes []int
	for index, record := range records {
		if record.Indent == 0 && record.Key == "on" {
			onIndexes = append(onIndexes, index)
		}
	}
	if len(onIndexes) != 1 {
		return []string{"workflow must declare exactly one canonical top-level on block"}
	}
	onIndex := onIndexes[0]
	if records[onIndex].Value != "" || records[onIndex].List {
		messages = append(messages, "top-level on must be a block mapping")
	}

	allowed := map[string]bool{"push": true, "pull_request": true, "workflow_dispatch": true}
	counts := map[string]int{}
	currentEvent := ""
	pushBranches := 0
	for index := onIndex + 1; index < len(records); index++ {
		record := records[index]
		if record.Indent == 0 {
			break
		}
		if record.Indent == 2 {
			currentEvent = record.Key
			if !allowed[record.Key] || record.Value != "" || record.List {
				messages = append(messages, fmt.Sprintf("line %d declares a non-canonical workflow event", record.Line))
				continue
			}
			counts[record.Key]++
			continue
		}
		if currentEvent == "push" && record.Indent == 4 && record.Key == "branches" && record.Value == "[main]" && !record.List {
			pushBranches++
			continue
		}
		messages = append(messages, fmt.Sprintf("line %d adds unsupported configuration to workflow event %s", record.Line, currentEvent))
	}
	for _, event := range []string{"push", "pull_request", "workflow_dispatch"} {
		if counts[event] != 1 {
			messages = append(messages, fmt.Sprintf("event %s must appear exactly once", event))
		}
	}
	if pushBranches != 1 {
		messages = append(messages, "push must contain exactly branches: [main]")
	}
	return messages
}

func checkWorkflowSecrets(records []workflowYAMLRecord, content string) []string {
	var messages []string
	for _, record := range records {
		switch record.Key {
		case "secrets", "credentials", "token", "password", "environment", "env":
			messages = append(messages, fmt.Sprintf("line %d declares prohibited credential-bearing key %s", record.Line, record.Key))
		}
	}
	activeContent := workflowActiveContent(content)
	expressionCounts := make(map[string]int)
	for _, expression := range workflowExpressions(activeContent) {
		expression = strings.TrimSpace(expression)
		if secretWord.MatchString(expression) || githubTokenWord.MatchString(expression) {
			messages = append(messages, "GitHub expressions must not reference secrets or github.token")
		}
		if _, allowed := allowedWorkflowExpressions[expression]; !allowed {
			messages = append(messages, "GitHub expression is not on the immutable H-001 allowlist")
			continue
		}
		expressionCounts[expression]++
	}
	for expression, required := range allowedWorkflowExpressions {
		if expressionCounts[expression] != required {
			messages = append(messages, fmt.Sprintf("GitHub expression %s must appear exactly %d time", expression, required))
		}
	}
	if regexp.MustCompile(`(?i)\bGITHUB_TOKEN\b`).MatchString(activeContent) {
		messages = append(messages, "GITHUB_TOKEN references are prohibited")
	}
	return messages
}

func workflowActiveContent(content string) string {
	lines := strings.Split(content, "\n")
	var active strings.Builder
	blockScalarIndent := -1
	for _, line := range lines {
		raw := strings.TrimSuffix(line, "\r")
		indent, structural, validIndent := workflowYAMLLine(raw)
		if !validIndent {
			continue
		}
		if blockScalarIndent >= 0 {
			if strings.TrimSpace(raw) == "" || indent > blockScalarIndent {
				active.WriteString(raw)
				active.WriteByte('\n')
				continue
			}
			blockScalarIndent = -1
		}
		if structural == "" || strings.HasPrefix(strings.TrimSpace(raw), "#") {
			continue
		}
		active.WriteString(structural)
		active.WriteByte('\n')
		_, value, mapping := workflowYAMLMapping(strings.TrimSpace(strings.TrimPrefix(structural, "-")))
		if mapping && workflowYAMLBlockScalar(value) {
			blockScalarIndent = indent
		}
	}
	return active.String()
}

func workflowExpressions(content string) []string {
	var expressions []string
	for offset := 0; offset < len(content); {
		start := strings.Index(content[offset:], "${{")
		if start < 0 {
			break
		}
		start += offset + 3
		end := strings.Index(content[start:], "}}")
		if end < 0 {
			expressions = append(expressions, content[start:])
			break
		}
		expressions = append(expressions, content[start:start+end])
		offset = start + end + 2
	}
	return expressions
}

func checkWorkflowActions(records []workflowYAMLRecord) []string {
	var messages []string
	counts := map[string]int{}
	for index, record := range records {
		if record.Key != "uses" {
			continue
		}
		reference := record.Value
		if !recordHasAncestor(record, "steps") {
			messages = append(messages, fmt.Sprintf("line %d declares a reusable, local, or job-level action", record.Line))
			continue
		}
		if !allowedWorkflowActions[reference] {
			messages = append(messages, fmt.Sprintf("line %d action is not on the immutable H-001 allowlist", record.Line))
			continue
		}
		counts[reference]++
		expected := map[string]string{}
		if strings.HasPrefix(reference, "actions/checkout@") {
			expected = map[string]string{"fetch-depth": "0", "persist-credentials": "false"}
		} else if strings.HasPrefix(reference, "actions/setup-go@") {
			expected = map[string]string{"go-version": "1.24.11", "cache": "false"}
		}
		for _, message := range checkWorkflowActionInputs(records, index, expected) {
			messages = append(messages, message)
		}
	}
	for reference := range allowedWorkflowActions {
		if counts[reference] != 1 {
			messages = append(messages, fmt.Sprintf("action %s must appear exactly once", reference))
		}
	}
	return messages
}

func checkWorkflowActionInputs(records []workflowYAMLRecord, actionIndex int, expected map[string]string) []string {
	record := records[actionIndex]
	stepIndent := -1
	stepIndex := -1
	for index := actionIndex - 1; index >= 0; index-- {
		if records[index].List && recordHasAncestor(records[index], "steps") {
			stepIndent = records[index].Indent
			stepIndex = index
			break
		}
	}
	if stepIndent < 0 || record.Indent != stepIndent+2 {
		return []string{fmt.Sprintf("line %d action is not attached to a canonical step", record.Line)}
	}
	stepEnd := len(records)
	for index := stepIndex + 1; index < len(records); index++ {
		if records[index].List && records[index].Indent == stepIndent && recordHasAncestor(records[index], "steps") {
			stepEnd = index
			break
		}
	}
	withIndex := -1
	withCount := 0
	for index := actionIndex + 1; index < stepEnd; index++ {
		candidate := records[index]
		if candidate.Indent == record.Indent && candidate.Key == "with" {
			withIndex = index
			withCount++
			if candidate.Value != "" || candidate.List {
				return []string{fmt.Sprintf("line %d action inputs must use one canonical with block", candidate.Line)}
			}
		}
	}
	if withCount != 1 {
		return []string{fmt.Sprintf("line %d action requires exactly one with block", record.Line)}
	}
	found := map[string]int{}
	for index := withIndex + 1; index < stepEnd; index++ {
		candidate := records[index]
		if candidate.Indent <= records[withIndex].Indent {
			break
		}
		expectedValue, ok := expected[candidate.Key]
		if !ok || candidate.Indent != records[withIndex].Indent+2 || candidate.List || !recordHasAncestor(candidate, "with") {
			return []string{fmt.Sprintf("line %d action input %s is not on the canonical allowlist", candidate.Line, candidate.Key)}
		}
		if candidate.Value != expectedValue {
			return []string{fmt.Sprintf("line %d action input %s has a non-canonical value", candidate.Line, candidate.Key)}
		}
		found[candidate.Key]++
	}
	var messages []string
	for key := range expected {
		if found[key] != 1 {
			messages = append(messages, fmt.Sprintf("line %d action requires exactly one %s input", record.Line, key))
		}
	}
	return messages
}

func recordHasAncestor(record workflowYAMLRecord, key string) bool {
	for _, ancestor := range record.Ancestors {
		if ancestor == key {
			return true
		}
	}
	return false
}

func checkWorkflowContainers(records []workflowYAMLRecord, content string) []string {
	var messages []string
	for _, record := range records {
		if record.Key == "container" || record.Key == "services" || record.Key == "image" {
			messages = append(messages, fmt.Sprintf("line %d declares a prohibited job container or service", record.Line))
		}
	}
	commands := dockerCommand.FindAllStringSubmatch(workflowActiveContent(content), -1)
	for _, command := range commands {
		if command[1] != "run" {
			messages = append(messages, fmt.Sprintf("docker subcommand %s is prohibited", command[1]))
		}
	}
	commandSignatures, commandMessages := workflowDockerRunCommands(content)
	messages = append(messages, commandMessages...)
	if len(commands) != 3 || len(commandSignatures) != 3 {
		messages = append(messages, "foundation workflow must contain exactly three canonical docker run commands")
	}
	expected := expectedWorkflowDockerCommands()
	actual := make(map[string]int)
	for _, signature := range commandSignatures {
		actual[signature]++
		if _, ok := expected[signature]; !ok {
			messages = append(messages, "docker run command is not on the exact H-001 allowlist")
		}
	}
	for signature, count := range expected {
		if actual[signature] != count {
			messages = append(messages, "required canonical docker run command is missing or duplicated")
		}
	}
	return messages
}

func workflowDockerRunCommands(content string) ([]string, []string) {
	lines := strings.Split(content, "\n")
	var signatures []string
	var messages []string
	for index := 0; index < len(lines); index++ {
		if !regexp.MustCompile(`\bdocker\s+run\b`).MatchString(lines[index]) {
			continue
		}
		logical := strings.TrimSpace(lines[index])
		for strings.HasSuffix(logical, "\\") && index+1 < len(lines) {
			logical = strings.TrimSpace(strings.TrimSuffix(logical, "\\")) + " " + strings.TrimSpace(lines[index+1])
			index++
		}
		fields, err := workflowShellFields(logical)
		if err != nil {
			messages = append(messages, "docker run command is not in canonical shell form")
			continue
		}
		signatures = append(signatures, strings.Join(fields, "\x00"))
	}
	return signatures, messages
}

func expectedWorkflowDockerCommands() map[string]int {
	common := []string{"docker", "run", "--rm", "--network", "none", "--read-only", "--cap-drop", "ALL", "--security-opt", "no-new-privileges"}
	canary := append(append([]string{"if"}, common...), "-v", "${RUNNER_TEMP}/gitleaks-canary:/scan:ro", allowedWorkflowContainer, "detect", "--no-git", "--source", "/scan", "--redact", "--no-banner;", "then")
	worktree := append(append([]string{}, common...), "-v", "${GITHUB_WORKSPACE}:/repo:ro", "-w", "/repo", allowedWorkflowContainer, "detect", "--no-git", "--source", ".", "--redact", "--no-banner")
	history := append(append([]string{}, common...), "-v", "${GITHUB_WORKSPACE}:/repo:ro", "-w", "/repo", allowedWorkflowContainer, "detect", "--source", ".", "--redact", "--no-banner")
	return map[string]int{
		strings.Join(canary, "\x00"):   1,
		strings.Join(worktree, "\x00"): 1,
		strings.Join(history, "\x00"):  1,
	}
}

func workflowShellFields(command string) ([]string, error) {
	var fields []string
	var field strings.Builder
	var quote byte
	escaped := false
	flush := func() {
		if field.Len() > 0 {
			fields = append(fields, field.String())
			field.Reset()
		}
	}
	for index := 0; index < len(command); index++ {
		character := command[index]
		if escaped {
			field.WriteByte(character)
			escaped = false
			continue
		}
		if character == '\\' && quote != '\'' {
			escaped = true
			continue
		}
		if quote != 0 {
			if character == quote {
				quote = 0
			} else {
				field.WriteByte(character)
			}
			continue
		}
		if character == '\'' || character == '"' {
			quote = character
			continue
		}
		if character == ' ' || character == '\t' || character == '\r' || character == '\n' {
			flush()
			continue
		}
		field.WriteByte(character)
	}
	if quote != 0 || escaped {
		return nil, fmt.Errorf("unterminated shell token")
	}
	flush()
	return fields, nil
}

// checkWorkflowPermissions accepts one deliberately small GitHub Actions
// permission shape:
//
// permissions:
//
//	contents: read
//
// Permission flow mappings, aliases, extra scopes, duplicate declarations,
// and job-level overrides fail closed. This is intentionally narrower than a
// general YAML parser: H-001 needs a mechanically obvious read-only token and
// must not accept a YAML spelling whose authority is hard to audit.
func checkWorkflowPermissions(content string) []string {
	lines := strings.Split(content, "\n")
	var messages []string
	topLevelDeclarations := 0
	blockScalarIndent := -1

	for index := 0; index < len(lines); index++ {
		raw := strings.TrimSuffix(lines[index], "\r")
		if strings.TrimSpace(raw) == "" || strings.HasPrefix(strings.TrimSpace(raw), "#") {
			continue
		}

		indent, structural, validIndent := workflowYAMLLine(raw)
		if !validIndent {
			messages = append(messages, fmt.Sprintf("line %d uses a tab in YAML indentation", index+1))
			continue
		}
		if blockScalarIndent >= 0 {
			if indent > blockScalarIndent {
				continue
			}
			blockScalarIndent = -1
		}

		key, value, mapping := workflowYAMLMapping(structural)
		if mapping && workflowYAMLBlockScalar(value) {
			blockScalarIndent = indent
		}
		if workflowYAMLExplicitKey(structural) {
			messages = append(messages, fmt.Sprintf("line %d uses a YAML explicit mapping key, which foundation workflows prohibit", index+1))
			continue
		}
		if workflowYAMLHasAnchorOrAlias(structural) {
			messages = append(messages, fmt.Sprintf("line %d uses a YAML anchor or alias, which foundation workflows prohibit", index+1))
			continue
		}
		permissionKeys := workflowYAMLPermissionKeyCount(structural)
		if permissionKeys == 0 {
			continue
		}
		if !mapping || key != "permissions" {
			messages = append(messages, fmt.Sprintf("line %d declares permissions inside an inline mapping", index+1))
			continue
		}
		if indent != 0 {
			messages = append(messages, fmt.Sprintf("line %d declares job-level or nested permissions", index+1))
			continue
		}

		topLevelDeclarations++
		if value != "" {
			messages = append(messages, fmt.Sprintf("line %d must use a block mapping containing only contents: read", index+1))
			continue
		}

		contentsRead := 0
		for childIndex := index + 1; childIndex < len(lines); childIndex++ {
			childRaw := strings.TrimSuffix(lines[childIndex], "\r")
			if strings.TrimSpace(childRaw) == "" || strings.HasPrefix(strings.TrimSpace(childRaw), "#") {
				continue
			}
			childIndent, childStructural, childIndentValid := workflowYAMLLine(childRaw)
			if !childIndentValid {
				messages = append(messages, fmt.Sprintf("line %d uses a tab in YAML indentation", childIndex+1))
				continue
			}
			if childIndent == 0 {
				break
			}
			childKey, childValue, childMapping := workflowYAMLMapping(childStructural)
			if childMapping && childKey == "contents" && childValue == "read" {
				contentsRead++
				continue
			}
			messages = append(messages, fmt.Sprintf("line %d adds a permission other than contents: read", childIndex+1))
		}
		if contentsRead != 1 {
			messages = append(messages, fmt.Sprintf("line %d must contain exactly one contents: read entry", index+1))
		}
	}

	if topLevelDeclarations == 0 {
		messages = append(messages, "workflow must declare top-level contents: read permissions")
	} else if topLevelDeclarations != 1 {
		messages = append(messages, "workflow must declare top-level permissions exactly once")
	}
	return messages
}

func workflowYAMLHasAnchorOrAlias(line string) bool {
	var quote byte
	for index := 0; index < len(line); index++ {
		character := line[index]
		if quote != 0 {
			if character == quote && (index == 0 || line[index-1] != '\\') {
				quote = 0
			}
			continue
		}
		if character == '\'' || character == '"' {
			quote = character
			continue
		}
		if character != '&' && character != '*' {
			continue
		}
		if index > 0 && !workflowYAMLIndicatorBoundary(line[index-1]) {
			continue
		}
		if index+1 >= len(line) || workflowYAMLIndicatorTerminator(line[index+1]) {
			continue
		}
		return true
	}
	return false
}

func workflowYAMLIndicatorBoundary(character byte) bool {
	return character == ' ' || character == '\t' || strings.ContainsRune("[{,:?-", rune(character))
}

func workflowYAMLIndicatorTerminator(character byte) bool {
	return character == ' ' || character == '\t' || strings.ContainsRune("[]{}:,", rune(character))
}

func workflowYAMLExplicitKey(line string) bool {
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "? ") || line == "?" {
		return true
	}
	if !strings.HasPrefix(line, "-") {
		return false
	}
	line = strings.TrimSpace(strings.TrimPrefix(line, "-"))
	return strings.HasPrefix(line, "? ") || line == "?"
}

func workflowYAMLLine(raw string) (int, string, bool) {
	indent := 0
	for indent < len(raw) {
		switch raw[indent] {
		case ' ':
			indent++
		case '\t':
			return indent, "", false
		default:
			return indent, strings.TrimSpace(stripWorkflowYAMLComment(raw[indent:])), true
		}
	}
	return indent, "", true
}

func workflowYAMLMapping(line string) (string, string, bool) {
	colon := workflowYAMLColon(line)
	if colon < 0 {
		return "", "", false
	}
	key := strings.TrimSpace(line[:colon])
	if len(key) >= 2 && ((key[0] == '\'' && key[len(key)-1] == '\'') || (key[0] == '"' && key[len(key)-1] == '"')) {
		key = key[1 : len(key)-1]
	}
	return key, strings.TrimSpace(line[colon+1:]), key != ""
}

func workflowYAMLColon(line string) int {
	var quote byte
	for index := 0; index < len(line); index++ {
		character := line[index]
		if quote != 0 {
			if character == quote && (index == 0 || line[index-1] != '\\') {
				quote = 0
			}
			continue
		}
		if character == '\'' || character == '"' {
			quote = character
			continue
		}
		if character == ':' {
			return index
		}
	}
	return -1
}

func workflowYAMLPermissionKeyCount(line string) int {
	count := 0
	var quote byte
	for index := 0; index < len(line); index++ {
		character := line[index]
		if quote != 0 {
			if character == quote && (index == 0 || line[index-1] != '\\') {
				quote = 0
			}
			continue
		}
		if character == '\'' || character == '"' {
			quote = character
			continue
		}
		if character == ':' && workflowYAMLKeyBeforeColon(line, index) == "permissions" {
			count++
		}
	}
	return count
}

func workflowYAMLKeyBeforeColon(line string, colon int) string {
	end := colon - 1
	for end >= 0 && (line[end] == ' ' || line[end] == '\t') {
		end--
	}
	if end < 0 {
		return ""
	}
	if line[end] == '\'' || line[end] == '"' {
		quote := line[end]
		for start := end - 1; start >= 0; start-- {
			if line[start] == quote && (start == 0 || line[start-1] != '\\') {
				key := line[start : end+1]
				if quote == '"' {
					decoded, err := strconv.Unquote(key)
					if err == nil {
						return decoded
					}
				}
				return line[start+1 : end]
			}
		}
		return ""
	}
	start := end
	for start >= 0 {
		character := line[start]
		if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9') || character == '_' || character == '-' {
			start--
			continue
		}
		break
	}
	return line[start+1 : end+1]
}

func stripWorkflowYAMLComment(line string) string {
	var quote byte
	for index := 0; index < len(line); index++ {
		character := line[index]
		if quote != 0 {
			if character == quote && (index == 0 || line[index-1] != '\\') {
				quote = 0
			}
			continue
		}
		if character == '\'' || character == '"' {
			quote = character
			continue
		}
		if character == '#' && (index == 0 || line[index-1] == ' ' || line[index-1] == '\t') {
			return strings.TrimSpace(line[:index])
		}
	}
	return strings.TrimSpace(line)
}

func workflowYAMLBlockScalar(value string) bool {
	if value == "" {
		return false
	}
	first := value[0]
	if first != '|' && first != '>' {
		return false
	}
	for _, character := range value[1:] {
		if character != '+' && character != '-' && (character < '1' || character > '9') {
			return false
		}
	}
	return true
}

func allowedBinaryAsset(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".gif", ".ico", ".jpeg", ".jpg", ".pdf", ".png", ".webp", ".woff", ".woff2":
		return true
	default:
		return false
	}
}

func isGeneratedPath(path string) bool {
	lower := strings.ToLower(path)
	return strings.Contains("/"+lower+"/", "/generated/") || strings.Contains(filepath.Base(lower), ".generated.")
}

type generatedDeclarationManifest struct {
	SchemaVersion int `json:"schemaVersion"`
	Files         []struct {
		Path    string `json:"path"`
		Source  string `json:"source"`
		Command string `json:"command"`
	} `json:"files"`
}

func loadGeneratedDeclarations(root string, findings *[]Finding) map[string]bool {
	const path = ".harness/generated/generated-files.json"
	result := make(map[string]bool)
	data, err := readRepoFile(root, path)
	if err != nil {
		addFinding(findings, path, "public.generated_manifest", "generated-file declaration manifest is required")
		return result
	}
	var manifest generatedDeclarationManifest
	if err := decodeStrictJSON(data, &manifest); err != nil {
		addFinding(findings, path, "public.generated_manifest", "%v", err)
		return result
	}
	if manifest.SchemaVersion != 1 {
		addFinding(findings, path, "public.generated_manifest", "schemaVersion must be 1")
	}
	for _, file := range manifest.Files {
		if !safeRelativePath(file.Path) || !isGeneratedPath(file.Path) || file.Path == path {
			addFinding(findings, path, "public.generated_path", "generated declaration contains an invalid path")
			continue
		}
		if strings.TrimSpace(file.Source) == "" || strings.TrimSpace(file.Command) == "" {
			addFinding(findings, path, "public.generated_reproduction", "%s needs both source and command", file.Path)
			continue
		}
		if result[file.Path] {
			addFinding(findings, path, "public.generated_duplicate", "%s is declared more than once", file.Path)
		}
		result[file.Path] = true
	}
	return result
}

func redactCount(label string, count int) string {
	return fmt.Sprintf("%s (%d redacted matches)", label, count)
}

func scanLines(content string) []string {
	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}
