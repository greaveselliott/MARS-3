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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"time"

	authorityv1 "github.com/greaveselliott/MARS-3/api/authority/v1"
)

var (
	ErrStaleWorkVersion    = errors.New("stale work version")
	ErrIdempotencyConflict = errors.New("idempotency conflict")
	commitSHA              = regexp.MustCompile(`^(?:[a-f0-9]{40}|[a-f0-9]{64})$`)
)

const (
	claimPhaseIntent    ClaimPhase = "intent-recorded"
	claimPhaseCanonical ClaimPhase = "canonical-claimed"
	claimPhaseComplete  ClaimPhase = "complete"

	operationClaimIntent  = "work.claim.intent"
	operationClaimPolicy  = "work.claim.policy"
	operationClaimReceipt = "work.claim.receipt"
	outcomeIntent         = "intent-recorded"
	outcomePolicyAllowed  = "policy-allowed"
	outcomeVerified       = "verified"
	outcomeUnknown        = "unknown"
	ruleClaimAllowed      = "authority.claim.allowed"
	ruleClaimInvalid      = "authority.claim.invalid"
	ruleClaimNotReady     = "authority.claim.not_ready"
	ruleClaimStale        = "authority.claim.stale_version"
	ruleClaimPaths        = "authority.claim.paths_mismatch"
	ruleClaimCapability   = "authority.claim.capability"
	ruleClaimIdempotency  = "authority.claim.idempotency_conflict"
	ruleClaimSagaInvalid  = "authority.claim.saga_invalid"
	ruleClaimPostimage    = "authority.claim.postimage_invalid"
	ruleLeaseInvalid      = "authority.lease.invalid"
	ruleClaimUnknown      = "authority.claim.unknown_effect"

	requiredFreshObservation = "read the current canonical WorkVersion and integrity digests"
	requiredReadyWork        = "resolve the first readiness rule and read again"
	requiredExactPaths       = "use the canonical normalized exclusive paths"
	requiredDeliveryLease    = "request the ticket.delivery capability"
	requiredNewIdempotency   = "use the original normalized request or a new idempotency key"
	requiredReconciliation   = "reconcile the pending claim saga before retry"
	requiredValidLease       = "repair lease authority and reconcile the canonical claim"
	requiredClaimCapability  = "obtain policy-approved work.claim capability"
	allowedReadyRead         = "work.ready"
	allowedClaimRetry        = "work.claim"
	allowedClaimPolicy       = "policy.request(work.claim)"
	allowedSagaRead          = "claim.reconcile"

	maximumInitialLeaseDuration = 15 * time.Minute
)

// ClaimMutation is the complete canonical Beads/Dolt compare-and-swap input.
// Implementations must mutate all named claim fields in one transaction.
type ClaimMutation struct {
	TenantID          string
	ProjectID         string
	BeadID            string
	ExpectedVersion   authorityv1.WorkVersion
	ExpectedIntegrity authorityv1.IntegrityDigests
	AttemptID         string
	Assignee          string
	IdempotencyKey    string
	BaseSHA           string
}

// ClaimStore owns the canonical work transition. It never issues capability.
type ClaimStore interface {
	WorkStore
	CompareAndSwapClaim(context.Context, ClaimMutation) (authorityv1.WorkItem, error)
}

type ClaimPhase string

const (
	ClaimPhaseIntent    ClaimPhase = claimPhaseIntent
	ClaimPhaseCanonical ClaimPhase = claimPhaseCanonical
	ClaimPhaseComplete  ClaimPhase = claimPhaseComplete
)

// ClaimIntent is the normalized cross-store saga preimage. RequestDigest is a
// deterministic hash over every authority-bearing input.
type ClaimIntent struct {
	RequestDigest  string
	TenantID       string
	ProjectID      string
	BeadID         string
	AttemptID      string
	IdempotencyKey string
	BaseSHA        string
	Capability     authorityv1.Capability
	ExclusivePaths []string
	TraceRef       string
	Labels         []authorityv1.Label
}

type LeaseRequest struct {
	RequestDigest           string
	TenantID                string
	ProjectID               string
	BeadID                  string
	AttemptID               string
	CanonicalClaimAttemptID string
	IdempotencyKey          string
	BaseSHA                 string
	Capability              authorityv1.Capability
	ExclusivePaths          []string
	Labels                  []authorityv1.Label
	ClaimVersion            authorityv1.WorkVersion
	MaximumExpiry           time.Time
}

// ClaimSaga is a durable PostgreSQL projection of a cross-store claim. It is
// execution authority only when Phase is complete and Lease passes validation.
type ClaimSaga struct {
	RequestDigest string
	Phase         ClaimPhase
	Intent        ClaimIntent
	Work          authorityv1.WorkItem
	Lease         authorityv1.CapabilityLease
	ReceiptRef    string
}

// ClaimSagaStore owns idempotency, pending-saga state, and live lease issuance.
// Begin must atomically reject reuse of a key for a different RequestDigest.
type ClaimSagaStore interface {
	Lookup(context.Context, string, string, string) (ClaimSaga, bool, error)
	Begin(context.Context, ClaimIntent) (ClaimSaga, error)
	MarkCanonicalClaimed(context.Context, string, string, string, string, authorityv1.WorkItem) (ClaimSaga, error)
	IssueLease(context.Context, string, string, LeaseRequest) (ClaimSaga, error)
}

// NewWithClaims enables the mutating claim path. Read-only Service values made
// with New remain intentionally unable to claim.
func NewWithClaims(store ClaimStore, sagas ClaimSagaStore, events EventSink, now func() time.Time) (*Service, error) {
	if sagas == nil {
		return nil, errors.New("claim saga store is required")
	}
	service, err := New(store, events, now)
	if err != nil {
		return nil, err
	}
	service.claims = store
	service.sagas = sagas
	if leases, ok := sagas.(LeaseValidator); ok {
		service.leases = leases
	}
	if effectLock, ok := sagas.(EffectFenceLocker); ok {
		service.effectLock = effectLock
	}
	if leaseOps, ok := sagas.(LeaseStore); ok {
		service.leaseOps = leaseOps
	}
	if barrier, ok := sagas.(ProjectBarrier); ok {
		service.barrier = barrier
	}
	if workLocks, ok := sagas.(WorkMutationLocker); ok {
		service.workLocks = workLocks
	}
	return service, nil
}

// Claim performs the only normal backlog-to-in-progress transition. It returns
// no capability until the canonical CAS and live lease are both verified.
func (s *Service) Claim(ctx context.Context, principal authorityv1.Principal, request authorityv1.ClaimRequest) (authorityv1.ClaimResponse, error) {
	traceRef, denial := admitRequest(principal, request.TraceRef, request.ProposedLabels)
	if denial != nil {
		return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimIntent, "", traceRef, nil, denial)
	}
	if !hasCapability(principal, authorityv1.CapabilityWorkClaim) {
		return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimIntent, request.BeadID, traceRef, nil, newDenial(authorityv1.ErrorUnauthorized, ruleCapabilityMissing, "", requiredClaimCapability, allowedClaimPolicy, traceRef))
	}
	if s.claims == nil || s.sagas == nil {
		return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimIntent, request.BeadID, traceRef, nil, newDenial(authorityv1.ErrorAuthorityDown, ruleAuthorityUnavailable, "", requiredCanonicalProjection, allowedRetryRead, traceRef))
	}
	request.ExclusivePaths = sortedStrings(request.ExclusivePaths)
	request.ProposedLabels = sortedLabels(request.ProposedLabels)
	if claimRule := validateClaimRequest(request); claimRule != "" {
		return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimIntent, request.BeadID, traceRef, nil, denialForClaimRule(claimRule, "", traceRef))
	}
	requestDigest, err := claimRequestDigest(principal, request)
	if err != nil {
		return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimIntent, request.BeadID, traceRef, nil, denialForClaimRule(ruleClaimInvalid, "", traceRef))
	}
	release := func() {}
	if s.barrier != nil {
		release, err = s.barrier.Enter(ctx, principal.TenantID, principal.ProjectID)
		if err != nil {
			return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimIntent, request.BeadID, traceRef, nil, newDenial(authorityv1.ErrorAuthorityDown, ruleAuthorityUnavailable, "", requiredReconciliation, allowedSagaRead, traceRef))
		}
	}
	defer release()
	releaseWork := func() {}
	if s.workLocks != nil {
		releaseWork, err = s.workLocks.EnterWork(ctx, principal.TenantID, principal.ProjectID, request.BeadID)
		if err != nil {
			return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimIntent, request.BeadID, traceRef, nil, newDenial(authorityv1.ErrorAuthorityDown, ruleAuthorityUnavailable, "", requiredReconciliation, allowedSagaRead, traceRef))
		}
	}
	defer releaseWork()
	existing, found, err := s.sagas.Lookup(ctx, principal.TenantID, principal.ProjectID, request.IdempotencyKey)
	if err != nil {
		return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimIntent, request.BeadID, traceRef, nil, newDenial(authorityv1.ErrorAuthorityDown, ruleAuthorityUnavailable, "", requiredReconciliation, allowedSagaRead, traceRef))
	}
	if found {
		if existing.RequestDigest != requestDigest {
			return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimIntent, request.BeadID, traceRef, nil, denialForClaimRule(ruleClaimIdempotency, "", traceRef))
		}
		if !validSagaIntentCore(existing.Intent, principal, request, requestDigest) {
			return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimIntent, request.BeadID, traceRef, nil, denialForClaimRule(ruleClaimSagaInvalid, existing.Work.LifecycleState, traceRef))
		}
		if existing.Phase == claimPhaseCanonical || existing.Phase == claimPhaseComplete {
			if !validStoredClaim(existing.Work, principal, request) {
				return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimIntent, request.BeadID, traceRef, nil, denialForClaimRule(ruleClaimSagaInvalid, existing.Work.LifecycleState, traceRef))
			}
			current, currentOK := s.revalidateStoredClaim(ctx, principal, request, existing.Work)
			if !currentOK {
				return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimReceipt, request.BeadID, traceRef, nil, newDenial(authorityv1.ErrorUnknownEffect, ruleClaimUnknown, existing.Work.LifecycleState, requiredReconciliation, allowedSagaRead, traceRef))
			}
			existing.Work = current
			labels, labelDenial := admittedLabels(principal.Labels, current.Labels, request.ProposedLabels, current.LifecycleState, traceRef)
			if labelDenial != nil {
				return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimIntent, request.BeadID, traceRef, labels, labelDenial)
			}
			if !equalLabels(existing.Intent.Labels, labels) {
				return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimIntent, request.BeadID, traceRef, labels, denialForClaimRule(ruleClaimSagaInvalid, current.LifecycleState, traceRef))
			}
			if existing.Phase == claimPhaseComplete {
				return s.replayCompletedClaim(ctx, principal, request, labels, existing)
			}
			return s.finishCanonicalClaim(ctx, principal, request, labels, requestDigest, existing)
		}
		if existing.Phase != claimPhaseIntent {
			return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimIntent, request.BeadID, traceRef, nil, denialForClaimRule(ruleClaimSagaInvalid, existing.Work.LifecycleState, traceRef))
		}
	}

	item, err := s.claims.Get(ctx, principal.TenantID, principal.ProjectID, request.BeadID)
	if err != nil {
		if errors.Is(err, ErrWorkNotFound) {
			return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimIntent, request.BeadID, traceRef, nil, newDenial(authorityv1.ErrorNotFound, ruleNotFound, "", requiredValidRequest, allowedRetryRead, traceRef))
		}
		return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimIntent, request.BeadID, traceRef, nil, newDenial(authorityv1.ErrorAuthorityDown, ruleAuthorityUnavailable, "", requiredCanonicalProjection, allowedRetryRead, traceRef))
	}
	item = normalizeWorkItem(item)
	if item.BeadID != request.BeadID || item.TenantID != principal.TenantID || item.ProjectID != principal.ProjectID {
		return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimIntent, request.BeadID, traceRef, nil, newDenial(authorityv1.ErrorTenantMismatch, ruleTenantMismatch, item.LifecycleState, requiredCorrectTenant, allowedSelectProject, traceRef))
	}
	if invalid := projectionRule(item); invalid != "" {
		return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimIntent, item.BeadID, traceRef, item.Labels, denialForProjection(invalid, item.LifecycleState, traceRef))
	}
	labels, labelDenial := admittedLabels(principal.Labels, item.Labels, request.ProposedLabels, item.LifecycleState, traceRef)
	if labelDenial != nil {
		return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimIntent, item.BeadID, traceRef, labels, labelDenial)
	}
	if found && existing.Phase == claimPhaseIntent && item.LifecycleState == authorityv1.LifecycleInProgress && item.ClaimAttemptID == request.AttemptID {
		return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimReceipt, item.BeadID, traceRef, labels, newDenial(authorityv1.ErrorUnknownEffect, ruleClaimUnknown, item.LifecycleState, requiredReconciliation, allowedSagaRead, traceRef))
	}
	if !equalWorkVersion(item.Version, request.ExpectedVersion) || item.Integrity != request.ExpectedIntegrity {
		return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimIntent, item.BeadID, traceRef, labels, denialForClaimRule(ruleClaimStale, item.LifecycleState, traceRef))
	}
	if !equalStrings(item.ExclusivePaths, request.ExclusivePaths) {
		return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimIntent, item.BeadID, traceRef, labels, denialForClaimRule(ruleClaimPaths, item.LifecycleState, traceRef))
	}
	if rules := readinessRules(item); len(rules) > 0 {
		return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimIntent, item.BeadID, traceRef, labels, newDenial(authorityv1.ErrorNotReady, ruleClaimNotReady+":"+rules[0], item.LifecycleState, requiredReadyWork, allowedActionForReadinessRule(rules[0]), traceRef))
	}

	intent := ClaimIntent{
		RequestDigest: requestDigest, TenantID: principal.TenantID, ProjectID: principal.ProjectID,
		BeadID: request.BeadID, AttemptID: request.AttemptID, IdempotencyKey: request.IdempotencyKey,
		BaseSHA: request.BaseSHA, Capability: request.Capability, ExclusivePaths: append([]string(nil), request.ExclusivePaths...),
		TraceRef: traceRef, Labels: append([]authorityv1.Label(nil), labels...),
	}
	saga, err := s.sagas.Begin(ctx, intent)
	if err != nil {
		if errors.Is(err, ErrIdempotencyConflict) {
			return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimIntent, item.BeadID, traceRef, labels, denialForClaimRule(ruleClaimIdempotency, item.LifecycleState, traceRef))
		}
		return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimIntent, item.BeadID, traceRef, labels, newDenial(authorityv1.ErrorAuthorityDown, ruleAuthorityUnavailable, item.LifecycleState, requiredReconciliation, allowedSagaRead, traceRef))
	}
	if saga.RequestDigest != requestDigest || !validSagaIntentCore(saga.Intent, principal, request, requestDigest) || !equalLabels(saga.Intent.Labels, labels) {
		return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimIntent, item.BeadID, traceRef, labels, denialForClaimRule(ruleClaimSagaInvalid, item.LifecycleState, traceRef))
	}
	if saga.Phase != claimPhaseIntent && saga.Phase != claimPhaseCanonical {
		if saga.Phase != claimPhaseComplete {
			return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimIntent, item.BeadID, traceRef, labels, denialForClaimRule(ruleClaimSagaInvalid, item.LifecycleState, traceRef))
		}
	}
	if saga.Phase == claimPhaseCanonical || saga.Phase == claimPhaseComplete {
		current, currentOK := s.revalidateStoredClaim(ctx, principal, request, saga.Work)
		if !currentOK {
			return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimReceipt, request.BeadID, traceRef, labels, newDenial(authorityv1.ErrorUnknownEffect, ruleClaimUnknown, saga.Work.LifecycleState, requiredReconciliation, allowedSagaRead, traceRef))
		}
		saga.Work = current
		labels, labelDenial = admittedLabels(principal.Labels, current.Labels, request.ProposedLabels, current.LifecycleState, traceRef)
		if labelDenial != nil {
			return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimReceipt, request.BeadID, traceRef, labels, labelDenial)
		}
		if saga.Phase == claimPhaseComplete {
			return s.replayCompletedClaim(ctx, principal, request, labels, saga)
		}
	}

	if saga.Phase == claimPhaseIntent {
		if err := s.appendClaimEvent(ctx, principal, request, labels, operationClaimIntent, outcomeIntent, ruleClaimAllowed, "intent"); err != nil {
			return authorityv1.ClaimResponse{}, err
		}
		if err := s.appendClaimEvent(ctx, principal, request, labels, operationClaimPolicy, outcomePolicyAllowed, ruleClaimAllowed, "policy"); err != nil {
			return authorityv1.ClaimResponse{}, err
		}
		post, claimErr := s.claims.CompareAndSwapClaim(ctx, ClaimMutation{
			TenantID: principal.TenantID, ProjectID: principal.ProjectID, BeadID: request.BeadID,
			ExpectedVersion: request.ExpectedVersion, ExpectedIntegrity: request.ExpectedIntegrity,
			AttemptID: request.AttemptID, Assignee: principal.ProfileID, IdempotencyKey: request.IdempotencyKey, BaseSHA: request.BaseSHA,
		})
		if claimErr != nil {
			if errors.Is(claimErr, ErrStaleWorkVersion) {
				return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimReceipt, request.BeadID, traceRef, labels, denialForClaimRule(ruleClaimStale, item.LifecycleState, traceRef))
			}
			return authorityv1.ClaimResponse{}, s.unknownAfterEffect(ctx, principal, request, labels, newDenial(authorityv1.ErrorUnknownEffect, ruleClaimUnknown, item.LifecycleState, requiredReconciliation, allowedSagaRead, traceRef))
		}
		post = normalizeWorkItem(post)
		if !validClaimPostimage(post, item, principal, request) {
			return authorityv1.ClaimResponse{}, s.unknownAfterEffect(ctx, principal, request, labels, newDenial(authorityv1.ErrorUnknownEffect, ruleClaimPostimage, post.LifecycleState, requiredReconciliation, allowedSagaRead, traceRef))
		}
		saga, err = s.sagas.MarkCanonicalClaimed(ctx, principal.TenantID, principal.ProjectID, request.IdempotencyKey, requestDigest, post)
		if err != nil || saga.RequestDigest != requestDigest || saga.Phase != claimPhaseCanonical {
			return authorityv1.ClaimResponse{}, s.unknownAfterEffect(ctx, principal, request, labels, newDenial(authorityv1.ErrorUnknownEffect, ruleClaimUnknown, post.LifecycleState, requiredReconciliation, allowedSagaRead, traceRef))
		}
	}

	return s.finishCanonicalClaim(ctx, principal, request, labels, requestDigest, saga)
}

func (s *Service) finishCanonicalClaim(ctx context.Context, principal authorityv1.Principal, request authorityv1.ClaimRequest, labels []authorityv1.Label, requestDigest string, saga ClaimSaga) (authorityv1.ClaimResponse, error) {
	leaseRequest := LeaseRequest{
		RequestDigest: requestDigest, TenantID: principal.TenantID, ProjectID: principal.ProjectID,
		BeadID: request.BeadID, AttemptID: request.AttemptID, CanonicalClaimAttemptID: saga.Work.ClaimAttemptID, BaseSHA: request.BaseSHA,
		IdempotencyKey: request.IdempotencyKey, Capability: request.Capability, ExclusivePaths: append([]string(nil), request.ExclusivePaths...), Labels: append([]authorityv1.Label(nil), labels...),
		ClaimVersion: saga.Work.Version, MaximumExpiry: s.now().UTC().Add(maximumInitialLeaseDuration),
	}
	completed, err := s.sagas.IssueLease(ctx, request.IdempotencyKey, requestDigest, leaseRequest)
	if err != nil {
		return authorityv1.ClaimResponse{}, s.unknownAfterEffect(ctx, principal, request, labels, newDenial(authorityv1.ErrorUnknownEffect, ruleClaimUnknown, saga.Work.LifecycleState, requiredReconciliation, allowedSagaRead, request.TraceRef))
	}
	if completed.RequestDigest != requestDigest || completed.Phase != claimPhaseComplete || !validClaimLease(completed.Lease, leaseRequest, s.now().UTC()) || !validID(completed.ReceiptRef) {
		return authorityv1.ClaimResponse{}, s.unknownAfterEffect(ctx, principal, request, labels, newDenial(authorityv1.ErrorUnknownEffect, ruleLeaseInvalid, completed.Work.LifecycleState, requiredValidLease, allowedSagaRead, request.TraceRef))
	}
	if err := s.appendClaimEvent(ctx, principal, request, labels, operationClaimReceipt, outcomeVerified, ruleClaimAllowed, "receipt", claimEventEvidenceFor(request, completed.Work, completed.Lease)); err != nil {
		return authorityv1.ClaimResponse{}, newDenial(authorityv1.ErrorUnknownEffect, ruleClaimUnknown, completed.Work.LifecycleState, requiredReconciliation, allowedSagaRead, request.TraceRef)
	}
	return authorityv1.ClaimResponse{Work: completed.Work, Lease: completed.Lease, ReceiptRef: completed.ReceiptRef}, nil
}

func (s *Service) replayCompletedClaim(ctx context.Context, principal authorityv1.Principal, request authorityv1.ClaimRequest, labels []authorityv1.Label, saga ClaimSaga) (authorityv1.ClaimResponse, error) {
	leaseRequest := LeaseRequest{
		RequestDigest: saga.RequestDigest, TenantID: principal.TenantID, ProjectID: principal.ProjectID,
		BeadID: request.BeadID, AttemptID: request.AttemptID, CanonicalClaimAttemptID: saga.Work.ClaimAttemptID, BaseSHA: request.BaseSHA,
		IdempotencyKey: request.IdempotencyKey, Capability: request.Capability, ExclusivePaths: request.ExclusivePaths, Labels: append([]authorityv1.Label(nil), labels...),
		ClaimVersion: saga.Work.Version, MaximumExpiry: saga.Lease.ExpiresAt,
	}
	if !validClaimLease(saga.Lease, leaseRequest, s.now().UTC()) || !validID(saga.ReceiptRef) {
		return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimReceipt, request.BeadID, request.TraceRef, labels, newDenial(authorityv1.ErrorUnknownEffect, ruleLeaseInvalid, saga.Work.LifecycleState, requiredValidLease, allowedSagaRead, request.TraceRef))
	}
	if err := s.appendClaimEvent(ctx, principal, request, labels, operationClaimReceipt, outcomeVerified, ruleClaimAllowed, "receipt", claimEventEvidenceFor(request, saga.Work, saga.Lease)); err != nil {
		return authorityv1.ClaimResponse{}, err
	}
	return authorityv1.ClaimResponse{Work: saga.Work, Lease: saga.Lease, Replayed: true, ReceiptRef: saga.ReceiptRef}, nil
}

func validateClaimRequest(request authorityv1.ClaimRequest) string {
	if !validID(request.BeadID) || !validID(request.AttemptID) || !validID(request.IdempotencyKey) || !commitSHA.MatchString(request.BaseSHA) || !validIntegrity(request.ExpectedIntegrity) {
		return ruleClaimInvalid
	}
	if !validID(request.ExpectedVersion.AuthorityGeneration) || !validID(request.ExpectedVersion.IssueIncarnation) || request.ExpectedVersion.IssueMutationSequence == 0 || request.ExpectedVersion.DependencyGraphRevision == 0 {
		return ruleClaimInvalid
	}
	if request.Capability != authorityv1.CapabilityTicketDelivery {
		return ruleClaimCapability
	}
	if len(request.ExclusivePaths) == 0 || len(request.ExclusivePaths) > maxProjectionValues || hasDuplicateStrings(request.ExclusivePaths) {
		return ruleClaimPaths
	}
	for _, exclusivePath := range request.ExclusivePaths {
		if !safeExclusivePath(exclusivePath) {
			return ruleClaimPaths
		}
	}
	return ""
}

func denialForClaimRule(rule string, state authorityv1.LifecycleState, traceRef string) *authorityv1.Denial {
	switch rule {
	case ruleClaimStale:
		return newDenial(authorityv1.ErrorStaleVersion, rule, state, requiredFreshObservation, allowedReadyRead, traceRef)
	case ruleClaimNotReady:
		return newDenial(authorityv1.ErrorNotReady, rule, state, requiredReadyWork, allowedReadyRead, traceRef)
	case ruleClaimPaths:
		return newDenial(authorityv1.ErrorPolicyDenied, rule, state, requiredExactPaths, allowedReadyRead, traceRef)
	case ruleClaimCapability:
		return newDenial(authorityv1.ErrorPolicyDenied, rule, state, requiredDeliveryLease, allowedClaimRetry, traceRef)
	case ruleClaimIdempotency:
		return newDenial(authorityv1.ErrorPolicyDenied, rule, state, requiredNewIdempotency, allowedSagaRead, traceRef)
	case ruleClaimSagaInvalid:
		return newDenial(authorityv1.ErrorUnknownEffect, rule, state, requiredReconciliation, allowedSagaRead, traceRef)
	default:
		return newDenial(authorityv1.ErrorInvalidRequest, ruleClaimInvalid, state, requiredValidRequest, allowedClaimRetry, traceRef)
	}
}

func validClaimPostimage(post, pre authorityv1.WorkItem, principal authorityv1.Principal, request authorityv1.ClaimRequest) bool {
	return post.TenantID == pre.TenantID && post.ProjectID == pre.ProjectID && post.BeadID == pre.BeadID && post.DisplayID == pre.DisplayID && post.NativeStatus == "in_progress" && post.LifecycleState == authorityv1.LifecycleInProgress && post.Assignee == principal.ProfileID && post.ClaimAttemptID == request.AttemptID && post.Version.AuthorityGeneration == pre.Version.AuthorityGeneration && post.Version.IssueIncarnation == pre.Version.IssueIncarnation && post.Version.IssueMutationSequence > pre.Version.IssueMutationSequence && post.Version.DependencyGraphRevision == pre.Version.DependencyGraphRevision && equalStrings(post.GoalIDs, pre.GoalIDs) && equalStrings(post.ProductDecisionIDs, pre.ProductDecisionIDs) && post.FeatureID == pre.FeatureID && equalStrings(post.ScenarioIDs, pre.ScenarioIDs) && equalStrings(post.ExclusivePaths, pre.ExclusivePaths) && equalStrings(post.VerificationOrder, pre.VerificationOrder) && equalStrings(post.Blockers, pre.Blockers) && equalDependencies(post.Dependencies, pre.Dependencies) && equalLabels(post.Labels, pre.Labels) && projectionRule(post) == ""
}

func validStoredClaim(work authorityv1.WorkItem, principal authorityv1.Principal, request authorityv1.ClaimRequest) bool {
	work = normalizeWorkItem(work)
	return work.TenantID == principal.TenantID && work.ProjectID == principal.ProjectID && work.BeadID == request.BeadID && work.NativeStatus == "in_progress" && work.LifecycleState == authorityv1.LifecycleInProgress && work.Assignee == principal.ProfileID && work.ClaimAttemptID == request.AttemptID && projectionRule(work) == ""
}

func (s *Service) revalidateStoredClaim(ctx context.Context, principal authorityv1.Principal, request authorityv1.ClaimRequest, recorded authorityv1.WorkItem) (authorityv1.WorkItem, bool) {
	current, err := s.claims.Get(ctx, principal.TenantID, principal.ProjectID, request.BeadID)
	if err != nil {
		return authorityv1.WorkItem{}, false
	}
	current = normalizeWorkItem(current)
	recorded = normalizeWorkItem(recorded)
	if !validStoredClaim(current, principal, request) || !equalWorkItems(current, recorded) {
		return authorityv1.WorkItem{}, false
	}
	return current, true
}

func equalWorkItems(left, right authorityv1.WorkItem) bool {
	return left.TenantID == right.TenantID && left.ProjectID == right.ProjectID && left.BeadID == right.BeadID && left.DisplayID == right.DisplayID && left.NativeStatus == right.NativeStatus && left.LifecycleState == right.LifecycleState && left.Assignee == right.Assignee && left.ClaimAttemptID == right.ClaimAttemptID && equalStrings(left.GoalIDs, right.GoalIDs) && equalStrings(left.ProductDecisionIDs, right.ProductDecisionIDs) && left.FeatureID == right.FeatureID && equalStrings(left.ScenarioIDs, right.ScenarioIDs) && equalStrings(left.ExclusivePaths, right.ExclusivePaths) && equalStrings(left.VerificationOrder, right.VerificationOrder) && equalStrings(left.Blockers, right.Blockers) && equalDependencies(left.Dependencies, right.Dependencies) && equalLabels(left.Labels, right.Labels) && left.Version == right.Version && left.Integrity == right.Integrity
}

func validClaimLease(lease authorityv1.CapabilityLease, request LeaseRequest, now time.Time) bool {
	return validID(lease.LeaseID) && lease.TenantID == request.TenantID && lease.ProjectID == request.ProjectID && lease.BeadID == request.BeadID && lease.AttemptID == request.AttemptID && lease.CanonicalClaimAttemptID == request.CanonicalClaimAttemptID && validID(lease.CanonicalClaimAttemptID) && lease.IdempotencyKey == request.IdempotencyKey && validID(lease.FenceGeneration) && lease.LeaseEpoch > 0 && lease.ClaimVersion == request.ClaimVersion && lease.BaseSHA == request.BaseSHA && lease.Capability == request.Capability && equalStrings(lease.ExclusivePaths, request.ExclusivePaths) && equalLabels(lease.Labels, request.Labels) && lease.State == authorityv1.LeaseActive && lease.Active && !lease.IssuedAt.After(now) && lease.ExpiresAt.After(now) && !lease.ExpiresAt.After(request.MaximumExpiry)
}

func equalWorkVersion(left, right authorityv1.WorkVersion) bool { return left == right }

func equalStrings(left, right []string) bool {
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

func equalDependencies(left, right []authorityv1.Dependency) bool {
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

func validSagaIntentCore(intent ClaimIntent, principal authorityv1.Principal, request authorityv1.ClaimRequest, requestDigest string) bool {
	return intent.RequestDigest == requestDigest && intent.TenantID == principal.TenantID && intent.ProjectID == principal.ProjectID && intent.BeadID == request.BeadID && intent.AttemptID == request.AttemptID && intent.IdempotencyKey == request.IdempotencyKey && intent.BaseSHA == request.BaseSHA && intent.Capability == request.Capability && equalStrings(intent.ExclusivePaths, request.ExclusivePaths)
}

func claimRequestDigest(principal authorityv1.Principal, request authorityv1.ClaimRequest) (string, error) {
	normalized := struct {
		TenantID          string                       `json:"tenant_id"`
		ProjectID         string                       `json:"project_id"`
		PrincipalID       string                       `json:"principal_id"`
		ProfileID         string                       `json:"profile_id"`
		BeadID            string                       `json:"bead_id"`
		ExpectedVersion   authorityv1.WorkVersion      `json:"expected_version"`
		ExpectedIntegrity authorityv1.IntegrityDigests `json:"expected_integrity"`
		AttemptID         string                       `json:"attempt_id"`
		BaseSHA           string                       `json:"base_sha"`
		ExclusivePaths    []string                     `json:"exclusive_paths"`
		Capability        authorityv1.Capability       `json:"capability"`
		IdempotencyKey    string                       `json:"idempotency_key"`
		Labels            []authorityv1.Label          `json:"labels"`
	}{
		TenantID: principal.TenantID, ProjectID: principal.ProjectID, PrincipalID: principal.PrincipalID, ProfileID: principal.ProfileID,
		BeadID: request.BeadID, ExpectedVersion: request.ExpectedVersion, ExpectedIntegrity: request.ExpectedIntegrity,
		AttemptID: request.AttemptID, BaseSHA: request.BaseSHA, ExclusivePaths: append([]string(nil), request.ExclusivePaths...),
		Capability: request.Capability, IdempotencyKey: request.IdempotencyKey, Labels: append([]authorityv1.Label(nil), request.ProposedLabels...),
	}
	sort.Strings(normalized.ExclusivePaths)
	sort.Slice(normalized.Labels, func(i, j int) bool { return normalized.Labels[i] < normalized.Labels[j] })
	data, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}

type claimEventEvidence struct {
	version    authorityv1.WorkVersion
	leaseEpoch uint64
	beforeHash string
	afterHash  string
}

func (s *Service) appendClaimEvent(ctx context.Context, principal authorityv1.Principal, request authorityv1.ClaimRequest, labels []authorityv1.Label, operation, outcome, rule, phase string, evidence ...claimEventEvidence) error {
	event := s.event(principal, operation, request.BeadID, request.TraceRef, outcome, rule, labels)
	event.AttemptID = request.AttemptID
	event.IdempotencyKey = request.IdempotencyKey
	event.EventID = deterministicEventID(request.IdempotencyKey, phase)
	if len(evidence) == 1 {
		event.CanonicalVersion = &evidence[0].version
		event.LeaseEpoch = evidence[0].leaseEpoch
		event.BeforeHash = evidence[0].beforeHash
		event.AfterHash = evidence[0].afterHash
	}
	receipt, err := s.events.Append(ctx, event)
	if err != nil || !validEventReceipt(event, receipt) {
		return newDenial(authorityv1.ErrorAuthorityDown, ruleAuthorityUnavailable, "", requiredReconciliation, allowedSagaRead, request.TraceRef)
	}
	return nil
}

func claimEventEvidenceFor(request authorityv1.ClaimRequest, work authorityv1.WorkItem, lease authorityv1.CapabilityLease) claimEventEvidence {
	return claimEventEvidence{
		version: work.Version, leaseEpoch: lease.LeaseEpoch,
		beforeHash: deterministicJSONDigest(request.ExpectedIntegrity),
		afterHash:  deterministicJSONDigest(work),
	}
}

func deterministicJSONDigest(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func (s *Service) unknownAfterEffect(ctx context.Context, principal authorityv1.Principal, request authorityv1.ClaimRequest, labels []authorityv1.Label, denial *authorityv1.Denial) error {
	event := s.event(principal, operationClaimReceipt, request.BeadID, request.TraceRef, outcomeUnknown, denial.Rule, labels)
	event.AttemptID = request.AttemptID
	event.IdempotencyKey = request.IdempotencyKey
	event.EventID = deterministicEventID(request.IdempotencyKey, "unknown:"+denial.Rule)
	_, _ = s.events.Append(ctx, event)
	return denial
}

func deterministicEventID(idempotencyKey, phase string) string {
	digest := sha256.Sum256([]byte(idempotencyKey + "\x00" + phase))
	return "evt-" + hex.EncodeToString(digest[:])
}
