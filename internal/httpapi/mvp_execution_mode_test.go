package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMVPRejectsGovernedExecutionMode(t *testing.T) {
	err := validateInput(RequestInput{
		Purpose:       "exercise MVP boundary",
		ExecutionMode: "governed",
	})
	if err == nil {
		t.Fatal("governed execution mode must be rejected in the read-only MVP")
	}
	if got, want := err.Error(), "execution_mode must be analysis_only in the MVP"; got != want {
		t.Fatalf("validation error=%q, want %q", got, want)
	}
}

func TestSubmitRejectsGovernedExecutionMode(t *testing.T) {
	h := testHandler()
	r := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/intelligence/requests",
		strings.NewReader(`{"purpose":"governed operation","execution_mode":"governed"}`),
	)
	r.Header.Set("Authorization", "Bearer fake")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, r)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d, want 422: %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), `"status":"accepted"`) {
		t.Fatalf("governed request must not be accepted: %s", rec.Body.String())
	}
}
