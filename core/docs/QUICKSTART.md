# Quick Start Guide

Get up and running with DevDashboard in 5 minutes!

## Prerequisites

- Go 1.24 or higher installed
- Git (for cloning repositories)
- Optional: GitHub or GitLab personal access token (for private repositories)

## Installation

### Option 1: Build from Source

```bash
# Navigate to the project directory
cd devdashboard

# Download dependencies
go mod download

# Build the CLI tool
make build

# The binary will be in bin/devdashboard
```

### Option 2: Install to GOPATH

```bash
# Install directly to your GOPATH/bin
make install

# Now you can run 'devdashboard' from anywhere
```

## Your First Command

Let's list files from a public GitHub repository:

```bash
# Set up environment variables
export REPO_PROVIDER=github
export REPO_OWNER=golang
export REPO_NAME=example

# Get repository information
./bin/devdashboard repo-info
```

You should see output like:

```
Repository Information (github)
========================================
ID:             22327691
Name:           example
Full Name:      golang/example
Description:    Go example projects
Default Branch: master
URL:            https://github.com/golang/example
```

## List All Files

Now let's list all files in the repository:

```bash
./bin/devdashboard list-files
```

This will recursively traverse the repository and display all files with their Git SHA.

## Working with GitLab

Switch to GitLab by changing the provider:

```bash
export REPO_PROVIDER=gitlab
export REPO_OWNER=gitlab-org
export REPO_NAME=gitlab-foss

./bin/devdashboard repo-info
```

## Accessing Private Repositories

To access private repositories, you'll need an authentication token.

### GitHub Token

1. Go to GitHub Settings → Developer settings → Personal access tokens
2. Generate a new token with `repo` scope
3. Copy the token

### GitLab Token

1. Go to GitLab User Settings → Access Tokens
2. Create a token with `read_api` and `read_repository` scopes
3. Copy the token

### Use the Token

```bash
export REPO_TOKEN=your-token-here
export REPO_PROVIDER=github
export REPO_OWNER=your-username
export REPO_NAME=your-private-repo

./bin/devdashboard repo-info
```

## Working with Specific Branches

You can specify a branch, tag, or commit:

```bash
export REPO_REF=develop
./bin/devdashboard list-files
```

## Self-Hosted Instances

For GitHub Enterprise or self-hosted GitLab:

```bash
export REPO_BASEURL=https://gitlab.example.com
export REPO_TOKEN=your-token
export REPO_PROVIDER=gitlab
export REPO_OWNER=team
export REPO_NAME=project

./bin/devdashboard repo-info
```

## Analyzing Dependencies

DevDashboard can find and analyze dependency files in repositories.

### Find Dependency Files

```bash
export REPO_PROVIDER=github
export REPO_OWNER=python-poetry
export REPO_NAME=poetry
export ANALYZER_TYPE=poetry
./bin/devdashboard find-dependencies
```

This will search the repository and list all Poetry lock files found.

### Analyze Dependencies

```bash
export REPO_PROVIDER=github
export REPO_OWNER=python-poetry
export REPO_NAME=poetry
export ANALYZER_TYPE=poetry
./bin/devdashboard analyze-dependencies
```

This will:
1. Find all dependency files
2. Parse each file
3. Extract dependency information
4. Display results with summary statistics

### Search Specific Paths

Limit the search to specific directories:

```bash
export SEARCH_PATHS="src,packages,services"
./bin/devdashboard find-dependencies
```

## Running the Examples

We've included example code that demonstrates various use cases:

```bash
# Build the example
make example

# Run it
./bin/basic_usage
```

The examples demonstrate:
- Connecting to GitHub and GitLab
- Public and private repository access
- Listing files and directories
- Using the factory pattern
- Dependency analysis
- Error handling

## Using as a Library

You can import DevDashboard into your own Go projects:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/greg-hellings/devdashboard/pkg/repository"
)

func main() {
    // Create a client
    config := repository.Config{
        Token: "optional-token",
    }

    client, err := repository.NewClient("github", config)
    if err != nil {
        log.Fatal(err)
    }

    // Get repository info
    ctx := context.Background()
    info, err := client.GetRepositoryInfo(ctx, "golang", "go")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Repository: %s\n", info.FullName)
    fmt.Printf("Description: %s\n", info.Description)
}
```

## Common Tasks

### List Files in a Specific Directory

```bash
# Use the library in your code to specify a path
# The CLI currently lists all files recursively
```

In code:

```go
files, err := client.ListFiles(ctx, "owner", "repo", "main", "src/path")
```

### Compare Two Repositories

```go
// Create clients for both providers
githubClient, _ := repository.NewClient("github", config)
gitlabClient, _ := repository.NewClient("gitlab", config)

// Fetch files from both
githubFiles, _ := githubClient.ListFilesRecursive(ctx, "owner", "repo1", "")
gitlabFiles, _ := gitlabClient.ListFilesRecursive(ctx, "owner", "repo2", "")

// Compare the results
```

### Work with Multiple Repositories

```go
factory := repository.NewFactory(config)

repos := []struct {
    provider, owner, name string
}{
    {"github", "golang", "go"},
    {"github", "golang", "tools"},
    {"gitlab", "gitlab-org", "gitlab-foss"},
}

for _, r := range repos {
    client, _ := factory.CreateClient(r.provider)
    info, _ := client.GetRepositoryInfo(ctx, r.owner, r.name)
    fmt.Printf("Repository: %s\n", info.FullName)
}
```

## Troubleshooting

### Authentication Errors

If you get authentication errors:
- Verify your token is correct
- Check token hasn't expired
- Ensure token has correct permissions (scopes)

### Rate Limiting

GitHub and GitLab have API rate limits:
- GitHub: 60 requests/hour (unauthenticated), 5000 requests/hour (authenticated)
- GitLab: 300 requests/minute (authenticated)

Use authentication to increase limits.

### Connection Errors

For self-hosted instances:
- Verify the REPO_BASEURL is correct
- Check network connectivity
- Ensure SSL certificates are valid

## Next Steps

- Read the full [README.md](../README.md) for detailed API documentation
- Explore the [examples](../examples/) directory for more code samples
- Check out the [pkg/repository](../pkg/repository/) package for interface details
- Start building your own integrations!

## Getting Help

- Check the full README for comprehensive documentation
- Review example code in the `examples/` directory
- Look at interface definitions in `pkg/repository/repository.go`

## Quick Reference

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `REPO_PROVIDER` | Yes | `github` or `gitlab` |
| `REPO_OWNER` | Yes | Repository owner/organization |
| `REPO_NAME` | Yes | Repository name |
| `REPO_TOKEN` | No | Authentication token |
| `REPO_BASEURL` | No | Custom API base URL |
| `REPO_REF` | No | Branch/tag/commit |
| `ANALYZER_TYPE` | No | Dependency analyzer type |
| `SEARCH_PATHS` | No | Comma-separated search paths |

### CLI Commands

| Command | Description |
|---------|-------------|
| `repo-info` | Get repository metadata |
| `list-files` | List all files recursively |
| `find-dependencies` | Find dependency files |
| `analyze-dependencies` | Analyze dependencies |
| `help` | Show help message |

### Makefile Targets

| Target | Description |
|--------|-------------|
| `make build` | Build CLI tool |
| `make example` | Build examples |
| `make test` | Run tests |
| `make clean` | Clean build artifacts |
| `make help` | Show all targets |
