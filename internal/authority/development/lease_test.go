/*
FactoryDocSync:
docs:
- docs/features/F-002-work-authority.md
- docs/design-docs/ADR-001-git-beads-authority.md
- docs/code-documentation-map.md
*/

package development

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	authorityv1 "github.com/greaveselliott/MARS-3/api/authority/v1"
	"github.com/greaveselliott/MARS-3/internal/authority/gateway"
	"github.com/greaveselliott/MARS-3/internal/doctrine"
)

type workFixture struct {
	item         authorityv1.WorkItem
	mutationCall int
}

func (fixture *workFixture) Get(_ context.Context, tenantID, projectID, beadID string) (authorityv1.WorkItem, error) {
	if tenantID != fixture.item.TenantID || projectID != fixture.item.ProjectID || beadID != fixture.item.BeadID {
		return authorityv1.WorkItem{}, gateway.ErrWorkNotFound
	}
	return fixture.item, nil
}
func (fixture *workFixture) List(context.Context, string, string) ([]authorityv1.WorkItem, error) {
	return []authorityv1.WorkItem{fixture.item}, nil
}
func (fixture *workFixture) CompareAndSwapClaim(context.Context, gateway.ClaimMutation) (authorityv1.WorkItem, error) {
	fixture.mutationCall++
	return authorityv1.WorkItem{}, errors.New("canonical mutation must not be called")
}

type operationalFixture struct {
	mu     sync.Mutex
	now    time.Time
	sagas  map[string]gateway.ClaimSaga
	epoch  uint64
	events []authorityv1.Event
}

func newOperationalFixture(now time.Time) *operationalFixture {
	return &operationalFixture{now: now, sagas: make(map[string]gateway.ClaimSaga)}
}
func (fixture *operationalFixture) Lookup(_ context.Context, _, _ string, key string) (gateway.ClaimSaga, bool, error) {
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	saga, ok := fixture.sagas[key]
	return saga, ok, nil
}
func (fixture *operationalFixture) Begin(_ context.Context, intent gateway.ClaimIntent) (gateway.ClaimSaga, error) {
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	if saga, ok := fixture.sagas[intent.IdempotencyKey]; ok {
		return saga, nil
	}
	saga := gateway.ClaimSaga{RequestDigest: intent.RequestDigest, Phase: gateway.ClaimPhaseIntent, Intent: intent}
	fixture.sagas[intent.IdempotencyKey] = saga
	return saga, nil
}
func (fixture *operationalFixture) MarkCanonicalClaimed(_ context.Context, _, _ string, key, digest string, work authorityv1.WorkItem) (gateway.ClaimSaga, error) {
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	saga := fixture.sagas[key]
	if saga.RequestDigest != digest {
		return gateway.ClaimSaga{}, gateway.ErrIdempotencyConflict
	}
	saga.Phase, saga.Work = gateway.ClaimPhaseCanonical, work
	fixture.sagas[key] = saga
	return saga, nil
}
func (fixture *operationalFixture) IssueLease(_ context.Context, key, digest string, request gateway.LeaseRequest) (gateway.ClaimSaga, error) {
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	saga := fixture.sagas[key]
	if saga.RequestDigest != digest {
		return gateway.ClaimSaga{}, gateway.ErrIdempotencyConflict
	}
	if saga.Phase == gateway.ClaimPhaseComplete {
		return saga, nil
	}
	fixture.epoch++
	saga.Phase = gateway.ClaimPhaseComplete
	saga.Lease = authorityv1.CapabilityLease{
		LeaseID: fmt.Sprintf("lease-%03d", fixture.epoch), TenantID: request.TenantID, ProjectID: request.ProjectID,
		BeadID: request.BeadID, AttemptID: request.AttemptID, CanonicalClaimAttemptID: request.CanonicalClaimAttemptID, IdempotencyKey: request.IdempotencyKey,
		FenceGeneration: "generation-development", LeaseEpoch: fixture.epoch, ClaimVersion: request.ClaimVersion,
		BaseSHA: request.BaseSHA, Capability: request.Capability, ExclusivePaths: append([]string(nil), request.ExclusivePaths...),
		Labels: append([]authorityv1.Label(nil), request.Labels...), IssuedAt: fixture.now, ExpiresAt: request.MaximumExpiry,
		State: authorityv1.LeaseActive, Active: true,
	}
	saga.ReceiptRef = "receipt-development-lease"
	fixture.sagas[key] = saga
	return saga, nil
}
func (fixture *operationalFixture) Append(_ context.Context, event authorityv1.Event) (authorityv1.Event, error) {
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	for _, existing := range fixture.events {
		if existing.EventID == event.EventID {
			return existing, nil
		}
	}
	event.Sequence = uint64(len(fixture.events) + 1)
	event.EventHash = digest64("e")
	fixture.events = append(fixture.events, event)
	return event, nil
}

func TestReconcileSeparatesCanonicalClaimAndDeliveryAttempt(t *testing.T) {
	now := time.Date(2026, 8, 27, 1, 0, 0, 0, time.UTC)
	grant := deliveryGrantFixture()
	work := &workFixture{item: claimedWorkFixture(grant)}
	operational := newOperationalFixture(now)
	options := Options{TenantID: work.item.TenantID, ProjectID: work.item.ProjectID, Work: work, Operational: operational, Now: func() time.Time { return now }}
	receipt, err := reconcileWithGrant(context.Background(), grant, options)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.AttemptID != grant.AttemptID || receipt.AttemptID == grant.CanonicalClaimAttemptID || receipt.LeaseEpoch != 1 ||
		receipt.ClaimVersion != work.item.Version || receipt.State != authorityv1.LeaseActive || work.mutationCall != 0 || len(operational.events) != 3 {
		t.Fatalf("receipt=%#v mutations=%d events=%d", receipt, work.mutationCall, len(operational.events))
	}
	replayed, err := reconcileWithGrant(context.Background(), grant, options)
	if err != nil || !replayed.Replayed || replayed.LeaseID != receipt.LeaseID || operational.epoch != 1 || work.mutationCall != 0 || len(operational.events) != 3 {
		t.Fatalf("replay=%#v err=%v epoch=%d mutations=%d events=%d", replayed, err, operational.epoch, work.mutationCall, len(operational.events))
	}
}

func TestReconcileRejectsSignedPreimageDrift(t *testing.T) {
	now := time.Date(2026, 8, 27, 1, 0, 0, 0, time.UTC)
	grant := deliveryGrantFixture()
	base := claimedWorkFixture(grant)
	for name, mutate := range map[string]func(*authorityv1.WorkItem){
		"claim attempt": func(item *authorityv1.WorkItem) { item.ClaimAttemptID = "other-bootstrap-attempt" },
		"assignee":      func(item *authorityv1.WorkItem) { item.Assignee = "other-principal" },
		"generation":    func(item *authorityv1.WorkItem) { item.Version.AuthorityGeneration = "other-generation" },
		"sequence":      func(item *authorityv1.WorkItem) { item.Version.IssueMutationSequence++ },
		"lifecycle":     func(item *authorityv1.WorkItem) { item.LifecycleState = authorityv1.LifecycleInReview },
	} {
		t.Run(name, func(t *testing.T) {
			item := base
			mutate(&item)
			work := &workFixture{item: item}
			options := Options{TenantID: item.TenantID, ProjectID: item.ProjectID, Work: work, Operational: newOperationalFixture(now), Now: func() time.Time { return now }}
			if _, err := reconcileWithGrant(context.Background(), grant, options); !errors.Is(err, ErrCanonicalPreimage) {
				t.Fatalf("drift error=%v, want ErrCanonicalPreimage", err)
			}
		})
	}
}

func deliveryGrantFixture() doctrine.W001DeliveryGrant {
	return doctrine.W001DeliveryGrant{
		ID: "W-001-delivery", Bead: "M3-W001", Principal: "work-authority-engineer",
		AttemptID: "w001-delivery-attempt", IdempotencyKey: "w001-delivery-key", BaseCommit: "1111111111111111111111111111111111111111",
		ExpectedNativeStatus: "in_progress", ExpectedLifecycleState: "in-progress", ExpectedAssignee: "work-authority-engineer",
		CanonicalClaimAttemptID: "w001-bootstrap-attempt", WorkVersionGeneration: "generation-001", WorkVersionIncarnation: "incarnation-001",
		IssueMutationSequence: 1, DependencyGraphRevision: 1, DevelopmentLeaseAllowed: true,
	}
}

func claimedWorkFixture(grant doctrine.W001DeliveryGrant) authorityv1.WorkItem {
	return authorityv1.WorkItem{
		TenantID: "tenant-development", ProjectID: "project-development", BeadID: grant.Bead, DisplayID: "W-001",
		NativeStatus: grant.ExpectedNativeStatus, LifecycleState: authorityv1.LifecycleInProgress, Assignee: grant.ExpectedAssignee,
		ClaimAttemptID: grant.CanonicalClaimAttemptID, GoalIDs: []string{"G-001"}, ProductDecisionIDs: []string{"PD-002"},
		FeatureID: "F-002", ScenarioIDs: []string{"F-002-S1"}, ExclusivePaths: []string{"internal/authority/"},
		VerificationOrder: []string{"qa", "security-reviewer", "delivery-orchestrator"}, Labels: []authorityv1.Label{authorityv1.LabelPublicAccepted},
		Version:   authorityv1.WorkVersion{AuthorityGeneration: grant.WorkVersionGeneration, IssueIncarnation: grant.WorkVersionIncarnation, IssueMutationSequence: 1, DependencyGraphRevision: 1},
		Integrity: authorityv1.IntegrityDigests{Lineage: digest64("a"), DependencyOutcomes: digest64("b"), Blockers: digest64("c"), ExclusivePaths: digest64("d")},
	}
}

func digest64(value string) string {
	return value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value + value
}
