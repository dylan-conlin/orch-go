package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

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
// This is UI-only state — does NOT affect comprehension:pending labels.
// orch complete remains the sole comprehension gate.
var (
	briefReadState   = make(map[string]bool)
	briefReadStateMu sync.RWMutex
)

// validBeadsID matches beads IDs like "orch-go-abc12" or "project-name-xyz99"
var validBeadsID = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// handleBrief serves brief content and handles mark-as-read.
// GET /api/briefs/{beads-id} - returns brief content from .kb/briefs/{beads-id}.md
// POST /api/briefs/{beads-id} - marks brief as read (UI state only)
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

	switch r.Method {
	case http.MethodGet:
		handleBriefGet(w, beadsID)
	case http.MethodPost:
		handleBriefMarkRead(w, beadsID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleBriefGet(w http.ResponseWriter, beadsID string) {
	briefPath := filepath.Join(sourceDir, ".kb", "briefs", beadsID+".md")
	content, err := os.ReadFile(briefPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "Brief not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to read brief: %v", err), http.StatusInternalServerError)
		return
	}

	briefReadStateMu.RLock()
	markedRead := briefReadState[beadsID]
	briefReadStateMu.RUnlock()

	resp := BriefAPIResponse{
		BeadsID:    beadsID,
		Content:    string(content),
		MarkedRead: markedRead,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleBriefMarkRead(w http.ResponseWriter, beadsID string) {
	// Verify brief file exists before marking as read
	briefPath := filepath.Join(sourceDir, ".kb", "briefs", beadsID+".md")
	if _, err := os.Stat(briefPath); os.IsNotExist(err) {
		http.Error(w, "Brief not found", http.StatusNotFound)
		return
	}

	briefReadStateMu.Lock()
	briefReadState[beadsID] = true
	briefReadStateMu.Unlock()

	resp := BriefMarkReadResponse{
		Success: true,
		Message: fmt.Sprintf("Brief %s marked as read", beadsID),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// hasBriefFile checks if a brief file exists for the given beads ID.
func hasBriefFile(beadsID string) bool {
	briefPath := filepath.Join(sourceDir, ".kb", "briefs", beadsID+".md")
	_, err := os.Stat(briefPath)
	return err == nil
}
