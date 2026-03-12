package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/agarcher/wt/internal/git"
	"golang.org/x/term"
)

// ANSI codes for bold text
const (
	bold  = "\033[1m"
	reset = "\033[0m"
)

// LinkContext controls whether PR numbers are rendered as clickable OSC 8 hyperlinks.
// Use NewLinkContext to construct; the zero value disables linking.
type LinkContext struct {
	RepoURL string // GitHub base URL (e.g. "https://github.com/owner/repo")
	IsTTY   bool   // whether stdout is a terminal
}

// NewLinkContext creates a LinkContext by resolving the remote URL and checking
// whether stdout is a terminal. This is the standard way to construct a LinkContext
// so that callers don't forget either check.
func NewLinkContext(repoRoot string) LinkContext {
	return LinkContext{
		RepoURL: git.GetRemoteURL(repoRoot, "origin"),
		IsTTY:   term.IsTerminal(int(os.Stdout.Fd())),
	}
}

// linksEnabled returns true when both a repo URL is available and output is a TTY.
func (lc LinkContext) linksEnabled() bool {
	return lc.RepoURL != "" && lc.IsTTY
}

// FormatCompactStatus builds the compact status string with arrows.
// State indicators (mutually exclusive): new, in_progress, merged
// dirty is additive and can appear alongside any state.
func FormatCompactStatus(status *git.WorktreeStatus, links LinkContext) string {
	if status == nil {
		return ""
	}

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
		statusTags = append(statusTags, FormatMergedStatus(status.MergedPRs, links))
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
// When links are enabled, PR numbers are wrapped in OSC 8 terminal hyperlinks.
func FormatMergedStatus(prs []string, links LinkContext) string {
	if len(prs) == 0 {
		return "merged"
	}
	if links.linksEnabled() {
		linked := make([]string, len(prs))
		for i, pr := range prs {
			// pr is like "#123", extract the number
			num := strings.TrimPrefix(pr, "#")
			url := links.RepoURL + "/pull/" + num
			linked[i] = fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", url, pr)
		}
		return "merged in " + strings.Join(linked, ", ")
	}
	return "merged in " + strings.Join(prs, ", ")
}
