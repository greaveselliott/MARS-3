/*
FactoryDocSync:
docs:
- docs/features/F-002-work-authority.md
- docs/design-docs/ADR-001-git-beads-authority.md
- docs/code-documentation-map.md
*/

package beads

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	authorityv1 "github.com/greaveselliott/MARS-3/api/authority/v1"
	"github.com/greaveselliott/MARS-3/internal/authority/gateway"
)

type fixtureReader struct {
	issues map[string][]byte
	ids    []string
}

func (reader *fixtureReader) ReadIssue(_ context.Context, id string) ([]byte, error) {
	value, found := reader.issues[id]
	if !found {
		return nil, errors.New("not found with private backend detail")
	}
	return value, nil
}

func (reader *fixtureReader) ListIssueIDs(context.Context) ([]string, error) {
	return append([]string(nil), reader.ids...), nil
}

type fixtureMutator struct {
	result []byte
	err    error
	input  AtomicClaim
	calls  int
}

func (mutator *fixtureMutator) CompareAndSwapClaim(_ context.Context, input AtomicClaim) ([]byte, error) {
	mutator.calls++
	mutator.input = input
	return mutator.result, mutator.err
}

func TestStoreProjectsCanonicalBeadWithoutRawContent(t *testing.T) {
	reader := &fixtureReader{issues: map[string][]byte{"M3-W002": issueFixture(t, authorityv1.LifecycleBacklog, "", "", 1, false)}, ids: []string{"M3-W002"}}
	store := mustStore(t, reader, &fixtureMutator{})
	item, err := store.Get(context.Background(), "tenant-fixture", "project-fixture", "M3-W002")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if item.BeadID != "M3-W002" || item.DisplayID != "W-002" || item.LifecycleState != authorityv1.LifecycleBacklog || item.ClaimAttemptID != "" || item.Version.IssueMutationSequence != 1 {
		t.Fatalf("item=%#v", item)
	}
	wantPaths := []string{"go.mod", "internal/authority/"}
	if !reflect.DeepEqual(item.ExclusivePaths, wantPaths) || item.Integrity.Lineage == "" || item.Integrity.DependencyOutcomes == "" || item.Integrity.Blockers == "" || item.Integrity.ExclusivePaths == "" {
		t.Fatalf("paths=%#v integrity=%#v", item.ExclusivePaths, item.Integrity)
	}
	if len(item.Dependencies) != 1 || !item.Dependencies[0].ReviewAccepted || !item.Dependencies[0].RunCompleted || !item.Dependencies[0].Reconciled {
		t.Fatalf("dependencies=%#v", item.Dependencies)
	}
	encoded, _ := json.Marshal(item)
	for _, forbidden := range []string{"synthetic description must not project", "synthetic title must not project", "synthetic acceptance must not project"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("raw canonical content escaped projection: %s", encoded)
		}
	}
}

func TestStoreAtomicClaimBindsExpectedProjectionAndPostimage(t *testing.T) {
	preRaw := issueFixture(t, authorityv1.LifecycleBacklog, "", "", 1, false)
	preSnapshot, err := decodeIssueSnapshot(preRaw, "tenant-fixture", "project-fixture", []authorityv1.Label{authorityv1.LabelPublicAccepted})
	if err != nil {
		t.Fatal(err)
	}
	postRaw := issueFixture(t, authorityv1.LifecycleInProgress, "attempt-001", "work-authority-engineer", 2, false)
	reader := &fixtureReader{issues: map[string][]byte{"M3-W002": preRaw}}
	mutator := &fixtureMutator{result: postRaw}
	store := mustStore(t, reader, mutator)
	pre, err := store.Get(context.Background(), "tenant-fixture", "project-fixture", "M3-W002")
	if err != nil {
		t.Fatal(err)
	}
	post, err := store.CompareAndSwapClaim(context.Background(), gateway.ClaimMutation{
		TenantID: "tenant-fixture", ProjectID: "project-fixture", BeadID: pre.BeadID,
		ExpectedVersion: pre.Version, ExpectedIntegrity: pre.Integrity,
		AttemptID: "attempt-001", Assignee: "work-authority-engineer", IdempotencyKey: "key-a",
		BaseSHA: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	})
	if err != nil {
		t.Fatalf("CompareAndSwapClaim: %v", err)
	}
	if post.LifecycleState != authorityv1.LifecycleInProgress || post.ClaimAttemptID != "attempt-001" || post.Version.IssueMutationSequence != 2 || mutator.calls != 1 || mutator.input.ExpectedDigest == "" || mutator.input.ExpectedVersion != pre.Version || mutator.input.ExpectedIntegrity != pre.Integrity {
		t.Fatalf("post=%#v input=%#v calls=%d", post, mutator.input, mutator.calls)
	}
	if mutator.input.ExpectedStatus != preSnapshot.NativeStatus || mutator.input.ExpectedAssignee != preSnapshot.NativeAssignee ||
		mutator.input.ExpectedCreatedAt != preSnapshot.CreatedAt || mutator.input.ExpectedUpdatedAt != preSnapshot.UpdatedAt ||
		mutator.input.MetadataSHA256 != preSnapshot.MetadataSHA256 || mutator.input.LabelsSHA256 != preSnapshot.LabelsSHA256 ||
		mutator.input.DependenciesSHA256 != preSnapshot.DependenciesSHA256 || mutator.input.BaseCommit != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("atomic preconditions were not fully bound: %#v", mutator.input)
	}
	if !validAtomicClaim(mutator.input) {
		t.Fatal("captured atomic claim does not satisfy the native boundary")
	}
	stale := pre.Version
	stale.IssueMutationSequence++
	if _, err := store.CompareAndSwapClaim(context.Background(), gateway.ClaimMutation{
		TenantID: "tenant-fixture", ProjectID: "project-fixture", BeadID: pre.BeadID,
		ExpectedVersion: stale, ExpectedIntegrity: pre.Integrity,
		AttemptID: "attempt-002", Assignee: "work-authority-engineer", IdempotencyKey: "key-b",
		BaseSHA: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}); !errors.Is(err, gateway.ErrStaleWorkVersion) || mutator.calls != 1 {
		t.Fatalf("stale error=%v calls=%d", err, mutator.calls)
	}
}

func TestStoreFailsClosedOnUnknownMetadataAndNonAtomicMutation(t *testing.T) {
	raw := issueFixture(t, authorityv1.LifecycleBacklog, "", "", 1, true)
	store := mustStore(t, &fixtureReader{issues: map[string][]byte{"M3-W002": raw}}, &fixtureMutator{})
	if _, err := store.Get(context.Background(), "tenant-fixture", "project-fixture", "M3-W002"); !errors.Is(err, ErrProjectionInvalid) {
		t.Fatalf("unknown metadata error=%v", err)
	}
	preRaw := issueFixture(t, authorityv1.LifecycleBacklog, "", "", 1, false)
	reader := &fixtureReader{issues: map[string][]byte{"M3-W002": preRaw}}
	mutator := &fixtureMutator{err: errors.New("ordinary multi-command update is not atomic")}
	store = mustStore(t, reader, mutator)
	pre, _ := store.Get(context.Background(), "tenant-fixture", "project-fixture", "M3-W002")
	_, err := store.CompareAndSwapClaim(context.Background(), gateway.ClaimMutation{
		TenantID: "tenant-fixture", ProjectID: "project-fixture", BeadID: pre.BeadID,
		ExpectedVersion: pre.Version, ExpectedIntegrity: pre.Integrity,
		AttemptID: "attempt-001", Assignee: "work-authority-engineer", IdempotencyKey: "key-a",
		BaseSHA: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	})
	if !errors.Is(err, ErrAtomicCASRequired) {
		t.Fatalf("non-atomic mutation error=%v", err)
	}
}

func TestStoreRejectsDuplicateAuthorityKeys(t *testing.T) {
	raw := issueFixture(t, authorityv1.LifecycleBacklog, "", "", 1, false)
	raw = bytes.Replace(raw, []byte(`"lifecycleState":"backlog"`), []byte(`"lifecycleState":"done","lifecycleState":"backlog"`), 1)
	store := mustStore(t, &fixtureReader{issues: map[string][]byte{"M3-W002": raw}}, &fixtureMutator{})
	if _, err := store.Get(context.Background(), "tenant-fixture", "project-fixture", "M3-W002"); !errors.Is(err, ErrProjectionInvalid) {
		t.Fatalf("duplicate authority key error=%v", err)
	}
}

func TestClaimPostMetadataRejectsPresentClaimKeysAndTrailingJSON(t *testing.T) {
	base := `{"lifecycleState":"backlog","workVersion":{"issueMutationSequence":1}}`
	for name, raw := range map[string]string{
		"null work claim":      `{"lifecycleState":"backlog","workClaim":null,"workVersion":{"issueMutationSequence":1}}`,
		"null bootstrap claim": `{"lifecycleState":"backlog","bootstrapClaim":null,"workVersion":{"issueMutationSequence":1}}`,
		"duplicate lifecycle":  `{"lifecycleState":"done","lifecycleState":"backlog","workVersion":{"issueMutationSequence":1}}`,
		"trailing JSON":        base + `{}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := claimPostMetadata([]byte(raw), "attempt-001", "key-a", strings.Repeat("a", 40)); !errors.Is(err, ErrProjectionInvalid) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestNativeMutatorClaimAndReceiptValidationFailsClosed(t *testing.T) {
	pre := issueFixture(t, authorityv1.LifecycleBacklog, "", "", 1, false)
	snapshot, err := decodeIssueSnapshot(pre, "tenant-fixture", "project-fixture", []authorityv1.Label{authorityv1.LabelPublicAccepted})
	if err != nil {
		t.Fatal(err)
	}
	post, err := claimPostMetadata(snapshot.MetadataRaw, "attempt-001", "key-a", strings.Repeat("a", 40))
	if err != nil {
		t.Fatal(err)
	}
	claim := AtomicClaim{
		BeadID: "M3-W002", ExpectedVersion: snapshot.Item.Version, ExpectedIntegrity: snapshot.Item.Integrity,
		ExpectedDigest: projectionDigest(snapshot.Item), ExpectedStatus: snapshot.NativeStatus,
		ExpectedAssignee: snapshot.NativeAssignee, ExpectedCreatedAt: snapshot.CreatedAt, ExpectedUpdatedAt: snapshot.UpdatedAt,
		MetadataSHA256: snapshot.MetadataSHA256, LabelsSHA256: snapshot.LabelsSHA256,
		DependenciesSHA256: snapshot.DependenciesSHA256, AttemptID: "attempt-001", Assignee: "work-authority-engineer",
		IdempotencyKey: "key-a", BaseCommit: strings.Repeat("a", 40), PostMetadata: post,
	}
	if !validAtomicClaim(claim) {
		t.Fatal("canonical atomic claim rejected")
	}
	invalidVersion := claim
	invalidVersion.ExpectedVersion.IssueMutationSequence = 0
	if validAtomicClaim(invalidVersion) {
		t.Fatal("zero mutation sequence accepted")
	}
	invalidIntegrity := claim
	invalidIntegrity.ExpectedIntegrity.Lineage = strings.Repeat("A", 64)
	if validAtomicClaim(invalidIntegrity) {
		t.Fatal("noncanonical integrity digest accepted")
	}
	withBootstrap := claim
	withBootstrap.PostMetadata = bytes.Replace(post, []byte(`"workClaim":`), []byte(`"bootstrapClaim":null,"workClaim":`), 1)
	if validAtomicClaim(withBootstrap) {
		t.Fatal("present bootstrap claim accepted")
	}
	wrongAttempt := claim
	wrongAttempt.PostMetadata = bytes.Replace(post, []byte(`"attemptId":"attempt-001"`), []byte(`"attemptId":"attempt-other"`), 1)
	if validAtomicClaim(wrongAttempt) {
		t.Fatal("mismatched post metadata accepted")
	}

	validReceipt := []byte(`{"operations":1,"schema_version":1,"status":"ok","results":[{"line":1,"op":"authority-claim","target":"M3-W002"}]}`)
	if !validAuthorityClaimReceipt(validReceipt, "M3-W002") {
		t.Fatal("canonical receipt rejected")
	}
	for name, raw := range map[string][]byte{
		"missing schema":  []byte(`{"operations":1,"status":"ok","results":[{"line":1,"op":"authority-claim","target":"M3-W002"}]}`),
		"unknown field":   []byte(`{"operations":1,"schema_version":1,"status":"ok","extra":true,"results":[{"line":1,"op":"authority-claim","target":"M3-W002"}]}`),
		"duplicate field": []byte(`{"operations":1,"schema_version":1,"status":"ok","status":"ok","results":[{"line":1,"op":"authority-claim","target":"M3-W002"}]}`),
		"trailing JSON":   append(append([]byte(nil), validReceipt...), []byte(`{}`)...),
	} {
		t.Run(name, func(t *testing.T) {
			if validAuthorityClaimReceipt(raw, "M3-W002") {
				t.Fatal("malformed receipt accepted")
			}
		})
	}
}

func TestNativeMutatorEnvironmentHasNoAmbientAuthority(t *testing.T) {
	mutator := &NativeMutator{reader: &CLIReader{workspace: "/synthetic/workspace", projectID: "project-fixture"}}
	got := make(map[string]string)
	for _, entry := range mutator.environment("/synthetic/scratch", strings.Repeat("b", 64)) {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			t.Fatalf("malformed environment entry %q", entry)
		}
		if _, exists := got[parts[0]]; exists {
			t.Fatalf("duplicate environment key %q", parts[0])
		}
		got[parts[0]] = parts[1]
	}
	want := map[string]string{
		"BEADS_DIR": "/synthetic/workspace/.beads", "BEADS_DOLT_SERVER_DATABASE": "M3",
		"BEADS_DOLT_SHARED_SERVER": "0", "BEADS_DOLT_SERVER_MODE": "0", "BD_DISABLE_METRICS": "1",
		"BD_NO_HOOKS": "1", "HOME": "/synthetic/scratch", "XDG_CONFIG_HOME": "/synthetic/scratch/.config",
		"MARS3_AUTHORITY_DIRECT_BEADS_DIR": "/synthetic/workspace/.beads", "MARS3_AUTHORITY_PROJECT_ID": "project-fixture",
		"MARS3_AUTHORITY_DATABASE": "M3", "MARS3_AUTHORITY_WORKSPACE_SHA256": strings.Repeat("b", 64),
		"PATH": "/usr/bin:/bin", "NO_COLOR": "1", "LANG": "C", "TZ": "UTC",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mutator environment=%#v, want %#v", got, want)
	}
}

func TestCLIReaderProjectionIntegration(t *testing.T) {
	binary := os.Getenv("MARS3_TEST_BEADS_BINARY")
	workspace := os.Getenv("MARS3_TEST_BEADS_WORKSPACE")
	workspaceProjectID := os.Getenv("MARS3_TEST_BEADS_PROJECT_ID")
	if binary == "" || workspace == "" || workspaceProjectID == "" {
		t.Skip("set public synthetic Beads fixture paths and project ID to run integration conformance")
	}
	file, err := os.Open(binary)
	if err != nil {
		t.Fatal("open pinned fixture binary")
	}
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		file.Close()
		t.Fatal("hash pinned fixture binary")
	}
	file.Close()
	reader, err := NewCLIReader(binary, hex.EncodeToString(digest.Sum(nil)), workspace, workspaceProjectID)
	if err != nil {
		t.Fatalf("NewCLIReader: %v", err)
	}
	store := mustStore(t, reader, &fixtureMutator{})
	item, err := store.Get(context.Background(), "tenant-fixture", "project-fixture", "M3-W001")
	if err != nil {
		t.Fatalf("read canonical public fixture: %v", err)
	}
	if item.BeadID != "M3-W001" || item.LifecycleState != authorityv1.LifecycleInProgress || item.ClaimAttemptID == "" || item.Version.IssueMutationSequence == 0 || len(item.ExclusivePaths) == 0 {
		t.Fatalf("canonical projection is incomplete: id=%s lifecycle=%s attempt=%t version=%d paths=%d", item.BeadID, item.LifecycleState, item.ClaimAttemptID != "", item.Version.IssueMutationSequence, len(item.ExclusivePaths))
	}
}

func TestCLIReaderEnvironmentDisablesAmbientConfigAndTelemetry(t *testing.T) {
	reader := &CLIReader{workspace: "/synthetic/workspace"}
	got := make(map[string]string)
	for _, entry := range reader.environment("/synthetic/scratch-home") {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 || got[parts[0]] != "" {
			t.Fatalf("duplicate or malformed environment entry %q", entry)
		}
		got[parts[0]] = parts[1]
	}
	want := map[string]string{
		"BEADS_DIR": "/synthetic/workspace/.beads", "BEADS_DOLT_SERVER_DATABASE": "M3",
		"BEADS_DOLT_SHARED_SERVER": "0", "BEADS_DOLT_SERVER_MODE": "0",
		"BD_DISABLE_METRICS": "1", "BD_NO_HOOKS": "1", "HOME": "/synthetic/scratch-home",
		"XDG_CONFIG_HOME": "/synthetic/scratch-home/.config", "NO_COLOR": "1", "LANG": "C", "TZ": "UTC",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("reader environment=%#v, want %#v", got, want)
	}
}

func TestNativeMutatorIntegration(t *testing.T) {
	binary := os.Getenv("MARS3_TEST_BEADS_GATEWAY_BINARY")
	if binary == "" {
		t.Skip("set the reviewed patched Beads binary to run native CAS conformance")
	}
	root := t.TempDir()
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	runFixtureBD := func(args ...string) []byte {
		t.Helper()
		commandArgs := append([]string{"--actor", "work-authority-engineer", "--json"}, args...)
		if len(args) == 0 || args[0] != "init" {
			commandArgs = append([]string{"-C", root}, commandArgs...)
		}
		command := exec.Command(binary, commandArgs...)
		command.Dir = root
		command.Env = []string{
			"BD_DISABLE_METRICS=1", "BD_NO_HOOKS=1", "HOME=" + home,
			"XDG_CONFIG_HOME=" + filepath.Join(home, ".config"), "PATH=/usr/bin:/bin",
			"NO_COLOR=1", "LANG=C", "TZ=UTC",
		}
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("synthetic bd fixture command failed: %v (%s)", err, boundedFixtureError(output))
		}
		return output
	}
	runFixtureBD("init", "--prefix", "M3", "--database", "M3", "--non-interactive", "--skip-hooks", "--skip-agents")
	configPath := filepath.Join(root, ".beads", "config.yaml")
	if err := os.WriteFile(configPath, []byte("# public synthetic authority fixture\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	metadataRaw, err := os.ReadFile(filepath.Join(root, ".beads", "metadata.json"))
	if err != nil {
		t.Fatal(err)
	}
	var identity struct {
		ProjectID string `json:"project_id"`
	}
	if json.Unmarshal(metadataRaw, &identity) != nil || identity.ProjectID == "" {
		t.Fatal("synthetic Beads identity is unavailable")
	}
	dependencyMetadata := `{"schemaVersion":1,"displayId":"D-001","lifecycleState":"done","goalIds":["G-001"],"productDecisionIds":["PD-002"],"featureId":"F-002","scenarioIds":["F-002-S2"],"exclusivePaths":["docs/**"],"verificationOrder":["qa","security-reviewer","delivery-orchestrator"],"risk":"high","workType":"enabler","coordinator":"delivery-orchestrator","failureOwnership":"foundation","publicDisclosure":true,"reviewAccepted":true,"runDisposition":"completed","reconciled":true}`
	runFixtureBD("create", "synthetic dependency", "--id", "M3-D001", "--type", "task", "--priority", "1", "--metadata", dependencyMetadata, "--labels", "done")
	runFixtureBD("close", "M3-D001", "--reason", "synthetic verification complete")
	workMetadata := `{"schemaVersion":1,"displayId":"W-002","lifecycleState":"backlog","goalIds":["G-001"],"productDecisionIds":["PD-002"],"featureId":"F-002","scenarioIds":["F-002-S2"],"exclusivePaths":["internal/authority/**"],"verificationOrder":["qa","security-reviewer","delivery-orchestrator"],"workVersion":{"authorityGeneration":"gen-a","issueIncarnation":"inc-a","issueMutationSequence":1,"dependencyGraphRevision":1},"risk":"high","workType":"feature","coordinator":"delivery-orchestrator","failureOwnership":"foundation","publicDisclosure":true}`
	runFixtureBD("create", "synthetic work", "--id", "M3-W002", "--type", "task", "--priority", "1", "--metadata", workMetadata, "--labels", "backlog", "--deps", "M3-D001")

	file, err := os.Open(binary)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		file.Close()
		t.Fatal(err)
	}
	file.Close()
	binaryDigest := hex.EncodeToString(digest.Sum(nil))
	reader, err := NewCLIReader(binary, binaryDigest, root, identity.ProjectID)
	if err != nil {
		t.Fatalf("NewCLIReader: %v", err)
	}
	mutator, err := NewNativeMutator(reader, binary, binaryDigest)
	if err != nil {
		t.Fatalf("NewNativeMutator: %v", err)
	}
	store := mustStore(t, reader, mutator)
	pre, err := store.Get(context.Background(), "tenant-fixture", "project-fixture", "M3-W002")
	if err != nil {
		t.Fatalf("Get preimage: %v", err)
	}
	mutation := gateway.ClaimMutation{
		TenantID: "tenant-fixture", ProjectID: "project-fixture", BeadID: pre.BeadID,
		ExpectedVersion: pre.Version, ExpectedIntegrity: pre.Integrity,
		AttemptID: "attempt-native-001", Assignee: "work-authority-engineer",
		IdempotencyKey: "key-native", BaseSHA: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
	post, err := store.CompareAndSwapClaim(context.Background(), mutation)
	if err != nil {
		t.Fatalf("native CompareAndSwapClaim: %v", err)
	}
	if post.LifecycleState != authorityv1.LifecycleInProgress || post.Assignee != mutation.Assignee || post.ClaimAttemptID != mutation.AttemptID || post.Version.IssueMutationSequence != 2 {
		t.Fatalf("native postimage=%#v", post)
	}
	if _, err := store.CompareAndSwapClaim(context.Background(), mutation); !errors.Is(err, gateway.ErrStaleWorkVersion) {
		t.Fatalf("replayed stale claim error=%v", err)
	}
	head := strings.Repeat("b", 40)
	applyLifecycle := func(current authorityv1.WorkItem, transition gateway.LifecycleMutation) authorityv1.WorkItem {
		t.Helper()
		transition.TenantID, transition.ProjectID, transition.BeadID = "tenant-fixture", "project-fixture", current.BeadID
		transition.ExpectedVersion, transition.ExpectedIntegrity = current.Version, current.Integrity
		next, err := store.CompareAndSwapLifecycle(context.Background(), transition)
		if err != nil {
			t.Fatalf("native %s: %v", transition.Operation, err)
		}
		return next
	}
	post = applyLifecycle(post, gateway.LifecycleMutation{
		Operation: gateway.LifecycleHandoff, PrincipalProfileID: "work-authority-engineer", AttemptID: "execution-native-001",
		CanonicalClaimAttemptID: mutation.AttemptID, HandoffFenceDigest: strings.Repeat("e", 64), HeadSHA: head, EvidenceRefs: []string{"evidence-handoff"},
		NextProfileID: "qa", IdempotencyKey: "handoff-native",
	})
	post = applyLifecycle(post, gateway.LifecycleMutation{
		Operation: gateway.LifecycleReview, PrincipalProfileID: "qa", HeadSHA: head, Verdict: authorityv1.ReviewAccepted,
		EvidenceRefs: []string{"evidence-qa"}, IdempotencyKey: "review-qa-native",
	})
	post = applyLifecycle(post, gateway.LifecycleMutation{
		Operation: gateway.LifecycleReview, PrincipalProfileID: "security-reviewer", HeadSHA: head, Verdict: authorityv1.ReviewAccepted,
		EvidenceRefs: []string{"evidence-security"}, IdempotencyKey: "review-security-native",
	})
	post = applyLifecycle(post, gateway.LifecycleMutation{
		Operation: gateway.LifecycleRun, PrincipalProfileID: "delivery-orchestrator", HeadSHA: head, RunStatus: authorityv1.RunCompleted,
		EvidenceRefs: []string{"evidence-run"}, IdempotencyKey: "run-native",
	})
	post = applyLifecycle(post, gateway.LifecycleMutation{
		Operation: gateway.LifecycleReconcile, PrincipalProfileID: "delivery-orchestrator", HeadSHA: head,
		MergedSHA: strings.Repeat("c", 40), MergedTree: strings.Repeat("d", 40), PullRequestID: "pr-native", ProtectedMainRunID: "run-native",
		EvidenceRefs: []string{"evidence-merge"}, IdempotencyKey: "reconcile-native",
	})
	post = applyLifecycle(post, gateway.LifecycleMutation{
		Operation: gateway.LifecycleTerminal, PrincipalProfileID: "delivery-orchestrator", HeadSHA: head,
		EvidenceRefs: []string{"evidence-terminal"}, IdempotencyKey: "terminal-native",
	})
	if post.LifecycleState != authorityv1.LifecycleDone || post.NativeStatus != "closed" || post.ClaimAttemptID != mutation.AttemptID || post.Terminal == nil || post.Version.IssueMutationSequence != 8 {
		t.Fatalf("native lifecycle terminal=%#v", post)
	}
}

func boundedFixtureError(output []byte) string {
	const maximum = 256
	if len(output) > maximum {
		output = output[:maximum]
	}
	return fmt.Sprintf("%q", output)
}

func mustStore(t *testing.T, reader Reader, mutator AtomicMutator) *Store {
	t.Helper()
	store, err := New("tenant-fixture", "project-fixture", []authorityv1.Label{authorityv1.LabelPublicAccepted}, reader, mutator)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return store
}

func issueFixture(t *testing.T, lifecycle authorityv1.LifecycleState, attemptID, assignee string, mutationSequence uint64, unknownMetadata bool) []byte {
	t.Helper()
	status := "open"
	if lifecycle == authorityv1.LifecycleInProgress {
		status = "in_progress"
	}
	metadata := map[string]any{
		"schemaVersion": 1, "displayId": "W-002", "lifecycleState": lifecycle,
		"goalIds": []string{"G-001"}, "productDecisionIds": []string{"PD-002"}, "featureId": "F-002",
		"scenarioIds": []string{"F-002-S2"}, "exclusivePaths": []string{"internal/authority/**", "go.mod"},
		"verificationOrder": []string{"qa", "security-reviewer", "delivery-orchestrator"},
		"workVersion": map[string]any{
			"authorityGeneration": "generation-a", "issueIncarnation": "incarnation-a",
			"issueMutationSequence": mutationSequence, "dependencyGraphRevision": 1,
		},
		"risk": "high", "workType": "feature", "coordinator": "delivery-orchestrator",
		"failureOwnership": "deployed", "publicDisclosure": true,
	}
	if lifecycle == authorityv1.LifecycleInProgress {
		metadata["workClaim"] = map[string]any{
			"attemptId": attemptID, "idempotencyKey": "key-a",
			"baseCommit": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		}
	}
	if unknownMetadata {
		metadata["unreviewedAuthority"] = true
	}
	issue := map[string]any{
		"id": "M3-W002", "status": status, "assignee": assignee,
		"title": "synthetic title must not project", "description": "synthetic description must not project",
		"acceptance_criteria": "synthetic acceptance must not project", "comment_count": 0,
		"created_at": "2026-08-27T00:00:00Z", "created_by": "fixture@example.com", "dependency_count": 1,
		"dependent_count": 0, "issue_type": "task", "labels": []string{"public-first"}, "metadata": metadata,
		"owner": "", "priority": 1, "started_at": nil, "updated_at": "2026-08-27T00:00:00Z",
		"dependencies": []map[string]any{{
			"id": "M3-H001", "status": "closed", "dependency_type": "blocks",
			"metadata": map[string]any{"lifecycleState": "done", "reviewAccepted": true, "runDisposition": "completed", "reconciled": true},
		}},
	}
	data, err := json.Marshal([]any{issue})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func mutateFixtureMetadata(t *testing.T, raw []byte, mutate func(map[string]any)) []byte {
	t.Helper()
	var issues []map[string]any
	if err := json.Unmarshal(raw, &issues); err != nil || len(issues) != 1 {
		t.Fatalf("decode fixture: %v", err)
	}
	metadata, ok := issues[0]["metadata"].(map[string]any)
	if !ok {
		t.Fatal("fixture metadata missing")
	}
	mutate(metadata)
	encoded, err := json.Marshal(issues)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func TestProjectionRejectsCaseFoldedClaimAlias(t *testing.T) {
	raw := issueFixture(t, authorityv1.LifecycleInProgress, "canonical-attempt", "work-authority-engineer", 2, false)
	raw = mutateFixtureMetadata(t, raw, func(metadata map[string]any) {
		metadata["workclaim"] = map[string]any{
			"attemptId": "case-alias-attempt", "idempotencyKey": "case-alias-key",
			"baseCommit": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		}
	})
	if _, err := decodeIssueSnapshot(raw, "tenant-fixture", "project-fixture", []authorityv1.Label{authorityv1.LabelPublicAccepted}); !errors.Is(err, ErrProjectionInvalid) {
		t.Fatalf("case-folded claim alias error=%v", err)
	}
}

func TestProjectionRejectsLegacyTerminalScalarsWithoutDetailedRecords(t *testing.T) {
	raw := issueFixture(t, authorityv1.LifecycleInProgress, "canonical-attempt", "work-authority-engineer", 2, false)
	raw = mutateFixtureMetadata(t, raw, func(metadata map[string]any) {
		metadata["reviewAccepted"] = true
		metadata["runDisposition"] = "completed"
		metadata["reconciled"] = true
	})
	if _, err := decodeIssueSnapshot(raw, "tenant-fixture", "project-fixture", []authorityv1.Label{authorityv1.LabelPublicAccepted}); !errors.Is(err, ErrProjectionInvalid) {
		t.Fatalf("orphan legacy lifecycle scalars error=%v", err)
	}
}

func TestProjectionRejectsDependencyDetailedLifecycleContradiction(t *testing.T) {
	raw := issueFixture(t, authorityv1.LifecycleBacklog, "", "", 1, false)
	var issues []map[string]any
	if err := json.Unmarshal(raw, &issues); err != nil {
		t.Fatal(err)
	}
	dependency := issues[0]["dependencies"].([]any)[0].(map[string]any)
	metadata := dependency["metadata"].(map[string]any)
	metadata["runDispositionRecord"] = map[string]any{
		"principalProfileId": "delivery-orchestrator", "status": "failed",
		"headSHA": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "evidenceRefs": []any{"evidence-failed"},
		"idempotencyKey": "run-failed", "failure": map[string]any{
			"reason": "runtime-failed", "failure_fingerprint": "runtime-failed", "attempt": float64(1), "next_action": "retry-once",
		},
	}
	encoded, err := json.Marshal(issues)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeIssueSnapshot(encoded, "tenant-fixture", "project-fixture", []authorityv1.Label{authorityv1.LabelPublicAccepted}); !errors.Is(err, ErrProjectionInvalid) {
		t.Fatalf("dependency scalar/detail contradiction error=%v", err)
	}
}

func TestProjectionRejectsVersionedDependencyWithStrippedLifecycleEvidence(t *testing.T) {
	for _, fixture := range []struct {
		name       string
		emptyProof bool
	}{
		{name: "omitted-detailed-keys"},
		{name: "empty-and-null-detailed-keys", emptyProof: true},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			raw := issueFixture(t, authorityv1.LifecycleBacklog, "", "", 1, false)
			var issues []map[string]any
			if err := json.Unmarshal(raw, &issues); err != nil {
				t.Fatal(err)
			}
			dependency := issues[0]["dependencies"].([]any)[0].(map[string]any)
			metadata := dependency["metadata"].(map[string]any)
			metadata["workVersion"] = map[string]any{
				"authorityGeneration": "dependency-generation", "issueIncarnation": "dependency-incarnation",
				"issueMutationSequence": float64(8), "dependencyGraphRevision": float64(3),
			}
			metadata["workClaim"] = map[string]any{
				"attemptId": "dependency-attempt", "idempotencyKey": "dependency-claim",
				"baseCommit": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			}
			if fixture.emptyProof {
				metadata["reviewRecords"] = []any{}
				metadata["runDispositionRecord"] = nil
				metadata["terminalRecord"] = nil
			}
			encoded, err := json.Marshal(issues)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := decodeIssueSnapshot(encoded, "tenant-fixture", "project-fixture", []authorityv1.Label{authorityv1.LabelPublicAccepted}); !errors.Is(err, ErrProjectionInvalid) {
				t.Fatalf("versioned dependency with stripped lifecycle evidence error=%v", err)
			}
		})
	}
}
