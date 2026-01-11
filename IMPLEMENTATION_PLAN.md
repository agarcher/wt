# Worktree Tool - Implementation Plan

A cross-platform, brew-installable CLI for managing git worktrees with lifecycle hooks.

## Current Status

| Phase | Status | Notes |
|-------|--------|-------|
| Phase 1: Core Binary | ✅ Complete | All commands working |
| Phase 2: Shell Integration | ✅ Complete | Functions and completions done |
| Phase 3: Hook System | ✅ Complete | Full lifecycle hooks |
| Phase 4: Distribution | ✅ Complete | CI/CD and Homebrew tap ready |
| Phase 5: Documentation | ✅ Complete | README with examples |

## Overview

**Name options:** `wt`, `worktree`, `gw` (git worktree), `grove` (worktrees are like a grove of branches)

**Architecture:** Go binary + shell integration (like direnv, zoxide, starship)

```
┌─────────────────────────────────────────────────────────────┐
│                        User Shell                           │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  Shell Function (from `wt init zsh`)                │   │
│  │  - Wraps binary calls                               │   │
│  │  - Handles `cd` based on binary output              │   │
│  │  - Passes through to PATH when not in wt repo       │   │
│  └──────────────────────┬──────────────────────────────┘   │
└─────────────────────────┼───────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                     Go Binary (`wt`)                        │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │
│  │   Commands  │  │   Config    │  │   Hook Executor     │ │
│  │  - create   │  │  - .wt.yaml │  │  - pre_create       │ │
│  │  - delete   │  │  - global   │  │  - post_create      │ │
│  │  - list     │  │             │  │  - pre_delete       │ │
│  │  - cd       │  │             │  │  - post_delete      │ │
│  │  - init     │  │             │  │  - pre_cleanup      │ │
│  └─────────────┘  └─────────────┘  └─────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

---

## Phase 1: Core Binary Foundation ✅

### 1.1 Project Setup ✅

- Initialize Go module: `github.com/agarcher/wt`
- Set up directory structure:
  ```
  wt/
  ├── cmd/
  │   └── wt/
  │       └── main.go
  ├── internal/
  │   ├── commands/      # CLI command implementations
  │   ├── config/        # Config file parsing
  │   ├── git/           # Git operations wrapper
  │   ├── hooks/         # Hook execution engine
  │   └── shell/         # Shell integration generators
  ├── examples/
  │   └── hooks/         # Example hook scripts for common use cases
  ├── scripts/
  │   └── completions/   # Shell completions
  ├── go.mod
  ├── go.sum
  ├── Makefile
  └── README.md
  ```
- Add dependencies:
  - `github.com/spf13/cobra` - CLI framework
  - `github.com/spf13/viper` - Configuration
  - `gopkg.in/yaml.v3` - YAML parsing

### 1.2 Configuration System ✅

**Repo-level config (`.wt.yaml`):** ✅ Implemented
```yaml
# .wt.yaml - placed in repository root
version: 1

# Where worktrees are stored (relative to repo root)
worktree_dir: worktrees

# Branch naming pattern (optional)
# Available variables: {name}, {date}, {user}
branch_pattern: "{name}"

# Lifecycle hooks (all optional)
hooks:
  # Runs before worktree is created
  # If exits non-zero, creation is aborted
  pre_create:
    - script: ./scripts/wt/pre-create.sh

  # Runs after worktree is created, in the new worktree directory
  post_create:
    - script: ./scripts/wt/post-create.sh
    - script: ./scripts/wt/setup-ports.sh
      # Pass worktree metadata as environment variables
      env:
        CUSTOM_VAR: "value"

  # Runs before worktree deletion
  # If exits non-zero, deletion is aborted (unless --force)
  pre_delete:
    - script: ./scripts/wt/pre-delete.sh

  # Runs after worktree is deleted, in repo root
  post_delete:
    - script: ./scripts/wt/post-delete.sh

# Variables available to all hooks as environment variables:
# WT_NAME          - worktree name
# WT_PATH          - full path to worktree
# WT_BRANCH        - git branch name
# WT_REPO_ROOT     - main repository root
# WT_WORKTREE_DIR  - worktree directory name (e.g., "worktrees")
```

**Global config (`~/.config/wt/config.yaml`):** ❌ Not implemented (deferred - repo-level config sufficient)
```yaml
# Global defaults
defaults:
  worktree_dir: worktrees

# Shell to use for hook execution
shell: /bin/bash
```

### 1.3 Core Commands ✅

**`wt init <shell>`**
- Outputs shell-specific initialization code
- Shells: `zsh`, `bash`, `fish`
- User adds `eval "$(wt init zsh)"` to their rc file

**`wt create <name> [--branch <branch>]`**
- Creates git worktree at `<worktree_dir>/<name>`
- Creates branch using pattern from config (default: same as name)
- Runs pre_create hooks (abort if any fail)
- Runs post_create hooks (in new worktree dir)
- **Output protocol:** Last line is the worktree path for shell wrapper to `cd`

**`wt delete [name] [--force]`**
- Auto-detects current worktree if name omitted
- Runs pre_delete hooks (abort if fail, unless --force)
- Removes worktree via `git worktree remove`
- Optionally deletes branch
- Runs post_delete hooks

**`wt list`**
- Lists all worktrees with status
- Shows: name, branch, path, uncommitted changes, unpushed commits

**`wt cd <name>`**
- **Output:** Path to worktree (shell wrapper handles actual cd)

**`wt exit`**
- **Output:** Path to repo root (shell wrapper handles actual cd)

**`wt root`**
- **Output:** Path to repo root (for scripting)

**`wt cleanup [--dry-run] [--force] [--keep-branch]`** ✅ Implemented
- Finds worktrees eligible for cleanup (branches merged into main/master)
- Runs pre_delete hooks for each
- Removes worktrees and their branches by default
- Use `--keep-branch` to preserve branches

---

## Phase 2: Shell Integration ✅

### 2.1 Shell Function Generator ✅

The `wt init <shell>` command outputs shell functions that:

1. **Detect context:** Check if in a repo with `.wt.yaml`
2. **Handle cd:** Parse binary output, execute `cd` if path returned
3. **Pass-through:** If not in wt repo, fall through to PATH (non-intrusive)

**Generated zsh function (simplified):**
```zsh
wt() {
  # Check if we're in a wt-enabled repo
  local repo_root
  repo_root=$(git rev-parse --show-toplevel 2>/dev/null)

  if [[ -z "$repo_root" ]] || [[ ! -f "$repo_root/.wt.yaml" ]]; then
    # Not in a wt repo - pass through to any wt in PATH
    command wt "$@"
    return $?
  fi

  # Commands that need cd handling
  case "$1" in
    create|cd)
      local output
      output=$(command wt "$@")
      local exit_code=$?

      if [[ $exit_code -eq 0 ]]; then
        # Print all but last line
        echo "$output" | sed '$d'
        # cd to path on last line
        local target=$(echo "$output" | tail -1)
        [[ -d "$target" ]] && cd "$target"
      else
        echo "$output"
      fi
      return $exit_code
      ;;
    exit)
      local target
      target=$(command wt root)
      [[ -d "$target" ]] && cd "$target"
      ;;
    *)
      command wt "$@"
      ;;
  esac
}
```

### 2.2 Shell Completions ✅

Generate completions for:
- zsh (place in `_wt`)
- bash (place in `wt.bash`)
- fish (place in `wt.fish`)

Completions include:
- Complete subcommands
- Complete worktree names for `cd`, `delete`
- Complete shell types for `init`
- Complete branch names for `create --branch`
- Complete options/flags

Use `wt completion <shell>` to generate completion scripts.

---

## Phase 3: Hook System ✅

### 3.1 Hook Execution Engine ✅

**Environment variables passed to all hooks:**
```
WT_NAME=feature-x
WT_PATH=/path/to/repo/worktrees/feature-x
WT_BRANCH=feature-x
WT_REPO_ROOT=/path/to/repo
WT_WORKTREE_DIR=worktrees
```

**Hook execution:**
1. Scripts run via configured shell (default: `/bin/bash`)
2. Working directory set appropriately (repo root or worktree)
3. stdout/stderr passed through to user
4. Exit code determines success/failure

### 3.2 Built-in Hook Scripts (Examples) ✅

Provide example hook scripts users can copy:

**Copy environment files (`examples/hooks/copy-env.sh`):**
```bash
#!/bin/bash
# Copy .env files from main repo
for env_file in "$WT_REPO_ROOT"/.env*; do
  [[ -f "$env_file" ]] && cp "$env_file" "$WT_PATH/"
done
```

**Safety check (`examples/hooks/pre-delete.sh`):**
```bash
#!/bin/bash
cd "$WT_PATH"

# Check for uncommitted changes
if [[ -n $(git status --porcelain) ]]; then
  echo "Warning: Uncommitted changes in $WT_NAME"
  read -p "Continue? [y/N] " -n 1 -r
  echo
  [[ ! $REPLY =~ ^[Yy]$ ]] && exit 1
fi
```

---

## Phase 4: Distribution ⚠️

### 4.1 Build System ✅

**Makefile targets:**
```makefile
build:           # Build for current platform
build-all:       # Build for all platforms (darwin/linux, amd64/arm64)
test:            # Run tests
lint:            # Run linters
install:         # Install to /usr/local/bin
release:         # Create release artifacts
```

### 4.2 Homebrew Formula ✅ Complete

**Setup requirements:**

1. Create tap repository: `github.com/agarcher/homebrew-tap`
2. Create `Formula/` directory in the tap repo
3. Add `HOMEBREW_TAP_TOKEN` secret to the main repo (GitHub PAT with repo access to tap)

**Installation:**
```bash
brew install agarcher/tap/wt
```

Or tap first:
```bash
brew tap agarcher/tap
brew install wt
```

**Formula location:** `Formula/wt.rb` (template in main repo, auto-updated in tap on release)

The release workflow automatically:
- Builds binaries for darwin/linux on amd64/arm64
- Computes SHA256 checksums
- Updates the tap formula with new version and checksums
- Skips tap update for pre-release versions (-rc, -beta, -alpha)

### 4.3 GitHub Actions CI/CD ✅

- Run tests on PRs
- Build binaries on release tags
- Publish to GitHub Releases
- Update Homebrew tap

---

## Phase 5: Documentation ✅

### 5.1 README.md ✅

- Quick start guide
- Installation (brew, manual)
- Configuration reference
- Hook examples
- Comparison with alternatives

### 5.2 Example Configurations ⚠️ Partial

Provide starter configs for common use cases:
- Basic worktree management
- Environment file copying
- Monorepo setups
- Port-shifted development (requires WT_INDEX - see Future Enhancements)

---

## Implementation Order

1. ✅ **Core binary skeleton** - cobra CLI, basic command structure
2. ✅ **Config parsing** - .wt.yaml (global config deferred)
3. ✅ **Git operations** - worktree create/delete/list wrappers
4. ✅ **Shell init** - `wt init zsh/bash/fish` with cd handling
5. ✅ **Hook system** - execution engine, environment variables
6. ✅ **List command** - with status info (uncommitted, unpushed)
7. ✅ **Cleanup command** - smart worktree cleanup
8. ✅ **Completions** - shell completions generation
9. ✅ **Build/release** - Makefile, GitHub Actions CI/CD
10. ✅ **Homebrew formula** - tap setup, formula, auto-update on release
11. ✅ **Documentation** - README, examples

---

## Design Decisions

### Why Go over Rust?
- Faster development cycle for this scope
- Excellent CLI libraries (cobra is industry standard)
- Simpler cross-compilation
- Good enough performance (not CPU-bound)
- Used by similar tools (direnv, gh, etc.)

### Why shell wrapper approach?
- **The problem:** Subprocesses cannot change parent's working directory
- **The solution:** Binary outputs path, shell function does `cd`
- **Precedent:** direnv, zoxide, starship all use this pattern

### Why repo-local config vs global?
- Each repo can have different needs (port schemes, custom hooks)
- No namespace pollution when not in a configured repo
- Teams can version control their `.wt.yaml`

### Why not direnv dependency?
- Reduces dependencies
- More control over shell integration
- direnv users can still use both (they don't conflict)

---

## Future Enhancements

### WT_INDEX: Stable Numeric Worktree Identifier

*Deferred to post-v1.0 release*

Add `WT_INDEX` environment variable providing a stable, reusable numeric identifier for each worktree, useful for port offsets and other resource allocation.

**Storage location:** `.git/worktrees/<name>/wt-index`

This leverages git's existing worktree metadata directory:
- Doesn't pollute the worktree (no untracked files in working directory)
- Gets cleaned up automatically by `git worktree remove`
- Is discoverable by scanning `.git/worktrees/*/wt-index`

**Allocation algorithm:**

```go
func GetWorktreeIndex(repoRoot, worktreeName string) (int, error) {
    indexPath := filepath.Join(repoRoot, ".git", "worktrees", worktreeName, "wt-index")
    data, err := os.ReadFile(indexPath)
    if err != nil {
        return 0, err
    }
    return strconv.Atoi(strings.TrimSpace(string(data)))
}

func AllocateIndex(repoRoot string, maxIndex int) (int, error) {
    used := make(map[int]bool)

    // Scan all existing worktree indexes
    entries, _ := os.ReadDir(filepath.Join(repoRoot, ".git", "worktrees"))
    for _, entry := range entries {
        if idx, err := GetWorktreeIndex(repoRoot, entry.Name()); err == nil {
            used[idx] = true
        }
    }

    // Find lowest unused (starting at 1, reserve 0 for main repo)
    for i := 1; ; i++ {
        // Check upper bound if configured
        if maxIndex > 0 && i > maxIndex {
            return 0, fmt.Errorf("no available index: all indexes 1-%d are in use", maxIndex)
        }
        if !used[i] {
            return i, nil
        }
    }
}
```

**Lifecycle:**

| Event | Action |
|-------|--------|
| `wt create` | Scan `.git/worktrees/*/wt-index`, allocate lowest unused, write to new worktree |
| `wt delete` | Git cleans up `.git/worktrees/<name>/` including index file automatically |
| `wt list` | Read indexes to display alongside worktree info |

**Example scenario:**
```
wt create feature-a  → index 1
wt create feature-b  → index 2
wt create feature-c  → index 3
wt delete feature-b  → index 2 freed
wt create feature-d  → index 2 (reused!)
```

**Edge cases:**

| Scenario | Handling |
|----------|----------|
| Manual worktree deletion | `git worktree prune` cleans up orphaned metadata, index becomes available |
| Corrupted/missing index file | Allocate new index, warn user |
| Main worktree | Index 0 reserved, allocation starts at 1 |
| Max index reached | Error with message: "no available index: all indexes 1-N are in use" |
| Concurrent creates | File locking on `.git/wt-index.lock` (future enhancement if needed) |

**Config options:**

```yaml
# .wt.yaml
index:
  # Maximum allowed index (optional)
  # Prevents runaway allocation that could cause port conflicts
  # If unset or 0, no upper bound is enforced
  max: 20
```

**Example hook using WT_INDEX (`examples/hooks/setup-ports.sh`):**
```bash
#!/bin/bash
# Calculate port offset based on WT_INDEX
PORT_OFFSET=$((WT_INDEX * 10))

# Write port configuration
cat > "$WT_PATH/.wt-ports.env" << EOF
PORT_OFFSET=$PORT_OFFSET
VITE_PORT=$((5173 + PORT_OFFSET))
API_PORT=$((3000 + PORT_OFFSET))
EOF

echo "Configured ports with offset $PORT_OFFSET"
```

---

## Open Questions

1. **Command name:** `wt` is short but common. Alternatives: `grove`, `gw`, `worktree`
2. ~~**Branch cleanup:** Should `wt delete` also delete the branch by default?~~ **Resolved:** Yes, both `wt delete` and `wt cleanup` delete the branch by default. Use `--keep-branch` to preserve it.
3. **Remote tracking:** Should `wt create` automatically push and track remote branch?
