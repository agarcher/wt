package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	ConfigFileName = ".wt.yaml"
)

// IndexConfig contains worktree index configuration
type IndexConfig struct {
	Max int `yaml:"max"` // Maximum allowed index (0 = no limit)
}

// Config represents the repository-level configuration
type Config struct {
	Version       int          `yaml:"version"`
	WorktreeDir   string       `yaml:"worktree_dir"`
	BranchPattern string       `yaml:"branch_pattern"`
	DefaultBranch string       `yaml:"default_branch"` // Branch to compare against (e.g., "main", "develop")
	Hooks         HooksConfig  `yaml:"hooks"`
	Index         IndexConfig  `yaml:"index"`
}

// HooksConfig contains all lifecycle hook configurations
type HooksConfig struct {
	PreCreate  []HookEntry `yaml:"pre_create"`
	PostCreate []HookEntry `yaml:"post_create"`
	PreDelete  []HookEntry `yaml:"pre_delete"`
	PostDelete []HookEntry `yaml:"post_delete"`
	Info       []HookEntry `yaml:"info"`
}

// HookEntry represents a single hook script configuration
type HookEntry struct {
	Script string            `yaml:"script"`
	Env    map[string]string `yaml:"env"`
}

// DefaultConfig returns a config with default values
func DefaultConfig() *Config {
	return &Config{
		Version:       1,
		WorktreeDir:   "worktrees",
		BranchPattern: "{name}",
	}
}

// Load reads the configuration from the given repository root
func Load(repoRoot string) (*Config, error) {
	configPath := filepath.Join(repoRoot, ConfigFileName)

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, err
		}
		return nil, err
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// Apply defaults for empty values
	if cfg.WorktreeDir == "" {
		cfg.WorktreeDir = "worktrees"
	}
	if cfg.BranchPattern == "" {
		cfg.BranchPattern = "{name}"
	}

	return cfg, nil
}

// Save writes the configuration to .wt.yaml in the given repository root.
// Uses atomic write (temp file + rename) to prevent corruption if interrupted.
func Save(repoRoot string, cfg *Config) error {
	configPath := filepath.Join(repoRoot, ConfigFileName)

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to temp file first for atomic save
	tempFile, err := os.CreateTemp(repoRoot, ".wt.yaml.tmp.*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()

	// Clean up temp file on any error
	success := false
	defer func() {
		if !success {
			_ = os.Remove(tempPath)
		}
	}()

	if _, err := tempFile.Write(data); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("failed to write config: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	success = true
	return nil
}

// Exists checks if a config file exists in the given repository root
func Exists(repoRoot string) bool {
	configPath := filepath.Join(repoRoot, ConfigFileName)
	_, err := os.Stat(configPath)
	return err == nil
}

// GetRepoRoot finds the root of the git repository from the current directory
func GetRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		gitDir := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitDir); err == nil {
			// .git can be a directory (normal repo) or a file (worktree)
			if info.IsDir() || info.Mode().IsRegular() {
				return dir, nil
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

// GetMainRepoRoot finds the main repository root, even if we're in a worktree
func GetMainRepoRoot() (string, error) {
	dir, err := GetRepoRoot()
	if err != nil {
		return "", err
	}

	gitPath := filepath.Join(dir, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return "", err
	}

	var repoRoot string

	// If .git is a directory, this is the main repo
	if info.IsDir() {
		repoRoot = dir
	} else {
		// If .git is a file, read it to find the main repo
		// Format: "gitdir: /path/to/main/.git/worktrees/name"
		data, err := os.ReadFile(gitPath)
		if err != nil {
			return "", err
		}

		// Parse the gitdir path
		content := string(data)
		if len(content) <= 8 || content[:8] != "gitdir: " {
			repoRoot = dir // fallback to current dir
		} else {
			gitdir := strings.TrimSpace(content[8:])
			if gitdir == "" {
				repoRoot = dir // fallback to current dir
			} else {
				gitdir = filepath.Clean(gitdir)

				// Navigate up from .git/worktrees/name to the main repo
				// gitdir is like: /path/to/main/.git/worktrees/name
				mainGitDir := filepath.Dir(filepath.Dir(gitdir))
				repoRoot = filepath.Dir(mainGitDir)
			}
		}
	}

	// Resolve symlinks to ensure path consistency with git worktree output
	return filepath.EvalSymlinks(repoRoot)
}
