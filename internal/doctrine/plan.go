/*
FactoryDocSync:
docs:
- docs/features/F-001-doctrine-foundation.md
- docs/features/F-002-work-authority.md
- docs/design-docs/mars-provenance.md
- docs/code-documentation-map.md
*/

package doctrine

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	canonicalActivePlan = "docs/exec-plans/active/current-operating-plan.md"
	ticketIDPattern     = regexp.MustCompile(`\b[A-Z]+-\d{3}\b`)
	activeStatusLine    = regexp.MustCompile(`(?im)^\s*(?:[-*]\s*)?(?:\*\*)?status(?:\s*:\s*\*\*|\*\*\s*:|\s*:)\s*active\s*$`)
	planOwnerLine       = regexp.MustCompile(`(?im)^\s*(?:[-*]\s*)?(?:\*\*)?owner(?:\s*:\s*\*\*|\*\*\s*:|\s*:)\s*delivery orchestrator\s*$`)
	markdownLink        = regexp.MustCompile(`\[[^\]]+\]\(([^)]+)\)`)
	markdownTableRule   = regexp.MustCompile(`^\s*:?-{3,}:?\s*$`)
	dependencyIDPattern = regexp.MustCompile(`^[A-Z]+-\d{3}$`)
	planPhaseLine       = regexp.MustCompile(`(?im)^\s*(?:[-*]\s*)?\*\*phase:\*\*\s*([a-z][a-z0-9-]*)\s*$`)
	planGoalLine        = regexp.MustCompile(`(?im)^\s*(?:[-*]\s*)?\*\*goal:\*\*\s*(G-\d{3})\s*$`)
	currentFeatureLine  = regexp.MustCompile(`(?im)^\s*(?:[-*]\s*)?\*\*current feature:\*\*\s*(F-\d{3})\s*$`)
	currentBeadLine     = regexp.MustCompile(`(?im)^\s*(?:[-*]\s*)?\*\*current bead:\*\*\s*(M3-([A-Z]+)(\d{3}))\s*\(display ID ((?:UI|[A-Z])-\d{3})\)\s*$`)
	authorityLine       = regexp.MustCompile(`(?im)^\s*(?:[-*]\s*)?\*\*authority:\*\*\s*\S.*$`)
	plainYAMLKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)
	goalIDPattern       = regexp.MustCompile(`^G-\d{3}$`)
	decisionListPattern = regexp.MustCompile(`^PD-\d{3}(?:\s*,\s*PD-\d{3})*$`)
	decisionIDPattern   = regexp.MustCompile(`PD-\d{3}`)
	anyScenarioPattern  = regexp.MustCompile(`\bF-\d{3}-S\d+\b`)
	featureBeadPattern  = regexp.MustCompile(`(?i)^(M3-([A-Z]+)(\d{3}))\s*\(display ID ((?:UI|[A-Z])-\d{3})\)$`)
	workBeadPattern     = regexp.MustCompile(`(?i)^external Bead (M3-([A-Z]+)(\d{3}))\s*\(display ID ((?:UI|[A-Z])-\d{3})\)$`)
)

const (
	planPhaseContractPublication = "contract-publication"
	planPhaseDelivery            = "delivery"
)

type activePlanMetadata struct {
	phase          string
	goal           string
	currentFeature string
	currentBead    string
	currentDisplay string
}

type deliveryTableLayout struct {
	headers          []string
	ticketColumn     int
	dependencyColumn int
	evidenceColumn   int
	ownerColumn      int
	stateColumn      int
}

type featureContractMetadata struct {
	goal             string
	productDecisions []string
	productSpec      string
	currentBead      string
	currentDisplay   string
	scenarios        []featureScenario
}

type featureScenario struct {
	id    string
	state string
}

// CheckPlan validates the Git-owned execution plan and its one-way link to
// the current Beads work item. It never reads or mutates the external ledger.
func CheckPlan(repo string) ([]Finding, error) {
	root, err := repositoryRoot(repo)
	if err != nil {
		return nil, err
	}
	var findings []Finding
	activeDir := filepath.Join(root, "docs", "exec-plans", "active")
	entries, err := os.ReadDir(activeDir)
	if err != nil {
		addFinding(&findings, "docs/exec-plans/active", "plan.active_directory", "active-plan directory is required")
		return findings, nil
	}
	var active []string
	for _, entry := range entries {
		if entry.IsDir() || strings.EqualFold(entry.Name(), "README.md") || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		active = append(active, cleanPublicPath(filepath.Join("docs/exec-plans/active", entry.Name())))
	}
	sort.Strings(active)
	if len(active) != 1 {
		addFinding(&findings, "docs/exec-plans/active", "plan.active_cardinality", "exactly one active Markdown plan is required; found %d", len(active))
		return findings, nil
	}
	if active[0] != canonicalActivePlan {
		addFinding(&findings, active[0], "plan.canonical_path", "the sole active plan must be %s", canonicalActivePlan)
	}
	checkActivePlan(root, active[0], &findings)
	checkNoGitTicketAuthority(root, &findings)
	sortFindings(findings)
	return findings, nil
}

func checkActivePlan(root, path string, findings *[]Finding) {
	data, err := readRepoFile(root, path)
	if err != nil {
		addFinding(findings, path, "plan.unreadable", "active plan cannot be read")
		return
	}
	content := string(data)
	linkedArtifacts := make(map[string]bool)
	metadata := parseActivePlanMetadata(content, path, findings)
	if !activeStatusLine.MatchString(content) {
		addFinding(findings, path, "plan.status", "active plan must declare Status: Active")
	}
	if !planOwnerLine.MatchString(content) {
		addFinding(findings, path, "plan.owner", "active plan must declare Delivery Orchestrator as owner")
	}
	authorityFields := metadataLabelCount(content, "Authority")
	if authorityFields != 1 {
		addFinding(findings, path, "plan.metadata_cardinality", "exactly one Authority field is required; found %d", authorityFields)
	} else if !authorityLine.MatchString(content) {
		addFinding(findings, path, "plan.required_metadata", "active plan Authority field must be non-empty")
	}
	for _, requiredSection := range []string{
		"## Current hypothesis and walking skeleton",
		"## Scenario priority",
		"## Delivery waves",
		"## Success evidence",
		"## Falsification evidence",
		"## Failure ownership and convergence",
	} {
		if !strings.Contains(content, requiredSection) {
			addFinding(findings, path, "plan.required_section", "active plan must contain %s", requiredSection)
		}
	}
	if metadata.goal != "" && !strings.Contains(content, metadata.goal) {
		addFinding(findings, path, "plan.required_link", "active plan must reference current goal %s", metadata.goal)
	}
	if metadata.currentFeature != "" {
		scenarioPattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(metadata.currentFeature) + `-S\d+\b`)
		if !scenarioPattern.MatchString(content) {
			addFinding(findings, path, "plan.required_link", "active plan must prioritize at least one scenario for %s", metadata.currentFeature)
		}
	}
	if !strings.Contains(strings.ToLower(content), "beads") || !strings.Contains(strings.ToLower(content), "git") {
		addFinding(findings, path, "plan.authority_split", "active plan must state the Beads work authority and Git doctrine authority")
	}
	for _, match := range markdownLink.FindAllStringSubmatch(content, -1) {
		target := strings.SplitN(match[1], "#", 2)[0]
		if target == "" || strings.Contains(target, "://") || strings.HasPrefix(target, "#") {
			continue
		}
		resolved := filepath.Clean(filepath.Join(filepath.Dir(filepath.Join(root, filepath.FromSlash(path))), filepath.FromSlash(target)))
		relative, relErr := filepath.Rel(root, resolved)
		if relErr != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			addFinding(findings, path, "plan.unsafe_link", "linked artifact %q resolves outside the repository", target)
			continue
		}
		linkedArtifacts[cleanPublicPath(relative)] = true
		if info, err := os.Stat(resolved); err != nil || !info.Mode().IsRegular() {
			addFinding(findings, path, "plan.broken_link", "linked repository artifact %q does not exist", target)
		}
	}
	checkPlanLineage(root, path, content, metadata, linkedArtifacts, findings)
	currentState := checkDeliveryTable(content, path, metadata, findings)
	checkPlanManifestProjection(root, path, metadata, currentState, findings)
}

func parseActivePlanMetadata(content, path string, findings *[]Finding) activePlanMetadata {
	metadata := activePlanMetadata{
		phase:          singleMetadataValue(content, path, "Phase", planPhaseLine, findings),
		goal:           strings.ToUpper(singleMetadataValue(content, path, "Goal", planGoalLine, findings)),
		currentFeature: strings.ToUpper(singleMetadataValue(content, path, "Current feature", currentFeatureLine, findings)),
	}
	beadMatches := currentBeadLine.FindAllStringSubmatch(content, -1)
	beadFields := metadataLabelCount(content, "Current Bead")
	if beadFields != 1 {
		addFinding(findings, path, "plan.current_bead_cardinality", "exactly one Current Bead field is required; found %d", beadFields)
	} else if len(beadMatches) != 1 {
		addFinding(findings, path, "plan.current_bead_format", "Current Bead must use canonical form M3-<prefix><digits> (display ID <prefix>-<digits>)")
	} else {
		metadata.currentBead = strings.ToUpper(beadMatches[0][1])
		metadata.currentDisplay = strings.ToUpper(beadMatches[0][4])
		derivedDisplay := strings.ToUpper(beadMatches[0][2] + "-" + beadMatches[0][3])
		if metadata.currentDisplay != derivedDisplay {
			addFinding(findings, path, "plan.current_bead_display", "%s must declare display ID %s, not %s", metadata.currentBead, derivedDisplay, metadata.currentDisplay)
		}
	}
	switch metadata.phase {
	case planPhaseContractPublication, planPhaseDelivery:
	case "":
	default:
		addFinding(findings, path, "plan.phase", "unsupported plan phase %q; allowed values are %s and %s", metadata.phase, planPhaseContractPublication, planPhaseDelivery)
	}
	return metadata
}

func singleMetadataValue(content, path, field string, pattern *regexp.Regexp, findings *[]Finding) string {
	fields := metadataLabelCount(content, field)
	if fields != 1 {
		addFinding(findings, path, "plan.metadata_cardinality", "exactly one %s field is required; found %d", field, fields)
		return ""
	}
	matches := pattern.FindAllStringSubmatch(content, -1)
	if len(matches) != 1 {
		addFinding(findings, path, "plan.metadata_value", "%s field has an invalid value or format", field)
		return ""
	}
	return strings.ToLower(strings.TrimSpace(matches[0][1]))
}

func metadataLabelCount(content, field string) int {
	label := "**" + strings.ToLower(field) + ":**"
	count := 0
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			line = strings.TrimSpace(line[2:])
		}
		if strings.HasPrefix(strings.ToLower(line), label) {
			count++
		}
	}
	return count
}

func checkPlanLineage(root, path, content string, metadata activePlanMetadata, linkedArtifacts map[string]bool, findings *[]Finding) {
	if !linkedArtifacts["docs/goals/active.md"] {
		addFinding(findings, path, "plan.lineage_link", "active plan must link durable artifact docs/goals/active.md")
	} else if metadata.goal != "" {
		goalData, err := readRepoFile(root, "docs/goals/active.md")
		if err == nil && !regexp.MustCompile(`\b`+regexp.QuoteMeta(metadata.goal)+`\b`).Match(goalData) {
			addFinding(findings, path, "plan.lineage_goal", "docs/goals/active.md does not declare current goal %s", metadata.goal)
		}
	}
	if !hasLinkedArtifactWithPrefix(linkedArtifacts, "docs/product-decisions/") {
		addFinding(findings, path, "plan.lineage_link", "active plan must link at least one product decision")
	}

	if metadata.currentFeature == "" {
		return
	}
	featurePrefix := "docs/features/" + metadata.currentFeature + "-"
	var featureTargets []string
	for target := range linkedArtifacts {
		if strings.HasPrefix(target, featurePrefix) && strings.HasSuffix(target, ".md") {
			featureTargets = append(featureTargets, target)
		}
	}
	sort.Strings(featureTargets)
	if len(featureTargets) != 1 {
		addFinding(findings, path, "plan.lineage_feature", "active plan must link exactly one feature artifact for %s; found %d", metadata.currentFeature, len(featureTargets))
		return
	}

	feature := parseFeatureContract(root, featureTargets[0], metadata.currentFeature, findings)
	if feature.goal != "" && metadata.goal != "" && feature.goal != metadata.goal {
		addFinding(findings, path, "plan.feature_goal", "%s goal %s does not match active-plan goal %s", featureTargets[0], feature.goal, metadata.goal)
	}
	if feature.currentBead != "" && metadata.currentBead != "" && (feature.currentBead != metadata.currentBead || feature.currentDisplay != metadata.currentDisplay) {
		addFinding(findings, path, "plan.feature_bead", "%s work authority %s (%s) does not match active-plan Current Bead %s (%s)", featureTargets[0], feature.currentBead, feature.currentDisplay, metadata.currentBead, metadata.currentDisplay)
	}
	for _, decision := range feature.productDecisions {
		prefix := "docs/product-decisions/" + decision + "-"
		matches := 0
		for target := range linkedArtifacts {
			if strings.HasPrefix(target, prefix) && strings.HasSuffix(target, ".md") {
				matches++
			}
		}
		if matches != 1 {
			addFinding(findings, path, "plan.feature_decision_link", "active plan must link exactly one product-decision artifact for %s required by %s; found %d", decision, featureTargets[0], matches)
		}
	}
	if feature.productSpec != "" {
		if !strings.HasPrefix(feature.productSpec, "docs/product-specs/") || !strings.HasSuffix(feature.productSpec, ".md") {
			addFinding(findings, path, "plan.feature_specification", "%s declares invalid product specification %q", featureTargets[0], feature.productSpec)
		} else if !repoFileExists(root, feature.productSpec) {
			addFinding(findings, path, "plan.broken_link", "feature product specification %q does not exist", feature.productSpec)
		} else if !linkedArtifacts[feature.productSpec] {
			addFinding(findings, path, "plan.lineage_specification", "active plan must link %s declared by %s", feature.productSpec, featureTargets[0])
		}
	}
	checkPlanScenarioPriority(content, path, metadata.currentFeature, feature.scenarios, findings)
}

func hasLinkedArtifactWithPrefix(linkedArtifacts map[string]bool, prefix string) bool {
	for target := range linkedArtifacts {
		if strings.HasPrefix(target, prefix) && strings.HasSuffix(target, ".md") {
			return true
		}
	}
	return false
}

func parseFeatureContract(root, featurePath, featureID string, findings *[]Finding) featureContractMetadata {
	var metadata featureContractMetadata
	data, err := readRepoFile(root, featurePath)
	if err != nil {
		addFinding(findings, featurePath, "plan.feature_unreadable", "current feature contract cannot be read")
		return metadata
	}
	content := string(data)

	goalValues := featureFieldValues(content, "Goal")
	if len(goalValues) != 1 || !goalIDPattern.MatchString(goalValues[0]) {
		addFinding(findings, featurePath, "plan.feature_goal", "feature must declare exactly one canonical Goal field")
	} else {
		metadata.goal = goalValues[0]
	}

	decisionValues := featureFieldValues(content, "Product decision", "Product decisions")
	if len(decisionValues) != 1 || !decisionListPattern.MatchString(decisionValues[0]) {
		addFinding(findings, featurePath, "plan.feature_decisions", "feature must declare one canonical Product decision or Product decisions field")
	} else {
		metadata.productDecisions = decisionIDPattern.FindAllString(decisionValues[0], -1)
		seen := make(map[string]bool)
		for _, decision := range metadata.productDecisions {
			if seen[decision] {
				addFinding(findings, featurePath, "plan.feature_decisions", "feature declares duplicate product decision %s", decision)
			}
			seen[decision] = true
		}
	}

	specificationValues := featureFieldValues(content, "Product specification")
	if len(specificationValues) != 1 {
		addFinding(findings, featurePath, "plan.feature_specification", "feature must declare exactly one Product specification field")
	} else if specification, ok := normalizeFeatureArtifactReference(featurePath, specificationValues[0]); !ok {
		addFinding(findings, featurePath, "plan.feature_specification", "feature Product specification must be one repository-local artifact path")
	} else {
		metadata.productSpec = specification
	}

	canonicalBeads := featureFieldValues(content, "Canonical Bead")
	workAuthorities := featureFieldValues(content, "Work authority")
	if len(canonicalBeads)+len(workAuthorities) != 1 {
		addFinding(findings, featurePath, "plan.feature_bead", "feature must declare exactly one Canonical Bead or Work authority field")
	} else {
		var match []string
		if len(canonicalBeads) == 1 {
			match = featureBeadPattern.FindStringSubmatch(canonicalBeads[0])
		} else {
			match = workBeadPattern.FindStringSubmatch(workAuthorities[0])
		}
		if len(match) != 5 {
			addFinding(findings, featurePath, "plan.feature_bead", "feature work authority must use one canonical Bead and matching display ID")
		} else {
			metadata.currentBead = strings.ToUpper(match[1])
			metadata.currentDisplay = strings.ToUpper(match[4])
			derivedDisplay := strings.ToUpper(match[2] + "-" + match[3])
			if metadata.currentDisplay != derivedDisplay {
				addFinding(findings, featurePath, "plan.feature_bead", "%s must declare display ID %s", metadata.currentBead, derivedDisplay)
			}
		}
	}

	metadata.scenarios = parseFeatureScenarioSchedule(content, featurePath, featureID, findings)
	return metadata
}

func featureFieldValues(content string, labels ...string) []string {
	var values []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			line = strings.TrimSpace(line[2:])
		}
		for _, label := range labels {
			prefix := "**" + strings.ToLower(label) + ":**"
			if strings.HasPrefix(strings.ToLower(line), prefix) {
				values = append(values, strings.TrimSpace(line[len(prefix):]))
				break
			}
		}
	}
	return values
}

func normalizeFeatureArtifactReference(featurePath, value string) (string, bool) {
	if strings.HasPrefix(value, "`") && strings.HasSuffix(value, "`") && len(value) > 1 {
		value = strings.TrimSpace(value[1 : len(value)-1])
	} else if match := markdownLink.FindStringSubmatch(value); len(match) == 2 && match[0] == value {
		value = strings.TrimSpace(match[1])
	}
	if value == "" || strings.Contains(value, "://") || strings.Contains(value, "#") {
		return "", false
	}
	valuePath := filepath.FromSlash(value)
	if filepath.IsAbs(valuePath) {
		return "", false
	}
	var target string
	if strings.HasPrefix(filepath.ToSlash(valuePath), "docs/") {
		target = filepath.Clean(valuePath)
	} else {
		target = filepath.Clean(filepath.Join(filepath.Dir(filepath.FromSlash(featurePath)), valuePath))
	}
	if target == ".." || strings.HasPrefix(target, ".."+string(filepath.Separator)) {
		return "", false
	}
	return cleanPublicPath(target), true
}

func parseFeatureScenarioSchedule(content, featurePath, featureID string, findings *[]Finding) []featureScenario {
	lines := strings.Split(content, "\n")
	headingIndex, sectionEnd, headingCount := markdownSection(lines, "## Scenario schedule")
	if headingCount != 1 {
		addFinding(findings, featurePath, "plan.scenario_schedule", "current feature must contain exactly one ## Scenario schedule section; found %d", headingCount)
		return nil
	}
	headerIndex := firstNonblankLine(lines, headingIndex+1, sectionEnd)
	if headerIndex < 0 || headerIndex+1 >= sectionEnd {
		addFinding(findings, featurePath, "plan.scenario_schedule", "Scenario schedule must begin with its Markdown table")
		return nil
	}
	headers := splitMarkdownRow(lines[headerIndex])
	separators := splitMarkdownRow(lines[headerIndex+1])
	if len(headers) < 4 || len(headers) != len(separators) {
		addFinding(findings, featurePath, "plan.scenario_schedule", "Scenario schedule must begin with its Markdown table")
		return nil
	}
	for _, separator := range separators {
		if !markdownTableRule.MatchString(separator) {
			addFinding(findings, featurePath, "plan.scenario_schedule", "Scenario schedule table separator is invalid")
			return nil
		}
	}
	wantedHeaders := []string{"scenario", "state", "verificationowner", "requiredevidence"}
	if len(headers) != len(wantedHeaders) {
		addFinding(findings, featurePath, "plan.scenario_schedule", "Scenario schedule must use exactly Scenario, State, Verification owner, and Required evidence columns in that order")
		return nil
	}
	for column, header := range headers {
		if normalizeKey(header) != wantedHeaders[column] {
			addFinding(findings, featurePath, "plan.scenario_schedule", "Scenario schedule must use exactly Scenario, State, Verification owner, and Required evidence columns in that order")
			return nil
		}
	}
	const (
		scenarioColumn = iota
		stateColumn
		ownerColumn
		evidenceColumn
	)

	stableID := regexp.MustCompile(`^` + regexp.QuoteMeta(featureID) + `-S[1-9]\d*$`)
	allowedStates := map[string]bool{
		"failing": true, "passing": true, "blocked": true, "deferred": true,
		"descoped": true, "superseded": true,
	}
	seen := make(map[string]bool)
	var scenarios []featureScenario
	tableEnded := false
	for rowIndex := headerIndex + 2; rowIndex < sectionEnd; rowIndex++ {
		line := strings.TrimSpace(lines[rowIndex])
		if !looksLikeMarkdownTableRow(line) {
			tableEnded = true
			continue
		}
		if tableEnded {
			addFinding(findings, featurePath, "plan.scenario_schedule_row", "Scenario schedule contains a trailing or shadow table row after the contiguous schedule")
			continue
		}
		row := splitMarkdownRow(line)
		if len(row) != len(headers) {
			addFinding(findings, featurePath, "plan.scenario_schedule_row", "Scenario schedule row must have exactly %d columns and both boundary pipes", len(headers))
			continue
		}
		id := strings.TrimSpace(row[scenarioColumn])
		if !stableID.MatchString(id) {
			addFinding(findings, featurePath, "plan.scenario_id", "scenario schedule row must use a stable %s-S<number> ID", featureID)
			continue
		}
		if seen[id] {
			addFinding(findings, featurePath, "plan.scenario_id", "scenario schedule declares %s more than once", id)
		}
		seen[id] = true
		state := strings.ToLower(strings.TrimSpace(row[stateColumn]))
		if !allowedStates[state] {
			addFinding(findings, featurePath, "plan.scenario_state", "%s has unsupported scenario state %q", id, state)
		}
		if strings.TrimSpace(row[ownerColumn]) == "" || strings.TrimSpace(row[evidenceColumn]) == "" {
			addFinding(findings, featurePath, "plan.scenario_schedule_row", "%s must declare a verification owner and required evidence", id)
		}
		scenarios = append(scenarios, featureScenario{id: id, state: state})
	}
	if len(scenarios) == 0 {
		addFinding(findings, featurePath, "plan.scenario_schedule", "current feature scenario schedule must contain at least one scenario")
	}
	checkScenarioHeadingEquality(content, featurePath, featureID, scenarios, findings)
	return scenarios
}

func checkScenarioHeadingEquality(content, featurePath, featureID string, scenarios []featureScenario, findings *[]Finding) {
	headingPattern := regexp.MustCompile(`^###\s+(F-\d{3}-S[1-9]\d*)(?:\s+.*)?$`)
	headingPrefix := "### " + featureID + "-S"
	scheduled := make(map[string]bool, len(scenarios))
	for _, scenario := range scenarios {
		scheduled[scenario.id] = true
	}
	detailed := make(map[string]bool)
	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(raw)
		match := headingPattern.FindStringSubmatch(line)
		if len(match) != 2 {
			if strings.HasPrefix(line, headingPrefix) {
				addFinding(findings, featurePath, "plan.scenario_heading", "detailed scenario heading must use a stable %s-S<number> ID", featureID)
			}
			continue
		}
		id := match[1]
		if !strings.HasPrefix(id, featureID+"-S") {
			addFinding(findings, featurePath, "plan.scenario_heading", "detailed scenario heading %s belongs to a different feature", id)
			continue
		}
		if detailed[id] {
			addFinding(findings, featurePath, "plan.scenario_heading", "detailed scenario heading %s is declared more than once", id)
		}
		detailed[id] = true
	}
	for id := range scheduled {
		if !detailed[id] {
			addFinding(findings, featurePath, "plan.scenario_heading", "scheduled scenario %s has no matching detailed scenario heading", id)
		}
	}
	for id := range detailed {
		if !scheduled[id] {
			addFinding(findings, featurePath, "plan.scenario_heading", "detailed scenario heading %s is absent from the scenario schedule", id)
		}
	}
}

func checkPlanScenarioPriority(content, path, featureID string, scenarios []featureScenario, findings *[]Finding) {
	lines := strings.Split(content, "\n")
	headingIndex, sectionEnd, headingCount := markdownSection(lines, "## Scenario priority")
	if headingCount != 1 {
		addFinding(findings, path, "plan.scenario_priority", "active plan must contain exactly one ## Scenario priority section; found %d", headingCount)
		return
	}
	known := make(map[string]bool, len(scenarios))
	var failing []string
	for _, scenario := range scenarios {
		known[scenario.id] = true
		if scenario.state == "failing" {
			failing = append(failing, scenario.id)
		}
	}
	if len(failing) == 0 {
		addFinding(findings, path, "plan.scenario_priority", "current feature must have at least one failing scenario for an active plan")
	}

	section := strings.Join(lines[headingIndex+1:sectionEnd], "\n")
	references := anyScenarioPattern.FindAllString(section, -1)
	seen := make(map[string]bool)
	var priority []string
	for _, id := range references {
		if !strings.HasPrefix(id, featureID+"-S") {
			addFinding(findings, path, "plan.scenario_reference", "Scenario priority references %s outside current feature %s", id, featureID)
			continue
		}
		if !known[id] {
			addFinding(findings, path, "plan.scenario_reference", "Scenario priority references nonexistent scenario %s", id)
		}
		if seen[id] {
			addFinding(findings, path, "plan.scenario_priority", "Scenario priority references %s more than once", id)
		}
		seen[id] = true
		priority = append(priority, id)
	}
	if !equalStringOrder(priority, failing) {
		addFinding(findings, path, "plan.scenario_priority", "Scenario priority must list every failing %s scenario exactly once in schedule order", featureID)
	}
}

func equalStringOrder(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func checkPlanManifestProjection(root, path string, metadata activePlanMetadata, currentState string, findings *[]Finding) {
	const manifestPath = ".harness/manifest.yaml"
	data, err := readRepoFile(root, manifestPath)
	if err != nil {
		addFinding(findings, path, "plan.manifest_projection", "%s is required to project active-plan selection", manifestPath)
		return
	}
	if reason := validatePlanManifestSyntax(data); reason != "" {
		addFinding(findings, manifestPath, "plan.manifest_yaml", "manifest must use unambiguous data-only YAML: %s", reason)
		return
	}
	if sections := yamlScalarOccurrences(data, "authority"); len(sections) != 1 || sections[0] != "" {
		addFinding(findings, manifestPath, "plan.manifest_yaml", "manifest must contain exactly one block-style authority mapping")
		return
	}
	checks := []struct {
		key     string
		want    string
		finding string
	}{
		{key: "authority.current_bead", want: metadata.currentBead, finding: "plan.manifest_current_bead"},
		{key: "authority.plan_phase", want: metadata.phase, finding: "plan.manifest_phase"},
	}
	if currentState != "" {
		checks = append(checks, struct {
			key     string
			want    string
			finding string
		}{key: "authority.current_bead_state", want: currentState, finding: "plan.manifest_state"})
	}
	for _, check := range checks {
		values := yamlScalarOccurrences(data, check.key)
		if len(values) != 1 {
			addFinding(findings, path, check.finding, "%s must occur exactly once in the manifest; found %d", check.key, len(values))
		} else if check.want != "" && values[0] != check.want {
			addFinding(findings, path, check.finding, "%s must equal %q from the active plan; found %q", check.key, check.want, values[0])
		}
	}

	wantClaimed := ""
	switch metadata.phase {
	case planPhaseContractPublication:
		wantClaimed = "false"
	case planPhaseDelivery:
		wantClaimed = "true"
	}
	claimedValues := yamlScalarOccurrences(data, "authority.current_bead_claimed")
	if len(claimedValues) != 1 {
		addFinding(findings, path, "plan.manifest_claimed", "authority.current_bead_claimed must occur exactly once in the manifest; found %d", len(claimedValues))
	} else if wantClaimed != "" && claimedValues[0] != wantClaimed {
		addFinding(findings, path, "plan.manifest_claimed", "authority.current_bead_claimed must be %s during %s; found %q", wantClaimed, metadata.phase, claimedValues[0])
	}
}

func validatePlanManifestSyntax(data []byte) string {
	for _, raw := range strings.Split(string(data), "\n") {
		if strings.Contains(raw, "\t") {
			return "tabs are prohibited"
		}
		trimmed := strings.TrimSpace(strings.TrimRight(raw, "\r"))
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "%") || trimmed == "---" || strings.HasPrefix(trimmed, "--- ") || trimmed == "..." || strings.HasPrefix(trimmed, "... ") {
			return "directives and multiple documents are prohibited"
		}
		if strings.ContainsAny(trimmed, "{}[]") {
			return "flow collections are prohibited"
		}
		line := trimmed
		isSequence := strings.HasPrefix(line, "- ")
		if isSequence {
			line = strings.TrimSpace(line[2:])
		}
		if strings.HasPrefix(line, "?") {
			return "explicit mapping keys are prohibited"
		}
		colon := strings.Index(line, ":")
		if colon < 0 {
			if !isSequence || hasYAMLIndirectionToken(line) {
				return "only plain scalar sequence items and mappings are permitted"
			}
			continue
		}
		key := strings.TrimSpace(line[:colon])
		if !plainYAMLKeyPattern.MatchString(key) {
			return "mapping keys must be unquoted plain identifiers"
		}
		value := trimYAMLScalar(line[colon+1:])
		if strings.HasPrefix(value, "|") || strings.HasPrefix(value, ">") || hasYAMLIndirectionToken(value) {
			return "anchors, aliases, tags, merges, and block scalars are prohibited"
		}
	}
	return ""
}

func hasYAMLIndirectionToken(value string) bool {
	for _, field := range strings.Fields(value) {
		field = strings.Trim(field, " ,")
		if strings.HasPrefix(field, "&") || strings.HasPrefix(field, "*") || strings.HasPrefix(field, "!") || strings.HasPrefix(field, "<<:") {
			return true
		}
	}
	return false
}

// yamlScalarOccurrences preserves duplicate paths so the plan check fails
// closed when the Git projection is ambiguous. yamlScalars intentionally
// exposes only a convenient map and therefore cannot provide this invariant.
func yamlScalarOccurrences(data []byte, dottedPath string) []string {
	var values []string
	var stack []yamlContext
	for _, raw := range strings.Split(string(data), "\n") {
		raw = strings.TrimRight(raw, " \t\r")
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "-") {
			continue
		}
		colon := strings.Index(trimmed, ":")
		if colon < 1 {
			continue
		}
		indent := len(raw) - len(strings.TrimLeft(raw, " "))
		key := strings.TrimSpace(trimmed[:colon])
		value := trimYAMLScalar(trimmed[colon+1:])
		for len(stack) > 0 && stack[len(stack)-1].indent >= indent {
			stack = stack[:len(stack)-1]
		}
		parts := make([]string, 0, len(stack)+1)
		for _, context := range stack {
			parts = append(parts, context.key)
		}
		parts = append(parts, key)
		if strings.Join(parts, ".") == dottedPath {
			values = append(values, value)
		}
		if value == "" {
			stack = append(stack, yamlContext{indent: indent, key: key})
		}
	}
	return values
}

func checkDeliveryTable(content, path string, metadata activePlanMetadata, findings *[]Finding) string {
	lines := strings.Split(content, "\n")
	headingIndex, sectionEnd, headingCount := markdownSection(lines, "## Delivery waves")
	if headingCount != 1 {
		addFinding(findings, path, "plan.delivery_section_cardinality", "active plan must contain exactly one ## Delivery waves section; found %d", headingCount)
		return ""
	}
	headerIndex := firstNonblankLine(lines, headingIndex+1, sectionEnd)
	layout, canonicalTable := deliveryTableAt(lines, headerIndex, sectionEnd)
	shadowTables := 0
	for index := 0; index+1 < len(lines); index++ {
		if _, ok := deliveryTableAt(lines, index, len(lines)); ok && (!canonicalTable || index != headerIndex) {
			shadowTables++
		}
	}
	if shadowTables != 0 {
		addFinding(findings, path, "plan.delivery_shadow_table", "delivery-like tables are allowed only immediately under ## Delivery waves; found %d shadow table(s)", shadowTables)
	}
	if !canonicalTable {
		addFinding(findings, path, "plan.delivery_table", "the first nonblank line under ## Delivery waves must be the delivery table header")
		return ""
	}

	known := map[string]bool{"Genesis": true}
	rows := 0
	activeRows := 0
	currentRows := 0
	currentState := ""
	for rowIndex := headerIndex + 2; rowIndex < sectionEnd; rowIndex++ {
		line := strings.TrimSpace(lines[rowIndex])
		if line == "" || !looksLikeMarkdownTableRow(line) {
			continue
		}
		row := splitMarkdownRow(line)
		if len(row) != len(layout.headers) {
			addFinding(findings, path, "plan.delivery_row_format", "delivery table row must have exactly %d columns and both boundary pipes", len(layout.headers))
			continue
		}
		ids := canonicalTicketIDs(row[layout.ticketColumn])
		if len(ids) != 1 {
			addFinding(findings, path, "plan.ticket_cell", "each delivery row must declare exactly one canonical Bead display ID; found %d", len(ids))
			continue
		}
		rows++
		if strings.TrimSpace(row[layout.ownerColumn]) == "" || strings.TrimSpace(row[layout.evidenceColumn]) == "" {
			addFinding(findings, path, "plan.incomplete_row", "%s must declare owner and exit evidence", ids[0])
		}
		state := strings.ToLower(strings.TrimSpace(row[layout.stateColumn]))
		switch state {
		case "backlog", "done", "superseded":
		case "in-progress", "in-review":
			activeRows++
			if metadata.currentDisplay != "" && ids[0] != metadata.currentDisplay {
				addFinding(findings, path, "plan.current_state", "active row %s does not match Current Bead display ID %s", ids[0], metadata.currentDisplay)
			}
		default:
			addFinding(findings, path, "plan.invalid_state", "%s has unsupported state %q", ids[0], state)
		}
		dependencies, validDependencies := parseDependencyCell(row[layout.dependencyColumn])
		if !validDependencies {
			addFinding(findings, path, "plan.dependency_cell", "%s dependencies must be exact, unique comma-separated display IDs or the signed genesis sentinel", ids[0])
		}
		if strings.TrimSpace(row[layout.dependencyColumn]) == "signed genesis" && ids[0] != "H-001" {
			addFinding(findings, path, "plan.dependency_cell", "only H-001 may use the signed genesis dependency sentinel")
		}
		for _, dependency := range dependencies {
			if !known[dependency] {
				addFinding(findings, path, "plan.dependency_order", "%s depends on %s before it is declared", ids[0], dependency)
			}
		}
		for _, id := range ids {
			if id == metadata.currentDisplay {
				currentRows++
				currentState = state
			}
			if known[id] {
				addFinding(findings, path, "plan.duplicate_ticket", "delivery table declares %s more than once", id)
			}
			known[id] = true
		}
	}
	if rows == 0 {
		addFinding(findings, path, "plan.delivery_rows", "delivery table must contain at least one ticket")
	}
	if metadata.currentDisplay != "" && currentRows != 1 {
		addFinding(findings, path, "plan.current_row_cardinality", "delivery table must contain Current Bead display ID %s exactly once; found %d", metadata.currentDisplay, currentRows)
	}
	switch metadata.phase {
	case planPhaseContractPublication:
		if activeRows != 0 {
			addFinding(findings, path, "plan.active_work_cardinality", "%s phase permits no in-progress or in-review work item; found %d", planPhaseContractPublication, activeRows)
		}
		if metadata.currentDisplay != "" && currentRows == 1 && currentState != "backlog" {
			addFinding(findings, path, "plan.current_state", "%s must remain backlog during %s; found %s", metadata.currentDisplay, planPhaseContractPublication, currentState)
		}
	case planPhaseDelivery:
		if activeRows != 1 {
			addFinding(findings, path, "plan.active_work_cardinality", "%s phase requires exactly one in-progress or in-review work item; found %d", planPhaseDelivery, activeRows)
		}
		if metadata.currentDisplay != "" && currentRows == 1 && currentState != "in-progress" && currentState != "in-review" {
			addFinding(findings, path, "plan.current_state", "%s must be in-progress or in-review during %s; found %s", metadata.currentDisplay, planPhaseDelivery, currentState)
		}
	}
	return currentState
}

func deliveryTableAt(lines []string, index, end int) (deliveryTableLayout, bool) {
	if index < 0 || index+1 >= end || index+1 >= len(lines) {
		return deliveryTableLayout{}, false
	}
	headers := splitMarkdownRow(lines[index])
	separators := splitMarkdownRow(lines[index+1])
	if len(headers) < 4 || len(headers) != len(separators) {
		return deliveryTableLayout{}, false
	}
	for _, field := range separators {
		if !markdownTableRule.MatchString(field) {
			return deliveryTableLayout{}, false
		}
	}
	normalized := make(map[string]int)
	for column, header := range headers {
		key := normalizeKey(header)
		if key == "" {
			return deliveryTableLayout{}, false
		}
		if _, duplicate := normalized[key]; duplicate {
			return deliveryTableLayout{}, false
		}
		normalized[key] = column
	}
	ticketColumn, hasTicket := uniqueTableColumn(normalized, "ticket", "tickets", "workitem", "bead")
	_, hasWave := uniqueTableColumn(normalized, "wave")
	dependencyColumn, hasDependency := uniqueTableColumn(normalized, "dependencies", "dependson", "dependency")
	evidenceColumn, hasEvidence := uniqueTableColumn(normalized, "exitevidence", "evidence")
	ownerColumn, hasOwner := uniqueTableColumn(normalized, "owner")
	stateColumn, hasState := uniqueTableColumn(normalized, "state", "status")
	if !hasTicket || !hasWave || !hasDependency || !hasEvidence || !hasOwner || !hasState {
		return deliveryTableLayout{}, false
	}
	return deliveryTableLayout{
		headers:          headers,
		ticketColumn:     ticketColumn,
		dependencyColumn: dependencyColumn,
		evidenceColumn:   evidenceColumn,
		ownerColumn:      ownerColumn,
		stateColumn:      stateColumn,
	}, true
}

func looksLikeMarkdownTableRow(line string) bool {
	line = strings.TrimSpace(line)
	return strings.Contains(line, "|")
}

func canonicalTicketIDs(value string) []string {
	var ids []string
	for _, index := range ticketIDPattern.FindAllStringIndex(value, -1) {
		if index[0] > 0 && isDisplayIDContinuation(value[index[0]-1]) {
			continue
		}
		if index[1] < len(value) && isDisplayIDContinuation(value[index[1]]) {
			continue
		}
		ids = append(ids, value[index[0]:index[1]])
	}
	return ids
}

func parseDependencyCell(value string) ([]string, bool) {
	value = strings.TrimSpace(value)
	if value == "signed genesis" {
		return nil, true
	}
	if value == "" {
		return nil, false
	}
	parts := strings.Split(value, ",")
	dependencies := make([]string, 0, len(parts))
	seen := make(map[string]bool, len(parts))
	for _, part := range parts {
		dependency := strings.TrimSpace(part)
		if !dependencyIDPattern.MatchString(dependency) || seen[dependency] {
			return nil, false
		}
		seen[dependency] = true
		dependencies = append(dependencies, dependency)
	}
	return dependencies, true
}

func isDisplayIDContinuation(value byte) bool {
	return value == '-' || value == '_' || value >= '0' && value <= '9' || value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z'
}

func markdownSection(lines []string, heading string) (headingIndex, sectionEnd, count int) {
	headingIndex = -1
	for index, line := range lines {
		if strings.TrimSpace(line) == heading {
			count++
			headingIndex = index
		}
	}
	if count != 1 {
		return headingIndex, len(lines), count
	}
	sectionEnd = len(lines)
	for index := headingIndex + 1; index < len(lines); index++ {
		trimmed := strings.TrimSpace(lines[index])
		if strings.HasPrefix(trimmed, "## ") && !strings.HasPrefix(trimmed, "### ") {
			sectionEnd = index
			break
		}
	}
	return headingIndex, sectionEnd, count
}

func firstNonblankLine(lines []string, start, end int) int {
	for index := start; index < end && index < len(lines); index++ {
		if strings.TrimSpace(lines[index]) != "" {
			return index
		}
	}
	return -1
}

func splitMarkdownRow(line string) []string {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") {
		return nil
	}
	line = strings.TrimPrefix(strings.TrimSuffix(line, "|"), "|")
	parts := strings.Split(line, "|")
	for index := range parts {
		parts[index] = strings.TrimSpace(parts[index])
	}
	return parts
}

func firstTableColumn(columns map[string]int, names ...string) (int, bool) {
	for _, name := range names {
		if column, ok := columns[name]; ok {
			return column, true
		}
	}
	return 0, false
}

func uniqueTableColumn(columns map[string]int, names ...string) (int, bool) {
	column := 0
	matches := 0
	for _, name := range names {
		if found, ok := columns[name]; ok {
			column = found
			matches++
		}
	}
	return column, matches == 1
}

func checkNoGitTicketAuthority(root string, findings *[]Finding) {
	directory := filepath.Join(root, "docs", "tickets")
	_ = filepath.WalkDir(directory, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".md") || strings.EqualFold(entry.Name(), "README.md") {
			return nil
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr == nil {
			addFinding(findings, cleanPublicPath(relative), "plan.duplicate_ticket_authority", "editable lifecycle tickets belong in Beads, not Git")
		}
		return nil
	})
}

func formatPlanReference(id string) string {
	return fmt.Sprintf("work item %s", id)
}
