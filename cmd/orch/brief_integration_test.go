package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestBriefPipeline_EndToEnd verifies the full comprehension artifact pipeline:
// workspace BRIEF.md -> CopyBrief -> .kb/briefs/ -> API serves content -> mark-as-read.
func TestBriefPipeline_EndToEnd(t *testing.T) {
	// Setup: temporary project dir simulating workspace and .kb structure
	projectDir := t.TempDir()
	workspacePath := filepath.Join(projectDir, ".orch", "workspace", "og-feat-test-24mar-abcd")
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	briefContent := `## Frame

The daemon was delivering raw SYNTHESIS.md to Dylan, but synthesis is written for orchestrators.

## Resolution

We built BRIEF.md as a 3-section artifact (Frame/Resolution/Tension).

## Tension

We don't know yet whether agents will write genuinely provocative Tension sections.
`
	if err := os.WriteFile(filepath.Join(workspacePath, "BRIEF.md"), []byte(briefContent), 0644); err != nil {
		t.Fatalf("Failed to write BRIEF.md: %v", err)
	}

	beadsID := "orch-go-test1"

	// Step 1: CopyBrief (daemon delivery)
	adapter := &workspaceAdapter{}
	if err := adapter.CopyBrief(workspacePath, beadsID, projectDir); err != nil {
		t.Fatalf("CopyBrief() failed: %v", err)
	}

	// Verify file landed in .kb/briefs/
	briefPath := filepath.Join(projectDir, ".kb", "briefs", beadsID+".md")
	stored, err := os.ReadFile(briefPath)
	if err != nil {
		t.Fatalf("Brief not found at %s after CopyBrief: %v", briefPath, err)
	}
	if string(stored) != briefContent {
		t.Fatalf("Brief content mismatch after CopyBrief")
	}

	// Step 2: API serves brief content (GET)
	oldSourceDir := sourceDir
	defer func() { sourceDir = oldSourceDir }()
	sourceDir = projectDir

	req := httptest.NewRequest(http.MethodGet, "/api/briefs/"+beadsID, nil)
	w := httptest.NewRecorder()
	handleBrief(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/briefs/%s returned %d, want 200", beadsID, resp.StatusCode)
	}

	var briefResp BriefAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&briefResp); err != nil {
		t.Fatalf("Failed to decode brief response: %v", err)
	}
	if briefResp.BeadsID != beadsID {
		t.Errorf("beads_id: got %q, want %q", briefResp.BeadsID, beadsID)
	}
	if briefResp.Content != briefContent {
		t.Error("API content does not match original BRIEF.md")
	}
	if briefResp.MarkedRead {
		t.Error("Brief should not be marked_read before POST")
	}

	// Verify brief has all 3 required sections
	for _, section := range []string{"## Frame", "## Resolution", "## Tension"} {
		if !strings.Contains(briefResp.Content, section) {
			t.Errorf("Brief missing required section: %s", section)
		}
	}

	// Step 3: Mark as read (POST)
	reqPost := httptest.NewRequest(http.MethodPost, "/api/briefs/"+beadsID, nil)
	wPost := httptest.NewRecorder()
	handleBrief(wPost, reqPost)

	postResp := wPost.Result()
	if postResp.StatusCode != http.StatusOK {
		t.Fatalf("POST /api/briefs/%s returned %d, want 200", beadsID, postResp.StatusCode)
	}

	var markResp BriefMarkReadResponse
	if err := json.NewDecoder(postResp.Body).Decode(&markResp); err != nil {
		t.Fatalf("Failed to decode mark-read response: %v", err)
	}
	if !markResp.Success {
		t.Error("Mark-as-read should return success=true")
	}

	// Step 4: Verify marked_read persists on subsequent GET
	reqGet2 := httptest.NewRequest(http.MethodGet, "/api/briefs/"+beadsID, nil)
	wGet2 := httptest.NewRecorder()
	handleBrief(wGet2, reqGet2)

	var briefResp2 BriefAPIResponse
	if err := json.NewDecoder(wGet2.Result().Body).Decode(&briefResp2); err != nil {
		t.Fatalf("Failed to decode second GET response: %v", err)
	}
	if !briefResp2.MarkedRead {
		t.Error("Brief should be marked_read=true after POST")
	}

	// Step 5: hasBriefFile reports correctly
	if !hasBriefFile(beadsID) {
		t.Error("hasBriefFile should return true for existing brief")
	}
	if hasBriefFile("orch-go-nonexistent") {
		t.Error("hasBriefFile should return false for nonexistent brief")
	}
}

// TestBriefPipeline_NoBriefInWorkspace verifies graceful handling when
// an agent doesn't produce BRIEF.md (e.g., light-tier spawn).
func TestBriefPipeline_NoBriefInWorkspace(t *testing.T) {
	projectDir := t.TempDir()
	workspacePath := filepath.Join(projectDir, ".orch", "workspace", "og-feat-light-24mar-efgh")
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// No BRIEF.md in workspace — light tier spawn
	adapter := &workspaceAdapter{}
	err := adapter.CopyBrief(workspacePath, "orch-go-light1", projectDir)
	if err != nil {
		t.Fatalf("CopyBrief() should return nil for missing BRIEF.md, got: %v", err)
	}

	// .kb/briefs/ should NOT be created
	briefsDir := filepath.Join(projectDir, ".kb", "briefs")
	if _, err := os.Stat(briefsDir); !os.IsNotExist(err) {
		t.Error(".kb/briefs/ should not be created when no BRIEF.md exists")
	}

	// API should return 404
	oldSourceDir := sourceDir
	defer func() { sourceDir = oldSourceDir }()
	sourceDir = projectDir

	req := httptest.NewRequest(http.MethodGet, "/api/briefs/orch-go-light1", nil)
	w := httptest.NewRecorder()
	handleBrief(w, req)

	if w.Result().StatusCode != http.StatusNotFound {
		t.Errorf("Expected 404 for brief from light-tier agent, got %d", w.Result().StatusCode)
	}
}

// TestBriefPipeline_CopyThenClean verifies that stale brief cleanup works.
func TestBriefPipeline_CopyThenClean(t *testing.T) {
	projectDir := t.TempDir()
	briefsDir := filepath.Join(projectDir, ".kb", "briefs")

	adapter := &workspaceAdapter{}

	// Create two workspaces with briefs and copy them
	for _, id := range []string{"orch-go-old1", "orch-go-new1"} {
		wsPath := filepath.Join(projectDir, ".orch", "workspace", id)
		os.MkdirAll(wsPath, 0755)
		os.WriteFile(filepath.Join(wsPath, "BRIEF.md"), []byte("## Frame\nTest "+id), 0644)
		if err := adapter.CopyBrief(wsPath, id, projectDir); err != nil {
			t.Fatalf("CopyBrief(%s) failed: %v", id, err)
		}
	}

	// Backdate the "old" brief to 31 days ago
	oldPath := filepath.Join(briefsDir, "orch-go-old1.md")
	staleTime := time.Now().Add(-31 * 24 * time.Hour)
	os.Chtimes(oldPath, staleTime, staleTime)

	// Clean stale briefs
	if err := adapter.CleanStaleBriefs(projectDir, 30*24*time.Hour); err != nil {
		t.Fatalf("CleanStaleBriefs() failed: %v", err)
	}

	// Old brief removed, new brief remains
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Error("Stale brief was NOT removed by CleanStaleBriefs")
	}
	newPath := filepath.Join(briefsDir, "orch-go-new1.md")
	if _, err := os.Stat(newPath); err != nil {
		t.Error("Fresh brief was incorrectly removed by CleanStaleBriefs")
	}
}
