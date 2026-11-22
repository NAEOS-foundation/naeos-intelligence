# ADR-005: Supabase Auth as the hosted NAEOS Identity Provider

**Status:** Accepted for IAM-01/02; IAM-04/05/09 accepted for the bounded human-session API by ADR-0004 amendment; remaining activation gates are open  
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

Selected IdP project: `naeos-crm-production` (`oruykkehukoiipldzcvc`, Singapore). The `naeos.db` project (`kqmhpsftijskesjdovjm`, Tokyo) remains the NAEOS Intelligence database and is not the token issuer. This selection keeps CRM's native Auth/Data API trust boundary within one Supabase project while allowing NAEOS Intelligence to verify CRM-issued tokens through the issuer's public JWKS. The exact issuer/audience contract still requires real-token verification under IAM-03. Do not configure production issuer/JWKS environment variables until the signing key, JWKS, token claims, and remaining IAM gates are verified.

## Current implementation evidence

- `internal/httpapi/jwks_verifier.go` validates RS256 signatures against a remote JWKS and checks issuer, audience, expiry, not-before, and key ID.
- `cmd/naeos-intelligence/main.go` reads `NAEOS_AUTH_ISSUER`, `NAEOS_AUTH_AUDIENCE`, and `NAEOS_AUTH_JWKS_URL`.
- `internal/httpapi/auth.go` fails closed when the verifier is not configured and enforces route scopes.
- Protected routes require `intelligence:read` or `intelligence:write`.

## Required follow-up decisions — still open

- IAM-03: exact issuer and audience pinning per environment.
- IAM-04/05: accepted by the 2026-11-22 amendment to ADR-0004 for the human-session API: canonical human principal is verified `iss + sub`; OAuth `client_id` is separate. Machine/service-principal support remains deferred.
- IAM-06/07: accepted single-tenant MVP boundary (ADR-0003); shared tenancy and tenant claims remain deferred.
- IAM-08: authorization vocabulary and trusted scope/role mapping. Supabase user-editable `user_metadata` must never drive authorization; use server-controlled `app_metadata` or a trusted server-side mapping.
- IAM-09: accepted for the read-only human-session API by the 2026-11-22 amendment to ADR-0004; owner key derives from verified `iss + sub`. Legacy records are not automatically reassigned.
- IAM-10: complete claim validation, including required claims and `iat`/clock-skew policy.
- IAM-11: token expiry, session revocation, JWKS cache and key-rotation/revocation response.
- IAM-12/13: accepted for the bounded read-only API by ADR-0004; cross-service propagation remains out of scope.
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

## Deployment contract (selected project; activation still gated)

Candidate values for `naeos-crm-production` (`oruykkehukoiipldzcvc`; not yet approved for deployment):

```env
NAEOS_AUTH_ISSUER=https://oruykkehukoiipldzcvc.supabase.co/auth/v1
NAEOS_AUTH_AUDIENCE=authenticated
NAEOS_AUTH_JWKS_URL=https://oruykkehukoiipldzcvc.supabase.co/auth/v1/.well-known/jwks.json
```

Confirm the actual `iss` and `aud` from a real access token and confirm the live JWKS contains the active RS256 public key before enabling any value. Keep all three unset in production until the signing-key, claim-mapping, and remaining IAM gates pass. The CRM project's own Data API authorization and cross-project Intelligence access are separate trust boundaries.

## Consequences

- Supabase owns user authentication, session lifecycle, refresh tokens, and signing-key custody.
- NAEOS owns API authorization, resource ownership, audit attribution, and fail-closed enforcement.
- NAEOS Intelligence can verify tokens locally using public JWKS without receiving Supabase private signing material.
- Cross-project CRM database access is not automatically granted by a token from another Supabase project. Any cross-project access must use an explicitly designed backend trust boundary rather than assuming PostgREST will accept a foreign project's JWT.
