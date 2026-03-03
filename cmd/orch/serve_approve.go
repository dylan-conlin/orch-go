package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// ApproveRequest represents the request body for POST /api/approve.
type ApproveRequest struct {
	// AgentID is the agent workspace ID or beads ID
	AgentID string `json:"agent_id"`

	// Description is an optional description of what was approved
	Description string `json:"description"`
}

// ApproveResponse represents the response for POST /api/approve.
type ApproveResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

// handleApprove handles POST /api/approve to approve an agent's work.
// Creates a beads comment with ✅ APPROVED format and updates workspace ReviewState.
func handleApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req ApproveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeApproveError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.AgentID == "" {
		writeApproveError(w, "agent_id is required", http.StatusBadRequest)
		return
	}

	// Find workspace by agent ID (could be workspace name or beads ID)
	workspacePath, beadsID, err := findWorkspaceAndBeadsID(req.AgentID)
	if err != nil {
		writeApproveError(w, fmt.Sprintf("Failed to find workspace: %v", err), http.StatusNotFound)
		return
	}

	if workspacePath == "" {
		writeApproveError(w, "Workspace not found for agent: "+req.AgentID, http.StatusNotFound)
		return
	}

	// Load or create ReviewState
	reviewState, err := verify.LoadReviewState(workspacePath)
	if err != nil {
		writeApproveError(w, fmt.Sprintf("Failed to load review state: %v", err), http.StatusInternalServerError)
		return
	}

	// Check if already approved (idempotency)
	if reviewState.IsApproved() {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ApproveResponse{
			Success: true,
			Message: "Agent already approved",
		})
		return
	}

	// Create approval comment in beads (if tracked)
	var approvalComment string
	if req.Description != "" {
		approvalComment = fmt.Sprintf("✅ APPROVED: %s", req.Description)
	} else {
		approvalComment = "✅ APPROVED - Visual changes reviewed and approved via dashboard"
	}

	if beadsID != "" {
		if err := addApprovalComment(beadsID, approvalComment); err != nil {
			// Log warning but don't fail - approval state update is still valuable
			fmt.Fprintf(os.Stderr, "Warning: failed to add approval comment to %s: %v\n", beadsID, err)
		}
	}

	// Update ReviewState with approval
	reviewState.SetApproval("orchestrator", req.Description)

	// Ensure WorkspaceID and BeadsID are set (in case this is first review state)
	if reviewState.WorkspaceID == "" {
		reviewState.WorkspaceID = filepath.Base(workspacePath)
	}
	if reviewState.BeadsID == "" && beadsID != "" {
		reviewState.BeadsID = beadsID
	}

	// Save ReviewState
	if err := verify.SaveReviewState(workspacePath, reviewState); err != nil {
		writeApproveError(w, fmt.Sprintf("Failed to save review state: %v", err), http.StatusInternalServerError)
		return
	}

	// Success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ApproveResponse{
		Success: true,
		Message: "Agent approved successfully",
	})
}

// findWorkspaceAndBeadsID finds the workspace path and beads ID for a given agent identifier.
// The identifier can be either a workspace name or a beads ID.
// Returns (workspacePath, beadsID, error).
func findWorkspaceAndBeadsID(agentID string) (string, string, error) {
	// Get current working directory as project base
	currentDir, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("failed to get working directory: %w", err)
	}

	// Try to find workspace by name first
	workspacePath := findWorkspaceByName(currentDir, agentID)
	if workspacePath != "" {
		// Found by workspace name - read beads ID from manifest (fallback handles dotfiles)
		manifest := spawn.ReadAgentManifestWithFallback(workspacePath)
		beadsID := strings.TrimSpace(manifest.BeadsID)
		if beadsID != "" {
			return workspacePath, beadsID, nil
		}
		// No beads ID - might be untracked
		return workspacePath, "", nil
	}

	// Try to resolve as beads ID
	resolvedID, err := resolveShortBeadsID(agentID)
	if err != nil {
		return "", "", fmt.Errorf("not a valid workspace name or beads ID: %s", agentID)
	}

	// Find workspace by beads ID
	workspacePath, _ = findWorkspaceByBeadsID(currentDir, resolvedID)
	return workspacePath, resolvedID, nil
}

// writeApproveError writes an error response for the approve endpoint.
func writeApproveError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ApproveResponse{
		Success: false,
		Error:   message,
	})
}
