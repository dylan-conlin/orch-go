package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// PendingReviewItem represents a single synthesis recommendation pending review.
type PendingReviewItem struct {
	WorkspaceID string `json:"workspace_id"`
	BeadsID     string `json:"beads_id"`
	Index       int    `json:"index"`     // Index of the recommendation (0-based)
	Text        string `json:"text"`      // The recommendation text
	Reviewed    bool   `json:"reviewed"`  // Whether this item has been reviewed
	ActedOn     bool   `json:"acted_on"`  // Whether an issue was created
	Dismissed   bool   `json:"dismissed"` // Whether this was dismissed
}

// PendingReviewAgent represents an agent with pending synthesis reviews.
type PendingReviewAgent struct {
	WorkspaceID          string              `json:"workspace_id"`
	WorkspacePath        string              `json:"workspace_path"`
	BeadsID              string              `json:"beads_id"`
	TLDR                 string              `json:"tldr,omitempty"`
	TotalRecommendations int                 `json:"total_recommendations"`
	UnreviewedCount      int                 `json:"unreviewed_count"`
	Items                []PendingReviewItem `json:"items"`
	IsLightTier          bool                `json:"is_light_tier,omitempty"` // True if this was a light tier spawn (no synthesis by design)
}

// PendingReviewsAPIResponse is the JSON structure returned by /api/pending-reviews.
type PendingReviewsAPIResponse struct {
	Agents          []PendingReviewAgent `json:"agents"`
	TotalAgents     int                  `json:"total_agents"`
	TotalUnreviewed int                  `json:"total_unreviewed"`
}

// handlePendingReviews returns pending synthesis reviews.
// This includes both full-tier agents with SYNTHESIS.md and light-tier agents
// that have completed (Phase: Complete) but have no synthesis by design.
//
// Performance optimization:
//  1. Light-tier processing is disabled - the PendingReviewsSection was removed from the
//     dashboard, and processing 200+ light-tier workspaces causes 15+ second API latency.
//  2. Recency filter: Workspaces older than 7 days are skipped to avoid stale data.
//
// Previously, each light-tier workspace would call isLightTierComplete which called
// GetComments individually. Even with batch fetching, 200+ beads API calls is too slow.
const pendingReviewsMaxAge = 7 * 24 * time.Hour

// skipLightTierProcessing disables light-tier completion detection in pending reviews.
// Set to false to re-enable light-tier processing (not recommended due to performance).
const skipLightTierProcessing = true

func (s *Server) handlePendingReviews(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Scan workspaces for SYNTHESIS.md and review state
	workspaceDir := filepath.Join(s.SourceDir, ".orch", "workspace")
	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		// No workspace directory is fine - just return empty response
		resp := PendingReviewsAPIResponse{
			Agents:          []PendingReviewAgent{},
			TotalAgents:     0,
			TotalUnreviewed: 0,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Phase 1: Scan workspaces and collect light-tier beads IDs for batch fetching
	type workspaceCandidate struct {
		dirName      string
		dirPath      string
		hasSynthesis bool
		isLightTier  bool
		beadsID      string // For light-tier workspaces
	}

	var candidates []workspaceCandidate
	var lightTierBeadsIDs []string
	beadsIDSet := make(map[string]bool) // Deduplicate beads IDs

	// Calculate cutoff time for recency filter
	cutoffTime := time.Now().Add(-pendingReviewsMaxAge)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirName := entry.Name()
		dirPath := filepath.Join(workspaceDir, dirName)

		// Recency filter: Skip workspaces older than 7 days
		// Check the SPAWN_CONTEXT.md modification time as the workspace creation indicator
		spawnContextPath := filepath.Join(dirPath, "SPAWN_CONTEXT.md")
		if info, err := os.Stat(spawnContextPath); err == nil {
			if info.ModTime().Before(cutoffTime) {
				continue
			}
		}

		// Check for SYNTHESIS.md (full-tier agents)
		synthesisPath := filepath.Join(dirPath, "SYNTHESIS.md")
		hasSynthesis := false
		if _, err := os.Stat(synthesisPath); err == nil {
			hasSynthesis = true
		}

		// Check if this is a light-tier workspace (has .tier file with "light")
		// Light-tier processing is disabled by default due to performance impact
		isLightTier := false
		if !skipLightTierProcessing {
			isLightTier = isLightTierWorkspace(dirPath)
		}

		// Skip workspaces that are neither full-tier with synthesis nor light-tier
		if !hasSynthesis && !isLightTier {
			continue
		}

		// For light-tier workspaces, extract beads ID for batch fetching
		var beadsID string
		if isLightTier {
			beadsID = extractBeadsIDFromWorkspace(dirPath)
			if beadsID != "" && !beadsIDSet[beadsID] {
				lightTierBeadsIDs = append(lightTierBeadsIDs, beadsID)
				beadsIDSet[beadsID] = true
			}
		}

		candidates = append(candidates, workspaceCandidate{
			dirName:      dirName,
			dirPath:      dirPath,
			hasSynthesis: hasSynthesis,
			isLightTier:  isLightTier,
			beadsID:      beadsID,
		})
	}

	// Phase 2: Batch fetch all comments for light-tier workspaces
	// This single batch call replaces O(n) individual GetComments calls
	lightTierCommentsMap := verify.GetCommentsBatch(lightTierBeadsIDs)

	// Phase 3: Process each candidate using pre-fetched comments
	var agents []PendingReviewAgent
	totalUnreviewed := 0

	for _, ws := range candidates {
		// Check for light-tier completion using pre-fetched comments
		isLightComplete := false
		lightBeadsID := ws.beadsID
		if ws.isLightTier && lightBeadsID != "" {
			if comments, ok := lightTierCommentsMap[lightBeadsID]; ok {
				phaseStatus := verify.ParsePhaseFromComments(comments)
				isLightComplete = phaseStatus.Found && strings.EqualFold(phaseStatus.Phase, "Complete")
			}
		}

		// Skip workspaces that are neither full-tier with synthesis nor light-tier complete
		if !ws.hasSynthesis && !isLightComplete {
			continue
		}

		// Handle full-tier agents with synthesis
		if ws.hasSynthesis {
			// Parse synthesis
			synthesis, err := verify.ParseSynthesis(ws.dirPath)
			if err != nil || synthesis == nil {
				continue
			}

			// Skip if no recommendations
			if len(synthesis.NextActions) == 0 {
				continue
			}

			// Load review state
			reviewState, err := verify.LoadReviewState(ws.dirPath)
			if err != nil {
				reviewState = &verify.ReviewState{}
			}

			// Extract beads ID from SPAWN_CONTEXT
			beadsID := extractBeadsIDFromWorkspace(ws.dirPath)

			// Build item list
			var items []PendingReviewItem
			unreviewedCount := 0

			for i, action := range synthesis.NextActions {
				actedOn := containsInt(reviewState.ActedOn, i)
				dismissed := containsInt(reviewState.Dismissed, i)
				reviewed := actedOn || dismissed

				if !reviewed {
					unreviewedCount++
				}

				items = append(items, PendingReviewItem{
					WorkspaceID: ws.dirName,
					BeadsID:     beadsID,
					Index:       i,
					Text:        action,
					Reviewed:    reviewed,
					ActedOn:     actedOn,
					Dismissed:   dismissed,
				})
			}

			// Only include agents with unreviewed recommendations
			if unreviewedCount > 0 {
				agents = append(agents, PendingReviewAgent{
					WorkspaceID:          ws.dirName,
					WorkspacePath:        ws.dirPath,
					BeadsID:              beadsID,
					TLDR:                 synthesis.TLDR,
					TotalRecommendations: len(synthesis.NextActions),
					UnreviewedCount:      unreviewedCount,
					Items:                items,
					IsLightTier:          false,
				})
				totalUnreviewed += unreviewedCount
			}
		} else if isLightComplete {
			// Handle light-tier agents (no synthesis by design)
			// Light-tier completions appear with a special indicator and no items
			// They still need orchestrator acknowledgment via beads close

			// Load review state to check if already acknowledged
			reviewState, err := verify.LoadReviewState(ws.dirPath)
			if err != nil {
				reviewState = &verify.ReviewState{}
			}

			// Skip if already reviewed (acknowledged)
			if reviewState.LightTierAcknowledged {
				continue
			}

			// Create a single "pseudo-item" indicating the light tier completion needs review
			items := []PendingReviewItem{
				{
					WorkspaceID: ws.dirName,
					BeadsID:     lightBeadsID,
					Index:       0,
					Text:        "Light tier agent completed - no synthesis produced (by design). Review and close via orch complete.",
					Reviewed:    false,
					ActedOn:     false,
					Dismissed:   false,
				},
			}

			agents = append(agents, PendingReviewAgent{
				WorkspaceID:          ws.dirName,
				WorkspacePath:        ws.dirPath,
				BeadsID:              lightBeadsID,
				TLDR:                 "Light tier completion - review agent output directly",
				TotalRecommendations: 1,
				UnreviewedCount:      1,
				Items:                items,
				IsLightTier:          true,
			})
			totalUnreviewed++
		}
	}

	resp := PendingReviewsAPIResponse{
		Agents:          agents,
		TotalAgents:     len(agents),
		TotalUnreviewed: totalUnreviewed,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode pending reviews: %v", err), http.StatusInternalServerError)
		return
	}
}

// containsInt checks if a slice contains a value.
func containsInt(slice []int, val int) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

// isLightTierWorkspace checks if a workspace is a light tier spawn.
// Light tier workspaces have a .tier file containing "light".
func isLightTierWorkspace(workspacePath string) bool {
	tierPath := filepath.Join(workspacePath, ".tier")
	data, err := os.ReadFile(tierPath)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(data)) == "light"
}

// isLightTierComplete checks if a light tier workspace has Phase: Complete in beads comments.
// Returns true if the workspace is light tier AND has Phase: Complete.
func isLightTierComplete(workspacePath string) (isComplete bool, beadsID string) {
	if !isLightTierWorkspace(workspacePath) {
		return false, ""
	}

	// Extract beads ID from SPAWN_CONTEXT.md
	beadsID = extractBeadsIDFromWorkspace(workspacePath)
	if beadsID == "" {
		return false, ""
	}

	// Get comments for this beads ID
	comments, err := verify.GetComments(beadsID)
	if err != nil {
		return false, beadsID
	}

	// Check for Phase: Complete
	phaseStatus := verify.ParsePhaseFromComments(comments)
	return phaseStatus.Found && strings.EqualFold(phaseStatus.Phase, "Complete"), beadsID
}

// DismissReviewRequest is the request body for POST /api/dismiss-review.
type DismissReviewRequest struct {
	WorkspaceID string `json:"workspace_id"`
	Index       int    `json:"index"` // Index of the recommendation to dismiss
}

// DismissReviewResponse is the response for POST /api/dismiss-review.
type DismissReviewResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// handleDismissReview dismisses a synthesis recommendation.
func (s *Server) handleDismissReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DismissReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DismissReviewResponse{
			Success: false,
			Error:   fmt.Sprintf("Invalid request body: %v", err),
		})
		return
	}

	if req.WorkspaceID == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DismissReviewResponse{
			Success: false,
			Error:   "workspace_id is required",
		})
		return
	}

	// Build workspace path
	workspacePath := filepath.Join(s.SourceDir, ".orch", "workspace", req.WorkspaceID)

	// Check for light-tier workspace - these don't have SYNTHESIS.md by design.
	// Light-tier dismissals set LightTierAcknowledged instead of tracking individual recommendations.
	if isLightTierWorkspace(workspacePath) {
		reviewState, err := verify.LoadReviewState(workspacePath)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(DismissReviewResponse{
				Success: false,
				Error:   fmt.Sprintf("Failed to load review state: %v", err),
			})
			return
		}

		// Check if already acknowledged
		if reviewState.LightTierAcknowledged {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(DismissReviewResponse{
				Success: true,
				Message: "Already acknowledged",
			})
			return
		}

		// Mark as acknowledged
		reviewState.LightTierAcknowledged = true
		reviewState.WorkspaceID = req.WorkspaceID
		reviewState.BeadsID = extractBeadsIDFromWorkspace(workspacePath)
		if reviewState.ReviewedAt.IsZero() {
			reviewState.ReviewedAt = time.Now()
		}

		if err := verify.SaveReviewState(workspacePath, reviewState); err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(DismissReviewResponse{
				Success: false,
				Error:   fmt.Sprintf("Failed to save review state: %v", err),
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DismissReviewResponse{
			Success: true,
			Message: "Light tier completion acknowledged",
		})
		return
	}

	// Full-tier path: Load existing review state
	reviewState, err := verify.LoadReviewState(workspacePath)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DismissReviewResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to load review state: %v", err),
		})
		return
	}

	// Parse synthesis to get total recommendations
	synthesis, err := verify.ParseSynthesis(workspacePath)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DismissReviewResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to parse synthesis: %v", err),
		})
		return
	}

	// Validate index
	if req.Index < 0 || req.Index >= len(synthesis.NextActions) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DismissReviewResponse{
			Success: false,
			Error:   fmt.Sprintf("Invalid index %d (total recommendations: %d)", req.Index, len(synthesis.NextActions)),
		})
		return
	}

	// Check if already reviewed
	if containsInt(reviewState.ActedOn, req.Index) || containsInt(reviewState.Dismissed, req.Index) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DismissReviewResponse{
			Success: true,
			Message: "Already reviewed",
		})
		return
	}

	// Add to dismissed
	reviewState.Dismissed = append(reviewState.Dismissed, req.Index)
	reviewState.TotalRecommendations = len(synthesis.NextActions)
	reviewState.WorkspaceID = req.WorkspaceID
	if reviewState.ReviewedAt.IsZero() {
		reviewState.ReviewedAt = time.Now()
	}

	// Extract beads ID if not set
	if reviewState.BeadsID == "" {
		reviewState.BeadsID = extractBeadsIDFromWorkspace(workspacePath)
	}

	// Save updated state
	if err := verify.SaveReviewState(workspacePath, reviewState); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DismissReviewResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to save review state: %v", err),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(DismissReviewResponse{
		Success: true,
		Message: fmt.Sprintf("Dismissed recommendation %d", req.Index),
	})
}
