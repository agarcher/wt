package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/agarcher/wt/internal/config"
	"github.com/agarcher/wt/internal/git"
	"github.com/agarcher/wt/internal/hooks"
	"github.com/spf13/cobra"
)

var (
	deleteForce        bool
	deleteDeleteBranch bool
)

func init() {
	deleteCmd.Flags().BoolVarP(&deleteForce, "force", "f", false, "Force deletion even with uncommitted changes")
	deleteCmd.Flags().BoolVarP(&deleteDeleteBranch, "delete-branch", "D", false, "Also delete the associated branch")
	rootCmd.AddCommand(deleteCmd)
}

var deleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete a worktree",
	Long: `Delete a git worktree.

If no name is provided and you're currently inside a worktree,
that worktree will be deleted.

By default, deletion will fail if there are uncommitted changes.
Use --force to override this check.

Use --delete-branch to also delete the associated git branch.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDelete,
}

func runDelete(cmd *cobra.Command, args []string) error {
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

	// Determine which worktree to delete
	var name string
	var worktreePath string

	if len(args) > 0 {
		name = args[0]
		worktreePath = filepath.Join(repoRoot, cfg.WorktreeDir, name)
	} else {
		// Auto-detect from current directory
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		worktreesDir := filepath.Join(repoRoot, cfg.WorktreeDir)
		if !strings.HasPrefix(cwd, worktreesDir) {
			return fmt.Errorf("not in a worktree (specify name or cd into a worktree)")
		}

		// Extract worktree name from path
		rel, err := filepath.Rel(worktreesDir, cwd)
		if err != nil {
			return fmt.Errorf("failed to determine worktree: %w", err)
		}
		parts := strings.Split(rel, string(filepath.Separator))
		name = parts[0]
		worktreePath = filepath.Join(worktreesDir, name)
	}

	// Check if worktree exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		return fmt.Errorf("worktree %q does not exist", name)
	}

	// Get branch name before deletion
	branch, _ := git.GetCurrentBranch(worktreePath)

	// Create hook environment
	env := &hooks.Env{
		Name:        name,
		Path:        worktreePath,
		Branch:      branch,
		RepoRoot:    repoRoot,
		WorktreeDir: cfg.WorktreeDir,
	}

	// Check for uncommitted changes (unless --force)
	if !deleteForce {
		hasChanges, err := git.HasUncommittedChanges(worktreePath)
		if err != nil {
			return fmt.Errorf("failed to check for changes: %w", err)
		}
		if hasChanges {
			fmt.Printf("Worktree %q has uncommitted changes.\n", name)
			if !confirmAction("Delete anyway?") {
				return fmt.Errorf("aborted")
			}
		}

		hasUnpushed, err := git.HasUnpushedCommits(worktreePath)
		if err == nil && hasUnpushed {
			fmt.Printf("Worktree %q has unpushed commits.\n", name)
			if !confirmAction("Delete anyway?") {
				return fmt.Errorf("aborted")
			}
		}
	}

	// Warn if user is in the worktree being deleted
	cwd, _ := os.Getwd()
	if strings.HasPrefix(cwd, worktreePath) {
		fmt.Println("Warning: You are currently in this worktree.")
		fmt.Println("After deletion, run 'wt exit' or 'cd' to another directory.")
	}

	// Run pre-delete hooks
	if err := hooks.RunPreDelete(cfg, env); err != nil {
		if !deleteForce {
			return fmt.Errorf("pre-delete hook failed: %w", err)
		}
		fmt.Printf("Warning: pre-delete hook failed: %v\n", err)
	}

	// Delete the worktree
	fmt.Printf("Deleting worktree %q...\n", name)
	if err := git.RemoveWorktree(repoRoot, worktreePath, deleteForce); err != nil {
		return fmt.Errorf("failed to delete worktree: %w", err)
	}

	// Delete the branch if requested
	if deleteDeleteBranch && branch != "" {
		fmt.Printf("Deleting branch %q...\n", branch)
		if err := git.DeleteBranch(repoRoot, branch, deleteForce); err != nil {
			fmt.Printf("Warning: failed to delete branch: %v\n", err)
		}
	}

	// Run post-delete hooks
	if err := hooks.RunPostDelete(cfg, env); err != nil {
		fmt.Printf("Warning: post-delete hook failed: %v\n", err)
	}

	fmt.Printf("Worktree %q deleted successfully\n", name)
	return nil
}

func confirmAction(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s [y/N] ", prompt)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}
