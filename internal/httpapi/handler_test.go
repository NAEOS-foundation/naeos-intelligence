package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
	if _, err = store.Cancel("client-a", req.RequestID); err != nil {
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
