# NAEOS Intelligence — Copilot Staged Implementation Roadmap

**Status:** Draft / implementation guidance  
**Version:** 0.1.0  
**Updated:** 2026-11-04

## Purpose
This roadmap translates the repository's design documents into bounded GitHub Copilot Agent tasks. Execute one prompt at a time from `.github/prompts/`. Do not skip phases or treat this roadmap as an approval of unresolved architectural decisions.

## Execution protocol
1. Start Copilot Agent in the repository.
2. Read `.github/copilot-instructions.md` and the current phase prompt.
3. Ask Copilot to inspect repository state and relevant specifications first.
4. Review plan and risks before permitting code changes.
5. Run actual test/check commands and inspect results.
6. Review diff, security posture, and acceptance criteria.
7. Advance only when the phase is accepted by the maintainer.

## Phases and prompt files

| Phase | Goal | Prompt |
|---|---|---|
| 0 | Repository assessment and traceability | `phase-00-assessment.prompt.md` |
| 1 | Bootstrap engineering baseline | `phase-01-engineering-baseline.prompt.md` |
| 2 | Configuration and dependency wiring | `phase-02-configuration.prompt.md` |
| 3 | HTTP API and request lifecycle | `phase-03-api-lifecycle.prompt.md` |
| 4 | PostgreSQL persistence | `phase-04-persistence.prompt.md` |
| 5 | Identity and access control | `phase-05-identity-access.prompt.md` |
| 6 | Policy Engine | `phase-06-policy-engine.prompt.md` |
| 7 | Governed Agent Runtime | `phase-07-agent-runtime.prompt.md` |
| 8 | Tool Registry | `phase-08-tool-registry.prompt.md` |
| 9 | Model provider abstraction | `phase-09-model-provider.prompt.md` |
| 10 | Human approval workflow | `phase-10-approval-workflow.prompt.md` |
| 11 | Observability and audit | `phase-11-observability.prompt.md` |
| 12 | Security hardening | `phase-12-security-hardening.prompt.md` |
| 13 | Integration, E2E, early-access readiness | `phase-13-e2e-readiness.prompt.md` |

## Phase gating
A phase is ready for maintainer review only when:
- Scope is limited to that phase.
- Applicable acceptance criteria are fulfilled or explicitly listed as blocked.
- Tests and checks have been run; output is truthfully reported.
- No security gate is bypassed.
- Relevant documentation and contracts are updated.
- Diff is free of unrelated changes.

Do not auto-merge PRs or deploy production as part of these prompts.

## Repository-specific notes
- Technology direction in README is proposed; verify current Go toolchain and decisions before choosing dependencies.
- Authentication provider, config precedence, and certain infrastructure decisions remain open in draft specifications. Flag these for maintainers rather than hardcoding assumptions.
- Several tracked document paths have unusual suffixes (for example, trailing punctuation or shortened extensions). Do not rename them as a side effect; propose a deliberate documentation hygiene change separately.
