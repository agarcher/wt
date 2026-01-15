package commands

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agarcher/wt/internal/git"
)

// resetFlags resets command flags to their default values between tests
func resetFlags() {
	createBranch = ""
	deleteForce = false
	deleteKeepBranch = false
	cleanupDryRun = false
	cleanupForce = false
	cleanupKeepBranch = false
	configGlobal = false
	configUnset = false
	configList = false
	configShowOrigin = false
}

// setupTestRepo creates a temporary git repository with .wt.yaml for testing
func setupTestRepo(t *testing.T) (string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "wt-cmd-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Resolve symlinks (macOS /var -> /private/var)
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatalf("failed to eval symlinks: %v", err)
	}

	cleanup := func() {
		_ = os.RemoveAll(tmpDir)
	}

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		cleanup()
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git user
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = tmpDir
	_ = cmd.Run()

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	_ = cmd.Run()

	// Create initial commit
	testFile := filepath.Join(tmpDir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test"), 0644); err != nil {
		cleanup()
		t.Fatalf("failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	_ = cmd.Run()

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		cleanup()
		t.Fatalf("failed to git commit: %v", err)
	}

	// Create .wt.yaml
	wtConfig := `version: 1
worktree_dir: worktrees
branch_pattern: "{name}"
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".wt.yaml"), []byte(wtConfig), 0644); err != nil {
		cleanup()
		t.Fatalf("failed to write .wt.yaml: %v", err)
	}

	return tmpDir, cleanup
}

// executeCommand runs a cobra command and returns stdout, stderr, and error
func executeCommand(args ...string) (string, string, error) {
	// Reset flags to default values to avoid state pollution between tests
	resetFlags()

	// Reset help flag on all subcommands (gets set by --help tests)
	for _, cmd := range rootCmd.Commands() {
		_ = cmd.Flags().Set("help", "false")
	}

	// Reset the command for fresh execution
	rootCmd.SetArgs(args)

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)

	err := rootCmd.Execute()
	return stdout.String(), stderr.String(), err
}

func TestVersionCommand(t *testing.T) {
	stdout, _, err := executeCommand("version")
	if err != nil {
		t.Fatalf("version command failed: %v", err)
	}

	if !strings.Contains(stdout, "wt version") {
		t.Errorf("expected version output, got: %s", stdout)
	}
}

func TestInitCommand(t *testing.T) {
	tests := []struct {
		shell   string
		wantErr bool
	}{
		{"zsh", false},
		{"bash", false},
		{"fish", false},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			stdout, _, err := executeCommand("init", tt.shell)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("init command failed: %v", err)
			}
			if stdout == "" {
				t.Error("expected shell script output")
			}
			// Check that it contains a function definition
			if !strings.Contains(stdout, "wt()") && !strings.Contains(stdout, "function wt") {
				t.Error("expected wt function in output")
			}
		})
	}
}

func TestInitCommandInvalidShell(t *testing.T) {
	_, _, err := executeCommand("init", "invalid-shell")
	if err == nil {
		t.Error("expected error for invalid shell")
	}
}

func TestInitShellIntegration(t *testing.T) {
	tests := []struct {
		name       string
		shell      string
		shellPath  string
		checkCmd   string
		wantOutput string
	}{
		{
			name:       "zsh",
			shell:      "zsh",
			shellPath:  "/bin/zsh",
			checkCmd:   "type wt",
			wantOutput: "function",
		},
		{
			name:       "bash",
			shell:      "bash",
			shellPath:  "/bin/bash",
			checkCmd:   "type wt",
			wantOutput: "function",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip if shell is not available
			if _, err := os.Stat(tt.shellPath); os.IsNotExist(err) {
				t.Skipf("%s not available at %s", tt.shell, tt.shellPath)
			}

			// Get the init script
			script, _, err := executeCommand("init", tt.shell)
			if err != nil {
				t.Fatalf("failed to get init script: %v", err)
			}

			// Run the shell with eval'd script and check function is defined
			cmd := exec.Command(tt.shellPath, "-c", script+"\n"+tt.checkCmd)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("shell command failed: %v\noutput: %s", err, output)
			}

			if !strings.Contains(string(output), tt.wantOutput) {
				t.Errorf("expected output to contain %q, got: %s", tt.wantOutput, output)
			}
		})
	}
}

func TestRootCommand(t *testing.T) {
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	// Change to the test repo
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	_ = os.Chdir(repoRoot)

	stdout, _, err := executeCommand("root")
	if err != nil {
		t.Fatalf("root command failed: %v", err)
	}

	if strings.TrimSpace(stdout) != repoRoot {
		t.Errorf("expected %q, got %q", repoRoot, strings.TrimSpace(stdout))
	}
}

func TestListCommandEmpty(t *testing.T) {
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	_ = os.Chdir(repoRoot)

	stdout, _, err := executeCommand("list")
	if err != nil {
		t.Fatalf("list command failed: %v", err)
	}

	if !strings.Contains(stdout, "No worktrees") {
		t.Errorf("expected 'No worktrees' message, got: %s", stdout)
	}
}

func TestCreateAndDeleteWorkflow(t *testing.T) {
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	_ = os.Chdir(repoRoot)

	// Create a worktree
	stdout, _, err := executeCommand("create", "test-feature")
	if err != nil {
		t.Fatalf("create command failed: %v", err)
	}

	// Verify output contains the path
	expectedPath := filepath.Join(repoRoot, "worktrees", "test-feature")
	if !strings.Contains(stdout, expectedPath) {
		t.Errorf("expected path %q in output, got: %s", expectedPath, stdout)
	}

	// Verify worktree was created
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Error("worktree directory was not created")
	}

	// List should show the worktree
	stdout, _, err = executeCommand("list")
	if err != nil {
		t.Fatalf("list command failed: %v", err)
	}
	if !strings.Contains(stdout, "test-feature") {
		t.Error("created worktree not in list")
	}

	// cd should output the path
	stdout, _, err = executeCommand("cd", "test-feature")
	if err != nil {
		t.Fatalf("cd command failed: %v", err)
	}
	if strings.TrimSpace(stdout) != expectedPath {
		t.Errorf("cd output expected %q, got %q", expectedPath, strings.TrimSpace(stdout))
	}

	// Delete the worktree (branch deleted by default)
	_, _, err = executeCommand("delete", "test-feature", "--force")
	if err != nil {
		t.Fatalf("delete command failed: %v", err)
	}

	// Verify worktree is gone
	if _, err := os.Stat(expectedPath); !os.IsNotExist(err) {
		t.Error("worktree still exists after deletion")
	}
}

func TestCreateAllocatesIndex(t *testing.T) {
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	_ = os.Chdir(repoRoot)

	// Create first worktree - should get index 1
	_, _, err := executeCommand("create", "wt-one")
	if err != nil {
		t.Fatalf("create command failed: %v", err)
	}

	index1, err := git.GetWorktreeIndex(repoRoot, "wt-one")
	if err != nil {
		t.Fatalf("failed to get index for wt-one: %v", err)
	}
	if index1 != 1 {
		t.Errorf("expected wt-one to have index 1, got %d", index1)
	}

	// Create second worktree - should get index 2
	_, _, err = executeCommand("create", "wt-two")
	if err != nil {
		t.Fatalf("create command failed: %v", err)
	}

	index2, err := git.GetWorktreeIndex(repoRoot, "wt-two")
	if err != nil {
		t.Fatalf("failed to get index for wt-two: %v", err)
	}
	if index2 != 2 {
		t.Errorf("expected wt-two to have index 2, got %d", index2)
	}

	// Delete first worktree - frees index 1
	_, _, err = executeCommand("delete", "wt-one", "--force")
	if err != nil {
		t.Fatalf("delete command failed: %v", err)
	}

	// Create third worktree - should reuse index 1
	_, _, err = executeCommand("create", "wt-three")
	if err != nil {
		t.Fatalf("create command failed: %v", err)
	}

	index3, err := git.GetWorktreeIndex(repoRoot, "wt-three")
	if err != nil {
		t.Fatalf("failed to get index for wt-three: %v", err)
	}
	if index3 != 1 {
		t.Errorf("expected wt-three to reuse index 1, got %d", index3)
	}

	// Cleanup
	_, _, _ = executeCommand("delete", "wt-two", "--force")
	_, _, _ = executeCommand("delete", "wt-three", "--force")
}

func TestListShowsIndex(t *testing.T) {
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	_ = os.Chdir(repoRoot)

	// Create a worktree
	_, _, err := executeCommand("create", "indexed-wt")
	if err != nil {
		t.Fatalf("create command failed: %v", err)
	}

	// List should show the index
	stdout, _, err := executeCommand("list")
	if err != nil {
		t.Fatalf("list command failed: %v", err)
	}

	// Should show INDEX header and index 1 for the worktree
	if !strings.Contains(stdout, "INDEX") {
		t.Error("list output should contain INDEX header")
	}
	// Check that the index column contains "1" (without # prefix)
	// The output format is: "  name  index  branch  status"
	lines := strings.Split(stdout, "\n")
	foundIndex := false
	for _, line := range lines {
		if strings.Contains(line, "indexed-wt") {
			// Parse the line to find the index value
			fields := strings.Fields(line)
			for _, field := range fields {
				if field == "1" {
					foundIndex = true
					break
				}
			}
			break
		}
	}
	if !foundIndex {
		t.Errorf("list output should show index 1 for the worktree, got: %s", stdout)
	}

	// Cleanup
	_, _, _ = executeCommand("delete", "indexed-wt", "--force")
}

func TestCreateWithExistingBranch(t *testing.T) {
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	_ = os.Chdir(repoRoot)

	// Create a branch first
	cmd := exec.Command("git", "branch", "existing-branch")
	cmd.Dir = repoRoot
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}

	// Create worktree from existing branch
	stdout, _, err := executeCommand("create", "from-existing", "--branch", "existing-branch")
	if err != nil {
		t.Fatalf("create command failed: %v", err)
	}

	expectedPath := filepath.Join(repoRoot, "worktrees", "from-existing")
	if !strings.Contains(stdout, expectedPath) {
		t.Errorf("expected path in output")
	}

	// Cleanup
	_, _, _ = executeCommand("delete", "from-existing", "--force")
}

func TestCreateDuplicateBranchFails(t *testing.T) {
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	_ = os.Chdir(repoRoot)

	// Create first worktree
	_, _, err := executeCommand("create", "feature-x")
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	// Try to create another with same name (should fail because branch exists)
	_, _, err = executeCommand("create", "feature-x")
	if err == nil {
		t.Error("expected error when creating duplicate branch")
	}

	// Cleanup
	_, _, _ = executeCommand("delete", "feature-x", "--force")
}

func TestDeleteNonexistent(t *testing.T) {
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	_ = os.Chdir(repoRoot)

	_, _, err := executeCommand("delete", "nonexistent-worktree")
	if err == nil {
		t.Error("expected error when deleting nonexistent worktree")
	}
}

func TestDeleteFailsWithUncommittedChanges(t *testing.T) {
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	_ = os.Chdir(repoRoot)

	// Create a worktree
	_, _, err := executeCommand("create", "dirty-wt")
	if err != nil {
		t.Fatalf("create command failed: %v", err)
	}

	// Add uncommitted changes (dirty file)
	worktreePath := filepath.Join(repoRoot, "worktrees", "dirty-wt")
	if err := os.WriteFile(filepath.Join(worktreePath, "dirty.txt"), []byte("dirty"), 0644); err != nil {
		t.Fatalf("failed to create dirty file: %v", err)
	}

	// Delete should fail without --force
	_, stderr, err := executeCommand("delete", "dirty-wt")
	if err == nil {
		t.Error("expected error when deleting worktree with uncommitted changes")
	}
	if !strings.Contains(stderr, "uncommitted changes") {
		t.Errorf("expected error message about uncommitted changes, got: %s", stderr)
	}
	if !strings.Contains(stderr, "--force") {
		t.Errorf("expected hint about --force, got: %s", stderr)
	}

	// Verify worktree still exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		t.Error("worktree should still exist after failed delete")
	}

	// Now delete with --force should succeed
	_, _, err = executeCommand("delete", "dirty-wt", "--force")
	if err != nil {
		t.Fatalf("delete --force command failed: %v", err)
	}

	// Verify worktree is gone
	if _, err := os.Stat(worktreePath); !os.IsNotExist(err) {
		t.Error("worktree should be deleted after --force")
	}
}

func TestDeleteFailsWithUnmergedCommits(t *testing.T) {
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	_ = os.Chdir(repoRoot)

	// Create a worktree
	_, _, err := executeCommand("create", "unmerged-wt")
	if err != nil {
		t.Fatalf("create command failed: %v", err)
	}

	// Add a commit in the worktree (not merged to main)
	worktreePath := filepath.Join(repoRoot, "worktrees", "unmerged-wt")
	if err := os.WriteFile(filepath.Join(worktreePath, "feature.txt"), []byte("feature"), 0644); err != nil {
		t.Fatalf("failed to create feature file: %v", err)
	}

	cmd := exec.Command("git", "add", "feature.txt")
	cmd.Dir = worktreePath
	if err := cmd.Run(); err != nil {
		t.Fatalf("git add failed: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Add feature")
	cmd.Dir = worktreePath
	if err := cmd.Run(); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	// Delete should fail without --force
	_, stderr, err := executeCommand("delete", "unmerged-wt")
	if err == nil {
		t.Error("expected error when deleting worktree with unmerged commits")
	}
	if !strings.Contains(stderr, "not merged") {
		t.Errorf("expected error message about unmerged commits, got: %s", stderr)
	}
	if !strings.Contains(stderr, "--force") {
		t.Errorf("expected hint about --force, got: %s", stderr)
	}

	// Verify worktree still exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		t.Error("worktree should still exist after failed delete")
	}

	// Now delete with --force should succeed
	_, _, err = executeCommand("delete", "unmerged-wt", "--force")
	if err != nil {
		t.Fatalf("delete --force command failed: %v", err)
	}

	// Verify worktree is gone
	if _, err := os.Stat(worktreePath); !os.IsNotExist(err) {
		t.Error("worktree should be deleted after --force")
	}
}

func TestDeleteSucceedsWithCleanWorktree(t *testing.T) {
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	_ = os.Chdir(repoRoot)

	// Create a worktree
	_, _, err := executeCommand("create", "clean-wt")
	if err != nil {
		t.Fatalf("create command failed: %v", err)
	}

	worktreePath := filepath.Join(repoRoot, "worktrees", "clean-wt")

	// Delete should succeed without --force (clean worktree, no new commits)
	_, _, err = executeCommand("delete", "clean-wt")
	if err != nil {
		t.Fatalf("delete command failed for clean worktree: %v", err)
	}

	// Verify worktree is gone
	if _, err := os.Stat(worktreePath); !os.IsNotExist(err) {
		t.Error("worktree should be deleted")
	}
}

func TestCdNonexistent(t *testing.T) {
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	_ = os.Chdir(repoRoot)

	_, _, err := executeCommand("cd", "nonexistent-worktree")
	if err == nil {
		t.Error("expected error when cd to nonexistent worktree")
	}
}

func TestExitCommand(t *testing.T) {
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	_ = os.Chdir(repoRoot)

	stdout, stderr, err := executeCommand("exit")
	if err != nil {
		t.Fatalf("exit command failed: %v", err)
	}

	if strings.TrimSpace(stdout) != repoRoot {
		t.Errorf("expected %q, got %q", repoRoot, strings.TrimSpace(stdout))
	}

	// Verify output goes to stdout, not stderr (critical for shell wrapper)
	if stderr != "" {
		t.Errorf("exit command should not write to stderr, got: %q", stderr)
	}
}

func TestExitFromWorktree(t *testing.T) {
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	_ = os.Chdir(repoRoot)

	// Create a worktree
	_, _, err := executeCommand("create", "test-exit-wt")
	if err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	// Change to worktree directory
	worktreePath := filepath.Join(repoRoot, "worktrees", "test-exit-wt")
	_ = os.Chdir(worktreePath)

	// Run exit from within worktree
	stdout, stderr, err := executeCommand("exit")
	if err != nil {
		t.Fatalf("exit command failed from worktree: %v", err)
	}

	// Should return main repo root
	if strings.TrimSpace(stdout) != repoRoot {
		t.Errorf("expected main repo %q, got %q", repoRoot, strings.TrimSpace(stdout))
	}

	// Verify output goes to stdout, not stderr (critical for shell wrapper)
	if stderr != "" {
		t.Errorf("exit command should not write to stderr, got: %q", stderr)
	}

	// Cleanup
	_ = os.Chdir(repoRoot)
	_, _, _ = executeCommand("delete", "test-exit-wt", "--force")
}

func TestPathOutputGoesToStdout(t *testing.T) {
	// This test verifies that commands outputting paths for shell wrappers
	// write to stdout (not stderr). Cobra's cmd.Println() writes to stderr
	// by default, which breaks shell wrappers.
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	_ = os.Chdir(repoRoot)

	tests := []struct {
		name string
		args []string
	}{
		{"exit", []string{"exit"}},
		{"root", []string{"root"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := executeCommand(tt.args...)
			if err != nil {
				t.Fatalf("%s command failed: %v", tt.name, err)
			}

			// Path must be in stdout for shell wrapper to capture
			if strings.TrimSpace(stdout) == "" {
				t.Errorf("%s: expected path in stdout, got empty", tt.name)
			}

			// Stderr should be empty (path should not go there)
			if stderr != "" {
				t.Errorf("%s: should not write to stderr, got: %q", tt.name, stderr)
			}
		})
	}
}

func TestHelpCommand(t *testing.T) {
	stdout, _, err := executeCommand("help")
	if err != nil {
		t.Fatalf("help command failed: %v", err)
	}

	// Should contain usage info
	if !strings.Contains(stdout, "Usage:") {
		t.Error("expected Usage in help output")
	}
	if !strings.Contains(stdout, "Available Commands:") {
		t.Error("expected Available Commands in help output")
	}
}

func TestSubcommandHelp(t *testing.T) {
	commands := []string{"create", "delete", "list", "cd", "exit", "init", "cleanup"}

	for _, cmd := range commands {
		t.Run(cmd, func(t *testing.T) {
			stdout, _, err := executeCommand(cmd, "--help")
			if err != nil {
				t.Errorf("%s --help failed: %v", cmd, err)
			}
			if stdout == "" {
				t.Errorf("%s --help produced no output", cmd)
			}
		})
	}
}

func TestCleanupNoWorktrees(t *testing.T) {
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	_ = os.Chdir(repoRoot)

	stdout, stderr, err := executeCommand("cleanup")
	if err != nil {
		t.Fatalf("cleanup command failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	if !strings.Contains(stdout, "No worktrees eligible for cleanup") {
		t.Errorf("expected no eligible worktrees message, got: %s", stdout)
	}
}

func TestCleanupDryRun(t *testing.T) {
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	_ = os.Chdir(repoRoot)

	// Create a worktree
	var cmd *exec.Cmd
	_, _, err := executeCommand("create", "feature-to-merge")
	if err != nil {
		t.Fatalf("create command failed: %v", err)
	}

	// Make a commit in the worktree so it's not considered "new"
	worktreePath := filepath.Join(repoRoot, "worktrees", "feature-to-merge")
	testFile := filepath.Join(worktreePath, "feature.txt")
	if err := os.WriteFile(testFile, []byte("feature content"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = worktreePath
	_ = cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Add feature")
	cmd.Dir = worktreePath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to commit in worktree: %v", err)
	}

	// Switch back to main repo and merge the branch (making it eligible for cleanup)
	cmd = exec.Command("git", "merge", "feature-to-merge")
	cmd.Dir = repoRoot
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to merge branch: %v", err)
	}

	// Run cleanup with --dry-run
	stdout, _, err := executeCommand("cleanup", "--dry-run")
	if err != nil {
		t.Fatalf("cleanup --dry-run failed: %v", err)
	}

	if !strings.Contains(stdout, "feature-to-merge") {
		t.Errorf("expected feature-to-merge in cleanup candidates, got: %s", stdout)
	}
	if !strings.Contains(stdout, "[merged]") {
		t.Errorf("expected '[merged]' status, got: %s", stdout)
	}
	if !strings.Contains(stdout, "Would delete") {
		t.Errorf("expected 'Would delete' message in dry run, got: %s", stdout)
	}

	// Verify worktree still exists (dry run shouldn't delete)
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		t.Error("worktree was deleted during dry run")
	}

	// Cleanup
	_, _, _ = executeCommand("delete", "feature-to-merge", "--force")
}

func TestCleanupMergedWorktree(t *testing.T) {
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	_ = os.Chdir(repoRoot)

	// Create a worktree
	_, _, err := executeCommand("create", "merged-feature")
	if err != nil {
		t.Fatalf("create command failed: %v", err)
	}

	// Make a commit in the worktree so it's not considered "new"
	worktreePath := filepath.Join(repoRoot, "worktrees", "merged-feature")
	testFile := filepath.Join(worktreePath, "feature.txt")
	if err := os.WriteFile(testFile, []byte("feature content"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = worktreePath
	_ = cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Add feature")
	cmd.Dir = worktreePath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to commit in worktree: %v", err)
	}

	// Switch back to main repo and merge the branch
	cmd = exec.Command("git", "merge", "merged-feature")
	cmd.Dir = repoRoot
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to merge branch: %v", err)
	}

	// Run cleanup with --force (skip confirmation)
	stdout, _, err := executeCommand("cleanup", "--force")
	if err != nil {
		t.Fatalf("cleanup --force failed: %v", err)
	}

	if !strings.Contains(stdout, "Cleaned up 1 worktree") {
		t.Errorf("expected cleanup success message, got: %s", stdout)
	}

	// Verify worktree is deleted
	if _, err := os.Stat(worktreePath); !os.IsNotExist(err) {
		t.Error("worktree still exists after cleanup")
	}
}

func TestCleanupUnmergedWorktree(t *testing.T) {
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	_ = os.Chdir(repoRoot)

	// Create a worktree
	_, _, err := executeCommand("create", "unmerged-feature")
	if err != nil {
		t.Fatalf("create command failed: %v", err)
	}

	// Make a commit in the worktree so it's NOT merged into main
	worktreePath := filepath.Join(repoRoot, "worktrees", "unmerged-feature")
	testFile := filepath.Join(worktreePath, "new-file.txt")
	if err := os.WriteFile(testFile, []byte("new content"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = worktreePath
	_ = cmd.Run()

	cmd = exec.Command("git", "commit", "-m", "Add new file")
	cmd.Dir = worktreePath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	// Run cleanup
	stdout, _, err := executeCommand("cleanup")
	if err != nil {
		t.Fatalf("cleanup command failed: %v", err)
	}

	// Should not find the unmerged worktree as eligible
	if strings.Contains(stdout, "unmerged-feature") {
		t.Errorf("unmerged worktree should not be in cleanup candidates, got: %s", stdout)
	}
	if !strings.Contains(stdout, "No worktrees eligible for cleanup") {
		t.Errorf("expected no eligible message, got: %s", stdout)
	}

	// Cleanup
	_, _, _ = executeCommand("delete", "unmerged-feature", "--force")
}

func TestCleanupSkipsNewWorktree(t *testing.T) {
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	_ = os.Chdir(repoRoot)

	// Create a worktree but don't make any commits
	// This worktree is "new" - still on its initial commit
	_, _, err := executeCommand("create", "new-feature")
	if err != nil {
		t.Fatalf("create command failed: %v", err)
	}

	// Run cleanup - should skip the new worktree even though it's technically "merged"
	// (same commit as main)
	stdout, _, err := executeCommand("cleanup")
	if err != nil {
		t.Fatalf("cleanup command failed: %v", err)
	}

	if strings.Contains(stdout, "new-feature") {
		t.Errorf("new worktree should not be in cleanup candidates, got: %s", stdout)
	}
	if !strings.Contains(stdout, "No worktrees eligible for cleanup") {
		t.Errorf("expected no eligible message, got: %s", stdout)
	}

	// Cleanup
	_, _, _ = executeCommand("delete", "new-feature", "--force")
}

func TestCleanupSkipsUncommittedChanges(t *testing.T) {
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	_ = os.Chdir(repoRoot)

	// Create a worktree and make a commit
	_, _, err := executeCommand("create", "dirty-feature")
	if err != nil {
		t.Fatalf("create command failed: %v", err)
	}

	worktreePath := filepath.Join(repoRoot, "worktrees", "dirty-feature")

	// Make a commit so it's not "new"
	testFile := filepath.Join(worktreePath, "feature.txt")
	if err := os.WriteFile(testFile, []byte("feature"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = worktreePath
	_ = cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Add feature")
	cmd.Dir = worktreePath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	// Merge into main so it would be eligible for cleanup
	cmd = exec.Command("git", "merge", "dirty-feature")
	cmd.Dir = repoRoot
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to merge: %v", err)
	}

	// Now add uncommitted changes to the worktree
	dirtyFile := filepath.Join(worktreePath, "uncommitted.txt")
	if err := os.WriteFile(dirtyFile, []byte("uncommitted work"), 0644); err != nil {
		t.Fatalf("failed to write dirty file: %v", err)
	}

	// Run cleanup - should skip due to uncommitted changes
	stdout, _, err := executeCommand("cleanup")
	if err != nil {
		t.Fatalf("cleanup command failed: %v", err)
	}

	if strings.Contains(stdout, "dirty-feature") {
		t.Errorf("worktree with uncommitted changes should not be in cleanup candidates, got: %s", stdout)
	}
	if !strings.Contains(stdout, "No worktrees eligible for cleanup") {
		t.Errorf("expected no eligible message, got: %s", stdout)
	}

	// Cleanup
	_, _, _ = executeCommand("delete", "dirty-feature", "--force")
}

// setupTestRepoWithIsolatedHome creates a test repo with isolated HOME directory
func setupTestRepoWithIsolatedHome(t *testing.T) (repoRoot string, homeDir string, cleanup func()) {
	t.Helper()

	repoRoot, repoCleanup := setupTestRepo(t)

	// Create isolated HOME directory for user config
	homeDir, err := os.MkdirTemp("", "wt-home-test-*")
	if err != nil {
		repoCleanup()
		t.Fatalf("failed to create temp home dir: %v", err)
	}

	// Resolve symlinks
	homeDir, err = filepath.EvalSymlinks(homeDir)
	if err != nil {
		_ = os.RemoveAll(homeDir)
		repoCleanup()
		t.Fatalf("failed to eval symlinks: %v", err)
	}

	// Set HOME to isolated directory
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", homeDir)

	cleanup = func() {
		_ = os.Setenv("HOME", oldHome)
		_ = os.RemoveAll(homeDir)
		repoCleanup()
	}

	return repoRoot, homeDir, cleanup
}

func TestConfigHelp(t *testing.T) {
	stdout, _, err := executeCommand("config", "--help")
	if err != nil {
		t.Fatalf("config --help failed: %v", err)
	}

	expectedStrings := []string{
		"config",
		"--global",
		"--list",
		"--show-origin",
		"--unset",
		"remote",
		"fetch",
	}
	for _, s := range expectedStrings {
		if !strings.Contains(stdout, s) {
			t.Errorf("expected help to contain %q, got: %s", s, stdout)
		}
	}
}

func TestConfigSetAndGetGlobal(t *testing.T) {
	repoRoot, homeDir, cleanup := setupTestRepoWithIsolatedHome(t)
	defer cleanup()

	// Change to test repo
	oldWd, _ := os.Getwd()
	_ = os.Chdir(repoRoot)
	defer func() { _ = os.Chdir(oldWd) }()

	// Set global remote
	_, _, err := executeCommand("config", "--global", "remote", "origin")
	if err != nil {
		t.Fatalf("config set failed: %v", err)
	}

	// Verify config file was created
	configPath := filepath.Join(homeDir, ".config", "wt", "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("config file not created at %s", configPath)
	}

	// Get global remote
	stdout, _, err := executeCommand("config", "--global", "remote")
	if err != nil {
		t.Fatalf("config get failed: %v", err)
	}
	if !strings.Contains(stdout, "origin") {
		t.Errorf("expected 'origin', got: %s", stdout)
	}

	// Set global fetch_interval
	_, _, err = executeCommand("config", "--global", "fetch_interval", "10m")
	if err != nil {
		t.Fatalf("config set fetch_interval failed: %v", err)
	}

	// Get global fetch_interval
	stdout, _, err = executeCommand("config", "--global", "fetch_interval")
	if err != nil {
		t.Fatalf("config get fetch_interval failed: %v", err)
	}
	if !strings.Contains(stdout, "10m") {
		t.Errorf("expected '10m', got: %s", stdout)
	}
}

func TestConfigSetPerRepo(t *testing.T) {
	repoRoot, _, cleanup := setupTestRepoWithIsolatedHome(t)
	defer cleanup()

	// Change to test repo
	oldWd, _ := os.Getwd()
	_ = os.Chdir(repoRoot)
	defer func() { _ = os.Chdir(oldWd) }()

	// Set per-repo remote
	_, _, err := executeCommand("config", "remote", "upstream")
	if err != nil {
		t.Fatalf("config set failed: %v", err)
	}

	// Get per-repo remote (should show effective value)
	stdout, _, err := executeCommand("config", "remote")
	if err != nil {
		t.Fatalf("config get failed: %v", err)
	}
	if !strings.Contains(stdout, "upstream") {
		t.Errorf("expected 'upstream', got: %s", stdout)
	}
}

func TestConfigList(t *testing.T) {
	repoRoot, _, cleanup := setupTestRepoWithIsolatedHome(t)
	defer cleanup()

	// Change to test repo
	oldWd, _ := os.Getwd()
	_ = os.Chdir(repoRoot)
	defer func() { _ = os.Chdir(oldWd) }()

	// Set some values
	_, _, _ = executeCommand("config", "--global", "remote", "origin")
	_, _, _ = executeCommand("config", "--global", "fetch_interval", "10m")
	_, _, _ = executeCommand("config", "remote", "upstream")

	// List all config
	stdout, _, err := executeCommand("config", "--list")
	if err != nil {
		t.Fatalf("config --list failed: %v", err)
	}

	if !strings.Contains(stdout, "remote = origin") {
		t.Errorf("expected global remote in list, got: %s", stdout)
	}
	if !strings.Contains(stdout, "fetch_interval = 10m") {
		t.Errorf("expected global fetch_interval in list, got: %s", stdout)
	}
	if !strings.Contains(stdout, "remote = upstream") {
		t.Errorf("expected per-repo remote in list, got: %s", stdout)
	}
}

func TestConfigShowOrigin(t *testing.T) {
	repoRoot, _, cleanup := setupTestRepoWithIsolatedHome(t)
	defer cleanup()

	// Change to test repo
	oldWd, _ := os.Getwd()
	_ = os.Chdir(repoRoot)
	defer func() { _ = os.Chdir(oldWd) }()

	// Set a global remote
	_, _, _ = executeCommand("config", "--global", "remote", "origin")

	// Show origin
	stdout, _, err := executeCommand("config", "--show-origin")
	if err != nil {
		t.Fatalf("config --show-origin failed: %v", err)
	}

	if !strings.Contains(stdout, "remote") {
		t.Errorf("expected 'remote' in show-origin output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "fetch") {
		t.Errorf("expected 'fetch' in show-origin output, got: %s", stdout)
	}
}

func TestConfigUnset(t *testing.T) {
	repoRoot, _, cleanup := setupTestRepoWithIsolatedHome(t)
	defer cleanup()

	// Change to test repo
	oldWd, _ := os.Getwd()
	_ = os.Chdir(repoRoot)
	defer func() { _ = os.Chdir(oldWd) }()

	// Set a per-repo value
	_, _, _ = executeCommand("config", "remote", "upstream")

	// Verify it's set
	stdout, _, _ := executeCommand("config", "remote")
	if !strings.Contains(stdout, "upstream") {
		t.Fatalf("expected 'upstream' before unset, got: %s", stdout)
	}

	// Unset it
	_, _, err := executeCommand("config", "--unset", "remote")
	if err != nil {
		t.Fatalf("config --unset failed: %v", err)
	}

	// Verify it's unset (should be empty or fall back to global)
	stdout, _, _ = executeCommand("config", "remote")
	if strings.Contains(stdout, "upstream") {
		t.Errorf("expected 'upstream' to be removed, got: %s", stdout)
	}
}

func TestConfigInvalidKey(t *testing.T) {
	repoRoot, _, cleanup := setupTestRepoWithIsolatedHome(t)
	defer cleanup()

	// Change to test repo
	oldWd, _ := os.Getwd()
	_ = os.Chdir(repoRoot)
	defer func() { _ = os.Chdir(oldWd) }()

	// Try to set invalid key
	_, _, err := executeCommand("config", "--global", "invalid_key", "value")
	if err == nil {
		t.Error("expected error for invalid key, got none")
	}
}

func TestConfigFetchIntervalValidation(t *testing.T) {
	repoRoot, _, cleanup := setupTestRepoWithIsolatedHome(t)
	defer cleanup()

	// Change to test repo
	oldWd, _ := os.Getwd()
	_ = os.Chdir(repoRoot)
	defer func() { _ = os.Chdir(oldWd) }()

	// Valid duration should work
	_, _, err := executeCommand("config", "--global", "fetch_interval", "10m")
	if err != nil {
		t.Fatalf("config set valid duration failed: %v", err)
	}

	// "never" should work
	_, _, err = executeCommand("config", "--global", "fetch_interval", "never")
	if err != nil {
		t.Fatalf("config set 'never' failed: %v", err)
	}

	// "0" should work (always fetch)
	_, _, err = executeCommand("config", "--global", "fetch_interval", "0")
	if err != nil {
		t.Fatalf("config set '0' failed: %v", err)
	}

	// Invalid duration should fail
	_, _, err = executeCommand("config", "--global", "fetch_interval", "invalid")
	if err == nil {
		t.Error("expected error for invalid duration, got none")
	}
}

func TestConfigGlobalUnset(t *testing.T) {
	repoRoot, _, cleanup := setupTestRepoWithIsolatedHome(t)
	defer cleanup()

	// Change to test repo
	oldWd, _ := os.Getwd()
	_ = os.Chdir(repoRoot)
	defer func() { _ = os.Chdir(oldWd) }()

	// Set a global value
	_, _, err := executeCommand("config", "--global", "remote", "origin")
	if err != nil {
		t.Fatalf("config set failed: %v", err)
	}

	// Verify it's set
	stdout, _, _ := executeCommand("config", "--global", "remote")
	if !strings.Contains(stdout, "origin") {
		t.Fatalf("expected 'origin' before unset, got: %s", stdout)
	}

	// Unset it globally
	_, _, err = executeCommand("config", "--global", "--unset", "remote")
	if err != nil {
		t.Fatalf("config --global --unset failed: %v", err)
	}

	// Verify it's unset (should be empty)
	stdout, _, _ = executeCommand("config", "--global", "remote")
	stdout = strings.TrimSpace(stdout)
	if stdout != "" {
		t.Errorf("expected empty remote after unset, got: %q", stdout)
	}
}

func TestListShowsRepoAndComparisonRef(t *testing.T) {
	repoRoot, _, cleanup := setupTestRepoWithIsolatedHome(t)
	defer cleanup()

	// Change to test repo
	oldWd, _ := os.Getwd()
	_ = os.Chdir(repoRoot)
	defer func() { _ = os.Chdir(oldWd) }()

	// Run list
	stdout, _, err := executeCommand("list")
	if err != nil {
		t.Fatalf("list command failed: %v", err)
	}

	// Should show repository path
	if !strings.Contains(stdout, "Repository:") {
		t.Errorf("expected 'Repository:' in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, repoRoot) {
		t.Errorf("expected repo path %s in output, got: %s", repoRoot, stdout)
	}

	// Should show comparison ref
	if !strings.Contains(stdout, "Comparing to:") {
		t.Errorf("expected 'Comparing to:' in output, got: %s", stdout)
	}
}
