# Approval Workflow Specification

**Document ID:** NAEOS-INT-SPEC-021  
**Version:** 0.1.0  
**Status:** Draft  
**Last Updated:** November 4, 2026  
**Repository:** `naeos-intelligence`

---

## 1. Purpose

This document defines the human-in-the-loop approval workflow for NAEOS Intelligence.

The Approval Workflow provides a controlled mechanism for reviewing and authorizing operations that require explicit approval before execution.

The subsystem MUST ensure that:

- Approval requirements are determined by policy.
- Only authorized approvers may approve operations.
- Approval decisions are bound to specific proposed operations.
- Approval cannot be reused for a different operation.
- Expired or revoked approvals cannot authorize execution.
- Approval decisions are auditable.
- The Runtime remains responsible for execution enforcement.

## 2. Design Principles

### 2.1 Human Authority

An agent MUST NOT approve its own proposed operation.

The agent MUST NOT fabricate approval records or modify approval decisions.

### 2.2 Operation Binding

Approval MUST apply to a specific operation, including its target and normalized parameters.

Changing the operation MUST invalidate the existing approval.

### 2.3 Least Privilege

Approvers MUST only be authorized to approve operations within their assigned scope.

### 2.4 Explicit Decision

Approval MUST require an explicit decision from an authorized approver.

Silence, timeout, or approver unavailability MUST NOT be interpreted as approval.

### 2.5 Auditability

Every approval lifecycle transition MUST produce an auditable event.

### 2.6 Fail Closed

If approval validity cannot be established, the operation MUST NOT execute.

---

## 3. Approval Workflow Architecture

```text
Agent Proposes Operation
          |
          v
Runtime Validates Proposal
          |
          v
Policy Engine Evaluation
          |
          +---- DENY ----------------> Reject
          |
          +---- ALLOW ---------------> Execute
          |
          +---- REQUIRE_APPROVAL
                    |
                    v
            Approval Request
                    |
                    v
             Authorized Review
                    |
          +---------+---------+
          |         |         |
          v         v         v
       APPROVE    REJECT    EXPIRE
          |         |         |
          v         v         v
    Revalidate    Reject    Reject
    Approval
          |
          v
    Runtime Executes
          |
          v
    Audit and Evidence
```

The Approval Workflow MUST NOT directly execute tools.

Approved operations MUST return to the Runtime for final authorization and execution.

---

## 4. Approval Lifecycle

The approval request MUST use the following lifecycle states:

| State | Description |
|---|---|
| `pending` | Awaiting an authorized decision |
| `approved` | Approved by an authorized approver |
| `rejected` | Explicitly rejected |
| `expired` | Approval request expired |
| `cancelled` | Approval request cancelled |
| `consumed` | Approval successfully used for its bound operation |
| `invalidated` | Approval invalidated by a relevant change |

### 4.1 Lifecycle Rules

- Approval requests MUST begin in `pending`.
- Only authorized approvers may transition a request to `approved` or `rejected`.
- Expired requests MUST NOT be approved.
- Cancelled requests MUST NOT be approved.
- Invalidated approvals MUST NOT authorize execution.
- A consumed approval MUST NOT be reused.
- Lifecycle transitions MUST be auditable.

---

## 5. Approval Request Contract

An approval request SHOULD contain:

| Field | Description |
|---|---|
| `approval_id` | Unique approval request identifier |
| `request_id` | Intelligence request identifier |
| `execution_id` | Agent execution identifier |
| `inv
