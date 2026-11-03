# Phase 1 — Repository Bootstrap and Engineering Baseline

Follow `.github/copilot-instructions.md`. Proceed only after Phase 0 is reviewed and explicitly authorized.

## Objective
Create a minimal, reproducible Go engineering foundation without implementing business capabilities.

## Before implementation
Inspect repository and phase-0 report. Confirm Go version/module path, tooling choices, and existing workflows. If required values are undecided, present options and wait instead of hardcoding.

## Tasks
- Establish minimal idiomatic Go layout (entrypoint and internal boundaries only where justified).
- Configure module/toolchain consistent with repository decision and current CI environment.
- Add reproducible developer commands (Makefile or documented equivalent).
- Add formatting, lint, unit-test, and build checks.
- Add CI workflow to validate baseline.
- Add safe .gitignore/.dockerignore and non-secret example configuration only if appropriate.
- Update README with setup, commands, and baseline status.
- Add a minimal smoke test for the application entrypoint.

## Out of scope
No API business endpoints, database, model integration, agent runtime, privileged tools, or production deployment.

## Validation
Run gofmt, lint if configured, go test ./..., and go build ./... (or documented equivalents). Report exact commands and results.

## Acceptance
Clean reproducible build; tests pass; CI checks defined; no secrets; only phase-scoped changes; setup documented.
