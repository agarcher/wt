package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/agarcher/wt/internal/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(cdCmd)
}

var cdCmd = &cobra.Command{
	Use:   "cd <name>",
	Short: "Change to a worktree directory",
	Long: `Output the path to a worktree directory.

The shell integration wrapper will use this output to change
to the worktree directory.

Note: This command requires shell integration. Add this to your
shell rc file:

  eval "$(wt init zsh)"  # or bash/fish`,
	Args: cobra.ExactArgs(1),
	RunE: runCd,
}

func runCd(cmd *cobra.Command, args []string) error {
	name := args[0]

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

	// Determine the worktree path
	worktreePath := filepath.Join(repoRoot, cfg.WorktreeDir, name)

	// Check if worktree exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		return fmt.Errorf("worktree %q does not exist", name)
	}

	// Output the path to stdout (shell wrapper will handle the actual cd)
	fmt.Fprintln(cmd.OutOrStdout(), worktreePath)
	return nil
}
