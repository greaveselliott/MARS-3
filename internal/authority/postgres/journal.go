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
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	authorityv1 "github.com/greaveselliott/MARS-3/api/authority/v1"
)

var (
	ErrJournalConflict   = errors.New("authority journal event conflict")
	ErrJournalCheckpoint = errors.New("authority journal checkpoint unknown")
	ErrJournalTruncated  = errors.New("authority journal history truncated")
	journalID            = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
	journalDigest        = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

const maximumJournalPage = 1000

// Append implements gateway.EventSink with a project-local monotonic sequence,
// idempotent event ID, and tamper-evident hash chain.
func (store *Store) Append(ctx context.Context, event authorityv1.Event) (authorityv1.Event, error) {
	for attempt := 0; attempt < 2; attempt++ {
		receipt, err := store.appendOnce(ctx, event)
		if err == nil || !retryableJournalTransaction(err) {
			return receipt, err
		}
	}
	return authorityv1.Event{}, ErrJournalConflict
}

func (store *Store) appendOnce(ctx context.Context, event authorityv1.Event) (authorityv1.Event, error) {
	event.Labels = sortedJournalLabels(event.Labels)
	if !validJournalEvent(event) {
		return authorityv1.Event{}, ErrJournalConflict
	}
	tx, err := store.beginScoped(ctx, event.TenantID, event.ProjectID, pgx.Serializable)
	if err != nil {
		return authorityv1.Event{}, err
	}
	defer rollback(tx, ctx)
	if _, err := store.verifyProject(ctx, tx, event.TenantID, event.ProjectID, true); err != nil {
		return authorityv1.Event{}, err
	}
	if existing, err := loadEventByID(ctx, tx, event.TenantID, event.ProjectID, event.EventID); err == nil {
		if !sameEventIntent(existing, event) {
			return authorityv1.Event{}, ErrJournalConflict
		}
		if err := tx.Commit(ctx); err != nil {
			return authorityv1.Event{}, err
		}
		return existing, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return authorityv1.Event{}, err
	}

	var highWatermark uint64
	var previousHash string
	if err := tx.QueryRow(ctx, `
		select journal_high_watermark, journal_chain_hash
		from mars3_authority.projects
		where tenant_id=$1 and project_id=$2
		for update`, event.TenantID, event.ProjectID).Scan(&highWatermark, &previousHash); err != nil {
		return authorityv1.Event{}, err
	}
	event.Sequence = highWatermark + 1
	if event.Sequence == 0 {
		return authorityv1.Event{}, ErrJournalConflict
	}
	event.PreviousHash = previousHash
	event.OccurredAt = event.OccurredAt.UTC().Truncate(time.Microsecond)
	event.EventHash, err = authorityv1.JournalEventHash(event)
	if err != nil {
		return authorityv1.Event{}, err
	}
	canonicalVersion, err := optionalJSON(event.CanonicalVersion)
	if err != nil {
		return authorityv1.Event{}, err
	}
	_, err = tx.Exec(ctx, `
		insert into mars3_authority.authority_events
		    (tenant_id, project_id, sequence, event_id, schema_version,
		     previous_hash, event_hash, trace_ref, bead_id, attempt_id,
		     idempotency_key, principal_id, operation, outcome, rule, labels,
		     canonical_version, lease_epoch, before_hash, after_hash, occurred_at)
		values ($1,$2,$3,$4,$5,$6,$7,$8,nullif($9,''),nullif($10,''),
		        nullif($11,''),$12,$13,$14,nullif($15,''),$16,$17,
		        nullif($18,0),nullif($19,''),nullif($20,''),$21)`,
		event.TenantID, event.ProjectID, event.Sequence, event.EventID,
		event.SchemaVersion, event.PreviousHash, event.EventHash, event.TraceRef,
		event.BeadID, event.AttemptID, event.IdempotencyKey, event.PrincipalID,
		event.Operation, event.Outcome, event.Rule, labelStrings(event.Labels),
		canonicalVersion, event.LeaseEpoch, event.BeforeHash, event.AfterHash,
		event.OccurredAt.UTC())
	if err != nil {
		return authorityv1.Event{}, err
	}
	command, err := tx.Exec(ctx, `
		update mars3_authority.projects
		set journal_high_watermark=$1, journal_chain_hash=$2
		where tenant_id=$3 and project_id=$4 and journal_high_watermark=$5`,
		event.Sequence, event.EventHash, event.TenantID, event.ProjectID, highWatermark)
	if err != nil {
		return authorityv1.Event{}, err
	}
	if command.RowsAffected() != 1 {
		return authorityv1.Event{}, ErrJournalConflict
	}
	if err := tx.Commit(ctx); err != nil {
		return authorityv1.Event{}, err
	}
	return event, nil
}

func retryableJournalTransaction(err error) bool {
	var postgresError *pgconn.PgError
	return errors.As(err, &postgresError) && (postgresError.Code == "40001" || postgresError.Code == "40P01")
}

// Replay returns only events after an exact checkpoint. Unknown or retained-
// away cursors fail closed instead of silently returning a partial history.
func (store *Store) Replay(ctx context.Context, tenantID, projectID string, after authorityv1.JournalCursor, limit int) (authorityv1.JournalPage, error) {
	if limit < 1 || limit > maximumJournalPage {
		return authorityv1.JournalPage{}, ErrJournalCheckpoint
	}
	tx, err := store.beginScoped(ctx, tenantID, projectID, pgx.RepeatableRead)
	if err != nil {
		return authorityv1.JournalPage{}, err
	}
	defer rollback(tx, ctx)
	if _, err := store.verifyProject(ctx, tx, tenantID, projectID, false); err != nil {
		return authorityv1.JournalPage{}, err
	}
	var highWatermark, lowestRetained uint64
	var chainHash string
	if err := tx.QueryRow(ctx, `
		select journal_high_watermark, journal_chain_hash, journal_lowest_retained
		from mars3_authority.projects
		where tenant_id=$1 and project_id=$2`, tenantID, projectID).Scan(&highWatermark, &chainHash, &lowestRetained); err != nil {
		return authorityv1.JournalPage{}, err
	}
	if after.Sequence > highWatermark || (after.Sequence == 0 && after.EventHash != "") || (after.Sequence > 0 && !journalDigest.MatchString(after.EventHash)) {
		return authorityv1.JournalPage{}, ErrJournalCheckpoint
	}
	if after.Sequence+1 < lowestRetained {
		return authorityv1.JournalPage{}, ErrJournalTruncated
	}
	if after.Sequence > 0 {
		var storedHash string
		if err := tx.QueryRow(ctx, `
			select event_hash from mars3_authority.authority_events
			where tenant_id=$1 and project_id=$2 and sequence=$3`, tenantID, projectID, after.Sequence).Scan(&storedHash); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return authorityv1.JournalPage{}, ErrJournalTruncated
			}
			return authorityv1.JournalPage{}, err
		}
		if storedHash != after.EventHash {
			return authorityv1.JournalPage{}, ErrJournalCheckpoint
		}
	}
	rows, err := tx.Query(ctx, `
		select sequence, event_id, schema_version, previous_hash, event_hash,
		       trace_ref, bead_id, attempt_id, idempotency_key, principal_id,
		       operation, outcome, rule, labels, canonical_version, lease_epoch,
		       before_hash, after_hash, occurred_at
		from mars3_authority.authority_events
		where tenant_id=$1 and project_id=$2 and sequence > $3
		order by sequence
		limit $4`, tenantID, projectID, after.Sequence, limit)
	if err != nil {
		return authorityv1.JournalPage{}, err
	}
	defer rows.Close()
	events := make([]authorityv1.Event, 0, limit)
	for rows.Next() {
		event, err := scanEvent(rows, tenantID, projectID)
		if err != nil {
			return authorityv1.JournalPage{}, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return authorityv1.JournalPage{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return authorityv1.JournalPage{}, err
	}
	return authorityv1.JournalPage{
		TenantID: tenantID, ProjectID: projectID, After: after,
		LowestRetained: lowestRetained,
		HighWatermark:  authorityv1.JournalCursor{Sequence: highWatermark, EventHash: chainHash},
		Events:         events,
	}, nil
}

func loadEventByID(ctx context.Context, tx pgx.Tx, tenantID, projectID, eventID string) (authorityv1.Event, error) {
	row := tx.QueryRow(ctx, `
		select sequence, event_id, schema_version, previous_hash, event_hash,
		       trace_ref, bead_id, attempt_id, idempotency_key, principal_id,
		       operation, outcome, rule, labels, canonical_version, lease_epoch,
		       before_hash, after_hash, occurred_at
		from mars3_authority.authority_events
		where tenant_id=$1 and project_id=$2 and event_id=$3`, tenantID, projectID, eventID)
	return scanEvent(row, tenantID, projectID)
}

type eventScanner interface{ Scan(...any) error }

func scanEvent(row eventScanner, tenantID, projectID string) (authorityv1.Event, error) {
	var event authorityv1.Event
	var beadID, attemptID, idempotencyKey, rule, beforeHash, afterHash *string
	var labels []string
	var canonicalVersion []byte
	var leaseEpoch *uint64
	err := row.Scan(
		&event.Sequence, &event.EventID, &event.SchemaVersion, &event.PreviousHash,
		&event.EventHash, &event.TraceRef, &beadID, &attemptID, &idempotencyKey,
		&event.PrincipalID, &event.Operation, &event.Outcome, &rule, &labels,
		&canonicalVersion, &leaseEpoch, &beforeHash, &afterHash, &event.OccurredAt,
	)
	if err != nil {
		return authorityv1.Event{}, err
	}
	event.TenantID, event.ProjectID = tenantID, projectID
	if beadID != nil {
		event.BeadID = *beadID
	}
	if attemptID != nil {
		event.AttemptID = *attemptID
	}
	if idempotencyKey != nil {
		event.IdempotencyKey = *idempotencyKey
	}
	if rule != nil {
		event.Rule = *rule
	}
	event.Labels = labelsFromStrings(labels)
	if len(canonicalVersion) > 0 {
		event.CanonicalVersion = &authorityv1.WorkVersion{}
		if err := decodeStrictJSON(canonicalVersion, event.CanonicalVersion); err != nil {
			return authorityv1.Event{}, err
		}
	}
	if leaseEpoch != nil {
		event.LeaseEpoch = *leaseEpoch
	}
	if beforeHash != nil {
		event.BeforeHash = *beforeHash
	}
	if afterHash != nil {
		event.AfterHash = *afterHash
	}
	event.OccurredAt = event.OccurredAt.UTC()
	digest, err := authorityv1.JournalEventHash(event)
	if err != nil || digest != event.EventHash {
		return authorityv1.Event{}, ErrJournalConflict
	}
	return event, nil
}

func optionalJSON(value *authorityv1.WorkVersion) (any, error) {
	if value == nil {
		return nil, nil
	}
	data, err := json.Marshal(value)
	return string(data), err
}

func validJournalEvent(event authorityv1.Event) bool {
	if event.SchemaVersion != 1 || !journalID.MatchString(event.EventID) || event.Sequence != 0 || event.PreviousHash != "" || event.EventHash != "" || !journalID.MatchString(event.TraceRef) || !journalID.MatchString(event.TenantID) || !journalID.MatchString(event.ProjectID) || !optionalJournalID(event.BeadID) || !optionalJournalID(event.AttemptID) || !optionalJournalID(event.IdempotencyKey) || !journalID.MatchString(event.PrincipalID) || !journalID.MatchString(event.Operation) || !journalID.MatchString(event.Outcome) || !optionalJournalID(event.Rule) || len(event.Labels) == 0 || len(event.Labels) > 64 || event.OccurredAt.IsZero() || (event.BeforeHash != "" && !journalDigest.MatchString(event.BeforeHash)) || (event.AfterHash != "" && !journalDigest.MatchString(event.AfterHash)) {
		return false
	}
	for index, label := range event.Labels {
		if !knownJournalLabel(label) || (index > 0 && event.Labels[index-1] >= label) {
			return false
		}
	}
	if event.CanonicalVersion != nil && (!journalID.MatchString(event.CanonicalVersion.AuthorityGeneration) || !journalID.MatchString(event.CanonicalVersion.IssueIncarnation) || event.CanonicalVersion.IssueMutationSequence == 0 || event.CanonicalVersion.DependencyGraphRevision == 0) {
		return false
	}
	return true
}

func optionalJournalID(value string) bool { return value == "" || journalID.MatchString(value) }

func knownJournalLabel(label authorityv1.Label) bool {
	switch label {
	case authorityv1.LabelPublicAccepted, authorityv1.LabelExternalUntrusted, authorityv1.LabelPrivateData, authorityv1.LabelExternalEffect:
		return true
	default:
		return false
	}
}

func sortedJournalLabels(labels []authorityv1.Label) []authorityv1.Label {
	result := append([]authorityv1.Label(nil), labels...)
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func sameEventIntent(existing, proposed authorityv1.Event) bool {
	existing.Sequence, existing.PreviousHash, existing.EventHash, existing.OccurredAt = 0, "", "", time.Time{}
	proposed.Sequence, proposed.PreviousHash, proposed.EventHash, proposed.OccurredAt = 0, "", "", time.Time{}
	left, leftErr := json.Marshal(existing)
	right, rightErr := json.Marshal(proposed)
	return leftErr == nil && rightErr == nil && string(left) == string(right)
}
