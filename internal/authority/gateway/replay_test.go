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
	"testing"
	"time"

	authorityv1 "github.com/greaveselliott/MARS-3/api/authority/v1"
)

func TestProjectionConsumerAppliesContiguousEventsAndIgnoresExactDuplicate(t *testing.T) {
	consumer := mustProjectionConsumer(t)
	events := chainedEvents(t, 1, "", 2)
	page := journalPage(authorityv1.JournalCursor{}, events)
	applied := make([]string, 0, 2)
	apply := func(_ context.Context, event authorityv1.Event) error {
		applied = append(applied, event.EventID)
		return nil
	}
	if err := consumer.ApplyPage(context.Background(), page, apply); err != nil {
		t.Fatalf("ApplyPage: %v", err)
	}
	checkpoint := consumer.Checkpoint()
	if checkpoint.Cursor.Sequence != 2 || checkpoint.Cursor.EventHash != events[1].EventHash || !checkpoint.Authorizing || len(applied) != 2 {
		t.Fatalf("checkpoint=%#v applied=%#v", checkpoint, applied)
	}
	duplicatePage := journalPage(checkpoint.Cursor, []authorityv1.Event{events[1]})
	duplicatePage.HighWatermark = checkpoint.Cursor
	if err := consumer.ApplyPage(context.Background(), duplicatePage, apply); err != nil {
		t.Fatalf("exact duplicate: %v", err)
	}
	if len(applied) != 2 {
		t.Fatalf("duplicate was applied: %#v", applied)
	}
}

func TestProjectionConsumerFailsClosedOnGapConflictAndTruncation(t *testing.T) {
	for name, mutate := range map[string]func(*authorityv1.JournalPage){
		"gap": func(page *authorityv1.JournalPage) {
			page.Events[0].Sequence = 2
			page.Events[0].EventHash = mustEventHash(t, page.Events[0])
			page.HighWatermark = authorityv1.JournalCursor{Sequence: 2, EventHash: page.Events[0].EventHash}
		},
		"conflict": func(page *authorityv1.JournalPage) {
			page.Events[0].Outcome = "changed"
		},
		"truncation": func(page *authorityv1.JournalPage) {
			page.LowestRetained = 2
		},
	} {
		t.Run(name, func(t *testing.T) {
			consumer := mustProjectionConsumer(t)
			events := chainedEvents(t, 1, "", 1)
			page := journalPage(authorityv1.JournalCursor{}, events)
			mutate(&page)
			calls := 0
			err := consumer.ApplyPage(context.Background(), page, func(context.Context, authorityv1.Event) error { calls++; return nil })
			if !errors.Is(err, ErrProjectionStale) || consumer.Checkpoint().Authorizing || consumer.Checkpoint().StaleReason == "" || calls != 0 {
				t.Fatalf("err=%v checkpoint=%#v calls=%d", err, consumer.Checkpoint(), calls)
			}
		})
	}
}

func TestProjectionConsumerDiscardsViewOnCanonicalVersionConflict(t *testing.T) {
	consumer := mustProjectionConsumer(t)
	events := chainedEvents(t, 1, "", 1)
	err := consumer.ApplyPage(context.Background(), journalPage(authorityv1.JournalCursor{}, events), func(context.Context, authorityv1.Event) error {
		return errors.New("derived canonical version differs")
	})
	if !errors.Is(err, ErrProjectionStale) {
		t.Fatalf("ApplyPage error=%v", err)
	}
	checkpoint := consumer.Checkpoint()
	if checkpoint.Authorizing || checkpoint.StaleReason != staleApply || checkpoint.Cursor.Sequence != 0 {
		t.Fatalf("checkpoint=%#v", checkpoint)
	}
}

func TestProjectionConsumerRejectsHighWatermarkWithoutContiguousProgress(t *testing.T) {
	for name, mutate := range map[string]func(*authorityv1.JournalPage){
		"empty page ahead": func(page *authorityv1.JournalPage) {
			page.Events = nil
		},
		"terminal hash conflict": func(page *authorityv1.JournalPage) {
			page.HighWatermark.EventHash = mustEventHash(t, authorityv1.Event{
				SchemaVersion: 1, EventID: "evt-other", Sequence: 1, TraceRef: "trace-fixture",
				TenantID: "tenant-fixture", ProjectID: "project-fixture", PrincipalID: "principal-fixture",
				Operation: "work.status", Outcome: "allowed", Labels: []authorityv1.Label{authorityv1.LabelPublicAccepted},
				OccurredAt: time.Date(2026, 8, 27, 3, 0, 0, 0, time.UTC),
			})
		},
	} {
		t.Run(name, func(t *testing.T) {
			consumer := mustProjectionConsumer(t)
			events := chainedEvents(t, 1, "", 1)
			page := journalPage(authorityv1.JournalCursor{}, events)
			mutate(&page)
			if err := consumer.ApplyPage(context.Background(), page, func(context.Context, authorityv1.Event) error { return nil }); !errors.Is(err, ErrProjectionStale) || consumer.Checkpoint().Authorizing {
				t.Fatalf("err=%v checkpoint=%#v", err, consumer.Checkpoint())
			}
		})
	}
}

func TestProjectionConsumerAllowsContiguousPaginationBelowHighWatermark(t *testing.T) {
	consumer := mustProjectionConsumer(t)
	events := chainedEvents(t, 1, "", 2)
	page := journalPage(authorityv1.JournalCursor{}, events[:1])
	page.HighWatermark = authorityv1.JournalCursor{Sequence: events[1].Sequence, EventHash: events[1].EventHash}
	if err := consumer.ApplyPage(context.Background(), page, func(context.Context, authorityv1.Event) error { return nil }); err != nil {
		t.Fatalf("paginated page: %v", err)
	}
	if checkpoint := consumer.Checkpoint(); !checkpoint.Authorizing || checkpoint.Cursor.Sequence != 1 {
		t.Fatalf("checkpoint=%#v", checkpoint)
	}
}

func mustProjectionConsumer(t *testing.T) *ProjectionConsumer {
	t.Helper()
	consumer, err := NewProjectionConsumer(authorityv1.ProjectionCheckpoint{
		TenantID: "tenant-fixture", ProjectID: "project-fixture", Authorizing: true,
	})
	if err != nil {
		t.Fatalf("NewProjectionConsumer: %v", err)
	}
	return consumer
}

func chainedEvents(t *testing.T, first uint64, previousHash string, count int) []authorityv1.Event {
	t.Helper()
	events := make([]authorityv1.Event, count)
	for index := range events {
		event := authorityv1.Event{
			SchemaVersion: 1, EventID: "evt-fixture-" + string(rune('a'+index)), Sequence: first + uint64(index), PreviousHash: previousHash,
			TraceRef: "trace-fixture", TenantID: "tenant-fixture", ProjectID: "project-fixture", BeadID: "M3-W002",
			PrincipalID: "principal-fixture", Operation: "work.status", Outcome: "allowed", Labels: []authorityv1.Label{authorityv1.LabelPublicAccepted},
			OccurredAt: time.Date(2026, 8, 27, 2, index, 0, 0, time.UTC),
		}
		event.EventHash = mustEventHash(t, event)
		events[index] = event
		previousHash = event.EventHash
	}
	return events
}

func mustEventHash(t *testing.T, event authorityv1.Event) string {
	t.Helper()
	digest, err := authorityv1.JournalEventHash(event)
	if err != nil {
		t.Fatalf("JournalEventHash: %v", err)
	}
	return digest
}

func journalPage(after authorityv1.JournalCursor, events []authorityv1.Event) authorityv1.JournalPage {
	high := after
	if len(events) > 0 {
		high = authorityv1.JournalCursor{Sequence: events[len(events)-1].Sequence, EventHash: events[len(events)-1].EventHash}
	}
	return authorityv1.JournalPage{TenantID: "tenant-fixture", ProjectID: "project-fixture", After: after, LowestRetained: 1, HighWatermark: high, Events: events}
}
