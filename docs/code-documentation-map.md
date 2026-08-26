# FactoryDocSync code-documentation map

**Configuration:** `.harness/docsync.yaml`
**Marker:** `FactoryDocSync`

FactoryDocSync establishes structural ownership and review coverage. It checks
that a material source file names existing governing documentation and meets
the documentation requirements for its path prefix. It does not claim that
code or documentation is semantically correct.

## Canonical prefix map

| Source prefix | Required governing documents |
| --- | --- |
| `.github/` | F-001, ADR-004, this map |
| `cmd/mars3/` | F-001, ADR-001, this map |
| `internal/doctrine/` | F-001, MARS provenance, this map |
| `internal/trace/` | ADR-002, this map |
| `internal/policy/` | ADR-003, this map |
| `internal/runtime/` | ADR-005, this map |
| `web/`, `electron/` | operator workspace specification, this map |
| `migrations/`, `database/` | foundation specification, ADR-003, this map |

The most-specific matching prefix wins, and its complete required document set
must be present in the file marker. New source prefixes require a map and
configuration update before source is accepted.

## Marker forms

Block comment:

```text
/*
FactoryDocSync:
docs:
- docs/features/F-001-doctrine-foundation.md
- docs/design-docs/ADR-001-git-beads-authority.md
- docs/code-documentation-map.md
*/
```

YAML comment:

```text
# FactoryDocSync:
# docs:
# - docs/features/F-001-doctrine-foundation.md
# - docs/design-docs/ADR-001-git-beads-authority.md
# - docs/code-documentation-map.md
```

Compact JSON comment:

```text
/* FactoryDocSync: {"docs":["docs/features/F-001-doctrine-foundation.md","docs/design-docs/ADR-001-git-beads-authority.md","docs/code-documentation-map.md"]} */
```

## Enforcement

`mars3 docsync audit --repo .` is non-mutating and fails for a missing marker,
malformed metadata, nonexistent document, path outside the repository, or an
incomplete prefix requirement. The pre-commit, publication, and disposition
gates reject a new or materially changed governed source file until the audit
passes. Engineer, Pipeline Fixer, Dependency Manager, and Dogfood run
dispositions, QA and Security review verdicts, and Release verdicts must state
whether linked documents changed or were checked and remained current. Native
runtime file tools remain contained inside an attempt workspace; DocSync is a
publication boundary, not an unverifiable syscall claim.
