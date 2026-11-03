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

The API exposes a `TokenVerifier` interface intended for the approved OAuth2 Client Credentials/JWT direction. It requires client identity and operation scopes (`intelligence:read`, `intelligence:write`). Resource reads/cancel enforce client ownership and respond with a non-disclosing 404 for other clients.

**Important current limitation:** No concrete JWT/JWKS verifier is wired in this phase. Protected endpoints fail closed with HTTP 503 while the verifier is unset. This prevents accidental unauthenticated access. Identity provider integration and verifier implementation should be delivered through the Identity & Access phase or an explicitly approved Phase 3 extension before protected API use.

## Request behavior

- Submissions start at `accepted`; they do not imply queued work, execution, or completion.
- Server-generated request IDs use cryptographic random bytes.
- `Idempotency-Key` is client-scoped: normalized equivalent payload replay returns the original request; payload mismatch returns 409. Purpose, context source fields and desired output labels are trimmed before digesting.
- Idempotency retention and request records are in-memory and lost on process restart. This adapter is not production-durable.
- Cancellation is supported only while status is `accepted`, `queued`, or `requires_approval`. This prototype has no worker execution, so running cancellation/cooperative propagation is not yet applicable.
- Unknown JSON fields, invalid payloads, and trailing JSON values are rejected. Body size is capped at 1 MiB.
- Errors use a Problem Details-like envelope with request correlation.

## Deferred

- PostgreSQL persistence and durable idempotency: Phase 4.
- Concrete JWT verification, JWKS trust/rotation configuration, client registration/secret lifecycle: Identity & Access implementation.
- Real lifecycle worker/scheduler, transition history, cancellation propagation during running work: subsequent runtime phases.
- Rate limiting, concurrency/queue controls, readiness probes and full OpenAPI contract conformance remain outstanding.
- No model/tool/policy integration or consequential side-effect execution is introduced.

## Acceptance checklist

- [x] Public minimal health handler.
- [x] HTTP request submission/retrieval/cancellation endpoints.
- [x] Request body schema validation and 1 MiB bound.
- [x] Client ownership isolation in the store interface and in-memory implementation.
- [x] Idempotency replay and conflict behavior.
- [x] Missing bearer token, insufficient scope, and oversized valid JSON payload rejection tests.
- [x] Protected routes fail closed if authentication verifier is not configured.
- [x] Bounded HTTP server timeouts and graceful shutdown.
- [ ] Concrete JWT verifier, scopes from validated claims, and auth negative/positive contract coverage.
- [ ] Running lifecycle transitions, cancellation race tests, durable storage and OpenAPI checks.
- [x] `gofmt`, `go vet ./...`, `go test ./...`, and `go build ./...` pass in CI (run #26).
