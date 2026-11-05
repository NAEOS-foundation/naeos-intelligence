# Contributing to NAEOS Intelligence

Thank you for considering a contribution to NAEOS Intelligence.

## How to contribute

1. Review the project documentation (`README.md`, `DOCUMENTATION-AUTHORITY.md`, `docs/`).
2. Understand the architectural boundaries and design principles.
3. Follow the contribution guidelines below.
4. Include appropriate tests with your implementation.
5. Provide evidence for relevant changes.

## Before you start

- Check open issues and pull requests to avoid duplicating work.
- For significant changes, open an issue first to discuss scope and design.
- Do not treat Draft or Proposed documents as accepted requirements.

## Development workflow

1. Fork the repository and create a feature branch from `main`.
2. Implement your change, keeping it narrow and phase-scoped.
3. Add or update tests covering the change.
4. Run `make check` (vet, test, build) and ensure all checks pass.
5. Update documentation and `api/openapi.yaml` if behavior or contracts change.
6. Open a pull request describing the change, impacted files, and verification results.

## Code style

- Idiomatic Go. Run `gofmt` before submitting.
- Explicit interfaces and dependency injection.
- No secrets, tokens, credentials, or unredacted personal data in logs, tests, or fixtures.
- Structured errors and bounded timeouts where applicable.

## Documentation changes

- Identify affected requirements, motivation, and architectural/security implications.
- Include validation criteria.
- Changes that alter architecture should include an Architecture Decision Record (ADR).

## Security

Report security issues according to `SECURITY.md`. Do not disclose vulnerabilities through public issues before assessment.

## License

The project license will be defined before the initial public release.
