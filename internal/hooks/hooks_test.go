package hooks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/agarcher/wt/internal/config"
)

func TestEnvToEnvVars(t *testing.T) {
	env := &Env{
		Name:        "test-wt",
		Path:        "/repo/worktrees/test-wt",
		Branch:      "test-branch",
		RepoRoot:    "/repo",
		WorktreeDir: "worktrees",
	}

	vars := env.ToEnvVars()

	expected := map[string]string{
		"WT_NAME":         "test-wt",
		"WT_PATH":         "/repo/worktrees/test-wt",
		"WT_BRANCH":       "test-branch",
		"WT_REPO_ROOT":    "/repo",
		"WT_WORKTREE_DIR": "worktrees",
	}

	if len(vars) != len(expected) {
		t.Errorf("expected %d vars, got %d", len(expected), len(vars))
	}

	for _, v := range vars {
		found := false
		for key, val := range expected {
			if v == key+"="+val {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("unexpected env var: %s", v)
		}
	}
}

func TestRunHook(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "wt-hooks-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a test script that writes env vars to a file
	scriptPath := filepath.Join(tmpDir, "test-hook.sh")
	outputPath := filepath.Join(tmpDir, "output.txt")
	scriptContent := `#!/bin/bash
echo "WT_NAME=$WT_NAME" > "` + outputPath + `"
echo "WT_PATH=$WT_PATH" >> "` + outputPath + `"
echo "CUSTOM_VAR=$CUSTOM_VAR" >> "` + outputPath + `"
`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("failed to write test script: %v", err)
	}

	// Create hook entry
	entry := config.HookEntry{
		Script: scriptPath,
		Env: map[string]string{
			"CUSTOM_VAR": "custom-value",
		},
	}

	// Create environment
	env := &Env{
		Name:        "test-wt",
		Path:        "/repo/worktrees/test-wt",
		Branch:      "test-branch",
		RepoRoot:    tmpDir,
		WorktreeDir: "worktrees",
	}

	// Run the hook
	err = Run([]config.HookEntry{entry}, env, tmpDir)
	if err != nil {
		t.Fatalf("hook execution failed: %v", err)
	}

	// Read and verify output
	output, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	outputStr := string(output)
	if !contains(outputStr, "WT_NAME=test-wt") {
		t.Error("WT_NAME not set correctly")
	}
	if !contains(outputStr, "WT_PATH=/repo/worktrees/test-wt") {
		t.Error("WT_PATH not set correctly")
	}
	if !contains(outputStr, "CUSTOM_VAR=custom-value") {
		t.Error("CUSTOM_VAR not set correctly")
	}
}

func TestRunHookFailure(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "wt-hooks-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a script that exits with error
	scriptPath := filepath.Join(tmpDir, "fail-hook.sh")
	scriptContent := `#!/bin/bash
exit 1
`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("failed to write test script: %v", err)
	}

	entry := config.HookEntry{
		Script: scriptPath,
	}

	env := &Env{
		Name:        "test-wt",
		Path:        tmpDir,
		Branch:      "test-branch",
		RepoRoot:    tmpDir,
		WorktreeDir: "worktrees",
	}

	// Run should fail
	err = Run([]config.HookEntry{entry}, env, tmpDir)
	if err == nil {
		t.Error("expected error from failing hook, got nil")
	}
}

func TestRunHookNotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wt-hooks-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	entry := config.HookEntry{
		Script: "/nonexistent/script.sh",
	}

	env := &Env{
		Name:        "test-wt",
		Path:        tmpDir,
		Branch:      "test-branch",
		RepoRoot:    tmpDir,
		WorktreeDir: "worktrees",
	}

	err = Run([]config.HookEntry{entry}, env, tmpDir)
	if err == nil {
		t.Error("expected error for missing script, got nil")
	}
}

func TestRunEmptyHooks(t *testing.T) {
	env := &Env{
		Name:        "test-wt",
		Path:        "/repo/worktrees/test-wt",
		Branch:      "test-branch",
		RepoRoot:    "/repo",
		WorktreeDir: "worktrees",
	}

	// Running empty hooks should succeed
	err := Run([]config.HookEntry{}, env, "/tmp")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = Run(nil, env, "/tmp")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunPreCreateHooks(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wt-hooks-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a marker file script
	markerPath := filepath.Join(tmpDir, "marker.txt")
	scriptPath := filepath.Join(tmpDir, "pre-create.sh")
	scriptContent := `#!/bin/bash
echo "pre-create ran" > "` + markerPath + `"
`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("failed to write test script: %v", err)
	}

	cfg := &config.Config{
		Hooks: config.HooksConfig{
			PreCreate: []config.HookEntry{
				{Script: scriptPath},
			},
		},
	}

	env := &Env{
		Name:        "test-wt",
		Path:        tmpDir,
		Branch:      "test-branch",
		RepoRoot:    tmpDir,
		WorktreeDir: "worktrees",
	}

	err = RunPreCreate(cfg, env)
	if err != nil {
		t.Fatalf("RunPreCreate failed: %v", err)
	}

	// Verify marker file was created
	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		t.Error("pre-create hook did not run")
	}
}

func TestRunPostCreateHooks(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wt-hooks-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	markerPath := filepath.Join(tmpDir, "marker.txt")
	scriptPath := filepath.Join(tmpDir, "post-create.sh")
	scriptContent := `#!/bin/bash
echo "post-create ran in $(pwd)" > "` + markerPath + `"
`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("failed to write test script: %v", err)
	}

	cfg := &config.Config{
		Hooks: config.HooksConfig{
			PostCreate: []config.HookEntry{
				{Script: scriptPath},
			},
		},
	}

	env := &Env{
		Name:        "test-wt",
		Path:        tmpDir,
		Branch:      "test-branch",
		RepoRoot:    tmpDir,
		WorktreeDir: "worktrees",
	}

	err = RunPostCreate(cfg, env)
	if err != nil {
		t.Fatalf("RunPostCreate failed: %v", err)
	}

	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		t.Error("post-create hook did not run")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
