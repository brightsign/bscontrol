#!/bin/bash
set -e

# Release script for bscontrol SDK
# Usage: ./scripts/release.sh <version> <release-name>
# Example: ./scripts/release.sh v1.2.0 "Type Safety Release"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check arguments
if [ $# -lt 2 ]; then
    echo -e "${RED}Error: Missing arguments${NC}"
    echo "Usage: $0 <version> <release-name>"
    echo "Example: $0 v1.2.0 \"Type Safety Release\""
    exit 1
fi

VERSION=$1
RELEASE_NAME=$2

# Validate version format (must start with 'v' followed by semver)
if ! [[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$ ]]; then
    echo -e "${RED}Error: Invalid version format${NC}"
    echo "Version must be in format: v<major>.<minor>.<patch> (e.g., v1.2.0)"
    echo "Optional pre-release: v1.2.0-beta.1"
    exit 1
fi

# Check if gh CLI is installed
if ! command -v gh &> /dev/null; then
    echo -e "${RED}Error: GitHub CLI (gh) is not installed${NC}"
    echo "Install from: https://cli.github.com/"
    exit 1
fi

# Check if we're authenticated with gh
if ! gh auth status &> /dev/null; then
    echo -e "${RED}Error: Not authenticated with GitHub CLI${NC}"
    echo "Run: gh auth login"
    exit 1
fi

# Check if we're in a git repository
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    echo -e "${RED}Error: Not in a git repository${NC}"
    exit 1
fi

# Check if working directory is clean
if ! git diff-index --quiet HEAD --; then
    echo -e "${YELLOW}Warning: Working directory has uncommitted changes${NC}"
    read -p "Continue anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Check if tag already exists
if git rev-parse "$VERSION" >/dev/null 2>&1; then
    echo -e "${RED}Error: Tag $VERSION already exists${NC}"
    exit 1
fi

# Get current branch
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
echo -e "${GREEN}Current branch: $CURRENT_BRANCH${NC}"

# Confirm release details
echo
echo -e "${GREEN}Release Details:${NC}"
echo "  Version: $VERSION"
echo "  Release Name: $RELEASE_NAME"
echo "  Branch: $CURRENT_BRANCH"
echo "  Commit: $(git rev-parse --short HEAD)"
echo
read -p "Create release? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Release cancelled"
    exit 0
fi

# Run tests
echo
echo -e "${GREEN}Running tests...${NC}"
if ! go test ./...; then
    echo -e "${RED}Error: Tests failed${NC}"
    exit 1
fi

# Build to verify
echo
echo -e "${GREEN}Building...${NC}"
if ! go build ./...; then
    echo -e "${RED}Error: Build failed${NC}"
    exit 1
fi

# Create git tag
echo
echo -e "${GREEN}Creating git tag $VERSION...${NC}"
git tag -a "$VERSION" -m "$RELEASE_NAME"

# Push tag to remote
echo -e "${GREEN}Pushing tag to remote...${NC}"
git push origin "$VERSION"

# Generate release notes
echo
echo -e "${GREEN}Generating release notes...${NC}"

# Get commits since last tag
LAST_TAG=$(git describe --tags --abbrev=0 HEAD^ 2>/dev/null || echo "")
if [ -z "$LAST_TAG" ]; then
    # First release - get all commits
    COMMITS=$(git log --oneline --pretty=format:"- %s (%h)" HEAD)
else
    # Get commits since last tag
    COMMITS=$(git log --oneline --pretty=format:"- %s (%h)" "$LAST_TAG"..HEAD)
fi

# Count changes
if [ -z "$LAST_TAG" ]; then
    FILE_CHANGES=$(git diff --stat HEAD | tail -1)
else
    FILE_CHANGES=$(git diff --stat "$LAST_TAG"..HEAD | tail -1)
fi

# Create release notes
RELEASE_NOTES=$(cat <<EOF
# $RELEASE_NAME

## Installation

\`\`\`bash
go get github.com/brightsign/bscontrol@$VERSION
\`\`\`

Or in your go.mod:
\`\`\`
require github.com/brightsign/bscontrol $VERSION
\`\`\`

## Changes

$COMMITS

## Statistics

$FILE_CHANGES

## Documentation

Full documentation: https://github.com/brightsign/bscontrol#readme

## Verifying the Release

\`\`\`bash
# View module info
go list -m github.com/brightsign/bscontrol@$VERSION

# Check dependencies
go mod graph | grep bscontrol

# Run tests
go get github.com/brightsign/bscontrol@$VERSION
go test github.com/brightsign/bscontrol@$VERSION
\`\`\`
EOF
)

# Create GitHub release
echo
echo -e "${GREEN}Creating GitHub release...${NC}"

# Check if this is a pre-release
if [[ "$VERSION" =~ -[a-zA-Z] ]]; then
    PRERELEASE_FLAG="--prerelease"
else
    PRERELEASE_FLAG=""
fi

gh release create "$VERSION" \
    --title "$RELEASE_NAME" \
    --notes "$RELEASE_NOTES" \
    $PRERELEASE_FLAG

# Success
echo
echo -e "${GREEN}✓ Release $VERSION created successfully!${NC}"
echo
echo "Release URL: $(gh release view "$VERSION" --json url -q .url)"
echo
echo "Users can install with:"
echo -e "${YELLOW}  go get github.com/brightsign/bscontrol@$VERSION${NC}"
echo
echo "Or add to go.mod:"
echo -e "${YELLOW}  require github.com/brightsign/bscontrol $VERSION${NC}"
