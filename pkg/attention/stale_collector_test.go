package attention

import (
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

func TestStaleIssueCollector_Collect(t *testing.T) {
	now := time.Now()
	old := now.AddDate(0, 0, -45)    // 45 days ago
	recent := now.AddDate(0, 0, -5)  // 5 days ago
	veryOld := now.AddDate(0, -4, 0) // 4 months ago

	tests := []struct {
		name        string
		issues      map[string]*beads.Issue
		staleDays   int
		role        string
		expectCount int
	}{
		{
			name: "detects stale issues",
			issues: map[string]*beads.Issue{
				"issue-1": {
					ID:        "issue-1",
					Title:     "Old forgotten task",
					Status:    "open",
					UpdatedAt: old.Format(time.RFC3339),
				},
				"issue-2": {
					ID:        "issue-2",
					Title:     "Recent active task",
					Status:    "open",
					UpdatedAt: recent.Format(time.RFC3339),
				},
			},
			staleDays:   30,
			role:        "human",
			expectCount: 1,
		},
		{
			name: "handles multiple stale issues",
			issues: map[string]*beads.Issue{
				"issue-1": {
					ID:        "issue-1",
					Title:     "Old task one",
					Status:    "open",
					UpdatedAt: old.Format(time.RFC3339),
				},
				"issue-2": {
					ID:        "issue-2",
					Title:     "Very old task",
					Status:    "open",
					UpdatedAt: veryOld.Format(time.RFC3339),
				},
			},
			staleDays:   30,
			role:        "human",
			expectCount: 2,
		},
		{
			name:        "returns empty for no issues",
			issues:      map[string]*beads.Issue{},
			staleDays:   30,
			role:        "human",
			expectCount: 0,
		},
		{
			name: "only considers open issues",
			issues: map[string]*beads.Issue{
				"issue-1": {
					ID:        "issue-1",
					Title:     "Old closed issue",
					Status:    "closed",
					UpdatedAt: old.Format(time.RFC3339),
				},
			},
			staleDays:   30,
			role:        "human",
			expectCount: 0, // closed issues excluded by List filter
		},
		{
			name: "falls back to CreatedAt when UpdatedAt missing",
			issues: map[string]*beads.Issue{
				"issue-1": {
					ID:        "issue-1",
					Title:     "Never updated task",
					Status:    "open",
					CreatedAt: old.Format(time.RFC3339),
				},
			},
			staleDays:   30,
			role:        "human",
			expectCount: 1,
		},
		{
			name: "defaults staleDays to 30 when 0",
			issues: map[string]*beads.Issue{
				"issue-1": {
					ID:        "issue-1",
					Title:     "Old task",
					Status:    "open",
					UpdatedAt: old.Format(time.RFC3339),
				},
			},
			staleDays:   0,
			role:        "human",
			expectCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := beads.NewMockClient()
			for id, issue := range tt.issues {
				mock.Issues[id] = issue
			}

			collector := NewStaleIssueCollector(mock, tt.staleDays)
			items, err := collector.Collect(tt.role)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(items) != tt.expectCount {
				t.Errorf("expected %d items, got %d", tt.expectCount, len(items))
			}

			for _, item := range items {
				if item.Source != "beads" {
					t.Errorf("expected source 'beads', got '%s'", item.Source)
				}
				if item.Concern != Observability {
					t.Errorf("expected concern Observability, got %v", item.Concern)
				}
				if item.Signal != "stale" {
					t.Errorf("expected signal 'stale', got '%s'", item.Signal)
				}
				if item.Metadata["stale_days"] == nil {
					t.Error("expected stale_days in metadata")
				}
			}
		})
	}
}

func TestStalePriority(t *testing.T) {
	tests := []struct {
		name    string
		days    int
		role    string
		minPrio int
		maxPrio int
	}{
		{
			name:    "just stale (30-60 days), human",
			days:    35,
			role:    "human",
			minPrio: 170,
			maxPrio: 195,
		},
		{
			name:    "moderately stale (60-90 days), human",
			days:    75,
			role:    "human",
			minPrio: 150,
			maxPrio: 180,
		},
		{
			name:    "very stale (>90 days), human",
			days:    120,
			role:    "human",
			minPrio: 140,
			maxPrio: 165,
		},
		{
			name:    "daemon gets low priority",
			days:    60,
			role:    "daemon",
			minPrio: 270,
			maxPrio: 310,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prio := calculateStalePriority(tt.days, tt.role)
			if prio < tt.minPrio || prio > tt.maxPrio {
				t.Errorf("expected priority between %d and %d, got %d", tt.minPrio, tt.maxPrio, prio)
			}
		})
	}
}
