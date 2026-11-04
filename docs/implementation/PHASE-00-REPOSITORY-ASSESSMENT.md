# Phase 0 — Repository Assessment and Engineering Baseline

**Repository:** `NAEOS-foundation/naeos-intelligence`  
**Assessment date:** 2026-11-04  
**Assessment baseline:** `main` at `a44e52e3c6956ead23568c5c28f34379337abf67`  
**Status:** Assessment complete — maintainer review required  
**Scope:** Read-only repository inspection; no application code changed.

## 1. Executive summary

The repository currently contains a substantial design/specification baseline and the staged Copilot implementation guidance. The inspected repository tree contains no Go application source, `go.mod`, automated test suite, or GitHub Actions workflow. Therefore the project remains documentation-first and has not yet established a reproducible software build baseline.

The recommended next engineering milestone is a minimal Phase 1 engineering baseline: settle the Go toolchain/module path and the smallest approved tooling choices, bootstrap the Go service skeleton, establish developer commands, and add CI for formatting, tests, and build. Do not implement agent, model, persistence, or privileged execution behavior during this milestone.

## 2. Repository inventory (observed)

### Present
- Root project guidance: `README.md`, `DOCUMENTATION-AUTHORITY.md`.
- Initial documentation blueprint: `docs/NAEOS-INTELLIGENCE-INITIAL-DOCUMENTATION-BLUEPRINT.md`.
- Vision/scope documents under `docs/vision/`.
- Architecture documents under `docs/architecture/`.
- Security requirements and threat model under `docs/security/`.
- Domain specifications under `docs/specifications/`.
- OpenAPI contract: `api/openapi.yaml`.
- Copilot governance instructions: `.github/copilot-instructions.md`.
- Phase prompt files: `.github/prompts/phase-00` through `phase-13`.
- Staged implementation roadmap: `docs/implementation/COPILOT-IMPLEMENTATION-ROADMAP.md`.

### Not present in inspected tree
- Go module file (`go.mod`) and Go source tree (`cmd/`, `internal/`).
- Unit/integration/contract test files or test harness.
- GitHub Actions workflow files under `.github/workflows/`.
- Build/lint task runner configuration such as a Makefile.
- Container build/deployment baseline.
- Dedicated identity-and-access-control specification.
- Dedicated ADR directory and accepted ADRs.
- CONTRIBUTING.md, SECURITY.md, and LICENSE (README references contribution/security docs; license is noted as not yet defined).

These observations refer to the tree at the assessment baseline and should be refreshed before implementation if main changes.

## 3. Documentation maturity and authority

The inspected README describes the project as **Early Design · Draft**. `DOCUMENTATION-AUTHORITY.md` says documents must not be considered authoritative merely because they exist; normative documentation is initially proposed until formally reviewed and accepted. The architecture and security documents inspected also declare Draft/Proposed statuses.

Implementation should consequently treat these documents as design input, not as a fully ratified architecture. Copilot must surface conflicts and open decisions rather than silently encode them.

## 4. Gaps and risks

| ID | Finding | Risk / implication | Suggested action |
|---|---|---|---|
| P0-01 | No Go module/source baseline | Cannot build or test service yet | Phase 1 bootstrap |
| P0-02 | No CI workflow observed | Changes lack automated baseline validation | Add GitHub Actions checks in Phase 1 |
| P0-03 | Go version and module path not concretely established in inspected root baseline | Reproducibility and toolchain mismatch risk | Maintainer approves toolchain and module path before bootstrap |
| P0-04 | Several technology choices are proposed (README/system architecture) | Copilot could introduce unnecessary or inconsistent dependencies | Keep initial dependency set minimal; record choices as ADRs if they materially constrain architecture |
| P0-05 | Identity/access-control specification not present in tree | Authn/authz implementation requirements lack a dedicated detailed spec | Author/review spec before Phase 5; avoid insecure protected API exposure meanwhile |
| P0-06 | Documentation filenames contained unusual suffixes (resolved in docs-only cleanup PR) | Links/tools and prompt references could fail or be confusing | Renamed to conventional `.md` paths; update internal references and verify links |
| P0-07 | README's high-level phase sequence (Intelligence Core first, Governance/Evidence after) differs from the staged prompt sequence that prioritizes persistence/access/policy boundaries before agent execution | Roadmap sequencing inconsistency | Maintainer confirms safety-first sequencing; update roadmap/README with a documented rationale |
| P0-08 | System Architecture lifecycle labels and Request Lifecycle specification should be cross-checked | Contract/state machine drift possible | Reconcile in documentation review before lifecycle implementation |
| P0-09 | No accepted ADRs observed in tree | Major technical choices may be informal | Create ADRs only for decisions actually reviewed and accepted |

## 5. Proposed Phase 1 scope

### Include
1. Agree and record Go toolchain version and module path.
2. Create a minimal idiomatic Go entrypoint and structure with no premature domain modules.
3. Add reproducible developer commands (Makefile or equivalent).
4. Add baseline formatting, build, and test commands.
5. Add GitHub Actions workflow for the approved Go version to run formatting checks, `go test ./...`, and `go build ./...`.
6. Add a minimal smoke/unit test for the bootstrap.
7. Update README with exact prerequisites and local commands.
8. Add safe ignore files if needed.

### Exclude
- API handlers/business endpoints.
- PostgreSQL, migrations, repositories.
- Authentication provider integration.
- Policy engine, agent planner/runtime, tools, approval execution.
- Model provider integrations.
- Production Docker deployment, secrets infrastructure, or NAEOS integration.
- Renaming documentation files or broad documentation restructuring.

## 6. Decisions requiring maintainer review

Before Phase 1 implementation, confirm:
- Go version/toolchain baseline.
- Canonical Go module path.
- Whether a Makefile is desired or commands should be documented another way.
- Linting tool choice (or keep initial CI to gofmt/build/test and add lint separately).
- Safety-first phase order: establish auth/policy and governed execution gates before enabling agent tool execution.
- Whether README roadmap is updated to align with staged prompt sequencing.

If a decision is not yet approved, Copilot should pause and present a recommendation with trade-offs rather than assume.

## 7. Phase 1 acceptance criteria

- [ ] Approved Go toolchain and module path recorded.
- [ ] Minimal application builds reproducibly.
- [ ] Developer setup and commands documented.
- [ ] CI runs formatting validation, tests, and build on pull requests.
- [ ] Bootstrap test exists and passes.
- [ ] No business capabilities or privileged execution are introduced.
- [ ] No secrets committed.
- [ ] Diff contains only Phase 1 scope.
- [ ] Exact commands and outcomes reported.

## 8. Traceability snapshot

| Design source | Assessment disposition | Implementation relevance |
|---|---|---|
| README / Project Vision / Scope | Draft / proposed context | Bound project purpose and initial exclusions |
| System Architecture | Draft Proposed | Guidance; reconcile decisions before binding implementation |
| Security Requirements / Threat Model | Draft Proposed | Security controls should be preserved; ratification/review remains needed |
| API Contract / OpenAPI | Existing contract draft | Not implemented; avoid API work in Phase 1 |
| Configuration, lifecycle, error handling, data model specs | Draft specifications | Future phases; resolve naming/contract consistency before implementation |
| Copilot instructions and staged prompts | Implementation guidance | Execution guardrails; not substitute for accepted requirements |

## 9. Recommendation

**Proceed to Phase 1 only after maintainer confirms the open toolchain/module decisions.** Keep the first PR small and infrastructure-only. After its CI baseline is green and reviewed, implement configuration wiring and then API/lifecycle in separate bounded changes. Do not enable model-driven tool execution before authentication, policy enforcement, and approval boundaries are implemented and validated.

---

This report records observations from the stated GitHub tree baseline. It does not claim local commands, tests, or CI were executed.
