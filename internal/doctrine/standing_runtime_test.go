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
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStandingRuntimeSourceGrantIsPinnedAndDistinctFromOperationalSignatures(t *testing.T) {
	spec := standingRuntimePublication()
	root := filepath.Join("..", "..")
	document, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(spec.grantPath)))
	if err != nil {
		t.Fatal(err)
	}
	signature, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(spec.grantPath+".sig")))
	if err != nil {
		t.Fatal(err)
	}
	key, err := os.ReadFile(filepath.Join(root, ".harness", "keys", "genesis-signing-key.pub"))
	if err != nil {
		t.Fatal(err)
	}
	if !spec.documentValid(document, signature, key) {
		t.Fatal("source grant digest/signature is not its pinned authority")
	}
	for _, verifier := range []func([]byte, []byte) error{VerifyStandingRuntimeSignature, VerifyStandingActivationSignature, VerifyStandingSessionSignature, VerifyStandingDeliverySignature} {
		if verifier(document, signature) == nil {
			t.Fatal("source signature substituted for runtime/activation/session/v1 authority")
		}
	}
	if spec.documentValid(append(append([]byte{}, document...), '\n'), signature, key) {
		t.Fatal("altered source grant admitted")
	}
	if len(spec.paths) != 18 || spec.pathsAllowed([]string{"internal/authority/gateway/lifecycle.go"}) || spec.pathsAllowed([]string{"outside.go"}) {
		t.Fatal("runtime source paths expanded")
	}
}

func TestStandingRuntimePublicationWindowHasImmutableDates(t *testing.T) {
	spec := standingRuntimePublication()
	issued := time.Date(2026, 10, 7, 23, 32, 36, 0, time.UTC)
	expires := time.Date(2026, 10, 14, 23, 32, 36, 0, time.UTC)
	if !spec.issued.Equal(issued) || !spec.expires.Equal(expires) || spec.windowValid(issued.Add(-time.Nanosecond)) || !spec.windowValid(issued) ||
		!spec.windowValid(expires.Add(-time.Nanosecond)) || spec.windowValid(expires) || spec.windowValid(expires.AddDate(1, 0, 0)) {
		t.Fatal("signed source window widened")
	}
	if spec.base != "880af5ddbc40d09ebe45af2cf0aec5b8a7286193" || spec.reviewTag == standingDeliveryTransitionPublication("").reviewTag {
		t.Fatal("runtime transition lost accepted base or distinct tag")
	}
}

func TestStandingRuntimeRetainsRejectedCandidateAndDistinctSuccessorTag(t *testing.T) {
	spec := standingRuntimePublication()
	if spec.reviewTag != "mars3/standing-delivery-runtime-v3" || spec.reviewMessage != "MARS-3 standing delivery runtime source attestation v3" {
		t.Fatal("corrected runtime candidate reused rejected attestation")
	}
	if !standingRuntimeRejectedTagObjectValid("6a91a2d0600e2634aa53cf932d3b7ed3f28c5b89\n") ||
		standingRuntimeRejectedTagObjectValid("0000000000000000000000000000000000000000") {
		t.Fatal("rejected runtime evidence can be replaced")
	}
	if !standingRuntimeRejectedV2TagObjectValid("099d8dbe63e46b0c479f3d4475126b1c2fa46612\n") ||
		standingRuntimeRejectedV2TagObjectValid("0000000000000000000000000000000000000000") {
		t.Fatal("rejected runtime v2 evidence can be replaced")
	}
}
