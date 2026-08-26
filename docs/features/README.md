# BDD feature contracts

BDD contracts are authoritative for business logic and observable behavior.
Every scenario has a stable identifier and one state: `failing`, `passing`,
`blocked`, `deferred`, `descoped`, or `superseded`.

A feature cannot release while an in-scope scenario is not `passing`, unless a
new Product decision explicitly descopes or supersedes it. A Bead implements
only the active plan's current failing scenario or an explicitly grouped set.

| Contract | Status | Current work |
| --- | --- | --- |
| [F-001](F-001-doctrine-foundation.md) | Passing | H-001 merged and reconciled. |
| [F-002](F-002-work-authority.md) | Failing | W-001 selected; contract publication precedes claim. |
| [F-003](F-003-local-substrate.md) | Failing | P-001 backlog; may be claimed after its contract is accepted. |
