package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/agarcher/wt/internal/config"
	"github.com/agarcher/wt/internal/git"
	"github.com/agarcher/wt/internal/hooks"
	"github.com/agarcher/wt/internal/userconfig"
	"github.com/spf13/cobra"
)

var (
	deleteForce      bool
	deleteKeepBranch bool
)

func init() {
	deleteCmd.Flags().BoolVarP(&deleteForce, "force", "f", false, "Force deletion even with uncommitted or unmerged changes")
	deleteCmd.Flags().BoolVarP(&deleteKeepBranch, "keep-branch", "k", false, "Keep the associated branch (default: delete it)")
	rootCmd.AddCommand(deleteCmd)
}

var deleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete a worktree",
	Long: `Delete a git worktree and its associated branch.

If no name is provided and you're currently inside a worktree,
that worktree will be deleted.

By default, deletion will fail if:
  - There are uncommitted changes (modified or untracked files)
  - There are commits not merged into the comparison branch

Use --force to override these safety checks.

By default, the associated git branch is also deleted.
Use --keep-branch to preserve it.`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeWorktreeNames,
	RunE:              runDelete,
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

	// Get index for hooks (before deletion cleans it up)
	if idx, err := git.GetWorktreeIndex(repoRoot, name); err == nil {
		env.Index = idx
	}

	// Safety checks (unless --force)
	if !deleteForce {
		var issues []string

		// Check for uncommitted changes (dirty files)
		hasChanges, err := git.HasUncommittedChanges(worktreePath)
		if err != nil {
			return fmt.Errorf("failed to check for changes: %w", err)
		}
		if hasChanges {
			issues = append(issues, "has uncommitted changes (modified or untracked files)")
		}

		// Check for unmerged commits (commits ahead of comparison ref)
		// Load user configuration for fetch/remote settings
		userCfg, _ := userconfig.Load()

		// Determine remote for this repo (empty = local comparison)
		remote := userCfg.GetRemoteForRepo(repoRoot)

		// Determine comparison branch from repo config, or auto-detect
		comparisonBranch := cfg.DefaultBranch
		if comparisonBranch == "" {
			comparisonBranch, _ = git.GetDefaultBranch(repoRoot)
			if comparisonBranch == "" {
				comparisonBranch = "main" // Ultimate fallback
			}
		}

		// Build comparison ref based on whether remote is configured
		var comparisonRef string
		if remote != "" {
			// Remote comparison mode - fetch first if enabled
			remoteRef := remote + "/" + comparisonBranch

			fetchInterval := userCfg.GetFetchIntervalForRepo(repoRoot)
			if fetchInterval != userconfig.FetchIntervalNever {
				lastFetch, _ := git.GetLastFetchTime(repoRoot, remote)
				timeSinceLastFetch := time.Since(lastFetch)

				if fetchInterval > 0 && timeSinceLastFetch < fetchInterval {
					// Skip fetch - within interval
					cmd.PrintErrf("Skipping fetch (last fetch %s ago)\n", formatDuration(timeSinceLastFetch))
				} else {
					if err := git.FetchRemoteQuiet(repoRoot, remote); err != nil {
						cmd.PrintErrf("Warning: failed to fetch from %s: %v\n", remote, err)
					} else {
						_ = git.SetLastFetchTime(repoRoot, remote)
					}
				}
			}

			// Verify the remote ref exists, fall back to local if not
			if git.RefExists(repoRoot, remoteRef) {
				comparisonRef = remoteRef
			} else {
				comparisonRef = comparisonBranch
			}
		} else {
			// Local comparison mode (default)
			comparisonRef = comparisonBranch
		}

		ahead, _, _ := git.GetCommitsAheadBehind(repoRoot, worktreePath, comparisonRef)
		if ahead > 0 {
			if ahead == 1 {
				issues = append(issues, fmt.Sprintf("has 1 commit not merged into %s", comparisonRef))
			} else {
				issues = append(issues, fmt.Sprintf("has %d commits not merged into %s", ahead, comparisonRef))
			}
		}

		if len(issues) > 0 {
			cmd.PrintErrf("Error: cannot delete worktree %q:\n", name)
			for _, issue := range issues {
				cmd.PrintErrf("  - %s\n", issue)
			}
			cmd.PrintErrln("\nUse --force to delete anyway.")
			return fmt.Errorf("worktree has uncommitted or unmerged changes")
		}
	}

	// Check if user is in the worktree being deleted
	cwd, _ := os.Getwd()
	inDeletedWorktree := strings.HasPrefix(cwd, worktreePath)

	// Run pre-delete hooks
	if err := hooks.RunPreDelete(cfg, env); err != nil {
		if !deleteForce {
			return fmt.Errorf("pre-delete hook failed: %w", err)
		}
		cmd.Printf("Warning: pre-delete hook failed: %v\n", err)
	}

	// Delete the worktree
	cmd.Printf("Deleting worktree %q...\n", name)
	if err := git.RemoveWorktree(repoRoot, worktreePath, deleteForce); err != nil {
		return fmt.Errorf("failed to delete worktree: %w", err)
	}

	// Delete the branch unless --keep-branch is specified
	if !deleteKeepBranch && branch != "" {
		cmd.Printf("Deleting branch %q...\n", branch)
		if err := git.DeleteBranch(repoRoot, branch, deleteForce); err != nil {
			cmd.Printf("Warning: failed to delete branch: %v\n", err)
		}
	}

	// Run post-delete hooks
	if err := hooks.RunPostDelete(cfg, env); err != nil {
		cmd.Printf("Warning: post-delete hook failed: %v\n", err)
	}

	cmd.Printf("Worktree %q deleted successfully\n", name)

	// If user was in the deleted worktree, help them navigate back
	if inDeletedWorktree {
		if cdFile := os.Getenv("WT_CD_FILE"); cdFile != "" {
			// Shell wrapper mode: write path to file for cd
			_ = os.WriteFile(cdFile, []byte(repoRoot+"\n"), 0600)
		} else {
			// Direct invocation: print helpful message
			cmd.Printf("\nRun `cd %s` to return to the repository root\n", repoRoot)
		}
	}

	return nil
}

func confirmAction(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)
	// Flush stdout to ensure all previous output is visible before prompting
	_ = os.Stdout.Sync()
	fmt.Printf("%s [y/N] ", prompt)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}
