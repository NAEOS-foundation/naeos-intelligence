# NAEOS Intelligence — Early Access Roadmap

**Status:** Proposed execution roadmap  
**Target:** Bounded, read-only Engineering Intelligence Early Access  
**Planning horizon:** Indicative 6–8 weeks; estimates depend on acceptance decisions, CI feedback, and scope  
**Last reviewed:** 2026-11-04

## Objective

Deliver a dependable, secure, observable, and testable read-only NAEOS Intelligence service suitable for a controlled Early Access cohort. The roadmap sequences specification alignment, lifecycle/API reliability, identity and access decisions, runtime foundations, policy/evidence, and release readiness.

This is a plan, not evidence of completion or production readiness. Existing implementation status is described in [Phase 00 Repository Assessment](PHASE-00-REPOSITORY-ASSESSMENT.md).

## Non-negotiable boundaries

- MVP remains read-only; no autonomous consequential actions or external side effects.
- Authorization must be explicit; model reasoning or a plan is never authorization.
- Security controls fail closed. Do not bypass CI, dependency risk governance, or security gates.
- IAM decisions and specification acceptance are maintainer/security decisions; unresolved items remain blockers.
- Any future execution capability requires a separately scoped design, authorization model, approval, security review, and acceptance.
- Production rollout is not implied by completing this roadmap; it requires operational approval and release evidence.

## Execution Snapshot (2026-11-04)

This snapshot tracks repository work, not milestone acceptance or Early Access approval.

| Milestone | Current status | Evidence / remaining gate |
|---|---|---|
| M0 — Documentation and Acceptance Alignment | Baseline documentation reconciliation completed; formal spec acceptance remains open | PR #23 aligned target lifecycle with MVP scope, completed normalized error properties/mapping, and clarified JWT/JWKS implementation versus deployment IAM decisions. The specifications remain Draft/Proposed until maintainer acceptance. |
| M1 — API Contract and Lifecycle Reliability | In progress | PR #24 merged normalized public error codes and problem-envelope correlation tests; PR #25 merged metadata bounds and response-schema alignment; PR #26 merged HTTP ownership isolation, client-scoped idempotency, and terminal cancellation conflict tests; PR #28 merged lightweight OpenAPI operation/schema contract-fragment tests (standard-library only). Remaining: deeper lifecycle/cancellation race validation and formal acceptance of the API/lifecycle contract. |
| M2 — IAM Decision and Security Hardening | Blocked on explicit IAM decisions / not started | IAM-01 through IAM-14 remain owner/security decisions. No production IAM acceptance is inferred. |
| M3 — Runtime and Worker Foundation | Not started | Requires accepted lifecycle/API contracts and approved scope. |
| M4 — Policy and Governance Enforcement | Not started | Requires accepted policy/IAM decisions; no side-effect execution enabled. |
| M5 — Early Access Readiness | Not started | Requires completion of prior milestones and explicit security/maintainer/operational sign-off. |

### Immediate next work

1. Continue M1 by adding automated OpenAPI contract conformance checks against HTTP handlers and response schemas.
2. Add focused cancellation/state-race coverage where the current MVP behavior supports it; keep worker-runtime cases deferred until M3.
3. Prepare the IAM decision review packet; do not resolve IAM worksheet items without the designated owner/security decision.
4. Update this snapshot after merged work and milestone exit reviews.

## Milestones

### M0 — Documentation and Acceptance Alignment (indicative: 1 week)

**Purpose:** Establish one consistent normative baseline before extending implementation.

Work:
- Reconcile architecture, request lifecycle, API contract, data model, and OpenAPI vocabulary.
- Resolve lifecycle-state differences and document legal transitions, terminal states, cancellation behavior, and invariants.
- Complete the normalized error handling specification, including stable codes, HTTP mapping, safe messages, and correlation.
- Review Phase 00 assessment and identify acceptance blockers explicitly.
- Mark documents as Draft, Proposed, or Accepted; do not treat proposed decisions as normative.

Exit criteria:
- A reviewed lifecycle/API/error contract with traceable requirements.
- Conflicts and unresolved decisions are explicitly recorded.
- No implementation work relies on an unaccepted spec.

### M1 — API Contract and Lifecycle Reliability (indicative: 1 week)

Work:
- Add contract tests that verify OpenAPI against HTTP behavior.
- Test request validation, ownership/tenant isolation, idempotency, and lifecycle transition invariants.
- Cover cancellation races and invalid/duplicate transitions.
- Normalize API error responses and request correlation.
- Preserve existing API compatibility or document an approved breaking change.

Exit criteria:
- Contract and integration tests pass in CI.
- Negative tests demonstrate rejection of unauthorized or invalid operations.
- API behavior is aligned to the accepted contract.

### M2 — IAM Decision and Security Hardening (indicative: 1–2 weeks)

Work:
- Resolve IAM-01 through IAM-14 decision worksheet items with evidence and explicit decision owners.
- Confirm issuer, audience, tenant/client/actor claims, JWKS trust, key rotation, token expiry, and revocation expectations.
- Harden verifier behavior and configuration; missing or invalid trust configuration fails closed.
- Add positive and negative authentication/authorization tests.
- Review secret handling, logging redaction, and cross-tenant isolation.

Exit criteria:
- Required IAM decisions are Accepted, not merely documented as proposals.
- Security tests cover invalid issuer/audience, malformed/expired tokens, unknown keys, and access boundaries.
- No unresolved IAM blocker is silently waived.

### M3 — Runtime and Worker Foundation (indicative: 1–2 weeks)

Work:
- Define durable work processing and request/attempt lifecycle boundaries.
- Specify bounded concurrency, timeout, retry/backoff, cancellation propagation, and recovery behavior.
- Implement only the minimum runtime needed for the accepted read-only scope.
- Add tests for worker interruption, retry exhaustion, cancellation, and persistence consistency.

Exit criteria:
- Runtime behavior is deterministic and covered by integration tests.
- Retry/cancellation policies are bounded and observable.
- Worker capabilities cannot invoke consequential external side effects.

### M4 — Policy and Governance Enforcement (indicative: 1 week)

Work:
- Align policy engine inputs/outputs to the accepted policy specification.
- Enforce deny-by-default behavior and explicit scoped permissions.
- Ensure audit/evidence records capture decision context, actor/request linkage, and outcome without leaking secrets.
- Add tests for denied access, missing policy, escalation attempts, and attempted bypasses.

Exit criteria:
- Policy decisions are enforced at the service boundary, not merely surfaced as model output.
- Audit evidence is append-only where specified and traceable.
- All policy/security negative tests pass.

### M5 — Early Access Readiness (indicative: 1 week)

Work:
- Run staging end-to-end tests for supported read-only user journeys.
- Validate database migration, backup/restore assumptions, and rollback runbook.
- Verify health/readiness checks, structured logs, request correlation, and operational alerts.
- Complete security review, release checklist, onboarding docs, and known-limitations register.
- Review Early Access cohort scope, data handling, support ownership, and incident response.

Exit criteria:
- CI, integration, and E2E evidence is recorded against release criteria.
- Staging validation and operational runbooks are reviewed.
- Security and maintainer sign-off is explicit.
- Release decision is recorded; absent sign-off, do not deploy to production.

## Dependency order

`M0 → M1 → M2 → M3 → M4 → M5`

Milestones may overlap only where their inputs are accepted and no dependency or security gate is bypassed. IAM and contract decisions are blockers for dependent implementation.

## Definition of Done for each milestone

- Scope and acceptance criteria are documented.
- Relevant unit, integration, contract, and/or E2E tests are added and pass.
- `go vet`, `go test`, and `go build` pass through repository CI.
- Security and dependency governance gates remain intact.
- Documentation reflects actual implementation and decision status.
- Review evidence and validation results are recorded in the PR.
- Changes merge only after required checks are green and repository protections are satisfied.

## Immediate execution queue

1. Complete M0 contract alignment: lifecycle states/transitions and normalized error specification.
2. Turn accepted M0 outcomes into M1 contract and lifecycle tests.
3. Close IAM worksheet blockers before enabling stronger deployment claims or extending access.
4. Proceed milestone-by-milestone; re-assess scope and readiness after each exit review.

## Out of scope for this Early Access roadmap

- Autonomous production deployment or unrestricted repository/data modification.
- General-purpose agent execution with arbitrary tools.
- Approval workflows or privileged operations that lack accepted governance and IAM design.
- Claims of production readiness based solely on a green build.
