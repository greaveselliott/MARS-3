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
