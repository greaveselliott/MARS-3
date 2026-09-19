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
