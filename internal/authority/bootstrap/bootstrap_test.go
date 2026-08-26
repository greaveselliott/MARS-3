/*
FactoryDocSync:
docs:
- docs/features/F-002-work-authority.md
- docs/design-docs/ADR-001-git-beads-authority.md
- docs/code-documentation-map.md
*/

package bootstrap

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/greaveselliott/MARS-3/internal/doctrine"
)

func TestVerifyPreimageBindsDependencyAndLineage(t *testing.T) {
	metadata := json.RawMessage(`{"contractPublicationRequired":true,"coordinator":"delivery-orchestrator","displayId":"W-001","exclusivePaths":["internal/authority/**","cmd/mars3-authority/**","api/authority/**","database/authority/**","deploy/authority/**","docs/evidence/W-001-validation.md","go.mod","go.sum","Makefile","NOTICE","THIRD_PARTY_NOTICES"],"failureOwnership":"foundation","featureId":"F-002","goalIds":["G-001"],"lifecycleState":"backlog","productDecisionIds":["PD-002"],"publicDisclosure":true,"risk":"critical","scenarioIds":["F-002-S1","F-002-S2","F-002-S3","F-002-S4","F-002-S5","F-002-S6"],"schemaVersion":1,"verificationOrder":["qa","security-reviewer","delivery-orchestrator"],"workType":"enabler"}`)
	issue := issueRecord{
		ID: "M3-W001", Status: "open", Assignee: "work-authority-engineer",
		CreatedAt: mustTime(t, "2026-08-26T05:09:05Z"), UpdatedAt: mustTime(t, "2026-08-26T06:22:03Z"),
		Metadata: metadata, Labels: []string{"public-first", "foundation", "enabler", "critical", "backlog"},
	}
	issue.Deps = append(issue.Deps, struct {
		ID             string          `json:"id"`
		Status         string          `json:"status"`
		DependencyType string          `json:"dependency_type"`
		Metadata       json.RawMessage `json:"metadata"`
	}{ID: "M3-H001", Status: "closed", DependencyType: "blocks", Metadata: json.RawMessage(`{"lifecycleState":"done"}`)})
	config := doctrine.W001BootstrapGrant{
		Bead: "M3-W001", Assignee: "work-authority-engineer", ExpectedNativeStatus: "open", ExpectedLifecycleState: "backlog",
		ExpectedCreatedAt: "2026-08-26T05:09:05Z", ExpectedUpdatedAt: "2026-08-26T06:22:03Z",
		ExpectedMetadataSHA256: "10c61003cb39518f57905620fcc0c47d29950fe82ae8d98a3111a057fa554dba",
		ExpectedLabelsSHA256:   "be506df06d8c206a3919a71a57e8aaacd2b5e1e233e25bafc2f5f87f306b188c",
		ExpectedDependency:     "M3-H001", ExpectedDependencyType: "blocks", ExpectedDependencyStatus: "closed", ExpectedDependencyLifecycle: "done",
		ExpectedDependencySHA256: "3ad0bca78b14e4e1fd5544477f131c0a86dd8a4d4e9563d43fa4ae1c202f4100",
		ExpectedLineageSHA256:    "9f3e91b4b642dc740898347c35e8f38abc35cc3ac1be83c81fe122cc308eaced",
	}
	if err := verifyPreimage(issue, config); err != nil {
		t.Fatalf("expected exact preimage to pass: %v", err)
	}

	issue.Deps[0].DependencyType = "related"
	if err := verifyPreimage(issue, config); err == nil {
		t.Fatal("expected dependency type drift to fail")
	}
	issue.Deps[0].DependencyType = "blocks"
	issue.Assignee = "different-principal"
	if err := verifyPreimage(issue, config); err == nil {
		t.Fatal("expected lineage principal drift to fail")
	}
}

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
