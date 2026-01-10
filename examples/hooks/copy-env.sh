#!/bin/bash
# Example post-create hook: Copy environment files from main repo
#
# Usage in .wt.yaml:
#   hooks:
#     post_create:
#       - script: ./examples/hooks/copy-env.sh

set -e

echo "Copying environment files..."

# Copy .env files from main repo to worktree
for env_file in "$WT_REPO_ROOT"/.env*; do
  if [[ -f "$env_file" ]]; then
    filename=$(basename "$env_file")
    cp "$env_file" "$WT_PATH/$filename"
    echo "  Copied $filename"
  fi
done

echo "Environment files copied successfully"
