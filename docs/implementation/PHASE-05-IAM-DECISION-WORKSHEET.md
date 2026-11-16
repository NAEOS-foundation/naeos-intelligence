# Phase 5 — Identity and Access Control Decision Worksheet

**Document ID:** NAEOS-INT-IAM-DEC-001  
**Version:** 0.1.0  
**Status:** Draft — IAM-01 and IAM-02 accepted; IAM-03 through IAM-14 remain open  
**Scope:** Decisions required before provider-specific Phase 5 implementation. This worksheet does not constitute an accepted ADR.

## 1. Decision gate

The maintainer has accepted IAM-01 and IAM-02 for Supabase Auth session JWTs signed with ES256 and selected the `naeos-crm-production` Supabase project as the hosted identity provider. The selected project is `oruykkehukoiipldzcvc` (Singapore). The `naeos.db` project (`kqmhpsftijskesjdovjm`, Tokyo) remains the NAEOS Intelligence database, not the token issuer. This project selection does not close IAM-03 through IAM-14. Provider-specific production configuration and authorization-claim changes remain gated until issuer/audience, principal/client semantics, tenant applicability, scope mapping, ownership, validation, revocation, propagation, denial, and audit decisions are recorded. A green test suite does not substitute for decision acceptance.

## 2. Decision register

| ID | Decision | Options to evaluate | Recommended evaluation criteria | State |
|---|---|---|---|---|
| IAM-01 | Authentication protocol and token format | Supabase Auth session access JWT; opaque token with introspection; service-specific signed credentials | ecosystem interoperability, validation reliability, revocation, operational complexity | Accepted — Supabase Auth JWT; ES256-only allowlist |
| IAM-02 | Identity provider / issuer authority | NAEOS-owned IdP; existing organization IdP; external compatible issuer(s) | control, partner onboarding, key lifecycle, availability, deployment fit | Accepted — `naeos-crm-production` Supabase Auth project (`oruykkehukoiipldzcvc`) |
| IAM-03 | Issuer and audience policy | Single pinned issuer/audience; allowlisted multi-issuer/audience | trust boundary, client separation, environment separation | Accepted policy — pin exact values per deployment only after safe token claim evidence; values remain unverified |
| IAM-04 | Principal canonicalization | Map validated `iss + sub` to principal; alternate stable provider subject mapping | uniqueness, stability, collision avoidance, attribution | Accepted — canonical human identity is scoped to the deployment's pinned issuer namespace; the verified `sub` is the owner key used by existing persistence, with no `client_id` fallback. The issuer namespace is implicit in the partner-isolated deployment boundary (no composite issuer-sub schema migration in this phase). |
| IAM-05 | Client identity semantics | Separate OAuth client identifier from end-user principal; service principal for machine clients | clear attribution and distinction between acting client and actor | Accepted — initial integration human-user session only; machine/client identities excluded until separately modeled |
| IAM-06 | Tenant applicability and source of truth | Single-tenant MVP; tenant-scoped deployment; claims-based multi-tenancy | early-access operating model, isolation requirements, schema migration cost | Accepted — isolated deployment per partner; no shared multi-tenant trust domain in MVP |
| IAM-07 | Tenant claim and ownership binding | Trusted claim mapped to canonical tenant ID; server-side client/tenant registry; combination | prevent tenant claim spoofing and mismatched client/tenant bindings | Accepted — partner binding is server-side deployment context; never caller body/header/unverified claim |
| IAM-08 | Authorization vocabulary | Roles; capabilities/scopes; resource ownership checks; layered model | least privilege, ease of audit, explicit operation mapping | Accepted — NAEOS capabilities use dedicated `naeos_scopes` JWT claim. Supabase Custom Access Token Hook sources administrator-managed `app_metadata.naeos_scopes`, allowlists `intelligence:read` and `intelligence:write`, and writes the dedicated claim. Verifier ignores generic `scope`/`scopes` and user metadata, filters unknown capability values; absent/empty claim yields no grants (fail closed). SQL hook reference: `docs/implementation/supabase/NAEOS-CUSTOM-ACCESS-TOKEN-HOOK.sql`. Code and unit tests implemented; actual hook configuration/token evidence and operational rollout remain unverified and gated. |
| IAM-09 | Resource ownership semantics | Owner client; principal; tenant; mixed explicit ownership policy | request/result access control and consistent persistence enforcement | Accepted policy — owner is verified human `sub` within the isolated partner deployment; all persistence reads/writes must constrain by this owner key. Cross-owner negative tests and store-level verification remain pending |
| IAM-10 | Token validation requirements | Signature algorithm allowlist; JWKS key selection; issuer/audience; exp/nbf/iat; required claims | standards conformance, reject malformed/ambiguous claims, clock skew policy | Implemented baseline — require exact issuer/audience, non-empty `sub`, `exp`, and `iat`; reject `iat >= exp`; permit 60-second clock skew for `exp` and `nbf`, and reject `iat` more than 60 seconds in the future. CI evidence and maintainer acceptance pending |
| IAM-11 | Revocation and key rotation | Short token lifetime; revocation/introspection; denylist; key rotation with cache policy | compromise response, operational availability, acceptable latency | Implemented baseline — local JWT expiry remains the revocation bound for user sessions; verifier JWKS cache TTL is 5 minutes, unknown `kid` triggers immediate refresh, and JWKS fetch/validation failure rejects requests without stale-cache fallback. Emergency signing-key revocation requires provider-side revoke + cache propagation allowance and verifier cache purge/restart or explicit cache-busting capability; CI and maintainer acceptance pending. |
| IAM-12 | Identity propagation | Validated immutable request context; signed internal propagation if crossing trust boundary | prevent spoofed headers and preserve actor attribution | Implemented baseline — HTTP bearer verifier returns principal; Authenticator inserts it into a private typed request-context key only after successful validation and scope check. Handlers consume this context; caller-controlled `X-Client-ID`/`X-Principal` headers do not set identity. Existing handler ownership-isolation tests cover spoofed headers. No separate cross-service identity propagation protocol exists in current request flow; if a future trust boundary is crossed, use authenticated service identity and signed/validated propagation rather than forwarding trusted headers. CI/maintainer acceptance pending. |
| IAM-13 | Authorization denial and information disclosure | 401 for unauthenticated; 403/404 policy for authenticated but unauthorized resource | API contract consistency and non-disclosure of resource existence | Open |
| IAM-14 | Audit attribution | Persist issuer, canonical principal, client, tenant (if applicable), operation and decision reference | accountability, minimization, retention and queryability | Open |

## 3. Proposed security invariants (independent of provider choice)

The following are proposed invariants to carry into the accepted IAM specification:

- Identity claims are trusted only after cryptographic and semantic validation.
- Request identity must not be derived from caller-controlled identity headers.
- Principal identity must be stable and namespaced by issuer when relevant.
- Client identity and human/service principal are distinct concepts unless a decision explicitly establishes otherwise.
- Every protected resource access checks authorization server-side.
- Tenant context, if enabled, is derived from trusted identity or a server-side binding; never from unvalidated request payloads.
- Authorization fails closed when identity is missing, invalid, ambiguous, revoked, or outside its permitted scope.
- Token, credential, and secret material is not logged or exposed in error messages.
- Access to a resource must be scoped consistently in API handlers and persistence queries.
- Any authorization result used for consequential execution must be revalidated at the final enforcement boundary.

These invariants remain proposals pending review.

## 4. Minimum acceptance criteria for Phase 5

Before Phase 5 can be accepted, the approved decisions and specification must support tests for:

- missing, malformed, invalid-signature, expired, not-yet-valid, wrong-issuer and wrong-audience credentials;
- malformed or missing required claims and claim-type confusion;
- identity spoofing through `X-Client-ID`, `X-Principal`, tenant headers or request-body fields;
- principal/client/tenant binding mismatch;
- unauthorized operation and cross-resource/cross-tenant access (if tenancy is enabled);
- key rotation and defined revocation behavior;
- authorization denial without sensitive resource disclosure;
- identity attribution in persisted records and security audit events;
- fail-closed behavior during validator/JWKS/introspection unavailability, as applicable.

Test cases must map to requirement IDs and produce retained CI evidence.

## 5. Dependencies and non-decisions

- Do not select a concrete IdP, token format, tenant mode, role/scope taxonomy or revocation mechanism solely from this proposal.
- API Contract, Data Model, Security Requirements, Phase 5 implementation prompt, and applicable threat-model entries must be updated consistently after decisions are accepted.
- Provider-specific code must not begin until the relevant decision(s) are accepted by the maintainer.
- Phase acceptance remains explicit; this document does not authorize merging PR #8 or any Phase 5 implementation.

## 6. Review record

| Decision | Maintainer outcome | Rationale / ADR link | Date |
|---|---|---|---|
| IAM-01 | Accepted: Supabase Auth session access JWT; ES256 only (P-256) | `ADR-005-SUPABASE-AUTH-HOSTED-IDP.md` | 2026-11-16 |
| IAM-02 | Accepted: `naeos-crm-production` Supabase Auth project (`oruykkehukoiipldzcvc`, Singapore) is the IdP; `naeos.db` (`kqmhpsftijskesjdovjm`, Tokyo) remains the Intelligence database | `ADR-005-SUPABASE-AUTH-HOSTED-IDP.md` | 2026-11-11 |
| IAM-03 | Accepted policy — issuer/audience values pinned per deployment after safe metadata verification; values not yet verified | Supabase Auth project selection; no production env activation | 2026-11-16 |
| IAM-04–05 | Accepted — deployment-pinned issuer namespace + verified `sub` owner key; human-user-only MVP, client identity separate. Existing persistence stores the subject key; issuer namespace is implicit in deployment isolation. | Prevent client_id/sub conflation; machine principals deferred | 2026-11-16 |
| IAM-06–07 | Accepted — partner-isolated deployment and server-side partner binding | Prevent cross-partner trust / caller-controlled tenant context | 2026-11-16 |
| IAM-08–09 | Accepted policy — server-controlled capability scopes; verified subject as resource owner within isolated partner deployment | No user_metadata/authenticated-role grants; store ownership enforcement and tests still required | 2026-11-16 |
| IAM-10 | Implemented baseline — required claims and 60-second clock-skew policy encoded in verifier/tests; CI and acceptance evidence pending | `jwks_verifier.go` and tests; implementation does not close operational IAM gates | 2026-11-16 |
| IAM-11 | Implemented baseline — 5-minute verifier JWKS cache, immediate refresh on unknown `kid`, fail-closed on JWKS errors, and document user-session revocation bound as JWT expiry. Provider JWKS edge cache may propagate changes for up to 10 minutes; CI/maintainer acceptance pending | Supabase signing-key and session guidance; local verifier cache policy | 2026-11-16 |
| IAM-12 | Implemented baseline — authenticated principal propagated through private typed Go request context after bearer verification and scope enforcement; caller identity headers ignored. Existing spoofed-header and owner-isolation test present. CI/maintainer acceptance pending; cross-service propagation protocol deferred until architecture introduces a trust boundary | `internal/httpapi/auth.go`, `handler.go`, `handler_test.go` | 2026-11-16 |
| IAM-13–14 | Open — denial and audit controls remain gated | Require explicit implementation and evidence | Pending |
