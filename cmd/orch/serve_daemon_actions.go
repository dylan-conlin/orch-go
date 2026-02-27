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

	if err := verify.CloseIssue(req.BeadsID, reason); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(CloseIssueResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to close issue %s: %v", req.BeadsID, err),
		})
		return
	}

	// Write verification signal so daemon knows human reviewed something
	// Best effort - don't fail the close if signal write fails
	_ = daemon.WriteVerificationSignal()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CloseIssueResponse{
		Success: true,
		Message: fmt.Sprintf("Issue %s closed", req.BeadsID),
	})
}
