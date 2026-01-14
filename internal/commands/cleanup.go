package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	status *git.WorktreeStatus
}

func runCleanup(cmd *cobra.Command, args []string) error {
	// Setup comparison context (prints repo root, fetches if configured, prints comparison ref)
	setup, err := SetupCompare(cmd)
	if err != nil {
		return err
	}

	// Get all worktrees
	worktrees, err := git.ListWorktrees(setup.RepoRoot)
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	worktreesDir := filepath.Join(setup.RepoRoot, setup.Config.WorktreeDir)

	// Get merged branches cache for efficiency
	mergedCache, err := git.GetMergedBranches(setup.RepoRoot, setup.ComparisonRef)
	if err != nil {
		cmd.Printf("Warning: could not get merged branches: %v\n", err)
	}

	// Find candidates for cleanup
	var candidates []cleanupCandidate

	for _, wt := range worktrees {
		// Skip the main worktree
		if wt.Path == setup.RepoRoot {
			continue
		}

		// Check if this worktree is in our managed directory
		if !strings.HasPrefix(wt.Path, worktreesDir) {
			continue
		}

		// Get worktree name
		name := git.GetWorktreeName(setup.RepoRoot, wt.Path, setup.Config.WorktreeDir)

		// Skip if no branch (detached HEAD)
		if wt.Branch == "" {
			continue
		}

		// Get full worktree status
		status, err := git.GetWorktreeStatus(setup.RepoRoot, wt.Path, name, wt.Branch, setup.ComparisonRef, mergedCache)
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
				status: status,
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

	// Calculate column widths based on content
	nameWidth := len("NAME")
	branchWidth := len("BRANCH")
	for _, c := range candidates {
		if len(c.name) > nameWidth {
			nameWidth = len(c.name)
		}
		if len(c.branch) > branchWidth {
			branchWidth = len(c.branch)
		}
	}

	// Print header and rows with dynamic widths
	_, _ = fmt.Fprintf(out, "  %-*s  %-*s  %s\n", nameWidth, "NAME", branchWidth, "BRANCH", "STATUS")
	for _, c := range candidates {
		statusStr := FormatCompactStatus(c.status)
		_, _ = fmt.Fprintf(out, "  %-*s  %-*s  %s\n", nameWidth, c.name, branchWidth, c.branch, statusStr)
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

	// Check if user is in any of the worktrees being deleted
	cwd, _ := os.Getwd()
	inDeletedWorktree := false
	for _, c := range candidates {
		if strings.HasPrefix(cwd, c.path) {
			inDeletedWorktree = true
			break
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
			RepoRoot:    setup.RepoRoot,
			WorktreeDir: setup.Config.WorktreeDir,
		}

		// Get index for hooks
		if idx, err := git.GetWorktreeIndex(setup.RepoRoot, c.name); err == nil {
			env.Index = idx
		}

		// Run pre-delete hooks
		if err := hooks.RunPreDelete(setup.Config, env); err != nil {
			if !cleanupForce {
				cmd.Printf("Skipping %s: pre-delete hook failed: %v\n", c.name, err)
				continue
			}
			cmd.Printf("Warning: pre-delete hook failed for %s: %v\n", c.name, err)
		}

		// Delete the worktree
		cmd.Printf("Deleting worktree %q...\n", c.name)
		if err := git.RemoveWorktree(setup.RepoRoot, c.path, cleanupForce); err != nil {
			cmd.Printf("Error: failed to delete %s: %v\n", c.name, err)
			continue
		}

		// Delete the branch unless --keep-branch is specified
		if !cleanupKeepBranch && c.branch != "" {
			cmd.Printf("Deleting branch %q...\n", c.branch)
			if err := git.DeleteBranch(setup.RepoRoot, c.branch, cleanupForce); err != nil {
				cmd.Printf("Warning: failed to delete branch %s: %v\n", c.branch, err)
			}
		}

		// Run post-delete hooks
		if err := hooks.RunPostDelete(setup.Config, env); err != nil {
			cmd.Printf("Warning: post-delete hook failed for %s: %v\n", c.name, err)
		}

		deleted++
	}

	cmd.Printf("Cleaned up %d worktree(s)\n", deleted)

	// If user was in a deleted worktree, help them navigate back
	if inDeletedWorktree && deleted > 0 {
		if cdFile := os.Getenv("WT_CD_FILE"); cdFile != "" {
			// Shell wrapper mode: write path to file for cd
			_ = os.WriteFile(cdFile, []byte(setup.RepoRoot+"\n"), 0600)
		} else {
			// Direct invocation: print helpful message
			cmd.Printf("\nRun `cd %s` to return to the repository root\n", setup.RepoRoot)
		}
	}

	return nil
}
