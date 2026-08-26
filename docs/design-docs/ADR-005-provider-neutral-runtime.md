# ADR-005 — Provider-neutral runtime boundary

**Status:** Accepted
**Date:** 2026-08-26
**Owners:** Runtime Architect, Security Engineer
**Goal:** G-001
**Decision:** PD-003
**Feature:** F-001

## Context

Hosted agent SDKs and open-source model transports are different abstractions.
Conflating them would either discard hosted lifecycle semantics or grant a raw
model responsibilities that belong to the factory harness.

## Decision

Full agent loops such as Codex app-server/SDK and Claude Agent SDK implement:

```text
AgentRuntimeAdapter {
  start, resume, steer, cancel, events, finalize, capabilities
}
```

Open-source models implement `ModelTransport` beneath a project-owned
`NativeHarnessRuntime`. That harness—not the model transport—owns context
building, typed action proposals, policy, tool brokering, verification,
feedback, and convergence. Colibri is experimental advisory/read-only until it
passes typed-action conformance. Kimi/model artifacts and model-serving
runtimes are separately versioned and qualified contracts.

One run attempt pins one adapter, adapter version, and model identity. Changing
any of them starts a new attempt. Opaque provider state is encrypted in
approved operational storage when needed and never migrates across adapters.
All paths preserve cancellation, trace correlation, bounded metadata, and
capability discovery without self-granting authority.

The deterministic orchestrator owns scheduling. Beads/Dolt owns work state;
the gateway/PostgreSQL owns live leases; Temporal only executes. No adapter or
transport claims work, edits Git directly, injects credentials, or marks
verification complete.

## Secret boundary

Hosted adapters use customer-managed provider API credentials, never consumer
subscription cookies or session tokens. Raw credentials remain in a platform
secret service (or an Electron-independent local credential daemon) and are
presented only through a private proxy using a single-use gateway token bound
to the tenant, attempt, provider, model, budget, and expiry. Raw provider
credentials cannot enter an attempt pod, context, command argument,
environment snapshot, terminal output, trace, screenshot, fixture, Temporal
history, Beads record, Git artifact, or evidence. Revocation and rotation do
not require repository changes or an Electron process to remain open.

## Adapter containment

[Codex app-server](https://developers.openai.com/codex/app-server/) is the
planned interactive adapter because its versioned JSON-RPC lifecycle, streamed
events, steering, interruption, and approval requests fit the operator
timeline. MARS-3 uses the local stdio or Unix-socket transport and a generated
schema pinned to the qualified Codex build; experimental WebSocket surfaces
are not a production dependency. The
[Codex SDK](https://developers.openai.com/codex/sdk/) may serve bounded
headless jobs that start, continue, or resume local Codex threads. Claude Agent
SDK is also an agent-loop adapter.

Each hosted adapter runs inside an isolated attempt sandbox with an isolated
home/config directory, no authority credentials, default-deny egress, and only
factory-brokered publication. Provider-native projects, goals, todos, memory,
plugins, tools, shell commands, and subagents are attempt-local behavior, not
Beads authority or a policy decision. Unsupported or unsafe RPCs are never
proxied to the UI.

## Qualification and service levels

Qualification is evidence for an exact adapter, version, model, serving
runtime, and hardware class. It covers p95 queue delay, time-to-first-token,
tokens per second, cancellation latency, crash/replay behavior, tool-call
validity, and tenant fairness under long prompts. Admission control is derived
from those results; no fixed one-generation-per-replica rule is doctrine.
Capacity pressure may queue, cancel, or degrade to advisory, but never weaken
policy or isolation.

## Consequences

Lowest-common-denominator behavior is avoided: adapters may advertise richer
features, but policy maps them into the same auditable action surface. Provider
parity is measured by conformance evidence, not matching prose output.
