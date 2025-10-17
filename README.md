# DevDashboard

A comprehensive development dashboard system providing CLI tools, web interface, and GUI applications for managing and monitoring repositories across multiple platforms.

## Project Structure

```
devdashboard/
├── cmd/
│   └── devdashboard/          # CLI application entry point
├── pkg/
│   └── repository/            # Repository connector modules
│       ├── repository.go      # Core interfaces and types
│       ├── github.go          # GitHub implementation
│       ├── gitlab.go          # GitLab implementation
│       └── factory.go         # Factory pattern for client creation
├── go.mod                     # Go module dependencies
└── README.md                  # This file
```

## Features

### Current Implementation

- **Multi-Provider Support**: Connect to GitHub and GitLab repositories
- **Public & Private Repositories**: Support for authenticated and unauthenticated access
- **Self-Hosted Instances**: Compatible with GitHub Enterprise and self-hosted GitLab
- **Flexible API**: List files, retrieve repository metadata, and traverse entire repository trees
- **Factory Pattern**: Easy client instantiation with provider selection

### Planned Features

- Web dashboard for repository visualization
- GUI application for desktop management
- Additional repository providers
- Advanced file analysis and metrics

## Installation

### Prerequisites

- Go 1.21 or higher

### Setup

1. Clone the repository:
```bash
cd /path/to/devdashboard
```

2. Download dependencies:
```bash
go mod download
```

3. Build the CLI tool:
```bash
go build -o devdashboard ./cmd/devdashboard
```

## Usage

### CLI Tool

The CLI tool uses environment variables for configuration to keep sensitive tokens out of command history.

#### Environment Variables

- `REPO_PROVIDER` (required): Repository provider (`github` or `gitlab`)
- `REPO_OWNER` (required): Repository owner or organization name
- `REPO_NAME` (required): Repository name
- `REPO_TOKEN` (optional): Authentication token for private repositories
- `REPO_BASEURL` (optional): Custom base URL for self-hosted instances
- `REPO_REF` (optional): Git reference (branch, tag, or commit SHA)

#### Commands

##### List Files

List all files in a repository recursively:

```bash
export REPO_PROVIDER=github
export REPO_OWNER=torvalds
export REPO_NAME=linux
./devdashboard list-files
```

##### Repository Information

Get metadata about a repository:

```bash
export REPO_PROVIDER=gitlab
export REPO_OWNER=gitlab-org
export REPO_NAME=gitlab
./devdashboard repo-info
```

#### Examples

**Public GitHub Repository:**
```bash
export REPO_PROVIDER=github
export REPO_OWNER=golang
export REPO_NAME=go
./devdashboard list-files
```

**Private GitLab Repository:**
```bash
export REPO_PROVIDER=gitlab
export REPO_TOKEN=your-gitlab-token
export REPO_OWNER=myorg
export REPO_NAME=private-repo
./devdashboard repo-info
```

**Self-Hosted GitLab Instance:**
```bash
export REPO_PROVIDER=gitlab
export REPO_BASEURL=https://gitlab.example.com
export REPO_TOKEN=your-token
export REPO_OWNER=team
export REPO_NAME=project
export REPO_REF=develop
./devdashboard list-files
```

### Using as a Library

The repository package can be imported and used in your own Go applications.

#### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/greg-hellings/devdashboard/pkg/repository"
)

func main() {
    // Configure the client
    config := repository.Config{
        Token: "your-token-here", // Optional for public repos
    }
    
    // Create a GitHub client using the factory
    client, err := repository.NewClient("github", config)
    if err != nil {
        log.Fatal(err)
    }
    
    // List files in a repository
    ctx := context.Background()
    files, err := client.ListFilesRecursive(ctx, "golang", "go", "master")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Found %d files\n", len(files))
    for _, file := range files {
        fmt.Printf("%s (%d bytes)\n", file.Path, file.Size)
    }
}
```

#### Using the Factory Pattern

```go
// Create a factory with shared configuration
factory := repository.NewFactory(repository.Config{
    Token: "your-token",
})

// Create different clients from the same factory
githubClient, _ := factory.CreateClient("github")
gitlabClient, _ := factory.CreateClient("gitlab")
```

#### Direct Client Instantiation

```go
// Create clients directly without the factory
githubClient, err := repository.NewGitHubClient(repository.Config{
    Token: "github-token",
})

gitlabClient, err := repository.NewGitLabClient(repository.Config{
    Token: "gitlab-token",
    BaseURL: "https://gitlab.example.com",
})
```

## API Reference

### Client Interface

All repository clients implement the `Client` interface:

```go
type Client interface {
    // List files at a specific path (non-recursive)
    ListFiles(ctx context.Context, owner, repo, ref, path string) ([]FileInfo, error)
    
    // Get repository metadata
    GetRepositoryInfo(ctx context.Context, owner, repo string) (*RepositoryInfo, error)
    
    // List all files recursively
    ListFilesRecursive(ctx context.Context, owner, repo, ref string) ([]FileInfo, error)
}
```

### FileInfo Structure

```go
type FileInfo struct {
    Path     string // Full path to the file
    Name     string // File name
    Type     string // "file", "dir", "symlink"
    Size     int64  // Size in bytes
    Mode     string // File permissions
    SHA      string // Git SHA/commit hash
    URL      string // Web URL to the file
}
```

### RepositoryInfo Structure

```go
type RepositoryInfo struct {
    ID            string // Repository ID
    Name          string // Repository name
    FullName      string // Full name (owner/repo)
    Description   string // Description
    DefaultBranch string // Default branch name
    URL           string // Web URL
}
```

## Authentication

### GitHub

Create a Personal Access Token:
1. Go to Settings → Developer settings → Personal access tokens
2. Generate new token with `repo` scope for private repositories
3. Use the token as `REPO_TOKEN`

### GitLab

Create a Personal Access Token:
1. Go to User Settings → Access Tokens
2. Create token with `read_api` and `read_repository` scopes
3. Use the token as `REPO_TOKEN`

## Development

### Running Tests

```bash
go test ./...
```

### Adding New Providers

To add support for a new repository provider:

1. Create a new file in `pkg/repository/` (e.g., `bitbucket.go`)
2. Implement the `Client` interface
3. Add the provider to the factory in `factory.go`
4. Update documentation

## Contributing

Contributions are welcome! Please ensure your code:
- Follows Go best practices and idioms
- Includes comments explaining complex logic
- Maintains the existing interface contracts
- Works with both public and private repositories

## License

[Specify your license here]

## Roadmap

- [ ] Add Bitbucket support
- [ ] Implement caching layer for API responses
- [ ] Add rate limit handling and retry logic
- [ ] Web dashboard implementation
- [ ] GUI application development
- [ ] File content retrieval and analysis
- [ ] Diff and comparison tools
- [ ] Webhook support for real-time updates