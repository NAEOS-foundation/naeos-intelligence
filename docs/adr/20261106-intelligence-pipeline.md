# ADR-0002: Read-Only Intelligence Pipeline (Classifier, Context Builder, Planner, Orchestrator)

**Status:** Accepted  
**Date:** 2026-11-06  
**Author:** NAEOS Intelligence Engineering

## Context

Phase 2 of the roadmap requires implementing the intelligence core: intent classification, context building, planning, and orchestration. The MVP remains read-only; governed execution is a future capability requiring separate maintainer and security acceptance. All components must be deterministic, side-effect free, and fail closed.

## Decision

Introduce four new packages under `internal/intelligence/`:

- `classifier` — deterministic, rule-based intent classification (no model invocation).
- `context` — deterministic context source validation and fingerprinting.
- `planner` — deterministic, prioritized task plan generation with dependencies; all tasks carry `status: proposed`.
- `orchestrator` — wires classifier → context → planner → runtime gate into a single read-only pipeline.

Each package exposes a small interface (`Classifier`, `Builder`, `Planner`) plus a `Rule*` implementation. The orchestrator accepts injected dependencies and returns a `PipelineResult` without invoking any tool or model.

## Rationale

- Rule-based implementations avoid model provider coupling and non-determinism in the MVP.
- Interface-based design allows future model-backed or plugin implementations without changing callers.
- `status: proposed` on every task enforces that plans are proposals, not authorization to execute.
- Fail-closed behavior: context cancellation, component errors, and invalid inputs all return errors rather than partial results.
- No external network or filesystem access; inputs are bounded and validated.

## Consequences

Positive:
- Deterministic, testable, auditable behavior.
- No new runtime dependencies; consistent with existing Go stdlib + pgx + goose stack.
- Clear boundary between intelligence (proposals) and governance/runtime (authorization/execution).

Negative:
- Intent classification accuracy is limited to keyword heuristics; model-backed classification is deferred.
- Planning does not consider historical execution data or resource constraints.

## References

- README.md — Phase 2 scope (Intent Classification, Context Builder, Planning Engine)
- internal/intelligence/classifier/classifier.go
- internal/intelligence/context/context.go
- internal/intelligence/planner/planner.go
- internal/intelligence/orchestrator/orchestrator.go
