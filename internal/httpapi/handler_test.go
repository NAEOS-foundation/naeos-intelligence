package httpapi

import (
	"context"
	"encoding/json"
	"errors"
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

func TestApprovalDecisionEndpointRecordsGovernedOutcome(t *testing.T) {
	auth := Authenticator{Verifier: fakeVerifier{principal: Principal{ClientID: "client-a", Scopes: map[string]bool{"intelligence:write": true, "intelligence:read": true}}}}
	store := NewMemoryStoreWithApproval(gov.NewApprovalWorkflow())
	request, _, err := store.Create(context.Background(), "client-a", "", RequestInput{Purpose: "perform mutation", ExecutionMode: "governed"}, "corr-1")
	if err != nil {
		t.Fatalf("create error: %v", err)
	}
	h := NewHandler(store, auth)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/intelligence/requests/"+request.RequestID+"/approval", strings.NewReader(`{"decision":"approved","approved_by":"ops-user","reason":"review accepted"}`))
	r.Header.Set("Authorization", "Bearer fake")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status=%d, want 202: %s", rec.Code, rec.Body.String())
	}
	decision, ok := store.ApprovalDecision(request.RequestID)
	if !ok {
		t.Fatal("approval record was not persisted")
	}
	if decision.Decision != "approved" {
		t.Fatalf("decision=%q, want approved", decision.Decision)
	}
	if decision.ApprovedBy != "ops-user" {
		t.Fatalf("approved_by=%q, want ops-user", decision.ApprovedBy)
	}
	updated, err := store.Get(context.Background(), "client-a", request.RequestID)
	if err != nil {
		t.Fatalf("get error: %v", err)
	}
	if updated.Status != StatusQueued {
		t.Fatalf("status=%q, want queued after approval", updated.Status)
	}
}

func TestApprovalDecisionCanBeRetrievedByRequestOwner(t *testing.T) {
	auth := Authenticator{Verifier: fakeVerifier{principal: Principal{ClientID: "client-a", Scopes: map[string]bool{"intelligence:write": true, "intelligence:read": true}}}}
	store := NewMemoryStoreWithApproval(gov.NewApprovalWorkflow())
	request, _, err := store.Create(context.Background(), "client-a", "", RequestInput{Purpose: "perform mutation", ExecutionMode: "governed"}, "corr-1")
	if err != nil {
		t.Fatalf("create error: %v", err)
	}
	if _, err := store.RecordApprovalDecision(request.RequestID, "approved", "ops-user", "review accepted"); err != nil {
		t.Fatalf("record approval error: %v", err)
	}
	h := NewHandler(store, auth)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/intelligence/requests/"+request.RequestID+"/approval", nil)
	r.Header.Set("Authorization", "Bearer fake")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"decision":"approved"`) {
		t.Fatalf("response=%s, want approved decision", rec.Body.String())
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
	conflict := submit(`{"purpose":"different","execution_mode":"analysis_only"}`)
	if conflict.Code != http.StatusConflict {
		t.Fatalf("conflict status=%d, want 409", conflict.Code)
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
