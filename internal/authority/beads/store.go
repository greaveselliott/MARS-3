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
	Risk                        string                     `json:"risk"`
	WorkType                    string                     `json:"workType"`
	Coordinator                 string                     `json:"coordinator"`
	FailureOwnership            string                     `json:"failureOwnership"`
	PublicDisclosure            bool                       `json:"publicDisclosure"`
	ContractPublicationRequired bool                       `json:"contractPublicationRequired,omitempty"`
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
	if !onlyObjectKeys(object, "acceptance_criteria", "assignee", "comment_count", "created_at", "created_by", "dependencies", "dependency_count", "dependent_count", "description", "id", "issue_type", "labels", "metadata", "owner", "priority", "started_at", "status", "title", "updated_at") {
		return issueSnapshot{}, ErrProjectionInvalid
	}
	var id, status, assignee, createdAt, updatedAt string
	var nativeLabels []string
	var metadata issueMetadata
	var dependenciesRaw []json.RawMessage
	if decodeField(object, "id", &id) != nil || decodeField(object, "status", &status) != nil || decodeOptionalString(object, "assignee", &assignee) != nil ||
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
	if metadata.LifecycleState == authorityv1.LifecycleInProgress || metadata.LifecycleState == authorityv1.LifecycleInReview {
		binding := metadata.WorkClaim
		if binding == nil {
			binding = metadata.BootstrapClaim
		}
		if binding == nil || binding.AttemptID == "" || binding.IdempotencyKey == "" || binding.BaseCommit == "" || assignee == "" {
			return authorityv1.WorkItem{}, ErrProjectionInvalid
		}
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
			RunCompleted:   dependency.Metadata.RunDisposition == "completed",
			Reconciled:     dependency.Metadata.Reconciled,
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
	item.Integrity = authorityv1.IntegrityDigests{
		Lineage: digestValue(struct {
			BeadID, DisplayID, Assignee, FeatureID                      string
			GoalIDs, ProductDecisionIDs, ScenarioIDs, VerificationOrder []string
		}{id, metadata.DisplayID, assignee, metadata.FeatureID, goalIDs, decisions, scenarios, verification}),
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
