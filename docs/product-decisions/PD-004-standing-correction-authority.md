# PD-004 — Ticket-lifetime authority for bounded corrections

**Status:** Accepted
**Owner:** Repository owner (`strategy`)
**Date:** 2026-08-31
**Goals:** G-001
**Features:** F-002
**Supersedes:** The per-correction human-approval requirement in PD-002
**Superseded by:** None

## Decision

The operational purpose is to let the initial build complete without constant
repository-owner involvement in routine delivery mechanics. Owner attention is
reserved for a new ticket boundary, material scope or risk expansion, and the
retained approval classes below.

One explicit claim or owner authorization for a specific Bead delegates its
complete non-production correction loop through accepted handoff. A fresh user
authorization is not required for each CI, QA, or Security correction when the
change stays inside that Bead's canonical goal, feature, declared paths,
effect class, and owning Git contract.

The bounded route may preserve rejected evidence, implement an in-scope
correction, create signed commits and a distinct signed review tag, push the
bound branch and tags, open the bounded successor pull request, run exact-head
CI, obtain independent QA followed by independent Security review, merge after
both accept the same tree, read back protected main, and perform gateway-only
reconciliation for the same Bead. Rejection returns the same Bead and route to
bounded correction; it does not require another grant document or chat prompt.

This is ticket-lifetime human delegation, not autonomous mutation or a trust
escalation. It ends at the accepted handoff and never transfers to another
Bead. The current Bead, its canonical scope, signatures, CI, independent
review order, gateway boundary, protected-main policy, retry limit, and public
evidence remain mandatory.

Explicit owner approval remains required for production or release effects;
destructive or irreversible operations; secrets, credentials, billing,
private data, or external accounts; trust escalation; another Bead or
downstream ticket; expansion of the goal, feature, declared paths, or effect
class; direct Beads, Dolt, PostgreSQL, or other authority-store writes; and
workflow, dependency, runtime, API, or database-design changes outside the
specific ticket contract.

One equivalent correction retry is allowed. A second equivalent failure is
recorded as blocked and escalated as a design problem. Another same-class
Security recurrence stops incremental patching and requires architectural
simplification.

Delivery also converges one pull request at a time for the bound Bead. An
accepted PR is merged and read back before the plan advances. A rejected or
superseded PR is not merged: its immutable head, tag, CI, and review disposition
are preserved, a public disposition is attached, and the PR is closed before a
successor opens. This prevents stale open PRs from accumulating while ensuring
that closing a rejected vehicle cannot erase its evidence.

## Alternatives considered

- Continue issuing a signed V-number grant for every finding. Rejected because
  repeated owner approval added no new scope decision and obscured the actual
  CI and independent-review gates.
- Create a repository-wide standing mutation grant. Rejected because an
  unspecified future target would weaken the authority boundary.
- Enable global autonomous mutation. Rejected because it would erase the
  claimed-Bead, trust, production, destructive-action, and authority-store
  boundaries.
- Allow self-approved merges after local tests. Rejected because exact-head CI
  and independent QA and Security acceptance are completion evidence, while
  local tests alone are not.

## Consequences

`.harness/standing-correction-authority.yaml` is the machine-readable current
binding. `.harness/manifest.yaml` projects whether it is active. Historical
grants, tags, runs, and rejected reviews remain immutable evidence, but no new
per-correction grant or detached grant signature is created for a correction
covered by the current ticket binding. A later Bead requires its own explicit
claim or owner authorization; scope expansion still stops for an owner
decision.
