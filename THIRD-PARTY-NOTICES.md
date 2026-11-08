# Third-Party Notices

This project uses the Go modules listed below, as declared in `go.mod` at the time this notice was prepared. License identifiers and source links correspond to the pinned module versions. The Go module graph can change; regenerate and review this notice when dependencies are added, removed, or upgraded.

## Direct and indirect Go modules

| Module | Version | License | License text / source |
|---|---:|---|---|
| github.com/getkin/kin-openapi | v0.152.0 | MIT | https://raw.githubusercontent.com/getkin/kin-openapi/v0.152.0/LICENSE |
| github.com/jackc/pgx/v5 | v5.9.2 | MIT | https://raw.githubusercontent.com/jackc/pgx/v5.9.2/LICENSE |
| github.com/pressly/goose/v3 | v3.24.1 | MIT | https://raw.githubusercontent.com/pressly/goose/v3.24.1/LICENSE |
| gopkg.in/yaml.v3 | v3.0.1 | MIT AND Apache-2.0 (file-level dual licensing) | https://raw.githubusercontent.com/go-yaml/yaml/v3.0.1/LICENSE |
| github.com/go-openapi/jsonpointer | v0.22.5 | Apache-2.0 | https://raw.githubusercontent.com/go-openapi/jsonpointer/v0.22.5/LICENSE |
| github.com/go-openapi/swag/jsonname | v0.25.5 | Apache-2.0 | https://raw.githubusercontent.com/go-openapi/swag/v0.25.5/LICENSE |
| github.com/jackc/pgpassfile | v1.0.0 | MIT | https://raw.githubusercontent.com/jackc/pgpassfile/v1.0.0/LICENSE |
| github.com/jackc/pgservicefile | 5a60cdf6a761 (pseudo-version in go.mod) | MIT | https://raw.githubusercontent.com/jackc/pgservicefile/5a60cdf6a761/LICENSE |
| github.com/jackc/puddle/v2 | v2.2.3 | MIT | https://raw.githubusercontent.com/jackc/puddle/v2.2.3/LICENSE |
| github.com/mfridman/interpolate | v0.0.2 | MIT | https://github.com/mfridman/interpolate/blob/v0.0.2/LICENSE.txt |
| github.com/oasdiff/yaml | v0.1.1 | MIT | https://raw.githubusercontent.com/oasdiff/yaml/v0.1.1/LICENSE |
| github.com/oasdiff/yaml3 | v0.0.14 | MIT AND Apache-2.0 (file-level dual licensing) | https://raw.githubusercontent.com/oasdiff/yaml3/v0.0.14/LICENSE |
| github.com/rogpeppe/go-internal | v1.16.0 | BSD-3-Clause | https://raw.githubusercontent.com/rogpeppe/go-internal/v1.16.0/LICENSE |
| github.com/santhosh-tekuri/jsonschema/v6 | v6.0.3 | Apache-2.0 | https://raw.githubusercontent.com/santhosh-tekuri/jsonschema/v6.0.3/LICENSE |
| github.com/sethvargo/go-retry | v0.3.0 | Apache-2.0 | https://raw.githubusercontent.com/sethvargo/go-retry/v0.3.0/LICENSE |
| go.uber.org/multierr | v1.11.0 | MIT | https://raw.githubusercontent.com/uber-go/multierr/v1.11.0/LICENSE.txt |
| golang.org/x/sync | v0.22.0 | BSD-3-Clause | https://raw.githubusercontent.com/golang/sync/v0.22.0/LICENSE |
| golang.org/x/text | v0.41.0 | BSD-3-Clause | https://raw.githubusercontent.com/golang/text/v0.41.0/LICENSE |

## Scope and maintenance

- This inventory covers the modules explicitly present in the project's `go.mod` (direct and indirect entries), not every module checksum in `go.sum`.
- `go.sum` can include checksums for module versions used by tests, tooling, or historical graph resolution. Those entries are not automatically treated as shipped runtime dependencies here.
- The upstream license files linked above contain the applicable copyright notices, license terms, and disclaimers. Preserve those notices when redistributing the corresponding third-party software.
- Review this file against the resolved module graph and source distribution before each release.
