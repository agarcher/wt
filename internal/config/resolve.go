package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/agarcher/wt/internal/userconfig"
	"gopkg.in/yaml.v3"
)

// repoFilePresence is used to detect which fields were explicitly set in .wt.yaml
type repoFilePresence struct {
	WorktreeDir   *string `yaml:"worktree_dir"`
	BranchPattern *string `yaml:"branch_pattern"`
	DefaultBranch *string `yaml:"default_branch"`
}

// detectRepoFileFields checks which fields are explicitly present in .wt.yaml
func detectRepoFileFields(repoRoot string) (worktreeDir, branchPattern, defaultBranch bool) {
	data, err := os.ReadFile(filepath.Join(repoRoot, ConfigFileName))
	if err != nil {
		return false, false, false
	}
	var presence repoFilePresence
	if err := yaml.Unmarshal(data, &presence); err != nil {
		return false, false, false
	}
	return presence.WorktreeDir != nil, presence.BranchPattern != nil, presence.DefaultBranch != nil
}

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
	Source  ConfigSource
	Warning string // Non-empty if config files could not be loaded

	// Per-field provenance: true means the field was explicitly set in .wt.yaml
	WorktreeDirFromRepo   bool
	BranchPatternFromRepo bool
	DefaultBranchFromRepo bool
}

// Resolve produces a merged config for the given repo root.
// Resolution order (later wins):
//  1. DefaultConfig()
//  2. .wt.yaml (if present) — sets WorktreeDir, BranchPattern, DefaultBranch, Hooks, Index
//  3. User per-repo config — WorktreeDir, BranchPattern, DefaultBranch override .wt.yaml
//
// Hooks and Index only come from .wt.yaml (never from personal config).
func Resolve(repoRoot string) *ResolvedConfig {
	cfg := DefaultConfig()
	source := SourceDefault

	// Layer .wt.yaml on top if present
	var warning string
	var wtFromRepo, bpFromRepo, dbFromRepo bool
	if repoCfg, err := Load(repoRoot); err == nil {
		cfg = repoCfg
		source = SourceRepoFile
		// Detect which fields were explicitly set in .wt.yaml
		wtFromRepo, bpFromRepo, dbFromRepo = detectRepoFileFields(repoRoot)
	} else if !errors.Is(err, fs.ErrNotExist) {
		// .wt.yaml exists but is corrupt or unreadable
		warning = fmt.Sprintf("Warning: failed to load %s: %v (using defaults)", ConfigFileName, err)
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
		userWarning := fmt.Sprintf("Warning: failed to load user config: %v (using defaults)", err)
		if warning != "" {
			warning += "; " + userWarning
		} else {
			warning = userWarning
		}
		return &ResolvedConfig{
		Config: cfg, Source: source, Warning: warning,
		WorktreeDirFromRepo: wtFromRepo, BranchPatternFromRepo: bpFromRepo, DefaultBranchFromRepo: dbFromRepo,
	}
	}

	repoConfig, hasRepo := userCfg.Repos[repoRoot]
	if !hasRepo {
		return &ResolvedConfig{
		Config: cfg, Source: source, Warning: warning,
		WorktreeDirFromRepo: wtFromRepo, BranchPatternFromRepo: bpFromRepo, DefaultBranchFromRepo: dbFromRepo,
	}
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

	return &ResolvedConfig{
		Config: cfg, Source: source, Warning: warning,
		WorktreeDirFromRepo: wtFromRepo, BranchPatternFromRepo: bpFromRepo, DefaultBranchFromRepo: dbFromRepo,
	}
}
