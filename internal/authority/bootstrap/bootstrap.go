/*
FactoryDocSync:
docs:
- docs/features/F-002-work-authority.md
- docs/design-docs/ADR-001-git-beads-authority.md
- docs/code-documentation-map.md
*/

package bootstrap

import (
	"bytes"
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
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/greaveselliott/MARS-3/internal/doctrine"
)

type Options struct {
	Repo           string
	BeadsSource    string
	BeadsWorkspace string
	BeadsBinary    string
	Apply          bool
	Stdout         io.Writer
	Stderr         io.Writer
}

type issueRecord struct {
	ID        string          `json:"id"`
	Status    string          `json:"status"`
	Assignee  string          `json:"assignee"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	Metadata  json.RawMessage `json:"metadata"`
	Labels    []string        `json:"labels"`
	Deps      []struct {
		ID             string          `json:"id"`
		Status         string          `json:"status"`
		DependencyType string          `json:"dependency_type"`
		Metadata       json.RawMessage `json:"metadata"`
	} `json:"dependencies"`
}

type receipt struct {
	SchemaVersion       int    `json:"schemaVersion"`
	Kind                string `json:"kind"`
	Classification      string `json:"classification"`
	GrantID             string `json:"grantId"`
	AttemptID           string `json:"attemptId"`
	IdempotencyKey      string `json:"idempotencyKey"`
	Bead                string `json:"bead"`
	BaseCommit          string `json:"baseCommit"`
	PatchedBinarySHA256 string `json:"patchedBinarySHA256"`
	Mode                string `json:"mode"`
	Result              string `json:"result"`
	NativeStatus        string `json:"nativeStatus"`
	LifecycleState      string `json:"lifecycleState"`
	LiveLeaseAsserted   bool   `json:"liveLeaseAsserted"`
}

func Run(options Options) error {
	if options.Stdout == nil {
		options.Stdout = io.Discard
	}
	if options.Stderr == nil {
		options.Stderr = io.Discard
	}
	for name, value := range map[string]string{
		"repo": options.Repo, "beads-source": options.BeadsSource,
		"beads-workspace": options.BeadsWorkspace, "beads-binary": options.BeadsBinary,
	} {
		if value == "" {
			return fmt.Errorf("--%s is required", name)
		}
	}

	config, err := doctrine.LoadW001BootstrapGrant(options.Repo)
	if err != nil {
		return err
	}
	expiresAt, err := time.Parse(time.RFC3339, config.ExpiresAt)
	if err != nil || !time.Now().UTC().Before(expiresAt) {
		return errors.New("signed W-001 bootstrap authority is expired")
	}
	if err := verifyRepository(options.Repo, config, options.Apply); err != nil {
		return err
	}
	if err := verifyWorkspace(options.BeadsWorkspace, config); err != nil {
		return err
	}
	if err := verifyOriginalBinary(options.BeadsBinary, config); err != nil {
		return err
	}
	if err := verifySource(options.BeadsSource, config); err != nil {
		return err
	}
	pre, err := readIssue(options.BeadsBinary, options.BeadsWorkspace, config.Bead)
	if err != nil {
		return err
	}
	alreadyApplied := false
	if err := verifyPreimage(pre, config); err != nil {
		if !options.Apply || verifyPostimage(pre, config) != nil {
			return err
		}
		alreadyApplied = true
	}

	patchedBinary, patchedDigest, cleanup, err := buildPatchedBinary(options.Repo, options.BeadsSource, config, options.Stdout, options.Stderr)
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		return err
	}

	mode := "dry-run"
	result := "conformance-passed-no-canonical-mutation"
	post := pre
	if options.Apply && !alreadyApplied {
		mode = "apply"
		script, err := claimScript(pre, config)
		if err != nil {
			return err
		}
		command := exec.Command(patchedBinary, "-C", options.BeadsWorkspace, "--actor", config.Assignee, "--json", "batch", "--message", "MARS-3 W-001 signed bootstrap claim")
		command.Stdin = strings.NewReader(script)
		command.Stdout = options.Stdout
		command.Stderr = options.Stderr
		if err := command.Run(); err != nil {
			return fmt.Errorf("atomic bootstrap claim failed: %w", err)
		}
		post, err = readIssue(options.BeadsBinary, options.BeadsWorkspace, config.Bead)
		if err != nil {
			return fmt.Errorf("read canonical claim postimage: %w", err)
		}
		if err := verifyPostimage(post, config); err != nil {
			return err
		}
		result = "canonical-claim-verified"
	} else if options.Apply {
		mode = "apply"
		result = "canonical-claim-already-verified"
	}

	lifecycle := metadataString(post.Metadata, "lifecycleState")
	encoded, err := json.Marshal(receipt{
		SchemaVersion: 1, Kind: "MARS3BootstrapClaimReceipt", Classification: "PUBLIC",
		GrantID: config.ID, AttemptID: config.AttemptID, IdempotencyKey: config.IdempotencyKey, Bead: config.Bead,
		BaseCommit: config.BaseCommit, PatchedBinarySHA256: patchedDigest,
		Mode: mode, Result: result, NativeStatus: post.Status,
		LifecycleState: lifecycle, LiveLeaseAsserted: false,
	})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(options.Stdout, string(encoded))
	return err
}

func verifyRepository(repo string, config doctrine.W001BootstrapGrant, apply bool) error {
	head, err := gitOutput(repo, "rev-parse", "HEAD")
	if err != nil {
		return fmt.Errorf("resolve repository HEAD: %w", err)
	}
	if _, err := gitOutput(repo, "merge-base", "--is-ancestor", config.BaseCommit, strings.TrimSpace(head)); err != nil {
		return errors.New("signed bootstrap base is not an ancestor of HEAD")
	}
	status, err := gitOutput(repo, "status", "--porcelain=v1", "--untracked-files=all")
	if err != nil || strings.TrimSpace(status) != "" {
		return errors.New("bootstrap claim requires a clean repository checkout")
	}
	branch, err := gitOutput(repo, "symbolic-ref", "--quiet", "--short", "HEAD")
	currentBranch := strings.TrimSpace(branch)
	if apply && (err != nil || currentBranch != "main") {
		return errors.New("canonical bootstrap claim executes only from accepted main")
	}
	if apply {
		remoteMain, remoteErr := gitOutput(repo, "rev-parse", "--verify", "refs/remotes/origin/main^{commit}")
		if remoteErr != nil || strings.TrimSpace(remoteMain) != strings.TrimSpace(head) {
			return errors.New("canonical bootstrap claim requires HEAD to equal the observed origin/main")
		}
	}
	if !apply && (err != nil || currentBranch != "main" && currentBranch != config.WorkingBranch) {
		return errors.New("bootstrap conformance requires accepted main or the exact signed working branch")
	}
	return nil
}

func verifyWorkspace(workspace string, config doctrine.W001BootstrapGrant) error {
	data, err := os.ReadFile(filepath.Join(workspace, ".beads", "metadata.json"))
	if err != nil {
		return fmt.Errorf("read Beads workspace identity: %w", err)
	}
	var metadata struct {
		ProjectID string `json:"project_id"`
		Database  string `json:"database"`
	}
	if err := json.Unmarshal(data, &metadata); err != nil || metadata.ProjectID != config.AuthorityProjectID || metadata.Database != "dolt" {
		return errors.New("Beads workspace identity does not match the signed authority project")
	}
	return nil
}

func verifyOriginalBinary(path string, config doctrine.W001BootstrapGrant) error {
	digest, err := fileDigest(path)
	if err != nil {
		return fmt.Errorf("hash pinned Beads binary: %w", err)
	}
	if digest != config.BeadsBinarySHA256 {
		return errors.New("Beads binary does not match the signed SHA-256")
	}
	command := exec.Command(path, "version", "--json")
	output, err := command.Output()
	if err != nil || !bytes.Contains(output, []byte(config.BeadsSourceCommit)) || !bytes.Contains(output, []byte(`"version": "`+config.BeadsVersion+`"`)) {
		return errors.New("Beads binary version/source revision does not match the signed grant")
	}
	return nil
}

func verifySource(source string, config doctrine.W001BootstrapGrant) error {
	head, err := gitOutput(source, "rev-parse", "HEAD")
	if err != nil || strings.TrimSpace(head) != config.BeadsSourceCommit {
		return errors.New("Beads source checkout is not the exact signed revision")
	}
	status, err := gitOutput(source, "status", "--porcelain=v1", "--untracked-files=all")
	if err != nil || strings.TrimSpace(status) != "" {
		return errors.New("Beads source checkout must be clean")
	}
	if runtime.Version() != config.GoVersion || runtime.GOOS != config.GoOS || runtime.GOARCH != config.GoArch {
		return errors.New("local Go toolchain does not match the signed bootstrap grant")
	}
	goMod, err := os.ReadFile(filepath.Join(source, "go.mod"))
	if err != nil || !bytes.Contains(goMod, []byte("\n\t"+config.DoltModule+"\n")) {
		return errors.New("Beads source does not pin the signed Dolt module")
	}
	goSum, err := os.ReadFile(filepath.Join(source, "go.sum"))
	if err != nil || !bytes.Contains(goSum, []byte(config.DoltModule+" h1:"+config.DoltModuleSHA256+"\n")) {
		return errors.New("Beads source Dolt module checksum does not match the signed grant")
	}
	return nil
}

func buildPatchedBinary(repo, source string, config doctrine.W001BootstrapGrant, stdout, stderr io.Writer) (string, string, func(), error) {
	temporary, err := os.MkdirTemp("", "mars3-w001-bootstrap-")
	if err != nil {
		return "", "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(temporary) }
	checkout := filepath.Join(temporary, "beads")
	if err := run(stdout, stderr, "git", "clone", "--quiet", "--no-hardlinks", source, checkout); err != nil {
		return "", "", cleanup, fmt.Errorf("clone pinned Beads source: %w", err)
	}
	if err := run(stdout, stderr, "git", "-C", checkout, "checkout", "--quiet", "--detach", config.BeadsSourceCommit); err != nil {
		return "", "", cleanup, fmt.Errorf("checkout pinned Beads source: %w", err)
	}
	patch := filepath.Join(repo, filepath.FromSlash(config.PatchPath))
	if digest, err := fileDigest(patch); err != nil || digest != config.PatchSHA256 {
		return "", "", cleanup, errors.New("bootstrap patch does not match the signed SHA-256")
	}
	if err := run(stdout, stderr, "git", "-C", checkout, "apply", "--unidiff-zero", "--check", patch); err != nil {
		return "", "", cleanup, fmt.Errorf("bootstrap patch preflight: %w", err)
	}
	if err := run(stdout, stderr, "git", "-C", checkout, "apply", "--unidiff-zero", patch); err != nil {
		return "", "", cleanup, fmt.Errorf("apply bootstrap patch: %w", err)
	}
	icuPrefix, err := verifyConformanceDependencies(config)
	if err != nil {
		return "", "", cleanup, err
	}
	test := exec.Command("go", "test", "-json", "./cmd/bd", "-run", "^TestBatchBootstrapClaim", "-count=1")
	test.Dir = checkout
	test.Env = append(os.Environ(), "GOTOOLCHAIN=local", "CGO_ENABLED=1", "CGO_CPPFLAGS=-I"+filepath.Join(icuPrefix, "include"), "CGO_LDFLAGS=-L"+filepath.Join(icuPrefix, "lib"))
	var testOutput bytes.Buffer
	test.Stdout, test.Stderr = io.MultiWriter(stdout, &testOutput), stderr
	if err := test.Run(); err != nil {
		return "", "", cleanup, fmt.Errorf("patched Beads conformance: %w", err)
	}
	if err := verifyAtomicityTestResults(testOutput.Bytes()); err != nil {
		return "", "", cleanup, err
	}
	binary := filepath.Join(temporary, "bd-w001-bootstrap")
	build := exec.Command("go", "build", "-trimpath", "-o", binary, "./cmd/bd")
	build.Dir = checkout
	build.Env = append(os.Environ(), "GOTOOLCHAIN=local", "CGO_ENABLED=0")
	build.Stdout, build.Stderr = stdout, stderr
	if err := build.Run(); err != nil {
		return "", "", cleanup, fmt.Errorf("build patched Beads binary: %w", err)
	}
	digest, err := fileDigest(binary)
	if err != nil {
		return "", "", cleanup, err
	}
	return binary, digest, cleanup, nil
}

func verifyConformanceDependencies(config doctrine.W001BootstrapGrant) (string, error) {
	formula, err := exec.Command("brew", "list", "--versions", "icu4c@78").Output()
	if err != nil || strings.TrimSpace(string(formula)) != config.ICUFormula {
		return "", errors.New("local ICU formula does not match the signed conformance toolchain")
	}
	prefix, err := exec.Command("brew", "--prefix", "icu4c").Output()
	icuPrefix := strings.TrimSpace(string(prefix))
	if err != nil || !filepath.IsAbs(icuPrefix) {
		return "", errors.New("local ICU prefix cannot be resolved")
	}
	for _, path := range []string{filepath.Join(icuPrefix, "include", "unicode", "regex.h"), filepath.Join(icuPrefix, "lib", "libicuuc.dylib")} {
		resolved, resolveErr := filepath.EvalSymlinks(path)
		info, statErr := os.Stat(path)
		if resolveErr != nil || statErr != nil || !info.Mode().IsRegular() ||
			!strings.HasPrefix(resolved, filepath.Clean(icuPrefix)+string(filepath.Separator)) {
			return "", errors.New("local ICU conformance material is missing or indirect")
		}
	}
	for tag, expected := range map[string]string{
		"dolthub/dolt-sql-server:2.1.0": config.DoltTestImage,
		"testcontainers/ryuk:0.13.0":    config.RyukTestImage,
	} {
		output, inspectErr := exec.Command("docker", "image", "inspect", tag, "--format", "{{json .RepoDigests}}").Output()
		if inspectErr != nil || !bytes.Contains(output, []byte(`"`+expected+`"`)) {
			return "", fmt.Errorf("pinned disposable test image is unavailable: %s", tag)
		}
	}
	return icuPrefix, nil
}

func verifyAtomicityTestResults(output []byte) error {
	wanted := map[string]bool{
		"TestBatchBootstrapClaimIsOneAtomicTransition":        false,
		"TestBatchBootstrapClaimPreconditionFailureRollsBack": false,
	}
	for _, line := range bytes.Split(output, []byte("\n")) {
		var event struct {
			Action string `json:"Action"`
			Test   string `json:"Test"`
		}
		if len(line) == 0 || json.Unmarshal(line, &event) != nil {
			continue
		}
		if _, tracked := wanted[event.Test]; !tracked {
			continue
		}
		if event.Action == "skip" || event.Action == "fail" {
			return errors.New("disposable atomicity conformance skipped or failed")
		}
		if event.Action == "pass" {
			wanted[event.Test] = true
		}
	}
	for _, passed := range wanted {
		if !passed {
			return errors.New("disposable atomicity conformance did not execute every required test")
		}
	}
	return nil
}

func readIssue(binary, workspace, id string) (issueRecord, error) {
	command := exec.Command(binary, "-C", workspace, "show", id, "--json")
	output, err := command.Output()
	if err != nil {
		return issueRecord{}, fmt.Errorf("read canonical Bead: %w", err)
	}
	var issues []issueRecord
	if err := json.Unmarshal(output, &issues); err != nil || len(issues) != 1 {
		return issueRecord{}, errors.New("canonical Bead read must return exactly one record")
	}
	return issues[0], nil
}

func verifyPreimage(issue issueRecord, config doctrine.W001BootstrapGrant) error {
	if issue.ID != config.Bead || issue.Status != config.ExpectedNativeStatus || issue.Assignee != config.Assignee ||
		issue.CreatedAt.UTC().Format(time.RFC3339) != config.ExpectedCreatedAt ||
		issue.UpdatedAt.UTC().Format(time.RFC3339) != config.ExpectedUpdatedAt ||
		metadataString(issue.Metadata, "lifecycleState") != config.ExpectedLifecycleState {
		return errors.New("canonical Bead does not match the signed claim preimage")
	}
	metadataDigest, err := canonicalJSONDigest(issue.Metadata)
	if err != nil || metadataDigest != config.ExpectedMetadataSHA256 {
		return errors.New("canonical metadata does not match the signed digest")
	}
	labels := append([]string(nil), issue.Labels...)
	sort.Strings(labels)
	labelsJSON, _ := json.Marshal(labels)
	if digest(labelsJSON) != config.ExpectedLabelsSHA256 {
		return errors.New("canonical labels do not match the signed digest")
	}
	return verifyAuthorityBinding(issue, config)
}

func verifyAuthorityBinding(issue issueRecord, config doctrine.W001BootstrapGrant) error {
	if len(issue.Deps) != 1 || issue.Deps[0].ID != config.ExpectedDependency || issue.Deps[0].DependencyType != config.ExpectedDependencyType || issue.Deps[0].Status != config.ExpectedDependencyStatus ||
		metadataString(issue.Deps[0].Metadata, "lifecycleState") != config.ExpectedDependencyLifecycle {
		return errors.New("canonical dependency does not match the signed authority binding")
	}
	dependencyDigest, err := digestJSON(map[string]any{"dependencies": []any{map[string]any{
		"dependency_type": issue.Deps[0].DependencyType,
		"id":              issue.Deps[0].ID,
		"lifecycleState":  metadataString(issue.Deps[0].Metadata, "lifecycleState"),
		"status":          issue.Deps[0].Status,
	}}})
	if err != nil || dependencyDigest != config.ExpectedDependencySHA256 {
		return errors.New("canonical dependency graph does not match the signed digest")
	}
	var metadata map[string]any
	if err := json.Unmarshal(issue.Metadata, &metadata); err != nil {
		return errors.New("canonical metadata is not valid JSON")
	}
	lineageDigest, err := digestJSON(map[string]any{
		"assignee":           issue.Assignee,
		"coordinator":        metadata["coordinator"],
		"exclusivePaths":     metadata["exclusivePaths"],
		"failureOwnership":   metadata["failureOwnership"],
		"featureId":          metadata["featureId"],
		"goalIds":            metadata["goalIds"],
		"id":                 issue.ID,
		"productDecisionIds": metadata["productDecisionIds"],
		"scenarioIds":        metadata["scenarioIds"],
	})
	if err != nil || lineageDigest != config.ExpectedLineageSHA256 {
		return errors.New("canonical lineage does not match the signed digest")
	}
	return nil
}

func verifyPostimage(issue issueRecord, config doctrine.W001BootstrapGrant) error {
	if issue.ID != config.Bead || issue.Status != config.PostNativeStatus || issue.Assignee != config.Assignee ||
		metadataString(issue.Metadata, "lifecycleState") != config.PostLifecycleState {
		return errors.New("canonical Bead does not match the signed claim postimage")
	}
	metadataDigest, err := canonicalJSONDigest(issue.Metadata)
	if err != nil || metadataDigest != config.PostMetadataSHA256 {
		return errors.New("canonical postclaim metadata does not match the signed digest")
	}
	labels := append([]string(nil), issue.Labels...)
	sort.Strings(labels)
	labelsJSON, _ := json.Marshal(labels)
	if digest(labelsJSON) != config.PostLabelsSHA256 {
		return errors.New("canonical postclaim labels do not match the signed digest")
	}
	var metadata struct {
		BootstrapClaim struct {
			AttemptID      string `json:"attemptId"`
			BaseCommit     string `json:"baseCommit"`
			GrantID        string `json:"grantId"`
			IdempotencyKey string `json:"idempotencyKey"`
		} `json:"bootstrapClaim"`
	}
	if json.Unmarshal(issue.Metadata, &metadata) != nil || metadata.BootstrapClaim.AttemptID != config.AttemptID ||
		metadata.BootstrapClaim.BaseCommit != config.BaseCommit || metadata.BootstrapClaim.GrantID != config.ID ||
		metadata.BootstrapClaim.IdempotencyKey != config.IdempotencyKey {
		return errors.New("canonical postclaim idempotency binding does not match the signed grant")
	}
	return verifyAuthorityBinding(issue, config)
}

func claimScript(issue issueRecord, config doctrine.W001BootstrapGrant) (string, error) {
	postMetadata, err := base64.RawStdEncoding.DecodeString(config.PostMetadataBase64)
	if err != nil || !json.Valid(postMetadata) || digest(postMetadata) != config.PostMetadataSHA256 {
		return "", errors.New("signed postclaim metadata payload is invalid")
	}
	return fmt.Sprintf("bootstrap-claim %s expected_status=%s expected_assignee=%s expected_created_at=%s expected_updated_at=%s expected_metadata_sha256=%s expected_labels_sha256=%s expected_dependency=%s expected_dependency_type=%s expected_dependency_status=%s expected_dependency_lifecycle=%s expected_dependency_sha256=%s post_metadata_base64=%s remove_label=%s add_label=%s\n",
		config.Bead, config.ExpectedNativeStatus, config.Assignee,
		issue.CreatedAt.UTC().Format(time.RFC3339), issue.UpdatedAt.UTC().Format(time.RFC3339),
		config.ExpectedMetadataSHA256, config.ExpectedLabelsSHA256,
		config.ExpectedDependency, config.ExpectedDependencyType, config.ExpectedDependencyStatus,
		config.ExpectedDependencyLifecycle, config.ExpectedDependencySHA256,
		config.PostMetadataBase64, config.RemoveLabel, config.AddLabel), nil
}

func metadataString(data json.RawMessage, key string) string {
	var values map[string]interface{}
	if json.Unmarshal(data, &values) != nil {
		return ""
	}
	value, _ := values[key].(string)
	return value
}

func canonicalJSONDigest(data []byte) (string, error) {
	var decoded interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return "", err
	}
	canonical, err := json.Marshal(decoded)
	if err != nil {
		return "", err
	}
	return digest(canonical), nil
}

func digestJSON(value any) (string, error) {
	canonical, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return digest(canonical), nil
}

func digest(data []byte) string {
	value := sha256.Sum256(data)
	return hex.EncodeToString(value[:])
}

func fileDigest(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return digest(data), nil
}

func gitOutput(repo string, args ...string) (string, error) {
	command := exec.Command("git", append([]string{"-C", repo}, args...)...)
	output, err := command.Output()
	return string(output), err
}

func run(stdout, stderr io.Writer, name string, args ...string) error {
	command := exec.Command(name, args...)
	command.Stdout, command.Stderr = stdout, stderr
	return command.Run()
}
