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
	"io"
	"os"
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
		AttemptID: "attempt-001", Assignee: "work-authority-engineer", IdempotencyKey: "idempotency-001",
	})
	if err != nil {
		t.Fatalf("CompareAndSwapClaim: %v", err)
	}
	if post.LifecycleState != authorityv1.LifecycleInProgress || post.ClaimAttemptID != "attempt-001" || post.Version.IssueMutationSequence != 2 || mutator.calls != 1 || mutator.input.ExpectedDigest == "" || mutator.input.ExpectedVersion != pre.Version || mutator.input.ExpectedIntegrity != pre.Integrity {
		t.Fatalf("post=%#v input=%#v calls=%d", post, mutator.input, mutator.calls)
	}
	stale := pre.Version
	stale.IssueMutationSequence++
	if _, err := store.CompareAndSwapClaim(context.Background(), gateway.ClaimMutation{
		TenantID: "tenant-fixture", ProjectID: "project-fixture", BeadID: pre.BeadID,
		ExpectedVersion: stale, ExpectedIntegrity: pre.Integrity,
		AttemptID: "attempt-002", Assignee: "work-authority-engineer", IdempotencyKey: "idempotency-002",
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
		AttemptID: "attempt-001", Assignee: "work-authority-engineer", IdempotencyKey: "idempotency-001",
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
			"authorityGeneration": "authority-generation-001", "issueIncarnation": "issue-incarnation-001",
			"issueMutationSequence": mutationSequence, "dependencyGraphRevision": 1,
		},
		"risk": "high", "workType": "feature", "coordinator": "delivery-orchestrator",
		"failureOwnership": "deployed", "publicDisclosure": true,
	}
	if lifecycle == authorityv1.LifecycleInProgress {
		metadata["workClaim"] = map[string]any{
			"attemptId": attemptID, "idempotencyKey": "idempotency-001",
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
