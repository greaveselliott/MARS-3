/*
FactoryDocSync:
docs:
- docs/features/F-001-doctrine-foundation.md
- docs/design-docs/mars-provenance.md
- docs/code-documentation-map.md
*/

package doctrine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRepositoryChecks(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	checks := []struct {
		name string
		run  func(string) ([]Finding, error)
	}{
		{name: "doctrine", run: CheckDoctrine},
		{name: "plan", run: CheckPlan},
		{name: "docsync", run: AuditDocSync},
		{name: "public", run: CheckPublic},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			findings, err := check.run(repo)
			if err != nil {
				t.Fatalf("check returned an error: %v", err)
			}
			if len(findings) != 0 {
				t.Fatalf("check returned findings: %v", findings)
			}
		})
	}
}

func TestDocSyncCommentForms(t *testing.T) {
	document := "docs/features/F-001-doctrine-foundation.md"
	tests := []struct {
		name string
		path string
		body string
	}{
		{
			name: "Go block",
			path: "sample.go",
			body: "/*\nFactoryDocSync:\ndocs:\n- " + document + "\n*/\npackage sample\n",
		},
		{
			name: "Go line",
			path: "sample.go",
			body: "// FactoryDocSync:\n// docs:\n// - " + document + "\npackage sample\n",
		},
		{
			name: "TypeScript compact JSON",
			path: "sample.ts",
			body: "// FactoryDocSync: {\"docs\":[\"" + document + "\"]}\nexport {};\n",
		},
		{
			name: "YAML comments",
			path: "sample.yaml",
			body: "# FactoryDocSync:\n# docs:\n# - " + document + "\nvalue: true\n",
		},
		{
			name: "HTML comments",
			path: "sample.html",
			body: "<!-- FactoryDocSync: {\"docs\":[\"" + document + "\"]} -->\n<main></main>\n",
		},
		{
			name: "CSS block comments",
			path: "sample.css",
			body: "/* FactoryDocSync: {\"docs\":[\"" + document + "\"]} */\n:root {}\n",
		},
		{
			name: "SQL line comments",
			path: "sample.sql",
			body: "-- FactoryDocSync:\n-- docs:\n-- - " + document + "\nselect 1;\n",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			docs, count, valid := parseDocSyncMarkers(test.path, []byte(test.body), factoryDocSyncMarker)
			if count != 1 {
				t.Fatalf("marker count = %d, want 1", count)
			}
			if valid != 1 {
				t.Fatalf("valid marker count = %d, want 1", valid)
			}
			if len(docs) != 1 || docs[0] != document {
				t.Fatalf("docs = %v, want %s", docs, document)
			}
		})
	}
}

func TestDocSyncRejectsMalformedMarker(t *testing.T) {
	body := "// FactoryDocSync: definitely-not-structured docs/features/F-001-doctrine-foundation.md\npackage sample\n"
	_, count, valid := parseDocSyncMarkers("sample.go", []byte(body), factoryDocSyncMarker)
	if count != 1 || valid != 0 {
		t.Fatalf("malformed marker count=%d valid=%d, want 1 and 0", count, valid)
	}
}

func TestStrictJSONRejectsTrailingValue(t *testing.T) {
	var target map[string]any
	if err := decodeStrictJSON([]byte("{} {}"), &target); err == nil {
		t.Fatal("trailing JSON value was accepted")
	}
}

func TestPublicWalkDoesNotSkipVendoredOrBuildContent(t *testing.T) {
	repo := t.TempDir()
	for _, path := range []string{"vendor/example/secret.txt", "dist/report.txt", "node_modules/example/data.txt"} {
		fullPath := filepath.Join(repo, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte("public\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	paths, err := walkPublicRepository(repo)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"dist/report.txt", "node_modules/example/data.txt", "vendor/example/secret.txt"} {
		if !containsString(paths, expected) {
			t.Fatalf("public walk omitted %s: %v", expected, paths)
		}
	}
}

func TestRoleParserIncludesProfileAuthorityFields(t *testing.T) {
	manifest := []byte("profiles:\n  - id: engineer\n    principal_id: engineer\n    mode: ticket-delivery\n    max_trust: contributor\n    effective_trust: observer\n    autonomous_mutation: disabled\n    prompt: .harness/roles/engineer.md\n")
	profiles := parseRoleDeclarations(manifest, "profiles")
	if len(profiles) != 1 {
		t.Fatalf("profiles = %d, want 1", len(profiles))
	}
	profile := profiles[0]
	if profile.principalID != "engineer" || profile.mode != "ticket-delivery" || profile.autonomy != "disabled" || profile.prompt != ".harness/roles/engineer.md" {
		t.Fatalf("profile authority fields were not parsed: %+v", profile)
	}
}

func TestRoleManifestRejectsProfileAuthorityEscalation(t *testing.T) {
	root := t.TempDir()
	rolesDirectory := filepath.Join(root, ".harness", "roles")
	if err := os.MkdirAll(rolesDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, role := range []string{"engineer", "foundation-maintainer"} {
		if err := os.WriteFile(filepath.Join(rolesDirectory, role+".md"), []byte("governed\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	manifest := `default_effective_trust: observer
autonomous_mutation: disabled
provenance:
  mars_commit: ` + marsCommit + `
security:
  accepted_base_label: public+project-accepted
  proposed_worktree_label: public+project-proposed
  rule_of_two: declared
  trace_spine: declared
roles:
  - id: engineer
    max_trust: contributor
    effective_trust: observer
    prompt: .harness/roles/engineer.md
foundation_roles:
  - id: foundation-maintainer
    scope: source-only
    max_trust: contributor
    effective_trust: observer
    prompt: .harness/roles/foundation-maintainer.md
profiles_grant_authority: true
profiles:
  - id: unsafe
    principal_id: engineer
    mode: ticket-delivery
    max_trust: contributor
    effective_trust: contributor
    autonomous_mutation: enabled
    prompt: .harness/roles/engineer.md
`
	if err := os.MkdirAll(filepath.Join(root, ".harness"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".harness", "manifest.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	var findings []Finding
	checkRoleManifest(root, &findings)
	for _, code := range []string{"doctrine.profile_authority", "doctrine.profile_effective_trust", "doctrine.profile_autonomy"} {
		if !findingCodePresent(findings, code) {
			t.Fatalf("missing %s in findings: %v", code, findings)
		}
	}
}

func TestPublicCheckRejectsBinaryInGovernedSource(t *testing.T) {
	var findings []Finding
	checkPublicContent(t.TempDir(), "fixture.png", []byte{0x00, 0x01, 0x02}, &findings)
	if !findingCodePresent(findings, "public.binary") {
		t.Fatalf("binary content was accepted: %v", findings)
	}
}

func TestGovernedPublicScopeAllowsDocSyncMarkedWave1Sources(t *testing.T) {
	tests := []struct {
		name string
		path string
		body string
	}{
		{
			name: "work authority Go source",
			path: "internal/authority/gateway.go",
			body: `/*
FactoryDocSync:
docs:
- docs/features/F-002-work-authority.md
- docs/design-docs/ADR-001-git-beads-authority.md
- docs/code-documentation-map.md
*/
package authority
`,
		},
		{
			name: "platform chart YAML",
			path: "charts/mars3/templates/config.yaml",
			body: `# FactoryDocSync:
# docs:
# - docs/features/F-003-local-substrate.md
# - docs/design-docs/ADR-006-local-substrate.md
# - docs/code-documentation-map.md
apiVersion: v1
kind: ConfigMap
`,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			var findings []Finding
			checkGovernedPublicScope(testCase.path, &findings)
			checkPublicContent(t.TempDir(), testCase.path, []byte(testCase.body), &findings)
			if len(findings) != 0 {
				t.Fatalf("valid Wave-1 source was rejected: %v", findings)
			}
			_, count, valid := parseDocSyncMarkers(testCase.path, []byte(testCase.body), factoryDocSyncMarker)
			if count != 1 || valid != 1 {
				t.Fatalf("fixture DocSync marker count=%d valid=%d, want 1 and 1", count, valid)
			}
		})
	}
}

func TestGovernedPublicScopeRejectsUnrelatedAndEscapingPaths(t *testing.T) {
	for _, path := range []string{
		"internal/private/gateway.go",
		"internal/runtime/adapter.go",
		"cmd/unrelated/main.go",
		"internal/authority/../../private.go",
		"../outside.go",
	} {
		t.Run(path, func(t *testing.T) {
			var findings []Finding
			checkGovernedPublicScope(path, &findings)
			if !findingCodePresent(findings, "public.scope") {
				t.Fatalf("unrelated or escaping path was accepted: %s (%v)", path, findings)
			}
		})
	}
}

func TestGovernedPublicScopeAllowsPinnedScannerFingerprintFile(t *testing.T) {
	var findings []Finding
	checkGovernedPublicScope(".gitleaksignore", &findings)
	if len(findings) != 0 {
		t.Fatalf("governed exact-fingerprint file was rejected: %v", findings)
	}
}

func TestClaimAttestationRequiresCompleteWorkLineage(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	claim, err := os.ReadFile(filepath.Join(repo, ".harness", "claims", "H-001.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	signature, err := os.ReadFile(filepath.Join(repo, ".harness", "claims", "H-001.yaml.sig"))
	if err != nil {
		t.Fatal(err)
	}
	publicKey, err := os.ReadFile(filepath.Join(repo, ".harness", "keys", "genesis-signing-key.pub"))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name        string
		claim       []byte
		findingCode string
	}{
		{name: "missing feature", claim: []byte(strings.Replace(string(claim), "  feature: F-001\n", "", 1)), findingCode: "doctrine.claim_attestation_value"},
		{name: "missing product decision", claim: []byte(strings.Replace(string(claim), "    - PD-003\n", "", 1)), findingCode: "doctrine.claim_attestation_scope"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.MkdirAll(filepath.Join(root, ".harness", "claims"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Join(root, ".harness", "keys"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, ".harness", "claims", "H-001.yaml"), testCase.claim, 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, ".harness", "claims", "H-001.yaml.sig"), signature, 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, ".harness", "keys", "genesis-signing-key.pub"), publicKey, 0o644); err != nil {
				t.Fatal(err)
			}
			var findings []Finding
			checkBootstrapClaimAttestation(root, &findings)
			if !findingCodePresent(findings, testCase.findingCode) {
				t.Fatalf("incomplete claim lineage was accepted: %v", findings)
			}
		})
	}
}

func TestClaimAttestationRequiresCurrentAuthorityBinding(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	claim, err := os.ReadFile(filepath.Join(repo, ".harness", "claims", "H-001.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	signature, err := os.ReadFile(filepath.Join(repo, ".harness", "claims", "H-001.yaml.sig"))
	if err != nil {
		t.Fatal(err)
	}
	publicKey, err := os.ReadFile(filepath.Join(repo, ".harness", "keys", "genesis-signing-key.pub"))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name    string
		current string
		stale   string
	}{
		{name: "ledger head", current: "blsidb8htct7d687cijiqcp51488jqo5", stale: "icj9j2a6h0nsrb3q9705nm6tgt75kr3p"},
		{name: "claim checkpoint", current: "kvofc5q57reond5aki5pgdcgfog8u7dr", stale: "pgi99ie4dpqvutoiv59b8ca8stmk466i"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.MkdirAll(filepath.Join(root, ".harness", "claims"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Join(root, ".harness", "keys"), 0o755); err != nil {
				t.Fatal(err)
			}
			staleClaim := []byte(strings.Replace(string(claim), testCase.current, testCase.stale, 1))
			if bytesEqual(staleClaim, claim) {
				t.Fatalf("claim does not contain current %s", testCase.name)
			}
			if err := os.WriteFile(filepath.Join(root, ".harness", "claims", "H-001.yaml"), staleClaim, 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, ".harness", "claims", "H-001.yaml.sig"), signature, 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, ".harness", "keys", "genesis-signing-key.pub"), publicKey, 0o644); err != nil {
				t.Fatal(err)
			}
			var findings []Finding
			checkBootstrapClaimAttestation(root, &findings)
			if !findingCodePresent(findings, "doctrine.claim_attestation_value") {
				t.Fatalf("stale %s was accepted: %v", testCase.name, findings)
			}
		})
	}
}

func TestClaimVerificationOrderReferencesExecutableRegistry(t *testing.T) {
	manifest := []byte(`roles:
  - id: qa
  - id: security-reviewer
  - id: delivery-orchestrator
profiles:
  - id: qa
    principal_id: qa
  - id: security-reviewer
    principal_id: security-reviewer
  - id: delivery-orchestrator
    principal_id: delivery-orchestrator
`)
	claim := []byte(`verification:
  order:
    - qa
    - security-reviewer
    - delivery-orchestrator
`)
	var findings []Finding
	checkClaimVerificationOrderData(".harness/claims/H-001.yaml", claim, manifest, &findings)
	if len(findings) != 0 {
		t.Fatalf("declared verification principals were rejected: %v", findings)
	}
}

func TestClaimVerificationOrderRejectsNonCanonicalSequence(t *testing.T) {
	manifest := []byte(`roles:
  - id: qa
  - id: security-reviewer
  - id: delivery-orchestrator
  - id: dogfood
profiles:
  - id: qa
    principal_id: qa
  - id: security-reviewer
    principal_id: security-reviewer
  - id: delivery-orchestrator
    principal_id: delivery-orchestrator
  - id: dogfood
    principal_id: dogfood
`)
	cases := []struct {
		name  string
		order string
	}{
		{name: "incomplete", order: "    - qa\n    - security-reviewer\n"},
		{name: "reordered", order: "    - security-reviewer\n    - qa\n    - delivery-orchestrator\n"},
		{name: "extra routable reviewer", order: "    - qa\n    - dogfood\n    - security-reviewer\n    - delivery-orchestrator\n"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			claim := []byte("verification:\n  order:\n" + testCase.order)
			var findings []Finding
			checkClaimVerificationOrderData(".harness/claims/H-001.yaml", claim, manifest, &findings)
			if !findingCodePresent(findings, "doctrine.claim_verification_order_exact") {
				t.Fatalf("non-canonical verification order was accepted: %v", findings)
			}
			if findingCodePresent(findings, "doctrine.claim_verifier_unroutable") {
				t.Fatalf("routable identity was misclassified while checking exact order: %v", findings)
			}
		})
	}
}

func TestClaimVerificationOrderRejectsUnroutableOrMalformedEntries(t *testing.T) {
	manifest := []byte(`roles:
  - id: qa
  - id: security-reviewer
profiles:
  - id: qa
    principal_id: qa
  - id: security-reviewer
    principal_id: security-reviewer
`)
	cases := []struct {
		name        string
		claim       string
		findingCode string
	}{
		{
			name:        "undeclared reviewer alias",
			claim:       "verification:\n  order:\n    - qa-reviewer\n    - security-reviewer\n",
			findingCode: "doctrine.claim_verifier_unroutable",
		},
		{
			name:        "duplicate reviewer",
			claim:       "verification:\n  order:\n    - qa\n    - qa\n",
			findingCode: "doctrine.claim_verifier_duplicate",
		},
		{
			name:        "inline sequence",
			claim:       "verification:\n  order: [qa, security-reviewer]\n",
			findingCode: "doctrine.claim_verification_order",
		},
		{
			name:        "nested mapping",
			claim:       "verification:\n  order:\n    reviewer: qa\n",
			findingCode: "doctrine.claim_verification_order",
		},
		{
			name:        "nested sequence",
			claim:       "verification:\n  order:\n    - qa\n      - security-reviewer\n",
			findingCode: "doctrine.claim_verification_order",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			var findings []Finding
			checkClaimVerificationOrderData(".harness/claims/H-001.yaml", []byte(testCase.claim), manifest, &findings)
			if !findingCodePresent(findings, testCase.findingCode) {
				t.Fatalf("verification order bypass was accepted; wanted %s, got %v", testCase.findingCode, findings)
			}
		})
	}
}

func TestWorkflowRejectsMutableContainerReference(t *testing.T) {
	var findings []Finding
	checkWorkflow(".github/workflows/test.yml", "permissions:\n  contents: read\nsteps:\n  - run: docker run docker.io/example/tool:latest\n", &findings)
	if !findingCodePresent(findings, "public.unpinned_container") {
		t.Fatalf("mutable container reference was accepted: %v", findings)
	}
}

func TestWorkflowPermissionsAcceptsExactTopLevelReadOnly(t *testing.T) {
	content := `name: test
permissions:
  contents: read
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: |
          printf '%s\n' 'permissions: { contents: write }'
`
	var findings []Finding
	checkWorkflow(".github/workflows/test.yml", content, &findings)
	if findingCodePresent(findings, "public.workflow_permissions") {
		t.Fatalf("exact top-level read-only permissions were rejected: %v", findings)
	}
}

func TestWorkflowPermissionsRejectsInlineMapping(t *testing.T) {
	for _, declaration := range []string{
		"permissions: { contents: write }",
		`permissions: {"contents": "write"}`,
		"permissions: { contents: read }",
		"permissions: &read-only",
		"permissions: *read-only",
		"permissions:\n\tcontents: read",
	} {
		t.Run(declaration, func(t *testing.T) {
			var findings []Finding
			checkWorkflow(".github/workflows/test.yml", declaration+"\njobs: {}\n", &findings)
			if !findingCodePresent(findings, "public.workflow_permissions") {
				t.Fatalf("inline permission mapping was accepted: %s", declaration)
			}
		})
	}
}

func TestWorkflowPermissionsRejectsJobLevelDeclaration(t *testing.T) {
	for _, declaration := range []string{
		"    permissions:\n      contents: write",
		"    permissions: { contents: write }",
		"    'permissions':\n      contents: read",
		"    test: { permissions: { contents: write }, runs-on: ubuntu-latest }",
	} {
		t.Run(declaration, func(t *testing.T) {
			content := "permissions:\n  contents: read\njobs:\n  test:\n" + declaration + "\n    runs-on: ubuntu-latest\n"
			var findings []Finding
			checkWorkflow(".github/workflows/test.yml", content, &findings)
			if !findingCodePresent(findings, "public.workflow_permissions") {
				t.Fatalf("job-level permission declaration was accepted:\n%s", content)
			}
		})
	}
}

func TestWorkflowPermissionsRejectsDuplicateTopLevelDeclarations(t *testing.T) {
	content := "permissions:\n  contents: read\npermissions:\n  contents: read\njobs: {}\n"
	var findings []Finding
	checkWorkflow(".github/workflows/test.yml", content, &findings)
	if !findingCodePresent(findings, "public.workflow_permissions") {
		t.Fatalf("duplicate top-level permission declarations were accepted: %v", findings)
	}
}

func TestWorkflowPermissionsRejectsFlowStyleAndEscapedKeys(t *testing.T) {
	for _, jobs := range []string{
		`jobs: { test: { permissions: { contents: write }, runs-on: ubuntu-latest } }`,
		`jobs: { test: { "permissio\u006es": { contents: write }, runs-on: ubuntu-latest } }`,
		`jobs: { test: { "\u0070ermissions": { contents: write }, runs-on: ubuntu-latest } }`,
	} {
		t.Run(jobs, func(t *testing.T) {
			content := "permissions:\n  contents: read\n" + jobs + "\n"
			var findings []Finding
			checkWorkflow(".github/workflows/test.yml", content, &findings)
			if !findingCodePresent(findings, "public.workflow_permissions") {
				t.Fatalf("flow-style or escaped permission key was accepted:\n%s", content)
			}
		})
	}
}

func TestWorkflowPermissionsRejectsExplicitMappingKeys(t *testing.T) {
	for _, declaration := range []string{
		"    ? permissions\n    : { contents: write }",
		"    - ? permissions\n      : { contents: write }",
	} {
		t.Run(declaration, func(t *testing.T) {
			content := "permissions:\n  contents: read\njobs:\n  test:\n" + declaration + "\n"
			var findings []Finding
			checkWorkflow(".github/workflows/test.yml", content, &findings)
			if !findingCodePresent(findings, "public.workflow_permissions") {
				t.Fatalf("explicit YAML permission key was accepted:\n%s", content)
			}
		})
	}
}

func TestWorkflowPermissionsRejectsAliasedPermissionKey(t *testing.T) {
	content := `name: &permission_key permissions
permissions:
  contents: read
jobs:
  test:
    runs-on: ubuntu-latest
    *permission_key:
      contents: write
    steps: []
`
	var findings []Finding
	checkWorkflow(".github/workflows/test.yml", content, &findings)
	if !findingCodePresent(findings, "public.workflow_permissions") {
		t.Fatalf("aliased job permission key was accepted:\n%s", content)
	}
}

func TestWorkflowPermissionsAllowsAnchorCharactersOnlyAsData(t *testing.T) {
	content := `name: test
permissions:
  contents: read
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: "printf '%s' 'a & b * c'"
      - run: |
          printf '%s' '&anchor *alias'
`
	var findings []Finding
	checkWorkflow(".github/workflows/test.yml", content, &findings)
	if findingCodePresent(findings, "public.workflow_permissions") {
		t.Fatalf("quoted or block-scalar data was treated as YAML indirection: %v", findings)
	}
}

func TestCanonicalFoundationWorkflowPolicy(t *testing.T) {
	content := canonicalFoundationWorkflow(t)
	var findings []Finding
	checkWorkflow(".github/workflows/foundation-quality.yml", content, &findings)
	if len(findings) != 0 {
		t.Fatalf("committed foundation workflow was rejected: %v", findings)
	}

	crlf := strings.ReplaceAll(content, "\n", "\r\n")
	findings = nil
	checkWorkflow(".github/workflows/foundation-quality.yml", crlf, &findings)
	if len(findings) != 0 {
		t.Fatalf("CRLF canonical workflow was rejected: %v", findings)
	}
}

func TestImmutableFoundationWorkflowRejectsEveryExecutionMutation(t *testing.T) {
	base := canonicalFoundationWorkflow(t)
	cases := []struct {
		name    string
		content string
	}{
		{name: "disabled step", content: replaceWorkflowFixture(t, base, "        run: go build", "        if: false\n        run: go build")},
		{name: "continued failure", content: replaceWorkflowFixture(t, base, "        run: go build", "        continue-on-error: true\n        run: go build")},
		{name: "neutralized test", content: replaceWorkflowFixture(t, base, "          go test ./...", "          true")},
		{name: "masked test failure", content: replaceWorkflowFixture(t, base, "          go test ./...", "          go test ./... || true")},
		{name: "self hosted runner", content: replaceWorkflowFixture(t, base, "    runs-on: ubuntu-24.04", "    runs-on: self-hosted")},
		{name: "extra network step", content: replaceWorkflowFixture(t, base, "      - name: Check doctrine", "      - name: Extra command\n        run: curl -fsS https://example.com/\n\n      - name: Check doctrine")},
		{name: "custom shell", content: replaceWorkflowFixture(t, base, "        run: go build", "        shell: python\n        run: go build")},
		{name: "dynamic container", content: replaceWorkflowFixture(t, base, "          go vet ./...", "          go vet ./...\n          d=docker\n          \"$d\" run alpine:latest")},
		{name: "additional arbitrary step", content: replaceWorkflowFixture(t, base, "      - name: Check whitespace", "      - name: Unreviewed step\n        run: true\n\n      - name: Check whitespace")},
		{name: "apparently harmless comment", content: "# workflow changes require a new contract\n" + base},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			var findings []Finding
			checkWorkflow(canonicalFoundationWorkflowPath, testCase.content, &findings)
			if !findingCodePresent(findings, "public.workflow_contract") {
				t.Fatalf("workflow mutation was accepted: %v", findings)
			}
		})
	}

	var findings []Finding
	checkWorkflow(".github/workflows/another.yml", base, &findings)
	if !findingCodePresent(findings, "public.workflow_contract") {
		t.Fatalf("additional workflow path was accepted: %v", findings)
	}
	findings = nil
	checkPublicContent(t.TempDir(), ".github/workflows/foundation-quality.YML", []byte(base), &findings)
	if !findingCodePresent(findings, "public.workflow_contract") {
		t.Fatalf("case-variant workflow extension bypassed the path contract: %v", findings)
	}
}

func TestCanonicalFoundationWorkflowRejectsAuthorityBypasses(t *testing.T) {
	base := canonicalFoundationWorkflow(t)
	action := "actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683"
	container := allowedWorkflowContainer
	cases := []struct {
		name        string
		content     string
		findingCode string
	}{
		{name: "quoted privileged trigger", content: replaceWorkflowFixture(t, base, "  pull_request:\n", "  \"pull_request_target\":\n"), findingCode: "public.workflow_yaml"},
		{name: "escaped privileged trigger", content: replaceWorkflowFixture(t, base, "  pull_request:\n", `  "pull_request\u005ftarget":`+"\n"), findingCode: "public.workflow_yaml"},
		{name: "literal privileged trigger", content: replaceWorkflowFixture(t, base, "  pull_request:\n", "  pull_request_target:\n"), findingCode: "public.workflow_event"},
		{name: "flow trigger map", content: replaceWorkflowFixture(t, base, "on:\n  push:\n    branches: [main]\n  pull_request:\n  workflow_dispatch:\n", "on: { push: null, pull_request_target: null }\n"), findingCode: "public.workflow_yaml"},
		{name: "trigger list", content: replaceWorkflowFixture(t, base, "on:\n  push:\n    branches: [main]\n  pull_request:\n  workflow_dispatch:\n", "on: [push, pull_request_target]\n"), findingCode: "public.workflow_yaml"},
		{name: "duplicate trigger", content: replaceWorkflowFixture(t, base, "  pull_request:\n", "  pull_request:\n  pull_request:\n"), findingCode: "public.workflow_event"},
		{name: "secret bracket context", content: replaceWorkflowFixture(t, base, "concurrency:\n", "env:\n  VALUE: ${{ secrets['CANARY'] }}\n\nconcurrency:\n"), findingCode: "public.workflow_secret"},
		{name: "secret mixed context", content: replaceWorkflowFixture(t, base, "concurrency:\n", "env:\n  VALUE: ${{ SeCrEtS . CANARY }}\n\nconcurrency:\n"), findingCode: "public.workflow_secret"},
		{name: "github token bracket context", content: replaceWorkflowFixture(t, base, "concurrency:\n", "env:\n  VALUE: ${{ github['token'] }}\n\nconcurrency:\n"), findingCode: "public.workflow_secret"},
		{name: "github token dynamic context", content: replaceWorkflowFixture(t, base, "${{ github.workflow }}", "${{ github[format('{0}{1}', 'to', 'ken')] }}"), findingCode: "public.workflow_secret"},
		{name: "duplicate allowed expression", content: replaceWorkflowFixture(t, base, "name: Foundation quality", "name: Foundation quality ${{ github.workflow }}"), findingCode: "public.workflow_secret"},
		{name: "block scalar secret expression", content: replaceWorkflowFixture(t, base, "          go test ./...\n", "          printf '%s\\n' \"${{ secrets.CANARY }}\"\n          go test ./...\n"), findingCode: "public.workflow_secret"},
		{name: "explicit token key", content: replaceWorkflowFixture(t, base, "    timeout-minutes: 20\n", "    timeout-minutes: 20\n    token: synthetic\n"), findingCode: "public.workflow_secret"},
		{name: "secrets inheritance", content: replaceWorkflowFixture(t, base, "    timeout-minutes: 20\n", "    timeout-minutes: 20\n    secrets: inherit\n"), findingCode: "public.workflow_secret"},
		{name: "container credentials", content: replaceWorkflowFixture(t, base, "    timeout-minutes: 20\n", "    timeout-minutes: 20\n    credentials:\n      username: synthetic\n"), findingCode: "public.workflow_secret"},
		{name: "action tag", content: replaceWorkflowFixture(t, base, action, "actions/checkout@v4"), findingCode: "public.workflow_action"},
		{name: "action expression", content: replaceWorkflowFixture(t, base, action, "actions/checkout@${{ github.ref }}"), findingCode: "public.workflow_action"},
		{name: "action uppercase SHA", content: replaceWorkflowFixture(t, base, action, "actions/checkout@11BD71901BBE5B1630CEEA73D27597364C9AF683"), findingCode: "public.workflow_action"},
		{name: "arbitrary pinned action", content: replaceWorkflowFixture(t, base, action, "example/action@11bd71901bbe5b1630ceea73d27597364c9af683"), findingCode: "public.workflow_action"},
		{name: "local action", content: replaceWorkflowFixture(t, base, action, "./local-action"), findingCode: "public.workflow_action"},
		{name: "Docker action", content: replaceWorkflowFixture(t, base, action, "docker://example/image@sha256:75bdb2b2f4db213cde0b8295f13a88d6b333091bbfbf3012a4e083d00d31caba"), findingCode: "public.workflow_action"},
		{name: "quoted uses key", content: replaceWorkflowFixture(t, base, "        uses: "+action, "        \"uses\": "+action), findingCode: "public.workflow_yaml"},
		{name: "quoted action value", content: replaceWorkflowFixture(t, base, action, `"`+action+`"`), findingCode: "public.workflow_action"},
		{name: "checkout extra input", content: replaceWorkflowFixture(t, base, "          persist-credentials: false\n", "          persist-credentials: false\n          repository: example/other\n"), findingCode: "public.workflow_action"},
		{name: "checkout privileged token input", content: replaceWorkflowFixture(t, base, "          persist-credentials: false\n", "          persist-credentials: false\n          github-token: synthetic\n"), findingCode: "public.workflow_action"},
		{name: "duplicate action with block", content: replaceWorkflowFixture(t, base, "          persist-credentials: false\n", "          persist-credentials: false\n        with:\n          fetch-depth: 0\n          persist-credentials: false\n"), findingCode: "public.workflow_action"},
		{name: "job reusable workflow", content: replaceWorkflowFixture(t, base, "    timeout-minutes: 20\n", "    timeout-minutes: 20\n    uses: actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683\n"), findingCode: "public.workflow_action"},
		{name: "job container", content: replaceWorkflowFixture(t, base, "    timeout-minutes: 20\n", "    timeout-minutes: 20\n    container: node:18\n"), findingCode: "public.workflow_container"},
		{name: "job services", content: replaceWorkflowFixture(t, base, "    timeout-minutes: 20\n", "    timeout-minutes: 20\n    services:\n      cache:\n        image: redis:latest\n"), findingCode: "public.workflow_container"},
		{name: "mutable shell container", content: replaceWorkflowFixture(t, base, container, "alpine:latest"), findingCode: "public.workflow_container"},
		{name: "privileged shell container", content: replaceWorkflowFixture(t, base, "docker run --rm", "docker run --rm --privileged=true"), findingCode: "public.workflow_container"},
		{name: "host network shell container", content: replaceWorkflowFixture(t, base, "--network none", "--network host"), findingCode: "public.workflow_container"},
		{name: "writable root mount", content: replaceWorkflowFixture(t, base, "${RUNNER_TEMP}/gitleaks-canary:/scan:ro", "/:/scan"), findingCode: "public.workflow_container"},
		{name: "anchor declaration", content: replaceWorkflowFixture(t, base, "name: Foundation quality", "name: &policy Foundation quality"), findingCode: "public.workflow_yaml"},
		{name: "alias value", content: replaceWorkflowFixture(t, base, "    timeout-minutes: 20", "    timeout-minutes: *policy"), findingCode: "public.workflow_yaml"},
		{name: "explicit key", content: replaceWorkflowFixture(t, base, "  workflow_dispatch:\n", "  ? workflow_dispatch\n  :\n"), findingCode: "public.workflow_yaml"},
		{name: "YAML tag", content: replaceWorkflowFixture(t, base, "name: Foundation quality", "name: !str Foundation quality"), findingCode: "public.workflow_yaml"},
		{name: "YAML directive", content: "%YAML 1.2\n" + base, findingCode: "public.workflow_yaml"},
		{name: "second document", content: base + "\n---\nname: second\n", findingCode: "public.workflow_yaml"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			var findings []Finding
			checkWorkflow(".github/workflows/foundation-quality.yml", testCase.content, &findings)
			if !findingCodePresent(findings, testCase.findingCode) {
				t.Fatalf("workflow bypass was accepted; wanted %s, got %v", testCase.findingCode, findings)
			}
		})
	}
}

func canonicalFoundationWorkflow(t *testing.T) string {
	t.Helper()
	path := filepath.Join("..", "..", ".github", "workflows", "foundation-quality.yml")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func replaceWorkflowFixture(t *testing.T, content, oldValue, newValue string) string {
	t.Helper()
	if !strings.Contains(content, oldValue) {
		t.Fatalf("workflow fixture does not contain %q", oldValue)
	}
	return strings.Replace(content, oldValue, newValue, 1)
}

func findingCodePresent(findings []Finding, code string) bool {
	for _, finding := range findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}

func TestDeveloperPathDetectionDistinguishesURLs(t *testing.T) {
	if containsDeveloperPath("https://substack.com/home/post/public-article") {
		t.Fatal("public URL path was classified as a developer path")
	}
	local := "workspace=" + "/" + "Users" + "/" + "developer" + "/project"
	if !containsDeveloperPath(local) {
		t.Fatal("developer home path was not detected")
	}
}

func TestRefreshDoctrineWritesOnlyGeneratedManifest(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses a POSIX fake git executable")
	}
	repo := t.TempDir()
	source := t.TempDir()
	manifestDirectory := filepath.Join(repo, ".harness", "generated", "mars")
	if err := os.MkdirAll(manifestDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	sentinelPath := filepath.Join(repo, "docs", "project-owned.md")
	sentinel := []byte("project owned\n")
	if err := os.WriteFile(sentinelPath, sentinel, 0o644); err != nil {
		t.Fatal(err)
	}
	sourceContent := []byte("pinned doctrine\n")
	if err := os.WriteFile(filepath.Join(source, "AGENTS.md"), sourceContent, 0o644); err != nil {
		t.Fatal(err)
	}

	var manifest marsSourceManifest
	manifest.SchemaVersion = 1
	manifest.Generated.Source = "local-checkout-only"
	manifest.Generated.Command = "mars3 doctrine refresh"
	manifest.Upstream.Repository = marsRepository
	manifest.Upstream.Commit = marsCommit
	manifest.Upstream.License = "Apache-2.0"
	manifest.SourceFiles = append(manifest.SourceFiles, struct {
		Path       string `json:"path"`
		GitBlob    string `json:"gitBlob"`
		Adaptation string `json:"adaptation"`
	}{Path: "AGENTS.md", GitBlob: gitBlobID(sourceContent), Adaptation: "bounded-work doctrine"})
	manifest.GeneratedScope = []string{".harness/generated/mars/source-manifest.json"}
	manifest.ProjectOwnedPaths = []string{"docs/"}
	manifest.Adaptations = []string{"project-owned adaptation"}
	manifest.SourceOnlyExclusions = []string{"upstream runtime"}
	manifestData, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(manifestDirectory, "source-manifest.json")
	if err := os.WriteFile(manifestPath, manifestData, 0o644); err != nil {
		t.Fatal(err)
	}

	fakeBin := t.TempDir()
	gitScript := "#!/bin/sh\nprintf '%s\\n' '" + marsCommit + "'\n"
	if err := os.WriteFile(filepath.Join(fakeBin, "git"), []byte(gitScript), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))

	beforeDryRun, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	required := map[string]string{"AGENTS.md": gitBlobID(sourceContent)}
	result, err := refreshDoctrine(repo, source, marsCommit, false, required)
	if err != nil {
		t.Fatalf("dry-run refresh: %v", err)
	}
	if result.Applied || result.Target != ".harness/generated/mars/source-manifest.json" {
		t.Fatalf("unexpected dry-run result: %+v", result)
	}
	afterDryRun, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytesEqual(beforeDryRun, afterDryRun) {
		t.Fatal("dry-run changed the generated manifest")
	}

	result, err = refreshDoctrine(repo, source, marsCommit, true, required)
	if err != nil {
		t.Fatalf("apply refresh: %v", err)
	}
	if !result.Applied || result.SourceFiles != 1 {
		t.Fatalf("unexpected apply result: %+v", result)
	}
	afterSentinel, err := os.ReadFile(sentinelPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytesEqual(sentinel, afterSentinel) {
		t.Fatal("refresh changed a project-owned file")
	}
	afterManifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if bytesEqual(beforeDryRun, afterManifest) || !strings.HasSuffix(string(afterManifest), "\n") {
		t.Fatal("apply did not canonicalize the generated manifest")
	}
}

func TestRefreshDoctrineRejectsIncompleteRequiredProvenance(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses a POSIX fake git executable")
	}
	repo := t.TempDir()
	source := t.TempDir()
	manifestDirectory := filepath.Join(repo, ".harness", "generated", "mars")
	if err := os.MkdirAll(manifestDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	content := []byte("pinned doctrine\n")
	if err := os.WriteFile(filepath.Join(source, "AGENTS.md"), content, 0o644); err != nil {
		t.Fatal(err)
	}
	var manifest marsSourceManifest
	manifest.SchemaVersion = 1
	manifest.Upstream.Repository = marsRepository
	manifest.Upstream.Commit = marsCommit
	manifest.Upstream.License = "Apache-2.0"
	manifest.GeneratedScope = []string{".harness/generated/mars/source-manifest.json"}
	manifest.SourceFiles = append(manifest.SourceFiles, struct {
		Path       string `json:"path"`
		GitBlob    string `json:"gitBlob"`
		Adaptation string `json:"adaptation"`
	}{Path: "AGENTS.md", GitBlob: gitBlobID(content), Adaptation: "test"})
	manifestData, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(manifestDirectory, "source-manifest.json"), manifestData, 0o644); err != nil {
		t.Fatal(err)
	}
	fakeBin := t.TempDir()
	gitScript := "#!/bin/sh\nprintf '%s\\n' '" + marsCommit + "'\n"
	if err := os.WriteFile(filepath.Join(fakeBin, "git"), []byte(gitScript), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))
	_, err = refreshDoctrine(repo, source, marsCommit, false, map[string]string{
		"AGENTS.md": contentHashForTest(content),
		"LICENSE":   strings.Repeat("0", 40),
	})
	if err == nil || !strings.Contains(err.Error(), "required MARS source") {
		t.Fatalf("incomplete provenance was accepted: %v", err)
	}
}

func TestDocSyncRequiresUnionOfMatchingAncestorRules(t *testing.T) {
	root := t.TempDir()
	writeDocSyncFixtureFile(t, root, ".harness/docsync.yaml", `schema_version: 1
marker: FactoryDocSync
source_extensions:
  - .sql
prefix_requirements:
  - prefix: database/
    docs:
      - docs/product-specs/foundation.md
      - docs/design-docs/ADR-003-rule-of-two.md
  - prefix: database/authority/leases/
    docs:
      - docs/code-documentation-map.md
`)
	for _, path := range []string{
		"docs/product-specs/foundation.md",
		"docs/design-docs/ADR-003-rule-of-two.md",
		"docs/code-documentation-map.md",
	} {
		writeDocSyncFixtureFile(t, root, path, "# Public fixture\n")
	}

	sourcePath := "database/authority/leases/schema.sql"
	writeDocSyncFixtureFile(t, root, sourcePath, `-- FactoryDocSync:
-- docs:
-- - docs/code-documentation-map.md
select 1;
`)
	findings, err := AuditDocSync(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"docs/product-specs/foundation.md",
		"docs/design-docs/ADR-003-rule-of-two.md",
	} {
		if !docSyncFindingMentions(findings, sourcePath, "docsync.prefix_requirement", required) {
			t.Fatalf("descendant rule dropped ancestor requirement %s: %v", required, findings)
		}
	}

	writeDocSyncFixtureFile(t, root, sourcePath, `-- FactoryDocSync:
-- docs:
-- - docs/product-specs/foundation.md
-- - docs/design-docs/ADR-003-rule-of-two.md
-- - docs/code-documentation-map.md
select 1;
`)
	findings, err = AuditDocSync(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Fatalf("complete ancestor union was rejected: %v", findings)
	}
}

func writeDocSyncFixtureFile(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func docSyncFindingMentions(findings []Finding, path, code, fragment string) bool {
	for _, finding := range findings {
		if finding.Path == path && finding.Code == code && strings.Contains(finding.Message, fragment) {
			return true
		}
	}
	return false
}

func contentHashForTest(content []byte) string {
	return gitBlobID(content)
}
