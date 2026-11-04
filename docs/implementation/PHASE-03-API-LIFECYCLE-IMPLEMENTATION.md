# Phase 3 — HTTP API and Request Lifecycle

**Status:** Implementation in progress — draft PR for maintainer review  
**Phase:** 3  
**Updated:** 2026-11-04

## Implemented surface

- `GET /api/v1/health` public health response with no sensitive details.
- `POST /api/v1/intelligence/requests` authenticated submission, strict JSON validation, size bound, 202 response.
- `GET /api/v1/intelligence/requests/{request_id}` authenticated retrieval with client ownership filtering.
- `POST /api/v1/intelligence/requests/{request_id}/cancel` authenticated cancellation of requests that have not entered execution.

The HTTP API is implemented in `internal/httpapi`. Bootstrap starts a server with bounded timeouts and SIGINT/SIGTERM graceful shutdown. The default bind address is loopback (`127.0.0.1:8080`); override with `NAEOS_INTELLIGENCE_HTTP_ADDR`.

## Authentication and authorization boundary

The API uses a `TokenVerifier` boundary with a concrete JWKS verifier wired at bootstrap when `NAEOS_AUTH_ISSUER`, `NAEOS_AUTH_AUDIENCE`, and `NAEOS_AUTH_JWKS_URL` are configured. The verifier requires HTTPS issuer/JWKS endpoints and validates JWT signature, allowlisted algorithm, issuer, audience, expiry/not-before, and trusted JWKS key material. Validated claims supply client identity and scopes (`intelligence:read`, `intelligence:write`). Resource reads/cancel enforce client ownership and return a non-disclosing 404 for other clients.

When all three authentication environment variables are absent, the verifier remains unset and protected endpoints fail closed with HTTP 503. Partial or invalid configuration fails startup rather than silently disabling verification. Identity-provider trust configuration, key rotation behavior, and production operational readiness still require deployment-level validation.

## Request behavior

- Submissions start at `accepted`; they do not imply queued work, execution, or completion.
- Server-generated request IDs use cryptographic random bytes.
- `Idempotency-Key` is client-scoped: normalized equivalent payload replay returns the original request; payload mismatch returns 409. Purpose, context source fields and desired output labels are trimmed before digesting.
- Idempotency retention and request records are in-memory and lost on process restart when the memory adapter is selected. Bootstrap selects PostgreSQL when `DATABASE_URL` is set; production requires `DATABASE_URL` and fails startup without it. PostgreSQL durability/migration/operational readiness must still be validated for the target deployment.
- Cancellation is supported only while status is `accepted`, `queued`, or `requires_approval`. This prototype has no worker execution, so running cancellation/cooperative propagation is not yet applicable.
- Unknown JSON fields, invalid payloads, and trailing JSON values are rejected. Body size is capped at 1 MiB.
- Errors use a Problem Details-like envelope with request correlation.

## Deferred

- Lifecycle worker/scheduler, durable state-transition history, and cancellation propagation during running work: later runtime phases.
- Identity-provider client registration/secret lifecycle, JWKS rotation operational validation, and production deployment hardening: Identity & Access implementation.
- Real lifecycle worker/scheduler, transition history, cancellation propagation during running work: subsequent runtime phases.
- Rate limiting, concurrency/queue controls, readiness probes, and automated OpenAPI contract conformance remain outstanding.
- Approval endpoints are currently registered in the HTTP handler as prototype scaffolding, but are not included in the MVP OpenAPI contract. This is a scope inconsistency against `docs/vision/PROJECT-SCOPE.md` (which states the MVP has no operational approval workflow); these routes must be removed or separately accepted before the MVP can be considered scope-compliant.
- No model/tool/policy integration or consequential side-effect execution is introduced through the documented MVP API.

## Acceptance checklist

- [x] Public minimal health handler.
- [x] HTTP request submission/retrieval/cancellation endpoints.
- [x] Request body schema validation and 1 MiB bound.
- [x] Client ownership isolation in the store interface and in-memory implementation.
- [x] Idempotency replay and conflict behavior.
- [x] Missing bearer token, insufficient scope, and oversized valid JSON payload rejection tests.
- [x] Protected routes fail closed if authentication verifier is not configured.
- [x] Bounded HTTP server timeouts and graceful shutdown.
- [x] Concrete JWKS verifier wired by environment configuration; JWT verifier tests cover claim validation.
- [ ] Deployment-level JWKS trust/rotation validation and end-to-end auth contract coverage.
- [ ] Running lifecycle transitions, cancellation race tests, durable storage and OpenAPI checks.
- [x] `gofmt`, `go vet ./...`, `go test ./...`, and `go build ./...` pass in CI (run #26).
