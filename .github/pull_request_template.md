# Pull Request

## Description

<!-- Provide a clear and concise description of what this PR does -->

## Type of Change

<!-- Mark the relevant option(s) with an 'x' -->

- [ ] Bug fix (non-breaking change which fixes an issue)
- [ ] New feature (non-breaking change which adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update
- [ ] Refactoring (no functional changes)
- [ ] Performance improvement
- [ ] Test improvement
- [ ] CI/CD improvement
- [ ] Dependency update
- [ ] Other (please describe):

## Related Issues

<!-- Link any related issues here using #issue_number -->

Fixes #
Relates to #

## Changes Made

<!-- List the main changes made in this PR -->

-
-
-

## Testing

<!-- Describe the tests you ran and how to reproduce them -->

### Test Configuration

- **Go version**:
- **OS**:
- **Installation method**: (source/nix/docker)

### Tests Performed

- [ ] Unit tests pass (`make test` or `go test ./...`)
- [ ] Integration tests pass (if applicable)
- [ ] Manual testing performed
- [ ] Nix build succeeds (`nix build`)
- [ ] Nix checks pass (`nix flake check`)

### Test Commands

```bash
# Commands used to test this PR
make test
make build
./devdashboard help
```

### Test Results

<!-- Paste relevant test output or screenshots -->

```
# Test output here
```

## Documentation

<!-- Mark the relevant option(s) with an 'x' -->

- [ ] Documentation has been updated (if needed)
- [ ] README.md updated (if needed)
- [ ] CHANGELOG.md updated
- [ ] Code comments added/updated for complex logic
- [ ] Examples updated (if needed)
- [ ] No documentation changes needed

## Checklist

<!-- Mark completed items with an 'x' -->

### Code Quality

- [ ] My code follows the project's style guidelines
- [ ] I have performed a self-review of my code
- [ ] I have commented my code, particularly in hard-to-understand areas
- [ ] I have made corresponding changes to the documentation
- [ ] My changes generate no new warnings
- [ ] I have added tests that prove my fix is effective or that my feature works
- [ ] New and existing unit tests pass locally with my changes
- [ ] Any dependent changes have been merged and published

### Go Specific

- [ ] Code is properly formatted (`make fmt` or `go fmt`)
- [ ] `go vet` passes with no warnings
- [ ] `go mod tidy` has been run
- [ ] No new dependencies added (or justified if added)
- [ ] All exported functions/types have documentation comments

### Nix Specific (if applicable)

- [ ] `flake.nix` updated (if dependencies changed)
- [ ] `vendorHash` updated (if go.mod/go.sum changed)
- [ ] Nix flake checks pass (`nix flake check`)
- [ ] Pre-commit hooks pass

### Security

- [ ] No sensitive information (tokens, passwords, keys) included
- [ ] Dependencies scanned for vulnerabilities
- [ ] Input validation added (if handling user input)
- [ ] Error messages don't leak sensitive information

## Breaking Changes

<!-- If this PR includes breaking changes, describe them here and provide migration instructions -->

### Description of Breaking Changes

<!-- What will break? -->

### Migration Guide

<!-- How should users update their code/config? -->

```yaml
# Before
old-configuration: value

# After
new-configuration: value
```

## Performance Impact

<!-- Describe any performance implications -->

- [ ] No performance impact
- [ ] Performance improved
- [ ] Performance degraded (justified because...)
- [ ] Performance benchmarks run (attach results if significant)

## Screenshots/Recordings

<!-- If applicable, add screenshots or recordings to demonstrate changes -->

## Additional Notes

<!-- Any additional information that reviewers should know -->

## Reviewer Notes

<!-- Specific areas you'd like reviewers to focus on -->

**Focus areas:**
-
-

**Questions for reviewers:**
-
-

---

## Post-Merge Tasks

<!-- Tasks to complete after merging (if any) -->

- [ ] Update related documentation
- [ ] Announce breaking changes
- [ ] Update dependent projects
- [ ] Create follow-up issues (if needed)
