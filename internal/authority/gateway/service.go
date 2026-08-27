/*
FactoryDocSync:
docs:
- docs/features/F-002-work-authority.md
- docs/design-docs/ADR-001-git-beads-authority.md
- docs/code-documentation-map.md
*/

// Package gateway owns authenticated, tenant-scoped access to canonical work
// authority. Models and workers receive these typed projections, never direct
// datastore handles or credentials.
package gateway

import (
	"context"
	"errors"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	authorityv1 "github.com/greaveselliott/MARS-3/api/authority/v1"
)

var (
	// ErrWorkNotFound is the only store error translated to a not-found denial.
	// Other errors are deliberately collapsed to authority_unavailable.
	ErrWorkNotFound = errors.New("work not found")
	boundedID       = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
	hexDigest       = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

const (
	maxProjectionItems             = 100
	maxProjectionValues            = 64
	ruleAllowed                    = "authority.allowed"
	ruleInvalidRequest             = "authority.request.invalid"
	ruleInvalidTrace               = "authority.trace.invalid"
	ruleUnauthenticated            = "authority.principal.invalid"
	ruleCapabilityMissing          = "authority.capability.missing"
	ruleTenantMismatch             = "authority.tenant.mismatch"
	ruleNotFound                   = "authority.work.not_found"
	ruleProjectionInvalid          = "authority.projection.invalid"
	ruleLabelInvalid               = "authority.label.invalid"
	ruleLabelDowngrade             = "authority.label.downgrade"
	rulePrivateProjection          = "authority.label.private_denied"
	ruleLethalTrifecta             = "authority.label.lethal_trifecta"
	ruleAuthorityUnavailable       = "authority.store.unavailable"
	ruleLifecycleNotBacklog        = "authority.ready.lifecycle"
	ruleBlocked                    = "authority.ready.blocked"
	ruleDependencyLifecycle        = "authority.ready.dependency_lifecycle"
	ruleDependencyReview           = "authority.ready.dependency_review"
	ruleDependencyRun              = "authority.ready.dependency_run"
	ruleDependencyReconciliation   = "authority.ready.dependency_reconciliation"
	ruleLineageMissing             = "authority.ready.lineage"
	rulePathsMissing               = "authority.ready.paths"
	ruleVerificationOrderMissing   = "authority.ready.verification_order"
	ruleVersionInvalid             = "authority.ready.version"
	ruleIntegrityInvalid           = "authority.ready.integrity"
	operationGet                   = "work.get"
	operationReady                 = "work.ready"
	outcomeAllowed                 = "allowed"
	outcomeDenied                  = "denied"
	invalidTraceRef                = "trace-invalid"
	unknownIdentity                = "unknown"
	requiredAuthenticate           = "authenticate a tenant-scoped principal"
	requiredReadCapability         = "obtain policy-approved work.read capability"
	requiredCorrectTenant          = "select the principal tenant and project"
	requiredValidRequest           = "submit a bounded typed request"
	requiredCanonicalProjection    = "repair the canonical work projection"
	requiredSafeCompartment        = "remove one risky capability or use human mediation"
	requiredLineage                = "record complete canonical lineage"
	requiredPaths                  = "record normalized exclusive paths"
	requiredVerificationOrder      = "record the independent verification order"
	requiredVersion                = "read a complete canonical WorkVersion"
	requiredIntegrity              = "recompute canonical integrity digests"
	allowedAuthenticate            = "session.authenticate"
	allowedReadPolicy              = "policy.request(work.read)"
	allowedSelectProject           = "project.select"
	allowedRetryRead               = "work.read"
	allowedHumanMediate            = "approval.request"
	allowedResolveDependencies     = "work.ready"
	allowedReplan                  = "plan.reconcile"
	allowedResolveBlocker          = "blocker.resolve"
	allowedRepairLineage           = "work.lineage.reconcile"
	allowedRepairPaths             = "work.paths.reconcile"
	allowedRepairVerificationOrder = "work.verification.reconcile"
	allowedRefreshVersion          = "work.read"
	allowedRebaseline              = "projection.rebaseline"
)

// WorkStore exposes bounded canonical Beads/Dolt projections. Implementations
// must already scope reads to the supplied tenant and project.
type WorkStore interface {
	Get(context.Context, string, string, string) (authorityv1.WorkItem, error)
	List(context.Context, string, string) ([]authorityv1.WorkItem, error)
}

// EventSink durably appends one bounded authority event and assigns its ordered
// sequence. A failed append makes the operation non-authorizing.
type EventSink interface {
	Append(context.Context, authorityv1.Event) (authorityv1.Event, error)
}

// Service contains only trusted dependencies; it owns admission and never
// exposes their handles to callers.
type Service struct {
	store  WorkStore
	claims ClaimStore
	sagas  ClaimSagaStore
	events EventSink
	now    func() time.Time
}

func New(store WorkStore, events EventSink, now func() time.Time) (*Service, error) {
	if store == nil || events == nil || now == nil {
		return nil, errors.New("gateway dependencies are required")
	}
	return &Service{store: store, events: events, now: now}, nil
}

// GetWork returns one label-filtered canonical projection after a durable read
// event exists. Backend failures and malformed records never escape verbatim.
func (s *Service) GetWork(ctx context.Context, principal authorityv1.Principal, request authorityv1.GetWorkRequest) (authorityv1.WorkItem, error) {
	traceRef, denial := admitRequest(principal, request.TraceRef, request.ProposedLabels)
	if denial != nil {
		return authorityv1.WorkItem{}, s.deny(ctx, principal, operationGet, "", traceRef, nil, denial)
	}
	if !hasCapability(principal, authorityv1.CapabilityWorkRead) {
		return authorityv1.WorkItem{}, s.deny(ctx, principal, operationGet, request.BeadID, traceRef, nil, newDenial(authorityv1.ErrorUnauthorized, ruleCapabilityMissing, "", requiredReadCapability, allowedReadPolicy, traceRef))
	}
	if !validID(request.BeadID) {
		return authorityv1.WorkItem{}, s.deny(ctx, principal, operationGet, "", traceRef, nil, newDenial(authorityv1.ErrorInvalidRequest, ruleInvalidRequest, "", requiredValidRequest, allowedRetryRead, traceRef))
	}

	item, err := s.store.Get(ctx, principal.TenantID, principal.ProjectID, request.BeadID)
	if err != nil {
		if errors.Is(err, ErrWorkNotFound) {
			return authorityv1.WorkItem{}, s.deny(ctx, principal, operationGet, request.BeadID, traceRef, nil, newDenial(authorityv1.ErrorNotFound, ruleNotFound, "", requiredValidRequest, allowedRetryRead, traceRef))
		}
		return authorityv1.WorkItem{}, s.deny(ctx, principal, operationGet, request.BeadID, traceRef, nil, newDenial(authorityv1.ErrorAuthorityDown, ruleAuthorityUnavailable, "", requiredCanonicalProjection, allowedRetryRead, traceRef))
	}
	item = normalizeWorkItem(item)
	if item.BeadID != request.BeadID {
		return authorityv1.WorkItem{}, s.deny(ctx, principal, operationGet, request.BeadID, traceRef, nil, newDenial(authorityv1.ErrorPolicyDenied, ruleProjectionInvalid, item.LifecycleState, requiredCanonicalProjection, allowedRebaseline, traceRef))
	}
	if item.TenantID != principal.TenantID || item.ProjectID != principal.ProjectID {
		return authorityv1.WorkItem{}, s.deny(ctx, principal, operationGet, request.BeadID, traceRef, nil, newDenial(authorityv1.ErrorTenantMismatch, ruleTenantMismatch, item.LifecycleState, requiredCorrectTenant, allowedSelectProject, traceRef))
	}
	if invalid := projectionRule(item); invalid != "" {
		return authorityv1.WorkItem{}, s.deny(ctx, principal, operationGet, item.BeadID, traceRef, item.Labels, denialForProjection(invalid, item.LifecycleState, traceRef))
	}

	labels, labelDenial := admittedLabels(principal.Labels, item.Labels, request.ProposedLabels, item.LifecycleState, traceRef)
	if labelDenial != nil {
		return authorityv1.WorkItem{}, s.deny(ctx, principal, operationGet, item.BeadID, traceRef, labels, labelDenial)
	}
	item.Labels = labels
	if err := s.allow(ctx, principal, operationGet, item.BeadID, traceRef, labels); err != nil {
		return authorityv1.WorkItem{}, err
	}
	return item, nil
}

// Ready returns the complete deterministic tenant/project ready view. Items
// remain visible when not ready so the bounded corrective rule is explicit.
func (s *Service) Ready(ctx context.Context, principal authorityv1.Principal, request authorityv1.ReadyRequest) (authorityv1.ReadyResponse, error) {
	traceRef, denial := admitRequest(principal, request.TraceRef, request.ProposedLabels)
	if denial != nil {
		return authorityv1.ReadyResponse{}, s.deny(ctx, principal, operationReady, "", traceRef, nil, denial)
	}
	if !hasCapability(principal, authorityv1.CapabilityWorkRead) {
		return authorityv1.ReadyResponse{}, s.deny(ctx, principal, operationReady, "", traceRef, nil, newDenial(authorityv1.ErrorUnauthorized, ruleCapabilityMissing, "", requiredReadCapability, allowedReadPolicy, traceRef))
	}

	items, err := s.store.List(ctx, principal.TenantID, principal.ProjectID)
	if err != nil {
		return authorityv1.ReadyResponse{}, s.deny(ctx, principal, operationReady, "", traceRef, nil, newDenial(authorityv1.ErrorAuthorityDown, ruleAuthorityUnavailable, "", requiredCanonicalProjection, allowedRetryRead, traceRef))
	}
	if len(items) > maxProjectionItems {
		return authorityv1.ReadyResponse{}, s.deny(ctx, principal, operationReady, "", traceRef, nil, newDenial(authorityv1.ErrorPolicyDenied, ruleProjectionInvalid, "", requiredCanonicalProjection, allowedRebaseline, traceRef))
	}
	response := authorityv1.ReadyResponse{Items: make([]authorityv1.ReadyItem, 0, len(items))}
	eventLabels, _ := admittedLabels(principal.Labels, nil, request.ProposedLabels, "", traceRef)
	seenBeads := make(map[string]bool, len(items))
	for _, original := range items {
		item := normalizeWorkItem(original)
		if seenBeads[item.BeadID] {
			return authorityv1.ReadyResponse{}, s.deny(ctx, principal, operationReady, item.BeadID, traceRef, nil, newDenial(authorityv1.ErrorPolicyDenied, ruleProjectionInvalid, item.LifecycleState, requiredCanonicalProjection, allowedRebaseline, traceRef))
		}
		seenBeads[item.BeadID] = true
		if item.TenantID != principal.TenantID || item.ProjectID != principal.ProjectID {
			return authorityv1.ReadyResponse{}, s.deny(ctx, principal, operationReady, item.BeadID, traceRef, nil, newDenial(authorityv1.ErrorTenantMismatch, ruleTenantMismatch, item.LifecycleState, requiredCorrectTenant, allowedSelectProject, traceRef))
		}
		if invalid := projectionRule(item); invalid != "" {
			return authorityv1.ReadyResponse{}, s.deny(ctx, principal, operationReady, item.BeadID, traceRef, item.Labels, denialForProjection(invalid, item.LifecycleState, traceRef))
		}
		labels, labelDenial := admittedLabels(principal.Labels, item.Labels, request.ProposedLabels, item.LifecycleState, traceRef)
		if labelDenial != nil {
			return authorityv1.ReadyResponse{}, s.deny(ctx, principal, operationReady, item.BeadID, traceRef, labels, labelDenial)
		}
		item.Labels = labels
		eventLabels = unionLabels(eventLabels, labels)
		rules := readinessRules(item)
		ready := len(rules) == 0
		action := allowedResolveDependencies
		if ready {
			action = string(authorityv1.CapabilityWorkClaim)
		} else {
			action = allowedActionForReadinessRule(rules[0])
		}
		response.Items = append(response.Items, authorityv1.ReadyItem{WorkItem: item, Ready: ready, Rules: rules, Action: action})
	}
	sort.Slice(response.Items, func(i, j int) bool { return response.Items[i].BeadID < response.Items[j].BeadID })
	if err := s.allow(ctx, principal, operationReady, "", traceRef, eventLabels); err != nil {
		return authorityv1.ReadyResponse{}, err
	}
	return response, nil
}

func admitRequest(principal authorityv1.Principal, traceRef string, proposed []authorityv1.Label) (string, *authorityv1.Denial) {
	if !validID(traceRef) {
		return invalidTraceRef, newDenial(authorityv1.ErrorInvalidRequest, ruleInvalidTrace, "", requiredValidRequest, allowedRetryRead, invalidTraceRef)
	}
	if !validID(principal.TenantID) || !validID(principal.ProjectID) || !validID(principal.PrincipalID) || !validID(principal.ProfileID) {
		return traceRef, newDenial(authorityv1.ErrorUnauthorized, ruleUnauthenticated, "", requiredAuthenticate, allowedAuthenticate, traceRef)
	}
	if len(principal.Labels) == 0 {
		return traceRef, newDenial(authorityv1.ErrorPolicyDenied, ruleLabelInvalid, "", requiredSafeCompartment, allowedHumanMediate, traceRef)
	}
	if len(principal.Capabilities) > maxProjectionValues || hasDuplicateCapabilities(principal.Capabilities) {
		return traceRef, newDenial(authorityv1.ErrorUnauthorized, ruleUnauthenticated, "", requiredAuthenticate, allowedAuthenticate, traceRef)
	}
	for _, capability := range principal.Capabilities {
		if !knownCapability(capability) {
			return traceRef, newDenial(authorityv1.ErrorUnauthorized, ruleUnauthenticated, "", requiredAuthenticate, allowedAuthenticate, traceRef)
		}
	}
	if _, denial := admittedLabels(principal.Labels, nil, proposed, "", traceRef); denial != nil {
		return traceRef, denial
	}
	return traceRef, nil
}

func admittedLabels(principal, resource, proposed []authorityv1.Label, state authorityv1.LifecycleState, traceRef string) ([]authorityv1.Label, *authorityv1.Denial) {
	if len(principal) > maxProjectionValues || len(resource) > maxProjectionValues || len(proposed) > maxProjectionValues || hasDuplicateLabels(principal) || hasDuplicateLabels(resource) || hasDuplicateLabels(proposed) {
		return nil, newDenial(authorityv1.ErrorPolicyDenied, ruleLabelInvalid, state, requiredSafeCompartment, allowedHumanMediate, traceRef)
	}
	combined := make(map[authorityv1.Label]bool)
	for _, label := range append(append([]authorityv1.Label{}, principal...), resource...) {
		if !knownLabel(label) {
			return nil, newDenial(authorityv1.ErrorPolicyDenied, ruleLabelInvalid, state, requiredCanonicalProjection, allowedRebaseline, traceRef)
		}
		combined[label] = true
	}
	for _, label := range proposed {
		if !knownLabel(label) {
			return nil, newDenial(authorityv1.ErrorPolicyDenied, ruleLabelInvalid, state, requiredSafeCompartment, allowedHumanMediate, traceRef)
		}
		if label == authorityv1.LabelPublicAccepted {
			return nil, newDenial(authorityv1.ErrorPolicyDenied, ruleLabelDowngrade, state, requiredSafeCompartment, allowedHumanMediate, traceRef)
		}
		combined[label] = true
	}
	labels := make([]authorityv1.Label, 0, len(combined))
	for label := range combined {
		labels = append(labels, label)
	}
	sort.Slice(labels, func(i, j int) bool { return labels[i] < labels[j] })
	if combined[authorityv1.LabelPrivateData] && !containsLabel(principal, authorityv1.LabelPrivateData) {
		return labels, newDenial(authorityv1.ErrorPolicyDenied, rulePrivateProjection, state, requiredSafeCompartment, allowedHumanMediate, traceRef)
	}
	if combined[authorityv1.LabelExternalUntrusted] && combined[authorityv1.LabelPrivateData] && combined[authorityv1.LabelExternalEffect] {
		return labels, newDenial(authorityv1.ErrorPolicyDenied, ruleLethalTrifecta, state, requiredSafeCompartment, allowedHumanMediate, traceRef)
	}
	return labels, nil
}

func projectionRule(item authorityv1.WorkItem) string {
	if !validID(item.BeadID) || !validID(item.DisplayID) || !validID(item.NativeStatus) || !knownLifecycle(item.LifecycleState) {
		return ruleProjectionInvalid
	}
	if !validUniqueIDs(item.GoalIDs, true) || !validUniqueIDs(item.ProductDecisionIDs, true) || !validID(item.FeatureID) || !validUniqueIDs(item.ScenarioIDs, true) {
		return ruleLineageMissing
	}
	if len(item.ExclusivePaths) == 0 || len(item.ExclusivePaths) > maxProjectionValues || hasDuplicateStrings(item.ExclusivePaths) {
		return rulePathsMissing
	}
	for _, exclusivePath := range item.ExclusivePaths {
		if !safeExclusivePath(exclusivePath) {
			return rulePathsMissing
		}
	}
	if !validUniqueIDs(item.VerificationOrder, true) {
		return ruleVerificationOrderMissing
	}
	if !validID(item.Version.AuthorityGeneration) || !validID(item.Version.IssueIncarnation) || item.Version.IssueMutationSequence == 0 || item.Version.DependencyGraphRevision == 0 {
		return ruleVersionInvalid
	}
	if !validUniqueIDs(item.Blockers, false) || len(item.Dependencies) > maxProjectionValues {
		return ruleProjectionInvalid
	}
	dependencyIDs := make([]string, 0, len(item.Dependencies))
	for _, dependency := range item.Dependencies {
		if !validID(dependency.BeadID) || !knownLifecycle(dependency.LifecycleState) {
			return ruleProjectionInvalid
		}
		dependencyIDs = append(dependencyIDs, dependency.BeadID)
	}
	if hasDuplicateStrings(dependencyIDs) {
		return ruleProjectionInvalid
	}
	if !validIntegrity(item.Integrity) {
		return ruleIntegrityInvalid
	}
	if len(item.Labels) == 0 || len(item.Labels) > maxProjectionValues || hasDuplicateLabels(item.Labels) {
		return ruleLabelInvalid
	}
	return ""
}

func readinessRules(item authorityv1.WorkItem) []string {
	var rules []string
	if item.LifecycleState != authorityv1.LifecycleBacklog {
		rules = append(rules, ruleLifecycleNotBacklog)
	}
	if len(item.Blockers) > 0 {
		rules = append(rules, ruleBlocked)
	}
	for _, dependency := range item.Dependencies {
		if dependency.LifecycleState != authorityv1.LifecycleDone {
			rules = appendRule(rules, ruleDependencyLifecycle)
		}
		if !dependency.ReviewAccepted {
			rules = appendRule(rules, ruleDependencyReview)
		}
		if !dependency.RunCompleted {
			rules = appendRule(rules, ruleDependencyRun)
		}
		if !dependency.Reconciled {
			rules = appendRule(rules, ruleDependencyReconciliation)
		}
	}
	return rules
}

func denialForProjection(rule string, state authorityv1.LifecycleState, traceRef string) *authorityv1.Denial {
	switch rule {
	case ruleLineageMissing:
		return newDenial(authorityv1.ErrorPolicyDenied, rule, state, requiredLineage, allowedRepairLineage, traceRef)
	case rulePathsMissing:
		return newDenial(authorityv1.ErrorPolicyDenied, rule, state, requiredPaths, allowedRepairPaths, traceRef)
	case ruleVerificationOrderMissing:
		return newDenial(authorityv1.ErrorPolicyDenied, rule, state, requiredVerificationOrder, allowedRepairVerificationOrder, traceRef)
	case ruleVersionInvalid:
		return newDenial(authorityv1.ErrorPolicyDenied, rule, state, requiredVersion, allowedRefreshVersion, traceRef)
	case ruleIntegrityInvalid:
		return newDenial(authorityv1.ErrorPolicyDenied, rule, state, requiredIntegrity, allowedRebaseline, traceRef)
	case ruleLabelInvalid:
		return newDenial(authorityv1.ErrorPolicyDenied, rule, state, requiredCanonicalProjection, allowedRebaseline, traceRef)
	default:
		return newDenial(authorityv1.ErrorPolicyDenied, ruleProjectionInvalid, state, requiredCanonicalProjection, allowedRebaseline, traceRef)
	}
}

func allowedActionForReadinessRule(rule string) string {
	switch rule {
	case ruleLifecycleNotBacklog:
		return allowedReplan
	case ruleBlocked:
		return allowedResolveBlocker
	default:
		return allowedResolveDependencies
	}
}

func newDenial(code authorityv1.ErrorCode, rule string, state authorityv1.LifecycleState, transition, action, traceRef string) *authorityv1.Denial {
	return &authorityv1.Denial{Code: code, CurrentState: state, Rule: rule, RequiredTransition: transition, AllowedAction: action, TraceRef: traceRef}
}

func (s *Service) allow(ctx context.Context, principal authorityv1.Principal, operation, beadID, traceRef string, labels []authorityv1.Label) error {
	event := s.event(principal, operation, beadID, traceRef, outcomeAllowed, ruleAllowed, labels)
	receipt, err := s.events.Append(ctx, event)
	if err == nil && validEventReceipt(event, receipt) {
		return nil
	}
	return newDenial(authorityv1.ErrorAuthorityDown, ruleAuthorityUnavailable, "", requiredCanonicalProjection, allowedRetryRead, traceRef)
}

func (s *Service) deny(ctx context.Context, principal authorityv1.Principal, operation, beadID, traceRef string, labels []authorityv1.Label, denial *authorityv1.Denial) error {
	event := s.event(principal, operation, beadID, traceRef, outcomeDenied, denial.Rule, labels)
	if receipt, err := s.events.Append(ctx, event); err != nil || !validEventReceipt(event, receipt) {
		return newDenial(authorityv1.ErrorAuthorityDown, ruleAuthorityUnavailable, denial.CurrentState, requiredCanonicalProjection, allowedRetryRead, traceRef)
	}
	return denial
}

func validEventReceipt(event, receipt authorityv1.Event) bool {
	return receipt.Sequence > 0 && receipt.SchemaVersion == event.SchemaVersion && receipt.EventID == event.EventID && receipt.TraceRef == event.TraceRef && receipt.TenantID == event.TenantID && receipt.ProjectID == event.ProjectID && receipt.BeadID == event.BeadID && receipt.AttemptID == event.AttemptID && receipt.IdempotencyKey == event.IdempotencyKey && receipt.PrincipalID == event.PrincipalID && receipt.Operation == event.Operation && receipt.Outcome == event.Outcome && receipt.Rule == event.Rule && equalLabels(receipt.Labels, event.Labels) && !receipt.OccurredAt.IsZero() && !receipt.OccurredAt.After(event.OccurredAt)
}

func (s *Service) event(principal authorityv1.Principal, operation, beadID, traceRef, outcome, rule string, labels []authorityv1.Label) authorityv1.Event {
	tenantID, projectID, principalID := principal.TenantID, principal.ProjectID, principal.PrincipalID
	if !validID(tenantID) {
		tenantID = unknownIdentity
	}
	if !validID(projectID) {
		projectID = unknownIdentity
	}
	if !validID(principalID) {
		principalID = unknownIdentity
	}
	if !validID(beadID) {
		beadID = ""
	}
	return authorityv1.Event{
		SchemaVersion: 1,
		EventID:       deterministicEventID(traceRef, operation+"\x00"+beadID),
		TraceRef:      traceRef,
		TenantID:      tenantID,
		ProjectID:     projectID,
		BeadID:        beadID,
		PrincipalID:   principalID,
		Operation:     operation,
		Outcome:       outcome,
		Rule:          rule,
		Labels:        append([]authorityv1.Label(nil), labels...),
		OccurredAt:    s.now().UTC(),
	}
}

func normalizeWorkItem(item authorityv1.WorkItem) authorityv1.WorkItem {
	item.GoalIDs = sortedStrings(item.GoalIDs)
	item.ProductDecisionIDs = sortedStrings(item.ProductDecisionIDs)
	item.ScenarioIDs = sortedStrings(item.ScenarioIDs)
	item.ExclusivePaths = sortedStrings(item.ExclusivePaths)
	item.VerificationOrder = append([]string(nil), item.VerificationOrder...)
	item.Blockers = sortedStrings(item.Blockers)
	item.Dependencies = append([]authorityv1.Dependency(nil), item.Dependencies...)
	sort.Slice(item.Dependencies, func(i, j int) bool { return item.Dependencies[i].BeadID < item.Dependencies[j].BeadID })
	item.Labels = sortedLabels(item.Labels)
	return item
}

func validIntegrity(digests authorityv1.IntegrityDigests) bool {
	return hexDigest.MatchString(digests.Lineage) && hexDigest.MatchString(digests.DependencyOutcomes) && hexDigest.MatchString(digests.Blockers) && hexDigest.MatchString(digests.ExclusivePaths)
}

func safeExclusivePath(value string) bool {
	if value == "" || strings.HasPrefix(value, "/") || strings.Contains(value, "\\") {
		return false
	}
	clean := path.Clean(value)
	canonical := clean == value || (strings.HasSuffix(value, "/") && clean+"/" == value)
	return canonical && clean != "." && clean != ".." && !strings.HasPrefix(clean, "../")
}

func validID(value string) bool { return boundedID.MatchString(value) }

func validUniqueIDs(values []string, required bool) bool {
	if (required && len(values) == 0) || len(values) > maxProjectionValues || hasDuplicateStrings(values) {
		return false
	}
	for _, value := range values {
		if !validID(value) {
			return false
		}
	}
	return true
}

func hasDuplicateStrings(values []string) bool {
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		if seen[value] {
			return true
		}
		seen[value] = true
	}
	return false
}

func knownLifecycle(state authorityv1.LifecycleState) bool {
	switch state {
	case authorityv1.LifecycleBacklog, authorityv1.LifecycleInProgress, authorityv1.LifecycleInReview, authorityv1.LifecycleDone, authorityv1.LifecycleSuperseded:
		return true
	default:
		return false
	}
}

func knownLabel(label authorityv1.Label) bool {
	switch label {
	case authorityv1.LabelPublicAccepted, authorityv1.LabelExternalUntrusted, authorityv1.LabelPrivateData, authorityv1.LabelExternalEffect:
		return true
	default:
		return false
	}
}

func knownCapability(capability authorityv1.Capability) bool {
	switch capability {
	case authorityv1.CapabilityWorkRead, authorityv1.CapabilityWorkClaim, authorityv1.CapabilityWorkStatus, authorityv1.CapabilityWorkHandoff, authorityv1.CapabilityReviewRecord, authorityv1.CapabilityRunDisposition, authorityv1.CapabilityLeaseIssue, authorityv1.CapabilityLeaseRenew, authorityv1.CapabilityLeaseRelease, authorityv1.CapabilityLeaseRevoke, authorityv1.CapabilityEffectValidate, authorityv1.CapabilityTicketDelivery:
		return true
	default:
		return false
	}
}

func hasCapability(principal authorityv1.Principal, capability authorityv1.Capability) bool {
	for _, candidate := range principal.Capabilities {
		if candidate == capability {
			return true
		}
	}
	return false
}

func hasDuplicateCapabilities(values []authorityv1.Capability) bool {
	seen := make(map[authorityv1.Capability]bool, len(values))
	for _, value := range values {
		if seen[value] {
			return true
		}
		seen[value] = true
	}
	return false
}

func hasDuplicateLabels(values []authorityv1.Label) bool {
	seen := make(map[authorityv1.Label]bool, len(values))
	for _, value := range values {
		if seen[value] {
			return true
		}
		seen[value] = true
	}
	return false
}

func containsLabel(labels []authorityv1.Label, expected authorityv1.Label) bool {
	for _, label := range labels {
		if label == expected {
			return true
		}
	}
	return false
}

func appendRule(rules []string, rule string) []string {
	for _, existing := range rules {
		if existing == rule {
			return rules
		}
	}
	return append(rules, rule)
}

func sortedStrings(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}

func sortedLabels(values []authorityv1.Label) []authorityv1.Label {
	result := append([]authorityv1.Label(nil), values...)
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func unionLabels(sets ...[]authorityv1.Label) []authorityv1.Label {
	seen := make(map[authorityv1.Label]bool)
	for _, labels := range sets {
		for _, label := range labels {
			seen[label] = true
		}
	}
	result := make([]authorityv1.Label, 0, len(seen))
	for label := range seen {
		result = append(result, label)
	}
	return sortedLabels(result)
}

func equalLabels(left, right []authorityv1.Label) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
