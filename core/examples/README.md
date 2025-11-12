# DevDashboard Examples

This directory contains example programs demonstrating how to use the DevDashboard library and CLI tool.

## Structure

Each example is in its own subdirectory with a separate Go module to avoid `main` function conflicts:

```
examples/
├── basic-usage/              # Basic repository client usage
│   ├── main.go
│   ├── go.mod
│   └── go.sum
├── dependency-analysis/      # Dependency analysis examples
│   ├── main.go
│   ├── go.mod
│   └── go.sum
├── logging-demo/             # Logging and verbosity examples
│   ├── main.go
│   ├── go.mod
│   └── go.sum
├── pipfile-uvlock-example/   # Pipfile and uv.lock examples
│   ├── main.go
│   ├── go.mod
│   └── go.sum
├── dependency-report-config.yaml
├── dependency-report-paths-example.yaml
└── test-report-config.yaml
```

## Examples

### 1. Basic Usage (`basic-usage/`)

Demonstrates basic repository operations:
- Creating GitHub and GitLab clients
- Listing repository files
- Getting repository information
- Handling authentication

**Run:**
```bash
cd examples/basic-usage
go run main.go
```

### 2. Dependency Analysis (`dependency-analysis/`)

Shows how to analyze dependencies in repositories:
- Finding dependency files (poetry.lock, Pipfile.lock, uv.lock)
- Parsing dependency information
- Extracting versions

**Run:**
```bash
cd examples/dependency-analysis
go run main.go
```

### 3. Logging Demo (`logging-demo/`)

Demonstrates logging capabilities:
- Setting up structured logging with slog
- Using different log levels (DEBUG, INFO, WARN, ERROR)
- Verbose output modes

**Run:**
```bash
cd examples/logging-demo
go run main.go
```

### 4. Pipfile & UV Lock Example (`pipfile-uvlock-example/`)

Shows how to work with Python dependency files:
- Analyzing Pipfile.lock files
- Parsing uv.lock files
- Extracting Python package information

**Run:**
```bash
cd examples/pipfile-uvlock-example
go run main.go
```

## Configuration Examples

The YAML files in this directory demonstrate dependency report configurations:

- **`dependency-report-config.yaml`**: Basic multi-repository configuration
- **`dependency-report-paths-example.yaml`**: Advanced example with explicit file paths
- **`test-report-config.yaml`**: Test configuration template

## Building Examples

### Build a specific example:
```bash
cd examples/basic-usage
go build -o basic-usage
./basic-usage
```

### Build all examples:
```bash
for dir in examples/*/; do
  if [ -f "$dir/main.go" ]; then
    echo "Building $(basename $dir)..."
    (cd "$dir" && go build)
  fi
done
```

## Using Examples in Your Code

Each example uses the DevDashboard library. To use it in your own project:

```go
import (
    "github.com/greg-hellings/devdashboard/pkg/repository"
    "github.com/greg-hellings/devdashboard/pkg/dependencies"
    "github.com/greg-hellings/devdashboard/pkg/config"
)
```

## Environment Variables

Most examples support environment variables for configuration:

| Variable | Description | Example |
|----------|-------------|---------|
| `REPO_PROVIDER` | Repository provider | `github`, `gitlab` |
| `REPO_TOKEN` | Authentication token | `ghp_xxxxx` |
| `REPO_OWNER` | Repository owner | `myorg` |
| `REPO_NAME` | Repository name | `myrepo` |
| `REPO_REF` | Git reference | `main`, `v1.0.0` |
| `ANALYZER_TYPE` | Dependency analyzer | `poetry`, `pipfile`, `uvlock` |

Example:
```bash
export REPO_PROVIDER=github
export REPO_OWNER=python-poetry
export REPO_NAME=poetry
export ANALYZER_TYPE=poetry
cd examples/dependency-analysis
go run main.go
```

## Module Structure

Each example directory contains:

- **`main.go`**: The example code
- **`go.mod`**: Module definition with replace directive
- **`go.sum`**: Dependency checksums

The `go.mod` files use a replace directive to reference the parent module:

```go
module github.com/greg-hellings/devdashboard/examples/basic-usage

go 1.24

replace github.com/greg-hellings/devdashboard => ../..

require github.com/greg-hellings/devdashboard v0.0.0-00010101000000-000000000000
```

This ensures examples always use the local development version of DevDashboard.

## Modifying Examples

1. Navigate to the example directory:
   ```bash
   cd examples/basic-usage
   ```

2. Edit `main.go`

3. Test your changes:
   ```bash
   go run main.go
   ```

4. If you add new dependencies, update the module:
   ```bash
   go mod tidy
   ```

## Common Issues

### Import errors
If you see import errors, ensure you're in the correct directory and have run `go mod tidy`:
```bash
cd examples/basic-usage
go mod tidy
go build
```

### Authentication failures
Many examples require authentication for private repositories. Set the appropriate token:
```bash
export REPO_TOKEN="your-token-here"
```

### Module conflicts
If you see "multiple main functions" errors when running `go vet` from the root:
- This is expected behavior
- Each example is isolated in its own module
- Run `go vet ./pkg/... ./cmd/...` to vet only the main codebase

## Testing Examples in CI

The examples are excluded from the main CI pipeline to avoid module conflicts. To test them:

```bash
# Test each example individually
for dir in examples/*/; do
  if [ -f "$dir/main.go" ]; then
    echo "Testing $(basename $dir)..."
    (cd "$dir" && go build && go vet)
  fi
done
```

## Contributing

When adding new examples:

1. Create a new subdirectory: `examples/my-example/`
2. Add `main.go` with your example code
3. Create `go.mod`:
   ```go
   module github.com/greg-hellings/devdashboard/examples/my-example

   go 1.24

   replace github.com/greg-hellings/devdashboard => ../..

   require github.com/greg-hellings/devdashboard v0.0.0-00010101000000-000000000000
   ```
4. Run `go mod tidy`
5. Test: `go build && go run main.go`
6. Update this README with your example

## Further Reading

- [Main README](../README.md)
- [Dependency Analysis Documentation](../docs/DEPENDENCIES.md)
- [Dependency Report Guide](../docs/DEPENDENCY_REPORT.md)
- [CLI Usage](../docs/CLI_GUIDE.md)
