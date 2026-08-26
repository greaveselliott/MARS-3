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
rejected.

The entire H-001 workflow uses a deliberately small, fail-closed YAML grammar.
It accepts one literal top-level `on` block containing only `push` to `main`,
`pull_request`, and manual dispatch. Quoted or escaped keys, flow mappings,
tags, directives, anchors, aliases, merge keys, extra events, filters, and
dispatch inputs are rejected rather than interpreted. The only accepted GitHub
expressions are the two public concurrency identifiers already in the file;
secret contexts, token contexts, credential-bearing mappings, and any other
expression are denied.

The action allowlist contains exactly the pinned checkout and Go setup commits,
each once, with exact inputs. Checkout cannot select another repository or
retain credentials. Job containers, services, reusable workflows, local
actions, and Docker actions are prohibited. The only shell container commands
are the three exact, read-only, no-network Gitleaks invocations using the pinned
OCI digest; additional flags, mounts, images, or commands fail admission. A
later workflow that genuinely needs another event, expression, action,
permission, or container requires a dedicated product or architecture decision
and a narrower admission rule; it cannot weaken this foundation gate in place.

Structural diagnostics are not the final authority: YAML and shell each have
too many equivalent spellings to prove the whole job from selected fields. The
validator therefore normalizes only CRLF to LF and binds the complete sole
workflow to SHA-256
`b087a9bacc60f895aa00d58c34bd4b3791500762330addee84691ddc7dda2c62`.
Any other byte, job, step, runner, shell, condition, failure behavior, command,
comment, or workflow path fails the H-001 contract. Updating that digest is an
authority-bearing code change requiring a new ADR decision, immutable evidence,
and fresh QA then Security review; the workflow cannot bless itself.

H-001 pins Go 1.24.11 in CI and uses Gitleaks v8.18.4 by immutable OCI digest.
The scanner must first reject a synthetic canary, then scan both the worktree
and complete history. A newer release that fails that canary is not qualified
merely because its version number is higher.

## Consequences

Git hosting outages block publication but do not justify a local completion
claim. Reconciliation checks the remote before retrying. Production and
destructive migrations retain human approval in every delivery mode.
