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
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	marsRepository                    = "https://github.com/greaveselliott/MARS"
	marsCommit                        = "f55d129bfc794510ca485bb54fc0a35c7b04a700"
	beadsCommit                       = "6c124203e771433a3550c348771a5b5e27fd3c21"
	doltCommit                        = "1bf533220ab0"
	genesisCharterSHA256              = "ccf9b8bd8d54140e352480f5066d6d72da4c7e8cab31427b3eaa0f8afa154917"
	genesisSignatureSHA256            = "cf2a77d6a45f8614c5c54a78e2184b1294290deab22e4a9d8c9bfa3a680a756a"
	genesisVerificationMaterialDigest = "4d60e17afffe09a14c141876bf31d8ce2615270ef664ecd67f8e88e54c7e08df"
	genesisSignerFingerprint          = "SHA256:i5VSHF257DhXJ5l/9oOUGHnT2mrqgXYSMryQHRsSBx8"
	genesisEffectChainSHA256          = "b5191967f91301713efb3a600fbd0efb6b22c3e43ab733390c195d2d21619667"
	genesisEffectRoot                 = "71dcdf4ca80379b5749583199dde4fb0b9ec19a5396eb2364fb0f4492b97cee6"
)

var (
	sha1Pattern   = regexp.MustCompile(`^[0-9a-f]{40}$`)
	sha256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

var requiredMARSSourceBlobs = map[string]string{
	"AGENTS.md": "cc9dece7266b66d3ae163136c7c7c31115481aa2",
	"LICENSE":   "1f7a565fab7b7507e68464305eee73e6908f91f3",
	"NOTICE":    "aee2de8c658e8a384d96ed97242a1a265a34b81a",
	"docs/design-docs/code-documentation-map.md":                   "dd41cdffe4360b0c5460540180d317292081165c",
	"docs/design-docs/convergence-state-machine.md":                "a185f45960208932c7bbeb40b8160eccd6c3c106",
	"docs/design-docs/conversation-as-system-record.md":            "d70f3ea1f8cda9873bd8c05af7fe2c83c6966c02",
	"docs/design-docs/delivery-operating-model.md":                 "dcbc923f7f9286d81211ee44e4a2c66be6adfeb6",
	"docs/design-docs/documentation-sync-architecture.md":          "4711a8006102c0f69c37590c953a0049ddbb9414",
	"docs/design-docs/dogfood-and-decisions.md":                    "76093045582abcb0c7a256bfe9a59b10f01a92be",
	"docs/design-docs/foundation-deployed-harness-architecture.md": "027f677ba3bc4c3327b19f5dbc9c104cd0afc77d",
	"docs/design-docs/foundation-operating-model.md":               "835e60eb0cf12627de61ff34b6250fcee61a24c2",
	"docs/design-docs/guardrails.md":                               "bb093d800ef90b53dab0878e2ab7116888fa7ac3",
	"docs/design-docs/harness-operating-model.md":                  "e54e023ee0ae6cb2b17aeaca3b14f0a961218758",
	"docs/design-docs/orchestrated-organization-layer.md":          "3066d1cf36aa01dba2da57dcad5ec22e25fec272",
	"docs/design-docs/product-spec-governance.md":                  "b6b3aaad84ec2b7b8fcbde08b8471f7ba2d1ce8e",
	"docs/exec-plans/README.md":                                    "90099e42d98629294e4365cfa959f5c380d2fe6e",
	"docs/features/README.md":                                      "34aaa76f85192188fba424d0f647b3a589f61085",
	"docs/goals/README.md":                                         "3ba1795b1f0998de2fb6cc56a57783495583d6bf",
	"docs/roles/ROLES.md":                                          "d65b4dc4b068ee7c5561a021aa82a2d93d34e6b5",
	"docs/tickets/README.md":                                       "c52c63f02d9adc2df21865f587745b2b980e0df5",
}

// CheckDoctrine validates the immutable root of trust, pinned provenance, and
// least-authority role declarations without contacting an upstream service.
func CheckDoctrine(repo string) ([]Finding, error) {
	root, err := repositoryRoot(repo)
	if err != nil {
		return nil, err
	}
	var findings []Finding
	checkGenesis(root, &findings)
	checkGenesisEffectChain(root, &findings)
	checkBootstrapClaimAttestation(root, &findings)
	checkWave1PlanningGrant(root, &findings)
	checkMARSProvenance(root, &findings)
	checkRoleManifest(root, &findings)
	checkClaimVerificationOrder(root, &findings)
	sortFindings(findings)
	return findings, nil
}

func checkBootstrapClaimAttestation(root string, findings *[]Finding) {
	const path = ".harness/claims/H-001.yaml"
	data, err := readRepoFile(root, path)
	if err != nil {
		addFinding(findings, path, "doctrine.claim_attestation_missing", "signed H-001 bootstrap claim attestation is required")
		return
	}
	values := yamlScalars(data)
	expected := map[string]string{
		"schemaVersion":                            "1",
		"kind":                                     "MARS3BootstrapClaimAttestation",
		"authority.system":                         "beads-dolt",
		"authority.version":                        "1.2.2",
		"authority.sourceCommit":                   beadsCommit,
		"authority.doltSourceCommit":               doltCommit,
		"authority.ledgerBranch":                   "main",
		"authority.ledgerHead":                     "blsidb8htct7d687cijiqcp51488jqo5",
		"authority.claimCheckpoint":                "kvofc5q57reond5aki5pgdcgfog8u7dr",
		"claim.bead":                               "M3-H001",
		"claim.displayId":                          "H-001",
		"claim.nativeStatus":                       "in_progress",
		"claim.lifecycleState":                     "in-progress",
		"claim.assignee":                           "foundation-maintainer",
		"claim.coordinator":                        "delivery-orchestrator",
		"claim.workType":                           "enabler",
		"claim.failureOwnership":                   "foundation",
		"claim.risk":                               "high",
		"claim.dependencyCount":                    "0",
		"claim.goal":                               "G-001",
		"claim.feature":                            "F-001",
		"verification.classification":              "PUBLIC",
		"verification.rawAuthorityPayloadIncluded": "false",
		"integrity.signatureFormat":                "openssh",
		"integrity.signatureNamespace":             "mars3-claim-attestation",
		"integrity.detachedSignature":              "H-001.yaml.sig",
		"integrity.publicKey":                      "../keys/genesis-signing-key.pub",
	}
	for key, wanted := range expected {
		if values[key] != wanted {
			addFinding(findings, path, "doctrine.claim_attestation_value", "%s must be %q", key, wanted)
		}
	}
	for _, required := range []string{
		"F-001-S1", "F-001-S2", "F-001-S3", "F-001-S4", "PD-001", "PD-002", "PD-003",
		".github/**", "AGENTS.md", ".harness/**", "docs/**", "cmd/mars3/**", "internal/doctrine/**",
	} {
		if !strings.Contains(string(data), "- "+required) {
			addFinding(findings, path, "doctrine.claim_attestation_scope", "signed claim attestation must include %s", required)
		}
	}
	signature, signatureErr := readRepoFile(root, ".harness/claims/H-001.yaml.sig")
	publicKey, keyErr := readRepoFile(root, ".harness/keys/genesis-signing-key.pub")
	if signatureErr != nil || keyErr != nil {
		addFinding(findings, path, "doctrine.claim_attestation_integrity", "claim attestation signature and anchored public key are required")
		return
	}
	if err := verifySSHSig(data, signature, publicKey, "mars3-claim-attestation"); err != nil {
		addFinding(findings, path, "doctrine.claim_attestation_signature", "%v", err)
	}
}

func checkClaimVerificationOrder(root string, findings *[]Finding) {
	const (
		claimPath    = ".harness/claims/H-001.yaml"
		manifestPath = ".harness/manifest.yaml"
	)
	claim, claimErr := readRepoFile(root, claimPath)
	manifest, manifestErr := readRepoFile(root, manifestPath)
	if claimErr != nil || manifestErr != nil {
		return
	}
	checkClaimVerificationOrderData(claimPath, claim, manifest, findings)
}

func checkClaimVerificationOrderData(claimPath string, claim, manifest []byte, findings *[]Finding) {
	order, found, valid := yamlStringSequence(claim, "verification.order")
	if !found || !valid || len(order) == 0 {
		addFinding(findings, claimPath, "doctrine.claim_verification_order", "verification.order must be one non-empty scalar sequence")
		return
	}
	canonicalOrder := [...]string{"qa", "security-reviewer", "delivery-orchestrator"}
	exactOrder := len(order) == len(canonicalOrder)
	if exactOrder {
		for index := range canonicalOrder {
			if order[index] != canonicalOrder[index] {
				exactOrder = false
				break
			}
		}
	}
	if !exactOrder {
		addFinding(findings, claimPath, "doctrine.claim_verification_order_exact", "verification.order must equal [qa, security-reviewer, delivery-orchestrator]")
	}

	executable := executableIdentityRegistry(manifest)
	seen := make(map[string]bool, len(order))
	for position, identity := range order {
		if seen[identity] {
			addFinding(findings, claimPath, "doctrine.claim_verifier_duplicate", "verification.order entry %q is duplicated", identity)
			continue
		}
		seen[identity] = true
		if !executable[identity] {
			addFinding(findings, claimPath, "doctrine.claim_verifier_unroutable", "verification.order entry %d (%q) is not declared in the executable role/profile registry", position+1, identity)
		}
	}
}

func executableIdentityRegistry(manifest []byte) map[string]bool {
	principals := make(map[string]bool)
	for _, section := range []string{"roles", "foundation_roles"} {
		for _, role := range parseRoleDeclarations(manifest, section) {
			if role.id != "" {
				principals[role.id] = true
			}
		}
	}

	executable := make(map[string]bool, len(principals))
	for principal := range principals {
		executable[principal] = true
	}
	for _, profile := range parseRoleDeclarations(manifest, "profiles") {
		if profile.id != "" && profile.principalID != "" && principals[profile.principalID] {
			executable[profile.id] = true
		}
	}
	return executable
}

func checkGenesis(root string, findings *[]Finding) {
	const path = ".harness/genesis.yaml"
	data, err := readRepoFile(root, path)
	if err != nil {
		addFinding(findings, path, "doctrine.genesis_missing", "signed genesis charter is required")
		return
	}
	if fileSHA256(data) != genesisCharterSHA256 {
		addFinding(findings, path, "doctrine.genesis_anchor", "genesis charter does not match the independently pinned foundation digest")
	}
	values := yamlScalars(data)
	expected := map[string]string{
		"schemaVersion":                  "1",
		"kind":                           "MARS3GenesisCharter",
		"project.name":                   "MARS-3",
		"project.repository":             "https://github.com/greaveselliott/MARS-3",
		"project.visibility":             "public",
		"project.license":                "Apache-2.0",
		"authority.system":               "beads-dolt",
		"authority.issue":                "M3-H001",
		"authority.sourceCommit":         beadsCommit,
		"authority.version":              "1.2.2",
		"authority.doltSourceCommit":     doltCommit,
		"doctrine.marsRepository":        marsRepository,
		"doctrine.marsCommit":            marsCommit,
		"plan.goal":                      "G-001",
		"plan.currentTicket":             "H-001",
		"plan.owner":                     "foundation-maintainer",
		"plan.coordinator":               "delivery-orchestrator",
		"plan.workingBranch":             "codex/h-001-doctrine-foundation",
		"security.publicFromFirstCommit": "true",
		"security.autonomousMutation":    "false",
		"security.effectiveTrust":        "observer",
		"security.rawSecretsPermitted":   "false",
		"integrity.signatureFormat":      "openssh",
		"integrity.signatureNamespace":   "mars3-genesis",
	}
	keys := make([]string, 0, len(expected))
	for key := range expected {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if values[key] != expected[key] {
			addFinding(findings, path, "doctrine.genesis_value", "%s must be %q", key, expected[key])
		}
	}

	signaturePath := scalar(values, "integrity.detachedSignature")
	keyPath := scalar(values, "integrity.publicKey")
	if signaturePath == "" || !safeRelativePath(signaturePath) {
		addFinding(findings, path, "doctrine.genesis_signature_path", "integrity.detachedSignature must be a safe relative path")
		return
	}
	if keyPath == "" || !safeRelativePath(keyPath) {
		addFinding(findings, path, "doctrine.genesis_key_path", "integrity.publicKey must be a safe relative path")
		return
	}
	signatureRelative := cleanPublicPath(filepath.Join(".harness", signaturePath))
	keyRelative := cleanPublicPath(filepath.Join(".harness", keyPath))
	signature, signatureErr := readRepoFile(root, signatureRelative)
	publicKey, keyErr := readRepoFile(root, keyRelative)
	if signatureErr != nil {
		addFinding(findings, signatureRelative, "doctrine.genesis_signature_missing", "detached signature is required")
	}
	if keyErr != nil {
		addFinding(findings, keyRelative, "doctrine.genesis_key_missing", "public verification key is required")
	}
	if signatureErr != nil || keyErr != nil {
		return
	}
	if fileSHA256(signature) != genesisSignatureSHA256 {
		addFinding(findings, signatureRelative, "doctrine.genesis_signature_anchor", "genesis signature does not match the independently pinned foundation digest")
	}
	if fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		addFinding(findings, keyRelative, "doctrine.genesis_key_anchor", "genesis public key does not match the independently pinned foundation digest")
	}
	fingerprint, err := openSSHPublicKeyFingerprint(publicKey)
	if err != nil {
		addFinding(findings, keyRelative, "doctrine.genesis_key_invalid", "%v", err)
		return
	}
	if values["integrity.signerFingerprint"] != fingerprint {
		addFinding(findings, path, "doctrine.genesis_fingerprint", "signer fingerprint does not match the declared public key")
	}
	if fingerprint != genesisSignerFingerprint {
		addFinding(findings, keyRelative, "doctrine.genesis_fingerprint_anchor", "genesis signer does not match the independently pinned foundation fingerprint")
	}
	if err := verifySSHSig(data, signature, publicKey, values["integrity.signatureNamespace"]); err != nil {
		addFinding(findings, signatureRelative, "doctrine.genesis_signature_invalid", "%v", err)
	}
}

type genesisEffectChain struct {
	SchemaVersion  int             `json:"schemaVersion"`
	Algorithm      string          `json:"algorithm"`
	CanonicalEvent string          `json:"canonicalEvent"`
	Events         []genesisEffect `json:"events"`
	RootHash       string          `json:"rootHash"`
}

type genesisEffect struct {
	Sequence       int    `json:"sequence"`
	OccurredAt     string `json:"occurredAt"`
	Operation      string `json:"operation"`
	Phase          string `json:"phase"`
	Resource       string `json:"resource"`
	Classification string `json:"classification"`
	PreviousHash   string `json:"previousHash"`
	Hash           string `json:"hash"`
}

func checkGenesisEffectChain(root string, findings *[]Finding) {
	const path = ".harness/generated/genesis-effect-chain.json"
	data, err := readRepoFile(root, path)
	if err != nil {
		addFinding(findings, path, "doctrine.effect_chain_missing", "genesis effect chain is required")
		return
	}
	if fileSHA256(data) != genesisEffectChainSHA256 {
		addFinding(findings, path, "doctrine.effect_chain_anchor", "genesis effect chain does not match the independently pinned foundation digest")
	}
	var chain genesisEffectChain
	if err := decodeStrictJSON(data, &chain); err != nil {
		addFinding(findings, path, "doctrine.effect_chain_schema", "%v", err)
		return
	}
	if chain.SchemaVersion != 1 || chain.Algorithm != "sha256" || chain.CanonicalEvent != "sequence|occurredAt|operation|phase|resource|classification|previousHash" {
		addFinding(findings, path, "doctrine.effect_chain_header", "unsupported effect-chain header")
	}
	if len(chain.Events) != 2 {
		addFinding(findings, path, "doctrine.effect_chain_cardinality", "genesis must contain exactly one intent and one receipt")
	}
	previous := strings.Repeat("0", 64)
	for index, event := range chain.Events {
		if event.Sequence != index+1 {
			addFinding(findings, path, "doctrine.effect_chain_sequence", "event %d has a non-contiguous sequence", index+1)
		}
		if event.PreviousHash != previous {
			addFinding(findings, path, "doctrine.effect_chain_link", "event %d does not reference the prior hash", event.Sequence)
		}
		canonical := fmt.Sprintf("%d|%s|%s|%s|%s|%s|%s", event.Sequence, event.OccurredAt, event.Operation, event.Phase, event.Resource, event.Classification, event.PreviousHash)
		digest := sha256.Sum256([]byte(canonical))
		calculated := hex.EncodeToString(digest[:])
		if event.Hash != calculated {
			addFinding(findings, path, "doctrine.effect_chain_hash", "event %d hash is invalid", event.Sequence)
		}
		if event.Operation != "repository.create" || event.Resource != "greaveselliott/MARS-3" || event.Classification != "PUBLIC" {
			addFinding(findings, path, "doctrine.effect_chain_scope", "event %d has an unexpected scope", event.Sequence)
		}
		expectedPhase := "intent"
		if index == 1 {
			expectedPhase = "receipt"
		}
		if event.Phase != expectedPhase {
			addFinding(findings, path, "doctrine.effect_chain_phase", "event %d must be %s", event.Sequence, expectedPhase)
		}
		previous = event.Hash
	}
	if chain.RootHash != previous || chain.RootHash != genesisEffectRoot || !sha256Pattern.MatchString(chain.RootHash) {
		addFinding(findings, path, "doctrine.effect_chain_root", "rootHash does not match the terminal event")
	}
}

func fileSHA256(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

type marsSourceManifest struct {
	SchemaVersion int `json:"schemaVersion"`
	Generated     struct {
		Source  string `json:"source"`
		Command string `json:"command"`
	} `json:"generated"`
	Upstream struct {
		Repository string `json:"repository"`
		Commit     string `json:"commit"`
		License    string `json:"license"`
	} `json:"upstream"`
	SourceFiles []struct {
		Path       string `json:"path"`
		GitBlob    string `json:"gitBlob"`
		Adaptation string `json:"adaptation"`
	} `json:"sourceFiles"`
	GeneratedScope       []string `json:"generatedScope"`
	ProjectOwnedPaths    []string `json:"projectOwnedPaths"`
	Adaptations          []string `json:"adaptations"`
	SourceOnlyExclusions []string `json:"sourceOnlyExclusions"`
}

func checkMARSProvenance(root string, findings *[]Finding) {
	const path = ".harness/generated/mars/source-manifest.json"
	data, err := readRepoFile(root, path)
	if err != nil {
		addFinding(findings, path, "doctrine.provenance_missing", "pinned MARS source manifest is required")
		return
	}
	var manifest marsSourceManifest
	if err := decodeStrictJSON(data, &manifest); err != nil {
		addFinding(findings, path, "doctrine.provenance_schema", "%v", err)
		return
	}
	if manifest.SchemaVersion != 1 {
		addFinding(findings, path, "doctrine.provenance_version", "schemaVersion must be 1")
	}
	if manifest.Upstream.Repository != marsRepository || manifest.Upstream.Commit != marsCommit || manifest.Upstream.License != "Apache-2.0" {
		addFinding(findings, path, "doctrine.provenance_pin", "upstream repository, commit, and license must match the approved pin")
	}
	if strings.TrimSpace(manifest.Generated.Source) == "" || strings.TrimSpace(manifest.Generated.Command) == "" {
		addFinding(findings, path, "doctrine.provenance_generation", "generated provenance must declare its source and reproducible command")
	}
	if len(manifest.SourceFiles) == 0 {
		addFinding(findings, path, "doctrine.provenance_empty", "sourceFiles must not be empty")
	}
	seenPaths := make(map[string]bool)
	observedBlobs := make(map[string]string)
	seenBlobs := make(map[string]bool)
	for _, source := range manifest.SourceFiles {
		if !safeRelativePath(source.Path) {
			addFinding(findings, path, "doctrine.provenance_source_path", "unsafe source path %q", source.Path)
		}
		if seenPaths[source.Path] {
			addFinding(findings, path, "doctrine.provenance_duplicate_path", "duplicate source path %q", source.Path)
		}
		seenPaths[source.Path] = true
		if !sha1Pattern.MatchString(source.GitBlob) {
			addFinding(findings, path, "doctrine.provenance_blob", "%q does not have a Git blob hash", source.Path)
		}
		if seenBlobs[source.GitBlob] {
			addFinding(findings, path, "doctrine.provenance_duplicate_blob", "duplicate Git blob hash for %q", source.Path)
		}
		if strings.TrimSpace(source.Adaptation) == "" {
			addFinding(findings, path, "doctrine.provenance_adaptation", "%q must declare its local adaptation", source.Path)
		}
		seenBlobs[source.GitBlob] = true
		observedBlobs[source.Path] = source.GitBlob
	}
	for sourcePath, expectedBlob := range requiredMARSSourceBlobs {
		observedBlob, exists := observedBlobs[sourcePath]
		if !exists {
			addFinding(findings, path, "doctrine.provenance_required_source", "required doctrine source %q is missing", sourcePath)
			continue
		}
		if observedBlob != expectedBlob {
			addFinding(findings, path, "doctrine.provenance_required_blob", "required doctrine source %q has an unexpected blob ID", sourcePath)
		}
	}
	if len(manifest.GeneratedScope) != 1 || manifest.GeneratedScope[0] != path {
		addFinding(findings, path, "doctrine.provenance_scope", "refresh may write only %s", path)
	}
	if len(manifest.ProjectOwnedPaths) == 0 {
		addFinding(findings, path, "doctrine.project_owned_paths", "projectOwnedPaths must declare protected local artifacts")
	}
	if len(manifest.SourceOnlyExclusions) == 0 {
		addFinding(findings, path, "doctrine.source_exclusions", "source-only runtime exclusions must be explicit")
	}
}

type roleDeclaration struct {
	id             string
	principalID    string
	mode           string
	maxTrust       string
	effectiveTrust string
	autonomy       string
	prompt         string
	scope          string
}

func checkRoleManifest(root string, findings *[]Finding) {
	const path = ".harness/manifest.yaml"
	data, err := readRepoFile(root, path)
	if err != nil {
		addFinding(findings, path, "doctrine.manifest_missing", "harness manifest is required")
		return
	}
	values := yamlScalars(data)
	if scalar(values, "provenance.commit", "provenance.mars_commit", "provenance.marsCommit", "doctrine.mars_commit", "doctrine.marsCommit") != marsCommit {
		addFinding(findings, path, "doctrine.manifest_mars_pin", "manifest must pin the approved MARS commit")
	}
	defaultTrust := scalar(values, "default_effective_trust", "defaultEffectiveTrust", "trust.default_effective_trust", "trust.defaultEffectiveTrust", "trust.default", "security.effectiveTrust")
	if strings.ToLower(defaultTrust) != "observer" {
		addFinding(findings, path, "doctrine.default_trust", "default effective trust must be observer")
	}
	autonomy := strings.ToLower(scalar(values, "autonomous_mutation", "autonomousMutation", "trust.autonomous_mutation", "trust.autonomousMutation", "security.autonomousMutation"))
	if autonomy != "disabled" && autonomy != "false" {
		addFinding(findings, path, "doctrine.autonomous_mutation", "autonomous mutation must be disabled")
	}
	if scalar(values, "security.accepted_base_label") != "public+project-accepted" || scalar(values, "security.proposed_worktree_label") != "public+project-proposed" {
		addFinding(findings, path, "doctrine.repository_labels", "accepted base and proposed worktree must have distinct public labels")
	}
	if strings.ToLower(scalar(values, "security.rule_of_two")) != "declared" || strings.ToLower(scalar(values, "security.trace_spine")) != "declared" {
		addFinding(findings, path, "doctrine.future_security_claim", "H-001 may declare, but must not claim enforcement of, trace and Rule-of-Two controls")
	}
	roles := parseRoleDeclarations(data, "roles")
	if len(roles) == 0 {
		addFinding(findings, path, "doctrine.roles_missing", "at least one governed role is required")
		return
	}
	seen := make(map[string]bool)
	for _, role := range roles {
		if role.id == "" {
			addFinding(findings, path, "doctrine.role_id", "every role must have an id")
			continue
		}
		if seen[role.id] {
			addFinding(findings, path, "doctrine.role_duplicate", "role %q is declared more than once", role.id)
		}
		seen[role.id] = true
		if role.maxTrust == "" {
			addFinding(findings, path, "doctrine.role_max_trust", "role %q must declare max_trust separately", role.id)
		} else if role.maxTrust != "observer" && role.maxTrust != "contributor" {
			addFinding(findings, path, "doctrine.role_max_trust", "role %q has unsupported max_trust %q", role.id, role.maxTrust)
		}
		if strings.ToLower(role.effectiveTrust) != "observer" {
			addFinding(findings, path, "doctrine.role_effective_trust", "role %q must start with effective_trust observer", role.id)
		}
		checkRolePrompt(root, path, role, findings)
	}
	for _, required := range []string{"ceo", "coo", "cto", "engineer", "pipeline-fixer", "qa", "dogfood", "security-reviewer", "dependency-manager", "release-manager", "janitor", "delivery-orchestrator"} {
		if !seen[required] {
			addFinding(findings, path, "doctrine.required_role", "required role %q is missing", required)
		}
	}
	foundationRoles := parseRoleDeclarations(data, "foundation_roles")
	if len(foundationRoles) != 1 || foundationRoles[0].id != "foundation-maintainer" {
		addFinding(findings, path, "doctrine.foundation_role", "foundation_roles must contain only foundation-maintainer")
	} else {
		foundation := foundationRoles[0]
		if strings.ToLower(foundation.scope) != "source-only" {
			addFinding(findings, path, "doctrine.foundation_scope", "foundation-maintainer must be source-only")
		}
		if foundation.maxTrust == "" || strings.ToLower(foundation.effectiveTrust) != "observer" {
			addFinding(findings, path, "doctrine.foundation_trust", "foundation-maintainer must declare max_trust and start as observer")
		}
		checkRolePrompt(root, path, foundation, findings)
	}
	if strings.ToLower(scalar(values, "profiles_grant_authority", "profilesGrantAuthority")) != "false" {
		addFinding(findings, path, "doctrine.profile_authority", "profiles_grant_authority must be false")
	}
	profiles := parseRoleDeclarations(data, "profiles")
	if len(profiles) == 0 {
		addFinding(findings, path, "doctrine.profiles_missing", "at least one governed execution profile is required")
	}
	seenProfiles := make(map[string]bool)
	knownPrincipals := make(map[string]bool)
	for principal := range seen {
		knownPrincipals[principal] = true
	}
	knownPrincipals["foundation-maintainer"] = true
	for _, profile := range profiles {
		if profile.id == "" || profile.mode == "" {
			addFinding(findings, path, "doctrine.profile_identity", "every profile must declare id and mode")
			continue
		}
		identity := profile.id + ":" + profile.mode
		if seenProfiles[identity] {
			addFinding(findings, path, "doctrine.profile_duplicate", "profile %q is declared more than once", identity)
		}
		seenProfiles[identity] = true
		if profile.principalID == "" || !knownPrincipals[profile.principalID] {
			addFinding(findings, path, "doctrine.profile_principal", "profile %q must map to one declared executable principal", identity)
		}
		if profile.maxTrust != "observer" && profile.maxTrust != "contributor" {
			addFinding(findings, path, "doctrine.profile_max_trust", "profile %q has unsupported max_trust %q", identity, profile.maxTrust)
		}
		if strings.ToLower(profile.effectiveTrust) != "observer" {
			addFinding(findings, path, "doctrine.profile_effective_trust", "profile %q must start with effective_trust observer", identity)
		}
		if strings.ToLower(profile.autonomy) != "disabled" && strings.ToLower(profile.autonomy) != "false" {
			addFinding(findings, path, "doctrine.profile_autonomy", "profile %q must disable autonomous mutation", identity)
		}
		checkRolePrompt(root, path, profile, findings)
	}
}

func checkRolePrompt(root, manifestPath string, role roleDeclaration, findings *[]Finding) {
	if !safeRelativePath(role.prompt) || !strings.HasPrefix(role.prompt, ".harness/roles/") {
		addFinding(findings, manifestPath, "doctrine.role_prompt", "role or profile %q must declare a safe .harness/roles prompt", role.id)
		return
	}
	if !repoFileExists(root, role.prompt) {
		addFinding(findings, manifestPath, "doctrine.role_prompt_missing", "role or profile %q references missing prompt %s", role.id, role.prompt)
	}
}

func parseRoleDeclarations(data []byte, sectionName string) []roleDeclaration {
	lines := strings.Split(string(data), "\n")
	inRoles := false
	rolesIndent := -1
	var roles []roleDeclaration
	var current *roleDeclaration
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if !inRoles {
			if strings.TrimSuffix(trimmed, ":") == sectionName && strings.HasSuffix(trimmed, ":") {
				inRoles = true
				rolesIndent = indent
			}
			continue
		}
		if indent <= rolesIndent && !strings.HasPrefix(trimmed, "-") {
			break
		}
		if strings.HasPrefix(trimmed, "-") {
			fields := strings.SplitN(strings.TrimSpace(strings.TrimPrefix(trimmed, "-")), ":", 2)
			if len(fields) == 2 && normalizeKey(fields[0]) == "id" {
				roles = append(roles, roleDeclaration{id: trimYAMLScalar(fields[1])})
				current = &roles[len(roles)-1]
			}
			continue
		}
		if current == nil {
			continue
		}
		fields := strings.SplitN(trimmed, ":", 2)
		if len(fields) != 2 {
			continue
		}
		value := trimYAMLScalar(fields[1])
		switch normalizeKey(fields[0]) {
		case "principalid":
			current.principalID = value
		case "mode":
			current.mode = value
		case "maxtrust":
			current.maxTrust = value
		case "effectivetrust":
			current.effectiveTrust = value
		case "autonomousmutation":
			current.autonomy = value
		case "prompt":
			current.prompt = value
		case "scope":
			current.scope = value
		}
	}
	return roles
}

func decodeStrictJSON(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values are not allowed")
		}
		return fmt.Errorf("invalid trailing JSON: %w", err)
	}
	return nil
}

func safeRelativePath(path string) bool {
	if path == "" || filepath.IsAbs(path) || strings.Contains(path, "\\") {
		return false
	}
	clean := filepath.ToSlash(filepath.Clean(path))
	return clean != "." && clean != ".." && !strings.HasPrefix(clean, "../") && clean == path
}

func openSSHPublicKeyFingerprint(data []byte) (string, error) {
	fields := strings.Fields(string(data))
	if len(fields) < 2 || fields[0] != "ssh-ed25519" {
		return "", errors.New("expected an ssh-ed25519 public key")
	}
	blob, err := base64.StdEncoding.DecodeString(fields[1])
	if err != nil {
		return "", errors.New("public key is not valid base64")
	}
	digest := sha256.Sum256(blob)
	return "SHA256:" + base64.RawStdEncoding.EncodeToString(digest[:]), nil
}

type sshReader struct {
	data []byte
	off  int
}

func (reader *sshReader) raw(length int) ([]byte, error) {
	if length < 0 || reader.off+length > len(reader.data) {
		return nil, errors.New("truncated SSH data")
	}
	result := reader.data[reader.off : reader.off+length]
	reader.off += length
	return result, nil
}

func (reader *sshReader) string() ([]byte, error) {
	prefix, err := reader.raw(4)
	if err != nil {
		return nil, err
	}
	length := int(binary.BigEndian.Uint32(prefix))
	return reader.raw(length)
}

func sshString(value []byte) []byte {
	result := make([]byte, 4+len(value))
	binary.BigEndian.PutUint32(result[:4], uint32(len(value)))
	copy(result[4:], value)
	return result
}

func verifySSHSig(message, armored, publicKey []byte, expectedNamespace string) error {
	text := strings.TrimSpace(string(armored))
	const begin = "-----BEGIN SSH SIGNATURE-----"
	const end = "-----END SSH SIGNATURE-----"
	if !strings.HasPrefix(text, begin) || !strings.HasSuffix(text, end) {
		return errors.New("invalid SSH signature armor")
	}
	body := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(text, begin), end))
	body = strings.Join(strings.Fields(body), "")
	blob, err := base64.StdEncoding.DecodeString(body)
	if err != nil {
		return errors.New("invalid SSH signature base64")
	}
	reader := &sshReader{data: blob}
	magic, err := reader.raw(6)
	if err != nil || string(magic) != "SSHSIG" {
		return errors.New("invalid SSH signature magic")
	}
	versionBytes, err := reader.raw(4)
	if err != nil || binary.BigEndian.Uint32(versionBytes) != 1 {
		return errors.New("unsupported SSH signature version")
	}
	keyBlob, err := reader.string()
	if err != nil {
		return err
	}
	namespace, err := reader.string()
	if err != nil {
		return err
	}
	reserved, err := reader.string()
	if err != nil {
		return err
	}
	hashAlgorithm, err := reader.string()
	if err != nil {
		return err
	}
	signatureBlob, err := reader.string()
	if err != nil || reader.off != len(reader.data) {
		return errors.New("invalid SSH signature envelope")
	}
	if string(namespace) != expectedNamespace {
		return errors.New("signature namespace does not match genesis metadata")
	}

	keyFields := strings.Fields(string(publicKey))
	declaredKeyBlob, err := base64.StdEncoding.DecodeString(keyFields[1])
	if err != nil || !bytes.Equal(keyBlob, declaredKeyBlob) {
		return errors.New("signature key does not match the declared public key")
	}
	keyReader := &sshReader{data: keyBlob}
	keyAlgorithm, err := keyReader.string()
	if err != nil || string(keyAlgorithm) != "ssh-ed25519" {
		return errors.New("signature does not use ssh-ed25519")
	}
	key, err := keyReader.string()
	if err != nil || len(key) != ed25519.PublicKeySize || keyReader.off != len(keyReader.data) {
		return errors.New("invalid ssh-ed25519 public key")
	}

	var messageDigest []byte
	switch string(hashAlgorithm) {
	case "sha256":
		digest := sha256.Sum256(message)
		messageDigest = digest[:]
	case "sha512":
		digest := sha512.Sum512(message)
		messageDigest = digest[:]
	default:
		return fmt.Errorf("unsupported signature hash %q", hashAlgorithm)
	}
	signed := make([]byte, 0, 6+len(namespace)+len(reserved)+len(hashAlgorithm)+len(messageDigest)+16)
	signed = append(signed, []byte("SSHSIG")...)
	signed = append(signed, sshString(namespace)...)
	signed = append(signed, sshString(reserved)...)
	signed = append(signed, sshString(hashAlgorithm)...)
	signed = append(signed, sshString(messageDigest)...)

	sigReader := &sshReader{data: signatureBlob}
	sigAlgorithm, err := sigReader.string()
	if err != nil || string(sigAlgorithm) != "ssh-ed25519" {
		return errors.New("invalid ssh-ed25519 signature algorithm")
	}
	signature, err := sigReader.string()
	if err != nil || len(signature) != ed25519.SignatureSize || sigReader.off != len(sigReader.data) {
		return errors.New("invalid ssh-ed25519 signature")
	}
	if !ed25519.Verify(ed25519.PublicKey(key), signed, signature) {
		return errors.New("signature verification failed")
	}
	return nil
}

func parsePositiveInt(value string) (int, bool) {
	number, err := strconv.Atoi(strings.TrimSpace(value))
	return number, err == nil && number > 0
}
