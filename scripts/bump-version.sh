#!/bin/bash
set -e

BUMP_TYPE=$1

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Validate bump type argument
if [ -z "$BUMP_TYPE" ]; then
    echo -e "${RED}Error: Bump type argument is required${NC}"
    echo "Usage: $0 <major|minor|patch>"
    exit 1
fi

# Validate bump type value
if [[ ! "$BUMP_TYPE" =~ ^(major|minor|patch)$ ]]; then
    echo -e "${RED}Error: Invalid bump type '${BUMP_TYPE}'${NC}"
    echo "Valid options: major, minor, patch"
    exit 1
fi

# Locate version file
VERSION_FILE="contextforge/version.go"
if [ ! -f "$VERSION_FILE" ]; then
    echo -e "${RED}Error: $VERSION_FILE not found${NC}"
    exit 1
fi

# Extract current version from version.go
CURRENT_VERSION=$(grep -E 'const Version = ".*"' "$VERSION_FILE" | sed -E 's/.*"(.*)".*/\1/')

if [ -z "$CURRENT_VERSION" ]; then
    echo -e "${RED}Error: Could not parse version from $VERSION_FILE${NC}"
    echo "Expected format: const Version = \"X.Y.Z\""
    exit 1
fi

# Validate version format
if ! [[ "$CURRENT_VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo -e "${RED}Error: Current version '$CURRENT_VERSION' is not in valid semver format${NC}"
    echo "Expected format: X.Y.Z (e.g., 0.1.0)"
    exit 1
fi

echo -e "${YELLOW}Current version: ${CURRENT_VERSION}${NC}"

# Parse version components
IFS=. read -r major minor patch <<EOF
$CURRENT_VERSION
EOF

# Bump version based on type
case "$BUMP_TYPE" in
    major)
        NEW_VERSION="$((major + 1)).0.0"
        ;;
    minor)
        NEW_VERSION="${major}.$((minor + 1)).0"
        ;;
    patch)
        NEW_VERSION="${major}.${minor}.$((patch + 1))"
        ;;
esac

echo -e "${GREEN}New version:     ${NEW_VERSION}${NC}"

# Write new version with 'v' prefix to temp file for Makefile
echo "v${NEW_VERSION}" > .next-version

echo -e "${GREEN}Version bump calculated successfully${NC}"
