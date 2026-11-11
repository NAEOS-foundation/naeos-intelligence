# Phase 5 IAM — MVP Decision Brief

**Document ID:** NAEOS-INT-IAM-BRIEF-001  
**Version:** 0.1.0  
**Status:** Partially accepted — IAM-01 and IAM-02 accepted; `naeos-crm-production` selected as IdP; IAM-03 through IAM-14 remain open  
**Related:** `PHASE-05-IAM-DECISION-WORKSHEET.md`, `API-CONTRACT.md`, `SECURITY-REQUIREMENTS.md`, `REQUIREMENT-TRACEABILITY-MATRIX.md`

> This brief remains a decision aid, not a complete accepted IAM specification. The maintainer has accepted Supabase Auth session JWTs with RS256 for IAM-01 and selected the `naeos-crm-production` project (`oruykkehukoiipldzcvc`, Singapore) as the hosted IdP for IAM-02. The `naeos.db` project (`kqmhpsftijskesjdovjm`, Tokyo) remains the NAEOS Intelligence database. IAM-03 through IAM-14 remain open. See `ADR-005-SUPABASE-AUTH-HOSTED-IDP.md`. Provider-specific production activation remains gated.

## 1. Decision objective

Select a minimal, auditable identity and access-control baseline for NAEOS Intelligence early access while preserving partner interoperability and avoiding premature multi-tenant or identity-provider complexity.

## 2. Constraints and decision principles

- The MVP currently exposes read-only analysis behavior; do not introduce operational approval or consequential execution through IAM work.
- Security-critical behavior must fail closed and be testable in CI.
- Separate end-user/service principal identity from OAuth client identity.
- Do not trust identity or tenant values supplied through arbitrary headers or request payloads.
- Provider-specific code and requirements status changes remain gated on maintainer-approved decisions.
- Recommendations below are provisional; operating model, partner onboarding, deployment topology, and threat model can change them.

## 3. Decision options and provisional recommendations

### IAM-01 — Authentication protocol and token format

| Option | Benefits | Costs / risks |
|---|---|---|
| OIDC/OAuth 2.0 with signed JWT access tokens | Broad ecosystem support; local signature validation; interoperable clients | Correct claim validation, key rotation, revocation and token lifetime policy required |
| OAuth 2.0 opaque token with introspection | Centralized revocation and policy | Runtime dependency/latency; availability and cache policy complexity |
| Service-specific signed credentials | Simple for tightly controlled machine integrations | Poorer interoperability; bespoke lifecycle and security burden |

**Provisional recommendation:** Prefer standard OIDC/OAuth 2.0 JWT access tokens for a partner-facing pilot only if the issuer supports required validation and lifecycle controls. Evaluate opaque introspection if prompt revocation is a requirement. Avoid bespoke credentials absent a compelling constraint.

**Maintainer decision:** Accepted — Supabase Auth session access JWT; RS256 allowlist. See `ADR-005-SUPABASE-AUTH-HOSTED-IDP.md`.

### IAM-02 — Identity provider / issuer authority

| Option | Benefits | Costs / risks |
|---|---|---|
| Existing organization or partner IdP | Reduces NAEOS identity operations; may fit pilot partners | Multiple issuers and onboarding variance |
| NAEOS-managed hosted IdP | Consistent onboarding and policy control | Operational ownership, availability, cost, account recovery and security responsibility |
| Build/operate a custom IdP | Maximum control | High complexity and security risk; inappropriate without demonstrated need |

**Provisional recommendation:** Use an established hosted/organization IdP for early access; do not build a custom IdP. Select the exact issuer only after partner and deployment requirements are known.

**Maintainer decision:** Accepted — hosted Supabase Auth in `naeos-crm-production` (`oruykkehukoiipldzcvc`, Singapore). `naeos.db` (`kqmhpsftijskesjdovjm`, Tokyo) remains the Intelligence database. The exact issuer/audience policy is still subject to IAM-03 and real-token verification. See `ADR-005-SUPABASE-AUTH-HOSTED-IDP.md`.

### IAM-03 — Issuer and audience policy

**Options:** single pinned issuer/audience per deployment; allowlisted multiple issuers/audiences.

**Provisional recommendation:** Pin issuer and audience per environment/deployment as the safer initial trust boundary. Expand to allowlisted multi-issuer only with explicit issuer-to-client/tenant binding and tests.

**Maintainer decision:** Open.

### IAM-04 / IAM-05 — Principal canonicalization and client semantics

**Options:** canonical principal as validated `iss + sub`; provider-specific stable mapping; conflated client/principal identity for machine-only service use.

**Provisional recommendation:** Namespace principal identity by issuer (validated `iss + sub` or a stable server-side mapping). Keep client identity distinct from principal. For machine-to-machine callers, define an explicit service principal instead of treating `client_id` as a human actor. Persist both actor and client where the audit model supports it.

**Maintainer decision:** Open.

### IAM-06 / IAM-07 — Tenant applicability and binding

| Model | Benefits | Costs / risks |
|---|---|---|
| Single-tenant deployment / isolated pilot | Smallest trust boundary and simplest isolation story | Separate deployments/configuration may be needed per partner |
| Tenant-scoped deployment | Clear tenant partitioning | Requires explicit tenant identity, binding, persistence and operational controls |
| Shared claims-based multi-tenancy | Efficient shared operations | Highest risk/cost: cross-tenant authorization, registry, migration, incident and test burden |

**Provisional recommendation:** Do not enable shared multi-tenancy by default for MVP. Prefer single-tenant deployment or an explicitly isolated tenant boundary based on the early-access deployment model. If shared tenancy is selected, require a trusted server-side client-to-tenant binding, persistence-level tenant scoping, and cross-tenant integration tests before acceptance. Never accept tenant identity solely from request body or caller-controlled headers.

**Maintainer decision:** Open.

### IAM-08 / IAM-09 — Authorization vocabulary and resource ownership

**Options:** role-based authorization; scopes/capabilities; resource ownership; layered scopes plus ownership.

**Provisional recommendation:** Use layered authorization: narrow operation scopes/capabilities at API boundaries plus server-side resource ownership checks. Keep vocabulary small and map each endpoint/action explicitly. Decide whether ownership attaches to client, principal, or tenant before changing persistence semantics; do not infer a policy from the current fallback implementation.

**Maintainer decision:** Open.

### IAM-10 — Token validation requirements

**Provisional recommendation:** Require algorithm allowlisting (no algorithm negotiation from token), trusted JWKS source, key selection, signature, issuer, audience, expiration, and explicit required claim/type checks. Define clock skew and treatment of `nbf`/`iat`; reject ambiguous or malformed claims. Tests should cover each requirement. This recommendation is a specification proposal, not an assertion that the current verifier fully satisfies it.

**Maintainer decision:** Open.

### IAM-11 — Revocation and key rotation

**Options:** short-lived tokens with JWKS rotation; introspection; denylist; hybrid.

**Provisional recommendation:** For JWT pilot, establish short token lifetime and documented JWKS cache/rotation behavior first; define emergency revocation path and acceptable propagation time. If that cannot meet partner risk requirements, use introspection or hybrid. Specify behavior for unknown `kid`, JWKS outage, cache refresh, and old/new key overlap, and test those cases.

**Maintainer decision:** Open.

### IAM-12 — Identity propagation

**Provisional recommendation:** Propagate validated identity through immutable server-created request context. Strip or ignore untrusted identity headers. If identity crosses a service trust boundary, specify authenticated/signed propagation with replay and audience controls.

**Maintainer decision:** Open.

### IAM-13 — Denial and information disclosure

**Provisional recommendation:** Return 401 for missing/invalid authentication and 403 for authenticated but insufficient authorization; consider 404 for protected resource existence disclosure where consistently applied. Normalize error details and avoid secrets or sensitive claim leakage.

**Maintainer decision:** Open.

### IAM-14 — Audit attribution

**Provisional recommendation:** Define minimum event fields: request/correlation ID, issuer, canonical principal or service actor, client ID, tenant ID only if enabled, operation/resource reference, authorization outcome, timestamp, and policy/decision version where applicable. Apply data minimization, retention, and access controls. Persisted actor attribution and audit integrity must be verified before Phase 5 acceptance.

**Maintainer decision:** Open.

## 4. Suggested acceptance sequence

1. Maintainer confirms early-access deployment model (isolated/single-tenant versus shared multi-tenant) and partner identity requirements.
2. Accept IAM-01 through IAM-03 (protocol, issuer, trust boundaries).
3. Accept IAM-04 through IAM-09 (actor/client/tenant and authorization semantics).
4. Accept IAM-10 through IAM-14 (validation, lifecycle, propagation, denial, audit).
5. Record rationale and date in the worksheet and create accepted ADR(s).
6. Align API Contract, Security Requirements, Data Model, traceability matrix, and Phase 5 implementation prompt.
7. Implement in scoped PRs with requirement-mapped tests and retained CI evidence. Do not mark requirements verified without evidence.

## 5. Early-access decision questions

Maintainer input is required for these questions; this document intentionally does not answer them on the maintainer's behalf:

- Will early access be isolated deployment per partner or a shared service?
- Which IdP/issuer must the first partner integration use, if already known?
- Are callers exclusively machine-to-machine, or will human users act through clients?
- Is rapid token revocation required beyond short-lived tokens and key rotation?
- What actor/client/tenant attribution and retention are mandatory for audit?

## 6. Review record

| Decision group | Maintainer outcome | Rationale / ADR | Date |
|---|---|---|---|
| IAM-01–02 | Accepted — Supabase Auth RS256 JWT; `naeos-crm-production` selected as IdP | ADR-005 | 2026-11-11 |
| IAM-03 | Pending — pin issuer/audience per environment after real-token verification | Pending | Pending |
| IAM-04–05 | Pending | Pending | Pending |
| IAM-06–07 | Pending | Pending | Pending |
| IAM-08–09 | Pending | Pending | Pending |
| IAM-10–14 | Pending | Pending | Pending |
