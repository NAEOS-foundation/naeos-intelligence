# IAM Principal and Ownership Compatibility Inventory

**Status:** Discovery baseline — implementation design gate  
**Date:** 2026-11-22  
**Source branch reviewed:** `main` at `bd396f55af39ff9ab763aae420fe9832b35402c0`  
**Scope:** Compatibility mapping only. No code/runtime changes.

## Current identity/ownership flow

1. `internal/httpapi/jwks_verifier.go` validates RS256/JWKS, issuer, audience, exp and optional nbf, then derives `Principal.ClientID` from `client_id` with `sub` fallback.
2. `internal/httpapi/auth.go` rejects a verified principal with empty ClientID and attaches identity to server-created request context.
3. `internal/httpapi/handler.go` uses `principal.ClientID` for storage ownership in request create and related handlers.
4. Storage ownership semantics are therefore coupled to the verifier's client claim mapping.
5. Protected routes use NAEOS-specific scopes `intelligence:read` / `intelligence:write`; verifier reads `scope` and `scopes`, combining values. Generic Supabase role or editable user metadata must not be treated as capability grants.
6. Core connector independently calls the Core API with a pre-issued service token requiring `naeos:repository:read`; Core allowlist is configured via `NAEOS_GITHUB_ALLOWED_REPOSITORIES`. Core access is shared platform integration data and is not tenant-isolated.

## Compatibility map

| Component | Existing contract | IAM target | Risk / migration note |
|---|---|---|---|
| Token verifier | `Principal.ClientID` from `client_id`, fallback `sub`; scopes parsed from `scope` + `scopes` | Canonical actor `iss+sub`; client separate; service actor typed; capabilities from trusted registry/grant | Breaking ownership semantics if applied in one step; requires explicit compatibility strategy |
| Auth middleware | Requires nonempty `ClientID`; request context identity | Require valid canonical actor and explicit owner/client binding; fail closed | Middleware contract and test fake verifiers need refactor |
| HTTP handlers | Use `ClientID` as persistence owner and authorization scope | Owner mapping explicitly defined and server-derived; never caller supplied | API behavior/data migration must preserve existing records or deliberately reject unsupported identity |
| Storage/persistence | Ownership keyed by current client identifier | Stable server-side owner key/binding with defined migration | Inventory schemas/indexes/queries/migration/backfill before behavior change |
| Scope parsing | `scope` string and `scopes` array merged | Capability registry is authoritative; token claims only accepted through a vetted mapping/profile | Existing tokens may not carry NAEOS scopes; no generic role fallback |
| JWKS cache | Five-minute refresh; unknown kid triggers refresh; timeout 5s; fetch failure rejects | Rotation and revocation lifecycle requirements; authoritative session/version lookup for revocation | JWKS key refresh is not user/session revocation |
| Core connector | Read-only service token with `naeos:repository:read`; explicit repository allowlist | Shared resource classification, read-only, platform allowlist, service principal | Ensure deny behavior and allowlist lifecycle audited; no tenant isolation assertion |
| Audit | Existing request ID/error handling; actor persistence incomplete per traceability | Actor issuer/principal, client/service actor, decision and operation evidence without token/secret logging | Define event schema/retention/access before claiming verified audit |

## Known consumer/test locations

- `internal/httpapi/auth.go` — middleware uses `Principal.ClientID` non-empty validation.
- `internal/httpapi/handler.go` — request creation passes `principal.ClientID` to store; ownership enforcement consumers to inspect across handler routes.
- `internal/httpapi/jwks_verifier.go` — claim parsing, fallback and scope behavior.
- `internal/httpapi/jwks_verifier_test.go` — verifier expectations include client claim and scopes.
- `internal/httpapi/e2e_postgres_test.go` — fake principal is seeded with `ClientID`; tests rely on owner-scoped behavior.
- `docs/adr/20261110-read-only-api-authorization-baseline.md` — ADR-0004; read-only endpoint scopes plus server-side ownership using verifier-derived ClientID.
- `docs/adr/20261110-single-tenant-mvp-baseline.md` — ADR-0003; one logical tenant per deployment, not a claim of runtime tenant isolation.
- `docs/security/NAEOS-IAM-SUPABASE-INTEGRATION-COMPATIBILITY-REVIEW.md` — prior compatibility blockers: client/sub conflation and no trusted capability mapping.
- `docs/specifications/REQUIREMENT-TRACEABILITY-MATRIX.md` — acceptance currently blocked pending IAM decisions and evidence.
- Core repository: README documents service token scope and explicit allowlist variable; Core policy and tests must be audited separately before implementation.

## Required follow-up before implementation PR

1. Inspect full handlers, store interface, SQL schema/migrations, ownership queries, audit events, fake/test principals, API/OpenAPI and traceability requirement references.
2. Define a stable owner key/binding migration; do not assume `sub` equals current `client_id` or existing owner IDs.
3. Decide and test Supabase claim profile using sanitized real-token metadata or authoritative issuer documentation plus non-production test tokens; no secrets/tokens committed.
4. Define capability registry lifecycle, grant source, mapping from token/session to actor/client, and denial behavior.
5. Define revocation lookup key/store, consistency/caching target, timeout/outage/recovery behavior.
6. Define Core allowlist ownership/change approval, deny tests, and audit.
7. Build a migration plan preserving existing data and backward compatibility or explicitly rejecting unsupported identities; do not alter schema before evidence.

## Verification invariants

- Distinct issuer with same subject yields distinct principal.
- Actor and OAuth client are separately attributable; no sub/client fallback.
- Missing, malformed, ambiguous or unbound identity fails closed.
- Generic Supabase authenticated role or user-editable metadata grants no capability.
- Revoked session denied despite otherwise valid token; lookup timeout/error/indeterminate denies.
- Human token `exp-iat` must be at most 900 seconds, and issuer compatibility is demonstrated.
- Ownership queries never accept owner from request body/header.
- Shared Core allowlisted read succeeds only for authorized service actor; unlisted repo, wrong actor, missing scope, or mutation deny with no outbound/mutation side effect.
- Audit records decision/outcome without token/secret material.
- Denied operation reaches a side effect = FAIL; untestable = BLOCKED/NOT TESTED.

## Status

Compatibility inventory identifies migration risk but does not authorize changes to token behavior, storage ownership, database schema, Supabase configuration, staging or production.


## Deep compatibility audit — persistence, approval, and ownership evidence

**Audit date:** 2026-11-22  
**Reviewed source baseline:** `main` at `bd396f55af39ff9ab763aae420fe9832b35402c0`

### Confirmed findings

1. **Owner key is persisted as `client_id` (not canonical actor).** `internal/httpapi/model.go` exposes Store methods taking `clientID`; `internal/postgres/store.go` inserts, selects, cancels, and idempotency-conflicts using `client_id`. Migration `00001_request_persistence.sql` defines a non-null `client_id`, a unique idempotency pair (`client_id,idempotency_key`), and an index `(client_id,created_at DESC)`. Directly changing owner key to `iss+sub` would affect ownership, idempotency and data access.
2. **Cross-client isolation is tested, but canonical actor isolation is not.** `internal/httpapi/e2e_postgres_test.go` tests client-a vs client-b GET isolation and currently uses fake principals with only ClientID. This does not verify distinct issuer namespaces, subject/client distinction, owner binding migration, or same subject across issuers.
3. **Request-transition audit actor is client-based.** Create/cancel persistence inserts `actor_type='client'` and `actor_id=clientID`. The transition table has actor_type/actor_id but no issuer/principal/client separation. Approval persistence records `approved_by` as actor_id and governance audit, but this method's signature receives `approvedBy` separately; call-site authorization/identity binding must be audited before treating that value as trusted actor attribution.
4. **Approval storage is also ownership-keyed by client_id.** `internal/postgres/approval_store.go` selects request, approval decision and update rows by client_id. Its decision record has client_id plus approved_by; migration/backfill must consider approval rows and audit history, not only request rows.
5. **Approval actor trust is a separate high-priority audit item.** `RecordApprovalDecision(... approvedBy ...)` validates non-empty/length but this storage method alone cannot establish that approvedBy is derived from authenticated context rather than caller input. Verify route/controller and tests; do not change it in this audit-only step.
6. **Single-deployment tenancy remains the accepted boundary.** Current client_id is a storage owner partition, not proof of multi-tenant isolation. ADR-0003 remains applicable.

### Migration/compatibility requirements derived from evidence

- Preserve old client owner values as legacy ownership references; do not blindly backfill them to subject IDs.
- Introduce an explicit, server-controlled owner-binding mapping (issuer + subject -> stable owner key) only after collision/identity and migration semantics are designed.
- Maintain idempotency uniqueness behavior during migration and define how keys are handled if owners map/merge.
- Plan for request rows, approval_decisions, request_transitions actor IDs, governance audit events, and indexes/constraints together.
- Prove legacy-owner access, bound-owner access, unbound/revoked denial, cross-issuer same-sub isolation, and idempotency replay/conflict behavior with side-effect assertions.
- Audit approval route identity source and verify actor attribution cannot be spoofed before any IAM code change.

### Audit scope still open

- Complete route and call-site audit for approval decision APIs and any other identity-consuming endpoints.
- Full schema inventory for approval_decisions and governance_audit_events, and all query/update ownership predicates.
- Core API allowlist implementation/tests and change ownership.
- Token profile/issuer compatibility and session revocation implementation details.

### Gate

This section is static repository evidence only. No code, database, Supabase, runtime, staging or production changes were performed. Implementation remains blocked until owner-binding/backfill design and approval actor trust-source audit are documented and tested.
