package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// setupTestRepo creates a temporary git repository for testing
func setupTestRepo(t *testing.T) (string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "wt-git-test-*")
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

	// Configure git user for commits
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = tmpDir
	_ = cmd.Run()

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	_ = cmd.Run()

	// Create an initial commit
	testFile := filepath.Join(tmpDir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test"), 0644); err != nil {
		cleanup()
		t.Fatalf("failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		cleanup()
		t.Fatalf("failed to git add: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		cleanup()
		t.Fatalf("failed to git commit: %v", err)
	}

	return tmpDir, cleanup
}

func TestCreateAndRemoveWorktree(t *testing.T) {
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	worktreePath := filepath.Join(repoRoot, "worktrees", "test-wt")
	branchName := "test-branch"

	// Create worktree
	err := CreateWorktree(repoRoot, worktreePath, branchName)
	if err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	// Verify worktree exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		t.Error("worktree directory was not created")
	}

	// Verify branch exists
	if !BranchExists(repoRoot, branchName) {
		t.Error("branch was not created")
	}

	// List worktrees
	worktrees, err := ListWorktrees(repoRoot)
	if err != nil {
		t.Fatalf("failed to list worktrees: %v", err)
	}

	found := false
	for _, wt := range worktrees {
		if wt.Path == worktreePath {
			found = true
			if wt.Branch != branchName {
				t.Errorf("expected branch %q, got %q", branchName, wt.Branch)
			}
		}
	}
	if !found {
		t.Error("created worktree not found in list")
	}

	// Remove worktree
	err = RemoveWorktree(repoRoot, worktreePath, false)
	if err != nil {
		t.Fatalf("failed to remove worktree: %v", err)
	}

	// Verify worktree is gone
	if _, err := os.Stat(worktreePath); !os.IsNotExist(err) {
		t.Error("worktree directory still exists after removal")
	}
}

func TestCreateWorktreeFromBranch(t *testing.T) {
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create a branch first
	branchName := "existing-branch"
	cmd := exec.Command("git", "branch", branchName)
	cmd.Dir = repoRoot
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}

	worktreePath := filepath.Join(repoRoot, "worktrees", "test-wt")

	// Create worktree from existing branch
	err := CreateWorktreeFromBranch(repoRoot, worktreePath, branchName)
	if err != nil {
		t.Fatalf("failed to create worktree from branch: %v", err)
	}

	// Verify worktree exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		t.Error("worktree directory was not created")
	}

	// Cleanup
	_ = RemoveWorktree(repoRoot, worktreePath, true)
}

func TestBranchExists(t *testing.T) {
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	// Main/master branch should exist
	mainExists := BranchExists(repoRoot, "main") || BranchExists(repoRoot, "master")
	if !mainExists {
		t.Error("expected main or master branch to exist")
	}

	// Non-existent branch should not exist
	if BranchExists(repoRoot, "non-existent-branch-xyz") {
		t.Error("expected non-existent branch to not exist")
	}
}

func TestGetCurrentBranch(t *testing.T) {
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	branch, err := GetCurrentBranch(repoRoot)
	if err != nil {
		t.Fatalf("failed to get current branch: %v", err)
	}

	if branch != "main" && branch != "master" {
		t.Errorf("expected branch 'main' or 'master', got %q", branch)
	}
}

func TestHasUncommittedChanges(t *testing.T) {
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	// Should have no uncommitted changes initially
	hasChanges, err := HasUncommittedChanges(repoRoot)
	if err != nil {
		t.Fatalf("failed to check for uncommitted changes: %v", err)
	}
	if hasChanges {
		t.Error("expected no uncommitted changes")
	}

	// Create an uncommitted change
	testFile := filepath.Join(repoRoot, "new-file.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Now should have uncommitted changes
	hasChanges, err = HasUncommittedChanges(repoRoot)
	if err != nil {
		t.Fatalf("failed to check for uncommitted changes: %v", err)
	}
	if !hasChanges {
		t.Error("expected uncommitted changes")
	}
}

func TestGetWorktreeName(t *testing.T) {
	tests := []struct {
		name         string
		repoRoot     string
		worktreePath string
		worktreeDir  string
		expected     string
	}{
		{
			name:         "simple path",
			repoRoot:     "/repo",
			worktreePath: "/repo/worktrees/feature-x",
			worktreeDir:  "worktrees",
			expected:     "feature-x",
		},
		{
			name:         "nested path",
			repoRoot:     "/repo",
			worktreePath: "/repo/worktrees/feature-x/src",
			worktreeDir:  "worktrees",
			expected:     "feature-x",
		},
		{
			name:         "custom worktree dir",
			repoRoot:     "/repo",
			worktreePath: "/repo/.wt/my-feature",
			worktreeDir:  ".wt",
			expected:     "my-feature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetWorktreeName(tt.repoRoot, tt.worktreePath, tt.worktreeDir)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestIsInsideWorktree(t *testing.T) {
	tests := []struct {
		name        string
		repoRoot    string
		path        string
		worktreeDir string
		expected    bool
	}{
		{
			name:        "inside worktree",
			repoRoot:    "/repo",
			path:        "/repo/worktrees/feature-x/src",
			worktreeDir: "worktrees",
			expected:    true,
		},
		{
			name:        "at worktree root",
			repoRoot:    "/repo",
			path:        "/repo/worktrees/feature-x",
			worktreeDir: "worktrees",
			expected:    true,
		},
		{
			name:        "outside worktree",
			repoRoot:    "/repo",
			path:        "/repo/src",
			worktreeDir: "worktrees",
			expected:    false,
		},
		{
			name:        "at repo root",
			repoRoot:    "/repo",
			path:        "/repo",
			worktreeDir: "worktrees",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsInsideWorktree(tt.repoRoot, tt.path, tt.worktreeDir)
			if got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestDeleteBranch(t *testing.T) {
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create a branch
	branchName := "branch-to-delete"
	cmd := exec.Command("git", "branch", branchName)
	cmd.Dir = repoRoot
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}

	// Verify branch exists
	if !BranchExists(repoRoot, branchName) {
		t.Fatal("branch was not created")
	}

	// Delete the branch
	err := DeleteBranch(repoRoot, branchName, false)
	if err != nil {
		t.Fatalf("failed to delete branch: %v", err)
	}

	// Verify branch is gone
	if BranchExists(repoRoot, branchName) {
		t.Error("branch still exists after deletion")
	}
}

func TestListWorktrees(t *testing.T) {
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	// Initially should have just the main worktree
	worktrees, err := ListWorktrees(repoRoot)
	if err != nil {
		t.Fatalf("failed to list worktrees: %v", err)
	}

	if len(worktrees) != 1 {
		t.Errorf("expected 1 worktree, got %d", len(worktrees))
	}

	// Create a worktree
	worktreePath := filepath.Join(repoRoot, "worktrees", "test-wt")
	if err := CreateWorktree(repoRoot, worktreePath, "test-branch"); err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	// Now should have 2 worktrees
	worktrees, err = ListWorktrees(repoRoot)
	if err != nil {
		t.Fatalf("failed to list worktrees: %v", err)
	}

	if len(worktrees) != 2 {
		t.Errorf("expected 2 worktrees, got %d", len(worktrees))
	}

	// Cleanup
	_ = RemoveWorktree(repoRoot, worktreePath, true)
}
