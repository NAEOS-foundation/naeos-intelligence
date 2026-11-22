# Phase 5 IAM — MVP Decision Brief

**Document ID:** NAEOS-INT-IAM-BRIEF-001  
**Version:** 0.1.0  
**Status:** Draft for maintainer decision — not an accepted ADR  
**Related:** `PHASE-05-IAM-DECISION-WORKSHEET.md`, `API-CONTRACT.md`, `SECURITY-REQUIREMENTS.md`, `REQUIREMENT-TRACEABILITY-MATRIX.md`, [ADR-0003](../adr/20261110-single-tenant-mvp-baseline.md)

> This brief provides options and provisional recommendations for the remaining IAM decisions. The MVP tenancy baseline is recorded in accepted ADR-0003; that decision does not close other IAM items or authorize provider-specific implementation.
**Status:** Partially accepted — IAM-01/02 accepted; IAM-04/05 and IAM-09 accepted for the human-session read-only API by ADR-0004 amendment; IAM-03, IAM-08, IAM-11, and IAM-14 remain open; IAM-10's 900-second verifier limit is implemented but provider configuration evidence remains gated  
**Related:** `PHASE-05-IAM-DECISION-WORKSHEET.md`, `API-CONTRACT.md`, `SECURITY-REQUIREMENTS.md`, `REQUIREMENT-TRACEABILITY-MATRIX.md`
> This brief remains a decision aid, not a complete accepted IAM specification. The maintainer has accepted Supabase Auth session JWTs with ES256 (P-256) for IAM-01 and selected the `naeos-crm-production` project (`oruykkehukoiipldzcvc`, Singapore) as the hosted IdP for IAM-02. The `naeos.db` project (`kqmhpsftijskesjdovjm`, Tokyo) remains the NAEOS Intelligence database. The 2026-11-22 amendment to ADR-0004 accepts verified `iss + sub` as the human principal (IAM-04), keeps OAuth `client_id` separate (IAM-05), and derives read-only API ownership from that principal (IAM-09). IAM-03 live token evidence, IAM-08 hook activation/claim evidence, IAM-10's verifier maximum lifetime (900 seconds) is implemented, but the Supabase Auth expiry setting and fresh-token evidence remain gated; IAM-11 authoritative session revocation and IAM-14 audit attribution remain open. See `ADR-005-SUPABASE-AUTH-HOSTED-IDP.md`. Provider-specific production activation remains gated.

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

**Maintainer decision:** Open; defer exact token format until a partner/deployment model is known.
**Maintainer decision:** Accepted — Supabase Auth session access JWT; ES256 (P-256) allowlist. Live `alg`/`kid`/issuer/audience evidence remains required before activation. See `ADR-005-SUPABASE-AUTH-HOSTED-IDP.md`.

### IAM-02 — Identity provider / issuer authority

| Option | Benefits | Costs / risks |
|---|---|---|
| Existing organization or partner IdP | Reduces NAEOS identity operations; may fit pilot partners | Multiple issuers and onboarding variance |
| NAEOS-managed hosted IdP | Consistent onboarding and policy control | Operational ownership, availability, cost, account recovery and security responsibility |
| Build/operate a custom IdP | Maximum control | High complexity and security risk; inappropriate without demonstrated need |

**Provisional recommendation:** Use an established hosted/organization IdP for early access; do not build a custom IdP. Select the exact issuer only after partner and deployment requirements are known.

**Maintainer decision:** Open; do not select an issuer without a partner/deployment requirement.
**Maintainer decision:** Accepted — hosted Supabase Auth in `naeos-crm-production` (`oruykkehukoiipldzcvc`, Singapore). `naeos.db` (`kqmhpsftijskesjdovjm`, Tokyo) remains the Intelligence database. The exact issuer/audience policy is still subject to IAM-03 and real-token verification. See `ADR-005-SUPABASE-AUTH-HOSTED-IDP.md`.

### IAM-03 — Issuer and audience policy

**Options:** single pinned issuer/audience per deployment; allowlisted multiple issuers/audiences.

**Provisional recommendation:** Pin issuer and audience per environment/deployment as the safer initial trust boundary. Expand to allowlisted multi-issuer only with explicit issuer-to-client/tenant binding and tests.

**Maintainer decision:** Open; proposed policy is one configured issuer/audience per deployment, with actual values deferred.

### IAM-04 / IAM-05 — Principal canonicalization and client semantics

**Options:** canonical principal as validated `iss + sub`; provider-specific stable mapping; conflated client/principal identity for machine-only service use.

**Provisional recommendation:** Namespace principal identity by issuer (validated `iss + sub` or a stable server-side mapping). Keep client identity distinct from principal. For machine-to-machine callers, define an explicit service principal instead of treating `client_id` as a human actor. Persist both actor and client where the audit model supports it.

**Maintainer decision:** Accepted for this human-session API phase by the 2026-11-22 ADR-0004 amendment: verified `iss + sub` is the canonical human principal; OAuth `client_id` remains a separate client attribute. Client-only/service principals are deferred and rejected by this phase.

### IAM-06 / IAM-07 — Tenant applicability and binding

| Model | Benefits | Costs / risks |
|---|---|---|
| Single-tenant deployment / isolated pilot | Smallest trust boundary and simplest isolation story | Separate deployments/configuration may be needed per partner |
| Tenant-scoped deployment | Clear tenant partitioning | Requires explicit tenant identity, binding, persistence and operational controls |
| Shared claims-based multi-tenancy | Efficient shared operations | Highest risk/cost: cross-tenant authorization, registry, migration, incident and test burden |

**Maintainer decision:** Accepted for MVP baseline by ADR-0003: one logical tenant per deployment; no shared claims-based multi-tenancy or tenant identity from caller-controlled input. IAM-07 tenant claim and binding work is deferred until a shared-tenancy requirement exists.

### IAM-08 / IAM-09 — Authorization vocabulary and resource ownership

**Options:** role-based authorization; scopes/capabilities; resource ownership; layered scopes plus ownership.

**Provisional recommendation:** Use layered authorization: narrow operation scopes/capabilities at API boundaries plus server-side resource ownership checks. Keep vocabulary small and map each endpoint/action explicitly. Decide whether ownership attaches to client, principal, or tenant before changing persistence semantics; do not infer a policy from the current fallback implementation.

**Maintainer decision:** Accepted policy: only the dedicated `naeos_scopes` claim sourced from administrator-managed `app_metadata.naeos_scopes` grants allowlisted NAEOS capabilities. The repository SQL hook is reference-only; actual hook enablement and real-token evidence remain gated. Resource ownership is separately accepted by ADR-0004 amendment as verified `iss + sub`; no public approval route or tool execution is authorized.

### IAM-10 — Token validation requirements

**Provisional recommendation:** Require algorithm allowlisting (no algorithm negotiation from token), trusted JWKS source, key selection, signature, issuer, audience, expiration, and explicit required claim/type checks. Define clock skew and treatment of `nbf`/`iat`; reject ambiguous or malformed claims. Tests should cover each requirement. This recommendation is a specification proposal, not an assertion that the current verifier fully satisfies it.

**Maintainer decision:** Implemented in PR #81: require `iss`, `aud`, `sub`, `exp`, and `iat`; apply 60-second skew; reject `exp - iat` greater than 900 seconds. The selected Supabase Auth project's actual JWT expiry setting and a freshly issued token within this limit remain unverified. Provider configuration was not changed.

### IAM-11 — Revocation and key rotation

**Options:** short-lived tokens with JWKS rotation; introspection; denylist; hybrid.

**Provisional recommendation:** For JWT pilot, establish short token lifetime and documented JWKS cache/rotation behavior first; define emergency revocation path and acceptable propagation time. If that cannot meet partner risk requirements, use introspection or hybrid. Specify behavior for unknown `kid`, JWKS outage, cache refresh, and old/new key overlap, and test those cases.

**Maintainer decision:** Baseline implemented: 5-minute local JWKS cache, immediate unknown-`kid` refresh, fail closed on refresh failure. Authoritative session/version revocation is not implemented; sign-out does not invalidate an already-issued JWT before `exp` in this offline verifier model. Acceptance remains open.

### IAM-12 — Identity propagation

**Provisional recommendation:** Propagate validated identity through immutable server-created request context. Strip or ignore untrusted identity headers. If identity crosses a service trust boundary, specify authenticated/signed propagation with replay and audience controls.

**Maintainer decision:** Accepted for in-process read-only API by ADR-0004; cross-service identity propagation remains out of scope.

### IAM-13 — Denial and information disclosure

**Provisional recommendation:** Return 401 for missing/invalid authentication and 403 for authenticated but insufficient authorization; consider 404 for protected resource existence disclosure where consistently applied. Normalize error details and avoid secrets or sensitive claim leakage.

**Maintainer decision:** Accepted for read-only API by ADR-0004: 401 for missing/invalid authentication, 403 for missing scope, and 404 for inaccessible resources.

### IAM-14 — Audit attribution

**Provisional recommendation:** Define minimum event fields: request/correlation ID, issuer, canonical principal or service actor, client ID, tenant ID only if enabled, operation/resource reference, authorization outcome, timestamp, and policy/decision version where applicable. Apply data minimization, retention, and access controls. Persisted actor attribution and audit integrity must be verified before Phase 5 acceptance.

**Maintainer decision:** Open.

## 4. Suggested acceptance sequence

1. Keep the accepted single-tenant logical boundary for MVP; reopen it only if a partner or deployment requirement calls for shared tenancy.
2. IAM-01/02 are accepted (Supabase Auth ES256); IAM-03 policy pins issuer/audience per deployment, but safe live-token metadata and matching JWKS key evidence remain required.
3. IAM-04/05/09 are accepted for the human-session read-only API by the ADR-0004 amendment; IAM-08 capability policy is accepted, but hook activation and real issued-token claim evidence must be verified before activation.
4. Verify the Supabase Auth expiry setting and real-token evidence for IAM-10; resolve IAM-11 authoritative session revocation and IAM-14 audit attribution. The in-process propagation and API denial contract for IAM-12/13 is accepted by ADR-0004.
5. Record rationale and date in the worksheet and create accepted ADR(s).
6. Align API Contract, Security Requirements, Data Model, traceability matrix, and Phase 5 implementation prompt.
7. Implement in scoped PRs with requirement-mapped tests and retained CI evidence. Do not mark requirements verified without evidence.

## 5. Early-access decision questions

Maintainer input is required for these questions; this document intentionally does not answer them on the maintainer's behalf:

- Does a future partner or deployment requirement require shared tenancy? If so, reopen IAM-06/07 before implementation.
- Which IdP/issuer must the first partner integration use, if already known?
- Human Supabase Auth sessions are the accepted caller model for this phase; do future partner requirements require a separate service-principal contract?
- Is rapid token revocation required beyond short-lived tokens and key rotation?
- What actor/client/tenant attribution and retention are mandatory for audit?

## 6. Review record

| Decision group | Maintainer outcome | Rationale / ADR | Date |
|---|---|---|---|
| IAM-01–03 | Partially accepted / pending | IAM-01/02 accepted by ADR-005; IAM-03 issuer/audience pinning and live token evidence remain open | Pending |
| IAM-06 | Accepted for MVP baseline | ADR-0003 | 2026-11-10 |
| IAM-07 | Deferred for MVP | No tenant identity in the single-tenant baseline; reopen before shared tenancy | 2026-11-10 |
| IAM-08–09 | Accepted for read-only API | ADR-0004; claim-to-client mapping remains open | 2026-11-10 |
| IAM-10 | Verifier limit implemented; provider evidence pending | Enforce `exp - iat ≤ 900s`; verify the selected Supabase Auth project's expiry setting and a fresh token | 2026-11-22 |
| IAM-11, IAM-14 | Pending / gated | Authoritative session revocation and durable actor-audit attribution remain open | Pending |
| IAM-12–13 | Accepted for read-only API | ADR-0004 | 2026-11-10 |
| IAM-01–02 | Accepted — Supabase Auth ES256 JWT; `naeos-crm-production` selected as IdP | ADR-005 | 2026-11-16 |
| IAM-03 | Pending — pin issuer/audience per environment after real-token verification | Pending | Pending |
| IAM-04–05, IAM-09 | Accepted for human-session read-only API | ADR-0004 amendment; verified `iss + sub`, separate `client_id`, and server-derived owner key | 2026-11-22 |
| IAM-06–07 | Pending | Pending | Pending |
| IAM-08 | Policy accepted; activation evidence pending | Dedicated `naeos_scopes` claim from admin-managed `app_metadata`, allowlisted capabilities; hook activation and real-token evidence remain open | Pending |
| IAM-10 | Verifier limit implemented; IdP evidence gated | 900-second maximum lifetime enforced in PR #81; live provider configuration and token evidence outstanding | 2026-11-22 |
| IAM-11, IAM-14 | Open — remain gated | Authoritative session revocation and durable actor audit attribution remain outstanding | Pending |