# PD-001 — Public from the first commit

**Status:** Accepted
**Owner:** CEO (`strategy`)
**Date:** 2026-08-26
**Goals:** G-001
**Features:** F-001
**Supersedes:** None
**Superseded by:** None

## Decision

MARS-3 is developed in a public Apache-2.0 repository. Every committed source,
document, fixture, generated artifact, plan, and evidence record is treated as
immediately disclosed. No workflow may depend on later redaction.

Evidence is limited to relative paths, commands, versions, hashes, bounded
outcomes, and opaque trace references. Fixtures use reserved domains,
synthetic identities, and non-authenticating canary values. Secrets, real
identity or tenant data, private source, raw traces, raw model or tool payloads,
provider state, and machine-specific metadata are prohibited.

## Alternatives considered

- Develop privately and publish later. Rejected because delayed review creates
  a separate sanitization event and hides whether safety was present at origin.
- Allow sensitive local evidence in Git and redact on release. Rejected because
  Git history is a disclosure boundary, not temporary storage.

## Consequences

The public commit gate is mandatory before every commit and push. Secret
exposure triggers rotation, private incident coordination, history remediation,
and verification; a later deletion is insufficient. Security vulnerabilities
are coordinated through private vulnerability reporting before disclosure.
