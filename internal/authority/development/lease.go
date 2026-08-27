/*
FactoryDocSync:
docs:
- docs/features/F-002-work-authority.md
- docs/design-docs/ADR-001-git-beads-authority.md
- docs/code-documentation-map.md
*/

// Package development composes the internal W-001 development-lease path. It
// accepts already-constructed trusted stores and has no credential, transport,
// listener, environment, or direct database surface.
package development

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	authorityv1 "github.com/greaveselliott/MARS-3/api/authority/v1"
	"github.com/greaveselliott/MARS-3/internal/authority/gateway"
	"github.com/greaveselliott/MARS-3/internal/doctrine"
)

var (
	ErrConfiguration     = errors.New("development lease configuration is invalid")
	ErrCanonicalPreimage = errors.New("canonical W-001 preimage does not match the signed delivery grant")
	ErrLeaseReconcile    = errors.New("development lease reconciliation failed")
	boundedToken         = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
)

// OperationalStore owns only the live claim saga, lease, and bounded journal.
// PostgreSQL implements this contract; Beads/Dolt never does.
type OperationalStore interface {
	gateway.ClaimSagaStore
	gateway.EventSink
}

// Options contains only public authority identifiers and injected stores.
// The caller owns secret acquisition and connection lifecycle outside this
// package's authority boundary.
type Options struct {
	Repo        string
	TenantID    string
	ProjectID   string
	Work        gateway.ClaimStore
	Operational OperationalStore
	Now         func() time.Time
}

// Receipt is the bounded result. It contains no datastore address, credential,
// raw Beads payload, or local filesystem path.
type Receipt struct {
	SchemaVersion   uint32                  `json:"schema_version"`
	GrantID         string                  `json:"grant_id"`
	BeadID          string                  `json:"bead_id"`
	AttemptID       string                  `json:"attempt_id"`
	LeaseID         string                  `json:"lease_id"`
	FenceGeneration string                  `json:"fence_generation"`
	LeaseEpoch      uint64                  `json:"lease_epoch"`
	ClaimVersion    authorityv1.WorkVersion `json:"claim_version"`
	BaseSHA         string                  `json:"base_sha"`
	ExpiresAt       time.Time               `json:"expires_at"`
	State           authorityv1.LeaseState  `json:"state"`
	Replayed        bool                    `json:"replayed"`
	ReceiptRef      string                  `json:"receipt_ref"`
}

// Reconcile loads the current signed delivery grant and issues or replays only
// its W-001 development lease. The gateway path never invokes canonical CAS.
func Reconcile(ctx context.Context, options Options) (Receipt, error) {
	grant, err := doctrine.LoadW001DeliveryGrant(options.Repo)
	if err != nil {
		return Receipt{}, ErrConfiguration
	}
	return reconcileWithGrant(ctx, grant, options)
}

func reconcileWithGrant(ctx context.Context, grant doctrine.W001DeliveryGrant, options Options) (Receipt, error) {
	if options.Work == nil || options.Operational == nil || options.Now == nil ||
		!boundedToken.MatchString(options.TenantID) || !boundedToken.MatchString(options.ProjectID) ||
		!grant.DevelopmentLeaseAllowed || grant.CanonicalWorkMutationAllowed ||
		!boundedToken.MatchString(grant.Principal) || !boundedToken.MatchString(grant.AttemptID) ||
		!boundedToken.MatchString(grant.IdempotencyKey) || !boundedToken.MatchString(grant.CanonicalClaimAttemptID) {
		return Receipt{}, ErrConfiguration
	}
	item, err := options.Work.Get(ctx, options.TenantID, options.ProjectID, grant.Bead)
	if err != nil || !matchesSignedPreimage(item, grant) {
		return Receipt{}, ErrCanonicalPreimage
	}
	principal := authorityv1.Principal{
		TenantID: options.TenantID, ProjectID: options.ProjectID, PrincipalID: grant.Principal, ProfileID: grant.Principal,
		Capabilities: []authorityv1.Capability{
			authorityv1.CapabilityWorkRead, authorityv1.CapabilityWorkClaim, authorityv1.CapabilityLeaseIssue,
			authorityv1.CapabilityEffectValidate, authorityv1.CapabilityTicketDelivery,
		},
		Labels: []authorityv1.Label{authorityv1.LabelPublicAccepted},
	}
	request := gateway.ClaimReconciliationRequest{
		ClaimRequest: authorityv1.ClaimRequest{
			BeadID: item.BeadID, ExpectedVersion: item.Version, ExpectedIntegrity: item.Integrity,
			AttemptID: grant.AttemptID, BaseSHA: grant.BaseCommit, ExclusivePaths: append([]string(nil), item.ExclusivePaths...),
			Capability: authorityv1.CapabilityTicketDelivery, IdempotencyKey: grant.IdempotencyKey,
			TraceRef: "trace-w001-development-lease",
		},
		CanonicalClaimAttemptID: grant.CanonicalClaimAttemptID,
	}
	service, err := gateway.NewWithClaims(options.Work, options.Operational, options.Operational, options.Now)
	if err != nil {
		return Receipt{}, ErrConfiguration
	}
	response, err := service.ReconcileClaimedWork(ctx, principal, request)
	if err != nil || !response.Lease.Active || response.Lease.AttemptID != grant.AttemptID ||
		response.Lease.ClaimVersion != item.Version || response.Work.ClaimAttemptID != grant.CanonicalClaimAttemptID {
		if err != nil {
			return Receipt{}, fmt.Errorf("%w: %w", ErrLeaseReconcile, err)
		}
		return Receipt{}, ErrLeaseReconcile
	}
	return Receipt{
		SchemaVersion: 1, GrantID: grant.ID, BeadID: response.Work.BeadID, AttemptID: response.Lease.AttemptID,
		LeaseID: response.Lease.LeaseID, FenceGeneration: response.Lease.FenceGeneration, LeaseEpoch: response.Lease.LeaseEpoch,
		ClaimVersion: response.Lease.ClaimVersion, BaseSHA: response.Lease.BaseSHA, ExpiresAt: response.Lease.ExpiresAt,
		State: response.Lease.State, Replayed: response.Replayed, ReceiptRef: response.ReceiptRef,
	}, nil
}

func matchesSignedPreimage(item authorityv1.WorkItem, grant doctrine.W001DeliveryGrant) bool {
	return item.BeadID == grant.Bead && item.NativeStatus == grant.ExpectedNativeStatus &&
		string(item.LifecycleState) == grant.ExpectedLifecycleState && item.Assignee == grant.ExpectedAssignee &&
		item.ClaimAttemptID == grant.CanonicalClaimAttemptID && item.Version.AuthorityGeneration == grant.WorkVersionGeneration &&
		item.Version.IssueIncarnation == grant.WorkVersionIncarnation && item.Version.IssueMutationSequence == grant.IssueMutationSequence &&
		item.Version.DependencyGraphRevision == grant.DependencyGraphRevision
}
