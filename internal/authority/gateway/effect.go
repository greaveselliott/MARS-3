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
	"strings"
	"time"

	authorityv1 "github.com/greaveselliott/MARS-3/api/authority/v1"
)

const (
	operationEffectIntent  = "effect.validate.intent"
	operationEffectPolicy  = "effect.validate.policy"
	operationEffectReceipt = "effect.validate.receipt"
	ruleEffectAllowed      = "authority.effect.allowed"
	ruleEffectInvalid      = "authority.effect.invalid"
	ruleEffectPath         = "authority.effect.path_denied"
	ruleEffectFence        = "authority.effect.fence_denied"
	ruleEffectClaim        = "authority.effect.claim_changed"
	ruleEffectLabels       = "authority.effect.labels_changed"
	ruleEffectUnknown      = "authority.effect.unknown"
	requiredExactFence     = "read and present the complete current fencing tuple"
	requiredEffectPath     = "select one path inside the active lease"
	requiredStableClaim    = "re-read the current canonical claim and lease"
	allowedEffectRecheck   = "effect.validate"
)

// LeaseValidator owns the operational PostgreSQL half of pre-effect fencing.
type LeaseValidator interface {
	ValidateFence(context.Context, authorityv1.FencingTuple) (authorityv1.CapabilityLease, error)
}

// ValidateEffect verifies authority at the last safe point before a trusted
// broker performs one effect. The receipt cannot be reused as a capability.
func (s *Service) ValidateEffect(ctx context.Context, principal authorityv1.Principal, request authorityv1.EffectValidationRequest) (authorityv1.EffectValidation, error) {
	traceRef, denial := admitRequest(principal, request.TraceRef, request.ProposedLabels)
	if denial != nil {
		return authorityv1.EffectValidation{}, s.deny(ctx, principal, operationEffectIntent, "", traceRef, nil, denial)
	}
	if !hasCapability(principal, authorityv1.CapabilityEffectValidate) {
		return authorityv1.EffectValidation{}, s.deny(ctx, principal, operationEffectIntent, request.Fence.BeadID, traceRef, nil, newDenial(authorityv1.ErrorUnauthorized, ruleCapabilityMissing, "", "obtain policy-approved effect.validate capability", "policy.request(effect.validate)", traceRef))
	}
	if s.claims == nil || s.leases == nil {
		return authorityv1.EffectValidation{}, s.deny(ctx, principal, operationEffectIntent, request.Fence.BeadID, traceRef, nil, newDenial(authorityv1.ErrorAuthorityDown, ruleAuthorityUnavailable, "", requiredCanonicalProjection, allowedEffectRecheck, traceRef))
	}
	request.ProposedLabels = sortedLabels(request.ProposedLabels)
	request.Fence.ExclusivePaths = sortedStrings(request.Fence.ExclusivePaths)
	request.Fence.Labels = sortedLabels(request.Fence.Labels)
	if !validEffectRequest(principal, request) {
		return authorityv1.EffectValidation{}, s.deny(ctx, principal, operationEffectIntent, request.Fence.BeadID, traceRef, nil, newDenial(authorityv1.ErrorInvalidRequest, ruleEffectInvalid, "", requiredExactFence, allowedEffectRecheck, traceRef))
	}
	if !pathWithinLease(request.Path, request.Fence.ExclusivePaths) {
		return authorityv1.EffectValidation{}, s.deny(ctx, principal, operationEffectIntent, request.Fence.BeadID, traceRef, request.Fence.Labels, newDenial(authorityv1.ErrorPolicyDenied, ruleEffectPath, "", requiredEffectPath, allowedEffectRecheck, traceRef))
	}
	release := func() {}
	if s.barrier != nil {
		var barrierErr error
		release, barrierErr = s.barrier.Enter(ctx, principal.TenantID, principal.ProjectID)
		if barrierErr != nil {
			return authorityv1.EffectValidation{}, s.deny(ctx, principal, operationEffectIntent, request.Fence.BeadID, traceRef, request.Fence.Labels, newDenial(authorityv1.ErrorAuthorityDown, ruleAuthorityUnavailable, "", requiredStableClaim, allowedEffectRecheck, traceRef))
		}
	}
	defer release()

	first, err := s.claims.Get(ctx, principal.TenantID, principal.ProjectID, request.Fence.BeadID)
	if err != nil {
		return authorityv1.EffectValidation{}, s.deny(ctx, principal, operationEffectIntent, request.Fence.BeadID, traceRef, request.Fence.Labels, newDenial(authorityv1.ErrorAuthorityDown, ruleAuthorityUnavailable, "", requiredStableClaim, allowedEffectRecheck, traceRef))
	}
	first = normalizeWorkItem(first)
	if !workMatchesFence(first, principal, request.Fence) {
		return authorityv1.EffectValidation{}, s.deny(ctx, principal, operationEffectIntent, request.Fence.BeadID, traceRef, request.Fence.Labels, newDenial(authorityv1.ErrorPolicyDenied, ruleEffectClaim, first.LifecycleState, requiredStableClaim, allowedEffectRecheck, traceRef))
	}
	labels, labelDenial := admittedLabels(principal.Labels, first.Labels, request.ProposedLabels, first.LifecycleState, traceRef)
	if labelDenial != nil {
		return authorityv1.EffectValidation{}, s.deny(ctx, principal, operationEffectPolicy, request.Fence.BeadID, traceRef, labels, labelDenial)
	}
	if !equalLabels(labels, request.Fence.Labels) {
		return authorityv1.EffectValidation{}, s.deny(ctx, principal, operationEffectPolicy, request.Fence.BeadID, traceRef, labels, newDenial(authorityv1.ErrorPolicyDenied, ruleEffectLabels, first.LifecycleState, requiredExactFence, allowedEffectRecheck, traceRef))
	}
	if _, err := s.appendEffectEvent(ctx, principal, request, labels, operationEffectIntent, outcomeIntent, ruleEffectAllowed, "intent"); err != nil {
		return authorityv1.EffectValidation{}, err
	}
	if _, err := s.appendEffectEvent(ctx, principal, request, labels, operationEffectPolicy, outcomePolicyAllowed, ruleEffectAllowed, "policy"); err != nil {
		return authorityv1.EffectValidation{}, err
	}
	lease, err := s.leases.ValidateFence(ctx, request.Fence)
	if err != nil || !leaseMatchesEffectFence(lease, request.Fence, s.now().UTC()) {
		return authorityv1.EffectValidation{}, s.deny(ctx, principal, operationEffectReceipt, request.Fence.BeadID, traceRef, labels, newDenial(authorityv1.ErrorPolicyDenied, ruleEffectFence, first.LifecycleState, requiredExactFence, allowedEffectRecheck, traceRef))
	}
	second, err := s.claims.Get(ctx, principal.TenantID, principal.ProjectID, request.Fence.BeadID)
	if err != nil {
		return authorityv1.EffectValidation{}, s.deny(ctx, principal, operationEffectReceipt, request.Fence.BeadID, traceRef, labels, newDenial(authorityv1.ErrorUnknownEffect, ruleEffectUnknown, first.LifecycleState, requiredStableClaim, allowedEffectRecheck, traceRef))
	}
	second = normalizeWorkItem(second)
	if !equalWorkItems(first, second) || !workMatchesFence(second, principal, request.Fence) {
		return authorityv1.EffectValidation{}, s.deny(ctx, principal, operationEffectReceipt, request.Fence.BeadID, traceRef, labels, newDenial(authorityv1.ErrorPolicyDenied, ruleEffectClaim, second.LifecycleState, requiredStableClaim, allowedEffectRecheck, traceRef))
	}
	receiptEvent, err := s.appendEffectEvent(ctx, principal, request, labels, operationEffectReceipt, outcomeVerified, ruleEffectAllowed, "receipt")
	if err != nil {
		return authorityv1.EffectValidation{}, err
	}
	receiptRef := effectReceiptRef(request, receiptEvent.OccurredAt.String())
	return authorityv1.EffectValidation{Allowed: true, EffectID: request.EffectID, LeaseID: request.Fence.LeaseID, CheckedAt: receiptEvent.OccurredAt, ReceiptRef: receiptRef}, nil
}

func validEffectRequest(principal authorityv1.Principal, request authorityv1.EffectValidationRequest) bool {
	fence := request.Fence
	if !validID(request.EffectID) || !safeExclusivePath(request.Path) || fence.TenantID != principal.TenantID || fence.ProjectID != principal.ProjectID || !validID(fence.BeadID) || !validID(fence.AttemptID) || !validID(fence.CanonicalClaimAttemptID) || !validID(fence.IdempotencyKey) || !validID(fence.LeaseID) || !validID(fence.FenceGeneration) || fence.LeaseEpoch == 0 || !validID(fence.ClaimVersion.AuthorityGeneration) || !validID(fence.ClaimVersion.IssueIncarnation) || fence.ClaimVersion.IssueMutationSequence == 0 || fence.ClaimVersion.DependencyGraphRevision == 0 || !commitSHA.MatchString(fence.BaseSHA) || fence.Capability != authorityv1.CapabilityTicketDelivery || len(fence.ExclusivePaths) == 0 || len(fence.ExclusivePaths) > maxProjectionValues || hasDuplicateStrings(fence.ExclusivePaths) || len(fence.Labels) == 0 || len(fence.Labels) > maxProjectionValues || hasDuplicateLabels(fence.Labels) {
		return false
	}
	for _, exclusivePath := range fence.ExclusivePaths {
		if !safeExclusivePath(exclusivePath) {
			return false
		}
	}
	return true
}

func workMatchesFence(work authorityv1.WorkItem, principal authorityv1.Principal, fence authorityv1.FencingTuple) bool {
	return projectionRule(work) == "" && work.TenantID == fence.TenantID && work.ProjectID == fence.ProjectID && work.BeadID == fence.BeadID && work.NativeStatus == "in_progress" && work.LifecycleState == authorityv1.LifecycleInProgress && work.Assignee == principal.ProfileID && work.ClaimAttemptID == fence.CanonicalClaimAttemptID && work.Version == fence.ClaimVersion && equalStrings(work.ExclusivePaths, fence.ExclusivePaths)
}

func leaseMatchesEffectFence(lease authorityv1.CapabilityLease, fence authorityv1.FencingTuple, now time.Time) bool {
	return lease.Active && lease.State == authorityv1.LeaseActive && lease.ExpiresAt.After(now) && lease.TenantID == fence.TenantID && lease.ProjectID == fence.ProjectID && lease.BeadID == fence.BeadID && lease.AttemptID == fence.AttemptID && lease.CanonicalClaimAttemptID == fence.CanonicalClaimAttemptID && lease.IdempotencyKey == fence.IdempotencyKey && lease.LeaseID == fence.LeaseID && lease.FenceGeneration == fence.FenceGeneration && lease.LeaseEpoch == fence.LeaseEpoch && lease.ClaimVersion == fence.ClaimVersion && lease.BaseSHA == fence.BaseSHA && lease.Capability == fence.Capability && equalStrings(lease.ExclusivePaths, fence.ExclusivePaths) && equalLabels(lease.Labels, fence.Labels)
}

func pathWithinLease(candidate string, exclusivePaths []string) bool {
	if !safeExclusivePath(candidate) {
		return false
	}
	for _, allowed := range exclusivePaths {
		if candidate == allowed || (strings.HasSuffix(allowed, "/") && strings.HasPrefix(candidate, allowed)) {
			return true
		}
	}
	return false
}

func (s *Service) appendEffectEvent(ctx context.Context, principal authorityv1.Principal, request authorityv1.EffectValidationRequest, labels []authorityv1.Label, operation, outcome, rule, phase string) (authorityv1.Event, error) {
	event := s.event(principal, operation, request.Fence.BeadID, request.TraceRef, outcome, rule, labels)
	event.AttemptID = request.Fence.AttemptID
	event.IdempotencyKey = request.Fence.IdempotencyKey
	event.CanonicalVersion = &request.Fence.ClaimVersion
	event.LeaseEpoch = request.Fence.LeaseEpoch
	event.EventID = deterministicEventID(request.EffectID, phase)
	receipt, err := s.events.Append(ctx, event)
	if err != nil || !validEventReceipt(event, receipt) {
		return authorityv1.Event{}, newDenial(authorityv1.ErrorAuthorityDown, ruleAuthorityUnavailable, "", requiredStableClaim, allowedEffectRecheck, request.TraceRef)
	}
	return receipt, nil
}

func effectReceiptRef(request authorityv1.EffectValidationRequest, checkedAt string) string {
	digest := sha256.Sum256([]byte(request.EffectID + "\x00" + request.Fence.LeaseID + "\x00" + checkedAt))
	return "effect-receipt-" + hex.EncodeToString(digest[:])
}
