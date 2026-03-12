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

func init() {
	rootCmd.AddCommand(infoCmd)
}

var infoCmd = &cobra.Command{
	Use:   "info [name]",
	Short: "Show detailed information about a worktree",
	Long: `Display comprehensive information about a worktree.

If no name is provided and you're currently inside a worktree,
information about that worktree will be displayed.

Shows:
- Name and branch
- Index number (if assigned)
- Creation date and age
- Status (commits ahead/behind, dirty state, merge status)
- Custom info from info hooks (if configured)`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeWorktreeNames,
	RunE:              runInfo,
}

func runInfo(cmd *cobra.Command, args []string) error {
	// Setup comparison context (prints repo root, fetches if configured, prints comparison ref)
	setup, err := SetupCompare(cmd)
	if err != nil {
		return err
	}

	// Determine which worktree to show info for
	var name string
	var worktreePath string

	if len(args) > 0 {
		name = args[0]
		worktreePath = filepath.Join(setup.RepoRoot, setup.Config.WorktreeDir, name)
	} else {
		// Auto-detect from current directory
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		worktreesDir := filepath.Clean(filepath.Join(setup.RepoRoot, setup.Config.WorktreeDir))
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

	// Get worktree details
	branch, _ := git.GetCurrentBranch(worktreePath)

	// Get merged branches cache for efficiency
	mergedCache, _ := git.GetMergedBranches(setup.RepoRoot, setup.ComparisonRef)

	// Get full worktree status
	status, _ := git.GetWorktreeStatus(setup.RepoRoot, worktreePath, name, branch, setup.ComparisonRef, mergedCache)

	// Get worktree index
	idx, _ := git.GetWorktreeIndex(setup.RepoRoot, name)

	// Determine current marker
	cwd, _ := os.Getwd()
	currentMarker := "  "
	if cwd == worktreePath || strings.HasPrefix(cwd, worktreePath+string(filepath.Separator)) {
		currentMarker = "* "
	}

	// Run info hooks to get custom output
	hookOutput := ""
	if len(setup.Config.Hooks.Info) > 0 {
		env := &hooks.Env{
			Name:        name,
			Path:        worktreePath,
			Branch:      branch,
			RepoRoot:    setup.RepoRoot,
			WorktreeDir: setup.Config.WorktreeDir,
			Index:       idx,
		}
		hookOutput, _ = hooks.RunInfo(setup.Config, env)
	}

	// Print output
	out := cmd.OutOrStdout()
	separator := strings.Repeat("=", 80)

	_, _ = fmt.Fprintln(out, separator)
	PrintVerboseWorktree(out, VerboseInfo{
		Name:          name,
		Branch:        branch,
		Index:         idx,
		CreatedAt:     status.CreatedAt,
		Status:        status,
		CurrentMarker: currentMarker,
		HookOutput:    hookOutput,
	})
	_, _ = fmt.Fprintln(out, separator)

	return nil
}
