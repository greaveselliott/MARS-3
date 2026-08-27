/*
FactoryDocSync:
docs:
- docs/features/F-002-work-authority.md
- docs/design-docs/ADR-001-git-beads-authority.md
- docs/code-documentation-map.md
*/

package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"

	authorityv1 "github.com/greaveselliott/MARS-3/api/authority/v1"
	"github.com/greaveselliott/MARS-3/internal/authority/gateway"
)

var ErrBaselineInconsistent = errors.New("authority baseline is inconsistent")

type pendingBaselineSaga struct {
	BeadID    string
	AttemptID string
	Phase     gateway.ClaimPhase
	Work      authorityv1.WorkItem
}

// CaptureBaseline takes the exclusive project barrier, waits for all shared
// gateway operations to drain, and verifies one stable cross-store cut. It
// reads but never mutates canonical Beads state.
func (store *Store) CaptureBaseline(ctx context.Context, tenantID, projectID string, canonical gateway.WorkStore) (authorityv1.AuthorityBaseline, error) {
	if canonical == nil {
		return authorityv1.AuthorityBaseline{}, ErrBaselineInconsistent
	}
	tx, err := store.beginExclusiveScoped(ctx, tenantID, projectID)
	if err != nil {
		return authorityv1.AuthorityBaseline{}, err
	}
	defer rollback(tx, ctx)
	generation, err := store.verifyProject(ctx, tx, tenantID, projectID, true)
	if err != nil {
		return authorityv1.AuthorityBaseline{}, err
	}
	if _, err := tx.Exec(ctx, `
		update mars3_authority.projects set barrier_state='rebaselining'
		where tenant_id=$1 and project_id=$2 and barrier_state='open'`, tenantID, projectID); err != nil {
		return authorityv1.AuthorityBaseline{}, err
	}

	first, err := canonical.List(ctx, tenantID, projectID)
	if err != nil {
		return authorityv1.AuthorityBaseline{}, ErrBaselineInconsistent
	}
	first = sortedBaselineWork(first)
	leases, err := loadActiveLeases(ctx, tx, tenantID, projectID, store.now().UTC())
	if err != nil {
		return authorityv1.AuthorityBaseline{}, err
	}
	pending, err := loadPendingSagas(ctx, tx, tenantID, projectID)
	if err != nil {
		return authorityv1.AuthorityBaseline{}, err
	}
	var watermark authorityv1.JournalCursor
	if err := tx.QueryRow(ctx, `
		select journal_high_watermark, journal_chain_hash
		from mars3_authority.projects where tenant_id=$1 and project_id=$2`, tenantID, projectID).Scan(&watermark.Sequence, &watermark.EventHash); err != nil {
		return authorityv1.AuthorityBaseline{}, err
	}
	second, err := canonical.List(ctx, tenantID, projectID)
	if err != nil {
		return authorityv1.AuthorityBaseline{}, ErrBaselineInconsistent
	}
	second = sortedBaselineWork(second)
	if !sameBaselineWork(first, second) || !consistentBaseline(first, leases, pending) {
		return authorityv1.AuthorityBaseline{}, ErrBaselineInconsistent
	}
	authorityGeneration, ok := singleAuthorityGeneration(first)
	if !ok {
		return authorityv1.AuthorityBaseline{}, ErrBaselineInconsistent
	}
	baseline := authorityv1.AuthorityBaseline{
		TenantID: tenantID, ProjectID: projectID, AuthorityGeneration: authorityGeneration,
		Watermark: watermark, WorkItems: first, LiveLeases: leases,
		PendingSagaCount: uint32(len(pending)), CapturedAt: store.now().UTC(), Verified: true,
	}
	baseline.BaselineDigest, err = baselineDigest(baseline)
	if err != nil {
		return authorityv1.AuthorityBaseline{}, err
	}
	command, err := tx.Exec(ctx, `
		update mars3_authority.projects
		set barrier_state='open', barrier_epoch=barrier_epoch+1
		where tenant_id=$1 and project_id=$2 and barrier_state='rebaselining'`, tenantID, projectID)
	if err != nil {
		return authorityv1.AuthorityBaseline{}, err
	}
	if command.RowsAffected() != 1 {
		return authorityv1.AuthorityBaseline{}, ErrProjectBarrier
	}
	if err := tx.Commit(ctx); err != nil {
		return authorityv1.AuthorityBaseline{}, err
	}
	_ = generation // verified against the externally anchored fence generation.
	return baseline, nil
}

func (store *Store) beginExclusiveScoped(ctx context.Context, tenantID, projectID string) (pgx.Tx, error) {
	if tenantID == "" || projectID == "" {
		return nil, ErrProjectNotProvisioned
	}
	tx, err := store.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `select set_config('mars3.tenant_id',$1,true), set_config('mars3.project_id',$2,true)`, tenantID, projectID); err != nil {
		_ = tx.Rollback(ctx)
		return nil, err
	}
	if _, err := tx.Exec(ctx, `select pg_advisory_xact_lock(hashtextextended($1, 732041))`, tenantID+"|"+projectID); err != nil {
		_ = tx.Rollback(ctx)
		return nil, err
	}
	return tx, nil
}

func loadActiveLeases(ctx context.Context, tx pgx.Tx, tenantID, projectID string, now time.Time) ([]authorityv1.CapabilityLease, error) {
	rows, err := tx.Query(ctx, `
		select lease_id from mars3_authority.leases
		where tenant_id=$1 and project_id=$2 and state='active' and expires_at > $3
		order by lease_id`, tenantID, projectID, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	leases := make([]authorityv1.CapabilityLease, 0, len(ids))
	for _, id := range ids {
		lease, err := loadLease(ctx, tx, tenantID, projectID, id, false)
		if err != nil {
			return nil, err
		}
		leases = append(leases, lease)
	}
	return leases, nil
}

func loadPendingSagas(ctx context.Context, tx pgx.Tx, tenantID, projectID string) ([]pendingBaselineSaga, error) {
	rows, err := tx.Query(ctx, `
		select bead_id, attempt_id, phase, work_json
		from mars3_authority.claim_sagas
		where tenant_id=$1 and project_id=$2 and phase <> 'complete'
		order by bead_id, idempotency_key`, tenantID, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []pendingBaselineSaga
	for rows.Next() {
		var saga pendingBaselineSaga
		var workJSON []byte
		if err := rows.Scan(&saga.BeadID, &saga.AttemptID, &saga.Phase, &workJSON); err != nil {
			return nil, err
		}
		if len(workJSON) > 0 {
			if err := decodeStrictJSON(workJSON, &saga.Work); err != nil {
				return nil, err
			}
		}
		result = append(result, saga)
	}
	return result, rows.Err()
}

func consistentBaseline(work []authorityv1.WorkItem, leases []authorityv1.CapabilityLease, pending []pendingBaselineSaga) bool {
	workByBead := make(map[string]authorityv1.WorkItem, len(work))
	for _, item := range work {
		if item.BeadID == "" || item.Version.AuthorityGeneration == "" || item.Version.IssueIncarnation == "" || item.Version.IssueMutationSequence == 0 || item.Version.DependencyGraphRevision == 0 {
			return false
		}
		if _, duplicate := workByBead[item.BeadID]; duplicate {
			return false
		}
		workByBead[item.BeadID] = item
	}
	covered := make(map[string]bool)
	for _, lease := range leases {
		item, found := workByBead[lease.BeadID]
		if !found || covered[lease.BeadID] || !lease.Active || item.NativeStatus != "in_progress" || item.LifecycleState != authorityv1.LifecycleInProgress || item.ClaimAttemptID != lease.AttemptID || item.Version != lease.ClaimVersion || !equalStrings(item.ExclusivePaths, lease.ExclusivePaths) || !equalLabels(item.Labels, lease.Labels) {
			return false
		}
		covered[lease.BeadID] = true
	}
	for _, saga := range pending {
		item, found := workByBead[saga.BeadID]
		if !found || covered[saga.BeadID] {
			return false
		}
		switch saga.Phase {
		case gateway.ClaimPhaseIntent:
			if item.LifecycleState != authorityv1.LifecycleBacklog {
				return false
			}
		case gateway.ClaimPhaseCanonical:
			if item.LifecycleState != authorityv1.LifecycleInProgress || item.ClaimAttemptID != saga.AttemptID || !sameStoredWork(item, saga.Work) {
				return false
			}
		default:
			return false
		}
		covered[saga.BeadID] = true
	}
	for _, item := range work {
		if item.LifecycleState == authorityv1.LifecycleInProgress && !covered[item.BeadID] {
			return false
		}
	}
	return true
}

func sortedBaselineWork(items []authorityv1.WorkItem) []authorityv1.WorkItem {
	result := append([]authorityv1.WorkItem(nil), items...)
	sort.Slice(result, func(i, j int) bool { return result[i].BeadID < result[j].BeadID })
	return result
}

func sameBaselineWork(left, right []authorityv1.WorkItem) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftJSON, rightJSON)
}

func singleAuthorityGeneration(items []authorityv1.WorkItem) (string, bool) {
	if len(items) == 0 {
		return "", false
	}
	generation := items[0].Version.AuthorityGeneration
	for _, item := range items[1:] {
		if item.Version.AuthorityGeneration != generation {
			return "", false
		}
	}
	return generation, generation != ""
}

func baselineDigest(baseline authorityv1.AuthorityBaseline) (string, error) {
	material := baseline
	material.BaselineDigest = ""
	data, err := json.Marshal(material)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}
