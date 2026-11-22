# ADR-005: Supabase Auth as the hosted NAEOS Identity Provider

**Status:** Accepted for IAM-01/02 and IAM-08 capability policy; IAM-04/05/09 accepted for the bounded human-session API by ADR-0004 amendment; activation evidence remains gated  
**Date:** 2026-11-11  
**Decision owner:** NAEOS maintainer  
**Related:** `PHASE-05-IAM-DECISION-WORKSHEET.md`, `PHASE-05-IAM-MVP-DECISION-BRIEF.md`

## Context

NAEOS Intelligence already has a provider-neutral `TokenVerifier` boundary and an ES256-only JWKS verifier. The repository does not establish a complete first-party token issuer. The maintainer selected a hosted identity provider and explicitly chose Supabase.

Supabase Auth issues session access JWTs and exposes asymmetric public signing keys through its project JWKS discovery endpoint. Supabase supports asymmetric ES256 (P-256) signing keys and publishes their public JWKs for local verification. The maintainer decision recorded in the Phase 2 IAM implementation branch is ES256-only verification; live token/JWKS metadata still must be verified before activation.

## Decisions

### IAM-01 — Protocol and token format

Accepted: use Supabase Auth session access JWTs as bearer credentials for human-user access to NAEOS APIs. Accept ES256 only (ECDSA P-256 with SHA-256) at the NAEOS Intelligence verifier boundary. Do not accept RS256, HS256, `none`, or algorithm negotiation.

This decision does not authorize using Supabase API keys as user access tokens. Publishable keys identify the client application; they are not substitutes for a user's access JWT. Secret keys remain server-only and must never be used as user bearer tokens.

### IAM-02 — Identity provider

Accepted: use hosted Supabase Auth. Do not build a custom NAEOS token issuer or manage a NAEOS-owned private signing key for this integration.

Selected IdP project: `naeos-crm-production` (`oruykkehukoiipldzcvc`, Singapore). The `naeos.db` project (`kqmhpsftijskesjdovjm`, Tokyo) remains the NAEOS Intelligence database and is not the token issuer. This selection keeps CRM's native Auth/Data API trust boundary within one Supabase project while allowing NAEOS Intelligence to verify CRM-issued tokens through the issuer's public JWKS. The exact issuer/audience contract still requires real-token verification under IAM-03. Do not configure production issuer/JWKS environment variables until the signing key, JWKS, token claims, and remaining IAM gates are verified.

## Current implementation evidence

- `internal/httpapi/jwks_verifier.go` validates ES256 signatures against P-256 public keys from a remote JWKS and checks issuer, audience, expiry, not-before, issue time, and key ID; requires `iss`, matching `aud`, non-empty `sub`, `exp`, and `iat`, with 60-second clock skew.
- `cmd/naeos-intelligence/main.go` reads `NAEOS_AUTH_ISSUER`, `NAEOS_AUTH_AUDIENCE`, and `NAEOS_AUTH_JWKS_URL`.
- `internal/httpapi/auth.go` fails closed when the verifier is not configured and enforces route scopes.
- Protected routes require `intelligence:read` or `intelligence:write`.

## Required follow-up decisions — still open

- IAM-03: exact issuer and audience pinning per environment.
- IAM-04/05: accepted by the 2026-11-22 amendment to ADR-0004 for the human-session API: canonical human principal is verified `iss + sub`; OAuth `client_id` is separate. Machine/service-principal support remains deferred.
- IAM-06/07: accepted single-tenant MVP boundary (ADR-0003); shared tenancy and tenant claims remain deferred.
- IAM-08: accepted policy — Supabase Custom Access Token Hook emits dedicated `naeos_scopes` from administrator-managed `app_metadata.naeos_scopes`; NAEOS allowlists only `intelligence:read` and `intelligence:write`, ignores generic `scope`/`scopes` and `user_metadata`. SQL reference: `docs/implementation/supabase/NAEOS-CUSTOM-ACCESS-TOKEN-HOOK.sql`. Hook activation and live issued-token evidence remain unverified and gated.
- IAM-09: accepted for the read-only human-session API by the 2026-11-22 amendment to ADR-0004; owner key derives from verified `iss + sub`. Legacy records are not automatically reassigned.
- IAM-10: baseline implementation requires `iss`, matching `aud`, non-empty `sub`, `exp`, and `iat`; rejects `iat >= exp` and applies 60-second clock skew. A maximum `exp - iat` lifetime policy is not yet enforced; maintainer acceptance and CI evidence remain open.
- IAM-11: baseline uses a 5-minute verifier JWKS cache, immediate refresh for unknown `kid`, and fails closed without stale-cache fallback. No authoritative online session/version revocation lookup is implemented; JWTs can remain valid until `exp` after sign-out. Key compromise propagation is bounded by Supabase's edge/cache layers and needs an explicit operational response.
- IAM-12/13: accepted for the bounded read-only API by ADR-0004; cross-service propagation remains out of scope.
- IAM-14: audit attribution and retention.

## Security gates

1. Keep production authentication fail-closed. Do not disable middleware or remove scope checks to make integration tests pass.
2. Do not assume the project's active signing algorithm from its creation date. Confirm the active signing key is EC P-256/ES256 in Supabase Dashboard and confirm the JWKS contains a matching EC public key.
3. Verify a real access JWT from the selected project: `alg=ES256`, non-empty `kid`, exact `iss`, expected `aud`, `sub`, `exp`, and `nbf` where present.
4. Do not deploy until the Custom Access Token Hook is explicitly enabled, a newly issued token demonstrates the dedicated `naeos_scopes` claim, and the scope permission matrix is tested. Supabase's default session JWT does not automatically grant NAEOS-specific API scopes.
5. Do not use `user_metadata` for roles, scopes, tenant IDs, or any authorization decision.
6. Keep Supabase secret keys out of browser code, logs, source control, and this repository.
7. Require CI evidence for invalid signature, unsupported algorithm, malformed/duplicate EC keys, unknown key, wrong issuer/audience, expiry, future not-before, missing identity, generic-scope rejection, allowlisted-scope handling, JWKS outage, key rotation, and resource ownership.
9. No production database schema changes are authorized by this ADR.

## Deployment contract (selected project; activation still gated)

Candidate values for `naeos-crm-production` (`oruykkehukoiipldzcvc`; not yet approved for deployment):

```env
NAEOS_AUTH_ISSUER=https://oruykkehukoiipldzcvc.supabase.co/auth/v1
NAEOS_AUTH_AUDIENCE=authenticated
NAEOS_AUTH_JWKS_URL=https://oruykkehukoiipldzcvc.supabase.co/auth/v1/.well-known/jwks.json
```

Confirm the actual `iss` and `aud` from a real access token and confirm the live JWKS contains the active ES256 P-256 public key before enabling any value. Keep all three unset in production until the signing-key, claim-mapping, and remaining IAM gates pass. The CRM project's own Data API authorization and cross-project Intelligence access are separate trust boundaries.

## Consequences

- Supabase owns user authentication, session lifecycle, refresh tokens, and signing-key custody.
- NAEOS owns API authorization, resource ownership, audit attribution, and fail-closed enforcement.
- NAEOS Intelligence can verify tokens locally using public JWKS without receiving Supabase private signing material.
- Cross-project CRM database access is not automatically granted by a token from another Supabase project. Any cross-project access must use an explicitly designed backend trust boundary rather than assuming PostgREST will accept a foreign project's JWT.
