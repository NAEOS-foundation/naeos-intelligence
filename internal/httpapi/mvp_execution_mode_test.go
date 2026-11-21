package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/NAEOS-foundation/naeos-intelligence/internal/ratelimit"
)

func TestValidateInputAcceptsGovernedModeAsApprovalGated(t *testing.T) {
	if err := validateInput(RequestInput{Purpose: "exercise governed approval boundary", ExecutionMode: "governed"}); err != nil {
		t.Fatalf("governed request should pass admission validation: %v", err)
	}
}

func TestSubmitStagesGovernedRequestForApproval(t *testing.T) {
	h := testHandler()
	r := jsonRequest(http.MethodPost, "/api/v1/intelligence/requests",
		`{"purpose":"governed operation","execution_mode":"governed"}`)
	r.Header.Set("Authorization", "Bearer fake")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status=%d, want 202: %s", rec.Code, rec.Body.String())
	}
	var accepted Accepted
	if err := json.Unmarshal(rec.Body.Bytes(), &accepted); err != nil {
		t.Fatalf("decode accepted response: %v", err)
	}
	if accepted.Status != StatusRequiresApproval || !accepted.ApprovalRequired {
		t.Fatalf("governed request must remain approval-gated: %+v", accepted)
	}
}

func TestAnalyzeRejectsGovernedRequestEvenAfterApproval(t *testing.T) {
	store := NewMemoryStore()
	principal := Principal{Issuer: "https://issuer.test", Subject: "alice", ClientID: "client-a",
		Scopes: map[string]bool{"intelligence:write": true, "intelligence:read": true, "intelligence:approve": true}}
	req, _, err := store.Create(context.Background(), principal.OwnerID(), "", RequestInput{Purpose: "governed action", ExecutionMode: "governed"}, "corr")
	if err != nil { t.Fatal(err) }
	if _, err := store.RecordApprovalDecision(context.Background(), principal.OwnerID(), req.RequestID, "approved", principal.Subject, "approved"); err != nil { t.Fatal(err) }
	auth := Authenticator{Verifier: fakeVerifier{principal: principal}}
	h := NewHandler(store, auth, ratelimit.NewLimiter(100, 100), MustTestValidator(), nil)
	r := jsonRequest(http.MethodPost, "/api/v1/intelligence/requests/"+req.RequestID+"/analyze", `{}`)
	r.Header.Set("Authorization", "Bearer fake")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if rec.Code != http.StatusConflict { t.Fatalf("status=%d want 409: %s", rec.Code, rec.Body.String()) }
}
