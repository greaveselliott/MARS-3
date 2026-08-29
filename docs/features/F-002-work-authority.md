# F-002 — Fenced work authority

**Status:** Active
**Goal:** G-001
**Product decision:** PD-002
**Product specification:** `docs/product-specs/work-authority.md`
**Canonical Bead:** M3-W001 (display ID W-001)

## Business logic

1. Beads/Dolt is the sole authority for work definition, dependency DAG,
   lifecycle, owner, claim, handoff, review verdict, run disposition, blockers,
   retry fingerprints, and declared exclusive paths.
2. PostgreSQL behind the gateway is the sole authority for live capability and
   path leases, SaaS identity bindings, and a factory-issued monotonic
   `lease_epoch` within a non-reusable `fence_generation`. The generation is
   anchored by canonical recovery authority so store restore cannot revive an
   old tuple. PostgreSQL never becomes a second ticket store.
3. Temporal schedules and retries execution but cannot claim, lease, transition,
   review, dispose, or close work.
4. Agents, models, sandboxes, public clients, workflow workers, humans, and
   administrators may interact with authority only through authenticated,
   tenant-scoped, typed gateway operations. Model output is an untrusted
   proposal, not permission. Break-glass remains a gateway operation with
   explicit human approval; it is never raw datastore access.
5. Ready selection validates canonical dependency outcomes. A claim uses the
   observed WorkVersion in a compare-and-swap and, except for the one bounded
   W-001 bootstrap described below, grants no capability until a non-overlapping
   live lease is durably issued and verified.
6. A new attempt or reacquisition receives an epoch greater than every prior
   issue in its tenant/project fence generation. Generations and their epochs
   are never reused; renewal changes neither value.
7. Every material effect revalidates tenant, project, Bead, attempt, fence
   generation, epoch, base SHA, current claim/version, capability, path,
   idempotency, labels, and policy immediately before execution.
8. The authority journal is an ordered rebuild feed, never an authority. A gap,
   truncation, or checkpoint conflict invalidates the projection and requires a
   full canonical rebaseline.
9. Every W-001 operation carries immutable data/capability labels and the
   gateway rejects missing, forged, or locally all-three lethal-trifecta label
   combinations. Authority credentials remain sealed capabilities, and public
   records contain bounded metadata and opaque references rather than raw
   payloads or private state. Cross-worker transitive taint is S-001 work.
10. Missing receipts and cross-store partial outcomes are unknown or pending
    reconciliation, never inferred success. Retried normalized effects preserve
    idempotency.

## Step-by-step behavior

1. An authenticated caller requests a tenant/project-scoped ready view from the
   gateway. The gateway reads Beads/Dolt, checks typed lineage, lifecycle,
   blockers, dependency review and run outcomes, declared paths, and returns the
   canonical Bead version plus a bounded readiness explanation.
2. A policy-approved claim request supplies M3-W001, that expected WorkVersion, a
   unique attempt, immutable base SHA, normalized declared paths, requested
   capability, caller-proposed taint additions, and an idempotency key. The
   gateway derives authoritative labels from trusted handles and attestations;
   caller input can add taint but cannot remove or downgrade it.
3. The gateway records an effect intent, re-reads the Bead, checks the caller and
   complete claim preconditions, then performs a version-guarded Beads claim.
   Exactly one concurrent request can win.
4. The gateway verifies the canonical fence generation, transactionally
   allocates a strictly newer PostgreSQL epoch and a non-overlapping live
   implementation lease. It returns a capability only
   after the claim and lease receipts reconcile; otherwise the operation is
   non-executable and enters idempotent recovery or compensation.
5. A worker presents the full fencing tuple for every write. During W-001, a
   synthetic broker conformance harness performs a just-in-time gateway
   validation before a simulated effect and records a bounded receipt. Earlier
   validation, a Temporal token, or cached projection state is insufficient.
   S-002 later applies this contract to real external-effect brokers.
6. A handoff binds the current execution attempt separately from the immutable
   canonical claim attempt, persists a digest of the complete normalized fence,
   releases the exact owning implementation lease, and moves the canonical
   Bead to `in-review` by one expected-version native transaction. An unknown
   post-release outcome is re-read from Beads and may report replay success
   only after the original receipt is verified or a deterministic
   reconciliation receipt is appended; the released lease cannot authorize
   another write. QA and then Security record only their own verdict against
   the same immutable commit, and every review operation rejects an active
   implementation lease for that Bead.
7. `changes-requested` or `blocked` review reopens M3-W001 rather than creating
   a duplicate. Every noncompleted run is recorded with public-safe failure
   context and a normalized fingerprint; an explicit `in-review` run retains
   review state while other noncompleted outcomes reopen the same Bead. The
   first occurrence is attempt 1. One equivalent retry is attempt 2 and must
   become durably `blocked`; a third automatic occurrence is denied across
   current and archived cycles. Rehandoff archives the earlier cycle and its
   run history, and a subsequent attempt receives a newer epoch.
   Only the Delivery Orchestrator records the completed run and merge
   reconciliation, then requests `done` after accepted reviews. The terminal
   record retains exactly one complete type-specific WorkClaim or
   BootstrapClaim, detailed public-safe evidence, and append-only prior cycles
   after closure. Null, malformed, incomplete, dual, or type-confused claim
   objects fail closed. Every current and archived handoff must name the sole
   retained claim attempt, and legacy lifecycle scalars are absent or exactly
   derived from their detailed records. Authority metadata keys must also be
   exact canonical spellings at every typed object boundary; duplicate,
   case-folded, or otherwise unknown aliases fail before decoding. When an
   active record has no detailed lifecycle evidence, accepted, completed,
   reconciled, and blocker legacy fields must be empty. Dependency readiness
   uses validated detailed lifecycle evidence when present and rejects any
   contradiction with its legacy projection.
8. Projection consumers replay journal events exactly once in sequence. On an
   irrecoverable gap, truncation, unknown checkpoint, or version conflict, they
   mark the view stale and non-authorizing, discard it, then ask the gateway for
   a coherent canonical baseline. The gateway's project barrier blocks new
   canonical, lease, journal, and pre-effect operations, drains or cancels
   in-flight mutations, and reconciles pending claim/lease sagas. It then reads
   Beads, reads live PostgreSQL leases in a repeatable-read snapshot with the
   journal watermark, validates claim-to-lease consistency, and double-reads
   Beads before release. A changed read retries or fails; only a verified stable
   cut may resume replay.

## Scenario schedule

| Scenario | State | Verification owner | Required evidence |
| --- | --- | --- | --- |
| F-002-S1 | failing | QA | gateway-only transition and role-separation conformance |
| F-002-S2 | failing | QA | deterministic ready-set plus concurrent compare-and-swap claim tests |
| F-002-S3 | failing | QA | monotonic epoch, lease exclusivity, expiry, and restart tests |
| F-002-S4 | failing | Security | stale-worker and synthetic just-in-time broker conformance tests |
| F-002-S5 | failing | Security | direct-access, credential-absence, tenant-boundary, and local label-admission tests |
| F-002-S6 | failing | QA + Security | ordered replay, truncation, full-rebaseline, and public-trace tests |

M3-W001 explicitly groups all six scenarios. None may become `passing` from a
design claim or unit test alone; each requires the evidence class named below
and the immutable review chain required by the canonical Bead.

### F-002-S1 — Gateway-only canonical mutation

**Given** M3-W001 exists in canonical Beads/Dolt state with typed lifecycle,
lineage, owner, declared paths, and verification order
**And** W-001 conformance clients and synthetic worker fixtures have no
datastore credentials or network route to Beads, Dolt, or the PostgreSQL lease
tables
**When** an authenticated, policy-approved principal performs an allowed work
transition through the typed authority gateway
**Then** the gateway compare-and-swap mutates only the concern owned by the
appropriate authoritative store
**And** it emits an intent, policy decision, verified receipt, and ordered
journal event with one trace identity
**And** a model proposal, chat message, Temporal task, cache write, direct
datastore request, wrong-role verdict, or invalid lifecycle transition cannot
produce the canonical change
**And** the denial states the bounded current state, required transition, and
exact safe operation available next.

### F-002-S2 — Ready selection and claims are compare-and-swap guarded

**Given** a ready query returns a canonical WorkVersion and integrity digests
**And** every dependency has its required accepted reviews, completed run
disposition, terminal reconciliation, and no active blocker
**When** two authorized attempts race to claim the same Bead or overlapping
declared paths using that expected version
**Then** exactly one compare-and-swap claim and compatible implementation lease
succeeds
**And** every loser receives a stale-version or lease-conflict response without
overwriting the winner
**And** changing then restoring a dependency, blocker, lifecycle, owner,
lineage link, or declared path between selection and claim still advances its
monotonic revision and makes the observed WorkVersion stale
**And** deleting and recreating the same Bead ID produces a new non-reusable
incarnation and cannot revive the old observation
**And** replaying the winning idempotency key returns the verified result while
reusing it for different normalized input is rejected.

### F-002-S3 — Lease epochs are monotonic and base-bound

**Given** a verified canonical claim for one tenant, project, Bead, attempt,
base SHA, capability, and normalized path set
**When** the gateway issues, renews, expires, revokes, releases, or reacquires an
implementation lease across process/database-client restarts or attempts to
start from a restored or reinitialized lease store
**Then** issuance is durable before capability delivery
**And** renewal retains the current fence generation and epoch while every new attempt or
reacquisition receives a strictly larger, never-reused epoch
**And** restore, rollback, or reinitialization cannot issue or validate a lease
until a human-approved recovery binds a new non-reusable fence generation to
canonical project authority; every token from an earlier generation is stale
**And** the lease remains bound to the original tenant, project, Bead, attempt,
base SHA, capability, and paths
**And** no second active implementation lease can cover the Bead or an
overlapping normalized path
**And** a partial Beads/PostgreSQL outcome grants no authority and converges by
idempotent reconciliation or compensation.

### F-002-S4 — Synthetic pre-effect validation rejects stale authority

**Given** an attempt once held a valid implementation lease
**And** the W-001 synthetic broker harness is about to record a simulated
filesystem, source-control, or other material external write
**When** the lease expires or is revoked, a newer epoch or fence generation is issued, the base SHA
drifts, claim/version or labels change, the path escapes scope, or the tenant,
project, Bead, or attempt differs
**Then** a just-in-time gateway check rejects the effect before it occurs
**And** an earlier successful check, cached projection, retry, or Temporal
checkpoint cannot authorize the late write
**And** the receipt records a bounded denial linked to the original intent
without including raw arguments, output, credentials, or private source
**And** the response identifies the exact safe read, rebase, reclaim, or human
approval transition available next.

### F-002-S5 — Direct authority access and local label admission fail closed

**Given** an unprivileged synthetic client can receive external-untrusted model
or repository fixture input
**When** it probes files, environment, network, service discovery, public API,
cross-tenant identifiers, or forged capability tokens for direct authority
access
**Then** it cannot obtain Beads, Dolt, PostgreSQL, or gateway service credentials
**And** network and application admission deny direct datastore access and
cross-tenant reads or mutations
**And** sealed credentials are never added to model context, tool payloads,
traces, evidence, or sandbox state
**And** the gateway derives labels from authenticated principal/resource
handles, accepted repository classification, signed capability attestations,
and policy registry state; caller-supplied labels are taint-only proposals
**And** it rejects missing or forged provenance, any attempted label downgrade,
and any single normalized W-001 operation whose derived route combines `external-untrusted`,
`private-data`, and `external-effect`
**And** only a compartmentalized or explicitly human-mediated safe transition
may proceed
**And** the result explicitly makes no claim about cross-worker, cross-tool,
retry, or handoff taint propagation, which remains S-001 work.

### F-002-S6 — Ordered journal recovery rebaselines after truncation

**Given** a projection consumer has a verified baseline watermark and last
applied sequence
**When** it receives duplicate, delayed, missing, conflicting, or truncated
authority journal events, or restarts with an unknown checkpoint
**Then** duplicates are idempotent and only contiguous events apply
**And** an unfillable gap, truncation, or canonical-version conflict marks the
projection stale and prevents it from authorizing mutation
**And** the consumer discards derived state and the gateway's project-scoped
barrier blocks new canonical, lease, journal, and pre-effect operations; drains
or cancels in-flight mutations; and reconciles every pending cross-store saga
before it reads the Beads authority generation, issue incarnations and
mutation sequences, and dependency-graph revision
**And** it captures live leases in a PostgreSQL repeatable-read snapshot with a
declared journal watermark, validates each claim/lease or pending-saga
relationship, then double-reads Beads before releasing the barrier
**And** a changed Beads read, failed barrier, or inconsistent digest retries or
fails closed as non-authorizing; only a verified stable cut may resume strictly
after the watermark
**And** neither rebaseline nor Temporal history writes back to canonical state
**And** public journal/evidence output contains only schema versions,
tenant-safe synthetic identifiers, sequences, labels, hashes, bounded outcomes,
and opaque trace references.

## Permissions, validations, state transitions, and failures

### Permissions

- Observers can read only label-filtered projections for their tenant/project.
- A claim requires an authenticated principal, an explicit policy-approved
  `work.claim` capability, a ready canonical Bead, and the expected version.
  The proposed assignee cannot self-authorize through model output.
- The active implementation principal may record bounded status and handoff
  entries only for its claimed Bead and active lease scope.
- Only the owning attempt may heartbeat or renew its unexpired lease within the
  bounded maximum duration, or release that exact lease. Renewal preserves its
  fence generation or epoch and cannot change the base SHA, capability, paths,
  or owner. An expired,
  handed-off, or mismatched lease cannot be renewed.
- The Delivery Orchestrator and an explicitly authorized Security/policy
  principal may revoke a lease with a durable reason, but cannot acquire or
  transfer the implementation lease. Expiry is automatic; release and revoke
  never make an epoch reusable.
- QA and Security may append only their ordered review verdict for the exact
  immutable commit and may not hold an implementation lease during review.
- Only the Delivery Orchestrator may record dependency replans, terminal run
  disposition, reconciliation, and a prerequisite-complete `done` transition.
- Handoff, verdict, run, reconciliation, and terminal idempotency keys are
  unique across the current and archived review cycles. An exact historical
  replay returns its verified record only after the matching durable receipt
  exists or a deterministic reconciliation receipt is appended. A different
  request using that key fails. Every handoff record retains the immutable
  canonical-claim attempt and a digest of the complete normalized fence.
- Every noncompleted run has a public-safe normalized fingerprint. The first
  occurrence uses attempt 1; the sole equivalent retry uses attempt 2 and
  records `blocked`; any third equivalent automatic occurrence is rejected,
  including when earlier occurrences are in archived review cycles.
- A skill, profile maximum, provider session, Temporal task, or cached event
  does not grant permission.

### Validations

Every mutating request validates authentication, tenant/project binding,
principal and mode, effective capability, policy/data labels and W-001-local
label admission, canonical Bead version, goal/decision/feature/scenario
lineage, lifecycle and role transition, dependencies and blockers, declared
normalized paths, attempt, fence generation, epoch, base SHA, expiry, idempotency, and trace
correlation. The synthetic broker conformance path repeats all current fencing
checks immediately before its simulated effect.

The canonical `WorkVersion` is the opaque tuple of project authority
generation, non-reusable issue incarnation, monotonic issue mutation sequence,
and project dependency-graph revision. These values live in canonical Beads
metadata and advance even when A→B→A restores identical visible content. A
deleted/recreated ID receives a new incarnation. Native Beads row version and
Dolt commit are audit/concurrency inputs, not substitutes. A ready token also
binds integrity digests over dependency outcomes and goal/decision/feature/
scenario/owner/blocker/path lineage. Claim revalidates every version component
and digest; digests prove integrity, never freshness by themselves.

### State transitions

```text
backlog --claim+lease--> in-progress --handoff--> in-review
backlog --signed W-001 atomic bootstrap claim (no capability)--> in-progress
in-review --changes-requested--> in-progress
in-review --blocked review+failure context--> in-progress
in-review --blocked|failed|preempted|cancelled|no-work|changes-requested run+failure context--> in-progress
in-review --in-review run--> in-review --accepted reviews+completed run--> in-review
in-review --accepted chain+merge+completed run+reconcile--> done
backlog|in-progress|in-review --authorized supersession--> superseded

lease: none --issue--> active(generation,epoch) --renew--> active(same generation,epoch)
lease: active --release|revoke|expire--> inactive --new issue--> active(same generation,new epoch)
lease-store: restore|rollback|reinit --> disabled --human recovery--> new generation,epoch 1
```

There is one declared bootstrap sequence: after the accepted F-002 contract,
a separately signed, human-directed W-001 grant binds the canonical claim,
attempt, immutable base commit, exact W-001 paths, publication effects, expiry,
  expected WorkVersion, exact `backlog/open/unclaimed` →
  `in-progress/in_progress/claimed` CAS transition, pinned Beads client and
  binary hash, idempotency key, and public-safe receipts. That first grant uses
one reviewed helper transaction to enter `in-progress`, but grants no
implementation capability. A second signed grant must reconcile the plan and
manifest to the verified canonical postimage; only a later bounded delivery
grant may authorize building the gateway without falsely asserting that its
not-yet-built lease exists. The sequence is not transferable or autonomous,
cannot authorize production/destructive effects, and its delivery exception
becomes unusable as soon as self-host conformance is accepted.

The reviewed helper tree alone cannot exercise the first transition. After its
squash merge and protected-main check, a distinct one-hour signed execution
authorization must bind the actual merged SHA/tree, review tag and feature
commit, QA and Security accepted commit, exact patched binary, attempt,
idempotency key, authority project, signed preimage, and opaque canonical
workspace-instance digest. The authorization is canonical JSON with one
trailing newline; its exact payload and detached-signature digests are part of
the in-memory identity. After conformance the helper reloads that exact object
and rejects any signed-byte change. Before the effect it revalidates both
expiries, resolves the fixed public GitHub `main` under a sanitized Git
environment, rejects ambient Beads/Dolt workspace overrides, and rechecks the
strict embedded metadata, direct database filesystem identity, absence of any
`.beads/redirect`, and complete preimage. The patched operation also disables
redirect following and repeats those direct-store checks inside the transaction
before mutation. A local squash, mutable remote, copied or redirected
workspace, swapped token, or command error cannot be accepted; command
uncertainty blocks retry until separately authorized reconciliation.
Every later mutation uses the normal live-lease transition above. P-001 and all
subsequent work receive no equivalent exception.

A blocked review and every noncompleted run retain public-safe `reason`,
`blocked_by` where applicable, normalized failure fingerprint, bounded attempt
count, and `next_action`. Except for an explicit `in-review` run, they reopen
the same Bead to `in-progress`; a later handoff archives the earlier review and
run history. Missing evidence or an unknown receipt cannot take any transition
to `done`.

### Failure behavior

The gateway fails closed on malformed or forged identity, tenant mismatch,
stale expected version, unmet or changed dependency, lineage mismatch, blocker,
invalid state/role transition, duplicate review, path overlap or escape, stale
or expired generation/epoch, base-SHA drift, capability mismatch, invalid or locally
all-three labels, authority outage, event uncertainty, and receipt uncertainty.
It returns a
stable error class and corrective operation without reflecting sensitive input.
One replay is permitted for the same normalized idempotent operation; an
equivalent repeated failure is recorded and escalated rather than looped.

## Required evidence

### Deterministic evidence

- Contract/schema tests for every typed read, mutation, denial, transition, and
  stable error response.
- A deterministic ready-set fixture proving lineage, dependency, blocker, role,
  and lifecycle validation against canonical Beads versions.
- Recursive canonical-key fixtures for every native authority metadata object,
  plus dependency fixtures proving that versioned, claim-bearing, empty/null,
  and detail-key-bearing records cannot downgrade to sparse legacy readiness.
- A→B→A fixtures for issue fields and dependency edges, plus delete/recreate of
  the same display ID, proving mutation sequence, graph revision, and
  incarnation—not content digests—make every old observation stale.
- Contention tests with concurrent claims and overlapping paths proving one
  winner, no lost update, and idempotent replay.
- Persistent generation/epoch tests across renewal, release, expiry,
  revocation, reacquisition, process restart, stale database clients, restored
  snapshots, rollback, and reinitialization. Old generations never validate.
- Synthetic broker race tests that revoke or supersede a lease between
  scheduling and the last pre-effect check and prove zero simulated late
  effects. This qualifies the gateway contract, not a production broker.
- Partial-outcome and outage tests for Beads, PostgreSQL, journal, and Temporal,
  proving fail-closed behavior and deterministic reconciliation.
- Journal replay tests for duplicates, order, gaps, truncation, unknown
  checkpoints, baseline digest failure, full rebaseline, and replay watermark.
- Label provenance tests proving labels are server-derived or signed by a
  trusted issuer, caller proposals can only add taint, and forged/downgraded
  labels fail before admission.
- Public-safe record tests proving every material W-001 authority event is
  emitted deterministically to the conformance sink rather than probabilistically
  sampled, and prohibited payload/private fields never enter logs, events,
  errors, fixtures, or Git evidence. T-001 owns the durable trace backend.

### Independent QA evidence

QA uses only public synthetic fixtures and black-box gateway clients to verify
the six scenarios, ordered role handoffs, restart behavior, one-winner
concurrency, truthful failure messages, deterministic rebaseline, and that
Temporal/cache loss cannot mutate or fabricate authority. QA records an
`accepted`, `changes-requested`, or `blocked` verdict against one immutable
commit and does not repair implementation during review.

### Independent Security evidence

Security adversarially tests forged and stale versions, epochs, attempts,
idempotency keys, identities and capabilities; path normalization and overlap;
cross-tenant isolation; direct datastore reachability; credential absence;
synthetic late-write races; confused-deputy and replay paths; W-001-local label
admission and credential sealing; and public-record redaction. Security reviews only the exact commit
accepted by QA and records its own canonical verdict.

The W-001 evidence manifest includes relative paths, commands, versions,
outcomes, hashes, opaque trace references, residual risks, and the canonical
Beads reconciliation reference. It includes no raw request/response payloads,
private source, tenant data, credentials, backend addresses, or developer
machine metadata.

### Dogfood and release evidence

W-001 is an enabler and may reach `done` after deterministic evidence,
independent QA and Security acceptance, merge, completed run disposition, and
Git/Beads reconciliation. That does not release product value. D-001 Dogfood
must later re-run a staged public operator journey that selects and claims a
synthetic Bead, observes a bounded denial, loses an epoch before a simulated
effect, and recovers a stale projection. Release remains blocked until that
dogfood evidence and the Release verdict bind the then-current gateway commit;
any intervening authority or policy change requires fresh verification.

## Falsification evidence

F-002 is falsified if any W-001 conformance client, synthetic worker, cache,
public client, or direct datastore route can mutate canonical work state; two
contenders can
both claim or lease overlapping scope; a fence generation/epoch is reused or a stale worker can
write; a base-SHA, path, tenant, or attempt mismatch reaches an effect; a
projection silently skips history or survives truncation without a coherent
barriered rebaseline; the W-001 gateway accepts forged, missing, or locally
all-three labels; a credential or raw
payload appears in public traces; PostgreSQL or Temporal is treated as ticket
authority; or `done` is possible without the exact accepted review chain,
completed run, merge, and reconciliation.

## Out of scope

- Implementing the T-001 OpenTelemetry/Tempo trace backend, S-001 transitive
  Rule-of-Two engine, I-001 Git publication saga, S-002 real sandbox/effect brokers,
  or any model/runtime adapter.
- Product repair by QA or Security during independent review.
- Autonomous mutation, production effects, destructive migrations, or trust
  escalation.
- A Markdown ticket mirror, a PostgreSQL copy of canonical Bead lifecycle, or
  using Temporal history/event projections as authority.
- Semantic correctness claims from DocSync, tracing, or successful transport
  alone.

## Descoped scenarios

None. Only the CEO strategy principal may descope an in-scope scenario through
a superseding product decision, affected-goal analysis, and active-plan update.
The Work Authority Engineer, reviewer, or Delivery Orchestrator cannot descope
a scenario through a claim, failure, or run disposition.
