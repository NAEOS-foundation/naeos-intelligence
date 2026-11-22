package httpapi

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"
)

type Principal struct {
	// Issuer and Subject form the canonical human identity. OAuth client identity
	// remains separate and is never substituted for a missing human subject.
	Issuer   string
	Subject  string
	ClientID string
	Scopes   map[string]bool
}

// OwnerID returns a stable, issuer-qualified storage owner key. The existing
// persistence column is retained for compatibility; it stores this opaque key,
// not the OAuth client_id claim. Human-only identity is required in this phase.
func (p Principal) OwnerID() string {
	if strings.TrimSpace(p.Issuer) == "" || strings.TrimSpace(p.Subject) == "" {
		return ""
	}
	owner := "user:" + base64.RawURLEncoding.EncodeToString([]byte(p.Issuer)) + ":" +
		base64.RawURLEncoding.EncodeToString([]byte(p.Subject))
	if len(owner) > 255 {
		return ""
	}
	return owner
}

type principalKey struct{}

func withPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, principal)
}

func principalFrom(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(Principal)
	return p, ok && p.OwnerID() != ""
}

// TokenVerifier is the replaceable JWT verification boundary. Implementations
// must validate signature, allowlisted algorithm, issuer, audience, expiry,
// not-before, and trusted JWKS key material.
type TokenVerifier interface {
	Verify(ctx context.Context, token string) (Principal, error)
}

type Authenticator struct{ Verifier TokenVerifier }

func (a Authenticator) Middleware(requiredScope string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.Verifier == nil {
			writeProblem(w, r, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Authentication is not configured")
			return
		}
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") || strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")) == "" {
			writeProblem(w, r, http.StatusUnauthorized, "UNAUTHENTICATED", "A valid bearer token is required")
			return
		}
		principal, err := a.Verifier.Verify(r.Context(), strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")))
		if err != nil || principal.OwnerID() == "" {
			writeProblem(w, r, http.StatusUnauthorized, "UNAUTHENTICATED", "Bearer token is invalid")
			return
		}
		if requiredScope != "" && !principal.Scopes[requiredScope] {
			writeProblem(w, r, http.StatusForbidden, "FORBIDDEN", "Client is not authorized for this operation")
			return
		}
		next.ServeHTTP(w, r.WithContext(withPrincipal(r.Context(), principal)))
	})
}
