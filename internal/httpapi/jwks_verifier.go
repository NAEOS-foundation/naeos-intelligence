package httpapi

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

const jwksCacheTTL = 5 * time.Minute
const maxJWKSBytes = 1 << 20

type JWKSVerifier struct {
	issuer string
	audience string
	jwksURL string
	client *http.Client
	mu sync.Mutex
	keys map[string]*rsa.PublicKey
	fetchedAt time.Time
}

type jwksDocument struct { Keys []jwk "json:\"keys\"" }
type jwk struct {
	Kty string "json:\"kty\""
	Use string "json:\"use\""
	Alg string "json:\"alg\""
	Kid string "json:\"kid\""
	N string "json:\"n\""
	E string "json:\"e\""
}
type jwtHeader struct { Alg string "json:\"alg\""; Kid string "json:\"kid\"" }
type jwtClaims struct {
	Issuer string "json:\"iss\""
	Subject string "json:\"sub\""
	ClientID string "json:\"client_id\""
	Audience json.RawMessage "json:\"aud\""
	Expires int64 "json:\"exp\""
	NotBefore int64 "json:\"nbf\""
	Scope string "json:\"scope\""
	Scopes []string "json:\"scopes\""
}

func NewJWKSVerifier(issuer, audience, jwksURL string) (*JWKSVerifier, error) {
	issuer, audience, jwksURL = strings.TrimSpace(issuer), strings.TrimSpace(audience), strings.TrimSpace(jwksURL)
	if issuer == "" || audience == "" || jwksURL == "" {
		return nil, errors.New("JWT issuer, audience, and JWKS URL must all be configured")
	}
	if !strings.HasPrefix(issuer, "https://") || !strings.HasPrefix(jwksURL, "https://") {
		return nil, errors.New("JWT issuer and JWKS URL must use HTTPS")
	}
	return &JWKSVerifier{issuer: issuer, audience: audience, jwksURL: jwksURL, client: &http.Client{Timeout: 5 * time.Second}}, nil
}

func (v *JWKSVerifier) Verify(ctx context.Context, token string) (Principal, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 { return Principal{}, errors.New("malformed JWT") }
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil { return Principal{}, errors.New("malformed JWT header") }
	var header jwtHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil || header.Alg != "RS256" || header.Kid == "" {
		return Principal{}, errors.New("unsupported JWT header")
	}
	key, err := v.key(ctx, header.Kid)
	if err != nil { return Principal{}, err }
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil { return Principal{}, errors.New("malformed JWT signature") }
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], signature); err != nil { return Principal{}, errors.New("invalid JWT signature") }
	claimBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil { return Principal{}, errors.New("malformed JWT claims") }
	var claims jwtClaims
	if err := json.Unmarshal(claimBytes, &claims); err != nil { return Principal{}, errors.New("malformed JWT claims") }
	now := time.Now().Unix()
	if claims.Issuer != v.issuer || !audienceContains(claims.Audience, v.audience) || claims.Expires <= now || (claims.NotBefore != 0 && claims.NotBefore > now) {
		return Principal{}, errors.New("JWT claims failed validation")
	}
	clientID := strings.TrimSpace(claims.ClientID)
	if clientID == "" { clientID = strings.TrimSpace(claims.Subject) }
	if clientID == "" { return Principal{}, errors.New("JWT has no client identity") }
	scopes := make(map[string]bool)
	for _, scope := range strings.Fields(claims.Scope) { scopes[scope] = true }
	for _, scope := range claims.Scopes { if strings.TrimSpace(scope) != "" { scopes[strings.TrimSpace(scope)] = true } }
	return Principal{ClientID: clientID, Scopes: scopes}, nil
}

func audienceContains(raw json.RawMessage, expected string) bool {
	var single string
	if json.Unmarshal(raw, &single) == nil { return single == expected }
	var multiple []string
	if json.Unmarshal(raw, &multiple) == nil {
		for _, audience := range multiple { if audience == expected { return true } }
	}
	return false
}

func (v *JWKSVerifier) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if time.Since(v.fetchedAt) < jwksCacheTTL && v.keys != nil {
		if key := v.keys[kid]; key != nil { return key, nil }
		return nil, errors.New("JWT signing key id is not trusted")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil { return nil, fmt.Errorf("create JWKS request: %w", err) }
	resp, err := v.client.Do(req)
	if err != nil { return nil, errors.New("JWKS endpoint unavailable") }
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK { return nil, errors.New("JWKS endpoint returned unsuccessful status") }
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxJWKSBytes+1))
	if err != nil || len(body) > maxJWKSBytes { return nil, errors.New("JWKS response is invalid or too large") }
	var document jwksDocument
	if err := json.Unmarshal(body, &document); err != nil { return nil, errors.New("JWKS response is invalid") }
	keys := make(map[string]*rsa.PublicKey)
	for _, item := range document.Keys {
		if item.Kty != "RSA" || (item.Use != "" && item.Use != "sig") || (item.Alg != "" && item.Alg != "RS256") || item.Kid == "" { continue }
		modulus, errN := base64.RawURLEncoding.DecodeString(item.N)
		exponent, errE := base64.RawURLEncoding.DecodeString(item.E)
		if errN != nil || errE != nil || len(modulus) < 256 || len(exponent) == 0 || len(exponent) > 4 { continue }
		e := 0
		for _, b := range exponent { e = e<<8 | int(b) }
		if e < 3 || e%2 == 0 { continue }
		keys[item.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(modulus), E: e}
	}
	if len(keys) == 0 { return nil, errors.New("JWKS contains no acceptable RS256 signing keys") }
	v.keys, v.fetchedAt = keys, time.Now()
	if key := keys[kid]; key != nil { return key, nil }
	return nil, errors.New("JWT signing key id is not trusted")
}
