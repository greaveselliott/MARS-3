# Wave-1 contract-publication evidence

**Classification:** PUBLIC
**Goal:** G-001
**Features:** F-002, F-003
**Selected Bead:** M3-W001
**Phase:** `contract-publication`
**Status:** signed CI recovery checkpoint recorded; v2 immutable publication
review pending

This manifest records the bounded publication of the W-001 and P-001
contracts. It is not implementation evidence for either feature and creates no
claim, lease, lifecycle transition, or shipped product value.

## Authority boundary

- Git base: `ee385ce236ae1f99da692d223d7666b80dd9108f`.
- Signed planning-grant commit:
  `fc9f6641d0f739a401a4f7be3bc0ee575df1310a`, tree
  `87e4a489241fc9783d3cd17d486c05cc76b283b8`.
- Signed prospective recovery disposition SHA-256:
  `1ff3abcb481b6b0177537f4d0b4e41a13707eb8fc507656eba6d90e0b5cf23e1`.
- Recovery-disposition detached-signature SHA-256:
  `641ca4a3d15608f48c7199873318679be86b4e5b234212ce23282e3a77f583b9`.
- Reconstructible state snapshot SHA-256:
  `4d3b5c9d90a223c0e9d974e836559309a2f4dac7f209a3966336e9152f57feca`.
- Signed CI recovery addendum SHA-256:
  `73cd581393c8798ad3a44a8c2d0d5fc5f211088a0b04368b960e0514176cd882`.
- CI recovery addendum detached-signature SHA-256:
  `c987c7aafbc47a715c50caedce36117096dd716e2c617b589d5c7067b5cb8ce0`.
- The original recovery artifact remains outside Git because the pinned
  scanner flags two public checksum fields. Its exact artifact and detached
  signature checksums are bound by the signed disposition. No scanner ignore,
  allowlist, workflow change, or GitHub policy change was made.

The disposition is prospective and states
`priorExternalEffectsAccepted: false`. Independent QA and Security must decide
whether the final safer Beads state is acceptable; this evidence does not
retroactively authorize the path used to reach it.

## Durable incident chronology

| Event | Durable reference | Assessment |
| --- | --- | --- |
| P-001 blocker intent | `01a03c9a-c1d0-76be-8054-0d412b7da6cf` | Intended P-001 → W-001 dependency before implementation. |
| P-001 blocker receipt | `01a03c9b-6663-79ac-8520-9942a54052a2` | Safer dependency observed, but the planning grant did not enumerate this Beads effect. |
| P-001 authority correction | `01a03ca7-1e96-7572-a76c-ab61ef4ad44e` | Classified foundation-owned and non-retroactive; state frozen for review. |
| Invalid Git intent | `01a03c8e-a7de-7e8a-8a3b-d8e080801b08` | Proposed commit identifier was mistyped and is preserved as invalid. |
| Verified Git receipt | `01a03c8f-33f9-7ecd-ac76-60f3ad229103` | Remote signed grant commit and tree verified. |
| Trace correction | `01a03cab-37b7-7488-bba1-df4d6793a778` | Binds the invalid intent to the verified receipt without overwriting history. |
| First recovery intent/receipt | W-001 `01a03cb6-f889-7459-9704-711fdcb02699`, `01a03cbd-2108-73c8-b197-d93606e45fd8`; P-001 `01a03cb7-2869-7203-8ad7-ca2f2e01983c`, `01a03cbd-2c0d-7a87-961d-6d6d33faf02e` | Added shared build/notice paths. W-001 passed through an incorrectly typed JSON-string value before correction. |
| Recovery-scope corrections | W-001 `01a03cc4-c5c8-785c-a25e-eb84bd640d3e`; P-001 `01a03cc4-d30d-7493-a8b2-e3f3e0c90ca4` | Records missing reconstructible state/version/idempotency bindings and freezes both Beads. |
| Prospective P-001 intent | `01a03cde-0845-7a58-8325-5ce37cae64f3` | Exact description-only preimage/postimage and idempotency key under the signed disposition. |
| Prospective P-001 receipt | `01a03cde-c1a7-747c-b574-1ff0b84511b0` | Exact postimage verified at revision `r9qvsh9a`; no other authority effect observed. |
| Checkpoint intent | `01a03ce5-d05a-7e21-b03b-afc56a46327d` | Authorized one signed local contract and admission-validator checkpoint after the complete preflight passed. |
| Checkpoint receipt | `01a03ce7-c3de-7416-90a7-903963e4a800` | Binds the signed checkpoint, tree, reproducible binary hash, unchanged backlog/unclaimed authority state, and next publication step. |
| Initial publication and PR | tag receipt `01a03ce9-dd0d-72f5-9c6f-3b4ccce7fab8`; push receipt `01a03ceb-222b-7236-8f1b-09105920205e`; PR receipt `01a03cec-2028-7013-b736-0cb0a485f674` | Published immutable v1 tag and opened PR 4 at `a22cfe6`; no main or work-authority mutation. |
| CI failure and bounded retry | status `01a03ced-75f1-7a87-be16-40ca1c61fa5f`; retry intent `01a03cf1-e8a3-73ff-9203-0dd773169ca2`; diagnostic receipt `01a03cf5-1782-7501-b537-e952869e10e1` | Both attempts of run `32941818590` failed at doctrine. Debug evidence proved the hosted checkout was exact, while the pull-request event field `merge_commit_sha` was null. |
| Blocked disposition | `01a03cf5-452b-76b6-8d76-6359d5d7b391` | Exhausted the allowed equivalent retry and prohibited another attempt on the unchanged v1 head. |
| Prospective CI recovery authority | replan `01a03cf5-cbef-7989-977a-1b37c1f9b982`; signed-addendum receipt `01a03cf6-e5a7-7c06-bfff-7a3643b31f9f` | Preserves v1, authorizes only addendum/code/test/evidence paths, and permits one v2 tag without changing workflow, ruleset, scanner, Beads, claim, or lease state. |
| CI recovery checkpoint | intent `01a03cfc-ed3d-7a93-b257-752394ac5978`; receipt `01a03cfd-9056-76f9-bb82-c84b588a6753` | Binds the signed correction commit, exact topology regressions, and new reproducible validator hash. |

## Canonical final read-back

Final read-only authority revision after the receipt is `pltnrchb`.

| Bead | Native / typed state | Claim / lease | Dependencies | Exclusive-path result |
| --- | --- | --- | --- | --- |
| M3-W001 | `open` / `backlog` | absent / absent | M3-H001 | Five W implementation roots, W evidence, and serialized `go.mod`, `go.sum`, `Makefile`, `NOTICE`, `THIRD_PARTY_NOTICES` |
| M3-P001 | `open` / `backlog` | absent / absent | M3-H001, M3-W001 | Five P implementation roots, P evidence, and the same serialized build/notice paths |

P-001 description SHA-256 changed only from
`828e32051ee661e6d31c5017e996e3d8ec82257ca0a17e263f974248c5772314`
to the signed postimage
`d24074a56a4df0d150450555b981f84fa377364f40c9c9191e49f4ecae73c20c`.
Its implementation roots are disjoint from W-001; the shared build and notice
files are serialized by the W-before-P dependency and later live leases.

## Admission hardening completed

- The plan validator binds phase, selected Bead, manifest projection, feature
  lineage, exact scenario schedule/headings, exact delivery-table grammar, and
  canonical dependency cells. Shadow, malformed, truncated, duplicate, and
  out-of-order forms fail closed.
- Changing the Git plan to `delivery` cannot disable grant enforcement. A
  separately signed W-001 bootstrap grant and canonical claim evidence are
  required before the validator can change authority modes.
- The grant gate validates the exact symbolic branch, every signed feature
  commit, per-commit path union including add-then-delete, current worktree,
  the signed recovery disposition/snapshot, and the signed CI addendum. Paths
  authorized before v1 cannot be reused after its target.
- Pull-request admission permits a null optional event `merge_commit_sha` only
  when `GITHUB_SHA`, the event base/head identities, and the exact two-parent
  Git checkout all agree. Any nonempty mismatch still fails closed.
- FactoryDocSync unions all matching ancestor-prefix requirements. A more
  specific database rule cannot drop foundation or Rule-of-Two coverage.
- The current squash-only linear GitHub policy is retained. The immutable v1
  tag preserves the failed reviewed attempt. Final PR and main CI require the
  separately authorized signed v2 tag; its target keeps the corrected signed
  feature history reachable and its tree must equal both reviewed PR and
  squash-main trees.

## Reproducible worktree checks

Run from the repository root with public inputs only:

```text
go run ./cmd/mars3 doctrine check --repo .
go run ./cmd/mars3 plan check --repo .
go run ./cmd/mars3 docsync audit --repo .
go run ./cmd/mars3 public-check --repo .
go test ./...
go vet ./...
git diff --check
gitleaks detect --no-git --source . --redact --no-banner
gitleaks detect --source . --redact --no-banner
go run ./cmd/mars3 doctrine refresh --repo . --source <exact-pinned-mars-checkout> --ref f55d129bfc794510ca485bb54fc0a35c7b04a700
```

All commands passed before the original checkpoint and again on the corrected
CI-recovery worktree on 2026-08-26. The offline refresh verified 20 source
files and remained a dry run. The pinned scanner reported no worktree or
history leaks and required no exception.

## Immutable publication evidence

- Implementation checkpoint commit:
  `fe72df5b81f1bd9dae2cb799948a10b36b12ab80`.
- Implementation checkpoint tree:
  `eb0cd99b684ce2e2f7837dabc8c90396f83fa75f`.
- Reproducible Linux/amd64 validator binary SHA-256:
  `46ae0935e766da18e21107685eda7968af3695ebf24f3034beafce66998a6728`.
  Two clean builds with distinct empty caches produced identical bytes using
  Go 1.26.2, `CGO_ENABLED=0`, `GOOS=linux`, `GOARCH=amd64`, `-trimpath`,
  `-buildvcs=false`, and an empty build ID.
- Immutable failed v1 tag: object
  `4bce7e7d4a8b2cc1a5b30b9feaee61232c3cc0de`, target
  `a22cfe6fada6f2bc787742eae50bca28cec80c89`, tree
  `3c5befaefab37a8d0a2e3a8af2efd6e1eb1d8cae`. It is preserved and never moved.
- CI recovery checkpoint commit:
  `178c5094a1462975b21b961c060890055a09deaf`.
- CI recovery checkpoint tree:
  `c2e28762618954cf68e34d8cbbd6ff76c4708cf0`.
- Corrected reproducible Linux/amd64 validator binary SHA-256:
  `8567f5bc5461f8e7fe0b9e9702b745d0e6e8ab74320c61d3f2d5cc63ba5f3867`.
  Two builds with distinct empty caches were byte-identical under the same
  Go 1.26.2 build tuple.
- Corrected signed publication v2 tag object/target/tree: created only after
  this evidence commit. The validator binds its exact ref, target ancestry,
  message, signer, equality with reviewed and squash-main trees, and continued
  immutability of v1. The external publication receipt records the resulting
  identifiers.
- Remote PR: [PR 4](https://github.com/greaveselliott/MARS-3/pull/4) is open;
  its v1 head failed run `32941818590` twice as preserved above. Corrected-head
  push and v2 check are pending.
- `Public commit gate` corrected PR and protected-main runs: pending.
- QA disposition: pending.
- Security disposition: pending.
- Orchestrator publication/reconciliation disposition: pending.

The remaining publication values keep this transition in progress. They are
not inferred from passing local checks, and no self-referential tag or merge
identifier is fabricated inside the commit that it identifies.
