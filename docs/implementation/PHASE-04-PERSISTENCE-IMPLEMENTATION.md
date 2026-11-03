# Phase 4 — Persistence Implementation Baseline

**Status:** In progress — approved limited scope, implementation branch  
**Branch:** `phase-04-postgres-persistence`

## Approved scope

This phase implements only durable request records, client-scoped idempotency, request attempts, and append-only transition history. Approval, policy decision, result, evidence, and general audit tables are intentionally excluded until their domain phases and specifications are accepted.

## Migration baseline

Migration `00001_request_persistence.sql` defines:
- `intelligence_requests`: stable UUID, client ownership, request payload fields, lifecycle state, client-scoped idempotency key and digest, UTC timestamps.
- `request_transitions`: request lifecycle history, append-only trigger, foreign key and lookup index.
- `request_attempts`: immutable attempt identity and unique sequential numbering per request.
- Integrity checks for enums, lengths, digest/key pairing, attempt numbering, and timestamp ordering.

Tenant scope is omitted because the API's current trusted principal carries client identity only; client-supplied tenant identifiers are not trusted.

## Explicitly unresolved

- Retention and deletion durations remain unset; no automatic purging is implemented.
- The Proposed data-model and lifecycle specifications are not being promoted to Accepted by this code change.
- The migration is a schema baseline; PostgreSQL adapter wiring, bounded pgx pool, migration runner, transaction-level repository operations, and integration tests remain required before Phase 4 can be called complete.

## Safety

- Migration is intended for local/test setup; never run against production without explicit deployment approval.
- SQL values are intended to be parameterized by repository implementation.
- No request payload is logged.
