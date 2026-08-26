# H-001 validation evidence

**Classification:** PUBLIC
**Evidence ID:** H-001-E2
**Status:** Superseded after QA `changes-requested`; not acceptance evidence
**Opaque trace reference:** `bootstrap-h001-security-remediation-v2`
**Recorded:** 2026-08-26
**Verification owners:** QA Reviewer, then Security Reviewer

This is a redacted evidence manifest, not a raw command log. The immutable
review target is the Git commit containing this file. The implementation
checkpoint it verifies is
`880e24fa3fdc55d3eec5b9780ef153022c3aa3c4`, based on signed genesis
`8c108460d7c0bb59b80a0b3942dc872a2e05785a`.

## Bound artifacts

| Artifact | Immutable identifier or SHA-256 |
| --- | --- |
| implementation tree | `7f616dc63064eedbff6d6f846fdf0ac2df2ba05d` |
| validation binary | `31a7661149c5261d03893ad49b63586f31306af1727f20dd126d8491db70c9e5` |
| MARS source manifest | `1d9e01d5f90ea6335299284befe77798809e381b238472446ae61427070b90b2` |
| signed claim attestation | `41079a23f6bb4a2a68cec481d2bf474cc53343a084e24406fa63343084fdd8c4` |
| claim signature | `3727f5e409a70ff5cf901923ad50ec962643be825c7b4c79d06a793395a6e905` |
| harness manifest | `58ad0c48a2f3677c183d1f6689d6893e0b9bbe508417fee3bf60a9451286230a` |
| Beads claim checkpoint | `vvbleat3bc69avj90ciauegpbfr2o3g6` |
| signed attestation ledger head | `7l67arkkk8bcis9pp9ns0u8uep1jq4vi` |

The signed attestation records M3-H001 as `in-progress`, owned by
`foundation-maintainer`, with zero dependencies and the complete exclusive
path set. Later lifecycle and review mutations remain canonical only in Beads.

## Reproducible checks

All commands run from the repository root. Exit outcome was zero unless the
row explicitly describes the expected failing canary.

| Check | Command | Outcome |
| --- | --- | --- |
| doctrine | `go run ./cmd/mars3 doctrine check --repo .` | pass |
| active plan | `go run ./cmd/mars3 plan check --repo .` | pass |
| FactoryDocSync | `go run ./cmd/mars3 docsync audit --repo .` | pass |
| public disclosure | `go run ./cmd/mars3 public-check --repo .` | pass |
| source behavior | `go test ./...` | pass |
| static analysis | `go vet ./...` | pass |
| whitespace | `git diff --check` and `git show --check --oneline --no-renames HEAD` | pass |
| worktree secrets | `gitleaks detect --no-git --source . --redact --no-banner` | pass |
| history secrets | `gitleaks detect --source . --redact --no-banner` | pass; four commits scanned at checkpoint |
| provenance refresh | `go run ./cmd/mars3 doctrine refresh --repo . --source ../MARS --ref f55d129bfc794510ca485bb54fc0a35c7b04a700` | pass; dry run; 20 files |
| claim signature | `ssh-keygen -Y verify` with the committed public key and namespace `mars3-claim-attestation` | pass |
| remote branch | `git ls-remote origin refs/heads/codex/h-001-doctrine-foundation` | exact checkpoint SHA |

Local source tests used Go 1.26.2. Public CI pins Go 1.24.11 and Gitleaks
v8.18.4 at OCI manifest
`sha256:75bdb2b2f4db213cde0b8295f13a88d6b333091bbfbf3012a4e083d00d31caba`.
The Gitleaks detection canary failed with exit 1 as required before either
repository scan was trusted. A newer source-built candidate that returned zero
for the same canary was rejected and is not part of the gate.

## Public CI and repository controls

- GitHub Actions run
  `https://github.com/greaveselliott/MARS-3/actions/runs/32923192961`
  completed successfully for the implementation checkpoint.
- The repository visibility API returned `PUBLIC` with default branch `main`.
- Active ruleset `21510926` prohibits deletion and non-fast-forward updates,
  requires linear PR delivery and resolved conversations, and strictly
  requires `Public commit gate` from GitHub Actions.
- Workflow permissions are `contents: read`; checkout credentials are not
  persisted; fork workflows receive no configured repository secret or write
  permission; every action and container is immutably pinned.

## Scenario evidence

- **F-001-S1:** the plan checker proved one canonical active plan, explicit
  goal/decision/spec/BDD links, one current Bead, dependency order, and no Git
  ticket-lifecycle shadow tree.
- **F-001-S2:** doctrine check and the exact-checkout dry run proved the MARS
  commit, all 20 required source blobs, attribution, exclusions, signed
  genesis, signed claim, and generated-manifest-only refresh scope.
- **F-001-S3:** doctrine tests proved every role and profile remains observer,
  profiles map to declared principals, autonomous mutation is disabled, and a
  mutation fixture that enables profile authority is rejected.
- **F-001-S4:** public, source, vet, whitespace, worktree/history secret, binary
  denial, mutable-container, malformed-DocSync, and workflow checks passed.
  Regression cases reject inline, indented job-level, duplicate, aliased,
  flow-nested, escaped, and explicit-key permission declarations while
  accepting only the canonical top-level `contents: read` block.

## Review and remediation history

- QA accepted prior target
  `54d778c2961cb69ed147deccc6fe8a32b6af2d73` after its deterministic and
  negative checks passed.
- Security returned `changes-requested` for that target after reproducing a
  job-level inline `permissions: { contents: write }` declaration that the
  earlier regex gate missed. M3-H001 reopened on the same branch and Bead.
- Checkpoint `880e24fa3fdc55d3eec5b9780ef153022c3aa3c4` replaces regex inference with
  the fail-closed permission structure described above. Prior review verdicts
  do not carry forward; the immutable commit containing H-001-E2 must complete
  a fresh QA → Security review sequence.
- QA returned `changes-requested` for H-001-E2 after an anchored scalar used as
  an aliased job key resolved to `permissions` without a finding. QA also found
  that the recorded validation-binary hash lacked a reproducible build tuple
  and did not match two clean checkpoint rebuilds. M3-H001 reopened on the same
  branch for a third candidate; H-001-E2 must not be used as acceptance proof.

FactoryDocSync was checked and remained current. This ticket changed its BDD,
authority, provenance, trace, security, publication, runtime, and operator
documentation together with the validation behavior.

## Remaining risk and handoff

- Pattern checks cannot prove the absence of every semantic disclosure;
  independent human Security review remains mandatory.
- H-001 declares but does not implement the Beads gateway, live lease epoch,
  trace service, Rule-of-Two policy engine, tool broker, sandbox, or credential
  proxy. No production or autonomous authority is claimed.
- The bootstrap exception remains direct human procedure until W-001 replaces
  it; it is not reusable by a tenant agent.

Next executable owner: Foundation Maintainer, limited to the two QA-requested
corrections. A replacement evidence manifest must bind an explicitly versioned,
cross-compiled, path-independent build recipe and route a new immutable commit
through fresh QA → Security review.
