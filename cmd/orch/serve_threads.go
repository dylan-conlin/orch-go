package main

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/thread"
)

// ThreadAPIResponse is the JSON structure for GET /api/threads/{slug}.
type ThreadAPIResponse struct {
	Title      string         `json:"title"`
	Status     string         `json:"status"`
	Slug       string         `json:"slug"`
	Created    string         `json:"created"`
	Updated    string         `json:"updated"`
	ResolvedTo string         `json:"resolved_to,omitempty"`
	Entries    []thread.Entry `json:"entries"`
	Content    string         `json:"content"`
	Filename   string         `json:"filename"`
}

// handleThreadsList serves GET /api/threads.
// Returns all threads as ThreadSummary list, sorted by updated date descending.
// Optional query params:
//
//	?status=forming   - filter by status (forming, active, converged, subsumed, resolved)
//	?project_dir=/path - override project directory
func handleThreadsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	projectDir := r.URL.Query().Get("project_dir")
	if projectDir == "" {
		projectDir = sourceDir
	}

	dir := filepath.Join(projectDir, ".kb", "threads")
	threads, err := thread.List(dir)
	if err != nil {
		http.Error(w, "Failed to list threads: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Apply status filter if provided
	if statusFilter := r.URL.Query().Get("status"); statusFilter != "" {
		var filtered []thread.ThreadSummary
		for _, t := range threads {
			if t.Status == statusFilter {
				filtered = append(filtered, t)
			}
		}
		threads = filtered
	}

	// Return empty array instead of null
	if threads == nil {
		threads = []thread.ThreadSummary{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(threads)
}

// handleThreadShow serves GET /api/threads/{slug}.
// Returns full thread content including entries and raw markdown.
func handleThreadShow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	slug := strings.TrimPrefix(r.URL.Path, "/api/threads/")
	slug = strings.TrimSuffix(slug, "/")

	if slug == "" {
		http.Error(w, "Thread slug required", http.StatusBadRequest)
		return
	}

	projectDir := r.URL.Query().Get("project_dir")
	if projectDir == "" {
		projectDir = sourceDir
	}

	dir := filepath.Join(projectDir, ".kb", "threads")
	t, err := thread.Show(dir, slug)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Thread not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to read thread: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp := ThreadAPIResponse{
		Title:      t.Title,
		Status:     t.Status,
		Slug:       t.Slug,
		Created:    t.Created,
		Updated:    t.Updated,
		ResolvedTo: t.ResolvedTo,
		Entries:    t.Entries,
		Content:    t.Content,
		Filename:   t.Filename,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
