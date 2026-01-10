# wt - Git Worktree Manager

A cross-platform CLI tool for managing git worktrees with lifecycle hooks.

## Features

- **Simple worktree management**: Create, delete, and list worktrees with ease
- **Shell integration**: Automatically `cd` into new worktrees
- **Lifecycle hooks**: Run custom scripts on worktree creation and deletion
- **Non-intrusive**: Only activates in repositories with `.wt.yaml` config

## Installation

### From Source

```bash
git clone https://github.com/agarcher/wt.git
cd wt
make build
make install  # Installs to /usr/local/bin
```

### Homebrew (coming soon)

```bash
brew install agarcher/tap/wt
```

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

## Commands

| Command | Description |
|---------|-------------|
| `wt create <name>` | Create a new worktree |
| `wt delete [name]` | Delete a worktree (auto-detects if in worktree) |
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
  -f, --force           Force deletion even with uncommitted changes
  -D, --delete-branch   Also delete the associated branch
```

## Example Hooks

See the `examples/hooks/` directory for example hook scripts:

- `copy-env.sh` - Copy `.env` files from main repo to worktree
- `pre-delete-check.sh` - Warn about uncommitted changes before deletion

## Why wt?

Managing multiple parallel development streams (especially when working with LLMs) often requires:

1. **Isolated environments**: Each worktree has its own working directory
2. **Quick context switching**: Jump between features without stashing
3. **Custom setup**: Run project-specific setup scripts automatically
4. **Clean teardown**: Ensure nothing is lost before deletion

`wt` provides all of this with a simple, consistent interface.

## License

MIT
