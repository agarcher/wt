# Vite Port Demo

This example demonstrates using `wt` hooks and `WT_INDEX` to automatically configure unique dev server ports for each worktree.

## How It Works

When you create a worktree with `wt create <name>`, the post_create hook:

1. Reads `WT_INDEX` (a stable numeric identifier assigned to each worktree)
2. Calculates a unique port: `5173 + (WT_INDEX * 10)`
3. Writes the port to `.env.local` in the worktree

Vite automatically loads `.env.local` and uses the configured port.

**Example:**
- Main repo: port 5173 (default)
- Worktree 1 (index=1): port 5183
- Worktree 2 (index=2): port 5193

## Setup

1. Copy this directory to a new location:
   ```bash
   cp -r examples/vite-port-demo ~/projects/my-app
   cd ~/projects/my-app
   ```

2. Initialize git and install dependencies:
   ```bash
   git init
   git add .
   git commit -m "Initial commit"
   npm install
   ```

3. Create worktrees:
   ```bash
   wt create feature-a
   wt create feature-b
   ```

4. Start dev servers in different worktrees:
   ```bash
   # In main repo (port 5173)
   npm run dev

   # In feature-a worktree (port 5183)
   cd worktrees/feature-a
   npm install
   npm run dev

   # In feature-b worktree (port 5193)
   cd worktrees/feature-b
   npm install
   npm run dev
   ```

Each worktree runs on its own port, allowing parallel development and testing.

## Files

- `.wt.yaml` - Configures the post_create hook
- `scripts/setup-ports.sh` - Hook that generates `.env.local` with the unique port
- `vite.config.js` - Reads `VITE_PORT` from environment
- `src/main.js` - Displays current port on the page
