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
	jwtClockSkew = 60 * time.Second
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
	Alg  string   `json:"alg"`
	Kid  string   `json:"kid"`
	Crit []string `json:"crit"`
	B64  *bool    `json:"b64"`
}

type jwtClaims struct {
	Issuer    string          `json:"iss"`
	Subject   string          `json:"sub"`
	ClientID  string          `json:"client_id"`
	Audience  json.RawMessage `json:"aud"`
	Expires   int64           `json:"exp"`
	IssuedAt  int64           `json:"iat"`
	NotBefore int64           `json:"nbf"`
	NAEOSScopes []string        `json:"naeos_scopes"`
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
	client := &http.Client{Timeout: 5 * time.Second}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 || len(via) == 0 || req.URL.Scheme != "https" || !strings.EqualFold(req.URL.Host, via[0].URL.Host) {
			return http.ErrUseLastResponse
		}
		return nil
	}
	return &JWKSVerifier{
		issuer:   issuer,
		audience: audience,
		jwksURL:  jwksURL,
		client:   client,
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
	if err := json.Unmarshal(headerBytes, &header); err != nil || header.Alg != "ES256" || header.Kid == "" || len(header.Crit) != 0 || (header.B64 != nil && !*header.B64) {
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
	if r.Sign() <= 0 || s.Sign() <= 0 || r.Cmp(key.Curve.Params().N) >= 0 || s.Cmp(key.Curve.Params().N) >= 0 {
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
	now := time.Now()
	nowUnix := now.Unix()
	skew := int64(jwtClockSkew.Seconds())
	if claims.Issuer != v.issuer ||
		!audienceContains(claims.Audience, v.audience) ||
		claims.Subject == "" ||
		claims.Expires == 0 ||
		claims.IssuedAt == 0 ||
		claims.Expires <= nowUnix-skew ||
		claims.IssuedAt > nowUnix+skew ||
		claims.IssuedAt >= claims.Expires ||
		(claims.NotBefore != 0 && claims.NotBefore > nowUnix+skew) ||
		(claims.NotBefore != 0 && claims.NotBefore >= claims.Expires) {
		return Principal{}, errors.New("JWT claims failed validation")
	}
	// IAM-04/05: Supabase Auth human-user identity is the verified issuer
	// namespace plus subject. Do not substitute OAuth client_id for sub:
	// they represent different identity domains. ClientID is retained in the
	// existing API model as the ownership key, but is populated only from sub.
	if strings.TrimSpace(claims.Subject) == "" {
		return Principal{}, errors.New("JWT has no subject identity")
	}
	clientID := strings.TrimSpace(claims.ClientID)
	scopes := make(map[string]bool)
	// IAM-08: accept only the dedicated capability claim produced by the
	// configured server-controlled Supabase Auth hook. Generic scope/scopes
	// claims and user_metadata are not authorization sources.
	for _, scope := range claims.NAEOSScopes {
		switch strings.TrimSpace(scope) {
		case "intelligence:read", "intelligence:write":
			scopes[strings.TrimSpace(scope)] = true
		}
	}
	return Principal{Issuer: claims.Issuer, Subject: claims.Subject, ClientID: clientID, Scopes: scopes}, nil
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
