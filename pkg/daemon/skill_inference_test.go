package daemon

import (
	"testing"
)

// TestInferSkillFromDescription tests the new description-aware heuristics.
func TestInferSkillFromDescription(t *testing.T) {
	tests := []struct {
		name        string
		description string
		want        string
	}{
		// Investigation signals
		{
			name:        "audit keyword",
			description: "We need to audit the authentication flow",
			want:        "investigation",
		},
		{
			name:        "analyze keyword",
			description: "Let's analyze the performance bottleneck",
			want:        "investigation",
		},
		{
			name:        "investigate keyword",
			description: "Investigate why the daemon is consuming CPU",
			want:        "investigation",
		},
		{
			name:        "how does pattern",
			description: "How does the skill inference work?",
			want:        "investigation",
		},
		{
			name:        "understand keyword",
			description: "Need to understand the existing architecture",
			want:        "investigation",
		},

		// Research signals
		{
			name:        "compare keyword",
			description: "Compare different logging frameworks",
			want:        "research",
		},
		{
			name:        "evaluate keyword",
			description: "Evaluate options for database migration",
			want:        "research",
		},
		{
			name:        "research keyword",
			description: "Research best practices for error handling",
			want:        "research",
		},
		{
			name:        "best practice phrase",
			description: "What are the best practice for API design?",
			want:        "research",
		},

		// Debugging signals - with cause described (systematic-debugging)
		{
			name:        "error with stack trace",
			description: "Fix the error: 'null pointer exception' at line 42 in handler.go",
			want:        "systematic-debugging",
		},
		{
			name:        "crash with reproduction",
			description: "Server crashes when I send POST /api/users with empty body",
			want:        "systematic-debugging",
		},
		{
			name:        "failing test with details",
			description: "Test failing: expected 200, actual 500. Error: connection refused",
			want:        "systematic-debugging",
		},
		{
			name:        "broken with specific error",
			description: "Login is broken - returns 'invalid token' error even with valid JWT",
			want:        "systematic-debugging",
		},

		// Debugging signals - vague (returns empty, falls back to type-based)
		{
			name:        "vague fix request",
			description: "Fix the authentication issue",
			want:        "",
		},
		{
			name:        "vague broken report",
			description: "The dashboard is broken",
			want:        "",
		},
		{
			name:        "vague error",
			description: "There's an error in production",
			want:        "",
		},

		// No match - returns empty
		{
			name:        "feature request",
			description: "Add dark mode support to the UI",
			want:        "",
		},
		{
			name:        "empty description",
			description: "",
			want:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InferSkillFromDescription(tt.description)
			if got != tt.want {
				t.Errorf("InferSkillFromDescription() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestInferSkillFromIssue_DescriptionHeuristic tests the updated inference with description heuristics.
func TestInferSkillFromIssue_DescriptionHeuristic(t *testing.T) {
	tests := []struct {
		name    string
		issue   *Issue
		want    string
		wantErr bool
	}{
		// Description heuristic (used when no label/title match)
		{
			name: "description heuristic - investigation",
			issue: &Issue{
				ID:          "test-002",
				Title:       "Database performance",
				Description: "We need to analyze why the queries are slow",
				IssueType:   "task",
				Labels:      []string{},
			},
			want:    "investigation",
			wantErr: false,
		},
		{
			name: "description heuristic - research",
			issue: &Issue{
				ID:          "test-003",
				Title:       "Choose framework",
				Description: "Compare React vs Vue for our frontend",
				IssueType:   "task",
				Labels:      []string{},
			},
			want:    "research",
			wantErr: false,
		},
		{
			name: "description heuristic - systematic-debugging",
			issue: &Issue{
				ID:          "test-004",
				Title:       "Login broken",
				Description: "Fix the error: 'session expired' at line 100 in auth.go",
				IssueType:   "bug",
				Labels:      []string{},
			},
			want:    "systematic-debugging",
			wantErr: false,
		},

		// Explicit skill label still takes priority over description
		{
			name: "label overrides description heuristic",
			issue: &Issue{
				ID:          "test-005",
				Title:       "Some task",
				Description: "We need to analyze this issue", // Would infer "investigation"
				IssueType:   "task",
				Labels:      []string{"skill:research"}, // But label says "research"
			},
			want:    "research",
			wantErr: false,
		},

		// Description heuristic used before type-based fallback
		{
			name: "description overrides type inference",
			issue: &Issue{
				ID:          "test-006",
				Title:       "Database slow",
				Description: "Compare PostgreSQL vs MySQL performance", // Research signal
				IssueType:   "task",                                    // Would infer "feature-impl"
				Labels:      []string{},
			},
			want:    "research", // Description heuristic wins
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := InferSkillFromIssue(tt.issue)
			if (err != nil) != tt.wantErr {
				t.Errorf("InferSkillFromIssue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("InferSkillFromIssue() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestInferModelFromSkill tests skill-to-model mapping.
func TestInferModelFromSkill(t *testing.T) {
	tests := []struct {
		skill string
		want  string
	}{
		// Deep reasoning skills → opus
		{"systematic-debugging", "opus"},
		{"investigation", "opus"},
		{"architect", "opus"},
		{"codebase-audit", "opus"},
		{"research", "opus"},

		// Implementation skills → sonnet (default)
		{"feature-impl", "sonnet"},
		{"issue-creation", "sonnet"},

		// Unknown skills → sonnet (default)
		{"unknown-skill", "sonnet"},
		{"", "sonnet"},
	}

	for _, tt := range tests {
		t.Run(tt.skill, func(t *testing.T) {
			got := InferModelFromSkill(tt.skill)
			if got != tt.want {
				t.Errorf("InferModelFromSkill(%q) = %q, want %q", tt.skill, got, tt.want)
			}
		})
	}
}

// TestInferModelFromSkill_DefaultModel verifies the default model constant.
func TestInferModelFromSkill_DefaultModel(t *testing.T) {
	if DefaultSkillModel != "sonnet" {
		t.Errorf("DefaultSkillModel = %q, want %q", DefaultSkillModel, "sonnet")
	}
}

// TestSkillModelMapping_AllOpusSkillsCovered verifies all opus skills are in the mapping.
func TestSkillModelMapping_AllOpusSkillsCovered(t *testing.T) {
	opusSkills := []string{
		"systematic-debugging",
		"investigation",
		"architect",
		"codebase-audit",
		"research",
	}

	for _, skill := range opusSkills {
		model := InferModelFromSkill(skill)
		if model != "opus" {
			t.Errorf("Expected opus for skill %q, got %q", skill, model)
		}
	}
}
