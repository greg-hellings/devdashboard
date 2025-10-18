# Dependency Report

The `dependency-report` command generates a comprehensive report showing dependency versions across multiple repositories. This is useful for tracking dependency consistency, identifying version drift, and planning upgrades across a microservices architecture or multi-repository project.

## Overview

The dependency report feature allows you to:
- Track specific package versions across multiple repositories
- Compare dependency versions between different projects
- Identify repositories using outdated or inconsistent dependencies
- Support multiple repository providers (GitHub, GitLab)
- Analyze different dependency file formats (poetry.lock, Pipfile.lock, uv.lock)

## Quick Start

### 1. Create a Configuration File

Create a YAML configuration file (e.g., `dependency-report.yaml`):

```yaml
providers:
  github:
    default:
      owner: "myorg"
      analyzer: "poetry"
      packages:
        - "requests"
        - "pytest"
        - "django"
    repositories:
      - repository: "api-service"
      - repository: "worker-service"
      - repository: "frontend-service"
```

### 2. Run the Report

```bash
devdashboard dependency-report dependency-report.yaml
```

### 3. View the Output

```
Dependency Version Report
================================================================================

Package                        | api-service          | worker-service       | frontend-service
------------------------------------------------------------------------------------------------
requests                       | 2.28.1               | 2.28.1               | 2.31.0
pytest                         | 7.2.0                | 7.2.0                | N/A
django                         | 4.1.0                | 4.2.0                | 4.2.0

Summary:
  Repositories analyzed: 3/3 successful
  Packages tracked: 3
```

## Configuration File Format

### Top-Level Structure

```yaml
providers:
  <provider-name>:
    default:
      # Default configuration
    repositories:
      # List of repositories
```

### Provider Configuration

Each provider (e.g., `github`, `gitlab`) contains:

- `default` - Default values inherited by all repositories
- `repositories` - List of repositories to analyze

### Default Section

The `default` section defines values that apply to all repositories unless explicitly overridden:

```yaml
default:
  token: "your-token-here"          # Authentication token
  owner: "default-owner"            # Repository owner/organization
  repository: "default-repo"        # Default repository name (rarely used)
  ref: "main"                       # Git reference (branch/tag/commit)
  analyzer: "poetry"                # Dependency analyzer type
  paths: []                         # Paths to search for lock files
  packages:                         # Packages to track
    - "package1"
    - "package2"
```

### Repository Configuration

Each repository can override any default value:

```yaml
repositories:
  # Minimal - inherits all defaults
  - repository: "repo1"

  # Override specific values
  - repository: "repo2"
    ref: "develop"
    analyzer: "pipfile"

  # Complete custom configuration
  - repository: "repo3"
    owner: "different-owner"
    ref: "v2.0"
    analyzer: "uvlock"
    paths:
      - "backend"
      - "services"
    packages:
      - "custom-package"
```

## Configuration Fields

### Required Fields

| Field | Description | Example |
|-------|-------------|---------|
| `owner` | Repository owner or organization | `"myorg"` |
| `repository` | Repository name | `"my-service"` |
| `analyzer` | Dependency analyzer type | `"poetry"`, `"pipfile"`, `"uvlock"` |

### Optional Fields

| Field | Description | Default | Example |
|-------|-------------|---------|---------|
| `token` | Authentication token | `""` | `"ghp_xxxx"` |
| `ref` | Git reference | `""` (default branch) | `"main"`, `"v1.0"`, `"abc123"` |
| `paths` | Paths to search | `[]` (entire repo) | `["src", "services"]` |
| `packages` | Packages to track | `[]` | `["requests", "django"]` |

## Analyzer Types

The `analyzer` field determines which dependency file format to parse:

| Analyzer | File Format | Description |
|----------|-------------|-------------|
| `poetry` | poetry.lock | Python Poetry projects |
| `pipfile` | Pipfile.lock | Python Pipenv projects |
| `uvlock` | uv.lock | Python uv projects |

## Examples

### Basic Single Provider

```yaml
providers:
  github:
    default:
      owner: "mycompany"
      analyzer: "poetry"
      packages: ["requests", "flask"]
    repositories:
      - repository: "api"
      - repository: "worker"
```

### Multiple Providers

```yaml
providers:
  github:
    default:
      owner: "mycompany"
      analyzer: "poetry"
      packages: ["requests"]
    repositories:
      - repository: "public-api"

  gitlab:
    default:
      owner: "internal"
      analyzer: "pipfile"
      packages: ["django"]
    repositories:
      - repository: "admin-panel"
```

### Repository-Specific Packages

Track different packages in different repositories:

```yaml
providers:
  github:
    default:
      owner: "myorg"
      analyzer: "poetry"
    repositories:
      - repository: "frontend"
        packages: ["fastapi", "pydantic"]
      - repository: "backend"
        packages: ["django", "celery"]
      - repository: "data-pipeline"
        packages: ["pandas", "numpy"]
```

### Different Analyzers Per Repository

```yaml
providers:
  github:
    default:
      owner: "myorg"
      packages: ["requests", "pytest"]
    repositories:
      - repository: "poetry-project"
        analyzer: "poetry"
      - repository: "pipenv-project"
        analyzer: "pipfile"
      - repository: "uv-project"
        analyzer: "uvlock"
```

### Monorepo with Multiple Lock Files

```yaml
providers:
  github:
    default:
      owner: "myorg"
      analyzer: "poetry"
    repositories:
      - repository: "monorepo"
        paths:
          - "services/api"
          - "services/worker"
          - "packages/common"
        packages: ["requests", "pytest"]
```

### Using Environment Variables

```yaml
providers:
  github:
    default:
      token: "${GITHUB_TOKEN}"
      owner: "${GITHUB_ORG}"
      analyzer: "poetry"
      packages: ["requests"]
    repositories:
      - repository: "my-service"
```

Then run:
```bash
export GITHUB_TOKEN="your-token"
export GITHUB_ORG="your-org"
devdashboard dependency-report config.yaml
```

## Output Format

### Table View

The default output is a table showing package versions across repositories:

```
Package                        | repo1                | repo2                | repo3
------------------------------------------------------------------------------------------------
requests                       | 2.28.1               | 2.28.1               | 2.31.0
pytest                         | 7.2.0                | N/A                  | 7.2.0
django                         | 4.1.0                | 4.2.0                | ERROR
```

### Column Meanings

- **Package** - Name of the tracked package
- **Repository columns** - Version found in each repository
  - `2.28.1` - Package version found
  - `N/A` - Package not found in lock file
  - `ERROR` - Error analyzing repository (see errors section)

### Summary Section

```
Summary:
  Repositories analyzed: 3/4 successful
  Packages tracked: 3
```

### Errors Section

If any repositories fail to analyze, errors are shown at the bottom:

```
Errors encountered:
================================================================================
  myorg/broken-repo: no dependency files found
  myorg/private-repo: failed to create repository client: authentication required
```

## Verbosity Levels

Control log output with verbosity flags:

### Default (Quiet)
```bash
devdashboard dependency-report config.yaml
```
Shows only warnings and errors.

### Verbose
```bash
devdashboard -v dependency-report config.yaml
```
Shows progress information (INFO level).

### Debug
```bash
devdashboard -vv dependency-report config.yaml
```
Shows detailed debug information including:
- Repository analysis progress
- Dependency file discovery
- Parse errors for individual files
- Package matching details

## Use Cases

### 1. Microservices Dependency Audit

Track critical dependencies across all services:

```yaml
providers:
  github:
    default:
      owner: "mycompany"
      analyzer: "poetry"
      packages:
        - "requests"
        - "pydantic"
        - "fastapi"
        - "sqlalchemy"
        - "redis"
    repositories:
      - repository: "auth-service"
      - repository: "payment-service"
      - repository: "notification-service"
      - repository: "analytics-service"
```

### 2. Security Vulnerability Tracking

Identify repositories using vulnerable package versions:

```yaml
providers:
  github:
    default:
      owner: "myorg"
      analyzer: "poetry"
      # Track packages with known vulnerabilities
      packages:
        - "requests"      # CVE-2023-xxxxx
        - "pillow"        # CVE-2023-xxxxx
        - "cryptography"  # CVE-2023-xxxxx
    repositories:
      - repository: "prod-api"
      - repository: "prod-worker"
      # ... all production services
```

### 3. Migration Planning

Before upgrading a dependency, see which repositories need updates:

```yaml
providers:
  github:
    default:
      owner: "myorg"
      analyzer: "poetry"
      packages:
        - "django"  # Planning to upgrade to 5.0
    repositories:
      - repository: "legacy-admin"
      - repository: "api-v1"
      - repository: "api-v2"
      - repository: "new-frontend"
```

### 4. Consistency Enforcement

Ensure all repositories use approved dependency versions:

```yaml
providers:
  github:
    default:
      owner: "myorg"
      analyzer: "poetry"
      # Approved/required versions
      packages:
        - "pydantic"
        - "fastapi"
        - "sqlalchemy"
    repositories:
      - repository: "service-a"
      - repository: "service-b"
      - repository: "service-c"
```

## Troubleshooting

### No dependency files found

**Problem:** Repository has no lock files or they're not in the expected location.

**Solution:**
1. Verify the repository has the correct lock file (poetry.lock, Pipfile.lock, or uv.lock)
2. Use `paths` to specify where lock files are located:
   ```yaml
   - repository: "monorepo"
     paths: ["backend", "services"]
   ```

### Authentication required

**Problem:** Token missing or invalid for private repositories.

**Solution:**
1. Set the token in the config file:
   ```yaml
   default:
     token: "your-token-here"
   ```
2. Or use environment variable:
   ```yaml
   default:
     token: "${GITHUB_TOKEN}"
   ```

### Package not found (N/A)

**Problem:** Package is tracked but not present in repository's lock file.

**This is not an error** - it means:
- The repository doesn't use this package
- The package has a different name in this repository
- The package is indirect and not in the lock file

### Wrong analyzer type

**Problem:** Using `poetry` analyzer on a Pipfile.lock.

**Solution:**
Match analyzer to file format:
```yaml
- repository: "pipenv-project"
  analyzer: "pipfile"  # Use pipfile for Pipfile.lock
```

### Rate limiting

**Problem:** GitHub API rate limit exceeded.

**Solution:**
1. Add authentication token to increase rate limit
2. Reduce number of repositories in config
3. Run with delays between repositories

## Performance Considerations

### Parallel Analysis

Repositories are analyzed in parallel for better performance. The report command uses goroutines to process multiple repositories simultaneously.

### Timeout

Default timeout is 5 minutes. For large numbers of repositories, you may need to:
1. Split into multiple config files
2. Increase timeout (requires code modification)
3. Run reports separately by provider

### Caching

Currently, no caching is implemented. Each run fetches fresh data from repository providers.

## Integration with CI/CD

### GitHub Actions Example

```yaml
name: Dependency Report
on:
  schedule:
    - cron: '0 0 * * 1'  # Weekly on Monday
  workflow_dispatch:

jobs:
  report:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: Install DevDashboard
        run: go install github.com/greg-hellings/devdashboard/cmd/devdashboard@latest
      - name: Generate Report
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: devdashboard dependency-report config.yaml > report.txt
      - name: Upload Report
        uses: actions/upload-artifact@v3
        with:
          name: dependency-report
          path: report.txt
```

### GitLab CI Example

```yaml
dependency-report:
  image: golang:1.21
  script:
    - go install github.com/greg-hellings/devdashboard/cmd/devdashboard@latest
    - devdashboard dependency-report config.yaml > report.txt
  artifacts:
    paths:
      - report.txt
    expire_in: 30 days
  only:
    - schedules
```

## Related Documentation

- [CLI Guide](CLI_GUIDE.md) - General CLI usage
- [Dependency Analysis](DEPENDENCIES.md) - Analyzer details
- [Configuration Reference](../examples/dependency-report-config.yaml) - Full config example
- [Logging](LOGGING.md) - Debug output

## API Reference

For programmatic usage, see:
- `pkg/config` - Configuration loading and validation
- `pkg/report` - Report generation logic