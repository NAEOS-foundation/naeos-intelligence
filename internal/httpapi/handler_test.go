package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NAEOS-foundation/naeos-intelligence/internal/github"
	gov "github.com/NAEOS-foundation/naeos-intelligence/internal/governance"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/intelligence/classifier"
	intctx "github.com/NAEOS-foundation/naeos-intelligence/internal/intelligence/context"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/intelligence/orchestrator"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/intelligence/planner"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/model"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/naeoscore"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/ratelimit"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/runtime"
)

type fakeVerifier struct {
	principal Principal
	err       error
}

func (f fakeVerifier) Verify(context.Context, string) (Principal, error) { return f.principal, f.err }

func MustTestValidator() *OpenAPIValidator {
	v, err := NewOpenAPIValidator("../../api/openapi.yaml")
	if err != nil {
		panic(fmt.Sprintf("create test OpenAPI validator: %v", err))
	}
	return v
}

func jsonRequest(method, path, body string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func testHandler() http.Handler {
	auth := Authenticator{Verifier: fakeVerifier{principal: Principal{ClientID: "client-a", Scopes: map[string]bool{"intelligence:write": true, "intelligence:read": true}}}}
	return NewHandler(NewMemoryStore(), auth, ratelimit.NewLimiter(100, 100), MustTestValidator(), nil)
}

func TestHealthIsPublicAndMinimal(t *testing.T) {
	rec := httptest.NewRecorder()
	testHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != `{"status":"healthy"}` {
		t.Fatalf("unexpected health response: %d %s", rec.Code, rec.Body.String())
	}
}

func TestMissingBearerTokenRejected(t *testing.T) {
	h := testHandler()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, jsonRequest(http.MethodPost, "/api/v1/intelligence/requests", `{"purpose":"test","execution_mode":"analysis_only"}`))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401", rec.Code)
	}
}

func TestInsufficientScopeRejected(t *testing.T) {
	auth := Authenticator{Verifier: fakeVerifier{principal: Principal{ClientID: "client-a", Scopes: map[string]bool{"intelligence:read": true}}}}
	h := NewHandler(NewMemoryStore(), auth, ratelimit.NewLimiter(100, 100), MustTestValidator(), nil)
	r := jsonRequest(http.MethodPost, "/api/v1/intelligence/requests", `{"purpose":"test","execution_mode":"analysis_only"}`)
	r.Header.Set("Authorization", "Bearer fake")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d, want 403", rec.Code)
	}
}

func TestOversizedRequestBodyRejected(t *testing.T) {
	h := testHandler()
	r := jsonRequest(http.MethodPost, "/api/v1/intelligence/requests", `{"purpose":"`+strings.Repeat("a", maxBodyBytes+1)+`","execution_mode":"analysis_only"}`)
	r.Header.Set("Authorization", "Bearer fake")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status=%d, want 413", rec.Code)
	}
}

func TestProtectedRouteFailsClosedWithoutVerifier(t *testing.T) {
	h := NewHandler(NewMemoryStore(), Authenticator{}, ratelimit.NewLimiter(100, 100), MustTestValidator(), nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, jsonRequest(http.MethodPost, "/api/v1/intelligence/requests", `{"purpose":"test","execution_mode":"analysis_only"}`))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d, want 503", rec.Code)
	}
}

func TestMemoryStoreMarksGovernedRequestsRequiresApproval(t *testing.T) {
	store := NewMemoryStore()
	req, replay, err := store.Create(context.Background(), "client-1", "", RequestInput{Purpose: "perform mutation", ExecutionMode: "governed"}, "corr-1")
	if err != nil {
		t.Fatalf("create error: %v", err)
	}
	if replay {
		t.Fatal("governed request should not be replayed")
	}
	if req.Status != StatusRequiresApproval {
		t.Fatalf("status=%q, want requires_approval", req.Status)
	}
	if !req.ApprovalRequired {
		t.Fatal("governed request should expose approval_required=true")
	}
}

func TestMemoryStoreTracksApprovalDecisionForGovernedRequest(t *testing.T) {
	workflow := gov.NewApprovalWorkflow()
	store := NewMemoryStoreWithApproval(workflow)
	req, _, err := store.Create(context.Background(), "client-1", "", RequestInput{Purpose: "perform mutation", ExecutionMode: "governed"}, "corr-1")
	if err != nil {
		t.Fatalf("create error: %v", err)
	}
	decision, ok := store.ApprovalDecision(req.RequestID)
	if !ok {
		t.Fatal("governed request should have a recorded approval decision")
	}
	if !decision.ApprovalRequired {
		t.Fatal("approval decision should require approval")
	}
	if decision.Decision != "pending" {
		t.Fatalf("decision=%q, want pending", decision.Decision)
	}
}

func TestApprovalEndpointRecordsDecision(t *testing.T) {
	store := NewMemoryStore()
	auth := Authenticator{Verifier: fakeVerifier{principal: Principal{
		ClientID: "client-a",
		Scopes:   map[string]bool{"intelligence:write": true, "intelligence:read": true},
	}}}
	h := NewHandler(store, auth, ratelimit.NewLimiter(100, 100), MustTestValidator(), nil)

	// Create a governed request that requires approval
	req, _, err := store.Create(context.Background(), "client-a", "", RequestInput{Purpose: "perform mutation", ExecutionMode: "governed"}, "corr-1")
	if err != nil {
		t.Fatalf("create error: %v", err)
	}
	if req.Status != StatusRequiresApproval {
		t.Fatalf("status=%q, want requires_approval", req.Status)
	}

	// Record an approval decision
	body := `{"decision":"approved","approved_by":"alice","reason":"risk assessed"}`
	r := jsonRequest(http.MethodPost, fmt.Sprintf("/api/v1/intelligence/requests/%s/approval", req.RequestID), body)
	r.Header.Set("Authorization", "Bearer fake")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if rec.Code != http.StatusOK {
		t.Fatalf("approve status=%d, want 200: %s", rec.Code, rec.Body.String())
	}
	var decision ApprovalDecisionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &decision); err != nil {
		t.Fatalf("decode decision: %v", err)
	}
	if decision.Decision != "approved" {
		t.Fatalf("decision=%q, want approved", decision.Decision)
	}
	if decision.ApprovedBy != "alice" {
		t.Fatalf("approved_by=%q, want alice", decision.ApprovedBy)
	}
	if decision.RequestID != req.RequestID {
		t.Fatalf("request_id=%q, want %q", decision.RequestID, req.RequestID)
	}
}

func TestApprovalEndpointRejectsInvalidDecision(t *testing.T) {
	store := NewMemoryStore()
	auth := Authenticator{Verifier: fakeVerifier{principal: Principal{
		ClientID: "client-a",
		Scopes:   map[string]bool{"intelligence:write": true, "intelligence:read": true},
	}}}
	h := NewHandler(store, auth, ratelimit.NewLimiter(100, 100), MustTestValidator(), nil)

	req, _, err := store.Create(context.Background(), "client-a", "", RequestInput{Purpose: "perform mutation", ExecutionMode: "governed"}, "corr-1")
	if err != nil {
		t.Fatalf("create error: %v", err)
	}

	// Invalid decision value
	body := `{"decision":"maybe"}`
	r := jsonRequest(http.MethodPost, fmt.Sprintf("/api/v1/intelligence/requests/%s/approval", req.RequestID), body)
	r.Header.Set("Authorization", "Bearer fake")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d, want 422: %s", rec.Code, rec.Body.String())
	}
}

func TestApprovalEndpointMissingDecision(t *testing.T) {
	store := NewMemoryStore()
	auth := Authenticator{Verifier: fakeVerifier{principal: Principal{
		ClientID: "client-a",
		Scopes:   map[string]bool{"intelligence:write": true, "intelligence:read": true},
	}}}
	h := NewHandler(store, auth, ratelimit.NewLimiter(100, 100), MustTestValidator(), nil)

	req, _, err := store.Create(context.Background(), "client-a", "", RequestInput{Purpose: "perform mutation", ExecutionMode: "governed"}, "corr-1")
	if err != nil {
		t.Fatalf("create error: %v", err)
	}

	body := `{}`
	r := jsonRequest(http.MethodPost, fmt.Sprintf("/api/v1/intelligence/requests/%s/approval", req.RequestID), body)
	r.Header.Set("Authorization", "Bearer fake")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d, want 422: %s", rec.Code, rec.Body.String())
	}
}

func TestApprovalEndpointCrossClientRejected(t *testing.T) {
	store := NewMemoryStore()
	authB := Authenticator{Verifier: fakeVerifier{principal: Principal{
		ClientID: "client-b",
		Scopes:   map[string]bool{"intelligence:write": true, "intelligence:read": true},
	}}}

	req, _, err := store.Create(context.Background(), "client-a", "", RequestInput{Purpose: "perform mutation", ExecutionMode: "governed"}, "corr-1")
	if err != nil {
		t.Fatalf("create error: %v", err)
	}

	// Client B tries to approve client A's request
	hB := NewHandler(store, authB, ratelimit.NewLimiter(100, 100), MustTestValidator(), nil)
	body := `{"decision":"approved"}`
	r := jsonRequest(http.MethodPost, fmt.Sprintf("/api/v1/intelligence/requests/%s/approval", req.RequestID), body)
	r.Header.Set("Authorization", "Bearer fake")
	rec := httptest.NewRecorder()
	hB.ServeHTTP(rec, r)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d, want 404: %s", rec.Code, rec.Body.String())
	}
}

func TestSubmitReplayAndConflict(t *testing.T) {
	h := testHandler()
	submit := func(body string) *httptest.ResponseRecorder {
		r := jsonRequest(http.MethodPost, "/api/v1/intelligence/requests", body)
		r.Header.Set("Authorization", "Bearer fake")
		r.Header.Set("Idempotency-Key", "key-1")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		return rec
	}
	body := `{"purpose":" inspect ","execution_mode":"analysis_only"}`
	first := submit(body)
	second := submit(body)
	if first.Code != http.StatusAccepted || second.Code != http.StatusAccepted {
		t.Fatalf("statuses first=%d second=%d", first.Code, second.Code)
	}
	if second.Header().Get("Idempotency-Replayed") != "true" {
		t.Fatal("expected replay header")
	}
	var accepted Accepted
	if err := json.Unmarshal(first.Body.Bytes(), &accepted); err != nil {
		t.Fatalf("decode accepted response: %v", err)
	}
	if accepted.Status != StatusAccepted || accepted.ApprovalRequired || accepted.StatusURL == "" || first.Header().Get("Location") != accepted.StatusURL {
		t.Fatalf("accepted response violates API contract: %+v headers=%v", accepted, first.Header())
	}
	conflict := submit(`{"purpose":"different","execution_mode":"analysis_only"}`)
	if conflict.Code != http.StatusConflict {
		t.Fatalf("conflict status=%d, want 409", conflict.Code)
	}
	var conflictProblem problem
	if err := json.Unmarshal(conflict.Body.Bytes(), &conflictProblem); err != nil {
		t.Fatalf("decode conflict problem: %v", err)
	}
	if conflictProblem.Code != "CONFLICT" || conflictProblem.Status != http.StatusConflict {
		t.Fatalf("conflict problem code/status=%s/%d", conflictProblem.Code, conflictProblem.Status)
	}
}

func TestMalformedAndUnknownFieldsRejected(t *testing.T) {
	h := testHandler()
	cases := []struct {
		body     string
		wantCode int
	}{
		{body: `{"purpose":`, wantCode: http.StatusBadRequest},
		{body: `{"purpose":"x","execution_mode":"analysis_only","extra":"no"}`, wantCode: http.StatusUnprocessableEntity},
		{body: `{"purpose":"x","execution_mode":"analysis_only"} {}`, wantCode: http.StatusBadRequest},
	}
	for _, tc := range cases {
		r := jsonRequest(http.MethodPost, "/api/v1/intelligence/requests", tc.body)
		r.Header.Set("Authorization", "Bearer fake")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		if rec.Code != tc.wantCode {
			t.Errorf("body=%q status=%d, want %d", tc.body, rec.Code, tc.wantCode)
		}
	}
}

func TestOwnerIsolationAndCancellation(t *testing.T) {
	store := NewMemoryStore()
	input := RequestInput{Purpose: "p", ExecutionMode: "analysis_only"}
	req, _, err := store.Create(context.Background(), "client-a", "", input, "test-correlation")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.Get(context.Background(), "client-b", req.RequestID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-client Get err=%v", err)
	}
	cancelled, err := store.Cancel(context.Background(), "client-a", req.RequestID, "test-correlation")
	if err != nil || cancelled.Status != StatusCancelled {
		t.Fatalf("cancel=%+v err=%v", cancelled, err)
	}
	if _, err = store.Cancel(context.Background(), "client-a", req.RequestID, "test-correlation"); err != nil {
		t.Fatalf("repeat cancellation should be idempotent: %v", err)
	}
}

func TestInvalidInputProduces422(t *testing.T) {
	h := testHandler()
	r := jsonRequest(http.MethodPost, "/api/v1/intelligence/requests", `{"purpose":" ","execution_mode":"analysis_only"}`)
	r.Header.Set("Authorization", "Bearer fake")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d, want 422", rec.Code)
	}
}

func TestIdentityHeadersCannotOverrideValidatedPrincipal(t *testing.T) {
	store := NewMemoryStore()
	clientA := Authenticator{Verifier: fakeVerifier{principal: Principal{
		ClientID: "client-a",
		Scopes:   map[string]bool{"intelligence:read": true, "intelligence:write": true},
	}}}
	handlerA := NewHandler(store, clientA, ratelimit.NewLimiter(100, 100), MustTestValidator(), nil)

	create := jsonRequest(http.MethodPost, "/api/v1/intelligence/requests", `{"purpose":"identity boundary","execution_mode":"analysis_only"}`)
	create.Header.Set("Authorization", "Bearer validated-client-a")
	create.Header.Set("X-Client-ID", "client-b")
	create.Header.Set("X-Principal", "client-b")
	created := httptest.NewRecorder()
	handlerA.ServeHTTP(created, create)
	if created.Code != http.StatusAccepted {
		t.Fatalf("create status=%d, want 202: %s", created.Code, created.Body.String())
	}
	var accepted Accepted
	if err := json.Unmarshal(created.Body.Bytes(), &accepted); err != nil {
		t.Fatal(err)
	}
	if accepted.RequestID == "" {
		t.Fatal("accepted response must contain request id")
	}

	// The spoofed headers must not change the validated client's ownership context.
	getA := httptest.NewRequest(http.MethodGet, accepted.StatusURL, nil)
	getA.Header.Set("Authorization", "Bearer validated-client-a")
	getA.Header.Set("X-Client-ID", "client-b")
	getA.Header.Set("X-Principal", "client-b")
	recA := httptest.NewRecorder()
	handlerA.ServeHTTP(recA, getA)
	if recA.Code != http.StatusOK {
		t.Fatalf("owner access status=%d, want 200: %s", recA.Code, recA.Body.String())
	}

	// A genuinely different validated client remains isolated from client-a's resource.
	clientB := Authenticator{Verifier: fakeVerifier{principal: Principal{
		ClientID: "client-b",
		Scopes:   map[string]bool{"intelligence:read": true},
	}}}
	handlerB := NewHandler(store, clientB, ratelimit.NewLimiter(100, 100), MustTestValidator(), nil)
	getB := httptest.NewRequest(http.MethodGet, accepted.StatusURL, nil)
	getB.Header.Set("Authorization", "Bearer validated-client-b")
	recB := httptest.NewRecorder()
	handlerB.ServeHTTP(recB, getB)
	if recB.Code != http.StatusNotFound {
		t.Fatalf("cross-client access status=%d, want 404: %s", recB.Code, recB.Body.String())
	}
}

// TestProblemResponsesMatchPublicContract verifies the stable public error envelope,
// status-to-code mapping, media type, and request correlation behavior.
func TestProblemResponsesMatchPublicContract(t *testing.T) {
	authReadOnly := Authenticator{Verifier: fakeVerifier{principal: Principal{
		ClientID: "client-a",
		Scopes:   map[string]bool{"intelligence:read": true},
	}}}
	cases := []struct {
		name       string
		handler    http.Handler
		request    *http.Request
		wantStatus int
		wantCode   string
	}{
		{
			name:       "missing bearer",
			handler:    testHandler(),
			request:    jsonRequest(http.MethodPost, "/api/v1/intelligence/requests", `{"purpose":"x","execution_mode":"analysis_only"}`),
			wantStatus: http.StatusUnauthorized, wantCode: "UNAUTHENTICATED",
		},
		{
			name:       "insufficient scope",
			handler:    NewHandler(NewMemoryStore(), authReadOnly, ratelimit.NewLimiter(100, 100), MustTestValidator(), nil),
			request:    jsonRequest(http.MethodPost, "/api/v1/intelligence/requests", `{"purpose":"x","execution_mode":"analysis_only"}`),
			wantStatus: http.StatusForbidden, wantCode: "FORBIDDEN",
		},
		{
			name:       "invalid input",
			handler:    testHandler(),
			request:    jsonRequest(http.MethodPost, "/api/v1/intelligence/requests", `{"purpose":" ","execution_mode":"analysis_only"}`),
			wantStatus: http.StatusUnprocessableEntity, wantCode: "UNPROCESSABLE_REQUEST",
		},
		{
			name:       "malformed body",
			handler:    testHandler(),
			request:    jsonRequest(http.MethodPost, "/api/v1/intelligence/requests", `{"purpose":`),
			wantStatus: http.StatusBadRequest, wantCode: "INVALID_REQUEST",
		},
		{
			name:       "missing verifier",
			handler:    NewHandler(NewMemoryStore(), Authenticator{}, ratelimit.NewLimiter(100, 100), MustTestValidator(), nil),
			request:    jsonRequest(http.MethodPost, "/api/v1/intelligence/requests", `{"purpose":"x","execution_mode":"analysis_only"}`),
			wantStatus: http.StatusServiceUnavailable, wantCode: "SERVICE_UNAVAILABLE",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			const correlationID = "contract-test-req-01"
			tc.request.Header.Set("X-Request-ID", correlationID)
			if tc.name != "missing bearer" && tc.name != "missing verifier" {
				tc.request.Header.Set("Authorization", "Bearer fake")
			}
			rec := httptest.NewRecorder()
			tc.handler.ServeHTTP(rec, tc.request)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status=%d want=%d body=%s", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if got := rec.Header().Get("Content-Type"); got != "application/problem+json" {
				t.Fatalf("Content-Type=%q want application/problem+json", got)
			}
			if got := rec.Header().Get("X-Request-ID"); got != correlationID {
				t.Fatalf("X-Request-ID=%q want %q", got, correlationID)
			}
			var body problem
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode problem: %v", err)
			}
			if body.Status != tc.wantStatus || body.Code != tc.wantCode || body.RequestID != correlationID || body.Detail == "" || body.Title == "" || body.Type == "" {
				t.Fatalf("problem envelope does not satisfy contract: %+v", body)
			}
		})
	}
}

func TestMetadataPropertyLimitEnforced(t *testing.T) {
	metadata := make(map[string]string, 51)
	for i := 0; i < 51; i++ {
		metadata[fmt.Sprintf("key-%d", i)] = "value"
	}
	body, err := json.Marshal(RequestInput{Purpose: "metadata limit", ExecutionMode: "analysis_only", Metadata: metadata})
	if err != nil {
		t.Fatal(err)
	}
	r := jsonRequest(http.MethodPost, "/api/v1/intelligence/requests", string(body))
	r.Header.Set("Authorization", "Bearer fake")
	rec := httptest.NewRecorder()
	testHandler().ServeHTTP(rec, r)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d want=422 body=%s", rec.Code, rec.Body.String())
	}
	var p problem
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if p.Code != "UNPROCESSABLE_REQUEST" {
		t.Fatalf("code=%q want UNPROCESSABLE_REQUEST", p.Code)
	}
}

func TestHTTPClientOwnershipIsolationAndIdempotencyScope(t *testing.T) {
	store := NewMemoryStore()
	authFor := func(clientID string) Authenticator {
		return Authenticator{Verifier: fakeVerifier{principal: Principal{
			ClientID: clientID,
			Scopes:   map[string]bool{"intelligence:read": true, "intelligence:write": true},
		}}}
	}
	handlerA := NewHandler(store, authFor("client-a"), ratelimit.NewLimiter(100, 100), MustTestValidator(), nil)
	handlerB := NewHandler(store, authFor("client-b"), ratelimit.NewLimiter(100, 100), MustTestValidator(), nil)
	const body = "{\"purpose\":\"ownership contract\",\"execution_mode\":\"analysis_only\"}"

	submit := func(h http.Handler, key string) Accepted {
		r := jsonRequest(http.MethodPost, "/api/v1/intelligence/requests", body)
		r.Header.Set("Authorization", "Bearer fake")
		r.Header.Set("Idempotency-Key", key)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		if rec.Code != http.StatusAccepted {
			t.Fatalf("submit status=%d body=%s", rec.Code, rec.Body.String())
		}
		var accepted Accepted
		if err := json.Unmarshal(rec.Body.Bytes(), &accepted); err != nil {
			t.Fatal(err)
		}
		return accepted
	}

	ownedByA := submit(handlerA, "tenant-scoped-key")
	// The same idempotency key in a different client scope must create a distinct request.
	ownedByB := submit(handlerB, "tenant-scoped-key")
	if ownedByA.RequestID == ownedByB.RequestID {
		t.Fatal("idempotency key must be scoped to authenticated client")
	}

	getForeign := httptest.NewRequest(http.MethodGet, ownedByA.StatusURL, nil)
	getForeign.Header.Set("Authorization", "Bearer fake")
	rec := httptest.NewRecorder()
	handlerB.ServeHTTP(rec, getForeign)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("foreign GET status=%d want=404 body=%s", rec.Code, rec.Body.String())
	}
	var p problem
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if p.Code != "RESOURCE_NOT_FOUND" || p.Detail != "Request was not found" {
		t.Fatalf("foreign access must be non-disclosing: %+v", p)
	}
}

func TestCancelCompletedRequestReturnsContractConflict(t *testing.T) {
	store := NewMemoryStore()
	req, _, err := store.Create(context.Background(), "client-a", "", RequestInput{Purpose: "terminal state", ExecutionMode: "analysis_only"}, "corr")
	if err != nil {
		t.Fatal(err)
	}
	store.mu.Lock()
	req.Status = StatusCompleted
	store.requests[req.RequestID] = req
	store.mu.Unlock()

	auth := Authenticator{Verifier: fakeVerifier{principal: Principal{
		ClientID: "client-a",
		Scopes:   map[string]bool{"intelligence:read": true, "intelligence:write": true},
	}}}
	h := NewHandler(store, auth, ratelimit.NewLimiter(100, 100), MustTestValidator(), nil)
	r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/intelligence/requests/%s/cancel", req.RequestID), nil)
	r.Header.Set("Authorization", "Bearer fake")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if rec.Code != http.StatusConflict {
		t.Fatalf("cancel completed request status=%d want=409 body=%s", rec.Code, rec.Body.String())
	}
	var p problem
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if p.Code != "CONFLICT" || p.Status != http.StatusConflict {
		t.Fatalf("conflict response violates public contract: %+v", p)
	}
}

func TestAnalyzeEndpoint_RunPipeline(t *testing.T) {
	cl := classifier.NewRuleClassifier()
	cb := intctx.NewRuleBuilder()
	pl := planner.NewRulePlanner()
	reg := model.NewRegistry()
	_ = reg.Register(&mockProvider{id: "test-provider"})
	rp := runtime.NewProcessor(reg)
	orch := orchestrator.NewOrchestrator(cl, cb, pl, rp)

	auth := Authenticator{Verifier: fakeVerifier{principal: Principal{ClientID: "client-a", Scopes: map[string]bool{"intelligence:write": true, "intelligence:read": true}}}}
	h := NewHandler(NewMemoryStore(), auth, ratelimit.NewLimiter(100, 100), MustTestValidator(), orch)

	// Submit a request first
	subReq := jsonRequest(http.MethodPost, "/api/v1/intelligence/requests", `{"purpose":"Analyze repository architecture and technical debt","execution_mode":"analysis_only","context_sources":[{"type":"repository","reference":"github.com/owner/repo"}],"desired_outputs":["risk_summary"]}`)
	subReq.Header.Set("Authorization", "Bearer fake")
	subReq.Header.Set("X-Request-ID", "test-req-001")
	subRec := httptest.NewRecorder()
	h.ServeHTTP(subRec, subReq)
	if subRec.Code != http.StatusAccepted {
		t.Fatalf("submit status=%d want=202 body=%s", subRec.Code, subRec.Body.String())
	}
	var accepted Accepted
	if err := json.Unmarshal(subRec.Body.Bytes(), &accepted); err != nil {
		t.Fatal(err)
	}
	if accepted.RequestID == "" {
		t.Fatal("expected request_id in accepted response")
	}

	// Run analyze on the submitted request
	anaReq := jsonRequest(http.MethodPost, fmt.Sprintf("/api/v1/intelligence/requests/%s/analyze", accepted.RequestID), `{}`)
	anaReq.Header.Set("Authorization", "Bearer fake")
	anaRec := httptest.NewRecorder()
	h.ServeHTTP(anaRec, anaReq)
	if anaRec.Code != http.StatusOK {
		t.Fatalf("analyze status=%d want=200 body=%s", anaRec.Code, anaRec.Body.String())
	}
	var result map[string]any
	if err := json.Unmarshal(anaRec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["intent"] != "repository_analysis" {
		t.Errorf("intent=%v want=repository_analysis", result["intent"])
	}
	if result["request_id"] != accepted.RequestID {
		t.Errorf("request_id=%v want=%s", result["request_id"], accepted.RequestID)
	}
	plan, ok := result["plan"].(map[string]any)
	if !ok {
		t.Fatal("plan not found in response")
	}
	tasks, ok := plan["tasks"].([]any)
	if !ok || len(tasks) == 0 {
		t.Fatalf("plan.tasks missing or empty: %v", plan["tasks"])
	}
	runtimeResult, ok := result["runtime_result"].(map[string]any)
	if !ok {
		t.Fatal("runtime_result not found in response")
	}
	if runtimeResult["allowed"] != true {
		t.Errorf("runtime_result.allowed=%v want=true", runtimeResult["allowed"])
	}
}

type mockProvider struct {
	id string
}

func (m *mockProvider) ID() string { return m.id }
func (m *mockProvider) Generate(context.Context, model.Request) (model.Response, error) {
	return model.Response{ModelID: m.id, Text: "fake"}, nil
}
func (m *mockProvider) HealthCheck(context.Context) error { return nil }
func (m *mockProvider) Capabilities() []string            { return []string{"text"} }

func TestAnalyzeEndpoint_WithGitHubResolver(t *testing.T) {
	// Mock GitHub API server
	repoJSON := `{"id":123,"name":"demo","full_name":"owner/demo","description":"A demo repository","private":false,"fork":false,"archived":false,"default_branch":"main","open_issues_count":5,"stargazers_count":42,"forks_count":7,"size":1024,"html_url":"https://github.com/owner/demo"}`
	ghSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/demo" {
			t.Errorf("unexpected GitHub path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(repoJSON))
	}))
	defer ghSrv.Close()

	ghClient := github.NewClient("test-token", "naeos-bot")
	ghClient.SetBaseURL(ghSrv.URL)

	ghResolver := github.NewGitHubSourceResolver(ghClient)

	cl := classifier.NewRuleClassifier()
	cb := intctx.NewRuleBuilder(intctx.WithResolver(ghResolver))
	pl := planner.NewRulePlanner()
	reg := model.NewRegistry()
	_ = reg.Register(&mockProvider{id: "test-provider"})
	rp := runtime.NewProcessor(reg)
	orch := orchestrator.NewOrchestrator(cl, cb, pl, rp)

	auth := Authenticator{Verifier: fakeVerifier{principal: Principal{ClientID: "client-a", Scopes: map[string]bool{"intelligence:write": true, "intelligence:read": true}}}}
	h := NewHandler(NewMemoryStore(), auth, ratelimit.NewLimiter(100, 100), MustTestValidator(), orch)

	// Submit request with GitHub repository context source
	body := `{"purpose":"Analyze repository architecture","execution_mode":"analysis_only","context_sources":[{"type":"repository","reference":"owner/demo"}]}`
	subReq := jsonRequest(http.MethodPost, "/api/v1/intelligence/requests", body)
	subReq.Header.Set("Authorization", "Bearer fake")
	subReq.Header.Set("X-Request-ID", "github-e2e-001")
	subRec := httptest.NewRecorder()
	h.ServeHTTP(subRec, subReq)
	if subRec.Code != http.StatusAccepted {
		t.Fatalf("submit status=%d want=202 body=%s", subRec.Code, subRec.Body.String())
	}
	var accepted Accepted
	if err := json.Unmarshal(subRec.Body.Bytes(), &accepted); err != nil {
		t.Fatal(err)
	}

	// Run analyze
	anaReq := jsonRequest(http.MethodPost, fmt.Sprintf("/api/v1/intelligence/requests/%s/analyze", accepted.RequestID), `{}`)
	anaReq.Header.Set("Authorization", "Bearer fake")
	anaRec := httptest.NewRecorder()
	h.ServeHTTP(anaRec, anaReq)
	if anaRec.Code != http.StatusOK {
		t.Fatalf("analyze status=%d want=200 body=%s", anaRec.Code, anaRec.Body.String())
	}
	var result map[string]any
	if err := json.Unmarshal(anaRec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["intent"] != "repository_analysis" {
		t.Errorf("intent=%v want=repository_analysis", result["intent"])
	}
	plan, ok := result["plan"].(map[string]any)
	if !ok {
		t.Fatalf("plan not found: %v", result["plan"])
	}
	tasks, ok := plan["tasks"].([]any)
	if !ok || len(tasks) == 0 {
		t.Fatalf("plan.tasks missing or empty: %v", plan["tasks"])
	}
	if result["runtime_result"].(map[string]any)["allowed"] != true {
		t.Error("runtime_result.allowed want=true")
	}
}

func TestAnalyzeEndpoint_WithNAEOSCoreResolver(t *testing.T) {
	// Mock NAEOS Core API server
	repoJSON := `{"id":"repo-123","name":"demo","full_name":"owner/demo","description":"A demo repository","private":false,"archived":false,"default_branch":"main","open_issues":5,"stars":42,"forks":7,"size_kb":1024,"created_at":"2024-01-01T00:00:00Z","updated_at":"2024-02-01T00:00:00Z","url":"https://naeos.dev/repos/owner/demo"}`
	ncSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/repositories/owner/demo" {
			t.Errorf("unexpected NAEOS Core path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(repoJSON))
	}))
	defer ncSrv.Close()

	ncClient := naeoscore.NewClient("test-token", "naeos-bot")
	ncClient.SetBaseURL(ncSrv.URL)

	ncResolver := naeoscore.NewCoreSourceResolver(ncClient)

	cl := classifier.NewRuleClassifier()
	cb := intctx.NewRuleBuilder(intctx.WithResolver(ncResolver))
	pl := planner.NewRulePlanner()
	reg := model.NewRegistry()
	_ = reg.Register(&mockProvider{id: "test-provider"})
	rp := runtime.NewProcessor(reg)
	orch := orchestrator.NewOrchestrator(cl, cb, pl, rp)

	auth := Authenticator{Verifier: fakeVerifier{principal: Principal{ClientID: "client-a", Scopes: map[string]bool{"intelligence:write": true, "intelligence:read": true}}}}
	h := NewHandler(NewMemoryStore(), auth, ratelimit.NewLimiter(100, 100), MustTestValidator(), orch)

	// Submit request with NAEOS Core repository context source
	body := `{"purpose":"Analyze repository architecture","execution_mode":"analysis_only","context_sources":[{"type":"naeos-core","reference":"owner/demo"}]}`
	subReq := jsonRequest(http.MethodPost, "/api/v1/intelligence/requests", body)
	subReq.Header.Set("Authorization", "Bearer fake")
	subReq.Header.Set("X-Request-ID", "naeoscore-e2e-001")
	subRec := httptest.NewRecorder()
	h.ServeHTTP(subRec, subReq)
	if subRec.Code != http.StatusAccepted {
		t.Fatalf("submit status=%d want=202 body=%s", subRec.Code, subRec.Body.String())
	}
	var accepted Accepted
	if err := json.Unmarshal(subRec.Body.Bytes(), &accepted); err != nil {
		t.Fatal(err)
	}

	// Run analyze
	anaReq := jsonRequest(http.MethodPost, fmt.Sprintf("/api/v1/intelligence/requests/%s/analyze", accepted.RequestID), `{}`)
	anaReq.Header.Set("Authorization", "Bearer fake")
	anaRec := httptest.NewRecorder()
	h.ServeHTTP(anaRec, anaReq)
	if anaRec.Code != http.StatusOK {
		t.Fatalf("analyze status=%d want=200 body=%s", anaRec.Code, anaRec.Body.String())
	}
	var result map[string]any
	if err := json.Unmarshal(anaRec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["intent"] != "repository_analysis" {
		t.Errorf("intent=%v want=repository_analysis", result["intent"])
	}
	plan, ok := result["plan"].(map[string]any)
	if !ok {
		t.Fatalf("plan not found: %v", result["plan"])
	}
	tasks, ok := plan["tasks"].([]any)
	if !ok || len(tasks) == 0 {
		t.Fatalf("plan.tasks missing or empty: %v", plan["tasks"])
	}
	if result["runtime_result"].(map[string]any)["allowed"] != true {
		t.Error("runtime_result.allowed want=true")
	}
}
