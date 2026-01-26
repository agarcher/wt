package commands

import (
	"fmt"
	"sync"
	"time"

	"github.com/agarcher/wt/internal/config"
	"github.com/agarcher/wt/internal/git"
	"github.com/agarcher/wt/internal/userconfig"
	"github.com/spf13/cobra"
)

// CompareSetup holds the result of setting up comparison context
type CompareSetup struct {
	RepoRoot      string
	Config        *config.Config
	ComparisonRef string
}

// resolveComparisonRef determines the comparison ref for a repo and optionally fetches from the remote.
// This is the core logic shared by SetupCompare and delete.
func resolveComparisonRef(cmd *cobra.Command, repoRoot string, cfg *config.Config) (string, error) {
	// Load user configuration
	userCfg, err := userconfig.Load()
	if err != nil {
		cmd.PrintErrf("Warning: %v (using defaults)\n", err)
	}

	// Determine remote for this repo (empty = local comparison)
	remote := userCfg.GetRemoteForRepo(repoRoot)

	// Determine comparison branch from repo config, or auto-detect
	branch := cfg.DefaultBranch
	if branch == "" {
		branch, _ = git.GetDefaultBranch(repoRoot)
		if branch == "" {
			branch = "main" // Ultimate fallback
		}
	}

	// Build comparison ref based on whether remote is configured
	var comparisonRef string
	if remote != "" {
		// Remote comparison mode
		remoteRef := remote + "/" + branch // e.g., "origin/main"

		// Fetch based on fetch_interval setting
		fetchInterval := userCfg.GetFetchIntervalForRepo(repoRoot)
		if fetchInterval != userconfig.FetchIntervalNever {
			lastFetch, _ := git.GetLastFetchTime(repoRoot, remote)
			timeSinceLastFetch := time.Since(lastFetch)

			if fetchInterval > 0 && timeSinceLastFetch < fetchInterval {
				// Skip fetch - within interval
				cmd.PrintErrf("Skipping fetch (last fetch %s ago)\n", formatDuration(timeSinceLastFetch))
			} else {
				if err := fetchWithSpinner(cmd, repoRoot, remote); err != nil {
					cmd.PrintErrf("Warning: failed to fetch from %s: %v\n", remote, err)
				}
			}
		}

		// Verify the remote ref exists, fall back to local if not
		if git.RefExists(repoRoot, remoteRef) {
			comparisonRef = remoteRef
		} else {
			cmd.PrintErrf("Warning: %s does not exist, comparing to local %s\n", remoteRef, branch)
			comparisonRef = branch
		}
	} else {
		// Local comparison mode (default) - no network, no fetch
		comparisonRef = branch // e.g., "main"
	}

	return comparisonRef, nil
}

// SetupCompare initializes the comparison context for list/cleanup commands.
// It prints the repo root, determines the comparison ref, and fetches if configured.
func SetupCompare(cmd *cobra.Command) (*CompareSetup, error) {
	// Find the main repository root
	repoRoot, err := config.GetMainRepoRoot()
	if err != nil {
		return nil, fmt.Errorf("not in a git repository: %w", err)
	}

	// Print repo root
	cmd.Printf("Repository: %s\n", repoRoot)

	// Load repo configuration
	cfg, err := config.Load(repoRoot)
	if err != nil {
		// Use defaults if no config file
		cfg = config.DefaultConfig()
	}

	comparisonRef, err := resolveComparisonRef(cmd, repoRoot, cfg)
	if err != nil {
		return nil, err
	}

	// Print comparison ref
	cmd.Printf("Comparing to: %s\n", comparisonRef)
	cmd.Println()

	return &CompareSetup{
		RepoRoot:      repoRoot,
		Config:        cfg,
		ComparisonRef: comparisonRef,
	}, nil
}

// fetchWithSpinner fetches from the remote while displaying a spinner
func fetchWithSpinner(cmd *cobra.Command, repoRoot, remote string) error {
	out := cmd.ErrOrStderr()

	// Start spinner
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		spinChars := []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}
		i := 0
		for {
			select {
			case <-done:
				// Clear the spinner line
				_, _ = fmt.Fprintf(out, "\r\033[K")
				return
			default:
				_, _ = fmt.Fprintf(out, "\r%c Fetching from %s...", spinChars[i%len(spinChars)], remote)
				i++
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()

	// Perform fetch (suppress git output since we have our own spinner)
	err := git.FetchRemoteQuiet(repoRoot, remote)

	// Stop spinner
	close(done)
	wg.Wait()

	if err != nil {
		return err
	}

	// Record successful fetch time
	_ = git.SetLastFetchTime(repoRoot, remote)

	// Update remote HEAD
	_ = git.UpdateRemoteHead(repoRoot, remote)

	// Print success message
	_, _ = fmt.Fprintf(out, "Fetched from %s\n", remote)

	return nil
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	if minutes == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh%dm", hours, minutes)
}
