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
			got := FormatCompactStatus(tt.status)

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
			got := stripANSI(FormatCompactStatus(status))

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
			got := stripANSI(FormatCompactStatus(tt.status))

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
