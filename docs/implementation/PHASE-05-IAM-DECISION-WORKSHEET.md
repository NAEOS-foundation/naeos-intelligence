# Phase 5 — Identity and Access Control Decision Worksheet

**Document ID:** NAEOS-INT-IAM-DEC-001  
**Version:** 0.1.0  
**Status:** Draft — Proposed; maintainer decisions required  
**Scope:** Decisions required before provider-specific Phase 5 implementation. This worksheet does not constitute an accepted ADR.

## 1. Decision gate

The Phase 5 Copilot prompt explicitly instructs implementers not to select an IdP or token format while unresolved. Therefore Phase 5 implementation must remain limited to provider-neutral interfaces and test scaffolding until the decisions below are accepted. A green test suite does not substitute for decision acceptance.

## 2. Decision register

| ID | Decision | Options to evaluate | Recommended evaluation criteria | State |
|---|---|---|---|---|
| IAM-01 | Authentication protocol and token format | OIDC/OAuth 2.0 JWT access token; opaque token with introspection; service-specific signed credentials | ecosystem interoperability, validation reliability, revocation, operational complexity | Open |
| IAM-02 | Identity provider / issuer authority | NAEOS-owned IdP; existing organization IdP; external compatible issuer(s) | control, partner onboarding, key lifecycle, availability, deployment fit | Open |
| IAM-03 | Issuer and audience policy | Single pinned issuer/audience; allowlisted multi-issuer/audience | trust boundary, client separation, environment separation | Open |
| IAM-04 | Principal canonicalization | Map validated `iss + sub` to principal; alternate stable provider subject mapping | uniqueness, stability, collision avoidance, attribution | Open |
| IAM-05 | Client identity semantics | Separate OAuth client identifier from end-user principal; service principal for machine clients | clear attribution and distinction between acting client and actor | Open |
| IAM-06 | Tenant applicability and source of truth | Single-tenant MVP; tenant-scoped deployment; claims-based multi-tenancy | early-access operating model, isolation requirements, schema migration cost | Open |
| IAM-07 | Tenant claim and ownership binding | Trusted claim mapped to canonical tenant ID; server-side client/tenant registry; combination | prevent tenant claim spoofing and mismatched client/tenant bindings | Open |
| IAM-08 | Authorization vocabulary | Roles; capabilities/scopes; resource ownership checks; layered model | least privilege, ease of audit, explicit operation mapping | Open |
| IAM-09 | Resource ownership semantics | Owner client; principal; tenant; mixed explicit ownership policy | request/result access control and consistent persistence enforcement | Open |
| IAM-10 | Token validation requirements | Signature algorithm allowlist; JWKS key selection; issuer/audience; exp/nbf/iat; required claims | standards conformance, reject malformed/ambiguous claims, clock skew policy | Open |
| IAM-11 | Revocation and key rotation | Short token lifetime; revocation/introspection; denylist; key rotation with cache policy | compromise response, operational availability, acceptable latency | Open |
| IAM-12 | Identity propagation | Validated immutable request context; signed internal propagation if crossing trust boundary | prevent spoofed headers and preserve actor attribution | Open |
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
| IAM-01 through IAM-14 | Pending | Pending | Pending |
