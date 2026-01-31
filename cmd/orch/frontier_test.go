package main

import (
	"os"
	"testing"
	"time"
)

func TestFilterOpenIssueAgents(t *testing.T) {
	tests := []struct {
		name    string
		agents  []ActiveOutput
		wantLen int
		wantIDs []string
	}{
		{
			name:    "empty list",
			agents:  []ActiveOutput{},
			wantLen: 0,
			wantIDs: nil,
		},
		{
			name: "no beads IDs",
			agents: []ActiveOutput{
				{BeadsID: "", Runtime: "1h"},
			},
			wantLen: 1, // Agents without beads IDs are kept
			wantIDs: []string{""},
		},
		{
			name: "all agents have beads IDs",
			agents: []ActiveOutput{
				{BeadsID: "test-1", Runtime: "1h"},
				{BeadsID: "test-2", Runtime: "2h"},
			},
			// These will be filtered based on actual beads status
			// In unit test context without mocking, behavior depends on
			// whether bd command is available and returns results
			wantLen: -1, // Skip length check since it depends on external command
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterOpenIssueAgents(tt.agents)
			if tt.wantLen >= 0 && len(result) != tt.wantLen {
				t.Errorf("filterOpenIssueAgents() returned %d agents, want %d", len(result), tt.wantLen)
			}
			if tt.wantIDs != nil {
				gotIDs := make([]string, len(result))
				for i, a := range result {
					gotIDs[i] = a.BeadsID
				}
				for i, wantID := range tt.wantIDs {
					if i < len(gotIDs) && gotIDs[i] != wantID {
						t.Errorf("filterOpenIssueAgents() agent %d has ID %q, want %q", i, gotIDs[i], wantID)
					}
				}
			}
		})
	}
}

func TestFilterOpenIssueAgents_PreservesNonBeadsAgents(t *testing.T) {
	// Agents without beads IDs should always be preserved
	agents := []ActiveOutput{
		{BeadsID: "", Runtime: "tmux", Skill: "test"},
		{BeadsID: "", Runtime: "1h", Skill: "another"},
	}

	result := filterOpenIssueAgents(agents)

	if len(result) != len(agents) {
		t.Errorf("filterOpenIssueAgents() returned %d agents, want %d (should preserve non-beads agents)", len(result), len(agents))
	}
}

func TestFilterOpenIssueAgents_PreservesAllFields(t *testing.T) {
	// Verify all fields are preserved through filtering
	agent := ActiveOutput{
		BeadsID:  "",
		Title:    "Test Title",
		Runtime:  "30m",
		Duration: 30 * time.Minute,
		Skill:    "test-skill",
	}

	result := filterOpenIssueAgents([]ActiveOutput{agent})

	if len(result) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(result))
	}

	got := result[0]
	if got.Title != agent.Title {
		t.Errorf("Title = %q, want %q", got.Title, agent.Title)
	}
	if got.Runtime != agent.Runtime {
		t.Errorf("Runtime = %q, want %q", got.Runtime, agent.Runtime)
	}
	if got.Duration != agent.Duration {
		t.Errorf("Duration = %v, want %v", got.Duration, agent.Duration)
	}
	if got.Skill != agent.Skill {
		t.Errorf("Skill = %q, want %q", got.Skill, agent.Skill)
	}
}

func TestGetClosedIssueIDs_EmptyInput(t *testing.T) {
	result := getClosedIssueIDs([]string{})
	if len(result) != 0 {
		t.Errorf("getClosedIssueIDs([]) = map with %d entries, want empty map", len(result))
	}
}

func TestGetClosedIssueIDs_NilInput(t *testing.T) {
	result := getClosedIssueIDs(nil)
	if result == nil {
		t.Error("getClosedIssueIDs(nil) returned nil, want non-nil empty map")
	}
	if len(result) != 0 {
		t.Errorf("getClosedIssueIDs(nil) = map with %d entries, want empty map", len(result))
	}
}

// Note: Integration tests that actually call bd would go in a separate file
// or require mocking the exec.Command call.

func TestFrontierCmdHasWorkdirFlag(t *testing.T) {
	// Verify the --workdir flag is registered on the frontier command
	flag := frontierCmd.Flags().Lookup("workdir")
	if flag == nil {
		t.Fatal("frontier command should have a --workdir flag")
	}
	if flag.DefValue != "" {
		t.Errorf("--workdir default value should be empty string, got %q", flag.DefValue)
	}
}

func TestFrontierWorkdirResolution(t *testing.T) {
	// Save and restore original value
	originalWorkdir := frontierWorkdir
	defer func() {
		frontierWorkdir = originalWorkdir
	}()

	// Test that an invalid workdir path returns an error
	frontierWorkdir = "/nonexistent/path/that/does/not/exist"

	// The runFrontier function should validate the workdir before proceeding
	// We can't fully test runFrontier without mocking bd commands,
	// but we can verify the workdir is validated
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}

	// Test resolveProjectDir with invalid path
	_, err = resolveProjectDir(frontierWorkdir, "", currentDir)
	if err == nil {
		t.Error("resolveProjectDir should return error for nonexistent workdir")
	}

	// Test resolveProjectDir with valid path (current directory)
	frontierWorkdir = currentDir
	result, err := resolveProjectDir(frontierWorkdir, "", currentDir)
	if err != nil {
		t.Fatalf("resolveProjectDir should succeed for valid path: %v", err)
	}
	if result.ProjectDir != currentDir {
		t.Errorf("ProjectDir = %q, want %q", result.ProjectDir, currentDir)
	}
	if result.Source != "workdir" {
		t.Errorf("Source = %q, want %q", result.Source, "workdir")
	}
}
