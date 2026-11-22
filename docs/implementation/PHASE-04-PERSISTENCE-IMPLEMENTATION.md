# Phase 4 — PostgreSQL Persistence Implementation

**Status:** Implementation on branch for CI validation  
**Branch:** `phase-04-postgres-persistence`

## Scope

Implements durable requests, owner-scoped idempotency, append-only transitions, and evidence/audit persistence. Attempt and approval tables remain present for future worker/admin paths but are deliberately unavailable to the API runtime. Draft/Proposed specifications are not promoted to Accepted.

## Implementation

- PostgreSQL adapter uses pgx v5 and a bounded pool (max 12 connections; bounded connection lifetime and idle time).
- All Intelligence persistence and the Goose version ledger are isolated in the naeos_intelligence schema; public remains reserved for CRM staging.
- DATABASE_URL is the dedicated runtime login. Startup verifies role membership, schema ownership, search_path, RLS context and exact table privileges; it rejects superuser/BYPASSRLS, public schema CREATE, CRM/Control Plane/Identity table and sequence access, externally executable SECURITY DEFINER functions, and migration-ledger access.
- MIGRATOR_DATABASE_URL is required by the separate migration command. It must authenticate as a dedicated login member of the schema-scoped migrator group. The migration command rejects a legacy public.goose_db_version ledger rather than silently rerunning migrations.
- Request ownership is set transaction-locally through naeos.owner_id. RLS policies enforce owner scope for requests, transitions, and audit events as defense in depth against missing owner predicates. This custom GUC is set by the trusted API after JWT verification; it is not an independent boundary against a compromised runtime credential or arbitrary SQL execution.
- The API runtime has no privileges on request_attempts or approval_decisions until separate worker/approval principals and authorization contracts are implemented.
- Request creation, idempotency conflict/replay resolution, and initial transition event are transactionally coupled.
- Cancellation serializes on the request row and records the lifecycle event within the same transaction.
- Attempt persistence code remains in the repository, but the API runtime is intentionally denied access to request_attempts because the worker principal and owner-scoped worker lifecycle are not yet implemented.
- Goose migrations are applied through the separate `naeos-intelligence-migrate` command. The HTTP server does not run migrations automatically.
- Production startup fails if `DATABASE_URL` is absent. Development/test may use in-memory storage if no database URL is configured.
- PostgreSQL integration tests are gated by `NAEOS_POSTGRES_INTEGRATION=1`; CI provisions PostgreSQL 16 and executes them.

## Local usage

For local Docker Compose, `docker compose up` starts PostgreSQL, provisions local-only least-privilege roles/schema, applies Goose migrations with the migrator DSN, and then starts the API. For manual execution, set DATABASE_URL to the runtime login URL and MIGRATOR_DATABASE_URL to the separate migrator login URL. To apply migrations from repository root:

```sh
go run ./cmd/naeos-intelligence-migrate
```

Then run the service with DATABASE_URL only. Do not expose MIGRATOR_DATABASE_URL to the HTTP server. The local bootstrap credentials in docker/postgres/bootstrap.sql are development-only and must never be reused in staging or production. The Supabase schema/group-role bootstrap reference is docs/operations/INTELLIGENCE-SCHEMA-BOOTSTRAP.sql; it has not been applied to staging. Do not apply migrations to production without explicit deployment approval.

## Limits and unresolved decisions

- Retention and deletion durations remain unset; no automatic purge is implemented.
- Migration rollback is available through Goose Down but must be reviewed before operational use; append-only transition history and FK constraints intentionally protect integrity.
- The application API does not yet expose attempt inspection or transition-history retrieval endpoints. Worker and approval table privileges remain BLOCKED until their separate principals and side-effect tests are implemented.
- Specifications remain Draft/Proposed; acceptance remains a separate governance action.
