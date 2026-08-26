# MARS doctrine provenance and ownership boundary

**Status:** Accepted
**Owner:** Foundation Maintainer
**Pinned upstream:** `greaveselliott/MARS` at
`f55d129bfc794510ca485bb54fc0a35c7b04a700`
**License:** Apache License 2.0

## Foundation and deployed surfaces

- **Foundation harness:** this repository's architecture, policy, generated
  target defaults, validation, and release process.
- **Runtime substrate:** future orchestration, policy, model gateway, brokers,
  sandboxes, telemetry, and operator interface.
- **Deployed harness:** project-owned doctrine installed into a target
  repository.
- **Target project:** customer product intent, source, tests, architecture, and
  release value.

Failures are classified before remediation as `foundation-owned`,
`deployed-owned`, or `mixed-or-unclear`. A foundation failure cannot create a
customer product Bead without an explicit ownership decision.

## Consulted sources and adaptations

`.harness/generated/mars/source-manifest.json` is the machine-readable record
of each consulted path, exact Git blob, local adaptation, generated scope, and
exclusion. MARS-3 retains durable repository records, goal-to-evidence flow,
bounded work, truthful state records, convergence, explicit roles, and the
foundation/deployed ownership distinction.

MARS-3 replaces upstream work tracking with Beads/Dolt, adapts strict-trunk to
PR-first publication, emits `FactoryDocSync`, adds the trace spine and
Rule-of-Two, and defines a provider-neutral runtime contract.

## Explicit exclusions

MARS Go source, runtime, CLI, SQLite queue, dashboard, inference lifecycle,
model downloads, provider state, GitHub or issue integration code, binary
release machinery, product routes, interface wording, prompts, and any source
not named in the generated manifest are excluded.

## Offline check and refresh

`mars3 doctrine check --repo .` validates the manifest schema, pinned commit,
license, expected paths and blobs, mapping, exclusions, and generated scope
without network access.

It also verifies the charter, detached signature, verification material,
effect chain, and signed H-001 claim attestation against foundation-pinned
digests and fingerprints. This proves internal integrity offline. Authenticity
of acquisition is rooted separately in the published signed genesis commit
`8c108460d7c0bb59b80a0b3942dc872a2e05785a` and its trusted Git remote.

An intentional refresh uses only a local MARS checkout at the exact pinned
revision:

```text
mars3 doctrine refresh --repo . --source ../MARS --ref f55d129bfc794510ca485bb54fc0a35c7b04a700 --apply
```

The refresh verifies the local revision and every required source blob before
it may atomically canonicalize only
`.harness/generated/mars/source-manifest.json`. A different revision,
incomplete source set, changed blob, or wider generated scope fails before a
write. The implementation never targets, snapshots, or restores project-owned
files; ordinary Git review remains responsible for unrelated worktree changes.

Changing the pin or semantic adaptation requires a dedicated Doctrine Update
Bead, a human-readable difference review, license review, and independent QA
and Security acceptance. Project-owned goals, prompts, roles, guardrails,
plans, decisions, features, and product code are never extractor targets.
