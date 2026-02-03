package attention

import (
	"fmt"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// mockBeadsClient is a mock implementation of beads.BeadsClient for testing.
type mockBeadsClientRecentlyClosed struct {
	issues []beads.Issue
	err    error
}

func (m *mockBeadsClientRecentlyClosed) List(args *beads.ListArgs) ([]beads.Issue, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.issues, nil
}

// Stub implementations for other BeadsClient methods (not used in tests)
func (m *mockBeadsClientRecentlyClosed) Ready(args *beads.ReadyArgs) ([]beads.Issue, error) {
	return nil, nil
}
func (m *mockBeadsClientRecentlyClosed) Show(id string) (*beads.Issue, error) { return nil, nil }
func (m *mockBeadsClientRecentlyClosed) Stats() (*beads.Stats, error)         { return nil, nil }
func (m *mockBeadsClientRecentlyClosed) Comments(id string) ([]beads.Comment, error) {
	return nil, nil
}
func (m *mockBeadsClientRecentlyClosed) AddComment(id, author, text string) error { return nil }
func (m *mockBeadsClientRecentlyClosed) CloseIssue(id, reason string) error       { return nil }
func (m *mockBeadsClientRecentlyClosed) Create(args *beads.CreateArgs) (*beads.Issue, error) {
	return nil, nil
}
func (m *mockBeadsClientRecentlyClosed) Update(args *beads.UpdateArgs) (*beads.Issue, error) {
	return nil, nil
}
func (m *mockBeadsClientRecentlyClosed) AddLabel(id, label string) error    { return nil }
func (m *mockBeadsClientRecentlyClosed) RemoveLabel(id, label string) error { return nil }
func (m *mockBeadsClientRecentlyClosed) ResolveID(partialID string) (string, error) {
	return "", nil
}

func TestRecentlyClosedCollector_Collect(t *testing.T) {
	tests := []struct {
		name          string
		issues        []beads.Issue
		lookbackHours int
		role          string
		expectCount   int
		expectError   bool
	}{
		{
			name: "collects recently closed issues",
			issues: []beads.Issue{
				{
					ID:          "test-1",
					Title:       "Test issue 1",
					Status:      "closed",
					ClosedAt:    time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
					IssueType:   "task",
					Priority:    1,
					CloseReason: "completed",
				},
				{
					ID:          "test-2",
					Title:       "Test issue 2",
					Status:      "closed",
					ClosedAt:    time.Now().Add(-6 * time.Hour).Format(time.RFC3339),
					IssueType:   "bug",
					Priority:    0,
					CloseReason: "fixed",
				},
			},
			lookbackHours: 24,
			role:          "human",
			expectCount:   2,
			expectError:   false,
		},
		{
			name:          "handles empty results",
			issues:        []beads.Issue{},
			lookbackHours: 24,
			role:          "human",
			expectCount:   0,
			expectError:   false,
		},
		{
			name:          "defaults to 24h when lookback is 0",
			issues:        []beads.Issue{},
			lookbackHours: 0,
			role:          "human",
			expectCount:   0,
			expectError:   false,
		},
		{
			name: "skips issues with invalid timestamps",
			issues: []beads.Issue{
				{
					ID:          "test-1",
					Title:       "Valid issue",
					Status:      "closed",
					ClosedAt:    time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
					IssueType:   "task",
					Priority:    1,
					CloseReason: "completed",
				},
				{
					ID:          "test-2",
					Title:       "Invalid timestamp",
					Status:      "closed",
					ClosedAt:    "invalid-timestamp",
					IssueType:   "bug",
					Priority:    0,
					CloseReason: "fixed",
				},
			},
			lookbackHours: 24,
			role:          "human",
			expectCount:   1, // Only valid issue should be collected
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := &mockBeadsClientRecentlyClosed{
				issues: tt.issues,
			}

			// Create collector
			collector := NewRecentlyClosedCollector(mockClient, tt.lookbackHours)

			// Collect items
			items, err := collector.Collect(tt.role)

			// Check error
			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Check count
			if len(items) != tt.expectCount {
				t.Errorf("expected %d items, got %d", tt.expectCount, len(items))
			}

			// Verify item structure for non-empty results
			if len(items) > 0 {
				item := items[0]
				if item.Source != "beads" {
					t.Errorf("expected source 'beads', got '%s'", item.Source)
				}
				if item.Concern != Observability {
					t.Errorf("expected concern Observability, got %v", item.Concern)
				}
				if item.Signal != "recently-closed" {
					t.Errorf("expected signal 'recently-closed', got '%s'", item.Signal)
				}
			}
		})
	}
}

func TestRecentlyClosedCollector_Priority(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		closedAt time.Time
		role     string
		minPrio  int // Minimum expected priority (lower = higher priority)
		maxPrio  int // Maximum expected priority
	}{
		{
			name:     "human - very recent (<1h)",
			closedAt: now.Add(-30 * time.Minute),
			role:     "human",
			minPrio:  100,
			maxPrio:  130, // Base 150 - 30 = 120
		},
		{
			name:     "human - recent (1-6h)",
			closedAt: now.Add(-3 * time.Hour),
			role:     "human",
			minPrio:  120,
			maxPrio:  140, // Base 150 - 20 = 130
		},
		{
			name:     "orchestrator - very recent (<2h)",
			closedAt: now.Add(-1 * time.Hour),
			role:     "orchestrator",
			minPrio:  100,
			maxPrio:  130, // Base 150 - 25 = 125
		},
		{
			name:     "daemon - any time (low priority)",
			closedAt: now.Add(-1 * time.Hour),
			role:     "daemon",
			minPrio:  240, // Base 150 + 100 = 250
			maxPrio:  260,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			priority := calculateRecentlyClosedPriority(tt.closedAt, tt.role)
			if priority < tt.minPrio || priority > tt.maxPrio {
				t.Errorf("expected priority between %d and %d, got %d", tt.minPrio, tt.maxPrio, priority)
			}
		})
	}
}

func TestFormatRelativeTime(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{
			name:     "just now",
			time:     now.Add(-30 * time.Second),
			expected: "just now",
		},
		{
			name:     "5 minutes ago",
			time:     now.Add(-5 * time.Minute),
			expected: "5m ago",
		},
		{
			name:     "2 hours ago",
			time:     now.Add(-2 * time.Hour),
			expected: "2h ago",
		},
		{
			name:     "1 day ago",
			time:     now.Add(-25 * time.Hour),
			expected: "1d ago",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatRelativeTime(tt.time)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestRecentlyClosedCollector_ErrorHandling(t *testing.T) {
	// Create mock client that returns error
	mockClient := &mockBeadsClientRecentlyClosed{
		err: fmt.Errorf("mock error"),
	}

	// Create collector
	collector := NewRecentlyClosedCollector(mockClient, 24)

	// Collect items - should return error
	_, err := collector.Collect("human")
	if err == nil {
		t.Errorf("expected error but got none")
	}
}
