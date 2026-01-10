package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Version != 1 {
		t.Errorf("expected version 1, got %d", cfg.Version)
	}
	if cfg.WorktreeDir != "worktrees" {
		t.Errorf("expected worktree_dir 'worktrees', got %q", cfg.WorktreeDir)
	}
	if cfg.BranchPattern != "{name}" {
		t.Errorf("expected branch_pattern '{name}', got %q", cfg.BranchPattern)
	}
}

func TestLoad(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "wt-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	tests := []struct {
		name        string
		configYAML  string
		wantErr     bool
		checkConfig func(*testing.T, *Config)
	}{
		{
			name: "basic config",
			configYAML: `version: 1
worktree_dir: my-worktrees
branch_pattern: "feature/{name}"
`,
			wantErr: false,
			checkConfig: func(t *testing.T, cfg *Config) {
				if cfg.Version != 1 {
					t.Errorf("expected version 1, got %d", cfg.Version)
				}
				if cfg.WorktreeDir != "my-worktrees" {
					t.Errorf("expected worktree_dir 'my-worktrees', got %q", cfg.WorktreeDir)
				}
				if cfg.BranchPattern != "feature/{name}" {
					t.Errorf("expected branch_pattern 'feature/{name}', got %q", cfg.BranchPattern)
				}
			},
		},
		{
			name: "config with hooks",
			configYAML: `version: 1
worktree_dir: worktrees
hooks:
  post_create:
    - script: ./setup.sh
    - script: ./install.sh
      env:
        DEBUG: "true"
  pre_delete:
    - script: ./cleanup.sh
`,
			wantErr: false,
			checkConfig: func(t *testing.T, cfg *Config) {
				if len(cfg.Hooks.PostCreate) != 2 {
					t.Errorf("expected 2 post_create hooks, got %d", len(cfg.Hooks.PostCreate))
				}
				if cfg.Hooks.PostCreate[0].Script != "./setup.sh" {
					t.Errorf("expected first hook script './setup.sh', got %q", cfg.Hooks.PostCreate[0].Script)
				}
				if cfg.Hooks.PostCreate[1].Env["DEBUG"] != "true" {
					t.Errorf("expected DEBUG env var 'true', got %q", cfg.Hooks.PostCreate[1].Env["DEBUG"])
				}
				if len(cfg.Hooks.PreDelete) != 1 {
					t.Errorf("expected 1 pre_delete hook, got %d", len(cfg.Hooks.PreDelete))
				}
			},
		},
		{
			name: "minimal config with defaults",
			configYAML: `version: 1
`,
			wantErr: false,
			checkConfig: func(t *testing.T, cfg *Config) {
				if cfg.WorktreeDir != "worktrees" {
					t.Errorf("expected default worktree_dir 'worktrees', got %q", cfg.WorktreeDir)
				}
				if cfg.BranchPattern != "{name}" {
					t.Errorf("expected default branch_pattern '{name}', got %q", cfg.BranchPattern)
				}
			},
		},
		{
			name:       "invalid yaml",
			configYAML: `version: [invalid`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create subdirectory for this test
			testDir := filepath.Join(tmpDir, tt.name)
			if err := os.MkdirAll(testDir, 0755); err != nil {
				t.Fatalf("failed to create test dir: %v", err)
			}

			// Write config file
			configPath := filepath.Join(testDir, ConfigFileName)
			if err := os.WriteFile(configPath, []byte(tt.configYAML), 0644); err != nil {
				t.Fatalf("failed to write config: %v", err)
			}

			// Load config
			cfg, err := Load(testDir)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.checkConfig != nil {
				tt.checkConfig(t, cfg)
			}
		})
	}
}

func TestLoadNotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wt-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	_, err = Load(tmpDir)
	if err == nil {
		t.Error("expected error for missing config, got nil")
	}
}

func TestExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wt-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Should not exist initially
	if Exists(tmpDir) {
		t.Error("expected config to not exist")
	}

	// Create config file
	configPath := filepath.Join(tmpDir, ConfigFileName)
	if err := os.WriteFile(configPath, []byte("version: 1"), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Should exist now
	if !Exists(tmpDir) {
		t.Error("expected config to exist")
	}
}

func TestGetRepoRoot(t *testing.T) {
	// Create a temporary directory with .git
	tmpDir, err := os.MkdirTemp("", "wt-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Resolve symlinks (macOS /var -> /private/var)
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatalf("failed to eval symlinks: %v", err)
	}

	// Create .git directory
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}

	// Create a subdirectory
	subDir := filepath.Join(tmpDir, "sub", "dir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create sub dir: %v", err)
	}

	// Save current dir and change to subdir
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	defer func() { _ = os.Chdir(oldDir) }()

	if err := os.Chdir(subDir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}

	// GetRepoRoot should find the repo root
	root, err := GetRepoRoot()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if root != tmpDir {
		t.Errorf("expected root %q, got %q", tmpDir, root)
	}
}

func TestGetRepoRootNotInRepo(t *testing.T) {
	// Create a temporary directory without .git
	tmpDir, err := os.MkdirTemp("", "wt-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Save current dir
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	defer func() { _ = os.Chdir(oldDir) }()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}

	_, err = GetRepoRoot()
	if err == nil {
		t.Error("expected error when not in repo, got nil")
	}
}
