# DevDashboard CLI Guide

Complete guide to using the DevDashboard command-line interface.

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Commands](#commands)
  - [Repository Commands](#repository-commands)
  - [Dependency Commands](#dependency-commands)
- [Environment Variables](#environment-variables)
- [Common Workflows](#common-workflows)
- [Examples](#examples)
- [Troubleshooting](#troubleshooting)

## Installation

### Build from Source

```bash
cd devdashboard
go build -o devdashboard ./cmd/devdashboard
```

### Install to PATH

```bash
make install
# Or manually:
go install ./cmd/devdashboard
```

## Quick Start

The CLI uses environment variables for configuration. Set up your environment and run commands:

```bash
# Set up repository access
export REPO_PROVIDER=github
export REPO_OWNER=golang
export REPO_NAME=go

# Get repository information
devdashboard repo-info

# List all files
devdashboard list-files

# Find dependency files
export ANALYZER_TYPE=poetry
devdashboard find-dependencies
```

## Commands

### Repository Commands

#### `repo-info`

Get metadata about a repository.

**Required Environment Variables:**
- `REPO_PROVIDER` - Repository provider (github, gitlab)
- `REPO_OWNER` - Repository owner/organization
- `REPO_NAME` - Repository name

**Optional Environment Variables:**
- `REPO_TOKEN` - Authentication token for private repos
- `REPO_BASEURL` - Custom API endpoint for self-hosted instances

**Example:**

```bash
export REPO_PROVIDER=github
export REPO_OWNER=torvalds
export REPO_NAME=linux
devdashboard repo-info
```

**Output:**

```
Repository Information (github)
========================================
ID:             2325298
Name:           linux
Full Name:      torvalds/linux
Description:    Linux kernel source tree
Default Branch: master
URL:            https://github.com/torvalds/linux
```

#### `list-files`

List all files in a repository recursively.

**Required Environment Variables:**
- `REPO_PROVIDER` - Repository provider
- `REPO_OWNER` - Repository owner
- `REPO_NAME` - Repository name

**Optional Environment Variables:**
- `REPO_REF` - Git reference (branch, tag, commit SHA)
- `REPO_TOKEN` - Authentication token
- `REPO_BASEURL` - Custom API endpoint

**Example:**

```bash
export REPO_PROVIDER=github
export REPO_OWNER=golang
export REPO_NAME=example
devdashboard list-files
```

**Output:**

```
Listing files from github repository: golang/example

Found 71 files:

LICENSE                                                                           (SHA: 2a7cf70d)
README.md                                                                         (SHA: 88aa3ed2)
appengine-hello/app.go                                                            (SHA: 2951fe77)
...
```

**With Specific Branch:**

```bash
export REPO_REF=develop
devdashboard list-files
```

### Dependency Commands

#### `find-dependencies`

Find dependency files in a repository.

**Required Environment Variables:**
- `REPO_PROVIDER` - Repository provider
- `REPO_OWNER` - Repository owner
- `REPO_NAME` - Repository name

**Optional Environment Variables:**
- `ANALYZER_TYPE` - Analyzer type (defaults to `poetry`)
- `SEARCH_PATHS` - Comma-separated paths to search
- `REPO_REF` - Git reference
- `REPO_TOKEN` - Authentication token

**Example:**

```bash
export REPO_PROVIDER=github
export REPO_OWNER=python-poetry
export REPO_NAME=poetry
export ANALYZER_TYPE=poetry
devdashboard find-dependencies
```

**Output:**

```
Searching for poetry dependency files in python-poetry/poetry

Found 14 dependency file(s):

1. poetry.lock
   Type: poetry.lock
   Analyzer: poetry

2. tests/fixtures/deleted_directory_dependency/poetry.lock
   Type: poetry.lock
   Analyzer: poetry

...
```

**Search Specific Paths:**

```bash
export SEARCH_PATHS="src,packages"
devdashboard find-dependencies
```

#### `analyze-dependencies`

Analyze dependencies from dependency files in a repository.

**Required Environment Variables:**
- `REPO_PROVIDER` - Repository provider
- `REPO_OWNER` - Repository owner
- `REPO_NAME` - Repository name

**Optional Environment Variables:**
- `ANALYZER_TYPE` - Analyzer type (defaults to `poetry`)
- `SEARCH_PATHS` - Comma-separated paths to search
- `REPO_REF` - Git reference
- `REPO_TOKEN` - Authentication token

**Example:**

```bash
export REPO_PROVIDER=github
export REPO_OWNER=python-poetry
export REPO_NAME=poetry
export ANALYZER_TYPE=poetry
devdashboard analyze-dependencies
```

**Output:**

```
Analyzing poetry dependencies in python-poetry/poetry

Step 1: Finding dependency files...
Found 14 dependency file(s)

Step 2: Analyzing dependencies...

Analysis Results
================

File: poetry.lock
Dependencies: 69

  build                           v1.0.3            [runtime]
  cachecontrol                    v0.14.0           [runtime]
  certifi                         v2024.2.2         [runtime]
  ...

--------------------------------------------------------------------------------

Summary
=======
Files analyzed: 13
Total dependencies: 118

Dependencies by type:
  runtime        : 117
  optional       : 1
```

#### `help`

Display help information about all available commands.

**Example:**

```bash
devdashboard help
```

## Environment Variables

### Repository Configuration

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `REPO_PROVIDER` | Yes | Repository provider | `github`, `gitlab` |
| `REPO_OWNER` | Yes | Repository owner/org | `torvalds`, `gitlab-org` |
| `REPO_NAME` | Yes | Repository name | `linux`, `gitlab` |
| `REPO_REF` | No | Git reference | `master`, `v1.0.0`, `abc123` |
| `REPO_TOKEN` | No | Authentication token | Your personal access token |
| `REPO_BASEURL` | No | Custom API endpoint | `https://github.example.com` |

### Dependency Analysis Configuration

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `ANALYZER_TYPE` | No | Dependency analyzer | `poetry`, `npm` (default: `poetry`) |
| `SEARCH_PATHS` | No | Paths to search | `src,packages,services` |

### Setting Environment Variables

**Linux/macOS:**

```bash
export REPO_PROVIDER=github
export REPO_OWNER=myorg
export REPO_NAME=myrepo
```

**Windows (PowerShell):**

```powershell
$env:REPO_PROVIDER="github"
$env:REPO_OWNER="myorg"
$env:REPO_NAME="myrepo"
```

**Windows (CMD):**

```cmd
set REPO_PROVIDER=github
set REPO_OWNER=myorg
set REPO_NAME=myrepo
```

### Using .env Files

Create a `.env` file (add to `.gitignore`):

```bash
REPO_PROVIDER=github
REPO_TOKEN=ghp_your_token_here
REPO_OWNER=myorg
REPO_NAME=myrepo
ANALYZER_TYPE=poetry
```

Load with:

```bash
export $(cat .env | xargs)
devdashboard repo-info
```

## Common Workflows

### Workflow 1: Explore a New Repository

```bash
# Set up
export REPO_PROVIDER=github
export REPO_OWNER=python-poetry
export REPO_NAME=poetry

# Get basic info
devdashboard repo-info

# See what files are there
devdashboard list-files

# Find dependency files
export ANALYZER_TYPE=poetry
devdashboard find-dependencies

# Analyze dependencies
devdashboard analyze-dependencies
```

### Workflow 2: Analyze Private Repository

```bash
# Set up with authentication
export REPO_PROVIDER=github
export REPO_TOKEN=ghp_your_token_here
export REPO_OWNER=mycompany
export REPO_NAME=private-api

# Get repository info
devdashboard repo-info

# Analyze dependencies
export ANALYZER_TYPE=poetry
devdashboard analyze-dependencies
```

### Workflow 3: Compare Dependencies Across Branches

```bash
# Analyze main branch
export REPO_PROVIDER=github
export REPO_OWNER=myorg
export REPO_NAME=myrepo
export REPO_REF=main
export ANALYZER_TYPE=poetry
devdashboard analyze-dependencies > deps-main.txt

# Analyze develop branch
export REPO_REF=develop
devdashboard analyze-dependencies > deps-develop.txt

# Compare
diff deps-main.txt deps-develop.txt
```

### Workflow 4: Audit Multiple Repositories

```bash
#!/bin/bash

REPOS=(
    "org1/repo1"
    "org2/repo2"
    "org3/repo3"
)

export REPO_PROVIDER=github
export ANALYZER_TYPE=poetry

for repo in "${REPOS[@]}"; do
    IFS='/' read -r owner name <<< "$repo"
    export REPO_OWNER=$owner
    export REPO_NAME=$name
    
    echo "Analyzing $repo..."
    devdashboard analyze-dependencies > "deps-${owner}-${name}.txt"
done
```

### Workflow 5: Search Specific Paths

```bash
# Only search in source directories
export REPO_PROVIDER=github
export REPO_OWNER=myorg
export REPO_NAME=monorepo
export SEARCH_PATHS="services/api,services/web,packages/shared"
export ANALYZER_TYPE=poetry

devdashboard find-dependencies
```

## Examples

### Example 1: Public GitHub Repository

```bash
export REPO_PROVIDER=github
export REPO_OWNER=golang
export REPO_NAME=go

# Get info
devdashboard repo-info

# List files
devdashboard list-files
```

### Example 2: Private GitLab Repository

```bash
export REPO_PROVIDER=gitlab
export REPO_TOKEN=glpat-your-token
export REPO_OWNER=myteam
export REPO_NAME=backend-api

devdashboard repo-info
```

### Example 3: Self-Hosted GitLab

```bash
export REPO_PROVIDER=gitlab
export REPO_BASEURL=https://gitlab.company.com
export REPO_TOKEN=your-token
export REPO_OWNER=engineering
export REPO_NAME=platform

devdashboard list-files
```

### Example 4: Analyze Python Poetry Project

```bash
export REPO_PROVIDER=github
export REPO_OWNER=python-poetry
export REPO_NAME=poetry
export ANALYZER_TYPE=poetry

# Find all poetry.lock files
devdashboard find-dependencies

# Analyze all dependencies
devdashboard analyze-dependencies
```

### Example 5: Specific Branch Analysis

```bash
export REPO_PROVIDER=github
export REPO_OWNER=myorg
export REPO_NAME=myapp
export REPO_REF=release/v2.0
export ANALYZER_TYPE=poetry

devdashboard analyze-dependencies
```

## Troubleshooting

### Error: "REPO_PROVIDER environment variable is required"

**Problem:** Required environment variable not set.

**Solution:**

```bash
export REPO_PROVIDER=github
export REPO_OWNER=owner
export REPO_NAME=repo
```

### Error: "failed to create github client"

**Problem:** Invalid provider name.

**Solution:** Use `github` or `gitlab` (case-insensitive):

```bash
export REPO_PROVIDER=github  # Not "GitHub" or "GITHUB"
```

### Error: "failed to get repository info"

**Possible Causes:**
1. Repository doesn't exist
2. Private repository without authentication
3. Incorrect owner/name
4. Network issues

**Solutions:**

```bash
# Check repository exists
# Verify owner and name are correct

# For private repos, add token
export REPO_TOKEN=your-token

# For self-hosted instances
export REPO_BASEURL=https://your-instance.com
```

### Error: "No dependency files found"

**Possible Causes:**
1. Repository doesn't have dependency files
2. Wrong analyzer type
3. Search paths exclude the files

**Solutions:**

```bash
# Verify analyzer type is correct
export ANALYZER_TYPE=poetry  # For Python projects

# Remove search paths restriction
unset SEARCH_PATHS

# List all files to see what's there
devdashboard list-files | grep -i lock
```

### Error: "context deadline exceeded"

**Problem:** Operation timed out.

**Solution:** Large repositories take time. The timeout is built-in, but you can:

1. Limit search paths:
   ```bash
   export SEARCH_PATHS="src"
   ```

2. Try a different branch with fewer files

3. Check network connection

### Slow Performance

**Problem:** Commands take a long time to run.

**Solutions:**

1. **Limit search scope:**
   ```bash
   export SEARCH_PATHS="src,packages"
   ```

2. **Use specific branches:**
   ```bash
   export REPO_REF=release/v1.0  # Smaller than main
   ```

3. **Cache authentication:**
   Save token in environment to avoid re-authenticating

### Authentication Issues

**GitHub Personal Access Token:**

1. Go to Settings → Developer settings → Personal access tokens
2. Generate new token with `repo` scope
3. Copy token
4. Set environment variable:
   ```bash
   export REPO_TOKEN=ghp_your_token_here
   ```

**GitLab Personal Access Token:**

1. Go to User Settings → Access Tokens
2. Create token with `read_api` and `read_repository` scopes
3. Copy token
4. Set environment variable:
   ```bash
   export REPO_TOKEN=glpat_your_token_here
   ```

### Rate Limiting

**Problem:** "API rate limit exceeded"

**Solutions:**

1. Use authentication (higher rate limits)
2. Wait for rate limit to reset
3. For GitHub: 60 req/hour (anonymous), 5000 req/hour (authenticated)

### Unsupported Analyzer

**Problem:** "unsupported analyzer type: npm"

**Solution:** Currently only `poetry` is supported. More analyzers coming soon.

Check supported analyzers in the codebase or documentation.

## Tips and Best Practices

### 1. Use Shell Aliases

Create aliases for common operations:

```bash
# Add to .bashrc or .zshrc
alias dd='devdashboard'
alias dd-info='devdashboard repo-info'
alias dd-deps='devdashboard analyze-dependencies'

# Usage
dd-info
dd-deps
```

### 2. Create Helper Scripts

```bash
#!/bin/bash
# analyze-repo.sh

export REPO_PROVIDER=${1:-github}
export REPO_OWNER=$2
export REPO_NAME=$3
export ANALYZER_TYPE=${4:-poetry}

devdashboard analyze-dependencies
```

Usage:
```bash
./analyze-repo.sh github python-poetry poetry poetry
```

### 3. Store Tokens Securely

Don't hardcode tokens. Use:
- Environment variables
- `.env` files (gitignored)
- Secret management tools (Vault, AWS Secrets Manager)
- Credential managers

### 4. Pipe Output for Processing

```bash
# Save to file
devdashboard list-files > files.txt

# Count files
devdashboard list-files | wc -l

# Filter results
devdashboard list-files | grep ".py$"

# Process with jq (if output is JSON in future)
# devdashboard repo-info --json | jq '.name'
```

### 5. Use in CI/CD Pipelines

```yaml
# .github/workflows/analyze-deps.yml
name: Analyze Dependencies

on: [push]

jobs:
  analyze:
    runs-on: ubuntu-latest
    steps:
      - name: Download DevDashboard
        run: |
          curl -L https://github.com/greg-hellings/devdashboard/releases/latest/download/devdashboard-linux > devdashboard
          chmod +x devdashboard
      
      - name: Analyze Dependencies
        env:
          REPO_PROVIDER: github
          REPO_OWNER: ${{ github.repository_owner }}
          REPO_NAME: ${{ github.event.repository.name }}
          REPO_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          ANALYZER_TYPE: poetry
        run: ./devdashboard analyze-dependencies
```

## Getting Help

- Run `devdashboard help` for command overview
- Check [README.md](../README.md) for full documentation
- See [DEPENDENCIES.md](DEPENDENCIES.md) for dependency analysis details
- Review [examples/](examples/) for code examples

## Next Steps

- Explore the [API documentation](README.md#api-reference)
- Try the [example programs](examples/)
- Learn about [extending the tool](DEPENDENCY_IMPLEMENTATION.md)
- Contribute to the project!