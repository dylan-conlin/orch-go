package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

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

// AttentionItemResponse represents an attention item in the API response.
type AttentionItemResponse struct {
	ID          string         `json:"id"`
	Source      string         `json:"source"`
	Concern     string         `json:"concern"`
	Signal      string         `json:"signal"`
	Subject     string         `json:"subject"`
	Summary     string         `json:"summary"`
	Priority    int            `json:"priority"`
	Role        string         `json:"role"`
	ActionHint  string         `json:"action_hint,omitempty"`
	CollectedAt string         `json:"collected_at"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// AttentionAPIResponse is the response structure for /api/attention endpoint.
type AttentionAPIResponse struct {
	Items       []AttentionItemResponse `json:"items"`
	Total       int                     `json:"total"`
	Sources     []string                `json:"sources"`
	Role        string                  `json:"role"`
	CollectedAt string                  `json:"collected_at"`
}

// handleAttention returns unified attention signals from multiple collectors.
// Query parameters:
//   - role: Role for priority scoring (human, orchestrator, daemon) - default: human
func handleAttention(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse role parameter
	role := r.URL.Query().Get("role")
	if role == "" {
		role = "human"
	}

	// Validate role parameter
	validRoles := map[string]bool{
		"human":        true,
		"orchestrator": true,
		"daemon":       true,
	}
	if !validRoles[role] {
		role = "human" // Default to human for invalid roles
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

	// Initialize collectors
	collectors := []attention.Collector{}
	sources := []string{}

	// BeadsCollector - ready issues
	beadsCollector := attention.NewBeadsCollector(client)
	collectors = append(collectors, beadsCollector)
	sources = append(sources, "beads")

	// GitCollector - likely-done signals
	if projectDir != "" {
		gitCollector := attention.NewGitCollector(projectDir, client)
		collectors = append(collectors, gitCollector)
		sources = append(sources, "git")
	}

	// Collect from all sources
	allItems := []attention.AttentionItem{}
	for _, collector := range collectors {
		items, err := collector.Collect(role)
		if err != nil {
			// Log error but continue with other collectors
			// This ensures partial results if one collector fails
			continue
		}
		allItems = append(allItems, items...)
	}

	// Sort by priority (lower = higher priority)
	sort.Slice(allItems, func(i, j int) bool {
		return allItems[i].Priority < allItems[j].Priority
	})

	// Transform to response format
	responseItems := make([]AttentionItemResponse, 0, len(allItems))
	for _, item := range allItems {
		responseItems = append(responseItems, AttentionItemResponse{
			ID:          item.ID,
			Source:      item.Source,
			Concern:     item.Concern.String(),
			Signal:      item.Signal,
			Subject:     item.Subject,
			Summary:     item.Summary,
			Priority:    item.Priority,
			Role:        item.Role,
			ActionHint:  item.ActionHint,
			CollectedAt: item.CollectedAt.Format(time.RFC3339),
			Metadata:    item.Metadata,
		})
	}

	// Build response
	response := AttentionAPIResponse{
		Items:       responseItems,
		Total:       len(responseItems),
		Sources:     sources,
		Role:        role,
		CollectedAt: time.Now().Format(time.RFC3339),
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
