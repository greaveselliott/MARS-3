# MARS-3

MARS-3 is a public-first software factory for governed, provider-agnostic
coding agents. It will support contained Codex and Claude agent-loop adapters
alongside a project-owned native harness for qualified open model runtimes.

This repository is being built through one active execution plan and bounded
Beads work items. The doctrine foundation is complete; Wave 1 publishes the
work-authority and local-substrate contracts before either implementation is
claimed.

## Authority and delivery

- Git `main` is accepted truth for code, goals, decisions, BDD contracts,
  plans, ADRs, and redacted evidence manifests.
- Beads/Dolt is authoritative for work items, dependency order, claims,
  declared exclusive paths, handoffs, and dispositions. Its database is deliberately outside
  this public repository.
- Chat and provider sessions coordinate attempts; neither can change durable
  intent or close work by itself.
- Every principal begins with effective observer trust. Mutation requires a
  bounded claim and later runtime enforcement.

The required chain is:

```text
goal → decision/spec → BDD → active plan → claimed Bead → implementation
     → evidence → QA → Security → disposition
```

## Foundation checks

The H-001 command surface is deliberately offline and standard-library-only:

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
```

See `CONTRIBUTING.md` before opening an issue or pull request. Every submitted
artifact is presumed immediately public. Security vulnerabilities belong in
GitHub private vulnerability reporting, as described in `SECURITY.md`.

Status: H-001 is merged and reconciled. W-001 is selected in
`contract-publication` and remains unclaimed until its Git-owned contract is
accepted. No provider adapter or production security boundary is implemented
yet.

## License

Apache License 2.0. See `LICENSE`, `NOTICE`, and
`THIRD_PARTY_NOTICES`.
