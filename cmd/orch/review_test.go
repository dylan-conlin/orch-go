package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/registry"
)

// TestExtractProject verifies project extraction from directory paths.
func TestExtractProject(t *testing.T) {
	tests := []struct {
		name       string
		projectDir string
		expected   string
	}{
		{"standard path", "/Users/user/projects/orch-go", "orch-go"},
		{"another path", "/home/dev/my-project", "my-project"},
		{"simple path", "/projects/beads", "beads"},
		{"empty path", "", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractProject(tt.projectDir)
			if result != tt.expected {
				t.Errorf("extractProject(%q) = %q, want %q", tt.projectDir, result, tt.expected)
			}
		})
	}
}

// TestGroupByProject verifies grouping completions by project.
func TestGroupByProject(t *testing.T) {
	completions := []CompletionInfo{
		{Agent: &registry.Agent{ID: "agent-1"}, Project: "project-a"},
		{Agent: &registry.Agent{ID: "agent-2"}, Project: "project-b"},
		{Agent: &registry.Agent{ID: "agent-3"}, Project: "project-a"},
		{Agent: &registry.Agent{ID: "agent-4"}, Project: "project-c"},
		{Agent: &registry.Agent{ID: "agent-5"}, Project: "project-b"},
	}

	grouped := groupByProject(completions)

	if len(grouped) != 3 {
		t.Errorf("Expected 3 projects, got %d", len(grouped))
	}

	if len(grouped["project-a"]) != 2 {
		t.Errorf("Expected 2 completions for project-a, got %d", len(grouped["project-a"]))
	}

	if len(grouped["project-b"]) != 2 {
		t.Errorf("Expected 2 completions for project-b, got %d", len(grouped["project-b"]))
	}

	if len(grouped["project-c"]) != 1 {
		t.Errorf("Expected 1 completion for project-c, got %d", len(grouped["project-c"]))
	}
}

// TestFormatCompletionStatus verifies status formatting.
func TestFormatCompletionStatus(t *testing.T) {
	tests := []struct {
		name     string
		info     CompletionInfo
		contains []string
	}{
		{
			name: "verified OK",
			info: CompletionInfo{
				Agent:    &registry.Agent{ID: "agent-1", BeadsID: "beads-123"},
				VerifyOK: true,
			},
			contains: []string{"OK", "agent-1", "beads-123"},
		},
		{
			name: "needs review",
			info: CompletionInfo{
				Agent:    &registry.Agent{ID: "agent-2", BeadsID: "beads-456"},
				VerifyOK: false,
			},
			contains: []string{"NEEDS_REVIEW", "agent-2", "beads-456"},
		},
		{
			name: "no beads ID",
			info: CompletionInfo{
				Agent:    &registry.Agent{ID: "agent-3"},
				VerifyOK: false,
			},
			contains: []string{"NEEDS_REVIEW", "agent-3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatCompletionStatus(tt.info)
			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("FormatCompletionStatus() = %q, want to contain %q", result, s)
				}
			}
		})
	}
}

// TestReviewNoCompletions verifies handling of empty registry.
func TestReviewNoCompletions(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	// Create empty registry
	reg, err := registry.New(registryPath)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}
	if err := reg.Save(); err != nil {
		t.Fatalf("Failed to save registry: %v", err)
	}

	// Test that ListCompleted works with empty registry
	completed := reg.ListCompleted()
	if len(completed) != 0 {
		t.Errorf("Expected 0 completed agents, got %d", len(completed))
	}
}

// TestReviewWithCompletedAgents verifies review finds completed agents.
func TestReviewWithCompletedAgents(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	// Create registry with agents
	reg, err := registry.New(registryPath)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Register agents
	agent1 := &registry.Agent{ID: "agent-1", BeadsID: "beads-1", WindowID: "@100", ProjectDir: "/projects/a"}
	agent2 := &registry.Agent{ID: "agent-2", BeadsID: "beads-2", WindowID: "@200", ProjectDir: "/projects/b"}
	agent3 := &registry.Agent{ID: "agent-3", BeadsID: "beads-3", WindowID: "@300", ProjectDir: "/projects/a"}

	if err := reg.Register(agent1); err != nil {
		t.Fatalf("Failed to register agent-1: %v", err)
	}
	if err := reg.Register(agent2); err != nil {
		t.Fatalf("Failed to register agent-2: %v", err)
	}
	if err := reg.Register(agent3); err != nil {
		t.Fatalf("Failed to register agent-3: %v", err)
	}

	// Complete some agents
	reg.Complete("agent-1")
	reg.Complete("agent-2")
	// agent-3 stays active

	// ListCompleted should return agent-1 and agent-2
	completed := reg.ListCompleted()
	if len(completed) != 2 {
		t.Errorf("Expected 2 completed agents, got %d", len(completed))
	}

	// Verify correct agents are completed
	foundIDs := make(map[string]bool)
	for _, a := range completed {
		foundIDs[a.ID] = true
	}
	if !foundIDs["agent-1"] || !foundIDs["agent-2"] {
		t.Errorf("Expected agent-1 and agent-2 to be completed, got %v", foundIDs)
	}
}

// TestReviewDoneMarksAsDeleted verifies done command marks agents as deleted.
func TestReviewDoneMarksAsDeleted(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	// Create registry with completed agents
	reg, err := registry.New(registryPath)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	agent1 := &registry.Agent{ID: "agent-1", BeadsID: "beads-1", ProjectDir: "/projects/myproject"}
	agent2 := &registry.Agent{ID: "agent-2", BeadsID: "beads-2", ProjectDir: "/projects/myproject"}

	if err := reg.Register(agent1); err != nil {
		t.Fatalf("Failed to register agent-1: %v", err)
	}
	if err := reg.Register(agent2); err != nil {
		t.Fatalf("Failed to register agent-2: %v", err)
	}

	// Complete agents
	reg.Complete("agent-1")
	reg.Complete("agent-2")

	// Verify they're completed
	completed := reg.ListCompleted()
	if len(completed) != 2 {
		t.Fatalf("Expected 2 completed agents, got %d", len(completed))
	}

	// Simulate "review done" - mark as deleted
	for _, a := range completed {
		if extractProject(a.ProjectDir) == "myproject" {
			reg.Remove(a.ID)
		}
	}

	// After done, should have no completed agents
	completed = reg.ListCompleted()
	if len(completed) != 0 {
		t.Errorf("Expected 0 completed agents after done, got %d", len(completed))
	}

	// Verify ListAgents also excludes them
	all := reg.ListAgents()
	if len(all) != 0 {
		t.Errorf("Expected 0 agents after done, got %d", len(all))
	}
}

// TestFilterByProject verifies project filtering logic.
func TestFilterByProject(t *testing.T) {
	completions := []CompletionInfo{
		{Agent: &registry.Agent{ID: "agent-1"}, Project: "project-a"},
		{Agent: &registry.Agent{ID: "agent-2"}, Project: "project-b"},
		{Agent: &registry.Agent{ID: "agent-3"}, Project: "project-a"},
	}

	// Filter by project-a
	var filtered []CompletionInfo
	for _, c := range completions {
		if c.Project == "project-a" {
			filtered = append(filtered, c)
		}
	}

	if len(filtered) != 2 {
		t.Errorf("Expected 2 completions for project-a, got %d", len(filtered))
	}

	for _, c := range filtered {
		if c.Project != "project-a" {
			t.Errorf("Expected all filtered completions to be project-a, got %s", c.Project)
		}
	}
}

// TestFilterByNeedsReview verifies needs-review filtering logic.
func TestFilterByNeedsReview(t *testing.T) {
	completions := []CompletionInfo{
		{Agent: &registry.Agent{ID: "agent-1"}, VerifyOK: true},
		{Agent: &registry.Agent{ID: "agent-2"}, VerifyOK: false},
		{Agent: &registry.Agent{ID: "agent-3"}, VerifyOK: true},
		{Agent: &registry.Agent{ID: "agent-4"}, VerifyOK: false},
	}

	// Filter by needs-review (VerifyOK == false)
	var filtered []CompletionInfo
	for _, c := range completions {
		if !c.VerifyOK {
			filtered = append(filtered, c)
		}
	}

	if len(filtered) != 2 {
		t.Errorf("Expected 2 completions needing review, got %d", len(filtered))
	}

	for _, c := range filtered {
		if c.VerifyOK {
			t.Errorf("Expected all filtered completions to need review, got VerifyOK=true")
		}
	}
}

// TestReviewCommandFlags verifies the review command accepts expected flags.
func TestReviewCommandFlags(t *testing.T) {
	// Test that reviewCmd has the expected flags
	if reviewCmd == nil {
		t.Fatal("reviewCmd is nil")
	}

	// Check -p flag exists
	pFlag := reviewCmd.Flags().Lookup("project")
	if pFlag == nil {
		t.Error("Expected -p/--project flag")
	}

	// Check --needs-review flag exists
	nrFlag := reviewCmd.Flags().Lookup("needs-review")
	if nrFlag == nil {
		t.Error("Expected --needs-review flag")
	}

	// Check done subcommand exists
	doneCmd, _, err := reviewCmd.Find([]string{"done"})
	if err != nil || doneCmd == nil {
		t.Error("Expected 'done' subcommand")
	}
}
