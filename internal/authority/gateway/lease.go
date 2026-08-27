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
	"time"

	authorityv1 "github.com/greaveselliott/MARS-3/api/authority/v1"
)

const (
	operationLeaseRenewIntent    = "lease.renew.intent"
	operationLeaseRenewPolicy    = "lease.renew.policy"
	operationLeaseRenewReceipt   = "lease.renew.receipt"
	operationLeaseReleaseIntent  = "lease.release.intent"
	operationLeaseReleasePolicy  = "lease.release.policy"
	operationLeaseReleaseReceipt = "lease.release.receipt"
	operationLeaseRevokeIntent   = "lease.revoke.intent"
	operationLeaseRevokePolicy   = "lease.revoke.policy"
	operationLeaseRevokeReceipt  = "lease.revoke.receipt"
	ruleLeaseLifecycleAllowed    = "authority.lease.lifecycle.allowed"
	ruleLeaseLifecycleInvalid    = "authority.lease.lifecycle.invalid"
	ruleLeaseLifecycleFence      = "authority.lease.lifecycle.fence_denied"
	ruleLeaseLifecycleClaim      = "authority.lease.lifecycle.claim_changed"
	ruleLeaseLifecycleUnknown    = "authority.lease.lifecycle.unknown"
	requiredLeaseLifecycleFence  = "read and present the complete current lease and canonical claim tuple"
	allowedLeaseLifecycleRetry   = "lease.read"
)

// LeaseStore is the sole operational lifecycle authority. Implementations
// persist lease state but never own canonical ticket lifecycle.
type LeaseStore interface {
	LeaseValidator
	GetLease(context.Context, string, string, string) (authorityv1.CapabilityLease, error)
	Renew(context.Context, authorityv1.FencingTuple, time.Time) (authorityv1.CapabilityLease, error)
	Release(context.Context, authorityv1.FencingTuple) (authorityv1.CapabilityLease, error)
	Revoke(context.Context, authorityv1.RevokeLeaseRequest) (authorityv1.CapabilityLease, error)
}

func (s *Service) RenewLease(ctx context.Context, principal authorityv1.Principal, request authorityv1.RenewLeaseRequest) (authorityv1.LeaseMutationResponse, error) {
	request.NewExpiry = request.NewExpiry.UTC()
	return s.mutateOwnedLease(ctx, principal, request.Fence, request.TraceRef, "renew", request.NewExpiry)
}

func (s *Service) ReleaseLease(ctx context.Context, principal authorityv1.Principal, request authorityv1.ReleaseLeaseRequest) (authorityv1.LeaseMutationResponse, error) {
	return s.mutateOwnedLease(ctx, principal, request.Fence, request.TraceRef, "release", time.Time{})
}

func (s *Service) mutateOwnedLease(ctx context.Context, principal authorityv1.Principal, fence authorityv1.FencingTuple, traceRef, action string, newExpiry time.Time) (authorityv1.LeaseMutationResponse, error) {
	traceRef, denial := admitRequest(principal, traceRef, nil)
	if denial != nil {
		return authorityv1.LeaseMutationResponse{}, s.deny(ctx, principal, leaseOperation(action, "intent"), "", traceRef, nil, denial)
	}
	wanted := authorityv1.CapabilityLeaseRenew
	if action == "release" {
		wanted = authorityv1.CapabilityLeaseRelease
	}
	if !hasCapability(principal, wanted) {
		return authorityv1.LeaseMutationResponse{}, s.deny(ctx, principal, leaseOperation(action, "intent"), fence.BeadID, traceRef, nil,
			newDenial(authorityv1.ErrorUnauthorized, ruleCapabilityMissing, "", "obtain policy-approved "+string(wanted)+" capability", "policy.request("+string(wanted)+")", traceRef))
	}
	if s.claims == nil || s.leaseOps == nil {
		return authorityv1.LeaseMutationResponse{}, s.deny(ctx, principal, leaseOperation(action, "intent"), fence.BeadID, traceRef, nil,
			newDenial(authorityv1.ErrorAuthorityDown, ruleAuthorityUnavailable, "", requiredLeaseLifecycleFence, allowedLeaseLifecycleRetry, traceRef))
	}
	fence.ExclusivePaths, fence.Labels = sortedStrings(fence.ExclusivePaths), sortedLabels(fence.Labels)
	if !validLifecycleFence(principal, fence) || action == "renew" && newExpiry.IsZero() || action != "renew" && action != "release" {
		return authorityv1.LeaseMutationResponse{}, s.deny(ctx, principal, leaseOperation(action, "intent"), fence.BeadID, traceRef, fence.Labels,
			newDenial(authorityv1.ErrorInvalidRequest, ruleLeaseLifecycleInvalid, "", requiredLeaseLifecycleFence, allowedLeaseLifecycleRetry, traceRef))
	}
	releaseBarrier := func() {}
	var err error
	if s.barrier != nil {
		releaseBarrier, err = s.barrier.Enter(ctx, principal.TenantID, principal.ProjectID)
		if err != nil {
			return authorityv1.LeaseMutationResponse{}, s.deny(ctx, principal, leaseOperation(action, "intent"), fence.BeadID, traceRef, fence.Labels,
				newDenial(authorityv1.ErrorAuthorityDown, ruleAuthorityUnavailable, "", requiredLeaseLifecycleFence, allowedLeaseLifecycleRetry, traceRef))
		}
	}
	defer releaseBarrier()
	work, err := s.claims.Get(ctx, principal.TenantID, principal.ProjectID, fence.BeadID)
	work = normalizeWorkItem(work)
	if err != nil || !workMatchesFence(work, principal, fence) {
		return authorityv1.LeaseMutationResponse{}, s.deny(ctx, principal, leaseOperation(action, "intent"), fence.BeadID, traceRef, fence.Labels,
			newDenial(authorityv1.ErrorPolicyDenied, ruleLeaseLifecycleClaim, work.LifecycleState, requiredLeaseLifecycleFence, allowedLeaseLifecycleRetry, traceRef))
	}
	labels, labelDenial := admittedLabels(principal.Labels, work.Labels, nil, work.LifecycleState, traceRef)
	if labelDenial != nil || !equalLabels(labels, fence.Labels) {
		return authorityv1.LeaseMutationResponse{}, s.deny(ctx, principal, leaseOperation(action, "policy"), fence.BeadID, traceRef, labels,
			newDenial(authorityv1.ErrorPolicyDenied, ruleLeaseLifecycleFence, work.LifecycleState, requiredLeaseLifecycleFence, allowedLeaseLifecycleRetry, traceRef))
	}
	stored, err := s.leaseOps.GetLease(ctx, principal.TenantID, principal.ProjectID, fence.LeaseID)
	if err != nil || !leaseTupleMatches(stored, fence) {
		return authorityv1.LeaseMutationResponse{}, s.deny(ctx, principal, leaseOperation(action, "policy"), fence.BeadID, traceRef, labels,
			newDenial(authorityv1.ErrorPolicyDenied, ruleLeaseLifecycleFence, work.LifecycleState, requiredLeaseLifecycleFence, allowedLeaseLifecycleRetry, traceRef))
	}
	replayed := action == "renew" && stored.Active && stored.State == authorityv1.LeaseActive && stored.ExpiresAt.Equal(newExpiry) ||
		action == "release" && !stored.Active && stored.State == authorityv1.LeaseReleased
	if !replayed && (!stored.Active || stored.State != authorityv1.LeaseActive || !stored.ExpiresAt.After(s.now().UTC())) {
		return authorityv1.LeaseMutationResponse{}, s.deny(ctx, principal, leaseOperation(action, "policy"), fence.BeadID, traceRef, labels,
			newDenial(authorityv1.ErrorPolicyDenied, ruleLeaseLifecycleFence, work.LifecycleState, requiredLeaseLifecycleFence, allowedLeaseLifecycleRetry, traceRef))
	}
	if action == "renew" && !replayed && (!newExpiry.After(stored.ExpiresAt) || newExpiry.After(s.now().UTC().Add(maximumInitialLeaseDuration))) {
		return authorityv1.LeaseMutationResponse{}, s.deny(ctx, principal, operationLeaseRenewPolicy, fence.BeadID, traceRef, labels,
			newDenial(authorityv1.ErrorPolicyDenied, ruleLeaseLifecycleInvalid, work.LifecycleState, requiredLeaseLifecycleFence, allowedLeaseLifecycleRetry, traceRef))
	}
	token := leaseMutationToken(action, newExpiry, "")
	result := stored
	if !replayed {
		if _, err := s.appendLeaseEvent(ctx, principal, fence, traceRef, leaseOperation(action, "intent"), outcomeIntent, ruleLeaseLifecycleAllowed, token, "intent", stored, authorityv1.CapabilityLease{}); err != nil {
			return authorityv1.LeaseMutationResponse{}, err
		}
		if _, err := s.appendLeaseEvent(ctx, principal, fence, traceRef, leaseOperation(action, "policy"), outcomePolicyAllowed, ruleLeaseLifecycleAllowed, token, "policy", stored, authorityv1.CapabilityLease{}); err != nil {
			return authorityv1.LeaseMutationResponse{}, err
		}
		if action == "renew" {
			result, err = s.leaseOps.Renew(ctx, fence, newExpiry)
		} else {
			result, err = s.leaseOps.Release(ctx, fence)
		}
		if err != nil {
			return authorityv1.LeaseMutationResponse{}, s.deny(ctx, principal, leaseOperation(action, "receipt"), fence.BeadID, traceRef, labels,
				newDenial(authorityv1.ErrorUnknownEffect, ruleLeaseLifecycleUnknown, work.LifecycleState, requiredLeaseLifecycleFence, allowedLeaseLifecycleRetry, traceRef))
		}
	}
	if !validLeaseMutationResult(action, stored, result, newExpiry) {
		return authorityv1.LeaseMutationResponse{}, s.deny(ctx, principal, leaseOperation(action, "receipt"), fence.BeadID, traceRef, labels,
			newDenial(authorityv1.ErrorUnknownEffect, ruleLeaseLifecycleUnknown, work.LifecycleState, requiredLeaseLifecycleFence, allowedLeaseLifecycleRetry, traceRef))
	}
	fresh, err := s.claims.Get(ctx, principal.TenantID, principal.ProjectID, fence.BeadID)
	fresh = normalizeWorkItem(fresh)
	if err != nil || !equalWorkItems(work, fresh) || !workMatchesFence(fresh, principal, fence) {
		return authorityv1.LeaseMutationResponse{}, s.deny(ctx, principal, leaseOperation(action, "receipt"), fence.BeadID, traceRef, labels,
			newDenial(authorityv1.ErrorUnknownEffect, ruleLeaseLifecycleClaim, work.LifecycleState, requiredLeaseLifecycleFence, allowedLeaseLifecycleRetry, traceRef))
	}
	receiptEvent, err := s.appendLeaseEvent(ctx, principal, fence, traceRef, leaseOperation(action, "receipt"), outcomeVerified, ruleLeaseLifecycleAllowed, token, "receipt", authorityv1.CapabilityLease{}, result)
	if err != nil {
		return authorityv1.LeaseMutationResponse{}, err
	}
	return authorityv1.LeaseMutationResponse{Lease: result, Replayed: replayed, ReceiptRef: leaseReceiptRef(action, result.LeaseID, receiptEvent.EventHash)}, nil
}

// RevokeLease is a safety termination route. It deliberately does not require
// a canonical work read, so Security can revoke a compromised lease even when
// the canonical work store is unavailable.
func (s *Service) RevokeLease(ctx context.Context, principal authorityv1.Principal, request authorityv1.RevokeLeaseRequest) (authorityv1.LeaseMutationResponse, error) {
	traceRef, denial := admitRequest(principal, request.TraceRef, nil)
	if denial != nil {
		return authorityv1.LeaseMutationResponse{}, s.deny(ctx, principal, operationLeaseRevokeIntent, "", traceRef, nil, denial)
	}
	if !hasCapability(principal, authorityv1.CapabilityLeaseRevoke) {
		return authorityv1.LeaseMutationResponse{}, s.deny(ctx, principal, operationLeaseRevokeIntent, "", traceRef, nil,
			newDenial(authorityv1.ErrorUnauthorized, ruleCapabilityMissing, "", "obtain policy-approved lease.revoke capability", "policy.request(lease.revoke)", traceRef))
	}
	if s.leaseOps == nil || request.TenantID != principal.TenantID || request.ProjectID != principal.ProjectID ||
		!validID(request.LeaseID) || !validID(request.FenceGeneration) || request.LeaseEpoch == 0 || !validID(request.Reason) {
		return authorityv1.LeaseMutationResponse{}, s.deny(ctx, principal, operationLeaseRevokeIntent, "", traceRef, nil,
			newDenial(authorityv1.ErrorInvalidRequest, ruleLeaseLifecycleInvalid, "", requiredLeaseLifecycleFence, allowedLeaseLifecycleRetry, traceRef))
	}
	releaseBarrier := func() {}
	var err error
	if s.barrier != nil {
		releaseBarrier, err = s.barrier.Enter(ctx, principal.TenantID, principal.ProjectID)
		if err != nil {
			return authorityv1.LeaseMutationResponse{}, s.deny(ctx, principal, operationLeaseRevokeIntent, "", traceRef, nil,
				newDenial(authorityv1.ErrorAuthorityDown, ruleAuthorityUnavailable, "", requiredLeaseLifecycleFence, allowedLeaseLifecycleRetry, traceRef))
		}
	}
	defer releaseBarrier()
	stored, err := s.leaseOps.GetLease(ctx, principal.TenantID, principal.ProjectID, request.LeaseID)
	if err != nil || stored.FenceGeneration != request.FenceGeneration || stored.LeaseEpoch != request.LeaseEpoch {
		return authorityv1.LeaseMutationResponse{}, s.deny(ctx, principal, operationLeaseRevokePolicy, stored.BeadID, traceRef, stored.Labels,
			newDenial(authorityv1.ErrorPolicyDenied, ruleLeaseLifecycleFence, "", requiredLeaseLifecycleFence, allowedLeaseLifecycleRetry, traceRef))
	}
	replayed := !stored.Active && stored.State == authorityv1.LeaseRevoked
	if !replayed && (!stored.Active || stored.State != authorityv1.LeaseActive) {
		return authorityv1.LeaseMutationResponse{}, s.deny(ctx, principal, operationLeaseRevokePolicy, stored.BeadID, traceRef, stored.Labels,
			newDenial(authorityv1.ErrorPolicyDenied, ruleLeaseLifecycleFence, "", requiredLeaseLifecycleFence, allowedLeaseLifecycleRetry, traceRef))
	}
	fence := fenceFromLease(stored)
	token := leaseMutationToken("revoke", time.Time{}, request.Reason)
	result := stored
	if !replayed {
		if _, err := s.appendLeaseEvent(ctx, principal, fence, traceRef, operationLeaseRevokeIntent, outcomeIntent, ruleLeaseLifecycleAllowed, token, "intent", stored, authorityv1.CapabilityLease{}); err != nil {
			return authorityv1.LeaseMutationResponse{}, err
		}
		if _, err := s.appendLeaseEvent(ctx, principal, fence, traceRef, operationLeaseRevokePolicy, outcomePolicyAllowed, ruleLeaseLifecycleAllowed, token, "policy", stored, authorityv1.CapabilityLease{}); err != nil {
			return authorityv1.LeaseMutationResponse{}, err
		}
		result, err = s.leaseOps.Revoke(ctx, request)
		if err != nil {
			return authorityv1.LeaseMutationResponse{}, s.deny(ctx, principal, operationLeaseRevokeReceipt, stored.BeadID, traceRef, stored.Labels,
				newDenial(authorityv1.ErrorUnknownEffect, ruleLeaseLifecycleUnknown, "", requiredLeaseLifecycleFence, allowedLeaseLifecycleRetry, traceRef))
		}
	}
	if result.Active || result.State != authorityv1.LeaseRevoked || !sameLeaseIdentity(stored, result) {
		return authorityv1.LeaseMutationResponse{}, s.deny(ctx, principal, operationLeaseRevokeReceipt, stored.BeadID, traceRef, stored.Labels,
			newDenial(authorityv1.ErrorUnknownEffect, ruleLeaseLifecycleUnknown, "", requiredLeaseLifecycleFence, allowedLeaseLifecycleRetry, traceRef))
	}
	receiptEvent, err := s.appendLeaseEvent(ctx, principal, fence, traceRef, operationLeaseRevokeReceipt, outcomeVerified, ruleLeaseLifecycleAllowed, token, "receipt", authorityv1.CapabilityLease{}, result)
	if err != nil {
		return authorityv1.LeaseMutationResponse{}, err
	}
	return authorityv1.LeaseMutationResponse{Lease: result, Replayed: replayed, ReceiptRef: leaseReceiptRef("revoke", result.LeaseID, receiptEvent.EventHash)}, nil
}

func validLifecycleFence(principal authorityv1.Principal, fence authorityv1.FencingTuple) bool {
	if fence.TenantID != principal.TenantID || fence.ProjectID != principal.ProjectID || !validID(fence.BeadID) || !validID(fence.AttemptID) ||
		!validID(fence.CanonicalClaimAttemptID) || !validID(fence.IdempotencyKey) || !validID(fence.LeaseID) ||
		!validID(fence.FenceGeneration) || fence.LeaseEpoch == 0 || !validID(fence.ClaimVersion.AuthorityGeneration) ||
		!validID(fence.ClaimVersion.IssueIncarnation) || fence.ClaimVersion.IssueMutationSequence == 0 ||
		fence.ClaimVersion.DependencyGraphRevision == 0 || !commitSHA.MatchString(fence.BaseSHA) ||
		fence.Capability != authorityv1.CapabilityTicketDelivery || len(fence.ExclusivePaths) == 0 ||
		len(fence.ExclusivePaths) > maxProjectionValues || hasDuplicateStrings(fence.ExclusivePaths) || len(fence.Labels) == 0 ||
		len(fence.Labels) > maxProjectionValues || hasDuplicateLabels(fence.Labels) {
		return false
	}
	for _, exclusivePath := range fence.ExclusivePaths {
		if !safeExclusivePath(exclusivePath) {
			return false
		}
	}
	return true
}

func leaseTupleMatches(lease authorityv1.CapabilityLease, fence authorityv1.FencingTuple) bool {
	return lease.TenantID == fence.TenantID && lease.ProjectID == fence.ProjectID && lease.BeadID == fence.BeadID &&
		lease.AttemptID == fence.AttemptID && lease.CanonicalClaimAttemptID == fence.CanonicalClaimAttemptID &&
		lease.IdempotencyKey == fence.IdempotencyKey && lease.LeaseID == fence.LeaseID && lease.FenceGeneration == fence.FenceGeneration &&
		lease.LeaseEpoch == fence.LeaseEpoch && lease.ClaimVersion == fence.ClaimVersion && lease.BaseSHA == fence.BaseSHA &&
		lease.Capability == fence.Capability && equalStrings(lease.ExclusivePaths, fence.ExclusivePaths) && equalLabels(lease.Labels, fence.Labels)
}

func sameLeaseIdentity(left, right authorityv1.CapabilityLease) bool {
	return leaseTupleMatches(right, fenceFromLease(left)) && left.IssuedAt.Equal(right.IssuedAt) && left.ExpiresAt.Equal(right.ExpiresAt)
}

func validLeaseMutationResult(action string, before, after authorityv1.CapabilityLease, expiry time.Time) bool {
	if !sameLeaseIdentity(before, after) {
		if action != "renew" || !leaseTupleMatches(after, fenceFromLease(before)) || !before.IssuedAt.Equal(after.IssuedAt) {
			return false
		}
	}
	if action == "renew" {
		return after.Active && after.State == authorityv1.LeaseActive && after.ExpiresAt.Equal(expiry)
	}
	return !after.Active && after.State == authorityv1.LeaseReleased
}

func fenceFromLease(lease authorityv1.CapabilityLease) authorityv1.FencingTuple {
	return authorityv1.FencingTuple{
		TenantID: lease.TenantID, ProjectID: lease.ProjectID, BeadID: lease.BeadID, AttemptID: lease.AttemptID,
		CanonicalClaimAttemptID: lease.CanonicalClaimAttemptID, IdempotencyKey: lease.IdempotencyKey, LeaseID: lease.LeaseID,
		FenceGeneration: lease.FenceGeneration, LeaseEpoch: lease.LeaseEpoch, ClaimVersion: lease.ClaimVersion,
		BaseSHA: lease.BaseSHA, Capability: lease.Capability, ExclusivePaths: append([]string(nil), lease.ExclusivePaths...),
		Labels: append([]authorityv1.Label(nil), lease.Labels...),
	}
}

func leaseOperation(action, phase string) string {
	switch action + ":" + phase {
	case "renew:intent":
		return operationLeaseRenewIntent
	case "renew:policy":
		return operationLeaseRenewPolicy
	case "renew:receipt":
		return operationLeaseRenewReceipt
	case "release:intent":
		return operationLeaseReleaseIntent
	case "release:policy":
		return operationLeaseReleasePolicy
	default:
		return operationLeaseReleaseReceipt
	}
}

func leaseMutationToken(action string, expiry time.Time, reason string) string {
	return action + "\x00" + expiry.UTC().Format(time.RFC3339Nano) + "\x00" + reason
}

func (s *Service) appendLeaseEvent(ctx context.Context, principal authorityv1.Principal, fence authorityv1.FencingTuple, traceRef, operation, outcome, rule, token, phase string, before, after authorityv1.CapabilityLease) (authorityv1.Event, error) {
	event := s.event(principal, operation, fence.BeadID, traceRef, outcome, rule, fence.Labels)
	event.EventID = deterministicEventID(fence.LeaseID+"\x00"+token, phase)
	event.AttemptID, event.IdempotencyKey = fence.AttemptID, fence.IdempotencyKey
	event.CanonicalVersion, event.LeaseEpoch = &fence.ClaimVersion, fence.LeaseEpoch
	event.BeforeHash, event.AfterHash = leaseDigest(before), leaseDigest(after)
	receipt, err := s.events.Append(ctx, event)
	if err != nil || !validEventReceipt(event, receipt) {
		return authorityv1.Event{}, newDenial(authorityv1.ErrorAuthorityDown, ruleAuthorityUnavailable, "", requiredLeaseLifecycleFence, allowedLeaseLifecycleRetry, traceRef)
	}
	return receipt, nil
}

func leaseDigest(lease authorityv1.CapabilityLease) string {
	if lease.LeaseID == "" {
		return ""
	}
	data, err := json.Marshal(lease)
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func leaseReceiptRef(action, leaseID, eventHash string) string {
	digest := sha256.Sum256([]byte(action + "\x00" + leaseID + "\x00" + eventHash))
	return "lease-receipt-" + hex.EncodeToString(digest[:])
}
