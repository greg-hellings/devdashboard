# Dependency Analysis Implementation Guide

This document provides a detailed guide for implementing new dependency analyzers in the DevDashboard project.

## Overview

The dependency analysis module is designed to be easily extensible. Adding support for a new dependency manager (npm, Maven, Gradle, etc.) follows a consistent pattern that integrates seamlessly with the existing architecture.

## Implementation Checklist

When implementing a new dependency analyzer, follow these steps:

- [ ] Create analyzer implementation file
- [ ] Define file type constants
- [ ] Implement the `Analyzer` interface
- [ ] Add parser logic for the dependency file format
- [ ] Update factory to recognize new analyzer
- [ ] Add analyzer type constant
- [ ] Create unit tests
- [ ] Update documentation
- [ ] Add example usage

## Step-by-Step Implementation

### Step 1: Create Analyzer File

Create a new file in `pkg/dependencies/` named after your dependency manager (e.g., `npm.go`, `maven.go`).

```go
package dependencies

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"
)

// NpmAnalyzer implements the Analyzer interface for Node.js npm projects
type NpmAnalyzer struct{}

// NewNpmAnalyzer creates a new npm dependency analyzer
func NewNpmAnalyzer() *NpmAnalyzer {
    return &NpmAnalyzer{}
}
```

### Step 2: Implement Name Method

```go
// Name returns the name of this analyzer
func (n *NpmAnalyzer) Name() string {
    return string(AnalyzerNpm) // We'll define this constant later
}
```

### Step 3: Implement CandidateFiles Method

This method searches for files that your analyzer can process:

```go
// CandidateFiles searches for package-lock.json files in the configured repository paths
func (n *NpmAnalyzer) CandidateFiles(ctx context.Context, owner, repo, ref string, config Config) ([]DependencyFile, error) {
    if config.RepositoryClient == nil {
        return nil, fmt.Errorf("repository client is required")
    }

    var candidates []DependencyFile
    searchPaths := config.RepositoryPaths

    // If no paths specified, search from root
    if len(searchPaths) == 0 {
        searchPaths = []string{""}
    }

    // Search each configured path
    for _, searchPath := range searchPaths {
        // List all files recursively
        files, err := config.RepositoryClient.ListFilesRecursive(ctx, owner, repo, ref)
        if err != nil {
            return nil, fmt.Errorf("failed to list files: %w", err)
        }

        // Filter for npm lock files
        for _, file := range files {
            if file.Type != "file" {
                continue
            }

            // Check for package-lock.json or yarn.lock
            isNpmLock := strings.HasSuffix(file.Path, "package-lock.json")
            isYarnLock := strings.HasSuffix(file.Path, "yarn.lock")
            
            if isNpmLock || isYarnLock {
                // If searchPath is specified, ensure file is within that path
                if searchPath != "" && !strings.HasPrefix(file.Path, searchPath) {
                    continue
                }

                fileType := "package-lock.json"
                if isYarnLock {
                    fileType = "yarn.lock"
                }

                candidates = append(candidates, DependencyFile{
                    Path:     file.Path,
                    Type:     fileType,
                    Analyzer: n.Name(),
                })
            }
        }
    }

    return candidates, nil
}
```

### Step 4: Implement AnalyzeDependencies Method

This method parses the dependency files:

```go
// AnalyzeDependencies analyzes npm/yarn lock files and extracts dependency information
func (n *NpmAnalyzer) AnalyzeDependencies(ctx context.Context, owner, repo, ref string, files []DependencyFile, config Config) (map[string][]Dependency, error) {
    if config.RepositoryClient == nil {
        return nil, fmt.Errorf("repository client is required")
    }

    result := make(map[string][]Dependency)

    for _, file := range files {
        deps, err := n.analyzeFile(ctx, owner, repo, ref, file, config)
        if err != nil {
            // Don't fail completely if one file fails
            continue
        }
        result[file.Path] = deps
    }

    return result, nil
}

// analyzeFile analyzes a single npm/yarn lock file
func (n *NpmAnalyzer) analyzeFile(ctx context.Context, owner, repo, ref string, file DependencyFile, config Config) ([]Dependency, error) {
    // Get the file content
    content, err := config.RepositoryClient.GetFileContent(ctx, owner, repo, ref, file.Path)
    if err != nil {
        return nil, fmt.Errorf("failed to get file content for %s: %w", file.Path, err)
    }

    // Parse based on file type
    var dependencies []Dependency
    switch file.Type {
    case "package-lock.json":
        dependencies, err = n.parsePackageLock(content)
    case "yarn.lock":
        dependencies, err = n.parseYarnLock(content)
    default:
        return nil, fmt.Errorf("unsupported file type: %s", file.Type)
    }

    if err != nil {
        return nil, fmt.Errorf("failed to parse %s: %w", file.Path, err)
    }

    return dependencies, nil
}
```

### Step 5: Implement Parser Logic

Define structures and parsing logic for your dependency file format:

```go
// packageLockFile represents the structure of package-lock.json
type packageLockFile struct {
    Name         string                       `json:"name"`
    Version      string                       `json:"version"`
    Dependencies map[string]packageLockEntry `json:"dependencies"`
}

type packageLockEntry struct {
    Version  string `json:"version"`
    Dev      bool   `json:"dev"`
    Optional bool   `json:"optional"`
}

// parsePackageLock parses a package-lock.json file
func (n *NpmAnalyzer) parsePackageLock(content string) ([]Dependency, error) {
    var lockFile packageLockFile

    if err := json.Unmarshal([]byte(content), &lockFile); err != nil {
        return nil, fmt.Errorf("failed to parse package-lock.json: %w", err)
    }

    dependencies := make([]Dependency, 0, len(lockFile.Dependencies))

    for name, entry := range lockFile.Dependencies {
        depType := "runtime"
        if entry.Dev {
            depType = "dev"
        }
        if entry.Optional {
            depType = "optional"
        }

        dep := Dependency{
            Name:    name,
            Version: entry.Version,
            Type:    depType,
            Source:  "npm",
        }

        dependencies = append(dependencies, dep)
    }

    return dependencies, nil
}

// parseYarnLock parses a yarn.lock file
func (n *NpmAnalyzer) parseYarnLock(content string) ([]Dependency, error) {
    // Yarn lock files use a custom format
    // Implementation depends on parsing library or custom parser
    // This is a simplified example
    return nil, fmt.Errorf("yarn.lock parsing not yet implemented")
}
```

### Step 6: Update Factory

Add your analyzer to `pkg/dependencies/factory.go`:

```go
// In dependencies.go, add constant:
const (
    AnalyzerPoetry AnalyzerType = "poetry"
    AnalyzerNpm    AnalyzerType = "npm"
)

// In factory.go, update CreateAnalyzer:
func (f *Factory) CreateAnalyzer(analyzerType string) (Analyzer, error) {
    normalized := strings.ToLower(strings.TrimSpace(analyzerType))

    switch AnalyzerType(normalized) {
    case AnalyzerPoetry:
        return NewPoetryAnalyzer(), nil
    case AnalyzerNpm:
        return NewNpmAnalyzer(), nil
    default:
        return nil, fmt.Errorf("unsupported analyzer type: %s (supported: poetry, npm)", analyzerType)
    }
}

// Update SupportedAnalyzers:
func SupportedAnalyzers() []string {
    return []string{
        string(AnalyzerPoetry),
        string(AnalyzerNpm),
    }
}
```

### Step 7: Create Unit Tests

Create `pkg/dependencies/npm_test.go`:

```go
package dependencies

import "testing"

func TestNpmAnalyzerName(t *testing.T) {
    analyzer := NewNpmAnalyzer()
    
    if analyzer.Name() != "npm" {
        t.Errorf("Expected name 'npm', got '%s'", analyzer.Name())
    }
}

func TestNewNpmAnalyzer(t *testing.T) {
    analyzer := NewNpmAnalyzer()
    
    if analyzer == nil {
        t.Fatal("NewNpmAnalyzer returned nil")
    }
}

// Add more tests for parsing logic, file discovery, etc.
```

### Step 8: Update Documentation

Update the following files:

**README.md:**
```markdown
### Supported Dependency Managers
- **Poetry (Python)**: `poetry.lock` files
- **npm (Node.js)**: `package-lock.json`, `yarn.lock` files
```

**DEPENDENCIES.md:**
```markdown
### npm (Node.js)

Analyzes Node.js package manager lock files.

**Analyzer Name:** `"npm"`

**File Types:**
- `package-lock.json` - npm lock file
- `yarn.lock` - Yarn lock file

**Example:**
```go
analyzer, _ := dependencies.NewAnalyzer("npm")
```
```

**CHANGELOG.md:**
```markdown
### Added
- npm dependency analyzer
  - Support for package-lock.json
  - Support for yarn.lock
  - Extract npm package dependencies
```

## Best Practices

### 1. Error Handling

- Don't fail the entire analysis if one file fails to parse
- Return partial results when possible
- Use descriptive error messages

```go
deps, err := n.analyzeFile(ctx, owner, repo, ref, file, config)
if err != nil {
    // Log but continue with other files
    continue
}
```

### 2. Performance

- Minimize API calls to the repository
- Parse files efficiently
- Use streaming for large files if necessary

### 3. Dependency Types

Normalize dependency types across analyzers:
- `runtime` - Production dependencies
- `dev` - Development dependencies
- `optional` - Optional dependencies
- `peer` - Peer dependencies (if applicable)

### 4. Version Formats

Preserve the original version specification:
- `"1.2.3"` - Exact version
- `"^1.2.3"` - Compatible version
- `"~1.2.3"` - Approximately equivalent
- `">=1.2.3"` - Range

### 5. Testing

Test with real-world files:
- Include sample lock files in test fixtures
- Test edge cases (empty files, malformed files)
- Test different versions of the lock file format

## Common Patterns

### Pattern 1: Multiple File Types

If your analyzer supports multiple file types:

```go
func (a *Analyzer) CandidateFiles(ctx context.Context, ...) ([]DependencyFile, error) {
    filePatterns := []string{
        "package-lock.json",
        "yarn.lock",
        "pnpm-lock.yaml",
    }
    
    for _, pattern := range filePatterns {
        // Search for each pattern
    }
}
```

### Pattern 2: Nested Dependencies

For lock files with nested dependency trees:

```go
func flattenDependencies(deps map[string]Entry, prefix string) []Dependency {
    var result []Dependency
    for name, entry := range deps {
        result = append(result, convertToDependency(name, entry))
        // Recursively process nested dependencies
        if len(entry.Dependencies) > 0 {
            result = append(result, flattenDependencies(entry.Dependencies, prefix+name+"/")...)
        }
    }
    return result
}
```

### Pattern 3: Version Resolution

For files that contain version ranges or resolution:

```go
type Dependency struct {
    Name            string
    Version         string // Requested version
    ResolvedVersion string // Actual resolved version
    Type            string
    Source          string
}
```

## External Dependencies

When adding parsing libraries, update `go.mod`:

```bash
go get github.com/example/parser-library
```

Common parsing libraries:
- JSON: `encoding/json` (built-in)
- YAML: `gopkg.in/yaml.v3`
- TOML: `github.com/BurntSushi/toml`
- XML: `encoding/xml` (built-in)

## Examples

See existing implementations:
- `pkg/dependencies/poetry.go` - Complete Python Poetry implementation
- Uses TOML parsing
- Handles multiple dependency types
- Robust error handling

## Testing Your Implementation

```bash
# Run unit tests
go test ./pkg/dependencies -v

# Build example program
go build -o bin/dependency_analysis ./examples/dependency_analysis.go

# Test with a real repository
export REPO_PROVIDER=github
export REPO_OWNER=owner
export REPO_NAME=repo
export ANALYZER_TYPE=npm
./bin/dependency_analysis
```

## Common Issues and Solutions

### Issue: Large Lock Files

**Problem:** Lock files with thousands of dependencies are slow to parse

**Solution:** 
- Stream the file content
- Parse incrementally
- Consider pagination or limiting results

### Issue: Different File Formats

**Problem:** Lock file format changed between versions

**Solution:**
- Detect version from file metadata
- Support multiple format versions
- Fail gracefully with unsupported versions

### Issue: Missing Dependencies

**Problem:** Not all dependencies are captured

**Solution:**
- Check for nested dependencies
- Look for dependency groups (dev, peer, optional)
- Verify file format documentation

## Contribution Guidelines

When submitting a new analyzer:

1. Follow the established code structure
2. Include comprehensive tests
3. Update all documentation
4. Add examples to the example programs
5. Ensure code passes all linters
6. Test with real-world repositories

## Resources

- [Poetry Lock Format](https://python-poetry.org/docs/basic-usage/#installing-with-poetrylock)
- [npm package-lock.json](https://docs.npmjs.com/cli/v9/configuring-npm/package-lock-json)
- [Yarn Lock Format](https://classic.yarnpkg.com/lang/en/docs/yarn-lock/)
- [Maven POM](https://maven.apache.org/pom.html)
- [Cargo.lock Format](https://doc.rust-lang.org/cargo/guide/cargo-toml-vs-cargo-lock.html)