# Phase 5 — Identity and Access Control Decision Worksheet

**Document ID:** NAEOS-INT-IAM-DEC-001  
**Version:** 0.1.0  
**Status:** Draft — Proposed; maintainer decisions required  
**Scope:** Decisions required before provider-specific Phase 5 implementation. This worksheet does not constitute an accepted ADR.

## 1. Decision gate

The Phase 5 Copilot prompt explicitly instructs implementers not to select an IdP or token format while unresolved. The repository already contains a concrete RS256/JWKS verifier scaffold; its presence does not accept its identity semantics or authorize provider-specific deployment configuration. Further provider-specific IAM work must remain gated until the decisions below are accepted. A green test suite does not substitute for decision acceptance.

## 2. Existing verifier evidence (not an IAM decision)

Static inspection of `internal/httpapi/jwks_verifier.go` found the following current behavior. This records implementation facts only and does not establish that the behavior is the accepted target:

- Requires HTTPS issuer and JWKS URL configuration, RS256 tokens, a trusted `kid`, valid signature, exact issuer, matching audience, and unexpired `exp`; a future non-zero `nbf` is rejected.
- Refreshes cached JWKS data after five minutes or immediately when a token presents an unknown `kid`; JWKS requests have a five-second client timeout and a one-megabyte response limit. Lookup and fetch failures reject the token.
- Uses `client_id` as `Principal.ClientID`, falling back to `sub` when `client_id` is blank. If both are present but differ, `client_id` wins.
- Accepts both space-delimited `scope` and array-valued `scopes`, combining their entries. The API's endpoint scopes and resource-owner checks are accepted by ADR-0004; claim parsing/precedence and the token-to-client mapping remain open.
- Does not validate `iat`, impose a maximum token lifetime, or implement token revocation/introspection. Tenant identity and server-side client-to-tenant binding are not represented; this is consistent with the accepted single-tenant MVP baseline and must be revisited only if shared tenancy is introduced.

The remaining token and identity-semantic gaps depend on IAM-01–05, IAM-10–11, and IAM-14. Do not resolve them by silently changing verifier behavior before the relevant decisions and contract updates are accepted. Reopen IAM-07 if shared tenancy becomes a requirement.

## 3. Decision register

| ID | Decision | Options to evaluate | Recommended evaluation criteria | State |
|---|---|---|---|---|
| IAM-01 | Authentication protocol and token format | OIDC/OAuth 2.0 JWT access token; opaque token with introspection; service-specific signed credentials | ecosystem interoperability, validation reliability, revocation, operational complexity | Open |
| IAM-02 | Identity provider / issuer authority | NAEOS-owned IdP; existing organization IdP; external compatible issuer(s) | control, partner onboarding, key lifecycle, availability, deployment fit | Open |
| IAM-03 | Issuer and audience policy | Single pinned issuer/audience; allowlisted multi-issuer/audience | trust boundary, client separation, environment separation | Open |
| IAM-04 | Principal canonicalization | Map validated `iss + sub` to principal; alternate stable provider subject mapping | uniqueness, stability, collision avoidance, attribution | Open |
| IAM-05 | Client identity semantics | Separate OAuth client identifier from end-user principal; service principal for machine clients | clear attribution and distinction between acting client and actor | Open |
| IAM-06 | Tenant applicability and source of truth | Single-tenant logical boundary per deployment for MVP; shared claims-based tenancy deferred | early-access operating model, isolation requirements, schema migration cost | Accepted for MVP by ADR-0003 |
| IAM-07 | Tenant claim and ownership binding | No tenant claim or caller-supplied tenant ID in the single-tenant MVP; revisit before shared tenancy | prevent tenant claim spoofing and mismatched client/tenant bindings | Deferred; not applicable to current baseline |
| IAM-08 | Authorization vocabulary | Roles; capabilities/scopes; resource ownership checks; layered model | least privilege, ease of audit, explicit operation mapping | Accepted for read-only API by ADR-0004: endpoint scopes plus server-side owner checks |
| IAM-09 | Resource ownership semantics | Owner client; principal; tenant; mixed explicit ownership policy | request/result access control and consistent persistence enforcement | Accepted for read-only API by ADR-0004: owner check uses verifier-derived `Principal.ClientID`; claim-to-client mapping remains open under IAM-01–05 |
| IAM-10 | Token validation requirements | Signature algorithm allowlist; JWKS key selection; issuer/audience; exp/nbf/iat; required claims | standards conformance, reject malformed/ambiguous claims, clock skew policy | Open |
| IAM-11 | Revocation and key rotation | Short token lifetime; revocation/introspection; denylist; key rotation with cache policy | compromise response, operational availability, acceptable latency | Open |
| IAM-12 | Identity propagation | Validated immutable request context; signed internal propagation if crossing trust boundary | prevent spoofed headers and preserve actor attribution | Accepted for in-process read-only API by ADR-0004; cross-service propagation remains out of scope |
| IAM-13 | Authorization denial and information disclosure | 401 for unauthenticated; 403/404 policy for authenticated but unauthorized resource | API contract consistency and non-disclosure of resource existence | Accepted for read-only API by ADR-0004 and API contract: 401 invalid/missing auth, 403 missing scope, 404 inaccessible resource |
| IAM-14 | Audit attribution | Persist issuer, canonical principal, client, tenant (if applicable), operation and decision reference | accountability, minimization, retention and queryability | Open |

## 4. Proposed security invariants (independent of provider choice)

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

These invariants remain proposals except for the read-only API subset accepted by ADR-0004.

## 5. Minimum acceptance criteria for Phase 5

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

## 6. Dependencies and non-decisions

- Do not select a concrete IdP, token format, role/scope taxonomy or revocation mechanism solely from this proposal. The MVP tenancy baseline is recorded separately in accepted ADR-0003.
- API Contract, Data Model, Security Requirements, Phase 5 implementation prompt, and applicable threat-model entries must be updated consistently after decisions are accepted.
- Provider-specific code must not begin until the relevant decision(s) are accepted by the maintainer.
- Phase acceptance remains explicit; this document does not authorize merging PR #8 or any Phase 5 implementation.

## 7. Review record

| Decision | Maintainer outcome | Rationale / ADR link | Date |
|---|---|---|---|
| IAM-01–05, IAM-10–11, IAM-14 | Pending | Partner, token lifecycle, and actor-audit decisions remain unresolved | Pending |
| IAM-06 | Accepted for MVP baseline | ADR-0003; no partner is selected, so this is a logical single-tenant boundary per deployment only | 2026-11-10 |
| IAM-07 | Deferred for MVP | ADR-0003; reopen before shared tenancy or tenant identity is introduced | 2026-11-10 |
| IAM-08–09, IAM-12–13 | Accepted for read-only API only | ADR-0004; scopes plus owner checks, server-derived request context, and non-disclosing denial behavior | 2026-11-10 |
