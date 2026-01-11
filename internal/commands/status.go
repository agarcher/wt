package commands

import (
	"fmt"
	"strings"

	"github.com/agarcher/wt/internal/git"
)

// ANSI codes for bold text
const (
	bold  = "\033[1m"
	reset = "\033[0m"
)

// FormatCompactStatus builds the compact status string with arrows.
// State indicators (mutually exclusive): new, in_progress, merged
// dirty is additive and can appear alongside any state.
func FormatCompactStatus(status *git.WorktreeStatus) string {
	var parts []string

	if status.CommitsAhead > 0 {
		parts = append(parts, fmt.Sprintf("↑%d", status.CommitsAhead))
	}
	if status.CommitsBehind > 0 {
		parts = append(parts, fmt.Sprintf("↓%d", status.CommitsBehind))
	}

	// Build status tags (state is mutually exclusive, dirty is additive)
	var statusTags []string

	// State indicator: new > in_progress > merged (mutually exclusive)
	if status.IsNew {
		statusTags = append(statusTags, "new")
	} else if status.CommitsAhead > 0 && !status.IsMerged {
		// in_progress: has commits ahead that aren't merged
		statusTags = append(statusTags, bold+"in_progress"+reset)
	} else if status.IsMerged && status.CommitsAhead == 0 {
		statusTags = append(statusTags, FormatMergedStatus(status.MergedPRs))
	}

	// dirty is additive - can appear with any state
	if status.HasUncommittedChanges {
		statusTags = append(statusTags, bold+"dirty"+reset)
	}

	if len(statusTags) > 0 {
		parts = append(parts, "["+strings.Join(statusTags, ", ")+"]")
	}

	return strings.Join(parts, " ")
}

// FormatMergedStatus returns the merged status string.
// If PR numbers are found, returns "merged in #1, #2", otherwise just "merged".
func FormatMergedStatus(prs []string) string {
	if len(prs) == 0 {
		return "merged"
	}
	return "merged in " + strings.Join(prs, ", ")
}
