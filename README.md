[![CI](https://github.com/greg-hellings/devdashboard/workflows/CI/badge.svg)](https://github.com/greg-hellings/devdashboard/actions/workflows/ci.yml)
[![Code Coverage](https://github.com/greg-hellings/devdashboard/workflows/Code%20Coverage/badge.svg)](https://github.com/greg-hellings/devdashboard/actions/workflows/coverage.yml)
[![Nightly Build](https://github.com/greg-hellings/devdashboard/workflows/Nightly%20Build/badge.svg)](https://github.com/greg-hellings/devdashboard/actions/workflows/nightly.yml)

# DevDashboard

A comprehensive development dashboard system for managing and analyzing repositories across multiple platforms (GitHub, GitLab) with dependency analysis capabilities.

## Features

- **Multi-Provider Repository Access** - Connect to GitHub and GitLab (public, private, self-hosted)
- **Dependency Analysis** - Analyze Python Poetry projects (more analyzers coming)
- **CLI Tool** - Focused `dependency-report` command for cross-repository dependency version comparison
- **Extensible Architecture** - Easy to add new providers and analyzers
- **Library & CLI** - Use as a Go library or standalone tool

## Quick Start

```bash
# Install
cd devdashboard
go build -o devdashboard ./cmd/devdashboard

# Generate dependency report (config-driven)
# Create a config file (repos.yaml) defining repositories, analyzers, and packages
./devdashboard dependency-report repos.yaml
```

## Commands

```bash
devdashboard dependency-report <config-file>   # Generate dependency version report
devdashboard help                              # Show help
```

## Configuration

The CLI now relies solely on a configuration file passed to the `dependency-report` command.
See the docs for the expected YAML structure (providers, repositories, analyzer type, package list).







## Documentation

📚 **[Complete Documentation](docs/INDEX.md)**

### Essential Guides
- **[Quick Start Guide](docs/QUICKSTART.md)** - Get started in 5 minutes
- **[CLI Guide](docs/CLI_GUIDE.md)** - Complete CLI reference
- **[Dependency Analysis](docs/DEPENDENCIES.md)** - Analyze project dependencies

### Developer Resources
- **[Architecture](docs/ARCHITECTURE.md)** - Design and patterns
- **[Adding Analyzers](docs/DEPENDENCY_IMPLEMENTATION.md)** - Extend functionality
- **[Recent Updates](docs/CLI_UPDATE_SUMMARY.md)** - Latest changes

## Library Usage

```go
import (
    "context"
    "github.com/greg-hellings/devdashboard/pkg/repository"
    "github.com/greg-hellings/devdashboard/pkg/dependencies"
)

// Repository operations
repoClient, _ := repository.NewClient("github", repository.Config{})
info, _ := repoClient.GetRepositoryInfo(ctx, "golang", "go")

// Dependency analysis
analyzer, _ := dependencies.NewAnalyzer("poetry")
candidates, _ := analyzer.CandidateFiles(ctx, "owner", "repo", "main", config)
results, _ := analyzer.AnalyzeDependencies(ctx, "owner", "repo", "main", candidates, config)
```

## Supported Platforms

| Platform | Status | Features |
|----------|--------|----------|
| GitHub | ✅ Full | Public, private, Enterprise |
| GitLab | ✅ Full | Public, private, self-hosted |

## Supported Analyzers

| Language | Analyzer | Files | Status |
|----------|----------|-------|--------|
| Python | Poetry | `poetry.lock` | ✅ Supported |
| Node.js | npm | `package-lock.json` | 🔜 Planned |
| Java | Maven | `pom.xml` | 🔜 Planned |
| Rust | Cargo | `Cargo.lock` | 🔜 Planned |

## Installation

### From Source
```bash
git clone https://github.com/greg-hellings/devdashboard.git
cd devdashboard
go build -o devdashboard ./cmd/devdashboard
```

### Using Go Install
```bash
go install github.com/greg-hellings/devdashboard/cmd/devdashboard@latest
```

### Quick Setup
```bash
./setup.sh  # Automated setup and testing
```

## Examples

See [examples/](examples/) directory for library-oriented examples:
- `basic_usage.go` - Repository client usage (library)
- `dependency_analysis.go` - Dependency analysis

## Development

```bash
# Run tests
go test ./...

# Run tests with coverage
make test-coverage

# Run linting
make lint

# Run all checks (fmt, lint, vet, test)
make check

# Build
make build

# Run examples
make example

# Clean
make clean
```

## Project Structure

```
devdashboard/
├── cmd/                    # CLI applications
├── pkg/                    # Library packages
│   ├── repository/        # Repository connectors
│   └── dependencies/      # Dependency analyzers
├── examples/              # Example programs
├── docs/                  # Documentation
└── bin/                   # Build output
```

## Contributing

Contributions welcome! Please:
1. Read the [Architecture Guide](docs/ARCHITECTURE.md)
2. Follow existing patterns
3. Add tests for new features
4. Update documentation

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for version history and changes.

## License

[Specify your license]

## Links

- **Documentation:** [docs/INDEX.md](docs/INDEX.md)
- **Issues:** [GitHub Issues](https://github.com/greg-hellings/devdashboard/issues)
- **Changelog:** [CHANGELOG.md](CHANGELOG.md)

---

**Version:** 0.1.0
**Go Version:** 1.24+
**Status:** Active Development
