package attention

import (
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

func TestExtractArea(t *testing.T) {
	tests := []struct {
		name     string
		labels   []string
		expected string
	}{
		{
			name:     "extracts area from labels",
			labels:   []string{"triage:ready", "area:dashboard", "effort:small"},
			expected: "dashboard",
		},
		{
			name:     "returns empty when no area label",
			labels:   []string{"triage:ready", "effort:small"},
			expected: "",
		},
		{
			name:     "handles empty labels",
			labels:   []string{},
			expected: "",
		},
		{
			name:     "handles nil labels",
			labels:   nil,
			expected: "",
		},
		{
			name:     "returns first area label",
			labels:   []string{"area:dashboard", "area:spawn"},
			expected: "dashboard",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractArea(tt.labels)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestCompetingCollector_Collect(t *testing.T) {
	tests := []struct {
		name        string
		issues      map[string]*beads.Issue
		threshold   float64
		expectCount int
	}{
		{
			name: "detects competing issues in same area",
			issues: map[string]*beads.Issue{
				"issue-1": {
					ID:     "issue-1",
					Title:  "Add stale detection to attention signals",
					Status: "open",
					Labels: []string{"area:dashboard"},
				},
				"issue-2": {
					ID:     "issue-2",
					Title:  "Add duplicate detection to attention signals",
					Status: "open",
					Labels: []string{"area:dashboard"},
				},
			},
			threshold:   0.4,
			expectCount: 1,
		},
		{
			name: "no competing when different areas",
			issues: map[string]*beads.Issue{
				"issue-1": {
					ID:     "issue-1",
					Title:  "Add rate limiting to API",
					Status: "open",
					Labels: []string{"area:cli"},
				},
				"issue-2": {
					ID:     "issue-2",
					Title:  "Add rate limiting to dashboard",
					Status: "open",
					Labels: []string{"area:dashboard"},
				},
			},
			threshold:   0.4,
			expectCount: 0,
		},
		{
			name: "skips unlabeled issues",
			issues: map[string]*beads.Issue{
				"issue-1": {
					ID:     "issue-1",
					Title:  "Add rate limiting",
					Status: "open",
				},
				"issue-2": {
					ID:     "issue-2",
					Title:  "Add rate limiting v2",
					Status: "open",
				},
			},
			threshold:   0.4,
			expectCount: 0,
		},
		{
			name: "no competing when titles are very different",
			issues: map[string]*beads.Issue{
				"issue-1": {
					ID:     "issue-1",
					Title:  "Fix database connection pooling",
					Status: "open",
					Labels: []string{"area:dashboard"},
				},
				"issue-2": {
					ID:     "issue-2",
					Title:  "Add dark mode toggle to settings",
					Status: "open",
					Labels: []string{"area:dashboard"},
				},
			},
			threshold:   0.4,
			expectCount: 0,
		},
		{
			name: "needs at least 2 issues in same area",
			issues: map[string]*beads.Issue{
				"issue-1": {
					ID:     "issue-1",
					Title:  "Solo dashboard issue",
					Status: "open",
					Labels: []string{"area:dashboard"},
				},
				"issue-2": {
					ID:     "issue-2",
					Title:  "Solo spawn issue",
					Status: "open",
					Labels: []string{"area:spawn"},
				},
			},
			threshold:   0.4,
			expectCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := beads.NewMockClient()
			for id, issue := range tt.issues {
				mock.Issues[id] = issue
			}

			collector := NewCompetingCollector(mock, tt.threshold)
			items, err := collector.Collect("human")

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(items) != tt.expectCount {
				t.Errorf("expected %d items, got %d", tt.expectCount, len(items))
			}

			for _, item := range items {
				if item.Signal != "competing" {
					t.Errorf("expected signal 'competing', got '%s'", item.Signal)
				}
				if item.Metadata["area"] == nil {
					t.Error("expected area in metadata")
				}
				if item.Metadata["competing_id"] == nil {
					t.Error("expected competing_id in metadata")
				}
			}
		})
	}
}

func TestCompetingPriority(t *testing.T) {
	tests := []struct {
		name    string
		score   float64
		role    string
		minPrio int
		maxPrio int
	}{
		{
			name:    "high similarity same area, human",
			score:   0.85,
			role:    "human",
			minPrio: 140,
			maxPrio: 165,
		},
		{
			name:    "orchestrator gets slightly higher priority",
			score:   0.7,
			role:    "orchestrator",
			minPrio: 155,
			maxPrio: 175,
		},
		{
			name:    "daemon gets low priority",
			score:   0.8,
			role:    "daemon",
			minPrio: 250,
			maxPrio: 310,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prio := calculateCompetingPriority(tt.score, tt.role)
			if prio < tt.minPrio || prio > tt.maxPrio {
				t.Errorf("expected priority between %d and %d, got %d", tt.minPrio, tt.maxPrio, prio)
			}
		})
	}
}
