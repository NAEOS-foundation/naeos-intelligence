# Phase 0 — Repository Assessment and Engineering Baseline

**Repository:** `NAEOS-foundation/naeos-intelligence`  
**Assessment date:** 2026-11-10
**Status:** M1 bounded API/lifecycle contract accepted; broader specification and IAM decisions remain open
**Scope:** Static source/documentation assessment, local checks, and PR #67 CI run #248 evidence review. No deployment systems or local opt-in PostgreSQL integration tests were run as part of this update.

## 1. Executive summary

The repository contains a Go service and entrypoint, configuration, HTTP API handlers and JWT/JWKS verification scaffolding, in-memory and PostgreSQL persistence adapters, migration tooling, policy/approval prototype code, tests, a Makefile, and GitHub Actions CI.

The MVP surface remains read-only and must not be interpreted as approval for model-driven or consequential tool execution. The architecture lifecycle vocabulary and README roadmap sequencing were reconciled in merged PR #42. Route/OpenAPI conformance and shared Problem error-envelope checks are present. M1's bounded API/lifecycle contract was accepted by the maintainer on 2026-11-10 after PR #67 CI run #248 passed, including OpenAPI validation and PostgreSQL integration. Local `make fmt`, `make vet`, `make test`, `make build`, `govulncheck`, and module verification passed. Merged PRs #38–#41 also record PostgreSQL CI evidence and add JWKS validation tests.

Deployment-level IAM validation, lifecycle-worker behavior, operational/retention decisions, and Early Access deployment acceptance remain outstanding. ADR-0003 accepts a single logical tenant per deployment for the MVP and defers tenant claims/shared tenancy. Provider-specific IAM choices remain open in the Phase 5 worksheet. Static verifier review also records unresolved `client_id`/`sub` precedence, combined `scope`/`scopes` handling, and absent `iat` validation; these remain decision-gated, not silently accepted implementation defaults.

## 2. Current repository inventory

### Present
- Go module: `go.mod`, `go.sum`; module path `github.com/NAEOS-foundation/naeos-intelligence`; Go baseline `1.26.10`.
- Service entrypoint: `cmd/naeos-intelligence/main.go` and bootstrap test.
- Separate migration command: `cmd/naeos-intelligence-migrate/main.go`.
- Application packages: `internal/config`, `internal/governance`, `internal/httpapi`, `internal/model`, `internal/observability`, `internal/policy`, `internal/postgres`, `internal/runtime`.
- PostgreSQL migration: `migrations/00001_request_persistence.sql`.
- API contract: `api/openapi.yaml`.
- Makefile and GitHub Actions workflow: `Makefile`, `.github/workflows/ci.yml`.
- Unit tests and PostgreSQL integration test file(s); CI provisions PostgreSQL 16 and enables integration tests.
- Draft architecture, security, domain specifications, Phase 0–5 implementation notes, Copilot instructions, and staged prompts.

### Not established as complete / still requiring review
- Maintainer acceptance/ratification of normative draft specifications.
- Provider-specific IAM trust configuration, client/actor semantics, tenant isolation assumptions, revocation/key rotation, and deployment-level validation.
- Full asynchronous worker/scheduler lifecycle, transition race validation, and cooperative cancellation.
- Complete automated OpenAPI error-response coverage and end-to-end auth contract coverage.
- Production retention/deletion policy, operational migration and rollback procedures, observability/readiness controls, and early-access deployment validation.
- Formal accepted ADRs for decisions that materially constrain architecture; proposed technology direction is not itself an accepted decision.

These are based on repository files and their stated status; this document does not claim runtime or production validation.

## 3. Implementation snapshot

| Area | Repository evidence | Assessment |
|---|---|---|
| Engineering baseline | `go.mod`, command entrypoint, Makefile, CI | Present; CI runs go vet, tests, and build. |
| Configuration | `internal/config`, Phase 2 implementation note | Limited environment configuration implemented; secret/config expansion intentionally deferred. |
| HTTP API | `internal/httpapi`, OpenAPI, Phase 3 implementation note | Read-only MVP submission/retrieval/cancellation/analyze surface; local route/error checks and PR #67 CI run #248 passed. Approval operation removal is an accepted pre-Early-Access scope correction; M1 bounded API/lifecycle contract was accepted on 2026-11-10. |
| Authentication | Token verifier and JWKS verifier code/tests | RS256/JWKS scaffold validates signature, issuer, audience, `exp`, and optional `nbf`; claim semantics, `iat`/lifetime policy, revocation, deployment trust, and operational readiness remain open. Single-tenant-per-deployment is the accepted MVP boundary; tenant binding is deferred unless shared tenancy is introduced. See the Phase 5 worksheet's implementation evidence section. |
| Persistence | `internal/postgres`, migration command and SQL migration | PostgreSQL adapter and persistence implemented; retention and operational rollout require decisions. |
| Policy / approval | `internal/policy`, `internal/governance` | Prototype components exist; no public operational approval routes in read-only MVP. |
| Agent/model/tool execution | runtime/model scaffolding and draft specifications | Do not infer production-ready or authorized consequential execution. |

## 4. Gaps, inconsistencies, and risks

| ID | Finding | Risk / implication | Suggested action / disposition |
|---|---|---|---|
| P0-01 | Phase notes may contain differing acceptance/status language as implementation evolves | Project status may be misread as accepted/production-ready | Reconcile phase status notes against merged PRs, CI evidence, and explicit acceptance as a continuing maintenance activity. |
| P0-02 | IAM protocol, issuer, actor/client, token-validation, revocation and audit decisions remain open; current verifier uses provisional `client_id`/`sub` fallback and combined `scope`/`scopes`, and does not validate `iat` | Provider-specific implementation or production trust setup may encode assumptions or misattribute principals | Follow ADR-0003's single-tenant MVP boundary; resolve remaining IAM decisions before provider-specific hardening and maintain fail-closed behavior. |
| P0-03 | Architecture state model previously differed from Request Lifecycle vocabulary | State machine and API/persistence contract drift | **Documentation reconciliation completed in merged PR #42.** Before worker implementation, verify implementation behavior and accepted specifications remain aligned. |
| P0-04 | README roadmap previously sequenced Intelligence Core before Governance/Evidence foundations | Roadmap sequence could be interpreted inconsistently | **Roadmap framing corrected in merged PR #42** to put API, persistence, and governance foundations before Intelligence Core; preserve security gates. |
| P0-05 | OpenAPI previously omitted `500 INTERNAL_ERROR` on submit/get/cancel/analyze and analyze's `400`, `413`, and `422` responses, although handlers or middleware can return them | Clients may not know all documented error outcomes; contract validation can miss endpoint behavior | PR #67 declares these statuses and extends the existing status/code mapping check; local tests and CI run #248 passed. M1 bounded API/lifecycle acceptance was recorded on 2026-11-10. |
| P0-06 | Four documentation filenames contained malformed suffixes (resolved in docs-only cleanup PR #20) | Links/tools and prompt references could fail or be confusing | Normalized paths; separately verify internal links as needed. |
| P0-07 | Production operational requirements are not all demonstrated by repository-level CI | Passing unit/CI checks alone do not establish early-access readiness | Plan deployment-level validation, secrets/trust configuration, migration operations, monitoring, and E2E evidence. |

## 5. Recommended next milestone

Prioritize a bounded contract and acceptance-alignment milestone rather than adding agent execution capabilities:

1. Keep endpoint-specific OpenAPI error response declarations aligned with handler and middleware behavior; verify status/code mappings in CI.
2. Review lifecycle contract against persisted API behavior and define focused transition/concurrency tests within the current read-only boundary.
3. Resolve IAM worksheet decisions that are prerequisites for provider-specific and deployment-level work.
4. Define operational acceptance evidence for migrations/rollback, retention/deletion, observability/readiness, and early-access deployment.
5. Refresh phase status summaries after each accepted milestone.

Do not add autonomous tool execution, unrestricted shell access, repository mutation, production deployment, or external side effects as part of this milestone.

## 6. Acceptance and governance

- Treat architecture/specification documents marked Draft or Proposed as design input, not automatically ratified requirements.
- Keep read-only MVP boundaries in force.
- Policy or authorization failures must fail closed; do not bypass security gates.
- Any implementation phase should be limited to its scope, include actual test/check evidence, and report unresolved acceptance criteria explicitly.
- Production deployment and consequential actions require separate explicit approval.

## 7. Traceability snapshot

| Source | Current disposition |
|---|---|
| README / Project Vision / Scope | Project purpose and MVP scope; roadmap foundation sequencing updated in merged PR #42. |
| System Architecture | Draft Proposed; conceptual processing stages clarified against lifecycle vocabulary in merged PR #42. |
| Request Lifecycle | Draft Proposed; lifecycle worker and transition behavior not fully implemented/validated. |
| API Contract / OpenAPI | Existing contract draft; route conformance and endpoint-specific error coverage passed CI run #248. The bounded M1 API/lifecycle contract is accepted; revisit compatibility if an external consumer is identified before onboarding or release. |
| Phase 1–4 implementation notes | Describe implemented foundation; status/acceptance language should be reconciled with current main and CI. |
| Phase 5 IAM worksheet and decision brief | Open decisions; provider-specific work is gated. |
| Copilot instructions and staged prompts | Engineering guidance; do not substitute for accepted decisions or requirements. |

---

This assessment records the current Git tree and documentation snapshot. It does not claim local command execution, complete E2E testing, production validation, or maintainer acceptance of draft specifications.
