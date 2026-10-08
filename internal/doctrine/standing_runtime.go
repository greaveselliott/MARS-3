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
	"strings"
	"time"
)

const standingRuntimeSourcePath = ".harness/grants/standing-delivery-runtime-source-v1.yaml"

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
		reviewTag:     "mars3/standing-delivery-runtime-v3",
		reviewMessage: "MARS-3 standing delivery runtime source attestation v3",
		findingPrefix: "public.standing_runtime_",
		issued:        time.Date(2026, 10, 7, 23, 32, 36, 0, time.UTC),
		expires:       time.Date(2026, 10, 14, 23, 32, 36, 0, time.UTC),
		paths: []string{
			standingRuntimeSourcePath, standingRuntimeSourcePath + ".sig",
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
	return nil
}

func standingRuntimeRejectedTagObjectValid(object string) bool {
	return strings.TrimSpace(object) == "6a91a2d0600e2634aa53cf932d3b7ed3f28c5b89"
}

func standingRuntimeRejectedV2TagObjectValid(object string) bool {
	return strings.TrimSpace(object) == "099d8dbe63e46b0c479f3d4475126b1c2fa46612"
}
