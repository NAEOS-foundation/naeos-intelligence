# Architecture Decision Records

This directory contains Architecture Decision Records (ADRs) for NAEOS Intelligence.

ADRs record architectural decisions and their rationale. An ADR MUST remain consistent with higher-level requirements unless an explicit amendment is approved.

## Accepted decisions

| ADR | Decision |
|---|---|
| [ADR-0002](20261106-intelligence-pipeline.md) | Read-only deterministic intelligence pipeline |
| [ADR-0003](20261110-single-tenant-mvp-baseline.md) | Single logical tenant per MVP deployment; no shared tenancy or IdP selection |

## ADR naming convention

`YYYYMMDD-title.md`

Example: `20261106-auth-jwks-verifier.md`

## ADR template

```markdown
# ADR-XXXX: Title

**Status:** Proposed | Accepted | Superseded | Rejected  
**Date:** YYYY-MM-DD  
**Author:** name  

## Context

Describe the issue or decision point.

## Decision

Describe the chosen approach.

## Rationale

Explain why this approach was chosen.

## Consequences

Positive and negative consequences.

## References

Links to related specifications, issues, or documents.
```
