# wt - Git Worktree Manager

[![CI](https://github.com/agarcher/wt/actions/workflows/ci.yml/badge.svg)](https://github.com/agarcher/wt/actions/workflows/ci.yml)
[![Release](https://github.com/agarcher/wt/actions/workflows/release.yml/badge.svg)](https://github.com/agarcher/wt/actions/workflows/release.yml)

A CLI for managing git worktrees with lifecycle hooks. Create isolated, fully testable environments for running multiple LLM coding agents in parallel.

https://github.com/user-attachments/assets/7ec0fffa-b39c-45c1-98f3-68a1a2f91cf3

## Installation

### Homebrew (macOS/Linux)

```bash
brew install agarcher/tap/wt
```

After installing, add shell integration to your shell config:

```bash
# zsh (~/.zshrc)
eval "$(wt init zsh)"

# bash (~/.bashrc)
eval "$(wt init bash)"

# fish (~/.config/fish/config.fish)
wt init fish | source
```

Restart your shell or source the config file.

### Alternative Installation

<details>
<summary>From Source</summary>

```bash
git clone https://github.com/agarcher/wt.git
cd wt
make build
make install  # Installs to /usr/local/bin
```

Then add shell integration as shown above.

</details>

<details>
<summary>Download Binary</summary>

Download the latest release from the [releases page](https://github.com/agarcher/wt/releases) and add it to your PATH.

Then add shell integration as shown above.

</details>

**Requirements:** Git 2.5+ (2.13+ recommended for optimal completion)

## Quick Start

1. Create a `.wt.yaml` file in your repository root:

```yaml
version: 1
worktree_dir: worktrees
```

2. Create and use worktrees:

```bash
wt create feature-x      # Create and cd into worktrees/feature-x
wt list                   # List all worktrees
wt cd feature-x           # Switch to a worktree
wt exit                   # Return to main repo
wt delete feature-x       # Delete worktree and branch
```

## Commands

| Command | Description | Details |
|---------|-------------|---------|
| `wt create <name>` | Create a new worktree | [docs](docs/USAGE.md#wt-create) |
| `wt delete [name]` | Delete a worktree and its branch | [docs](docs/USAGE.md#wt-delete) |
| `wt list` | List all worktrees with status | [docs](docs/USAGE.md#wt-list) |
| `wt info [name]` | Show detailed worktree information | [docs](docs/USAGE.md#wt-info) |
| `wt cd <name>` | Change to a worktree directory | [docs](docs/USAGE.md#wt-cd) |
| `wt exit` | Return to main repository | [docs](docs/USAGE.md#wt-exit) |
| `wt cleanup` | Remove worktrees with merged branches | [docs](docs/USAGE.md#wt-cleanup) |
| `wt config` | Manage user configuration | [docs](docs/USAGE.md#wt-config) |
| `wt init <shell>` | Generate shell integration | [docs](docs/USAGE.md#wt-init) |
| `wt root` | Print main repository path | [docs](docs/USAGE.md#wt-root) |
| `wt version` | Print version | [docs](docs/USAGE.md#wt-version) |

See the [Usage Guide](docs/USAGE.md) for detailed command documentation.

## Configuration

### Repository Configuration

Each repository is configured via `.wt.yaml` at the repository root:

```yaml
version: 1
worktree_dir: worktrees       # Where worktrees are stored
branch_pattern: "{name}"      # Branch naming pattern
default_branch: main          # Branch for comparison

hooks:
  post_create:
    - script: ./scripts/setup.sh
```

See [Repository Configuration](docs/USAGE.md#repository-configuration) for all options.

### User Configuration

Global settings are stored in `~/.config/wt/config.yaml`:

```bash
wt config --global remote origin      # Compare against remote
wt config --global fetch_interval 5m  # Fetch throttling
```

See [User Configuration](docs/USAGE.md#user-configuration) for details.

## Hooks

Hooks run custom scripts at key points in the worktree lifecycle:

| Hook | When it runs |
|------|--------------|
| `pre_create` | Before worktree creation |
| `post_create` | After worktree creation |
| `pre_delete` | Before worktree deletion |
| `post_delete` | After worktree deletion |
| `info` | During `wt info` and `wt list -v` |

All hooks receive environment variables like `WT_NAME`, `WT_PATH`, `WT_BRANCH`, and `WT_INDEX`.

See the [Hooks Guide](docs/HOOKS.md) for configuration and examples.

## Why wt?

LLM coding agents work best with full access to build, test, and run your project. But running multiple agents on the same codebase creates conflicts: port collisions, shared state, and dependency issues.

`wt` solves this with lifecycle hooks that automatically configure each worktree:

- **Unique ports**: Assign dev server ports based on `WT_INDEX`
- **Copied dependencies**: Clone `node_modules` so each worktree is immediately runnable
- **Custom setup**: Run any project-specific initialization scripts

Each agent gets a fully isolated environment where it can build, test, and iterate without affecting others. See the [Vite port demo](examples/vite-port-demo/) for a working example.

## Best Practices

**Compare against remote:** By default, `wt` compares branches locally. Set `remote` to compare against the remote tracking branch instead, so merge status reflects what's actually been pushed:

```bash
wt config --global remote origin
```

**Tune fetch frequency:** When `remote` is set, `wt` fetches at most every 5 minutes. Adjust with `fetch_interval`:

```bash
wt config --global fetch_interval 10m   # Less frequent
wt config --global fetch_interval 0     # Always fetch
```

See [User Configuration](docs/USAGE.md#user-configuration) for all options.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and guidelines.

## License

MIT
