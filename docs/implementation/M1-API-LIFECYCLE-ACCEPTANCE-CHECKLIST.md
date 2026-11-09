# M1 API and Lifecycle Acceptance Checklist

**Status:** API scope decision accepted; maintainer approval recorded; M1 verification pending required CI evidence
**Scope:** Read-only Early Access API contract  
**Purpose:** Provide a traceable checklist for maintainer acceptance after implementation and CI evidence are reviewed. This checklist does not itself constitute acceptance.

## Acceptance evidence

| Area | Required evidence | Status |
|---|---|---|
| Submission contract | OpenAPI request/response schema and handler behavior agree; validation errors use the documented Problem envelope | Evidence verified in handler tests for validation, accepted response, and Problem Details behavior; full maintainer contract review pending |
| Retrieval contract | Ownership isolation is demonstrated; resource fields and terminal states match the specification | Evidence verified in HTTP handler tests for owner access, cross-client 404 non-disclosure, identity-header spoofing resistance, and client-scoped retrieval; PostgreSQL integration test executed successfully in CI run #119 |
| Cancellation contract | Legal/illegal transitions are explicit; repeated and concurrent cancellation behavior is covered by PostgreSQL integration tests | Evidence verified: PR #30 merged; PostgreSQL-backed integration test includes repeated/concurrent cancellation and one-transition assertion; package integration tests passed in CI run #119; maintainer review pending |
| Idempotency | Key scope, replay behavior, and conflict behavior are specified and tested | Evidence verified in handler and PostgreSQL integration test sources for replay, payload conflict, and client-scoped key; PostgreSQL integration test executed successfully in CI run #119 |
| Error contract | Stable error codes, HTTP status mapping, safe detail, and correlation ID are consistent | Evidence verified in public Problem envelope contract tests for status/code/media type/correlation ID and non-disclosing cross-client 404; maintainer review pending |
| OpenAPI coverage | Structured checks parse YAML and verify operations, scopes, response codes, and local reference resolution; full OpenAPI schema validation is required | Local `make test` passed on 2026-11-10, including OpenAPI 3.1 validation, route conformance, and the updated error-response map after removing the approval operation and declaring missing responses. Maintainer contract review remains pending. |
| Compatibility | No unapproved breaking API changes; schema changes reviewed against consumers | Approval operation removal is accepted as a pre-Early-Access MVP scope correction: there are no known partners or in-repository consumers. External consumer inventory is unavailable; reassess before onboarding or release if an external consumer is identified. |
| Security boundary | Unauthorized access and cross-client non-disclosure are tested; no side-effect execution is enabled | Local tests passed for missing bearer, insufficient scope, fail-closed verifier absence, owner isolation, identity-header spoofing, and approval-route non-exposure. Deployment-level IAM/JWKS validation remains separate and pending. |
| Local checks | Format, vet, unit/integration suite enabled by the default target, and build succeed | On 2026-11-10, `make fmt`, `make vet`, `make test`, and `make build` all passed. `NAEOS_POSTGRES_INTEGRATION` was not enabled, so opt-in PostgreSQL integration tests were not included. |
| CI / governance | Required checks including vet, tests, build, and dependency/security governance pass | On 2026-11-10, local `make fmt`, `make vet`, `make test`, and `make build` passed; `govulncheck ./...` reported no vulnerabilities after updating `pgx` and `x/text`; the revised workflow secret scan passed; and `go mod verify` passed. PostgreSQL integration was not enabled locally. GitHub Actions has not run against this uncommitted working tree, so required CI evidence and maintainer milestone acceptance remain pending. |

## Maintainer acceptance record

Complete only after reviewing the evidence above.

- Contract version / commit: `d48d0f94d824918775aec40a5dc41d6b784bffce` (PR #37 merged; evidence execution in CI run #119)
- API scope decision: **Accepted — approval operation is outside the read-only MVP API**
- Maintainer approval: **Granted by the project maintainer in this conversation**
- M1 milestone acceptance: **Pending — required GitHub Actions evidence for this change set is not yet available**
- Decision owner: Project maintainer
- Decision date: 2026-11-10
- Exceptions (must be explicit and justified): No partner or in-repository consumer is known. External consumer inventory is unavailable, so reassess compatibility before onboarding or release if an external consumer is identified. This scope decision does not waive any security or dependency gate.
- Follow-up issues: Run required GitHub Actions checks on the reviewed change set; retain M1 as pending until all mandatory evidence is green.

## Non-negotiable conditions

- “Pending” or “Partial” evidence is not a pass.
- Do not waive security or dependency governance gates.
- This acceptance applies only to the bounded read-only API/lifecycle contract; it does not approve IAM deployment decisions, worker runtime, autonomous actions, or production rollout.
- PostgreSQL integration tests are opt-in via `NAEOS_POSTGRES_INTEGRATION=1`; CI run #119 explicitly enabled this variable and supplied a PostgreSQL 16 service, and its Go test step passed.

---
