package commands

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// resetFlags resets command flags to their default values between tests
func resetFlags() {
	createBranch = ""
	deleteForce = false
	deleteDeleteBranch = false
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

	// Delete the worktree
	_, _, err = executeCommand("delete", "test-feature", "--force", "--delete-branch")
	if err != nil {
		t.Fatalf("delete command failed: %v", err)
	}

	// Verify worktree is gone
	if _, err := os.Stat(expectedPath); !os.IsNotExist(err) {
		t.Error("worktree still exists after deletion")
	}
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
	_, _, _ = executeCommand("delete", "feature-x", "--force", "--delete-branch")
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

	stdout, _, err := executeCommand("exit")
	if err != nil {
		t.Fatalf("exit command failed: %v", err)
	}

	if strings.TrimSpace(stdout) != repoRoot {
		t.Errorf("expected %q, got %q", repoRoot, strings.TrimSpace(stdout))
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
	commands := []string{"create", "delete", "list", "cd", "exit", "init"}

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
