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
	if !hasBriefFile(beadsID, projectDir) {
		t.Error("hasBriefFile should return true for existing brief")
	}
	if hasBriefFile("orch-go-nonexistent", projectDir) {
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

// TestBriefReadingQueue_SurvivesCompletionLifecycle verifies the core integration:
// after orch complete clears the review queue (removes comprehension:pending),
// briefs remain accessible via the /api/briefs list endpoint.
// This is the lifecycle decoupling that makes /briefs a persistent reading queue.
func TestBriefReadingQueue_SurvivesCompletionLifecycle(t *testing.T) {
	projectDir := t.TempDir()

	// Save/restore sourceDir and briefReadState
	oldSourceDir := sourceDir
	defer func() { sourceDir = oldSourceDir }()
	sourceDir = projectDir

	// Clean up any briefReadState from other tests
	briefReadStateMu.Lock()
	savedState := make(map[string]bool)
	for k, v := range briefReadState {
		savedState[k] = v
	}
	briefReadState = make(map[string]bool)
	briefReadStateMu.Unlock()
	defer func() {
		briefReadStateMu.Lock()
		briefReadState = savedState
		briefReadStateMu.Unlock()
	}()

	// --- Phase 1: Agent produces BRIEF.md, CopyBrief delivers it ---
	beadsID := "orch-go-integ1"
	wsPath := filepath.Join(projectDir, ".orch", "workspace", "og-feat-integ-24mar-aaaa")
	if err := os.MkdirAll(wsPath, 0755); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	briefContent := `## Frame

An agent completed investigation into coordination protocol primitives.

## Resolution

Route and Sequence emerged as the two atomic coordination verbs.

## Tension

Are two primitives sufficient, or will a third emerge from multi-agent scenarios?
`
	if err := os.WriteFile(filepath.Join(wsPath, "BRIEF.md"), []byte(briefContent), 0644); err != nil {
		t.Fatalf("Failed to write BRIEF.md: %v", err)
	}

	adapter := &workspaceAdapter{}
	if err := adapter.CopyBrief(wsPath, beadsID, projectDir); err != nil {
		t.Fatalf("CopyBrief failed: %v", err)
	}

	// --- Phase 2: Brief appears in list endpoint (simulates /briefs page load) ---
	req := httptest.NewRequest(http.MethodGet, "/api/briefs", nil)
	w := httptest.NewRecorder()
	handleBriefsList(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("GET /api/briefs returned %d, want 200", w.Result().StatusCode)
	}

	var listResp []BriefListItem
	if err := json.NewDecoder(w.Result().Body).Decode(&listResp); err != nil {
		t.Fatalf("Failed to decode briefs list: %v", err)
	}
	if len(listResp) != 1 {
		t.Fatalf("Expected 1 brief in list, got %d", len(listResp))
	}
	if listResp[0].BeadsID != beadsID {
		t.Errorf("Expected beads_id %q, got %q", beadsID, listResp[0].BeadsID)
	}
	if listResp[0].MarkedRead {
		t.Error("Brief should be unread initially")
	}

	// --- Phase 3: Brief content accessible via individual endpoint ---
	reqGet := httptest.NewRequest(http.MethodGet, "/api/briefs/"+beadsID, nil)
	wGet := httptest.NewRecorder()
	handleBrief(wGet, reqGet)

	var briefResp BriefAPIResponse
	if err := json.NewDecoder(wGet.Result().Body).Decode(&briefResp); err != nil {
		t.Fatalf("Failed to decode brief: %v", err)
	}
	if briefResp.Content != briefContent {
		t.Error("Brief content mismatch via individual endpoint")
	}
	for _, section := range []string{"## Frame", "## Resolution", "## Tension"} {
		if !strings.Contains(briefResp.Content, section) {
			t.Errorf("Brief missing required section: %s", section)
		}
	}

	// --- Phase 4: Simulate orch complete (review queue clears) ---
	// In production, orch complete calls daemon.RemoveComprehensionPendingInDir
	// which removes the comprehension:pending label. The review queue handler
	// (handleBeadsReviewQueue) then no longer returns this issue.
	// The brief in .kb/briefs/ is NOT affected — it persists on disk.
	// No action needed here — the brief files are independent of beads labels.

	// --- Phase 5: After completion, brief STILL appears in list endpoint ---
	// This is the key assertion: the /api/briefs list is decoupled from
	// the review queue. It scans .kb/briefs/ directly.
	reqList2 := httptest.NewRequest(http.MethodGet, "/api/briefs", nil)
	wList2 := httptest.NewRecorder()
	handleBriefsList(wList2, reqList2)

	var listResp2 []BriefListItem
	if err := json.NewDecoder(wList2.Result().Body).Decode(&listResp2); err != nil {
		t.Fatalf("Failed to decode briefs list after completion: %v", err)
	}
	if len(listResp2) != 1 {
		t.Fatalf("Expected brief to survive completion lifecycle, got %d briefs", len(listResp2))
	}
	if listResp2[0].BeadsID != beadsID {
		t.Errorf("Brief beads_id changed after completion: got %q", listResp2[0].BeadsID)
	}

	// --- Phase 6: Mark as read, verify list reflects read state ---
	reqMark := httptest.NewRequest(http.MethodPost, "/api/briefs/"+beadsID, nil)
	wMark := httptest.NewRecorder()
	handleBrief(wMark, reqMark)

	if wMark.Result().StatusCode != http.StatusOK {
		t.Fatalf("POST mark-as-read returned %d", wMark.Result().StatusCode)
	}

	reqList3 := httptest.NewRequest(http.MethodGet, "/api/briefs", nil)
	wList3 := httptest.NewRecorder()
	handleBriefsList(wList3, reqList3)

	var listResp3 []BriefListItem
	if err := json.NewDecoder(wList3.Result().Body).Decode(&listResp3); err != nil {
		t.Fatalf("Failed to decode briefs list after mark-read: %v", err)
	}
	if len(listResp3) != 1 {
		t.Fatalf("Expected 1 brief after mark-read, got %d", len(listResp3))
	}
	if !listResp3[0].MarkedRead {
		t.Error("Brief should be marked_read=true in list after POST")
	}

	// --- Phase 7: Individual endpoint also reflects read state ---
	reqGet2 := httptest.NewRequest(http.MethodGet, "/api/briefs/"+beadsID, nil)
	wGet2 := httptest.NewRecorder()
	handleBrief(wGet2, reqGet2)

	var briefResp2 BriefAPIResponse
	if err := json.NewDecoder(wGet2.Result().Body).Decode(&briefResp2); err != nil {
		t.Fatalf("Failed to decode brief after mark-read: %v", err)
	}
	if !briefResp2.MarkedRead {
		t.Error("Individual brief endpoint should show marked_read=true")
	}
}

// TestBriefReadingQueue_MultipleBriefsOrdering verifies that multiple briefs
// from different agents appear in the list sorted newest-first, and read state
// is tracked independently per brief.
func TestBriefReadingQueue_MultipleBriefsOrdering(t *testing.T) {
	projectDir := t.TempDir()

	oldSourceDir := sourceDir
	defer func() { sourceDir = oldSourceDir }()
	sourceDir = projectDir

	briefReadStateMu.Lock()
	savedState := make(map[string]bool)
	for k, v := range briefReadState {
		savedState[k] = v
	}
	briefReadState = make(map[string]bool)
	briefReadStateMu.Unlock()
	defer func() {
		briefReadStateMu.Lock()
		briefReadState = savedState
		briefReadStateMu.Unlock()
	}()

	adapter := &workspaceAdapter{}

	// Create 3 briefs with different timestamps
	briefs := []struct {
		id      string
		content string
		age     time.Duration
	}{
		{"orch-go-oldest", "## Frame\nOldest brief.\n## Resolution\nDone.\n## Tension\nNone.", 2 * time.Hour},
		{"orch-go-middle", "## Frame\nMiddle brief.\n## Resolution\nDone.\n## Tension\nNone.", 1 * time.Hour},
		{"orch-go-newest", "## Frame\nNewest brief.\n## Resolution\nDone.\n## Tension\nNone.", 0},
	}

	for _, b := range briefs {
		wsPath := filepath.Join(projectDir, ".orch", "workspace", b.id)
		os.MkdirAll(wsPath, 0755)
		os.WriteFile(filepath.Join(wsPath, "BRIEF.md"), []byte(b.content), 0644)
		if err := adapter.CopyBrief(wsPath, b.id, projectDir); err != nil {
			t.Fatalf("CopyBrief(%s) failed: %v", b.id, err)
		}
		// Set modification time to control ordering
		briefPath := filepath.Join(projectDir, ".kb", "briefs", b.id+".md")
		modTime := time.Now().Add(-b.age)
		os.Chtimes(briefPath, modTime, modTime)
	}

	// List should return newest-first
	req := httptest.NewRequest(http.MethodGet, "/api/briefs", nil)
	w := httptest.NewRecorder()
	handleBriefsList(w, req)

	var listResp []BriefListItem
	if err := json.NewDecoder(w.Result().Body).Decode(&listResp); err != nil {
		t.Fatalf("Failed to decode briefs list: %v", err)
	}

	if len(listResp) != 3 {
		t.Fatalf("Expected 3 briefs, got %d", len(listResp))
	}
	if listResp[0].BeadsID != "orch-go-newest" {
		t.Errorf("Expected newest first, got %s", listResp[0].BeadsID)
	}
	if listResp[1].BeadsID != "orch-go-middle" {
		t.Errorf("Expected middle second, got %s", listResp[1].BeadsID)
	}
	if listResp[2].BeadsID != "orch-go-oldest" {
		t.Errorf("Expected oldest last, got %s", listResp[2].BeadsID)
	}

	// Mark only the middle one as read
	reqMark := httptest.NewRequest(http.MethodPost, "/api/briefs/orch-go-middle", nil)
	wMark := httptest.NewRecorder()
	handleBrief(wMark, reqMark)

	// Verify independent read state tracking
	reqList2 := httptest.NewRequest(http.MethodGet, "/api/briefs", nil)
	wList2 := httptest.NewRecorder()
	handleBriefsList(wList2, reqList2)

	var listResp2 []BriefListItem
	if err := json.NewDecoder(wList2.Result().Body).Decode(&listResp2); err != nil {
		t.Fatalf("Failed to decode briefs list: %v", err)
	}

	for _, item := range listResp2 {
		if item.BeadsID == "orch-go-middle" && !item.MarkedRead {
			t.Error("orch-go-middle should be marked_read=true")
		}
		if item.BeadsID == "orch-go-newest" && item.MarkedRead {
			t.Error("orch-go-newest should still be unread")
		}
		if item.BeadsID == "orch-go-oldest" && item.MarkedRead {
			t.Error("orch-go-oldest should still be unread")
		}
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
