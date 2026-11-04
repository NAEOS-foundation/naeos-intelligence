# Requirement Traceability Matrix

**Document ID:** NAEOS-INT-TRACE-001  
**Version:** 0.1.0  
**Status:** Draft — Proposed  
**Purpose:** Establish a reviewable mapping from requirements to normative sources, implementation phases, verification, and evidence. This matrix records current traceability gaps; it does not imply implementation or acceptance.

## 1. Status vocabulary

- **Specified:** A requirement exists in a referenced specification.
- **Mapped:** The expected implementation area and verification approach are identified.
- **Implemented:** Implementation evidence has been inspected and verified.
- **Accepted:** Maintainer acceptance is recorded for the applicable phase.
- **Gap:** The requirement lacks a sufficiently concrete criterion, implementation mapping, or verification evidence.
- **Blocked:** A decision or dependency must be resolved before implementation can be accepted.

Do not mark a row Implemented or Accepted based only on a specification or a green CI run.

## 2. MVP and API traceability

| Requirement / capability | Normative source | Implementation phase | Verification expectation | Current traceability status |
|---|---|---:|---|---|
| Protected API requests authenticate | `docs/specifications/API-CONTRACT.md` §4; `docs/security/SECURITY-REQUIREMENTS.md` SEC-AUTH-001 | 5 | Tests for missing, invalid, expired credentials; protected route coverage | Specified; identity-provider/token decision blocked |
| Validated principal attribution | Security Requirements SEC-AUTH-002; API Contract §4 | 5 | Principal context derived from validated credentials; spoofed headers rejected | Specified; claim mapping decision blocked |
| Service-to-service identity | Security Requirements SEC-AUTH-003 | 5 | Integration tests verify authenticated service and authorization context | Specified; mechanism/provider unresolved |
| Least-privilege authorization | Security Requirements SEC-AC-001; API Contract §4 | 5 | Positive/negative permission matrix across protected operations | Specified; permission vocabulary and enforcement map gap |
| Tenant isolation (if multi-tenant) | Security Requirements SEC-AC-002; Data Model §6–7 | 5, 4 | Cross-tenant denial and tenant-scoped persistence tests | Conditional requirement; tenant model blocked |
| Model outputs cannot grant permissions | Security Requirements SEC-AC-003; Project Vision principle “Policy-Aware Intelligence” | 6–7 | Adversarial model-output test demonstrates authorization unchanged | Specified; implementation/evidence not mapped to test IDs |
| API input schema validation | Security Requirements SEC-INPUT-001; API Contract | 3 | Schema boundary, malformed and contract tests | Specified; test inventory gap |
| Context retrieval authorization and provenance | Security Requirements SEC-CTX-001/002; Project Scope §4.2 | MVP / integration phase | Unauthorized-source denial; provenance assertions | Specified; connector-specific tests to be mapped |
| Prompt-injection/untrusted context isolation | Security Requirements SEC-CTX-003 | 7, 12 | Adversarial prompt/context evaluation | Specified; evaluation cases and evidence location gap |
| Executable actions require policy evaluation | Security Requirements SEC-POL-001 | 6–7 | Runtime denial if policy decision absent | Later-phase requirement; not part of read-only MVP execution |
| Policy errors fail closed | Security Requirements SEC-POL-002 | 6–7 | Policy timeout/error/indeterminate tests deny execution | Specified; failure semantics need detailed contract |
| Policy decision bound to proposal | Security Requirements SEC-POL-003 | 6–7 | Digest binding and mutation invalidation tests | Specified; digest format/version and expiry gap |
| Final authorization revalidation | Security Requirements SEC-POL-004 | 7 | Stale decision and relevant resource-state change tests | Specified; resource-state/TOCTOU semantics gap |
| Approval is enforced | Security Requirements SEC-APP-001; Approval Workflow Specification | 10 | State-transition and bypass tests | Specified; state machine/actor rules need completion |
| Durable request lifecycle and attempts | API Contract §6; Data Model; Phase 4 persistence prompt | 3–4 | Transition integrity, restart recovery, idempotency tests | Phase 4 implementation exists; map exact test/evidence identifiers |
| Evidence distinguishable from model claims | Project Vision §Design Principles; Data Model §2; Observability Specification | 11–12 | Evidence schema and provenance tests; claims not treated as verification | Specified; evidence taxonomy/retention crosswalk gap |
| Read-only MVP boundary | Project Scope §§3–5; README Initial Scope | MVP | Integration allowlist and test demonstrating no write-capable tool path | Documented by proposed scope update; implementation test evidence pending |
| Security audit events and privacy | Security Requirements SEC-AUD/SEC-PRIV/SEC-OBS; Observability Specification | 11–12 | Audit integrity, redaction, retention, access tests | Specified; cross-document retention alignment gap |

## 3. Requirement-to-evidence lifecycle

Each implementation PR or phase acceptance should identify:

1. Stable requirement ID (existing SEC-* identifiers or an assigned domain prefix).
2. Source document and section.
3. Code package/file or component owner.
4. Test name(s) and command(s).
5. CI run/check URL or other immutable evidence reference.
6. Phase acceptance record and reviewer.
7. Any deviation, risk, and follow-up owner.

Suggested compact record:

| Requirement ID | Source | Implementation | Test(s) | Evidence | Phase acceptance |
|---|---|---|---|---|---|
| SEC-AUTH-001 | Security Requirements §4 | TBD | TBD | TBD | Blocked |
| SEC-AUTH-002 | Security Requirements §4 | TBD | TBD | TBD | Blocked |
| SEC-AC-002 | Security Requirements §5 | TBD | TBD | TBD | Blocked pending tenant decision |

## 4. Known cross-document gaps

- API Contract leaves authentication mechanism open; Phase 5 prompt correctly forbids choosing IdP/token format while unresolved.
- Data Model treats tenant_id as conditional, while API authorization and SEC-AC-002 depend on tenant applicability. An explicit tenancy decision is required.
- Security requirements reference Policy Engine authority; concrete policy decision structure, versioning, expiry, proposal binding and runtime enforcement need one contract and corresponding tests.
- Approval Workflow must define the state machine, authorized approver identity/role, expiry/revocation, idempotency and proposal-change invalidation.
- Audit, evidence, context and observability docs need a consistent privacy/retention/redaction model.
- The roadmap's phase numbers should be cross-referenced to README milestone phases; the two numbering schemes describe different decompositions.

## 5. Governance

This matrix is a tracking artifact and is not itself acceptance of any unresolved architecture decision. Update it in the same reviewed PR that adds or changes a normative requirement, implementation, or verification criterion. Security-sensitive rows must not be marked Accepted without test evidence and phase acceptance.
