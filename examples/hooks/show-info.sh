#!/bin/bash
# Example info hook: Display dev server URL based on worktree index
#
# This hook runs during `wt info` and `wt list -v` to show custom
# worktree-specific information. Output lines matching "Key: value"
# format are aligned with built-in fields.
#
# Usage in .wt.yaml:
#   hooks:
#     info:
#       - script: ./examples/hooks/show-info.sh

if [ -z "$WT_INDEX" ]; then
    exit 0
fi

PORT=$((5173 + WT_INDEX * 10))
echo "URL: http://localhost:$PORT"
