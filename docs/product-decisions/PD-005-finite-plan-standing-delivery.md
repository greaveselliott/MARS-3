# PD-005 - Finite-plan standing delivery

**Status:** Proposed; not runtime authority
**Owner:** CEO (`strategy`)
**Date:** 2026-10-07
**Goal:** G-001
**Feature:** F-002
**Related decisions:** PD-002, PD-004
**Architecture:** ADR-008

## Owner direction and outcome

The owner directed activation of bounded standing delegation and continuation
through the existing active plan without repeated routine approval prompts.
Accepted pull requests must be merged and protected main read back before
delivery advances. Rejected candidates must be preserved and closed, not
merged to clear the queue.

Success means a trusted operator can admit an in-scope non-production request
under one finite owner delegation without obtaining another owner signature
for each request. This proposal does not claim that source publication alone
achieves that outcome.

## Decision proposed for implementation and review

Keep the existing v1 standing-delivery contract as non-operational data. Add a
distinct operational contract kind and signature namespace. A v1 signature,
source-transition signature, runtime-profile signature or individual-operation
signature must never substitute for the operational parent delegation.

The operational parent binds the repository, tenant/project, accepted plan
digest and activation base, exact protected runtime profile digest, finite
Bead contract/path scopes, finite principal/profile/operation assignments,
revocation identity and a maximum thirty-day validity window. A trusted
factory composition resolves these bindings; model output cannot nominate its
own identity, accepted contracts or protected resource handles.

Each invocation remains a typed proposal. The operator checks its exact
operation and request digest against the parent, assigns only the exact
capability bundle required by that action, and durably consumes a tenant/project-scoped operation
identity before dispatch. It must bound execution by the shorter parent and
operation expiry, and recheck context, expiry and revocation immediately
before dispatch. Uncertain replay persistence denies and never removes the
consumption record. Backend failure consumes the operation too.

Claims, dependency readiness, current canonical versions, declared paths,
handoff, independent QA then Security, run disposition, reconciliation and
terminal closure remain gateway-owned. Standing delegation cannot fabricate
an implementation claim, live lease, reviewer acceptance or successful merge.
Read requests cannot become a back door to mutation or other projects/Beads.

Handoff's action-bound bundle includes release of its exact presented lease;
it does not grant a separate lease-release operation. A parent may explicitly
delegate `work.reclaim` for same-owner correction after requested changes.
That factory control-plane action requires an existing in-progress canonical
claim, its original claim-attempt provenance and the authenticated owner's
profile, current scope/base and gateway fencing. It may restore only an
operational lease with a newer epoch; it cannot bootstrap backlog work,
transfer a claim or mutate canonical provenance. Its claim/lease-issue bundle
exists only during that exact dispatch, not as general administrative power.

Canonical directory scopes retain their explicit trailing slash and match
only descendants beneath that prefix. No wildcard or path normalization may
widen a signed scope; traversal, metadata paths and prefix siblings deny.

No embedded or separately supplied operation may expand the parent scope,
select a different runtime profile, renew its parent, add a capability or
silently recover a revoked/expired parent. Unknown operation classes deny.

## Boundaries retained

- Production and release effects require a separate owner decision.
- Destructive or irreversible effects require a separate owner decision.
- New private resources, secrets, credentials and spending are not delegated.
- Trust escalation, scope expansion and direct authority-store mutation are
  not delegated.
- D-001 membership permits only non-production dogfood; O-001 membership
  permits only non-production qualification.
- An unavailable protected runtime handle is a concrete setup blocker, not
  permission to discover credentials or bypass the gateway.
- The first normalized failure permits at most one equivalent automatic
  retry; recurrence records the required blocked disposition and escalates.

## Migration and acceptance

PD-004's historical one-Bead grants and their immutable evidence remain
unchanged. This decision supersedes no historical authorization and is not
operational until its implementation is independently accepted and the
separate protected activation bindings are installed through a trusted route.

Acceptance must prove positive in-scope dispatch and negative namespace,
identity, cross-project, contract/path drift, role, dependency, scope,
expiry/revocation and replay cases using synthetic fixtures. Existing gateway
claim/lease and lifecycle checks must remain exercised, not mocked away in
place of integration evidence. The same immutable candidate needs exact-head
CI, independent QA acceptance followed by Security acceptance, accepted merge
and protected-main readback.

Actual activation additionally requires a reproducible accepted operator,
valid protected runtime profile and resource handles, owner-signed operational
parent and a bounded gateway readback. The subsequent P-001 selection and
claim must use the normal authority gateway; neither this document nor a
source-transition grant selects or claims it.
