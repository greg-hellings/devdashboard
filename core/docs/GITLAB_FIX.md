# GitLab Base64 Decoding Fix

## Issue

The GitLab API returns file content as base64-encoded strings in the `Content` field of the response. The initial implementation of `GetFileContent()` for GitLab incorrectly assumed that the go-gitlab library automatically decoded this content, but it does not.

## Problem Description

When retrieving file content from GitLab repositories using the `GetFileContent()` method, the returned string was base64-encoded rather than plain text. This caused issues when:

- Parsing dependency files (e.g., poetry.lock, package.json)
- Displaying file content to users
- Processing text-based files

## Root Cause

The GitLab API response includes file content in the `Content` field as a base64-encoded string. The go-gitlab library (`github.com/xanzy/go-gitlab`) does **not** automatically decode this content, unlike what was initially assumed.

## Solution

Added explicit base64 decoding to the `GetFileContent()` method in `pkg/repository/gitlab.go`:

```go
// GitLab returns base64 encoded content in the Content field
// We need to decode it manually
if file.Content == "" {
    return "", fmt.Errorf("file content is empty: %s", path)
}

// Decode base64 content
decodedContent, err := base64.StdEncoding.DecodeString(file.Content)
if err != nil {
    return "", fmt.Errorf("failed to decode base64 content: %w", err)
}

return string(decodedContent), nil
```

## Changes Made

### Code Changes

**File:** `pkg/repository/gitlab.go`

1. Added import: `encoding/base64`
2. Updated `GetFileContent()` method to decode base64 content
3. Added error handling for decoding failures
4. Updated comments to reflect actual behavior

### Test Coverage

**File:** `pkg/repository/gitlab_test.go` (new)

Added comprehensive unit tests:
- `TestBase64Decoding` - Basic decoding functionality
- `TestBase64DecodingMultiline` - Multi-line content (code files)
- `TestBase64DecodingUnicode` - Unicode characters
- `TestBase64DecodingEmpty` - Empty content edge case
- `TestNewGitLabClient` - Client creation with config
- `TestNewGitLabClientWithoutToken` - Client creation without auth

All tests pass successfully.

## Verification

### Manual Testing

Tested with real GitLab repositories:
```bash
export REPO_PROVIDER=gitlab
export REPO_OWNER=gitlab-org
export REPO_NAME=gitlab-foss
./bin/devdashboard repo-info
```

### Automated Testing

```bash
go test ./pkg/repository -v
# All tests pass (20 tests total)
```

## Impact

### Before Fix
- GitLab file content returned as base64 strings
- Dependency analysis would fail on GitLab repositories
- File content was unreadable

### After Fix
- GitLab file content properly decoded to plain text
- Dependency analysis works with GitLab repositories
- File content matches GitHub behavior
- Consistent interface across providers

## Comparison with GitHub

The GitHub API client (`go-github`) **does** automatically decode base64 content, which is why the GitHub implementation worked correctly without explicit decoding. The GitLab implementation now matches this behavior by manually performing the decoding step.

## Future Considerations

1. **Error Handling**: The fix includes proper error handling for invalid base64 content
2. **Performance**: Base64 decoding is fast and has negligible performance impact
3. **Encoding**: All text content is decoded to UTF-8 strings
4. **Binary Files**: Base64 decoding works for binary content, but the return type is string

## Breaking Changes

None. This is a bug fix that corrects the behavior to match the documented interface. Any code relying on the old (incorrect) base64-encoded output was likely broken anyway.

## Related Files

- `pkg/repository/gitlab.go` - Implementation fix
- `pkg/repository/gitlab_test.go` - New test file
- `pkg/repository/repository.go` - Interface definition (unchanged)
- `pkg/repository/github.go` - Reference implementation (unchanged)
- `CHANGELOG.md` - Documented in Fixed section

## Testing Recommendations

When testing GitLab integration:

1. Test with various file types (text, JSON, TOML, YAML)
2. Test with unicode content
3. Test with empty files
4. Test with large files
5. Verify dependency analysis works end-to-end

## References

- [GitLab API Documentation](https://docs.gitlab.com/ee/api/repository_files.html)
- [go-gitlab Library](https://github.com/xanzy/go-gitlab)
- [Go base64 Package](https://pkg.go.dev/encoding/base64)

## Resolution Date

2024-01-XX

## Status

✅ Fixed and Tested
