# Maintainers Guide

This document contains information for maintainers of the `wt` repository.

## Merging Pull Requests

**Never squash or rebase on merge.** Always use merge commits:

```bash
gh pr merge --merge
```

This preserves the full commit history and makes it easier to generate release notes.

## Releasing

Version is tracked in the `VERSION` file. Releases should be done using Claude Code to automatically summarize merged PRs into release notes.

Run the `/release` command in Claude Code, which will:
1. Analyze commits since the last release
2. Propose a version bump (patch/minor/major) based on changes
3. Generate release notes from merged PRs
4. Execute the release after approval

Alternatively, release manually:

```bash
make release patch "Fix bug in cleanup"
make release minor "Add new feature"
make release major "Breaking change"
```

This bumps `VERSION`, commits, tags, and pushes. GitHub Actions then builds binaries and updates the Homebrew tap.
