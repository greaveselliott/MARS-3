/*
FactoryDocSync:
docs:
- docs/features/F-002-work-authority.md
- docs/design-docs/ADR-001-git-beads-authority.md
- docs/code-documentation-map.md
*/

package bootstrap

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/greaveselliott/MARS-3/internal/doctrine"
)

func TestVerifyAtomicityTestResultsRequiresEveryExecutedCase(t *testing.T) {
	tests := []string{
		"TestBatchBootstrapClaimIsOneAtomicTransition",
		"TestBatchBootstrapClaimPreconditionFailureRollsBack",
		"TestBatchBootstrapClaimPostClaimFailureRollsBack",
		"TestBatchBootstrapClaimContentionHasOneWinner",
	}
	var output bytes.Buffer
	for _, name := range tests {
		encoded, err := json.Marshal(map[string]string{"Action": "pass", "Test": name})
		if err != nil {
			t.Fatal(err)
		}
		output.Write(encoded)
		output.WriteByte('\n')
	}
	if err := verifyAtomicityTestResults(output.Bytes()); err != nil {
		t.Fatalf("complete atomicity suite was rejected: %v", err)
	}
	if err := verifyAtomicityTestResults(bytes.Replace(output.Bytes(), []byte(`"Action":"pass","Test":"TestBatchBootstrapClaimContentionHasOneWinner"`), []byte(`"Action":"skip","Test":"TestBatchBootstrapClaimContentionHasOneWinner"`), 1)); err == nil {
		t.Fatal("skipped contention case was accepted")
	}
}

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

func TestPostclaimLabelDigestMatchesDeclaredTransition(t *testing.T) {
	labels := []string{"public-first", "foundation", "enabler", "critical", "backlog"}
	for index, label := range labels {
		if label == "backlog" {
			labels[index] = "in-progress"
		}
	}
	const declared = "3e4e77e20ee7a46dd77c4a9884dee51aa9f0fa9f2445099a0cb457d72cb83bbb"
	if actual := digestLabels(labels); actual != declared {
		t.Fatalf("postclaim label digest = %s, want %s", actual, declared)
	}
}

func TestExecutionAuthorizationCannotChangeDuringConformance(t *testing.T) {
	initial := doctrine.W001BootstrapExecutionAuthorization{MergedCommit: strings.Repeat("1", 40), ProtectedMainCheckRun: 17}
	if err := requireUnchangedExecutionAuthorization(initial, initial); err != nil {
		t.Fatalf("unchanged authorization was rejected: %v", err)
	}
	fresh := initial
	fresh.ProtectedMainCheckRun++
	if err := requireUnchangedExecutionAuthorization(initial, fresh); err == nil {
		t.Fatal("changed authorization was accepted")
	}
}

func TestWorkspaceInstanceRejectsAClone(t *testing.T) {
	const projectID = "11111111-2222-3333-4444-555555555555"
	makeWorkspace := func(root string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Join(root, ".beads", "embeddeddolt", "M3"), 0o755); err != nil {
			t.Fatal(err)
		}
		metadata := []byte(`{"project_id":"` + projectID + `","database":"dolt"}`)
		if err := os.WriteFile(filepath.Join(root, ".beads", "metadata.json"), metadata, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	original, clone := t.TempDir(), t.TempDir()
	makeWorkspace(original)
	makeWorkspace(clone)
	config := doctrine.W001BootstrapGrant{AuthorityProjectID: projectID}
	originalDigest, err := verifyWorkspace(original, config)
	if err != nil {
		t.Fatal(err)
	}
	cloneDigest, err := verifyWorkspace(clone, config)
	if err != nil {
		t.Fatal(err)
	}
	if originalDigest == cloneDigest {
		t.Fatal("copied workspace instance was indistinguishable from the canonical instance")
	}
}

func TestGitOutputIgnoresAmbientAuthorityOverrides(t *testing.T) {
	correct, attacker := t.TempDir(), t.TempDir()
	for _, root := range []string{correct, attacker} {
		command := exec.Command(bootstrapGitExecutable, "init", "--quiet", root)
		command.Env = bootstrapGitEnvironment()
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("initialize Git fixture: %v: %s", err, output)
		}
	}
	t.Setenv("GIT_DIR", filepath.Join(attacker, ".git"))
	t.Setenv("GIT_WORK_TREE", attacker)
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(attacker, "redirect.gitconfig"))
	t.Setenv("GIT_NO_REPLACE_OBJECTS", "0")
	t.Setenv("HTTPS_PROXY", "http://proxy.example.invalid")
	t.Setenv("SSL_CERT_FILE", filepath.Join(attacker, "ca.pem"))
	topLevel, err := gitOutput(correct, "rev-parse", "--show-toplevel")
	if err != nil {
		t.Fatal(err)
	}
	resolvedCorrect, err := filepath.EvalSymlinks(correct)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Clean(strings.TrimSpace(topLevel)) != filepath.Clean(resolvedCorrect) {
		t.Fatalf("ambient Git authority redirected the audited root to %q", strings.TrimSpace(topLevel))
	}
}

func TestRemoteMainParserRequiresOneCanonicalLowercaseSHA(t *testing.T) {
	valid := strings.Repeat("a", 40)
	resolved, err := parseRemoteMain([]byte(valid + "\trefs/heads/main\n"))
	if err != nil || resolved != valid {
		t.Fatalf("valid remote main was rejected: %q: %v", resolved, err)
	}
	for _, invalid := range []string{
		strings.Repeat("A", 40) + "\trefs/heads/main\n",
		strings.Repeat("z", 40) + "\trefs/heads/main\n",
		strings.Repeat("a", 39) + "\trefs/heads/main\n",
		valid + "\trefs/heads/other\n",
		valid + "\trefs/heads/main\n" + valid + "\trefs/heads/main\n",
	} {
		if _, err := parseRemoteMain([]byte(invalid)); err == nil {
			t.Fatalf("non-canonical remote main was accepted: %q", invalid)
		}
	}
}

func TestBeadsCommandEnvironmentRemovesAmbientWorkspaceOverrides(t *testing.T) {
	for _, key := range []string{"BEADS_DIR", "BEADS_DB", "DOLT_HOST", "BD_CONFIG", "GT_ROOT"} {
		t.Setenv(key, "attacker-controlled")
	}
	for _, value := range beadsCommandEnvironment() {
		key := strings.ToUpper(strings.SplitN(value, "=", 2)[0])
		if strings.HasPrefix(key, "BEADS_") || strings.HasPrefix(key, "DOLT_") || strings.HasPrefix(key, "BD_") || strings.HasPrefix(key, "GT_") {
			t.Fatalf("ambient Beads override %q survived sanitization", key)
		}
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
