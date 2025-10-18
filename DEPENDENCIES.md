# Dependency Analysis Module

This document provides comprehensive documentation for the dependency analysis module in DevDashboard.

## Overview

The dependency analysis module provides a unified interface for detecting and analyzing dependency files across different package managers and programming languages. It follows the same extensible architecture as the repository module, using the factory pattern to support multiple dependency analyzers.

## Architecture

### Core Components

1. **Analyzer Interface**: Defines the contract all dependency analyzers must implement
2. **Config Structure**: Configures repository paths and provides a repository client
3. **Factory Pattern**: Creates analyzers based on string identifiers
4. **Concrete Implementations**: Specific analyzers for different dependency managers

### Key Types

#### Dependency

Represents a single dependency with its version information:

```go
type Dependency struct {
    Name    string // Name of the dependency package
    Version string // Currently specified version (e.g., "1.2.3", "^2.0.0", ">=1.0.0")
    Type    string // Type of dependency (e.g., "runtime", "dev", "optional")
    Source  string // Source/registry (e.g., "pypi", "npm", "rubygems")
}
```

#### DependencyFile

Represents a file that contains dependency information:

```go
type DependencyFile struct {
    Path     string // Full path to the dependency file in the repository
    Type     string // Type of dependency file (e.g., "poetry.lock", "package-lock.json")
    Analyzer string // Name of the analyzer that handles this file type
}
```

#### Config

Configuration for dependency analyzers:

```go
type Config struct {
    // RepositoryPaths is a list of paths within the repository to search
    // for dependency files. Empty list means search the entire repository.
    RepositoryPaths []string
    
    // RepositoryClient is the repository client implementation used to
    // fetch files from the repository
    RepositoryClient repository.Client
}
```

## Usage

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/greg-hellings/devdashboard/pkg/dependencies"
    "github.com/greg-hellings/devdashboard/pkg/repository"
)

func main() {
    // 1. Create a repository client
    repoConfig := repository.Config{
        Token: "your-github-token", // Optional for public repos
    }
    repoClient, err := repository.NewClient("github", repoConfig)
    if err != nil {
        log.Fatal(err)
    }
    
    // 2. Create a dependency analyzer
    analyzer, err := dependencies.NewAnalyzer("poetry")
    if err != nil {
        log.Fatal(err)
    }
    
    // 3. Configure the analyzer
    depConfig := dependencies.Config{
        RepositoryPaths:  []string{""}, // Empty string = search entire repo
        RepositoryClient: repoClient,
    }
    
    ctx := context.Background()
    
    // 4. Find candidate dependency files
    candidates, err := analyzer.CandidateFiles(
        ctx, 
        "python-poetry",  // owner
        "poetry",         // repo
        "master",         // ref (branch/tag/commit)
        depConfig,
    )
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Found %d dependency files\n", len(candidates))
    
    // 5. Analyze dependencies
    results, err := analyzer.AnalyzeDependencies(
        ctx,
        "python-poetry",
        "poetry",
        "master",
        candidates,
        depConfig,
    )
    if err != nil {
        log.Fatal(err)
    }
    
    // 6. Process results
    for filePath, deps := range results {
        fmt.Printf("\n%s:\n", filePath)
        for _, dep := range deps {
            fmt.Printf("  %s v%s (%s from %s)\n", 
                dep.Name, dep.Version, dep.Type, dep.Source)
        }
    }
}
```

### Searching Specific Paths

You can limit the search to specific directories within a repository:

```go
depConfig := dependencies.Config{
    RepositoryPaths:  []string{"src", "packages", "services"},
    RepositoryClient: repoClient,
}
```

### Using the Factory

```go
factory := dependencies.NewFactory()

// Create different analyzers
poetryAnalyzer, _ := factory.CreateAnalyzer("poetry")
// npmAnalyzer, _ := factory.CreateAnalyzer("npm")  // Future
```

### Two-Step Workflow

The analyzer interface provides two methods that can be used independently:

#### Step 1: Find Candidate Files

```go
candidates, err := analyzer.CandidateFiles(ctx, owner, repo, ref, config)
```

This searches the repository for files that match the analyzer's patterns. You can:
- Inspect the candidates
- Filter them
- Store them for later analysis
- Display them to users for selection

#### Step 2: Analyze Dependencies

```go
results, err := analyzer.AnalyzeDependencies(ctx, owner, repo, ref, candidates, config)
```

This analyzes the actual dependency information from the specified files. You can:
- Analyze all candidates
- Analyze a subset of files
- Analyze files from different searches
- Re-analyze files with different configurations

## Supported Analyzers

### Poetry (Python)

Analyzes Python Poetry lock files (`poetry.lock`).

**Analyzer Name:** `"poetry"`

**File Types:**
- `poetry.lock` - Poetry lock file

**Example:**

```go
analyzer, _ := dependencies.NewAnalyzer("poetry")
```

**Dependency Types:**
- `runtime` - Regular dependencies
- `dev` - Development dependencies
- `optional` - Optional dependencies

**Source:** `pypi`

**Example Output:**

```
Name: requests
Version: 2.31.0
Type: runtime
Source: pypi
```

### Future Analyzers

The following analyzers are planned for future releases:

- **npm** - Node.js package manager (`package-lock.json`, `yarn.lock`)
- **maven** - Java build tool (`pom.xml`)
- **gradle** - Java/Kotlin build tool (`build.gradle`)
- **cargo** - Rust package manager (`Cargo.lock`)
- **go-modules** - Go module system (`go.mod`, `go.sum`)
- **bundler** - Ruby package manager (`Gemfile.lock`)
- **composer** - PHP package manager (`composer.lock`)
- **nuget** - .NET package manager (`packages.lock.json`)

## Error Handling

### Partial Failures

The `AnalyzeDependencies` method is designed to be resilient to partial failures. If one file fails to parse, the analysis continues with other files:

```go
results, err := analyzer.AnalyzeDependencies(ctx, owner, repo, ref, files, config)
// err is nil even if some files failed
// Check map size vs input size to detect failures
if len(results) < len(files) {
    fmt.Printf("Warning: Only analyzed %d of %d files\n", len(results), len(files))
}
```

### Common Errors

**Missing Repository Client:**
```go
config := dependencies.Config{
    RepositoryPaths: []string{""},
    // RepositoryClient: nil  // Error!
}
// Error: "repository client is required"
```

**Invalid Analyzer Type:**
```go
analyzer, err := dependencies.NewAnalyzer("unsupported")
// Error: "unsupported analyzer type: unsupported (supported: poetry)"
```

**File Not Found:**
```go
// If a file doesn't exist or can't be accessed
// Error: "failed to get file content for poetry.lock: ..."
```

## Best Practices

### 1. Use Context with Timeouts

Always use contexts with timeouts for network operations:

```go
ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
defer cancel()

candidates, err := analyzer.CandidateFiles(ctx, owner, repo, ref, config)
```

### 2. Limit Search Scope

For large repositories, limit the search to relevant directories:

```go
config := dependencies.Config{
    RepositoryPaths: []string{"backend", "services"},
    RepositoryClient: repoClient,
}
```

### 3. Process Results Incrementally

For large result sets, process dependencies as you receive them:

```go
for filePath, deps := range results {
    processDependencies(filePath, deps)
}
```

### 4. Check for Empty Results

Always check if any files were found:

```go
if len(candidates) == 0 {
    fmt.Println("No dependency files found")
    return
}
```

### 5. Handle Different Dependency Types

Filter or categorize by dependency type:

```go
for _, dep := range deps {
    switch dep.Type {
    case "dev":
        // Handle development dependencies
    case "runtime":
        // Handle production dependencies
    case "optional":
        // Handle optional dependencies
    }
}
```

## Advanced Usage

### Analyzing Multiple Repositories

```go
repos := []struct {
    owner, repo, ref string
}{
    {"org1", "project1", "main"},
    {"org2", "project2", "master"},
}

for _, r := range repos {
    candidates, _ := analyzer.CandidateFiles(ctx, r.owner, r.repo, r.ref, config)
    results, _ := analyzer.AnalyzeDependencies(ctx, r.owner, r.repo, r.ref, candidates, config)
    
    fmt.Printf("\n%s/%s:\n", r.owner, r.repo)
    for filePath, deps := range results {
        fmt.Printf("  %s: %d dependencies\n", filePath, len(deps))
    }
}
```

### Dependency Statistics

```go
// Count dependencies by type
typeCount := make(map[string]int)
for _, deps := range results {
    for _, dep := range deps {
        typeCount[dep.Type]++
    }
}

// Count unique dependencies
uniqueDeps := make(map[string]bool)
for _, deps := range results {
    for _, dep := range deps {
        uniqueDeps[dep.Name] = true
    }
}

fmt.Printf("Total dependencies: %d\n", len(uniqueDeps))
for depType, count := range typeCount {
    fmt.Printf("  %s: %d\n", depType, count)
}
```

### Version Analysis

```go
// Find all versions of a specific package
targetPackage := "requests"
versions := make(map[string][]string) // version -> file paths

for filePath, deps := range results {
    for _, dep := range deps {
        if dep.Name == targetPackage {
            versions[dep.Version] = append(versions[dep.Version], filePath)
        }
    }
}

fmt.Printf("Versions of %s found:\n", targetPackage)
for version, files := range versions {
    fmt.Printf("  v%s in %d file(s)\n", version, len(files))
}
```

## Integration with Repository Module

The dependency module is tightly integrated with the repository module:

```go
// Both modules share the same repository client
repoClient, _ := repository.NewClient("github", repository.Config{
    Token: token,
})

// Repository operations
files, _ := repoClient.ListFilesRecursive(ctx, owner, repo, ref)
content, _ := repoClient.GetFileContent(ctx, owner, repo, ref, "poetry.lock")

// Dependency operations using the same client
depConfig := dependencies.Config{
    RepositoryClient: repoClient,  // Reuse the same client
}
```

## Testing

### Unit Tests

```go
func TestAnalyzer(t *testing.T) {
    analyzer := dependencies.NewPoetryAnalyzer()
    
    if analyzer.Name() != "poetry" {
        t.Errorf("Expected name 'poetry', got '%s'", analyzer.Name())
    }
}
```

### Integration Tests

For integration tests, you'll need access to actual repositories with dependency files.

## Extending the Module

### Adding a New Analyzer

To add support for a new dependency manager:

1. **Create the analyzer file** (e.g., `npm.go`):

```go
package dependencies

type NpmAnalyzer struct{}

func NewNpmAnalyzer() *NpmAnalyzer {
    return &NpmAnalyzer{}
}

func (n *NpmAnalyzer) Name() string {
    return "npm"
}

func (n *NpmAnalyzer) CandidateFiles(ctx context.Context, owner, repo, ref string, config Config) ([]DependencyFile, error) {
    // Implementation
}

func (n *NpmAnalyzer) AnalyzeDependencies(ctx context.Context, owner, repo, ref string, files []DependencyFile, config Config) (map[string][]Dependency, error) {
    // Implementation
}
```

2. **Add constant** in `dependencies.go`:

```go
const (
    AnalyzerPoetry AnalyzerType = "poetry"
    AnalyzerNpm    AnalyzerType = "npm"
)
```

3. **Update factory** in `factory.go`:

```go
func (f *Factory) CreateAnalyzer(analyzerType string) (Analyzer, error) {
    normalized := strings.ToLower(strings.TrimSpace(analyzerType))
    
    switch AnalyzerType(normalized) {
    case AnalyzerPoetry:
        return NewPoetryAnalyzer(), nil
    case AnalyzerNpm:
        return NewNpmAnalyzer(), nil
    default:
        return nil, fmt.Errorf("unsupported analyzer type: %s", analyzerType)
    }
}

func SupportedAnalyzers() []string {
    return []string{
        string(AnalyzerPoetry),
        string(AnalyzerNpm),
    }
}
```

4. **Add tests** in `npm_test.go`

5. **Update documentation**

## Performance Considerations

### File Searching

- `CandidateFiles` lists all files recursively, which can be slow for large repositories
- Use `RepositoryPaths` to limit search scope
- Results are cached by the repository client for the session

### File Parsing

- Each file requires a separate API call to fetch content
- Large lock files can take time to parse
- Consider implementing caching for repeated analyses

### Optimization Tips

```go
// 1. Limit the number of files analyzed
if len(candidates) > 10 {
    candidates = candidates[:10]  // Analyze first 10 only
}

// 2. Use shorter timeouts for quick checks
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

// 3. Search specific paths instead of entire repo
config := dependencies.Config{
    RepositoryPaths: []string{"src/main"},  // Not entire repo
}
```

## Troubleshooting

### No Files Found

**Problem:** `CandidateFiles` returns empty slice

**Solutions:**
- Verify the repository actually contains dependency files
- Check that `RepositoryPaths` includes the correct directories
- Ensure you have access to the repository (for private repos)
- Verify the ref (branch/tag) exists and is spelled correctly

### Parse Errors

**Problem:** `AnalyzeDependencies` fails to parse files

**Solutions:**
- Check that the file format is valid
- Verify you're using the correct analyzer type
- Ensure the file is not corrupted or empty
- Check for encoding issues

### Performance Issues

**Problem:** Analysis takes too long

**Solutions:**
- Reduce search scope with `RepositoryPaths`
- Limit number of files analyzed
- Use shorter context timeouts
- Analyze files in parallel (custom implementation)

## Examples

See `examples/dependency_analysis.go` for complete working examples including:
- Basic Poetry analysis
- Finding dependency files
- Using environment variables for configuration
- Processing and displaying results

## Related Documentation

- [README.md](README.md) - Main project documentation
- [ARCHITECTURE.md](ARCHITECTURE.md) - Design decisions and patterns
- [Repository Module](pkg/repository/repository.go) - Repository interface documentation