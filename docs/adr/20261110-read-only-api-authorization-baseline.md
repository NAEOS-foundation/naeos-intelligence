# ADR-0004: Provider-Neutral Authorization Baseline for the Read-Only API

**Status:** Accepted; amended 2026-11-22 for human-principal ownership  
**Original date:** 2026-11-10  
**Decision owner:** NAEOS maintainer  
**Scope:** Protected read-only HTTP API only. This ADR does not authorize model/tool execution, public approval routes, or shared tenancy.

## Context

The read-only MVP has protected HTTP operations, request ownership checks, and non-disclosing resource errors. The original implementation used a verifier-derived `Principal.ClientID` as the storage owner while the JWT claim mapping remained provisional. The Phase 5 IAM review identified that an OAuth application client identifier must not be conflated with the authenticated human user.

The selected hosted identity provider is Supabase Auth session JWTs (ADR-005). The current integration phase is human-session-only. Machine/service principals require a separately typed identity contract and are not silently treated as human users.

## Decision

- For the bounded read-only API, authorization uses endpoint scopes plus server-side resource ownership checks. The protected API currently maps `intelligence:read` and `intelligence:write` to operations.
- A human principal is the exact, cryptographically verified JWT issuer and subject tuple (`iss + sub`). The issuer namespace is part of the identity; equal `sub` values from different issuers are distinct principals.
- `client_id` remains a separate OAuth client attribute. It is not a human identity, is not an ownership fallback, and does not override `iss + sub`.
- The API derives an opaque owner key server-side from the verified tuple: `user:<base64url(iss)>:<base64url(sub)>`. Request create, idempotency, read, cancellation, and analysis ownership use this same value. The existing persistence column named `client_id` temporarily stores this derived owner key for schema compatibility; the name is not a claim about the actor.
- This phase accepts human Supabase Auth session tokens with non-empty issuer and subject only. Client-only/service tokens fail closed. Machine/service-principal support is deferred to a separate decision and typed contract.
- Existing rows keyed by legacy raw `client_id` are not automatically reassigned. They remain outside the new human-owner namespace until a separately reviewed migration or explicit compatibility policy is approved. No schema DDL or backfill is authorized here.
- Within a request, validated identity is propagated through server-created request context. Caller-supplied identity headers are not trusted.
- Return 401 for missing or invalid authentication, 403 for an authenticated caller lacking the required operation scope, and 404 when a resource is missing or owned by another principal, preserving the non-disclosure contract.
- This ADR covers only the read-only HTTP API. It does not authorize model/tool execution or define runtime authorization for consequential actions.

## Rationale

The `iss + sub` tuple is stable within its issuer namespace and avoids collisions between providers. Keeping `client_id` separate preserves the distinction between the OAuth application and the human acting through it. A server-derived owner key prevents callers from choosing an owner through request fields or headers. The implementation deliberately avoids a schema migration and does not silently transfer legacy records.

## Consequences

Positive:
- Protected read-only operations require both endpoint permission and issuer-qualified human resource ownership.
- Two human subjects sharing one OAuth client ID remain isolated, including idempotency records.
- Equal subjects from different issuers do not collide.
- Missing identity fails closed and inaccessible resources retain the 404 non-disclosure contract.

Limitations and remaining gates:
- IAM-03 issuer/audience pinning and real-token evidence remain open.
- IAM-08 trusted NAEOS scope/capability mapping remains open; generic Supabase `authenticated` role or user-editable metadata must not be treated as NAEOS scopes.
- IAM-10/11 claim-lifetime, revocation, and key-rotation decisions remain open.
- IAM-14 durable actor/client audit attribution and retention remain open.
- Legacy client-owned rows are not mapped to human owners; any future migration must account for requests, idempotency, evidence, transitions, and audit attribution.
- Service-principal support remains deferred.

## References

- `docs/implementation/PHASE-05-IAM-DECISION-WORKSHEET.md` — IAM-04, IAM-05, IAM-08, IAM-09, IAM-14
- `docs/implementation/ADR-005-SUPABASE-AUTH-HOSTED-IDP.md` — selected human-session IdP
- `docs/specifications/API-CONTRACT.md` — protected routes and error contract
- `docs/security/SECURITY-REQUIREMENTS.md` — SEC-AUTH-001, SEC-AC-001
- `docs/adr/20261110-single-tenant-mvp-baseline.md` — single-tenant MVP deployment assumption
