# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test

```bash
make build    # Build for current platform (outputs to build/wt)
make test     # Run tests with verbose output
make lint     # Run go vet and golangci-lint
```

Run a single test:
```bash
go test -v -run TestName ./internal/commands/
```

## Documentation

Keep docs current when changing features. Structure:

- **README.md** - High-level overview, installation, quick start, command table with links to docs/
- **docs/USAGE.md** - Comprehensive command reference and configuration options
- **docs/HOOKS.md** - Hook types, environment variables, worktree index, examples
- **docs/ARCHITECTURE.md** - Technical implementation details for contributors
- **CONTRIBUTING.md** - Build/test instructions, links to architecture

When modifying commands or config, update the corresponding docs/USAGE.md section.

## Package Structure

- **cmd/wt/main.go** - Entry point
- **internal/commands/** - Cobra command implementations
- **internal/config/** - `.wt.yaml` repository configuration
- **internal/userconfig/** - `~/.config/wt/config.yaml` user preferences
- **internal/git/** - Git worktree operations wrapper
- **internal/hooks/** - Lifecycle hook execution
- **internal/shell/** - Shell integration generators (zsh/bash/fish)

## Key Design Patterns

**Shell Integration**: Binary outputs paths to stdout; shell wrapper function (`wt init <shell>`) handles `cd`. Required because subprocesses cannot change parent shell directory.

**Repository Detection**: `.git` as file = worktree, `.git` as directory = main repo.

**Stdout vs Stderr**: Paths and listings go to stdout (for shell parsing). Messages and errors go to stderr via `cmd.Println()`.

**Hook Environment**: All hooks receive `WT_NAME`, `WT_PATH`, `WT_BRANCH`, `WT_REPO_ROOT`, `WT_WORKTREE_DIR`, `WT_INDEX`.

**Config Layering**: User config (`~/.config/wt/config.yaml`) has global defaults with per-repo overrides keyed by repo path.

## Git Workflow

**Never squash or rebase on merge.** Always use `gh pr merge --merge`.
