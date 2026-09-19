/*
FactoryDocSync:
docs:
- docs/features/F-002-work-authority.md
- docs/design-docs/ADR-001-git-beads-authority.md
- docs/code-documentation-map.md
*/

// Package operator admits explicitly signed operator requests before invoking
// the existing gateway. It does not grant source authority, claims, or leases.
package operator

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	authorityv1 "github.com/greaveselliott/MARS-3/api/authority/v1"
	"github.com/greaveselliott/MARS-3/internal/authority/beads"
	"github.com/greaveselliott/MARS-3/internal/authority/gateway"
	"github.com/greaveselliott/MARS-3/internal/authority/postgres"
	"github.com/greaveselliott/MARS-3/internal/doctrine"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrAuthorization = errors.New("operator authorization denied")
	ErrReplay        = errors.New("operator authorization replay or unavailable replay protection")
	ErrGateway       = errors.New("operator gateway unavailable")
	identifier       = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
	commitDigest     = regexp.MustCompile(`^[a-f0-9]{40}$`)
	contentDigest    = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

// RuntimeBinding must be resolved by the trusted launcher, not the request.
// Digests identify public artifacts or opaque workspace identity, not secrets.
type RuntimeBinding struct {
	ProfileSHA256      string `json:"profile_sha256"`
	TenantID           string `json:"tenant_id"`
	ProjectID          string `json:"project_id"`
	BaseSHA            string `json:"base_sha"`
	TreeSHA            string `json:"tree_sha"`
	WorkspaceDigest    string `json:"workspace_digest"`
	NativeBinarySHA256 string `json:"native_binary_sha256"`
	FenceGeneration    string `json:"fence_generation"`
}

// ReadAuthorization uses exactly json.Marshal's encoding, without a newline.
// It requires a separate operator-execution signature, not a source-grant
// signature. One authorization admits at most one attempt, including failures.
type ReadAuthorization struct {
	SchemaVersion   int            `json:"schema_version"`
	Kind            string         `json:"kind"`
	AuthorizationID string         `json:"authorization_id"`
	IssuedAt        time.Time      `json:"issued_at"`
	ExpiresAt       time.Time      `json:"expires_at"`
	PrincipalID     string         `json:"principal_id"`
	ProfileID       string         `json:"profile_id"`
	Runtime         RuntimeBinding `json:"runtime"`
	Operation       string         `json:"operation"`
	RequestSHA256   string         `json:"request_sha256"`
}

// ReadRequest deliberately has no principal, capability, or label fields.
type ReadRequest struct {
	Operation string `json:"operation"`
	BeadID    string `json:"bead_id,omitempty"`
	TraceRef  string `json:"trace_ref"`
}

// MutationAuthorization has the same envelope fields as a read authorization,
// but requires kind MARS3OperatorMutationAuthorization and one exact operation.
// A read document, source grant, or profile cannot substitute for this signature.
type MutationAuthorization ReadAuthorization

type MutationRequest struct {
	Operation string                               `json:"operation"`
	Claim     *authorityv1.ClaimRequest            `json:"claim,omitempty"`
	Renew     *authorityv1.RenewLeaseRequest       `json:"renew,omitempty"`
	Release   *authorityv1.ReleaseLeaseRequest     `json:"release,omitempty"`
	Effect    *authorityv1.EffectValidationRequest `json:"effect,omitempty"`
}

// MutationGateway is a trusted adapter to existing gateway methods, not a
// datastore or arbitrary-effect interface. It cannot perform a real tool effect.
type MutationGateway interface {
	Mutate(context.Context, authorityv1.Principal, MutationRequest) (any, error)
}

func (request MutationRequest) capability() (authorityv1.Capability, bool) {
	count := 0
	if request.Claim != nil {
		count++
	}
	if request.Renew != nil {
		count++
	}
	if request.Release != nil {
		count++
	}
	if request.Effect != nil {
		count++
	}
	if count != 1 {
		return "", false
	}
	switch request.Operation {
	case "work.claim":
		return authorityv1.CapabilityWorkClaim, request.Claim != nil
	case "lease.renew":
		return authorityv1.CapabilityLeaseRenew, request.Renew != nil
	case "lease.release":
		return authorityv1.CapabilityLeaseRelease, request.Release != nil
	case "effect.validate":
		return authorityv1.CapabilityEffectValidate, request.Effect != nil
	default:
		return "", false
	}
}

// ReadGateway is implemented by the existing gateway.Service. Datastore handles
// must never be passed here or returned to the request's caller.
type ReadGateway interface {
	GetWork(context.Context, authorityv1.Principal, authorityv1.GetWorkRequest) (authorityv1.WorkItem, error)
	Ready(context.Context, authorityv1.Principal, authorityv1.ReadyRequest) (authorityv1.ReadyResponse, error)
}

// ReplayStore is trusted operator infrastructure, not canonical work state.
// Consume must atomically and durably reserve a key across processes before
// returning nil. Existing keys and uncertain persistence must return an error.
// Entries cannot be deleted while any corresponding authorization may be live.
type ReplayStore interface {
	Consume(context.Context, string) error
}

// FileReplayStore retains consumed identities in a private, operator-owned
// directory. It is local admission state, never a canonical work or lease store.
// The directory must already exist on a filesystem supporting exclusive create
// and file/directory fsync. Entries are deliberately never removed here.
type FileReplayStore struct {
	root      *os.Root
	directory *os.File
}

func NewFileReplayStore(directory string) (*FileReplayStore, error) {
	if !filepath.IsAbs(directory) || filepath.Clean(directory) != directory {
		return nil, ErrReplay
	}
	before, err := os.Lstat(directory)
	if err != nil || !privateReplayDirectory(before) {
		return nil, ErrReplay
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, ErrReplay
	}
	after, err := root.Stat(".")
	if err != nil || !privateReplayDirectory(after) || !os.SameFile(before, after) {
		root.Close()
		return nil, ErrReplay
	}
	handle, err := root.Open(".")
	if err != nil {
		root.Close()
		return nil, ErrReplay
	}
	return &FileReplayStore{root: root, directory: handle}, nil
}

func privateReplayDirectory(info os.FileInfo) bool {
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 {
		return false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && uint64(stat.Uid) == uint64(os.Geteuid())
}

func (s *FileReplayStore) Consume(ctx context.Context, key string) error {
	if s == nil || s.root == nil || s.directory == nil || !contentDigest.MatchString(key) || ctx.Err() != nil {
		return ErrReplay
	}
	info, err := s.root.Stat(".")
	if err != nil || !privateReplayDirectory(info) {
		return ErrReplay
	}
	file, err := s.root.OpenFile("used-"+key, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return ErrReplay
	}
	// Never remove an entry on an uncertain write, sync, close, or cancellation.
	// A crash after reservation can burn an unused authorization, but cannot
	// permit an executed operation to be replayed after successful fsync.
	written, writeErr := file.WriteString("consumed\n")
	syncErr := file.Sync()
	closeErr := file.Close()
	if writeErr != nil || written != len("consumed\n") || syncErr != nil || closeErr != nil {
		return ErrReplay
	}
	if s.directory.Sync() != nil || ctx.Err() != nil {
		return ErrReplay
	}
	return nil
}

func (s *FileReplayStore) Close() error {
	if s == nil || s.root == nil || s.directory == nil {
		return ErrReplay
	}
	directoryErr := s.directory.Close()
	rootErr := s.root.Close()
	if directoryErr != nil || rootErr != nil {
		return ErrReplay
	}
	return nil
}

// ReadGate's constructor is for trusted composition, never request deserialization.
type ReadGate struct {
	runtime         RuntimeBinding
	gateway         ReadGateway
	replays         ReplayStore
	now             func() time.Time
	verifySignature func([]byte, []byte) error
}

func NewReadGate(runtime RuntimeBinding, gateway ReadGateway, replays ReplayStore) (*ReadGate, error) {
	if !validRuntime(runtime) || gateway == nil || replays == nil {
		return nil, ErrAuthorization
	}
	return &ReadGate{
		runtime: runtime, gateway: gateway, replays: replays, now: time.Now,
		verifySignature: doctrine.VerifyOperatorExecutionSignature,
	}, nil
}

// Execute admits only work.get and work.ready. Gateway reads still append their
// normal bounded audit events; this is not a raw or audit-free datastore read.
// No backend, signature-parser, or replay-store error text escapes this boundary.
func (g *ReadGate) Execute(ctx context.Context, document, signature, request []byte) (any, error) {
	if g == nil || g.gateway == nil || g.replays == nil || g.now == nil || g.verifySignature == nil ||
		len(document) == 0 || len(document) > 16384 || len(signature) == 0 || len(signature) > 4096 ||
		len(request) == 0 || len(request) > 4096 || ctx.Err() != nil {
		return nil, ErrAuthorization
	}
	document = bytes.Clone(document)
	signature = bytes.Clone(signature)
	request = bytes.Clone(request)
	if g.verifySignature(document, signature) != nil {
		return nil, ErrAuthorization
	}
	var auth ReadAuthorization
	var read ReadRequest
	if !canonicalJSON(document, &auth) || !canonicalJSON(request, &read) ||
		!g.admits(auth, read, request, g.now()) {
		return nil, ErrAuthorization
	}
	key := sha256.Sum256([]byte(auth.Runtime.TenantID + "\x00" + auth.Runtime.ProjectID + "\x00" + auth.AuthorizationID))
	if g.replays.Consume(ctx, hex.EncodeToString(key[:])) != nil {
		return nil, ErrReplay
	}
	// Consumption is fail-closed: expiry or a backend failure burns this attempt.
	if ctx.Err() != nil || !validWindow(auth, g.now()) {
		return nil, ErrAuthorization
	}
	principal := authorityv1.Principal{
		TenantID: auth.Runtime.TenantID, ProjectID: auth.Runtime.ProjectID,
		PrincipalID: auth.PrincipalID, ProfileID: auth.ProfileID,
		Capabilities: []authorityv1.Capability{authorityv1.CapabilityWorkRead},
		Labels:       []authorityv1.Label{authorityv1.LabelExternalUntrusted, authorityv1.LabelExternalEffect},
	}
	var result any
	var err error
	switch read.Operation {
	case "work.get":
		result, err = g.gateway.GetWork(ctx, principal, authorityv1.GetWorkRequest{BeadID: read.BeadID, TraceRef: read.TraceRef})
	case "work.ready":
		result, err = g.gateway.Ready(ctx, principal, authorityv1.ReadyRequest{TraceRef: read.TraceRef})
	}
	if err != nil {
		return nil, ErrGateway
	}
	return result, nil
}

// ExecuteMutation authenticates a separate one-attempt document and delegates
// unchanged typed requests to the normal gateway. A signature supplies no
// canonical claim, lease, epoch, or bypass of gateway admission.
func (g *ReadGate) ExecuteMutation(ctx context.Context, document, signature, request []byte) (any, error) {
	if g == nil || g.gateway == nil || g.replays == nil || g.now == nil || g.verifySignature == nil ||
		len(document) == 0 || len(document) > 16384 || len(signature) == 0 || len(signature) > 4096 ||
		len(request) == 0 || len(request) > 64<<10 || ctx.Err() != nil {
		return nil, ErrAuthorization
	}
	backend, ok := g.gateway.(MutationGateway)
	if !ok {
		return nil, ErrAuthorization
	}
	document, signature, request = bytes.Clone(document), bytes.Clone(signature), bytes.Clone(request)
	if g.verifySignature(document, signature) != nil {
		return nil, ErrAuthorization
	}
	var auth MutationAuthorization
	var mutation MutationRequest
	if !canonicalJSON(document, &auth) || !canonicalJSON(request, &mutation) ||
		auth.SchemaVersion != 1 || auth.Kind != "MARS3OperatorMutationAuthorization" ||
		!identifier.MatchString(auth.AuthorizationID) || !identifier.MatchString(auth.PrincipalID) ||
		!identifier.MatchString(auth.ProfileID) || auth.Runtime != g.runtime || !validRuntime(auth.Runtime) ||
		!validWindow(ReadAuthorization(auth), g.now()) || auth.Operation != mutation.Operation {
		return nil, ErrAuthorization
	}
	digest := sha256.Sum256(request)
	if auth.RequestSHA256 != hex.EncodeToString(digest[:]) {
		return nil, ErrAuthorization
	}
	capability, ok := mutation.capability()
	if !ok {
		return nil, ErrAuthorization
	}
	key := sha256.Sum256([]byte(auth.Runtime.TenantID + "\x00" + auth.Runtime.ProjectID + "\x00" + auth.AuthorizationID))
	if g.replays.Consume(ctx, hex.EncodeToString(key[:])) != nil {
		return nil, ErrReplay
	}
	if ctx.Err() != nil || !validWindow(ReadAuthorization(auth), g.now()) {
		return nil, ErrAuthorization
	}
	// Bound backend setup and all typed gateway calls by remaining authorization
	// lifetime as well as the CLI deadline. Never retry an uncertain mutation here.
	ctx, cancel := context.WithTimeout(ctx, auth.ExpiresAt.Sub(g.now()))
	defer cancel()
	principal := authorityv1.Principal{
		TenantID: auth.Runtime.TenantID, ProjectID: auth.Runtime.ProjectID,
		PrincipalID: auth.PrincipalID, ProfileID: auth.ProfileID,
		Capabilities: []authorityv1.Capability{capability},
		Labels:       []authorityv1.Label{authorityv1.LabelExternalUntrusted, authorityv1.LabelExternalEffect},
	}
	result, err := backend.Mutate(ctx, principal, mutation)
	if err != nil {
		return nil, ErrGateway
	}
	return result, nil
}

func (g *ReadGate) admits(auth ReadAuthorization, read ReadRequest, request []byte, now time.Time) bool {
	if auth.SchemaVersion != 1 || auth.Kind != "MARS3OperatorReadAuthorization" ||
		!identifier.MatchString(auth.AuthorizationID) || !identifier.MatchString(auth.PrincipalID) ||
		!identifier.MatchString(auth.ProfileID) || auth.Runtime != g.runtime || !validRuntime(auth.Runtime) ||
		!validWindow(auth, now) || auth.Operation != read.Operation || !identifier.MatchString(read.TraceRef) ||
		!contentDigest.MatchString(auth.RequestSHA256) {
		return false
	}
	digest := sha256.Sum256(request)
	if auth.RequestSHA256 != hex.EncodeToString(digest[:]) {
		return false
	}
	switch read.Operation {
	case "work.get":
		return identifier.MatchString(read.BeadID)
	case "work.ready":
		return read.BeadID == ""
	default:
		return false
	}
}

func validRuntime(runtime RuntimeBinding) bool {
	return contentDigest.MatchString(runtime.ProfileSHA256) && identifier.MatchString(runtime.TenantID) && identifier.MatchString(runtime.ProjectID) &&
		commitDigest.MatchString(runtime.BaseSHA) && commitDigest.MatchString(runtime.TreeSHA) &&
		contentDigest.MatchString(runtime.WorkspaceDigest) && contentDigest.MatchString(runtime.NativeBinarySHA256) &&
		identifier.MatchString(runtime.FenceGeneration)
}

func validWindow(auth ReadAuthorization, now time.Time) bool {
	return !auth.IssuedAt.IsZero() && auth.ExpiresAt.After(auth.IssuedAt) &&
		auth.ExpiresAt.Sub(auth.IssuedAt) <= time.Hour && !now.Before(auth.IssuedAt) && now.Before(auth.ExpiresAt)
}

func canonicalJSON[T any](data []byte, value *T) bool {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if decoder.Decode(value) != nil {
		return false
	}
	encoded, err := json.Marshal(value)
	return err == nil && bytes.Equal(data, encoded)
}

// Profile is private operator configuration. Real profiles and their paths or
// connection digests must never be committed, logged, or included in responses.
// The source grant is not a profile or an execution authorization.
type Profile struct {
	SchemaVersion      int       `json:"schema_version"`
	Kind               string    `json:"kind"`
	IssuedAt           time.Time `json:"issued_at"`
	ExpiresAt          time.Time `json:"expires_at"`
	TenantID           string    `json:"tenant_id"`
	ProjectID          string    `json:"project_id"`
	BaseSHA            string    `json:"base_sha"`
	TreeSHA            string    `json:"tree_sha"`
	LauncherSHA256     string    `json:"launcher_sha256"`
	Workspace          string    `json:"workspace"`
	WorkspaceDigest    string    `json:"workspace_digest"`
	NativeBinary       string    `json:"native_binary"`
	NativeBinarySHA256 string    `json:"native_binary_sha256"`
	FenceGeneration    string    `json:"fence_generation"`
	PostgresURLFile    string    `json:"postgres_url_file"`
	ConnectionSHA256   string    `json:"connection_sha256"`
	ReplayDirectory    string    `json:"replay_directory"`
}

// RunCLI accepts protected file handles, never a raw connection string, caller
// principal, or execution bypass. It exposes no listener or database setup path.
func RunCLI(args []string, output io.Writer) error {
	flags := flag.NewFlagSet("operator", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	profilePath := flags.String("profile", "", "protected signed operator profile")
	authorizationPath := flags.String("authorization", "", "protected signed one-attempt authorization")
	requestPath := flags.String("request", "", "protected canonical JSON read request")
	mode := flags.String("mode", "read", "read or separately authorized mutation")
	if flags.Parse(args) != nil || flags.NArg() != 0 || *profilePath == "" || *authorizationPath == "" || *requestPath == "" || output == nil || (*mode != "read" && *mode != "mutation") {
		return ErrAuthorization
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	profile, runtime, err := loadProfile(*profilePath, time.Now())
	if err != nil {
		return ErrAuthorization
	}
	document, err := readProtectedFile(*authorizationPath, 16384)
	if err != nil {
		return ErrAuthorization
	}
	signature, err := readProtectedFile(*authorizationPath+".sig", 4096)
	if err != nil {
		return ErrAuthorization
	}
	request, err := readProtectedFile(*requestPath, 64<<10)
	if err != nil {
		return ErrAuthorization
	}
	replays, err := NewFileReplayStore(profile.ReplayDirectory)
	if err != nil {
		return ErrReplay
	}
	defer replays.Close()
	backend := &lazyReadGateway{profile: profile, mutations: *mode == "mutation"}
	defer backend.close()
	gate, err := NewReadGate(runtime, backend, replays)
	if err != nil {
		return ErrAuthorization
	}
	var result any
	if *mode == "mutation" {
		result, err = gate.ExecuteMutation(ctx, document, signature, request)
	} else {
		result, err = gate.Execute(ctx, document, signature, request)
	}
	if err != nil {
		return err
	}
	if json.NewEncoder(output).Encode(result) != nil {
		return ErrGateway
	}
	return nil
}

func loadProfile(path string, now time.Time) (Profile, RuntimeBinding, error) {
	document, err := readProtectedFile(path, 16384)
	if err != nil {
		return Profile{}, RuntimeBinding{}, ErrAuthorization
	}
	signature, err := readProtectedFile(path+".sig", 4096)
	if err != nil || doctrine.VerifyOperatorProfileSignature(document, signature) != nil {
		return Profile{}, RuntimeBinding{}, ErrAuthorization
	}
	profile, runtime, err := decodeProfile(document, now)
	if err != nil {
		return Profile{}, RuntimeBinding{}, err
	}
	executable, err := os.Executable()
	if err != nil || !matchesExecutable(executable, profile.LauncherSHA256) ||
		!matchesExecutable(profile.NativeBinary, profile.NativeBinarySHA256) {
		return Profile{}, RuntimeBinding{}, ErrAuthorization
	}
	digest, err := workspaceIdentity(profile.Workspace, profile.TenantID, profile.ProjectID)
	if err != nil || digest != profile.WorkspaceDigest {
		return Profile{}, RuntimeBinding{}, ErrAuthorization
	}
	return profile, runtime, nil
}

func decodeProfile(document []byte, now time.Time) (Profile, RuntimeBinding, error) {
	var profile Profile
	if !canonicalJSON(document, &profile) || profile.SchemaVersion != 1 || profile.Kind != "MARS3OperatorProfile" ||
		!validProfileWindow(profile, now) || !contentDigest.MatchString(profile.LauncherSHA256) ||
		!contentDigest.MatchString(profile.ConnectionSHA256) {
		return Profile{}, RuntimeBinding{}, ErrAuthorization
	}
	for _, path := range []string{profile.Workspace, profile.NativeBinary, profile.PostgresURLFile, profile.ReplayDirectory} {
		if !filepath.IsAbs(path) || filepath.Clean(path) != path {
			return Profile{}, RuntimeBinding{}, ErrAuthorization
		}
	}
	digest := sha256.Sum256(document)
	runtime := RuntimeBinding{
		ProfileSHA256: hex.EncodeToString(digest[:]), TenantID: profile.TenantID, ProjectID: profile.ProjectID,
		BaseSHA: profile.BaseSHA, TreeSHA: profile.TreeSHA, WorkspaceDigest: profile.WorkspaceDigest,
		NativeBinarySHA256: profile.NativeBinarySHA256, FenceGeneration: profile.FenceGeneration,
	}
	if !validRuntime(runtime) {
		return Profile{}, RuntimeBinding{}, ErrAuthorization
	}
	return profile, runtime, nil
}

func validProfileWindow(profile Profile, now time.Time) bool {
	return !profile.IssuedAt.IsZero() && profile.ExpiresAt.After(profile.IssuedAt) &&
		profile.ExpiresAt.Sub(profile.IssuedAt) <= 7*24*time.Hour &&
		!now.Before(profile.IssuedAt) && now.Before(profile.ExpiresAt)
}

func readProtectedFile(path string, limit int64) ([]byte, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return nil, ErrAuthorization
	}
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, ErrAuthorization
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() <= 0 || info.Size() > limit {
		return nil, ErrAuthorization
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || uint64(stat.Uid) != uint64(os.Geteuid()) {
		return nil, ErrAuthorization
	}
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil || int64(len(data)) != info.Size() || int64(len(data)) > limit {
		return nil, ErrAuthorization
	}
	return data, nil
}

func matchesExecutable(path, expected string) bool {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || !contentDigest.MatchString(expected) {
		return false
	}
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return false
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 || info.Mode().Perm()&0022 != 0 || info.Size() > 256<<20 {
		return false
	}
	hash := sha256.New()
	size, err := io.Copy(hash, io.LimitReader(file, (256<<20)+1))
	return err == nil && size == info.Size() && hex.EncodeToString(hash.Sum(nil)) == expected
}

func workspaceIdentity(workspace, tenant, project string) (string, error) {
	if !filepath.IsAbs(workspace) || filepath.Clean(workspace) != workspace {
		return "", ErrAuthorization
	}
	var identity strings.Builder
	fmt.Fprintf(&identity, "mars3-operator-workspace-v1\n%s\n%s\n", tenant, project)
	for _, relative := range []string{".", ".beads", ".beads/embeddeddolt", ".beads/embeddeddolt/M3"} {
		info, err := os.Lstat(filepath.Join(workspace, relative))
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return "", ErrAuthorization
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			return "", ErrAuthorization
		}
		fmt.Fprintf(&identity, "%d:%d\n", stat.Dev, stat.Ino)
	}
	digest := sha256.Sum256([]byte(identity.String()))
	return hex.EncodeToString(digest[:]), nil
}

type lazyReadGateway struct {
	mutations bool
	profile   Profile
	service   *gateway.Service
	cleanup   func()
}

func (backend *lazyReadGateway) get(ctx context.Context) (*gateway.Service, error) {
	if backend.service != nil {
		return backend.service, nil
	}
	if !validProfileWindow(backend.profile, time.Now()) {
		return nil, ErrAuthorization
	}
	service, cleanup, err := connectOperatorGateway(ctx, backend.profile, backend.mutations)
	if err != nil {
		return nil, ErrGateway
	}
	backend.service, backend.cleanup = service, cleanup
	return service, nil
}

func (backend *lazyReadGateway) close() {
	if backend.cleanup != nil {
		backend.cleanup()
	}
}

func (backend *lazyReadGateway) GetWork(ctx context.Context, principal authorityv1.Principal, request authorityv1.GetWorkRequest) (authorityv1.WorkItem, error) {
	service, err := backend.get(ctx)
	if err != nil {
		return authorityv1.WorkItem{}, err
	}
	return service.GetWork(ctx, principal, request)
}

func (backend *lazyReadGateway) Ready(ctx context.Context, principal authorityv1.Principal, request authorityv1.ReadyRequest) (authorityv1.ReadyResponse, error) {
	service, err := backend.get(ctx)
	if err != nil {
		return authorityv1.ReadyResponse{}, err
	}
	return service.Ready(ctx, principal, request)
}

func (backend *lazyReadGateway) Mutate(ctx context.Context, principal authorityv1.Principal, request MutationRequest) (any, error) {
	if _, valid := request.capability(); !valid || !backend.mutations {
		return nil, ErrAuthorization
	}
	service, err := backend.get(ctx)
	if err != nil {
		return nil, err
	}
	switch request.Operation {
	case "work.claim":
		return service.Claim(ctx, principal, *request.Claim)
	case "lease.renew":
		return service.RenewLease(ctx, principal, *request.Renew)
	case "lease.release":
		return service.ReleaseLease(ctx, principal, *request.Release)
	case "effect.validate":
		return service.ValidateEffect(ctx, principal, *request.Effect)
	default:
		return nil, ErrAuthorization
	}
}

// This adapter cannot invoke a native mutation, even if incorrectly called.
type readOnlyMutator struct{}

func (readOnlyMutator) CompareAndSwapClaim(context.Context, beads.AtomicClaim) ([]byte, error) {
	return nil, beads.ErrAtomicCASRequired
}

func connectOperatorGateway(ctx context.Context, profile Profile, mutations bool) (*gateway.Service, func(), error) {
	digest, err := workspaceIdentity(profile.Workspace, profile.TenantID, profile.ProjectID)
	if err != nil || digest != profile.WorkspaceDigest {
		return nil, nil, ErrGateway
	}
	reader, err := beads.NewCLIReader(profile.NativeBinary, profile.NativeBinarySHA256, profile.Workspace, profile.ProjectID)
	if err != nil {
		return nil, nil, ErrGateway
	}
	var mutator beads.AtomicMutator = readOnlyMutator{}
	if mutations {
		mutator, err = beads.NewNativeMutator(reader, profile.NativeBinary, profile.NativeBinarySHA256)
		if err != nil {
			return nil, nil, ErrGateway
		}
	}
	work, err := beads.New(profile.TenantID, profile.ProjectID, []authorityv1.Label{authorityv1.LabelExternalUntrusted}, reader, mutator)
	if err != nil {
		return nil, nil, ErrGateway
	}
	connection, err := readProtectedFile(profile.PostgresURLFile, 4096)
	if err != nil {
		return nil, nil, ErrGateway
	}
	connectionHash := sha256.Sum256(connection)
	if hex.EncodeToString(connectionHash[:]) != profile.ConnectionSHA256 {
		return nil, nil, ErrGateway
	}
	config, err := localPostgresConfig(connection)
	if err != nil {
		return nil, nil, ErrGateway
	}
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, nil, ErrGateway
	}
	store, err := postgres.New(pool, func(_ context.Context, tenant, project string) (string, error) {
		if tenant != profile.TenantID || project != profile.ProjectID {
			return "", ErrAuthorization
		}
		return profile.FenceGeneration, nil
	}, time.Now, nil)
	if err != nil {
		pool.Close()
		return nil, nil, ErrGateway
	}
	var service *gateway.Service
	if mutations {
		service, err = gateway.NewWithClaims(work, store, store, time.Now)
	} else {
		service, err = gateway.New(work, store, time.Now)
	}
	if err != nil {
		pool.Close()
		return nil, nil, ErrGateway
	}
	return service, pool.Close, nil
}

func localPostgresConfig(connection []byte) (*pgxpool.Config, error) {
	// pgx/libpq environment defaults and implicit credential files are not
	// authority. Require one fully scoped local URI and suppress .pgpass reads.
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(name, "PG") {
			return nil, ErrGateway
		}
	}
	parsed, err := url.Parse(strings.TrimSpace(string(connection)))
	if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") || parsed.Opaque != "" ||
		parsed.Fragment != "" || parsed.User == nil || parsed.User.Username() == "" || len(parsed.Path) <= 1 || strings.Contains(parsed.Path[1:], "/") {
		return nil, ErrGateway
	}
	query, err := url.ParseQuery(parsed.RawQuery)
	if err != nil {
		return nil, ErrGateway
	}
	for name, values := range query {
		if len(values) != 1 || (name != "host" && name != "port" && name != "sslmode") {
			return nil, ErrGateway
		}
	}
	host := parsed.Hostname()
	if query.Has("host") {
		if host != "" {
			return nil, ErrGateway
		}
		host = query.Get("host")
	}
	localSocket := filepath.IsAbs(host) && filepath.Clean(host) == host && !strings.ContainsAny(host, ",\x00")
	if host != "127.0.0.1" && host != "::1" && !localSocket {
		return nil, ErrGateway
	}
	if query.Get("sslmode") != "disable" || (parsed.Port() == "" && query.Get("port") == "") {
		return nil, ErrGateway
	}
	query.Set("passfile", os.DevNull)
	parsed.RawQuery = query.Encode()
	config, err := pgxpool.ParseConfig(parsed.String())
	if err != nil || config.ConnConfig.Host != host || config.ConnConfig.Database == "" || config.ConnConfig.User == "" || len(config.ConnConfig.Fallbacks) != 0 {
		return nil, ErrGateway
	}
	config.MinConns = 0
	config.MaxConns = 2
	config.ConnConfig.ConnectTimeout = 5 * time.Second
	return config, nil
}
