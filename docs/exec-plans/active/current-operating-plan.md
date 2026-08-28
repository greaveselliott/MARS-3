# Active operating plan — governed work-authority walking skeleton

**Status:** Active
**Owner:** Delivery Orchestrator
**Updated:** 2026-08-26
**Phase:** delivery
**Goal:** G-001
**Current feature:** F-002
**Current Bead:** M3-W001 (display ID W-001)
**Authority:** Beads/Dolt for work state; Git for this durable plan

This is the only active execution plan. It is a Git-owned ordering and evidence
contract, not a second ticket database. The reviewed helper is accepted on
`main`, and the signed W-001 bootstrap flow has compare-and-swap claimed the
canonical Bead as `in-progress`. This `delivery` projection records that
durable fact. The accepted postclaim tree has been squash-merged with exact
tree equality, protected-main CI passed, and the Orchestrator recorded the
bounded reconciliation receipt in M3-W001. The separately signed
`W-001-delivery-v2` grant authorized the merged core implementation. Exact-tree
QA and Security review and protected-main run `33069887434` accepted that
checkpoint. The completion audit then found that handoff, ordered review
verdict, run disposition, reconciliation, and terminal lifecycle routes remain
absent. W-001 therefore remains `in-progress`. Independent review of the
immutable v5 candidate requested changes for terminal claim lineage,
full-fence replay, missing-receipt recovery, truthful nonterminal convergence,
and reproducible qualification. The separately signed
`W-001-lifecycle-correction-v6` authorizes only those bounded corrections; no
live lease exists.

## Durable lineage

- Goal: [G-001](../../goals/active.md)
- Decision: [PD-002](../../product-decisions/PD-002-git-beads-authority.md)
- Product promise: [work-authority specification](../../product-specs/work-authority.md)
- Behavior contract: [F-002](../../features/F-002-work-authority.md)
- Work authority: external Bead `M3-W001`; this Git plan selects it and mirrors
  bounded state but never creates a claim, lease, transition, or disposition.

The local-substrate contract and P-001 are prepared in the same planning wave,
but F-002 is the sole current feature and W-001 is the sole selected Bead.
The Orchestrator reconciled P-001's canonical dependency set to closed M3-H001
plus backlog M3-W001 under opaque replan correlation
`wave1-p001-w001-dependency-replan-v1`; P-001 remains backlog and unclaimed.

That restrictive dependency mutation followed a durable `REPLAN` intent and
receipt, but `WAVE-1-contract-publication` did not explicitly enumerate a
Beads dependency effect in its allowed-effect list. This is recorded as a
foundation-owned authority intervention, not retroactively described as
preauthorized. The safer blocking edge remains frozen, no further Beads
mutation is permitted under the planning grant, and contract acceptance
requires the signed Git evidence plus explicit independent QA and Security
disposition on this correction.

The first recovery attempt then under-specified its own state encoding and
intermediate metadata/description effects. Those records remain durable and
are not retroactively accepted. The prospective, signed
`WAVE-1-recovery-disposition` binds a reconstructible public snapshot and
authorized exactly one P-001 description postimage, which read-back verified
without lifecycle, dependency, claim, lease, or W-001 changes. Both Beads
remain backlog and unclaimed. The original scanner-triggering recovery file is
represented by signed public checksums rather than admitted through a secret
scanner exception.

GitHub's existing linear, squash-only protection remains unchanged. The final
reviewed branch tree must be retained by the signed annotated tag
`mars3/wave1-contract-publication-v1`; PR and protected-main CI require the tag
target tree, reviewed PR tree, and squash-main tree to be identical. This
preserves the signed feature history without weakening branch or scanner
policy.

## Current hypothesis and walking skeleton

If every work mutation passes through one typed gateway that joins canonical
Beads state to a factory-issued, monotonically fenced live lease, stale workers
and provider runtimes can be prevented from claiming authority through local
state, retries, or direct database access. W-001 proves this at its gateway and
synthetic pre-effect boundary; S-002 later qualifies real external brokers.

The walking skeleton is one synthetic public project and one W-001 attempt:
read a ready Bead, compare-and-swap claim it, issue a scoped lease epoch,
heartbeat it, reject stale or mismatched writes, append a bounded event, and
rebuild the read projection without giving Temporal or PostgreSQL ownership of
the work graph. The current phase schedules delivery against the verified
claim. The signed delivery grant is active for its exact attempt, base,
principal, and paths; no lease exists yet. Its v2 publication route preserved
the original delivery branch as public foundation-failure evidence and the v4
correction admitted exactly ten immutable synthetic scanner fingerprints. The
current correction attempt closes the exact v5 findings without relabeling the
accepted core checkpoint or the unaccepted v5 candidate as terminally complete.

## Scenario priority

1. F-002-S1 — gateway-only canonical mutation and role separation.
2. F-002-S2 — atomic compare-and-swap claim.
3. F-002-S3 — monotonic live lease epoch and heartbeat.
4. F-002-S4 — effect fencing and immediate lease-loss denial.
5. F-002-S5 — direct authority access and local label admission fail closed.
6. F-002-S6 — ordered journal recovery and full rebaseline.

The scenarios are ordered to establish read truth before mutation, then prove
that losing authority blocks the W-001 synthetic effect boundary and defines
the contract later real brokers must enforce. M3-W001 already
declares this exact group and required evidence. The canonical claim itself
grants neither a lease nor source-code authority; the separately signed v6
lifecycle-correction grant supplies the current bounded source authority.

## Delivery waves

| Wave | Bead | Owner | Depends on | State | Exit evidence |
| --- | --- | --- | --- | --- | --- |
| 0 | H-001 Doctrine foundation | Foundation Maintainer | signed genesis | done | QA/Security accepted E7, verified public merge, completed run, and reconciliation receipt |
| 1 | W-001 Work Authority | Work Authority Engineer | H-001 | in-progress | Beads gateway, CAS claims, PostgreSQL lease epochs, stale-effect denial, and projection recovery |
| 1 | P-001 Local substrate | Platform Engineer | H-001, W-001 | backlog | Lima/k3s, OIDC, RLS, Temporal, storage, and isolation evidence |
| 2 | T-001 Trace spine | Trace Engineer | W-001, P-001 | backlog | audit ledger, OTel/Tempo, effect intents, receipts, and replay |
| 3 | S-001 Rule-of-Two policy | Security Engineer | T-001 | backlog | labels, taint, tool contracts, and hard admission policy |
| 4 | I-001 Git/evidence reconciliation | Integration Engineer | W-001, T-001, S-001 | backlog | PR publication saga and merged-evidence closure |
| 4 | S-002 Secure effects | Security/Platform Engineer | P-001, T-001, S-001 | backlog | gVisor, brokers, credential proxy, and deterministic publication |
| 5 | A-001 Runtime contracts | Runtime Architect | T-001, S-001, S-002 | backlog | adapter conformance, qualification, and routing |
| 5 | UI-001 Operator workspace | Frontend Engineer | P-001, T-001, S-001 | backlog | shared React/Electron workspace and trace/security views |
| 6 | C-001 Codex adapter | Runtime Engineer | A-001, S-002 | backlog | contained Codex ticket execution |
| 6 | L-001 Colibri advisory | Model Runtime Engineer | A-001, P-001 | backlog | local advisory generation with mutation disabled |
| 7 | E-001 Public first-slice fixture | Engineer → QA → Security | I-001, UI-001, C-001, L-001 | backlog | public fixture delivery and merged PR |
| 8 | C-002 Claude parity | Runtime Engineer | E-001 | backlog | Claude and mixed-routing conformance |
| 9 | D-001 Dogfood/release | Dogfood/Release | C-002 | backlog | compartments, trust ledger, re-verification, and approval |
| 10 | K-001 Skills/code graph | Foundation Maintainer | D-001 | backlog | signed, licensed, quarantined capability registry |
| 11 | O-001 Hosted hardening | Platform/Security | K-001 | backlog | multi-tenant isolation and capacity suite |

No backlog Bead is claimable before its dependencies and Git-owned feature
contract are accepted. P-001 is deliberately sequenced after W-001 so it can
use accepted gateway fencing instead of receiving another bootstrap exception.
The Orchestrator may schedule it only through a later truthful plan transition.

## Current delivery transition

- Canonical work: M3-W001, native `in_progress`, typed lifecycle `in-progress`.
- Work type: enabler.
- Intended owner/profile: Work Authority Engineer (`work-authority-engineer`).
- Coordinator: Delivery Orchestrator.
- Risk: critical.
- Failure ownership: foundation.
- Scenarios: F-002-S1 through F-002-S6.
- Verification order: `qa` → `security-reviewer` → `delivery-orchestrator`.
- Accepted helper commit: `663d19bf190f9e3bd27edc96ee08acaa6778c853`;
  squash-merged as `adfd64feb565fb703a3568122cc032d4d1a450f5` with
  reviewed tree equality.
- Claim state: verified by Dolt commit
  `67hmen0cmq0he08n7ujlqpcsmmi94fhb`, WorkVersion mutation sequence `1`,
  dependency-graph revision `1`, and exact signed postimage digests.
- Live lease: absent by design; the bootstrap claim grants none.
- Postclaim reconciliation: QA and Security accepted v6 tree
  `7febda7ec2fec47b7d6bf11fdd5b24e605b9e2b2`; PR #8 squash-merged as
  `59f1fe24952b68bd3bbb6994bfee46c350b7c9cd`; protected-main run
  `33025602656` passed; canonical comment
  `01a0408e-ca08-71f0-b1ac-0dec0039706a` records the consistent Git/Beads
  read-back.
- Delivery authority: signed grant `W-001-delivery-v2`, attempt
  `w001-delivery-87d9680d-ca5a-4f3d-9afc-741884232e73`, exact base
  `59f1fe24952b68bd3bbb6994bfee46c350b7c9cd`.
- Required next transition: bind the V17 shared one-shot, descriptor-stream
  fetch, and closed production-process inventory correction to a signed
  immutable checkpoint; verify that exact tree from a normal clone through the
  public and leak gates; then publish its signed tag, run exact-head CI, and
  route it through independent QA and Security. The independent cold builds,
  non-skipped native Beads suite, exact-artifact integration, and PostgreSQL
  lifecycle suite have passed. No
  canonical handoff or later lifecycle mutation may execute until the reviewed
  tree is merged and a separate reconciliation authority binds the
  protected-main result.

The W-001 bootstrap grant is deliberately not a live lease: it is
human-directed, binds one base commit and attempt, permits only canonical W-001
paths plus required evidence/publication, and expires when the gateway passes
self-host conformance. The first PostgreSQL epoch is W-001 acceptance evidence,
not a prerequisite for building the epoch service. The authoritative Bead
holds exclusive paths and mutable lifecycle; this plan does not copy lease
values or represent a proposed owner as a current grant.

## Lifecycle-correction and CI-stabilization candidate

The v9 bounded candidate retains the typed handoff, ordered review, run,
reconciliation, and terminal routes while closing the independent v5 findings.
Terminal versioned work must retain exactly one complete WorkClaim or
BootstrapClaim plus detailed lifecycle evidence. Every current or archived
handoff binds the immutable canonical-claim attempt and a digest of the full
normalized fence. A replay may report success only after verifying or
repairing a durable reconciliation receipt. Blocked review and every declared
noncompleted run retain public-safe reason, blocker, normalized fingerprint,
attempt, and next action. The first equivalent failure is attempt 1, the sole
retry is attempt 2 and becomes durably blocked, and a third automatic attempt
is denied across current and archived cycles. Null, malformed, incomplete,
dual, or type-confused claim objects fail closed; every handoff claim attempt
equals the sole retained claim; and legacy lifecycle scalars cannot contradict
detailed evidence. Completed closure still requires QA and Security acceptance,
merged evidence, completed run, and reconciliation.

Independent QA and Security changes-requested the signed v6 checkpoint. They
confirmed the earlier full-fence replay, missing-receipt, recovery-route, native,
and PostgreSQL corrections, but found that claim attempts were not joined across
all handoff history, failure fingerprints were not monotonic across equivalent
retries, and the published binary hash did not reproduce from independent cold
builds. The signed `W-001-lifecycle-correction-v7` grant preserves v6 and permits
only those additive corrections and their qualification.

Independent QA and Security also changes-requested the immutable v7 checkpoint
at head `36d8c981ebde65e694416caf16fc02d50aac2a67`, tree
`be55454779c2c0dd08adc08666c2b7ee3826448f`. They confirmed the v6 corrections
and reproducible builder, but found that case-folded JSON claim aliases could
overwrite canonical fields, active legacy-only metadata could claim terminal
state, and dependency readiness ignored contradictory detailed lifecycle
records. The signed `W-001-lifecycle-correction-v8` grant permits only those
three fail-closed corrections, native parity, qualification, and fresh review.

Independent QA and Security changes-requested the immutable v8 checkpoint at
head `6d6b90ef495cd64286e755e90d199a3cb622cd54`, tree
`f596e2a148f055bcac90960419b2e22928bd471c`. They confirmed the project
adapter's case-folded claim rejection, active legacy-scalar denial, and
detailed-state contradiction handling, but found two remaining parity gaps:
the patched native transaction validated canonical keys only at the top-level
metadata object, and a versioned or claim-bearing dependency could strip its
detailed lifecycle records and fall back to sparse legacy readiness. The
signed `W-001-lifecycle-correction-v9` grant permits only recursive native
canonical-key admission, strict sparse-legacy dependency compatibility, their
qualification, public evidence, and fresh immutable review.

V9 head `ad845ff81f1e64b9e4110162a77a65a844891731`, tree
`e4a08e5a4b211003dc29609a0128856eec306061`, passed every local and
no-skip qualification gate. Public run `33104553091` then exhausted its two
allowed attempts on the same foundation-owned disposable Git-pack cleanup
race, after authority packages had passed. The signed
`W-001-lifecycle-ci-stabilization-v10` grant preserves the V9 runtime,
contracts, qualification bytes, tag, and failed runs; it permits only applying
bounded no-maintenance/no-auto-GC/no-detach configuration to every disposable
test Git command, updating the validator and evidence, and fresh review.

Independent QA and Security changes-requested the immutable V10 checkpoint at
head `47b19b2c89d72fbf9eb5356ceefe33783d691aa4`, tree
`0ebe496c48871b040a7fcd7a286073f2c1d40153`. They verified the V9 runtime and
qualification remained byte-exact and public run `33105792480`, job
`98635155160`, passed, but found two test-fixture fencing defects: one raw
pre-repository clone bypassed the sanitized wrapper
(`ci.test_git_sanitization_incomplete`), and disposable repositories retained
local maintenance configuration despite the command-local-only contract
(`ci.test_git_configuration_persisted`). The signed
`W-001-lifecycle-ci-fencing-v11` grant permits only routing every disposable
Git operation through the single bounded wrapper, removing persistent fixture
configuration, adding fail-closed regressions, and fresh immutable review.

Independent QA accepted the immutable V11 checkpoint at head
`54f4593b1730ff9ae04a2e5cce0589c6baedfee6`, tree
`44ba564be30e0db0aa735d76539c3604a5d79e3f`; Security changes-requested the
same subject after public run `33108126981`, job `98643418071`, passed. The
remaining findings were caller-overridable maintenance fences
(`ci.test_git_fences_caller_overridable`), ambient Git exec/template execution
(`ci.test_git_environment_execution_injection`), and a one-file literal source
guard that missed equivalent process calls (`ci.test_process_guard_fail_open`).
The signed `W-001-lifecycle-ci-hardening-v12` grant permits only absolute
trusted-Git execution, fail-closed argument and environment admission, a
repository-wide AST process allowlist, adversarial regressions, and fresh
immutable review.

Independent QA and Security changes-requested the immutable V12 checkpoint at
head `3c8d55aa39e4e099d8a922f8e13a71efcbe2c78b`, tree
`c4bb80ab477b7fcbe73a7a237479e44703393952`, after public run `33110339883`,
job `98651204635`, passed. Exact Git long-option abbreviations, compact
upload-pack syntax, outside-root Git-directory selection, and config-producing
subcommands escaped the denylist (`ci.test_git_argv_schema_fail_open`). Direct
`exec.Cmd` construction, indirect syscall function values, and nested test
files escaped the nonrecursive process guard
(`ci.test_process_guard_incomplete`). The signed
`W-001-lifecycle-ci-hardening-v13` grant permits only exact per-subcommand Git
argv schemas, recursive closed process admission, the matching adversarial
regressions, and fresh immutable review. W-001 therefore remains
`in-progress`; no merge, canonical lifecycle mutation, or live lease is
authorized by this correction.

Independent QA and Security changes-requested the immutable V13 checkpoint at
head `ce934054aed66c074e99a032191a6a51c620b947`, tree
`73cab7fb7b1bd2fc1102dc4b16e9617fd7c26680`, after public run `33112938711`,
job `98660186954`, passed. A lexically contained clone destination could resolve
through a symlinked ancestor and write outside the disposable root
(`ci.test_git_clone_physical_escape`). Doctrine tests could also call the
production `planningGrantGitOutput` executor directly and bypass the closed
test wrapper (`ci.test_process_guard_transitive_bypass`). The prospective
signed `W-001-lifecycle-ci-hardening-v14` grant permits only canonical physical
clone containment, routing the three historical read-only calls through the
bounded test wrapper, transitive production-executor denial, their adversarial
regressions, and fresh immutable review. W-001 therefore remains
`in-progress`; no merge, canonical lifecycle mutation, or live lease is
authorized by this correction.

Independent QA and Security changes-requested the immutable V14 checkpoint at
head `d631bec4ed786116c13e36995722d91d48d64109`, tree
`b9467f12b2031c5159ef749938bbd4f475eb6153`, after public run `33123061855`,
job `98694494697`, passed. A real disposable root reached through a symlinked
ancestor remained admitted (`ci.test_git_root_ancestor_alias_admitted`), clone
target reservation was released before Git consumed the pathname
(`ci.test_git_clone_reservation_toctou`), and dot-imported `os` or `syscall`
process entrypoints escaped selector-only admission
(`ci.test_process_guard_dot_import_bypass`). The signed
`W-001-lifecycle-ci-hardening-v15` grant permits only removing the writable
clone subprocess, descriptor-binding every remaining test Git process to the
verified physical root, rejecting dot/blank guarded process imports, retaining
the prior adversarial corpus, reproducing the pinned V9 native qualification,
and fresh immutable review. W-001 therefore remains `in-progress`; no merge,
canonical lifecycle mutation, or live lease is authorized by this correction.

Independent QA and Security changes-requested the immutable V15 checkpoint at
head `a46f16deff2fc06c5d0d21377a3bb2c65e873fc9`, tree
`c2e482717f182040708cbf2551ee266de2485a30`, after public run `33165311496`,
job `98829194619`, passed. The V14 findings were closed, but the new descriptor
helper could be called directly outside its intended parent provenance
(`ci.test_git_descriptor_helper_transitive_bypass`), and its mutable Go test
executable pathname could be replaced between resolution and execution
(`ci.test_git_helper_executable_path_toctou`). The prospective signed
`W-001-lifecycle-ci-hardening-v16` grant permits only removing that self-exec
helper surface, opening the root descriptor at command admission, using one
fixed non-input descriptor trampoline and literal system executables, rejecting
dynamic or transitive process bypasses, restricting fetch to canonical local
sources, and fresh qualification and review. W-001 therefore remains
`in-progress`; no merge, canonical lifecycle mutation, or live lease is
authorized by this correction.

Independent QA accepted and Security changes-requested the immutable V16
checkpoint at head `25d2f14e20e74f1415caa4118a93c359f9370031`, tree
`d9bf0e3f89807c12c5be5a58ea68fd04715aa740`, after public run
`33206197037`, job `98967743138`, passed. A copied invocation or extracted
executor field could bypass the wrapper-level one-shot transition
(`ci.test_git_invocation_one_shot_field_bypass`), the canonical local fetch
source remained replaceable before Git consumed its pathname
(`ci.test_git_fetch_source_toctou`), and same-package tests could call the
ambient arbitrary-argv production `gitOutput` executor outside the test
constructor inventory (`ci.test_process_guard_refresh_executor_bypass`). The
signed `W-001-lifecycle-ci-hardening-v17` grant permits only moving one-shot
consumption into shared captured state, replacing path-based fetch with a
source- and destination-descriptor-bound pack stream, deriving and denying the
closed production process-entry inventory from tests, retaining the prior
regression corpus, and fresh qualification and review. W-001 therefore remains
`in-progress`; no merge, canonical lifecycle mutation, live lease, or
production effect is authorized by this correction.

This is candidate implementation evidence only. F-002 scenarios remain
`failing`, M3-W001 remains `in-progress`, and no canonical lifecycle mutation
is authorized until a signed checkpoint receives independent QA and Security
acceptance and protected-main reconciliation.

## Success evidence

- A fresh clone follows G-001 → PD-002 → work-authority specification
  → F-002 → this plan → M3-W001 without needing the operational database to
  understand intended behavior.
- The plan checker accepts exactly one selected backlog Bead only during
  `contract-publication`, and rejects an active implementation row in that
  phase.
- In `delivery`, the checker requires exactly one `in-progress` or `in-review`
  row and requires it to match the current Bead.
- W-001 later proves CAS/version conflicts, dependency rejection, monotonic
  fencing, owner-only heartbeat, synthetic stale-effect denial, ordered event
  replay, coherent projection rebaseline, and denial of direct Beads/Dolt access.
- QA and Security independently validate the same immutable W-001 commit before
  the Orchestrator can record a completed disposition or close the Bead.
- Contract-publication CI proves every feature commit has the pinned SSH
  signature and the signed publication tag preserves the exact reviewed tree
  across the protected squash merge.

## Falsification evidence

The hypothesis is false if contract publication creates a hidden claim; the
plan and Beads disagree without a blocking finding; two workers can claim the
same Bead; a stale epoch passes the synthetic pre-effect boundary; an
agent can reach Beads/Dolt directly; a projection becomes writable authority;
event truncation silently loses work; or W-001 reaches `done` without required
review, completed disposition, remote durability, and reconciliation.

## Failure ownership and convergence

Classify every failure as `foundation-owned`, `deployed-owned`, or
`mixed-or-unclear` before remediation. Provider outage and authority-substrate
failure are foundation/runtime findings and cannot silently create customer
product work. One automatic retry is allowed per normalized fingerprint;
equivalent recurrence records a durable `blocked` disposition and escalates.

Only the Delivery Orchestrator changes dependency order or the selected Bead.
During contract publication, the signed one-time grant permits only its listed
paths and effect intents/receipts without an implementation claim. During
W-001 implementation, the signed delivery grant supplies bounded source
authority while the lease service does not yet exist. Once self-hosted fencing
exists, every mutation must pass the current authoritative state,
exact required transition, allowed corrective action, and live epoch checks;
the bootstrap grant is then unusable.
