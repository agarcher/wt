package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/agarcher/wt/internal/userconfig"
)

func TestResolveDefault(t *testing.T) {
	// Create a temp directory that acts as a repo root (no .wt.yaml)
	tmpDir, err := os.MkdirTemp("", "wt-resolve-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Override HOME so userconfig doesn't find any personal config
	oldHome := os.Getenv("HOME")
	homeDir, _ := os.MkdirTemp("", "wt-resolve-home")
	defer func() { _ = os.RemoveAll(homeDir) }()
	_ = os.Setenv("HOME", homeDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	resolved, err := Resolve(tmpDir)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if resolved.Source != SourceDefault {
		t.Errorf("expected SourceDefault, got %d", resolved.Source)
	}

	// Default for repos without .wt.yaml should be sibling dir
	basename := filepath.Base(tmpDir)
	expectedDir := filepath.Join("..", basename+"-worktrees")
	if resolved.WorktreeDir != expectedDir {
		t.Errorf("expected worktree_dir %q, got %q", expectedDir, resolved.WorktreeDir)
	}
	if resolved.BranchPattern != "{name}" {
		t.Errorf("expected branch_pattern '{name}', got %q", resolved.BranchPattern)
	}
}

func TestResolveWithWtYaml(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wt-resolve-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Override HOME
	oldHome := os.Getenv("HOME")
	homeDir, _ := os.MkdirTemp("", "wt-resolve-home")
	defer func() { _ = os.RemoveAll(homeDir) }()
	_ = os.Setenv("HOME", homeDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Create .wt.yaml
	wtYaml := []byte("version: 1\nworktree_dir: custom-worktrees\nbranch_pattern: feat/{name}\ndefault_branch: develop\n")
	if err := os.WriteFile(filepath.Join(tmpDir, ".wt.yaml"), wtYaml, 0644); err != nil {
		t.Fatalf("failed to write .wt.yaml: %v", err)
	}

	resolved, err := Resolve(tmpDir)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if resolved.Source != SourceRepoFile {
		t.Errorf("expected SourceRepoFile, got %d", resolved.Source)
	}
	if resolved.WorktreeDir != "custom-worktrees" {
		t.Errorf("expected worktree_dir 'custom-worktrees', got %q", resolved.WorktreeDir)
	}
	if resolved.BranchPattern != "feat/{name}" {
		t.Errorf("expected branch_pattern 'feat/{name}', got %q", resolved.BranchPattern)
	}
	if resolved.DefaultBranch != "develop" {
		t.Errorf("expected default_branch 'develop', got %q", resolved.DefaultBranch)
	}
}

func TestResolvePersonalOverridesWtYaml(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wt-resolve-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Override HOME
	oldHome := os.Getenv("HOME")
	homeDir, _ := os.MkdirTemp("", "wt-resolve-home")
	defer func() { _ = os.RemoveAll(homeDir) }()
	_ = os.Setenv("HOME", homeDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Create .wt.yaml
	wtYaml := []byte("version: 1\nworktree_dir: team-worktrees\nbranch_pattern: feat/{name}\ndefault_branch: develop\n")
	if err := os.WriteFile(filepath.Join(tmpDir, ".wt.yaml"), wtYaml, 0644); err != nil {
		t.Fatalf("failed to write .wt.yaml: %v", err)
	}

	// Create personal config that overrides worktree_dir
	cfg := userconfig.DefaultUserConfig()
	personalDir := "../my-worktrees"
	cfg.Repos[tmpDir] = userconfig.RepoConfig{
		WorktreeDir: &personalDir,
	}
	if err := userconfig.Save(cfg); err != nil {
		t.Fatalf("failed to save user config: %v", err)
	}

	resolved, err := Resolve(tmpDir)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if resolved.Source != SourceUserRepo {
		t.Errorf("expected SourceUserRepo, got %d", resolved.Source)
	}
	// Personal config should override .wt.yaml
	if resolved.WorktreeDir != "../my-worktrees" {
		t.Errorf("expected worktree_dir '../my-worktrees', got %q", resolved.WorktreeDir)
	}
	// .wt.yaml values should still be used for non-overridden fields
	if resolved.BranchPattern != "feat/{name}" {
		t.Errorf("expected branch_pattern 'feat/{name}', got %q", resolved.BranchPattern)
	}
	if resolved.DefaultBranch != "develop" {
		t.Errorf("expected default_branch 'develop', got %q", resolved.DefaultBranch)
	}
}

func TestResolveCorruptUserConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wt-resolve-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	oldHome := os.Getenv("HOME")
	homeDir, _ := os.MkdirTemp("", "wt-resolve-home")
	defer func() { _ = os.RemoveAll(homeDir) }()
	_ = os.Setenv("HOME", homeDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Write corrupt user config
	configDir := filepath.Join(homeDir, ".config", "wt")
	_ = os.MkdirAll(configDir, 0755)
	_ = os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("{{invalid yaml"), 0644)

	// Create .wt.yaml
	wtYaml := []byte("version: 1\nworktree_dir: team-dir\n")
	_ = os.WriteFile(filepath.Join(tmpDir, ".wt.yaml"), wtYaml, 0644)

	// Should still resolve using .wt.yaml, ignoring corrupt user config
	resolved, err := Resolve(tmpDir)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if resolved.Source != SourceRepoFile {
		t.Errorf("expected SourceRepoFile, got %d", resolved.Source)
	}
	if resolved.WorktreeDir != "team-dir" {
		t.Errorf("expected worktree_dir 'team-dir', got %q", resolved.WorktreeDir)
	}
}

func TestResolvePartialPersonalOverrides(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wt-resolve-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	oldHome := os.Getenv("HOME")
	homeDir, _ := os.MkdirTemp("", "wt-resolve-home")
	defer func() { _ = os.RemoveAll(homeDir) }()
	_ = os.Setenv("HOME", homeDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Create .wt.yaml with all three fields
	wtYaml := []byte("version: 1\nworktree_dir: team-dir\nbranch_pattern: feat/{name}\ndefault_branch: develop\n")
	_ = os.WriteFile(filepath.Join(tmpDir, ".wt.yaml"), wtYaml, 0644)

	// Override only branch_pattern in personal config
	cfg := userconfig.DefaultUserConfig()
	bp := "user/{name}"
	cfg.Repos[tmpDir] = userconfig.RepoConfig{
		BranchPattern: &bp,
	}
	_ = userconfig.Save(cfg)

	resolved, err := Resolve(tmpDir)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if resolved.Source != SourceUserRepo {
		t.Errorf("expected SourceUserRepo, got %d", resolved.Source)
	}
	// Only branch_pattern should be overridden
	if resolved.WorktreeDir != "team-dir" {
		t.Errorf("expected worktree_dir 'team-dir', got %q", resolved.WorktreeDir)
	}
	if resolved.BranchPattern != "user/{name}" {
		t.Errorf("expected branch_pattern 'user/{name}', got %q", resolved.BranchPattern)
	}
	if resolved.DefaultBranch != "develop" {
		t.Errorf("expected default_branch 'develop', got %q", resolved.DefaultBranch)
	}
}

func TestResolvePreservesHooksFromWtYaml(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wt-resolve-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	oldHome := os.Getenv("HOME")
	homeDir, _ := os.MkdirTemp("", "wt-resolve-home")
	defer func() { _ = os.RemoveAll(homeDir) }()
	_ = os.Setenv("HOME", homeDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Create .wt.yaml with hooks
	wtYaml := []byte(`version: 1
worktree_dir: team-dir
hooks:
  post_create:
    - script: ./scripts/setup.sh
  info:
    - script: ./scripts/info.sh
`)
	_ = os.WriteFile(filepath.Join(tmpDir, ".wt.yaml"), wtYaml, 0644)

	// Override worktree_dir in personal config
	cfg := userconfig.DefaultUserConfig()
	dir := "../my-trees"
	cfg.Repos[tmpDir] = userconfig.RepoConfig{
		WorktreeDir: &dir,
	}
	_ = userconfig.Save(cfg)

	resolved, err := Resolve(tmpDir)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// Hooks should be preserved from .wt.yaml
	if len(resolved.Hooks.PostCreate) != 1 {
		t.Errorf("expected 1 post_create hook, got %d", len(resolved.Hooks.PostCreate))
	}
	if len(resolved.Hooks.Info) != 1 {
		t.Errorf("expected 1 info hook, got %d", len(resolved.Hooks.Info))
	}
	if resolved.Hooks.PostCreate[0].Script != "./scripts/setup.sh" {
		t.Errorf("expected post_create script './scripts/setup.sh', got %q", resolved.Hooks.PostCreate[0].Script)
	}
	// worktree_dir should be the personal override
	if resolved.WorktreeDir != "../my-trees" {
		t.Errorf("expected worktree_dir '../my-trees', got %q", resolved.WorktreeDir)
	}
}
