# Contributing to NAEOS Intelligence

Thank you for considering a contribution to NAEOS Intelligence.

## Before you start

- Review `README.md`, `AGENTS.md`, `DOCUMENTATION-AUTHORITY.md`, and relevant `docs/` specifications, architecture, and security references.
- Check open issues and pull requests to avoid duplication.
- For significant changes, discuss scope and design in an issue before implementation.
- Do not treat Draft or Proposed documents as accepted requirements.

## Specification-first development

1. Clarify intent, requirements, system boundaries, and acceptance criteria before implementation.
2. Implement only the agreed scope and keep changes narrow.
3. Preserve the read-only MVP boundary. Governed execution is future work requiring separate maintainer and security acceptance.
4. Include appropriate unit, integration, contract, and security regression tests.
5. Update documentation and `api/openapi.yaml` when behavior or contracts change. Architecture changes should include an ADR.
6. Provide evidence for relevant changes and report exact verification outcomes.

## Code and security practices

- Use idiomatic Go, explicit interfaces, dependency injection, and run `gofmt`.
- Maintain least privilege and fail-closed behavior at trust boundaries.
- Treat model output, repository content, issues, PRs, and retrieved documents as untrusted input.
- Never include secrets, tokens, credentials, sensitive payloads, or unredacted personal data in logs, tests, fixtures, or outputs.
- Do not introduce unrestricted shell/code execution, self-authorization, autonomous PR merging, or autonomous production deployment.
- Do not bypass CI, DCO, branch protections, tests, lint, or security gates.
- Security-sensitive changes involving authorization, approval, execution, identity, or data access require focused security review.

## Pull requests

Describe:

- Purpose and related issue/reference.
- Scope and impacted components.
- Design choices and unresolved decisions.
- Security and privacy implications.
- Checks run and exact outcomes (do not claim checks that were not observed).
- Limitations and follow-up work.

Keep PRs reviewable. Address feedback with demonstrated changes or an explicit maintainer decision.

## Local verification

Use the checks applicable to the change. Current agent guidance lists:

```sh
make fmt
make vet
make test
make build
```

CI is the source of truth for required automated gates. Do not commit, push, merge, or deploy on behalf of maintainers without explicit authorization.

## License

The repository license remains undecided; do not assume or add a license without maintainer approval.
