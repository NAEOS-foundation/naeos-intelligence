# Observability Specification

**Document ID:** NAEOS-INT-SPEC-017  
**Version:** 0.1.0  
**Status:** Draft  
**Last Updated:** November 4, 2026  
**Repository:** `naeos-intelligence`

---

## 1. Purpose

This document defines the observability requirements for NAEOS Intelligence.

The observability subsystem provides visibility into system health, agent execution, tool invocation, policy decisions, request lifecycle, and operational reliability.

The system MUST provide sufficient observability to:

- Diagnose execution failures.
- Trace requests across components.
- Monitor service health and reliability.
- Identify latency and resource bottlenecks.
- Detect policy enforcement failures.
- Investigate security-relevant events.
- Support operational incident response.

Observability MUST NOT compromise data confidentiality, authorization boundaries, or tenant isolation.

## 2. Observability Principles

### 2.1 Structured Telemetry

Logs, metrics, and traces MUST use structured formats.

### 2.2 Correlation

Every intelligence request MUST have a stable request identifier.

Related execution, tool invocation, policy decision, and audit events SHOULD reference that identifier.

### 2.3 Separation of Telemetry and Audit

Operational telemetry and security audit records serve different purposes.

- Telemetry supports monitoring and troubleshooting.
- Audit records support accountability and security investigations.

Audit records MUST NOT depend solely on the availability of operational telemetry.

### 2.4 Sensitive Data Protection

Telemetry MUST NOT contain:

- Plaintext credentials.
- API keys or access tokens.
- Passwords.
- Private cryptographic keys.
- Unredacted secret values.
- Unrestricted model prompts or outputs by default.

Sensitive payload capture requires an explicit, reviewed configuration.

### 2.5 Reliability

Telemetry failures SHOULD NOT prevent ordinary read-only operations.

However, if a mandatory audit record cannot be persisted for a consequential operation, the Runtime MUST follow a fail-closed policy.

---

## 3. Observability Architecture

```text
Intelligence API
      |
      v
Request Context
      |
      +------------------+
      |                  |
      v                  v
 Structured Logs      Trace Context
      |                  |
      v                  v
 Telemetry SDK       Distributed Tracing
      |                  |
      +--------+---------+
               |
               v
      Telemetry Pipeline
               |
      +--------+---------+
      |        |         |
      v        v         v
    Logs    Metrics    Traces

Security-Relevant Operations
               |
               v
        Audit Subsystem
               |
               v
     Durable Audit Storage
```

The implementation SHOULD use OpenTelemetry-compatible instrumentation.

The initial implementation SHOULD avoid introducing a separate observability service unless operational requirements justify it.

---

## 4. Correlation Identifiers

The system MUST support the following identifiers:

| Identifier | Purpose |
|---|---|
| `request_id` | Identifies an intelligence request |
| `execution_id` | Identifies an agent execution |
| `attempt_id` | Identifies an execution attempt |
| `invocation_id` | Identifies a tool invocation |
| `trace_id` | Identifies a distributed trace |
| `span_id` | Identifies an individual trace span |
| `policy_decision_id` | Identifies a policy evaluation |
| `approval_id` | Identifies an approval record |
| `audit_event_id` | Identifies an audit event |

Identifiers MUST be generated or validated by trusted system components.

Agents MUST NOT be permitted to overwrite trusted correlation identifiers.

The identifiers SHOULD be propagated through internal service boundaries and adapter calls where technically supported.

---

## 5. Logging Specification

### 5.1 Log Format

Structured JSON logging MUST be used.

A log entry SHOULD contain:

```json
{
  "timestamp": "2026-11-04T10:00:00Z",
  "level": "INFO",
  "service": "naeos-intelligence",
  "component": "runtime",
  "event": "tool_invocation_completed",
  "request_id": "request-uuid",
  "execution_id": "execution-uuid",
  "invocation_id": "invocation-uuid",
  "trace_id": "trace-id",
  "duration_ms": 125,
  "status": "completed"
}
```

The example is illustrative.

### 5.2 Log Levels

| Level | Usage |
|---|---|
| `DEBUG` | Detailed development diagnostics |
| `INFO` | Normal lifecycle and operational events |
| `WARN` | Recoverable abnormal conditions |
| `ERROR` | Failed operations requiring investigation |

Debug logging MUST NOT be enabled by default in production.

### 5.3 Required Events

The system SHOULD log the following events:

- Request accepted.
- Request rejected.
- Execution started.
- Execution completed.
- Execution failed.
- Execution cancelled.
- Policy decision generated.
- Approval requested.
- Approval granted or denied.
- Tool invocation started.
- Tool invocation completed.
- Tool invocation failed.
- External dependency unavailable.
- Runtime timeout.
- Configuration validation failure.

Logs MUST avoid recording complete input or output payloads unless explicitly authorized.

---

## 6. Metrics Specification

Metrics MUST support operational monitoring and reliability analysis.

### 6.1 Request Metrics

| Metric | Type | Description |
|---|---|---|
| `intelligence_requests_total` | Counter | Total accepted requests |
| `intelligence_requests_completed_total` | Counter | Completed requests |
| `intelligence_requests_failed_total` | Counter | Failed requests |
| `intelligence_requests_cancelled_total` | Counter | Cancelled requests |
| `intelligence_request_duration_seconds` | Histogram | Request duration |

### 6.2 Agent Metrics

| Metric | Type | Description |
|---|---|---|
| `agent_executions_total` | Counter | Total agent executions |
| `agent_execution_duration_seconds` | Histogram | Agent execution duration |
| `agent_execution_failures_total` | Counter | Agent execution failures |
| `agent_steps_total` | Counter | Total agent steps |
| `agent_step_duration_seconds` | Histogram | Agent step duration |

### 6.3 Tool Metrics

| Metric | Type | Description |
|---|---|---|
| `tool_invocations_total` | Counter | Total tool invocations |
| `tool_invocation_duration_seconds` | Histogram | Tool execution duration |
| `tool_invocation_failures_total` | Counter | Tool execution failures |
| `tool_invocation_timeouts_total` | Counter | Tool timeouts |
| `tool_policy_denials_total` | Counter | Policy-denied invocations |

### 6.4 Policy Metrics

| Metric | Type | Description |
|---|---|---|
| `policy_evaluations_total` | Counter | Total policy evaluations |
| `policy_evaluation_duration_seconds` | Histogram | Policy evaluation latency |
| `policy_denials_total` | Counter | Denied operations |
| `policy_approval_required_total` | Counter | Operations requiring approval |

### 6.5 Infrastructure Metrics

The implementation SHOULD monitor:

- HTTP request duration.
- HTTP error rates.
- Database connection pool utilization.
- Database query duration.
- Database transaction failures.
- Worker queue depth.
- Worker utilization.
- Memory usage.
- CPU utilization.
- External dependency latency.
- External dependency availability.

Metric labels MUST use bounded cardinality.

Request IDs, execution IDs, user identifiers, and arbitrary resource identifiers MUST NOT be used as metric labels.

---

## 7. Distributed Tracing

The system SHOULD implement distributed tracing using OpenTelemetry.

### 7.1 Required Trace Boundaries

Trace spans SHOULD cover:

- API request processing.
- Request persistence.
- Agent execution.
- Model provider interaction.
- Policy evaluation.
- Approval resolution.
- Tool invocation.
- Adapter execution.
- Database operations.

### 7.2 Span Attributes

Relevant span attributes MAY include:

- `service.name`
- `component.name`
- `request.id`
- `execution.id`
- `tool.id`
- `tool.version`
- `policy.outcome`
- `execution.status`
- `error.code`

Sensitive input and output payloads MUST NOT be attached to spans by default.

### 7.3 Trace Propagation

Trace context SHOULD be propagated across internal service boundaries.

External integrations MUST only receive trace context where propagation is supported and approved.

Untrusted incoming trace context MUST NOT be used as an authorization signal.

---

## 8. Audit Integration

Security-relevant operations MUST generate durable audit records.

Examples include:

- Tool invocation authorization.
- Policy decisions.
- Approval decisions.
- Capability changes.
- Tool registration.
- Tool enablement or disablement.
- Credential configuration changes.
- Consequential mutations.

Audit records SHOULD include:

- Event identity.
- Timestamp.
- Actor identity.
- Request identity.
- Operation identity.
- Resource scope.
- Policy decision reference.
- Approval reference.
- Outcome.
- Evidence references.

Audit records MUST be protected against unauthorized modification.

Audit retention MUST be defined by the applicable governance and operational requirements.

The observability pipeline MUST NOT be treated as the authoritative audit store.

---

## 9. Health Checks

The API MUST expose a health endpoint.

Initial endpoint:

`GET /v1/health`

The implementation SHOULD distinguish:

### 9.1 Liveness

Indicates whether the process is alive and able to continue execution.

Liveness MUST NOT depend on every external integration being available.

### 9.2 Readiness

Indicates whether the service is ready to accept requests.

Readiness SHOULD consider required dependencies such as database connectivity and mandatory runtime initialization.

Optional model providers and external tools SHOULD NOT automatically make the entire service unready.

The exact health response contract MUST be aligned with `api/openapi.yaml`.

---

## 10. Alerting Requirements

The deployment SHOULD support alerts for:

- Elevated request failure rates.
- Excessive request latency.
- Agent execution failures.
- Tool invocation failures.
- Repeated policy evaluation errors.
- Database unavailability.
- Worker queue saturation.
- External provider outages.
- Audit persistence failures.
- Unexpected execution volume.
- Consequential operation anomalies.

Alert thresholds MUST be configurable.

Thresholds MUST be established using observed baseline behavior rather than arbitrary permanent defaults.

---

## 11. Sampling and Data Retention

Trace sampling MAY be used to manage telemetry volume.

Sampling MUST NOT eliminate mandatory audit records.

The system SHOULD support:

- Configurable trace sampling.
- Higher sampling for error traces.
- Reduced sampling for routine successful operations.
- Independent audit retention.
- Telemetry retention policies.
- Controlled access to historical telemetry.

Retention durations remain an operational decision.

---

## 12. Privacy and Access Control

Observability data MUST be protected by access controls.

Requirements:

- Access MUST be limited to authorized operators.
- Sensitive data MUST be redacted before persistence.
- Production telemetry access MUST be auditable.
- Tenant boundaries MUST be preserved where applicable.
- Debug logging MUST be controlled.
- Prompt and response capture MUST be disabled by default.
- Exported telemetry MUST follow approved data handling rules.

---

## 13. Failure Handling

Telemetry failures MUST be handled according to their operational importance.

| Failure | Required Behavior |
|---|---|
| Log exporter unavailable | Buffer or degrade according to configured limits |
| Metrics exporter unavailable | Continue operation where safe |
| Trace exporter unavailable | Continue operation where safe |
| Audit persistence unavailable | Fail closed for mandatory audited actions |
| Health dependency unavailable | Report degraded readiness |
| Telemetry buffer exhausted | Drop non-critical telemetry according to configured policy |
| Sensitive data detected | Redact or reject telemetry record |

Telemetry buffering MUST be bounded.

The system MUST NOT allow telemetry backpressure to cause unbounded memory growth.

---

## 14. Security Requirements

The observability subsystem MUST:

- Prevent secret leakage.
- Prevent telemetry from influencing authorization.
- Protect audit integrity.
- Enforce access controls.
- Use bounded metric cardinality.
- Separate audit records from ordinary logs.
- Avoid unrestricted payload capture.
- Preserve correlation across execution boundaries.
- Prevent untrusted trace context from elevating privileges.

---

## 15. Testing Requirements

### Unit Tests

- Structured log field generation.
- Metric instrumentation.
- Correlation identifier propagation.
- Sensitive data redaction.
- Metric label cardinality controls.

### Integration Tests

- Trace propagation across components.
- Audit record persistence.
- Health endpoint behavior.
- Telemetry exporter failure handling.
- Database and worker instrumentation.

### Security Tests

- Secret leakage through logs.
- Unauthorized telemetry access.
- Unauthorized audit access.
- High-cardinality metric abuse.
- Trace context manipulation.
- Audit persistence failure during consequential operations.

---

## 16. Acceptance Criteria

The observability specification is considered implemented when:

- [ ] Structured JSON logging is available.
- [ ] Request and execution correlation identifiers are propagated.
- [ ] Core request metrics are instrumented.
- [ ] Agent execution metrics are instrumented.
- [ ] Tool invocation metrics are instrumented.
- [ ] Policy evaluation metrics are instrumented.
- [ ] Distributed tracing is supported.
- [ ] Audit events are persisted independently.
- [ ] Health checks distinguish liveness and readiness.
- [ ] Sensitive data is redacted.
- [ ] Telemetry buffering is bounded.
- [ ] Observability failure behavior is tested.
- [ ] Security-related observability tests pass.

---

## 17. Open Decisions

The following decisions remain subject to architecture and operational review:

- Telemetry backend selection.
- Metrics storage and dashboard platform.
- Trace sampling defaults.
- Log retention duration.
- Audit retention duration.
- Alerting platform.
- Production telemetry access model.
- Audit storage integrity mechanism.
- Sensitive payload capture policy.
- Service-level objectives and alert thresholds.

---

## 18. Related Documents

- `README.md`
- `DOCUMENTATION-AUTHORITY.md`
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
- `docs/specifications/TOOL-REGISTRY-SPECIFICATION.md`
