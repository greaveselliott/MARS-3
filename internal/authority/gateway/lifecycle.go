/*
FactoryDocSync:
docs:
- docs/features/F-002-work-authority.md
- docs/product-specs/work-authority.md
- docs/design-docs/ADR-001-git-beads-authority.md
- docs/code-documentation-map.md
*/

package gateway

import (
	"context"
	"errors"
	"fmt"

	authorityv1 "github.com/greaveselliott/MARS-3/api/authority/v1"
)

type LifecycleOperation string

const (
	LifecycleHandoff   LifecycleOperation = "handoff"
	LifecycleReview    LifecycleOperation = "review-verdict"
	LifecycleRun       LifecycleOperation = "run-disposition"
	LifecycleReconcile LifecycleOperation = "reconciliation"
	LifecycleTerminal  LifecycleOperation = "terminal-transition"
)

// LifecycleMutation is the complete normalized canonical transition. Fields
// that do not apply to Operation must remain empty and are rejected otherwise.
type LifecycleMutation struct {
	TenantID                string
	ProjectID               string
	BeadID                  string
	ExpectedVersion         authorityv1.WorkVersion
	ExpectedIntegrity       authorityv1.IntegrityDigests
	Operation               LifecycleOperation
	PrincipalProfileID      string
	AttemptID               string
	CanonicalClaimAttemptID string
	HandoffFenceDigest      string
	HeadSHA                 string
	EvidenceRefs            []string
	NextProfileID           string
	Verdict                 authorityv1.ReviewVerdict
	RunStatus               authorityv1.RunDispositionStatus
	Failure                 *authorityv1.FailureContext
	MergedSHA               string
	MergedTree              string
	PullRequestID           string
	ProtectedMainRunID      string
	IdempotencyKey          string
}

type LifecycleStore interface {
	WorkStore
	CompareAndSwapLifecycle(context.Context, LifecycleMutation) (authorityv1.WorkItem, error)
}

const (
	ruleLifecycleAllowed       = "authority.lifecycle.allowed"
	ruleLifecycleInvalid       = "authority.lifecycle.invalid"
	ruleLifecycleOrder         = "authority.lifecycle.review_order"
	ruleLifecyclePrerequisite  = "authority.lifecycle.prerequisite"
	ruleLifecycleStale         = "authority.lifecycle.stale_version"
	ruleLifecycleLease         = "authority.lifecycle.lease_required"
	ruleLifecycleIdempotency   = "authority.lifecycle.idempotency_conflict"
	ruleLifecycleUnknown       = "authority.lifecycle.unknown_effect"
	requiredLifecycleRead      = "read the current canonical lifecycle and WorkVersion"
	requiredLifecycleReconcile = "reconcile the exact lifecycle operation before retry"
)

func (s *Service) Handoff(ctx context.Context, principal authorityv1.Principal, request authorityv1.HandoffRequest) (authorityv1.LifecycleMutationResponse, error) {
	request.EvidenceRefs = sortedStrings(request.EvidenceRefs)
	request.Fence.ExclusivePaths = sortedStrings(request.Fence.ExclusivePaths)
	request.Fence.Labels = sortedLabels(request.Fence.Labels)
	fenceDigest := deterministicJSONDigest(request.Fence)
	mutation := LifecycleMutation{
		TenantID: principal.TenantID, ProjectID: principal.ProjectID, BeadID: request.BeadID,
		ExpectedVersion: request.ExpectedVersion, ExpectedIntegrity: request.ExpectedIntegrity, Operation: LifecycleHandoff,
		PrincipalProfileID: principal.ProfileID, AttemptID: request.Fence.AttemptID, CanonicalClaimAttemptID: request.Fence.CanonicalClaimAttemptID,
		HandoffFenceDigest: fenceDigest, HeadSHA: request.HeadSHA,
		EvidenceRefs: request.EvidenceRefs, NextProfileID: request.NextProfileID, IdempotencyKey: request.IdempotencyKey,
	}
	return s.mutateLifecycle(ctx, principal, request.TraceRef, mutation, &request.Fence, authorityv1.CapabilityWorkHandoff)
}

func (s *Service) RecordReviewVerdict(ctx context.Context, principal authorityv1.Principal, request authorityv1.ReviewVerdictRequest) (authorityv1.LifecycleMutationResponse, error) {
	request.EvidenceRefs = sortedStrings(request.EvidenceRefs)
	failure := normalizedFailureContext(request.Failure)
	mutation := LifecycleMutation{
		TenantID: principal.TenantID, ProjectID: principal.ProjectID, BeadID: request.BeadID,
		ExpectedVersion: request.ExpectedVersion, ExpectedIntegrity: request.ExpectedIntegrity, Operation: LifecycleReview,
		PrincipalProfileID: principal.ProfileID, HeadSHA: request.HeadSHA, Verdict: request.Verdict,
		EvidenceRefs: request.EvidenceRefs, IdempotencyKey: request.IdempotencyKey, Failure: failure,
	}
	return s.mutateLifecycle(ctx, principal, request.TraceRef, mutation, nil, authorityv1.CapabilityReviewRecord)
}

func (s *Service) RecordRunDisposition(ctx context.Context, principal authorityv1.Principal, request authorityv1.RunDispositionRequest) (authorityv1.LifecycleMutationResponse, error) {
	request.EvidenceRefs = sortedStrings(request.EvidenceRefs)
	failure := normalizedFailureContext(request.Failure)
	mutation := LifecycleMutation{
		TenantID: principal.TenantID, ProjectID: principal.ProjectID, BeadID: request.BeadID,
		ExpectedVersion: request.ExpectedVersion, ExpectedIntegrity: request.ExpectedIntegrity, Operation: LifecycleRun,
		PrincipalProfileID: principal.ProfileID, HeadSHA: request.HeadSHA, RunStatus: request.Status,
		EvidenceRefs: request.EvidenceRefs, IdempotencyKey: request.IdempotencyKey, Failure: failure,
	}
	return s.mutateLifecycle(ctx, principal, request.TraceRef, mutation, nil, authorityv1.CapabilityRunDisposition)
}

func (s *Service) RecordReconciliation(ctx context.Context, principal authorityv1.Principal, request authorityv1.ReconciliationRequest) (authorityv1.LifecycleMutationResponse, error) {
	request.EvidenceRefs = sortedStrings(request.EvidenceRefs)
	mutation := LifecycleMutation{
		TenantID: principal.TenantID, ProjectID: principal.ProjectID, BeadID: request.BeadID,
		ExpectedVersion: request.ExpectedVersion, ExpectedIntegrity: request.ExpectedIntegrity, Operation: LifecycleReconcile,
		PrincipalProfileID: principal.ProfileID, HeadSHA: request.HeadSHA, MergedSHA: request.MergedSHA, MergedTree: request.MergedTree,
		PullRequestID: request.PullRequestID, ProtectedMainRunID: request.ProtectedMainRunID,
		EvidenceRefs: request.EvidenceRefs, IdempotencyKey: request.IdempotencyKey,
	}
	return s.mutateLifecycle(ctx, principal, request.TraceRef, mutation, nil, authorityv1.CapabilityWorkReconcile)
}

func (s *Service) CloseWork(ctx context.Context, principal authorityv1.Principal, request authorityv1.TerminalTransitionRequest) (authorityv1.LifecycleMutationResponse, error) {
	request.EvidenceRefs = sortedStrings(request.EvidenceRefs)
	mutation := LifecycleMutation{
		TenantID: principal.TenantID, ProjectID: principal.ProjectID, BeadID: request.BeadID,
		ExpectedVersion: request.ExpectedVersion, ExpectedIntegrity: request.ExpectedIntegrity, Operation: LifecycleTerminal,
		PrincipalProfileID: principal.ProfileID, HeadSHA: request.HeadSHA,
		EvidenceRefs: request.EvidenceRefs, IdempotencyKey: request.IdempotencyKey,
	}
	return s.mutateLifecycle(ctx, principal, request.TraceRef, mutation, nil, authorityv1.CapabilityWorkClose)
}

func (s *Service) mutateLifecycle(ctx context.Context, principal authorityv1.Principal, traceRef string, mutation LifecycleMutation, fence *authorityv1.FencingTuple, capability authorityv1.Capability) (authorityv1.LifecycleMutationResponse, error) {
	traceRef, denial := admitRequest(principal, traceRef, nil)
	operation := "work." + string(mutation.Operation)
	if denial != nil {
		return authorityv1.LifecycleMutationResponse{}, s.deny(ctx, principal, operation+".intent", mutation.BeadID, traceRef, nil, denial)
	}
	if !hasCapability(principal, capability) || mutation.Operation == LifecycleHandoff && !hasCapability(principal, authorityv1.CapabilityLeaseRelease) {
		return authorityv1.LifecycleMutationResponse{}, s.deny(ctx, principal, operation+".intent", mutation.BeadID, traceRef, nil,
			newDenial(authorityv1.ErrorUnauthorized, ruleCapabilityMissing, "", "obtain policy-approved "+string(capability)+" capability", "policy.request("+string(capability)+")", traceRef))
	}
	if s.lifecycle == nil || s.barrier == nil || s.workLocks == nil || s.leaseOps == nil {
		return authorityv1.LifecycleMutationResponse{}, s.deny(ctx, principal, operation+".intent", mutation.BeadID, traceRef, nil,
			newDenial(authorityv1.ErrorAuthorityDown, ruleAuthorityUnavailable, "", requiredLifecycleRead, "work.read", traceRef))
	}
	if !validLifecycleMutation(mutation) {
		return authorityv1.LifecycleMutationResponse{}, s.deny(ctx, principal, operation+".intent", mutation.BeadID, traceRef, nil,
			newDenial(authorityv1.ErrorInvalidRequest, ruleLifecycleInvalid, "", requiredLifecycleRead, operation, traceRef))
	}
	releaseBarrier, err := s.barrier.Enter(ctx, principal.TenantID, principal.ProjectID)
	if err != nil {
		return authorityv1.LifecycleMutationResponse{}, s.deny(ctx, principal, operation+".intent", mutation.BeadID, traceRef, nil,
			newDenial(authorityv1.ErrorAuthorityDown, ruleAuthorityUnavailable, "", requiredLifecycleReconcile, "work.reconcile", traceRef))
	}
	defer releaseBarrier()
	releaseWork, err := s.workLocks.EnterWork(ctx, principal.TenantID, principal.ProjectID, mutation.BeadID)
	if err != nil {
		return authorityv1.LifecycleMutationResponse{}, s.deny(ctx, principal, operation+".intent", mutation.BeadID, traceRef, nil,
			newDenial(authorityv1.ErrorAuthorityDown, ruleAuthorityUnavailable, "", requiredLifecycleReconcile, "work.reconcile", traceRef))
	}
	defer releaseWork()

	work, err := s.lifecycle.Get(ctx, principal.TenantID, principal.ProjectID, mutation.BeadID)
	work = normalizeWorkItem(work)
	if err != nil || work.TenantID != principal.TenantID || work.ProjectID != principal.ProjectID {
		return authorityv1.LifecycleMutationResponse{}, s.deny(ctx, principal, operation+".intent", mutation.BeadID, traceRef, nil,
			newDenial(authorityv1.ErrorAuthorityDown, ruleAuthorityUnavailable, work.LifecycleState, requiredLifecycleRead, "work.read", traceRef))
	}
	labels, labelDenial := admittedLabels(principal.Labels, work.Labels, nil, work.LifecycleState, traceRef)
	if labelDenial != nil {
		return authorityv1.LifecycleMutationResponse{}, s.deny(ctx, principal, operation+".policy", mutation.BeadID, traceRef, labels, labelDenial)
	}
	if lifecycleReplayMatches(work, mutation) {
		receiptRef, receiptErr := s.appendLifecycleReconciliationReceipt(ctx, principal, traceRef, mutation, labels, work)
		if receiptErr != nil {
			return authorityv1.LifecycleMutationResponse{}, receiptErr
		}
		return authorityv1.LifecycleMutationResponse{Work: work, Replayed: true, ReceiptRef: receiptRef}, nil
	}
	if lifecycleIdempotencyUsed(work, mutation.IdempotencyKey) {
		return authorityv1.LifecycleMutationResponse{}, s.deny(ctx, principal, operation+".policy", mutation.BeadID, traceRef, labels,
			newDenial(authorityv1.ErrorPolicyDenied, ruleLifecycleIdempotency, work.LifecycleState, "use the original normalized lifecycle request or a new idempotency key", "work.read", traceRef))
	}
	if work.Version != mutation.ExpectedVersion || work.Integrity != mutation.ExpectedIntegrity {
		return authorityv1.LifecycleMutationResponse{}, s.deny(ctx, principal, operation+".policy", mutation.BeadID, traceRef, labels,
			newDenial(authorityv1.ErrorStaleVersion, ruleLifecycleStale, work.LifecycleState, requiredLifecycleRead, "work.read", traceRef))
	}
	if rule := lifecycleAdmissionRule(work, mutation); rule != "" {
		return authorityv1.LifecycleMutationResponse{}, s.deny(ctx, principal, operation+".policy", mutation.BeadID, traceRef, labels,
			newDenial(authorityv1.ErrorPolicyDenied, rule, work.LifecycleState, requiredLifecycleRead, lifecycleCorrectiveAction(work, mutation), traceRef))
	}
	if mutation.Operation != LifecycleHandoff {
		if _, active, leaseErr := s.leaseOps.ActiveLeaseForBead(ctx, principal.TenantID, principal.ProjectID, mutation.BeadID); leaseErr != nil {
			return authorityv1.LifecycleMutationResponse{}, s.deny(ctx, principal, operation+".policy", mutation.BeadID, traceRef, labels,
				newDenial(authorityv1.ErrorAuthorityDown, ruleAuthorityUnavailable, work.LifecycleState, requiredLifecycleReconcile, "lease.read", traceRef))
		} else if active {
			return authorityv1.LifecycleMutationResponse{}, s.deny(ctx, principal, operation+".policy", mutation.BeadID, traceRef, labels,
				newDenial(authorityv1.ErrorPolicyDenied, ruleLifecycleLease, work.LifecycleState, "release or revoke the implementation lease before independent review", "lease.release", traceRef))
		}
	}
	if err := s.appendLifecycleEvent(ctx, principal, traceRef, mutation, labels, "intent", outcomeIntent, work, authorityv1.WorkItem{}); err != nil {
		return authorityv1.LifecycleMutationResponse{}, err
	}
	if err := s.appendLifecycleEvent(ctx, principal, traceRef, mutation, labels, "policy", outcomePolicyAllowed, work, authorityv1.WorkItem{}); err != nil {
		return authorityv1.LifecycleMutationResponse{}, err
	}
	if mutation.Operation == LifecycleHandoff {
		if err := s.releaseHandoffLease(ctx, principal, work, *fence, traceRef); err != nil {
			return authorityv1.LifecycleMutationResponse{}, s.deny(ctx, principal, operation+".receipt", mutation.BeadID, traceRef, labels, err)
		}
	}
	post, err := s.lifecycle.CompareAndSwapLifecycle(ctx, mutation)
	post = normalizeWorkItem(post)
	if err != nil {
		fresh, readErr := s.lifecycle.Get(ctx, principal.TenantID, principal.ProjectID, mutation.BeadID)
		fresh = normalizeWorkItem(fresh)
		if readErr == nil && lifecycleReplayMatches(fresh, mutation) {
			receiptRef, receiptErr := s.appendLifecycleReconciliationReceipt(ctx, principal, traceRef, mutation, labels, fresh)
			if receiptErr != nil {
				return authorityv1.LifecycleMutationResponse{}, receiptErr
			}
			return authorityv1.LifecycleMutationResponse{Work: fresh, Replayed: true, ReceiptRef: receiptRef}, nil
		} else if errors.Is(err, ErrStaleWorkVersion) {
			return authorityv1.LifecycleMutationResponse{}, s.deny(ctx, principal, operation+".receipt", mutation.BeadID, traceRef, labels,
				newDenial(authorityv1.ErrorStaleVersion, ruleLifecycleStale, work.LifecycleState, requiredLifecycleRead, "work.read", traceRef))
		} else {
			_ = s.appendLifecycleUnknown(ctx, principal, traceRef, mutation, labels)
			return authorityv1.LifecycleMutationResponse{}, newDenial(authorityv1.ErrorUnknownEffect, ruleLifecycleUnknown, work.LifecycleState, requiredLifecycleReconcile, "work.reconcile", traceRef)
		}
	}
	if !lifecycleReplayMatches(post, mutation) || post.Version.IssueMutationSequence != work.Version.IssueMutationSequence+1 {
		_ = s.appendLifecycleUnknown(ctx, principal, traceRef, mutation, labels)
		return authorityv1.LifecycleMutationResponse{}, newDenial(authorityv1.ErrorUnknownEffect, ruleLifecycleUnknown, work.LifecycleState, requiredLifecycleReconcile, "work.reconcile", traceRef)
	}
	if err := s.appendLifecycleEvent(ctx, principal, traceRef, mutation, labels, "receipt", outcomeVerified, work, post); err != nil {
		return authorityv1.LifecycleMutationResponse{}, newDenial(authorityv1.ErrorUnknownEffect, ruleLifecycleUnknown, post.LifecycleState, requiredLifecycleReconcile, "work.reconcile", traceRef)
	}
	return authorityv1.LifecycleMutationResponse{Work: post, ReceiptRef: lifecycleReceiptRef(mutation)}, nil
}

func validLifecycleMutation(mutation LifecycleMutation) bool {
	if !validID(mutation.TenantID) || !validID(mutation.ProjectID) || !validID(mutation.BeadID) || !validID(mutation.PrincipalProfileID) ||
		!commitSHA.MatchString(mutation.HeadSHA) || !validID(mutation.IdempotencyKey) || len(mutation.EvidenceRefs) == 0 ||
		len(mutation.EvidenceRefs) > 16 || hasDuplicateStrings(mutation.EvidenceRefs) || !validIntegrity(mutation.ExpectedIntegrity) ||
		!validID(mutation.ExpectedVersion.AuthorityGeneration) || !validID(mutation.ExpectedVersion.IssueIncarnation) ||
		mutation.ExpectedVersion.IssueMutationSequence == 0 || mutation.ExpectedVersion.DependencyGraphRevision == 0 {
		return false
	}
	for _, reference := range mutation.EvidenceRefs {
		if !validID(reference) {
			return false
		}
	}
	switch mutation.Operation {
	case LifecycleHandoff:
		return validID(mutation.AttemptID) && validID(mutation.CanonicalClaimAttemptID) && hexDigest.MatchString(mutation.HandoffFenceDigest) &&
			validID(mutation.NextProfileID) && mutation.Verdict == "" && mutation.RunStatus == "" && mutation.Failure == nil && noMergeFields(mutation)
	case LifecycleReview:
		return noAttemptFields(mutation) && mutation.NextProfileID == "" && knownReviewVerdict(mutation.Verdict) && mutation.RunStatus == "" &&
			validFailureForReview(mutation.Verdict, mutation.Failure) && noMergeFields(mutation)
	case LifecycleRun:
		return noAttemptFields(mutation) && mutation.NextProfileID == "" && mutation.Verdict == "" && knownRunStatus(mutation.RunStatus) &&
			validFailureForRun(mutation.RunStatus, mutation.Failure) && noMergeFields(mutation)
	case LifecycleReconcile:
		return noAttemptFields(mutation) && mutation.NextProfileID == "" && mutation.Verdict == "" && mutation.RunStatus == "" &&
			mutation.Failure == nil && commitSHA.MatchString(mutation.MergedSHA) && commitSHA.MatchString(mutation.MergedTree) && validID(mutation.PullRequestID) && validID(mutation.ProtectedMainRunID)
	case LifecycleTerminal:
		return noAttemptFields(mutation) && mutation.NextProfileID == "" && mutation.Verdict == "" && mutation.RunStatus == "" && mutation.Failure == nil && noMergeFields(mutation)
	default:
		return false
	}
}

func lifecycleAdmissionRule(work authorityv1.WorkItem, mutation LifecycleMutation) string {
	if projectionRule(work) != "" || len(work.VerificationOrder) < 3 {
		return ruleProjectionInvalid
	}
	orchestrator := work.VerificationOrder[len(work.VerificationOrder)-1]
	switch mutation.Operation {
	case LifecycleHandoff:
		if work.LifecycleState != authorityv1.LifecycleInProgress || work.Assignee != mutation.PrincipalProfileID ||
			work.ClaimAttemptID != mutation.CanonicalClaimAttemptID || work.VerificationOrder[0] != mutation.NextProfileID {
			return ruleLifecyclePrerequisite
		}
	case LifecycleReview:
		if work.LifecycleState != authorityv1.LifecycleInReview || work.Handoff == nil || work.Handoff.HeadSHA != mutation.HeadSHA ||
			len(work.Reviews) >= len(work.VerificationOrder)-1 || work.VerificationOrder[len(work.Reviews)] != mutation.PrincipalProfileID {
			return ruleLifecycleOrder
		}
	case LifecycleRun:
		if mutation.PrincipalProfileID != orchestrator || !runDispositionAllowed(work, mutation) {
			return ruleLifecyclePrerequisite
		}
	case LifecycleReconcile:
		if mutation.PrincipalProfileID != orchestrator || !acceptedReviews(work, mutation.HeadSHA) || work.RunDisposition == nil ||
			work.RunDisposition.Status != authorityv1.RunCompleted || work.Reconciliation != nil {
			return ruleLifecyclePrerequisite
		}
	case LifecycleTerminal:
		if mutation.PrincipalProfileID != orchestrator || !acceptedReviews(work, mutation.HeadSHA) || work.RunDisposition == nil ||
			work.RunDisposition.Status != authorityv1.RunCompleted || work.Reconciliation == nil || work.Terminal != nil {
			return ruleLifecyclePrerequisite
		}
	}
	return ""
}

func (s *Service) releaseHandoffLease(ctx context.Context, principal authorityv1.Principal, work authorityv1.WorkItem, fence authorityv1.FencingTuple, traceRef string) *authorityv1.Denial {
	if !validLifecycleFence(principal, fence) || !workMatchesFence(work, principal, fence) || work.Version != fence.ClaimVersion {
		return newDenial(authorityv1.ErrorPolicyDenied, ruleLifecycleLease, work.LifecycleState, "present the current owning implementation lease", "lease.read", traceRef)
	}
	stored, err := s.leaseOps.GetLease(ctx, principal.TenantID, principal.ProjectID, fence.LeaseID)
	if err != nil || !leaseTupleMatches(stored, fence) {
		return newDenial(authorityv1.ErrorPolicyDenied, ruleLifecycleLease, work.LifecycleState, "present the current owning implementation lease", "lease.read", traceRef)
	}
	if !stored.Active && stored.State == authorityv1.LeaseReleased {
		return nil
	}
	if !stored.Active || stored.State != authorityv1.LeaseActive || !stored.ExpiresAt.After(s.now().UTC()) {
		return newDenial(authorityv1.ErrorPolicyDenied, ruleLifecycleLease, work.LifecycleState, "obtain a current implementation lease", "work.claim", traceRef)
	}
	released, err := s.leaseOps.Release(ctx, fence)
	if err != nil || released.Active || released.State != authorityv1.LeaseReleased || !leaseTupleMatches(released, fence) {
		return newDenial(authorityv1.ErrorUnknownEffect, ruleLifecycleUnknown, work.LifecycleState, requiredLifecycleReconcile, "work.reconcile", traceRef)
	}
	return nil
}

func lifecycleReplayMatches(work authorityv1.WorkItem, mutation LifecycleMutation) bool {
	switch mutation.Operation {
	case LifecycleHandoff:
		if handoffReplayMatches(work.Handoff, mutation) {
			return true
		}
		for _, cycle := range work.ReviewHistory {
			if handoffReplayMatches(&cycle.Handoff, mutation) {
				return true
			}
		}
	case LifecycleReview:
		for _, review := range work.Reviews {
			if reviewReplayMatches(review, mutation) {
				return true
			}
		}
		for _, cycle := range work.ReviewHistory {
			for _, review := range cycle.Reviews {
				if reviewReplayMatches(review, mutation) {
					return true
				}
			}
		}
	case LifecycleRun:
		if runReplayMatches(work.RunDisposition, mutation) {
			return true
		}
		for index := range work.RunHistory {
			if runReplayMatches(&work.RunHistory[index], mutation) {
				return true
			}
		}
		for _, cycle := range work.ReviewHistory {
			if runReplayMatches(cycle.RunDisposition, mutation) {
				return true
			}
			for index := range cycle.RunHistory {
				if runReplayMatches(&cycle.RunHistory[index], mutation) {
					return true
				}
			}
		}
	case LifecycleReconcile:
		return work.Reconciliation != nil && work.Reconciliation.IdempotencyKey == mutation.IdempotencyKey &&
			work.Reconciliation.PrincipalProfileID == mutation.PrincipalProfileID && work.Reconciliation.HeadSHA == mutation.HeadSHA &&
			work.Reconciliation.MergedSHA == mutation.MergedSHA && work.Reconciliation.MergedTree == mutation.MergedTree &&
			work.Reconciliation.PullRequestID == mutation.PullRequestID && work.Reconciliation.ProtectedMainRunID == mutation.ProtectedMainRunID &&
			equalStrings(work.Reconciliation.EvidenceRefs, mutation.EvidenceRefs)
	case LifecycleTerminal:
		return work.LifecycleState == authorityv1.LifecycleDone && work.Terminal != nil && work.Terminal.IdempotencyKey == mutation.IdempotencyKey &&
			work.Terminal.PrincipalProfileID == mutation.PrincipalProfileID && work.Terminal.HeadSHA == mutation.HeadSHA &&
			equalStrings(work.Terminal.EvidenceRefs, mutation.EvidenceRefs)
	}
	return false
}

func handoffReplayMatches(handoff *authorityv1.HandoffRecord, mutation LifecycleMutation) bool {
	return handoff != nil && handoff.AttemptID == mutation.AttemptID && handoff.CanonicalClaimAttemptID == mutation.CanonicalClaimAttemptID &&
		handoff.FenceDigest == mutation.HandoffFenceDigest && handoff.HeadSHA == mutation.HeadSHA &&
		handoff.NextProfileID == mutation.NextProfileID && handoff.IdempotencyKey == mutation.IdempotencyKey &&
		equalStrings(handoff.EvidenceRefs, mutation.EvidenceRefs)
}

func reviewReplayMatches(review authorityv1.ReviewRecord, mutation LifecycleMutation) bool {
	return review.IdempotencyKey == mutation.IdempotencyKey && review.ReviewerProfileID == mutation.PrincipalProfileID &&
		review.HeadSHA == mutation.HeadSHA && review.Verdict == mutation.Verdict && equalStrings(review.EvidenceRefs, mutation.EvidenceRefs) &&
		equalFailureContext(review.Failure, mutation.Failure)
}

func runReplayMatches(run *authorityv1.RunDispositionRecord, mutation LifecycleMutation) bool {
	return run != nil && run.IdempotencyKey == mutation.IdempotencyKey && run.PrincipalProfileID == mutation.PrincipalProfileID &&
		run.HeadSHA == mutation.HeadSHA && run.Status == mutation.RunStatus && equalStrings(run.EvidenceRefs, mutation.EvidenceRefs) &&
		equalFailureContext(run.Failure, mutation.Failure)
}

func lifecycleIdempotencyUsed(work authorityv1.WorkItem, key string) bool {
	if work.Handoff != nil && work.Handoff.IdempotencyKey == key {
		return true
	}
	for _, review := range work.Reviews {
		if review.IdempotencyKey == key {
			return true
		}
	}
	for _, cycle := range work.ReviewHistory {
		if cycle.Handoff.IdempotencyKey == key {
			return true
		}
		for _, review := range cycle.Reviews {
			if review.IdempotencyKey == key {
				return true
			}
		}
		for _, run := range cycle.RunHistory {
			if run.IdempotencyKey == key {
				return true
			}
		}
		if cycle.RunDisposition != nil && cycle.RunDisposition.IdempotencyKey == key {
			return true
		}
	}
	for _, run := range work.RunHistory {
		if run.IdempotencyKey == key {
			return true
		}
	}
	return work.RunDisposition != nil && work.RunDisposition.IdempotencyKey == key ||
		work.Reconciliation != nil && work.Reconciliation.IdempotencyKey == key ||
		work.Terminal != nil && work.Terminal.IdempotencyKey == key
}

func acceptedReviews(work authorityv1.WorkItem, headSHA string) bool {
	if work.LifecycleState != authorityv1.LifecycleInReview || work.Handoff == nil || work.Handoff.HeadSHA != headSHA ||
		len(work.Reviews) != len(work.VerificationOrder)-1 {
		return false
	}
	for index, review := range work.Reviews {
		if review.ReviewerProfileID != work.VerificationOrder[index] || review.Verdict != authorityv1.ReviewAccepted || review.HeadSHA != headSHA {
			return false
		}
	}
	return true
}

func runDispositionAllowed(work authorityv1.WorkItem, mutation LifecycleMutation) bool {
	if work.Handoff == nil || work.Handoff.HeadSHA != mutation.HeadSHA || work.Reconciliation != nil || work.Terminal != nil ||
		work.RunDisposition != nil && work.RunDisposition.Status == authorityv1.RunCompleted {
		return false
	}
	if mutation.RunStatus == authorityv1.RunCompleted {
		return acceptedReviews(work, mutation.HeadSHA)
	}
	if work.LifecycleState != authorityv1.LifecycleInProgress && work.LifecycleState != authorityv1.LifecycleInReview {
		return false
	}
	for index, review := range work.Reviews {
		if index >= len(work.VerificationOrder)-1 || review.ReviewerProfileID != work.VerificationOrder[index] || review.HeadSHA != mutation.HeadSHA ||
			(index < len(work.Reviews)-1 && review.Verdict != authorityv1.ReviewAccepted) {
			return false
		}
	}
	return true
}

func (s *Service) appendLifecycleEvent(ctx context.Context, principal authorityv1.Principal, traceRef string, mutation LifecycleMutation, labels []authorityv1.Label, phase, outcome string, before, after authorityv1.WorkItem) error {
	operation := "work." + string(mutation.Operation) + "." + phase
	event := s.event(principal, operation, mutation.BeadID, traceRef, outcome, ruleLifecycleAllowed, labels)
	event.EventID = deterministicEventID(mutation.IdempotencyKey, string(mutation.Operation)+":"+phase)
	event.AttemptID = mutation.AttemptID
	if event.AttemptID == "" {
		event.AttemptID = before.ClaimAttemptID
	}
	event.IdempotencyKey = mutation.IdempotencyKey
	if after.BeadID != "" {
		event.CanonicalVersion = &after.Version
	}
	event.BeforeHash, event.AfterHash = deterministicJSONDigest(before), deterministicJSONDigest(after)
	if before.BeadID == "" {
		event.BeforeHash = ""
	}
	if after.BeadID == "" {
		event.AfterHash = ""
	}
	receipt, err := s.events.Append(ctx, event)
	if err != nil || !validEventReceipt(event, receipt) {
		return newDenial(authorityv1.ErrorAuthorityDown, ruleAuthorityUnavailable, before.LifecycleState, requiredLifecycleReconcile, "work.reconcile", traceRef)
	}
	return nil
}

func (s *Service) appendLifecycleUnknown(ctx context.Context, principal authorityv1.Principal, traceRef string, mutation LifecycleMutation, labels []authorityv1.Label) error {
	event := s.event(principal, "work."+string(mutation.Operation)+".receipt", mutation.BeadID, traceRef, outcomeUnknown, ruleLifecycleUnknown, labels)
	event.EventID = deterministicEventID(mutation.IdempotencyKey, string(mutation.Operation)+":unknown")
	event.IdempotencyKey = mutation.IdempotencyKey
	_, err := s.events.Append(ctx, event)
	return err
}

func (s *Service) appendLifecycleReconciliationReceipt(ctx context.Context, principal authorityv1.Principal, traceRef string, mutation LifecycleMutation, labels []authorityv1.Label, work authorityv1.WorkItem) (string, error) {
	event := s.event(principal, "work."+string(mutation.Operation)+".reconciliation", mutation.BeadID, traceRef, outcomeVerified, ruleLifecycleAllowed, labels)
	event.EventID = deterministicEventID(mutation.IdempotencyKey, string(mutation.Operation)+":reconciliation")
	event.AttemptID = mutation.AttemptID
	if event.AttemptID == "" {
		event.AttemptID = work.ClaimAttemptID
	}
	event.IdempotencyKey = mutation.IdempotencyKey
	event.CanonicalVersion = &work.Version
	event.BeforeHash, event.AfterHash = deterministicJSONDigest(work), deterministicJSONDigest(work)
	receipt, err := s.events.Append(ctx, event)
	if err != nil || !validEventReceipt(event, receipt) {
		return "", newDenial(authorityv1.ErrorAuthorityDown, ruleAuthorityUnavailable, work.LifecycleState, requiredLifecycleReconcile, "work.reconcile", traceRef)
	}
	return fmt.Sprintf("lifecycle-reconciliation-%s-%s", mutation.Operation, event.EventID[4:]), nil
}

func lifecycleReceiptRef(mutation LifecycleMutation) string {
	return fmt.Sprintf("lifecycle-receipt-%s-%s", mutation.Operation, deterministicEventID(mutation.IdempotencyKey, "receipt")[4:])
}

func lifecycleCorrectiveAction(work authorityv1.WorkItem, mutation LifecycleMutation) string {
	if mutation.Operation == LifecycleReview && work.LifecycleState == authorityv1.LifecycleInReview && len(work.Reviews) < len(work.VerificationOrder)-1 {
		return "review.record(" + work.VerificationOrder[len(work.Reviews)] + ")"
	}
	return "work.read"
}

func noMergeFields(mutation LifecycleMutation) bool {
	return mutation.MergedSHA == "" && mutation.MergedTree == "" && mutation.PullRequestID == "" && mutation.ProtectedMainRunID == ""
}

func noAttemptFields(mutation LifecycleMutation) bool {
	return mutation.AttemptID == "" && mutation.CanonicalClaimAttemptID == "" && mutation.HandoffFenceDigest == ""
}

func normalizedFailureContext(value *authorityv1.FailureContext) *authorityv1.FailureContext {
	if value == nil {
		return nil
	}
	clone := *value
	clone.BlockedBy = sortedStrings(value.BlockedBy)
	return &clone
}

func validFailureContext(value *authorityv1.FailureContext, requireBlockedBy, requireFingerprint bool) bool {
	if value == nil || !validID(value.Reason) || !validID(value.NextAction) || value.Attempt == 0 || value.Attempt > 2 ||
		len(value.BlockedBy) > 16 || hasDuplicateStrings(value.BlockedBy) || requireBlockedBy && len(value.BlockedBy) == 0 ||
		requireFingerprint && !validID(value.FailureFingerprint) || !requireFingerprint && value.FailureFingerprint != "" && !validID(value.FailureFingerprint) {
		return false
	}
	for _, blocker := range value.BlockedBy {
		if !validID(blocker) {
			return false
		}
	}
	return true
}

func validFailureForReview(verdict authorityv1.ReviewVerdict, failure *authorityv1.FailureContext) bool {
	if verdict == authorityv1.ReviewBlocked {
		return validFailureContext(failure, true, true)
	}
	return failure == nil
}

func validFailureForRun(status authorityv1.RunDispositionStatus, failure *authorityv1.FailureContext) bool {
	if status == authorityv1.RunCompleted {
		return failure == nil
	}
	return validFailureContext(failure, status == authorityv1.RunBlocked, status != authorityv1.RunInReview && status != authorityv1.RunNoWork)
}

func equalFailureContext(left, right *authorityv1.FailureContext) bool {
	return left == nil && right == nil || left != nil && right != nil && left.Reason == right.Reason && left.FailureFingerprint == right.FailureFingerprint &&
		left.Attempt == right.Attempt && left.NextAction == right.NextAction && equalStrings(left.BlockedBy, right.BlockedBy)
}

func knownReviewVerdict(verdict authorityv1.ReviewVerdict) bool {
	return verdict == authorityv1.ReviewAccepted || verdict == authorityv1.ReviewChangesRequested || verdict == authorityv1.ReviewBlocked
}

func knownRunStatus(status authorityv1.RunDispositionStatus) bool {
	switch status {
	case authorityv1.RunCompleted, authorityv1.RunBlocked, authorityv1.RunInReview, authorityv1.RunChangesRequested,
		authorityv1.RunNoWork, authorityv1.RunPreempted, authorityv1.RunCancelled, authorityv1.RunFailed:
		return true
	default:
		return false
	}
}
