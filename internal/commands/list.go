package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/agarcher/wt/internal/config"
	"github.com/agarcher/wt/internal/git"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all worktrees",
	Long: `List all git worktrees with their status.

Shows each worktree with:
- Name
- Branch
- Uncommitted changes indicator
- Unpushed commits indicator`,
	RunE: runList,
}

func runList(cmd *cobra.Command, args []string) error {
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

	// Get all worktrees
	worktrees, err := git.ListWorktrees(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	// Get current directory to highlight current worktree
	cwd, _ := os.Getwd()
	worktreesDir := filepath.Join(repoRoot, cfg.WorktreeDir)

	// Collect managed worktrees (excluding main repo)
	type worktreeInfo struct {
		name          string
		branch        string
		statusStr     string
		currentMarker string
	}
	var managedWorktrees []worktreeInfo

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

		// Check status
		var status []string

		hasChanges, err := git.HasUncommittedChanges(wt.Path)
		if err == nil && hasChanges {
			status = append(status, "uncommitted")
		}

		hasUnpushed, err := git.HasUnpushedCommits(wt.Path)
		if err == nil && hasUnpushed {
			status = append(status, "unpushed")
		}

		// Build status string
		statusStr := ""
		if len(status) > 0 {
			statusStr = fmt.Sprintf("[%s]", strings.Join(status, ", "))
		}

		// Check if this is the current worktree
		currentMarker := "  "
		if strings.HasPrefix(cwd, wt.Path) {
			currentMarker = "* "
		}

		managedWorktrees = append(managedWorktrees, worktreeInfo{
			name:          name,
			branch:        wt.Branch,
			statusStr:     statusStr,
			currentMarker: currentMarker,
		})
	}

	// If no worktrees, display message and return
	if len(managedWorktrees) == 0 {
		cmd.Println("No worktrees")
		return nil
	}

	// Print header and worktrees
	cmd.Printf("  %-20s  %s\n", "NAME", "BRANCH")
	for _, wt := range managedWorktrees {
		if wt.statusStr != "" {
			cmd.Printf("%s%-20s  %-30s %s\n", wt.currentMarker, wt.name, wt.branch, wt.statusStr)
		} else {
			cmd.Printf("%s%-20s  %s\n", wt.currentMarker, wt.name, wt.branch)
		}
	}

	return nil
}
