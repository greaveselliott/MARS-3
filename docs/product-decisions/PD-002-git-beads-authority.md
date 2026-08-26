# PD-002 — Split Git and Beads authority

**Status:** Accepted
**Owner:** CEO (`strategy`)
**Date:** 2026-08-26
**Goals:** G-001
**Features:** F-001, F-002
**Supersedes:** None
**Superseded by:** None

## Decision

Git is authoritative for durable product intent and reviewed state: goals,
specifications, product decisions, BDD, the single active execution plan,
architecture, code, evidence, and releases. An external Beads/Dolt store is
authoritative for work definition, DAG, lifecycle, owner, claim, handoff, run
disposition, blockers, retry state, and declared exclusive paths. After W-001,
the gateway and PostgreSQL own live capability/path leases and the
factory-issued monotonic `lease_epoch`; pinned Beads v1.2.2 does not. Temporal
executes workflows and is never an authority source.

The two authorities meet at stable identifiers, commit hashes, trace
references, review verdicts, and run dispositions. A reconciled view can be
regenerated; it cannot silently resolve disagreement. No Markdown ticket
lifecycle is allowed.

## Alternatives considered

- Git-only Markdown tickets. Rejected because concurrent claims and work-state
  transitions need transactional compare-and-swap semantics.
- Beads/Dolt as the product record. Rejected because a clone must retain the
  product contract, reasoning, and reviewed evidence without an operational
  database.
- Dual writable copies of ticket state. Rejected because conflict resolution
  would make both stores ambiguous.

## Consequences

Work-state mutations pass through the authority gateway after W-001. Before it
exists, the signed, non-autonomous bootstrap exception is limited to H-001's
external claim, public-safe lifecycle/evidence/review/run records, and
creation/claim routing for W-001 and P-001; each effect has an intent and
receipt. Closing work requires Git evidence plus successful Beads
reconciliation; neither side alone can claim delivery. Beads owns the review
verdict and run disposition. Git stores the immutable reviewed artifact and a
redacted evidence manifest that references the canonical Beads record and its
digest.
