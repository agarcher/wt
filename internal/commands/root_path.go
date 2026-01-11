package commands

import (
	"fmt"

	"github.com/agarcher/wt/internal/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(rootPathCmd)
}

var rootPathCmd = &cobra.Command{
	Use:   "root",
	Short: "Print the main repository root path",
	Long: `Print the path to the main repository root.

This is useful for scripting and automation.`,
	RunE: runRootPath,
}

func runRootPath(cmd *cobra.Command, args []string) error {
	// Find the main repository root
	repoRoot, err := config.GetMainRepoRoot()
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), repoRoot)
	return nil
}
