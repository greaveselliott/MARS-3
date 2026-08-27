-- FactoryDocSync:
-- docs:
-- - docs/product-specs/foundation.md
-- - docs/features/F-002-work-authority.md
-- - docs/design-docs/ADR-001-git-beads-authority.md
-- - docs/design-docs/ADR-003-rule-of-two.md
-- - docs/code-documentation-map.md

begin;

create schema if not exists mars3_authority;

create table if not exists mars3_authority.projects (
    tenant_id text not null,
    project_id text not null,
    fence_generation text not null,
    issuance_enabled boolean not null default false,
    generation_anchored_at timestamptz not null,
    primary key (tenant_id, project_id),
    check (tenant_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
    check (project_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
    check (fence_generation ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$')
);

create table if not exists mars3_authority.claim_sagas (
    tenant_id text not null,
    project_id text not null,
    idempotency_key text not null,
    request_digest text not null,
    phase text not null,
    bead_id text not null,
    attempt_id text not null,
    base_sha text not null,
    capability text not null,
    exclusive_paths text[] not null,
    labels text[] not null,
    trace_ref text not null,
    work_json jsonb,
    lease_id text,
    receipt_ref text,
    created_at timestamptz not null,
    updated_at timestamptz not null,
    primary key (tenant_id, project_id, idempotency_key),
    check (phase in ('intent-recorded', 'canonical-claimed', 'complete')),
    check (request_digest ~ '^[a-f0-9]{64}$'),
    check (idempotency_key ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
    check (bead_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
    check (attempt_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
    check (base_sha ~ '^([a-f0-9]{40}|[a-f0-9]{64})$'),
    check (capability ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
    check (trace_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
    check (cardinality(exclusive_paths) between 1 and 64),
    check (array_position(exclusive_paths, null) is null),
    check (cardinality(labels) between 1 and 64),
    check (array_position(labels, null) is null),
    check ((phase = 'intent-recorded' and work_json is null) or (phase <> 'intent-recorded' and jsonb_typeof(work_json) = 'object')),
    check ((phase = 'complete') = (lease_id is not null and receipt_ref is not null))
);

create table if not exists mars3_authority.lease_epochs (
    tenant_id text not null,
    project_id text not null,
    fence_generation text not null,
    last_epoch bigint not null default 0,
    primary key (tenant_id, project_id, fence_generation),
    check (fence_generation ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
    check (last_epoch >= 0)
);

create table if not exists mars3_authority.leases (
    tenant_id text not null,
    project_id text not null,
    lease_id text not null,
    bead_id text not null,
    attempt_id text not null,
    idempotency_key text not null,
    fence_generation text not null,
    lease_epoch bigint not null,
    claim_version jsonb not null,
    base_sha text not null,
    capability text not null,
    exclusive_paths text[] not null,
    labels text[] not null,
    issued_at timestamptz not null,
    expires_at timestamptz not null,
    state text not null,
    terminal_reason text,
    updated_at timestamptz not null,
    primary key (tenant_id, project_id, lease_id),
    unique (tenant_id, project_id, idempotency_key),
    unique (tenant_id, project_id, fence_generation, lease_epoch),
    check (lease_epoch > 0),
    check (lease_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
    check (bead_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
    check (attempt_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
    check (idempotency_key ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
    check (fence_generation ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
    check (base_sha ~ '^([a-f0-9]{40}|[a-f0-9]{64})$'),
    check (capability ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
    check (jsonb_typeof(claim_version) = 'object'),
    check (expires_at > issued_at),
    check (state in ('active', 'released', 'revoked', 'expired')),
    check (cardinality(exclusive_paths) between 1 and 64),
    check (array_position(exclusive_paths, null) is null),
    check (cardinality(labels) between 1 and 64),
    check (array_position(labels, null) is null)
);

create unique index if not exists leases_one_active_bead
    on mars3_authority.leases (tenant_id, project_id, bead_id)
    where state = 'active';

create unique index if not exists leases_one_active_attempt
    on mars3_authority.leases (tenant_id, project_id, attempt_id)
    where state = 'active';

create index if not exists leases_active_expiry
    on mars3_authority.leases (tenant_id, project_id, expires_at)
    where state = 'active';

alter table mars3_authority.projects enable row level security;
alter table mars3_authority.projects force row level security;
alter table mars3_authority.claim_sagas enable row level security;
alter table mars3_authority.claim_sagas force row level security;
alter table mars3_authority.lease_epochs enable row level security;
alter table mars3_authority.lease_epochs force row level security;
alter table mars3_authority.leases enable row level security;
alter table mars3_authority.leases force row level security;

drop policy if exists tenant_project_projects on mars3_authority.projects;
create policy tenant_project_projects on mars3_authority.projects
    using (
        tenant_id = current_setting('mars3.tenant_id', true)
        and project_id = current_setting('mars3.project_id', true)
    )
    with check (
        tenant_id = current_setting('mars3.tenant_id', true)
        and project_id = current_setting('mars3.project_id', true)
    );

drop policy if exists tenant_project_claim_sagas on mars3_authority.claim_sagas;
create policy tenant_project_claim_sagas on mars3_authority.claim_sagas
    using (
        tenant_id = current_setting('mars3.tenant_id', true)
        and project_id = current_setting('mars3.project_id', true)
    )
    with check (
        tenant_id = current_setting('mars3.tenant_id', true)
        and project_id = current_setting('mars3.project_id', true)
    );

drop policy if exists tenant_project_lease_epochs on mars3_authority.lease_epochs;
create policy tenant_project_lease_epochs on mars3_authority.lease_epochs
    using (
        tenant_id = current_setting('mars3.tenant_id', true)
        and project_id = current_setting('mars3.project_id', true)
    )
    with check (
        tenant_id = current_setting('mars3.tenant_id', true)
        and project_id = current_setting('mars3.project_id', true)
    );

drop policy if exists tenant_project_leases on mars3_authority.leases;
create policy tenant_project_leases on mars3_authority.leases
    using (
        tenant_id = current_setting('mars3.tenant_id', true)
        and project_id = current_setting('mars3.project_id', true)
    )
    with check (
        tenant_id = current_setting('mars3.tenant_id', true)
        and project_id = current_setting('mars3.project_id', true)
    );

commit;
