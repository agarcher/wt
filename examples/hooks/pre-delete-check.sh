#!/bin/bash
# Example pre-delete hook: Check for uncommitted changes
#
# Usage in .wt.yaml:
#   hooks:
#     pre_delete:
#       - script: ./examples/hooks/pre-delete-check.sh

set -e

cd "$WT_PATH"

# Check for uncommitted changes
if [[ -n $(git status --porcelain) ]]; then
  echo "Warning: Worktree '$WT_NAME' has uncommitted changes:"
  git status --short
  echo
  read -p "Are you sure you want to delete? [y/N] " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 1
  fi
fi

# Check for unpushed commits
branch=$(git branch --show-current)
if git rev-parse --verify "$branch@{upstream}" >/dev/null 2>&1; then
  unpushed=$(git rev-list --count "$branch@{upstream}..HEAD" 2>/dev/null || echo "0")
  if [[ "$unpushed" -gt 0 ]]; then
    echo "Warning: Worktree '$WT_NAME' has $unpushed unpushed commit(s)"
    read -p "Are you sure you want to delete? [y/N] " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
      echo "Aborted."
      exit 1
    fi
  fi
fi

echo "Pre-delete checks passed"
