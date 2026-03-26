package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// DaemonResumeResponse is the response for POST /api/daemon/resume.
type DaemonResumeResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// handleDaemonResume writes a resume signal to unpause the daemon.
// This is the API equivalent of `orch daemon resume`.
func handleDaemonResume(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := daemon.WriteResumeSignal(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DaemonResumeResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to write resume signal: %v", err),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(DaemonResumeResponse{
		Success: true,
		Message: "Resume signal sent - daemon will resume on next poll cycle",
	})
}

// CloseIssueRequest is the request body for POST /api/issues/close.
type CloseIssueRequest struct {
	BeadsID string `json:"beads_id"`
	Reason  string `json:"reason,omitempty"`
}

// CloseIssueResponse is the response for POST /api/issues/close.
type CloseIssueResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// handleCloseIssue closes a beads issue and writes a verification signal
// to notify the daemon that human verification has occurred.
func handleCloseIssue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CloseIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(CloseIssueResponse{
			Success: false,
			Error:   fmt.Sprintf("Invalid request body: %v", err),
		})
		return
	}

	if req.BeadsID == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(CloseIssueResponse{
			Success: false,
			Error:   "beads_id is required",
		})
		return
	}

	reason := req.Reason
	if reason == "" {
		reason = "Acknowledged via dashboard review"
	}

	if err := verify.CloseIssue(req.BeadsID, reason, sourceDir); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(CloseIssueResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to close issue %s: %v", req.BeadsID, err),
		})
		return
	}

	// NOTE: Do NOT write verification signal here. Dashboard API is used by
	// both humans and automated orchestrator sessions. Writing the signal from
	// this path defeats the comprehension throttle — the daemon's
	// completionsSinceVerification counter gets reset by non-human callers,
	// preventing the pause threshold from ever being reached.
	// Human verification signal is only written from interactive `orch complete`.

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CloseIssueResponse{
		Success: true,
		Message: fmt.Sprintf("Issue %s closed", req.BeadsID),
	})
}

// CloseIssueBatchRequest is the request body for POST /api/issues/close-batch.
type CloseIssueBatchRequest struct {
	BeadsIDs []string `json:"beads_ids"`
	Reason   string   `json:"reason,omitempty"`
}

// CloseIssueBatchResult represents the result of closing a single issue in a batch.
type CloseIssueBatchResult struct {
	BeadsID string `json:"beads_id"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// CloseIssueBatchResponse is the response for POST /api/issues/close-batch.
type CloseIssueBatchResponse struct {
	Results     []CloseIssueBatchResult `json:"results"`
	TotalClosed int                     `json:"total_closed"`
	TotalFailed int                     `json:"total_failed"`
}

// handleCloseIssueBatch closes multiple beads issues and writes a single
// verification signal to reset the daemon's completions_since_verification counter.
func handleCloseIssueBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CloseIssueBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(CloseIssueBatchResponse{
			Results: []CloseIssueBatchResult{},
		})
		return
	}

	if len(req.BeadsIDs) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(CloseIssueBatchResponse{
			Results: []CloseIssueBatchResult{},
		})
		return
	}

	reason := req.Reason
	if reason == "" {
		reason = "Acknowledged via dashboard batch review"
	}

	var results []CloseIssueBatchResult
	totalClosed := 0
	totalFailed := 0

	for _, beadsID := range req.BeadsIDs {
		if beadsID == "" {
			continue
		}

		if err := verify.CloseIssue(beadsID, reason, sourceDir); err != nil {
			results = append(results, CloseIssueBatchResult{
				BeadsID: beadsID,
				Success: false,
				Error:   err.Error(),
			})
			totalFailed++
		} else {
			results = append(results, CloseIssueBatchResult{
				BeadsID: beadsID,
				Success: true,
			})
			totalClosed++
		}
	}

	// NOTE: Do NOT write verification signal here. See single-close handler
	// for rationale — dashboard batch close is used by automated orchestrator
	// sessions, not exclusively by Dylan.

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CloseIssueBatchResponse{
		Results:     results,
		TotalClosed: totalClosed,
		TotalFailed: totalFailed,
	})
}
