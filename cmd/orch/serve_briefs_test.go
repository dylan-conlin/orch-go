package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

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

	// Test hasBriefFile helper
	if !hasBriefFile("test-id-1") {
		t.Error("Expected hasBriefFile to return true for existing brief")
	}
	if hasBriefFile("nonexistent-id") {
		t.Error("Expected hasBriefFile to return false for nonexistent brief")
	}
}
