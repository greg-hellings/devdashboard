# Logging

DevDashboard uses Go's standard `log/slog` package for structured logging. This document describes how logging works and how to control log output.

## Overview

The logging system provides visibility into the application's operations, especially useful for:
- Debugging issues with dependency parsing
- Understanding which files are being processed
- Tracking API calls to repository providers
- Identifying errors that are handled gracefully

## Log Levels

DevDashboard supports four log levels, in order of increasing verbosity:

| Level | Purpose | What's Logged |
|-------|---------|---------------|
| **ERROR** | Critical failures | Fatal errors that prevent operation |
| **WARN** | Warnings | Potential issues that don't stop execution |
| **INFO** | Informational | High-level operation progress |
| **DEBUG** | Debug | Detailed execution information, including handled errors |

## CLI Usage

### Default Behavior (WARN Level)

By default, the CLI runs at WARN level, showing only warnings and errors:

```bash
devdashboard list-files
```

This keeps output clean and focused on the command results.

### Verbose Mode (INFO Level)

Use `-v` or `--verbose` to enable INFO level logging:

```bash
devdashboard -v find-dependencies
devdashboard --verbose analyze-dependencies
```

**Example output:**
```
time=2025-01-15T10:30:00.000-05:00 level=INFO msg="Finding dependency files" provider=github owner=myorg repo=myrepo ref=main analyzer=poetry
time=2025-01-15T10:30:01.234-05:00 level=INFO msg="Candidate files found" count=3
```

### Debug Mode (DEBUG Level)

Use `-vv` or `--debug` to enable DEBUG level logging:

```bash
devdashboard -vv analyze-dependencies
devdashboard --debug find-dependencies
```

**Example output:**
```
time=2025-01-15T10:30:00.000-05:00 level=DEBUG msg="Logging initialized" level=DEBUG
time=2025-01-15T10:30:00.100-05:00 level=DEBUG msg="Starting analyze-dependencies command"
time=2025-01-15T10:30:01.234-05:00 level=INFO msg="Finding dependency files" provider=github owner=myorg repo=myrepo
time=2025-01-15T10:30:02.456-05:00 level=DEBUG msg="Failed to parse poetry.lock content" file=broken/poetry.lock error="toml: line 1: syntax error"
```

## What Gets Logged

### INFO Level Messages

- Command execution start/completion
- Repository operations (listing files, getting info)
- Number of files found/processed
- High-level progress indicators

### DEBUG Level Messages

- Detailed command initialization
- Individual file processing attempts
- **Parse failures** - When a lock file has invalid syntax
- **Network errors** - When fetching file content fails
- **Configuration details** - Repository paths, analyzer types, etc.

## Error Handling and Logging

DevDashboard is designed to be resilient. When analyzing multiple dependency files:

1. **Fatal errors** stop execution and return an error code
2. **Non-fatal errors** (like one corrupt file among many) are logged at DEBUG level and skipped

This means you can run analysis on a repository with some broken files, and it will still process the valid ones.

### Example: Analyzing with Some Invalid Files

```bash
export REPO_PROVIDER=github
export REPO_OWNER=myorg
export REPO_NAME=myrepo
export ANALYZER_TYPE=poetry

# Without debug - silently skips bad files
devdashboard analyze-dependencies
# Output: Shows only successfully parsed files

# With debug - shows why files were skipped
devdashboard --debug analyze-dependencies
# Output: Shows parse errors for invalid files
```

## Library Usage

When using DevDashboard as a library, you can configure slog yourself:

```go
import (
    "log/slog"
    "os"
    
    "github.com/greg-hellings/devdashboard/pkg/dependencies"
)

func main() {
    // Set log level to DEBUG
    handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
        Level: slog.LevelDebug,
    })
    slog.SetDefault(slog.New(handler))
    
    // Now all DevDashboard operations will log at DEBUG level
    analyzer := dependencies.NewPoetryAnalyzer()
    // ... use analyzer
}
```

### JSON Logging

For production environments, you might prefer JSON-formatted logs:

```go
handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
    Level: slog.LevelInfo,
})
slog.SetDefault(slog.New(handler))
```

### Custom Handler

You can use any `slog.Handler` implementation:

```go
// Send logs to a custom destination
type CustomHandler struct {
    // ... implementation
}

slog.SetDefault(slog.New(&CustomHandler{}))
```

## Log Message Format

Log messages use structured logging with key-value pairs:

```
time=<timestamp> level=<LEVEL> msg="<message>" key1=value1 key2=value2 ...
```

### Common Fields

- `time` - ISO 8601 timestamp
- `level` - Log level (DEBUG, INFO, WARN, ERROR)
- `msg` - Human-readable message
- `file` - File being processed
- `owner` - Repository owner
- `repo` - Repository name
- `ref` - Git reference (branch/tag/commit)
- `analyzer` - Analyzer type (poetry, pipfile, uvlock)
- `error` - Error message (when applicable)
- `count` - Number of items (files, dependencies, etc.)

### Example Log Entry

```
time=2025-01-15T10:30:02.456-05:00 level=DEBUG msg="Failed to analyze poetry.lock file" file=api/poetry.lock owner=myorg repo=myrepo ref=main error="failed to parse api/poetry.lock: toml: line 5: syntax error"
```

## Best Practices

### During Development

Use DEBUG level to see everything:
```bash
devdashboard --debug <command>
```

### In Scripts

Use INFO level to track progress:
```bash
devdashboard -v <command>
```

### In CI/CD

Use default (WARN) level to keep logs clean:
```bash
devdashboard <command>
```

Enable DEBUG only when investigating failures.

### Redirecting Logs

Logs are written to stderr, so you can redirect them separately:

```bash
# Save command output to file, logs to console
devdashboard list-files > files.txt

# Save both output and logs to separate files
devdashboard -v list-files > files.txt 2> logs.txt

# Suppress logs entirely
devdashboard list-files 2>/dev/null
```

## Troubleshooting with Logs

### Problem: Dependency files not found

```bash
devdashboard --debug find-dependencies
```

Look for DEBUG messages about file discovery to see what paths are being searched.

### Problem: Dependencies not parsing

```bash
devdashboard --debug analyze-dependencies
```

Look for DEBUG messages like:
- `"Failed to parse poetry.lock content"` - Shows the parsing error
- `"Failed to analyze poetry.lock file"` - Shows which file failed and why

### Problem: Empty results

Enable debug logging to see if:
1. Files are being found but failing to parse
2. Network errors are preventing file retrieval
3. Authentication issues with the repository provider

## Performance Considerations

- DEBUG logging adds minimal overhead (~1-5% on most operations)
- Most debug messages are about error handling, not hot paths
- Log level can be changed at runtime when using the library

## Examples

See `examples/logging_demo.go` for a complete demonstration of logging at different levels.

```bash
go run examples/logging_demo.go
```

This example shows:
- How log output changes at different levels
- How errors are logged when files fail to parse
- The difference between fatal and non-fatal errors

## Related Documentation

- [CLI Guide](CLI_GUIDE.md) - Using the command-line interface
- [Dependency Analysis](DEPENDENCIES.md) - Understanding dependency analyzers
- [Troubleshooting](TROUBLESHOOTING.md) - Common issues and solutions