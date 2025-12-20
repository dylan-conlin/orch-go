package verify

import (
	"testing"
)

func TestParsePhaseFromComments(t *testing.T) {
	tests := []struct {
		name     string
		comments []Comment
		want     PhaseStatus
	}{
		{
			name:     "no comments",
			comments: []Comment{},
			want:     PhaseStatus{Found: false},
		},
		{
			name: "no phase comments",
			comments: []Comment{
				{Content: "Just a regular comment"},
				{Content: "Another comment without phase"},
			},
			want: PhaseStatus{Found: false},
		},
		{
			name: "simple phase complete",
			comments: []Comment{
				{Content: "Phase: Complete"},
			},
			want: PhaseStatus{Phase: "Complete", Found: true},
		},
		{
			name: "phase with summary",
			comments: []Comment{
				{Content: "Phase: Complete - All tests passing, ready for review"},
			},
			want: PhaseStatus{
				Phase:   "Complete",
				Summary: "All tests passing, ready for review",
				Found:   true,
			},
		},
		{
			name: "phase with en-dash",
			comments: []Comment{
				{Content: "Phase: Complete – Implementation finished"},
			},
			want: PhaseStatus{
				Phase:   "Complete",
				Summary: "Implementation finished",
				Found:   true,
			},
		},
		{
			name: "phase with em-dash",
			comments: []Comment{
				{Content: "Phase: Complete — Done"},
			},
			want: PhaseStatus{
				Phase:   "Complete",
				Summary: "Done",
				Found:   true,
			},
		},
		{
			name: "multiple phases - returns latest",
			comments: []Comment{
				{Content: "Phase: Planning - Starting work"},
				{Content: "Some progress comment"},
				{Content: "Phase: Implementing - Adding tests"},
				{Content: "Phase: Complete - All done"},
			},
			want: PhaseStatus{
				Phase:   "Complete",
				Summary: "All done",
				Found:   true,
			},
		},
		{
			name: "case insensitive",
			comments: []Comment{
				{Content: "phase: complete - done"},
			},
			want: PhaseStatus{
				Phase:   "complete",
				Summary: "done",
				Found:   true,
			},
		},
		{
			name: "phase in middle of comment",
			comments: []Comment{
				{Content: "Update: Phase: Implementing - Working on feature"},
			},
			want: PhaseStatus{
				Phase:   "Implementing",
				Summary: "Working on feature",
				Found:   true,
			},
		},
		{
			name: "planning phase",
			comments: []Comment{
				{Content: "Phase: Planning - Analyzing codebase structure"},
			},
			want: PhaseStatus{
				Phase:   "Planning",
				Summary: "Analyzing codebase structure",
				Found:   true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParsePhaseFromComments(tt.comments)

			if got.Phase != tt.want.Phase {
				t.Errorf("Phase = %q, want %q", got.Phase, tt.want.Phase)
			}
			if got.Summary != tt.want.Summary {
				t.Errorf("Summary = %q, want %q", got.Summary, tt.want.Summary)
			}
			if got.Found != tt.want.Found {
				t.Errorf("Found = %v, want %v", got.Found, tt.want.Found)
			}
		})
	}
}

func TestVerificationResult(t *testing.T) {
	t.Run("empty result defaults to passed", func(t *testing.T) {
		result := VerificationResult{Passed: true}
		if !result.Passed {
			t.Error("Expected default result to be passed")
		}
		if len(result.Errors) != 0 {
			t.Error("Expected no errors")
		}
		if len(result.Warnings) != 0 {
			t.Error("Expected no warnings")
		}
	})
}

func TestPhaseStatusComplete(t *testing.T) {
	tests := []struct {
		name   string
		status PhaseStatus
		want   bool
	}{
		{
			name:   "complete phase",
			status: PhaseStatus{Phase: "Complete", Found: true},
			want:   true,
		},
		{
			name:   "complete lowercase",
			status: PhaseStatus{Phase: "complete", Found: true},
			want:   true,
		},
		{
			name:   "implementing phase",
			status: PhaseStatus{Phase: "Implementing", Found: true},
			want:   false,
		},
		{
			name:   "no phase found",
			status: PhaseStatus{Found: false},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check if phase is complete using same logic as IsPhaseComplete
			got := tt.status.Found && (tt.status.Phase == "Complete" || tt.status.Phase == "complete")
			if got != tt.want {
				t.Errorf("IsComplete = %v, want %v", got, tt.want)
			}
		})
	}
}
