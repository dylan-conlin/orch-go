package main

import (
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

func TestRecommendationType(t *testing.T) {
	tests := []struct {
		name string
		typ  RecommendationType
		want string
	}{
		{"blocker", RecommendationBlocker, "BLOCKER"},
		{"focus", RecommendationFocus, "FOCUS"},
		{"maintenance", RecommendationMaintenance, "MAINTENANCE"},
		{"backlog", RecommendationBacklog, "BACKLOG"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.typ) != tt.want {
				t.Errorf("RecommendationType = %s, want %s", tt.typ, tt.want)
			}
		})
	}
}

func TestGetTypeIcon(t *testing.T) {
	tests := []struct {
		typ  RecommendationType
		want string
	}{
		{RecommendationBlocker, "🚨"},
		{RecommendationFocus, "🎯"},
		{RecommendationMaintenance, "🔧"},
		{RecommendationBacklog, "📋"},
		{RecommendationType("unknown"), "•"},
	}

	for _, tt := range tests {
		t.Run(string(tt.typ), func(t *testing.T) {
			got := getTypeIcon(tt.typ)
			if got != tt.want {
				t.Errorf("getTypeIcon(%s) = %s, want %s", tt.typ, got, tt.want)
			}
		})
	}
}

func TestExtractKeywords(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"ship the snap MVP", []string{"ship", "snap", "MVP"}}, // Keeps original case
		{"fix authentication bugs", []string{"fix", "authentication", "bugs"}},
		{"a and the to for", []string{}}, // All stop words
		{"implement oauth flow", []string{"implement", "oauth", "flow"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractKeywords(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("extractKeywords(%q) returned %d keywords, want %d", tt.input, len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("extractKeywords(%q)[%d] = %s, want %s", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestMatchesFocusGoal(t *testing.T) {
	tests := []struct {
		name      string
		issue     beads.Issue
		focusGoal string
		want      bool
	}{
		{
			name:      "title matches keyword",
			issue:     beads.Issue{Title: "Implement OAuth login", Description: ""},
			focusGoal: "Ship OAuth feature",
			want:      true,
		},
		{
			name:      "description matches keyword",
			issue:     beads.Issue{Title: "Add button", Description: "Part of the OAuth implementation"},
			focusGoal: "Ship OAuth feature",
			want:      true,
		},
		{
			name:      "no match",
			issue:     beads.Issue{Title: "Fix typo", Description: "Minor correction"},
			focusGoal: "Ship OAuth feature",
			want:      false,
		},
		{
			name:      "case insensitive match",
			issue:     beads.Issue{Title: "OAUTH Integration", Description: ""},
			focusGoal: "oauth feature",
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesFocusGoal(tt.issue, tt.focusGoal)
			if got != tt.want {
				t.Errorf("matchesFocusGoal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInferSkillFromIssue(t *testing.T) {
	tests := []struct {
		issueType string
		want      string
	}{
		{"bug", "systematic-debugging"},
		{"feature", "feature-impl"},
		{"task", "feature-impl"},
		{"investigation", "investigation"},
		{"unknown", "feature-impl"},
		{"", "feature-impl"},
	}

	for _, tt := range tests {
		t.Run(tt.issueType, func(t *testing.T) {
			issue := beads.Issue{IssueType: tt.issueType}
			got := inferSkillFromIssue(issue)
			if got != tt.want {
				t.Errorf("inferSkillFromIssue(%s) = %s, want %s", tt.issueType, got, tt.want)
			}
		})
	}
}

func TestTruncateDescription(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"short", "hello", 10, "hello"},
		{"exact", "hello", 5, "hello"},
		{"truncate", "hello world", 8, "hello..."},
		{"with newlines", "hello\nworld", 20, "hello world"},
		{"empty", "", 10, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateDescription(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateDescription(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestSortRecommendations(t *testing.T) {
	recs := []Recommendation{
		{Type: RecommendationBacklog, Priority: 2, Title: "backlog1"},
		{Type: RecommendationBlocker, Priority: 1, Title: "blocker1"},
		{Type: RecommendationFocus, Priority: 1, Title: "focus1"},
		{Type: RecommendationMaintenance, Priority: 3, Title: "maint1"},
		{Type: RecommendationFocus, Priority: 2, Title: "focus2"},
	}

	sortRecommendations(recs)

	// Expected order: blocker, focus1 (P1), focus2 (P2), maintenance, backlog
	expected := []struct {
		typ   RecommendationType
		title string
	}{
		{RecommendationBlocker, "blocker1"},
		{RecommendationFocus, "focus1"},
		{RecommendationFocus, "focus2"},
		{RecommendationMaintenance, "maint1"},
		{RecommendationBacklog, "backlog1"},
	}

	for i, exp := range expected {
		if recs[i].Type != exp.typ || recs[i].Title != exp.title {
			t.Errorf("sortRecommendations()[%d] = {%s, %s}, want {%s, %s}",
				i, recs[i].Type, recs[i].Title, exp.typ, exp.title)
		}
	}
}

func TestNextOutputStructure(t *testing.T) {
	output := NextOutput{
		Focus:        "Ship snap MVP",
		FocusIssue:   "proj-123",
		TotalReady:   5,
		BlockerCount: 1,
		Recommendations: []Recommendation{
			{
				Type:        RecommendationBlocker,
				Priority:    1,
				BeadsID:     "proj-456",
				Title:       "Fix critical bug",
				Description: "This is blocking",
				Reason:      "Failed 3x",
				Command:     "orch spawn systematic-debugging --issue proj-456",
			},
		},
	}

	// Verify fields are set correctly
	if output.Focus != "Ship snap MVP" {
		t.Errorf("Focus = %s, want Ship snap MVP", output.Focus)
	}
	if output.BlockerCount != 1 {
		t.Errorf("BlockerCount = %d, want 1", output.BlockerCount)
	}
	if len(output.Recommendations) != 1 {
		t.Errorf("len(Recommendations) = %d, want 1", len(output.Recommendations))
	}
	if output.Recommendations[0].FocusMatch != false {
		t.Errorf("FocusMatch = true, want false")
	}
}

func TestRecommendationFields(t *testing.T) {
	rec := Recommendation{
		Type:        RecommendationFocus,
		Priority:    1,
		BeadsID:     "test-123",
		Title:       "Test Issue",
		Description: "A test description",
		Reason:      "Aligned with focus",
		Command:     "orch work test-123",
		FocusMatch:  true,
	}

	if rec.Type != RecommendationFocus {
		t.Errorf("Type = %s, want FOCUS", rec.Type)
	}
	if !rec.FocusMatch {
		t.Errorf("FocusMatch = false, want true")
	}
	if rec.Command == "" {
		t.Error("Command should not be empty")
	}
}
