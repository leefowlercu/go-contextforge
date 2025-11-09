#!/bin/bash
set -e

# Cleanup temporary files on exit
trap 'rm -f .next-version' EXIT

VERSION=$1

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Validate version argument
if [ -z "$VERSION" ]; then
    echo -e "${RED}Error: VERSION argument is required${NC}"
    echo "Usage: $0 v0.2.0"
    exit 1
fi

# Validate version format (vX.Y.Z)
if ! [[ $VERSION =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo -e "${RED}Error: VERSION must be in format vX.Y.Z (e.g., v0.2.0)${NC}"
    exit 1
fi

# Extract version without 'v' prefix
VERSION_NUMBER=${VERSION#v}

echo -e "${GREEN}Preparing release ${VERSION}...${NC}"

# Check if tag already exists
if git rev-parse "$VERSION" >/dev/null 2>&1; then
    echo -e "${RED}Error: Tag $VERSION already exists${NC}"
    exit 1
fi

# Update version in contextforge/version.go
echo -e "${YELLOW}Updating version constant...${NC}"
VERSION_FILE="contextforge/version.go"
if [ ! -f "$VERSION_FILE" ]; then
    echo -e "${RED}Error: $VERSION_FILE not found${NC}"
    exit 1
fi

# Use sed to update the version constant
sed -i.bak "s/const Version = \".*\"/const Version = \"$VERSION_NUMBER\"/" "$VERSION_FILE"
rm -f "${VERSION_FILE}.bak"

echo -e "${GREEN}Updated version in $VERSION_FILE to $VERSION_NUMBER${NC}"

# Stage version file
git add "$VERSION_FILE"

# Commit version change
echo -e "${YELLOW}Creating version bump commit...${NC}"
COMMIT_MESSAGE="release: prepare ${VERSION}"
git commit -m "$COMMIT_MESSAGE"
echo -e "${GREEN}Created commit: $COMMIT_MESSAGE${NC}"

# Create annotated tag
echo -e "${YELLOW}Creating git tag ${VERSION}...${NC}"
git tag -a "$VERSION" -m "Go ContextForge SDK $VERSION"
echo -e "${GREEN}Created annotated tag: $VERSION${NC}"

# Check for GoReleaser and GITHUB_TOKEN
if ! command -v goreleaser &> /dev/null; then
    echo -e "${RED}Error: goreleaser is not installed${NC}"
    echo -e "${YELLOW}Install with: go install github.com/goreleaser/goreleaser/v2@latest${NC}"
    git tag -d "$VERSION"
    git reset --hard HEAD~1
    exit 1
fi

if [ -z "$GITHUB_TOKEN" ]; then
    echo -e "${RED}Warning: GITHUB_TOKEN environment variable not set${NC}"
    echo -e "${YELLOW}GoReleaser needs this to create GitHub releases${NC}"
    echo -e "${YELLOW}See README.md for setup instructions${NC}"
    echo ""
    echo -e "${YELLOW}Continue anyway? Release will fail but you can retry later. (y/N)${NC}"
    read -r response
    if [[ ! "$response" =~ ^[Yy]$ ]]; then
        echo -e "${YELLOW}Undoing tag and commit...${NC}"
        git tag -d "$VERSION"
        git reset --hard HEAD~1
        exit 1
    fi
fi

# Run GoReleaser to create draft release and update CHANGELOG.md
echo -e "${YELLOW}Running GoReleaser...${NC}"
if ! goreleaser release --clean; then
    echo -e "${RED}Error: GoReleaser failed${NC}"
    echo -e "${YELLOW}Undoing tag and commit...${NC}"
    git tag -d "$VERSION"
    git reset --hard HEAD~1
    exit 1
fi

echo -e "${GREEN}GoReleaser completed successfully!${NC}"

# Display next steps
echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Release preparation complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "${YELLOW}GoReleaser has:${NC}"
echo "  - Updated CHANGELOG.md"
echo "  - Created draft GitHub release"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo "1. Review the draft release on GitHub:"
echo "   https://github.com/leefowlercu/go-contextforge/releases"
echo ""
echo "2. Review CHANGELOG.md changes:"
echo "   git diff CHANGELOG.md"
echo ""
echo "3. If changes needed:"
echo "   - Edit release notes on GitHub and/or CHANGELOG.md locally"
echo "   - Amend commit: git add CHANGELOG.md && git commit --amend --no-edit"
echo "   - OR undo completely: git tag -d $VERSION && git reset --hard HEAD~1"
echo ""
echo "4. When ready to publish:"
echo "   - Push commit and tag: git push && git push --tags"
echo "   - Publish draft release on GitHub"
echo ""
echo -e "${YELLOW}Note: GoReleaser created a DRAFT release. Review before publishing.${NC}"
echo ""
