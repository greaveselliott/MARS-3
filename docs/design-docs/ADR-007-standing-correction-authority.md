# ADR-007 — Ticket-lifetime bounded-correction authority

**Status:** Accepted
**Date:** 2026-08-31
**Owners:** Product authority, Delivery Orchestrator
**Goal:** G-001
**Decision:** PD-004
**Features:** F-002

## Context

The initial build must be able to converge without making the repository owner
act as a manual acknowledgement service for every routine recovery step.

W-001 used a sequence of signed one-attempt grants while bootstrapping the
gateway and qualifying immutable publication evidence. Once a candidate failed
CI, QA, or Security, even a correction inside the same paths and effect class
paused for another owner message and another V-number document. That repeated
an already-settled scope decision, lengthened recovery, and encouraged
procedural patches instead of convergence.

The repository still needs a hard distinction between delegated correction
and autonomous authority. The current Bead and Git contract define what may
change. The gateway defines how canonical work state changes. CI and
independent review define whether a tree can merge. Production, destructive
effects, future work, secrets, trust, and scope expansion are separate owner
decisions.

## Decision

Adopt `ticket-lifetime-non-production-correction-v1`. Keep
`autonomous_mutation: disabled`. Each activation is bound to one specifically
claimed or owner-authorized Bead, exact base, purpose, paths, effects, retry
limit, and terminal handoff. It never authorizes an unspecified future Bead.

Within that binding the Delivery Orchestrator may continue correction,
publication, accepted merge, readback, and gateway-only current-Bead
reconciliation without seeking another user authorization. A correction does
not create a new V-number grant artifact. The binding ends at accepted handoff.

The route requires signed commits and a distinct signed tag, exact-head CI,
independent QA then independent Security review of the same tree, merge only
after acceptance, protected-main readback, immutable rejected evidence, and no
more than one equivalent retry.

The following always stop the route and require explicit owner approval:

- production or release effects;
- destructive or irreversible effects;
- credentials, secrets, billing, private data, or external-account access;
- trust escalation;
- another Bead or a downstream ticket;
- expansion of the current goal, feature, declared paths, or effect class;
- direct mutation of Beads, Dolt, PostgreSQL, or another authority store; and
- workflow, dependency, runtime, API, or database-design changes outside the
  bound ticket contract.

One equivalent failure may be corrected. The second equivalent failure becomes
a durable blocked disposition and architecture escalation. A repeated
same-class Security finding ends incremental correction and routes to design
simplification.

## State model

```text
one explicitly bound Bead
        |
        v
bounded correction -> signed candidate -> exact-head CI -> QA -> Security
        ^                                                   |
        |                 changes-requested, under limit     |
        +---------------------------------------------------+
                                                            |
                                                     accepted tree
                                                            |
                                                            v
                                              merge -> main readback
                                                            |
                                                            v
                                                accepted handoff -> end
```

Any retained approval boundary exits this state model and waits for an owner
decision. The second equivalent failure exits to `blocked`; it does not loop.

## Current W-001 binding

The initial binding starts at preserved V4 head
`96ec3410b16d381b102e6c1c0bd36e5ea9a9e426`, tree
`417ec5233fe0f6ec438643837c2170774a700dd9`, and preserves signed tag object
`78058a1083b841b54fc7a8d0a2be4a14d2890f00`, exact-head CI
`33340046444/99333914035`, and QA `changes-requested`. It permits the exact
paths listed in `.harness/standing-correction-authority.yaml`, one successor
pull request, and ends after accepted merge plus protected-main readback.
Canonical W-001 is already `done`, so this binding authorizes no lifecycle or
lease mutation.

## Integrity and enforcement

The machine-readable binding is strict YAML at
`.harness/standing-correction-authority.yaml`. Doctrine validation rejects
missing, altered, widened, duplicated, or unsafe fields. The manifest names
the policy and projects `none-required-under-ticket-lifetime-authority` instead
of an active V-number grant.

The policy never supplies a lease, capability, reviewer verdict, or merge
result. Those remain independently verifiable facts. Model output, chat,
automation, and schedules may coordinate the route but cannot widen it.

## Migration

All earlier W-001 grants, commits, review tags, runs, QA/Security dispositions,
and pull requests remain immutable historical evidence. This decision
supersedes only the need to mint another correction grant inside the current
binding. It does not retroactively accept rejected work.

The W-001 binding fixes the two stale active-plan statements identified by V4
QA, then runs the normal signed candidate, exact-head CI, QA, Security, merge,
and readback sequence. Future Beads persistence work is a separate downstream
design decision and cannot start under this W-001 binding.

## Consequences

- Routine in-scope recovery continues without owner-message latency.
- The meaningful approval boundary becomes a new ticket or scope/risk
  expansion, not each correction attempt.
- Independent gates remain mandatory and cannot be self-approved.
- Historical grant machinery remains readable for provenance but is not the
  normal correction route inside an active ticket binding.
- Losing the canonical current-Bead scope is a blocker, not permission to
  infer it.
