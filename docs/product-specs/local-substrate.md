# Local substrate specification

**Status:** Planned
**Goal:** G-001
**Decision links:** PD-001, PD-003
**Feature contract:** F-003
**Work authority:** external Bead M3-P001 (display ID P-001)

## Promise

A contributor can create, verify, stop, restart, upgrade, roll back, and remove
one isolated MARS-3 reference environment from a public clone without adding a
credential or machine-specific value to Git. The environment supplies OIDC
identity, tenant-isolated PostgreSQL, Temporal, and S3-compatible storage on a
pinned Lima/k3s substrate. The same project-owned Helm chart graph is the
hosted baseline; only validated values and external-service bindings differ.

P-001 is an enabler. Passing this contract proves a reproducible substrate, not
an autonomous software factory, a hardened hostile-code sandbox, hosted
multi-tenancy, or shipped user value.

## Operator contract

The canonical local profile is managed through one project command surface:

```text
go run ./cmd/mars3-platform up --profile local-reference
go run ./cmd/mars3-platform verify --profile local-reference
go run ./cmd/mars3-platform down --profile local-reference
go run ./cmd/mars3-platform upgrade --profile local-reference --to <locked-revision>
go run ./cmd/mars3-platform rollback --profile local-reference --to <locked-revision>
go run ./cmd/mars3-platform purge --profile local-reference --confirm mars3-local --approval <opaque-id>
```

`up`, `verify`, and `down` are safe to repeat. Every `purge` invocation requires
the literal confirmation value and a one-use human approval reference. The
trusted approval record is bound to operation `purge`, exact instance
`mars3-local`, the observed lifecycle state and platform-lock digest, the
authenticated requester, a distinct approver, an expiry, and one idempotency
key. Its ledger, effect intent, reconciliation state, and final receipt live in
a trusted control-plane store outside and unmounted from the Lima deletion
target. The reference is durably redeemed before deletion; replay, expiry,
stale observed state or
lock, wrong operation or instance, and requester/approver equality all fail
before an effect. Repeating purge after absence requires a fresh approval bound
to that observed state and returns an already-absent no-op. No command may
discover a broad deletion target from a glob, a home-directory variable, or an
unresolved environment value.

A crash after redemption resumes only the same idempotent purge intent. It
reconciles the exact bound instance, completes or verifies deletion, and
persists a final receipt in the out-of-target store. It cannot turn the
redeemed approval into a new operation or observed-state authorization;
uncertain binding blocks for human recovery. The receipt remains available
after the VM and guest volumes are gone.

The first P-001 reference baseline pins:

| Component | Exact version | Contract |
| --- | --- | --- |
| Lima | `v2.2.0` | The instance has no host-home or repository mount. |
| k3s | `v1.36.3+k3s1` | The single-node local cluster enables admission and network-policy enforcement. |
| Helm | `v3.21.4` | The project chart is installed with atomic wait/rollback behavior. |
| Temporal chart | `temporal-1.6.0` | Temporal uses PostgreSQL persistence and grants no work authority. |

P-001 must add a reviewed platform lock manifest containing the exact guest
image checksums, architecture-specific binaries, chart archives, container
image digests, and public source URLs for this baseline. Floating tags,
unversioned chart repositories, and install-time version discovery are invalid.
A later pin change is an explicit reviewed platform upgrade, never an
automatic consequence of rerunning `up`.

## Work-authority dependency

M3-P001 is sequenced behind closed M3-H001 and accepted M3-W001. Their
implementation roots are disjoint. The shared `go.mod`, `go.sum`, `Makefile`,
`NOTICE`, and `THIRD_PARTY_NOTICES` paths are serialized governance surfaces:
W-001 must be accepted first, and P-001 may write them only under its later
canonical claim and lease with no concurrent W-001 lease. Path separation and
dependency order do not replace live fencing. P-001 receives no unfenced
bootstrap implementation exception.

After W-001 is accepted, direct agent access to Beads is denied and the P-001
claim must pass the authority gateway's compare-and-swap operation plus a
PostgreSQL-backed monotonic lease epoch. This contract publication neither
claims P-001 nor grants a lease.

## Identity and tenant isolation

- The local profile provides an embedded OIDC issuer and generates its signing
  material and synthetic test credentials inside the named environment. No
  token, password, signing key, client secret, cookie, or reusable credential
  enters the repository, a command argument, Helm release values, logs, traces,
  screenshots, or evidence.
- Public synthetic subjects use reserved identities and carry an immutable
  `mars3_tenant_id` claim. Runtime tenant selection comes only from a verified
  issuer, audience, signature, expiry, and subject-to-tenant binding; headers,
  query parameters, and model output cannot choose a tenant.
- Every tenant-bearing PostgreSQL table has a non-null tenant key and forced
  row-level security. The runtime role is not the table owner, has no
  `BYPASSRLS`, and sets tenant context transaction-locally after OIDC
  verification so a pooled connection cannot retain another tenant's context.
- A local-only conformance probe validates the full OIDC-to-RLS path with two
  synthetic tenants. It is not a production API and is disabled outside
  conformance profiles.

## Persistence and restart behavior

PostgreSQL, Temporal history, and S3-compatible object data use guest-resident
persistent volumes; durable state cannot use `emptyDir`. A controlled pod
restart, deployment rollout, or Lima stop/start preserves committed rows,
Temporal history and continuation, and object bytes and checksums. Temporal
activity delivery remains retryable and does not imply exactly-once external
effects. Backup, disaster recovery, and production high availability require
separate contracts.

## Isolation baseline

- The reviewed Lima configuration renders literal `mounts: []` and
  `portForwards: []`. SSH dynamic, local, and reverse forwarding are disabled;
  the Kubernetes API and every application/service route remain guest-only.
  The bounded Lima management channel may execute installer and conformance
  commands but cannot expose a guest listener on the host.
- k3s starts with the bundled Traefik and ServiceLB components disabled. The
  local profile admits no `NodePort`, `LoadBalancer`, `hostPort`, or
  `hostNetwork` exposure and proves that the host has no listener for a cluster
  API or workload service.
- System, data, and synthetic workload compartments have dedicated service
  accounts, least-privilege RBAC, disabled service-account-token automount by
  default, and no wildcard permissions.
- Namespaces enforce the Kubernetes restricted pod-security profile. Workloads
  cannot request privilege escalation, host namespaces, host paths, added
  Linux capabilities, or a writable root filesystem unless a later reviewed
  security decision grants a narrower exception.
- Network policy is default deny for ingress and egress. Workloads receive only
  declared internal service routes and an internal-only resolver that cannot
  recursively forward public names. Runtime Internet egress is absent; pinned
  public artifacts are fetched and verified by the installer boundary.
- Local fixture data is always synthetic. Public source remains
  `public+project-accepted` only after review, and external dependencies remain
  `external-untrusted`.

These controls provide the P-001 Kubernetes compartment baseline. They do not
claim the transitive Rule-of-Two admission engine scheduled for S-001 or the
gVisor/Kata hostile-code boundary and effect brokers scheduled for S-002.

## Lifecycle guarantees

Repeated installation reconciles to the same rendered resources and does not
rotate secrets, recreate durable volumes, or append duplicate fixtures.
Upgrade preflight verifies the current lock, target lock, chart schema, storage
capacity, and migration reversibility before mutation. A failed atomic upgrade
returns to the prior verified chart revision. If a database change cannot be
safely rolled back, the command fails before mutation and names the exact
supported next action.

`down` preserves the instance and durable state. `purge` removes only the named
instance, its generated secrets, and its guest-resident volumes after exact
confirmation and successful one-use approval redemption. The approval service,
not the CLI argument, holds binding fields, idempotency, redemption state,
intent, and final receipt outside the deletion target. Neither operation may
touch another Lima instance or a host directory.

## Public-safe evidence

P-001 evidence records repository-relative commands, exact lock and chart
hashes, normalized host class, component versions, exit outcomes, bounded
verification summaries, opaque operation identifiers, and only the opaque
approval reference and bounded outcome for purge. It never records approval
record contents, requester or approver identity, absolute paths, hostnames, IP
addresses, raw JWTs, secret values, SQL rows, workflow payloads, object
payloads, terminal recordings, or private telemetry.

QA must reproduce installation, idempotence, restart, upgrade, rollback, and
cleanup from a clean named instance. Security must independently exercise OIDC
claim validation, RLS bypass attempts, RBAC, pod admission, network denial,
secret non-disclosure, and scope-safe cleanup against the same immutable
commit. The required evidence manifest belongs at
`docs/evidence/P-001-validation.md`; this contract does not claim that evidence
already exists.

## Falsification

The promise is false if any baseline input floats; a clean run needs private
state; a token or secret reaches Git, Helm history, logs, or evidence; a request
can select another tenant; the runtime database role can bypass RLS; a
workload can reach another compartment or public network without an explicit
route; a controlled restart loses committed state; a repeated install drifts;
rollback leaves a partially upgraded system; purge can affect a different
instance; purge accepts a replayed, expired, stale, wrongly bound, or same-party
approval; approval/replay proof is deleted with the VM or crash recovery can
cross intents; Lima mounts a host path or forwards a guest port; k3s exposes Traefik,
ServiceLB, its API, or a workload listener on the host; the hosted baseline
requires a forked chart graph; or P-001 evidence claims T-001, S-001, S-002, or
provider-runtime controls that were not built. It is also false if
implementation begins before accepted W-001 gateway fencing or invents an
unfenced P-001 bootstrap exception.

## Out of scope

- Beads claim/lease enforcement and the authority gateway (W-001).
- The append-only trace ledger, OpenTelemetry, and Tempo (T-001).
- Transitive Rule-of-Two taint admission (S-001).
- gVisor/Kata, credential proxying, publication brokers, and hardened execution
  of hostile code (S-002).
- Model providers, model transports, runtime adapters, or provider credentials.
- The web/Electron operator interface, production publication, real tenant
  data, hosted control-plane hardening, multi-region availability, backup, and
  disaster recovery.
