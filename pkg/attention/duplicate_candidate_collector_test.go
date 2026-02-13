package attention

import (
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

func TestTitleSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		minScore float64
		maxScore float64
	}{
		{
			name:     "identical titles",
			a:        "Add rate limiting to API",
			b:        "Add rate limiting to API",
			minScore: 0.99,
			maxScore: 1.01,
		},
		{
			name:     "very similar titles",
			a:        "Add rate limiting to API endpoints",
			b:        "Add rate limiting to the API",
			minScore: 0.5,
			maxScore: 1.0,
		},
		{
			name:     "completely different titles",
			a:        "Fix database connection pooling",
			b:        "Add dark mode toggle to settings",
			minScore: 0.0,
			maxScore: 0.2,
		},
		{
			name:     "stop words filtered out",
			a:        "the fix for the bug in the API",
			b:        "fix bug API",
			minScore: 0.9,
			maxScore: 1.01,
		},
		{
			name:     "empty title",
			a:        "",
			b:        "Some title",
			minScore: 0.0,
			maxScore: 0.01,
		},
		{
			name:     "partial overlap",
			a:        "Add stale issue detection to attention system",
			b:        "Add duplicate detection to attention system",
			minScore: 0.4,
			maxScore: 0.85,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := titleSimilarity(tt.a, tt.b)
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("expected score between %.2f and %.2f, got %.2f", tt.minScore, tt.maxScore, score)
			}
		})
	}
}

func TestSignificantWords(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		expected int // Minimum significant word count
	}{
		{
			name:     "filters stop words",
			title:    "the quick brown fox is a very fast runner",
			expected: 3, // quick, brown, fox, fast, runner (varies with exact stop list)
		},
		{
			name:     "removes short words",
			title:    "a b c fix bug",
			expected: 2, // fix, bug
		},
		{
			name:     "strips punctuation",
			title:    "Fix: bug (critical) [API]",
			expected: 3, // fix, bug, critical, api
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			words := significantWords(tt.title)
			if len(words) < tt.expected {
				t.Errorf("expected at least %d significant words, got %d: %v", tt.expected, len(words), words)
			}
		})
	}
}

func TestDuplicateCandidateCollector_Collect(t *testing.T) {
	tests := []struct {
		name        string
		issues      map[string]*beads.Issue
		threshold   float64
		expectCount int
	}{
		{
			name: "detects similar titles",
			issues: map[string]*beads.Issue{
				"issue-1": {
					ID:     "issue-1",
					Title:  "Add rate limiting to API endpoints",
					Status: "open",
				},
				"issue-2": {
					ID:     "issue-2",
					Title:  "Add rate limiting to the API",
					Status: "open",
				},
			},
			threshold:   0.5,
			expectCount: 1,
		},
		{
			name: "no duplicates for different titles",
			issues: map[string]*beads.Issue{
				"issue-1": {
					ID:     "issue-1",
					Title:  "Fix database connection pooling",
					Status: "open",
				},
				"issue-2": {
					ID:     "issue-2",
					Title:  "Add dark mode toggle to settings",
					Status: "open",
				},
			},
			threshold:   0.6,
			expectCount: 0,
		},
		{
			name: "needs at least 2 issues",
			issues: map[string]*beads.Issue{
				"issue-1": {
					ID:     "issue-1",
					Title:  "Solo issue",
					Status: "open",
				},
			},
			threshold:   0.6,
			expectCount: 0,
		},
		{
			name:        "handles empty issue list",
			issues:      map[string]*beads.Issue{},
			threshold:   0.6,
			expectCount: 0,
		},
		{
			name: "defaults threshold to 0.6 when invalid",
			issues: map[string]*beads.Issue{
				"issue-1": {
					ID:     "issue-1",
					Title:  "Add rate limiting to API",
					Status: "open",
				},
				"issue-2": {
					ID:     "issue-2",
					Title:  "Add rate limiting to API endpoints",
					Status: "open",
				},
			},
			threshold:   0, // Should default to 0.6
			expectCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := beads.NewMockClient()
			for id, issue := range tt.issues {
				mock.Issues[id] = issue
			}

			collector := NewDuplicateCandidateCollector(mock, tt.threshold)
			items, err := collector.Collect("human")

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(items) != tt.expectCount {
				t.Errorf("expected %d items, got %d", tt.expectCount, len(items))
			}

			for _, item := range items {
				if item.Signal != "duplicate-candidate" {
					t.Errorf("expected signal 'duplicate-candidate', got '%s'", item.Signal)
				}
				if item.Metadata["similar_to"] == nil {
					t.Error("expected similar_to in metadata")
				}
				if item.Metadata["score"] == nil {
					t.Error("expected score in metadata")
				}
			}
		})
	}
}

func TestDuplicatePriority(t *testing.T) {
	tests := []struct {
		name    string
		score   float64
		role    string
		minPrio int
		maxPrio int
	}{
		{
			name:    "very high similarity, human",
			score:   0.95,
			role:    "human",
			minPrio: 120,
			maxPrio: 145,
		},
		{
			name:    "moderate similarity, human",
			score:   0.7,
			role:    "human",
			minPrio: 140,
			maxPrio: 160,
		},
		{
			name:    "daemon gets low priority",
			score:   0.9,
			role:    "daemon",
			minPrio: 230,
			maxPrio: 290,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prio := calculateDuplicatePriority(tt.score, tt.role)
			if prio < tt.minPrio || prio > tt.maxPrio {
				t.Errorf("expected priority between %d and %d, got %d", tt.minPrio, tt.maxPrio, prio)
			}
		})
	}
}
