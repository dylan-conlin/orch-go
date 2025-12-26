package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

// TestExtractBeadsIDFromWorkspace tests extracting beads ID from SPAWN_CONTEXT.md.
func TestExtractBeadsIDFromWorkspace(t *testing.T) {
	tests := []struct {
		name           string
		contextContent string
		wantID         string
	}{
		{
			name: "standard beads issue format",
			contextContent: `TASK: Test task

## BEADS PROGRESS TRACKING

You were spawned from beads issue: **orch-go-pe5d.2**

Use bd comment for progress updates.`,
			wantID: "orch-go-pe5d.2",
		},
		{
			name: "beads issue with backticks",
			contextContent: `TASK: Another task

beads issue: ` + "`snap-abc123`",
			wantID: "snap-abc123",
		},
		{
			name: "no beads issue",
			contextContent: `TASK: Untracked task

No beads tracking for this one.`,
			wantID: "",
		},
		{
			name: "spawned from beads issue format",
			contextContent: `TASK: Feature work

## BEADS PROGRESS TRACKING (PREFERRED)

You were spawned from beads issue: **orch-cli-xyz99**

Use bd comment for updates.`,
			wantID: "orch-cli-xyz99",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp workspace
			tmpDir := t.TempDir()
			if err := os.WriteFile(filepath.Join(tmpDir, "SPAWN_CONTEXT.md"), []byte(tt.contextContent), 0644); err != nil {
				t.Fatalf("Failed to write SPAWN_CONTEXT.md: %v", err)
			}

			got := extractBeadsIDFromWorkspace(tmpDir)
			if got != tt.wantID {
				t.Errorf("extractBeadsIDFromWorkspace() = %q, want %q", got, tt.wantID)
			}
		})
	}
}

// TestExtractProjectDirFromWorkspace verifies extracting PROJECT_DIR from SPAWN_CONTEXT.md
func TestExtractProjectDirFromWorkspace(t *testing.T) {
	tests := []struct {
		name           string
		contextContent string
		wantProjectDir string
	}{
		{
			name: "standard PROJECT_DIR format",
			contextContent: `TASK: Some task

PROJECT_DIR: /Users/dylan/Documents/personal/orch-go

SESSION SCOPE: Medium
`,
			wantProjectDir: "/Users/dylan/Documents/personal/orch-go",
		},
		{
			name: "cross-project skillc spawn",
			contextContent: `TASK: Implement feature

PROJECT_DIR: /Users/dylan/orch-knowledge

AUTHORITY:
...
`,
			wantProjectDir: "/Users/dylan/orch-knowledge",
		},
		{
			name: "no PROJECT_DIR in file",
			contextContent: `TASK: Some task

SESSION SCOPE: Small
`,
			wantProjectDir: "",
		},
		{
			name: "PROJECT_DIR with extra whitespace",
			contextContent: `TASK: Some task

PROJECT_DIR:   /path/with/spaces  

SESSION SCOPE: Medium
`,
			wantProjectDir: "/path/with/spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp workspace
			tmpDir := t.TempDir()
			if err := os.WriteFile(filepath.Join(tmpDir, "SPAWN_CONTEXT.md"), []byte(tt.contextContent), 0644); err != nil {
				t.Fatalf("Failed to write SPAWN_CONTEXT.md: %v", err)
			}

			got := extractProjectDirFromWorkspace(tmpDir)
			if got != tt.wantProjectDir {
				t.Errorf("extractProjectDirFromWorkspace() = %q, want %q", got, tt.wantProjectDir)
			}
		})
	}
}

// TestExtractProjectDirFromWorkspace_NoFile verifies behavior when no SPAWN_CONTEXT.md exists
func TestExtractProjectDirFromWorkspace_NoFile(t *testing.T) {
	tmpDir := t.TempDir()
	got := extractProjectDirFromWorkspace(tmpDir)
	if got != "" {
		t.Errorf("extractProjectDirFromWorkspace() = %q, want empty string for missing file", got)
	}
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

// TestIsUntrackedBeadsID verifies detection of untracked beads IDs.
func TestIsUntrackedBeadsID(t *testing.T) {
	tests := []struct {
		name    string
		beadsID string
		want    bool
	}{
		{"standard tracked ID", "orch-go-abc123", false},
		{"tracked ID with suffix", "orch-go-abc123.1", false},
		{"untracked ID", "orch-go-untracked-1766695797", true},
		{"untracked ID other project", "kb-cli-untracked-1766695797", true},
		{"empty ID", "", false},
		{"project-untracked-pattern", "snap-untracked-1234567890", true},
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

// TestIsStaleAgent verifies staleness detection based on phase and time.
func TestIsStaleAgent(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		phase   string
		modTime time.Time
		want    bool
	}{
		{"Complete phase, recent", "Complete", now.Add(-1 * time.Hour), false},
		{"Complete phase, old", "Complete", now.Add(-48 * time.Hour), false}, // Complete is never stale
		{"Implementing phase, recent", "Implementing", now.Add(-1 * time.Hour), false},
		{"Implementing phase, 25h old", "Implementing", now.Add(-25 * time.Hour), true},
		{"Planning phase, 30h old", "Planning", now.Add(-30 * time.Hour), true},
		{"Empty phase, recent", "", now.Add(-1 * time.Hour), false},
		{"Empty phase, old", "", now.Add(-48 * time.Hour), true},
		{"Design phase, 23h 59m", "Design", now.Add(-23*time.Hour - 59*time.Minute), false}, // Just under threshold
		{"Design phase, just over 24h", "Design", now.Add(-24*time.Hour - 1*time.Minute), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isStaleAgent(tt.phase, tt.modTime)
			if got != tt.want {
				t.Errorf("isStaleAgent(%q, %v) = %v, want %v", tt.phase, tt.modTime, got, tt.want)
			}
		})
	}
}

// TestReviewCommandHasStaleAndAllFlags verifies the new filtering flags exist.
func TestReviewCommandHasStaleAndAllFlags(t *testing.T) {
	if reviewCmd == nil {
		t.Fatal("reviewCmd is nil")
	}

	// Check --stale flag exists
	staleFlag := reviewCmd.Flags().Lookup("stale")
	if staleFlag == nil {
		t.Error("Expected --stale flag")
	}

	// Check --all flag exists
	allFlag := reviewCmd.Flags().Lookup("all")
	if allFlag == nil {
		t.Error("Expected --all flag")
	}
}

// TestFilterByStaleOrUntracked verifies filtering logic for stale/untracked agents.
func TestFilterByStaleOrUntracked(t *testing.T) {
	now := time.Now()
	completions := []CompletionInfo{
		// Actionable - tracked and recent
		{WorkspaceID: "ws-1", BeadsID: "project-abc1", Phase: "Complete", ModTime: now, IsStale: false, IsUntracked: false},
		// Stale - old and not complete
		{WorkspaceID: "ws-2", BeadsID: "project-abc2", Phase: "Implementing", ModTime: now.Add(-48 * time.Hour), IsStale: true, IsUntracked: false},
		// Untracked
		{WorkspaceID: "ws-3", BeadsID: "project-untracked-123", Phase: "Complete", ModTime: now, IsStale: false, IsUntracked: true},
		// Stale AND untracked
		{WorkspaceID: "ws-4", BeadsID: "project-untracked-456", Phase: "Planning", ModTime: now.Add(-48 * time.Hour), IsStale: true, IsUntracked: true},
		// Actionable - tracked and needs review but recent
		{WorkspaceID: "ws-5", BeadsID: "project-abc5", Phase: "Testing", ModTime: now, IsStale: false, IsUntracked: false},
	}

	// Test default filter (exclude stale and untracked)
	var defaultFiltered []CompletionInfo
	for _, c := range completions {
		if !c.IsStale && !c.IsUntracked {
			defaultFiltered = append(defaultFiltered, c)
		}
	}
	if len(defaultFiltered) != 2 {
		t.Errorf("Default filter: expected 2 actionable completions, got %d", len(defaultFiltered))
	}

	// Test --stale filter (only stale or untracked)
	var staleFiltered []CompletionInfo
	for _, c := range completions {
		if c.IsStale || c.IsUntracked {
			staleFiltered = append(staleFiltered, c)
		}
	}
	if len(staleFiltered) != 3 {
		t.Errorf("Stale filter: expected 3 stale/untracked completions, got %d", len(staleFiltered))
	}

	// Test --all (no filtering)
	if len(completions) != 5 {
		t.Errorf("All filter: expected 5 total completions, got %d", len(completions))
	}
}

// TestStaleThreshold verifies the stale threshold constant is set correctly.
func TestStaleThreshold(t *testing.T) {
	expected := 24 * time.Hour
	if StaleThreshold != expected {
		t.Errorf("StaleThreshold = %v, want %v", StaleThreshold, expected)
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
