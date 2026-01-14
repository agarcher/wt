package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/agarcher/wt/internal/config"
	"github.com/agarcher/wt/internal/git"
	"github.com/agarcher/wt/internal/hooks"
	"github.com/spf13/cobra"
)

var (
	createBranch string
)

func init() {
	createCmd.Flags().StringVarP(&createBranch, "branch", "b", "", "Use existing branch instead of creating a new one")
	_ = createCmd.RegisterFlagCompletionFunc("branch", completeBranchNames)
	rootCmd.AddCommand(createCmd)
}

var createCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new worktree",
	Long: `Create a new git worktree with the specified name.

By default, a new branch with the same name will be created.
Use --branch to checkout an existing branch instead.

The worktree will be created in the directory specified by worktree_dir
in your .wt.yaml configuration (default: worktrees/).

After creation, any post_create hooks defined in .wt.yaml will be executed.`,
	Args: cobra.ExactArgs(1),
	RunE: runCreate,
}

func runCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Find the main repository root
	repoRoot, err := config.GetMainRepoRoot()
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	// Load configuration
	cfg, err := config.Load(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to load config: %w (is .wt.yaml present?)", err)
	}

	// Determine the worktree path
	worktreePath := filepath.Join(repoRoot, cfg.WorktreeDir, name)

	// Determine the branch name
	branchName := createBranch
	if branchName == "" {
		// Apply branch pattern
		branchName = strings.ReplaceAll(cfg.BranchPattern, "{name}", name)
	}

	// Create hook environment
	env := &hooks.Env{
		Name:        name,
		Path:        worktreePath,
		Branch:      branchName,
		RepoRoot:    repoRoot,
		WorktreeDir: cfg.WorktreeDir,
	}

	// Run pre-create hooks
	if err := hooks.RunPreCreate(cfg, env); err != nil {
		return fmt.Errorf("pre-create hook failed: %w", err)
	}

	// Create the worktree
	if createBranch != "" {
		// Use existing branch
		if !git.BranchExists(repoRoot, createBranch) {
			return fmt.Errorf("branch %q does not exist", createBranch)
		}
		cmd.Printf("Creating worktree %q from branch %q...\n", name, createBranch)
		if err := git.CreateWorktreeFromBranch(repoRoot, worktreePath, createBranch); err != nil {
			return fmt.Errorf("failed to create worktree: %w", err)
		}
	} else {
		// Create new branch
		if git.BranchExists(repoRoot, branchName) {
			return fmt.Errorf("branch %q already exists (use --branch to checkout existing branch)", branchName)
		}
		cmd.Printf("Creating worktree %q with new branch %q...\n", name, branchName)
		if err := git.CreateWorktree(repoRoot, worktreePath, branchName); err != nil {
			return fmt.Errorf("failed to create worktree: %w", err)
		}
	}

	// Store creation metadata for status tracking
	if err := git.SetWorktreeCreatedAt(repoRoot, name, time.Now()); err != nil {
		cmd.Printf("Warning: could not store creation time: %v\n", err)
	}
	if initialCommit, err := git.GetCurrentCommit(worktreePath); err == nil {
		if err := git.SetWorktreeInitialCommit(repoRoot, name, initialCommit); err != nil {
			cmd.Printf("Warning: could not store initial commit: %v\n", err)
		}
	}

	// Allocate and store worktree index
	index, err := git.AllocateIndex(repoRoot, cfg.Index.Max)
	if err != nil {
		cmd.Printf("Warning: could not allocate index: %v\n", err)
	} else {
		if err := git.SetWorktreeIndex(repoRoot, name, index); err != nil {
			cmd.Printf("Warning: could not store index: %v\n", err)
		} else {
			env.Index = index
		}
	}

	// Run post-create hooks
	if err := hooks.RunPostCreate(cfg, env); err != nil {
		cmd.Printf("Warning: post-create hook failed: %v\n", err)
		// Don't fail the whole operation for post-create hooks
	}

	cmd.Printf("Worktree %q created successfully\n", name)

	// Output the path for shell wrapper or print helpful message for direct invocation
	if cdFile := os.Getenv("WT_CD_FILE"); cdFile != "" {
		// Shell wrapper mode: write path to file for cd
		_ = os.WriteFile(cdFile, []byte(worktreePath+"\n"), 0600)
	} else {
		// Direct invocation: print helpful message
		cmd.Printf("\nRun `cd %s` to open your new worktree\n", worktreePath)
	}

	return nil
}
