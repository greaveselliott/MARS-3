# Quality and release-readiness ledger

**Updated:** 2026-08-26
**Owner:** Delivery Orchestrator
**Current release readiness:** Foundation accepted; product runtime not releasable

This ledger is evidence-backed. A missing result is `pending`, not an inferred
pass and not a numeric score.

| Feature/scenario | Mode | Failure ownership | Evidence state | Review state |
| --- | --- | --- | --- | --- |
| F-001-S1 authority route | doctrine-foundation | foundation | H-001-E7: exact ordered chain, registry routing, and signed-claim lineage passed | QA accepted exact E7 |
| F-001-S2 provenance | doctrine-foundation | foundation | H-001-E7: 20-source offline refresh passed | QA accepted exact E7 |
| F-001-S3 trust defaults | security-review | foundation | H-001-E7: profile mutation denial passed | Security accepted exact E7 |
| F-001-S4 public safety | security-review | foundation | H-001-E7: exact workflow, public, secret, and reproducible-build checks passed | Security accepted exact E7 |
| F-002-S1–S6 work authority | authority-gateway | foundation | Contract publication in progress; no implementation claim or runtime evidence; signed recovery snapshot and disposition validate locally | Pending QA/Security disposition |
| F-003-S1–S5 local substrate | substrate-delivery | foundation | P-001 remains backlog behind W-001; exact prospective description correction passed read-back; earlier authority intervention remains non-retroactive | Pending QA/Security disposition |
| Wave-1 publication provenance | contract-publication | foundation | Every feature commit and the reviewed tree must be retained by the signed publication tag; branch and scanner protections remain unchanged | Pending immutable tag/PR evidence |

## Release gate

All F-001 scenarios pass; QA and Security accepted the same immutable E7
commit, and the Orchestrator reconciled the verified public merge to closed
M3-H001. This establishes the doctrine foundation only. F-002 and F-003 have
no runtime evidence, so no agent mutation service, local substrate, or
production release is authorized.
