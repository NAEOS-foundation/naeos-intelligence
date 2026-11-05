# Phase 0 — Repository Assessment and Engineering Baseline

**Repository:** `NAEOS-foundation/naeos-intelligence`  
**Assessment date:** 2026-11-04  
**Assessment baseline:** `main` at `b843009040bf1fe55fb4ffe73f77be8cb6e45082`  
**Status:** Refreshed inventory — maintainer review required  
**Scope:** Read-only Git tree and documentation inspection. No application code, tests, or deployment systems were executed as part of this refresh.

## 1. Executive summary

The repository has progressed beyond a documentation-only baseline. The current `main` tree contains a Go module and service entrypoint, configuration, HTTP API handlers and JWT/JWKS verification scaffolding, in-memory and PostgreSQL persistence adapters, migration tooling, policy/approval prototype code, tests, a Makefile, and GitHub Actions CI.

The MVP surface remains read-only and must not be interpreted as approval for model-driven or consequential tool execution. Phase 3 implementation notes still identify deployment-level IAM validation, lifecycle-worker behavior, contract conformance, and related acceptance work as outstanding. Phase 4 notes describe PostgreSQL persistence implementation but also leave operational and retention decisions unresolved. Provider-specific IAM choices remain open in the Phase 5 worksheet.

This refreshed assessment replaces the earlier inventory statements that no Go source, module, CI, or persistence baseline existed. Those statements were accurate only for the earlier assessment snapshot.

## 2. Current repository inventory (observed in main tree)

### Present
- Go module: `go.mod`, `go.sum`; module path `github.com/NAEOS-foundation/naeos-intelligence`; Go baseline `1.26.10`.
- Service entrypoint: `cmd/naeos-intelligence/main.go` and bootstrap test.
- Separate migration command: `cmd/naeos-intelligence-migrate/main.go`.
- Application packages: `internal/config`, `internal/governance`, `internal/httpapi`, `internal/model`, `internal/observability`, `internal/policy`, `internal/postgres`, `internal/runtime`.
- PostgreSQL migration: `migrations/00001_request_persistence.sql`.
- API contract: `api/openapi.yaml`.
- Makefile and GitHub Actions workflow: `Makefile`, `.github/workflows/ci.yml`.
- Unit tests and PostgreSQL integration test file(s); CI config provisions PostgreSQL 16 and enables integration tests.
- Draft architecture, security, domain specifications, Phase 0–5 implementation notes, Copilot instructions, and staged prompts.

### Not established as complete / still requiring review
- Maintainer acceptance/ratification of normative draft specifications.
- Provider-specific IAM trust configuration, client/actor semantics, tenant isolation assumptions, revocation/key rotation and deployment-level validation.
- Full asynchronous worker/scheduler lifecycle, transition race validation and cooperative cancellation.
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
| Persistence | `internal/postgres`, migration command and SQL migration | PostgreSQL adapter and persistence implemented; retention and operational rollout require decisions. |
| Policy / approval | `internal/policy`, `internal/governance` | Prototype components exist; no public operational approval routes in read-only MVP. |
| Agent/model/tool execution | runtime/model scaffolding and draft specifications | Do not infer production-ready or authorized consequential execution. |

## 4. Gaps, inconsistencies, and risks

| ID | Finding | Risk / implication | Suggested action |
|---|---|---|---|
| P0-01 | Current implementation has multiple phases represented in code, while phase notes contain differing acceptance/status language | Project status may be misread as accepted/production-ready | Refresh phase status notes against merged PRs, CI evidence, and explicit acceptance. |
| P0-02 | IAM worksheet decisions remain open | Provider-specific implementation or production trust setup may encode assumptions | Resolve Phase 5 IAM decisions before provider-specific hardening; maintain fail-closed behavior. |
| P0-03 | System Architecture state labels differ from Request Lifecycle states (e.g. `validated`, `contextualizing`, `planning`, `authorized` versus `accepted`, `queued`, `running`) | State machine and API/persistence contract drift | Reconcile lifecycle vocabulary in a deliberate docs/spec review before worker implementation. |
| P0-04 | README's high-level roadmap puts Intelligence Core before Governance/Evidence, while staged prompts place persistence/IAM/policy gates before agent execution | Roadmap sequence can be interpreted inconsistently | Align README roadmap with the staged implementation plan; preserve security-first gates and clarify phase labels. |
| P0-05 | Error envelope and shared public error-code vocabulary are documented in `ERROR-HANDLING-SPECIFICATION.md` and `API-CONTRACT.md`, but OpenAPI does not yet declare every listed status/code pair for each operation | Endpoint-level error behavior may drift from the shared contract; clients may lack complete response coverage | Close endpoint-specific OpenAPI response coverage gaps and verify status/code mapping through contract tests. |
| P0-06 | Four documentation filenames contained malformed suffixes (resolved in docs-only cleanup PR #20) | Links/tools and prompt references could fail or be confusing | Normalized paths; separately verify internal links as needed. |
| P0-07 | Production operational requirements are not all demonstrated by repository-level CI | Passing unit/CI checks alone do not establish early-access readiness | Plan deployment-level validation, secrets/trust configuration, migration operations, monitoring and E2E evidence. |

## 5. Recommended next milestone

Prioritize a bounded documentation and acceptance-alignment milestone rather than adding agent execution capabilities:

1. Reconcile lifecycle state vocabulary across architecture, lifecycle specification, API contract, and persistence behavior.
2. Refresh README and phase status summaries so roadmap and implementation evidence agree.
3. Reconcile endpoint-specific OpenAPI error response declarations with the shared error vocabulary and add contract tests for the remaining status/code mappings.
4. Resolve the IAM worksheet decisions that are prerequisites for provider-specific and deployment-level work.
5. After these review gates, define targeted contract, lifecycle, and deployment validation tasks.

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
| README / Project Vision / Scope | Project purpose and MVP scope; some roadmap/status alignment work remains. |
| System Architecture | Draft Proposed; state model requires reconciliation with lifecycle spec. |
| Request Lifecycle | Draft Proposed; lifecycle worker and transition behavior not fully implemented/validated. |
| API Contract / OpenAPI | Existing contract draft; conformance automation remains outstanding. |
| Phase 1–4 implementation notes | Describe substantial implemented foundation; status/acceptance language should be reconciled with current main and CI. |
| Phase 5 IAM worksheet and decision brief | Open decisions; provider-specific work is gated. |
| Copilot instructions and staged prompts | Engineering guidance; do not substitute for accepted decisions or requirements. |

---

This assessment records the current Git tree and documentation snapshot. It does not claim local command execution, complete E2E testing, production validation, or maintainer acceptance of draft specifications.
