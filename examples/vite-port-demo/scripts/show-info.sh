#!/bin/bash
# Info hook: Displays the dev server URL for this worktree
#
# This hook is run by `wt info` and `wt list -v` to show custom
# worktree-specific information. It calculates the port the same
# way as setup-ports.sh and outputs a clickable URL.

if [ -z "$WT_INDEX" ]; then
    exit 0
fi

PORT=$((5173 + WT_INDEX * 10))
echo "URL: http://localhost:$PORT"
