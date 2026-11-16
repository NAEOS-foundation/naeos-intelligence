package httpapi

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
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

const (
	jwksCacheTTL = 5 * time.Minute
	maxJWKSBytes = 1 << 20
)

type JWKSVerifier struct {
	issuer    string
	audience  string
	jwksURL   string
	client    *http.Client
	mu        sync.Mutex
	keys      map[string]*ecdsa.PublicKey
	fetchedAt time.Time
}

type jwksDocument struct {
	Keys []jwk `json:"keys"`
}

type jwk struct {
	Kty     string   `json:"kty"`
	Use     string   `json:"use"`
	Alg     string   `json:"alg"`
	Kid     string   `json:"kid"`
	Crv     string   `json:"crv"`
	X       string   `json:"x"`
	Y       string   `json:"y"`
	KeyOps  []string `json:"key_ops"`
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Kid string `json:"kid"`
}

type jwtClaims struct {
	Issuer    string          `json:"iss"`
	Subject   string          `json:"sub"`
	ClientID  string          `json:"client_id"`
	Audience  json.RawMessage `json:"aud"`
	Expires   int64           `json:"exp"`
	NotBefore int64           `json:"nbf"`
	Scope     string          `json:"scope"`
	Scopes    []string        `json:"scopes"`
}

func NewJWKSVerifier(issuer, audience, jwksURL string) (*JWKSVerifier, error) {
	issuer = strings.TrimSpace(issuer)
	audience = strings.TrimSpace(audience)
	jwksURL = strings.TrimSpace(jwksURL)
	if issuer == "" || audience == "" || jwksURL == "" {
		return nil, errors.New("JWT issuer, audience, and JWKS URL must all be configured")
	}
	if !strings.HasPrefix(issuer, "https://") || !strings.HasPrefix(jwksURL, "https://") {
		return nil, errors.New("JWT issuer and JWKS URL must use HTTPS")
	}
	return &JWKSVerifier{
		issuer:   issuer,
		audience: audience,
		jwksURL:  jwksURL,
		client:   &http.Client{Timeout: 5 * time.Second},
	}, nil
}

func (v *JWKSVerifier) Verify(ctx context.Context, token string) (Principal, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Principal{}, errors.New("malformed JWT")
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Principal{}, errors.New("malformed JWT header")
	}
	var header jwtHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil || header.Alg != "ES256" || header.Kid == "" {
		return Principal{}, errors.New("unsupported JWT header")
	}
	key, err := v.key(ctx, header.Kid)
	if err != nil {
		return Principal{}, err
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(signature) != 64 {
		return Principal{}, errors.New("malformed ES256 JWT signature")
	}
	r := new(big.Int).SetBytes(signature[:32])
	s := new(big.Int).SetBytes(signature[32:])
	if r.Sign() <= 0 || s.Sign() <= 0 || r.Cmp(key.Params().N) >= 0 || s.Cmp(key.Params().N) >= 0 {
		return Principal{}, errors.New("invalid ES256 JWT signature")
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if !ecdsa.Verify(key, digest[:], r, s) {
		return Principal{}, errors.New("invalid JWT signature")
	}
	claimBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Principal{}, errors.New("malformed JWT claims")
	}
	var claims jwtClaims
	if err := json.Unmarshal(claimBytes, &claims); err != nil {
		return Principal{}, errors.New("malformed JWT claims")
	}
	now := time.Now().Unix()
	if claims.Issuer != v.issuer ||
		!audienceContains(claims.Audience, v.audience) ||
		claims.Expires <= now ||
		(claims.NotBefore != 0 && claims.NotBefore > now) {
		return Principal{}, errors.New("JWT claims failed validation")
	}
	clientID := strings.TrimSpace(claims.ClientID)
	if clientID == "" {
		clientID = strings.TrimSpace(claims.Subject)
	}
	if clientID == "" {
		return Principal{}, errors.New("JWT has no client identity")
	}
	scopes := make(map[string]bool)
	for _, scope := range strings.Fields(claims.Scope) {
		scopes[scope] = true
	}
	for _, scope := range claims.Scopes {
		if strings.TrimSpace(scope) != "" {
			scopes[strings.TrimSpace(scope)] = true
		}
	}
	return Principal{ClientID: clientID, Scopes: scopes}, nil
}

func audienceContains(raw json.RawMessage, expected string) bool {
	var single string
	if json.Unmarshal(raw, &single) == nil {
		return single == expected
	}
	var multiple []string
	if json.Unmarshal(raw, &multiple) == nil {
		for _, audience := range multiple {
			if audience == expected {
				return true
			}
		}
	}
	return false
}

func (v *JWKSVerifier) key(ctx context.Context, kid string) (*ecdsa.PublicKey, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if time.Since(v.fetchedAt) < jwksCacheTTL && v.keys != nil {
		if key := v.keys[kid]; key != nil {
			return key, nil
		}
		// An unknown kid can indicate key rotation. Refresh the trusted JWKS
		// immediately instead of rejecting the token until the cache expires.
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create JWKS request: %w", err)
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return nil, errors.New("JWKS endpoint unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("JWKS endpoint returned unsuccessful status")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxJWKSBytes+1))
	if err != nil || len(body) > maxJWKSBytes {
		return nil, errors.New("JWKS response is invalid or too large")
	}
	var document jwksDocument
	if err := json.Unmarshal(body, &document); err != nil {
		return nil, errors.New("JWKS response is invalid")
	}
	keys := make(map[string]*ecdsa.PublicKey)
	for _, item := range document.Keys {
		if item.Kty != "EC" ||
			item.Crv != "P-256" ||
			(item.Use != "" && item.Use != "sig") ||
			(item.Alg != "" && item.Alg != "ES256") ||
			item.Kid == "" ||
			(len(item.KeyOps) > 0 && !containsKeyOp(item.KeyOps, "verify")) {
			continue
		}
		xBytes, errX := base64.RawURLEncoding.DecodeString(item.X)
		yBytes, errY := base64.RawURLEncoding.DecodeString(item.Y)
		if errX != nil || errY != nil || len(xBytes) != 32 || len(yBytes) != 32 {
			continue
		}
		x := new(big.Int).SetBytes(xBytes)
		y := new(big.Int).SetBytes(yBytes)
		curve := elliptic.P256()
		if !curve.IsOnCurve(x, y) {
			continue
		}
		if _, duplicate := keys[item.Kid]; duplicate {
			return nil, errors.New("JWKS contains duplicate signing key ids")
		}
		keys[item.Kid] = &ecdsa.PublicKey{Curve: curve, X: x, Y: y}
	}
	if len(keys) == 0 {
		return nil, errors.New("JWKS contains no acceptable ES256 signing keys")
	}
	v.keys, v.fetchedAt = keys, time.Now()
	if key := keys[kid]; key != nil {
		return key, nil
	}
	return nil, errors.New("JWT signing key id is not trusted")
}

func containsKeyOp(ops []string, expected string) bool {
	for _, op := range ops {
		if op == expected {
			return true
		}
	}
	return false
}
