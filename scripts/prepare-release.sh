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

# Update CHANGELOG.md
echo -e "${YELLOW}Updating CHANGELOG.md...${NC}"
CHANGELOG_FILE="CHANGELOG.md"
if [ ! -f "$CHANGELOG_FILE" ]; then
    echo -e "${RED}Error: $CHANGELOG_FILE not found${NC}"
    exit 1
fi

# Get current date in YYYY-MM-DD format
RELEASE_DATE=$(date +%Y-%m-%d)

# Replace [Unreleased] with [VERSION] - DATE and add new [Unreleased] section
# This is a simple approach - for more sophisticated changelog management, consider git-chglog
if grep -q "## \[Unreleased\]" "$CHANGELOG_FILE"; then
    # Create temp file with updated changelog
    awk -v version="$VERSION_NUMBER" -v date="$RELEASE_DATE" '
    /## \[Unreleased\]/ {
        print $0
        print ""
        print "## [" version "] - " date
        next
    }
    { print }
    ' "$CHANGELOG_FILE" > "${CHANGELOG_FILE}.tmp"
    mv "${CHANGELOG_FILE}.tmp" "$CHANGELOG_FILE"
    echo -e "${GREEN}Updated CHANGELOG.md with release $VERSION${NC}"
else
    echo -e "${YELLOW}Warning: [Unreleased] section not found in CHANGELOG.md${NC}"
    echo -e "${YELLOW}Please manually update CHANGELOG.md${NC}"
fi

# Stage changes
echo -e "${YELLOW}Staging changes...${NC}"
git add "$VERSION_FILE" "$CHANGELOG_FILE"

# Commit changes
echo -e "${YELLOW}Creating release commit...${NC}"
COMMIT_MESSAGE="release: prepare ${VERSION}"
git commit -m "$COMMIT_MESSAGE"

echo -e "${GREEN}Created commit: $COMMIT_MESSAGE${NC}"

# Create annotated tag
echo -e "${YELLOW}Creating git tag ${VERSION}...${NC}"
git tag -a "$VERSION" -m "Go ContextForge SDK $VERSION"

echo -e "${GREEN}Created annotated tag: $VERSION${NC}"

# Display next steps
echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Release preparation complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo "1. Review the changes:"
echo "   git show HEAD"
echo ""
echo "2. Push the changes and tag:"
echo "   git push && git push --tags"
echo ""
echo "3. Create a GitHub release at:"
echo "   https://github.com/leefowlercu/go-contextforge/releases/new?tag=$VERSION"
echo ""
echo -e "${YELLOW}Note: If you need to undo this release preparation:${NC}"
echo "   git tag -d $VERSION"
echo "   git reset --hard HEAD~1"
echo ""
