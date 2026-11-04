# Error Handling Specification

**Document ID:** NAEOS-INT-SPEC-020  
**Version:** 0.1.0  
**Status:** Draft  
**Last Updated:** November 4, 2026  
**Repository:** `naeos-intelligence`

---

## 1. Purpose

This document defines the error handling, failure classification, retry, timeout, cancellation, and recovery requirements for NAEOS Intelligence.

The error handling subsystem MUST provide:

- Consistent error classification.
- Normalized error responses.
- Bounded retries.
- Controlled timeout behavior.
- Safe cancellation.
- Dependency failure handling.
- Recovery mechanisms.
- Error observability.
- Security-safe error reporting.

Errors MUST be handled without bypassing policy enforcement, authorization, approval, or audit requirements.

## 2. Design Principles

### 2.1 Fail Closed

Authorization and policy enforcement failures MUST prevent consequential execution.

### 2.2 Bounded Recovery

Retries and recovery attempts MUST be bounded by configured limits.

### 2.3 Error Normalization

Internal components MUST use normalized error categories.

### 2.4 Context Preservation

Errors SHOULD preserve request, execution, attempt, and invocation correlation identifiers.

### 2.5 Sensitive Data Protection

Error messages MUST NOT expose secrets, credentials, internal tokens, or sensitive provider payloads.

### 2.6 No Silent Failure

Failures MUST be observable and recorded according to their operational importance.

---

## 3. Error Architecture

```text id="l7twkq"
Component Failure
       |
       v
Error Classification
       |
       v
Error Normalization
       |
       v
Recovery Eligibility
       |
       +---- Non-Retryable ----> Record Failure
       |
       +---- Retryable --------> Retry Budget Check
                                      |
                                      v
                               Retry or Exhaust
                                      |
                                      v
                               Final Error
                                      |
                 +--------------------+-------------------+
                 |                    |                   |
                 v                    v                   v
           API Response          Audit Record       Telemetry
```

Each subsystem MUST classify errors before passing them to the Runtime.

The Runtime is responsible for coordinating recovery behavior.

---

## 4. Error Categories

The system MUST support the following error categories.

| Category | Description |
|---|---|
| `validation` | Invalid request or schema |
| `authentication` | Authentication failure |
| `authorization` | Authorization failure |
| `policy` | Policy evaluation or enforcement failure |
| `approval` | Approval workflow failure |
| `provider` | Model provider failure |
| `tool` | Tool execution failure |
| `dependency` | External dependency failure |
| `database` | Database operation failure |
| `timeout` | Operation exceeded its time limit |
| `cancelled` | Operation was cancelled |
| `resource_limit` | Configured resource limit exceeded |
| `internal` | Unexpected internal failure |

Error categories MUST remain stable across API and internal component boundaries.

---

## 5. Normalized Error Contract

Internal errors SHOULD use a structured representation.

Conceptual Go structure:

```go id="9skqzr"
type IntelligenceError struct {
    Code       string
    Category   string
    Message    string
    Retryable  bool
    RequestID  string
    ExecutionID string
    Cause      error
}
```

This structure is illustrative and MUST be reconciled with the implementation.

### 5.1 Required Properties

Normalized errors MUST contain:

