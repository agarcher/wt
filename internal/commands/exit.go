package commands

import (
	"fmt"

	"github.com/agarcher/wt/internal/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(exitCmd)
}

var exitCmd = &cobra.Command{
	Use:   "exit",
	Short: "Return to the main repository",
	Long: `Output the path to the main repository root.

The shell integration wrapper will use this output to change
to the main repository directory.

Note: This command requires shell integration. Add this to your
shell rc file:

  eval "$(wt init zsh)"  # or bash/fish`,
	RunE: runExit,
}

func runExit(cmd *cobra.Command, args []string) error {
	// Find the main repository root
	repoRoot, err := config.GetMainRepoRoot()
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	// Check that config exists (to confirm this is a wt-enabled repo)
	if !config.Exists(repoRoot) {
		return fmt.Errorf("not in a wt-enabled repository (no .wt.yaml found)")
	}

	// Output the path to stdout (shell wrapper will handle the actual cd)
	fmt.Fprintln(cmd.OutOrStdout(), repoRoot)
	return nil
}
