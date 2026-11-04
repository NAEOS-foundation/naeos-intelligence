# M1 API and Lifecycle Acceptance Checklist

**Status:** Evidence review updated; maintainer acceptance pending  
**Scope:** Read-only Early Access API contract  
**Purpose:** Provide a traceable checklist for maintainer acceptance after implementation and CI evidence are reviewed. This checklist does not itself constitute acceptance.

## Acceptance evidence

| Area | Required evidence | Status |
|---|---|---|
| Submission contract | OpenAPI request/response schema and handler behavior agree; validation errors use the documented Problem envelope | Evidence verified in handler tests for validation, accepted response, and Problem Details behavior; full maintainer contract review pending |
| Retrieval contract | Ownership isolation is demonstrated; resource fields and terminal states match the specification | Evidence verified in HTTP handler tests for owner access, cross-client 404 non-disclosure, identity-header spoofing resistance, and client-scoped retrieval; PostgreSQL integration execution evidence remains pending |
| Cancellation contract | Legal/illegal transitions are explicit; repeated and concurrent cancellation behavior is covered by PostgreSQL integration tests | Evidence verified: PR #30 merged; PostgreSQL-backed integration test includes repeated/concurrent cancellation and one-transition assertion; execution against configured PostgreSQL CI/service and maintainer review pending |
| Idempotency | Key scope, replay behavior, and conflict behavior are specified and tested | Evidence verified in handler and PostgreSQL integration test sources for replay, payload conflict, and client-scoped key; PostgreSQL integration execution evidence remains pending |
| Error contract | Stable error codes, HTTP status mapping, safe detail, and correlation ID are consistent | Evidence verified in public Problem envelope contract tests for status/code/media type/correlation ID and non-disclosing cross-client 404; maintainer review pending |
| OpenAPI coverage | Structured checks parse YAML and verify operations, scopes, response codes, and local reference resolution; full OpenAPI schema validation remains a gap | Full OpenAPI 3.1 validation added in PR #36 and passed CI run #118; structured operation/route checks also present; maintainer contract review pending |
| Compatibility | No unapproved breaking API changes; schema changes reviewed against consumers | OpenAPI-to-route conformance tests provide implementation evidence; consumer compatibility review/approval is not recorded and remains pending |
| Security boundary | Unauthorized access and cross-client non-disclosure are tested; no side-effect execution is enabled | Evidence verified in tests for missing bearer, insufficient scope, fail-closed verifier absence, owner isolation, identity-header spoofing, and absence of approval HTTP routes; deployment-level IAM/JWKS validation remains outside this bounded acceptance and pending separately |
| CI | Required checks including vet, tests, build, and dependency/security governance pass | Main CI run #118 completed successfully on commit `a37b36c2718707170a8cfec82365725e8bba6090`; maintainer acceptance and any separately configured required gates must still be confirmed |

## Maintainer acceptance record

Complete only after reviewing the evidence above.

- Contract version / commit: `a37b36c2718707170a8cfec82365725e8bba6090`
- Maintainer decision: **Not accepted — review pending**
- Decision owner: TBD
- Decision date: TBD
- Exceptions (must be explicit and justified): None recorded
- Follow-up issues: TBD

## Non-negotiable conditions

- “Pending” or “Partial” evidence is not a pass.
- Do not waive security or dependency governance gates.
- This acceptance applies only to the bounded read-only API/lifecycle contract; it does not approve IAM deployment decisions, worker runtime, autonomous actions, or production rollout.
- PostgreSQL integration tests are opt-in via `NAEOS_POSTGRES_INTEGRATION=1`; a test being present in source is not evidence that its configured database execution completed successfully.

---
