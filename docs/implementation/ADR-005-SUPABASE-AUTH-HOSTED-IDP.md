# ADR-005: Supabase Auth as the hosted NAEOS Identity Provider

**Status:** Accepted for IAM-01, IAM-02, and IAM-08 policy; `naeos-crm-production` selected as IdP; IAM-03/04/05/06/07/09/10/11/12/13/14 remain open or gated  
**Date:** 2026-11-11  
**Decision owner:** NAEOS maintainer  
**Related:** `PHASE-05-IAM-DECISION-WORKSHEET.md`, `PHASE-05-IAM-MVP-DECISION-BRIEF.md`

## Context

NAEOS Intelligence already has a provider-neutral `TokenVerifier` boundary and an ES256-only JWKS verifier. The repository does not establish a complete first-party token issuer. The maintainer selected a hosted identity provider and explicitly chose Supabase.

Supabase Auth issues session access JWTs and exposes asymmetric public signing keys through its project JWKS discovery endpoint. Supabase supports asymmetric ES256 (P-256) signing keys and publishes their public JWKs for local verification. The maintainer confirmed ES256 is active in the selected Supabase Auth project and approved ES256-only verification for NAEOS.

## Decisions

### IAM-01 — Protocol and token format

Accepted: use Supabase Auth session access JWTs as bearer credentials for human-user access to NAEOS APIs. Accept ES256 only (ECDSA P-256 with SHA-256) at the NAEOS Intelligence verifier boundary. Do not accept RS256, HS256, `none`, or algorithm negotiation.

This decision does not authorize using Supabase API keys as user access tokens. Publishable keys identify the client application; they are not substitutes for a user's access JWT. Secret keys remain server-only and must never be used as user bearer tokens.

### IAM-02 — Identity provider

Accepted: use hosted Supabase Auth. Do not build a custom NAEOS token issuer or manage a NAEOS-owned private signing key for this integration.

Selected IdP project: `naeos-crm-production` (`oruykkehukoiipldzcvc`, Singapore). The `naeos.db` project (`kqmhpsftijskesjdovjm`, Tokyo) remains the NAEOS Intelligence database and is not the token issuer. This selection keeps CRM's native Auth/Data API trust boundary within one Supabase project while allowing NAEOS Intelligence to verify CRM-issued tokens through the issuer's public JWKS. The exact issuer/audience contract still requires real-token verification under IAM-03. Do not configure production issuer/JWKS environment variables until the signing key, JWKS, token claims, and remaining IAM gates are verified.

## Current implementation evidence

- `internal/httpapi/jwks_verifier.go` validates ES256 signatures against P-256 public keys from a remote JWKS and checks issuer, audience, expiry, not-before, issue time, and key ID. It requires `iss`, matching `aud`, non-empty `sub`, `exp`, and `iat`; rejects `iat >= exp`; and applies a bounded 60-second clock skew to expiry/not-before and future-issued-at checks.
- `cmd/naeos-intelligence/main.go` reads `NAEOS_AUTH_ISSUER`, `NAEOS_AUTH_AUDIENCE`, and `NAEOS_AUTH_JWKS_URL`.
- `internal/httpapi/auth.go` fails closed when the verifier is not configured, enforces route scopes, and propagates verified principal only via a private typed request-context key. Caller-controlled identity headers are not an identity source; existing handler ownership tests include spoofed `X-Client-ID`/`X-Principal` values.
- Protected routes require `intelligence:read` or `intelligence:write`.

## Required follow-up decisions — still open

- IAM-03: exact issuer and audience pinning per environment.
- IAM-04: canonical principal key, including issuer namespace and subject.
- IAM-05: distinction between human principal and OAuth/service client. Supabase Auth session JWTs are not a client-credentials grant.
- IAM-06/07: single-tenant pilot or tenant model and trusted tenant binding.
- IAM-08: Accepted capability policy and code path: Supabase Custom Access Token Hook emits dedicated `naeos_scopes` from administrator-managed `app_metadata.naeos_scopes`; NAEOS verifier allowlists `intelligence:read` and `intelligence:write`, ignores generic `scope`/`scopes` and `user_metadata`. SQL reference: `docs/implementation/supabase/NAEOS-CUSTOM-ACCESS-TOKEN-HOOK.sql`. Actual hook enablement and a real issued-token evidence remain gated; do not activate production yet.
- IAM-09: persisted resource ownership semantics.
- IAM-10: Implemented baseline requires `iss`, matching `aud`, non-empty `sub`, `exp`, and `iat`; rejects `iat >= exp`; uses 60-second clock skew for `exp`/`nbf` and future `iat`. CI verification and maintainer acceptance remain pending.
- IAM-11: Implemented baseline (acceptance pending): local JWT verification cannot immediately revoke an already-issued access token on Supabase session sign-out; its `exp` remains the revocation bound unless an online session check/denylist is added. The verifier caches JWKS for 5 minutes, refreshes immediately for unknown `kid`, and fails closed when JWKS retrieval or validation fails (no stale-key fallback on fetch error). Supabase documents a 10-minute edge cache for JWKS and propagation/cache layers that may clear every 20 minutes or longer. Routine rotation: publish/prepare the new standby key and allow caches to observe it before rotating; retain the previously used key until issued tokens expire, then revoke it provider-side. Emergency compromise: revoke the compromised signing key at Supabase, purge/restart local verifier cache where operationally possible, monitor rejection behavior, and account for documented provider cache propagation; this implementation does not promise instantaneous revocation. If the pilot requires immediate per-session revocation, add a trusted online session-status check (e.g. verify `session_id` against Supabase Auth session state) or a centrally managed denylist before deployment.
- IAM-12: Implemented baseline (acceptance pending): the `Authenticator` validates the bearer credential and required scope, then stores the verified `Principal` in a private typed Go request-context key (`internal/httpapi/auth.go`). Protected handlers read identity only through `principalFrom`; they do not trust caller-supplied `X-Client-ID` or `X-Principal` headers. Existing `handler_test.go` ownership-isolation coverage demonstrates spoofed identity headers do not override the authenticated principal. This service currently propagates identity within the in-process request context; no internal service-to-service trust boundary or forwarded identity headers are introduced. If a future service boundary requires propagation, authenticate the receiving service and use a versioned signed propagation contract with issuer, subject, audience, expiry, and replay protection rather than trusting plain headers.
- IAM-13: 401/403/404 denial and non-disclosure behavior.
- IAM-14: audit attribution and retention.

## Security gates

1. Keep production authentication fail-closed. Do not disable middleware or remove scope checks to make integration tests pass.
2. Do not assume the project's active signing algorithm from its creation date. Confirm the active signing key is EC P-256/ES256 in Supabase Dashboard and confirm the JWKS contains a matching EC public key.
3. Verify a real access JWT from the selected project: `alg=ES256`, non-empty `kid`, exact `iss`, expected `aud`, `sub`, `exp`, and `nbf` where present.
4. Do not deploy until the Custom Access Token Hook is explicitly configured, a real newly issued access JWT demonstrates the dedicated `naeos_scopes` claim, and the end-to-end scope permission matrix is verified. Supabase's default session JWT does not automatically grant NAEOS-specific API scopes.
5. Do not use `user_metadata` for roles, scopes, tenant IDs, or any authorization decision.
6. Keep Supabase secret keys out of browser code, logs, source control, and this repository.
7. Require CI evidence for invalid signature, unsupported algorithm, malformed/duplicate EC keys, unknown key, wrong issuer/audience, expiry, future not-before, missing identity, insufficient scope, JWKS outage, key rotation, and resource ownership.
8. Account for Supabase JWKS edge caching (documented at 10 minutes, with cache layers clearing every 20 minutes or longer) and the NAEOS verifier cache (5 minutes); key compromise response is not instantaneous in this local-verification baseline. Do not promise immediate user-session revocation from sign-out: an issued JWT can remain valid until `exp` unless an online session check/denylist is implemented.
8. No production database schema changes are authorized by this ADR.

## Deployment contract (selected project; activation still gated)

Candidate values for `naeos-crm-production` (`oruykkehukoiipldzcvc`; not yet approved for deployment):

```env
NAEOS_AUTH_ISSUER=https://oruykkehukoiipldzcvc.supabase.co/auth/v1
NAEOS_AUTH_AUDIENCE=authenticated
NAEOS_AUTH_JWKS_URL=https://oruykkehukoiipldzcvc.supabase.co/auth/v1/.well-known/jwks.json
```

Confirm the actual `iss` and `aud` from a newly issued real access token and confirm its `kid` resolves to the active ES256 P-256 public key in live JWKS before enabling any value. Keep all three unset in production until the signing-key, claim-mapping, and remaining IAM gates pass. The CRM project's own Data API authorization and cross-project Intelligence access are separate trust boundaries.

## Consequences

- Supabase owns user authentication, session lifecycle, refresh tokens, and signing-key custody.
- NAEOS owns API authorization, resource ownership, audit attribution, and fail-closed enforcement.
- NAEOS Intelligence can verify tokens locally using public JWKS without receiving Supabase private signing material.
- Cross-project CRM database access is not automatically granted by a token from another Supabase project. Any cross-project access must use an explicitly designed backend trust boundary rather than assuming PostgREST will accept a foreign project's JWT.
