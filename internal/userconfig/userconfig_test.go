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
	if cfg.Fetch != false {
		t.Errorf("expected fetch to be false, got %v", cfg.Fetch)
	}
	if cfg.Repos == nil {
		t.Error("expected Repos to be initialized")
	}
}

func TestGetRemoteForRepo(t *testing.T) {
	cfg := &UserConfig{
		Remote: "origin",
		Repos: map[string]RepoConfig{
			"/path/to/repo1": {Remote: "upstream"},
		},
	}

	tests := []struct {
		name     string
		repoPath string
		want     string
	}{
		{"uses per-repo override", "/path/to/repo1", "upstream"},
		{"falls back to global", "/path/to/repo2", "origin"},
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

func TestGetFetchForRepo(t *testing.T) {
	trueVal := true
	falseVal := false

	cfg := &UserConfig{
		Fetch: true,
		Repos: map[string]RepoConfig{
			"/path/to/repo1": {Fetch: &falseVal},
			"/path/to/repo2": {Fetch: &trueVal},
		},
	}

	tests := []struct {
		name     string
		repoPath string
		want     bool
	}{
		{"per-repo override false", "/path/to/repo1", false},
		{"per-repo override true", "/path/to/repo2", true},
		{"falls back to global", "/path/to/repo3", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cfg.GetFetchForRepo(tt.repoPath)
			if got != tt.want {
				t.Errorf("GetFetchForRepo(%q) = %v, want %v", tt.repoPath, got, tt.want)
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

	if err := cfg.SetGlobal("fetch", "true"); err != nil {
		t.Errorf("SetGlobal failed: %v", err)
	}
	if cfg.Fetch != true {
		t.Errorf("expected fetch to be true, got %v", cfg.Fetch)
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
	if cfg.Repos[repoPath].Remote != "upstream" {
		t.Errorf("expected remote to be 'upstream', got %q", cfg.Repos[repoPath].Remote)
	}

	if err := cfg.SetForRepo(repoPath, "fetch", "true"); err != nil {
		t.Errorf("SetForRepo failed: %v", err)
	}
	if cfg.Repos[repoPath].Fetch == nil || *cfg.Repos[repoPath].Fetch != true {
		t.Errorf("expected fetch to be true")
	}
}

func TestUnsetForRepo(t *testing.T) {
	trueVal := true
	cfg := &UserConfig{
		Repos: map[string]RepoConfig{
			"/path/to/repo": {Remote: "upstream", Fetch: &trueVal},
		},
	}

	// Unset remote but keep fetch
	if err := cfg.UnsetForRepo("/path/to/repo", "remote"); err != nil {
		t.Errorf("UnsetForRepo failed: %v", err)
	}
	if cfg.Repos["/path/to/repo"].Remote != "" {
		t.Errorf("expected remote to be empty")
	}
	if cfg.Repos["/path/to/repo"].Fetch == nil {
		t.Errorf("expected fetch to still be set")
	}

	// Unset fetch too - should remove the entire repo entry
	if err := cfg.UnsetForRepo("/path/to/repo", "fetch"); err != nil {
		t.Errorf("UnsetForRepo failed: %v", err)
	}
	if _, ok := cfg.Repos["/path/to/repo"]; ok {
		t.Errorf("expected repo entry to be removed")
	}
}

func TestUnsetGlobal(t *testing.T) {
	cfg := &UserConfig{
		Remote: "origin",
		Fetch:  true,
	}

	// Unset remote
	if err := cfg.UnsetGlobal("remote"); err != nil {
		t.Errorf("UnsetGlobal failed: %v", err)
	}
	if cfg.Remote != "" {
		t.Errorf("expected remote to be empty, got %q", cfg.Remote)
	}
	if cfg.Fetch != true {
		t.Errorf("expected fetch to still be true")
	}

	// Unset fetch
	if err := cfg.UnsetGlobal("fetch"); err != nil {
		t.Errorf("UnsetGlobal failed: %v", err)
	}
	if cfg.Fetch != false {
		t.Errorf("expected fetch to be false, got %v", cfg.Fetch)
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
	cfg.Fetch = true
	cfg.Repos["/path/to/repo"] = RepoConfig{Remote: "upstream"}

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
	if loaded.Fetch != true {
		t.Errorf("expected fetch true, got %v", loaded.Fetch)
	}
	if loaded.Repos["/path/to/repo"].Remote != "upstream" {
		t.Errorf("expected per-repo remote 'upstream', got %q", loaded.Repos["/path/to/repo"].Remote)
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
	if cfg.Fetch != false {
		t.Errorf("expected fetch false, got %v", cfg.Fetch)
	}
}

func TestValidKeys(t *testing.T) {
	keys := ValidKeys()
	if len(keys) != 3 {
		t.Errorf("expected 3 valid keys, got %d", len(keys))
	}

	expected := map[string]bool{"remote": true, "fetch": true, "fetch_interval": true}
	for _, key := range keys {
		if !expected[key] {
			t.Errorf("unexpected key: %s", key)
		}
	}
}
