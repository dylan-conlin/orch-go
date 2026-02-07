// Package verify provides verification helpers for agent completion.
package verify

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// ReviewStateFile is the filename for review state persistence.
const ReviewStateFile = ".review-state.json"

// ReviewState tracks the review status of synthesis recommendations for an agent workspace.
// This enables sharing review state between CLI and dashboard to avoid duplicate prompts.
type ReviewState struct {
	// ReviewedAt is when the orchestrator reviewed this synthesis
	ReviewedAt time.Time `json:"reviewed_at,omitempty"`

	// ActedOn contains indices of recommendations that became beads issues
	ActedOn []int `json:"acted_on,omitempty"`

	// Dismissed contains indices of recommendations explicitly skipped
	Dismissed []int `json:"dismissed,omitempty"`

	// WorkspaceID is the workspace directory name (for reference)
	WorkspaceID string `json:"workspace_id,omitempty"`

	// BeadsID is the beads issue ID (for reference)
	BeadsID string `json:"beads_id,omitempty"`

	// TotalRecommendations is the number of recommendations at review time
	TotalRecommendations int `json:"total_recommendations,omitempty"`

	// LightTierAcknowledged is true when a light-tier agent completion has been reviewed.
	// Light-tier agents don't produce SYNTHESIS.md, so this field tracks acknowledgment
	// of the completion itself rather than synthesis recommendations.
	LightTierAcknowledged bool `json:"light_tier_acknowledged,omitempty"`
}

// IsReviewed returns true if this synthesis has been reviewed.
func (rs *ReviewState) IsReviewed() bool {
	return !rs.ReviewedAt.IsZero()
}

// AllActedOn returns true if all recommendations were acted on (became issues).
func (rs *ReviewState) AllActedOn() bool {
	if rs.TotalRecommendations == 0 {
		return true
	}
	return len(rs.ActedOn) == rs.TotalRecommendations
}

// AllDismissed returns true if all recommendations were dismissed.
func (rs *ReviewState) AllDismissed() bool {
	if rs.TotalRecommendations == 0 {
		return true
	}
	return len(rs.Dismissed) == rs.TotalRecommendations
}

// ReviewedCount returns the number of recommendations that have been reviewed (acted on or dismissed).
func (rs *ReviewState) ReviewedCount() int {
	return len(rs.ActedOn) + len(rs.Dismissed)
}

// UnreviewedCount returns the number of recommendations not yet reviewed.
func (rs *ReviewState) UnreviewedCount() int {
	reviewed := rs.ReviewedCount()
	if reviewed >= rs.TotalRecommendations {
		return 0
	}
	return rs.TotalRecommendations - reviewed
}

// LoadReviewState loads the review state from a workspace directory.
// Returns an empty ReviewState if the file doesn't exist.
func LoadReviewState(workspacePath string) (*ReviewState, error) {
	statePath := filepath.Join(workspacePath, ReviewStateFile)
	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &ReviewState{}, nil
		}
		return nil, err
	}

	var rs ReviewState
	if err := json.Unmarshal(data, &rs); err != nil {
		return nil, err
	}

	return &rs, nil
}

// SaveReviewState saves the review state to a workspace directory.
func SaveReviewState(workspacePath string, state *ReviewState) error {
	statePath := filepath.Join(workspacePath, ReviewStateFile)
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(statePath, data, 0644)
}

// ReviewStateFromCompletion creates a new ReviewState after reviewing recommendations.
// actedOnIndices are the 0-based indices of recommendations that became issues.
// dismissedIndices are the 0-based indices of recommendations that were skipped.
func ReviewStateFromCompletion(workspaceID, beadsID string, totalRecommendations int, actedOnIndices, dismissedIndices []int) *ReviewState {
	return &ReviewState{
		ReviewedAt:           time.Now(),
		ActedOn:              actedOnIndices,
		Dismissed:            dismissedIndices,
		WorkspaceID:          workspaceID,
		BeadsID:              beadsID,
		TotalRecommendations: totalRecommendations,
	}
}

// GetReviewStatePath returns the full path to the review state file for a workspace.
func GetReviewStatePath(workspacePath string) string {
	return filepath.Join(workspacePath, ReviewStateFile)
}
