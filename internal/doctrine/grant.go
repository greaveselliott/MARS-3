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
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	wave1PlanningGrantPath            = ".harness/grants/WAVE-1-contract-publication.yaml"
	wave1PlanningGrantSignature       = ".harness/grants/WAVE-1-contract-publication.yaml.sig"
	wave1PlanningGrantKey             = ".harness/keys/genesis-signing-key.pub"
	wave1PlanningGrantNamespace       = "mars3-planning-grant"
	wave1PlanningGrantBase            = "ee385ce236ae1f99da692d223d7666b80dd9108f"
	wave1PlanningGrantBranch          = "codex/w-001-work-authority"
	planningGrantCommitNS             = "git"
	planningGrantRepository           = "greaveselliott/MARS-3"
	planningGrantWorkflow             = "Foundation quality"
	planningGrantWorkflowPath         = ".github/workflows/foundation-quality.yml"
	planningGrantWorkflowJob          = "public-commit-gate"
	wave1DispositionPath              = ".harness/grants/WAVE-1-recovery-disposition.yaml"
	wave1DispositionSignature         = ".harness/grants/WAVE-1-recovery-disposition.yaml.sig"
	wave1DispositionSnapshot          = ".harness/grants/WAVE-1-authority-recovery-state.json"
	wave1DispositionNamespace         = "mars3-recovery-disposition"
	wave1DispositionSnapshotSHA       = "4d3b5c9d90a223c0e9d974e836559309a2f4dac7f209a3966336e9152f57feca"
	wave1PriorPublicationTag          = "mars3/wave1-contract-publication-v1"
	wave1PriorPublicationTagMessage   = "MARS-3 Wave-1 contract-publication tree attestation v1"
	wave1PriorPublicationTagObject    = "4bce7e7d4a8b2cc1a5b30b9feaee61232c3cc0de"
	wave1AddendumPath                 = ".harness/grants/WAVE-1-ci-recovery-addendum.yaml"
	wave1AddendumSignature            = ".harness/grants/WAVE-1-ci-recovery-addendum.yaml.sig"
	wave1AddendumNamespace            = "mars3-ci-recovery-addendum"
	wave1AddendumBase                 = "a22cfe6fada6f2bc787742eae50bca28cec80c89"
	wave1AddendumBaseTree             = "3c5befaefab37a8d0a2e3a8af2efd6e1eb1d8cae"
	wave1V2PublicationTag             = "mars3/wave1-contract-publication-v2"
	wave1V2PublicationTagMessage      = "MARS-3 Wave-1 contract-publication tree attestation v2"
	wave1V2PublicationTagObject       = "e334356519188fc0906549515ae57fbffa646829"
	wave1V3AddendumPath               = ".harness/grants/WAVE-1-ci-recovery-addendum-v3.yaml"
	wave1V3AddendumSignature          = ".harness/grants/WAVE-1-ci-recovery-addendum-v3.yaml.sig"
	wave1V3AddendumNamespace          = "mars3-ci-recovery-addendum-v3"
	wave1V3AddendumBase               = "412a9b857265af250ee40d36d0a6c127714e4ec9"
	wave1V3AddendumBaseTree           = "8c7f3ccac3e31d0e8b45431934cd95a91e448c0f"
	wave1V3ObservedStaleMerge         = "fff2bea9bffa9400d3ecfc147b7338849ecfbbb0"
	wave1PublicationTag               = "mars3/wave1-contract-publication-v3"
	wave1PublicationTagMessage        = "MARS-3 Wave-1 contract-publication tree attestation v3"
	wave1PublishedMain                = "265b0b78a19d0ac50611c360f4614ed24d1cfcd7"
	wave1PublishedTree                = "87dc32acce9c767f01f94aae25936665e26650ab"
	wave1DirectMainGrantPath          = ".harness/grants/WAVE-1-direct-main-transition.yaml"
	wave1DirectMainGrantSignature     = ".harness/grants/WAVE-1-direct-main-transition.yaml.sig"
	wave1DirectMainGrantNamespace     = "mars3-direct-main-transition"
	wave1PRFallbackPath               = ".harness/grants/WAVE-1-pr-fallback-addendum.yaml"
	wave1PRFallbackSignature          = ".harness/grants/WAVE-1-pr-fallback-addendum.yaml.sig"
	wave1PRFallbackNamespace          = "mars3-pr-fallback-addendum"
	wave1PRFallbackBranch             = "codex/w-001-bootstrap-transition"
	wave1PRFallbackFirstCommit        = "0602a7e2036f53d2d7b9da10b406d27f87621196"
	wave1PRFallbackFirstTree          = "69d9ab152f5b0a243855ddf613192c662b52306b"
	wave1TransitionTag                = "mars3/w001-transition-v1"
	wave1TransitionTagMessage         = "MARS-3 W-001 transition tree attestation v1"
	wave1TransitionTagObject          = "394c9ce749142c2222c1b8081b62f43a895be326"
	wave1TransitionReviewedHead       = "a9e203540a9e280fe7059dfa579a9a4089fc3f52"
	wave1TransitionReviewedTree       = "c326c5d33f075a2c8b7dc4c41a698d57175c8805"
	wave1MainCIFixPath                = ".harness/grants/WAVE-1-pr-fallback-main-ci-addendum.yaml"
	wave1MainCIFixSignature           = ".harness/grants/WAVE-1-pr-fallback-main-ci-addendum.yaml.sig"
	wave1MainCIFixNamespace           = "mars3-pr-fallback-main-ci-addendum"
	wave1SuccessorTransitionTag       = "mars3/w001-transition-v2"
	wave1SuccessorTagMessage          = "MARS-3 W-001 transition tree attestation v2"
	wave1SuccessorTagObject           = "cd7d6daeea77041167b5aa3763952b47b4ad09c0"
	wave1CIFixtureReviewedHead        = "b9ca75dfe42001a9632d8c752e2ddb80624fa4ae"
	wave1CIFixtureReviewedTree        = "f8af794a23f67fec73af71328dc6aeae0fa9e104"
	wave1CIFixtureFixPath             = ".harness/grants/WAVE-1-pr-fallback-fixture-stabilization-addendum.yaml"
	wave1CIFixtureFixSignature        = ".harness/grants/WAVE-1-pr-fallback-fixture-stabilization-addendum.yaml.sig"
	wave1CIFixtureFixNamespace        = "mars3-pr-fallback-fixture-stabilization-addendum"
	wave1FinalTransitionTag           = "mars3/w001-transition-v3"
	wave1FinalTransitionTagMessage    = "MARS-3 W-001 transition tree attestation v3"
	w001BootstrapGrantPath            = ".harness/grants/W-001-bootstrap.yaml"
	w001BootstrapGrantSignature       = ".harness/grants/W-001-bootstrap.yaml.sig"
	w001BootstrapGrantNamespace       = "mars3-w001-bootstrap-grant"
	w001BootstrapExecutionNamespace   = "mars3-w001-bootstrap-execution"
	w001BootstrapBase                 = "37b55b912b20715349bc50e0524c85d4b22f1772"
	w001BootstrapBaseTree             = "f06864b0802cea793cf7a0c08b60b7e734539a94"
	w001BootstrapBranch               = "codex/w-001-work-authority"
	w001BootstrapReviewTag            = "mars3/w001-bootstrap-helper-v9"
	w001BootstrapReviewTagMessage     = "MARS-3 W-001 bootstrap helper tree attestation v9"
	w001PostclaimGrantPath            = ".harness/grants/W-001-postclaim-reconciliation.yaml"
	w001PostclaimGrantSignature       = ".harness/grants/W-001-postclaim-reconciliation.yaml.sig"
	w001PostclaimGrantNamespace       = "mars3-w001-postclaim-reconciliation"
	w001PostclaimBase                 = "adfd64feb565fb703a3568122cc032d4d1a450f5"
	w001PostclaimBaseTree             = "bbfa7f59f7bd29c1a0546ffcb77dd8fa4982ef6d"
	w001PostclaimBranch               = "codex/w-001-postclaim-reconciliation"
	w001PostclaimReviewTag            = "mars3/w001-postclaim-reconciliation-v1"
	w001PostclaimReviewTagMessage     = "MARS-3 W-001 postclaim reconciliation tree attestation v1"
	w001PostclaimCIFixPath            = ".harness/grants/W-001-postclaim-ci-stabilization-v2.yaml"
	w001PostclaimCIFixSignature       = ".harness/grants/W-001-postclaim-ci-stabilization-v2.yaml.sig"
	w001PostclaimCIFixNamespace       = "mars3-w001-postclaim-ci-stabilization-v2"
	w001PostclaimCIFixBase            = "eda666569de379543a170119ccb7c560478c7346"
	w001PostclaimCIFixBaseTree        = "b8de9ac4cf26b4561ce26abcf729529f65bd9b9f"
	w001PostclaimCIFixReviewTag       = "mars3/w001-postclaim-reconciliation-v2"
	w001PostclaimCIFixTagMessage      = "MARS-3 W-001 postclaim reconciliation tree attestation v2"
	w001PostclaimV1TagObject          = "7492f63fd88567a284eff43c670098295824aaf8"
	w001PostclaimV2TagObject          = "684ba6c536e8e8d4c33b587da9e7d05893168958"
	w001PostclaimSecurityFixPath      = ".harness/grants/W-001-postclaim-security-correction-v3.yaml"
	w001PostclaimSecurityFixSig       = ".harness/grants/W-001-postclaim-security-correction-v3.yaml.sig"
	w001PostclaimSecurityFixNS        = "mars3-w001-postclaim-security-correction-v3"
	w001PostclaimSecurityFixBase      = "20542d8e696abe0a71b6ec3ceb23f042919fbc04"
	w001PostclaimSecurityFixTree      = "499ea91f5002b32d57d6f20b4ca3ea07dbdc73f5"
	w001PostclaimSecurityFixTag       = "mars3/w001-postclaim-reconciliation-v3"
	w001PostclaimSecurityFixTagMsg    = "MARS-3 W-001 postclaim reconciliation tree attestation v3"
	w001PostclaimSecurityHelperSHA    = "21ff880af1135db6c13ad6dcfac621e9231fee8e635ca7d47e5fcc5c39314f25"
	w001PostclaimSecurityTestSHA      = "0221972a39c14108719b80a23448f9c42f2b9417a3d4ea3dc4f5c1b2975a0f33"
	w001PostclaimSecurityBasePatchSHA = "50128252828352366ced6560371468a5746c2603ef89ea746a33be8994ffceb6"
	w001PostclaimSecurityPatchPath    = "internal/authority/bootstrap/beads-v1.2.2-effective-db-security.patch"
	w001PostclaimSecurityPatchSHA     = "d48b398a8688d337192ab030c69fd9df0809f72051da7850ff2fdbad5e322d45"
	w001PostclaimSecurityBinarySHA    = "8d13927671519fd74470820a72c6ff069589655e338649f31db7e654b2b36c00"
)

// W001BootstrapGrant is the validated public projection consumed by the
// one-shot bootstrap helper. It contains no local path, credential, or signer
// state.
type W001BootstrapGrant struct {
	ID                          string
	AttemptID                   string
	IdempotencyKey              string
	Bead                        string
	BaseCommit                  string
	WorkingBranch               string
	Assignee                    string
	AuthorityProjectID          string
	ExpectedNativeStatus        string
	ExpectedLifecycleState      string
	ExpectedCreatedAt           string
	ExpectedUpdatedAt           string
	ExpectedMetadataSHA256      string
	ExpectedLabelsSHA256        string
	ExpectedDependency          string
	ExpectedDependencyType      string
	ExpectedDependencyStatus    string
	ExpectedDependencyLifecycle string
	ExpectedDependencySHA256    string
	ExpectedLineageSHA256       string
	ExpiresAt                   string
	PostNativeStatus            string
	PostLifecycleState          string
	PostMetadataSHA256          string
	PostLabelsSHA256            string
	PostMetadataBase64          string
	RemoveLabel                 string
	AddLabel                    string
	BeadsVersion                string
	BeadsSourceCommit           string
	BeadsBinarySHA256           string
	DoltModule                  string
	DoltModuleSHA256            string
	GoVersion                   string
	GoOS                        string
	GoArch                      string
	ICUFormula                  string
	DoltTestImage               string
	PatchPath                   string
	PatchSHA256                 string
	CorrectionPatchPath         string
	CorrectionPatchSHA256       string
	PatchedBinarySHA256         string
	GoBinarySHA256              string
	ReviewTag                   string
}

// W001BootstrapExecutionAuthorization is a short-lived, independently signed
// post-review effect token. It is intentionally external to Git so it can bind
// the actual protected-main commit and check after the helper is merged.
type W001BootstrapExecutionAuthorization struct {
	SchemaVersion           int    `json:"schemaVersion"`
	Kind                    string `json:"kind"`
	Classification          string `json:"classification"`
	GrantID                 string `json:"grantId"`
	Repository              string `json:"repository"`
	AttemptID               string `json:"attemptId"`
	IdempotencyKey          string `json:"idempotencyKey"`
	Bead                    string `json:"bead"`
	AuthorityProjectID      string `json:"authorityProjectId"`
	MergedCommit            string `json:"mergedCommit"`
	MergedTree              string `json:"mergedTree"`
	ReviewTag               string `json:"reviewTag"`
	ReviewedFeatureCommit   string `json:"reviewedFeatureCommit"`
	PullRequest             int    `json:"pullRequest"`
	ProtectedMainCheckRun   int64  `json:"protectedMainCheckRun"`
	QAReviewedCommit        string `json:"qaReviewedCommit"`
	QADisposition           string `json:"qaDisposition"`
	SecurityReviewedCommit  string `json:"securityReviewedCommit"`
	SecurityDisposition     string `json:"securityDisposition"`
	PatchedBinarySHA256     string `json:"patchedBinarySHA256"`
	ExpectedMetadataSHA256  string `json:"expectedMetadataSHA256"`
	WorkspaceInstanceSHA256 string `json:"workspaceInstanceSHA256"`
	AllowedEffect           string `json:"allowedEffect"`
	IssuedAt                string `json:"issuedAt"`
	ExpiresAt               string `json:"expiresAt"`
	payloadSHA256           string
	signatureSHA256         string
}

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

var wave1DirectMainGrantScalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3DirectMainTransitionGrant"},
	{path: "grant.id", value: "WAVE-1-direct-main-transition"},
	{path: "grant.classification", value: "PUBLIC"},
	{path: "grant.issuedAt", value: "2026-08-26T08:30:17Z"},
	{path: "grant.expiresAt", value: "2026-08-29T08:30:17Z"},
	{path: "grant.baseCommit", value: wave1PublishedMain},
	{path: "grant.baseTree", value: wave1PublishedTree},
	{path: "grant.repository", value: planningGrantRepository},
	{path: "grant.workingBranch", value: "main"},
	{path: "grant.signerRole", value: "human-bootstrap-authority"},
	{path: "grant.coordinator", value: "delivery-orchestrator"},
	{path: "grant.failureOwnership", value: "foundation"},
	{path: "grant.sourceDirective", value: "commit-small-semantic-changes-directly-to-main-frequently"},
	{path: "grant.purpose", value: "prepare fail-closed admission for the separately signed W-001 bootstrap grant without claiming or implementing W-001"},
	{path: "grant.priorPublicationTag", value: wave1PublicationTag},
	{path: "grant.priorPublicationTagObject", value: "b53728e3a57e6dc0d57151aa7f0bed8e44aaaa2f"},
	{path: "grant.priorPublicationTagTarget", value: "7e6b765c284788442553d40792db0afb128c4872"},
	{path: "grant.priorPublicationTree", value: wave1PublishedTree},
	{path: "grant.successorGrant", value: "W-001-bootstrap"},
	{path: "grant.autonomousMutation", value: "false"},
	{path: "grant.implementationClaimed", value: "false"},
	{path: "grant.liveLeaseAsserted", value: "false"},
	{path: "grant.canonicalWorkMutationAllowed", value: "false"},
	{path: "grant.githubPolicyMutationAllowed", value: "false"},
	{path: "grant.secretScannerExceptionAllowed", value: "false"},
	{path: "verification.publicCommitGateRequiredAfterEveryCommit", value: "true"},
	{path: "verification.directMainPushRequired", value: "true"},
	{path: "verification.exactBaseAndTreeRequired", value: "true"},
	{path: "verification.externalStateReadbackRequired", value: "true"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: wave1DirectMainGrantNamespace},
	{path: "integrity.detachedSignature", value: "WAVE-1-direct-main-transition.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var wave1DirectMainGrantSequences = map[string][]string{
	"grant.allowedEffects": {
		"edit-listed-public-paths",
		"create-pinned-signer-direct-main-commits",
		"push-protected-main",
		"run-public-commit-gate-after-every-commit",
		"prepare-separately-signed-W001-bootstrap-grant",
		"append-public-safe-transition-intent-and-receipt",
	},
	"grant.authorizedPaths": {
		wave1DirectMainGrantPath,
		wave1DirectMainGrantSignature,
		".harness/grants/W-001-bootstrap.yaml",
		".harness/grants/W-001-bootstrap.yaml.sig",
		"internal/doctrine/grant.go",
		"internal/doctrine/grant_test.go",
	},
	"grant.requiredProperties": {
		"every-post-publication-commit-is-linear-and-pinned-signer-verified",
		"every-protected-main-push-binds-event-before-after-and-first-parent",
		"every-commit-and-worktree-path-is-inside-the-active-signed-authority",
		"contract-publication-remains-backlog-unclaimed-until-the-successor-grant-and-claim-receipt-validate",
		"v1-v2-v3-publication-tags-remain-immutable",
		"no-transition-gap-is-selected-by-mutable-plan-text",
	},
	"grant.prohibitedEffects": {
		"authorize-prior-effects-retroactively",
		"move-or-delete-v1-v2-or-v3-publication-tag",
		"create-or-move-another-publication-tag",
		"mutate-workflow-ruleset-repository-settings-or-secret-scanner-policy",
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

var wave1PRFallbackScalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3PRFallbackAddendum"},
	{path: "addendum.id", value: "WAVE-1-pr-fallback-addendum"},
	{path: "addendum.classification", value: "PUBLIC"},
	{path: "addendum.issuedAt", value: "2026-08-26T08:40:31Z"},
	{path: "addendum.expiresAt", value: "2026-08-29T08:40:31Z"},
	{path: "addendum.baseCommit", value: wave1PublishedMain},
	{path: "addendum.baseTree", value: wave1PublishedTree},
	{path: "addendum.repository", value: planningGrantRepository},
	{path: "addendum.workingBranch", value: wave1PRFallbackBranch},
	{path: "addendum.signerRole", value: "human-bootstrap-authority"},
	{path: "addendum.coordinator", value: "delivery-orchestrator"},
	{path: "addendum.failureOwnership", value: "foundation"},
	{path: "addendum.sourceDirective", value: "if-using-prs-validate-and-merge-them-promptly"},
	{path: "addendum.purpose", value: "route the signed W-001 transition gate through the repository-required pull-request check without weakening protected main"},
	{path: "addendum.priorGrant", value: "WAVE-1-direct-main-transition"},
	{path: "addendum.priorGrantSHA256", value: "0e8a8179c8fb44d6a23421955bdfb74878cc7cb21cbb32b8b589a2fa9e99e4ba"},
	{path: "addendum.priorGrantSignatureSHA256", value: "93ab5b024bc859eea7d5a3adc026cfd87636c6160cfee07b99fc8f8e8f4d3555"},
	{path: "addendum.rejectedDirectCommit", value: wave1PRFallbackFirstCommit},
	{path: "addendum.rejectedDirectTree", value: wave1PRFallbackFirstTree},
	{path: "addendum.rejectedDirectParent", value: wave1PublishedMain},
	{path: "addendum.failureFingerprint", value: "github-rules-direct-main-requires-pr"},
	{path: "addendum.rulesetId", value: "21510926"},
	{path: "addendum.retryDirectPush", value: "false"},
	{path: "addendum.mergeMode", value: "squash"},
	{path: "addendum.transitionTag", value: wave1TransitionTag},
	{path: "addendum.transitionTagMessage", value: wave1TransitionTagMessage},
	{path: "addendum.autonomousMutation", value: "false"},
	{path: "addendum.canonicalWorkMutationAllowed", value: "false"},
	{path: "addendum.githubPolicyMutationAllowed", value: "false"},
	{path: "addendum.secretScannerExceptionAllowed", value: "false"},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.externalStateReadbackRequired", value: "true"},
	{path: "verification.protectedMainUnchangedUntilMerge", value: "true"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: wave1PRFallbackNamespace},
	{path: "integrity.detachedSignature", value: "WAVE-1-pr-fallback-addendum.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var wave1PRFallbackSequences = map[string][]string{
	"addendum.allowedEffects": {
		"create-exact-working-branch-at-rejected-direct-commit",
		"edit-listed-public-paths",
		"create-pinned-signer-correction-commit",
		"create-and-push-one-pinned-signer-transition-tag",
		"push-working-branch",
		"open-one-ready-pull-request",
		"run-required-public-commit-gate",
		"squash-merge-promptly-after-qa-and-security-acceptance",
		"append-public-safe-fallback-intent-and-receipt",
	},
	"addendum.authorizedPaths": {
		wave1PRFallbackPath,
		wave1PRFallbackSignature,
		"internal/doctrine/grant.go",
		"internal/doctrine/grant_test.go",
	},
	"addendum.requiredProperties": {
		"preserve-the-rejected-signed-commit-exactly",
		"verify-every-feature-commit-with-the-pinned-signer",
		"require-exact-base-head-event-and-two-parent-PR-topology",
		"require-signed-transition-tag-tree-equal-reviewed-and-squash-main-trees",
		"require-public-gate-on-PR-and-protected-main",
		"do-not-retry-the-rejected-direct-push",
	},
	"addendum.prohibitedEffects": {
		"authorize-the-rejected-effect-retroactively",
		"retry-direct-main-push",
		"mutate-workflow-ruleset-repository-settings-or-secret-scanner-policy",
		"move-or-delete-existing-publication-tags",
		"create-or-move-any-tag-other-than-the-transition-tag",
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

var wave1MainCIFixScalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3PRFallbackMainCIAddendum"},
	{path: "addendum.id", value: "WAVE-1-pr-fallback-main-ci-addendum"},
	{path: "addendum.classification", value: "PUBLIC"},
	{path: "addendum.issuedAt", value: "2026-08-26T09:36:15Z"},
	{path: "addendum.expiresAt", value: "2026-08-29T09:36:15Z"},
	{path: "addendum.baseCommit", value: wave1TransitionReviewedHead},
	{path: "addendum.baseTree", value: wave1TransitionReviewedTree},
	{path: "addendum.publicationBaseCommit", value: wave1PublishedMain},
	{path: "addendum.publicationBaseTree", value: wave1PublishedTree},
	{path: "addendum.repository", value: planningGrantRepository},
	{path: "addendum.workingBranch", value: wave1PRFallbackBranch},
	{path: "addendum.pullRequest", value: "5"},
	{path: "addendum.signerRole", value: "human-bootstrap-authority"},
	{path: "addendum.coordinator", value: "delivery-orchestrator"},
	{path: "addendum.failureOwnership", value: "foundation"},
	{path: "addendum.sourceDirective", value: "authorised"},
	{path: "addendum.purpose", value: "correct protected-main squash admission by binding the signed reviewed feature tree without requiring impossible commit identity"},
	{path: "addendum.priorAddendum", value: "WAVE-1-pr-fallback-addendum"},
	{path: "addendum.priorAddendumSHA256", value: "32002ec48fdc8d7c2d5c8300df9be39b5b7c944edf0b16419fdd9b2fd7561fe9"},
	{path: "addendum.priorAddendumSignatureSHA256", value: "00bad1bea742a9c63bc9ef3016e6bf887d7a1d0ab1c7f1ec2b222651f80d9d9e"},
	{path: "addendum.reviewDispositionComment", value: "01a03d6c-f79c-77e3-848c-df769ff27d48"},
	{path: "addendum.reviewedHead", value: wave1TransitionReviewedHead},
	{path: "addendum.reviewedTree", value: wave1TransitionReviewedTree},
	{path: "addendum.failureFingerprint", value: "public-pr-fallback-tag-main-squash-commit-identity-unsatisfiable"},
	{path: "addendum.priorTransitionTag", value: wave1TransitionTag},
	{path: "addendum.priorTransitionTagObject", value: wave1TransitionTagObject},
	{path: "addendum.priorTransitionTagTarget", value: wave1TransitionReviewedHead},
	{path: "addendum.priorTransitionTagTree", value: wave1TransitionReviewedTree},
	{path: "addendum.successorTransitionTag", value: wave1SuccessorTransitionTag},
	{path: "addendum.successorTransitionTagMessage", value: wave1SuccessorTagMessage},
	{path: "addendum.mergeMode", value: "squash"},
	{path: "addendum.autonomousMutation", value: "false"},
	{path: "addendum.canonicalWorkMutationAllowed", value: "false"},
	{path: "addendum.githubPolicyMutationAllowed", value: "false"},
	{path: "addendum.secretScannerExceptionAllowed", value: "false"},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.externalStateReadbackRequired", value: "true"},
	{path: "verification.protectedMainUnchangedUntilMerge", value: "true"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: wave1MainCIFixNamespace},
	{path: "integrity.detachedSignature", value: "WAVE-1-pr-fallback-main-ci-addendum.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var wave1MainCIFixSequences = map[string][]string{
	"addendum.allowedEffects": {
		"edit-listed-public-paths",
		"create-one-pinned-signer-correction-commit",
		"preserve-the-v1-transition-tag-exactly",
		"create-and-push-one-pinned-signer-v2-transition-tag",
		"update-existing-pull-request",
		"run-required-public-commit-gate",
		"squash-merge-promptly-after-fresh-qa-and-security-acceptance",
		"append-public-safe-correction-intent-and-receipt",
	},
	"addendum.authorizedPaths": {
		wave1MainCIFixPath,
		wave1MainCIFixSignature,
		"internal/doctrine/grant.go",
		"internal/doctrine/grant_test.go",
	},
	"addendum.requiredProperties": {
		"preserve-reviewed-head-and-v1-tag-as-immutable-history",
		"verify-every-feature-commit-with-the-pinned-signer",
		"require-v2-tag-target-equal-current-reviewed-feature-head-on-pull-request",
		"require-v2-tag-tree-equal-pull-request-and-squash-main-trees",
		"require-squash-main-commit-distinct-from-v2-tag-target",
		"require-exact-base-head-event-and-topology",
		"require-public-gate-on-updated-pull-request-and-protected-main",
	},
	"addendum.prohibitedEffects": {
		"authorize-the-failed-review-retroactively",
		"move-or-delete-the-v1-transition-tag",
		"retry-direct-main-push",
		"mutate-workflow-ruleset-repository-settings-or-secret-scanner-policy",
		"create-or-move-any-tag-other-than-the-v2-transition-tag",
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

var wave1CIFixtureFixScalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3PRFallbackFixtureStabilizationAddendum"},
	{path: "addendum.id", value: "WAVE-1-pr-fallback-fixture-stabilization-addendum"},
	{path: "addendum.classification", value: "PUBLIC"},
	{path: "addendum.issuedAt", value: "2026-08-26T09:53:55Z"},
	{path: "addendum.expiresAt", value: "2026-08-29T09:53:55Z"},
	{path: "addendum.baseCommit", value: wave1CIFixtureReviewedHead},
	{path: "addendum.baseTree", value: wave1CIFixtureReviewedTree},
	{path: "addendum.publicationBaseCommit", value: wave1PublishedMain},
	{path: "addendum.publicationBaseTree", value: wave1PublishedTree},
	{path: "addendum.repository", value: planningGrantRepository},
	{path: "addendum.workingBranch", value: wave1PRFallbackBranch},
	{path: "addendum.pullRequest", value: "5"},
	{path: "addendum.signerRole", value: "human-bootstrap-authority"},
	{path: "addendum.coordinator", value: "delivery-orchestrator"},
	{path: "addendum.failureOwnership", value: "foundation"},
	{path: "addendum.sourceDirective", value: "authorised"},
	{path: "addendum.purpose", value: "disable Git auto-maintenance only inside disposable test repositories to eliminate post-assertion cleanup races"},
	{path: "addendum.priorAddendum", value: "WAVE-1-pr-fallback-main-ci-addendum"},
	{path: "addendum.priorAddendumSHA256", value: "8880fd546cf70fff39a02ba2851a74414f892a06741ae41f17a40532044bc130"},
	{path: "addendum.priorAddendumSignatureSHA256", value: "b757aa7fef86ec8f4f6e417f795e02d97a6ee1a0fa5f2edfd39f8d51efee4256"},
	{path: "addendum.blockedDispositionComment", value: "01a03d7d-5f5a-70e2-8ca2-93e1c76e3c02"},
	{path: "addendum.failedRun", value: "32955169341"},
	{path: "addendum.failedAttempts", value: "1,2"},
	{path: "addendum.failureFingerprint", value: "go-test-tempdir-cleanup-git-pack-race"},
	{path: "addendum.priorTransitionTag", value: wave1SuccessorTransitionTag},
	{path: "addendum.priorTransitionTagObject", value: wave1SuccessorTagObject},
	{path: "addendum.priorTransitionTagTarget", value: wave1CIFixtureReviewedHead},
	{path: "addendum.priorTransitionTagTree", value: wave1CIFixtureReviewedTree},
	{path: "addendum.successorTransitionTag", value: wave1FinalTransitionTag},
	{path: "addendum.successorTransitionTagMessage", value: wave1FinalTransitionTagMessage},
	{path: "addendum.mergeMode", value: "squash"},
	{path: "addendum.autonomousMutation", value: "false"},
	{path: "addendum.canonicalWorkMutationAllowed", value: "false"},
	{path: "addendum.githubPolicyMutationAllowed", value: "false"},
	{path: "addendum.secretScannerExceptionAllowed", value: "false"},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.externalStateReadbackRequired", value: "true"},
	{path: "verification.protectedMainUnchangedUntilMerge", value: "true"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: wave1CIFixtureFixNamespace},
	{path: "integrity.detachedSignature", value: "WAVE-1-pr-fallback-fixture-stabilization-addendum.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var wave1CIFixtureFixSequences = map[string][]string{
	"addendum.allowedEffects": {
		"edit-listed-public-paths",
		"create-one-pinned-signer-fixture-stabilization-commit",
		"preserve-v1-and-v2-transition-tags-exactly",
		"create-and-push-one-pinned-signer-v3-transition-tag",
		"update-existing-pull-request",
		"run-one-fresh-public-commit-gate",
		"squash-merge-promptly-after-fresh-qa-and-security-acceptance",
		"append-public-safe-stabilization-intent-and-receipt",
	},
	"addendum.authorizedPaths": {
		wave1CIFixtureFixPath,
		wave1CIFixtureFixSignature,
		"internal/doctrine/grant.go",
		"internal/doctrine/grant_test.go",
	},
	"addendum.requiredProperties": {
		"preserve-b9ca75d-and-v2-tag-as-immutable-history",
		"verify-every-feature-commit-with-the-pinned-signer",
		"disable-git-auto-maintenance-only-in-disposable-test-repositories",
		"require-v3-tag-target-equal-current-reviewed-feature-head-on-pull-request",
		"require-v3-tag-tree-equal-pull-request-and-squash-main-trees",
		"require-exact-base-head-event-and-topology",
		"require-fresh-public-gate-and-immutable-reviews",
	},
	"addendum.prohibitedEffects": {
		"rerun-the-exhausted-failed-workflow",
		"mutate-production-or-developer-git-configuration",
		"move-or-delete-v1-or-v2-transition-tags",
		"retry-direct-main-push",
		"mutate-workflow-ruleset-repository-settings-or-secret-scanner-policy",
		"create-or-move-any-tag-other-than-the-v3-transition-tag",
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

var w001BootstrapGrantScalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3ImplementationBootstrapGrant"},
	{path: "grant.id", value: "W-001-bootstrap"},
	{path: "grant.classification", value: "PUBLIC"},
	{path: "grant.issuedAt", value: "2026-08-26T10:30:00Z"},
	{path: "grant.expiresAt", value: "2026-08-29T10:30:00Z"},
	{path: "grant.repository", value: planningGrantRepository},
	{path: "grant.baseCommit", value: w001BootstrapBase},
	{path: "grant.baseTree", value: w001BootstrapBaseTree},
	{path: "grant.workingBranch", value: w001BootstrapBranch},
	{path: "grant.reviewTag", value: w001BootstrapReviewTag},
	{path: "grant.reviewTagMessage", value: w001BootstrapReviewTagMessage},
	{path: "grant.signerRole", value: "human-bootstrap-authority"},
	{path: "grant.coordinator", value: "delivery-orchestrator"},
	{path: "grant.failureOwnership", value: "foundation"},
	{path: "grant.blockerComment", value: "01a03d98-a2e2-77db-bbd6-76575366454f"},
	{path: "grant.purpose", value: "publish and execute one reviewed atomic W-001 bootstrap claim without asserting a live lease"},
	{path: "grant.bead", value: "M3-W001"},
	{path: "grant.displayId", value: "W-001"},
	{path: "grant.profile", value: "work-authority-engineer"},
	{path: "grant.attemptId", value: "w001-bootstrap-3135f1d1-b0d4-4956-9fc9-1852310bfd77"},
	{path: "grant.replayRef", value: "w001-claim-97a7e81e-e749-4cb0-8a0a-8e269e13e38f"},
	{path: "grant.autonomousMutation", value: "false"},
	{path: "grant.liveLeaseAsserted", value: "false"},
	{path: "grant.productionEffects", value: "false"},
	{path: "expected.beadsProject", value: "e9669a62-5be6-4b94-85f8-c502c29d394a"},
	{path: "expected.nativeStatus", value: "open"},
	{path: "expected.lifecycleState", value: "backlog"},
	{path: "expected.claimState", value: "unclaimed"},
	{path: "expected.assignee", value: "work-authority-engineer"},
	{path: "expected.createdAt", value: "2026-08-26T05:09:05Z"},
	{path: "expected.updatedAt", value: "2026-08-26T06:22:03Z"},
	{path: "expected.metadataSHA256", value: "10c61003cb39518f57905620fcc0c47d29950fe82ae8d98a3111a057fa554dba"},
	{path: "expected.labelsSHA256", value: "be506df06d8c206a3919a71a57e8aaacd2b5e1e233e25bafc2f5f87f306b188c"},
	{path: "expected.dependency", value: "M3-H001"},
	{path: "expected.dependencyType", value: "blocks"},
	{path: "expected.dependencyNativeStatus", value: "closed"},
	{path: "expected.dependencyLifecycleState", value: "done"},
	{path: "expected.dependencyDigest", value: "3ad0bca78b14e4e1fd5544477f131c0a86dd8a4d4e9563d43fa4ae1c202f4100"},
	{path: "expected.lineageDigest", value: "9f3e91b4b642dc740898347c35e8f38abc35cc3ac1be83c81fe122cc308eaced"},
	{path: "postimage.nativeStatus", value: "in_progress"},
	{path: "postimage.lifecycleState", value: "in-progress"},
	{path: "postimage.claimState", value: "claimed"},
	{path: "postimage.removeLabel", value: "backlog"},
	{path: "postimage.addLabel", value: "in-progress"},
	{path: "postimage.generationRef", value: "6e79ff81-a007-42a5-a178-7ce58dbb718b"},
	{path: "postimage.issueIncarnation", value: "e1e8d2d3f80871096a568fb489f49575a42abd37b269df9faf777a09cd689b41"},
	{path: "postimage.issueMutationSequence", value: "1"},
	{path: "postimage.dependencyGraphRevision", value: "1"},
	{path: "postimage.metadataSHA256", value: "3a0bbed5ca93acf77d04eedfad6bcaa16a9701f3e6da3f8669eb9928f6d09139"},
	{path: "postimage.labelsSHA256", value: "3e4e77e20ee7a46dd77c4a9884dee51aa9f0fa9f2445099a0cb457d72cb83bbb"},
	{path: "postimage.metadataBase64", value: "eyJib290c3RyYXBDbGFpbSI6eyJhdHRlbXB0SWQiOiJ3MDAxLWJvb3RzdHJhcC0zMTM1ZjFkMS1iMGQ0LTQ5NTYtOWZjOS0xODUyMzEwYmZkNzciLCJiYXNlQ29tbWl0IjoiMzdiNTViOTEyYjIwNzE1MzQ5YmM1MGUwNTI0Yzg1ZDRiMjJmMTc3MiIsImdyYW50SWQiOiJXLTAwMS1ib290c3RyYXAiLCJpZGVtcG90ZW5jeUtleSI6IncwMDEtY2xhaW0tOTdhN2U4MWUtZTc0OS00Y2IwLThhMGEtOGUyNjllMTNlMzhmIn0sImNvbnRyYWN0UHVibGljYXRpb25SZXF1aXJlZCI6dHJ1ZSwiY29vcmRpbmF0b3IiOiJkZWxpdmVyeS1vcmNoZXN0cmF0b3IiLCJkaXNwbGF5SWQiOiJXLTAwMSIsImV4Y2x1c2l2ZVBhdGhzIjpbImludGVybmFsL2F1dGhvcml0eS8qKiIsImNtZC9tYXJzMy1hdXRob3JpdHkvKioiLCJhcGkvYXV0aG9yaXR5LyoqIiwiZGF0YWJhc2UvYXV0aG9yaXR5LyoqIiwiZGVwbG95L2F1dGhvcml0eS8qKiIsImRvY3MvZXZpZGVuY2UvVy0wMDEtdmFsaWRhdGlvbi5tZCIsImdvLm1vZCIsImdvLnN1bSIsIk1ha2VmaWxlIiwiTk9USUNFIiwiVEhJUkRfUEFSVFlfTk9USUNFUyJdLCJmYWlsdXJlT3duZXJzaGlwIjoiZm91bmRhdGlvbiIsImZlYXR1cmVJZCI6IkYtMDAyIiwiZ29hbElkcyI6WyJHLTAwMSJdLCJsaWZlY3ljbGVTdGF0ZSI6ImluLXByb2dyZXNzIiwicHJvZHVjdERlY2lzaW9uSWRzIjpbIlBELTAwMiJdLCJwdWJsaWNEaXNjbG9zdXJlIjp0cnVlLCJyaXNrIjoiY3JpdGljYWwiLCJzY2VuYXJpb0lkcyI6WyJGLTAwMi1TMSIsIkYtMDAyLVMyIiwiRi0wMDItUzMiLCJGLTAwMi1TNCIsIkYtMDAyLVM1IiwiRi0wMDItUzYiXSwic2NoZW1hVmVyc2lvbiI6MSwidmVyaWZpY2F0aW9uT3JkZXIiOlsicWEiLCJzZWN1cml0eS1yZXZpZXdlciIsImRlbGl2ZXJ5LW9yY2hlc3RyYXRvciJdLCJ3b3JrVHlwZSI6ImVuYWJsZXIiLCJ3b3JrVmVyc2lvbiI6eyJhdXRob3JpdHlHZW5lcmF0aW9uIjoiNmU3OWZmODEtYTAwNy00MmE1LWExNzgtN2NlNThkYmI3MThiIiwiZGVwZW5kZW5jeUdyYXBoUmV2aXNpb24iOjEsImlzc3VlSW5jYXJuYXRpb24iOiJlMWU4ZDJkM2Y4MDg3MTA5NmE1NjhmYjQ4OWY0OTU3NWE0MmFiZDM3YjI2OWRmOWZhZjc3N2EwOWNkNjg5YjQxIiwiaXNzdWVNdXRhdGlvblNlcXVlbmNlIjoxfX0"},
	{path: "toolchain.beadsVersion", value: "1.2.2"},
	{path: "toolchain.beadsSourceCommit", value: beadsCommit},
	{path: "toolchain.beadsBinarySHA256", value: "6cc5cf1d3fea5774606af82410ac05e35b78ad5f404f1da5be33c93ff087cffb"},
	{path: "toolchain.doltSourceCommit", value: doltCommit},
	{path: "toolchain.doltModule", value: "github.com/dolthub/dolt/go v0.40.5-0.20260605230755-1bf533220ab0"},
	{path: "toolchain.doltModuleSHA256", value: "oPg5f5bYFy5x7Ws2qtVG7wiva96cIh9SFg7nrC4n7QA="},
	{path: "toolchain.goVersion", value: "go1.26.2"},
	{path: "toolchain.goOS", value: "darwin"},
	{path: "toolchain.goArch", value: "arm64"},
	{path: "toolchain.goBinarySHA256", value: "005640c7ff93028cb704283b0f737f2db3faf8b51b2561170c769b83905da646"},
	{path: "toolchain.cgoEnabled", value: "true"},
	{path: "toolchain.icuFormula", value: "icu4c@78 78.2"},
	{path: "toolchain.doltTestImage", value: "dolthub/dolt-sql-server@sha256:6b651663c5024d98a98a4db7226a5e85f90a9344c78fee85617c0fb4a30c6e64"},
	{path: "toolchain.ryukDisabled", value: "true"},
	{path: "toolchain.patchPath", value: "internal/authority/bootstrap/beads-v1.2.2-atomic-claim.patch"},
	{path: "toolchain.patchSHA256", value: "50128252828352366ced6560371468a5746c2603ef89ea746a33be8994ffceb6"},
	{path: "toolchain.patchedBinarySHA256", value: "949e1d535e19ecb39e974b90b7321ef1f7f7d6b77c3958d72edb07e78d9def5a"},
	{path: "toolchain.helperCommandPath", value: "cmd/mars3-authority/main.go"},
	{path: "toolchain.helperCommandSHA256", value: "d8ae9fcf5b04902fa3f2ece3369688ca7abf1e55f0cd4f57a611006a861979ea"},
	{path: "toolchain.helperLibraryPath", value: "internal/authority/bootstrap/bootstrap.go"},
	{path: "toolchain.helperLibrarySHA256", value: "d039c787f73e98f059937242e068d76c12753cc9accedc025bf619e1fa63c0fd"},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.protectedMainRequiredBeforeClaim", value: "true"},
	{path: "verification.externalStateReadbackRequired", value: "true"},
	{path: "verification.disposableAtomicityConformanceRequired", value: "true"},
	{path: "verification.postReviewExecutionAuthorizationRequired", value: "true"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: w001BootstrapGrantNamespace},
	{path: "integrity.detachedSignature", value: "W-001-bootstrap.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var w001BootstrapGrantSequences = map[string][]string{
	"grant.allowedEffects": {
		"edit-exact-preclaim-paths",
		"build-patched-Beads-from-the-exact-signed-source-and-patch",
		"run-disposable-atomic-claim-conformance",
		"create-pinned-signer-feature-commits-and-one-signed-review-tag",
		"push-one-feature-branch-and-review-tag-and-open-one-ready-PR",
		"run-public-commit-gate-and-obtain-independent-QA-and-Security-review",
		"squash-merge-promptly-with-reviewed-tree-equality",
		"append-public-safe-authority-intents-receipts-and-dispositions",
		"execute-one-expected-preimage-CAS-claim-on-accepted-main",
		"append-one-public-safe-canonical-claim-receipt",
	},
	"grant.preclaimPaths": {
		w001BootstrapGrantPath,
		w001BootstrapGrantSignature,
		"cmd/mars3-authority/main.go",
		"internal/authority/bootstrap/bootstrap.go",
		"internal/authority/bootstrap/bootstrap_test.go",
		"internal/authority/bootstrap/beads-v1.2.2-atomic-claim.patch",
		"internal/doctrine/grant.go",
		"internal/doctrine/grant_test.go",
		"docs/design-docs/ADR-001-git-beads-authority.md",
		"docs/product-specs/work-authority.md",
		"docs/features/F-002-work-authority.md",
		"docs/evidence/W-001-bootstrap-transition.md",
		"NOTICE",
		"THIRD_PARTY_NOTICES",
	},
	"grant.implementationPaths": {
		"internal/authority/**", "cmd/mars3-authority/**", "api/authority/**",
		"database/authority/**", "deploy/authority/**", "docs/evidence/W-001-validation.md",
		"go.mod", "go.sum", "Makefile", "NOTICE", "THIRD_PARTY_NOTICES",
	},
	"grant.prohibitedEffects": {
		"execute-claim-before-helper-merge-protected-main-success-QA-and-Security-acceptance",
		"execute-claim-without-a-separate-signed-short-lived-post-review-authorization",
		"use-raw-SQL-hidden-code-or-a-two-transaction-claim",
		"mutate-another-Bead-dependency-or-authority-project",
		"issue-or-assert-a-live-lease-before-verified-gateway-issuance",
		"edit-outside-the-current-signed-phase-paths",
		"move-delete-or-reuse-the-review-tag",
		"mutate-workflow-ruleset-repository-settings-or-secret-scanner-policy",
		"production-or-destructive-effect",
		"reconcile-plan-or-manifest-without-a-separate-signed-postclaim-grant",
		"begin-gateway-implementation-under-this-bootstrap-claim-grant",
		"autonomous-mutation", "trust-escalation",
		"credentials-provider-session-customer-data-or-raw-payloads",
	},
	"verification.order": {"qa", "security-reviewer", "delivery-orchestrator"},
}

var w001PostclaimGrantScalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3W001PostclaimReconciliationGrant"},
	{path: "grant.id", value: "W-001-postclaim-reconciliation"},
	{path: "grant.classification", value: "PUBLIC"},
	{path: "grant.issuedAt", value: "2026-08-26T20:33:58Z"},
	{path: "grant.expiresAt", value: "2026-08-28T20:33:58Z"},
	{path: "grant.repository", value: planningGrantRepository},
	{path: "grant.baseCommit", value: w001PostclaimBase},
	{path: "grant.baseTree", value: w001PostclaimBaseTree},
	{path: "grant.workingBranch", value: w001PostclaimBranch},
	{path: "grant.reviewTag", value: w001PostclaimReviewTag},
	{path: "grant.reviewTagMessage", value: w001PostclaimReviewTagMessage},
	{path: "grant.signerRole", value: "human-bootstrap-authority"},
	{path: "grant.coordinator", value: "delivery-orchestrator"},
	{path: "grant.failureOwnership", value: "foundation"},
	{path: "grant.purpose", value: "reconcile the verified canonical W-001 claim into Git without a lease or implementation authority"},
	{path: "grant.bead", value: "M3-W001"},
	{path: "grant.displayId", value: "W-001"},
	{path: "grant.priorGrant", value: "W-001-bootstrap"},
	{path: "grant.priorGrantSHA256", value: "37b4b0df773fa6bace2a06af34ff546e12c9e2f42eb8bb5865f1b9405727e34d"},
	{path: "grant.priorGrantSignatureSHA256", value: "eeba6098de123f4d9d47c33d4349015775591511784d6c29b709ed1d45927e15"},
	{path: "grant.autonomousMutation", value: "false"},
	{path: "grant.liveLeaseAsserted", value: "false"},
	{path: "grant.implementationAllowed", value: "false"},
	{path: "claim.attemptId", value: "w001-bootstrap-3135f1d1-b0d4-4956-9fc9-1852310bfd77"},
	{path: "claim.replayReferenceSHA256", value: "321df42aa6cd67c3ae42b687b16927d81fc19b42e124819bd266b84b22c2d1a0"},
	{path: "claim.executionAuthorizationPayloadSHA256", value: "2a2428537a5e42ed6f69afaa42771b9b404705e5593b554129c6ce45f5aec297"},
	{path: "claim.executionAuthorizationSignatureSHA256", value: "d7bc5a1c824bde6ee72d755660afccaad569840b2db29971bf3b8b087540ce5c"},
	{path: "claim.receiptSHA256", value: "04cef4e421a34e0908d392fc794181db3ddb754a134e34599fa41a520c78d126"},
	{path: "claim.previousDoltCommit", value: "mm6m0b4655k5gpt5eren5cfkgvjtabsm"},
	{path: "claim.doltCommit", value: "67hmen0cmq0he08n7ujlqpcsmmi94fhb"},
	{path: "claim.nativeStatus", value: "in_progress"},
	{path: "claim.lifecycleState", value: "in-progress"},
	{path: "claim.claimed", value: "true"},
	{path: "claim.updatedAt", value: "2026-08-26T20:23:37Z"},
	{path: "claim.metadataSHA256", value: "3a0bbed5ca93acf77d04eedfad6bcaa16a9701f3e6da3f8669eb9928f6d09139"},
	{path: "claim.labelsSHA256", value: "3e4e77e20ee7a46dd77c4a9884dee51aa9f0fa9f2445099a0cb457d72cb83bbb"},
	{path: "claim.workVersionGeneration", value: "6e79ff81-a007-42a5-a178-7ce58dbb718b"},
	{path: "claim.workVersionIncarnation", value: "e1e8d2d3f80871096a568fb489f49575a42abd37b269df9faf777a09cd689b41"},
	{path: "claim.issueMutationSequence", value: "1"},
	{path: "claim.dependencyGraphRevision", value: "1"},
	{path: "claim.liveLeaseAsserted", value: "false"},
	{path: "publication.helperFeatureCommit", value: "663d19bf190f9e3bd27edc96ee08acaa6778c853"},
	{path: "publication.helperFeatureTree", value: w001PostclaimBaseTree},
	{path: "publication.helperReviewTag", value: w001BootstrapReviewTag},
	{path: "publication.helperReviewTagObject", value: "6409b5daecb472b415dec60b96b12bfffa3a4cd0"},
	{path: "publication.helperPullRequest", value: "6"},
	{path: "publication.helperPullRequestRun", value: "33005091777"},
	{path: "publication.mergedCommit", value: w001PostclaimBase},
	{path: "publication.mergedTree", value: w001PostclaimBaseTree},
	{path: "publication.protectedMainRun", value: "33006386000"},
	{path: "publication.qaReviewedCommit", value: "663d19bf190f9e3bd27edc96ee08acaa6778c853"},
	{path: "publication.qaDisposition", value: "accepted"},
	{path: "publication.securityReviewedCommit", value: "663d19bf190f9e3bd27edc96ee08acaa6778c853"},
	{path: "publication.securityDisposition", value: "accepted"},
	{path: "postimage.manifestSHA256", value: "ee37f6262e714aa1f853fe7b81ce9b50a1d299482dde905262e80cbb133d13e6"},
	{path: "postimage.activePlanSHA256", value: "d5ac48639cd8bd98ce57e1f0f91ddd4c358498eddb45c8255c0ca67ab56abbbc"},
	{path: "postimage.evidenceSHA256", value: "93dd425ef42e6c929a97d20e08b9d75c339b45303975162cc863dad3764dc1a0"},
	{path: "postimage.planPhase", value: "delivery"},
	{path: "postimage.currentBeadState", value: "in-progress"},
	{path: "postimage.currentBeadClaimed", value: "true"},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.protectedMainRequired", value: "true"},
	{path: "verification.externalBeadsReadbackRequired", value: "true"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: w001PostclaimGrantNamespace},
	{path: "integrity.detachedSignature", value: "W-001-postclaim-reconciliation.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var w001PostclaimGrantSequences = map[string][]string{
	"grant.allowedEffects": {
		"create-and-verify-this-signed-postclaim-grant",
		"edit-the-exact-authorized-Git-paths",
		"bind-the-exact-canonical-claim-receipt-digest-and-bounded-public-summary",
		"reconcile-the-active-plan-to-delivery-and-W-001-in-progress",
		"reconcile-the-manifest-to-claimed-in-progress-delivery",
		"create-pinned-signer-commits-and-one-signed-review-tag",
		"push-one-review-branch-and-tag-and-open-one-ready-PR",
		"run-public-gates-and-obtain-independent-QA-and-Security-review",
		"squash-merge-promptly-with-reviewed-tree-equality",
	},
	"grant.authorizedPaths": {
		w001PostclaimGrantPath,
		w001PostclaimGrantSignature,
		".harness/manifest.yaml",
		canonicalActivePlan,
		"docs/evidence/W-001-bootstrap-transition.md",
		"internal/doctrine/grant.go",
		"internal/doctrine/grant_test.go",
	},
	"grant.requiredProperties": {
		"canonical-Beads-state-remains-the-work-authority",
		"claim-receipt-binds-the-signed-execution-authorization-and-accepted-helper",
		"plan-and-manifest-postimages-exactly-match-the-signed-digests",
		"every-commit-and-current-change-stays-inside-the-signed-path-set",
		"every-feature-commit-and-review-tag-use-the-pinned-signer",
		"reviewed-feature-tree-equals-the-protected-main-squash-tree",
		"no-live-lease-or-implementation-authority-is-created",
	},
	"grant.prohibitedEffects": {
		"mutate-any-Bead-Dolt-row-dependency-label-comment-or-history",
		"issue-assert-renew-release-or-revoke-a-live-lease",
		"edit-gateway-runtime-platform-or-product-implementation",
		"change-feature-contract-product-decision-goal-or-scenario-schedule",
		"move-delete-or-reuse-any-existing-review-tag",
		"mutate-workflow-ruleset-repository-settings-or-secret-scanner-policy",
		"production-or-destructive-effect",
		"autonomous-mutation",
		"trust-escalation",
		"credentials-provider-session-customer-data-or-raw-payloads",
	},
	"verification.order": {"qa", "security-reviewer", "delivery-orchestrator"},
}

var w001PostclaimCIFixScalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3W001PostclaimCIStabilizationAddendum"},
	{path: "addendum.id", value: "W-001-postclaim-ci-stabilization-v2"},
	{path: "addendum.classification", value: "PUBLIC"},
	{path: "addendum.issuedAt", value: "2026-08-26T20:53:07Z"},
	{path: "addendum.expiresAt", value: "2026-08-28T20:53:07Z"},
	{path: "addendum.repository", value: planningGrantRepository},
	{path: "addendum.baseCommit", value: w001PostclaimCIFixBase},
	{path: "addendum.baseTree", value: w001PostclaimCIFixBaseTree},
	{path: "addendum.workingBranch", value: w001PostclaimBranch},
	{path: "addendum.reviewTag", value: w001PostclaimCIFixReviewTag},
	{path: "addendum.reviewTagMessage", value: w001PostclaimCIFixTagMessage},
	{path: "addendum.signerRole", value: "human-bootstrap-authority"},
	{path: "addendum.coordinator", value: "delivery-orchestrator"},
	{path: "addendum.failureOwnership", value: "foundation"},
	{path: "addendum.purpose", value: "isolate repository-local Git fixtures from inherited GitHub runner identity without weakening real checkout admission"},
	{path: "addendum.priorGrant", value: "W-001-postclaim-reconciliation"},
	{path: "addendum.priorGrantSHA256", value: "7fb4b9e6aa65661bef80887b95225973e556fe8dbc4cf77fb66aaa0f10da5dfe"},
	{path: "addendum.priorGrantSignatureSHA256", value: "3afc9d46c8c9ea436246d3bd33c2228e0afc1c1c27ceb6ff44114c761e029edd"},
	{path: "addendum.priorReviewTag", value: w001PostclaimReviewTag},
	{path: "addendum.priorReviewTagObject", value: w001PostclaimV1TagObject},
	{path: "addendum.priorReviewTagTarget", value: w001PostclaimCIFixBase},
	{path: "addendum.failedRun", value: "33012491197"},
	{path: "addendum.failedJob", value: "98322099024"},
	{path: "addendum.failureFingerprint", value: "go-test-fixtures-inherited-github-actions-environment"},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.protectedMainRequired", value: "true"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: w001PostclaimCIFixNamespace},
	{path: "integrity.detachedSignature", value: "W-001-postclaim-ci-stabilization-v2.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var w001PostclaimCIFixSequences = map[string][]string{
	"addendum.allowedEffects": {
		"create-and-verify-this-signed-CI-stabilization-addendum",
		"clear-GitHub-runner-environment-only-inside-repository-local-Git-fixtures",
		"add-the-exact-regression-for-the-preserved-CI-failure",
		"preserve-production-GitHub-checkout-identity-enforcement",
		"create-pinned-signer-correction-commits-and-one-signed-v2-review-tag",
		"push-the-existing-review-branch-and-v2-tag-and-rerun-the-ready-PR",
		"obtain-independent-QA-and-Security-review-before-squash-merge",
	},
	"addendum.authorizedPaths": {
		w001PostclaimCIFixPath,
		w001PostclaimCIFixSignature,
		"internal/doctrine/grant.go",
		"internal/doctrine/grant_test.go",
	},
	"addendum.requiredProperties": {
		"v1-review-tag-object-and-target-remain-immutable",
		"v1-failed-run-remains-public-and-unchanged",
		"fixture-environment-isolation-is-test-only",
		"real-GitHub-checkout-admission-still-requires-canonical-runner-identity",
		"every-correction-commit-and-current-change-stays-inside-the-signed-path-set",
		"every-correction-commit-and-v2-review-tag-use-the-pinned-signer",
		"reviewed-v2-tree-equals-the-protected-main-squash-tree",
		"no-Beads-lease-implementation-production-or-policy-effect-is-created",
	},
	"addendum.prohibitedEffects": {
		"move-delete-or-reuse-the-v1-review-tag",
		"edit-plan-manifest-evidence-feature-or-product-contracts",
		"mutate-any-Bead-Dolt-row-dependency-label-comment-or-history",
		"issue-assert-renew-release-or-revoke-a-live-lease",
		"edit-gateway-runtime-platform-or-product-implementation",
		"weaken-production-GitHub-runner-event-topology-workflow-or-tag-validation",
		"mutate-workflow-ruleset-repository-settings-or-secret-scanner-policy",
		"production-or-destructive-effect",
		"autonomous-mutation",
		"trust-escalation",
		"credentials-provider-session-customer-data-or-raw-payloads",
	},
	"verification.order": {"qa", "security-reviewer", "delivery-orchestrator"},
}

var w001PostclaimSecurityFixScalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3W001PostclaimSecurityCorrectionGrant"},
	{path: "grant.id", value: "W-001-postclaim-security-correction-v3"},
	{path: "grant.classification", value: "PUBLIC"},
	{path: "grant.issuedAt", value: "2026-08-26T21:21:01Z"},
	{path: "grant.expiresAt", value: "2026-08-28T21:21:01Z"},
	{path: "grant.repository", value: planningGrantRepository},
	{path: "grant.baseCommit", value: w001PostclaimSecurityFixBase},
	{path: "grant.baseTree", value: w001PostclaimSecurityFixTree},
	{path: "grant.workingBranch", value: w001PostclaimBranch},
	{path: "grant.reviewTag", value: w001PostclaimSecurityFixTag},
	{path: "grant.reviewTagMessage", value: w001PostclaimSecurityFixTagMsg},
	{path: "grant.signerRole", value: "human-bootstrap-authority"},
	{path: "grant.coordinator", value: "delivery-orchestrator"},
	{path: "grant.failureOwnership", value: "foundation"},
	{path: "grant.purpose", value: "correct the superseded W-001 helper Security disposition and bind effective direct embedded M3 authority without mutating canonical work"},
	{path: "grant.bead", value: "M3-W001"},
	{path: "grant.displayId", value: "W-001"},
	{path: "grant.priorAddendum", value: "W-001-postclaim-ci-stabilization-v2"},
	{path: "grant.priorAddendumSHA256", value: "526f73a4200fc294bb8cf5ad82bb4e12f6259d1490a8c98316da1ddb44edabef"},
	{path: "grant.priorAddendumSignatureSHA256", value: "4ad7b094a2328eedd0f24ae671e28a5797026b7bb8f14d89cab4829cc70d244d"},
	{path: "grant.priorReviewTag", value: w001PostclaimCIFixReviewTag},
	{path: "grant.priorReviewTagObject", value: w001PostclaimV2TagObject},
	{path: "grant.priorReviewTagTarget", value: w001PostclaimSecurityFixBase},
	{path: "grant.successfulRun", value: "33013185662"},
	{path: "grant.successfulJob", value: "98324496309"},
	{path: "grant.autonomousMutation", value: "false"},
	{path: "grant.liveLeaseAsserted", value: "false"},
	{path: "grant.implementationAllowed", value: "false"},
	{path: "grant.canonicalWorkMutationAllowed", value: "false"},
	{path: "finding.helperFeatureCommit", value: "663d19bf190f9e3bd27edc96ee08acaa6778c853"},
	{path: "finding.helperFeatureTree", value: w001PostclaimBaseTree},
	{path: "finding.affectedPostclaimHead", value: w001PostclaimSecurityFixBase},
	{path: "finding.affectedPostclaimTree", value: w001PostclaimSecurityFixTree},
	{path: "finding.priorSecurityDisposition", value: "accepted"},
	{path: "finding.currentSecurityDisposition", value: "changes-requested"},
	{path: "finding.priorSecurityAcceptanceStatus", value: "superseded"},
	{path: "finding.failureFingerprint", value: "bootstrap-effective-database-selector-splice"},
	{path: "finding.failureClass", value: "foundation-owned"},
	{path: "finding.findingScope", value: "helper-admission-and-transaction-authority"},
	{path: "canonicalEffect.bead", value: "M3-W001"},
	{path: "canonicalEffect.nativeStatus", value: "in_progress"},
	{path: "canonicalEffect.lifecycleState", value: "in-progress"},
	{path: "canonicalEffect.assignee", value: "work-authority-engineer"},
	{path: "canonicalEffect.updatedAt", value: "2026-08-26T20:23:37Z"},
	{path: "canonicalEffect.workVersionGeneration", value: "6e79ff81-a007-42a5-a178-7ce58dbb718b"},
	{path: "canonicalEffect.workVersionIncarnation", value: "e1e8d2d3f80871096a568fb489f49575a42abd37b269df9faf777a09cd689b41"},
	{path: "canonicalEffect.issueMutationSequence", value: "1"},
	{path: "canonicalEffect.dependencyGraphRevision", value: "1"},
	{path: "canonicalEffect.claimReceiptSHA256", value: "04cef4e421a34e0908d392fc794181db3ddb754a134e34599fa41a520c78d126"},
	{path: "canonicalEffect.doltCommit", value: "67hmen0cmq0he08n7ujlqpcsmmi94fhb"},
	{path: "canonicalEffect.independentlyReadBackFromM3", value: "true"},
	{path: "canonicalEffect.alternateDatabaseUseObservedInCanonicalEffect", value: "false"},
	{path: "canonicalEffect.liveLeaseAsserted", value: "false"},
	{path: "materials.helperLibraryPath", value: "internal/authority/bootstrap/bootstrap.go"},
	{path: "materials.helperLibrarySHA256", value: w001PostclaimSecurityHelperSHA},
	{path: "materials.helperTestPath", value: "internal/authority/bootstrap/bootstrap_test.go"},
	{path: "materials.helperTestSHA256", value: w001PostclaimSecurityTestSHA},
	{path: "materials.basePatchPath", value: "internal/authority/bootstrap/beads-v1.2.2-atomic-claim.patch"},
	{path: "materials.basePatchSHA256", value: w001PostclaimSecurityBasePatchSHA},
	{path: "materials.securityPatchPath", value: w001PostclaimSecurityPatchPath},
	{path: "materials.securityPatchSHA256", value: w001PostclaimSecurityPatchSHA},
	{path: "materials.patchedBinarySHA256", value: w001PostclaimSecurityBinarySHA},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.protectedMainRequired", value: "true"},
	{path: "verification.externalBeadsReadbackRequired", value: "true"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: w001PostclaimSecurityFixNS},
	{path: "integrity.detachedSignature", value: "W-001-postclaim-security-correction-v3.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var w001PostclaimSecurityFixSequences = map[string][]string{
	"grant.allowedEffects": {
		"create-and-verify-this-signed-Security-correction-grant",
		"record-the-additive-Security-changes-requested-and-supersession-evidence",
		"reject-or-bind-Beads-selector-files-and-force-effective-direct-embedded-M3",
		"require-same-transaction-embedded-M3-authority-before-any-bootstrap-read-or-write",
		"remove-server-transaction-bootstrap-claim-capability",
		"add-alternate-database-server-backend-and-configuration-race-regressions",
		"edit-only-the-exact-authorized-Git-paths",
		"preserve-the-observed-canonical-M3-postimage-as-effect-evidence-not-helper-acceptance",
		"create-pinned-signer-correction-commits-and-one-signed-v3-review-tag",
		"push-the-existing-review-branch-and-v3-tag-and-rerun-the-ready-PR",
		"obtain-fresh-independent-QA-and-Security-review-before-squash-merge",
	},
	"grant.authorizedPaths": {
		w001PostclaimSecurityFixPath,
		w001PostclaimSecurityFixSig,
		"docs/evidence/W-001-validation.md",
		"internal/authority/bootstrap/bootstrap.go",
		"internal/authority/bootstrap/bootstrap_test.go",
		"internal/authority/bootstrap/beads-v1.2.2-atomic-claim.patch",
		w001PostclaimSecurityPatchPath,
		"internal/doctrine/grant.go",
		"internal/doctrine/grant_test.go",
	},
	"grant.requiredProperties": {
		"prior-v1-and-v2-grants-signatures-tags-runs-and-history-remain-immutable",
		"prior-helper-Security-acceptance-is-explicitly-superseded",
		"actual-canonical-M3-postimage-remains-valid-effect-evidence-only",
		"helper-selection-and-transaction-authority-both-require-direct-embedded-M3",
		"selector-file-or-effective-database-change-fails-before-authority-state-mutation",
		"server-backed-transactions-cannot-implement-bootstrap-claim-authority",
		"every-correction-commit-and-current-change-stays-inside-the-signed-path-set",
		"every-correction-commit-and-v3-review-tag-use-the-pinned-signer",
		"reviewed-v3-tree-equals-the-protected-main-squash-tree",
		"no-Beads-lease-implementation-production-or-policy-effect-is-created",
	},
	"grant.prohibitedEffects": {
		"mutate-any-Bead-Dolt-row-dependency-label-comment-or-history",
		"execute-or-replay-the-canonical-W-001-bootstrap-claim",
		"issue-assert-renew-release-or-revoke-a-live-lease",
		"begin-gateway-runtime-platform-or-product-implementation",
		"edit-plan-manifest-feature-product-goal-or-scenario-contracts",
		"move-delete-or-reuse-any-existing-review-tag",
		"mutate-workflow-ruleset-repository-settings-or-secret-scanner-policy",
		"production-or-destructive-effect",
		"autonomous-mutation",
		"trust-escalation",
		"credentials-provider-session-customer-data-or-raw-payloads",
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
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(wave1DirectMainGrantPath))); err == nil {
		checkWave1DirectMainTransitionGrant(root, findings)
	} else if !os.IsNotExist(err) {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_state", "direct-main transition grant state cannot be established")
	}
}

func checkWave1DirectMainTransitionGrant(root string, findings *[]Finding) {
	data, err := readRepoFile(root, wave1DirectMainGrantPath)
	if err != nil {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_missing", "signed direct-main transition grant is required")
		return
	}
	document := parseStrictGrant(data, wave1DirectMainGrantScalars, wave1DirectMainGrantSequences, []string{"grant", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_schema", "%s", message)
	}
	for _, expected := range wave1DirectMainGrantScalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_value", "%s does not match the signed transition contract", expected.path)
		}
	}
	for path, expected := range wave1DirectMainGrantSequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_sequence", "%s must equal the exact ordered transition contract", path)
		}
	}
	for _, section := range []string{"grant", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_schema", "%s mapping must occur exactly once", section)
		}
	}

	signature, signatureErr := readRepoFile(root, wave1DirectMainGrantSignature)
	if signatureErr != nil {
		addFinding(findings, wave1DirectMainGrantSignature, "public.direct_main_transition_signature_missing", "detached direct-main transition signature is required")
	}
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.direct_main_transition_key", "direct-main transition must use the independently pinned genesis key")
	} else if signatureErr == nil {
		if err := verifySSHSig(data, signature, publicKey, wave1DirectMainGrantNamespace); err != nil {
			addFinding(findings, wave1DirectMainGrantSignature, "public.direct_main_transition_signature", "%v", err)
		}
	}
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(wave1PRFallbackPath))); err == nil {
		checkWave1PRFallbackAddendum(root, findings)
	} else if !os.IsNotExist(err) {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_state", "PR-fallback addendum state cannot be established")
	}
}

func checkWave1PRFallbackAddendum(root string, findings *[]Finding) {
	data, err := readRepoFile(root, wave1PRFallbackPath)
	if err != nil {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_missing", "signed PR-fallback addendum is required")
		return
	}
	document := parseStrictGrant(data, wave1PRFallbackScalars, wave1PRFallbackSequences, []string{"addendum", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_schema", "%s", message)
	}
	for _, expected := range wave1PRFallbackScalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_value", "%s does not match the signed fallback contract", expected.path)
		}
	}
	for path, expected := range wave1PRFallbackSequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_sequence", "%s must equal the exact ordered fallback contract", path)
		}
	}
	for _, section := range []string{"addendum", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_schema", "%s mapping must occur exactly once", section)
		}
	}
	signature, signatureErr := readRepoFile(root, wave1PRFallbackSignature)
	if signatureErr != nil {
		addFinding(findings, wave1PRFallbackSignature, "public.pr_fallback_signature_missing", "detached PR-fallback signature is required")
	}
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.pr_fallback_key", "PR fallback must use the independently pinned genesis key")
	} else if signatureErr == nil {
		if err := verifySSHSig(data, signature, publicKey, wave1PRFallbackNamespace); err != nil {
			addFinding(findings, wave1PRFallbackSignature, "public.pr_fallback_signature", "%v", err)
		}
	}
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(wave1MainCIFixPath))); err == nil {
		checkWave1MainCIFixAddendum(root, findings)
	} else if !os.IsNotExist(err) {
		addFinding(findings, wave1MainCIFixPath, "public.pr_fallback_main_ci_state", "main-CI correction addendum state cannot be established")
	}
}

func checkWave1MainCIFixAddendum(root string, findings *[]Finding) {
	data, err := readRepoFile(root, wave1MainCIFixPath)
	if err != nil {
		addFinding(findings, wave1MainCIFixPath, "public.pr_fallback_main_ci_missing", "signed main-CI correction addendum is required")
		return
	}
	document := parseStrictGrant(data, wave1MainCIFixScalars, wave1MainCIFixSequences, []string{"addendum", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, wave1MainCIFixPath, "public.pr_fallback_main_ci_schema", "%s", message)
	}
	for _, expected := range wave1MainCIFixScalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, wave1MainCIFixPath, "public.pr_fallback_main_ci_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, wave1MainCIFixPath, "public.pr_fallback_main_ci_value", "%s does not match the signed main-CI correction contract", expected.path)
		}
	}
	for path, expected := range wave1MainCIFixSequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, wave1MainCIFixPath, "public.pr_fallback_main_ci_sequence", "%s must equal the exact ordered main-CI correction contract", path)
		}
	}
	for _, section := range []string{"addendum", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, wave1MainCIFixPath, "public.pr_fallback_main_ci_schema", "%s mapping must occur exactly once", section)
		}
	}
	signature, signatureErr := readRepoFile(root, wave1MainCIFixSignature)
	if signatureErr != nil {
		addFinding(findings, wave1MainCIFixSignature, "public.pr_fallback_main_ci_signature_missing", "detached main-CI correction signature is required")
	}
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.pr_fallback_main_ci_key", "main-CI correction must use the independently pinned genesis key")
	} else if signatureErr == nil {
		if err := verifySSHSig(data, signature, publicKey, wave1MainCIFixNamespace); err != nil {
			addFinding(findings, wave1MainCIFixSignature, "public.pr_fallback_main_ci_signature", "%v", err)
		}
	}
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(wave1CIFixtureFixPath))); err == nil {
		checkWave1CIFixtureFixAddendum(root, findings)
	} else if !os.IsNotExist(err) {
		addFinding(findings, wave1CIFixtureFixPath, "public.pr_fallback_fixture_state", "fixture-stabilization addendum state cannot be established")
	}
}

func checkWave1CIFixtureFixAddendum(root string, findings *[]Finding) {
	data, err := readRepoFile(root, wave1CIFixtureFixPath)
	if err != nil {
		addFinding(findings, wave1CIFixtureFixPath, "public.pr_fallback_fixture_missing", "signed fixture-stabilization addendum is required")
		return
	}
	document := parseStrictGrant(data, wave1CIFixtureFixScalars, wave1CIFixtureFixSequences, []string{"addendum", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, wave1CIFixtureFixPath, "public.pr_fallback_fixture_schema", "%s", message)
	}
	for _, expected := range wave1CIFixtureFixScalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, wave1CIFixtureFixPath, "public.pr_fallback_fixture_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, wave1CIFixtureFixPath, "public.pr_fallback_fixture_value", "%s does not match the signed fixture-stabilization contract", expected.path)
		}
	}
	for path, expected := range wave1CIFixtureFixSequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, wave1CIFixtureFixPath, "public.pr_fallback_fixture_sequence", "%s must equal the exact ordered fixture-stabilization contract", path)
		}
	}
	for _, section := range []string{"addendum", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, wave1CIFixtureFixPath, "public.pr_fallback_fixture_schema", "%s mapping must occur exactly once", section)
		}
	}
	signature, signatureErr := readRepoFile(root, wave1CIFixtureFixSignature)
	if signatureErr != nil {
		addFinding(findings, wave1CIFixtureFixSignature, "public.pr_fallback_fixture_signature_missing", "detached fixture-stabilization signature is required")
	}
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.pr_fallback_fixture_key", "fixture stabilization must use the independently pinned genesis key")
	} else if signatureErr == nil {
		if err := verifySSHSig(data, signature, publicKey, wave1CIFixtureFixNamespace); err != nil {
			addFinding(findings, wave1CIFixtureFixSignature, "public.pr_fallback_fixture_signature", "%v", err)
		}
	}
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001BootstrapGrantPath))); err == nil {
		checkW001BootstrapGrant(root, findings)
	} else if !os.IsNotExist(err) {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_state", "W-001 bootstrap grant state cannot be established")
	}
}

func checkW001BootstrapGrant(root string, findings *[]Finding) {
	data, err := readRepoFile(root, w001BootstrapGrantPath)
	if err != nil {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_missing", "signed W-001 bootstrap grant is required")
		return
	}
	document := parseStrictGrant(data, w001BootstrapGrantScalars, w001BootstrapGrantSequences,
		[]string{"grant", "expected", "postimage", "toolchain", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_schema", "%s", message)
	}
	for _, expected := range w001BootstrapGrantScalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_value", "%s does not match the signed W-001 bootstrap contract", expected.path)
		}
	}
	for path, expected := range w001BootstrapGrantSequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_sequence", "%s must equal the exact ordered W-001 bootstrap contract", path)
		}
	}
	for _, section := range []string{"grant", "expected", "postimage", "toolchain", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_schema", "%s mapping must occur exactly once", section)
		}
	}

	issuedAt, issueErr := time.Parse(time.RFC3339, scalarValue(document, "grant.issuedAt"))
	expiresAt, expiryErr := time.Parse(time.RFC3339, scalarValue(document, "grant.expiresAt"))
	if issueErr != nil || expiryErr != nil || !expiresAt.After(issuedAt) || expiresAt.Sub(issuedAt) > 72*time.Hour {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_expiry", "bootstrap grant must use one RFC3339 interval no longer than 72 hours")
	}
	incarnationInput, _ := json.Marshal(map[string]string{
		"created_at": scalarValue(document, "expected.createdAt"),
		"id":         scalarValue(document, "grant.bead"),
	})
	if fileSHA256(incarnationInput) != scalarValue(document, "postimage.issueIncarnation") {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_incarnation", "issue incarnation must be the signed digest of the canonical issue identity")
	}

	signature, signatureErr := readRepoFile(root, w001BootstrapGrantSignature)
	if signatureErr != nil {
		addFinding(findings, w001BootstrapGrantSignature, "public.w001_bootstrap_signature_missing", "detached W-001 bootstrap signature is required")
	}
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.w001_bootstrap_key", "W-001 bootstrap grant must use the independently pinned genesis key")
	} else if signatureErr == nil {
		if err := verifySSHSig(data, signature, publicKey, w001BootstrapGrantNamespace); err != nil {
			addFinding(findings, w001BootstrapGrantSignature, "public.w001_bootstrap_signature", "%v", err)
		}
	}

	securityCorrectionActive := false
	if _, stateErr := os.Lstat(filepath.Join(root, filepath.FromSlash(w001PostclaimSecurityFixPath))); stateErr == nil {
		securityCorrectionActive = true
	} else if !os.IsNotExist(stateErr) {
		addFinding(findings, w001PostclaimSecurityFixPath, "public.w001_postclaim_security_state", "postclaim Security-correction state cannot be established")
	}
	for _, binding := range []struct {
		pathField string
		hashField string
		code      string
	}{
		{"toolchain.patchPath", "toolchain.patchSHA256", "public.w001_bootstrap_patch_digest"},
		{"toolchain.helperCommandPath", "toolchain.helperCommandSHA256", "public.w001_bootstrap_helper_digest"},
		{"toolchain.helperLibraryPath", "toolchain.helperLibrarySHA256", "public.w001_bootstrap_helper_digest"},
	} {
		path := scalarValue(document, binding.pathField)
		expectedDigest := scalarValue(document, binding.hashField)
		if securityCorrectionActive && binding.pathField == "toolchain.helperLibraryPath" {
			expectedDigest = w001PostclaimSecurityHelperSHA
		}
		content, err := readRepoFile(root, path)
		if err != nil || !sha256Pattern.MatchString(expectedDigest) || fileSHA256(content) != expectedDigest {
			addFinding(findings, path, binding.code, "bootstrap helper material must match its exact signed SHA-256")
		}
	}

	postPayload, err := base64.RawStdEncoding.DecodeString(scalarValue(document, "postimage.metadataBase64"))
	if err != nil || !json.Valid(postPayload) || fileSHA256(postPayload) != scalarValue(document, "postimage.metadataSHA256") {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_postimage", "postimage metadata must be valid unpadded base64 JSON with the signed digest")
	} else {
		var metadata map[string]any
		if json.Unmarshal(postPayload, &metadata) != nil || metadata["lifecycleState"] != "in-progress" {
			addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_postimage", "postimage metadata must set the exact lifecycle")
		}
		workVersion, ok := metadata["workVersion"].(map[string]any)
		if !ok || workVersion["authorityGeneration"] != scalarValue(document, "postimage.generationRef") ||
			workVersion["issueIncarnation"] != scalarValue(document, "postimage.issueIncarnation") ||
			workVersion["issueMutationSequence"] != float64(1) || workVersion["dependencyGraphRevision"] != float64(1) {
			addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_work_version", "postimage metadata must carry the exact bootstrap WorkVersion")
		}
	}
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001PostclaimGrantPath))); err == nil {
		checkW001PostclaimGrant(root, findings)
	} else if !os.IsNotExist(err) {
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_state", "W-001 postclaim reconciliation state cannot be established")
	}
}

func checkW001PostclaimGrant(root string, findings *[]Finding) {
	data, err := readRepoFile(root, w001PostclaimGrantPath)
	if err != nil {
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_missing", "signed W-001 postclaim reconciliation grant is required")
		return
	}
	document := parseStrictGrant(data, w001PostclaimGrantScalars, w001PostclaimGrantSequences,
		[]string{"grant", "claim", "publication", "postimage", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_schema", "%s", message)
	}
	for _, expected := range w001PostclaimGrantScalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_value", "%s does not match the signed postclaim contract", expected.path)
		}
	}
	for path, expected := range w001PostclaimGrantSequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_sequence", "%s must equal the exact ordered postclaim contract", path)
		}
	}
	for _, section := range []string{"grant", "claim", "publication", "postimage", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_schema", "%s mapping must occur exactly once", section)
		}
	}
	issuedAt, issueErr := time.Parse(time.RFC3339, scalarValue(document, "grant.issuedAt"))
	expiresAt, expiryErr := time.Parse(time.RFC3339, scalarValue(document, "grant.expiresAt"))
	if issueErr != nil || expiryErr != nil || !expiresAt.After(issuedAt) || expiresAt.Sub(issuedAt) > 72*time.Hour {
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_expiry", "postclaim grant must use one RFC3339 interval no longer than 72 hours")
	}

	signature, signatureErr := readRepoFile(root, w001PostclaimGrantSignature)
	if signatureErr != nil {
		addFinding(findings, w001PostclaimGrantSignature, "public.w001_postclaim_signature_missing", "detached postclaim reconciliation signature is required")
	}
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.w001_postclaim_key", "postclaim reconciliation must use the independently pinned genesis key")
	} else if signatureErr == nil {
		if err := verifySSHSig(data, signature, publicKey, w001PostclaimGrantNamespace); err != nil {
			addFinding(findings, w001PostclaimGrantSignature, "public.w001_postclaim_signature", "%v", err)
		}
	}

	for _, binding := range []struct {
		path    string
		digest  string
		code    string
		message string
	}{
		{w001BootstrapGrantPath, scalarValue(document, "grant.priorGrantSHA256"), "public.w001_postclaim_prior_grant", "prior bootstrap grant"},
		{w001BootstrapGrantSignature, scalarValue(document, "grant.priorGrantSignatureSHA256"), "public.w001_postclaim_prior_grant", "prior bootstrap signature"},
		{".harness/manifest.yaml", scalarValue(document, "postimage.manifestSHA256"), "public.w001_postclaim_postimage", "manifest postimage"},
		{canonicalActivePlan, scalarValue(document, "postimage.activePlanSHA256"), "public.w001_postclaim_postimage", "active-plan postimage"},
		{"docs/evidence/W-001-bootstrap-transition.md", scalarValue(document, "postimage.evidenceSHA256"), "public.w001_postclaim_postimage", "claim-evidence postimage"},
	} {
		content, readErr := readRepoFile(root, binding.path)
		if readErr != nil || !sha256Pattern.MatchString(binding.digest) || fileSHA256(content) != binding.digest {
			addFinding(findings, binding.path, binding.code, "%s must match its exact signed SHA-256", binding.message)
		}
	}

	evidence, evidenceErr := readRepoFile(root, "docs/evidence/W-001-bootstrap-transition.md")
	receiptDigest, receiptErr := extractW001ClaimReceiptDigest(evidence)
	if evidenceErr != nil || receiptErr != nil || receiptDigest != scalarValue(document, "claim.receiptSHA256") {
		addFinding(findings, "docs/evidence/W-001-bootstrap-transition.md", "public.w001_postclaim_receipt", "canonical claim receipt must be bound by one exact public SHA-256 reference")
	}
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001PostclaimCIFixPath))); err == nil {
		checkW001PostclaimCIFixAddendum(root, findings)
	} else if !os.IsNotExist(err) {
		addFinding(findings, w001PostclaimCIFixPath, "public.w001_postclaim_ci_state", "postclaim CI-stabilization state cannot be established")
	}
}

func checkW001PostclaimCIFixAddendum(root string, findings *[]Finding) {
	data, err := readRepoFile(root, w001PostclaimCIFixPath)
	if err != nil {
		addFinding(findings, w001PostclaimCIFixPath, "public.w001_postclaim_ci_missing", "signed postclaim CI-stabilization addendum is required")
		return
	}
	document := parseStrictGrant(data, w001PostclaimCIFixScalars, w001PostclaimCIFixSequences,
		[]string{"addendum", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, w001PostclaimCIFixPath, "public.w001_postclaim_ci_schema", "%s", message)
	}
	for _, expected := range w001PostclaimCIFixScalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, w001PostclaimCIFixPath, "public.w001_postclaim_ci_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, w001PostclaimCIFixPath, "public.w001_postclaim_ci_value", "%s does not match the signed CI-stabilization contract", expected.path)
		}
	}
	for path, expected := range w001PostclaimCIFixSequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, w001PostclaimCIFixPath, "public.w001_postclaim_ci_sequence", "%s must equal the exact ordered CI-stabilization contract", path)
		}
	}
	for _, section := range []string{"addendum", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, w001PostclaimCIFixPath, "public.w001_postclaim_ci_schema", "%s mapping must occur exactly once", section)
		}
	}
	issuedAt, issueErr := time.Parse(time.RFC3339, scalarValue(document, "addendum.issuedAt"))
	expiresAt, expiryErr := time.Parse(time.RFC3339, scalarValue(document, "addendum.expiresAt"))
	if issueErr != nil || expiryErr != nil || !expiresAt.After(issuedAt) || expiresAt.Sub(issuedAt) > 72*time.Hour {
		addFinding(findings, w001PostclaimCIFixPath, "public.w001_postclaim_ci_expiry", "CI-stabilization addendum must use one RFC3339 interval no longer than 72 hours")
	}

	signature, signatureErr := readRepoFile(root, w001PostclaimCIFixSignature)
	if signatureErr != nil {
		addFinding(findings, w001PostclaimCIFixSignature, "public.w001_postclaim_ci_signature_missing", "detached CI-stabilization signature is required")
	}
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.w001_postclaim_ci_key", "CI stabilization must use the independently pinned genesis key")
	} else if signatureErr == nil {
		if err := verifySSHSig(data, signature, publicKey, w001PostclaimCIFixNamespace); err != nil {
			addFinding(findings, w001PostclaimCIFixSignature, "public.w001_postclaim_ci_signature", "%v", err)
		}
	}
	for _, binding := range []struct {
		path   string
		digest string
	}{
		{w001PostclaimGrantPath, scalarValue(document, "addendum.priorGrantSHA256")},
		{w001PostclaimGrantSignature, scalarValue(document, "addendum.priorGrantSignatureSHA256")},
	} {
		content, readErr := readRepoFile(root, binding.path)
		if readErr != nil || !sha256Pattern.MatchString(binding.digest) || fileSHA256(content) != binding.digest {
			addFinding(findings, binding.path, "public.w001_postclaim_ci_prior_grant", "prior postclaim grant material must match its exact signed SHA-256")
		}
	}
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001PostclaimSecurityFixPath))); err == nil {
		checkW001PostclaimSecurityFix(root, findings)
	} else if !os.IsNotExist(err) {
		addFinding(findings, w001PostclaimSecurityFixPath, "public.w001_postclaim_security_state", "postclaim Security-correction state cannot be established")
	}
}

func checkW001PostclaimSecurityFix(root string, findings *[]Finding) {
	data, err := readRepoFile(root, w001PostclaimSecurityFixPath)
	if err != nil {
		addFinding(findings, w001PostclaimSecurityFixPath, "public.w001_postclaim_security_missing", "signed postclaim Security-correction grant is required")
		return
	}
	document := parseStrictGrant(data, w001PostclaimSecurityFixScalars, w001PostclaimSecurityFixSequences,
		[]string{"grant", "finding", "canonicalEffect", "materials", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, w001PostclaimSecurityFixPath, "public.w001_postclaim_security_schema", "%s", message)
	}
	for _, expected := range w001PostclaimSecurityFixScalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, w001PostclaimSecurityFixPath, "public.w001_postclaim_security_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, w001PostclaimSecurityFixPath, "public.w001_postclaim_security_value", "%s does not match the signed Security-correction contract", expected.path)
		}
	}
	for path, expected := range w001PostclaimSecurityFixSequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, w001PostclaimSecurityFixPath, "public.w001_postclaim_security_sequence", "%s must equal the exact ordered Security-correction contract", path)
		}
	}
	for _, section := range []string{"grant", "finding", "canonicalEffect", "materials", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, w001PostclaimSecurityFixPath, "public.w001_postclaim_security_schema", "%s mapping must occur exactly once", section)
		}
	}
	issuedAt, issueErr := time.Parse(time.RFC3339, scalarValue(document, "grant.issuedAt"))
	expiresAt, expiryErr := time.Parse(time.RFC3339, scalarValue(document, "grant.expiresAt"))
	if issueErr != nil || expiryErr != nil || !expiresAt.After(issuedAt) || expiresAt.Sub(issuedAt) > 72*time.Hour {
		addFinding(findings, w001PostclaimSecurityFixPath, "public.w001_postclaim_security_expiry", "Security-correction grant must use one RFC3339 interval no longer than 72 hours")
	}

	signature, signatureErr := readRepoFile(root, w001PostclaimSecurityFixSig)
	if signatureErr != nil {
		addFinding(findings, w001PostclaimSecurityFixSig, "public.w001_postclaim_security_signature_missing", "detached Security-correction signature is required")
	}
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.w001_postclaim_security_key", "Security correction must use the independently pinned genesis key")
	} else if signatureErr == nil {
		if err := verifySSHSig(data, signature, publicKey, w001PostclaimSecurityFixNS); err != nil {
			addFinding(findings, w001PostclaimSecurityFixSig, "public.w001_postclaim_security_signature", "%v", err)
		}
	}

	for _, binding := range []struct {
		path   string
		digest string
		code   string
	}{
		{w001PostclaimCIFixPath, scalarValue(document, "grant.priorAddendumSHA256"), "public.w001_postclaim_security_prior_addendum"},
		{w001PostclaimCIFixSignature, scalarValue(document, "grant.priorAddendumSignatureSHA256"), "public.w001_postclaim_security_prior_addendum"},
		{scalarValue(document, "materials.helperLibraryPath"), scalarValue(document, "materials.helperLibrarySHA256"), "public.w001_postclaim_security_material"},
		{scalarValue(document, "materials.helperTestPath"), scalarValue(document, "materials.helperTestSHA256"), "public.w001_postclaim_security_material"},
		{scalarValue(document, "materials.basePatchPath"), scalarValue(document, "materials.basePatchSHA256"), "public.w001_postclaim_security_material"},
		{scalarValue(document, "materials.securityPatchPath"), scalarValue(document, "materials.securityPatchSHA256"), "public.w001_postclaim_security_material"},
	} {
		content, readErr := readRepoFile(root, binding.path)
		if readErr != nil || !sha256Pattern.MatchString(binding.digest) || fileSHA256(content) != binding.digest {
			addFinding(findings, binding.path, binding.code, "Security-correction material must match its exact signed SHA-256")
		}
	}
	evidence, evidenceErr := readRepoFile(root, "docs/evidence/W-001-validation.md")
	if evidenceErr != nil || !bytes.Contains(evidence, []byte("bootstrap-effective-database-selector-splice")) ||
		!bytes.Contains(evidence, []byte("**Current disposition:** changes-requested")) ||
		!bytes.Contains(evidence, []byte("earlier Security acceptance")) || !bytes.Contains(evidence, []byte("is superseded")) {
		addFinding(findings, "docs/evidence/W-001-validation.md", "public.w001_postclaim_security_evidence", "Security correction evidence must preserve the exact additive supersession and finding fingerprint")
	}
}

func extractW001ClaimReceiptDigest(evidence []byte) (string, error) {
	const heading = "## Canonical claim receipt\n"
	sectionStart := bytes.Index(evidence, []byte(heading))
	if sectionStart < 0 || bytes.Count(evidence, []byte(heading)) != 1 {
		return "", errors.New("canonical claim receipt section is missing or duplicated")
	}
	section := evidence[sectionStart+len(heading):]
	if next := bytes.Index(section, []byte("\n## ")); next >= 0 {
		section = section[:next]
	}
	const prefix = "- Receipt SHA-256: `"
	start := bytes.Index(section, []byte(prefix))
	if start < 0 || bytes.Count(section, []byte(prefix)) != 1 {
		return "", errors.New("canonical claim receipt digest is missing or duplicated")
	}
	digestStart := start + len(prefix)
	digestEnd := bytes.IndexByte(section[digestStart:], '`')
	if digestEnd < 0 {
		return "", errors.New("canonical claim receipt digest is not closed")
	}
	digest := string(section[digestStart : digestStart+digestEnd])
	if !sha256Pattern.MatchString(digest) {
		return "", errors.New("canonical claim receipt digest must be lowercase SHA-256")
	}
	return digest, nil
}

func scalarValue(document strictPlanningGrant, path string) string {
	values := document.scalars[path]
	if len(values) != 1 {
		return ""
	}
	return values[0]
}

// LoadW001BootstrapGrant returns the exact validated public configuration for
// the one-shot helper. It performs the complete repository grant check first.
func LoadW001BootstrapGrant(repo string) (W001BootstrapGrant, error) {
	root, err := repositoryRoot(repo)
	if err != nil {
		return W001BootstrapGrant{}, err
	}
	var findings []Finding
	checkWave1PlanningGrant(root, &findings)
	if len(findings) != 0 {
		sortFindings(findings)
		return W001BootstrapGrant{}, fmt.Errorf("W-001 bootstrap grant validation failed: %s: %s", findings[0].Code, findings[0].Message)
	}
	data, err := readRepoFile(root, w001BootstrapGrantPath)
	if err != nil {
		return W001BootstrapGrant{}, err
	}
	document := parseStrictGrant(data, w001BootstrapGrantScalars, w001BootstrapGrantSequences,
		[]string{"grant", "expected", "postimage", "toolchain", "verification", "integrity"})
	grant := W001BootstrapGrant{
		ID: scalarValue(document, "grant.id"), AttemptID: scalarValue(document, "grant.attemptId"), IdempotencyKey: scalarValue(document, "grant.replayRef"),
		Bead: scalarValue(document, "grant.bead"), BaseCommit: scalarValue(document, "grant.baseCommit"), ExpiresAt: scalarValue(document, "grant.expiresAt"),
		WorkingBranch: scalarValue(document, "grant.workingBranch"), Assignee: scalarValue(document, "expected.assignee"),
		AuthorityProjectID:   scalarValue(document, "expected.beadsProject"),
		ExpectedNativeStatus: scalarValue(document, "expected.nativeStatus"), ExpectedLifecycleState: scalarValue(document, "expected.lifecycleState"),
		ExpectedCreatedAt: scalarValue(document, "expected.createdAt"), ExpectedUpdatedAt: scalarValue(document, "expected.updatedAt"),
		ExpectedMetadataSHA256: scalarValue(document, "expected.metadataSHA256"), ExpectedLabelsSHA256: scalarValue(document, "expected.labelsSHA256"),
		ExpectedDependency: scalarValue(document, "expected.dependency"), ExpectedDependencyType: scalarValue(document, "expected.dependencyType"),
		ExpectedDependencyStatus: scalarValue(document, "expected.dependencyNativeStatus"), ExpectedDependencyLifecycle: scalarValue(document, "expected.dependencyLifecycleState"),
		ExpectedDependencySHA256: scalarValue(document, "expected.dependencyDigest"), ExpectedLineageSHA256: scalarValue(document, "expected.lineageDigest"),
		PostNativeStatus: scalarValue(document, "postimage.nativeStatus"), PostLifecycleState: scalarValue(document, "postimage.lifecycleState"),
		PostMetadataSHA256: scalarValue(document, "postimage.metadataSHA256"), PostLabelsSHA256: scalarValue(document, "postimage.labelsSHA256"),
		PostMetadataBase64: scalarValue(document, "postimage.metadataBase64"), RemoveLabel: scalarValue(document, "postimage.removeLabel"), AddLabel: scalarValue(document, "postimage.addLabel"),
		BeadsVersion: scalarValue(document, "toolchain.beadsVersion"), BeadsSourceCommit: scalarValue(document, "toolchain.beadsSourceCommit"),
		BeadsBinarySHA256: scalarValue(document, "toolchain.beadsBinarySHA256"),
		DoltModule:        scalarValue(document, "toolchain.doltModule"), DoltModuleSHA256: scalarValue(document, "toolchain.doltModuleSHA256"),
		GoVersion: scalarValue(document, "toolchain.goVersion"), GoOS: scalarValue(document, "toolchain.goOS"), GoArch: scalarValue(document, "toolchain.goArch"),
		ICUFormula: scalarValue(document, "toolchain.icuFormula"), DoltTestImage: scalarValue(document, "toolchain.doltTestImage"),
		PatchPath: scalarValue(document, "toolchain.patchPath"), PatchSHA256: scalarValue(document, "toolchain.patchSHA256"),
		PatchedBinarySHA256: scalarValue(document, "toolchain.patchedBinarySHA256"), GoBinarySHA256: scalarValue(document, "toolchain.goBinarySHA256"),
		ReviewTag: scalarValue(document, "grant.reviewTag"),
	}
	if _, stateErr := os.Lstat(filepath.Join(root, filepath.FromSlash(w001PostclaimSecurityFixPath))); stateErr == nil {
		grant.CorrectionPatchPath = w001PostclaimSecurityPatchPath
		grant.CorrectionPatchSHA256 = w001PostclaimSecurityPatchSHA
		grant.PatchedBinarySHA256 = w001PostclaimSecurityBinarySHA
	}
	return grant, nil
}

// LoadW001BootstrapExecutionAuthorization verifies the separately signed,
// short-lived authorization that can exist only after protected-main review.
func LoadW001BootstrapExecutionAuthorization(repo, path string, grant W001BootstrapGrant) (W001BootstrapExecutionAuthorization, error) {
	root, err := repositoryRoot(repo)
	if err != nil {
		return W001BootstrapExecutionAuthorization{}, err
	}
	if path == "" || filepath.Ext(path) != ".json" {
		return W001BootstrapExecutionAuthorization{}, errors.New("execution authorization must be one explicit JSON file")
	}
	for _, candidate := range []string{path, path + ".sig"} {
		info, statErr := os.Lstat(candidate)
		if statErr != nil || !info.Mode().IsRegular() {
			return W001BootstrapExecutionAuthorization{}, errors.New("execution authorization and detached signature must be regular files")
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return W001BootstrapExecutionAuthorization{}, err
	}
	authorization, err := decodeCanonicalW001ExecutionAuthorization(data)
	if err != nil {
		return W001BootstrapExecutionAuthorization{}, err
	}
	signature, err := os.ReadFile(path + ".sig")
	if err != nil {
		return W001BootstrapExecutionAuthorization{}, err
	}
	publicKey, err := readRepoFile(root, wave1PlanningGrantKey)
	if err != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest || verifySSHSig(data, signature, publicKey, w001BootstrapExecutionNamespace) != nil {
		return W001BootstrapExecutionAuthorization{}, errors.New("execution authorization signature is invalid")
	}
	authorization.payloadSHA256 = fileSHA256(data)
	authorization.signatureSHA256 = fileSHA256(signature)
	issuedAt, issueErr := time.Parse(time.RFC3339, authorization.IssuedAt)
	expiresAt, expiryErr := time.Parse(time.RFC3339, authorization.ExpiresAt)
	now := time.Now().UTC()
	if issueErr != nil || expiryErr != nil || expiresAt.Sub(issuedAt) <= 0 || expiresAt.Sub(issuedAt) > time.Hour || now.Before(issuedAt.Add(-5*time.Minute)) || !now.Before(expiresAt) {
		return W001BootstrapExecutionAuthorization{}, errors.New("execution authorization is outside its signed one-hour validity window")
	}
	if authorization.SchemaVersion != 1 || authorization.Kind != "MARS3W001BootstrapExecutionAuthorization" || authorization.Classification != "PUBLIC" ||
		authorization.GrantID != grant.ID || authorization.Repository != planningGrantRepository || authorization.AttemptID != grant.AttemptID ||
		authorization.IdempotencyKey != grant.IdempotencyKey || authorization.Bead != grant.Bead || authorization.AuthorityProjectID != grant.AuthorityProjectID ||
		authorization.ReviewTag != grant.ReviewTag || authorization.PullRequest != 6 || authorization.ProtectedMainCheckRun <= 0 ||
		authorization.QADisposition != "accepted" || authorization.SecurityDisposition != "accepted" ||
		authorization.QAReviewedCommit != authorization.ReviewedFeatureCommit || authorization.SecurityReviewedCommit != authorization.ReviewedFeatureCommit ||
		authorization.PatchedBinarySHA256 != grant.PatchedBinarySHA256 || authorization.ExpectedMetadataSHA256 != grant.ExpectedMetadataSHA256 ||
		!sha256Pattern.MatchString(authorization.WorkspaceInstanceSHA256) ||
		authorization.AllowedEffect != "execute-one-expected-preimage-W-001-CAS-claim" ||
		!sha1Pattern.MatchString(authorization.MergedCommit) || !sha1Pattern.MatchString(authorization.ReviewedFeatureCommit) ||
		!sha1Pattern.MatchString(authorization.MergedTree) || authorization.MergedCommit == authorization.ReviewedFeatureCommit {
		return W001BootstrapExecutionAuthorization{}, errors.New("execution authorization does not match the exact post-review claim contract")
	}
	return authorization, nil
}

func decodeCanonicalW001ExecutionAuthorization(data []byte) (W001BootstrapExecutionAuthorization, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var authorization W001BootstrapExecutionAuthorization
	if err := decoder.Decode(&authorization); err != nil {
		return W001BootstrapExecutionAuthorization{}, fmt.Errorf("decode execution authorization: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return W001BootstrapExecutionAuthorization{}, errors.New("execution authorization must contain exactly one JSON object")
	}
	canonical, err := json.Marshal(authorization)
	if err != nil {
		return W001BootstrapExecutionAuthorization{}, err
	}
	canonical = append(canonical, '\n')
	if !bytes.Equal(data, canonical) {
		return W001BootstrapExecutionAuthorization{}, errors.New("execution authorization must use exact canonical JSON with one trailing newline")
	}
	return authorization, nil
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
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001BootstrapGrantPath))); err == nil {
		checkW001BootstrapGrantGitDiff(root, findings)
		return
	} else if !os.IsNotExist(err) {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_state", "W-001 bootstrap grant state cannot be established")
		return
	}
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(wave1DirectMainGrantPath))); err == nil {
		checkWave1DirectMainTransitionGitDiff(root, findings)
		return
	} else if !os.IsNotExist(err) {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_state", "direct-main transition grant state cannot be established")
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

func checkW001PostclaimGrantGitDiff(root string, findings *[]Finding) {
	ciFixActive := false
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001PostclaimCIFixPath))); err == nil {
		ciFixActive = true
	} else if !os.IsNotExist(err) {
		addFinding(findings, w001PostclaimCIFixPath, "public.w001_postclaim_ci_state", "postclaim CI-stabilization Git state cannot be established")
		return
	}
	securityFixActive := false
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001PostclaimSecurityFixPath))); err == nil {
		securityFixActive = true
	} else if !os.IsNotExist(err) {
		addFinding(findings, w001PostclaimSecurityFixPath, "public.w001_postclaim_security_state", "postclaim Security-correction Git state cannot be established")
		return
	}
	if securityFixActive && !ciFixActive {
		addFinding(findings, w001PostclaimSecurityFixPath, "public.w001_postclaim_security_ancestry", "Security correction requires the preserved v2 CI stabilization")
		return
	}
	base, err := planningGrantGitOutput(root, "rev-parse", "--verify", w001PostclaimBase+"^{commit}")
	if err != nil || strings.TrimSpace(string(base)) != w001PostclaimBase {
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_base", "exact accepted helper squash must resolve locally")
		return
	}
	baseTree, err := planningGrantGitOutput(root, "rev-parse", "--verify", w001PostclaimBase+"^{tree}")
	if err != nil || strings.TrimSpace(string(baseTree)) != w001PostclaimBaseTree {
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_base", "accepted helper squash tree must match the signed base tree")
		return
	}
	if ciFixActive {
		fixBase, err := planningGrantGitOutput(root, "rev-parse", "--verify", w001PostclaimCIFixBase+"^{commit}")
		fixTree, treeErr := planningGrantGitOutput(root, "rev-parse", "--verify", w001PostclaimCIFixBase+"^{tree}")
		if err != nil || treeErr != nil || strings.TrimSpace(string(fixBase)) != w001PostclaimCIFixBase || strings.TrimSpace(string(fixTree)) != w001PostclaimCIFixBaseTree {
			addFinding(findings, w001PostclaimCIFixPath, "public.w001_postclaim_ci_base", "CI stabilization must descend from the exact preserved failed head and tree")
			return
		}
		if !checkW001PostclaimPriorReviewTag(root, findings) {
			return
		}
	}
	if securityFixActive {
		fixBase, err := planningGrantGitOutput(root, "rev-parse", "--verify", w001PostclaimSecurityFixBase+"^{commit}")
		fixTree, treeErr := planningGrantGitOutput(root, "rev-parse", "--verify", w001PostclaimSecurityFixBase+"^{tree}")
		if err != nil || treeErr != nil || strings.TrimSpace(string(fixBase)) != w001PostclaimSecurityFixBase || strings.TrimSpace(string(fixTree)) != w001PostclaimSecurityFixTree {
			addFinding(findings, w001PostclaimSecurityFixPath, "public.w001_postclaim_security_base", "Security correction must descend from the exact accepted v2 head and tree")
			return
		}
		if !checkW001PostclaimPriorV2ReviewTag(root, findings) {
			return
		}
	}
	headOutput, err := planningGrantGitOutput(root, "rev-parse", "--verify", "HEAD^{commit}")
	head := strings.TrimSpace(string(headOutput))
	if err != nil || !sha1Pattern.MatchString(head) {
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_git", "HEAD must resolve to one exact commit")
		return
	}
	branchOutput, branchErr := planningGrantGitOutput(root, "symbolic-ref", "--quiet", "--short", "HEAD")
	branch := strings.TrimSpace(string(branchOutput))
	featureHead := head
	requireTag := false
	mainTreeCheck := false
	switch {
	case branchErr == nil && branch == w001PostclaimBranch && os.Getenv("GITHUB_ACTIONS") != "true":
		if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", w001PostclaimBase, head); err != nil {
			addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_ancestry", "local reconciliation must descend from the exact accepted helper squash")
			return
		}
	case branchErr == nil && branch == "main" && os.Getenv("GITHUB_ACTIONS") != "true":
		requireTag, mainTreeCheck = true, true
	case os.Getenv("GITHUB_ACTIONS") == "true":
		featureHead, requireTag, mainTreeCheck = w001PostclaimGitHubCheckout(root, head, branch, findings)
		if featureHead == "" {
			return
		}
	default:
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_branch", "postclaim reconciliation requires its signed branch or accepted main")
		return
	}

	if requireTag {
		expected := featureHead
		if mainTreeCheck {
			expected = ""
		}
		reviewTag := w001PostclaimReviewTag
		reviewTagMessage := w001PostclaimReviewTagMessage
		if ciFixActive {
			reviewTag = w001PostclaimCIFixReviewTag
			reviewTagMessage = w001PostclaimCIFixTagMessage
		}
		if securityFixActive {
			reviewTag = w001PostclaimSecurityFixTag
			reviewTagMessage = w001PostclaimSecurityFixTagMsg
		}
		target, ok := checkW001PostclaimReviewTag(root, expected, reviewTag, reviewTagMessage, findings)
		if !ok {
			return
		}
		featureHead = target
	}
	if mainTreeCheck {
		parents, err := planningGrantCommitParents(root, head)
		if err != nil || len(parents) != 1 || parents[0] != w001PostclaimBase {
			addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_main_topology", "accepted reconciliation must be one squash commit over the signed base")
			return
		}
		mainTree, _ := planningGrantGitOutput(root, "rev-parse", "--verify", head+"^{tree}")
		featureTree, _ := planningGrantGitOutput(root, "rev-parse", "--verify", featureHead+"^{tree}")
		if strings.TrimSpace(string(mainTree)) != strings.TrimSpace(string(featureTree)) {
			addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_main_tree", "accepted reconciliation tree must equal the signed reviewed feature tree")
			return
		}
	}

	if featureHead != w001PostclaimBase {
		commits, err := planningGrantCommitRangeFrom(root, w001PostclaimBase, featureHead)
		if err != nil || len(commits) == 0 {
			addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_history", "reconciliation feature history must be a nonempty linear chain")
			return
		}
		publicKey, err := readRepoFile(root, wave1PlanningGrantKey)
		if err != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
			addFinding(findings, wave1PlanningGrantKey, "public.w001_postclaim_commit_signature", "reconciliation commits require the pinned genesis signer")
			return
		}
		previous := w001PostclaimBase
		v1Authorized := w001PostclaimPathSet()
		fixAuthorized := w001PostclaimCIFixPathSet()
		securityAuthorized := w001PostclaimSecurityFixPathSet()
		for _, commit := range commits {
			if len(commit.parents) != 1 || commit.parents[0] != previous {
				addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_topology", "reconciliation feature history must be a contiguous one-parent chain")
				return
			}
			authorized := v1Authorized
			if ciFixActive {
				if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", commit.id, w001PostclaimCIFixBase); err != nil {
					if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", w001PostclaimCIFixBase, commit.id); err != nil {
						addFinding(findings, w001PostclaimCIFixPath, "public.w001_postclaim_ci_ancestry", "correction history diverges from the preserved failed head")
						return
					}
					authorized = fixAuthorized
				}
			}
			if securityFixActive {
				if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", commit.id, w001PostclaimSecurityFixBase); err != nil {
					if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", w001PostclaimSecurityFixBase, commit.id); err != nil {
						addFinding(findings, w001PostclaimSecurityFixPath, "public.w001_postclaim_security_ancestry", "Security-correction history diverges from the preserved v2 head")
						return
					}
					authorized = securityAuthorized
				}
			}
			paths, err := planningGrantGitOutput(root, "diff-tree", "--no-commit-id", "--no-renames", "--no-ext-diff", "--no-textconv", "--name-only", "-z", "-r", commit.id+"^", commit.id)
			normalized, normalizeErr := normalizedPlanningGrantGitPaths(paths)
			if err != nil || normalizeErr != nil || !planningGrantPathsAllowed(normalized, authorized) {
				addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_scope", "a reconciliation commit includes a path outside its signed scope")
				return
			}
			object, err := planningGrantGitOutput(root, "cat-file", "commit", commit.id)
			if err != nil || verifyPlanningGrantCommit(object, publicKey) != nil {
				addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_commit_signature", "every reconciliation feature commit must carry the pinned SSH signature")
				return
			}
			previous = commit.id
		}
	}

	tracked, err := planningGrantGitOutput(root, "diff", "--no-renames", "--no-ext-diff", "--no-textconv", "--name-only", "-z", "HEAD", "--")
	if err != nil {
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_scope", "current tracked reconciliation paths cannot be enumerated")
		return
	}
	untracked, err := planningGrantGitOutput(root, "ls-files", "--others", "--exclude-standard", "-z", "--")
	if err != nil {
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_scope", "current untracked reconciliation paths cannot be enumerated")
		return
	}
	paths, err := normalizedPlanningGrantGitPaths(tracked, untracked)
	currentAuthorized := w001PostclaimPathSet()
	if ciFixActive {
		currentAuthorized = w001PostclaimCIFixPathSet()
	}
	if securityFixActive {
		currentAuthorized = w001PostclaimSecurityFixPathSet()
	}
	if err != nil || !planningGrantPathsAllowed(paths, currentAuthorized) {
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_scope", "current changes include a path outside the signed reconciliation scope")
	}
}

func w001PostclaimPathSet() map[string]bool {
	authorized := make(map[string]bool, len(w001PostclaimGrantSequences["grant.authorizedPaths"]))
	for _, path := range w001PostclaimGrantSequences["grant.authorizedPaths"] {
		authorized[path] = true
	}
	return authorized
}

func w001PostclaimCIFixPathSet() map[string]bool {
	authorized := make(map[string]bool, len(w001PostclaimCIFixSequences["addendum.authorizedPaths"]))
	for _, path := range w001PostclaimCIFixSequences["addendum.authorizedPaths"] {
		authorized[path] = true
	}
	return authorized
}

func w001PostclaimSecurityFixPathSet() map[string]bool {
	authorized := make(map[string]bool, len(w001PostclaimSecurityFixSequences["grant.authorizedPaths"]))
	for _, path := range w001PostclaimSecurityFixSequences["grant.authorizedPaths"] {
		authorized[path] = true
	}
	return authorized
}

func checkW001PostclaimReviewTag(root, expectedFeatureHead, reviewTag, reviewTagMessage string, findings *[]Finding) (string, bool) {
	ref := "refs/tags/" + reviewTag
	objectID, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{tag}")
	if err != nil || !sha1Pattern.MatchString(strings.TrimSpace(string(objectID))) {
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_tag", "postclaim CI requires the signed immutable review tag")
		return "", false
	}
	object, err := planningGrantGitOutput(root, "cat-file", "tag", strings.TrimSpace(string(objectID)))
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	if err != nil || keyErr != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_tag", "postclaim review tag cannot be verified with the pinned key")
		return "", false
	}
	target, err := verifyPinnedPlanningGrantTag(object, publicKey, reviewTag, reviewTagMessage)
	if err != nil {
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_tag", "postclaim review tag must be an exact pinned-signer tree attestation")
		return "", false
	}
	if expectedFeatureHead != "" && expectedFeatureHead != target {
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_tag_target", "postclaim review tag must target the immutable feature head")
		return "", false
	}
	if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", w001PostclaimBase, target); err != nil || target == w001PostclaimBase {
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_tag_target", "postclaim tag target must preserve nonempty reconciliation history")
		return "", false
	}
	return target, true
}

func checkW001PostclaimPriorReviewTag(root string, findings *[]Finding) bool {
	ref := "refs/tags/" + w001PostclaimReviewTag
	objectID, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{tag}")
	if err != nil || strings.TrimSpace(string(objectID)) != w001PostclaimV1TagObject {
		addFinding(findings, w001PostclaimCIFixPath, "public.w001_postclaim_ci_v1_tag", "v1 review tag object must remain exact and immutable")
		return false
	}
	object, err := planningGrantGitOutput(root, "cat-file", "tag", w001PostclaimV1TagObject)
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	if err != nil || keyErr != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		addFinding(findings, w001PostclaimCIFixPath, "public.w001_postclaim_ci_v1_tag", "v1 review tag cannot be verified with the pinned key")
		return false
	}
	target, err := verifyPinnedPlanningGrantTag(object, publicKey, w001PostclaimReviewTag, w001PostclaimReviewTagMessage)
	if err != nil || target != w001PostclaimCIFixBase {
		addFinding(findings, w001PostclaimCIFixPath, "public.w001_postclaim_ci_v1_tag", "v1 review tag target and signature must remain exact")
		return false
	}
	return true
}

func checkW001PostclaimPriorV2ReviewTag(root string, findings *[]Finding) bool {
	ref := "refs/tags/" + w001PostclaimCIFixReviewTag
	objectID, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{tag}")
	if err != nil || strings.TrimSpace(string(objectID)) != w001PostclaimV2TagObject {
		addFinding(findings, w001PostclaimSecurityFixPath, "public.w001_postclaim_security_v2_tag", "v2 review tag object must remain exact and immutable")
		return false
	}
	object, err := planningGrantGitOutput(root, "cat-file", "tag", w001PostclaimV2TagObject)
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	if err != nil || keyErr != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		addFinding(findings, w001PostclaimSecurityFixPath, "public.w001_postclaim_security_v2_tag", "v2 review tag cannot be verified with the pinned key")
		return false
	}
	target, err := verifyPinnedPlanningGrantTag(object, publicKey, w001PostclaimCIFixReviewTag, w001PostclaimCIFixTagMessage)
	if err != nil || target != w001PostclaimSecurityFixBase {
		addFinding(findings, w001PostclaimSecurityFixPath, "public.w001_postclaim_security_v2_tag", "v2 review tag target and signature must remain exact")
		return false
	}
	return true
}

func w001PostclaimGitHubCheckout(root, head, branch string, findings *[]Finding) (string, bool, bool) {
	if os.Getenv("CI") != "true" || os.Getenv("GITHUB_ACTIONS") != "true" || os.Getenv("RUNNER_ENVIRONMENT") != "github-hosted" ||
		os.Getenv("GITHUB_REPOSITORY") != planningGrantRepository || os.Getenv("GITHUB_WORKFLOW") != planningGrantWorkflow ||
		os.Getenv("GITHUB_JOB") != planningGrantWorkflowJob || os.Getenv("GITHUB_SHA") != head ||
		!samePlanningGrantRepositoryRoot(root, os.Getenv("GITHUB_WORKSPACE")) {
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_runner", "postclaim GitHub checkout lacks canonical runner identity")
		return "", false, false
	}
	if _, ok := parsePositiveInt(os.Getenv("GITHUB_RUN_ID")); !ok {
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_runner", "postclaim GitHub run ID is invalid")
		return "", false, false
	}
	if _, ok := parsePositiveInt(os.Getenv("GITHUB_RUN_ATTEMPT")); !ok {
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_runner", "postclaim GitHub run attempt is invalid")
		return "", false, false
	}
	workflowRef := os.Getenv("GITHUB_WORKFLOW_REF")
	workflowPrefix := planningGrantRepository + "/" + planningGrantWorkflowPath + "@"
	workflow, err := readRepoFile(root, planningGrantWorkflowPath)
	if err != nil || fileSHA256(workflow) != canonicalFoundationWorkflowSHA256 || !strings.HasPrefix(workflowRef, workflowPrefix) {
		addFinding(findings, planningGrantWorkflowPath, "public.w001_postclaim_workflow", "postclaim CI requires the pinned protected workflow")
		return "", false, false
	}
	event, ok := readPlanningGrantGitHubEvent(os.Getenv("GITHUB_EVENT_PATH"))
	if !ok || event.Repository.FullName != planningGrantRepository {
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_event", "postclaim CI event identity is invalid")
		return "", false, false
	}
	switch os.Getenv("GITHUB_EVENT_NAME") {
	case "pull_request":
		ref := os.Getenv("GITHUB_REF")
		if branch != "" || !validPlanningGrantPullRequestRef(ref) || os.Getenv("GITHUB_HEAD_REF") != w001PostclaimBranch ||
			os.Getenv("GITHUB_BASE_REF") != "main" || event.PullRequest == nil || event.PullRequest.Head.Ref != w001PostclaimBranch ||
			event.PullRequest.Base.Ref != "main" || event.PullRequest.Base.SHA != w001PostclaimBase ||
			!sha1Pattern.MatchString(event.PullRequest.Head.SHA) || !validAdvisoryPullRequestMergeSHA(event.PullRequest.MergeCommitSHA) {
			addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_event", "pull-request event does not bind the signed postclaim branch and base")
			return "", false, false
		}
		workflowSuffix := strings.TrimPrefix(workflowRef, workflowPrefix)
		if workflowSuffix != ref && workflowSuffix != "refs/heads/main" {
			addFinding(findings, planningGrantWorkflowPath, "public.w001_postclaim_workflow", "pull-request workflow ref is not canonical")
			return "", false, false
		}
		parents, err := planningGrantCommitParents(root, head)
		if err != nil || len(parents) != 2 || parents[0] != w001PostclaimBase || parents[1] != event.PullRequest.Head.SHA {
			addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_pr_topology", "pull-request checkout must be the exact two-parent synthetic merge")
			return "", false, false
		}
		mergeTree, mergeErr := planningGrantGitOutput(root, "rev-parse", "--verify", head+"^{tree}")
		featureTree, featureErr := planningGrantGitOutput(root, "rev-parse", "--verify", event.PullRequest.Head.SHA+"^{tree}")
		if mergeErr != nil || featureErr != nil || strings.TrimSpace(string(mergeTree)) != strings.TrimSpace(string(featureTree)) {
			addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_pr_tree", "pull-request synthetic merge tree must equal the reviewed feature tree")
			return "", false, false
		}
		return event.PullRequest.Head.SHA, true, false
	case "push":
		if branch != "" && branch != "main" || os.Getenv("GITHUB_REF") != "refs/heads/main" || os.Getenv("GITHUB_REF_PROTECTED") != "true" ||
			os.Getenv("GITHUB_HEAD_REF") != "" || os.Getenv("GITHUB_BASE_REF") != "" || workflowRef != workflowPrefix+"refs/heads/main" ||
			event.Ref != "refs/heads/main" || event.Before != w001PostclaimBase || event.After != head || event.HeadCommit == nil || event.HeadCommit.ID != head || event.PullRequest != nil {
			addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_event", "protected-main event does not bind the signed reconciliation base and squash")
			return "", false, false
		}
		return head, true, true
	default:
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_event", "unsupported GitHub event for postclaim reconciliation")
		return "", false, false
	}
}

func checkW001BootstrapGrantGitDiff(root string, findings *[]Finding) {
	topLevel, err := planningGrantGitOutput(root, "rev-parse", "--show-toplevel")
	if err != nil || !samePlanningGrantRepositoryRoot(root, strings.TrimSpace(string(topLevel))) {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_git", "Git metadata must resolve to the audited repository root")
		return
	}
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001PostclaimGrantPath))); err == nil {
		checkW001PostclaimGrantGitDiff(root, findings)
		return
	} else if !os.IsNotExist(err) {
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_state", "postclaim reconciliation Git state cannot be established")
		return
	}
	base, err := planningGrantGitOutput(root, "rev-parse", "--verify", w001BootstrapBase+"^{commit}")
	if err != nil || strings.TrimSpace(string(base)) != w001BootstrapBase {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_base", "the exact signed bootstrap base must resolve locally")
		return
	}
	baseTree, err := planningGrantGitOutput(root, "rev-parse", "--verify", w001BootstrapBase+"^{tree}")
	if err != nil || strings.TrimSpace(string(baseTree)) != w001BootstrapBaseTree {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_base", "the exact signed bootstrap base tree must remain unchanged")
		return
	}
	headOutput, err := planningGrantGitOutput(root, "rev-parse", "--verify", "HEAD^{commit}")
	head := strings.TrimSpace(string(headOutput))
	if err != nil || !sha1Pattern.MatchString(head) {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_git", "HEAD must resolve to one exact commit")
		return
	}
	branchOutput, branchErr := planningGrantGitOutput(root, "symbolic-ref", "--quiet", "--short", "HEAD")
	branch := strings.TrimSpace(string(branchOutput))
	featureHead := head
	requireTag := false
	mainTreeCheck := false

	switch {
	case branchErr == nil && branch == w001BootstrapBranch && os.Getenv("GITHUB_ACTIONS") != "true":
		if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", w001BootstrapBase, head); err != nil {
			addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_ancestry", "local bootstrap work must descend from the exact signed base")
			return
		}
	case branchErr == nil && branch == "main" && os.Getenv("GITHUB_ACTIONS") != "true":
		requireTag, mainTreeCheck = true, true
	case os.Getenv("GITHUB_ACTIONS") == "true":
		featureHead, requireTag, mainTreeCheck = w001BootstrapGitHubCheckout(root, head, branch, findings)
		if featureHead == "" {
			return
		}
	default:
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_branch", "bootstrap work requires the exact signed branch or accepted main")
		return
	}

	if requireTag {
		tagExpected := featureHead
		if mainTreeCheck {
			tagExpected = ""
		}
		target, ok := checkW001BootstrapReviewTag(root, tagExpected, findings)
		if !ok {
			return
		}
		featureHead = target
	}
	if mainTreeCheck {
		parents, err := planningGrantCommitParents(root, head)
		if err != nil || len(parents) != 1 || parents[0] != w001BootstrapBase {
			addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_main_topology", "accepted main must be one squash commit over the signed bootstrap base")
			return
		}
		mainTree, _ := planningGrantGitOutput(root, "rev-parse", "--verify", head+"^{tree}")
		featureTree, _ := planningGrantGitOutput(root, "rev-parse", "--verify", featureHead+"^{tree}")
		if strings.TrimSpace(string(mainTree)) != strings.TrimSpace(string(featureTree)) {
			addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_main_tree", "accepted main tree must equal the signed reviewed feature tree")
			return
		}
	}

	if featureHead != w001BootstrapBase {
		commits, err := planningGrantCommitRangeFrom(root, w001BootstrapBase, featureHead)
		if err != nil || len(commits) == 0 {
			addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_history", "bootstrap feature history must be a nonempty linear chain")
			return
		}
		publicKey, err := readRepoFile(root, wave1PlanningGrantKey)
		if err != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
			addFinding(findings, wave1PlanningGrantKey, "public.w001_bootstrap_commit_signature", "bootstrap commits require the pinned genesis signer")
			return
		}
		previous := w001BootstrapBase
		authorized := w001BootstrapPreclaimPathSet()
		for _, commit := range commits {
			if len(commit.parents) != 1 || commit.parents[0] != previous {
				addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_topology", "bootstrap feature history must be a contiguous one-parent chain")
				return
			}
			object, err := planningGrantGitOutput(root, "cat-file", "commit", commit.id)
			if err != nil || verifyPlanningGrantCommit(object, publicKey) != nil {
				addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_commit_signature", "every bootstrap feature commit must carry the pinned SSH signature")
				return
			}
			paths, err := planningGrantGitOutput(root, "diff-tree", "--no-commit-id", "--no-renames", "--no-ext-diff", "--no-textconv", "--name-only", "-z", "-r", commit.id+"^", commit.id)
			normalized, normalizeErr := normalizedPlanningGrantGitPaths(paths)
			if err != nil || normalizeErr != nil || !planningGrantPathsAllowed(normalized, authorized) {
				addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_scope", "a bootstrap feature commit includes a path outside its signed preclaim scope")
				return
			}
			previous = commit.id
		}
	}

	tracked, err := planningGrantGitOutput(root, "diff", "--no-renames", "--no-ext-diff", "--no-textconv", "--name-only", "-z", "HEAD", "--")
	if err != nil {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_scope", "current tracked bootstrap paths cannot be enumerated")
		return
	}
	untracked, err := planningGrantGitOutput(root, "ls-files", "--others", "--exclude-standard", "-z", "--")
	if err != nil {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_scope", "current untracked bootstrap paths cannot be enumerated")
		return
	}
	paths, err := normalizedPlanningGrantGitPaths(tracked, untracked)
	if err != nil || !planningGrantPathsAllowed(paths, w001BootstrapPreclaimPathSet()) {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_scope", "current changes include a path outside the signed preclaim scope")
	}
}

func w001BootstrapPreclaimPathSet() map[string]bool {
	authorized := make(map[string]bool, len(w001BootstrapGrantSequences["grant.preclaimPaths"]))
	for _, path := range w001BootstrapGrantSequences["grant.preclaimPaths"] {
		authorized[path] = true
	}
	return authorized
}

func w001BootstrapGitHubCheckout(root, head, branch string, findings *[]Finding) (string, bool, bool) {
	if os.Getenv("CI") != "true" || os.Getenv("GITHUB_ACTIONS") != "true" || os.Getenv("RUNNER_ENVIRONMENT") != "github-hosted" ||
		os.Getenv("GITHUB_REPOSITORY") != planningGrantRepository || os.Getenv("GITHUB_WORKFLOW") != planningGrantWorkflow ||
		os.Getenv("GITHUB_JOB") != planningGrantWorkflowJob || os.Getenv("GITHUB_SHA") != head ||
		!samePlanningGrantRepositoryRoot(root, os.Getenv("GITHUB_WORKSPACE")) {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_runner", "GitHub bootstrap checkout lacks canonical runner identity")
		return "", false, false
	}
	if _, ok := parsePositiveInt(os.Getenv("GITHUB_RUN_ID")); !ok {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_runner", "GitHub bootstrap run ID is invalid")
		return "", false, false
	}
	if _, ok := parsePositiveInt(os.Getenv("GITHUB_RUN_ATTEMPT")); !ok {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_runner", "GitHub bootstrap run attempt is invalid")
		return "", false, false
	}
	workflowRef := os.Getenv("GITHUB_WORKFLOW_REF")
	workflowPrefix := planningGrantRepository + "/" + planningGrantWorkflowPath + "@"
	workflow, err := readRepoFile(root, planningGrantWorkflowPath)
	if err != nil || fileSHA256(workflow) != canonicalFoundationWorkflowSHA256 || !strings.HasPrefix(workflowRef, workflowPrefix) {
		addFinding(findings, planningGrantWorkflowPath, "public.w001_bootstrap_workflow", "bootstrap CI requires the pinned protected workflow")
		return "", false, false
	}
	event, ok := readPlanningGrantGitHubEvent(os.Getenv("GITHUB_EVENT_PATH"))
	if !ok || event.Repository.FullName != planningGrantRepository {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_event", "bootstrap CI event identity is invalid")
		return "", false, false
	}
	switch os.Getenv("GITHUB_EVENT_NAME") {
	case "pull_request":
		ref := os.Getenv("GITHUB_REF")
		if branch != "" || !validPlanningGrantPullRequestRef(os.Getenv("GITHUB_REF")) || os.Getenv("GITHUB_HEAD_REF") != w001BootstrapBranch ||
			os.Getenv("GITHUB_BASE_REF") != "main" || event.PullRequest == nil || event.PullRequest.Head.Ref != w001BootstrapBranch ||
			event.PullRequest.Base.Ref != "main" || event.PullRequest.Base.SHA != w001BootstrapBase ||
			!sha1Pattern.MatchString(event.PullRequest.Head.SHA) ||
			!validAdvisoryPullRequestMergeSHA(event.PullRequest.MergeCommitSHA) {
			addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_event", "pull-request event does not bind the signed branch and base")
			return "", false, false
		}
		workflowSuffix := strings.TrimPrefix(workflowRef, workflowPrefix)
		if workflowSuffix != ref && workflowSuffix != "refs/heads/main" {
			addFinding(findings, planningGrantWorkflowPath, "public.w001_bootstrap_workflow", "pull-request workflow ref is not canonical")
			return "", false, false
		}
		parents, err := planningGrantCommitParents(root, head)
		if err != nil || len(parents) != 2 || parents[0] != w001BootstrapBase || parents[1] != event.PullRequest.Head.SHA {
			addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_pr_topology", "pull-request checkout must be the exact two-parent synthetic merge")
			return "", false, false
		}
		mergeTree, mergeErr := planningGrantGitOutput(root, "rev-parse", "--verify", head+"^{tree}")
		featureTree, featureErr := planningGrantGitOutput(root, "rev-parse", "--verify", event.PullRequest.Head.SHA+"^{tree}")
		if mergeErr != nil || featureErr != nil || strings.TrimSpace(string(mergeTree)) != strings.TrimSpace(string(featureTree)) {
			addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_pr_tree", "pull-request synthetic merge tree must equal the reviewed feature tree")
			return "", false, false
		}
		return event.PullRequest.Head.SHA, true, false
	case "push":
		if branch != "" && branch != "main" || os.Getenv("GITHUB_REF") != "refs/heads/main" || os.Getenv("GITHUB_REF_PROTECTED") != "true" ||
			os.Getenv("GITHUB_HEAD_REF") != "" || os.Getenv("GITHUB_BASE_REF") != "" || workflowRef != workflowPrefix+"refs/heads/main" ||
			event.Ref != "refs/heads/main" || event.Before != w001BootstrapBase || event.After != head || event.HeadCommit == nil || event.HeadCommit.ID != head || event.PullRequest != nil {
			addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_event", "protected-main event does not bind the signed bootstrap base and squash")
			return "", false, false
		}
		return head, true, true
	default:
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_event", "unsupported GitHub event for bootstrap publication")
		return "", false, false
	}
}

func checkW001BootstrapReviewTag(root, expectedFeatureHead string, findings *[]Finding) (string, bool) {
	ref := "refs/tags/" + w001BootstrapReviewTag
	objectID, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{tag}")
	if err != nil || !sha1Pattern.MatchString(strings.TrimSpace(string(objectID))) {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_tag", "bootstrap CI requires the signed immutable review tag")
		return "", false
	}
	object, err := planningGrantGitOutput(root, "cat-file", "tag", strings.TrimSpace(string(objectID)))
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	if err != nil || keyErr != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_tag", "bootstrap review tag cannot be verified with the pinned key")
		return "", false
	}
	target, err := verifyPinnedPlanningGrantTag(object, publicKey, w001BootstrapReviewTag, w001BootstrapReviewTagMessage)
	if err != nil {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_tag", "bootstrap review tag must be an exact pinned-signer tree attestation")
		return "", false
	}
	if expectedFeatureHead != "" && expectedFeatureHead != target {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_tag_target", "review tag must target the immutable feature head")
		return "", false
	}
	if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", w001BootstrapBase, target); err != nil || target == w001BootstrapBase {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_tag_target", "review tag target must preserve nonempty bootstrap feature history")
		return "", false
	}
	return target, true
}

func checkWave1DirectMainTransitionGitDiff(root string, findings *[]Finding) {
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(wave1PRFallbackPath))); err == nil {
		checkWave1PRFallbackGitDiff(root, findings)
		return
	} else if !os.IsNotExist(err) {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_state", "PR-fallback addendum state cannot be established")
		return
	}
	plan, err := readRepoFile(root, canonicalActivePlan)
	if err != nil {
		addFinding(findings, canonicalActivePlan, "public.direct_main_transition_phase", "active plan phase cannot be established")
		return
	}
	phaseMatches := planPhaseLine.FindAllSubmatch(plan, -1)
	if len(phaseMatches) != 1 || string(phaseMatches[0][1]) != planPhaseContractPublication {
		addFinding(findings, canonicalActivePlan, "public.direct_main_transition_phase", "transition authority permits preparation only while the canonical Bead remains backlog and unclaimed")
		return
	}

	resolvedBase, err := planningGrantGitOutput(root, "rev-parse", "--verify", wave1PublishedMain+"^{commit}")
	if err != nil || strings.TrimSpace(string(resolvedBase)) != wave1PublishedMain {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_base", "the exact verified publication commit must resolve locally")
		return
	}
	baseTree, err := planningGrantGitOutput(root, "rev-parse", "--verify", wave1PublishedMain+"^{tree}")
	if err != nil || strings.TrimSpace(string(baseTree)) != wave1PublishedTree {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_base_tree", "the verified publication base tree does not match the signed transition")
		return
	}
	if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", wave1PublishedMain, "HEAD"); err != nil {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_ancestry", "the verified publication commit must be an ancestor of HEAD")
		return
	}
	if !checkWave1PriorPublicationTag(root, findings) || !checkWave1V3PublicationTag(root, findings) {
		return
	}
	headOutput, err := planningGrantGitOutput(root, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil || !sha1Pattern.MatchString(strings.TrimSpace(string(headOutput))) {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_head", "HEAD must resolve to one exact commit")
		return
	}
	head := strings.TrimSpace(string(headOutput))
	if !checkWave1DirectMainCheckout(root, head, findings) {
		return
	}

	commits, err := planningGrantCommitRangeFrom(root, wave1PublishedMain, head)
	if err != nil {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_history", "post-publication commit ancestry cannot be enumerated")
		return
	}
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.direct_main_transition_commit_signature", "post-publication commits require the pinned genesis signer")
		return
	}
	authorized := make(map[string]bool, len(wave1DirectMainGrantSequences["grant.authorizedPaths"]))
	for _, path := range wave1DirectMainGrantSequences["grant.authorizedPaths"] {
		authorized[path] = true
	}
	previous := wave1PublishedMain
	for _, commit := range commits {
		if len(commit.parents) != 1 || commit.parents[0] != previous {
			addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_topology", "post-publication main history must be a contiguous one-parent chain")
			return
		}
		object, err := planningGrantGitOutput(root, "cat-file", "commit", commit.id)
		if err != nil || verifyPlanningGrantCommit(object, publicKey) != nil {
			addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_commit_signature", "every post-publication commit must carry the pinned SSH signature")
			return
		}
		paths, err := planningGrantGitOutput(root, "diff-tree", "--no-commit-id", "--no-renames", "--no-ext-diff", "--no-textconv", "--name-only", "-z", "-r", commit.id+"^", commit.id)
		if err != nil {
			addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_paths", "post-publication commit paths cannot be enumerated")
			return
		}
		normalized, err := normalizedPlanningGrantGitPaths(paths)
		if err != nil || !planningGrantPathsAllowed(normalized, authorized) {
			addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_scope", "a post-publication commit includes a path outside the signed transition authority")
			return
		}
		previous = commit.id
	}

	tracked, err := planningGrantGitOutput(root, "diff", "--no-renames", "--no-ext-diff", "--no-textconv", "--name-only", "-z", "HEAD", "--")
	if err != nil {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_paths", "current index and worktree paths cannot be enumerated")
		return
	}
	untracked, err := planningGrantGitOutput(root, "ls-files", "--others", "--exclude-standard", "-z", "--")
	if err != nil {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_paths", "untracked transition paths cannot be enumerated")
		return
	}
	paths, err := normalizedPlanningGrantGitPaths(tracked, untracked)
	if err != nil || !planningGrantPathsAllowed(paths, authorized) {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_scope", "current changes include a path outside the signed transition authority")
	}
}

func checkWave1PRFallbackGitDiff(root string, findings *[]Finding) {
	plan, err := readRepoFile(root, canonicalActivePlan)
	if err != nil {
		addFinding(findings, canonicalActivePlan, "public.pr_fallback_phase", "active plan phase cannot be established")
		return
	}
	phaseMatches := planPhaseLine.FindAllSubmatch(plan, -1)
	if len(phaseMatches) != 1 || string(phaseMatches[0][1]) != planPhaseContractPublication {
		addFinding(findings, canonicalActivePlan, "public.pr_fallback_phase", "PR fallback permits transition preparation only while W-001 remains backlog and unclaimed")
		return
	}
	baseTree, err := planningGrantGitOutput(root, "rev-parse", "--verify", wave1PublishedMain+"^{tree}")
	if err != nil || strings.TrimSpace(string(baseTree)) != wave1PublishedTree {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_base", "the exact verified publication base and tree must resolve locally")
		return
	}
	if !checkWave1PriorPublicationTag(root, findings) || !checkWave1V3PublicationTag(root, findings) {
		return
	}
	mainCIFixActive := false
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(wave1MainCIFixPath))); err == nil {
		mainCIFixActive = true
		before := len(*findings)
		checkWave1MainCIFixAddendum(root, findings)
		if len(*findings) != before {
			return
		}
		if !checkWave1PriorTransitionTag(root, findings) {
			return
		}
	} else if !os.IsNotExist(err) {
		addFinding(findings, wave1MainCIFixPath, "public.pr_fallback_main_ci_state", "main-CI correction addendum state cannot be established")
		return
	}
	fixtureFixActive := false
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(wave1CIFixtureFixPath))); err == nil {
		fixtureFixActive = true
		before := len(*findings)
		checkWave1CIFixtureFixAddendum(root, findings)
		if len(*findings) != before {
			return
		}
		if !checkWave1PriorV2TransitionTag(root, findings) {
			return
		}
	} else if !os.IsNotExist(err) {
		addFinding(findings, wave1CIFixtureFixPath, "public.pr_fallback_fixture_state", "fixture-stabilization addendum state cannot be established")
		return
	}
	headOutput, err := planningGrantGitOutput(root, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil || !sha1Pattern.MatchString(strings.TrimSpace(string(headOutput))) {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_head", "HEAD must resolve to one exact commit")
		return
	}
	head := strings.TrimSpace(string(headOutput))
	checkout, ok := checkWave1PRFallbackCheckout(root, head, findings)
	if !ok {
		return
	}
	historyEnd := head
	if checkout.kind == planningGrantPullRequestMerge {
		historyEnd = checkout.secondParent
	}
	if checkout.kind != planningGrantLocalBranch {
		tagName := wave1TransitionTag
		tagMessage := wave1TransitionTagMessage
		if mainCIFixActive {
			tagName = wave1SuccessorTransitionTag
			tagMessage = wave1SuccessorTagMessage
		}
		if fixtureFixActive {
			tagName = wave1FinalTransitionTag
			tagMessage = wave1FinalTransitionTagMessage
		}
		expectedTarget := historyEnd
		requireDistinct := false
		if checkout.kind == planningGrantMainSquash {
			expectedTarget = ""
			requireDistinct = true
		}
		tagTarget, tagOK := checkWave1TransitionTag(root, tagName, tagMessage, expectedTarget, head, requireDistinct, findings)
		if !tagOK {
			return
		}
		if checkout.kind == planningGrantMainSquash {
			historyEnd = tagTarget
		}
	}
	commits, err := planningGrantCommitRangeFrom(root, wave1PublishedMain, historyEnd)
	if err != nil || len(commits) == 0 {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_history", "fallback feature history must be a nonempty linear chain")
		return
	}
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.pr_fallback_commit_signature", "fallback commits require the pinned genesis signer")
		return
	}
	firstAuthorized := make(map[string]bool, len(wave1DirectMainGrantSequences["grant.authorizedPaths"]))
	for _, path := range wave1DirectMainGrantSequences["grant.authorizedPaths"] {
		firstAuthorized[path] = true
	}
	fallbackAuthorized := make(map[string]bool, len(wave1PRFallbackSequences["addendum.authorizedPaths"]))
	for _, path := range wave1PRFallbackSequences["addendum.authorizedPaths"] {
		fallbackAuthorized[path] = true
	}
	correctionAuthorized := make(map[string]bool, len(wave1MainCIFixSequences["addendum.authorizedPaths"]))
	for _, path := range wave1MainCIFixSequences["addendum.authorizedPaths"] {
		correctionAuthorized[path] = true
	}
	fixtureAuthorized := make(map[string]bool, len(wave1CIFixtureFixSequences["addendum.authorizedPaths"]))
	for _, path := range wave1CIFixtureFixSequences["addendum.authorizedPaths"] {
		fixtureAuthorized[path] = true
	}
	if fixtureFixActive && len(commits) > 4 {
		addFinding(findings, wave1CIFixtureFixPath, "public.pr_fallback_fixture_commit_count", "fixture stabilization permits exactly one successor commit")
		return
	}
	if !fixtureFixActive && mainCIFixActive && len(commits) > 3 {
		addFinding(findings, wave1MainCIFixPath, "public.pr_fallback_main_ci_commit_count", "main-CI correction permits exactly one successor commit")
		return
	}
	previous := wave1PublishedMain
	for index, commit := range commits {
		if len(commit.parents) != 1 || commit.parents[0] != previous {
			addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_topology", "fallback feature history must be a contiguous one-parent chain")
			return
		}
		object, err := planningGrantGitOutput(root, "cat-file", "commit", commit.id)
		if err != nil || verifyPlanningGrantCommit(object, publicKey) != nil {
			addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_commit_signature", "every fallback feature commit must carry the pinned SSH signature")
			return
		}
		paths, err := planningGrantGitOutput(root, "diff-tree", "--no-commit-id", "--no-renames", "--no-ext-diff", "--no-textconv", "--name-only", "-z", "-r", commit.id+"^", commit.id)
		if err != nil {
			addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_paths", "fallback commit paths cannot be enumerated")
			return
		}
		normalized, err := normalizedPlanningGrantGitPaths(paths)
		authorized := fallbackAuthorized
		if index == 0 {
			authorized = firstAuthorized
			if commit.id != wave1PRFallbackFirstCommit {
				addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_first_commit", "the rejected signed transition commit must remain the first exact fallback commit")
				return
			}
			tree, treeErr := planningGrantGitOutput(root, "rev-parse", "--verify", commit.id+"^{tree}")
			if treeErr != nil || strings.TrimSpace(string(tree)) != wave1PRFallbackFirstTree {
				addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_first_commit", "the rejected signed transition tree must remain unchanged")
				return
			}
		} else if index == 1 && mainCIFixActive {
			if commit.id != wave1TransitionReviewedHead {
				addFinding(findings, wave1MainCIFixPath, "public.pr_fallback_main_ci_reviewed_head", "the reviewed changes-requested head must remain the exact second feature commit")
				return
			}
		} else if index == 2 && fixtureFixActive {
			authorized = correctionAuthorized
			if commit.id != wave1CIFixtureReviewedHead {
				addFinding(findings, wave1CIFixtureFixPath, "public.pr_fallback_fixture_reviewed_head", "the failed-CI head must remain the exact third feature commit")
				return
			}
		} else if index >= 3 && fixtureFixActive {
			authorized = fixtureAuthorized
		} else if index >= 2 && mainCIFixActive {
			authorized = correctionAuthorized
		}
		if err != nil || !planningGrantPathsAllowed(normalized, authorized) {
			addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_scope", "a fallback commit includes a path outside its signed authorization phase")
			return
		}
		previous = commit.id
	}
	tracked, err := planningGrantGitOutput(root, "diff", "--no-renames", "--no-ext-diff", "--no-textconv", "--name-only", "-z", "HEAD", "--")
	if err != nil {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_paths", "current index and worktree paths cannot be enumerated")
		return
	}
	untracked, err := planningGrantGitOutput(root, "ls-files", "--others", "--exclude-standard", "-z", "--")
	if err != nil {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_paths", "untracked fallback paths cannot be enumerated")
		return
	}
	paths, err := normalizedPlanningGrantGitPaths(tracked, untracked)
	liveAuthorized := fallbackAuthorized
	if fixtureFixActive {
		liveAuthorized = fixtureAuthorized
	} else if mainCIFixActive {
		liveAuthorized = correctionAuthorized
	}
	if err != nil || !planningGrantPathsAllowed(paths, liveAuthorized) {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_scope", "current changes include a path outside the signed PR fallback")
	}
}

func checkWave1PRFallbackCheckout(root, head string, findings *[]Finding) (planningGrantCheckout, bool) {
	branchOutput, branchErr := planningGrantGitOutput(root, "symbolic-ref", "--quiet", "--short", "HEAD")
	branch := strings.TrimSpace(string(branchOutput))
	if os.Getenv("GITHUB_ACTIONS") != "true" {
		if branchErr != nil || branch != wave1PRFallbackBranch {
			addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_branch", "local fallback work must use the exact signed branch")
			return planningGrantCheckout{}, false
		}
		return planningGrantCheckout{kind: planningGrantLocalBranch, expectedHead: head}, true
	}
	if branchErr == nil && branch != "main" || branchErr != nil && branch != "" {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_branch", "GitHub fallback branch state is ambiguous")
		return planningGrantCheckout{}, false
	}
	if os.Getenv("CI") != "true" || os.Getenv("RUNNER_ENVIRONMENT") != "github-hosted" ||
		os.Getenv("GITHUB_REPOSITORY") != planningGrantRepository || os.Getenv("GITHUB_WORKFLOW") != planningGrantWorkflow ||
		os.Getenv("GITHUB_JOB") != planningGrantWorkflowJob || os.Getenv("GITHUB_SHA") != head ||
		!samePlanningGrantRepositoryRoot(root, os.Getenv("GITHUB_WORKSPACE")) {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_runner", "GitHub fallback CI lacks canonical immutable runner facts")
		return planningGrantCheckout{}, false
	}
	if _, ok := parsePositiveInt(os.Getenv("GITHUB_RUN_ID")); !ok {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_runner", "GitHub fallback run ID is invalid")
		return planningGrantCheckout{}, false
	}
	if _, ok := parsePositiveInt(os.Getenv("GITHUB_RUN_ATTEMPT")); !ok {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_runner", "GitHub fallback attempt is invalid")
		return planningGrantCheckout{}, false
	}
	workflow, err := readRepoFile(root, planningGrantWorkflowPath)
	if err != nil || fileSHA256(workflow) != canonicalFoundationWorkflowSHA256 {
		addFinding(findings, planningGrantWorkflowPath, "public.pr_fallback_workflow", "fallback workflow bytes do not match the pinned digest")
		return planningGrantCheckout{}, false
	}
	event, ok := readPlanningGrantGitHubEvent(os.Getenv("GITHUB_EVENT_PATH"))
	if !ok || event.Repository.FullName != planningGrantRepository {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_event", "fallback event payload is not canonical")
		return planningGrantCheckout{}, false
	}
	workflowPrefix := planningGrantRepository + "/" + planningGrantWorkflowPath + "@"
	switch os.Getenv("GITHUB_EVENT_NAME") {
	case "pull_request":
		ref := os.Getenv("GITHUB_REF")
		if branch != "" || !validPlanningGrantPullRequestRef(ref) || os.Getenv("GITHUB_HEAD_REF") != wave1PRFallbackBranch ||
			os.Getenv("GITHUB_BASE_REF") != "main" || event.PullRequest == nil || event.PullRequest.Head.Ref != wave1PRFallbackBranch ||
			event.PullRequest.Base.Ref != "main" || event.PullRequest.Base.SHA != wave1PublishedMain ||
			!sha1Pattern.MatchString(event.PullRequest.Head.SHA) ||
			!validAdvisoryPullRequestMergeSHA(event.PullRequest.MergeCommitSHA) {
			addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_event", "pull-request event does not bind the exact fallback base and branch")
			return planningGrantCheckout{}, false
		}
		workflowRef := os.Getenv("GITHUB_WORKFLOW_REF")
		if workflowRef != workflowPrefix+ref && workflowRef != workflowPrefix+"refs/heads/main" {
			addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_runner", "pull-request workflow ref is not canonical")
			return planningGrantCheckout{}, false
		}
		parents, err := planningGrantCommitParents(root, head)
		if err != nil || len(parents) != 2 || parents[0] != wave1PublishedMain || parents[1] != event.PullRequest.Head.SHA {
			addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_topology", "pull-request checkout must be the exact ordered two-parent merge")
			return planningGrantCheckout{}, false
		}
		mergeTree, mergeErr := planningGrantGitOutput(root, "rev-parse", "--verify", head+"^{tree}")
		headTree, headErr := planningGrantGitOutput(root, "rev-parse", "--verify", event.PullRequest.Head.SHA+"^{tree}")
		if mergeErr != nil || headErr != nil || strings.TrimSpace(string(mergeTree)) != strings.TrimSpace(string(headTree)) {
			addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_tree", "pull-request merge tree must equal the immutable fallback head tree")
			return planningGrantCheckout{}, false
		}
		return planningGrantCheckout{kind: planningGrantPullRequestMerge, expectedHead: head, firstParent: parents[0], secondParent: parents[1]}, true
	case "push":
		if branch != "" && branch != "main" || os.Getenv("GITHUB_REF") != "refs/heads/main" || os.Getenv("GITHUB_REF_PROTECTED") != "true" ||
			os.Getenv("GITHUB_HEAD_REF") != "" || os.Getenv("GITHUB_BASE_REF") != "" || os.Getenv("GITHUB_WORKFLOW_REF") != workflowPrefix+"refs/heads/main" ||
			event.Ref != "refs/heads/main" || event.Before != wave1PublishedMain || event.After != head || event.HeadCommit == nil || event.HeadCommit.ID != head || event.PullRequest != nil {
			addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_event", "protected-main event does not bind the exact fallback squash")
			return planningGrantCheckout{}, false
		}
		parents, ok := checkWave1MainSquashTopology(root, head, findings)
		if !ok {
			return planningGrantCheckout{}, false
		}
		return planningGrantCheckout{kind: planningGrantMainSquash, expectedHead: head, firstParent: parents[0]}, true
	default:
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_event", "only pull-request and protected-main push events are admitted")
		return planningGrantCheckout{}, false
	}
}

func checkWave1MainSquashTopology(root, head string, findings *[]Finding) ([]string, bool) {
	parents, err := planningGrantCommitParents(root, head)
	if err != nil || len(parents) != 1 || parents[0] != wave1PublishedMain {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_topology", "protected main must contain one exact squash commit over the signed base")
		return nil, false
	}
	return parents, true
}

func checkWave1PriorTransitionTag(root string, findings *[]Finding) bool {
	ref := "refs/tags/" + wave1TransitionTag
	tagObject, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{tag}")
	if err != nil || strings.TrimSpace(string(tagObject)) != wave1TransitionTagObject {
		addFinding(findings, wave1MainCIFixPath, "public.pr_fallback_v1_tag", "the signed v1 transition tag object must remain unchanged")
		return false
	}
	object, err := planningGrantGitOutput(root, "cat-file", "tag", wave1TransitionTagObject)
	if err != nil {
		addFinding(findings, wave1MainCIFixPath, "public.pr_fallback_v1_tag", "the signed v1 transition tag cannot be read")
		return false
	}
	publicKey, err := readRepoFile(root, wave1PlanningGrantKey)
	if err != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		addFinding(findings, wave1PlanningGrantKey, "public.pr_fallback_v1_tag", "v1 transition tag requires the pinned genesis key")
		return false
	}
	target, err := verifyPinnedPlanningGrantTag(object, publicKey, wave1TransitionTag, wave1TransitionTagMessage)
	if err != nil || target != wave1TransitionReviewedHead {
		addFinding(findings, wave1MainCIFixPath, "public.pr_fallback_v1_tag", "the signed v1 transition tag target must remain unchanged")
		return false
	}
	tree, err := planningGrantGitOutput(root, "rev-parse", "--verify", target+"^{tree}")
	if err != nil || strings.TrimSpace(string(tree)) != wave1TransitionReviewedTree {
		addFinding(findings, wave1MainCIFixPath, "public.pr_fallback_v1_tag", "the signed v1 transition tag tree must remain unchanged")
		return false
	}
	return true
}

func checkWave1PriorV2TransitionTag(root string, findings *[]Finding) bool {
	ref := "refs/tags/" + wave1SuccessorTransitionTag
	tagObject, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{tag}")
	if err != nil || strings.TrimSpace(string(tagObject)) != wave1SuccessorTagObject {
		addFinding(findings, wave1CIFixtureFixPath, "public.pr_fallback_v2_tag", "the signed v2 transition tag object must remain unchanged")
		return false
	}
	object, err := planningGrantGitOutput(root, "cat-file", "tag", wave1SuccessorTagObject)
	if err != nil {
		addFinding(findings, wave1CIFixtureFixPath, "public.pr_fallback_v2_tag", "the signed v2 transition tag cannot be read")
		return false
	}
	publicKey, err := readRepoFile(root, wave1PlanningGrantKey)
	if err != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		addFinding(findings, wave1PlanningGrantKey, "public.pr_fallback_v2_tag", "v2 transition tag requires the pinned genesis key")
		return false
	}
	target, err := verifyPinnedPlanningGrantTag(object, publicKey, wave1SuccessorTransitionTag, wave1SuccessorTagMessage)
	if err != nil || target != wave1CIFixtureReviewedHead {
		addFinding(findings, wave1CIFixtureFixPath, "public.pr_fallback_v2_tag", "the signed v2 transition tag target must remain unchanged")
		return false
	}
	tree, err := planningGrantGitOutput(root, "rev-parse", "--verify", target+"^{tree}")
	if err != nil || strings.TrimSpace(string(tree)) != wave1CIFixtureReviewedTree {
		addFinding(findings, wave1CIFixtureFixPath, "public.pr_fallback_v2_tag", "the signed v2 transition tag tree must remain unchanged")
		return false
	}
	return true
}

func checkWave1TransitionTag(root, tagName, tagMessage, expectedTarget, expectedTreeCommit string, requireDistinct bool, findings *[]Finding) (string, bool) {
	ref := "refs/tags/" + tagName
	tagObject, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{tag}")
	if err != nil {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_tag", "canonical CI requires the signed transition tag")
		return "", false
	}
	object, err := planningGrantGitOutput(root, "cat-file", "tag", strings.TrimSpace(string(tagObject)))
	if err != nil {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_tag", "signed transition tag cannot be read")
		return "", false
	}
	publicKey, err := readRepoFile(root, wave1PlanningGrantKey)
	if err != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		addFinding(findings, wave1PlanningGrantKey, "public.pr_fallback_tag", "transition tag requires the pinned genesis key")
		return "", false
	}
	target, err := verifyPinnedPlanningGrantTag(object, publicKey, tagName, tagMessage)
	if err != nil || expectedTarget != "" && target != expectedTarget {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_tag", "signed transition tag must target the current reviewed feature head")
		return "", false
	}
	if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", wave1TransitionReviewedHead, target); err != nil {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_tag", "signed transition tag target must preserve the reviewed fallback history")
		return "", false
	}
	if requireDistinct && target == expectedTreeCommit {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_tag", "squash-main commit must remain distinct from the reviewed feature head")
		return "", false
	}
	targetTree, targetErr := planningGrantGitOutput(root, "rev-parse", "--verify", target+"^{tree}")
	expectedTree, expectedErr := planningGrantGitOutput(root, "rev-parse", "--verify", expectedTreeCommit+"^{tree}")
	if targetErr != nil || expectedErr != nil || strings.TrimSpace(string(targetTree)) != strings.TrimSpace(string(expectedTree)) {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_tag", "signed transition tag tree must equal the reviewed fallback tree")
		return "", false
	}
	return target, true
}

func checkWave1V3PublicationTag(root string, findings *[]Finding) bool {
	ref := "refs/tags/" + wave1PublicationTag
	tagObject, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{tag}")
	if err != nil || strings.TrimSpace(string(tagObject)) != "b53728e3a57e6dc0d57151aa7f0bed8e44aaaa2f" {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_v3_tag", "the signed v3 publication tag object must remain unchanged")
		return false
	}
	object, err := planningGrantGitOutput(root, "cat-file", "tag", strings.TrimSpace(string(tagObject)))
	if err != nil {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_v3_tag", "the signed v3 publication tag cannot be read")
		return false
	}
	publicKey, err := readRepoFile(root, wave1PlanningGrantKey)
	if err != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		addFinding(findings, wave1PlanningGrantKey, "public.direct_main_transition_v3_tag", "the signed v3 publication tag requires the pinned genesis key")
		return false
	}
	target, err := verifyPlanningGrantTag(object, publicKey)
	if err != nil || target != "7e6b765c284788442553d40792db0afb128c4872" {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_v3_tag", "the signed v3 publication tag target must remain unchanged")
		return false
	}
	tree, err := planningGrantGitOutput(root, "rev-parse", "--verify", target+"^{tree}")
	if err != nil || strings.TrimSpace(string(tree)) != wave1PublishedTree {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_v3_tag", "the signed v3 publication tree must remain unchanged")
		return false
	}
	return true
}

func checkWave1DirectMainCheckout(root, head string, findings *[]Finding) bool {
	branchOutput, branchErr := planningGrantGitOutput(root, "symbolic-ref", "--quiet", "--short", "HEAD")
	branch := strings.TrimSpace(string(branchOutput))
	if os.Getenv("GITHUB_ACTIONS") != "true" {
		if branchErr != nil || branch != "main" {
			addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_branch", "local transition work must use the exact main branch")
			return false
		}
		return true
	}
	if branchErr == nil && branch != "main" || branchErr != nil && branch != "" {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_branch", "protected-main CI branch state is ambiguous")
		return false
	}
	if os.Getenv("CI") != "true" || os.Getenv("RUNNER_ENVIRONMENT") != "github-hosted" ||
		os.Getenv("GITHUB_REPOSITORY") != planningGrantRepository || os.Getenv("GITHUB_WORKFLOW") != planningGrantWorkflow ||
		os.Getenv("GITHUB_JOB") != planningGrantWorkflowJob || os.Getenv("GITHUB_SHA") != head ||
		os.Getenv("GITHUB_EVENT_NAME") != "push" || os.Getenv("GITHUB_REF") != "refs/heads/main" ||
		os.Getenv("GITHUB_REF_PROTECTED") != "true" || os.Getenv("GITHUB_HEAD_REF") != "" || os.Getenv("GITHUB_BASE_REF") != "" ||
		!samePlanningGrantRepositoryRoot(root, os.Getenv("GITHUB_WORKSPACE")) {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_runner", "protected-main CI lacks canonical immutable runner facts")
		return false
	}
	if _, ok := parsePositiveInt(os.Getenv("GITHUB_RUN_ID")); !ok {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_runner", "protected-main CI run ID is invalid")
		return false
	}
	if _, ok := parsePositiveInt(os.Getenv("GITHUB_RUN_ATTEMPT")); !ok {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_runner", "protected-main CI attempt is invalid")
		return false
	}
	workflowRef := os.Getenv("GITHUB_WORKFLOW_REF")
	if workflowRef != planningGrantRepository+"/"+planningGrantWorkflowPath+"@refs/heads/main" {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_runner", "protected-main workflow ref is not canonical")
		return false
	}
	workflow, err := readRepoFile(root, planningGrantWorkflowPath)
	if err != nil || fileSHA256(workflow) != canonicalFoundationWorkflowSHA256 {
		addFinding(findings, planningGrantWorkflowPath, "public.direct_main_transition_workflow", "protected-main workflow bytes do not match the pinned digest")
		return false
	}
	event, ok := readPlanningGrantGitHubEvent(os.Getenv("GITHUB_EVENT_PATH"))
	if !ok || event.Repository.FullName != planningGrantRepository || event.Ref != "refs/heads/main" || event.After != head ||
		event.HeadCommit == nil || event.HeadCommit.ID != head || event.PullRequest != nil {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_event", "protected-main event does not bind the exact pushed commit")
		return false
	}
	parents, err := planningGrantCommitParents(root, head)
	if err != nil || len(parents) != 1 || event.Before != parents[0] {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_event", "protected-main event before SHA must equal the pushed commit's sole parent")
		return false
	}
	return true
}

func planningGrantCommitRangeFrom(root, base, end string) ([]planningGrantCommit, error) {
	output, err := planningGrantGitOutput(root, "rev-list", "--no-abbrev-commit", "--reverse", "--topo-order", "--parents", base+".."+end)
	if err != nil {
		return nil, err
	}
	var commits []planningGrantCommit
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 || !sha1Pattern.MatchString(fields[0]) || !sha1Pattern.MatchString(fields[1]) {
			return nil, fmt.Errorf("post-publication history is not a linear commit chain")
		}
		commits = append(commits, planningGrantCommit{id: fields[0], parents: []string{fields[1]}})
	}
	return commits, nil
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
			!validAdvisoryPullRequestMergeSHA(event.PullRequest.MergeCommitSHA) ||
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

func validAdvisoryPullRequestMergeSHA(value string) bool {
	return value == "" || sha1Pattern.MatchString(value)
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
	return verifyPinnedPlanningGrantTag(object, publicKey, wave1PublicationTag, wave1PublicationTagMessage)
}

func verifyPinnedPlanningGrantTag(object, publicKey []byte, expectedTag, expectedMessage string) (string, error) {
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
			if fields[1] != expectedTag {
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
	if string(signed[headerEnd+2:]) != expectedMessage+"\n" {
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
