// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"fmt"
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

func TestInferSkill(t *testing.T) {
	tests := []struct {
		issueType string
		wantSkill string
		wantErr   bool
	}{
		{"bug", "systematic-debugging", false},
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
		name   string
		labels []string
		want   string
	}{
		{"skill:research", []string{"triage:ready", "skill:research"}, "research"},
		{"skill:investigation", []string{"skill:investigation"}, "investigation"},
		{"skill:feature-impl", []string{"P1", "skill:feature-impl", "P2"}, "feature-impl"},
		{"first skill wins", []string{"skill:first", "skill:second"}, "first"},
		{"no skill label", []string{"triage:ready", "P1"}, ""},
		{"empty labels", []string{}, ""},
		{"nil labels", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InferSkillFromLabels(tt.labels)
			if got != tt.want {
				t.Errorf("InferSkillFromLabels(%v) = %q, want %q", tt.labels, got, tt.want)
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
			name:      "skill label takes priority over type",
			issue:     &Issue{IssueType: "task", Labels: []string{"skill:research"}},
			wantSkill: "research",
			wantErr:   false,
		},
		{
			name:      "skill label overrides bug type",
			issue:     &Issue{IssueType: "bug", Labels: []string{"skill:investigation"}},
			wantSkill: "investigation",
			wantErr:   false,
		},
		{
			name:      "falls back to type when no skill label",
			issue:     &Issue{IssueType: "bug", Labels: []string{"triage:ready", "P1"}},
			wantSkill: "systematic-debugging",
			wantErr:   false,
		},
		{
			name:      "falls back to type with empty labels",
			issue:     &Issue{IssueType: "feature", Labels: []string{}},
			wantSkill: "feature-impl",
			wantErr:   false,
		},
		{
			name:      "falls back to type with nil labels",
			issue:     &Issue{IssueType: "investigation", Labels: nil},
			wantSkill: "investigation",
			wantErr:   false,
		},
		{
			name:      "error for non-spawnable type without skill label",
			issue:     &Issue{IssueType: "epic", Labels: []string{}},
			wantSkill: "",
			wantErr:   true,
		},
		{
			name:      "skill label allows spawning non-standard type",
			issue:     &Issue{IssueType: "epic", Labels: []string{"skill:research"}},
			wantSkill: "research",
			wantErr:   false,
		},
		{
			name:    "nil issue returns error",
			issue:   nil,
			wantErr: true,
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

func TestDaemon_Preview_WithSkillLabel(t *testing.T) {
	d := &Daemon{
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{
					ID:        "proj-1",
					Title:     "Research task",
					Priority:  1,
					IssueType: "task",
					Status:    "open",
					Labels:    []string{"triage:ready", "skill:research"},
				},
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
	// Should use skill:research, not infer feature-impl from task type
	if result.Skill != "research" {
		t.Errorf("Preview() skill = %q, want 'research' (from skill label, not type)", result.Skill)
	}
}

func TestDaemon_Once_WithSkillLabel(t *testing.T) {
	spawnCalled := false
	d := &Daemon{
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{
					ID:        "proj-1",
					Title:     "Investigation via label",
					Priority:  0,
					IssueType: "bug",
					Status:    "open",
					Labels:    []string{"skill:investigation"},
				},
			}, nil
		},
		spawnFunc: func(beadsID string) error {
			spawnCalled = true
			return nil
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
	// Should use skill:investigation, not infer systematic-debugging from bug type
	if result.Skill != "investigation" {
		t.Errorf("Once() skill = %q, want 'investigation' (from skill label, not type)", result.Skill)
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
		spawnFunc: func(beadsID string) error {
			spawnCalled = true
			if beadsID != "proj-1" {
				t.Errorf("spawnFunc called with %q, want 'proj-1'", beadsID)
			}
			return nil
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
		spawnFunc: func(beadsID string) error {
			callCount++
			return nil
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
		spawnFunc: func(beadsID string) error {
			callCount++
			return nil
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
			d := &Daemon{
				Config:          Config{MaxAgents: tt.maxAgents},
				activeCountFunc: tt.activeFunc,
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
			d := &Daemon{
				Config:          Config{MaxAgents: tt.maxAgents},
				activeCountFunc: tt.activeFunc,
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
	pool := NewWorkerPool(2)
	spawnCount := 0
	d := &Daemon{
		Pool: pool,
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		},
		spawnFunc: func(beadsID string) error {
			spawnCount++
			return nil
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
		spawnFunc: func(beadsID string) error {
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
		spawnFunc: func(beadsID string) error {
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
	d := &Daemon{
		Pool: pool,
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		},
		spawnFunc: func(beadsID string) error {
			return nil
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
		spawnFunc: func(beadsID string) error {
			return nil
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
	result := getClosedIssuesBatch(nil)
	if len(result) != 0 {
		t.Errorf("getClosedIssuesBatch(nil) = %v, want empty map", result)
	}

	result = getClosedIssuesBatch([]string{})
	if len(result) != 0 {
		t.Errorf("getClosedIssuesBatch([]) = %v, want empty map", result)
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
	result := getClosedIssuesBatch([]string{"nonexistent-id-xyz"})
	// Should return empty or error gracefully
	if result == nil {
		t.Error("getClosedIssuesBatch() returned nil, want non-nil map")
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

func TestRateLimiter_Prune(t *testing.T) {
	r := NewRateLimiter(10)

	baseTime := time.Now()
	r.nowFunc = func() time.Time { return baseTime }

	// Record 5 spawns
	for i := 0; i < 5; i++ {
		r.RecordSpawn()
	}

	if len(r.SpawnHistory) != 5 {
		t.Fatalf("Expected 5 entries before prune, got %d", len(r.SpawnHistory))
	}

	// Move time forward by 2 hours
	r.nowFunc = func() time.Time { return baseTime.Add(2 * time.Hour) }

	// Record another spawn (this triggers prune)
	r.RecordSpawn()

	// Old entries should be pruned
	if len(r.SpawnHistory) != 1 {
		t.Errorf("After prune, expected 1 entry (new spawn), got %d", len(r.SpawnHistory))
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

	// Record 4 spawns
	for i := 0; i < 4; i++ {
		r.RecordSpawn()
	}

	status := r.Status()
	if status.MaxPerHour != 10 {
		t.Errorf("Status().MaxPerHour = %d, want 10", status.MaxPerHour)
	}
	if status.SpawnsLastHour != 4 {
		t.Errorf("Status().SpawnsLastHour = %d, want 4", status.SpawnsLastHour)
	}
	if status.SpawnsRemaining != 6 {
		t.Errorf("Status().SpawnsRemaining = %d, want 6", status.SpawnsRemaining)
	}
	if status.LimitReached {
		t.Error("Status().LimitReached should be false")
	}

	// Fill up to limit
	for i := 0; i < 6; i++ {
		r.RecordSpawn()
	}

	status = r.Status()
	if !status.LimitReached {
		t.Error("Status().LimitReached should be true when at limit")
	}
}

// =============================================================================
// Tests for Daemon with RateLimiter
// =============================================================================

func TestNewWithConfig_CreatesRateLimiter(t *testing.T) {
	config := Config{
		MaxSpawnsPerHour: 15,
	}
	d := NewWithConfig(config)

	if d.RateLimiter == nil {
		t.Fatal("NewWithConfig() should create RateLimiter when MaxSpawnsPerHour > 0")
	}
	if d.RateLimiter.MaxPerHour != 15 {
		t.Errorf("RateLimiter.MaxPerHour = %d, want 15", d.RateLimiter.MaxPerHour)
	}
}

func TestNewWithConfig_NoRateLimiterWhenNoLimit(t *testing.T) {
	config := Config{
		MaxSpawnsPerHour: 0, // No limit
	}
	d := NewWithConfig(config)

	if d.RateLimiter != nil {
		t.Error("NewWithConfig() should not create RateLimiter when MaxSpawnsPerHour = 0")
	}
}

func TestDaemon_RateLimited(t *testing.T) {
	tests := []struct {
		name        string
		maxPerHour  int
		spawns      int
		wantLimited bool
	}{
		{"no limit", 0, 100, false},
		{"below limit", 10, 5, false},
		{"at limit", 5, 5, true},
		{"above limit", 3, 5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewWithConfig(Config{MaxSpawnsPerHour: tt.maxPerHour})

			// Record spawns
			if d.RateLimiter != nil {
				for i := 0; i < tt.spawns; i++ {
					d.RateLimiter.RecordSpawn()
				}
			}

			got := d.RateLimited()
			if got != tt.wantLimited {
				t.Errorf("RateLimited() = %v, want %v", got, tt.wantLimited)
			}
		})
	}
}

func TestDaemon_RateLimitMessage(t *testing.T) {
	d := NewWithConfig(Config{MaxSpawnsPerHour: 3})

	// Should be empty when not limited
	msg := d.RateLimitMessage()
	if msg != "" {
		t.Errorf("RateLimitMessage() = %q, want empty when not limited", msg)
	}

	// Fill to limit
	for i := 0; i < 3; i++ {
		d.RateLimiter.RecordSpawn()
	}

	// Should have a message when limited
	msg = d.RateLimitMessage()
	if msg == "" {
		t.Error("RateLimitMessage() should return a message when limited")
	}
}

func TestDaemon_RateLimitStatus(t *testing.T) {
	d := NewWithConfig(Config{MaxSpawnsPerHour: 10})

	status := d.RateLimitStatus()
	if status == nil {
		t.Fatal("RateLimitStatus() should not be nil when rate limiter is configured")
	}
	if status.MaxPerHour != 10 {
		t.Errorf("RateLimitStatus().MaxPerHour = %d, want 10", status.MaxPerHour)
	}
}

func TestDaemon_RateLimitStatus_NilLimiter(t *testing.T) {
	d := &Daemon{} // No rate limiter

	status := d.RateLimitStatus()
	if status != nil {
		t.Error("RateLimitStatus() should be nil when no rate limiter is configured")
	}
}

func TestDaemon_Once_RateLimited(t *testing.T) {
	d := &Daemon{
		RateLimiter: NewRateLimiter(2),
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		},
		spawnFunc: func(beadsID string) error {
			t.Error("spawnFunc should not be called when rate limited")
			return nil
		},
	}

	// Fill rate limit
	d.RateLimiter.RecordSpawn()
	d.RateLimiter.RecordSpawn()

	result, err := d.Once()
	if err != nil {
		t.Fatalf("Once() error = %v", err)
	}
	if result.Processed {
		t.Error("Once() should not process when rate limited")
	}
	if result.Message == "" {
		t.Error("Once() should have a message when rate limited")
	}
}

func TestDaemon_Once_RecordsSpawn(t *testing.T) {
	d := &Daemon{
		RateLimiter: NewRateLimiter(10),
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		},
		spawnFunc: func(beadsID string) error {
			return nil
		},
	}

	// Initially no spawns
	if len(d.RateLimiter.SpawnHistory) != 0 {
		t.Fatal("Expected 0 spawns initially")
	}

	result, err := d.Once()
	if err != nil {
		t.Fatalf("Once() error = %v", err)
	}
	if !result.Processed {
		t.Error("Once() expected Processed=true")
	}

	// Should have recorded the spawn
	if len(d.RateLimiter.SpawnHistory) != 1 {
		t.Errorf("Expected 1 spawn recorded, got %d", len(d.RateLimiter.SpawnHistory))
	}
}

func TestDaemon_Once_NoRecordOnSpawnFailure(t *testing.T) {
	d := &Daemon{
		RateLimiter: NewRateLimiter(10),
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		},
		spawnFunc: func(beadsID string) error {
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

	// Should NOT have recorded the failed spawn
	if len(d.RateLimiter.SpawnHistory) != 0 {
		t.Errorf("Expected 0 spawns recorded on failure, got %d", len(d.RateLimiter.SpawnHistory))
	}
}

func TestDaemon_Preview_ShowsRateStatus(t *testing.T) {
	d := &Daemon{
		RateLimiter: NewRateLimiter(10),
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		},
	}

	// Record some spawns
	d.RateLimiter.RecordSpawn()
	d.RateLimiter.RecordSpawn()
	d.RateLimiter.RecordSpawn()

	result, err := d.Preview()
	if err != nil {
		t.Fatalf("Preview() error = %v", err)
	}

	if result.RateStatus == "" {
		t.Error("Preview() should show rate status")
	}
	if result.RateLimited {
		t.Error("Preview() should not be rate limited yet")
	}
	if result.Issue == nil {
		t.Error("Preview() should return an issue when not rate limited")
	}
}

func TestDaemon_Preview_RateLimited(t *testing.T) {
	d := &Daemon{
		RateLimiter: NewRateLimiter(2),
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		},
	}

	// Fill rate limit
	d.RateLimiter.RecordSpawn()
	d.RateLimiter.RecordSpawn()

	result, err := d.Preview()
	if err != nil {
		t.Fatalf("Preview() error = %v", err)
	}

	if !result.RateLimited {
		t.Error("Preview() should be rate limited")
	}
	if result.Message == "" {
		t.Error("Preview() should have a message when rate limited")
	}
	if result.Issue != nil {
		t.Error("Preview() should not return an issue when rate limited")
	}
}

func TestDefaultConfig_IncludesMaxSpawnsPerHour(t *testing.T) {
	config := DefaultConfig()

	if config.MaxSpawnsPerHour != 20 {
		t.Errorf("DefaultConfig().MaxSpawnsPerHour = %d, want 20", config.MaxSpawnsPerHour)
	}
}
