# ADR-002 — Trace-correlated effect and evidence spine

**Status:** Accepted
**Date:** 2026-08-26
**Owners:** Trace Engineer, Security Engineer
**Goal:** G-001
**Feature:** F-001

**Implementation state:** Declared contract; runtime enforcement is T-001.

## Context

Agents cross context, model, policy, tool, sandbox, verification, Git, and work
authority boundaries. Logs collected independently at each boundary cannot
prove which proposal caused an effect or whether its result was verified.
Tracing therefore forms the correlation spine, not a dashboard added after the
runtime exists. A [tracing essay URL supplied as design context](https://substack.com/home/post/p-206409710)
is non-normative; no inaccessible source text is copied.

## Decision

Every job, run attempt, action proposal, policy decision, effect intent, tool
receipt, verification report, and run disposition carries a common trace identity
and its own span identity. The canonical material-effect sequence is:

`intent -> policy decision -> execution receipt -> verification -> run disposition`

An intent is durable before execution. A receipt records what the trusted
broker observed, not what a model claims. Verification targets the receipt's
immutable artifact. A run disposition references both receipt and verification.

## Public trace envelope

A public evidence record may contain:

- opaque trace and span references;
- tenant-safe project, Bead, job, attempt, and scenario identifiers;
- timestamps, component and version;
- action kind, outcome, policy rule identifiers, and data labels;
- hashes, byte counts, bounded error classes, and relative paths; and
- cost, latency, retry count, and quality metrics that cannot identify a
  person or private system.

It must not contain raw prompts, completions, private reasoning, context
documents, source fragments, tool arguments or output, secrets, provider
session state, private repository content, terminal recordings, or backend
addresses. Sensitive operational detail remains in access-controlled storage
under retention policy; Git receives only redacted summaries and opaque
references.

## Effect integrity

- Each intent has an idempotency key derived from stable public identifiers and
  a normalized action fingerprint.
- Every receipt hashes the bounded canonical result and links to its intent.
- Missing receipts are `unknown`, not success. Reconciliation queries the
  target before a retry.
- Trace sampling may drop advisory spans but never material effect intents,
  policy decisions, receipts, verification, or run dispositions.
- Clock order is not trusted across systems; parentage and monotonic sequence
  numbers establish causal order.
- OpenTelemetry is the initial transport contract and Tempo the planned local
  store, but evidence schemas are backend-neutral.

## Consequences

The trace schema becomes an early public contract and must be versioned.
Observability is useful for debugging and quality scoring without becoming an
authority source. A trace can explain a transition; it cannot grant one.
