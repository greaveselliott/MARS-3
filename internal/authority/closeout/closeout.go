/*
FactoryDocSync:
docs:
- docs/features/F-002-work-authority.md
- docs/product-specs/work-authority.md
- docs/exec-plans/active/current-operating-plan.md
- docs/evidence/W-001-validation.md
*/

// Package closeout owns the single bounded W-001 terminal-reconciliation
// workflow. It composes existing authority stores and invokes only typed
// gateway operations; the command surface never receives a raw mutation API.
package closeout

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	authorityv1 "github.com/greaveselliott/MARS-3/api/authority/v1"
	"github.com/greaveselliott/MARS-3/internal/authority/beads"
	"github.com/greaveselliott/MARS-3/internal/authority/gateway"
	"github.com/greaveselliott/MARS-3/internal/authority/postgres"
	"github.com/greaveselliott/MARS-3/internal/doctrine"
)

var (
	ErrConfiguration      = errors.New("terminal reconciliation configuration is invalid")
	ErrCanonicalPreimage  = errors.New("canonical W-001 preimage does not match the signed terminal grant")
	ErrExecutionBoundary  = errors.New("terminal reconciliation execution boundary is invalid")
	ErrReconciliation     = errors.New("terminal reconciliation failed")
	boundedToken          = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
	lowerHexDigest        = regexp.MustCompile(`^[a-f0-9]{64}$`)
	terminalGitExecutable = "/usr/bin/git"
)

// Options accepts store handles by path. Secret contents are read only inside
// Run and are never returned, logged, or passed as command-line arguments.
type Options struct {
	Repo                   string
	TenantID               string
	ProjectID              string
	BeadsWorkspace         string
	BeadsBinary            string
	BeadsBinarySHA256      string
	PostgreSQLURLFile      string
	FenceGeneration        string
	ExecutionAuthorization string
	Apply                  bool
	Now                    func() time.Time
	Stdout                 io.Writer
	Stderr                 io.Writer
}

// Receipt is the public-safe exact readback. It deliberately excludes paths,
// datastore addresses, credentials, raw authority documents, and event bodies.
type Receipt struct {
	SchemaVersion      uint32                           `json:"schema_version"`
	GrantID            string                           `json:"grant_id"`
	BeadID             string                           `json:"bead_id"`
	Mode               string                           `json:"mode"`
	Result             string                           `json:"result"`
	NativeStatus       string                           `json:"native_status"`
	LifecycleState     authorityv1.LifecycleState       `json:"lifecycle_state"`
	Version            authorityv1.WorkVersion          `json:"work_version"`
	LiveLeasePresent   bool                             `json:"live_lease_present"`
	HandoffHead        string                           `json:"handoff_head,omitempty"`
	ReviewCount        int                              `json:"review_count"`
	RunStatus          authorityv1.RunDispositionStatus `json:"run_status,omitempty"`
	MergedSHA          string                           `json:"merged_sha,omitempty"`
	MergedTree         string                           `json:"merged_tree,omitempty"`
	PullRequestID      string                           `json:"pull_request_id,omitempty"`
	ProtectedMainRunID string                           `json:"protected_main_run_id,omitempty"`
	TerminalRecorded   bool                             `json:"terminal_recorded"`
}

type lifecycleClient interface {
	ReconcileClaimedWork(context.Context, authorityv1.Principal, gateway.ClaimReconciliationRequest) (authorityv1.ClaimResponse, error)
	Handoff(context.Context, authorityv1.Principal, authorityv1.HandoffRequest) (authorityv1.LifecycleMutationResponse, error)
	RecordReviewVerdict(context.Context, authorityv1.Principal, authorityv1.ReviewVerdictRequest) (authorityv1.LifecycleMutationResponse, error)
	RecordRunDisposition(context.Context, authorityv1.Principal, authorityv1.RunDispositionRequest) (authorityv1.LifecycleMutationResponse, error)
	RecordReconciliation(context.Context, authorityv1.Principal, authorityv1.ReconciliationRequest) (authorityv1.LifecycleMutationResponse, error)
	CloseWork(context.Context, authorityv1.Principal, authorityv1.TerminalTransitionRequest) (authorityv1.LifecycleMutationResponse, error)
}

type workReader interface {
	Get(context.Context, string, string, string) (authorityv1.WorkItem, error)
}

type leaseReader interface {
	ActiveLeaseForBead(context.Context, string, string, string) (authorityv1.CapabilityLease, bool, error)
}

type sagaReader interface {
	Lookup(context.Context, string, string, string) (gateway.ClaimSaga, bool, error)
}

type dependencies struct {
	client lifecycleClient
	work   workReader
	leases leaseReader
	sagas  sagaReader
}

// Run verifies the signed grant and post-review execution token, opens the
// canonical stores, and either proves readiness without mutation or executes
// the idempotent gateway-only sequence once.
func Run(ctx context.Context, options Options) error {
	options = withDefaults(options)
	if err := validateOptions(options); err != nil {
		return err
	}
	grant, err := doctrine.LoadW001TerminalReconciliationGrant(options.Repo, options.Now().UTC())
	if err != nil {
		return ErrConfiguration
	}
	authorization, err := doctrine.LoadW001TerminalReconciliationExecutionAuthorization(
		options.Repo, options.ExecutionAuthorization, grant, options.Now().UTC(),
	)
	if err != nil {
		return ErrExecutionBoundary
	}
	if authorization.TenantID != options.TenantID || authorization.ProjectID != options.ProjectID ||
		authorization.FenceGeneration != options.FenceGeneration ||
		authorization.BeadsBinarySHA256 != options.BeadsBinarySHA256 {
		return ErrExecutionBoundary
	}
	if err := verifyRepository(options.Repo, authorization); err != nil {
		return ErrExecutionBoundary
	}
	if digest, err := digestRegularFile(options.BeadsBinary); err != nil || digest != options.BeadsBinarySHA256 {
		return ErrExecutionBoundary
	}
	workspaceDigest, err := workspaceInstanceDigest(options.BeadsWorkspace, options.ProjectID)
	if err != nil || workspaceDigest != authorization.WorkspaceInstanceSHA256 {
		return ErrExecutionBoundary
	}
	reader, err := beads.NewCLIReader(options.BeadsBinary, options.BeadsBinarySHA256, options.BeadsWorkspace, options.ProjectID)
	if err != nil {
		return ErrExecutionBoundary
	}
	mutator, err := beads.NewNativeMutator(reader, options.BeadsBinary, options.BeadsBinarySHA256)
	if err != nil {
		return ErrExecutionBoundary
	}
	work, err := beads.New(options.TenantID, options.ProjectID, []authorityv1.Label{authorityv1.LabelPublicAccepted}, reader, mutator)
	if err != nil {
		return ErrConfiguration
	}
	url, err := readSecretFile(options.PostgreSQLURLFile)
	if err != nil {
		return ErrConfiguration
	}
	pool, err := pgxpool.New(ctx, url)
	zeroString(&url)
	if err != nil {
		return ErrConfiguration
	}
	defer pool.Close()
	operational, err := postgres.New(pool, func(_ context.Context, tenantID, projectID string) (string, error) {
		if tenantID != options.TenantID || projectID != options.ProjectID {
			return "", postgres.ErrFenceGeneration
		}
		return options.FenceGeneration, nil
	}, options.Now, nil)
	if err != nil {
		return ErrConfiguration
	}
	client, err := gateway.NewWithClaims(work, operational, operational, options.Now)
	if err != nil {
		return ErrConfiguration
	}
	receipt, err := runWithDependencies(ctx, grant, authorization, dependencies{client: client, work: work, leases: operational, sagas: operational}, options.Apply)
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(receipt)
	if err != nil {
		return ErrReconciliation
	}
	_, err = fmt.Fprintln(options.Stdout, string(encoded))
	return err
}

func withDefaults(options Options) Options {
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.Stdout == nil {
		options.Stdout = io.Discard
	}
	if options.Stderr == nil {
		options.Stderr = io.Discard
	}
	return options
}

func validateOptions(options Options) error {
	for _, value := range []string{options.Repo, options.TenantID, options.ProjectID, options.BeadsWorkspace, options.BeadsBinary,
		options.BeadsBinarySHA256, options.PostgreSQLURLFile, options.FenceGeneration, options.ExecutionAuthorization} {
		if value == "" {
			return ErrConfiguration
		}
	}
	if !boundedToken.MatchString(options.TenantID) || !boundedToken.MatchString(options.ProjectID) ||
		!boundedToken.MatchString(options.FenceGeneration) || !lowerHexDigest.MatchString(options.BeadsBinarySHA256) {
		return ErrConfiguration
	}
	return nil
}

func runWithDependencies(ctx context.Context, grant doctrine.W001TerminalReconciliationGrant, authorization doctrine.W001TerminalReconciliationExecutionAuthorization, deps dependencies, apply bool) (Receipt, error) {
	if deps.client == nil || deps.work == nil || deps.leases == nil || deps.sagas == nil {
		return Receipt{}, ErrConfiguration
	}
	item, err := deps.work.Get(ctx, authorization.TenantID, authorization.ProjectID, grant.Bead)
	if err != nil {
		return Receipt{}, ErrCanonicalPreimage
	}
	lease, active, err := deps.leases.ActiveLeaseForBead(ctx, authorization.TenantID, authorization.ProjectID, grant.Bead)
	if err != nil {
		return Receipt{}, ErrReconciliation
	}
	if !apply {
		if active || !matchesInitialPreimage(item, grant) {
			return Receipt{}, ErrCanonicalPreimage
		}
		second, err := deps.work.Get(ctx, authorization.TenantID, authorization.ProjectID, grant.Bead)
		if err != nil || !reflect.DeepEqual(second, item) {
			return Receipt{}, ErrCanonicalPreimage
		}
		return receiptFor(grant, item, false, "dry-run", "ready-no-mutation"), nil
	}
	if active && (lease.AttemptID != grant.AttemptID || lease.BaseSHA != authorization.MergedCommit || lease.FenceGeneration != authorization.FenceGeneration) {
		return Receipt{}, ErrCanonicalPreimage
	}
	item, err = applySequence(ctx, grant, authorization, deps, item)
	if err != nil {
		return Receipt{}, err
	}
	_, active, err = deps.leases.ActiveLeaseForBead(ctx, authorization.TenantID, authorization.ProjectID, grant.Bead)
	if err != nil || active || !matchesTerminalPostimage(item, grant) {
		return Receipt{}, ErrReconciliation
	}
	return receiptFor(grant, item, false, "apply", "terminal-close-verified"), nil
}

func applySequence(ctx context.Context, grant doctrine.W001TerminalReconciliationGrant, authorization doctrine.W001TerminalReconciliationExecutionAuthorization, deps dependencies, item authorityv1.WorkItem) (authorityv1.WorkItem, error) {
	engineer := principal(authorization, "work-authority-engineer", authorityv1.CapabilityWorkRead, authorityv1.CapabilityWorkClaim,
		authorityv1.CapabilityLeaseIssue, authorityv1.CapabilityWorkHandoff, authorityv1.CapabilityLeaseRelease, authorityv1.CapabilityTicketDelivery)
	qa := principal(authorization, "qa", authorityv1.CapabilityReviewRecord)
	security := principal(authorization, "security-reviewer", authorityv1.CapabilityReviewRecord)
	orchestrator := principal(authorization, "delivery-orchestrator", authorityv1.CapabilityRunDisposition, authorityv1.CapabilityWorkReconcile, authorityv1.CapabilityWorkClose)

	if item.LifecycleState == authorityv1.LifecycleInProgress {
		if !matchesInitialPreimage(item, grant) {
			return authorityv1.WorkItem{}, ErrCanonicalPreimage
		}
		request := gateway.ClaimReconciliationRequest{
			ClaimRequest: authorityv1.ClaimRequest{
				BeadID: item.BeadID, ExpectedVersion: item.Version, ExpectedIntegrity: item.Integrity,
				AttemptID: grant.AttemptID, BaseSHA: authorization.MergedCommit, ExclusivePaths: append([]string(nil), item.ExclusivePaths...),
				Capability: authorityv1.CapabilityTicketDelivery, IdempotencyKey: operationKey(grant, "lease"), TraceRef: operationTrace("lease"),
			},
			CanonicalClaimAttemptID: grant.CanonicalClaimAttemptID,
		}
		lease, err := reconcileLease(ctx, deps, engineer, request, item, grant)
		if err != nil {
			return authorityv1.WorkItem{}, err
		}
		response, err := deps.client.Handoff(ctx, engineer, authorityv1.HandoffRequest{
			BeadID: item.BeadID, ExpectedVersion: item.Version, ExpectedIntegrity: item.Integrity,
			Fence: fenceFromLease(lease), HeadSHA: grant.AcceptedCandidateHead,
			EvidenceRefs:  []string{"candidate-56c2a8d", "ci-33226093357", "qa-accepted", "security-accepted"},
			NextProfileID: "qa", IdempotencyKey: operationKey(grant, "handoff"), TraceRef: operationTrace("handoff"),
		})
		if err != nil {
			return authorityv1.WorkItem{}, wrapReconciliation(err)
		}
		item = response.Work
	}
	if item.LifecycleState == authorityv1.LifecycleDone {
		if matchesTerminalPostimage(item, grant) {
			return item, nil
		}
		return authorityv1.WorkItem{}, ErrCanonicalPreimage
	}
	if !matchesHandoff(item, grant) {
		return authorityv1.WorkItem{}, ErrCanonicalPreimage
	}
	if len(item.Reviews) == 0 {
		response, err := deps.client.RecordReviewVerdict(ctx, qa, reviewRequest(item, grant, "qa", "qa-accepted"))
		if err != nil {
			return authorityv1.WorkItem{}, wrapReconciliation(err)
		}
		item = response.Work
	}
	if len(item.Reviews) == 1 {
		if !reviewMatches(item.Reviews[0], "qa", grant.AcceptedCandidateHead) {
			return authorityv1.WorkItem{}, ErrCanonicalPreimage
		}
		response, err := deps.client.RecordReviewVerdict(ctx, security, reviewRequest(item, grant, "security-reviewer", "security-accepted"))
		if err != nil {
			return authorityv1.WorkItem{}, wrapReconciliation(err)
		}
		item = response.Work
	}
	if len(item.Reviews) != 2 || !reviewMatches(item.Reviews[0], "qa", grant.AcceptedCandidateHead) ||
		!reviewMatches(item.Reviews[1], "security-reviewer", grant.AcceptedCandidateHead) {
		return authorityv1.WorkItem{}, ErrCanonicalPreimage
	}
	if item.RunDisposition == nil {
		response, err := deps.client.RecordRunDisposition(ctx, orchestrator, authorityv1.RunDispositionRequest{
			BeadID: item.BeadID, ExpectedVersion: item.Version, ExpectedIntegrity: item.Integrity, HeadSHA: grant.AcceptedCandidateHead,
			Status: authorityv1.RunCompleted, EvidenceRefs: []string{"ci-33226093357", "protected-main-33246178629"},
			IdempotencyKey: operationKey(grant, "run"), TraceRef: operationTrace("run"),
		})
		if err != nil {
			return authorityv1.WorkItem{}, wrapReconciliation(err)
		}
		item = response.Work
	}
	if item.RunDisposition.Status != authorityv1.RunCompleted || item.RunDisposition.HeadSHA != grant.AcceptedCandidateHead {
		return authorityv1.WorkItem{}, ErrCanonicalPreimage
	}
	if item.Reconciliation == nil {
		response, err := deps.client.RecordReconciliation(ctx, orchestrator, authorityv1.ReconciliationRequest{
			BeadID: item.BeadID, ExpectedVersion: item.Version, ExpectedIntegrity: item.Integrity, HeadSHA: grant.AcceptedCandidateHead,
			MergedSHA: grant.MergedCommit, MergedTree: grant.MergedTree, PullRequestID: "pr-10", ProtectedMainRunID: "run-33246178629",
			EvidenceRefs:   []string{"merge-f607369", "tree-1025c596", "ci-33246178629"},
			IdempotencyKey: operationKey(grant, "reconcile"), TraceRef: operationTrace("reconcile"),
		})
		if err != nil {
			return authorityv1.WorkItem{}, wrapReconciliation(err)
		}
		item = response.Work
	}
	if !reconciliationMatches(item.Reconciliation, grant) {
		return authorityv1.WorkItem{}, ErrCanonicalPreimage
	}
	if item.Terminal == nil {
		response, err := deps.client.CloseWork(ctx, orchestrator, authorityv1.TerminalTransitionRequest{
			BeadID: item.BeadID, ExpectedVersion: item.Version, ExpectedIntegrity: item.Integrity, HeadSHA: grant.AcceptedCandidateHead,
			EvidenceRefs:   []string{"reconciliation-f607369", "terminal-authorized"},
			IdempotencyKey: operationKey(grant, "terminal"), TraceRef: operationTrace("terminal"),
		})
		if err != nil {
			return authorityv1.WorkItem{}, wrapReconciliation(err)
		}
		item = response.Work
	}
	return item, nil
}

func reconcileLease(ctx context.Context, deps dependencies, engineer authorityv1.Principal, request gateway.ClaimReconciliationRequest, item authorityv1.WorkItem, grant doctrine.W001TerminalReconciliationGrant) (authorityv1.CapabilityLease, error) {
	claim, err := deps.client.ReconcileClaimedWork(ctx, engineer, request)
	if err == nil && claim.Lease.AttemptID == grant.AttemptID && claim.Lease.ClaimVersion == item.Version && claim.Lease.BaseSHA == request.BaseSHA {
		return claim.Lease, nil
	}
	saga, found, lookupErr := deps.sagas.Lookup(ctx, engineer.TenantID, engineer.ProjectID, request.IdempotencyKey)
	if lookupErr != nil || !found || !completedSagaMatches(saga, engineer, request, item) ||
		saga.Lease.AttemptID != grant.AttemptID || saga.Lease.CanonicalClaimAttemptID != grant.CanonicalClaimAttemptID ||
		saga.Lease.ClaimVersion != item.Version || saga.Lease.BaseSHA != request.BaseSHA || saga.Lease.State != authorityv1.LeaseReleased {
		return authorityv1.CapabilityLease{}, wrapReconciliation(err)
	}
	return saga.Lease, nil
}

func completedSagaMatches(saga gateway.ClaimSaga, principal authorityv1.Principal, request gateway.ClaimReconciliationRequest, item authorityv1.WorkItem) bool {
	intent := saga.Intent
	lease := saga.Lease
	return saga.Phase == gateway.ClaimPhaseComplete && lowerHexDigest.MatchString(saga.RequestDigest) && intent.RequestDigest == saga.RequestDigest &&
		intent.TenantID == principal.TenantID && intent.ProjectID == principal.ProjectID && intent.BeadID == request.BeadID &&
		intent.AttemptID == request.AttemptID && intent.IdempotencyKey == request.IdempotencyKey && intent.BaseSHA == request.BaseSHA &&
		intent.Capability == request.Capability && intent.TraceRef == request.TraceRef && reflect.DeepEqual(intent.ExclusivePaths, request.ExclusivePaths) &&
		reflect.DeepEqual(intent.Labels, []authorityv1.Label{authorityv1.LabelPublicAccepted}) && reflect.DeepEqual(saga.Work, item) &&
		lease.TenantID == principal.TenantID && lease.ProjectID == principal.ProjectID && lease.BeadID == request.BeadID &&
		lease.IdempotencyKey == request.IdempotencyKey && lease.Capability == request.Capability &&
		reflect.DeepEqual(lease.ExclusivePaths, request.ExclusivePaths) && reflect.DeepEqual(lease.Labels, intent.Labels)
}

func principal(authorization doctrine.W001TerminalReconciliationExecutionAuthorization, profile string, capabilities ...authorityv1.Capability) authorityv1.Principal {
	return authorityv1.Principal{TenantID: authorization.TenantID, ProjectID: authorization.ProjectID, PrincipalID: profile, ProfileID: profile,
		Capabilities: capabilities, Labels: []authorityv1.Label{authorityv1.LabelPublicAccepted}}
}

func reviewRequest(item authorityv1.WorkItem, grant doctrine.W001TerminalReconciliationGrant, reviewer, evidence string) authorityv1.ReviewVerdictRequest {
	return authorityv1.ReviewVerdictRequest{
		BeadID: item.BeadID, ExpectedVersion: item.Version, ExpectedIntegrity: item.Integrity, HeadSHA: grant.AcceptedCandidateHead,
		Verdict: authorityv1.ReviewAccepted, EvidenceRefs: []string{evidence, "candidate-56c2a8d"},
		IdempotencyKey: operationKey(grant, reviewer), TraceRef: operationTrace(reviewer),
	}
}

func operationKey(grant doctrine.W001TerminalReconciliationGrant, operation string) string {
	return grant.AttemptID + "-" + operation
}

func operationTrace(operation string) string { return "trace-w001-terminal-" + operation }

func fenceFromLease(lease authorityv1.CapabilityLease) authorityv1.FencingTuple {
	return authorityv1.FencingTuple{
		TenantID: lease.TenantID, ProjectID: lease.ProjectID, BeadID: lease.BeadID, AttemptID: lease.AttemptID,
		CanonicalClaimAttemptID: lease.CanonicalClaimAttemptID, IdempotencyKey: lease.IdempotencyKey, LeaseID: lease.LeaseID,
		FenceGeneration: lease.FenceGeneration, LeaseEpoch: lease.LeaseEpoch, ClaimVersion: lease.ClaimVersion,
		BaseSHA: lease.BaseSHA, Capability: lease.Capability, ExclusivePaths: append([]string(nil), lease.ExclusivePaths...),
		Labels: append([]authorityv1.Label(nil), lease.Labels...),
	}
}

func matchesInitialPreimage(item authorityv1.WorkItem, grant doctrine.W001TerminalReconciliationGrant) bool {
	return item.BeadID == grant.Bead && item.NativeStatus == grant.ExpectedNativeStatus && item.LifecycleState == authorityv1.LifecycleInProgress &&
		item.Assignee == grant.Assignee && item.ClaimAttemptID == grant.CanonicalClaimAttemptID && item.Version == grant.ExpectedVersion &&
		item.Handoff == nil && len(item.Reviews) == 0 && item.RunDisposition == nil && item.Reconciliation == nil && item.Terminal == nil
}

func matchesHandoff(item authorityv1.WorkItem, grant doctrine.W001TerminalReconciliationGrant) bool {
	return item.BeadID == grant.Bead && item.NativeStatus == "in_progress" && item.LifecycleState == authorityv1.LifecycleInReview &&
		item.Handoff != nil && item.Handoff.AttemptID == grant.AttemptID && item.Handoff.CanonicalClaimAttemptID == grant.CanonicalClaimAttemptID &&
		item.Handoff.HeadSHA == grant.AcceptedCandidateHead && item.Handoff.NextProfileID == "qa" &&
		item.Handoff.IdempotencyKey == operationKey(grant, "handoff")
}

func reviewMatches(review authorityv1.ReviewRecord, reviewer, head string) bool {
	return review.ReviewerProfileID == reviewer && review.Verdict == authorityv1.ReviewAccepted && review.HeadSHA == head && review.Failure == nil
}

func reconciliationMatches(record *authorityv1.ReconciliationRecord, grant doctrine.W001TerminalReconciliationGrant) bool {
	return record != nil && record.PrincipalProfileID == "delivery-orchestrator" && record.HeadSHA == grant.AcceptedCandidateHead &&
		record.MergedSHA == grant.MergedCommit && record.MergedTree == grant.MergedTree && record.PullRequestID == "pr-10" &&
		record.ProtectedMainRunID == "run-33246178629" && record.IdempotencyKey == operationKey(grant, "reconcile")
}

func matchesTerminalPostimage(item authorityv1.WorkItem, grant doctrine.W001TerminalReconciliationGrant) bool {
	return item.BeadID == grant.Bead && item.NativeStatus == "closed" && item.LifecycleState == authorityv1.LifecycleDone &&
		matchesHandoffRecord(item.Handoff, grant) && len(item.Reviews) == 2 && reviewMatches(item.Reviews[0], "qa", grant.AcceptedCandidateHead) &&
		reviewMatches(item.Reviews[1], "security-reviewer", grant.AcceptedCandidateHead) && item.RunDisposition != nil &&
		item.RunDisposition.Status == authorityv1.RunCompleted && item.RunDisposition.HeadSHA == grant.AcceptedCandidateHead &&
		reconciliationMatches(item.Reconciliation, grant) && item.Terminal != nil && item.Terminal.PrincipalProfileID == "delivery-orchestrator" &&
		item.Terminal.HeadSHA == grant.AcceptedCandidateHead && item.Terminal.IdempotencyKey == operationKey(grant, "terminal")
}

func matchesHandoffRecord(record *authorityv1.HandoffRecord, grant doctrine.W001TerminalReconciliationGrant) bool {
	return record != nil && record.AttemptID == grant.AttemptID && record.CanonicalClaimAttemptID == grant.CanonicalClaimAttemptID &&
		record.HeadSHA == grant.AcceptedCandidateHead && record.NextProfileID == "qa" && record.IdempotencyKey == operationKey(grant, "handoff")
}

func receiptFor(grant doctrine.W001TerminalReconciliationGrant, item authorityv1.WorkItem, active bool, mode, result string) Receipt {
	receipt := Receipt{SchemaVersion: 1, GrantID: grant.ID, BeadID: item.BeadID, Mode: mode, Result: result,
		NativeStatus: item.NativeStatus, LifecycleState: item.LifecycleState, Version: item.Version, LiveLeasePresent: active,
		ReviewCount: len(item.Reviews), TerminalRecorded: item.Terminal != nil}
	if item.Handoff != nil {
		receipt.HandoffHead = item.Handoff.HeadSHA
	}
	if item.RunDisposition != nil {
		receipt.RunStatus = item.RunDisposition.Status
	}
	if item.Reconciliation != nil {
		receipt.MergedSHA, receipt.MergedTree = item.Reconciliation.MergedSHA, item.Reconciliation.MergedTree
		receipt.PullRequestID, receipt.ProtectedMainRunID = item.Reconciliation.PullRequestID, item.Reconciliation.ProtectedMainRunID
	}
	return receipt
}

func wrapReconciliation(err error) error {
	if err == nil {
		return ErrReconciliation
	}
	return fmt.Errorf("%w: gateway operation denied", ErrReconciliation)
}

func verifyRepository(repo string, authorization doctrine.W001TerminalReconciliationExecutionAuthorization) error {
	root, err := filepath.Abs(repo)
	if err != nil || root != filepath.Clean(repo) {
		return ErrExecutionBoundary
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil || resolved != root {
		return ErrExecutionBoundary
	}
	rootInfo, err := os.Lstat(root)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return ErrExecutionBoundary
	}
	info, err := os.Lstat(filepath.Join(root, ".git"))
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return ErrExecutionBoundary
	}
	head, err := gitOutput(root, "rev-parse", "HEAD")
	if err != nil || strings.TrimSpace(head) != authorization.MergedCommit {
		return ErrExecutionBoundary
	}
	tree, err := gitOutput(root, "rev-parse", "HEAD^{tree}")
	if err != nil || strings.TrimSpace(tree) != authorization.MergedTree {
		return ErrExecutionBoundary
	}
	branch, err := gitOutput(root, "symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil || strings.TrimSpace(branch) != "main" {
		return ErrExecutionBoundary
	}
	status, err := gitOutput(root, "status", "--porcelain=v1", "--untracked-files=all")
	if err != nil || strings.TrimSpace(status) != "" {
		return ErrExecutionBoundary
	}
	tagTarget, err := gitOutput(root, "rev-parse", "refs/tags/"+authorization.ReviewTag+"^{}")
	if err != nil || strings.TrimSpace(tagTarget) != authorization.ReviewedFeatureCommit {
		return ErrExecutionBoundary
	}
	tagObject, err := gitOutput(root, "rev-parse", "refs/tags/"+authorization.ReviewTag+"^{tag}")
	if err != nil || strings.TrimSpace(tagObject) != authorization.ReviewTagObject {
		return ErrExecutionBoundary
	}
	tagTree, err := gitOutput(root, "rev-parse", "refs/tags/"+authorization.ReviewTag+"^{}^{tree}")
	if err != nil || strings.TrimSpace(tagTree) != authorization.MergedTree {
		return ErrExecutionBoundary
	}
	return nil
}

func gitOutput(root string, arguments ...string) (string, error) {
	command := exec.Command(terminalGitExecutable, append([]string{"-C", root}, arguments...)...)
	command.Env = []string{"HOME=/nonexistent", "XDG_CONFIG_HOME=/nonexistent", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "LC_ALL=C", "LANG=C", "TZ=UTC", "PATH=/usr/bin:/bin"}
	var stdout bytes.Buffer
	command.Stdout, command.Stderr = &stdout, io.Discard
	if err := command.Run(); err != nil {
		return "", ErrExecutionBoundary
	}
	return stdout.String(), nil
}

func readSecretFile(path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() <= 0 || info.Size() > 4096 {
		return "", ErrConfiguration
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", ErrConfiguration
	}
	value := strings.TrimSpace(string(data))
	for index := range data {
		data[index] = 0
	}
	if value == "" || strings.ContainsAny(value, "\r\n") {
		zeroString(&value)
		return "", ErrConfiguration
	}
	return value, nil
}

func zeroString(value *string) { *value = "" }

func digestRegularFile(path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return "", ErrExecutionBoundary
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", ErrExecutionBoundary
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}

type workspaceEntry struct {
	Path          string `json:"path"`
	Kind          string `json:"kind"`
	Device        uint64 `json:"device"`
	Inode         uint64 `json:"inode"`
	ContentSHA256 string `json:"contentSHA256,omitempty"`
}

func workspaceInstanceDigest(root, projectID string) (string, error) {
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil || !filepath.IsAbs(resolved) || resolved != filepath.Clean(root) {
		return "", ErrExecutionBoundary
	}
	beadsDir := filepath.Join(resolved, ".beads")
	for _, forbidden := range []string{"redirect", ".env", "config.local.yaml"} {
		if _, err := os.Lstat(filepath.Join(beadsDir, forbidden)); !errors.Is(err, os.ErrNotExist) {
			return "", ErrExecutionBoundary
		}
	}
	paths := []struct{ relative, kind string }{{".", "directory"}, {".beads", "directory"}, {".beads/metadata.json", "file"},
		{".beads/config.yaml", "file"}, {".beads/embeddeddolt", "directory"}, {".beads/embeddeddolt/M3", "directory"}}
	entries := make([]workspaceEntry, 0, len(paths))
	for _, required := range paths {
		info, err := os.Lstat(filepath.Join(resolved, filepath.FromSlash(required.relative)))
		if err != nil || info.Mode()&os.ModeSymlink != 0 || required.kind == "directory" && !info.IsDir() || required.kind == "file" && !info.Mode().IsRegular() {
			return "", ErrExecutionBoundary
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			return "", ErrExecutionBoundary
		}
		entry := workspaceEntry{Path: required.relative, Kind: required.kind, Device: uint64(stat.Dev), Inode: uint64(stat.Ino)}
		if required.kind == "file" {
			content, err := os.ReadFile(filepath.Join(resolved, filepath.FromSlash(required.relative)))
			if err != nil {
				return "", ErrExecutionBoundary
			}
			digest := sha256.Sum256(content)
			entry.ContentSHA256 = hex.EncodeToString(digest[:])
		}
		entries = append(entries, entry)
	}
	metadataRaw, err := os.ReadFile(filepath.Join(beadsDir, "metadata.json"))
	if err != nil || rejectDuplicateJSONKeys(metadataRaw) != nil {
		return "", ErrExecutionBoundary
	}
	var metadata struct {
		Database     string `json:"database"`
		Backend      string `json:"backend"`
		DoltMode     string `json:"dolt_mode"`
		DoltDatabase string `json:"dolt_database"`
		ProjectID    string `json:"project_id"`
	}
	decoder := json.NewDecoder(bytes.NewReader(metadataRaw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&metadata) != nil || metadata.Database != "dolt" || metadata.Backend != "dolt" || metadata.DoltMode != "embedded" ||
		metadata.DoltDatabase != "M3" || metadata.ProjectID != projectID {
		return "", ErrExecutionBoundary
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return "", ErrExecutionBoundary
	}
	payload := struct {
		SchemaVersion int              `json:"schemaVersion"`
		ProjectID     string           `json:"projectId"`
		Database      string           `json:"database"`
		Backend       string           `json:"backend"`
		DoltMode      string           `json:"doltMode"`
		DoltDatabase  string           `json:"doltDatabase"`
		Root          string           `json:"root"`
		Entries       []workspaceEntry `json:"entries"`
	}{3, metadata.ProjectID, metadata.Database, metadata.Backend, metadata.DoltMode, metadata.DoltDatabase, filepath.Clean(resolved), entries}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", ErrExecutionBoundary
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

func rejectDuplicateJSONKeys(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	var visit func() error
	visit = func() error {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delimiter, composite := token.(json.Delim)
		if !composite {
			return nil
		}
		switch delimiter {
		case '{':
			seen := make(map[string]bool)
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return err
				}
				key, ok := keyToken.(string)
				if !ok || seen[key] {
					return ErrExecutionBoundary
				}
				seen[key] = true
				if err := visit(); err != nil {
					return err
				}
			}
			_, err := decoder.Token()
			return err
		case '[':
			for decoder.More() {
				if err := visit(); err != nil {
					return err
				}
			}
			_, err := decoder.Token()
			return err
		default:
			return ErrExecutionBoundary
		}
	}
	if err := visit(); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return ErrExecutionBoundary
	}
	return nil
}
