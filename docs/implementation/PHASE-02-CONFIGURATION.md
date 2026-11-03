# Phase 2 — Configuration and Dependency Wiring

**Status:** Implemented in the Phase 2 pull request  
**Decision basis:** Approved configuration decision proposal (PR #4)

## Scope

The initial configuration layer is deliberately limited to non-sensitive application settings:

- `app.environment`: `development`, `test`, `staging`, or `production`; default `development`.
- `app.log_level`: `debug`, `info`, `warn`, or `error`; default `info`.

Environment variables are the sole external source in this phase and override safe code defaults:

| Setting | Environment variable | Default |
|---|---|---|
| Application environment | `NAEOS_INTELLIGENCE_APP_ENVIRONMENT` | `development` |
| Log level | `NAEOS_INTELLIGENCE_APP_LOG_LEVEL` | `info` |

Unknown variables using the `NAEOS_INTELLIGENCE_` prefix are rejected. Configuration is validated at startup and treated as startup-only. Errors identify the setting and constraints without reflecting supplied values.

## Explicitly deferred

Configuration files and their format, secret provider/resolution, runtime reload, feature flags, configuration fingerprinting/snapshot persistence, HTTP/database/model/tool/policy settings remain out of scope. No secret values are accepted by the initial schema.

## Validation

The implementation adds unit tests for defaults, environment overrides, supported profiles, invalid settings, unknown prefixed settings, and error redaction. CI's formatting, vet, test, and build jobs are the acceptance checks.
