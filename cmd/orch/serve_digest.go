package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/daemon"
)

// DigestAPIResponse is the JSON structure returned by GET /api/digest.
type DigestAPIResponse struct {
	Products    []daemon.DigestProduct `json:"products"`
	UnreadCount int                    `json:"unread_count"`
	Total       int                    `json:"total"`
	Error       string                 `json:"error,omitempty"`
}

// DigestUpdateRequest is the JSON body for PATCH /api/digest/:id.
type DigestUpdateRequest struct {
	State string `json:"state"`
}

// DigestArchiveResponse is the JSON structure returned by POST /api/digest/archive-read.
type DigestArchiveResponse struct {
	Archived int    `json:"archived"`
	Error    string `json:"error,omitempty"`
}

func digestDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".orch", "digest")
}

// handleDigest handles GET /api/digest with optional query params:
//
//	state=new|read|starred|archived
//	type=thread_progression|model_update|decision_brief
//	limit=N
func handleDigest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	dir := digestDir()
	if dir == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DigestAPIResponse{Error: "cannot determine home directory"})
		return
	}

	store := daemon.NewDigestStore(dir)

	opts := daemon.DigestListOpts{}
	if s := r.URL.Query().Get("state"); s != "" {
		opts.State = daemon.DigestProductState(s)
	}
	if t := r.URL.Query().Get("type"); t != "" {
		opts.Type = daemon.DigestProductType(t)
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			opts.Limit = n
		}
	}

	products, err := store.List(opts)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DigestAPIResponse{Error: fmt.Sprintf("list products: %v", err)})
		return
	}

	// Get unread count
	stats, _ := store.Stats()

	if products == nil {
		products = []daemon.DigestProduct{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(DigestAPIResponse{
		Products:    products,
		UnreadCount: stats.Unread,
		Total:       stats.Total,
	})
}

// handleDigestStats handles GET /api/digest/stats.
func handleDigestStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	dir := digestDir()
	if dir == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(daemon.DigestStatsResponse{})
		return
	}

	store := daemon.NewDigestStore(dir)
	stats, err := store.Stats()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// handleDigestUpdate handles PATCH /api/digest/{id} to update product state.
// URL pattern: /api/digest/update?id=PRODUCT_ID
func handleDigestUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract product ID from query param or URL path
	id := r.URL.Query().Get("id")
	if id == "" {
		// Try extracting from path: /api/digest/update/PRODUCT_ID
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) > 4 {
			id = parts[len(parts)-1]
		}
	}
	if id == "" {
		http.Error(w, "missing product id", http.StatusBadRequest)
		return
	}

	var req DigestUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate state
	state := daemon.DigestProductState(req.State)
	switch state {
	case daemon.DigestStateRead, daemon.DigestStateStarred, daemon.DigestStateArchived:
		// valid
	default:
		http.Error(w, "invalid state: must be read, starred, or archived", http.StatusBadRequest)
		return
	}

	dir := digestDir()
	if dir == "" {
		http.Error(w, "cannot determine home directory", http.StatusInternalServerError)
		return
	}

	store := daemon.NewDigestStore(dir)
	if err := store.UpdateState(id, state); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
}

// handleDigestArchiveRead handles POST /api/digest/archive-read?older_than=7d.
func handleDigestArchiveRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse older_than duration (default 7 days)
	olderThan := 7 * 24 * time.Hour
	if ot := r.URL.Query().Get("older_than"); ot != "" {
		if d, err := parseDuration(ot); err == nil {
			olderThan = d
		}
	}

	dir := digestDir()
	if dir == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DigestArchiveResponse{Error: "cannot determine home directory"})
		return
	}

	store := daemon.NewDigestStore(dir)
	archived, err := store.ArchiveRead(olderThan)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DigestArchiveResponse{Error: err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(DigestArchiveResponse{Archived: archived})
}

// parseDuration parses a duration string like "7d", "24h", "30m".
func parseDuration(s string) (time.Duration, error) {
	if strings.HasSuffix(s, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err != nil {
			return 0, err
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}
