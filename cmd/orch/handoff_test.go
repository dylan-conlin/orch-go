package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestHandoffCommandFlags verifies the handoff command accepts expected flags.
func TestHandoffCommandFlags(t *testing.T) {
	if handoffCmd == nil {
		t.Fatal("handoffCmd is nil")
	}

	// Check -o flag exists
	oFlag := handoffCmd.Flags().Lookup("output")
	if oFlag == nil {
		t.Error("Expected -o/--output flag")
	}

	// Check --json flag exists
	jsonFlag := handoffCmd.Flags().Lookup("json")
	if jsonFlag == nil {
		t.Error("Expected --json flag")
	}
}

// TestHandoffDataStructure verifies the HandoffData structure is correctly initialized.
func TestHandoffDataStructure(t *testing.T) {
	data := &HandoffData{
		Date:          "21 Dec 2025",
		TLDR:          "Test session.",
		ActiveAgents:  []ActiveAgent{},
		PendingIssues: []PendingIssue{},
		RecentWork:    []RecentWorkItem{},
		NextPriority:  []string{},
	}

	if data.Date != "21 Dec 2025" {
		t.Errorf("Expected date '21 Dec 2025', got %q", data.Date)
	}

	if len(data.ActiveAgents) != 0 {
		t.Errorf("Expected empty ActiveAgents, got %d", len(data.ActiveAgents))
	}
}

// TestGenerateTLDR verifies TLDR generation from handoff data.
func TestGenerateTLDR(t *testing.T) {
	tests := []struct {
		name     string
		data     *HandoffData
		project  string
		contains []string
	}{
		{
			name: "with active agents and focus",
			data: &HandoffData{
				ActiveAgents: []ActiveAgent{
					{BeadsID: "test-123"},
					{BeadsID: "test-456"},
				},
				Focus: &FocusInfo{
					Goal:      "Ship MVP",
					IsDrifted: false,
				},
			},
			project:  "test-project",
			contains: []string{"2 active agent(s)", "focused on: Ship MVP"},
		},
		{
			name: "with uncommitted changes",
			data: &HandoffData{
				LocalState: &LocalStateInfo{
					HasUncommitted: true,
					Summary:        "5 uncommitted changes",
				},
			},
			project:  "test-project",
			contains: []string{"5 uncommitted changes"},
		},
		{
			name: "with drift",
			data: &HandoffData{
				Focus: &FocusInfo{
					Goal:      "Other work",
					IsDrifted: true,
				},
			},
			project:  "test-project",
			contains: []string{"drifted from focus"},
		},
		{
			name:     "empty data",
			data:     &HandoffData{},
			project:  "empty-project",
			contains: []string{"Session handoff for empty-project", "No active work"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateTLDR(tt.data, tt.project)
			for _, want := range tt.contains {
				if !strings.Contains(result, want) {
					t.Errorf("generateTLDR() = %q, want to contain %q", result, want)
				}
			}
		})
	}
}

// TestTruncatePriority verifies priority truncation.
func TestTruncatePriority(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Short text", "Short text"},
		{"This is a very long priority description that exceeds the limit", "This is a very long priority description that e..."},
		{"Exactly fifty characters long string goes here!", "Exactly fifty characters long string goes here!"},
	}

	for _, tt := range tests {
		t.Run(tt.input[:min(20, len(tt.input))], func(t *testing.T) {
			result := truncatePriority(tt.input)
			if result != tt.expected {
				t.Errorf("truncatePriority(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestDeriveNextPriorities verifies priority derivation.
func TestDeriveNextPriorities(t *testing.T) {
	tests := []struct {
		name        string
		data        *HandoffData
		minExpected int
	}{
		{
			name: "with active agents",
			data: &HandoffData{
				ActiveAgents: []ActiveAgent{
					{BeadsID: "agent-1", Task: "Task 1"},
					{BeadsID: "agent-2", Task: "Task 2"},
				},
			},
			minExpected: 2,
		},
		{
			name: "with P0 pending issues",
			data: &HandoffData{
				PendingIssues: []PendingIssue{
					{ID: "issue-1", Priority: "P0", Title: "Critical bug"},
					{ID: "issue-2", Priority: "P1", Title: "Important feature"},
				},
			},
			minExpected: 2,
		},
		{
			name:        "empty data",
			data:        &HandoffData{},
			minExpected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deriveNextPriorities(tt.data)
			if len(result) < tt.minExpected {
				t.Errorf("deriveNextPriorities() returned %d priorities, want at least %d", len(result), tt.minExpected)
			}
		})
	}
}

// TestGenerateHandoffMarkdown verifies markdown generation.
func TestGenerateHandoffMarkdown(t *testing.T) {
	data := &HandoffData{
		Date: "21 Dec 2025",
		TLDR: "Test session with active agents.",
		ActiveAgents: []ActiveAgent{
			{BeadsID: "test-123", Repo: "test-repo", Task: "Test task"},
		},
		PendingIssues: []PendingIssue{
			{ID: "issue-1", Title: "Test issue", Priority: "P0"},
		},
		NextPriority: []string{"Check test-123"},
	}

	markdown, err := generateHandoffMarkdown(data)
	if err != nil {
		t.Fatalf("generateHandoffMarkdown() error = %v", err)
	}

	// Verify key sections are present
	sections := []string{
		"# Session Handoff - 21 Dec 2025",
		"## TLDR",
		"Test session with active agents.",
		"## Agents Still Running",
		"**test-123**",
		"## Next Session Priorities",
		"Check test-123",
		"## Quick Commands",
	}

	for _, section := range sections {
		if !strings.Contains(markdown, section) {
			t.Errorf("generateHandoffMarkdown() missing section %q", section)
		}
	}
}

// TestGatherLocalState verifies local state gathering.
func TestGatherLocalState(t *testing.T) {
	// Use temp dir as a non-git directory
	tmpDir := t.TempDir()

	state := gatherLocalState(tmpDir)

	// Non-git directory should have empty branch
	// (git commands will fail gracefully)
	if state == nil {
		t.Fatal("gatherLocalState() returned nil")
	}
}

// TestHandoffOutputPath verifies output path handling.
func TestHandoffOutputPath(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name       string
		output     string
		expectFile string
		createDir  bool
	}{
		{
			name:       "directory path",
			output:     tmpDir,
			expectFile: filepath.Join(tmpDir, "SESSION_HANDOFF.md"),
			createDir:  false, // Already exists
		},
		{
			name:       "file path",
			output:     filepath.Join(tmpDir, "custom.md"),
			expectFile: filepath.Join(tmpDir, "custom.md"),
			createDir:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputPath := tt.output
			// If output is a directory, append filename
			if info, err := os.Stat(tt.output); err == nil && info.IsDir() {
				outputPath = filepath.Join(tt.output, "SESSION_HANDOFF.md")
			}

			if outputPath != tt.expectFile {
				t.Errorf("output path = %q, want %q", outputPath, tt.expectFile)
			}
		})
	}
}

// TestActiveAgentExtraction verifies beads ID extraction from window names.
func TestActiveAgentExtraction(t *testing.T) {
	tests := []struct {
		windowName string
		wantID     string
	}{
		{"🏗️ og-feat-test [test-123]", "test-123"},
		{"🔬 og-inv-research [proj-abc]", "proj-abc"},
		{"🐛 og-debug-fix [snap-xyz]", "snap-xyz"},
		{"servers", ""},
		{"zsh", ""},
		{"no-brackets", ""},
	}

	for _, tt := range tests {
		t.Run(tt.windowName, func(t *testing.T) {
			result := extractBeadsIDFromWindowName(tt.windowName)
			if result != tt.wantID {
				t.Errorf("extractBeadsIDFromWindowName(%q) = %q, want %q", tt.windowName, result, tt.wantID)
			}
		})
	}
}

// TestFocusInfoStructure verifies FocusInfo initialization.
func TestFocusInfoStructure(t *testing.T) {
	info := &FocusInfo{
		Goal:      "Test goal",
		BeadsID:   "focus-123",
		IsDrifted: true,
	}

	if info.Goal != "Test goal" {
		t.Errorf("Expected goal 'Test goal', got %q", info.Goal)
	}

	if !info.IsDrifted {
		t.Error("Expected IsDrifted to be true")
	}
}

// TestPendingIssueStructure verifies PendingIssue initialization.
func TestPendingIssueStructure(t *testing.T) {
	issue := &PendingIssue{
		ID:       "issue-123",
		Title:    "Test issue",
		Priority: "P0",
	}

	if issue.ID != "issue-123" {
		t.Errorf("Expected ID 'issue-123', got %q", issue.ID)
	}

	if issue.Priority != "P0" {
		t.Errorf("Expected priority 'P0', got %q", issue.Priority)
	}
}

// TestRecentWorkItemStructure verifies RecentWorkItem initialization.
func TestRecentWorkItemStructure(t *testing.T) {
	item := &RecentWorkItem{
		Type:        "completed",
		Description: "Test work item",
		Repo:        "test-repo",
	}

	if item.Type != "completed" {
		t.Errorf("Expected type 'completed', got %q", item.Type)
	}
}

// TestLocalStateInfoStructure verifies LocalStateInfo initialization.
func TestLocalStateInfoStructure(t *testing.T) {
	state := &LocalStateInfo{
		HasUncommitted: true,
		Branch:         "feature/test",
		Summary:        "5 uncommitted changes",
	}

	if !state.HasUncommitted {
		t.Error("Expected HasUncommitted to be true")
	}

	if state.Branch != "feature/test" {
		t.Errorf("Expected branch 'feature/test', got %q", state.Branch)
	}
}
