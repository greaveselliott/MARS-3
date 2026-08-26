# BDD feature contracts

BDD contracts are authoritative for business logic and observable behavior.
Every scenario has a stable identifier and one state: `failing`, `passing`,
`blocked`, `deferred`, `descoped`, or `superseded`.

A feature cannot release while an in-scope scenario is not `passing`, unless a
new Product decision explicitly descopes or supersedes it. A Bead implements
only the active plan's current failing scenario or an explicitly grouped set.
