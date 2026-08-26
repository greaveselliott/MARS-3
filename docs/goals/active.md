# Active goals

**Owner:** CEO (`strategy`)
**Updated:** 2026-08-26

## G-001 — Establish a public, governed software-factory foundation

Create a public-first foundation from which MARS-3 can safely deliver a
provider-agnostic software factory. The foundation must make authority,
evidence, provenance, trust, tracing, and security mechanically inspectable
before any autonomous mutation or production effect is possible.

**Priority:** P0
**Confidence:** Medium until H-001 independent review
**Source/dedupe:** MARS-3 public-first plan and signed genesis charter; unique
active foundation goal
**Supports:** future governed project delivery and provider/runtime choice
**Conflicts:** feature throughput that bypasses public, authority, evidence, or
security gates
**Review trigger:** H-001 rejection; authority-model change; secret incident;
or first end-to-end fixture evidence

### Hypothesis

If authority, doctrine provenance, observer-first trust, trace correlation,
Rule-of-Two admission, and public disclosure checks are established before a
runtime can mutate, later adapters and agents can add capability without
granting model output implicit authority.

### Success measures

- A fresh public clone has one durable delivery route and exactly one active
  execution plan.
- Git and Beads/Dolt have explicit, non-overlapping authority, with one current
  claimed Bead and no Markdown lifecycle shadow system.
- MARS doctrine provenance validates offline at the pinned revision, and an
  intentional refresh can modify generated provenance only.
- Every principal has a separately declared maximum trust and an initial
  effective trust of `observer`; autonomous mutation is disabled.
- Trace-spine doctrine defines how effect intent, policy, receipt,
  verification, and run disposition will correlate without public raw payloads;
  T-001 supplies runtime evidence.
- Rule-of-Two doctrine defines transitive admission over external-untrusted
  input, private data, and external effects; S-001 supplies enforcement evidence.
- The complete public commit gate passes before publication, with independent
  QA and Security review verdicts attached to an immutable commit.

### Priority trade-offs

Safety, truthful authority, and reproducibility take priority over feature
throughput. A provider adapter, operator interface, or runtime service that
cannot preserve these controls is deferred rather than special-cased.

### Explicit non-goals for H-001

- No MARS runtime or user-facing MARS concept.
- No provider SDK, model runtime, agent sandbox, hosted service, or production
  control plane.
- No autonomous mutation or production publication.

### Falsification evidence

The hypothesis is false if a public clone needs private state to understand
intent; Git and Beads overlap mutable authority; Beads is claimed to provide
live fencing it lacks; an adapter can self-authorize; traces require raw
payloads; Rule-of-Two can be bypassed by composition; provenance refresh edits
project-owned files; or H-001 closes without same-commit QA and Security
review.
