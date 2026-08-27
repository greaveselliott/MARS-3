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
