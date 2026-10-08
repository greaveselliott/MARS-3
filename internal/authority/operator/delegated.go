/*
FactoryDocSync:
docs:
- docs/features/F-002-work-authority.md
- docs/design-docs/ADR-001-git-beads-authority.md
- docs/design-docs/ADR-008-standing-delivery-delegation.md
- docs/code-documentation-map.md
*/

package operator

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"io"
	"path"
	"regexp"
	"slices"
	"strings"
	"time"

	authorityv1 "github.com/greaveselliott/MARS-3/api/authority/v1"
	"github.com/greaveselliott/MARS-3/internal/authority/gateway"
	"github.com/greaveselliott/MARS-3/internal/doctrine"
)

var standingIdentifierPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
var standingContentPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
var standingCommitPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

func decodeStandingCanonical(document []byte, target any) error {
	if json.Unmarshal(document, target) != nil {
		return ErrAuthorization
	}
	encoded, err := json.Marshal(target)
	if err != nil || !bytes.Equal(document, encoded) {
		return ErrAuthorization
	}
	return nil
}

// StandingScope is an exact accepted contract projection. A trailing slash is
// the gateway's explicit directory scope; paths are never wildcard patterns.
type StandingScope struct {
	Bead           string   `json:"bead"`
	FeatureID      string   `json:"feature_id"`
	ContractSHA256 string   `json:"contract_sha256"`
	Paths          []string `json:"paths"`
}

type StandingRole struct {
	Bead        string   `json:"bead"`
	PrincipalID string   `json:"principal_id"`
	ProfileID   string   `json:"profile_id"`
	Class       string   `json:"class"`
	Operations  []string `json:"operations"`
}

// StandingRuntime is operational v2, never a reinterpretation of v1 data.
type StandingRuntime struct {
	SchemaVersion int             `json:"schema_version"`
	Kind          string          `json:"kind"`
	ID            string          `json:"id"`
	Repository    string          `json:"repository"`
	Runtime       RuntimeBinding  `json:"runtime"`
	PlanSHA256    string          `json:"plan_sha256"`
	IssuedAt      time.Time       `json:"issued_at"`
	ExpiresAt     time.Time       `json:"expires_at"`
	Scopes        []StandingScope `json:"scopes"`
	Roles         []StandingRole  `json:"roles"`
}

// A trusted operator installs and reloads this separately signed record. A
// missing/revoked record denies; the model cannot supply activation bindings.
type StandingActivation struct {
	SchemaVersion    int             `json:"schema_version"`
	Kind             string          `json:"kind"`
	DelegationSHA256 string          `json:"delegation_sha256"`
	Runtime          RuntimeBinding  `json:"runtime"`
	PlanSHA256       string          `json:"plan_sha256"`
	SourceBaseSHA    string          `json:"source_base_sha"`
	IssuedAt         time.Time       `json:"issued_at"`
	ExpiresAt        time.Time       `json:"expires_at"`
	Revoked          bool            `json:"revoked"`
	Contracts        []StandingScope `json:"contracts"`
}

// Sessions authenticate a role, not a request-supplied claim of identity.
// Their protected delivery to independent workers is a trusted operator duty.
type StandingSession struct {
	SchemaVersion    int       `json:"schema_version"`
	Kind             string    `json:"kind"`
	ID               string    `json:"id"`
	DelegationSHA256 string    `json:"delegation_sha256"`
	Bead             string    `json:"bead"`
	PrincipalID      string    `json:"principal_id"`
	ProfileID        string    `json:"profile_id"`
	Class            string    `json:"class"`
	IssuedAt         time.Time `json:"issued_at"`
	ExpiresAt        time.Time `json:"expires_at"`
}

// Exactly one payload is allowed. Individual requests need no owner signature:
// the factory derives their admission from the parent and authenticated role.
type StandingRequest struct {
	ID        string                                 `json:"id"`
	Operation string                                 `json:"operation"`
	IssuedAt  time.Time                              `json:"issued_at"`
	ExpiresAt time.Time                              `json:"expires_at"`
	Read      *ReadRequest                           `json:"read,omitempty"`
	Mutation  *MutationRequest                       `json:"mutation,omitempty"`
	Reclaim   *gateway.ClaimReconciliationRequest    `json:"reclaim,omitempty"`
	Handoff   *authorityv1.HandoffRequest            `json:"handoff,omitempty"`
	Review    *authorityv1.ReviewVerdictRequest      `json:"review,omitempty"`
	Run       *authorityv1.RunDispositionRequest     `json:"run,omitempty"`
	Reconcile *authorityv1.ReconciliationRequest     `json:"reconcile,omitempty"`
	Close     *authorityv1.TerminalTransitionRequest `json:"close,omitempty"`
}

type StandingActivationSource interface {
	Current(context.Context) (StandingActivation, error)
}

type StandingGateway interface {
	ReadGateway
	DispatchStanding(context.Context, authorityv1.Principal, StandingRequest) (any, error)
}

type StandingGate struct {
	runtime       RuntimeBinding
	activation    StandingActivationSource
	gateway       StandingGateway
	replays       ReplayStore
	now           func() time.Time
	verifyRuntime func([]byte, []byte) error
	verifySession func([]byte, []byte) error
}

func NewStandingGate(runtime RuntimeBinding, activation StandingActivationSource, backend StandingGateway, replays ReplayStore) (*StandingGate, error) {
	if !validRuntime(runtime) || activation == nil || backend == nil || replays == nil {
		return nil, ErrAuthorization
	}
	return &StandingGate{runtime: runtime, activation: activation, gateway: backend, replays: replays,
		now: time.Now, verifyRuntime: doctrine.VerifyStandingRuntimeSignature, verifySession: doctrine.VerifyStandingSessionSignature}, nil
}

// Execute is the factory issuer and dispatch boundary. Admission is local to
// this exact decoded request, never returned as a reusable bearer capability.
func (gate *StandingGate) Execute(ctx context.Context, document, signature, sessionDocument, sessionSignature, requestDocument []byte) (any, error) {
	if ctx.Err() != nil || len(document) == 0 || len(document) > 65536 || len(signature) == 0 || len(signature) > 4096 ||
		len(sessionDocument) == 0 || len(sessionDocument) > 16384 || len(sessionSignature) == 0 || len(sessionSignature) > 4096 ||
		len(requestDocument) == 0 || len(requestDocument) > 65536 {
		return nil, ErrAuthorization
	}
	document, signature = slices.Clone(document), slices.Clone(signature)
	sessionDocument, sessionSignature = slices.Clone(sessionDocument), slices.Clone(sessionSignature)
	requestDocument = slices.Clone(requestDocument)
	if gate.verifyRuntime(document, signature) != nil || gate.verifySession(sessionDocument, sessionSignature) != nil {
		return nil, ErrAuthorization
	}
	var parent StandingRuntime
	var session StandingSession
	var request StandingRequest
	if decodeStandingCanonical(document, &parent) != nil || decodeStandingCanonical(sessionDocument, &session) != nil || decodeStandingCanonical(requestDocument, &request) != nil {
		return nil, ErrAuthorization
	}
	parentHash := standingDigest(document)
	now := gate.now()
	if !validStandingParent(parent, gate.runtime, now) || session.SchemaVersion != 2 || session.Kind != "MARS3StandingDeliverySession" ||
		!standingIdentifierPattern.MatchString(session.ID) || session.DelegationSHA256 != parentHash ||
		!standingWindow(session.IssuedAt, session.ExpiresAt, now, time.Hour) ||
		session.IssuedAt.Before(parent.IssuedAt) || session.ExpiresAt.After(parent.ExpiresAt) ||
		!standingIdentifierPattern.MatchString(request.ID) || !standingWindow(request.IssuedAt, request.ExpiresAt, now, 5*time.Minute) ||
		request.IssuedAt.Before(session.IssuedAt) || request.ExpiresAt.After(session.ExpiresAt) {
		return nil, ErrAuthorization
	}
	bead, trace, capability, err := request.target()
	if err != nil || !standingIdentifierPattern.MatchString(trace) {
		return nil, ErrAuthorization
	}
	if request.Operation == "work.ready" {
		bead = session.Bead
	}
	if bead != session.Bead || !standingRoleAllows(parent, session, request.Operation) {
		return nil, ErrAuthorization
	}
	scope, ok := standingScope(parent.Scopes, bead)
	if !ok {
		return nil, ErrAuthorization
	}
	principal := authorityv1.Principal{TenantID: gate.runtime.TenantID, ProjectID: gate.runtime.ProjectID,
		PrincipalID: session.PrincipalID, ProfileID: session.ProfileID,
		Capabilities: []authorityv1.Capability{authorityv1.CapabilityWorkRead},
		Labels:       []authorityv1.Label{authorityv1.LabelExternalUntrusted, authorityv1.LabelExternalEffect}}
	activation, err := gate.currentActivation(ctx, parent, parentHash, scope)
	if err != nil {
		return nil, ErrAuthorization
	}
	// The accepted operator release is not the source base of every future
	// Bead. The trusted activation pins the current protected source base.
	deadline := request.ExpiresAt
	if activation.ExpiresAt.Before(deadline) {
		deadline = activation.ExpiresAt
	}
	remaining := deadline.Sub(gate.now())
	if remaining <= 0 {
		return nil, ErrAuthorization
	}
	bounded, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	// Reserve the request identity across all roles/operations, not its digest:
	// changing a consumed request's payload must not create another replay key.
	key := standingDigest([]byte(gate.runtime.TenantID + "\x00" + gate.runtime.ProjectID + "\x00" + parentHash + "\x00" + request.ID))
	if gate.replays.Consume(bounded, key) != nil {
		return nil, ErrReplay
	}
	if !standingWindow(parent.IssuedAt, parent.ExpiresAt, gate.now(), 30*24*time.Hour) ||
		!standingWindow(session.IssuedAt, session.ExpiresAt, gate.now(), time.Hour) ||
		!standingWindow(request.IssuedAt, request.ExpiresAt, gate.now(), 5*time.Minute) || bounded.Err() != nil {
		return nil, ErrAuthorization
	}
	if _, err := gate.currentActivation(bounded, parent, parentHash, scope); err != nil {
		return nil, ErrAuthorization
	}
	if request.Operation == "work.ready" {
		ready, err := gate.gateway.Ready(bounded, principal, authorityv1.ReadyRequest{TraceRef: trace})
		if err != nil {
			return nil, ErrGateway
		}
		filtered := authorityv1.ReadyResponse{Items: make([]authorityv1.ReadyItem, 0)}
		for _, item := range ready.Items {
			if item.BeadID == bead && standingWorkMatches(item.WorkItem, scope, gate.runtime) {
				filtered.Items = append(filtered.Items, item)
			}
		}
		return filtered, nil
	}
	work, err := gate.gateway.GetWork(bounded, principal, authorityv1.GetWorkRequest{BeadID: bead, TraceRef: trace})
	if err != nil {
		return nil, ErrGateway
	}
	if !standingWorkMatches(work, scope, gate.runtime) {
		return nil, ErrAuthorization
	}
	if request.Operation == "work.get" {
		return work, nil
	}
	// Restore only this authenticated owner's existing canonical claim lease.
	// Never bootstrap backlog work, transfer ownership or change provenance.
	if request.Reclaim != nil && (work.LifecycleState != authorityv1.LifecycleInProgress || work.Assignee != session.ProfileID ||
		!standingIdentifierPattern.MatchString(work.ClaimAttemptID) || request.Reclaim.CanonicalClaimAttemptID != work.ClaimAttemptID) {
		return nil, ErrAuthorization
	}
	// A fresh read is not a claim or lease. The gateway validates the payload's
	// expected version, dependencies, role and full live fence at the mutation.
	if bounded.Err() != nil || !standingWindow(request.IssuedAt, request.ExpiresAt, gate.now(), 5*time.Minute) {
		return nil, ErrAuthorization
	}
	activation, err = gate.currentActivation(bounded, parent, parentHash, scope)
	if err != nil || !request.scopeMatches(scope, gate.runtime, activation.SourceBaseSHA) {
		return nil, ErrAuthorization
	}
	// Never extend an already-issued deadline when activation is replaced.
	dispatchCtx, dispatchCancel := context.WithDeadline(bounded, activation.ExpiresAt)
	defer dispatchCancel()
	if dispatchCtx.Err() != nil {
		return nil, ErrAuthorization
	}
	principal.Capabilities = standingActionCapabilities(request.Operation, capability)
	result, err := gate.gateway.DispatchStanding(dispatchCtx, principal, request)
	if err != nil {
		return nil, ErrGateway
	}
	return result, nil
}

func (gate *StandingGate) currentActivation(ctx context.Context, parent StandingRuntime, parentHash string, scope StandingScope) (StandingActivation, error) {
	activation, err := gate.activation.Current(ctx)
	if err != nil || ctx.Err() != nil || activation.SchemaVersion != 2 || activation.Kind != "MARS3StandingDeliveryActivation" ||
		activation.Revoked || activation.DelegationSHA256 != parentHash || activation.Runtime != gate.runtime ||
		!standingCommitPattern.MatchString(activation.SourceBaseSHA) ||
		activation.PlanSHA256 != parent.PlanSHA256 || !standingWindow(activation.IssuedAt, activation.ExpiresAt, gate.now(), 7*24*time.Hour) ||
		activation.IssuedAt.Before(parent.IssuedAt) || activation.ExpiresAt.After(parent.ExpiresAt) ||
		!validStandingScopes(activation.Contracts) {
		return StandingActivation{}, ErrAuthorization
	}
	current, ok := standingScope(activation.Contracts, scope.Bead)
	if !ok || current.FeatureID != scope.FeatureID || current.ContractSHA256 != scope.ContractSHA256 || !standingPathsEqual(current.Paths, scope.Paths) {
		return StandingActivation{}, ErrAuthorization
	}
	return activation, nil
}

func validStandingParent(parent StandingRuntime, runtime RuntimeBinding, now time.Time) bool {
	if parent.SchemaVersion != 2 || parent.Kind != "MARS3StandingDeliveryRuntime" || parent.Repository != "greaveselliott/MARS-3" ||
		!standingIdentifierPattern.MatchString(parent.ID) || parent.Runtime != runtime || !standingContentPattern.MatchString(parent.PlanSHA256) ||
		!standingWindow(parent.IssuedAt, parent.ExpiresAt, now, 30*24*time.Hour) || !validStandingScopes(parent.Scopes) ||
		len(parent.Roles) == 0 || len(parent.Roles) > 128 {
		return false
	}
	identities, assignments := map[string]string{}, map[string]bool{}
	for _, role := range parent.Roles {
		if !standingCanonicalRoleMatches(role) {
			return false
		}
		_, found := standingScope(parent.Scopes, role.Bead)
		if !found || !standingIdentifierPattern.MatchString(role.PrincipalID) || !standingIdentifierPattern.MatchString(role.ProfileID) || len(role.Operations) == 0 || len(role.Operations) > 12 {
			return false
		}
		// A single principal or profile may not impersonate another role class.
		for _, identity := range []string{"principal:" + role.PrincipalID, "profile:" + role.ProfileID} {
			if previous, ok := identities[identity]; ok && previous != role.Class {
				return false
			}
			identities[identity] = role.Class
		}
		key := role.Bead + "\x00" + role.PrincipalID + "\x00" + role.ProfileID
		if assignments[key] {
			return false
		}
		assignments[key] = true
		seen := map[string]bool{}
		for _, operation := range role.Operations {
			if seen[operation] || !standingClassAllows(role.Class, operation) {
				return false
			}
			seen[operation] = true
		}
	}
	return true
}

func standingClassAllows(class, operation string) bool {
	if operation == "work.get" {
		return class == "implementation" || class == "qa" || class == "security" || class == "orchestrator"
	}
	switch class {
	case "implementation":
		return slices.Contains([]string{"work.claim", "work.reclaim", "lease.renew", "lease.release", "effect.validate", "work.handoff"}, operation)
	case "qa", "security":
		return operation == "review.record"
	case "orchestrator":
		return slices.Contains([]string{"work.ready", "run.disposition", "work.reconcile", "work.close"}, operation)
	}
	return false
}

// Canonical gateway reviewers are profiles, not caller-declared class names.
// Validate that binding before using Class to separate principal identities.
func standingCanonicalRoleMatches(role StandingRole) bool {
	switch role.ProfileID {
	case "qa":
		return role.Class == "qa"
	case "security-reviewer":
		return role.Class == "security"
	case "delivery-orchestrator":
		return role.Class == "orchestrator"
	default:
		return role.Class == "implementation"
	}
}

func standingRoleAllows(parent StandingRuntime, session StandingSession, operation string) bool {
	for _, role := range parent.Roles {
		if role.Bead == session.Bead && role.PrincipalID == session.PrincipalID && role.ProfileID == session.ProfileID &&
			role.Class == session.Class && slices.Contains(role.Operations, operation) {
			return true
		}
	}
	return false
}

func validStandingScopes(scopes []StandingScope) bool {
	if len(scopes) == 0 || len(scopes) > 14 {
		return false
	}
	seen := map[string]bool{}
	for _, scope := range scopes {
		if !slices.Contains([]string{"M3-P001", "M3-T001", "M3-S001", "M3-I001", "M3-S002", "M3-A001", "M3-UI001", "M3-C001", "M3-L001", "M3-E001", "M3-C002", "M3-D001", "M3-K001", "M3-O001"}, scope.Bead) ||
			seen[scope.Bead] || !standingIdentifierPattern.MatchString(scope.FeatureID) || !standingContentPattern.MatchString(scope.ContractSHA256) || len(scope.Paths) == 0 || len(scope.Paths) > 256 {
			return false
		}
		seen[scope.Bead] = true
		paths := map[string]bool{}
		for _, item := range scope.Paths {
			if !standingSafePath(item, true) || paths[item] {
				return false
			}
			paths[item] = true
		}
	}
	return true
}

func standingSafePath(value string, directory bool) bool {
	base := value
	if directory {
		base = strings.TrimSuffix(base, "/")
	}
	if base == "" || len(value) > 1024 || path.IsAbs(base) || path.Clean(base) != base || base == "." || base == ".." ||
		strings.HasPrefix(base, "../") || strings.ContainsAny(base, "\\:*?[]{}\x00\r\n\t") {
		return false
	}
	for _, component := range strings.Split(base, "/") {
		if component == ".git" {
			return false
		}
	}
	for _, character := range base {
		if character < 32 || character > 126 {
			return false
		}
	}
	return true
}

func standingPathWithin(scopes []string, target string) bool {
	if !standingSafePath(target, false) {
		return false
	}
	for _, scope := range scopes {
		if target == scope || strings.HasSuffix(scope, "/") && strings.HasPrefix(target, scope) {
			return true
		}
	}
	return false
}

func standingScope(scopes []StandingScope, bead string) (StandingScope, bool) {
	for _, scope := range scopes {
		if scope.Bead == bead {
			return scope, true
		}
	}
	return StandingScope{}, false
}

func standingPathsEqual(left, right []string) bool {
	left, right = slices.Clone(left), slices.Clone(right)
	slices.Sort(left)
	slices.Sort(right)
	return slices.Equal(left, right)
}

func standingWorkMatches(work authorityv1.WorkItem, scope StandingScope, runtime RuntimeBinding) bool {
	return work.TenantID == runtime.TenantID && work.ProjectID == runtime.ProjectID && work.BeadID == scope.Bead &&
		work.FeatureID == scope.FeatureID && standingPathsEqual(work.ExclusivePaths, scope.Paths)
}

func standingWindow(issued, expires, now time.Time, maximum time.Duration) bool {
	return !issued.IsZero() && expires.After(issued) && expires.Sub(issued) <= maximum && !now.Before(issued) && now.Before(expires)
}

func standingDigest(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

// Handoff is one gateway operation that releases this request's exact live
// lease before entering review. Its companion capability never leaves this
// action's dispatch and cannot authorize a different lease or operation.
func standingActionCapabilities(operation string, capability authorityv1.Capability) []authorityv1.Capability {
	if operation == "work.reclaim" {
		return []authorityv1.Capability{authorityv1.CapabilityWorkClaim, authorityv1.CapabilityLeaseIssue}
	}
	if operation == "work.handoff" {
		return []authorityv1.Capability{authorityv1.CapabilityWorkHandoff, authorityv1.CapabilityLeaseRelease}
	}
	return []authorityv1.Capability{capability}
}

func (request StandingRequest) target() (string, string, authorityv1.Capability, error) {
	count := 0
	for _, present := range []bool{request.Read != nil, request.Mutation != nil, request.Reclaim != nil, request.Handoff != nil, request.Review != nil, request.Run != nil, request.Reconcile != nil, request.Close != nil} {
		if present {
			count++
		}
	}
	if count != 1 {
		return "", "", "", ErrAuthorization
	}
	if read := request.Read; read != nil {
		if read.Operation != request.Operation {
			return "", "", "", ErrAuthorization
		}
		if request.Operation == "work.get" && standingIdentifierPattern.MatchString(read.BeadID) {
			return read.BeadID, read.TraceRef, authorityv1.CapabilityWorkRead, nil
		}
		if request.Operation == "work.ready" && read.BeadID == "" {
			return "", read.TraceRef, authorityv1.CapabilityWorkRead, nil
		}
	}
	if mutation := request.Mutation; mutation != nil && mutation.Operation == request.Operation {
		capability, valid := mutation.capability()
		if !valid {
			return "", "", "", ErrAuthorization
		}
		switch {
		case mutation.Claim != nil:
			return mutation.Claim.BeadID, mutation.Claim.TraceRef, capability, nil
		case mutation.Renew != nil:
			return mutation.Renew.Fence.BeadID, mutation.Renew.TraceRef, capability, nil
		case mutation.Release != nil:
			return mutation.Release.Fence.BeadID, mutation.Release.TraceRef, capability, nil
		case mutation.Effect != nil:
			return mutation.Effect.Fence.BeadID, mutation.Effect.TraceRef, capability, nil
		}
	}
	switch {
	case request.Operation == "work.reclaim" && request.Reclaim != nil:
		return request.Reclaim.BeadID, request.Reclaim.TraceRef, authorityv1.CapabilityWorkClaim, nil
	case request.Operation == "work.handoff" && request.Handoff != nil:
		return request.Handoff.BeadID, request.Handoff.TraceRef, authorityv1.CapabilityWorkHandoff, nil
	case request.Operation == "review.record" && request.Review != nil:
		return request.Review.BeadID, request.Review.TraceRef, authorityv1.CapabilityReviewRecord, nil
	case request.Operation == "run.disposition" && request.Run != nil:
		return request.Run.BeadID, request.Run.TraceRef, authorityv1.CapabilityRunDisposition, nil
	case request.Operation == "work.reconcile" && request.Reconcile != nil:
		return request.Reconcile.BeadID, request.Reconcile.TraceRef, authorityv1.CapabilityWorkReconcile, nil
	case request.Operation == "work.close" && request.Close != nil:
		return request.Close.BeadID, request.Close.TraceRef, authorityv1.CapabilityWorkClose, nil
	}
	return "", "", "", ErrAuthorization
}

func (request StandingRequest) scopeMatches(scope StandingScope, runtime RuntimeBinding, sourceBase string) bool {
	if claim := request.Reclaim; claim != nil {
		return claim.BaseSHA == sourceBase && standingPathsEqual(claim.ExclusivePaths, scope.Paths) && claim.Capability == authorityv1.CapabilityTicketDelivery
	}
	var fence *authorityv1.FencingTuple
	if mutation := request.Mutation; mutation != nil {
		if claim := mutation.Claim; claim != nil {
			return claim.BaseSHA == sourceBase && standingPathsEqual(claim.ExclusivePaths, scope.Paths) && claim.Capability == authorityv1.CapabilityTicketDelivery
		}
		if mutation.Renew != nil {
			fence = &mutation.Renew.Fence
		}
		if mutation.Release != nil {
			fence = &mutation.Release.Fence
		}
		if mutation.Effect != nil {
			fence = &mutation.Effect.Fence
			if !standingPathWithin(scope.Paths, mutation.Effect.Path) {
				return false
			}
		}
	}
	if request.Handoff != nil {
		fence = &request.Handoff.Fence
	}
	return fence == nil || (fence.TenantID == runtime.TenantID && fence.ProjectID == runtime.ProjectID && fence.BeadID == scope.Bead &&
		fence.BaseSHA == sourceBase && fence.Capability == authorityv1.CapabilityTicketDelivery && standingPathsEqual(fence.ExclusivePaths, scope.Paths))
}

type protectedStandingActivation struct{ file string }

func (source protectedStandingActivation) Current(ctx context.Context) (StandingActivation, error) {
	var activation StandingActivation
	if ctx.Err() != nil {
		return activation, ErrAuthorization
	}
	document, err := readProtectedFile(source.file, 65536)
	if err != nil {
		return activation, ErrAuthorization
	}
	signature, err := readProtectedFile(source.file+".sig", 4096)
	if err != nil || doctrine.VerifyStandingActivationSignature(document, signature) != nil || decodeStandingCanonical(document, &activation) != nil {
		return StandingActivation{}, ErrAuthorization
	}
	return activation, nil
}

func (backend *lazyReadGateway) DispatchStanding(ctx context.Context, principal authorityv1.Principal, request StandingRequest) (any, error) {
	if !backend.mutations {
		return nil, ErrGateway
	}
	_, _, capability, err := request.target()
	if err != nil || !slices.Equal(principal.Capabilities, standingActionCapabilities(request.Operation, capability)) {
		return nil, ErrGateway
	}
	if request.Mutation != nil {
		return backend.Mutate(ctx, principal, *request.Mutation)
	}
	service, err := backend.get(ctx)
	if err != nil {
		return nil, ErrGateway
	}
	switch request.Operation {
	case "work.reclaim":
		return service.ReconcileClaimedWork(ctx, principal, *request.Reclaim)
	case "work.handoff":
		return service.Handoff(ctx, principal, *request.Handoff)
	case "review.record":
		return service.RecordReviewVerdict(ctx, principal, *request.Review)
	case "run.disposition":
		return service.RecordRunDisposition(ctx, principal, *request.Run)
	case "work.reconcile":
		return service.RecordReconciliation(ctx, principal, *request.Reconcile)
	case "work.close":
		return service.CloseWork(ctx, principal, *request.Close)
	}
	return nil, ErrGateway
}

// RunDelegatedCLI consumes only preinstalled protected handles. It does not
// discover credentials, install activation, sign sessions or provision state.
func RunDelegatedCLI(args []string, output io.Writer) error {
	flags := flag.NewFlagSet("operator-delegated", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	profileFile := flags.String("profile", "", "protected signed runtime profile")
	activationFile := flags.String("activation", "", "protected signed activation")
	delegationFile := flags.String("delegation", "", "protected signed operational delegation")
	sessionFile := flags.String("session", "", "protected signed role session")
	requestFile := flags.String("request", "", "protected typed request")
	if flags.Parse(args) != nil || flags.NArg() != 0 || output == nil || *profileFile == "" || *activationFile == "" || *delegationFile == "" || *sessionFile == "" || *requestFile == "" {
		return ErrAuthorization
	}
	profile, runtime, err := loadProfile(*profileFile, time.Now())
	if err != nil {
		return ErrAuthorization
	}
	document, err := readProtectedFile(*delegationFile, 65536)
	if err != nil {
		return ErrAuthorization
	}
	signature, err := readProtectedFile(*delegationFile+".sig", 4096)
	if err != nil {
		return ErrAuthorization
	}
	session, err := readProtectedFile(*sessionFile, 16384)
	if err != nil {
		return ErrAuthorization
	}
	sessionSignature, err := readProtectedFile(*sessionFile+".sig", 4096)
	if err != nil {
		return ErrAuthorization
	}
	request, err := readProtectedFile(*requestFile, 65536)
	if err != nil {
		return ErrAuthorization
	}
	replays, err := NewFileReplayStore(profile.ReplayDirectory)
	if err != nil {
		return ErrAuthorization
	}
	defer replays.Close()
	backend := &lazyReadGateway{profile: profile, mutations: true}
	defer backend.close()
	gate, err := NewStandingGate(runtime, protectedStandingActivation{file: *activationFile}, backend, replays)
	if err != nil {
		return ErrAuthorization
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	profileCtx, profileCancel := context.WithDeadline(ctx, profile.ExpiresAt)
	defer profileCancel()
	result, err := gate.Execute(profileCtx, document, signature, session, sessionSignature, request)
	if err != nil {
		return err
	}
	if json.NewEncoder(output).Encode(result) != nil {
		return ErrGateway
	}
	return nil
}
