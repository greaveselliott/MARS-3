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
)

const wave1PlanningGrantFirstCommitFixture = "fc9f6641d0f739a401a4f7be3bc0ee575df1310a"

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
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, wave1V3AddendumBase)
	runPlanningGrantTestGit(t, root, "checkout", "--quiet", "--detach", "FETCH_HEAD")
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, "refs/tags/"+wave1PriorPublicationTag+":refs/tags/"+wave1PriorPublicationTag)
	runPlanningGrantTestGit(t, root, "fetch", "--quiet", "--no-tags", source, "refs/tags/"+wave1V2PublicationTag+":refs/tags/"+wave1V2PublicationTag)
	runPlanningGrantTestGit(t, root, "checkout", "--quiet", "-b", wave1PlanningGrantBranch)

	writePlanningGrantCurrentFiles(t, root)
	return root
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
	plan, err := os.ReadFile(filepath.Join(source, filepath.FromSlash(canonicalActivePlan)))
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
