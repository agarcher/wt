package git

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
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

// WorktreeStatus holds detailed status information for a worktree
type WorktreeStatus struct {
	HasUncommittedChanges bool
	CommitsAhead          int
	CommitsBehind         int
	IsMerged              bool
	MergedPRs             []string // PR numbers found in merge commits (e.g., ["#1", "#2"])
	IsNew                 bool     // true if still on the initial commit (no new commits yet)
	CreatedAt             time.Time
}

// GetCommitsAheadBehind returns the number of commits ahead and behind the main branch
func GetCommitsAheadBehind(repoRoot, worktreePath, mainBranch string) (ahead, behind int, err error) {
	// Get the current branch for the worktree
	branchCmd := exec.Command("git", "branch", "--show-current")
	branchCmd.Dir = worktreePath
	branchOutput, err := branchCmd.Output()
	if err != nil {
		return 0, 0, nil // No branch, return zeros
	}
	branch := strings.TrimSpace(string(branchOutput))
	if branch == "" {
		return 0, 0, nil // Detached HEAD
	}

	// Use rev-list with left-right to count commits in both directions
	// Format: <behind>\t<ahead>
	cmd := exec.Command("git", "rev-list", "--count", "--left-right", mainBranch+"..."+branch)
	cmd.Dir = repoRoot

	output, err := cmd.Output()
	if err != nil {
		return 0, 0, nil // Branch comparison failed, likely no common ancestor
	}

	parts := strings.Split(strings.TrimSpace(string(output)), "\t")
	if len(parts) != 2 {
		return 0, 0, nil
	}

	behind, _ = strconv.Atoi(parts[0])
	ahead, _ = strconv.Atoi(parts[1])
	return ahead, behind, nil
}

// GetMergedBranches returns a set of branch names that have been merged into the main branch
func GetMergedBranches(repoRoot, mainBranch string) (map[string]bool, error) {
	merged := make(map[string]bool)

	// Get local merged branches
	cmd := exec.Command("git", "branch", "--merged", mainBranch)
	cmd.Dir = repoRoot
	output, err := cmd.Output()
	if err != nil {
		return merged, nil
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Remove leading markers: * for current branch, + for worktree branches
		line = strings.TrimPrefix(line, "* ")
		line = strings.TrimPrefix(line, "+ ")
		if line != "" && line != mainBranch {
			merged[line] = true
		}
	}

	return merged, nil
}

// IsBranchMerged checks if a branch has been merged into the main branch
func IsBranchMerged(repoRoot, branchName, mainBranch string) (bool, error) {
	merged, err := GetMergedBranches(repoRoot, mainBranch)
	if err != nil {
		return false, err
	}
	return merged[branchName], nil
}

// prNumberRegex matches GitHub-style PR references like "pull request #123"
var prNumberRegex = regexp.MustCompile(`(?i)pull request #(\d+)`)

// GetMergePRs finds PR numbers from merge commits that reference the given branch.
// It searches recent merge commits on the main branch for GitHub-style merge commit messages.
// Returns PR numbers like ["#1", "#2"] or nil if none found.
func GetMergePRs(repoRoot, branchName, mainBranch string) []string {
	// Search last 100 merge commits on main branch for mentions of this branch
	// GitHub merge commit format: "Merge pull request #123 from owner/branch-name"
	// Use --pretty=%s to get just the subject line without SHA prefix
	cmd := exec.Command("git", "log", mainBranch, "--merges", "-n", "100", "--pretty=%s")
	cmd.Dir = repoRoot

	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var prs []string
	seen := make(map[string]bool)

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		// Check if this merge commit mentions our branch exactly
		// Typical formats:
		//   "Merge pull request #123 from owner/branch-name"
		//   "Merge branch 'branch-name' into main"
		if !matchesBranchName(line, branchName) {
			continue
		}

		// Extract PR number using regex for "pull request #123" pattern
		matches := prNumberRegex.FindStringSubmatch(line)
		if len(matches) >= 2 {
			pr := "#" + matches[1]
			if !seen[pr] {
				seen[pr] = true
				prs = append(prs, pr)
			}
		}
	}

	if scanner.Err() != nil {
		return prs // best-effort on scan error
	}
	return prs
}

// matchesBranchName checks if a merge commit message references the exact branch name.
// It handles GitHub format "from owner/branch-name" and git format "'branch-name'".
func matchesBranchName(line, branchName string) bool {
	// Check for GitHub PR format: "from owner/branch-name" or "from branch-name"
	// The branch name should be at the end of the line or followed by whitespace
	fromIdx := strings.Index(line, "from ")
	if fromIdx != -1 {
		afterFrom := line[fromIdx+5:]
		fields := strings.Fields(afterFrom)
		if len(fields) > 0 {
			token := fields[0] // "owner/feature/cleanup" or "feature/cleanup"
			// First try exact match (handles "from feature/cleanup" with no owner)
			if token == branchName {
				return true
			}
			// If token includes an owner prefix, strip only that first segment
			// e.g., "owner/feature/cleanup" -> "feature/cleanup"
			if slashIdx := strings.Index(token, "/"); slashIdx != -1 && token[slashIdx+1:] == branchName {
				return true
			}
		}
	}

	// Check for git merge format: "Merge branch 'branch-name'"
	// Look for the branch name in single quotes
	pattern := "'" + branchName + "'"
	return strings.Contains(line, pattern)
}

// SetWorktreeCreatedAt stores the creation timestamp in the worktree's git config
func SetWorktreeCreatedAt(repoRoot, worktreeName string, timestamp time.Time) error {
	configPath := filepath.Join(repoRoot, ".git", "worktrees", worktreeName, "config")

	// Verify the worktree directory exists (git will create the config file)
	worktreeDir := filepath.Dir(configPath)
	if _, err := os.Stat(worktreeDir); os.IsNotExist(err) {
		return fmt.Errorf("worktree directory not found: %s", worktreeDir)
	}

	cmd := exec.Command("git", "config", "--file", configPath, "wt.createdAt", strconv.FormatInt(timestamp.Unix(), 10))
	cmd.Dir = repoRoot
	return cmd.Run()
}

// GetWorktreeCreatedAt retrieves the creation timestamp from the worktree's git config
func GetWorktreeCreatedAt(repoRoot, worktreeName string) (time.Time, error) {
	configPath := filepath.Join(repoRoot, ".git", "worktrees", worktreeName, "config")

	cmd := exec.Command("git", "config", "--file", configPath, "--get", "wt.createdAt")
	cmd.Dir = repoRoot

	output, err := cmd.Output()
	if err != nil {
		return time.Time{}, nil // Not set, return zero time
	}

	timestamp, err := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64)
	if err != nil {
		return time.Time{}, nil
	}

	return time.Unix(timestamp, 0), nil
}

// SetWorktreeInitialCommit stores the initial commit SHA in the worktree's git config
func SetWorktreeInitialCommit(repoRoot, worktreeName, commitSHA string) error {
	configPath := filepath.Join(repoRoot, ".git", "worktrees", worktreeName, "config")

	// Verify the worktree directory exists (git will create the config file)
	worktreeDir := filepath.Dir(configPath)
	if _, err := os.Stat(worktreeDir); os.IsNotExist(err) {
		return fmt.Errorf("worktree directory not found: %s", worktreeDir)
	}

	cmd := exec.Command("git", "config", "--file", configPath, "wt.initialCommit", commitSHA)
	cmd.Dir = repoRoot
	return cmd.Run()
}

// GetWorktreeInitialCommit retrieves the initial commit SHA from the worktree's git config
func GetWorktreeInitialCommit(repoRoot, worktreeName string) (string, error) {
	configPath := filepath.Join(repoRoot, ".git", "worktrees", worktreeName, "config")

	cmd := exec.Command("git", "config", "--file", configPath, "--get", "wt.initialCommit")
	cmd.Dir = repoRoot

	output, err := cmd.Output()
	if err != nil {
		return "", nil // Not set, return empty string
	}

	return strings.TrimSpace(string(output)), nil
}

// GetCurrentCommit returns the current HEAD commit SHA for a path
func GetCurrentCommit(path string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = path

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// GetWorktreeStatus gathers all status information for a worktree
func GetWorktreeStatus(repoRoot, worktreePath, worktreeName, branchName, mainBranch string, mergedCache map[string]bool) (*WorktreeStatus, error) {
	status := &WorktreeStatus{}

	// Check for uncommitted changes
	hasChanges, err := HasUncommittedChanges(worktreePath)
	if err == nil {
		status.HasUncommittedChanges = hasChanges
	}

	// Get commits ahead/behind
	ahead, behind, _ := GetCommitsAheadBehind(repoRoot, worktreePath, mainBranch)
	status.CommitsAhead = ahead
	status.CommitsBehind = behind

	// Check if merged (use cache if provided)
	if mergedCache != nil {
		status.IsMerged = mergedCache[branchName]
	} else {
		merged, _ := IsBranchMerged(repoRoot, branchName, mainBranch)
		status.IsMerged = merged
	}

	// If merged, find associated PR numbers from merge commits
	if status.IsMerged {
		status.MergedPRs = GetMergePRs(repoRoot, branchName, mainBranch)
	}

	// Check if still on initial commit (new worktree with no changes committed)
	initialCommit, _ := GetWorktreeInitialCommit(repoRoot, worktreeName)
	if initialCommit != "" {
		currentCommit, _ := GetCurrentCommit(worktreePath)
		status.IsNew = (currentCommit == initialCommit)
	}

	// Get creation time
	createdAt, _ := GetWorktreeCreatedAt(repoRoot, worktreeName)
	status.CreatedAt = createdAt

	return status, nil
}
