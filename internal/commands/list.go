package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/agarcher/wt/internal/config"
	"github.com/agarcher/wt/internal/git"
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

	// Get main branch for comparisons
	mainBranch, err := git.GetDefaultBranch(repoRoot)
	if err != nil {
		mainBranch = "main" // Fallback
	}

	// Get merged branches cache for efficiency
	mergedCache, _ := git.GetMergedBranches(repoRoot, mainBranch)

	// Get current directory to highlight current worktree
	cwd, _ := os.Getwd()
	worktreesDir := filepath.Join(repoRoot, cfg.WorktreeDir)

	// Collect managed worktrees (excluding main repo)
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

		// Get full worktree status
		status, _ := git.GetWorktreeStatus(repoRoot, wt.Path, name, wt.Branch, mainBranch, mergedCache)

		// Check if this is the current worktree
		currentMarker := "  "
		if strings.HasPrefix(cwd, wt.Path) {
			currentMarker = "* "
		}

		managedWorktrees = append(managedWorktrees, worktreeInfo{
			name:          name,
			branch:        wt.Branch,
			path:          wt.Path,
			currentMarker: currentMarker,
			status:        status,
		})
	}

	// If no worktrees, display message and return
	if len(managedWorktrees) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No worktrees")
		return nil
	}

	// Print based on verbose flag
	if verboseFlag {
		printVerboseWorktrees(cmd, managedWorktrees)
	} else {
		printCompactWorktrees(cmd, managedWorktrees)
	}

	return nil
}

// ANSI codes for bold text
const (
	bold  = "\033[1m"
	reset = "\033[0m"
)

// formatCompactStatus builds the compact status string with arrows
// State indicators (mutually exclusive): new, in_progress, merged
// dirty is additive and can appear alongside any state
func formatCompactStatus(status *git.WorktreeStatus) string {
	var parts []string

	if status.CommitsAhead > 0 {
		parts = append(parts, fmt.Sprintf("↑%d", status.CommitsAhead))
	}
	if status.CommitsBehind > 0 {
		parts = append(parts, fmt.Sprintf("↓%d", status.CommitsBehind))
	}

	// Build status tags (state is mutually exclusive, dirty is additive)
	var statusTags []string

	// State indicator: new > in_progress > merged (mutually exclusive)
	if status.IsNew {
		statusTags = append(statusTags, "new")
	} else if status.CommitsAhead > 0 && !status.IsMerged {
		// in_progress: has commits ahead that aren't merged
		statusTags = append(statusTags, bold+"in_progress"+reset)
	} else if status.IsMerged && status.CommitsAhead == 0 {
		statusTags = append(statusTags, "merged")
	}

	// dirty is additive - can appear with any state
	if status.HasUncommittedChanges {
		statusTags = append(statusTags, bold+"dirty"+reset)
	}

	if len(statusTags) > 0 {
		parts = append(parts, "["+strings.Join(statusTags, ", ")+"]")
	}

	return strings.Join(parts, " ")
}

// printCompactWorktrees prints worktrees in compact table format
func printCompactWorktrees(cmd *cobra.Command, worktrees []worktreeInfo) {
	out := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(out, "  %-20s  %-30s %s\n", "NAME", "BRANCH", "STATUS")
	for _, wt := range worktrees {
		statusStr := formatCompactStatus(wt.status)
		_, _ = fmt.Fprintf(out, "%s%-20s  %-30s %s\n", wt.currentMarker, wt.name, wt.branch, statusStr)
	}
}

// printVerboseWorktrees prints worktrees in detailed multi-line format
func printVerboseWorktrees(cmd *cobra.Command, worktrees []worktreeInfo) {
	out := cmd.OutOrStdout()
	separator := strings.Repeat("=", 80)

	for _, wt := range worktrees {
		_, _ = fmt.Fprintln(out, separator)
		_, _ = fmt.Fprintf(out, "%s%s\n", wt.currentMarker, wt.name)
		_, _ = fmt.Fprintf(out, "  Branch: %s\n", wt.branch)

		// Age
		if !wt.status.CreatedAt.IsZero() {
			age := formatAge(time.Since(wt.status.CreatedAt))
			_, _ = fmt.Fprintf(out, "  Age: %s\n", age)
		}

		// Ahead/Behind
		if wt.status.CommitsAhead > 0 || wt.status.CommitsBehind > 0 {
			aheadStr := "commit"
			if wt.status.CommitsAhead != 1 {
				aheadStr = "commits"
			}
			behindStr := "commit"
			if wt.status.CommitsBehind != 1 {
				behindStr = "commits"
			}
			_, _ = fmt.Fprintf(out, "  Ahead: %d %s  Behind: %d %s\n",
				wt.status.CommitsAhead, aheadStr,
				wt.status.CommitsBehind, behindStr)
		}

		// Status: state is mutually exclusive (new, in_progress, merged), dirty is additive
		var statusLabels []string

		// State indicator: new > in_progress > merged (mutually exclusive)
		if wt.status.IsNew {
			statusLabels = append(statusLabels, "new")
		} else if wt.status.CommitsAhead > 0 && !wt.status.IsMerged {
			statusLabels = append(statusLabels, bold+"in_progress"+reset)
		} else if wt.status.IsMerged && wt.status.CommitsAhead == 0 {
			statusLabels = append(statusLabels, "merged")
		}

		// dirty is additive
		if wt.status.HasUncommittedChanges {
			statusLabels = append(statusLabels, bold+"dirty"+reset)
		}

		if len(statusLabels) > 0 {
			_, _ = fmt.Fprintf(out, "  Status: %s\n", strings.Join(statusLabels, ", "))
		}
	}
	_, _ = fmt.Fprintln(out, separator)
}

// formatAge formats a duration as a human-readable age string
func formatAge(d time.Duration) string {
	days := int(d.Hours() / 24)

	if days == 0 {
		hours := int(d.Hours())
		if hours == 0 {
			return "less than an hour"
		}
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	}

	if days == 1 {
		return "1 day"
	}

	weeks := days / 7
	if weeks >= 1 {
		if weeks == 1 {
			return "1 week"
		}
		return fmt.Sprintf("%d weeks", weeks)
	}

	return fmt.Sprintf("%d days", days)
}
