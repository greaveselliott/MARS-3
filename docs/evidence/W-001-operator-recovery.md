# W-001 operator recovery: source-only preparation

## Authority and scope

The owner approved a fresh seven-day `W-001-operator-recovery-v1` grant,
issued 2026-09-19T14:03:20Z and expiring 2026-09-26T14:03:20Z. Its detached
signature verified under the pinned human key. The earlier expired document
remains preserved outside Git. The signed base is
`ee0ef97e1a3c246e342ef3f467c3b95947a327b5`, tree
`8fd175afaf40335d8c8148dc5407c1047ee60cbf`, on the bounded delivery branch
`codex/w-001-operator-recovery`.

This record does not assert a current canonical claim, live lease, accepted
candidate, or canonical run disposition. The exception supplies only its
explicit source/publication scope. Canonical execution needs a separate fresh
authorization and normal gateway policy admission.

## Initial implementation slice

- Added pinned-key signature verification for the separate operator-execution
  namespace, without exposing signature-parser diagnostics.
- Added canonical, bounded read authorization and exact-request/runtime binding.
- Limited admission to work inspection/readiness, read-only capability, and
  conservative labels; requests cannot construct principals.
- Required trusted durable one-attempt replay consumption before dispatch and
  rechecked expiry afterwards. Added a local exclusive-create replay adapter
  with private-directory admission and file/directory synchronization.
- Added synthetic tests for admission scope, expiry, replay, malformed input,
  forged/missing signatures, least privilege, and backend-error redaction.
- Added synthetic replay tests for reopen, concurrent independent handles,
  cancellation, unsafe paths, permission drift, and existing symlinks.
- Added a read-only CLI with a separately signed protected profile, executable
  and workspace binding, sealed local connection handle, and lazy gateway
  composition after authorization/replay admission. No canonical invocation
  has been made.
- Added separately authorized typed claim, renewal/heartbeat, release, and
  pre-effect validation routing through the unchanged normal gateway.
- Added exact signed-document, seven-day window, local branch/base/history,
  and sixteen-path source-grant admission. Added hosted CI event/runner binding,
  exact-head signed-tag chronology, synthetic PR merge topology, and protected
  main squash/tree/clean-checkout admission. No actual recovery PR or CI run
  exists yet.
- Kept deployment durability qualification, real canonical execution, and
  accepted publication pending.

## Evidence disposition

The initial admission suite passed with `go test ./internal/authority/operator
-count=1`, using an external writable build cache and offline module resolution.
The first invocation was denied access to the default build cache; the single
corrected invocation passed. The first replay-adapter run failed during fixture
admission: Go's temporary subdirectories inherit mode `0755` under umask `022`,
while the adapter requires private permissions. The four replay fixtures now
explicitly use `0700`; production admission was not weakened. The corrected
replay suite passed on its corrected rerun. The CLI/profile slice passed
`go test ./internal/authority/operator ./cmd/mars3-authority -count=1` with
offline resolution and the external cache. The mutation-routing slice also
passed the operator, CLI, and gateway packages. Synthetic positive admission tests
substitute a package-private signature verifier; they are not proof of an
accepted real operator authorization. The production constructor always uses
the pinned verifier. No canonical stores were accessed, no canonical state or
lease changed, and no P-001 work was claimed.

Nothing in this record substitutes for complete public gates, exact-head CI,
independent QA then Security acceptance, or protected-main readback. No commit,
push, or pull request has been made for this slice.

## Local publication-gate blocker

`BLOCKED` here describes a local Git publication gate, not canonical ticket
lifecycle or a gateway-recorded run. Normalized failure:
`publication/retained-git-history-unavailable`. Two gate observations have
occurred. The first reported missing postclaim history plus the old binding's
branch rejection. A single bounded recovery fetched missing immutable v4, v5,
and v6 postclaim tags without force or tag rewriting. The next gate advanced
through those checks and exposed additional missing delivery/lifecycle review
objects and retained feature commits. No historical hash or signature check was
weakened and no historical tag was rewritten.

Required owner decision before another history-repair attempt: authorize
retrieval of the complete missing public immutable tag history without
overwriting existing refs, or supply a complete trusted clone. Plan and DocSync
checks passed. Doctrine/public checks and the full doctrine suite remain failed;
their result is not an implementation acceptance. Source-grant unit tests and
the updated plan/DocSync checks passed. The additional CI topology, untrusted
runner, and strict chronology tests also passed in the focused recovery suite.
Public CI implementation is present,
but real CI and protected-main evidence remain absent. The owner decision on
complete missing-history retrieval remains outstanding; no further fetch has
been attempted.

## Independent local checks and scanner availability

The following additional checks passed with offline module resolution and an
external writable build cache:

```text
go test ./api/authority/... ./internal/authority/... ./cmd/mars3-authority -count=1
go vet ./...
git diff --check
```

The authority test invocation supplied the previously qualified native client
through `MARS3_TEST_BEADS_GATEWAY_BINARY`; the Beads package completed in about
21 seconds. These are fixture tests, not canonical execution or full repository
acceptance. The full doctrine suite remains blocked by retained-history gaps.

The native secret scanner is not on PATH. The unchanged CI workflow instead
pins the scanner container to
`sha256:75bdb2b2f4db213cde0b8295f13a88d6b333091bbfbf3012a4e083d00d31caba`.
The initial daemon query was denied by the sandbox. A single explicitly
approved read-only query outside that boundary reported no running Docker
daemon. No image was pulled, no container or daemon was started, and no canary,
worktree, or complete-history scan is claimed.

Publication is blocked pending explicit owner authorization for complete
missing-history retrieval without overwriting refs, plus availability of the
existing local scanner runtime (or explicit approval to start it). Source work
is not accepted; there is no recovery commit, push, PR, canonical read, claim,
or lease. The broader active plan has not advanced.

## Approved publication prerequisite recovery, 2026-09-19

The owner explicitly approved retrieving missing public historical tags without
replacing existing refs and starting the existing local container runtime for
the pinned scanner. An initial refspec construction failed before any ref update;
the corrected atomic, non-forced fetch added 34 missing public tags. Existing
refs were not replaced. The previously recorded retained-history blocker is
resolved for this checkout; its failure history remains above.

The installed container runtime started successfully. The existing pinned image
`docker.io/zricethezav/gitleaks@sha256:75bdb2b2f4db213cde0b8295f13a88d6b333091bbfbf3012a4e083d00d31caba`
ran with networking disabled, a read-only filesystem and repository mount, all
capabilities dropped, and no-new-privileges enabled. The synthetic canary
returned exit status 1 with one detected leak. The worktree scan and Git-history
scan both returned exit status 0 with no leaks; the history scan covered 151
commits. No scanner policy or historical evidence requirement was weakened.

Qualification outcomes after restoring history:

- `go run ./cmd/mars3 doctrine check --repo .`: PASS.
- `go run ./cmd/mars3 public-check --repo .`: PASS.
- `go test ./...`: PASS, including the complete doctrine suite and the native
  Beads fixture qualification using the previously recorded deterministic client.
- Pinned scanner canary: expected detection, PASS.
- Pinned scanner worktree and history checks: PASS.

These outcomes precede this evidence-only addition and do not constitute an
exact-head publication verdict. The source recovery remains unpublished and
pending signed candidate publication, exact-head CI, ordered independent QA and
Security review, accepted merge, and protected-main readback. No canonical
workspace read, claim, lease, lifecycle mutation, or active-plan advancement was
performed. This entry is local Git evidence, not a canonical gateway RUN record.

## Post-merge readback and missing-review exception, 2026-09-19

The owner reported merging PR #18 and explicitly authorized protected-main
verification and recording the review-gate exception. GitHub readback found
PR #18 merged at 2026-09-19T17:13:07Z as
`779ce8d585743fccf7abdc55631a1e9323516cfb`, the current protected main head.
Its sole parent is the grant base
`ee0ef97e1a3c246e342ef3f467c3b95947a327b5`. Its tree
`e63e7e58e35034110a9d895281b646553a3a4bfd` equals candidate
`a8153480a7927c2eb7d77ea0d0ba36d85573930e`. Retained review tag object
`4f6ce48322d000d17aabb38cf3473c07e87456e1` targets that candidate.

PR Foundation quality run/job `35457245192`/`105934609998` passed.
Protected-main push run/job `35457395834`/`105935006502` passed at the exact
merge commit, including doctrine, plan, documentation, public-content, test/vet,
whitespace, scanner-canary, worktree-scan, and history-scan steps.

At readback, the PR reviews and comments collections were empty. No ordered
independent QA acceptance followed by Security acceptance was recorded before
merge. Successful CI and tree equality do not satisfy or waive that signed
requirement. This is a missing-pre-merge-review exception, not an accepted
completion, retroactive authorization, or a claim that reviewers rejected the
candidate. A later review cannot change the historical ordering.

Local publication disposition: `blocked` pending independent QA then Security
review of the immutable merged tree and an explicit prospective recovery
resolution. This is not a canonical gateway RUN record. No canonical work,
lease, profile execution, or plan advancement occurred. Historical refs remain
unchanged. This evidence addition is uncommitted; a corrective Git publication
must preserve the original candidate and exception history.

## Independent post-merge QA verdict, 2026-09-19

A separate reviewer, not the implementing agent, returned `changes-requested`
for candidate `a8153480a7927c2eb7d77ea0d0ba36d85573930e`, tree
`e63e7e58e35034110a9d895281b646553a3a4bfd`.

P1: `internal/authority/operator/operator.go:773` sets PostgreSQL pool capacity
to two. The claim path retains a project-barrier transaction and a work-lock
transaction while acquiring a third transaction for saga lookup. The third
acquisition cannot succeed before cancellation, so an otherwise valid claim
can time out after consuming its one-attempt authorization. References:
`internal/authority/gateway/claim.go:205` and
`internal/authority/postgres/store.go:398` at the immutable candidate.

The reviewer reproduced the production store's Enter -> EnterWork -> Lookup
sequence with a bounded transaction-pool double in a disposable fixture.
Capacity two returned context deadline exceeded; capacity three completed
lookup. No database connection was made. This is a targeted reproduction, not
a successful end-to-end signed CLI run against disposable real backends.

Required correction: sufficient pool capacity for the nested transaction
sequence, with a composition regression tied to the launcher's configuration.
No correction has been applied. Focused operator, CLI admission, gateway
claim/lease, recovery-validator, signature/tag chronology, and whitespace
checks passed during review. Existing full-gate and CI results were supplied
context, not independently re-fetched by the reviewer.

Security review has not started because QA has not accepted this candidate.
This post-merge verdict does not cure the historical missing-pre-merge-review
exception or authorize canonical operations. This addition remains local,
uncommitted evidence; the matching PR comment records the public disposition.

## Owner-approved QA correction, 2026-09-19

The owner approved fixing the reported pool-capacity defect and adding
regression coverage. The local operator now configures three connections,
retaining both claim locks while a third transaction performs store work.
The new TestLocalPostgresConfigSupportsClaimTransactions exercises production
Enter -> EnterWork -> Lookup methods using a bounded pool double and the actual
launcher configuration. Its negative case specifies the prior two-connection
timeout; its positive case specifies successful lookup and complete capacity
release without releasing the locks prematurely. Linked feature and architecture
records describe the correction and the fixture's limits.

The correction is local and uncommitted. The new regression has not been run in
this correction turn. No passing-test or review-acceptance claim is made here.
The prior candidate, review tag, merge, and QA changes-requested verdict remain
unchanged. Fresh qualification and QA acceptance are required before Security
review; corrective publication must preserve the historical ordering exception.
No canonical access, claim, lease, or plan advancement occurred.

## Signed prospective v2 correction, 2026-10-07

The owner explicitly approved signing the prepared eleven-path grant and
continuing qualification and publication. W-001-operator-correction-v2 binds
base 779ce8d585743fccf7abdc55631a1e9323516cfb and tree
e63e7e58e35034110a9d895281b646553a3a4bfd. Its exact document SHA256 is
cf661e03a44968bc48e7564988f6987264f95b6f91d27890031c7054adb90267.
The pinned agent-backed Ed25519 signer produced a detached signature in namespace
mars3-w001-operator-correction-v2; signature verification passed. The window is
2026-10-07T20:31:36Z through 2026-10-14T20:31:36Z. The existing local correction
was retained on codex/w-001-operator-correction-v2 from that exact base.

Qualification before publication-validator changes found an older admission
assertion still expecting two connections. Correcting that assertion to three
made the operator and CLI suites pass with count=1, including the production-store
capacity regression. Focused vet and whitespace checks passed. These are focused
results, not a full publication or independent acceptance verdict.

V2 publication admission pins the new signed bytes and reuses publication
mechanics while retaining separate historical checks for V1 and the PR #16
lineage. The active plan and manifest identify the current correction and the
unresolved historical review exception. Full public gate, immutable candidate
CI, ordered independent review and accepted corrective publication remain
pending. No canonical store or lease operation occurred.

### V2 fixture restoration and qualification

The first complete-gate attempt passed doctrine, plan, docsync and public-content
checks, then the Beads integration test failed because the previously disposable
native executable was no longer present. The retained candidate binary's SHA256
still matched 7325651446eb58de6e80c11e00647286de04da0940e590debb90eaa8704eb0f0.
A new disposable executable copy was created without accessing canonical stores
or changing the client source. The Beads package integration suite then passed
with count=1. This restores the fixture prerequisite; it is not runtime authority
or independent acceptance of the native client.

V1 and V2 focused publication regressions passed with count=1. The pinned
scanner canary detected one synthetic leak with expected exit status 1; worktree
and history scans both returned zero with no findings. The history scan covered
153 commits at this point. A separate QA reviewer is inspecting the local
correction as a preliminary review; final acceptance must bind the immutable
signed candidate after public CI. The first complete suite's missing-fixture
failure remains recorded; the complete gate will be rerun with the restored
fixture before any corrective commit or push.

### V2 local gate and preliminary independent QA

The corrected complete public gate passed with the restored native fixture:
doctrine, plan, docsync, public-content checks, go test ./..., go vet ./...,
whitespace, and pinned worktree/history secret scans. Some unchanged Go package
results were cached; the restored native integration package ran successfully.
No canonical database was used.

A separate QA reviewer previewed the correction and found no blocking issue.
The reviewer confirmed production-store capacity requirements, regression
coverage of starvation and lock retention, preservation of V1 publication
checks, and the retained historical review exception. The preview used static
inspection and read-only Git checks, not independent runtime qualification.
It is not final acceptance of an immutable published candidate. Final QA and
Security verdicts must bind the same signed candidate after public CI.
