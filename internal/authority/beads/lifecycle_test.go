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
	"encoding/json"
	"errors"
	"strings"
	"testing"

	authorityv1 "github.com/greaveselliott/MARS-3/api/authority/v1"
	"github.com/greaveselliott/MARS-3/internal/authority/gateway"
)

func TestLifecycleMetadataPreservesReworkHistoryAndTerminalPrerequisites(t *testing.T) {
	raw := lifecycleMetadataFixture(t)
	firstHead := strings.Repeat("a", 40)
	secondHead := strings.Repeat("b", 40)

	raw = applyLifecycleMetadata(t, raw, gateway.LifecycleMutation{
		Operation: gateway.LifecycleHandoff, PrincipalProfileID: "work-authority-engineer", AttemptID: "execution-attempt-1",
		CanonicalClaimAttemptID: "canonical-attempt", HandoffFenceDigest: strings.Repeat("1", 64), HeadSHA: firstHead, EvidenceRefs: []string{"evidence-first"},
		NextProfileID: "qa", IdempotencyKey: "handoff-first",
	}, "in_progress", "in-progress", "in-review")
	raw = applyLifecycleMetadata(t, raw, gateway.LifecycleMutation{
		Operation: gateway.LifecycleReview, PrincipalProfileID: "qa", HeadSHA: firstHead, Verdict: authorityv1.ReviewChangesRequested,
		EvidenceRefs: []string{"evidence-changes"}, IdempotencyKey: "review-changes",
	}, "in_progress", "in-review", "in-progress")
	raw = applyLifecycleMetadata(t, raw, gateway.LifecycleMutation{
		Operation: gateway.LifecycleHandoff, PrincipalProfileID: "work-authority-engineer", AttemptID: "execution-attempt-2",
		CanonicalClaimAttemptID: "canonical-attempt", HandoffFenceDigest: strings.Repeat("2", 64), HeadSHA: secondHead, EvidenceRefs: []string{"evidence-second"},
		NextProfileID: "qa", IdempotencyKey: "handoff-second",
	}, "in_progress", "in-progress", "in-review")

	var historyState issueMetadata
	if json.Unmarshal(raw, &historyState) != nil || len(historyState.ReviewHistory) != 1 ||
		historyState.ReviewHistory[0].Handoff.HeadSHA != firstHead ||
		historyState.ReviewHistory[0].Reviews[0].Verdict != authorityv1.ReviewChangesRequested ||
		historyState.Handoff.HeadSHA != secondHead || len(historyState.ReviewRecords) != 0 {
		t.Fatalf("history state=%#v", historyState)
	}

	for _, reviewer := range []string{"qa", "security-reviewer"} {
		raw = applyLifecycleMetadata(t, raw, gateway.LifecycleMutation{
			Operation: gateway.LifecycleReview, PrincipalProfileID: reviewer, HeadSHA: secondHead, Verdict: authorityv1.ReviewAccepted,
			EvidenceRefs: []string{"evidence-" + reviewer}, IdempotencyKey: "review-" + reviewer,
		}, "in_progress", "", "")
	}
	raw = applyLifecycleMetadata(t, raw, gateway.LifecycleMutation{
		Operation: gateway.LifecycleRun, PrincipalProfileID: "delivery-orchestrator", HeadSHA: secondHead,
		RunStatus: authorityv1.RunCompleted, EvidenceRefs: []string{"evidence-run"}, IdempotencyKey: "run-completed",
	}, "in_progress", "", "")
	raw = applyLifecycleMetadata(t, raw, gateway.LifecycleMutation{
		Operation: gateway.LifecycleReconcile, PrincipalProfileID: "delivery-orchestrator", HeadSHA: secondHead,
		MergedSHA: strings.Repeat("c", 40), MergedTree: strings.Repeat("d", 40), PullRequestID: "pr-009", ProtectedMainRunID: "run-009",
		EvidenceRefs: []string{"evidence-merge"}, IdempotencyKey: "reconciled",
	}, "in_progress", "", "")
	raw = applyLifecycleMetadata(t, raw, gateway.LifecycleMutation{
		Operation: gateway.LifecycleTerminal, PrincipalProfileID: "delivery-orchestrator", HeadSHA: secondHead,
		EvidenceRefs: []string{"evidence-terminal"}, IdempotencyKey: "terminal-done",
	}, "closed", "in-review", "done")

	var terminal issueMetadata
	if json.Unmarshal(raw, &terminal) != nil || !validLifecycleRecords(terminal) || terminal.LifecycleState != authorityv1.LifecycleDone ||
		terminal.WorkVersion.IssueMutationSequence != 10 || terminal.TerminalRecord == nil || len(terminal.ReviewHistory) != 1 {
		t.Fatalf("terminal metadata=%#v", terminal)
	}
	missingClaim := terminal
	missingClaim.WorkClaim, missingClaim.BootstrapClaim = nil, nil
	if validLifecycleRecords(missingClaim) {
		t.Fatal("terminal metadata without claim lineage was accepted")
	}
	dualClaim := terminal
	bootstrap := *terminal.WorkClaim
	bootstrap.GrantID = "bootstrap-grant"
	dualClaim.BootstrapClaim = &bootstrap
	if validLifecycleRecords(dualClaim) {
		t.Fatal("terminal metadata with dual claim lineage was accepted")
	}
	stripped := terminal
	stripped.Handoff, stripped.RunDispositionRecord, stripped.ReconciliationRecord, stripped.TerminalRecord = nil, nil, nil, nil
	stripped.ReviewRecords, stripped.ReviewHistory, stripped.RunHistory = nil, nil, nil
	if validLifecycleRecords(stripped) {
		t.Fatal("terminal metadata without detailed evidence was accepted")
	}
}

func TestLifecycleMetadataRejectsWrongCanonicalClaimAndPrematureTerminal(t *testing.T) {
	raw := lifecycleMetadataFixture(t)
	_, _, _, _, err := lifecyclePostMetadata(raw, gateway.LifecycleMutation{
		BeadID: "M3-W002", Operation: gateway.LifecycleHandoff, PrincipalProfileID: "work-authority-engineer", AttemptID: "execution-attempt",
		CanonicalClaimAttemptID: "wrong-canonical-attempt", HandoffFenceDigest: strings.Repeat("3", 64), HeadSHA: strings.Repeat("a", 40), EvidenceRefs: []string{"evidence-handoff"},
		NextProfileID: "qa", IdempotencyKey: "handoff-wrong",
	})
	if !errors.Is(err, ErrProjectionInvalid) {
		t.Fatalf("wrong canonical claim error=%v", err)
	}
	_, _, _, _, err = lifecyclePostMetadata(raw, gateway.LifecycleMutation{
		BeadID: "M3-W002", Operation: gateway.LifecycleTerminal, PrincipalProfileID: "delivery-orchestrator", HeadSHA: strings.Repeat("a", 40),
		EvidenceRefs: []string{"evidence-terminal"}, IdempotencyKey: "terminal-premature",
	})
	if !errors.Is(err, ErrProjectionInvalid) {
		t.Fatalf("premature terminal error=%v", err)
	}
}

func lifecycleMetadataFixture(t *testing.T) []byte {
	t.Helper()
	raw := issueFixture(t, authorityv1.LifecycleInProgress, "canonical-attempt", "work-authority-engineer", 2, false)
	snapshot, err := decodeIssueSnapshot(raw, "tenant-fixture", "project-fixture", []authorityv1.Label{authorityv1.LabelPublicAccepted})
	if err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	return snapshot.MetadataRaw
}

func applyLifecycleMetadata(t *testing.T, raw []byte, mutation gateway.LifecycleMutation, wantStatus, wantRemove, wantAdd string) []byte {
	t.Helper()
	mutation.BeadID = "M3-W002"
	post, status, remove, add, err := lifecyclePostMetadata(raw, mutation)
	if err != nil {
		t.Fatalf("%s: %v", mutation.Operation, err)
	}
	if status != wantStatus || remove != wantRemove || add != wantAdd {
		t.Fatalf("%s transition=(%s,%s,%s), want (%s,%s,%s)", mutation.Operation, status, remove, add, wantStatus, wantRemove, wantAdd)
	}
	return post
}
