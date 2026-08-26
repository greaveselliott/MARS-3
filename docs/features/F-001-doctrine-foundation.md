# F-001 — Public doctrine foundation

**Status:** Active
**Goal:** G-001
**Product decisions:** PD-001, PD-002, PD-003
**Product specification:** `docs/product-specs/foundation.md`
**Active Bead:** M3-H001 (display ID H-001)

## Business logic

1. Git, Beads/Dolt, and the future live-lease store have distinct authority and
   meet only through stable identifiers, immutable commits, trace references,
   review verdicts, and run dispositions.
2. Exactly one active plan selects the current failing scenario and claimed
   Bead.
3. MARS-derived doctrine is accepted only from one pinned commit with path and
   Git-blob provenance; refresh writes generated provenance only.
4. Every principal begins as an observer. A maximum trust declaration is not
   an active capability grant.
5. Every repository artifact must be safe for immediate public disclosure.
6. Completion requires deterministic checks, independent QA, independent
   Security review, and Orchestrator reconciliation.
7. Trace-spine and Rule-of-Two artifacts are declared contracts in H-001;
   runtime enforcement remains explicitly scheduled for T-001 and S-001.
8. Verification-order entries are executable identity IDs. Each must resolve
   to a declared role or to a profile backed by a declared principal; display
   labels and undeclared aliases never grant routing authority.

## Step-by-step behavior

1. The operator reads G-001, its decisions, this contract, and the one active
   plan from Git.
2. The authorized bootstrap operator verifies the signed external M3-H001
   claim with the pinned Beads client and supplies its public-safe identity,
   state, owner, goal, feature, applicable product decisions, scenarios, and
   exclusive paths. Its ordered reviewer IDs must equal the canonical Bead and
   resolve through the executable registry. Those links must equal Git's active
   lineage. No W-001 gateway or lease enforcement is claimed by H-001.
3. The Foundation Maintainer changes only authorized doctrine and validation
   paths and emits public-safe evidence.
4. Validation checks authority uniqueness, provenance, trust defaults,
   documentation links, public disclosure constraints, source behavior, and
   repository history.
5. QA validates behavior against the same immutable commit. Security then
   validates disclosure, provenance, trace, and Rule-of-Two invariants.
6. The Orchestrator records accepted QA and Security review verdicts plus a
   `completed` run disposition, reconciles the Git commit to M3-H001, and only
   then moves the Bead to `done`.

## Scenario schedule

| Scenario | State | Verification owner | Required evidence |
| --- | --- | --- | --- |
| F-001-S1 | passing | QA | corrected typed Beads lineage, signed claim, and executable-reviewer registry tests |
| F-001-S2 | passing | QA | offline provenance and refresh-scope tests |
| F-001-S3 | passing | Security | manifest/trust and mutation-denial tests |
| F-001-S4 | passing | Security | H-001-E4 immutable workflow and public-gate evidence |

All four scenarios pass deterministically at the current implementation
checkpoint. The same Bead carries typed `featureId: F-001`, all three product
decisions, and the exact routable order `qa → security-reviewer →
delivery-orchestrator`. The fresh signed attestation plus negative omission,
stale-binding, duplicate, malformed, and undeclared-reviewer tests bind that
contract offline. H-001-E5 received `changes-requested`; no earlier verdict
carries to a changed commit. Passing never means accepted or done: release
readiness remains blocked until both reviewers accept the same immutable
containing commit and Beads records reconciliation.

### F-001-S1 — One durable delivery route

**Given** a fresh public clone with no operational database
**And** the immutable genesis charter links to M3-H001
**And** the authorized bootstrap verifier confirms the external M3-H001 claim
**And** the Bead and signed claim explicitly match Git's goal, product
decisions, feature, and scenario identifiers
**And** every ordered verification identity is an executable registry ID
**When** the operator runs doctrine and plan checks
**Then** Git exposes G-001 → decisions/spec → F-001 → one active plan
**And** the plan points to M3-H001 without becoming ticket authority
**And** an undeclared reviewer alias is rejected as unroutable
**And** no Markdown ticket lifecycle shadow system exists.

### F-001-S2 — Offline doctrine provenance

**Given** the generated manifest pins MARS revision
`f55d129bfc794510ca485bb54fc0a35c7b04a700`
**When** doctrine validation runs without network access
**Then** every required source path, Git blob, license, adaptation, exclusion,
and generated scope validates
**And** a refresh rejects every other revision
**And** an accepted refresh can modify only the generated source manifest.

### F-001-S3 — Observer-first trust

**Given** the executable role manifest declares both maximum and effective
trust
**When** the static executable manifest and every role/profile declaration are
validated before a capability lease exists
**Then** each effective trust value is `observer`
**And** each profile maps to a declared principal and disables autonomous
mutation
**And** profile configuration is explicitly incapable of granting authority.

### F-001-S4 — Immediate public disclosure safety

**Given** every tracked byte is treated as immediately public
**When** the complete public commit gate scans the worktree and Git history
**Then** deterministic gates reject recognized secrets, non-synthetic email
identity, local metadata, raw-payload fields, unsafe fixtures, binaries during
H-001, missing license or provenance, undeclared generated files, and stale
documentation links
**And** workflow admission accepts only the literal public triggers, read-only
permission block, pinned actions with exact inputs, public concurrency
expressions, and three exact no-network/read-only scanner containers
**And** it rejects quoted or indirect authority keys, privileged triggers,
secret or token contexts, extra action inputs, reusable or local actions, job
containers or services, and modified Docker flags, mounts, images, or commands
instead of inferring safety from one recognized spelling
**And** a CRLF-normalized full-workflow digest rejects disabled or masked gates,
extra steps, runner or shell changes, dynamically constructed commands, and all
other execution mutations outside the structural allowlists
**And** deterministic tests, vet, diff checks, and secret history scan pass
**And** independent human QA and Security review the same immutable commit for
semantic or unrecognized disclosure risks that pattern checks cannot prove.

## Permissions, validation, transitions, and failures

- Only the Foundation Maintainer may implement M3-H001's exclusive paths under
  the human-authorized bootstrap exception.
- QA and Security are independent observers and may not repair product code
  during their review attempts.
- H-001 does not claim gateway enforcement, compare-and-swap claims, or lease
  epochs; those are W-001 acceptance requirements.
- A `changes-requested` review verdict returns the same Bead to `in-progress`.
  It does not create duplicate work.
- One automatic retry is allowed for a normalized failure fingerprint. A
  repeated equivalent failure becomes a durable `blocked` run disposition and
  human escalation.
- Missing or mismatched evidence leaves the Bead `in-review` and the run
  disposition `in_review` or `blocked`; it never moves the Bead to `done`.

## Evidence manifest requirements

Evidence records the immutable commit, relative paths, exact commands,
versions, exit outcomes, content hashes where relevant, opaque trace
references, reviewer identity by role, residual risks, and next owner. It never
contains raw prompts, completions, tool payloads, secrets, private source, or
machine-specific paths.

## Out of scope

- Beads gateway, compare-and-swap claim API, and live fencing enforcement.
- PostgreSQL, Temporal, Kubernetes, object storage, OIDC, or hosted tenancy.
- Codex, Claude, Colibri, model artifacts, serving runtimes, or credential
  brokering.
- Operator web/Electron UI, browser, terminal, and production release paths.

## Descoped scenarios

None. Only the CEO strategy principal may descope an in-scope scenario, and it
must first record a superseding Product decision, affected-goal analysis, and
active-plan update. A ticket owner, reviewer, or Orchestrator cannot descope a
scenario through a run disposition.
