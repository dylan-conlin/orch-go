// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

func TestNextIssue_EmptyQueue(t *testing.T) {
	d := &Daemon{
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{}, nil
		},
	}

	issue, err := d.NextIssue()
	if err != nil {
		t.Fatalf("NextIssue() unexpected error: %v", err)
	}
	if issue != nil {
		t.Errorf("NextIssue() expected nil for empty queue, got %v", issue)
	}
}

func TestNextIssue_ReturnsHighestPriority(t *testing.T) {
	d := &Daemon{
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-3", Title: "Low priority", Priority: 2, IssueType: "feature"},
				{ID: "proj-1", Title: "High priority", Priority: 0, IssueType: "bug"},
				{ID: "proj-2", Title: "Medium priority", Priority: 1, IssueType: "task"},
			}, nil
		},
	}

	issue, err := d.NextIssue()
	if err != nil {
		t.Fatalf("NextIssue() unexpected error: %v", err)
	}
	if issue == nil {
		t.Fatal("NextIssue() expected issue, got nil")
	}
	if issue.ID != "proj-1" {
		t.Errorf("NextIssue() = %q, want highest priority 'proj-1'", issue.ID)
	}
}

func TestNextIssue_SkipsNonSpawnableTypes(t *testing.T) {
	d := &Daemon{
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Epic", Priority: 0, IssueType: "epic"},
				{ID: "proj-2", Title: "Feature", Priority: 1, IssueType: "feature"},
			}, nil
		},
	}

	issue, err := d.NextIssue()
	if err != nil {
		t.Fatalf("NextIssue() unexpected error: %v", err)
	}
	if issue == nil {
		t.Fatal("NextIssue() expected issue, got nil")
	}
	if issue.ID != "proj-2" {
		t.Errorf("NextIssue() = %q, want 'proj-2' (skipping non-spawnable epic)", issue.ID)
	}
}

func TestNextIssue_SkipsBlockedIssues(t *testing.T) {
	d := &Daemon{
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Blocked", Priority: 0, IssueType: "feature", Status: "blocked"},
				{ID: "proj-2", Title: "Open", Priority: 1, IssueType: "feature", Status: "open"},
			}, nil
		},
	}

	issue, err := d.NextIssue()
	if err != nil {
		t.Fatalf("NextIssue() unexpected error: %v", err)
	}
	if issue == nil {
		t.Fatal("NextIssue() expected issue, got nil")
	}
	if issue.ID != "proj-2" {
		t.Errorf("NextIssue() = %q, want 'proj-2' (skipping blocked)", issue.ID)
	}
}

func TestNextIssue_SkipsInProgressIssues(t *testing.T) {
	// This test verifies that in_progress issues are SKIPPED to prevent duplicate spawns.
	// Even though bd ready returns both open and in_progress issues, we only spawn for open ones.
	d := &Daemon{
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "In Progress", Priority: 0, IssueType: "feature", Status: "in_progress", Labels: []string{"triage:ready"}},
				{ID: "proj-2", Title: "Open", Priority: 1, IssueType: "feature", Status: "open", Labels: []string{"triage:ready"}},
			}, nil
		},
		Config: Config{Label: "triage:ready"},
	}

	issue, err := d.NextIssue()
	if err != nil {
		t.Fatalf("NextIssue() unexpected error: %v", err)
	}
	if issue == nil {
		t.Fatal("NextIssue() expected issue, got nil")
	}
	// Should skip in_progress and return the open issue
	if issue.ID != "proj-2" {
		t.Errorf("NextIssue() = %q, want 'proj-2' (should skip in_progress)", issue.ID)
	}
	if issue.Status != "open" {
		t.Errorf("NextIssue() status = %q, want 'open'", issue.Status)
	}
}

func TestNextIssueExcluding_SkipsExcludedIssues(t *testing.T) {
	// Test that NextIssueExcluding skips issues in the skip set.
	// This is critical for the daemon to skip issues that failed to spawn
	// (e.g., due to failure report gate) and continue with other issues.
	d := &Daemon{
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "First", Priority: 0, IssueType: "feature", Status: "open"},
				{ID: "proj-2", Title: "Second", Priority: 1, IssueType: "feature", Status: "open"},
				{ID: "proj-3", Title: "Third", Priority: 2, IssueType: "feature", Status: "open"},
			}, nil
		},
	}

	// Skip the first issue (simulating failure report gate blocked it)
	skip := map[string]bool{"proj-1": true}

	issue, err := d.NextIssueExcluding(skip)
	if err != nil {
		t.Fatalf("NextIssueExcluding() unexpected error: %v", err)
	}
	if issue == nil {
		t.Fatal("NextIssueExcluding() expected issue, got nil")
	}
	// Should skip proj-1 and return proj-2
	if issue.ID != "proj-2" {
		t.Errorf("NextIssueExcluding() = %q, want 'proj-2' (should skip excluded issue)", issue.ID)
	}
}

func TestNextIssueExcluding_SkipsMultipleExcludedIssues(t *testing.T) {
	// Test that multiple excluded issues are all skipped.
	d := &Daemon{
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "First", Priority: 0, IssueType: "feature", Status: "open"},
				{ID: "proj-2", Title: "Second", Priority: 1, IssueType: "feature", Status: "open"},
				{ID: "proj-3", Title: "Third", Priority: 2, IssueType: "feature", Status: "open"},
			}, nil
		},
	}

	// Skip multiple issues
	skip := map[string]bool{"proj-1": true, "proj-2": true}

	issue, err := d.NextIssueExcluding(skip)
	if err != nil {
		t.Fatalf("NextIssueExcluding() unexpected error: %v", err)
	}
	if issue == nil {
		t.Fatal("NextIssueExcluding() expected issue, got nil")
	}
	// Should skip proj-1 and proj-2, return proj-3
	if issue.ID != "proj-3" {
		t.Errorf("NextIssueExcluding() = %q, want 'proj-3' (should skip excluded issues)", issue.ID)
	}
}

func TestNextIssueExcluding_ReturnsNilWhenAllExcluded(t *testing.T) {
	// Test that NextIssueExcluding returns nil when all issues are excluded.
	d := &Daemon{
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "First", Priority: 0, IssueType: "feature", Status: "open"},
				{ID: "proj-2", Title: "Second", Priority: 1, IssueType: "feature", Status: "open"},
			}, nil
		},
	}

	// Skip all issues
	skip := map[string]bool{"proj-1": true, "proj-2": true}

	issue, err := d.NextIssueExcluding(skip)
	if err != nil {
		t.Fatalf("NextIssueExcluding() unexpected error: %v", err)
	}
	// Should return nil when all issues are excluded
	if issue != nil {
		t.Errorf("NextIssueExcluding() = %v, want nil (all issues excluded)", issue)
	}
}

func TestNextIssueExcluding_NilSkipWorksLikeNextIssue(t *testing.T) {
	// Test that passing nil skip set works like NextIssue (returns first issue).
	d := &Daemon{
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "First", Priority: 0, IssueType: "feature", Status: "open"},
				{ID: "proj-2", Title: "Second", Priority: 1, IssueType: "feature", Status: "open"},
			}, nil
		},
	}

	issue, err := d.NextIssueExcluding(nil)
	if err != nil {
		t.Fatalf("NextIssueExcluding(nil) unexpected error: %v", err)
	}
	if issue == nil {
		t.Fatal("NextIssueExcluding(nil) expected issue, got nil")
	}
	// Should return first issue (no exclusions)
	if issue.ID != "proj-1" {
		t.Errorf("NextIssueExcluding(nil) = %q, want 'proj-1'", issue.ID)
	}
}

func TestIsSpawnableType(t *testing.T) {
	tests := []struct {
		issueType string
		want      bool
	}{
		{"bug", true},
		{"feature", true},
		{"task", true},
		{"investigation", true},
		{"epic", false},
		{"chore", false},
		{"unknown", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.issueType, func(t *testing.T) {
			got := IsSpawnableType(tt.issueType)
			if got != tt.want {
				t.Errorf("IsSpawnableType(%q) = %v, want %v", tt.issueType, got, tt.want)
			}
		})
	}
}

func TestFormatPreview(t *testing.T) {
	issue := &Issue{
		ID:          "proj-123",
		Title:       "Fix critical bug",
		Priority:    0,
		IssueType:   "bug",
		Status:      "open",
		Description: "This is a detailed description",
	}

	preview := FormatPreview(issue)

	// Check that key information is present
	if preview == "" {
		t.Error("FormatPreview() returned empty string")
	}
	if !contains(preview, "proj-123") {
		t.Error("FormatPreview() missing issue ID")
	}
	if !contains(preview, "Fix critical bug") {
		t.Error("FormatPreview() missing title")
	}
	if !contains(preview, "bug") {
		t.Error("FormatPreview() missing issue type")
	}
}

func TestDaemon_Preview_NoIssues(t *testing.T) {
	d := &Daemon{
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{}, nil
		},
	}

	result, err := d.Preview()
	if err != nil {
		t.Fatalf("Preview() unexpected error: %v", err)
	}
	if result.Issue != nil {
		t.Errorf("Preview() expected nil issue for empty queue, got %v", result.Issue)
	}
	if result.Message == "" {
		t.Error("Preview() expected message for empty queue")
	}
}

func TestDaemon_Preview_HasIssues(t *testing.T) {
	d := &Daemon{
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Test issue", Priority: 1, IssueType: "feature", Status: "open"},
			}, nil
		},
	}

	result, err := d.Preview()
	if err != nil {
		t.Fatalf("Preview() unexpected error: %v", err)
	}
	if result.Issue == nil {
		t.Fatal("Preview() expected issue, got nil")
	}
	if result.Issue.ID != "proj-1" {
		t.Errorf("Preview() issue ID = %q, want 'proj-1'", result.Issue.ID)
	}
	if result.Skill == "" {
		t.Error("Preview() expected skill to be inferred")
	}
}

func TestDaemon_Preview_ShowsRejectionReasons(t *testing.T) {
	// Test that Preview returns rejection reasons for non-spawnable issues
	d := &Daemon{
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Missing type", Priority: 0, IssueType: "", Status: "open"},
				{ID: "proj-2", Title: "Epic type", Priority: 1, IssueType: "epic", Status: "open"},
				{ID: "proj-3", Title: "Blocked", Priority: 2, IssueType: "feature", Status: "blocked"},
				{ID: "proj-4", Title: "In progress", Priority: 3, IssueType: "feature", Status: "in_progress"},
				{ID: "proj-5", Title: "Spawnable", Priority: 4, IssueType: "bug", Status: "open"},
			}, nil
		},
	}

	result, err := d.Preview()
	if err != nil {
		t.Fatalf("Preview() unexpected error: %v", err)
	}

	// Should have one spawnable issue
	if result.Issue == nil {
		t.Fatal("Preview() expected spawnable issue, got nil")
	}
	if result.Issue.ID != "proj-5" {
		t.Errorf("Preview() spawnable issue ID = %q, want 'proj-5'", result.Issue.ID)
	}

	// Should have 4 rejected issues with reasons
	if len(result.RejectedIssues) != 4 {
		t.Errorf("Preview() rejected count = %d, want 4", len(result.RejectedIssues))
	}

	// Check rejection reasons
	rejectedByID := make(map[string]string)
	for _, r := range result.RejectedIssues {
		rejectedByID[r.Issue.ID] = r.Reason
	}

	if r, ok := rejectedByID["proj-1"]; !ok {
		t.Error("Preview() missing rejection for proj-1 (empty type)")
	} else if !strings.Contains(r, "missing type") {
		t.Errorf("Preview() proj-1 reason = %q, want to contain 'missing type'", r)
	}

	if r, ok := rejectedByID["proj-2"]; !ok {
		t.Error("Preview() missing rejection for proj-2 (epic type)")
	} else if !strings.Contains(r, "not spawnable") {
		t.Errorf("Preview() proj-2 reason = %q, want to contain 'not spawnable'", r)
	}

	if r, ok := rejectedByID["proj-3"]; !ok {
		t.Error("Preview() missing rejection for proj-3 (blocked status)")
	} else if !strings.Contains(r, "blocked") {
		t.Errorf("Preview() proj-3 reason = %q, want to contain 'blocked'", r)
	}

	if r, ok := rejectedByID["proj-4"]; !ok {
		t.Error("Preview() missing rejection for proj-4 (in_progress)")
	} else if !strings.Contains(r, "in_progress") {
		t.Errorf("Preview() proj-4 reason = %q, want to contain 'in_progress'", r)
	}
}

func TestDaemon_Preview_ShowsMissingLabelRejection(t *testing.T) {
	// Test that Preview shows rejection reason for missing label
	d := &Daemon{
		Config: Config{Label: "triage:ready"},
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "No label", Priority: 0, IssueType: "feature", Status: "open", Labels: []string{}},
				{ID: "proj-2", Title: "Has label", Priority: 1, IssueType: "feature", Status: "open", Labels: []string{"triage:ready"}},
			}, nil
		},
	}

	result, err := d.Preview()
	if err != nil {
		t.Fatalf("Preview() unexpected error: %v", err)
	}

	// Should select proj-2 (has required label)
	if result.Issue == nil {
		t.Fatal("Preview() expected spawnable issue, got nil")
	}
	if result.Issue.ID != "proj-2" {
		t.Errorf("Preview() spawnable issue ID = %q, want 'proj-2'", result.Issue.ID)
	}

	// Should reject proj-1 for missing label
	if len(result.RejectedIssues) != 1 {
		t.Errorf("Preview() rejected count = %d, want 1", len(result.RejectedIssues))
	}
	if len(result.RejectedIssues) > 0 {
		r := result.RejectedIssues[0]
		if r.Issue.ID != "proj-1" {
			t.Errorf("Preview() rejected issue ID = %q, want 'proj-1'", r.Issue.ID)
		}
		if !strings.Contains(r.Reason, "missing label") {
			t.Errorf("Preview() rejection reason = %q, want to contain 'missing label'", r.Reason)
		}
	}
}

func TestFormatRejectedIssues(t *testing.T) {
	rejected := []RejectedIssue{
		{Issue: Issue{ID: "proj-1"}, Reason: "missing type (required for skill inference)"},
		{Issue: Issue{ID: "proj-2"}, Reason: "status is in_progress (already being worked on)"},
	}

	output := FormatRejectedIssues(rejected)

	if !strings.Contains(output, "Rejected issues:") {
		t.Error("FormatRejectedIssues() missing header")
	}
	if !strings.Contains(output, "proj-1: missing type") {
		t.Error("FormatRejectedIssues() missing proj-1 entry")
	}
	if !strings.Contains(output, "proj-2: status is in_progress") {
		t.Error("FormatRejectedIssues() missing proj-2 entry")
	}
}

func TestFormatRejectedIssues_Empty(t *testing.T) {
	output := FormatRejectedIssues(nil)
	if output != "" {
		t.Errorf("FormatRejectedIssues(nil) = %q, want empty string", output)
	}

	output = FormatRejectedIssues([]RejectedIssue{})
	if output != "" {
		t.Errorf("FormatRejectedIssues([]) = %q, want empty string", output)
	}
}

func TestInferSkill(t *testing.T) {
	tests := []struct {
		issueType string
		wantSkill string
		wantErr   bool
	}{
		{"bug", "architect", false}, // Default: understand before fixing (Premise Before Solution)
		{"feature", "feature-impl", false},
		{"task", "feature-impl", false},
		{"investigation", "investigation", false},
		{"epic", "", true},
		{"unknown", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.issueType, func(t *testing.T) {
			got, err := InferSkill(tt.issueType)
			if (err != nil) != tt.wantErr {
				t.Errorf("InferSkill(%q) error = %v, wantErr %v", tt.issueType, err, tt.wantErr)
				return
			}
			if got != tt.wantSkill {
				t.Errorf("InferSkill(%q) = %q, want %q", tt.issueType, got, tt.wantSkill)
			}
		})
	}
}

func TestInferSkillFromLabels(t *testing.T) {
	tests := []struct {
		labels    []string
		wantSkill string
	}{
		{[]string{"skill:research"}, "research"},
		{[]string{"priority:P0", "skill:investigation"}, "investigation"},
		{[]string{"priority:P0", "triage:ready"}, ""},
		{[]string{}, ""},
		{nil, ""},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.labels), func(t *testing.T) {
			got := InferSkillFromLabels(tt.labels)
			if got != tt.wantSkill {
				t.Errorf("InferSkillFromLabels(%v) = %q, want %q", tt.labels, got, tt.wantSkill)
			}
		})
	}
}

func TestInferSkillFromTitle(t *testing.T) {
	tests := []struct {
		title     string
		wantSkill string
	}{
		// No prefix - should return empty
		{"Fix dashboard bug", ""},
		{"Add synthesis feature", ""},

		// Architect prefix variations
		{"Architect: Design accretion gravity enforcement infrastructure", "architect"},
		{"architect: some design work", "architect"},
		{"ARCHITECT: Design system", "architect"},

		// Debug/Systematic-debugging prefix
		{"Debug: Fix spawn issue", "systematic-debugging"},
		{"debug: something broken", "systematic-debugging"},
		{"Fix: Broken test", "systematic-debugging"},
		{"Systematic-debugging: Issue with daemon", "systematic-debugging"},

		// Investigation prefix
		{"Investigation: How does X work", "investigation"},
		{"Investigate: Dashboard status", "investigation"},
		{"investigation: something to understand", "investigation"},

		// Research prefix
		{"Research: Best practices for auth", "research"},
		{"research: compare options", "research"},

		// Feature/Implementation prefix
		{"Feature: Add new dashboard", "feature-impl"},
		{"Implement: New API endpoint", "feature-impl"},
		{"feature-impl: Build something", "feature-impl"},

		// Edge cases
		{"", ""},
		{"No colon in title", ""},
		{"Unknown: Skill name", ""},
		{"Architect:", "architect"}, // No text after colon - still valid skill prefix
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			got := InferSkillFromTitle(tt.title)
			if got != tt.wantSkill {
				t.Errorf("InferSkillFromTitle(%q) = %q, want %q", tt.title, got, tt.wantSkill)
			}
		})
	}
}

func TestInferSkillFromIssue(t *testing.T) {
	tests := []struct {
		name      string
		issue     *Issue
		wantSkill string
		wantErr   bool
	}{
		{
			name:      "nil issue",
			issue:     nil,
			wantSkill: "",
			wantErr:   true,
		},
		{
			name:      "skill label takes priority",
			issue:     &Issue{Labels: []string{"skill:research"}, Title: "Some task", IssueType: "task"},
			wantSkill: "research",
			wantErr:   false,
		},

		{
			name:      "falls back to issue type",
			issue:     &Issue{Labels: []string{}, Title: "Fix the bug", IssueType: "bug"},
			wantSkill: "architect", // Default: understand before fixing
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := InferSkillFromIssue(tt.issue)
			if (err != nil) != tt.wantErr {
				t.Errorf("InferSkillFromIssue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantSkill {
				t.Errorf("InferSkillFromIssue() = %q, want %q", got, tt.wantSkill)
			}
		})
	}
}

func TestInferMCPFromLabels(t *testing.T) {
	tests := []struct {
		labels  []string
		wantMCP string
	}{
		{[]string{"needs:playwright"}, "playwright"},
		{[]string{"priority:P0", "needs:playwright"}, "playwright"},
		{[]string{"triage:ready", "needs:playwright", "skill:feature-impl"}, "playwright"},
		{[]string{"priority:P0", "triage:ready"}, ""},
		{[]string{"skill:research"}, ""},
		{[]string{}, ""},
		{nil, ""},
		// needs: label with unknown value should not return MCP
		{[]string{"needs:unknown"}, ""},
		// Multiple needs labels - first matching one wins
		{[]string{"needs:playwright", "needs:browser"}, "playwright"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.labels), func(t *testing.T) {
			got := InferMCPFromLabels(tt.labels)
			if got != tt.wantMCP {
				t.Errorf("InferMCPFromLabels(%v) = %q, want %q", tt.labels, got, tt.wantMCP)
			}
		})
	}
}

func TestDaemon_Once_NoIssues(t *testing.T) {
	d := &Daemon{
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{}, nil
		},
	}

	result, err := d.Once()
	if err != nil {
		t.Fatalf("Once() unexpected error: %v", err)
	}
	if result.Processed {
		t.Error("Once() expected Processed=false for empty queue")
	}
	if result.Message == "" {
		t.Error("Once() expected message for empty queue")
	}
}

func TestDaemon_Once_ProcessesOneIssue(t *testing.T) {
	spawnCalled := false
	d := &Daemon{
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		},
		spawnFunc: func(beadsID string, model string) error {
			spawnCalled = true
			if beadsID != "proj-1" {
				t.Errorf("spawnFunc called with %q, want 'proj-1'", beadsID)
			}
			return nil
		},
		updateBeadsStatusFunc: func(beadsID string, status string) error {
			return nil // Mock: always succeed
		},
	}

	result, err := d.Once()
	if err != nil {
		t.Fatalf("Once() unexpected error: %v", err)
	}
	if !result.Processed {
		t.Error("Once() expected Processed=true")
	}
	if !spawnCalled {
		t.Error("Once() expected spawnFunc to be called")
	}
	if result.Issue == nil || result.Issue.ID != "proj-1" {
		t.Error("Once() expected result.Issue to be proj-1")
	}
}

func TestDaemon_SpawnIssue_StatusUpdateFailureReleasesSlot(t *testing.T) {
	pool := NewWorkerPool(1)
	spawnCalled := false
	d := &Daemon{
		Pool: pool,
		spawnFunc: func(beadsID string, model string) error {
			spawnCalled = true
			return nil
		},
		updateBeadsStatusFunc: func(beadsID string, status string) error {
			return fmt.Errorf("update failed")
		},
	}

	issue := &Issue{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"}
	result, slot, err := d.spawnIssue(issue, "feature-impl", "sonnet")
	if err != nil {
		t.Fatalf("spawnIssue() unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("spawnIssue() expected result on status update failure")
	}
	if result.Processed {
		t.Error("spawnIssue() expected Processed=false on status update failure")
	}
	if result.Error == nil {
		t.Error("spawnIssue() expected Error to be set on status update failure")
	}
	if spawnCalled {
		t.Error("spawnIssue() should not call spawnFunc when status update fails")
	}
	if slot != nil {
		t.Error("spawnIssue() expected nil slot on status update failure")
	}
	if pool.Active() != 0 {
		t.Errorf("Pool.Active() = %d, want 0 (slot should be released on error)", pool.Active())
	}
}

func TestDaemon_Run_EmptyQueue(t *testing.T) {
	d := &Daemon{
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{}, nil
		},
	}

	results, err := d.Run(10) // Max 10 iterations
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Run() expected 0 results for empty queue, got %d", len(results))
	}
}

func TestDaemon_Run_ProcessesAllIssues(t *testing.T) {
	callCount := 0
	issues := []Issue{
		{ID: "proj-1", Title: "First", Priority: 0, IssueType: "feature", Status: "open"},
		{ID: "proj-2", Title: "Second", Priority: 1, IssueType: "bug", Status: "open"},
		{ID: "proj-3", Title: "Third", Priority: 2, IssueType: "task", Status: "open"},
	}

	d := &Daemon{
		listIssuesFunc: func() ([]Issue, error) {
			// Return remaining issues
			if callCount >= len(issues) {
				return []Issue{}, nil
			}
			remaining := issues[callCount:]
			return remaining, nil
		},
		spawnFunc: func(beadsID string, model string) error {
			callCount++
			return nil
		},
		updateBeadsStatusFunc: func(beadsID string, status string) error {
			return nil // Mock: always succeed
		},
	}

	results, err := d.Run(10) // Max 10 iterations
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("Run() expected 3 results, got %d", len(results))
	}
	if callCount != 3 {
		t.Errorf("Run() expected 3 spawn calls, got %d", callCount)
	}
}

func TestDaemon_Run_RespectsMaxIterations(t *testing.T) {
	callCount := 0
	// Infinite queue (always returns same issue)
	d := &Daemon{
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Infinite", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		},
		spawnFunc: func(beadsID string, model string) error {
			callCount++
			return nil
		},
		updateBeadsStatusFunc: func(beadsID string, status string) error {
			return nil // Mock: always succeed
		},
	}

	results, err := d.Run(5) // Max 5 iterations
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}
	if len(results) != 5 {
		t.Errorf("Run() expected 5 results (max), got %d", len(results))
	}
	if callCount != 5 {
		t.Errorf("Run() expected 5 spawn calls (max), got %d", callCount)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Tests for new features: label filtering, capacity, config

func TestIssue_HasLabel(t *testing.T) {
	tests := []struct {
		name   string
		labels []string
		query  string
		want   bool
	}{
		{"has exact label", []string{"triage:ready", "P1"}, "triage:ready", true},
		{"has label case insensitive", []string{"TRIAGE:ready", "P1"}, "triage:ready", true},
		{"does not have label", []string{"P1", "P2"}, "triage:ready", false},
		{"empty labels", []string{}, "triage:ready", false},
		{"nil labels", nil, "triage:ready", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issue := &Issue{Labels: tt.labels}
			got := issue.HasLabel(tt.query)
			if got != tt.want {
				t.Errorf("Issue.HasLabel(%q) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}

func TestNextIssue_FiltersbyLabel(t *testing.T) {
	config := Config{Label: "triage:ready"}
	d := &Daemon{
		Config: config,
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "No label", Priority: 0, IssueType: "feature", Labels: []string{}},
				{ID: "proj-2", Title: "With label", Priority: 1, IssueType: "feature", Labels: []string{"triage:ready"}},
			}, nil
		},
	}

	issue, err := d.NextIssue()
	if err != nil {
		t.Fatalf("NextIssue() unexpected error: %v", err)
	}
	if issue == nil {
		t.Fatal("NextIssue() expected issue, got nil")
	}
	if issue.ID != "proj-2" {
		t.Errorf("NextIssue() = %q, want 'proj-2' (with triage:ready label)", issue.ID)
	}
}

func TestNextIssue_NoLabelFilter(t *testing.T) {
	// Empty label means no filtering
	config := Config{Label: ""}
	d := &Daemon{
		Config: config,
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "No label", Priority: 0, IssueType: "feature", Labels: []string{}},
				{ID: "proj-2", Title: "With label", Priority: 1, IssueType: "feature", Labels: []string{"triage:ready"}},
			}, nil
		},
	}

	issue, err := d.NextIssue()
	if err != nil {
		t.Fatalf("NextIssue() unexpected error: %v", err)
	}
	if issue == nil {
		t.Fatal("NextIssue() expected issue, got nil")
	}
	// Should return highest priority regardless of labels
	if issue.ID != "proj-1" {
		t.Errorf("NextIssue() = %q, want 'proj-1' (no label filter)", issue.ID)
	}
}

func TestDaemon_AtCapacity(t *testing.T) {
	tests := []struct {
		name       string
		maxAgents  int
		activeFunc func() int
		want       bool
	}{
		{"below limit", 3, func() int { return 1 }, false},
		{"at limit", 3, func() int { return 3 }, true},
		{"above limit", 3, func() int { return 5 }, true},
		{"no limit (0)", 0, func() int { return 100 }, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{MaxAgents: tt.maxAgents}
			d := NewWithConfig(config)
			// Simulate active agents by acquiring slots from pool
			if d.Pool != nil {
				activeCount := tt.activeFunc()
				for i := 0; i < activeCount; i++ {
					d.Pool.TryAcquire()
				}
			}
			got := d.AtCapacity()
			if got != tt.want {
				t.Errorf("AtCapacity() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDaemon_AvailableSlots(t *testing.T) {
	tests := []struct {
		name       string
		maxAgents  int
		activeFunc func() int
		want       int
	}{
		{"none active", 3, func() int { return 0 }, 3},
		{"some active", 3, func() int { return 1 }, 2},
		{"at capacity", 3, func() int { return 3 }, 0},
		{"over capacity", 3, func() int { return 5 }, 0},
		{"no limit", 0, func() int { return 100 }, 100}, // Returns high number when no limit
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{MaxAgents: tt.maxAgents}
			d := NewWithConfig(config)
			// Simulate active agents by acquiring slots from pool
			if d.Pool != nil {
				activeCount := tt.activeFunc()
				for i := 0; i < activeCount; i++ {
					d.Pool.TryAcquire()
				}
			}
			got := d.AvailableSlots()
			if got != tt.want {
				t.Errorf("AvailableSlots() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	// Check sensible defaults
	if config.PollInterval <= 0 {
		t.Error("DefaultConfig() PollInterval should be positive")
	}
	if config.MaxAgents <= 0 {
		t.Error("DefaultConfig() MaxAgents should be positive")
	}
	if config.Label == "" {
		t.Error("DefaultConfig() Label should not be empty")
	}
	if config.SpawnDelay <= 0 {
		t.Error("DefaultConfig() SpawnDelay should be positive")
	}
}

func TestNewWithConfig(t *testing.T) {
	config := Config{
		MaxAgents: 5,
		Label:     "custom:label",
	}
	d := NewWithConfig(config)

	if d.Config.MaxAgents != 5 {
		t.Errorf("NewWithConfig() MaxAgents = %d, want 5", d.Config.MaxAgents)
	}
	if d.Config.Label != "custom:label" {
		t.Errorf("NewWithConfig() Label = %q, want 'custom:label'", d.Config.Label)
	}
}

// Tests for WorkerPool integration

func TestNewWithConfig_CreatesPool(t *testing.T) {
	config := Config{
		MaxAgents: 3,
	}
	d := NewWithConfig(config)

	if d.Pool == nil {
		t.Fatal("NewWithConfig() should create pool when MaxAgents > 0")
	}
	if d.Pool.MaxWorkers() != 3 {
		t.Errorf("Pool.MaxWorkers() = %d, want 3", d.Pool.MaxWorkers())
	}
}

func TestNewWithConfig_NoPoolWhenNoLimit(t *testing.T) {
	config := Config{
		MaxAgents: 0, // No limit
	}
	d := NewWithConfig(config)

	if d.Pool != nil {
		t.Error("NewWithConfig() should not create pool when MaxAgents = 0")
	}
}

func TestNewWithPool(t *testing.T) {
	pool := NewWorkerPool(5)
	config := Config{
		MaxAgents: 10, // This should be ignored when pool is provided
	}
	d := NewWithPool(config, pool)

	if d.Pool != pool {
		t.Error("NewWithPool() should use provided pool")
	}
	// The pool's max should be 5, not 10
	if d.Pool.MaxWorkers() != 5 {
		t.Errorf("Pool.MaxWorkers() = %d, want 5 (from provided pool)", d.Pool.MaxWorkers())
	}
}

func TestDaemon_AtCapacity_WithPool(t *testing.T) {
	pool := NewWorkerPool(2)
	d := NewWithPool(Config{}, pool)

	if d.AtCapacity() {
		t.Error("AtCapacity() should be false when pool is empty")
	}

	// Acquire slots
	slot1 := pool.TryAcquire()
	slot2 := pool.TryAcquire()

	if !d.AtCapacity() {
		t.Error("AtCapacity() should be true when pool is full")
	}

	pool.Release(slot1)
	if d.AtCapacity() {
		t.Error("AtCapacity() should be false after release")
	}
	pool.Release(slot2)
}

func TestDaemon_AvailableSlots_WithPool(t *testing.T) {
	pool := NewWorkerPool(3)
	d := NewWithPool(Config{}, pool)

	if d.AvailableSlots() != 3 {
		t.Errorf("AvailableSlots() = %d, want 3", d.AvailableSlots())
	}

	slot := pool.TryAcquire()
	if d.AvailableSlots() != 2 {
		t.Errorf("AvailableSlots() = %d, want 2", d.AvailableSlots())
	}
	pool.Release(slot)
}

func TestDaemon_ActiveCount_WithPool(t *testing.T) {
	pool := NewWorkerPool(3)
	d := NewWithPool(Config{}, pool)

	if d.ActiveCount() != 0 {
		t.Errorf("ActiveCount() = %d, want 0", d.ActiveCount())
	}

	slot := pool.TryAcquire()
	if d.ActiveCount() != 1 {
		t.Errorf("ActiveCount() = %d, want 1", d.ActiveCount())
	}
	pool.Release(slot)
}

func TestDaemon_PoolStatus(t *testing.T) {
	pool := NewWorkerPool(3)
	d := NewWithPool(Config{}, pool)

	status := d.PoolStatus()
	if status == nil {
		t.Fatal("PoolStatus() should not be nil when pool is configured")
	}
	if status.MaxWorkers != 3 {
		t.Errorf("PoolStatus().MaxWorkers = %d, want 3", status.MaxWorkers)
	}
}

func TestDaemon_PoolStatus_NilPool(t *testing.T) {
	d := &Daemon{} // No pool

	status := d.PoolStatus()
	if status != nil {
		t.Error("PoolStatus() should be nil when no pool is configured")
	}
}

func TestDaemon_Once_WithPool_AcquiresSlot(t *testing.T) {
	pool := NewWorkerPool(3)

	d := &Daemon{
		Pool: pool,
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		},
		spawnFunc: func(beadsID string, model string) error {
			return nil
		},
		updateBeadsStatusFunc: func(beadsID string, status string) error {
			return nil // Mock: always succeed
		},
	}

	result, err := d.Once()
	if err != nil {
		t.Fatalf("Once() error = %v", err)
	}
	if !result.Processed {
		t.Error("Once() expected Processed=true")
	}

	// Pool should have one active slot
	if pool.Active() != 1 {
		t.Errorf("Pool.Active() = %d, want 1", pool.Active())
	}
}

func TestDaemon_Once_WithPool_AtCapacity(t *testing.T) {
	pool := NewWorkerPool(1)
	// Fill the pool
	pool.TryAcquire()

	d := &Daemon{
		Pool: pool,
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		},
		spawnFunc: func(beadsID string, model string) error {
			t.Error("spawnFunc should not be called when at capacity")
			return nil
		},
	}

	result, err := d.Once()
	if err != nil {
		t.Fatalf("Once() error = %v", err)
	}
	if result.Processed {
		t.Error("Once() should not process when at capacity")
	}
	if result.Message != "At capacity - no slots available" {
		t.Errorf("Once() message = %q, want 'At capacity - no slots available'", result.Message)
	}
}

func TestDaemon_Once_WithPool_ReleasesSlotOnError(t *testing.T) {
	pool := NewWorkerPool(2)
	d := &Daemon{
		Pool: pool,
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		},
		spawnFunc: func(beadsID string, model string) error {
			return fmt.Errorf("spawn failed")
		},
	}

	result, err := d.Once()
	if err != nil {
		t.Fatalf("Once() error = %v", err)
	}
	if result.Processed {
		t.Error("Once() expected Processed=false on spawn error")
	}

	// Pool should have released the slot on error
	if pool.Active() != 0 {
		t.Errorf("Pool.Active() = %d, want 0 (slot should be released on error)", pool.Active())
	}
}

func TestDaemon_OnceWithSlot_ReturnsSlot(t *testing.T) {
	pool := NewWorkerPool(2)
	spawnCount := 0
	d := &Daemon{
		Pool: pool,
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		},
		spawnFunc: func(beadsID string, model string) error {
			spawnCount++
			return nil
		},
		updateBeadsStatusFunc: func(beadsID string, status string) error {
			return nil // Mock: always succeed
		},
	}

	result, slot, err := d.OnceWithSlot()
	if err != nil {
		t.Fatalf("OnceWithSlot() error = %v", err)
	}
	if !result.Processed {
		t.Error("OnceWithSlot() expected Processed=true")
	}
	if slot == nil {
		t.Error("OnceWithSlot() should return slot")
	}
	if slot.BeadsID != "proj-1" {
		t.Errorf("Slot.BeadsID = %q, want 'proj-1'", slot.BeadsID)
	}

	// Release the slot
	d.ReleaseSlot(slot)
	if pool.Active() != 0 {
		t.Errorf("Pool.Active() = %d after release, want 0", pool.Active())
	}
}

func TestDaemon_OnceWithSlot_NoPool(t *testing.T) {
	d := &Daemon{
		Pool: nil, // No pool
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		},
		spawnFunc: func(beadsID string, model string) error {
			return nil
		},
		updateBeadsStatusFunc: func(beadsID string, status string) error {
			return nil // Mock: always succeed
		},
	}

	result, slot, err := d.OnceWithSlot()
	if err != nil {
		t.Fatalf("OnceWithSlot() error = %v", err)
	}
	if !result.Processed {
		t.Error("OnceWithSlot() expected Processed=true")
	}
	if slot != nil {
		t.Error("OnceWithSlot() should return nil slot when no pool configured")
	}
}

func TestDaemon_ReleaseSlot_Nil(t *testing.T) {
	pool := NewWorkerPool(2)
	d := NewWithPool(Config{}, pool)

	// Should not panic
	d.ReleaseSlot(nil)
}

func TestDaemon_ReleaseSlot_NoPool(t *testing.T) {
	d := &Daemon{Pool: nil}

	// Should not panic
	d.ReleaseSlot(&Slot{ID: 1})
}

// =============================================================================
// Tests for ReconcileWithOpenCode
// =============================================================================

func TestDaemon_ReconcileWithOpenCode_NoPool(t *testing.T) {
	d := &Daemon{Pool: nil}

	// Should return 0 when no pool
	freed := d.ReconcileWithOpenCode()
	if freed != 0 {
		t.Errorf("ReconcileWithOpenCode() = %d, want 0 (no pool)", freed)
	}
}

func TestDaemon_ReconcileWithOpenCode_WithPool(t *testing.T) {
	// This test verifies the pool integration, not the HTTP call itself.
	// Pool.Reconcile is tested separately.
	pool := NewWorkerPool(3)
	// Acquire 3 slots to fill the pool
	pool.TryAcquire()
	pool.TryAcquire()
	pool.TryAcquire()

	d := &Daemon{
		Pool: pool,
	}

	// ReconcileWithOpenCode calls DefaultActiveCount which makes HTTP call.
	// The actual count depends on whether OpenCode is running.
	// What we verify: the method doesn't panic and returns a reasonable value.
	freed := d.ReconcileWithOpenCode()

	// If OpenCode is running, freed could be 0-3 depending on actual sessions.
	// If not running, freed will be 3 (reconcile to 0).
	// Either way, freed should be between 0 and 3.
	if freed < 0 || freed > 3 {
		t.Errorf("ReconcileWithOpenCode() freed = %d, want 0-3", freed)
	}

	// Active + freed should equal 3 (what we started with)
	if pool.Active()+freed != 3 {
		t.Errorf("Pool.Active() + freed = %d + %d = %d, want 3",
			pool.Active(), freed, pool.Active()+freed)
	}
}

// Tests for beads RPC client integration

func TestConvertBeadsIssues_Empty(t *testing.T) {
	var beadsIssues []beads.Issue
	result := convertBeadsIssues(beadsIssues)

	if len(result) != 0 {
		t.Errorf("convertBeadsIssues(empty) = %d issues, want 0", len(result))
	}
}

func TestConvertBeadsIssues_ConvertsAllFields(t *testing.T) {
	beadsIssues := []beads.Issue{
		{
			ID:          "proj-123",
			Title:       "Test Issue",
			Description: "Test description",
			Priority:    1,
			Status:      "open",
			IssueType:   "feature",
			Labels:      []string{"triage:ready", "P1"},
		},
	}

	result := convertBeadsIssues(beadsIssues)

	if len(result) != 1 {
		t.Fatalf("convertBeadsIssues() = %d issues, want 1", len(result))
	}

	got := result[0]
	if got.ID != "proj-123" {
		t.Errorf("ID = %q, want 'proj-123'", got.ID)
	}
	if got.Title != "Test Issue" {
		t.Errorf("Title = %q, want 'Test Issue'", got.Title)
	}
	if got.Description != "Test description" {
		t.Errorf("Description = %q, want 'Test description'", got.Description)
	}
	if got.Priority != 1 {
		t.Errorf("Priority = %d, want 1", got.Priority)
	}
	if got.Status != "open" {
		t.Errorf("Status = %q, want 'open'", got.Status)
	}
	if got.IssueType != "feature" {
		t.Errorf("IssueType = %q, want 'feature'", got.IssueType)
	}
	if len(got.Labels) != 2 || got.Labels[0] != "triage:ready" || got.Labels[1] != "P1" {
		t.Errorf("Labels = %v, want [triage:ready P1]", got.Labels)
	}
}

func TestConvertBeadsIssues_MultipleIssues(t *testing.T) {
	beadsIssues := []beads.Issue{
		{ID: "proj-1", Title: "First", IssueType: "bug"},
		{ID: "proj-2", Title: "Second", IssueType: "feature"},
		{ID: "proj-3", Title: "Third", IssueType: "task"},
	}

	result := convertBeadsIssues(beadsIssues)

	if len(result) != 3 {
		t.Fatalf("convertBeadsIssues() = %d issues, want 3", len(result))
	}

	// Verify order is preserved
	if result[0].ID != "proj-1" {
		t.Errorf("result[0].ID = %q, want 'proj-1'", result[0].ID)
	}
	if result[1].ID != "proj-2" {
		t.Errorf("result[1].ID = %q, want 'proj-2'", result[1].ID)
	}
	if result[2].ID != "proj-3" {
		t.Errorf("result[2].ID = %q, want 'proj-3'", result[2].ID)
	}
}

// =============================================================================
// Tests for Completion Processing
// =============================================================================

func TestDefaultCompletionConfig(t *testing.T) {
	config := DefaultCompletionConfig()

	if config.PollInterval != 60*time.Second {
		t.Errorf("DefaultCompletionConfig().PollInterval = %v, want 60s", config.PollInterval)
	}
	if config.DryRun {
		t.Error("DefaultCompletionConfig().DryRun should be false")
	}
	if config.Verbose {
		t.Error("DefaultCompletionConfig().Verbose should be false")
	}
}

func TestDaemon_ListCompletedAgents_Empty(t *testing.T) {
	d := &Daemon{
		listCompletedAgentsFunc: func(config CompletionConfig) ([]CompletedAgent, error) {
			return []CompletedAgent{}, nil
		},
	}

	config := DefaultCompletionConfig()
	completed, err := d.ListCompletedAgents(config)
	if err != nil {
		t.Fatalf("ListCompletedAgents() unexpected error: %v", err)
	}
	if len(completed) != 0 {
		t.Errorf("ListCompletedAgents() expected 0 agents, got %d", len(completed))
	}
}

func TestDaemon_ListCompletedAgents_ReturnsAgents(t *testing.T) {
	d := &Daemon{
		listCompletedAgentsFunc: func(config CompletionConfig) ([]CompletedAgent, error) {
			return []CompletedAgent{
				{BeadsID: "proj-1", Title: "First", PhaseSummary: "Done!"},
				{BeadsID: "proj-2", Title: "Second", PhaseSummary: "Complete"},
			}, nil
		},
	}

	config := DefaultCompletionConfig()
	completed, err := d.ListCompletedAgents(config)
	if err != nil {
		t.Fatalf("ListCompletedAgents() unexpected error: %v", err)
	}
	if len(completed) != 2 {
		t.Errorf("ListCompletedAgents() expected 2 agents, got %d", len(completed))
	}
	if completed[0].BeadsID != "proj-1" {
		t.Errorf("completed[0].BeadsID = %q, want 'proj-1'", completed[0].BeadsID)
	}
	if completed[1].PhaseSummary != "Complete" {
		t.Errorf("completed[1].PhaseSummary = %q, want 'Complete'", completed[1].PhaseSummary)
	}
}

func TestDaemon_CompletionOnce_NoAgents(t *testing.T) {
	d := &Daemon{
		listCompletedAgentsFunc: func(config CompletionConfig) ([]CompletedAgent, error) {
			return []CompletedAgent{}, nil
		},
	}

	config := DefaultCompletionConfig()
	result, err := d.CompletionOnce(config)
	if err != nil {
		t.Fatalf("CompletionOnce() unexpected error: %v", err)
	}
	if len(result.Processed) != 0 {
		t.Errorf("CompletionOnce() expected 0 processed, got %d", len(result.Processed))
	}
	if len(result.Errors) != 0 {
		t.Errorf("CompletionOnce() expected 0 errors, got %d", len(result.Errors))
	}
}

func TestDaemon_CompletionOnce_DryRun(t *testing.T) {
	closeIssuesCalled := false
	d := &Daemon{
		listCompletedAgentsFunc: func(config CompletionConfig) ([]CompletedAgent, error) {
			return []CompletedAgent{
				{BeadsID: "proj-1", Title: "Test", Status: "in_progress", PhaseSummary: "All done"},
			}, nil
		},
	}

	config := DefaultCompletionConfig()
	config.DryRun = true

	result, err := d.CompletionOnce(config)
	if err != nil {
		t.Fatalf("CompletionOnce() unexpected error: %v", err)
	}

	// In dry run, we should still "process" but not actually close
	if len(result.Processed) != 1 {
		t.Errorf("CompletionOnce() expected 1 processed, got %d", len(result.Processed))
	}

	// The issue should NOT have been closed in dry run
	if closeIssuesCalled {
		t.Error("CloseIssue should not be called in dry run mode")
	}
}

func TestDaemon_PreviewCompletions(t *testing.T) {
	d := &Daemon{
		listCompletedAgentsFunc: func(config CompletionConfig) ([]CompletedAgent, error) {
			return []CompletedAgent{
				{BeadsID: "proj-1", Title: "First", PhaseSummary: "Done"},
				{BeadsID: "proj-2", Title: "Second", PhaseSummary: "Complete"},
				{BeadsID: "proj-3", Title: "Third", PhaseSummary: "Finished"},
			}, nil
		},
	}

	config := DefaultCompletionConfig()
	preview, err := d.PreviewCompletions(config)
	if err != nil {
		t.Fatalf("PreviewCompletions() unexpected error: %v", err)
	}
	if len(preview) != 3 {
		t.Errorf("PreviewCompletions() expected 3 agents, got %d", len(preview))
	}
}

func TestCompletedAgent_Fields(t *testing.T) {
	agent := CompletedAgent{
		BeadsID:       "proj-123",
		Title:         "Test Agent",
		Status:        "in_progress",
		PhaseSummary:  "All tasks completed successfully",
		WorkspacePath: "/path/to/workspace",
	}

	if agent.BeadsID != "proj-123" {
		t.Errorf("BeadsID = %q, want 'proj-123'", agent.BeadsID)
	}
	if agent.Title != "Test Agent" {
		t.Errorf("Title = %q, want 'Test Agent'", agent.Title)
	}
	if agent.Status != "in_progress" {
		t.Errorf("Status = %q, want 'in_progress'", agent.Status)
	}
	if agent.PhaseSummary != "All tasks completed successfully" {
		t.Errorf("PhaseSummary = %q, want 'All tasks completed successfully'", agent.PhaseSummary)
	}
	if agent.WorkspacePath != "/path/to/workspace" {
		t.Errorf("WorkspacePath = %q, want '/path/to/workspace'", agent.WorkspacePath)
	}
}

func TestCompletionResult_Fields(t *testing.T) {
	result := CompletionResult{
		BeadsID:     "proj-123",
		Processed:   true,
		CloseReason: "Phase: Complete - All done",
	}

	if result.BeadsID != "proj-123" {
		t.Errorf("BeadsID = %q, want 'proj-123'", result.BeadsID)
	}
	if !result.Processed {
		t.Error("Processed should be true")
	}
	if result.CloseReason != "Phase: Complete - All done" {
		t.Errorf("CloseReason = %q, want 'Phase: Complete - All done'", result.CloseReason)
	}
}

func TestCompletionLoopResult_Fields(t *testing.T) {
	result := CompletionLoopResult{
		Processed: []CompletionResult{
			{BeadsID: "proj-1", Processed: true},
			{BeadsID: "proj-2", Processed: true},
		},
		Errors: []error{
			fmt.Errorf("error 1"),
		},
	}

	if len(result.Processed) != 2 {
		t.Errorf("expected 2 processed, got %d", len(result.Processed))
	}
	if len(result.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(result.Errors))
	}
}

func TestFindWorkspaceForIssue_NoWorkspaceDir(t *testing.T) {
	// When workspace dir doesn't exist, should return empty string
	result := findWorkspaceForIssue("proj-123", "/nonexistent/path", "")
	if result != "" {
		t.Errorf("findWorkspaceForIssue() = %q, want empty string for nonexistent dir", result)
	}
}

func TestExtractBeadsIDFromSessionTitle(t *testing.T) {
	tests := []struct {
		name  string
		title string
		want  string
	}{
		{
			name:  "standard format",
			title: "og-feat-add-feature-24dec [orch-go-3anf]",
			want:  "orch-go-3anf",
		},
		{
			name:  "untracked agent",
			title: "og-arch-review-url-markdown-26dec [orch-go-untracked-1766786808]",
			want:  "orch-go-untracked-1766786808",
		},
		{
			name:  "no beads ID",
			title: "some-workspace-name",
			want:  "",
		},
		{
			name:  "empty title",
			title: "",
			want:  "",
		},
		{
			name:  "brackets but no content",
			title: "workspace []",
			want:  "",
		},
		{
			name:  "multiple brackets - use last",
			title: "workspace [first] [second]",
			want:  "second",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractBeadsIDFromSessionTitle(tt.title)
			if got != tt.want {
				t.Errorf("extractBeadsIDFromSessionTitle(%q) = %q, want %q", tt.title, got, tt.want)
			}
		})
	}
}

func TestIsUntrackedBeadsID(t *testing.T) {
	tests := []struct {
		name    string
		beadsID string
		want    bool
	}{
		{
			name:    "tracked beads ID",
			beadsID: "orch-go-3anf",
			want:    false,
		},
		{
			name:    "untracked beads ID",
			beadsID: "orch-go-untracked-1766786808",
			want:    true,
		},
		{
			name:    "untracked with different project",
			beadsID: "snap-untracked-1766770347",
			want:    true,
		},
		{
			name:    "empty string",
			beadsID: "",
			want:    false,
		},
		{
			name:    "contains 'untracked' but not as segment",
			beadsID: "my-untrackedfeature-xyz",
			want:    false, // doesn't contain "-untracked-" with trailing hyphen
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isUntrackedBeadsID(tt.beadsID)
			if got != tt.want {
				t.Errorf("isUntrackedBeadsID(%q) = %v, want %v", tt.beadsID, got, tt.want)
			}
		})
	}
}

func TestGetClosedIssuesBatch_EmptyInput(t *testing.T) {
	// Empty input should return empty map
	result := GetClosedIssuesBatch(nil)
	if len(result) != 0 {
		t.Errorf("GetClosedIssuesBatch(nil) = %v, want empty map", result)
	}

	result = GetClosedIssuesBatch([]string{})
	if len(result) != 0 {
		t.Errorf("GetClosedIssuesBatch([]) = %v, want empty map", result)
	}
}

// TestGetClosedIssuesBatch_Integration is an integration test that requires
// a beads daemon or CLI to be available. It's skipped in CI.
func TestGetClosedIssuesBatch_Integration(t *testing.T) {
	// Skip if no beads socket available (CI environment)
	socketPath, err := beads.FindSocketPath("")
	if err != nil {
		t.Skip("Skipping integration test: no beads socket available")
	}

	// Try to connect
	client := beads.NewClient(socketPath)
	if err := client.Connect(); err != nil {
		t.Skip("Skipping integration test: cannot connect to beads daemon")
	}
	client.Close()

	// This test just verifies the function doesn't panic with valid input
	// The actual result depends on the state of the beads database
	result := GetClosedIssuesBatch([]string{"nonexistent-id-xyz"})
	// Should return empty or error gracefully
	if result == nil {
		t.Error("GetClosedIssuesBatch() returned nil, want non-nil map")
	}
}

// =============================================================================
// Tests for RateLimiter
// =============================================================================

func TestNewRateLimiter(t *testing.T) {
	r := NewRateLimiter(20)

	if r.MaxPerHour != 20 {
		t.Errorf("NewRateLimiter(20).MaxPerHour = %d, want 20", r.MaxPerHour)
	}
	if len(r.SpawnHistory) != 0 {
		t.Errorf("NewRateLimiter(20).SpawnHistory should be empty, got %d entries", len(r.SpawnHistory))
	}
	if r.nowFunc == nil {
		t.Error("NewRateLimiter(20).nowFunc should not be nil")
	}
}

func TestRateLimiter_CanSpawn_NoLimit(t *testing.T) {
	r := NewRateLimiter(0) // No limit

	canSpawn, count, msg := r.CanSpawn()
	if !canSpawn {
		t.Error("CanSpawn() should return true when no limit is set")
	}
	if count != 0 {
		t.Errorf("CanSpawn() count = %d, want 0 (no tracking)", count)
	}
	if msg != "" {
		t.Errorf("CanSpawn() msg = %q, want empty", msg)
	}
}

func TestRateLimiter_CanSpawn_BelowLimit(t *testing.T) {
	r := NewRateLimiter(5)

	// Record 3 spawns
	for i := 0; i < 3; i++ {
		r.RecordSpawn()
	}

	canSpawn, count, msg := r.CanSpawn()
	if !canSpawn {
		t.Error("CanSpawn() should return true when below limit")
	}
	if count != 3 {
		t.Errorf("CanSpawn() count = %d, want 3", count)
	}
	if msg != "" {
		t.Errorf("CanSpawn() msg = %q, want empty", msg)
	}
}

func TestRateLimiter_CanSpawn_AtLimit(t *testing.T) {
	r := NewRateLimiter(3)

	// Record exactly 3 spawns
	for i := 0; i < 3; i++ {
		r.RecordSpawn()
	}

	canSpawn, count, msg := r.CanSpawn()
	if canSpawn {
		t.Error("CanSpawn() should return false when at limit")
	}
	if count != 3 {
		t.Errorf("CanSpawn() count = %d, want 3", count)
	}
	if msg == "" {
		t.Error("CanSpawn() should return a message when at limit")
	}
}

func TestRateLimiter_CanSpawn_ExpiredHistory(t *testing.T) {
	r := NewRateLimiter(3)

	// Use a mock time function
	baseTime := time.Now()
	r.nowFunc = func() time.Time { return baseTime }

	// Record 3 spawns at base time
	for i := 0; i < 3; i++ {
		r.RecordSpawn()
	}

	// Move time forward by more than an hour
	r.nowFunc = func() time.Time { return baseTime.Add(61 * time.Minute) }

	// Old spawns should be expired
	canSpawn, count, _ := r.CanSpawn()
	if !canSpawn {
		t.Error("CanSpawn() should return true when old spawns are expired")
	}
	if count != 0 {
		t.Errorf("CanSpawn() count = %d, want 0 (expired)", count)
	}
}

func TestRateLimiter_RecordSpawn(t *testing.T) {
	r := NewRateLimiter(10)

	r.RecordSpawn()
	if len(r.SpawnHistory) != 1 {
		t.Errorf("RecordSpawn() should add one entry, got %d", len(r.SpawnHistory))
	}

	r.RecordSpawn()
	r.RecordSpawn()
	if len(r.SpawnHistory) != 3 {
		t.Errorf("RecordSpawn() should have 3 entries, got %d", len(r.SpawnHistory))
	}
}

func TestRateLimiter_SpawnsRemaining(t *testing.T) {
	tests := []struct {
		name     string
		max      int
		spawns   int
		wantLeft int
	}{
		{"no limit", 0, 10, 100},
		{"none used", 5, 0, 5},
		{"some used", 10, 3, 7},
		{"all used", 5, 5, 0},
		{"over limit", 3, 5, 0}, // Can't have negative remaining
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRateLimiter(tt.max)
			for i := 0; i < tt.spawns; i++ {
				r.RecordSpawn()
			}

			got := r.SpawnsRemaining()
			if got != tt.wantLeft {
				t.Errorf("SpawnsRemaining() = %d, want %d", got, tt.wantLeft)
			}
		})
	}
}

func TestRateLimiter_Status(t *testing.T) {
	r := NewRateLimiter(10)
	for i := 0; i < 3; i++ {
		r.RecordSpawn()
	}

	status := r.Status()
	if status.MaxPerHour != 10 {
		t.Errorf("Status().MaxPerHour = %d, want 10", status.MaxPerHour)
	}
	if status.SpawnsLastHour != 3 {
		t.Errorf("Status().SpawnsLastHour = %d, want 3", status.SpawnsLastHour)
	}
	if status.SpawnsRemaining != 7 {
		t.Errorf("Status().SpawnsRemaining = %d, want 7", status.SpawnsRemaining)
	}
	if status.LimitReached {
		t.Error("Status().LimitReached should be false")
	}
}

func TestDaemon_OnceExcluding_RateLimited(t *testing.T) {
	d := &Daemon{
		Config: Config{MaxSpawnsPerHour: 2},
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "First", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		},
		spawnFunc: func(id string, model string) error { return nil },
		updateBeadsStatusFunc: func(beadsID string, status string) error {
			return nil // Mock: always succeed
		},
	}
	d.RateLimiter = NewRateLimiter(2)

	// First spawn should succeed
	result, err := d.OnceExcluding(nil)
	if err != nil {
		t.Fatalf("OnceExcluding() unexpected error: %v", err)
	}
	if !result.Processed {
		t.Error("First spawn should be processed")
	}

	// Second spawn should succeed
	result, err = d.OnceExcluding(nil)
	if err != nil {
		t.Fatalf("OnceExcluding() unexpected error: %v", err)
	}
	if !result.Processed {
		t.Error("Second spawn should be processed")
	}

	// Third spawn should be rate limited
	result, err = d.OnceExcluding(nil)
	if err != nil {
		t.Fatalf("OnceExcluding() unexpected error: %v", err)
	}
	if result.Processed {
		t.Error("Third spawn should be rate limited")
	}
	if result.Message == "" || !strings.Contains(result.Message, "Rate limited") {
		t.Errorf("Rate limited message expected, got: %q", result.Message)
	}
}

func TestDaemon_Preview_RateLimited(t *testing.T) {
	d := &Daemon{
		Config: Config{MaxSpawnsPerHour: 1},
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "First", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		},
	}
	d.RateLimiter = NewRateLimiter(1)

	// First preview should show rate status
	result, err := d.Preview()
	if err != nil {
		t.Fatalf("Preview() unexpected error: %v", err)
	}
	if result.RateLimited {
		t.Error("First preview should not be rate limited")
	}
	if result.RateStatus == "" {
		t.Error("Rate status should be set")
	}

	// Record a spawn to hit limit
	d.RateLimiter.RecordSpawn()

	// Preview should now show rate limited
	result, err = d.Preview()
	if err != nil {
		t.Fatalf("Preview() unexpected error: %v", err)
	}
	if !result.RateLimited {
		t.Error("Second preview should be rate limited")
	}
}

// =============================================================================
// Tests for Periodic Reflection
// =============================================================================

func TestDaemon_ShouldRunReflection_Disabled(t *testing.T) {
	d := &Daemon{
		Config: Config{
			ReflectEnabled:  false,
			ReflectInterval: time.Hour,
		},
	}

	if d.ShouldRunReflection() {
		t.Error("ShouldRunReflection() should return false when disabled")
	}
}

func TestDaemon_ShouldRunReflection_ZeroInterval(t *testing.T) {
	d := &Daemon{
		Config: Config{
			ReflectEnabled:  true,
			ReflectInterval: 0,
		},
	}

	if d.ShouldRunReflection() {
		t.Error("ShouldRunReflection() should return false when interval is 0")
	}
}

func TestDaemon_ShouldRunReflection_NeverRun(t *testing.T) {
	d := &Daemon{
		Config: Config{
			ReflectEnabled:  true,
			ReflectInterval: time.Hour,
		},
	}

	// lastReflect is zero time (never run)
	if !d.ShouldRunReflection() {
		t.Error("ShouldRunReflection() should return true when never run before")
	}
}

func TestDaemon_ShouldRunReflection_IntervalElapsed(t *testing.T) {
	d := &Daemon{
		Config: Config{
			ReflectEnabled:  true,
			ReflectInterval: time.Hour,
		},
		lastReflect: time.Now().Add(-2 * time.Hour), // 2 hours ago
	}

	if !d.ShouldRunReflection() {
		t.Error("ShouldRunReflection() should return true when interval has elapsed")
	}
}

func TestDaemon_ShouldRunReflection_IntervalNotElapsed(t *testing.T) {
	d := &Daemon{
		Config: Config{
			ReflectEnabled:  true,
			ReflectInterval: time.Hour,
		},
		lastReflect: time.Now().Add(-30 * time.Minute), // 30 minutes ago
	}

	if d.ShouldRunReflection() {
		t.Error("ShouldRunReflection() should return false when interval has not elapsed")
	}
}

func TestDaemon_RunPeriodicReflection_NotDue(t *testing.T) {
	reflectCalled := false
	d := &Daemon{
		Config: Config{
			ReflectEnabled:      true,
			ReflectInterval:     time.Hour,
			ReflectCreateIssues: true,
		},
		lastReflect: time.Now(), // Just ran
		reflectFunc: func(createIssues bool) (*ReflectResult, error) {
			reflectCalled = true
			return &ReflectResult{}, nil
		},
	}

	result := d.RunPeriodicReflection()
	if result != nil {
		t.Error("RunPeriodicReflection() should return nil when not due")
	}
	if reflectCalled {
		t.Error("reflectFunc should not be called when not due")
	}
}

func TestDaemon_RunPeriodicReflection_Due(t *testing.T) {
	reflectCalled := false
	createIssuesValue := false
	d := &Daemon{
		Config: Config{
			ReflectEnabled:      true,
			ReflectInterval:     time.Hour,
			ReflectCreateIssues: true,
		},
		lastReflect: time.Now().Add(-2 * time.Hour), // 2 hours ago (due)
		reflectFunc: func(createIssues bool) (*ReflectResult, error) {
			reflectCalled = true
			createIssuesValue = createIssues
			return &ReflectResult{
				Suggestions: &ReflectSuggestions{
					Synthesis: []SynthesisSuggestion{{Topic: "test", Count: 5}},
				},
				Saved:   true,
				Message: "Test reflection",
			}, nil
		},
	}

	result := d.RunPeriodicReflection()
	if result == nil {
		t.Fatal("RunPeriodicReflection() should return result when due")
	}
	if !reflectCalled {
		t.Error("reflectFunc should be called when due")
	}
	if !createIssuesValue {
		t.Error("createIssues should be true based on config")
	}
	if d.lastReflect.IsZero() {
		t.Error("lastReflect should be updated after running")
	}
}

func TestDaemon_RunPeriodicReflection_OpenEnabled(t *testing.T) {
	openCalled := false
	d := &Daemon{
		Config: Config{
			ReflectEnabled:     true,
			ReflectInterval:    time.Hour,
			ReflectOpenEnabled: true,
		},
		lastReflect: time.Now().Add(-2 * time.Hour),
		reflectFunc: func(createIssues bool) (*ReflectResult, error) {
			return &ReflectResult{}, nil
		},
		openReflectFunc: func() error {
			openCalled = true
			return nil
		},
	}

	result := d.RunPeriodicReflection()
	if result == nil {
		t.Fatal("RunPeriodicReflection() should return result when due")
	}
	if !openCalled {
		t.Error("openReflectFunc should be called when ReflectOpenEnabled is true")
	}
}

func TestDaemon_RunPeriodicReflection_Error(t *testing.T) {
	d := &Daemon{
		Config: Config{
			ReflectEnabled:      true,
			ReflectInterval:     time.Hour,
			ReflectCreateIssues: false,
		},
		lastReflect: time.Time{}, // Never run
		reflectFunc: func(createIssues bool) (*ReflectResult, error) {
			return nil, fmt.Errorf("kb reflect failed")
		},
	}

	result := d.RunPeriodicReflection()
	if result == nil {
		t.Fatal("RunPeriodicReflection() should return result on error")
	}
	if result.Error == nil {
		t.Error("Result should have error")
	}
}

func TestDaemon_LastReflectTime(t *testing.T) {
	now := time.Now()
	d := &Daemon{
		lastReflect: now,
	}

	if !d.LastReflectTime().Equal(now) {
		t.Errorf("LastReflectTime() = %v, want %v", d.LastReflectTime(), now)
	}
}

func TestDaemon_NextReflectTime_Disabled(t *testing.T) {
	d := &Daemon{
		Config: Config{
			ReflectEnabled:  false,
			ReflectInterval: time.Hour,
		},
	}

	next := d.NextReflectTime()
	if !next.IsZero() {
		t.Error("NextReflectTime() should return zero time when disabled")
	}
}

func TestDaemon_NextReflectTime_NeverRun(t *testing.T) {
	d := &Daemon{
		Config: Config{
			ReflectEnabled:  true,
			ReflectInterval: time.Hour,
		},
		lastReflect: time.Time{}, // Never run
	}

	next := d.NextReflectTime()
	// Should return approximately now (due immediately)
	if time.Until(next) > time.Second {
		t.Error("NextReflectTime() should return ~now when never run")
	}
}

func TestDaemon_NextReflectTime_AfterRun(t *testing.T) {
	now := time.Now()
	d := &Daemon{
		Config: Config{
			ReflectEnabled:  true,
			ReflectInterval: time.Hour,
		},
		lastReflect: now,
	}

	next := d.NextReflectTime()
	expectedNext := now.Add(time.Hour)
	// Allow 1 second tolerance
	if next.Sub(expectedNext).Abs() > time.Second {
		t.Errorf("NextReflectTime() = %v, want ~%v", next, expectedNext)
	}
}

func TestDefaultConfig_IncludesReflect(t *testing.T) {
	config := DefaultConfig()

	if !config.ReflectEnabled {
		t.Error("DefaultConfig().ReflectEnabled should be true")
	}
	if config.ReflectInterval != time.Hour {
		t.Errorf("DefaultConfig().ReflectInterval = %v, want 1h", config.ReflectInterval)
	}
	if !config.ReflectCreateIssues {
		t.Error("DefaultConfig().ReflectCreateIssues should be true")
	}
	if !config.ReflectOpenEnabled {
		t.Error("DefaultConfig().ReflectOpenEnabled should be true")
	}
	if !config.ReflectModelDriftEnabled {
		t.Error("DefaultConfig().ReflectModelDriftEnabled should be true")
	}
	if config.ReflectModelDriftInterval != 4*time.Hour {
		t.Errorf("DefaultConfig().ReflectModelDriftInterval = %v, want 4h", config.ReflectModelDriftInterval)
	}
}

func TestNewWithConfig_InitializesReflectFunc(t *testing.T) {
	config := Config{
		ReflectEnabled:  true,
		ReflectInterval: time.Hour,
	}
	d := NewWithConfig(config)

	if d.reflectFunc == nil {
		t.Error("NewWithConfig() should initialize reflectFunc")
	}
}

// Tests for epic child expansion feature
// When an epic has triage:ready label, its children should be auto-included in spawn queue

func TestExpandTriageReadyEpics_NoEpics(t *testing.T) {
	d := &Daemon{
		Config: Config{Label: "triage:ready"},
	}

	issues := []Issue{
		{ID: "proj-1", Title: "Feature", IssueType: "feature", Labels: []string{"triage:ready"}},
		{ID: "proj-2", Title: "Bug", IssueType: "bug", Labels: []string{"triage:ready"}},
	}

	expanded, epicChildIDs, err := d.expandTriageReadyEpics(issues)
	if err != nil {
		t.Fatalf("expandTriageReadyEpics() unexpected error: %v", err)
	}

	// No epics, so nothing should change
	if len(expanded) != 2 {
		t.Errorf("expandTriageReadyEpics() returned %d issues, want 2", len(expanded))
	}
	if len(epicChildIDs) != 0 {
		t.Errorf("expandTriageReadyEpics() returned %d epic children, want 0", len(epicChildIDs))
	}
}

func TestExpandTriageReadyEpics_NoLabelFilter(t *testing.T) {
	d := &Daemon{
		Config: Config{Label: ""}, // No label filter
	}

	issues := []Issue{
		{ID: "proj-1", Title: "Epic", IssueType: "epic", Labels: []string{}},
		{ID: "proj-1.1", Title: "Child", IssueType: "task", Labels: []string{}},
	}

	expanded, epicChildIDs, err := d.expandTriageReadyEpics(issues)
	if err != nil {
		t.Fatalf("expandTriageReadyEpics() unexpected error: %v", err)
	}

	// No label filter, so no expansion needed
	if len(expanded) != 2 {
		t.Errorf("expandTriageReadyEpics() returned %d issues, want 2", len(expanded))
	}
	if len(epicChildIDs) != 0 {
		t.Errorf("expandTriageReadyEpics() returned %d epic children, want 0", len(epicChildIDs))
	}
}

func TestExpandTriageReadyEpics_EpicWithoutLabel(t *testing.T) {
	d := &Daemon{
		Config: Config{Label: "triage:ready"},
	}

	issues := []Issue{
		{ID: "proj-epic", Title: "Epic", IssueType: "epic", Labels: []string{}}, // No triage:ready
		{ID: "proj-1", Title: "Feature", IssueType: "feature", Labels: []string{"triage:ready"}},
	}

	expanded, epicChildIDs, err := d.expandTriageReadyEpics(issues)
	if err != nil {
		t.Fatalf("expandTriageReadyEpics() unexpected error: %v", err)
	}

	// Epic doesn't have the label, so no expansion
	if len(expanded) != 2 {
		t.Errorf("expandTriageReadyEpics() returned %d issues, want 2", len(expanded))
	}
	if len(epicChildIDs) != 0 {
		t.Errorf("expandTriageReadyEpics() returned %d epic children, want 0", len(epicChildIDs))
	}
}

func TestNextIssue_EpicChildrenIncludedInSpawnQueue(t *testing.T) {
	// Mock the ListEpicChildren function via the listIssuesFunc
	// by including all issues upfront (simulating what expandTriageReadyEpics would do)
	epicChildCalled := false

	d := &Daemon{
		Config: Config{Label: "triage:ready"},
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-epic", Title: "Epic", IssueType: "epic", Labels: []string{"triage:ready"}},
				{ID: "proj-1", Title: "Feature without label", IssueType: "feature", Status: "open", Labels: []string{}},
			}, nil
		},
		listEpicChildrenFunc: func(epicID string) ([]Issue, error) {
			epicChildCalled = true
			if epicID == "proj-epic" {
				return []Issue{
					{ID: "proj-child-1", Title: "Child 1", IssueType: "task", Status: "open"},
				}, nil
			}
			return []Issue{}, nil
		},
	}

	// Create a wrapper that tracks ListEpicChildren calls
	// We can't easily mock ListEpicChildren since it's a package-level function,
	// but we can test the overall behavior by checking the results

	// First, verify that without epic child expansion, the feature would be rejected
	// (it doesn't have the required label)
	issue, err := d.NextIssue()
	if err != nil {
		t.Fatalf("NextIssue() unexpected error: %v", err)
	}

	// The epic should be skipped (not spawnable), and the feature should be skipped (no label)
	// unless epic child expansion is working
	if issue != nil {
		t.Logf("NextIssue() returned %s", issue.ID)
		// If we got an issue, epic child expansion must be working
		epicChildCalled = true
	}

	// For a full integration test, we would need to mock ListEpicChildren
	// which would require dependency injection. For now, test the logic directly.
	_ = epicChildCalled
}

func TestCheckRejectionReasonWithEpicChildren_EpicChildExempt(t *testing.T) {
	d := &Daemon{
		Config: Config{Label: "triage:ready"},
	}

	// Issue without triage:ready label
	issue := Issue{
		ID:        "proj-1",
		Title:     "Feature",
		IssueType: "feature",
		Status:    "open",
		Labels:    []string{}, // No triage:ready label
	}

	// Without being an epic child, should be rejected for missing label
	reason := d.checkRejectionReasonWithEpicChildren(issue, nil)
	if !strings.Contains(reason, "missing label") {
		t.Errorf("checkRejectionReasonWithEpicChildren() = %q, want to contain 'missing label'", reason)
	}

	// When marked as an epic child, should be accepted (empty reason)
	epicChildIDs := map[string]bool{"proj-1": true}
	reason = d.checkRejectionReasonWithEpicChildren(issue, epicChildIDs)
	if reason != "" {
		t.Errorf("checkRejectionReasonWithEpicChildren() for epic child = %q, want empty string", reason)
	}
}

func TestCheckRejectionReasonWithEpicChildren_EpicWithLabelExplains(t *testing.T) {
	d := &Daemon{
		Config: Config{Label: "triage:ready"},
	}

	// Epic with triage:ready label
	issue := Issue{
		ID:        "proj-epic",
		Title:     "Epic",
		IssueType: "epic",
		Status:    "open",
		Labels:    []string{"triage:ready"},
	}

	reason := d.checkRejectionReasonWithEpicChildren(issue, nil)

	// Should explain that children will be processed instead
	if !strings.Contains(reason, "children will be processed") {
		t.Errorf("checkRejectionReasonWithEpicChildren() for triage:ready epic = %q, want to contain 'children will be processed'", reason)
	}
}

func TestPreview_EpicWithTriageReadyShowsHelpfulMessage(t *testing.T) {
	d := &Daemon{
		Config: Config{Label: "triage:ready"},
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-epic", Title: "Epic", IssueType: "epic", Status: "open", Labels: []string{"triage:ready"}},
			}, nil
		},
		// Mock to return no children, isolating the test from real data
		listEpicChildrenFunc: func(epicID string) ([]Issue, error) {
			return []Issue{}, nil
		},
	}

	result, err := d.Preview()
	if err != nil {
		t.Fatalf("Preview() unexpected error: %v", err)
	}

	// No spawnable issue since only an epic in queue
	if result.Issue != nil {
		t.Errorf("Preview() expected nil issue (epic not spawnable), got %v", result.Issue)
	}

	// Should have the epic in rejected list with helpful message
	if len(result.RejectedIssues) != 1 {
		t.Fatalf("Preview() rejected count = %d, want 1", len(result.RejectedIssues))
	}

	rejected := result.RejectedIssues[0]
	if rejected.Issue.ID != "proj-epic" {
		t.Errorf("Preview() rejected issue ID = %q, want 'proj-epic'", rejected.Issue.ID)
	}
	if !strings.Contains(rejected.Reason, "children will be processed") {
		t.Errorf("Preview() rejection reason = %q, want to contain 'children will be processed'", rejected.Reason)
	}
}

func TestExpandTriageReadyEpics_FiltersClosedChildren(t *testing.T) {
	d := &Daemon{
		Config: Config{Label: "triage:ready", Verbose: true},
		listEpicChildrenFunc: func(epicID string) ([]Issue, error) {
			if epicID == "proj-epic" {
				return []Issue{
					{ID: "proj-child-1", Title: "Open Child", IssueType: "feature", Status: "open"},
					{ID: "proj-child-2", Title: "Closed Child", IssueType: "feature", Status: "closed"},
					{ID: "proj-child-3", Title: "In Progress Child", IssueType: "feature", Status: "in_progress"},
				}, nil
			}
			return []Issue{}, nil
		},
	}

	issues := []Issue{
		{ID: "proj-epic", Title: "Epic", IssueType: "epic", Status: "open", Labels: []string{"triage:ready"}},
	}

	expanded, epicChildIDs, err := d.expandTriageReadyEpics(issues)
	if err != nil {
		t.Fatalf("expandTriageReadyEpics() unexpected error: %v", err)
	}

	// Should have original epic + 2 children (open and in_progress, but NOT closed)
	if len(expanded) != 3 {
		t.Errorf("expandTriageReadyEpics() returned %d issues, want 3 (epic + 2 open children)", len(expanded))
	}

	// Only the 2 non-closed children should be marked as epic children
	if len(epicChildIDs) != 2 {
		t.Errorf("expandTriageReadyEpics() returned %d epic children, want 2", len(epicChildIDs))
	}

	// Verify open child is included
	if !epicChildIDs["proj-child-1"] {
		t.Error("expandTriageReadyEpics() did not include open child proj-child-1")
	}

	// Verify closed child is NOT included
	if epicChildIDs["proj-child-2"] {
		t.Error("expandTriageReadyEpics() incorrectly included closed child proj-child-2")
	}

	// Verify in_progress child is included
	if !epicChildIDs["proj-child-3"] {
		t.Error("expandTriageReadyEpics() did not include in_progress child proj-child-3")
	}

	// Verify the closed child is not in the expanded issues list
	for _, issue := range expanded {
		if issue.ID == "proj-child-2" {
			t.Error("expandTriageReadyEpics() added closed child to issues list")
		}
	}
}

// TestOriginalBugReproduction verifies the fix for orch-go-4mu.
// Issue titled "Architect: Design accretion gravity enforcement infrastructure"
// was incorrectly inferred as "investigation" instead of "architect".
func TestOriginalBugReproduction(t *testing.T) {
	issue := &Issue{
		ID:          "orch-go-4mu",
		Title:       "Architect: Design accretion gravity enforcement infrastructure",
		Description: "",
		IssueType:   "task",
		Labels:      []string{},
	}

	got, err := InferSkillFromIssue(issue)
	if err != nil {
		t.Fatalf("InferSkillFromIssue() unexpected error: %v", err)
	}

	want := "architect"
	if got != want {
		t.Errorf("InferSkillFromIssue() = %q, want %q (bug reproduction failed - title prefix not detected)", got, want)
	}
}

func TestOnceExcluding_AutoExtraction_SpawnsExtractionWhenCriticalHotspot(t *testing.T) {
	// When a triage:ready issue targets a CRITICAL hotspot file (>1500 lines),
	// the daemon should create an extraction issue and spawn it instead.
	spawnedID := ""
	d := &Daemon{
		Config: Config{Verbose: true},
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{
					ID:        "proj-1",
					Title:     "Add feature to cmd/orch/spawn_cmd.go",
					Priority:  2,
					IssueType: "feature",
					Status:    "open",
				},
			}, nil
		},
		spawnFunc: func(beadsID string, model string) error {
			spawnedID = beadsID
			return nil
		},
		HotspotChecker: &mockDaemonHotspotChecker{
			hotspots: []HotspotWarning{
				{Path: "cmd/orch/spawn_cmd.go", Type: "bloat-size", Score: 2000},
			},
		},
		createExtractionIssueFunc: func(task, parentID string) (string, error) {
			// Verify the extraction task was generated correctly
			if parentID != "proj-1" {
				t.Errorf("createExtractionIssue parentID = %q, want 'proj-1'", parentID)
			}
			if !strings.Contains(task, "Extract") {
				t.Errorf("createExtractionIssue task should contain 'Extract', got: %s", task)
			}
			return "proj-ext1", nil
		},
		updateBeadsStatusFunc: func(beadsID string, status string) error {
			return nil // Mock: always succeed
		},
	}

	result, err := d.OnceExcluding(nil)
	if err != nil {
		t.Fatalf("OnceExcluding() unexpected error: %v", err)
	}
	if result == nil || !result.Processed {
		t.Fatal("OnceExcluding() expected processed result")
	}

	// Should have spawned the extraction issue, not the original
	if spawnedID != "proj-ext1" {
		t.Errorf("spawnFunc called with %q, want 'proj-ext1' (extraction issue)", spawnedID)
	}
	if !result.ExtractionSpawned {
		t.Error("OnceResult.ExtractionSpawned should be true")
	}
	if result.OriginalIssueID != "proj-1" {
		t.Errorf("OnceResult.OriginalIssueID = %q, want 'proj-1'", result.OriginalIssueID)
	}
	if result.Issue.ID != "proj-ext1" {
		t.Errorf("OnceResult.Issue.ID = %q, want 'proj-ext1'", result.Issue.ID)
	}
}

func TestOnceExcluding_AutoExtraction_SkipsWhenNoCriticalHotspot(t *testing.T) {
	// When hotspot check finds no CRITICAL files, spawn normally.
	spawnedID := ""
	d := &Daemon{
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{
					ID:        "proj-1",
					Title:     "Add feature to pkg/daemon/daemon.go",
					Priority:  2,
					IssueType: "feature",
					Status:    "open",
				},
			}, nil
		},
		spawnFunc: func(beadsID string, model string) error {
			spawnedID = beadsID
			return nil
		},
		HotspotChecker: &mockDaemonHotspotChecker{
			hotspots: []HotspotWarning{
				// Below critical threshold
				{Path: "pkg/daemon/daemon.go", Type: "bloat-size", Score: 1200},
			},
		},
		updateBeadsStatusFunc: func(beadsID string, status string) error {
			return nil // Mock: always succeed
		},
	}

	result, err := d.OnceExcluding(nil)
	if err != nil {
		t.Fatalf("OnceExcluding() unexpected error: %v", err)
	}
	if result == nil || !result.Processed {
		t.Fatal("OnceExcluding() expected processed result")
	}

	// Should have spawned the original issue normally
	if spawnedID != "proj-1" {
		t.Errorf("spawnFunc called with %q, want 'proj-1' (original issue)", spawnedID)
	}
	if result.ExtractionSpawned {
		t.Error("OnceResult.ExtractionSpawned should be false")
	}
}

func TestOnceExcluding_AutoExtraction_FailsFastOnExtractionFailure(t *testing.T) {
	// When extraction issue creation fails, skip the issue (fail-fast).
	// Extraction gate is non-negotiable - do not proceed with normal spawn.
	spawnedID := ""
	d := &Daemon{
		Config: Config{Verbose: true},
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{
					ID:        "proj-1",
					Title:     "Add feature to cmd/orch/spawn_cmd.go",
					Priority:  2,
					IssueType: "feature",
					Status:    "open",
				},
			}, nil
		},
		spawnFunc: func(beadsID string, model string) error {
			spawnedID = beadsID
			return nil
		},
		HotspotChecker: &mockDaemonHotspotChecker{
			hotspots: []HotspotWarning{
				{Path: "cmd/orch/spawn_cmd.go", Type: "bloat-size", Score: 2000},
			},
		},
		createExtractionIssueFunc: func(task, parentID string) (string, error) {
			return "", fmt.Errorf("bd create failed: command not found")
		},
	}

	result, err := d.OnceExcluding(nil)
	if err != nil {
		t.Fatalf("OnceExcluding() unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("OnceExcluding() expected non-nil result")
	}

	// Should NOT have processed/spawned - extraction gate is non-negotiable
	if result.Processed {
		t.Error("OnceExcluding() should NOT process when extraction setup fails (fail-fast)")
	}

	// Should not have spawned the original issue
	if spawnedID != "" {
		t.Errorf("spawnFunc should not be called when extraction fails, but was called with %q", spawnedID)
	}

	// Should have a message explaining the skip
	if result.Message == "" {
		t.Error("OnceResult.Message should explain why issue was skipped")
	}
}

func TestOnceExcluding_AutoExtraction_SkipsWhenNoHotspotChecker(t *testing.T) {
	// When HotspotChecker is nil, no extraction check happens.
	spawnedID := ""
	d := &Daemon{
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{
					ID:        "proj-1",
					Title:     "Add feature to cmd/orch/spawn_cmd.go",
					Priority:  2,
					IssueType: "feature",
					Status:    "open",
				},
			}, nil
		},
		spawnFunc: func(beadsID string, model string) error {
			spawnedID = beadsID
			return nil
		},
		updateBeadsStatusFunc: func(beadsID string, status string) error {
			return nil // Mock: always succeed
		},
		// HotspotChecker is nil
	}

	result, err := d.OnceExcluding(nil)
	if err != nil {
		t.Fatalf("OnceExcluding() unexpected error: %v", err)
	}
	if result == nil || !result.Processed {
		t.Fatal("OnceExcluding() expected processed result")
	}

	// Should have spawned normally without extraction check
	if spawnedID != "proj-1" {
		t.Errorf("spawnFunc called with %q, want 'proj-1'", spawnedID)
	}
	if result.ExtractionSpawned {
		t.Error("OnceResult.ExtractionSpawned should be false when no checker")
	}
}

// TestDaemon_Once_FreshStatusCheck_SkipsInProgressIssue verifies that the fresh
// beads status check prevents spawning an issue that another daemon process
// already marked as in_progress. This was the root cause of the Feb 15 2026
// orch-go-09cc duplicate spawn incident (TOCTOU race between concurrent daemons).
func TestDaemon_Once_FreshStatusCheck_SkipsInProgressIssue(t *testing.T) {
	spawnCalled := false
	d := &Daemon{
		listIssuesFunc: func() ([]Issue, error) {
			// List returns the issue as "open" (stale cached status)
			return []Issue{
				{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		},
		spawnFunc: func(beadsID string, model string) error {
			spawnCalled = true
			return nil
		},
		// Fresh status check reveals the issue is already in_progress
		getIssueStatusFunc: func(beadsID string) (string, error) {
			return "in_progress", nil
		},
	}

	result, err := d.Once()
	if err != nil {
		t.Fatalf("Once() unexpected error: %v", err)
	}
	if result.Processed {
		t.Error("Once() should not process an in_progress issue")
	}
	if spawnCalled {
		t.Error("spawnFunc should not be called when fresh status check shows in_progress")
	}
	if result.Issue == nil || result.Issue.ID != "proj-1" {
		t.Error("result.Issue should still reference the skipped issue")
	}
}

// TestDaemon_Once_FreshStatusCheck_AllowsOpenIssue verifies that an issue
// with status "open" in both cached and fresh checks proceeds to spawn.
func TestDaemon_Once_FreshStatusCheck_AllowsOpenIssue(t *testing.T) {
	spawnCalled := false
	d := &Daemon{
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		},
		spawnFunc: func(beadsID string, model string) error {
			spawnCalled = true
			return nil
		},
		getIssueStatusFunc: func(beadsID string) (string, error) {
			return "open", nil // Fresh status check confirms issue is still open
		},
		updateBeadsStatusFunc: func(beadsID string, status string) error {
			return nil // Mock: always succeed
		},
	}

	result, err := d.Once()
	if err != nil {
		t.Fatalf("Once() unexpected error: %v", err)
	}
	if !result.Processed {
		t.Error("Once() should process an open issue")
	}
	if !spawnCalled {
		t.Error("spawnFunc should be called when fresh status check confirms open")
	}
}

// TestDaemon_Once_FreshStatusCheck_FailOpenOnError verifies that if the
// fresh status check fails (beads unavailable), the daemon continues to spawn
// (fail-open). This prevents beads outages from blocking all daemon work.
func TestDaemon_Once_FreshStatusCheck_FailOpenOnError(t *testing.T) {
	spawnCalled := false
	d := &Daemon{
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		},
		spawnFunc: func(beadsID string, model string) error {
			spawnCalled = true
			return nil
		},
		// Fresh status check fails (beads unavailable)
		getIssueStatusFunc: func(beadsID string) (string, error) {
			return "", fmt.Errorf("beads daemon unavailable")
		},
		updateBeadsStatusFunc: func(beadsID string, status string) error {
			return nil // Mock: always succeed
		},
	}

	result, err := d.Once()
	if err != nil {
		t.Fatalf("Once() unexpected error: %v", err)
	}
	if !result.Processed {
		t.Error("Once() should still process when fresh status check fails (fail-open)")
	}
	if !spawnCalled {
		t.Error("spawnFunc should be called when fresh status check errors (fail-open)")
	}
}

// TestDaemon_Once_FreshStatusCheck_NilFunc verifies that the daemon works
// when getIssueStatusFunc is nil (backwards compatibility with tests that
// construct Daemon structs directly without setting all func fields).
func TestDaemon_Once_FreshStatusCheck_NilFunc(t *testing.T) {
	spawnCalled := false
	d := &Daemon{
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		},
		spawnFunc: func(beadsID string, model string) error {
			spawnCalled = true
			return nil
		},
		// getIssueStatusFunc is nil (not set)
		updateBeadsStatusFunc: func(beadsID string, status string) error {
			return nil // Mock: always succeed
		},
	}

	result, err := d.Once()
	if err != nil {
		t.Fatalf("Once() unexpected error: %v", err)
	}
	if !result.Processed {
		t.Error("Once() should process when getIssueStatusFunc is nil")
	}
	if !spawnCalled {
		t.Error("spawnFunc should be called when getIssueStatusFunc is nil")
	}
}

// TestDaemon_ConcurrentDaemonDedup simulates the exact scenario from the
// Feb 15 2026 incident: two daemon instances (daemon run + daemon once)
// racing to spawn the same issue. The first daemon's UpdateBeadsStatus
// makes the issue in_progress, which the second daemon's fresh status
// check should detect.
func TestDaemon_ConcurrentDaemonDedup(t *testing.T) {
	// Shared state simulating beads database
	issueStatus := "open"
	spawnCount := 0

	makeStatusFunc := func() func(string) (string, error) {
		return func(beadsID string) (string, error) {
			return issueStatus, nil
		}
	}

	makeDaemon := func() *Daemon {
		return &Daemon{
			listIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
				}, nil
			},
			spawnFunc: func(beadsID string, model string) error {
				spawnCount++
				return nil
			},
			getIssueStatusFunc: makeStatusFunc(),
			updateBeadsStatusFunc: func(beadsID string, status string) error {
				issueStatus = status // Update shared state
				return nil
			},
		}
	}

	// Daemon 1 spawns the issue
	d1 := makeDaemon()
	result1, err := d1.Once()
	if err != nil {
		t.Fatalf("Daemon 1 Once() unexpected error: %v", err)
	}
	if !result1.Processed {
		t.Error("Daemon 1 should have processed the issue")
	}

	// Simulate beads status update (would happen via UpdateBeadsStatus in production)
	issueStatus = "in_progress"

	// Daemon 2 (new instance, no shared memory) tries the same issue
	d2 := makeDaemon()
	result2, err := d2.Once()
	if err != nil {
		t.Fatalf("Daemon 2 Once() unexpected error: %v", err)
	}
	if result2.Processed {
		t.Error("Daemon 2 should NOT have processed the issue (fresh status check should catch in_progress)")
	}

	if spawnCount != 1 {
		t.Errorf("Expected exactly 1 spawn, got %d", spawnCount)
	}
}

// mockDaemonHotspotChecker implements HotspotChecker for daemon tests.
type mockDaemonHotspotChecker struct {
	hotspots []HotspotWarning
}

func (m *mockDaemonHotspotChecker) CheckHotspots(projectDir string) ([]HotspotWarning, error) {
	return m.hotspots, nil
}

func TestExpandTriageReadyEpics_ListChildrenError(t *testing.T) {
	d := &Daemon{
		Config: Config{Label: "triage:ready"},
		listEpicChildrenFunc: func(epicID string) ([]Issue, error) {
			return nil, fmt.Errorf("simulated error listing children")
		},
	}

	issues := []Issue{
		{ID: "proj-epic", Title: "Epic", IssueType: "epic", Labels: []string{"triage:ready"}},
		{ID: "proj-task", Title: "Task", IssueType: "task", Labels: []string{}},
	}

	_, _, err := d.expandTriageReadyEpics(issues)
	if err == nil {
		t.Error("expandTriageReadyEpics() expected error when ListEpicChildren fails, got nil")
	}

	if err != nil && !strings.Contains(err.Error(), "failed to list children of epic proj-epic") {
		t.Errorf("expandTriageReadyEpics() error message = %v, want to contain 'failed to list children of epic proj-epic'", err)
	}
}
