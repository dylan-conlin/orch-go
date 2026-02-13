package attention

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

func TestBeadsCollectorImplementsCollectorInterface(t *testing.T) {
	// Verify BeadsCollector implements Collector interface
	mockClient := beads.NewMockClient()
	collector := NewBeadsCollector(mockClient)

	var _ Collector = collector
}

func TestBeadsCollectorCollectReturnsReadyIssues(t *testing.T) {
	// Setup mock client with ready issues
	mockClient := beads.NewMockClient()
	mockClient.Issues["orch-go-123"] = &beads.Issue{
		ID:        "orch-go-123",
		Title:     "Test issue 1",
		Status:    "open",
		Priority:  1,
		IssueType: "task",
	}
	mockClient.Issues["orch-go-456"] = &beads.Issue{
		ID:        "orch-go-456",
		Title:     "Test issue 2",
		Status:    "open",
		Priority:  2,
		IssueType: "bug",
	}

	collector := NewBeadsCollector(mockClient)

	// Collect for human role
	items, err := collector.Collect("human")
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	// Should return 2 items
	if len(items) != 2 {
		t.Errorf("len(items) = %v, want %v", len(items), 2)
	}

	// Verify first item structure
	item := items[0]
	if item.Source != "beads" {
		t.Errorf("item.Source = %v, want %v", item.Source, "beads")
	}
	if item.Concern != Actionability {
		t.Errorf("item.Concern = %v, want %v", item.Concern, Actionability)
	}
	if item.Signal != "issue-ready" {
		t.Errorf("item.Signal = %v, want %v", item.Signal, "issue-ready")
	}
	if item.Subject != "orch-go-123" {
		t.Errorf("item.Subject = %v, want %v", item.Subject, "orch-go-123")
	}
	if item.Role != "human" {
		t.Errorf("item.Role = %v, want %v", item.Role, "human")
	}
	if item.ActionHint == "" {
		t.Error("item.ActionHint should not be empty")
	}

	// Verify timestamp is recent
	now := time.Now()
	if item.CollectedAt.After(now) {
		t.Errorf("item.CollectedAt is in the future")
	}
	if now.Sub(item.CollectedAt) > time.Minute {
		t.Errorf("item.CollectedAt is too old")
	}
}

func TestBeadsCollectorHandlesEmptyResults(t *testing.T) {
	mockClient := beads.NewMockClient()
	// Empty Issues map - no ready issues

	collector := NewBeadsCollector(mockClient)

	items, err := collector.Collect("human")
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	if len(items) != 0 {
		t.Errorf("len(items) = %v, want %v", len(items), 0)
	}
}

func TestBeadsCollectorPriorityMapping(t *testing.T) {
	mockClient := beads.NewMockClient()
	mockClient.Issues["orch-go-p0"] = &beads.Issue{
		ID:        "orch-go-p0",
		Title:     "P0 issue",
		Status:    "open",
		Priority:  0,
		IssueType: "task",
	}
	mockClient.Issues["orch-go-p1"] = &beads.Issue{
		ID:        "orch-go-p1",
		Title:     "P1 issue",
		Status:    "open",
		Priority:  1,
		IssueType: "task",
	}
	mockClient.Issues["orch-go-p2"] = &beads.Issue{
		ID:        "orch-go-p2",
		Title:     "P2 issue",
		Status:    "open",
		Priority:  2,
		IssueType: "task",
	}

	collector := NewBeadsCollector(mockClient)
	items, err := collector.Collect("human")
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	// Should return 3 items
	if len(items) != 3 {
		t.Fatalf("len(items) = %v, want %v", len(items), 3)
	}

	// Verify priority mapping (beads priority directly maps to attention priority)
	// Build a map for easier lookup since order is non-deterministic
	itemsBySubject := make(map[string]AttentionItem)
	for _, item := range items {
		itemsBySubject[item.Subject] = item
	}

	if item, ok := itemsBySubject["orch-go-p0"]; ok {
		if item.Priority != 0 {
			t.Errorf("P0 issue priority = %v, want %v", item.Priority, 0)
		}
	} else {
		t.Error("P0 issue not found in results")
	}

	if item, ok := itemsBySubject["orch-go-p1"]; ok {
		if item.Priority != 1 {
			t.Errorf("P1 issue priority = %v, want %v", item.Priority, 1)
		}
	} else {
		t.Error("P1 issue not found in results")
	}

	if item, ok := itemsBySubject["orch-go-p2"]; ok {
		if item.Priority != 2 {
			t.Errorf("P2 issue priority = %v, want %v", item.Priority, 2)
		}
	} else {
		t.Error("P2 issue not found in results")
	}
}

func TestBeadsCollectorMetadata(t *testing.T) {
	mockClient := beads.NewMockClient()
	mockClient.Issues["orch-go-123"] = &beads.Issue{
		ID:        "orch-go-123",
		Title:     "Test issue",
		Status:    "open",
		Priority:  1,
		IssueType: "bug",
	}

	collector := NewBeadsCollector(mockClient)
	items, err := collector.Collect("human")
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	// Verify metadata contains issue details
	item := items[0]
	if item.Metadata == nil {
		t.Fatal("item.Metadata should not be nil")
	}

	if item.Metadata["status"] != "open" {
		t.Errorf("Metadata[status] = %v, want %v", item.Metadata["status"], "open")
	}
	if item.Metadata["issue_type"] != "bug" {
		t.Errorf("Metadata[issue_type] = %v, want %v", item.Metadata["issue_type"], "bug")
	}
	if item.Metadata["beads_priority"] != 1 {
		t.Errorf("Metadata[beads_priority] = %v, want %v", item.Metadata["beads_priority"], 1)
	}
}

func TestBeadsCollectorErrorHandling(t *testing.T) {
	mockClient := beads.NewMockClient()
	// Inject error for Ready operation
	mockClient.Errors["ready"] = fmt.Errorf("connection failed")

	collector := NewBeadsCollector(mockClient)

	_, err := collector.Collect("human")
	if err == nil {
		t.Fatal("Collect() should return error when client.Ready fails")
	}

	// Verify error message contains context
	if !strings.Contains(err.Error(), "failed to query ready issues") {
		t.Errorf("Error message should contain context, got: %v", err)
	}
}

func TestBeadsCollectorRoleParameter(t *testing.T) {
	mockClient := beads.NewMockClient()
	mockClient.Issues["orch-go-123"] = &beads.Issue{
		ID:        "orch-go-123",
		Title:     "Test issue",
		Status:    "open",
		Priority:  1,
		IssueType: "task",
	}

	collector := NewBeadsCollector(mockClient)

	// Test with different roles
	roles := []string{"human", "orchestrator", "daemon"}
	for _, role := range roles {
		items, err := collector.Collect(role)
		if err != nil {
			t.Fatalf("Collect(%q) error = %v", role, err)
		}

		if len(items) != 1 {
			t.Fatalf("Collect(%q) returned %d items, want 1", role, len(items))
		}

		if items[0].Role != role {
			t.Errorf("Item role = %v, want %v", items[0].Role, role)
		}
	}
}

func TestBeadsCollectorIDGeneration(t *testing.T) {
	mockClient := beads.NewMockClient()
	mockClient.Issues["orch-go-123"] = &beads.Issue{
		ID:        "orch-go-123",
		Title:     "Test issue",
		Status:    "open",
		Priority:  1,
		IssueType: "task",
	}

	collector := NewBeadsCollector(mockClient)
	items, err := collector.Collect("human")
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	// Verify ID is prefixed with "beads-"
	item := items[0]
	expectedID := "beads-orch-go-123"
	if item.ID != expectedID {
		t.Errorf("Item ID = %v, want %v", item.ID, expectedID)
	}

	// Subject should be the original issue ID
	if item.Subject != "orch-go-123" {
		t.Errorf("Item Subject = %v, want %v", item.Subject, "orch-go-123")
	}
}

func TestBeadsCollectorActionHint(t *testing.T) {
	mockClient := beads.NewMockClient()
	mockClient.Issues["orch-go-456"] = &beads.Issue{
		ID:        "orch-go-456",
		Title:     "Test issue",
		Status:    "open",
		Priority:  1,
		IssueType: "task",
	}

	collector := NewBeadsCollector(mockClient)
	items, err := collector.Collect("human")
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	// Verify action hint
	item := items[0]
	expectedHint := "orch spawn orch-go-456"
	if item.ActionHint != expectedHint {
		t.Errorf("Item ActionHint = %v, want %v", item.ActionHint, expectedHint)
	}
}
