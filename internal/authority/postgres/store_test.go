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
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	authorityv1 "github.com/greaveselliott/MARS-3/api/authority/v1"
	"github.com/greaveselliott/MARS-3/internal/authority/gateway"
)

func TestPathsOverlapUsesDirectoryBoundaries(t *testing.T) {
	for name, test := range map[string]struct {
		left, right []string
		want        bool
	}{
		"same file":          {[]string{"go.mod"}, []string{"go.mod"}, true},
		"directory child":    {[]string{"internal/authority/"}, []string{"internal/authority/gateway/a.go"}, true},
		"reverse child":      {[]string{"database/authority/schema.sql"}, []string{"database/authority/"}, true},
		"shared prefix only": {[]string{"internal/authority/"}, []string{"internal/authorityx/file.go"}, false},
		"different":          {[]string{"api/authority/"}, []string{"docs/evidence/"}, false},
	} {
		t.Run(name, func(t *testing.T) {
			if got := pathsOverlap(test.left, test.right); got != test.want {
				t.Fatalf("pathsOverlap(%v,%v)=%v, want %v", test.left, test.right, got, test.want)
			}
		})
	}
}

func TestDecodeStrictJSONRejectsUnknownAndTrailingContent(t *testing.T) {
	var version authorityv1.WorkVersion
	if err := decodeStrictJSON([]byte(`{"authority_generation":"g","issue_incarnation":"i","issue_mutation_sequence":1,"dependency_graph_revision":1}`), &version); err != nil {
		t.Fatalf("valid JSON: %v", err)
	}
	if err := decodeStrictJSON([]byte(`{"authority_generation":"g","issue_incarnation":"i","issue_mutation_sequence":1,"dependency_graph_revision":1,"unknown":true}`), &version); err == nil {
		t.Fatal("unknown stored field was accepted")
	}
	if err := decodeStrictJSON([]byte(`{} {}`), &version); err == nil {
		t.Fatal("trailing stored JSON was accepted")
	}
}

func TestConsistentBaselineSeparatesCanonicalClaimAndLeaseAttempts(t *testing.T) {
	work := postgresClaimedWork("tenant-synthetic", "project-synthetic", "M3-W001", "canonical-claim-attempt", strings.Repeat("a", 64))
	lease := authorityv1.CapabilityLease{
		LeaseID: "lease-001", TenantID: work.TenantID, ProjectID: work.ProjectID, BeadID: work.BeadID,
		AttemptID: "delivery-attempt", CanonicalClaimAttemptID: work.ClaimAttemptID, IdempotencyKey: "test-key",
		FenceGeneration: "fence-generation-001", LeaseEpoch: 1, ClaimVersion: work.Version,
		BaseSHA: strings.Repeat("b", 40), Capability: authorityv1.CapabilityTicketDelivery,
		ExclusivePaths: append([]string(nil), work.ExclusivePaths...), Labels: append([]authorityv1.Label(nil), work.Labels...),
		State: authorityv1.LeaseActive, Active: true,
	}
	if !consistentBaseline([]authorityv1.WorkItem{work}, []authorityv1.CapabilityLease{lease}, nil) {
		t.Fatal("valid baseline rejected because delivery and canonical claim attempts differ")
	}
	lease.CanonicalClaimAttemptID = "other-canonical-claim-attempt"
	if consistentBaseline([]authorityv1.WorkItem{work}, []authorityv1.CapabilityLease{lease}, nil) {
		t.Fatal("baseline accepted a lease bound to a different canonical claim attempt")
	}
}

func TestConsistentBaselineSeparatesCanonicalClaimAndPendingSagaAttempts(t *testing.T) {
	work := postgresClaimedWork("tenant-synthetic", "project-synthetic", "M3-W001", "canonical-claim-attempt", strings.Repeat("a", 64))
	saga := pendingBaselineSaga{
		BeadID: work.BeadID, AttemptID: "delivery-attempt", Phase: gateway.ClaimPhaseCanonical, Work: work,
	}
	if !consistentBaseline([]authorityv1.WorkItem{work}, nil, []pendingBaselineSaga{saga}) {
		t.Fatal("valid pending reconciliation rejected because delivery and canonical claim attempts differ")
	}
	saga.Work.ClaimAttemptID = "other-canonical-claim-attempt"
	if consistentBaseline([]authorityv1.WorkItem{work}, nil, []pendingBaselineSaga{saga}) {
		t.Fatal("pending reconciliation accepted a different canonical claim attempt")
	}
}

func TestPostgresLeaseLifecycleAndRestart(t *testing.T) {
	adminURL := os.Getenv("MARS3_TEST_POSTGRES_ADMIN_URL")
	appURL := os.Getenv("MARS3_TEST_POSTGRES_APP_URL")
	if adminURL == "" || appURL == "" {
		t.Skip("set synthetic PostgreSQL admin and app URLs to run integration conformance")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	admin, err := pgxpool.New(ctx, adminURL)
	if err != nil {
		t.Fatal("connect synthetic admin database")
	}
	defer admin.Close()
	migration, err := os.ReadFile(filepath.Join("..", "..", "..", "database", "authority", "001_work_authority.sql"))
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	if _, err := admin.Exec(ctx, string(migration)); err != nil {
		t.Fatalf("apply migration: %v", err)
	}
	if _, err := admin.Exec(ctx, `
		grant usage on schema mars3_authority to mars3_test_app;
		grant select, insert, update, delete on all tables in schema mars3_authority to mars3_test_app;`); err != nil {
		t.Fatalf("grant synthetic app role: %v", err)
	}
	app, err := pgxpool.New(ctx, appURL)
	if err != nil {
		t.Fatal("connect synthetic app database")
	}
	defer app.Close()
	if err := app.Ping(ctx); err != nil {
		t.Fatal("ping synthetic app database")
	}

	tenantID := "tenant-pg-" + time.Now().UTC().Format("20060102t150405000000000")
	projectID := "project-fixture"
	generation := "fence-generation-001"
	clock := time.Date(2026, 8, 27, 2, 0, 0, 0, time.UTC)
	var idMu sync.Mutex
	idCounter := 0
	newID := func(prefix string) (string, error) {
		idMu.Lock()
		defer idMu.Unlock()
		idCounter++
		return fmt.Sprintf("%s-fixture-%03d", prefix, idCounter), nil
	}
	resolver := func(_ context.Context, gotTenant, gotProject string) (string, error) {
		if gotTenant != tenantID || gotProject != projectID {
			return "", ErrProjectNotProvisioned
		}
		return generation, nil
	}
	store, err := New(app, resolver, func() time.Time { return clock }, newID)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := store.ProvisionProject(ctx, tenantID, projectID); err != nil {
		t.Fatalf("ProvisionProject: %v", err)
	}
	assertTenantRLS(t, ctx, app, tenantID, projectID, generation, clock)
	releaseFirstWork, err := store.EnterWork(ctx, tenantID, projectID, "M3-W002")
	if err != nil {
		t.Fatalf("EnterWork first: %v", err)
	}
	secondWorkLock := make(chan func(), 1)
	secondWorkError := make(chan error, 1)
	go func() {
		release, err := store.EnterWork(ctx, tenantID, projectID, "M3-W002")
		secondWorkLock <- release
		secondWorkError <- err
	}()
	select {
	case err := <-secondWorkError:
		t.Fatalf("same-Bead mutation lock did not serialize: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	releaseFirstWork()
	releaseSecondWork := <-secondWorkLock
	if err := <-secondWorkError; err != nil {
		t.Fatalf("EnterWork second: %v", err)
	}
	releaseSecondWork()
	journalEvent := authorityv1.Event{
		SchemaVersion: 1, EventID: "evt-pg-001", TraceRef: "trace-journal-001",
		TenantID: tenantID, ProjectID: projectID, BeadID: "M3-W002",
		PrincipalID: "work-authority-engineer", Operation: "work.claim.intent",
		Outcome: "intent-recorded", Labels: []authorityv1.Label{authorityv1.LabelPublicAccepted}, OccurredAt: clock,
	}
	firstEvent, err := store.Append(ctx, journalEvent)
	if err != nil || firstEvent.Sequence != 1 || firstEvent.PreviousHash != "" || firstEvent.EventHash == "" {
		t.Fatalf("first journal event=%#v err=%v", firstEvent, err)
	}
	replayedEvent, err := store.Append(ctx, journalEvent)
	if err != nil || !reflect.DeepEqual(replayedEvent, firstEvent) {
		t.Fatalf("idempotent event=%#v err=%v", replayedEvent, err)
	}
	conflictEvent := journalEvent
	conflictEvent.Outcome = "denied"
	if _, err := store.Append(ctx, conflictEvent); !errors.Is(err, ErrJournalConflict) {
		t.Fatalf("event conflict error=%v, want ErrJournalConflict", err)
	}
	journalEvent.EventID = "evt-pg-002"
	journalEvent.Operation = "work.claim.receipt"
	journalEvent.Outcome = "verified"
	journalEvent.OccurredAt = clock.Add(time.Second)
	secondEvent, err := store.Append(ctx, journalEvent)
	if err != nil || secondEvent.Sequence != 2 || secondEvent.PreviousHash != firstEvent.EventHash {
		t.Fatalf("second journal event=%#v err=%v", secondEvent, err)
	}
	page, err := store.Replay(ctx, tenantID, projectID, authorityv1.JournalCursor{}, 100)
	if err != nil || len(page.Events) != 2 || page.HighWatermark.Sequence != 2 || page.HighWatermark.EventHash != secondEvent.EventHash {
		t.Fatalf("journal page=%#v err=%v", page, err)
	}
	page, err = store.Replay(ctx, tenantID, projectID, authorityv1.JournalCursor{Sequence: 1, EventHash: firstEvent.EventHash}, 100)
	if err != nil || len(page.Events) != 1 || page.Events[0].EventID != secondEvent.EventID {
		t.Fatalf("cursor page=%#v err=%v", page, err)
	}
	if _, err := store.Replay(ctx, tenantID, projectID, authorityv1.JournalCursor{Sequence: 1, EventHash: strings.Repeat("f", 64)}, 100); !errors.Is(err, ErrJournalCheckpoint) {
		t.Fatalf("wrong checkpoint error=%v, want ErrJournalCheckpoint", err)
	}

	first := prepareCanonicalSaga(t, ctx, store, tenantID, projectID, "M3-W002", "attempt-001", "idempotency-001", "a")
	first.MaximumExpiry = clock.Add(10 * time.Minute)
	firstSaga, err := store.IssueLease(ctx, "idempotency-001", first.RequestDigest, first)
	if err != nil {
		t.Fatalf("IssueLease: %v", err)
	}
	if firstSaga.Lease.LeaseEpoch != 1 || firstSaga.Lease.FenceGeneration != generation || firstSaga.Lease.State != authorityv1.LeaseActive {
		t.Fatalf("first lease = %#v", firstSaga.Lease)
	}

	canonical := &baselineStore{items: []authorityv1.WorkItem{firstSaga.Work}, firstRead: make(chan struct{}), releaseFirst: make(chan struct{})}
	baselineResult := make(chan authorityv1.AuthorityBaseline, 1)
	baselineErrors := make(chan error, 1)
	go func() {
		baseline, err := store.CaptureBaseline(ctx, tenantID, projectID, canonical)
		baselineResult <- baseline
		baselineErrors <- err
	}()
	select {
	case <-canonical.firstRead:
	case <-time.After(5 * time.Second):
		t.Fatal("baseline did not enter canonical read")
	}
	journalEvent.EventID = "evt-pg-003"
	journalEvent.Operation = "work.baseline.concurrent"
	journalEvent.OccurredAt = clock.Add(2 * time.Second)
	appendDone := make(chan error, 1)
	go func() {
		_, err := store.Append(ctx, journalEvent)
		appendDone <- err
	}()
	select {
	case err := <-appendDone:
		t.Fatalf("journal append crossed the exclusive baseline barrier: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	close(canonical.releaseFirst)
	baseline := <-baselineResult
	if err := <-baselineErrors; err != nil {
		t.Fatalf("CaptureBaseline: %v", err)
	}
	if err := <-appendDone; err != nil {
		t.Fatalf("append after baseline: %v", err)
	}
	if !baseline.Verified || baseline.Watermark.Sequence != 2 || baseline.AuthorityGeneration != firstSaga.Work.Version.AuthorityGeneration || len(baseline.WorkItems) != 1 || len(baseline.LiveLeases) != 1 || baseline.BaselineDigest == "" {
		t.Fatalf("baseline = %#v", baseline)
	}
	driftedWork := firstSaga.Work
	driftedWork.Version.IssueMutationSequence++
	if _, err := store.CaptureBaseline(ctx, tenantID, projectID, &baselineStore{items: []authorityv1.WorkItem{firstSaga.Work}, second: []authorityv1.WorkItem{driftedWork}}); !errors.Is(err, ErrBaselineInconsistent) {
		t.Fatalf("drifted baseline error=%v, want ErrBaselineInconsistent", err)
	}

	restarted, err := New(app, resolver, func() time.Time { return clock }, newID)
	if err != nil {
		t.Fatalf("restart New: %v", err)
	}
	persisted, err := restarted.GetLease(ctx, tenantID, projectID, firstSaga.Lease.LeaseID)
	if err != nil || persisted.LeaseEpoch != firstSaga.Lease.LeaseEpoch || persisted.FenceGeneration != generation {
		t.Fatalf("restart lease=%#v err=%v", persisted, err)
	}

	wrongGeneration, err := New(app, func(context.Context, string, string) (string, error) { return "restored-old-generation", nil }, func() time.Time { return clock }, newID)
	if err != nil {
		t.Fatalf("wrong generation New: %v", err)
	}
	if _, err := wrongGeneration.GetLease(ctx, tenantID, projectID, firstSaga.Lease.LeaseID); !errors.Is(err, ErrFenceGeneration) {
		t.Fatalf("restored generation error=%v, want ErrFenceGeneration", err)
	}

	fence := fenceFromLease(firstSaga.Lease)
	clock = clock.Add(time.Minute)
	renewed, err := store.Renew(ctx, fence, clock.Add(14*time.Minute))
	if err != nil {
		t.Fatalf("Renew: %v", err)
	}
	if renewed.LeaseEpoch != firstSaga.Lease.LeaseEpoch || renewed.FenceGeneration != generation || !renewed.ExpiresAt.Equal(clock.Add(14*time.Minute)) {
		t.Fatalf("renewed lease changed fence: %#v", renewed)
	}
	guarded, releaseEffect, err := store.EnterEffect(ctx, fenceFromLease(renewed))
	if err != nil || guarded.LeaseID != renewed.LeaseID {
		t.Fatalf("EnterEffect lease=%#v err=%v", guarded, err)
	}
	guardRenewed := make(chan authorityv1.CapabilityLease, 1)
	guardErrors := make(chan error, 1)
	go func() {
		lease, err := store.Renew(ctx, fenceFromLease(renewed), clock.Add(15*time.Minute))
		guardRenewed <- lease
		guardErrors <- err
	}()
	select {
	case err := <-guardErrors:
		t.Fatalf("renew crossed the held effect fence: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	releaseEffect()
	renewed = <-guardRenewed
	if err := <-guardErrors; err != nil || !renewed.ExpiresAt.Equal(clock.Add(15*time.Minute)) {
		t.Fatalf("renew after effect fence lease=%#v err=%v", renewed, err)
	}
	staleFence := fence
	staleFence.LeaseEpoch++
	if _, err := store.ValidateFence(ctx, staleFence); !errors.Is(err, ErrLeaseFence) {
		t.Fatalf("stale epoch error=%v, want ErrLeaseFence", err)
	}

	overlap := prepareCanonicalSaga(t, ctx, store, tenantID, projectID, "M3-W003", "attempt-002", "idempotency-002", "b")
	overlap.MaximumExpiry = clock.Add(10 * time.Minute)
	if _, err := store.IssueLease(ctx, "idempotency-002", overlap.RequestDigest, overlap); !errors.Is(err, ErrLeaseConflict) {
		t.Fatalf("overlap error=%v, want ErrLeaseConflict", err)
	}
	if _, err := store.Release(ctx, fence); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if _, err := store.ValidateFence(ctx, fence); !errors.Is(err, ErrLeaseFence) {
		t.Fatalf("released fence error=%v, want ErrLeaseFence", err)
	}

	third := prepareCanonicalSaga(t, ctx, store, tenantID, projectID, "M3-W004", "attempt-003", "idempotency-003", "c")
	third.MaximumExpiry = clock.Add(10 * time.Minute)
	type issueResult struct {
		key     string
		request gateway.LeaseRequest
		saga    gateway.ClaimSaga
		err     error
	}
	results := make(chan issueResult, 2)
	for key, request := range map[string]gateway.LeaseRequest{"idempotency-002": overlap, "idempotency-003": third} {
		key, request := key, request
		go func() {
			saga, err := store.IssueLease(ctx, key, request.RequestDigest, request)
			results <- issueResult{key: key, request: request, saga: saga, err: err}
		}()
	}
	var winner, loser issueResult
	for range 2 {
		result := <-results
		if result.err == nil {
			if winner.key != "" {
				t.Fatal("two overlapping PostgreSQL leases were issued")
			}
			winner = result
			continue
		}
		if !errors.Is(result.err, ErrLeaseConflict) {
			t.Fatalf("concurrent loser error=%v, want ErrLeaseConflict", result.err)
		}
		loser = result
	}
	if winner.key == "" || loser.key == "" || winner.saga.Lease.LeaseEpoch <= firstSaga.Lease.LeaseEpoch {
		t.Fatalf("winner=%#v loser=%#v", winner, loser)
	}
	if _, err := store.Release(ctx, fenceFromLease(winner.saga.Lease)); err != nil {
		t.Fatalf("release concurrent winner: %v", err)
	}
	lastSaga, err := store.IssueLease(ctx, loser.key, loser.request.RequestDigest, loser.request)
	if err != nil {
		t.Fatalf("issue prior loser after release: %v", err)
	}
	if lastSaga.Lease.LeaseEpoch <= winner.saga.Lease.LeaseEpoch {
		t.Fatalf("last epoch=%d, winner=%d", lastSaga.Lease.LeaseEpoch, winner.saga.Lease.LeaseEpoch)
	}

	clock = lastSaga.Lease.ExpiresAt.Add(time.Second)
	if _, err := store.ValidateFence(ctx, fenceFromLease(lastSaga.Lease)); !errors.Is(err, ErrLeaseFence) {
		t.Fatalf("expired fence error=%v, want ErrLeaseFence", err)
	}
	expired, err := store.GetLease(ctx, tenantID, projectID, lastSaga.Lease.LeaseID)
	if err != nil || expired.Active || expired.State != authorityv1.LeaseExpired {
		t.Fatalf("expired lease=%#v err=%v", expired, err)
	}
	fourth := prepareCanonicalSaga(t, ctx, store, tenantID, projectID, "M3-W005", "attempt-004", "idempotency-004", "d")
	fourth.MaximumExpiry = clock.Add(10 * time.Minute)
	fourthSaga, err := store.IssueLease(ctx, "idempotency-004", fourth.RequestDigest, fourth)
	if err != nil {
		t.Fatalf("issue after expiry: %v", err)
	}
	if fourthSaga.Lease.LeaseEpoch <= lastSaga.Lease.LeaseEpoch {
		t.Fatalf("post-expiry epoch=%d, expired=%d", fourthSaga.Lease.LeaseEpoch, lastSaga.Lease.LeaseEpoch)
	}
	revoked, err := store.Revoke(ctx, authorityv1.RevokeLeaseRequest{
		TenantID: tenantID, ProjectID: projectID, LeaseID: fourthSaga.Lease.LeaseID,
		FenceGeneration: generation, LeaseEpoch: fourthSaga.Lease.LeaseEpoch,
		Reason: "security-fixture-revoke", TraceRef: "trace-revoke-001",
	})
	if err != nil || revoked.Active || revoked.State != authorityv1.LeaseRevoked {
		t.Fatalf("revoked lease=%#v err=%v", revoked, err)
	}

	if _, found, err := store.Lookup(ctx, "tenant-other", projectID, "idempotency-002"); err == nil || found {
		t.Fatalf("cross-tenant lookup found=%v err=%v", found, err)
	}
	if _, err := admin.Exec(ctx, `delete from mars3_authority.authority_events where tenant_id=$1 and project_id=$2 and sequence=1`, tenantID, projectID); err != nil {
		t.Fatalf("simulate journal retention delete: %v", err)
	}
	if _, err := admin.Exec(ctx, `update mars3_authority.projects set journal_lowest_retained=2 where tenant_id=$1 and project_id=$2`, tenantID, projectID); err != nil {
		t.Fatalf("simulate journal retention: %v", err)
	}
	if _, err := store.Replay(ctx, tenantID, projectID, authorityv1.JournalCursor{}, 100); !errors.Is(err, ErrJournalTruncated) {
		t.Fatalf("truncated replay error=%v, want ErrJournalTruncated", err)
	}
}

func assertTenantRLS(t *testing.T, ctx context.Context, app *pgxpool.Pool, tenantID, projectID, generation string, clock time.Time) {
	t.Helper()
	tx, err := app.Begin(ctx)
	if err != nil {
		t.Fatalf("begin cross-tenant RLS check: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `select set_config('mars3.tenant_id','tenant-other',true), set_config('mars3.project_id',$1,true)`, projectID); err != nil {
		t.Fatalf("scope cross-tenant RLS check: %v", err)
	}
	var visible int
	if err := tx.QueryRow(ctx, `select count(*) from mars3_authority.projects where tenant_id=$1 and project_id=$2`, tenantID, projectID).Scan(&visible); err != nil || visible != 0 {
		t.Fatalf("cross-tenant read visible=%d err=%v", visible, err)
	}
	command, err := tx.Exec(ctx, `update mars3_authority.projects set barrier_epoch=barrier_epoch+1 where tenant_id=$1 and project_id=$2`, tenantID, projectID)
	if err != nil || command.RowsAffected() != 0 {
		t.Fatalf("cross-tenant update rows=%d err=%v", command.RowsAffected(), err)
	}
	if _, err := tx.Exec(ctx, `insert into mars3_authority.projects (tenant_id,project_id,fence_generation,issuance_enabled,generation_anchored_at) values ($1,$2,$3,true,$4)`, tenantID, "project-cross-tenant-insert", generation, clock); err == nil {
		t.Fatal("cross-tenant insert bypassed RLS WITH CHECK")
	}
}

func prepareCanonicalSaga(t *testing.T, ctx context.Context, store *Store, tenantID, projectID, beadID, attemptID, key, digestRune string) gateway.LeaseRequest {
	t.Helper()
	digest := strings.Repeat(digestRune, 64)
	paths := []string{"internal/authority/"}
	labels := []authorityv1.Label{authorityv1.LabelPublicAccepted}
	intent := gateway.ClaimIntent{
		RequestDigest: digest, TenantID: tenantID, ProjectID: projectID, BeadID: beadID,
		AttemptID: attemptID, IdempotencyKey: key, BaseSHA: strings.Repeat("a", 40),
		Capability: authorityv1.CapabilityTicketDelivery, ExclusivePaths: paths,
		TraceRef: "trace-" + key, Labels: labels,
	}
	if _, err := store.Begin(ctx, intent); err != nil {
		t.Fatalf("Begin %s: %v", key, err)
	}
	work := postgresClaimedWork(tenantID, projectID, beadID, attemptID, digest)
	if _, err := store.MarkCanonicalClaimed(ctx, tenantID, projectID, key, digest, work); err != nil {
		t.Fatalf("MarkCanonicalClaimed %s: %v", key, err)
	}
	return gateway.LeaseRequest{
		RequestDigest: digest, TenantID: tenantID, ProjectID: projectID, BeadID: beadID,
		AttemptID: attemptID, CanonicalClaimAttemptID: attemptID, IdempotencyKey: key, BaseSHA: intent.BaseSHA,
		Capability: intent.Capability, ExclusivePaths: paths, Labels: labels,
		ClaimVersion: work.Version,
	}
}

func postgresClaimedWork(tenantID, projectID, beadID, attemptID, digest string) authorityv1.WorkItem {
	return authorityv1.WorkItem{
		TenantID: tenantID, ProjectID: projectID, BeadID: beadID, DisplayID: strings.TrimPrefix(beadID, "M3-"),
		NativeStatus: "in_progress", LifecycleState: authorityv1.LifecycleInProgress,
		Assignee: "work-authority-engineer", ClaimAttemptID: attemptID,
		GoalIDs: []string{"G-001"}, ProductDecisionIDs: []string{"PD-002"}, FeatureID: "F-002",
		ScenarioIDs: []string{"F-002-S2"}, ExclusivePaths: []string{"internal/authority/"},
		VerificationOrder: []string{"qa", "security-reviewer", "delivery-orchestrator"},
		Labels:            []authorityv1.Label{authorityv1.LabelPublicAccepted},
		Version: authorityv1.WorkVersion{
			AuthorityGeneration: "work-generation-001", IssueIncarnation: "issue-incarnation-001",
			IssueMutationSequence: 2, DependencyGraphRevision: 1,
		},
		Integrity: authorityv1.IntegrityDigests{Lineage: digest, DependencyOutcomes: digest, Blockers: digest, ExclusivePaths: digest},
	}
}

func fenceFromLease(lease authorityv1.CapabilityLease) authorityv1.FencingTuple {
	return authorityv1.FencingTuple{
		TenantID: lease.TenantID, ProjectID: lease.ProjectID, BeadID: lease.BeadID,
		AttemptID: lease.AttemptID, IdempotencyKey: lease.IdempotencyKey, LeaseID: lease.LeaseID,
		CanonicalClaimAttemptID: lease.CanonicalClaimAttemptID,
		FenceGeneration:         lease.FenceGeneration, LeaseEpoch: lease.LeaseEpoch, ClaimVersion: lease.ClaimVersion,
		BaseSHA: lease.BaseSHA, Capability: lease.Capability, ExclusivePaths: append([]string(nil), lease.ExclusivePaths...),
		Labels: append([]authorityv1.Label(nil), lease.Labels...),
	}
}

type baselineStore struct {
	mu           sync.Mutex
	items        []authorityv1.WorkItem
	second       []authorityv1.WorkItem
	firstRead    chan struct{}
	releaseFirst chan struct{}
	reads        int
}

func (store *baselineStore) Get(_ context.Context, tenantID, projectID, beadID string) (authorityv1.WorkItem, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	for _, item := range store.items {
		if item.TenantID == tenantID && item.ProjectID == projectID && item.BeadID == beadID {
			return item, nil
		}
	}
	return authorityv1.WorkItem{}, gateway.ErrWorkNotFound
}

func (store *baselineStore) List(_ context.Context, tenantID, projectID string) ([]authorityv1.WorkItem, error) {
	store.mu.Lock()
	store.reads++
	read := store.reads
	items := store.items
	if read > 1 && store.second != nil {
		items = store.second
	}
	firstRead, releaseFirst := store.firstRead, store.releaseFirst
	store.mu.Unlock()
	if read == 1 && firstRead != nil {
		close(firstRead)
		<-releaseFirst
	}
	result := make([]authorityv1.WorkItem, 0, len(items))
	for _, item := range items {
		if item.TenantID == tenantID && item.ProjectID == projectID {
			result = append(result, item)
		}
	}
	return result, nil
}
