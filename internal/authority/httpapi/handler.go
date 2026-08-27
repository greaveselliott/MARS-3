/*
FactoryDocSync:
docs:
- docs/features/F-002-work-authority.md
- docs/design-docs/ADR-001-git-beads-authority.md
- docs/design-docs/ADR-003-rule-of-two.md
- docs/code-documentation-map.md
*/

// Package httpapi exposes only typed, tenant-bound authority operations. The
// transport never accepts a Principal from request JSON; authentication owns
// that object outside model and sandbox context.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	authorityv1 "github.com/greaveselliott/MARS-3/api/authority/v1"
)

const maximumRequestBytes = 64 << 10

type Authenticator interface {
	Authenticate(*http.Request) (authorityv1.Principal, error)
}

type Gateway interface {
	GetWork(context.Context, authorityv1.Principal, authorityv1.GetWorkRequest) (authorityv1.WorkItem, error)
	Ready(context.Context, authorityv1.Principal, authorityv1.ReadyRequest) (authorityv1.ReadyResponse, error)
	Claim(context.Context, authorityv1.Principal, authorityv1.ClaimRequest) (authorityv1.ClaimResponse, error)
	ValidateEffect(context.Context, authorityv1.Principal, authorityv1.EffectValidationRequest) (authorityv1.EffectValidation, error)
}

type Journal interface {
	Replay(context.Context, string, string, authorityv1.JournalCursor, int) (authorityv1.JournalPage, error)
}

type Handler struct {
	auth    Authenticator
	gateway Gateway
	journal Journal
}

func New(auth Authenticator, gateway Gateway, journal Journal) (*Handler, error) {
	if auth == nil || gateway == nil || journal == nil {
		return nil, errors.New("authority HTTP dependencies are required")
	}
	return &Handler{auth: auth, gateway: gateway, journal: journal}, nil
}

func (handler *Handler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	setSafeHeaders(response)
	principal, err := handler.auth.Authenticate(request)
	if err != nil || principal.TenantID == "" || principal.ProjectID == "" || principal.PrincipalID == "" {
		writeDenial(response, http.StatusUnauthorized, authorityv1.Denial{Code: authorityv1.ErrorUnauthorized, Rule: "authority.transport.authentication", RequiredTransition: "authenticate a tenant-scoped principal", AllowedAction: "session.authenticate", TraceRef: "trace-invalid"})
		return
	}
	projectID, route, ok := parseProjectRoute(request.URL.Path)
	if !ok {
		writeDenial(response, http.StatusNotFound, transportDenial(authorityv1.ErrorNotFound, "authority.transport.route", "select a typed authority route", "work.read"))
		return
	}
	if projectID != principal.ProjectID {
		writeDenial(response, http.StatusForbidden, transportDenial(authorityv1.ErrorTenantMismatch, "authority.transport.project", "select the authenticated project", "project.select"))
		return
	}

	switch {
	case request.Method == http.MethodGet && route == "work/ready":
		handler.ready(response, request, principal)
	case request.Method == http.MethodGet && strings.HasPrefix(route, "work/"):
		handler.getWork(response, request, principal, strings.TrimPrefix(route, "work/"))
	case request.Method == http.MethodPost && route == "claims":
		handler.claim(response, request, principal)
	case request.Method == http.MethodPost && route == "effects/validate":
		handler.validateEffect(response, request, principal)
	case request.Method == http.MethodGet && route == "journal":
		handler.replay(response, request, principal)
	default:
		writeDenial(response, http.StatusMethodNotAllowed, transportDenial(authorityv1.ErrorInvalidRequest, "authority.transport.method", "use the typed route method", "work.read"))
	}
}

func (handler *Handler) ready(response http.ResponseWriter, request *http.Request, principal authorityv1.Principal) {
	if !onlyQueryKeys(request, "trace_ref") {
		writeInvalid(response)
		return
	}
	result, err := handler.gateway.Ready(request.Context(), principal, authorityv1.ReadyRequest{TraceRef: request.URL.Query().Get("trace_ref")})
	writeResult(response, result, err)
}

func (handler *Handler) getWork(response http.ResponseWriter, request *http.Request, principal authorityv1.Principal, beadID string) {
	if beadID == "" || strings.Contains(beadID, "/") || !onlyQueryKeys(request, "trace_ref") {
		writeInvalid(response)
		return
	}
	result, err := handler.gateway.GetWork(request.Context(), principal, authorityv1.GetWorkRequest{BeadID: beadID, TraceRef: request.URL.Query().Get("trace_ref")})
	writeResult(response, result, err)
}

func (handler *Handler) claim(response http.ResponseWriter, request *http.Request, principal authorityv1.Principal) {
	var input authorityv1.ClaimRequest
	if !onlyQueryKeys(request) {
		writeInvalid(response)
		return
	}
	if decodeBody(response, request, &input) != nil {
		return
	}
	result, err := handler.gateway.Claim(request.Context(), principal, input)
	writeResult(response, result, err)
}

func (handler *Handler) validateEffect(response http.ResponseWriter, request *http.Request, principal authorityv1.Principal) {
	var input authorityv1.EffectValidationRequest
	if !onlyQueryKeys(request) {
		writeInvalid(response)
		return
	}
	if decodeBody(response, request, &input) != nil {
		return
	}
	result, err := handler.gateway.ValidateEffect(request.Context(), principal, input)
	writeResult(response, result, err)
}

func (handler *Handler) replay(response http.ResponseWriter, request *http.Request, principal authorityv1.Principal) {
	if !onlyQueryKeys(request, "after", "event_hash", "limit") {
		writeInvalid(response)
		return
	}
	if !hasCapability(principal, authorityv1.CapabilityWorkRead) {
		writeDenial(response, http.StatusForbidden, transportDenial(authorityv1.ErrorUnauthorized, "authority.transport.capability", "obtain policy-approved work.read capability", "policy.request(work.read)"))
		return
	}
	after, err := strconv.ParseUint(request.URL.Query().Get("after"), 10, 64)
	if request.URL.Query().Get("after") == "" {
		after = 0
		err = nil
	}
	limit, limitErr := strconv.Atoi(request.URL.Query().Get("limit"))
	if request.URL.Query().Get("limit") == "" {
		limit = 100
		limitErr = nil
	}
	if err != nil || limitErr != nil {
		writeInvalid(response)
		return
	}
	page, replayErr := handler.journal.Replay(request.Context(), principal.TenantID, principal.ProjectID, authorityv1.JournalCursor{Sequence: after, EventHash: request.URL.Query().Get("event_hash")}, limit)
	writeResult(response, page, replayErr)
}

func decodeBody(response http.ResponseWriter, request *http.Request, target any) error {
	request.Body = http.MaxBytesReader(response, request.Body, maximumRequestBytes)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeInvalid(response)
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeInvalid(response)
		return errors.New("request body must contain one JSON object")
	}
	return nil
}

func writeResult(response http.ResponseWriter, result any, err error) {
	if err == nil {
		response.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(response).Encode(result)
		return
	}
	var denial *authorityv1.Denial
	if errors.As(err, &denial) {
		writeDenial(response, denialStatus(denial.Code), *denial)
		return
	}
	writeDenial(response, http.StatusServiceUnavailable, transportDenial(authorityv1.ErrorAuthorityDown, "authority.transport.unavailable", "retry after authority recovery", "work.read"))
}

func writeInvalid(response http.ResponseWriter) {
	writeDenial(response, http.StatusBadRequest, transportDenial(authorityv1.ErrorInvalidRequest, "authority.transport.request", "submit one bounded typed request", "work.read"))
}

func writeDenial(response http.ResponseWriter, status int, denial authorityv1.Denial) {
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(denial)
}

func transportDenial(code authorityv1.ErrorCode, rule, transition, action string) authorityv1.Denial {
	return authorityv1.Denial{Code: code, Rule: rule, RequiredTransition: transition, AllowedAction: action, TraceRef: "trace-invalid"}
}

func denialStatus(code authorityv1.ErrorCode) int {
	switch code {
	case authorityv1.ErrorInvalidRequest:
		return http.StatusBadRequest
	case authorityv1.ErrorUnauthorized:
		return http.StatusForbidden
	case authorityv1.ErrorTenantMismatch:
		return http.StatusForbidden
	case authorityv1.ErrorNotFound:
		return http.StatusNotFound
	case authorityv1.ErrorNotReady, authorityv1.ErrorStaleVersion, authorityv1.ErrorPolicyDenied, authorityv1.ErrorUnknownEffect:
		return http.StatusConflict
	default:
		return http.StatusServiceUnavailable
	}
}

func parseProjectRoute(path string) (string, string, bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 4 || parts[0] != "v1" || parts[1] != "projects" || parts[2] == "" {
		return "", "", false
	}
	return parts[2], strings.Join(parts[3:], "/"), true
}

func onlyQueryKeys(request *http.Request, allowed ...string) bool {
	set := make(map[string]bool, len(allowed))
	for _, key := range allowed {
		set[key] = true
	}
	for key, values := range request.URL.Query() {
		if !set[key] || len(values) != 1 {
			return false
		}
	}
	return true
}

func hasCapability(principal authorityv1.Principal, wanted authorityv1.Capability) bool {
	for _, capability := range principal.Capabilities {
		if capability == wanted {
			return true
		}
	}
	return false
}

func setSafeHeaders(response http.ResponseWriter) {
	response.Header().Set("Content-Type", "application/json")
	response.Header().Set("Cache-Control", "no-store")
	response.Header().Set("X-Content-Type-Options", "nosniff")
	response.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
}
