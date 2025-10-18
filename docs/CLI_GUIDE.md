# DevDashboard CLI Guide (Simplified)

This guide documents the current DevDashboard command-line interface.
The CLI now focuses exclusively on generating cross-repository dependency version reports via a single command: `dependency-report`.

---

## Overview

The `dependency-report` command:
- Reads a YAML configuration file describing providers, repositories, analyzers, and package names you want to track.
- Retrieves dependency information from each repository using the configured analyzer (currently Poetry for Python).
- Produces either:
  - An adaptive console table (default), or
  - Structured JSON suitable for automation.

Legacy commands (`repo-info`, `list-files`, `find-dependencies`, `analyze-dependencies`) have been removed to streamline functionality.

---

## Installation

```bash
git clone https://github.com/greg-hellings/devdashboard.git
cd devdashboard
go build -o devdashboard ./cmd/devdashboard
# Optional global install:
# go install ./cmd/devdashboard
```

Check version:
```bash
./devdashboard version
```

---

## Quick Start

1. Create a configuration file (e.g. `repos.yaml`):

```yaml
providers:
  - name: github
    token: ""     # or set a PAT for private repos

repositories:
  - provider: github
    owner: python-poetry
    repository: poetry
    analyzer: poetry
    packages:
      - poetry
      - requests
      - virtualenv
```

2. Run the report:

```bash
./devdashboard dependency-report repos.yaml
```

3. JSON output:

```bash
./devdashboard dependency-report repos.yaml --format json --json-indent
```

---

## Configuration File Structure

Top-level keys:
- `providers`: List of provider definitions.
  - `name`: Provider identifier (`github`, `gitlab`).
  - `token`: (Optional) Personal Access Token for private repositories.
- `repositories`: List of repositories to analyze.
  - `provider`: Matches a provider name.
  - `owner`: Account/org/group.
  - `repository`: Repository name.
  - `ref`: (Optional) Branch/tag/commit (default: provider default branch).
  - `analyzer`: Dependency analyzer (currently `poetry`).
  - `paths`: (Optional) Explicit dependency file paths — skips auto-discovery.
  - `packages`: List of package names to track across all repos.

Example with multiple providers:

```yaml
providers:
  - name: github
    token: ghp_xxx
  - name: gitlab
    token: glpat_xxx

repositories:
  - provider: github
    owner: org1
    repository: service-a
    analyzer: poetry
    packages: [requests, fastapi]

  - provider: gitlab
    owner: backend-team
    repository: billing-api
    analyzer: poetry
    packages: [requests, fastapi]
```

---

## Command Reference

### `dependency-report`

Generate a dependency version comparison across all configured repositories.

Usage:
```bash
devdashboard dependency-report <config-file> [flags]
```

Required:
- `<config-file>`: Path to the YAML configuration.

#### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-f`, `--format` | string | `console` | Output format: `console` or `json` |
| `-o`, `--out` | string | (stdout) | Write output to file |
| `--no-color` | bool | false | Disable ANSI colors (console) |
| `--package-col-width` | int | 0 | Max width of package column (0 = auto) |
| `--repo-col-width` | int | 0 | Max width per repo/version column (0 = auto) |
| `--timeout` | duration | 5m | Total reporting timeout |
| `--fail-on-error` | bool | false | Exit non-zero if any repository fails |
| `--json-indent` | bool | false | Pretty-print JSON |
| `--json-include-errors` | bool | true | Include error map in JSON |
| `-v`, `--verbose` | bool | false | Info-level logging |
| `--debug` | bool | false | Debug-level logging |
| `--version` | (root) |  | Show version |

---

## Console Output Format

- Dynamically sized table fitting terminal width.
- Truncates long values with ellipsis.
- Marks failed repositories with `ERROR`.
- Missing package in a repo shown with a dash (—).
- Summary and error details printed below the table.

Example:

```
Dependency Version Report (format=console)

┌───────────┬──────────────┬───────────────┐
│ Package   │ org1/service │ org2/service2 │
├───────────┼──────────────┼───────────────┤
│ requests  │ 2.32.3       │ 2.31.0        │
│ fastapi   │ —            │ 0.110.0       │
│ poetry    │ ERROR        │ 1.8.3         │
└───────────┴──────────────┴───────────────┘

Summary:
  Repositories analyzed: 1/2 successful
  Packages tracked: 3

Errors:
  org1/service1                  failed to create analyzer: unsupported analyzer type "..."
```

---

## JSON Output Format

Example command:
```bash
devdashboard dependency-report repos.yaml --format json --json-indent > report.json
```

Example structure:
```json
{
  "cliVersion": "dev",
  "generatedAt": "2025-01-30T14:12:05Z",
  "repositories": [
    {
      "Provider": "github",
      "Owner": "org1",
      "Repository": "service-a",
      "Ref": "",
      "Analyzer": "poetry",
      "Dependencies": { "requests": "2.32.3" },
      "Error": null
    }
  ],
  "packages": ["requests", "fastapi"],
  "summary": {
    "repositoryCount": 2,
    "packageCount": 2,
    "successCount": 1,
    "errorCount": 1
  },
  "errors": {
    "org2/service-b": "failed to analyze dependencies: no dependency files found"
  }
}
```

Notes:
- `Error` inside each repository element is `null` or omitted (marshaled from the internal error field).
- The `errors` map is omitted if there are no errors or `--json-include-errors=false`.

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error / invalid config / internal failure |
| 2 | (Reserved) Future: validation errors |
| 3 | One or more repos failed AND `--fail-on-error` was set |

Currently, non-zero codes all default to 1 except the explicit `fail-on-error` failure path.

---

## Examples

Console (default):
```bash
devdashboard dependency-report repos.yaml
```

JSON (pretty):
```bash
devdashboard dependency-report repos.yaml --format json --json-indent
```

Write output to file:
```bash
devdashboard dependency-report repos.yaml --format json -o out/report.json
```

Fail build if any repository fails:
```bash
devdashboard dependency-report repos.yaml --fail-on-error
```

Custom widths (wide package names):
```bash
devdashboard dependency-report repos.yaml --package-col-width 40 --repo-col-width 18
```

Disable colors (CI logs):
```bash
devdashboard dependency-report repos.yaml --no-color
```

---

## Best Practices

1. Keep repository list small per invocation when using long timeouts.
2. Use `--fail-on-error` in CI pipelines to enforce complete success.
3. Archive JSON reports for historical diffs (e.g., in an artifacts store).
4. Separate configuration files per team or domain to minimize noise.
5. Provide tokens only for providers that need private repo access.

---

## Migration Notes (Legacy Removal)

Removed legacy commands:
- `repo-info`
- `list-files`
- `find-dependencies`
- `analyze-dependencies`

If you previously scripted these, replace workflows with:
- Use a config file and `dependency-report`.
- Extend configuration or add analyzers (future enhancements) instead of imperative commands.

---

## Troubleshooting

| Symptom | Possible Cause | Resolution |
|---------|----------------|-----------|
| `unsupported analyzer type` | Typo or unsupported analyzer | Use `poetry` (current support) |
| All repos show `ERROR` | Invalid tokens / network | Verify provider token/scopes |
| Table too narrow | Small terminal width | Pipe to file or widen terminal; use JSON |
| JSON missing errors map | No errors or `--json-include-errors=false` | Remove the flag or re-run without it |
| Exit code 1 with valid config | Internal failure / timeout | Increase `--timeout` or run with `--debug` |

Enable debug logs:
```bash
devdashboard --debug dependency-report repos.yaml
```

---

## Roadmap (Planned Enhancements)

- Additional analyzers (npm, cargo, maven)
- Optional caching of repository content
- HTML or Markdown report exporters
- Policy evaluation (e.g., version drift detection)

---

## Getting Help

- Run `devdashboard dependency-report --help`
- Inspect errors with `--debug`
- Review repository documentation (README + docs/)
- Open issues on GitHub if you encounter reproducible problems

---

**Happy reporting!**
