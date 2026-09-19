/*
FactoryDocSync:
docs:
- docs/features/F-002-work-authority.md
- docs/design-docs/ADR-001-git-beads-authority.md
- docs/code-documentation-map.md
*/

package operator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	authorityv1 "github.com/greaveselliott/MARS-3/api/authority/v1"
)

func TestFileReplayStoreSurvivesReopen(t *testing.T) {
	directory := privateReplayFixture(t)
	store, err := NewFileReplayStore(directory)
	if err != nil {
		t.Fatal(err)
	}
	key := strings.Repeat("a", 64)
	if err := store.Consume(context.Background(), key); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := NewFileReplayStore(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if err := reopened.Consume(context.Background(), key); err != ErrReplay {
		t.Fatalf("consumed identity survived as executable: %v", err)
	}
	if err := reopened.Consume(context.Background(), strings.Repeat("b", 64)); err != nil {
		t.Fatal(err)
	}
}

func TestFileReplayStoreConcurrentIndependentHandlesHaveOneWinner(t *testing.T) {
	directory := privateReplayFixture(t)
	first, err := NewFileReplayStore(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	second, err := NewFileReplayStore(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	start := make(chan struct{})
	results := make(chan error, 16)
	for i := 0; i < cap(results); i++ {
		store := first
		if i%2 == 1 {
			store = second
		}
		go func(s *FileReplayStore) {
			<-start
			results <- s.Consume(context.Background(), strings.Repeat("c", 64))
		}(store)
	}
	close(start)
	winners := 0
	for i := 0; i < cap(results); i++ {
		err := <-results
		if err == nil {
			winners++
		} else if err != ErrReplay {
			t.Fatal(err)
		}
	}
	if winners != 1 {
		t.Fatalf("got %d winners, want exactly one", winners)
	}
}

func TestFileReplayStoreRejectsUnsafeDirectoryAndKeys(t *testing.T) {
	directory := privateReplayFixture(t)
	link := filepath.Join(t.TempDir(), "replay-link")
	if err := os.Symlink(directory, link); err != nil {
		t.Fatal(err)
	}
	for _, candidate := range []string{link, "relative", filepath.Join(directory, "missing")} {
		if store, err := NewFileReplayStore(candidate); err != ErrReplay {
			if store != nil {
				store.Close()
			}
			t.Fatalf("unsafe directory admitted: %v", err)
		}
	}
	store, err := NewFileReplayStore(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	for _, key := range []string{"", "../escape", strings.Repeat("A", 64), strings.Repeat("a", 65)} {
		if err := store.Consume(context.Background(), key); err != ErrReplay {
			t.Fatalf("unsafe key admitted: %v", err)
		}
	}
	if err := os.Chmod(directory, 0755); err != nil {
		t.Fatal(err)
	}
	if err := store.Consume(context.Background(), strings.Repeat("d", 64)); err != ErrReplay {
		t.Fatalf("permission drift admitted: %v", err)
	}
}

func TestFileReplayStoreCancelledAttemptAndExistingSymlink(t *testing.T) {
	directory := privateReplayFixture(t)
	store, err := NewFileReplayStore(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	key := strings.Repeat("e", 64)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := store.Consume(ctx, key); err != ErrReplay {
		t.Fatalf("cancelled attempt admitted: %v", err)
	}
	if err := store.Consume(context.Background(), key); err != nil {
		t.Fatalf("cancelled attempt reserved a key: %v", err)
	}
	otherKey := strings.Repeat("f", 64)
	target := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(target, []byte("unchanged"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(directory, "used-"+otherKey)); err != nil {
		t.Fatal(err)
	}
	if err := store.Consume(context.Background(), otherKey); err != ErrReplay {
		t.Fatalf("existing symlink admitted: %v", err)
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "unchanged" {
		t.Fatal("replay store modified symlink target")
	}
}

func privateReplayFixture(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	if err := os.Chmod(directory, 0700); err != nil {
		t.Fatal(err)
	}
	return directory
}

type fixtureGateway struct {
	calls     int
	principal authorityv1.Principal
	err       error
}

type fixtureMutationGateway struct {
	*fixtureGateway
	mutationCalls int
	request       MutationRequest
}

func (backend *fixtureMutationGateway) Mutate(_ context.Context, principal authorityv1.Principal, request MutationRequest) (any, error) {
	backend.mutationCalls++
	backend.principal = principal
	backend.request = request
	return struct{}{}, backend.err
}

func TestMutationGateRoutesOnlySeparatelyAuthorizedOperation(t *testing.T) {
	cases := []struct {
		request    MutationRequest
		capability authorityv1.Capability
	}{
		{MutationRequest{Operation: "work.claim", Claim: &authorityv1.ClaimRequest{}}, authorityv1.CapabilityWorkClaim},
		{MutationRequest{Operation: "lease.renew", Renew: &authorityv1.RenewLeaseRequest{}}, authorityv1.CapabilityLeaseRenew},
		{MutationRequest{Operation: "lease.release", Release: &authorityv1.ReleaseLeaseRequest{}}, authorityv1.CapabilityLeaseRelease},
		{MutationRequest{Operation: "effect.validate", Effect: &authorityv1.EffectValidationRequest{}}, authorityv1.CapabilityEffectValidate},
	}
	for _, test := range cases {
		t.Run(test.request.Operation, func(t *testing.T) {
			gate, readAuth, _, backend, _ := readFixture(t)
			mutations := &fixtureMutationGateway{fixtureGateway: backend}
			gate.gateway = mutations
			request := marshalFixture(t, test.request)
			digest := sha256.Sum256(request)
			auth := MutationAuthorization(readAuth)
			auth.Kind = "MARS3OperatorMutationAuthorization"
			auth.Operation, auth.RequestSHA256 = test.request.Operation, hex.EncodeToString(digest[:])
			document := marshalFixture(t, auth)
			if _, err := gate.ExecuteMutation(context.Background(), document, []byte("fixture"), request); err != nil {
				t.Fatal(err)
			}
			if mutations.mutationCalls != 1 || len(backend.principal.Capabilities) != 1 || backend.principal.Capabilities[0] != test.capability {
				t.Fatal("mutation routing broadened capability")
			}
			if _, err := gate.ExecuteMutation(context.Background(), document, []byte("fixture"), request); err != ErrReplay {
				t.Fatal("mutation replay admitted")
			}
			if mutations.mutationCalls != 1 {
				t.Fatal("replayed mutation reached gateway")
			}
		})
	}
}

func TestMutationGateRejectsReadAuthorityAndAmbiguousPayloads(t *testing.T) {
	cases := []MutationRequest{
		{Operation: "work.claim"},
		{Operation: "work.claim", Claim: &authorityv1.ClaimRequest{}, Release: &authorityv1.ReleaseLeaseRequest{}},
		{Operation: "work.claim", Renew: &authorityv1.RenewLeaseRequest{}},
		{Operation: "work.close", Claim: &authorityv1.ClaimRequest{}},
	}
	for _, candidate := range cases {
		gate, auth, _, backend, replays := readFixture(t)
		mutations := &fixtureMutationGateway{fixtureGateway: backend}
		gate.gateway = mutations
		request := marshalFixture(t, candidate)
		digest := sha256.Sum256(request)
		auth.Kind, auth.Operation, auth.RequestSHA256 = "MARS3OperatorMutationAuthorization", candidate.Operation, hex.EncodeToString(digest[:])
		if _, err := gate.ExecuteMutation(context.Background(), marshalFixture(t, auth), []byte("fixture"), request); err != ErrAuthorization {
			t.Fatal("ambiguous mutation admitted")
		}
		if mutations.mutationCalls != 0 || len(replays.keys) != 0 {
			t.Fatal("invalid payload caused an effect")
		}
	}
	gate, auth, _, backend, _ := readFixture(t)
	mutations := &fixtureMutationGateway{fixtureGateway: backend}
	gate.gateway = mutations
	request := marshalFixture(t, MutationRequest{Operation: "work.claim", Claim: &authorityv1.ClaimRequest{}})
	digest := sha256.Sum256(request)
	auth.Operation, auth.RequestSHA256 = "work.claim", hex.EncodeToString(digest[:])
	if _, err := gate.ExecuteMutation(context.Background(), marshalFixture(t, auth), []byte("fixture"), request); err != ErrAuthorization {
		t.Fatal("read signature escalated to mutation")
	}
	if mutations.mutationCalls != 0 {
		t.Fatal("read authority reached mutation gateway")
	}
}

func TestMutationGateBurnsAttemptOnBackendFailure(t *testing.T) {
	gate, auth, _, backend, _ := readFixture(t)
	backend.err = errors.New("synthetic private failure")
	mutations := &fixtureMutationGateway{fixtureGateway: backend}
	gate.gateway = mutations
	request := marshalFixture(t, MutationRequest{Operation: "lease.release", Release: &authorityv1.ReleaseLeaseRequest{}})
	digest := sha256.Sum256(request)
	auth.Kind, auth.Operation, auth.RequestSHA256 = "MARS3OperatorMutationAuthorization", "lease.release", hex.EncodeToString(digest[:])
	document := marshalFixture(t, auth)
	if _, err := gate.ExecuteMutation(context.Background(), document, []byte("fixture"), request); err != ErrGateway {
		t.Fatal("backend diagnostic escaped")
	}
	if _, err := gate.ExecuteMutation(context.Background(), document, []byte("fixture"), request); err != ErrReplay {
		t.Fatal("failed mutation replayed")
	}
	if mutations.mutationCalls != 1 {
		t.Fatal("uncertain mutation retried")
	}
}

func (g *fixtureGateway) GetWork(_ context.Context, principal authorityv1.Principal, request authorityv1.GetWorkRequest) (authorityv1.WorkItem, error) {
	g.calls++
	g.principal = principal
	return authorityv1.WorkItem{BeadID: request.BeadID}, g.err
}

func (g *fixtureGateway) Ready(_ context.Context, principal authorityv1.Principal, _ authorityv1.ReadyRequest) (authorityv1.ReadyResponse, error) {
	g.calls++
	g.principal = principal
	return authorityv1.ReadyResponse{}, g.err
}

// This in-memory double proves ordering only, not durable replay protection.
type fixtureReplays struct {
	keys         map[string]bool
	err          error
	afterConsume func()
}

func (r *fixtureReplays) Consume(_ context.Context, key string) error {
	if r.err != nil {
		return r.err
	}
	if r.keys[key] {
		return errors.New("already used")
	}
	r.keys[key] = true
	if r.afterConsume != nil {
		r.afterConsume()
	}
	return nil
}

func readFixture(t *testing.T) (*ReadGate, ReadAuthorization, []byte, *fixtureGateway, *fixtureReplays) {
	t.Helper()
	runtime := RuntimeBinding{
		ProfileSHA256: strings.Repeat("e", 64),
		TenantID:      "tenant-fixture", ProjectID: "project-fixture", BaseSHA: strings.Repeat("a", 40),
		TreeSHA: strings.Repeat("b", 40), WorkspaceDigest: strings.Repeat("c", 64),
		NativeBinarySHA256: strings.Repeat("d", 64), FenceGeneration: "generation-fixture",
	}
	backend := &fixtureGateway{}
	replays := &fixtureReplays{keys: map[string]bool{}}
	gate, err := NewReadGate(runtime, backend, replays)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)
	gate.now = func() time.Time { return now }
	// No production API permits caller-supplied signature verification. This
	// package-private substitution isolates semantic admission from cryptography.
	gate.verifySignature = func([]byte, []byte) error { return nil }
	request := marshalFixture(t, ReadRequest{Operation: "work.get", BeadID: "M3-W001", TraceRef: "trace-fixture"})
	digest := sha256.Sum256(request)
	auth := ReadAuthorization{
		SchemaVersion: 1, Kind: "MARS3OperatorReadAuthorization", AuthorizationID: "authorization-fixture",
		IssuedAt: now, ExpiresAt: now.Add(time.Hour), PrincipalID: "operator-fixture", ProfileID: "observer-fixture",
		Runtime: runtime, Operation: "work.get", RequestSHA256: hex.EncodeToString(digest[:]),
	}
	return gate, auth, request, backend, replays
}

func marshalFixture(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestReadGateOneAttemptAndLeastPrivilege(t *testing.T) {
	gate, auth, request, backend, _ := readFixture(t)
	document := marshalFixture(t, auth)
	if _, err := gate.Execute(context.Background(), document, []byte("fixture"), request); err != nil {
		t.Fatal(err)
	}
	if backend.calls != 1 || backend.principal.PrincipalID != auth.PrincipalID ||
		len(backend.principal.Capabilities) != 1 || backend.principal.Capabilities[0] != authorityv1.CapabilityWorkRead {
		t.Fatal("read did not use the signed identity and read-only capability")
	}
	for _, label := range backend.principal.Labels {
		if label == authorityv1.LabelPrivateData || label == authorityv1.LabelPublicAccepted {
			t.Fatal("read elevated label trust")
		}
	}
	if _, err := gate.Execute(context.Background(), document, []byte("fixture"), request); !errors.Is(err, ErrReplay) {
		t.Fatalf("replay admitted: %v", err)
	}
	if backend.calls != 1 {
		t.Fatal("replay reached gateway")
	}
}

func TestReadGateRejectsWrongScopeAndWindow(t *testing.T) {
	cases := map[string]func(*ReadAuthorization){
		"expired": func(a *ReadAuthorization) {
			a.IssuedAt = a.IssuedAt.Add(-time.Hour)
			a.ExpiresAt = a.ExpiresAt.Add(-time.Hour)
		},
		"future":            func(a *ReadAuthorization) { a.IssuedAt = a.IssuedAt.Add(time.Second) },
		"long-window":       func(a *ReadAuthorization) { a.ExpiresAt = a.ExpiresAt.Add(time.Second) },
		"empty-window":      func(a *ReadAuthorization) { a.ExpiresAt = a.IssuedAt },
		"tenant":            func(a *ReadAuthorization) { a.Runtime.TenantID = "other" },
		"project":           func(a *ReadAuthorization) { a.Runtime.ProjectID = "other" },
		"base":              func(a *ReadAuthorization) { a.Runtime.BaseSHA = strings.Repeat("e", 40) },
		"tree":              func(a *ReadAuthorization) { a.Runtime.TreeSHA = strings.Repeat("e", 40) },
		"workspace":         func(a *ReadAuthorization) { a.Runtime.WorkspaceDigest = strings.Repeat("e", 64) },
		"binary":            func(a *ReadAuthorization) { a.Runtime.NativeBinarySHA256 = strings.Repeat("e", 64) },
		"fence-generation":  func(a *ReadAuthorization) { a.Runtime.FenceGeneration = "other" },
		"principal":         func(a *ReadAuthorization) { a.PrincipalID = "" },
		"profile":           func(a *ReadAuthorization) { a.ProfileID = "" },
		"operation":         func(a *ReadAuthorization) { a.Operation = "work.claim" },
		"request-digest":    func(a *ReadAuthorization) { a.RequestSHA256 = strings.Repeat("e", 64) },
		"source-grant-kind": func(a *ReadAuthorization) { a.Kind = "MARS3W001OperatorRecoveryV1Grant" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			gate, auth, request, backend, replays := readFixture(t)
			mutate(&auth)
			if _, err := gate.Execute(context.Background(), marshalFixture(t, auth), []byte("fixture"), request); !errors.Is(err, ErrAuthorization) {
				t.Fatalf("invalid authorization admitted: %v", err)
			}
			if backend.calls != 0 || len(replays.keys) != 0 {
				t.Fatal("denial caused an effect")
			}
		})
	}
}

func TestReadGateRejectsAmbiguousAndPrincipalBearingRequests(t *testing.T) {
	requests := []string{
		`{"operation":"work.get","bead_id":"M3-W001","trace_ref":"trace-fixture","principal":{}}`,
		`{"Operation":"work.get","bead_id":"M3-W001","trace_ref":"trace-fixture"}`,
		`{"operation":"work.get","operation":"work.get","bead_id":"M3-W001","trace_ref":"trace-fixture"}`,
		`{"operation":"work.get","bead_id":"M3-W001","trace_ref":"trace-fixture"} {}`,
		`{"operation":"work.claim","bead_id":"M3-W001","trace_ref":"trace-fixture"}`,
	}
	for _, text := range requests {
		gate, auth, _, backend, _ := readFixture(t)
		request := []byte(text)
		digest := sha256.Sum256(request)
		auth.RequestSHA256 = hex.EncodeToString(digest[:])
		if _, err := gate.Execute(context.Background(), marshalFixture(t, auth), []byte("fixture"), request); !errors.Is(err, ErrAuthorization) {
			t.Fatalf("ambiguous request admitted: %v", err)
		}
		if backend.calls != 0 {
			t.Fatal("denial reached gateway")
		}
	}
}

func TestReadGateProductionVerifierRejectsMissingAndForgedSignatures(t *testing.T) {
	fixture, auth, request, backend, replays := readFixture(t)
	gate, err := NewReadGate(fixture.runtime, backend, replays)
	if err != nil {
		t.Fatal(err)
	}
	gate.now = fixture.now
	for _, signature := range [][]byte{nil, []byte("forged"), []byte("-----BEGIN SSH SIGNATURE-----\nZm9yZ2Vk\n-----END SSH SIGNATURE-----")} {
		if _, err := gate.Execute(context.Background(), marshalFixture(t, auth), signature, request); !errors.Is(err, ErrAuthorization) {
			t.Fatalf("invalid signature admitted: %v", err)
		}
	}
	if backend.calls != 0 || len(replays.keys) != 0 {
		t.Fatal("invalid signature caused an effect")
	}
}

func TestReadGateFailClosedAtReplayAndExpiryBoundary(t *testing.T) {
	gate, auth, request, backend, replays := readFixture(t)
	replays.err = errors.New("synthetic private backend diagnostic")
	if _, err := gate.Execute(context.Background(), marshalFixture(t, auth), []byte("fixture"), request); err != ErrReplay {
		t.Fatalf("replay store error escaped: %v", err)
	}
	replays.err = nil
	replays.afterConsume = func() { gate.now = func() time.Time { return auth.ExpiresAt } }
	if _, err := gate.Execute(context.Background(), marshalFixture(t, auth), []byte("fixture"), request); err != ErrAuthorization {
		t.Fatalf("expiry during consumption admitted: %v", err)
	}
	if backend.calls != 0 {
		t.Fatal("expired authorization reached gateway")
	}
}

func TestReadGateReadyAndBackendErrorRedaction(t *testing.T) {
	gate, auth, _, backend, _ := readFixture(t)
	request := marshalFixture(t, ReadRequest{Operation: "work.ready", TraceRef: "trace-fixture"})
	digest := sha256.Sum256(request)
	auth.Operation = "work.ready"
	auth.RequestSHA256 = hex.EncodeToString(digest[:])
	backend.err = errors.New("synthetic private backend diagnostic")
	if _, err := gate.Execute(context.Background(), marshalFixture(t, auth), []byte("fixture"), request); err != ErrGateway {
		t.Fatalf("backend error escaped: %v", err)
	}
	if backend.calls != 1 {
		t.Fatal("ready request did not reach gateway")
	}
}

func TestProfileDecodingBindsExactBytesAndLifetime(t *testing.T) {
	now := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)
	profile := Profile{
		SchemaVersion: 1, Kind: "MARS3OperatorProfile", IssuedAt: now, ExpiresAt: now.Add(time.Hour),
		TenantID: "tenant-fixture", ProjectID: "project-fixture", BaseSHA: strings.Repeat("a", 40), TreeSHA: strings.Repeat("b", 40),
		LauncherSHA256: strings.Repeat("c", 64), Workspace: "/synthetic/workspace", WorkspaceDigest: strings.Repeat("d", 64),
		NativeBinary: "/synthetic/bin/bd", NativeBinarySHA256: strings.Repeat("e", 64), FenceGeneration: "generation-fixture",
		PostgresURLFile: "/synthetic/operator/connection", ConnectionSHA256: strings.Repeat("f", 64), ReplayDirectory: "/synthetic/operator/replays",
	}
	document := marshalFixture(t, profile)
	_, runtime, err := decodeProfile(document, now)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(document)
	if runtime.ProfileSHA256 != hex.EncodeToString(digest[:]) {
		t.Fatal("profile digest not bound")
	}
	if _, _, err := decodeProfile(append(document, '\n'), now); err != ErrAuthorization {
		t.Fatal("noncanonical profile admitted")
	}
	if _, _, err := decodeProfile(document, profile.ExpiresAt); err != ErrAuthorization {
		t.Fatal("expired profile admitted")
	}
	profile.ExpiresAt = now.Add(7*24*time.Hour + time.Second)
	if _, _, err := decodeProfile(marshalFixture(t, profile), now); err != ErrAuthorization {
		t.Fatal("overlong profile admitted")
	}
}

func TestProtectedFileRejectsPermissionsSymlinksAndOversize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fixture.json")
	if err := os.WriteFile(path, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readProtectedFile(path, 16); err != nil {
		t.Fatal(err)
	}
	if _, err := readProtectedFile(path, 1); err != ErrAuthorization {
		t.Fatal("oversize file admitted")
	}
	link := filepath.Join(t.TempDir(), "fixture-link")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if _, err := readProtectedFile(link, 16); err != ErrAuthorization {
		t.Fatal("symlink admitted")
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := readProtectedFile(path, 16); err != ErrAuthorization {
		t.Fatal("public file admitted")
	}
}

func TestLocalPostgresConfigRejectsAmbientAndRemoteConfiguration(t *testing.T) {
	for _, entry := range os.Environ() {
		name, value, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(name, "PG") {
			if err := os.Unsetenv(name); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { os.Setenv(name, value) })
		}
	}
	local := []byte("postgresql://operator_fixture@127.0.0.1:5432/authority_fixture?sslmode=disable")
	config, err := localPostgresConfig(local)
	if err != nil || config.ConnConfig.Host != "127.0.0.1" || config.MaxConns != 2 {
		t.Fatal("explicit local connection rejected")
	}
	for _, uri := range []string{
		"postgresql://operator_fixture@example.invalid:5432/authority_fixture?sslmode=disable",
		"postgresql://operator_fixture@127.0.0.1:5432/authority_fixture?sslmode=disable&service=fixture",
		"postgresql://operator_fixture@127.0.0.1:5432/authority_fixture?sslmode=disable&passfile=/synthetic/private",
		"postgresql://operator_fixture@127.0.0.1:5432/authority_fixture?sslmode=disable&host=example.invalid",
	} {
		if _, err := localPostgresConfig([]byte(uri)); err != ErrGateway {
			t.Fatal("unsafe connection admitted")
		}
	}
	t.Setenv("PGHOST", "example.invalid")
	if _, err := localPostgresConfig(local); err != ErrGateway {
		t.Fatal("ambient override admitted")
	}
}

func TestOperatorCLIRejectsIncompleteInputsBeforeConnecting(t *testing.T) {
	for _, args := range [][]string{nil, {"unexpected"}, {"--connection", "forbidden"}, {"--profile", "/synthetic/missing"}} {
		if err := RunCLI(args, &strings.Builder{}); err != ErrAuthorization {
			t.Fatalf("incomplete CLI admitted: %v", err)
		}
	}
}
