package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/agarcher/wt/internal/config"
	"github.com/agarcher/wt/internal/git"
	"github.com/agarcher/wt/internal/userconfig"
	"github.com/spf13/cobra"
)

var (
	setupPersonal bool
	setupShared   bool
	setupGlobal   bool
)

func init() {
	setupCmd.Flags().BoolVar(&setupPersonal, "personal", false, "Save to personal config (~/.config/wt/config.yaml) for this repo")
	setupCmd.Flags().BoolVar(&setupShared, "shared", false, "Save to repository (.wt.yaml)")
	setupCmd.Flags().BoolVar(&setupGlobal, "global", false, "Configure global defaults (~/.config/wt/config.yaml)")
	setupCmd.MarkFlagsMutuallyExclusive("personal", "shared", "global")
	rootCmd.AddCommand(setupCmd)
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive setup for worktree management",
	Long: `Interactively configure worktree management for the current repository.

Configuration can be saved in three modes:
  --global    Set global defaults in ~/.config/wt/config.yaml (remote, fetch interval)
  --personal  Save per-repo overrides to ~/.config/wt/config.yaml (just for you)
  --shared    Save to .wt.yaml in the repo root (shared with the team)

If no flag is provided, you'll be prompted to choose.

Personal config overrides .wt.yaml for worktree_dir, branch_pattern,
and default_branch. Hooks and index settings are only read from .wt.yaml.`,
	RunE: runSetup,
}

// setupInput allows tests to override stdin
var setupInput *os.File

func runSetup(cmd *cobra.Command, args []string) error {
	input := os.Stdin
	if setupInput != nil {
		input = setupInput
	}
	reader := bufio.NewReader(input)

	// Determine mode
	mode := ""
	if setupGlobal {
		mode = "1"
	} else if setupPersonal {
		mode = "2"
	} else if setupShared {
		mode = "3"
	} else {
		var err error
		mode, err = prompt(reader, "What would you like to configure?\n  [1] Global defaults (~/.config/wt/config.yaml) — remote, fetch interval\n  [2] Personal repo config (~/.config/wt/config.yaml) — just for you\n  [3] Shared repo config (.wt.yaml) — shared with the team\nChoose", "1")
		if err != nil {
			return err
		}
	}

	switch mode {
	case "1":
		return runGlobalSetup(cmd, reader)
	case "2", "3":
		// continue below
	default:
		return fmt.Errorf("invalid choice %q: expected 1, 2, or 3", mode)
	}

	// Find the main repository root
	repoRoot, err := config.GetMainRepoRoot()
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	cmd.Printf("Repository: %s\n\n", repoRoot)

	personal := mode == "2"

	// Auto-detect defaults
	basename := filepath.Base(repoRoot)
	defaultWorktreeDir := filepath.Join("..", basename+"-worktrees")

	defaultBranch, _ := git.GetDefaultBranch(repoRoot)
	if defaultBranch == "" {
		defaultBranch = "main"
	}

	// Check for existing config to pre-populate
	existingWorktreeDir := defaultWorktreeDir
	existingBranchPattern := "{name}"
	existingDefaultBranch := defaultBranch
	existingRemote := ""
	existingFetchInterval := userconfig.DefaultFetchInterval

	if repoCfg, err := config.Load(repoRoot); err == nil {
		existingWorktreeDir = repoCfg.WorktreeDir
		if repoCfg.BranchPattern != "" {
			existingBranchPattern = repoCfg.BranchPattern
		}
		if repoCfg.DefaultBranch != "" {
			existingDefaultBranch = repoCfg.DefaultBranch
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to load %s: %w", config.ConfigFileName, err)
	}

	userCfg, err := userconfig.Load()
	if err != nil {
		return fmt.Errorf("failed to load user config: %w", err)
	}
	if v, ok := userCfg.GetForRepo(repoRoot, "worktree_dir"); ok {
		existingWorktreeDir = v
	}
	if v, ok := userCfg.GetForRepo(repoRoot, "branch_pattern"); ok {
		existingBranchPattern = v
	}
	if v, ok := userCfg.GetForRepo(repoRoot, "default_branch"); ok {
		existingDefaultBranch = v
	}
	existingRemote = userCfg.GetRemoteForRepo(repoRoot)
	fetchInterval := userCfg.GetFetchIntervalForRepo(repoRoot)
	if fetchInterval == userconfig.FetchIntervalNever {
		existingFetchInterval = "never"
	} else if fetchInterval >= 0 {
		existingFetchInterval = fetchInterval.String()
	}

	if personal {
		return runPersonalSetup(cmd, reader, repoRoot, existingRemote, existingFetchInterval, existingWorktreeDir, existingBranchPattern, existingDefaultBranch)
	}
	return runSharedSetup(cmd, reader, repoRoot, existingRemote, existingWorktreeDir, existingBranchPattern, existingDefaultBranch)
}

func runGlobalSetup(cmd *cobra.Command, reader *bufio.Reader) error {
	cfg, err := userconfig.Load()
	if err != nil {
		return fmt.Errorf("failed to load user config: %w", err)
	}

	existingRemote := cfg.Remote
	existingFetchInterval := cfg.FetchInterval
	if existingFetchInterval == "" {
		existingFetchInterval = userconfig.DefaultFetchInterval
	}

	remote, err := prompt(reader, "Default remote (empty for local comparison)", existingRemote)
	if err != nil {
		return err
	}

	if err := cfg.SetGlobal("remote", remote); err != nil {
		return fmt.Errorf("failed to set remote: %w", err)
	}

	if remote != "" {
		fetchInterval, err := prompt(reader, "Default fetch interval", existingFetchInterval)
		if err != nil {
			return err
		}

		if err := validateFetchInterval(fetchInterval); err != nil {
			return err
		}

		if err := cfg.SetGlobal("fetch_interval", fetchInterval); err != nil {
			return fmt.Errorf("failed to set fetch_interval: %w", err)
		}
	}

	if err := userconfig.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	configPath, _ := userconfig.GetConfigPath()
	cmd.Printf("\nConfiguration saved to %s\n", configPath)
	return nil
}

func runPersonalSetup(cmd *cobra.Command, reader *bufio.Reader, repoRoot, existingRemote, existingFetchInterval, existingWorktreeDir, existingBranchPattern, existingDefaultBranch string) error {
	cfg, err := userconfig.Load()
	if err != nil {
		return fmt.Errorf("failed to load user config: %w", err)
	}

	worktreeDir, err := prompt(reader, "Worktree directory", existingWorktreeDir)
	if err != nil {
		return err
	}
	if strings.TrimSpace(worktreeDir) == "" {
		return fmt.Errorf("worktree directory cannot be empty")
	}

	branchPattern, err := prompt(reader, "Branch pattern", existingBranchPattern)
	if err != nil {
		return err
	}

	defaultBranch, err := prompt(reader, "Default branch", existingDefaultBranch)
	if err != nil {
		return err
	}

	remote, err := prompt(reader, "Remote (empty for local comparison)", existingRemote)
	if err != nil {
		return err
	}

	repoSettings := [][2]string{
		{"worktree_dir", worktreeDir},
		{"branch_pattern", branchPattern},
		{"default_branch", defaultBranch},
		{"remote", remote},
	}

	if remote != "" {
		fetchInterval, err := prompt(reader, "Fetch interval", existingFetchInterval)
		if err != nil {
			return err
		}

		if err := validateFetchInterval(fetchInterval); err != nil {
			return err
		}

		repoSettings = append(repoSettings, [2]string{"fetch_interval", fetchInterval})
	}

	for _, kv := range repoSettings {
		if err := cfg.SetForRepo(repoRoot, kv[0], kv[1]); err != nil {
			return fmt.Errorf("failed to set %s: %w", kv[0], err)
		}
	}

	if err := userconfig.Save(cfg); err != nil {
		return fmt.Errorf("failed to save personal config: %w", err)
	}

	configPath, _ := userconfig.GetConfigPath()
	cmd.Printf("\nConfiguration saved to %s\n", configPath)
	return nil
}

func runSharedSetup(cmd *cobra.Command, reader *bufio.Reader, repoRoot, existingRemote, existingWorktreeDir, existingBranchPattern, existingDefaultBranch string) error {
	worktreeDir, err := prompt(reader, "Worktree directory", existingWorktreeDir)
	if err != nil {
		return err
	}
	if strings.TrimSpace(worktreeDir) == "" {
		return fmt.Errorf("worktree directory cannot be empty")
	}

	branchPattern, err := prompt(reader, "Branch pattern", existingBranchPattern)
	if err != nil {
		return err
	}

	defaultBranchVal, err := prompt(reader, "Default branch", existingDefaultBranch)
	if err != nil {
		return err
	}

	// Save shared config — preserve existing hooks/index from .wt.yaml
	cfg := config.DefaultConfig()
	if existing, err := config.Load(repoRoot); err == nil {
		cfg = existing
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to load %s: %w", config.ConfigFileName, err)
	}

	cfg.WorktreeDir = worktreeDir
	cfg.BranchPattern = branchPattern
	cfg.DefaultBranch = defaultBranchVal

	if err := config.Save(repoRoot, cfg); err != nil {
		return fmt.Errorf("failed to save .wt.yaml: %w", err)
	}

	cmd.Printf("\nConfiguration saved to %s\n", filepath.Join(repoRoot, config.ConfigFileName))

	// Ask about personal repo settings
	cmd.Println()
	setupPersonalRepo, err := prompt(reader, "Would you also like to configure personal settings for this repository? [y/N]", "n")
	if err != nil {
		return err
	}

	if !strings.EqualFold(setupPersonalRepo, "y") && !strings.EqualFold(setupPersonalRepo, "yes") {
		return nil
	}

	cmd.Println()
	remote, err := prompt(reader, "Remote (empty for local comparison)", existingRemote)
	if err != nil {
		return err
	}

	userCfg, err := userconfig.Load()
	if err != nil {
		return fmt.Errorf("failed to load user config: %w", err)
	}

	if err := userCfg.SetForRepo(repoRoot, "remote", remote); err != nil {
		return fmt.Errorf("failed to set remote: %w", err)
	}

	if remote != "" {
		existingFetchInterval := userconfig.DefaultFetchInterval
		fetchInterval := userCfg.GetFetchIntervalForRepo(repoRoot)
		if fetchInterval == userconfig.FetchIntervalNever {
			existingFetchInterval = "never"
		} else if fetchInterval >= 0 {
			existingFetchInterval = fetchInterval.String()
		}

		fetchIntervalVal, err := prompt(reader, "Fetch interval", existingFetchInterval)
		if err != nil {
			return err
		}

		if err := validateFetchInterval(fetchIntervalVal); err != nil {
			return err
		}

		if err := userCfg.SetForRepo(repoRoot, "fetch_interval", fetchIntervalVal); err != nil {
			return fmt.Errorf("failed to set fetch_interval: %w", err)
		}
	}

	if err := userconfig.Save(userCfg); err != nil {
		return fmt.Errorf("failed to save user config: %w", err)
	}

	configPath, _ := userconfig.GetConfigPath()
	cmd.Printf("Personal settings saved to %s\n", configPath)
	return nil
}

func validateFetchInterval(fetchInterval string) error {
	if fetchInterval != "never" {
		if _, err := time.ParseDuration(fetchInterval); err != nil {
			return fmt.Errorf("invalid fetch interval %q: must be a valid duration (e.g., '5m', '1h', '0') or 'never'", fetchInterval)
		}
	}
	return nil
}

func prompt(reader *bufio.Reader, label, defaultVal string) (string, error) {
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", label, defaultVal)
	} else {
		fmt.Printf("%s: ", label)
	}

	response, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	response = strings.TrimSpace(response)
	if response == "" {
		return defaultVal, nil
	}
	return response, nil
}
