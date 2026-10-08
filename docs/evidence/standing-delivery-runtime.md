# Standing delivery runtime candidate evidence

**Status:** V1/V2/V3 rejected; approved test-only recovery locally qualified, independent acceptance pending
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

PR #22's immutable v1 candidate is rejected, not accepted source:
head `031a792970452fb859fee7dce65e7bc37d133420`, tree
`d49c8a4900cee6e91b9039b29cab44943d08f15b`, tag object
`6a91a2d0600e2634aa53cf932d3b7ed3f28c5b89`. Exact-head run/job
`37706308491` / `113081395802` passed. Independent QA then reproduced
P1 `standing_delivery.role_class_canonical_profile_mismatch`: one principal
could hold QA and Security canonical profiles with both declared class QA,
pass both production-gateway reviews and close the synthetic Bead.

The [public QA disposition](https://github.com/greaveselliott/MARS-3/pull/22#issuecomment-6049474538)
records the finding. PR #22 closed without merge before a successor opened;
its head, signed tag and CI remain immutable. Security did not run and no
activation is accepted.

The V2 correction binds canonical profile/class pairs before principal
separation. QA, Security and Orchestrator profiles cannot masquerade as another
class. A negative production-gateway regression rejects the reproduced parent
before replay consumption, gateway events, claim CAS or lease issuance.
Distinct correctly declared QA/Security classes also reject a shared principal.
The historical V1 qualification below does not accept its rejected tree.
V2 focused tests and the complete local public gate passed, including the
124.682-second complete doctrine suite and scanner canary/worktree/history
checks. Two corrected builds were byte-identical, SHA-256
`2a8aa075d53b22364ae233c0c1c49aefa347abbdd8519b2369eec95337459ad7`.
A separate synthetic probe verified real owner signatures on the malformed
parent and its session, then confirmed denial before activation lookup,
replay consumption or gateway access. This is policy rejection of an
authenticated invalid assignment, not a forgery test.

V2 still requires a distinct signed tag, exact-head CI, fresh QA then
Security, accepted merge and main readback inside the original signed scope.

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

## V2 independent review and V3 correction

PR #23 retains head `7dbe1b718e585d43458a3e91f622481490ef70f4`, tree
`943a97f4787833fb59c7456400e200ba26764207` and signed v2 tag object
`099d8dbe63e46b0c479f3d4475126b1c2fa46612`. Exact-head CI run/job
`37708418675` / `113088251542` passed. Independent QA accepted the unchanged
candidate with 24 bounded regressions and independently verified CI:
[QA disposition](https://github.com/greaveselliott/MARS-3/pull/23#issuecomment-6049717876).
Subsequent independent Security requested changes:
[Security rejection and closure disposition](https://github.com/greaveselliott/MARS-3/pull/23#issuecomment-6049795320).
PR #23 is closed without merge; its immutable objects and ordered verdicts
remain rejected evidence.

Foundation-owned `standing_delivery.runtime_absolute_deadline_construction`
(public disposition alias `standing_delivery.activation_relative_deadline_gap`)
is a new runtime deadline-construction finding, not the previously corrected
source-publication unsigned-read-swap finding. A genuine wall-clock sample
followed by a scheduling delay before relative timeout creation extended the
effective deadline beyond a shortened activation expiry. A synthetic execution
dispatched once and returned success after expiry. No canonical resources or
signature bypass were involved.

The single bounded correction uses absolute deadlines for initial admission
and replacement activation, preserves earlier caller deadlines and retains
the pre-dispatch cancellation check. New regressions model scheduling delay
after a genuine clock sample at initial admission and after shortened activation
reload; neither may dispatch after expiry. Focused regressions and the full
local doctrine, plan, documentation-sync, public-content, Go test, vet and
whitespace gates passed; the doctrine suite took 107.363 seconds. Two builds
were byte-identical with SHA-256
`9b3ce67710ffa036f645d5964bc93e0839d6c8f63a4ae2837975a3d413bc4763`.
These bytes are not independently accepted. A distinct v3 attestation, final public
gate, exact-head CI, fresh QA then Security, accepted merge and protected-main
readback remain required. Equivalent recurrence must stop automatic correction.
No canonical RUN disposition or live authority is asserted.

## V3 review and bounded source-run stop

PR #24 retains immutable head `07c3930b1c074c8dbce5b943a8363e9a645859a4`,
tree `9545c0dcbc98aeca0a202dd0efb54205a020ecfe` and signed v3 tag object
`a8a3cb9fcffab0a6ec7eb5246d4d658dbcad3013`. Final local public qualification
passed, including the 107.547-second doctrine suite, expected scanner canary,
clean worktree and all 161 commits. Exact-head CI run/job
`37709892361` / `113093085222` passed and QA independently verified it.

Independent QA requested changes with foundation-owned
`standing_delivery.absolute_deadline_regression_not_sensitive`. All 26 bounded
candidate tests passed, but both scheduling-gap cases also passed with rejected
V2 implementation substituted in a disposable fixture. The shortened case
delayed validation rather than the vulnerable relative-timeout construction
sample; a later fresh sample therefore denied without exposing the bug.
An independent synthetic probe reproduced a dispatch context deadline about
one second beyond signed activation expiry on V2 and passed on V3. No remaining
implementation defect was reproduced, but the candidate's regression coverage
does not protect the correction. Security was not run after QA rejection.

PR #24 is closed without merge, with public QA rejection and stop disposition.
The source-qualification run is `blocked`: the single automatic runtime-deadline
correction did not achieve independent acceptance. No further automatic patch
or successor PR is scheduled at this boundary. Required deliberate next action
is a test-design disposition for a construction-gap/deadline-bound regression
that demonstrably fails on V2 and passes on V3, followed by fresh qualification
and ordered review under applicable authority. This is Git-owned source evidence,
not a gateway or canonical Bead RUN record. All rejected objects remain retained;
no accepted source merge, live authority or plan advancement is asserted.

## Runtime activation remains outstanding

No protected operational parent, activation or authenticated role session has
been installed. Existing protected resource handles must be supplied through
the trusted operator route; source work cannot discover or recreate them.
Actual activation requires an accepted reproducible operator and bounded
gateway readback before ordinary selection/claim of P-001 can proceed.

Synthetic dispatcher fixtures prove admission sequencing only. They do not
prove worker isolation, independence of reviewers, live gateway conformance,
private-resource provisioning or an active plan-wide delegation.

## Owner-approved prospective test recovery

The owner approved the recorded test-only recovery. The separately signed
`standing-delivery-runtime-test-recovery-v1` binds rejected V3 head
`07c3930b1c074c8dbce5b943a8363e9a645859a4` and tree
`9545c0dcbc98aeca0a202dd0efb54205a020ecfe`, nine exact paths and the window
2026-10-08T05:31:22Z inclusive to 2026-10-14T23:32:36Z exclusive. Its SHA-256 is
`bf4a29d4b1bbf4e206f5f0df6e8826654af218306fd73a6e7955be6ea420b3ec`.
This is a prospective exception, not a retry reset, runtime change or acceptance
of any rejected candidate. The prior blocked source-run record remains history.

The recovery adds signed-expiry deadline bounds at initial replay admission
and replacement dispatch. The replacement case introduces the scheduling gap
at the rejected relative-timeout construction sample, rather than at earlier
validation. Runtime source is unchanged. Qualification and immutable V4
publication results must be recorded before fresh independent acceptance.

Recovery local qualification passed. Both new deadline-bound tests failed
against immutable rejected V2 with the expected initial-admission and dispatch
expiry-bound messages, and passed against the corrected candidate. This negative
control used only a disposable public source tree and synthetic gateway fixtures.
The complete doctrine suite took 118.763 seconds; all public validators, full Go
tests, vet and whitespace checks passed. Runtime source diff against V3 was empty.
Two builds were byte-identical, SHA-256
`bdccda7a9bd6b10b37c9084fcd52a6e32ac6bfcfadc326cdd90dadd5fe6cde45`.
The pinned scanner detected its one synthetic canary and found no worktree leaks.
These are local qualification results, not independent acceptance or activation.
