/*
FactoryDocSync:
docs:
- docs/features/F-002-work-authority.md
- docs/design-docs/ADR-001-git-beads-authority.md
- docs/code-documentation-map.md
*/

package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	authorityv1 "github.com/greaveselliott/MARS-3/api/authority/v1"
)

const maximumRenewalDuration = 15 * time.Minute

// GetLease returns a bounded lease projection. Expired time makes Active false
// even if a maintenance transaction has not yet persisted the terminal state.
func (store *Store) GetLease(ctx context.Context, tenantID, projectID, leaseID string) (authorityv1.CapabilityLease, error) {
	tx, err := store.beginScoped(ctx, tenantID, projectID, pgx.RepeatableRead)
	if err != nil {
		return authorityv1.CapabilityLease{}, err
	}
	defer rollback(tx, ctx)
	if _, err := store.verifyProject(ctx, tx, tenantID, projectID, false); err != nil {
		return authorityv1.CapabilityLease{}, err
	}
	lease, err := loadLease(ctx, tx, tenantID, projectID, leaseID, false)
	if errors.Is(err, pgx.ErrNoRows) {
		return authorityv1.CapabilityLease{}, ErrLeaseNotFound
	}
	if err != nil {
		return authorityv1.CapabilityLease{}, err
	}
	if !lease.ExpiresAt.After(store.now().UTC()) {
		lease.Active = false
		lease.State = authorityv1.LeaseExpired
	}
	if err := tx.Commit(ctx); err != nil {
		return authorityv1.CapabilityLease{}, err
	}
	return lease, nil
}

// Renew extends only the owning, unexpired exact lease tuple. Generation,
// epoch, base SHA, capability, paths, labels, and claim version cannot change.
func (store *Store) Renew(ctx context.Context, fence authorityv1.FencingTuple, newExpiry time.Time) (authorityv1.CapabilityLease, error) {
	return store.mutateOwnedLease(ctx, fence, "renew", "", newExpiry.UTC())
}

// Release terminates only the exact owning lease tuple.
func (store *Store) Release(ctx context.Context, fence authorityv1.FencingTuple) (authorityv1.CapabilityLease, error) {
	return store.mutateOwnedLease(ctx, fence, "release", "lease.released", time.Time{})
}

// Revoke terminates a lease by exact generation and epoch. Authorization for
// the orchestrator/security principal is enforced by the gateway before this
// narrow store method is called.
func (store *Store) Revoke(ctx context.Context, request authorityv1.RevokeLeaseRequest) (authorityv1.CapabilityLease, error) {
	if request.TenantID == "" || request.ProjectID == "" || request.LeaseID == "" || request.FenceGeneration == "" || request.LeaseEpoch == 0 || request.Reason == "" || len(request.Reason) > 128 {
		return authorityv1.CapabilityLease{}, ErrLeaseFence
	}
	tx, err := store.beginScoped(ctx, request.TenantID, request.ProjectID, pgx.ReadCommitted)
	if err != nil {
		return authorityv1.CapabilityLease{}, err
	}
	defer rollback(tx, ctx)
	generation, err := store.verifyProject(ctx, tx, request.TenantID, request.ProjectID, true)
	if err != nil {
		return authorityv1.CapabilityLease{}, err
	}
	if generation != request.FenceGeneration {
		return authorityv1.CapabilityLease{}, ErrLeaseFence
	}
	lease, err := loadLease(ctx, tx, request.TenantID, request.ProjectID, request.LeaseID, true)
	if errors.Is(err, pgx.ErrNoRows) {
		return authorityv1.CapabilityLease{}, ErrLeaseNotFound
	}
	if err != nil {
		return authorityv1.CapabilityLease{}, err
	}
	if lease.FenceGeneration != request.FenceGeneration || lease.LeaseEpoch != request.LeaseEpoch {
		return authorityv1.CapabilityLease{}, ErrLeaseFence
	}
	if lease.Active {
		if _, err := tx.Exec(ctx, `
			update mars3_authority.leases
			set state='revoked', terminal_reason=$1, updated_at=$2
			where tenant_id=$3 and project_id=$4 and lease_id=$5 and state='active'`,
			request.Reason, store.now().UTC(), request.TenantID, request.ProjectID, request.LeaseID); err != nil {
			return authorityv1.CapabilityLease{}, err
		}
		lease.Active = false
		lease.State = authorityv1.LeaseRevoked
	}
	if err := tx.Commit(ctx); err != nil {
		return authorityv1.CapabilityLease{}, err
	}
	return lease, nil
}

// ValidateFence performs the PostgreSQL half of just-in-time effect admission.
// The gateway must additionally re-read and match canonical Beads claim state.
func (store *Store) ValidateFence(ctx context.Context, fence authorityv1.FencingTuple) (authorityv1.CapabilityLease, error) {
	tx, err := store.beginScoped(ctx, fence.TenantID, fence.ProjectID, pgx.ReadCommitted)
	if err != nil {
		return authorityv1.CapabilityLease{}, err
	}
	defer rollback(tx, ctx)
	if _, err := store.verifyProject(ctx, tx, fence.TenantID, fence.ProjectID, true); err != nil {
		return authorityv1.CapabilityLease{}, err
	}
	if err := expireProjectLeases(ctx, tx, fence.TenantID, fence.ProjectID, store.now().UTC()); err != nil {
		return authorityv1.CapabilityLease{}, err
	}
	lease, err := loadLease(ctx, tx, fence.TenantID, fence.ProjectID, fence.LeaseID, true)
	if errors.Is(err, pgx.ErrNoRows) {
		return authorityv1.CapabilityLease{}, ErrLeaseNotFound
	}
	if err != nil {
		return authorityv1.CapabilityLease{}, err
	}
	if !lease.Active || !lease.ExpiresAt.After(store.now().UTC()) {
		if err := tx.Commit(ctx); err != nil {
			return authorityv1.CapabilityLease{}, err
		}
		return authorityv1.CapabilityLease{}, ErrLeaseFence
	}
	if !leaseMatchesFence(lease, fence) {
		return authorityv1.CapabilityLease{}, ErrLeaseFence
	}
	if err := tx.Commit(ctx); err != nil {
		return authorityv1.CapabilityLease{}, err
	}
	return lease, nil
}

func (store *Store) mutateOwnedLease(ctx context.Context, fence authorityv1.FencingTuple, action, reason string, newExpiry time.Time) (authorityv1.CapabilityLease, error) {
	tx, err := store.beginScoped(ctx, fence.TenantID, fence.ProjectID, pgx.ReadCommitted)
	if err != nil {
		return authorityv1.CapabilityLease{}, err
	}
	defer rollback(tx, ctx)
	if _, err := store.verifyProject(ctx, tx, fence.TenantID, fence.ProjectID, true); err != nil {
		return authorityv1.CapabilityLease{}, err
	}
	now := store.now().UTC()
	if err := expireProjectLeases(ctx, tx, fence.TenantID, fence.ProjectID, now); err != nil {
		return authorityv1.CapabilityLease{}, err
	}
	lease, err := loadLease(ctx, tx, fence.TenantID, fence.ProjectID, fence.LeaseID, true)
	if errors.Is(err, pgx.ErrNoRows) {
		return authorityv1.CapabilityLease{}, ErrLeaseNotFound
	}
	if err != nil {
		return authorityv1.CapabilityLease{}, err
	}
	if !lease.Active || !lease.ExpiresAt.After(now) {
		if err := tx.Commit(ctx); err != nil {
			return authorityv1.CapabilityLease{}, err
		}
		return authorityv1.CapabilityLease{}, ErrLeaseFence
	}
	if !leaseMatchesFence(lease, fence) {
		return authorityv1.CapabilityLease{}, ErrLeaseFence
	}
	switch action {
	case "renew":
		if !newExpiry.After(lease.ExpiresAt) || newExpiry.After(now.Add(maximumRenewalDuration)) {
			return authorityv1.CapabilityLease{}, ErrLeaseFence
		}
		if _, err := tx.Exec(ctx, `
			update mars3_authority.leases set expires_at=$1, updated_at=$2
			where tenant_id=$3 and project_id=$4 and lease_id=$5 and state='active'`,
			newExpiry, now, fence.TenantID, fence.ProjectID, fence.LeaseID); err != nil {
			return authorityv1.CapabilityLease{}, err
		}
		lease.ExpiresAt = newExpiry
	case "release":
		if _, err := tx.Exec(ctx, `
			update mars3_authority.leases
			set state='released', terminal_reason=$1, updated_at=$2
			where tenant_id=$3 and project_id=$4 and lease_id=$5 and state='active'`,
			reason, now, fence.TenantID, fence.ProjectID, fence.LeaseID); err != nil {
			return authorityv1.CapabilityLease{}, err
		}
		lease.Active = false
		lease.State = authorityv1.LeaseReleased
	default:
		return authorityv1.CapabilityLease{}, ErrLeaseFence
	}
	if err := tx.Commit(ctx); err != nil {
		return authorityv1.CapabilityLease{}, err
	}
	return lease, nil
}

func leaseMatchesFence(lease authorityv1.CapabilityLease, fence authorityv1.FencingTuple) bool {
	return lease.TenantID == fence.TenantID && lease.ProjectID == fence.ProjectID && lease.BeadID == fence.BeadID && lease.AttemptID == fence.AttemptID && lease.IdempotencyKey == fence.IdempotencyKey && lease.LeaseID == fence.LeaseID && lease.FenceGeneration == fence.FenceGeneration && lease.LeaseEpoch == fence.LeaseEpoch && lease.ClaimVersion == fence.ClaimVersion && lease.BaseSHA == fence.BaseSHA && lease.Capability == fence.Capability && equalStrings(lease.ExclusivePaths, fence.ExclusivePaths) && equalLabels(lease.Labels, fence.Labels)
}

func expireProjectLeases(ctx context.Context, tx pgx.Tx, tenantID, projectID string, now time.Time) error {
	_, err := tx.Exec(ctx, `
		update mars3_authority.leases
		set state='expired', terminal_reason='lease.expired', updated_at=$1
		where tenant_id=$2 and project_id=$3 and state='active' and expires_at <= $1`,
		now, tenantID, projectID)
	return err
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func equalLabels(left, right []authorityv1.Label) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
