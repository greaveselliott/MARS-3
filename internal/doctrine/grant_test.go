/*
FactoryDocSync:
docs:
- docs/features/F-001-doctrine-foundation.md
- docs/features/F-002-work-authority.md
- docs/design-docs/ADR-001-git-beads-authority.md
- docs/design-docs/mars-provenance.md
- docs/code-documentation-map.md
*/

package doctrine

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

const wave1PlanningGrantFirstCommitFixture = "fc9f6641d0f739a401a4f7be3bc0ee575df1310a"

func TestW001BootstrapGrantAcceptsPinnedSignedContract(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	var findings []Finding
	checkW001BootstrapGrant(repo, &findings)
	if len(findings) != 0 {
		t.Fatalf("valid signed W-001 bootstrap grant was rejected: %v", findings)
	}
}

func TestW001BootstrapGrantRejectsTampering(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	read := func(path string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	grant := read(w001BootstrapGrantPath)
	signature := read(w001BootstrapGrantSignature)
	publicKey := read(wave1PlanningGrantKey)
	materials := map[string][]byte{
		scalarPathFromGrant(t, grant, "patchPath"):         nil,
		scalarPathFromGrant(t, grant, "helperCommandPath"): nil,
		scalarPathFromGrant(t, grant, "helperLibraryPath"): nil,
	}
	for path := range materials {
		materials[path] = read(path)
	}

	for _, testCase := range []struct {
		name string
		old  string
		new  string
		code string
	}{
		{name: "claim_state", old: "claimState: unclaimed", new: "claimState: claimed", code: "public.w001_bootstrap_value"},
		{name: "scope", old: "    - internal/authority/bootstrap/bootstrap_test.go", new: "    - internal/runtime/escape.go", code: "public.w001_bootstrap_sequence"},
		{name: "dependency_type", old: "dependencyType: blocks", new: "dependencyType: related", code: "public.w001_bootstrap_value"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			root := writeW001BootstrapGrantFixture(t, bytes.Replace(grant, []byte(testCase.old), []byte(testCase.new), 1), signature, publicKey, materials)
			var findings []Finding
			checkW001BootstrapGrant(root, &findings)
			if !findingCodePresent(findings, testCase.code) || !findingCodePresent(findings, "public.w001_bootstrap_signature") {
				t.Fatalf("tampered bootstrap grant was not rejected by contract and signature: %v", findings)
			}
		})
	}

	t.Run("helper_bytes", func(t *testing.T) {
		tamperedMaterials := make(map[string][]byte, len(materials))
		for path, data := range materials {
			tamperedMaterials[path] = append([]byte(nil), data...)
		}
		path := scalarPathFromGrant(t, grant, "helperLibraryPath")
		tamperedMaterials[path] = append(tamperedMaterials[path], []byte("\n// unauthorized drift\n")...)
		root := writeW001BootstrapGrantFixture(t, grant, signature, publicKey, tamperedMaterials)
		var findings []Finding
		checkW001BootstrapGrant(root, &findings)
		if !findingCodePresent(findings, "public.w001_bootstrap_helper_digest") {
			t.Fatalf("tampered helper bytes were not rejected: %v", findings)
		}
	})
}

func TestW001ExecutionAuthorizationFailsClosedWithoutPinnedSignature(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	grant, err := LoadW001BootstrapGrant(repo)
	if err != nil {
		t.Skipf("bootstrap grant is being re-signed in this fixture: %v", err)
	}
	issuedAt := time.Now().UTC().Truncate(time.Second)
	authorization := W001BootstrapExecutionAuthorization{
		SchemaVersion: 1, Kind: "MARS3W001BootstrapExecutionAuthorization", Classification: "PUBLIC",
		GrantID: grant.ID, Repository: planningGrantRepository, AttemptID: grant.AttemptID, IdempotencyKey: grant.IdempotencyKey,
		Bead: grant.Bead, AuthorityProjectID: grant.AuthorityProjectID,
		MergedCommit: strings.Repeat("1", 40), MergedTree: strings.Repeat("2", 40), ReviewTag: grant.ReviewTag,
		ReviewedFeatureCommit: strings.Repeat("3", 40), PullRequest: 6, ProtectedMainCheckRun: 1,
		QAReviewedCommit: strings.Repeat("3", 40), QADisposition: "accepted",
		SecurityReviewedCommit: strings.Repeat("3", 40), SecurityDisposition: "accepted",
		PatchedBinarySHA256: grant.PatchedBinarySHA256, ExpectedMetadataSHA256: grant.ExpectedMetadataSHA256,
		WorkspaceInstanceSHA256: strings.Repeat("4", 64),
		AllowedEffect:           "execute-one-expected-preimage-W-001-CAS-claim",
		IssuedAt:                issuedAt.Format(time.RFC3339), ExpiresAt: issuedAt.Add(time.Hour).Format(time.RFC3339),
	}
	data, err := json.Marshal(authorization)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(planningGrantCanonicalTempDir(t), "execution.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".sig", []byte("not a signature\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadW001BootstrapExecutionAuthorization(repo, path, grant); err == nil {
		t.Fatal("unsigned execution authorization was accepted")
	}
}

func TestW001ExecutionAuthorizationRequiresCanonicalSingleObject(t *testing.T) {
	authorization := W001BootstrapExecutionAuthorization{
		SchemaVersion:  1,
		Kind:           "MARS3W001BootstrapExecutionAuthorization",
		Classification: "PUBLIC",
	}
	canonical, err := json.Marshal(authorization)
	if err != nil {
		t.Fatal(err)
	}
	canonical = append(canonical, '\n')
	if _, err := decodeCanonicalW001ExecutionAuthorization(canonical); err != nil {
		t.Fatalf("canonical authorization was rejected: %v", err)
	}
	duplicates := append([]byte(`{"schemaVersion":1,`), canonical[1:]...)
	if _, err := decodeCanonicalW001ExecutionAuthorization(duplicates); err == nil {
		t.Fatal("duplicate authorization key was accepted")
	}
	var indented bytes.Buffer
	if err := json.Indent(&indented, canonical[:len(canonical)-1], "", "  "); err != nil {
		t.Fatal(err)
	}
	indented.WriteByte('\n')
	if _, err := decodeCanonicalW001ExecutionAuthorization(indented.Bytes()); err == nil {
		t.Fatal("non-canonical authorization formatting was accepted")
	}
}

func TestW001ExecutionAuthorizationIdentityIncludesSignedBytes(t *testing.T) {
	first := W001BootstrapExecutionAuthorization{MergedCommit: strings.Repeat("1", 40), payloadSHA256: strings.Repeat("a", 64), signatureSHA256: strings.Repeat("b", 64)}
	second := first
	second.signatureSHA256 = strings.Repeat("c", 64)
	if first == second {
		t.Fatal("distinct detached-signature bytes had equal authorization identity")
	}
	second = first
	second.payloadSHA256 = strings.Repeat("d", 64)
	if first == second {
		t.Fatal("distinct signed-payload bytes had equal authorization identity")
	}
}

func TestW001PostclaimGrantAcceptsPinnedSignedContract(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	var findings []Finding
	checkW001PostclaimGrant(repo, &findings)
	if len(findings) != 0 {
		t.Fatalf("valid signed W-001 postclaim reconciliation grant was rejected: %v", findings)
	}
}

func TestW001PostclaimGrantFailsClosed(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	read := func(path string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	grant := read(w001PostclaimGrantPath)
	signature := read(w001PostclaimGrantSignature)
	publicKey := read(wave1PlanningGrantKey)
	materials := map[string][]byte{
		w001BootstrapGrantPath:                        read(w001BootstrapGrantPath),
		w001BootstrapGrantSignature:                   read(w001BootstrapGrantSignature),
		".harness/manifest.yaml":                      read(".harness/manifest.yaml"),
		canonicalActivePlan:                           read(canonicalActivePlan),
		"docs/evidence/W-001-bootstrap-transition.md": read("docs/evidence/W-001-bootstrap-transition.md"),
	}

	for _, testCase := range []struct {
		name string
		old  string
		new  string
		code string
	}{
		{name: "claim", old: "claimed: true", new: "claimed: false", code: "public.w001_postclaim_value"},
		{name: "lease", old: "implementationAllowed: false", new: "implementationAllowed: true", code: "public.w001_postclaim_value"},
		{name: "scope", old: "    - internal/doctrine/grant_test.go", new: "    - internal/authority/escape.go", code: "public.w001_postclaim_sequence"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			root := writeW001PostclaimGrantFixture(t, bytes.Replace(grant, []byte(testCase.old), []byte(testCase.new), 1), signature, publicKey, materials)
			var findings []Finding
			checkW001PostclaimGrant(root, &findings)
			if !findingCodePresent(findings, testCase.code) || !findingCodePresent(findings, "public.w001_postclaim_signature") {
				t.Fatalf("tampered postclaim grant was not rejected by contract and signature: %v", findings)
			}
		})
	}

	t.Run("evidence postimage", func(t *testing.T) {
		tampered := make(map[string][]byte, len(materials))
		for path, data := range materials {
			tampered[path] = append([]byte(nil), data...)
		}
		const evidence = "docs/evidence/W-001-bootstrap-transition.md"
		tampered[evidence] = append(tampered[evidence], []byte("\nunauthorized receipt drift\n")...)
		root := writeW001PostclaimGrantFixture(t, grant, signature, publicKey, tampered)
		var findings []Finding
		checkW001PostclaimGrant(root, &findings)
		if !findingCodePresent(findings, "public.w001_postclaim_postimage") {
			t.Fatalf("tampered evidence postimage was accepted: %v", findings)
		}
	})

	t.Run("duplicate receipt fence", func(t *testing.T) {
		tampered := make(map[string][]byte, len(materials))
		for path, data := range materials {
			tampered[path] = append([]byte(nil), data...)
		}
		const evidence = "docs/evidence/W-001-bootstrap-transition.md"
		const receipt = "- Receipt SHA-256: `04cef4e421a34e0908d392fc794181db3ddb754a134e34599fa41a520c78d126`.\n"
		tampered[evidence] = bytes.Replace(tampered[evidence], []byte(receipt), []byte(receipt+receipt), 1)
		root := writeW001PostclaimGrantFixture(t, grant, signature, publicKey, tampered)
		var findings []Finding
		checkW001PostclaimGrant(root, &findings)
		if !findingCodePresent(findings, "public.w001_postclaim_receipt") {
			t.Fatalf("duplicated claim receipt was accepted: %v", findings)
		}
	})
}

func writeW001PostclaimGrantFixture(t *testing.T, grant, signature, publicKey []byte, materials map[string][]byte) string {
	t.Helper()
	root := planningGrantCanonicalTempDir(t)
	writePlanningGrantTestFile(t, root, w001PostclaimGrantPath, grant)
	writePlanningGrantTestFile(t, root, w001PostclaimGrantSignature, signature)
	writePlanningGrantTestFile(t, root, wave1PlanningGrantKey, publicKey)
	for path, data := range materials {
		writePlanningGrantTestFile(t, root, path, data)
	}
	return root
}

func TestW001PostclaimCIFixAcceptsPinnedSignedContract(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	var findings []Finding
	checkW001PostclaimCIFixAddendum(repo, &findings)
	if len(findings) != 0 {
		t.Fatalf("valid signed postclaim CI-stabilization addendum was rejected: %v", findings)
	}
}

func TestW001PostclaimCIFixRejectsTampering(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	read := func(path string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	addendum := read(w001PostclaimCIFixPath)
	signature := read(w001PostclaimCIFixSignature)
	publicKey := read(wave1PlanningGrantKey)
	priorGrant := read(w001PostclaimGrantPath)
	priorSignature := read(w001PostclaimGrantSignature)

	for _, testCase := range []struct {
		name string
		old  string
		new  string
	}{
		{name: "base", old: w001PostclaimCIFixBase, new: strings.Repeat("0", 40)},
		{name: "failure", old: "go-test-fixtures-inherited-github-actions-environment", new: "unrelated-failure"},
		{name: "scope", old: "    - internal/doctrine/grant_test.go", new: "    - internal/authority/escape.go"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			root := planningGrantCanonicalTempDir(t)
			writePlanningGrantTestFile(t, root, w001PostclaimCIFixPath, bytes.Replace(addendum, []byte(testCase.old), []byte(testCase.new), 1))
			writePlanningGrantTestFile(t, root, w001PostclaimCIFixSignature, signature)
			writePlanningGrantTestFile(t, root, wave1PlanningGrantKey, publicKey)
			writePlanningGrantTestFile(t, root, w001PostclaimGrantPath, priorGrant)
			writePlanningGrantTestFile(t, root, w001PostclaimGrantSignature, priorSignature)
			var findings []Finding
			checkW001PostclaimCIFixAddendum(root, &findings)
			if !findingCodePresent(findings, "public.w001_postclaim_ci_signature") {
				t.Fatalf("tampered CI-stabilization addendum retained a valid signature: %v", findings)
			}
		})
	}
}

func TestW001PostclaimSecurityFixAcceptsPinnedSignedContract(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	var findings []Finding
	checkW001PostclaimSecurityFix(repo, &findings)
	if len(findings) != 0 {
		t.Fatalf("valid signed postclaim Security correction was rejected: %v", findings)
	}
	grant, err := LoadW001BootstrapGrant(repo)
	if err != nil {
		t.Fatalf("valid Security correction was not projected into the bootstrap helper: %v", err)
	}
	if grant.PatchSHA256 != w001PostclaimSecurityBasePatchSHA || grant.CorrectionPatchPath != w001PostclaimSecurityPatchPath ||
		grant.CorrectionPatchSHA256 != w001PostclaimSecurityPatchSHA || grant.HookIsolationPatchPath != w001PostclaimHookPatchPath ||
		grant.HookIsolationPatchSHA256 != w001PostclaimHookPatchSHA || grant.PatchedBinarySHA256 != w001PostclaimHookBinarySHA {
		t.Fatalf("bootstrap helper did not preserve the Security correction beneath the signed hook-isolation successor: %+v", grant)
	}
}

func TestW001PostclaimSecurityFixFailsClosed(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	read := func(path string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	grant := read(w001PostclaimSecurityFixPath)
	signature := read(w001PostclaimSecurityFixSig)
	publicKey := read(wave1PlanningGrantKey)
	materials := map[string][]byte{
		w001PostclaimCIFixPath:                                         read(w001PostclaimCIFixPath),
		w001PostclaimCIFixSignature:                                    read(w001PostclaimCIFixSignature),
		"docs/evidence/W-001-validation.md":                            read("docs/evidence/W-001-validation.md"),
		"internal/authority/bootstrap/bootstrap.go":                    read("internal/authority/bootstrap/bootstrap.go"),
		"internal/authority/bootstrap/bootstrap_test.go":               read("internal/authority/bootstrap/bootstrap_test.go"),
		"internal/authority/bootstrap/beads-v1.2.2-atomic-claim.patch": read("internal/authority/bootstrap/beads-v1.2.2-atomic-claim.patch"),
		w001PostclaimSecurityPatchPath:                                 read(w001PostclaimSecurityPatchPath),
	}

	t.Run("signed grant tamper", func(t *testing.T) {
		tampered := bytes.Replace(grant, []byte("canonicalWorkMutationAllowed: false"), []byte("canonicalWorkMutationAllowed: true"), 1)
		root := writeW001PostclaimSecurityFixture(t, tampered, signature, publicKey, materials)
		var findings []Finding
		checkW001PostclaimSecurityFix(root, &findings)
		if !findingCodePresent(findings, "public.w001_postclaim_security_value") || !findingCodePresent(findings, "public.w001_postclaim_security_signature") {
			t.Fatalf("tampered Security-correction grant was accepted: %v", findings)
		}
	})

	t.Run("security patch material tamper", func(t *testing.T) {
		tampered := clonePlanningGrantMaterials(materials)
		tampered[w001PostclaimSecurityPatchPath] = append(tampered[w001PostclaimSecurityPatchPath], []byte("\nunauthorized drift\n")...)
		root := writeW001PostclaimSecurityFixture(t, grant, signature, publicKey, tampered)
		var findings []Finding
		checkW001PostclaimSecurityFix(root, &findings)
		if !findingCodePresent(findings, "public.w001_postclaim_security_material") {
			t.Fatalf("tampered Security-correction patch was accepted: %v", findings)
		}
	})

	t.Run("helper material tamper", func(t *testing.T) {
		tampered := clonePlanningGrantMaterials(materials)
		const path = "internal/authority/bootstrap/bootstrap.go"
		tampered[path] = append(tampered[path], []byte("\n// unauthorized drift\n")...)
		root := writeW001PostclaimSecurityFixture(t, grant, signature, publicKey, tampered)
		var findings []Finding
		checkW001PostclaimSecurityFix(root, &findings)
		if !findingCodePresent(findings, "public.w001_postclaim_security_material") {
			t.Fatalf("tampered Security-correction helper was accepted: %v", findings)
		}
	})

	t.Run("supersession evidence removed", func(t *testing.T) {
		tampered := clonePlanningGrantMaterials(materials)
		const path = "docs/evidence/W-001-validation.md"
		tampered[path] = bytes.Replace(tampered[path], []byte("**Current disposition:** changes-requested"), []byte("**Current disposition:** accepted"), 1)
		root := writeW001PostclaimSecurityFixture(t, grant, signature, publicKey, tampered)
		var findings []Finding
		checkW001PostclaimSecurityFix(root, &findings)
		if !findingCodePresent(findings, "public.w001_postclaim_security_evidence") {
			t.Fatalf("Security supersession evidence was not required: %v", findings)
		}
	})
}

func writeW001PostclaimSecurityFixture(t *testing.T, grant, signature, publicKey []byte, materials map[string][]byte) string {
	t.Helper()
	root := planningGrantCanonicalTempDir(t)
	writePlanningGrantTestFile(t, root, w001PostclaimSecurityFixPath, grant)
	writePlanningGrantTestFile(t, root, w001PostclaimSecurityFixSig, signature)
	writePlanningGrantTestFile(t, root, wave1PlanningGrantKey, publicKey)
	for path, data := range materials {
		writePlanningGrantTestFile(t, root, path, data)
	}
	return root
}

func TestW001PostclaimHookFixAcceptsPinnedSignedContract(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	var findings []Finding
	checkW001PostclaimHookFix(repo, &findings)
	if len(findings) != 0 {
		t.Fatalf("valid signed postclaim hook-isolation correction was rejected: %v", findings)
	}
	grant, err := LoadW001BootstrapGrant(repo)
	if err != nil {
		t.Fatalf("valid hook-isolation correction was not projected into the bootstrap helper: %v", err)
	}
	if grant.PatchSHA256 != w001PostclaimSecurityBasePatchSHA || grant.CorrectionPatchPath != w001PostclaimSecurityPatchPath ||
		grant.CorrectionPatchSHA256 != w001PostclaimSecurityPatchSHA || grant.HookIsolationPatchPath != w001PostclaimHookPatchPath ||
		grant.HookIsolationPatchSHA256 != w001PostclaimHookPatchSHA || grant.PatchedBinarySHA256 != w001PostclaimHookBinarySHA {
		t.Fatalf("bootstrap helper did not bind the exact three-layer hook-isolation correction: %+v", grant)
	}
}

func TestW001PostclaimHookFixFailsClosed(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	read := func(path string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	grant := read(w001PostclaimHookFixPath)
	signature := read(w001PostclaimHookFixSig)
	publicKey := read(wave1PlanningGrantKey)
	materials := map[string][]byte{
		w001PostclaimSecurityFixPath:                                   read(w001PostclaimSecurityFixPath),
		w001PostclaimSecurityFixSig:                                    read(w001PostclaimSecurityFixSig),
		"docs/evidence/W-001-validation.md":                            read("docs/evidence/W-001-validation.md"),
		"docs/evidence/W-001-bootstrap-transition.md":                  read("docs/evidence/W-001-bootstrap-transition.md"),
		"internal/authority/bootstrap/bootstrap.go":                    read("internal/authority/bootstrap/bootstrap.go"),
		"internal/authority/bootstrap/bootstrap_test.go":               read("internal/authority/bootstrap/bootstrap_test.go"),
		"internal/authority/bootstrap/beads-v1.2.2-atomic-claim.patch": read("internal/authority/bootstrap/beads-v1.2.2-atomic-claim.patch"),
		w001PostclaimSecurityPatchPath:                                 read(w001PostclaimSecurityPatchPath),
		w001PostclaimHookPatchPath:                                     read(w001PostclaimHookPatchPath),
	}

	t.Run("signed grant tamper", func(t *testing.T) {
		tampered := bytes.Replace(grant, []byte("canonicalWorkMutationAllowed: false"), []byte("canonicalWorkMutationAllowed: true"), 1)
		root := writeW001PostclaimHookFixture(t, tampered, signature, publicKey, materials)
		var findings []Finding
		checkW001PostclaimHookFix(root, &findings)
		if !findingCodePresent(findings, "public.w001_postclaim_hook_value") || !findingCodePresent(findings, "public.w001_postclaim_hook_signature") {
			t.Fatalf("tampered hook-isolation grant was accepted: %v", findings)
		}
	})

	t.Run("hook patch material tamper", func(t *testing.T) {
		tampered := clonePlanningGrantMaterials(materials)
		tampered[w001PostclaimHookPatchPath] = append(tampered[w001PostclaimHookPatchPath], []byte("\nunauthorized drift\n")...)
		root := writeW001PostclaimHookFixture(t, grant, signature, publicKey, tampered)
		var findings []Finding
		checkW001PostclaimHookFix(root, &findings)
		if !findingCodePresent(findings, "public.w001_postclaim_hook_material") {
			t.Fatalf("tampered hook-isolation patch was accepted: %v", findings)
		}
	})

	t.Run("helper material tamper", func(t *testing.T) {
		tampered := clonePlanningGrantMaterials(materials)
		const path = "internal/authority/bootstrap/bootstrap.go"
		tampered[path] = append(tampered[path], []byte("\n// unauthorized drift\n")...)
		root := writeW001PostclaimHookFixture(t, grant, signature, publicKey, tampered)
		var findings []Finding
		checkW001PostclaimHookFix(root, &findings)
		if !findingCodePresent(findings, "public.w001_postclaim_hook_material") {
			t.Fatalf("tampered hook-isolation helper was accepted: %v", findings)
		}
	})

	t.Run("finding evidence removed", func(t *testing.T) {
		tampered := clonePlanningGrantMaterials(materials)
		const path = "docs/evidence/W-001-validation.md"
		tampered[path] = bytes.Replace(tampered[path], []byte("bootstrap-workspace-hook-postcommit-effect"), []byte("missing-fingerprint"), 1)
		root := writeW001PostclaimHookFixture(t, grant, signature, publicKey, tampered)
		var findings []Finding
		checkW001PostclaimHookFix(root, &findings)
		if !findingCodePresent(findings, "public.w001_postclaim_hook_evidence") {
			t.Fatalf("hook-isolation finding evidence was not required: %v", findings)
		}
	})

	t.Run("canonical status correction removed", func(t *testing.T) {
		tampered := clonePlanningGrantMaterials(materials)
		const path = "docs/evidence/W-001-bootstrap-transition.md"
		tampered[path] = bytes.Replace(tampered[path], []byte("Canonical claim verified and reconciled"), []byte("Canonical claim pending"), 1)
		root := writeW001PostclaimHookFixture(t, grant, signature, publicKey, tampered)
		var findings []Finding
		checkW001PostclaimHookFix(root, &findings)
		if !findingCodePresent(findings, "public.w001_postclaim_hook_evidence") {
			t.Fatalf("truthful canonical status correction was not required: %v", findings)
		}
	})
}

func writeW001PostclaimHookFixture(t *testing.T, grant, signature, publicKey []byte, materials map[string][]byte) string {
	t.Helper()
	root := planningGrantCanonicalTempDir(t)
	writePlanningGrantTestFile(t, root, w001PostclaimHookFixPath, grant)
	writePlanningGrantTestFile(t, root, w001PostclaimHookFixSig, signature)
	writePlanningGrantTestFile(t, root, wave1PlanningGrantKey, publicKey)
	for path, data := range materials {
		writePlanningGrantTestFile(t, root, path, data)
	}
	return root
}

func TestW001PostclaimPRFixAcceptsPinnedSignedContract(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	var findings []Finding
	checkW001PostclaimPRFix(repo, &findings)
	if len(findings) != 0 {
		t.Fatalf("valid signed postclaim publication binding was rejected: %v", findings)
	}
}

func TestW001PostclaimPRFixFailsClosed(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	read := func(path string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	grant := read(w001PostclaimPRFixPath)
	signature := read(w001PostclaimPRFixSig)
	publicKey := read(wave1PlanningGrantKey)
	materials := map[string][]byte{
		w001PostclaimHookFixPath:            read(w001PostclaimHookFixPath),
		w001PostclaimHookFixSig:             read(w001PostclaimHookFixSig),
		"docs/evidence/W-001-validation.md": read("docs/evidence/W-001-validation.md"),
	}

	t.Run("signed active PR tamper", func(t *testing.T) {
		tampered := bytes.Replace(grant, []byte("activePullRequest: 8"), []byte("activePullRequest: 7"), 1)
		root := writeW001PostclaimPRFixFixture(t, tampered, signature, publicKey, materials)
		var findings []Finding
		checkW001PostclaimPRFix(root, &findings)
		if !findingCodePresent(findings, "public.w001_postclaim_pr_binding_value") || !findingCodePresent(findings, "public.w001_postclaim_pr_binding_signature") {
			t.Fatalf("tampered active PR binding was accepted: %v", findings)
		}
	})

	t.Run("stale PR evidence", func(t *testing.T) {
		tampered := clonePlanningGrantMaterials(materials)
		const path = "docs/evidence/W-001-validation.md"
		tampered[path] = bytes.Replace(tampered[path], []byte("PR #8 is the sole\nactive publication vehicle"), []byte("PR #7 is the sole\nactive publication vehicle"), 1)
		root := writeW001PostclaimPRFixFixture(t, grant, signature, publicKey, tampered)
		var findings []Finding
		checkW001PostclaimPRFix(root, &findings)
		if !findingCodePresent(findings, "public.w001_postclaim_pr_binding_material") || !findingCodePresent(findings, "public.w001_postclaim_pr_binding_evidence") {
			t.Fatalf("stale publication evidence was accepted: %v", findings)
		}
	})

	t.Run("pull request event number", func(t *testing.T) {
		root := planningGrantCanonicalTempDir(t)
		writePlanningGrantTestFile(t, root, w001PostclaimPRFixPath, grant)
		if w001PostclaimPullRequestNumberAllowed(root, 7) || !w001PostclaimPullRequestNumberAllowed(root, 8) {
			t.Fatal("signed active PR number was not enforced")
		}
	})
}

func writeW001PostclaimPRFixFixture(t *testing.T, grant, signature, publicKey []byte, materials map[string][]byte) string {
	t.Helper()
	root := planningGrantCanonicalTempDir(t)
	writePlanningGrantTestFile(t, root, w001PostclaimPRFixPath, grant)
	writePlanningGrantTestFile(t, root, w001PostclaimPRFixSig, signature)
	writePlanningGrantTestFile(t, root, wave1PlanningGrantKey, publicKey)
	for path, data := range materials {
		writePlanningGrantTestFile(t, root, path, data)
	}
	return root
}

func TestW001PostclaimChronoFixAcceptsPinnedSignedContract(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	var findings []Finding
	checkW001PostclaimChronoFix(repo, &findings)
	if len(findings) != 0 {
		t.Fatalf("valid signed postclaim chronology correction was rejected: %v", findings)
	}
}

func TestW001PostclaimChronoFixFailsClosed(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	read := func(path string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	grant := read(w001PostclaimChronoFixPath)
	signature := read(w001PostclaimChronoFixSig)
	publicKey := read(wave1PlanningGrantKey)
	materials := map[string][]byte{
		w001PostclaimPRFixPath:              read(w001PostclaimPRFixPath),
		w001PostclaimPRFixSig:               read(w001PostclaimPRFixSig),
		"docs/evidence/W-001-validation.md": read("docs/evidence/W-001-validation.md"),
	}

	t.Run("signed chronology tamper", func(t *testing.T) {
		tampered := bytes.Replace(grant, []byte("v5EffectsAuthorizedByV5: false"), []byte("v5EffectsAuthorizedByV5: true"), 1)
		root := writeW001PostclaimChronoFixFixture(t, tampered, signature, publicKey, materials)
		var findings []Finding
		checkW001PostclaimChronoFix(root, &findings)
		if !findingCodePresent(findings, "public.w001_postclaim_chronology_value") || !findingCodePresent(findings, "public.w001_postclaim_chronology_signature") {
			t.Fatalf("tampered chronology disposition was accepted: %v", findings)
		}
	})

	t.Run("current evidence tamper", func(t *testing.T) {
		tampered := clonePlanningGrantMaterials(materials)
		const path = "docs/evidence/W-001-validation.md"
		tampered[path] = bytes.Replace(tampered[path], []byte("grant-effective-after-governed-effects"), []byte("missing-chronology-fingerprint"), 1)
		root := writeW001PostclaimChronoFixFixture(t, grant, signature, publicKey, tampered)
		var findings []Finding
		checkW001PostclaimChronoFix(root, &findings)
		if !findingCodePresent(findings, "public.w001_postclaim_chronology_material") || !findingCodePresent(findings, "public.w001_postclaim_chronology_evidence") {
			t.Fatalf("tampered chronology evidence was accepted: %v", findings)
		}
	})

	t.Run("historical phase chronology must remain pre-effective", func(t *testing.T) {
		document := parseStrictGrant(grant, w001PostclaimChronoFixScalars, w001PostclaimChronoFixSequences,
			[]string{"grant", "finding", "chronology", "publication", "materials", "verification", "integrity"})
		document.scalars["chronology.v5CommitAt"] = []string{"2026-08-27T00:00:01Z"}
		issued, err := time.Parse(time.RFC3339, scalarValue(document, "grant.issuedAt"))
		if err != nil {
			t.Fatal(err)
		}
		var findings []Finding
		checkW001PostclaimChronology(repo, document, issued, &findings)
		if !findingCodePresent(findings, "public.w001_postclaim_chronology_record") {
			t.Fatalf("false historical chronology was accepted: %v", findings)
		}
	})
}

func writeW001PostclaimChronoFixFixture(t *testing.T, grant, signature, publicKey []byte, materials map[string][]byte) string {
	t.Helper()
	root := planningGrantCanonicalTempDir(t)
	writePlanningGrantTestFile(t, root, w001PostclaimChronoFixPath, grant)
	writePlanningGrantTestFile(t, root, w001PostclaimChronoFixSig, signature)
	writePlanningGrantTestFile(t, root, wave1PlanningGrantKey, publicKey)
	for path, data := range materials {
		writePlanningGrantTestFile(t, root, path, data)
	}
	return root
}

func TestW001DeliveryGrantAcceptsPinnedSignedContract(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	var findings []Finding
	checkW001DeliveryGrant(repo, &findings)
	if len(findings) != 0 {
		t.Fatalf("valid signed W-001 delivery grant was rejected: %v", findings)
	}
}

func TestLoadW001DeliveryGrantReturnsBoundedRuntimeProjection(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	grant, err := loadW001DeliveryGrant(repo, time.Date(2026, 8, 27, 1, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if grant.ID != "W-001-delivery-v2" || grant.Repository != planningGrantRepository || grant.Bead != "M3-W001" ||
		grant.Principal != "work-authority-engineer" || grant.AttemptID != "w001-delivery-87d9680d-ca5a-4f3d-9afc-741884232e73" ||
		grant.IdempotencyKey != "w001-key" || grant.BaseCommit != w001DeliveryBase ||
		grant.ExpectedNativeStatus != "in_progress" || grant.ExpectedLifecycleState != "in-progress" || grant.ExpectedAssignee != grant.Principal ||
		grant.CanonicalClaimAttemptID != "w001-bootstrap-3135f1d1-b0d4-4956-9fc9-1852310bfd77" ||
		grant.WorkVersionGeneration != "6e79ff81-a007-42a5-a178-7ce58dbb718b" ||
		grant.WorkVersionIncarnation != "e1e8d2d3f80871096a568fb489f49575a42abd37b269df9faf777a09cd689b41" ||
		grant.IssueMutationSequence != 1 || grant.DependencyGraphRevision != 1 || grant.CanonicalWorkMutationAllowed || !grant.DevelopmentLeaseAllowed {
		t.Fatalf("unexpected runtime projection: %+v", grant)
	}
	if _, err := loadW001DeliveryGrant(repo, grant.ExpiresAt); err == nil {
		t.Fatal("expired delivery grant was accepted")
	}
}

func TestW001DeliveryGrantFailsClosed(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	read := func(path string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	grant := read(w001DeliveryGrantPath)
	signature := read(w001DeliveryGrantSignature)
	publicKey := read(wave1PlanningGrantKey)

	t.Run("implementation authority tamper", func(t *testing.T) {
		tampered := bytes.Replace(grant, []byte("implementationAllowed: true"), []byte("implementationAllowed: false"), 1)
		root := planningGrantCanonicalTempDir(t)
		writePlanningGrantTestFile(t, root, w001DeliveryGrantPath, tampered)
		writePlanningGrantTestFile(t, root, w001DeliveryGrantSignature, signature)
		writePlanningGrantTestFile(t, root, wave1PlanningGrantKey, publicKey)
		var findings []Finding
		checkW001DeliveryGrant(root, &findings)
		if !findingCodePresent(findings, "public.w001_delivery_value") || !findingCodePresent(findings, "public.w001_delivery_signature") {
			t.Fatalf("tampered delivery authority was accepted: %v", findings)
		}
	})

	t.Run("path scope tamper", func(t *testing.T) {
		tampered := bytes.Replace(grant, []byte("    - internal/authority/**"), []byte("    - internal/runtime/**"), 1)
		root := planningGrantCanonicalTempDir(t)
		writePlanningGrantTestFile(t, root, w001DeliveryGrantPath, tampered)
		writePlanningGrantTestFile(t, root, w001DeliveryGrantSignature, signature)
		writePlanningGrantTestFile(t, root, wave1PlanningGrantKey, publicKey)
		var findings []Finding
		checkW001DeliveryGrant(root, &findings)
		if !findingCodePresent(findings, "public.w001_delivery_sequence") || !findingCodePresent(findings, "public.w001_delivery_signature") {
			t.Fatalf("tampered delivery path scope was accepted: %v", findings)
		}
	})
}

func TestW001DeliveryCIFixAcceptsPinnedSignedContract(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	var findings []Finding
	checkW001DeliveryCIFix(repo, &findings)
	if len(findings) != 0 {
		t.Fatalf("valid signed W-001 delivery CI correction was rejected: %v", findings)
	}
}

func TestW001DeliveryCIFixFailsClosed(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	read := func(path string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	grant := read(w001DeliveryCIFixPath)
	signature := read(w001DeliveryCIFixSignature)
	tampered := bytes.Replace(grant, []byte("requiredTagger: release-manager"), []byte("requiredTagger: work-authority-engineer"), 1)
	root := planningGrantCanonicalTempDir(t)
	runPlanningGrantTestGit(t, root, "init", "--quiet")
	source, err := filepath.Abs(repo)
	if err != nil {
		t.Fatal(err)
	}
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source,
		"refs/tags/"+w001DeliveryReviewTag+":refs/tags/"+w001DeliveryReviewTag)
	for path, data := range map[string][]byte{
		w001DeliveryCIFixPath:               tampered,
		w001DeliveryCIFixSignature:          signature,
		w001DeliveryGrantPath:               read(w001DeliveryGrantPath),
		w001DeliveryGrantSignature:          read(w001DeliveryGrantSignature),
		wave1PlanningGrantKey:               read(wave1PlanningGrantKey),
		"docs/evidence/W-001-validation.md": read("docs/evidence/W-001-validation.md"),
	} {
		writePlanningGrantTestFile(t, root, path, data)
	}
	var findings []Finding
	checkW001DeliveryCIFix(root, &findings)
	if !findingCodePresent(findings, "public.w001_delivery_ci_value") || !findingCodePresent(findings, "public.w001_delivery_ci_signature") {
		t.Fatalf("tampered delivery CI correction was accepted: %v", findings)
	}
}

func TestW001DeliveryScannerFixAcceptsPinnedSignedContract(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	var findings []Finding
	checkW001DeliveryScannerFix(repo, &findings)
	if len(findings) != 0 {
		t.Fatalf("valid signed W-001 scanner correction was rejected: %v", findings)
	}
}

func TestW001DeliveryScannerFixGrantTamperFails(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	read := func(path string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	tampered := bytes.Replace(read(w001DeliveryScannerFixPath), []byte("historyFindings: 10"), []byte("historyFindings: 11"), 1)
	root := planningGrantCanonicalTempDir(t)
	runPlanningGrantTestGit(t, root, "init", "--quiet")
	source, err := filepath.Abs(repo)
	if err != nil {
		t.Fatal(err)
	}
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, w001DeliveryV1PreservedHead)
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source,
		"refs/tags/"+w001DeliveryCIFixReviewTag+":refs/tags/"+w001DeliveryCIFixReviewTag)
	for path, data := range map[string][]byte{
		w001DeliveryScannerFixPath:          tampered,
		w001DeliveryScannerFixSignature:     read(w001DeliveryScannerFixSignature),
		w001DeliveryCIFixPath:               read(w001DeliveryCIFixPath),
		w001DeliveryCIFixSignature:          read(w001DeliveryCIFixSignature),
		w001DeliveryScannerIgnorePath:       read(w001DeliveryScannerIgnorePath),
		wave1PlanningGrantKey:               read(wave1PlanningGrantKey),
		"docs/evidence/W-001-validation.md": read("docs/evidence/W-001-validation.md"),
	} {
		writePlanningGrantTestFile(t, root, path, data)
	}
	var findings []Finding
	checkW001DeliveryScannerFix(root, &findings)
	if !findingCodePresent(findings, "public.w001_delivery_scanner_value") || !findingCodePresent(findings, "public.w001_delivery_scanner_signature") {
		t.Fatalf("tampered scanner correction was accepted: %v", findings)
	}
}

func TestW001DeliveryScannerIgnoreFailsClosed(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	source, err := filepath.Abs(repo)
	if err != nil {
		t.Fatal(err)
	}
	exact := strings.Join(w001DeliveryScannerLegacyFingerprints, "\n") + "\n"
	for _, testCase := range []struct {
		name string
		body string
	}{
		{name: "changed", body: strings.Replace(exact, "0faf9071", "1faf9071", 1)},
		{name: "extra", body: exact + "*:generic-api-key:*\n"},
		{name: "missing", body: strings.TrimPrefix(exact, w001DeliveryScannerLegacyFingerprints[0]+"\n")},
		{name: "duplicate", body: exact + w001DeliveryScannerLegacyFingerprints[0] + "\n"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			root := planningGrantCanonicalTempDir(t)
			runPlanningGrantTestGit(t, root, "init", "--quiet")
			runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, w001DeliveryV1PreservedHead)
			writePlanningGrantTestFile(t, root, w001DeliveryScannerIgnorePath, []byte(testCase.body))
			var findings []Finding
			checkW001DeliveryScannerIgnore(root, &findings)
			if !findingCodePresent(findings, "public.w001_delivery_scanner_ignore") {
				t.Fatalf("unsafe scanner exception was accepted: %v", findings)
			}
		})
	}
}

func TestW001DeliveryScannerFingerprintSourcesFailClosed(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	source, err := filepath.Abs(repo)
	if err != nil {
		t.Fatal(err)
	}
	root := planningGrantCanonicalTempDir(t)
	runPlanningGrantTestGit(t, root, "init", "--quiet")
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, w001DeliveryV1PreservedHead)

	tests := []struct {
		name string
		list []string
		code string
	}{
		{name: "duplicate", list: append(append([]string{}, w001DeliveryScannerFingerprints...), w001DeliveryScannerFingerprints[0]), code: "public.w001_delivery_scanner_duplicate"},
		{name: "wildcard", list: []string{"*:internal/authority/beads/store_test.go:generic-api-key:96"}, code: "public.w001_delivery_scanner_fingerprint"},
		{name: "wrong rule", list: []string{"0faf90716d40aa3c5251c0a9c887cc70f06cfa1e:internal/authority/beads/store_test.go:github-pat:96"}, code: "public.w001_delivery_scanner_fingerprint"},
		{name: "missing line", list: []string{"0faf90716d40aa3c5251c0a9c887cc70f06cfa1e:internal/authority/beads/store_test.go:generic-api-key:99999"}, code: "public.w001_delivery_scanner_source"},
		{name: "outside history", list: []string{w001DeliveryScannerFixBase + ":docs/evidence/W-001-validation.md:generic-api-key:1"}, code: "public.w001_delivery_scanner_history"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			var findings []Finding
			checkW001DeliveryScannerFingerprintSources(root, testCase.list, &findings)
			if !findingCodePresent(findings, testCase.code) {
				t.Fatalf("unsafe scanner source tuple was accepted: %v", findings)
			}
		})
	}
}

func TestW001LifecycleCompletionGrantAcceptsPinnedSignedContract(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	var findings []Finding
	checkW001LifecycleCompletionGrant(repo, &findings)
	if len(findings) != 0 {
		t.Fatalf("valid signed W-001 lifecycle-completion grant was rejected: %v", findings)
	}
}

func TestW001LifecycleCompletionGrantFailsClosed(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	read := func(path string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	source, err := filepath.Abs(repo)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture := func(t *testing.T, grant []byte, evidence []byte) string {
		t.Helper()
		root := planningGrantCanonicalTempDir(t)
		runPlanningGrantTestGit(t, root, "init", "--quiet")
		runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, w001LifecycleBase)
		runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source,
			"refs/tags/"+w001DeliveryScannerFixReviewTag+":refs/tags/"+w001DeliveryScannerFixReviewTag)
		for path, data := range map[string][]byte{
			w001LifecycleGrantPath:              grant,
			w001LifecycleGrantSignature:         read(w001LifecycleGrantSignature),
			w001DeliveryScannerFixPath:          read(w001DeliveryScannerFixPath),
			w001DeliveryScannerFixSignature:     read(w001DeliveryScannerFixSignature),
			wave1PlanningGrantKey:               read(wave1PlanningGrantKey),
			"docs/evidence/W-001-validation.md": evidence,
			canonicalActivePlan:                 read(canonicalActivePlan),
			".harness/manifest.yaml":            read(".harness/manifest.yaml"),
		} {
			writePlanningGrantTestFile(t, root, path, data)
		}
		return root
	}

	t.Run("authority tamper", func(t *testing.T) {
		grant := bytes.Replace(read(w001LifecycleGrantPath), []byte("canonicalLifecycleMutationAllowed: false"), []byte("canonicalLifecycleMutationAllowed: true"), 1)
		root := writeFixture(t, grant, read("docs/evidence/W-001-validation.md"))
		var findings []Finding
		checkW001LifecycleCompletionGrant(root, &findings)
		if !findingCodePresent(findings, "public.w001_lifecycle_value") || !findingCodePresent(findings, "public.w001_lifecycle_signature") {
			t.Fatalf("tampered lifecycle authority was accepted: %v", findings)
		}
	})

	t.Run("evidence tamper", func(t *testing.T) {
		evidence := bytes.Replace(read("docs/evidence/W-001-validation.md"), []byte("completion-audit/governed-lifecycle-routes-missing"), []byte("missing-completion-fingerprint"), 1)
		root := writeFixture(t, read(w001LifecycleGrantPath), evidence)
		var findings []Finding
		checkW001LifecycleCompletionGrant(root, &findings)
		if !findingCodePresent(findings, "public.w001_lifecycle_evidence") {
			t.Fatalf("tampered lifecycle evidence was accepted: %v", findings)
		}
	})
}

func TestW001LifecycleCompletionPathScope(t *testing.T) {
	for _, path := range []string{
		w001LifecycleGrantPath,
		"docs/features/F-002-work-authority.md",
		"api/authority/v1/types.go",
		"internal/authority/beads/mutator.go",
		"internal/authority/gateway/lifecycle.go",
		"internal/authority/httpapi/handler.go",
		"internal/authority/postgres/lifecycle.go",
		"database/authority/002_lifecycle.sql",
	} {
		if !w001LifecyclePathsAllowed([]string{path}) {
			t.Fatalf("authorized lifecycle path was rejected: %s", path)
		}
	}
	for _, path := range []string{".github/workflows/foundation-quality.yml", "internal/platform/runtime.go", "docs/features/F-003-local-substrate.md", "go.mod"} {
		if w001LifecyclePathsAllowed([]string{path}) {
			t.Fatalf("out-of-scope lifecycle path was accepted: %s", path)
		}
	}
}

func TestW001LifecycleCorrectionGrantAcceptsPinnedSignedContract(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	var findings []Finding
	checkW001LifecycleCorrectionGrant(repo, &findings)
	if len(findings) != 0 {
		t.Fatalf("valid signed W-001 lifecycle correction was rejected: %v", findings)
	}
}

func TestW001LifecycleCorrectionGrantFailsClosed(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	read := func(path string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	source, err := filepath.Abs(repo)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture := func(t *testing.T, grant, evidence []byte) string {
		t.Helper()
		root := planningGrantCanonicalTempDir(t)
		runPlanningGrantTestGit(t, root, "init", "--quiet")
		runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, w001LifecycleCorrectionBase)
		runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source,
			"refs/tags/"+w001LifecycleReviewTag+":refs/tags/"+w001LifecycleReviewTag)
		for path, data := range map[string][]byte{
			w001LifecycleCorrectionPath:         grant,
			w001LifecycleCorrectionSignature:    read(w001LifecycleCorrectionSignature),
			w001LifecycleGrantPath:              read(w001LifecycleGrantPath),
			w001LifecycleGrantSignature:         read(w001LifecycleGrantSignature),
			wave1PlanningGrantKey:               read(wave1PlanningGrantKey),
			"docs/evidence/W-001-validation.md": evidence,
			canonicalActivePlan:                 read(canonicalActivePlan),
			".harness/manifest.yaml":            read(".harness/manifest.yaml"),
		} {
			writePlanningGrantTestFile(t, root, path, data)
		}
		return root
	}

	t.Run("authority tamper", func(t *testing.T) {
		grant := bytes.Replace(read(w001LifecycleCorrectionPath), []byte("canonicalLifecycleMutationAllowed: false"), []byte("canonicalLifecycleMutationAllowed: true"), 1)
		root := writeFixture(t, grant, read("docs/evidence/W-001-validation.md"))
		var findings []Finding
		checkW001LifecycleCorrectionGrant(root, &findings)
		if !findingCodePresent(findings, "public.w001_lifecycle_correction_value") || !findingCodePresent(findings, "public.w001_lifecycle_correction_signature") {
			t.Fatalf("tampered lifecycle correction authority was accepted: %v", findings)
		}
	})

	t.Run("evidence tamper", func(t *testing.T) {
		evidence := bytes.Replace(read("docs/evidence/W-001-validation.md"), []byte("lifecycle.handoff_replay_fence_splice"), []byte("missing-fence-splice-finding"), 1)
		root := writeFixture(t, read(w001LifecycleCorrectionPath), evidence)
		var findings []Finding
		checkW001LifecycleCorrectionGrant(root, &findings)
		if !findingCodePresent(findings, "public.w001_lifecycle_correction_evidence") {
			t.Fatalf("tampered lifecycle correction evidence was accepted: %v", findings)
		}
	})
}

func TestW001LifecycleCorrectionPathScope(t *testing.T) {
	for _, path := range []string{
		w001LifecycleCorrectionPath,
		"docs/evidence/W-001-validation.md",
		"api/authority/v1/types.go",
		"internal/authority/beads/mutator.go",
		"internal/authority/gateway/lifecycle.go",
		"internal/doctrine/grant_test.go",
	} {
		if !w001LifecycleCorrectionPathsAllowed([]string{path}) {
			t.Fatalf("authorized lifecycle correction path was rejected: %s", path)
		}
	}
	for _, path := range []string{
		w001LifecycleGrantPath,
		".github/workflows/foundation-quality.yml",
		"internal/platform/runtime.go",
		"go.mod",
	} {
		if w001LifecycleCorrectionPathsAllowed([]string{path}) {
			t.Fatalf("out-of-scope lifecycle correction path was accepted: %s", path)
		}
	}
}

func TestW001LifecycleCorrectionV7GrantAcceptsPinnedSignedContract(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	var findings []Finding
	checkW001LifecycleCorrectionV7Grant(repo, &findings)
	if len(findings) != 0 {
		t.Fatalf("valid signed W-001 v7 lifecycle correction was rejected: %v", findings)
	}
}

func TestW001LifecycleCorrectionV7EvidenceBindingsFailClosed(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	var findings []Finding
	checkW001LifecycleCorrectionV7Evidence(repo, &findings)
	if len(findings) != 0 {
		t.Fatalf("valid v7 lifecycle evidence bindings were rejected: %v", findings)
	}
	root := planningGrantCanonicalTempDir(t)
	evidence, err := os.ReadFile(filepath.Join(repo, "docs", "evidence", "W-001-validation.md"))
	if err != nil {
		t.Fatal(err)
	}
	tamperedEvidence := bytes.Replace(evidence, []byte("bb8dd437802943670b4e882a3cdc30d5ea5a3b2035171fb765d7d82db7f624de"), []byte(strings.Repeat("0", 64)), 1)
	writePlanningGrantTestFile(t, root, "docs/evidence/W-001-validation.md", tamperedEvidence)
	writePlanningGrantTestFile(t, root, w001LifecycleCorrectionV7PatchPath, []byte("not-the-reviewed-patch\n"))
	findings = nil
	checkW001LifecycleCorrectionV7Evidence(root, &findings)
	if !findingCodePresent(findings, "public.w001_lifecycle_correction_v7_evidence") ||
		!findingCodePresent(findings, "public.w001_lifecycle_correction_v7_patch") {
		t.Fatalf("tampered v7 evidence bindings were accepted: %v", findings)
	}
}

func TestW001LifecycleCorrectionV7GrantFailsClosed(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	read := func(path string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	source, err := filepath.Abs(repo)
	if err != nil {
		t.Fatal(err)
	}
	root := planningGrantCanonicalTempDir(t)
	runPlanningGrantTestGit(t, root, "init", "--quiet")
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, w001LifecycleCorrectionV7Base)
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source,
		"refs/tags/"+w001LifecycleCorrectionReviewTag+":refs/tags/"+w001LifecycleCorrectionReviewTag)
	tampered := bytes.Replace(read(w001LifecycleCorrectionV7Path), []byte("canonicalLifecycleMutationAllowed: false"), []byte("canonicalLifecycleMutationAllowed: true"), 1)
	for path, data := range map[string][]byte{
		w001LifecycleCorrectionV7Path:       tampered,
		w001LifecycleCorrectionV7Signature:  read(w001LifecycleCorrectionV7Signature),
		w001LifecycleCorrectionPath:         read(w001LifecycleCorrectionPath),
		w001LifecycleCorrectionSignature:    read(w001LifecycleCorrectionSignature),
		wave1PlanningGrantKey:               read(wave1PlanningGrantKey),
		"docs/evidence/W-001-validation.md": read("docs/evidence/W-001-validation.md"),
		canonicalActivePlan:                 read(canonicalActivePlan),
		".harness/manifest.yaml":            read(".harness/manifest.yaml"),
	} {
		writePlanningGrantTestFile(t, root, path, data)
	}
	var findings []Finding
	checkW001LifecycleCorrectionV7Grant(root, &findings)
	if !findingCodePresent(findings, "public.w001_lifecycle_correction_v7_value") ||
		!findingCodePresent(findings, "public.w001_lifecycle_correction_v7_signature") {
		t.Fatalf("tampered v7 lifecycle correction authority was accepted: %v", findings)
	}
}

func TestW001LifecycleCorrectionV7PathScope(t *testing.T) {
	for _, path := range []string{
		w001LifecycleCorrectionV7Path,
		"docs/evidence/W-001-validation.md",
		"api/authority/v1/types.go",
		"internal/authority/beads/reproducible_build_test.go",
		"internal/authority/gateway/lifecycle.go",
		"internal/doctrine/grant_test.go",
	} {
		if !w001LifecycleCorrectionV7PathsAllowed([]string{path}) {
			t.Fatalf("authorized v7 lifecycle correction path was rejected: %s", path)
		}
	}
	for _, path := range []string{
		w001LifecycleCorrectionPath,
		".github/workflows/foundation-quality.yml",
		"internal/platform/runtime.go",
		"go.mod",
	} {
		if w001LifecycleCorrectionV7PathsAllowed([]string{path}) {
			t.Fatalf("out-of-scope v7 lifecycle correction path was accepted: %s", path)
		}
	}
}

func TestW001LifecycleCorrectionV8GrantAcceptsPinnedSignedContract(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	var findings []Finding
	checkW001LifecycleCorrectionV8Grant(repo, &findings)
	if len(findings) != 0 {
		t.Fatalf("valid signed W-001 v8 lifecycle correction was rejected: %v", findings)
	}
}

func TestW001LifecycleCorrectionV8GrantFailsClosed(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	read := func(path string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	source, err := filepath.Abs(repo)
	if err != nil {
		t.Fatal(err)
	}
	root := planningGrantCanonicalTempDir(t)
	runPlanningGrantTestGit(t, root, "init", "--quiet")
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, w001LifecycleCorrectionV8Base)
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source,
		"refs/tags/"+w001LifecycleCorrectionV7ReviewTag+":refs/tags/"+w001LifecycleCorrectionV7ReviewTag)
	tampered := bytes.Replace(read(w001LifecycleCorrectionV8Path), []byte("canonicalLifecycleMutationAllowed: false"), []byte("canonicalLifecycleMutationAllowed: true"), 1)
	for path, data := range map[string][]byte{
		w001LifecycleCorrectionV8Path:       tampered,
		w001LifecycleCorrectionV8Signature:  read(w001LifecycleCorrectionV8Signature),
		w001LifecycleCorrectionV7Path:       read(w001LifecycleCorrectionV7Path),
		w001LifecycleCorrectionV7Signature:  read(w001LifecycleCorrectionV7Signature),
		wave1PlanningGrantKey:               read(wave1PlanningGrantKey),
		"docs/evidence/W-001-validation.md": read("docs/evidence/W-001-validation.md"),
		canonicalActivePlan:                 read(canonicalActivePlan),
		".harness/manifest.yaml":            read(".harness/manifest.yaml"),
	} {
		writePlanningGrantTestFile(t, root, path, data)
	}
	var findings []Finding
	checkW001LifecycleCorrectionV8Grant(root, &findings)
	if !findingCodePresent(findings, "public.w001_lifecycle_correction_v8_value") ||
		!findingCodePresent(findings, "public.w001_lifecycle_correction_v8_signature") {
		t.Fatalf("tampered v8 lifecycle correction authority was accepted: %v", findings)
	}
}

func TestW001LifecycleCorrectionV8PathScope(t *testing.T) {
	for _, path := range []string{
		w001LifecycleCorrectionV8Path,
		w001LifecycleCorrectionV8Signature,
		"docs/evidence/W-001-validation.md",
		"internal/authority/beads/store.go",
		"internal/doctrine/grant_test.go",
	} {
		if !w001LifecycleCorrectionV8PathsAllowed([]string{path}) {
			t.Fatalf("authorized v8 lifecycle correction path was rejected: %s", path)
		}
	}
	for _, path := range []string{
		w001LifecycleCorrectionV7Path,
		".github/workflows/foundation-quality.yml",
		"internal/authority/gateway/lifecycle.go",
		"go.mod",
	} {
		if w001LifecycleCorrectionV8PathsAllowed([]string{path}) {
			t.Fatalf("out-of-scope v8 lifecycle correction path was accepted: %s", path)
		}
	}
}

func TestW001LifecycleCorrectionV9GrantAcceptsPinnedSignedContract(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	var findings []Finding
	checkW001LifecycleCorrectionV9Grant(repo, &findings)
	if len(findings) != 0 {
		t.Fatalf("valid signed W-001 v9 lifecycle correction was rejected: %v", findings)
	}
}

func TestW001LifecycleCorrectionV9GrantFailsClosed(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	read := func(path string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	source, err := filepath.Abs(repo)
	if err != nil {
		t.Fatal(err)
	}
	root := planningGrantCanonicalTempDir(t)
	runPlanningGrantTestGit(t, root, "init", "--quiet")
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, w001LifecycleCorrectionV9Base)
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source,
		"refs/tags/"+w001LifecycleCorrectionV8ReviewTag+":refs/tags/"+w001LifecycleCorrectionV8ReviewTag)
	tampered := bytes.Replace(read(w001LifecycleCorrectionV9Path), []byte("canonicalLifecycleMutationAllowed: false"), []byte("canonicalLifecycleMutationAllowed: true"), 1)
	for path, data := range map[string][]byte{
		w001LifecycleCorrectionV9Path:       tampered,
		w001LifecycleCorrectionV9Signature:  read(w001LifecycleCorrectionV9Signature),
		w001LifecycleCorrectionV8Path:       read(w001LifecycleCorrectionV8Path),
		w001LifecycleCorrectionV8Signature:  read(w001LifecycleCorrectionV8Signature),
		w001LifecycleCorrectionV7PatchPath:  read(w001LifecycleCorrectionV7PatchPath),
		wave1PlanningGrantKey:               read(wave1PlanningGrantKey),
		"docs/evidence/W-001-validation.md": read("docs/evidence/W-001-validation.md"),
		canonicalActivePlan:                 read(canonicalActivePlan),
		".harness/manifest.yaml":            read(".harness/manifest.yaml"),
	} {
		writePlanningGrantTestFile(t, root, path, data)
	}
	var findings []Finding
	checkW001LifecycleCorrectionV9Grant(root, &findings)
	if !findingCodePresent(findings, "public.w001_lifecycle_correction_v9_value") ||
		!findingCodePresent(findings, "public.w001_lifecycle_correction_v9_signature") {
		t.Fatalf("tampered v9 lifecycle correction authority was accepted: %v", findings)
	}
}

func TestW001LifecycleCorrectionV9PathScope(t *testing.T) {
	for _, path := range []string{
		w001LifecycleCorrectionV9Path,
		w001LifecycleCorrectionV9Signature,
		"docs/evidence/W-001-validation.md",
		"docs/product-specs/work-authority.md",
		"internal/authority/beads/store.go",
		"internal/doctrine/grant_test.go",
	} {
		if !w001LifecycleCorrectionV9PathsAllowed([]string{path}) {
			t.Fatalf("authorized v9 lifecycle correction path was rejected: %s", path)
		}
	}
	for _, path := range []string{
		w001LifecycleCorrectionV8Path,
		".github/workflows/foundation-quality.yml",
		"internal/authority/gateway/lifecycle.go",
		"go.mod",
	} {
		if w001LifecycleCorrectionV9PathsAllowed([]string{path}) {
			t.Fatalf("out-of-scope v9 lifecycle correction path was accepted: %s", path)
		}
	}
}

func TestW001LifecycleStabilizationV10GrantAcceptsPinnedSignedContract(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	var findings []Finding
	checkW001LifecycleStabilizationV10Grant(repo, &findings)
	if len(findings) != 0 {
		t.Fatalf("valid signed W-001 v10 lifecycle CI stabilization was rejected: %v", findings)
	}
}

func TestW001LifecycleStabilizationV10GrantFailsClosed(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	read := func(path string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	source, err := filepath.Abs(repo)
	if err != nil {
		t.Fatal(err)
	}
	root := planningGrantCanonicalTempDir(t)
	runPlanningGrantTestGit(t, root, "init", "--quiet")
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, w001LifecycleStabilizationV10Base)
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source,
		"refs/tags/"+w001LifecycleCorrectionV9ReviewTag+":refs/tags/"+w001LifecycleCorrectionV9ReviewTag)
	tampered := bytes.Replace(read(w001LifecycleStabilizationV10Path), []byte("canonicalLifecycleMutationAllowed: false"), []byte("canonicalLifecycleMutationAllowed: true"), 1)
	for path, data := range map[string][]byte{
		w001LifecycleStabilizationV10Path:      tampered,
		w001LifecycleStabilizationV10Signature: read(w001LifecycleStabilizationV10Signature),
		w001LifecycleCorrectionV9Path:          read(w001LifecycleCorrectionV9Path),
		w001LifecycleCorrectionV9Signature:     read(w001LifecycleCorrectionV9Signature),
		wave1PlanningGrantKey:                  read(wave1PlanningGrantKey),
		"docs/evidence/W-001-validation.md":    read("docs/evidence/W-001-validation.md"),
		canonicalActivePlan:                    read(canonicalActivePlan),
		".harness/manifest.yaml":               read(".harness/manifest.yaml"),
		"internal/doctrine/grant_test.go":      read("internal/doctrine/grant_test.go"),
	} {
		writePlanningGrantTestFile(t, root, path, data)
	}
	var findings []Finding
	checkW001LifecycleStabilizationV10Grant(root, &findings)
	if !findingCodePresent(findings, "public.w001_lifecycle_stabilization_v10_value") ||
		!findingCodePresent(findings, "public.w001_lifecycle_stabilization_v10_signature") {
		t.Fatalf("tampered v10 lifecycle CI stabilization authority was accepted: %v", findings)
	}
}

func TestW001LifecycleStabilizationV10PathScope(t *testing.T) {
	for _, path := range []string{
		w001LifecycleStabilizationV10Path,
		w001LifecycleStabilizationV10Signature,
		"docs/evidence/W-001-validation.md",
		"internal/doctrine/grant.go",
		"internal/doctrine/grant_test.go",
	} {
		if !w001LifecycleStabilizationV10PathsAllowed([]string{path}) {
			t.Fatalf("authorized v10 lifecycle CI stabilization path was rejected: %s", path)
		}
	}
	for _, path := range []string{
		w001LifecycleCorrectionV9Path,
		".github/workflows/foundation-quality.yml",
		"internal/authority/beads/store.go",
		"go.mod",
	} {
		if w001LifecycleStabilizationV10PathsAllowed([]string{path}) {
			t.Fatalf("out-of-scope v10 lifecycle CI stabilization path was accepted: %s", path)
		}
	}
}

func TestW001LifecycleCIFencingV11GrantAcceptsPinnedSignedContract(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	var findings []Finding
	checkW001LifecycleCIFencingV11Grant(repo, &findings)
	if len(findings) != 0 {
		t.Fatalf("valid signed W-001 v11 lifecycle CI fencing correction was rejected: %v", findings)
	}
}

func TestW001LifecycleCIFencingV11GrantFailsClosed(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	read := func(path string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	source, err := filepath.Abs(repo)
	if err != nil {
		t.Fatal(err)
	}
	root := planningGrantCanonicalTempDir(t)
	runPlanningGrantTestGit(t, root, "init", "--quiet")
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, w001LifecycleCIFencingV11Base)
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source,
		"refs/tags/"+w001LifecycleStabilizationV10ReviewTag+":refs/tags/"+w001LifecycleStabilizationV10ReviewTag)
	tampered := bytes.Replace(read(w001LifecycleCIFencingV11Path), []byte("canonicalLifecycleMutationAllowed: false"), []byte("canonicalLifecycleMutationAllowed: true"), 1)
	for path, data := range map[string][]byte{
		w001LifecycleCIFencingV11Path:          tampered,
		w001LifecycleCIFencingV11Signature:     read(w001LifecycleCIFencingV11Signature),
		w001LifecycleStabilizationV10Path:      read(w001LifecycleStabilizationV10Path),
		w001LifecycleStabilizationV10Signature: read(w001LifecycleStabilizationV10Signature),
		wave1PlanningGrantKey:                  read(wave1PlanningGrantKey),
		"docs/evidence/W-001-validation.md":    read("docs/evidence/W-001-validation.md"),
		canonicalActivePlan:                    read(canonicalActivePlan),
		".harness/manifest.yaml":               read(".harness/manifest.yaml"),
		"internal/doctrine/grant_test.go":      read("internal/doctrine/grant_test.go"),
	} {
		writePlanningGrantTestFile(t, root, path, data)
	}
	var findings []Finding
	checkW001LifecycleCIFencingV11Grant(root, &findings)
	if !findingCodePresent(findings, "public.w001_lifecycle_ci_fencing_v11_value") ||
		!findingCodePresent(findings, "public.w001_lifecycle_ci_fencing_v11_signature") {
		t.Fatalf("tampered v11 lifecycle CI fencing authority was accepted: %v", findings)
	}
}

func TestW001LifecycleCIFencingV11PathScope(t *testing.T) {
	for _, path := range []string{
		w001LifecycleCIFencingV11Path,
		w001LifecycleCIFencingV11Signature,
		"docs/evidence/W-001-validation.md",
		"internal/doctrine/grant.go",
		"internal/doctrine/grant_test.go",
	} {
		if !w001LifecycleCIFencingV11PathsAllowed([]string{path}) {
			t.Fatalf("authorized v11 lifecycle CI fencing path was rejected: %s", path)
		}
	}
	for _, path := range []string{
		w001LifecycleStabilizationV10Path,
		".github/workflows/foundation-quality.yml",
		"internal/authority/beads/store.go",
		"go.mod",
	} {
		if w001LifecycleCIFencingV11PathsAllowed([]string{path}) {
			t.Fatalf("out-of-scope v11 lifecycle CI fencing path was accepted: %s", path)
		}
	}
}

func TestW001LifecycleCIHardeningV12GrantAcceptsPinnedSignedContract(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	var findings []Finding
	checkW001LifecycleCIHardeningV12Grant(repo, &findings)
	if len(findings) != 0 {
		t.Fatalf("valid signed W-001 v12 lifecycle CI hardening was rejected: %v", findings)
	}
}

func TestW001LifecycleCIHardeningV12GrantFailsClosed(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	read := func(path string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	source, err := filepath.Abs(repo)
	if err != nil {
		t.Fatal(err)
	}
	root := planningGrantCanonicalTempDir(t)
	runPlanningGrantTestGit(t, root, "init", "--quiet")
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, w001LifecycleCIHardeningV12Base)
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source,
		"refs/tags/"+w001LifecycleCIFencingV11ReviewTag+":refs/tags/"+w001LifecycleCIFencingV11ReviewTag)
	tampered := bytes.Replace(read(w001LifecycleCIHardeningV12Path), []byte("canonicalLifecycleMutationAllowed: false"), []byte("canonicalLifecycleMutationAllowed: true"), 1)
	for path, data := range map[string][]byte{
		w001LifecycleCIHardeningV12Path:      tampered,
		w001LifecycleCIHardeningV12Signature: read(w001LifecycleCIHardeningV12Signature),
		w001LifecycleCIFencingV11Path:        read(w001LifecycleCIFencingV11Path),
		w001LifecycleCIFencingV11Signature:   read(w001LifecycleCIFencingV11Signature),
		wave1PlanningGrantKey:                read(wave1PlanningGrantKey),
		"docs/evidence/W-001-validation.md":  read("docs/evidence/W-001-validation.md"),
		canonicalActivePlan:                  read(canonicalActivePlan),
		".harness/manifest.yaml":             read(".harness/manifest.yaml"),
		"internal/doctrine/grant_test.go":    read("internal/doctrine/grant_test.go"),
	} {
		writePlanningGrantTestFile(t, root, path, data)
	}
	var findings []Finding
	checkW001LifecycleCIHardeningV12Grant(root, &findings)
	if !findingCodePresent(findings, "public.w001_lifecycle_ci_hardening_v12_value") ||
		!findingCodePresent(findings, "public.w001_lifecycle_ci_hardening_v12_signature") {
		t.Fatalf("tampered v12 lifecycle CI hardening authority was accepted: %v", findings)
	}
}

func TestW001LifecycleCIHardeningV12PathScope(t *testing.T) {
	for _, path := range []string{
		w001LifecycleCIHardeningV12Path,
		w001LifecycleCIHardeningV12Signature,
		"docs/evidence/W-001-validation.md",
		"internal/doctrine/grant.go",
		"internal/doctrine/grant_test.go",
	} {
		if !w001LifecycleCIHardeningV12PathsAllowed([]string{path}) {
			t.Fatalf("authorized v12 lifecycle CI hardening path was rejected: %s", path)
		}
	}
	for _, path := range []string{
		w001LifecycleCIFencingV11Path,
		".github/workflows/foundation-quality.yml",
		"internal/authority/beads/store.go",
		"go.mod",
	} {
		if w001LifecycleCIHardeningV12PathsAllowed([]string{path}) {
			t.Fatalf("out-of-scope v12 lifecycle CI hardening path was accepted: %s", path)
		}
	}
}

func TestW001LifecycleCIHardeningV13GrantAcceptsPinnedSignedContract(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	var findings []Finding
	checkW001LifecycleCIHardeningV13Grant(repo, &findings)
	if len(findings) != 0 {
		t.Fatalf("valid signed W-001 v13 lifecycle CI hardening was rejected: %v", findings)
	}
}

func TestW001LifecycleCIHardeningV13GrantFailsClosed(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	read := func(path string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	source, err := filepath.Abs(repo)
	if err != nil {
		t.Fatal(err)
	}
	root := planningGrantCanonicalTempDir(t)
	runPlanningGrantTestGit(t, root, "init", "--quiet")
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, w001LifecycleCIHardeningV13Base)
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source,
		"refs/tags/"+w001LifecycleCIHardeningV12ReviewTag+":refs/tags/"+w001LifecycleCIHardeningV12ReviewTag)
	tampered := bytes.Replace(read(w001LifecycleCIHardeningV13Path), []byte("canonicalLifecycleMutationAllowed: false"), []byte("canonicalLifecycleMutationAllowed: true"), 1)
	for path, data := range map[string][]byte{
		w001LifecycleCIHardeningV13Path:      tampered,
		w001LifecycleCIHardeningV13Signature: read(w001LifecycleCIHardeningV13Signature),
		w001LifecycleCIHardeningV12Path:      read(w001LifecycleCIHardeningV12Path),
		w001LifecycleCIHardeningV12Signature: read(w001LifecycleCIHardeningV12Signature),
		wave1PlanningGrantKey:                read(wave1PlanningGrantKey),
		"docs/evidence/W-001-validation.md":  read("docs/evidence/W-001-validation.md"),
		canonicalActivePlan:                  read(canonicalActivePlan),
		".harness/manifest.yaml":             read(".harness/manifest.yaml"),
		"internal/doctrine/grant_test.go":    read("internal/doctrine/grant_test.go"),
	} {
		writePlanningGrantTestFile(t, root, path, data)
	}
	var findings []Finding
	checkW001LifecycleCIHardeningV13Grant(root, &findings)
	if !findingCodePresent(findings, "public.w001_lifecycle_ci_hardening_v13_value") ||
		!findingCodePresent(findings, "public.w001_lifecycle_ci_hardening_v13_signature") {
		t.Fatalf("tampered v13 lifecycle CI hardening authority was accepted: %v", findings)
	}
}

func TestW001LifecycleCIHardeningV13PathScope(t *testing.T) {
	for _, path := range []string{
		w001LifecycleCIHardeningV13Path,
		w001LifecycleCIHardeningV13Signature,
		".harness/manifest.yaml",
		canonicalActivePlan,
		"docs/evidence/W-001-validation.md",
		"internal/doctrine/grant.go",
		"internal/doctrine/grant_test.go",
	} {
		if !w001LifecycleCIHardeningV13PathsAllowed([]string{path}) {
			t.Fatalf("authorized v13 lifecycle CI hardening path was rejected: %s", path)
		}
	}
	for _, path := range []string{
		w001LifecycleCIHardeningV12Path,
		".github/workflows/foundation-quality.yml",
		"internal/authority/beads/store.go",
		"go.mod",
	} {
		if w001LifecycleCIHardeningV13PathsAllowed([]string{path}) {
			t.Fatalf("out-of-scope v13 lifecycle CI hardening path was accepted: %s", path)
		}
	}
}

func TestW001LifecycleCIHardeningV14GrantAcceptsPinnedSignedContract(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	var findings []Finding
	checkW001LifecycleCIHardeningV14Grant(repo, &findings)
	if len(findings) != 0 {
		t.Fatalf("valid signed W-001 v14 lifecycle CI hardening was rejected: %v", findings)
	}
}

func TestW001LifecycleCIHardeningV14GrantFailsClosed(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	read := func(path string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	source, err := filepath.Abs(repo)
	if err != nil {
		t.Fatal(err)
	}
	root := planningGrantCanonicalTempDir(t)
	runPlanningGrantTestGit(t, root, "init", "--quiet")
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, w001LifecycleCIHardeningV14Base)
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source,
		"refs/tags/"+w001LifecycleCIHardeningV13ReviewTag+":refs/tags/"+w001LifecycleCIHardeningV13ReviewTag)
	tampered := bytes.Replace(read(w001LifecycleCIHardeningV14Path), []byte("canonicalLifecycleMutationAllowed: false"), []byte("canonicalLifecycleMutationAllowed: true"), 1)
	for path, data := range map[string][]byte{
		w001LifecycleCIHardeningV14Path:      tampered,
		w001LifecycleCIHardeningV14Signature: read(w001LifecycleCIHardeningV14Signature),
		w001LifecycleCIHardeningV13Path:      read(w001LifecycleCIHardeningV13Path),
		w001LifecycleCIHardeningV13Signature: read(w001LifecycleCIHardeningV13Signature),
		wave1PlanningGrantKey:                read(wave1PlanningGrantKey),
		"docs/evidence/W-001-validation.md":  read("docs/evidence/W-001-validation.md"),
		canonicalActivePlan:                  read(canonicalActivePlan),
		".harness/manifest.yaml":             read(".harness/manifest.yaml"),
		"internal/doctrine/grant_test.go":    read("internal/doctrine/grant_test.go"),
	} {
		writePlanningGrantTestFile(t, root, path, data)
	}
	var findings []Finding
	checkW001LifecycleCIHardeningV14Grant(root, &findings)
	if !findingCodePresent(findings, "public.w001_lifecycle_ci_hardening_v14_value") ||
		!findingCodePresent(findings, "public.w001_lifecycle_ci_hardening_v14_signature") {
		t.Fatalf("tampered v14 lifecycle CI hardening authority was accepted: %v", findings)
	}
}

func TestW001LifecycleCIHardeningV14PathScope(t *testing.T) {
	for _, path := range []string{
		w001LifecycleCIHardeningV14Path,
		w001LifecycleCIHardeningV14Signature,
		".harness/manifest.yaml",
		canonicalActivePlan,
		"docs/evidence/W-001-validation.md",
		"internal/doctrine/grant.go",
		"internal/doctrine/grant_test.go",
	} {
		if !w001LifecycleCIHardeningV14PathsAllowed([]string{path}) {
			t.Fatalf("authorized v14 lifecycle CI hardening path was rejected: %s", path)
		}
	}
	for _, path := range []string{
		w001LifecycleCIHardeningV13Path,
		".github/workflows/foundation-quality.yml",
		"internal/authority/beads/store.go",
		"go.mod",
	} {
		if w001LifecycleCIHardeningV14PathsAllowed([]string{path}) {
			t.Fatalf("out-of-scope v14 lifecycle CI hardening path was accepted: %s", path)
		}
	}
}

func TestW001LifecycleCIHardeningV15GrantAcceptsPinnedSignedContract(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	var findings []Finding
	checkW001LifecycleCIHardeningV15Grant(repo, &findings)
	if len(findings) != 0 {
		t.Fatalf("valid signed W-001 v15 lifecycle CI hardening was rejected: %v", findings)
	}
}

func TestW001LifecycleCIHardeningV15GrantFailsClosed(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	read := func(path string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	source, err := filepath.Abs(repo)
	if err != nil {
		t.Fatal(err)
	}
	root := planningGrantCanonicalTempDir(t)
	runPlanningGrantTestGit(t, root, "init", "--quiet")
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, w001LifecycleCIHardeningV15Base)
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source,
		"refs/tags/"+w001LifecycleCIHardeningV14ReviewTag+":refs/tags/"+w001LifecycleCIHardeningV14ReviewTag)
	tampered := bytes.Replace(read(w001LifecycleCIHardeningV15Path), []byte("canonicalLifecycleMutationAllowed: false"), []byte("canonicalLifecycleMutationAllowed: true"), 1)
	for path, data := range map[string][]byte{
		w001LifecycleCIHardeningV15Path:      tampered,
		w001LifecycleCIHardeningV15Signature: read(w001LifecycleCIHardeningV15Signature),
		w001LifecycleCIHardeningV14Path:      read(w001LifecycleCIHardeningV14Path),
		w001LifecycleCIHardeningV14Signature: read(w001LifecycleCIHardeningV14Signature),
		wave1PlanningGrantKey:                read(wave1PlanningGrantKey),
		"docs/evidence/W-001-validation.md":  read("docs/evidence/W-001-validation.md"),
		canonicalActivePlan:                  read(canonicalActivePlan),
		".harness/manifest.yaml":             read(".harness/manifest.yaml"),
		"internal/doctrine/grant_test.go":    read("internal/doctrine/grant_test.go"),
	} {
		writePlanningGrantTestFile(t, root, path, data)
	}
	var findings []Finding
	checkW001LifecycleCIHardeningV15Grant(root, &findings)
	if !findingCodePresent(findings, "public.w001_lifecycle_ci_hardening_v15_value") ||
		!findingCodePresent(findings, "public.w001_lifecycle_ci_hardening_v15_signature") {
		t.Fatalf("tampered v15 lifecycle CI hardening authority was accepted: %v", findings)
	}
}

func TestW001LifecycleCIHardeningV15PathScope(t *testing.T) {
	for _, path := range []string{
		w001LifecycleCIHardeningV15Path,
		w001LifecycleCIHardeningV15Signature,
		".harness/manifest.yaml",
		canonicalActivePlan,
		"docs/evidence/W-001-validation.md",
		"internal/doctrine/grant.go",
		"internal/doctrine/grant_test.go",
	} {
		if !w001LifecycleCIHardeningV15PathsAllowed([]string{path}) {
			t.Fatalf("authorized v15 lifecycle CI hardening path was rejected: %s", path)
		}
	}
	for _, path := range []string{
		w001LifecycleCIHardeningV14Path,
		".github/workflows/foundation-quality.yml",
		"internal/authority/beads/store.go",
		"go.mod",
	} {
		if w001LifecycleCIHardeningV15PathsAllowed([]string{path}) {
			t.Fatalf("out-of-scope v15 lifecycle CI hardening path was accepted: %s", path)
		}
	}
}

func TestW001LifecycleCIHardeningV16GrantAcceptsPinnedSignedContract(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	var findings []Finding
	checkW001LifecycleCIHardeningV16Grant(repo, &findings)
	if len(findings) != 0 {
		t.Fatalf("valid signed W-001 v16 lifecycle CI hardening was rejected: %v", findings)
	}
}

func TestW001LifecycleCIHardeningV16GrantFailsClosed(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	read := func(path string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	source, err := filepath.Abs(repo)
	if err != nil {
		t.Fatal(err)
	}
	root := planningGrantCanonicalTempDir(t)
	runPlanningGrantTestGit(t, root, "init", "--quiet")
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, w001LifecycleCIHardeningV16Base)
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source,
		"refs/tags/"+w001LifecycleCIHardeningV15ReviewTag+":refs/tags/"+w001LifecycleCIHardeningV15ReviewTag)
	tampered := bytes.Replace(read(w001LifecycleCIHardeningV16Path), []byte("canonicalLifecycleMutationAllowed: false"), []byte("canonicalLifecycleMutationAllowed: true"), 1)
	for path, data := range map[string][]byte{
		w001LifecycleCIHardeningV16Path:      tampered,
		w001LifecycleCIHardeningV16Signature: read(w001LifecycleCIHardeningV16Signature),
		w001LifecycleCIHardeningV15Path:      read(w001LifecycleCIHardeningV15Path),
		w001LifecycleCIHardeningV15Signature: read(w001LifecycleCIHardeningV15Signature),
		wave1PlanningGrantKey:                read(wave1PlanningGrantKey),
		"docs/evidence/W-001-validation.md":  read("docs/evidence/W-001-validation.md"),
		canonicalActivePlan:                  read(canonicalActivePlan),
		".harness/manifest.yaml":             read(".harness/manifest.yaml"),
		"internal/doctrine/grant_test.go":    read("internal/doctrine/grant_test.go"),
	} {
		writePlanningGrantTestFile(t, root, path, data)
	}
	var findings []Finding
	checkW001LifecycleCIHardeningV16Grant(root, &findings)
	if !findingCodePresent(findings, "public.w001_lifecycle_ci_hardening_v16_value") ||
		!findingCodePresent(findings, "public.w001_lifecycle_ci_hardening_v16_signature") {
		t.Fatalf("tampered v16 lifecycle CI hardening authority was accepted: %v", findings)
	}
}

func TestW001LifecycleCIHardeningV16PathScope(t *testing.T) {
	for _, path := range []string{
		w001LifecycleCIHardeningV16Path,
		w001LifecycleCIHardeningV16Signature,
		".harness/manifest.yaml",
		canonicalActivePlan,
		"docs/evidence/W-001-validation.md",
		"internal/doctrine/grant.go",
		"internal/doctrine/grant_test.go",
	} {
		if !w001LifecycleCIHardeningV16PathsAllowed([]string{path}) {
			t.Fatalf("authorized v16 lifecycle CI hardening path was rejected: %s", path)
		}
	}
	for _, path := range []string{
		w001LifecycleCIHardeningV15Path,
		".github/workflows/foundation-quality.yml",
		"internal/authority/beads/store.go",
		"go.mod",
	} {
		if w001LifecycleCIHardeningV16PathsAllowed([]string{path}) {
			t.Fatalf("out-of-scope v16 lifecycle CI hardening path was accepted: %s", path)
		}
	}
}

func TestW001LifecycleCIHardeningV17GrantAcceptsPinnedSignedContract(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	var findings []Finding
	checkW001LifecycleCIHardeningV17Grant(repo, &findings)
	if len(findings) != 0 {
		t.Fatalf("valid signed W-001 v17 lifecycle CI hardening was rejected: %v", findings)
	}
}

func TestW001LifecycleCIHardeningV17GrantFailsClosed(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	read := func(path string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	source, err := filepath.Abs(repo)
	if err != nil {
		t.Fatal(err)
	}
	root := planningGrantCanonicalTempDir(t)
	runPlanningGrantTestGit(t, root, "init", "--quiet")
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, w001LifecycleCIHardeningV17Base)
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source,
		"refs/tags/"+w001LifecycleCIHardeningV16ReviewTag+":refs/tags/"+w001LifecycleCIHardeningV16ReviewTag)
	tampered := bytes.Replace(read(w001LifecycleCIHardeningV17Path), []byte("canonicalLifecycleMutationAllowed: false"), []byte("canonicalLifecycleMutationAllowed: true"), 1)
	for path, data := range map[string][]byte{
		w001LifecycleCIHardeningV17Path:      tampered,
		w001LifecycleCIHardeningV17Signature: read(w001LifecycleCIHardeningV17Signature),
		w001LifecycleCIHardeningV16Path:      read(w001LifecycleCIHardeningV16Path),
		w001LifecycleCIHardeningV16Signature: read(w001LifecycleCIHardeningV16Signature),
		wave1PlanningGrantKey:                read(wave1PlanningGrantKey),
		"docs/evidence/W-001-validation.md":  read("docs/evidence/W-001-validation.md"),
		canonicalActivePlan:                  read(canonicalActivePlan),
		".harness/manifest.yaml":             read(".harness/manifest.yaml"),
		"internal/doctrine/grant_test.go":    read("internal/doctrine/grant_test.go"),
	} {
		writePlanningGrantTestFile(t, root, path, data)
	}
	var findings []Finding
	checkW001LifecycleCIHardeningV17Grant(root, &findings)
	if !findingCodePresent(findings, "public.w001_lifecycle_ci_hardening_v17_value") ||
		!findingCodePresent(findings, "public.w001_lifecycle_ci_hardening_v17_signature") {
		t.Fatalf("tampered v17 lifecycle CI hardening authority was accepted: %v", findings)
	}
}

func TestW001LifecycleCIHardeningV17PathScope(t *testing.T) {
	for _, path := range []string{
		w001LifecycleCIHardeningV17Path,
		w001LifecycleCIHardeningV17Signature,
		".harness/manifest.yaml",
		canonicalActivePlan,
		"docs/evidence/W-001-validation.md",
		"internal/doctrine/grant.go",
		"internal/doctrine/grant_test.go",
	} {
		if !w001LifecycleCIHardeningV17PathsAllowed([]string{path}) {
			t.Fatalf("authorized v17 lifecycle CI hardening path was rejected: %s", path)
		}
	}
	for _, path := range []string{
		w001LifecycleCIHardeningV16Path,
		".github/workflows/foundation-quality.yml",
		"internal/authority/beads/store.go",
		"go.mod",
	} {
		if w001LifecycleCIHardeningV17PathsAllowed([]string{path}) {
			t.Fatalf("out-of-scope v17 lifecycle CI hardening path was accepted: %s", path)
		}
	}
}

func TestW001LifecycleAuthorityRecoveryGrantAcceptsPinnedSignedContract(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	var findings []Finding
	checkW001LifecycleAuthorityRecoveryGrant(repo, &findings)
	if len(findings) != 0 {
		t.Fatalf("valid signed W-001 lifecycle authority recovery was rejected: %v", findings)
	}
}

func TestW001LifecycleAuthorityRecoveryPathScope(t *testing.T) {
	for _, path := range []string{
		w001LifecycleAuthorityRecoveryPath,
		w001LifecycleAuthorityRecoverySignature,
		".harness/manifest.yaml",
		canonicalActivePlan,
		"docs/evidence/W-001-validation.md",
		"internal/doctrine/grant.go",
		"internal/doctrine/grant_test.go",
	} {
		if !w001LifecycleAuthorityRecoveryPathsAllowed([]string{path}) {
			t.Fatalf("authorized lifecycle authority-recovery path was rejected: %s", path)
		}
	}
	for _, path := range []string{
		w001LifecycleTestHarnessRetirementPath,
		".github/workflows/foundation-quality.yml",
		"internal/authority/beads/store.go",
		"go.mod",
	} {
		if w001LifecycleAuthorityRecoveryPathsAllowed([]string{path}) {
			t.Fatalf("out-of-scope lifecycle authority-recovery path was accepted: %s", path)
		}
	}
}

func TestW001LifecycleEvidencePreservationGrantAcceptsPinnedSignedContract(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	var findings []Finding
	checkW001LifecycleEvidencePreservationGrant(repo, &findings)
	if len(findings) != 0 {
		t.Fatalf("valid signed W-001 lifecycle evidence preservation was rejected: %v", findings)
	}
}

func TestW001LifecycleEvidencePreservationPathScope(t *testing.T) {
	for _, path := range []string{
		w001LifecycleEvidencePreservationPath,
		w001LifecycleEvidencePreservationSignature,
		".harness/manifest.yaml",
		canonicalActivePlan,
		"docs/evidence/W-001-validation.md",
		"internal/doctrine/grant.go",
		"internal/doctrine/grant_test.go",
	} {
		if !w001LifecycleEvidencePreservationPathsAllowed([]string{path}) {
			t.Fatalf("authorized lifecycle evidence-preservation path was rejected: %s", path)
		}
	}
	for _, path := range []string{
		w001LifecycleAuthorityRecoveryPath,
		".github/workflows/foundation-quality.yml",
		"internal/authority/beads/store.go",
		"go.mod",
	} {
		if w001LifecycleEvidencePreservationPathsAllowed([]string{path}) {
			t.Fatalf("out-of-scope lifecycle evidence-preservation path was accepted: %s", path)
		}
	}
}

func TestW001TerminalReconciliationGrantAcceptsPinnedSignedContract(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	var findings []Finding
	checkW001TerminalReconciliationGrant(repo, &findings)
	if len(findings) != 0 {
		t.Fatalf("valid signed W-001 terminal reconciliation was rejected: %v", findings)
	}
	grant, err := LoadW001TerminalReconciliationGrant(repo, time.Date(2026, 8, 29, 11, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("LoadW001TerminalReconciliationGrant: %v", err)
	}
	if grant.ID != "W-001-lifecycle-terminal-reconciliation-v1" || grant.Bead != "M3-W001" ||
		grant.BaseCommit != w001TerminalReconciliationBase || grant.AcceptedCandidateHead != "56c2a8d95927bc552882aacc30aa886ea0be9ba5" ||
		grant.ExpectedVersion.IssueMutationSequence != 1 || grant.ExpectedVersion.DependencyGraphRevision != 1 {
		t.Fatalf("terminal grant projection=%#v", grant)
	}
}

func TestW001TerminalReconciliationPathScope(t *testing.T) {
	for _, path := range []string{
		w001TerminalReconciliationPath, w001TerminalReconciliationSignature, ".harness/manifest.yaml", canonicalActivePlan,
		"docs/evidence/W-001-validation.md", "internal/doctrine/grant.go", "internal/doctrine/grant_test.go",
		"internal/authority/closeout/closeout.go", "internal/authority/closeout/closeout_test.go",
		"cmd/mars3-authority/main.go", "cmd/mars3-authority/main_test.go",
	} {
		if !w001TerminalReconciliationPathsAllowed([]string{path}) {
			t.Fatalf("authorized terminal path rejected: %s", path)
		}
	}
	for _, path := range []string{".github/workflows/foundation-quality.yml", "go.mod", "api/authority/v1/types.go", "internal/authority/gateway/lifecycle.go", "database/authority/001_work_authority.sql"} {
		if w001TerminalReconciliationPathsAllowed([]string{path}) {
			t.Fatalf("out-of-scope terminal path accepted: %s", path)
		}
	}
}

func TestW001TerminalCIRecoveryGrantAcceptsPinnedSignedContract(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	var findings []Finding
	checkW001TerminalCIRecoveryGrant(repo, &findings)
	if len(findings) != 0 {
		t.Fatalf("valid signed W-001 terminal CI recovery was rejected: %v", findings)
	}
	grant, err := LoadW001TerminalReconciliationGrant(repo, time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("LoadW001TerminalReconciliationGrant: %v", err)
	}
	if grant.ReviewTag != w001TerminalTagIdentityRecoveryReviewTag {
		t.Fatalf("terminal recovery review tag=%q", grant.ReviewTag)
	}
}

func TestW001TerminalCIRecoveryPathScope(t *testing.T) {
	for _, path := range w001TerminalCIRecoverySequences["grant.authorizedPaths"] {
		if !w001TerminalCIRecoveryPathsAllowed([]string{path}) {
			t.Fatalf("authorized terminal CI-recovery path rejected: %s", path)
		}
	}
	for _, path := range []string{"internal/authority/closeout/closeout.go", "cmd/mars3-authority/main.go", ".github/workflows/foundation-quality.yml", "go.mod"} {
		if w001TerminalCIRecoveryPathsAllowed([]string{path}) {
			t.Fatalf("out-of-scope terminal CI-recovery path accepted: %s", path)
		}
	}
}

func TestW001TerminalHistoryScanRecoveryGrantAcceptsPinnedSignedContract(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	var findings []Finding
	checkW001TerminalHistoryScanRecoveryGrant(repo, &findings)
	if len(findings) != 0 {
		t.Fatalf("valid signed W-001 terminal history-scan recovery was rejected: %v", findings)
	}
}

func TestW001TerminalHistoryScanRecoveryPathScope(t *testing.T) {
	for _, path := range w001TerminalHistoryScanRecoverySequences["grant.authorizedPaths"] {
		if !w001TerminalHistoryScanRecoveryPathsAllowed([]string{path}) {
			t.Fatalf("authorized terminal history-scan recovery path rejected: %s", path)
		}
	}
	for _, path := range []string{
		"internal/authority/closeout/closeout.go", "internal/authority/closeout/closeout_test.go",
		"cmd/mars3-authority/main.go", ".github/workflows/foundation-quality.yml", "go.mod",
	} {
		if w001TerminalHistoryScanRecoveryPathsAllowed([]string{path}) {
			t.Fatalf("out-of-scope terminal history-scan recovery path accepted: %s", path)
		}
	}
}

func TestW001TerminalTagIdentityRecoveryGrantAcceptsPinnedSignedContract(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	var findings []Finding
	checkW001TerminalTagIdentityRecoveryGrant(repo, &findings)
	if len(findings) != 0 {
		t.Fatalf("valid signed W-001 terminal tag-identity recovery was rejected: %v", findings)
	}
	grant, err := LoadW001TerminalReconciliationGrant(repo, time.Date(2026, 8, 29, 17, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("LoadW001TerminalReconciliationGrant: %v", err)
	}
	if grant.ReviewTag != w001TerminalTagIdentityRecoveryReviewTag {
		t.Fatalf("terminal tag-identity recovery review tag=%q", grant.ReviewTag)
	}
}

func TestW001TerminalTagIdentityRecoveryPathScope(t *testing.T) {
	var authorized []string
	for _, path := range w001TerminalTagIdentityRecoverySequences["grant.authorizedPaths"] {
		if !w001TerminalTagIdentityRecoveryPathsAllowed([]string{path}) {
			t.Fatalf("authorized terminal tag-identity recovery path rejected: %s", path)
		}
		authorized = append(authorized, path)
	}
	if !w001TerminalTagIdentityRecoveryPathsAllowed(authorized) {
		t.Fatal("exact seven-path terminal tag-identity recovery scope was rejected")
	}
	for _, path := range []string{
		"internal/authority/closeout/closeout.go", "cmd/mars3-authority/main.go",
		".github/workflows/foundation-quality.yml", "go.mod", "internal/authority/gateway/lifecycle.go",
	} {
		if w001TerminalTagIdentityRecoveryPathsAllowed([]string{path}) {
			t.Fatalf("out-of-scope terminal tag-identity recovery path accepted: %s", path)
		}
	}
}

func TestW001TerminalV2TagIdentityIsHistoricalOnly(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	object := planningGrantTestGitRawOutput(t, repo, "cat-file", "tag", w001TerminalV2TagObject)
	publicKey, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(wave1PlanningGrantKey)))
	if err != nil {
		t.Fatal(err)
	}
	target, err := verifyPinnedPlanningGrantTagForIdentity(object, publicKey, w001TerminalCIRecoveryReviewTag, w001TerminalCIRecoveryTagMessage, "engineer@example.com")
	if err != nil || target != w001TerminalTagIdentityRecoveryBase {
		t.Fatalf("authorized historical terminal v2 Engineer tag was rejected: target=%q err=%v", target, err)
	}
	if _, err := verifyPinnedPlanningGrantTag(object, publicKey, w001TerminalCIRecoveryReviewTag, w001TerminalCIRecoveryTagMessage); err == nil {
		t.Fatal("historical terminal v2 Engineer tag was accepted as a Release Manager review tag")
	}
}

func TestW001TerminalHistoryScannerFingerprintIsExactAndImmutable(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	var findings []Finding
	checkW001DeliveryScannerFingerprintSources(repo, []string{w001TerminalHistoryScannerFingerprint}, &findings)
	if len(findings) != 0 {
		t.Fatalf("exact terminal history scanner fingerprint was rejected: %v", findings)
	}

	wrongLine := strings.TrimSuffix(w001TerminalHistoryScannerFingerprint, ":231") + ":232"
	findings = nil
	checkW001DeliveryScannerFingerprintSources(repo, []string{wrongLine}, &findings)
	if !findingCodePresent(findings, "public.w001_delivery_scanner_history") {
		t.Fatalf("non-authorized terminal history scanner tuple was accepted: %v", findings)
	}
}

func TestW001TerminalExecutionAuthorizationRequiresCanonicalJSON(t *testing.T) {
	authorization := W001TerminalReconciliationExecutionAuthorization{
		SchemaVersion: 1, Kind: "MARS3W001TerminalReconciliationExecutionAuthorization", Classification: "PUBLIC",
		GrantID: "W-001-lifecycle-terminal-reconciliation-v1", Repository: planningGrantRepository,
		AttemptID: "w001-lifecycle-terminal-reconciliation-v1", Bead: "M3-W001", TenantID: "tenant-academy", ProjectID: "project-mars3",
		ReviewTag: w001TerminalReconciliationReviewTag, ReviewTagObject: strings.Repeat("f", 40), ReviewedFeatureCommit: strings.Repeat("a", 40), PullRequest: 11,
		MergedCommit: strings.Repeat("b", 40), MergedTree: strings.Repeat("c", 40), ProtectedMainCheckRun: 1,
		QAReviewedCommit: strings.Repeat("a", 40), QADisposition: "accepted", SecurityReviewedCommit: strings.Repeat("a", 40), SecurityDisposition: "accepted",
		BeadsBinarySHA256: strings.Repeat("d", 64), WorkspaceInstanceSHA256: strings.Repeat("e", 64), FenceGeneration: "generation-terminal",
		AllowedEffect: "execute-one-gateway-only-W001-terminal-reconciliation", IssuedAt: "2026-08-29T12:00:00Z", ExpiresAt: "2026-08-29T13:00:00Z",
	}
	data, err := json.Marshal(authorization)
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	decoded, err := decodeW001TerminalExecutionAuthorization(data)
	if err != nil || decoded != authorization {
		t.Fatalf("decoded=%#v err=%v", decoded, err)
	}
	if _, err := decodeW001TerminalExecutionAuthorization(bytes.Replace(data, []byte(`"schemaVersion":1`), []byte(`"schemaVersion":1,"extra":true`), 1)); err == nil {
		t.Fatal("unknown execution-authorization field was accepted")
	}
	if _, err := decodeW001TerminalExecutionAuthorization(bytes.TrimSpace(data)); err == nil {
		t.Fatal("noncanonical execution authorization was accepted")
	}
}

func TestW001TerminalExecutionAuthorizationWindowIsProspectiveAndBounded(t *testing.T) {
	issuedAt := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	expiresAt := issuedAt.Add(time.Hour)
	for _, test := range []struct {
		name string
		now  time.Time
		want bool
	}{
		{name: "nanosecond before issuance", now: issuedAt.Add(-time.Nanosecond), want: false},
		{name: "exactly at issuance", now: issuedAt, want: true},
		{name: "inside window", now: expiresAt.Add(-time.Nanosecond), want: true},
		{name: "exactly at expiry", now: expiresAt, want: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := validW001TerminalExecutionWindow(test.now, issuedAt, expiresAt); got != test.want {
				t.Fatalf("validW001TerminalExecutionWindow(%s)=%t want=%t", test.now, got, test.want)
			}
		})
	}
	if validW001TerminalExecutionWindow(issuedAt, issuedAt, expiresAt.Add(time.Nanosecond)) {
		t.Fatal("execution window longer than one hour was accepted")
	}
}

func TestW001LifecycleArchivedRejectedRetirementTagIsDurableAndUnaccepted(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	var findings []Finding
	if !checkW001LifecycleArchivedRejectedRetirementTag(repo, &findings) || len(findings) != 0 {
		t.Fatalf("exact rejected retirement tag was not preserved: %v", findings)
	}
	if w001LifecycleRejectedRetirementArchiveTag == w001LifecycleEvidencePreservationReviewTag ||
		w001LifecycleRejectedRetirementArchiveTag == w001LifecycleTestHarnessRetirementReviewTag ||
		w001LifecycleRejectedRetirementArchiveTag == w001LifecycleAuthorityRecoveryReviewTag {
		t.Fatal("rejected evidence archival ref was selected as a review tag")
	}
	object := planningGrantTestGitOutput(t, repo, "rev-parse", "refs/tags/"+w001LifecycleRejectedRetirementArchiveTag+"^{tag}")
	if object != w001LifecycleRejectedRetirementTagObject {
		t.Fatalf("archival ref resolved to %s, want %s", object, w001LifecycleRejectedRetirementTagObject)
	}
}

func TestW001DeliveryV2TagIdentityIsHistoricalOnly(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	object := planningGrantTestGitRawOutput(t, repo, "cat-file", "tag", w001DeliveryV2TagObject)
	publicKey, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(wave1PlanningGrantKey)))
	if err != nil {
		t.Fatal(err)
	}
	target, err := verifyPinnedPlanningGrantTagForIdentity(object, publicKey, w001DeliveryReviewTag, w001DeliveryReviewTagMessage, "engineer@example.com")
	if err != nil || target != w001DeliveryCIFixBase {
		t.Fatalf("authorized historical Engineer tag was rejected: target=%q err=%v", target, err)
	}
	if _, err := verifyPinnedPlanningGrantTag(object, publicKey, w001DeliveryReviewTag, w001DeliveryReviewTagMessage); err == nil {
		t.Fatal("historical Engineer tag was accepted as a Release Manager review tag")
	}
}

func TestW001DeliveryPullRequestCheckoutBindsEventHead(t *testing.T) {
	repo, err := filepath.Abs(filepath.Clean(filepath.Join("..", "..")))
	if err != nil {
		t.Fatal(err)
	}
	parent := planningGrantCanonicalTempDir(t)
	root := filepath.Join(parent, "repo")
	feature := initializePlanningGrantTestRepository(t, root, repo, "HEAD")
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", repo, w001DeliveryBase)
	tree := planningGrantTestGitOutput(t, root, "rev-parse", feature+"^{tree}")
	merge := planningGrantTestGitOutput(t, root,
		"-c", "user.name=Synthetic Merge Bot",
		"-c", "user.email=merge-bot@example.com",
		"-c", "commit.gpgsign=false",
		"commit-tree", tree,
		"-p", w001DeliveryBase,
		"-p", feature,
		"-m", "synthetic W-001 delivery merge",
	)
	runPlanningGrantTestGit(t, root, "checkout", "--quiet", "--force", "--detach", merge)
	event := map[string]any{
		"number":     9,
		"repository": map[string]any{"full_name": planningGrantRepository},
		"pull_request": map[string]any{
			"base":             map[string]any{"ref": "main", "sha": w001DeliveryBase},
			"head":             map[string]any{"ref": w001DeliveryBranch, "sha": feature},
			"merge_commit_sha": merge,
		},
	}
	eventPath := writePlanningGrantGitHubEvent(t, event)
	setPlanningGrantCommonGitHubFacts(t, root, merge, eventPath)
	t.Setenv("GITHUB_EVENT_NAME", "pull_request")
	t.Setenv("GITHUB_REF", "refs/pull/9/merge")
	t.Setenv("GITHUB_HEAD_REF", w001DeliveryBranch)
	t.Setenv("GITHUB_BASE_REF", "main")
	t.Setenv("GITHUB_REF_PROTECTED", "false")
	t.Setenv("GITHUB_WORKFLOW_REF", planningGrantRepository+"/"+planningGrantWorkflowPath+"@refs/pull/9/merge")
	var findings []Finding
	actual, requireTag, mainTree := w001DeliveryGitHubCheckout(root, merge, "", &findings)
	if len(findings) != 0 || actual != feature || !requireTag || mainTree {
		t.Fatalf("canonical PR checkout was not bound to its feature head: head=%q tag=%t main=%t findings=%v", actual, requireTag, mainTree, findings)
	}

	t.Run("event head mismatch fails", func(t *testing.T) {
		event["pull_request"].(map[string]any)["head"] = map[string]any{"ref": w001DeliveryBranch, "sha": w001DeliveryBase}
		badEventPath := writePlanningGrantGitHubEvent(t, event)
		t.Setenv("GITHUB_EVENT_PATH", badEventPath)
		var rejected []Finding
		if actual, _, _ := w001DeliveryGitHubCheckout(root, merge, "", &rejected); actual != "" || !findingCodePresent(rejected, "public.w001_delivery_pr_topology") {
			t.Fatalf("event-head mismatch was accepted: head=%q findings=%v", actual, rejected)
		}
	})

	t.Run("wrong parent order fails", func(t *testing.T) {
		wrong := planningGrantTestGitOutput(t, root,
			"-c", "user.name=Synthetic Merge Bot",
			"-c", "user.email=merge-bot@example.com",
			"-c", "commit.gpgsign=false",
			"commit-tree", tree,
			"-p", feature,
			"-p", w001DeliveryBase,
			"-m", "wrong-order W-001 merge",
		)
		runPlanningGrantTestGit(t, root, "checkout", "--quiet", "--force", "--detach", wrong)
		event["pull_request"].(map[string]any)["head"] = map[string]any{"ref": w001DeliveryBranch, "sha": feature}
		event["pull_request"].(map[string]any)["merge_commit_sha"] = wrong
		wrongEventPath := writePlanningGrantGitHubEvent(t, event)
		t.Setenv("GITHUB_SHA", wrong)
		t.Setenv("GITHUB_EVENT_PATH", wrongEventPath)
		var rejected []Finding
		if actual, _, _ := w001DeliveryGitHubCheckout(root, wrong, "", &rejected); actual != "" || !findingCodePresent(rejected, "public.w001_delivery_pr_topology") {
			t.Fatalf("wrong-parent synthetic merge was accepted: head=%q findings=%v", actual, rejected)
		}
	})

	t.Run("wrong tree fails", func(t *testing.T) {
		baseTree := planningGrantTestGitOutput(t, root, "rev-parse", w001DeliveryBase+"^{tree}")
		wrong := planningGrantTestGitOutput(t, root,
			"-c", "user.name=Synthetic Merge Bot",
			"-c", "user.email=merge-bot@example.com",
			"-c", "commit.gpgsign=false",
			"commit-tree", baseTree,
			"-p", w001DeliveryBase,
			"-p", feature,
			"-m", "wrong-tree W-001 merge",
		)
		runPlanningGrantTestGit(t, root, "checkout", "--quiet", "--force", "--detach", wrong)
		event["pull_request"].(map[string]any)["head"] = map[string]any{"ref": w001DeliveryBranch, "sha": feature}
		event["pull_request"].(map[string]any)["merge_commit_sha"] = wrong
		wrongEventPath := writePlanningGrantGitHubEvent(t, event)
		t.Setenv("GITHUB_SHA", wrong)
		t.Setenv("GITHUB_EVENT_PATH", wrongEventPath)
		var rejected []Finding
		if actual, _, _ := w001DeliveryGitHubCheckout(root, wrong, "", &rejected); actual != "" || !findingCodePresent(rejected, "public.w001_delivery_pr_tree") {
			t.Fatalf("wrong-tree synthetic merge was accepted: head=%q findings=%v", actual, rejected)
		}
	})
}

func TestW001DeliveryPathScope(t *testing.T) {
	for _, path := range []string{
		w001DeliveryGrantPath,
		"internal/authority/gateway/service.go",
		"cmd/mars3-authority/main.go",
		"api/authority/v1/types.go",
		"database/authority/001_leases.sql",
		"deploy/authority/network-policy.yaml",
		"go.mod",
	} {
		if !w001DeliveryPathsAllowed([]string{path}) {
			t.Fatalf("signed W-001 delivery path was rejected: %s", path)
		}
	}
	for _, path := range []string{"internal/runtime/escape.go", "internal/authority", "docs/features/F-002-work-authority.md", ".github/workflows/foundation-quality.yml"} {
		if w001DeliveryPathsAllowed([]string{path}) {
			t.Fatalf("out-of-scope delivery path was accepted: %s", path)
		}
	}
	for _, path := range []string{w001DeliveryCIFixPath, w001DeliveryCIFixSignature, "docs/evidence/W-001-validation.md", "internal/doctrine/grant.go", "internal/doctrine/grant_test.go"} {
		if !w001DeliveryCIFixPathsAllowed([]string{path}) {
			t.Fatalf("signed CI-correction path was rejected: %s", path)
		}
	}
	for _, path := range []string{"internal/authority/gateway/service.go", ".github/workflows/foundation-quality.yml", "go.mod"} {
		if w001DeliveryCIFixPathsAllowed([]string{path}) {
			t.Fatalf("out-of-scope CI-correction path was accepted: %s", path)
		}
	}
	for _, path := range []string{w001DeliveryScannerIgnorePath, w001DeliveryScannerFixPath, w001DeliveryScannerFixSignature, "docs/evidence/W-001-validation.md", "internal/doctrine/grant.go", "internal/doctrine/grant_test.go", "internal/doctrine/public.go", "internal/doctrine/doctrine_test.go"} {
		if !w001DeliveryScannerFixPathsAllowed([]string{path}) {
			t.Fatalf("signed scanner-correction path was rejected: %s", path)
		}
	}
	for _, path := range []string{".github/workflows/foundation-quality.yml", "internal/authority/gateway/service.go", "go.mod"} {
		if w001DeliveryScannerFixPathsAllowed([]string{path}) {
			t.Fatalf("out-of-scope scanner-correction path was accepted: %s", path)
		}
	}
}

func TestW001DeliveryGitDiffIsFenced(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	var findings []Finding
	checkW001DeliveryGrantGitDiff(repo, &findings)
	if len(findings) != 0 {
		t.Fatalf("current signed W-001 delivery checkout was rejected: %v", findings)
	}
}

func clonePlanningGrantMaterials(materials map[string][]byte) map[string][]byte {
	cloned := make(map[string][]byte, len(materials))
	for path, data := range materials {
		cloned[path] = append([]byte(nil), data...)
	}
	return cloned
}

func TestW001PostclaimGitDiffIsFenced(t *testing.T) {
	t.Run("repository fixture ignores inherited GitHub runner identity", func(t *testing.T) {
		t.Setenv("GITHUB_ACTIONS", "true")
		root := writeW001PostclaimGitFixture(t)
		var findings []Finding
		checkW001PostclaimGrantGitDiff(root, &findings)
		if len(findings) != 0 {
			t.Fatalf("fixture-local Git reconciliation inherited ambient runner identity: %v", findings)
		}
	})

	t.Run("exact dirty reconciliation passes", func(t *testing.T) {
		root := writeW001PostclaimGitFixture(t)
		var findings []Finding
		checkW001PostclaimGrantGitDiff(root, &findings)
		if len(findings) != 0 {
			t.Fatalf("authorized postclaim working tree was rejected: %v", findings)
		}
	})

	t.Run("wrong branch fails", func(t *testing.T) {
		root := writeW001PostclaimGitFixture(t)
		runPlanningGrantTestGit(t, root, "branch", "-m", "codex/postclaim-copy")
		var findings []Finding
		checkW001PostclaimGrantGitDiff(root, &findings)
		if !findingCodePresent(findings, "public.w001_postclaim_branch") {
			t.Fatalf("wrong postclaim branch was accepted: %v", findings)
		}
	})

	t.Run("unauthorized live path fails", func(t *testing.T) {
		root := writeW001PostclaimGitFixture(t)
		writePlanningGrantTestFile(t, root, "internal/authority/escape.go", []byte("package authority\n"))
		var findings []Finding
		checkW001PostclaimGrantGitDiff(root, &findings)
		if !findingCodePresent(findings, "public.w001_postclaim_scope") {
			t.Fatalf("unauthorized postclaim path was accepted: %v", findings)
		}
	})

	t.Run("unsigned authorized commit fails", func(t *testing.T) {
		root := writeW001PostclaimGitFixture(t)
		commitPlanningGrantTestPaths(t, root, "unsigned postclaim reconciliation", w001PostclaimCIFixPath)
		var findings []Finding
		checkW001PostclaimGrantGitDiff(root, &findings)
		if !findingCodePresent(findings, "public.w001_postclaim_commit_signature") {
			t.Fatalf("unsigned postclaim commit was accepted: %v", findings)
		}
	})

	t.Run("transient unauthorized commit remains visible", func(t *testing.T) {
		root := writeW001PostclaimGitFixture(t)
		const unauthorized = "internal/authority/escape.go"
		writePlanningGrantTestFile(t, root, unauthorized, []byte("package authority\n"))
		commitPlanningGrantTestPaths(t, root, "add transient postclaim escape", unauthorized)
		if err := os.Remove(filepath.Join(root, filepath.FromSlash(unauthorized))); err != nil {
			t.Fatal(err)
		}
		commitPlanningGrantTestPaths(t, root, "delete transient postclaim escape", unauthorized)
		var findings []Finding
		checkW001PostclaimGrantGitDiff(root, &findings)
		if !findingCodePresent(findings, "public.w001_postclaim_scope") {
			t.Fatalf("transient unauthorized postclaim path was accepted: %v", findings)
		}
	})
}

func TestW001PostclaimSecurityGitDiffIsFenced(t *testing.T) {
	t.Run("exact dirty correction passes", func(t *testing.T) {
		root := writeW001PostclaimSecurityGitFixture(t)
		var findings []Finding
		checkW001PostclaimGrantGitDiff(root, &findings)
		if len(findings) != 0 {
			t.Fatalf("authorized Security correction was rejected: %v", findings)
		}
	})

	t.Run("unauthorized correction path fails", func(t *testing.T) {
		root := writeW001PostclaimSecurityGitFixture(t)
		writePlanningGrantTestFile(t, root, "internal/authority/escape.go", []byte("package authority\n"))
		var findings []Finding
		checkW001PostclaimGrantGitDiff(root, &findings)
		if !findingCodePresent(findings, "public.w001_postclaim_scope") {
			t.Fatalf("unauthorized Security-correction path was accepted: %v", findings)
		}
	})

	t.Run("missing prior v2 tag fails", func(t *testing.T) {
		root := writeW001PostclaimSecurityGitFixture(t)
		runPlanningGrantTestGit(t, root, "tag", "-d", w001PostclaimCIFixReviewTag)
		var findings []Finding
		checkW001PostclaimGrantGitDiff(root, &findings)
		if !findingCodePresent(findings, "public.w001_postclaim_security_v2_tag") {
			t.Fatalf("missing immutable v2 tag was accepted: %v", findings)
		}
	})
}

func TestW001PostclaimHookGitDiffIsFenced(t *testing.T) {
	t.Run("exact dirty correction passes", func(t *testing.T) {
		root := writeW001PostclaimHookGitFixture(t)
		var findings []Finding
		checkW001PostclaimGrantGitDiff(root, &findings)
		if len(findings) != 0 {
			t.Fatalf("authorized hook-isolation correction was rejected: %v", findings)
		}
	})

	t.Run("unauthorized correction path fails", func(t *testing.T) {
		root := writeW001PostclaimHookGitFixture(t)
		writePlanningGrantTestFile(t, root, "internal/authority/escape.go", []byte("package authority\n"))
		var findings []Finding
		checkW001PostclaimGrantGitDiff(root, &findings)
		if !findingCodePresent(findings, "public.w001_postclaim_scope") {
			t.Fatalf("unauthorized hook-isolation path was accepted: %v", findings)
		}
	})

	t.Run("missing prior v3 tag fails", func(t *testing.T) {
		root := writeW001PostclaimHookGitFixture(t)
		runPlanningGrantTestGit(t, root, "tag", "-d", w001PostclaimSecurityFixTag)
		var findings []Finding
		checkW001PostclaimGrantGitDiff(root, &findings)
		if !findingCodePresent(findings, "public.w001_postclaim_hook_v3_tag") {
			t.Fatalf("missing immutable v3 tag was accepted: %v", findings)
		}
	})
}

func TestW001PostclaimPRBindingGitDiffIsFenced(t *testing.T) {
	t.Run("exact dirty correction passes", func(t *testing.T) {
		root := writeW001PostclaimPRBindingGitFixture(t)
		var findings []Finding
		checkW001PostclaimGrantGitDiff(root, &findings)
		if len(findings) != 0 {
			t.Fatalf("authorized PR-binding correction was rejected: %v", findings)
		}
	})

	t.Run("unauthorized correction path fails", func(t *testing.T) {
		root := writeW001PostclaimPRBindingGitFixture(t)
		writePlanningGrantTestFile(t, root, "internal/authority/escape.go", []byte("package authority\n"))
		var findings []Finding
		checkW001PostclaimGrantGitDiff(root, &findings)
		if !findingCodePresent(findings, "public.w001_postclaim_scope") {
			t.Fatalf("unauthorized PR-binding path was accepted: %v", findings)
		}
	})

	t.Run("missing prior v4 tag fails", func(t *testing.T) {
		root := writeW001PostclaimPRBindingGitFixture(t)
		runPlanningGrantTestGit(t, root, "tag", "-d", w001PostclaimHookFixTag)
		var findings []Finding
		checkW001PostclaimGrantGitDiff(root, &findings)
		if !findingCodePresent(findings, "public.w001_postclaim_pr_binding_v4_tag") {
			t.Fatalf("missing immutable v4 tag was accepted: %v", findings)
		}
	})
}

func TestW001PostclaimChronologyGitDiffIsFenced(t *testing.T) {
	t.Run("exact dirty correction passes", func(t *testing.T) {
		root := writeW001PostclaimChronologyGitFixture(t)
		var findings []Finding
		checkW001PostclaimGrantGitDiff(root, &findings)
		if len(findings) != 0 {
			t.Fatalf("authorized chronology correction was rejected: %v", findings)
		}
	})

	t.Run("unauthorized correction path fails", func(t *testing.T) {
		root := writeW001PostclaimChronologyGitFixture(t)
		writePlanningGrantTestFile(t, root, "internal/authority/escape.go", []byte("package authority\n"))
		var findings []Finding
		checkW001PostclaimGrantGitDiff(root, &findings)
		if !findingCodePresent(findings, "public.w001_postclaim_scope") {
			t.Fatalf("unauthorized chronology path was accepted: %v", findings)
		}
	})

	t.Run("missing prior v5 tag fails", func(t *testing.T) {
		root := writeW001PostclaimChronologyGitFixture(t)
		runPlanningGrantTestGit(t, root, "tag", "-d", w001PostclaimPRFixTag)
		var findings []Finding
		checkW001PostclaimGrantGitDiff(root, &findings)
		if !findingCodePresent(findings, "public.w001_postclaim_chronology_v5_tag") {
			t.Fatalf("missing immutable v5 tag was accepted: %v", findings)
		}
	})

	t.Run("pre-effective v6 target fails", func(t *testing.T) {
		root := writeW001PostclaimChronologyGitFixture(t)
		runPlanningGrantTestGit(t, root, "-c", "user.name=Synthetic Release Manager", "-c", "user.email=release-manager@example.com", "tag", "-a", "-m", w001PostclaimChronoFixTagMsg, w001PostclaimChronoFixTag, w001PostclaimChronoFixBase)
		grant, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(w001PostclaimChronoFixPath)))
		if err != nil {
			t.Fatal(err)
		}
		document := parseStrictGrant(grant, w001PostclaimChronoFixScalars, w001PostclaimChronoFixSequences,
			[]string{"grant", "finding", "chronology", "publication", "materials", "verification", "integrity"})
		issued, _ := time.Parse(time.RFC3339, scalarValue(document, "grant.issuedAt"))
		var findings []Finding
		checkW001PostclaimChronology(root, document, issued, &findings)
		if !findingCodePresent(findings, "public.w001_postclaim_chronology_effect") {
			t.Fatalf("pre-effective v6 target was accepted: %v", findings)
		}
	})
}

func writeW001PostclaimGitFixture(t *testing.T) string {
	t.Helper()
	t.Setenv("GITHUB_ACTIONS", "")
	source, err := filepath.Abs(filepath.Clean(filepath.Join("..", "..")))
	if err != nil {
		t.Fatal(err)
	}
	root := planningGrantCanonicalTempDir(t)
	runPlanningGrantTestGit(t, root, "init", "--quiet")
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, w001PostclaimCIFixBase)
	runPlanningGrantTestGit(t, root, "checkout", "--quiet", "-b", w001PostclaimBranch, "FETCH_HEAD")
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, "refs/tags/"+w001PostclaimReviewTag+":refs/tags/"+w001PostclaimReviewTag)
	for _, path := range w001PostclaimCIFixSequences["addendum.authorizedPaths"] {
		data, err := os.ReadFile(filepath.Join(source, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		writePlanningGrantTestFile(t, root, path, data)
	}
	return root
}

func writeW001PostclaimSecurityGitFixture(t *testing.T) string {
	t.Helper()
	t.Setenv("GITHUB_ACTIONS", "")
	source, err := filepath.Abs(filepath.Clean(filepath.Join("..", "..")))
	if err != nil {
		t.Fatal(err)
	}
	root := planningGrantCanonicalTempDir(t)
	runPlanningGrantTestGit(t, root, "init", "--quiet")
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, w001PostclaimSecurityFixBase)
	runPlanningGrantTestGit(t, root, "checkout", "--quiet", "-b", w001PostclaimBranch, "FETCH_HEAD")
	for _, tag := range []string{w001PostclaimReviewTag, w001PostclaimCIFixReviewTag} {
		runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, "refs/tags/"+tag+":refs/tags/"+tag)
	}
	for _, path := range w001PostclaimSecurityFixSequences["grant.authorizedPaths"] {
		data, err := os.ReadFile(filepath.Join(source, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		writePlanningGrantTestFile(t, root, path, data)
	}
	return root
}

func writeW001PostclaimHookGitFixture(t *testing.T) string {
	t.Helper()
	t.Setenv("GITHUB_ACTIONS", "")
	source, err := filepath.Abs(filepath.Clean(filepath.Join("..", "..")))
	if err != nil {
		t.Fatal(err)
	}
	root := planningGrantCanonicalTempDir(t)
	runPlanningGrantTestGit(t, root, "init", "--quiet")
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, w001PostclaimHookFixBase)
	runPlanningGrantTestGit(t, root, "checkout", "--quiet", "-b", w001PostclaimBranch, "FETCH_HEAD")
	for _, tag := range []string{w001PostclaimReviewTag, w001PostclaimCIFixReviewTag, w001PostclaimSecurityFixTag} {
		runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, "refs/tags/"+tag+":refs/tags/"+tag)
	}
	for _, path := range w001PostclaimHookFixSequences["grant.authorizedPaths"] {
		data, err := os.ReadFile(filepath.Join(source, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		writePlanningGrantTestFile(t, root, path, data)
	}
	return root
}

func writeW001PostclaimPRBindingGitFixture(t *testing.T) string {
	t.Helper()
	t.Setenv("GITHUB_ACTIONS", "")
	source, err := filepath.Abs(filepath.Clean(filepath.Join("..", "..")))
	if err != nil {
		t.Fatal(err)
	}
	root := planningGrantCanonicalTempDir(t)
	runPlanningGrantTestGit(t, root, "init", "--quiet")
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, w001PostclaimPRFixBase)
	runPlanningGrantTestGit(t, root, "checkout", "--quiet", "-b", w001PostclaimBranch, "FETCH_HEAD")
	for _, tag := range []string{w001PostclaimReviewTag, w001PostclaimCIFixReviewTag, w001PostclaimSecurityFixTag, w001PostclaimHookFixTag} {
		runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, "refs/tags/"+tag+":refs/tags/"+tag)
	}
	for _, path := range w001PostclaimPRFixSequences["grant.authorizedPaths"] {
		data, err := os.ReadFile(filepath.Join(source, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		writePlanningGrantTestFile(t, root, path, data)
	}
	return root
}

func writeW001PostclaimChronologyGitFixture(t *testing.T) string {
	t.Helper()
	t.Setenv("GITHUB_ACTIONS", "")
	source, err := filepath.Abs(filepath.Clean(filepath.Join("..", "..")))
	if err != nil {
		t.Fatal(err)
	}
	root := planningGrantCanonicalTempDir(t)
	runPlanningGrantTestGit(t, root, "init", "--quiet")
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, w001PostclaimChronoFixBase)
	runPlanningGrantTestGit(t, root, "checkout", "--quiet", "-b", w001PostclaimBranch, "FETCH_HEAD")
	for _, tag := range []string{w001PostclaimReviewTag, w001PostclaimCIFixReviewTag, w001PostclaimSecurityFixTag, w001PostclaimHookFixTag, w001PostclaimPRFixTag} {
		runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, "refs/tags/"+tag+":refs/tags/"+tag)
	}
	for _, path := range w001PostclaimChronoFixSequences["grant.authorizedPaths"] {
		data, err := os.ReadFile(filepath.Join(source, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		writePlanningGrantTestFile(t, root, path, data)
	}
	return root
}

func scalarPathFromGrant(t *testing.T, grant []byte, key string) string {
	t.Helper()
	prefix := "  " + key + ": "
	for _, line := range strings.Split(string(grant), "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimPrefix(line, prefix)
		}
	}
	t.Fatalf("missing %s", key)
	return ""
}

func writeW001BootstrapGrantFixture(t *testing.T, grant, signature, publicKey []byte, materials map[string][]byte) string {
	t.Helper()
	root := planningGrantCanonicalTempDir(t)
	writePlanningGrantTestFile(t, root, w001BootstrapGrantPath, grant)
	writePlanningGrantTestFile(t, root, w001BootstrapGrantSignature, signature)
	writePlanningGrantTestFile(t, root, wave1PlanningGrantKey, publicKey)
	for path, data := range materials {
		writePlanningGrantTestFile(t, root, path, data)
	}
	return root
}

func TestWave1DirectMainTransitionAcceptsPinnedSignedContract(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	var findings []Finding
	checkWave1DirectMainTransitionGrant(repo, &findings)
	if len(findings) != 0 {
		t.Fatalf("valid signed direct-main transition was rejected: %v", findings)
	}
}

func TestWave1DirectMainTransitionRejectsTampering(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	read := func(path string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	grant := read(wave1DirectMainGrantPath)
	signature := read(wave1DirectMainGrantSignature)
	publicKey := read(wave1PlanningGrantKey)

	for _, testCase := range []struct {
		name string
		old  string
		new  string
		code string
	}{
		{name: "branch", old: "workingBranch: main", new: "workingBranch: codex/escape", code: "public.direct_main_transition_value"},
		{name: "authority", old: "canonicalWorkMutationAllowed: false", new: "canonicalWorkMutationAllowed: true", code: "public.direct_main_transition_value"},
		{name: "scope", old: "    - internal/doctrine/grant_test.go", new: "    - internal/runtime/escape.go", code: "public.direct_main_transition_sequence"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			tampered := bytes.Replace(grant, []byte(testCase.old), []byte(testCase.new), 1)
			root := planningGrantCanonicalTempDir(t)
			writePlanningGrantTestFile(t, root, wave1DirectMainGrantPath, tampered)
			writePlanningGrantTestFile(t, root, wave1DirectMainGrantSignature, signature)
			writePlanningGrantTestFile(t, root, wave1PlanningGrantKey, publicKey)
			var findings []Finding
			checkWave1DirectMainTransitionGrant(root, &findings)
			if !findingCodePresent(findings, testCase.code) || !findingCodePresent(findings, "public.direct_main_transition_signature") {
				t.Fatalf("tampered transition was not rejected by contract and signature: %v", findings)
			}
		})
	}
}

func TestWave1DirectMainTransitionAcceptsCurrentMainCheckout(t *testing.T) {
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		t.Skip("the top-level public gate exercises canonical GitHub push facts")
	}
	repo := filepath.Clean(filepath.Join("..", ".."))
	if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(w001BootstrapGrantPath))); err == nil {
		t.Skip("the signed W-001 bootstrap grant supersedes the historical direct-main checkout")
	}
	var findings []Finding
	checkWave1DirectMainTransitionGitDiff(repo, &findings)
	if len(findings) != 0 {
		t.Fatalf("current signed direct-main transition checkout was rejected: %v", findings)
	}
}

func TestWave1PRFallbackAcceptsPinnedSignedAddendum(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	var findings []Finding
	checkWave1PRFallbackAddendum(repo, &findings)
	if len(findings) != 0 {
		t.Fatalf("valid signed PR-fallback addendum was rejected: %v", findings)
	}
}

func TestWave1PRFallbackRejectsTampering(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	read := func(path string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	grant := read(wave1PRFallbackPath)
	signature := read(wave1PRFallbackSignature)
	publicKey := read(wave1PlanningGrantKey)
	for _, testCase := range []struct {
		name string
		old  string
		new  string
		code string
	}{
		{name: "retry", old: "retryDirectPush: false", new: "retryDirectPush: true", code: "public.pr_fallback_value"},
		{name: "branch", old: "workingBranch: codex/w-001-bootstrap-transition", new: "workingBranch: codex/escape", code: "public.pr_fallback_value"},
		{name: "scope", old: "    - internal/doctrine/grant_test.go", new: "    - internal/runtime/escape.go", code: "public.pr_fallback_sequence"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			tampered := bytes.Replace(grant, []byte(testCase.old), []byte(testCase.new), 1)
			root := planningGrantCanonicalTempDir(t)
			writePlanningGrantTestFile(t, root, wave1PRFallbackPath, tampered)
			writePlanningGrantTestFile(t, root, wave1PRFallbackSignature, signature)
			writePlanningGrantTestFile(t, root, wave1PlanningGrantKey, publicKey)
			var findings []Finding
			checkWave1PRFallbackAddendum(root, &findings)
			if !findingCodePresent(findings, testCase.code) || !findingCodePresent(findings, "public.pr_fallback_signature") {
				t.Fatalf("tampered PR fallback was not rejected by contract and signature: %v", findings)
			}
		})
	}
}

func TestWave1PRFallbackMainCIFixAcceptsPinnedSignedAddendum(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	var findings []Finding
	checkWave1MainCIFixAddendum(repo, &findings)
	if len(findings) != 0 {
		t.Fatalf("valid signed main-CI correction addendum was rejected: %v", findings)
	}
}

func TestWave1PRFallbackMainCIFixRejectsTampering(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	read := func(path string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	grant := read(wave1MainCIFixPath)
	signature := read(wave1MainCIFixSignature)
	publicKey := read(wave1PlanningGrantKey)
	for _, testCase := range []struct {
		name string
		old  string
		new  string
		code string
	}{
		{name: "v1 tag", old: "priorTransitionTagObject: 394c9ce749142c2222c1b8081b62f43a895be326", new: "priorTransitionTagObject: 1111111111111111111111111111111111111111", code: "public.pr_fallback_main_ci_value"},
		{name: "authority", old: "canonicalWorkMutationAllowed: false", new: "canonicalWorkMutationAllowed: true", code: "public.pr_fallback_main_ci_value"},
		{name: "scope", old: "    - internal/doctrine/grant_test.go", new: "    - internal/runtime/escape.go", code: "public.pr_fallback_main_ci_sequence"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			tampered := bytes.Replace(grant, []byte(testCase.old), []byte(testCase.new), 1)
			root := planningGrantCanonicalTempDir(t)
			writePlanningGrantTestFile(t, root, wave1MainCIFixPath, tampered)
			writePlanningGrantTestFile(t, root, wave1MainCIFixSignature, signature)
			writePlanningGrantTestFile(t, root, wave1PlanningGrantKey, publicKey)
			var findings []Finding
			checkWave1MainCIFixAddendum(root, &findings)
			if !findingCodePresent(findings, testCase.code) || !findingCodePresent(findings, "public.pr_fallback_main_ci_signature") {
				t.Fatalf("tampered main-CI correction was not rejected by contract and signature: %v", findings)
			}
		})
	}
}

func TestWave1PRFallbackFixtureFixAcceptsPinnedSignedAddendum(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	var findings []Finding
	checkWave1CIFixtureFixAddendum(repo, &findings)
	if len(findings) != 0 {
		t.Fatalf("valid signed fixture-stabilization addendum was rejected: %v", findings)
	}
}

func TestWave1PRFallbackFixtureFixRejectsTampering(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	read := func(path string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	grant := read(wave1CIFixtureFixPath)
	signature := read(wave1CIFixtureFixSignature)
	publicKey := read(wave1PlanningGrantKey)
	for _, testCase := range []struct {
		name string
		old  string
		new  string
		code string
	}{
		{name: "failed attempts", old: "failedAttempts: 1,2", new: "failedAttempts: 1", code: "public.pr_fallback_fixture_value"},
		{name: "production config", old: "mutate-production-or-developer-git-configuration", new: "mutate-production-git-configuration", code: "public.pr_fallback_fixture_sequence"},
		{name: "scope", old: "    - internal/doctrine/grant_test.go", new: "    - internal/runtime/escape.go", code: "public.pr_fallback_fixture_sequence"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			tampered := bytes.Replace(grant, []byte(testCase.old), []byte(testCase.new), 1)
			root := planningGrantCanonicalTempDir(t)
			writePlanningGrantTestFile(t, root, wave1CIFixtureFixPath, tampered)
			writePlanningGrantTestFile(t, root, wave1CIFixtureFixSignature, signature)
			writePlanningGrantTestFile(t, root, wave1PlanningGrantKey, publicKey)
			var findings []Finding
			checkWave1CIFixtureFixAddendum(root, &findings)
			if !findingCodePresent(findings, testCase.code) || !findingCodePresent(findings, "public.pr_fallback_fixture_signature") {
				t.Fatalf("tampered fixture stabilization was not rejected by contract and signature: %v", findings)
			}
		})
	}
}

func TestWave1PlanningGrantAcceptsPinnedSignedContract(t *testing.T) {
	grant, signature, publicKey := loadPlanningGrantFixture(t)
	root := writePlanningGrantFixture(t, grant, signature, publicKey)
	var findings []Finding
	checkWave1PlanningGrant(root, &findings)
	if len(findings) != 0 {
		t.Fatalf("valid signed planning grant was rejected: %v", findings)
	}
}

func TestWave1PlanningGrantFailsClosedOnSchemaAndContractChanges(t *testing.T) {
	grant, signature, publicKey := loadPlanningGrantFixture(t)
	tests := []struct {
		name        string
		mutate      func(string) string
		findingCode string
	}{
		{
			name: "missing critical field",
			mutate: func(value string) string {
				return strings.Replace(value, "  baseCommit: ee385ce236ae1f99da692d223d7666b80dd9108f\n", "", 1)
			},
			findingCode: "public.planning_grant_field",
		},
		{
			name: "duplicate critical field",
			mutate: func(value string) string {
				line := "  workingBranch: codex/w-001-work-authority\n"
				return strings.Replace(value, line, line+line, 1)
			},
			findingCode: "public.planning_grant_field",
		},
		{
			name: "unknown critical field",
			mutate: func(value string) string {
				line := "  coordinator: delivery-orchestrator\n"
				return strings.Replace(value, line, line+"  authorityOverride: true\n", 1)
			},
			findingCode: "public.planning_grant_schema",
		},
		{
			name: "YAML anchor indirection",
			mutate: func(value string) string {
				return strings.Replace(value, "  workingBranch: codex/w-001-work-authority\n", "  workingBranch: &branch codex/w-001-work-authority\n", 1)
			},
			findingCode: "public.planning_grant_schema",
		},
		{
			name: "tampered base commit",
			mutate: func(value string) string {
				return strings.Replace(value, "ee385ce236ae1f99da692d223d7666b80dd9108f", "ee385ce236ae1f99da692d223d7666b80dd91080", 1)
			},
			findingCode: "public.planning_grant_value",
		},
		{
			name: "reordered reviewers",
			mutate: func(value string) string {
				old := "    - qa\n    - security-reviewer\n    - delivery-orchestrator\n"
				updated := "    - security-reviewer\n    - qa\n    - delivery-orchestrator\n"
				return strings.Replace(value, old, updated, 1)
			},
			findingCode: "public.planning_grant_sequence",
		},
		{
			name: "extra authorized path",
			mutate: func(value string) string {
				line := "    - internal/doctrine/public_test.go\n"
				return strings.Replace(value, line, line+"    - internal/runtime/unsafe.go\n", 1)
			},
			findingCode: "public.planning_grant_sequence",
		},
		{
			name: "unknown effect",
			mutate: func(value string) string {
				line := "    - open-or-update-pull-request\n"
				return strings.Replace(value, line, line+"    - deploy-production\n", 1)
			},
			findingCode: "public.planning_grant_sequence",
		},
		{
			name: "wrong signature namespace metadata",
			mutate: func(value string) string {
				return strings.Replace(value, "signatureNamespace: mars3-planning-grant", "signatureNamespace: mars3-other-grant", 1)
			},
			findingCode: "public.planning_grant_value",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			modified := []byte(testCase.mutate(string(grant)))
			if string(modified) == string(grant) {
				t.Fatal("test mutation did not change the planning grant")
			}
			root := writePlanningGrantFixture(t, modified, signature, publicKey)
			var findings []Finding
			checkWave1PlanningGrant(root, &findings)
			if !findingCodePresent(findings, testCase.findingCode) {
				t.Fatalf("unsafe planning-grant change was accepted; wanted %s, got %v", testCase.findingCode, findings)
			}
			if !findingCodePresent(findings, "public.planning_grant_signature") {
				t.Fatalf("tampered signed bytes did not fail signature verification: %v", findings)
			}
		})
	}
}

func TestWave1PlanningGrantRequiresDetachedSignatureAndPinnedGenesisKey(t *testing.T) {
	grant, signature, publicKey := loadPlanningGrantFixture(t)

	t.Run("missing signature", func(t *testing.T) {
		root := writePlanningGrantFixture(t, grant, nil, publicKey)
		var findings []Finding
		checkWave1PlanningGrant(root, &findings)
		if !findingCodePresent(findings, "public.planning_grant_signature_missing") {
			t.Fatalf("missing detached signature was accepted: %v", findings)
		}
	})

	t.Run("tampered signature", func(t *testing.T) {
		tampered := append([]byte(nil), signature...)
		for index, value := range tampered {
			if value == 'A' {
				tampered[index] = 'B'
				break
			}
		}
		root := writePlanningGrantFixture(t, grant, tampered, publicKey)
		var findings []Finding
		checkWave1PlanningGrant(root, &findings)
		if !findingCodePresent(findings, "public.planning_grant_signature") {
			t.Fatalf("tampered detached signature was accepted: %v", findings)
		}
	})

	t.Run("unanchored public key", func(t *testing.T) {
		tampered := []byte(strings.Replace(string(publicKey), "ssh-ed25519", "ssh-rsa", 1))
		root := writePlanningGrantFixture(t, grant, signature, tampered)
		var findings []Finding
		checkWave1PlanningGrant(root, &findings)
		for _, code := range []string{"public.planning_grant_key_anchor", "public.planning_grant_key_fingerprint"} {
			if !findingCodePresent(findings, code) {
				t.Fatalf("unanchored key did not produce %s: %v", code, findings)
			}
		}
	})
}

func TestWave1RecoveryDispositionBindsSignatureSnapshotAndLegacyExclusion(t *testing.T) {
	grant, signature, publicKey := loadPlanningGrantFixture(t)

	t.Run("canonical disposition passes", func(t *testing.T) {
		root := writePlanningGrantFixture(t, grant, signature, publicKey)
		var findings []Finding
		checkWave1RecoveryDisposition(root, &findings)
		if len(findings) != 0 {
			t.Fatalf("valid recovery disposition rejected: %v", findings)
		}
	})

	t.Run("tampered signed field fails", func(t *testing.T) {
		root := writePlanningGrantFixture(t, grant, signature, publicKey)
		path := filepath.Join(root, filepath.FromSlash(wave1DispositionPath))
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		data = bytes.Replace(data, []byte("retroactiveAuthorization: false"), []byte("retroactiveAuthorization: true"), 1)
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
		var findings []Finding
		checkWave1RecoveryDisposition(root, &findings)
		for _, code := range []string{"public.recovery_disposition_value", "public.recovery_disposition_signature"} {
			if !findingCodePresent(findings, code) {
				t.Fatalf("tampered recovery disposition did not produce %s: %v", code, findings)
			}
		}
	})

	t.Run("snapshot change fails digest", func(t *testing.T) {
		root := writePlanningGrantFixture(t, grant, signature, publicKey)
		path := filepath.Join(root, filepath.FromSlash(wave1DispositionSnapshot))
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		data = bytes.Replace(data, []byte(`"claimState": "absent"`), []byte(`"claimState": "present"`), 1)
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
		var findings []Finding
		checkWave1RecoveryDisposition(root, &findings)
		if !findingCodePresent(findings, "public.recovery_snapshot_digest") {
			t.Fatalf("tampered recovery snapshot was accepted: %v", findings)
		}
	})

	t.Run("legacy scanner-triggering artifact is rejected", func(t *testing.T) {
		root := writePlanningGrantFixture(t, grant, signature, publicKey)
		writePlanningGrantTestFile(t, root, ".harness/grants/WAVE-1-authority-recovery.yaml", []byte("legacy\n"))
		var findings []Finding
		checkWave1RecoveryDisposition(root, &findings)
		if !findingCodePresent(findings, "public.recovery_legacy_artifact") {
			t.Fatalf("legacy recovery artifact was accepted into Git scope: %v", findings)
		}
	})
}

func TestWave1CIRecoveryAddendumBindsExactProspectiveCorrection(t *testing.T) {
	grant, signature, publicKey := loadPlanningGrantFixture(t)

	t.Run("canonical addendum passes", func(t *testing.T) {
		root := writePlanningGrantFixture(t, grant, signature, publicKey)
		var findings []Finding
		checkWave1CIRecoveryAddendum(root, &findings)
		if len(findings) != 0 {
			t.Fatalf("valid signed CI recovery addendum rejected: %v", findings)
		}
	})

	t.Run("tampered authority fails", func(t *testing.T) {
		root := writePlanningGrantFixture(t, grant, signature, publicKey)
		path := filepath.Join(root, filepath.FromSlash(wave1AddendumPath))
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		data = bytes.Replace(data, []byte("canonicalWorkMutationAllowed: false"), []byte("canonicalWorkMutationAllowed: true"), 1)
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
		var findings []Finding
		checkWave1CIRecoveryAddendum(root, &findings)
		for _, code := range []string{"public.ci_recovery_addendum_value", "public.ci_recovery_addendum_signature"} {
			if !findingCodePresent(findings, code) {
				t.Fatalf("tampered CI recovery addendum did not produce %s: %v", code, findings)
			}
		}
	})

	t.Run("extra path fails", func(t *testing.T) {
		root := writePlanningGrantFixture(t, grant, signature, publicKey)
		path := filepath.Join(root, filepath.FromSlash(wave1AddendumPath))
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		data = bytes.Replace(data, []byte("    - internal/doctrine/grant_test.go\n"), []byte("    - internal/doctrine/grant_test.go\n    - .github/workflows/foundation-quality.yml\n"), 1)
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
		var findings []Finding
		checkWave1CIRecoveryAddendum(root, &findings)
		if !findingCodePresent(findings, "public.ci_recovery_addendum_sequence") || !findingCodePresent(findings, "public.ci_recovery_addendum_signature") {
			t.Fatalf("widened CI recovery addendum was accepted: %v", findings)
		}
	})
}

func TestWave1V3CIRecoveryAddendumBindsExactProspectiveCorrection(t *testing.T) {
	grant, signature, publicKey := loadPlanningGrantFixture(t)

	t.Run("canonical v3 addendum passes", func(t *testing.T) {
		root := writePlanningGrantFixture(t, grant, signature, publicKey)
		var findings []Finding
		checkWave1V3CIRecoveryAddendum(root, &findings)
		if len(findings) != 0 {
			t.Fatalf("valid signed v3 CI recovery addendum rejected: %v", findings)
		}
	})

	t.Run("tampered v3 authority fails", func(t *testing.T) {
		root := writePlanningGrantFixture(t, grant, signature, publicKey)
		path := filepath.Join(root, filepath.FromSlash(wave1V3AddendumPath))
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		data = bytes.Replace(data, []byte("canonicalWorkMutationAllowed: false"), []byte("canonicalWorkMutationAllowed: true"), 1)
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
		var findings []Finding
		checkWave1V3CIRecoveryAddendum(root, &findings)
		for _, code := range []string{"public.ci_recovery_v3_addendum_value", "public.ci_recovery_v3_addendum_signature"} {
			if !findingCodePresent(findings, code) {
				t.Fatalf("tampered v3 CI recovery addendum did not produce %s: %v", code, findings)
			}
		}
	})

	t.Run("v3 path widening fails", func(t *testing.T) {
		root := writePlanningGrantFixture(t, grant, signature, publicKey)
		path := filepath.Join(root, filepath.FromSlash(wave1V3AddendumPath))
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		data = bytes.Replace(data, []byte("    - internal/doctrine/grant_test.go\n"), []byte("    - internal/doctrine/grant_test.go\n    - .github/workflows/foundation-quality.yml\n"), 1)
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
		var findings []Finding
		checkWave1V3CIRecoveryAddendum(root, &findings)
		if !findingCodePresent(findings, "public.ci_recovery_v3_addendum_sequence") || !findingCodePresent(findings, "public.ci_recovery_v3_addendum_signature") {
			t.Fatalf("widened v3 CI recovery addendum was accepted: %v", findings)
		}
	})
}

func TestWave1PlanningGrantIsRequired(t *testing.T) {
	var findings []Finding
	checkWave1PlanningGrant(planningGrantCanonicalTempDir(t), &findings)
	if !findingCodePresent(findings, "public.planning_grant_missing") {
		t.Fatalf("missing planning grant was accepted: %v", findings)
	}
}

func TestWave1PlanningGrantConfinesLiveContractPublicationDiff(t *testing.T) {
	t.Run("authorized untracked file on signed branch passes", func(t *testing.T) {
		root := writePlanningGrantGitFixture(t)
		writePlanningGrantTestFile(t, root, "docs/evidence/WAVE-1-contract-publication.md", []byte("# Authorized CI recovery evidence fixture\n"))

		var findings []Finding
		checkWave1PlanningGrant(root, &findings)
		if len(findings) != 0 {
			t.Fatalf("authorized untracked contract path was rejected in detached checkout: %v", findings)
		}
	})

	t.Run("legacy path is not reusable after CI recovery base", func(t *testing.T) {
		root := writePlanningGrantGitFixture(t)
		writePlanningGrantTestFile(t, root, "docs/features/F-002-work-authority.md", []byte("# Legacy authorization must not leak forward\n"))

		var findings []Finding
		checkWave1PlanningGrant(root, &findings)
		if !findingCodePresent(findings, "public.planning_grant_diff_scope") {
			t.Fatalf("legacy broad path authorization was reusable after the CI recovery base: %v", findings)
		}
	})

	t.Run("unauthorized otherwise governed path fails", func(t *testing.T) {
		root := writePlanningGrantGitFixture(t)
		const unauthorized = "internal/doctrine/paths.go"
		var scopeFindings []Finding
		checkGovernedPublicScope(unauthorized, &scopeFindings)
		if len(scopeFindings) != 0 {
			t.Fatalf("regression path must otherwise be in the governed public root: %v", scopeFindings)
		}
		writePlanningGrantTestFile(t, root, unauthorized, []byte("package doctrine\n"))

		var findings []Finding
		checkWave1PlanningGrant(root, &findings)
		if !findingCodePresent(findings, "public.planning_grant_diff_scope") {
			t.Fatalf("unauthorized live-diff path was accepted: %v", findings)
		}
	})
}

func TestWave1PlanningGrantBindsCheckoutAndCommitHistory(t *testing.T) {
	t.Run("published v1 tag cannot move or disappear", func(t *testing.T) {
		root := writePlanningGrantGitFixture(t)
		runPlanningGrantTestGit(t, root, "tag", "-d", wave1PriorPublicationTag)

		var findings []Finding
		checkWave1PlanningGrant(root, &findings)
		if !findingCodePresent(findings, "public.prior_publication_tag") {
			t.Fatal("missing immutable v1 publication tag was accepted")
		}
	})

	t.Run("published v2 tag cannot move or disappear", func(t *testing.T) {
		root := writePlanningGrantGitFixture(t)
		runPlanningGrantTestGit(t, root, "tag", "-d", wave1V2PublicationTag)

		var findings []Finding
		checkWave1PlanningGrant(root, &findings)
		if !findingCodePresent(findings, "public.prior_v2_publication_tag") {
			t.Fatal("missing immutable v2 publication tag was accepted")
		}
	})

	t.Run("nullable pull request event merge SHA uses exact checkout topology", func(t *testing.T) {
		root := writePlanningGrantGitFixture(t)
		merge := checkoutPlanningGrantSyntheticMerge(t, root, true)
		setPlanningGrantPullRequestFactsWithEventMerge(t, root, merge, nil)

		checkout, ok := planningGrantGitHubCheckout(root, merge, "")
		if !ok {
			t.Fatal("canonical GitHub pull-request checkout with a null optional event merge SHA was rejected")
		}
		if checkout.kind != planningGrantPullRequestMerge || checkout.firstParent != wave1PlanningGrantBase || checkout.secondParent != wave1PlanningGrantFirstCommitFixture {
			t.Fatalf("nullable event merge SHA did not retain exact base/head topology: %+v", checkout)
		}
	})

	t.Run("absent pull request event merge SHA uses exact checkout topology", func(t *testing.T) {
		root := writePlanningGrantGitFixture(t)
		merge := checkoutPlanningGrantSyntheticMerge(t, root, true)
		setPlanningGrantPullRequestFactsWithoutEventMerge(t, root, merge)

		if _, ok := planningGrantGitHubCheckout(root, merge, ""); !ok {
			t.Fatal("canonical GitHub pull-request checkout with an absent optional event merge SHA was rejected")
		}
	})

	t.Run("stale well formed pull request event merge SHA is advisory", func(t *testing.T) {
		root := writePlanningGrantGitFixture(t)
		merge := checkoutPlanningGrantSyntheticMerge(t, root, true)
		setPlanningGrantPullRequestFactsWithEventMerge(t, root, merge, wave1V3ObservedStaleMerge)

		if _, ok := planningGrantGitHubCheckout(root, merge, ""); !ok {
			t.Fatal("well-formed stale advisory event merge SHA overrode exact checkout topology")
		}
	})

	t.Run("stale advisory identity cannot mask event head mismatch", func(t *testing.T) {
		root := writePlanningGrantGitFixture(t)
		merge := checkoutPlanningGrantSyntheticMerge(t, root, true)
		pullRequest := canonicalPlanningGrantPullRequestPayload()
		pullRequest["merge_commit_sha"] = wave1V3ObservedStaleMerge
		pullRequest["head"] = map[string]any{"ref": wave1PlanningGrantBranch, "sha": "1111111111111111111111111111111111111111"}
		setPlanningGrantPullRequestFactsWithPayload(t, root, merge, pullRequest)

		checkout, ok := planningGrantGitHubCheckout(root, merge, "")
		if !ok {
			t.Fatal("test setup did not reach topology validation")
		}
		commits, err := planningGrantCommitRange(root, merge)
		if err != nil {
			t.Fatal(err)
		}
		var findings []Finding
		if checkPlanningGrantCommitTopology(root, checkout, merge, commits, &findings) || !findingCodePresent(findings, "public.planning_grant_commit_topology") {
			t.Fatalf("stale advisory field masked an event-head/topology mismatch: %v", findings)
		}
	})

	t.Run("malformed pull request event merge SHA fails", func(t *testing.T) {
		root := writePlanningGrantGitFixture(t)
		merge := checkoutPlanningGrantSyntheticMerge(t, root, true)
		setPlanningGrantPullRequestFactsWithEventMerge(t, root, merge, "not-a-commit")

		if _, ok := planningGrantGitHubCheckout(root, merge, ""); ok {
			t.Fatal("malformed advisory event merge identity was accepted")
		}
	})

	t.Run("wrong symbolic branch fails", func(t *testing.T) {
		root := writePlanningGrantGitFixture(t)
		runPlanningGrantTestGit(t, root, "branch", "-m", "codex/unauthorized-contract-copy")

		var findings []Finding
		checkWave1PlanningGrant(root, &findings)
		if !findingCodePresent(findings, "public.planning_grant_branch") {
			t.Fatalf("wrong symbolic branch was accepted: %v", findings)
		}
	})

	t.Run("detached checkout without canonical runner facts fails", func(t *testing.T) {
		root := writePlanningGrantGitFixture(t)
		runPlanningGrantTestGit(t, root, "checkout", "--quiet", "--detach")
		t.Setenv("GITHUB_ACTIONS", "")

		var findings []Finding
		checkWave1PlanningGrant(root, &findings)
		if !findingCodePresent(findings, "public.planning_grant_branch") {
			t.Fatalf("unattested detached checkout was accepted: %v", findings)
		}
	})

	t.Run("pull request merge without publication tag fails closed", func(t *testing.T) {
		root := writePlanningGrantGitFixture(t)
		merge := checkoutPlanningGrantSyntheticMerge(t, root, true)
		setPlanningGrantPullRequestFacts(t, root, merge)

		var findings []Finding
		checkWave1PlanningGrant(root, &findings)
		if !findingCodePresent(findings, "public.publication_tag_missing") {
			t.Fatalf("untagged pull-request publication was accepted: %v", findings)
		}
	})

	t.Run("protected main merge without publication tag fails closed", func(t *testing.T) {
		root := writePlanningGrantGitFixture(t)
		merge := checkoutPlanningGrantSyntheticMerge(t, root, false)
		setPlanningGrantMainPushFacts(t, root, merge)

		var findings []Finding
		checkWave1PlanningGrant(root, &findings)
		if !findingCodePresent(findings, "public.publication_tag_missing") {
			t.Fatalf("untagged protected-main publication was accepted: %v", findings)
		}
	})

	t.Run("protected main squash-style push fails closed", func(t *testing.T) {
		root := writePlanningGrantGitFixture(t)
		const authorized = "docs/evidence/WAVE-1-contract-publication.md"
		writePlanningGrantTestFile(t, root, authorized, []byte("# Rewritten squash fixture\n"))
		commitPlanningGrantTestPaths(t, root, "squash-style rewritten tip", authorized)
		head := planningGrantTestGitOutput(t, root, "rev-parse", "HEAD")
		runPlanningGrantTestGit(t, root, "branch", "-m", "main")
		setPlanningGrantMainPushFacts(t, root, head)

		var findings []Finding
		checkWave1PlanningGrant(root, &findings)
		if !findingCodePresent(findings, "public.publication_tag_missing") {
			t.Fatalf("untagged squash-style main history was accepted: %v", findings)
		}
	})

	t.Run("unsigned tip fails", func(t *testing.T) {
		root := writePlanningGrantGitFixture(t)
		const authorized = "docs/evidence/WAVE-1-contract-publication.md"
		writePlanningGrantTestFile(t, root, authorized, []byte("# Unsigned authorized change\n"))
		commitPlanningGrantTestPaths(t, root, "unsigned authorized tip", authorized)

		var findings []Finding
		checkWave1PlanningGrant(root, &findings)
		if !findingCodePresent(findings, "public.planning_grant_commit_signature") {
			t.Fatalf("unsigned contract-publication tip was accepted: %v", findings)
		}
	})

	t.Run("unauthorized add then delete remains visible", func(t *testing.T) {
		root := writePlanningGrantGitFixture(t)
		const unauthorized = "internal/doctrine/escape.go"
		var scopeFindings []Finding
		checkGovernedPublicScope(unauthorized, &scopeFindings)
		if len(scopeFindings) != 0 {
			t.Fatalf("regression path must otherwise be governed: %v", scopeFindings)
		}
		writePlanningGrantTestFile(t, root, unauthorized, []byte("package doctrine\n"))
		commitPlanningGrantTestPaths(t, root, "add transient unauthorized path", unauthorized)
		if err := os.Remove(filepath.Join(root, filepath.FromSlash(unauthorized))); err != nil {
			t.Fatal(err)
		}
		commitPlanningGrantTestPaths(t, root, "delete transient unauthorized path", unauthorized)
		netPaths := planningGrantTestGitOutput(t, root, "diff", "--name-only", wave1PlanningGrantBase, "--")
		if strings.Contains(netPaths, unauthorized) {
			t.Fatal("test setup did not erase the unauthorized path from the net base diff")
		}

		var findings []Finding
		checkWave1PlanningGrant(root, &findings)
		if !findingCodePresent(findings, "public.planning_grant_diff_scope") {
			t.Fatalf("transient unauthorized commit path was accepted: %v", findings)
		}
	})

	t.Run("delivery projection cannot disable grant enforcement", func(t *testing.T) {
		root := writePlanningGrantGitFixture(t)
		planPath := filepath.Join(root, filepath.FromSlash(canonicalActivePlan))
		plan, err := os.ReadFile(planPath)
		if err != nil {
			t.Fatal(err)
		}
		updated := strings.Replace(string(plan), "**Phase:** contract-publication", "**Phase:** delivery", 1)
		if updated == string(plan) {
			t.Fatal("test plan did not contain contract-publication phase")
		}
		if err := os.WriteFile(planPath, []byte(updated), 0o644); err != nil {
			t.Fatal(err)
		}

		var findings []Finding
		checkWave1PlanningGrant(root, &findings)
		if !findingCodePresent(findings, "public.planning_grant_transition_authority") {
			t.Fatalf("mutable plan phase disabled grant enforcement: %v", findings)
		}
	})
}

func TestWave1PRFallbackMainSquashUsesSignedTreeIdentity(t *testing.T) {
	root, squash := writeWave1TransitionSquashFixture(t, wave1PublishedMain)
	var findings []Finding
	if _, ok := checkWave1MainSquashTopology(root, squash, &findings); !ok {
		t.Fatalf("canonical one-parent squash topology was rejected: %v", findings)
	}
	target, ok := checkWave1TransitionTag(root, wave1TransitionTag, wave1TransitionTagMessage, wave1TransitionReviewedHead, squash, true, &findings)
	if !ok || target != wave1TransitionReviewedHead || len(findings) != 0 {
		t.Fatalf("signed feature tag tree did not admit the distinct squash commit: target=%s findings=%v", target, findings)
	}
}

func TestWave1PRFallbackProtectedMainAdmissionUsesReviewedTree(t *testing.T) {
	root, squash := writeWave1PRFallbackMainFixture(t, true)
	setWave1PRFallbackMainPushFacts(t, root, squash)
	var findings []Finding
	checkWave1PRFallbackGitDiff(root, &findings)
	if len(findings) != 0 {
		t.Fatalf("protected-main squash with the signed reviewed tree was rejected: %v", findings)
	}
}

func TestWave1PRFallbackMainSquashRejectsWrongTreeParentAndTarget(t *testing.T) {
	t.Run("wrong tree", func(t *testing.T) {
		root, _ := writeWave1TransitionSquashFixture(t, wave1PublishedMain)
		var findings []Finding
		if _, ok := checkWave1TransitionTag(root, wave1TransitionTag, wave1TransitionTagMessage, wave1TransitionReviewedHead, wave1PublishedMain, true, &findings); ok || !findingCodePresent(findings, "public.pr_fallback_tag") {
			t.Fatalf("wrong squash tree was accepted: %v", findings)
		}
	})

	t.Run("wrong parent", func(t *testing.T) {
		root, squash := writeWave1TransitionSquashFixture(t, wave1TransitionReviewedHead)
		var findings []Finding
		if _, ok := checkWave1MainSquashTopology(root, squash, &findings); ok || !findingCodePresent(findings, "public.pr_fallback_topology") {
			t.Fatalf("squash commit with the wrong parent was accepted: %v", findings)
		}
	})

	t.Run("wrong feature target", func(t *testing.T) {
		root, squash := writeWave1TransitionSquashFixture(t, wave1PublishedMain)
		var findings []Finding
		if _, ok := checkWave1TransitionTag(root, wave1TransitionTag, wave1TransitionTagMessage, wave1PublishedMain, squash, true, &findings); ok || !findingCodePresent(findings, "public.pr_fallback_tag") {
			t.Fatalf("transition tag with the wrong reviewed target was accepted: %v", findings)
		}
	})
}

func TestVerifyPlanningGrantTagRequiresExactSignedTreeAttestation(t *testing.T) {
	root := planningGrantCanonicalTempDir(t)
	keyPath := filepath.Join(root, "synthetic-tag-key")
	command := exec.Command("ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-f", keyPath)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("generate synthetic tag key: %v: %s", err, output)
	}
	publicKey, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		t.Fatal(err)
	}
	const target = "1111111111111111111111111111111111111111"
	signed := []byte("object " + target + "\ntype commit\ntag " + wave1PublicationTag + "\ntagger MARS-3 Release Manager <release-manager@example.com> 0 +0000\n\n" + wave1PublicationTagMessage + "\n")
	messagePath := filepath.Join(root, "tag-object")
	if err := os.WriteFile(messagePath, signed, 0o644); err != nil {
		t.Fatal(err)
	}
	command = exec.Command("ssh-keygen", "-Y", "sign", "-f", keyPath, "-n", planningGrantCommitNS, messagePath)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("sign synthetic tag: %v: %s", err, output)
	}
	signature, err := os.ReadFile(messagePath + ".sig")
	if err != nil {
		t.Fatal(err)
	}
	object := append(append([]byte(nil), signed...), signature...)
	actual, err := verifyPlanningGrantTag(object, publicKey)
	if err != nil || actual != target {
		t.Fatalf("valid signed tag rejected: target=%s err=%v", actual, err)
	}

	tampered := bytes.Replace(object, []byte(wave1PublicationTagMessage), []byte("different tree attestation"), 1)
	if _, err := verifyPlanningGrantTag(tampered, publicKey); err == nil {
		t.Fatal("tampered signed tag was accepted")
	}
}

func writeWave1TransitionSquashFixture(t *testing.T, parent string) (string, string) {
	t.Helper()
	source, err := filepath.Abs(filepath.Clean(filepath.Join("..", "..")))
	if err != nil {
		t.Fatal(err)
	}
	root := planningGrantCanonicalTempDir(t)
	runPlanningGrantTestGit(t, root, "init", "--quiet")
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, wave1PublishedMain)
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, "refs/tags/"+wave1TransitionTag+":refs/tags/"+wave1TransitionTag)
	publicKey, err := os.ReadFile(filepath.Join(source, filepath.FromSlash(wave1PlanningGrantKey)))
	if err != nil {
		t.Fatal(err)
	}
	writePlanningGrantTestFile(t, root, wave1PlanningGrantKey, publicKey)
	tree := planningGrantTestGitOutput(t, root, "rev-parse", "--verify", wave1TransitionReviewedHead+"^{tree}")
	squash := planningGrantTestGitOutput(t, root,
		"-c", "user.name=Synthetic Merge Bot",
		"-c", "user.email=merge-bot@example.com",
		"-c", "commit.gpgsign=false",
		"commit-tree", tree,
		"-p", parent,
		"-m", "synthetic W-001 squash",
	)
	return root, squash
}

func writeWave1PRFallbackMainFixture(t *testing.T, reviewedTree bool) (string, string) {
	t.Helper()
	source, err := filepath.Abs(filepath.Clean(filepath.Join("..", "..")))
	if err != nil {
		t.Fatal(err)
	}
	sourceHead := planningGrantTestGitOutput(t, source, "rev-parse", "HEAD")
	root := planningGrantCanonicalTempDir(t)
	runPlanningGrantTestGit(t, root, "init", "--quiet")
	for _, revision := range []string{wave1PublishedMain, sourceHead} {
		runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, revision)
	}
	for _, tag := range []string{wave1PriorPublicationTag, wave1V2PublicationTag, wave1PublicationTag, wave1TransitionTag, wave1SuccessorTransitionTag, wave1FinalTransitionTag} {
		if _, err := planningGrantTestGitTryOutput(source, "rev-parse", "--verify", "refs/tags/"+tag+"^{tag}"); err == nil {
			runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, "refs/tags/"+tag+":refs/tags/"+tag)
		}
	}
	treeSource := wave1PublishedMain
	if reviewedTree {
		treeSource = "refs/tags/" + wave1FinalTransitionTag + "^{}"
	}
	tree := planningGrantTestGitOutput(t, root, "rev-parse", "--verify", treeSource+"^{tree}")
	squash := planningGrantTestGitOutput(t, root,
		"-c", "user.name=Synthetic Merge Bot",
		"-c", "user.email=merge-bot@example.com",
		"-c", "commit.gpgsign=false",
		"commit-tree", tree,
		"-p", wave1PublishedMain,
		"-m", "synthetic protected-main W-001 squash",
	)
	runPlanningGrantTestGit(t, root, "checkout", "--quiet", "--force", "-B", "main", squash)
	return root, squash
}

func setWave1PRFallbackMainPushFacts(t *testing.T, root, head string) {
	t.Helper()
	event := map[string]any{
		"before":      wave1PublishedMain,
		"after":       head,
		"ref":         "refs/heads/main",
		"head_commit": map[string]any{"id": head},
		"repository":  map[string]any{"full_name": planningGrantRepository},
	}
	eventPath := writePlanningGrantGitHubEvent(t, event)
	setPlanningGrantCommonGitHubFacts(t, root, head, eventPath)
	t.Setenv("GITHUB_EVENT_NAME", "push")
	t.Setenv("GITHUB_REF", "refs/heads/main")
	t.Setenv("GITHUB_HEAD_REF", "")
	t.Setenv("GITHUB_BASE_REF", "")
	t.Setenv("GITHUB_REF_PROTECTED", "true")
	t.Setenv("GITHUB_WORKFLOW_REF", planningGrantRepository+"/"+planningGrantWorkflowPath+"@refs/heads/main")
}

func TestNormalizedPlanningGrantGitPathsFailsClosed(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		output []byte
	}{
		{name: "not NUL terminated", output: []byte("docs/features/F-002-work-authority.md")},
		{name: "empty path", output: []byte{0}},
		{name: "non UTF-8", output: []byte{0xff, 0}},
		{name: "absolute", output: []byte("/outside\x00")},
		{name: "traversal", output: []byte("docs/../outside\x00")},
		{name: "backslash", output: []byte("docs\\outside\x00")},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := normalizedPlanningGrantGitPaths(testCase.output); err == nil {
				t.Fatal("unsafe Git path output was accepted")
			}
		})
	}
}

func TestValidAdvisoryPullRequestMergeSHA(t *testing.T) {
	for _, value := range []string{"", "1111111111111111111111111111111111111111", wave1V3ObservedStaleMerge} {
		if !validAdvisoryPullRequestMergeSHA(value) {
			t.Fatalf("valid advisory merge identity %q was rejected", value)
		}
	}
	for _, value := range []string{"not-a-commit", "111111111111111111111111111111111111111", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"} {
		if validAdvisoryPullRequestMergeSHA(value) {
			t.Fatalf("malformed advisory merge identity %q was accepted", value)
		}
	}
}

func loadPlanningGrantFixture(t *testing.T) ([]byte, []byte, []byte) {
	t.Helper()
	repo := filepath.Clean(filepath.Join("..", ".."))
	read := func(path string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	return read(wave1PlanningGrantPath), read(wave1PlanningGrantSignature), read(wave1PlanningGrantKey)
}

func writePlanningGrantFixture(t *testing.T, grant, signature, publicKey []byte) string {
	t.Helper()
	root := planningGrantCanonicalTempDir(t)
	writePlanningGrantTestFile(t, root, wave1PlanningGrantPath, grant)
	writePlanningGrantTestFile(t, root, wave1PlanningGrantSignature, signature)
	writePlanningGrantTestFile(t, root, wave1PlanningGrantKey, publicKey)
	writePlanningGrantDispositionFiles(t, root)
	return root
}

func writePlanningGrantGitFixture(t *testing.T) string {
	t.Helper()
	source, err := filepath.Abs(filepath.Clean(filepath.Join("..", "..")))
	if err != nil {
		t.Fatal(err)
	}
	root := planningGrantCanonicalTempDir(t)
	runPlanningGrantTestGit(t, root, "init", "--quiet")
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, wave1V3AddendumBase)
	runPlanningGrantTestGit(t, root, "checkout", "--quiet", "--detach", "FETCH_HEAD")
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, "refs/tags/"+wave1PriorPublicationTag+":refs/tags/"+wave1PriorPublicationTag)
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, "refs/tags/"+wave1V2PublicationTag+":refs/tags/"+wave1V2PublicationTag)
	runPlanningGrantTestGit(t, root, "checkout", "--quiet", "-b", wave1PlanningGrantBranch)

	writePlanningGrantCurrentFiles(t, root)
	return root
}

func TestPlanningGrantGitFixtureDoesNotPersistMaintenanceConfiguration(t *testing.T) {
	root := writePlanningGrantGitFixture(t)
	for _, key := range []string{"maintenance.auto", "gc.auto", "gc.autoDetach", "maintenance.autoDetach"} {
		assertPlanningGrantTestGitConfigAbsent(t, root, "--local", key)
		assertPlanningGrantTestGitConfigAbsent(t, root, "--global", key)
	}
}

func writePlanningGrantCurrentFiles(t *testing.T, root string) {
	t.Helper()
	grant, signature, publicKey := loadPlanningGrantFixture(t)
	writePlanningGrantTestFile(t, root, wave1PlanningGrantPath, grant)
	writePlanningGrantTestFile(t, root, wave1PlanningGrantSignature, signature)
	writePlanningGrantTestFile(t, root, wave1PlanningGrantKey, publicKey)
	writePlanningGrantDispositionFiles(t, root)
	source, err := filepath.Abs(filepath.Clean(filepath.Join("..", "..")))
	if err != nil {
		t.Fatal(err)
	}
	plan := planningGrantTestGitRawOutput(t, source, "show", wave1V3AddendumBase+":"+canonicalActivePlan)
	writePlanningGrantTestFile(t, root, canonicalActivePlan, plan)
}

func writePlanningGrantDispositionFiles(t *testing.T, root string) {
	t.Helper()
	source, err := filepath.Abs(filepath.Clean(filepath.Join("..", "..")))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{wave1DispositionPath, wave1DispositionSignature, wave1DispositionSnapshot, wave1AddendumPath, wave1AddendumSignature, wave1V3AddendumPath, wave1V3AddendumSignature} {
		data, err := os.ReadFile(filepath.Join(source, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		writePlanningGrantTestFile(t, root, path, data)
	}
}

func checkoutPlanningGrantSyntheticMerge(t *testing.T, root string, detached bool) string {
	t.Helper()
	source, err := filepath.Abs(filepath.Clean(filepath.Join("..", "..")))
	if err != nil {
		t.Fatal(err)
	}
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, wave1PlanningGrantFirstCommitFixture)
	tree := planningGrantTestGitOutput(t, root, "rev-parse", "--verify", wave1PlanningGrantFirstCommitFixture+"^{tree}")
	merge := planningGrantTestGitOutput(t, root,
		"-c", "user.name=Synthetic Merge Bot",
		"-c", "user.email=merge-bot@example.com",
		"-c", "commit.gpgsign=false",
		"commit-tree", tree,
		"-p", wave1PlanningGrantBase,
		"-p", wave1PlanningGrantFirstCommitFixture,
		"-m", "synthetic canonical merge",
	)
	if detached {
		runPlanningGrantTestGit(t, root, "checkout", "--quiet", "--force", "--detach", merge)
	} else {
		runPlanningGrantTestGit(t, root, "checkout", "--quiet", "--force", "-B", "main", merge)
	}
	writePlanningGrantCurrentFiles(t, root)
	return merge
}

func setPlanningGrantPullRequestFacts(t *testing.T, root, merge string) {
	setPlanningGrantPullRequestFactsWithEventMerge(t, root, merge, merge)
}

func setPlanningGrantPullRequestFactsWithEventMerge(t *testing.T, root, merge string, eventMerge any) {
	t.Helper()
	pullRequest := canonicalPlanningGrantPullRequestPayload()
	pullRequest["merge_commit_sha"] = eventMerge
	setPlanningGrantPullRequestFactsWithPayload(t, root, merge, pullRequest)
}

func setPlanningGrantPullRequestFactsWithoutEventMerge(t *testing.T, root, merge string) {
	t.Helper()
	setPlanningGrantPullRequestFactsWithPayload(t, root, merge, canonicalPlanningGrantPullRequestPayload())
}

func canonicalPlanningGrantPullRequestPayload() map[string]any {
	return map[string]any{
		"base": map[string]any{"ref": "main", "sha": wave1PlanningGrantBase},
		"head": map[string]any{"ref": wave1PlanningGrantBranch, "sha": wave1PlanningGrantFirstCommitFixture},
	}
}

func setPlanningGrantPullRequestFactsWithPayload(t *testing.T, root, merge string, pullRequest map[string]any) {
	t.Helper()
	const ref = "refs/pull/17/merge"
	event := map[string]any{
		"repository":   map[string]any{"full_name": planningGrantRepository},
		"pull_request": pullRequest,
	}
	eventPath := writePlanningGrantGitHubEvent(t, event)
	setPlanningGrantCommonGitHubFacts(t, root, merge, eventPath)
	t.Setenv("GITHUB_EVENT_NAME", "pull_request")
	t.Setenv("GITHUB_REF", ref)
	t.Setenv("GITHUB_HEAD_REF", wave1PlanningGrantBranch)
	t.Setenv("GITHUB_BASE_REF", "main")
	t.Setenv("GITHUB_REF_PROTECTED", "false")
	t.Setenv("GITHUB_WORKFLOW_REF", planningGrantRepository+"/"+planningGrantWorkflowPath+"@"+ref)
}

func setPlanningGrantMainPushFacts(t *testing.T, root, merge string) {
	t.Helper()
	event := map[string]any{
		"before":      wave1PlanningGrantBase,
		"after":       merge,
		"ref":         "refs/heads/main",
		"head_commit": map[string]any{"id": merge},
		"repository":  map[string]any{"full_name": planningGrantRepository},
	}
	eventPath := writePlanningGrantGitHubEvent(t, event)
	setPlanningGrantCommonGitHubFacts(t, root, merge, eventPath)
	t.Setenv("GITHUB_EVENT_NAME", "push")
	t.Setenv("GITHUB_REF", "refs/heads/main")
	t.Setenv("GITHUB_HEAD_REF", "")
	t.Setenv("GITHUB_BASE_REF", "")
	t.Setenv("GITHUB_REF_PROTECTED", "true")
	t.Setenv("GITHUB_WORKFLOW_REF", planningGrantRepository+"/"+planningGrantWorkflowPath+"@refs/heads/main")
}

func setPlanningGrantCommonGitHubFacts(t *testing.T, root, head, eventPath string) {
	t.Helper()
	t.Setenv("CI", "true")
	t.Setenv("GITHUB_ACTIONS", "true")
	t.Setenv("RUNNER_ENVIRONMENT", "github-hosted")
	t.Setenv("GITHUB_REPOSITORY", planningGrantRepository)
	t.Setenv("GITHUB_WORKFLOW", planningGrantWorkflow)
	t.Setenv("GITHUB_JOB", planningGrantWorkflowJob)
	t.Setenv("GITHUB_SHA", head)
	t.Setenv("GITHUB_WORKSPACE", root)
	t.Setenv("GITHUB_RUN_ID", "101")
	t.Setenv("GITHUB_RUN_ATTEMPT", "1")
	t.Setenv("GITHUB_EVENT_PATH", eventPath)
}

func writePlanningGrantGitHubEvent(t *testing.T, event map[string]any) string {
	t.Helper()
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	root := planningGrantCanonicalTempDir(t)
	const path = "event.json"
	writePlanningGrantTestFile(t, root, path, data)
	return filepath.Join(root, path)
}

func commitPlanningGrantTestPaths(t *testing.T, root, message string, paths ...string) {
	t.Helper()
	arguments := append([]string{"add", "--"}, paths...)
	runPlanningGrantTestGit(t, root, arguments...)
	runPlanningGrantTestGit(t, root,
		"-c", "user.name=Synthetic Engineer",
		"-c", "user.email=engineer@example.com",
		"-c", "commit.gpgsign=false",
		"commit", "--quiet", "--no-gpg-sign", "-m", message,
	)
}

func writePlanningGrantTestFile(t *testing.T, root, path string, data []byte) {
	t.Helper()
	if data == nil {
		return
	}
	fullPath := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func planningGrantCanonicalTempDir(t *testing.T) string {
	t.Helper()
	lexical := t.TempDir()
	physical, err := filepath.EvalSymlinks(lexical)
	if err != nil {
		t.Fatal(err)
	}
	physical, err = filepath.Abs(filepath.Clean(physical))
	if err != nil {
		t.Fatal(err)
	}
	return physical
}

func initializePlanningGrantTestRepository(t *testing.T, root, source, revision string) string {
	t.Helper()
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	runPlanningGrantTestGit(t, root, "init", "--quiet")
	if source == "" {
		return ""
	}
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, revision)
	return planningGrantTestGitOutput(t, root, "rev-parse", "FETCH_HEAD^{commit}")
}

func runPlanningGrantTestGit(t *testing.T, root string, arguments ...string) {
	t.Helper()
	command, commandErr := planningGrantTestGitCommand(root, arguments...)
	if commandErr != nil {
		t.Fatalf("unsafe git %v: %v", arguments, commandErr)
	}
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v: %s", arguments, err, output)
	}
}

func planningGrantTestGitOutput(t *testing.T, root string, arguments ...string) string {
	t.Helper()
	command, commandErr := planningGrantTestGitCommand(root, arguments...)
	if commandErr != nil {
		t.Fatalf("unsafe git %v: %v", arguments, commandErr)
	}
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v: %s", arguments, err, output)
	}
	return strings.TrimSpace(string(output))
}

func planningGrantTestGitRawOutput(t *testing.T, root string, arguments ...string) []byte {
	t.Helper()
	output, err := planningGrantTestGitTryOutput(root, arguments...)
	if err != nil {
		t.Fatalf("git %v failed: %v: %s", arguments, err, output)
	}
	return output
}

func planningGrantTestGitTryOutput(root string, arguments ...string) ([]byte, error) {
	command, commandErr := planningGrantTestGitCommand(root, arguments...)
	if commandErr != nil {
		return nil, commandErr
	}
	return command.CombinedOutput()
}

func planningGrantTestGitCommand(root string, arguments ...string) (*exec.Cmd, error) {
	if err := validatePlanningGrantTestGitArguments(arguments); err != nil {
		return nil, err
	}
	canonicalRoot, err := canonicalPlanningGrantTestRoot(root)
	if err != nil {
		return nil, err
	}
	if err := validatePlanningGrantTestGitFetch(arguments); err != nil {
		return nil, err
	}
	bounded := []string{
		"-c", "core.hooksPath=/dev/null",
		"-c", "maintenance.auto=false",
		"-c", "gc.auto=0",
		"-c", "gc.autoDetach=false",
		"-c", "maintenance.autoDetach=false",
		"-C", canonicalRoot,
	}
	command := exec.Command("/usr/bin/git", append(bounded, arguments...)...)
	command.Env = planningGrantTestGitEnvironment()
	return command, nil
}

func canonicalPlanningGrantTestRoot(root string) (string, error) {
	cleanRoot, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return "", fmt.Errorf("resolve disposable Git root: %w", err)
	}
	info, err := os.Lstat(cleanRoot)
	if err != nil {
		return "", fmt.Errorf("inspect disposable Git root: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return "", errors.New("disposable Git root must be one direct directory")
	}
	resolved, err := filepath.EvalSymlinks(cleanRoot)
	if err != nil {
		return "", errors.New("disposable Git root must have canonical physical ancestry")
	}
	if filepath.Clean(resolved) != cleanRoot {
		return "", errors.New("disposable Git root must equal its canonical physical path")
	}
	return cleanRoot, nil
}

func validatePlanningGrantTestGitFetch(arguments []string) error {
	for index, argument := range arguments {
		if argument != "fetch" {
			continue
		}
		if index+4 >= len(arguments) {
			return errors.New("bounded Git fetch source or revision is missing")
		}
		source := filepath.Clean(arguments[index+3])
		canonicalSource, err := canonicalPlanningGrantTestRoot(source)
		if err != nil || !filepath.IsAbs(source) || source != canonicalSource {
			return errors.New("bounded Git fetch source must be one canonical local directory")
		}
		_, _, err = validatePlanningGrantTestGitFetchRefspec(arguments[index+4])
		return err
	}
	return nil
}

func validatePlanningGrantTestGitFetchRefspec(value string) (string, string, error) {
	if value == "HEAD" || len(value) == 40 && sha1Pattern.MatchString(value) {
		return value, "", nil
	}
	parts := strings.Split(value, ":")
	if len(parts) != 2 || parts[0] != parts[1] || !validPlanningGrantTestTagRef(parts[0]) {
		return "", "", errors.New("bounded Git fetch revision must be HEAD, one SHA-1 object, or one identical tag refspec")
	}
	return parts[0], parts[1], nil
}

func validPlanningGrantTestTagRef(ref string) bool {
	if !strings.HasPrefix(ref, "refs/tags/") {
		return false
	}
	name := strings.TrimPrefix(ref, "refs/tags/")
	if name == "" || strings.HasPrefix(name, ".") || strings.HasSuffix(name, ".") || strings.HasSuffix(name, ".lock") ||
		strings.Contains(name, "..") || strings.Contains(name, "//") || strings.Contains(name, "@{") {
		return false
	}
	for _, character := range name {
		if character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' ||
			character >= '0' && character <= '9' || strings.ContainsRune("-_./", character) {
			continue
		}
		return false
	}
	return true
}

func planningGrantTestGitEnvironment() []string {
	return []string{
		"GIT_ALLOW_PROTOCOL=file",
		"GIT_ATTR_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=" + os.DevNull,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_NO_REPLACE_OBJECTS=1",
		"GIT_OPTIONAL_LOCKS=0",
		"GIT_PROTOCOL_FROM_USER=0",
		"GIT_TERMINAL_PROMPT=0",
		"HOME=/nonexistent",
		"LC_ALL=C",
		"PATH=/usr/bin:/bin",
		"PERL5LIB=",
		"PERL5OPT=",
		"TMPDIR=/tmp",
	}
}

func validatePlanningGrantTestGitArguments(arguments []string) error {
	if len(arguments) == 0 {
		return errors.New("Git subcommand is required")
	}
	allowedConfig := map[string]string{
		"user.name=Synthetic Merge Bot":          "",
		"user.name=Synthetic Release Manager":    "",
		"user.name=Synthetic Engineer":           "",
		"user.email=merge-bot@example.com":       "",
		"user.email=release-manager@example.com": "",
		"user.email=engineer@example.com":        "",
		"commit.gpgsign=false":                   "",
	}
	subcommand := ""
	subcommandIndex := -1
	for index := 0; index < len(arguments); index++ {
		argument := arguments[index]
		if argument == "-c" {
			if subcommand != "" {
				return errors.New("Git subcommand-local configuration override is not admitted")
			}
			if index+1 >= len(arguments) {
				return errors.New("Git -c requires one exact configuration assignment")
			}
			index++
			if _, ok := allowedConfig[arguments[index]]; !ok {
				return fmt.Errorf("Git configuration override %q is not admitted", arguments[index])
			}
			continue
		}
		if subcommand == "" && strings.HasPrefix(argument, "-") {
			return fmt.Errorf("Git global option %q is not admitted", argument)
		}
		if subcommand == "" {
			subcommand, subcommandIndex = argument, index
		}
	}
	if subcommand == "" {
		return errors.New("Git subcommand is required")
	}
	return validatePlanningGrantTestGitSubcommand(subcommand, arguments[subcommandIndex+1:])
}

func validatePlanningGrantTestGitSubcommand(subcommand string, arguments []string) error {
	nonOption := func(value string) bool { return value != "" && !strings.HasPrefix(value, "-") }
	exact := func(want ...string) bool {
		return len(arguments) == len(want) && slices.Equal(arguments, want)
	}
	switch subcommand {
	case "init":
		if exact("--quiet") {
			return nil
		}
	case "fetch":
		if len(arguments) == 4 && arguments[0] == "--quiet" && arguments[1] == "--no-tags" && nonOption(arguments[2]) && nonOption(arguments[3]) {
			return nil
		}
	case "rev-parse":
		if len(arguments) == 1 && nonOption(arguments[0]) || len(arguments) == 2 && arguments[0] == "--verify" && nonOption(arguments[1]) {
			return nil
		}
	case "cat-file":
		if len(arguments) == 2 && arguments[0] == "tag" && nonOption(arguments[1]) {
			return nil
		}
	case "show":
		if len(arguments) == 1 && nonOption(arguments[0]) && strings.Contains(arguments[0], ":") {
			return nil
		}
	case "commit-tree":
		if len(arguments) >= 5 && nonOption(arguments[0]) {
			index := 1
			parents := 0
			for index+1 < len(arguments) && arguments[index] == "-p" && nonOption(arguments[index+1]) {
				parents++
				index += 2
			}
			if parents > 0 && index+2 == len(arguments) && arguments[index] == "-m" && arguments[index+1] != "" {
				return nil
			}
		}
	case "checkout":
		valid := len(arguments) == 2 && exact("--quiet", "--detach") ||
			len(arguments) == 3 && arguments[0] == "--quiet" && arguments[1] == "--detach" && nonOption(arguments[2]) ||
			len(arguments) == 3 && arguments[0] == "--quiet" && arguments[1] == "-b" && nonOption(arguments[2]) ||
			len(arguments) == 4 && arguments[0] == "--quiet" && arguments[1] == "-b" && nonOption(arguments[2]) && nonOption(arguments[3]) ||
			len(arguments) == 4 && arguments[0] == "--quiet" && arguments[1] == "--force" && arguments[2] == "--detach" && nonOption(arguments[3]) ||
			len(arguments) == 5 && arguments[0] == "--quiet" && arguments[1] == "--force" && arguments[2] == "-B" && nonOption(arguments[3]) && nonOption(arguments[4])
		if valid {
			return nil
		}
	case "tag":
		if len(arguments) == 2 && arguments[0] == "-d" && nonOption(arguments[1]) ||
			len(arguments) == 5 && arguments[0] == "-a" && arguments[1] == "-m" && arguments[2] != "" && nonOption(arguments[3]) && nonOption(arguments[4]) {
			return nil
		}
	case "branch":
		if len(arguments) == 2 && arguments[0] == "-m" && nonOption(arguments[1]) {
			return nil
		}
	case "add":
		if len(arguments) >= 2 && arguments[0] == "--" {
			for _, path := range arguments[1:] {
				if !nonOption(path) {
					return fmt.Errorf("Git add path %q is not admitted", path)
				}
			}
			return nil
		}
	case "commit":
		if len(arguments) == 4 && arguments[0] == "--quiet" && arguments[1] == "--no-gpg-sign" && arguments[2] == "-m" && arguments[3] != "" {
			return nil
		}
	case "diff":
		if len(arguments) == 3 && arguments[0] == "--name-only" && nonOption(arguments[1]) && arguments[2] == "--" {
			return nil
		}
	case "config":
		if len(arguments) == 2 && arguments[0] == "--get" && nonOption(arguments[1]) ||
			len(arguments) == 3 && (arguments[0] == "--local" || arguments[0] == "--global") && arguments[1] == "--get" && nonOption(arguments[2]) {
			return nil
		}
	}
	return fmt.Errorf("Git argv is outside the exact %s fixture schema: %q", subcommand, arguments)
}

func TestPlanningGrantTestGitCommandDisablesBackgroundMaintenance(t *testing.T) {
	parent := planningGrantCanonicalTempDir(t)
	hostileGlobal := filepath.Join(parent, "hostile-global-config")
	if err := os.WriteFile(hostileGlobal, []byte("[maintenance]\n\tauto = true\n[gc]\n\tauto = 999\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", hostileGlobal)
	root := filepath.Join(parent, "repo")
	initializePlanningGrantTestRepository(t, root, "", "")
	for key, want := range map[string]string{
		"core.hooksPath":         os.DevNull,
		"maintenance.auto":       "false",
		"gc.auto":                "0",
		"gc.autoDetach":          "false",
		"maintenance.autoDetach": "false",
	} {
		if got := planningGrantTestGitOutput(t, root, "config", "--get", key); got != want {
			t.Fatalf("bounded disposable Git config %s=%q, want %q", key, got, want)
		}
		assertPlanningGrantTestGitConfigAbsent(t, root, "--local", key)
		assertPlanningGrantTestGitConfigAbsent(t, root, "--global", key)
	}
}

func TestPlanningGrantTestGitCommandRejectsAmbientExecutionInjection(t *testing.T) {
	source, err := filepath.Abs(filepath.Clean(filepath.Join("..", "..")))
	if err != nil {
		t.Fatal(err)
	}
	root := planningGrantCanonicalTempDir(t)
	execRoot := filepath.Join(root, "hostile-exec")
	templateRoot := filepath.Join(root, "hostile-template")
	if err := os.MkdirAll(filepath.Join(templateRoot, "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(execRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	execSentinel := filepath.Join(root, "exec-path-ran")
	hookSentinel := filepath.Join(root, "template-hook-ran")
	localHookSentinel := filepath.Join(root, "local-hook-ran")
	uploadPack := []byte(fmt.Sprintf("#!/bin/sh\n: > %q\nexec /usr/bin/git-upload-pack \"$@\"\n", execSentinel))
	postCheckout := []byte(fmt.Sprintf("#!/bin/sh\n: > %q\n", hookSentinel))
	localPostCheckout := []byte(fmt.Sprintf("#!/bin/sh\n: > %q\n", localHookSentinel))
	if err := os.WriteFile(filepath.Join(execRoot, "git-upload-pack"), uploadPack, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(templateRoot, "hooks", "post-checkout"), postCheckout, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_EXEC_PATH", execRoot)
	t.Setenv("GIT_TEMPLATE_DIR", templateRoot)
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "maintenance.auto")
	t.Setenv("GIT_CONFIG_VALUE_0", "true")
	t.Setenv("GIT_ALLOW_PROTOCOL", "ext:ssh:http:https:file")
	t.Setenv("PERL5LIB", execRoot)
	t.Setenv("PERL5OPT", "-MDefinitelyMissingMars3TestModule")
	t.Setenv("LD_PRELOAD", "/nonexistent/mars3-test-loader.so")
	t.Setenv("DYLD_INSERT_LIBRARIES", "/nonexistent/mars3-test-loader.dylib")
	t.Setenv("PATH", execRoot)
	clone := filepath.Join(root, "repo")
	initializePlanningGrantTestRepository(t, clone, source, "HEAD")
	if err := os.WriteFile(filepath.Join(clone, ".git", "hooks", "post-checkout"), localPostCheckout, 0o755); err != nil {
		t.Fatal(err)
	}
	runPlanningGrantTestGit(t, clone, "checkout", "--quiet", "--force", "--detach", "FETCH_HEAD")
	for _, sentinel := range []string{execSentinel, hookSentinel, localHookSentinel} {
		if _, err := os.Lstat(sentinel); !os.IsNotExist(err) {
			t.Fatalf("ambient Git execution injection ran: %s err=%v", sentinel, err)
		}
	}
	if installed, err := os.ReadFile(filepath.Join(clone, ".git", "hooks", "post-checkout")); err != nil || !bytes.Equal(installed, localPostCheckout) {
		t.Fatalf("ambient Git template replaced the local hook fixture: err=%v", err)
	}
}

func TestPlanningGrantTestGitCommandRequiresCanonicalLocalRoot(t *testing.T) {
	t.Run("symlinked root", func(t *testing.T) {
		physicalRoot := planningGrantCanonicalTempDir(t)
		rootLink := filepath.Join(planningGrantCanonicalTempDir(t), "root-link")
		if err := os.Symlink(physicalRoot, rootLink); err != nil {
			t.Fatal(err)
		}
		if _, err := planningGrantTestGitCommand(rootLink, "init", "--quiet"); err == nil {
			t.Fatal("symlinked disposable root was admitted")
		}
	})

	t.Run("symlinked root ancestor", func(t *testing.T) {
		physicalParent := planningGrantCanonicalTempDir(t)
		physicalRoot := filepath.Join(physicalParent, "repo")
		if err := os.Mkdir(physicalRoot, 0o700); err != nil {
			t.Fatal(err)
		}
		aliasParent := filepath.Join(planningGrantCanonicalTempDir(t), "parent-alias")
		if err := os.Symlink(physicalParent, aliasParent); err != nil {
			t.Fatal(err)
		}
		if _, err := planningGrantTestGitCommand(filepath.Join(aliasParent, "repo"), "init", "--quiet"); err == nil {
			t.Fatal("disposable root below a symlinked ancestor was admitted")
		}
	})
}

func TestPlanningGrantTestGitFetchCreatesPortableFetchHead(t *testing.T) {
	parent := planningGrantCanonicalTempDir(t)
	source := filepath.Join(parent, "source")
	target := filepath.Join(parent, "target")
	initializePlanningGrantTestRepository(t, source, "", "")
	writePlanningGrantTestFile(t, source, "identity.txt", []byte("portable fetch\n"))
	commitPlanningGrantTestPaths(t, source, "portable fetch", "identity.txt")
	expected := planningGrantTestGitOutput(t, source, "rev-parse", "HEAD^{commit}")
	initializePlanningGrantTestRepository(t, target, "", "")
	runPlanningGrantTestGit(t, target, "fetch", "--quiet", "--no-tags", source, "HEAD")
	if got := planningGrantTestGitOutput(t, target, "rev-parse", "FETCH_HEAD^{commit}"); got != expected {
		t.Fatalf("ordinary fetch wrote FETCH_HEAD=%s, want %s", got, expected)
	}
}

func TestPlanningGrantTestGitArgumentsFailClosed(t *testing.T) {
	for _, arguments := range [][]string{
		{"-c", "maintenance.auto=true", "status"},
		{"-cgc.auto=999", "status"},
		{"status", "-c", "maintenance.auto=true"},
		{"--config-env=maintenance.auto=HOSTILE", "status"},
		{"-C", "/tmp/hostile", "status"},
		{"--exec-path=/tmp/hostile", "status"},
		{"clone", "--quiet", "--no-local", "/tmp/source", "/tmp/target"},
		{"clone", "--template=/tmp/hostile", "source", "target"},
		{"clone", "--templ=/tmp/hostile", "source", "target"},
		{"clone", "--upload-p=/tmp/hostile-upload-pack", "source", "target"},
		{"clone", "--conf=maintenance.auto=true", "source", "target"},
		{"clone", "-u", "/tmp/hostile-upload-pack", "source", "target"},
		{"clone", "-u/tmp/hostile-upload-pack", "source", "target"},
		{"clone", "--separate-git-dir=/tmp/outside", "source", "target"},
		{"clone", "--quiet", "--no-local", "/tmp/source", "/tmp/outside"},
		{"fetch", "--quiet", "--no-tags", "https://example.com/repository.git", "HEAD"},
		{"fetch", "--quiet", "--no-tags", "ssh://example.com/repository.git", "HEAD"},
		{"remote", "add", "hostile", "/tmp/source"},
		{"config", "--local", "maintenance.auto", "true"},
		{"config", "--global", "gc.auto", "999"},
	} {
		if _, err := planningGrantTestGitCommand(planningGrantCanonicalTempDir(t), arguments...); err == nil {
			t.Fatalf("unsafe disposable Git arguments were accepted: %v", arguments)
		}
	}
	t.Run("symlinked fetch source", func(t *testing.T) {
		root := planningGrantCanonicalTempDir(t)
		source := planningGrantCanonicalTempDir(t)
		alias := filepath.Join(planningGrantCanonicalTempDir(t), "source-alias")
		if err := os.Symlink(source, alias); err != nil {
			t.Fatal(err)
		}
		if _, err := planningGrantTestGitCommand(root, "fetch", "--quiet", "--no-tags", alias, "HEAD"); err == nil {
			t.Fatal("symlinked bounded Git fetch source was admitted")
		}
	})
	t.Run("malformed fetch refspec", func(t *testing.T) {
		root := planningGrantCanonicalTempDir(t)
		source := planningGrantCanonicalTempDir(t)
		for _, revision := range []string{
			"HEAD\n--all",
			"refs/tags/one:refs/tags/two",
			"refs/tags/../escape:refs/tags/../escape",
			"refs/tags/name.lock:refs/tags/name.lock",
		} {
			if _, err := planningGrantTestGitCommand(root, "fetch", "--quiet", "--no-tags", source, revision); err == nil {
				t.Fatalf("malformed bounded Git fetch refspec was admitted: %q", revision)
			}
		}
	})
}

func TestPlanningGrantCommitAuthorityIsProspective(t *testing.T) {
	issuedAt := time.Date(2026, time.August, 29, 0, 28, 29, 0, time.UTC)
	for _, test := range []struct {
		name        string
		committedAt time.Time
		want        bool
	}{
		{name: "nineteen seconds before issuance", committedAt: issuedAt.Add(-19 * time.Second), want: false},
		{name: "exactly at issuance", committedAt: issuedAt, want: true},
		{name: "after issuance", committedAt: issuedAt.Add(time.Second), want: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := planningGrantCommitAtOrAfterGrant(test.committedAt, issuedAt); got != test.want {
				t.Fatalf("planningGrantCommitAtOrAfterGrant(%s, %s) = %t, want %t", test.committedAt, issuedAt, got, test.want)
			}
		})
	}
}

func assertPlanningGrantTestGitConfigAbsent(t *testing.T, root, scope, key string) {
	t.Helper()
	command, commandErr := planningGrantTestGitCommand(root, "config", scope, "--get", key)
	if commandErr != nil {
		t.Fatal(commandErr)
	}
	output, err := command.CombinedOutput()
	if err == nil || len(bytes.TrimSpace(output)) != 0 {
		t.Fatalf("disposable Git configuration persisted %s %s: output=%q err=%v", scope, key, output, err)
	}
}
