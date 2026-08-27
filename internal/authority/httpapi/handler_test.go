/*
FactoryDocSync:
docs:
- docs/features/F-002-work-authority.md
- docs/design-docs/ADR-001-git-beads-authority.md
- docs/design-docs/ADR-003-rule-of-two.md
- docs/code-documentation-map.md
*/

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	authorityv1 "github.com/greaveselliott/MARS-3/api/authority/v1"
)

type fixtureAuth struct {
	principal authorityv1.Principal
	err       error
}

func (auth fixtureAuth) Authenticate(*http.Request) (authorityv1.Principal, error) {
	return auth.principal, auth.err
}

type fixtureGateway struct {
	principal authorityv1.Principal
	claim     authorityv1.ClaimRequest
	err       error
}

func (gateway *fixtureGateway) GetWork(_ context.Context, principal authorityv1.Principal, request authorityv1.GetWorkRequest) (authorityv1.WorkItem, error) {
	gateway.principal = principal
	return authorityv1.WorkItem{TenantID: principal.TenantID, ProjectID: principal.ProjectID, BeadID: request.BeadID}, gateway.err
}

func (gateway *fixtureGateway) Ready(_ context.Context, principal authorityv1.Principal, _ authorityv1.ReadyRequest) (authorityv1.ReadyResponse, error) {
	gateway.principal = principal
	return authorityv1.ReadyResponse{}, gateway.err
}

func (gateway *fixtureGateway) Claim(_ context.Context, principal authorityv1.Principal, request authorityv1.ClaimRequest) (authorityv1.ClaimResponse, error) {
	gateway.principal, gateway.claim = principal, request
	return authorityv1.ClaimResponse{}, gateway.err
}

func (gateway *fixtureGateway) ValidateEffect(_ context.Context, principal authorityv1.Principal, _ authorityv1.EffectValidationRequest) (authorityv1.EffectValidation, error) {
	gateway.principal = principal
	return authorityv1.EffectValidation{Allowed: true}, gateway.err
}

type fixtureJournal struct {
	tenantID, projectID string
	cursor              authorityv1.JournalCursor
}

func (journal *fixtureJournal) Replay(_ context.Context, tenantID, projectID string, cursor authorityv1.JournalCursor, _ int) (authorityv1.JournalPage, error) {
	journal.tenantID, journal.projectID, journal.cursor = tenantID, projectID, cursor
	return authorityv1.JournalPage{TenantID: tenantID, ProjectID: projectID, After: cursor}, nil
}

func TestHandlerDerivesPrincipalAndNeverAcceptsItFromJSON(t *testing.T) {
	gateway := &fixtureGateway{}
	handler := mustHandler(t, fixtureAuth{principal: principalFixture()}, gateway, &fixtureJournal{})
	body := `{"bead_id":"M3-W002","attempt_id":"attempt-001","principal":{"tenant_id":"tenant-forged"}}`
	response := serve(handler, http.MethodPost, "/v1/projects/project-fixture/claims", body)
	if response.Code != http.StatusBadRequest || gateway.principal.PrincipalID != "" {
		t.Fatalf("response=%d gateway principal=%#v", response.Code, gateway.principal)
	}

	body = `{"bead_id":"M3-W002","attempt_id":"attempt-001"}`
	response = serve(handler, http.MethodPost, "/v1/projects/project-fixture/claims", body)
	if response.Code != http.StatusOK || !reflect.DeepEqual(gateway.principal, principalFixture()) || gateway.claim.BeadID != "M3-W002" {
		t.Fatalf("response=%d principal=%#v claim=%#v", response.Code, gateway.principal, gateway.claim)
	}
}

func TestHandlerDeniesCrossProjectBeforeGateway(t *testing.T) {
	gateway := &fixtureGateway{}
	handler := mustHandler(t, fixtureAuth{principal: principalFixture()}, gateway, &fixtureJournal{})
	response := serve(handler, http.MethodGet, "/v1/projects/project-other/work/M3-W002?trace_ref=trace-001", "")
	if response.Code != http.StatusForbidden || gateway.principal.PrincipalID != "" {
		t.Fatalf("response=%d gateway principal=%#v", response.Code, gateway.principal)
	}
	assertSafeHeaders(t, response)
}

func TestHandlerJournalUsesAuthenticatedScopeAndExactCursor(t *testing.T) {
	journal := &fixtureJournal{}
	handler := mustHandler(t, fixtureAuth{principal: principalFixture()}, &fixtureGateway{}, journal)
	hash := strings.Repeat("a", 64)
	response := serve(handler, http.MethodGet, "/v1/projects/project-fixture/journal?after=7&event_hash="+hash+"&limit=25", "")
	if response.Code != http.StatusOK || journal.tenantID != "tenant-fixture" || journal.projectID != "project-fixture" || journal.cursor != (authorityv1.JournalCursor{Sequence: 7, EventHash: hash}) {
		t.Fatalf("response=%d journal=%#v", response.Code, journal)
	}
}

func TestHandlerCollapsesBackendErrorsAndRejectsMalformedInput(t *testing.T) {
	gateway := &fixtureGateway{err: errors.New("backend address and credential details")}
	handler := mustHandler(t, fixtureAuth{principal: principalFixture()}, gateway, &fixtureJournal{})
	response := serve(handler, http.MethodGet, "/v1/projects/project-fixture/work/M3-W002?trace_ref=trace-001", "")
	if response.Code != http.StatusServiceUnavailable || strings.Contains(response.Body.String(), "credential") || strings.Contains(response.Body.String(), "backend address") {
		t.Fatalf("response=%d body=%s", response.Code, response.Body.String())
	}
	response = serve(handler, http.MethodPost, "/v1/projects/project-fixture/claims", `{} {}`)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("trailing JSON response=%d", response.Code)
	}
	response = serve(handler, http.MethodGet, "/v1/projects/project-fixture/work/M3-W002?unknown=value", "")
	if response.Code != http.StatusBadRequest {
		t.Fatalf("unknown query response=%d", response.Code)
	}
}

func TestHandlerRequiresAuthenticationAndReadCapability(t *testing.T) {
	handler := mustHandler(t, fixtureAuth{err: errors.New("invalid token with private detail")}, &fixtureGateway{}, &fixtureJournal{})
	response := serve(handler, http.MethodGet, "/v1/projects/project-fixture/work/ready", "")
	if response.Code != http.StatusUnauthorized || strings.Contains(response.Body.String(), "private") {
		t.Fatalf("authentication response=%d body=%s", response.Code, response.Body.String())
	}
	principal := principalFixture()
	principal.Capabilities = nil
	handler = mustHandler(t, fixtureAuth{principal: principal}, &fixtureGateway{}, &fixtureJournal{})
	response = serve(handler, http.MethodGet, "/v1/projects/project-fixture/journal", "")
	if response.Code != http.StatusForbidden {
		t.Fatalf("journal capability response=%d", response.Code)
	}
}

func mustHandler(t *testing.T, auth Authenticator, gateway Gateway, journal Journal) *Handler {
	t.Helper()
	handler, err := New(auth, gateway, journal)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return handler
}

func principalFixture() authorityv1.Principal {
	return authorityv1.Principal{
		TenantID: "tenant-fixture", ProjectID: "project-fixture", PrincipalID: "principal-fixture", ProfileID: "work-authority-engineer",
		Capabilities: []authorityv1.Capability{authorityv1.CapabilityWorkRead, authorityv1.CapabilityWorkClaim}, Labels: []authorityv1.Label{authorityv1.LabelPublicAccepted},
	}
}

func serve(handler http.Handler, method, target, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func assertSafeHeaders(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	for key := range map[string]bool{"Cache-Control": true, "X-Content-Type-Options": true, "Content-Security-Policy": true} {
		if response.Header().Get(key) == "" {
			t.Fatalf("missing safe response header %s", key)
		}
	}
	var denial authorityv1.Denial
	if err := json.Unmarshal(response.Body.Bytes(), &denial); err != nil || denial.Rule == "" || denial.RequiredTransition == "" || denial.AllowedAction == "" {
		t.Fatalf("denial=%#v err=%v", denial, err)
	}
}
