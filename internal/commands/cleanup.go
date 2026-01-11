package commands

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/agarcher/wt/internal/config"
	"github.com/agarcher/wt/internal/git"
	"github.com/agarcher/wt/internal/hooks"
	"github.com/spf13/cobra"
)

var (
	cleanupDryRun     bool
	cleanupForce      bool
	cleanupKeepBranch bool
)

func init() {
	cleanupCmd.Flags().BoolVarP(&cleanupDryRun, "dry-run", "n", false, "Show what would be deleted without deleting")
	cleanupCmd.Flags().BoolVarP(&cleanupForce, "force", "f", false, "Skip confirmation prompts")
	cleanupCmd.Flags().BoolVarP(&cleanupKeepBranch, "keep-branch", "k", false, "Keep the associated branches (default: delete them)")
	rootCmd.AddCommand(cleanupCmd)
}

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Clean up merged worktrees",
	Long: `Find and remove worktrees whose branches have been merged.

This command identifies worktrees that are eligible for cleanup:
- Branches that have been merged into the default branch (main/master)

By default, both the worktree and its associated branch are deleted.

Use --dry-run to see what would be deleted without actually deleting.
Use --force to skip confirmation prompts.
Use --keep-branch to preserve the associated git branches.`,
	RunE: runCleanup,
}

// cleanupCandidate represents a worktree eligible for cleanup
type cleanupCandidate struct {
	name   string
	path   string
	branch string
	reason string
}

func runCleanup(cmd *cobra.Command, args []string) error {
	// Find the main repository root
	repoRoot, err := config.GetMainRepoRoot()
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	// Load configuration
	cfg, err := config.Load(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get default branch
	defaultBranch, err := git.GetDefaultBranch(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to determine default branch: %w", err)
	}

	// Get all worktrees
	worktrees, err := git.ListWorktrees(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	worktreesDir := filepath.Join(repoRoot, cfg.WorktreeDir)

	// Get merged branches cache for efficiency
	mergedCache, err := git.GetMergedBranches(repoRoot, defaultBranch)
	if err != nil {
		cmd.Printf("Warning: could not get merged branches: %v\n", err)
	}

	// Find candidates for cleanup
	var candidates []cleanupCandidate

	for _, wt := range worktrees {
		// Skip the main worktree
		if wt.Path == repoRoot {
			continue
		}

		// Check if this worktree is in our managed directory
		if !strings.HasPrefix(wt.Path, worktreesDir) {
			continue
		}

		// Get worktree name
		name := git.GetWorktreeName(repoRoot, wt.Path, cfg.WorktreeDir)

		// Skip if no branch (detached HEAD)
		if wt.Branch == "" {
			continue
		}

		// Get full worktree status
		status, err := git.GetWorktreeStatus(repoRoot, wt.Path, name, wt.Branch, defaultBranch, mergedCache)
		if err != nil {
			cmd.Printf("Warning: could not get status for %s: %v\n", name, err)
			continue
		}

		// Skip worktrees with uncommitted changes
		if status.HasUncommittedChanges {
			continue
		}

		// Skip new worktrees (no commits yet - still being worked on)
		if status.IsNew {
			continue
		}

		// Skip worktrees with commits ahead of main (unmerged work)
		if status.CommitsAhead > 0 {
			continue
		}

		// Only cleanup if merged
		if status.IsMerged {
			candidates = append(candidates, cleanupCandidate{
				name:   name,
				path:   wt.Path,
				branch: wt.Branch,
				reason: fmt.Sprintf("merged into %s", defaultBranch),
			})
		}
	}

	// No candidates found
	if len(candidates) == 0 {
		cmd.Println("No worktrees eligible for cleanup")
		return nil
	}

	// Display candidates
	out := cmd.OutOrStdout()
	_, _ = fmt.Fprintln(out, "Worktrees eligible for cleanup:")
	_, _ = fmt.Fprintln(out)
	for _, c := range candidates {
		_, _ = fmt.Fprintf(out, "  %-20s  %-30s  (%s)\n", c.name, c.branch, c.reason)
	}
	_, _ = fmt.Fprintln(out)

	// Dry run - just show what would be deleted
	if cleanupDryRun {
		cmd.Printf("Would delete %d worktree(s)", len(candidates))
		if !cleanupKeepBranch {
			cmd.Print(" and their branches")
		}
		cmd.Println()
		return nil
	}

	// Confirm deletion
	if !cleanupForce {
		cmd.Printf("Delete %d worktree(s)", len(candidates))
		if !cleanupKeepBranch {
			cmd.Print(" and their branches")
		}
		cmd.Println("?")
		if !confirmAction("Proceed?") {
			return fmt.Errorf("aborted")
		}
	}

	// Delete each candidate
	var deleted int
	for _, c := range candidates {
		// Create hook environment
		env := &hooks.Env{
			Name:        c.name,
			Path:        c.path,
			Branch:      c.branch,
			RepoRoot:    repoRoot,
			WorktreeDir: cfg.WorktreeDir,
		}

		// Run pre-delete hooks
		if err := hooks.RunPreDelete(cfg, env); err != nil {
			if !cleanupForce {
				cmd.Printf("Skipping %s: pre-delete hook failed: %v\n", c.name, err)
				continue
			}
			cmd.Printf("Warning: pre-delete hook failed for %s: %v\n", c.name, err)
		}

		// Delete the worktree
		cmd.Printf("Deleting worktree %q...\n", c.name)
		if err := git.RemoveWorktree(repoRoot, c.path, cleanupForce); err != nil {
			cmd.Printf("Error: failed to delete %s: %v\n", c.name, err)
			continue
		}

		// Delete the branch unless --keep-branch is specified
		if !cleanupKeepBranch && c.branch != "" {
			cmd.Printf("Deleting branch %q...\n", c.branch)
			if err := git.DeleteBranch(repoRoot, c.branch, cleanupForce); err != nil {
				cmd.Printf("Warning: failed to delete branch %s: %v\n", c.branch, err)
			}
		}

		// Run post-delete hooks
		if err := hooks.RunPostDelete(cfg, env); err != nil {
			cmd.Printf("Warning: post-delete hook failed for %s: %v\n", c.name, err)
		}

		deleted++
	}

	cmd.Printf("Cleaned up %d worktree(s)\n", deleted)
	return nil
}
