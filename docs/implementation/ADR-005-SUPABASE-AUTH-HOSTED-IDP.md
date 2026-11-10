# ADR-005: Supabase Auth as the hosted NAEOS Identity Provider

**Status:** Accepted for IAM-01 and IAM-02; deployment project and remaining IAM decisions are open  
**Date:** 2026-11-11  
**Decision owner:** NAEOS maintainer  
**Related:** `PHASE-05-IAM-DECISION-WORKSHEET.md`, `PHASE-05-IAM-MVP-DECISION-BRIEF.md`

## Context

NAEOS Intelligence already has a provider-neutral `TokenVerifier` boundary and an RS256-only JWKS verifier. The repository does not establish a complete first-party token issuer. The maintainer selected a hosted identity provider and explicitly chose Supabase.

Supabase Auth issues session access JWTs and exposes asymmetric public signing keys through its project JWKS discovery endpoint. Supabase signing keys support RS256 and key rotation. Supabase's documentation currently recommends considering ES256 for efficiency, but this ADR records RS256 as the NAEOS compatibility requirement and verifier allowlist for this integration.

## Decisions

### IAM-01 — Protocol and token format

Accepted: use Supabase Auth session access JWTs as bearer credentials for human-user access to NAEOS APIs. Accept RS256 only at the NAEOS Intelligence verifier boundary. Do not accept HS256 or algorithm negotiation.

This decision does not authorize using Supabase API keys as user access tokens. Publishable keys identify the client application; they are not substitutes for a user's access JWT. Secret keys remain server-only and must never be used as user bearer tokens.

### IAM-02 — Identity provider

Accepted: use hosted Supabase Auth. Do not build a custom NAEOS token issuer or manage a NAEOS-owned private signing key for this integration.

The exact Supabase project and issuer URL are intentionally not selected by this ADR. Project selection must account for CRM's native Auth/Data API trust boundary and the separate `naeos.db` Intelligence database. Do not configure production issuer/JWKS environment variables until the project decision is recorded.

## Current implementation evidence

- `internal/httpapi/jwks_verifier.go` validates RS256 signatures against a remote JWKS and checks issuer, audience, expiry, not-before, and key ID.
- `cmd/naeos-intelligence/main.go` reads `NAEOS_AUTH_ISSUER`, `NAEOS_AUTH_AUDIENCE`, and `NAEOS_AUTH_JWKS_URL`.
- `internal/httpapi/auth.go` fails closed when the verifier is not configured and enforces route scopes.
- Protected routes require `intelligence:read` or `intelligence:write`.

## Required follow-up decisions — still open

- IAM-03: exact issuer and audience pinning per environment.
- IAM-04: canonical principal key, including issuer namespace and subject.
- IAM-05: distinction between human principal and OAuth/service client. Supabase Auth session JWTs are not a client-credentials grant.
- IAM-06/07: single-tenant pilot or tenant model and trusted tenant binding.
- IAM-08: authorization vocabulary and trusted scope/role mapping. Supabase user-editable `user_metadata` must never drive authorization; use server-controlled `app_metadata` or a trusted server-side mapping.
- IAM-09: persisted resource ownership semantics.
- IAM-10: complete claim validation, including required claims and `iat`/clock-skew policy.
- IAM-11: token expiry, session revocation, JWKS cache and key-rotation/revocation response.
- IAM-12: trusted identity propagation.
- IAM-13: 401/403/404 denial and non-disclosure behavior.
- IAM-14: audit attribution and retention.

## Security gates

1. Keep production authentication fail-closed. Do not disable middleware or remove scope checks to make integration tests pass.
2. Do not assume the project's active signing algorithm from its creation date. Confirm the active signing key is RSA/RS256 in Supabase Dashboard and confirm the JWKS contains a matching RSA public key.
3. Verify a real access JWT from the selected project: `alg=RS256`, non-empty `kid`, exact `iss`, expected `aud`, `sub`, `exp`, and `nbf` where present.
4. Do not deploy until scope claims or trusted authorization mapping are decided and tested. Supabase's default session JWT does not automatically grant NAEOS-specific API scopes.
5. Do not use `user_metadata` for roles, scopes, tenant IDs, or any authorization decision.
6. Keep Supabase secret keys out of browser code, logs, source control, and this repository.
7. Require CI evidence for invalid signature, unsupported algorithm, unknown key, wrong issuer/audience, expiry, future not-before, missing identity, insufficient scope, JWKS outage, key rotation, and resource ownership.
8. No production database schema changes are authorized by this ADR.

## Deployment contract (after project selection)

For the selected project:

```env
NAEOS_AUTH_ISSUER=https://<PROJECT_REF>.supabase.co/auth/v1
NAEOS_AUTH_AUDIENCE=authenticated
NAEOS_AUTH_JWKS_URL=https://<PROJECT_REF>.supabase.co/auth/v1/.well-known/jwks.json
```

These values are a proposed mapping only. Confirm `iss` and `aud` from an actual access token before enabling the configuration. Keep all three unset in production until the project and claims mapping pass the gates above.

## Consequences

- Supabase owns user authentication, session lifecycle, refresh tokens, and signing-key custody.
- NAEOS owns API authorization, resource ownership, audit attribution, and fail-closed enforcement.
- NAEOS Intelligence can verify tokens locally using public JWKS without receiving Supabase private signing material.
- Cross-project CRM database access is not automatically granted by a token from another Supabase project. Any cross-project access must use an explicitly designed backend trust boundary rather than assuming PostgREST will accept a foreign project's JWT.
