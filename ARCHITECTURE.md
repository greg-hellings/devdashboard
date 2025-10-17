# DevDashboard Architecture

This document describes the architectural decisions, design patterns, and structure of the DevDashboard project.

## Overview

DevDashboard is designed as a modular, extensible system for managing and monitoring development repositories across multiple platforms. The architecture emphasizes clean separation of concerns, testability, and ease of extension.

## Project Structure

```
devdashboard/
├── cmd/                        # Application entry points
│   └── devdashboard/          # CLI application
│       └── main.go            # CLI implementation
├── pkg/                        # Reusable library packages
│   └── repository/            # Repository connector package
│       ├── repository.go      # Core interfaces and types
│       ├── github.go          # GitHub implementation
│       ├── gitlab.go          # GitLab implementation
│       ├── factory.go         # Factory pattern implementation
│       └── factory_test.go    # Unit tests
├── examples/                   # Example code and usage
│   └── basic_usage.go         # Demonstration program
├── bin/                        # Build output directory (gitignored)
├── go.mod                      # Go module definition
├── go.sum                      # Dependency checksums
├── Makefile                    # Build automation
├── setup.sh                    # Setup script
├── README.md                   # User documentation
├── QUICKSTART.md              # Getting started guide
└── ARCHITECTURE.md            # This file
```

## Design Principles

### 1. Interface-Driven Design

The core of the architecture is the `Client` interface defined in `pkg/repository/repository.go`:

```go
type Client interface {
    ListFiles(ctx context.Context, owner, repo, ref, path string) ([]FileInfo, error)
    GetRepositoryInfo(ctx context.Context, owner, repo string) (*RepositoryInfo, error)
    ListFilesRecursive(ctx context.Context, owner, repo, ref string) ([]FileInfo, error)
}
```

**Benefits:**
- Provider implementations are interchangeable
- Easy to mock for testing
- Clear contract between components
- New providers can be added without changing existing code

### 2. Factory Pattern

The Factory pattern (`factory.go`) provides a unified way to instantiate repository clients:

```go
client, err := repository.NewClient("github", config)
```

**Why Factory Pattern?**
- Decouples client code from concrete implementations
- Simplifies client instantiation
- Centralizes provider selection logic
- Makes it easy to add new providers
- Supports runtime provider selection

### 3. Provider Abstraction

Each repository provider (GitHub, GitLab) implements the `Client` interface independently:

- **GitHubClient**: Uses `github.com/google/go-github` (official GitHub library)
- **GitLabClient**: Uses `github.com/xanzy/go-gitlab` (official GitLab library)

**Design Decisions:**
- Use official provider libraries when available (better maintained, more features)
- Normalize differences between providers in the adapter layer
- Keep provider-specific logic isolated to respective files
- Map provider-specific types to common `FileInfo` and `RepositoryInfo` structures

### 4. Separation of Concerns

The architecture separates different responsibilities:

- **pkg/repository/**: Pure library code, no CLI dependencies
- **cmd/devdashboard/**: CLI-specific code (argument parsing, environment variables, output formatting)
- **examples/**: Demonstration code showing library usage

This separation means:
- The library can be imported into any Go project
- CLI can be replaced with a web interface without touching library code
- Multiple interfaces (CLI, web, GUI) can share the same core library

### 5. Configuration Management

Configuration is handled through a simple `Config` struct:

```go
type Config struct {
    Token   string  // Authentication token
    BaseURL string  // Custom API endpoint
}
```

**Rationale:**
- Simple struct is easy to construct and pass around
- Supports both public and private repositories
- Allows for self-hosted instances
- No hidden global state
- Easy to test with different configurations

## Component Details

### Repository Client Interface

**Purpose:** Define a common API for all repository providers

**Responsibilities:**
- List files in repositories (recursive and non-recursive)
- Retrieve repository metadata
- Handle authentication
- Support different Git references (branches, tags, commits)

**Why This Design?**
- Start with essential operations needed for file management
- Keep interface focused and cohesive
- Easy to extend with additional methods
- All providers can reasonably implement these operations

### GitHub Client

**Implementation Notes:**
- Uses official `go-github` v57 library
- Leverages GitHub's tree API for efficient recursive listing
- Supports GitHub Enterprise via custom BaseURL
- Uses OAuth2 for authentication

**Tradeoffs:**
- Tree API is efficient but doesn't provide file sizes for all entries
- Directory listing API has limitations but provides more metadata
- Chose to use both APIs strategically for different operations

### GitLab Client

**Implementation Notes:**
- Uses official `go-gitlab` library
- Handles pagination for large repositories
- Supports self-hosted GitLab instances
- Uses Personal Access Tokens for authentication

**Tradeoffs:**
- Pagination adds complexity but handles large repositories correctly
- GitLab API uses "project ID or namespace/project" format
- Must construct web URLs manually (not provided by API)

### Factory

**Pattern:** Abstract Factory with string-based selection

**Why String-Based?**
- Allows runtime provider selection (e.g., from config files, environment variables)
- Simpler than enumeration types for user input
- Case-insensitive matching improves usability
- Easy to extend with new providers

**Alternative Considered:**
- Enumeration types: More type-safe but less flexible
- Configuration files: More complex, not needed yet
- Plugin system: Over-engineered for current needs

## Data Flow

### Basic Repository Query Flow

```
User Code
    ↓
Factory.CreateClient("github")
    ↓
GitHubClient created with Config
    ↓
client.ListFiles(ctx, owner, repo, ref, path)
    ↓
GitHub API Call (via go-github)
    ↓
Raw GitHub data structures
    ↓
Convert to FileInfo structs
    ↓
Return []FileInfo to user
```

### Multi-Provider Flow

```
Factory with shared Config
    ↓
Create multiple clients
    ├─→ GitHubClient
    └─→ GitLabClient
         ↓
Query different repositories
         ↓
Normalize to common types
         ↓
User processes unified data
```

## Extensibility Points

### Adding New Repository Providers

To add a new provider (e.g., Bitbucket):

1. Create `bitbucket.go` implementing `Client` interface
2. Add provider constant to `factory.go`
3. Update factory's `CreateClient()` switch statement
4. Add provider to `SupportedProviders()` function
5. Update documentation

**No changes required to:**
- Existing provider implementations
- Client interface
- CLI tool (unless adding provider-specific features)
- User code using the library

### Adding New Operations

To add new capabilities (e.g., file content retrieval):

1. Add method to `Client` interface
2. Implement in all existing providers
3. Existing code continues to work
4. New functionality available to all providers

## Testing Strategy

### Unit Tests

**What We Test:**
- Factory creates correct client types
- Case-insensitive provider matching
- Error handling for unsupported providers
- Configuration passing
- Helper functions

**What We Don't Test (Yet):**
- Actual API calls (would require mocking or integration tests)
- Network failures and retries
- Rate limiting behavior

### Future Testing Plans

- Integration tests with test repositories
- Mock API responses for predictable testing
- Rate limit handling tests
- Authentication failure scenarios
- Timeout and cancellation tests

## Security Considerations

### Token Handling

**Current Approach:**
- Tokens passed via environment variables or Config struct
- Never logged or printed
- No token storage or caching
- User responsible for token security

**Future Enhancements:**
- Support for credential managers
- Token encryption at rest
- Automatic token refresh (OAuth flows)
- Audit logging of API access

### API Security

- Uses HTTPS for all API calls (enforced by libraries)
- Supports custom CA certificates (via BaseURL)
- No sensitive data in URLs (tokens in headers)
- Respects provider API rate limits

## Performance Considerations

### Current Optimizations

1. **Recursive Listing**: Uses tree API when available (GitHub) for single-request recursive listing
2. **Pagination**: Handles large repositories by processing in chunks (GitLab)
3. **Context Support**: All operations support cancellation and timeouts
4. **Streaming**: No unnecessary buffering of large result sets

### Known Limitations

1. **No Caching**: Every request hits the API (by design for freshness)
2. **Rate Limiting**: No automatic rate limit handling or backoff
3. **Parallelization**: No concurrent requests for multiple repositories
4. **Memory**: Large repositories loaded entirely into memory

### Future Optimizations

- Response caching layer with TTL
- Automatic rate limit detection and backoff
- Concurrent repository queries
- Streaming results for very large repositories
- Incremental/diff queries

## Dependencies

### Core Dependencies

- `github.com/google/go-github/v57`: Official GitHub API library
- `github.com/xanzy/go-gitlab`: Official GitLab API library
- `golang.org/x/oauth2`: OAuth2 authentication

**Why These Libraries?**
- Official or widely-adopted libraries
- Well-maintained and documented
- Handle API versioning and changes
- Include rate limiting and retry logic
- Type-safe API wrappers

### Dependency Management

- Go modules for version management
- Semantic versioning where available
- Regular dependency updates planned
- No transitive dependency conflicts currently

## Future Architecture Plans

### Web Dashboard

```
Web Server (cmd/webdashboard/)
    ↓
HTTP Handlers
    ↓
Uses pkg/repository/ (same as CLI)
    ↓
Renders HTML/JSON responses
```

- Same core library
- Add web-specific presentation layer
- WebSocket support for real-time updates
- REST API for external integrations

### GUI Application

```
GUI Framework (cmd/gui/)
    ↓
UI Components
    ↓
Uses pkg/repository/ (same library)
    ↓
Native desktop application
```

- Platform-specific builds
- Share business logic with CLI and web
- Local configuration management
- Desktop notifications

### Planned Features

1. **Caching Layer**: Add `pkg/cache/` for response caching
2. **Metrics**: Add `pkg/metrics/` for monitoring and observability
3. **Webhooks**: Add `pkg/webhooks/` for event-driven updates
4. **Analysis**: Add `pkg/analysis/` for code metrics and insights
5. **Compare**: Add `pkg/compare/` for repository diffing

## Design Patterns Used

1. **Factory Pattern**: Client creation
2. **Interface Segregation**: Small, focused interfaces
3. **Dependency Injection**: Config passed to constructors
4. **Adapter Pattern**: Normalize provider-specific APIs
5. **Builder Pattern**: (Future) Complex query construction

## Lessons Learned

### What Worked Well

- Interface-first design made testing easy
- Factory pattern simplified provider selection
- Official libraries saved significant development time
- Clear separation between library and CLI

### Challenges

- Different pagination strategies across providers
- Mapping different type systems to common structs
- Balancing efficiency vs. completeness in file listings
- Determining minimal viable interface

### Would Do Differently

- Add caching from the start for better performance
- Include rate limit handling earlier
- More comprehensive error types
- Consider async/streaming APIs for large operations

## Contributing Guidelines

When adding new features:

1. **Maintain Interface Compatibility**: Don't break existing code
2. **Update All Providers**: New interface methods need implementations
3. **Add Tests**: Unit tests for new functionality
4. **Document**: Update README and this architecture doc
5. **Follow Patterns**: Use established patterns in the codebase

## Conclusion

DevDashboard's architecture prioritizes:
- **Extensibility**: Easy to add new providers and features
- **Testability**: Interfaces enable comprehensive testing
- **Reusability**: Library can be used in multiple contexts
- **Maintainability**: Clear structure and separation of concerns

The foundation is solid for building a comprehensive multi-interface development dashboard system.