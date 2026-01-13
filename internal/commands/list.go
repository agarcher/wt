package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/agarcher/wt/internal/config"
	"github.com/agarcher/wt/internal/git"
	"github.com/agarcher/wt/internal/hooks"
	"github.com/spf13/cobra"
)

var verboseFlag bool

func init() {
	listCmd.Flags().BoolVarP(&verboseFlag, "verbose", "v", false, "Show detailed status for each worktree")
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
- Commits ahead/behind main branch (↑↓)
- Status: [new], [in_progress], [merged], [dirty]
  - new: no commits yet (still on initial commit)
  - in_progress: has unmerged commits (bold)
  - merged: branch has been merged to main
  - dirty: has uncommitted changes (bold, additive)

Use -v/--verbose for detailed multi-line output including worktree age.`,
	RunE: runList,
}

// worktreeInfo holds display information for a worktree
type worktreeInfo struct {
	name          string
	branch        string
	path          string
	currentMarker string
	status        *git.WorktreeStatus
	index         int
}

func runList(cmd *cobra.Command, args []string) error {
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

	// Get merged branches cache for efficiency
	mergedCache, _ := git.GetMergedBranches(setup.RepoRoot, setup.ComparisonRef)

	// Get current directory to highlight current worktree
	cwd, _ := os.Getwd()
	worktreesDir := filepath.Join(setup.RepoRoot, setup.Config.WorktreeDir)

	// Collect managed worktrees (excluding main repo)
	var managedWorktrees []worktreeInfo

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

		// Get full worktree status
		status, _ := git.GetWorktreeStatus(setup.RepoRoot, wt.Path, name, wt.Branch, setup.ComparisonRef, mergedCache)

		// Get worktree index
		idx, _ := git.GetWorktreeIndex(setup.RepoRoot, name)

		// Check if this is the current worktree (exact match or inside it)
		currentMarker := "  "
		if cwd == wt.Path || strings.HasPrefix(cwd, wt.Path+string(filepath.Separator)) {
			currentMarker = "* "
		}

		managedWorktrees = append(managedWorktrees, worktreeInfo{
			name:          name,
			branch:        wt.Branch,
			path:          wt.Path,
			currentMarker: currentMarker,
			status:        status,
			index:         idx,
		})
	}

	// If no worktrees, display message and return
	if len(managedWorktrees) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No worktrees")
		return nil
	}

	// Print based on verbose flag
	if verboseFlag {
		printVerboseWorktrees(cmd, managedWorktrees, setup.Config, setup.RepoRoot)
	} else {
		printCompactWorktrees(cmd, managedWorktrees)
	}

	return nil
}


// printCompactWorktrees prints worktrees in compact table format
func printCompactWorktrees(cmd *cobra.Command, worktrees []worktreeInfo) {
	out := cmd.OutOrStdout()

	// Calculate column widths based on content
	nameWidth := len("NAME")
	branchWidth := len("BRANCH")
	for _, wt := range worktrees {
		if len(wt.name) > nameWidth {
			nameWidth = len(wt.name)
		}
		if len(wt.branch) > branchWidth {
			branchWidth = len(wt.branch)
		}
	}

	// Print header and rows with dynamic widths
	_, _ = fmt.Fprintf(out, "  %-*s  %5s  %-*s  %s\n", nameWidth, "NAME", "INDEX", branchWidth, "BRANCH", "STATUS")
	for _, wt := range worktrees {
		statusStr := FormatCompactStatus(wt.status)
		indexStr := "-"
		if wt.index > 0 {
			indexStr = fmt.Sprintf("%d", wt.index)
		}
		_, _ = fmt.Fprintf(out, "%s%-*s  %5s  %-*s  %s\n", wt.currentMarker, nameWidth, wt.name, indexStr, branchWidth, wt.branch, statusStr)
	}
}

// printVerboseWorktrees prints worktrees in detailed multi-line format
func printVerboseWorktrees(cmd *cobra.Command, worktrees []worktreeInfo, cfg *config.Config, repoRoot string) {
	out := cmd.OutOrStdout()
	separator := strings.Repeat("=", 80)

	for _, wt := range worktrees {
		// Run info hooks to get custom output
		hookOutput := ""
		if len(cfg.Hooks.Info) > 0 {
			env := &hooks.Env{
				Name:        wt.name,
				Path:        wt.path,
				Branch:      wt.branch,
				RepoRoot:    repoRoot,
				WorktreeDir: cfg.WorktreeDir,
				Index:       wt.index,
			}
			hookOutput, _ = hooks.RunInfo(cfg, env)
		}

		_, _ = fmt.Fprintln(out, separator)
		PrintVerboseWorktree(out, VerboseInfo{
			Name:          wt.name,
			Branch:        wt.branch,
			Index:         wt.index,
			CreatedAt:     wt.status.CreatedAt,
			Status:        wt.status,
			CurrentMarker: wt.currentMarker,
			HookOutput:    hookOutput,
		})
	}
	_, _ = fmt.Fprintln(out, separator)
}
