package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
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

func TestGetCommitsAheadBehind(t *testing.T) {
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	// Get the main branch name
	mainBranch, err := GetCurrentBranch(repoRoot)
	if err != nil {
		t.Fatalf("failed to get current branch: %v", err)
	}

	// Create a worktree with a new branch
	worktreePath := filepath.Join(repoRoot, "worktrees", "test-wt")
	if err := CreateWorktree(repoRoot, worktreePath, "test-branch"); err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}
	defer func() { _ = RemoveWorktree(repoRoot, worktreePath, true) }()

	// Initially should be 0 ahead, 0 behind
	ahead, behind, err := GetCommitsAheadBehind(repoRoot, worktreePath, mainBranch)
	if err != nil {
		t.Fatalf("failed to get commits ahead/behind: %v", err)
	}
	if ahead != 0 || behind != 0 {
		t.Errorf("expected 0 ahead, 0 behind; got %d ahead, %d behind", ahead, behind)
	}

	// Add a commit in the worktree
	testFile := filepath.Join(worktreePath, "new-file.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = worktreePath
	_ = cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Add new file")
	cmd.Dir = worktreePath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	// Now should be 1 ahead, 0 behind
	ahead, behind, err = GetCommitsAheadBehind(repoRoot, worktreePath, mainBranch)
	if err != nil {
		t.Fatalf("failed to get commits ahead/behind: %v", err)
	}
	if ahead != 1 || behind != 0 {
		t.Errorf("expected 1 ahead, 0 behind; got %d ahead, %d behind", ahead, behind)
	}

	// Add a commit on main branch
	mainFile := filepath.Join(repoRoot, "main-file.txt")
	if err := os.WriteFile(mainFile, []byte("main"), 0644); err != nil {
		t.Fatalf("failed to create main file: %v", err)
	}
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = repoRoot
	_ = cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Add main file")
	cmd.Dir = repoRoot
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to commit on main: %v", err)
	}

	// Now should be 1 ahead, 1 behind
	ahead, behind, err = GetCommitsAheadBehind(repoRoot, worktreePath, mainBranch)
	if err != nil {
		t.Fatalf("failed to get commits ahead/behind: %v", err)
	}
	if ahead != 1 || behind != 1 {
		t.Errorf("expected 1 ahead, 1 behind; got %d ahead, %d behind", ahead, behind)
	}
}

func TestGetMergedBranches(t *testing.T) {
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	mainBranch, err := GetCurrentBranch(repoRoot)
	if err != nil {
		t.Fatalf("failed to get current branch: %v", err)
	}

	// Create a branch and merge it
	cmd := exec.Command("git", "branch", "merged-branch")
	cmd.Dir = repoRoot
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}

	// Create an unmerged branch with a commit
	cmd = exec.Command("git", "checkout", "-b", "unmerged-branch")
	cmd.Dir = repoRoot
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create unmerged branch: %v", err)
	}
	testFile := filepath.Join(repoRoot, "unmerged.txt")
	if err := os.WriteFile(testFile, []byte("unmerged"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = repoRoot
	_ = cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Unmerged commit")
	cmd.Dir = repoRoot
	_ = cmd.Run()

	// Go back to main
	cmd = exec.Command("git", "checkout", mainBranch)
	cmd.Dir = repoRoot
	_ = cmd.Run()

	merged, err := GetMergedBranches(repoRoot, mainBranch)
	if err != nil {
		t.Fatalf("failed to get merged branches: %v", err)
	}

	// merged-branch should be in the list (it's at same point as main)
	if !merged["merged-branch"] {
		t.Error("expected merged-branch to be in merged list")
	}

	// unmerged-branch should not be in the list
	if merged["unmerged-branch"] {
		t.Error("expected unmerged-branch to not be in merged list")
	}
}

func TestIsBranchMerged(t *testing.T) {
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	mainBranch, err := GetCurrentBranch(repoRoot)
	if err != nil {
		t.Fatalf("failed to get current branch: %v", err)
	}

	// Create a branch at the same point as main
	cmd := exec.Command("git", "branch", "merged-branch")
	cmd.Dir = repoRoot
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}

	isMerged, err := IsBranchMerged(repoRoot, "merged-branch", mainBranch)
	if err != nil {
		t.Fatalf("failed to check if branch merged: %v", err)
	}
	if !isMerged {
		t.Error("expected merged-branch to be merged")
	}

	isNonExistentMerged, _ := IsBranchMerged(repoRoot, "non-existent", mainBranch)
	if isNonExistentMerged {
		t.Error("expected non-existent branch to not be merged")
	}
}

func TestSetAndGetWorktreeCreatedAt(t *testing.T) {
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create a worktree
	worktreePath := filepath.Join(repoRoot, "worktrees", "test-wt")
	worktreeName := "test-wt"
	if err := CreateWorktree(repoRoot, worktreePath, "test-branch"); err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}
	defer func() { _ = RemoveWorktree(repoRoot, worktreePath, true) }()

	// Initially should return zero time
	createdAt, err := GetWorktreeCreatedAt(repoRoot, worktreeName)
	if err != nil {
		t.Fatalf("failed to get created at: %v", err)
	}
	if !createdAt.IsZero() {
		t.Errorf("expected zero time, got %v", createdAt)
	}

	// Set creation time
	now := time.Now().Truncate(time.Second) // Truncate to second precision
	if err := SetWorktreeCreatedAt(repoRoot, worktreeName, now); err != nil {
		t.Fatalf("failed to set created at: %v", err)
	}

	// Get it back
	createdAt, err = GetWorktreeCreatedAt(repoRoot, worktreeName)
	if err != nil {
		t.Fatalf("failed to get created at: %v", err)
	}
	if createdAt.Unix() != now.Unix() {
		t.Errorf("expected %v, got %v", now, createdAt)
	}
}

func TestGetWorktreeStatus(t *testing.T) {
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	mainBranch, err := GetCurrentBranch(repoRoot)
	if err != nil {
		t.Fatalf("failed to get current branch: %v", err)
	}

	// Create a worktree
	worktreePath := filepath.Join(repoRoot, "worktrees", "test-wt")
	worktreeName := "test-wt"
	branchName := "test-branch"
	if err := CreateWorktree(repoRoot, worktreePath, branchName); err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}
	defer func() { _ = RemoveWorktree(repoRoot, worktreePath, true) }()

	// Set creation time
	now := time.Now()
	_ = SetWorktreeCreatedAt(repoRoot, worktreeName, now)

	// Get status
	status, err := GetWorktreeStatus(repoRoot, worktreePath, worktreeName, branchName, mainBranch, nil)
	if err != nil {
		t.Fatalf("failed to get worktree status: %v", err)
	}

	// Should have no uncommitted changes
	if status.HasUncommittedChanges {
		t.Error("expected no uncommitted changes")
	}

	// Should be 0 ahead, 0 behind
	if status.CommitsAhead != 0 || status.CommitsBehind != 0 {
		t.Errorf("expected 0 ahead, 0 behind; got %d ahead, %d behind", status.CommitsAhead, status.CommitsBehind)
	}

	// Should be merged (at same point as main)
	if !status.IsMerged {
		t.Error("expected branch to be merged")
	}

	// Should have creation time
	if status.CreatedAt.Unix() != now.Unix() {
		t.Errorf("expected created at %v, got %v", now, status.CreatedAt)
	}

	// Add uncommitted changes
	testFile := filepath.Join(worktreePath, "uncommitted.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	status, err = GetWorktreeStatus(repoRoot, worktreePath, worktreeName, branchName, mainBranch, nil)
	if err != nil {
		t.Fatalf("failed to get worktree status: %v", err)
	}
	if !status.HasUncommittedChanges {
		t.Error("expected uncommitted changes")
	}
}

func TestSetAndGetWorktreeInitialCommit(t *testing.T) {
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create a worktree
	worktreePath := filepath.Join(repoRoot, "worktrees", "test-wt")
	worktreeName := "test-wt"
	if err := CreateWorktree(repoRoot, worktreePath, "test-branch"); err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}
	defer func() { _ = RemoveWorktree(repoRoot, worktreePath, true) }()

	// Initially should return empty string
	initialCommit, err := GetWorktreeInitialCommit(repoRoot, worktreeName)
	if err != nil {
		t.Fatalf("failed to get initial commit: %v", err)
	}
	if initialCommit != "" {
		t.Errorf("expected empty string, got %q", initialCommit)
	}

	// Get current commit
	currentCommit, err := GetCurrentCommit(worktreePath)
	if err != nil {
		t.Fatalf("failed to get current commit: %v", err)
	}

	// Set initial commit
	if err := SetWorktreeInitialCommit(repoRoot, worktreeName, currentCommit); err != nil {
		t.Fatalf("failed to set initial commit: %v", err)
	}

	// Get it back
	initialCommit, err = GetWorktreeInitialCommit(repoRoot, worktreeName)
	if err != nil {
		t.Fatalf("failed to get initial commit: %v", err)
	}
	if initialCommit != currentCommit {
		t.Errorf("expected %q, got %q", currentCommit, initialCommit)
	}
}

func TestIsNewStatus(t *testing.T) {
	repoRoot, cleanup := setupTestRepo(t)
	defer cleanup()

	mainBranch, err := GetCurrentBranch(repoRoot)
	if err != nil {
		t.Fatalf("failed to get current branch: %v", err)
	}

	// Create a worktree
	worktreePath := filepath.Join(repoRoot, "worktrees", "test-wt")
	worktreeName := "test-wt"
	branchName := "test-branch"
	if err := CreateWorktree(repoRoot, worktreePath, branchName); err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}
	defer func() { _ = RemoveWorktree(repoRoot, worktreePath, true) }()

	// Store initial commit (simulating what create command does)
	initialCommit, _ := GetCurrentCommit(worktreePath)
	_ = SetWorktreeInitialCommit(repoRoot, worktreeName, initialCommit)

	// Should be marked as new (still on initial commit)
	status, err := GetWorktreeStatus(repoRoot, worktreePath, worktreeName, branchName, mainBranch, nil)
	if err != nil {
		t.Fatalf("failed to get worktree status: %v", err)
	}
	if !status.IsNew {
		t.Error("expected IsNew to be true")
	}

	// Add a commit in the worktree
	testFile := filepath.Join(worktreePath, "new-file.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = worktreePath
	_ = cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Add new file")
	cmd.Dir = worktreePath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	// Should no longer be new
	status, err = GetWorktreeStatus(repoRoot, worktreePath, worktreeName, branchName, mainBranch, nil)
	if err != nil {
		t.Fatalf("failed to get worktree status: %v", err)
	}
	if status.IsNew {
		t.Error("expected IsNew to be false after committing")
	}
}
