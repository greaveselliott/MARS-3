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
	"syscall"
	"time"

	"github.com/greaveselliott/MARS-3/internal/doctrine"
)

type Options struct {
	Repo                   string
	BeadsSource            string
	BeadsWorkspace         string
	BeadsBinary            string
	ExecutionAuthorization string
	Apply                  bool
	Stdout                 io.Writer
	Stderr                 io.Writer
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

type versionState struct {
	Branch string `json:"branch"`
	Commit string `json:"commit"`
}

type receipt struct {
	SchemaVersion           int    `json:"schemaVersion"`
	Kind                    string `json:"kind"`
	Classification          string `json:"classification"`
	GrantID                 string `json:"grantId"`
	AttemptID               string `json:"attemptId"`
	IdempotencyKey          string `json:"idempotencyKey"`
	Bead                    string `json:"bead"`
	BaseCommit              string `json:"baseCommit"`
	PatchedBinarySHA256     string `json:"patchedBinarySHA256"`
	WorkspaceInstanceSHA256 string `json:"workspaceInstanceSHA256"`
	DisposableBackend       string `json:"disposableBackend"`
	DisposableVerified      bool   `json:"disposableVerified"`
	CanonicalUnchanged      bool   `json:"canonicalUnchangedBeforeEffect"`
	Mode                    string `json:"mode"`
	Result                  string `json:"result"`
	NativeStatus            string `json:"nativeStatus"`
	LifecycleState          string `json:"lifecycleState"`
	LiveLeaseAsserted       bool   `json:"liveLeaseAsserted"`
}

const (
	bootstrapGitExecutable = "/usr/bin/git"
	canonicalRepositoryURL = "https://github.com/greaveselliott/MARS-3.git"
)

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
	repoRoot, err := filepath.Abs(options.Repo)
	if err != nil {
		return fmt.Errorf("resolve repository root: %w", err)
	}
	options.Repo = filepath.Clean(repoRoot)

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
	var execution doctrine.W001BootstrapExecutionAuthorization
	if options.Apply {
		if options.ExecutionAuthorization == "" {
			return errors.New("--execution-authorization is required for the canonical claim")
		}
		execution, err = doctrine.LoadW001BootstrapExecutionAuthorization(options.Repo, options.ExecutionAuthorization, config)
		if err != nil {
			return err
		}
		if err := verifyAcceptedMain(options.Repo, execution); err != nil {
			return err
		}
	}
	workspaceInstance, err := verifyWorkspace(options.BeadsWorkspace, config)
	if err != nil {
		return err
	}
	if options.Apply && execution.WorkspaceInstanceSHA256 != workspaceInstance {
		return errors.New("execution authorization does not bind the canonical workspace instance")
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
	if err := verifyPreimage(pre, config); err != nil {
		return err
	}

	patchedBinary, patchedDigest, cleanup, err := buildPatchedBinary(options.Repo, options.BeadsSource, config, options.Stdout, options.Stderr)
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		return err
	}
	if patchedDigest != config.PatchedBinarySHA256 {
		return errors.New("patched Beads binary does not match the signed SHA-256")
	}
	disposableBackend, err := verifyDisposableWorkspace(patchedBinary, options.BeadsBinary, options.BeadsWorkspace, pre, config, options.Stdout, options.Stderr)
	if err != nil {
		return err
	}
	canonicalBeforeEffect, err := readIssue(options.BeadsBinary, options.BeadsWorkspace, config.Bead)
	if err != nil || verifyPreimage(canonicalBeforeEffect, config) != nil {
		return errors.New("canonical Bead changed during disposable conformance")
	}

	mode := "dry-run"
	result := "conformance-passed-no-canonical-mutation"
	post := pre
	if options.Apply {
		freshExecution, err := doctrine.LoadW001BootstrapExecutionAuthorization(options.Repo, options.ExecutionAuthorization, config)
		if err != nil {
			return fmt.Errorf("revalidate execution authorization immediately before effect: %w", err)
		}
		if err := requireUnchangedExecutionAuthorization(execution, freshExecution); err != nil {
			return err
		}
		execution = freshExecution
		grantExpiry, err := time.Parse(time.RFC3339, config.ExpiresAt)
		if err != nil || !time.Now().UTC().Before(grantExpiry) {
			return errors.New("signed W-001 bootstrap authority expired during conformance")
		}
		if err := verifyAcceptedMain(options.Repo, execution); err != nil {
			return err
		}
		freshWorkspaceInstance, err := verifyWorkspace(options.BeadsWorkspace, config)
		if err != nil {
			return err
		}
		if freshWorkspaceInstance != workspaceInstance || freshWorkspaceInstance != execution.WorkspaceInstanceSHA256 {
			return errors.New("canonical workspace instance changed during conformance")
		}
		fresh, err := readIssue(options.BeadsBinary, options.BeadsWorkspace, config.Bead)
		if err != nil || verifyPreimage(fresh, config) != nil {
			return errors.New("canonical Bead preimage changed before the authorized effect")
		}
		beforeVersion, err := readVersionState(options.BeadsBinary, options.BeadsWorkspace)
		if err != nil {
			return err
		}
		mode = "apply"
		script, err := claimScript(fresh, config)
		if err != nil {
			return err
		}
		effectWorkspaceInstance, err := verifyWorkspace(options.BeadsWorkspace, config)
		if err != nil || effectWorkspaceInstance != execution.WorkspaceInstanceSHA256 {
			return errors.New("canonical workspace instance changed at the effect boundary")
		}
		command := exec.Command(patchedBinary, "-C", ".", "--actor", config.Assignee, "--json", "batch", "--message", "MARS-3 W-001 signed bootstrap claim")
		command.Dir = options.BeadsWorkspace
		command.Env = beadsCommandEnvironment()
		command.Stdin = strings.NewReader(script)
		command.Stdout = options.Stdout
		command.Stderr = options.Stderr
		if err := command.Run(); err != nil {
			return fmt.Errorf("atomic bootstrap claim failed with unknown acceptance; do not retry until canonical issue and Dolt history are reconciled: %w", err)
		}
		post, err = readIssue(options.BeadsBinary, options.BeadsWorkspace, config.Bead)
		if err != nil {
			return fmt.Errorf("read canonical claim postimage: %w", err)
		}
		if err := verifyPostimage(post, config); err != nil {
			return err
		}
		afterVersion, err := readVersionState(options.BeadsBinary, options.BeadsWorkspace)
		if err != nil || afterVersion.Commit == beforeVersion.Commit {
			return errors.New("canonical claim lacks a verified new Dolt version commit; block and reconcile before retry")
		}
		result = "canonical-claim-verified"
	}

	lifecycle := metadataString(post.Metadata, "lifecycleState")
	encoded, err := json.Marshal(receipt{
		SchemaVersion: 1, Kind: "MARS3BootstrapClaimReceipt", Classification: "PUBLIC",
		GrantID: config.ID, AttemptID: config.AttemptID, IdempotencyKey: config.IdempotencyKey, Bead: config.Bead,
		BaseCommit: config.BaseCommit, PatchedBinarySHA256: patchedDigest, WorkspaceInstanceSHA256: workspaceInstance,
		DisposableBackend: disposableBackend, DisposableVerified: true, CanonicalUnchanged: true,
		Mode: mode, Result: result, NativeStatus: post.Status,
		LifecycleState: lifecycle, LiveLeaseAsserted: false,
	})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(options.Stdout, string(encoded))
	return err
}

func requireUnchangedExecutionAuthorization(initial, fresh doctrine.W001BootstrapExecutionAuthorization) error {
	if initial != fresh {
		return errors.New("execution authorization changed during conformance")
	}
	return nil
}

func verifyRepository(repo string, config doctrine.W001BootstrapGrant, apply bool) error {
	metadata, err := os.Lstat(filepath.Join(repo, ".git"))
	if err != nil || !metadata.IsDir() || metadata.Mode()&os.ModeSymlink != 0 {
		return errors.New("bootstrap repository requires one direct Git directory")
	}
	topLevel, err := gitOutput(repo, "rev-parse", "--show-toplevel")
	if err != nil || filepath.Clean(strings.TrimSpace(topLevel)) != filepath.Clean(repo) {
		return errors.New("Git metadata does not resolve to the audited repository root")
	}
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
		remoteMain, remoteErr := gitRemoteMain()
		if remoteErr != nil || remoteMain != strings.TrimSpace(head) {
			return errors.New("canonical bootstrap claim requires HEAD to equal authenticated GitHub main")
		}
	}
	if !apply && (err != nil || currentBranch != "main" && currentBranch != config.WorkingBranch) {
		return errors.New("bootstrap conformance requires accepted main or the exact signed working branch")
	}
	return nil
}

func verifyAcceptedMain(repo string, authorization doctrine.W001BootstrapExecutionAuthorization) error {
	head, err := gitOutput(repo, "rev-parse", "HEAD")
	if err != nil || strings.TrimSpace(head) != authorization.MergedCommit {
		return errors.New("execution authorization does not bind the checked-out main commit")
	}
	tree, err := gitOutput(repo, "rev-parse", "HEAD^{tree}")
	if err != nil || strings.TrimSpace(tree) != authorization.MergedTree {
		return errors.New("accepted main tree does not match the execution authorization")
	}
	remoteMain, err := gitRemoteMain()
	if err != nil || remoteMain != authorization.MergedCommit {
		return errors.New("authenticated GitHub main does not match the execution authorization")
	}
	target, err := gitOutput(repo, "rev-list", "-n", "1", authorization.ReviewTag)
	if err != nil || strings.TrimSpace(target) != authorization.ReviewedFeatureCommit {
		return errors.New("execution authorization does not bind the immutable reviewed feature tag")
	}
	reviewedTree, err := gitOutput(repo, "rev-parse", authorization.ReviewedFeatureCommit+"^{tree}")
	if err != nil || strings.TrimSpace(reviewedTree) != authorization.MergedTree {
		return errors.New("reviewed feature and accepted main trees differ")
	}
	return nil
}

func verifyWorkspace(workspace string, config doctrine.W001BootstrapGrant) (string, error) {
	info, err := os.Lstat(workspace)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("Beads workspace must be one direct directory, not an indirect path")
	}
	metadataPath := filepath.Join(workspace, ".beads", "metadata.json")
	metadataInfo, err := os.Lstat(metadataPath)
	if err != nil || !metadataInfo.Mode().IsRegular() || metadataInfo.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("Beads workspace metadata must be one direct regular file")
	}
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return "", fmt.Errorf("read Beads workspace identity: %w", err)
	}
	var metadata struct {
		ProjectID string `json:"project_id"`
		Database  string `json:"database"`
	}
	if err := json.Unmarshal(data, &metadata); err != nil || metadata.ProjectID != config.AuthorityProjectID || metadata.Database != "dolt" {
		return "", errors.New("Beads workspace identity does not match the signed authority project")
	}
	return workspaceInstanceDigest(workspace, config.AuthorityProjectID)
}

func workspaceInstanceDigest(workspace, projectID string) (string, error) {
	absolute, err := filepath.Abs(workspace)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", errors.New("Beads workspace path cannot be resolved")
	}
	type entry struct {
		Path   string `json:"path"`
		Device uint64 `json:"device"`
		Inode  uint64 `json:"inode"`
	}
	payload := struct {
		SchemaVersion int     `json:"schemaVersion"`
		ProjectID     string  `json:"projectId"`
		Root          string  `json:"root"`
		Entries       []entry `json:"entries"`
	}{SchemaVersion: 1, ProjectID: projectID, Root: filepath.Clean(resolved)}
	for _, relative := range []string{".", ".beads", ".beads/embeddeddolt", ".beads/embeddeddolt/M3"} {
		path := filepath.Join(resolved, filepath.FromSlash(relative))
		info, statErr := os.Lstat(path)
		if statErr != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return "", errors.New("Beads workspace instance path is not one direct directory")
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			return "", errors.New("Beads workspace instance has no stable filesystem identity")
		}
		payload.Entries = append(payload.Entries, entry{Path: relative, Device: uint64(stat.Dev), Inode: uint64(stat.Ino)})
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return digest(encoded), nil
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
	command.Env = beadsCommandEnvironment()
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
	goBinary, baseEnv, err := verifiedGoEnvironment(config, temporary)
	if err != nil {
		return "", "", cleanup, err
	}
	icuPrefix, err := verifyConformanceDependencies(config)
	if err != nil {
		return "", "", cleanup, err
	}
	test := exec.Command(goBinary, "test", "-json", "./cmd/bd", "-run", "^TestBatchBootstrapClaim", "-count=1")
	test.Dir = checkout
	test.Env = append(append([]string(nil), baseEnv...),
		"CGO_ENABLED=1", "CC=/usr/bin/clang", "CXX=/usr/bin/clang++",
		"CGO_CPPFLAGS=-I"+filepath.Join(icuPrefix, "include"),
		"CGO_LDFLAGS=-L"+filepath.Join(icuPrefix, "lib"),
		"DOCKER_HOST=unix:///var/run/docker.sock",
		"TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock",
		"TESTCONTAINERS_RYUK_DISABLED=true",
	)
	var testOutput bytes.Buffer
	test.Stdout, test.Stderr = io.MultiWriter(stdout, &testOutput), stderr
	if err := test.Run(); err != nil {
		return "", "", cleanup, fmt.Errorf("patched Beads conformance: %w", err)
	}
	if err := verifyAtomicityTestResults(testOutput.Bytes()); err != nil {
		return "", "", cleanup, err
	}
	binary := filepath.Join(temporary, "bd-w001-bootstrap")
	build := exec.Command(goBinary, "build", "-trimpath", "-buildvcs=false", "-o", binary, "./cmd/bd")
	build.Dir = checkout
	build.Env = append(append([]string(nil), baseEnv...),
		"CGO_ENABLED=1", "CC=/usr/bin/clang", "CXX=/usr/bin/clang++",
		"CGO_CPPFLAGS=-I"+filepath.Join(icuPrefix, "include"),
		"CGO_LDFLAGS=-L"+filepath.Join(icuPrefix, "lib"),
	)
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

func verifiedGoEnvironment(config doctrine.W001BootstrapGrant, temporary string) (string, []string, error) {
	goBinary, err := exec.LookPath("go")
	if err != nil {
		return "", nil, errors.New("pinned Go executable is unavailable")
	}
	goBinary, err = filepath.EvalSymlinks(goBinary)
	if err != nil || !filepath.IsAbs(goBinary) {
		return "", nil, errors.New("pinned Go executable cannot be resolved")
	}
	if binaryDigest, err := fileDigest(goBinary); err != nil || binaryDigest != config.GoBinarySHA256 {
		return "", nil, errors.New("Go executable does not match the signed SHA-256")
	}
	moduleCommand := exec.Command(goBinary, "env", "GOMODCACHE")
	moduleCommand.Env = []string{"HOME=" + os.Getenv("HOME"), "PATH=" + filepath.Dir(goBinary) + ":/usr/bin:/bin", "GOENV=off", "GOWORK=off", "GOTOOLCHAIN=local"}
	moduleOutput, err := moduleCommand.Output()
	moduleCache := strings.TrimSpace(string(moduleOutput))
	if err != nil || !filepath.IsAbs(moduleCache) {
		return "", nil, errors.New("local Go module cache cannot be resolved")
	}
	resolvedModuleCache, err := filepath.EvalSymlinks(moduleCache)
	if err != nil || !filepath.IsAbs(resolvedModuleCache) {
		return "", nil, errors.New("local Go module cache cannot be resolved without indirection")
	}
	home := filepath.Join(temporary, "home")
	cache := filepath.Join(temporary, "gocache")
	for _, path := range []string{home, cache} {
		if err := os.MkdirAll(path, 0o700); err != nil {
			return "", nil, err
		}
	}
	environment := []string{
		"HOME=" + home,
		"PATH=" + filepath.Dir(goBinary) + ":/usr/bin:/bin:/opt/homebrew/bin",
		"TMPDIR=" + temporary,
		"GOCACHE=" + cache,
		"GOMODCACHE=" + resolvedModuleCache,
		"GOPATH=" + filepath.Join(temporary, "gopath"),
		"GOENV=off",
		"GOFLAGS=-mod=readonly",
		"GOWORK=off",
		"GOTOOLCHAIN=local",
		"GOPROXY=off",
		"GOSUMDB=off",
	}
	return goBinary, environment, nil
}

func verifyDisposableWorkspace(patchedBinary, originalBinary, workspace string, canonical issueRecord, config doctrine.W001BootstrapGrant, stdout, stderr io.Writer) (string, error) {
	resolved, err := filepath.EvalSymlinks(workspace)
	info, statErr := os.Lstat(workspace)
	if err != nil || statErr != nil || !info.IsDir() || resolved == "" {
		return "", errors.New("canonical Beads workspace must be one resolved directory")
	}
	temporary, err := os.MkdirTemp("", "mars3-w001-disposable-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(temporary)
	copyRoot := filepath.Join(temporary, "workspace")
	if err := run(stdout, stderr, "/bin/cp", "-R", resolved, copyRoot); err != nil {
		return "", fmt.Errorf("copy canonical workspace for conformance: %w", err)
	}
	if _, err := verifyWorkspace(copyRoot, config); err != nil {
		return "", err
	}
	disposablePre, err := readIssue(originalBinary, copyRoot, config.Bead)
	if err != nil || verifyPreimage(disposablePre, config) != nil {
		return "", errors.New("disposable workspace does not preserve the signed preimage")
	}
	if digestIssue(disposablePre) != digestIssue(canonical) {
		return "", errors.New("disposable and canonical issue preimages differ")
	}
	backend, err := readBackendMode(originalBinary, copyRoot)
	if err != nil || backend != "embedded" {
		return "", errors.New("disposable workspace did not select the canonical embedded backend")
	}
	beforeVersion, err := readVersionState(originalBinary, copyRoot)
	if err != nil {
		return "", err
	}
	script, err := claimScript(disposablePre, config)
	if err != nil {
		return "", err
	}
	command := exec.Command(patchedBinary, "-C", ".", "--actor", config.Assignee, "--json", "batch", "--message", "MARS-3 W-001 disposable bootstrap claim")
	command.Dir = copyRoot
	command.Env = beadsCommandEnvironment()
	command.Stdin, command.Stdout, command.Stderr = strings.NewReader(script), stdout, stderr
	if err := command.Run(); err != nil {
		return "", fmt.Errorf("disposable canonical-backend claim: %w", err)
	}
	post, err := readIssue(originalBinary, copyRoot, config.Bead)
	if err != nil {
		return "", fmt.Errorf("read disposable claim postimage: %w", err)
	}
	if err := verifyPostimage(post, config); err != nil {
		return "", fmt.Errorf("disposable claim postimage is invalid: %w", err)
	}
	afterVersion, err := readVersionState(originalBinary, copyRoot)
	if err != nil || afterVersion.Commit == beforeVersion.Commit {
		return "", errors.New("disposable claim did not publish one Dolt version commit")
	}
	return backend, nil
}

func verifyConformanceDependencies(config doctrine.W001BootstrapGrant) (string, error) {
	formula, err := exec.Command("brew", "list", "--versions", "icu4c@78").Output()
	if err != nil || strings.TrimSpace(string(formula)) != config.ICUFormula {
		return "", errors.New("local ICU formula does not match the signed conformance toolchain")
	}
	prefix, err := exec.Command("brew", "--prefix", "icu4c@78").Output()
	icuPrefix := strings.TrimSpace(string(prefix))
	if err != nil || !filepath.IsAbs(icuPrefix) {
		return "", errors.New("local ICU prefix cannot be resolved")
	}
	resolvedPrefix, err := filepath.EvalSymlinks(icuPrefix)
	if err != nil || !filepath.IsAbs(resolvedPrefix) {
		return "", errors.New("local ICU installation root cannot be resolved")
	}
	if filepath.Base(resolvedPrefix) != "78.2" || filepath.Base(filepath.Dir(resolvedPrefix)) != "icu4c@78" {
		return "", errors.New("local ICU installation root does not match the signed versioned formula")
	}
	for _, path := range []string{filepath.Join(icuPrefix, "include", "unicode", "regex.h"), filepath.Join(icuPrefix, "lib", "libicuuc.dylib")} {
		resolved, resolveErr := filepath.EvalSymlinks(path)
		info, statErr := os.Stat(path)
		if resolveErr != nil || statErr != nil || !info.Mode().IsRegular() ||
			!strings.HasPrefix(resolved, filepath.Clean(resolvedPrefix)+string(filepath.Separator)) {
			return "", errors.New("local ICU conformance material is missing or indirect")
		}
	}
	for _, expected := range []string{config.DoltTestImage} {
		output, inspectErr := exec.Command("docker", "image", "inspect", expected, "--format", "{{json .RepoDigests}}").Output()
		if inspectErr != nil || !bytes.Contains(output, []byte(`"`+expected+`"`)) {
			return "", fmt.Errorf("pinned disposable test image is unavailable: %s", expected)
		}
	}
	return icuPrefix, nil
}

func verifyAtomicityTestResults(output []byte) error {
	wanted := map[string]bool{
		"TestBatchBootstrapClaimIsOneAtomicTransition":        false,
		"TestBatchBootstrapClaimPreconditionFailureRollsBack": false,
		"TestBatchBootstrapClaimPostClaimFailureRollsBack":    false,
		"TestBatchBootstrapClaimContentionHasOneWinner":       false,
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
	command.Env = beadsCommandEnvironment()
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

func readBackendMode(binary, workspace string) (string, error) {
	command := exec.Command(binary, "-C", workspace, "dolt", "status", "--json")
	command.Env = beadsCommandEnvironment()
	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("read Beads backend mode: %w", err)
	}
	var status struct {
		Mode string `json:"mode"`
	}
	if json.Unmarshal(output, &status) != nil || status.Mode == "" {
		return "", errors.New("Beads backend mode is invalid")
	}
	return status.Mode, nil
}

func readVersionState(binary, workspace string) (versionState, error) {
	command := exec.Command(binary, "-C", workspace, "vc", "status", "--json")
	command.Env = beadsCommandEnvironment()
	output, err := command.Output()
	if err != nil {
		return versionState{}, fmt.Errorf("read Beads version state: %w", err)
	}
	var state versionState
	if json.Unmarshal(output, &state) != nil || state.Branch != "main" || state.Commit == "" {
		return versionState{}, errors.New("Beads version state is invalid or not on main")
	}
	return state, nil
}

func digestIssue(issue issueRecord) string {
	encoded, _ := json.Marshal(issue)
	return digest(encoded)
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
	if digestLabels(issue.Labels) != config.ExpectedLabelsSHA256 {
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
	if digestLabels(issue.Labels) != config.PostLabelsSHA256 {
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

func digestLabels(values []string) string {
	labels := append([]string(nil), values...)
	sort.Strings(labels)
	encoded, _ := json.Marshal(labels)
	return digest(encoded)
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
	prefix := []string{"-c", "core.fsmonitor=false", "-c", "diff.external=", "-C", repo}
	command := exec.Command(bootstrapGitExecutable, append(prefix, args...)...)
	command.Env = bootstrapGitEnvironment()
	output, err := command.Output()
	return string(output), err
}

func gitRemoteMain() (string, error) {
	directory, err := os.MkdirTemp("", "mars3-w001-remote-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(directory)
	command := exec.Command(bootstrapGitExecutable, "ls-remote", "--exit-code", canonicalRepositoryURL, "refs/heads/main")
	command.Dir = directory
	command.Env = bootstrapGitEnvironment()
	output, err := command.Output()
	if err != nil {
		return "", errors.New("authenticated GitHub main cannot be resolved")
	}
	return parseRemoteMain(output)
}

func parseRemoteMain(output []byte) (string, error) {
	fields := strings.Fields(string(output))
	if len(fields) != 2 || fields[1] != "refs/heads/main" || !isLowerHex(fields[0], 40) {
		return "", errors.New("authenticated GitHub main cannot be resolved")
	}
	return fields[0], nil
}

func isLowerHex(value string, length int) bool {
	if len(value) != length {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			if character < 'a' || character > 'f' {
				return false
			}
		}
	}
	return true
}

func bootstrapGitEnvironment() []string {
	environment := make([]string, 0, len(os.Environ())+8)
	for _, value := range os.Environ() {
		key := value
		if separator := strings.IndexByte(value, '='); separator >= 0 {
			key = value[:separator]
		}
		upper := strings.ToUpper(key)
		if strings.HasPrefix(upper, "GIT_") || strings.HasPrefix(upper, "SSH_") || strings.HasPrefix(upper, "GITLAB_") ||
			(strings.HasPrefix(upper, "GITHUB_") && upper != "GITHUB_ACTIONS") ||
			upper == "HTTP_PROXY" || upper == "HTTPS_PROXY" || upper == "ALL_PROXY" || upper == "NO_PROXY" ||
			upper == "SSL_CERT_FILE" || upper == "SSL_CERT_DIR" || upper == "CURL_CA_BUNDLE" || upper == "XDG_CONFIG_HOME" {
			continue
		}
		environment = append(environment, value)
	}
	return append(environment,
		"GIT_ATTR_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL="+os.DevNull,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_NO_REPLACE_OBJECTS=1",
		"GIT_OPTIONAL_LOCKS=0",
		"GIT_TERMINAL_PROMPT=0",
		"LC_ALL=C",
	)
}

func beadsCommandEnvironment() []string {
	environment := make([]string, 0, len(os.Environ()))
	for _, value := range os.Environ() {
		key := value
		if separator := strings.IndexByte(value, '='); separator >= 0 {
			key = value[:separator]
		}
		upper := strings.ToUpper(key)
		if strings.HasPrefix(upper, "BEADS_") || strings.HasPrefix(upper, "DOLT_") || strings.HasPrefix(upper, "BD_") || strings.HasPrefix(upper, "GT_") {
			continue
		}
		environment = append(environment, value)
	}
	return environment
}

func run(stdout, stderr io.Writer, name string, args ...string) error {
	command := exec.Command(name, args...)
	command.Stdout, command.Stderr = stdout, stderr
	return command.Run()
}
