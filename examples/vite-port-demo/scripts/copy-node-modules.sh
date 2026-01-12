#!/bin/bash
# Post-create hook: Copies node_modules from main repo to new worktree
#
# This avoids needing to run npm install in each worktree. Since worktrees
# share the same package.json, the dependencies are identical.

set -e

SOURCE="$WT_REPO_ROOT/node_modules"
DEST="$WT_PATH/node_modules"

if [ -d "$SOURCE" ]; then
    cp -R "$SOURCE" "$DEST"
    echo "Copied node_modules to worktree '$WT_NAME'"
else
    echo "Warning: node_modules not found in main repo, run npm install first"
fi
