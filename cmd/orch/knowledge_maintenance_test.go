package main

import (
	"strings"
	"testing"
)

func TestExtractCompletionKeywords(t *testing.T) {
	tests := []struct {
		name      string
		skill     string
		issueDesc string
		summary   string
		want      []string // expected keywords (subset check)
		wantMin   int      // minimum number of keywords
	}{
		{
			name:      "skill name contributes keywords",
			skill:     "feature-impl",
			issueDesc: "",
			summary:   "",
			want:      []string{"feature", "impl"},
			wantMin:   2,
		},
		{
			name:      "issue description contributes keywords",
			skill:     "",
			issueDesc: "Add knowledge maintenance step to orch complete flow",
			summary:   "",
			want:      []string{"knowledge", "maintenance", "complete"},
			wantMin:   3,
		},
		{
			name:      "summary contributes keywords",
			skill:     "",
			issueDesc: "",
			summary:   "Implemented dashboard filtering for agent status",
			want:      []string{"dashboard", "filtering", "agent", "status"},
			wantMin:   3,
		},
		{
			name:      "combined sources deduplicate",
			skill:     "feature-impl",
			issueDesc: "Implement feature for dashboard",
			summary:   "Feature implementation complete",
			want:      []string{"feature", "dashboard"},
			wantMin:   3,
		},
		{
			name:      "stop words filtered out",
			skill:     "",
			issueDesc: "the agent is not working on a task for the project",
			summary:   "",
			want:      []string{"agent", "working", "task", "project"},
			wantMin:   2,
		},
		{
			name:      "short words filtered out",
			skill:     "",
			issueDesc: "to be or not to be",
			summary:   "",
			want:      nil,
			wantMin:   0,
		},
		{
			name:      "empty inputs",
			skill:     "",
			issueDesc: "",
			summary:   "",
			want:      nil,
			wantMin:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCompletionKeywords(tt.skill, tt.issueDesc, tt.summary)

			if tt.wantMin > 0 && len(got) < tt.wantMin {
				t.Errorf("extractCompletionKeywords() returned %d keywords, want at least %d: %v", len(got), tt.wantMin, got)
			}

			for _, w := range tt.want {
				found := false
				for _, g := range got {
					if strings.EqualFold(g, w) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("extractCompletionKeywords() missing expected keyword %q in %v", w, got)
				}
			}
		})
	}
}

func TestFilterQuickEntries(t *testing.T) {
	entries := []QuickEntry{
		{
			ID:      "kb-001",
			Type:    "decision",
			Content: "Use JWT for authentication tokens",
			Reason:  "Better for distributed systems",
			Status:  "active",
		},
		{
			ID:      "kb-002",
			Type:    "constraint",
			Content: "Dashboard must use SSE for real-time updates",
			Reason:  "WebSocket too complex for current needs",
			Status:  "active",
		},
		{
			ID:      "kb-003",
			Type:    "decision",
			Content: "Spawn agents in headless mode by default",
			Reason:  "Tmux mode is opt-in for visual monitoring",
			Status:  "active",
		},
		{
			ID:      "kb-004",
			Type:    "attempt",
			Content: "Tried memory caching for agent status",
			Reason:  "Race condition with invalidation",
			Status:  "active",
		},
	}

	tests := []struct {
		name     string
		keywords []string
		maxCount int
		wantIDs  []string
	}{
		{
			name:     "matches on content",
			keywords: []string{"dashboard", "real-time"},
			maxCount: 10,
			wantIDs:  []string{"kb-002"},
		},
		{
			name:     "matches on reason",
			keywords: []string{"distributed"},
			maxCount: 10,
			wantIDs:  []string{"kb-001"},
		},
		{
			name:     "multiple matches",
			keywords: []string{"agent", "spawn"},
			maxCount: 10,
			wantIDs:  []string{"kb-003", "kb-004"},
		},
		{
			name:     "respects max count",
			keywords: []string{"agent", "spawn", "dashboard"},
			maxCount: 1,
			wantIDs:  nil, // just check count
		},
		{
			name:     "no matches",
			keywords: []string{"postgres", "migration"},
			maxCount: 10,
			wantIDs:  nil,
		},
		{
			name:     "empty keywords",
			keywords: nil,
			maxCount: 10,
			wantIDs:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterQuickEntries(entries, tt.keywords, tt.maxCount)

			if tt.maxCount > 0 && len(got) > tt.maxCount {
				t.Errorf("filterQuickEntries() returned %d entries, max is %d", len(got), tt.maxCount)
			}

			if tt.wantIDs != nil {
				if len(got) != len(tt.wantIDs) {
					t.Errorf("filterQuickEntries() returned %d entries, want %d", len(got), len(tt.wantIDs))
					return
				}
				for i, wantID := range tt.wantIDs {
					if got[i].ID != wantID {
						t.Errorf("filterQuickEntries()[%d].ID = %s, want %s", i, got[i].ID, wantID)
					}
				}
			}
		})
	}
}

func TestScoreEntry(t *testing.T) {
	entry := QuickEntry{
		ID:      "kb-001",
		Type:    "decision",
		Content: "Use headless spawn mode for dashboard agents",
		Reason:  "Better for batch processing and daemon automation",
	}

	tests := []struct {
		name     string
		keywords []string
		wantGt0  bool
	}{
		{
			name:     "single keyword match in content",
			keywords: []string{"dashboard"},
			wantGt0:  true,
		},
		{
			name:     "single keyword match in reason",
			keywords: []string{"automation"},
			wantGt0:  true,
		},
		{
			name:     "multiple keyword matches",
			keywords: []string{"headless", "spawn", "dashboard"},
			wantGt0:  true,
		},
		{
			name:     "no keyword matches",
			keywords: []string{"postgresql", "migration"},
			wantGt0:  false,
		},
		{
			name:     "empty keywords",
			keywords: nil,
			wantGt0:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := scoreEntry(entry, tt.keywords)
			if tt.wantGt0 && score <= 0 {
				t.Errorf("scoreEntry() = %d, want > 0", score)
			}
			if !tt.wantGt0 && score > 0 {
				t.Errorf("scoreEntry() = %d, want 0", score)
			}
		})
	}
}

func TestScoreEntryMultipleMatchesHigher(t *testing.T) {
	entry := QuickEntry{
		Content: "Use headless spawn for dashboard agent automation",
		Reason:  "Dashboard needs fast agent spawn",
	}

	score1 := scoreEntry(entry, []string{"dashboard"})
	score2 := scoreEntry(entry, []string{"dashboard", "agent", "spawn"})

	if score2 <= score1 {
		t.Errorf("Multiple keyword matches (%d) should score higher than single (%d)", score2, score1)
	}
}

func TestFormatEntryForReview(t *testing.T) {
	entry := QuickEntry{
		ID:      "kb-abc123",
		Type:    "decision",
		Content: "Use JWT for auth tokens",
		Reason:  "Better for distributed systems",
	}

	result := formatEntryForReview(entry)

	if !strings.Contains(result, "kb-abc123") {
		t.Error("formatEntryForReview() should contain entry ID")
	}
	if !strings.Contains(result, "decision") {
		t.Error("formatEntryForReview() should contain entry type")
	}
	if !strings.Contains(result, "Use JWT") {
		t.Error("formatEntryForReview() should contain entry content")
	}
	if !strings.Contains(result, "Better for distributed") {
		t.Error("formatEntryForReview() should contain entry reason")
	}
}

func TestParseKnowledgeAction(t *testing.T) {
	tests := []struct {
		input string
		want  KnowledgeAction
	}{
		{"p", ActionPromote},
		{"P", ActionPromote},
		{"promote", ActionPromote},
		{"o", ActionObsolete},
		{"O", ActionObsolete},
		{"obsolete", ActionObsolete},
		{"", ActionSkip},
		{"s", ActionSkip},
		{"S", ActionSkip},
		{"skip", ActionSkip},
		{"anything_else", ActionSkip},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseKnowledgeAction(tt.input)
			if got != tt.want {
				t.Errorf("parseKnowledgeAction(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
