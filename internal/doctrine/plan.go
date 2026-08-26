/*
FactoryDocSync:
docs:
- docs/features/F-001-doctrine-foundation.md
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
	ticketIDPattern     = regexp.MustCompile(`\b(?:UI|[A-Z])-\d{3}\b`)
	currentBeadLine     = regexp.MustCompile(`(?im)^\s*(?:[-*]\s*)?(?:\*\*)?current(?:\s+claimed)?\s+bead(?:\s*:\s*\*\*|\*\*\s*:)\s*`)
	activeStatusLine    = regexp.MustCompile(`(?im)^\s*(?:[-*]\s*)?(?:\*\*)?status(?:\s*:\s*\*\*|\*\*\s*:|\s*:)\s*active\s*$`)
	planOwnerLine       = regexp.MustCompile(`(?im)^\s*(?:[-*]\s*)?(?:\*\*)?owner(?:\s*:\s*\*\*|\*\*\s*:|\s*:)\s*delivery orchestrator\s*$`)
	markdownLink        = regexp.MustCompile(`\[[^\]]+\]\(([^)]+)\)`)
	currentBeadValue    = regexp.MustCompile(`(?i)M3-[A-Z0-9-]+`)
	markdownTableRule   = regexp.MustCompile(`^\s*:?-{3,}:?\s*$`)
)

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
	if !activeStatusLine.MatchString(content) {
		addFinding(findings, path, "plan.status", "active plan must declare Status: Active")
	}
	if !planOwnerLine.MatchString(content) {
		addFinding(findings, path, "plan.owner", "active plan must declare Delivery Orchestrator as owner")
	}
	for _, requiredField := range []string{"**Goal:** G-001", "**Current feature:** F-001", "**Current Bead:** M3-H001", "**Authority:**"} {
		if !strings.Contains(content, requiredField) {
			addFinding(findings, path, "plan.required_metadata", "active plan must declare %s", requiredField)
		}
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
	for _, required := range []string{"G-001", "F-001-S1", "F-001-S2", "F-001-S3", "F-001-S4", "H-001", "M3-H001"} {
		if !strings.Contains(content, required) {
			addFinding(findings, path, "plan.required_link", "active plan must link %s", required)
		}
	}
	if !strings.Contains(strings.ToLower(content), "beads") || !strings.Contains(strings.ToLower(content), "git") {
		addFinding(findings, path, "plan.authority_split", "active plan must state the Beads work authority and Git doctrine authority")
	}
	beadLines := currentBeadLine.FindAllStringIndex(content, -1)
	if len(beadLines) != 1 {
		addFinding(findings, path, "plan.current_bead_cardinality", "exactly one Current Bead field is required; found %d", len(beadLines))
	} else {
		lineEnd := strings.IndexByte(content[beadLines[0][1]:], '\n')
		if lineEnd < 0 {
			lineEnd = len(content) - beadLines[0][1]
		}
		value := content[beadLines[0][1] : beadLines[0][1]+lineEnd]
		if currentBeadValue.FindString(value) != "M3-H001" {
			addFinding(findings, path, "plan.current_bead", "the current Bead must be M3-H001 during H-001")
		}
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
	for _, requiredTarget := range []string{
		"docs/goals/active.md",
		"docs/product-decisions/PD-001-public-first.md",
		"docs/product-decisions/PD-002-git-beads-authority.md",
		"docs/product-decisions/PD-003-provider-neutral.md",
		"docs/product-specs/foundation.md",
		"docs/features/F-001-doctrine-foundation.md",
	} {
		if !linkedArtifacts[requiredTarget] {
			addFinding(findings, path, "plan.lineage_link", "active plan must link durable artifact %s", requiredTarget)
		}
	}
	checkDeliveryTable(content, path, findings)
}

func checkDeliveryTable(content, path string, findings *[]Finding) {
	lines := strings.Split(content, "\n")
	for index := 0; index+2 < len(lines); index++ {
		headers := splitMarkdownRow(lines[index])
		separators := splitMarkdownRow(lines[index+1])
		if len(headers) < 4 || len(headers) != len(separators) {
			continue
		}
		validRule := true
		for _, field := range separators {
			if !markdownTableRule.MatchString(field) {
				validRule = false
				break
			}
		}
		if !validRule {
			continue
		}
		normalized := make(map[string]int)
		for column, header := range headers {
			normalized[normalizeKey(header)] = column
		}
		ticketColumn, hasTicket := firstTableColumn(normalized, "ticket", "tickets", "workitem", "bead")
		_, hasWave := firstTableColumn(normalized, "wave")
		dependencyColumn, hasDependency := firstTableColumn(normalized, "dependencies", "dependson", "dependency")
		evidenceColumn, hasEvidence := firstTableColumn(normalized, "exitevidence", "evidence")
		ownerColumn, hasOwner := firstTableColumn(normalized, "owner")
		stateColumn, hasState := firstTableColumn(normalized, "state", "status")
		if !hasTicket || !hasWave || !hasDependency || !hasEvidence || !hasOwner || !hasState {
			continue
		}
		known := map[string]bool{"Genesis": true}
		rows := 0
		activeRows := 0
		for rowIndex := index + 2; rowIndex < len(lines); rowIndex++ {
			row := splitMarkdownRow(lines[rowIndex])
			if len(row) != len(headers) {
				break
			}
			ids := ticketIDPattern.FindAllString(row[ticketColumn], -1)
			if len(ids) == 0 {
				continue
			}
			rows++
			if strings.TrimSpace(row[ownerColumn]) == "" || strings.TrimSpace(row[evidenceColumn]) == "" {
				addFinding(findings, path, "plan.incomplete_row", "%s must declare owner and exit evidence", ids[0])
			}
			state := strings.ToLower(strings.TrimSpace(row[stateColumn]))
			switch state {
			case "backlog", "done", "superseded":
			case "in-progress", "in-review":
				activeRows++
				if ids[0] != "H-001" {
					addFinding(findings, path, "plan.current_state", "only H-001 may be current during the foundation milestone")
				}
			default:
				addFinding(findings, path, "plan.invalid_state", "%s has unsupported state %q", ids[0], state)
			}
			for _, dependency := range ticketIDPattern.FindAllString(row[dependencyColumn], -1) {
				if !known[dependency] {
					addFinding(findings, path, "plan.dependency_order", "%s depends on %s before it is declared", ids[0], dependency)
				}
			}
			for _, id := range ids {
				if known[id] {
					addFinding(findings, path, "plan.duplicate_ticket", "delivery table declares %s more than once", id)
				}
				known[id] = true
			}
		}
		if rows == 0 {
			addFinding(findings, path, "plan.delivery_rows", "delivery table must contain at least one ticket")
		}
		if activeRows != 1 {
			addFinding(findings, path, "plan.active_work_cardinality", "delivery table must contain exactly one in-progress or in-review work item; found %d", activeRows)
		}
		return
	}
	addFinding(findings, path, "plan.delivery_table", "active plan needs Wave, Bead, Owner, Dependencies, State, and Exit evidence columns")
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
