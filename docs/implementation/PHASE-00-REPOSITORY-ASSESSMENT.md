# Phase 0 — Repository Assessment and Engineering Baseline

**Repository:** `NAEOS-foundation/naeos-intelligence`  
**Assessment date:** 2026-11-05  
**Status:** Refreshed after merged documentation and test-evidence PRs — maintainer review required  
**Scope:** Read-only Git tree and documentation inspection. No application code, tests, or deployment systems were executed as part of this assessment refresh.

## 1. Executive summary

The repository contains a Go service and entrypoint, configuration, HTTP API handlers and JWT/JWKS verification scaffolding, in-memory and PostgreSQL persistence adapters, migration tooling, policy/approval prototype code, tests, a Makefile, and GitHub Actions CI.

The MVP surface remains read-only and must not be interpreted as approval for model-driven or consequential tool execution. The architecture lifecycle vocabulary and README roadmap sequencing were reconciled in merged PR #42. The shared error envelope and public error-code vocabulary are documented; PR #43 corrected the assessment to identify the remaining endpoint-specific OpenAPI response coverage and contract verification gap. Merged PRs #38–#41 also record PostgreSQL CI evidence and add JWKS validation tests.

Deployment-level IAM validation, lifecycle-worker behavior, contract conformance, operational/retention decisions, and early-access deployment acceptance remain outstanding. Provider-specific IAM choices remain open in the Phase 5 worksheet.

## 2. Current repository inventory

### Present
- Go module: `go.mod`, `go.sum`; module path `github.com/NAEOS-foundation/naeos-intelligence`; Go baseline `1.26.11` (security-patched uplift from `1.26.10`).
- Service entrypoint: `cmd/naeos-intelligence/main.go` and bootstrap test.
- Separate migration command: `cmd/naeos-intelligence-migrate/main.go`.
- Application packages: `internal/config`, `internal/governance`, `internal/httpapi`, `internal/intelligence`, `internal/model`, `internal/observability`, `internal/policy`, `internal/postgres`, `internal/runtime`.
- PostgreSQL migration: `migrations/00001_request_persistence.sql`.
- API contract: `api/openapi.yaml`.
- Makefile and GitHub Actions workflow: `Makefile`, `.github/workflows/ci.yml`.
- Unit tests and PostgreSQL integration test file(s); CI provisions PostgreSQL 16 and enables integration tests.
- Draft architecture, security, domain specifications, Phase 0–5 implementation notes, Copilot instructions, and staged prompts.

### Not established as complete / still requiring review
- Maintainer acceptance/ratification of normative draft specifications.
- Provider-specific IAM trust configuration, client/actor semantics, tenant isolation assumptions, revocation/key rotation, and deployment-level validation.
- Full asynchronous worker/scheduler lifecycle, transition race validation, and cooperative cancellation.
- Automated OpenAPI contract conformance and end-to-end auth contract coverage.
- Production retention/deletion policy, operational migration and rollback procedures, observability/readiness controls, and early-access deployment validation.
- Formal accepted ADRs for decisions that materially constrain architecture; proposed technology direction is not itself an accepted decision.

These are based on repository files and their stated status; this document does not claim runtime or production validation.

## 3. Implementation snapshot

| Area | Repository evidence | Assessment |
|---|---|---|
| Engineering baseline | `go.mod`, command entrypoint, Makefile, CI | Present; CI runs go vet, tests, and build. |
| Configuration | `internal/config`, Phase 2 implementation note | Limited environment configuration implemented; secret/config expansion intentionally deferred. |
| HTTP API | `internal/httpapi`, OpenAPI, Phase 3 implementation note | Read-only MVP request submission/retrieval/cancellation surface described; acceptance remains in progress. |
| Authentication | Token verifier and JWKS verifier code/tests | JWT/JWKS scaffolding exists; deployment trust and operational readiness remain open. |
