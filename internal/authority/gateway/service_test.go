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

type fakeStore struct {
	items     []authorityv1.WorkItem
	getResult *authorityv1.WorkItem
	getErr    error
	listErr   error
	got       []string
}

func (store *fakeStore) Get(_ context.Context, tenantID, projectID, beadID string) (authorityv1.WorkItem, error) {
	store.got = append(store.got, tenantID+"/"+projectID+"/"+beadID)
	if store.getErr != nil {
		return authorityv1.WorkItem{}, store.getErr
	}
	if store.getResult != nil {
		return *store.getResult, nil
	}
	for _, item := range store.items {
		if item.BeadID == beadID {
			return item, nil
		}
	}
	return authorityv1.WorkItem{}, ErrWorkNotFound
}

func (store *fakeStore) List(_ context.Context, tenantID, projectID string) ([]authorityv1.WorkItem, error) {
	store.got = append(store.got, tenantID+"/"+projectID)
	return append([]authorityv1.WorkItem(nil), store.items...), store.listErr
}

type fakeEvents struct {
	mu             sync.Mutex
	events         []authorityv1.Event
	err            error
	failAt         int
	invalidReceipt bool
}

func (sink *fakeEvents) Append(_ context.Context, event authorityv1.Event) (authorityv1.Event, error) {
	sink.mu.Lock()
	defer sink.mu.Unlock()
	if sink.err != nil || (sink.failAt > 0 && len(sink.events)+1 == sink.failAt) {
		return authorityv1.Event{}, sink.err
	}
	if event.EventID != "" {
		for _, existing := range sink.events {
			if existing.EventID == event.EventID {
				return existing, nil
			}
		}
	}
	event.Sequence = uint64(len(sink.events) + 1)
	sink.events = append(sink.events, event)
	if sink.invalidReceipt {
		event.Sequence = 0
	}
	return event, nil
}

func TestNewRequiresTrustedDependencies(t *testing.T) {
	clock := func() time.Time { return time.Unix(0, 0) }
	store := &fakeStore{}
	events := &fakeEvents{}
	for name, test := range map[string]struct {
		store  WorkStore
		events EventSink
		now    func() time.Time
	}{
		"store":  {events: events, now: clock},
		"events": {store: store, now: clock},
		"clock":  {store: store, events: events},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := New(test.store, test.events, test.now); err == nil {
				t.Fatal("New accepted a missing trusted dependency")
			}
		})
	}
}

func TestGetWorkReturnsBoundedTenantProjectionAndEvent(t *testing.T) {
	item := readyWork("M3-W002")
	item.GoalIDs = []string{"G-002", "G-001"}
	store := &fakeStore{items: []authorityv1.WorkItem{item}}
	events := &fakeEvents{}
	service := mustService(t, store, events)

	got, err := service.GetWork(context.Background(), reader(), authorityv1.GetWorkRequest{BeadID: item.BeadID, TraceRef: "trace-get-1", ProposedLabels: []authorityv1.Label{authorityv1.LabelExternalUntrusted}})
	if err != nil {
		t.Fatalf("GetWork: %v", err)
	}
	if !reflect.DeepEqual(got.GoalIDs, []string{"G-001", "G-002"}) {
		t.Fatalf("GoalIDs = %#v, want deterministic order", got.GoalIDs)
	}
	if !reflect.DeepEqual(got.Labels, []authorityv1.Label{authorityv1.LabelExternalUntrusted, authorityv1.LabelPublicAccepted}) {
		t.Fatalf("Labels = %#v, want derived union", got.Labels)
	}
	if !reflect.DeepEqual(store.got, []string{"tenant-fixture/project-fixture/M3-W002"}) {
		t.Fatalf("store scope = %#v", store.got)
	}
	assertEvent(t, events, operationGet, outcomeAllowed, ruleAllowed, "trace-get-1")
}

func TestGetWorkDeniesWithoutCapabilityBeforeStoreRead(t *testing.T) {
	store := &fakeStore{items: []authorityv1.WorkItem{readyWork("M3-W002")}}
	events := &fakeEvents{}
	service := mustService(t, store, events)
	principal := reader()
	principal.Capabilities = nil

	_, err := service.GetWork(context.Background(), principal, authorityv1.GetWorkRequest{BeadID: "M3-W002", TraceRef: "trace-deny-1"})
	assertDenial(t, err, authorityv1.ErrorUnauthorized, ruleCapabilityMissing, requiredReadCapability, allowedReadPolicy)
	if len(store.got) != 0 {
		t.Fatalf("store was read before capability admission: %#v", store.got)
	}
	assertEvent(t, events, operationGet, outcomeDenied, ruleCapabilityMissing, "trace-deny-1")
}

func TestGetWorkFailsClosedOnCrossTenantStoreResult(t *testing.T) {
	item := readyWork("M3-W002")
	item.TenantID = "tenant-other"
	store := &fakeStore{items: []authorityv1.WorkItem{item}}
	events := &fakeEvents{}
	service := mustService(t, store, events)

	_, err := service.GetWork(context.Background(), reader(), authorityv1.GetWorkRequest{BeadID: item.BeadID, TraceRef: "trace-tenant-1"})
	assertDenial(t, err, authorityv1.ErrorTenantMismatch, ruleTenantMismatch, requiredCorrectTenant, allowedSelectProject)
	assertEvent(t, events, operationGet, outcomeDenied, ruleTenantMismatch, "trace-tenant-1")
}

func TestGetWorkDoesNotExposeBackendErrors(t *testing.T) {
	store := &fakeStore{getErr: errors.New("password=private database=customer")}
	events := &fakeEvents{}
	service := mustService(t, store, events)

	_, err := service.GetWork(context.Background(), reader(), authorityv1.GetWorkRequest{BeadID: "M3-W002", TraceRef: "trace-store-1"})
	assertDenial(t, err, authorityv1.ErrorAuthorityDown, ruleAuthorityUnavailable, requiredCanonicalProjection, allowedRetryRead)
	if strings.Contains(err.Error(), "password") || strings.Contains(err.Error(), "customer") {
		t.Fatalf("backend detail escaped: %v", err)
	}
}

func TestGetWorkDeniesForgedAndLethalLabels(t *testing.T) {
	for name, test := range map[string]struct {
		principal authorityv1.Principal
		item      authorityv1.WorkItem
		proposed  []authorityv1.Label
		rule      string
	}{
		"accepted-provenance-is-not-caller-controlled": {
			principal: reader(), item: readyWork("M3-W002"), proposed: []authorityv1.Label{authorityv1.LabelPublicAccepted}, rule: ruleLabelDowngrade,
		},
		"private-projection-needs-derived-clearance": {
			principal: reader(), item: workWithLabels("M3-W002", authorityv1.LabelPrivateData), rule: rulePrivateProjection,
		},
		"local-lethal-trifecta": {
			principal: readerWithLabels(authorityv1.LabelPrivateData), item: workWithLabels("M3-W002", authorityv1.LabelExternalUntrusted), proposed: []authorityv1.Label{authorityv1.LabelExternalEffect}, rule: ruleLethalTrifecta,
		},
	} {
		t.Run(name, func(t *testing.T) {
			events := &fakeEvents{}
			service := mustService(t, &fakeStore{items: []authorityv1.WorkItem{test.item}}, events)
			_, err := service.GetWork(context.Background(), test.principal, authorityv1.GetWorkRequest{BeadID: test.item.BeadID, TraceRef: "trace-label-1", ProposedLabels: test.proposed})
			assertDenial(t, err, authorityv1.ErrorPolicyDenied, test.rule, requiredSafeCompartment, allowedHumanMediate)
			assertEvent(t, events, operationGet, outcomeDenied, test.rule, "trace-label-1")
		})
	}
}

func TestGetWorkRequiresDurableEventBeforeReturning(t *testing.T) {
	for name, events := range map[string]*fakeEvents{
		"append-failure":  {err: errors.New("journal offline with private details")},
		"invalid-receipt": {invalidReceipt: true},
	} {
		t.Run(name, func(t *testing.T) {
			service := mustService(t, &fakeStore{items: []authorityv1.WorkItem{readyWork("M3-W002")}}, events)
			_, err := service.GetWork(context.Background(), reader(), authorityv1.GetWorkRequest{BeadID: "M3-W002", TraceRef: "trace-event-1"})
			assertDenial(t, err, authorityv1.ErrorAuthorityDown, ruleAuthorityUnavailable, requiredCanonicalProjection, allowedRetryRead)
		})
	}
}

func TestReadyIsDeterministicAndExplainsEveryNonReadyRule(t *testing.T) {
	ready := readyWork("M3-W002")
	blocked := readyWork("M3-W003")
	blocked.LifecycleState = authorityv1.LifecycleInProgress
	blocked.Blockers = []string{"blocker-z", "blocker-a"}
	blocked.Labels = []authorityv1.Label{authorityv1.LabelExternalUntrusted}
	blocked.Dependencies = []authorityv1.Dependency{{
		BeadID: "M3-H002", LifecycleState: authorityv1.LifecycleInProgress,
	}}
	store := &fakeStore{items: []authorityv1.WorkItem{blocked, ready}}
	events := &fakeEvents{}
	service := mustService(t, store, events)

	response, err := service.Ready(context.Background(), reader(), authorityv1.ReadyRequest{TraceRef: "trace-ready-1"})
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}
	if len(response.Items) != 2 || response.Items[0].BeadID != "M3-W002" || response.Items[1].BeadID != "M3-W003" {
		t.Fatalf("items are not deterministically ordered: %#v", response.Items)
	}
	if !response.Items[0].Ready || response.Items[0].Action != string(authorityv1.CapabilityWorkClaim) || len(response.Items[0].Rules) != 0 {
		t.Fatalf("ready item = %#v", response.Items[0])
	}
	wantRules := []string{ruleLifecycleNotBacklog, ruleBlocked, ruleDependencyLifecycle, ruleDependencyReview, ruleDependencyRun, ruleDependencyReconciliation}
	if response.Items[1].Ready || !reflect.DeepEqual(response.Items[1].Rules, wantRules) || response.Items[1].Action != allowedReplan {
		t.Fatalf("blocked item = %#v, want rules %#v", response.Items[1], wantRules)
	}
	assertEvent(t, events, operationReady, outcomeAllowed, ruleAllowed, "trace-ready-1")
	if !reflect.DeepEqual(events.events[0].Labels, []authorityv1.Label{authorityv1.LabelExternalUntrusted, authorityv1.LabelPublicAccepted}) {
		t.Fatalf("ready event labels = %#v, want union of returned projections", events.events[0].Labels)
	}
}

func TestReadyRejectsMalformedProjectionInsteadOfOmittingIt(t *testing.T) {
	item := readyWork("M3-W002")
	item.Integrity.Lineage = "not-a-digest"
	events := &fakeEvents{}
	service := mustService(t, &fakeStore{items: []authorityv1.WorkItem{item}}, events)

	_, err := service.Ready(context.Background(), reader(), authorityv1.ReadyRequest{TraceRef: "trace-projection-1"})
	assertDenial(t, err, authorityv1.ErrorPolicyDenied, ruleIntegrityInvalid, requiredIntegrity, allowedRebaseline)
	assertEvent(t, events, operationReady, outcomeDenied, ruleIntegrityInvalid, "trace-projection-1")
}

func TestReadyRejectsDuplicateCanonicalItems(t *testing.T) {
	item := readyWork("M3-W002")
	events := &fakeEvents{}
	service := mustService(t, &fakeStore{items: []authorityv1.WorkItem{item, item}}, events)

	_, err := service.Ready(context.Background(), reader(), authorityv1.ReadyRequest{TraceRef: "trace-duplicate-1"})
	assertDenial(t, err, authorityv1.ErrorPolicyDenied, ruleProjectionInvalid, requiredCanonicalProjection, allowedRebaseline)
}

func TestReadyBoundsProjectionCardinality(t *testing.T) {
	items := make([]authorityv1.WorkItem, maxProjectionItems+1)
	for index := range items {
		items[index] = readyWork("M3-W002")
	}
	events := &fakeEvents{}
	service := mustService(t, &fakeStore{items: items}, events)

	_, err := service.Ready(context.Background(), reader(), authorityv1.ReadyRequest{TraceRef: "trace-cardinality-1"})
	assertDenial(t, err, authorityv1.ErrorPolicyDenied, ruleProjectionInvalid, requiredCanonicalProjection, allowedRebaseline)
	assertEvent(t, events, operationReady, outcomeDenied, ruleProjectionInvalid, "trace-cardinality-1")
}

func TestGetWorkRejectsDuplicateAndUnsafeCanonicalFields(t *testing.T) {
	for name, mutate := range map[string]func(*authorityv1.WorkItem){
		"duplicate-dependency": func(item *authorityv1.WorkItem) { item.Dependencies = append(item.Dependencies, item.Dependencies[0]) },
		"unsafe-path":          func(item *authorityv1.WorkItem) { item.ExclusivePaths = []string{"../private"} },
		"unbounded-blocker":    func(item *authorityv1.WorkItem) { item.Blockers = []string{"raw blocker with spaces"} },
	} {
		t.Run(name, func(t *testing.T) {
			item := readyWork("M3-W002")
			mutate(&item)
			events := &fakeEvents{}
			service := mustService(t, &fakeStore{items: []authorityv1.WorkItem{item}}, events)
			_, err := service.GetWork(context.Background(), reader(), authorityv1.GetWorkRequest{BeadID: item.BeadID, TraceRef: "trace-malformed-1"})
			var denial *authorityv1.Denial
			if !errors.As(err, &denial) || denial.Code != authorityv1.ErrorPolicyDenied {
				t.Fatalf("error = %v, want policy denial", err)
			}
		})
	}
}

func TestInvalidIdentityAndTraceAreNotReflectedIntoEvents(t *testing.T) {
	principal := reader()
	principal.TenantID = "tenant secret with spaces"
	events := &fakeEvents{}
	service := mustService(t, &fakeStore{}, events)

	_, err := service.Ready(context.Background(), principal, authorityv1.ReadyRequest{TraceRef: "private raw trace value"})
	assertDenial(t, err, authorityv1.ErrorInvalidRequest, ruleInvalidTrace, requiredValidRequest, allowedRetryRead)
	if len(events.events) != 1 || events.events[0].TraceRef != invalidTraceRef || events.events[0].TenantID != unknownIdentity {
		t.Fatalf("unbounded input reached event: %#v", events.events)
	}
}

func TestMissingLabelProvenanceIsDeniedBeforeCanonicalRead(t *testing.T) {
	principal := reader()
	principal.Labels = nil
	store := &fakeStore{items: []authorityv1.WorkItem{readyWork("M3-W002")}}
	events := &fakeEvents{}
	service := mustService(t, store, events)

	_, err := service.GetWork(context.Background(), principal, authorityv1.GetWorkRequest{BeadID: "M3-W002", TraceRef: "trace-label-missing"})
	assertDenial(t, err, authorityv1.ErrorPolicyDenied, ruleLabelInvalid, requiredSafeCompartment, allowedHumanMediate)
	if len(store.got) != 0 {
		t.Fatalf("canonical store read without principal label provenance: %#v", store.got)
	}
}

func TestGetWorkRejectsMismatchedCanonicalIdentity(t *testing.T) {
	item := readyWork("M3-W003")
	store := &fakeStore{getResult: &item}
	events := &fakeEvents{}
	service := mustService(t, store, events)

	_, err := service.GetWork(context.Background(), reader(), authorityv1.GetWorkRequest{BeadID: "M3-W002", TraceRef: "trace-identity-1"})
	assertDenial(t, err, authorityv1.ErrorPolicyDenied, ruleProjectionInvalid, requiredCanonicalProjection, allowedRebaseline)
}

func readyWork(beadID string) authorityv1.WorkItem {
	digest := strings.Repeat("a", 64)
	return authorityv1.WorkItem{
		TenantID:           "tenant-fixture",
		ProjectID:          "project-fixture",
		BeadID:             beadID,
		DisplayID:          strings.TrimPrefix(beadID, "M3-"),
		NativeStatus:       "open",
		LifecycleState:     authorityv1.LifecycleBacklog,
		GoalIDs:            []string{"G-001"},
		ProductDecisionIDs: []string{"PD-002"},
		FeatureID:          "F-002",
		ScenarioIDs:        []string{"F-002-S1"},
		ExclusivePaths:     []string{"internal/authority/"},
		VerificationOrder:  []string{"qa", "security-reviewer", "delivery-orchestrator"},
		Dependencies: []authorityv1.Dependency{{
			BeadID: "M3-H001", LifecycleState: authorityv1.LifecycleDone, ReviewAccepted: true, RunCompleted: true, Reconciled: true,
		}},
		Labels: []authorityv1.Label{authorityv1.LabelPublicAccepted},
		Version: authorityv1.WorkVersion{
			AuthorityGeneration: "generation-fixture", IssueIncarnation: "incarnation-fixture", IssueMutationSequence: 1, DependencyGraphRevision: 1,
		},
		Integrity: authorityv1.IntegrityDigests{Lineage: digest, DependencyOutcomes: digest, Blockers: digest, ExclusivePaths: digest},
	}
}

func workWithLabels(beadID string, labels ...authorityv1.Label) authorityv1.WorkItem {
	item := readyWork(beadID)
	item.Labels = labels
	return item
}

func reader() authorityv1.Principal {
	return readerWithLabels(authorityv1.LabelPublicAccepted)
}

func readerWithLabels(labels ...authorityv1.Label) authorityv1.Principal {
	return authorityv1.Principal{
		TenantID: "tenant-fixture", ProjectID: "project-fixture", PrincipalID: "principal-fixture", ProfileID: "observer", Capabilities: []authorityv1.Capability{authorityv1.CapabilityWorkRead}, Labels: labels,
	}
}

func mustService(t *testing.T, store WorkStore, events EventSink) *Service {
	t.Helper()
	service, err := New(store, events, func() time.Time { return time.Date(2026, 8, 27, 0, 0, 0, 0, time.UTC) })
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return service
}

func assertDenial(t *testing.T, err error, code authorityv1.ErrorCode, rule, transition, action string) {
	t.Helper()
	var denial *authorityv1.Denial
	if !errors.As(err, &denial) {
		t.Fatalf("error %T %v, want Denial", err, err)
	}
	if denial.Code != code || denial.Rule != rule || denial.RequiredTransition != transition || denial.AllowedAction != action {
		t.Fatalf("denial = %#v", denial)
	}
}

func assertEvent(t *testing.T, events *fakeEvents, operation, outcome, rule, traceRef string) {
	t.Helper()
	if len(events.events) != 1 {
		t.Fatalf("events = %#v, want one", events.events)
	}
	event := events.events[0]
	if event.SchemaVersion != 1 || event.Sequence != 1 || event.Operation != operation || event.Outcome != outcome || event.Rule != rule || event.TraceRef != traceRef {
		t.Fatalf("event = %#v", event)
	}
}
