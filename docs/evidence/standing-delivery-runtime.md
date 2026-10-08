# Standing delivery runtime candidate evidence

**Status:** Local and mutation-integration qualification passed; publication and acceptance pending
**Goal:** G-001
**Feature:** F-002
**Decision:** PD-005 (proposed)
**Architecture:** ADR-008 (proposed)

## Authority and immutable base

The separately signed `standing-delivery-runtime-source-v1` grant binds
eighteen exact paths, branch `codex/standing-delivery-runtime`, accepted main
`880af5ddbc40d09ebe45af2cf0aec5b8a7286193` and tree
`7849d50fedcd8fc8794f5c7617f46a9ed3ae77f8`. It permits source implementation,
synthetic qualification and reviewed publication only. It does not permit
canonical execution, protected resource discovery or new private resources.

PR #21's accepted foundation remains immutable. Its signed v2 review tag
object is `12d832740cae5a32331b24089e10528d417ed4af`; the rejected PR #20
v1 object and earlier reviewed lineage remain retained, not retroaccepted.

## Implementation candidate

- Distinct operational parent, activation and authenticated role-session kinds
  and signature namespaces; v1 remains non-operational data.
- Finite Bead, contract, feature, path and identity/role/operation admission.
- Short-lived typed requests without a per-request owner signature.
- Durable replay consumption before reads/effects; no retry after backend or
  uncertain persistence failure with the same operation identity.
- Activation reload and revocation/expiry checks before dispatch.
- Separate accepted operator identity and current protected source base.
- Typed gateway claim, lease, handoff, review, run, reconciliation and closure
  dispatch; guarded same-owner operational-lease recovery, no direct-store or
  unfenced bootstrap route.
- Existing protected runtime loader and replay adapter, with bounded execution
  contexts and public-safe error classes.

## Qualification state

Local qualification on 2026-10-07 passed:

- Focused operator, doctrine and command-entry regressions via
  `go test ./internal/authority/operator ./internal/doctrine ./cmd/mars3-authority -run 'TestStanding|TestDelegatedOperator' -count=1`.
- The real production gateway and lazy operator adapter returned an audited
  canonical synthetic read and denied claiming without configured claim/lease
  stores. Delegation did not fabricate the missing substrate.
- Three owner-key-signed synthetic objects passed their respective operational
  signature verifiers. Six cross-namespace substitutions and three v1
  substitutions denied. A signed parent/session request reached the real
  gateway, and the production file replay adapter denied its reuse.
- Doctrine, plan, documentation-sync and public-policy checks passed.
- `go test ./...`, `go vet ./...` and `git diff --check` passed. The complete
  doctrine package took 124.205 seconds; no pending test process remains.
- The pinned v8.18.4 scanner detected the one synthetic canary as expected,
  found no worktree leaks and found no leaks across 158 retained commits.
  Scanner containers had no network, read-only mounts, dropped capabilities
  and no-new-privileges enabled.

Qualification corrected three distinct foundation-owned candidate defects:
the dispatcher treated the existing capability helper's boolean as an error;
the replay fixture omitted the required private directory mode; and the real
gateway fixture omitted its required display identifier. Each corrected case
passed its single equivalent retry. No canonical RUN disposition is asserted
by these Git-owned source-qualification records.

Extended integration qualification also passed:

- A delegated initial claim issued exactly epoch 1 after canonical CAS.
- Current effect fencing succeeded and a stale epoch denied.
- Renewal retained the lease's epoch; handoff released it before review.
- Security-before-QA and premature terminal closure denied without lifecycle
  mutation. QA then Security acceptance, completed run, reconciliation and
  closure passed with six ordered lifecycle transitions.
- Stale-version and dependency-blocked claims issued no lease and made no CAS.
- Canonical directory scopes admitted safe descendants and denied traversal,
  sibling prefixes, malformed paths and Git metadata descendants.
- Handoff's exact gateway-required release companion stayed action-bound;
  handoff permission did not authorize an independent lease-release request.
- Explicit same-owner `work.reclaim` rejected backlog bootstrap, active lease
  replacement, wrong provenance and ownership transfer. After QA requested
  changes it issued epoch 2 while leaving the existing canonical claim and
  its original attempt unchanged.

The first integrated handoff exposed one additional foundation-owned defect:
the dispatcher omitted the gateway-required lease-release companion capability.
The bounded action bundle corrected it and its single equivalent retry passed.
No general lease or administrative capability is returned to the caller.

These are local source results, not independent acceptance. The signed probe
used synthetic resources only; no protected operational resource was loaded
and no canonical work changed. The extended integration uses production
gateway policy with single-threaded in-memory synthetic stores; it does not
qualify persistence, concurrency or real private resources. No signed
candidate, CI result or runtime activation
is claimed, and no delivery PR has opened yet.

The next evidence must rerun the publication gate for the final candidate,
bind an immutable candidate/tag and
exact-head CI, obtain QA then Security acceptance, merge and read back
protected main.

## Runtime activation remains outstanding

No protected operational parent, activation or authenticated role session has
been installed. Existing protected resource handles must be supplied through
the trusted operator route; source work cannot discover or recreate them.
Actual activation requires an accepted reproducible operator and bounded
gateway readback before ordinary selection/claim of P-001 can proceed.

Synthetic dispatcher fixtures prove admission sequencing only. They do not
prove worker isolation, independence of reviewers, live gateway conformance,
private-resource provisioning or an active plan-wide delegation.
