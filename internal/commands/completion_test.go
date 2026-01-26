package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestCompletionCommand(t *testing.T) {
	tests := []struct {
		shell       string
		wantErr     bool
		wantContain string
	}{
		{"bash", false, "bash completion"},
		{"zsh", false, "#compdef wt"},
		{"fish", false, "complete -c wt"},
		{"powershell", false, "Register-ArgumentCompleter"},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			stdout, _, err := executeCommand("completion", tt.shell)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("completion %s failed: %v", tt.shell, err)
				return
			}
			if stdout == "" {
				t.Errorf("completion %s produced no output", tt.shell)
			}
			if !strings.Contains(stdout, tt.wantContain) {
				t.Errorf("completion %s output should contain %q", tt.shell, tt.wantContain)
			}
		})
	}
}

func TestCompletionCommandInvalidShell(t *testing.T) {
	_, _, err := executeCommand("completion", "invalid-shell")
	if err == nil {
		t.Error("expected error for invalid shell")
	}
}

func TestCompletionCommandNoArgs(t *testing.T) {
	_, _, err := executeCommand("completion")
	if err == nil {
		t.Error("expected error when no shell specified")
	}
}

func TestCompleteWorktreeNames(t *testing.T) {
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	_ = os.Chdir(repoRoot)

	// Create some worktrees
	_, _, err := executeCommand("create", "feature-one")
	if err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}
	_, _, err = executeCommand("create", "feature-two")
	if err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	// Test the completion function directly
	cmd := &cobra.Command{}
	completions, directive := completeWorktreeNames(cmd, []string{}, "")

	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("expected ShellCompDirectiveNoFileComp, got %v", directive)
	}

	// Should have both worktrees
	if len(completions) != 2 {
		t.Errorf("expected 2 completions, got %d: %v", len(completions), completions)
	}

	found := make(map[string]bool)
	for _, c := range completions {
		found[c] = true
	}
	if !found["feature-one"] {
		t.Error("expected feature-one in completions")
	}
	if !found["feature-two"] {
		t.Error("expected feature-two in completions")
	}

	// Cleanup
	_, _, _ = executeCommand("delete", "feature-one", "--force")
	_, _, _ = executeCommand("delete", "feature-two", "--force")
}

func TestCompleteWorktreeNamesWithPrefix(t *testing.T) {
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	_ = os.Chdir(repoRoot)

	// Create worktrees with different prefixes
	_, _, _ = executeCommand("create", "feat-one")
	_, _, _ = executeCommand("create", "feat-two")
	_, _, _ = executeCommand("create", "bug-fix")

	cmd := &cobra.Command{}

	// Test with prefix "feat"
	completions, _ := completeWorktreeNames(cmd, []string{}, "feat")
	if len(completions) != 2 {
		t.Errorf("expected 2 completions for 'feat' prefix, got %d: %v", len(completions), completions)
	}

	// Test with prefix "bug"
	completions, _ = completeWorktreeNames(cmd, []string{}, "bug")
	if len(completions) != 1 {
		t.Errorf("expected 1 completion for 'bug' prefix, got %d: %v", len(completions), completions)
	}
	if len(completions) == 1 && completions[0] != "bug-fix" {
		t.Errorf("expected 'bug-fix', got %s", completions[0])
	}

	// Cleanup
	_, _, _ = executeCommand("delete", "feat-one", "--force")
	_, _, _ = executeCommand("delete", "feat-two", "--force")
	_, _, _ = executeCommand("delete", "bug-fix", "--force")
}

func TestCompleteWorktreeNamesNoSecondArg(t *testing.T) {
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	_ = os.Chdir(repoRoot)

	_, _, _ = executeCommand("create", "test-wt")

	cmd := &cobra.Command{}

	// When there's already an argument, should return no completions
	completions, directive := completeWorktreeNames(cmd, []string{"already-provided"}, "")
	if len(completions) != 0 {
		t.Errorf("expected no completions when arg already provided, got %v", completions)
	}
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("expected ShellCompDirectiveNoFileComp, got %v", directive)
	}

	// Cleanup
	_, _, _ = executeCommand("delete", "test-wt", "--force")
}

func TestCompleteBranchNames(t *testing.T) {
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	_ = os.Chdir(repoRoot)

	// Create some branches
	cmd := exec.Command("git", "branch", "feature-branch")
	cmd.Dir = repoRoot
	_ = cmd.Run()

	cmd = exec.Command("git", "branch", "bugfix-branch")
	cmd.Dir = repoRoot
	_ = cmd.Run()

	// Test the completion function directly
	cobraCmd := &cobra.Command{}
	completions, directive := completeBranchNames(cobraCmd, []string{}, "")

	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("expected ShellCompDirectiveNoFileComp, got %v", directive)
	}

	// Should have at least the branches we created (plus main/master)
	if len(completions) < 3 {
		t.Errorf("expected at least 3 branches, got %d: %v", len(completions), completions)
	}

	found := make(map[string]bool)
	for _, c := range completions {
		found[c] = true
	}
	if !found["feature-branch"] {
		t.Error("expected feature-branch in completions")
	}
	if !found["bugfix-branch"] {
		t.Error("expected bugfix-branch in completions")
	}
}

func TestCompleteBranchNamesWithPrefix(t *testing.T) {
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	_ = os.Chdir(repoRoot)

	// Create branches with different prefixes
	cmd := exec.Command("git", "branch", "feat-one")
	cmd.Dir = repoRoot
	_ = cmd.Run()

	cmd = exec.Command("git", "branch", "feat-two")
	cmd.Dir = repoRoot
	_ = cmd.Run()

	cmd = exec.Command("git", "branch", "bug-fix")
	cmd.Dir = repoRoot
	_ = cmd.Run()

	cobraCmd := &cobra.Command{}

	// Test with prefix "feat"
	completions, _ := completeBranchNames(cobraCmd, []string{}, "feat")
	if len(completions) != 2 {
		t.Errorf("expected 2 completions for 'feat' prefix, got %d: %v", len(completions), completions)
	}

	// Test with prefix "bug"
	completions, _ = completeBranchNames(cobraCmd, []string{}, "bug")
	if len(completions) != 1 {
		t.Errorf("expected 1 completion for 'bug' prefix, got %d: %v", len(completions), completions)
	}
}

func TestInitCommandValidArgs(t *testing.T) {
	// Test that init command has ValidArgs set correctly
	if initCmd.ValidArgs == nil {
		t.Fatal("init command should have ValidArgs set")
	}

	expected := []string{"zsh", "bash", "fish"}
	if len(initCmd.ValidArgs) != len(expected) {
		t.Errorf("expected %d ValidArgs, got %d", len(expected), len(initCmd.ValidArgs))
	}

	for _, shell := range expected {
		found := false
		for _, arg := range initCmd.ValidArgs {
			if arg == shell {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected %s in ValidArgs", shell)
		}
	}
}

func TestCompletionCommandValidArgs(t *testing.T) {
	// Test that completion command has ValidArgs set correctly
	if completionCmd.ValidArgs == nil {
		t.Fatal("completion command should have ValidArgs set")
	}

	expected := []string{"bash", "zsh", "fish", "powershell"}
	if len(completionCmd.ValidArgs) != len(expected) {
		t.Errorf("expected %d ValidArgs, got %d", len(expected), len(completionCmd.ValidArgs))
	}

	for _, shell := range expected {
		found := false
		for _, arg := range completionCmd.ValidArgs {
			if arg == shell {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected %s in ValidArgs", shell)
		}
	}
}

func TestCdCommandHasValidArgsFunction(t *testing.T) {
	if cdCmd.ValidArgsFunction == nil {
		t.Error("cd command should have ValidArgsFunction set")
	}
}

func TestDeleteCommandHasValidArgsFunction(t *testing.T) {
	if deleteCmd.ValidArgsFunction == nil {
		t.Error("delete command should have ValidArgsFunction set")
	}
}

func TestCompletionScriptSyntax(t *testing.T) {
	// Test that generated completion scripts have valid syntax
	tests := []struct {
		shell     string
		shellPath string
		checkCmd  string
	}{
		{"bash", "/bin/bash", "-n"}, // bash -n checks syntax without executing
		{"zsh", "/bin/zsh", "-n"},   // zsh -n checks syntax without executing
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			// Skip if shell is not available
			if _, err := os.Stat(tt.shellPath); os.IsNotExist(err) {
				t.Skipf("%s not available at %s", tt.shell, tt.shellPath)
			}

			// Generate completion script
			stdout, _, err := executeCommand("completion", tt.shell)
			if err != nil {
				t.Fatalf("failed to generate completion: %v", err)
			}

			// Write to temp file
			tmpFile, err := os.CreateTemp("", "wt-completion-*."+tt.shell)
			if err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}
			defer func() { _ = os.Remove(tmpFile.Name()) }()

			if _, err := tmpFile.WriteString(stdout); err != nil {
				t.Fatalf("failed to write completion script: %v", err)
			}
			_ = tmpFile.Close()

			// Check syntax
			cmd := exec.Command(tt.shellPath, tt.checkCmd, tmpFile.Name())
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Errorf("%s completion script has syntax errors: %v\nOutput: %s", tt.shell, err, output)
			}
		})
	}
}

func TestCompletionIntegration(t *testing.T) {
	// Integration test: create worktrees and verify completion helper functions work
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	_ = os.Chdir(repoRoot)

	// Create worktrees
	_, _, err := executeCommand("create", "alpha-wt")
	if err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}
	_, _, err = executeCommand("create", "beta-wt")
	if err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	// Test worktree name completion directly via the helper function
	cmd := &cobra.Command{}
	completions, directive := completeWorktreeNames(cmd, []string{}, "")

	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("expected ShellCompDirectiveNoFileComp, got %v", directive)
	}

	found := make(map[string]bool)
	for _, c := range completions {
		found[c] = true
	}

	if !found["alpha-wt"] {
		t.Error("expected alpha-wt in completions")
	}
	if !found["beta-wt"] {
		t.Error("expected beta-wt in completions")
	}

	// Verify init command has proper ValidArgs (these are static)
	if len(initCmd.ValidArgs) == 0 {
		t.Error("init command should have ValidArgs set")
	}
	validArgsMap := make(map[string]bool)
	for _, arg := range initCmd.ValidArgs {
		validArgsMap[arg] = true
	}
	if !validArgsMap["zsh"] {
		t.Error("expected zsh in init ValidArgs")
	}
	if !validArgsMap["bash"] {
		t.Error("expected bash in init ValidArgs")
	}
	if !validArgsMap["fish"] {
		t.Error("expected fish in init ValidArgs")
	}

	// Cleanup
	_, _, _ = executeCommand("delete", "alpha-wt", "--force")
	_, _, _ = executeCommand("delete", "beta-wt", "--force")
}

func TestCompletionSubcommands(t *testing.T) {
	// Test that all expected subcommands are registered on the root command
	expectedCommands := []string{
		"cd", "cleanup", "completion", "create", "delete",
		"exit", "info", "init", "list", "root", "version",
	}

	registeredCommands := make(map[string]bool)
	for _, cmd := range rootCmd.Commands() {
		registeredCommands[cmd.Name()] = true
	}

	for _, expected := range expectedCommands {
		if !registeredCommands[expected] {
			t.Errorf("expected %s command to be registered", expected)
		}
	}
}

func TestCompletionScriptsExist(t *testing.T) {
	// Test that pre-generated completion scripts exist
	completionFiles := []struct {
		path string
		name string
	}{
		{"scripts/completions/_wt", "zsh"},
		{"scripts/completions/wt.bash", "bash"},
		{"scripts/completions/wt.fish", "fish"},
	}

	// Get repository root
	repoRoot, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	// Walk up to find repo root (where scripts/ directory should be)
	for {
		if _, err := os.Stat(filepath.Join(repoRoot, "scripts", "completions")); err == nil {
			break
		}
		parent := filepath.Dir(repoRoot)
		if parent == repoRoot {
			t.Skip("scripts/completions directory not found - running outside repository")
		}
		repoRoot = parent
	}

	for _, cf := range completionFiles {
		t.Run(cf.name, func(t *testing.T) {
			fullPath := filepath.Join(repoRoot, cf.path)
			info, err := os.Stat(fullPath)
			if err != nil {
				t.Errorf("completion file %s not found: %v", cf.path, err)
				return
			}
			if info.Size() == 0 {
				t.Errorf("completion file %s is empty", cf.path)
			}
		})
	}
}
