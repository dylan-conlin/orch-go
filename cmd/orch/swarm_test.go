package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/daemon"
)

func TestSwarmProgress(t *testing.T) {
	t.Run("initial state", func(t *testing.T) {
		progress := &SwarmProgress{
			Total:   5,
			Results: make([]SwarmResult, 0),
		}

		if progress.Total != 5 {
			t.Errorf("Expected total=5, got %d", progress.Total)
		}
		if progress.Spawned != 0 {
			t.Errorf("Expected spawned=0, got %d", progress.Spawned)
		}
		if progress.Active != 0 {
			t.Errorf("Expected active=0, got %d", progress.Active)
		}
		if progress.Completed != 0 {
			t.Errorf("Expected completed=0, got %d", progress.Completed)
		}
		if progress.Failed != 0 {
			t.Errorf("Expected failed=0, got %d", progress.Failed)
		}
	})

	t.Run("add spawned increments active", func(t *testing.T) {
		progress := &SwarmProgress{
			Total:   5,
			Results: make([]SwarmResult, 0),
		}

		progress.AddSpawned()

		if progress.Spawned != 1 {
			t.Errorf("Expected spawned=1, got %d", progress.Spawned)
		}
		if progress.Active != 1 {
			t.Errorf("Expected active=1, got %d", progress.Active)
		}
	})

	t.Run("add completed decrements active", func(t *testing.T) {
		progress := &SwarmProgress{
			Total:   5,
			Results: make([]SwarmResult, 0),
		}
		progress.AddSpawned()
		progress.AddSpawned()

		// Complete one successfully
		progress.AddCompleted(SwarmResult{
			IssueID: "test-1",
			Skill:   "investigation",
		})

		if progress.Active != 1 {
			t.Errorf("Expected active=1, got %d", progress.Active)
		}
		if progress.Completed != 1 {
			t.Errorf("Expected completed=1, got %d", progress.Completed)
		}
		if progress.Failed != 0 {
			t.Errorf("Expected failed=0, got %d", progress.Failed)
		}
	})

	t.Run("add failed increments failed counter", func(t *testing.T) {
		progress := &SwarmProgress{
			Total:   5,
			Results: make([]SwarmResult, 0),
		}
		progress.AddSpawned()

		// Complete with error
		progress.AddCompleted(SwarmResult{
			IssueID: "test-1",
			Error:   fmt.Errorf("spawn failed"),
		})

		if progress.Active != 0 {
			t.Errorf("Expected active=0, got %d", progress.Active)
		}
		if progress.Completed != 0 {
			t.Errorf("Expected completed=0, got %d", progress.Completed)
		}
		if progress.Failed != 1 {
			t.Errorf("Expected failed=1, got %d", progress.Failed)
		}
	})

	t.Run("String format", func(t *testing.T) {
		progress := &SwarmProgress{
			Total:     10,
			Spawned:   5,
			Active:    3,
			Completed: 2,
			Failed:    0,
			Results:   make([]SwarmResult, 0),
		}

		str := progress.String()
		expected := "Progress: 5 spawned / 3 active / 2 completed / 0 failed (of 10)"
		if str != expected {
			t.Errorf("Expected %q, got %q", expected, str)
		}
	})
}

func TestSwarmAgentTracker(t *testing.T) {
	t.Run("tracker fields", func(t *testing.T) {
		tracker := swarmAgentTracker{
			IssueID:   "test-123",
			Skill:     "feature-impl",
			SessionID: "ses_abc123",
			SpawnTime: time.Now(),
		}

		if tracker.IssueID != "test-123" {
			t.Errorf("Expected IssueID=test-123, got %s", tracker.IssueID)
		}
		if tracker.Skill != "feature-impl" {
			t.Errorf("Expected Skill=feature-impl, got %s", tracker.Skill)
		}
	})
}

func TestExtractSessionIDFromOutput(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected string
	}{
		{
			name: "standard output format",
			output: `Spawned agent in tmux:
  Session:    workers-orch-go
  Session ID: ses_abc123def456
  Window:     workers-orch-go:2`,
			expected: "ses_abc123def456",
		},
		{
			name: "with extra whitespace",
			output: `Spawned agent:
  Session ID:   ses_xyz789
  Workspace:  test`,
			expected: "ses_xyz789",
		},
		{
			name:     "no session ID in output",
			output:   "Spawned agent in tmux:\n  Window: workers:1",
			expected: "",
		},
		{
			name:     "empty output",
			output:   "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSessionIDFromOutput(tt.output)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestPrintSwarmDryRun(t *testing.T) {
	// Test that dry-run doesn't panic with various inputs
	t.Run("empty issues", func(t *testing.T) {
		issues := []daemon.Issue{}
		err := printSwarmDryRun(issues)
		if err != nil {
			t.Errorf("Expected no error for empty issues, got: %v", err)
		}
	})

	t.Run("single issue", func(t *testing.T) {
		issues := []daemon.Issue{
			{
				ID:        "test-123",
				Title:     "Test issue",
				IssueType: "feature",
				Status:    "open",
			},
		}
		err := printSwarmDryRun(issues)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("multiple issues", func(t *testing.T) {
		issues := []daemon.Issue{
			{
				ID:        "test-1",
				Title:     "Feature 1",
				IssueType: "feature",
				Status:    "open",
			},
			{
				ID:        "test-2",
				Title:     "Bug 1",
				IssueType: "bug",
				Status:    "open",
			},
			{
				ID:        "test-3",
				Title:     "Investigation 1",
				IssueType: "investigation",
				Status:    "open",
			},
		}
		err := printSwarmDryRun(issues)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})
}
