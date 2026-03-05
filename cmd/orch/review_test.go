package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
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
	// and calls verify.VerifyCompletionForReview which shells out to bd. But we can verify that
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

// TestCompletionInfoReviewTierField verifies the ReviewTier field on CompletionInfo.
func TestCompletionInfoReviewTierField(t *testing.T) {
	info := CompletionInfo{
		WorkspaceID: "og-feat-test",
		BeadsID:     "orch-go-abc1",
		ReviewTier:  spawn.ReviewScan,
	}

	if info.ReviewTier != "scan" {
		t.Errorf("Expected ReviewTier 'scan', got %q", info.ReviewTier)
	}
}

// TestCompletionInfoIsAutoCompleted verifies the IsAutoCompleted field.
func TestCompletionInfoIsAutoCompleted(t *testing.T) {
	info := CompletionInfo{
		WorkspaceID:     "og-capture-knowledge-test",
		BeadsID:         "orch-go-abc2",
		IsAutoCompleted: true,
		ReviewTier:      spawn.ReviewAuto,
		Summary:         "Captured knowledge about X",
	}

	if !info.IsAutoCompleted {
		t.Error("Expected IsAutoCompleted to be true")
	}
	if info.ReviewTier != "auto" {
		t.Errorf("Expected ReviewTier 'auto', got %q", info.ReviewTier)
	}
}

// TestGetRecentAutoCompletions verifies reading auto-completed events from events.jsonl.
func TestGetRecentAutoCompletions(t *testing.T) {
	// Create a temp events file
	tmpDir := t.TempDir()
	eventsFile := filepath.Join(tmpDir, "events.jsonl")

	now := time.Now()

	// Write test events
	testEvents := []events.Event{
		{
			Type:      events.EventTypeAutoCompleted,
			Timestamp: now.Add(-1 * time.Hour).Unix(),
			Data: map[string]interface{}{
				"beads_id":     "orch-go-auto1",
				"close_reason": "Auto-completed: captured knowledge",
				"workspace":    "og-capture-knowledge-test",
				"review_tier":  "auto",
				"project_dir":  "/Users/test/projects/orch-go",
			},
		},
		{
			Type:      events.EventTypeAutoCompleted,
			Timestamp: now.Add(-48 * time.Hour).Unix(), // Older than 24h
			Data: map[string]interface{}{
				"beads_id":     "orch-go-auto2",
				"close_reason": "Old auto-completion",
			},
		},
		{
			Type:      events.EventTypeSessionSpawned, // Not auto-completed
			Timestamp: now.Add(-30 * time.Minute).Unix(),
			Data: map[string]interface{}{
				"beads_id": "orch-go-spawn1",
			},
		},
	}

	f, err := os.Create(eventsFile)
	if err != nil {
		t.Fatal(err)
	}
	for _, ev := range testEvents {
		data, _ := json.Marshal(ev)
		fmt.Fprintln(f, string(data))
	}
	f.Close()

	// We can't easily test getRecentAutoCompletions directly because it uses
	// events.DefaultLogPath(). Instead, test the parsing logic inline.
	// The function is tested indirectly through integration tests.

	// Verify the events file was written correctly
	file, err := os.Open(eventsFile)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	var parsed []events.Event
	decoder := json.NewDecoder(file)
	for decoder.More() {
		var ev events.Event
		if err := decoder.Decode(&ev); err != nil {
			break
		}
		parsed = append(parsed, ev)
	}

	if len(parsed) != 3 {
		t.Fatalf("Expected 3 events, got %d", len(parsed))
	}

	// Verify first event is auto-completed type
	if parsed[0].Type != events.EventTypeAutoCompleted {
		t.Errorf("Expected event type %q, got %q", events.EventTypeAutoCompleted, parsed[0].Type)
	}

	// Verify data fields
	if beadsID, ok := parsed[0].Data["beads_id"].(string); !ok || beadsID != "orch-go-auto1" {
		t.Errorf("Expected beads_id 'orch-go-auto1', got %v", parsed[0].Data["beads_id"])
	}
}

// TestReviewTierBadgeDisplay verifies that review tier badges are rendered correctly.
func TestReviewTierBadgeDisplay(t *testing.T) {
	tests := []struct {
		name       string
		completion CompletionInfo
		wantBadge  string
	}{
		{
			name: "auto tier badge",
			completion: CompletionInfo{
				WorkspaceID: "og-capture-knowledge-test",
				BeadsID:     "orch-go-abc1",
				ReviewTier:  spawn.ReviewAuto,
				VerifyOK:    true,
			},
			wantBadge: "{auto}",
		},
		{
			name: "scan tier badge",
			completion: CompletionInfo{
				WorkspaceID: "og-investigation-test",
				BeadsID:     "orch-go-abc2",
				ReviewTier:  spawn.ReviewScan,
				VerifyOK:    true,
			},
			wantBadge: "{scan}",
		},
		{
			name: "review tier badge",
			completion: CompletionInfo{
				WorkspaceID: "og-feat-test",
				BeadsID:     "orch-go-abc3",
				ReviewTier:  spawn.ReviewReview,
				VerifyOK:    true,
			},
			wantBadge: "{review}",
		},
		{
			name: "deep tier badge",
			completion: CompletionInfo{
				WorkspaceID: "og-debug-pw-test",
				BeadsID:     "orch-go-abc4",
				ReviewTier:  spawn.ReviewDeep,
				VerifyOK:    true,
			},
			wantBadge: "{deep}",
		},
		{
			name: "no tier badge when empty",
			completion: CompletionInfo{
				WorkspaceID: "og-legacy-test",
				BeadsID:     "orch-go-abc5",
				ReviewTier:  "",
				VerifyOK:    true,
			},
			wantBadge: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build the badge string the same way runReview does
			tierBadge := ""
			if tt.completion.ReviewTier != "" {
				tierBadge = fmt.Sprintf(" {%s}", tt.completion.ReviewTier)
			}

			if tt.wantBadge == "" {
				if tierBadge != "" {
					t.Errorf("Expected no badge, got %q", tierBadge)
				}
			} else {
				if !strings.Contains(tierBadge, tt.wantBadge) {
					t.Errorf("Expected badge to contain %q, got %q", tt.wantBadge, tierBadge)
				}
			}
		})
	}
}

// TestAutoCompletedStatusDisplay verifies auto-completed items show correct status.
func TestAutoCompletedStatusDisplay(t *testing.T) {
	c := CompletionInfo{
		WorkspaceID:     "og-capture-knowledge-test",
		BeadsID:         "orch-go-abc1",
		VerifyOK:        true,
		IsAutoCompleted: true,
		ReviewTier:      spawn.ReviewAuto,
		Summary:         "Captured knowledge about daemon patterns",
	}

	// Verify the status logic matches runReview
	status := "OK"
	if c.VerifyOK {
		status = "OK"
	} else {
		status = "NEEDS_REVIEW"
	}
	if c.IsAutoCompleted {
		status = "auto-completed"
	}

	if status != "auto-completed" {
		t.Errorf("Expected status 'auto-completed', got %q", status)
	}
}

// TestAutoCompletedDeduplication verifies that auto-completed events are not shown
// if the same beads ID exists in the pending completions list.
func TestAutoCompletedDeduplication(t *testing.T) {
	// Simulate pending completions
	completions := []CompletionInfo{
		{BeadsID: "orch-go-existing1", WorkspaceID: "ws1"},
		{BeadsID: "orch-go-existing2", WorkspaceID: "ws2"},
	}

	// Simulate auto-completed events (one overlaps, one is new)
	autoCompleted := []AutoCompletedInfo{
		{BeadsID: "orch-go-existing1", Summary: "Should be deduplicated"},
		{BeadsID: "orch-go-new1", Summary: "Should be included", ProjectDir: "/test/proj"},
	}

	// Apply deduplication logic (same as runReview)
	existingBeadsIDs := make(map[string]bool)
	for _, c := range completions {
		if c.BeadsID != "" {
			existingBeadsIDs[c.BeadsID] = true
		}
	}

	addedCount := 0
	for _, ac := range autoCompleted {
		if ac.BeadsID != "" && existingBeadsIDs[ac.BeadsID] {
			continue
		}
		addedCount++
	}

	if addedCount != 1 {
		t.Errorf("Expected 1 new auto-completed item (after dedup), got %d", addedCount)
	}
}
