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

func TestPublicCheckRejectsBinaryDuringFoundation(t *testing.T) {
	var findings []Finding
	checkPublicContent(t.TempDir(), "fixture.png", []byte{0x00, 0x01, 0x02}, &findings)
	if !findingCodePresent(findings, "public.h001_binary") {
		t.Fatalf("binary content was accepted: %v", findings)
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

func contentHashForTest(content []byte) string {
	return gitBlobID(content)
}
