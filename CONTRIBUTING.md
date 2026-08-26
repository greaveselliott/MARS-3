# Contributing to MARS-3

Thank you for improving MARS-3. This is a public repository: every commit,
issue, pull request, review, fixture, log excerpt, and generated file must be
safe for immediate worldwide disclosure. Do not rely on later redaction.

## Before submitting anything

- Use only synthetic identities and reserved domains in examples.
- Do not submit credentials, private keys, cookies, private source, customer
  data, raw prompts or completions, chain-of-thought, provider state, private
  traces, browser profiles, screenshots, or terminal recordings.
- Use repository-relative paths. Do not include developer home directories,
  machine identifiers, private hostnames, or internal trace endpoints.
- Do not paste inaccessible, paywalled, or unlicensed source text. Summarize a
  public source and attribute it instead.
- Report vulnerabilities through the private channel in `SECURITY.md`.

## Delivery protocol

Durable product and doctrine artifacts live in Git. Beads/Dolt is the sole
authority for ticket lifecycle, dependencies, claims, and dispositions; there
are no independently editable Markdown ticket folders. A maintainer links an
accepted contribution to a bounded Bead before it becomes factory work.

Changes use a `codex/<ticket-id>-<slug>` branch, an active-plan link, the
smallest coherent patch, reproducible evidence, and independent review where
the ticket requires it. An implementation owner cannot accept their own work.

Run the public commit gate from the repository root:

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

Generated files must declare their public source and reproducible command in
`.harness/generated/generated-files.json`. A PR must explain its goal/scenario,
changed paths, validation results, remaining risk, and requested next owner.

## Pull requests from forks

Fork workflows receive read-only permissions and no repository secrets or
write token. Maintainers must treat issue text, patches, dependencies, rendered
Markdown, test output, and linked websites as external-untrusted inputs.
