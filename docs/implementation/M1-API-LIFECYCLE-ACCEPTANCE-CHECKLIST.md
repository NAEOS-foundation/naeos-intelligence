# M1 API and Lifecycle Acceptance Checklist

**Status:** Proposed; not accepted
**Scope:** Read-only Early Access API contract
**Purpose:** Provide a traceable checklist for maintainer acceptance after implementation and CI evidence are reviewed. This checklist does not itself constitute acceptance.

## Acceptance evidence

| Area | Required evidence | Status |
|---|---|---|
| Submission contract | OpenAPI request/response schema and handler behavior agree; validation errors use the documented Problem envelope | Pending review |
| Retrieval contract | Ownership isolation is demonstrated; resource fields and terminal states match the specification | Pending review |
| Cancellation contract | Legal/illegal transitions are explicit; repeated and concurrent cancellation behavior is covered by integration tests | Pending CI for PR #30 |
| Idempotency | Key scope, replay behavior, and conflict behavior are specified and tested | Pending review |
| Error contract | Stable error codes, HTTP status mapping, safe detail, and correlation ID are consistent | Pending review |
| OpenAPI coverage | Automated checks cover operations and referenced schemas; limitations of fragment checks are documented | Partial; PR #28 merged |
| Compatibility | No unapproved breaking API changes; schema changes reviewed against consumers | Pending review |
| Security boundary | Unauthorized access and cross-client non-disclosure are tested; no side-effect execution is enabled | Pending review |
| CI | Required checks including vet, tests, build, and dependency/security governance pass | Pending PR #30 CI and final review |

## Maintainer acceptance record

Complete only after reviewing the evidence above.

- Contract version / commit: TBD
- Maintainer decision: **Not accepted**
- Decision owner: TBD
- Decision date: TBD
- Exceptions (must be explicit and justified): None recorded
- Follow-up issues: TBD

## Non-negotiable conditions

- “Pending” or “Partial” evidence is not a pass.
- Do not waive security or dependency governance gates.
- This acceptance applies only to the bounded read-only API/lifecycle contract; it does not approve IAM deployment decisions, worker runtime, autonomous actions, or production rollout.
