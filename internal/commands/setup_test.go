package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agarcher/wt/internal/config"
	"github.com/agarcher/wt/internal/userconfig"
)

// executeCommandWithInput runs a cobra command with simulated stdin
func executeCommandWithInput(input string, args ...string) (string, string, error) {
	// Create temp file with input
	tmpFile, err := os.CreateTemp("", "wt-test-input-*")
	if err != nil {
		return "", "", err
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if _, err := tmpFile.WriteString(input); err != nil {
		_ = tmpFile.Close()
		return "", "", err
	}
	if _, err := tmpFile.Seek(0, 0); err != nil {
		_ = tmpFile.Close()
		return "", "", err
	}

	setupInput = tmpFile
	defer func() {
		setupInput = nil
		_ = tmpFile.Close()
	}()

	return executeCommand(args...)
}

func TestSetupHelp(t *testing.T) {
	stdout, _, err := executeCommand("setup", "--help")
	if err != nil {
		t.Fatalf("setup --help failed: %v", err)
	}

	for _, s := range []string{"--global", "--personal", "--shared"} {
		if !strings.Contains(stdout, s) {
			t.Errorf("expected help to contain %q, got: %s", s, stdout)
		}
	}
}

func TestSetupFlagsMutuallyExclusive(t *testing.T) {
	repoRoot, _, cleanup := setupTestRepoWithIsolatedHome(t)
	defer cleanup()

	oldWd, _ := os.Getwd()
	_ = os.Chdir(repoRoot)
	defer func() { _ = os.Chdir(oldWd) }()

	_, _, err := executeCommandWithInput("\n", "setup", "--personal", "--shared")
	if err == nil {
		t.Error("expected error for mutually exclusive flags")
	}
}

func TestSetupGlobalWithRemote(t *testing.T) {
	_, homeDir, cleanup := setupTestRepoWithIsolatedHome(t)
	defer cleanup()

	// Provide: remote=origin, fetch_interval=10m
	stdout, _, err := executeCommandWithInput("origin\n10m\n", "setup", "--global")
	if err != nil {
		t.Fatalf("setup --global failed: %v", err)
	}

	if !strings.Contains(stdout, "Configuration saved") {
		t.Errorf("expected save confirmation, got: %s", stdout)
	}

	// Verify the config was saved
	cfg, err := userconfig.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Remote != "origin" {
		t.Errorf("expected remote 'origin', got %q", cfg.Remote)
	}
	if cfg.FetchInterval != "10m" {
		t.Errorf("expected fetch_interval '10m', got %q", cfg.FetchInterval)
	}

	// Verify file exists
	configPath := filepath.Join(homeDir, ".config", "wt", "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("config file not created at %s", configPath)
	}
}

func TestSetupGlobalLocalOnly(t *testing.T) {
	_, _, cleanup := setupTestRepoWithIsolatedHome(t)
	defer cleanup()

	// Provide: remote="" (accept default empty) - fetch_interval should NOT be prompted
	stdout, _, err := executeCommandWithInput("\n", "setup", "--global")
	if err != nil {
		t.Fatalf("setup --global failed: %v", err)
	}

	if !strings.Contains(stdout, "Configuration saved") {
		t.Errorf("expected save confirmation, got: %s", stdout)
	}

	// Verify remote is empty and fetch_interval is unchanged
	cfg, err := userconfig.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Remote != "" {
		t.Errorf("expected empty remote, got %q", cfg.Remote)
	}
}

func TestSetupGlobalInvalidFetchInterval(t *testing.T) {
	_, _, cleanup := setupTestRepoWithIsolatedHome(t)
	defer cleanup()

	// Provide: remote=origin, fetch_interval=invalid
	_, _, err := executeCommandWithInput("origin\ninvalid\n", "setup", "--global")
	if err == nil {
		t.Error("expected error for invalid fetch interval")
	}
}

func TestSetupPersonalWithRemote(t *testing.T) {
	repoRoot, _, cleanup := setupTestRepoWithIsolatedHome(t)
	defer cleanup()

	oldWd, _ := os.Getwd()
	_ = os.Chdir(repoRoot)
	defer func() { _ = os.Chdir(oldWd) }()

	// Provide: worktree_dir, branch_pattern, default_branch, remote, fetch_interval
	stdout, _, err := executeCommandWithInput("my-worktrees\n{name}\nmain\norigin\n10m\n", "setup", "--personal")
	if err != nil {
		t.Fatalf("setup --personal failed: %v", err)
	}

	if !strings.Contains(stdout, "Configuration saved") {
		t.Errorf("expected save confirmation, got: %s", stdout)
	}

	// Verify config
	cfg, err := userconfig.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if v, ok := cfg.GetForRepo(repoRoot, "worktree_dir"); !ok || v != "my-worktrees" {
		t.Errorf("expected worktree_dir 'my-worktrees', got %q (ok=%v)", v, ok)
	}
	if v, ok := cfg.GetForRepo(repoRoot, "remote"); !ok || v != "origin" {
		t.Errorf("expected remote 'origin', got %q (ok=%v)", v, ok)
	}
	if v, ok := cfg.GetForRepo(repoRoot, "fetch_interval"); !ok || v != "10m" {
		t.Errorf("expected fetch_interval '10m', got %q (ok=%v)", v, ok)
	}
}

func TestSetupPersonalLocalOnly(t *testing.T) {
	repoRoot, _, cleanup := setupTestRepoWithIsolatedHome(t)
	defer cleanup()

	oldWd, _ := os.Getwd()
	_ = os.Chdir(repoRoot)
	defer func() { _ = os.Chdir(oldWd) }()

	// Provide: worktree_dir, branch_pattern, default_branch, remote="" (empty)
	// fetch_interval should NOT be prompted
	stdout, _, err := executeCommandWithInput("my-worktrees\n{name}\nmain\n\n", "setup", "--personal")
	if err != nil {
		t.Fatalf("setup --personal failed: %v", err)
	}

	if !strings.Contains(stdout, "Configuration saved") {
		t.Errorf("expected save confirmation, got: %s", stdout)
	}

	// Verify remote is empty and no fetch_interval set
	cfg, err := userconfig.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if v, ok := cfg.GetForRepo(repoRoot, "remote"); !ok || v != "" {
		t.Errorf("expected empty remote, got %q (ok=%v)", v, ok)
	}
	if _, ok := cfg.GetForRepo(repoRoot, "fetch_interval"); ok {
		t.Error("expected no fetch_interval to be set for local-only mode")
	}
}

func TestSetupSharedOnly(t *testing.T) {
	repoRoot, _, cleanup := setupTestRepoWithIsolatedHome(t)
	defer cleanup()

	oldWd, _ := os.Getwd()
	_ = os.Chdir(repoRoot)
	defer func() { _ = os.Chdir(oldWd) }()

	// Provide: worktree_dir, branch_pattern, default_branch, then decline personal settings
	stdout, _, err := executeCommandWithInput("shared-trees\nprefix/{name}\ndevelop\nn\n", "setup", "--shared")
	if err != nil {
		t.Fatalf("setup --shared failed: %v", err)
	}

	if !strings.Contains(stdout, "Configuration saved") {
		t.Errorf("expected save confirmation, got: %s", stdout)
	}

	// Verify .wt.yaml was saved
	repoCfg, err := config.Load(repoRoot)
	if err != nil {
		t.Fatalf("failed to load .wt.yaml: %v", err)
	}

	if repoCfg.WorktreeDir != "shared-trees" {
		t.Errorf("expected worktree_dir 'shared-trees', got %q", repoCfg.WorktreeDir)
	}
	if repoCfg.BranchPattern != "prefix/{name}" {
		t.Errorf("expected branch_pattern 'prefix/{name}', got %q", repoCfg.BranchPattern)
	}
	if repoCfg.DefaultBranch != "develop" {
		t.Errorf("expected default_branch 'develop', got %q", repoCfg.DefaultBranch)
	}

	// Verify NO user config was saved (user declined personal settings)
	userCfg, err := userconfig.Load()
	if err != nil {
		t.Fatalf("failed to load user config: %v", err)
	}

	if _, ok := userCfg.GetForRepo(repoRoot, "remote"); ok {
		t.Error("expected no per-repo remote since personal setup was declined")
	}
}

func TestSetupSharedWithPersonalSettings(t *testing.T) {
	repoRoot, _, cleanup := setupTestRepoWithIsolatedHome(t)
	defer cleanup()

	oldWd, _ := os.Getwd()
	_ = os.Chdir(repoRoot)
	defer func() { _ = os.Chdir(oldWd) }()

	// Provide: worktree_dir, branch_pattern, default_branch, then accept personal, remote, fetch_interval
	stdout, _, err := executeCommandWithInput("shared-trees\n{name}\nmain\ny\norigin\n15m\n", "setup", "--shared")
	if err != nil {
		t.Fatalf("setup --shared with personal failed: %v", err)
	}

	if !strings.Contains(stdout, "Configuration saved") {
		t.Errorf("expected save confirmation, got: %s", stdout)
	}
	if !strings.Contains(stdout, "Personal settings saved") {
		t.Errorf("expected personal save confirmation, got: %s", stdout)
	}

	// Verify .wt.yaml
	repoCfg, err := config.Load(repoRoot)
	if err != nil {
		t.Fatalf("failed to load .wt.yaml: %v", err)
	}
	if repoCfg.WorktreeDir != "shared-trees" {
		t.Errorf("expected worktree_dir 'shared-trees', got %q", repoCfg.WorktreeDir)
	}

	// Verify user config has remote and fetch_interval
	userCfg, err := userconfig.Load()
	if err != nil {
		t.Fatalf("failed to load user config: %v", err)
	}

	if v, ok := userCfg.GetForRepo(repoRoot, "remote"); !ok || v != "origin" {
		t.Errorf("expected per-repo remote 'origin', got %q (ok=%v)", v, ok)
	}
	if v, ok := userCfg.GetForRepo(repoRoot, "fetch_interval"); !ok || v != "15m" {
		t.Errorf("expected per-repo fetch_interval '15m', got %q (ok=%v)", v, ok)
	}
}

func TestSetupSharedWithPersonalLocalOnly(t *testing.T) {
	repoRoot, _, cleanup := setupTestRepoWithIsolatedHome(t)
	defer cleanup()

	oldWd, _ := os.Getwd()
	_ = os.Chdir(repoRoot)
	defer func() { _ = os.Chdir(oldWd) }()

	// Provide: worktree_dir, branch_pattern, default_branch, accept personal, remote="" (empty)
	// fetch_interval should NOT be prompted
	stdout, _, err := executeCommandWithInput("shared-trees\n{name}\nmain\ny\n\n", "setup", "--shared")
	if err != nil {
		t.Fatalf("setup --shared with personal local-only failed: %v", err)
	}

	if !strings.Contains(stdout, "Personal settings saved") {
		t.Errorf("expected personal save confirmation, got: %s", stdout)
	}

	// Verify remote is empty and no fetch_interval
	userCfg, err := userconfig.Load()
	if err != nil {
		t.Fatalf("failed to load user config: %v", err)
	}

	if v, ok := userCfg.GetForRepo(repoRoot, "remote"); !ok || v != "" {
		t.Errorf("expected empty remote, got %q (ok=%v)", v, ok)
	}
	if _, ok := userCfg.GetForRepo(repoRoot, "fetch_interval"); ok {
		t.Error("expected no fetch_interval for local-only mode")
	}
}

func TestSetupSharedInvalidFetchInterval(t *testing.T) {
	repoRoot, _, cleanup := setupTestRepoWithIsolatedHome(t)
	defer cleanup()

	oldWd, _ := os.Getwd()
	_ = os.Chdir(repoRoot)
	defer func() { _ = os.Chdir(oldWd) }()

	// Provide: worktree_dir, branch_pattern, default_branch, accept personal, remote=origin, invalid fetch
	_, _, err := executeCommandWithInput("trees\n{name}\nmain\ny\norigin\nbad\n", "setup", "--shared")
	if err == nil {
		t.Error("expected error for invalid fetch interval")
	}
}

func TestSetupPersonalInvalidFetchInterval(t *testing.T) {
	repoRoot, _, cleanup := setupTestRepoWithIsolatedHome(t)
	defer cleanup()

	oldWd, _ := os.Getwd()
	_ = os.Chdir(repoRoot)
	defer func() { _ = os.Chdir(oldWd) }()

	// Provide: worktree_dir, branch_pattern, default_branch, remote=origin, invalid fetch
	_, _, err := executeCommandWithInput("trees\n{name}\nmain\norigin\nbad\n", "setup", "--personal")
	if err == nil {
		t.Error("expected error for invalid fetch interval")
	}
}

func TestSetupSharedAcceptsDefaults(t *testing.T) {
	repoRoot, _, cleanup := setupTestRepoWithIsolatedHome(t)
	defer cleanup()

	oldWd, _ := os.Getwd()
	_ = os.Chdir(repoRoot)
	defer func() { _ = os.Chdir(oldWd) }()

	// Accept all defaults (enter through everything), decline personal
	stdout, _, err := executeCommandWithInput("\n\n\nn\n", "setup", "--shared")
	if err != nil {
		t.Fatalf("setup --shared with defaults failed: %v", err)
	}

	if !strings.Contains(stdout, "Configuration saved") {
		t.Errorf("expected save confirmation, got: %s", stdout)
	}

	// Verify .wt.yaml was saved with defaults
	repoCfg, err := config.Load(repoRoot)
	if err != nil {
		t.Fatalf("failed to load .wt.yaml: %v", err)
	}

	if repoCfg.BranchPattern != "{name}" {
		t.Errorf("expected default branch_pattern '{name}', got %q", repoCfg.BranchPattern)
	}
}

func TestSetupPromptChoosesGlobal(t *testing.T) {
	_, _, cleanup := setupTestRepoWithIsolatedHome(t)
	defer cleanup()

	// Choose "1" (global), then provide: remote=origin, fetch_interval=10m
	stdout, _, err := executeCommandWithInput("1\norigin\n10m\n", "setup")
	if err != nil {
		t.Fatalf("setup (prompt global) failed: %v", err)
	}

	if !strings.Contains(stdout, "Configuration saved") {
		t.Errorf("expected save confirmation, got: %s", stdout)
	}

	cfg, err := userconfig.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Remote != "origin" {
		t.Errorf("expected remote 'origin', got %q", cfg.Remote)
	}
}

func TestSetupPromptChoosesPersonal(t *testing.T) {
	repoRoot, _, cleanup := setupTestRepoWithIsolatedHome(t)
	defer cleanup()

	oldWd, _ := os.Getwd()
	_ = os.Chdir(repoRoot)
	defer func() { _ = os.Chdir(oldWd) }()

	// Choose "2" (personal), then provide: worktree_dir, branch_pattern, default_branch, remote=""
	stdout, _, err := executeCommandWithInput("2\nmy-trees\n{name}\nmain\n\n", "setup")
	if err != nil {
		t.Fatalf("setup (prompt personal) failed: %v", err)
	}

	if !strings.Contains(stdout, "Configuration saved") {
		t.Errorf("expected save confirmation, got: %s", stdout)
	}

	// Verify saved to user config, not .wt.yaml
	cfg, err := userconfig.Load()
	if err != nil {
		t.Fatalf("failed to load user config: %v", err)
	}

	if v, ok := cfg.GetForRepo(repoRoot, "worktree_dir"); !ok || v != "my-trees" {
		t.Errorf("expected worktree_dir 'my-trees', got %q (ok=%v)", v, ok)
	}
}

func TestSetupPromptChoosesShared(t *testing.T) {
	repoRoot, _, cleanup := setupTestRepoWithIsolatedHome(t)
	defer cleanup()

	oldWd, _ := os.Getwd()
	_ = os.Chdir(repoRoot)
	defer func() { _ = os.Chdir(oldWd) }()

	// Choose "3" (shared), then: worktree_dir, branch_pattern, default_branch, decline personal
	stdout, _, err := executeCommandWithInput("3\nshared-trees\n{name}\nmain\nn\n", "setup")
	if err != nil {
		t.Fatalf("setup (prompt shared) failed: %v", err)
	}

	if !strings.Contains(stdout, "Configuration saved") {
		t.Errorf("expected save confirmation, got: %s", stdout)
	}

	// Verify saved to .wt.yaml
	repoCfg, err := config.Load(repoRoot)
	if err != nil {
		t.Fatalf("failed to load .wt.yaml: %v", err)
	}

	if repoCfg.WorktreeDir != "shared-trees" {
		t.Errorf("expected worktree_dir 'shared-trees', got %q", repoCfg.WorktreeDir)
	}
}

func TestSetupPromptDefaultIsGlobal(t *testing.T) {
	_, _, cleanup := setupTestRepoWithIsolatedHome(t)
	defer cleanup()

	// Accept default (empty = "1" = global), then provide: remote=""
	stdout, _, err := executeCommandWithInput("\n\n", "setup")
	if err != nil {
		t.Fatalf("setup (prompt default) failed: %v", err)
	}

	if !strings.Contains(stdout, "Configuration saved") {
		t.Errorf("expected save confirmation, got: %s", stdout)
	}
}
