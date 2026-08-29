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
	"testing"
	"time"

	authorityv1 "github.com/greaveselliott/MARS-3/api/authority/v1"
)

func (store *memorySagaStore) GetLease(_ context.Context, tenantID, projectID, leaseID string) (authorityv1.CapabilityLease, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	for _, saga := range store.sagas {
		if saga.Lease.TenantID == tenantID && saga.Lease.ProjectID == projectID && saga.Lease.LeaseID == leaseID {
			return cloneSaga(saga).Lease, nil
		}
	}
	return authorityv1.CapabilityLease{}, errors.New("lease not found")
}

func (store *memorySagaStore) ActiveLeaseForBead(_ context.Context, tenantID, projectID, beadID string) (authorityv1.CapabilityLease, bool, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	for _, saga := range store.sagas {
		lease := saga.Lease
		if lease.TenantID == tenantID && lease.ProjectID == projectID && lease.BeadID == beadID && lease.Active && lease.State == authorityv1.LeaseActive && lease.ExpiresAt.After(store.now) {
			return cloneSaga(saga).Lease, true, nil
		}
	}
	return authorityv1.CapabilityLease{}, false, nil
}

func (store *memorySagaStore) ValidateFence(ctx context.Context, fence authorityv1.FencingTuple) (authorityv1.CapabilityLease, error) {
	lease, err := store.GetLease(ctx, fence.TenantID, fence.ProjectID, fence.LeaseID)
	if err != nil || !leaseTupleMatches(lease, fence) || !lease.Active || !lease.ExpiresAt.After(store.now) {
		return authorityv1.CapabilityLease{}, errors.New("lease fence mismatch")
	}
	return lease, nil
}

func (store *memorySagaStore) EnterEffect(_ context.Context, fence authorityv1.FencingTuple) (authorityv1.CapabilityLease, func(), error) {
	store.mu.Lock()
	for _, saga := range store.sagas {
		if saga.Lease.LeaseID == fence.LeaseID && leaseTupleMatches(saga.Lease, fence) && saga.Lease.Active && saga.Lease.ExpiresAt.After(store.now) {
			lease := cloneSaga(saga).Lease
			var released bool
			return lease, func() {
				if !released {
					released = true
					store.mu.Unlock()
				}
			}, nil
		}
	}
	store.mu.Unlock()
	return authorityv1.CapabilityLease{}, nil, errors.New("lease fence mismatch")
}

func (store *memorySagaStore) Renew(_ context.Context, fence authorityv1.FencingTuple, expiry time.Time) (authorityv1.CapabilityLease, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	for key, saga := range store.sagas {
		if saga.Lease.LeaseID != fence.LeaseID || !leaseTupleMatches(saga.Lease, fence) || !saga.Lease.Active {
			continue
		}
		saga.Lease.ExpiresAt = expiry
		store.sagas[key] = saga
		return cloneSaga(saga).Lease, nil
	}
	return authorityv1.CapabilityLease{}, errors.New("lease fence mismatch")
}

func (store *memorySagaStore) Release(_ context.Context, fence authorityv1.FencingTuple) (authorityv1.CapabilityLease, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	for key, saga := range store.sagas {
		if saga.Lease.LeaseID != fence.LeaseID || !leaseTupleMatches(saga.Lease, fence) || !saga.Lease.Active {
			continue
		}
		saga.Lease.Active, saga.Lease.State = false, authorityv1.LeaseReleased
		store.sagas[key] = saga
		return cloneSaga(saga).Lease, nil
	}
	return authorityv1.CapabilityLease{}, errors.New("lease fence mismatch")
}

func (store *memorySagaStore) Revoke(_ context.Context, request authorityv1.RevokeLeaseRequest) (authorityv1.CapabilityLease, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	for key, saga := range store.sagas {
		if saga.Lease.TenantID != request.TenantID || saga.Lease.ProjectID != request.ProjectID || saga.Lease.LeaseID != request.LeaseID ||
			saga.Lease.FenceGeneration != request.FenceGeneration || saga.Lease.LeaseEpoch != request.LeaseEpoch || !saga.Lease.Active {
			continue
		}
		saga.Lease.Active, saga.Lease.State = false, authorityv1.LeaseRevoked
		store.sagas[key] = saga
		return cloneSaga(saga).Lease, nil
	}
	return authorityv1.CapabilityLease{}, errors.New("lease fence mismatch")
}

func TestLeaseOwnerLifecycleIsFencedJournaledAndIdempotent(t *testing.T) {
	clock := time.Date(2026, 8, 27, 2, 0, 0, 0, time.UTC)
	service, store, sagas, events, principal, lease := lifecycleFixture(t, &clock)
	fence := fenceFromLease(lease)
	principal.Capabilities = append(principal.Capabilities, authorityv1.CapabilityLeaseRenew, authorityv1.CapabilityLeaseRelease)

	clock = clock.Add(2 * time.Minute)
	sagas.now = clock
	newExpiry := clock.Add(14 * time.Minute)
	renewed, err := service.RenewLease(context.Background(), principal, authorityv1.RenewLeaseRequest{Fence: fence, NewExpiry: newExpiry, TraceRef: "trace-lease-renew"})
	if err != nil || renewed.Replayed || !renewed.Lease.ExpiresAt.Equal(newExpiry) || renewed.Lease.LeaseEpoch != lease.LeaseEpoch || store.claimCalls != 0 {
		t.Fatalf("renewed=%#v err=%v claimCalls=%d", renewed, err, store.claimCalls)
	}
	fence = fenceFromLease(renewed.Lease)
	replay, err := service.RenewLease(context.Background(), principal, authorityv1.RenewLeaseRequest{Fence: fence, NewExpiry: newExpiry, TraceRef: "trace-lease-renew"})
	if err != nil || !replay.Replayed || replay.ReceiptRef != renewed.ReceiptRef {
		t.Fatalf("renew replay=%#v err=%v", replay, err)
	}
	released, err := service.ReleaseLease(context.Background(), principal, authorityv1.ReleaseLeaseRequest{Fence: fence, TraceRef: "trace-lease-release"})
	if err != nil || released.Replayed || released.Lease.Active || released.Lease.State != authorityv1.LeaseReleased {
		t.Fatalf("released=%#v err=%v", released, err)
	}
	replayRelease, err := service.ReleaseLease(context.Background(), principal, authorityv1.ReleaseLeaseRequest{Fence: fence, TraceRef: "trace-lease-release"})
	if err != nil || !replayRelease.Replayed || replayRelease.ReceiptRef != released.ReceiptRef || store.claimCalls != 0 {
		t.Fatalf("release replay=%#v err=%v claimCalls=%d", replayRelease, err, store.claimCalls)
	}
	if len(events.events) != 9 {
		t.Fatalf("events=%d, want claim+renew+release triples", len(events.events))
	}
}

func TestLeaseRevocationIsIndependentOfCanonicalAvailability(t *testing.T) {
	clock := time.Date(2026, 8, 27, 2, 0, 0, 0, time.UTC)
	service, store, _, events, _, lease := lifecycleFixture(t, &clock)
	store.getErr = errors.New("canonical Beads unavailable")
	security := reader()
	security.PrincipalID, security.ProfileID = "security-reviewer", "security-reviewer"
	security.Capabilities = append(security.Capabilities, authorityv1.CapabilityLeaseRevoke)
	request := authorityv1.RevokeLeaseRequest{
		TenantID: lease.TenantID, ProjectID: lease.ProjectID, LeaseID: lease.LeaseID,
		FenceGeneration: lease.FenceGeneration, LeaseEpoch: lease.LeaseEpoch, Reason: "security-fixture-revoke", TraceRef: "trace-lease-revoke",
	}
	revoked, err := service.RevokeLease(context.Background(), security, request)
	if err != nil || revoked.Replayed || revoked.Lease.Active || revoked.Lease.State != authorityv1.LeaseRevoked || store.claimCalls != 0 {
		t.Fatalf("revoked=%#v err=%v claimCalls=%d", revoked, err, store.claimCalls)
	}
	replay, err := service.RevokeLease(context.Background(), security, request)
	if err != nil || !replay.Replayed || replay.ReceiptRef != revoked.ReceiptRef {
		t.Fatalf("revoke replay=%#v err=%v", replay, err)
	}
	if len(events.events) != 6 {
		t.Fatalf("events=%d, want claim and revoke triples", len(events.events))
	}
}

func TestLeaseLifecycleRejectsWrongAuthority(t *testing.T) {
	clock := time.Date(2026, 8, 27, 2, 0, 0, 0, time.UTC)
	service, _, sagas, _, principal, lease := lifecycleFixture(t, &clock)
	fence := fenceFromLease(lease)
	principal.Capabilities = append(principal.Capabilities, authorityv1.CapabilityLeaseRenew)

	for name, mutate := range map[string]func(*authorityv1.FencingTuple){
		"epoch":             func(value *authorityv1.FencingTuple) { value.LeaseEpoch++ },
		"generation":        func(value *authorityv1.FencingTuple) { value.FenceGeneration = "other-generation" },
		"claim attempt":     func(value *authorityv1.FencingTuple) { value.CanonicalClaimAttemptID = "other-bootstrap-attempt" },
		"execution attempt": func(value *authorityv1.FencingTuple) { value.AttemptID = "other-delivery-attempt" },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := fence
			mutate(&candidate)
			if _, err := service.RenewLease(context.Background(), principal, authorityv1.RenewLeaseRequest{Fence: candidate, NewExpiry: clock.Add(14 * time.Minute), TraceRef: "trace-wrong-fence"}); err == nil {
				t.Fatal("wrong fence was accepted")
			}
		})
	}
	stored, err := sagas.GetLease(context.Background(), lease.TenantID, lease.ProjectID, lease.LeaseID)
	if err != nil || stored.LeaseEpoch != lease.LeaseEpoch || !stored.ExpiresAt.Equal(lease.ExpiresAt) {
		t.Fatalf("lease changed after denials: %#v err=%v", stored, err)
	}
}

func lifecycleFixture(t *testing.T, clock *time.Time) (*Service, *memoryClaimStore, *memorySagaStore, *fakeEvents, authorityv1.Principal, authorityv1.CapabilityLease) {
	t.Helper()
	item := claimedWork("M3-W001", "w001-bootstrap-attempt")
	store := &memoryClaimStore{item: item}
	sagas := newMemorySagaStore(*clock)
	events := &fakeEvents{}
	service := mustClaimService(t, store, sagas, events, *clock)
	service.now = func() time.Time { return *clock }
	request := ClaimReconciliationRequest{
		ClaimRequest:            claimRequest(item, "w001-delivery-attempt", "w001-delivery-lease-001"),
		CanonicalClaimAttemptID: item.ClaimAttemptID,
	}
	principal := claimant()
	principal.Capabilities = append(principal.Capabilities, authorityv1.CapabilityLeaseIssue)
	response, err := service.ReconcileClaimedWork(context.Background(), principal, request)
	if err != nil {
		t.Fatalf("initial lease: %v", err)
	}
	return service, store, sagas, events, principal, response.Lease
}
