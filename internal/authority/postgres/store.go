/*
FactoryDocSync:
docs:
- docs/features/F-002-work-authority.md
- docs/design-docs/ADR-001-git-beads-authority.md
- docs/code-documentation-map.md
*/

// Package postgres implements the operational lease and claim-saga authority.
// It never stores canonical ticket lifecycle or product intent.
package postgres

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	authorityv1 "github.com/greaveselliott/MARS-3/api/authority/v1"
	"github.com/greaveselliott/MARS-3/internal/authority/gateway"
)

var (
	ErrProjectNotProvisioned = errors.New("authority project not provisioned")
	ErrFenceGeneration       = errors.New("fence generation mismatch")
	ErrIssuanceDisabled      = errors.New("lease issuance disabled")
	ErrLeaseConflict         = errors.New("exclusive path lease conflict")
	ErrLeaseNotFound         = errors.New("lease not found")
	ErrLeaseFence            = errors.New("lease fence mismatch")
	ErrProjectBarrier        = errors.New("authority project rebaseline barrier active")
)

// Beginner is the narrow pgx pool surface used by Store.
type Beginner interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// GenerationResolver returns the externally anchored, non-reusable fence
// generation expected for a tenant/project. Database state cannot override it.
type GenerationResolver func(context.Context, string, string) (string, error)

type IDSource func(string) (string, error)

type Store struct {
	db         Beginner
	generation GenerationResolver
	now        func() time.Time
	newID      IDSource
}

func New(db Beginner, generation GenerationResolver, now func() time.Time, newID IDSource) (*Store, error) {
	if db == nil || generation == nil || now == nil {
		return nil, errors.New("postgres authority dependencies are required")
	}
	if newID == nil {
		newID = randomID
	}
	return &Store{db: db, generation: generation, now: now, newID: newID}, nil
}

// ProvisionProject installs a previously authorized generation anchor. It does
// not rotate or re-enable an existing project and is not an agent-facing API.
func (store *Store) ProvisionProject(ctx context.Context, tenantID, projectID string) error {
	generation, err := store.generation(ctx, tenantID, projectID)
	if err != nil || generation == "" {
		return ErrFenceGeneration
	}
	tx, err := store.beginScoped(ctx, tenantID, projectID, pgx.Serializable)
	if err != nil {
		return err
	}
	defer rollback(tx, ctx)
	now := store.now().UTC()
	command, err := tx.Exec(ctx, `
		insert into mars3_authority.projects
		    (tenant_id, project_id, fence_generation, issuance_enabled, generation_anchored_at)
		values ($1, $2, $3, true, $4)
		on conflict (tenant_id, project_id) do nothing`, tenantID, projectID, generation, now)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		var existing string
		if err := tx.QueryRow(ctx, `select fence_generation from mars3_authority.projects where tenant_id=$1 and project_id=$2`, tenantID, projectID).Scan(&existing); err != nil {
			return err
		}
		if existing != generation {
			return ErrFenceGeneration
		}
	}
	return tx.Commit(ctx)
}

func (store *Store) Lookup(ctx context.Context, tenantID, projectID, key string) (gateway.ClaimSaga, bool, error) {
	tx, err := store.beginScoped(ctx, tenantID, projectID, pgx.RepeatableRead)
	if err != nil {
		return gateway.ClaimSaga{}, false, err
	}
	defer rollback(tx, ctx)
	if _, err := store.verifyProject(ctx, tx, tenantID, projectID, false); err != nil {
		return gateway.ClaimSaga{}, false, err
	}
	saga, err := loadSaga(ctx, tx, tenantID, projectID, key, false)
	if errors.Is(err, pgx.ErrNoRows) {
		return gateway.ClaimSaga{}, false, tx.Commit(ctx)
	}
	if err != nil {
		return gateway.ClaimSaga{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return gateway.ClaimSaga{}, false, err
	}
	return saga, true, nil
}

func (store *Store) Begin(ctx context.Context, intent gateway.ClaimIntent) (gateway.ClaimSaga, error) {
	tx, err := store.beginScoped(ctx, intent.TenantID, intent.ProjectID, pgx.Serializable)
	if err != nil {
		return gateway.ClaimSaga{}, err
	}
	defer rollback(tx, ctx)
	if _, err := store.verifyProject(ctx, tx, intent.TenantID, intent.ProjectID, true); err != nil {
		return gateway.ClaimSaga{}, err
	}
	now := store.now().UTC()
	paths := append([]string(nil), intent.ExclusivePaths...)
	labels := labelStrings(intent.Labels)
	_, err = tx.Exec(ctx, `
		insert into mars3_authority.claim_sagas
		    (tenant_id, project_id, idempotency_key, request_digest, phase, bead_id,
		     attempt_id, base_sha, capability, exclusive_paths, labels, trace_ref,
		     created_at, updated_at)
		values ($1,$2,$3,$4,'intent-recorded',$5,$6,$7,$8,$9,$10,$11,$12,$12)
		on conflict (tenant_id, project_id, idempotency_key) do nothing`,
		intent.TenantID, intent.ProjectID, intent.IdempotencyKey, intent.RequestDigest,
		intent.BeadID, intent.AttemptID, intent.BaseSHA, string(intent.Capability), paths,
		labels, intent.TraceRef, now)
	if err != nil {
		return gateway.ClaimSaga{}, err
	}
	saga, err := loadSaga(ctx, tx, intent.TenantID, intent.ProjectID, intent.IdempotencyKey, true)
	if err != nil {
		return gateway.ClaimSaga{}, err
	}
	if saga.RequestDigest != intent.RequestDigest {
		return gateway.ClaimSaga{}, gateway.ErrIdempotencyConflict
	}
	if err := tx.Commit(ctx); err != nil {
		return gateway.ClaimSaga{}, err
	}
	return saga, nil
}

func (store *Store) MarkCanonicalClaimed(ctx context.Context, tenantID, projectID, key, digest string, work authorityv1.WorkItem) (gateway.ClaimSaga, error) {
	workJSON, err := json.Marshal(work)
	if err != nil {
		return gateway.ClaimSaga{}, err
	}
	tx, err := store.beginScoped(ctx, tenantID, projectID, pgx.Serializable)
	if err != nil {
		return gateway.ClaimSaga{}, err
	}
	defer rollback(tx, ctx)
	if _, err := store.verifyProject(ctx, tx, tenantID, projectID, true); err != nil {
		return gateway.ClaimSaga{}, err
	}
	saga, err := loadSaga(ctx, tx, tenantID, projectID, key, true)
	if err != nil {
		return gateway.ClaimSaga{}, err
	}
	if saga.RequestDigest != digest {
		return gateway.ClaimSaga{}, gateway.ErrIdempotencyConflict
	}
	if saga.Phase == gateway.ClaimPhaseCanonical || saga.Phase == gateway.ClaimPhaseComplete {
		if !sameStoredWork(saga.Work, work) {
			return gateway.ClaimSaga{}, gateway.ErrIdempotencyConflict
		}
		if err := tx.Commit(ctx); err != nil {
			return gateway.ClaimSaga{}, err
		}
		return saga, nil
	}
	if saga.Phase != gateway.ClaimPhaseIntent {
		return gateway.ClaimSaga{}, gateway.ErrIdempotencyConflict
	}
	command, err := tx.Exec(ctx, `
		update mars3_authority.claim_sagas
		set phase='canonical-claimed', work_json=$1, updated_at=$2
		where tenant_id=$3 and project_id=$4 and idempotency_key=$5
		  and request_digest=$6 and phase='intent-recorded'`,
		string(workJSON), store.now().UTC(), tenantID, projectID, key, digest)
	if err != nil {
		return gateway.ClaimSaga{}, err
	}
	if command.RowsAffected() != 1 {
		return gateway.ClaimSaga{}, gateway.ErrIdempotencyConflict
	}
	saga, err = loadSaga(ctx, tx, tenantID, projectID, key, true)
	if err != nil {
		return gateway.ClaimSaga{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return gateway.ClaimSaga{}, err
	}
	return saga, nil
}

func (store *Store) IssueLease(ctx context.Context, key, digest string, request gateway.LeaseRequest) (gateway.ClaimSaga, error) {
	tx, err := store.beginScoped(ctx, request.TenantID, request.ProjectID, pgx.ReadCommitted)
	if err != nil {
		return gateway.ClaimSaga{}, err
	}
	defer rollback(tx, ctx)
	generation, err := store.verifyProject(ctx, tx, request.TenantID, request.ProjectID, true)
	if err != nil {
		return gateway.ClaimSaga{}, err
	}
	saga, err := loadSaga(ctx, tx, request.TenantID, request.ProjectID, key, true)
	if err != nil {
		return gateway.ClaimSaga{}, err
	}
	if saga.RequestDigest != digest {
		return gateway.ClaimSaga{}, gateway.ErrIdempotencyConflict
	}
	if saga.Phase == gateway.ClaimPhaseComplete {
		if err := tx.Commit(ctx); err != nil {
			return gateway.ClaimSaga{}, err
		}
		return saga, nil
	}
	if saga.Phase != gateway.ClaimPhaseCanonical || !sameLeaseRequest(saga, request) {
		return gateway.ClaimSaga{}, gateway.ErrIdempotencyConflict
	}

	now := store.now().UTC()
	if !request.MaximumExpiry.After(now) || request.MaximumExpiry.After(now.Add(15*time.Minute)) {
		return gateway.ClaimSaga{}, ErrLeaseFence
	}
	if _, err := tx.Exec(ctx, `
		update mars3_authority.leases
		set state='expired', terminal_reason='lease.expired', updated_at=$1
		where tenant_id=$2 and project_id=$3 and state='active' and expires_at <= $1`,
		now, request.TenantID, request.ProjectID); err != nil {
		return gateway.ClaimSaga{}, err
	}
	rows, err := tx.Query(ctx, `
		select exclusive_paths from mars3_authority.leases
		where tenant_id=$1 and project_id=$2 and state='active'
		order by lease_id for update`, request.TenantID, request.ProjectID)
	if err != nil {
		return gateway.ClaimSaga{}, err
	}
	for rows.Next() {
		var activePaths []string
		if err := rows.Scan(&activePaths); err != nil {
			rows.Close()
			return gateway.ClaimSaga{}, err
		}
		if pathsOverlap(activePaths, request.ExclusivePaths) {
			rows.Close()
			return gateway.ClaimSaga{}, ErrLeaseConflict
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return gateway.ClaimSaga{}, err
	}
	rows.Close()

	if _, err := tx.Exec(ctx, `
		insert into mars3_authority.lease_epochs
		    (tenant_id, project_id, fence_generation, last_epoch)
		values ($1,$2,$3,0)
		on conflict (tenant_id, project_id, fence_generation) do nothing`,
		request.TenantID, request.ProjectID, generation); err != nil {
		return gateway.ClaimSaga{}, err
	}
	var lastEpoch uint64
	if err := tx.QueryRow(ctx, `
		select last_epoch from mars3_authority.lease_epochs
		where tenant_id=$1 and project_id=$2 and fence_generation=$3
		for update`, request.TenantID, request.ProjectID, generation).Scan(&lastEpoch); err != nil {
		return gateway.ClaimSaga{}, err
	}
	nextEpoch := lastEpoch + 1
	if nextEpoch == 0 {
		return gateway.ClaimSaga{}, ErrLeaseFence
	}
	if _, err := tx.Exec(ctx, `
		update mars3_authority.lease_epochs set last_epoch=$1
		where tenant_id=$2 and project_id=$3 and fence_generation=$4`,
		nextEpoch, request.TenantID, request.ProjectID, generation); err != nil {
		return gateway.ClaimSaga{}, err
	}

	leaseID, err := store.newID("lease")
	if err != nil {
		return gateway.ClaimSaga{}, err
	}
	receiptRef, err := store.newID("receipt")
	if err != nil {
		return gateway.ClaimSaga{}, err
	}
	claimVersion, err := json.Marshal(request.ClaimVersion)
	if err != nil {
		return gateway.ClaimSaga{}, err
	}
	labels := labelStrings(request.Labels)
	if _, err := tx.Exec(ctx, `
		insert into mars3_authority.leases
		    (tenant_id, project_id, lease_id, bead_id, attempt_id, idempotency_key,
		     fence_generation, lease_epoch, claim_version, base_sha, capability,
		     exclusive_paths, labels, issued_at, expires_at, state, updated_at)
		values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,'active',$14)`,
		request.TenantID, request.ProjectID, leaseID, request.BeadID, request.AttemptID,
		request.IdempotencyKey, generation, nextEpoch, string(claimVersion), request.BaseSHA,
		string(request.Capability), request.ExclusivePaths, labels, now, request.MaximumExpiry); err != nil {
		return gateway.ClaimSaga{}, err
	}
	command, err := tx.Exec(ctx, `
		update mars3_authority.claim_sagas
		set phase='complete', lease_id=$1, receipt_ref=$2, updated_at=$3
		where tenant_id=$4 and project_id=$5 and idempotency_key=$6
		  and request_digest=$7 and phase='canonical-claimed'`,
		leaseID, receiptRef, now, request.TenantID, request.ProjectID, key, digest)
	if err != nil {
		return gateway.ClaimSaga{}, err
	}
	if command.RowsAffected() != 1 {
		return gateway.ClaimSaga{}, gateway.ErrIdempotencyConflict
	}
	saga, err = loadSaga(ctx, tx, request.TenantID, request.ProjectID, key, true)
	if err != nil {
		return gateway.ClaimSaga{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return gateway.ClaimSaga{}, err
	}
	return saga, nil
}

func (store *Store) beginScoped(ctx context.Context, tenantID, projectID string, isolation pgx.TxIsoLevel) (pgx.Tx, error) {
	if tenantID == "" || projectID == "" {
		return nil, ErrProjectNotProvisioned
	}
	tx, err := store.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: isolation})
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `select set_config('mars3.tenant_id',$1,true), set_config('mars3.project_id',$2,true)`, tenantID, projectID); err != nil {
		_ = tx.Rollback(ctx)
		return nil, err
	}
	if _, err := tx.Exec(ctx, `select pg_advisory_xact_lock_shared(hashtextextended($1, 732041))`, tenantID+"|"+projectID); err != nil {
		_ = tx.Rollback(ctx)
		return nil, err
	}
	return tx, nil
}

func (store *Store) verifyProject(ctx context.Context, tx pgx.Tx, tenantID, projectID string, lock bool) (string, error) {
	query := `select fence_generation, issuance_enabled, barrier_state from mars3_authority.projects where tenant_id=$1 and project_id=$2`
	if lock {
		query += ` for update`
	}
	var generation string
	var enabled bool
	var barrierState string
	if err := tx.QueryRow(ctx, query, tenantID, projectID).Scan(&generation, &enabled, &barrierState); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrProjectNotProvisioned
		}
		return "", err
	}
	expected, err := store.generation(ctx, tenantID, projectID)
	if err != nil || expected == "" || expected != generation {
		return "", ErrFenceGeneration
	}
	if !enabled {
		return "", ErrIssuanceDisabled
	}
	if barrierState != "open" {
		return "", ErrProjectBarrier
	}
	return generation, nil
}

// Enter holds the shared side of the project barrier across a cross-store
// gateway operation. Release is idempotent and rolls back the lock-only tx.
func (store *Store) Enter(ctx context.Context, tenantID, projectID string) (func(), error) {
	tx, err := store.beginScoped(ctx, tenantID, projectID, pgx.RepeatableRead)
	if err != nil {
		return nil, err
	}
	if _, err := store.verifyProject(ctx, tx, tenantID, projectID, false); err != nil {
		_ = tx.Rollback(context.Background())
		return nil, err
	}
	var once bool
	return func() {
		if !once {
			once = true
			_ = tx.Rollback(context.Background())
		}
	}, nil
}

// EnterWork takes a distributed transaction-scoped lock for one canonical
// Bead after entering the shared project barrier.
func (store *Store) EnterWork(ctx context.Context, tenantID, projectID, beadID string) (func(), error) {
	if beadID == "" || strings.Contains(beadID, "|") {
		return nil, ErrProjectNotProvisioned
	}
	tx, err := store.beginScoped(ctx, tenantID, projectID, pgx.RepeatableRead)
	if err != nil {
		return nil, err
	}
	if _, err := store.verifyProject(ctx, tx, tenantID, projectID, false); err != nil {
		_ = tx.Rollback(context.Background())
		return nil, err
	}
	if _, err := tx.Exec(ctx, `select pg_advisory_xact_lock(hashtextextended($1, 981733))`, tenantID+"|"+projectID+"|"+beadID); err != nil {
		_ = tx.Rollback(context.Background())
		return nil, err
	}
	var once bool
	return func() {
		if !once {
			once = true
			_ = tx.Rollback(context.Background())
		}
	}, nil
}

func loadSaga(ctx context.Context, tx pgx.Tx, tenantID, projectID, key string, lock bool) (gateway.ClaimSaga, error) {
	query := `
		select request_digest, phase, bead_id, attempt_id, idempotency_key, base_sha,
		       capability, exclusive_paths, labels, trace_ref, work_json, lease_id, receipt_ref
		from mars3_authority.claim_sagas
		where tenant_id=$1 and project_id=$2 and idempotency_key=$3`
	if lock {
		query += ` for update`
	}
	var requestDigest, phase string
	var intent gateway.ClaimIntent
	var capability string
	var labels []string
	var workJSON []byte
	var leaseID, receiptRef *string
	if err := tx.QueryRow(ctx, query, tenantID, projectID, key).Scan(
		&requestDigest, &phase, &intent.BeadID, &intent.AttemptID, &intent.IdempotencyKey,
		&intent.BaseSHA, &capability, &intent.ExclusivePaths, &labels, &intent.TraceRef,
		&workJSON, &leaseID, &receiptRef,
	); err != nil {
		return gateway.ClaimSaga{}, err
	}
	intent.RequestDigest = requestDigest
	intent.TenantID = tenantID
	intent.ProjectID = projectID
	intent.Capability = authorityv1.Capability(capability)
	intent.Labels = labelsFromStrings(labels)
	saga := gateway.ClaimSaga{RequestDigest: requestDigest, Phase: gateway.ClaimPhase(phase), Intent: intent}
	if len(workJSON) > 0 {
		if err := decodeStrictJSON(workJSON, &saga.Work); err != nil {
			return gateway.ClaimSaga{}, err
		}
	}
	if leaseID != nil {
		lease, err := loadLease(ctx, tx, tenantID, projectID, *leaseID, lock)
		if err != nil {
			return gateway.ClaimSaga{}, err
		}
		saga.Lease = lease
	}
	if receiptRef != nil {
		saga.ReceiptRef = *receiptRef
	}
	return saga, nil
}

func loadLease(ctx context.Context, tx pgx.Tx, tenantID, projectID, leaseID string, lock bool) (authorityv1.CapabilityLease, error) {
	query := `
		select lease_id, tenant_id, project_id, bead_id, attempt_id, idempotency_key,
		       fence_generation, lease_epoch, claim_version, base_sha, capability,
		       exclusive_paths, labels, issued_at, expires_at, state
		from mars3_authority.leases
		where tenant_id=$1 and project_id=$2 and lease_id=$3`
	if lock {
		query += ` for update`
	}
	var lease authorityv1.CapabilityLease
	var claimVersion []byte
	var capability, state string
	var labels []string
	err := tx.QueryRow(ctx, query, tenantID, projectID, leaseID).Scan(
		&lease.LeaseID, &lease.TenantID, &lease.ProjectID, &lease.BeadID,
		&lease.AttemptID, &lease.IdempotencyKey, &lease.FenceGeneration,
		&lease.LeaseEpoch, &claimVersion, &lease.BaseSHA, &capability,
		&lease.ExclusivePaths, &labels, &lease.IssuedAt, &lease.ExpiresAt, &state,
	)
	if err != nil {
		return authorityv1.CapabilityLease{}, err
	}
	if err := decodeStrictJSON(claimVersion, &lease.ClaimVersion); err != nil {
		return authorityv1.CapabilityLease{}, err
	}
	lease.Capability = authorityv1.Capability(capability)
	lease.Labels = labelsFromStrings(labels)
	lease.State = authorityv1.LeaseState(state)
	lease.Active = state == "active"
	return lease, nil
}

func sameLeaseRequest(saga gateway.ClaimSaga, request gateway.LeaseRequest) bool {
	intent := saga.Intent
	return saga.RequestDigest == request.RequestDigest && intent.TenantID == request.TenantID && intent.ProjectID == request.ProjectID && intent.BeadID == request.BeadID && intent.AttemptID == request.AttemptID && intent.IdempotencyKey == request.IdempotencyKey && intent.BaseSHA == request.BaseSHA && intent.Capability == request.Capability && equalStrings(intent.ExclusivePaths, request.ExclusivePaths) && equalLabels(intent.Labels, request.Labels) && saga.Work.TenantID == request.TenantID && saga.Work.ProjectID == request.ProjectID && saga.Work.BeadID == request.BeadID && saga.Work.ClaimAttemptID == request.AttemptID && saga.Work.Version == request.ClaimVersion
}

func sameStoredWork(left, right authorityv1.WorkItem) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftJSON, rightJSON)
}

func pathsOverlap(left, right []string) bool {
	for _, leftPath := range left {
		for _, rightPath := range right {
			if leftPath == rightPath || (strings.HasSuffix(leftPath, "/") && strings.HasPrefix(rightPath, leftPath)) || (strings.HasSuffix(rightPath, "/") && strings.HasPrefix(leftPath, rightPath)) {
				return true
			}
		}
	}
	return false
}

func labelStrings(labels []authorityv1.Label) []string {
	result := make([]string, len(labels))
	for index, label := range labels {
		result[index] = string(label)
	}
	return result
}

func labelsFromStrings(labels []string) []authorityv1.Label {
	result := make([]authorityv1.Label, len(labels))
	for index, label := range labels {
		result[index] = authorityv1.Label(label)
	}
	return result
}

func decodeStrictJSON(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("stored JSON has trailing content")
	}
	return nil
}

func randomID(prefix string) (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(buffer)), nil
}

func rollback(tx pgx.Tx, ctx context.Context) { _ = tx.Rollback(ctx) }
