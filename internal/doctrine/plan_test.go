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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlanAllowsBoundedContractPublicationWithoutAClaim(t *testing.T) {
	repo := writePlanFixture(t, planPhaseContractPublication, "backlog", "backlog")
	findings, err := CheckPlan(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Fatalf("valid contract-publication transition was rejected: %v", findings)
	}
}

func TestPlanRequiresClaimedCurrentBeadDuringDelivery(t *testing.T) {
	for _, state := range []string{"in-progress", "in-review"} {
		t.Run(state, func(t *testing.T) {
			repo := writePlanFixture(t, planPhaseDelivery, state, "backlog")
			findings, err := CheckPlan(repo)
			if err != nil {
				t.Fatal(err)
			}
			if len(findings) != 0 {
				t.Fatalf("valid delivery phase was rejected: %v", findings)
			}
		})
	}
}

func TestPlanRejectsUnsafePhaseTransitions(t *testing.T) {
	tests := []struct {
		name        string
		phase       string
		current     string
		parallel    string
		findingCode string
	}{
		{name: "contract publication claims current work", phase: planPhaseContractPublication, current: "in-progress", parallel: "backlog", findingCode: "plan.active_work_cardinality"},
		{name: "contract publication activates other work", phase: planPhaseContractPublication, current: "backlog", parallel: "in-progress", findingCode: "plan.current_state"},
		{name: "delivery leaves current work unclaimed", phase: planPhaseDelivery, current: "backlog", parallel: "backlog", findingCode: "plan.active_work_cardinality"},
		{name: "delivery activates a different bead", phase: planPhaseDelivery, current: "backlog", parallel: "in-progress", findingCode: "plan.current_state"},
		{name: "unsupported phase", phase: "planning", current: "backlog", parallel: "backlog", findingCode: "plan.phase"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			repo := writePlanFixture(t, testCase.phase, testCase.current, testCase.parallel)
			findings, err := CheckPlan(repo)
			if err != nil {
				t.Fatal(err)
			}
			if !findingCodePresent(findings, testCase.findingCode) {
				t.Fatalf("unsafe transition was accepted; wanted %s, got %v", testCase.findingCode, findings)
			}
		})
	}
}

func TestPlanBindsCurrentMetadataToTableAndFeatureLineage(t *testing.T) {
	tests := []struct {
		name        string
		mutate      func(string) string
		findingCode string
	}{
		{
			name: "mismatched display identifier",
			mutate: func(plan string) string {
				return strings.Replace(plan, "M3-W001 (display ID W-001)", "M3-W001 (display ID H-001)", 1)
			},
			findingCode: "plan.current_bead_display",
		},
		{
			name: "current bead absent from delivery table",
			mutate: func(plan string) string {
				return strings.Replace(plan, "W-001 Work Authority", "A-001 Work Authority", 1)
			},
			findingCode: "plan.current_row_cardinality",
		},
		{
			name: "current feature artifact absent",
			mutate: func(plan string) string {
				return strings.Replace(plan, "[F-002](../../features/F-002-work-authority.md)", "F-002", 1)
			},
			findingCode: "plan.lineage_feature",
		},
		{
			name: "feature specification absent from lineage",
			mutate: func(plan string) string {
				return strings.Replace(plan, "- Product promise: [work authority](../../product-specs/work-authority.md)\n", "", 1)
			},
			findingCode: "plan.lineage_specification",
		},
		{
			name: "duplicate phase metadata",
			mutate: func(plan string) string {
				return strings.Replace(plan, "**Phase:** contract-publication", "**Phase:** contract-publication\n**Phase:** contract-publication", 1)
			},
			findingCode: "plan.metadata_cardinality",
		},
		{
			name: "second malformed current bead field",
			mutate: func(plan string) string {
				return strings.Replace(plan, "**Current Bead:** M3-W001 (display ID W-001)", "**Current Bead:** M3-W001 (display ID W-001)\n**Current Bead:** not-canonical", 1)
			},
			findingCode: "plan.current_bead_cardinality",
		},
		{
			name: "noncanonical extra bead separator",
			mutate: func(plan string) string {
				return strings.Replace(plan, "M3-W001 (display ID W-001)", "M3-W-001 (display ID W-001)", 1)
			},
			findingCode: "plan.current_bead_format",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			repo := writePlanFixture(t, planPhaseContractPublication, "backlog", "backlog")
			planPath := filepath.Join(repo, filepath.FromSlash(canonicalActivePlan))
			data, err := os.ReadFile(planPath)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(planPath, []byte(testCase.mutate(string(data))), 0o644); err != nil {
				t.Fatal(err)
			}
			findings, err := CheckPlan(repo)
			if err != nil {
				t.Fatal(err)
			}
			if !findingCodePresent(findings, testCase.findingCode) {
				t.Fatalf("metadata or lineage mismatch was accepted; wanted %s, got %v", testCase.findingCode, findings)
			}
		})
	}
}

func TestPlanRejectsFeatureSpecificationOutsideProductSpecs(t *testing.T) {
	repo := writePlanFixture(t, planPhaseContractPublication, "backlog", "backlog")
	writePlanFile(t, repo, "docs/features/F-002-work-authority.md", "# F-002\n\n**Product specification:** `../../../private.md`\n")
	assertPlanFinding(t, repo, "plan.feature_specification")
}

func TestPlanBindsFeatureAuthorityMetadata(t *testing.T) {
	tests := []struct {
		name        string
		oldValue    string
		newValue    string
		findingCode string
	}{
		{name: "different goal", oldValue: "**Goal:** G-001", newValue: "**Goal:** G-999", findingCode: "plan.feature_goal"},
		{name: "unlinked product decision", oldValue: "**Product decision:** PD-002", newValue: "**Product decision:** PD-003", findingCode: "plan.feature_decision_link"},
		{name: "different canonical bead", oldValue: "**Canonical Bead:** M3-W001 (display ID W-001)", newValue: "**Canonical Bead:** M3-P001 (display ID P-001)", findingCode: "plan.feature_bead"},
		{name: "duplicate authority fields", oldValue: "**Canonical Bead:** M3-W001 (display ID W-001)", newValue: "**Canonical Bead:** M3-W001 (display ID W-001)\n**Work authority:** external Bead M3-W001 (display ID W-001)", findingCode: "plan.feature_bead"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			repo := writePlanFixture(t, planPhaseContractPublication, "backlog", "backlog")
			featurePath := filepath.Join(repo, "docs", "features", "F-002-work-authority.md")
			data, err := os.ReadFile(featurePath)
			if err != nil {
				t.Fatal(err)
			}
			mutated := strings.Replace(string(data), testCase.oldValue, testCase.newValue, 1)
			if mutated == string(data) {
				t.Fatalf("feature fixture does not contain %q", testCase.oldValue)
			}
			if err := os.WriteFile(featurePath, []byte(mutated), 0o644); err != nil {
				t.Fatal(err)
			}
			assertPlanFinding(t, repo, testCase.findingCode)
		})
	}
}

func TestFeatureMetadataAcceptsWorkAuthorityVariant(t *testing.T) {
	repo := t.TempDir()
	path := "docs/features/F-003-local-substrate.md"
	writePlanFile(t, repo, path, `# F-003

**Goal:** G-001
**Product decisions:** PD-001, PD-003
**Product specification:** `+"`docs/product-specs/local-substrate.md`"+`
**Work authority:** external Bead M3-P001 (display ID P-001)

## Scenario schedule

| Scenario | State | Verification owner | Required evidence |
| --- | --- | --- | --- |
| F-003-S1 | failing | QA | deterministic evidence |

### F-003-S1 — Pinned local substrate
`)
	var findings []Finding
	metadata := parseFeatureContract(repo, path, "F-003", &findings)
	if len(findings) != 0 {
		t.Fatalf("valid Work authority metadata was rejected: %v", findings)
	}
	if metadata.goal != "G-001" || metadata.currentBead != "M3-P001" || metadata.currentDisplay != "P-001" || !equalStringOrder(metadata.productDecisions, []string{"PD-001", "PD-003"}) {
		t.Fatalf("unexpected parsed metadata: %+v", metadata)
	}
}

func TestPlanBindsPriorityToCurrentFeatureScenarioSchedule(t *testing.T) {
	t.Run("nonexistent priority scenario", func(t *testing.T) {
		repo := writePlanFixture(t, planPhaseContractPublication, "backlog", "backlog")
		mutatePlanFixture(t, repo, func(plan string) string {
			return strings.Replace(plan, "F-002-S1", "F-002-S999", 1)
		})
		assertPlanFinding(t, repo, "plan.scenario_reference")
	})

	for _, testCase := range []struct {
		name        string
		oldValue    string
		newValue    string
		findingCode string
	}{
		{name: "missing verification owner", oldValue: "| F-002-S1 | failing | QA | deterministic evidence |", newValue: "| F-002-S1 | failing | | deterministic evidence |", findingCode: "plan.scenario_schedule_row"},
		{name: "missing required evidence", oldValue: "| F-002-S1 | failing | QA | deterministic evidence |", newValue: "| F-002-S1 | failing | QA | |", findingCode: "plan.scenario_schedule_row"},
		{name: "invalid scenario state", oldValue: "| F-002-S1 | failing | QA | deterministic evidence |", newValue: "| F-002-S1 | invented | QA | deterministic evidence |", findingCode: "plan.scenario_state"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			repo := writePlanFixture(t, planPhaseContractPublication, "backlog", "backlog")
			featurePath := filepath.Join(repo, "docs", "features", "F-002-work-authority.md")
			data, err := os.ReadFile(featurePath)
			if err != nil {
				t.Fatal(err)
			}
			mutated := strings.Replace(string(data), testCase.oldValue, testCase.newValue, 1)
			if mutated == string(data) {
				t.Fatalf("feature fixture does not contain %q", testCase.oldValue)
			}
			if err := os.WriteFile(featurePath, []byte(mutated), 0o644); err != nil {
				t.Fatal(err)
			}
			assertPlanFinding(t, repo, testCase.findingCode)
		})
	}
}

func TestPlanRejectsScenarioScheduleParserTruncation(t *testing.T) {
	repo := writePlanFixture(t, planPhaseContractPublication, "backlog", "backlog")
	featurePath := filepath.Join(repo, "docs", "features", "F-002-work-authority.md")
	data, err := os.ReadFile(featurePath)
	if err != nil {
		t.Fatal(err)
	}
	feature := strings.Replace(string(data),
		"| F-002-S1 | failing | QA | deterministic evidence |\n\n### F-002-S1 — Governed claim route",
		`| F-002-S1 | failing | QA | deterministic evidence |
F-002-S2 | passing | QA | deterministic evidence |
| F-002-S3 | passing | QA | deterministic evidence |
| F-002-S4 | passing | Security | deterministic evidence |
| F-002-S5 | passing | Security | deterministic evidence |
| F-002-S6 | passing | QA + Security | deterministic evidence |

### F-002-S1 — Governed claim route

### F-002-S2 — Compare-and-swap claim

### F-002-S3 — Monotonic lease

### F-002-S4 — Stale authority denial

### F-002-S5 — Direct access denial

### F-002-S6 — Ordered recovery`, 1)
	if feature == string(data) {
		t.Fatal("feature fixture did not contain the scenario schedule marker")
	}
	if err := os.WriteFile(featurePath, []byte(feature), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, err := CheckPlan(repo)
	if err != nil {
		t.Fatal(err)
	}
	for _, code := range []string{"plan.scenario_schedule_row", "plan.scenario_heading"} {
		if !findingCodePresent(findings, code) {
			t.Fatalf("malformed row truncation bypass was accepted; wanted %s, got %v", code, findings)
		}
	}
}

func TestPlanRejectsAmbiguousScenarioScheduleColumns(t *testing.T) {
	repo := writePlanFixture(t, planPhaseContractPublication, "backlog", "backlog")
	featurePath := filepath.Join(repo, "docs", "features", "F-002-work-authority.md")
	data, err := os.ReadFile(featurePath)
	if err != nil {
		t.Fatal(err)
	}
	feature := strings.Replace(string(data),
		"| Scenario | State | Verification owner | Required evidence |\n| --- | --- | --- | --- |\n| F-002-S1 | failing | QA | deterministic evidence |",
		"| Scenario | State | State | Verification owner | Required evidence |\n| --- | --- | --- | --- | --- |\n| F-002-S1 | passing | failing | QA | deterministic evidence |", 1)
	if feature == string(data) {
		t.Fatal("feature fixture did not contain the canonical scenario table")
	}
	if err := os.WriteFile(featurePath, []byte(feature), 0o644); err != nil {
		t.Fatal(err)
	}
	assertPlanFinding(t, repo, "plan.scenario_schedule")
}

func TestPlanRequiresScenarioScheduleAndDetailedHeadingsToMatch(t *testing.T) {
	tests := []struct {
		name     string
		oldValue string
		newValue string
	}{
		{
			name:     "scheduled scenario lacks detail",
			oldValue: "### F-002-S1 — Governed claim route",
			newValue: "### Context",
		},
		{
			name:     "detail is absent from schedule",
			oldValue: "### F-002-S1 — Governed claim route",
			newValue: "### F-002-S1 — Governed claim route\n\n### F-002-S2 — Undeclared detail",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			repo := writePlanFixture(t, planPhaseContractPublication, "backlog", "backlog")
			featurePath := filepath.Join(repo, "docs", "features", "F-002-work-authority.md")
			data, err := os.ReadFile(featurePath)
			if err != nil {
				t.Fatal(err)
			}
			feature := strings.Replace(string(data), testCase.oldValue, testCase.newValue, 1)
			if err := os.WriteFile(featurePath, []byte(feature), 0o644); err != nil {
				t.Fatal(err)
			}
			assertPlanFinding(t, repo, "plan.scenario_heading")
		})
	}
}

func TestPlanRejectsStaleOrContradictoryManifestProjection(t *testing.T) {
	tests := []struct {
		name        string
		phase       string
		current     string
		oldValue    string
		newValue    string
		findingCode string
	}{
		{name: "stale current bead", phase: planPhaseContractPublication, current: "backlog", oldValue: "current_bead: M3-W001", newValue: "current_bead: M3-H001", findingCode: "plan.manifest_current_bead"},
		{name: "stale phase", phase: planPhaseContractPublication, current: "backlog", oldValue: "plan_phase: contract-publication", newValue: "plan_phase: delivery", findingCode: "plan.manifest_phase"},
		{name: "contract publication claims bead", phase: planPhaseContractPublication, current: "backlog", oldValue: "current_bead_claimed: false", newValue: "current_bead_claimed: true", findingCode: "plan.manifest_claimed"},
		{name: "contract publication projects active state", phase: planPhaseContractPublication, current: "backlog", oldValue: "current_bead_state: backlog", newValue: "current_bead_state: in-progress", findingCode: "plan.manifest_state"},
		{name: "delivery denies claim", phase: planPhaseDelivery, current: "in-progress", oldValue: "current_bead_claimed: true", newValue: "current_bead_claimed: false", findingCode: "plan.manifest_claimed"},
		{name: "delivery projects backlog", phase: planPhaseDelivery, current: "in-progress", oldValue: "current_bead_state: in-progress", newValue: "current_bead_state: backlog", findingCode: "plan.manifest_state"},
		{name: "duplicate current bead projection", phase: planPhaseContractPublication, current: "backlog", oldValue: "current_bead: M3-W001", newValue: "current_bead: M3-H001\n  current_bead: M3-W001", findingCode: "plan.manifest_current_bead"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			repo := writePlanFixture(t, testCase.phase, testCase.current, "backlog")
			manifestPath := filepath.Join(repo, ".harness", "manifest.yaml")
			data, err := os.ReadFile(manifestPath)
			if err != nil {
				t.Fatal(err)
			}
			mutated := strings.Replace(string(data), testCase.oldValue, testCase.newValue, 1)
			if mutated == string(data) {
				t.Fatalf("manifest fixture does not contain %q", testCase.oldValue)
			}
			if err := os.WriteFile(manifestPath, []byte(mutated), 0o644); err != nil {
				t.Fatal(err)
			}
			assertPlanFinding(t, repo, testCase.findingCode)
		})
	}
}

func TestPlanRejectsManifestYAMLIndirection(t *testing.T) {
	tests := []struct {
		name      string
		injection string
	}{
		{
			name: "alias key",
			injection: "authority_name: &authority_key authority\n" +
				"*authority_key:\n" +
				"  current_bead: M3-H001\n",
		},
		{
			name: "merge key",
			injection: "authority_defaults: &authority_defaults\n" +
				"  current_bead: M3-H001\n" +
				"authority_overlay:\n" +
				"  <<: *authority_defaults\n",
		},
		{name: "explicit key", injection: "? shadow\n: value\n"},
		{name: "tag", injection: "shadow: !unsafe value\n"},
		{name: "directive", injection: "%YAML 1.2\n"},
		{name: "multiple documents", injection: "--- # second document\n"},
		{name: "quoted structural key", injection: "\"shadow\": value\n"},
		{name: "escaped structural key", injection: "shad\\u006fw: value\n"},
		{name: "tab indentation", injection: "shadow:\n\tvalue: unsafe\n"},
		{name: "flow mapping", injection: "shadow: { current_bead: M3-H001 }\n"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			repo := writePlanFixture(t, planPhaseContractPublication, "backlog", "backlog")
			manifestPath := filepath.Join(repo, ".harness", "manifest.yaml")
			data, err := os.ReadFile(manifestPath)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(manifestPath, []byte(testCase.injection+string(data)), 0o644); err != nil {
				t.Fatal(err)
			}
			assertPlanFinding(t, repo, "plan.manifest_yaml")
		})
	}
}

func TestPlanPreservesOrderingUniquenessAndAuthorityChecks(t *testing.T) {
	t.Run("dependency order", func(t *testing.T) {
		repo := writePlanFixture(t, planPhaseContractPublication, "backlog", "backlog")
		mutatePlanFixture(t, repo, func(plan string) string {
			foundation := "| 0 | H-001 Doctrine foundation | Foundation Maintainer | signed genesis | done | accepted evidence |\n"
			return strings.Replace(plan, foundation, "", 1) + foundation
		})
		assertPlanFinding(t, repo, "plan.dependency_order")
	})

	t.Run("duplicate ticket", func(t *testing.T) {
		repo := writePlanFixture(t, planPhaseContractPublication, "backlog", "backlog")
		mutatePlanFixture(t, repo, func(plan string) string {
			row := "| 1 | W-001 Work Authority | Work Authority Engineer | H-001 | backlog | gateway evidence |\n"
			return strings.Replace(plan, row, row+row, 1)
		})
		assertPlanFinding(t, repo, "plan.duplicate_ticket")
	})

	t.Run("Git shadow ticket", func(t *testing.T) {
		repo := writePlanFixture(t, planPhaseContractPublication, "backlog", "backlog")
		writePlanFile(t, repo, "docs/tickets/in-progress/W-001.md", "shadow authority\n")
		assertPlanFinding(t, repo, "plan.duplicate_ticket_authority")
	})
}

func TestPlanRejectsNoncanonicalDependencyCells(t *testing.T) {
	tests := []struct {
		name     string
		oldValue string
		newValue string
	}{
		{
			name:     "malformed token is not silently dropped",
			oldValue: "| 1 | P-001 Local substrate | Platform Engineer | H-001 | backlog | substrate evidence |",
			newValue: "| 1 | P-001 Local substrate | Platform Engineer | H-001, W001 | backlog | substrate evidence |",
		},
		{
			name:     "leftover prose is rejected",
			oldValue: "| 1 | P-001 Local substrate | Platform Engineer | H-001 | backlog | substrate evidence |",
			newValue: "| 1 | P-001 Local substrate | Platform Engineer | H-001 accepted | backlog | substrate evidence |",
		},
		{
			name:     "duplicate dependency is rejected",
			oldValue: "| 1 | P-001 Local substrate | Platform Engineer | H-001 | backlog | substrate evidence |",
			newValue: "| 1 | P-001 Local substrate | Platform Engineer | H-001, H-001 | backlog | substrate evidence |",
		},
		{
			name:     "genesis sentinel is foundation-only",
			oldValue: "| 1 | P-001 Local substrate | Platform Engineer | H-001 | backlog | substrate evidence |",
			newValue: "| 1 | P-001 Local substrate | Platform Engineer | signed genesis | backlog | substrate evidence |",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			repo := writePlanFixture(t, planPhaseContractPublication, "backlog", "backlog")
			mutatePlanFixture(t, repo, func(plan string) string {
				return strings.Replace(plan, testCase.oldValue, testCase.newValue, 1)
			})
			assertPlanFinding(t, repo, "plan.dependency_cell")
		})
	}
}

func TestPlanRejectsShadowDeliveryTableMaskingUnsafeRealTable(t *testing.T) {
	repo := writePlanFixture(t, planPhaseContractPublication, "backlog", "backlog")
	mutatePlanFixture(t, repo, func(plan string) string {
		heading := strings.Index(plan, "## Delivery waves")
		if heading < 0 {
			t.Fatal("plan fixture has no Delivery waves section")
		}
		section := strings.Replace(plan[heading:], "| 1 | W-001 Work Authority | Work Authority Engineer | H-001 | backlog | gateway evidence |", "| 1 | W-001 Work Authority | Work Authority Engineer | H-001 | in-progress | gateway evidence |", 1)
		shadow := `| Wave | Bead | Owner | Depends on | State | Exit evidence |
| --- | --- | --- | --- | --- | --- |
| 0 | H-001 Doctrine foundation | Foundation Maintainer | signed genesis | done | accepted evidence |
| 1 | W-001 Work Authority | Work Authority Engineer | H-001 | backlog | gateway evidence |

`
		return shadow + plan[:heading] + section
	})
	findings, err := CheckPlan(repo)
	if err != nil {
		t.Fatal(err)
	}
	for _, code := range []string{"plan.delivery_shadow_table", "plan.active_work_cardinality"} {
		if !findingCodePresent(findings, code) {
			t.Fatalf("shadow delivery table bypass was accepted; wanted %s, got %v", code, findings)
		}
	}
}

func TestPlanRejectsMalformedDeliveryRows(t *testing.T) {
	tests := []struct {
		name        string
		row         string
		findingCode string
	}{
		{
			name:        "active row without canonical display ID",
			row:         "| 1 | P001 Shadow platform work | Platform Engineer | H-001 | in-progress | unauthorized shadow execution |",
			findingCode: "plan.ticket_cell",
		},
		{
			name:        "row missing trailing boundary pipe",
			row:         "| 2 | T-001 Trace | Trace Engineer | W-001 | backlog | trace evidence",
			findingCode: "plan.delivery_row_format",
		},
		{
			name:        "active row missing both boundary pipes",
			row:         "1 | P001 Shadow platform work | Platform Engineer | H-001 | in-progress | unauthorized shadow execution",
			findingCode: "plan.delivery_row_format",
		},
		{
			name:        "row with trailing column",
			row:         "| 2 | T-001 Trace | Trace Engineer | W-001 | backlog | trace evidence | unauthorized |",
			findingCode: "plan.delivery_row_format",
		},
		{
			name:        "row with two canonical display IDs",
			row:         "| 2 | T-001 and S-001 combined | Trace Engineer | W-001 | backlog | trace evidence |",
			findingCode: "plan.ticket_cell",
		},
		{
			name:        "row with display ID embedded in a larger token",
			row:         "| 2 | T-001-shadow | Trace Engineer | W-001 | backlog | trace evidence |",
			findingCode: "plan.ticket_cell",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			repo := writePlanFixture(t, planPhaseContractPublication, "backlog", "backlog")
			mutatePlanFixture(t, repo, func(plan string) string {
				marker := "\n## Success evidence"
				return strings.Replace(plan, marker, "\nDelivery annotation that does not end the governed section.\n"+testCase.row+marker, 1)
			})
			assertPlanFinding(t, repo, testCase.findingCode)
		})
	}
}

func TestPlanRejectsAmbiguousDeliveryHeaderOrSeparator(t *testing.T) {
	tests := []struct {
		name     string
		oldValue string
		newValue string
	}{
		{
			name:     "duplicate normalized header",
			oldValue: "| Wave | Bead | Owner | Depends on | State | Exit evidence |",
			newValue: "| Wave | Bead | Owner | Owner | State | Exit evidence |",
		},
		{
			name:     "duplicate logical Bead header",
			oldValue: "| Wave | Bead | Owner | Depends on | State | Exit evidence |",
			newValue: "| Wave | Bead | Ticket | Depends on | State | Exit evidence |",
		},
		{
			name:     "invalid separator width",
			oldValue: "| --- | --- | --- | --- | --- | --- |",
			newValue: "| --- | --- | -- | --- | --- | --- |",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			repo := writePlanFixture(t, planPhaseContractPublication, "backlog", "backlog")
			mutatePlanFixture(t, repo, func(plan string) string {
				return strings.Replace(plan, testCase.oldValue, testCase.newValue, 1)
			})
			assertPlanFinding(t, repo, "plan.delivery_table")
		})
	}
}

func writePlanFixture(t *testing.T, phase, currentState, parallelState string) string {
	t.Helper()
	repo := t.TempDir()
	writePlanFile(t, repo, "docs/goals/active.md", "# G-001\n")
	writePlanFile(t, repo, "docs/product-decisions/PD-002-git-beads-authority.md", "# PD-002\n")
	writePlanFile(t, repo, "docs/product-specs/work-authority.md", "# Work authority\n")
	writePlanFile(t, repo, "docs/features/F-002-work-authority.md", `# F-002

**Goal:** G-001
**Product decision:** PD-002
**Product specification:** `+"`docs/product-specs/work-authority.md`"+`
**Canonical Bead:** M3-W001 (display ID W-001)

## Scenario schedule

| Scenario | State | Verification owner | Required evidence |
| --- | --- | --- | --- |
| F-002-S1 | failing | QA | deterministic evidence |

### F-002-S1 — Governed claim route
`)
	claimed := "false"
	if phase == planPhaseDelivery {
		claimed = "true"
	}
	manifest := "authority:\n" +
		"  current_bead: M3-W001\n" +
		"  current_bead_state: " + currentState + "\n" +
		"  current_bead_claimed: " + claimed + "\n" +
		"  plan_phase: " + phase + "\n"
	writePlanFile(t, repo, ".harness/manifest.yaml", manifest)
	plan := `# Active operating plan

**Status:** Active
**Owner:** Delivery Orchestrator
**Phase:** ` + phase + `
**Goal:** G-001
**Current feature:** F-002
**Current Bead:** M3-W001 (display ID W-001)
**Authority:** Beads/Dolt for work state; Git for this durable plan

## Durable lineage

- Goal: [G-001](../../goals/active.md)
- Decision: [PD-002](../../product-decisions/PD-002-git-beads-authority.md)
- Product promise: [work authority](../../product-specs/work-authority.md)
- Behavior contract: [F-002](../../features/F-002-work-authority.md)

## Current hypothesis and walking skeleton

Publish the F-002 contract before claiming M3-W001 in Beads.

## Scenario priority

1. F-002-S1 — one governed claim route.

## Delivery waves

| Wave | Bead | Owner | Depends on | State | Exit evidence |
| --- | --- | --- | --- | --- | --- |
| 0 | H-001 Doctrine foundation | Foundation Maintainer | signed genesis | done | accepted evidence |
| 1 | W-001 Work Authority | Work Authority Engineer | H-001 | ` + currentState + ` | gateway evidence |
| 1 | P-001 Local substrate | Platform Engineer | H-001 | ` + parallelState + ` | substrate evidence |

## Success evidence

The Git contract is merged before Beads records a W-001 claim.

## Falsification evidence

An implementation row becomes active before contract publication.

## Failure ownership and convergence

Foundation failures are classified before recovery.
`
	writePlanFile(t, repo, canonicalActivePlan, plan)
	return repo
}

func mutatePlanFixture(t *testing.T, repo string, mutate func(string) string) {
	t.Helper()
	path := filepath.Join(repo, filepath.FromSlash(canonicalActivePlan))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(mutate(string(data))), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writePlanFile(t *testing.T, repo, relative, content string) {
	t.Helper()
	path := filepath.Join(repo, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertPlanFinding(t *testing.T, repo, code string) {
	t.Helper()
	findings, err := CheckPlan(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !findingCodePresent(findings, code) {
		t.Fatalf("wanted %s, got %v", code, findings)
	}
}
