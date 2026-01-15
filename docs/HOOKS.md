# Hooks

Hooks are custom scripts that run at specific points in the worktree lifecycle. They enable automatic setup, cleanup, and custom status display.

## Overview

Hooks are configured in your repository's `.wt.yaml` file and receive standardized environment variables. Each hook type runs at a specific point and has different capabilities.

**Environment variables available to all hooks:**

| Variable | Description |
|----------|-------------|
| `WT_NAME` | Worktree name |
| `WT_PATH` | Absolute path to the worktree |
| `WT_BRANCH` | Git branch name |
| `WT_REPO_ROOT` | Absolute path to the main repository |
| `WT_WORKTREE_DIR` | Worktree directory name (e.g., `worktrees`) |
| `WT_INDEX` | Worktree index number (see [Worktree Index](#worktree-index)) |

## Hook Types

### pre_create

Runs **before** a worktree is created.

| | |
|---|---|
| **Working directory** | Repository root |
| **Can block creation** | Yes - if script exits non-zero, creation is aborted |

**Use cases:**
- Validate worktree or branch names
- Check prerequisites (disk space, permissions)
- Create parent directories

**Example:**

```bash
#!/bin/bash
# Validate worktree name format
if [[ ! "$WT_NAME" =~ ^[a-z0-9-]+$ ]]; then
    echo "Error: Worktree name must be lowercase alphanumeric with hyphens"
    exit 1
fi
```

**Triggered by:** [`wt create`](USAGE.md#wt-create)

---

### post_create

Runs **after** a worktree is created and ready.

| | |
|---|---|
| **Working directory** | Worktree directory |
| **Can block creation** | No - failure produces a warning only |

**Use cases:**
- Install dependencies (`npm install`, `pip install`)
- Copy environment files (`.env`, `.env.local`)
- Configure ports based on `WT_INDEX`
- Set up database connections

**Example:**

```bash
#!/bin/bash
# Copy .env files from main repo
for env_file in "$WT_REPO_ROOT"/.env*; do
    if [[ -f "$env_file" ]]; then
        cp "$env_file" "$WT_PATH/"
        echo "Copied $(basename "$env_file")"
    fi
done
```

**Triggered by:** [`wt create`](USAGE.md#wt-create)

---

### pre_delete

Runs **before** a worktree is deleted.

| | |
|---|---|
| **Working directory** | Worktree directory |
| **Can block deletion** | Yes - unless `--force` is used |

**Use cases:**
- Warn about uncommitted changes
- Check for unpushed commits
- Clean up related resources (containers, databases)
- Interactive confirmation prompts

**Example:**

```bash
#!/bin/bash
# Warn about uncommitted changes
if [[ -n $(git status --porcelain) ]]; then
    echo "Warning: '$WT_NAME' has uncommitted changes"
    read -p "Delete anyway? [y/N] " -n 1 -r
    echo
    [[ $REPLY =~ ^[Yy]$ ]] || exit 1
fi
```

**Triggered by:** [`wt delete`](USAGE.md#wt-delete), [`wt cleanup`](USAGE.md#wt-cleanup)

---

### post_delete

Runs **after** a worktree is deleted.

| | |
|---|---|
| **Working directory** | Repository root |
| **Can block deletion** | No - failure produces a warning only |

**Use cases:**
- Clean up external resources
- Update monitoring or documentation
- Notify other systems

**Example:**

```bash
#!/bin/bash
# Log deletion
echo "$(date): Deleted worktree $WT_NAME" >> "$WT_REPO_ROOT/.wt-log"
```

**Triggered by:** [`wt delete`](USAGE.md#wt-delete), [`wt cleanup`](USAGE.md#wt-cleanup)

---

### info

Runs during `wt info` and `wt list -v` to display custom information.

| | |
|---|---|
| **Working directory** | Worktree directory |
| **Output** | stdout is captured and displayed |
| **Can block** | No - failure is silent |

**Use cases:**
- Display dev server URLs
- Show database connection info
- Display deployment status
- Show resource allocation

**Output format:**
- Lines matching `Key: value` format are aligned with built-in fields
- Other output is displayed as-is below the key-value section

**Example:**

```bash
#!/bin/bash
# Display dev server URL based on worktree index
if [ -n "$WT_INDEX" ]; then
    PORT=$((5173 + WT_INDEX * 10))
    echo "URL: http://localhost:$PORT"
fi
```

**Triggered by:** [`wt info`](USAGE.md#wt-info), [`wt list -v`](USAGE.md#wt-list)

---

## Configuration

Configure hooks in your `.wt.yaml` file:

```yaml
version: 1
worktree_dir: worktrees

hooks:
  pre_create:
    - script: ./scripts/validate.sh

  post_create:
    - script: ./scripts/copy-env.sh
    - script: ./scripts/setup-ports.sh
      env:
        BASE_PORT: "5000"

  pre_delete:
    - script: ./scripts/pre-delete-check.sh

  post_delete:
    - script: ./scripts/cleanup.sh

  info:
    - script: ./scripts/show-info.sh
```

### Multiple hooks

You can specify multiple hooks for any event. They execute in order:

```yaml
hooks:
  post_create:
    - script: ./scripts/first.sh    # Runs first
    - script: ./scripts/second.sh   # Runs second
```

### Custom environment variables

Add custom environment variables to any hook:

```yaml
hooks:
  post_create:
    - script: ./scripts/setup.sh
      env:
        DATABASE_PREFIX: "dev_"
        LOG_LEVEL: "debug"
```

### Script paths

Scripts can be absolute or relative to the repository root:

```yaml
hooks:
  post_create:
    - script: ./scripts/setup.sh        # Relative
    - script: /usr/local/bin/custom.sh  # Absolute
```

Scripts must be executable (`chmod +x`).

---

## Worktree Index

Each worktree is assigned a stable numeric index starting at 1. The index provides a unique identifier useful for resource isolation.

### How it works

1. When a worktree is created, the next available index is allocated
2. The index is stored in `.git/worktrees/<name>/wt-index`
3. When a worktree is deleted, its index becomes available for reuse
4. The index is available to hooks via `WT_INDEX`

### Common use cases

**Port allocation:**

```bash
# Each worktree gets a unique port offset
PORT_OFFSET=$((WT_INDEX * 10))
VITE_PORT=$((5173 + PORT_OFFSET))
API_PORT=$((3000 + PORT_OFFSET))
```

**Database isolation:**

```bash
# Each worktree gets its own database
DATABASE_NAME="dev_${WT_NAME}"
```

**Container naming:**

```bash
# Unique container names per worktree
CONTAINER_NAME="app-${WT_INDEX}"
```

### Limiting the index range

You can limit the maximum index value in `.wt.yaml`:

```yaml
version: 1
worktree_dir: worktrees
index:
  max: 20  # Indexes limited to 1-20
```

This is useful when you have a fixed range of ports or resources.

See the [Vite port demo](../examples/vite-port-demo/) for a working example.

---

## Examples

Example hook scripts are available in [`examples/hooks/`](../examples/hooks/):

### copy-env.sh

Copies `.env*` files from the main repository to the worktree.

```bash
#!/bin/bash
for env_file in "$WT_REPO_ROOT"/.env*; do
    if [[ -f "$env_file" ]]; then
        cp "$env_file" "$WT_PATH/"
    fi
done
```

**Use as:** `post_create`

### setup-ports.sh

Generates a `.wt-ports.env` file with unique port assignments based on `WT_INDEX`.

```bash
#!/bin/bash
PORT_OFFSET=$((WT_INDEX * 10))
cat > "$WT_PATH/.wt-ports.env" << EOF
VITE_PORT=$((5173 + PORT_OFFSET))
API_PORT=$((3000 + PORT_OFFSET))
DB_PORT=$((5432 + PORT_OFFSET))
EOF
```

**Use as:** `post_create`

### pre-delete-check.sh

Warns about uncommitted changes and unpushed commits before deletion.

```bash
#!/bin/bash
if [[ -n $(git status --porcelain) ]]; then
    echo "Warning: Uncommitted changes"
    read -p "Delete anyway? [y/N] " -n 1 -r
    [[ $REPLY =~ ^[Yy]$ ]] || exit 1
fi
```

**Use as:** `pre_delete`

### show-info.sh

Displays the dev server URL based on worktree index.

```bash
#!/bin/bash
PORT=$((5173 + WT_INDEX * 10))
echo "URL: http://localhost:$PORT"
```

**Use as:** `info`

---

## Vite Port Demo

The [`examples/vite-port-demo/`](../examples/vite-port-demo/) directory contains a complete working example that demonstrates:

- Copying `node_modules` for instant setup
- Configuring unique Vite dev server ports per worktree
- Displaying the dev server URL via info hooks

See the [demo README](../examples/vite-port-demo/README.md) for setup instructions.
