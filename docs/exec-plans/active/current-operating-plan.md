# Active operating plan — MARS-3 walking skeleton

**Status:** Active
**Owner:** Delivery Orchestrator
**Updated:** 2026-08-26
**Goal:** G-001
**Current feature:** F-001
**Current Bead:** M3-H001 (display ID H-001)
**Authority:** Beads/Dolt for work state; Git for this durable plan

This is the only active execution plan. It is a Git-owned ordering and evidence
contract, not a second ticket database.

## Durable lineage

- Goal: [G-001](../../goals/active.md)
- Decisions: [PD-001](../../product-decisions/PD-001-public-first.md),
  [PD-002](../../product-decisions/PD-002-git-beads-authority.md), and
  [PD-003](../../product-decisions/PD-003-provider-neutral.md)
- Product promise: [foundation specification](../../product-specs/foundation.md)
- Behavior contract: [F-001](../../features/F-001-doctrine-foundation.md)
- Work authority: external Bead `M3-H001`; this Git plan is a one-way link,
  never lifecycle authority.

## Current hypothesis and walking skeleton

If a public clone can prove one authority route, pinned doctrine, observer-first
trust, public-safe evidence, and independent review verdicts before any runtime is
introduced, later agents can be added without granting model output implicit
authority.

The walking skeleton is H-001: doctrine plus deterministic validation only.
It relies on the signed external Beads bootstrap record, not on the gateway or
lease fencing that W-001 will build. Runtime, UI, provider, and production
surfaces remain excluded.

## Scenario priority

1. F-001-S1 — one durable delivery route.
2. F-001-S2 — offline doctrine provenance.
3. F-001-S3 — observer-first trust.
4. F-001-S4 — immediate public disclosure safety.

These scenarios are an explicit H-001 group because the public gate cannot be
accepted safely without all four.

## Delivery waves

| Wave | Bead | Owner | Depends on | State | Exit evidence |
| --- | --- | --- | --- | --- | --- |
| 0 | H-001 Doctrine foundation | Foundation Maintainer | signed genesis | in-review | public-safe harness, provenance, BDD, plan, validation CLI |
| 1 | W-001 Work Authority | Work Authority Engineer | H-001 | backlog | Beads gateway, CAS claims, PostgreSQL lease epochs, direct access denied |
| 1 | P-001 Local substrate | Platform Engineer | H-001 | backlog | Lima/k3s, OIDC, RLS, Temporal, object storage |
| 2 | T-001 Trace spine | Trace Engineer | W-001, P-001 | backlog | audit ledger, OTel/Tempo, effect intents and receipts |
| 3 | S-001 Rule-of-Two policy | Security Engineer | T-001 | backlog | labels, taint, tool contracts, hard admission policy |
| 4 | I-001 Git/evidence reconciliation | Integration Engineer | W-001, T-001, S-001 | backlog | PR publication saga and merged-evidence closure |
| 4 | S-002 Secure effects | Security/Platform Engineer | P-001, T-001, S-001 | backlog | gVisor, brokers, credential proxy, deterministic publication |
| 5 | A-001 Runtime contracts | Runtime Architect | T-001, S-001, S-002 | backlog | adapter conformance, qualification, routing |
| 5 | UI-001 Operator workspace | Frontend Engineer | P-001, T-001, S-001 | backlog | shared React/Electron workspace and trace/security views |
| 6 | C-001 Codex adapter | Runtime Engineer | A-001, S-002 | backlog | contained Codex ticket execution |
| 6 | L-001 Colibri advisory | Model Runtime Engineer | A-001, P-001 | backlog | local advisory generation with mutation disabled |
| 7 | E-001 Public first-slice fixture | Engineer → QA → Security | I-001, UI-001, C-001, L-001 | backlog | public fixture delivery and merged PR |
| 8 | C-002 Claude parity | Runtime Engineer | E-001 | backlog | Claude and mixed-routing conformance |
| 9 | D-001 Dogfood/release | Dogfood/Release | C-002 | backlog | compartments, trust ledger, re-verification, approval |
| 10 | K-001 Skills/code graph | Foundation Maintainer | D-001 | backlog | signed, licensed, quarantined capability registry |
| 11 | O-001 Hosted hardening | Platform/Security | K-001 | backlog | multi-tenant isolation and capacity suite |

No backlog Bead is claimable before its dependencies have `accepted` review
verdicts and a `completed` run disposition. Parallel entries in one wave still require distinct exclusive
paths and capability leases.

## H-001 claim contract

- Work type: enabler
- Owner: Foundation Maintainer
- Coordinator: Delivery Orchestrator
- Risk: high
- Failure ownership: foundation
- Scenarios: F-001-S1 through F-001-S4
- Verification order: `qa` → `security-reviewer` → `delivery-orchestrator`
- Branch: `codex/h-001-doctrine-foundation`
- Expected evidence: all public commit gates, immutable commit identifier,
  generated-provenance scope test, trust-default test, QA review verdict,
  Security review verdict, completed run disposition, and Git/Beads
  reconciliation receipt.

Exclusive paths are held in the authoritative Bead. This plan deliberately
does not copy mutable claim or lease data.

## Success evidence

- A fresh clone follows G-001 → PD-001/PD-002/PD-003 → F-001 → this plan →
  M3-H001 without ambiguity.
- Doctrine, plan, DocSync, public, test, vet, diff, and Git history scans pass
  on one immutable commit.
- Offline provenance verifies the pinned commit, file paths, blobs, license,
  adaptations, exclusions, and generated-only refresh scope.
- All roles load as observers, with autonomous mutation disabled.
- QA and Security independently return `accepted` for the same commit, the run
  records `completed`, then the Orchestrator reconciles it to the Bead.

## Falsification evidence

The hypothesis is false if any check requires private state to understand the
product contract; Git and Beads claim the same mutable authority; a provider or
role can self-escalate; raw payloads or local identity enter evidence; doctrine
refresh changes project-owned files; review targets differ; or H-001 reaches
`done` before both independent review verdicts, a completed run disposition,
and reconciliation.

## Failure ownership and convergence

Classify every failure as `foundation-owned`, `deployed-owned`, or
`mixed-or-unclear` before remediation. This foundation plan can create only
foundation-owned work. One automatic retry is allowed per normalized failure
fingerprint. A repeat records `blocked` with the current state, required
transition, exact allowed corrective action, and human escalation.

Only the Delivery Orchestrator changes dependency order. A replan records the
affected Beads, old and new order, evidence, and decision owner in Git. During
H-001 the authorized bootstrap operator reconciles the external Beads record
with effect intents and receipts; W-001 moves later reconciliation behind the
authority gateway.
