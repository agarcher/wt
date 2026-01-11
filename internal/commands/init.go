package commands

import (
	"fmt"

	"github.com/agarcher/wt/internal/shell"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init <shell>",
	Short: "Generate shell integration script",
	Long: `Generate shell integration script for the specified shell.

Supported shells: zsh, bash, fish

Add the following to your shell configuration file:

  For zsh (~/.zshrc):
    eval "$(wt init zsh)"

  For bash (~/.bashrc):
    eval "$(wt init bash)"

  For fish (~/.config/fish/config.fish):
    wt init fish | source`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		shellName := args[0]
		script, err := shell.Generate(shellName)
		if err != nil {
			return err
		}
		_, err = fmt.Fprint(cmd.OutOrStdout(), script)
		return err
	},
}
