package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/dylan-conlin/orch-go/pkg/attention"
	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// handleLikelyDone returns LIKELY_DONE attention signals for the dashboard.
// These are issues with recent commits but no active workspace, suggesting
// they may be complete but not yet closed.
func handleLikelyDone(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get project directory from query parameter (default to sourceDir)
	projectDir := r.URL.Query().Get("project")
	if projectDir == "" {
		projectDir = sourceDir
	}

	// Get beads client (RPC or CLI fallback)
	// Note: Must check beadsClient before assigning to interface to avoid
	// Go's nil interface gotcha (interface with nil data is not == nil)
	beadsClientMu.RLock()
	rpcClient := beadsClient
	beadsClientMu.RUnlock()

	var client beads.BeadsClient
	if rpcClient != nil {
		client = rpcClient
	} else {
		client = beads.NewCLIClient(beads.WithWorkDir(projectDir))
	}

	// Check if cache is initialized
	if globalLikelyDoneCache == nil {
		// Fallback: fetch without cache if not initialized
		data, err := attention.CollectLikelyDoneSignals(projectDir, client)
		if err != nil {
			resp := &attention.LikelyDoneResponse{
				Error: fmt.Sprintf("Failed to collect likely done signals: %v", err),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
		return
	}

	// Get cached or fresh data
	data, err := globalLikelyDoneCache.Get(projectDir, client)
	if err != nil {
		resp := &attention.LikelyDoneResponse{
			Error: fmt.Sprintf("Failed to collect likely done signals: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode likely done signals: %v", err), http.StatusInternalServerError)
		return
	}
}
