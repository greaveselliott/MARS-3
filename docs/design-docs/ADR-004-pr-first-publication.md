# ADR-004 — Pull-request-first publication

**Status:** Accepted
**Date:** 2026-08-26
**Owner:** Integration Engineer
**Goal:** G-001
**Feature:** F-001

## Context

The inherited doctrine values prompt publication of validated commits and
forbids stranded local-only state. A public multi-tenant factory also needs
review, status checks, provenance, and an explicit publication boundary.

## Decision

The hosted default is a branch named `codex/<ticket-id>-<slug>` followed by a
pull request through the trusted Git integrator. Trusted repositories may opt
into direct-trunk publication only through project policy. Both modes require:

- a current claim and, after W-001, a capability lease; H-001 uses only the
  signed human bootstrap exception;
- the complete public commit gate;
- an effect intent before publication and receipt after it;
- immutable commit and check identifiers;
- required review and approval;
- prompt publication or an explicit blocked/preempted state; and
- no stranded local-only completion claim.

Fork pull requests receive neither repository secrets nor a write token.
`pull_request_target` is prohibited. Workflow permissions default to read-only,
and third-party actions are pinned to immutable commit SHAs.

For the H-001 foundation workflow, read-only is a fail-closed syntax contract,
not an inferred default. The workflow must contain exactly one column-zero,
multiline `permissions` mapping whose sole entry is `contents: read`.
Job-level permission declarations, inline mappings, duplicate declarations,
anchors or aliases, explicit YAML mapping keys, and additional scopes are
rejected. A later workflow that genuinely needs another permission requires a
dedicated product or architecture decision and a narrower admission rule; it
cannot weaken this foundation gate in place.

H-001 pins Go 1.24.11 in CI and uses Gitleaks v8.18.4 by immutable OCI digest.
The scanner must first reject a synthetic canary, then scan both the worktree
and complete history. A newer release that fails that canary is not qualified
merely because its version number is higher.

## Consequences

Git hosting outages block publication but do not justify a local completion
claim. Reconciliation checks the remote before retrying. Production and
destructive migrations retain human approval in every delivery mode.
