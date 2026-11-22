# NAEOS IAM — Supabase Integration Compatibility Review

**Date:** 2026-11-11  
**Repository:** `NAEOS-foundation/naeos-intelligence`  
**Branch:** `docs/iam-supabase-hosted-idp-decision`  
**Scope:** Static review of the current JWT verifier, auth middleware, and protected HTTP handlers. No runtime or Supabase configuration was changed.

## Status update — 2026-11-22

This document is a historical static review of the 2026-11-11 implementation. PR #80 and the 2026-11-22 amendment to ADR-0004 implement the human-session identity decision: verified `iss + sub` is the canonical principal, OAuth `client_id` is separate, and resource ownership derives from the issuer-qualified human owner key. The earlier IAM-04/05/09 decision blockers are resolved for this bounded API phase, subject to PR #80 final review and merge. IAM-03, IAM-08, IAM-10/11, and IAM-14 remain open; production activation remains blocked.

## 1. Executive finding

The current verifier can validate an RS256 JWT against a configured JWKS and checks issuer, audience, expiry, and not-before. However, the current principal model is not yet a safe, fully specified mapping for Supabase Auth human-user sessions.

At the time of this historical review, two compatibility decisions were blockers:

1. The verifier maps `client_id` to `Principal.ClientID` when present, falling back to `sub`. The HTTP handlers then use `Principal.ClientID` as the storage ownership key. This conflates OAuth client identity and the authenticated subject unless an explicit policy defines the intended mapping.
2. Protected routes require NAEOS-specific scopes (`intelligence:read` and `intelligence:write`). The verifier extracts only `scope` and `scopes`; there is no trusted mapping from Supabase Auth's authenticated role or application metadata to NAEOS capabilities in the reviewed code. Do not treat the generic `authenticated` database role as an NAEOS API scope.

These are design/authorization gaps, not evidence that the RS256 signature verifier is broken.

## 2. Code evidence

### JWT principal construction

File: `internal/httpapi/jwks_verifier.go`

- `jwtClaims` includes `Subject`, `ClientID`, `Scope`, and `Scopes`.
- `Verify` prefers `claims.ClientID`; when blank, it falls back to `claims.Subject`.
- The resulting value is stored in `Principal.ClientID`.
- The verifier extracts authorization only from `scope` and `scopes`.

### Protected route authorization and ownership

Files: `internal/httpapi/auth.go`, `internal/httpapi/handler.go`

- Middleware rejects a protected request with 403 when the required NAEOS scope is absent.
- Submit, get, cancel, approve, and analyze handlers pass `Principal.ClientID` to storage/orchestration operations.
- Therefore, the same field currently represents both the identity accepted by authentication and the data ownership partition used by the application.

## 3. Historical decisions required before provider-specific activation

- **IAM-04:** Define the canonical human principal. Recommended candidate: namespace the validated subject by issuer (`iss + sub`), rather than using an OAuth client identifier as the human principal.
- **IAM-05:** Decide whether the first integration is human-user-only. If OAuth clients or machine identities are later supported, represent client identity separately from the acting human/service principal.
- **IAM-08:** Approve a trusted NAEOS capability mapping. Candidate approaches include a server-controlled authorization lookup or a reviewed Supabase access-token hook that emits narrowly scoped claims. Do not trust user-editable `user_metadata`; do not map the generic `authenticated` role to NAEOS read/write scopes.
- **IAM-09:** Decide whether persisted request ownership is by human principal, client, tenant, or an explicit composite key. Update storage interfaces and tests consistently once approved.

The current implementation decision is recorded in the 2026-11-22 amendment to ADR-0004. This does not resolve the still-open trusted scope mapping (IAM-08), issuer/audience evidence (IAM-03), token lifecycle (IAM-10/11), or audit attribution (IAM-14).

## 4. Verification required

Before enabling the selected CRM issuer in any deployed environment:

1. Inspect a real access token locally and confirm `alg`, `kid`, `iss`, `aud`, `sub`, `exp`, and optional `nbf`. Never share the raw token.
2. Confirm the live JWKS has a matching RSA signing key for the token's `kid`.
3. Add tests for subject/client separation and the approved canonical principal mapping.
4. Add positive and negative tests for the trusted NAEOS scope mapping.
5. Add resource ownership tests proving one principal/client cannot read, cancel, or approve another principal/client's request.
6. Preserve fail-closed behavior for missing claims, unknown keys, JWKS outages, invalid signatures, and insufficient scope.

## 5. Non-actions

- No code or runtime authorization behavior was changed by this review.
- No Supabase Auth hook, JWT template, signing-key setting, or production environment variable was changed.
- No RLS policy, database grant, migration, or production database setting was changed.
- This historical report does not authorize production activation. IAM-04/05/09 are now addressed for the bounded human-session API by ADR-0004 amendment; IAM-03/08/10/11/14 remain gated.
