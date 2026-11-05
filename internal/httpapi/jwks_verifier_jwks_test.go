package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// TestJWKSVerifierRefreshesWhenSigningKeyIDIsUnknown verifies that a cached
// JWKS is refreshed immediately when a token references a newly rotated key.
func TestJWKSVerifierRefreshesWhenSigningKeyIDIsUnknown(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := requests.Add(1)
		kid := "key-1"
		if count > 1 {
			kid = "key-2"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]string{{
				"kty": "RSA",
				"use": "sig",
				"alg": "RS256",
				"kid": kid,
				"n": strings.Repeat("A", 350),
				"e": "AQAB",
			}},
		})
	}))
	defer server.Close()

	verifier := &JWKSVerifier{
		issuer: "https://issuer.test",
		audience: "naeos-api",
		jwksURL: server.URL,
		client: server.Client(),
	}

	if _, err := verifier.key(context.Background(), "key-1"); err != nil {
		t.Fatalf("initial key fetch failed: %v", err)
	}
	if _, err := verifier.key(context.Background(), "key-2"); err != nil {
		t.Fatalf("unknown key ID should trigger JWKS refresh: %v", err)
	}
	if got := requests.Load(); got != 2 {
		t.Fatalf("expected 2 JWKS requests after key rotation, got %d", got)
	}
}
