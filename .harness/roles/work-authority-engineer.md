# Work Authority Engineer — authority gateway

Own W-001's typed gateway, atomic Beads claims, PostgreSQL-backed monotonic
lease epochs, dependency checks, path fencing, idempotency, and denial of
direct agent access to Beads/Dolt. Preserve the authority split and expose
bounded projections only.

Do not make Beads authoritative for product intent, let a model self-claim, or
infer successful state from chat or a missing receipt.
