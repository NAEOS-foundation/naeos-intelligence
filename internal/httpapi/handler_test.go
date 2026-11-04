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

	gov "github.com/NAEOS-foundation/naeos-intelligence/internal/governance"
)

type fakeVerifier struct {
	principal Principal
	err       error
}

func (f fakeVerifier) Verify(context.Context, string) (Principal, error) { return f.principal, f.err }

func testHandler() http.Handler {
	auth := Authenticator{Verifier: fakeVerifier{principal: Principal{ClientID: "client-a", Scopes: map[string]bool{"intelligence:write": true, "intelligence:read": true}}}}
	return NewHandler(NewMemoryStore(), auth)
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
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/intelligence/requests", strings.NewReader(`{"purpose":"test","execution_mode":"analysis_only"}`)))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401", rec.Code)
	}
}

func TestInsufficientScopeRejected(t *testing.T) {
	auth := Authenticator{Verifier: fakeVerifier{principal: Principal{ClientID: "client-a", Scopes: map[string]bool{"intelligence:read": true}}}}
	h := NewHandler(NewMemoryStore(), auth)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/intelligence/requests", strings.NewReader(`{"purpose":"test","execution_mode":"analysis_only"}`))
	r.Header.Set("Authorization", "Bearer fake")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d, want 403", rec.Code)
	}
}

func TestOversizedRequestBodyRejected(t *testing.T) {
	h := testHandler()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/intelligence/requests", strings.NewReader(`{"purpose":"`+strings.Repeat("a", maxBodyBytes+1)+`","execution_mode":"analysis_only"}`))
	r.Header.Set("Authorization", "Bearer fake")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status=%d, want 413", rec.Code)
	}
}

func TestProtectedRouteFailsClosedWithoutVerifier(t *testing.T) {
	h := NewHandler(NewMemoryStore(), Authenticator{})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/intelligence/requests", strings.NewReader(`{"purpose":"test","execution_mode":"analysis_only"}`)))
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

func TestApprovalHTTPRoutesAreNotExposedInMVP(t *testing.T) {
	h := testHandler()
	for _, methodAndPath := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/intelligence/requests/request-1/approval"},
		{http.MethodPost, "/api/v1/intelligence/requests/request-1/approval"},
	} {
		r := httptest.NewRequest(methodAndPath.method, methodAndPath.path, strings.NewReader(`{"decision":"approved"}`))
		r.Header.Set("Authorization", "Bearer fake")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		if rec.Code != http.StatusNotFound {
			t.Errorf("%s %s status=%d, want 404", methodAndPath.method, methodAndPath.path, rec.Code)
		}
	}
}

func TestSubmitReplayAndConflict(t *testing.T) {
	h := testHandler()
	submit := func(body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, "/api/v1/intelligence/requests", strings.NewReader(body))
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
	for _, body := range []string{`{"purpose":`, `{"purpose":"x","execution_mode":"analysis_only","extra":"no"}`, `{"purpose":"x","execution_mode":"analysis_only"} {}`} {
		r := httptest.NewRequest(http.MethodPost, "/api/v1/intelligence/requests", strings.NewReader(body))
		r.Header.Set("Authorization", "Bearer fake")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("body=%q status=%d, want 400", body, rec.Code)
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
	r := httptest.NewRequest(http.MethodPost, "/api/v1/intelligence/requests", strings.NewReader(`{"purpose":" ","execution_mode":"analysis_only"}`))
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
	handlerA := NewHandler(store, clientA)

	create := httptest.NewRequest(http.MethodPost, "/api/v1/intelligence/requests", strings.NewReader(`{"purpose":"identity boundary","execution_mode":"analysis_only"}`))
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
	handlerB := NewHandler(store, clientB)
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
		Scopes: map[string]bool{"intelligence:read": true},
	}}}
	cases := []struct {
		name string
		handler http.Handler
		request *http.Request
		wantStatus int
		wantCode string
	}{
		{
			name: "missing bearer",
			handler: testHandler(),
			request: httptest.NewRequest(http.MethodPost, "/api/v1/intelligence/requests", strings.NewReader(`{"purpose":"x","execution_mode":"analysis_only"}`)),
			wantStatus: http.StatusUnauthorized, wantCode: "UNAUTHENTICATED",
		},
		{
			name: "insufficient scope",
			handler: NewHandler(NewMemoryStore(), authReadOnly),
			request: httptest.NewRequest(http.MethodPost, "/api/v1/intelligence/requests", strings.NewReader(`{"purpose":"x","execution_mode":"analysis_only"}`)),
			wantStatus: http.StatusForbidden, wantCode: "FORBIDDEN",
		},
		{
			name: "invalid input",
			handler: testHandler(),
			request: httptest.NewRequest(http.MethodPost, "/api/v1/intelligence/requests", strings.NewReader(`{"purpose":" ","execution_mode":"analysis_only"}`)),
			wantStatus: http.StatusUnprocessableEntity, wantCode: "UNPROCESSABLE_REQUEST",
		},
		{
			name: "malformed body",
			handler: testHandler(),
			request: httptest.NewRequest(http.MethodPost, "/api/v1/intelligence/requests", strings.NewReader(`{"purpose":`)),
			wantStatus: http.StatusBadRequest, wantCode: "INVALID_REQUEST",
		},
		{
			name: "missing verifier",
			handler: NewHandler(NewMemoryStore(), Authenticator{}),
			request: httptest.NewRequest(http.MethodPost, "/api/v1/intelligence/requests", strings.NewReader(`{"purpose":"x","execution_mode":"analysis_only"}`)),
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
	r := httptest.NewRequest(http.MethodPost, "/api/v1/intelligence/requests", strings.NewReader(string(body)))
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
			Scopes: map[string]bool{"intelligence:read": true, "intelligence:write": true},
		}}}
	}
	handlerA := NewHandler(store, authFor("client-a"))
	handlerB := NewHandler(store, authFor("client-b"))
	const body = "{\"purpose\":\"ownership contract\",\"execution_mode\":\"analysis_only\"}"

	submit := func(h http.Handler, key string) Accepted {
		r := httptest.NewRequest(http.MethodPost, "/api/v1/intelligence/requests", strings.NewReader(body))
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
		Scopes: map[string]bool{"intelligence:read": true, "intelligence:write": true},
	}}}
	h := NewHandler(store, auth)
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
