package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestJWKSVerifierRejectsDocumentsWithoutUsableSigningKeys exercises the
// filtering rules applied to untrusted JWK documents before a key is trusted.
func TestJWKSVerifierRejectsDocumentsWithoutUsableSigningKeys(t *testing.T) {
	tests := []struct {
		name string
		keys []map[string]string
	}{
		{
			name: "non RSA key",
			keys: []map[string]string{{"kty": "EC", "use": "sig", "alg": "RS256", "kid": "key-1", "n": strings.Repeat("A", 350), "e": "AQAB"}},
		},
		{
			name: "encryption key",
			keys: []map[string]string{{"kty": "RSA", "use": "enc", "alg": "RS256", "kid": "key-1", "n": strings.Repeat("A", 350), "e": "AQAB"}},
		},
		{
			name: "wrong algorithm",
			keys: []map[string]string{{"kty": "RSA", "use": "sig", "alg": "RS384", "kid": "key-1", "n": strings.Repeat("A", 350), "e": "AQAB"}},
		},
		{
			name: "missing key id",
			keys: []map[string]string{{"kty": "RSA", "use": "sig", "alg": "RS256", "n": strings.Repeat("A", 350), "e": "AQAB"}},
		},
		{
			name: "weak modulus",
			keys: []map[string]string{{"kty": "RSA", "use": "sig", "alg": "RS256", "kid": "key-1", "n": "AQ", "e": "AQAB"}},
		},
		{
			name: "invalid exponent",
			keys: []map[string]string{{"kty": "RSA", "use": "sig", "alg": "RS256", "kid": "key-1", "n": strings.Repeat("A", 350), "e": "Ag"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(map[string]any{"keys": tt.keys})
			}))
			defer server.Close()

			verifier := &JWKSVerifier{
				issuer: "https://issuer.test",
				audience: "naeos-api",
				jwksURL: server.URL,
				client: server.Client(),
			}
			if _, err := verifier.key(context.Background(), "key-1"); err == nil {
				t.Fatal("JWKS containing no acceptable signing keys must be rejected")
			}
		})
	}
}

func TestJWKSVerifierRejectsUnsuccessfulAndMalformedJWKSResponses(t *testing.T) {
	tests := []struct {
		name string
		status int
		body string
	}{
		{name: "HTTP error", status: http.StatusServiceUnavailable, body: ""},
		{name: "malformed JSON", status: http.StatusOK, body: "{not-json"},
		{name: "missing keys member", status: http.StatusOK, body: "{}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			verifier := &JWKSVerifier{
				issuer: "https://issuer.test",
				audience: "naeos-api",
				jwksURL: server.URL,
				client: server.Client(),
			}
			if _, err := verifier.key(context.Background(), "key-1"); err == nil {
				t.Fatal("unsuccessful or malformed JWKS response must be rejected")
			}
		})
	}
}

func TestJWKSVerifierRejectsOversizedJWKSResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat(" ", maxJWKSBytes+1)))
	}))
	defer server.Close()

	verifier := &JWKSVerifier{
		issuer: "https://issuer.test",
		audience: "naeos-api",
		jwksURL: server.URL,
		client: server.Client(),
	}
	if _, err := verifier.key(context.Background(), "key-1"); err == nil {
		t.Fatal("oversized JWKS response must be rejected")
	}
}
