package attention

import (
	"encoding/json"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// MockBeadsClientForUnblocked implements beads.BeadsClient for testing UnblockedCollector.
type MockBeadsClientForUnblocked struct {
	issues []beads.Issue
	err    error
}

func (m *MockBeadsClientForUnblocked) List(args *beads.ListArgs) ([]beads.Issue, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.issues, nil
}

func (m *MockBeadsClientForUnblocked) Ready(args *beads.ReadyArgs) ([]beads.Issue, error) {
	return nil, nil
}
func (m *MockBeadsClientForUnblocked) Show(id string) (*beads.Issue, error) { return nil, nil }
func (m *MockBeadsClientForUnblocked) Stats() (*beads.Stats, error)         { return nil, nil }
func (m *MockBeadsClientForUnblocked) Comments(id string) ([]beads.Comment, error) {
	return nil, nil
}
func (m *MockBeadsClientForUnblocked) AddComment(id, author, text string) error { return nil }
func (m *MockBeadsClientForUnblocked) CloseIssue(id, reason string) error       { return nil }
func (m *MockBeadsClientForUnblocked) Create(args *beads.CreateArgs) (*beads.Issue, error) {
	return nil, nil
}
func (m *MockBeadsClientForUnblocked) Update(args *beads.UpdateArgs) (*beads.Issue, error) {
	return nil, nil
}
func (m *MockBeadsClientForUnblocked) AddLabel(id, label string) error    { return nil }
func (m *MockBeadsClientForUnblocked) RemoveLabel(id, label string) error { return nil }
func (m *MockBeadsClientForUnblocked) ResolveID(partialID string) (string, error) {
	return partialID, nil
}

func makeDepsJSON(deps []beads.Dependency) json.RawMessage {
	data, _ := json.Marshal(deps)
	return data
}

func TestUnblockedCollector_Collect(t *testing.T) {
	tests := []struct {
		name        string
		issues      []beads.Issue
		role        string
		expectCount int
		expectError bool
	}{
		{
			name: "finds unblocked issue with closed blocker",
			issues: []beads.Issue{
				{
					ID:        "test-1",
					Title:     "Blocked task",
					Status:    "open",
					Priority:  1,
					IssueType: "task",
					Dependencies: makeDepsJSON([]beads.Dependency{
						{
							ID:             "blocker-1",
							Title:          "Blocker issue",
							Status:         "closed", // Blocker resolved
							DependencyType: "blocks",
						},
					}),
				},
			},
			role:        "human",
			expectCount: 1,
		},
		{
			name: "skips issue still blocked",
			issues: []beads.Issue{
				{
					ID:        "test-1",
					Title:     "Blocked task",
					Status:    "open",
					Priority:  1,
					IssueType: "task",
					Dependencies: makeDepsJSON([]beads.Dependency{
						{
							ID:             "blocker-1",
							Title:          "Blocker issue",
							Status:         "open", // Still blocking
							DependencyType: "blocks",
						},
					}),
				},
			},
			role:        "human",
			expectCount: 0, // Still blocked
		},
		{
			name: "skips issue with no dependencies",
			issues: []beads.Issue{
				{
					ID:           "test-1",
					Title:        "Independent task",
					Status:       "open",
					Priority:     1,
					IssueType:    "task",
					Dependencies: nil, // No dependencies
				},
			},
			role:        "human",
			expectCount: 0, // Not blocked in the first place
		},
		{
			name: "handles question dependency answered",
			issues: []beads.Issue{
				{
					ID:        "test-1",
					Title:     "Waiting on question",
					Status:    "open",
					Priority:  1,
					IssueType: "task",
					Dependencies: makeDepsJSON([]beads.Dependency{
						{
							ID:             "q-1",
							Title:          "Architecture question",
							Status:         "answered", // Question answered
							DependencyType: "blocks",
						},
					}),
				},
			},
			role:        "human",
			expectCount: 1, // Question answered = unblocked
		},
		{
			name: "skips parent-child relationships",
			issues: []beads.Issue{
				{
					ID:        "epic-1.1",
					Title:     "Child task",
					Status:    "open",
					Priority:  1,
					IssueType: "task",
					Dependencies: makeDepsJSON([]beads.Dependency{
						{
							ID:             "epic-1",
							Title:          "Parent epic",
							Status:         "open", // Parent still open
							DependencyType: "parent-child",
						},
					}),
				},
			},
			role:        "human",
			expectCount: 0, // Parent-child never blocks
		},
		{
			name: "multiple deps, all resolved",
			issues: []beads.Issue{
				{
					ID:        "test-1",
					Title:     "Multi-dependency task",
					Status:    "open",
					Priority:  1,
					IssueType: "task",
					Dependencies: makeDepsJSON([]beads.Dependency{
						{
							ID:             "dep-1",
							Title:          "First blocker",
							Status:         "closed",
							DependencyType: "blocks",
						},
						{
							ID:             "dep-2",
							Title:          "Second blocker",
							Status:         "closed",
							DependencyType: "blocks",
						},
					}),
				},
			},
			role:        "human",
			expectCount: 1,
		},
		{
			name: "multiple deps, one still blocking",
			issues: []beads.Issue{
				{
					ID:        "test-1",
					Title:     "Multi-dependency task",
					Status:    "open",
					Priority:  1,
					IssueType: "task",
					Dependencies: makeDepsJSON([]beads.Dependency{
						{
							ID:             "dep-1",
							Title:          "First blocker",
							Status:         "closed",
							DependencyType: "blocks",
						},
						{
							ID:             "dep-2",
							Title:          "Second blocker",
							Status:         "open", // Still blocking
							DependencyType: "blocks",
						},
					}),
				},
			},
			role:        "human",
			expectCount: 0, // Still blocked by dep-2
		},
		{
			name:        "empty issues list",
			issues:      []beads.Issue{},
			role:        "daemon",
			expectCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &MockBeadsClientForUnblocked{issues: tt.issues}
			collector := NewUnblockedCollector(client)

			items, err := collector.Collect(tt.role)
			if (err != nil) != tt.expectError {
				t.Errorf("Collect() error = %v, expectError = %v", err, tt.expectError)
				return
			}

			if len(items) != tt.expectCount {
				t.Errorf("Collect() returned %d items, expected %d", len(items), tt.expectCount)
			}

			// Verify item properties for unblocked issue
			if tt.expectCount > 0 && len(items) > 0 {
				item := items[0]
				if item.Signal != "unblocked" {
					t.Errorf("item.Signal = %q, expected 'unblocked'", item.Signal)
				}
				if item.Source != "beads" {
					t.Errorf("item.Source = %q, expected 'beads'", item.Source)
				}
				if item.Concern != Actionability {
					t.Errorf("item.Concern = %v, expected Actionability", item.Concern)
				}
				if item.ActionHint == "" {
					t.Error("item.ActionHint should not be empty")
				}
			}
		})
	}
}

func TestFindResolvedDependencies(t *testing.T) {
	tests := []struct {
		name        string
		deps        []beads.Dependency
		expectCount int
	}{
		{
			name: "closed blocking dep",
			deps: []beads.Dependency{
				{ID: "dep-1", Status: "closed", DependencyType: "blocks"},
			},
			expectCount: 1,
		},
		{
			name: "open blocking dep",
			deps: []beads.Dependency{
				{ID: "dep-1", Status: "open", DependencyType: "blocks"},
			},
			expectCount: 0,
		},
		{
			name: "answered question",
			deps: []beads.Dependency{
				{ID: "q-1", Status: "answered", DependencyType: "blocks"},
			},
			expectCount: 1,
		},
		{
			name: "parent-child not counted",
			deps: []beads.Dependency{
				{ID: "parent-1", Status: "closed", DependencyType: "parent-child"},
			},
			expectCount: 0,
		},
		{
			name: "mixed deps",
			deps: []beads.Dependency{
				{ID: "dep-1", Status: "closed", DependencyType: "blocks"},
				{ID: "parent-1", Status: "open", DependencyType: "parent-child"},
				{ID: "q-1", Status: "answered", DependencyType: "blocks"},
			},
			expectCount: 2, // dep-1 and q-1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findResolvedDependencies(tt.deps)
			if len(result) != tt.expectCount {
				t.Errorf("findResolvedDependencies() returned %d, expected %d", len(result), tt.expectCount)
			}
		})
	}
}

func TestCalculateUnblockedPriority(t *testing.T) {
	issue := beads.Issue{
		ID:        "test-1",
		Priority:  1, // P1
		IssueType: "task",
	}

	tests := []struct {
		role           string
		expectPriority int
	}{
		{"human", 50},       // 40 base + 10 for P1
		{"orchestrator", 40}, // Lower priority number = higher priority
		{"daemon", 45},      // 40 base - 5 for daemon + 10 for P1
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			priority := calculateUnblockedPriority(issue, tt.role)
			if priority != tt.expectPriority {
				t.Errorf("calculateUnblockedPriority(%q) = %d, expected %d", tt.role, priority, tt.expectPriority)
			}
		})
	}
}
