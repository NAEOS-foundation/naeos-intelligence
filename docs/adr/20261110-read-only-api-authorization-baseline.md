# ADR-0004: Provider-Neutral Authorization Baseline for the Read-Only API

**Status:** Accepted
**Date:** 2026-11-10
**Author:** NAEOS Intelligence Engineering

## Context

The read-only MVP has protected HTTP operations and request ownership checks, but the IAM worksheet still leaves authorization vocabulary, ownership enforcement, identity propagation, and denial behavior open. No Early Access partner or issuer has been selected, so provider-specific token and principal semantics remain undecided.

The current verifier creates a server-side `Principal` only after token signature and claim validation. It places a `client_id` claim into `Principal.ClientID`, falling back to `sub` when `client_id` is absent. That claim mapping remains provisional and is not accepted by this ADR.

## Decision

- For the bounded read-only API, authorization uses both endpoint scopes and server-side resource ownership checks. The API currently maps `intelligence:read` and `intelligence:write` to protected operations.
- Resource reads and mutations are scoped to the `Principal.ClientID` produced by the configured credential verifier. Callers cannot choose or override that owner through request fields or identity headers.
- Within a request, validated identity is propagated through server-created request context. Caller-supplied identity headers are not trusted.
- Return 401 for missing or invalid authentication, 403 for an authenticated caller lacking the required operation scope, and 404 when a resource is missing or owned by another client, preserving the existing non-disclosure contract.
- This ADR covers only the read-only HTTP API. It does not authorize model/tool execution or define runtime authorization for consequential actions.
- The mapping from token claims to `Principal.ClientID`, canonical principal identity, service actor semantics, and durable actor audit attribution remain open under IAM-01–05 and IAM-14. Revisit this ADR if those decisions change.

## Rationale

The existing API already implements scope checks and client-scoped ownership, and its tests cover spoofed identity headers and cross-client non-disclosure. Recording this limited contract makes those boundaries explicit without selecting an IdP, issuer, token profile, or shared tenancy model.

## Consequences

Positive:
- Protected read-only operations require both the endpoint permission and resource ownership.
- Authorization does not depend on caller-provided identity data.
- Missing-resource and cross-client behavior has a stable non-disclosure response.

Limitations:
- `Principal.ClientID` claim mapping is still provisional; a valid token may currently use `sub` as its fallback.
- The API model does not yet distinguish a human or service actor from the OAuth client.
- This decision does not satisfy Phase 5 identity attribution, token lifecycle, revocation, or deployment trust acceptance criteria.

## References

- `docs/implementation/PHASE-05-IAM-DECISION-WORKSHEET.md` — IAM-08, IAM-09, IAM-12, IAM-13
- `docs/specifications/API-CONTRACT.md` — protected routes, ownership, and error contract
- `docs/security/SECURITY-REQUIREMENTS.md` — SEC-AUTH-001, SEC-AC-001
- `docs/adr/20261110-single-tenant-mvp-baseline.md` — single-tenant MVP deployment assumption
