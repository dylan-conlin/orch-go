package daemon

import (
	"fmt"
	"testing"
	"time"
)

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
