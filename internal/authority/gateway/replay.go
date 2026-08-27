/*
FactoryDocSync:
docs:
- docs/features/F-002-work-authority.md
- docs/design-docs/ADR-001-git-beads-authority.md
- docs/code-documentation-map.md
*/

package gateway

import (
	"context"
	"errors"
	"sync"

	authorityv1 "github.com/greaveselliott/MARS-3/api/authority/v1"
)

var ErrProjectionStale = errors.New("authority projection is stale and non-authorizing")

const (
	staleCheckpoint = "projection.checkpoint.unknown"
	staleTruncated  = "projection.journal.truncated"
	staleGap        = "projection.journal.gap"
	staleConflict   = "projection.journal.conflict"
	staleApply      = "projection.canonical_version.conflict"
)

// ProjectionApply updates one disposable derived view. An error means its
// whole view must be discarded; it cannot authorize a repair or mutation.
type ProjectionApply func(context.Context, authorityv1.Event) error

// ProjectionConsumer applies one trusted journal sequence exactly once. It is
// deliberately stateful so duplicate receipts can be checked, not guessed.
type ProjectionConsumer struct {
	mu         sync.Mutex
	checkpoint authorityv1.ProjectionCheckpoint
	applied    map[uint64]string
}

func NewProjectionConsumer(checkpoint authorityv1.ProjectionCheckpoint) (*ProjectionConsumer, error) {
	if checkpoint.TenantID == "" || checkpoint.ProjectID == "" || !checkpoint.Authorizing || checkpoint.StaleReason != "" || checkpoint.Cursor.Sequence < checkpoint.BaselineWatermark.Sequence || (checkpoint.Cursor.Sequence > 0 && checkpoint.Cursor.EventHash == "") || (checkpoint.BaselineWatermark.Sequence > 0 && checkpoint.BaselineWatermark.EventHash == "") {
		return nil, ErrProjectionStale
	}
	applied := make(map[uint64]string)
	if checkpoint.Cursor.Sequence > 0 {
		applied[checkpoint.Cursor.Sequence] = checkpoint.Cursor.EventHash
	}
	return &ProjectionConsumer{checkpoint: checkpoint, applied: applied}, nil
}

func (consumer *ProjectionConsumer) Checkpoint() authorityv1.ProjectionCheckpoint {
	consumer.mu.Lock()
	defer consumer.mu.Unlock()
	return consumer.checkpoint
}

// ApplyPage accepts only a page rooted at the exact persisted cursor. A gap,
// altered duplicate, hash-chain mismatch, or derived-version conflict makes
// the consumer permanently non-authorizing until a full verified rebaseline.
func (consumer *ProjectionConsumer) ApplyPage(ctx context.Context, page authorityv1.JournalPage, apply ProjectionApply) error {
	consumer.mu.Lock()
	defer consumer.mu.Unlock()
	if !consumer.checkpoint.Authorizing || consumer.checkpoint.StaleReason != "" {
		return ErrProjectionStale
	}
	if apply == nil || page.TenantID != consumer.checkpoint.TenantID || page.ProjectID != consumer.checkpoint.ProjectID || page.After != consumer.checkpoint.Cursor || page.HighWatermark.Sequence < consumer.checkpoint.Cursor.Sequence {
		return consumer.stale(staleCheckpoint)
	}
	if page.HighWatermark.Sequence == 0 && page.HighWatermark.EventHash != "" || page.HighWatermark.Sequence > 0 && page.HighWatermark.EventHash == "" ||
		page.HighWatermark.Sequence == consumer.checkpoint.Cursor.Sequence && page.HighWatermark.EventHash != consumer.checkpoint.Cursor.EventHash {
		return consumer.stale(staleConflict)
	}
	if page.LowestRetained > consumer.checkpoint.Cursor.Sequence+1 {
		return consumer.stale(staleTruncated)
	}
	startSequence := consumer.checkpoint.Cursor.Sequence
	for _, event := range page.Events {
		if event.TenantID != page.TenantID || event.ProjectID != page.ProjectID || event.SchemaVersion != 1 || event.Sequence > page.HighWatermark.Sequence {
			return consumer.stale(staleConflict)
		}
		digest, err := authorityv1.JournalEventHash(event)
		if err != nil || digest != event.EventHash {
			return consumer.stale(staleConflict)
		}
		if event.Sequence <= consumer.checkpoint.Cursor.Sequence {
			if prior, known := consumer.applied[event.Sequence]; !known || prior != event.EventHash {
				return consumer.stale(staleConflict)
			}
			continue
		}
		if event.Sequence != consumer.checkpoint.Cursor.Sequence+1 || event.PreviousHash != consumer.checkpoint.Cursor.EventHash {
			return consumer.stale(staleGap)
		}
		if err := apply(ctx, event); err != nil {
			return consumer.stale(staleApply)
		}
		consumer.checkpoint.Cursor = authorityv1.JournalCursor{Sequence: event.Sequence, EventHash: event.EventHash}
		consumer.applied[event.Sequence] = event.EventHash
	}
	if page.HighWatermark.Sequence > startSequence && consumer.checkpoint.Cursor.Sequence == startSequence {
		return consumer.stale(staleGap)
	}
	if page.HighWatermark.Sequence == consumer.checkpoint.Cursor.Sequence && page.HighWatermark.EventHash != consumer.checkpoint.Cursor.EventHash {
		return consumer.stale(staleConflict)
	}
	return nil
}

func (consumer *ProjectionConsumer) stale(reason string) error {
	consumer.checkpoint.Authorizing = false
	consumer.checkpoint.StaleReason = reason
	return ErrProjectionStale
}
