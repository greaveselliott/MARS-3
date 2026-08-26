# H-001 validation evidence

**Classification:** PUBLIC
**Evidence ID:** H-001-E4
**Status:** Candidate; independent QA and Security review pending
**Opaque trace reference:** `bootstrap-h001-workflow-contract-v5`
**Recorded:** 2026-08-26
**Verification owners:** QA Reviewer, then Security Reviewer

This is a redacted evidence manifest, not a raw command log. The immutable
review target is the Git commit containing this file. The implementation
checkpoint it verifies is
`1214ce5f148c6c3253a877e3f8a108c0253c6448`, based on signed genesis
`8c108460d7c0bb59b80a0b3942dc872a2e05785a`.

## Bound artifacts

| Artifact | Immutable identifier or SHA-256 |
| --- | --- |
| implementation tree | `5a387bba8c8affe1a65beb7e2744f81f641f2f50` |
| validation binary | `766662b93986a3611cd0709e6accf0ae59c09237703b2460bcda0edf84b50861` |
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
| history secrets | `gitleaks detect --source . --redact --no-banner` | pass; nine commits scanned at checkpoint |
| provenance refresh | `go run ./cmd/mars3 doctrine refresh --repo . --source ../MARS --ref f55d129bfc794510ca485bb54fc0a35c7b04a700` | pass; dry run; 20 files |
| claim signature | `ssh-keygen -Y verify` with the committed public key and namespace `mars3-claim-attestation` | pass |
| remote branch | `git ls-remote origin refs/heads/codex/h-001-doctrine-foundation` | implementation checkpoint published; evidence-containing review target is verified after publication and recorded in Beads |

Local source tests used Go 1.26.2. Public CI pins Go 1.24.11 and Gitleaks
v8.18.4 at OCI manifest
`sha256:75bdb2b2f4db213cde0b8295f13a88d6b333091bbfbf3012a4e083d00d31caba`.
The Gitleaks detection canary failed with exit 1 as required before either
repository scan was trusted. A newer source-built candidate that returned zero
for the same canary was rejected and is not part of the gate.

## Reproducible validation binary

The bound binary is a static `linux/amd64` artifact built with the local Go
toolchain fixed at `go1.26.2`. From a clean checkout of the implementation
checkpoint, run this repository-relative command:

```text
GOTOOLCHAIN=local CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -buildvcs=false -ldflags=-buildid= -o ../mars3-h001-linux-amd64 ./cmd/mars3
```

Two rebuilds with distinct empty Go build and temporary caches produced the
same SHA-256 shown above. `go version -m` reported `go1.26.2`, `CGO_ENABLED=0`,
`GOOS=linux`, `GOARCH=amd64`, `GOAMD64=v1`, `-trimpath=true`, and no VCS stamp.

## Public CI and repository controls

- GitHub Actions run
  `https://github.com/greaveselliott/MARS-3/actions/runs/32927392259`
  completed successfully for the implementation checkpoint.
- The repository visibility API returned `PUBLIC` with default branch `main`.
- Active ruleset `21510926` prohibits deletion and non-fast-forward updates,
  requires linear PR delivery and resolved conversations, and strictly
  requires `Public commit gate` from GitHub Actions.
- Workflow permissions are `contents: read`; checkout credentials are not
  persisted; fork workflows receive no configured repository secret or write
  permission; every action and container is immutably pinned.
- GitHub reported that the pinned checkout and Go setup actions still target
  Node.js 20 and were forced onto Node.js 24. The job passed, no authority was
  added, and the annotation remains a bounded action-pin maintenance risk; an
  update requires a new reviewed workflow digest rather than silent drift.

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
  Structural regressions reject privileged or indirect events, permissions,
  secret expressions, action references and inputs, and container changes.
  The independent full-workflow digest additionally rejects disabled or masked
  gates, runner or shell changes, arbitrary steps, dynamic Docker execution,
  comments, and any second workflow. An independent preflight replay accepted
  the bounded contract after all nine prior job/effect mutations failed.

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
- Checkpoint `e928a243d3c43e167d673f3c1fd12ec693635620` rejects structural YAML
  anchors and aliases before permission evaluation and includes the exact
  alias-key regression. H-001-E3 also replaces the underspecified binary hash
  with the reproducible recipe and matching rebuilds above. No prior review
  verdict carries forward to the immutable commit containing H-001-E3.
- QA accepted the immutable H-001-E3 review target
  `54a1a1b062238d58f3bb3f9de61ab019ad3170c2`. Security then reproduced a
  quoted `pull_request_target` key that GitHub accepts but the E3 spelling-based
  event check did not detect. The Foundation Maintainer separately reproduced
  a bracket-form `secrets` expression omitted by the E3 dot-form check.
  Security returned `changes-requested`; M3-H001 reopened on the same Bead and
  branch. E3 is superseded and neither verdict carries to E4.
- Signed remediation checkpoint
  `74005d6a8f1e93aeb9f7f1faeff296c1f233fe48` and GitHub Actions run
  `32926762404` passed the then-current gates. An independent preflight review
  correctly withheld E4: selected-field validation still admitted disabled or
  masked gates, arbitrary steps, runner and shell changes, and a dynamically
  constructed Docker invocation. Its raw container scans also produced
  misleading findings on inert text, and the positive fixture read the live
  workflow. These are foundation-owned findings on the same Bead, not a new
  ticket. The subsequent remediation binds the entire CRLF-normalized workflow
  to an independent hard-coded digest while retaining structural diagnostics.
- Checkpoint `1214ce5f148c6c3253a877e3f8a108c0253c6448` adds that immutable contract,
  closes case-variant workflow-extension routing, and passed independent
  preflight plus exact-SHA CI run `32927392259`. Two clean Linux/amd64 builds
  produced the same validation-binary hash above. No previous QA or Security
  verdict carries to the immutable commit containing H-001-E4.

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
- The two pinned GitHub actions have a Node.js 20 deprecation annotation while
  GitHub currently forces them to Node.js 24. This is not an H-001 authority or
  correctness failure, but future pin maintenance must update the workflow
  contract through a new decision and review.

Next executable owner: QA Reviewer. On `accepted`, the exact same immutable E4
commit routes to Security Reviewer. The Delivery Orchestrator may reconcile and
close M3-H001 only after both canonical Beads verdicts and a completed run
disposition.
