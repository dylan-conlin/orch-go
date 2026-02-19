package main

import (
	"encoding/json"
	"net/http"
)

// handleCacheInvalidate clears all dashboard caches to force fresh data on next request.
// This is called by orch complete to ensure the dashboard shows updated agent status.
// Without this, the TTL cache holds stale "active" status after agents complete.
func handleCacheInvalidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Invalidate beads cache (open issues, all issues, comments)
	if globalBeadsCache != nil {
		globalBeadsCache.invalidate()
	}

	// Invalidate beads stats cache (stats, ready issues)
	// Empty string clears all project caches
	if globalBeadsStatsCache != nil {
		globalBeadsStatsCache.invalidate("")
	}

	// Invalidate workspace cache (workspace metadata)
	globalWorkspaceCacheInstance.invalidate()

	// Invalidate tracked agents cache (queryTrackedAgents results)
	globalTrackedAgentsCache.invalidate()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": "Cache invalidated",
	})
}
