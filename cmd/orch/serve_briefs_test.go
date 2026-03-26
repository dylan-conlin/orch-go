package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHandleBriefsList(t *testing.T) {
	oldSourceDir := sourceDir
	defer func() { sourceDir = oldSourceDir }()
	sourceDir = t.TempDir()

	briefsDir := filepath.Join(sourceDir, ".kb", "briefs")
	if err := os.MkdirAll(briefsDir, 0755); err != nil {
		t.Fatalf("Failed to create briefs dir: %v", err)
	}

	// Create two brief files with different mod times
	brief1 := filepath.Join(briefsDir, "orch-go-aaa11.md")
	if err := os.WriteFile(brief1, []byte("brief 1"), 0644); err != nil {
		t.Fatalf("Failed to write brief: %v", err)
	}
	brief2 := filepath.Join(briefsDir, "orch-go-bbb22.md")
	if err := os.WriteFile(brief2, []byte("brief 2"), 0644); err != nil {
		t.Fatalf("Failed to write brief: %v", err)
	}

	// Mark one as read (keyed by project+beadsID)
	readKey := briefReadKey(sourceDir, "orch-go-aaa11")
	briefReadStateMu.Lock()
	briefReadState[readKey] = true
	briefReadStateMu.Unlock()
	defer func() {
		briefReadStateMu.Lock()
		delete(briefReadState, readKey)
		briefReadStateMu.Unlock()
	}()

	req := httptest.NewRequest(http.MethodGet, "/api/briefs", nil)
	w := httptest.NewRecorder()

	handleBriefsList(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var items []BriefListItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("Expected 2 briefs, got %d", len(items))
	}

	// Newest first — bbb22 was written after aaa11
	if items[0].BeadsID != "orch-go-bbb22" {
		t.Errorf("Expected first item to be orch-go-bbb22, got %s", items[0].BeadsID)
	}
	if items[0].MarkedRead {
		t.Error("Expected orch-go-bbb22 to not be marked read")
	}
	if items[1].BeadsID != "orch-go-aaa11" {
		t.Errorf("Expected second item to be orch-go-aaa11, got %s", items[1].BeadsID)
	}
	if !items[1].MarkedRead {
		t.Error("Expected orch-go-aaa11 to be marked read")
	}
}

func TestHandleBriefsListEmptyDir(t *testing.T) {
	oldSourceDir := sourceDir
	defer func() { sourceDir = oldSourceDir }()
	sourceDir = t.TempDir()

	// No .kb/briefs/ directory
	req := httptest.NewRequest(http.MethodGet, "/api/briefs", nil)
	w := httptest.NewRecorder()

	handleBriefsList(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var items []BriefListItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(items) != 0 {
		t.Errorf("Expected 0 briefs, got %d", len(items))
	}
}

func TestHandleBriefNotFound(t *testing.T) {
	// Save and restore sourceDir
	oldSourceDir := sourceDir
	defer func() { sourceDir = oldSourceDir }()
	sourceDir = t.TempDir()

	req := httptest.NewRequest(http.MethodGet, "/api/briefs/nonexistent-id", nil)
	w := httptest.NewRecorder()

	handleBrief(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestHandleBriefSuccess(t *testing.T) {
	oldSourceDir := sourceDir
	defer func() { sourceDir = oldSourceDir }()
	sourceDir = t.TempDir()

	// Create .kb/briefs/ directory with a brief file
	briefsDir := filepath.Join(sourceDir, ".kb", "briefs")
	if err := os.MkdirAll(briefsDir, 0755); err != nil {
		t.Fatalf("Failed to create briefs dir: %v", err)
	}

	briefContent := `## Frame

The daemon needed a way to deliver comprehension artifacts asynchronously.

## Resolution

We built BRIEF.md as a 3-section artifact produced by agents at completion time.

## Tension

Does mark-as-read actually create comprehension, or just a feeling of comprehension?
`
	briefPath := filepath.Join(briefsDir, "orch-go-abc12.md")
	if err := os.WriteFile(briefPath, []byte(briefContent), 0644); err != nil {
		t.Fatalf("Failed to write brief: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/briefs/orch-go-abc12", nil)
	w := httptest.NewRecorder()

	handleBrief(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var briefResp BriefAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&briefResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if briefResp.BeadsID != "orch-go-abc12" {
		t.Errorf("Expected beads_id 'orch-go-abc12', got '%s'", briefResp.BeadsID)
	}
	if briefResp.Content != briefContent {
		t.Errorf("Expected brief content to match, got '%s'", briefResp.Content)
	}
	if briefResp.MarkedRead {
		t.Error("Expected marked_read to be false initially")
	}
}

func TestHandleBriefMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/api/briefs/test-id", nil)
	w := httptest.NewRecorder()

	handleBrief(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestHandleBriefMarkAsRead(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // isolate state file writes
	oldSourceDir := sourceDir
	defer func() { sourceDir = oldSourceDir }()
	sourceDir = t.TempDir()

	// Create brief file
	briefsDir := filepath.Join(sourceDir, ".kb", "briefs")
	if err := os.MkdirAll(briefsDir, 0755); err != nil {
		t.Fatalf("Failed to create briefs dir: %v", err)
	}
	briefPath := filepath.Join(briefsDir, "orch-go-xyz99.md")
	if err := os.WriteFile(briefPath, []byte("## Frame\nTest brief."), 0644); err != nil {
		t.Fatalf("Failed to write brief: %v", err)
	}

	// POST to mark as read
	req := httptest.NewRequest(http.MethodPost, "/api/briefs/orch-go-xyz99", nil)
	w := httptest.NewRecorder()

	handleBrief(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var markResp BriefMarkReadResponse
	if err := json.NewDecoder(resp.Body).Decode(&markResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if !markResp.Success {
		t.Error("Expected success to be true")
	}

	// Now GET should show marked_read = true
	req2 := httptest.NewRequest(http.MethodGet, "/api/briefs/orch-go-xyz99", nil)
	w2 := httptest.NewRecorder()

	handleBrief(w2, req2)

	var briefResp BriefAPIResponse
	if err := json.NewDecoder(w2.Result().Body).Decode(&briefResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if !briefResp.MarkedRead {
		t.Error("Expected marked_read to be true after marking as read")
	}
}

func TestHandleBriefMarkAsReadNotFound(t *testing.T) {
	oldSourceDir := sourceDir
	defer func() { sourceDir = oldSourceDir }()
	sourceDir = t.TempDir()

	req := httptest.NewRequest(http.MethodPost, "/api/briefs/nonexistent-id", nil)
	w := httptest.NewRecorder()

	handleBrief(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestHandleBriefPathTraversal(t *testing.T) {
	oldSourceDir := sourceDir
	defer func() { sourceDir = oldSourceDir }()
	sourceDir = t.TempDir()

	// Attempt path traversal
	req := httptest.NewRequest(http.MethodGet, "/api/briefs/../../../etc/passwd", nil)
	w := httptest.NewRecorder()

	handleBrief(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400 for path traversal, got %d", resp.StatusCode)
	}
}

func TestReviewQueueHasBriefField(t *testing.T) {
	oldSourceDir := sourceDir
	defer func() { sourceDir = oldSourceDir }()
	sourceDir = t.TempDir()

	// Create .kb/briefs/ directory with a brief
	briefsDir := filepath.Join(sourceDir, ".kb", "briefs")
	if err := os.MkdirAll(briefsDir, 0755); err != nil {
		t.Fatalf("Failed to create briefs dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(briefsDir, "test-id-1.md"), []byte("brief"), 0644); err != nil {
		t.Fatalf("Failed to write brief: %v", err)
	}

	// Test hasBriefFile helper (uses sourceDir when projectDir is empty)
	if !hasBriefFile("test-id-1", "") {
		t.Error("Expected hasBriefFile to return true for existing brief")
	}
	if hasBriefFile("nonexistent-id", "") {
		t.Error("Expected hasBriefFile to return false for nonexistent brief")
	}
	// Also test with explicit projectDir
	if !hasBriefFile("test-id-1", sourceDir) {
		t.Error("Expected hasBriefFile to return true with explicit projectDir")
	}
}

func TestHandleBriefsListWithProjectDir(t *testing.T) {
	// Create two separate project dirs with different briefs
	projectA := t.TempDir()
	projectB := t.TempDir()

	briefsDirA := filepath.Join(projectA, ".kb", "briefs")
	briefsDirB := filepath.Join(projectB, ".kb", "briefs")
	os.MkdirAll(briefsDirA, 0755)
	os.MkdirAll(briefsDirB, 0755)

	os.WriteFile(filepath.Join(briefsDirA, "proj-a-001.md"), []byte("brief A"), 0644)
	os.WriteFile(filepath.Join(briefsDirB, "proj-b-001.md"), []byte("brief B"), 0644)

	// Query project A
	req := httptest.NewRequest(http.MethodGet, "/api/briefs?project_dir="+projectA, nil)
	w := httptest.NewRecorder()
	handleBriefsList(w, req)

	var itemsA []BriefListItem
	json.NewDecoder(w.Result().Body).Decode(&itemsA)
	if len(itemsA) != 1 || itemsA[0].BeadsID != "proj-a-001" {
		t.Errorf("Expected 1 brief from project A (proj-a-001), got %v", itemsA)
	}

	// Query project B
	req2 := httptest.NewRequest(http.MethodGet, "/api/briefs?project_dir="+projectB, nil)
	w2 := httptest.NewRecorder()
	handleBriefsList(w2, req2)

	var itemsB []BriefListItem
	json.NewDecoder(w2.Result().Body).Decode(&itemsB)
	if len(itemsB) != 1 || itemsB[0].BeadsID != "proj-b-001" {
		t.Errorf("Expected 1 brief from project B (proj-b-001), got %v", itemsB)
	}
}

func TestHandleBriefsListEnrichedFields(t *testing.T) {
	oldSourceDir := sourceDir
	defer func() { sourceDir = oldSourceDir }()
	sourceDir = t.TempDir()

	// Create briefs directory
	briefsDir := filepath.Join(sourceDir, ".kb", "briefs")
	if err := os.MkdirAll(briefsDir, 0755); err != nil {
		t.Fatalf("Failed to create briefs dir: %v", err)
	}

	// Create threads directory with a thread that references a beads ID
	threadsDir := filepath.Join(sourceDir, ".kb", "threads")
	if err := os.MkdirAll(threadsDir, 0755); err != nil {
		t.Fatalf("Failed to create threads dir: %v", err)
	}

	// Brief WITH tension section
	briefWithTension := `# Brief: orch-go-aaa11

## Frame

Something was investigated.

## Resolution

It was resolved.

## Tension

But this question remains open.
`
	if err := os.WriteFile(filepath.Join(briefsDir, "orch-go-aaa11.md"), []byte(briefWithTension), 0644); err != nil {
		t.Fatalf("Failed to write brief: %v", err)
	}

	// Brief WITHOUT tension section
	briefNoTension := `# Brief: orch-go-bbb22

## Frame

Something simple.

## Resolution

Done.
`
	if err := os.WriteFile(filepath.Join(briefsDir, "orch-go-bbb22.md"), []byte(briefNoTension), 0644); err != nil {
		t.Fatalf("Failed to write brief: %v", err)
	}

	// Thread with active_work referencing orch-go-aaa11
	threadContent := `---
title: "Test Thread"
status: forming
created: 2026-03-26
updated: 2026-03-26
resolved_to: ""
active_work:
  - "orch-go-aaa11"
---

# Test Thread

## 2026-03-26

Entry about something.
`
	if err := os.WriteFile(filepath.Join(threadsDir, "2026-03-26-test-thread.md"), []byte(threadContent), 0644); err != nil {
		t.Fatalf("Failed to write thread: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/briefs", nil)
	w := httptest.NewRecorder()

	handleBriefsList(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var items []BriefListItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("Expected 2 briefs, got %d", len(items))
	}

	// Find items by beads_id (order depends on mod time)
	var aaa, bbb *BriefListItem
	for i := range items {
		switch items[i].BeadsID {
		case "orch-go-aaa11":
			aaa = &items[i]
		case "orch-go-bbb22":
			bbb = &items[i]
		}
	}

	if aaa == nil || bbb == nil {
		t.Fatalf("Missing expected briefs, got %v", items)
	}

	// aaa11 should have thread linkage and tension
	if aaa.ThreadSlug != "test-thread" {
		t.Errorf("Expected thread_slug 'test-thread', got %q", aaa.ThreadSlug)
	}
	if aaa.ThreadTitle != "Test Thread" {
		t.Errorf("Expected thread_title 'Test Thread', got %q", aaa.ThreadTitle)
	}
	if !aaa.HasTension {
		t.Error("Expected has_tension=true for brief with ## Tension section")
	}

	// bbb22 should have no thread linkage and no tension
	if bbb.ThreadSlug != "" {
		t.Errorf("Expected empty thread_slug, got %q", bbb.ThreadSlug)
	}
	if bbb.ThreadTitle != "" {
		t.Errorf("Expected empty thread_title, got %q", bbb.ThreadTitle)
	}
	if bbb.HasTension {
		t.Error("Expected has_tension=false for brief without ## Tension section")
	}
}

func TestBriefHasTension(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "with tension content",
			content: "## Frame\n\nStuff.\n\n## Tension\n\nOpen question here.\n",
			want:    true,
		},
		{
			name:    "empty tension section",
			content: "## Frame\n\nStuff.\n\n## Tension\n\n## Next\n\nMore.\n",
			want:    false,
		},
		{
			name:    "no tension section",
			content: "## Frame\n\nStuff.\n\n## Resolution\n\nDone.\n",
			want:    false,
		},
		{
			name:    "tension with only whitespace",
			content: "## Frame\n\nStuff.\n\n## Tension\n\n   \n\n",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := briefHasTension(tt.content)
			if got != tt.want {
				t.Errorf("briefHasTension() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBriefReadStatePersistence(t *testing.T) {
	// Save and restore global state
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

	// Override HOME to isolate the state file
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	orchDir := filepath.Join(tmpHome, ".orch")
	os.MkdirAll(orchDir, 0755)

	// Set up a project with a brief
	oldSourceDir := sourceDir
	defer func() { sourceDir = oldSourceDir }()
	sourceDir = t.TempDir()

	briefsDir := filepath.Join(sourceDir, ".kb", "briefs")
	os.MkdirAll(briefsDir, 0755)
	os.WriteFile(filepath.Join(briefsDir, "orch-go-persist1.md"), []byte("brief"), 0644)

	// Mark as read via POST (triggers saveBriefReadState)
	req := httptest.NewRequest(http.MethodPost, "/api/briefs/orch-go-persist1", nil)
	w := httptest.NewRecorder()
	handleBrief(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("Mark as read returned %d", w.Result().StatusCode)
	}

	// Verify state file was written
	stateFile := filepath.Join(orchDir, "briefs-read-state.json")
	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("State file not written: %v", err)
	}
	var persisted map[string]bool
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatalf("State file invalid JSON: %v", err)
	}
	key := briefReadKey(sourceDir, "orch-go-persist1")
	if !persisted[key] {
		t.Error("Expected persisted state to contain the marked-read brief")
	}

	// Simulate restart: clear in-memory state, then load from disk
	briefReadStateMu.Lock()
	briefReadState = make(map[string]bool)
	briefReadStateMu.Unlock()

	loadBriefReadState()

	// Verify the state survived the "restart"
	briefReadStateMu.RLock()
	restored := briefReadState[key]
	briefReadStateMu.RUnlock()
	if !restored {
		t.Error("Expected brief read state to survive simulated restart")
	}
}

func TestBriefReadStateCorruptFile(t *testing.T) {
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

	// Override HOME with corrupt state file
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	orchDir := filepath.Join(tmpHome, ".orch")
	os.MkdirAll(orchDir, 0755)
	os.WriteFile(filepath.Join(orchDir, "briefs-read-state.json"), []byte("not json"), 0644)

	// Should not panic, just start with empty state
	loadBriefReadState()

	briefReadStateMu.RLock()
	count := len(briefReadState)
	briefReadStateMu.RUnlock()
	if count != 0 {
		t.Errorf("Expected empty state after corrupt file, got %d entries", count)
	}
}

func TestHandleBriefReadStateIsolation(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // isolate state file writes
	// Two projects sharing the same beads ID should have independent read state
	projectA := t.TempDir()
	projectB := t.TempDir()

	briefsDirA := filepath.Join(projectA, ".kb", "briefs")
	briefsDirB := filepath.Join(projectB, ".kb", "briefs")
	os.MkdirAll(briefsDirA, 0755)
	os.MkdirAll(briefsDirB, 0755)

	// Same beads ID in both projects
	os.WriteFile(filepath.Join(briefsDirA, "shared-id.md"), []byte("brief A"), 0644)
	os.WriteFile(filepath.Join(briefsDirB, "shared-id.md"), []byte("brief B"), 0644)

	// Clean up read state
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

	// Mark as read in project A only
	reqMark := httptest.NewRequest(http.MethodPost, "/api/briefs/shared-id?project_dir="+projectA, nil)
	wMark := httptest.NewRecorder()
	handleBrief(wMark, reqMark)

	if wMark.Result().StatusCode != http.StatusOK {
		t.Fatalf("Mark as read returned %d", wMark.Result().StatusCode)
	}

	// Project A should show read
	reqA := httptest.NewRequest(http.MethodGet, "/api/briefs/shared-id?project_dir="+projectA, nil)
	wA := httptest.NewRecorder()
	handleBrief(wA, reqA)

	var respA BriefAPIResponse
	json.NewDecoder(wA.Result().Body).Decode(&respA)
	if !respA.MarkedRead {
		t.Error("Expected shared-id to be marked_read in project A")
	}

	// Project B should still be unread
	reqB := httptest.NewRequest(http.MethodGet, "/api/briefs/shared-id?project_dir="+projectB, nil)
	wB := httptest.NewRecorder()
	handleBrief(wB, reqB)

	var respB BriefAPIResponse
	json.NewDecoder(wB.Result().Body).Decode(&respB)
	if respB.MarkedRead {
		t.Error("Expected shared-id to be unread in project B (isolation failure)")
	}
}
