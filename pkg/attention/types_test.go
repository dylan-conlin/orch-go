package attention

import (
	"testing"
	"time"
)

func TestConcernTypeString(t *testing.T) {
	tests := []struct {
		name     string
		concern  ConcernType
		expected string
	}{
		{
			name:     "Observability",
			concern:  Observability,
			expected: "Observability",
		},
		{
			name:     "Actionability",
			concern:  Actionability,
			expected: "Actionability",
		},
		{
			name:     "Authority",
			concern:  Authority,
			expected: "Authority",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.concern.String()
			if got != tt.expected {
				t.Errorf("ConcernType.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAttentionItemCreation(t *testing.T) {
	now := time.Now()
	metadata := map[string]any{
		"issue_id": "orch-go-123",
		"priority": 1,
	}

	item := AttentionItem{
		ID:          "test-item-1",
		Source:      "beads",
		Concern:     Actionability,
		Signal:      "issue-ready",
		Subject:     "orch-go-123",
		Summary:     "Issue ready for work",
		Priority:    1,
		Role:        "human",
		ActionHint:  "orch spawn orch-go-123",
		CollectedAt: now,
		Metadata:    metadata,
	}

	// Verify all fields are set correctly
	if item.ID != "test-item-1" {
		t.Errorf("ID = %v, want %v", item.ID, "test-item-1")
	}
	if item.Source != "beads" {
		t.Errorf("Source = %v, want %v", item.Source, "beads")
	}
	if item.Concern != Actionability {
		t.Errorf("Concern = %v, want %v", item.Concern, Actionability)
	}
	if item.Signal != "issue-ready" {
		t.Errorf("Signal = %v, want %v", item.Signal, "issue-ready")
	}
	if item.Subject != "orch-go-123" {
		t.Errorf("Subject = %v, want %v", item.Subject, "orch-go-123")
	}
	if item.Summary != "Issue ready for work" {
		t.Errorf("Summary = %v, want %v", item.Summary, "Issue ready for work")
	}
	if item.Priority != 1 {
		t.Errorf("Priority = %v, want %v", item.Priority, 1)
	}
	if item.Role != "human" {
		t.Errorf("Role = %v, want %v", item.Role, "human")
	}
	if item.ActionHint != "orch spawn orch-go-123" {
		t.Errorf("ActionHint = %v, want %v", item.ActionHint, "orch spawn orch-go-123")
	}
	if !item.CollectedAt.Equal(now) {
		t.Errorf("CollectedAt = %v, want %v", item.CollectedAt, now)
	}
	if item.Metadata["issue_id"] != "orch-go-123" {
		t.Errorf("Metadata[issue_id] = %v, want %v", item.Metadata["issue_id"], "orch-go-123")
	}
}

func TestCollectorInterface(t *testing.T) {
	// Test that a mock collector can implement the interface
	mockCollector := &MockCollector{
		items: []AttentionItem{
			{
				ID:          "mock-1",
				Source:      "test",
				Concern:     Observability,
				Signal:      "test-signal",
				Subject:     "test-subject",
				Summary:     "Test summary",
				Priority:    0,
				Role:        "human",
				ActionHint:  "test action",
				CollectedAt: time.Now(),
			},
		},
	}

	// Verify it implements Collector
	var _ Collector = mockCollector

	items, err := mockCollector.Collect("human")
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	if len(items) != 1 {
		t.Errorf("len(items) = %v, want %v", len(items), 1)
	}

	if items[0].ID != "mock-1" {
		t.Errorf("items[0].ID = %v, want %v", items[0].ID, "mock-1")
	}
}

// MockCollector implements the Collector interface for testing
type MockCollector struct {
	items []AttentionItem
	err   error
}

func (m *MockCollector) Collect(role string) ([]AttentionItem, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.items, nil
}
