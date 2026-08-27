/*
FactoryDocSync:
docs:
- docs/features/F-002-work-authority.md
- docs/product-specs/work-authority.md
- docs/design-docs/ADR-001-git-beads-authority.md
- docs/code-documentation-map.md
*/

package beads

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"

	authorityv1 "github.com/greaveselliott/MARS-3/api/authority/v1"
	"github.com/greaveselliott/MARS-3/internal/authority/gateway"
)

func lifecyclePostMetadata(pre []byte, mutation gateway.LifecycleMutation) ([]byte, string, string, string, error) {
	if rejectDuplicateJSONKeys(pre) != nil || !validLifecycleMutationIdentity(mutation) {
		return nil, "", "", "", ErrProjectionInvalid
	}
	var metadata issueMetadata
	decoder := json.NewDecoder(bytes.NewReader(pre))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&metadata) != nil {
		return nil, "", "", "", ErrProjectionInvalid
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) || !validLifecycleRecords(metadata) ||
		metadata.WorkVersion.IssueMutationSequence == math.MaxUint64 {
		return nil, "", "", "", ErrProjectionInvalid
	}
	postStatus, removeLabel, addLabel := "in_progress", "", ""
	switch mutation.Operation {
	case gateway.LifecycleHandoff:
		if metadata.LifecycleState != authorityv1.LifecycleInProgress || mutation.PrincipalProfileID == "" ||
			mutation.AttemptID == "" || mutation.NextProfileID == "" || mutation.Verdict != "" || mutation.RunStatus != "" ||
			mutation.MergedSHA != "" || mutation.MergedTree != "" || mutation.PullRequestID != "" || mutation.ProtectedMainRunID != "" ||
			metadata.VerificationOrder[0] != mutation.NextProfileID || !claimMatchesAttempt(metadata, mutation.CanonicalClaimAttemptID) {
			return nil, "", "", "", ErrProjectionInvalid
		}
		if metadata.Handoff != nil {
			if len(metadata.ReviewRecords) == 0 || metadata.ReviewRecords[len(metadata.ReviewRecords)-1].Verdict != authorityv1.ReviewChangesRequested {
				return nil, "", "", "", ErrProjectionInvalid
			}
			metadata.ReviewHistory = append(metadata.ReviewHistory, metadataReviewCycle{
				Handoff: *metadata.Handoff, Reviews: append([]metadataReview(nil), metadata.ReviewRecords...),
			})
		}
		metadata.LifecycleState = authorityv1.LifecycleInReview
		metadata.Handoff = &metadataHandoff{
			AttemptID: mutation.AttemptID, HeadSHA: mutation.HeadSHA, EvidenceRefs: append([]string(nil), mutation.EvidenceRefs...),
			NextProfileID: mutation.NextProfileID, IdempotencyKey: mutation.IdempotencyKey,
		}
		metadata.ReviewRecords = nil
		metadata.ReviewAccepted = false
		metadata.RunDisposition = ""
		metadata.RunDispositionRecord = nil
		metadata.Reconciled = false
		metadata.ReconciliationRecord = nil
		metadata.TerminalRecord = nil
		removeLabel, addLabel = "in-progress", "in-review"
	case gateway.LifecycleReview:
		if metadata.LifecycleState != authorityv1.LifecycleInReview || metadata.Handoff == nil || mutation.HeadSHA != metadata.Handoff.HeadSHA ||
			mutation.AttemptID != "" || mutation.CanonicalClaimAttemptID != "" || mutation.NextProfileID != "" || mutation.RunStatus != "" || mutation.MergedSHA != "" ||
			mutation.MergedTree != "" || mutation.PullRequestID != "" || mutation.ProtectedMainRunID != "" ||
			len(metadata.ReviewRecords) >= len(metadata.VerificationOrder)-1 || metadata.VerificationOrder[len(metadata.ReviewRecords)] != mutation.PrincipalProfileID ||
			!knownReviewVerdict(mutation.Verdict) {
			return nil, "", "", "", ErrProjectionInvalid
		}
		metadata.ReviewRecords = append(metadata.ReviewRecords, metadataReview{
			ReviewerProfileID: mutation.PrincipalProfileID, Verdict: mutation.Verdict, HeadSHA: mutation.HeadSHA,
			EvidenceRefs: append([]string(nil), mutation.EvidenceRefs...), IdempotencyKey: mutation.IdempotencyKey,
		})
		if mutation.Verdict == authorityv1.ReviewChangesRequested {
			metadata.LifecycleState = authorityv1.LifecycleInProgress
			metadata.ReviewAccepted = false
			removeLabel, addLabel = "in-review", "in-progress"
		} else if mutation.Verdict == authorityv1.ReviewAccepted && len(metadata.ReviewRecords) == len(metadata.VerificationOrder)-1 {
			metadata.ReviewAccepted = true
		}
	case gateway.LifecycleRun:
		if !acceptedReviewChain(metadata, mutation.HeadSHA) || mutation.PrincipalProfileID != metadata.Coordinator ||
			mutation.AttemptID != "" || mutation.CanonicalClaimAttemptID != "" || mutation.NextProfileID != "" || mutation.Verdict != "" || mutation.MergedSHA != "" ||
			mutation.MergedTree != "" || mutation.PullRequestID != "" || mutation.ProtectedMainRunID != "" || !knownRunDisposition(mutation.RunStatus) {
			return nil, "", "", "", ErrProjectionInvalid
		}
		metadata.RunDisposition = string(mutation.RunStatus)
		metadata.RunDispositionRecord = &metadataRunDisposition{
			PrincipalProfileID: mutation.PrincipalProfileID, Status: mutation.RunStatus, HeadSHA: mutation.HeadSHA,
			EvidenceRefs: append([]string(nil), mutation.EvidenceRefs...), IdempotencyKey: mutation.IdempotencyKey,
		}
	case gateway.LifecycleReconcile:
		if !acceptedReviewChain(metadata, mutation.HeadSHA) || metadata.RunDispositionRecord == nil ||
			metadata.RunDispositionRecord.Status != authorityv1.RunCompleted || mutation.PrincipalProfileID != metadata.Coordinator ||
			mutation.AttemptID != "" || mutation.CanonicalClaimAttemptID != "" || mutation.NextProfileID != "" || mutation.Verdict != "" || mutation.RunStatus != "" ||
			!commitID(mutation.MergedSHA) || !commitID(mutation.MergedTree) || !safeToken(mutation.PullRequestID) || !safeToken(mutation.ProtectedMainRunID) {
			return nil, "", "", "", ErrProjectionInvalid
		}
		metadata.Reconciled = true
		metadata.ReconciliationRecord = &metadataReconciliation{
			PrincipalProfileID: mutation.PrincipalProfileID, HeadSHA: mutation.HeadSHA, MergedSHA: mutation.MergedSHA,
			MergedTree: mutation.MergedTree, PullRequestID: mutation.PullRequestID, ProtectedMainRunID: mutation.ProtectedMainRunID,
			EvidenceRefs: append([]string(nil), mutation.EvidenceRefs...), IdempotencyKey: mutation.IdempotencyKey,
		}
	case gateway.LifecycleTerminal:
		if !acceptedReviewChain(metadata, mutation.HeadSHA) || metadata.RunDispositionRecord == nil ||
			metadata.RunDispositionRecord.Status != authorityv1.RunCompleted || metadata.ReconciliationRecord == nil ||
			mutation.PrincipalProfileID != metadata.Coordinator || mutation.AttemptID != "" || mutation.CanonicalClaimAttemptID != "" || mutation.NextProfileID != "" ||
			mutation.Verdict != "" || mutation.RunStatus != "" || mutation.MergedSHA != "" || mutation.MergedTree != "" ||
			mutation.PullRequestID != "" || mutation.ProtectedMainRunID != "" {
			return nil, "", "", "", ErrProjectionInvalid
		}
		metadata.LifecycleState = authorityv1.LifecycleDone
		metadata.ReviewAccepted = true
		metadata.TerminalRecord = &metadataTerminal{
			PrincipalProfileID: mutation.PrincipalProfileID, HeadSHA: mutation.HeadSHA,
			EvidenceRefs: append([]string(nil), mutation.EvidenceRefs...), IdempotencyKey: mutation.IdempotencyKey,
		}
		postStatus, removeLabel, addLabel = "closed", "in-review", "done"
	default:
		return nil, "", "", "", ErrProjectionInvalid
	}
	metadata.WorkVersion.IssueMutationSequence++
	post, err := json.Marshal(metadata)
	if err != nil || !validLifecycleRecords(metadata) {
		return nil, "", "", "", ErrProjectionInvalid
	}
	return post, postStatus, removeLabel, addLabel, nil
}

func validLifecycleMutationIdentity(mutation gateway.LifecycleMutation) bool {
	return safeToken(mutation.BeadID) && safeToken(mutation.PrincipalProfileID) && commitID(mutation.HeadSHA) &&
		safeToken(mutation.IdempotencyKey) && validEvidenceRefs(mutation.EvidenceRefs)
}

func claimMatchesAttempt(metadata issueMetadata, attemptID string) bool {
	binding := metadata.WorkClaim
	if binding == nil {
		binding = metadata.BootstrapClaim
	}
	return binding != nil && binding.AttemptID == attemptID
}

func acceptedReviewChain(metadata issueMetadata, headSHA string) bool {
	if metadata.LifecycleState != authorityv1.LifecycleInReview || metadata.Handoff == nil || metadata.Handoff.HeadSHA != headSHA ||
		len(metadata.ReviewRecords) != len(metadata.VerificationOrder)-1 {
		return false
	}
	for index, review := range metadata.ReviewRecords {
		if review.ReviewerProfileID != metadata.VerificationOrder[index] || review.Verdict != authorityv1.ReviewAccepted || review.HeadSHA != headSHA {
			return false
		}
	}
	return true
}

func validLifecyclePostimage(pre, post authorityv1.WorkItem, mutation gateway.LifecycleMutation) bool {
	if post.TenantID != pre.TenantID || post.ProjectID != pre.ProjectID || post.BeadID != pre.BeadID ||
		post.Version.AuthorityGeneration != pre.Version.AuthorityGeneration || post.Version.IssueIncarnation != pre.Version.IssueIncarnation ||
		post.Version.IssueMutationSequence != pre.Version.IssueMutationSequence+1 ||
		post.Version.DependencyGraphRevision != pre.Version.DependencyGraphRevision || post.Assignee != pre.Assignee ||
		post.ClaimAttemptID != pre.ClaimAttemptID || !equalStringSlices(post.ExclusivePaths, pre.ExclusivePaths) {
		return false
	}
	switch mutation.Operation {
	case gateway.LifecycleHandoff:
		historyIncrement := 0
		if pre.Handoff != nil {
			historyIncrement = 1
		}
		return post.LifecycleState == authorityv1.LifecycleInReview && post.NativeStatus == "in_progress" && post.Handoff != nil &&
			post.Handoff.IdempotencyKey == mutation.IdempotencyKey && len(post.ReviewHistory) == len(pre.ReviewHistory)+historyIncrement
	case gateway.LifecycleReview:
		if len(post.Reviews) != len(pre.Reviews)+1 || post.Reviews[len(post.Reviews)-1].IdempotencyKey != mutation.IdempotencyKey {
			return false
		}
		if mutation.Verdict == authorityv1.ReviewChangesRequested {
			return post.LifecycleState == authorityv1.LifecycleInProgress
		}
		return post.LifecycleState == authorityv1.LifecycleInReview
	case gateway.LifecycleRun:
		return post.RunDisposition != nil && post.RunDisposition.IdempotencyKey == mutation.IdempotencyKey
	case gateway.LifecycleReconcile:
		return post.Reconciliation != nil && post.Reconciliation.IdempotencyKey == mutation.IdempotencyKey
	case gateway.LifecycleTerminal:
		return post.LifecycleState == authorityv1.LifecycleDone && post.NativeStatus == "closed" && post.Terminal != nil &&
			post.Terminal.IdempotencyKey == mutation.IdempotencyKey && equalStringSlices(post.Terminal.EvidenceRefs, mutation.EvidenceRefs)
	default:
		return false
	}
}

func equalStringSlices(left, right []string) bool {
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
