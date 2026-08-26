# ADR-003 — Rule-of-Two admission policy

**Status:** Accepted
**Date:** 2026-08-26
**Owners:** Security Engineer, Runtime Architect
**Goal:** G-001
**Feature:** F-001

**Implementation state:** Declared contract; hard admission policy is S-001.

## Context

Prompt injection becomes materially dangerous when one path combines all
three capabilities described by Simon Willison as the
[lethal trifecta](https://simonwillison.net/2025/Jun/16/the-lethal-trifecta/):
exposure to untrusted content, access to private data, and an ability to
communicate externally. MARS-3 must constrain the composition, not merely ask a
model to ignore malicious instructions.

## Decision

Every input and output carries data labels, and every tool declares capability
labels:

- `external-untrusted`: issues, pull requests, websites, images, dependencies,
  retrieved text, model output, and unconstrained test output;
- `private-data`: secrets, tenant data, provider sessions, private source, or
  non-public telemetry; and
- `external-effect`: network transmission, publication, messages, source
  control mutation, deployment, or any effect observable outside the sandbox.

No principal, context envelope, session, tool, broker route, or composed chain
may hold all three labels. This is the Rule of Two. Admission is computed over
the transitive route, not one call at a time.

## Taint and declassification

- Labels union as data flows through context, model, tools, files, summaries,
  and generated artifacts.
- Model output is always untrusted and cannot remove a label.
- Encoding, summarizing, translating, or hashing does not declassify content.
- Only a deterministic, separately authorized transformer may produce a
  narrower artifact, and its contract must prove that the sensitive value is
  absent rather than asking a model to judge it.
- Public repository source is `public+project-accepted` only after review.
  External contributions remain `external-untrusted` until accepted.
- Credentials are capabilities, never context. A credential proxy performs a
  pre-authorized request without revealing the credential to the model or
  sandbox.

## Admission examples

| Route | Result |
| --- | --- |
| public accepted source + external publication | allowed with policy and lease |
| external-untrusted issue + read-only public repository | allowed in isolated advisory mode |
| private repository + bounded internal analysis without network egress | allowed |
| external-untrusted web content + private source + network or Git write | denied |
| external-untrusted pull request + credential-bearing environment + message tool | denied |

Splitting the final route across two agents, tools, retries, or side chats does
not evade the rule; taint and capability history follow the job lineage.

## Policy rejection contract

A denial returns:

1. current state and labels;
2. the prohibited three-way composition;
3. the required transition, such as removing egress or private access; and
4. the exact safe action available next.

It never echoes the secret or untrusted payload.

## Consequences

Some useful workflows require compartmentalized handoffs, deterministic
redaction, or human mediation. Reduced convenience is accepted. Public source
classification never makes credentials, tenant state, provider sessions, or
local telemetry public.
