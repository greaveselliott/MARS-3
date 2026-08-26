# W-001 bootstrap claim transition evidence

**Status:** Successor v6 public-CI stabilization pending; no canonical claim has executed
**Classification:** PUBLIC
**Goal:** G-001
**Decision:** PD-002
**Feature/scenarios:** F-002 / F-002-S1, F-002-S2
**Bead:** M3-W001 (display ID W-001)
**Failure ownership:** foundation
**Attempt:** `w001-bootstrap-3135f1d1-b0d4-4956-9fc9-1852310bfd77`
**Grant:** `.harness/grants/W-001-bootstrap.yaml`
**Blocker record:** `01a03d98-a2e2-77db-bbd6-76575366454f`

## Authority boundary

The human-directed grant authorizes publication of the exact reviewed helper
tree, disposable conformance, one expected-preimage atomic claim of M3-W001,
and one bounded claim receipt. It grants no live lease or implementation
capability. It cannot mutate P-001 or another Bead, reconcile the plan or
manifest, start gateway implementation, use raw SQL, or execute a two-step
claim. Publication authority alone cannot execute the claim: `--apply`
requires a separate, externally supplied, short-lived signed execution
authorization binding the actual protected-main commit and check plus QA and
Security acceptance of the immutable reviewed feature commit. That
authorization also binds the opaque canonical-workspace instance digest
emitted by the dry run. The helper resolves the fixed public GitHub HTTPS
`main` with caller Git, SSH, proxy, certificate, and user configuration
removed; mutable remotes and remote-tracking refs are not authority. It rejects
copied workspaces, ambient Beads/Dolt workspace overrides, and any execution
authorization whose freshly reloaded fields differ after conformance. It
rechecks both expiries, the direct workspace filesystem identity, and the
complete preimage at the effect boundary. Postclaim Git reconciliation
requires another signed grant.

## Preimage

The read-only canonical check observed native status `open`, lifecycle
`backlog`, intended assignee `work-authority-engineer`, dependency M3-H001 in
`closed/done`, and no claim or lease. The grant binds:

- accepted base commit `37b55b912b20715349bc50e0524c85d4b22f1772`;
- authority project `e9669a62-5be6-4b94-85f8-c502c29d394a`;
- metadata SHA-256
  `10c61003cb39518f57905620fcc0c47d29950fe82ae8d98a3111a057fa554dba`;
- sorted-label SHA-256
  `be506df06d8c206a3919a71a57e8aaacd2b5e1e233e25bafc2f5f87f306b188c`;
- dependency SHA-256
  `3ad0bca78b14e4e1fd5544477f131c0a86dd8a4d4e9563d43fa4ae1c202f4100`;
- lineage SHA-256
  `9f3e91b4b642dc740898347c35e8f38abc35cc3ac1be83c81fe122cc308eaced`;
- pinned Beads v1.2.2 source revision
  `6c124203e771433a3550c348771a5b5e27fd3c21` and binary SHA-256
  `6cc5cf1d3fea5774606af82410ac05e35b78ad5f404f1da5be33c93ff087cffb`.

The issue incarnation is the SHA-256 of canonical JSON containing only the
stable Bead ID and creation timestamp. Dependency and lineage digests are
integrity bindings; the non-reusable incarnation and monotonic mutation/graph
sequences are the freshness boundary.

## Atomicity correction

Pinned Beads `update --claim --metadata` performs two SQL transactions, while
unmodified `batch` provides no compare-and-swap claim operation. Using either
would expose a partial canonical state. The public patch adds one bounded
`bootstrap-claim` batch operation that uses Beads' native `ClaimIssueInTx`
primitive and applies metadata plus lifecycle-label changes inside the same
outer Dolt transaction. Any stale status, owner, timestamps, metadata, labels,
or dependency precondition rolls the entire transaction back.

The helper verifies that a successful embedded-workspace claim advances the
Dolt version commit. A command error is an unknown-acceptance blocker: it is
never converted into idempotent success, and no retry is allowed until the
issue, working set, and history are reconciled under new authority.

## Required verification before claim

The immutable reviewed commit must pass:

```text
go run ./cmd/mars3 doctrine check --repo .
go run ./cmd/mars3 plan check --repo .
go run ./cmd/mars3 docsync audit --repo .
go run ./cmd/mars3 public-check --repo .
go test ./...
go vet ./...
git diff --check
gitleaks git --redact --no-banner
```

Disposable conformance must exercise successful transition, stale-precondition
rollback, rollback after a later operation fails, and concurrent one-winner
contention. The built patched binary must then claim a disposable copy of the
canonical workspace through its actual `embedded` backend, publish a new Dolt
version, and leave the canonical workspace unchanged. QA and Security must
accept the same tagged immutable tree, the
PR must be squash-merged with exact tree equality, and protected-main CI must
pass. Only then may the human authority sign a one-hour execution authorization
binding those exact facts and the opaque canonical-workspace instance digest.
The helper reloads the exact authorization after conformance, resolves only the
fixed public GitHub `main` under its sanitized environment, and revalidates the
grant expiry, workspace instance, and preimage immediately before the sole
effect.

## Successor v2 disposable conformance

The signed implementation checkpoint was
`d3159dbb84bc1ad5da35c2a63d0e6217cf334ac2`, with tree
`c9b168d7a93fa8f9800c42de9b043ea05823217f`. From that clean checkpoint,
the dry-run helper rebuilt the pinned patched client and returned
`conformance-passed-no-canonical-mutation` with these public-safe bindings:

- patched binary SHA-256
  `8ba8ba8ba97e83582bc552af367e053106a39c20a4c8e0fe8a2580cd17d70475`;
- patch SHA-256
  `dadeafcc0c3fc1b27752129f8904e1aa6e828a9a14624ffa6750064b6a0afd3b`;
- helper command SHA-256
  `d8ae9fcf5b04902fa3f2ece3369688ca7abf1e55f0cd4f57a611006a861979ea`;
- helper library SHA-256
  `23c0c6bbfa28525f5f2612173970a8f77ab9d7986805ec9cd8045c02fe9da94a`;
- disposable backend `embedded`, disposable postimage verified, canonical
  unchanged before any effect, native status `open`, lifecycle `backlog`, and
  `liveLeaseAsserted: false`.

All four executable Docker/Dolt conformance cases passed:

- `TestBatchBootstrapClaimIsOneAtomicTransition`;
- `TestBatchBootstrapClaimPreconditionFailureRollsBack`;
- `TestBatchBootstrapClaimPostClaimFailureRollsBack`;
- `TestBatchBootstrapClaimContentionHasOneWinner`.

The patched CLI then executed the exact claim operation against a disposable
copy of the canonical embedded workspace. The copy matched the signed
preimage, reached the signed postimage, and advanced its Dolt version commit.
The original workspace was read again and matched its pre-run issue digest.

An additional read-only post-run check normalized only ID, status, assignee,
timestamps, sorted labels, lifecycle, and dependency tuples. The W-001 summary
SHA-256 was
`8cc4cb33c443d417c7a88fb0cadda4942f5331f44d1fdf4c3072deb75de7d6b7`:
W-001 remained `open/backlog`, assigned to `work-authority-engineer`, with its
sole H-001 dependency `closed/done`. The P-001 summary SHA-256 was
`c1d04bf54be0c2332e4b12b2eb843e3a695e77f82ce33e8a7fae9987272e6be6`:
P-001 remained `open/backlog`, assigned to `platform-engineer`, with H-001
`closed/done` and W-001 `open/backlog`. Neither Bead was claimed and no lease
was issued.

Three bounded disposable-only corrections preceded the accepted run. A
CGO-disabled client could not open the copied embedded workspace; the first
CGO-enabled client exposed that the transaction decorator did not forward the
native compare-and-swap primitive; and a later successful disposable claim
exposed a one-nibble error in the signed post-label digest. Each attempt failed
closed before canonical mutation. The final grant uses the digest derived from
the sorted label transition, and a deterministic regression now binds that
derivation.

## Preserved v2 publication finding

Signed tag `mars3/w001-bootstrap-helper-v2` remains immutably attached to
commit `c95f159cadef6ccc6fcaf2dccfad5c7457e60c3a`, tree
`bf13f5c9de0651b5e3edfa8a1723ca4789a3c336`. GitHub run `32971431988`
failed at `public.w001_bootstrap_event`: the validator incorrectly required
GitHub's advisory pull-request merge field to equal the exact checked-out
synthetic merge. The runner had current branch and base identities, but this
optional payload field can be absent, null, current, or a stale lowercase
40-hex value.

The successor accepts only those well-formed advisory representations. It
still binds the event's exact feature head and signed base, requires the
checkout to have those exact two parents, requires the checked-out tree to
equal the feature tree, and requires the immutable signed review tag to target
that feature head. Malformed advisory values and every head, parent, tree, or
tag mismatch continue to fail closed. The v2 tag is not moved or reused; that
correction used `mars3/w001-bootstrap-helper-v3`, preserved below.

## Preserved v3 tag-envelope finding

Signed tag `mars3/w001-bootstrap-helper-v3` remains immutably attached to
commit `a97b6f90ec5f3b554c00bf6efa723fcef1d60602`, tree
`2edf3fe18e0183fc0a15e403d82a35d4216a0e23`. GitHub run `32971964972`
confirmed the stale merge field correction and then failed at
`public.w001_bootstrap_tag`: the tag used the synthetic Engineer identity,
while the public tag contract requires
`MARS-3 Release Manager <release-manager@example.com>`.

The tag signature, target, and message are preserved, but v3 is not an
accepted release-manager attestation and is never moved or reused. The
successor review uses `mars3/w001-bootstrap-helper-v4`, created with the exact
public release-manager identity already enforced by the validator.

## Preserved v4 security disposition

Signed tag `mars3/w001-bootstrap-helper-v4` remains immutably attached to
commit `ef30493cbea8133dcbd88934b8103fc0f5abdaff`, tree
`3519dde9da164427db05b3410d03f09dc85de6d2`. GitHub run `32972354229`
passed the complete public gate, and independent QA accepted that exact
commit. Security requested changes before merge because the helper reused its
initial execution-authorization object after a fresh signature check, trusted
caller-controlled Git environment and remote configuration, and identified a
workspace only by cloneable project metadata.

No merge or canonical mutation followed that disposition. The successor
reloads and compares the complete signed execution authorization, resolves the
fixed public GitHub HTTPS endpoint under a sanitized Git environment, removes
ambient Beads/Dolt workspace overrides, and binds an opaque digest of the
direct canonical workspace directories and filesystem identity. The v4 tag is
not moved or reused; the successor review uses
`mars3/w001-bootstrap-helper-v5`.

## Preserved v5 first-conformance finding

Signed checkpoint `4c8a35270253fefc6f09cf65541020a0cc1b4ac6`, tree
`1f1b5e072780b0cf6a90201f7c9b188c6f1bb651`, passed the deterministic
public gate and all four Docker/Dolt atomicity cases. The non-mutating helper
then rejected its disposable workspace because macOS presents the temporary
root through `/var` while resolving that ancestor to `/private/var`. The
workspace endpoint itself was a direct directory, and no canonical claim or
other authority effect occurred.

The foundation-owned fingerprint is
`authority.bootstrap/workspace-ancestor-canonicalization`. The correction
continues to reject a symlinked workspace endpoint and symlinked database
components, but hashes the resolved ancestor path so the disposable copy and
canonical workspace use the same fail-closed instance algorithm. One bounded
retry was permitted; an equivalent recurrence would have blocked this attempt.

## Successor v5 disposable conformance

The signed correction checkpoint was
`e9b73b24461602a20085a9f04003037af7de4c65`, tree
`e83f544b2841aec73cade86bab225a5518dc4256`. From that clean checkpoint,
the one bounded retry rebuilt the pinned source and patch, ran all four
Docker/Dolt atomicity cases without a skip, and executed the patched CLI claim
against a disposable copy of the canonical `embedded` workspace. The
disposable postimage and new Dolt version were verified. The public-safe dry
run receipt bound:

- patched binary SHA-256
  `8ba8ba8ba97e83582bc552af367e053106a39c20a4c8e0fe8a2580cd17d70475`;
- helper library SHA-256
  `16c34ff26fb9b4eac10a455fb3c67856801f78d49f3a789a3dd6f6a22d1c1885`;
- opaque canonical-workspace instance SHA-256
  `bc74cc2c9592d027a7d54a19f26c69111f81b023d5c2a6d68d43e7e21095fb8f`;
- backend `embedded`, `disposableVerified: true`,
  `canonicalUnchangedBeforeEffect: true`, `mode: dry-run`, result
  `conformance-passed-no-canonical-mutation`, native status `open`, lifecycle
  `backlog`, and `liveLeaseAsserted: false`.

An independent post-run read normalized only public-safe ID, status, assignee,
creation/update timestamps, sorted labels, lifecycle, nullable WorkVersion,
and dependency ID/status/lifecycle/type. W-001 remained `open/backlog`, with
`workVersion: null` and only closed/done H-001; its normalized summary SHA-256
was `910e3d0b9ba5b3c36755781daffcfbc146341f06c46fcefcb7d714b6310e237f`.
P-001 remained `open/backlog`, with `workVersion: null` and closed/done H-001
plus open/backlog W-001; its normalized summary SHA-256 was
`c1d3774bc61b39417c68a90714c0811c23ae322d49abfbe166910c8537ba965f`.
No claim, lease, plan transition, or other canonical mutation occurred.

## Preserved v5 public-CI finding

Signed tag object `6443af6a22a49b2d46bcc2215ca78ad9b17ee0b2`
for `mars3/w001-bootstrap-helper-v5` remains immutably attached to commit
`ac43a21d7e344c03aa41e626a7c1f55ae02ee9a3`, tree
`9685f47ae1b92e5e678cf0ed710c4e04dc336890`. Public pull-request run
`32997852398`, job `98271649495`, passed build, doctrine, plan, DocSync, and
public checks, then failed in `Test and vet`. The exact failure was
`TestWave1PlanningGrantBindsCheckoutAndCommitHistory/malformed pull request
event merge SHA fails`: Go's `t.TempDir` cleanup observed a nonempty disposable
Git directory after the assertion. Later steps were skipped. No rerun, review,
merge, or canonical effect followed.

This is a recurrence of the foundation-owned
`go-test-tempdir-cleanup-git-pack-race` fingerprint after a meaningful accepted
fixture-stabilization change. The earlier correction disabled repository-local
maintenance in two fixture constructors, but the failing
`writePlanningGrantGitFixture` constructor did not apply it. The bounded
correction sets only `maintenance.auto=false` and `gc.auto=0` immediately after
`git init` in that disposable repository and asserts both local values. It does
not mutate developer or production Git configuration. The v5 tag is not moved
or reused; the single successor attempt uses
`mars3/w001-bootstrap-helper-v6` and an entirely fresh public run.

## Preserved v6 Security finding and v7 correction

Independent QA accepted v6 commit
`b71a9f02311ae71ced793f376f21d9876323adfb`, tree
`f1e2f801e5b5ad6d5bb745ae2081195e80e75885`, after exact-identity, public-CI,
signature, deterministic-build, and disposable-conformance checks. Independent
Security requested changes on that same immutable subject because the helper
did not reject `.beads/redirect`. Pinned Beads resolves that file before opening
storage, so a redirect to an exact workspace copy could produce an apparently
valid claim receipt without changing canonical authority. W-001 and P-001
remained `open/backlog`, unclaimed, and without a lease.

The bounded v7 correction treats routing as part of authority identity:

- every initial, post-conformance, and immediate pre-effect check rejects a
  `redirect` path in any filesystem form;
- the workspace digest schema is v2 and binds the resolved root, project,
  strict `dolt`/`embedded`/`M3` metadata, and device/inode identity of the root,
  `.beads`, `metadata.json`, `embeddeddolt`, and `embeddeddolt/M3`;
- the patched Beads process disables redirect following only for the bounded
  bootstrap operation and repeats the direct-store metadata, digest, and
  redirect checks inside the transaction before mutation;
- conformance adds
  `TestBatchBootstrapClaimRedirectAtTransactionBoundaryFailsClosed`; and
- external execution authorization accepts one canonical JSON object with one
  trailing newline, verifies its detached signature, and compares the exact
  payload and signature digests after the long-running conformance step.

Signed v7 code checkpoint
`b35fd0f154ce959adbe9d57cbf90666d6f5327d3`, tree
`c05436d75e99109c4ae28e8ae370ff15ab260ff8`, was retained by signed tag object
`f574145487630f248732cf7108b5edd15660de4e`. From that clean checkpoint, the
helper rebuilt the exact pinned source and patch, ran all five Docker/Dolt cases
without a skip, claimed only a disposable copy through the actual embedded
backend, and verified a new disposable Dolt version. Its bounded receipt was
schema 1, kind `MARS3BootstrapClaimReceipt`, classification `PUBLIC`, grant
`W-001-bootstrap`, and Bead `M3-W001`; the attempt, replay, and base bindings
matched the signed grant. It reported patched-binary SHA-256
`949e1d535e19ecb39e974b90b7321ef1f7f7d6b77c3958d72edb07e78d9def5a`,
opaque workspace-instance SHA-256
`4717a2c2b92ab1c0196ed119d7ec760f6c2f9f40ddc52372536e436d7467ee85`,
backend `embedded`, `disposableVerified: true`,
`canonicalUnchangedBeforeEffect: true`, `mode: dry-run`, result
`conformance-passed-no-canonical-mutation`, native status `open`, lifecycle
`backlog`, and `liveLeaseAsserted: false`.

An immediate independent readback found W-001 and P-001 still
`open/backlog`, with `workVersion: null`, their original dependency sets, and
no claim or lease. Their normalized public-safe summaries remain
`910e3d0b9ba5b3c36755781daffcfbc146341f06c46fcefcb7d714b6310e237f`
and `c1d3774bc61b39417c68a90714c0811c23ae322d49abfbe166910c8537ba965f`.

The corrected public material binds patch SHA-256
`50128252828352366ced6560371468a5746c2603ef89ea746a33be8994ffceb6`,
patched-binary SHA-256
`949e1d535e19ecb39e974b90b7321ef1f7f7d6b77c3958d72edb07e78d9def5a`,
and helper-library SHA-256
`d039c787f73e98f059937242e068d76c12753cc9accedc025bf619e1fa63c0fd`.
The evidence-only v8 successor commit was
`1ec5ad636d4d4288703607932e9c06a414a84d37`, tree
`4c188ce59039c3eb81a403b5223463488c82a8b0`, retained by tag object
`3645b291f6d5e5bed7d652afa22c1aa9e14eb232`. Public run `33004634256`, job
`98294933116`, built the validator and then failed at doctrine before later
steps because the v8 tag used the Engineer identity rather than the exact
synthetic Release Manager identity required by the signed-tag verifier. The
tag is preserved and not moved; no test, merge, Beads mutation, claim, or lease
followed that failed admission.

The bounded identity-only successor uses
`mars3/w001-bootstrap-helper-v9` with exact tagger
`MARS-3 Release Manager <release-manager@example.com>`. Its exact commit, tree,
public run, and independent dispositions remain pending until the new immutable
subject is published and reviewed. The v7 and v8 tags remain immutable. No
claim is authorized by this correction or dry run.

## Pending receipt

No claim receipt exists yet. After the authorized effect, append only the
public-safe helper receipt, the independently read-back postimage hashes, and
the opaque Beads trace/comment reference. Do not include local paths, raw
database content, terminal recordings, credentials, or provider state.

## Superseded v1 dry-run checkpoint

The signed helper was executed without `--apply` from clean commit
`e0ab3109aea538b1b883e2642da80358691f8268`. It rebuilt the pinned Beads source
and public patch, verified the signed source/toolchain/image bindings, and
executed both required disposable tests against Dolt:

- `TestBatchBootstrapClaimIsOneAtomicTransition` — passed;
- `TestBatchBootstrapClaimPreconditionFailureRollsBack` — passed.

The temporary patched binary SHA-256 was
`67d52f0e8aac0bd737c20b6d7f2596d487af9f85a8660c96c05da003681ad6e2`.
The bounded result was `conformance-passed-no-canonical-mutation`; independent
readback remained native `open`, lifecycle `backlog`, and
`liveLeaseAsserted: false`.

This checkpoint and tag `mars3/w001-bootstrap-helper-v1` are preserved as
historical evidence but are not accepted: they did not exercise the canonical
embedded backend, a post-claim failure, or contention. The v1 tag is immutable
and is not moved or reused. Successor v2 evidence must bind the four-test suite,
the disposable embedded-workspace claim, the exact Go executable and patched
binary hashes, and the short-lived post-review execution boundary.

Repository-relative verification command shape:

```text
go run ./cmd/mars3-authority bootstrap-claim --repo . \
  --beads-source <pinned-public-source-checkout> \
  --beads-workspace <external-authority-workspace> \
  --beads-binary <pinned-public-binary>
```

The two bracketed paths are external local inputs and are intentionally not
recorded. The helper verified their hashes/identity against the public grant;
the evidence does not disclose machine paths.
