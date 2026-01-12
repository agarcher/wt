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

## Commands

| Command | Description |
|---------|-------------|
| `wt create <name>` | Create a new worktree |
| `wt delete [name]` | Delete a worktree and its branch |
| `wt cleanup` | Remove worktrees with merged branches |
| `wt list` | List all worktrees |
| `wt cd <name>` | Change to a worktree directory |
| `wt exit` | Return to main repository |
| `wt root` | Print main repository path |
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

## Example Hooks

See the `examples/hooks/` directory for example hook scripts:

- `copy-env.sh` - Copy `.env` files from main repo to worktree
- `pre-delete-check.sh` - Warn about uncommitted changes before deletion
- `setup-ports.sh` - Configure unique ports based on `WT_INDEX`

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
