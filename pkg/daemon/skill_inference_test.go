package daemon

import (
	"fmt"
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

		// Implementation skills → empty (resolve pipeline handles default)
		{"feature-impl", ""},
		{"issue-creation", ""},

		// Unknown skills → empty (resolve pipeline handles default)
		{"unknown-skill", ""},
		{"", ""},
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

// TestInferModelFromSkill_NoDefaultOverride verifies non-mapped skills return empty string
// to let the resolve pipeline respect user config default_model.
func TestInferModelFromSkill_NoDefaultOverride(t *testing.T) {
	got := InferModelFromSkill("feature-impl")
	if got != "" {
		t.Errorf("InferModelFromSkill(\"feature-impl\") = %q, want empty string (resolve pipeline handles default)", got)
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

func TestInferSkill(t *testing.T) {
	tests := []struct {
		issueType string
		wantSkill string
		wantErr   bool
	}{
		{"bug", "systematic-debugging", false},
		{"feature", "feature-impl", false},
		{"task", "feature-impl", false},
		{"investigation", "investigation", false},
		{"epic", "", true},
		{"unknown", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.issueType, func(t *testing.T) {
			got, err := InferSkill(tt.issueType)
			if (err != nil) != tt.wantErr {
				t.Errorf("InferSkill(%q) error = %v, wantErr %v", tt.issueType, err, tt.wantErr)
				return
			}
			if got != tt.wantSkill {
				t.Errorf("InferSkill(%q) = %q, want %q", tt.issueType, got, tt.wantSkill)
			}
		})
	}
}

func TestInferSkillFromLabels(t *testing.T) {
	tests := []struct {
		labels    []string
		wantSkill string
	}{
		{[]string{"skill:research"}, "research"},
		{[]string{"priority:P0", "skill:investigation"}, "investigation"},
		{[]string{"priority:P0", "triage:ready"}, ""},
		{[]string{}, ""},
		{nil, ""},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.labels), func(t *testing.T) {
			got := InferSkillFromLabels(tt.labels)
			if got != tt.wantSkill {
				t.Errorf("InferSkillFromLabels(%v) = %q, want %q", tt.labels, got, tt.wantSkill)
			}
		})
	}
}

func TestInferSkillFromTitle(t *testing.T) {
	tests := []struct {
		title     string
		wantSkill string
	}{
		// No colon prefix but first-word keyword detection
		{"Fix dashboard bug", "systematic-debugging"},
		{"Add synthesis feature", ""},

		// Architect prefix variations
		{"Architect: Design accretion gravity enforcement infrastructure", "architect"},
		{"architect: some design work", "architect"},
		{"ARCHITECT: Design system", "architect"},

		// Debug/Systematic-debugging prefix
		{"Debug: Fix spawn issue", "systematic-debugging"},
		{"debug: something broken", "systematic-debugging"},
		{"Fix: Broken test", "systematic-debugging"},
		{"Systematic-debugging: Issue with daemon", "systematic-debugging"},

		// Investigation prefix
		{"Investigation: How does X work", "investigation"},
		{"Investigate: Dashboard status", "investigation"},
		{"investigation: something to understand", "investigation"},

		// Research prefix
		{"Research: Best practices for auth", "research"},
		{"research: compare options", "research"},

		// Feature/Implementation prefix
		{"Feature: Add new dashboard", "feature-impl"},
		{"Implement: New API endpoint", "feature-impl"},
		{"feature-impl: Build something", "feature-impl"},

		// First-word keyword detection (no colon prefix)
		{"Investigate Claude Code --worktree flag for agent isolation", "investigation"},
		{"Design orchestrator diagnostic mode: time-limited read-only code access", "architect"},
		{"Explore caching options for daemon", "investigation"},
		{"Broken CI pipeline after upgrade", "systematic-debugging"},
		{"Debug the flaky test in spawn_test.go", "systematic-debugging"},

		// Edge cases
		{"", ""},
		{"No colon in title", ""},
		{"Unknown: Skill name", ""},
		{"Architect:", "architect"}, // No text after colon - still valid skill prefix
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			got := InferSkillFromTitle(tt.title)
			if got != tt.wantSkill {
				t.Errorf("InferSkillFromTitle(%q) = %q, want %q", tt.title, got, tt.wantSkill)
			}
		})
	}
}

func TestInferSkillFromIssue(t *testing.T) {
	tests := []struct {
		name      string
		issue     *Issue
		wantSkill string
		wantErr   bool
	}{
		{
			name:      "nil issue",
			issue:     nil,
			wantSkill: "",
			wantErr:   true,
		},
		{
			name:      "skill label takes priority",
			issue:     &Issue{Labels: []string{"skill:research"}, Title: "Some task", IssueType: "task"},
			wantSkill: "research",
			wantErr:   false,
		},

		{
			name:      "falls back to issue type",
			issue:     &Issue{Labels: []string{}, Title: "Fix the bug", IssueType: "bug"},
			wantSkill: "systematic-debugging",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := InferSkillFromIssue(tt.issue)
			if (err != nil) != tt.wantErr {
				t.Errorf("InferSkillFromIssue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantSkill {
				t.Errorf("InferSkillFromIssue() = %q, want %q", got, tt.wantSkill)
			}
		})
	}
}

func TestInferBrowserToolFromLabels(t *testing.T) {
	tests := []struct {
		labels     []string
		wantTool   string
	}{
		{[]string{"needs:playwright"}, "playwright-cli"},
		{[]string{"priority:P0", "needs:playwright"}, "playwright-cli"},
		{[]string{"triage:ready", "needs:playwright", "skill:feature-impl"}, "playwright-cli"},
		{[]string{"priority:P0", "triage:ready"}, ""},
		{[]string{"skill:research"}, ""},
		{[]string{}, ""},
		{nil, ""},
		// needs: label with unknown value should not return browser tool
		{[]string{"needs:unknown"}, ""},
		// Multiple needs labels - first matching one wins
		{[]string{"needs:playwright", "needs:browser"}, "playwright-cli"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.labels), func(t *testing.T) {
			got := InferBrowserToolFromLabels(tt.labels)
			if got != tt.wantTool {
				t.Errorf("InferBrowserToolFromLabels(%v) = %q, want %q", tt.labels, got, tt.wantTool)
			}
		})
	}
}

// TestInferSkillFromIssue_TitleKeywordDetection verifies that titles with
// investigation/architect/debug keywords (without colon prefix) get correct skills.
// Reproduces: orch-go-3wga (investigate→feature-impl), orch-go-cp52 (design→feature-impl).
func TestInferSkillFromIssue_TitleKeywordDetection(t *testing.T) {
	tests := []struct {
		name    string
		issue   *Issue
		want    string
		wantErr bool
	}{
		{
			name: "orch-go-3wga: investigate title → investigation (was feature-impl)",
			issue: &Issue{
				ID:        "orch-go-3wga",
				Title:     "Investigate Claude Code --worktree flag for agent isolation",
				IssueType: "task",
				Labels:    []string{},
			},
			want: "investigation",
		},
		{
			name: "orch-go-cp52: design title → architect (was feature-impl)",
			issue: &Issue{
				ID:        "orch-go-cp52",
				Title:     "Design orchestrator diagnostic mode: time-limited read-only code access",
				IssueType: "task",
				Labels:    []string{},
			},
			want: "architect",
		},
		{
			name: "explore title → investigation",
			issue: &Issue{
				ID:        "test-explore",
				Title:     "Explore alternative spawn backends",
				IssueType: "task",
				Labels:    []string{},
			},
			want: "investigation",
		},
		{
			name: "fix title → systematic-debugging",
			issue: &Issue{
				ID:        "test-fix",
				Title:     "Fix daemon skill inference bug",
				IssueType: "task",
				Labels:    []string{},
			},
			want: "systematic-debugging",
		},
		{
			name: "label still overrides title keyword",
			issue: &Issue{
				ID:        "test-label-override",
				Title:     "Investigate something",
				IssueType: "task",
				Labels:    []string{"skill:feature-impl"},
			},
			want: "feature-impl",
		},
		{
			name: "model inference: investigation gets opus",
			issue: &Issue{
				ID:        "test-model-inv",
				Title:     "Investigate spawn failures",
				IssueType: "task",
				Labels:    []string{},
			},
			want: "investigation", // InferModelFromSkill("investigation") → "opus"
		},
		{
			name: "model inference: architect gets opus",
			issue: &Issue{
				ID:        "test-model-arch",
				Title:     "Design new API contract",
				IssueType: "task",
				Labels:    []string{},
			},
			want: "architect", // InferModelFromSkill("architect") → "opus"
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
				t.Errorf("InferSkillFromIssue() = %q, want %q", got, tt.want)
			}

			// Verify model inference for skills that should get opus
			model := InferModelFromSkill(got)
			switch got {
			case "investigation", "architect", "systematic-debugging":
				if model != "opus" {
					t.Errorf("InferModelFromSkill(%q) = %q, want opus", got, model)
				}
			}
		})
	}
}

// TestOriginalBugReproduction verifies the fix for orch-go-4mu.
// Issue titled "Architect: Design accretion gravity enforcement infrastructure"
// was incorrectly inferred as "investigation" instead of "architect".
func TestOriginalBugReproduction(t *testing.T) {
	issue := &Issue{
		ID:          "orch-go-4mu",
		Title:       "Architect: Design accretion gravity enforcement infrastructure",
		Description: "",
		IssueType:   "task",
		Labels:      []string{},
	}

	got, err := InferSkillFromIssue(issue)
	if err != nil {
		t.Fatalf("InferSkillFromIssue() unexpected error: %v", err)
	}

	want := "architect"
	if got != want {
		t.Errorf("InferSkillFromIssue() = %q, want %q (bug reproduction failed - title prefix not detected)", got, want)
	}
}
