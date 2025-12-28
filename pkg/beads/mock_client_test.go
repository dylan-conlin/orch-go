package beads

import (
	"fmt"
	"testing"
)

func TestNewMockClient(t *testing.T) {
	m := NewMockClient()
	if m.Issues == nil {
		t.Error("Issues should not be nil")
	}
	if m.CommentsStore == nil {
		t.Error("CommentsStore should not be nil")
	}
	if m.Errors == nil {
		t.Error("Errors should not be nil")
	}
}

func TestMockClient_Create(t *testing.T) {
	m := NewMockClient()

	args := &CreateArgs{
		Title:       "Test Issue",
		Description: "Test description",
		IssueType:   "task",
		Priority:    1,
		Labels:      []string{"bug"},
	}

	issue, err := m.Create(args)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if issue.Title != "Test Issue" {
		t.Errorf("Title = %q, want %q", issue.Title, "Test Issue")
	}
	if issue.Status != "open" {
		t.Errorf("Status = %q, want %q", issue.Status, "open")
	}
	if issue.ID == "" {
		t.Error("ID should not be empty")
	}

	// Verify it's stored
	stored, err := m.Show(issue.ID)
	if err != nil {
		t.Fatalf("Show failed: %v", err)
	}
	if stored.Title != "Test Issue" {
		t.Errorf("Stored Title = %q, want %q", stored.Title, "Test Issue")
	}
}

func TestMockClient_Comments(t *testing.T) {
	m := NewMockClient()

	// Create an issue first
	issue, _ := m.Create(&CreateArgs{Title: "Test", IssueType: "task"})

	// Add comments
	if err := m.AddComment(issue.ID, "agent", "Phase: Planning"); err != nil {
		t.Fatalf("AddComment failed: %v", err)
	}
	if err := m.AddComment(issue.ID, "agent", "Phase: Complete"); err != nil {
		t.Fatalf("AddComment failed: %v", err)
	}

	comments, err := m.Comments(issue.ID)
	if err != nil {
		t.Fatalf("Comments failed: %v", err)
	}

	if len(comments) != 2 {
		t.Errorf("Comments count = %d, want %d", len(comments), 2)
	}
	if comments[0].Text != "Phase: Planning" {
		t.Errorf("Comments[0].Text = %q, want %q", comments[0].Text, "Phase: Planning")
	}
	if comments[1].Text != "Phase: Complete" {
		t.Errorf("Comments[1].Text = %q, want %q", comments[1].Text, "Phase: Complete")
	}
}

func TestMockClient_CloseIssue(t *testing.T) {
	m := NewMockClient()

	issue, _ := m.Create(&CreateArgs{Title: "Test", IssueType: "task"})
	if issue.Status != "open" {
		t.Errorf("Initial status = %q, want %q", issue.Status, "open")
	}

	if err := m.CloseIssue(issue.ID, "Fixed"); err != nil {
		t.Fatalf("CloseIssue failed: %v", err)
	}

	closed, _ := m.Show(issue.ID)
	if closed.Status != "closed" {
		t.Errorf("Status = %q, want %q", closed.Status, "closed")
	}
	if closed.CloseReason != "Fixed" {
		t.Errorf("CloseReason = %q, want %q", closed.CloseReason, "Fixed")
	}
}

func TestMockClient_ErrorInjection(t *testing.T) {
	m := NewMockClient()

	// Inject a general error
	m.Errors["ready"] = fmt.Errorf("daemon unavailable")

	_, err := m.Ready(nil)
	if err == nil {
		t.Error("Expected error from Ready")
	}
	if err.Error() != "daemon unavailable" {
		t.Errorf("Error = %q, want %q", err.Error(), "daemon unavailable")
	}

	// Inject a specific error
	m.Create(&CreateArgs{ID: "issue-1", Title: "Test", IssueType: "task"})
	m.Errors["show:issue-1"] = fmt.Errorf("issue locked")

	_, err = m.Show("issue-1")
	if err == nil {
		t.Error("Expected error from Show")
	}
	if err.Error() != "issue locked" {
		t.Errorf("Error = %q, want %q", err.Error(), "issue locked")
	}
}

func TestMockClient_CallLog(t *testing.T) {
	m := NewMockClient()

	// Make some calls
	m.Ready(nil)
	m.Create(&CreateArgs{ID: "test-1", Title: "Test", IssueType: "task"})
	m.Show("test-1")
	m.AddComment("test-1", "agent", "Hello")

	// Verify call log
	if len(m.CallLog) != 4 {
		t.Errorf("CallLog length = %d, want %d", len(m.CallLog), 4)
	}

	// Get specific calls
	showCalls := m.GetCalls("Show")
	if len(showCalls) != 1 {
		t.Errorf("Show calls = %d, want %d", len(showCalls), 1)
	}
	if showCalls[0].Args[0] != "test-1" {
		t.Errorf("Show call arg = %v, want %q", showCalls[0].Args[0], "test-1")
	}
}

func TestMockClient_Update(t *testing.T) {
	m := NewMockClient()

	issue, _ := m.Create(&CreateArgs{
		ID:        "test-1",
		Title:     "Original Title",
		IssueType: "task",
		Labels:    []string{"bug"},
	})

	newTitle := "Updated Title"
	newStatus := "in_progress"
	updated, err := m.Update(&UpdateArgs{
		ID:          issue.ID,
		Title:       &newTitle,
		Status:      &newStatus,
		AddLabels:   []string{"urgent"},
		RemoveLabels: []string{"bug"},
	})

	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Title != "Updated Title" {
		t.Errorf("Title = %q, want %q", updated.Title, "Updated Title")
	}
	if updated.Status != "in_progress" {
		t.Errorf("Status = %q, want %q", updated.Status, "in_progress")
	}
	// Check labels
	hasUrgent := false
	hasBug := false
	for _, l := range updated.Labels {
		if l == "urgent" {
			hasUrgent = true
		}
		if l == "bug" {
			hasBug = true
		}
	}
	if !hasUrgent {
		t.Error("Should have 'urgent' label")
	}
	if hasBug {
		t.Error("Should not have 'bug' label")
	}
}

func TestMockClient_Stats(t *testing.T) {
	m := NewMockClient()

	// Create some issues
	m.Create(&CreateArgs{ID: "1", Title: "Open 1", IssueType: "task"})
	m.Create(&CreateArgs{ID: "2", Title: "Open 2", IssueType: "task"})
	m.CloseIssue("1", "Done")

	status := "in_progress"
	m.Create(&CreateArgs{ID: "3", Title: "In Progress", IssueType: "task"})
	m.Update(&UpdateArgs{ID: "3", Status: &status})

	stats, err := m.Stats()
	if err != nil {
		t.Fatalf("Stats failed: %v", err)
	}

	if stats.Summary.TotalIssues != 3 {
		t.Errorf("TotalIssues = %d, want %d", stats.Summary.TotalIssues, 3)
	}
	if stats.Summary.OpenIssues != 1 {
		t.Errorf("OpenIssues = %d, want %d", stats.Summary.OpenIssues, 1)
	}
	if stats.Summary.ClosedIssues != 1 {
		t.Errorf("ClosedIssues = %d, want %d", stats.Summary.ClosedIssues, 1)
	}
	if stats.Summary.InProgressIssues != 1 {
		t.Errorf("InProgressIssues = %d, want %d", stats.Summary.InProgressIssues, 1)
	}
}

func TestMockClient_Reset(t *testing.T) {
	m := NewMockClient()

	m.Create(&CreateArgs{ID: "1", Title: "Test", IssueType: "task"})
	m.AddComment("1", "agent", "Hello")
	m.Errors["ready"] = fmt.Errorf("error")

	if len(m.Issues) != 1 {
		t.Error("Should have 1 issue before reset")
	}

	m.Reset()

	if len(m.Issues) != 0 {
		t.Error("Should have 0 issues after reset")
	}
	if len(m.CommentsStore) != 0 {
		t.Error("Should have 0 comments after reset")
	}
	if len(m.Errors) != 0 {
		t.Error("Should have 0 errors after reset")
	}
	if len(m.CallLog) != 0 {
		t.Error("Should have 0 call log entries after reset")
	}
}

func TestMockClient_ResolveID(t *testing.T) {
	m := NewMockClient()

	m.Create(&CreateArgs{ID: "proj-abc123", Title: "Test", IssueType: "task"})

	// Exact match
	id, err := m.ResolveID("proj-abc123")
	if err != nil {
		t.Fatalf("ResolveID exact failed: %v", err)
	}
	if id != "proj-abc123" {
		t.Errorf("Resolved ID = %q, want %q", id, "proj-abc123")
	}

	// Prefix match
	id, err = m.ResolveID("proj-abc")
	if err != nil {
		t.Fatalf("ResolveID prefix failed: %v", err)
	}
	if id != "proj-abc123" {
		t.Errorf("Resolved ID = %q, want %q", id, "proj-abc123")
	}

	// Ambiguous match
	m.Create(&CreateArgs{ID: "proj-abc456", Title: "Test 2", IssueType: "task"})
	_, err = m.ResolveID("proj-abc")
	if err == nil {
		t.Error("Expected ambiguous error")
	}
}

func TestMockClient_Labels(t *testing.T) {
	m := NewMockClient()

	issue, _ := m.Create(&CreateArgs{ID: "1", Title: "Test", IssueType: "task", Labels: []string{"initial"}})

	// Add label
	if err := m.AddLabel(issue.ID, "new-label"); err != nil {
		t.Fatalf("AddLabel failed: %v", err)
	}

	updated, _ := m.Show(issue.ID)
	found := false
	for _, l := range updated.Labels {
		if l == "new-label" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Label 'new-label' should be present")
	}

	// Remove label
	if err := m.RemoveLabel(issue.ID, "initial"); err != nil {
		t.Fatalf("RemoveLabel failed: %v", err)
	}

	updated, _ = m.Show(issue.ID)
	for _, l := range updated.Labels {
		if l == "initial" {
			t.Error("Label 'initial' should be removed")
		}
	}
}

func TestMockClient_ImplementsBeadsClient(t *testing.T) {
	// This test verifies that MockClient implements the BeadsClient interface.
	var _ BeadsClient = (*MockClient)(nil)
	var _ BeadsClient = NewMockClient()
}
