# PD-003 — Provider-neutral governed runtime

**Status:** Accepted
**Owner:** CEO (`strategy`)
**Date:** 2026-08-26
**Goals:** G-001
**Features:** F-001, F-003
**Supersedes:** None
**Superseded by:** None

## Decision

MARS-3 separates full agent-loop runtimes from model transports. OpenAI Codex
and Anthropic Claude implement `AgentRuntimeAdapter`, whose lifecycle is
`start`, `resume`, `steer`, `cancel`, `events`, `finalize`, and
`capabilities`. Open models use `ModelTransport` beneath a project-owned
`NativeHarnessRuntime`, which owns context, typed actions, policy, tools,
verification, and convergence.

One run attempt pins one adapter, adapter version, and model. Switching an
adapter or model starts a new attempt; opaque provider state never migrates.
Colibri remains experimental and read-only until typed-action conformance.
Kimi/model artifacts and serving runtimes have separate qualification
contracts. Credentials are injected by a broker and never enter prompts,
repositories, telemetry, or evidence.

## Alternatives considered

- Couple orchestration to one provider SDK. Rejected because provider-specific
  authority and session semantics would leak into product policy.
- Normalize only text generation. Rejected because tool calls, resumability,
  cancellation, tracing, and secret handling are the consequential boundary.

## Consequences

Qualification precedes mutation authority and measures p95 queue delay,
time-to-first-token, tokens per second, cancellation latency, crash/replay,
tool-call validity, and fairness under long prompts. No fixed generation count
is doctrine; admission limits follow measured capability and SLO evidence.
