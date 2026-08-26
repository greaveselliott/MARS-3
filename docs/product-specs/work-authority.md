# Work authority specification

**Status:** Current
**Goal:** G-001
**Decision links:** PD-002
**Feature contract:** F-002
**Canonical work item:** M3-W001 (display ID W-001)

## Promise

MARS-3 exposes one authenticated, policy-gated authority gateway for selecting,
claiming, fencing, and transitioning work. Operators and agent runtimes can see
why work is ready, obtain a bounded implementation lease, and receive an exact
corrective action when a request is rejected. Model output, Temporal history,
chat, and database projections never become work authority.

The gateway preserves the split established by PD-002:

| Concern | Authoritative store |
| --- | --- |
| Work definition, dependency graph, lifecycle, owner, claim, handoff, review verdict, run disposition, blockers, retry fingerprints, and declared exclusive paths | Beads/Dolt |
| Live capability and path leases, non-reusable fence generation plus monotonic epochs, and SaaS identity bindings | PostgreSQL behind the gateway, with generation anchored by canonical recovery authority |
| Goals, product decisions/specifications, BDD, the active plan, architecture, code, reviewed evidence, and release records | Git |
| Workflow timers, retries, and execution checkpoints | Temporal, which has no authority to mutate any row above |

## Governed operations

The product contract exposes typed operations rather than datastore access:

- Read a tenant/project-scoped work item, ready set, dependency explanation,
  public-safe event stream, or full projection baseline.
- Claim a ready Bead with an expected Bead version, attempt identity, immutable
  base commit, normalized declared paths, capability scope, and idempotency key.
- Renew, release, revoke, or validate a live lease without changing canonical
  Bead history.
- Record a role-authorized status, handoff, review verdict, run disposition, or
  lifecycle transition using compare-and-swap.
- Revalidate the complete fencing tuple immediately before a material effect.

Every mutation records `intent -> policy decision -> execution receipt ->
verification -> run disposition` correlation. A successful response means the
authoritative write and its bounded receipt were verified. A timeout or missing
receipt is `unknown` or `pending-reconciliation`, never success.

## Ready selection and claim

1. The gateway authenticates a caller outside model context and resolves one
   tenant, project, principal, profile, and allowed operation.
2. It reads the canonical Beads record and WorkVersion, validates its Git lineage,
   typed lifecycle, owner, blockers, declared paths, verification order, and
   dependency states, then returns a ready decision with that observed version.
3. The caller submits that expected version plus a unique attempt, base SHA,
   requested capability, and idempotency key. A model proposal by itself is not
   a claim or permission.
4. The gateway re-reads the canonical record and performs a version-guarded
   claim. Competing or stale requests lose without overwriting the winner.
5. PostgreSQL verifies the canonical non-reusable `fence_generation`, then
   allocates a `lease_epoch` strictly greater than every epoch previously
   issued in that generation, binds both to the
   Bead, attempt, base SHA, capability, and non-overlapping paths, and persists
   the lease before any capability is returned.
6. The gateway verifies both sides of the logical claim saga. A partial outcome
   grants no capability and is reconciled idempotently before retry or
   compensation.

`WorkVersion` is the opaque tuple of project authority generation,
non-reusable issue incarnation, monotonic issue mutation sequence, and project
dependency-graph revision, all stored in canonical Beads metadata. Field and
edge A→B→A changes still advance a sequence/revision; delete/recreate receives
a new incarnation. Native row version and Dolt commit remain additional audit
inputs, not substitutes. Each ready token also binds integrity digests over
dependency outcomes and goal/decision/feature/scenario/owner/blocker/path
lineage. Claim revalidates every version component and digest; a digest alone
never establishes freshness.

Lease renewal preserves the current generation and epoch. Expiry, revocation, release, claim
transfer, or a new attempt requires a newly issued, never-reused epoch. At most
one implementation lease may cover the same Bead or overlapping normalized
exclusive path at a time.

A restored, rolled-back, or reinitialized lease store is disabled for issuance
and effect validation until a human-approved recovery binds a new non-reusable
fence generation through canonical project authority. Earlier-generation
tokens remain stale even if an epoch number collides.

Only the owning attempt may heartbeat or renew its unexpired lease within its
bounded maximum duration, or release that exact lease. Renewal cannot change
the epoch, owner, base SHA, capability, or paths. The Delivery Orchestrator and
an explicitly authorized Security/policy principal may revoke with a durable
reason, but cannot acquire or transfer the implementation lease. Expired,
released, revoked, handed-off, and mismatched leases cannot be renewed.

## Effect fencing

Every trusted mutating broker must ask the gateway to validate
`(tenant, project, bead, attempt, fence_generation, lease_epoch, base_sha)` immediately before the
external effect. Validation also confirms the current canonical claim and
version, active capability, unexpired path lease, requested path/action scope,
data labels, policy decision, and idempotency key.

A check performed earlier in a workflow is insufficient. Lease loss, a newer
epoch, base-SHA drift, claim transfer, path escape, tenant/project mismatch, or
policy-label change cancels execution and rejects every late effect. The denial
returns the current safe state, required transition, and exact allowed next
operation without echoing credentials or payloads.

## State and role behavior

- A normal `backlog -> in-progress` transition requires a ready,
  version-matched claim and a verified live implementation lease. The sole
  W-001 bootstrap claim is the declared exception below and grants no tool,
  path, or implementation capability.
- `in-progress -> in-review` requires the owning attempt's public-safe evidence
  reference and handoff. The implementation lease no longer authorizes writes
  after handoff.
- An ordered reviewer may record only its own verdict against the exact
  immutable commit. `changes-requested` returns the same Bead to `in-progress`;
  a fresh implementation lease receives a newer epoch.
- `done` requires the accepted review chain, merged immutable Git evidence,
  `completed` run disposition, and a successful reconciliation receipt. Only
  the Delivery Orchestrator can request that terminal transition.
- A blocked attempt remains in its truthful lifecycle with `blocker`,
  `blocked_by`, normalized failure fingerprint, and exact next action. It is not
  silently closed or duplicated.
- `superseded` requires an authorized reason and successor; it never masquerades
  as delivered value.

W-001 has one non-autonomous bootstrap sequence because it implements the
gateway that will fence subsequent work. Its first separately signed human
grant publishes an atomic claim helper and permits exactly one canonical
W-001 claim; that claim grants no implementation capability and cannot change
the Git plan or manifest. A second signed postclaim grant must bind the
verified receipt and exact Git reconciliation. Only then may a separately
bounded delivery grant let the claimed attempt build and qualify the lease
service without pretending that its not-yet-built lease exists. None of these
grants authorizes production or destructive effects, is transferable, or is
available to P-001 or later work.

The publication grant is not sufficient to execute the claim. After merge,
protected-main CI, and immutable QA and Security acceptance, a separate
one-hour human-signed execution authorization must bind the actual accepted
commit/tree and the reviewed helper digest. The helper must reject locally
synthesized `main`, stale or missing review/check facts, an expired token,
workspace/preimage drift, and any run whose acceptance becomes unknown.

ADR-001 defines the exact signed-file schema, signer namespace, static effects,
transition paths, and implementation-path equality rule for that grant. The
delivery phase cannot begin from prose alone: the signed grant, pinned-state
CAS claim, and initial offline validator must agree on the same base, attempt,
Bead version, dependency/lineage digests, and canonical paths. A mismatch
leaves W-001 in `backlog` or records a blocked bootstrap attempt. The grant
also binds the exact CAS transition, pinned Beads client revision and binary
hash, and one idempotency key; it is not a general direct-client permission.

Observers may read only tenant-scoped, label-filtered projections. Contributor
operations require the exact policy-approved capability; maximum trust, a role
name, a skill, a model response, or a Temporal task is never sufficient.
Engineers cannot record their own review verdicts, reviewers cannot obtain an
implementation lease during review, and Temporal cannot claim, lease,
transition, review, or dispose work.

## Projection and recovery

The gateway publishes an ordered, append-only authority journal for rebuilding
operational views. Events carry a schema version, tenant-safe stable IDs,
monotonic sequence, canonical Bead version or lease epoch as applicable,
bounded before/after hashes, labels, outcome, and opaque trace references. The
journal and its consumers are projections, not a fourth authority.

After W-001, every canonical Beads mutation—including human, administrator,
maintenance, and break-glass operations—and every lease/journal mutation
passes through the gateway's project barrier. Consumers apply events once in
sequence and persist a checkpoint. Duplicates
are idempotent. A gap, out-of-order event that cannot be filled, unknown
checkpoint, journal truncation, or canonical-version conflict marks the view
stale and stops mutation routing through that view. The consumer discards the
  derived state and requests a coherent gateway baseline. The gateway blocks
  new canonical, lease, journal, and pre-effect operations, drains or cancels
  in-flight mutations, and reconciles pending claim/lease sagas. It then reads
  Beads; captures live leases and the journal high-watermark in a PostgreSQL
  repeatable-read snapshot; validates claim-to-lease/pending-saga consistency;
  and double-reads authority generation, issue incarnations/mutation sequences,
  and dependency-graph revision before releasing the barrier. A changed read or inconsistent digest
  retries or fails closed; only a verified stable cut may resume replay after
  its watermark. Rebaseline never writes back to either authority, and an
  incomplete baseline is always non-authorizing.

Beads unavailability disables work-state mutation. PostgreSQL or fencing
unavailability disables lease issuance and all effects. Journal uncertainty
forces reconciliation or rebaseline. Temporal unavailability can pause jobs but
does not change canonical state.

## W-001 label boundary and public evidence

Every request, projection, journal event, and synthetic broker route carries
immutable data and capability labels derived from authenticated principal and
resource handles, accepted repository classification, signed capability
attestations, and policy registry state. Caller labels are proposals that may
only add taint. W-001 rejects missing/forged provenance or a downgrade attempt
and any one normalized gateway operation that combines `external-untrusted`,
`private-data`, and `external-effect`.
Authority credentials are sealed broker capabilities and never model context
or sandbox environment data. An untrusted model proposal can reach a mutation
only through a separately authorized, typed command whose route does not also
expose private data; confidential routes require compartmentalized or explicit
human mediation. S-001 later implements taint propagation and hard admission
across workers, tools, retries, and handoffs; W-001 does not claim that engine.

Public traces and Git evidence contain only synthetic or tenant-safe IDs,
relative paths, schema and component versions, labels, rule identifiers,
hashes, counts, bounded error classes, outcomes, and opaque trace references.
They never contain raw Bead descriptions, prompts, completions, private
reasoning, source fragments, tool arguments/output, credentials, private tenant
state, backend addresses, or provider session state.

## Failure contract

Mutations fail closed for authentication or tenant mismatch, insufficient
capability, stale Bead version, unmet dependency, lineage divergence, blocker,
invalid lifecycle transition, duplicate reviewer, incompatible path lease,
expired or stale fence generation/epoch, base-SHA mismatch, invalid path, missing, forged, or
locally all-three labels, receipt uncertainty, or unavailable authority store.
Idempotent replay returns
the prior verified result; reusing an idempotency key for a different normalized
operation is rejected.

Every rejection exposes only:

1. the current bounded state and labels;
2. the violated rule or conflicting version/epoch;
3. the required transition; and
4. the exact safe read or corrective operation available next.

## Completion and release boundary

W-001 may close only when F-002-S1 through F-002-S6 pass deterministic,
concurrency, restart, stale-worker, tenant-isolation, independent QA, and
independent Security verification on one immutable commit. The evidence must be
public-safe and reconciled to M3-W001. Because W-001 is an enabler, closure is
not a product release. D-001 Dogfood must later re-verify a staged synthetic
operator journey against the current authority commit; any intervening
authority or policy change invalidates that evidence. Release remains blocked
until dogfood and Release verdicts are recorded.

## Out of scope

- Implementing Temporal workflows, the trace backend, hard Rule-of-Two policy,
  Git publication brokers, sandboxes, provider adapters, or the operator UI.
- Treating PostgreSQL, Temporal, an event stream, a cache, Git plan text, or
  chat as canonical Bead work state.
- Autonomous mutation, production publication, destructive migrations, or
  trust escalation.
- Copying Beads/Dolt data into Git or exposing direct database and credential
  access to agents, public clients, or browser sessions.
