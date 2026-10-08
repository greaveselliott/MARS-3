/*
FactoryDocSync:
docs:
- docs/features/F-001-doctrine-foundation.md
- docs/design-docs/mars-provenance.md
- docs/code-documentation-map.md
- docs/features/F-002-work-authority.md
- docs/design-docs/ADR-008-standing-delivery-delegation.md
*/

package doctrine

import (
	"errors"
	"os"
	"strings"
	"time"
)

const standingRuntimeSourcePath = ".harness/grants/standing-delivery-runtime-source-v1.yaml"
const standingRuntimeTestRecoveryPath = ".harness/grants/standing-delivery-runtime-test-recovery-v1.yaml"

// Operational, activation and identity signatures are deliberately distinct
// from the non-operational v1 contract and every source-publication signature.
func VerifyStandingRuntimeSignature(document, signature []byte) error {
	return verifyStandingRuntimeSignature(document, signature, "mars3-standing-delivery-runtime-v2")
}

func VerifyStandingActivationSignature(document, signature []byte) error {
	return verifyStandingRuntimeSignature(document, signature, "mars3-standing-delivery-activation-v2")
}

func VerifyStandingSessionSignature(document, signature []byte) error {
	return verifyStandingRuntimeSignature(document, signature, "mars3-standing-delivery-session-v2")
}

func verifyStandingRuntimeSignature(document, signature []byte, namespace string) error {
	if len(document) == 0 || len(document) > 65536 || len(signature) == 0 || len(signature) > 4096 {
		return errors.New("standing runtime signature denied")
	}
	publicKey := []byte("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGEG9tVIYixGqx/Kl4Ag53SbGWnIMj7HIrHs0+0EYaMl")
	if verifySSHSig(document, signature, publicKey, namespace) != nil {
		return errors.New("standing runtime signature denied")
	}
	return nil
}

func standingRuntimePublication() operatorPublicationSpec {
	return operatorPublicationSpec{
		grantPath:     standingRuntimeSourcePath,
		digest:        "714a24e598997b5896b0619a154e9ae8e198219b1810d4a46bb511b583dc37c0",
		namespace:     "mars3-standing-delivery-runtime-source-v1",
		base:          "880af5ddbc40d09ebe45af2cf0aec5b8a7286193",
		baseTree:      "7849d50fedcd8fc8794f5c7617f46a9ed3ae77f8",
		branch:        "codex/standing-delivery-runtime",
		reviewTag:     "mars3/standing-delivery-runtime-v4",
		reviewMessage: "MARS-3 standing delivery runtime source attestation v4",
		findingPrefix: "public.standing_runtime_",
		issued:        time.Date(2026, 10, 7, 23, 32, 36, 0, time.UTC),
		expires:       time.Date(2026, 10, 14, 23, 32, 36, 0, time.UTC),
		paths: []string{
			standingRuntimeSourcePath, standingRuntimeSourcePath + ".sig",
			standingRuntimeTestRecoveryPath, standingRuntimeTestRecoveryPath + ".sig",
			"internal/authority/operator/delegated.go", "internal/authority/operator/delegated_test.go",
			"internal/authority/operator/operator.go", "internal/authority/operator/operator_test.go",
			"internal/doctrine/grant.go", "internal/doctrine/standing_runtime.go", "internal/doctrine/standing_runtime_test.go",
			"cmd/mars3-authority/main.go", "cmd/mars3-authority/main_test.go",
			"docs/product-decisions/PD-005-finite-plan-standing-delivery.md", "docs/product-decisions/index.md",
			"docs/design-docs/ADR-008-standing-delivery-delegation.md", "docs/design-docs/ADR-001-git-beads-authority.md",
			"docs/features/F-002-work-authority.md", "docs/exec-plans/active/current-operating-plan.md",
			"docs/evidence/standing-delivery-runtime.md",
		},
		retainedHistory: standingRuntimeRetainedHistory,
	}
}

func standingRuntimeRetainedHistory(root string, publicKey []byte) error {
	if err := standingDeliveryTransitionRetainedHistory(root, publicKey); err != nil {
		return err
	}
	previous := standingDeliveryTransitionPublication(root)
	object, err := planningGrantGitOutput(root, "rev-parse", "--verify", "refs/tags/"+previous.reviewTag+"^{tag}")
	if err != nil || strings.TrimSpace(string(object)) != "12d832740cae5a32331b24089e10528d417ed4af" {
		return errors.New("accepted standing foundation tag changed or missing")
	}
	target, err := operatorPublicationReviewTarget(root, publicKey, previous)
	if err != nil || target != "bc4a1d32257ca7750c4538926225d5b8b34be53d" {
		return errors.New("accepted standing foundation target invalid")
	}
	object, err = planningGrantGitOutput(root, "cat-file", "commit", target)
	if err != nil || verifyPlanningGrantCommit(object, publicKey) != nil {
		return errors.New("accepted standing foundation signature invalid")
	}
	rejected := standingRuntimePublication()
	rejected.reviewTag = "mars3/standing-delivery-runtime-v1"
	rejected.reviewMessage = "MARS-3 standing delivery runtime source attestation v1"
	object, err = planningGrantGitOutput(root, "rev-parse", "--verify", "refs/tags/"+rejected.reviewTag+"^{tag}")
	if err != nil || !standingRuntimeRejectedTagObjectValid(string(object)) {
		return errors.New("rejected runtime tag changed or missing")
	}
	target, err = operatorPublicationReviewTarget(root, publicKey, rejected)
	if err != nil || target != "031a792970452fb859fee7dce65e7bc37d133420" {
		return errors.New("rejected runtime target invalid")
	}
	object, err = planningGrantGitOutput(root, "cat-file", "commit", target)
	if err != nil || verifyPlanningGrantCommit(object, publicKey) != nil {
		return errors.New("rejected runtime candidate signature invalid")
	}
	rejected.reviewTag = "mars3/standing-delivery-runtime-v2"
	rejected.reviewMessage = "MARS-3 standing delivery runtime source attestation v2"
	object, err = planningGrantGitOutput(root, "rev-parse", "--verify", "refs/tags/"+rejected.reviewTag+"^{tag}")
	if err != nil || !standingRuntimeRejectedV2TagObjectValid(string(object)) {
		return errors.New("rejected runtime v2 tag changed or missing")
	}
	target, err = operatorPublicationReviewTarget(root, publicKey, rejected)
	if err != nil || target != "7dbe1b718e585d43458a3e91f622481490ef70f4" {
		return errors.New("rejected runtime v2 target invalid")
	}
	object, err = planningGrantGitOutput(root, "cat-file", "commit", target)
	if err != nil || verifyPlanningGrantCommit(object, publicKey) != nil {
		return errors.New("rejected runtime v2 candidate signature invalid")
	}
	return standingRuntimeTestRecoveryHistory(root, publicKey)
}

func standingRuntimeRejectedTagObjectValid(object string) bool {
	return strings.TrimSpace(object) == "6a91a2d0600e2634aa53cf932d3b7ed3f28c5b89"
}

func standingRuntimeRejectedV2TagObjectValid(object string) bool {
	return strings.TrimSpace(object) == "099d8dbe63e46b0c479f3d4475126b1c2fa46612"
}

// The original grant fences aggregate publication; this additional prospective
// grant fences only the post-rejection delta and excludes all runtime sources.
func standingRuntimeTestRecoverySpec() operatorPublicationSpec {
	return operatorPublicationSpec{
		grantPath: standingRuntimeTestRecoveryPath,
		digest:    "bf4a29d4b1bbf4e206f5f0df6e8826654af218306fd73a6e7955be6ea420b3ec",
		namespace: "mars3-standing-delivery-runtime-test-recovery-v1",
		base:      "07c3930b1c074c8dbce5b943a8363e9a645859a4", baseTree: "9545c0dcbc98aeca0a202dd0efb54205a020ecfe",
		issued: time.Date(2026, 10, 8, 5, 31, 22, 0, time.UTC), expires: time.Date(2026, 10, 14, 23, 32, 36, 0, time.UTC),
		paths: []string{standingRuntimeTestRecoveryPath, standingRuntimeTestRecoveryPath + ".sig",
			"internal/authority/operator/delegated_test.go", "internal/doctrine/standing_runtime.go", "internal/doctrine/standing_runtime_test.go",
			"docs/features/F-002-work-authority.md", "docs/design-docs/ADR-008-standing-delivery-delegation.md",
			"docs/exec-plans/active/current-operating-plan.md", "docs/evidence/standing-delivery-runtime.md"},
	}
}

func standingRuntimeTestRecoveryHistory(root string, key []byte) error {
	spec := standingRuntimeTestRecoverySpec()
	document, err := readRepoFile(root, spec.grantPath)
	signature, sigErr := readRepoFile(root, spec.grantPath+".sig")
	if err != nil || sigErr != nil || !spec.documentValid(document, signature, key) {
		return errors.New("test recovery requires its exact signed prospective grant")
	}
	rejected := standingRuntimePublication()
	rejected.reviewTag = "mars3/standing-delivery-runtime-v3"
	rejected.reviewMessage = "MARS-3 standing delivery runtime source attestation v3"
	object, err := planningGrantGitOutput(root, "rev-parse", "--verify", "refs/tags/"+rejected.reviewTag+"^{tag}")
	if err != nil || strings.TrimSpace(string(object)) != "a8a3cb9fcffab0a6ec7eb5246d4d658dbcad3013" {
		return errors.New("rejected v3 tag changed")
	}
	target, err := operatorPublicationReviewTarget(root, key, rejected)
	if err != nil || target != spec.base {
		return errors.New("rejected v3 target changed")
	}
	object, err = planningGrantGitOutput(root, "cat-file", "commit", target)
	if err != nil || verifyPlanningGrantCommit(object, key) != nil {
		return errors.New("rejected v3 signature invalid")
	}
	tree, err := planningGrantGitOutput(root, "rev-parse", "--verify", spec.base+"^{tree}")
	if err != nil || strings.TrimSpace(string(tree)) != spec.baseTree {
		return errors.New("test recovery preimage changed")
	}
	headBytes, err := planningGrantGitOutput(root, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return err
	}
	head := strings.TrimSpace(string(headBytes))
	branch, _ := planningGrantGitOutput(root, "symbolic-ref", "--quiet", "--short", "HEAD")
	if os.Getenv("GITHUB_ACTIONS") == "true" || strings.TrimSpace(string(branch)) == "main" {
		head, err = operatorPublicationReviewTarget(root, key, standingRuntimePublication())
		if err != nil {
			return err
		}
	}
	if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", spec.base, head); err != nil {
		return err
	}
	commits, err := planningGrantCommitRangeFrom(root, spec.base, head)
	if err != nil {
		return err
	}
	previous := spec.base
	for _, commit := range commits {
		changed, diffErr := planningGrantGitOutput(root, "diff-tree", "--no-commit-id", "--no-renames", "--no-ext-diff", "--no-textconv", "--name-only", "-z", "-r", previous, commit.id)
		paths, pathsErr := normalizedPlanningGrantGitPaths(changed)
		object, objectErr := planningGrantGitOutput(root, "cat-file", "commit", commit.id)
		committedAt, timeErr := planningGrantCommitTime(root, commit.id)
		grant, grantErr := planningGrantGitOutput(root, "show", commit.id+":"+spec.grantPath)
		sig, sigErr := planningGrantGitOutput(root, "show", commit.id+":"+spec.grantPath+".sig")
		if len(commit.parents) != 1 || commit.parents[0] != previous || diffErr != nil || pathsErr != nil || !spec.pathsAllowed(paths) || objectErr != nil || verifyPlanningGrantCommit(object, key) != nil || timeErr != nil || !spec.windowValid(committedAt) || grantErr != nil || sigErr != nil || !spec.documentValid(grant, sig, key) {
			return errors.New("test recovery must be prospective, signed, exact-scope and runtime-frozen")
		}
		previous = commit.id
	}
	tracked, trackedErr := planningGrantGitOutput(root, "diff", "--no-renames", "--no-ext-diff", "--no-textconv", "--name-only", "-z", "HEAD", "--")
	untracked, untrackedErr := planningGrantGitOutput(root, "ls-files", "--others", "--exclude-standard", "-z", "--")
	paths, pathsErr := normalizedPlanningGrantGitPaths(tracked, untracked)
	if trackedErr != nil || untrackedErr != nil || pathsErr != nil || !spec.pathsAllowed(paths) {
		return errors.New("test recovery worktree exceeds exact delta scope")
	}
	return nil
}
