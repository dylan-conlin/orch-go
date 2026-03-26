package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/thread"
)

func TestHandleThreadsList(t *testing.T) {
	oldSourceDir := sourceDir
	defer func() { sourceDir = oldSourceDir }()
	sourceDir = t.TempDir()

	threadsDir := filepath.Join(sourceDir, ".kb", "threads")
	if err := os.MkdirAll(threadsDir, 0755); err != nil {
		t.Fatalf("Failed to create threads dir: %v", err)
	}

	// Create a thread file
	content := `---
title: "Test thread"
status: forming
created: 2026-03-26
updated: 2026-03-26
resolved_to: ""
---

# Test thread

## 2026-03-26

First entry here.
`
	if err := os.WriteFile(filepath.Join(threadsDir, "2026-03-26-test-thread.md"), []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write thread: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/threads", nil)
	w := httptest.NewRecorder()

	handleThreadsList(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var threads []thread.ThreadSummary
	if err := json.NewDecoder(resp.Body).Decode(&threads); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(threads) != 1 {
		t.Fatalf("Expected 1 thread, got %d", len(threads))
	}
	if threads[0].Title != "Test thread" {
		t.Errorf("Expected title %q, got %q", "Test thread", threads[0].Title)
	}
	if threads[0].Name != "test-thread" {
		t.Errorf("Expected name %q, got %q", "test-thread", threads[0].Name)
	}
	if threads[0].Status != "forming" {
		t.Errorf("Expected status %q, got %q", "forming", threads[0].Status)
	}
}

func TestHandleThreadsListEmpty(t *testing.T) {
	oldSourceDir := sourceDir
	defer func() { sourceDir = oldSourceDir }()
	sourceDir = t.TempDir()

	req := httptest.NewRequest(http.MethodGet, "/api/threads", nil)
	w := httptest.NewRecorder()

	handleThreadsList(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var threads []thread.ThreadSummary
	if err := json.NewDecoder(resp.Body).Decode(&threads); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(threads) != 0 {
		t.Errorf("Expected 0 threads, got %d", len(threads))
	}
}

func TestHandleThreadShow(t *testing.T) {
	oldSourceDir := sourceDir
	defer func() { sourceDir = oldSourceDir }()
	sourceDir = t.TempDir()

	threadsDir := filepath.Join(sourceDir, ".kb", "threads")
	if err := os.MkdirAll(threadsDir, 0755); err != nil {
		t.Fatalf("Failed to create threads dir: %v", err)
	}

	content := `---
title: "Enforcement comprehension"
status: active
created: 2026-03-20
updated: 2026-03-26
resolved_to: ""
---

# Enforcement comprehension

## 2026-03-20

Initial thinking about enforcement.

## 2026-03-26

Further development of the idea.
`
	if err := os.WriteFile(filepath.Join(threadsDir, "2026-03-20-enforcement-comprehension.md"), []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write thread: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/threads/enforcement-comprehension", nil)
	w := httptest.NewRecorder()

	handleThreadShow(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result ThreadAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Title != "Enforcement comprehension" {
		t.Errorf("Expected title %q, got %q", "Enforcement comprehension", result.Title)
	}
	if result.Status != "active" {
		t.Errorf("Expected status %q, got %q", "active", result.Status)
	}
	if result.Slug != "enforcement-comprehension" {
		t.Errorf("Expected slug %q, got %q", "enforcement-comprehension", result.Slug)
	}
	if len(result.Entries) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(result.Entries))
	}
	if result.Content == "" {
		t.Error("Expected non-empty content")
	}
}

func TestHandleThreadShowNotFound(t *testing.T) {
	oldSourceDir := sourceDir
	defer func() { sourceDir = oldSourceDir }()
	sourceDir = t.TempDir()

	threadsDir := filepath.Join(sourceDir, ".kb", "threads")
	if err := os.MkdirAll(threadsDir, 0755); err != nil {
		t.Fatalf("Failed to create threads dir: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/threads/nonexistent", nil)
	w := httptest.NewRecorder()

	handleThreadShow(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestHandleThreadShowMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/threads/some-slug", nil)
	w := httptest.NewRecorder()

	handleThreadShow(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestHandleThreadsListMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/threads", nil)
	w := httptest.NewRecorder()

	handleThreadsList(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestHandleThreadsListWithStatusFilter(t *testing.T) {
	oldSourceDir := sourceDir
	defer func() { sourceDir = oldSourceDir }()
	sourceDir = t.TempDir()

	threadsDir := filepath.Join(sourceDir, ".kb", "threads")
	if err := os.MkdirAll(threadsDir, 0755); err != nil {
		t.Fatalf("Failed to create threads dir: %v", err)
	}

	// Create forming thread
	forming := `---
title: "Forming thread"
status: forming
created: 2026-03-26
updated: 2026-03-26
resolved_to: ""
---

# Forming thread

## 2026-03-26

Entry.
`
	if err := os.WriteFile(filepath.Join(threadsDir, "2026-03-26-forming-thread.md"), []byte(forming), 0644); err != nil {
		t.Fatalf("Failed to write thread: %v", err)
	}

	// Create resolved thread
	resolved := `---
title: "Resolved thread"
status: resolved
created: 2026-03-20
updated: 2026-03-25
resolved_to: ".kb/models/test.md"
---

# Resolved thread

## 2026-03-20

Entry.
`
	if err := os.WriteFile(filepath.Join(threadsDir, "2026-03-20-resolved-thread.md"), []byte(resolved), 0644); err != nil {
		t.Fatalf("Failed to write thread: %v", err)
	}

	// Filter for forming only
	req := httptest.NewRequest(http.MethodGet, "/api/threads?status=forming", nil)
	w := httptest.NewRecorder()

	handleThreadsList(w, req)

	var threads []thread.ThreadSummary
	if err := json.NewDecoder(w.Result().Body).Decode(&threads); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(threads) != 1 {
		t.Fatalf("Expected 1 thread with status=forming, got %d", len(threads))
	}
	if threads[0].Name != "forming-thread" {
		t.Errorf("Expected forming-thread, got %q", threads[0].Name)
	}
}
