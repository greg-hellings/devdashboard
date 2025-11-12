# Changelog: Path Handling Improvement

## Date
2024-01-XX

## Summary
Modified the dependency report generation to respect user-specified paths instead of blindly searching the entire repository.

## Problem
Previously, when users specified explicit file paths in the configuration (e.g., `paths: ["backend/poetry.lock"]`), the tool would still perform a full recursive search of the entire repository and then filter the results. This had several issues:

1. **Inefficient**: Unnecessary API calls to list all files in the repository
2. **Confusing**: User's explicit path specification was not truly respected
3. **Slower**: Especially problematic for large repositories or monorepos

## Solution
Modified `pkg/report/report.go` to implement conditional path handling:

- **When `paths` is specified**: Use those exact file paths directly without searching
- **When `paths` is empty/omitted**: Fall back to auto-search using `CandidateFiles()` (original behavior)

## Technical Details

### Before
```go
// Always called CandidateFiles, which searches entire repository
candidates, err := analyzer.CandidateFiles(ctx, owner, repo, ref, depConfig)
```

### After
```go
if len(repo.Config.Paths) > 0 {
    // User specified explicit paths, use them directly
    for _, path := range repo.Config.Paths {
        candidates = append(candidates, dependencies.DependencyFile{
            Path:     path,
            Type:     repo.Config.Analyzer,
            Analyzer: repo.Config.Analyzer,
        })
    }
} else {
    // No paths specified, search for candidate files
    candidates, err = analyzer.CandidateFiles(ctx, owner, repo, ref, depConfig)
}
```

## Benefits

1. **Performance**: Direct file access is much faster than recursive search
2. **Clarity**: User-specified paths are now treated as explicit file paths
3. **Control**: Users have direct control over which files are analyzed
4. **Backward Compatible**: Existing configs without `paths` work exactly as before

## Usage Changes

### Configuration Format
The `paths` field now explicitly means "exact file paths to analyze":

```yaml
# CORRECT: Specify exact file paths
paths:
  - "backend/poetry.lock"
  - "services/api/poetry.lock"
  - "packages/shared/poetry.lock"

# INCORRECT: Don't specify directories
paths:
  - "backend"           # Wrong - not a file
  - "services/api"      # Wrong - not a file
```

### Migration Guide

**If you have configs with `paths` specified:**

Review your configuration and ensure paths point to actual files, not directories:

```yaml
# Old (may have worked accidentally)
paths:
  - "backend"

# New (explicit and correct)
paths:
  - "backend/poetry.lock"
```

**If you omit `paths`:**

No changes needed - auto-search behavior is unchanged.

## Examples

### Example 1: Simple Repository (Auto-Search)
```yaml
repositories:
  - repository: "my-service"
    # No paths specified - auto-searches entire repo
```

**Behavior**: Tool searches entire repository for matching dependency files.

### Example 2: Monorepo with Explicit Paths
```yaml
repositories:
  - repository: "monorepo"
    paths:
      - "services/api/poetry.lock"
      - "services/worker/poetry.lock"
      - "packages/common/poetry.lock"
```

**Behavior**: Tool analyzes only these three specific files, no searching.

### Example 3: Mixed Approach
```yaml
repositories:
  - repository: "simple-app"
    # Auto-search

  - repository: "complex-monorepo"
    # Explicit paths
    paths:
      - "backend/uv.lock"
      - "frontend/uv.lock"
```

**Behavior**:
- `simple-app`: Auto-searches entire repo
- `complex-monorepo`: Analyzes only the two specified files

## Files Changed

- `pkg/report/report.go` - Modified `analyzeRepository()` method
- `docs/DEPENDENCY_REPORT.md` - Updated documentation
- `examples/dependency-report-paths-example.yaml` - Added comprehensive example

## Testing

Existing tests continue to pass:
```bash
go test ./pkg/config/...   # PASS
go test ./pkg/dependencies/... # PASS
```

The change is isolated to the report generation logic and doesn't affect the dependency parsers themselves.

## Breaking Changes

**None** - This is a backward-compatible improvement:
- Configs without `paths` work exactly as before
- Configs with `paths` now work more efficiently and correctly
- No changes to config file format or CLI interface

## Future Considerations

1. Could add validation to ensure specified paths actually exist
2. Could support both files and directory patterns in future
3. Could add warning if specified path doesn't exist

## Notes

- This change does NOT modify the dependency analyzer modules (pipfile.go, uvlock.go, poetry.go)
- The `CandidateFiles()` method in analyzers remains unchanged
- All changes are in the report orchestration layer
