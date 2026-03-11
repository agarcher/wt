package config

import (
	"path/filepath"

	"github.com/agarcher/wt/internal/userconfig"
)

// ConfigSource indicates where a resolved config value came from
type ConfigSource int

const (
	SourceDefault  ConfigSource = iota // No .wt.yaml and no user per-repo config
	SourceRepoFile                     // .wt.yaml present, no user per-repo overrides
	SourceUserRepo                     // User per-repo config contributed at least one field
)

// ResolvedConfig wraps a Config with its source information
type ResolvedConfig struct {
	*Config
	Source ConfigSource
}

// Resolve produces a merged config for the given repo root.
// Resolution order (later wins):
//  1. DefaultConfig()
//  2. .wt.yaml (if present) — sets WorktreeDir, BranchPattern, DefaultBranch, Hooks, Index
//  3. User per-repo config — WorktreeDir, BranchPattern, DefaultBranch override .wt.yaml
//
// Hooks and Index only come from .wt.yaml (never from personal config).
func Resolve(repoRoot string) (*ResolvedConfig, error) {
	cfg := DefaultConfig()
	source := SourceDefault

	// Layer .wt.yaml on top if present
	if repoCfg, err := Load(repoRoot); err == nil {
		cfg = repoCfg
		source = SourceRepoFile
	}

	// Compute default worktree dir for repos without .wt.yaml
	// Use sibling directory: ../<basename>-worktrees
	if source == SourceDefault {
		basename := filepath.Base(repoRoot)
		cfg.WorktreeDir = filepath.Join("..", basename+"-worktrees")
	}

	// Layer user per-repo config on top
	userCfg, err := userconfig.Load()
	if err != nil {
		return &ResolvedConfig{Config: cfg, Source: source}, nil
	}

	repoConfig, hasRepo := userCfg.Repos[repoRoot]
	if !hasRepo {
		return &ResolvedConfig{Config: cfg, Source: source}, nil
	}

	if repoConfig.WorktreeDir != nil {
		cfg.WorktreeDir = *repoConfig.WorktreeDir
		source = SourceUserRepo
	}
	if repoConfig.BranchPattern != nil {
		cfg.BranchPattern = *repoConfig.BranchPattern
		source = SourceUserRepo
	}
	if repoConfig.DefaultBranch != nil {
		cfg.DefaultBranch = *repoConfig.DefaultBranch
		source = SourceUserRepo
	}

	return &ResolvedConfig{Config: cfg, Source: source}, nil
}
