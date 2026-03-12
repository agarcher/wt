package commands

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/agarcher/wt/internal/git"
)

// KeyValue represents a key-value pair for formatting
type KeyValue struct {
	Key   string
	Value string
}

// VerboseInfo holds all the data needed to format verbose worktree output
type VerboseInfo struct {
	Name          string
	Branch        string
	Index         int
	CreatedAt     time.Time
	Status        *git.WorktreeStatus
	CurrentMarker string
	HookOutput    string
	RepoURL       string // GitHub base URL for PR hyperlinks (empty disables linking)
}

// PrintVerboseWorktree prints a single worktree in verbose format
func PrintVerboseWorktree(out io.Writer, info VerboseInfo) {
	// Collect all key-value pairs
	pairs := collectBuiltinPairs(info)

	// Parse and append hook output
	hookPairs, rawLines := ParseHookKeyValues(info.HookOutput)
	pairs = append(pairs, hookPairs...)

	// Calculate max key width for alignment
	maxKeyWidth := 0
	for _, p := range pairs {
		if len(p.Key) > maxKeyWidth {
			maxKeyWidth = len(p.Key)
		}
	}

	// Print the header line with name and current marker
	_, _ = fmt.Fprintf(out, "%s%s\n", info.CurrentMarker, info.Name)

	// Print all key-value pairs with alignment
	for _, p := range pairs {
		_, _ = fmt.Fprintf(out, "  %-*s %s\n", maxKeyWidth+1, p.Key+":", p.Value)
	}

	// Print raw lines if any (non key-value hook output)
	if len(rawLines) > 0 {
		_, _ = fmt.Fprintln(out)
		for _, line := range rawLines {
			_, _ = fmt.Fprintf(out, "  %s\n", line)
		}
	}
}

// collectBuiltinPairs collects the built-in key-value pairs for a worktree
func collectBuiltinPairs(info VerboseInfo) []KeyValue {
	var pairs []KeyValue

	pairs = append(pairs, KeyValue{Key: "Branch", Value: info.Branch})

	if info.Index > 0 {
		pairs = append(pairs, KeyValue{Key: "Index", Value: fmt.Sprintf("%d", info.Index)})
	}

	if !info.CreatedAt.IsZero() {
		age := formatAge(time.Since(info.CreatedAt))
		pairs = append(pairs, KeyValue{
			Key:   "Created",
			Value: fmt.Sprintf("%s (%s ago)", info.CreatedAt.Format("2006-01-02"), age),
		})
	}

	statusStr := FormatCompactStatus(info.Status, info.RepoURL)
	if statusStr != "" {
		pairs = append(pairs, KeyValue{Key: "Status", Value: statusStr})
	}

	return pairs
}

// ParseHookKeyValues parses hook output into key-value pairs and raw lines.
// Lines matching "Key: value" pattern become KeyValue pairs.
// Other non-empty lines are returned as raw lines.
func ParseHookKeyValues(output string) (pairs []KeyValue, raw []string) {
	if output == "" {
		return nil, nil
	}

	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Try to match "Key: value" pattern
		if idx := strings.Index(trimmed, ":"); idx > 0 {
			key := strings.TrimSpace(trimmed[:idx])
			value := strings.TrimSpace(trimmed[idx+1:])
			if len(key) > 0 && len(value) > 0 {
				pairs = append(pairs, KeyValue{Key: key, Value: value})
				continue
			}
		}
		raw = append(raw, trimmed)
	}
	return pairs, raw
}

// formatAge formats a duration as a human-readable age string
func formatAge(d time.Duration) string {
	days := int(d.Hours() / 24)

	if days == 0 {
		hours := int(d.Hours())
		if hours == 0 {
			return "less than an hour"
		}
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	}

	if days == 1 {
		return "1 day"
	}

	weeks := days / 7
	if weeks >= 1 {
		if weeks == 1 {
			return "1 week"
		}
		return fmt.Sprintf("%d weeks", weeks)
	}

	return fmt.Sprintf("%d days", days)
}
