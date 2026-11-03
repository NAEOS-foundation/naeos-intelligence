# Tool Registry Specification

**Document ID:** NAEOS-INT-SPEC-016  
**Version:** 0.1.0  
**Status:** Draft  
**Last Updated:** November 4, 2026  
**Repository:** `naeos-intelligence`

---

## 1. Purpose

This document defines the Tool Registry for NAEOS Intelligence.

The Tool Registry provides a governed mechanism for discovering, validating, authorizing, and invoking tools exposed to intelligence agents.

Tools may interact with internal NAEOS services, external systems, repositories, issue trackers, CI pipelines, or other approved integrations.

The registry must ensure that every tool invocation is:

- Explicitly registered.
- Bound to a defined contract.
- Validated against an input schema.
- Authorized through the Policy Engine.
- Executed by the Runtime.
- Observable and auditable.
- Subject to capability and resource restrictions.

## 2. Design Principles

### 2.1 Explicit Registration

The Runtime MUST only execute tools registered and enabled in the Tool Registry.

Unregistered tools MUST be rejected.

### 2.2 Least Privilege

Each tool MUST declare the minimum capabilities required for its operation.

A tool MUST NOT inherit unrestricted permissions from the agent that requested it.

### 2.3 Policy-Enforced Execution

The Tool Registry defines what a tool is and how it can be invoked.

The Policy Engine determines whether a specific invocation is permitted.

The Runtime enforces the policy decision.

### 2.4 Schema-Validated Contracts

Every tool MUST define input and output schemas.

Inputs MUST be validated before policy evaluation and execution.

Outputs MUST be validated before being returned to the agent.

### 2.5 Separation of Proposal and Execution

Agents may propose tool invocations.

Agents MUST NOT directly execute tools or override policy decisions.

### 2.6 Auditable Operations

Every invocation MUST produce an audit record containing the tool identity, version, request context, policy decision, execution outcome, and relevant evidence references.

Sensitive values MUST NOT be written to audit logs.

---

## 3. Architectural Placement

The Tool Registry is a logical component within the NAEOS Intelligence modular monolith.

```text
Agent Orchestrator
       |
       v
Tool Discovery
       |
       v
Tool Invocation Proposal
       |
       v
Input Schema Validation
       |
       v
Policy Engine
       |
       +---- DENY ----------------> Reject Invocation
       |
       +---- REQUIRE_APPROVAL ----> Approval Workflow
       |
       +---- ALLOW
                  |
                  v
           Runtime Executor
                  |
                  v
             Tool Adapter
                  |
                  v
          External or Internal
              System
                  |
                  v
       Output Validation
                  |
                  v
       Evidence and Audit
                  |
                  v
       Agent Observation
```

The Tool Registry MUST NOT execute tools itself.

Execution is delegated to the Runtime Executor through registered adapters.

---

## 4. Tool Definition

A tool is a versioned, explicitly registered capability that performs a bounded operation.

Each tool MUST have a unique identifier.

### 4.1 Required Metadata

Every registered tool MUST define:

| Field | Description |
|---|---|
| `tool_id` | Stable unique identifier |
| `name` | Human-readable tool name |
| `description` | Purpose and operational behavior |
| `version` | Immutable tool version |
| `status` | Registration lifecycle status |
| `input_schema` | Input validation schema |
| `output_schema` | Output validation schema |
| `capabilities` | Required capability identifiers |
| `risk_level` | Operational risk classification |
| `execution_mode` | Read-only or governed mutation |
| `timeout_seconds` | Maximum execution duration |
| `max_retries` | Maximum permitted retries |
| `owner` | Responsible component or team |
| `adapter_type` | Registered adapter implementation |
| `audit_required` | Whether invocation audit is mandatory |

### 4.2 Optional Metadata

Optional fields MAY include:

- `tags`
- `documentation_url`
- `rate_limit`
- `max_input_size`
- `max_output_size`
- `resource_limits`
- `supported_environments`
- `deprecation_date`
- `replacement_tool_id`

Tool definitions MUST NOT contain plaintext credentials or secret values.

---

## 5. Tool Lifecycle

A tool MUST follow a controlled lifecycle.

| Status | Description |
|---|---|
| `draft` | Tool definition is being developed |
| `registered` | Definition has passed registration validation |
| `enabled` | Tool is available for permitted invocation |
| `disabled` | Tool is temporarily unavailable |
| `deprecated` | Tool is scheduled for removal |
| `retired` | Tool is permanently unavailable |

### Lifecycle Rules

- Only `enabled` tools may be invoked.
- `draft`, `registered`, `disabled`, `deprecated`, and `retired` tools MUST NOT be invoked unless a separately approved migration policy explicitly permits it.
- Retired tool identifiers MUST NOT be reused.
- Tool versions MUST be immutable after publication.
- Changes to schemas, capabilities, or execution behavior MUST result in a new version.
- Disabling a tool MUST prevent new invocations.
- In-flight execution cancellation MUST follow the Runtime cancellation contract.

---

## 6. Capability Model

Capabilities describe the operations a tool is permitted to perform.

Examples:

| Capability | Description |
|---|---|
| `repository.read` | Read approved repository information |
| `repository.issue.read` | Read repository issues |
| `repository.issue.create` | Create repository issues |
| `repository.pull_request.read` | Read pull request information |
| `repository.pull_request.comment` | Comment on pull requests |
| `repository.pull_request.modify` | Modify pull request content |
| `ci.run.read` | Read CI execution results |
| `ci.run.trigger` | Trigger an approved CI workflow |
| `deployment.read` | Read deployment information |
| `deployment.execute` | Execute an approved deployment |

Capability identifiers are illustrative and MUST be finalized through the capability governance process.

Capabilities MUST be scoped wherever applicable.

Examples of scope dimensions:

- Repository.
- Organization.
- Environment.
- Resource identifier.
- Operation type.
- Tenant.
- Execution context.

A capability MUST NOT grant broader access than its declared scope.

---

## 7. Input and Output Contracts

### 7.1 Input Schema

Tool input schemas MUST use a supported machine-readable schema format.

JSON Schema SHOULD be used for the initial implementation.

The schema MUST define:

- Required fields.
- Data types.
- Field constraints.
- Maximum input size.
- Allowed values where applicable.
- Additional-property behavior.

Unknown fields SHOULD be rejected unless explicitly allowed.

### 7.2 Output Schema

Tool output schemas MUST define the expected result structure.

Outputs SHOULD distinguish:

- Successful operation results.
- Structured operational errors.
- Evidence references.
- Execution metadata.

A tool MUST NOT return credentials, secret values, or unrestricted internal configuration.

### 7.3 Schema Versioning

Schema changes MUST be reviewed for compatibility.

Breaking changes MUST produce a new tool version.

The Runtime MUST validate tool outputs before exposing them to the agent.

Invalid output MUST be treated as an execution contract failure.

---

## 8. Invocation Contract

A tool invocation MUST include a structured invocation request.

Conceptual request:

```json
{
  "invocation_id": "invocation-uuid",
  "request_id": "intelligence-request-uuid",
  "tool_id": "repository.issue.read",
  "tool_version": "1.0.0",
  "input": {
    "repository": "NAEOS-foundation/naeos",
    "issue_number": 123
  },
  "context": {
    "agent_id": "engineering-agent",
    "execution_id": "execution-uuid"
  }
}
```

The example is illustrative. Actual identifiers MUST follow the implementation's identifier standards.

The Runtime MUST derive authorization context from trusted execution context.

The agent MUST NOT be allowed to fabricate or elevate its authorization context.

### 8.1 Invocation Processing

The Runtime MUST perform the following steps:

1. Resolve the registered tool and exact version.
2. Verify that the tool is enabled.
3. Validate the invocation input.
4. Resolve the trusted execution context.
5. Evaluate the invocation through the Policy Engine.
6. Initiate approval when required.
7. Verify approval against the exact proposed operation.
8. Apply execution limits.
9. Invoke the registered adapter.
10. Validate the output.
11. Persist execution metadata and audit events.
12. Return the structured result to the agent.

The Runtime MUST reject the invocation if any mandatory validation or authorization step fails.

---

## 9. Policy Integration

The Policy Engine MUST evaluate each invocation using the following context:

- Tool identifier and version.
- Requested capabilities.
- Agent identity.
- Intelligence request identity.
- Execution identity.
- Input resource scope.
- Environment.
- Risk classification.
- Relevant approval state.
- Applicable policy rules.

Policy outcomes:

- `allow`
- `deny`
- `require_approval`

### 9.1 Approval Binding

An approval MUST be bound to the specific proposed operation.

The approval context SHOULD include:

- Tool identifier.
- Tool version.
- Normalized input.
- Target resource.
- Requested capabilities.
- Request identity.
- Expiration.
- Approval identity.

Changes to the approved operation MUST invalidate the approval.

Approval MUST NOT grant reusable unrestricted access to a tool.

### 9.2 Fail-Closed Behavior

If the Policy Engine is unavailable, returns an invalid decision, or cannot establish the required authorization context, the Runtime MUST NOT execute the tool.

---

## 10. Tool Risk Classification

Tools MUST declare a risk level.

| Risk Level | Description | Default Control |
|---|---|---|
| `low` | Read-only, bounded operations | Policy authorization |
| `medium` | Limited mutations or externally visible actions | Explicit policy evaluation |
| `high` | Consequential changes to repositories, systems, or environments | Human approval unless explicitly exempted by approved policy |
| `critical` | Irreversible or highly consequential operations | Disabled by default; dedicated authorization controls |

Risk classification MUST NOT replace capability-based authorization.

A low-risk tool can still be denied if the requested scope is unauthorized.

A high-risk tool MUST NOT be made safe solely by changing its declared risk level.

---

## 11. Execution Modes

### 11.1 Read-Only

Read-only tools retrieve information without intentionally changing external state.

Examples:

- Repository metadata retrieval.
- Issue retrieval.
- CI result retrieval.
- Documentation search.

Read-only tools MUST still enforce authorization, resource scoping, and audit requirements.

### 11.2 Governed Mutation

Governed mutation tools modify external or internal state.

Examples:

- Creating an issue.
- Adding a pull request comment.
- Triggering an approved workflow.

Mutation tools MUST require explicit capability authorization.

Human approval MUST be required when the applicable policy determines that the operation is consequential.

### 11.3 Prohibited Initial Execution Modes

The initial MVP MUST NOT expose:

- Unrestricted shell execution.
- Arbitrary code execution.
- Unrestricted filesystem access.
- Unrestricted network access.
- Direct model-controlled production deployment.
- Unrestricted repository modification.
- Autonomous pull request merging.

Any future introduction of these capabilities requires a separate security review and explicit policy design.

---

## 12. Adapter Interface

Each tool MUST be implemented through a registered adapter.

Conceptual interface:

```go
type ToolAdapter interface {
    Execute(
        ctx context.Context,
        input json.RawMessage,
    ) (json.RawMessage, error)
}
```

This interface is illustrative and MUST be reconciled with the final Runtime implementation.

Adapter requirements:

- MUST honor execution context cancellation.
- MUST respect configured timeouts.
- MUST NOT bypass policy authorization.
- MUST NOT independently elevate privileges.
- MUST NOT expose credentials in errors.
- MUST return structured results.
- MUST report execution failures through normalized errors.
- MUST NOT execute unregistered operations.

Adapters SHOULD be stateless where practical.

Persistent adapter state MUST be explicitly documented and governed.

---

## 13. Authentication and Secret Handling

Tools that interact with external systems MUST use approved credential mechanisms.

The Tool Registry MUST store references to credential providers, not plaintext credentials.

Requirements:

- Credentials MUST be supplied through a trusted execution context.
- Secrets MUST NOT be included in agent prompts.
- Secrets MUST NOT be included in tool input or output schemas.
- Secrets MUST NOT be written to logs, traces, or audit records.
- Credentials MUST be scoped to the required operations.
- Credential rotation MUST be supported by the integration architecture.
- Missing or invalid credentials MUST result in a controlled failure.

The authentication mechanism remains subject to a separate architecture decision.

---

## 14. Error Handling

Tool invocation errors MUST use normalized categories.

| Error Code | Description |
|---|---|
| `TOOL_NOT_FOUND` | Tool or version is not registered |
| `TOOL_DISABLED` | Tool is not enabled |
| `TOOL_INPUT_INVALID` | Input failed schema validation |
| `TOOL_OUTPUT_INVALID` | Output failed schema validation |
| `TOOL_POLICY_DENIED` | Policy denied invocation |
| `TOOL_APPROVAL_REQUIRED` | Approval is required |
| `TOOL_APPROVAL_INVALID` | Approval does not match the operation |
| `TOOL_TIMEOUT` | Execution exceeded its timeout |
| `TOOL_RATE_LIMITED` | Invocation exceeded rate limits |
| `TOOL_CREDENTIAL_ERROR` | Credential resolution or authentication failed |
| `TOOL_EXECUTION_FAILED` | Adapter execution failed |
| `TOOL_UNAVAILABLE` | External dependency is unavailable |

Error responses MUST NOT disclose sensitive internal details.

Retry behavior MUST be explicitly defined per tool.

Mutation tools MUST NOT be retried automatically unless idempotency behavior is established.

---

## 15. Rate Limits and Resource Controls

Every tool MUST be subject to applicable execution limits.

Controls MAY include:

- Maximum execution duration.
- Maximum invocation count.
- Maximum concurrent invocations.
- Maximum input size.
- Maximum output size.
- Maximum retries.
- External API rate limits.
- Tenant-level quotas.
- Agent-level quotas.

The Runtime MUST enforce these controls independently of agent instructions.

A tool MUST NOT be able to override its own configured limits.

---

## 16. Audit and Observability

Each invocation MUST produce an audit event.

Required audit attributes:

- Invocation ID.
- Intelligence request ID.
- Execution ID.
- Tool ID.
- Tool version.
- Agent ID.
- Policy decision reference.
- Approval reference, when applicable.
- Execution status.
- Start and completion timestamps.
- Duration.
- Error code, when applicable.
- Evidence references.

Observability SHOULD include:

- Invocation count.
- Success and failure rates.
- Latency distribution.
- Timeout rate.
- Policy denial rate.
- Approval-required rate.
- Adapter availability.
- External dependency failures.

Audit records MUST remain distinct from agent observations and tool output.

---

## 17. Tool Registration and Review

A tool MUST undergo a registration review before being enabled.

The review SHOULD evaluate:

1. Purpose and use cases.
2. Input and output schemas.
3. Capability requirements.
4. Risk classification.
5. Resource scope.
6. Authentication requirements.
7. Error handling.
8. Idempotency behavior.
9. Audit coverage.
10. Security and testing evidence.

The registration process MUST reject tools that lack required metadata or have unresolved security requirements.

Tool registration and enablement MUST be controlled operations.

The implementation MUST define who or what is authorized to register, enable, disable, or retire tools.

---

## 18. Disable and Emergency Controls

The system MUST support disabling a tool without requiring an application-wide shutdown.

When a tool is disabled:

- New invocations MUST be rejected.
- The rejection MUST be auditable.
- In-flight execution behavior MUST follow the configured cancellation policy.
- Agents MUST receive a structured unavailable response.
- The tool MUST NOT be automatically re-enabled.

The system SHOULD support an operational kill switch for integrations with elevated risk.

---

## 19. Security Requirements

The Tool Registry and Runtime MUST enforce the following requirements:

- Unregistered tools cannot execute.
- Disabled tools cannot execute.
- Tool input is validated before execution.
- Policy authorization is mandatory.
- Approval cannot be reused for a different operation.
- Tool execution cannot elevate agent privileges.
- Credentials remain outside model-visible context.
- External results are treated as untrusted data.
- Tool outputs cannot modify policy configuration.
- Tool errors cannot expose secrets.
- Runtime limits cannot be overridden by agents.
- All consequential operations are auditable.

External tool results MUST be treated as potentially malicious, including content that attempts to instruct the agent to bypass system policies.

---

## 20. Testing Requirements

The Tool Registry MUST be covered by automated tests.

### Unit Tests

- Tool definition validation.
- Version uniqueness.
- Schema validation.
- Lifecycle transition validation.
- Capability declaration validation.
- Error normalization.

### Integration Tests

- Registry lookup and version resolution.
- Policy Engine integration.
- Approval binding.
- Adapter invocation.
- Audit event persistence.
- Timeout and cancellation behavior.

### Security Tests

- Invocation of unregistered tools.
- Invocation of disabled tools.
- Capability escalation attempts.
- Approval replay.
- Input schema bypass.
- Secret leakage through logs.
- Malicious tool output.
- Unauthorized resource access.

### Failure Tests

- Policy Engine unavailable.
- Adapter unavailable.
- Invalid adapter output.
- External dependency timeout.
- Rate-limit exhaustion.
- Database persistence failure.

---

## 21. Acceptance Criteria

The Tool Registry specification is considered implemented when:

- [ ] Tools can be registered with immutable versions.
- [ ] Tool lifecycle states are enforced.
- [ ] Tool discovery only exposes authorized tools.
- [ ] Tool input and output schemas are validated.
- [ ] All invocations pass through the Policy Engine.
- [ ] Approval requirements are enforced by the Runtime.
- [ ] Registered adapters execute within configured limits.
- [ ] Unregistered and disabled tools are rejected.
- [ ] Invocation audit events are persisted.
- [ ] Sensitive values are excluded from audit records.
- [ ] Tool disablement prevents new execution.
- [ ] Automated unit, integration, and security tests pass.

---

## 22. Related Documents

- `README.md`
- `DOCUMENTATION-AUTHORITY.md`
- `docs/vision/PROJECT-VISION.md`
- `docs/vision/PROJECT-SCOPE.md`
- `docs/architecture/SYSTEM-ARCHITECTURE.md`
- `docs/architecture/COMPONENT-BOUNDARIES.md`
- `docs/architecture/DATA-FLOW.md`
- `docs/security/THREAT-MODEL.md`
- `docs/security/SECURITY-REQUIREMENTS.md`
- `docs/specifications/API-CONTRACT.md`
- `docs/specifications/REQUEST-LIFECYCLE.md`
- `docs/specifications/DATA-MODEL.md`
- `docs/specifications/POLICY-ENGINE-SPECIFICATION.md`
- `docs/specifications/AGENT-EXECUTION-MODEL.md`

---

## 23. Open Decisions

The following decisions remain subject to implementation planning and architecture review:

- Tool registry persistence strategy.
- Tool definition storage format.
- Adapter loading mechanism.
- Capability taxonomy and canonical identifiers.
- Tool registration authorization model.
- Secret provider integration.
- Tool-specific rate limit defaults.
- Approval UX and approval authority.
- Tool version compatibility policy.
- External integration test strategy.

These decisions MUST be documented before production implementation where they materially affect security, compatibility, or execution behavior.
