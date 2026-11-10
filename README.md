# NAEOS Intelligence

**An independent intelligence layer for the NAEOS ecosystem.**

> Status: Implementation in Progress · MVP Foundation  
> Version: 0.1.0

NAEOS Intelligence is an independent, modular intelligence service designed to provide contextual reasoning, engineering analysis, workflow planning, and governed agent assistance across the NAEOS ecosystem.

The project aims to bridge engineering intent and structured execution plans through explainable, policy-aware, and evidence-driven AI capabilities.

## Overview

NAEOS Intelligence is designed to help users and engineering systems:

- Understand engineering context across repositories and services.
- Analyze architecture, technical debt, and engineering risks.
- Interpret issues and requirements.
- Generate structured and prioritized execution plans.
- Assist with pull request reviews.
- Integrate with NAEOS through explicit, versioned contracts.
- Produce traceable findings and evidence.
- Provide read-only engineering intelligence in the MVP; controlled tool execution is a future capability requiring separate acceptance.

NAEOS Intelligence is not intended to replace NAEOS Core, Runtime, or existing governance mechanisms.

## Vision

Build a vendor-neutral intelligence layer that enables AI agents to reason about engineering context and assist with complex workflows while preserving policy authority, accountability, and verification.

## Design Principles

### 1. Independent Architecture

NAEOS Intelligence is developed in a separate repository to maintain clear architectural boundaries and independent evolution.

### 2. Specification-First Development

Architecture, contracts, security requirements, and acceptance criteria should be defined before implementation.

### 3. Policy-Aware Intelligence

Agent reasoning must not be treated as authorization. Consequential actions must be subject to explicit governance.

### 4. Evidence-Driven Results

Findings and recommendations should be traceable to their supporting context. Model-generated claims alone are not considered verification evidence.

### 5. Provider Neutrality

The system should support multiple model providers through an abstraction layer without coupling its core architecture to a single vendor.

### 6. Modular Design

The initial implementation follows a modular monolith architecture. Service decomposition should be introduced only when justified by operational or scaling requirements.

### 7. Secure by Default

Tool invocation, data access, credential handling, and external side effects must follow least-privilege principles.

## Initial Scope

The first development phase focuses on Engineering Intelligence.

| Capability | Description | Phase |
|---|---|---|
| Repository Intelligence | Analyze repository structure and architecture | MVP |
| Issue Intelligence | Interpret engineering issues and requirements | MVP |
| PR Review Intelligence | Analyze changes and identify risks | MVP |
| Engineering Planner | Generate structured task plans and dependencies | MVP |
| Evidence Generation | Produce traceable findings and references | MVP |
| Knowledge Retrieval | Retrieve relevant engineering context | Future |
| Controlled Execution | Execute approved engineering actions only after separate authorization, approval, and security acceptance | Future |
| Multi-Agent Coordination | Coordinate specialized agents | Future |

## Architecture

The initial logical architecture consists of:

- Agent API
- Intent Classification
- Context Builder
- Planning Engine
- Agent Orchestrator
- Model Provider Abstraction
- Tool Registry
- Governance Integration
- Approval Manager
- Evidence Manager
- Audit and Observability

The architecture is designed to integrate with NAEOS Core and other ecosystem components through versioned API contracts.

```text
                    NAEOS Ecosystem
                           |
                           v
                 NAEOS Intelligence API
                           |
                  Agent Orchestrator
                           |
         +-----------------+-----------------+
         |                 |                 |
         v                 v                 v
   Context Builder     Planning Engine   Tool Registry
         |                 |                 |
         v                 v                 v
   Knowledge Layer    Policy Boundary    Approved Tools
                           |
                           v
                   Evidence & Audit
```

This diagram represents the proposed architecture, not an implemented system.

## Technology Direction

The initial technology direction is:

| Component | Proposed Technology |
|---|---|
| Primary Language | Go |
| API | REST + OpenAPI |
| Persistence | PostgreSQL |
| Model Integration | Provider abstraction |
| Observability | OpenTelemetry |
| Deployment | Docker |
| CI/CD | GitHub Actions |
| Vector Retrieval | Optional PostgreSQL + pgvector |

Technology choices remain subject to architecture review.

## Repository Structure

The proposed repository structure is:

```text
naeos-intelligence/
├── cmd/
├── internal/
├── pkg/
├── api/
├── configs/
├── deployments/
├── docs/
│   ├── architecture/
│   ├── specifications/
│   ├── security/
│   ├── operations/
│   ├── evaluation/
│   └── adr/
├── tests/
├── examples/
├── README.md
├── AGENTS.md
├── CONTRIBUTING.md
├── SECURITY.md
└── LICENSE
```

The repository structure will evolve alongside approved specifications and implementation milestones.

## Governance and Security

NAEOS Intelligence follows these foundational rules:

- Intent is not authorization.
- Reasoning is not policy evaluation.
- Planning is not execution.
- Execution is not verification.
- Tool access must be explicitly scoped.
- External side effects must be authorized.
- Model outputs must be treated as untrusted.
- Audit records must preserve action traceability.
- Unauthorized execution must fail closed.

The initial MVP will not permit autonomous production deployment or unrestricted production data modification.

## NAEOS service connections

The API enriches explicitly supplied engineering context from NAEOS Core and GitHub. These connectors are read-only and run only for matching source types (`naeos-core` and `repository`); other source types do not trigger outbound requests. CRM records and personal data are not connector inputs. The Core connector calls `GET /api/v1/repositories/{owner}/{name}` on the standalone Core API; Core limits requests to `NAEOS_GITHUB_ALLOWED_REPOSITORIES` and requires the `naeos:repository:read` service scope. For a local host setup, Core can listen on `8081` while Intelligence listens on `8080`; set the Core base URL to `http://127.0.0.1:8081/api`.

Configure Core with `NAEOS_INTELLIGENCE_CORE_API_BASE_URL` (default: `https://api.naeos.dev/api`) and provide a pre-issued RS256 service JWT in `NAEOS_CORE_SERVICE_TOKEN`. The JWT must be trusted by Core's configured JWKS verifier and include only `naeos:repository:read`. GitHub authentication is optional and can be supplied with `GITHUB_TOKEN` to raise the unauthenticated API rate limit. Keep credentials in a secrets manager or local environment, never in repository files. The Compose stack forwards these variables to the Intelligence container.

Core and GitHub lookup failures stop context building. Retrieved context is still untrusted input and does not grant permission to execute actions; Intelligence outputs remain proposals under the existing policy and runtime boundaries.

## Development Roadmap

### Phase 0 — Documentation and Architecture

- Establish project vision and scope.
- Define system boundaries.
- Develop reference architecture.
- Define technical specifications.
- Establish architecture decision records.
- Define security and trust requirements.

### Phase 1 — API, Persistence, and Governance Foundations

- Establish the bounded read-only API and lifecycle contract.
- Implement and validate persistence, ownership isolation, and API error semantics.
- Resolve required IAM/security decisions through maintainer and security review before provider-specific trust configuration.
- Implement audit/evidence records appropriate to the read-only MVP.
- Keep tool execution and approval workflows disabled unless separately accepted.

### Phase 2 — Intelligence Core

- Implement provider abstraction behind explicit interfaces.
- Implement intent classification.
- Implement context building with source authorization and provenance.
- Implement planning and orchestration within the read-only MVP boundary.
- Validate outputs and traceability; model-generated plans remain proposals, not authorization.

### Phase 3 — NAEOS Integration

- Integrate GitHub.
- Integrate NAEOS APIs.
- Establish CRM integration contracts.
- Validate integration through automated tests.

### Phase 4 — MVP Validation

- Unit and integration testing.
- End-to-end testing.
- Security validation.
- Agent evaluation.
- Staging deployment.
- Early Access readiness assessment.

## Documentation

Technical documentation will be maintained within the repository.

Documentation domains:

- Architecture
- Technical Specifications
- API Contracts
- Security
- Operations
- Agent Evaluation
- Architecture Decision Records

Normative documents will define system requirements and architectural constraints.

## Project Status

NAEOS Intelligence has moved beyond the initial documentation phase: a Go service foundation, read-only MVP API surface, request validation, and JWT/JWKS authentication scaffolding are present in the repository. Documentation and acceptance decisions remain in progress.

Provider-specific IAM choices, production hardening, end-to-end acceptance, and early-access readiness are not implied by the existing implementation. The MVP remains read-only; governed execution is a future capability requiring separate maintainer and security acceptance.

## Contributing

Contributions should follow the project's specifications, security requirements, and engineering governance.

Before contributing:

1. Review the project documentation.
2. Understand the architectural boundaries.
3. Follow the contribution guidelines.
4. Include appropriate tests.
5. Provide evidence for relevant changes.

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution requirements.

## Security

Security issues should be reported according to the process defined in [SECURITY.md](SECURITY.md).

Do not disclose vulnerabilities through public issues before they have been appropriately assessed.

## License

NAEOS Intelligence is licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for the full license text.

## Acknowledgments

NAEOS Intelligence is part of the broader NAEOS ecosystem and is intended to complement its existing runtime, control-plane, and engineering governance capabilities.
