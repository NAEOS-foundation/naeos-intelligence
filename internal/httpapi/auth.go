package httpapi

import (
	"context"
	"net/http"
	"strings"
)

type Principal struct { ClientID string; Scopes map[string]bool }

type principalKey struct{}

func withPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, principal)
}

func principalFrom(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(Principal); return p, ok && p.ClientID != ""
}

// TokenVerifier is the replaceable JWT verification boundary. Implementations
// must validate signature, allowlisted algorithm, issuer, audience, expiry,
// not-before, and trusted JWKS key material.
type TokenVerifier interface { Verify(ctx context.Context, token string) (Principal, error) }

type Authenticator struct { Verifier TokenVerifier }

func (a Authenticator) Middleware(requiredScope string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.Verifier == nil { writeProblem(w, r, http.StatusServiceUnavailable, "AUTHENTICATION_NOT_CONFIGURED", "Authentication is not configured"); return }
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") || strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")) == "" {
			writeProblem(w, r, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "A valid bearer token is required"); return
		}
		principal, err := a.Verifier.Verify(r.Context(), strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")))
		if err != nil || principal.ClientID == "" {
			writeProblem(w, r, http.StatusUnauthorized, "INVALID_TOKEN", "Bearer token is invalid"); return
		}
		if requiredScope != "" && !principal.Scopes[requiredScope] {
			writeProblem(w, r, http.StatusForbidden, "INSUFFICIENT_SCOPE", "Client is not authorized for this operation"); return
		}
		next.ServeHTTP(w, r.WithContext(withPrincipal(r.Context(), principal)))
	})
}
