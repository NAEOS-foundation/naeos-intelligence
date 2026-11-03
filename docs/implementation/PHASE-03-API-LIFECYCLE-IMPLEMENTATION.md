# Phase 3 — HTTP API and Request Lifecycle

**Status:** Implementation proposal — maintainer review required  
**Phase:** 3  
**Updated:** 2026-11-04

## Scope

Implement the first HTTP API surface defined in `docs/specifications/API-CONTRACT.md`:

- `GET /v1/health` (public, minimal response only)
- `POST /v1/intelligence/requests` (authenticated)
- `GET /v1/intelligence/requests/{request_id}` (authenticated, owner-scoped)
- `POST /v1/intelligence/requests/{request_id}/cancel` (authenticated, owner-scoped)

Add input validation, stable Problem Details-style errors, request correlation, deterministic idempotency, lifecycle transition enforcement, bounded request handling, cancellation propagation, graceful shutdown, and unit/HTTP contract/security tests.

## Authentication design adopted for implementation

Use OAuth 2.0 Client Credentials for service-to-service clients and short-lived JWT access tokens. Authentication is separated behind a verifier interface so identity-provider details remain replaceable.

JWT validation requirements:
- Allowlist asymmetric algorithms (initially RS256; never accept `none` or algorithm confusion).
- Validate signature against trusted JWKS keys, with bounded fetch timeouts and key refresh behavior.
- Validate issuer, audience, expiry, and not-before claims.
- Derive client principal from a stable client identity claim.
- Enforce operation scopes and per-request ownership; do not rely on unguessable request IDs alone.
- Return 401 for missing/invalid credentials and 403 for authenticated clients lacking required scope.
- Fail closed when identity configuration or JWKS verification is unavailable.
- Never log bearer tokens or raw credential-bearing headers.

Identity Provider vendor selection remains deferred. The verifier interface must not couple the API to a specific provider. JWKS endpoint trust/configuration and rotation operations must be documented before production use.

## Request and lifecycle behavior

- Request submission returns `202 Accepted` only after validation and admission into the service's request store; this does not claim execution or completion.
- The initial implementation may use a repository interface with an explicitly non-durable in-memory adapter pending Phase 4 persistence. This is development-only and must be clearly documented.
- Request identifiers are generated server-side.
- Idempotency keys are scoped to authenticated client identity. Same key and same normalized payload returns the original request reference; same key with a different payload returns `409 CONFLICT`. Retention is process-lifetime in the in-memory adapter only, and production retention is deferred to Phase 4.
- Retrieval and cancellation enforce resource ownership and return a non-disclosing `404 RESOURCE_NOT_FOUND` for unknown or inaccessible IDs.
- Cancellation is valid only for non-terminal requests; terminal-state conflicts return `409 CONFLICT`. It must not represent work as cancelled unless the orchestrator confirms cancellation.
- No model, tool, policy, or external side effect execution is introduced in this phase.

## API error envelope

Use fields `type`, `title`, `status`, `code`, `detail`, and `request_id`. Details must be safe and must not reflect sensitive input. Correlation ID is accepted only after strict format validation; otherwise generate a server ID.

## Operational safeguards

- Set bounded read-header, read, write, and idle timeouts.
- Use context cancellation for request processing.
- Gracefully shut down the HTTP server on SIGINT/SIGTERM with a bounded deadline.
- Limit JSON request body size and reject unknown fields/trailing JSON values.
- Health endpoint exposes only `{"status":"healthy"}`; no config, credentials, or infrastructure details.

## Deferred decisions and limitations

- PostgreSQL persistence, durable idempotency, and retention policy: Phase 4.
- Identity Provider vendor and client onboarding/secret rotation: separately decided before deployment.
- Full policy engine, approval workflow, model/provider integration, and governed runtime: later phases.
- OpenAPI contract may require maintainer reconciliation if normative Draft Proposed details conflict.

## Acceptance checklist

- [ ] OpenAPI and HTTP handlers align for the implemented endpoints.
- [ ] Missing/invalid authentication is rejected for protected operations.
- [ ] Client identity/scope and owner isolation are enforced.
- [ ] Invalid JSON, oversized bodies, and invalid fields produce stable safe errors.
- [ ] Idempotency same-key replay and payload-conflict cases are tested.
- [ ] Lifecycle transitions and cancellation races are tested.
- [ ] Request IDs/correlation propagate and sensitive headers are excluded from logs.
- [ ] Graceful shutdown and bounded timeouts are implemented.
- [ ] `gofmt`, `go vet ./...`, `go test ./...`, and `go build ./...` pass in CI.
- [ ] In-memory/non-durable limitation is prominently documented.
