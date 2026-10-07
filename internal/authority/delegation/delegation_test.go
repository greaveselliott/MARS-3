/*
FactoryDocSync:
docs:
- docs/features/F-002-work-authority.md
- docs/design-docs/ADR-001-git-beads-authority.md
- docs/design-docs/ADR-008-standing-delivery-delegation.md
- docs/code-documentation-map.md
*/
package delegation

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func fixture() (Contract, Binding, time.Time) {
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	binding := Binding{"tenant-fixture", "project-fixture", strings.Repeat("a", 64), strings.Repeat("b", 40)}
	value := Contract{1, "MARS3StandingDeliveryContract", "delegation-fixture", "greaveselliott/MARS-3",
		binding.TenantID, binding.ProjectID, binding.PlanSHA256, binding.AcceptedBase,
		now.Add(-time.Minute), now.Add(time.Hour), []Scope{{"M3-P001", strings.Repeat("c", 64), []string{"internal/platform/example.go"}}}}
	return value, binding, now
}

func TestFiniteContractValidation(t *testing.T) {
	mutations := map[string]func(*Contract){
		"other-repository": func(c *Contract) { c.Repository = "example.invalid/other" },
		"other-tenant":     func(c *Contract) { c.TenantID = "other-tenant" },
		"other-project":    func(c *Contract) { c.ProjectID = "other-project" },
		"plan-drift":       func(c *Contract) { c.PlanSHA256 = strings.Repeat("d", 64) },
		"base-drift":       func(c *Contract) { c.AcceptedBase = strings.Repeat("e", 40) },
		"unknown-bead":     func(c *Contract) { c.Beads[0].Bead = "M3-NEW001" },
		"historical-bead":  func(c *Contract) { c.Beads[0].Bead = "M3-W001" },
		"duplicate-bead":   func(c *Contract) { c.Beads = append(c.Beads, c.Beads[0]) },
		"no-beads":         func(c *Contract) { c.Beads = nil },
		"invalid-contract": func(c *Contract) { c.Beads[0].ContractSHA256 = "unbound" },
		"no-paths":         func(c *Contract) { c.Beads[0].Paths = nil },
		"duplicate-path":   func(c *Contract) { c.Beads[0].Paths = append(c.Beads[0].Paths, c.Beads[0].Paths[0]) },
		"wildcard":         func(c *Contract) { c.Beads[0].Paths = []string{"**"} },
		"traversal":        func(c *Contract) { c.Beads[0].Paths = []string{"../outside"} },
		"git-metadata":     func(c *Contract) { c.Beads[0].Paths = []string{".git/config"} },
		"absolute-path":    func(c *Contract) { c.Beads[0].Paths = []string{"/outside"} },
		"expired":          func(c *Contract) { c.ExpiresAt = c.IssuedAt.Add(time.Second) },
		"premature":        func(c *Contract) { c.IssuedAt = c.ExpiresAt.Add(-time.Second) },
		"unbounded-window": func(c *Contract) { c.ExpiresAt = c.IssuedAt.Add(31 * 24 * time.Hour) },
	}
	accept := func([]byte, []byte) error { return nil }
	value, binding, now := fixture()
	document, _ := json.Marshal(value)
	if _, err := verify(document, []byte("synthetic-signature"), binding, now, accept); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			value, binding, now := fixture()
			mutate(&value)
			document, _ := json.Marshal(value)
			if _, err := verify(document, []byte("synthetic-signature"), binding, now, accept); !errors.Is(err, ErrDenied) {
				t.Fatal("broadened contract admitted")
			}
		})
	}
}

func TestCanonicalEncodingAndSignatureFailClosed(t *testing.T) {
	value, binding, now := fixture()
	document, _ := json.Marshal(value)
	accept := func([]byte, []byte) error { return nil }
	for _, altered := range [][]byte{append(append([]byte{}, document...), '\n'), []byte(`{"unknown":true}`),
		append([]byte(`{"schema_version":1,`), document[1:]...)} {
		if _, err := verify(altered, []byte("synthetic"), binding, now, accept); !errors.Is(err, ErrDenied) {
			t.Fatal("ambiguous encoding admitted")
		}
	}
	if _, err := VerifyContract(document, []byte("forged"), binding, now); !errors.Is(err, ErrDenied) {
		t.Fatal("forged owner signature admitted")
	}
	if _, err := verify(document, nil, binding, now, accept); !errors.Is(err, ErrDenied) {
		t.Fatal("missing signature admitted")
	}
	if _, err := verify(document, []byte("synthetic"), binding, value.ExpiresAt, accept); !errors.Is(err, ErrDenied) {
		t.Fatal("expiry boundary admitted")
	}
}
