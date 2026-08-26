# Wave-1 contract-publication evidence

**Classification:** PUBLIC
**Goal:** G-001
**Features:** F-002, F-003
**Selected Bead:** M3-W001
**Phase:** `contract-publication`
**Status:** signed implementation checkpoint recorded; immutable publication
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
  and the signed recovery disposition and snapshot.
- FactoryDocSync unions all matching ancestor-prefix requirements. A more
  specific database rule cannot drop foundation or Rule-of-Two coverage.
- The current squash-only linear GitHub policy is retained. Final PR and main
  CI require signed tag `mars3/wave1-contract-publication-v1`; its target keeps
  the signed feature history reachable and its tree must equal both reviewed
  PR and squash-main trees.

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

All commands passed on the pre-checkpoint worktree on 2026-08-26. The offline
refresh verified 20 source files and remained a dry run. The pinned scanner
reported no worktree or history leaks and required no exception.

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
- Signed publication tag object/target/tree: created only after this evidence
  commit. The validator binds its exact ref, target ancestry, message, signer,
  and equality with the reviewed and squash-main trees; the external
  publication receipt records the resulting immutable object identifiers.
- Remote branch and pull request: pending.
- `Public commit gate` PR and protected-main runs: pending.
- QA disposition: pending.
- Security disposition: pending.
- Orchestrator publication/reconciliation disposition: pending.

The remaining publication values keep this transition in progress. They are
not inferred from passing local checks, and no self-referential tag or merge
identifier is fabricated inside the commit that it identifies.
