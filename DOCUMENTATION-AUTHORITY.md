# Documentation Authority

**Project:** NAEOS Intelligence  
**Document ID:** NAEOS-INT-DOC-AUTH-001  
**Version:** 0.1.0  
**Status:** Draft — Proposed  
**Classification:** Normative

## 1. Purpose

This document establishes the authority, hierarchy, and governance of technical documentation within the NAEOS Intelligence repository.

Its purpose is to ensure that architecture, specifications, implementation requirements, and engineering decisions remain consistent, traceable, and independently reviewable.

## 2. Scope

This document applies to all project documentation, including:

- Project vision and scope.
- Architecture documentation.
- Technical specifications.
- API contracts.
- Security documentation.
- Architecture Decision Records (ADRs).
- Implementation documentation.
- Operational documentation.
- Testing and evaluation documentation.

## 3. Documentation Hierarchy

The documentation authority hierarchy is defined as follows:

| Level | Document Category | Authority |
|---|---|---|
| L0 | Project Constitution | Foundational principles and constraints |
| L1 | Reference Architecture | System boundaries and architectural principles |
| L2 | Master Technical Specification | System-level requirements |
| L3 | Domain Specifications | Component-level requirements |
| L4 | API Contracts | Interface behavior and compatibility |
| L5 | Architecture Decision Records | Recorded architectural decisions |
| L6 | Implementation Documentation | Implementation-specific details |
| L7 | Experiments and Evaluation Reports | Empirical findings and observations |
| L8 | Informational Documentation | Guides, onboarding, and explanatory content |

This hierarchy establishes precedence for normative requirements. ADRs record decisions and rationale; an ADR MUST remain consistent with higher-level requirements unless an explicit amendment is approved. An accepted ADR is authoritative for the specific decision it records, but it does not silently override an API contract or another normative specification. When a decision changes a higher-level requirement or contract, the affected document(s) MUST be amended in the same reviewed change. Where a conflict remains, the conflict-resolution process in Section 9 applies; implementation assumptions MUST NOT be used to choose a winner.

## 4. Normative Language

The following keywords are used in normative documentation:

- **MUST**: An absolute requirement.
- **MUST NOT**: An absolute prohibition.
- **SHOULD**: A recommended requirement that may be deviated from with documented justification.
- **SHOULD NOT**: A discouraged practice that may be used with documented justification.
- **MAY**: An optional capability or behavior.

These keywords are interpreted according to their conventional RFC-style meaning.

## 5. Document Status

Every normative document MUST declare a status.

| Status | Meaning |
|---|---|
| Draft | Under development; not approved |
| Proposed | Ready for formal review |
| Accepted | Approved and authoritative |
| Implemented | Requirements have corresponding implementation |
| Deprecated | No longer recommended |
| Superseded | Replaced by another document |
| Rejected | Explicitly not accepted |

A document MUST NOT be considered authoritative solely because it exists in the repository.

## 6. Document Identification

Normative documents SHOULD include:

- Document title.
- Document identifier.
- Version.
- Status.
- Purpose.
- Scope.
- Normative requirements.
- Dependencies.
- Review history.

Document identifiers SHOULD remain stable across revisions.

## 7. Requirement Traceability

Normative requirements SHOULD have stable identifiers.

Example:

```text
INT-ARCH-001
INT-SEC-001
INT-API-001
INT-AGENT-001
INT-EVAL-001
```

Requirements SHOULD be traceable through the following lifecycle:

```text
Requirement
    |
    v
Specification
    |
    v
Implementation
    |
    v
Test
    |
    v
Evidence
```

Requirements that affect security, authorization, execution, or data integrity MUST have corresponding validation criteria.

## 8. Documentation Change Governance

Changes to normative documentation MUST:

1. Identify the affected requirements.
2. Explain the motivation for the change.
3. Assess architectural and security implications.
4. Identify dependent specifications.
5. Include appropriate validation criteria.
6. Be reviewed before acceptance.

Changes that alter established architecture SHOULD include an Architecture Decision Record.

## 9. Conflict Resolution

When documentation conflicts:

1. Identify the conflicting requirements.
2. Determine the applicable authority level.
3. Review relevant accepted ADRs.
4. Resolve the conflict through an explicit documentation change.
5. Update affected documents.
6. Record the resolution and rationale.

Conflicts MUST NOT be resolved through undocumented implementation assumptions.

## 10. Implementation Conformance

Implementation MUST conform to applicable accepted specifications.

Where implementation deviates from an accepted requirement:

- The deviation MUST be documented.
- Its impact MUST be assessed.
- The specification MUST be updated or the implementation corrected.
- Security-sensitive deviations MUST NOT bypass required governance.

## 11. Documentation Review

Normative documentation SHOULD be reviewed for:

- Technical correctness.
- Architectural consistency.
- Security implications.
- Requirement completeness.
- Testability.
- Traceability.
- Compatibility with related specifications.

Documents affecting execution authorization or trust boundaries SHOULD receive independent review.

## 12. Deprecation and Supersession

When a document is deprecated or superseded:

- Its status MUST be updated.
- The replacement document SHOULD be identified.
- Relevant references SHOULD be updated.
- Historical records SHOULD be preserved.

## 13. Initial Documentation Baseline

The initial documentation baseline consists of:

1. README.md
2. DOCUMENTATION-AUTHORITY.md
3. PROJECT-VISION.md
4. SCOPE-AND-NON-GOALS.md
5. REFERENCE-ARCHITECTURE.md
6. MASTER-TECHNICAL-SPECIFICATION.md
7. THREAT-MODEL.md
8. Architecture Decision Records

These documents are initially considered proposed documentation until formally reviewed and accepted.

## 14. Acceptance Criteria

This document is considered ready for acceptance when:

- The documentation hierarchy is reviewed.
- Normative language is consistent.
- Document status definitions are agreed upon.
- Requirement traceability is established.
- Documentation change governance is approved.

## 15. Final Statement

Documentation authority is a foundational engineering control for NAEOS Intelligence.

All future architecture, implementation, and governance documentation SHOULD follow the hierarchy and change procedures defined in this document.
```

Related normative documents: `docs/vision/PROJECT-VISION.md` and `docs/vision/PROJECT-SCOPE.md`.
