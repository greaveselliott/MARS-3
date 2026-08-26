# W-001 bootstrap claim transition evidence

**Status:** Prepared; no canonical claim has executed
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
Security acceptance of the immutable reviewed feature commit. Postclaim Git
reconciliation requires another signed grant.

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
binding those exact facts. The helper revalidates the authorization, grant
expiry, authenticated remote `main`, workspace, and preimage immediately
before the sole effect.

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
