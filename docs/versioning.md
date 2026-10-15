# Version Management for bscontrol SDK

## For Maintainers: Creating a Release

### Prerequisites

1. **GitHub CLI** (`gh`) installed and authenticated
   ```bash
   brew install gh
   gh auth login
   ```

2. **Clean working directory** - commit or stash all changes

3. **All tests passing**
   ```bash
   go test ./...
   ```

### Release Process

Use the release script:

```bash
./scripts/release.sh <version> "<release-name>"
```

**Examples:**

```bash
# Standard release
./scripts/release.sh v1.2.0 "Type Safety Release"

# Pre-release
./scripts/release.sh v1.3.0-beta.1 "Beta Release"

# Patch release
./scripts/release.sh v1.2.1 "Bug Fix Release"
```

### What the Script Does

1. ✅ Validates version format (v<major>.<minor>.<patch>)
2. ✅ Checks for uncommitted changes
3. ✅ Runs full test suite
4. ✅ Verifies build succeeds
5. ✅ Creates annotated git tag
6. ✅ Pushes tag to GitHub
7. ✅ Generates release notes from commit history
8. ✅ Creates GitHub release with usage instructions

### Semantic Versioning

Follow [Semantic Versioning 2.0.0](https://semver.org/):

- **Major** (v2.0.0) - Breaking changes to public API
- **Minor** (v1.3.0) - New features, backward compatible
- **Patch** (v1.2.1) - Bug fixes, backward compatible
- **Pre-release** (v1.3.0-beta.1) - Beta/alpha releases

**Examples:**

```bash
# Adding new exported types (backward compatible)
v1.1.0 "Added device status type exports"

# Bug fix only
v1.0.1 "Fixed authentication timeout"

# Breaking API change
v2.0.0 "Renamed Client.Auth to Client.Authenticate"

# Pre-release testing
v1.2.0-beta.1 "Beta: New RDWS features"
```

### Release Checklist

Before releasing:

- [ ] All tests pass: `go test ./...`
- [ ] Documentation updated (README, type-safety.md, etc.)
- [ ] CHANGELOG updated (if maintained)
- [ ] Examples tested with new changes
- [ ] Breaking changes documented (if v2.x.x)
- [ ] Review `git log` for commit quality
- [ ] Verify go.mod dependencies are current

---

## For Users: Using Versioned Releases

### Recommended: Pin to Specific Versions

Always use specific version tags in production:

```bash
# Install specific version
go get github.com/brightsign/bscontrol@v1.2.0
```

### In go.mod

**Explicit version (recommended):**
```go
module your-app

go 1.24

require (
    github.com/brightsign/bscontrol v1.2.0
)
```

**Latest patch (auto-updates patches):**
```go
require (
    github.com/brightsign/bscontrol v1.2
)
```

**Latest minor (auto-updates features):**
```go
require (
    github.com/brightsign/bscontrol v1
)
```

**Not recommended for production:**
```go
// This uses the latest commit on main branch
require (
    github.com/brightsign/bscontrol latest
)
```

### Updating Versions

```bash
# Update to specific version
go get github.com/brightsign/bscontrol@v1.3.0

# Update to latest version
go get -u github.com/brightsign/bscontrol

# Update to latest patch only (safer)
go get -u=patch github.com/brightsign/bscontrol
```

### Checking Current Version

```bash
# View installed version
go list -m github.com/brightsign/bscontrol

# View available versions
go list -m -versions github.com/brightsign/bscontrol

# View module details
go list -m -json github.com/brightsign/bscontrol
```

### Version Compatibility

The SDK follows Go module compatibility rules:

- **v1.x.x** - All v1 versions are compatible
- **v2.x.x** - Breaking changes, requires code updates
- **v1.x.x-beta.x** - Pre-release, may have breaking changes

### Example: Upgrading

**From v1.0.0 to v1.2.0 (safe):**
```bash
go get github.com/brightsign/bscontrol@v1.2.0
go mod tidy
go test ./...
```

**From v1.x.x to v2.0.0 (breaking):**
```bash
# Read migration guide first
# Update import paths if needed
go get github.com/brightsign/bscontrol@v2.0.0
go mod tidy
# Fix breaking changes
go test ./...
```

### CI/CD Recommendations

**GitHub Actions example:**

```yaml
name: Build
on: [push, pull_request]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Install dependencies
        run: |
          # Pin to specific version
          go get github.com/brightsign/bscontrol@v1.2.0
          go mod download

      - name: Build
        run: go build -v ./...

      - name: Test
        run: go test -v ./...
```

**Dockerfile example:**

```dockerfile
FROM golang:1.24-alpine

WORKDIR /app

# Copy go.mod with pinned version
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o app

CMD ["./app"]
```

### Dependabot Configuration

Keep dependencies updated automatically:

```yaml
# .github/dependabot.yml
version: 2
updates:
  - package-ecosystem: "gomod"
    directory: "/"
    schedule:
      interval: "weekly"
    # Only allow patch and minor updates
    open-pull-requests-limit: 10
    reviewers:
      - "your-team"
```

### Version Matrix Testing

Test against multiple SDK versions:

```yaml
# .github/workflows/matrix.yml
strategy:
  matrix:
    bscontrol-version: ['v1.0.0', 'v1.1.0', 'v1.2.0']
steps:
  - run: go get github.com/brightsign/bscontrol@${{ matrix.bscontrol-version }}
  - run: go test ./...
```

## Version History

Use GitHub releases page to view all versions:
https://github.com/brightsign/bscontrol/releases

Or via CLI:
```bash
gh release list --repo brightdevelopers/bscontrol
```

## Support Policy

- **Latest stable** - Full support, all features
- **Previous minor** - Security fixes only
- **Pre-releases** - Experimental, no support guarantee

## Breaking Changes

Breaking changes require a major version bump (v2.x.x). Examples:

- ❌ Removing exported functions/types
- ❌ Changing function signatures
- ❌ Renaming exported identifiers
- ❌ Changing behavior of existing methods
- ✅ Adding new exports (minor bump)
- ✅ Bug fixes (patch bump)
- ✅ Internal refactoring (patch bump)

## Questions?

- Issues: https://github.com/brightsign/bscontrol/issues
- Discussions: https://github.com/brightsign/bscontrol/discussions
