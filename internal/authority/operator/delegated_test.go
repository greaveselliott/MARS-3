/*
FactoryDocSync:
docs:
- docs/features/F-002-work-authority.md
- docs/design-docs/ADR-001-git-beads-authority.md
- docs/design-docs/ADR-008-standing-delivery-delegation.md
- docs/code-documentation-map.md
*/

package operator

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	authorityv1 "github.com/greaveselliott/MARS-3/api/authority/v1"
	"github.com/greaveselliott/MARS-3/internal/authority/gateway"
)

// This bridge uses the production Service and lazy operator adapter, not the
// synthetic dispatch stub. It does not pretend that absent lease/claim stores
// are configured or that a delegated identity can bootstrap those stores.
type standingCanonicalFixture struct{ work authorityv1.WorkItem }

func (store *standingCanonicalFixture) Get(_ context.Context, tenant, project, bead string) (authorityv1.WorkItem, error) {
	if store.work.TenantID != tenant || store.work.ProjectID != project || store.work.BeadID != bead {
		return authorityv1.WorkItem{}, gateway.ErrWorkNotFound
	}
	return store.work, nil
}

func (store *standingCanonicalFixture) List(_ context.Context, tenant, project string) ([]authorityv1.WorkItem, error) {
	if store.work.TenantID != tenant || store.work.ProjectID != project {
		return nil, nil
	}
	return []authorityv1.WorkItem{store.work}, nil
}

type standingEventFixture struct{ events []authorityv1.Event }

func (sink *standingEventFixture) Append(_ context.Context, event authorityv1.Event) (authorityv1.Event, error) {
	event.Sequence = uint64(len(sink.events) + 1)
	sink.events = append(sink.events, event)
	return event, nil
}

func TestStandingRealGatewayReadsButCannotInventClaimOrLeaseAuthority(t *testing.T) {
	fixture := newStandingFixture(t)
	work := fixture.backend.work
	work.DisplayID = "P-001"
	work.NativeStatus, work.LifecycleState = "open", authorityv1.LifecycleBacklog
	work.GoalIDs, work.ProductDecisionIDs = []string{"G-001"}, []string{"PD-002"}
	work.ScenarioIDs, work.VerificationOrder = []string{"F-003-S1"}, []string{"qa", "security-reviewer", "delivery-orchestrator"}
	work.Labels = []authorityv1.Label{authorityv1.LabelPublicAccepted}
	work.Version = authorityv1.WorkVersion{AuthorityGeneration: "synthetic-authority", IssueIncarnation: "synthetic-incarnation", IssueMutationSequence: 1, DependencyGraphRevision: 1}
	work.Integrity = authorityv1.IntegrityDigests{Lineage: strings.Repeat("1", 64), DependencyOutcomes: strings.Repeat("2", 64), Blockers: strings.Repeat("3", 64), ExclusivePaths: strings.Repeat("4", 64)}
	canonical := &standingCanonicalFixture{work: work}
	events := &standingEventFixture{}
	service, err := gateway.New(canonical, events, func() time.Time { return fixture.now })
	if err != nil {
		t.Fatal(err)
	}
	fixture.gate.gateway = &lazyReadGateway{service: service, mutations: true}
	claim := fixture.request
	fixture.request.Operation, fixture.request.Mutation = "work.get", nil
	fixture.request.Read = &ReadRequest{Operation: "work.get", BeadID: "M3-P001", TraceRef: "synthetic-real-read"}
	result, err := fixture.execute(t)
	if err != nil {
		t.Fatalf("real gateway read: %v; bounded events: %v", err, events.events)
	}
	if got, ok := result.(authorityv1.WorkItem); !ok || got.BeadID != work.BeadID || len(events.events) == 0 {
		t.Fatal("real gateway did not return audited canonical projection")
	}
	fixture.request = claim
	fixture.request.ID = "synthetic-unconfigured-claim"
	fixture.request.Mutation.Claim.ExpectedVersion, fixture.request.Mutation.Claim.ExpectedIntegrity = work.Version, work.Integrity
	if _, err := fixture.execute(t); !errors.Is(err, ErrGateway) {
		t.Fatalf("delegation invented claim/lease substrate: %v", err)
	}
	if canonical.work.LifecycleState != authorityv1.LifecycleBacklog || canonical.work.ClaimAttemptID != "" {
		t.Fatal("unconfigured gateway changed canonical work")
	}
}

type standingFixture struct {
	gate       *StandingGate
	parent     StandingRuntime
	session    StandingSession
	request    StandingRequest
	activation *standingActivationFixture
	backend    *standingGatewayFixture
	replays    *standingReplayFixture
	now        time.Time
}

// Single-threaded disposable stores for the operator -> production gateway
// composition. They are neither persistence nor concurrency qualification;
// the production gateway owns policy, postimage and transition validation.
type standingDeliveryWork struct {
	work           authorityv1.WorkItem
	claimCalls     int
	lifecycleCalls int
}

func standingCopy[T any](value T) T {
	document, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	var clone T
	if err := json.Unmarshal(document, &clone); err != nil {
		panic(err)
	}
	return clone
}

func (store *standingDeliveryWork) Get(_ context.Context, tenant, project, bead string) (authorityv1.WorkItem, error) {
	if tenant != store.work.TenantID || project != store.work.ProjectID || bead != store.work.BeadID {
		return authorityv1.WorkItem{}, gateway.ErrWorkNotFound
	}
	return standingCopy(store.work), nil
}

func (store *standingDeliveryWork) List(_ context.Context, tenant, project string) ([]authorityv1.WorkItem, error) {
	if tenant != store.work.TenantID || project != store.work.ProjectID {
		return nil, nil
	}
	return []authorityv1.WorkItem{standingCopy(store.work)}, nil
}

func (store *standingDeliveryWork) CompareAndSwapClaim(_ context.Context, mutation gateway.ClaimMutation) (authorityv1.WorkItem, error) {
	if mutation.TenantID != store.work.TenantID || mutation.ProjectID != store.work.ProjectID || mutation.BeadID != store.work.BeadID ||
		mutation.ExpectedVersion != store.work.Version || mutation.ExpectedIntegrity != store.work.Integrity || store.work.LifecycleState != authorityv1.LifecycleBacklog {
		return authorityv1.WorkItem{}, gateway.ErrStaleWorkVersion
	}
	store.claimCalls++
	store.work.LifecycleState, store.work.NativeStatus = authorityv1.LifecycleInProgress, "in_progress"
	store.work.Assignee, store.work.ClaimAttemptID = mutation.Assignee, mutation.AttemptID
	store.work.Version.IssueMutationSequence++
	store.work.Integrity.Lineage = standingDigest([]byte(mutation.IdempotencyKey))
	return standingCopy(store.work), nil
}

func (store *standingDeliveryWork) CompareAndSwapLifecycle(_ context.Context, mutation gateway.LifecycleMutation) (authorityv1.WorkItem, error) {
	if mutation.TenantID != store.work.TenantID || mutation.ProjectID != store.work.ProjectID || mutation.BeadID != store.work.BeadID ||
		mutation.ExpectedVersion != store.work.Version || mutation.ExpectedIntegrity != store.work.Integrity {
		return authorityv1.WorkItem{}, gateway.ErrStaleWorkVersion
	}
	work := standingCopy(store.work)
	switch mutation.Operation {
	case gateway.LifecycleHandoff:
		work.LifecycleState = authorityv1.LifecycleInReview
		work.Handoff = &authorityv1.HandoffRecord{AttemptID: mutation.AttemptID, CanonicalClaimAttemptID: mutation.CanonicalClaimAttemptID,
			FenceDigest: mutation.HandoffFenceDigest, HeadSHA: mutation.HeadSHA, EvidenceRefs: mutation.EvidenceRefs, NextProfileID: mutation.NextProfileID, IdempotencyKey: mutation.IdempotencyKey}
	case gateway.LifecycleReview:
		work.Reviews = append(work.Reviews, authorityv1.ReviewRecord{ReviewerProfileID: mutation.PrincipalProfileID, Verdict: mutation.Verdict,
			HeadSHA: mutation.HeadSHA, EvidenceRefs: mutation.EvidenceRefs, IdempotencyKey: mutation.IdempotencyKey, Failure: mutation.Failure})
		if mutation.Verdict != authorityv1.ReviewAccepted {
			work.LifecycleState = authorityv1.LifecycleInProgress
		}
	case gateway.LifecycleRun:
		work.RunDisposition = &authorityv1.RunDispositionRecord{PrincipalProfileID: mutation.PrincipalProfileID, Status: mutation.RunStatus,
			HeadSHA: mutation.HeadSHA, EvidenceRefs: mutation.EvidenceRefs, IdempotencyKey: mutation.IdempotencyKey, Failure: mutation.Failure}
	case gateway.LifecycleReconcile:
		work.Reconciliation = &authorityv1.ReconciliationRecord{PrincipalProfileID: mutation.PrincipalProfileID, HeadSHA: mutation.HeadSHA,
			MergedSHA: mutation.MergedSHA, MergedTree: mutation.MergedTree, PullRequestID: mutation.PullRequestID,
			ProtectedMainRunID: mutation.ProtectedMainRunID, EvidenceRefs: mutation.EvidenceRefs, IdempotencyKey: mutation.IdempotencyKey}
	case gateway.LifecycleTerminal:
		work.LifecycleState, work.NativeStatus = authorityv1.LifecycleDone, "closed"
		work.Terminal = &authorityv1.TerminalRecord{PrincipalProfileID: mutation.PrincipalProfileID, HeadSHA: mutation.HeadSHA,
			EvidenceRefs: mutation.EvidenceRefs, IdempotencyKey: mutation.IdempotencyKey}
	default:
		return authorityv1.WorkItem{}, gateway.ErrStaleWorkVersion
	}
	store.lifecycleCalls++
	work.Version.IssueMutationSequence++
	work.Integrity.Lineage = standingDigest([]byte(mutation.IdempotencyKey + ":" + string(mutation.Operation)))
	store.work = standingCopy(work)
	return standingCopy(work), nil
}

type standingDeliveryLeases struct {
	sagas map[string]gateway.ClaimSaga
	now   func() time.Time
	epoch uint64
}

func (store *standingDeliveryLeases) Lookup(_ context.Context, tenant, project, key string) (gateway.ClaimSaga, bool, error) {
	saga, ok := store.sagas[key]
	if ok && (saga.Intent.TenantID != tenant || saga.Intent.ProjectID != project) {
		return gateway.ClaimSaga{}, false, gateway.ErrIdempotencyConflict
	}
	return standingCopy(saga), ok, nil
}

func (store *standingDeliveryLeases) Begin(_ context.Context, intent gateway.ClaimIntent) (gateway.ClaimSaga, error) {
	if saga, ok := store.sagas[intent.IdempotencyKey]; ok {
		if saga.RequestDigest != intent.RequestDigest {
			return gateway.ClaimSaga{}, gateway.ErrIdempotencyConflict
		}
		return standingCopy(saga), nil
	}
	saga := gateway.ClaimSaga{RequestDigest: intent.RequestDigest, Phase: gateway.ClaimPhaseIntent, Intent: standingCopy(intent)}
	store.sagas[intent.IdempotencyKey] = saga
	return standingCopy(saga), nil
}

func (store *standingDeliveryLeases) MarkCanonicalClaimed(_ context.Context, tenant, project, key, digest string, work authorityv1.WorkItem) (gateway.ClaimSaga, error) {
	saga, ok := store.sagas[key]
	if !ok || saga.RequestDigest != digest || saga.Phase != gateway.ClaimPhaseIntent || saga.Intent.TenantID != tenant || saga.Intent.ProjectID != project {
		return gateway.ClaimSaga{}, gateway.ErrIdempotencyConflict
	}
	saga.Phase, saga.Work = gateway.ClaimPhaseCanonical, standingCopy(work)
	store.sagas[key] = saga
	return standingCopy(saga), nil
}

func (store *standingDeliveryLeases) IssueLease(_ context.Context, key, digest string, request gateway.LeaseRequest) (gateway.ClaimSaga, error) {
	saga, ok := store.sagas[key]
	if !ok || saga.RequestDigest != digest || saga.Phase != gateway.ClaimPhaseCanonical {
		return gateway.ClaimSaga{}, gateway.ErrIdempotencyConflict
	}
	for _, existing := range store.sagas {
		if existing.Lease.BeadID == request.BeadID && existing.Lease.Active && existing.Lease.ExpiresAt.After(store.now()) {
			return gateway.ClaimSaga{}, errors.New("synthetic live lease conflict")
		}
	}
	store.epoch++
	saga.Phase = gateway.ClaimPhaseComplete
	saga.Lease = authorityv1.CapabilityLease{LeaseID: "synthetic-lease-" + digest[:16], TenantID: request.TenantID, ProjectID: request.ProjectID,
		BeadID: request.BeadID, AttemptID: request.AttemptID, CanonicalClaimAttemptID: request.CanonicalClaimAttemptID,
		IdempotencyKey: request.IdempotencyKey, FenceGeneration: "synthetic-generation", LeaseEpoch: store.epoch, ClaimVersion: request.ClaimVersion,
		BaseSHA: request.BaseSHA, Capability: request.Capability, ExclusivePaths: request.ExclusivePaths, Labels: request.Labels,
		IssuedAt: store.now(), ExpiresAt: request.MaximumExpiry, State: authorityv1.LeaseActive, Active: true}
	saga.ReceiptRef = "synthetic-receipt-" + digest[:16]
	store.sagas[key] = standingCopy(saga)
	return standingCopy(saga), nil
}

func standingFence(lease authorityv1.CapabilityLease) authorityv1.FencingTuple {
	return authorityv1.FencingTuple{TenantID: lease.TenantID, ProjectID: lease.ProjectID, BeadID: lease.BeadID, AttemptID: lease.AttemptID,
		CanonicalClaimAttemptID: lease.CanonicalClaimAttemptID, IdempotencyKey: lease.IdempotencyKey, LeaseID: lease.LeaseID,
		FenceGeneration: lease.FenceGeneration, LeaseEpoch: lease.LeaseEpoch, ClaimVersion: lease.ClaimVersion, BaseSHA: lease.BaseSHA,
		Capability: lease.Capability, ExclusivePaths: slices.Clone(lease.ExclusivePaths), Labels: slices.Clone(lease.Labels)}
}

func (store *standingDeliveryLeases) GetLease(_ context.Context, tenant, project, id string) (authorityv1.CapabilityLease, error) {
	for _, saga := range store.sagas {
		if saga.Lease.LeaseID == id && saga.Lease.TenantID == tenant && saga.Lease.ProjectID == project {
			return standingCopy(saga.Lease), nil
		}
	}
	return authorityv1.CapabilityLease{}, errors.New("synthetic lease not found")
}

func (store *standingDeliveryLeases) ActiveLeaseForBead(_ context.Context, tenant, project, bead string) (authorityv1.CapabilityLease, bool, error) {
	for _, saga := range store.sagas {
		lease := saga.Lease
		if lease.TenantID == tenant && lease.ProjectID == project && lease.BeadID == bead && lease.Active && lease.ExpiresAt.After(store.now()) {
			return standingCopy(lease), true, nil
		}
	}
	return authorityv1.CapabilityLease{}, false, nil
}

func (store *standingDeliveryLeases) ValidateFence(ctx context.Context, fence authorityv1.FencingTuple) (authorityv1.CapabilityLease, error) {
	lease, err := store.GetLease(ctx, fence.TenantID, fence.ProjectID, fence.LeaseID)
	if err != nil || !lease.Active || lease.State != authorityv1.LeaseActive || !lease.ExpiresAt.After(store.now()) || !reflect.DeepEqual(standingFence(lease), fence) {
		return authorityv1.CapabilityLease{}, errors.New("synthetic stale fence")
	}
	return lease, nil
}

func (store *standingDeliveryLeases) updateLease(ctx context.Context, fence authorityv1.FencingTuple, expiry time.Time, release bool) (authorityv1.CapabilityLease, error) {
	lease, err := store.ValidateFence(ctx, fence)
	if err != nil {
		return authorityv1.CapabilityLease{}, err
	}
	if release {
		lease.Active, lease.State = false, authorityv1.LeaseReleased
	} else {
		lease.ExpiresAt = expiry
	}
	for key, saga := range store.sagas {
		if saga.Lease.LeaseID == lease.LeaseID {
			saga.Lease = standingCopy(lease)
			store.sagas[key] = saga
			return standingCopy(lease), nil
		}
	}
	return authorityv1.CapabilityLease{}, errors.New("synthetic lease not found")
}

func (store *standingDeliveryLeases) Renew(ctx context.Context, fence authorityv1.FencingTuple, expiry time.Time) (authorityv1.CapabilityLease, error) {
	return store.updateLease(ctx, fence, expiry, false)
}
func (store *standingDeliveryLeases) Release(ctx context.Context, fence authorityv1.FencingTuple) (authorityv1.CapabilityLease, error) {
	return store.updateLease(ctx, fence, time.Time{}, true)
}
func (*standingDeliveryLeases) Revoke(context.Context, authorityv1.RevokeLeaseRequest) (authorityv1.CapabilityLease, error) {
	return authorityv1.CapabilityLease{}, errors.New("revocation is outside this synthetic fixture")
}
func (*standingDeliveryLeases) Enter(context.Context, string, string) (func(), error) {
	return func() {}, nil
}
func (*standingDeliveryLeases) EnterWork(context.Context, string, string, string) (func(), error) {
	return func() {}, nil
}
func (store *standingDeliveryLeases) EnterEffect(ctx context.Context, fence authorityv1.FencingTuple) (authorityv1.CapabilityLease, func(), error) {
	lease, err := store.ValidateFence(ctx, fence)
	return lease, func() {}, err
}

func TestStandingRealGatewayCompletesClaimLeaseAndOrderedDelivery(t *testing.T) {
	fixture := newStandingFixture(t)
	fixture.parent.Roles = append(fixture.parent.Roles,
		StandingRole{Bead: "M3-P001", PrincipalID: "synthetic-qa", ProfileID: "qa", Class: "qa", Operations: []string{"work.get", "review.record"}},
		StandingRole{Bead: "M3-P001", PrincipalID: "synthetic-security", ProfileID: "security-reviewer", Class: "security", Operations: []string{"work.get", "review.record"}},
		StandingRole{Bead: "M3-P001", PrincipalID: "synthetic-orchestrator", ProfileID: "delivery-orchestrator", Class: "orchestrator", Operations: []string{"work.get", "run.disposition", "work.reconcile", "work.close"}})
	work := standingCopy(fixture.backend.work)
	work.DisplayID, work.NativeStatus, work.LifecycleState = "P-001", "open", authorityv1.LifecycleBacklog
	work.GoalIDs, work.ProductDecisionIDs = []string{"G-001"}, []string{"PD-002"}
	work.ScenarioIDs, work.VerificationOrder = []string{"F-003-S1"}, []string{"qa", "security-reviewer", "delivery-orchestrator"}
	work.Labels = []authorityv1.Label{authorityv1.LabelPublicAccepted}
	work.Version = authorityv1.WorkVersion{AuthorityGeneration: "synthetic-authority", IssueIncarnation: "synthetic-incarnation", IssueMutationSequence: 1, DependencyGraphRevision: 1}
	work.Integrity = authorityv1.IntegrityDigests{Lineage: strings.Repeat("1", 64), DependencyOutcomes: strings.Repeat("2", 64), Blockers: strings.Repeat("3", 64), ExclusivePaths: strings.Repeat("4", 64)}
	canonical := &standingDeliveryWork{work: work}
	leases := &standingDeliveryLeases{sagas: map[string]gateway.ClaimSaga{}, now: func() time.Time { return fixture.now }}
	events := &standingEventFixture{}
	service, err := gateway.NewWithClaims(canonical, leases, events, func() time.Time { return fixture.now })
	if err != nil {
		t.Fatal(err)
	}
	fixture.gate.gateway = &lazyReadGateway{service: service, mutations: true}
	fixture.request.Mutation.Claim.ExpectedVersion, fixture.request.Mutation.Claim.ExpectedIntegrity = work.Version, work.Integrity
	result, err := fixture.execute(t)
	if err != nil {
		t.Fatalf("claim: %v; events=%v", err, events.events)
	}
	claim, ok := result.(authorityv1.ClaimResponse)
	if !ok || !claim.Lease.Active || claim.Lease.LeaseEpoch != 1 || canonical.claimCalls != 1 {
		t.Fatal("claim did not issue one verified lease")
	}
	fence := standingFence(claim.Lease)
	issue := func(operation, id string) {
		fixture.request = StandingRequest{ID: id, Operation: operation, IssuedAt: fixture.now.Add(-time.Second), ExpiresAt: fixture.now.Add(time.Minute)}
	}
	role := func(class string) {
		for _, assignment := range fixture.parent.Roles {
			if assignment.Class == class {
				fixture.session.PrincipalID, fixture.session.ProfileID, fixture.session.Class = assignment.PrincipalID, assignment.ProfileID, class
				fixture.session.ID = "synthetic-session-" + class
				return
			}
		}
		t.Fatal("missing synthetic role")
	}
	issue("effect.validate", "synthetic-effect")
	fixture.request.Mutation = &MutationRequest{Operation: "effect.validate", Effect: &authorityv1.EffectValidationRequest{Fence: fence, Path: work.ExclusivePaths[0], EffectID: "synthetic-effect", TraceRef: "synthetic-effect"}}
	if _, err := fixture.execute(t); err != nil {
		t.Fatalf("current effect fence: %v; events=%v", err, events.events)
	}
	issue("effect.validate", "synthetic-stale-effect")
	stale := standingCopy(fence)
	stale.LeaseEpoch++
	fixture.request.Mutation = &MutationRequest{Operation: "effect.validate", Effect: &authorityv1.EffectValidationRequest{Fence: stale, Path: work.ExclusivePaths[0], EffectID: "synthetic-stale-effect", TraceRef: "synthetic-stale-effect"}}
	if _, err := fixture.execute(t); !errors.Is(err, ErrGateway) {
		t.Fatalf("stale epoch admitted: %v", err)
	}
	fixture.now = fixture.now.Add(2 * time.Minute)
	issue("lease.renew", "synthetic-renew")
	fixture.request.Mutation = &MutationRequest{Operation: "lease.renew", Renew: &authorityv1.RenewLeaseRequest{Fence: fence, NewExpiry: claim.Lease.ExpiresAt.Add(time.Minute), TraceRef: "synthetic-renew"}}
	if _, err := fixture.execute(t); err != nil {
		t.Fatalf("renew: %v; events=%v", err, events.events)
	}
	head := strings.Repeat("c", 40)
	issue("work.handoff", "synthetic-handoff")
	fixture.request.Handoff = &authorityv1.HandoffRequest{BeadID: work.BeadID, ExpectedVersion: canonical.work.Version, ExpectedIntegrity: canonical.work.Integrity, Fence: fence, HeadSHA: head, EvidenceRefs: []string{"synthetic-implementation-evidence"}, NextProfileID: "qa", IdempotencyKey: "synthetic-handoff", TraceRef: "synthetic-handoff"}
	if _, err := fixture.execute(t); err != nil {
		t.Fatalf("handoff: %v; events=%v", err, events.events)
	}
	if lease, found, err := leases.ActiveLeaseForBead(context.Background(), work.TenantID, work.ProjectID, work.BeadID); err != nil || found || lease.Active {
		t.Fatal("handoff retained active lease")
	}
	review := func(class, id string) error {
		role(class)
		issue("review.record", id)
		fixture.request.Review = &authorityv1.ReviewVerdictRequest{BeadID: work.BeadID, ExpectedVersion: canonical.work.Version, ExpectedIntegrity: canonical.work.Integrity, HeadSHA: head, Verdict: authorityv1.ReviewAccepted, EvidenceRefs: []string{id + "-evidence"}, IdempotencyKey: id, TraceRef: id}
		_, err := fixture.execute(t)
		return err
	}
	if err := review("security", "synthetic-premature-security"); !errors.Is(err, ErrGateway) || canonical.lifecycleCalls != 1 {
		t.Fatalf("security bypassed QA: %v", err)
	}
	role("orchestrator")
	issue("work.close", "synthetic-premature-close")
	fixture.request.Close = &authorityv1.TerminalTransitionRequest{BeadID: work.BeadID, ExpectedVersion: canonical.work.Version, ExpectedIntegrity: canonical.work.Integrity, HeadSHA: head, EvidenceRefs: []string{"synthetic-premature-close"}, IdempotencyKey: "synthetic-premature-close", TraceRef: "synthetic-premature-close"}
	if _, err := fixture.execute(t); !errors.Is(err, ErrGateway) || canonical.lifecycleCalls != 1 {
		t.Fatalf("premature close admitted: %v", err)
	}
	if err := review("qa", "synthetic-qa-accept"); err != nil {
		t.Fatalf("QA: %v; events=%v", err, events.events)
	}
	if err := review("security", "synthetic-security-accept"); err != nil {
		t.Fatalf("Security: %v; events=%v", err, events.events)
	}
	role("orchestrator")
	issue("run.disposition", "synthetic-completed-run")
	fixture.request.Run = &authorityv1.RunDispositionRequest{BeadID: work.BeadID, ExpectedVersion: canonical.work.Version, ExpectedIntegrity: canonical.work.Integrity, HeadSHA: head, Status: authorityv1.RunCompleted, EvidenceRefs: []string{"synthetic-completed-run"}, IdempotencyKey: "synthetic-completed-run", TraceRef: "synthetic-completed-run"}
	if _, err := fixture.execute(t); err != nil {
		t.Fatalf("completed run: %v; events=%v", err, events.events)
	}
	issue("work.reconcile", "synthetic-reconciliation")
	fixture.request.Reconcile = &authorityv1.ReconciliationRequest{BeadID: work.BeadID, ExpectedVersion: canonical.work.Version, ExpectedIntegrity: canonical.work.Integrity, HeadSHA: head, MergedSHA: strings.Repeat("d", 40), MergedTree: strings.Repeat("e", 40), PullRequestID: "synthetic-pr", ProtectedMainRunID: "synthetic-main-ci", EvidenceRefs: []string{"synthetic-merge"}, IdempotencyKey: "synthetic-reconciliation", TraceRef: "synthetic-reconciliation"}
	if _, err := fixture.execute(t); err != nil {
		t.Fatalf("reconciliation: %v; events=%v", err, events.events)
	}
	issue("work.close", "synthetic-close")
	fixture.request.Close = &authorityv1.TerminalTransitionRequest{BeadID: work.BeadID, ExpectedVersion: canonical.work.Version, ExpectedIntegrity: canonical.work.Integrity, HeadSHA: head, EvidenceRefs: []string{"synthetic-close"}, IdempotencyKey: "synthetic-close", TraceRef: "synthetic-close"}
	if _, err := fixture.execute(t); err != nil {
		t.Fatalf("close: %v; events=%v", err, events.events)
	}
	if canonical.work.LifecycleState != authorityv1.LifecycleDone || canonical.work.NativeStatus != "closed" || canonical.lifecycleCalls != 6 || len(canonical.work.Reviews) != 2 || canonical.work.Reviews[0].ReviewerProfileID != "qa" || canonical.work.Reviews[1].ReviewerProfileID != "security-reviewer" {
		t.Fatal("delivery did not retain ordered independent-role evidence")
	}
}

func TestStandingDirectoryScopesAreCanonicalAndDoNotEscape(t *testing.T) {
	scopes := []string{"internal/platform/", "docs/features/F-003-local-substrate.md"}
	for _, test := range []struct {
		target  string
		allowed bool
	}{
		{"internal/platform/worker.go", true}, {"internal/platform/nested/worker.go", true},
		{"docs/features/F-003-local-substrate.md", true}, {"internal/platform-other/worker.go", false},
		{"internal/platform/../outside.go", false}, {"internal/platform//worker.go", false},
		{"internal/platform/.git/config", false}, {"/internal/platform/worker.go", false},
		{"internal/platform/worker.go/", false}, {"internal/platform/", false},
	} {
		if got := standingPathWithin(scopes, test.target); got != test.allowed {
			t.Fatalf("scope matcher %q = %v", test.target, got)
		}
	}
	fixture := newStandingFixture(t)
	fixture.parent.Scopes[0].Paths = []string{"internal/platform/"}
	if !validStandingScopes(fixture.parent.Scopes) {
		t.Fatal("canonical directory scope rejected")
	}
	for _, invalid := range []string{"internal/platform//", "internal/platform/../", "internal/.git/", "../", "./"} {
		fixture.parent.Scopes[0].Paths = []string{invalid}
		if validStandingScopes(fixture.parent.Scopes) {
			t.Fatalf("unsafe directory %q admitted", invalid)
		}
	}
}

func TestStandingHandoffCapabilityBundleIsActionBound(t *testing.T) {
	if !slices.Equal(standingActionCapabilities("work.handoff", authorityv1.CapabilityWorkHandoff), []authorityv1.Capability{authorityv1.CapabilityWorkHandoff, authorityv1.CapabilityLeaseRelease}) {
		t.Fatal("handoff did not retain the exact gateway-required bundle")
	}
	if !slices.Equal(standingActionCapabilities("work.claim", authorityv1.CapabilityWorkClaim), []authorityv1.Capability{authorityv1.CapabilityWorkClaim}) {
		t.Fatal("claim gained another capability")
	}
	fixture := newStandingFixture(t)
	fixture.parent.Roles[0].Operations = []string{"work.handoff"}
	if !standingRoleAllows(fixture.parent, fixture.session, "work.handoff") || standingRoleAllows(fixture.parent, fixture.session, "lease.release") {
		t.Fatal("handoff permission widened to independent lease release")
	}
}

func TestStandingReclaimResumesOnlySameCanonicalOwnerAfterChanges(t *testing.T) {
	fixture := newStandingFixture(t)
	fixture.parent.Roles[0].Operations = append(fixture.parent.Roles[0].Operations, "work.reclaim")
	fixture.parent.Roles = append(fixture.parent.Roles, StandingRole{Bead: "M3-P001", PrincipalID: "synthetic-qa", ProfileID: "qa", Class: "qa", Operations: []string{"review.record"}})
	work := standingCopy(fixture.backend.work)
	work.DisplayID, work.NativeStatus, work.LifecycleState = "P-001", "open", authorityv1.LifecycleBacklog
	work.GoalIDs, work.ProductDecisionIDs = []string{"G-001"}, []string{"PD-002"}
	work.ScenarioIDs, work.VerificationOrder = []string{"F-003-S1"}, []string{"qa", "security-reviewer", "delivery-orchestrator"}
	work.Labels = []authorityv1.Label{authorityv1.LabelPublicAccepted}
	work.Version = authorityv1.WorkVersion{AuthorityGeneration: "synthetic-authority", IssueIncarnation: "synthetic-incarnation", IssueMutationSequence: 1, DependencyGraphRevision: 1}
	work.Integrity = authorityv1.IntegrityDigests{Lineage: strings.Repeat("1", 64), DependencyOutcomes: strings.Repeat("2", 64), Blockers: strings.Repeat("3", 64), ExclusivePaths: strings.Repeat("4", 64)}
	canonical := &standingDeliveryWork{work: work}
	leases := &standingDeliveryLeases{sagas: map[string]gateway.ClaimSaga{}, now: func() time.Time { return fixture.now }}
	events := &standingEventFixture{}
	service, err := gateway.NewWithClaims(canonical, leases, events, func() time.Time { return fixture.now })
	if err != nil {
		t.Fatal(err)
	}
	fixture.gate.gateway = &lazyReadGateway{service: service, mutations: true}
	initial := standingCopy(fixture.request)
	initial.Mutation.Claim.ExpectedVersion, initial.Mutation.Claim.ExpectedIntegrity = work.Version, work.Integrity
	reclaim := func(id, canonicalAttempt string) {
		fixture.request = StandingRequest{ID: id, Operation: "work.reclaim", IssuedAt: fixture.now.Add(-time.Second), ExpiresAt: fixture.now.Add(time.Minute), Reclaim: &gateway.ClaimReconciliationRequest{ClaimRequest: authorityv1.ClaimRequest{BeadID: work.BeadID, ExpectedVersion: canonical.work.Version, ExpectedIntegrity: canonical.work.Integrity, AttemptID: id + "-attempt", BaseSHA: fixture.activation.value.SourceBaseSHA, ExclusivePaths: slices.Clone(work.ExclusivePaths), Capability: authorityv1.CapabilityTicketDelivery, IdempotencyKey: id, TraceRef: id}, CanonicalClaimAttemptID: canonicalAttempt}}
	}
	reclaim("synthetic-backlog-reclaim", "synthetic-attempt")
	if _, err := fixture.execute(t); !errors.Is(err, ErrAuthorization) || canonical.claimCalls != 0 || leases.epoch != 0 {
		t.Fatal("reclaim bootstrapped backlog work")
	}
	fixture.request = initial
	result, err := fixture.execute(t)
	if err != nil {
		t.Fatalf("initial claim: %v", err)
	}
	claim := result.(authorityv1.ClaimResponse)
	reclaim("synthetic-active-reclaim", claim.Work.ClaimAttemptID)
	if _, err := fixture.execute(t); !errors.Is(err, ErrGateway) || leases.epoch != 1 {
		t.Fatal("reclaim replaced an active lease")
	}
	fixture.request = StandingRequest{ID: "synthetic-rework-handoff", Operation: "work.handoff", IssuedAt: fixture.now.Add(-time.Second), ExpiresAt: fixture.now.Add(time.Minute), Handoff: &authorityv1.HandoffRequest{BeadID: work.BeadID, ExpectedVersion: canonical.work.Version, ExpectedIntegrity: canonical.work.Integrity, Fence: standingFence(claim.Lease), HeadSHA: strings.Repeat("c", 40), EvidenceRefs: []string{"synthetic-rework-evidence"}, NextProfileID: "qa", IdempotencyKey: "synthetic-rework-handoff", TraceRef: "synthetic-rework-handoff"}}
	if _, err := fixture.execute(t); err != nil {
		t.Fatalf("handoff: %v", err)
	}
	fixture.session.PrincipalID, fixture.session.ProfileID, fixture.session.Class = "synthetic-qa", "qa", "qa"
	fixture.request = StandingRequest{ID: "synthetic-qa-changes", Operation: "review.record", IssuedAt: fixture.now.Add(-time.Second), ExpiresAt: fixture.now.Add(time.Minute), Review: &authorityv1.ReviewVerdictRequest{BeadID: work.BeadID, ExpectedVersion: canonical.work.Version, ExpectedIntegrity: canonical.work.Integrity, HeadSHA: strings.Repeat("c", 40), Verdict: authorityv1.ReviewChangesRequested, EvidenceRefs: []string{"synthetic-qa-changes"}, IdempotencyKey: "synthetic-qa-changes", TraceRef: "synthetic-qa-changes"}}
	if _, err := fixture.execute(t); err != nil {
		t.Fatalf("requested changes: %v; events=%v", err, events.events)
	}
	fixture.session.PrincipalID, fixture.session.ProfileID, fixture.session.Class = "synthetic-implementer", "platform-engineer", "implementation"
	reclaim("synthetic-wrong-provenance", "synthetic-other-canonical-attempt")
	if _, err := fixture.execute(t); !errors.Is(err, ErrAuthorization) || leases.epoch != 1 {
		t.Fatal("reclaim changed canonical provenance")
	}
	owner := canonical.work.Assignee
	canonical.work.Assignee = "synthetic-other-owner"
	reclaim("synthetic-wrong-owner", claim.Work.ClaimAttemptID)
	if _, err := fixture.execute(t); !errors.Is(err, ErrAuthorization) || leases.epoch != 1 {
		t.Fatal("reclaim transferred ownership")
	}
	canonical.work.Assignee = owner
	reclaim("synthetic-correction-lease", claim.Work.ClaimAttemptID)
	before := standingCopy(canonical.work)
	result, err = fixture.execute(t)
	if err != nil {
		t.Fatalf("same-Bead correction lease: %v; events=%v", err, events.events)
	}
	resumed, ok := result.(authorityv1.ClaimResponse)
	if !ok || !resumed.Lease.Active || resumed.Lease.LeaseEpoch != 2 || resumed.Lease.CanonicalClaimAttemptID != claim.Work.ClaimAttemptID || resumed.Lease.AttemptID == claim.Lease.AttemptID || !reflect.DeepEqual(before, canonical.work) || canonical.claimCalls != 1 {
		t.Fatal("correction did not retain canonical claim and issue a newer bounded lease")
	}
}

func TestStandingRealGatewayRejectsStaleAndDependencyBlockedClaims(t *testing.T) {
	for _, failure := range []string{"stale-version", "blocked-dependency"} {
		t.Run(failure, func(t *testing.T) {
			fixture := newStandingFixture(t)
			work := standingCopy(fixture.backend.work)
			work.DisplayID, work.NativeStatus, work.LifecycleState = "P-001", "open", authorityv1.LifecycleBacklog
			work.GoalIDs, work.ProductDecisionIDs = []string{"G-001"}, []string{"PD-002"}
			work.ScenarioIDs, work.VerificationOrder = []string{"F-003-S1"}, []string{"qa", "security-reviewer", "delivery-orchestrator"}
			work.Labels = []authorityv1.Label{authorityv1.LabelPublicAccepted}
			work.Version = authorityv1.WorkVersion{AuthorityGeneration: "synthetic-authority", IssueIncarnation: "synthetic-incarnation", IssueMutationSequence: 1, DependencyGraphRevision: 1}
			work.Integrity = authorityv1.IntegrityDigests{Lineage: strings.Repeat("1", 64), DependencyOutcomes: strings.Repeat("2", 64), Blockers: strings.Repeat("3", 64), ExclusivePaths: strings.Repeat("4", 64)}
			fixture.request.Mutation.Claim.ExpectedVersion, fixture.request.Mutation.Claim.ExpectedIntegrity = work.Version, work.Integrity
			if failure == "stale-version" {
				fixture.request.Mutation.Claim.ExpectedVersion.IssueMutationSequence++
			} else {
				work.Dependencies = []authorityv1.Dependency{{BeadID: "M3-W001", LifecycleState: authorityv1.LifecycleInProgress}}
			}
			canonical := &standingDeliveryWork{work: work}
			leases := &standingDeliveryLeases{sagas: map[string]gateway.ClaimSaga{}, now: func() time.Time { return fixture.now }}
			service, err := gateway.NewWithClaims(canonical, leases, &standingEventFixture{}, func() time.Time { return fixture.now })
			if err != nil {
				t.Fatal(err)
			}
			fixture.gate.gateway = &lazyReadGateway{service: service, mutations: true}
			if _, err := fixture.execute(t); !errors.Is(err, ErrGateway) || canonical.claimCalls != 0 || leases.epoch != 0 || canonical.work.LifecycleState != authorityv1.LifecycleBacklog {
				t.Fatalf("%s changed canonical work or issued a lease: %v", failure, err)
			}
		})
	}
}

type standingActivationFixture struct {
	value       StandingActivation
	unavailable bool
}

func (source *standingActivationFixture) Current(context.Context) (StandingActivation, error) {
	if source.unavailable {
		return StandingActivation{}, ErrAuthorization
	}
	return source.value, nil
}

type standingReplayFixture struct {
	used   map[string]bool
	hook   func()
	failed bool
}

func (store *standingReplayFixture) Consume(_ context.Context, key string) error {
	if store.failed || store.used[key] {
		return ErrReplay
	}
	store.used[key] = true
	if store.hook != nil {
		store.hook()
	}
	return nil
}

type standingGatewayFixture struct {
	work       authorityv1.WorkItem
	ready      authorityv1.ReadyResponse
	reads      int
	dispatches int
	principal  authorityv1.Principal
	deadline   time.Time
	onRead     func()
	fail       bool
}

func (backend *standingGatewayFixture) GetWork(_ context.Context, principal authorityv1.Principal, _ authorityv1.GetWorkRequest) (authorityv1.WorkItem, error) {
	backend.reads++
	backend.principal = principal
	if backend.onRead != nil {
		backend.onRead()
	}
	return backend.work, nil
}
func (backend *standingGatewayFixture) Ready(_ context.Context, principal authorityv1.Principal, _ authorityv1.ReadyRequest) (authorityv1.ReadyResponse, error) {
	backend.reads++
	backend.principal = principal
	return backend.ready, nil
}
func (backend *standingGatewayFixture) DispatchStanding(ctx context.Context, principal authorityv1.Principal, _ StandingRequest) (any, error) {
	backend.dispatches++
	backend.principal = principal
	backend.deadline, _ = ctx.Deadline()
	if backend.fail {
		return nil, errors.New("synthetic backend failure")
	}
	return "synthetic-dispatched", nil
}

func newStandingFixture(t *testing.T) *standingFixture {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)
	runtime := RuntimeBinding{ProfileSHA256: strings.Repeat("a", 64), TenantID: "synthetic-tenant", ProjectID: "synthetic-project",
		BaseSHA: strings.Repeat("a", 40), TreeSHA: strings.Repeat("b", 40), WorkspaceDigest: strings.Repeat("b", 64),
		NativeBinarySHA256: strings.Repeat("c", 64), FenceGeneration: "synthetic-generation"}
	scope := StandingScope{Bead: "M3-P001", FeatureID: "F-003", ContractSHA256: strings.Repeat("d", 64), Paths: []string{"internal/platform/fixture.go"}}
	parent := StandingRuntime{SchemaVersion: 2, Kind: "MARS3StandingDeliveryRuntime", ID: "synthetic-standing", Repository: "greaveselliott/MARS-3",
		Runtime: runtime, PlanSHA256: strings.Repeat("e", 64), IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(24 * time.Hour), Scopes: []StandingScope{scope},
		Roles: []StandingRole{{Bead: scope.Bead, PrincipalID: "synthetic-implementer", ProfileID: "platform-engineer", Class: "implementation",
			Operations: []string{"work.get", "work.claim", "lease.renew", "lease.release", "effect.validate", "work.handoff"}}}}
	session := StandingSession{SchemaVersion: 2, Kind: "MARS3StandingDeliverySession", ID: "synthetic-session", Bead: scope.Bead,
		PrincipalID: "synthetic-implementer", ProfileID: "platform-engineer", Class: "implementation", IssuedAt: now.Add(-30 * time.Second), ExpiresAt: now.Add(30 * time.Minute)}
	// The source base deliberately differs from the accepted operator release.
	sourceBase := strings.Repeat("f", 40)
	request := StandingRequest{ID: "synthetic-operation", Operation: "work.claim", IssuedAt: now.Add(-time.Second), ExpiresAt: now.Add(time.Minute),
		Mutation: &MutationRequest{Operation: "work.claim", Claim: &authorityv1.ClaimRequest{BeadID: scope.Bead, BaseSHA: sourceBase,
			ExclusivePaths: []string{"internal/platform/fixture.go"}, Capability: authorityv1.CapabilityTicketDelivery,
			AttemptID: "synthetic-attempt", IdempotencyKey: "synthetic-claim", TraceRef: "synthetic-trace"}}}
	activation := &standingActivationFixture{value: StandingActivation{SchemaVersion: 2, Kind: "MARS3StandingDeliveryActivation", Runtime: runtime,
		PlanSHA256: parent.PlanSHA256, SourceBaseSHA: sourceBase, IssuedAt: now.Add(-20 * time.Second), ExpiresAt: parent.ExpiresAt, Contracts: []StandingScope{scope}}}
	backend := &standingGatewayFixture{work: authorityv1.WorkItem{TenantID: runtime.TenantID, ProjectID: runtime.ProjectID, BeadID: scope.Bead,
		FeatureID: scope.FeatureID, ExclusivePaths: []string{"internal/platform/fixture.go"}}}
	replays := &standingReplayFixture{used: map[string]bool{}}
	gate, err := NewStandingGate(runtime, activation, backend, replays)
	if err != nil {
		t.Fatal(err)
	}
	fixture := &standingFixture{gate: gate, parent: parent, session: session, request: request, activation: activation, backend: backend, replays: replays, now: now}
	gate.now = func() time.Time { return fixture.now }
	// Admission sequencing fixtures do not claim real signature qualification.
	gate.verifyRuntime = func([]byte, []byte) error { return nil }
	gate.verifySession = func([]byte, []byte) error { return nil }
	return fixture
}

func standingEncode(t *testing.T, value any) []byte {
	t.Helper()
	document, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return document
}
func (fixture *standingFixture) execute(t *testing.T) (any, error) {
	t.Helper()
	document := standingEncode(t, fixture.parent)
	fixture.session.DelegationSHA256 = standingDigest(document)
	fixture.activation.value.DelegationSHA256 = fixture.session.DelegationSHA256
	return fixture.gate.Execute(context.Background(), document, []byte("synthetic-signature"), standingEncode(t, fixture.session), []byte("synthetic-signature"), standingEncode(t, fixture.request))
}

func TestStandingDispatchDerivesOneCapabilityAndConsumesExactIdentity(t *testing.T) {
	fixture := newStandingFixture(t)
	result, err := fixture.execute(t)
	if err != nil || result != "synthetic-dispatched" || fixture.backend.dispatches != 1 || fixture.backend.reads != 1 {
		t.Fatalf("dispatch = %v, %v", result, err)
	}
	principal := fixture.backend.principal
	if principal.PrincipalID != fixture.session.PrincipalID || principal.ProfileID != fixture.session.ProfileID ||
		len(principal.Capabilities) != 1 || principal.Capabilities[0] != authorityv1.CapabilityWorkClaim || len(principal.Labels) != 2 {
		t.Fatal("dispatch gained identity/capability/label authority")
	}
	if fixture.backend.deadline.IsZero() || fixture.backend.deadline.After(time.Now().Add(61*time.Second)) {
		t.Fatal("dispatch not bounded by request expiry")
	}
	// Reusing the ID with changed bytes must not produce a new replay identity.
	fixture.request.Mutation.Claim.IdempotencyKey = "synthetic-other-claim"
	if _, err := fixture.execute(t); !errors.Is(err, ErrReplay) || fixture.backend.dispatches != 1 {
		t.Fatalf("changed-payload replay = %v", err)
	}
}

func TestStandingAdmissionDeniesScopeIdentityAndWindowDrift(t *testing.T) {
	tests := []struct {
		name   string
		change func(*standingFixture)
	}{
		{"v1-kind", func(f *standingFixture) { f.parent.Kind = "MARS3StandingDeliveryContract" }},
		{"v1-schema", func(f *standingFixture) { f.parent.SchemaVersion = 1 }},
		{"other-runtime", func(f *standingFixture) { f.parent.Runtime.ProjectID = "other-project" }},
		{"expired-parent", func(f *standingFixture) { f.parent.ExpiresAt = f.now }},
		{"overlong-parent", func(f *standingFixture) { f.parent.ExpiresAt = f.parent.IssuedAt.Add(31 * 24 * time.Hour) }},
		{"unsafe-path", func(f *standingFixture) { f.parent.Scopes[0].Paths = []string{"../escape.go"} }},
		{"wildcard-path", func(f *standingFixture) { f.parent.Scopes[0].Paths = []string{"internal/*"} }},
		{"historical-bead", func(f *standingFixture) { f.parent.Scopes[0].Bead = "M3-W001" }},
		{"unknown-operation", func(f *standingFixture) { f.request.Operation = "production.release" }},
		{"role-escalation", func(f *standingFixture) {
			f.parent.Roles[0].Operations = append(f.parent.Roles[0].Operations, "review.record")
		}},
		{"impersonated-session", func(f *standingFixture) { f.session.PrincipalID = "synthetic-attacker" }},
		{"reviewer-session", func(f *standingFixture) { f.session.Class = "qa" }},
		{"session-expired", func(f *standingFixture) { f.session.ExpiresAt = f.now }},
		{"session-overlong", func(f *standingFixture) { f.session.ExpiresAt = f.session.IssuedAt.Add(2 * time.Hour) }},
		{"request-expired", func(f *standingFixture) { f.request.ExpiresAt = f.now }},
		{"request-future", func(f *standingFixture) { f.request.IssuedAt = f.now.Add(time.Second) }},
		{"request-overlong", func(f *standingFixture) { f.request.ExpiresAt = f.request.IssuedAt.Add(6 * time.Minute) }},
		{"mixed-payloads", func(f *standingFixture) {
			f.request.Read = &ReadRequest{Operation: "work.get", BeadID: "M3-P001", TraceRef: "synthetic-trace"}
		}},
		{"other-bead", func(f *standingFixture) { f.request.Mutation.Claim.BeadID = "M3-T001" }},
		{"claim-path-drift", func(f *standingFixture) { f.request.Mutation.Claim.ExclusivePaths = []string{"other.go"} }},
		{"claim-base-drift", func(f *standingFixture) { f.request.Mutation.Claim.BaseSHA = f.parent.Runtime.BaseSHA }},
		{"claim-capability-drift", func(f *standingFixture) { f.request.Mutation.Claim.Capability = authorityv1.CapabilityReviewRecord }},
		{"revoked-activation", func(f *standingFixture) { f.activation.value.Revoked = true }},
		{"missing-activation", func(f *standingFixture) { f.activation.unavailable = true }},
		{"activation-expired", func(f *standingFixture) { f.activation.value.ExpiresAt = f.now }},
		{"activation-plan-drift", func(f *standingFixture) { f.activation.value.PlanSHA256 = strings.Repeat("0", 64) }},
		{"activation-runtime-drift", func(f *standingFixture) { f.activation.value.Runtime.FenceGeneration = "other-generation" }},
		{"activation-contract-drift", func(f *standingFixture) { f.activation.value.Contracts[0].ContractSHA256 = strings.Repeat("0", 64) }},
		{"canonical-path-drift", func(f *standingFixture) { f.backend.work.ExclusivePaths = []string{"other.go"} }},
		{"canonical-feature-drift", func(f *standingFixture) { f.backend.work.FeatureID = "F-OTHER" }},
		{"canonical-project-drift", func(f *standingFixture) { f.backend.work.ProjectID = "other-project" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newStandingFixture(t)
			test.change(fixture)
			if _, err := fixture.execute(t); err == nil || fixture.backend.dispatches != 0 {
				t.Fatalf("invalid request dispatched: %v", err)
			}
		})
	}
}

func TestStandingChecksRevocationAndExpiryAfterConsumptionAndRead(t *testing.T) {
	for _, stage := range []string{"consumption-revocation", "consumption-expiry", "read-revocation"} {
		t.Run(stage, func(t *testing.T) {
			fixture := newStandingFixture(t)
			switch stage {
			case "consumption-revocation":
				fixture.replays.hook = func() { fixture.activation.value.Revoked = true }
			case "consumption-expiry":
				fixture.replays.hook = func() { fixture.now = fixture.request.ExpiresAt }
			case "read-revocation":
				fixture.backend.onRead = func() { fixture.activation.value.Revoked = true }
			}
			if _, err := fixture.execute(t); err == nil || fixture.backend.dispatches != 0 || len(fixture.replays.used) != 1 {
				t.Fatalf("late denial = %v", err)
			}
		})
	}
}

// Sample real time, then delay returning it until after the signed expiry.
// This models preemption without inventing a clock offset from context timers.
func TestStandingAbsoluteDeadlinesDenySchedulingGap(t *testing.T) {
	for _, stage := range []string{"initial", "shortened-after-read"} {
		t.Run(stage, func(t *testing.T) {
			fixture := newStandingFixture(t)
			afterRead, samples, delayed := false, 0, false
			if stage == "initial" {
				fixture.activation.value.ExpiresAt = time.Now().UTC().Add(time.Second)
			} else {
				fixture.backend.onRead = func() {
					fixture.activation.value.ExpiresAt = time.Now().UTC().Add(time.Second)
					afterRead = true
				}
			}
			fixture.gate.now = func() time.Time {
				sampled := time.Now().UTC()
				if stage == "initial" || afterRead {
					samples++
					// Initial admission: root, activation, remaining check.
					// After read: request check, replacement activation check.
					target := 3
					if stage != "initial" {
						target = 2
					}
					if samples == target {
						if !sampled.Before(fixture.activation.value.ExpiresAt) {
							t.Fatal("fixture expired before the scheduling gap")
						}
						time.Sleep(time.Until(fixture.activation.value.ExpiresAt) + 25*time.Millisecond)
						delayed = true
					}
				}
				return sampled
			}
			result, err := fixture.execute(t)
			if !delayed || !errors.Is(err, ErrAuthorization) || result != nil || fixture.backend.dispatches != 0 {
				t.Fatalf("scheduling gap: delayed=%v result=%v error=%v dispatches=%d", delayed, result, err, fixture.backend.dispatches)
			}
			if stage == "initial" && fixture.backend.reads != 0 {
				t.Fatal("expired initial deadline reached the gateway")
			}
		})
	}
}

func TestStandingAbsoluteDeadlinePreservesEarlierCallerDeadline(t *testing.T) {
	fixture := newStandingFixture(t)
	document := standingEncode(t, fixture.parent)
	fixture.session.DelegationSHA256 = standingDigest(document)
	fixture.activation.value.DelegationSHA256 = fixture.session.DelegationSHA256
	deadline := time.Now().Add(10 * time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	_, err := fixture.gate.Execute(ctx, document, []byte("synthetic-signature"), standingEncode(t, fixture.session), []byte("synthetic-signature"), standingEncode(t, fixture.request))
	if err != nil || fixture.backend.dispatches != 1 || !fixture.backend.deadline.Equal(deadline) {
		t.Fatalf("caller deadline extended: error=%v deadline=%v", err, fixture.backend.deadline)
	}
}

func TestStandingBackendFailureAndUncertainPersistenceDoNotRetryEffects(t *testing.T) {
	fixture := newStandingFixture(t)
	fixture.backend.fail = true
	if _, err := fixture.execute(t); !errors.Is(err, ErrGateway) {
		t.Fatalf("backend failure = %v", err)
	}
	if _, err := fixture.execute(t); !errors.Is(err, ErrReplay) || fixture.backend.dispatches != 1 {
		t.Fatalf("backend replay = %v", err)
	}
	fixture = newStandingFixture(t)
	fixture.replays.failed = true
	if _, err := fixture.execute(t); !errors.Is(err, ErrReplay) || fixture.backend.reads != 0 || fixture.backend.dispatches != 0 {
		t.Fatalf("persistence failure = %v", err)
	}
}

func TestStandingCanonicalEncodingAndProductionSignaturesFailClosed(t *testing.T) {
	fixture := newStandingFixture(t)
	document := standingEncode(t, fixture.parent)
	session := standingEncode(t, fixture.session)
	request := standingEncode(t, fixture.request)
	for _, altered := range [][]byte{append(append([]byte{}, document...), '\n'), []byte(`{"schema_version":2,"schema_version":1}`), []byte(`{"unknown":"value"}`)} {
		if _, err := fixture.gate.Execute(context.Background(), altered, []byte("signature"), session, []byte("signature"), request); err == nil {
			t.Fatal("noncanonical parent admitted")
		}
	}
	production, err := NewStandingGate(fixture.gate.runtime, fixture.activation, fixture.backend, fixture.replays)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := production.Execute(context.Background(), document, []byte("forged"), session, []byte("forged"), request); err == nil || fixture.backend.reads != 0 {
		t.Fatal("production verifier accepted forgery")
	}
}

func TestStandingReadyDoesNotExposeOtherBeads(t *testing.T) {
	fixture := newStandingFixture(t)
	fixture.parent.Roles[0].Class, fixture.session.Class = "orchestrator", "orchestrator"
	fixture.parent.Roles[0].ProfileID, fixture.session.ProfileID = "delivery-orchestrator", "delivery-orchestrator"
	fixture.parent.Roles[0].Operations = []string{"work.ready"}
	fixture.request.Operation = "work.ready"
	fixture.request.Mutation = nil
	fixture.request.Read = &ReadRequest{Operation: "work.ready", TraceRef: "synthetic-ready"}
	other := fixture.backend.work
	other.BeadID = "M3-T001"
	fixture.backend.ready = authorityv1.ReadyResponse{Items: []authorityv1.ReadyItem{{WorkItem: fixture.backend.work, Ready: true}, {WorkItem: other, Ready: true}}}
	result, err := fixture.execute(t)
	if err != nil {
		t.Fatal(err)
	}
	ready, ok := result.(authorityv1.ReadyResponse)
	if !ok || len(ready.Items) != 1 || ready.Items[0].BeadID != "M3-P001" || fixture.backend.dispatches != 0 {
		t.Fatal("ready leaked out-of-scope work")
	}
}

func TestStandingRoleClassesRemainDisjoint(t *testing.T) {
	for _, identity := range []string{"principal", "profile"} {
		t.Run(identity, func(t *testing.T) {
			fixture := newStandingFixture(t)
			role := StandingRole{Bead: "M3-P001", PrincipalID: "synthetic-reviewer", ProfileID: "qa", Class: "qa", Operations: []string{"review.record"}}
			if identity == "principal" {
				role.PrincipalID = fixture.session.PrincipalID
			} else {
				role.ProfileID = fixture.session.ProfileID
			}
			fixture.parent.Roles = append(fixture.parent.Roles, role)
			if _, err := fixture.execute(t); err == nil || fixture.backend.dispatches != 0 {
				t.Fatal("one identity acquired implementation and review roles")
			}
		})
	}
}

// Regression for independent QA's production-gateway reproduction. A signed
// parent is still invalid when its declared class masks canonical profiles.
func TestStandingCanonicalReviewerProfilesCannotShareOneQAClassPrincipal(t *testing.T) {
	fixture := newStandingFixture(t)
	fixture.parent.Roles = append(fixture.parent.Roles,
		StandingRole{Bead: "M3-P001", PrincipalID: "synthetic-qa", ProfileID: "qa", Class: "qa", Operations: []string{"review.record"}},
		StandingRole{Bead: "M3-P001", PrincipalID: "synthetic-qa", ProfileID: "security-reviewer", Class: "qa", Operations: []string{"review.record"}},
		StandingRole{Bead: "M3-P001", PrincipalID: "synthetic-orchestrator", ProfileID: "delivery-orchestrator", Class: "orchestrator", Operations: []string{"work.close"}})
	work := standingCopy(fixture.backend.work)
	work.DisplayID, work.NativeStatus, work.LifecycleState = "P-001", "open", authorityv1.LifecycleBacklog
	work.GoalIDs, work.ProductDecisionIDs = []string{"G-001"}, []string{"PD-002"}
	work.ScenarioIDs, work.VerificationOrder = []string{"F-003-S1"}, []string{"qa", "security-reviewer", "delivery-orchestrator"}
	work.Labels = []authorityv1.Label{authorityv1.LabelPublicAccepted}
	work.Version = authorityv1.WorkVersion{AuthorityGeneration: "synthetic-authority", IssueIncarnation: "synthetic-incarnation", IssueMutationSequence: 1, DependencyGraphRevision: 1}
	work.Integrity = authorityv1.IntegrityDigests{Lineage: strings.Repeat("1", 64), DependencyOutcomes: strings.Repeat("2", 64), Blockers: strings.Repeat("3", 64), ExclusivePaths: strings.Repeat("4", 64)}
	fixture.request.Mutation.Claim.ExpectedVersion, fixture.request.Mutation.Claim.ExpectedIntegrity = work.Version, work.Integrity
	canonical := &standingDeliveryWork{work: work}
	leases := &standingDeliveryLeases{sagas: map[string]gateway.ClaimSaga{}, now: func() time.Time { return fixture.now }}
	events := &standingEventFixture{}
	service, err := gateway.NewWithClaims(canonical, leases, events, func() time.Time { return fixture.now })
	if err != nil {
		t.Fatal(err)
	}
	fixture.gate.gateway = &lazyReadGateway{service: service, mutations: true}
	if _, err := fixture.execute(t); !errors.Is(err, ErrAuthorization) || canonical.claimCalls != 0 || canonical.lifecycleCalls != 0 || leases.epoch != 0 || len(events.events) != 0 || len(fixture.replays.used) != 0 {
		t.Fatalf("one principal reached the production gateway with two canonical review profiles: %v", err)
	}
}

func TestStandingRoleClassIsDerivedFromCanonicalProfile(t *testing.T) {
	for _, test := range []struct {
		profile, class string
		valid          bool
	}{
		{"qa", "qa", true}, {"security-reviewer", "security", true}, {"delivery-orchestrator", "orchestrator", true}, {"platform-engineer", "implementation", true},
		{"qa", "security", false}, {"qa", "implementation", false}, {"security-reviewer", "qa", false}, {"security-reviewer", "implementation", false},
		{"delivery-orchestrator", "qa", false}, {"delivery-orchestrator", "implementation", false}, {"platform-engineer", "qa", false}, {"qa-engineer", "qa", false},
	} {
		if standingCanonicalRoleMatches(StandingRole{ProfileID: test.profile, Class: test.class}) != test.valid {
			t.Fatalf("profile %q class %q mapping drifted", test.profile, test.class)
		}
	}
	fixture := newStandingFixture(t)
	fixture.parent.Roles = append(fixture.parent.Roles,
		StandingRole{Bead: "M3-P001", PrincipalID: "synthetic-shared-reviewer", ProfileID: "qa", Class: "qa", Operations: []string{"review.record"}},
		StandingRole{Bead: "M3-P001", PrincipalID: "synthetic-shared-reviewer", ProfileID: "security-reviewer", Class: "security", Operations: []string{"review.record"}})
	if _, err := fixture.execute(t); !errors.Is(err, ErrAuthorization) || fixture.backend.reads != 0 {
		t.Fatal("canonical class names allowed one principal to span independent reviewers")
	}
}

func TestStandingFileReplaySurvivesReopen(t *testing.T) {
	fixture := newStandingFixture(t)
	directory := t.TempDir()
	if err := os.Chmod(directory, 0700); err != nil {
		t.Fatal(err)
	}
	store, err := NewFileReplayStore(directory)
	if err != nil {
		t.Fatal(err)
	}
	fixture.gate.replays = store
	if _, err := fixture.execute(t); err != nil {
		store.Close()
		t.Fatal(err)
	}
	store.Close()
	store, err = NewFileReplayStore(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	fixture.gate.replays = store
	if _, err := fixture.execute(t); !errors.Is(err, ErrReplay) || fixture.backend.dispatches != 1 {
		t.Fatalf("reopened replay = %v", err)
	}
}

func TestStandingPayloadTargetsCoverLifecycleAndLeaseRoutes(t *testing.T) {
	fixture := newStandingFixture(t)
	fence := authorityv1.FencingTuple{TenantID: fixture.gate.runtime.TenantID, ProjectID: fixture.gate.runtime.ProjectID, BeadID: "M3-P001",
		BaseSHA: fixture.activation.value.SourceBaseSHA, Capability: authorityv1.CapabilityTicketDelivery, ExclusivePaths: []string{"internal/platform/fixture.go"}}
	requests := []StandingRequest{
		{Operation: "lease.renew", Mutation: &MutationRequest{Operation: "lease.renew", Renew: &authorityv1.RenewLeaseRequest{Fence: fence, TraceRef: "synthetic-trace"}}},
		{Operation: "lease.release", Mutation: &MutationRequest{Operation: "lease.release", Release: &authorityv1.ReleaseLeaseRequest{Fence: fence, TraceRef: "synthetic-trace"}}},
		{Operation: "effect.validate", Mutation: &MutationRequest{Operation: "effect.validate", Effect: &authorityv1.EffectValidationRequest{Fence: fence, Path: "internal/platform/fixture.go", TraceRef: "synthetic-trace"}}},
		{Operation: "work.handoff", Handoff: &authorityv1.HandoffRequest{BeadID: "M3-P001", Fence: fence, TraceRef: "synthetic-trace"}},
		{Operation: "review.record", Review: &authorityv1.ReviewVerdictRequest{BeadID: "M3-P001", TraceRef: "synthetic-trace"}},
		{Operation: "run.disposition", Run: &authorityv1.RunDispositionRequest{BeadID: "M3-P001", TraceRef: "synthetic-trace"}},
		{Operation: "work.reconcile", Reconcile: &authorityv1.ReconciliationRequest{BeadID: "M3-P001", TraceRef: "synthetic-trace"}},
		{Operation: "work.close", Close: &authorityv1.TerminalTransitionRequest{BeadID: "M3-P001", TraceRef: "synthetic-trace"}},
	}
	for _, request := range requests {
		t.Run(request.Operation, func(t *testing.T) {
			bead, trace, capability, err := request.target()
			if err != nil || bead != "M3-P001" || trace != "synthetic-trace" || capability == "" || !request.scopeMatches(fixture.parent.Scopes[0], fixture.gate.runtime, fence.BaseSHA) {
				t.Fatal("typed lifecycle/lease route not admitted")
			}
		})
	}
}
