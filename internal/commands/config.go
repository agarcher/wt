package commands

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/agarcher/wt/internal/config"
	"github.com/agarcher/wt/internal/userconfig"
	"github.com/spf13/cobra"
)

var (
	configGlobal     bool
	configUnset      bool
	configList       bool
	configShowOrigin bool
)

func init() {
	configCmd.Flags().BoolVar(&configGlobal, "global", false, "Set/get global configuration")
	configCmd.Flags().BoolVar(&configUnset, "unset", false, "Remove a per-repo configuration value")
	configCmd.Flags().BoolVar(&configList, "list", false, "List all configuration values")
	configCmd.Flags().BoolVar(&configShowOrigin, "show-origin", false, "Show where each configuration value comes from")
	rootCmd.AddCommand(configCmd)
}

var configCmd = &cobra.Command{
	Use:   "config [key] [value]",
	Short: "Manage user configuration",
	Long: `Get and set wt user configuration options.

User settings are stored in ~/.config/wt/config.yaml

Configuration keys:
  remote          Remote to compare against (empty = local comparison)
  fetch_interval  Fetch interval: "5m", "1h", "0" (always), or "never" (disable)
  worktree_dir    Directory for worktrees (per-repo only)
  branch_pattern  Branch naming pattern, e.g. "{name}" (per-repo only)
  default_branch  Default branch to compare against (per-repo only)

Examples:
  wt config --list                       # List all settings
  wt config --show-origin                # Show where each value comes from
  wt config --global remote origin       # Set global remote
  wt config --global fetch_interval 10m  # Fetch at most every 10 minutes
  wt config --global fetch_interval 0    # Always fetch (no caching)
  wt config fetch_interval never         # Disable fetch for current repo
  wt config --unset fetch_interval       # Remove per-repo override

Note: fetch_interval only has an effect when 'remote' is set.
If remote is empty, comparisons are done against the local branch.`,
	RunE: runConfig,
}

func runConfig(cmd *cobra.Command, args []string) error {
	// Load user config
	cfg, err := userconfig.Load()
	if err != nil {
		return fmt.Errorf("failed to load user config: %w", err)
	}

	// Handle --list
	if configList {
		return printConfigList(cmd, cfg)
	}

	// Handle --show-origin
	if configShowOrigin {
		return printConfigShowOrigin(cmd, cfg)
	}

	// Handle --unset
	if configUnset {
		if len(args) < 1 {
			return fmt.Errorf("usage: wt config --unset <key>")
		}
		return unsetConfig(cmd, cfg, args[0])
	}

	// Get or set
	switch len(args) {
	case 0:
		return fmt.Errorf("usage: wt config [--global] <key> [value]\n       wt config --list\n       wt config --show-origin")
	case 1:
		// Get value
		return getConfig(cmd, cfg, args[0])
	case 2:
		// Set value
		return setConfig(cmd, cfg, args[0], args[1])
	default:
		return fmt.Errorf("too many arguments")
	}
}

func printConfigList(cmd *cobra.Command, cfg *userconfig.UserConfig) error {
	out := cmd.OutOrStdout()

	// Print global values
	if cfg.Remote != "" {
		_, _ = fmt.Fprintf(out, "remote = %s (global)\n", cfg.Remote)
	}
	if cfg.FetchInterval != "" {
		_, _ = fmt.Fprintf(out, "fetch_interval = %s (global)\n", cfg.FetchInterval)
	}

	// Print per-repo values
	for repoPath, repoConfig := range cfg.Repos {
		if repoConfig.Remote != nil {
			remoteVal := *repoConfig.Remote
			if remoteVal == "" {
				remoteVal = "\"\"" // Display empty explicitly
			}
			_, _ = fmt.Fprintf(out, "repos.%s.remote = %s\n", repoPath, remoteVal)
		}
		if repoConfig.FetchInterval != nil {
			_, _ = fmt.Fprintf(out, "repos.%s.fetch_interval = %s\n", repoPath, *repoConfig.FetchInterval)
		}
		if repoConfig.WorktreeDir != nil {
			_, _ = fmt.Fprintf(out, "repos.%s.worktree_dir = %s\n", repoPath, *repoConfig.WorktreeDir)
		}
		if repoConfig.BranchPattern != nil {
			_, _ = fmt.Fprintf(out, "repos.%s.branch_pattern = %s\n", repoPath, *repoConfig.BranchPattern)
		}
		if repoConfig.DefaultBranch != nil {
			_, _ = fmt.Fprintf(out, "repos.%s.default_branch = %s\n", repoPath, *repoConfig.DefaultBranch)
		}
	}

	return nil
}

func printConfigShowOrigin(cmd *cobra.Command, cfg *userconfig.UserConfig) error {
	out := cmd.OutOrStdout()

	// Get current repo path for context
	repoRoot, _ := config.GetMainRepoRoot()

	configPath, err := userconfig.GetConfigPath()
	if err != nil {
		configPath = "(unknown)"
	}

	// Show effective values for current repo
	if repoRoot != "" {
		remote := cfg.GetRemoteForRepo(repoRoot)
		fetchInterval := cfg.GetFetchIntervalForRepo(repoRoot)

		// Determine source of remote
		if repoConfig, ok := cfg.Repos[repoRoot]; ok && repoConfig.Remote != nil {
			remoteDisplay := remote
			if remoteDisplay == "" {
				remoteDisplay = "\"\""
			}
			_, _ = fmt.Fprintf(out, "remote = %-20s %s (repos.%s)\n", remoteDisplay, configPath, repoRoot)
		} else if cfg.Remote != "" {
			_, _ = fmt.Fprintf(out, "remote = %-20s %s (global)\n", remote, configPath)
		} else {
			_, _ = fmt.Fprintf(out, "remote = %-20s (default: local comparison)\n", "\"\"")
		}

		// Determine source of fetch_interval
		if repoConfig, ok := cfg.Repos[repoRoot]; ok && repoConfig.FetchInterval != nil {
			_, _ = fmt.Fprintf(out, "fetch_interval = %-14s %s (repos.%s)\n", *repoConfig.FetchInterval, configPath, repoRoot)
		} else if cfg.FetchInterval != "" {
			_, _ = fmt.Fprintf(out, "fetch_interval = %-14s %s (global)\n", cfg.FetchInterval, configPath)
		} else {
			_, _ = fmt.Fprintf(out, "fetch_interval = %-14s (default)\n", fetchInterval)
		}

		// Show worktree settings with their source
		resolved, resolveErr := config.Resolve(repoRoot)
		if resolveErr == nil {
			repoConfig := cfg.Repos[repoRoot]
			fromRepo := resolved.Source >= config.SourceRepoFile

			printOriginField(out, "worktree_dir", 16, repoConfig.WorktreeDir, resolved.WorktreeDir, fromRepo, configPath, repoRoot)
			printOriginField(out, "branch_pattern", 14, repoConfig.BranchPattern, resolved.BranchPattern, fromRepo, configPath, repoRoot)

			// default_branch has special handling for empty value
			defaultDisplay := resolved.DefaultBranch
			if defaultDisplay == "" {
				defaultDisplay = "(auto-detected)"
			}
			printOriginField(out, "default_branch", 14, repoConfig.DefaultBranch, defaultDisplay, fromRepo && resolved.DefaultBranch != "", configPath, repoRoot)
		}
	} else {
		// Not in a repo, just show global values
		_, _ = fmt.Fprintf(out, "remote = %-20s %s (global)\n", cfg.Remote, configPath)
		fetchInterval := cfg.FetchInterval
		if fetchInterval == "" {
			fetchInterval = userconfig.DefaultFetchInterval
		}
		_, _ = fmt.Fprintf(out, "fetch_interval = %-14s %s (global)\n", fetchInterval, configPath)
	}

	return nil
}

func getConfig(cmd *cobra.Command, cfg *userconfig.UserConfig, key string) error {
	// Validate key
	if !isValidKey(key) {
		return fmt.Errorf("unknown config key: %s\nValid keys: %s", key, strings.Join(userconfig.ValidKeys(), ", "))
	}

	if configGlobal {
		// Get global value
		value, err := cfg.GetGlobal(key)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), value)
	} else {
		// Get effective value for current repo
		repoRoot, err := config.GetMainRepoRoot()
		if err != nil {
			return fmt.Errorf("not in a git repository (use --global for global config)")
		}

		switch key {
		case "remote":
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), cfg.GetRemoteForRepo(repoRoot))
		case "fetch_interval":
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), cfg.GetFetchIntervalForRepo(repoRoot))
		case "worktree_dir", "branch_pattern", "default_branch":
			if v, ok := cfg.GetForRepo(repoRoot, key); ok {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), v)
			} else {
				// Fall back to resolved config
				resolved, err := config.Resolve(repoRoot)
				if err != nil {
					return err
				}
				switch key {
				case "worktree_dir":
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), resolved.WorktreeDir)
				case "branch_pattern":
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), resolved.BranchPattern)
				case "default_branch":
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), resolved.DefaultBranch)
				}
			}
		}
	}

	return nil
}

func setConfig(cmd *cobra.Command, cfg *userconfig.UserConfig, key, value string) error {
	// Validate key
	if !isValidKey(key) {
		return fmt.Errorf("unknown config key: %s\nValid keys: %s", key, strings.Join(userconfig.ValidKeys(), ", "))
	}

	// Validate fetch_interval value (must be a valid duration or "never")
	if key == "fetch_interval" {
		if value != "never" {
			if _, err := time.ParseDuration(value); err != nil {
				return fmt.Errorf("fetch_interval must be a valid duration (e.g., '5m', '1h', '0') or 'never'")
			}
		}
	}

	if configGlobal {
		// Set global value
		if err := cfg.SetGlobal(key, value); err != nil {
			return err
		}
	} else {
		// Set per-repo value
		repoRoot, err := config.GetMainRepoRoot()
		if err != nil {
			return fmt.Errorf("not in a git repository (use --global for global config)")
		}

		if err := cfg.SetForRepo(repoRoot, key, value); err != nil {
			return err
		}
	}

	// Save config
	if err := userconfig.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

func unsetConfig(cmd *cobra.Command, cfg *userconfig.UserConfig, key string) error {
	// Validate key
	if !isValidKey(key) {
		return fmt.Errorf("unknown config key: %s\nValid keys: %s", key, strings.Join(userconfig.ValidKeys(), ", "))
	}

	if configGlobal {
		// Unset global value
		if err := cfg.UnsetGlobal(key); err != nil {
			return err
		}
	} else {
		// Get current repo
		repoRoot, err := config.GetMainRepoRoot()
		if err != nil {
			return fmt.Errorf("not in a git repository (use --global to unset global config)")
		}

		// Unset per-repo value
		if err := cfg.UnsetForRepo(repoRoot, key); err != nil {
			return err
		}
	}

	// Save config
	if err := userconfig.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// printOriginField prints a config key with its value and source origin.
// personalPtr is the user's per-repo override (nil if unset).
// resolvedValue is the effective value from config resolution.
// fromRepo indicates whether .wt.yaml contributed the value.
func printOriginField(out io.Writer, key string, width int, personalPtr *string, resolvedValue string, fromRepo bool, configPath, repoRoot string) {
	format := fmt.Sprintf("%%s = %%-%ds", width)
	if personalPtr != nil {
		_, _ = fmt.Fprintf(out, format+" %s (repos.%s)\n", key, *personalPtr, configPath, repoRoot)
	} else if fromRepo {
		_, _ = fmt.Fprintf(out, format+" .wt.yaml (repo)\n", key, resolvedValue)
	} else {
		_, _ = fmt.Fprintf(out, format+" (default)\n", key, resolvedValue)
	}
}

func isValidKey(key string) bool {
	for _, k := range userconfig.ValidKeys() {
		if k == key {
			return true
		}
	}
	return false
}
