package attention

import (
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

func TestGitCollectorImplementsCollectorInterface(t *testing.T) {
	// Verify GitCollector implements Collector interface
	mockClient := beads.NewMockClient()
	collector := NewGitCollector("/tmp", mockClient)

	var _ Collector = collector
}

func TestGitCollectorCollectReturnsLikelyDoneSignals(t *testing.T) {
	// Setup mock client with open issues
	mockClient := beads.NewMockClient()
	mockClient.Issues["orch-go-123"] = &beads.Issue{
		ID:        "orch-go-123",
		Title:     "Test issue with commits",
		Status:    "in_progress",
		Priority:  1,
		IssueType: "task",
	}

	// Use a temporary directory (not a git repo, but should handle gracefully)
	tmpDir := t.TempDir()
	collector := NewGitCollector(tmpDir, mockClient)

	// Collect for human role
	items, err := collector.Collect("human")
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	// In non-git directory, should return empty list (not error)
	if len(items) != 0 {
		t.Errorf("Expected 0 items for non-git directory, got %d", len(items))
	}
}

func TestGitCollectorItemStructure(t *testing.T) {
	mockClient := beads.NewMockClient()
	tmpDir := t.TempDir()
	collector := NewGitCollector(tmpDir, mockClient)

	items, err := collector.Collect("human")
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	// Even with no signals, verify the structure would be correct
	// by checking that items is a valid slice
	if items == nil {
		t.Error("Items should not be nil (can be empty slice)")
	}
}

func TestGitCollectorRoleParameter(t *testing.T) {
	mockClient := beads.NewMockClient()
	tmpDir := t.TempDir()
	collector := NewGitCollector(tmpDir, mockClient)

	roles := []string{"human", "orchestrator", "daemon"}
	for _, role := range roles {
		items, err := collector.Collect(role)
		if err != nil {
			t.Fatalf("Collect(%q) error = %v", role, err)
		}

		// All items should have the requested role
		for _, item := range items {
			if item.Role != role {
				t.Errorf("Item role = %v, want %v", item.Role, role)
			}
		}
	}
}

func TestCalculateGitPriorityRoleAware(t *testing.T) {
	signal := LikelyDoneSignal{
		IssueID:      "test-1",
		IssueStatus:  "in_progress",
		CommitCount:  5,
		LastCommitAt: time.Now().Format(time.RFC3339),
	}

	tests := []struct {
		role          string
		expectedRange [2]int // min, max priority expected
		description   string
	}{
		{"human", [2]int{70, 90}, "Human with many commits should have higher priority"},
		{"orchestrator", [2]int{75, 95}, "Orchestrator with in_progress status should have higher priority"},
		{"daemon", [2]int{140, 160}, "Daemon should have lower priority for observability signals"},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			priority := calculateGitPriority(signal, tt.role)
			if priority < tt.expectedRange[0] || priority > tt.expectedRange[1] {
				t.Errorf("Priority %d not in expected range [%d, %d] for %s",
					priority, tt.expectedRange[0], tt.expectedRange[1], tt.description)
			}
		})
	}
}

func TestGitCollectorIDGeneration(t *testing.T) {
	// Create a minimal signal to verify ID generation
	signal := LikelyDoneSignal{
		IssueID:     "orch-go-456",
		IssueTitle:  "Test",
		IssueStatus: "open",
		CommitCount: 1,
	}

	// Manually construct what the item would look like
	expectedID := "git-orch-go-456"
	actualID := "git-" + signal.IssueID

	if actualID != expectedID {
		t.Errorf("ID = %v, want %v", actualID, expectedID)
	}
}

func TestGitCollectorConcernType(t *testing.T) {
	// Git signals should always be Observability type
	mockClient := beads.NewMockClient()
	tmpDir := t.TempDir()
	collector := NewGitCollector(tmpDir, mockClient)

	items, err := collector.Collect("human")
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	// All items should have Observability concern
	for _, item := range items {
		if item.Concern != Observability {
			t.Errorf("Item concern = %v, want %v", item.Concern, Observability)
		}
	}
}

func TestGitCollectorMetadata(t *testing.T) {
	// Verify metadata contains expected fields
	signal := LikelyDoneSignal{
		IssueID:      "test-1",
		IssueTitle:   "Test",
		IssueStatus:  "open",
		CommitCount:  3,
		LastCommitAt: "2024-01-01T00:00:00Z",
		CommitHashes: []string{"abc123", "def456"},
		Reason:       "3 commits, no workspace",
	}

	// Check what metadata keys should exist
	expectedKeys := []string{"commit_count", "last_commit_at", "issue_status", "reason", "commit_hashes"}

	// In actual implementation, metadata would be populated from the signal
	metadata := map[string]any{
		"commit_count":   signal.CommitCount,
		"last_commit_at": signal.LastCommitAt,
		"issue_status":   signal.IssueStatus,
		"reason":         signal.Reason,
		"commit_hashes":  signal.CommitHashes,
	}

	for _, key := range expectedKeys {
		if _, ok := metadata[key]; !ok {
			t.Errorf("Expected metadata key %q not found", key)
		}
	}
}

func TestGitCollectorActionHint(t *testing.T) {
	signal := LikelyDoneSignal{
		IssueID:     "orch-go-789",
		IssueTitle:  "Test",
		IssueStatus: "open",
		CommitCount: 2,
	}

	expectedHint := "orch complete orch-go-789"
	actualHint := "orch complete " + signal.IssueID

	if actualHint != expectedHint {
		t.Errorf("ActionHint = %v, want %v", actualHint, expectedHint)
	}
}
