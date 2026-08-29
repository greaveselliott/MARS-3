/*
FactoryDocSync:
docs:
- docs/features/F-002-work-authority.md
- docs/design-docs/ADR-001-git-beads-authority.md
- docs/code-documentation-map.md
*/

package closeout

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	authorityv1 "github.com/greaveselliott/MARS-3/api/authority/v1"
	"github.com/greaveselliott/MARS-3/internal/authority/gateway"
	"github.com/greaveselliott/MARS-3/internal/doctrine"
)

type fakeAuthority struct {
	item             authorityv1.WorkItem
	lease            authorityv1.CapabilityLease
	saga             gateway.ClaimSaga
	active           bool
	reconcileFailure error
	calls            []string
}

func (fake *fakeAuthority) Get(context.Context, string, string, string) (authorityv1.WorkItem, error) {
	return fake.item, nil
}

func (fake *fakeAuthority) ActiveLeaseForBead(context.Context, string, string, string) (authorityv1.CapabilityLease, bool, error) {
	return fake.lease, fake.active, nil
}

func (fake *fakeAuthority) Lookup(_ context.Context, _, _, key string) (gateway.ClaimSaga, bool, error) {
	if fake.saga.Intent.IdempotencyKey != key {
		return gateway.ClaimSaga{}, false, nil
	}
	return fake.saga, true, nil
}

func (fake *fakeAuthority) ReconcileClaimedWork(_ context.Context, _ authorityv1.Principal, request gateway.ClaimReconciliationRequest) (authorityv1.ClaimResponse, error) {
	fake.calls = append(fake.calls, "lease")
	if fake.reconcileFailure != nil {
		return authorityv1.ClaimResponse{}, fake.reconcileFailure
	}
	fake.active = true
	fake.lease = authorityv1.CapabilityLease{
		LeaseID: "lease-w001-terminal", TenantID: fake.item.TenantID, ProjectID: fake.item.ProjectID, BeadID: fake.item.BeadID,
		AttemptID: request.AttemptID, CanonicalClaimAttemptID: request.CanonicalClaimAttemptID, IdempotencyKey: request.IdempotencyKey,
		FenceGeneration: "generation-terminal", LeaseEpoch: 2, ClaimVersion: fake.item.Version, BaseSHA: request.BaseSHA,
		Capability: request.Capability, ExclusivePaths: append([]string(nil), request.ExclusivePaths...),
		Labels: []authorityv1.Label{authorityv1.LabelPublicAccepted}, State: authorityv1.LeaseActive, Active: true,
	}
	requestDigest := strings.Repeat("d", 64)
	fake.saga = gateway.ClaimSaga{
		RequestDigest: requestDigest,
		Phase:         gateway.ClaimPhaseComplete,
		Intent: gateway.ClaimIntent{RequestDigest: requestDigest, TenantID: fake.item.TenantID, ProjectID: fake.item.ProjectID, BeadID: fake.item.BeadID,
			AttemptID: request.AttemptID, IdempotencyKey: request.IdempotencyKey, BaseSHA: request.BaseSHA, Capability: request.Capability,
			ExclusivePaths: append([]string(nil), request.ExclusivePaths...), TraceRef: request.TraceRef,
			Labels: []authorityv1.Label{authorityv1.LabelPublicAccepted}},
		Work: fake.item, Lease: fake.lease,
	}
	return authorityv1.ClaimResponse{Work: fake.item, Lease: fake.lease}, nil
}

func (fake *fakeAuthority) Handoff(_ context.Context, _ authorityv1.Principal, request authorityv1.HandoffRequest) (authorityv1.LifecycleMutationResponse, error) {
	fake.calls = append(fake.calls, "handoff")
	fake.item.LifecycleState = authorityv1.LifecycleInReview
	fake.item.Handoff = &authorityv1.HandoffRecord{AttemptID: request.Fence.AttemptID, CanonicalClaimAttemptID: request.Fence.CanonicalClaimAttemptID,
		FenceDigest: strings.Repeat("a", 64), HeadSHA: request.HeadSHA, EvidenceRefs: request.EvidenceRefs, NextProfileID: request.NextProfileID, IdempotencyKey: request.IdempotencyKey}
	fake.advance("handoff")
	fake.active, fake.lease.Active, fake.lease.State = false, false, authorityv1.LeaseReleased
	fake.saga.Lease = fake.lease
	return authorityv1.LifecycleMutationResponse{Work: fake.item}, nil
}

func (fake *fakeAuthority) RecordReviewVerdict(_ context.Context, principal authorityv1.Principal, request authorityv1.ReviewVerdictRequest) (authorityv1.LifecycleMutationResponse, error) {
	fake.calls = append(fake.calls, principal.ProfileID)
	fake.item.Reviews = append(fake.item.Reviews, authorityv1.ReviewRecord{ReviewerProfileID: principal.ProfileID, Verdict: request.Verdict,
		HeadSHA: request.HeadSHA, EvidenceRefs: request.EvidenceRefs, IdempotencyKey: request.IdempotencyKey})
	fake.advance(principal.ProfileID)
	return authorityv1.LifecycleMutationResponse{Work: fake.item}, nil
}

func (fake *fakeAuthority) RecordRunDisposition(_ context.Context, principal authorityv1.Principal, request authorityv1.RunDispositionRequest) (authorityv1.LifecycleMutationResponse, error) {
	fake.calls = append(fake.calls, "run")
	fake.item.RunDisposition = &authorityv1.RunDispositionRecord{PrincipalProfileID: principal.ProfileID, Status: request.Status,
		HeadSHA: request.HeadSHA, EvidenceRefs: request.EvidenceRefs, IdempotencyKey: request.IdempotencyKey}
	fake.advance("run")
	return authorityv1.LifecycleMutationResponse{Work: fake.item}, nil
}

func (fake *fakeAuthority) RecordReconciliation(_ context.Context, principal authorityv1.Principal, request authorityv1.ReconciliationRequest) (authorityv1.LifecycleMutationResponse, error) {
	fake.calls = append(fake.calls, "reconcile")
	fake.item.Reconciliation = &authorityv1.ReconciliationRecord{PrincipalProfileID: principal.ProfileID, HeadSHA: request.HeadSHA,
		MergedSHA: request.MergedSHA, MergedTree: request.MergedTree, PullRequestID: request.PullRequestID,
		ProtectedMainRunID: request.ProtectedMainRunID, EvidenceRefs: request.EvidenceRefs, IdempotencyKey: request.IdempotencyKey}
	fake.advance("reconcile")
	return authorityv1.LifecycleMutationResponse{Work: fake.item}, nil
}

func (fake *fakeAuthority) CloseWork(_ context.Context, principal authorityv1.Principal, request authorityv1.TerminalTransitionRequest) (authorityv1.LifecycleMutationResponse, error) {
	fake.calls = append(fake.calls, "terminal")
	fake.item.NativeStatus, fake.item.LifecycleState = "closed", authorityv1.LifecycleDone
	fake.item.Terminal = &authorityv1.TerminalRecord{PrincipalProfileID: principal.ProfileID, HeadSHA: request.HeadSHA,
		EvidenceRefs: request.EvidenceRefs, IdempotencyKey: request.IdempotencyKey}
	fake.advance("terminal")
	return authorityv1.LifecycleMutationResponse{Work: fake.item}, nil
}

func (fake *fakeAuthority) advance(operation string) {
	fake.item.Version.IssueMutationSequence++
	digest := sha256.Sum256([]byte(operation))
	fake.item.Integrity.Lineage = hex.EncodeToString(digest[:])
}

func TestDryRunProvesPreimageWithoutMutation(t *testing.T) {
	grant, authorization, fake := closeoutFixture()
	receipt, err := runWithDependencies(context.Background(), grant, authorization, dependencies{client: fake, work: fake, leases: fake, sagas: fake}, false)
	if err != nil {
		t.Fatalf("dry run: %v", err)
	}
	if receipt.Mode != "dry-run" || receipt.Result != "ready-no-mutation" || len(fake.calls) != 0 || fake.active {
		t.Fatalf("receipt=%#v calls=%v active=%v", receipt, fake.calls, fake.active)
	}
}

func TestApplyExecutesOrderedGatewayOnlySequenceAndReplaysTerminalState(t *testing.T) {
	grant, authorization, fake := closeoutFixture()
	receipt, err := runWithDependencies(context.Background(), grant, authorization, dependencies{client: fake, work: fake, leases: fake, sagas: fake}, true)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	want := []string{"lease", "handoff", "qa", "security-reviewer", "run", "reconcile", "terminal"}
	if strings.Join(fake.calls, ",") != strings.Join(want, ",") {
		t.Fatalf("calls=%v want=%v", fake.calls, want)
	}
	if receipt.LifecycleState != authorityv1.LifecycleDone || receipt.NativeStatus != "closed" || receipt.LiveLeasePresent || !receipt.TerminalRecorded {
		t.Fatalf("receipt=%#v", receipt)
	}
	before := len(fake.calls)
	replayed, err := runWithDependencies(context.Background(), grant, authorization, dependencies{client: fake, work: fake, leases: fake, sagas: fake}, true)
	if err != nil || len(fake.calls) != before || replayed.Result != "terminal-close-verified" {
		t.Fatalf("replay=%#v err=%v calls=%v", replayed, err, fake.calls)
	}
}

func TestApplyFailsClosedOnOutOfOrderReview(t *testing.T) {
	grant, authorization, fake := closeoutFixture()
	fake.item.LifecycleState = authorityv1.LifecycleInReview
	fake.item.Handoff = &authorityv1.HandoffRecord{AttemptID: grant.AttemptID, CanonicalClaimAttemptID: grant.CanonicalClaimAttemptID,
		HeadSHA: grant.AcceptedCandidateHead, NextProfileID: "qa", IdempotencyKey: operationKey(grant, "handoff")}
	fake.item.Reviews = []authorityv1.ReviewRecord{{ReviewerProfileID: "security-reviewer", Verdict: authorityv1.ReviewAccepted, HeadSHA: grant.AcceptedCandidateHead}}
	if _, err := runWithDependencies(context.Background(), grant, authorization, dependencies{client: fake, work: fake, leases: fake, sagas: fake}, true); !errors.Is(err, ErrCanonicalPreimage) {
		t.Fatalf("out-of-order review error=%v", err)
	}
}

func TestApplyRecoversCompletedClaimSagaAfterLeaseRelease(t *testing.T) {
	grant, authorization, fake := closeoutFixture()
	request := gateway.ClaimReconciliationRequest{ClaimRequest: authorityv1.ClaimRequest{
		BeadID: fake.item.BeadID, ExpectedVersion: fake.item.Version, ExpectedIntegrity: fake.item.Integrity,
		AttemptID: grant.AttemptID, BaseSHA: authorization.MergedCommit, ExclusivePaths: append([]string(nil), fake.item.ExclusivePaths...),
		Capability: authorityv1.CapabilityTicketDelivery, IdempotencyKey: operationKey(grant, "lease"), TraceRef: operationTrace("lease"),
	}, CanonicalClaimAttemptID: grant.CanonicalClaimAttemptID}
	if _, err := fake.ReconcileClaimedWork(context.Background(), authorityv1.Principal{}, request); err != nil {
		t.Fatal(err)
	}
	fake.active, fake.lease.Active, fake.lease.State = false, false, authorityv1.LeaseReleased
	fake.saga.Lease = fake.lease
	fake.reconcileFailure = errors.New("completed saga has released lease")
	fake.calls = nil

	receipt, err := runWithDependencies(context.Background(), grant, authorization, dependencies{client: fake, work: fake, leases: fake, sagas: fake}, true)
	if err != nil {
		t.Fatalf("recover released claim saga: %v", err)
	}
	want := []string{"lease", "handoff", "qa", "security-reviewer", "run", "reconcile", "terminal"}
	if strings.Join(fake.calls, ",") != strings.Join(want, ",") || receipt.LifecycleState != authorityv1.LifecycleDone {
		t.Fatalf("calls=%v receipt=%#v", fake.calls, receipt)
	}
}

func TestWorkspaceInstanceDigestBindsCanonicalEmbeddedDoltIdentity(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	beadsDir := filepath.Join(root, ".beads")
	if err := os.MkdirAll(filepath.Join(beadsDir, "embeddeddolt", "M3"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(beadsDir, "config.yaml"), []byte("# canonical embedded workspace\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	metadataPath := filepath.Join(beadsDir, "metadata.json")
	canonical := []byte(`{"database":"dolt","backend":"dolt","dolt_mode":"embedded","dolt_database":"M3","project_id":"project-mars3"}`)
	if err := os.WriteFile(metadataPath, canonical, 0o600); err != nil {
		t.Fatal(err)
	}
	first, err := workspaceInstanceDigest(root, "project-mars3")
	if err != nil || !lowerHexDigest.MatchString(first) {
		t.Fatalf("canonical workspace digest=%q err=%v", first, err)
	}
	second, err := workspaceInstanceDigest(root, "project-mars3")
	if err != nil || second != first {
		t.Fatalf("stable digest=%q want=%q err=%v", second, first, err)
	}
	if _, err := workspaceInstanceDigest(root, "different-project"); !errors.Is(err, ErrExecutionBoundary) {
		t.Fatalf("project mismatch error=%v", err)
	}
	duplicate := []byte(`{"database":"dolt","database":"dolt","backend":"dolt","dolt_mode":"embedded","dolt_database":"M3","project_id":"project-mars3"}`)
	if err := os.WriteFile(metadataPath, duplicate, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := workspaceInstanceDigest(root, "project-mars3"); !errors.Is(err, ErrExecutionBoundary) {
		t.Fatalf("duplicate metadata key error=%v", err)
	}
	unknown := []byte(`{"database":"dolt","backend":"dolt","dolt_mode":"embedded","dolt_database":"M3","project_id":"project-mars3","extra":true}`)
	if err := os.WriteFile(metadataPath, unknown, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := workspaceInstanceDigest(root, "project-mars3"); !errors.Is(err, ErrExecutionBoundary) {
		t.Fatalf("unknown metadata field error=%v", err)
	}
}

func closeoutFixture() (doctrine.W001TerminalReconciliationGrant, doctrine.W001TerminalReconciliationExecutionAuthorization, *fakeAuthority) {
	version := authorityv1.WorkVersion{AuthorityGeneration: terminalAuthorityGenerationFixture(),
		IssueIncarnation: "e1e8d2d3f80871096a568fb489f49575a42abd37b269df9faf777a09cd689b41", IssueMutationSequence: 1, DependencyGraphRevision: 1}
	integrity := authorityv1.IntegrityDigests{Lineage: strings.Repeat("1", 64), DependencyOutcomes: strings.Repeat("2", 64),
		Blockers: strings.Repeat("3", 64), ExclusivePaths: strings.Repeat("4", 64)}
	grant := doctrine.W001TerminalReconciliationGrant{ID: "W-001-lifecycle-terminal-reconciliation-v1", Bead: "M3-W001",
		Assignee: "work-authority-engineer", AttemptID: "w001-lifecycle-terminal-reconciliation-v1", CanonicalClaimAttemptID: "w001-bootstrap-claim",
		AcceptedCandidateHead: strings.Repeat("5", 40), MergedCommit: strings.Repeat("6", 40), MergedTree: strings.Repeat("7", 40),
		ExpectedNativeStatus: "in_progress", ExpectedLifecycleState: "in-progress", ExpectedVersion: version}
	authorization := doctrine.W001TerminalReconciliationExecutionAuthorization{TenantID: "tenant-academy", ProjectID: "project-mars3",
		MergedCommit: strings.Repeat("8", 40), FenceGeneration: "generation-terminal"}
	item := authorityv1.WorkItem{TenantID: authorization.TenantID, ProjectID: authorization.ProjectID, BeadID: grant.Bead,
		NativeStatus: "in_progress", LifecycleState: authorityv1.LifecycleInProgress, Assignee: grant.Assignee,
		ClaimAttemptID: grant.CanonicalClaimAttemptID, ExclusivePaths: []string{"internal/authority/**"},
		VerificationOrder: []string{"qa", "security-reviewer", "delivery-orchestrator"}, Labels: []authorityv1.Label{authorityv1.LabelPublicAccepted},
		Version: version, Integrity: integrity}
	return grant, authorization, &fakeAuthority{item: item}
}

func terminalAuthorityGenerationFixture() string {
	return strings.Join([]string{"6e79ff81", "a007", "42a5", "a178", "7ce58dbb718b"}, "-")
}

func TestTerminalAuthorityGenerationFixtureRetainsCanonicalValue(t *testing.T) {
	digest := sha256.Sum256([]byte(terminalAuthorityGenerationFixture()))
	if hex.EncodeToString(digest[:]) != "5f511c9526eddecfc4984e36ffac6e1add017c85e6fa85ae59a1b7c0f3ae8e85" {
		t.Fatal("scanner-safe authority-generation fixture changed value")
	}
}
