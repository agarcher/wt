# wt - Git Worktree Manager

[![CI](https://github.com/agarcher/worktree/actions/workflows/ci.yml/badge.svg)](https://github.com/agarcher/worktree/actions/workflows/ci.yml)
[![Release](https://github.com/agarcher/worktree/actions/workflows/release.yml/badge.svg)](https://github.com/agarcher/worktree/actions/workflows/release.yml)

A cross-platform CLI tool for managing git worktrees with lifecycle hooks.

## Features

- **Simple worktree management**: Create, delete, and list worktrees with ease
- **Shell integration**: Automatically `cd` into new worktrees
- **Lifecycle hooks**: Run custom scripts on worktree creation and deletion
- **Non-intrusive**: Only activates in repositories with `.wt.yaml` config

## Installation

### From Source

```bash
git clone https://github.com/agarcher/worktree.git
cd worktree
make build
make install  # Installs to /usr/local/bin
```

### Homebrew (coming soon)

```bash
brew install agarcher/tap/worktree
```

### Download Binary

Download the latest release from the [releases page](https://github.com/agarcher/worktree/releases).

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

## Why wt?

Managing multiple parallel development streams (especially when working with LLMs) often requires:

1. **Isolated environments**: Each worktree has its own working directory
2. **Quick context switching**: Jump between features without stashing
3. **Custom setup**: Run project-specific setup scripts automatically
4. **Clean teardown**: Ensure nothing is lost before deletion

`wt` provides all of this with a simple, consistent interface.

## Development

### Building

```bash
# Build for current platform
go build -o wt ./cmd/wt

# Build for all platforms
GOOS=darwin GOARCH=amd64 go build -o wt-darwin-amd64 ./cmd/wt
GOOS=darwin GOARCH=arm64 go build -o wt-darwin-arm64 ./cmd/wt
GOOS=linux GOARCH=amd64 go build -o wt-linux-amd64 ./cmd/wt
GOOS=linux GOARCH=arm64 go build -o wt-linux-arm64 ./cmd/wt
```

### Testing Locally

This repository includes a `.wt.yaml` config, so you can test the tool against this repo itself:

```bash
# Build and add to PATH
make build
export PATH="$(pwd)/build:$PATH"

# Set up shell integration (overwrites any existing wt function)
eval "$(./build/wt init zsh)"  # or bash

# Test commands
wt list
wt create test-feature
wt cd test-feature      # now in the worktree
wt list                 # works from within worktree
wt exit                 # back to main repo
wt delete test-feature
```

### Running Tests

```bash
make test
```

### Linting

```bash
make lint
```

### Releasing

Releases are automated via GitHub Actions. To create a new release:

1. Update the version in your code if needed
2. Create and push a version tag:
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```
3. The release workflow will automatically:
   - Run tests
   - Build binaries for all platforms (darwin/linux, amd64/arm64)
   - Create a GitHub release with the binaries
   - Generate SHA256 checksums

## License

MIT
