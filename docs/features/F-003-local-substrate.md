# F-003 — Reproducible isolated local substrate

**Status:** Backlog contract
**Goal:** G-001
**Product decisions:** PD-001, PD-003
**Product specification:** `docs/product-specs/local-substrate.md`
**Architecture:** `docs/design-docs/ADR-006-local-substrate.md`
**Work authority:** external Bead M3-P001 (display ID P-001)

## Business logic

1. The local reference environment is one named Lima instance containing one
   pinned k3s cluster; the project-owned Helm chart graph is also the hosted
   baseline.
2. Inputs are content-addressed. Exact versions, checksums, chart archives, and
   image digests come from a reviewed lock manifest; no floating tag or
   install-time version choice is allowed.
3. Identity is established by OIDC. A verified signed claim, never request
   data or model output, determines the tenant.
4. PostgreSQL row-level security is forced on tenant tables. Runtime roles do
   not own those tables and cannot bypass RLS; tenant context is scoped to one
   transaction.
5. PostgreSQL, Temporal history, and S3-compatible objects persist through
   controlled service and Lima restarts.
6. Helm lifecycle operations are idempotent, bounded to the named environment,
   and keep secret values outside Git, arguments, Helm history, telemetry, and
   evidence.
7. Service accounts, pod admission, and network policy start at least
   privilege and default deny. P-001 proves this Kubernetes baseline, not the
   later Rule-of-Two engine or hardened hostile-code sandbox.
8. Temporal schedules durable work but grants no claim, capability, review, or
   disposition authority.
9. P-001 remains an enabler: its completion cannot claim released product
   value or provider-runtime readiness.
10. Lima renders `mounts: []` and `portForwards: []`; SSH forwarding is
    disabled, and no Kubernetes API or workload service is exposed on the host.
11. Every purge requires a one-use, two-person human approval bound to the
    exact operation, instance, observed state, lock digest, requester, distinct
    approver, expiry, and idempotency key. Its ledger and final receipt survive
    outside the deletion target; public records contain only opaque references.

## Step-by-step behavior

1. The Product and Architect contracts are published in Git and linked to the
   backlog Bead. Publishing these documents neither claims M3-P001 nor changes
   its lifecycle.
2. After closed M3-H001 and accepted M3-W001 satisfy the canonical dependencies,
   the Delivery Orchestrator routes the same M3-P001 Bead to
   `platform-engineer`. The gateway's compare-and-swap claim and monotonic live
   lease are mandatory; P-001 receives no unfenced bootstrap exception. The
   Engineer revalidates that authority proof together with the goal, decisions,
   feature, scenarios, and paths.
3. The installer verifies its lock and host prerequisites before making an
   effect, creates the exact named Lima instance with literal `mounts: []` and
   `portForwards: []`, disables SSH forwarding, installs pinned k3s without
   bundled Traefik or ServiceLB, and reconciles the locked project Helm chart.
4. It generates OIDC and service credentials inside the environment without
   printing them or placing them in Helm values, then creates two reserved,
   synthetic tenant identities for conformance only.
5. It applies tenant schema and forced RLS, persistent services, restricted
   namespaces, dedicated service accounts, and default-deny policies.
6. Deterministic probes exercise OIDC-to-RLS isolation, restart recovery,
   repeated install, compatible upgrade and rollback, approval binding and
   replay denial, cleanup scope, RBAC, pod admission, network denial, and
   absence of host-facing API or workload listeners.
7. The Engineer records public-safe evidence for the immutable implementation
   commit and hands the same commit to QA, then Security. A failed review
   reopens M3-P001; it never creates a replacement ticket.
8. Only the Delivery Orchestrator records the completed run disposition and
   Git/Beads reconciliation after accepted QA and Security verdicts and merge.

## Scenario schedule

| Scenario | State | Verification owner | Required evidence |
| --- | --- | --- | --- |
| F-003-S1 | failing | QA | clean-instance one-command creation, pin/provenance, explicit Lima/k3s exposure configuration, and idempotence transcript |
| F-003-S2 | failing | QA, Security | OIDC token validation and positive/negative cross-tenant RLS matrix |
| F-003-S3 | failing | QA | PostgreSQL, Temporal, and object-store controlled-restart matrix |
| F-003-S4 | failing | QA, Security | render diff, repeated install, compatible upgrade/rollback, one-use approval matrix, secret scan, and scoped cleanup |
| F-003-S5 | failing | Security | synthetic RBAC, restricted-pod, network, namespace, forwarding, and host-boundary denial matrix |

No P-001 implementation or review evidence exists yet, so all scenarios remain
`failing`. `blocked`, `deferred`, `descoped`, or `superseded` requires a
durable reason and the authorized product/plan transition; prose in a run does
not change scenario state.

### F-003-S1 — One command creates the pinned reference environment

**Given** a clean supported host with hardware virtualization and a public
clone at an immutable commit
**And** M3-W001 is accepted and reconciled and the gateway has issued the
current M3-P001 claim and live fencing epoch
**And** the reviewed platform lock names Lima `v2.2.0`, k3s
`v1.36.3+k3s1`, Helm `v3.21.4`, Temporal chart `temporal-1.6.0`, and exact
artifact checksums and image digests
**And** no `mars3-local` instance exists
**When** the operator runs
`go run ./cmd/mars3-platform up --profile local-reference`
**Then** the command verifies every fetched artifact before execution
**And** it creates one `mars3-local` Lima instance whose reviewed configuration
contains literal `mounts: []` and `portForwards: []`, with dynamic, local, and
reverse SSH forwarding disabled
**And** the instance runs the pinned k3s and Helm versions and the locked chart
graph reaches its declared readiness conditions
**And** k3s has disabled bundled Traefik and ServiceLB and exposes neither its
API nor a workload service through `NodePort`, `LoadBalancer`, `hostPort`,
`hostNetwork`, or a host listener
**And** the result contains a bounded operation ID, lock digest, component
versions, and outcome without local identity or secret material
**And** running the same command again produces no material resource, secret,
fixture, or volume change.

### F-003-S2 — Verified identity and forced RLS isolate tenants

**Given** the local issuer has generated credentials for two reserved synthetic
subjects bound to distinct `mars3_tenant_id` values
**And** each tenant owns one synthetic row with the same public fixture key
**And** the runtime database role is not a table owner and has no `BYPASSRLS`
**When** the conformance probe validates each token's issuer, audience,
signature, expiry, subject, and tenant binding and opens a transaction
**Then** it sets tenant context locally for that transaction and returns only
the matching tenant's row
**And** a forged tenant header, query parameter, stale token, wrong audience,
cross-tenant read, cross-tenant write, direct runtime-role query without tenant
context, and pooled-connection reuse all fail closed
**And** neither raw tokens nor row values appear in evidence.

### F-003-S3 — Durable services recover from controlled restarts

**Given** the conformance suite has committed a synthetic PostgreSQL marker,
started a waiting Temporal workflow, and stored a synthetic object with a
known digest
**When** it separately restarts the PostgreSQL, Temporal, and S3-compatible
pods, rolls their deployments, and stops and starts the named Lima instance
**Then** the committed row is present under the correct tenant policy
**And** the Temporal workflow continues from persisted history without losing
or duplicating its deterministic completion marker
**And** the object bytes retain the expected digest
**And** all state comes from guest-resident persistent volumes rather than
`emptyDir` or a host mount
**And** the result makes no exactly-once claim about future external effects.

### F-003-S4 — Lifecycle operations are idempotent and secret-free

**Given** the verified baseline is healthy and all generated secrets are
referenced by name rather than embedded in charts or values
**And** a compatible reversible upgrade fixture and its previous lock revision
are available
**And** each purge approval is a one-use record bound to operation `purge`,
instance `mars3-local`, the observed lifecycle state and lock digest, the
authenticated requester, a distinct approver, an unexpired deadline, and one
idempotency key in a trusted store outside and unmounted from the deletion
target
**When** the operator repeats `up`, upgrades to the fixture, rolls back to the
previous revision, runs `down` twice, restores with `up`, and invokes
`go run ./cmd/mars3-platform purge --profile local-reference --confirm mars3-local --approval <opaque-id>`
once for the existing instance and once with a fresh approval bound to the
observed absent state
**Then** each no-op reports no drift, the upgrade reaches its verified target,
and rollback restores the prior rendered resources and durable schema contract
**And** an injected upgrade failure is atomic and leaves the prior verified
revision usable
**And** a non-reversible migration is rejected before mutation with the exact
safe next action
**And** Git, command arguments, process/environment snapshots, Helm release
history, logs, traces, and evidence contain no secret value
**And** replayed, expired, stale-state, stale-lock, wrong-operation,
wrong-instance, and same-requester/approver approvals fail before an effect
**And** a crash after redemption resumes only the same intent, reconciles the
exact deletion, and leaves a verified final receipt outside the deleted VM;
unknown or mismatched recovery blocks
**And** cleanup affects only `mars3-local`, public evidence retains only each
opaque approval reference and bounded outcome, and the separately approved
second purge succeeds as an already-absent no-op.

### F-003-S5 — Public fixtures prove the baseline isolation boundaries

**Given** two public synthetic tenant compartments and a synthetic workload
using a dedicated service account with token automount disabled
**And** their namespaces enforce the restricted pod-security profile and
default-deny ingress and egress
**When** the isolation suite attempts cross-namespace service access, public
Internet and recursive-DNS egress, Kubernetes secret reads, wildcard API
access, service-account-token access, privilege escalation, host namespaces,
host paths, added capabilities, a writable root filesystem, SSH dynamic/local/
reverse forwarding, bundled Traefik or ServiceLB exposure, `NodePort`,
`LoadBalancer`, `hostPort`, `hostNetwork`, and direct host access to the
Kubernetes API or a workload listener
**Then** every attempt is rejected by admission, authorization, or network
policy at the declared boundary
**And** the allowed internal OIDC, PostgreSQL, Temporal, and object-store routes
are the minimal explicit exceptions and still pass readiness
**And** the report contains only reserved identities, action classes, policy
names, outcome codes, and opaque operation references
**And** it states that kernel escape resistance, gVisor/Kata, transitive
Rule-of-Two admission, and effect brokering remain S-001/S-002 work.

## Permissions, validation, transitions, and failures

- M3-P001 is currently `backlog`. These Git contracts grant no claim, live
  lease, service credential, or implementation authority.
- A gateway-validated claim and current monotonic lease epoch are mandatory for
  P-001's declared implementation paths; direct Beads access is denied. Product,
  Architect, QA, and Security remain observers for implementation code unless
  separately authorized.
- Preflight validates the exact instance name, architecture, virtualization,
  disk capacity, lock signature/checksums, chart schema, dependency state,
  declared paths, and the current gateway lease epoch before mutation.
- Purge additionally validates and durably redeems a one-use human approval
  bound to the exact operation, instance, observed state and lock digest,
  authenticated requester, distinct approver, expiry, and idempotency key. Its
  ledger/receipt store is outside and unmounted from the target; crash recovery
  reconciles only the original intent. A bare opaque ID is not authority.
- Every mutating lifecycle operation is `preflight -> intent -> policy ->
  bounded effect -> receipt -> verification`. P-001 emits bounded operation
  metadata but does not claim the T-001 trace ledger exists.
- Pin mismatch, signature/checksum failure, RLS bypass, unauthorized egress,
  excessive RBAC, secret disclosure, state loss, partial rollback, or
  out-of-scope cleanup blocks the run and identifies the exact safe corrective
  action. Safety checks are never downgraded to warnings.
- One automatic retry is allowed per normalized failure fingerprint. A repeat
  records a durable blocked run disposition and requires human routing.
- Missing deterministic evidence leaves M3-P001 `in-progress`; missing QA or
  Security acceptance leaves it `in-review`; neither state is `done`.

## Evidence and falsification

The P-001 evidence manifest records the immutable commit, scenario IDs,
platform lock digest, repository-relative commands, normalized host and
architecture class, exact component versions, exit outcomes, bounded negative
test results, opaque operation IDs, only opaque purge approval references and
outcomes, reviewer IDs, residual risks, and next owner. QA reproduces F-003-S1
through S4 from a clean named instance. Security
independently reproduces S2, S4, and S5. Later D-001 dogfood/release evidence
must re-verify the substrate before product release; P-001 alone cannot satisfy
that release gate.

F-003 is falsified by any successful counterexample to S1 through S5, by
non-reproducible evidence, by a local-only fork of the hosted chart graph, by
state that survives only through a host mount, or by a claim that this baseline
implements later authority, trace, Rule-of-Two, hardened-sandbox, provider, or
hosted-production controls. A replayable or wrongly bound purge approval, lost
post-deletion receipt, cross-intent crash recovery, a non-empty Lima
mount/forward set, a host-facing cluster/service listener, or
beginning P-001 before accepted W-001 gateway fencing also falsifies the
contract.

## Out of scope

- W-001 authority gateway behavior and PostgreSQL lease semantics.
- T-001 audit ledger, OpenTelemetry/Tempo collection, and trace retention.
- S-001 transitive labels, taint propagation, and hard Rule-of-Two admission.
- S-002 gVisor/Kata sandboxes, credential proxy, tool/effect brokers, and
  deterministic publication.
- Agent runtime adapters, model transports, inference, provider keys, and the
  operator web/Electron UI.
- Real identities or tenant data, production HA, backup/restore, disaster
  recovery, destructive database migrations, hosted multi-tenant hardening,
  and product release.

## Descoped scenarios

None. Only the Product strategy authority may descope an in-scope scenario,
and it must record a superseding product decision, affected-goal analysis, and
active-plan update before the Orchestrator changes the Bead contract.
