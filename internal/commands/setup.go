package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/agarcher/wt/internal/config"
	"github.com/agarcher/wt/internal/git"
	"github.com/agarcher/wt/internal/userconfig"
	"github.com/spf13/cobra"
)

var (
	setupPersonal bool
	setupShared   bool
)

func init() {
	setupCmd.Flags().BoolVar(&setupPersonal, "personal", false, "Save to personal config (~/.config/wt/config.yaml)")
	setupCmd.Flags().BoolVar(&setupShared, "shared", false, "Save to repository (.wt.yaml)")
	setupCmd.MarkFlagsMutuallyExclusive("personal", "shared")
	rootCmd.AddCommand(setupCmd)
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive setup for worktree management",
	Long: `Interactively configure worktree management for the current repository.

Configuration can be saved in two locations:
  --personal  Save to ~/.config/wt/config.yaml (just for you)
  --shared    Save to .wt.yaml in the repo root (shared with the team)

If neither flag is provided, you'll be prompted to choose.

Personal config overrides .wt.yaml for worktree_dir, branch_pattern,
and default_branch. Hooks and index settings are only read from .wt.yaml.`,
	RunE: runSetup,
}

func runSetup(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	// Find the main repository root
	repoRoot, err := config.GetMainRepoRoot()
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	cmd.Printf("Repository: %s\n\n", repoRoot)

	// Determine personal vs shared
	personal := setupPersonal
	if !setupPersonal && !setupShared {
		choice, err := prompt(reader, "Where should the configuration be saved?\n  [1] Personal config (~/.config/wt/config.yaml) — just for you\n  [2] Repository (.wt.yaml) — shared with the team\nChoose", "1")
		if err != nil {
			return err
		}
		personal = choice != "2"
	}

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
	}

	if userCfg, err := userconfig.Load(); err == nil {
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
		} else if fetchInterval > 0 {
			existingFetchInterval = fetchInterval.String()
		}
	}

	// Prompt for each setting
	worktreeDir, err := prompt(reader, "Worktree directory", existingWorktreeDir)
	if err != nil {
		return err
	}

	branchPattern, err := prompt(reader, "Branch pattern", existingBranchPattern)
	if err != nil {
		return err
	}

	defaultBranchVal, err := prompt(reader, "Default branch", existingDefaultBranch)
	if err != nil {
		return err
	}

	remote, err := prompt(reader, "Remote (empty for local comparison)", existingRemote)
	if err != nil {
		return err
	}

	fetchInterval, err := prompt(reader, "Fetch interval", existingFetchInterval)
	if err != nil {
		return err
	}

	// Save configuration
	if personal {
		return savePersonalConfig(cmd, repoRoot, worktreeDir, branchPattern, defaultBranchVal, remote, fetchInterval)
	}
	return saveSharedConfig(cmd, repoRoot, worktreeDir, branchPattern, defaultBranchVal, remote, fetchInterval)
}

func savePersonalConfig(cmd *cobra.Command, repoRoot, worktreeDir, branchPattern, defaultBranch, remote, fetchInterval string) error {
	cfg, err := userconfig.Load()
	if err != nil {
		cfg = userconfig.DefaultUserConfig()
	}

	if err := cfg.SetForRepo(repoRoot, "worktree_dir", worktreeDir); err != nil {
		return err
	}
	if err := cfg.SetForRepo(repoRoot, "branch_pattern", branchPattern); err != nil {
		return err
	}
	if err := cfg.SetForRepo(repoRoot, "default_branch", defaultBranch); err != nil {
		return err
	}
	if err := cfg.SetForRepo(repoRoot, "remote", remote); err != nil {
		return err
	}
	if err := cfg.SetForRepo(repoRoot, "fetch_interval", fetchInterval); err != nil {
		return err
	}

	if err := userconfig.Save(cfg); err != nil {
		return fmt.Errorf("failed to save personal config: %w", err)
	}

	configPath, _ := userconfig.GetConfigPath()
	cmd.Printf("\nConfiguration saved to %s\n", configPath)
	return nil
}

func saveSharedConfig(cmd *cobra.Command, repoRoot, worktreeDir, branchPattern, defaultBranch, remote, fetchInterval string) error {
	cfg := config.DefaultConfig()

	// Load existing .wt.yaml to preserve hooks and index
	if existing, err := config.Load(repoRoot); err == nil {
		cfg = existing
	}

	cfg.WorktreeDir = worktreeDir
	cfg.BranchPattern = branchPattern
	cfg.DefaultBranch = defaultBranch

	if err := config.Save(repoRoot, cfg); err != nil {
		return fmt.Errorf("failed to save .wt.yaml: %w", err)
	}

	// Save remote and fetch_interval to user config (these are always personal)
	userCfg, err := userconfig.Load()
	if err != nil {
		userCfg = userconfig.DefaultUserConfig()
	}

	if remote != "" {
		_ = userCfg.SetForRepo(repoRoot, "remote", remote)
	}
	if fetchInterval != userconfig.DefaultFetchInterval {
		_ = userCfg.SetForRepo(repoRoot, "fetch_interval", fetchInterval)
	}

	if err := userconfig.Save(userCfg); err != nil {
		cmd.Printf("Warning: failed to save user config: %v\n", err)
	}

	cmd.Printf("\nConfiguration saved to %s\n", filepath.Join(repoRoot, config.ConfigFileName))
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
