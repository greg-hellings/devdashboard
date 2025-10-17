# Changelog

All notable changes to the DevDashboard project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Planned
- Web dashboard interface
- GUI desktop application
- Bitbucket support
- Response caching layer
- Rate limit handling and retry logic
- File content retrieval
- Repository comparison tools
- Webhook support for real-time updates
- Advanced code analysis and metrics

## [0.1.0] - 2024-01-XX

### Added
- Initial project structure and Go module setup
- Core repository client interface (`Client`)
- GitHub repository connector using official `go-github` library
  - Support for public and private repositories
  - List files in directories (recursive and non-recursive)
  - Retrieve repository metadata
  - GitHub Enterprise support via custom base URL
  - OAuth2 authentication
- GitLab repository connector using official `go-gitlab` library
  - Support for public and private repositories
  - List files in directories with pagination support
  - Retrieve repository metadata
  - Self-hosted GitLab instance support
  - Personal access token authentication
- Factory pattern for creating repository clients
  - Case-insensitive provider selection
  - Support for "github" and "gitlab" providers
  - Convenience function for one-off client creation
- CLI tool (`cmd/devdashboard`)
  - `repo-info` command for repository metadata
  - `list-files` command for recursive file listing
  - Environment variable-based configuration
  - Help command and usage documentation
- Common data structures
  - `FileInfo`: Unified file metadata across providers
  - `RepositoryInfo`: Unified repository metadata
  - `Config`: Authentication and endpoint configuration
- Comprehensive documentation
  - README.md with full API reference
  - QUICKSTART.md for new users
  - ARCHITECTURE.md explaining design decisions
  - Inline code comments explaining implementation details
- Example programs demonstrating library usage
  - Basic usage examples for both GitHub and GitLab
  - Factory pattern demonstration
  - Private repository access examples
  - Directory listing examples
- Build automation
  - Makefile with common targets (build, test, clean, etc.)
  - Setup script for quick installation
  - Go module dependency management
- Testing
  - Unit tests for factory pattern
  - Unit tests for helper functions
  - Test coverage for client creation and configuration
- Development tools
  - .gitignore for Go projects
  - Makefile for build automation
  - Shell script for automated setup

### Technical Details
- Go 1.21 compatibility
- Uses official provider libraries for better maintainability
- Context support for cancellation and timeouts
- Interface-driven design for extensibility
- Zero external CLI dependencies (pure Go)

### Dependencies
- `github.com/google/go-github/v57` - Official GitHub API client
- `github.com/xanzy/go-gitlab` - Official GitLab API client
- `golang.org/x/oauth2` - OAuth2 authentication support

## Version History

### Version Numbering
- **Major version (X.0.0)**: Incompatible API changes
- **Minor version (0.X.0)**: New functionality, backward compatible
- **Patch version (0.0.X)**: Bug fixes, backward compatible

### Support Policy
- Latest version receives active development
- Previous minor version receives security fixes
- Older versions are unsupported

## How to Upgrade

### From Nothing (New Installation)
Follow the instructions in QUICKSTART.md or run:
```bash
./setup.sh
```

### Future Upgrades
When upgrading between versions:
1. Read the changelog for breaking changes
2. Update your `go.mod`: `go get -u github.com/gregoryhellings/devdashboard`
3. Run tests: `make test`
4. Check for deprecated features in your code

## Contributing

See the main README.md for contribution guidelines. When contributing:
- Add your changes to the "Unreleased" section
- Follow the format: `### Added/Changed/Deprecated/Removed/Fixed/Security`
- Include issue/PR numbers where applicable
- Update version number when releasing

## Links

- [Project Repository](https://github.com/gregoryhellings/devdashboard)
- [Issue Tracker](https://github.com/gregoryhellings/devdashboard/issues)
- [Documentation](README.md)