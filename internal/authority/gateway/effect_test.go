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
	"testing"
	"time"

	authorityv1 "github.com/greaveselliott/MARS-3/api/authority/v1"
)

type effectStore struct {
	items []authorityv1.WorkItem
	errAt int
	reads int
}

func (store *effectStore) Get(_ context.Context, tenantID, projectID, beadID string) (authorityv1.WorkItem, error) {
	store.reads++
	if store.errAt == store.reads {
		return authorityv1.WorkItem{}, errors.New("canonical store unavailable with private backend details")
	}
	index := store.reads - 1
	if index >= len(store.items) {
		index = len(store.items) - 1
	}
	item := store.items[index]
	if item.TenantID != tenantID || item.ProjectID != projectID || item.BeadID != beadID {
		return authorityv1.WorkItem{}, ErrWorkNotFound
	}
	return cloneWork(item), nil
}

func (store *effectStore) List(context.Context, string, string) ([]authorityv1.WorkItem, error) {
	return nil, errors.New("not used")
}

func (store *effectStore) CompareAndSwapClaim(context.Context, ClaimMutation) (authorityv1.WorkItem, error) {
	return authorityv1.WorkItem{}, errors.New("not used")
}

type effectLeaseValidator struct {
	lease authorityv1.CapabilityLease
	err   error
	calls int
}

func (validator *effectLeaseValidator) ValidateFence(_ context.Context, _ authorityv1.FencingTuple) (authorityv1.CapabilityLease, error) {
	validator.calls++
	return validator.lease, validator.err
}

func TestValidateEffectRequiresStableCanonicalClaimAndExactLease(t *testing.T) {
	now := time.Date(2026, 8, 27, 2, 0, 0, 0, time.UTC)
	work, lease, request := effectFixture(now)
	store := &effectStore{items: []authorityv1.WorkItem{work, work}}
	validator := &effectLeaseValidator{lease: lease}
	events := &fakeEvents{}
	service := mustEffectService(t, store, validator, events, now)

	response, err := service.ValidateEffect(context.Background(), effectPrincipal(), request)
	if err != nil {
		t.Fatalf("ValidateEffect: %v", err)
	}
	if !response.Allowed || response.EffectID != request.EffectID || response.LeaseID != lease.LeaseID || response.CheckedAt != now || response.ReceiptRef == "" {
		t.Fatalf("response = %#v", response)
	}
	if store.reads != 2 || validator.calls != 1 {
		t.Fatalf("canonical reads=%d lease validations=%d, want 2 and 1", store.reads, validator.calls)
	}
	wantOperations := []string{operationEffectIntent, operationEffectPolicy, operationEffectReceipt}
	if len(events.events) != len(wantOperations) {
		t.Fatalf("events = %#v", events.events)
	}
	for index, event := range events.events {
		if event.Operation != wantOperations[index] || event.EventID == "" || event.AttemptID != request.Fence.AttemptID || event.IdempotencyKey != request.Fence.IdempotencyKey || event.TraceRef != request.TraceRef {
			t.Fatalf("event[%d] = %#v", index, event)
		}
	}

	replayed, err := service.ValidateEffect(context.Background(), effectPrincipal(), request)
	if err != nil {
		t.Fatalf("ValidateEffect replay: %v", err)
	}
	if !reflect.DeepEqual(replayed, response) || len(events.events) != 3 {
		t.Fatalf("replay=%#v events=%d, want stable receipt %#v and 3 events", replayed, len(events.events), response)
	}
}

func TestValidateEffectDeniesCanonicalDriftAfterLeaseCheck(t *testing.T) {
	now := time.Date(2026, 8, 27, 2, 0, 0, 0, time.UTC)
	work, lease, request := effectFixture(now)
	drifted := cloneWork(work)
	drifted.Version.IssueMutationSequence++
	store := &effectStore{items: []authorityv1.WorkItem{work, drifted}}
	events := &fakeEvents{}
	service := mustEffectService(t, store, &effectLeaseValidator{lease: lease}, events, now)

	_, err := service.ValidateEffect(context.Background(), effectPrincipal(), request)
	assertDenial(t, err, authorityv1.ErrorPolicyDenied, ruleEffectClaim, requiredStableClaim, allowedEffectRecheck)
	if len(events.events) != 3 || events.events[2].Outcome != outcomeDenied {
		t.Fatalf("events = %#v, want denied receipt after intent and policy", events.events)
	}
}

func TestValidateEffectDeniesStaleFenceAndOutOfScopePath(t *testing.T) {
	now := time.Date(2026, 8, 27, 2, 0, 0, 0, time.UTC)
	work, lease, request := effectFixture(now)

	t.Run("stale-fence", func(t *testing.T) {
		stale := lease
		stale.LeaseEpoch++
		store := &effectStore{items: []authorityv1.WorkItem{work, work}}
		events := &fakeEvents{}
		service := mustEffectService(t, store, &effectLeaseValidator{lease: stale}, events, now)
		_, err := service.ValidateEffect(context.Background(), effectPrincipal(), request)
		assertDenial(t, err, authorityv1.ErrorPolicyDenied, ruleEffectFence, requiredExactFence, allowedEffectRecheck)
		if store.reads != 1 {
			t.Fatalf("canonical reads=%d, want one before stale lease denial", store.reads)
		}
	})

	t.Run("changed-canonical-claim-attempt", func(t *testing.T) {
		request := request
		request.Fence.CanonicalClaimAttemptID = "other-bootstrap-attempt"
		store := &effectStore{items: []authorityv1.WorkItem{work}}
		validator := &effectLeaseValidator{lease: lease}
		service := mustEffectService(t, store, validator, &fakeEvents{}, now)
		_, err := service.ValidateEffect(context.Background(), effectPrincipal(), request)
		assertDenial(t, err, authorityv1.ErrorPolicyDenied, ruleEffectClaim, requiredStableClaim, allowedEffectRecheck)
		if store.reads != 1 || validator.calls != 0 {
			t.Fatalf("claim-attempt drift reached lease authority: reads=%d lease=%d", store.reads, validator.calls)
		}
	})

	t.Run("path-outside-lease", func(t *testing.T) {
		request := request
		request.Path = "docs/private.md"
		store := &effectStore{items: []authorityv1.WorkItem{work}}
		validator := &effectLeaseValidator{lease: lease}
		service := mustEffectService(t, store, validator, &fakeEvents{}, now)
		_, err := service.ValidateEffect(context.Background(), effectPrincipal(), request)
		assertDenial(t, err, authorityv1.ErrorPolicyDenied, ruleEffectPath, requiredEffectPath, allowedEffectRecheck)
		if store.reads != 0 || validator.calls != 0 {
			t.Fatalf("out-of-scope path reached trusted stores: reads=%d lease=%d", store.reads, validator.calls)
		}
	})

	t.Run("noncanonical-lease-path", func(t *testing.T) {
		request := request
		request.Fence.ExclusivePaths = []string{"internal/authority/../private/"}
		service := mustEffectService(t, &effectStore{items: []authorityv1.WorkItem{work}}, &effectLeaseValidator{lease: lease}, &fakeEvents{}, now)
		_, err := service.ValidateEffect(context.Background(), effectPrincipal(), request)
		assertDenial(t, err, authorityv1.ErrorInvalidRequest, ruleEffectInvalid, requiredExactFence, allowedEffectRecheck)
	})
}

func TestValidateEffectDeniesLethalTrifectaAndChangedLabels(t *testing.T) {
	now := time.Date(2026, 8, 27, 2, 0, 0, 0, time.UTC)
	work, lease, request := effectFixture(now)

	t.Run("lethal-trifecta", func(t *testing.T) {
		privatePrincipal := effectPrincipal()
		privatePrincipal.Labels = []authorityv1.Label{authorityv1.LabelPrivateData}
		work := cloneWork(work)
		work.Labels = []authorityv1.Label{authorityv1.LabelExternalUntrusted}
		request := request
		request.ProposedLabels = []authorityv1.Label{authorityv1.LabelExternalEffect}
		request.Fence.Labels = []authorityv1.Label{authorityv1.LabelExternalEffect, authorityv1.LabelExternalUntrusted, authorityv1.LabelPrivateData}
		store := &effectStore{items: []authorityv1.WorkItem{work}}
		validator := &effectLeaseValidator{lease: lease}
		service := mustEffectService(t, store, validator, &fakeEvents{}, now)
		_, err := service.ValidateEffect(context.Background(), privatePrincipal, request)
		assertDenial(t, err, authorityv1.ErrorPolicyDenied, ruleLethalTrifecta, requiredSafeCompartment, allowedHumanMediate)
		if validator.calls != 0 {
			t.Fatal("lethal-trifecta request reached lease validation")
		}
	})

	t.Run("changed-labels", func(t *testing.T) {
		request := request
		request.Fence.Labels = []authorityv1.Label{authorityv1.LabelExternalUntrusted}
		service := mustEffectService(t, &effectStore{items: []authorityv1.WorkItem{work}}, &effectLeaseValidator{lease: lease}, &fakeEvents{}, now)
		_, err := service.ValidateEffect(context.Background(), effectPrincipal(), request)
		assertDenial(t, err, authorityv1.ErrorPolicyDenied, ruleEffectLabels, requiredExactFence, allowedEffectRecheck)
	})
}

func TestValidateEffectWithholdsReceiptWhenJournalFails(t *testing.T) {
	now := time.Date(2026, 8, 27, 2, 0, 0, 0, time.UTC)
	work, lease, request := effectFixture(now)
	events := &fakeEvents{err: errors.New("trace backend unavailable with private details"), failAt: 3}
	service := mustEffectService(t, &effectStore{items: []authorityv1.WorkItem{work, work}}, &effectLeaseValidator{lease: lease}, events, now)

	_, err := service.ValidateEffect(context.Background(), effectPrincipal(), request)
	assertDenial(t, err, authorityv1.ErrorAuthorityDown, ruleAuthorityUnavailable, requiredStableClaim, allowedEffectRecheck)
}

func effectFixture(now time.Time) (authorityv1.WorkItem, authorityv1.CapabilityLease, authorityv1.EffectValidationRequest) {
	work := readyWork("M3-W002")
	work.NativeStatus = "in_progress"
	work.LifecycleState = authorityv1.LifecycleInProgress
	work.Assignee = "work-authority-engineer"
	work.ClaimAttemptID = "attempt-001"
	work.Version.IssueMutationSequence = 2
	lease := authorityv1.CapabilityLease{
		LeaseID: "lease-001", TenantID: work.TenantID, ProjectID: work.ProjectID, BeadID: work.BeadID,
		AttemptID: "delivery-attempt-001", CanonicalClaimAttemptID: work.ClaimAttemptID, IdempotencyKey: "idempotency-001", FenceGeneration: "generation-lease-001", LeaseEpoch: 7,
		ClaimVersion: work.Version, BaseSHA: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Capability: authorityv1.CapabilityTicketDelivery,
		ExclusivePaths: append([]string(nil), work.ExclusivePaths...), Labels: append([]authorityv1.Label(nil), work.Labels...),
		IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Minute), State: authorityv1.LeaseActive, Active: true,
	}
	fence := authorityv1.FencingTuple{
		TenantID: lease.TenantID, ProjectID: lease.ProjectID, BeadID: lease.BeadID, AttemptID: lease.AttemptID, CanonicalClaimAttemptID: lease.CanonicalClaimAttemptID,
		IdempotencyKey: lease.IdempotencyKey, LeaseID: lease.LeaseID, FenceGeneration: lease.FenceGeneration, LeaseEpoch: lease.LeaseEpoch,
		ClaimVersion: lease.ClaimVersion, BaseSHA: lease.BaseSHA, Capability: lease.Capability,
		ExclusivePaths: append([]string(nil), lease.ExclusivePaths...), Labels: append([]authorityv1.Label(nil), lease.Labels...),
	}
	request := authorityv1.EffectValidationRequest{Fence: fence, EffectID: "effect-001", Path: "internal/authority/gateway/effect.go", TraceRef: "trace-effect-001"}
	return work, lease, request
}

func effectPrincipal() authorityv1.Principal {
	principal := claimant()
	principal.Capabilities = append(principal.Capabilities, authorityv1.CapabilityEffectValidate)
	return principal
}

func mustEffectService(t *testing.T, store *effectStore, validator LeaseValidator, events EventSink, now time.Time) *Service {
	t.Helper()
	service, err := New(store, events, func() time.Time { return now })
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	service.claims = store
	service.leases = validator
	return service
}
