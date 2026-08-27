# W-001 Security correction validation

**Classification:** PUBLIC
**Work authority:** M3-W001
**Failure ownership:** foundation
**Correction grant:** `W-001-postclaim-chronology-correction-v6`
**Current disposition:** postclaim reconciliation accepted, merged, and completed; W-001 delivery is active under a separate signed grant

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

The separately signed `W-001-delivery` grant is now the bounded implementation
authority for the Work Authority Engineer. It permits only W-001's canonical
exclusive paths plus the named Orchestrator-owned plan and validator paths,
public synthetic development fixtures, and W-001 development/test leases. It
does not authorize another canonical claim, a lifecycle transition, review or
terminal disposition, production, destructive work, repository-control
changes, secrets, private data, or any other Bead. No development lease exists
at this handoff checkpoint.
