# NAEOS Intelligence — Threat Model

**Document ID:** NAEOS-INT-SEC-001  
**Version:** 0.1.0  
**Status:** Draft Proposed  
**Audience:** Maintainers, architects, security reviewers, and contributors

---

## 1. Purpose

This document identifies security threats, trust boundaries, attack surfaces, and mitigation requirements for NAEOS Intelligence.

The threat model establishes a security baseline for the MVP and MUST be reviewed as the architecture or capabilities evolve.

This document is governed by `DOCUMENTATION-AUTHORITY.md` and must remain consistent with:

- `PROJECT-VISION.md`
- `PROJECT-SCOPE.md`
- `SYSTEM-ARCHITECTURE.md`
- `COMPONENT-BOUNDARIES.md`
- `DATA-FLOW.md`

## 2. Security Objectives

NAEOS Intelligence MUST preserve the following objectives:

- Prevent unauthorized actions.
- Prevent model-generated instructions from bypassing policy.
- Protect credentials and sensitive data.
- Preserve integrity of authorization decisions.
- Preserve integrity and traceability of audit evidence.
- Prevent privilege escalation.
- Restrict tools to explicit capabilities.
- Maintain secure behavior under failure.
- Ensure that execution outcomes are evidence-backed.

## 3. System Overview

NAEOS Intelligence accepts requests from authorized clients, retrieves contextual information, generates plans, evaluates action proposals, and may execute approved tool actions.

The system includes:

- API Layer.
- Request Orchestrator.
- Context Retrieval.
- Planner.
- Model Provider Adapter.
- Policy Engine.
- Approval Workflow.
- Governed Tool Runtime.
- Verification Engine.
- Evidence and Audit Store.
- External data sources and tool adapters.

The MVP is designed to provide read-only engineering intelligence first.

## 4. Security Assumptions

The following assumptions apply to the initial threat model:

1. Model outputs are untrusted.
2. User inputs may be malicious.
3. Repository contents may contain adversarial instructions.
4. External data sources may be compromised or manipulated.
5. Tool outputs may be incorrect or malicious.
6. Credentials may be exposed through application defects.
7. Network communication may be interrupted or intercepted if improperly configured.
8. Authorized users may make mistakes or abuse privileges.
9. Model providers are external dependencies.
10. Authorization must be enforced by application controls, not by prompts.

These assumptions MUST be revisited before production deployment.

## 5. Trust Boundaries

### TB-001 — Client to API

The API receives untrusted requests.

Controls:

- Authentication.
- Authorization.
- Schema validation.
- Rate limiting.
- Request size limits.
- Transport security.

### TB-002 — External Context to Intelligence

Retrieved repository content, issues, documentation, and other external data are untrusted.

Controls:

- Source access validation.
- Provenance tracking.
- Context filtering.
- Prompt injection defenses.
- Separation of data from system instructions.

### TB-003 — Model to Policy

Model-generated plans and proposals are untrusted.

Controls:

- Structured output validation.
- Action schema validation.
- Policy evaluation.
- Explicit capability checks.

### TB-004 — Policy to Runtime

This is the critical authorization boundary.

Controls:

- Signed or integrity-protected authorization artifacts where appropriate.
- Proposal digest binding.
- Expiration enforcement.
- Final authorization revalidation.
- Runtime permission checks.

### TB-005 — Runtime to External Tools

Tool execution may create external side effects.

Controls:

- Tool allowlists.
- Least privilege.
- Input validation.
- Execution limits.
- Isolated execution environments where applicable.
- Explicit approval requirements.

### TB-006 — Application to Evidence Store

Security-relevant records require integrity and durability.

Controls:

- Access restrictions.
- Durable persistence.
- Integrity checks.
- Audit of administrative access.
- Defined retention policy.

## 6. Threat Classification

The threat model uses STRIDE:

| Category | Threat |
|---|---|
| Spoofing | Attacker impersonates a user, service, or approver |
| Tampering | Attacker modifies proposals, policies, or evidence |
| Repudiation | Actions cannot be attributed or proven |
| Information Disclosure | Sensitive information is exposed |
| Denial of Service | Service or dependencies become unavailable |
| Elevation of Privilege | Agent or user obtains unauthorized capabilities |

## 7. Threat Register

### TM-001 — Prompt Injection Through Repository Content

**Category:** Tampering / Elevation of Privilege  
**Severity:** Critical

Threat:

An attacker places instructions in repository files, issue comments, pull request descriptions, or documentation that attempt to override agent behavior.

Potential impact:

- Unauthorized tool execution.
- Data exfiltration.
- Policy bypass attempts.
- Manipulation of engineering recommendations.

Mitigations:

- Treat retrieved content as untrusted data.
- Separate system instructions from retrieved content.
- Do not allow contextual content to grant permissions.
- Validate every proposed action through policy.
- Restrict tool capabilities.
- Test adversarial repository content.

Acceptance criteria:

A malicious repository instruction MUST NOT authorize or directly execute a privileged action.

### TM-002 — Unauthorized Tool Execution

**Category:** Elevation of Privilege  
**Severity:** Critical

Threat:

The Planner or model attempts to execute a tool without authorization.

Potential impact:

- Unauthorized repository changes.
- External side effects.
- Credential exposure.
- System compromise.

Mitigations:

- Tool registry and allowlist.
- Policy enforcement before execution.
- Runtime authorization revalidation.
- Least-privilege tool credentials.
- No unrestricted shell execution in the MVP.

Acceptance criteria:

A denied or indeterminate policy decision MUST prevent tool invocation.

### TM-003 — Policy Bypass

**Category:** Elevation of Privilege / Tampering  
**Severity:** Critical

Threat:

An attacker manipulates the policy decision process or bypasses the Policy Engine.

Potential impact:

- Unauthorized execution.
- Loss of governance guarantees.

Mitigations:

- Policy Engine separated from Planner.
- Runtime validates authorization artifacts.
- Authorization decisions bound to proposal digests.
- Fail-closed behavior.
- Policy changes require controlled review.

Acceptance criteria:

No executable action can bypass the runtime authorization boundary.

### TM-004 — Approval Replay or Misbinding

**Category:** Spoofing / Elevation of Privilege  
**Severity:** High

Threat:

An approval is reused for a different action, target, or parameter set.

Potential impact:

- Unauthorized execution under a previously valid approval.

Mitigations:

- Bind approval to immutable proposal digest.
- Validate approver identity.
- Enforce expiration.
- Reject modified proposals.
- Record approval lifecycle.

Acceptance criteria:

Approval for one proposal MUST NOT authorize a materially different proposal.

### TM-005 — Secret Leakage

**Category:** Information Disclosure  
**Severity:** Critical

Threat:

Credentials, API keys, tokens, or sensitive data are exposed through model context, logs, tool output, or persisted evidence.

Potential impact:

- Unauthorized access to external systems.
- Account compromise.
- Data exfiltration.

Mitigations:

- Secret management system.
- Credential redaction.
- Context minimization.
- No secrets in logs.
- Scoped credentials.
- Restricted access to evidence records.
- Rotation and revocation procedures.

Acceptance criteria:

Automated tests MUST verify that known secret patterns are redacted from designated logs and outputs.

### TM-006 — Malicious Tool Output

**Category:** Tampering / Information Disclosure  
**Severity:** High

Threat:

A tool returns crafted content containing malicious instructions or sensitive data.

Potential impact:

- Prompt injection during subsequent model calls.
- Incorrect recommendations.
- Sensitive data propagation.

Mitigations:

- Treat tool output as untrusted.
- Validate output schemas.
- Preserve source provenance.
- Avoid interpreting tool output as trusted instructions.
- Apply data minimization.

Acceptance criteria:

Tool output MUST NOT alter authorization rules or runtime permissions.

### TM-007 — Identity Spoofing

**Category:** Spoofing  
**Severity:** High

Threat:

An attacker impersonates a user, service, or approver.

Potential impact:

- Unauthorized requests.
- Invalid approval.
- Unauthorized data access.

Mitigations:

- Strong authentication.
- Service identity validation.
- Role and permission enforcement.
- Secure session handling.
- Approval identity verification.

Acceptance criteria:

Protected requests and approval actions MUST be attributable to a validated identity.

### TM-008 — Audit Trail Tampering

**Category:** Tampering / Repudiation  
**Severity:** High

Threat:

An attacker modifies, deletes, or falsifies audit records.

Potential impact:

- Loss of accountability.
- Inability to investigate incidents.
- False claims of successful execution.

Mitigations:

- Restricted write and administrative access.
- Durable audit persistence.
- Integrity protection.
- Administrative action logging.
- Tamper-evident mechanisms where appropriate.

Acceptance criteria:

Security-relevant events MUST be protected against unauthorized modification and deletion.

### TM-009 — Data Exfiltration Through Context

**Category:** Information Disclosure  
**Severity:** Critical

Threat:

The agent retrieves or sends data beyond the caller's authorized scope.

Potential impact:

- Confidential repository or organizational data disclosure.
- Cross-tenant data leakage.

Mitigations:

- Source-level access controls.
- Tenant and principal context propagation.
- Context minimization.
- Provider data handling review.
- Explicit data retention configuration.
- Authorization checks on retrieval.

Acceptance criteria:

A requester MUST NOT retrieve context outside their authorized access scope.

### TM-010 — Denial of Service

**Category:** Denial of Service  
**Severity:** High

Threat:

An attacker causes excessive model calls, expensive context retrieval, or resource exhaustion.

Potential impact:

- Service unavailability.
- Increased provider costs.
- Database overload.

Mitigations:

- Rate limits.
- Request quotas.
- Input size limits.
- Timeouts.
- Bounded retries.
- Concurrency limits.
- Resource budgets.

Acceptance criteria:

The service MUST enforce configured request and execution limits.

### TM-011 — Model Provider Compromise or Unavailability

**Category:** Information Disclosure / Denial of Service  
**Severity:** High

Threat:

A model provider is compromised, unavailable, or behaves unexpectedly.

Potential impact:

- Sensitive context exposure.
- Incorrect output.
- Service disruption.

Mitigations:

- Provider abstraction.
- Provider security review.
- Data minimization before invocation.
- Timeout and retry policies.
- Output validation.
- No provider-dependent authorization.

Acceptance criteria:

Provider failure MUST NOT result in implicit execution permission.

### TM-012 — Dependency or Supply Chain Compromise

**Category:** Tampering / Elevation of Privilege  
**Severity:** High

Threat:

A dependency, build artifact, or CI process is compromised.

Potential impact:

- Malicious application behavior.
- Credential theft.
- Unauthorized deployment.

Mitigations:

- Dependency pinning.
- Vulnerability scanning.
- Software composition analysis.
- Protected CI workflows.
- Artifact provenance.
- Review of dependency changes.

Acceptance criteria:

Security-sensitive dependency changes MUST be reviewed and tested.

### TM-013 — Authorization Time-of-Check to Time-of-Use

**Category:** Elevation of Privilege / Tampering  
**Severity:** Critical

Threat:

A resource or permission changes after policy evaluation but before execution.

Potential impact:

- Execution against an unauthorized target.
- Stale authorization.

Mitigations:

- Final authorization revalidation.
- Proposal digest binding.
- Resource version checks where supported.
- Short authorization validity windows.
- Runtime-side enforcement.

Acceptance criteria:

Runtime MUST reject stale or invalid authorization before a consequential action.

### TM-014 — Cross-Tenant Data Leakage

**Category:** Information Disclosure  
**Severity:** Critical

Threat:

Requests from one tenant access context, records, or execution results belonging to another tenant.

Potential impact:

- Confidential data disclosure.
- Tenant isolation failure.

Mitigations:

- Tenant identity in authorization context.
- Tenant-scoped persistence queries.
- Tenant-bound resource identifiers.
- Isolation tests.
- No reliance on model prompts for tenant separation.

Acceptance criteria:

Cross-tenant access MUST be denied by deterministic application controls.

### TM-015 — False Verification or Completion Claims

**Category:** Repudiation / Tampering  
**Severity:** High

Threat:

The system reports an action as completed or verified without sufficient evidence.

Potential impact:

- Incorrect operational decisions.
- False accountability.
- Hidden execution failures.

Mitigations:

- Explicit lifecycle states.
- Separate execution and verification records.
- Independent observations where possible.
- Evidence-backed result generation.

Acceptance criteria:

The system MUST distinguish attempted, executed, completed, and verified states.

## 8. Risk Evaluation

Risk severity is initially classified qualitatively.

| Severity | Definition |
|---|---|
| Critical | Could enable unauthorized consequential action or major sensitive data disclosure |
| High | Could compromise important security controls or materially disrupt service |
| Medium | Limited impact requiring additional conditions |
| Low | Minor impact with constrained consequences |

Threat severity MUST be reassessed using likelihood, exposure, and impact as deployment details become known.

## 9. Security Controls for MVP

The following controls are required before enabling governed tool execution:

- Authenticated API requests.
- Explicit principal and tenant context where applicable.
- Structured and validated action proposals.
- Independent policy evaluation.
- Runtime authorization revalidation.
- Tool allowlist.
- Least-privilege credentials.
- Approval binding and expiration.
- Durable audit records.
- Sensitive-data redaction.
- Execution limits.
- Security-focused integration tests.
- Documented incident response procedure.

The initial read-only Engineering Intelligence capability SHOULD be delivered before enabling consequential execution.

## 10. Security Testing

The security test suite SHOULD include:

### Prompt Injection Tests

- Malicious repository README instructions.
- Issue comments requesting secret disclosure.
- Pull request descriptions requesting policy bypass.
- Tool output containing adversarial instructions.

### Authorization Tests

- Denied action execution attempt.
- Indeterminate policy decision.
- Expired authorization.
- Modified proposal after approval.
- Unauthorized tool identifier.

### Data Protection Tests

- Secret redaction.
- Tenant isolation.
- Unauthorized context retrieval.
- Sensitive data in logs.

### Runtime Tests

- Tool timeout.
- Tool failure.
- Invalid tool output.
- Resource state changes before execution.

### Evidence Tests

- Audit record integrity.
- Missing evidence.
- Failed persistence.
- Incorrect verification status.

## 11. Incident Response

The project SHOULD define a documented response process for:

- Credential exposure.
- Unauthorized execution.
- Policy bypass.
- Sensitive data disclosure.
- Audit integrity compromise.
- Compromised dependency or build artifact.

The process SHOULD include:

1. Detection.
2. Containment.
3. Credential revocation.
4. Evidence preservation.
5. Impact assessment.
6. Remediation.
7. Recovery.
8. Post-incident review.

## 12. Open Security Decisions

The following decisions remain open:

- Authentication mechanism.
- Tenant isolation model.
- Policy engine technology.
- Authorization artifact integrity mechanism.
- Approval identity provider.
- Audit retention and tamper-evidence strategy.
- Secret management implementation.
- Model provider data handling requirements.
- Tool sandboxing architecture.
- Incident response ownership.

These decisions MUST be resolved before production-critical execution capabilities are enabled.

## 13. Document Governance

This document is governed by `DOCUMENTATION-AUTHORITY.md`.

Threats and mitigations MUST be reviewed whenever:

- New tools are introduced.
- Execution capabilities expand.
- New data sources are integrated.
- Authentication or authorization changes.
- Deployment architecture changes.
- A security incident reveals a new attack path.

Security-sensitive architecture changes SHOULD include an updated threat assessment and relevant tests.
