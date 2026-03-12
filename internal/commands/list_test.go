package commands

import (
	"strings"
	"testing"

	"github.com/agarcher/wt/internal/git"
)

func TestFormatCompactStatus(t *testing.T) {
	tests := []struct {
		name       string
		status     *git.WorktreeStatus
		want       string
		wantBold   []string // substrings that should be bold
		wantNoBold []string // substrings that should NOT be bold
	}{
		{
			name: "new worktree",
			status: &git.WorktreeStatus{
				IsNew: true,
			},
			want:       "[new]",
			wantNoBold: []string{"new"},
		},
		{
			name: "in_progress - has commits ahead, not merged",
			status: &git.WorktreeStatus{
				CommitsAhead: 3,
				IsMerged:     false,
			},
			want:     "↑3 [in_progress]",
			wantBold: []string{"in_progress"},
		},
		{
			name: "merged - no commits ahead",
			status: &git.WorktreeStatus{
				IsMerged:     true,
				CommitsAhead: 0,
			},
			want:       "[merged]",
			wantNoBold: []string{"merged"},
		},
		{
			name: "dirty only",
			status: &git.WorktreeStatus{
				HasUncommittedChanges: true,
			},
			want:     "[dirty]",
			wantBold: []string{"dirty"},
		},
		{
			name: "new and dirty",
			status: &git.WorktreeStatus{
				IsNew:                 true,
				HasUncommittedChanges: true,
			},
			want:       "[new, dirty]",
			wantBold:   []string{"dirty"},
			wantNoBold: []string{"new"},
		},
		{
			name: "in_progress and dirty",
			status: &git.WorktreeStatus{
				CommitsAhead:          2,
				IsMerged:              false,
				HasUncommittedChanges: true,
			},
			want:     "↑2 [in_progress, dirty]",
			wantBold: []string{"in_progress", "dirty"},
		},
		{
			name: "merged and dirty",
			status: &git.WorktreeStatus{
				IsMerged:              true,
				CommitsAhead:          0,
				HasUncommittedChanges: true,
			},
			want:       "[merged, dirty]",
			wantBold:   []string{"dirty"},
			wantNoBold: []string{"merged"},
		},
		{
			name: "no status - commits ahead but merged",
			status: &git.WorktreeStatus{
				CommitsAhead: 1,
				IsMerged:     true,
			},
			want: "↑1",
		},
		{
			name: "commits behind only",
			status: &git.WorktreeStatus{
				CommitsBehind: 5,
			},
			want: "↓5",
		},
		{
			name: "commits ahead and behind with in_progress",
			status: &git.WorktreeStatus{
				CommitsAhead:  2,
				CommitsBehind: 3,
				IsMerged:      false,
			},
			want:     "↑2 ↓3 [in_progress]",
			wantBold: []string{"in_progress"},
		},
		{
			name:   "empty status",
			status: &git.WorktreeStatus{},
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatCompactStatus(tt.status, LinkContext{})

			// Strip ANSI codes for content comparison
			stripped := stripANSI(got)
			if stripped != tt.want {
				t.Errorf("FormatCompactStatus() content = %q, want %q", stripped, tt.want)
			}

			// Check bold formatting
			for _, s := range tt.wantBold {
				if !containsBold(got, s) {
					t.Errorf("FormatCompactStatus() expected %q to be bold", s)
				}
			}
			for _, s := range tt.wantNoBold {
				if containsBold(got, s) {
					t.Errorf("FormatCompactStatus() expected %q to NOT be bold", s)
				}
			}
		})
	}
}

func TestFormatCompactStatusInProgressRequiresUnmerged(t *testing.T) {
	// Verify that in_progress only shows when commits are ahead AND not merged
	tests := []struct {
		name         string
		commitsAhead int
		isMerged     bool
		wantStatus   string
	}{
		{
			name:         "ahead and not merged = in_progress",
			commitsAhead: 1,
			isMerged:     false,
			wantStatus:   "in_progress",
		},
		{
			name:         "ahead but merged = no status tag",
			commitsAhead: 1,
			isMerged:     true,
			wantStatus:   "",
		},
		{
			name:         "not ahead and not merged = no status",
			commitsAhead: 0,
			isMerged:     false,
			wantStatus:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := &git.WorktreeStatus{
				CommitsAhead: tt.commitsAhead,
				IsMerged:     tt.isMerged,
			}
			got := stripANSI(FormatCompactStatus(status, LinkContext{}))

			if tt.wantStatus == "" {
				if strings.Contains(got, "[") {
					t.Errorf("expected no status tag, got %q", got)
				}
			} else {
				if !strings.Contains(got, tt.wantStatus) {
					t.Errorf("expected status to contain %q, got %q", tt.wantStatus, got)
				}
			}
		})
	}
}

func TestDirtyIsAdditive(t *testing.T) {
	// Verify dirty can appear with any other state
	states := []struct {
		name   string
		status *git.WorktreeStatus
	}{
		{
			name: "new",
			status: &git.WorktreeStatus{
				IsNew:                 true,
				HasUncommittedChanges: true,
			},
		},
		{
			name: "in_progress",
			status: &git.WorktreeStatus{
				CommitsAhead:          1,
				IsMerged:              false,
				HasUncommittedChanges: true,
			},
		},
		{
			name: "merged",
			status: &git.WorktreeStatus{
				IsMerged:              true,
				CommitsAhead:          0,
				HasUncommittedChanges: true,
			},
		},
	}

	for _, tt := range states {
		t.Run(tt.name+" with dirty", func(t *testing.T) {
			got := stripANSI(FormatCompactStatus(tt.status, LinkContext{}))

			// Should contain both the state and dirty
			if !strings.Contains(got, tt.name) {
				t.Errorf("expected %q in output, got %q", tt.name, got)
			}
			if !strings.Contains(got, "dirty") {
				t.Errorf("expected 'dirty' in output, got %q", got)
			}
			// Should be comma-separated inside brackets
			if !strings.Contains(got, ", ") {
				t.Errorf("expected comma-separated statuses, got %q", got)
			}
		})
	}
}

func TestFormatMergedStatus(t *testing.T) {
	tests := []struct {
		name  string
		prs   []string
		links LinkContext
		want  string
	}{
		{
			name: "no PRs",
			prs:  nil,
			want: "merged",
		},
		{
			name: "single PR without links",
			prs:  []string{"#42"},
			want: "merged in #42",
		},
		{
			name: "multiple PRs without links",
			prs:  []string{"#1", "#2"},
			want: "merged in #1, #2",
		},
		{
			name:  "single PR with links enabled",
			prs:   []string{"#42"},
			links: LinkContext{RepoURL: "https://github.com/owner/repo", IsTTY: true},
			want:  "merged in \033]8;;https://github.com/owner/repo/pull/42\033\\#42\033]8;;\033\\",
		},
		{
			name:  "multiple PRs with links enabled",
			prs:   []string{"#1", "#2"},
			links: LinkContext{RepoURL: "https://github.com/owner/repo", IsTTY: true},
			want:  "merged in \033]8;;https://github.com/owner/repo/pull/1\033\\#1\033]8;;\033\\, \033]8;;https://github.com/owner/repo/pull/2\033\\#2\033]8;;\033\\",
		},
		{
			name:  "repo URL but not a TTY - no links",
			prs:   []string{"#42"},
			links: LinkContext{RepoURL: "https://github.com/owner/repo", IsTTY: false},
			want:  "merged in #42",
		},
		{
			name:  "TTY but no repo URL - no links",
			prs:   []string{"#42"},
			links: LinkContext{RepoURL: "", IsTTY: true},
			want:  "merged in #42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatMergedStatus(tt.prs, tt.links)
			if got != tt.want {
				t.Errorf("FormatMergedStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatCompactStatusWithLinks(t *testing.T) {
	status := &git.WorktreeStatus{
		IsMerged:  true,
		MergedPRs: []string{"#99"},
	}
	links := LinkContext{RepoURL: "https://github.com/owner/repo", IsTTY: true}
	got := FormatCompactStatus(status, links)

	// Should contain OSC 8 hyperlink
	if !strings.Contains(got, "\033]8;;https://github.com/owner/repo/pull/99\033\\") {
		t.Errorf("expected OSC 8 link in output, got %q", got)
	}
	// Should still contain the visible text
	if !strings.Contains(got, "#99") {
		t.Errorf("expected #99 in output, got %q", got)
	}
}

func TestFormatCompactStatusNoLinksWhenNotTTY(t *testing.T) {
	status := &git.WorktreeStatus{
		IsMerged:  true,
		MergedPRs: []string{"#99"},
	}
	links := LinkContext{RepoURL: "https://github.com/owner/repo", IsTTY: false}
	got := FormatCompactStatus(status, links)

	// Should NOT contain OSC 8 sequences
	if strings.Contains(got, "\033]8;;") {
		t.Errorf("expected no OSC 8 link when not a TTY, got %q", got)
	}
	// Should contain plain text
	if !strings.Contains(got, "#99") {
		t.Errorf("expected #99 in output, got %q", got)
	}
}

// stripANSI removes ANSI escape codes from a string
func stripANSI(s string) string {
	result := s
	result = strings.ReplaceAll(result, bold, "")
	result = strings.ReplaceAll(result, reset, "")
	return result
}

// containsBold checks if a substring appears with bold formatting
func containsBold(s, substr string) bool {
	boldSubstr := bold + substr + reset
	return strings.Contains(s, boldSubstr)
}
