# Security Policy

## Supported versions

NAEOS Intelligence is in pre-release (MVP foundation) status. No versions are currently designated for security support.

## Reporting a vulnerability

Security issues should be reported privately to the maintainers. Do not disclose vulnerabilities through public issues or pull requests before they have been appropriately assessed.

## Security boundaries

- The MVP is read-only; governed execution is a future capability requiring separate acceptance.
- Authentication failures fail closed (503 when the verifier is not configured).
- Model outputs and agent plans are proposals, never authorization.
- Tool access must be explicitly scoped; external side effects must be authorized.
- Audit records preserve action traceability.
- Unauthorized execution must fail closed.

## What is out of scope

- Autonomous production deployment.
- Unrestricted production data modification.
- Arbitrary shell or code execution.

## Disclosure process

1. Maintainers acknowledge receipt.
2. The issue is triaged and validated.
3. A fix is developed and reviewed.
4. The disclosure is coordinated with the reporter before publication.
