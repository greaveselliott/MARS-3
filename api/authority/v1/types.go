/*
FactoryDocSync:
docs:
- docs/features/F-002-work-authority.md
- docs/design-docs/ADR-001-git-beads-authority.md
- docs/code-documentation-map.md
*/

// Package v1 defines the provider-neutral public contracts for the work
// authority gateway. Values are bounded metadata; credentials and raw payloads
// are deliberately absent from every type in this package.
package v1

import "time"

type LifecycleState string

const (
	LifecycleBacklog    LifecycleState = "backlog"
	LifecycleInProgress LifecycleState = "in-progress"
	LifecycleInReview   LifecycleState = "in-review"
	LifecycleDone       LifecycleState = "done"
	LifecycleSuperseded LifecycleState = "superseded"
)

type Capability string

const (
	CapabilityWorkRead       Capability = "work.read"
	CapabilityWorkClaim      Capability = "work.claim"
	CapabilityWorkStatus     Capability = "work.status"
	CapabilityWorkHandoff    Capability = "work.handoff"
	CapabilityReviewRecord   Capability = "review.record"
	CapabilityRunDisposition Capability = "run.disposition"
	CapabilityLeaseIssue     Capability = "lease.issue"
	CapabilityLeaseRenew     Capability = "lease.renew"
	CapabilityLeaseRelease   Capability = "lease.release"
	CapabilityLeaseRevoke    Capability = "lease.revoke"
	CapabilityEffectValidate Capability = "effect.validate"
)

type Label string

const (
	LabelPublicAccepted    Label = "public-project-accepted"
	LabelExternalUntrusted Label = "external-untrusted"
	LabelPrivateData       Label = "private-data"
	LabelExternalEffect    Label = "external-effect"
)

// Principal is resolved by trusted authentication outside model context.
// Capabilities and labels are server-derived; caller input cannot construct
// this value at the public transport boundary.
type Principal struct {
	TenantID     string       `json:"tenant_id"`
	ProjectID    string       `json:"project_id"`
	PrincipalID  string       `json:"principal_id"`
	ProfileID    string       `json:"profile_id"`
	Capabilities []Capability `json:"capabilities"`
	Labels       []Label      `json:"labels"`
}

// WorkVersion is freshness authority. Digests can additionally prove
// integrity but never replace any monotonic component.
type WorkVersion struct {
	AuthorityGeneration     string `json:"authority_generation"`
	IssueIncarnation        string `json:"issue_incarnation"`
	IssueMutationSequence   uint64 `json:"issue_mutation_sequence"`
	DependencyGraphRevision uint64 `json:"dependency_graph_revision"`
}

// IntegrityDigests bind the projection used to make a readiness decision.
// They are integrity evidence only; WorkVersion provides freshness.
type IntegrityDigests struct {
	Lineage            string `json:"lineage"`
	DependencyOutcomes string `json:"dependency_outcomes"`
	Blockers           string `json:"blockers"`
	ExclusivePaths     string `json:"exclusive_paths"`
}

type Dependency struct {
	BeadID         string         `json:"bead_id"`
	LifecycleState LifecycleState `json:"lifecycle_state"`
	ReviewAccepted bool           `json:"review_accepted"`
	RunCompleted   bool           `json:"run_completed"`
	Reconciled     bool           `json:"reconciled"`
}

// WorkItem is a bounded projection of canonical Beads/Dolt state. It does not
// contain descriptions, comments, credentials, backend addresses, or private
// source content.
type WorkItem struct {
	TenantID           string           `json:"tenant_id"`
	ProjectID          string           `json:"project_id"`
	BeadID             string           `json:"bead_id"`
	DisplayID          string           `json:"display_id"`
	NativeStatus       string           `json:"native_status"`
	LifecycleState     LifecycleState   `json:"lifecycle_state"`
	Assignee           string           `json:"assignee"`
	GoalIDs            []string         `json:"goal_ids"`
	ProductDecisionIDs []string         `json:"product_decision_ids"`
	FeatureID          string           `json:"feature_id"`
	ScenarioIDs        []string         `json:"scenario_ids"`
	ExclusivePaths     []string         `json:"exclusive_paths"`
	VerificationOrder  []string         `json:"verification_order"`
	Blockers           []string         `json:"blockers"`
	Dependencies       []Dependency     `json:"dependencies"`
	Labels             []Label          `json:"labels"`
	Version            WorkVersion      `json:"version"`
	Integrity          IntegrityDigests `json:"integrity"`
}

type ReadyRequest struct {
	TraceRef       string  `json:"trace_ref"`
	ProposedLabels []Label `json:"proposed_labels,omitempty"`
}

type ReadyItem struct {
	WorkItem
	Ready  bool     `json:"ready"`
	Rules  []string `json:"rules"`
	Action string   `json:"action"`
}

type ReadyResponse struct {
	Items []ReadyItem `json:"items"`
}

type GetWorkRequest struct {
	BeadID         string  `json:"bead_id"`
	TraceRef       string  `json:"trace_ref"`
	ProposedLabels []Label `json:"proposed_labels,omitempty"`
}

type ErrorCode string

const (
	ErrorInvalidRequest ErrorCode = "invalid_request"
	ErrorUnauthorized   ErrorCode = "unauthorized"
	ErrorTenantMismatch ErrorCode = "tenant_mismatch"
	ErrorNotFound       ErrorCode = "not_found"
	ErrorNotReady       ErrorCode = "not_ready"
	ErrorStaleVersion   ErrorCode = "stale_version"
	ErrorPolicyDenied   ErrorCode = "policy_denied"
	ErrorAuthorityDown  ErrorCode = "authority_unavailable"
	ErrorUnknownEffect  ErrorCode = "unknown_effect"
)

// Denial is the only public error envelope for governed operations. Detail is
// a stable bounded rule identifier, never reflected input or backend output.
type Denial struct {
	Code               ErrorCode      `json:"code"`
	CurrentState       LifecycleState `json:"current_state,omitempty"`
	Rule               string         `json:"rule"`
	RequiredTransition string         `json:"required_transition"`
	AllowedAction      string         `json:"allowed_action"`
	TraceRef           string         `json:"trace_ref"`
}

func (d *Denial) Error() string { return string(d.Code) + ": " + d.Rule }

type Event struct {
	SchemaVersion uint32    `json:"schema_version"`
	Sequence      uint64    `json:"sequence"`
	TraceRef      string    `json:"trace_ref"`
	TenantID      string    `json:"tenant_id"`
	ProjectID     string    `json:"project_id"`
	BeadID        string    `json:"bead_id,omitempty"`
	PrincipalID   string    `json:"principal_id"`
	Operation     string    `json:"operation"`
	Outcome       string    `json:"outcome"`
	Rule          string    `json:"rule,omitempty"`
	Labels        []Label   `json:"labels"`
	OccurredAt    time.Time `json:"occurred_at"`
}
