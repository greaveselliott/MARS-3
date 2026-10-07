/*
FactoryDocSync:
docs:
- docs/features/F-002-work-authority.md
- docs/design-docs/ADR-001-git-beads-authority.md
- docs/design-docs/ADR-008-standing-delivery-delegation.md
- docs/code-documentation-map.md
*/

// Package delegation validates owner-signed finite-plan contracts. It does not
// create a principal, capability, claim, lease, or runtime execution permission.
package delegation

import (
	"bytes"
	"encoding/json"
	"errors"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/greaveselliott/MARS-3/internal/doctrine"
)

var ErrDenied = errors.New("standing delivery contract denied")

var identifier = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,127}$`)
var digest = regexp.MustCompile(`^[0-9a-f]{64}$`)
var commit = regexp.MustCompile(`^[0-9a-f]{40}$`)

// Contract is canonical JSON, without a trailing newline. Scope is finite;
// changing any field requires a new owner signature. No credential is included.
type Contract struct {
	SchemaVersion int       `json:"schema_version"`
	Kind          string    `json:"kind"`
	ID            string    `json:"id"`
	Repository    string    `json:"repository"`
	TenantID      string    `json:"tenant_id"`
	ProjectID     string    `json:"project_id"`
	PlanSHA256    string    `json:"plan_sha256"`
	AcceptedBase  string    `json:"accepted_base"`
	IssuedAt      time.Time `json:"issued_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	Beads         []Scope   `json:"beads"`
}

type Scope struct {
	Bead           string   `json:"bead"`
	ContractSHA256 string   `json:"contract_sha256"`
	Paths          []string `json:"paths"`
}

// Binding must be resolved by a trusted caller, not copied from model/request
// fields. It binds the plan and accepted base at contract activation.
type Binding struct {
	TenantID     string
	ProjectID    string
	PlanSHA256   string
	AcceptedBase string
}

var permittedBeads = map[string]bool{
	"M3-P001": true, "M3-T001": true, "M3-S001": true,
	"M3-I001": true, "M3-S002": true, "M3-A001": true,
	"M3-UI001": true, "M3-C001": true, "M3-L001": true,
	"M3-E001": true, "M3-C002": true, "M3-D001": true,
	"M3-K001": true, "M3-O001": true,
}

// VerifyContract validates a signed proposal only. Callers must independently
// enforce non-production effects, dependencies, roles, claims, live fencing,
// revocation, review gates and protected resource handles before any effect.
func VerifyContract(document, signature []byte, binding Binding, now time.Time) (Contract, error) {
	return verify(document, signature, binding, now, doctrine.VerifyStandingDeliverySignature)
}

func verify(document, signature []byte, binding Binding, now time.Time, verifier func([]byte, []byte) error) (Contract, error) {
	deny := func() (Contract, error) { return Contract{}, ErrDenied }
	if len(document) == 0 || len(document) > 65536 || len(signature) == 0 || len(signature) > 4096 || verifier == nil {
		return deny()
	}
	if verifier(document, signature) != nil {
		return deny()
	}
	var value Contract
	if json.Unmarshal(document, &value) != nil {
		return deny()
	}
	canonical, err := json.Marshal(value)
	if err != nil || !bytes.Equal(document, canonical) {
		return deny()
	}
	if value.SchemaVersion != 1 || value.Kind != "MARS3StandingDeliveryContract" ||
		!identifier.MatchString(value.ID) || value.Repository != "greaveselliott/MARS-3" ||
		!identifier.MatchString(value.TenantID) || !identifier.MatchString(value.ProjectID) ||
		!digest.MatchString(value.PlanSHA256) || !commit.MatchString(value.AcceptedBase) ||
		value.TenantID != binding.TenantID || value.ProjectID != binding.ProjectID ||
		value.PlanSHA256 != binding.PlanSHA256 || value.AcceptedBase != binding.AcceptedBase ||
		value.IssuedAt.IsZero() || !value.ExpiresAt.After(value.IssuedAt) ||
		value.ExpiresAt.Sub(value.IssuedAt) > 30*24*time.Hour ||
		now.Before(value.IssuedAt) || !now.Before(value.ExpiresAt) ||
		len(value.Beads) == 0 || len(value.Beads) > len(permittedBeads) {
		return deny()
	}
	seen := make(map[string]bool)
	for _, scope := range value.Beads {
		if !permittedBeads[scope.Bead] || seen[scope.Bead] || !digest.MatchString(scope.ContractSHA256) ||
			len(scope.Paths) == 0 || len(scope.Paths) > 256 {
			return deny()
		}
		seen[scope.Bead] = true
		paths := make(map[string]bool)
		for _, item := range scope.Paths {
			if !safePath(item) || paths[item] {
				return deny()
			}
			paths[item] = true
		}
	}
	return value, nil
}

func safePath(value string) bool {
	if value == "" || len(value) > 1024 || path.IsAbs(value) || path.Clean(value) != value ||
		value == "." || value == ".." || strings.HasPrefix(value, "../") ||
		strings.ContainsAny(value, "\\:*?[]{}\x00\r\n\t") ||
		value == ".git" || strings.HasPrefix(value, ".git/") {
		return false
	}
	for _, character := range value {
		if character < 32 || character > 126 {
			return false
		}
	}
	return true
}
