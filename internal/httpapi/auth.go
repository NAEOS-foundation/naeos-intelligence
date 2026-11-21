package httpapi

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"
)

type Principal struct {
	// Issuer and Subject identify the authenticated human principal. For hosted
	// Supabase Auth sessions, storage ownership is derived from this pair.
	Issuer  string
	Subject string

	// ClientID remains separate for OAuth/service-client identity and backward
	// compatibility with existing client-only tokens.
	ClientID string
	Scopes   map[string]bool
}

// ApproverID returns the human subject from the verified issuer/subject pair.
// The request body must never be trusted to identify the approval actor.
func (p Principal) ApproverID() string {
	if strings.TrimSpace(p.Issuer) == "" || strings.TrimSpace(p.Subject) == "" {
		return ""
	}
	return p.Subject
}

func (p Principal) OwnerID() string {
	subject := p.Subject
	issuer := p.Issuer
	if strings.TrimSpace(subject) != "" {
		if strings.TrimSpace(issuer) == "" {
			return ""
		}
		return "user:" + base64.RawURLEncoding.EncodeToString([]byte(issuer)) + ":" + base64.RawURLEncoding.EncodeToString([]byte(subject))
	}
	// Preserve existing storage ownership keys for client-only tokens.
	return strings.TrimSpace(p.ClientID)
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
