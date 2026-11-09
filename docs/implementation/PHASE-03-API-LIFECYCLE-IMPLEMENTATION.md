# Phase 3 — HTTP API and Request Lifecycle

**Status:** Read-only API scope decision accepted; maintainer approval recorded; M1 verification pending GitHub Actions evidence
**Phase:** 3  
**Updated:** 2026-11-10

## Implemented surface

- `GET /api/v1/health` and `GET /api/v1/ready` public probes with no sensitive details.
- `POST /api/v1/intelligence/requests` authenticated submission, strict JSON validation, size bound, 202 response.
- `GET /api/v1/intelligence/requests/{request_id}` authenticated retrieval with client ownership filtering.
- `POST /api/v1/intelligence/requests/{request_id}/cancel` authenticated cancellation of requests that have not entered execution.
- `POST /api/v1/intelligence/requests/{request_id}/analyze` authenticated, read-only analysis returning a deterministic intent, context provenance, proposed plan, runtime gate result, and policy assessment.

The analysis route uses rule-based classifier, context builder, and planner components. Bootstrap configures an empty model-provider registry and empty tool registry, so this path does not invoke a model provider or execute a tool. When PostgreSQL is configured, orchestration evidence events are recorded; these records do not authorize actions.

The HTTP API is implemented in `internal/httpapi`. Bootstrap starts a server with bounded timeouts and SIGINT/SIGTERM graceful shutdown. The default bind address is loopback (`127.0.0.1:8080`); override with `NAEOS_INTELLIGENCE_HTTP_ADDR`.

## Authentication and authorization boundary

The API uses a `TokenVerifier` boundary with a concrete JWKS verifier wired at bootstrap when `NAEOS_AUTH_ISSUER`, `NAEOS_AUTH_AUDIENCE`, and `NAEOS_AUTH_JWKS_URL` are configured. The verifier requires HTTPS issuer/JWKS endpoints and validates JWT signature, allowlisted algorithm, issuer, audience, expiry/not-before, and trusted JWKS key material. Validated claims supply client identity and scopes (`intelligence:read`, `intelligence:write`). Resource reads/cancel enforce client ownership and return a non-disclosing 404 for other clients.

When all three authentication environment variables are absent, the verifier remains unset and protected endpoints fail closed with HTTP 503. Partial or invalid configuration fails startup rather than silently disabling verification. Identity-provider trust configuration, key rotation behavior, and production operational readiness still require deployment-level validation.

## Request behavior

- Submissions start at `accepted`; they do not imply queued work, execution, or completion.
- Server-generated request IDs use cryptographic random bytes.
- `Idempotency-Key` is client-scoped: normalized equivalent payload replay returns the original request; payload mismatch returns 409. Purpose, context source fields and desired output labels are trimmed before digesting.
- Idempotency retention and request records are in-memory and lost on process restart when the memory adapter is selected. Bootstrap selects PostgreSQL when `DATABASE_URL` is set; production requires `DATABASE_URL` and fails startup without it. PostgreSQL durability/migration/operational readiness must still be validated for the target deployment.
- New MVP submissions use `analysis_only` and start at `accepted`. The current service has no worker that advances them through `queued`, `running`, or `completed`; cancellation of running work and cooperative propagation are therefore not implemented. The persistence adapter retains cancellation handling for `accepted`, `queued`, and `requires_approval` records.
- Unknown JSON fields, invalid payloads, and trailing JSON values are rejected. Body size is capped at 1 MiB.
- Errors use a Problem Details-like envelope with request correlation.

## Deferred

- Lifecycle worker/scheduler and cancellation propagation during running work: later runtime phases. PostgreSQL request transition rows are persisted, but a worker-driven lifecycle is not implemented.
- Identity-provider client registration/secret lifecycle, JWKS rotation operational validation, and production deployment hardening: Identity & Access implementation.
- A configurable in-process rate limiter and readiness endpoint are implemented. Deployment-level limit sizing, queue/concurrency controls, readiness semantics, and auth validation remain outstanding. Structured OpenAPI contract checks and full OpenAPI 3.1 validation exist; the current local contract edits have not been rerun through those checks.
- Approval workflow internals remain prototype scaffolding. Approval decisions are not exposed through the MVP HTTP API, consistent with `docs/vision/PROJECT-SCOPE.md`; governed execution and operational approval require separate acceptance.
- The analyze endpoint evaluates the read-only policy gate, but does not invoke a model, execute tools, or initiate consequential side effects.

## Acceptance checklist

- [x] Public minimal health handler.
- [x] HTTP request submission/retrieval/cancellation/analyze endpoints.
- [x] Request body schema validation and 1 MiB bound.
- [x] Client ownership isolation in the store interface and in-memory implementation.
- [x] Idempotency replay and conflict behavior.
- [x] Missing bearer token, insufficient scope, and oversized valid JSON payload rejection tests.
- [x] Protected routes fail closed if authentication verifier is not configured.
- [x] Bounded HTTP server timeouts and graceful shutdown.
- [x] Concrete JWKS verifier wired by environment configuration; JWT verifier tests cover claim validation.
- [ ] Deployment-level JWKS trust/rotation validation and end-to-end auth contract coverage.
- [x] PostgreSQL lifecycle/idempotency integration test execution evidence: CI run #119 configured PostgreSQL 16 with `DATABASE_URL` and `NAEOS_POSTGRES_INTEGRATION=1`; the Go test step passed, including `internal/postgres` (see PR #38 evidence update, still awaiting maintainer acceptance).
- [x] Concurrent cancellation integration coverage: 12 concurrent cancel calls against the same persisted request assert exactly one `request.cancelled` transition; executed in CI run #119.
- [x] Full OpenAPI 3.1 schema validation was added in PR #36 and passed CI run #118; current working-tree OpenAPI and route/error contract tests passed in `make test` on 2026-11-10.
- [ ] Worker-running lifecycle transitions and cancellation propagation remain deferred until the worker/runtime phase; deployment-level IAM/JWKS validation remains pending.
- [x] Local working-tree checks on 2026-11-10: `make fmt`, `make vet`, `make test`, and `make build` all passed; `govulncheck ./...` reported no vulnerabilities after dependency updates; the repository secret scan passed; and `go mod verify` passed. `NAEOS_POSTGRES_INTEGRATION` was not enabled; opt-in PostgreSQL integration tests were not part of this run. Historical PostgreSQL integration evidence is recorded for CI run #119. Maintainer approval is recorded; GitHub Actions has not run against this change set, so M1 verification remains pending.
