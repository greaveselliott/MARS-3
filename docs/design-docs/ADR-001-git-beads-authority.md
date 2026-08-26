# ADR-001 — Git/Beads authority and reconciliation

**Status:** Accepted
**Date:** 2026-08-26
**Owners:** Product authority, Delivery Orchestrator
**Goal:** G-001
**Decision:** PD-002
**Features:** F-001, F-002

## Context

Git is durable, reviewable, cloneable, and ideal for product intent and
evidence. Beads backed by Dolt provides canonical mutable work definitions and
lifecycle. Pinned Beads v1.2.2 does not provide the end-to-end fencing epoch
needed to guard live filesystem and external effects. An operational database
should not be required to understand the public product contract.

## Decision

Use distinct, non-overlapping authorities:

| Concern | Authority |
| --- | --- |
| goals, specs, product decisions, BDD, one active plan | Git |
| architecture, code, tests, reviewed artifact/evidence manifests, releases | Git |
| work definition, DAG, lifecycle, owner, claim, handoff, review verdict, run disposition | Beads/Dolt |
| declared exclusive paths, blockers, retry fingerprints | Beads/Dolt |
| live capability/path lease and monotonic `lease_epoch` | PostgreSQL behind the gateway |
| workflow execution, retry, timers | Temporal; explicitly not an authority |

The stable join is `{tenant, project, bead, attempt, fence_generation, epoch, base_sha,
commit_sha, trace_ref, review_verdict, run_disposition}`. Operational views are
projections and can be rebuilt. They are never authoritative writes.

Every Beads work definition also carries explicit `goalIds`, `featureId`,
applicable `productDecisionIds`, and `scenarioIds`. Those identifiers must
equal the Git lineage selected by the active plan. A missing link is an
authority reconciliation failure: Git may describe product intent, but it may
not silently supplement an incomplete canonical work definition.

Review routing uses executable identifiers, not display names. Every entry in
the Beads `verificationOrder` and signed claim `verification.order` must be the
exact ID of a declared role or of a profile whose `principal_id` resolves to a
declared role. The two ordered lists must agree. An alias such as
`qa-reviewer` is not routable when the registry declares only `qa`, even if a
human can infer the intended reviewer.

M3-H001 was the genesis exception: a signed charter records the effect that
created and claimed the work item before the gateway existed. That authority
ended when H-001 closed. A separately signed, one-use contract-publication
grant now permits only the exact Wave-1 Git paths and public effects needed to
publish F-002/F-003 without claiming either Bead. After the contract is
accepted, W-001 receives one final human-directed bootstrap grant binding its
canonical claim, attempt, base commit, exact paths, public effects, and expiry.
It does not assert a live lease and expires at accepted self-host conformance.
P-001 is blocked by W-001 and receives no such exception. Thereafter direct
Beads/Dolt access is denied to agents; compare-and-swap claims and lease
renewals pass through the typed gateway, with live leases stored in PostgreSQL.

Wave-1 contract shaping exposed two bootstrap-authority incidents before any
claim: the P-001 blocker was added outside the planning grant's enumerated
effects, and the first recovery path did not bind every intermediate metadata
and description write precisely enough. The original signed recovery artifact
is checksum-bound in Beads but is not committed because the pinned secret
scanner classifies two public checksum fields as generic credentials. The
separately signed `WAVE-1-recovery-disposition` is prospective and
non-retroactive: it binds the reconstructible public state snapshot, permits
only one exact P-001 description postimage, and leaves all earlier effects
pending independent QA/Security disposition. It grants no claim, lease,
implementation, trust, ruleset, or scanner exception.

The repository retains its linear, squash-only public policy. To preserve the
reviewed signed branch history across GitHub's squash rewrite, final
contract-publication CI requires the signed annotated tag
`mars3/wave1-contract-publication-v1`. The tag target descends from the exact
grant base, remains reachable, and has the same tree as the reviewed PR head.
The protected-main squash commit must have the same tree. A tag mismatch,
unsigned feature commit, missing evidence, altered base, or untrusted runner
event fails closed.

### W-001 bootstrap grant contract

The accepted F-002 contract is the durable schema authority for the one future
W-001 bootstrap grant. Before any delivery-phase change, the human bootstrap
authority creates `.harness/grants/W-001-bootstrap.yaml` and detached
`.sig`, signs the exact bytes under SSH namespace
`mars3-w001-bootstrap-grant`, and verifies them with the pinned genesis public
key. The first signed transition commit adds a fail-closed offline validator;
until that validator exists, the human verifier follows this accepted schema
and records the command outcome without exposing signer state.

The strict document contains exactly:

- schema version `1`, kind `MARS3ImplementationBootstrapGrant`, public
  classification, ID `W-001-bootstrap`, signer
  `human-bootstrap-authority`, coordinator `delivery-orchestrator`, Bead
  `M3-W001`, display ID `W-001`, and branch
  `codex/w-001-work-authority`;
- RFC 3339 issue and expiry times no more than 72 hours apart; one immutable
  40-lowercase-hex accepted-contract base commit; one public-safe unique
  attempt ID; expected authority generation, non-reusable issue incarnation,
  positive issue mutation sequence, positive dependency-graph revision;
  lowercase SHA-256 dependency and Git-lineage digests; the exact
  `backlog/open/unclaimed` → `in-progress/in_progress/claimed` CAS transition;
  pinned Beads source revision and binary SHA-256; and one idempotency key;
- `autonomousMutation: false`, `liveLeaseAsserted: false`,
  `productionEffects: false`, and exact verification order
  `qa`, `security-reviewer`, `delivery-orchestrator`;
- exact preclaim paths for the grant and signature, validator, public helper,
  pinned Beads source patch, tests, doctrine/spec/BDD updates, bootstrap
  evidence, and third-party notices;
- exact implementation path patterns equal to the canonical M3-W001
  `exclusivePaths`; a mismatch blocks the claim rather than widening either
  side; and
- allowed effects limited to pinned-state verification, disposable atomicity
  conformance, publication of the reviewed helper tree, one expected-preimage
  CAS claim of M3-W001, and one public-safe claim receipt. It explicitly
  prohibits plan/manifest reconciliation, gateway implementation, other-Bead
  or unlisted mutation, asserting a lease before verified issuance,
  production/destructive effects, autonomous mutation, trust escalation,
  credentials, and provider or customer data.

The helper patches the pinned Beads source in a temporary directory and uses
the native `ClaimIssueInTx` primitive plus metadata and lifecycle-label updates
inside one outer Dolt transaction. The pinned v1.2.2 `update --claim
--metadata` route is not used because it commits those effects in separate
transactions; `batch` without this reviewed patch has no claim CAS. Raw SQL and
hidden local code are prohibited.

Unknown fields, YAML aliases/anchors/tags/flow syntax, duplicate keys,
unresolved paths, changed signed bytes, signer drift, stale expected version or
digests, path disagreement, expiry, and any actual diff outside the signed path
union fail closed. The canonical claim must succeed before implementation. The
claim grant cannot reconcile the Git plan or start implementation. A separate
signed postclaim reconciliation grant must bind the verified claim receipt,
accepted helper commit, and exact plan/manifest postimage. A later delivery
grant remains bounded until self-host conformance verifies the gateway and the
first real W-001 lease; all subsequent effects use the accepted gateway.

## Invariants

- One Bead may have many jobs and run attempts. Its Beads record declares
  exclusive paths, while PostgreSQL holds at most one compatible active
  implementation lease for those paths.
- The gateway verifies a canonical, non-reusable `fence_generation` and issues
  a monotonic `lease_epoch` within it after validating Beads claim,
  dependencies, attempt identity, declared paths, and base commit. A stale
  generation or epoch fails closed and returns the required read/transition.
- Restored, rolled-back, or reinitialized lease storage cannot issue or
  validate leases until a human-approved recovery anchors a new generation in
  canonical project authority; old-generation tokens never become current.
- The canonical WorkVersion contains project authority generation,
  non-reusable issue incarnation, monotonic issue mutation sequence, and
  project dependency-graph revision. A→B→A still advances a revision, and
  delete/recreate assigns a new incarnation. Native row version/Dolt commit and
  dependency/lineage digests are additional concurrency/integrity inputs, not
  freshness substitutes.
- Every external write revalidates the complete tuple `(tenant, project, bead,
  attempt, fence_generation, epoch, base_sha)` immediately before the effect. The sole temporary
  exception is an effect explicitly enumerated by the signed W-001 bootstrap
  grant before the lease service can exist; it remains human-directed and
  cannot be transferred, widened, or used after self-host conformance.
- Temporal can schedule or retry an attempt but cannot create a claim, lease,
  review verdict, run disposition, or lifecycle transition.
- The active Git plan can name current Bead identifiers and expected evidence,
  but it cannot set ownership or lifecycle state.
- The plan phase `contract-publication` selects exactly one canonical backlog
  Bead while explicitly recording that no claim exists. It permits no
  `in-progress` or `in-review` delivery row. The phase changes to `delivery`
  only after the contract is accepted and the canonical claim exists; then
  exactly one active row must match the selected Bead.
- A claim attestation must bind the same goal, feature, product decisions, and
  scenarios as the canonical Bead and Git plan; doctrine validation rejects an
  omitted or divergent lineage link.
- A claim attestation's verification order must resolve entry-by-entry against
  the executable role/profile registry; missing, duplicate, malformed, or
  undeclared reviewer identities fail closed.
- No Markdown ticket lifecycle tree is created.
- Chat and database projections cannot be the only record of a material
  product decision.
- `done` requires an immutable Git commit, required accepted review verdicts,
  a completed run disposition, and a successful reconciliation receipt in
  Beads.

The harness manifest is a Git-owned projection of the selected Bead, plan
phase, claim presence, and mirrored lifecycle. Validation rejects disagreement
between that projection and the active plan. Neither file can create or repair
canonical Beads state.

An authority projection is non-authorizing during full rebaseline. After
W-001, human/admin/break-glass writes and every other canonical/lease/journal
mutation use the gateway project barrier. Rebaseline blocks new mutations and
pre-effect validations, drains or cancels in-flight work, and reconciles
pending cross-store sagas before reading. It reads canonical Beads state,
captures live PostgreSQL leases and the journal high-watermark in one
repeatable-read snapshot, validates claim/lease consistency, then double-reads
the authority generation, issue incarnations/mutation sequences, and
dependency-graph revision. It releases the barrier only after a verified stable
cut. A changed read, unavailable store, inconsistent saga, or digest mismatch
retries or fails closed; no consumer combines two uncoordinated snapshots.

## Pinned Beads compatibility mapping

The pinned Beads release has a smaller native status vocabulary than the
factory lifecycle. The authoritative Bead therefore stores typed
`lifecycle_state` metadata alongside its native compatibility status:

| Factory lifecycle | Native compatibility status |
| --- | --- |
| `backlog` | `open` |
| `in-progress` | `in_progress` |
| `in-review` | `in_progress` |
| `done` | `closed` |
| `superseded` | `closed` with a supersession reason |

Consumers must read the typed metadata and expected Bead version; they may not
infer `in-review`, `done`, or `superseded` from the compatibility status alone.
Review verdicts, run dispositions, and handoffs remain distinct canonical
records on the Bead. Git evidence holds their immutable references and digests,
not a second writable verdict. W-001 makes these transitions typed and compare-and-swap
guarded; H-001 records the same distinctions through the signed bootstrap
procedure.

## Reconciliation

Publication is a saga, not a distributed transaction:

1. Record an effect intent with the complete fencing tuple, proposed commit,
   data labels, and idempotency key.
2. Revalidate policy, Beads claim, declared paths, live PostgreSQL lease,
   epoch, and base SHA immediately before the external effect, or verify that
   the exact still-active W-001 bootstrap grant enumerates this pre-conformance
   effect.
3. For this bootstrap publication, create and push the one signed immutable
   tree-attestation tag; after W-001, publication uses the trusted Git
   integrator and live fence.
4. Publish the PR through the trusted Git integrator and record a bounded
   receipt with immutable Git and tag identifiers.
5. Verify the remote branch, tag, protected-main tree identity, and required
   checks.
6. Attach accepted QA and Security review verdicts to the reviewed commit.
7. Record the completed run disposition and compare-and-swap the Bead to its
   terminal lifecycle state.

Retries reuse the idempotency key and reconcile before repeating an effect.

## Consequences

The authority gateway and PostgreSQL lease store are security boundaries for
mutating work. A missing Beads store prevents work transitions; outside the
one signed W-001 bootstrap scope, a missing or expired live lease prevents
external writes. Neither prevents a public clone from understanding product
intent. A Git/Beads/lease mismatch produces an explicit reconciliation
incident; no side silently wins.
