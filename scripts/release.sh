#!/bin/bash
set -euo pipefail

# Release script for wt
# Usage: release.sh <major|minor|patch> "Release notes"
#
# This script:
# 1. Reads current version from VERSION file
# 2. Bumps the appropriate segment
# 3. Updates VERSION file
# 4. Commits the change
# 5. Creates an annotated tag
# 6. Pushes commit and tag to origin main

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
VERSION_FILE="${REPO_ROOT}/VERSION"

usage() {
    echo "Usage: $0 <major|minor|patch> \"Release notes\""
    echo ""
    echo "Examples:"
    echo "  $0 patch \"Fix bug in cleanup command\""
    echo "  $0 minor \"Add new feature X\""
    echo "  $0 major \"Breaking changes to config format\""
    exit 1
}

if [[ $# -lt 2 ]]; then
    usage
fi

BUMP_TYPE="$1"
shift
RELEASE_NOTES="$*"

if [[ ! "$BUMP_TYPE" =~ ^(major|minor|patch)$ ]]; then
    echo "Error: First argument must be 'major', 'minor', or 'patch'"
    usage
fi

if [[ -z "$RELEASE_NOTES" ]]; then
    echo "Error: Release notes are required"
    usage
fi

# Safety checks
cd "$REPO_ROOT"

# Check we're on main branch
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [[ "$CURRENT_BRANCH" != "main" ]]; then
    echo "Error: Must be on main branch (currently on '$CURRENT_BRANCH')"
    exit 1
fi

# Fetch latest from origin
echo "Fetching from origin..."
git fetch origin main

# Check working directory is clean
if [[ -n $(git status --porcelain) ]]; then
    echo "Error: Working directory is dirty. Commit or stash changes first."
    exit 1
fi

# Check we're up to date with origin/main
LOCAL=$(git rev-parse HEAD)
REMOTE=$(git rev-parse origin/main)
if [[ "$LOCAL" != "$REMOTE" ]]; then
    echo "Error: Local main is not up to date with origin/main"
    echo "  Local:  $LOCAL"
    echo "  Remote: $REMOTE"
    echo "Run 'git pull' first."
    exit 1
fi

echo "Pre-flight checks passed."
echo ""

# Read current version
if [[ ! -f "$VERSION_FILE" ]]; then
    echo "Error: VERSION file not found at $VERSION_FILE"
    exit 1
fi

CURRENT_VERSION=$(cat "$VERSION_FILE" | tr -d '[:space:]')
echo "Current version: $CURRENT_VERSION"

# Parse version components
IFS='.' read -r MAJOR MINOR PATCH <<< "$CURRENT_VERSION"

# Bump version
case "$BUMP_TYPE" in
    major)
        MAJOR=$((MAJOR + 1))
        MINOR=0
        PATCH=0
        ;;
    minor)
        MINOR=$((MINOR + 1))
        PATCH=0
        ;;
    patch)
        PATCH=$((PATCH + 1))
        ;;
esac

NEW_VERSION="${MAJOR}.${MINOR}.${PATCH}"
echo "New version: $NEW_VERSION"

# Update VERSION file
echo "$NEW_VERSION" > "$VERSION_FILE"

# Write release notes to file for workflow to use
RELEASE_NOTES_FILE="${REPO_ROOT}/RELEASE_NOTES.md"
echo "$RELEASE_NOTES" > "$RELEASE_NOTES_FILE"

# Commit, tag, and push
cd "$REPO_ROOT"

git add VERSION RELEASE_NOTES.md
git commit -m "Bump version to ${NEW_VERSION}"

git tag -a "v${NEW_VERSION}" -m "Release v${NEW_VERSION}"

echo ""
echo "Pushing to origin main..."
git push origin main
git push origin "v${NEW_VERSION}"

echo ""
echo "Released v${NEW_VERSION}"
echo "GitHub Actions will now build and publish the release."
