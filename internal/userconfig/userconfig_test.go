package userconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultUserConfig(t *testing.T) {
	cfg := DefaultUserConfig()

	if cfg.Remote != "" {
		t.Errorf("expected empty remote, got %q", cfg.Remote)
	}
	if cfg.FetchInterval != DefaultFetchInterval {
		t.Errorf("expected fetch_interval to be %q, got %q", DefaultFetchInterval, cfg.FetchInterval)
	}
	if cfg.Repos == nil {
		t.Error("expected Repos to be initialized")
	}
}

func TestGetRemoteForRepo(t *testing.T) {
	upstream := "upstream"
	empty := ""
	cfg := &UserConfig{
		Remote: "origin",
		Repos: map[string]RepoConfig{
			"/path/to/repo1": {Remote: &upstream},
			"/path/to/repo2": {Remote: &empty}, // explicit empty override
		},
	}

	tests := []struct {
		name     string
		repoPath string
		want     string
	}{
		{"uses per-repo override", "/path/to/repo1", "upstream"},
		{"explicit empty override", "/path/to/repo2", ""},
		{"falls back to global", "/path/to/repo3", "origin"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cfg.GetRemoteForRepo(tt.repoPath)
			if got != tt.want {
				t.Errorf("GetRemoteForRepo(%q) = %q, want %q", tt.repoPath, got, tt.want)
			}
		})
	}
}

func TestSetGlobal(t *testing.T) {
	cfg := DefaultUserConfig()

	if err := cfg.SetGlobal("remote", "upstream"); err != nil {
		t.Errorf("SetGlobal failed: %v", err)
	}
	if cfg.Remote != "upstream" {
		t.Errorf("expected remote to be 'upstream', got %q", cfg.Remote)
	}

	if err := cfg.SetGlobal("fetch_interval", "10m"); err != nil {
		t.Errorf("SetGlobal failed: %v", err)
	}
	if cfg.FetchInterval != "10m" {
		t.Errorf("expected fetch_interval to be '10m', got %q", cfg.FetchInterval)
	}

	if err := cfg.SetGlobal("unknown", "value"); err == nil {
		t.Error("expected error for unknown key")
	}
}

func TestSetForRepo(t *testing.T) {
	cfg := DefaultUserConfig()
	repoPath := "/path/to/repo"

	if err := cfg.SetForRepo(repoPath, "remote", "upstream"); err != nil {
		t.Errorf("SetForRepo failed: %v", err)
	}
	if cfg.Repos[repoPath].Remote == nil || *cfg.Repos[repoPath].Remote != "upstream" {
		t.Errorf("expected remote to be 'upstream'")
	}

	if err := cfg.SetForRepo(repoPath, "fetch_interval", "10m"); err != nil {
		t.Errorf("SetForRepo failed: %v", err)
	}
	if cfg.Repos[repoPath].FetchInterval == nil || *cfg.Repos[repoPath].FetchInterval != "10m" {
		t.Errorf("expected fetch_interval to be '10m'")
	}

	// Test setting remote to empty string explicitly
	if err := cfg.SetForRepo(repoPath, "remote", ""); err != nil {
		t.Errorf("SetForRepo failed: %v", err)
	}
	if cfg.Repos[repoPath].Remote == nil {
		t.Errorf("expected remote to be set (not nil)")
	} else if *cfg.Repos[repoPath].Remote != "" {
		t.Errorf("expected remote to be empty string, got %q", *cfg.Repos[repoPath].Remote)
	}
}

func TestUnsetForRepo(t *testing.T) {
	intervalVal := "10m"
	remoteVal := "upstream"
	cfg := &UserConfig{
		Repos: map[string]RepoConfig{
			"/path/to/repo": {Remote: &remoteVal, FetchInterval: &intervalVal},
		},
	}

	// Unset remote but keep fetch_interval
	if err := cfg.UnsetForRepo("/path/to/repo", "remote"); err != nil {
		t.Errorf("UnsetForRepo failed: %v", err)
	}
	if cfg.Repos["/path/to/repo"].Remote != nil {
		t.Errorf("expected remote to be nil")
	}
	if cfg.Repos["/path/to/repo"].FetchInterval == nil {
		t.Errorf("expected fetch_interval to still be set")
	}

	// Unset fetch_interval too - should remove the entire repo entry
	if err := cfg.UnsetForRepo("/path/to/repo", "fetch_interval"); err != nil {
		t.Errorf("UnsetForRepo failed: %v", err)
	}
	if _, ok := cfg.Repos["/path/to/repo"]; ok {
		t.Errorf("expected repo entry to be removed")
	}
}

func TestUnsetGlobal(t *testing.T) {
	cfg := &UserConfig{
		Remote:        "origin",
		FetchInterval: "10m",
	}

	// Unset remote
	if err := cfg.UnsetGlobal("remote"); err != nil {
		t.Errorf("UnsetGlobal failed: %v", err)
	}
	if cfg.Remote != "" {
		t.Errorf("expected remote to be empty, got %q", cfg.Remote)
	}
	if cfg.FetchInterval != "10m" {
		t.Errorf("expected fetch_interval to still be '10m'")
	}

	// Unset fetch_interval
	if err := cfg.UnsetGlobal("fetch_interval"); err != nil {
		t.Errorf("UnsetGlobal failed: %v", err)
	}
	if cfg.FetchInterval != "" {
		t.Errorf("expected fetch_interval to be empty, got %q", cfg.FetchInterval)
	}

	// Invalid key
	if err := cfg.UnsetGlobal("invalid"); err == nil {
		t.Error("expected error for invalid key")
	}
}

func TestLoadAndSave(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "wt-userconfig-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Override home dir for test
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Test saving
	cfg := DefaultUserConfig()
	cfg.Remote = "origin"
	cfg.FetchInterval = "10m"
	upstream := "upstream"
	cfg.Repos["/path/to/repo"] = RepoConfig{Remote: &upstream}

	if err := Save(cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	configPath := filepath.Join(tmpDir, ConfigDir, ConfigFile)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("config file not created at %s", configPath)
	}

	// Test loading
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.Remote != "origin" {
		t.Errorf("expected remote 'origin', got %q", loaded.Remote)
	}
	if loaded.FetchInterval != "10m" {
		t.Errorf("expected fetch_interval '10m', got %q", loaded.FetchInterval)
	}
	if loaded.Repos["/path/to/repo"].Remote == nil || *loaded.Repos["/path/to/repo"].Remote != "upstream" {
		t.Errorf("expected per-repo remote 'upstream'")
	}
}

func TestLoadNonexistent(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "wt-userconfig-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Override home dir for test
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Load should return defaults when file doesn't exist
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Remote != "" {
		t.Errorf("expected empty remote, got %q", cfg.Remote)
	}
	if cfg.FetchInterval != DefaultFetchInterval {
		t.Errorf("expected fetch_interval to be default %q, got %q", DefaultFetchInterval, cfg.FetchInterval)
	}
}

func TestValidKeys(t *testing.T) {
	keys := ValidKeys()
	if len(keys) != 5 {
		t.Errorf("expected 5 valid keys, got %d", len(keys))
	}

	expected := map[string]bool{
		"remote": true, "fetch_interval": true,
		"worktree_dir": true, "branch_pattern": true, "default_branch": true,
	}
	for _, key := range keys {
		if !expected[key] {
			t.Errorf("unexpected key: %s", key)
		}
	}
}

func TestRepoOnlyKeys(t *testing.T) {
	keys := RepoOnlyKeys()
	expected := map[string]bool{"worktree_dir": true, "branch_pattern": true, "default_branch": true}
	for _, key := range keys {
		if !expected[key] {
			t.Errorf("unexpected repo-only key: %s", key)
		}
	}
}

func TestSetGlobalRepoOnlyKeys(t *testing.T) {
	cfg := DefaultUserConfig()

	for _, key := range RepoOnlyKeys() {
		if err := cfg.SetGlobal(key, "value"); err == nil {
			t.Errorf("SetGlobal(%q) should return error for repo-only key", key)
		}
		if _, err := cfg.GetGlobal(key); err == nil {
			t.Errorf("GetGlobal(%q) should return error for repo-only key", key)
		}
		if err := cfg.UnsetGlobal(key); err == nil {
			t.Errorf("UnsetGlobal(%q) should return error for repo-only key", key)
		}
	}
}

func TestSetForRepoNewKeys(t *testing.T) {
	cfg := DefaultUserConfig()
	repoPath := "/path/to/repo"

	if err := cfg.SetForRepo(repoPath, "worktree_dir", "../my-worktrees"); err != nil {
		t.Errorf("SetForRepo failed: %v", err)
	}
	if v, ok := cfg.GetForRepo(repoPath, "worktree_dir"); !ok || v != "../my-worktrees" {
		t.Errorf("expected worktree_dir '../my-worktrees', got %q (ok=%v)", v, ok)
	}

	if err := cfg.SetForRepo(repoPath, "branch_pattern", "user/{name}"); err != nil {
		t.Errorf("SetForRepo failed: %v", err)
	}
	if v, ok := cfg.GetForRepo(repoPath, "branch_pattern"); !ok || v != "user/{name}" {
		t.Errorf("expected branch_pattern 'user/{name}', got %q (ok=%v)", v, ok)
	}

	if err := cfg.SetForRepo(repoPath, "default_branch", "develop"); err != nil {
		t.Errorf("SetForRepo failed: %v", err)
	}
	if v, ok := cfg.GetForRepo(repoPath, "default_branch"); !ok || v != "develop" {
		t.Errorf("expected default_branch 'develop', got %q (ok=%v)", v, ok)
	}
}

func TestUnsetForRepoNewKeys(t *testing.T) {
	wtDir := "../wt"
	branchPattern := "feat/{name}"
	defaultBranch := "develop"
	cfg := &UserConfig{
		Repos: map[string]RepoConfig{
			"/path/to/repo": {
				WorktreeDir:   &wtDir,
				BranchPattern: &branchPattern,
				DefaultBranch: &defaultBranch,
			},
		},
	}

	// Unset one at a time
	if err := cfg.UnsetForRepo("/path/to/repo", "worktree_dir"); err != nil {
		t.Errorf("UnsetForRepo failed: %v", err)
	}
	if _, ok := cfg.Repos["/path/to/repo"]; !ok {
		t.Fatal("repo entry should still exist after unsetting one field")
	}

	if err := cfg.UnsetForRepo("/path/to/repo", "branch_pattern"); err != nil {
		t.Errorf("UnsetForRepo failed: %v", err)
	}

	// Unsetting the last field should remove the entire entry
	if err := cfg.UnsetForRepo("/path/to/repo", "default_branch"); err != nil {
		t.Errorf("UnsetForRepo failed: %v", err)
	}
	if _, ok := cfg.Repos["/path/to/repo"]; ok {
		t.Error("repo entry should be removed when all fields are nil")
	}
}
