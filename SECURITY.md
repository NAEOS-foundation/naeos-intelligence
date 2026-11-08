# Security Policy

## Supported versions

NAEOS Intelligence is in pre-release (MVP foundation) status. No versions are currently designated for security support.

## Reporting a vulnerability privately

**Do not disclose suspected vulnerabilities in public issues, pull requests, discussions, or social media before coordinated assessment.**

Use GitHub's **Private vulnerability reporting** feature for this repository if enabled (repository **Security** tab → **Report a vulnerability**). If this feature is unavailable, contact repository maintainers to establish an appropriate private reporting channel. Do not post exploit details publicly while awaiting a private channel.

A report should include, when safe:

- A concise description and affected component/path/version or commit.
- Impact and plausible attack conditions.
- Reproduction steps or a minimal proof of concept.
- Relevant artifacts, with secrets and personal data redacted.
- Any known mitigation.

Never attach real credentials, production data, or sensitive user information.

## Triage and response expectations

The following response targets are **Proposed and require maintainer acceptance**; they are not a binding service-level commitment:

- Acknowledge receipt within 5 business days.
- Provide an initial triage assessment within 10 business days where feasible.
- Coordinate remediation, disclosure timing, and reporter credit with the reporter and maintainers.

For urgent or actively exploited issues, state this clearly in the private report.

## Disclosure process

1. Maintainers acknowledge receipt and restrict report access to people who need it for triage/remediation.
2. The report is triaged and validated.
3. A fix is developed, reviewed, and tested through required gates.
4. Disclosure timing and any advisory are coordinated with the reporter and maintainers.
5. Avoid publishing exploit details before remediation or an explicit coordinated disclosure decision.

No specific disclosure date or bounty program is implied.

## Security boundaries

- The MVP is read-only; governed execution is future capability requiring separate acceptance.
- Authentication failures fail closed.
- Model outputs and agent plans are proposals, never authorization.
- Tool access must be explicitly scoped; external side effects must be authorized.
- Audit records preserve action traceability.
- Unauthorized execution must fail closed.

## Out of scope

- Autonomous production deployment.
- Unrestricted production data modification.
- Arbitrary shell or code execution.

## Policy status

Private reporting instructions depend on GitHub Private vulnerability reporting being enabled or maintainers establishing a private channel. Response targets above are proposed. Maintainers should confirm the reporting channel, response targets, and disclosure process before treating them as accepted policy.
