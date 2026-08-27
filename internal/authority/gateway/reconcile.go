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

	authorityv1 "github.com/greaveselliott/MARS-3/api/authority/v1"
)

const (
	operationClaimReconcileIntent = "work.claim.reconcile.intent"
	operationClaimReconcilePolicy = "work.claim.reconcile.policy"
	ruleClaimReconcileAllowed     = "authority.claim.reconcile.allowed"
	requiredReconcileCapability   = "obtain policy-approved lease.issue capability"
	allowedReconcilePolicy        = "policy.request(lease.issue)"
)

// ClaimReconciliationRequest separates the immutable canonical claim attempt
// from the current execution attempt that owns the new lease. A ticket can
// have many run attempts without rewriting the original Beads claim.
type ClaimReconciliationRequest struct {
	authorityv1.ClaimRequest
	CanonicalClaimAttemptID string
}

// ReconcileClaimedWork reconstructs only the operational saga and lease for a
// canonical claim that already exists. It never invokes the Beads mutator. The
// route is control-plane-only and intentionally absent from the public HTTP
// handler; it is used for verified partial-outcome recovery and W-001's first
// self-host development lease.
func (s *Service) ReconcileClaimedWork(ctx context.Context, principal authorityv1.Principal, reconciliation ClaimReconciliationRequest) (authorityv1.ClaimResponse, error) {
	request := reconciliation.ClaimRequest
	traceRef, denial := admitRequest(principal, request.TraceRef, request.ProposedLabels)
	if denial != nil {
		return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimReconcileIntent, "", traceRef, nil, denial)
	}
	if !hasCapability(principal, authorityv1.CapabilityWorkClaim) || !hasCapability(principal, authorityv1.CapabilityLeaseIssue) {
		return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimReconcileIntent, request.BeadID, traceRef, nil,
			newDenial(authorityv1.ErrorUnauthorized, ruleCapabilityMissing, "", requiredReconcileCapability, allowedReconcilePolicy, traceRef))
	}
	if s.claims == nil || s.sagas == nil {
		return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimReconcileIntent, request.BeadID, traceRef, nil,
			newDenial(authorityv1.ErrorAuthorityDown, ruleAuthorityUnavailable, "", requiredCanonicalProjection, allowedSagaRead, traceRef))
	}
	request.ExclusivePaths = sortedStrings(request.ExclusivePaths)
	request.ProposedLabels = sortedLabels(request.ProposedLabels)
	if claimRule := validateClaimRequest(request); claimRule != "" {
		return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimReconcileIntent, request.BeadID, traceRef, nil, denialForClaimRule(claimRule, "", traceRef))
	}
	requestDigest, err := claimRequestDigest(principal, request)
	if err != nil {
		return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimReconcileIntent, request.BeadID, traceRef, nil, denialForClaimRule(ruleClaimInvalid, "", traceRef))
	}
	release := func() {}
	if s.barrier != nil {
		release, err = s.barrier.Enter(ctx, principal.TenantID, principal.ProjectID)
		if err != nil {
			return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimReconcileIntent, request.BeadID, traceRef, nil,
				newDenial(authorityv1.ErrorAuthorityDown, ruleAuthorityUnavailable, "", requiredReconciliation, allowedSagaRead, traceRef))
		}
	}
	defer release()
	releaseWork := func() {}
	if s.workLocks != nil {
		releaseWork, err = s.workLocks.EnterWork(ctx, principal.TenantID, principal.ProjectID, request.BeadID)
		if err != nil {
			return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimReconcileIntent, request.BeadID, traceRef, nil,
				newDenial(authorityv1.ErrorAuthorityDown, ruleAuthorityUnavailable, "", requiredReconciliation, allowedSagaRead, traceRef))
		}
	}
	defer releaseWork()

	current, err := s.claims.Get(ctx, principal.TenantID, principal.ProjectID, request.BeadID)
	if err != nil {
		return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimReconcileIntent, request.BeadID, traceRef, nil,
			newDenial(authorityv1.ErrorAuthorityDown, ruleAuthorityUnavailable, "", requiredCanonicalProjection, allowedSagaRead, traceRef))
	}
	current = normalizeWorkItem(current)
	if current.TenantID != principal.TenantID || current.ProjectID != principal.ProjectID || current.BeadID != request.BeadID {
		return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimReconcileIntent, request.BeadID, traceRef, nil,
			newDenial(authorityv1.ErrorTenantMismatch, ruleTenantMismatch, current.LifecycleState, requiredCorrectTenant, allowedSelectProject, traceRef))
	}
	if invalid := projectionRule(current); invalid != "" {
		return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimReconcileIntent, request.BeadID, traceRef, current.Labels, denialForProjection(invalid, current.LifecycleState, traceRef))
	}
	labels, labelDenial := admittedLabels(principal.Labels, current.Labels, request.ProposedLabels, current.LifecycleState, traceRef)
	if labelDenial != nil {
		return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimReconcileIntent, request.BeadID, traceRef, labels, labelDenial)
	}
	if !equalWorkVersion(current.Version, request.ExpectedVersion) || current.Integrity != request.ExpectedIntegrity {
		return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimReconcileIntent, request.BeadID, traceRef, labels, denialForClaimRule(ruleClaimStale, current.LifecycleState, traceRef))
	}
	if !equalStrings(current.ExclusivePaths, request.ExclusivePaths) {
		return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimReconcileIntent, request.BeadID, traceRef, labels, denialForClaimRule(ruleClaimPaths, current.LifecycleState, traceRef))
	}
	if !validReconciliationClaim(current, principal, request, reconciliation.CanonicalClaimAttemptID) {
		return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimReconcileIntent, request.BeadID, traceRef, labels,
			newDenial(authorityv1.ErrorPolicyDenied, ruleClaimSagaInvalid, current.LifecycleState, requiredReconciliation, allowedSagaRead, traceRef))
	}

	intent := ClaimIntent{
		RequestDigest: requestDigest, TenantID: principal.TenantID, ProjectID: principal.ProjectID,
		BeadID: request.BeadID, AttemptID: request.AttemptID, IdempotencyKey: request.IdempotencyKey,
		BaseSHA: request.BaseSHA, Capability: request.Capability, ExclusivePaths: append([]string(nil), request.ExclusivePaths...),
		TraceRef: traceRef, Labels: append([]authorityv1.Label(nil), labels...),
	}
	saga, found, err := s.sagas.Lookup(ctx, principal.TenantID, principal.ProjectID, request.IdempotencyKey)
	if err != nil {
		return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimReconcileIntent, request.BeadID, traceRef, labels,
			newDenial(authorityv1.ErrorAuthorityDown, ruleAuthorityUnavailable, current.LifecycleState, requiredReconciliation, allowedSagaRead, traceRef))
	}
	if found {
		if saga.RequestDigest != requestDigest || !validSagaIntentCore(saga.Intent, principal, request, requestDigest) || !equalLabels(saga.Intent.Labels, labels) {
			return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimReconcileIntent, request.BeadID, traceRef, labels, denialForClaimRule(ruleClaimIdempotency, current.LifecycleState, traceRef))
		}
	} else {
		saga, err = s.sagas.Begin(ctx, intent)
		if err != nil {
			if errors.Is(err, ErrIdempotencyConflict) {
				return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimReconcileIntent, request.BeadID, traceRef, labels, denialForClaimRule(ruleClaimIdempotency, current.LifecycleState, traceRef))
			}
			return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimReconcileIntent, request.BeadID, traceRef, labels,
				newDenial(authorityv1.ErrorAuthorityDown, ruleAuthorityUnavailable, current.LifecycleState, requiredReconciliation, allowedSagaRead, traceRef))
		}
	}
	if saga.RequestDigest != requestDigest || !validSagaIntentCore(saga.Intent, principal, request, requestDigest) || !equalLabels(saga.Intent.Labels, labels) {
		return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimReconcileIntent, request.BeadID, traceRef, labels, denialForClaimRule(ruleClaimSagaInvalid, current.LifecycleState, traceRef))
	}
	if saga.Phase == claimPhaseComplete {
		if !equalWorkItems(saga.Work, current) {
			return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimReconcileIntent, request.BeadID, traceRef, labels,
				newDenial(authorityv1.ErrorUnknownEffect, ruleClaimUnknown, current.LifecycleState, requiredReconciliation, allowedSagaRead, traceRef))
		}
		return s.replayCompletedClaim(ctx, principal, request, labels, saga)
	}
	if saga.Phase == claimPhaseIntent {
		if err := s.appendClaimEvent(ctx, principal, request, labels, operationClaimReconcileIntent, outcomeIntent, ruleClaimReconcileAllowed, "reconcile-intent"); err != nil {
			return authorityv1.ClaimResponse{}, err
		}
		if err := s.appendClaimEvent(ctx, principal, request, labels, operationClaimReconcilePolicy, outcomePolicyAllowed, ruleClaimReconcileAllowed, "reconcile-policy"); err != nil {
			return authorityv1.ClaimResponse{}, err
		}
		saga, err = s.sagas.MarkCanonicalClaimed(ctx, principal.TenantID, principal.ProjectID, request.IdempotencyKey, requestDigest, current)
		if err != nil || saga.RequestDigest != requestDigest || saga.Phase != claimPhaseCanonical || !equalWorkItems(saga.Work, current) {
			return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimReconcileIntent, request.BeadID, traceRef, labels,
				newDenial(authorityv1.ErrorUnknownEffect, ruleClaimUnknown, current.LifecycleState, requiredReconciliation, allowedSagaRead, traceRef))
		}
	}
	if saga.Phase != claimPhaseCanonical || !equalWorkItems(saga.Work, current) {
		return authorityv1.ClaimResponse{}, s.deny(ctx, principal, operationClaimReconcileIntent, request.BeadID, traceRef, labels,
			newDenial(authorityv1.ErrorUnknownEffect, ruleClaimUnknown, current.LifecycleState, requiredReconciliation, allowedSagaRead, traceRef))
	}
	return s.finishCanonicalClaim(ctx, principal, request, labels, requestDigest, saga)
}

func validReconciliationClaim(work authorityv1.WorkItem, principal authorityv1.Principal, request authorityv1.ClaimRequest, canonicalAttemptID string) bool {
	return validID(canonicalAttemptID) && work.TenantID == principal.TenantID && work.ProjectID == principal.ProjectID &&
		work.BeadID == request.BeadID && work.NativeStatus == "in_progress" && work.LifecycleState == authorityv1.LifecycleInProgress &&
		work.Assignee == principal.ProfileID && work.ClaimAttemptID == canonicalAttemptID && projectionRule(work) == ""
}
