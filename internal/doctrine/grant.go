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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	wave1PlanningGrantPath          = ".harness/grants/WAVE-1-contract-publication.yaml"
	wave1PlanningGrantSignature     = ".harness/grants/WAVE-1-contract-publication.yaml.sig"
	wave1PlanningGrantKey           = ".harness/keys/genesis-signing-key.pub"
	wave1PlanningGrantNamespace     = "mars3-planning-grant"
	wave1PlanningGrantBase          = "ee385ce236ae1f99da692d223d7666b80dd9108f"
	wave1PlanningGrantBranch        = "codex/w-001-work-authority"
	planningGrantCommitNS           = "git"
	planningGrantRepository         = "greaveselliott/MARS-3"
	planningGrantWorkflow           = "Foundation quality"
	planningGrantWorkflowPath       = ".github/workflows/foundation-quality.yml"
	planningGrantWorkflowJob        = "public-commit-gate"
	wave1DispositionPath            = ".harness/grants/WAVE-1-recovery-disposition.yaml"
	wave1DispositionSignature       = ".harness/grants/WAVE-1-recovery-disposition.yaml.sig"
	wave1DispositionSnapshot        = ".harness/grants/WAVE-1-authority-recovery-state.json"
	wave1DispositionNamespace       = "mars3-recovery-disposition"
	wave1DispositionSnapshotSHA     = "4d3b5c9d90a223c0e9d974e836559309a2f4dac7f209a3966336e9152f57feca"
	wave1PriorPublicationTag        = "mars3/wave1-contract-publication-v1"
	wave1PriorPublicationTagMessage = "MARS-3 Wave-1 contract-publication tree attestation v1"
	wave1PriorPublicationTagObject  = "4bce7e7d4a8b2cc1a5b30b9feaee61232c3cc0de"
	wave1AddendumPath               = ".harness/grants/WAVE-1-ci-recovery-addendum.yaml"
	wave1AddendumSignature          = ".harness/grants/WAVE-1-ci-recovery-addendum.yaml.sig"
	wave1AddendumNamespace          = "mars3-ci-recovery-addendum"
	wave1AddendumBase               = "a22cfe6fada6f2bc787742eae50bca28cec80c89"
	wave1AddendumBaseTree           = "3c5befaefab37a8d0a2e3a8af2efd6e1eb1d8cae"
	wave1V2PublicationTag           = "mars3/wave1-contract-publication-v2"
	wave1V2PublicationTagMessage    = "MARS-3 Wave-1 contract-publication tree attestation v2"
	wave1V2PublicationTagObject     = "e334356519188fc0906549515ae57fbffa646829"
	wave1V3AddendumPath             = ".harness/grants/WAVE-1-ci-recovery-addendum-v3.yaml"
	wave1V3AddendumSignature        = ".harness/grants/WAVE-1-ci-recovery-addendum-v3.yaml.sig"
	wave1V3AddendumNamespace        = "mars3-ci-recovery-addendum-v3"
	wave1V3AddendumBase             = "412a9b857265af250ee40d36d0a6c127714e4ec9"
	wave1V3AddendumBaseTree         = "8c7f3ccac3e31d0e8b45431934cd95a91e448c0f"
	wave1V3ObservedStaleMerge       = "fff2bea9bffa9400d3ecfc147b7338849ecfbbb0"
	wave1PublicationTag             = "mars3/wave1-contract-publication-v3"
	wave1PublicationTagMessage      = "MARS-3 Wave-1 contract-publication tree attestation v3"
)

type grantScalarExpectation struct {
	path  string
	value string
}

var wave1PlanningGrantScalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3ContractPublicationGrant"},
	{path: "grant.id", value: "WAVE-1-contract-publication"},
	{path: "grant.classification", value: "PUBLIC"},
	{path: "grant.issuedAt", value: "2026-08-26T05:30:55Z"},
	{path: "grant.baseCommit", value: wave1PlanningGrantBase},
	{path: "grant.workingBranch", value: wave1PlanningGrantBranch},
	{path: "grant.signerRole", value: "human-bootstrap-authority"},
	{path: "grant.coordinator", value: "delivery-orchestrator"},
	{path: "grant.purpose", value: "publish reviewed Wave-1 Git contracts and admission validators before any implementation claim"},
	{path: "grant.autonomousMutation", value: "false"},
	{path: "grant.implementationClaimed", value: "false"},
	{path: "grant.expiresOn", value: "first verified merge of this grant and its bounded contract-publication change to main"},
	{path: "grant.successorAuthority", value: "signed W-001 implementation bootstrap grant or the accepted authority gateway"},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.rawAuthorityPayloadIncluded", value: "false"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: wave1PlanningGrantNamespace},
	{path: "integrity.detachedSignature", value: "WAVE-1-contract-publication.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var wave1PlanningGrantSequences = map[string][]string{
	"grant.targetBeads": {
		"M3-W001",
		"M3-P001",
	},
	"grant.allowedEffects": {
		"edit-listed-public-paths",
		"run-public-commit-gate",
		"create-signed-git-commits",
		"push-working-branch",
		"open-or-update-pull-request",
		"append-public-safe-contract-status-intent-and-receipt",
	},
	"grant.authorizedPaths": {
		"AGENTS.md",
		"README.md",
		".harness/docsync.yaml",
		".harness/manifest.yaml",
		".harness/grants/WAVE-1-contract-publication.yaml",
		".harness/grants/WAVE-1-contract-publication.yaml.sig",
		"docs/QUALITY_SCORE.md",
		"docs/code-documentation-map.md",
		"docs/design-docs/ADR-001-git-beads-authority.md",
		"docs/design-docs/ADR-006-local-substrate.md",
		"docs/design-docs/index.md",
		"docs/evidence/H-001-review-disposition.md",
		"docs/evidence/WAVE-1-contract-publication.md",
		"docs/exec-plans/active/current-operating-plan.md",
		"docs/features/F-001-doctrine-foundation.md",
		"docs/features/F-002-work-authority.md",
		"docs/features/F-003-local-substrate.md",
		"docs/features/README.md",
		"docs/goals/active.md",
		"docs/product-decisions/PD-001-public-first.md",
		"docs/product-decisions/PD-002-git-beads-authority.md",
		"docs/product-decisions/PD-003-provider-neutral.md",
		"docs/product-specs/foundation.md",
		"docs/product-specs/index.md",
		"docs/product-specs/local-substrate.md",
		"docs/product-specs/work-authority.md",
		"internal/doctrine/doctrine.go",
		"internal/doctrine/doctrine_test.go",
		"internal/doctrine/grant.go",
		"internal/doctrine/grant_test.go",
		"internal/doctrine/plan.go",
		"internal/doctrine/plan_test.go",
		"internal/doctrine/public.go",
		"internal/doctrine/public_test.go",
	},
	"grant.prohibitedEffects": {
		"claim-or-transition-bead",
		"issue-or-assert-live-lease",
		"runtime-or-platform-implementation",
		"production-or-destructive-effect",
		"trust-escalation",
		"provider-or-customer-data-access",
		"credential-persistence",
	},
	"verification.order": {
		"qa",
		"security-reviewer",
		"delivery-orchestrator",
	},
}

var wave1DispositionScalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3RecoveryDispositionGrant"},
	{path: "disposition.id", value: "WAVE-1-recovery-disposition"},
	{path: "disposition.classification", value: "PUBLIC"},
	{path: "disposition.issuedAt", value: "2026-08-26T06:44:37Z"},
	{path: "disposition.expiresAt", value: "2026-08-29T06:44:37Z"},
	{path: "disposition.baseCommit", value: "fc9f6641d0f739a401a4f7be3bc0ee575df1310a"},
	{path: "disposition.workingBranch", value: wave1PlanningGrantBranch},
	{path: "disposition.signerRole", value: "human-bootstrap-authority"},
	{path: "disposition.coordinator", value: "delivery-orchestrator"},
	{path: "disposition.failureOwnership", value: "foundation"},
	{path: "disposition.purpose", value: "prospectively constrain Wave-1 recovery disposition, final description repair, and squash-safe signed-tree publication"},
	{path: "disposition.retroactiveAuthorization", value: "false"},
	{path: "disposition.priorExternalEffectsAccepted", value: "false"},
	{path: "disposition.autonomousMutation", value: "false"},
	{path: "disposition.implementationClaimed", value: "false"},
	{path: "disposition.liveLeaseAsserted", value: "false"},
	{path: "disposition.originalArtifactSHA256", value: "46a7e8459204b29e739a27859539deb44f6a54d79e4ac64ad6c3cc358d9b5d04"},
	{path: "disposition.originalDetachedSignatureSHA256", value: "4ab35549b3cb64bd58f59006aeadcbfdca0754850662481e44c0f9c8e51f5356"},
	{path: "disposition.originalArtifactGitDisposition", value: "not-committed-because-the-pinned-scanner-flags-two-public-checksum-fields"},
	{path: "disposition.stateSnapshot", value: "WAVE-1-authority-recovery-state.json"},
	{path: "disposition.stateSnapshotSHA256", value: wave1DispositionSnapshotSHA},
	{path: "disposition.stateSnapshotSchema", value: "mars3-authority-observation/v2"},
	{path: "disposition.preAuthorityRevision", value: "k46ees2p"},
	{path: "disposition.legacyStateDigestsReconstructible", value: "false"},
	{path: "disposition.p001CorrectionIdempotencyKey", value: "wave1-p001-shared-path-description-correction-v1"},
	{path: "disposition.p001PreDescriptionSHA256", value: "828e32051ee661e6d31c5017e996e3d8ec82257ca0a17e263f974248c5772314"},
	{path: "disposition.p001PostDescriptionSHA256", value: "d24074a56a4df0d150450555b981f84fa377364f40c9c9191e49f4ecae73c20c"},
	{path: "disposition.publicationMode", value: "squash-with-signed-tree-tag"},
	{path: "disposition.publicationTag", value: wave1PriorPublicationTag},
	{path: "disposition.publicationTagMessage", value: wave1PriorPublicationTagMessage},
	{path: "disposition.rulesetId", value: "21510926"},
	{path: "disposition.rulesetObservedUpdatedAt", value: "2026-08-26T03:00:36.071+01:00"},
	{path: "disposition.rulesetMutationAllowed", value: "false"},
	{path: "disposition.requiredStatusCheck", value: "Public commit gate"},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.recoveryDispositionRequired", value: "true"},
	{path: "verification.externalStateReadbackRequired", value: "true"},
	{path: "verification.rawAuthorityPayloadIncluded", value: "false"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: wave1DispositionNamespace},
	{path: "integrity.detachedSignature", value: "WAVE-1-recovery-disposition.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var wave1DispositionSequences = map[string][]string{
	"disposition.targetBeads":  {"M3-W001", "M3-P001"},
	"disposition.mutableBeads": {"M3-P001"},
	"disposition.allowedEffects": {
		"verify-and-independently-disposition-observed-final-authority-state",
		"replace-m3-p001-description-with-exact-postimage",
		"append-public-safe-disposition-intent-and-receipt",
		"edit-listed-public-paths",
		"run-public-commit-gate",
		"create-pinned-signer-git-commits",
		"create-and-push-one-pinned-signer-publication-tag",
		"push-working-branch",
		"open-or-update-pull-request",
	},
	"disposition.authorizedPaths": {
		".harness/docsync.yaml",
		".harness/manifest.yaml",
		wave1DispositionSnapshot,
		wave1DispositionPath,
		wave1DispositionSignature,
		"AGENTS.md",
		"README.md",
		"docs/QUALITY_SCORE.md",
		"docs/code-documentation-map.md",
		"docs/design-docs/ADR-001-git-beads-authority.md",
		"docs/design-docs/ADR-006-local-substrate.md",
		"docs/design-docs/index.md",
		"docs/evidence/H-001-review-disposition.md",
		"docs/evidence/WAVE-1-contract-publication.md",
		"docs/exec-plans/active/current-operating-plan.md",
		"docs/features/F-001-doctrine-foundation.md",
		"docs/features/F-002-work-authority.md",
		"docs/features/F-003-local-substrate.md",
		"docs/features/README.md",
		"docs/goals/active.md",
		"docs/product-decisions/PD-001-public-first.md",
		"docs/product-decisions/PD-002-git-beads-authority.md",
		"docs/product-decisions/PD-003-provider-neutral.md",
		"docs/product-specs/foundation.md",
		"docs/product-specs/index.md",
		"docs/product-specs/local-substrate.md",
		"docs/product-specs/work-authority.md",
		"internal/doctrine/docsync.go",
		"internal/doctrine/doctrine.go",
		"internal/doctrine/doctrine_test.go",
		"internal/doctrine/grant.go",
		"internal/doctrine/grant_test.go",
		"internal/doctrine/plan.go",
		"internal/doctrine/plan_test.go",
		"internal/doctrine/public.go",
	},
	"disposition.requiredP001Postconditions": {
		"native-status-open",
		"lifecycle-backlog",
		"claim-absent",
		"lease-absent",
		"dependencies-exactly-M3-H001-and-M3-W001",
		"exclusive-paths-unchanged",
		"all-other-Beads-unchanged",
	},
	"disposition.requiredPublicationProperties": {
		"every-feature-commit-has-the-pinned-ssh-signature",
		"signed-tag-target-descends-from-the-exact-grant-base",
		"signed-tag-target-tree-equals-reviewed-pr-head-tree",
		"squash-main-tree-equals-signed-tag-target-tree",
		"tag-keeps-reviewed-feature-history-reachable",
		"required-public-commit-gate-passes-on-pr-and-main",
	},
	"disposition.prohibitedEffects": {
		"authorize-prior-effects-retroactively",
		"mutate-m3-w001",
		"mutate-p001-metadata-dependencies-lifecycle-claim-or-lease",
		"mutate-any-other-bead",
		"any-description-write-other-than-the-exact-p001-postimage",
		"issue-or-assert-live-lease",
		"runtime-or-platform-implementation",
		"production-or-destructive-effect",
		"autonomous-mutation",
		"trust-escalation",
		"provider-or-customer-data-access",
		"credential-persistence",
		"add-secret-scanner-ignore-or-allowlist",
		"mutate-github-rules-or-repository-settings",
		"create-or-move-any-other-tag",
	},
	"verification.order": {"qa", "security-reviewer", "delivery-orchestrator"},
}

var wave1AddendumScalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3CIRecoveryAddendum"},
	{path: "addendum.id", value: "WAVE-1-ci-recovery-addendum"},
	{path: "addendum.classification", value: "PUBLIC"},
	{path: "addendum.issuedAt", value: "2026-08-26T07:25:36Z"},
	{path: "addendum.expiresAt", value: "2026-08-29T07:25:36Z"},
	{path: "addendum.baseCommit", value: wave1AddendumBase},
	{path: "addendum.workingBranch", value: wave1PlanningGrantBranch},
	{path: "addendum.signerRole", value: "human-bootstrap-authority"},
	{path: "addendum.coordinator", value: "delivery-orchestrator"},
	{path: "addendum.failureOwnership", value: "foundation"},
	{path: "addendum.purpose", value: "prospectively repair nullable GitHub pull-request merge identity admission without weakening publication controls"},
	{path: "addendum.retroactiveAuthorization", value: "false"},
	{path: "addendum.priorHead", value: wave1AddendumBase},
	{path: "addendum.priorTree", value: wave1AddendumBaseTree},
	{path: "addendum.priorTag", value: wave1PriorPublicationTag},
	{path: "addendum.priorTagObject", value: wave1PriorPublicationTagObject},
	{path: "addendum.failedRunId", value: "32941818590"},
	{path: "addendum.failedRunAttempts", value: "2"},
	{path: "addendum.finalFailedJobId", value: "98095787715"},
	{path: "addendum.failureFingerprint", value: "public.planning_grant_branch-github-pr-null-merge-commit-sha"},
	{path: "addendum.rootCause", value: "pull-request-event-merge-commit-sha-was-null"},
	{path: "addendum.correctionInvariant", value: "nullable-event-field-requires-github-sha-event-base-head-and-exact-two-parent-git-topology"},
	{path: "addendum.successorTag", value: wave1V2PublicationTag},
	{path: "addendum.successorTagMessage", value: wave1V2PublicationTagMessage},
	{path: "addendum.v1TagImmutable", value: "true"},
	{path: "addendum.autonomousMutation", value: "false"},
	{path: "addendum.implementationClaimed", value: "false"},
	{path: "addendum.liveLeaseAsserted", value: "false"},
	{path: "addendum.canonicalWorkMutationAllowed", value: "false"},
	{path: "addendum.githubPolicyMutationAllowed", value: "false"},
	{path: "addendum.secretScannerExceptionAllowed", value: "false"},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.failedAttemptsPreserved", value: "true"},
	{path: "verification.externalStateReadbackRequired", value: "true"},
	{path: "verification.rawRunnerPayloadIncluded", value: "false"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: wave1AddendumNamespace},
	{path: "integrity.detachedSignature", value: "WAVE-1-ci-recovery-addendum.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var wave1AddendumSequences = map[string][]string{
	"addendum.allowedEffects": {
		"edit-listed-public-paths",
		"run-public-commit-gate",
		"create-pinned-signer-git-commits",
		"create-and-push-one-pinned-signer-retry-tag",
		"push-working-branch",
		"update-existing-pull-request",
		"append-public-safe-recovery-intent-and-receipt",
	},
	"addendum.authorizedPaths": {
		wave1AddendumPath,
		wave1AddendumSignature,
		"docs/evidence/WAVE-1-contract-publication.md",
		"internal/doctrine/grant.go",
		"internal/doctrine/grant_test.go",
	},
	"addendum.requiredCorrectionProperties": {
		"accept-null-event-merge-sha-only-for-verified-pull-request-merge",
		"reject-nonempty-mismatched-event-merge-sha",
		"require-github-sha-equals-checked-out-merge",
		"require-event-base-and-head-equal-git-merge-parents",
		"retain-hosted-runner-workflow-job-repository-and-workspace-facts",
		"retain-pinned-workflow-digest-and-signed-feature-commit-validation",
		"enforce-addendum-only-paths-after-the-v1-target",
		"preserve-v1-tag-and-record-both-failed-attempts",
		"require-v2-tag-tree-equal-updated-review-and-squash-main-trees",
	},
	"addendum.prohibitedEffects": {
		"authorize-prior-effects-retroactively",
		"move-or-delete-v1-tag",
		"create-or-move-any-tag-other-than-v2",
		"accept-nonempty-mismatched-event-merge-sha",
		"use-nullable-merge-sha-rule-outside-pull-request-ci",
		"mutate-workflow-ruleset-or-repository-settings",
		"add-secret-scanner-ignore-or-allowlist",
		"mutate-any-bead-work-field-or-dependency",
		"claim-or-transition-bead",
		"issue-or-assert-live-lease",
		"runtime-or-platform-implementation",
		"production-or-destructive-effect",
		"autonomous-mutation",
		"trust-escalation",
	},
	"verification.order": {"qa", "security-reviewer", "delivery-orchestrator"},
}

var wave1V3AddendumScalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3CIRecoveryAddendumV3"},
	{path: "addendum.id", value: "WAVE-1-ci-recovery-addendum-v3"},
	{path: "addendum.classification", value: "PUBLIC"},
	{path: "addendum.issuedAt", value: "2026-08-26T07:56:09Z"},
	{path: "addendum.expiresAt", value: "2026-08-29T07:56:09Z"},
	{path: "addendum.baseCommit", value: wave1V3AddendumBase},
	{path: "addendum.baseTree", value: wave1V3AddendumBaseTree},
	{path: "addendum.workingBranch", value: wave1PlanningGrantBranch},
	{path: "addendum.signerRole", value: "human-bootstrap-authority"},
	{path: "addendum.coordinator", value: "delivery-orchestrator"},
	{path: "addendum.failureOwnership", value: "foundation"},
	{path: "addendum.purpose", value: "prospectively admit advisory stale GitHub pull-request merge metadata without weakening canonical checkout and topology controls"},
	{path: "addendum.retroactiveAuthorization", value: "false"},
	{path: "addendum.priorHead", value: wave1V3AddendumBase},
	{path: "addendum.priorTree", value: wave1V3AddendumBaseTree},
	{path: "addendum.v1Tag", value: wave1PriorPublicationTag},
	{path: "addendum.v1TagObject", value: wave1PriorPublicationTagObject},
	{path: "addendum.v1TagTarget", value: wave1AddendumBase},
	{path: "addendum.v2Tag", value: wave1V2PublicationTag},
	{path: "addendum.v2TagObject", value: wave1V2PublicationTagObject},
	{path: "addendum.v2TagTarget", value: wave1V3AddendumBase},
	{path: "addendum.failedRunId", value: "32943782330"},
	{path: "addendum.failedRunAttempts", value: "2"},
	{path: "addendum.finalFailedJobId", value: "98101129557"},
	{path: "addendum.observedCheckout", value: "3ffd69c107f1492883b65a67678f7239602299a4"},
	{path: "addendum.observedBase", value: wave1PlanningGrantBase},
	{path: "addendum.observedHead", value: wave1V3AddendumBase},
	{path: "addendum.observedStalePayloadMerge", value: wave1V3ObservedStaleMerge},
	{path: "addendum.failureFingerprint", value: "public.planning_grant_branch-github-pr-stale-nonempty-merge-commit-sha"},
	{path: "addendum.rootCause", value: "pull-request-event-payload-retained-the-prior-test-merge-identity-after-the-head-advanced"},
	{path: "addendum.correctionInvariant", value: "payload-merge-identity-is-advisory-while-github-sha-event-base-head-signed-tree-and-exact-two-parent-topology-remain-authoritative"},
	{path: "addendum.successorTag", value: wave1PublicationTag},
	{path: "addendum.successorTagMessage", value: wave1PublicationTagMessage},
	{path: "addendum.v1TagImmutable", value: "true"},
	{path: "addendum.v2TagImmutable", value: "true"},
	{path: "addendum.autonomousMutation", value: "false"},
	{path: "addendum.implementationClaimed", value: "false"},
	{path: "addendum.liveLeaseAsserted", value: "false"},
	{path: "addendum.canonicalWorkMutationAllowed", value: "false"},
	{path: "addendum.githubPolicyMutationAllowed", value: "false"},
	{path: "addendum.secretScannerExceptionAllowed", value: "false"},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.failedAttemptsPreserved", value: "true"},
	{path: "verification.externalStateReadbackRequired", value: "true"},
	{path: "verification.rawRunnerPayloadIncluded", value: "false"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: wave1V3AddendumNamespace},
	{path: "integrity.detachedSignature", value: "WAVE-1-ci-recovery-addendum-v3.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var wave1V3AddendumSequences = map[string][]string{
	"addendum.allowedEffects": {
		"edit-listed-public-paths",
		"run-public-commit-gate",
		"create-pinned-signer-git-commits",
		"create-and-push-one-pinned-signer-v3-tag",
		"push-working-branch",
		"update-existing-pull-request",
		"append-public-safe-recovery-intent-and-receipt",
	},
	"addendum.authorizedPaths": {
		wave1V3AddendumPath,
		wave1V3AddendumSignature,
		"docs/evidence/WAVE-1-contract-publication.md",
		"internal/doctrine/grant.go",
		"internal/doctrine/grant_test.go",
	},
	"addendum.requiredCorrectionProperties": {
		"treat-optional-payload-merge-sha-as-advisory-only-in-pull-request-ci",
		"accept-absent-null-current-or-stale-lowercase-forty-hex-payload-merge-sha",
		"reject-malformed-nonempty-payload-merge-identity",
		"require-github-sha-equals-checked-out-merge",
		"require-event-base-and-head-equal-git-merge-parents-in-order",
		"retain-hosted-runner-workflow-job-repository-workspace-and-workflow-digest-facts",
		"retain-signed-feature-history-phase-specific-path-and-publication-tree-validation",
		"enforce-v3-addendum-only-paths-after-the-v2-target",
		"preserve-v1-and-v2-tags-and-record-all-failed-attempts",
		"require-v3-tag-tree-equal-updated-review-and-squash-main-trees",
	},
	"addendum.prohibitedEffects": {
		"authorize-prior-effects-retroactively",
		"move-or-delete-v1-or-v2-tag",
		"create-or-move-any-tag-other-than-v3",
		"use-advisory-merge-identity-rule-outside-pull-request-ci",
		"accept-malformed-payload-merge-identity",
		"weaken-github-sha-event-base-head-or-exact-two-parent-topology-validation",
		"mutate-workflow-ruleset-or-repository-settings",
		"add-secret-scanner-ignore-or-allowlist",
		"mutate-any-bead-work-field-or-dependency",
		"claim-or-transition-bead",
		"issue-or-assert-live-lease",
		"runtime-or-platform-implementation",
		"production-or-destructive-effect",
		"autonomous-mutation",
		"trust-escalation",
	},
	"verification.order": {"qa", "security-reviewer", "delivery-orchestrator"},
}

type strictPlanningGrant struct {
	scalars          map[string][]string
	sequences        map[string][]string
	sections         map[string]int
	sequenceHeaders  map[string]int
	structuralErrors []string
}

// CheckPlanningGrant validates the signed, public, contract-publication grant
// without consulting an operational authority or executing a signer process.
func CheckPlanningGrant(repo string) ([]Finding, error) {
	root, err := repositoryRoot(repo)
	if err != nil {
		return nil, err
	}
	var findings []Finding
	checkWave1PlanningGrant(root, &findings)
	sortFindings(findings)
	return findings, nil
}

func checkWave1PlanningGrant(root string, findings *[]Finding) {
	data, err := readRepoFile(root, wave1PlanningGrantPath)
	if err != nil {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_missing", "signed Wave-1 contract-publication grant is required")
		return
	}
	defer checkWave1PlanningGrantGitDiff(root, findings)

	document := parseStrictPlanningGrant(data)
	for _, message := range document.structuralErrors {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_schema", "%s", message)
	}
	for _, expected := range wave1PlanningGrantScalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_value", "%s does not match the signed Wave-1 contract", expected.path)
		}
	}
	for path, expected := range wave1PlanningGrantSequences {
		if document.sequenceHeaders[path] != 1 {
			addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_sequence", "%s must occur exactly once", path)
			continue
		}
		actual := document.sequences[path]
		if !equalStringSequence(actual, expected) {
			addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_sequence", "%s must equal the exact ordered Wave-1 contract", path)
		}
	}
	for _, section := range []string{"grant", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_schema", "%s mapping must occur exactly once", section)
		}
	}

	signature, signatureErr := readRepoFile(root, wave1PlanningGrantSignature)
	if signatureErr != nil {
		addFinding(findings, wave1PlanningGrantSignature, "public.planning_grant_signature_missing", "detached planning-grant signature is required")
	}
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	if keyErr != nil {
		addFinding(findings, wave1PlanningGrantKey, "public.planning_grant_key_missing", "pinned genesis public key is required")
		return
	}

	keyValid := true
	if fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		keyValid = false
		addFinding(findings, wave1PlanningGrantKey, "public.planning_grant_key_anchor", "planning grant must use the independently pinned genesis key")
	}
	fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey)
	if fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
		addFinding(findings, wave1PlanningGrantKey, "public.planning_grant_key_fingerprint", "planning-grant signer fingerprint does not match the genesis authority")
	}
	if signatureErr == nil && keyValid {
		if err := verifySSHSig(data, signature, publicKey, wave1PlanningGrantNamespace); err != nil {
			addFinding(findings, wave1PlanningGrantSignature, "public.planning_grant_signature", "%v", err)
		}
	}
	checkWave1RecoveryDisposition(root, findings)
}

func checkWave1RecoveryDisposition(root string, findings *[]Finding) {
	data, err := readRepoFile(root, wave1DispositionPath)
	if err != nil {
		addFinding(findings, wave1DispositionPath, "public.recovery_disposition_missing", "signed Wave-1 recovery disposition is required")
		return
	}
	document := parseStrictGrant(data, wave1DispositionScalars, wave1DispositionSequences, []string{"disposition", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, wave1DispositionPath, "public.recovery_disposition_schema", "%s", message)
	}
	for _, expected := range wave1DispositionScalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, wave1DispositionPath, "public.recovery_disposition_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, wave1DispositionPath, "public.recovery_disposition_value", "%s does not match the signed recovery contract", expected.path)
		}
	}
	for path, expected := range wave1DispositionSequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, wave1DispositionPath, "public.recovery_disposition_sequence", "%s must equal the exact ordered recovery contract", path)
		}
	}
	for _, section := range []string{"disposition", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, wave1DispositionPath, "public.recovery_disposition_schema", "%s mapping must occur exactly once", section)
		}
	}

	signature, signatureErr := readRepoFile(root, wave1DispositionSignature)
	if signatureErr != nil {
		addFinding(findings, wave1DispositionSignature, "public.recovery_disposition_signature_missing", "detached recovery-disposition signature is required")
	}
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.recovery_disposition_key", "recovery disposition must use the independently pinned genesis key")
	} else if signatureErr == nil {
		if err := verifySSHSig(data, signature, publicKey, wave1DispositionNamespace); err != nil {
			addFinding(findings, wave1DispositionSignature, "public.recovery_disposition_signature", "%v", err)
		}
	}

	snapshot, snapshotErr := readRepoFile(root, wave1DispositionSnapshot)
	if snapshotErr != nil {
		addFinding(findings, wave1DispositionSnapshot, "public.recovery_snapshot_missing", "reconstructible public recovery snapshot is required")
	} else {
		if fileSHA256(snapshot) != wave1DispositionSnapshotSHA {
			addFinding(findings, wave1DispositionSnapshot, "public.recovery_snapshot_digest", "recovery snapshot does not match the signed SHA-256 binding")
		}
		var decoded map[string]any
		if err := decodeStrictJSON(snapshot, &decoded); err != nil {
			addFinding(findings, wave1DispositionSnapshot, "public.recovery_snapshot_schema", "recovery snapshot must be one strict JSON document")
		}
	}
	for _, legacy := range []string{
		".harness/grants/WAVE-1-authority-recovery.yaml",
		".harness/grants/WAVE-1-authority-recovery.yaml.sig",
	} {
		if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(legacy))); err == nil {
			addFinding(findings, legacy, "public.recovery_legacy_artifact", "scanner-triggering legacy recovery artifact must remain outside Git; the signed disposition binds its public checksum")
		} else if !os.IsNotExist(err) {
			addFinding(findings, legacy, "public.recovery_legacy_artifact", "legacy recovery artifact state cannot be established")
		}
	}
	checkWave1CIRecoveryAddendum(root, findings)
}

func checkWave1CIRecoveryAddendum(root string, findings *[]Finding) {
	data, err := readRepoFile(root, wave1AddendumPath)
	if err != nil {
		addFinding(findings, wave1AddendumPath, "public.ci_recovery_addendum_missing", "signed Wave-1 CI recovery addendum is required after the preserved failed publication")
		return
	}
	document := parseStrictGrant(data, wave1AddendumScalars, wave1AddendumSequences, []string{"addendum", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, wave1AddendumPath, "public.ci_recovery_addendum_schema", "%s", message)
	}
	for _, expected := range wave1AddendumScalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, wave1AddendumPath, "public.ci_recovery_addendum_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, wave1AddendumPath, "public.ci_recovery_addendum_value", "%s does not match the signed CI recovery contract", expected.path)
		}
	}
	for path, expected := range wave1AddendumSequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, wave1AddendumPath, "public.ci_recovery_addendum_sequence", "%s must equal the exact ordered CI recovery contract", path)
		}
	}
	for _, section := range []string{"addendum", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, wave1AddendumPath, "public.ci_recovery_addendum_schema", "%s mapping must occur exactly once", section)
		}
	}

	signature, signatureErr := readRepoFile(root, wave1AddendumSignature)
	if signatureErr != nil {
		addFinding(findings, wave1AddendumSignature, "public.ci_recovery_addendum_signature_missing", "detached CI recovery addendum signature is required")
	}
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.ci_recovery_addendum_key", "CI recovery addendum must use the independently pinned genesis key")
	} else if signatureErr == nil {
		if err := verifySSHSig(data, signature, publicKey, wave1AddendumNamespace); err != nil {
			addFinding(findings, wave1AddendumSignature, "public.ci_recovery_addendum_signature", "%v", err)
		}
	}
	checkWave1V3CIRecoveryAddendum(root, findings)
}

func checkWave1V3CIRecoveryAddendum(root string, findings *[]Finding) {
	data, err := readRepoFile(root, wave1V3AddendumPath)
	if err != nil {
		addFinding(findings, wave1V3AddendumPath, "public.ci_recovery_v3_addendum_missing", "signed Wave-1 v3 CI recovery addendum is required after the preserved stale-merge failure")
		return
	}
	document := parseStrictGrant(data, wave1V3AddendumScalars, wave1V3AddendumSequences, []string{"addendum", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, wave1V3AddendumPath, "public.ci_recovery_v3_addendum_schema", "%s", message)
	}
	for _, expected := range wave1V3AddendumScalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, wave1V3AddendumPath, "public.ci_recovery_v3_addendum_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, wave1V3AddendumPath, "public.ci_recovery_v3_addendum_value", "%s does not match the signed v3 CI recovery contract", expected.path)
		}
	}
	for path, expected := range wave1V3AddendumSequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, wave1V3AddendumPath, "public.ci_recovery_v3_addendum_sequence", "%s must equal the exact ordered v3 CI recovery contract", path)
		}
	}
	for _, section := range []string{"addendum", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, wave1V3AddendumPath, "public.ci_recovery_v3_addendum_schema", "%s mapping must occur exactly once", section)
		}
	}

	signature, signatureErr := readRepoFile(root, wave1V3AddendumSignature)
	if signatureErr != nil {
		addFinding(findings, wave1V3AddendumSignature, "public.ci_recovery_v3_addendum_signature_missing", "detached v3 CI recovery addendum signature is required")
	}
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.ci_recovery_v3_addendum_key", "v3 CI recovery addendum must use the independently pinned genesis key")
	} else if signatureErr == nil {
		if err := verifySSHSig(data, signature, publicKey, wave1V3AddendumNamespace); err != nil {
			addFinding(findings, wave1V3AddendumSignature, "public.ci_recovery_v3_addendum_signature", "%v", err)
		}
	}
}

type planningGrantCheckoutKind int

const (
	planningGrantLocalBranch planningGrantCheckoutKind = iota + 1
	planningGrantPullRequestMerge
	planningGrantMainSquash
)

type planningGrantCheckout struct {
	kind         planningGrantCheckoutKind
	expectedHead string
	firstParent  string
	secondParent string
	tagTarget    string
}

type planningGrantCommit struct {
	id      string
	parents []string
}

type planningGrantGitHubEvent struct {
	After      string `json:"after"`
	Before     string `json:"before"`
	Ref        string `json:"ref"`
	HeadCommit *struct {
		ID string `json:"id"`
	} `json:"head_commit"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	PullRequest *struct {
		MergeCommitSHA string `json:"merge_commit_sha"`
		Base           struct {
			Ref string `json:"ref"`
			SHA string `json:"sha"`
		} `json:"base"`
		Head struct {
			Ref string `json:"ref"`
			SHA string `json:"sha"`
		} `json:"head"`
	} `json:"pull_request"`
}

func checkWave1PlanningGrantGitDiff(root string, findings *[]Finding) {
	metadata, err := os.Lstat(filepath.Join(root, ".git"))
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_diff_git", "repository Git metadata cannot be inspected")
		return
	}
	if metadata.Mode()&os.ModeSymlink != 0 || (!metadata.IsDir() && !metadata.Mode().IsRegular()) {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_diff_git", "repository Git metadata is not a regular directory or worktree pointer")
		return
	}

	plan, err := readRepoFile(root, canonicalActivePlan)
	if err != nil {
		addFinding(findings, canonicalActivePlan, "public.planning_grant_diff_phase", "active plan phase cannot be established for a Git checkout")
		return
	}
	phaseMatches := planPhaseLine.FindAllSubmatch(plan, -1)
	if len(phaseMatches) != 1 {
		addFinding(findings, canonicalActivePlan, "public.planning_grant_diff_phase", "active plan must declare exactly one phase before grant diff validation")
		return
	}
	if string(phaseMatches[0][1]) != planPhaseContractPublication {
		addFinding(findings, canonicalActivePlan, "public.planning_grant_transition_authority", "delivery requires a separately signed W-001 bootstrap grant and canonical claim evidence before contract-publication enforcement can end")
		return
	}

	topLevel, err := planningGrantGitOutput(root, "rev-parse", "--show-toplevel")
	if err != nil || !samePlanningGrantRepositoryRoot(root, strings.TrimSpace(string(topLevel))) {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_diff_git", "Git metadata must resolve to the audited repository root")
		return
	}
	resolvedBase, err := planningGrantGitOutput(root, "rev-parse", "--verify", wave1PlanningGrantBase+"^{commit}")
	if err != nil || strings.TrimSpace(string(resolvedBase)) != wave1PlanningGrantBase {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_diff_base", "the exact signed planning-grant base commit must resolve locally")
		return
	}
	if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", wave1PlanningGrantBase, "HEAD"); err != nil {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_diff_ancestry", "the exact signed planning-grant base commit must be an ancestor of HEAD")
		return
	}
	if !checkWave1PriorPublicationTag(root, findings) {
		return
	}

	headOutput, err := planningGrantGitOutput(root, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil || !sha1Pattern.MatchString(strings.TrimSpace(string(headOutput))) {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_diff_git", "HEAD must resolve to one exact commit")
		return
	}
	head := strings.TrimSpace(string(headOutput))
	checkout, ok := planningGrantCheckoutState(root, head, findings)
	if !ok {
		return
	}
	historyEnd := head
	if checkout.kind != planningGrantLocalBranch {
		tagTarget, tagOK := planningGrantPublicationTagTarget(root, checkout, head, findings)
		if !tagOK {
			return
		}
		checkout.tagTarget = tagTarget
		if checkout.kind == planningGrantMainSquash {
			historyEnd = tagTarget
		}
	}
	if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", wave1V3AddendumBase, historyEnd); err != nil {
		addFinding(findings, wave1V3AddendumPath, "public.ci_recovery_v3_addendum_ancestry", "the exact v3 CI recovery base must be an ancestor of the effective publication history")
		return
	}
	commits, err := planningGrantCommitRange(root, historyEnd)
	if err != nil {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_commit_history", "contract-publication commit ancestry cannot be enumerated")
		return
	}
	if !checkPlanningGrantCommitTopology(root, checkout, head, commits, findings) {
		return
	}

	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, err := openSSHPublicKeyFingerprint(publicKey); err != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if !keyValid {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_commit_signature", "contract-publication commits cannot be verified with the pinned genesis signer")
	}

	legacyAuthorized := make(map[string]bool, len(wave1PlanningGrantSequences["grant.authorizedPaths"])+len(wave1DispositionSequences["disposition.authorizedPaths"]))
	for _, path := range wave1PlanningGrantSequences["grant.authorizedPaths"] {
		legacyAuthorized[path] = true
	}
	for _, path := range wave1DispositionSequences["disposition.authorizedPaths"] {
		legacyAuthorized[path] = true
	}
	addendumAuthorized := make(map[string]bool, len(wave1AddendumSequences["addendum.authorizedPaths"]))
	for _, path := range wave1AddendumSequences["addendum.authorizedPaths"] {
		addendumAuthorized[path] = true
	}
	v3AddendumAuthorized := make(map[string]bool, len(wave1V3AddendumSequences["addendum.authorizedPaths"]))
	for _, path := range wave1V3AddendumSequences["addendum.authorizedPaths"] {
		v3AddendumAuthorized[path] = true
	}

	signatureFailure := false
	for _, commit := range commits {
		if len(commit.parents) == 1 && keyValid {
			object, err := planningGrantGitOutput(root, "cat-file", "commit", commit.id)
			if err != nil || verifyPlanningGrantCommit(object, publicKey) != nil {
				signatureFailure = true
			}
		}
		if len(commit.parents) != 1 {
			continue
		}
		paths, err := planningGrantGitOutput(root, "diff-tree", "--root", "-m", "--no-commit-id", "--no-renames", "--no-ext-diff", "--no-textconv", "--name-only", "-z", "-r", commit.id)
		if err != nil {
			addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_diff_paths", "per-commit contract-publication paths cannot be enumerated")
			return
		}
		normalized, err := normalizedPlanningGrantGitPaths(paths)
		if err != nil {
			addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_diff_paths", "per-commit contract-publication paths are not canonical safe repository-relative names")
			return
		}
		authorized := v3AddendumAuthorized
		if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", commit.id, wave1AddendumBase); err == nil {
			authorized = legacyAuthorized
		} else if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", commit.id, wave1V3AddendumBase); err == nil {
			authorized = addendumAuthorized
		} else if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", wave1V3AddendumBase, commit.id); err != nil {
			addFinding(findings, wave1AddendumPath, "public.ci_recovery_addendum_ancestry", "contract-publication history diverges from the exact CI recovery base")
			return
		}
		if !planningGrantPathsAllowed(normalized, authorized) {
			addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_diff_scope", "a contract-publication commit includes a path outside its signed authorization phase")
			return
		}
	}
	if signatureFailure {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_commit_signature", "every non-merge contract-publication commit must carry a valid pinned SSH signature")
	}

	tracked, err := planningGrantGitOutput(root, "diff", "--no-renames", "--no-ext-diff", "--no-textconv", "--name-only", "-z", "HEAD", "--")
	if err != nil {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_diff_paths", "current index and worktree paths cannot be enumerated")
		return
	}
	untracked, err := planningGrantGitOutput(root, "ls-files", "--others", "--exclude-standard", "-z", "--")
	if err != nil {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_diff_paths", "untracked contract-publication paths cannot be enumerated")
		return
	}
	paths, err := normalizedPlanningGrantGitPaths(tracked, untracked)
	if err != nil {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_diff_paths", "contract-publication paths are not canonical safe repository-relative names")
		return
	}
	if !planningGrantPathsAllowed(paths, v3AddendumAuthorized) {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_diff_scope", "current v3 CI recovery changes include a path outside the signed v3 addendum")
		return
	}
}

func planningGrantPathsAllowed(paths []string, authorized map[string]bool) bool {
	for _, path := range paths {
		if !authorized[path] {
			return false
		}
	}
	return true
}

func checkWave1PriorPublicationTag(root string, findings *[]Finding) bool {
	ref := "refs/tags/" + wave1PriorPublicationTag
	object, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{tag}")
	if err != nil || strings.TrimSpace(string(object)) != wave1PriorPublicationTagObject {
		addFinding(findings, wave1AddendumPath, "public.prior_publication_tag", "the preserved failed v1 publication tag must resolve to its exact immutable object")
		return false
	}
	target, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{}")
	if err != nil || strings.TrimSpace(string(target)) != wave1AddendumBase {
		addFinding(findings, wave1AddendumPath, "public.prior_publication_tag", "the preserved failed v1 publication tag target must remain unchanged")
		return false
	}
	tree, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{}^{tree}")
	if err != nil || strings.TrimSpace(string(tree)) != wave1AddendumBaseTree {
		addFinding(findings, wave1AddendumPath, "public.prior_publication_tag", "the preserved failed v1 publication tag tree must remain unchanged")
		return false
	}
	v2Ref := "refs/tags/" + wave1V2PublicationTag
	v2Object, err := planningGrantGitOutput(root, "rev-parse", "--verify", v2Ref+"^{tag}")
	if err != nil || strings.TrimSpace(string(v2Object)) != wave1V2PublicationTagObject {
		addFinding(findings, wave1V3AddendumPath, "public.prior_v2_publication_tag", "the preserved failed v2 publication tag must resolve to its exact immutable object")
		return false
	}
	v2Target, err := planningGrantGitOutput(root, "rev-parse", "--verify", v2Ref+"^{}")
	if err != nil || strings.TrimSpace(string(v2Target)) != wave1V3AddendumBase {
		addFinding(findings, wave1V3AddendumPath, "public.prior_v2_publication_tag", "the preserved failed v2 publication tag target must remain unchanged")
		return false
	}
	v2Tree, err := planningGrantGitOutput(root, "rev-parse", "--verify", v2Ref+"^{}^{tree}")
	if err != nil || strings.TrimSpace(string(v2Tree)) != wave1V3AddendumBaseTree {
		addFinding(findings, wave1V3AddendumPath, "public.prior_v2_publication_tag", "the preserved failed v2 publication tag tree must remain unchanged")
		return false
	}
	return true
}

// Contract publication has exactly three admissible checkout states:
//
//   - A local pre-merge checkout must be the exact symbolic branch named by the
//     signed grant. It must remain linear, so an unsigned local merge cannot
//     exploit the deliberate GitHub merge-signature exception.
//   - Pull-request CI must be detached at GitHub's two-parent synthetic merge.
//     Canonical runner facts and the bounded event payload bind HEAD to the
//     immutable base/head SHAs and to the signed feature branch.
//   - Main-push CI may be detached or symbolic main. The one-parent squash
//     commit must have the event's exact protected base and the same tree as a
//     pinned-signer annotated tag. That tag keeps the reviewed signed feature
//     history reachable after GitHub rewrites the public commit identity.
//
// There is no generic detached, arbitrary-branch, or local-main fallback.
func planningGrantCheckoutState(root, head string, findings *[]Finding) (planningGrantCheckout, bool) {
	branchOutput, branchErr := planningGrantGitOutput(root, "symbolic-ref", "--quiet", "--short", "HEAD")
	branch := strings.TrimSpace(string(branchOutput))
	if branchErr == nil && branch == wave1PlanningGrantBranch {
		return planningGrantCheckout{kind: planningGrantLocalBranch, expectedHead: head}, true
	}
	if branchErr == nil && branch != "main" {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_branch", "contract publication must run on the exact signed working branch")
		return planningGrantCheckout{}, false
	}
	if branchErr != nil && branch != "" {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_branch", "symbolic branch state is ambiguous")
		return planningGrantCheckout{}, false
	}

	checkout, ok := planningGrantGitHubCheckout(root, head, branch)
	if !ok {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_branch", "detached or main checkout lacks canonical immutable workflow facts")
		return planningGrantCheckout{}, false
	}
	return checkout, true
}

func planningGrantGitHubCheckout(root, head, branch string) (planningGrantCheckout, bool) {
	if os.Getenv("CI") != "true" || os.Getenv("GITHUB_ACTIONS") != "true" ||
		os.Getenv("RUNNER_ENVIRONMENT") != "github-hosted" ||
		os.Getenv("GITHUB_REPOSITORY") != planningGrantRepository ||
		os.Getenv("GITHUB_WORKFLOW") != planningGrantWorkflow ||
		os.Getenv("GITHUB_JOB") != planningGrantWorkflowJob ||
		os.Getenv("GITHUB_SHA") != head ||
		!samePlanningGrantRepositoryRoot(root, os.Getenv("GITHUB_WORKSPACE")) {
		return planningGrantCheckout{}, false
	}
	if _, ok := parsePositiveInt(os.Getenv("GITHUB_RUN_ID")); !ok {
		return planningGrantCheckout{}, false
	}
	if _, ok := parsePositiveInt(os.Getenv("GITHUB_RUN_ATTEMPT")); !ok {
		return planningGrantCheckout{}, false
	}
	workflowRef := os.Getenv("GITHUB_WORKFLOW_REF")
	workflowPrefix := planningGrantRepository + "/" + planningGrantWorkflowPath + "@"
	if !strings.HasPrefix(workflowRef, workflowPrefix) {
		return planningGrantCheckout{}, false
	}
	workflow, err := readRepoFile(root, planningGrantWorkflowPath)
	if err != nil || fileSHA256(workflow) != canonicalFoundationWorkflowSHA256 {
		return planningGrantCheckout{}, false
	}
	event, ok := readPlanningGrantGitHubEvent(os.Getenv("GITHUB_EVENT_PATH"))
	if !ok || event.Repository.FullName != planningGrantRepository {
		return planningGrantCheckout{}, false
	}

	switch os.Getenv("GITHUB_EVENT_NAME") {
	case "pull_request":
		ref := os.Getenv("GITHUB_REF")
		if branch != "" || !validPlanningGrantPullRequestRef(ref) ||
			os.Getenv("GITHUB_HEAD_REF") != wave1PlanningGrantBranch ||
			os.Getenv("GITHUB_BASE_REF") != "main" || event.PullRequest == nil ||
			event.PullRequest.Head.Ref != wave1PlanningGrantBranch ||
			event.PullRequest.Base.Ref != "main" ||
			(event.PullRequest.MergeCommitSHA != "" && !sha1Pattern.MatchString(event.PullRequest.MergeCommitSHA)) ||
			!sha1Pattern.MatchString(event.PullRequest.Head.SHA) ||
			!sha1Pattern.MatchString(event.PullRequest.Base.SHA) {
			return planningGrantCheckout{}, false
		}
		workflowSuffix := strings.TrimPrefix(workflowRef, workflowPrefix)
		if workflowSuffix != ref && workflowSuffix != "refs/heads/main" {
			return planningGrantCheckout{}, false
		}
		return planningGrantCheckout{
			kind:         planningGrantPullRequestMerge,
			expectedHead: head,
			firstParent:  event.PullRequest.Base.SHA,
			secondParent: event.PullRequest.Head.SHA,
		}, true
	case "push":
		if branch != "" && branch != "main" {
			return planningGrantCheckout{}, false
		}
		if os.Getenv("GITHUB_REF") != "refs/heads/main" || os.Getenv("GITHUB_REF_PROTECTED") != "true" ||
			os.Getenv("GITHUB_HEAD_REF") != "" || os.Getenv("GITHUB_BASE_REF") != "" ||
			workflowRef != workflowPrefix+"refs/heads/main" ||
			event.Ref != "refs/heads/main" || event.After != head || event.HeadCommit == nil || event.HeadCommit.ID != head ||
			event.Before != wave1PlanningGrantBase {
			return planningGrantCheckout{}, false
		}
		return planningGrantCheckout{
			kind:         planningGrantMainSquash,
			expectedHead: head,
			firstParent:  event.Before,
		}, true
	default:
		return planningGrantCheckout{}, false
	}
}

func readPlanningGrantGitHubEvent(path string) (planningGrantGitHubEvent, bool) {
	var event planningGrantGitHubEvent
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() <= 0 || info.Size() > 1<<20 {
		return event, false
	}
	data, err := os.ReadFile(path)
	if err != nil || json.Unmarshal(data, &event) != nil {
		return event, false
	}
	return event, true
}

func validPlanningGrantPullRequestRef(ref string) bool {
	parts := strings.Split(strings.TrimPrefix(ref, "refs/pull/"), "/")
	if len(parts) != 2 || parts[1] != "merge" {
		return false
	}
	_, ok := parsePositiveInt(parts[0])
	return ok
}

func planningGrantPublicationTagTarget(root string, checkout planningGrantCheckout, head string, findings *[]Finding) (string, bool) {
	ref := "refs/tags/" + wave1PublicationTag
	tagObject, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{tag}")
	if err != nil || !sha1Pattern.MatchString(strings.TrimSpace(string(tagObject))) {
		addFinding(findings, wave1DispositionPath, "public.publication_tag_missing", "canonical CI requires the signed immutable Wave-1 publication tag")
		return "", false
	}
	object, err := planningGrantGitOutput(root, "cat-file", "tag", strings.TrimSpace(string(tagObject)))
	if err != nil {
		addFinding(findings, wave1DispositionPath, "public.publication_tag_object", "publication tag object cannot be read")
		return "", false
	}
	publicKey, err := readRepoFile(root, wave1PlanningGrantKey)
	if err != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		addFinding(findings, wave1PlanningGrantKey, "public.publication_tag_key", "publication tag requires the pinned genesis signer")
		return "", false
	}
	target, err := verifyPlanningGrantTag(object, publicKey)
	if err != nil {
		addFinding(findings, wave1DispositionPath, "public.publication_tag_signature", "publication tag is not an exact pinned-signer tree attestation")
		return "", false
	}
	if target == wave1PlanningGrantBase {
		addFinding(findings, wave1DispositionPath, "public.publication_tag_target", "publication tag must attest nonempty reviewed feature history")
		return "", false
	}
	if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", wave1PlanningGrantBase, target); err != nil {
		addFinding(findings, wave1DispositionPath, "public.publication_tag_target", "publication tag target must descend from the exact signed base")
		return "", false
	}
	targetTree, err := planningGrantGitOutput(root, "rev-parse", "--verify", target+"^{tree}")
	if err != nil {
		addFinding(findings, wave1DispositionPath, "public.publication_tag_tree", "publication tag target tree cannot be resolved")
		return "", false
	}
	expectedCommit := head
	if checkout.kind == planningGrantPullRequestMerge {
		expectedCommit = checkout.secondParent
		if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", target, checkout.secondParent); err != nil {
			addFinding(findings, wave1DispositionPath, "public.publication_tag_target", "publication tag target must be an ancestor of the immutable pull-request head")
			return "", false
		}
	}
	expectedTree, err := planningGrantGitOutput(root, "rev-parse", "--verify", expectedCommit+"^{tree}")
	if err != nil || strings.TrimSpace(string(targetTree)) != strings.TrimSpace(string(expectedTree)) {
		addFinding(findings, wave1DispositionPath, "public.publication_tag_tree", "signed publication tag tree must equal the reviewed publication tree")
		return "", false
	}
	return target, true
}

func verifyPlanningGrantTag(object, publicKey []byte) (string, error) {
	const armor = "-----BEGIN SSH SIGNATURE-----"
	signatureIndex := bytes.Index(object, []byte(armor))
	if signatureIndex < 0 || bytes.Count(object, []byte(armor)) != 1 {
		return "", fmt.Errorf("tag must contain exactly one SSH signature")
	}
	signed := object[:signatureIndex]
	signature := object[signatureIndex:]
	if err := verifySSHSig(signed, signature, publicKey, planningGrantCommitNS); err != nil {
		return "", err
	}
	headerEnd := bytes.Index(signed, []byte("\n\n"))
	if headerEnd < 0 {
		return "", fmt.Errorf("tag object has no header boundary")
	}
	var target string
	counts := map[string]int{}
	for _, line := range strings.Split(string(signed[:headerEnd]), "\n") {
		fields := strings.SplitN(line, " ", 2)
		if len(fields) != 2 {
			return "", fmt.Errorf("invalid tag header")
		}
		counts[fields[0]]++
		switch fields[0] {
		case "object":
			target = fields[1]
		case "type":
			if fields[1] != "commit" {
				return "", fmt.Errorf("tag target must be a commit")
			}
		case "tag":
			if fields[1] != wave1PublicationTag {
				return "", fmt.Errorf("tag name does not match the signed recovery disposition")
			}
		case "tagger":
			if !strings.Contains(fields[1], " <release-manager@example.com> ") {
				return "", fmt.Errorf("tagger must be the synthetic public release identity")
			}
		default:
			return "", fmt.Errorf("unexpected tag header")
		}
	}
	for _, field := range []string{"object", "type", "tag", "tagger"} {
		if counts[field] != 1 {
			return "", fmt.Errorf("tag header cardinality is invalid")
		}
	}
	if !sha1Pattern.MatchString(target) {
		return "", fmt.Errorf("tag target is not a canonical commit ID")
	}
	if string(signed[headerEnd+2:]) != wave1PublicationTagMessage+"\n" {
		return "", fmt.Errorf("tag message does not match the signed recovery disposition")
	}
	return target, nil
}

func planningGrantCommitRange(root, end string) ([]planningGrantCommit, error) {
	output, err := planningGrantGitOutput(root, "rev-list", "--no-abbrev-commit", "--reverse", "--topo-order", "--parents", wave1PlanningGrantBase+".."+end)
	if err != nil {
		return nil, err
	}
	var commits []planningGrantCommit
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 || !sha1Pattern.MatchString(fields[0]) {
			return nil, fmt.Errorf("invalid commit ancestry record")
		}
		commit := planningGrantCommit{id: fields[0]}
		for _, parent := range fields[1:] {
			if !sha1Pattern.MatchString(parent) {
				return nil, fmt.Errorf("invalid commit parent")
			}
			commit.parents = append(commit.parents, parent)
		}
		commits = append(commits, commit)
	}
	return commits, nil
}

func planningGrantCommitParents(root, commit string) ([]string, error) {
	output, err := planningGrantGitOutput(root, "rev-list", "--no-abbrev-commit", "--parents", "-n", "1", commit)
	if err != nil {
		return nil, err
	}
	fields := strings.Fields(strings.TrimSpace(string(output)))
	if len(fields) < 1 || fields[0] != commit {
		return nil, fmt.Errorf("invalid commit parent record")
	}
	parents := make([]string, 0, len(fields)-1)
	for _, parent := range fields[1:] {
		if !sha1Pattern.MatchString(parent) {
			return nil, fmt.Errorf("invalid commit parent")
		}
		parents = append(parents, parent)
	}
	return parents, nil
}

func checkPlanningGrantCommitTopology(root string, checkout planningGrantCheckout, head string, commits []planningGrantCommit, findings *[]Finding) bool {
	if checkout.kind == planningGrantLocalBranch {
		for _, commit := range commits {
			if len(commit.parents) != 1 {
				addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_commit_topology", "the pre-merge signed branch must have linear commit history")
				return false
			}
		}
		return true
	}
	if checkout.kind == planningGrantMainSquash {
		if len(commits) == 0 || commits[len(commits)-1].id != checkout.tagTarget {
			addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_commit_topology", "signed publication tag must terminate the preserved feature history")
			return false
		}
		for _, commit := range commits {
			if len(commit.parents) != 1 {
				addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_commit_topology", "tag-preserved feature history must be linear")
				return false
			}
		}
		parents, err := planningGrantCommitParents(root, head)
		if err != nil || len(parents) != 1 || parents[0] != checkout.firstParent {
			addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_commit_topology", "protected main must contain one exact squash commit over the recorded base")
			return false
		}
		return true
	}
	if len(commits) == 0 || commits[len(commits)-1].id != head {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_commit_topology", "canonical CI must end at its declared merge commit")
		return false
	}
	for _, commit := range commits[:len(commits)-1] {
		if len(commit.parents) != 1 {
			addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_commit_topology", "only the canonical terminal CI merge may be unsigned")
			return false
		}
	}
	merge := commits[len(commits)-1]
	if len(merge.parents) != 2 || merge.parents[0] != checkout.firstParent {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_commit_topology", "canonical CI requires the exact two-parent merge recorded by the runner event")
		return false
	}
	if checkout.kind == planningGrantPullRequestMerge && merge.parents[1] != checkout.secondParent {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_commit_topology", "pull-request merge does not contain the immutable signed feature head")
		return false
	}
	return true
}

func verifyPlanningGrantCommit(object, publicKey []byte) error {
	// Git SSH-signs the commit payload under the protocol-defined "git"
	// namespace. The detached grant document is independently verified under
	// mars3-planning-grant; accepting either namespace for both artifacts would
	// make a valid signature replayable across authority surfaces.
	headerEnd := bytes.Index(object, []byte("\n\n"))
	if headerEnd < 0 {
		return fmt.Errorf("commit object has no header boundary")
	}
	headers := bytes.Split(object[:headerEnd], []byte("\n"))
	var signed bytes.Buffer
	var signature bytes.Buffer
	signatureCount := 0
	for index := 0; index < len(headers); index++ {
		line := headers[index]
		if !bytes.HasPrefix(line, []byte("gpgsig ")) {
			signed.Write(line)
			signed.WriteByte('\n')
			continue
		}
		signatureCount++
		signature.Write(bytes.TrimPrefix(line, []byte("gpgsig ")))
		signature.WriteByte('\n')
		for index+1 < len(headers) && bytes.HasPrefix(headers[index+1], []byte(" ")) {
			index++
			signature.Write(headers[index][1:])
			signature.WriteByte('\n')
		}
	}
	if signatureCount != 1 {
		return fmt.Errorf("commit must have exactly one signature header")
	}
	signed.WriteByte('\n')
	signed.Write(object[headerEnd+2:])
	return verifySSHSig(signed.Bytes(), signature.Bytes(), publicKey, planningGrantCommitNS)
}

func planningGrantGitOutput(root string, arguments ...string) ([]byte, error) {
	prefix := []string{"-c", "core.fsmonitor=false", "-c", "diff.external=", "-C", root}
	command := exec.Command("git", append(prefix, arguments...)...)
	command.Env = planningGrantGitEnvironment()
	return command.Output()
}

func planningGrantGitEnvironment() []string {
	environment := make([]string, 0, len(os.Environ())+5)
	for _, entry := range os.Environ() {
		key := entry
		if separator := strings.IndexByte(entry, '='); separator >= 0 {
			key = entry[:separator]
		}
		switch key {
		case "GIT_ALTERNATE_OBJECT_DIRECTORIES", "GIT_ATTR_NOSYSTEM", "GIT_COMMON_DIR", "GIT_CONFIG_COUNT", "GIT_CONFIG_GLOBAL", "GIT_CONFIG_NOSYSTEM", "GIT_CONFIG_PARAMETERS", "GIT_DIR", "GIT_INDEX_FILE", "GIT_NO_REPLACE_OBJECTS", "GIT_OBJECT_DIRECTORY", "GIT_PREFIX", "GIT_SHALLOW_FILE", "GIT_WORK_TREE":
			continue
		}
		environment = append(environment, entry)
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

func samePlanningGrantRepositoryRoot(root, topLevel string) bool {
	if topLevel == "" {
		return false
	}
	rootInfo, rootErr := os.Stat(root)
	topInfo, topErr := os.Stat(topLevel)
	if rootErr != nil || topErr != nil {
		return false
	}
	return os.SameFile(rootInfo, topInfo)
}

func normalizedPlanningGrantGitPaths(outputs ...[]byte) ([]string, error) {
	unique := make(map[string]bool)
	for _, output := range outputs {
		if len(output) == 0 {
			continue
		}
		if output[len(output)-1] != 0 {
			return nil, fmt.Errorf("Git path output is not NUL terminated")
		}
		for _, encoded := range bytes.Split(output[:len(output)-1], []byte{0}) {
			if len(encoded) == 0 || !utf8.Valid(encoded) {
				return nil, fmt.Errorf("Git returned an empty or non-UTF-8 path")
			}
			path := string(encoded)
			if !safeRelativePath(path) || filepath.ToSlash(filepath.Clean(path)) != path {
				return nil, fmt.Errorf("Git returned a non-canonical path")
			}
			unique[path] = true
		}
	}
	paths := make([]string, 0, len(unique))
	for path := range unique {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths, nil
}

func parseStrictPlanningGrant(data []byte) strictPlanningGrant {
	return parseStrictGrant(data, wave1PlanningGrantScalars, wave1PlanningGrantSequences, []string{"grant", "verification", "integrity"})
}

func parseStrictGrant(data []byte, expectedScalars []grantScalarExpectation, expectedSequences map[string][]string, expectedSections []string) strictPlanningGrant {
	document := strictPlanningGrant{
		scalars:         make(map[string][]string),
		sequences:       make(map[string][]string),
		sections:        make(map[string]int),
		sequenceHeaders: make(map[string]int),
	}
	if !utf8.Valid(data) {
		document.structuralErrors = append(document.structuralErrors, "grant must be valid UTF-8")
		return document
	}

	knownScalars := make(map[string]bool, len(expectedScalars))
	for _, expected := range expectedScalars {
		knownScalars[expected.path] = true
	}
	knownSequences := make(map[string]bool, len(expectedSequences))
	for path := range expectedSequences {
		knownSequences[path] = true
	}
	knownSections := make(map[string]bool, len(expectedSections))
	for _, section := range expectedSections {
		knownSections[section] = true
	}

	section := ""
	sequence := ""
	for index, raw := range strings.Split(string(data), "\n") {
		lineNumber := index + 1
		raw = strings.TrimSuffix(raw, "\r")
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		if strings.Contains(raw, "\t") {
			document.structuralErrors = append(document.structuralErrors, fmt.Sprintf("line %d uses a tab", lineNumber))
			continue
		}
		if strings.HasPrefix(trimmed, "#") || trimmed == "---" || trimmed == "..." || strings.HasPrefix(trimmed, "%") {
			document.structuralErrors = append(document.structuralErrors, fmt.Sprintf("line %d uses unsupported YAML syntax", lineNumber))
			continue
		}
		indent := len(raw) - len(strings.TrimLeft(raw, " "))
		if strings.HasPrefix(trimmed, "-") {
			if indent != 4 || sequence == "" || !strings.HasPrefix(trimmed, "- ") {
				document.structuralErrors = append(document.structuralErrors, fmt.Sprintf("line %d is not a scalar item of a declared sequence", lineNumber))
				continue
			}
			item := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
			if item == "" || unsafeGrantYAMLValue(item) {
				document.structuralErrors = append(document.structuralErrors, fmt.Sprintf("line %d uses an empty or indirect sequence value", lineNumber))
				continue
			}
			document.sequences[sequence] = append(document.sequences[sequence], item)
			continue
		}

		sequence = ""
		colon := strings.Index(trimmed, ":")
		if colon < 1 {
			document.structuralErrors = append(document.structuralErrors, fmt.Sprintf("line %d is not a plain mapping entry", lineNumber))
			continue
		}
		key := strings.TrimSpace(trimmed[:colon])
		value := strings.TrimSpace(trimmed[colon+1:])
		if key == "" || unsafeGrantYAMLKey(key) || unsafeGrantYAMLValue(value) {
			document.structuralErrors = append(document.structuralErrors, fmt.Sprintf("line %d uses YAML indirection or an unsafe mapping", lineNumber))
			continue
		}

		switch indent {
		case 0:
			section = ""
			if knownSections[key] {
				if value != "" {
					document.structuralErrors = append(document.structuralErrors, fmt.Sprintf("line %d must open a mapping", lineNumber))
					continue
				}
				document.sections[key]++
				section = key
				continue
			}
			if !knownScalars[key] {
				document.structuralErrors = append(document.structuralErrors, fmt.Sprintf("line %d declares an unknown root field", lineNumber))
				continue
			}
			if value == "" {
				document.structuralErrors = append(document.structuralErrors, fmt.Sprintf("line %d requires a scalar value", lineNumber))
				continue
			}
			document.scalars[key] = append(document.scalars[key], value)
		case 2:
			if section == "" {
				document.structuralErrors = append(document.structuralErrors, fmt.Sprintf("line %d has no declared parent mapping", lineNumber))
				continue
			}
			path := section + "." + key
			if knownSequences[path] {
				if value != "" {
					document.structuralErrors = append(document.structuralErrors, fmt.Sprintf("line %d must open a block scalar sequence", lineNumber))
					continue
				}
				document.sequenceHeaders[path]++
				sequence = path
				continue
			}
			if !knownScalars[path] {
				document.structuralErrors = append(document.structuralErrors, fmt.Sprintf("line %d declares an unknown contract field", lineNumber))
				continue
			}
			if value == "" {
				document.structuralErrors = append(document.structuralErrors, fmt.Sprintf("line %d requires a scalar value", lineNumber))
				continue
			}
			document.scalars[path] = append(document.scalars[path], value)
		default:
			document.structuralErrors = append(document.structuralErrors, fmt.Sprintf("line %d has unsupported nesting", lineNumber))
		}
	}
	return document
}

func unsafeGrantYAMLKey(value string) bool {
	return strings.ContainsAny(value, "&*!{}[]|><#") || strings.Contains(value, "<<")
}

func unsafeGrantYAMLValue(value string) bool {
	if value == "" {
		return false
	}
	if strings.Contains(value, "<<:") || strings.ContainsAny(value, "{}[]") {
		return true
	}
	for _, prefix := range []string{"&", "*", "!", "|", ">", "'", "\""} {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	for _, token := range []string{" &", " *", " !"} {
		if strings.Contains(value, token) {
			return true
		}
	}
	return false
}

func equalStringSequence(actual, expected []string) bool {
	if len(actual) != len(expected) {
		return false
	}
	for index := range expected {
		if actual[index] != expected[index] {
			return false
		}
	}
	return true
}
