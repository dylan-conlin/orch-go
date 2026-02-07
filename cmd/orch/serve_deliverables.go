package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// DeliverableOverrideRequest is the request body for POST /api/deliverables/override.
type DeliverableOverrideRequest struct {
	BeadsID    string            `json:"beads_id"`
	Reasons    map[string]string `json:"reasons"`               // Map of deliverable type -> reason for override
	OverrideBy string            `json:"override_by,omitempty"` // "orchestrator" or "user"
}

// DeliverableOverrideResponse is the response for POST /api/deliverables/override.
type DeliverableOverrideResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// handleDeliverablesStatus handles GET /api/deliverables/{beads-id}
// Returns the deliverables status for a specific issue.
func (s *Server) handleDeliverablesStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract beads ID from path: /api/deliverables/{beads-id}
	path := strings.TrimPrefix(r.URL.Path, "/api/deliverables/")
	if path == "" || strings.Contains(path, "/") {
		http.Error(w, "beads ID required", http.StatusBadRequest)
		return
	}
	beadsID := path

	// Get issue info from beads using fallback (CLI) method
	issue, err := beads.FallbackShow(beadsID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get issue: %v", err), http.StatusInternalServerError)
		return
	}

	// Find workspace for this issue
	workspacePath := ""
	projectDir, _ := s.currentProjectDir()

	// Try to find workspace
	workspaces, err := findWorkspacesForBeadsID(beadsID)
	if err == nil && len(workspaces) > 0 {
		workspacePath = workspaces[0]
	}

	// Get skill from workspace
	skill := ""
	if workspacePath != "" {
		skill, _ = verify.ExtractSkillNameFromSpawnContext(workspacePath)
	}

	// Get beads comments for evidence detection using fallback (CLI) method
	var comments []verify.Comment
	rawComments, err := beads.FallbackComments(beadsID)
	if err == nil {
		for _, c := range rawComments {
			comments = append(comments, verify.Comment{Text: c.Text})
		}
	}

	// Check deliverables
	result, err := verify.CheckDeliverables(beadsID, issue.IssueType, skill, workspacePath, projectDir, comments)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to check deliverables: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleDeliverablesOverride handles POST /api/deliverables/override
// Logs an override when closing with missing deliverables.
func (s *Server) handleDeliverablesOverride(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DeliverableOverrideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.BeadsID == "" {
		http.Error(w, "beads_id is required", http.StatusBadRequest)
		return
	}

	if len(req.Reasons) == 0 {
		http.Error(w, "reasons are required for override", http.StatusBadRequest)
		return
	}

	// Get issue info for metadata using fallback (CLI) method
	issue, err := beads.FallbackShow(req.BeadsID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get issue: %v", err), http.StatusInternalServerError)
		return
	}

	// Get deliverables status to capture current state
	workspacePath := ""
	projectDir, _ := s.currentProjectDir()

	workspaces, err := findWorkspacesForBeadsID(req.BeadsID)
	if err == nil && len(workspaces) > 0 {
		workspacePath = workspaces[0]
	}

	skill := ""
	if workspacePath != "" {
		skill, _ = verify.ExtractSkillNameFromSpawnContext(workspacePath)
	}

	var comments []verify.Comment
	rawComments, _ := beads.FallbackComments(req.BeadsID)
	for _, c := range rawComments {
		comments = append(comments, verify.Comment{Text: c.Text})
	}

	result, _ := verify.CheckDeliverables(req.BeadsID, issue.IssueType, skill, workspacePath, projectDir, comments)

	// Collect missing deliverables and reasons
	var missing []string
	var reasons []string
	for dtype, reason := range req.Reasons {
		missing = append(missing, dtype)
		reasons = append(reasons, reason)
	}

	// Log the override
	logger := events.NewDefaultLogger()
	overrideBy := req.OverrideBy
	if overrideBy == "" {
		overrideBy = "user"
	}

	totalRequired := 0
	totalSatisfied := 0
	if result != nil {
		totalRequired = result.Required
		totalSatisfied = result.Satisfied
	}

	if err := logger.LogDeliverableOverride(events.DeliverableOverrideData{
		BeadsID:        req.BeadsID,
		IssueType:      issue.IssueType,
		Skill:          skill,
		Missing:        missing,
		Reasons:        reasons,
		OverrideBy:     overrideBy,
		TotalRequired:  totalRequired,
		TotalSatisfied: totalSatisfied,
	}); err != nil {
		http.Error(w, fmt.Sprintf("failed to log override: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(DeliverableOverrideResponse{
		Success: true,
		Message: fmt.Sprintf("Override logged for %d missing deliverables", len(missing)),
	})
}

// findWorkspacesForBeadsID finds workspace paths associated with a beads ID.
// Returns a slice of workspace paths (may be empty).
func findWorkspacesForBeadsID(beadsID string) ([]string, error) {
	// Look in standard workspace locations
	// Workspace names typically contain the beads ID
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	var workspaces []string

	// Check active workspaces
	for _, searchPath := range []string{
		// Project-local workspaces
		".orch/workspace",
		// Global orchestrator workspaces
		home + "/.orch/workspace",
	} {
		entries, err := os.ReadDir(searchPath)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() && strings.Contains(entry.Name(), beadsID) {
				workspaces = append(workspaces, fmt.Sprintf("%s/%s", searchPath, entry.Name()))
			}
		}
	}

	return workspaces, nil
}
