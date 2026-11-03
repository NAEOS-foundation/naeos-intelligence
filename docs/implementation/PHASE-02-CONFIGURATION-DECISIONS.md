# Phase 2 — Configuration Decision Proposal

**Status:** Proposed — maintainer approval required  
**Date:** 2026-11-04  
**Scope:** Resolve only the open decisions required to begin a safe configuration bootstrap.

This document is a recommendation, not an accepted architecture decision. No configuration implementation or secret-provider integration is authorized by this proposal.

## 1. Decisions to resolve

| Decision | Recommendation | Rationale / boundary |
|---|---|---|
| Initial configuration source | Environment variables only, plus code-defined defaults for non-sensitive fields | Avoid selecting a file format/parser or layering semantics prematurely |
| Canonical prefix | `NAEOS_INTELLIGENCE_` | Consistent with the Draft Configuration Specification |
| Initial precedence | Explicit environment variable overrides a non-sensitive default; otherwise use the default | Single source layer plus defaults is deterministic and easy to test; no runtime overrides or file precedence |
| Config file format | Defer YAML/JSON selection | No config file is needed for the initial application bootstrap |
| Environment profiles | Allow only `development`, `test`, `staging`, `production`; default to `development` only for local/bootstrap context | Production deployments must explicitly provide their environment |
| Initial fields | `app.environment`, `app.log_level` only; expand when components exist | Avoid defining unused settings for database, models, tools, policy, or HTTP server |
| Secret provider | Defer provider integration; no secrets consumed in Phase 2 bootstrap | Provider choice remains open; do not treat plaintext secret environment values as a secret-management solution |
| Secret references | Define a typed opaque reference shape only if a component requires one; do not resolve it yet | Avoid embedding vendor-specific semantics |
| Reload behavior | Startup-only; no live reload | Prevent partial snapshots and uncontrolled changes |
| Unknown settings | Reject unknown recognized-prefix keys where practical; never dump the full process environment | Reduce silent configuration errors without leaking values |
| Validation errors | Report key and safe reason, never raw values | Prevent secret or sensitive-value exposure |
| Configuration fingerprint | Defer until persisted execution records exist; if added later, exclude secret values | Avoid premature snapshot semantics |

## 2. Suggested initial configuration contract

Initial proposed fields:

- `app.environment`: enumerated environment profile.
- `app.log_level`: one of `debug`, `info`, `warn`, `error`; production default should be `info`.

Proposed environment mapping:

- `NAEOS_INTELLIGENCE_APP_ENVIRONMENT`
- `NAEOS_INTELLIGENCE_APP_LOG_LEVEL`

No database, model-provider, tool, policy, audit persistence, server-listener, credential, or secret fields should be implemented until their owning phase and decisions are reviewed.

## 3. Security invariants

- Configuration does not grant authorization or permit policy enforcement to be disabled.
- No secrets are logged, echoed in errors, or added to agent-visible context.
- No production credential fallback to development/staging values.
- Missing or invalid explicitly-required production configuration fails closed.
- Configuration parsing and validation happen before any future listener or consequential component starts.
- Configuration remains immutable after startup in this initial implementation.

## 4. Phase 2 implementation acceptance proposal

After decisions above are accepted:

1. Implement a small typed configuration package using the Go standard library unless a reviewed dependency is justified.
2. Add deterministic environment parsing and validation.
3. Add unit tests for defaults, supported environment profiles, invalid values, explicit environment overrides, unknown recognized-prefix keys (if supported), and safe error redaction.
4. Wire configuration into the minimal application entrypoint without introducing network services or domain components.
5. Document the exact environment variables and behavior.
6. Run gofmt, go vet, go test, and go build in CI.
7. Keep secrets-provider integration, file parsing, dynamic reload, and unrelated settings out of scope.

## 5. Maintainer approval needed

Please review/accept or revise the recommendations for:

- Environment-only initial source and default/override behavior.
- Minimal initial field set and environment profiles.
- Deferral of config file format and secret provider.
- Startup-only immutable configuration.

Until accepted, this proposal should remain Draft and Phase 2 code should not encode these open decisions.
