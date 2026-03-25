package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// BriefListItem is the JSON structure for each item in the GET /api/briefs list.
type BriefListItem struct {
	BeadsID    string `json:"beads_id"`
	MarkedRead bool   `json:"marked_read"`
}

// BriefAPIResponse is the JSON structure returned by GET /api/briefs/{beads-id}.
type BriefAPIResponse struct {
	BeadsID    string `json:"beads_id"`
	Content    string `json:"content"`
	MarkedRead bool   `json:"marked_read"`
}

// BriefMarkReadResponse is the JSON structure returned by POST /api/briefs/{beads-id}.
type BriefMarkReadResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// briefReadState tracks which briefs have been marked as read.
// This is UI-only state — does NOT affect comprehension labels.
// orch complete remains the sole comprehension gate.
// Keys are "project_dir:beadsID" to avoid collisions across projects.
var (
	briefReadState   = make(map[string]bool)
	briefReadStateMu sync.RWMutex
)

// briefReadKey returns the map key for briefReadState, scoped by project.
func briefReadKey(projectDir, beadsID string) string {
	return projectDir + ":" + beadsID
}

// briefReadStatePath returns the path to the persistent read state file.
func briefReadStatePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".orch", "briefs-read-state.json")
}

// loadBriefReadState loads persisted read state from disk.
// Called once at server startup. Missing or corrupt file is not an error — starts fresh.
func loadBriefReadState() {
	path := briefReadStatePath()
	if path == "" {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return // file doesn't exist yet — normal on first run
	}
	var loaded map[string]bool
	if err := json.Unmarshal(data, &loaded); err != nil {
		log.Printf("Warning: could not parse %s: %v (starting with empty read state)", path, err)
		return
	}
	briefReadStateMu.Lock()
	for k, v := range loaded {
		briefReadState[k] = v
	}
	briefReadStateMu.Unlock()
}

// saveBriefReadState persists the current read state to disk.
// Called after each mark-as-read. Writes atomically via temp file + rename.
func saveBriefReadState() {
	path := briefReadStatePath()
	if path == "" {
		return
	}
	briefReadStateMu.RLock()
	data, err := json.Marshal(briefReadState)
	briefReadStateMu.RUnlock()
	if err != nil {
		log.Printf("Warning: could not marshal brief read state: %v", err)
		return
	}
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		log.Printf("Warning: could not create directory for %s: %v", path, err)
		return
	}
	// Atomic write: temp file + rename
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		log.Printf("Warning: could not write %s: %v", tmp, err)
		return
	}
	if err := os.Rename(tmp, path); err != nil {
		log.Printf("Warning: could not rename %s to %s: %v", tmp, path, err)
	}
}

// validBeadsID matches beads IDs like "orch-go-abc12" or "project-name-xyz99"
var validBeadsID = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// handleBrief serves brief content and handles mark-as-read.
// GET /api/briefs/{beads-id}?project_dir=/path - returns brief content from .kb/briefs/{beads-id}.md
// POST /api/briefs/{beads-id}?project_dir=/path - marks brief as read (UI state only)
func handleBrief(w http.ResponseWriter, r *http.Request) {
	// Extract beads ID from URL path
	beadsID := strings.TrimPrefix(r.URL.Path, "/api/briefs/")
	beadsID = strings.TrimSuffix(beadsID, "/")

	if beadsID == "" || !validBeadsID.MatchString(beadsID) {
		http.Error(w, "Invalid beads ID", http.StatusBadRequest)
		return
	}

	// Security: reject path traversal
	if strings.Contains(beadsID, "..") || strings.Contains(beadsID, "/") {
		http.Error(w, "Invalid beads ID", http.StatusBadRequest)
		return
	}

	// Get project directory from query parameter (default to sourceDir)
	projectDir := r.URL.Query().Get("project_dir")
	if projectDir == "" {
		projectDir = sourceDir
	}

	switch r.Method {
	case http.MethodGet:
		handleBriefGet(w, beadsID, projectDir)
	case http.MethodPost:
		handleBriefMarkRead(w, beadsID, projectDir)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleBriefGet(w http.ResponseWriter, beadsID, projectDir string) {
	briefPath := filepath.Join(projectDir, ".kb", "briefs", beadsID+".md")
	content, err := os.ReadFile(briefPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "Brief not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to read brief: %v", err), http.StatusInternalServerError)
		return
	}

	key := briefReadKey(projectDir, beadsID)
	briefReadStateMu.RLock()
	markedRead := briefReadState[key]
	briefReadStateMu.RUnlock()

	resp := BriefAPIResponse{
		BeadsID:    beadsID,
		Content:    string(content),
		MarkedRead: markedRead,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleBriefMarkRead(w http.ResponseWriter, beadsID, projectDir string) {
	// Verify brief file exists before marking as read
	briefPath := filepath.Join(projectDir, ".kb", "briefs", beadsID+".md")
	if _, err := os.Stat(briefPath); os.IsNotExist(err) {
		http.Error(w, "Brief not found", http.StatusNotFound)
		return
	}

	key := briefReadKey(projectDir, beadsID)
	briefReadStateMu.Lock()
	briefReadState[key] = true
	briefReadStateMu.Unlock()

	saveBriefReadState()

	resp := BriefMarkReadResponse{
		Success: true,
		Message: fmt.Sprintf("Brief %s marked as read", beadsID),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleBriefsList serves the list of all briefs, sorted newest-first by mod time.
// GET /api/briefs?project_dir=/path - returns [{beads_id, marked_read}]
func handleBriefsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get project directory from query parameter (default to sourceDir)
	projectDir := r.URL.Query().Get("project_dir")
	if projectDir == "" {
		projectDir = sourceDir
	}

	briefsDir := filepath.Join(projectDir, ".kb", "briefs")
	entries, err := os.ReadDir(briefsDir)
	if err != nil {
		if os.IsNotExist(err) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]BriefListItem{})
			return
		}
		http.Error(w, fmt.Sprintf("Failed to read briefs directory: %v", err), http.StatusInternalServerError)
		return
	}

	type entryWithTime struct {
		beadsID string
		modTime int64
	}

	var items []entryWithTime
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		beadsID := strings.TrimSuffix(e.Name(), ".md")
		info, err := e.Info()
		if err != nil {
			continue
		}
		items = append(items, entryWithTime{beadsID: beadsID, modTime: info.ModTime().UnixNano()})
	}

	// Sort newest-first
	sort.Slice(items, func(i, j int) bool {
		return items[i].modTime > items[j].modTime
	})

	briefReadStateMu.RLock()
	result := make([]BriefListItem, len(items))
	for i, item := range items {
		key := briefReadKey(projectDir, item.beadsID)
		result[i] = BriefListItem{
			BeadsID:    item.beadsID,
			MarkedRead: briefReadState[key],
		}
	}
	briefReadStateMu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// hasBriefFile checks if a brief file exists for the given beads ID in the given project.
// If projectDir is empty, falls back to sourceDir.
func hasBriefFile(beadsID, projectDir string) bool {
	if projectDir == "" {
		projectDir = sourceDir
	}
	briefPath := filepath.Join(projectDir, ".kb", "briefs", beadsID+".md")
	_, err := os.Stat(briefPath)
	return err == nil
}
