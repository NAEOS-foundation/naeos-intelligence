package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// secretPatterns are representative secret-like values that must not leak
// into HTTP error responses.
var secretPatterns = []string{
	"sk_live_",
	"sk_test_",
	"ghp_",
	"github_pat_",
	"AKIA",
	"super-secret-password",
}

func containsSecret(body string) bool {
	lower := strings.ToLower(body)
	for _, p := range secretPatterns {
		if strings.Contains(lower, strings.ToLower(p)) {
			return true
		}
	}
	return false
}

func TestErrorResponseDoesNotLeakAPIKey(t *testing.T) {
	h := testHandler()

	// Submit a request whose purpose contains an API key pattern.
	body := `{"purpose":"analyze repo with key sk_live_abc123def456","execution_mode":"analysis_only"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/intelligence/requests", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer fake")
	req.Header.Set("Idempotency-Key", "secret-redaction-1")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	bodyStr := rec.Body.String()
	if containsSecret(bodyStr) {
		t.Errorf("error response leaked secret pattern: %s", bodyStr)
	}
}

func TestErrorResponseDoesNotLeakPassword(t *testing.T) {
	h := testHandler()

	body := `{"purpose":"analyze repo with password super-secret-password","execution_mode":"analysis_only"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/intelligence/requests", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer fake")
	req.Header.Set("Idempotency-Key", "secret-redaction-2")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	bodyStr := rec.Body.String()
	if containsSecret(bodyStr) {
		t.Errorf("error response leaked secret pattern: %s", bodyStr)
	}
}

func TestMissingAuthDoesNotLeakSecrets(t *testing.T) {
	h := testHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/intelligence/requests", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	bodyStr := rec.Body.String()
	if containsSecret(bodyStr) {
		t.Errorf("auth error response leaked secret pattern: %s", bodyStr)
	}
}
