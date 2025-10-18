# Paths Configuration - Quick Reference

## TL;DR

- **No `paths` field** → Auto-searches entire repository
- **With `paths` field** → Uses only those exact file paths (no search)

## Syntax

### Auto-Search (Recommended for simple repos)
```yaml
repositories:
  - repository: "my-service"
    # Omit 'paths' field entirely
```

### Explicit Paths (Recommended for monorepos)
```yaml
repositories:
  - repository: "monorepo"
    paths:
      - "backend/poetry.lock"      # ✅ Exact file path
      - "frontend/uv.lock"          # ✅ Exact file path
```

## Common Mistakes

### ❌ Wrong: Directory paths
```yaml
paths:
  - "backend"           # Wrong - not a file
  - "services/api"      # Wrong - not a file
  - "src/"              # Wrong - not a file
```

### ✅ Correct: File paths
```yaml
paths:
  - "backend/poetry.lock"
  - "services/api/uv.lock"
  - "src/Pipfile.lock"
```

## Decision Tree

```
Do you know the exact location of your dependency files?
│
├─ YES → Specify exact paths
│         └─ Benefits: Faster, more efficient, fewer API calls
│
└─ NO  → Omit paths field
          └─ Benefits: Automatic discovery, convenient
```

## Examples

### Single File in Root
```yaml
repositories:
  - repository: "simple-app"
    paths:
      - "poetry.lock"
```

### Multiple Files in Monorepo
```yaml
repositories:
  - repository: "platform"
    paths:
      - "services/api/poetry.lock"
      - "services/worker/poetry.lock"
      - "services/scheduler/poetry.lock"
      - "packages/common/poetry.lock"
```

### Different Analyzers
```yaml
repositories:
  - repository: "poetry-project"
    analyzer: "poetry"
    paths:
      - "poetry.lock"

  - repository: "pipenv-project"
    analyzer: "pipfile"
    paths:
      - "Pipfile.lock"

  - repository: "uv-project"
    analyzer: "uvlock"
    paths:
      - "uv.lock"
```

### Mixed Auto-Search and Explicit
```yaml
repositories:
  # Auto-search
  - repository: "small-service"

  # Explicit paths
  - repository: "large-monorepo"
    paths:
      - "backend/poetry.lock"
      - "frontend/uv.lock"
```

## Comparison Table

| Aspect | Auto-Search (no paths) | Explicit Paths (with paths) |
|--------|------------------------|------------------------------|
| Speed | Slower | Faster |
| API Calls | More | Fewer |
| When to Use | Unknown file locations | Known file locations |
| Monorepo Support | Finds all files | Specify which files |
| Configuration | Simpler | More explicit |

## Performance Impact

### Small Repository (< 100 files)
- **Auto-search**: ~1-2 seconds
- **Explicit paths**: ~0.5-1 second
- **Difference**: Minimal

### Large Monorepo (> 10,000 files)
- **Auto-search**: ~5-10 seconds
- **Explicit paths**: ~1 second
- **Difference**: 5-10x faster

## Rules to Remember

1. **Paths are file paths, not directories**
2. **Paths are relative to repository root** (no leading `/`)
3. **Empty `paths: []` is different from omitting `paths`**
   - Empty list → tries to use paths but finds none
   - Omitted → triggers auto-search
4. **Each path should end with the lock file name**
   - `poetry.lock`, `Pipfile.lock`, or `uv.lock`

## Troubleshooting

### "No dependency files found"
- ✅ Check file path is correct
- ✅ Ensure file exists in repository
- ✅ Verify path is relative to repo root
- ✅ Make sure path includes filename

### Analysis is slow
- ✅ Use explicit paths instead of auto-search
- ✅ Specify only files you need to analyze

### File not being analyzed
- ✅ If using `paths`, ensure file is in the list
- ✅ If omitting `paths`, ensure file name matches analyzer type

## See Also

- [Full Documentation](DEPENDENCY_REPORT.md)
- [Examples](../examples/dependency-report-paths-example.yaml)
- [Path Handling Details](PATH_HANDLING_IMPROVEMENT.md)
