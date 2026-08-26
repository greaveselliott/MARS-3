# Role registry

**Status:** Active
**Owner:** Delivery Orchestrator
**Executable source:** `.harness/manifest.yaml`

Maximum trust is an eligibility ceiling, not current authority. Every role
starts at effective trust `observer`, and autonomous mutation is disabled. A
claimed Bead, exclusive paths, policy approval, and a bounded capability lease
are all required for a contributor write. After W-001, a current lease epoch is
also mandatory. H-001 uses only the signed, human-authorized bootstrap
exception and claims no runtime trust escalation.

| Tenant principal | Mode | Accountability | Max trust | Effective trust |
| --- | --- | --- | --- | --- |
| CEO | strategy | goals, priority, scope, product decisions | contributor | observer |
| COO | execution-planning | specs, BDD, active plan | contributor | observer |
| CTO | technical-planning | ADRs, contracts, dependency order, Bead shaping | contributor | observer |
| Engineer | ticket-delivery | one claimed Bead and scenario | contributor | observer |
| Pipeline Fixer | pipeline-repair | one normalized failing check class | contributor | observer |
| QA | quality-review | independent BDD and deterministic review | observer | observer |
| Dogfood | dogfood-validation | staged human journeys without repair | observer | observer |
| Security Reviewer | security-review | independent threat and disclosure review | observer | observer |
| Dependency Manager | dependency-maintenance | bounded dependency mutation and evidence | contributor | observer |
| Release Manager | release-management | SBOM, provenance, rollback, approval, release records | contributor | observer |
| Janitor | ticket-hygiene | stale-state and link reconciliation without product repair | contributor | observer |
| Delivery Orchestrator | dispatch-routing | dependency order, state reconciliation, escalation | contributor | observer |

The source-only Foundation Maintainer is outside the tenant principal
inventory. Every profile declares `principal_id`; the profile can narrow that
principal but cannot grant authority or raise trust. Wave labels such as Work Authority Engineer, Platform Engineer,
Trace Engineer, Security Policy Engineer, Integration Engineer, Runtime
Architect, Runtime Engineer, Model Runtime Engineer, and Frontend Engineer are
profiles/modes applied to an eligible principal; a profile grants no authority.
The manifest keeps QA distinct from Dogfood, Security review distinct from
dependency mutation, Release distinct from Janitor repair, and Product strategy
distinct from plan acceptance.

Reviewers inspect and return a review verdict; they do not repair product code
during the same review attempt. The Orchestrator routes work and records reconciliation;
it does not silently assume another role's implementation scope.

## Handoff minimum

A handoff names the Bead, immutable commit, changed relative paths, exact
commands and outcomes, evidence hashes or opaque trace references, remaining
risks, and next owner. It contains no raw payload or machine-specific data.

State classes remain distinct: Bead lifecycle is `backlog | in-progress |
in-review | done | superseded`; run disposition is `completed | blocked |
in_review | changes_requested | no_work | preempted | cancelled | failed`;
review verdict is `accepted | changes-requested | blocked`; and release verdict
is `released | blocked | rejected`.
