package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// Version is set at build time
	Version = "dev"
)

var rootCmd = &cobra.Command{
	Use:   "wt",
	Short: "Git worktree manager with lifecycle hooks",
	Long: `wt is a CLI tool for managing git worktrees with lifecycle hooks.

It provides a simple interface for creating, managing, and cleaning up
git worktrees, with support for custom hooks that run during worktree
lifecycle events.

To enable shell integration (required for 'cd' functionality), add this
to your shell rc file:

  For zsh:  eval "$(wt init zsh)"
  For bash: eval "$(wt init bash)"
  For fish: wt init fish | source`,
	SilenceUsage: true,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "wt version", Version)
	},
}
