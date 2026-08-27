/*
FactoryDocSync:
docs:
- docs/features/F-002-work-authority.md
- docs/product-specs/work-authority.md
- docs/design-docs/ADR-001-git-beads-authority.md
- docs/code-documentation-map.md
*/

package gateway

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	authorityv1 "github.com/greaveselliott/MARS-3/api/authority/v1"
)

type openLifecycleLocks struct{}

func cloneRunDisposition(value *authorityv1.RunDispositionRecord) *authorityv1.RunDispositionRecord {
	if value == nil {
		return nil
	}
	clone := *value
	clone.EvidenceRefs = append([]string(nil), value.EvidenceRefs...)
	clone.Failure = normalizedFailureContext(value.Failure)
	return &clone
}

func (openLifecycleLocks) Enter(context.Context, string, string) (func(), error) {
	return func() {}, nil
}

func (openLifecycleLocks) EnterWork(context.Context, string, string, string) (func(), error) {
	return func() {}, nil
}

func (store *memoryClaimStore) CompareAndSwapLifecycle(_ context.Context, mutation LifecycleMutation) (authorityv1.WorkItem, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.lifecycleCalls++
	if store.lifecycleErr != nil && !store.lifecycleApplyThenErr {
		return authorityv1.WorkItem{}, store.lifecycleErr
	}
	if store.item.TenantID != mutation.TenantID || store.item.ProjectID != mutation.ProjectID || store.item.BeadID != mutation.BeadID ||
		store.item.Version != mutation.ExpectedVersion || store.item.Integrity != mutation.ExpectedIntegrity {
		return authorityv1.WorkItem{}, ErrStaleWorkVersion
	}
	item := cloneWork(store.item)
	switch mutation.Operation {
	case LifecycleHandoff:
		if item.Handoff != nil {
			item.ReviewHistory = append(item.ReviewHistory, authorityv1.ReviewCycle{Handoff: *item.Handoff, Reviews: append([]authorityv1.ReviewRecord(nil), item.Reviews...),
				RunHistory: append([]authorityv1.RunDispositionRecord(nil), item.RunHistory...), RunDisposition: cloneRunDisposition(item.RunDisposition)})
		}
		item.LifecycleState = authorityv1.LifecycleInReview
		item.Handoff = &authorityv1.HandoffRecord{
			AttemptID: mutation.AttemptID, CanonicalClaimAttemptID: mutation.CanonicalClaimAttemptID, FenceDigest: mutation.HandoffFenceDigest,
			HeadSHA: mutation.HeadSHA, EvidenceRefs: append([]string(nil), mutation.EvidenceRefs...),
			NextProfileID: mutation.NextProfileID, IdempotencyKey: mutation.IdempotencyKey,
		}
		item.Reviews, item.RunHistory, item.RunDisposition, item.Reconciliation, item.Terminal, item.Blockers = nil, nil, nil, nil, nil, nil
	case LifecycleReview:
		item.Reviews = append(item.Reviews, authorityv1.ReviewRecord{
			ReviewerProfileID: mutation.PrincipalProfileID, Verdict: mutation.Verdict, HeadSHA: mutation.HeadSHA,
			EvidenceRefs: append([]string(nil), mutation.EvidenceRefs...), IdempotencyKey: mutation.IdempotencyKey, Failure: normalizedFailureContext(mutation.Failure),
		})
		if mutation.Verdict == authorityv1.ReviewChangesRequested || mutation.Verdict == authorityv1.ReviewBlocked {
			item.LifecycleState = authorityv1.LifecycleInProgress
			if mutation.Verdict == authorityv1.ReviewBlocked {
				item.Blockers = append(append([]string(nil), mutation.Failure.BlockedBy...), mutation.Failure.Reason)
			}
		}
	case LifecycleRun:
		if item.RunDisposition != nil {
			item.RunHistory = append(item.RunHistory, *cloneRunDisposition(item.RunDisposition))
		}
		item.RunDisposition = &authorityv1.RunDispositionRecord{
			PrincipalProfileID: mutation.PrincipalProfileID, Status: mutation.RunStatus, HeadSHA: mutation.HeadSHA,
			EvidenceRefs: append([]string(nil), mutation.EvidenceRefs...), IdempotencyKey: mutation.IdempotencyKey, Failure: normalizedFailureContext(mutation.Failure),
		}
		item.Blockers = nil
		if mutation.RunStatus == authorityv1.RunBlocked {
			item.Blockers = append(append([]string(nil), mutation.Failure.BlockedBy...), mutation.Failure.Reason)
		}
		if mutation.RunStatus != authorityv1.RunCompleted && mutation.RunStatus != authorityv1.RunInReview {
			item.LifecycleState = authorityv1.LifecycleInProgress
		}
	case LifecycleReconcile:
		item.Reconciliation = &authorityv1.ReconciliationRecord{
			PrincipalProfileID: mutation.PrincipalProfileID, HeadSHA: mutation.HeadSHA, MergedSHA: mutation.MergedSHA,
			MergedTree: mutation.MergedTree, PullRequestID: mutation.PullRequestID, ProtectedMainRunID: mutation.ProtectedMainRunID,
			EvidenceRefs: append([]string(nil), mutation.EvidenceRefs...), IdempotencyKey: mutation.IdempotencyKey,
		}
	case LifecycleTerminal:
		item.LifecycleState, item.NativeStatus = authorityv1.LifecycleDone, "closed"
		item.Terminal = &authorityv1.TerminalRecord{PrincipalProfileID: mutation.PrincipalProfileID, HeadSHA: mutation.HeadSHA,
			EvidenceRefs: append([]string(nil), mutation.EvidenceRefs...), IdempotencyKey: mutation.IdempotencyKey}
	default:
		return authorityv1.WorkItem{}, ErrStaleWorkVersion
	}
	item.Version.IssueMutationSequence++
	digest := sha256.Sum256([]byte(mutation.IdempotencyKey + ":" + string(mutation.Operation)))
	item.Integrity.Lineage = hex.EncodeToString(digest[:])
	store.item = cloneWork(item)
	if store.lifecycleErr != nil {
		return authorityv1.WorkItem{}, store.lifecycleErr
	}
	return item, nil
}

func TestLifecycleCompletesOnlyAfterOrderedIndependentEvidence(t *testing.T) {
	clock := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	service, store, sagas, events, engineer, lease := lifecycleFixture(t, &clock)
	service.barrier, service.workLocks = openLifecycleLocks{}, openLifecycleLocks{}
	engineer.Capabilities = append(engineer.Capabilities, authorityv1.CapabilityWorkHandoff, authorityv1.CapabilityLeaseRelease)
	head := strings.Repeat("c", 40)

	current := lifecycleWork(t, store)
	handoffRequest := authorityv1.HandoffRequest{
		BeadID: current.BeadID, ExpectedVersion: current.Version, ExpectedIntegrity: current.Integrity, Fence: fenceFromLease(lease),
		HeadSHA: head, EvidenceRefs: []string{"evidence-engineer"}, NextProfileID: "qa", IdempotencyKey: "handoff-001", TraceRef: "trace-handoff-001",
	}
	handoff, err := service.Handoff(context.Background(), engineer, handoffRequest)
	if err != nil || handoff.Replayed || handoff.Work.LifecycleState != authorityv1.LifecycleInReview {
		t.Fatalf("handoff=%#v err=%v", handoff, err)
	}
	if stored, err := sagas.GetLease(context.Background(), lease.TenantID, lease.ProjectID, lease.LeaseID); err != nil || stored.Active || stored.State != authorityv1.LeaseReleased {
		t.Fatalf("handoff lease=%#v err=%v", stored, err)
	}
	if replay, err := service.Handoff(context.Background(), engineer, handoffRequest); err != nil || !replay.Replayed || !strings.HasPrefix(replay.ReceiptRef, "lifecycle-reconciliation-") {
		t.Fatalf("handoff replay=%#v err=%v", replay, err)
	}
	orchestrator := lifecyclePrincipal("delivery-orchestrator", authorityv1.CapabilityRunDisposition, authorityv1.CapabilityWorkReconcile, authorityv1.CapabilityWorkClose)
	if _, err := service.CloseWork(context.Background(), orchestrator, authorityv1.TerminalTransitionRequest{
		BeadID: handoff.Work.BeadID, ExpectedVersion: handoff.Work.Version, ExpectedIntegrity: handoff.Work.Integrity,
		HeadSHA: head, EvidenceRefs: []string{"evidence-premature"}, IdempotencyKey: "terminal-premature", TraceRef: "trace-terminal-premature",
	}); err == nil {
		t.Fatal("terminal transition passed before reviews, run, and reconciliation")
	}

	qa := lifecyclePrincipal("qa", authorityv1.CapabilityReviewRecord)
	qaResult := recordReview(t, service, qa, lifecycleWork(t, store), head, authorityv1.ReviewAccepted, "qa-review-001")
	security := lifecyclePrincipal("security-reviewer", authorityv1.CapabilityReviewRecord)
	securityResult := recordReview(t, service, security, qaResult.Work, head, authorityv1.ReviewAccepted, "security-review-001")

	run, err := service.RecordRunDisposition(context.Background(), orchestrator, authorityv1.RunDispositionRequest{
		BeadID: securityResult.Work.BeadID, ExpectedVersion: securityResult.Work.Version, ExpectedIntegrity: securityResult.Work.Integrity,
		HeadSHA: head, Status: authorityv1.RunCompleted, EvidenceRefs: []string{"evidence-run"}, IdempotencyKey: "run-001", TraceRef: "trace-run-001",
	})
	if err != nil || run.Work.RunDisposition == nil || run.Work.RunDisposition.Status != authorityv1.RunCompleted {
		t.Fatalf("run=%#v err=%v", run, err)
	}
	reconciled, err := service.RecordReconciliation(context.Background(), orchestrator, authorityv1.ReconciliationRequest{
		BeadID: run.Work.BeadID, ExpectedVersion: run.Work.Version, ExpectedIntegrity: run.Work.Integrity, HeadSHA: head,
		MergedSHA: strings.Repeat("d", 40), MergedTree: strings.Repeat("e", 40), PullRequestID: "pr-009", ProtectedMainRunID: "run-009",
		EvidenceRefs: []string{"evidence-merge"}, IdempotencyKey: "reconcile-001", TraceRef: "trace-reconcile-001",
	})
	if err != nil || reconciled.Work.Reconciliation == nil {
		t.Fatalf("reconciliation=%#v err=%v", reconciled, err)
	}
	closed, err := service.CloseWork(context.Background(), orchestrator, authorityv1.TerminalTransitionRequest{
		BeadID: reconciled.Work.BeadID, ExpectedVersion: reconciled.Work.Version, ExpectedIntegrity: reconciled.Work.Integrity,
		HeadSHA: head, EvidenceRefs: []string{"evidence-terminal"}, IdempotencyKey: "terminal-001", TraceRef: "trace-terminal-001",
	})
	if err != nil || closed.Work.LifecycleState != authorityv1.LifecycleDone || closed.Work.NativeStatus != "closed" || closed.Work.Terminal == nil {
		t.Fatalf("closed=%#v err=%v", closed, err)
	}
	if store.lifecycleCalls != 6 {
		t.Fatalf("lifecycle calls=%d, want 6", store.lifecycleCalls)
	}
	if countLifecycleReceipts(events.events) != 6 {
		t.Fatalf("lifecycle receipts=%d, want 6", countLifecycleReceipts(events.events))
	}
}

func TestChangesRequestedReopensSameBeadWithNewerLeaseEpoch(t *testing.T) {
	clock := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	service, store, _, _, engineer, firstLease := lifecycleFixture(t, &clock)
	service.barrier, service.workLocks = openLifecycleLocks{}, openLifecycleLocks{}
	engineer.Capabilities = append(engineer.Capabilities, authorityv1.CapabilityWorkHandoff, authorityv1.CapabilityLeaseRelease)
	firstHead := strings.Repeat("c", 40)
	first := lifecycleWork(t, store)
	if _, err := service.Handoff(context.Background(), engineer, authorityv1.HandoffRequest{
		BeadID: first.BeadID, ExpectedVersion: first.Version, ExpectedIntegrity: first.Integrity, Fence: fenceFromLease(firstLease), HeadSHA: firstHead,
		EvidenceRefs: []string{"evidence-first"}, NextProfileID: "qa", IdempotencyKey: "handoff-first", TraceRef: "trace-handoff-first",
	}); err != nil {
		t.Fatalf("first handoff: %v", err)
	}
	reopened := recordReview(t, service, lifecyclePrincipal("qa", authorityv1.CapabilityReviewRecord), lifecycleWork(t, store), firstHead, authorityv1.ReviewChangesRequested, "qa-changes-001")
	if reopened.Work.BeadID != first.BeadID || reopened.Work.LifecycleState != authorityv1.LifecycleInProgress {
		t.Fatalf("reopened=%#v", reopened.Work)
	}

	reworkRequest := ClaimReconciliationRequest{
		ClaimRequest: authorityv1.ClaimRequest{
			BeadID: reopened.Work.BeadID, ExpectedVersion: reopened.Work.Version, ExpectedIntegrity: reopened.Work.Integrity,
			AttemptID: "w001-rework-attempt", BaseSHA: strings.Repeat("f", 40), ExclusivePaths: append([]string(nil), reopened.Work.ExclusivePaths...),
			Capability: authorityv1.CapabilityTicketDelivery, IdempotencyKey: "w001-rework-lease", TraceRef: "trace-rework-lease",
		},
		CanonicalClaimAttemptID: reopened.Work.ClaimAttemptID,
	}
	rework, err := service.ReconcileClaimedWork(context.Background(), engineer, reworkRequest)
	if err != nil || !rework.Lease.Active || rework.Lease.LeaseEpoch <= firstLease.LeaseEpoch || rework.Work.BeadID != first.BeadID {
		t.Fatalf("rework=%#v err=%v firstEpoch=%d", rework, err, firstLease.LeaseEpoch)
	}
	secondHead := strings.Repeat("a", 40)
	secondHandoff, err := service.Handoff(context.Background(), engineer, authorityv1.HandoffRequest{
		BeadID: rework.Work.BeadID, ExpectedVersion: rework.Work.Version, ExpectedIntegrity: rework.Work.Integrity, Fence: fenceFromLease(rework.Lease),
		HeadSHA: secondHead, EvidenceRefs: []string{"evidence-second"}, NextProfileID: "qa", IdempotencyKey: "handoff-second", TraceRef: "trace-handoff-second",
	})
	if err != nil || secondHandoff.Work.LifecycleState != authorityv1.LifecycleInReview || len(secondHandoff.Work.Reviews) != 0 || secondHandoff.Work.Handoff.HeadSHA != secondHead ||
		len(secondHandoff.Work.ReviewHistory) != 1 || secondHandoff.Work.ReviewHistory[0].Reviews[0].Verdict != authorityv1.ReviewChangesRequested {
		t.Fatalf("second handoff=%#v err=%v", secondHandoff, err)
	}
	if replay, err := service.Handoff(context.Background(), engineer, authorityv1.HandoffRequest{
		BeadID: first.BeadID, ExpectedVersion: first.Version, ExpectedIntegrity: first.Integrity, Fence: fenceFromLease(firstLease), HeadSHA: firstHead,
		EvidenceRefs: []string{"evidence-first"}, NextProfileID: "qa", IdempotencyKey: "handoff-first", TraceRef: "trace-handoff-first",
	}); err != nil || !replay.Replayed {
		t.Fatalf("historical handoff replay=%#v err=%v", replay, err)
	}
	historicalSplice := authorityv1.HandoffRequest{
		BeadID: first.BeadID, ExpectedVersion: first.Version, ExpectedIntegrity: first.Integrity, Fence: fenceFromLease(firstLease), HeadSHA: firstHead,
		EvidenceRefs: []string{"evidence-first"}, NextProfileID: "qa", IdempotencyKey: "handoff-first", TraceRef: "trace-handoff-first",
	}
	historicalSplice.Fence.CanonicalClaimAttemptID = "spliced-historical-claim"
	_, err = service.Handoff(context.Background(), engineer, historicalSplice)
	assertDenial(t, err, authorityv1.ErrorPolicyDenied, ruleLifecycleIdempotency, "use the original normalized lifecycle request or a new idempotency key", "work.read")
	conflict := authorityv1.HandoffRequest{
		BeadID: secondHandoff.Work.BeadID, ExpectedVersion: secondHandoff.Work.Version, ExpectedIntegrity: secondHandoff.Work.Integrity,
		Fence: fenceFromLease(rework.Lease), HeadSHA: strings.Repeat("b", 40), EvidenceRefs: []string{"evidence-conflict"},
		NextProfileID: "qa", IdempotencyKey: "handoff-first", TraceRef: "trace-handoff-conflict",
	}
	_, err = service.Handoff(context.Background(), engineer, conflict)
	assertDenial(t, err, authorityv1.ErrorPolicyDenied, ruleLifecycleIdempotency, "use the original normalized lifecycle request or a new idempotency key", "work.read")
}

func TestLifecycleRejectsWrongReviewerAndAnyActiveLease(t *testing.T) {
	clock := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	service, store, sagas, _, engineer, lease := lifecycleFixture(t, &clock)
	service.barrier, service.workLocks = openLifecycleLocks{}, openLifecycleLocks{}
	engineer.Capabilities = append(engineer.Capabilities, authorityv1.CapabilityWorkHandoff, authorityv1.CapabilityLeaseRelease)
	head := strings.Repeat("c", 40)
	current := lifecycleWork(t, store)
	if _, err := service.Handoff(context.Background(), engineer, authorityv1.HandoffRequest{
		BeadID: current.BeadID, ExpectedVersion: current.Version, ExpectedIntegrity: current.Integrity, Fence: fenceFromLease(lease), HeadSHA: head,
		EvidenceRefs: []string{"evidence-engineer"}, NextProfileID: "qa", IdempotencyKey: "handoff-001", TraceRef: "trace-handoff-001",
	}); err != nil {
		t.Fatalf("handoff: %v", err)
	}
	item := lifecycleWork(t, store)
	_, err := service.RecordReviewVerdict(context.Background(), lifecyclePrincipal("security-reviewer", authorityv1.CapabilityReviewRecord), reviewRequest(item, head, authorityv1.ReviewAccepted, "wrong-reviewer"))
	assertDenial(t, err, authorityv1.ErrorPolicyDenied, ruleLifecycleOrder, requiredLifecycleRead, "review.record(qa)")

	sagas.mu.Lock()
	for key, saga := range sagas.sagas {
		if saga.Lease.LeaseID == lease.LeaseID {
			saga.Lease.Active, saga.Lease.State = true, authorityv1.LeaseActive
			sagas.sagas[key] = saga
		}
	}
	sagas.mu.Unlock()
	_, err = service.RecordReviewVerdict(context.Background(), lifecyclePrincipal("qa", authorityv1.CapabilityReviewRecord), reviewRequest(item, head, authorityv1.ReviewAccepted, "qa-active-lease"))
	assertDenial(t, err, authorityv1.ErrorPolicyDenied, ruleLifecycleLease, "release or revoke the implementation lease before independent review", "lease.release")
}

func TestHandoffUnknownEffectAfterLeaseReleaseRetriesSafely(t *testing.T) {
	clock := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	service, store, sagas, _, engineer, lease := lifecycleFixture(t, &clock)
	service.barrier, service.workLocks = openLifecycleLocks{}, openLifecycleLocks{}
	engineer.Capabilities = append(engineer.Capabilities, authorityv1.CapabilityWorkHandoff, authorityv1.CapabilityLeaseRelease)
	current := lifecycleWork(t, store)
	request := authorityv1.HandoffRequest{
		BeadID: current.BeadID, ExpectedVersion: current.Version, ExpectedIntegrity: current.Integrity, Fence: fenceFromLease(lease),
		HeadSHA: strings.Repeat("c", 40), EvidenceRefs: []string{"evidence-unknown"}, NextProfileID: "qa",
		IdempotencyKey: "handoff-unknown", TraceRef: "trace-handoff-unknown",
	}
	store.lifecycleErr = errors.New("synthetic unknown effect")
	_, err := service.Handoff(context.Background(), engineer, request)
	assertDenial(t, err, authorityv1.ErrorUnknownEffect, ruleLifecycleUnknown, requiredLifecycleReconcile, "work.reconcile")
	if stored, getErr := sagas.GetLease(context.Background(), lease.TenantID, lease.ProjectID, lease.LeaseID); getErr != nil || stored.Active || stored.State != authorityv1.LeaseReleased {
		t.Fatalf("released lease after unknown effect=%#v err=%v", stored, getErr)
	}
	if got := lifecycleWork(t, store); got.LifecycleState != authorityv1.LifecycleInProgress || got.Handoff != nil {
		t.Fatalf("unknown pre-effect changed work=%#v", got)
	}
	store.lifecycleErr = nil
	retried, err := service.Handoff(context.Background(), engineer, request)
	if err != nil || retried.Replayed || retried.Work.LifecycleState != authorityv1.LifecycleInReview {
		t.Fatalf("handoff retry=%#v err=%v", retried, err)
	}
}

func TestHandoffAppliedUnknownEffectUsesCanonicalReadback(t *testing.T) {
	clock := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	service, store, _, _, engineer, lease := lifecycleFixture(t, &clock)
	service.barrier, service.workLocks = openLifecycleLocks{}, openLifecycleLocks{}
	engineer.Capabilities = append(engineer.Capabilities, authorityv1.CapabilityWorkHandoff, authorityv1.CapabilityLeaseRelease)
	current := lifecycleWork(t, store)
	store.lifecycleErr = errors.New("synthetic lost receipt")
	store.lifecycleApplyThenErr = true
	result, err := service.Handoff(context.Background(), engineer, authorityv1.HandoffRequest{
		BeadID: current.BeadID, ExpectedVersion: current.Version, ExpectedIntegrity: current.Integrity, Fence: fenceFromLease(lease),
		HeadSHA: strings.Repeat("c", 40), EvidenceRefs: []string{"evidence-readback"}, NextProfileID: "qa",
		IdempotencyKey: "handoff-readback", TraceRef: "trace-handoff-readback",
	})
	if err != nil || result.Work.LifecycleState != authorityv1.LifecycleInReview || result.Work.Handoff == nil {
		t.Fatalf("handoff readback=%#v err=%v", result, err)
	}
}

func TestHandoffReplayRejectsSplicedCanonicalClaimAndFence(t *testing.T) {
	clock := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	service, store, _, _, engineer, lease := lifecycleFixture(t, &clock)
	service.barrier, service.workLocks = openLifecycleLocks{}, openLifecycleLocks{}
	engineer.Capabilities = append(engineer.Capabilities, authorityv1.CapabilityWorkHandoff, authorityv1.CapabilityLeaseRelease)
	current := lifecycleWork(t, store)
	request := authorityv1.HandoffRequest{
		BeadID: current.BeadID, ExpectedVersion: current.Version, ExpectedIntegrity: current.Integrity, Fence: fenceFromLease(lease),
		HeadSHA: strings.Repeat("c", 40), EvidenceRefs: []string{"evidence-fence"}, NextProfileID: "qa",
		IdempotencyKey: "handoff-fence", TraceRef: "trace-handoff-fence",
	}
	if _, err := service.Handoff(context.Background(), engineer, request); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*authorityv1.FencingTuple){
		"canonical claim": func(fence *authorityv1.FencingTuple) { fence.CanonicalClaimAttemptID = "spliced-claim" },
		"lease":           func(fence *authorityv1.FencingTuple) { fence.LeaseID = "spliced-lease" },
		"generation":      func(fence *authorityv1.FencingTuple) { fence.FenceGeneration = "spliced-generation" },
		"epoch":           func(fence *authorityv1.FencingTuple) { fence.LeaseEpoch++ },
		"base":            func(fence *authorityv1.FencingTuple) { fence.BaseSHA = strings.Repeat("d", 40) },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := request
			mutate(&candidate.Fence)
			_, err := service.Handoff(context.Background(), engineer, candidate)
			assertDenial(t, err, authorityv1.ErrorPolicyDenied, ruleLifecycleIdempotency, "use the original normalized lifecycle request or a new idempotency key", "work.read")
		})
	}
}

func TestLifecycleReplayRepairsMissingReceiptBeforeSuccess(t *testing.T) {
	clock := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	service, store, _, events, engineer, lease := lifecycleFixture(t, &clock)
	service.barrier, service.workLocks = openLifecycleLocks{}, openLifecycleLocks{}
	engineer.Capabilities = append(engineer.Capabilities, authorityv1.CapabilityWorkHandoff, authorityv1.CapabilityLeaseRelease)
	current := lifecycleWork(t, store)
	request := authorityv1.HandoffRequest{
		BeadID: current.BeadID, ExpectedVersion: current.Version, ExpectedIntegrity: current.Integrity, Fence: fenceFromLease(lease),
		HeadSHA: strings.Repeat("c", 40), EvidenceRefs: []string{"evidence-receipt"}, NextProfileID: "qa",
		IdempotencyKey: "handoff-receipt", TraceRef: "trace-handoff-receipt",
	}
	events.failOperation = "work.handoff.receipt"
	if _, err := service.Handoff(context.Background(), engineer, request); err == nil {
		t.Fatal("missing lifecycle receipt was reported as success")
	}
	replayed, err := service.Handoff(context.Background(), engineer, request)
	if err != nil || !replayed.Replayed || !strings.HasPrefix(replayed.ReceiptRef, "lifecycle-reconciliation-") {
		t.Fatalf("repaired replay=%#v err=%v", replayed, err)
	}
	found := false
	for _, event := range events.events {
		found = found || event.Operation == "work.handoff.reconciliation" && event.Outcome == outcomeVerified
	}
	if !found {
		t.Fatal("replay succeeded without a durable reconciliation receipt")
	}
}

func TestNonterminalLifecycleOutcomesRemainRecoverable(t *testing.T) {
	statuses := []authorityv1.RunDispositionStatus{
		authorityv1.RunBlocked, authorityv1.RunInReview, authorityv1.RunChangesRequested, authorityv1.RunNoWork,
		authorityv1.RunPreempted, authorityv1.RunCancelled, authorityv1.RunFailed,
	}
	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			clock := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
			service, store, _, _, engineer, lease := lifecycleFixture(t, &clock)
			service.barrier, service.workLocks = openLifecycleLocks{}, openLifecycleLocks{}
			engineer.Capabilities = append(engineer.Capabilities, authorityv1.CapabilityWorkHandoff, authorityv1.CapabilityLeaseRelease)
			head := strings.Repeat("c", 40)
			current := lifecycleWork(t, store)
			if _, err := service.Handoff(context.Background(), engineer, authorityv1.HandoffRequest{
				BeadID: current.BeadID, ExpectedVersion: current.Version, ExpectedIntegrity: current.Integrity, Fence: fenceFromLease(lease), HeadSHA: head,
				EvidenceRefs: []string{"evidence-handoff"}, NextProfileID: "qa", IdempotencyKey: "handoff-" + string(status), TraceRef: "trace-handoff-" + string(status),
			}); err != nil {
				t.Fatal(err)
			}
			current = lifecycleWork(t, store)
			failure := failureForStatus(status)
			result, err := service.RecordRunDisposition(context.Background(), lifecyclePrincipal("delivery-orchestrator", authorityv1.CapabilityRunDisposition), authorityv1.RunDispositionRequest{
				BeadID: current.BeadID, ExpectedVersion: current.Version, ExpectedIntegrity: current.Integrity, HeadSHA: head, Status: status,
				EvidenceRefs: []string{"evidence-run"}, IdempotencyKey: "run-" + string(status), TraceRef: "trace-run-" + string(status), Failure: failure,
			})
			if err != nil || result.Work.RunDisposition == nil || result.Work.RunDisposition.Status != status {
				t.Fatalf("result=%#v err=%v", result, err)
			}
			if status == authorityv1.RunInReview {
				if result.Work.LifecycleState != authorityv1.LifecycleInReview {
					t.Fatalf("in-review lifecycle=%s", result.Work.LifecycleState)
				}
				qa := recordReview(t, service, lifecyclePrincipal("qa", authorityv1.CapabilityReviewRecord), result.Work, head, authorityv1.ReviewAccepted, "qa-after-run-in-review")
				security := recordReview(t, service, lifecyclePrincipal("security-reviewer", authorityv1.CapabilityReviewRecord), qa.Work, head, authorityv1.ReviewAccepted, "security-after-run-in-review")
				completed, completeErr := service.RecordRunDisposition(context.Background(), lifecyclePrincipal("delivery-orchestrator", authorityv1.CapabilityRunDisposition), authorityv1.RunDispositionRequest{
					BeadID: security.Work.BeadID, ExpectedVersion: security.Work.Version, ExpectedIntegrity: security.Work.Integrity, HeadSHA: head,
					Status: authorityv1.RunCompleted, EvidenceRefs: []string{"evidence-completed"}, IdempotencyKey: "run-completed-after-in-review", TraceRef: "trace-completed-after-in-review",
				})
				if completeErr != nil || completed.Work.RunDisposition == nil || completed.Work.RunDisposition.Status != authorityv1.RunCompleted || len(completed.Work.RunHistory) != 1 {
					t.Fatalf("completed after in-review=%#v err=%v", completed, completeErr)
				}
				return
			}
			if result.Work.LifecycleState != authorityv1.LifecycleInProgress {
				t.Fatalf("nonterminal lifecycle=%s", result.Work.LifecycleState)
			}
		})
	}

	t.Run("blocked review", func(t *testing.T) {
		clock := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
		service, store, _, _, engineer, lease := lifecycleFixture(t, &clock)
		service.barrier, service.workLocks = openLifecycleLocks{}, openLifecycleLocks{}
		engineer.Capabilities = append(engineer.Capabilities, authorityv1.CapabilityWorkHandoff, authorityv1.CapabilityLeaseRelease)
		head := strings.Repeat("d", 40)
		current := lifecycleWork(t, store)
		if _, err := service.Handoff(context.Background(), engineer, authorityv1.HandoffRequest{
			BeadID: current.BeadID, ExpectedVersion: current.Version, ExpectedIntegrity: current.Integrity, Fence: fenceFromLease(lease), HeadSHA: head,
			EvidenceRefs: []string{"evidence-handoff"}, NextProfileID: "qa", IdempotencyKey: "handoff-blocked-review", TraceRef: "trace-handoff-blocked-review",
		}); err != nil {
			t.Fatal(err)
		}
		current = lifecycleWork(t, store)
		request := reviewRequest(current, head, authorityv1.ReviewBlocked, "qa-blocked")
		request.Failure = &authorityv1.FailureContext{Reason: "dependency-unavailable", BlockedBy: []string{"M3-P999"}, FailureFingerprint: "review-dependency-unavailable", Attempt: 2, NextAction: "resolve-dependency"}
		result, err := service.RecordReviewVerdict(context.Background(), lifecyclePrincipal("qa", authorityv1.CapabilityReviewRecord), request)
		if err != nil || result.Work.LifecycleState != authorityv1.LifecycleInProgress || len(result.Work.Blockers) == 0 {
			t.Fatalf("blocked review=%#v err=%v", result, err)
		}
	})
}

func failureForStatus(status authorityv1.RunDispositionStatus) *authorityv1.FailureContext {
	failure := &authorityv1.FailureContext{Reason: "run-" + string(status), Attempt: 2, NextAction: "resume-same-ticket"}
	if status == authorityv1.RunBlocked {
		failure.BlockedBy = []string{"M3-P999"}
	}
	if status != authorityv1.RunInReview && status != authorityv1.RunNoWork {
		failure.FailureFingerprint = "fingerprint-" + string(status)
	}
	return failure
}

func lifecycleWork(t *testing.T, store *memoryClaimStore) authorityv1.WorkItem {
	t.Helper()
	item, err := store.Get(context.Background(), store.item.TenantID, store.item.ProjectID, store.item.BeadID)
	if err != nil {
		t.Fatalf("Get lifecycle work: %v", err)
	}
	return item
}

func lifecyclePrincipal(profileID string, capabilities ...authorityv1.Capability) authorityv1.Principal {
	principal := reader()
	principal.ProfileID, principal.PrincipalID = profileID, profileID
	principal.Capabilities = append(principal.Capabilities, capabilities...)
	return principal
}

func recordReview(t *testing.T, service *Service, principal authorityv1.Principal, item authorityv1.WorkItem, head string, verdict authorityv1.ReviewVerdict, key string) authorityv1.LifecycleMutationResponse {
	t.Helper()
	response, err := service.RecordReviewVerdict(context.Background(), principal, reviewRequest(item, head, verdict, key))
	if err != nil {
		t.Fatalf("review %s: %v", principal.ProfileID, err)
	}
	return response
}

func reviewRequest(item authorityv1.WorkItem, head string, verdict authorityv1.ReviewVerdict, key string) authorityv1.ReviewVerdictRequest {
	return authorityv1.ReviewVerdictRequest{
		BeadID: item.BeadID, ExpectedVersion: item.Version, ExpectedIntegrity: item.Integrity, HeadSHA: head,
		Verdict: verdict, EvidenceRefs: []string{"evidence-" + key}, IdempotencyKey: key, TraceRef: "trace-" + key,
	}
}

func countLifecycleReceipts(events []authorityv1.Event) int {
	count := 0
	for _, event := range events {
		switch event.Operation {
		case "work.handoff.receipt", "work.review-verdict.receipt", "work.run-disposition.receipt", "work.reconciliation.receipt", "work.terminal-transition.receipt":
			count++
		}
	}
	return count
}
