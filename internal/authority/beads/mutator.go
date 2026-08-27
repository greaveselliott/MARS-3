/*
FactoryDocSync:
docs:
- docs/features/F-002-work-authority.md
- docs/design-docs/ADR-001-git-beads-authority.md
- docs/code-documentation-map.md
*/

package beads

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	authorityv1 "github.com/greaveselliott/MARS-3/api/authority/v1"
	"github.com/greaveselliott/MARS-3/internal/authority/gateway"
)

// NativeMutator invokes the reviewed Beads v1.2.2 gateway-CAS patch. The
// command has no ambient configuration, network credential, hook, redirect,
// or generic update authority. Its embedded transaction repeats the effective
// database and workspace binding before reading or writing canonical state.
type NativeMutator struct {
	reader       *CLIReader
	binary       string
	binarySHA256 string
}

func NewNativeMutator(reader *CLIReader, binary, binarySHA256 string) (*NativeMutator, error) {
	mutator := &NativeMutator{reader: reader, binary: binary, binarySHA256: binarySHA256}
	if reader == nil || mutator.verifyBoundary() != nil {
		return nil, ErrAtomicCASRequired
	}
	return mutator, nil
}

func (mutator *NativeMutator) CompareAndSwapClaim(ctx context.Context, claim AtomicClaim) ([]byte, error) {
	if mutator.verifyBoundary() != nil || !validAtomicClaim(claim) {
		return nil, ErrAtomicCASRequired
	}
	workspaceDigest, err := authorityWorkspaceDigest(mutator.reader.workspace, mutator.reader.projectID)
	if err != nil {
		return nil, ErrAtomicCASRequired
	}
	scratchHome, err := os.MkdirTemp("", "mars3-beads-mutation-")
	if err != nil {
		return nil, ErrAtomicCASRequired
	}
	command := exec.CommandContext(ctx, mutator.binary,
		"-C", ".", "--actor", claim.Assignee, "--json", "batch", "--message", "MARS-3 governed authority claim")
	command.Dir = mutator.reader.workspace
	command.Env = mutator.environment(scratchHome, workspaceDigest)
	command.Stdin = strings.NewReader(authorityClaimScript(claim))
	var stdout, stderr boundedBuffer
	command.Stdout, command.Stderr = &stdout, &stderr
	commandErr := command.Run()
	cleanupErr := os.RemoveAll(scratchHome)
	if cleanupErr != nil {
		return nil, ErrAtomicCASRequired
	}
	if commandErr != nil {
		if staleAuthorityClaim(stderr.Bytes()) {
			return nil, ErrCASStale
		}
		return nil, fmt.Errorf("%w: authority-claim.%s", ErrAtomicCASRequired, authorityFailureClass(stderr.Bytes()))
	}
	if !validAuthorityClaimReceipt(stdout.Bytes(), claim.BeadID) {
		return nil, fmt.Errorf("%w: authority-claim.receipt", ErrAtomicCASRequired)
	}
	post, err := mutator.reader.ReadIssue(ctx, claim.BeadID)
	if err != nil {
		return nil, ErrAtomicCASRequired
	}
	return post, nil
}

func (mutator *NativeMutator) CompareAndSwapLifecycle(ctx context.Context, transition AtomicLifecycleTransition) ([]byte, error) {
	if mutator.verifyBoundary() != nil || !validAtomicLifecycleTransition(transition) {
		return nil, ErrAtomicCASRequired
	}
	workspaceDigest, err := authorityWorkspaceDigest(mutator.reader.workspace, mutator.reader.projectID)
	if err != nil {
		return nil, ErrAtomicCASRequired
	}
	scratchHome, err := os.MkdirTemp("", "mars3-beads-lifecycle-")
	if err != nil {
		return nil, ErrAtomicCASRequired
	}
	command := exec.CommandContext(ctx, mutator.binary,
		"-C", ".", "--actor", transition.Transition.PrincipalProfileID, "--json", "batch", "--message", "MARS-3 governed lifecycle transition")
	command.Dir = mutator.reader.workspace
	command.Env = mutator.environment(scratchHome, workspaceDigest)
	command.Stdin = strings.NewReader(authorityTransitionScript(transition))
	var stdout, stderr boundedBuffer
	command.Stdout, command.Stderr = &stdout, &stderr
	commandErr := command.Run()
	cleanupErr := os.RemoveAll(scratchHome)
	if cleanupErr != nil {
		return nil, ErrAtomicCASRequired
	}
	if commandErr != nil {
		if staleAuthorityTransition(stderr.Bytes()) {
			return nil, ErrCASStale
		}
		return nil, fmt.Errorf("%w: authority-transition.%s", ErrAtomicCASRequired, authorityTransitionFailureClass(stderr.Bytes()))
	}
	if !validAuthorityTransitionReceipt(stdout.Bytes(), transition.BeadID) {
		return nil, fmt.Errorf("%w: authority-transition.receipt", ErrAtomicCASRequired)
	}
	post, err := mutator.reader.ReadIssue(ctx, transition.BeadID)
	if err != nil {
		return nil, ErrAtomicCASRequired
	}
	return post, nil
}

type nativeLifecycleIntent struct {
	Operation               string                           `json:"operation"`
	PrincipalProfileID      string                           `json:"principalProfileId"`
	AttemptID               string                           `json:"attemptId,omitempty"`
	CanonicalClaimAttemptID string                           `json:"canonicalClaimAttemptId,omitempty"`
	HeadSHA                 string                           `json:"headSHA"`
	EvidenceRefs            []string                         `json:"evidenceRefs"`
	NextProfileID           string                           `json:"nextProfileId,omitempty"`
	Verdict                 authorityv1.ReviewVerdict        `json:"verdict,omitempty"`
	RunStatus               authorityv1.RunDispositionStatus `json:"runStatus,omitempty"`
	MergedSHA               string                           `json:"mergedSHA,omitempty"`
	MergedTree              string                           `json:"mergedTree,omitempty"`
	PullRequestID           string                           `json:"pullRequestId,omitempty"`
	ProtectedMainRunID      string                           `json:"protectedMainRunId,omitempty"`
	IdempotencyKey          string                           `json:"idempotencyKey"`
}

func lifecycleIntent(transition gateway.LifecycleMutation) nativeLifecycleIntent {
	return nativeLifecycleIntent{
		Operation: string(transition.Operation), PrincipalProfileID: transition.PrincipalProfileID,
		AttemptID: transition.AttemptID, CanonicalClaimAttemptID: transition.CanonicalClaimAttemptID, HeadSHA: transition.HeadSHA, EvidenceRefs: append([]string(nil), transition.EvidenceRefs...),
		NextProfileID: transition.NextProfileID, Verdict: transition.Verdict, RunStatus: transition.RunStatus,
		MergedSHA: transition.MergedSHA, MergedTree: transition.MergedTree, PullRequestID: transition.PullRequestID,
		ProtectedMainRunID: transition.ProtectedMainRunID, IdempotencyKey: transition.IdempotencyKey,
	}
}

func validAtomicLifecycleTransition(transition AtomicLifecycleTransition) bool {
	intent := lifecycleIntent(transition.Transition)
	if !safeToken(transition.BeadID) || !safeToken(intent.PrincipalProfileID) || !isLowerHex(intent.HeadSHA, 40) && !isLowerHex(intent.HeadSHA, 64) ||
		!safeToken(intent.IdempotencyKey) || !validLifecycleIntent(intent) || !validWorkVersion(transition.ExpectedVersion) || !validIntegrity(transition.ExpectedIntegrity) ||
		transition.ExpectedStatus != "in_progress" || !safeToken(transition.ExpectedAssignee) || !validRFC3339(transition.ExpectedCreatedAt) ||
		!validRFC3339(transition.ExpectedUpdatedAt) || !isLowerHex(transition.ExpectedDigest, 64) || !isLowerHex(transition.MetadataSHA256, 64) ||
		!isLowerHex(transition.LabelsSHA256, 64) || !isLowerHex(transition.DependenciesSHA256, 64) ||
		len(intent.EvidenceRefs) == 0 || len(intent.EvidenceRefs) > 16 || hasDuplicateStrings(intent.EvidenceRefs) ||
		len(transition.PostMetadata) == 0 || len(transition.PostMetadata) > 1<<20 || rejectDuplicateJSONKeys(transition.PostMetadata) != nil {
		return false
	}
	for _, reference := range intent.EvidenceRefs {
		if !safeToken(reference) {
			return false
		}
	}
	switch transition.Transition.Operation {
	case gateway.LifecycleHandoff:
		return transition.PostStatus == "in_progress" && transition.RemoveLabel == "in-progress" && transition.AddLabel == "in-review"
	case gateway.LifecycleReview:
		if transition.Transition.Verdict == authorityv1.ReviewChangesRequested {
			return transition.PostStatus == "in_progress" && transition.RemoveLabel == "in-review" && transition.AddLabel == "in-progress"
		}
		return transition.PostStatus == "in_progress" && transition.RemoveLabel == "" && transition.AddLabel == ""
	case gateway.LifecycleRun, gateway.LifecycleReconcile:
		return transition.PostStatus == "in_progress" && transition.RemoveLabel == "" && transition.AddLabel == ""
	case gateway.LifecycleTerminal:
		return transition.PostStatus == "closed" && transition.RemoveLabel == "in-review" && transition.AddLabel == "done"
	default:
		return false
	}
}

func validLifecycleIntent(intent nativeLifecycleIntent) bool {
	noAttempt := intent.AttemptID == "" && intent.CanonicalClaimAttemptID == ""
	noMerge := intent.MergedSHA == "" && intent.MergedTree == "" && intent.PullRequestID == "" && intent.ProtectedMainRunID == ""
	switch gateway.LifecycleOperation(intent.Operation) {
	case gateway.LifecycleHandoff:
		return safeToken(intent.AttemptID) && safeToken(intent.CanonicalClaimAttemptID) && safeToken(intent.NextProfileID) && intent.Verdict == "" && intent.RunStatus == "" && noMerge
	case gateway.LifecycleReview:
		return noAttempt && intent.NextProfileID == "" && knownReviewVerdict(intent.Verdict) && intent.RunStatus == "" && noMerge
	case gateway.LifecycleRun:
		return noAttempt && intent.NextProfileID == "" && intent.Verdict == "" && knownRunDisposition(intent.RunStatus) && noMerge
	case gateway.LifecycleReconcile:
		return noAttempt && intent.NextProfileID == "" && intent.Verdict == "" && intent.RunStatus == "" &&
			(isLowerHex(intent.MergedSHA, 40) || isLowerHex(intent.MergedSHA, 64)) &&
			(isLowerHex(intent.MergedTree, 40) || isLowerHex(intent.MergedTree, 64)) && safeToken(intent.PullRequestID) && safeToken(intent.ProtectedMainRunID)
	case gateway.LifecycleTerminal:
		return noAttempt && intent.NextProfileID == "" && intent.Verdict == "" && intent.RunStatus == "" && noMerge
	default:
		return false
	}
}

func authorityTransitionScript(transition AtomicLifecycleTransition) string {
	intentJSON, _ := json.Marshal(lifecycleIntent(transition.Transition))
	return fmt.Sprintf("authority-transition %s expected_status=%s expected_assignee=%s expected_created_at=%s expected_updated_at=%s expected_metadata_sha256=%s expected_labels_sha256=%s expected_dependencies_sha256=%s transition_base64=%s post_status=%s remove_label=%s add_label=%s post_metadata_base64=%s\n",
		transition.BeadID, transition.ExpectedStatus, transition.ExpectedAssignee, transition.ExpectedCreatedAt, transition.ExpectedUpdatedAt,
		transition.MetadataSHA256, transition.LabelsSHA256, transition.DependenciesSHA256,
		base64.RawStdEncoding.EncodeToString(intentJSON), transition.PostStatus, transition.RemoveLabel, transition.AddLabel,
		base64.RawStdEncoding.EncodeToString(transition.PostMetadata))
}

func staleAuthorityTransition(stderr []byte) bool {
	message := string(stderr)
	for _, fingerprint := range []string{
		"authority-transition: issue precondition mismatch", "authority-transition: metadata precondition mismatch",
		"authority-transition: label precondition mismatch", "authority-transition: dependency precondition mismatch",
	} {
		if strings.Contains(message, fingerprint) {
			return true
		}
	}
	return false
}

func authorityTransitionFailureClass(stderr []byte) string {
	message := string(stderr)
	for _, check := range []struct{ fingerprint, class string }{
		{"authority-transition: direct", "boundary"}, {"authority-transition: transaction", "transaction"},
		{"authority-transition: post metadata", "post-metadata"}, {"authority-transition: issue update", "issue-update"},
		{"authority-transition: close issue", "issue-close"}, {"authority-transition: remove lifecycle label", "label-update"},
		{"authority-transition: add lifecycle label", "label-update"},
	} {
		if strings.Contains(message, check.fingerprint) {
			return check.class
		}
	}
	return "execution"
}

func validAuthorityTransitionReceipt(raw []byte, beadID string) bool {
	if rejectDuplicateJSONKeys(raw) != nil {
		return false
	}
	var receipt struct {
		Operations    int    `json:"operations"`
		SchemaVersion int    `json:"schema_version"`
		Status        string `json:"status"`
		Results       []struct {
			Line   int    `json:"line"`
			Op     string `json:"op"`
			Target string `json:"target"`
		} `json:"results"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&receipt) != nil || receipt.Operations != 1 || receipt.SchemaVersion != 1 || receipt.Status != "ok" || len(receipt.Results) != 1 {
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return false
	}
	result := receipt.Results[0]
	return result.Line == 1 && result.Op == "authority-transition" && result.Target == beadID
}

func (mutator *NativeMutator) verifyBoundary() error {
	if mutator == nil || mutator.reader == nil || mutator.reader.verifyBoundary() != nil {
		return ErrAtomicCASRequired
	}
	binary, err := filepath.Abs(mutator.binary)
	if err != nil || binary != filepath.Clean(mutator.binary) || !digestFile(binary, mutator.binarySHA256) {
		return ErrAtomicCASRequired
	}
	info, err := os.Lstat(binary)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return ErrAtomicCASRequired
	}
	_, err = authorityWorkspaceDigest(mutator.reader.workspace, mutator.reader.projectID)
	return err
}

func (mutator *NativeMutator) environment(scratchHome, workspaceDigest string) []string {
	return []string{
		"BEADS_DIR=" + filepath.Join(mutator.reader.workspace, ".beads"),
		"BEADS_DOLT_SERVER_DATABASE=M3", "BEADS_DOLT_SHARED_SERVER=0", "BEADS_DOLT_SERVER_MODE=0",
		"BD_DISABLE_METRICS=1", "BD_NO_HOOKS=1",
		"HOME=" + scratchHome, "XDG_CONFIG_HOME=" + filepath.Join(scratchHome, ".config"),
		"MARS3_AUTHORITY_DIRECT_BEADS_DIR=" + filepath.Join(mutator.reader.workspace, ".beads"),
		"MARS3_AUTHORITY_PROJECT_ID=" + mutator.reader.projectID, "MARS3_AUTHORITY_DATABASE=M3",
		"MARS3_AUTHORITY_WORKSPACE_SHA256=" + workspaceDigest,
		"PATH=/usr/bin:/bin", "NO_COLOR=1", "LANG=C", "TZ=UTC",
	}
}

func validAtomicClaim(claim AtomicClaim) bool {
	if !safeToken(claim.BeadID) || !safeToken(claim.AttemptID) || !safeToken(claim.Assignee) ||
		!safeToken(claim.IdempotencyKey) || !isLowerHex(claim.BaseCommit, 40) ||
		claim.ExpectedStatus != "open" || !safeOptionalToken(claim.ExpectedAssignee) ||
		!validRFC3339(claim.ExpectedCreatedAt) || !validRFC3339(claim.ExpectedUpdatedAt) ||
		!validWorkVersion(claim.ExpectedVersion) || !validIntegrity(claim.ExpectedIntegrity) ||
		!isLowerHex(claim.MetadataSHA256, 64) || !isLowerHex(claim.LabelsSHA256, 64) ||
		!isLowerHex(claim.DependenciesSHA256, 64) || !isLowerHex(claim.ExpectedDigest, 64) ||
		len(claim.PostMetadata) == 0 || len(claim.PostMetadata) > 1<<20 || rejectDuplicateJSONKeys(claim.PostMetadata) != nil {
		return false
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(claim.PostMetadata, &object) != nil {
		return false
	}
	if _, exists := object["bootstrapClaim"]; exists {
		return false
	}
	if value, exists := object["workClaim"]; !exists || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
		return false
	}
	var metadata issueMetadata
	decoder := json.NewDecoder(bytes.NewReader(claim.PostMetadata))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&metadata) != nil {
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return false
	}
	return metadata.LifecycleState == authorityv1.LifecycleInProgress && metadata.WorkClaim != nil &&
		metadata.WorkClaim.AttemptID == claim.AttemptID && metadata.WorkClaim.IdempotencyKey == claim.IdempotencyKey &&
		metadata.WorkClaim.BaseCommit == claim.BaseCommit && metadata.WorkClaim.GrantID == "" &&
		metadata.WorkVersion.AuthorityGeneration == claim.ExpectedVersion.AuthorityGeneration &&
		metadata.WorkVersion.IssueIncarnation == claim.ExpectedVersion.IssueIncarnation &&
		claim.ExpectedVersion.IssueMutationSequence != ^uint64(0) &&
		metadata.WorkVersion.IssueMutationSequence == claim.ExpectedVersion.IssueMutationSequence+1 &&
		metadata.WorkVersion.DependencyGraphRevision == claim.ExpectedVersion.DependencyGraphRevision
}

func validWorkVersion(version authorityv1.WorkVersion) bool {
	return safeToken(version.AuthorityGeneration) && safeToken(version.IssueIncarnation) &&
		version.IssueMutationSequence > 0 && version.DependencyGraphRevision > 0
}

func validIntegrity(integrity authorityv1.IntegrityDigests) bool {
	return isLowerHex(integrity.Lineage, 64) && isLowerHex(integrity.DependencyOutcomes, 64) &&
		isLowerHex(integrity.Blockers, 64) && isLowerHex(integrity.ExclusivePaths, 64)
}

func authorityClaimScript(claim AtomicClaim) string {
	return fmt.Sprintf("authority-claim %s expected_status=%s expected_assignee=%s expected_created_at=%s expected_updated_at=%s expected_metadata_sha256=%s expected_labels_sha256=%s expected_dependencies_sha256=%s attempt_id=%s idempotency_key=%s base_commit=%s post_metadata_base64=%s\n",
		claim.BeadID, claim.ExpectedStatus, claim.ExpectedAssignee, claim.ExpectedCreatedAt, claim.ExpectedUpdatedAt,
		claim.MetadataSHA256, claim.LabelsSHA256, claim.DependenciesSHA256, claim.AttemptID, claim.IdempotencyKey,
		claim.BaseCommit, base64.RawStdEncoding.EncodeToString(claim.PostMetadata))
}

func staleAuthorityClaim(stderr []byte) bool {
	message := string(stderr)
	for _, fingerprint := range []string{
		"authority-claim: issue precondition mismatch",
		"authority-claim: metadata precondition mismatch",
		"authority-claim: label precondition mismatch",
		"authority-claim: dependency precondition mismatch",
	} {
		if strings.Contains(message, fingerprint) {
			return true
		}
	}
	return false
}

func authorityFailureClass(stderr []byte) string {
	message := string(stderr)
	for _, check := range []struct{ fingerprint, class string }{
		{"authority-claim: direct", "boundary"},
		{"authority-claim: storage transaction", "transaction"},
		{"authority-claim: transaction is not", "transaction"},
		{"authority-claim: post metadata", "post-metadata"},
		{"authority-claim: native CAS claim", "native-cas"},
		{"authority-claim: metadata update", "metadata-update"},
		{"authority-claim: remove lifecycle label", "label-update"},
		{"authority-claim: add lifecycle label", "label-update"},
	} {
		if strings.Contains(message, check.fingerprint) {
			return check.class
		}
	}
	return "execution"
}

func validAuthorityClaimReceipt(raw []byte, beadID string) bool {
	if rejectDuplicateJSONKeys(raw) != nil {
		return false
	}
	var receipt struct {
		Operations    int    `json:"operations"`
		SchemaVersion int    `json:"schema_version"`
		Status        string `json:"status"`
		Results       []struct {
			Line   int    `json:"line"`
			Op     string `json:"op"`
			Target string `json:"target"`
		} `json:"results"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&receipt) != nil || receipt.Operations != 1 || receipt.SchemaVersion != 1 || receipt.Status != "ok" || len(receipt.Results) != 1 {
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return false
	}
	result := receipt.Results[0]
	return result.Line == 1 && result.Op == "authority-claim" && result.Target == beadID
}

type workspaceEntry struct {
	Path          string `json:"path"`
	Kind          string `json:"kind"`
	Device        uint64 `json:"device"`
	Inode         uint64 `json:"inode"`
	ContentSHA256 string `json:"contentSHA256,omitempty"`
}

func authorityWorkspaceDigest(root, projectID string) (string, error) {
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil || !filepath.IsAbs(resolved) {
		return "", ErrAtomicCASRequired
	}
	beadsDir := filepath.Join(resolved, ".beads")
	for _, forbidden := range []string{"redirect", ".env", "config.local.yaml"} {
		if _, err := os.Lstat(filepath.Join(beadsDir, forbidden)); !errors.Is(err, os.ErrNotExist) {
			return "", ErrAtomicCASRequired
		}
	}
	paths := []struct{ relative, kind string }{
		{".", "directory"}, {".beads", "directory"}, {".beads/metadata.json", "file"},
		{".beads/config.yaml", "file"}, {".beads/embeddeddolt", "directory"}, {".beads/embeddeddolt/M3", "directory"},
	}
	entries := make([]workspaceEntry, 0, len(paths))
	for _, required := range paths {
		path := filepath.Join(resolved, filepath.FromSlash(required.relative))
		info, statErr := os.Lstat(path)
		if statErr != nil || info.Mode()&os.ModeSymlink != 0 || required.kind == "directory" && !info.IsDir() || required.kind == "file" && !info.Mode().IsRegular() {
			return "", ErrAtomicCASRequired
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			return "", ErrAtomicCASRequired
		}
		entry := workspaceEntry{Path: required.relative, Kind: required.kind, Device: uint64(stat.Dev), Inode: uint64(stat.Ino)}
		if required.kind == "file" {
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return "", ErrAtomicCASRequired
			}
			entry.ContentSHA256 = digestBytes(content)
		}
		entries = append(entries, entry)
	}
	metadataRaw, err := os.ReadFile(filepath.Join(beadsDir, "metadata.json"))
	if err != nil || rejectDuplicateJSONKeys(metadataRaw) != nil {
		return "", ErrAtomicCASRequired
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
	if decoder.Decode(&metadata) != nil || metadata.Database != "dolt" || metadata.Backend != "dolt" || metadata.DoltMode != "embedded" || metadata.DoltDatabase != "M3" || metadata.ProjectID != projectID {
		return "", ErrAtomicCASRequired
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return "", ErrAtomicCASRequired
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
	return digestValue(payload), nil
}

func digestBytes(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func safeToken(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for _, character := range value {
		if !(character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character >= '0' && character <= '9' || strings.ContainsRune("._:-", character)) {
			return false
		}
	}
	return true
}

func safeOptionalToken(value string) bool { return value == "" || safeToken(value) }

func isLowerHex(value string, length int) bool {
	if len(value) != length {
		return false
	}
	for _, character := range value {
		if !(character >= '0' && character <= '9' || character >= 'a' && character <= 'f') {
			return false
		}
	}
	return true
}

func validRFC3339(value string) bool {
	parsed, err := time.Parse(time.RFC3339, value)
	return err == nil && parsed.UTC().Format(time.RFC3339) == value
}
