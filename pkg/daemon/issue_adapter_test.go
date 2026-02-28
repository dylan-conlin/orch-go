package daemon

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

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

func TestFindWorkspaceForIssue_NoWorkspaceDir(t *testing.T) {
	// When workspace dir doesn't exist, should return empty string
	result := findWorkspaceForIssue("proj-123", "/nonexistent/path", "")
	if result != "" {
		t.Errorf("findWorkspaceForIssue() = %q, want empty string for nonexistent dir", result)
	}
}

func TestFindWorkspaceForIssue_PrefersSynthesis(t *testing.T) {
	// When duplicate workspaces exist for the same beads ID,
	// should prefer the one with SYNTHESIS.md
	tmpDir := t.TempDir()
	wsDir := filepath.Join(tmpDir, ".orch", "workspace")

	// Create two workspaces that both reference the same beads ID
	ws1 := filepath.Join(wsDir, "og-feat-task-27feb-aaaa")
	ws2 := filepath.Join(wsDir, "og-feat-task-28feb-bbbb")
	if err := os.MkdirAll(ws1, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(ws2, 0755); err != nil {
		t.Fatal(err)
	}

	// Both have SPAWN_CONTEXT.md referencing same beads ID
	ctx := "bd comment proj-abc \"Phase: Planning\"\n"
	os.WriteFile(filepath.Join(ws1, "SPAWN_CONTEXT.md"), []byte(ctx), 0644)
	os.WriteFile(filepath.Join(ws2, "SPAWN_CONTEXT.md"), []byte(ctx), 0644)

	// Only ws2 has SYNTHESIS.md (completed work)
	os.WriteFile(filepath.Join(ws2, "SYNTHESIS.md"), []byte("# Synthesis\n"), 0644)

	// ws1 has older spawn time, ws2 has newer
	os.WriteFile(filepath.Join(ws1, ".spawn_time"), []byte("1000000000"), 0644)
	os.WriteFile(filepath.Join(ws2, ".spawn_time"), []byte("2000000000"), 0644)

	result := findWorkspaceForIssue("proj-abc", wsDir, "")
	if result != ws2 {
		t.Errorf("findWorkspaceForIssue() = %q, want %q (workspace with SYNTHESIS.md)", result, ws2)
	}
}

func TestFindWorkspaceForIssue_PrefersNewestWhenNoSynthesis(t *testing.T) {
	// When neither workspace has SYNTHESIS.md, prefer most recent spawn time
	tmpDir := t.TempDir()
	wsDir := filepath.Join(tmpDir, ".orch", "workspace")

	ws1 := filepath.Join(wsDir, "og-feat-task-27feb-aaaa")
	ws2 := filepath.Join(wsDir, "og-feat-task-28feb-bbbb")
	os.MkdirAll(ws1, 0755)
	os.MkdirAll(ws2, 0755)

	ctx := "bd comment proj-abc \"Phase: Planning\"\n"
	os.WriteFile(filepath.Join(ws1, "SPAWN_CONTEXT.md"), []byte(ctx), 0644)
	os.WriteFile(filepath.Join(ws2, "SPAWN_CONTEXT.md"), []byte(ctx), 0644)

	// ws1 is older, ws2 is newer
	os.WriteFile(filepath.Join(ws1, ".spawn_time"), []byte("1000000000"), 0644)
	os.WriteFile(filepath.Join(ws2, ".spawn_time"), []byte("2000000000"), 0644)

	result := findWorkspaceForIssue("proj-abc", wsDir, "")
	if result != ws2 {
		t.Errorf("findWorkspaceForIssue() = %q, want %q (most recent workspace)", result, ws2)
	}
}

func TestFindWorkspaceForIssue_SynthesisBeatsNewer(t *testing.T) {
	// SYNTHESIS.md should win even if it's on the older workspace
	tmpDir := t.TempDir()
	wsDir := filepath.Join(tmpDir, ".orch", "workspace")

	wsOld := filepath.Join(wsDir, "og-feat-task-27feb-aaaa")
	wsNew := filepath.Join(wsDir, "og-feat-task-28feb-bbbb")
	os.MkdirAll(wsOld, 0755)
	os.MkdirAll(wsNew, 0755)

	ctx := "bd comment proj-abc \"Phase: Planning\"\n"
	os.WriteFile(filepath.Join(wsOld, "SPAWN_CONTEXT.md"), []byte(ctx), 0644)
	os.WriteFile(filepath.Join(wsNew, "SPAWN_CONTEXT.md"), []byte(ctx), 0644)

	// Old workspace has SYNTHESIS.md, new one doesn't
	os.WriteFile(filepath.Join(wsOld, "SYNTHESIS.md"), []byte("# Done\n"), 0644)
	os.WriteFile(filepath.Join(wsOld, ".spawn_time"), []byte("1000000000"), 0644)
	os.WriteFile(filepath.Join(wsNew, ".spawn_time"), []byte("2000000000"), 0644)

	result := findWorkspaceForIssue("proj-abc", wsDir, "")
	if result != wsOld {
		t.Errorf("findWorkspaceForIssue() = %q, want %q (workspace with SYNTHESIS.md wins over newer)", result, wsOld)
	}
}

func TestPickBestWorkspacePath(t *testing.T) {
	tmpDir := t.TempDir()

	// Create 3 workspace directories
	ws1 := filepath.Join(tmpDir, "ws1")
	ws2 := filepath.Join(tmpDir, "ws2")
	ws3 := filepath.Join(tmpDir, "ws3")
	os.MkdirAll(ws1, 0755)
	os.MkdirAll(ws2, 0755)
	os.MkdirAll(ws3, 0755)

	// ws1: no synthesis, oldest
	os.WriteFile(filepath.Join(ws1, ".spawn_time"), []byte("1000"), 0644)
	// ws2: has synthesis, middle
	os.WriteFile(filepath.Join(ws2, ".spawn_time"), []byte("2000"), 0644)
	os.WriteFile(filepath.Join(ws2, "SYNTHESIS.md"), []byte("done"), 0644)
	// ws3: no synthesis, newest
	os.WriteFile(filepath.Join(ws3, ".spawn_time"), []byte("3000"), 0644)

	result := pickBestWorkspacePath([]string{ws1, ws2, ws3})
	if result != ws2 {
		t.Errorf("pickBestWorkspacePath() = %q, want %q (SYNTHESIS.md wins)", result, ws2)
	}
}

func TestReadSpawnTime(t *testing.T) {
	tmpDir := t.TempDir()

	// Valid spawn time
	os.WriteFile(filepath.Join(tmpDir, ".spawn_time"), []byte("1772217838171079000\n"), 0644)
	got := readSpawnTime(tmpDir)
	if got != 1772217838171079000 {
		t.Errorf("readSpawnTime() = %d, want 1772217838171079000", got)
	}

	// Missing file returns 0
	got = readSpawnTime(filepath.Join(tmpDir, "nonexistent"))
	if got != 0 {
		t.Errorf("readSpawnTime() for missing dir = %d, want 0", got)
	}
}
