package git

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Worktree represents a git worktree
type Worktree struct {
	Path   string
	Branch string
	Commit string
	Bare   bool
}

// CreateWorktree creates a new git worktree with a new branch
func CreateWorktree(repoRoot, worktreePath, branchName string) error {
	cmd := exec.Command("git", "worktree", "add", "-b", branchName, worktreePath)
	cmd.Dir = repoRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// CreateWorktreeFromBranch creates a new git worktree from an existing branch
func CreateWorktreeFromBranch(repoRoot, worktreePath, branchName string) error {
	cmd := exec.Command("git", "worktree", "add", worktreePath, branchName)
	cmd.Dir = repoRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RemoveWorktree removes a git worktree
func RemoveWorktree(repoRoot, worktreePath string, force bool) error {
	args := []string{"worktree", "remove", worktreePath}
	if force {
		args = append(args, "--force")
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ListWorktrees returns all worktrees for a repository
func ListWorktrees(repoRoot string) ([]Worktree, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = repoRoot

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var worktrees []Worktree
	var current *Worktree

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "worktree ") {
			if current != nil {
				worktrees = append(worktrees, *current)
			}
			current = &Worktree{
				Path: strings.TrimPrefix(line, "worktree "),
			}
		} else if strings.HasPrefix(line, "HEAD ") && current != nil {
			current.Commit = strings.TrimPrefix(line, "HEAD ")
		} else if strings.HasPrefix(line, "branch ") && current != nil {
			branch := strings.TrimPrefix(line, "branch ")
			// Remove refs/heads/ prefix
			current.Branch = strings.TrimPrefix(branch, "refs/heads/")
		} else if line == "bare" && current != nil {
			current.Bare = true
		}
	}

	if current != nil {
		worktrees = append(worktrees, *current)
	}

	return worktrees, scanner.Err()
}

// BranchExists checks if a branch exists
func BranchExists(repoRoot, branchName string) bool {
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+branchName)
	cmd.Dir = repoRoot
	return cmd.Run() == nil
}

// GetCurrentBranch returns the current branch name
func GetCurrentBranch(repoRoot string) (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = repoRoot

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// HasUncommittedChanges checks if there are uncommitted changes
func HasUncommittedChanges(path string) (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = path

	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	return len(strings.TrimSpace(string(output))) > 0, nil
}

// HasUnpushedCommits checks if there are unpushed commits
func HasUnpushedCommits(path string) (bool, error) {
	// Get the current branch
	branchCmd := exec.Command("git", "branch", "--show-current")
	branchCmd.Dir = path
	branchOutput, err := branchCmd.Output()
	if err != nil {
		return false, nil // No branch, no unpushed commits
	}
	branch := strings.TrimSpace(string(branchOutput))
	if branch == "" {
		return false, nil
	}

	// Check if there's a remote tracking branch
	remoteCmd := exec.Command("git", "rev-parse", "--abbrev-ref", branch+"@{upstream}")
	remoteCmd.Dir = path
	if remoteCmd.Run() != nil {
		return false, nil // No upstream, can't have unpushed commits
	}

	// Count commits ahead of upstream
	cmd := exec.Command("git", "rev-list", "--count", branch+"@{upstream}..HEAD")
	cmd.Dir = path

	output, err := cmd.Output()
	if err != nil {
		return false, nil
	}

	count := strings.TrimSpace(string(output))
	return count != "0", nil
}

// GetWorktreeName extracts the worktree name from a path
func GetWorktreeName(repoRoot, worktreePath, worktreeDir string) string {
	worktreesPath := filepath.Join(repoRoot, worktreeDir)
	rel, err := filepath.Rel(worktreesPath, worktreePath)
	if err != nil {
		return filepath.Base(worktreePath)
	}
	// Get the first component of the relative path
	parts := strings.Split(rel, string(filepath.Separator))
	if len(parts) > 0 {
		return parts[0]
	}
	return filepath.Base(worktreePath)
}

// IsInsideWorktree checks if the given path is inside a worktree directory
func IsInsideWorktree(repoRoot, path, worktreeDir string) bool {
	worktreesPath := filepath.Join(repoRoot, worktreeDir)
	rel, err := filepath.Rel(worktreesPath, path)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, "..") && rel != "."
}

// PruneWorktrees cleans up stale worktree references
func PruneWorktrees(repoRoot string) error {
	cmd := exec.Command("git", "worktree", "prune")
	cmd.Dir = repoRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// DeleteBranch deletes a local branch
func DeleteBranch(repoRoot, branchName string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	cmd := exec.Command("git", "branch", flag, branchName)
	cmd.Dir = repoRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// GetDefaultBranch returns the default branch name (main or master)
func GetDefaultBranch(repoRoot string) (string, error) {
	// Try to get the default branch from remote
	cmd := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD")
	cmd.Dir = repoRoot
	output, err := cmd.Output()
	if err == nil {
		ref := strings.TrimSpace(string(output))
		return strings.TrimPrefix(ref, "refs/remotes/origin/"), nil
	}

	// Fallback: check if main or master exists
	if BranchExists(repoRoot, "main") {
		return "main", nil
	}
	if BranchExists(repoRoot, "master") {
		return "master", nil
	}

	return "", fmt.Errorf("could not determine default branch")
}
