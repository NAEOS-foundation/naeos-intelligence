# Phase 3 — HTTP API and Request Lifecycle

**Status:** M1 implementation evidence substantially complete; formal contract acceptance pending  
**Phase:** 3  
**Updated:** 2026-11-05

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
- Rate limiting, concurrency/queue controls, readiness probes, and deployment-level auth validation remain outstanding. Structured OpenAPI contract tests were added in PR #34 (YAML parsing, operation/security/response assertions, and local reference resolution); full OpenAPI 3.1 schema validation was added in PR #36 and passed CI run #118.
- Approval workflow internals remain as non-public prototype scaffolding, but approval HTTP routes are not registered or documented in the MVP API. The MVP does not expose an operational approval workflow, consistent with `docs/vision/PROJECT-SCOPE.md`.
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
- [x] PostgreSQL lifecycle/idempotency integration test execution evidence: CI run #119 configured PostgreSQL 16 with `DATABASE_URL` and `NAEOS_POSTGRES_INTEGRATION=1`; the Go test step passed, including `internal/postgres` (see PR #38 evidence update, still awaiting maintainer acceptance).\n- [x] Concurrent cancellation integration coverage: 12 concurrent cancel calls against the same persisted request assert exactly one `request.cancelled` transition; executed in CI run #119.\n- [x] Full OpenAPI 3.1 schema validation added in PR #36 and passed CI run #118.\n- [ ] Worker-running lifecycle transitions and cancellation propagation remain deferred until the worker/runtime phase; deployment-level IAM/JWKS validation remains pending.
- [x] `go vet ./...`, `go test ./...`, and `go build ./...` pass in CI run #121 for PR #38 (documentation-only evidence update); PostgreSQL integration suite passed in CI run #119. PR #38 itself remains open pending maintainer contract acceptance and confirmation of any separately configured dependency/security governance gates.
