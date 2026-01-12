#!/bin/bash
# Post-create hook: Configures unique dev server port based on WT_INDEX
#
# This hook is run after a worktree is created. It calculates a unique port
# by adding (WT_INDEX * 10) to the base Vite port (5173), then writes it
# to .env.local so Vite picks it up automatically.
#
# Example: WT_INDEX=1 -> port 5183, WT_INDEX=2 -> port 5193

set -e

if [ -z "$WT_INDEX" ]; then
    echo "Warning: WT_INDEX not set, skipping port configuration"
    exit 0
fi

PORT=$((5173 + WT_INDEX * 10))
echo "VITE_PORT=$PORT" > "$WT_PATH/.env.local"
echo "Configured VITE_PORT=$PORT for worktree '$WT_NAME' (index $WT_INDEX)"
