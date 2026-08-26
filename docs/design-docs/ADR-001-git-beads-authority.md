# ADR-001 — Git/Beads authority and reconciliation

**Status:** Accepted
**Date:** 2026-08-26
**Owners:** Product authority, Delivery Orchestrator
**Goal:** G-001
**Decision:** PD-002
**Feature:** F-001

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

The stable join is `{tenant, project, bead, attempt, epoch, base_sha,
commit_sha, trace_ref, review_verdict, run_disposition}`. Operational views are
projections and can be rebuilt. They are never authoritative writes.

Every Beads work definition also carries explicit `goalIds`, `featureId`,
applicable `productDecisionIds`, and `scenarioIds`. Those identifiers must
equal the Git lineage selected by the active plan. A missing link is an
authority reconciliation failure: Git may describe product intent, but it may
not silently supplement an incomplete canonical work definition.

M3-H001 is the bootstrap exception: a signed charter records the effect that
created and claimed the work item before the gateway existed. The exception is
non-autonomous and limited to H-001's claim, public-safe lifecycle/evidence/
review/run updates, and creation/claim routing for W-001 and P-001; each
external operation records an effect intent and receipt. H-001 does not claim
lease fencing. After W-001,
direct Beads/Dolt access is denied to agents; compare-and-swap claims and lease
renewals pass through the typed gateway, with live leases stored in PostgreSQL.

## Invariants

- One Bead may have many jobs and run attempts. Its Beads record declares
  exclusive paths, while PostgreSQL holds at most one compatible active
  implementation lease for those paths.
- The gateway issues a monotonic `lease_epoch` after validating Beads claim,
  dependencies, attempt identity, declared paths, and base commit. A stale
  epoch fails closed and returns the required read/transition.
- Every external write revalidates the complete tuple `(tenant, project, bead,
  attempt, epoch, base_sha)` immediately before the effect.
- Temporal can schedule or retry an attempt but cannot create a claim, lease,
  review verdict, run disposition, or lifecycle transition.
- The active Git plan can name current Bead identifiers and expected evidence,
  but it cannot set ownership or lifecycle state.
- A claim attestation must bind the same goal, feature, product decisions, and
  scenarios as the canonical Bead and Git plan; doctrine validation rejects an
  omitted or divergent lineage link.
- No Markdown ticket lifecycle tree is created.
- Chat and database projections cannot be the only record of a material
  product decision.
- `done` requires an immutable Git commit, required accepted review verdicts,
  a completed run disposition, and a successful reconciliation receipt in
  Beads.

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
   epoch, and base SHA immediately before the external effect.
3. Publish through the trusted Git integrator.
4. Record a bounded receipt with immutable Git identifiers.
5. Verify the remote state and required checks.
6. Attach accepted QA and Security review verdicts to the reviewed commit.
7. Record the completed run disposition and compare-and-swap the Bead to its
   terminal lifecycle state.

Retries reuse the idempotency key and reconcile before repeating an effect.

## Consequences

The authority gateway and PostgreSQL lease store are security boundaries for
mutating work. A missing Beads store prevents work transitions; a missing or
expired live lease prevents external writes. Neither prevents a public clone
from understanding product intent. A Git/Beads/lease mismatch produces an
explicit reconciliation incident; no side silently wins.
