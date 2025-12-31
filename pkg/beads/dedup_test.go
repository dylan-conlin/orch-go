package beads

import (
	"testing"
)

func TestMockClient_FindByTitle(t *testing.T) {
	m := NewMockClient()

	// Create some issues
	m.Create(&CreateArgs{ID: "1", Title: "First Issue", IssueType: "task", Force: true})
	m.Create(&CreateArgs{ID: "2", Title: "Second Issue", IssueType: "task", Force: true})
	
	// Close one issue
	m.CloseIssue("2", "Done")

	// Find open issue
	found, err := m.FindByTitle("First Issue")
	if err != nil {
		t.Fatalf("FindByTitle failed: %v", err)
	}
	if found == nil {
		t.Fatal("Expected to find issue")
	}
	if found.ID != "1" {
		t.Errorf("Found ID = %q, want %q", found.ID, "1")
	}

	// Find non-existent issue
	notFound, err := m.FindByTitle("Non-existent Issue")
	if err != nil {
		t.Fatalf("FindByTitle failed: %v", err)
	}
	if notFound != nil {
		t.Error("Expected nil for non-existent issue")
	}

	// Find closed issue (should not find it)
	closedNotFound, err := m.FindByTitle("Second Issue")
	if err != nil {
		t.Fatalf("FindByTitle failed: %v", err)
	}
	if closedNotFound != nil {
		t.Error("Expected nil for closed issue")
	}
}

func TestMockClient_FindByTitle_InProgress(t *testing.T) {
	m := NewMockClient()

	// Create an issue and set to in_progress
	m.Create(&CreateArgs{ID: "1", Title: "In Progress Issue", IssueType: "task", Force: true})
	status := "in_progress"
	m.Update(&UpdateArgs{ID: "1", Status: &status})

	// Find in_progress issue
	found, err := m.FindByTitle("In Progress Issue")
	if err != nil {
		t.Fatalf("FindByTitle failed: %v", err)
	}
	if found == nil {
		t.Fatal("Expected to find in_progress issue")
	}
	if found.ID != "1" {
		t.Errorf("Found ID = %q, want %q", found.ID, "1")
	}
}

func TestMockClient_Create_Deduplication(t *testing.T) {
	m := NewMockClient()

	// Create first issue
	first, err := m.Create(&CreateArgs{
		ID:        "first-1",
		Title:     "Duplicate Test",
		IssueType: "task",
	})
	if err != nil {
		t.Fatalf("First create failed: %v", err)
	}
	if first.ID != "first-1" {
		t.Errorf("First ID = %q, want %q", first.ID, "first-1")
	}

	// Try to create duplicate (should return existing)
	second, err := m.Create(&CreateArgs{
		ID:        "second-1", // Different ID
		Title:     "Duplicate Test", // Same title
		IssueType: "bug",
	})
	if err != nil {
		t.Fatalf("Second create failed: %v", err)
	}
	// Should return the first issue, not create a new one
	if second.ID != "first-1" {
		t.Errorf("Second ID = %q, want %q (should be first issue)", second.ID, "first-1")
	}

	// Verify only one issue exists
	stats, _ := m.Stats()
	if stats.Summary.TotalIssues != 1 {
		t.Errorf("TotalIssues = %d, want %d", stats.Summary.TotalIssues, 1)
	}
}

func TestMockClient_Create_Force(t *testing.T) {
	m := NewMockClient()

	// Create first issue
	first, err := m.Create(&CreateArgs{
		ID:        "first-1",
		Title:     "Force Test",
		IssueType: "task",
	})
	if err != nil {
		t.Fatalf("First create failed: %v", err)
	}

	// Create duplicate with Force=true (should create new issue)
	second, err := m.Create(&CreateArgs{
		ID:        "second-1",
		Title:     "Force Test", // Same title
		IssueType: "bug",
		Force:     true, // Force creation
	})
	if err != nil {
		t.Fatalf("Second create failed: %v", err)
	}
	// Should create new issue with different ID
	if second.ID != "second-1" {
		t.Errorf("Second ID = %q, want %q", second.ID, "second-1")
	}
	if first.ID == second.ID {
		t.Error("Force=true should create new issue, not return existing")
	}

	// Verify two issues exist
	stats, _ := m.Stats()
	if stats.Summary.TotalIssues != 2 {
		t.Errorf("TotalIssues = %d, want %d", stats.Summary.TotalIssues, 2)
	}
}

func TestMockClient_Create_ClosedIssueNotDuplicate(t *testing.T) {
	m := NewMockClient()

	// Create and close an issue
	first, _ := m.Create(&CreateArgs{
		ID:        "first-1",
		Title:     "Closed Issue Test",
		IssueType: "task",
	})
	m.CloseIssue(first.ID, "Done")

	// Create issue with same title (should create new since old is closed)
	second, err := m.Create(&CreateArgs{
		ID:        "second-1",
		Title:     "Closed Issue Test", // Same title as closed issue
		IssueType: "task",
	})
	if err != nil {
		t.Fatalf("Second create failed: %v", err)
	}
	// Should create new issue since the original is closed
	if second.ID != "second-1" {
		t.Errorf("Second ID = %q, want %q", second.ID, "second-1")
	}

	// Verify two issues exist
	stats, _ := m.Stats()
	if stats.Summary.TotalIssues != 2 {
		t.Errorf("TotalIssues = %d, want %d", stats.Summary.TotalIssues, 2)
	}
}

func TestMockClient_Create_CaseSensitiveTitle(t *testing.T) {
	m := NewMockClient()

	// Create first issue
	first, _ := m.Create(&CreateArgs{
		ID:        "first-1",
		Title:     "Case Sensitive",
		IssueType: "task",
	})

	// Create issue with different case (should create new issue)
	second, err := m.Create(&CreateArgs{
		ID:        "second-1",
		Title:     "case sensitive", // Different case
		IssueType: "task",
	})
	if err != nil {
		t.Fatalf("Second create failed: %v", err)
	}
	// Should create new issue since titles are case-sensitive
	if second.ID == first.ID {
		t.Error("Different case titles should create separate issues")
	}

	// Verify two issues exist
	stats, _ := m.Stats()
	if stats.Summary.TotalIssues != 2 {
		t.Errorf("TotalIssues = %d, want %d", stats.Summary.TotalIssues, 2)
	}
}
