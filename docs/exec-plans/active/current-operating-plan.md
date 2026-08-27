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
absent. W-001 therefore remains `in-progress` under the separately signed
`W-001-lifecycle-completion-v5` correction; no live lease exists.

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
current completion attempt adds the missing governed lifecycle surface without
relabeling the accepted core checkpoint as terminally complete.

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
grants neither a lease nor source-code authority; the separately signed
lifecycle-completion grant supplies the current bounded source authority.

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
- Required next transition: bind the lifecycle-completion implementation to a
  signed immutable checkpoint, execute the non-skipped native Beads and
  PostgreSQL conformance suites, then route that exact tree through QA and
  Security. No canonical handoff or later lifecycle mutation may execute until
  the reviewed tree is merged and a separate reconciliation authority binds
  the protected-main result.

The W-001 bootstrap grant is deliberately not a live lease: it is
human-directed, binds one base commit and attempt, permits only canonical W-001
paths plus required evidence/publication, and expires when the gateway passes
self-host conformance. The first PostgreSQL epoch is W-001 acceptance evidence,
not a prerequisite for building the epoch service. The authoritative Bead
holds exclusive paths and mutable lifecycle; this plan does not copy lease
values or represent a proposed owner as a current grant.

## Lifecycle-completion candidate

The current bounded candidate adds typed handoff, ordered review verdict, run
disposition, merge reconciliation, and terminal routes. Each canonical change
uses one expected-version native Beads transaction. Handoff releases the exact
owning development lease first, separates the current execution attempt from
the immutable canonical claim attempt, and recovers unknown outcomes through
canonical readback and idempotent retry. Review and disposition routes reject
any active implementation lease. `changes-requested` preserves the rejected
cycle and reopens the same Bead; terminal closure requires QA and Security
acceptance, a completed run, merge evidence, and reconciliation, and retains
evidence references plus claim lineage after close.

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
