# wt - Git Worktree Manager

[![CI](https://github.com/agarcher/wt/actions/workflows/ci.yml/badge.svg)](https://github.com/agarcher/wt/actions/workflows/ci.yml)
[![Release](https://github.com/agarcher/wt/actions/workflows/release.yml/badge.svg)](https://github.com/agarcher/wt/actions/workflows/release.yml)

A CLI for managing local git worktrees with lifecycle hooks. Create isolated, fully testable environments—ideal for running multiple LLM coding agents in parallel. See the [Vite port demo](examples/vite-port-demo/) for a working example.

## Features

- **Simple worktree management**: Create, delete, and list worktrees with ease
- **Shell integration**: Automatically `cd` into new worktrees
- **Tab completion**: Complete commands, worktree names, and branches
- **Lifecycle hooks**: Run custom scripts on worktree creation and deletion
- **Non-intrusive**: Only activates in repositories with `.wt.yaml` config

## Requirements

- **Git 2.5+** for basic functionality (worktree support)
- **Git 2.13+** recommended for optimal branch name completion (older versions fall back gracefully)

## Installation

### From Source

```bash
git clone https://github.com/agarcher/wt.git
cd wt
make build
make install  # Installs to /usr/local/bin
```

### Homebrew (macOS/Linux)

```bash
brew install agarcher/tap/wt
```

### Download Binary

Download the latest release from the [releases page](https://github.com/agarcher/wt/releases).

## Shell Integration

Add one of the following to your shell configuration:

```bash
# For zsh (~/.zshrc)
eval "$(wt init zsh)"

# For bash (~/.bashrc)
eval "$(wt init bash)"

# For fish (~/.config/fish/config.fish)
wt init fish | source
```

## Quick Start

1. Create a `.wt.yaml` file in your repository root:

```yaml
version: 1
worktree_dir: worktrees
branch_pattern: "{name}"
```

2. Create a worktree:

```bash
wt create feature-x
# Automatically creates and cd's into worktrees/feature-x
```

3. List worktrees:

```bash
wt list
```

4. Switch between worktrees:

```bash
wt cd feature-x    # Switch to a worktree
wt exit            # Return to main repo
```

5. Delete a worktree:

```bash
wt delete feature-x
# Or from within the worktree:
wt delete
```

6. Clean up merged worktrees:

```bash
wt cleanup              # Remove worktrees with merged branches
wt cleanup --dry-run    # Preview what would be deleted
```

## Configuration

Create a `.wt.yaml` file in your repository root:

```yaml
# .wt.yaml
version: 1

# Where worktrees are stored (relative to repo root)
worktree_dir: worktrees

# Branch naming pattern
# Available variables: {name}
branch_pattern: "{name}"

# Branch to compare against for list/cleanup status (optional)
# If not set, auto-detected from remote HEAD or defaults to "main"
default_branch: main

# Lifecycle hooks (all optional)
hooks:
  # Runs before worktree creation (in repo root)
  pre_create:
    - script: ./scripts/pre-create.sh

  # Runs after worktree creation (in new worktree)
  post_create:
    - script: ./scripts/copy-env.sh
    - script: ./scripts/setup.sh
      env:
        CUSTOM_VAR: "value"

  # Runs before worktree deletion (in worktree)
  pre_delete:
    - script: ./scripts/pre-delete-check.sh

  # Runs after worktree deletion (in repo root)
  post_delete:
    - script: ./scripts/cleanup.sh

  # Runs for wt info and wt list -v (in worktree)
  info:
    - script: ./scripts/show-info.sh
```

### Hook Environment Variables

All hooks receive these environment variables:

| Variable | Description |
|----------|-------------|
| `WT_NAME` | Worktree name |
| `WT_PATH` | Full path to worktree |
| `WT_BRANCH` | Git branch name |
| `WT_REPO_ROOT` | Main repository root path |
| `WT_WORKTREE_DIR` | Worktree directory name |
| `WT_INDEX` | Stable numeric index (1+), useful for port offsets |

### Worktree Index

Each worktree is assigned a stable numeric index starting at 1. When a worktree is deleted, its index becomes available for reuse by the next created worktree. This is useful for:

- **Port allocation**: Assign unique ports per worktree (e.g., `VITE_PORT=$((5173 + WT_INDEX * 10))`)
- **Resource isolation**: Unique database names, container names, etc.
- **Parallel testing**: Run multiple worktree environments simultaneously without conflicts

The index is stored in `.git/worktrees/<name>/wt-index` and is automatically cleaned up when the worktree is removed.

You can optionally limit the maximum index value:

```yaml
# .wt.yaml
version: 1
worktree_dir: worktrees
index:
  max: 20  # Optional: limit indexes to 1-20
```

### Info Hooks

Info hooks let you display custom worktree-specific information in `wt info` and `wt list -v`. Unlike other hooks that run during lifecycle events, info hooks run on-demand and their output is captured and displayed.

**Configuration:**
```yaml
hooks:
  info:
    - script: ./scripts/show-info.sh
```

**Example script:**
```bash
#!/bin/bash
# Output key-value pairs that align with built-in fields
echo "URL: http://localhost:$((5173 + WT_INDEX * 10))"
echo "Database: dev_${WT_NAME}"
```

**Output format:**
- Lines matching `Key: value` format are aligned with built-in keys (Branch, Index, etc.)
- Other output is displayed as-is below the key-value section

This is useful for displaying:
- Dev server URLs
- Database connection info
- Port assignments
- Any worktree-specific configuration

## Commands

| Command | Description |
|---------|-------------|
| `wt create <name>` | Create a new worktree |
| `wt delete [name]` | Delete a worktree and its branch |
| `wt cleanup` | Remove worktrees with merged branches |
| `wt list` | List all worktrees |
| `wt info [name]` | Show detailed worktree information |
| `wt cd <name>` | Change to a worktree directory |
| `wt exit` | Return to main repository |
| `wt root` | Print main repository path |
| `wt config` | Manage user configuration |
| `wt init <shell>` | Generate shell integration script |
| `wt version` | Print version |

### Create Options

```bash
wt create <name> [flags]

Flags:
  -b, --branch string   Use existing branch instead of creating new one
```

### Delete Options

```bash
wt delete [name] [flags]

Flags:
  -f, --force         Force deletion even with uncommitted changes
  -k, --keep-branch   Keep the associated branch (default: delete it)
```

### Cleanup Options

```bash
wt cleanup [flags]

Flags:
  -n, --dry-run       Show what would be deleted without deleting
  -f, --force         Skip confirmation prompts
  -k, --keep-branch   Keep the associated branches (default: delete them)
```

The `cleanup` command finds worktrees whose branches have been merged into the default branch (main/master) and removes them. This is useful for cleaning up after completing work on feature branches.

### Info Command

```bash
wt info [name]
```

Displays detailed information about a worktree. If no name is provided and you're inside a worktree, shows info for the current worktree.

**Example output:**
```
================================================================================
* feature-auth
  Branch:  feature-auth
  Index:   2
  Created: 2025-01-10 (3 days ago)
  Status:  ↑3 ↓1 [in_progress, dirty]
  URL:     http://localhost:5193
================================================================================
```

The `URL` line comes from an info hook (see [Info Hooks](#info-hooks) below).

### Config Options

```bash
wt config [key] [value] [flags]

Flags:
  --global        Set/get global configuration
  --unset         Remove a per-repo configuration value
  --list          List all configuration values
  --show-origin   Show where each configuration value comes from
```

## User Configuration

User settings are stored in `~/.config/wt/config.yaml` and control how `wt list` and `wt cleanup` compare worktree branches.

### Remote Comparison Mode

By default, worktrees are compared against the local default branch (e.g., `main`). You can configure comparison against a remote branch instead:

```bash
# Compare against origin/main globally
wt config --global remote origin

# Set minimum time between fetches (default: 5m)
wt config --global fetch_interval 10m

# Override remote for a specific repository
wt config remote upstream

# Disable fetch caching for current repo (always fetch)
wt config fetch_interval 0

# Disable fetch entirely for current repo
wt config fetch_interval never

# View current settings
wt config --list

# See where each value comes from
wt config --show-origin
```

### Configuration Keys

| Key | Default | Description |
|-----|---------|-------------|
| `remote` | `""` (empty) | Remote to compare against. Empty = local comparison |
| `fetch_interval` | `5m` | Minimum time between fetches. Set to `0` to always fetch, or `never` to disable |

### Configuration File Structure

```yaml
# ~/.config/wt/config.yaml

# Global settings
remote: origin         # Compare to origin/branch
fetch_interval: 5m     # Minimum time between fetches

# Per-repo overrides (keyed by repo path)
repos:
  /path/to/repo1:
    remote: upstream       # This repo compares to upstream/branch
  /path/to/repo2:
    remote: ""             # This repo uses local comparison
    fetch_interval: never  # Never fetch for this repo
```

## Example Hooks

See the `examples/hooks/` directory for example hook scripts:

- `copy-env.sh` - Copy `.env` files from main repo to worktree
- `pre-delete-check.sh` - Warn about uncommitted changes before deletion
- `setup-ports.sh` - Configure unique ports based on `WT_INDEX`
- `show-info.sh` - Display dev server URL in `wt info` output

## Why wt?

LLM coding agents work best with full access to build, test, and run your project. But running multiple agents on the same codebase creates conflicts—port collisions, shared state, and dependency issues.

`wt` solves this with lifecycle hooks that automatically configure each worktree:

- **Unique ports**: Assign dev server ports based on `WT_INDEX` (e.g., agent 1 gets port 5183, agent 2 gets 5193)
- **Copied dependencies**: Clone `node_modules` so each worktree is immediately runnable
- **Custom setup**: Run any project-specific initialization scripts

Each agent gets a fully isolated environment where it can build, test, and iterate without affecting others.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, building, and release instructions.

## License

MIT
