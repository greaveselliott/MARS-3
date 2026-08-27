/*
FactoryDocSync:
docs:
- docs/features/F-002-work-authority.md
- docs/design-docs/ADR-001-git-beads-authority.md
- docs/code-documentation-map.md
*/

// Package beads maps the canonical Beads/Dolt work graph into bounded gateway
// projections. The injected mutator must provide one native, transaction-bound
// expected-version CAS; a sequence of ordinary CLI updates is never accepted.
package beads

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	authorityv1 "github.com/greaveselliott/MARS-3/api/authority/v1"
	"github.com/greaveselliott/MARS-3/internal/authority/gateway"
)

var (
	ErrProjectionInvalid = errors.New("canonical Beads projection invalid")
	ErrAtomicCASRequired = errors.New("native atomic Beads CAS required")
	ErrCASStale          = errors.New("canonical Beads CAS stale")
)

type Reader interface {
	ReadIssue(context.Context, string) ([]byte, error)
	ListIssueIDs(context.Context) ([]string, error)
}

// AtomicClaim binds every expected projection field. Implementations must
// validate and mutate these values in one native embedded-Dolt transaction.
type AtomicClaim struct {
	BeadID             string
	ExpectedVersion    authorityv1.WorkVersion
	ExpectedIntegrity  authorityv1.IntegrityDigests
	ExpectedDigest     string
	ExpectedStatus     string
	ExpectedAssignee   string
	ExpectedCreatedAt  string
	ExpectedUpdatedAt  string
	MetadataSHA256     string
	LabelsSHA256       string
	DependenciesSHA256 string
	AttemptID          string
	Assignee           string
	IdempotencyKey     string
	BaseCommit         string
	PostMetadata       []byte
}

type AtomicMutator interface {
	CompareAndSwapClaim(context.Context, AtomicClaim) ([]byte, error)
}

// AtomicLifecycleTransition binds one operation-specific metadata postimage to
// the same issue, label, dependency, and WorkVersion preimage used by reads.
type AtomicLifecycleTransition struct {
	BeadID             string
	ExpectedVersion    authorityv1.WorkVersion
	ExpectedIntegrity  authorityv1.IntegrityDigests
	ExpectedDigest     string
	ExpectedStatus     string
	ExpectedAssignee   string
	ExpectedCreatedAt  string
	ExpectedUpdatedAt  string
	MetadataSHA256     string
	LabelsSHA256       string
	DependenciesSHA256 string
	Transition         gateway.LifecycleMutation
	PostStatus         string
	RemoveLabel        string
	AddLabel           string
	PostMetadata       []byte
}

type AtomicLifecycleMutator interface {
	CompareAndSwapLifecycle(context.Context, AtomicLifecycleTransition) ([]byte, error)
}

type Store struct {
	tenantID       string
	projectID      string
	resourceLabels []authorityv1.Label
	reader         Reader
	mutator        AtomicMutator
}

func New(tenantID, projectID string, resourceLabels []authorityv1.Label, reader Reader, mutator AtomicMutator) (*Store, error) {
	labels := append([]authorityv1.Label(nil), resourceLabels...)
	sort.Slice(labels, func(i, j int) bool { return labels[i] < labels[j] })
	if tenantID == "" || projectID == "" || reader == nil || mutator == nil || len(labels) == 0 || hasDuplicateLabels(labels) {
		return nil, ErrProjectionInvalid
	}
	for _, label := range labels {
		if !knownLabel(label) {
			return nil, ErrProjectionInvalid
		}
	}
	return &Store{tenantID: tenantID, projectID: projectID, resourceLabels: labels, reader: reader, mutator: mutator}, nil
}

func (store *Store) Get(ctx context.Context, tenantID, projectID, beadID string) (authorityv1.WorkItem, error) {
	if tenantID != store.tenantID || projectID != store.projectID || beadID == "" {
		return authorityv1.WorkItem{}, gateway.ErrWorkNotFound
	}
	raw, err := store.reader.ReadIssue(ctx, beadID)
	if err != nil {
		return authorityv1.WorkItem{}, ErrProjectionInvalid
	}
	item, err := decodeIssueProjection(raw, tenantID, projectID, store.resourceLabels)
	if err != nil || item.BeadID != beadID {
		return authorityv1.WorkItem{}, ErrProjectionInvalid
	}
	return item, nil
}

func (store *Store) List(ctx context.Context, tenantID, projectID string) ([]authorityv1.WorkItem, error) {
	if tenantID != store.tenantID || projectID != store.projectID {
		return nil, ErrProjectionInvalid
	}
	ids, err := store.reader.ListIssueIDs(ctx)
	if err != nil || len(ids) > 100 || hasDuplicateStrings(ids) {
		return nil, ErrProjectionInvalid
	}
	sort.Strings(ids)
	items := make([]authorityv1.WorkItem, 0, len(ids))
	for _, id := range ids {
		item, err := store.Get(ctx, tenantID, projectID, id)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (store *Store) CompareAndSwapClaim(ctx context.Context, mutation gateway.ClaimMutation) (authorityv1.WorkItem, error) {
	if mutation.TenantID != store.tenantID || mutation.ProjectID != store.projectID || mutation.BeadID == "" || mutation.AttemptID == "" || mutation.Assignee == "" || mutation.IdempotencyKey == "" || mutation.BaseSHA == "" {
		return authorityv1.WorkItem{}, ErrProjectionInvalid
	}
	snapshot, err := store.readSnapshot(ctx, mutation.TenantID, mutation.ProjectID, mutation.BeadID)
	if err != nil {
		return authorityv1.WorkItem{}, err
	}
	pre := snapshot.Item
	if pre.LifecycleState != authorityv1.LifecycleBacklog || pre.Version != mutation.ExpectedVersion || pre.Integrity != mutation.ExpectedIntegrity {
		return authorityv1.WorkItem{}, gateway.ErrStaleWorkVersion
	}
	postMetadata, err := claimPostMetadata(snapshot.MetadataRaw, mutation.AttemptID, mutation.IdempotencyKey, mutation.BaseSHA)
	if err != nil {
		return authorityv1.WorkItem{}, ErrProjectionInvalid
	}
	raw, err := store.mutator.CompareAndSwapClaim(ctx, AtomicClaim{
		BeadID: mutation.BeadID, ExpectedVersion: mutation.ExpectedVersion,
		ExpectedIntegrity: mutation.ExpectedIntegrity, ExpectedDigest: projectionDigest(pre),
		ExpectedStatus: snapshot.NativeStatus, ExpectedAssignee: snapshot.NativeAssignee,
		ExpectedCreatedAt: snapshot.CreatedAt, ExpectedUpdatedAt: snapshot.UpdatedAt,
		MetadataSHA256: snapshot.MetadataSHA256, LabelsSHA256: snapshot.LabelsSHA256,
		DependenciesSHA256: snapshot.DependenciesSHA256,
		AttemptID:          mutation.AttemptID, Assignee: mutation.Assignee, IdempotencyKey: mutation.IdempotencyKey,
		BaseCommit: mutation.BaseSHA, PostMetadata: postMetadata,
	})
	if errors.Is(err, ErrCASStale) {
		return authorityv1.WorkItem{}, gateway.ErrStaleWorkVersion
	}
	if err != nil {
		return authorityv1.WorkItem{}, ErrAtomicCASRequired
	}
	post, err := decodeIssueProjection(raw, mutation.TenantID, mutation.ProjectID, store.resourceLabels)
	if err != nil || post.BeadID != mutation.BeadID {
		return authorityv1.WorkItem{}, ErrProjectionInvalid
	}
	if post.NativeStatus != "in_progress" || post.LifecycleState != authorityv1.LifecycleInProgress ||
		post.Assignee != mutation.Assignee || post.ClaimAttemptID != mutation.AttemptID ||
		post.Version.AuthorityGeneration != pre.Version.AuthorityGeneration ||
		post.Version.IssueIncarnation != pre.Version.IssueIncarnation ||
		post.Version.IssueMutationSequence != pre.Version.IssueMutationSequence+1 ||
		post.Version.DependencyGraphRevision != pre.Version.DependencyGraphRevision {
		return authorityv1.WorkItem{}, ErrProjectionInvalid
	}
	return post, nil
}

// CompareAndSwapLifecycle performs one operation-specific lifecycle mutation
// through the reviewed native Beads transaction. It never synthesizes a
// sequence of ordinary update commands.
func (store *Store) CompareAndSwapLifecycle(ctx context.Context, mutation gateway.LifecycleMutation) (authorityv1.WorkItem, error) {
	mutator, ok := store.mutator.(AtomicLifecycleMutator)
	if !ok || mutation.TenantID != store.tenantID || mutation.ProjectID != store.projectID || mutation.BeadID == "" {
		return authorityv1.WorkItem{}, ErrAtomicCASRequired
	}
	snapshot, err := store.readSnapshot(ctx, mutation.TenantID, mutation.ProjectID, mutation.BeadID)
	if err != nil {
		return authorityv1.WorkItem{}, err
	}
	pre := snapshot.Item
	if pre.Version != mutation.ExpectedVersion || pre.Integrity != mutation.ExpectedIntegrity {
		return authorityv1.WorkItem{}, gateway.ErrStaleWorkVersion
	}
	postMetadata, postStatus, removeLabel, addLabel, err := lifecyclePostMetadata(snapshot.MetadataRaw, mutation)
	if err != nil {
		return authorityv1.WorkItem{}, ErrProjectionInvalid
	}
	raw, err := mutator.CompareAndSwapLifecycle(ctx, AtomicLifecycleTransition{
		BeadID: mutation.BeadID, ExpectedVersion: mutation.ExpectedVersion, ExpectedIntegrity: mutation.ExpectedIntegrity,
		ExpectedDigest: projectionDigest(pre), ExpectedStatus: snapshot.NativeStatus, ExpectedAssignee: snapshot.NativeAssignee,
		ExpectedCreatedAt: snapshot.CreatedAt, ExpectedUpdatedAt: snapshot.UpdatedAt, MetadataSHA256: snapshot.MetadataSHA256,
		LabelsSHA256: snapshot.LabelsSHA256, DependenciesSHA256: snapshot.DependenciesSHA256, Transition: mutation,
		PostStatus: postStatus, RemoveLabel: removeLabel, AddLabel: addLabel, PostMetadata: postMetadata,
	})
	if errors.Is(err, ErrCASStale) {
		return authorityv1.WorkItem{}, gateway.ErrStaleWorkVersion
	}
	if err != nil {
		return authorityv1.WorkItem{}, ErrAtomicCASRequired
	}
	post, err := decodeIssueProjection(raw, mutation.TenantID, mutation.ProjectID, store.resourceLabels)
	if err != nil || !validLifecyclePostimage(pre, post, mutation) {
		return authorityv1.WorkItem{}, ErrProjectionInvalid
	}
	return post, nil
}

func (store *Store) readSnapshot(ctx context.Context, tenantID, projectID, beadID string) (issueSnapshot, error) {
	if tenantID != store.tenantID || projectID != store.projectID || beadID == "" {
		return issueSnapshot{}, gateway.ErrWorkNotFound
	}
	raw, err := store.reader.ReadIssue(ctx, beadID)
	if err != nil {
		return issueSnapshot{}, ErrProjectionInvalid
	}
	snapshot, err := decodeIssueSnapshot(raw, tenantID, projectID, store.resourceLabels)
	if err != nil || snapshot.Item.BeadID != beadID {
		return issueSnapshot{}, ErrProjectionInvalid
	}
	return snapshot, nil
}

type issueMetadata struct {
	SchemaVersion               uint32                     `json:"schemaVersion"`
	DisplayID                   string                     `json:"displayId"`
	LifecycleState              authorityv1.LifecycleState `json:"lifecycleState"`
	GoalIDs                     []string                   `json:"goalIds"`
	ProductDecisionIDs          []string                   `json:"productDecisionIds"`
	FeatureID                   string                     `json:"featureId"`
	ScenarioIDs                 []string                   `json:"scenarioIds"`
	ExclusivePaths              []string                   `json:"exclusivePaths"`
	VerificationOrder           []string                   `json:"verificationOrder"`
	WorkVersion                 metadataWorkVersion        `json:"workVersion"`
	BootstrapClaim              *claimBinding              `json:"bootstrapClaim,omitempty"`
	WorkClaim                   *claimBinding              `json:"workClaim,omitempty"`
	Blocker                     string                     `json:"blocker,omitempty"`
	BlockedBy                   []string                   `json:"blockedBy,omitempty"`
	ReviewAccepted              bool                       `json:"reviewAccepted,omitempty"`
	RunDisposition              string                     `json:"runDisposition,omitempty"`
	Reconciled                  bool                       `json:"reconciled,omitempty"`
	Handoff                     *metadataHandoff           `json:"handoff,omitempty"`
	ReviewRecords               []metadataReview           `json:"reviewRecords,omitempty"`
	ReviewHistory               []metadataReviewCycle      `json:"reviewHistory,omitempty"`
	RunHistory                  []metadataRunDisposition   `json:"runHistory,omitempty"`
	RunDispositionRecord        *metadataRunDisposition    `json:"runDispositionRecord,omitempty"`
	ReconciliationRecord        *metadataReconciliation    `json:"reconciliationRecord,omitempty"`
	TerminalRecord              *metadataTerminal          `json:"terminalRecord,omitempty"`
	Risk                        string                     `json:"risk"`
	WorkType                    string                     `json:"workType"`
	Coordinator                 string                     `json:"coordinator"`
	FailureOwnership            string                     `json:"failureOwnership"`
	PublicDisclosure            bool                       `json:"publicDisclosure"`
	ContractPublicationRequired bool                       `json:"contractPublicationRequired,omitempty"`
}

type metadataHandoff struct {
	AttemptID               string   `json:"attemptId"`
	CanonicalClaimAttemptID string   `json:"canonicalClaimAttemptId"`
	FenceDigest             string   `json:"fenceDigest"`
	HeadSHA                 string   `json:"headSHA"`
	EvidenceRefs            []string `json:"evidenceRefs"`
	NextProfileID           string   `json:"nextProfileId"`
	IdempotencyKey          string   `json:"idempotencyKey"`
}

type metadataReview struct {
	ReviewerProfileID string                      `json:"reviewerProfileId"`
	Verdict           authorityv1.ReviewVerdict   `json:"verdict"`
	HeadSHA           string                      `json:"headSHA"`
	EvidenceRefs      []string                    `json:"evidenceRefs"`
	IdempotencyKey    string                      `json:"idempotencyKey"`
	Failure           *authorityv1.FailureContext `json:"failure,omitempty"`
}

type metadataReviewCycle struct {
	Handoff        metadataHandoff          `json:"handoff"`
	Reviews        []metadataReview         `json:"reviews"`
	RunHistory     []metadataRunDisposition `json:"runHistory,omitempty"`
	RunDisposition *metadataRunDisposition  `json:"runDisposition,omitempty"`
}

type metadataRunDisposition struct {
	PrincipalProfileID string                           `json:"principalProfileId"`
	Status             authorityv1.RunDispositionStatus `json:"status"`
	HeadSHA            string                           `json:"headSHA"`
	EvidenceRefs       []string                         `json:"evidenceRefs"`
	IdempotencyKey     string                           `json:"idempotencyKey"`
	Failure            *authorityv1.FailureContext      `json:"failure,omitempty"`
}

type metadataReconciliation struct {
	PrincipalProfileID string   `json:"principalProfileId"`
	HeadSHA            string   `json:"headSHA"`
	MergedSHA          string   `json:"mergedSHA"`
	MergedTree         string   `json:"mergedTree"`
	PullRequestID      string   `json:"pullRequestId"`
	ProtectedMainRunID string   `json:"protectedMainRunId"`
	EvidenceRefs       []string `json:"evidenceRefs"`
	IdempotencyKey     string   `json:"idempotencyKey"`
}

type metadataTerminal struct {
	PrincipalProfileID string   `json:"principalProfileId"`
	HeadSHA            string   `json:"headSHA"`
	EvidenceRefs       []string `json:"evidenceRefs"`
	IdempotencyKey     string   `json:"idempotencyKey"`
}

type metadataWorkVersion struct {
	AuthorityGeneration     string `json:"authorityGeneration"`
	IssueIncarnation        string `json:"issueIncarnation"`
	IssueMutationSequence   uint64 `json:"issueMutationSequence"`
	DependencyGraphRevision uint64 `json:"dependencyGraphRevision"`
}

type claimBinding struct {
	AttemptID      string `json:"attemptId"`
	IdempotencyKey string `json:"idempotencyKey"`
	BaseCommit     string `json:"baseCommit"`
	GrantID        string `json:"grantId,omitempty"`
}

type dependencyProjection struct {
	ID             string
	Status         string
	DependencyType string
	Metadata       issueMetadata
	MetadataSHA256 string
}

type authorityDependencyCondition struct {
	ID             string `json:"id"`
	DependencyType string `json:"dependency_type"`
	Status         string `json:"status"`
	MetadataSHA256 string `json:"metadata_sha256"`
}

type issueSnapshot struct {
	Item               authorityv1.WorkItem
	NativeStatus       string
	NativeAssignee     string
	CreatedAt          string
	UpdatedAt          string
	MetadataRaw        []byte
	MetadataSHA256     string
	LabelsSHA256       string
	DependenciesSHA256 string
}

func decodeIssueProjection(raw []byte, tenantID, projectID string, labels []authorityv1.Label) (authorityv1.WorkItem, error) {
	snapshot, err := decodeIssueSnapshot(raw, tenantID, projectID, labels)
	return snapshot.Item, err
}

func decodeIssueSnapshot(raw []byte, tenantID, projectID string, labels []authorityv1.Label) (issueSnapshot, error) {
	objects, err := decodeIssueArray(raw)
	if err != nil || len(objects) != 1 {
		return issueSnapshot{}, ErrProjectionInvalid
	}
	object := objects[0]
	if !onlyObjectKeys(object, "acceptance_criteria", "assignee", "close_reason", "closed_at", "comment_count", "created_at", "created_by", "dependencies", "dependency_count", "dependent_count", "description", "id", "issue_type", "labels", "metadata", "owner", "priority", "started_at", "status", "title", "updated_at") {
		return issueSnapshot{}, ErrProjectionInvalid
	}
	var id, status, assignee, createdAt, updatedAt string
	var nativeLabels []string
	var metadata issueMetadata
	var dependenciesRaw []json.RawMessage
	if !validRawClaimFields(object["metadata"]) || decodeField(object, "id", &id) != nil || decodeField(object, "status", &status) != nil || decodeOptionalString(object, "assignee", &assignee) != nil ||
		decodeField(object, "created_at", &createdAt) != nil || decodeField(object, "updated_at", &updatedAt) != nil ||
		decodeField(object, "labels", &nativeLabels) != nil || decodeStrictField(object, "metadata", &metadata) != nil ||
		decodeField(object, "dependencies", &dependenciesRaw) != nil {
		return issueSnapshot{}, ErrProjectionInvalid
	}
	createdTime, createdErr := time.Parse(time.RFC3339Nano, createdAt)
	updatedTime, updatedErr := time.Parse(time.RFC3339Nano, updatedAt)
	if createdErr != nil || updatedErr != nil || hasDuplicateStrings(nativeLabels) {
		return issueSnapshot{}, ErrProjectionInvalid
	}
	sort.Strings(nativeLabels)
	dependencies := make([]dependencyProjection, 0, len(dependenciesRaw))
	for _, dependencyRaw := range dependenciesRaw {
		dependency, err := decodeDependency(dependencyRaw)
		if err != nil {
			return issueSnapshot{}, ErrProjectionInvalid
		}
		dependencies = append(dependencies, dependency)
	}
	item, err := projectIssue(tenantID, projectID, id, status, assignee, metadata, dependencies, labels)
	if err != nil {
		return issueSnapshot{}, err
	}
	conditions := make([]authorityDependencyCondition, 0, len(dependencies))
	for _, dependency := range dependencies {
		conditions = append(conditions, authorityDependencyCondition{
			ID: dependency.ID, DependencyType: dependency.DependencyType,
			Status: dependency.Status, MetadataSHA256: dependency.MetadataSHA256,
		})
	}
	sort.Slice(conditions, func(i, j int) bool {
		if conditions[i].ID == conditions[j].ID {
			return conditions[i].DependencyType < conditions[j].DependencyType
		}
		return conditions[i].ID < conditions[j].ID
	})
	metadataRaw := append([]byte(nil), object["metadata"]...)
	return issueSnapshot{
		Item: item, NativeStatus: status, NativeAssignee: assignee,
		CreatedAt: createdTime.UTC().Format(time.RFC3339), UpdatedAt: updatedTime.UTC().Format(time.RFC3339),
		MetadataRaw: metadataRaw, MetadataSHA256: canonicalJSONDigest(metadataRaw),
		LabelsSHA256: digestValue(nativeLabels), DependenciesSHA256: digestValue(conditions),
	}, nil
}

func projectIssue(tenantID, projectID, id, status, assignee string, metadata issueMetadata, dependencies []dependencyProjection, labels []authorityv1.Label) (authorityv1.WorkItem, error) {
	if metadata.SchemaVersion != 1 || !metadata.PublicDisclosure || id == "" || metadata.DisplayID == "" || metadata.FeatureID == "" || len(metadata.GoalIDs) == 0 || len(metadata.ProductDecisionIDs) == 0 || len(metadata.ScenarioIDs) == 0 || len(metadata.ExclusivePaths) == 0 || len(metadata.VerificationOrder) == 0 || metadata.WorkVersion.AuthorityGeneration == "" || metadata.WorkVersion.IssueIncarnation == "" || metadata.WorkVersion.IssueMutationSequence == 0 || metadata.WorkVersion.DependencyGraphRevision == 0 || hasDuplicateStrings(metadata.GoalIDs) || hasDuplicateStrings(metadata.ProductDecisionIDs) || hasDuplicateStrings(metadata.ScenarioIDs) || hasDuplicateStrings(metadata.VerificationOrder) {
		return authorityv1.WorkItem{}, ErrProjectionInvalid
	}
	if !compatibleLifecycle(status, metadata.LifecycleState) {
		return authorityv1.WorkItem{}, ErrProjectionInvalid
	}
	if !validLifecycleRecords(metadata) {
		return authorityv1.WorkItem{}, ErrProjectionInvalid
	}
	paths := make([]string, len(metadata.ExclusivePaths))
	for index, value := range metadata.ExclusivePaths {
		path, ok := normalizeDeclaredPath(value)
		if !ok {
			return authorityv1.WorkItem{}, ErrProjectionInvalid
		}
		paths[index] = path
	}
	sort.Strings(paths)
	if hasDuplicateStrings(paths) {
		return authorityv1.WorkItem{}, ErrProjectionInvalid
	}
	claimAttemptID := ""
	if metadata.WorkClaim != nil && metadata.BootstrapClaim != nil {
		return authorityv1.WorkItem{}, ErrProjectionInvalid
	}
	binding := metadata.WorkClaim
	bindingValid := validWorkClaimBinding(metadata.WorkClaim)
	if binding == nil {
		binding = metadata.BootstrapClaim
		bindingValid = validBootstrapClaimBinding(metadata.BootstrapClaim)
	}
	if metadata.LifecycleState == authorityv1.LifecycleBacklog {
		if binding != nil {
			return authorityv1.WorkItem{}, ErrProjectionInvalid
		}
	} else if !bindingValid || assignee == "" {
		return authorityv1.WorkItem{}, ErrProjectionInvalid
	}
	if binding != nil {
		claimAttemptID = binding.AttemptID
	}
	blockers := append([]string(nil), metadata.BlockedBy...)
	if metadata.Blocker != "" {
		blockers = append(blockers, metadata.Blocker)
	}
	sort.Strings(blockers)
	projectedDependencies := make([]authorityv1.Dependency, 0, len(dependencies))
	for _, dependency := range dependencies {
		if dependency.DependencyType != "blocks" || !compatibleLifecycle(dependency.Status, dependency.Metadata.LifecycleState) {
			return authorityv1.WorkItem{}, ErrProjectionInvalid
		}
		projectedDependencies = append(projectedDependencies, authorityv1.Dependency{
			BeadID: dependency.ID, LifecycleState: dependency.Metadata.LifecycleState,
			ReviewAccepted: dependency.Metadata.ReviewAccepted,
			RunCompleted: dependency.Metadata.RunDisposition == "completed" ||
				dependency.Metadata.RunDispositionRecord != nil && dependency.Metadata.RunDispositionRecord.Status == authorityv1.RunCompleted,
			Reconciled: dependency.Metadata.Reconciled || dependency.Metadata.ReconciliationRecord != nil,
		})
	}
	sort.Slice(projectedDependencies, func(i, j int) bool { return projectedDependencies[i].BeadID < projectedDependencies[j].BeadID })
	goalIDs, decisions, scenarios, verification := sortedCopy(metadata.GoalIDs), sortedCopy(metadata.ProductDecisionIDs), sortedCopy(metadata.ScenarioIDs), append([]string(nil), metadata.VerificationOrder...)
	item := authorityv1.WorkItem{
		TenantID: tenantID, ProjectID: projectID, BeadID: id, DisplayID: metadata.DisplayID,
		NativeStatus: status, LifecycleState: metadata.LifecycleState, Assignee: assignee, ClaimAttemptID: claimAttemptID,
		GoalIDs: goalIDs, ProductDecisionIDs: decisions, FeatureID: metadata.FeatureID, ScenarioIDs: scenarios,
		ExclusivePaths: paths, VerificationOrder: verification, Blockers: blockers, Dependencies: projectedDependencies,
		Labels: append([]authorityv1.Label(nil), labels...), Version: authorityv1.WorkVersion{
			AuthorityGeneration: metadata.WorkVersion.AuthorityGeneration, IssueIncarnation: metadata.WorkVersion.IssueIncarnation,
			IssueMutationSequence: metadata.WorkVersion.IssueMutationSequence, DependencyGraphRevision: metadata.WorkVersion.DependencyGraphRevision,
		},
	}
	if metadata.Handoff != nil {
		item.Handoff = &authorityv1.HandoffRecord{
			AttemptID: metadata.Handoff.AttemptID, CanonicalClaimAttemptID: metadata.Handoff.CanonicalClaimAttemptID,
			FenceDigest: metadata.Handoff.FenceDigest, HeadSHA: metadata.Handoff.HeadSHA,
			EvidenceRefs: append([]string(nil), metadata.Handoff.EvidenceRefs...), NextProfileID: metadata.Handoff.NextProfileID,
			IdempotencyKey: metadata.Handoff.IdempotencyKey,
		}
	}
	item.Reviews = make([]authorityv1.ReviewRecord, 0, len(metadata.ReviewRecords))
	for _, review := range metadata.ReviewRecords {
		item.Reviews = append(item.Reviews, authorityv1.ReviewRecord{
			ReviewerProfileID: review.ReviewerProfileID, Verdict: review.Verdict, HeadSHA: review.HeadSHA,
			EvidenceRefs: append([]string(nil), review.EvidenceRefs...), IdempotencyKey: review.IdempotencyKey, Failure: cloneFailureContext(review.Failure),
		})
	}
	item.ReviewHistory = make([]authorityv1.ReviewCycle, 0, len(metadata.ReviewHistory))
	for _, cycle := range metadata.ReviewHistory {
		projected := authorityv1.ReviewCycle{
			Handoff: authorityv1.HandoffRecord{
				AttemptID: cycle.Handoff.AttemptID, CanonicalClaimAttemptID: cycle.Handoff.CanonicalClaimAttemptID,
				FenceDigest: cycle.Handoff.FenceDigest, HeadSHA: cycle.Handoff.HeadSHA,
				EvidenceRefs: append([]string(nil), cycle.Handoff.EvidenceRefs...), NextProfileID: cycle.Handoff.NextProfileID,
				IdempotencyKey: cycle.Handoff.IdempotencyKey,
			},
			Reviews: make([]authorityv1.ReviewRecord, 0, len(cycle.Reviews)),
		}
		for _, review := range cycle.Reviews {
			projected.Reviews = append(projected.Reviews, authorityv1.ReviewRecord{
				ReviewerProfileID: review.ReviewerProfileID, Verdict: review.Verdict, HeadSHA: review.HeadSHA,
				EvidenceRefs: append([]string(nil), review.EvidenceRefs...), IdempotencyKey: review.IdempotencyKey, Failure: cloneFailureContext(review.Failure),
			})
		}
		projected.RunHistory = projectRunHistory(cycle.RunHistory)
		projected.RunDisposition = projectRunDisposition(cycle.RunDisposition)
		item.ReviewHistory = append(item.ReviewHistory, projected)
	}
	item.RunHistory = projectRunHistory(metadata.RunHistory)
	if metadata.RunDispositionRecord != nil {
		item.RunDisposition = projectRunDisposition(metadata.RunDispositionRecord)
	}
	if metadata.ReconciliationRecord != nil {
		item.Reconciliation = &authorityv1.ReconciliationRecord{
			PrincipalProfileID: metadata.ReconciliationRecord.PrincipalProfileID, HeadSHA: metadata.ReconciliationRecord.HeadSHA,
			MergedSHA: metadata.ReconciliationRecord.MergedSHA, MergedTree: metadata.ReconciliationRecord.MergedTree,
			PullRequestID: metadata.ReconciliationRecord.PullRequestID, ProtectedMainRunID: metadata.ReconciliationRecord.ProtectedMainRunID,
			EvidenceRefs: append([]string(nil), metadata.ReconciliationRecord.EvidenceRefs...), IdempotencyKey: metadata.ReconciliationRecord.IdempotencyKey,
		}
	}
	if metadata.TerminalRecord != nil {
		item.Terminal = &authorityv1.TerminalRecord{
			PrincipalProfileID: metadata.TerminalRecord.PrincipalProfileID,
			HeadSHA:            metadata.TerminalRecord.HeadSHA, EvidenceRefs: append([]string(nil), metadata.TerminalRecord.EvidenceRefs...),
			IdempotencyKey: metadata.TerminalRecord.IdempotencyKey,
		}
	}
	item.Integrity = authorityv1.IntegrityDigests{
		Lineage: digestValue(struct {
			BeadID, DisplayID, Assignee, FeatureID                      string
			GoalIDs, ProductDecisionIDs, ScenarioIDs, VerificationOrder []string
			Handoff                                                     *authorityv1.HandoffRecord
			Reviews                                                     []authorityv1.ReviewRecord
			ReviewHistory                                               []authorityv1.ReviewCycle
			RunHistory                                                  []authorityv1.RunDispositionRecord
			RunDisposition                                              *authorityv1.RunDispositionRecord
			Reconciliation                                              *authorityv1.ReconciliationRecord
			Terminal                                                    *authorityv1.TerminalRecord
		}{id, metadata.DisplayID, assignee, metadata.FeatureID, goalIDs, decisions, scenarios, verification,
			item.Handoff, item.Reviews, item.ReviewHistory, item.RunHistory, item.RunDisposition, item.Reconciliation, item.Terminal}),
		DependencyOutcomes: digestValue(projectedDependencies), Blockers: digestValue(blockers), ExclusivePaths: digestValue(paths),
	}
	return item, nil
}

func decodeIssueArray(raw []byte) ([]map[string]json.RawMessage, error) {
	if rejectDuplicateJSONKeys(raw) != nil {
		return nil, ErrProjectionInvalid
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	var objects []map[string]json.RawMessage
	if err := decoder.Decode(&objects); err != nil {
		return nil, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, ErrProjectionInvalid
	}
	return objects, nil
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
					return ErrProjectionInvalid
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
			return ErrProjectionInvalid
		}
	}
	if err := visit(); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return ErrProjectionInvalid
	}
	return nil
}

func decodeDependency(raw json.RawMessage) (dependencyProjection, error) {
	var object map[string]json.RawMessage
	if json.Unmarshal(raw, &object) != nil || !onlyObjectKeys(object,
		"acceptance_criteria", "assignee", "close_reason", "closed_at", "created_at", "created_by",
		"dependency_type", "description", "id", "issue_type", "labels", "metadata", "owner", "priority",
		"started_at", "status", "title", "updated_at",
	) {
		return dependencyProjection{}, ErrProjectionInvalid
	}
	var result dependencyProjection
	if decodeField(object, "id", &result.ID) != nil || decodeField(object, "status", &result.Status) != nil || decodeField(object, "dependency_type", &result.DependencyType) != nil || decodeDependencyMetadata(object["metadata"], &result.Metadata) != nil {
		return dependencyProjection{}, ErrProjectionInvalid
	}
	result.MetadataSHA256 = canonicalJSONDigest(object["metadata"])
	if result.MetadataSHA256 == "" {
		return dependencyProjection{}, ErrProjectionInvalid
	}
	return result, nil
}

func decodeDependencyMetadata(raw json.RawMessage, target *issueMetadata) error {
	var object map[string]json.RawMessage
	if json.Unmarshal(raw, &object) != nil {
		return ErrProjectionInvalid
	}
	for key, value := range object {
		switch key {
		case "lifecycleState":
			if json.Unmarshal(value, &target.LifecycleState) != nil {
				return ErrProjectionInvalid
			}
		case "reviewAccepted":
			if json.Unmarshal(value, &target.ReviewAccepted) != nil {
				return ErrProjectionInvalid
			}
		case "runDisposition":
			if json.Unmarshal(value, &target.RunDisposition) != nil {
				return ErrProjectionInvalid
			}
		case "reconciled":
			if json.Unmarshal(value, &target.Reconciled) != nil {
				return ErrProjectionInvalid
			}
		}
	}
	return nil
}

func decodeField(object map[string]json.RawMessage, key string, target any) error {
	value, found := object[key]
	if !found || json.Unmarshal(value, target) != nil {
		return ErrProjectionInvalid
	}
	return nil
}

func decodeOptionalString(object map[string]json.RawMessage, key string, target *string) error {
	value, found := object[key]
	if !found || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
		*target = ""
		return nil
	}
	if json.Unmarshal(value, target) != nil {
		return ErrProjectionInvalid
	}
	return nil
}

func decodeStrictField(object map[string]json.RawMessage, key string, target any) error {
	value, found := object[key]
	if !found {
		return ErrProjectionInvalid
	}
	decoder := json.NewDecoder(bytes.NewReader(value))
	decoder.DisallowUnknownFields()
	if decoder.Decode(target) != nil {
		return ErrProjectionInvalid
	}
	return nil
}

func onlyObjectKeys(object map[string]json.RawMessage, allowed ...string) bool {
	set := make(map[string]bool, len(allowed))
	for _, key := range allowed {
		set[key] = true
	}
	for key := range object {
		if !set[key] {
			return false
		}
	}
	return true
}

func normalizeDeclaredPath(value string) (string, bool) {
	if strings.HasSuffix(value, "/**") {
		value = strings.TrimSuffix(value, "**")
	}
	if value == "" || strings.HasPrefix(value, "/") || strings.Contains(value, "\\") || strings.ContainsAny(value, "*?[") || strings.Contains(value, "..") {
		return "", false
	}
	return value, true
}

func compatibleLifecycle(native string, lifecycle authorityv1.LifecycleState) bool {
	switch lifecycle {
	case authorityv1.LifecycleBacklog:
		return native == "open"
	case authorityv1.LifecycleInProgress, authorityv1.LifecycleInReview:
		return native == "in_progress"
	case authorityv1.LifecycleDone, authorityv1.LifecycleSuperseded:
		return native == "closed"
	default:
		return false
	}
}

func validClaimBinding(binding *claimBinding) bool {
	return validWorkClaimBinding(binding) || validBootstrapClaimBinding(binding)
}

func validWorkClaimBinding(binding *claimBinding) bool {
	return binding != nil && safeToken(binding.AttemptID) && safeToken(binding.IdempotencyKey) && commitID(binding.BaseCommit) && binding.GrantID == ""
}

func validBootstrapClaimBinding(binding *claimBinding) bool {
	return binding != nil && safeToken(binding.AttemptID) && safeToken(binding.IdempotencyKey) && commitID(binding.BaseCommit) && safeToken(binding.GrantID)
}

func validRawClaimFields(raw json.RawMessage) bool {
	if len(raw) == 0 || rejectDuplicateJSONKeys(raw) != nil {
		return false
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(raw, &object) != nil {
		return false
	}
	work, workPresent := object["workClaim"]
	bootstrap, bootstrapPresent := object["bootstrapClaim"]
	if workPresent && bootstrapPresent {
		return false
	}
	if workPresent {
		return validRawClaimObject(work, false)
	}
	if bootstrapPresent {
		return validRawClaimObject(bootstrap, true)
	}
	return true
}

func validRawClaimObject(raw json.RawMessage, bootstrap bool) bool {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return false
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(raw, &object) != nil {
		return false
	}
	expected := []string{"attemptId", "idempotencyKey", "baseCommit"}
	if bootstrap {
		expected = append(expected, "grantId")
	}
	if !onlyObjectKeys(object, expected...) || len(object) != len(expected) {
		return false
	}
	var binding claimBinding
	if decodeStrictField(map[string]json.RawMessage{"binding": raw}, "binding", &binding) != nil {
		return false
	}
	if bootstrap {
		return validBootstrapClaimBinding(&binding)
	}
	return validWorkClaimBinding(&binding)
}

func cloneFailureContext(value *authorityv1.FailureContext) *authorityv1.FailureContext {
	if value == nil {
		return nil
	}
	clone := *value
	clone.BlockedBy = append([]string(nil), value.BlockedBy...)
	return &clone
}

func projectRunDisposition(value *metadataRunDisposition) *authorityv1.RunDispositionRecord {
	if value == nil {
		return nil
	}
	return &authorityv1.RunDispositionRecord{
		PrincipalProfileID: value.PrincipalProfileID, Status: value.Status, HeadSHA: value.HeadSHA,
		EvidenceRefs: append([]string(nil), value.EvidenceRefs...), IdempotencyKey: value.IdempotencyKey,
		Failure: cloneFailureContext(value.Failure),
	}
}

func projectRunHistory(values []metadataRunDisposition) []authorityv1.RunDispositionRecord {
	result := make([]authorityv1.RunDispositionRecord, 0, len(values))
	for index := range values {
		result = append(result, *projectRunDisposition(&values[index]))
	}
	return result
}

func validLifecycleRecords(metadata issueMetadata) bool {
	if metadata.WorkClaim != nil && metadata.BootstrapClaim != nil {
		return false
	}
	binding := metadata.WorkClaim
	bindingValid := validWorkClaimBinding(metadata.WorkClaim)
	if binding == nil {
		binding = metadata.BootstrapClaim
		bindingValid = validBootstrapClaimBinding(metadata.BootstrapClaim)
	}
	if metadata.LifecycleState == authorityv1.LifecycleBacklog {
		if binding != nil {
			return false
		}
	} else if !bindingValid {
		return false
	}
	hasDetailed := metadata.Handoff != nil || len(metadata.ReviewRecords) > 0 || len(metadata.ReviewHistory) > 0 || len(metadata.RunHistory) > 0 || metadata.RunDispositionRecord != nil || metadata.ReconciliationRecord != nil || metadata.TerminalRecord != nil
	if !hasDetailed {
		return metadata.LifecycleState != authorityv1.LifecycleDone
	}
	if binding == nil {
		return false
	}
	if metadata.Handoff == nil || !validMetadataHandoff(*metadata.Handoff) ||
		metadata.Handoff.CanonicalClaimAttemptID != binding.AttemptID ||
		!safeToken(metadata.Handoff.NextProfileID) || !safeToken(metadata.Handoff.IdempotencyKey) || !validEvidenceRefs(metadata.Handoff.EvidenceRefs) {
		return false
	}
	if metadata.LifecycleState != authorityv1.LifecycleInReview && metadata.LifecycleState != authorityv1.LifecycleInProgress && metadata.LifecycleState != authorityv1.LifecycleDone {
		return false
	}
	reviewerCount := len(metadata.VerificationOrder) - 1
	if reviewerCount < 1 || metadata.Coordinator != metadata.VerificationOrder[len(metadata.VerificationOrder)-1] ||
		metadata.Handoff.NextProfileID != metadata.VerificationOrder[0] || len(metadata.ReviewRecords) > reviewerCount {
		return false
	}
	idempotencyKeys := map[string]bool{}
	if !addLifecycleKey(idempotencyKeys, metadata.Handoff.IdempotencyKey) {
		return false
	}
	for _, cycle := range metadata.ReviewHistory {
		if !validArchivedReviewCycle(cycle, metadata.VerificationOrder, reviewerCount) || cycle.Handoff.CanonicalClaimAttemptID != binding.AttemptID ||
			!addLifecycleKey(idempotencyKeys, cycle.Handoff.IdempotencyKey) {
			return false
		}
		for _, review := range cycle.Reviews {
			if !addLifecycleKey(idempotencyKeys, review.IdempotencyKey) {
				return false
			}
		}
		for _, run := range cycle.RunHistory {
			if !addLifecycleKey(idempotencyKeys, run.IdempotencyKey) {
				return false
			}
		}
		if cycle.RunDisposition != nil && !addLifecycleKey(idempotencyKeys, cycle.RunDisposition.IdempotencyKey) {
			return false
		}
	}
	nonAcceptedReview := false
	for index, review := range metadata.ReviewRecords {
		if review.ReviewerProfileID != metadata.VerificationOrder[index] || review.HeadSHA != metadata.Handoff.HeadSHA ||
			!safeToken(review.IdempotencyKey) || !validEvidenceRefs(review.EvidenceRefs) || !validMetadataReviewFailure(review) {
			return false
		}
		if !addLifecycleKey(idempotencyKeys, review.IdempotencyKey) {
			return false
		}
		if index < len(metadata.ReviewRecords)-1 && review.Verdict != authorityv1.ReviewAccepted {
			return false
		}
		nonAcceptedReview = nonAcceptedReview || review.Verdict != authorityv1.ReviewAccepted
	}
	if nonAcceptedReview && (metadata.LifecycleState != authorityv1.LifecycleInProgress || metadata.ReviewRecords[len(metadata.ReviewRecords)-1].Verdict == authorityv1.ReviewAccepted) {
		return false
	}
	for index := range metadata.RunHistory {
		if !validMetadataRun(&metadata.RunHistory[index], metadata.Handoff.HeadSHA, metadata.Coordinator) ||
			metadata.RunHistory[index].Status == authorityv1.RunCompleted || !addLifecycleKey(idempotencyKeys, metadata.RunHistory[index].IdempotencyKey) {
			return false
		}
	}
	if metadata.RunDispositionRecord != nil {
		run := metadata.RunDispositionRecord
		if !validMetadataRun(run, metadata.Handoff.HeadSHA, metadata.Coordinator) {
			return false
		}
		if !addLifecycleKey(idempotencyKeys, run.IdempotencyKey) {
			return false
		}
		if run.Status != authorityv1.RunCompleted && run.Status != authorityv1.RunInReview && metadata.LifecycleState != authorityv1.LifecycleInProgress {
			return false
		}
		if run.Status == authorityv1.RunInReview && metadata.LifecycleState != authorityv1.LifecycleInReview {
			return false
		}
	}
	if !validFailureAttemptSequence(metadata) {
		return false
	}
	reviewAccepted := len(metadata.ReviewRecords) == reviewerCount
	for _, review := range metadata.ReviewRecords {
		reviewAccepted = reviewAccepted && review.Verdict == authorityv1.ReviewAccepted
	}
	if metadata.ReviewAccepted != reviewAccepted ||
		(metadata.RunDispositionRecord == nil && metadata.RunDisposition != "") ||
		(metadata.RunDispositionRecord != nil && metadata.RunDisposition != string(metadata.RunDispositionRecord.Status)) ||
		metadata.Reconciled != (metadata.ReconciliationRecord != nil) {
		return false
	}
	blockedFailure := currentBlockedFailure(metadata)
	if blockedFailure == nil {
		if metadata.Blocker != "" || len(metadata.BlockedBy) != 0 {
			return false
		}
	} else if metadata.Blocker != blockedFailure.Reason || !equalMetadataStrings(metadata.BlockedBy, blockedFailure.BlockedBy) {
		return false
	}
	if metadata.ReconciliationRecord != nil {
		reconciliation := metadata.ReconciliationRecord
		if !safeToken(reconciliation.PrincipalProfileID) || reconciliation.HeadSHA != metadata.Handoff.HeadSHA ||
			!commitID(reconciliation.MergedSHA) || !commitID(reconciliation.MergedTree) || !safeToken(reconciliation.PullRequestID) ||
			!safeToken(reconciliation.ProtectedMainRunID) || !safeToken(reconciliation.IdempotencyKey) || !validEvidenceRefs(reconciliation.EvidenceRefs) {
			return false
		}
		if !addLifecycleKey(idempotencyKeys, reconciliation.IdempotencyKey) {
			return false
		}
	}
	if metadata.TerminalRecord != nil {
		terminal := metadata.TerminalRecord
		if metadata.LifecycleState != authorityv1.LifecycleDone || !safeToken(terminal.PrincipalProfileID) || terminal.HeadSHA != metadata.Handoff.HeadSHA ||
			!validEvidenceRefs(terminal.EvidenceRefs) || !safeToken(terminal.IdempotencyKey) {
			return false
		}
		if !addLifecycleKey(idempotencyKeys, terminal.IdempotencyKey) {
			return false
		}
	}
	if metadata.LifecycleState == authorityv1.LifecycleDone {
		if len(metadata.ReviewRecords) != reviewerCount || metadata.RunDispositionRecord == nil ||
			metadata.RunDispositionRecord.Status != authorityv1.RunCompleted || metadata.ReconciliationRecord == nil || metadata.TerminalRecord == nil {
			return false
		}
		for _, review := range metadata.ReviewRecords {
			if review.Verdict != authorityv1.ReviewAccepted {
				return false
			}
		}
	}
	return true
}

func validFailureAttemptSequence(metadata issueMetadata) bool {
	type fingerprintState struct {
		count   uint32
		blocked bool
	}
	states := map[string]fingerprintState{}
	visit := func(run *metadataRunDisposition) bool {
		if run == nil || run.Status == authorityv1.RunCompleted {
			return true
		}
		if run.Failure == nil || !safeToken(run.Failure.FailureFingerprint) {
			return false
		}
		state := states[run.Failure.FailureFingerprint]
		switch state.count {
		case 0:
			if run.Failure.Attempt != 1 {
				return false
			}
		case 1:
			if state.blocked || run.Failure.Attempt != 2 || run.Status != authorityv1.RunBlocked {
				return false
			}
		default:
			return false
		}
		state.count++
		state.blocked = state.blocked || run.Status == authorityv1.RunBlocked
		states[run.Failure.FailureFingerprint] = state
		return true
	}
	for _, cycle := range metadata.ReviewHistory {
		for index := range cycle.RunHistory {
			if !visit(&cycle.RunHistory[index]) {
				return false
			}
		}
		if !visit(cycle.RunDisposition) {
			return false
		}
	}
	for index := range metadata.RunHistory {
		if !visit(&metadata.RunHistory[index]) {
			return false
		}
	}
	return visit(metadata.RunDispositionRecord)
}

func validArchivedReviewCycle(cycle metadataReviewCycle, verificationOrder []string, reviewerCount int) bool {
	if !validMetadataHandoff(cycle.Handoff) || cycle.Handoff.NextProfileID != verificationOrder[0] || len(cycle.Reviews) > reviewerCount {
		return false
	}
	for index, review := range cycle.Reviews {
		if review.ReviewerProfileID != verificationOrder[index] || review.HeadSHA != cycle.Handoff.HeadSHA || !safeToken(review.IdempotencyKey) ||
			!validEvidenceRefs(review.EvidenceRefs) || !validMetadataReviewFailure(review) ||
			(index < len(cycle.Reviews)-1 && review.Verdict != authorityv1.ReviewAccepted) {
			return false
		}
	}
	for index := range cycle.RunHistory {
		if !validMetadataRun(&cycle.RunHistory[index], cycle.Handoff.HeadSHA, verificationOrder[len(verificationOrder)-1]) || cycle.RunHistory[index].Status == authorityv1.RunCompleted {
			return false
		}
	}
	if cycle.RunDisposition != nil && (!validMetadataRun(cycle.RunDisposition, cycle.Handoff.HeadSHA, verificationOrder[len(verificationOrder)-1]) || cycle.RunDisposition.Status == authorityv1.RunCompleted) {
		return false
	}
	lastReviewNonAccepted := len(cycle.Reviews) > 0 && cycle.Reviews[len(cycle.Reviews)-1].Verdict != authorityv1.ReviewAccepted
	return lastReviewNonAccepted || cycle.RunDisposition != nil && cycle.RunDisposition.Status != authorityv1.RunCompleted
}

func validMetadataHandoff(handoff metadataHandoff) bool {
	return safeToken(handoff.AttemptID) && safeToken(handoff.CanonicalClaimAttemptID) && isLowerHex(handoff.FenceDigest, 64) &&
		commitID(handoff.HeadSHA) && safeToken(handoff.NextProfileID) && safeToken(handoff.IdempotencyKey) && validEvidenceRefs(handoff.EvidenceRefs)
}

func validMetadataReviewFailure(review metadataReview) bool {
	if !knownReviewVerdict(review.Verdict) {
		return false
	}
	if review.Verdict == authorityv1.ReviewBlocked {
		return validMetadataFailure(review.Failure, true, true)
	}
	return review.Failure == nil
}

func validMetadataRun(run *metadataRunDisposition, headSHA, coordinator string) bool {
	if run == nil || run.PrincipalProfileID != coordinator || run.HeadSHA != headSHA || !safeToken(run.IdempotencyKey) ||
		!validEvidenceRefs(run.EvidenceRefs) || !knownRunDisposition(run.Status) {
		return false
	}
	if run.Status == authorityv1.RunCompleted {
		return run.Failure == nil
	}
	return validMetadataFailure(run.Failure, run.Status == authorityv1.RunBlocked, true)
}

func validMetadataFailure(value *authorityv1.FailureContext, requireBlockedBy, requireFingerprint bool) bool {
	if value == nil || !safeToken(value.Reason) || !safeToken(value.NextAction) || value.Attempt == 0 || value.Attempt > 2 ||
		len(value.BlockedBy) > 16 || hasDuplicateStrings(value.BlockedBy) || requireBlockedBy && len(value.BlockedBy) == 0 ||
		requireFingerprint && !safeToken(value.FailureFingerprint) || !requireFingerprint && value.FailureFingerprint != "" && !safeToken(value.FailureFingerprint) {
		return false
	}
	for _, blocker := range value.BlockedBy {
		if !safeToken(blocker) {
			return false
		}
	}
	return true
}

func currentBlockedFailure(metadata issueMetadata) *authorityv1.FailureContext {
	if metadata.RunDispositionRecord != nil && metadata.RunDispositionRecord.Status == authorityv1.RunBlocked {
		return metadata.RunDispositionRecord.Failure
	}
	if len(metadata.ReviewRecords) > 0 {
		last := metadata.ReviewRecords[len(metadata.ReviewRecords)-1]
		if last.Verdict == authorityv1.ReviewBlocked {
			return last.Failure
		}
	}
	return nil
}

func equalMetadataStrings(left, right []string) bool {
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

func addLifecycleKey(seen map[string]bool, key string) bool {
	if seen[key] {
		return false
	}
	seen[key] = true
	return true
}

func validEvidenceRefs(values []string) bool {
	if len(values) == 0 || len(values) > 16 || hasDuplicateStrings(values) {
		return false
	}
	for _, value := range values {
		if !safeToken(value) {
			return false
		}
	}
	return true
}

func commitID(value string) bool { return isLowerHex(value, 40) || isLowerHex(value, 64) }

func knownReviewVerdict(verdict authorityv1.ReviewVerdict) bool {
	return verdict == authorityv1.ReviewAccepted || verdict == authorityv1.ReviewChangesRequested || verdict == authorityv1.ReviewBlocked
}

func knownRunDisposition(status authorityv1.RunDispositionStatus) bool {
	switch status {
	case authorityv1.RunCompleted, authorityv1.RunBlocked, authorityv1.RunInReview, authorityv1.RunChangesRequested,
		authorityv1.RunNoWork, authorityv1.RunPreempted, authorityv1.RunCancelled, authorityv1.RunFailed:
		return true
	default:
		return false
	}
}

func projectionDigest(item authorityv1.WorkItem) string { return digestValue(item) }

func canonicalJSONDigest(data []byte) string {
	if rejectDuplicateJSONKeys(data) != nil {
		return ""
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if decoder.Decode(&value) != nil {
		return ""
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return ""
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(canonical)
	return hex.EncodeToString(digest[:])
}

func claimPostMetadata(pre []byte, attemptID, idempotencyKey, baseCommit string) ([]byte, error) {
	if rejectDuplicateJSONKeys(pre) != nil {
		return nil, ErrProjectionInvalid
	}
	decoder := json.NewDecoder(bytes.NewReader(pre))
	decoder.UseNumber()
	var values map[string]any
	if decoder.Decode(&values) != nil || values["lifecycleState"] != string(authorityv1.LifecycleBacklog) || attemptID == "" || idempotencyKey == "" || baseCommit == "" {
		return nil, ErrProjectionInvalid
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, ErrProjectionInvalid
	}
	if _, exists := values["workClaim"]; exists {
		return nil, ErrProjectionInvalid
	}
	if _, exists := values["bootstrapClaim"]; exists {
		return nil, ErrProjectionInvalid
	}
	version, ok := values["workVersion"].(map[string]any)
	if !ok {
		return nil, ErrProjectionInvalid
	}
	sequenceNumber, ok := version["issueMutationSequence"].(json.Number)
	if !ok {
		return nil, ErrProjectionInvalid
	}
	sequence, err := strconv.ParseUint(sequenceNumber.String(), 10, 64)
	if err != nil || sequence == 0 || sequence == ^uint64(0) {
		return nil, ErrProjectionInvalid
	}
	version["issueMutationSequence"] = json.Number(strconv.FormatUint(sequence+1, 10))
	values["lifecycleState"] = string(authorityv1.LifecycleInProgress)
	values["workClaim"] = map[string]any{
		"attemptId": attemptID, "idempotencyKey": idempotencyKey, "baseCommit": baseCommit,
	}
	result, err := json.Marshal(values)
	if err != nil {
		return nil, ErrProjectionInvalid
	}
	return result, nil
}

func digestValue(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func sortedCopy(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}

func knownLabel(label authorityv1.Label) bool {
	switch label {
	case authorityv1.LabelPublicAccepted, authorityv1.LabelExternalUntrusted, authorityv1.LabelPrivateData, authorityv1.LabelExternalEffect:
		return true
	default:
		return false
	}
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

func hasDuplicateStrings(values []string) bool {
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			return true
		}
		seen[value] = true
	}
	return false
}
