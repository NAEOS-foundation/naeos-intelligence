package httpapi

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestJWKSVerifierAcceptsValidRS256AndExtractsScopes(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	const kid = "test-key-01"
	exponent := []byte{1, 0, 1}
	jwks := map[string]any{
		"keys": []map[string]string{
			{
				"kty": "RSA",
				"use": "sig",
				"alg": "RS256",
				"kid": kid,
				"n":   base64.RawURLEncoding.EncodeToString(privateKey.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(exponent),
			},
		},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(jwks)
	}))
	defer server.Close()

	verifier := &JWKSVerifier{
		issuer:   "https://issuer.test",
		audience: "naeos-api",
		jwksURL:  server.URL,
		client:   server.Client(),
	}
	now := time.Now().Unix()
	claims := map[string]any{
		"iss":       verifier.issuer,
		"aud":       verifier.audience,
		"client_id": "svc-crm",
		"exp":       now + 300,
		"nbf":       now - 10,
		"scope":     "intelligence:read intelligence:write",
	}
	token := signedTestJWT(t, privateKey, kid, claims)
	principal, err := verifier.Verify(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	if principal.ClientID != "svc-crm" || !principal.Scopes["intelligence:read"] || !principal.Scopes["intelligence:write"] {
		t.Fatalf("unexpected principal: %+v", principal)
	}
}

func TestJWKSVerifierRejectsBadClaimsAndAlgorithm(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	verifier := &JWKSVerifier{issuer: "https://issuer.test", audience: "naeos-api", keys: map[string]*rsa.PublicKey{"kid": &privateKey.PublicKey}, fetchedAt: time.Now()}
	expired := signedTestJWT(t, privateKey, "kid", map[string]any{
		"iss": verifier.issuer,
		"aud": verifier.audience,
		"sub": "svc",
		"exp": time.Now().Unix() - 1,
	})
	if _, err := verifier.Verify(context.Background(), expired); err == nil {
		t.Fatal("expired JWT must be rejected")
	}
	noneHeader, _ := json.Marshal(map[string]string{"alg": "none", "kid": "kid"})
	noneToken := base64.RawURLEncoding.EncodeToString(noneHeader) + "." +
		base64.RawURLEncoding.EncodeToString([]byte("{}")) + "."
	if _, err := verifier.Verify(context.Background(), noneToken); err == nil {
		t.Fatal("non-RS256 token must be rejected")
	}
}

func TestNewJWKSVerifierRequiresHTTPSAndCompleteConfig(t *testing.T) {
	if _, err := NewJWKSVerifier("https://issuer", "aud", "http://jwks"); err == nil {
		t.Fatal("HTTP JWKS URL must be rejected")
	}
	if _, err := NewJWKSVerifier("https://issuer", "", "https://jwks"); err == nil {
		t.Fatal("incomplete verifier configuration must be rejected")
	}
}


func TestJWKSVerifierAcceptsAudienceArrayAndRejectsFutureNotBefore(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil { t.Fatal(err) }
	verifier := &JWKSVerifier{
		issuer: "https://issuer.test",
		audience: "naeos-api",
		keys: map[string]*rsa.PublicKey{"kid": &privateKey.PublicKey},
		fetchedAt: time.Now(),
	}
	now := time.Now().Unix()
	valid := signedTestJWT(t, privateKey, "kid", map[string]any{
		"iss": verifier.issuer, "aud": []string{"other-api", verifier.audience},
		"sub": "svc-test", "exp": now + 300, "nbf": now - 1,
	})
	if _, err := verifier.Verify(context.Background(), valid); err != nil {
		t.Fatalf("JWT with matching audience in array should be accepted: %v", err)
	}
	future := signedTestJWT(t, privateKey, "kid", map[string]any{
		"iss": verifier.issuer, "aud": verifier.audience,
		"sub": "svc-test", "exp": now + 300, "nbf": now + 60,
	})
	if _, err := verifier.Verify(context.Background(), future); err == nil {
		t.Fatal("JWT with future not-before must be rejected")
	}
}

func TestJWKSVerifierRejectsTamperedSignatureAndUnknownKeyID(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil { t.Fatal(err) }
	const kid = "kid"
	jwks := map[string]any{
		"keys": []map[string]string{
			{
				"kty": "RSA",
				"use": "sig",
				"alg": "RS256",
				"kid": kid,
				"n":   base64.RawURLEncoding.EncodeToString(privateKey.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1}),
			},
		},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(jwks)
	}))
	defer server.Close()

	verifier := &JWKSVerifier{
		issuer:    "https://issuer.test",
		audience:  "naeos-api",
		jwksURL:   server.URL,
		client:    server.Client(),
		keys:      map[string]*rsa.PublicKey{kid: &privateKey.PublicKey},
		fetchedAt: time.Now(),
	}
	token := signedTestJWT(t, privateKey, "kid", map[string]any{
		"iss": verifier.issuer, "aud": verifier.audience,
		"sub": "svc-test", "exp": time.Now().Unix() + 300,
	})
	parts := strings.Split(token, ".")
	parts[1] = base64.RawURLEncoding.EncodeToString([]byte(`{"iss":"https://issuer.test","aud":"naeos-api","sub":"attacker","exp":9999999999}`))
	tampered := strings.Join(parts, ".")
	if _, err := verifier.Verify(context.Background(), tampered); err == nil {
		t.Fatal("JWT with tampered claims and original signature must be rejected")
	}
	unknown := signedTestJWT(t, privateKey, "unknown-kid", map[string]any{
		"iss": verifier.issuer, "aud": verifier.audience,
		"sub": "svc-test", "exp": time.Now().Unix() + 300,
	})
	if _, err := verifier.Verify(context.Background(), unknown); err == nil {
		t.Fatal("JWT with untrusted key id must be rejected")
	}
}

func signedTestJWT(t *testing.T, key *rsa.PrivateKey, kid string, claims any) string {
	t.Helper()
	header, err := json.Marshal(map[string]string{"alg": "RS256", "kid": kid, "typ": "JWT"})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	encodedHeader := base64.RawURLEncoding.EncodeToString(header)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signingInput := encodedHeader + "." + encodedPayload
	digest := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%s.%s", signingInput, base64.RawURLEncoding.EncodeToString(signature))
}


func TestJWKSVerifierRejectsMissingOrMalformedRequiredClaims(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	verifier := &JWKSVerifier{
		issuer:   "https://issuer.test",
		audience: "naeos-api",
		keys:     map[string]*rsa.PublicKey{"kid": &privateKey.PublicKey},
		fetchedAt: time.Now(),
	}
	base := map[string]any{
		"iss": verifier.issuer,
		"aud": verifier.audience,
		"sub": "svc-test",
		"exp": time.Now().Unix() + 300,
	}
	tests := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{name: "missing expiry", mutate: func(c map[string]any) { delete(c, "exp") }},
		{name: "missing issuer", mutate: func(c map[string]any) { delete(c, "iss") }},
		{name: "wrong issuer", mutate: func(c map[string]any) { c["iss"] = "https://attacker.test" }},
		{name: "wrong audience", mutate: func(c map[string]any) { c["aud"] = []string{"other-api"} }},
		{name: "missing identity", mutate: func(c map[string]any) { delete(c, "sub") }},
		{name: "blank identity", mutate: func(c map[string]any) { c["sub"] = "  " }},
		{name: "malformed not-before", mutate: func(c map[string]any) { c["nbf"] = "not-a-timestamp" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := make(map[string]any, len(base))
			for key, value := range base {
				claims[key] = value
			}
			tt.mutate(claims)
			token := signedTestJWT(t, privateKey, "kid", claims)
			if _, err := verifier.Verify(context.Background(), token); err == nil {
				t.Fatal("JWT with invalid required claims must be rejected")
			}
		})
	}
}
