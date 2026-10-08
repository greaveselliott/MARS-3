# ADR-008 - Finite-plan standing delivery delegation

**Status:** Proposed; not operational authority
**Date:** 2026-10-07
**Goal:** G-001
**Feature:** F-002

## Owner intent

The owner directed activation of standing delegation so routine non-production
delivery no longer requires repeated chat approval. Historical PD-004/ADR-007
bindings remain unchanged until a reviewed replacement is accepted.

## Admission foundation

A separately owner-signed canonical JSON contract lists a finite set of Beads,
their accepted contract digests and exact source paths, and binds tenant,
project, approved plan digest, accepted activation base and a maximum thirty-day
window. The distinct signature namespace is `mars3-standing-delivery-v1`.
Source-transition, profile and per-operation signatures cannot substitute.

The first slice validates this contract only. It creates no principal,
capability, claim, lease, runtime authorization or external effect. A caller
must obtain activation bindings from a trusted route rather than echoing input.
Unknown/duplicate fields fail canonical encoding; unknown/historical Beads,
empty/duplicate/unsafe paths, forged signatures, context drift and invalid
windows fail closed. No wildcard source authority is admitted in this slice.

## Remaining activation work

The first source candidate, PR #20, passed CI and QA but Security rejected a
publication-window read-swap defect. Its signed tag and verdicts remain
immutable rejected evidence. Corrected PR #21 passed ordered QA and Security
and merged as `880af5ddbc40d09ebe45af2cf0aec5b8a7286193`, retaining
tree `7849d50fedcd8fc8794f5c7617f46a9ed3ae77f8`. Protected-main CI and
readback passed. Its source grant is consumed, not reusable runtime authority.

The prospective `standing-delivery-runtime-source-v1` grant binds that accepted
base and eighteen exact paths on `codex/standing-delivery-runtime`. It permits
runtime implementation, synthetic qualification and reviewed source publication
only. It does not authorize canonical execution or private-resource discovery.

PD-005 specifies the proposed operational route. Operational delegation uses
a distinct kind and signature namespace; the existing v1 contract remains
non-operational. The parent binds an exact trusted runtime profile, finite
principal/profile/operation assignments and accepted contract/path scopes.
The issuer derives short-lived exact-request admission from that verified
parent, not from model-supplied authority or a fresh owner prompt per action.
Durable replay consumption precedes dispatch, including attempts that fail in
the backend. The dispatcher rechecks revocation, context and expiry and bounds
backend execution by the shorter parent/operation deadline.

The trusted issuer and gateway dispatcher must enforce current dependency
readiness, per-Bead canonical contracts/paths, role separation, fresh claims,
lease fencing, revocation and exact-operation durable replay. A signed contract
does not mean these checks have passed. Renewing a contract requires a new
owner signature; operation credentials cannot refresh the parent delegation.

Production/release, destructive effects, new private resources, spending,
scope expansion and trust escalation remain separate owner decisions. Existing
review and public evidence gates remain mandatory. D-001 is limited to
non-production dogfood, and O-001 to non-production qualification; membership
does not authorize a production deployment or release.

No live contract is installed by this source change. Independent review,
accepted publication and protected-main readback must precede runtime
activation. The source transition cannot access canonical/private state.

## Operational candidate protocol

The candidate operator accepts five protected file handles: runtime profile,
operational parent, activation, role session and typed request. Profile
verification and resource identity checks reuse the existing operator loader.
The parent, activation and session have distinct v2 signature namespaces.
Their canonical encoding prevents alias/unknown/duplicate field ambiguity.

The protected activation names the parent digest, accepted plan digest,
runtime binding, accepted contracts and current protected source base. It is
reloaded for each dispatch, so deleting or revoking it denies pending work.
Operator release identity and per-Bead source base are different bindings:
the accepted binary/profile may remain stable while reviewed source advances.
Updating the activation's source base is a trusted installer responsibility,
not authority derived from a request or an unverified branch name.

Role sessions bind one parent, principal, profile, role class and Bead.
Canonical profiles define independent review classes: `qa` is QA,
`security-reviewer` is Security and `delivery-orchestrator` is Orchestrator.
Class/profile disagreement denies before principal separation is evaluated.
One principal cannot span those canonical profiles by giving them the same
declared class. This corrects the independently reproduced PR #22 bypass;
that rejected candidate remains preserved and closed without merge.

Only the trusted operator may deliver sessions to independently authenticated workers.
The factory derives an in-memory exact-operation admission from that session,
the finite parent role assignment and canonical typed request. It does not
publish a new reusable capability or require an owner signature per request.
The root does not authorize itself to renew, expand or survive revocation.

Replay IDs span roles and operations within the same tenant/project/parent.
Consumption is durable before backend reads or effects. A fresh current-work
read enforces feature/path scope; the gateway remains responsible for actual
claim CAS, readiness, live fences and ordered lifecycle transitions. Claims,
leases and review verdicts are never synthesized by the issuer. Backend
execution is bounded by all applicable expiry windows. Protected profile or
activation renewal cannot extend an already-dispatched operation.

The current regression candidate exercises this issuer/dispatcher sequencing
using synthetic gateway adapters and the production durable file replay
adapter. Local qualification also passed signed namespace probes and a signed
read through the real production gateway; a real-gateway denial confirms that
delegation cannot invent missing claim/lease stores. Positive delegated
claim/lease, ordered lifecycle and same-owner correction integration now pass
through the real gateway with disposable, single-threaded synthetic stores.
This is not datastore durability or concurrency qualification. Independent
acceptance and installed protected activation remain outstanding. Existing gateway
conformance is not replaced by the synthetic dispatch fixture.

Handoff requires the action-bound handoff/release bundle. Its release
capability cannot escape that one dispatch. Explicit `work.reclaim` delegation
uses the existing control-plane gateway route only after verifying an
in-progress canonical claim owned by the authenticated profile with matching
original claim provenance. It restores operational fencing, not canonical
ownership. The normal claim route remains mandatory for backlog work, and
active lease overlap remains denied by the lease authority. Correction tests
retain the canonical claim and issue epoch 2 after QA requests changes.

Directory scope syntax follows canonical gateway paths: an explicit trailing
slash admits descendants, not wildcard expansion or sibling prefixes. The
same signed strings must still equal canonical paths. Traversal and Git
metadata descendants deny before effect validation.
# Absolute runtime expiry deadlines

The runtime constructs contexts with absolute signed deadlines at initial
admission and after activation reload. Converting a sampled remaining duration
into a new relative timeout could extend authorization across a scheduling gap.
Replacement activation can shorten, never extend, an issued context; an earlier
caller deadline remains binding. The pre-dispatch cancellation check remains
required. PR #23's rejected source candidate and Security disposition are retained
in runtime evidence; this correction does not activate the proposed delegation.
