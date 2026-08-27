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
	"os"
	"os/exec"
	"path/filepath"
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
	path := filepath.Join(t.TempDir(), "execution.json")
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
	root := t.TempDir()
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
			root := t.TempDir()
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
	root := t.TempDir()
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
	root := t.TempDir()
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
		root := t.TempDir()
		writePlanningGrantTestFile(t, root, w001PostclaimPRFixPath, grant)
		if w001PostclaimPullRequestNumberAllowed(root, 7) || !w001PostclaimPullRequestNumberAllowed(root, 8) {
			t.Fatal("signed active PR number was not enforced")
		}
	})
}

func writeW001PostclaimPRFixFixture(t *testing.T, grant, signature, publicKey []byte, materials map[string][]byte) string {
	t.Helper()
	root := t.TempDir()
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
	root := t.TempDir()
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
		root := t.TempDir()
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
		root := t.TempDir()
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
	root := t.TempDir()
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
	root := t.TempDir()
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
	exact := strings.Join(w001DeliveryScannerFingerprints, "\n") + "\n"
	for _, testCase := range []struct {
		name string
		body string
	}{
		{name: "changed", body: strings.Replace(exact, "0faf9071", "1faf9071", 1)},
		{name: "extra", body: exact + "*:generic-api-key:*\n"},
		{name: "missing", body: strings.TrimPrefix(exact, w001DeliveryScannerFingerprints[0]+"\n")},
		{name: "duplicate", body: exact + w001DeliveryScannerFingerprints[0] + "\n"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			root := t.TempDir()
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
	root := t.TempDir()
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

func TestW001DeliveryV2TagIdentityIsHistoricalOnly(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	object, err := planningGrantGitOutput(repo, "cat-file", "tag", w001DeliveryV2TagObject)
	if err != nil {
		t.Fatal(err)
	}
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
	repo := filepath.Clean(filepath.Join("..", ".."))
	root := filepath.Join(t.TempDir(), "repo")
	if output, err := exec.Command("git", "clone", "--quiet", "--no-local", repo, root).CombinedOutput(); err != nil {
		t.Fatalf("clone delivery fixture: %v: %s", err, output)
	}
	feature := planningGrantTestGitOutput(t, root, "rev-parse", "HEAD^{commit}")
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
	root := t.TempDir()
	runPlanningGrantTestGit(t, root, "init", "--quiet")
	disablePlanningGrantTestGitMaintenance(t, root)
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
	root := t.TempDir()
	runPlanningGrantTestGit(t, root, "init", "--quiet")
	disablePlanningGrantTestGitMaintenance(t, root)
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
	root := t.TempDir()
	runPlanningGrantTestGit(t, root, "init", "--quiet")
	disablePlanningGrantTestGitMaintenance(t, root)
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
	root := t.TempDir()
	runPlanningGrantTestGit(t, root, "init", "--quiet")
	disablePlanningGrantTestGitMaintenance(t, root)
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
	root := t.TempDir()
	runPlanningGrantTestGit(t, root, "init", "--quiet")
	disablePlanningGrantTestGitMaintenance(t, root)
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
	root := t.TempDir()
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
			root := t.TempDir()
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
			root := t.TempDir()
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
			root := t.TempDir()
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
			root := t.TempDir()
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
	checkWave1PlanningGrant(t.TempDir(), &findings)
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
	root := t.TempDir()
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
	root := t.TempDir()
	runPlanningGrantTestGit(t, root, "init", "--quiet")
	disablePlanningGrantTestGitMaintenance(t, root)
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
	root := t.TempDir()
	runPlanningGrantTestGit(t, root, "init", "--quiet")
	disablePlanningGrantTestGitMaintenance(t, root)
	for _, revision := range []string{wave1PublishedMain, sourceHead} {
		runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, revision)
	}
	for _, tag := range []string{wave1PriorPublicationTag, wave1V2PublicationTag, wave1PublicationTag, wave1TransitionTag, wave1SuccessorTransitionTag, wave1FinalTransitionTag} {
		if _, err := planningGrantGitOutput(source, "rev-parse", "--verify", "refs/tags/"+tag+"^{tag}"); err == nil {
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

func disablePlanningGrantTestGitMaintenance(t *testing.T, root string) {
	t.Helper()
	runPlanningGrantTestGit(t, root, "config", "maintenance.auto", "false")
	runPlanningGrantTestGit(t, root, "config", "gc.auto", "0")
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
	root := t.TempDir()
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
	root := t.TempDir()
	runPlanningGrantTestGit(t, root, "init", "--quiet")
	disablePlanningGrantTestGitMaintenance(t, root)
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, wave1V3AddendumBase)
	runPlanningGrantTestGit(t, root, "checkout", "--quiet", "--detach", "FETCH_HEAD")
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, "refs/tags/"+wave1PriorPublicationTag+":refs/tags/"+wave1PriorPublicationTag)
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, "refs/tags/"+wave1V2PublicationTag+":refs/tags/"+wave1V2PublicationTag)
	runPlanningGrantTestGit(t, root, "checkout", "--quiet", "-b", wave1PlanningGrantBranch)

	writePlanningGrantCurrentFiles(t, root)
	return root
}

func TestPlanningGrantGitFixtureDisablesAutoMaintenance(t *testing.T) {
	root := writePlanningGrantGitFixture(t)
	if value := planningGrantTestGitOutput(t, root, "config", "--local", "--get", "maintenance.auto"); value != "false" {
		t.Fatalf("disposable planning fixture enabled Git maintenance: %q", value)
	}
	if value := planningGrantTestGitOutput(t, root, "config", "--local", "--get", "gc.auto"); value != "0" {
		t.Fatalf("disposable planning fixture enabled Git auto-GC: %q", value)
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
	plan, err := planningGrantGitOutput(source, "show", wave1V3AddendumBase+":"+canonicalActivePlan)
	if err != nil {
		t.Fatal(err)
	}
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
	root := t.TempDir()
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

func runPlanningGrantTestGit(t *testing.T, root string, arguments ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", root}, arguments...)...)
	command.Env = planningGrantGitEnvironment()
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v: %s", arguments, err, output)
	}
}

func planningGrantTestGitOutput(t *testing.T, root string, arguments ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", root}, arguments...)...)
	command.Env = planningGrantGitEnvironment()
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v: %s", arguments, err, output)
	}
	return strings.TrimSpace(string(output))
}
