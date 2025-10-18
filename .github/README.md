# GitHub Actions Workflows

This directory contains all GitHub Actions workflows and configuration for the DevDashboard project.

## Overview

The CI/CD pipeline is designed to ensure code quality, security, and reliability through automated checks and builds.

## Workflows

### 🔄 CI (`ci.yml`)

**Trigger:** Push to main/master/develop, Pull Requests

**Jobs:**
- **pre-commit**: Runs pre-commit hooks on all files
- **go-tests**: Tests on Go 1.21, 1.22, and 1.23
- **go-build**: Cross-platform builds (Linux, macOS, Windows)
- **nix-build**: Builds with Nix flake
- **nix-checks**: Runs all Nix flake checks
- **code-quality**: Runs golangci-lint and checks module tidiness
- **security**: Runs Gosec security scanner
- **all-checks**: Final gate requiring all checks to pass

**Secrets Required:**
- `CODECOV_TOKEN` (optional): For code coverage reporting
- `CACHIX_AUTH_TOKEN` (optional): For Nix binary caching

**Matrix Strategy:**
- Go versions: 1.21, 1.22, 1.23
- Platforms: Ubuntu, macOS, Windows

### 📊 Coverage (`coverage.yml`)

**Trigger:** Push to main branches, Pull Requests

**Jobs:**
- **coverage**: Generates coverage report with 70% threshold
- **coverage-diff**: Compares coverage between base and PR (PRs only)

**Features:**
- Uploads to Codecov and Coveralls
- Posts coverage comparison as PR comment
- Enforces minimum 70% coverage threshold
- Generates HTML coverage report

**Secrets Required:**
- `CODECOV_TOKEN` (optional): For Codecov integration
- `GITHUB_TOKEN` (automatic): For PR comments

### 🏷️ Labeler (`labeler.yml`)

**Trigger:** Pull Request open/sync/reopen

**Jobs:**
- **labeler**: Auto-labels based on changed files
- **size-labeler**: Adds size labels (xs/s/m/l/xl)
- **auto-label**: Labels based on PR title conventions

**Label Conventions:**
- `feat:` → enhancement
- `fix:` → bug
- `docs:` → documentation
- `chore:` → chore
- `test:` → tests
- `ci:` → ci
- `perf:` → performance
- `!:` or `breaking` → breaking-change
- `WIP` or `draft:` → work-in-progress

**File-based Labels:**
- Changes to `**/*.go` → go
- Changes to `flake.nix` → nix
- Changes to `docs/**/*` → documentation
- Changes to `.github/**/*` → ci
- Changes to `**/*_test.go` → tests

### 🌙 Nightly (`nightly.yml`)

**Trigger:** Daily at 2 AM UTC, Manual dispatch

**Jobs:**
- **nightly-tests**: Tests with race detector and benchmarks on Go 1.21, 1.22, 1.23, and tip
- **nightly-build**: Integration tests on all platforms
- **nix-nightly**: Comprehensive Nix checks
- **dependency-audit**: Runs govulncheck for vulnerabilities
- **notification**: Creates issue if build fails

**Features:**
- Tests against Go tip (development version)
- Runs tests 3 times with race detector
- Executes benchmarks
- Checks for outdated dependencies
- Creates GitHub issue on failure

### 🚀 Release (`release.yml`)

**Trigger:** Push tags matching `v*.*.*`, Manual dispatch

**Jobs:**
- **create-release**: Creates GitHub release with changelog
- **build-binaries**: Builds for multiple platforms
- **build-nix**: Builds with Nix
- **docker**: Builds and pushes Docker images

**Artifacts:**
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)
- Nix build
- Docker images on ghcr.io

**Secrets Required:**
- `GITHUB_TOKEN` (automatic): For creating releases
- `CACHIX_AUTH_TOKEN` (optional): For Nix caching

**Tags:**
- Semantic versioning: `v1.2.3`
- Pre-release detection: `alpha`, `beta`, `rc`

## Configuration Files

### `dependabot.yml`

Automated dependency updates for:
- **Go modules**: Weekly on Mondays at 9 AM
- **GitHub Actions**: Weekly on Mondays at 9 AM
- **Docker**: Weekly on Mondays at 9 AM

**Settings:**
- Groups minor and patch updates together for Go
- Maximum 10 open PRs for Go dependencies
- Maximum 5 open PRs for Actions and Docker
- Auto-assigns to @greg-hellings
- Adds appropriate labels

### `labeler.yml`

Defines file patterns for automatic labeling:
- **dependencies**: `go.mod`, `go.sum`, `flake.nix`, `flake.lock`
- **go**: `**/*.go`, `go.mod`, `go.sum`
- **nix**: `flake.nix`, `.nix-helpers/**/*`
- **documentation**: `docs/**/*`, `**/*.md`
- **ci**: `.github/**/*`, `Makefile`
- **tests**: `**/*_test.go`
- **config**: `pkg/config/**/*`
- **dependencies** (module): `pkg/dependencies/**/*`
- **repository** (module): `pkg/repository/**/*`
- **report** (module): `pkg/report/**/*`
- **examples**: `examples/**/*`
- **cli**: `cmd/**/*`

### `CODEOWNERS`

Defines code ownership for automatic review requests:
- Default owner: @greg-hellings
- All modules and directories assigned to maintainer
- Triggers review requests on PRs

## Issue Templates

### Bug Report (`bug_report.yml`)

Structured form for reporting bugs with fields:
- Description and expected/actual behavior
- Reproduction steps
- Logs and version information
- Installation method and OS
- Repository provider and analyzer
- Configuration file (sanitized)
- Comprehensive checklist

### Feature Request (`feature_request.yml`)

Structured form for feature requests with fields:
- Problem statement and proposed solution
- Alternatives considered
- Feature category and priority
- Use case and examples
- Contribution willingness
- Validation checklist

## Pull Request Template

Comprehensive PR template including:
- Description and type of change
- Related issues
- Testing performed (unit, integration, manual, Nix)
- Documentation updates
- Code quality checklist
- Go-specific checks
- Nix-specific checks
- Security considerations
- Breaking changes and migration guide
- Performance impact
- Reviewer notes

## Required Secrets

### Optional Secrets
- `CODECOV_TOKEN`: For uploading coverage to Codecov
- `CACHIX_AUTH_TOKEN`: For Nix binary caching

### Automatic Secrets
- `GITHUB_TOKEN`: Automatically provided by GitHub Actions

## Setting Up Secrets

### Codecov Token
1. Visit https://codecov.io
2. Link your GitHub repository
3. Copy the upload token
4. Add as `CODECOV_TOKEN` in repository secrets

### Cachix Token
1. Create account at https://cachix.org
2. Create a cache named `devdashboard`
3. Generate an auth token
4. Add as `CACHIX_AUTH_TOKEN` in repository secrets

## Branch Protection

Recommended branch protection rules for `main`:

- ✅ Require status checks to pass before merging
  - `All checks passed`
  - `go-tests (1.21)`
  - `nix-build`
  - `nix-checks`
  - `pre-commit`
  - `code-quality`
  - `security`
- ✅ Require branches to be up to date before merging
- ✅ Require conversation resolution before merging
- ✅ Require signed commits (recommended)
- ✅ Require linear history (recommended)
- ✅ Require pull request reviews (1 approval)
- ✅ Dismiss stale reviews when new commits are pushed
- ✅ Require review from Code Owners

## Workflow Badges

Add these to your README.md:

```markdown
[![CI](https://github.com/greg-hellings/devdashboard/workflows/CI/badge.svg)](https://github.com/greg-hellings/devdashboard/actions/workflows/ci.yml)
[![Code Coverage](https://github.com/greg-hellings/devdashboard/workflows/Code%20Coverage/badge.svg)](https://github.com/greg-hellings/devdashboard/actions/workflows/coverage.yml)
[![codecov](https://codecov.io/gh/greg-hellings/devdashboard/branch/main/graph/badge.svg)](https://codecov.io/gh/greg-hellings/devdashboard)
[![Nightly Build](https://github.com/greg-hellings/devdashboard/workflows/Nightly%20Build/badge.svg)](https://github.com/greg-hellings/devdashboard/actions/workflows/nightly.yml)
```

## Troubleshooting

### CI Failures

**Pre-commit fails:**
```bash
# Run locally (requires Nix)
nix develop --command pre-commit run --all-files
```

**Go tests fail:**
```bash
# Run with verbose output
go test -v ./...
```

**Nix build fails:**
```bash
# Build locally
nix build -L

# Check flake
nix flake check -L
```

**Coverage below threshold:**
```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
# Open coverage.html in browser
```

### Dependabot Issues

**Too many PRs:**
- Adjust `open-pull-requests-limit` in `dependabot.yml`
- Review and merge dependency updates more frequently

**Failed dependency updates:**
- Check if `go.mod` has replace directives
- Verify vendorHash is updated for Nix builds
- Review breaking changes in dependency changelogs

### Release Issues

**Release build fails:**
- Ensure all tests pass on main branch
- Verify version tag matches semantic versioning
- Check CHANGELOG.md has entry for version

**Docker build fails:**
- Test Dockerfile locally: `docker build -t devdashboard .`
- Verify all source files are committed to git
- Check .dockerignore doesn't exclude necessary files

## Performance Optimization

### Caching

The workflows use multiple caching strategies:
- Go module cache (actions/setup-go)
- Pre-commit hooks cache
- Docker layer cache
- Nix binary cache (Cachix)

### Concurrency

Workflows use concurrency groups to cancel outdated runs:
```yaml
concurrency:
  group: ${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true
```

### Matrix Builds

Jobs run in parallel using matrix strategies:
- Multiple Go versions
- Multiple operating systems
- Fail-fast disabled for comprehensive testing

## Best Practices

1. **Always run checks locally before pushing**
   ```bash
   make check
   nix flake check
   ```

2. **Keep dependencies up to date**
   - Review Dependabot PRs weekly
   - Update GitHub Actions monthly

3. **Monitor workflow runs**
   - Check Actions tab regularly
   - Address failures promptly

4. **Use conventional commits**
   - Enables automatic labeling
   - Improves changelog generation

5. **Write tests for new features**
   - Maintain or improve coverage
   - Add integration tests when appropriate

6. **Document breaking changes**
   - Update CHANGELOG.md
   - Provide migration guide
   - Mark PRs appropriately

## Contributing

When adding new workflows:
1. Test locally with [act](https://github.com/nektos/act)
2. Use reusable workflows when possible
3. Add appropriate secrets documentation
4. Update this README with new workflow details
5. Test on fork before merging to main

## Resources

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Go Actions Setup](https://github.com/actions/setup-go)
- [Nix Install Action](https://github.com/cachix/install-nix-action)
- [Codecov Action](https://github.com/codecov/codecov-action)
- [golangci-lint Action](https://github.com/golangci/golangci-lint-action)
