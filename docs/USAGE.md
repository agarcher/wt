# Usage Guide

This document provides detailed information about all `wt` commands and configuration options.

## Commands

### wt create

Create a new git worktree.

```bash
wt create <name> [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-b, --branch <branch>` | Use an existing branch instead of creating a new one |

**Behavior:**

- Creates a worktree in the directory specified by [`worktree_dir`](#worktree_dir)
- Creates a new branch using [`branch_pattern`](#branch_pattern) (or uses existing branch with `-b`)
- Allocates a [worktree index](HOOKS.md#worktree-index) for resource isolation
- Automatically `cd`s into the new worktree (requires [shell integration](../README.md#installation))

**Example:**

```bash
# Create worktree with new branch
wt create feature-auth
# Creates worktrees/feature-auth with branch "feature-auth"

# Create worktree using existing branch
wt create hotfix -b hotfix/urgent-fix
# Creates worktrees/hotfix using branch "hotfix/urgent-fix"
```

**Hooks triggered:** [`pre_create`](HOOKS.md#pre_create), [`post_create`](HOOKS.md#post_create)

---

### wt delete

Delete a git worktree and its associated branch.

```bash
wt delete [name] [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `-f, --force` | Force deletion even with uncommitted changes | `false` |
| `-k, --keep-branch` | Keep the associated branch after deletion | `false` |

**Behavior:**

- If no name provided, deletes the current worktree (must be inside one)
- Safety checks (skip with `--force`):
  - Fails if worktree has uncommitted changes
  - Fails if worktree has commits not merged into the comparison branch
- Deletes the associated branch unless `--keep-branch` is specified
- Returns to repository root if deleting the current worktree

**Example:**

```bash
# Delete a specific worktree
wt delete feature-auth

# Delete current worktree
wt delete

# Force delete with uncommitted changes
wt delete feature-auth --force

# Delete worktree but keep the branch
wt delete feature-auth --keep-branch
```

**Hooks triggered:** [`pre_delete`](HOOKS.md#pre_delete), [`post_delete`](HOOKS.md#post_delete)

---

### wt list

List all managed worktrees with status information.

```bash
wt list [flags]
```

**Aliases:** `ls`

**Flags:**

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Show detailed multi-line output with age and hook info |

**Example output:**

```
  NAME         INDEX  BRANCH         STATUS
* feature-auth   1    feature-auth   ↑2 [in_progress]
  feature-nav    2    feature-nav    ↑5 [in_progress, dirty]
  doc-fix        3    doc-fix        [merged]
  experiment     -    experiment     [new]
```

**Status indicators:**

| Indicator | Meaning |
|-----------|---------|
| `*` | Current worktree |
| `↑N` | N commits ahead of comparison branch |
| `↓N` | N commits behind comparison branch |
| `[new]` | No commits yet (still on initial commit) |
| `[in_progress]` | Has unmerged commits |
| `[merged]` | Branch merged into comparison branch |
| `[merged in #123]` | Merged via specific PR |
| `[dirty]` | Has uncommitted changes |

**Hooks triggered:** [`info`](HOOKS.md#info) (verbose mode only)

---

### wt info

Show detailed information about a worktree.

```bash
wt info [name]
```

**Behavior:**

- If no name provided, shows info for the current worktree
- Displays branch, index, creation date, and status
- Runs info hooks and displays their output

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

The `URL` line comes from an [info hook](HOOKS.md#info).

**Hooks triggered:** [`info`](HOOKS.md#info)

---

### wt cd

Change to a worktree directory.

```bash
wt cd <name>
```

**Behavior:**

- Changes the shell's working directory to the specified worktree
- Requires [shell integration](../README.md#installation)

**Example:**

```bash
wt cd feature-auth
# Now in worktrees/feature-auth
```

---

### wt exit

Return to the main repository root.

```bash
wt exit
```

**Behavior:**

- Changes the shell's working directory back to the repository root
- Works from any worktree or subdirectory
- Requires [shell integration](../README.md#installation)

**Example:**

```bash
# Inside worktrees/feature-auth
wt exit
# Now in repository root
```

---

### wt cleanup

Remove worktrees whose branches have been merged.

```bash
wt cleanup [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `-n, --dry-run` | Show what would be deleted without deleting | `false` |
| `-f, --force` | Skip confirmation prompts | `false` |
| `-k, --keep-branch` | Keep the associated branches | `false` |

**Behavior:**

- Identifies worktrees eligible for cleanup:
  - Branch merged into default branch
  - No uncommitted changes
  - Not newly created
- Prompts for confirmation (skip with `--force`)
- Returns to repository root if current worktree is deleted

**Example:**

```bash
# Preview what would be cleaned up
wt cleanup --dry-run

# Clean up with confirmation
wt cleanup

# Clean up without confirmation
wt cleanup --force
```

**Hooks triggered:** [`pre_delete`](HOOKS.md#pre_delete), [`post_delete`](HOOKS.md#post_delete) (per worktree)

---

### wt config

Get and set user configuration options.

```bash
wt config [key] [value] [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--global` | Operate on global config |
| `--list` | List all configuration values |
| `--show-origin` | Show where each value comes from |
| `--unset` | Remove a per-repo configuration value |

**Configuration keys:**

| Key | Default | Description |
|-----|---------|-------------|
| [`remote`](#remote) | `""` (empty) | Remote to compare against |
| [`fetch_interval`](#fetch_interval) | `5m` | Minimum time between fetches |

**Examples:**

```bash
# View all settings
wt config --list

# See where each value comes from
wt config --show-origin

# Set global remote
wt config --global remote origin

# Set fetch interval for current repo
wt config fetch_interval 10m

# Disable fetching for current repo
wt config fetch_interval never

# Remove per-repo override
wt config --unset fetch_interval

# Remove global setting
wt config --global --unset remote
```

See [User Configuration](#user-configuration) for details.

---

### wt init

Generate shell integration script.

```bash
wt init <shell>
```

**Arguments:**

| Argument | Options |
|----------|---------|
| `<shell>` | `zsh`, `bash`, `fish` |

**Behavior:**

Outputs a shell-specific script that provides:
- `wt` shell function with `cd` support
- Command and argument completion

**Installation:**

```bash
# zsh (~/.zshrc)
eval "$(wt init zsh)"

# bash (~/.bashrc)
eval "$(wt init bash)"

# fish (~/.config/fish/config.fish)
wt init fish | source
```

---

### wt root

Print the main repository root path.

```bash
wt root
```

**Behavior:**

- Outputs the absolute path to the repository root
- Works from anywhere in the repo or any worktree
- Useful for scripting

**Example:**

```bash
$ wt root
/Users/dev/projects/my-app
```

---

### wt version

Print the version number.

```bash
wt version
```

**Example output:**

```
wt version 1.2.3
```

---

### wt completion

Generate shell completion script.

```bash
wt completion <shell>
```

**Arguments:**

| Argument | Options |
|----------|---------|
| `<shell>` | `bash`, `zsh`, `fish`, `powershell` |

**Note:** Shell integration (`wt init`) already includes completions. Use this command only if you need standalone completion scripts.

---

## Configuration

### Repository Configuration

Repository-specific settings are stored in `.wt.yaml` at the repository root. This file is required for `wt` to operate.

**Full schema:**

```yaml
version: 1                    # Required: config version

worktree_dir: worktrees       # Directory for worktrees (relative to repo root)
branch_pattern: "{name}"      # Pattern for new branch names
default_branch: main          # Branch for comparison (auto-detected if not set)

index:
  max: 20                     # Maximum worktree index (0 = no limit)

hooks:
  pre_create:
    - script: ./scripts/setup.sh
      env:
        CUSTOM_VAR: value
  post_create:
    - script: ./scripts/post-create.sh
  pre_delete:
    - script: ./scripts/cleanup.sh
  post_delete:
    - script: ./scripts/post-delete.sh
  info:
    - script: ./scripts/show-info.sh
```

#### worktree_dir

Directory where worktrees are created, relative to the repository root.

| | |
|---|---|
| **Default** | `worktrees` |
| **Example** | `worktree_dir: .worktrees` |

#### branch_pattern

Pattern used to generate branch names when creating worktrees.

| | |
|---|---|
| **Default** | `{name}` |
| **Variables** | `{name}` - worktree name |
| **Example** | `branch_pattern: "feature/{name}"` |

#### default_branch

Branch used for comparison in `list`, `cleanup`, and `delete` safety checks.

| | |
|---|---|
| **Default** | Auto-detected from remote HEAD, or `main` |
| **Example** | `default_branch: develop` |

#### index.max

Maximum value for worktree indexes. Set to limit the range of `WT_INDEX` values.

| | |
|---|---|
| **Default** | `0` (no limit) |
| **Example** | `index: { max: 10 }` |

See [Worktree Index](HOOKS.md#worktree-index) for more information.

---

### User Configuration

User preferences are stored in `~/.config/wt/config.yaml`. These settings control how `wt list`, `wt cleanup`, and `wt delete` compare worktree branches.

**File structure:**

```yaml
# Global settings
remote: origin
fetch_interval: 5m

# Per-repo overrides (keyed by absolute repo path)
repos:
  /path/to/repo:
    remote: upstream
    fetch_interval: 10m
  /path/to/another/repo:
    remote: ""
    fetch_interval: never
```

**Precedence:** Per-repo settings override global settings.

#### remote

Remote to compare against when checking merge status.

| | |
|---|---|
| **Default** | `""` (empty, local comparison) |
| **Valid values** | Any remote name (e.g., `origin`, `upstream`) |

When set, `wt` compares worktree branches against `<remote>/<default_branch>` instead of the local branch.

**Example:**

```bash
# Compare against origin/main globally
wt config --global remote origin

# Use upstream for current repo
wt config remote upstream

# Disable remote comparison for current repo
wt config remote ""
```

**Affects:** [`wt list`](#wt-list), [`wt delete`](#wt-delete), [`wt cleanup`](#wt-cleanup), [`wt info`](#wt-info)

#### fetch_interval

Minimum time between fetches from the remote.

| | |
|---|---|
| **Default** | `5m` |
| **Valid values** | Duration (`5m`, `1h`), `0` (always fetch), `never` (disable) |

Only takes effect when [`remote`](#remote) is set.

**Example:**

```bash
# Fetch at most every 10 minutes
wt config --global fetch_interval 10m

# Always fetch (no caching)
wt config fetch_interval 0

# Never fetch (use stale data)
wt config fetch_interval never
```

**Affects:** [`wt list`](#wt-list), [`wt delete`](#wt-delete), [`wt cleanup`](#wt-cleanup), [`wt info`](#wt-info)
