package hooks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/agarcher/wt/internal/config"
)

// Env contains environment variables passed to hooks
type Env struct {
	Name        string
	Path        string
	Branch      string
	RepoRoot    string
	WorktreeDir string
}

// ToEnvVars converts the Env struct to environment variable format
func (e *Env) ToEnvVars() []string {
	return []string{
		"WT_NAME=" + e.Name,
		"WT_PATH=" + e.Path,
		"WT_BRANCH=" + e.Branch,
		"WT_REPO_ROOT=" + e.RepoRoot,
		"WT_WORKTREE_DIR=" + e.WorktreeDir,
	}
}

// Run executes a list of hook entries
func Run(entries []config.HookEntry, env *Env, workDir string) error {
	for _, entry := range entries {
		if err := runHook(entry, env, workDir); err != nil {
			return err
		}
	}
	return nil
}

// runHook executes a single hook entry
func runHook(entry config.HookEntry, env *Env, workDir string) error {
	scriptPath := entry.Script

	// Resolve relative paths from repo root
	if !filepath.IsAbs(scriptPath) {
		scriptPath = filepath.Join(env.RepoRoot, scriptPath)
	}

	// Check if script exists
	if _, err := os.Stat(scriptPath); err != nil {
		return fmt.Errorf("hook script not found: %s", scriptPath)
	}

	// Build the command
	cmd := exec.Command("/bin/bash", scriptPath)
	cmd.Dir = workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set environment variables
	cmd.Env = append(os.Environ(), env.ToEnvVars()...)

	// Add custom environment variables from hook config
	for k, v := range entry.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	return cmd.Run()
}

// RunPreCreate runs pre-create hooks
func RunPreCreate(cfg *config.Config, env *Env) error {
	if len(cfg.Hooks.PreCreate) == 0 {
		return nil
	}
	fmt.Println("Running pre-create hooks...")
	return Run(cfg.Hooks.PreCreate, env, env.RepoRoot)
}

// RunPostCreate runs post-create hooks
func RunPostCreate(cfg *config.Config, env *Env) error {
	if len(cfg.Hooks.PostCreate) == 0 {
		return nil
	}
	fmt.Println("Running post-create hooks...")
	return Run(cfg.Hooks.PostCreate, env, env.Path)
}

// RunPreDelete runs pre-delete hooks
func RunPreDelete(cfg *config.Config, env *Env) error {
	if len(cfg.Hooks.PreDelete) == 0 {
		return nil
	}
	fmt.Println("Running pre-delete hooks...")
	return Run(cfg.Hooks.PreDelete, env, env.Path)
}

// RunPostDelete runs post-delete hooks
func RunPostDelete(cfg *config.Config, env *Env) error {
	if len(cfg.Hooks.PostDelete) == 0 {
		return nil
	}
	fmt.Println("Running post-delete hooks...")
	return Run(cfg.Hooks.PostDelete, env, env.RepoRoot)
}
