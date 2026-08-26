# Foundation specification

**Status:** Current
**Goal:** G-001
**Decision links:** PD-001, PD-002, PD-003
**Feature contract:** F-001

H-001 completed this specification on 2026-08-26. The validator remains a
required publication gate while later features add runtime surfaces.

## Promise

A public clone makes the MARS-3 delivery route, authority split, trust posture,
source provenance, trace contract, and security constraints inspectable without
network access or private operational state.

## Public commit gate

Run from the repository root:

```text
mars3 doctrine check --repo .
mars3 plan check --repo .
mars3 docsync audit --repo .
mars3 public-check --repo .
go test ./...
go vet ./...
git diff --check
gitleaks detect --no-git --source . --redact --no-banner
gitleaks detect --source . --redact --no-banner
```

Equivalent `go run ./cmd/mars3` invocation is permitted before installing the
binary. The scanner is pinned to the qualified v8.18.4 artifact; its synthetic
canary must fail closed. Every command must return zero before a commit or push.

## Boundaries

H-001 contains validation and doctrine only. It did not introduce MARS runtime
code, a provider adapter, model inference, an agent sandbox, the production
control plane, or the operator interface. Later Beads may add those surfaces
only under their own contracts and evidence.

Generated provenance has a declared public source and command. Only
`.harness/generated/mars/source-manifest.json` may be changed by a MARS doctrine
refresh; every other project artifact remains project-owned.
