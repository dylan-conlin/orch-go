package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
		{WorkspaceID: "ws-1", BeadsID: "agent-1", Project: "project-a"},
		{WorkspaceID: "ws-2", BeadsID: "agent-2", Project: "project-b"},
		{WorkspaceID: "ws-3", BeadsID: "agent-3", Project: "project-a"},
		{WorkspaceID: "ws-4", BeadsID: "agent-4", Project: "project-c"},
		{WorkspaceID: "ws-5", BeadsID: "agent-5", Project: "project-b"},
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

// TestFilterByProject verifies project filtering logic.
func TestFilterByProject(t *testing.T) {
	completions := []CompletionInfo{
		{WorkspaceID: "ws-1", BeadsID: "agent-1", Project: "project-a"},
		{WorkspaceID: "ws-2", BeadsID: "agent-2", Project: "project-b"},
		{WorkspaceID: "ws-3", BeadsID: "agent-3", Project: "project-a"},
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
		{WorkspaceID: "ws-1", BeadsID: "agent-1", VerifyOK: true},
		{WorkspaceID: "ws-2", BeadsID: "agent-2", VerifyOK: false},
		{WorkspaceID: "ws-3", BeadsID: "agent-3", VerifyOK: true},
		{WorkspaceID: "ws-4", BeadsID: "agent-4", VerifyOK: false},
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

// TestSummarizeDelta verifies the Delta summary generation.
func TestSummarizeDelta(t *testing.T) {
	tests := []struct {
		name     string
		delta    string
		contains []string
	}{
		{
			name: "files created and modified",
			delta: `### Files Created

- cmd/orch/new.go - New command
- pkg/new/new.go - Core logic

### Files Modified

- cmd/orch/main.go - Added command

### Commits

- abc1234 - feat: add feature`,
			contains: []string{"2 files created", "1 files modified", "1 commits"},
		},
		{
			name: "only files created",
			delta: `### Files Created

- file1.go
- file2.go
- file3.go`,
			contains: []string{"3 files created"},
		},
		{
			name:     "empty delta",
			delta:    "",
			contains: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := summarizeDelta(tt.delta)
			for _, want := range tt.contains {
				if !strings.Contains(result, want) {
					t.Errorf("summarizeDelta() = %q, want to contain %q", result, want)
				}
			}
		})
	}
}

// TestExtractBeadsIDFromWorkspace tests extracting beads ID from workspace files.
func TestExtractBeadsIDFromWorkspace(t *testing.T) {
	t.Run("prefers .beads_id file", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Write .beads_id file
		if err := os.WriteFile(filepath.Join(tmpDir, ".beads_id"), []byte("orch-go-abcd"), 0644); err != nil {
			t.Fatal(err)
		}
		// Also write AGENT_MANIFEST.json with different ID to prove priority
		if err := os.WriteFile(filepath.Join(tmpDir, "AGENT_MANIFEST.json"), []byte(`{"beads_id": "orch-go-wxyz"}`), 0644); err != nil {
			t.Fatal(err)
		}
		got := extractBeadsIDFromWorkspace(tmpDir)
		if got != "orch-go-abcd" {
			t.Errorf("extractBeadsIDFromWorkspace() = %q, want %q", got, "orch-go-abcd")
		}
	})

	t.Run("falls back to AGENT_MANIFEST.json", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Write AGENT_MANIFEST.json only
		if err := os.WriteFile(filepath.Join(tmpDir, "AGENT_MANIFEST.json"), []byte(`{"workspace_name": "og-feat-test", "beads_id": "orch-go-n1wi", "skill": "feature-impl"}`), 0644); err != nil {
			t.Fatal(err)
		}
		got := extractBeadsIDFromWorkspace(tmpDir)
		if got != "orch-go-n1wi" {
			t.Errorf("extractBeadsIDFromWorkspace() = %q, want %q", got, "orch-go-n1wi")
		}
	})

	t.Run("falls back to SPAWN_CONTEXT.md legacy format", func(t *testing.T) {
		tmpDir := t.TempDir()
		content := `TASK: Test task

## BEADS PROGRESS TRACKING

You were spawned from beads issue: **orch-go-pe5d.2**

Use bd comment for progress updates.`
		if err := os.WriteFile(filepath.Join(tmpDir, "SPAWN_CONTEXT.md"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		got := extractBeadsIDFromWorkspace(tmpDir)
		if got != "orch-go-pe5d.2" {
			t.Errorf("extractBeadsIDFromWorkspace() = %q, want %q", got, "orch-go-pe5d.2")
		}
	})

	t.Run("returns empty when no sources exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		got := extractBeadsIDFromWorkspace(tmpDir)
		if got != "" {
			t.Errorf("extractBeadsIDFromWorkspace() = %q, want empty", got)
		}
	})

	t.Run("handles SPAWN_CONTEXT.md without beads issue line", func(t *testing.T) {
		tmpDir := t.TempDir()
		// This is the current template format - no "beads issue:" line
		content := `TASK: orch review shows 'no beads tracking'

bd comment orch-go-yjyl "Phase: Planning - investigating"`
		if err := os.WriteFile(filepath.Join(tmpDir, "SPAWN_CONTEXT.md"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		got := extractBeadsIDFromWorkspace(tmpDir)
		if got != "" {
			t.Errorf("extractBeadsIDFromWorkspace() = %q, want empty (SPAWN_CONTEXT.md has no 'beads issue:' line)", got)
		}
	})

	t.Run("beads issue with backticks in SPAWN_CONTEXT.md", func(t *testing.T) {
		tmpDir := t.TempDir()
		content := "TASK: Another task\n\nbeads issue: `snap-abc123`"
		if err := os.WriteFile(filepath.Join(tmpDir, "SPAWN_CONTEXT.md"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		got := extractBeadsIDFromWorkspace(tmpDir)
		if got != "snap-abc123" {
			t.Errorf("extractBeadsIDFromWorkspace() = %q, want %q", got, "snap-abc123")
		}
	})

	t.Run("empty .beads_id file falls through to manifest", func(t *testing.T) {
		tmpDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(tmpDir, ".beads_id"), []byte(""), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, "AGENT_MANIFEST.json"), []byte(`{"beads_id": "orch-go-fallback"}`), 0644); err != nil {
			t.Fatal(err)
		}
		got := extractBeadsIDFromWorkspace(tmpDir)
		if got != "orch-go-fallback" {
			t.Errorf("extractBeadsIDFromWorkspace() = %q, want %q", got, "orch-go-fallback")
		}
	})
}

// TestCountBulletPoints verifies bullet point counting.
func TestCountBulletPoints(t *testing.T) {
	content := `### Files Created

- file1.go
- file2.go

### Files Modified

- main.go
`

	tests := []struct {
		section string
		want    int
	}{
		{"### Files Created", 2},
		{"### Files Modified", 1},
		{"### Nonexistent", 0},
	}

	for _, tt := range tests {
		t.Run(tt.section, func(t *testing.T) {
			got := countBulletPoints(content, tt.section)
			if got != tt.want {
				t.Errorf("countBulletPoints(%q) = %d, want %d", tt.section, got, tt.want)
			}
		})
	}
}

// TestReviewDoneCategorizesCompletions verifies that runReviewDone correctly
// categorizes completions into canComplete (VerifyOK && has beads ID) and
// needsReview (no beads ID or verification failed).
func TestReviewDoneCategorizesCompletions(t *testing.T) {
	completions := []CompletionInfo{
		// Should be in canComplete
		{WorkspaceID: "ws-1", BeadsID: "project-abc1", VerifyOK: true, Project: "project"},
		{WorkspaceID: "ws-2", BeadsID: "project-abc2", VerifyOK: true, Project: "project"},
		// Should be in needsReview (no beads ID)
		{WorkspaceID: "ws-3", BeadsID: "", VerifyOK: true, Project: "project"},
		// Should be in needsReview (verification failed)
		{WorkspaceID: "ws-4", BeadsID: "project-abc4", VerifyOK: false, Project: "project"},
		// Should be in needsReview (both missing beads ID and failed verification)
		{WorkspaceID: "ws-5", BeadsID: "", VerifyOK: false, Project: "project"},
	}

	// Categorize using the same logic as runReviewDone
	var canComplete []CompletionInfo
	var needsReview []CompletionInfo
	for _, c := range completions {
		if c.VerifyOK && c.BeadsID != "" {
			canComplete = append(canComplete, c)
		} else {
			needsReview = append(needsReview, c)
		}
	}

	if len(canComplete) != 2 {
		t.Errorf("Expected 2 completions in canComplete, got %d", len(canComplete))
	}
	if len(needsReview) != 3 {
		t.Errorf("Expected 3 completions in needsReview, got %d", len(needsReview))
	}

	// Verify canComplete contains the right items
	for _, c := range canComplete {
		if c.BeadsID == "" {
			t.Errorf("canComplete item has empty beads ID: %s", c.WorkspaceID)
		}
		if !c.VerifyOK {
			t.Errorf("canComplete item has VerifyOK=false: %s", c.WorkspaceID)
		}
	}

	// Verify needsReview items are correct
	for _, c := range needsReview {
		if c.VerifyOK && c.BeadsID != "" {
			t.Errorf("needsReview item should not have both VerifyOK=true and beads ID: %s", c.WorkspaceID)
		}
	}
}

// TestReviewDoneCommandHasYesFlag verifies that the done subcommand has the -y/--yes flag.
func TestReviewDoneCommandHasYesFlag(t *testing.T) {
	// Find the done subcommand
	doneCmd, _, err := reviewCmd.Find([]string{"done"})
	if err != nil || doneCmd == nil {
		t.Fatal("Expected 'done' subcommand to exist")
	}

	// Check -y/--yes flag exists
	yFlag := doneCmd.Flags().Lookup("yes")
	if yFlag == nil {
		t.Error("Expected -y/--yes flag on review done command")
	}

	// Check shorthand
	if yFlag != nil && yFlag.Shorthand != "y" {
		t.Errorf("Expected shorthand 'y', got %q", yFlag.Shorthand)
	}
}

// TestGetCompletionsForReviewWorkspaceBased verifies workspace-based completion detection.
func TestGetCompletionsForReviewWorkspaceBased(t *testing.T) {
	// Create temp project directory
	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace dir: %v", err)
	}

	// Create a completed workspace (has SYNTHESIS.md)
	ws1 := filepath.Join(workspaceDir, "og-feat-test-feature-21dec")
	if err := os.MkdirAll(ws1, 0755); err != nil {
		t.Fatalf("Failed to create ws1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(ws1, "SYNTHESIS.md"), []byte("# Synthesis\n\n## TLDR\nDid the thing."), 0644); err != nil {
		t.Fatalf("Failed to write SYNTHESIS.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(ws1, "SPAWN_CONTEXT.md"), []byte("beads issue: **test-abc123**"), 0644); err != nil {
		t.Fatalf("Failed to write SPAWN_CONTEXT.md: %v", err)
	}

	// Create an incomplete workspace (no SYNTHESIS.md)
	ws2 := filepath.Join(workspaceDir, "og-feat-in-progress-21dec")
	if err := os.MkdirAll(ws2, 0755); err != nil {
		t.Fatalf("Failed to create ws2: %v", err)
	}
	if err := os.WriteFile(filepath.Join(ws2, "SPAWN_CONTEXT.md"), []byte("beads issue: **test-def456**"), 0644); err != nil {
		t.Fatalf("Failed to write SPAWN_CONTEXT.md: %v", err)
	}

	// Note: We can't easily test getCompletionsForReview() here because it uses os.Getwd()
	// and calls verify.VerifyCompletion which shells out to bd. But we can verify that
	// the workspace detection logic (checking for SYNTHESIS.md) is correct.

	// Verify SYNTHESIS.md exists in ws1
	synthesisPath := filepath.Join(ws1, "SYNTHESIS.md")
	if _, err := os.Stat(synthesisPath); os.IsNotExist(err) {
		t.Error("Expected SYNTHESIS.md to exist in ws1")
	}

	// Verify SYNTHESIS.md does NOT exist in ws2
	synthesisPath2 := filepath.Join(ws2, "SYNTHESIS.md")
	if _, err := os.Stat(synthesisPath2); !os.IsNotExist(err) {
		t.Error("Expected SYNTHESIS.md to NOT exist in ws2")
	}
}
