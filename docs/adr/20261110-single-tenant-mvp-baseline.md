# ADR-0003: Single-Tenant Logical Boundary for the Read-Only MVP

**Status:** Accepted
**Date:** 2026-11-10
**Author:** NAEOS Intelligence Engineering

## Context

NAEOS Intelligence has no Early Access partner yet. IAM decisions about an identity provider, token profile, issuer, actor/client semantics, and revocation cannot be grounded in partner requirements. At the same time, the MVP needs a bounded tenancy assumption so that implementation work does not silently introduce shared multi-tenancy.

The current API scopes persisted requests by authenticated `client_id`. The exact mapping from token claims to that client identifier remains provisional and is not decided by this ADR.

## Decision

- Treat each NAEOS Intelligence deployment as one logical tenant for the read-only MVP.
- Do not implement shared claims-based multi-tenancy or accept tenant identity from request headers or request bodies.
- This is a deployment/topology assumption, not a runtime-enforced tenant boundary. Do not claim production tenant isolation until trusted issuer/client onboarding and deployment separation are validated.
- Keep existing per-client resource ownership checks. Their relationship to authenticated principal and OAuth client identity remains open under IAM-04, IAM-05, and IAM-09.
- Preserve the provider-neutral `TokenVerifier` boundary and fail-closed behavior. The existing RS256/JWKS verifier remains a scaffold; this ADR does not select an IdP, token format/profile, issuer, audience, or revocation mechanism.
- Revisit the tenancy boundary before any deployment is shared across tenants or a partner requirement needs tenant-level isolation within one service instance.

## Rationale

There is no partner operating model to justify the extra identity registry, tenant claims, persistence scoping, migration, and cross-tenant test requirements of shared multi-tenancy. A one-tenant-per-deployment boundary keeps the MVP model explicit while leaving future identity choices open.

## Consequences

Positive:
- The MVP does not need tenant claims or tenant identifiers in request data.
- Shared cross-tenant access is outside the MVP baseline.
- Provider and partner selection can wait until deployment requirements are known.

Negative:
- Serving multiple tenants from one deployment is not supported by this baseline.
- Separate tenant deployments may add operational cost if the Early Access cohort grows.
- The current client identity mapping still requires IAM decisions and must not be treated as canonical principal identity.

## References

- `docs/implementation/PHASE-05-IAM-DECISION-WORKSHEET.md` — IAM-06 and IAM-07
- `docs/implementation/PHASE-05-IAM-MVP-DECISION-BRIEF.md`
- `docs/vision/PROJECT-SCOPE.md`
- `docs/security/THREAT-MODEL.md`
