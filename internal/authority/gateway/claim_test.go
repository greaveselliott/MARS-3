/*
FactoryDocSync:
docs:
- docs/features/F-002-work-authority.md
- docs/design-docs/ADR-001-git-beads-authority.md
- docs/code-documentation-map.md
*/

package gateway

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	authorityv1 "github.com/greaveselliott/MARS-3/api/authority/v1"
)

type memoryClaimStore struct {
	mu               sync.Mutex
	item             authorityv1.WorkItem
	claimCalls       int
	invalidPostimage bool
	getBarrier       chan struct{}
	getCount         int
}

func (store *memoryClaimStore) Get(_ context.Context, tenantID, projectID, beadID string) (authorityv1.WorkItem, error) {
	store.mu.Lock()
	if store.item.TenantID != tenantID || store.item.ProjectID != projectID || store.item.BeadID != beadID {
		store.mu.Unlock()
		return authorityv1.WorkItem{}, ErrWorkNotFound
	}
	item := cloneWork(store.item)
	if store.getBarrier != nil {
		store.getCount++
		if store.getCount == 2 {
			close(store.getBarrier)
		}
		barrier := store.getBarrier
		store.mu.Unlock()
		<-barrier
		return item, nil
	}
	store.mu.Unlock()
	return item, nil
}

func (store *memoryClaimStore) List(_ context.Context, tenantID, projectID string) ([]authorityv1.WorkItem, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.item.TenantID != tenantID || store.item.ProjectID != projectID {
		return nil, nil
	}
	return []authorityv1.WorkItem{cloneWork(store.item)}, nil
}

func (store *memoryClaimStore) CompareAndSwapClaim(_ context.Context, mutation ClaimMutation) (authorityv1.WorkItem, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.claimCalls++
	if store.item.TenantID != mutation.TenantID || store.item.ProjectID != mutation.ProjectID || store.item.BeadID != mutation.BeadID || store.item.LifecycleState != authorityv1.LifecycleBacklog || store.item.Version != mutation.ExpectedVersion || store.item.Integrity != mutation.ExpectedIntegrity {
		return authorityv1.WorkItem{}, ErrStaleWorkVersion
	}
	store.item.LifecycleState = authorityv1.LifecycleInProgress
	store.item.NativeStatus = "in_progress"
	store.item.Assignee = mutation.Assignee
	store.item.ClaimAttemptID = mutation.AttemptID
	store.item.Version.IssueMutationSequence++
	store.item.Integrity.Lineage = strings.Repeat("b", 64)
	if store.invalidPostimage {
		store.item.ClaimAttemptID = "wrong-attempt"
	}
	return cloneWork(store.item), nil
}

type memorySagaStore struct {
	mu       sync.Mutex
	sagas    map[string]ClaimSaga
	epoch    uint64
	now      time.Time
	issueErr error
}

func newMemorySagaStore(now time.Time) *memorySagaStore {
	return &memorySagaStore{sagas: make(map[string]ClaimSaga), now: now}
}

func (store *memorySagaStore) Lookup(_ context.Context, key string) (ClaimSaga, bool, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	saga, found := store.sagas[key]
	return cloneSaga(saga), found, nil
}

func (store *memorySagaStore) Begin(_ context.Context, intent ClaimIntent) (ClaimSaga, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if saga, found := store.sagas[intent.IdempotencyKey]; found {
		if saga.RequestDigest != intent.RequestDigest {
			return ClaimSaga{}, ErrIdempotencyConflict
		}
		return cloneSaga(saga), nil
	}
	saga := ClaimSaga{RequestDigest: intent.RequestDigest, Phase: claimPhaseIntent}
	store.sagas[intent.IdempotencyKey] = saga
	return saga, nil
}

func (store *memorySagaStore) MarkCanonicalClaimed(_ context.Context, key, digest string, work authorityv1.WorkItem) (ClaimSaga, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	saga, found := store.sagas[key]
	if !found || saga.RequestDigest != digest || saga.Phase != claimPhaseIntent {
		return ClaimSaga{}, ErrIdempotencyConflict
	}
	saga.Phase = claimPhaseCanonical
	saga.Work = cloneWork(work)
	store.sagas[key] = saga
	return cloneSaga(saga), nil
}

func (store *memorySagaStore) IssueLease(_ context.Context, key, digest string, request LeaseRequest) (ClaimSaga, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.issueErr != nil {
		return ClaimSaga{}, store.issueErr
	}
	saga, found := store.sagas[key]
	if !found || saga.RequestDigest != digest || saga.Phase != claimPhaseCanonical {
		return ClaimSaga{}, ErrIdempotencyConflict
	}
	for existingKey, existing := range store.sagas {
		if existingKey != key && existing.Phase == claimPhaseComplete && existing.Lease.Active && existing.Lease.ExpiresAt.After(store.now) && pathsOverlap(existing.Lease.ExclusivePaths, request.ExclusivePaths) {
			return ClaimSaga{}, errors.New("lease conflict")
		}
	}
	store.epoch++
	saga.Phase = claimPhaseComplete
	saga.Lease = authorityv1.CapabilityLease{
		LeaseID: "lease-" + digest[:16], TenantID: request.TenantID, ProjectID: request.ProjectID,
		BeadID: request.BeadID, AttemptID: request.AttemptID, FenceGeneration: "generation-fixture",
		IdempotencyKey: request.IdempotencyKey, LeaseEpoch: store.epoch, ClaimVersion: request.ClaimVersion,
		BaseSHA: request.BaseSHA, Capability: request.Capability,
		ExclusivePaths: append([]string(nil), request.ExclusivePaths...), Labels: append([]authorityv1.Label(nil), request.Labels...), IssuedAt: store.now,
		ExpiresAt: request.MaximumExpiry, Active: true,
	}
	saga.ReceiptRef = "receipt-" + digest[:16]
	store.sagas[key] = saga
	return cloneSaga(saga), nil
}

func TestClaimCASIssuesCapabilityOnlyAfterCanonicalAndLeaseVerification(t *testing.T) {
	now := time.Date(2026, 8, 27, 1, 0, 0, 0, time.UTC)
	item := readyWork("M3-W002")
	store := &memoryClaimStore{item: item}
	sagas := newMemorySagaStore(now)
	events := &fakeEvents{}
	service := mustClaimService(t, store, sagas, events, now)
	request := claimRequest(item, "attempt-001", "idempotency-001")

	response, err := service.Claim(context.Background(), claimant(), request)
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if response.Replayed || response.Work.LifecycleState != authorityv1.LifecycleInProgress || response.Work.ClaimAttemptID != request.AttemptID {
		t.Fatalf("work response = %#v", response)
	}
	if !response.Lease.Active || response.Lease.LeaseEpoch != 1 || response.Lease.FenceGeneration == "" || response.Lease.Capability != authorityv1.CapabilityTicketDelivery {
		t.Fatalf("lease = %#v", response.Lease)
	}
	if response.ReceiptRef == "" || store.claimCalls != 1 {
		t.Fatalf("receipt=%q claimCalls=%d", response.ReceiptRef, store.claimCalls)
	}
	assertClaimEvents(t, events, request)

	replay, err := service.Claim(context.Background(), claimant(), request)
	if err != nil {
		t.Fatalf("Claim replay: %v", err)
	}
	if !replay.Replayed || replay.ReceiptRef != response.ReceiptRef || !reflect.DeepEqual(replay.Lease, response.Lease) || store.claimCalls != 1 {
		t.Fatalf("replay = %#v claimCalls=%d", replay, store.claimCalls)
	}
	if len(events.events) != 3 {
		t.Fatalf("idempotent event count = %d, want 3", len(events.events))
	}
}

func TestClaimConcurrentCASHasExactlyOneWinner(t *testing.T) {
	now := time.Date(2026, 8, 27, 1, 0, 0, 0, time.UTC)
	item := readyWork("M3-W002")
	store := &memoryClaimStore{item: item, getBarrier: make(chan struct{})}
	service := mustClaimService(t, store, newMemorySagaStore(now), &fakeEvents{}, now)
	requests := []authorityv1.ClaimRequest{
		claimRequest(item, "attempt-001", "idempotency-001"),
		claimRequest(item, "attempt-002", "idempotency-002"),
	}

	results := make(chan error, 2)
	for _, request := range requests {
		request := request
		go func() {
			_, err := service.Claim(context.Background(), claimant(), request)
			results <- err
		}()
	}
	var successes, stale int
	for range requests {
		err := <-results
		if err == nil {
			successes++
			continue
		}
		var denial *authorityv1.Denial
		if errors.As(err, &denial) && denial.Code == authorityv1.ErrorStaleVersion {
			stale++
		} else {
			t.Fatalf("loser error = %v, want stale version", err)
		}
	}
	if successes != 1 || stale != 1 || store.claimCalls != 2 {
		t.Fatalf("successes=%d stale=%d CAS calls=%d", successes, stale, store.claimCalls)
	}
}

func TestClaimRecoversCanonicalClaimedSagaWithoutRepeatingCAS(t *testing.T) {
	now := time.Date(2026, 8, 27, 1, 0, 0, 0, time.UTC)
	item := readyWork("M3-W002")
	store := &memoryClaimStore{item: item}
	sagas := newMemorySagaStore(now)
	sagas.issueErr = errors.New("synthetic lease outage")
	service := mustClaimService(t, store, sagas, &fakeEvents{}, now)
	request := claimRequest(item, "attempt-001", "idempotency-001")

	response, err := service.Claim(context.Background(), claimant(), request)
	if response.Lease.Active {
		t.Fatal("claim returned capability after lease failure")
	}
	assertDenial(t, err, authorityv1.ErrorUnknownEffect, ruleClaimUnknown, requiredReconciliation, allowedSagaRead)
	if store.claimCalls != 1 {
		t.Fatalf("claimCalls=%d, want 1", store.claimCalls)
	}

	sagas.mu.Lock()
	sagas.issueErr = nil
	sagas.mu.Unlock()
	recovered, err := service.Claim(context.Background(), claimant(), request)
	if err != nil {
		t.Fatalf("recovery: %v", err)
	}
	if !recovered.Lease.Active || store.claimCalls != 1 {
		t.Fatalf("recovered=%#v claimCalls=%d", recovered, store.claimCalls)
	}
}

func TestClaimRejectsIdempotencyReuseForDifferentInput(t *testing.T) {
	now := time.Date(2026, 8, 27, 1, 0, 0, 0, time.UTC)
	item := readyWork("M3-W002")
	store := &memoryClaimStore{item: item}
	service := mustClaimService(t, store, newMemorySagaStore(now), &fakeEvents{}, now)
	request := claimRequest(item, "attempt-001", "idempotency-001")
	if _, err := service.Claim(context.Background(), claimant(), request); err != nil {
		t.Fatalf("first claim: %v", err)
	}
	request.BaseSHA = strings.Repeat("b", 40)
	_, err := service.Claim(context.Background(), claimant(), request)
	assertDenial(t, err, authorityv1.ErrorPolicyDenied, ruleClaimIdempotency, requiredNewIdempotency, allowedSagaRead)
}

func TestClaimReplayRevalidatesCanonicalPostimage(t *testing.T) {
	now := time.Date(2026, 8, 27, 1, 0, 0, 0, time.UTC)
	item := readyWork("M3-W002")
	store := &memoryClaimStore{item: item}
	service := mustClaimService(t, store, newMemorySagaStore(now), &fakeEvents{}, now)
	request := claimRequest(item, "attempt-001", "idempotency-001")
	if _, err := service.Claim(context.Background(), claimant(), request); err != nil {
		t.Fatalf("first claim: %v", err)
	}
	store.mu.Lock()
	store.item.LifecycleState = authorityv1.LifecycleInReview
	store.item.NativeStatus = "in_review"
	store.item.Version.IssueMutationSequence++
	store.mu.Unlock()

	response, err := service.Claim(context.Background(), claimant(), request)
	if response.Lease.Active {
		t.Fatal("stale replay returned a capability")
	}
	assertDenial(t, err, authorityv1.ErrorUnknownEffect, ruleClaimUnknown, requiredReconciliation, allowedSagaRead)
}

func TestClaimRejectsStalePathsCapabilitiesAndNotReadyWork(t *testing.T) {
	for name, test := range map[string]struct {
		mutate func(*authorityv1.WorkItem, *authorityv1.ClaimRequest)
		code   authorityv1.ErrorCode
		rule   string
	}{
		"stale-version": {
			mutate: func(_ *authorityv1.WorkItem, request *authorityv1.ClaimRequest) {
				request.ExpectedVersion.IssueMutationSequence++
			},
			code: authorityv1.ErrorStaleVersion, rule: ruleClaimStale,
		},
		"path-mismatch": {
			mutate: func(_ *authorityv1.WorkItem, request *authorityv1.ClaimRequest) {
				request.ExclusivePaths = []string{"internal/other/"}
			},
			code: authorityv1.ErrorPolicyDenied, rule: ruleClaimPaths,
		},
		"wrong-capability": {
			mutate: func(_ *authorityv1.WorkItem, request *authorityv1.ClaimRequest) {
				request.Capability = authorityv1.CapabilityWorkStatus
			},
			code: authorityv1.ErrorPolicyDenied, rule: ruleClaimCapability,
		},
		"blocked": {
			mutate: func(item *authorityv1.WorkItem, _ *authorityv1.ClaimRequest) { item.Blockers = []string{"blocker-001"} },
			code:   authorityv1.ErrorNotReady, rule: ruleClaimNotReady + ":" + ruleBlocked,
		},
	} {
		t.Run(name, func(t *testing.T) {
			now := time.Date(2026, 8, 27, 1, 0, 0, 0, time.UTC)
			item := readyWork("M3-W002")
			request := claimRequest(item, "attempt-001", "idempotency-001")
			test.mutate(&item, &request)
			service := mustClaimService(t, &memoryClaimStore{item: item}, newMemorySagaStore(now), &fakeEvents{}, now)
			_, err := service.Claim(context.Background(), claimant(), request)
			var denial *authorityv1.Denial
			if !errors.As(err, &denial) || denial.Code != test.code || denial.Rule != test.rule {
				t.Fatalf("denial = %#v (%v)", denial, err)
			}
		})
	}
}

func TestClaimInvalidPostimageReturnsNoCapability(t *testing.T) {
	now := time.Date(2026, 8, 27, 1, 0, 0, 0, time.UTC)
	item := readyWork("M3-W002")
	store := &memoryClaimStore{item: item, invalidPostimage: true}
	service := mustClaimService(t, store, newMemorySagaStore(now), &fakeEvents{}, now)

	response, err := service.Claim(context.Background(), claimant(), claimRequest(item, "attempt-001", "idempotency-001"))
	if response.Lease.Active {
		t.Fatal("invalid postimage returned a capability")
	}
	assertDenial(t, err, authorityv1.ErrorUnknownEffect, ruleClaimPostimage, requiredReconciliation, allowedSagaRead)
}

func mustClaimService(t *testing.T, store ClaimStore, sagas ClaimSagaStore, events EventSink, now time.Time) *Service {
	t.Helper()
	service, err := NewWithClaims(store, sagas, events, func() time.Time { return now })
	if err != nil {
		t.Fatalf("NewWithClaims: %v", err)
	}
	return service
}

func claimant() authorityv1.Principal {
	principal := reader()
	principal.ProfileID = "work-authority-engineer"
	principal.Capabilities = append(principal.Capabilities, authorityv1.CapabilityWorkClaim)
	return principal
}

func claimRequest(item authorityv1.WorkItem, attemptID, idempotencyKey string) authorityv1.ClaimRequest {
	return authorityv1.ClaimRequest{
		BeadID: item.BeadID, ExpectedVersion: item.Version, ExpectedIntegrity: item.Integrity,
		AttemptID: attemptID, BaseSHA: strings.Repeat("a", 40), ExclusivePaths: append([]string(nil), item.ExclusivePaths...),
		Capability: authorityv1.CapabilityTicketDelivery, IdempotencyKey: idempotencyKey, TraceRef: "trace-claim-001",
	}
}

func cloneWork(item authorityv1.WorkItem) authorityv1.WorkItem {
	item.GoalIDs = append([]string(nil), item.GoalIDs...)
	item.ProductDecisionIDs = append([]string(nil), item.ProductDecisionIDs...)
	item.ScenarioIDs = append([]string(nil), item.ScenarioIDs...)
	item.ExclusivePaths = append([]string(nil), item.ExclusivePaths...)
	item.VerificationOrder = append([]string(nil), item.VerificationOrder...)
	item.Blockers = append([]string(nil), item.Blockers...)
	item.Dependencies = append([]authorityv1.Dependency(nil), item.Dependencies...)
	item.Labels = append([]authorityv1.Label(nil), item.Labels...)
	return item
}

func cloneSaga(saga ClaimSaga) ClaimSaga {
	saga.Work = cloneWork(saga.Work)
	saga.Lease.ExclusivePaths = append([]string(nil), saga.Lease.ExclusivePaths...)
	saga.Lease.Labels = append([]authorityv1.Label(nil), saga.Lease.Labels...)
	return saga
}

func pathsOverlap(left, right []string) bool {
	for _, leftPath := range left {
		for _, rightPath := range right {
			if leftPath == rightPath || strings.HasPrefix(leftPath, rightPath) || strings.HasPrefix(rightPath, leftPath) {
				return true
			}
		}
	}
	return false
}

func assertClaimEvents(t *testing.T, events *fakeEvents, request authorityv1.ClaimRequest) {
	t.Helper()
	if len(events.events) != 3 {
		t.Fatalf("events=%#v, want intent, policy, receipt", events.events)
	}
	wantOperations := []string{operationClaimIntent, operationClaimPolicy, operationClaimReceipt}
	for index, event := range events.events {
		if event.Sequence != uint64(index+1) || event.Operation != wantOperations[index] || event.TraceRef != request.TraceRef || event.AttemptID != request.AttemptID || event.IdempotencyKey != request.IdempotencyKey || event.EventID == "" {
			t.Fatalf("event[%d] = %#v", index, event)
		}
	}
}
