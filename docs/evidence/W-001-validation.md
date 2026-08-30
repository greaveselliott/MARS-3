# W-001 Security correction validation

**Classification:** PUBLIC
**Work authority:** M3-W001
**Failure ownership:** foundation
**Correction grant:** `W-001-lifecycle-correction-v6`
**Current disposition:** core gateway accepted and merged; independent review changes-requested the v5 lifecycle candidate; W-001 remains in-progress
**Historical disposition:** postclaim reconciliation accepted, merged, and completed

## Superseded Security disposition

The earlier Security acceptance of helper commit
`663d19bf190f9e3bd27edc96ee08acaa6778c853` is superseded. A later bounded
adversarial test showed that the helper could validate direct embedded metadata
for database `M3` while a project-local environment selector caused the pinned
Beads client to open an alternate embedded database. Backend status still
reported `embedded`, so mode-only validation did not prove the effective
database.

The additive Security `changes-requested` disposition is bound to postclaim
head `20542d8e696abe0a71b6ec3ceb23f042919fbc04`, tree
`499ea91f5002b32d57d6f20b4ca3ea07dbdc73f5`, and normalized failure
fingerprint `bootstrap-effective-database-selector-splice`. The earlier signed
grant, commits, tags, CI records, and review records remain immutable historical
evidence; this record does not rewrite or relabel them.

## Canonical effect remains independently verified

The finding applies to the helper's admission and transaction-authority proof.
It does not negate the independently read-back canonical `M3` effect:

- M3-W001 is `in_progress` with lifecycle `in-progress` and assignee
  `work-authority-engineer`.
- The WorkVersion generation is
  `6e79ff81-a007-42a5-a178-7ce58dbb718b`, incarnation is
  `e1e8d2d3f80871096a568fb489f49575a42abd37b269df9faf777a09cd689b41`,
  issue mutation sequence is `1`, and dependency-graph revision is `1`.
- The canonical Dolt commit is `67hmen0cmq0he08n7ujlqpcsmmi94fhb`.
- The bounded claim receipt SHA-256 is
  `04cef4e421a34e0908d392fc794181db3ddb754a134e34599fa41a520c78d126`.
- M3-P001 remains backlog and unclaimed.
- No alternate-database use was observed in the canonical effect, and no live
  lease exists.

These facts establish what happened in canonical `M3`; they do not establish
that the superseded helper was safe for another invocation. The one-shot claim
must not be replayed.

## v3 Security disposition

QA accepted v3 commit `9e8a587f8187c2d385a6c5fa023346405733d7ff`
and tree `fb4a0abea8f3e77a92ca86672078d2d68df7a9e0`. Security requested
changes after a later bounded synthetic test proved that the helper did not
disable project workspace hooks. Pinned Beads therefore wrapped the embedded
transaction and could execute `.beads/hooks/on_update` after commit. The v3
patch also forwarded bootstrap authority through that hook decorator and did
not reject the last-merged `config.local.yaml` selector. The normalized
foundation-owned failure fingerprint is
`bootstrap-workspace-hook-postcommit-effect`.

The v3 commit, tag, successful public checks, QA acceptance, and Security
changes request remain immutable evidence. They are not final acceptance and
must not be used to merge or replay the one-shot claim.

## v4 corrective control

The v4 correction keeps both earlier patch layers immutable and adds one narrow
hook-isolation layer:

1. The helper rejects project `.env` and `config.local.yaml` selectors, accepts
   only a comments-only direct `config.yaml`, binds metadata and config content
   plus filesystem identity, and repeats that binding at every pre-effect
   boundary.
2. Every helper Beads command receives an exact direct `BEADS_DIR`, effective
   database `M3`, disabled server/shared-server selectors, and
   `BD_NO_HOOKS=1`.
3. Backend verification requires both embedded mode and a separate effective
   identity report naming backend `dolt`, database `M3`, and `embedded: true`.
4. The patched batch operation asks the same SQL transaction to prove its
   effective database and embedded identity before any issue, label, or
   dependency read.
5. Only the bare embedded transaction implements the concrete proof.
   Server-backed and hook-decorated transactions cannot claim bootstrap
   authority.
6. The transaction-boundary workspace proof uses schema version 3 and includes
   SHA-256 content bindings for both `metadata.json` and `config.yaml`.

The human authority explicitly approved the additive integrity transition:
historical v3 helper and evidence bytes are validated from the immutable signed
v3 Git tree, while current bytes are validated from the signed v4 grant. Both
checks are mandatory; neither check substitutes for the other or grants an
operational effect.

## v5 publication-vehicle correction

Independent QA and Security reviewed v4 head
`d890a96014f79438d36bde3c8967664163e9d961`, tree
`3d3252b0559e664203521af2e85d0d87cdb9fcd1`, and the successful exact-head
pull-request run `33022606025` / job `98356474178`. Both requested changes
because the v4 evidence still named PR #7 as the merge vehicle after that PR
had been closed without merging. The normalized foundation-owned failure
fingerprint is `stale-publication-vehicle-binding`.

PR #7 remains closed and unmerged historical evidence. PR #8 is the sole
active publication vehicle for the successor reviewed tree. This additive
correction does not rewrite v1 through v4, move any retained tag, alter the
canonical M3 postimage, or grant a lease or implementation capability.

## v6 publication-authority chronology correction

Security subsequently found that the v4 and v5 signed grants were not yet
effective when their governed public Git effects occurred. This is recorded as
the foundation-owned failure fingerprint
`grant-effective-after-governed-effects`, not retroactively relabelled as
authorized:

- v1 through v3 have `issuedAt` timestamps no later than their governed
  commits and signed tags.
- v4 declares `2026-08-26T23:45:00Z`, while its commit and tag were created at
  `2026-08-26T22:59:53Z` and `2026-08-26T23:00:29Z`; its exact-head PR run
  began at `2026-08-26T23:15:21Z`.
- v5 declares `2026-08-27T00:00:00Z`, while its commit and tag were created at
  `2026-08-26T23:38:40Z` and `2026-08-26T23:38:54Z`; its exact-head PR run
  began at `2026-08-26T23:40:36Z`.

The v4 and v5 commit, tag, CI, QA, and Security records remain immutable
historical evidence, but their grants do not authorize those pre-effective
publication effects. The already-effective v6 grant compensates by publishing
the complete reviewed tree through a new signed commit, tag, PR #8 run, and
protected-main merge path. It changes no helper, patch, Bead, lease, runtime,
workflow, or repository control.

## Reproducible verification

Commands run from the repository root use only synthetic identities and
fixtures:

```text
gofmt -w internal/authority/bootstrap/bootstrap.go internal/authority/bootstrap/bootstrap_test.go
go test ./internal/authority/bootstrap -count=1

# From an exact pinned Beads checkout after applying the immutable base patch
# then the bounded effective-database patch, and finally the hook-isolation
# patch:
ICU_PREFIX="$(brew --prefix icu4c@78)"
CGO_ENABLED=1 CC=/usr/bin/clang CXX=/usr/bin/clang++ \
CGO_CPPFLAGS="-I${ICU_PREFIX}/include" \
CGO_LDFLAGS="-L${ICU_PREFIX}/lib" \
DOCKER_HOST=unix:///var/run/docker.sock \
TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock \
TESTCONTAINERS_RYUK_DISABLED=true \
go test ./cmd/bd -run '^TestBatchBootstrapClaim' -count=1 -v
```

The patched Beads conformance suite is generated from the pinned public source
revision and covers:

- one atomic embedded transition;
- stale-precondition rollback;
- post-claim rollback;
- one-winner contention;
- redirect insertion at the transaction boundary;
- alternate-database `.env` selector rejection with both stores unchanged;
- shared-server config rejection with canonical state unchanged;
- `config.local.yaml` rejection at the transaction boundary;
- server-backed transaction rejection before an authority read or write; and
- hook-decorated embedded rejection with no state or sentinel effect.

The ten-case patched suite executed without skips and passed against the
digest-pinned Dolt server image. The repository helper tests also passed. No
canonical workspace or Bead was used by either test suite.

The immutable base patch remains SHA-256
`50128252828352366ced6560371468a5746c2603ef89ea746a33be8994ffceb6`.
The five-file security delta is independently bound as
`d48b398a8688d337192ab030c69fd9df0809f72051da7850ff2fdbad5e322d45`.
The three-file hook-isolation delta is independently bound as
`fc282ebc257fc41c15ab7b8ffdd50a1600f840254cf6e10b6997f9e30a0dc1fc`.
Two clean builds with distinct caches produced the same corrected binary
SHA-256, `22042fc0844ab7700417d917c386f2eab4bab5dd6a6404be091cbd5edbe9e154`.
This composition preserves the previously reviewed patch bytes and prevents
the correction grant from silently rebinding unrelated historical changes.

The exact generated patches, helper, tests, signed correction grants, retained
v1 through v6 tags, public commit gates, and independent dispositions are
bound by their immutable reviewed Git trees.

## Postclaim merge and delivery handoff

Independent QA and Security accepted v6 head
`c6749bceb7114b16d7941afc7609c158295ccd2b`, tree
`7febda7ec2fec47b7d6bf11fdd5b24e605b9e2b2`, and signed annotated tag
object `2346a1388272569bb64817ea7e9b6463c4e84e5a`. PR #8 was squash-merged
without bypass as main commit
`59f1fe24952b68bd3bbb6994bfee46c350b7c9cd`, whose sole parent is
`adfd64feb565fb703a3568122cc032d4d1a450f5` and whose tree is exactly the
reviewed v6 tree. Protected-main run `33025602656`, job `98366054428`,
completed successfully.

The Delivery Orchestrator appended the exact public-safe reconciliation
comment to canonical M3-W001 as comment
`01a0408e-ca08-71f0-b1ac-0dec0039706a` at
`2026-08-27T00:11:35Z`. Its text SHA-256 is
`9c1becf8bc3e1efd7b59b41439cbb7b382f536271d5478f1750545c49fffae74`.
Read-back left W-001 `in_progress` / `in-progress`, WorkVersion mutation and
dependency revisions `1` / `1`, P-001 backlog and unclaimed, and the live
lease absent. This completed the postclaim reconciliation subjob without an
implementation effect.

The first delivery publication branch was retained at signed head
`919f1189fb0703e42bcc11570a59527ad8e7a444`, tree
`08f35a046a4de7583f9b2f3b80374d4f808d4263`, after the pinned scanner
classified public synthetic idempotency fixtures and the signed grant token as
generic-key candidates. No credential was present, but the required worktree
and full-history scans did not pass. The finding is foundation-owned. No
scanner exception, ignore file, ruleset change, force-push, or history rewrite
was authorized or performed. Delivery v2 instead starts from accepted main,
uses scanner-safe synthetic identifiers, preserves the first branch as public
history, and must pass both pinned scans with zero findings before review.
The digest-pinned scanner image
`sha256:75bdb2b2f4db213cde0b8295f13a88d6b333091bbfbf3012a4e083d00d31caba`
reported zero findings for the candidate worktree and for a disposable
standalone 43-commit history containing accepted `main`, the preserved public
transition tags, and one synthetic candidate commit with the detached v2 grant
signature. No ignore or alternate scanner configuration was present.

The separately signed `W-001-delivery-v2` grant is now the bounded implementation
authority for the Work Authority Engineer. It permits only W-001's canonical
exclusive paths plus the named Orchestrator-owned plan and validator paths,
public synthetic development fixtures, and W-001 development/test leases. It
does not authorize another canonical claim, a lifecycle transition, review or
terminal disposition, production, destructive work, repository-control
changes, secrets, private data, or any other Bead. No development lease exists
at this handoff checkpoint.

## Native gateway CAS implementation checkpoint

Historical signed delivery-v1 commit
`85848a524d40e3041199c21b89e82f2cf8910b39` adds the normal gateway claim
path without replaying the one-shot bootstrap. Delivery v2 republishes those
reviewed source semantics from accepted `main`; the historical commit remains
evidence and is not represented as an ancestor of the v2 review branch.
The MARS-3 adapter reads one bounded Beads projection, binds native status,
assignee, timestamps, canonical metadata, sorted labels, dependency metadata,
WorkVersion, integrity digests, attempt, idempotency key, base commit, and exact
post-metadata, then invokes one reviewed native transaction. The transaction
proves that it is the bare embedded `M3` store, rechecks the direct workspace,
performs `ClaimIssue`, metadata, and lifecycle-label changes atomically, and
returns one strict receipt. Direct SQL, ordinary multi-command updates,
workspace hooks, redirects, selectors, server transactions, and ambient
configuration remain denied.

The scanner-clean incremental gateway patch is SHA-256
`70c2482cfa1fa9adf85c1bc130897c6ea46c13b194010f205a95ac3ac645e47b`.
It applies with zero context after the three immutable bootstrap patch layers
listed above to pinned Beads revision
`6c124203e771433a3550c348771a5b5e27fd3c21`. A fresh composition matched the
reviewed source byte-for-byte, passed `git diff --check`, and passed both native
gateway tests:

- one bounded atomic transition with exact postimage; and
- concurrent identical claims with exactly one winner.

Two clean `go1.26.2` Darwin/arm64 CGO builds, using distinct build caches,
`-trimpath`, `-buildvcs=false`, and an empty build ID, were byte-identical at
SHA-256
`7ac16c8bd399baf5dd91d57345b39cf8136b490aaf51e26948c0ff1dfc3e87b3`.
That exact binary then passed the repository's public synthetic
`TestNativeMutatorIntegration`, including the first claim and stale replay
denial. The full MARS-3 suite passed under the race detector; vet, doctrine,
plan, DocSync, public, and whitespace gates also passed.

All Beads mutation tests used disposable public synthetic workspaces. This
checkpoint did not read or mutate canonical M3-W001, did not provision or
issue a PostgreSQL lease, and did not create an implementation effect.

## Rebaseline claim-identity correction

Delivery v2 also corrects the coherent-baseline check to compare canonical
Beads `ClaimAttemptID` with the lease's separately bound
`CanonicalClaimAttemptID`, not with the delivery execution attempt. The focused
regression accepts a valid lease whose delivery and canonical-claim attempts
differ, and rejects a lease bound to a different canonical claim. A second
regression proves the same separation for an interrupted canonical-phase
claim-to-lease reconciliation saga. The complete authority package tests and
vet pass. The isolated PostgreSQL integration checkpoint below also exercises
this corrected baseline against the v2 tree. F-002-S3 and F-002-S6 remain
failing until the complete immutable evidence is independently reviewed.

## Synthetic pre-effect linearization checkpoint

The control-plane-only synthetic broker now acquires the exact lease row before
its final canonical read and holds that guard through the simulated effect.
Lease release, renewal, and Security revocation serialize at that boundary: a
revocation that wins first denies the effect, while a revocation that arrives
after the guard waits until the already-authorized synthetic effect finishes.
The receipt is appended only after the row guard is released, preserving the
PostgreSQL project-then-lease lock order. A failed synthetic callback records a
bounded unknown outcome and never a verified receipt; a nil callback is denied
as invalid. The executor is in-process only and is deliberately absent from the
HTTP surface. Real filesystem, source-control, and network broker execution
remains S-002 work.

Deterministic gateway tests prove revocation serialization, post-revocation
denial, one-effect execution, unknown-outcome handling, and nil-effect denial.
The isolated PostgreSQL integration fixture also passed the matching row-lock
contention case against this v2 tree. This checkpoint does not by itself move
F-002-S4 to passing because Security has not yet reviewed an immutable
candidate.

## Journal high-watermark fail-close checkpoint

The projection consumer now verifies that a page's declared high watermark is
internally coherent with its sequence and hash. A page that advertises
unseen canonical events but provides no contiguous progress makes the
projection stale instead of being accepted as an empty poll. Reaching the
declared terminal sequence with a different terminal hash is likewise a
conflict. A bounded page may still advance contiguously below a later high
watermark so normal pagination remains available.

Focused deterministic tests cover the empty-ahead page, terminal-hash
conflict, contiguous pagination below the high watermark, exact duplicate,
sequence gap, hash conflict, truncation, and derived-version conflict. The
focused projection suite and the complete authority package suite pass. This
closes a deterministic replay fail-open but does not move F-002-S6 to passing:
the stable-cut/rebaseline integration fixture now passes, but independent
immutable QA and Security review remain outstanding.

## Isolated PostgreSQL integration checkpoint

The credential-free integration suite ran against an ephemeral PostgreSQL
17.11 container bound only to loopback and pinned by image digest
`sha256:051f7b7b3abdd564d5d1bd1e8c4b9c1b6e77087d1dd22020ede611c096a272e0`.
The container used trust authentication only for its synthetic, short-lived
database and synthetic `mars3_test_app` role; no password, API key, private
identifier, or persistent volume was created. The container was stopped and
automatically removed after the test.

`TestPostgresLeaseLifecycleAndRestart` executed without a skip and passed. It
applied the checked-in migration, exercised direct application-role RLS by
proving an out-of-scope project could not be read, updated, or inserted,
durable epoch allocation and restart, generation mismatch, renewal,
release/revocation/expiry, path-overlap contention with one winner, exact
effect-row serialization, ordered journal replay and truncation, the exclusive
project barrier, stable-cut double-read rejection, and claim-to-lease baseline
consistency. All identifiers and trace references were synthetic. No canonical
Bead, production service, external network, or durable local database was
used. These results remain candidate evidence until bound to a signed commit
and independently reproduced or inspected by QA and Security.

The explicit `test-authority-postgres` Make target now rejects a run when
either synthetic endpoint is absent instead of allowing Go's integration-test
skip to look like positive evidence. Its missing-endpoint negative check exits
nonzero, and the same target passes with both isolated loopback endpoints.
Separately, the complete API and authority package suite passes under Go's
race detector on the v2 candidate.

## Signed delivery-v2 publication checkpoint

The scanner-clean delivery series is rooted at accepted `main`
`59f1fe24952b68bd3bbb6994bfee46c350b7c9cd` and contains these signed semantic
checkpoints before this evidence-only update:

- `fce3985e6ccca600bd3c808bdafa89587bc95638` establishes delivery authority;
- `0edcee81cd303ba886062bad5bd8b5846487844a` adds the typed gateway and canonical CAS;
- `2821e81afa6972106828489b68dcc1c7e7a90a09` adds PostgreSQL fencing and rebaseline;
- `26f7cb3048add2af1c65e84df9164382d1e96246` adds transport and sandbox isolation.

The immutable implementation checkpoint tree is
`6e156306584d736f26df33f9f2a6c4d7ce4b5637`. Each commit verifies with the
pinned human-bootstrap-authority ED25519 key. The signed delivery grant and
detached signature have SHA-256 values
`3f4d6ee6075e40ec49eefd24a9a20d734619833be61a58547d97f402258b055a` and
`3b7951067a50975875fdeb194c51680c66869868a27e94bcb53a031d4c438f45`.

A clean standalone clone containing accepted `main`, the preserved public
transition tags, and the signed delivery series passed doctrine, plan,
DocSync, public-check, all Go tests, vet, strict Git checks, and the complete
authority race suite. Digest-pinned worktree and full-history secret scans
reported zero findings. The independent reviewers must bind their verdicts to
the final evidence commit and verify that its parent is the implementation
checkpoint above; this paragraph does not self-attest its own commit identity.

A first local, unpublished version of this evidence checkpoint placed a commit
digest immediately after an authority-like field label, which the pinned
generic-key heuristic rejected. It contained no credential and was never
pushed, tagged, or submitted for review. The local candidate was replaced
before publication, without changing any implementation commit or weakening
the scanner configuration.

## Delivery-v2 review-tag admission finding

The signed v2 tag object
`9eb770c85a1df06dd90e993c9447176c9bbbffd0` immutably targets delivery head
`ac20b235724b2219e5db230a7a44b507e46d5547`, tree
`4812b71b88500101688be7c80f41461a79619646`. Its signature uses the pinned
human-bootstrap-authority key and its synthetic tagger is the Work Authority
Engineer, matching the v2 grant's principal and explicit permission to create
one v2 review tag.

Public run `33053544349` nevertheless rejected that tag twice: attempt-one job
`98454619462` and the single allowed retry job `98454903898` both emitted the
same `public.w001_delivery_tag_target` finding. Read-back proved the target was
exact. Code inspection established the normalized foundation failure as
`delivery-review-tag/release-identity-mismatch`: the shared tag parser required
the Release Manager identity but reported every parser error as a target
failure. The retry budget is exhausted; the run will not be retried again.

The signed `W-001-delivery-ci-correction-v3` grant preserves the v2 tag and both
failed attempts as historical evidence. It authorizes only a fail-closed
identity/diagnostic correction and one successor Release Manager tag. It does
not authorize a Beads mutation, lease, runtime behavior change, policy change,
or production effect. Fresh QA and Security review must bind the final v3 head,
tree, tag, and successful public run before merge.

## Delivery-v3 repository-history scanner finding

Signed v3 head `383ea617ad2bcbe06522a30014a1b19127b5239f`, tree
`e91776b8de9e9d1e1e193ae9588363c4d87a62e6`, and Release Manager tag object
`700d85715981fb6e9def191b414c815c8f543dd0` corrected the tag admission defect.
Public run `33066374068`, job `98497338894`, then passed doctrine, plan, DocSync,
public-check, Go tests, vet, whitespace, and the synthetic secret canary. Its
worktree scan reported zero findings. The complete-history scan fetched 69
reachable commits and reported ten `generic-api-key` findings in the preserved
delivery-v1 branch, so the run correctly failed.

The normalized failure fingerprint is
`delivery-history-scanner/preserved-v1-synthetic-generic-key`.

All ten findings are public synthetic idempotency, generation, or grant values
already documented by the scanner-clean v2 replacement. Their full immutable
fingerprints are the exact contents of `.gitleaksignore` and the signed
`W-001-delivery-scanner-correction-v4` grant. A disposable 69-commit clone with
that exact file passed the complete-history scan. Adding a new committed
synthetic credential canary still produced one `github-pat` finding and a
nonzero result. No wildcard, regex, rule, path, commit-range, workflow, image,
or scanner-version exception is permitted.

The v4 correction must mechanically pin the complete ten-line file and resolve
every commit, path, rule, line, and preserved-branch ancestry before review.
Changed, extra, missing, duplicate, or unresolvable entries must fail closed.
The v2 and v3 tags and failed runs remain immutable; neither failed head may be
rerun. No Beads, lease, runtime, production, repository-setting, trust-root, or
destructive effect is authorized.

## Core delivery merge and completion audit

QA and Security independently accepted delivery-v4 head
`cac4231ddcb69edd298766c5bbe3854c8269fb2a`, tree
`5a9f006b0cd65364c2fdcfb403efd554f0e34dda`, and signed tag object
`98a3f34c24868e49ca4909c8b0303f34c25390f3`. Pull request 9 run/job
`33068767781/98505346256` passed every public gate. The protected-main squash
`7f35c8a7112946a9569efe6085f49da8fd28530e` has sole parent
`59f1fe24952b68bd3bbb6994bfee46c350b7c9cd`, preserves the exact reviewed
tree, and protected-main run/job `33069887434/98509103754` passed.

That evidence accepts the core gateway, claim, live-lease, effect-fencing, and
rebaseline implementation, but it does not make W-001 terminal. A completion
audit found no typed gateway route or native bounded CAS for handoff, ordered
QA/Security verdicts, run disposition, merge reconciliation, or the guarded
`done` transition required by F-002. The normalized foundation finding is
`completion-audit/governed-lifecycle-routes-missing`. W-001 remains the same
in-progress Bead with WorkVersion mutation/dependency revisions `1/1`, and no
live lease exists. The exact public-safe Beads status comment is SHA-256
`d7ddb1c0d4ecb00b93fcbec4d56b740da581a725e91e6381601d2d295203c38d`.

The prospective signed `W-001-lifecycle-completion-v5` grant authorizes only
the missing lifecycle implementation, public synthetic tests and leases, this
truthful plan/evidence correction, and one exact status comment. It does not
authorize a canonical lifecycle transition, another Bead, a production lease,
repository-control changes, secrets, private data, or destructive work.

Canonical read-back confirms that the authorized public-safe finding was
appended exactly once as M3-W001 comment
`01a04328-9ad4-749f-8e46-ce5e15b68f94` at `2026-08-27T12:18:50Z`. Its exact
text reproduces SHA-256
`d7ddb1c0d4ecb00b93fcbec4d56b740da581a725e91e6381601d2d295203c38d`.
The same read-back showed comment count `81`, native status `in_progress`,
typed lifecycle `in-progress`, WorkVersion mutation/dependency revisions
`1/1`, and no lease field. This resolves the initially unknown comment-command
outcome without retrying it; no canonical work mutation accompanied the
comment append.

## Lifecycle-completion v5 review disposition

Independent QA and Security reviewed exact public PR 10 v5 head
`523ead6f899c413cb0a388c60a30b33aed88b8b6`, tree
`aaff531b1b0fee9dfa907a5a52c0afd98abf050c`, signed tag object
`15dbd1be9d1d098eb2f5da3dbafe824064dbff1f`, and successful run/job
`33077554760/98535652734`. Both verdicts were `changes-requested`; the same
Bead remains `in-progress` and the tag, commit, CI record, and verdicts remain
immutable evidence.

The exact foundation findings are:

- `lifecycle.terminal_claim_binding_fail_open`: terminal projection accepted
  missing or contradictory claim lineage and stripped detailed evidence;
- `lifecycle.handoff_replay_fence_splice`: replay did not retain the canonical
  claim attempt or complete authority fence;
- `lifecycle.missing_receipt_replay_success`: retry after native CAS plus a
  failed journal append could report success without repairing durable trace;
- `lifecycle.nonterminal_convergence_deadlock`: blocked review and declared
  noncompleted run outcomes could consume their only recovery route;
- `lifecycle.qualification_not_reproducible`: the attested patched-binary hash
  lacked a complete reproducible build contract.

## Lifecycle-correction v6 candidate

The prospective signed `W-001-lifecycle-correction-v6` grant permits only the
five corrections above, their public contracts and tests, reproducible
qualification, and a fresh v6 review publication. It grants no canonical Beads
mutation or live lease and cannot merge before fresh QA and Security
acceptance.

The candidate now requires exactly one complete WorkClaim or BootstrapClaim on
every active or terminal versioned work record and rejects stripped terminal
evidence. Current and archived handoffs retain the canonical-claim attempt and
a SHA-256 digest over the complete normalized fence. A replay returns success
only after appending a deterministic reconciliation receipt when the original
receipt is absent. Blocked review and every noncompleted run retain public-safe
reason, blocker, normalized fingerprint, bounded attempt, and next action;
reopen and rehandoff archive their full earlier review/run cycle rather than
creating duplicate work. Completed closure still requires the accepted QA and
Security chain, merged evidence, completed run, and reconciliation.

Candidate material hashes:

- lifecycle patch SHA-256:
  `2dd3e2be93e3e0a571f384d077cb9739c144c5f22a00005d9b712a03da575411`;
- twice-reproduced patched Beads binary SHA-256:
  `6d273b90d0a6626f1903dd8b66a95a2c4650c7bad0aae124029369c15fc49432`;
- Go tool `go1.26.2 darwin/arm64` executable SHA-256:
  `005640c7ff93028cb704283b0f737f2db3faf8b51b2561170c769b83905da646`;
- Apple clang `17.0.0` executable SHA-256:
  `a961f78075d8e7621ef4f5d764c64ef8a41bf66c0a98ab5cb6ca39b85ce31c93`;
- ICU package: `icu4c@78` version `78.3`.

The reproducible build starts from pinned Beads commit
`6c124203e771433a3550c348771a5b5e27fd3c21`, applies the three immutable
bootstrap patches, the gateway patch, then the lifecycle patch in repository
order, and uses this environment contract with distinct empty home, temporary,
and build-cache directories for each build:

```text
env -i PATH=<go-root>/bin:/usr/bin:/bin HOME=<empty-home> TMPDIR=<empty-tmp> \
  GOCACHE=<empty-build-cache> GOMODCACHE=<go.sum-populated-module-cache> \
  GOTOOLCHAIN=local CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 \
  CC=/usr/bin/clang CXX=/usr/bin/clang++ \
  CGO_CPPFLAGS=-I<icu4c@78-78.3-prefix>/include \
  CGO_LDFLAGS=-L<icu4c@78-78.3-prefix>/lib LANG=C LC_ALL=C TZ=UTC \
  <go-root>/bin/go build -trimpath -buildvcs=false -ldflags=-buildid= \
  -o <output> ./cmd/bd
```

Two clean builds with distinct home, temporary, and build-cache directories
were byte-identical at the bound hash. The exact composed patched source then
ran every `AuthorityLifecycle` native test without skips against the
digest-pinned Dolt server image. Direct transition, changes-requested history,
transaction rollback/contention, hook denial, server-transaction denial,
blocked review, all seven noncompleted run outcomes, and missing/dual/malformed
shadow claim rejection passed. Project `TestNativeMutatorIntegration` executed
the reproduced binary without a skip and passed.

The credential-free PostgreSQL suite ran against an ephemeral PostgreSQL 17.11
container bound only to loopback at image digest
`sha256:051f7b7b3abdd564d5d1bd1e8c4b9c1b6e77087d1dd22020ede611c096a272e0`.
`make test-authority-postgres` executed
`TestPostgresLeaseLifecycleAndRestart` without a skip and passed; the temporary
container was then stopped and auto-removed. Scenario states remain `failing`
until the full public gate and independent QA and Security verdicts bind one
signed immutable v6 tree. No canonical Beads lifecycle, live lease, or
production effect was performed by this candidate verification.

## Lifecycle-correction v6 review disposition

Independent QA and Security reviewed public PR 10 exact head
`e0f27046ec28ab924eac910d40e244cb26b30323`, tree
`bbc1aa76b3965f0740e54f984ad713978a3be9f8`, signed tag object
`d8637c7443ab04e05892ecf5489f0b45fa41e43d`, and successful run/job
`33083662143/98557343299`. Both verdicts were `changes-requested`; the same
Bead remains `in-progress`, no live lease exists, and the immutable v6 commit,
tag, run, and dispositions remain evidence.

The exact additive findings are:

- `lifecycle.claim_lineage_not_joined`: current and archived handoff attempts
  were not joined to the sole retained claim, incomplete type-specific claims
  and contradictory legacy lifecycle shadows could still project;
- `lifecycle.failure_fingerprint_retry_not_monotonic`: some noncompleted runs
  could omit their fingerprint and equivalent failures could recur without a
  monotonic second-attempt blocked escalation;
- `lifecycle.qualification_not_independently_reproducible`: independent cold
  builds from the published source and recipe produced internally stable but
  different binary hashes, so the build boundary was underspecified.

## Lifecycle-correction v7 candidate

The prospective signed `W-001-lifecycle-correction-v7` grant permits only the
three corrections above, their public contracts and tests, hermetic independent
build qualification, and one fresh v7 review publication. It grants no canonical
Beads mutation or live lease and cannot merge before fresh QA and Security
acceptance. Historical v6 materials remain immutable.

The v7 candidate now rejects null, malformed, incomplete, dual, and
type-confused WorkClaim or BootstrapClaim objects. Exactly one type-specific
claim is retained, and the canonical-claim attempt on every current and
archived handoff must equal that claim's attempt. Detailed review, run, and
reconciliation records are authoritative; legacy lifecycle scalars may be
absent or exactly derived, but cannot contradict them. Every noncompleted run,
including `in-review` and `no-work`, retains a normalized public-safe
fingerprint. The first occurrence is attempt 1; the only equivalent automatic
retry is attempt 2 and must become `blocked`; a third occurrence is rejected
across current and archived cycles.

V7 material bindings are:

- lifecycle patch SHA-256:
  `2db1615df7bc1c5b4bd0d2d17cecb22a43b2bf4be72a1ebcf750820170b5ff66`;
- full six-file patched-source diff SHA-256 after all five patches:
  `70dfa9e28546dc0c6dbe8046f5960577514b0bc6793fda98b3383931a50a72d8`;
- patched-source `go.mod` SHA-256:
  `82794b69209f2d2e8ad23fccc94a84d07ac46fc99040964a89ff5566e42c8044`;
- patched-source `go.sum` SHA-256:
  `ad753874d566d22c81da097ed3d8d59f2f17ff6e69a437aca914ad178a488efb`;
- official Go 1.26.5 Bookworm image index:
  `sha256:53eeac89074db483fdf0ab3be1df32bf6e47562263d2d0d6baa7f26acb4957dd`;
- exact Linux/arm64 builder manifest:
  `sha256:b1a0cc29a7e13e0595e21087eeb930dc494976b18ba68279bf52c665f3170aa0`;
- builder `/usr/local/go/bin/go` SHA-256:
  `22201b57b855105df064a291863c3fc04f22a7431187a9205122aff42a0c825b`;
- twice-reproduced Linux/arm64 CGO binary SHA-256:
  `bb8dd437802943670b4e882a3cdc30d5ea5a3b2035171fb765d7d82db7f624de`.

Two independently cloned pinned Beads sources were patched in this exact
order, always with `git apply --unidiff-zero`: atomic claim, effective database
security, hook isolation, gateway claim, then lifecycle. Their full source
diff hashes matched. Each used a different empty module cache, build cache,
and host source path, all mounted at fixed container paths. Dependencies were
downloaded through the Go checksum database, then `go mod verify` and the build
ran with networking disabled. The normalized build operation was:

```text
cwd: public scratch root
builder: docker.io/library/golang@sha256:b1a0cc29a7e13e0595e21087eeb930dc494976b18ba68279bf52c665f3170aa0
platform: linux/arm64
source mount: <independent-patched-source> -> /src (read-only)
cache mount: <independent-empty-cache> -> /cache

go mod download
docker --network=none: go mod verify
docker --network=none environment:
  HOME=/tmp/mars3-home GOTOOLCHAIN=local GOFLAGS=-mod=readonly
  GOMODCACHE=/cache/modcache GOCACHE=/cache/gocache
  CGO_ENABLED=1 GOOS=linux GOARCH=arm64 GOARM64=v8.0
  SOURCE_DATE_EPOCH=1787852400 TZ=UTC LANG=C.UTF-8 LC_ALL=C.UTF-8
docker --network=none command:
  go build -tags gms_pure_go -trimpath -buildvcs=false
  -ldflags=-buildid= -o /cache/out/bd ./cmd/bd
```

Both sequential cold builds produced byte-identical binaries at the bound
hash. The second binary passed the repository's public synthetic
`TestNativeMutatorIntegration` inside the same pinned, network-disabled Linux
builder without a skip. The exact patched source ran every selected native
`AuthorityLifecycle` case with Docker available and without a skip, including
direct completion, rework history, rollback/contention, hook denial,
server-transaction denial, all seven noncompleted statuses, type-specific and
raw claim corruption, current and archived claim-attempt mismatch, legacy
shadow contradiction, missing fingerprints, bounded blocked retry, and third
attempt denial. The server case used the existing digest-pinned Dolt and Ryuk
images.

The credential-free PostgreSQL suite again ran against an ephemeral,
loopback-only PostgreSQL 17.11 container at the previously recorded digest.
`TestPostgresLeaseLifecycleAndRestart` executed without a skip and passed; the
container was stopped and auto-removed. This verification did not mutate the
canonical Beads workspace or create a canonical live lease. M3-W001 remains
`in-progress` at WorkVersion mutation/dependency revisions `1/1`, and M3-P001
remains `backlog` pending the immutable v7 review chain.

## Lifecycle-correction v7 review disposition

Independent QA and Security reviewed public PR 10 exact head
`36d8c981ebde65e694416caf16fc02d50aac2a67`, tree
`be55454779c2c0dd08adc08666c2b7ee3826448f`, signed tag object
`217585d45ec414f55e5d326419a4f79b96a48915`, and successful bounded rerun/job
`33091727157/98586729802`. Both verdicts were `changes-requested`; the immutable
v7 commit, tag, run, and dispositions remain evidence, M3-W001 remains
`in-progress`, and no canonical live lease exists.

The exact additive findings are:

- `lifecycle.claim_key_alias_not_canonical`: case-folded aliases such as
  `workclaim` could bypass exact raw-key checks and overwrite the canonical
  typed WorkClaim during Go JSON decoding;
- `lifecycle.legacy_active_scalar_contradiction`: active metadata without
  detailed lifecycle evidence could retain terminal legacy accepted,
  completed, or reconciled values;
- `lifecycle.dependency_detailed_state_ignored`: dependency readiness trusted
  legacy completion scalars while ignoring a contradictory detailed failed
  run.

The prospective signed `W-001-lifecycle-correction-v8` grant is based on exact
v7 head `36d8c981ebde65e694416caf16fc02d50aac2a67` and tree
`be55454779c2c0dd08adc08666c2b7ee3826448f`. It permits only canonical-key
admission, legacy/detailed lifecycle consistency, dependency-readiness
derivation, native parity, qualification, public evidence, and fresh immutable
review. It grants no canonical Beads mutation, live lease, merge, production
effect, or trust expansion.

## Lifecycle-correction v8 candidate

The v8 adapter rejects every noncanonical metadata key before typed decoding,
including top-level and nested case-fold aliases. A record without detailed
lifecycle evidence can no longer carry legacy accepted, completed, reconciled,
or blocker state. Dependency projections decode the full canonical metadata;
when detailed lifecycle records exist, they must pass the same lifecycle
validator and agree with their legacy scalars before readiness can project.
The pinned native transaction path rejects unknown top-level authority keys and
the exact case-fold claim reproduction, while its existing detailed lifecycle
preimage validation rejects the orphan legacy terminal state.

V8 material bindings are:

- lifecycle patch SHA-256:
  `116c3b59744f1d6c3065ef8baf89d2bfac372bab66282b8cd9443e0843fc65c5`;
- full six-file patched-source diff SHA-256 after all five patches:
  `5fb4120f30c9d54d4dd847755a8070d305c1a7a14b783e7ce33157b432b02665`;
- unchanged patched-source `go.mod` SHA-256:
  `82794b69209f2d2e8ad23fccc94a84d07ac46fc99040964a89ff5566e42c8044`;
- unchanged patched-source `go.sum` SHA-256:
  `ad753874d566d22c81da097ed3d8d59f2f17ff6e69a437aca914ad178a488efb`;
- exact Linux/arm64 builder manifest:
  `sha256:b1a0cc29a7e13e0595e21087eeb930dc494976b18ba68279bf52c665f3170aa0`;
- twice-reproduced Linux/arm64 patched binary SHA-256:
  `a478f5090ca1b616e5aa8e5b74f4277814a8f0b1a88d990f9b7876761a3a7cc7`.

The two builds used different host source trees and fresh v8 Go build caches at
the fixed `/src` and `/cache` container paths. Because v8 did not change the
bound `go.mod` or `go.sum`, each build mounted one of the two independently
downloaded and verified v7 module caches read-only at `/cache/modcache`. The
fresh module-download attempt was stopped after Docker networking stalled with
no CPU or cache progress; the local Docker daemon was then fully restarted.
Both retained module graphs passed `go mod verify` with networking disabled,
and both builds used the same v7 normalized `go build` command and environment.
The outputs were byte-identical.

The exact patched source executed every selected native `AuthorityLifecycle`
case with Docker available and no skip. This included embedded completion,
rework, rollback, contention, hook denial, server-transaction denial, all
noncompleted outcomes, canonical-claim history, fingerprint retry limits,
case-fold claim alias rejection, and orphan legacy terminal-scalar rejection.
The reproduced binary then passed `TestNativeMutatorIntegration` in the pinned
network-disabled Linux builder without a skip.

The PostgreSQL lifecycle/restart suite executed without a skip against an
ephemeral loopback-only PostgreSQL 17.11 container at digest
`sha256:051f7b7b3abdd564d5d1bd1e8c4b9c1b6e77087d1dd22020ede611c096a272e0`.
It passed and the container was stopped and auto-removed. No canonical Beads
metadata or live lease was created; W-001 remains `in-progress` pending fresh
immutable v8 QA and Security review.

## Lifecycle-correction v8 review disposition

Independent QA and Security changes-requested exact v8 head
`6d6b90ef495cd64286e755e90d199a3cb622cd54`, tree
`f596e2a148f055bcac90960419b2e22928bd471c`, signed tag object
`fb99ef24abb1176e7bcec01bddffe305979d8464`, and successful run/job
`33097381660/98606506699`. The immutable v8 bytes remain historical evidence.
The two normalized findings were:

- `lifecycle.native_recursive_key_alias_not_canonical`: the native transition
  rejected unknown top-level keys but admitted noncanonical case-folded keys
  inside WorkVersion, claims, handoffs, reviews, runs, failures,
  reconciliation, terminal evidence, and archived cycles;
- `lifecycle.dependency_lineage_stripping`: a dependency with WorkVersion,
  claim, or raw detailed-lifecycle keys could omit or empty/null its detailed
  proof and project ready through the sparse legacy compatibility path.

Neither review performed a canonical lifecycle or lease effect. M3-W001
remained `in-progress`, and the signed `W-001-lifecycle-correction-v9` grant
prospectively bounded only these corrections, qualification, public evidence,
and fresh immutable review.

## Lifecycle-correction v9 candidate

The v9 project adapter treats WorkVersion, either claim variant, or any raw
detailed-lifecycle key as an irreversible detailed-proof boundary for
dependency projection. Even omitted detailed records after a retained version
or claim, and explicitly empty or null detailed fields, fail closed. Sparse
The sparse legacy dependency compatibility path is available only to an unversioned,
unclaimed record with no detailed key.

The v9 native patch performs recursive canonical-key rejection before semantic
parsing across WorkVersion, WorkClaim, BootstrapClaim, handoff, review,
failure, run, reconciliation, terminal, and current and archived history
objects. Its adversarial corpus changes one nested key at a time and proves
every case-fold alias is rejected. The first callback-oriented implementation
encountered bounded linker output-capacity failures. It was replaced with
explicit object traversal without changing the schema contract; the same
recursive corpus passed before qualification resumed.

V9 material bindings are:

- v8 base head/tree:
  `6d6b90ef495cd64286e755e90d199a3cb622cd54` /
  `f596e2a148f055bcac90960419b2e22928bd471c`;
- lifecycle patch SHA-256:
  `6cca8ab8bd5bd0d5f179612ece7e68e002caa69c455c80cdb00335d5e75a31c4`;
- full six-file patched-source diff SHA-256 after all five patches:
  `91b3e8dd5c8c0c01b5953c4c38ca508a150b05cd719f4e80fec293365afddf7f`;
- unchanged patched-source `go.mod` SHA-256:
  `82794b69209f2d2e8ad23fccc94a84d07ac46fc99040964a89ff5566e42c8044`;
- unchanged patched-source `go.sum` SHA-256:
  `ad753874d566d22c81da097ed3d8d59f2f17ff6e69a437aca914ad178a488efb`;
- exact Linux/arm64 builder manifest:
  `sha256:b1a0cc29a7e13e0595e21087eeb930dc494976b18ba68279bf52c665f3170aa0`;
- twice-reproduced Linux/arm64 patched binary SHA-256:
  `d72ab6b406a62930083cb9801d74336ea10fd7e871453c19f935252a77dccb18`.

Two independently composed pinned Beads source trees applied the five patches
in the documented order with `git apply --unidiff-zero`. Each used a distinct,
previously independently populated module cache that passed network-disabled
`go mod verify`, a fresh build cache, and the v7 normalized builder command.
Both outputs were byte-identical.

The exact patched source executed every selected bootstrap, authority-claim,
and authority-lifecycle Docker/Dolt case with Docker available and no skip.
This included atomicity, rollback, contention, redirect and selector denial,
hook and server-transaction denial, every noncompleted outcome, bounded
equivalent-failure retry, claim lineage, legacy-scalar denial, and the full
recursive canonical-key corpus. The reproduced binary passed
`TestNativeMutatorIntegration` inside the pinned network-disabled builder
without a skip.

The PostgreSQL lifecycle/restart suite executed without a skip against an
ephemeral loopback-only PostgreSQL 17.11 container at digest
`sha256:051f7b7b3abdd564d5d1bd1e8c4b9c1b6e77087d1dd22020ede611c096a272e0`.
It passed and the container was stopped and auto-removed. The candidate makes
no canonical Beads or live-lease effect. A fresh sanitized, read-only canonical
projection showed W-001 still `in_progress` / `in-progress` with generation
`6e79ff81-a007-42a5-a178-7ce58dbb718b`, incarnation
`e1e8d2d3f80871096a568fb489f49575a42abd37b269df9faf777a09cd689b41`,
mutation sequence `1`, and dependency revision `1`; P-001 remained
`open` / `backlog` without WorkVersion. W-001 remains pending fresh immutable
v9 QA and Security review.

## Lifecycle-correction v9 public-CI disposition and v10 stabilization

The immutable v9 candidate is head
`ad845ff81f1e64b9e4110162a77a65a844891731`, tree
`e4a08e5a4b211003dc29609a0128856eec306061`, signed tag object
`47933c4957b9af2e8d7a38f971d7a20c5de8122f`. Public run `33104553091`
admitted doctrine, plan, DocSync, and immediate-public-disclosure checks on
both attempts. Attempt-one job `98630789458` and attempt-two job
`98631170195` each failed only during doctrine test cleanup after authority
packages passed.

The normalized fingerprint is `ci/doctrine-tempdir-git-pack-cleanup`: Git pack
activity outlived a disposable `t.TempDir` repository, so Go cleanup observed
a nonempty pack directory. The bounded retry was used once; the run therefore
exhausted two identical retries and was not retried a third time. QA and
Security were not asked to accept a CI-red checkpoint.

The prospective signed `W-001-lifecycle-ci-stabilization-v10` grant preserves
the complete v9 commit, tree, tag, runtime, native patches, product contracts,
and qualification hashes. It changes only the test Git wrapper so every
disposable command receives command-local `maintenance.auto=false`,
`gc.auto=0`, `gc.autoDetach=false`, and `maintenance.autoDetach=false`, plus a
regression that reads those exact effective values. No user, system, global,
production, or repository-persistent Git configuration is changed.
Accordingly, no authority runtime bytes changed in V10. It grants no Beads
mutation, live lease, merge, production effect, or trust expansion.

## Lifecycle-correction v10 review and v11 Git fencing

The immutable V10 candidate is head
`47b19b2c89d72fbf9eb5356ceefe33783d691aa4`, tree
`0ebe496c48871b040a7fcd7a286073f2c1d40153`, signed tag object
`84672df5f046995bb7efd79cf8f9a333946aecfa`. Public run `33105792480`, job
`98635155160`, passed every required gate. Independent QA and Security both
returned `changes-requested` on that exact subject while confirming the V9
runtime, native patches, product contracts, and qualification bytes remained
unchanged.

The two findings are additive test-fixture defects:

- `ci.test_git_sanitization_incomplete`: a raw pre-repository `git clone`
  bypassed the sanitized bounded wrapper and could inherit hostile global Git
  configuration;
- `ci.test_git_configuration_persisted`: the fixture helper wrote
  `maintenance.auto` and `gc.auto` into repository-local configuration despite
  V10's command-local-only claim.

The prospective signed `W-001-lifecycle-ci-fencing-v11` grant preserves the
complete V9/V10 history and permits only routing the clone and every other
disposable Git operation through the single sanitized wrapper, removing all
persistent fixture configuration, and proving the four effective maintenance
fences remain command-local while local and global persistent values are
absent. A source-level regression permits one raw `exec.Command("git", ...)`
site only: the audited wrapper itself. No authority runtime, native patch,
database schema, API, product contract, workflow, scanner, repository setting,
Beads state, or live lease is changed.

## Lifecycle-correction v11 review and v12 execution hardening

The immutable V11 candidate is head
`54f4593b1730ff9ae04a2e5cce0589c6baedfee6`, tree
`44ba564be30e0db0aa735d76539c3604a5d79e3f`, signed tag object
`7313ee2e38dd1d4f4f5ca62237e0be89b0b4f13a`. Public run `33108126981`, job
`98643418071`, passed every required gate. QA accepted that exact subject;
Security returned `changes-requested` after reproducing three additional
test-process boundary failures:

- `ci.test_git_fences_caller_overridable`: later caller `-c` values could
  override all four command-local maintenance fences and protected config
  mutations remained callable;
- `ci.test_git_environment_execution_injection`: inherited `GIT_EXEC_PATH`
  and `GIT_TEMPLATE_DIR` could execute a hostile upload-pack helper and a
  template-installed post-checkout hook;
- `ci.test_process_guard_fail_open`: the one-file literal counter did not
  detect aliases, concatenated command names, `CommandContext`, shells, or
  process calls in another doctrine test file.

The prospective signed `W-001-lifecycle-ci-hardening-v12` grant preserves the
complete V9-V11 runtime and publication history. The bounded correction pins
test Git execution to `/usr/bin/git`, strips every inherited `GIT_*` variable
before adding the exact safe set, admits only the synthetic identity `-c`
values needed by fixtures, rejects protected config/env/template/upload-pack
and repository-redirection options, and replaces the literal counter with a
repository-wide Go AST allowlist. The allowlist admits exactly the two
synthetic `ssh-keygen` calls and one trusted Git-wrapper call; adversarial
fixtures cover aliases, concatenation, `CommandContext`, shells, indirect
calls, `os.StartProcess`, hostile exec paths, and hostile templates. No
authority runtime, native patch, database schema, API, product contract,
workflow, scanner, repository setting, Beads state, or live lease is changed.

## Lifecycle-correction v12 review and v13 closed admission

The immutable V12 candidate is head
`3c8d55aa39e4e099d8a922f8e13a71efcbe2c78b`, tree
`c4bb80ab477b7fcbe73a7a237479e44703393952`, signed tag object
`d0176029978e0c49d795a02ad36f7f7992c3bdfa`. Public run `33110339883`, job
`98651204635`, passed every required gate. Independent QA and Security both
returned `changes-requested` for that exact subject:

- `ci.test_git_argv_schema_fail_open`: Git long-option abbreviations
  (`--upload-p`, `--templ`, and `--conf`), compact `-u/path`,
  `--separate-git-dir`, and config-producing `remote add` escaped denylist
  admission and could execute a helper, write outside the fixture, or persist
  repository configuration;
- `ci.test_process_guard_incomplete`: direct `exec.Cmd` construction,
  indirect syscall process function values, and nested doctrine test files
  escaped the nonrecursive selector guard.

The prospective signed `W-001-lifecycle-ci-hardening-v13` grant preserves all
V9-V12 runtime, qualification, commit, tag, run, and disposition evidence. It
replaces Git option denylists with exact per-subcommand argv schemas, rejects
every unenumerated subcommand or option before execution, recursively walks
the doctrine test tree, and admits only the two exact synthetic `ssh-keygen`
calls plus the one `/usr/bin/git` wrapper construction. Regressions retain the
earlier hostile environment, alias, concatenation, `CommandContext`, shell,
and persistence cases and add the exact V12 bypasses. No authority runtime,
native patch, database schema, API, product contract, workflow, scanner,
repository setting, Beads state, canonical lifecycle, or live lease is
changed.

## Lifecycle-correction v13 review and v14 physical boundary

The immutable V13 candidate is head
`ce934054aed66c074e99a032191a6a51c620b947`, tree
`73cab7fb7b1bd2fc1102dc4b16e9617fd7c26680`, signed tag object
`8d50fbb230503f4ad24cfc7301e4e4924be30ec0`. Public run `33112938711`, job
`98660186954`, passed every required gate. Independent QA and Security both
returned `changes-requested` for that exact subject:

- `ci.test_git_clone_physical_escape`: lexical `filepath.Rel` containment
  admitted `root/link/repo` when `root/link` was a symlink to an outside
  directory, and Git wrote the clone outside the disposable boundary;
- `ci.test_process_guard_transitive_bypass`: doctrine tests called the
  production `planningGrantGitOutput` executor directly, so the recursive
  constructor guard observed no test-side `os/exec` use while arbitrary Git
  arguments still executed.

The prospective signed `W-001-lifecycle-ci-hardening-v14` grant preserves all
V9-V13 runtime, qualification, commit, tag, run, and disposition evidence. It
requires a direct canonical disposable root, admits only a nonexistent direct
child clone target whose existing parent resolves to that root, and reserves
the target as a direct directory before Git starts. It routes the three
historical read-only tag and plan calls through the exact test wrapper and
rejects `planningGrantGitOutput` identifiers in every recursively enumerated
doctrine test file. Regressions cover symlinked roots, target ancestors,
existing target symlinks, and an alternate-file production-executor call. No
authority runtime, native patch, database schema, API, product contract,
workflow, scanner, repository setting, Beads state, canonical lifecycle, or
live lease is changed.

### V14 implementation checkpoint

The signed implementation checkpoint is commit
`80184b070fcf339e9880486ee330f7f69c80c0ae`, tree
`a87f9d1c69c20d55ae961a45edf1970590b5ca37`, with sole parent the immutable
V13 head. Its seven changed paths are exactly the signed V14 grant scope; the
diff is empty for `api/**`, `internal/authority/**`, `go.mod`, and `go.sum`.
The commit has a valid pinned ED25519 Git signature.

From a normal disposable clone of that exact commit, all of these commands
passed with repository-relative working directories:

```text
mars3 doctrine check --repo .
mars3 plan check --repo .
mars3 docsync audit --repo .
mars3 public-check --repo .
go test ./... -count=1
go vet ./...
git diff --check
git show --check --oneline --no-renames HEAD
```

The V14 Git/path/process and tag-admission suite passed ten consecutive runs,
and the same focused suite passed under the Go race detector. The
digest-pinned, network-disabled Gitleaks image
`docker.io/zricethezav/gitleaks@sha256:75bdb2b2f4db213cde0b8295f13a88d6b333091bbfbf3012a4e083d00d31caba`
detected the synthetic canary, then found no leak in the exact worktree or all
89 commits of public history. This checkpoint is implementation evidence, not
QA or Security acceptance and not merge, lease, or lifecycle authority.

## Lifecycle-correction v14 review and v15 descriptor binding

The immutable V14 candidate is head
`d631bec4ed786116c13e36995722d91d48d64109`, tree
`b9467f12b2031c5159ef749938bbd4f475eb6153`, signed tag object
`f97b9ebd0150ee4d75bf691c16c176b792d42461`. Public run `33123061855`, job
`98694494697`, passed every required gate. Independent QA and Security both
returned `changes-requested` for that exact subject:

- `ci.test_git_root_ancestor_alias_admitted`: the final root component was
  checked, but a real directory reached through a symlinked ancestor remained
  admitted;
- `ci.test_git_clone_reservation_toctou`: clone target reservation and
  pathname verification ended before Git consumed the writable destination;
- `ci.test_process_guard_dot_import_bypass`: dot-imported `os` and `syscall`
  exposed bare process entrypoints outside selector-only admission.

The signed `W-001-lifecycle-ci-hardening-v15` grant binds V14 as base, preserves
the V9 runtime and native-patch qualification bytes and all V10-V14 history,
and permits only the bounded test correction and fresh qualification. The V15
candidate removes `clone` from the exact Git argv schema and fixtures, prepares
repositories with direct directory creation plus the existing closed
`init`/`fetch`/`checkout` wrapper, rejects roots that differ from their resolved
physical path, and holds a verified directory descriptor through child
execution. A test-only descriptor helper changes directory through that handle
before launching `/usr/bin/git`, so deterministic root or ancestor replacement
cannot redirect Git to the replacement pathname. The recursive process guard
now rejects dot or blank imports of `os`, `os/exec`, and `syscall`; adversarial
fixtures cover the bare dot-imported and blank-import cases.

Focused local regressions passed for descriptor binding, root and ancestor
replacement, symlinked ancestry, clone-schema denial, hostile environment
fencing, historical pull-request synthesis, and recursive dot/blank process
imports. The focused suite passed ten consecutive runs and the Go race
detector. `go vet ./...`, the complete doctrine suite excluding only the known
linked-worktree public-root shape, and rebuilt doctrine, plan, and doc-sync
validators all passed.

Two fresh pinned Beads clones at commit
`6c124203e771433a3550c348771a5b5e27fd3c21` applied the unchanged five patches
in repository order with `git apply --unidiff-zero`. Both full six-file source
diffs matched SHA-256
`91b3e8dd5c8c0c01b5953c4c38ca508a150b05cd719f4e80fec293365afddf7f`;
their `go.mod` and `go.sum` hashes remained
`82794b69209f2d2e8ad23fccc94a84d07ac46fc99040964a89ff5566e42c8044`
and `ad753874d566d22c81da097ed3d8d59f2f17ff6e69a437aca914ad178a488efb`.
Two independently populated module caches passed `go mod verify` with
networking disabled. Two fresh build caches in the exact pinned Linux/arm64
builder produced byte-identical binaries at the V9-bound SHA-256
`d72ab6b406a62930083cb9801d74336ea10fd7e871453c19f935252a77dccb18`;
the builder Go executable remained
`22201b57b855105df064a291863c3fc04f22a7431187a9205122aff42a0c825b`.

An initial containerized native-test compile omitted the required
`gms_pure_go` tag and failed closed on absent ICU headers. The corrected
container route then exposed that the Beads readiness probe requires the
Docker CLI as well as the mounted socket and skipped one server case; that run
was rejected. The final Go 1.26.2 host runner used the independently verified
module cache with `GOPROXY=off`, the exact `gms_pure_go` tag, and the cached
digest-pinned Dolt/Ryuk images. Every selected bootstrap-claim,
authority-claim, and authority-lifecycle case passed without a skip, including
server-transaction denial, rollback/contention, all seven nonterminal
outcomes, claim lineage, bounded retry, and recursive canonical-key rejection.
The Dolt container was terminated. The exact reproduced Linux artifact then
passed MARS-3 `TestNativeMutatorIntegration` inside the pinned builder with
networking disabled.

The credential-free PostgreSQL suite passed
`TestPostgresLeaseLifecycleAndRestart` without a skip against an ephemeral
loopback-only PostgreSQL 17.11 container at digest
`sha256:051f7b7b3abdd564d5d1bd1e8c4b9c1b6e77087d1dd22020ede611c096a272e0`.
The container was stopped and auto-removed. This remains implementation
evidence pending the signed checkpoint/tag, normal-clone public and scanner
gates, public run, and independent QA and Security review. It makes no
authority-runtime, native-patch, Beads, lease, merge, repository-setting, or
production effect.

### V15 implementation checkpoint

The signed implementation checkpoint is commit
`dbd55e1ad1b39fee3267a33a54d3306dbca660c5`, tree
`83eb5e68413c6668266554c93693abc95f683c25`, with sole parent the immutable
V14 head. Its seven changed paths are exactly the signed V15 grant scope; the
diff remains empty for `api/**`, `internal/authority/**`, `database/**`,
`.github/**`, `go.mod`, and `go.sum`. The commit has a valid pinned ED25519 Git
signature.

From a normal disposable clone of that exact commit, all of these commands
passed with repository-relative working directories:

```text
mars3 doctrine check --repo .
mars3 plan check --repo .
mars3 docsync audit --repo .
mars3 public-check --repo .
go test ./... -count=1
go vet ./...
git diff --check
git show --check --oneline --no-renames HEAD
git verify-commit HEAD
```

The all-package run included the complete doctrine suite in 232.916 seconds.
The digest-pinned, network-disabled Gitleaks image first detected one synthetic
canary, then found no leak in the exact V15 worktree or all 91 commits of
public history. This checkpoint and its qualification remain implementation
evidence, not QA or Security acceptance, merge authority, canonical lifecycle
authority, or lease authority.

### V15 immutable review and V16 closed launcher

The signed V15 evidence commit is
`a46f16deff2fc06c5d0d21377a3bb2c65e873fc9`, tree
`c2e482717f182040708cbf2551ee266de2485a30`. Signed annotated tag object
`71230aed1661a987dbd1b63b058180a6b33f7825` targets that exact commit and
tree. Public run `33165311496`, job `98829194619`, passed every required gate.
Independent QA and Security both returned `changes-requested` on that immutable
subject despite the green gates:

- `ci.test_git_descriptor_helper_transitive_bypass`: a same-package test could
  call `runPlanningGrantTestGitDescriptorHelper` directly with ambient
  environment and descriptor state, bypassing the wrapper's root-admission
  provenance;
- `ci.test_git_helper_executable_path_toctou`: the wrapper resolved
  `os.Executable`, invoked its deterministic mutation hook, and then executed
  the earlier unpinned pathname, while the AST allowlist represented every
  dynamic executable expression identically.

Both reviewers confirmed that V15 closed all three V14 findings. Their fresh
normal-clone validation also passed the four validators, full package tests,
focused ten-repeat and race suites, vet, whitespace, signatures, topology,
seven-path scope, exact tag/run binding, and independent reproducible builds.
The V9 native artifact remained `d72ab6b406a62930083cb9801d74336ea10fd7e871453c19f935252a77dccb18`;
its selected native suites and project integration passed without skips. The
ephemeral digest-pinned PostgreSQL 17.11 lifecycle suite also passed without a
skip and removed its container. These passing results do not override either
source-boundary finding.

The signed `W-001-lifecycle-ci-hardening-v16` grant binds the exact V15 head,
tree, tag object, public run, and both changes-requested dispositions. Its grant
SHA-256 is `95fa2caa2befd270ed15f9c317a37ceec442b70c0826a869323d34c3d612d835`;
its detached-signature SHA-256 is
`26711612175f7969ec168159e535e7b6b7273641690ed4c486e609cddaa844e5`.
The signature verifies under namespace
`mars3-w001-lifecycle-ci-hardening-v16` with the pinned ED25519 fingerprint.
It authorizes exactly seven paths and no merge, canonical Beads mutation, live
lease, production effect, workflow or repository-control change.

The V16 implementation removes the environment-selected `TestMain` mode,
`os.Executable`, and the directly callable descriptor helper. The command
constructor now validates its closed argv and canonical local fetch source,
opens and verifies the root descriptor at invocation admission, and captures
that handle inside a one-shot closure. Execution invokes literal
`/usr/bin/perl` with one byte-exact, non-input program: it aliases inherited fd
3, changes directory through that descriptor, closes the descriptor, and uses
list-form `exec` of literal `/usr/bin/git` with only the already admitted argv.
The launcher receives a fixed environment with only file transport admitted;
ambient Git, Perl, loader, editor, shell, and executable-path inputs are absent.
The recursive AST gate now rejects every dynamic process executable and every
reference to the removed helper, and adversarial fixtures cover both cases plus
network fetch sources. Root or ancestor replacement after command admission
therefore cannot redirect the Git process.

The exact pre-commit V16 bytes passed the focused grant, descriptor-admission,
root/ancestor replacement, one-shot contention, environment, fetch, and
recursive process suite ten consecutive times in 27.537 seconds. The same
boundary suite passed under the Go race detector in 5.336 seconds. The complete
project suite passed with only the known linked-worktree `.git` public-root
subtest excluded; `internal/doctrine` completed in 87.028 seconds. `go vet
./...`, `git diff --check`, and rebuilt doctrine, plan, and doc-sync validators
passed. Public-check remains intentionally deferred to a normal clone because
the delivery worktree represents `.git` as a link file outside the public
source roots.

Two new independent local clones of pinned Beads commit
`6c124203e771433a3550c348771a5b5e27fd3c21` applied the five unchanged patches
in their documented order. Both six-file source diffs matched
`91b3e8dd5c8c0c01b5953c4c38ca508a150b05cd719f4e80fec293365afddf7f`;
their `go.mod` and `go.sum` hashes remained
`82794b69209f2d2e8ad23fccc94a84d07ac46fc99040964a89ff5566e42c8044`
and `ad753874d566d22c81da097ed3d8d59f2f17ff6e69a437aca914ad178a488efb`.
Distinct module and fresh build caches passed `go mod verify` with networking
disabled in the pinned Linux/arm64 builder. Both outputs were byte-identical at
`d72ab6b406a62930083cb9801d74336ea10fd7e871453c19f935252a77dccb18`.
The selected bootstrap, authority-claim, and lifecycle suite passed every case
without a skip against the digest-pinned Dolt container in 52.684 seconds and
removed its containers. That exact Linux artifact passed MARS-3
`TestNativeMutatorIntegration` inside the network-disabled pinned builder in
12.004 seconds.

The non-skipped `TestPostgresLeaseLifecycleAndRestart` suite passed against an
ephemeral loopback-only PostgreSQL 17.11 container at digest
`sha256:051f7b7b3abdd564d5d1bd1e8c4b9c1b6e77087d1dd22020ede611c096a272e0`.
The test itself completed in 0.61 seconds; the container was stopped and
auto-removed. The digest-pinned, network-disabled Gitleaks image detected the
existing synthetic canary with exit 42 and found no leak in the exact V16
worktree. No canonical Beads workspace, live lease, production service, or
repository setting was read or mutated by these qualifications.

The signed V16 implementation checkpoint is commit
`3cc199b4e165780067f1891d193ff1a54eadfe23`, tree
`8846e5792f87c3147daee10ba21a7cc0c3b6071e`, with sole parent the exact V15
head. Its diff contains exactly the seven V16-authorized paths and remains
empty for `api/**`, `internal/authority/**`, `database/**`, `.github/**`,
`go.mod`, and `go.sum`. The commit signature verifies as
`engineer@example.com` with the pinned ED25519 fingerprint.

A first disposable normal clone was intentionally rejected after its exact
commit was checked out with detached `HEAD`; the lifecycle validator admits
only the signed review branch or accepted `main`. Switching that same exact
clone to local branch `codex/w-001-lifecycle-completion` preserved the commit
and tree and made the checkout contract truthful. The unmodified
`go test ./... -count=1` then passed every package; `internal/doctrine`
completed in 99.225 seconds. `go vet ./...`, all four rebuilt validators,
whitespace, show-check, commit-signature verification, and a clean checkout
also passed. The digest-pinned, network-disabled Gitleaks image found no leak in
the exact implementation worktree or all 93 commits of its public history.
This is qualified implementation evidence pending the final evidence commit,
signed V16 tag, exact-head public run, and fresh independent QA and Security
review. It grants no merge, canonical lifecycle, live lease, or production
authority.

### V16 immutable review and V17 descriptor-stream correction

The signed V16 evidence checkpoint is
`25d2f14e20e74f1415caa4118a93c359f9370031`, tree
`d9bf0e3f89807c12c5be5a58ea68fd04715aa740`. Signed annotated tag object
`125a596c00a5a00f40fbda002f38cc06e3f0b5cb` targets that exact commit and
tree. Public run `33206197037`, job `98967743138`, passed every required gate.
Independent QA returned `accepted`; independent Security returned
`changes-requested` on the same immutable subject:

- `ci.test_git_invocation_one_shot_field_bypass`: copied invocation values or
  the extracted `combinedOutput` field could invoke the captured executor
  outside the wrapper mutex and nil transition;
- `ci.test_git_fetch_source_toctou`: fetch validated one canonical source
  pathname before the deterministic process hook but did not bind that source
  to a descriptor before Git consumed it;
- `ci.test_process_guard_refresh_executor_bypass`: same-package tests could
  call the ambient arbitrary-argv production `gitOutput` executor because the
  recursive guard named only the planning-grant executor.

Both reviewers confirmed that V16 closed the V15 self-executable helper and
mutable Go test-image findings. Fresh normal-clone validators, the full suite,
the focused race suite, vet, whitespace, signatures, topology, seven-path
scope, exact tag/run binding, native qualification, PostgreSQL qualification,
and leak scans passed. Those passing results do not override the three
Security findings.

The signed `W-001-lifecycle-ci-hardening-v17` grant binds the exact V16 head,
tree, tag object, public run, QA acceptance, and Security changes-requested
disposition. Its grant SHA-256 is
`8898b65846209758f892c1b965c2d69bb13a6c1a97714f953b3f8735d96a1f7d`;
its detached-signature SHA-256 is
`f3c1fcc7ab46205f00d45029add526fa7b9df06b19aa7a65b7f08f12a7b87df6`.
The detached signature verifies under namespace
`mars3-w001-lifecycle-ci-hardening-v17` with pinned ED25519 fingerprint
`SHA256:i5VSHF257DhXJ5l/9oOUGHnT2mrqgXYSMryQHRsSBx8`. It authorizes exactly
seven paths and no merge, canonical Beads mutation, live lease, workflow,
repository-control, production, or trust change.

The V17 implementation moves the one-shot transition into mutex-protected
state captured by the executor closure, so every invocation copy and extracted
function value shares the same irreversible consumption state. Fetch opens and
verifies both source and destination directory descriptors before returning an
invocation. It resolves and packs the admitted revision while entered through
the source descriptor, imports that byte stream through the destination
descriptor, and updates only `FETCH_HEAD` plus an admitted identical tag
refspec. No child receives or reopens the admitted source pathname. Every child
still uses literal `/usr/bin/perl`, the byte-exact fd-3 trampoline, literal
`/usr/bin/git`, and the fixed zero-ambient environment.

The recursive process gate now inventories every direct production process
entrypoint from all regular non-test Go source, requires that inventory to
remain exactly `planningGrantGitOutput` and `gitOutput`, and rejects either
identifier from every doctrine test. It also rejects direct
`combinedOutput` field selectors outside the sole wrapper method, unexpected
production constructors, malformed production source, symlinked surfaces,
dynamic executables, indirect constructors, and dot or blank guarded imports.
Deterministic regressions replace the fetch source after command admission,
race copied invocation values, and inject both production executor names and a
direct executor-field access into the recursive fixture corpus.

The exact pre-commit V17 bytes passed the focused grant, shared one-shot,
descriptor-bound fetch, closed-argv/environment, and recursive process corpus
ten consecutive times in 29.775 seconds. The same boundary suite passed under
the Go race detector in 13.260 seconds. `go vet ./...`, `git diff --check`, and
rebuilt doctrine, plan, and doc-sync validators passed. The complete project
suite passed with only the known linked-worktree `.git` public-root subtest
excluded; `internal/doctrine` completed in 123.617 seconds. Public-check remains
deferred to a normal clone because this delivery worktree represents `.git` as
a link file outside the public source roots.

Two fresh local clones of pinned Beads commit
`6c124203e771433a3550c348771a5b5e27fd3c21` independently applied the same five
V9 patches in their documented order. Both ordinary binary diff
serializations matched the six-file SHA-256
`91b3e8dd5c8c0c01b5953c4c38ca508a150b05cd719f4e80fec293365afddf7f`;
their `go.mod` and `go.sum` hashes remained
`82794b69209f2d2e8ad23fccc94a84d07ac46fc99040964a89ff5566e42c8044`
and `ad753874d566d22c81da097ed3d8d59f2f17ff6e69a437aca914ad178a488efb`.
Two independently populated module stores passed `go mod verify` while mounted
read-only with networking disabled in the pinned Linux/arm64 builder at digest
`sha256:b1a0cc29a7e13e0595e21087eeb930dc494976b18ba68279bf52c665f3170aa0`.
Two fresh build caches then emitted byte-identical binaries at the V9-bound
SHA-256 `d72ab6b406a62930083cb9801d74336ea10fd7e871453c19f935252a77dccb18`.

The first copied bind-cache recovery failed closed when its offline verifier
found missing module metadata. After independent population, Docker Desktop
left the bind-mounted verification containers before execution in `created`
state. That same cache route was abandoned after the second failure. A bounded
named-volume route was used instead. Docker Desktop still could not start a
no-bind `go version` container; a graceful restart replaced only its Linux VM,
so the exact enumerated Docker Desktop processes were terminated and the app
was relaunched once. The daemon then reported healthy version `20.10.23`, and
all subsequent qualification completed. None of these rejected runs produced
qualification evidence or touched repository, Beads, lease, or production
state.

The Go 1.26.2 host runner used `GOPROXY=off`, the exact `gms_pure_go` tag, and
the cached digest-pinned Dolt/Ryuk images. Every selected bootstrap-claim,
authority-claim, and authority-lifecycle case passed without a skip in 55.928
seconds, including pre-effect server denial, rollback and contention, all seven
nonterminal outcomes, claim lineage, bounded retry, and recursive
canonical-key rejection. Dolt and Ryuk were terminated and removed. The exact
reproduced Linux artifact then passed MARS-3 `TestNativeMutatorIntegration` in
18.345 seconds inside the pinned builder with networking disabled.

`TestPostgresLeaseLifecycleAndRestart` passed without a skip in 0.67 seconds
against an ephemeral loopback-only PostgreSQL 17.11 container at digest
`sha256:051f7b7b3abdd564d5d1bd1e8c4b9c1b6e77087d1dd22020ede611c096a272e0`
using synthetic test-only roles. The container was stopped and auto-removed;
`docker ps -a` was empty after qualification. This remains implementation
evidence pending signed commits and tag, normal-clone public and leak gates,
exact-head CI, and fresh independent QA and Security review. It grants no
merge, canonical lifecycle mutation, live lease, or production authority.

The signed V17 implementation checkpoint is commit
`a3b9351b0070d99b4c210814113b5377c13ed574`, tree
`73754f9ca4bb173587eea0d45aee49b451acaa49`, with sole parent the exact V16
head. Its diff contains exactly the seven V17-authorized paths and remains
empty for `api/**`, `internal/authority/**`, `database/**`, `.github/**`,
`go.mod`, and `go.sum`. The commit signature verifies as
`engineer@example.com` with the pinned ED25519 fingerprint.

A first normal-clone validator pass was rejected because `--single-branch`
omitted historical annotated tags and branch-only scanner-history commits. The
same local disposable clone fetched all tags and all source branch refs without
changing its branch, head, tree, or worktree. Doctrine, plan, doc-sync, and
public-check then passed on the exact implementation commit. The unmodified
`go test ./... -count=1` passed every package; `internal/doctrine` completed in
140.152 seconds. `go vet ./...`, whitespace, show-check, commit-signature
verification, and clean-checkout gates passed.

The digest-pinned, network-disabled Gitleaks image at
`sha256:75bdb2b2f4db213cde0b8295f13a88d6b333091bbfbf3012a4e083d00d31caba`
detected one synthetic canary and returned nonzero, then found no leak in the
exact implementation worktree or all 95 fetched commits. This checkpoint is
qualified implementation evidence pending the signed evidence commit and V17
tag, exact-head public CI, and fresh independent QA and Security review. It is
not merge, Beads, lease, reconciliation, or production authority.

### V17 failed disposition and test-harness retirement authority

The immutable V17 subject is head
`0ed9482fea1bd22bf4198ff9d9223e004853212a`, tree
`e1c3179abf82bef70f56ee330735072b9ed8b510`, signed annotated tag object
`3767eab4895d567ece26ab42eef006926fb8dddd`. Exact-head public run
`33213446709`, job `98991727867`, failed. Independent QA and Security both
returned `changes-requested` against that exact subject.

The public failure is
`ci.public_gate_fetch_head_pseudoref_portability`: Ubuntu Git 2.55 refused the
fixture's manual `update-ref FETCH_HEAD` because `FETCH_HEAD` is a pseudoref
written by ordinary fetch. Security also recorded
`ci.test_git_gitdir_rebinding_bypass`,
`ci.test_process_guard_transitive_production_caller_bypass`,
`ci.test_git_local_config_process_bypass`, and
`ci.test_process_guard_top_level_funclit_bypass`. The shared one-shot state and
outer source/destination descriptor identity were correctly improved, but the
recurring same-class findings mean that passing focused tests cannot establish
the attempted in-package security boundary.

The signed `W-001-lifecycle-test-harness-retirement-v1` grant binds that exact
V17 commit, tree, tag object, failed run, and both review dispositions. Its
grant SHA-256 is
`4d2cc1ba4f537715283631e2b2f57a6098b522bc0135e3dc8bff4254aa5e19b3`;
its detached-signature SHA-256 is
`4e093b74ce43191da23d9e34d68c77bbe3428f38fee5c75612580e708ec06486`.
The detached signature verifies under namespace
`mars3-w001-lifecycle-test-harness-retirement-v1` with pinned ED25519
fingerprint `SHA256:i5VSHF257DhXJ5l/9oOUGHnT2mrqgXYSMryQHRsSBx8`. Signed grant
commit `adf068e9a329c8748357343dcb5976e317c5ec12` contains only the two new
grant paths and verifies with the same key.

The authorized retirement removes the descriptor/Perl trampoline, copied
one-shot executor, descriptor pack/index transfer, manual `FETCH_HEAD` update,
and AST process self-admission. The replacement is a transparent deterministic
Git fixture: literal `/usr/bin/git`, exact admitted argv, zero ambient Git
configuration, canonical local-only fetch sources, disabled hooks, and
disabled background maintenance. Ordinary `git fetch` creates `FETCH_HEAD`.
The accepted immutable read-only/no-credential CI workflow is recorded as the
candidate-code security boundary; the fixture runner is not described as a
sandbox or authority mechanism.

Before durable documentation sync, the exact focused Git fixture suite passed:
`go test ./internal/doctrine -run 'TestPlanningGrantTestGit' -count=1` in
1.682 seconds. The doctrine package compiled with the retirement validator and
the combined focused Git/plan-grant selection passed in 1.660 seconds. Full
qualification, signed candidate identity, retirement tag, exact-head CI, and
fresh ordered QA then Security review remain pending. This evidence grants no
merge, Beads/lease mutation, downstream-ticket start, or production effect.

The first rebuilt doctrine validator identified three ordinary integration
defects before candidate commit: a case-sensitive ADR marker, a removed-symbol
check that matched its own string literals, and the exact signed grant commit's
Git timestamp `2026-08-28T22:23:41Z` preceding the grant's rounded
`2026-08-28T22:24:00Z` issuance time by 19 seconds. The one authorized
ordinary correction removes the self-matching scan, uses the durable ADR
heading's exact case, and admits only commit
`adf068e9a329c8748357343dcb5976e317c5ec12` within a maximum 60-second
issuance skew. Every other retirement commit must remain at or after issuance.
This is a bounded timestamp/validator correction, not a change to the retired
harness design or to any prior signed object.

After that correction, rebuilt doctrine, plan, and doc-sync validators all
passed, and `git diff --check` passed. The focused deterministic Git and
planning-grant suite passed ten consecutive runs in 15.828 seconds; the same
suite passed under the race detector in 2.362 seconds. `go vet ./...` passed.
The unmodified `go test ./... -count=1` passed every authority package and
reached the known linked-worktree-only failure in
`TestRepositoryChecks/public`: public-check correctly rejects this worktree's
`.git` link file as outside the governed source roots. `internal/doctrine`
reported that sole failure after 133.825 seconds. No product, authority,
database, workflow, dependency, Beads, lease, or production byte was changed.
The identical public and full-suite gates therefore remain assigned to the
fresh normal-clone checkpoint after the implementation commit is signed.

The signed retirement implementation checkpoint is commit
`67d4a1d2aff15f4d4bdfe7d69ca6704cedf20cb5`, tree
`af3eafc6341599ce1478a3e874599596cb2e30bd`, with sole parent the signed grant
commit. Its V17-base diff is exactly the nine authorized paths. The diff is
empty for `api/**`, `internal/authority/**`, `database/**`, `.github/**`,
`go.mod`, and `go.sum`; the V9 authority runtime, native patches, product
contracts, workflow, and dependency bytes remain unchanged. Both retirement
commits verify with the pinned ED25519 key.

A fresh normal local clone with a real `.git` directory checked out the exact
implementation commit and tree on
`codex/w-001-lifecycle-completion`. Rebuilt doctrine, plan, doc-sync, and
public-check validators all passed. `go vet ./...`, `git diff --check`, commit
signature verification, and clean-checkout verification passed. The unmodified
`go test ./... -count=1` passed every package; `internal/doctrine` completed in
109.490 seconds. The public and complete-suite results close the source
worktree's linked-`.git` limitation without changing the candidate. Local
Gitleaks was unavailable; the unchanged exact workflow's canary, worktree, and
history scanner steps remain required at exact-head CI. This is qualified
implementation evidence pending the signed evidence checkpoint, retirement
tag, exact-head public run, and independent QA followed by Security review.

The first signed evidence checkpoint was commit
`ce9d0e7e82a33350263acd6199219a5349ef9bce`, tree
`3d6a1c44fa076d4755ec96c5755b45deae7a1ee5`. Signed annotated retirement tag
object `e08beb772696b078783d0c75d23c1029581cdeb1` targeted that exact commit and
tree, but its tagger was the synthetic Work Authority Engineer identity
`engineer@example.com`. Exact-head public run `33218389712`, job
`99007023124`, therefore failed at doctrine with
`public.w001_delivery_tag_identity`: the delivery validator requires the
synthetic Release Manager identity `release-manager@example.com` for review
tags. Later plan, public, test, whitespace, canary, and scanner steps were
skipped; the run is failure evidence, not acceptance evidence.

The user authorized one publication-only correction after that failure. It may
record this disposition, create one signed evidence commit, and replace only
`mars3/w001-lifecycle-test-harness-retirement-v1` with a signed annotated tag
using `MARS-3 Release Manager <release-manager@example.com>` and the exact
grant-bound message. It then permits branch/tag publication, fresh exact-head
CI, and independent QA followed by Security. It authorizes no source,
workflow, dependency, runtime, API, database, native-patch, product,
Beads/lease, merge, downstream-ticket, or production change.

### Retirement disposition and prospective authority recovery

The publication-only correction produced signed evidence commit
`0b0195ba0953d1d6d387aad699605ff864cfac1d`, tree
`f96506ca6b7b053d41d4067eb1c926b9d4b39b40`, and signed annotated retirement
tag object `60edcd3fd4603d66d7dd4feedbdae034fff13efa`. The tag uses the required
`MARS-3 Release Manager <release-manager@example.com>` identity and targets
that exact commit and tree. Exact-head public run `33222628400`, job
`99019716295`, passed doctrine, plan, public, full test, vet, whitespace,
signature, canary, worktree scan, and history scan gates. Independent QA
returned `accepted` against the same immutable subject.

Independent Security returned `changes-requested`. The retirement architecture
and deterministic Git fixture were accepted, but two signed-authority findings
prevent the successful run from becoming acceptance evidence:

- `authority.retirement_grant_retroactive_admission`: retirement grant commit
  `adf068e9a329c8748357343dcb5976e317c5ec12` was committed at
  `2026-08-28T22:23:41Z`, nineteen seconds before the signed grant's
  `2026-08-28T22:24:00Z` issue time, and the validator admitted that exact
  commit through a 60-second retrospective exception.
- `authority.retirement_publication_out_of_scope_retry`: the signed retirement
  grant's one ordinary correction was already consumed before the retirement
  tag was replaced and the second public run was started.

Both retirement tag objects, both public runs, and both review dispositions are
immutable evidence. Neither the successful run nor its accepted QA review is
accepted delivery authority, and this record does not retroactively authorize
any retirement commit, tag, run, review, or publication.

The prospective signed
`W-001-lifecycle-authority-recovery-v1` grant adopts exact head
`0b0195ba0953d1d6d387aad699605ff864cfac1d` and tree
`f96506ca6b7b053d41d4067eb1c926b9d4b39b40` only as an unaccepted recovery
preimage. It was issued at `2026-08-29T00:28:29Z`. Its grant SHA-256 is
`68e8051b477eb9bd3c6846874794b9e83e5e4341f5784eb3569dc6c598c0150f`;
its detached-signature SHA-256 is
`5eb415fbed952821daaebb10736b70a045ca6f1ee8e2e24834446436b81eea3b`.
The detached signature verifies under namespace
`mars3-w001-lifecycle-authority-recovery-v1` with pinned ED25519 fingerprint
`SHA256:i5VSHF257DhXJ5l/9oOUGHnT2mrqgXYSMryQHRsSBx8`. Signed grant-only commit
`ff303a95d9216d839f5b5abfe3feceb64e412e91` was committed at
`2026-08-29T00:29:37Z`, after grant issuance, and verifies with the pinned key.

The recovery may remove only the retrospective validator exception and add its
focused chronology regression while retaining the complete retirement design
and deterministic fixture bytes unchanged. It requires a distinct signed tag
`mars3/w001-lifecycle-authority-recovery-v1`, one fresh exact-head public run,
and fresh independent QA followed by Security. W-001 remains `in-progress`;
merge, Beads/lease mutation, reconciliation, production change, and downstream
work remain unauthorized.

The signed prospective recovery implementation checkpoint is commit
`2e00e8a6bf1fda7cc170d6cfc2da847c4742fb35`, tree
`2b312ddac56bcdd230a3abd925654b8063043651`, with sole parent the signed
grant-only commit. Both recovery commits verify with the pinned ED25519 key and
were committed after the grant's issue time. The exact recovery-preimage diff
contains only the seven signed paths. Diffs remain empty for `api/**`,
`internal/authority/**`, `database/**`, `.github/**`, `go.mod`, and `go.sum`;
the retirement fixture, workflow, dependencies, authority runtime, database,
native patches, and product contracts remain unchanged.

Focused authority-recovery, prospective-chronology, and deterministic Git
fixture tests passed once in 1.671 seconds and ten consecutive times in 9.488
seconds; the same focused suite passed under the race detector in 2.352
seconds. Doctrine, plan, and doc-sync repository checks passed in the source
worktree. Its public check retained the expected linked-worktree-only `.git`
scope rejection.

A fresh normal clone with a real `.git` directory checked out the exact signed
implementation commit and tree. The unmodified `go test ./... -count=1` passed
every package; `internal/doctrine` completed in 110.115 seconds. `go vet ./...`,
`git diff --check`, commit show-check, protected-surface diff checks, and clean
checkout verification passed. This is qualified prospective implementation
evidence only, pending a signed evidence checkpoint, distinct recovery tag,
one fresh exact-head public run, and fresh independent QA then Security review.

### Authority-recovery QA disposition and rejected-tag preservation

The signed authority-recovery evidence checkpoint is head
`5251ae37a2914e9f750d4c5900f46c7bb736b2d9`, tree
`82a4a84cae31d0003aecddf654217cbfb2bd29a6`, with signed annotated recovery
tag object `f561cb14a471e0cc773ba6b4bd81308ee8f0d873`. Exact-head public run
`33224636939`, job `99025722797`, passed every doctrine, plan, DocSync, public,
test/vet, whitespace, signature, canary, worktree-scan, and history-scan step.

Independent QA returned `changes-requested` with
`QA-W001-RECOVERY-001`. The original rejected retirement tag object
`e08beb772696b078783d0c75d23c1029581cdeb1` was unreachable from every
durable remote ref and absent from a fresh clone, even though evidence claimed
both retirement tag objects were preserved. Security was not run because QA
did not accept. All passing recovery evidence remains immutable but the
recovery attempt is unaccepted.

The rejected object still exists byte-for-byte in the governed source object
database. It is an annotated tag whose original tag name is
`mars3/w001-lifecycle-test-harness-retirement-v1`, target is
`ce9d0e7e82a33350263acd6199219a5349ef9bce`, target tree is
`3d6a1c44fa076d4755ec96c5755b45deae7a1ee5`, tagger is
`MARS-3 Work Authority Engineer <engineer@example.com>`, message is
`MARS-3 W-001 lifecycle test-harness retirement attestation v1`, and signature
verifies with the pinned ED25519 key. It remains rejected and unaccepted.

The prospective signed `W-001-lifecycle-evidence-preservation-v1` grant binds
exact recovery head `5251ae37a2914e9f750d4c5900f46c7bb736b2d9` and tree
`82a4a84cae31d0003aecddf654217cbfb2bd29a6`. Its grant SHA-256 is
`c8a1741d3b5a53e563999b825cded8819618ca554172ea9d2c9d4edf92fc6bda`;
its detached-signature SHA-256 is
`3dd30a5c213fe762f0dc093fb303f717abe26d7652529347160236cfc04950ac`.
The detached signature verifies under namespace
`mars3-w001-lifecycle-evidence-preservation-v1` with pinned fingerprint
`SHA256:i5VSHF257DhXJ5l/9oOUGHnT2mrqgXYSMryQHRsSBx8`. Signed grant-only commit
`fd9e093bc3cc6a282fea3362e3058fba3e310d2c`, tree
`36c8580fcd21fbe89ba27430e35626ec76e4a453`, was committed at
`2026-08-29T01:05:53Z`, after the `2026-08-29T01:04:40Z` issue time.

The grant authorizes publication of the existing rejected tag object under the
distinct archival ref
`refs/tags/mars3/w001-lifecycle-test-harness-retirement-rejected-v1`. The ref
already resolves locally to exact object `e08beb772696b078783d0c75d23c1029581cdeb1`,
target `ce9d0e7e82a33350263acd6199219a5349ef9bce`, and tree
`3d6a1c44fa076d4755ec96c5755b45deae7a1ee5`. It is not a current review tag,
does not authorize or accept the object, and must become fetchable from a fresh
clone before QA may accept. W-001 remains `in-progress`; Security, merge,
Beads/lease mutation, reconciliation, production change, and downstream work
remain pending or unauthorized as applicable.

The signed evidence-preservation implementation checkpoint is commit
`5c5cc086bb5099f4cfd16c1e8c4cd5230d8ca78e`, tree
`0b593f90f2188e585dfe497483ecb416e25d3ff6`, with sole parent the signed
grant-only commit. Both preservation commits verify with the pinned ED25519
key and were committed after grant issuance. The exact preservation-base diff
contains only the seven authorized paths; protected runtime, database,
workflow, dependency, product, feature, and product-spec surfaces are
unchanged.

Focused evidence-preservation, archival-tag, prospective-chronology, and
deterministic Git fixture tests passed once in 1.697 seconds and ten
consecutive times in 10.379 seconds; the same focused suite passed under the
race detector in 2.418 seconds. Doctrine, plan, and DocSync repository checks,
`go vet ./...`, and whitespace checks passed.

A fresh normal clone with a real `.git` directory checked out exact commit
`5c5cc086bb5099f4cfd16c1e8c4cd5230d8ca78e` and tree
`0b593f90f2188e585dfe497483ecb416e25d3ff6`. In that clone, archival ref
`mars3/w001-lifecycle-test-harness-retirement-rejected-v1` resolved to tag
object `e08beb772696b078783d0c75d23c1029581cdeb1`, target
`ce9d0e7e82a33350263acd6199219a5349ef9bce`, and tree
`3d6a1c44fa076d4755ec96c5755b45deae7a1ee5`; the object type was `tag`.
The unmodified `go test ./... -count=1` passed every package, with
`internal/doctrine` completing in 111.320 seconds. Vet, whitespace,
show-check, protected-surface diff, and clean-checkout checks passed. This is
qualified prospective implementation evidence pending a signed evidence
checkpoint, distinct preservation review tag, one exact-head public run, and
fresh independent QA then Security review.

### Accepted evidence-preservation merge and deferred terminal closeout

The evidence-preservation candidate completed at signed head
`56c2a8d95927bc552882aacc30aa886ea0be9ba5`, tree
`1025c59672c70b8d14ad904315677ffac56b9b81`, and signed annotated review tag
object `eb8838eccd63f42f5a81bdcdd32cd597d84bfc2c`. The distinct archival ref
`mars3/w001-lifecycle-test-harness-retirement-rejected-v1` resolves in a fresh
normal clone to exact rejected tag object
`e08beb772696b078783d0c75d23c1029581cdeb1`; it remains rejected and is not an
acceptance signal. Exact-head public run `33226093357`, job `99030041600`,
passed. Independent QA accepted and closed `QA-W001-RECOVERY-001`; Independent
Security accepted with `security.w001_evidence_preservation.accepted`. No open
finding remained on that immutable tree.

User-authorized PR #10 squash-merged the accepted candidate at protected-main
commit `f6073696fab0ecf9e80d34f5c199ca54f431b5f7`, sole parent
`7f35c8a7112946a9569efe6085f49da8fd28530e`, with tree
`1025c59672c70b8d14ad904315677ffac56b9b81` exactly equal to the reviewed
candidate tree. Protected-main run `33246178629`, job `99083920743`, passed.
The review and archival tags remain exact. This merge accepted the gateway and
its durable evidence; it did not mutate canonical Beads or PostgreSQL state,
and M3-W001 remains `in-progress` with no live lease.

The prospective signed `W-001-lifecycle-terminal-reconciliation-v1` grant is
the only current closeout authority. It binds base commit
`f6073696fab0ecf9e80d34f5c199ca54f431b5f7`, base tree
`1025c59672c70b8d14ad904315677ffac56b9b81`, the accepted candidate and both
passed runs, and exactly eleven implementation/evidence paths. Its grant-only
commit is `ce12312228ce2a4b57e9d68c7113c90c85018b4f`, committed after the grant's
`2026-08-29T10:53:24Z` issue time. The grant SHA-256 is
`835841a5cd5b1cbcb1debb4a6a98c4ce2eb83ccff396f84b4823444bb504eef5`; its
detached-signature SHA-256 is
`ca4e2376f65c6dc23e90a4e925330c1f5f3f84163148758c907f6b14234f2e9b`, and
the signature verifies under namespace
`mars3-w001-lifecycle-terminal-reconciliation-v1` with the pinned ED25519 key.

The implementation adds one gateway-only closeout composition. Dry-run must
verify the exact signed Git/CI and canonical store preimage with zero mutation.
Canonical apply remains deferred until the exact PR #11 candidate passes CI,
Independent QA and Independent Security accept the same immutable tree, the
tree is squash-merged to protected main, and a fresh one-hour signed execution
authorization binds the protected-main commit/run plus external Beads and
PostgreSQL identities. No credential, database URL, raw authority payload,
production effect, or downstream ticket is authorized.

### Terminal-reconciliation implementation qualification

The signed implementation checkpoint is commit
`20335cbc274785ee68651b244a11d5f768abe30b`. It adds the deep
`closeout.Run` composition and thin `mars3-authority terminal-reconcile`
command. Dry-run performs only signed-boundary and canonical-store reads;
apply composes the existing typed claim-reconciliation, handoff, ordered QA and
Security verdict, completed-run, merge-reconciliation, and terminal-close
gateway methods. Every step carries the returned `WorkVersion` and integrity
postimage into the next CAS. A completed claim saga whose lease was released
before an unknown handoff receipt is recovered by exact idempotency key,
intent, work, and full released-fence validation; it cannot issue a second
lease. The public receipt excludes paths, credentials, connection strings, raw
authority documents, and private state.

Local qualification found one bounded pre-publication correction set. Commit
`c77683ef9d3e5b69ab8147c315de30e3e0fcfc10` preserves the legacy bootstrap
command byte attestation against immutable PR #10 base
`f6073696fab0ecf9e80d34f5c199ca54f431b5f7` while the terminal validator owns
the grant-authorized current command. Commit
`654d2b5271886e75b0426d7baaeedb839d2d5210` adds the required existing F-002,
ADR-001, and code-map lineage markers to the three new source/test files. This
consumes the grant's one ordinary qualification correction. It does not alter
the terminal architecture, workflow, dependencies, API, database, gateway,
Beads store, PostgreSQL store, production, or downstream work.

The implementation checkpoint tree is
`889a0c6319125c92682ef3211fd09f92104188f0`. Its four commits form a
prospective, signed, one-parent chain from the exact signed base. The complete
base-to-checkpoint diff is exactly the eleven authorized paths. Doctrine,
plan, and DocSync checks passed in the source worktree. Public-check and the
embedded full-suite public assertion correctly rejected that linked
worktree's `.git` pointer; no validator weakening was made.

A fresh normal local clone at the exact checkpoint passed doctrine, plan,
DocSync, public-check, `git diff --check`, and `git show --check`. In that clean
clone, unmodified `go test ./... -count=1` passed every package, with
`internal/doctrine` completing in 111.148 seconds; `go vet ./...` passed.
Focused closeout, command, signed-grant, prospective execution-window,
workspace-identity, released-lease recovery, and terminal-scope tests passed
ten consecutive times. The same focused suite passed with the race detector.
This remains qualified implementation evidence only: a final signed evidence
checkpoint, signed annotated review tag, exact-head public CI, independent QA
then Security, protected-main merge, fresh one-hour execution authorization,
canonical dry-run/apply, and external readback are still required.

### Rejected terminal v1 candidate and bounded CI recovery

The first terminal-reconciliation candidate is immutably rejected at signed
head `cff16fda43acd9d621f701064427b0d7fc3bf30d`, tree
`447cac655e6d4d34bce9a8f9641a677e3e7511cc`, and signed annotated v1 tag
object `06a9075aef9a3741eca7b5ea03f6d9aca0ab2dda`. PR #11 public run
`33250298926`, job `99094724907`, passed doctrine, plan, DocSync, public
validation, tests/vet, whitespace, commit/tag validation, and the synthetic
scanner canary. Its worktree scan returned one `generic-api-key` finding at
`internal/authority/closeout/closeout_test.go:239`. Read-only replay with the
same pinned scanner identified the value as the public canonical
authority-generation UUID used by `closeoutFixture`; it is not a credential.
The v1 commit, tag, run, and failed disposition remain immutable and are not
acceptance evidence.

The prospective signed `W-001-lifecycle-terminal-ci-recovery-v1` grant binds
that exact rejected head, tree, tag object, run, job, file, line, rule, and
fingerprint. Its grant SHA-256 is
`8380dfd8eba71c349f2fc2ac00265d59f43256d147db4be45ff07f2279f6558e`;
its detached-signature SHA-256 is
`196c0e472c0ac54942ccaa6e7c46380ade3bb042f44fbdc10a1fdc73f0dd3242`.
The signature verifies under namespace
`mars3-w001-lifecycle-terminal-ci-recovery-v1` with the pinned ED25519 key.
Signed grant-only commit `411a864ee213019b3700c9668772a10df6263f54`,
tree `808f0680497734b575a2c0de0ec333c512660f1f`, was committed at
`2026-08-29T11:34:50Z`, after its `2026-08-29T11:33:56Z` issue time.

Recovery may change only the scanner-triggering public test literal into a
deterministic byte-identical value plus this signed grant, manifest, plan,
evidence, and terminal validator/tests. Closeout runtime, command, API,
database, gateway, Beads/PostgreSQL stores, workflow, dependencies, product,
canonical state, and production remain unchanged. The recovery allows one
distinct signed v2 review tag and fresh CI, QA, and Security on PR #11. Any
further failure stops and escalates; no additional correction is authorized.

### Exact terminal history-scanner recovery

The scanner-safe fixture replacement is signed at head
`8dcb71192107589c4a5abe5e2d84ff8825c4d17f`, tree
`ad4a78f374e64c6f0e38e9a43cdec86eef03ab78`. Worktree scanning is clean, but
the required full-history scan truthfully rediscovers the immutable public UUID
fixture in implementation commit `20335cbc274785ee68651b244a11d5f768abe30b`.
The exact non-secret finding is
`20335cbc274785ee68651b244a11d5f768abe30b:internal/authority/closeout/closeout_test.go:generic-api-key:231`.
The rejected v1 candidate, tag object, failed run, first recovery grant, and
recovered fixture head remain immutable and unaccepted.

The prospective signed `W-001-lifecycle-terminal-history-scan-recovery-v1`
grant binds the exact recovered head and tree. Its SHA-256 is
`fc721be90f8a98435d8041b3709f26a2c86c22eca576cab31a3d2edd842ae32d` and its
detached-signature SHA-256 is
`3086ab04191ee657bc7a5e84afb93b974ea888b0bb0ed84c8433ef157cfac463`.
Signed grant-only commit `02825a52e6484613154374b0bf1b2102179647d7`,
tree `0b334898d6e8d04bfda64b74b078019c81e78c89`, was committed after the
grant's `2026-08-29T15:03:58Z` issue time. Its sole scanner effect is to extend
the closed ignore file from ten to eleven exact commit:path:rule:line tuples.
Wildcards, scanner configuration changes, source rewrites, and any
runtime, workflow, dependency, canonical Beads/PostgreSQL, production, or
downstream effect remain prohibited. The exact source tuple must resolve to a
nonempty line in the immutable commit and must be an ancestor of the signed
recovery base. Fresh pinned worktree and full-history scans, clean-clone gates,
one signed v2 tag, exact-head CI, independent QA, and then independent Security
are required. Any further CI or review failure stops the attempt. W-001 remains
`in-progress`.

Signed implementation commit `974799b1a9246ef40d1afba04a0a1cc858e25f27`,
tree `a1a85b006841fa9fe27a2684a3b924be85a6944e`, changes exactly the six
remaining paths allowed by the grant; together with grant-only commit
`02825a52e6484613154374b0bf1b2102179647d7`, the base-to-head diff is exactly
the eight authorized paths. The focused signed-grant, path-scope, scanner-ignore,
source-tuple, and prior-recovery suite passed ten consecutive runs and passed
under the Go race detector.

A fresh normal clone of that exact implementation commit passed doctrine,
plan, DocSync, public-check, whitespace, and show-check. Unmodified
`go test ./... -count=1` passed every package, with `internal/doctrine`
completing in 113.975 seconds, and `go vet ./...` passed. The digest-pinned,
network-disabled Gitleaks image
`docker.io/zricethezav/gitleaks@sha256:75bdb2b2f4db213cde0b8295f13a88d6b333091bbfbf3012a4e083d00d31caba`
found no leak in the worktree and no leak across all 116 commits of history.
This is local qualification only; the signed v2 review tag, exact-head public
CI, independent QA, independent Security, and protected-main merge remain
pending.

### Rejected v2 tag identity and prospective v3 recovery

The terminal history-scanner candidate is immutably rejected at signed head
`b80dfbb289f150cbe812dcb67a289227e20c9cea`, tree
`ea8739b1339ce20460ec665cd6082552079c2d83`, and signed annotated v2 tag object
`7682185819444d1fc38c863aa9ed869827146eca`. Exact-head run `33260154000`, job
`99120586295`, failed at `Check doctrine` with
`public.w001_delivery_tag_identity`: the v2 tag used
`engineer@example.com`, while the signed review contract requires
`release-manager@example.com`. All later CI steps were skipped. The v2 tag,
target, message, signature, failed run, and changes-requested disposition remain
immutable and are not acceptance evidence.

The prospective signed `W-001-lifecycle-terminal-tag-identity-recovery-v1`
grant binds that exact head, tree, tag object, run, job, observed identity, and
required Release Manager identity. Its SHA-256 is
`5522bf5cc0245c5f432fecd8b765aaf5ffc5e2a0a2c13933f15f77fd2f08e28f`; its
detached-signature SHA-256 is
`25dcd076db3dea9f834a5c00ea9332534f58c3aaef439aac7a0c435d732869f0`.
Signed grant-only commit `3a28830b64faf2d2d62e6ed5ede1b0a4a7a9434b`, tree
`009864d983f5bfaaf81da7b6c3605d01e4087a66`, was committed after the grant's
`2026-08-29T16:40:47Z` issue time.

Recovery may update only the seven signed grant, manifest, plan, evidence, and
terminal-validator/test paths. Runtime, command, workflow, dependency, scanner,
API, database, gateway, canonical Beads/PostgreSQL, production, product, and
downstream bytes remain unchanged. One distinct v3 tag must target the exact
new feature head and must verify under the pinned key with tagger identity
`MARS-3 Release Manager <release-manager@example.com>`. Fresh exact-head CI
must precede independent QA and then independent Security. Any further failure
stops the attempt. W-001 remains `in-progress`.

Signed implementation commit
`a34640a73b2113c738017a66734fb6ea96beb0b4`, tree
`33b2cec58006b457553ee2a9afdc397bcb181af4`, changes the five remaining
manifest, plan, evidence, terminal-validator, and focused-test paths. Together
with grant-only commit `3a28830b64faf2d2d62e6ed5ede1b0a4a7a9434b`, the
base-to-head diff is exactly the seven authorized paths. Both commits verify
under the pinned ED25519 Git key.

The focused terminal tag-identity, prior-recovery, path-scope, and scanner
regressions passed ten consecutive runs and passed under the Go race detector.
A fresh normal clone of the exact implementation commit passed doctrine, plan,
DocSync, public-check, whitespace, and show-check. Unmodified
`go test ./... -count=1` passed every package, with `internal/doctrine`
completing in 147.025 seconds, and `go vet ./...` passed. The digest-pinned,
network-disabled Gitleaks image
`docker.io/zricethezav/gitleaks@sha256:75bdb2b2f4db213cde0b8295f13a88d6b333091bbfbf3012a4e083d00d31caba`
found no leak in the exact worktree and no leak across all 119 commits of
history. This remains local qualification only; the signed v3 review tag,
exact-head public CI, independent QA, independent Security, and protected-main
merge remain pending.

### QA-rejected v3 and prospective exact-tagger v4 recovery

The v3 candidate is immutably rejected at signed head
`70ed4a5c502ff3184479d08fd50a19e10cd4af0b`, tree
`39134c754f2263f2ec10d9bd4e42789ab86cf0fd`, and signed annotated tag object
`5f621e96520d933b4ff5c751948d9dd4710a7f0c`. Exact-head run
`33264433302`, job `99131880421`, passed every public gate. Independent QA
returned `changes-requested` with finding
`public.w001_terminal_tag_identity_name_not_exact`: the actual v3 tag uses
`MARS-3 Release Manager <release-manager@example.com>`, but the validator
matched only `release-manager@example.com` and would also accept a different
name. Independent Security did not run, and PR #11 was not merged.

The prospective signed `W-001-lifecycle-terminal-exact-tagger-recovery-v1`
grant supersedes the v3 stop clause only for exact name-plus-email enforcement
and a wrong-name/correct-email regression. Its SHA-256 is
`c9ccb6480d9b96b31b45c131288256c04a3f30b868b2c527682d80b16f10a33c`;
its detached-signature SHA-256 is
`e178e5f0e3fbb109dd9750740b2f71b394fc1adad510c4d115d76ae4e6c517a5`.
Signed grant-only commit `d95d46a61aab9761cf502dc34c4ecd59df85dc74`,
tree `d4d1bca8f9b618527e8dce8ab1bfc074eaf22b9a`, was committed after the
grant's `2026-08-29T22:39:34Z` issue time.

The adversarial regression first reproduced the defect by accepting a signed
`Wrong Name <release-manager@example.com>` tag, then passed after the narrow
tag-header parser required the exact signed name and email. Recovery may update
only the seven signed grant, manifest, plan, evidence, terminal-validator, and
focused-test paths. Runtime, command, workflow, dependency, scanner, API,
database, gateway, canonical Beads/PostgreSQL, lease, production, product, and
downstream bytes and effects remain prohibited. One distinct v4 tag, fresh
exact-head CI, independent QA, and then independent Security are required.
Any further failure stops the attempt. W-001 remains `in-progress`.

Signed implementation commit
`8df0fb62c4f2b0624b5891695d8f651e8a55b10a`, tree
`3c4ef3998fc830fef8cce7749491ed3fe30c6d6b`, changes the five remaining
manifest, plan, evidence, terminal-validator, and focused-test paths. Together
with grant-only commit `d95d46a61aab9761cf502dc34c4ecd59df85dc74`,
the base-to-head diff is exactly the seven authorized paths. Both commits
verify under the pinned ED25519 Git key.

The signed wrong-name/correct-email regression failed before the parser change
and passed after exact name-plus-email enforcement. The focused exact-tagger,
prior-recovery, and path-scope suite passed ten consecutive runs and passed
under the Go race detector. A fresh normal clone of the exact implementation
commit passed doctrine, plan, DocSync, public-check, whitespace, and show-check.
Unmodified `go test ./... -count=1` passed every package, with
`internal/doctrine` completing in 159.583 seconds, and `go vet ./...` passed.
The digest-pinned, network-disabled Gitleaks image
`docker.io/zricethezav/gitleaks@sha256:75bdb2b2f4db213cde0b8295f13a88d6b333091bbfbf3012a4e083d00d31caba`
found no leak in the exact worktree and no leak across all 122 commits of
history. This remains local qualification only; the signed v4 review tag,
exact-head public CI, independent QA, independent Security, and protected-main
merge remain pending.

## Canonical terminal reconciliation and external readback

**Disposition:** terminal reconciliation verified; public evidence handoff in
review

Protected main
`4cfa28b42679651d0f198418b54fb05fc3483c4d`, tree
`64697884630206f76004321fbca4b787bf1c427c`, remained byte-equal to the
independently accepted v4 candidate tree. Remote readback retained signed v4
tag object `c471bf1df39ca7035fb3e4a3cfc90a35eea2404e` targeting candidate
`327ce7fe81327517d8a00a5254f63ab3bfb4c7f5`. Protected-main run/job
`33280991676`/`99176016186` completed successfully against that exact main
commit. The prior v1-v3 tags, failed runs, and review dispositions remain
immutable.

### External execution boundary

The accepted patched Beads source remained pinned to upstream commit
`6c124203e771433a3550c348771a5b5e27fd3c21` with the exact six-file patch
digest `91b3e8dd5c8c01b5953c4c38ca508a150b05cd719f4e80fec293365afddf7f`.
Two isolated, network-disabled Darwin arm64 builds were byte-identical at
SHA-256 `4bce9c19511f9e718b3506d5c29a90a2eea7484f3b64b634d590aff8d3bcaec3`.
The non-skipped `TestNativeMutatorIntegration` passed against that exact host
binary. The canonical Beads workspace identity digest was
`b38697aa895101ef72c47bb299f9f858054a3f2c31bf5a239a09620996fb3ae1`;
its project ID remained `e9669a62-5be6-4b94-85f8-c502c29d394a`.

The authorized local, non-production database `mars3_authority_w001` applied
the checked-in migration unchanged at SHA-256
`fb29d13c4038ee37c8d7fe50603596c53df810767558c1d7a4b120afcc74bcf9`.
Its dedicated application role is non-superuser, the temporary connection URL
file was mode `0600`, and typed `postgres.Store.ProvisionProject` anchored
tenant `tenant-academy`, the canonical project ID, and fresh fence generation
`a8995f20-846c-4ea1-8a22-215c518df50a`. No lifecycle, lease, event, or Bead
row was written directly.

The fresh external authorization was valid from `2026-08-30T00:36:29Z` to
`2026-08-30T01:36:29Z`. Its canonical JSON SHA-256 was
`82cbc3f93502dea081e0a857e8d81e22e8386d67bbfc7a74bdd1c2f9f18f1ff4`;
detached-signature SHA-256 was
`8679a30a9ed69a9e6079815a4b33076ff58be00886b47367f176024af3e7e433`.
The signature verified with the pinned ED25519 genesis key under namespace
`mars3-w001-lifecycle-terminal-reconciliation-execution-v1`. The public
evidence records only hashes and non-secret identities; it excludes the
authorization payload, private signing material, and database URL.

The first dry-run failed closed before mutation because the delivery checkout
was a linked worktree whose `.git` entry is a file; the reviewed closeout
boundary accepts only a normal clone with a real `.git` directory. A fresh
normal local clone at the same protected-main commit and tree, with the same
immutable tags, was the materially different bounded route. Its canonical
dry-run returned `ready-no-mutation` with native `in_progress`, lifecycle
`in-progress`, WorkVersion sequence `1`, no live lease, no reviews, and no
terminal record.

### Gateway apply receipt

The one authorized `terminal-reconcile --apply` invocation returned
`terminal-close-verified` for M3-W001. The receipt recorded:

- native status `closed` and lifecycle `done`;
- WorkVersion authority generation
  `6e79ff81-a007-42a5-a178-7ce58dbb718b`, issue incarnation
  `e1e8d2d3f80871096a568fb489f49575a42abd37b269df9faf777a09cd689b41`,
  mutation sequence `7`, and dependency revision `1`;
- handoff head `56c2a8d95927bc552882aacc30aa886ea0be9ba5`;
- exactly two ordered accepted reviews: QA then Security;
- completed run disposition;
- reconciliation to merged commit
  `f6073696fab0ecf9e80d34f5c199ca54f431b5f7`, tree
  `1025c59672c70b8d14ad904315677ffac56b9b81`, PR `pr-10`, and protected-main
  run `run-33246178629`;
- terminal record present; and
- no live lease after handoff.

The native Beads readback independently showed M3-W001 `closed`/`done`, labels
including `done` and `public-first`, the original bootstrap claim, terminal
WorkVersion sequence `7`, two ordered accepted review records, completed run
record, PR #10 reconciliation record, and Delivery Orchestrator terminal
record. No other Bead was selected for mutation.

The least-privilege PostgreSQL readback independently showed exactly one lease,
lease epoch `1`, state `released`, and zero active leases. The project journal
high watermark and event count were both `21`, with chain hash
`703b109123eebbf56ec295b70ff528a8c9fcfee9077d30e8ed3b4f8e0f1dece4`.
Its ordered events were seven intent/policy/receipt triples for claim
reconciliation, handoff, QA review, Security review, completed run,
reconciliation, and terminal transition. PostgreSQL retained no canonical
ticket definition or lifecycle projection.

Remote Git readback still resolved protected main to
`4cfa28b42679651d0f198418b54fb05fc3483c4d`, v4 tag object to
`c471bf1df39ca7035fb3e4a3cfc90a35eea2404e`, and the tag target to
`327ce7fe81327517d8a00a5254f63ab3bfb4c7f5`. GitHub readback reported
protected-main run `33280991676` and public-commit job `99176016186` completed
with conclusion `success`.

Canonical W-001 execution authority is consumed. This three-file branch may
receive exact-head CI, independent QA, independent Security, and merge after
acceptance. It authorizes no further source or store mutation, no new lease,
no production effect, no P-001 or downstream work, and no change to workflows,
dependencies, runtime, API, or database design.

### PR #12 topology blocker and prospective publication correction

Signed evidence commit
`8648be7fd2f36f872b36f7764897343cb84135b2`, tree
`104a6bd0331d99a55ed09853e2256bc4a9379308`, remains the immutable head of
PR #12. Its sole parent is accepted v4 candidate
`327ce7fe81327517d8a00a5254f63ab3bfb4c7f5`, not the exact-tree squash on
protected main. GitHub therefore reports the PR `open` with merge state
`dirty` and cannot construct a merge ref or exact-head run. The commit, branch,
and PR remain unchanged as process evidence; they are not acceptance evidence
and will not be closed, rewritten, recreated, or merged by this correction.

Finding `public.w001_terminal_evidence_topology_unqualified` records that the
terminal validator admits only PR #11 and the preterminal base. It consequently
cannot qualify a post-terminal durable-evidence publication descended from
protected main `4cfa28b42679651d0f198418b54fb05fc3483c4d`, tree
`64697884630206f76004321fbca4b787bf1c427c`, despite the accepted v4 tag
object `c471bf1df39ca7035fb3e4a3cfc90a35eea2404e` and successful protected-main
run/job `33280991676`/`99176016186`.

The prospective signed
`W-001-lifecycle-terminal-evidence-publication-v1` grant is byte-exact at
SHA-256 `2a1219e8859da812a773dabfe0e18ce2c5382299e7ebe485c6b9e2c296cae8d2`;
its detached-signature SHA-256 is
`1c9a8ea13381a162fbb8b01a85ee2498b6e0faed8f86456862409c3f00103145`.
The signature verifies under namespace
`mars3-w001-lifecycle-terminal-evidence-publication-v1` with the pinned
ED25519 key. Signed grant-only commit
`2eb932a2cf7210190da919da63690b512b0acefb` is prospective to the
`2026-08-30T18:37:30Z` issue time. Signed evidence replay commit
`a09cb3262f7a9132664f481386b643ed520b1e4a` reproduces only the original three-file
terminal handoff from protected main; it does not modify the preserved commit
or PR.

The correction is limited to the grant/signature, manifest, active plan,
W-001 evidence, terminal validator, and focused tests. It adds one exact PR #13
two-parent synthetic-merge topology over protected main plus its exact-tree
squash readback. Runtime, gateway, stores, workflows, dependencies, database
design, API, production, downstream work, and every Beads/PostgreSQL lifecycle,
event, or lease mutation remain prohibited. Canonical M3-W001 remains native
`closed`, lifecycle `done`, and without a live lease. A distinct signed review
tag, exact-head CI, independent QA, and then independent Security are required
before PR #13 may merge.

Signed validator/evidence implementation commit
`e0ea17507d9b843a73023655aebd85ef0c8945fa`, tree
`be6e5cba8004a7a0faaf34defc87a4cb3bac3144`, adds the exact PR #13 topology,
preserved-PR rejection regression, signed-grant verifier, prospective linear
commit check, exact seven-path allowlist, and protected-main squash check. The
focused terminal-evidence and inherited exact-tagger suite passed. Doctrine,
active-plan, DocSync, public-content, whitespace, and `go vet ./...` gates
passed. Unmodified `go test ./... -count=1` passed every package, with
`internal/doctrine` completing in 158.171 seconds. This is local qualification
only; the distinct signed review tag, exact-head public CI, independent QA,
independent Security, and accepted merge remain pending.

### Publication v1 QA rejection and bounded v2 correction

Publication v1 evidence remains immutable at candidate
`63eed57083e31a18b9ac2c3143802d021d37e4f0`, tree
`b16861c96a7553e1abe6c62b591710104e93287b`. Signed tag
`mars3/w001-lifecycle-terminal-evidence-publication-v1` remains object
`91b2d9ffdc7d4d5a679678a6d30f0f52561c1c19`, targets that exact candidate,
and records tagger time `2026-08-30T18:52:46Z`. Exact-head CI run/job
`33329491442`/`99305391610` passed. Independent QA reviewed synthetic PR #13
merge `b3c176a6d5959d0706cbf5692000ca1dea697617` and returned
`changes-requested` with two open findings:

- `public.w001_terminal_evidence_state_reopened`: the Git projection reopened
  canonical W-001 as `in-review` instead of keeping the Bead and delivery row
  `done` with publication review represented separately.
- `public.w001_terminal_evidence_tag_chronology_unbounded`: signed tag identity
  and target were verified, but the signed tagger timestamp was not parsed and
  bounded after grant issuance and target commit and before grant expiry.

PR #12 remains preserved at signed head
`8648be7fd2f36f872b36f7764897343cb84135b2`; it is not an acceptance route.
After PR #13 receives accepted v2 QA and Security review, merges, and passes
protected-main CI, PR #12 is to be commented on and closed as superseded
without changing its head or branch and without merging or rebasing it.

The prospective signed
`W-001-lifecycle-terminal-evidence-publication-v2` grant is byte-exact at
SHA-256 `5fd6504d2409b344dcc138358bc1146d85b3c84b738577bd987a6ad1884a918e`;
its detached-signature SHA-256 is
`e3f299188775875cbc267a160368c20f5dbaf6043652ecc906e26d2fb3a9cff9`.
The signature verifies under namespace
`mars3-w001-lifecycle-terminal-evidence-publication-v2` with the pinned
ED25519 key. Signed grant-only commit
`884213d1ae6521fe0b832d6dfe7589fb4c1addec`, tree
`4b788cf81bcf1fb62d50ca8114a55209cf04ec66`, is prospective to the
`2026-08-30T19:10:00Z` issue time.

The v2 implementation is restricted to admitting exactly one current `done`
W-001 row with zero active delivery rows, preserving the existing nonterminal
delivery cardinality, and parsing the signed annotated-tag object so tagger
time is on or after grant issuance and target commit time and strictly before
grant expiry. Focused regressions cover terminal projection, parallel active
work denial, backdating, pre-target timestamps, expiry, and malformed tagger
time metadata. This is not canonical lifecycle work: the publication remains
`in-review` while M3-W001 remains `done` and the live lease remains absent.

### Premature PR #13 merge and forward-only v3 recovery

PR #13 merged before the required v2 exact-head publication and ordered
QA-to-Security review. GitHub readback binds the event to rejected v1 head
`63eed57083e31a18b9ac2c3143802d021d37e4f0` and squash commit
`4ec60790abdd34a12b209f70d631def0b44ab465`. The protected-main tree is
`b16861c96a7553e1abe6c62b591710104e93287b`, exactly the rejected v1 tree.
Protected-main run/job `33330996615`/`99309411316` passed, but green CI does
not reclassify the immutable QA `changes-requested` disposition or supply the
Security review that was correctly gated behind QA. The merge is retained as
`unaccepted-process-evidence`; it is not reverted, amended, or rewritten.

PR #12 is closed without merge and retains exact head
`8648be7fd2f36f872b36f7764897343cb84135b2`. It remains process evidence and
will not be reopened, rebased, or merged.

The locally qualified but unpublished v2 attempt remains reachable at exact
head `8d24e259d8d078d888c9c6d2060dc57b6eda1ee9`, tree
`999ca4d942c41eab0e6568150276b81772e4965c`. Signed archival tag
`mars3/w001-lifecycle-terminal-evidence-publication-v2-unmerged` is object
`2d36c895cfd9f47927f635b16bf911d3a070758e`, targets that head, and uses the
exact Release Manager identity. The archive preserves signed grant-only commit
`884213d1ae6521fe0b832d6dfe7589fb4c1addec`, implementation commit
`d5eff09fc6b11eb3f5898c5b1e05ca2c2f549450`, and focused-test commit
`8d24e259d8d078d888c9c6d2060dc57b6eda1ee9` without presenting them as a
reviewed or merged candidate.

The forward-only signed
`W-001-lifecycle-terminal-evidence-publication-v3` grant begins at protected
main `4ec60790abdd34a12b209f70d631def0b44ab465`, tree
`b16861c96a7553e1abe6c62b591710104e93287b`. Its SHA-256 is
`eff2bd4a1e546dccf87a9d0ee072841974ec85ce9b99c2108da91302da08a4aa`;
detached-signature SHA-256 is
`d9bc55cddfe12fee99e7c8d7184f46fe764719126ce77c1554fda1256ce6a041`.
The signature verifies under namespace
`mars3-w001-lifecycle-terminal-evidence-publication-v3` with the pinned
ED25519 key. Prospective signed grant-only commit
`856434ebb798245b81aa46c779571b0669f1c038`, tree
`690c9190b5244ecd3785f99c372c6168a11e4937`, precedes signed replay commits
`1906539f063bbd85e4539bbacc963a2f69e14704` and
`da327368a94032e918456a3f26a1b463e8d8faee`.

V3 may reproduce only the already-qualified terminal `done` projection and
signed tag chronology enforcement. PR #14 must carry a distinct signed v3
review tag, pass exact-head public CI, and receive independent QA followed by
independent Security acceptance before squash merge. Runtime, gateway, store,
workflow, dependency, database design, API, Beads/lease, production, and
downstream changes remain prohibited.

Signed v3 implementation head
`791132d6a7047f658bdb58db7313c1f53be9c1fa` was then qualified from a fresh
canonical GitHub clone with the immutable branch head fetched by object ID.
The exact nine-path base-to-head diff matched the v3 grant. Repository build,
doctrine, active-plan, DocSync, public-content, whitespace, and `go vet ./...`
gates passed. The focused terminal-publication race suite passed in 4.362
seconds. Unmodified `go test ./... -count=1` passed every package, with
`internal/doctrine` completing in 186.542 seconds. Those results were local
qualification only. The subsequent distinct signed V3 review tag and exact-head
CI passed, but the independent QA disposition below rejected the candidate
before Security or merge.

### Publication v3 QA rejection and strict v4 recovery

PR #14 remains open at exact signed V3 head
`0d193d29e9087a8a2022d70aa8bb9943f5e84a3d`, tree
`6a8fce667d4b57134a902ed31a4843ada20f6948`. Signed tag
`mars3/w001-lifecycle-terminal-evidence-publication-v3` remains object
`e708daafba37e4efe3096b64de30eb10033d2201` and targets that head. Its tagger
timestamp is `2026-08-30T20:28:53Z`, strictly after the target commit timestamp
`2026-08-30T20:28:08Z`, using exact identity
`MARS-3 Release Manager <release-manager@example.com>`. Exact-head CI run/job
`33333798814`/`99316989127` passed.

Independent QA returned `changes-requested` with finding
`public.w001_terminal_evidence_tag_chronology_allows_target_equality`. The V3
tag itself is correctly ordered, but the validator accepted a hypothetical tag
whose timestamp equalled its target commit. Security did not run because QA
acceptance is the required preceding gate. PR #14, its head and tree, the V3
tag, CI, and QA disposition remain immutable rejected process evidence.

The prospective signed
`W-001-lifecycle-terminal-evidence-publication-v4` grant is byte-exact at
SHA-256 `72875c2578d73c960319ee95bf8093ceb738b42d261f83bcc115bffd555a65ce`;
its detached-signature SHA-256 is
`e3b83fe3ee62d07a99df5c9037614d515b4997b7dfdd2a179714d7056be3a858`.
The signature verifies under namespace
`mars3-w001-lifecycle-terminal-evidence-publication-v4` with the pinned
ED25519 key. Signed grant-only commit
`24158593af20bada70956eb4c79c0e19451c577c` begins at exact protected main
`4ec60790abdd34a12b209f70d631def0b44ab465`, tree
`b16861c96a7553e1abe6c62b591710104e93287b`, and is prospective to the
`2026-08-30T20:40:00Z` issue time.

V4 replays the accepted V3 terminal `done` projection, changes tag chronology
from “not before target” to “strictly after target,” adds the equality
regression, and corrects the stale active-plan V2 wording to V4. PR #15 must
carry a distinct signed V4 review tag, pass exact-head public CI, and receive
independent QA followed by independent Security acceptance before squash
merge. Runtime, gateway, store, workflow, dependency, database design, API,
Beads/lease, production, and downstream changes remain prohibited.

Signed V4 implementation head
`59caf1f168421791e367f9292bcc7e84be08af5b`, tree
`f924b59946cd47276ad824eec8f6a3175f5767c0`, changes exactly the nine
grant-authorized paths from protected main. Both V4 commits verify with the
pinned ED25519 key. Focused V4 grant, scope, strict-chronology, PR #15 topology,
and inherited exact-tagger tests passed. The full doctrine package passed in
179.167 seconds before the commit and 170.755 seconds at the signed head.
Build, doctrine, active-plan, DocSync, public-content, whitespace, unmodified
`go test ./... -count=1`, and `go vet ./...` gates passed at that head.

The first clean normal clone failed closed because its default refspec did not
make preserved delivery-v1 head
`919f1189fb0703e42bcc11570a59527ad8e7a444` locally reachable for the existing
scanner-source proof. No source was changed and the same recovery method was
not repeated. The materially different bounded route fetched that immutable
object by exact ID into the normal clone. Doctrine and public-content then
passed, as did build, plan, DocSync, whitespace, `go vet ./...`, and unmodified
`go test ./... -count=1`; clean-clone `internal/doctrine` completed in 174.418
seconds. These remain local qualification results. The distinct signed V4 tag,
exact-head public CI, independent QA, independent Security, and accepted merge
remain required.
