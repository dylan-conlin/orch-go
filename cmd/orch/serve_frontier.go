package main

import (
	"encoding/json"
	"net/http"

	"github.com/dylan-conlin/orch-go/pkg/frontier"
)

// FrontierAPIResponse is the JSON structure returned by /api/frontier.
type FrontierAPIResponse struct {
	Warnings     []string              `json:"warnings,omitempty"`
	Ready        []FrontierIssue       `json:"ready"`
	ReadyTotal   int                   `json:"ready_total"`
	Blocked      []BlockedOutput       `json:"blocked"`
	BlockedTotal int                   `json:"blocked_total"`
	Active       []ActiveOutput        `json:"active"`
	ActiveTotal  int                   `json:"active_total"`
	Stuck        []ActiveOutput        `json:"stuck"`
	StuckTotal   int                   `json:"stuck_total"`
	Error        string                `json:"error,omitempty"`
}

// handleFrontier returns the decidability frontier state.
// This endpoint provides the same data as `orch frontier --json` but via HTTP.
func handleFrontier(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Calculate frontier state from beads
	state, err := frontier.CalculateFrontier()
	if err != nil {
		resp := FrontierAPIResponse{
			Ready:   []FrontierIssue{},
			Blocked: []BlockedOutput{},
			Active:  []ActiveOutput{},
			Stuck:   []ActiveOutput{},
			Error:   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Get active agents from registry and split into active vs stuck
	activeAgents, stuckAgents := getActiveAndStuckAgents()

	// Build response using same types as frontier.go
	resp := FrontierAPIResponse{
		Ready:        make([]FrontierIssue, 0, len(state.Ready)),
		ReadyTotal:   len(state.Ready),
		Blocked:      make([]BlockedOutput, 0, len(state.Blocked)),
		BlockedTotal: len(state.Blocked),
		Active:       activeAgents,
		ActiveTotal:  len(activeAgents),
		Stuck:        stuckAgents,
		StuckTotal:   len(stuckAgents),
	}

	// Add health warnings
	if len(stuckAgents) > 0 {
		resp.Warnings = append(resp.Warnings, "Stuck agents detected (> 2h) - run 'orch clean --stale' to clean up")
	}

	for _, issue := range state.Ready {
		resp.Ready = append(resp.Ready, FrontierIssue{
			ID:        issue.ID,
			Title:     issue.Title,
			IssueType: issue.IssueType,
			Priority:  issue.Priority,
		})
	}

	for _, bi := range state.Blocked {
		resp.Blocked = append(resp.Blocked, BlockedOutput{
			ID:            bi.Issue.ID,
			Title:         bi.Issue.Title,
			IssueType:     bi.Issue.IssueType,
			Priority:      bi.Issue.Priority,
			WouldUnblock:  bi.WouldUnblock,
			TotalLeverage: bi.TotalLeverage,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "Failed to encode frontier", http.StatusInternalServerError)
		return
	}
}
