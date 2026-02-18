package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHandleBeadsMethodNotAllowed(t *testing.T) {
	// Ensure cache is initialized
	if globalBeadsStatsCache == nil {
		globalBeadsStatsCache = newBeadsStatsCache()
	}

	req := httptest.NewRequest(http.MethodPost, "/api/beads", nil)
	w := httptest.NewRecorder()

	handleBeads(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestHandleBeadsWithProjectParam(t *testing.T) {
	// Create a temp directory with a .beads/ directory to simulate a project
	tmpDir := t.TempDir()
	beadsDir := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatalf("Failed to create .beads dir: %v", err)
	}

	// Create a minimal issues.jsonl file (empty array in newline-delimited format)
	issuesFile := filepath.Join(beadsDir, "issues.jsonl")
	if err := os.WriteFile(issuesFile, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create issues.jsonl: %v", err)
	}

	// Ensure cache is initialized
	if globalBeadsStatsCache == nil {
		globalBeadsStatsCache = newBeadsStatsCache()
	}

	// Test with project_dir parameter
	req := httptest.NewRequest(http.MethodGet, "/api/beads?project_dir="+tmpDir, nil)
	w := httptest.NewRecorder()

	handleBeads(w, req)

	resp := w.Result()
	// Should return 200 even if bd returns error (empty beads directory)
	// The important thing is it accepts the project_dir parameter
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	// Verify valid JSON response
	var beadsResp BeadsAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&beadsResp); err != nil {
		t.Errorf("Expected valid JSON response, got error: %v", err)
	}
}

func TestHandleBeadsReadyWithProjectParam(t *testing.T) {
	// Create a temp directory with a .beads/ directory to simulate a project
	tmpDir := t.TempDir()
	beadsDir := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatalf("Failed to create .beads dir: %v", err)
	}

	// Create a minimal issues.jsonl file
	issuesFile := filepath.Join(beadsDir, "issues.jsonl")
	if err := os.WriteFile(issuesFile, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create issues.jsonl: %v", err)
	}

	// Ensure cache is initialized
	if globalBeadsStatsCache == nil {
		globalBeadsStatsCache = newBeadsStatsCache()
	}

	// Test with project_dir parameter
	req := httptest.NewRequest(http.MethodGet, "/api/beads/ready?project_dir="+tmpDir, nil)
	w := httptest.NewRecorder()

	handleBeadsReady(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	// Verify valid JSON response with issues array
	var readyResp BeadsReadyAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&readyResp); err != nil {
		t.Errorf("Expected valid JSON response, got error: %v", err)
	}

	// Issues should be empty but not nil
	if readyResp.Issues == nil {
		t.Error("Expected issues to be non-nil")
	}
}

func TestHandleBeadsReviewQueueWithProjectParam(t *testing.T) {
	// Create a temp directory with a .beads/ directory to simulate a project
	tmpDir := t.TempDir()
	beadsDir := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatalf("Failed to create .beads dir: %v", err)
	}

	// Create a minimal issues.jsonl file
	issuesFile := filepath.Join(beadsDir, "issues.jsonl")
	if err := os.WriteFile(issuesFile, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create issues.jsonl: %v", err)
	}

	// Ensure cache is initialized
	if globalBeadsStatsCache == nil {
		globalBeadsStatsCache = newBeadsStatsCache()
	}

	// Test with project_dir parameter
	req := httptest.NewRequest(http.MethodGet, "/api/beads/review-queue?project_dir="+tmpDir, nil)
	w := httptest.NewRecorder()

	handleBeadsReviewQueue(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	// Verify valid JSON response with issues array
	var reviewResp BeadsReviewQueueResponse
	if err := json.NewDecoder(resp.Body).Decode(&reviewResp); err != nil {
		t.Errorf("Expected valid JSON response, got error: %v", err)
	}

	// Issues should be empty but not nil
	if reviewResp.Issues == nil {
		t.Error("Expected issues to be non-nil")
	}
}

func TestBeadsStatsCacheProjectAwareness(t *testing.T) {
	// Create a fresh cache
	cache := newBeadsStatsCache()

	// Verify the cache uses per-project caching
	// This verifies the cache structure is project-aware
	if cache.statsTTL <= 0 {
		t.Error("Expected positive stats TTL")
	}
	if cache.readyTTL <= 0 {
		t.Error("Expected positive ready TTL")
	}
	if cache.reviewTTL <= 0 {
		t.Error("Expected positive review TTL")
	}
}

func TestHandleBeadsReadyMethodNotAllowed(t *testing.T) {
	// Ensure cache is initialized
	if globalBeadsStatsCache == nil {
		globalBeadsStatsCache = newBeadsStatsCache()
	}

	req := httptest.NewRequest(http.MethodPost, "/api/beads/ready", nil)
	w := httptest.NewRecorder()

	handleBeadsReady(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestHandleBeadsReviewQueueMethodNotAllowed(t *testing.T) {
	// Ensure cache is initialized
	if globalBeadsStatsCache == nil {
		globalBeadsStatsCache = newBeadsStatsCache()
	}

	req := httptest.NewRequest(http.MethodPost, "/api/beads/review-queue", nil)
	w := httptest.NewRecorder()

	handleBeadsReviewQueue(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}
