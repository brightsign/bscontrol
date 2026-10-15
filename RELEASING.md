# Release Process Quick Reference

## Creating a New Release

### 1. Ensure Everything is Ready

```bash
# Make sure all tests pass
go test ./...

# Make sure build works
go build ./...

# Check for uncommitted changes
git status
```

### 2. Run the Release Script

```bash
./scripts/release.sh <version> "<release-name>"
```

**Examples:**

```bash
# Feature release
./scripts/release.sh v1.2.0 "Type Safety Improvements"

# Bug fix release
./scripts/release.sh v1.1.1 "Authentication Timeout Fix"

# Beta release
./scripts/release.sh v1.3.0-beta.1 "Beta: New RDWS Features"
```

### 3. The Script Will:

1. Validate version format
2. Check for uncommitted changes
3. Run full test suite
4. Verify build succeeds
5. Create annotated git tag
6. Push tag to GitHub
7. Generate release notes from commits
8. Create GitHub release

### 4. Verify Release

```bash
# Check the release was created
gh release view v1.2.0

# View release URL
gh release view v1.2.0 --json url -q .url
```

## Version Guidelines

### Semantic Versioning

- **v1.0.0 → v2.0.0** - Breaking changes
- **v1.0.0 → v1.1.0** - New features (backward compatible)
- **v1.0.0 → v1.0.1** - Bug fixes only

### What Requires a Version Bump?

**Major (v2.0.0):**
- Removing exported functions/types
- Changing function signatures
- Breaking API changes

**Minor (v1.1.0):**
- Adding new exported types (like our type safety update)
- Adding new methods to existing types
- New features that don't break existing code

**Patch (v1.0.1):**
- Bug fixes
- Documentation updates
- Performance improvements
- Internal refactoring

## User Instructions

Once released, users can install with:

```bash
# Specific version (recommended)
go get github.com/brightsign/bscontrol@v1.2.0

# Latest version
go get -u github.com/brightsign/bscontrol
```

In go.mod:
```go
require github.com/brightsign/bscontrol v1.2.0
```

## Rollback a Release

If you need to remove a bad release:

```bash
# Delete GitHub release
gh release delete v1.2.0

# Delete local tag
git tag -d v1.2.0

# Delete remote tag
git push --delete origin v1.2.0
```

## Troubleshooting

**"Tag already exists"**
- Tag already pushed to GitHub
- Use a different version number or delete the existing tag first

**"Not authenticated with GitHub CLI"**
```bash
gh auth login
```

**"Tests failed"**
- Fix failing tests before releasing
- Run `go test -v ./...` to see details

**"Working directory has uncommitted changes"**
- Commit or stash changes first
- Or answer 'y' to continue anyway (not recommended)

## Pre-release Testing

For beta/alpha releases:

```bash
./scripts/release.sh v1.3.0-beta.1 "Beta: New Features"
```

This marks the release as "pre-release" in GitHub and users can test:

```bash
go get github.com/brightsign/bscontrol@v1.3.0-beta.1
```

## Documentation

Full versioning documentation: [docs/versioning.md](docs/versioning.md)
