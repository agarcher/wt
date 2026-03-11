package commands

import (
	"os/exec"
	"strings"

	"github.com/agarcher/wt/internal/config"
	"github.com/agarcher/wt/internal/git"
	"github.com/spf13/cobra"
)

// completeWorktreeNames returns a completion function that provides worktree names
func completeWorktreeNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Only complete the first argument
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	repoRoot, err := config.GetMainRepoRoot()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	resolved, err := config.Resolve(repoRoot)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	cfg := resolved.Config

	worktrees, err := git.ListWorktrees(repoRoot)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var names []string
	for _, wt := range worktrees {
		if wt.Bare {
			continue // Skip bare repo
		}
		name := git.GetWorktreeName(repoRoot, wt.Path, cfg.WorktreeDir)
		// Skip empty names (which means it's the main worktree)
		if name == "" || name == "." || strings.HasPrefix(name, "..") {
			continue
		}
		if strings.HasPrefix(name, toComplete) {
			names = append(names, name)
		}
	}

	return names, cobra.ShellCompDirectiveNoFileComp
}

// completeBranchNames returns a completion function that provides branch names
func completeBranchNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	repoRoot, err := config.GetMainRepoRoot()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Get local branches - try --format first (Git 2.13+), fall back to plain git branch
	output, err := exec.Command("git", "-C", repoRoot, "branch", "--format=%(refname:short)").Output()
	if err != nil {
		// Fallback for older Git versions
		output, err = exec.Command("git", "-C", repoRoot, "branch").Output()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
	}

	var branches []string
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		branch := strings.TrimSpace(line)
		// Remove leading * or + markers from git branch output
		branch = strings.TrimPrefix(branch, "* ")
		branch = strings.TrimPrefix(branch, "+ ")
		if branch != "" && strings.HasPrefix(branch, toComplete) {
			branches = append(branches, branch)
		}
	}

	return branches, cobra.ShellCompDirectiveNoFileComp
}
