# Phase 4 — PostgreSQL Persistence Implementation

**Status:** Implementation on branch for CI validation  
**Branch:** `phase-04-postgres-persistence`

## Scope

Implements durable requests, client-scoped idempotency, sequential request attempts, and append-only transition history. Approval, policy-decision, result, evidence, and general audit entities remain deferred. Draft/Proposed specifications are not promoted to Accepted.

## Implementation

- PostgreSQL adapter uses pgx v5 and a bounded pool (max 12 connections; bounded connection lifetime and idle time).
- Request creation, idempotency conflict/replay resolution, and initial transition event are transactionally coupled.
- Cancellation serializes on the request row and records the lifecycle event within the same transaction.
- Attempt creation allocates a monotonically increasing per-request attempt number under parent-row lock; start and terminal finish transitions are conditional, preventing repeat state changes.
- Goose migrations are applied through the separate `naeos-intelligence-migrate` command. The HTTP server does not run migrations automatically.
- Production startup fails if `DATABASE_URL` is absent. Development/test may use in-memory storage if no database URL is configured.
- PostgreSQL integration tests are gated by `NAEOS_POSTGRES_INTEGRATION=1`; CI provisions PostgreSQL 16 and executes them.

## Local usage

Set `DATABASE_URL` to a PostgreSQL connection URL. To apply migrations from repository root:

```sh
go run ./cmd/naeos-intelligence-migrate
```

Then run the service with the same `DATABASE_URL`. Do not apply migrations to production without explicit deployment approval.

## Limits and unresolved decisions

- Retention and deletion durations remain unset; no automatic purge is implemented.
- Migration rollback is available through Goose Down but must be reviewed before operational use; append-only transition history and FK constraints intentionally protect integrity.
- The application API does not yet expose attempt inspection or transition-history retrieval endpoints.
- Specifications remain Draft/Proposed; acceptance remains a separate governance action.
