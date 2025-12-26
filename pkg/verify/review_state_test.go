package verify

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReviewState_IsReviewed(t *testing.T) {
	tests := []struct {
		name     string
		state    ReviewState
		expected bool
	}{
		{
			name:     "empty state is not reviewed",
			state:    ReviewState{},
			expected: false,
		},
		{
			name: "state with ReviewedAt is reviewed",
			state: ReviewState{
				ReviewedAt: time.Now(),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.state.IsReviewed(); got != tt.expected {
				t.Errorf("IsReviewed() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestReviewState_AllActedOn(t *testing.T) {
	tests := []struct {
		name     string
		state    ReviewState
		expected bool
	}{
		{
			name:     "empty recommendations means all acted on",
			state:    ReviewState{TotalRecommendations: 0},
			expected: true,
		},
		{
			name: "all indices in ActedOn",
			state: ReviewState{
				TotalRecommendations: 3,
				ActedOn:              []int{0, 1, 2},
			},
			expected: true,
		},
		{
			name: "partial ActedOn",
			state: ReviewState{
				TotalRecommendations: 3,
				ActedOn:              []int{0, 1},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.state.AllActedOn(); got != tt.expected {
				t.Errorf("AllActedOn() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestReviewState_AllDismissed(t *testing.T) {
	tests := []struct {
		name     string
		state    ReviewState
		expected bool
	}{
		{
			name:     "empty recommendations means all dismissed",
			state:    ReviewState{TotalRecommendations: 0},
			expected: true,
		},
		{
			name: "all indices in Dismissed",
			state: ReviewState{
				TotalRecommendations: 2,
				Dismissed:            []int{0, 1},
			},
			expected: true,
		},
		{
			name: "partial Dismissed",
			state: ReviewState{
				TotalRecommendations: 3,
				Dismissed:            []int{0},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.state.AllDismissed(); got != tt.expected {
				t.Errorf("AllDismissed() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestReviewState_ReviewedCount(t *testing.T) {
	state := ReviewState{
		ActedOn:   []int{0, 1},
		Dismissed: []int{2},
	}
	if got := state.ReviewedCount(); got != 3 {
		t.Errorf("ReviewedCount() = %v, want 3", got)
	}
}

func TestReviewState_UnreviewedCount(t *testing.T) {
	tests := []struct {
		name     string
		state    ReviewState
		expected int
	}{
		{
			name: "some unreviewed",
			state: ReviewState{
				TotalRecommendations: 5,
				ActedOn:              []int{0, 1},
				Dismissed:            []int{2},
			},
			expected: 2,
		},
		{
			name: "all reviewed",
			state: ReviewState{
				TotalRecommendations: 3,
				ActedOn:              []int{0},
				Dismissed:            []int{1, 2},
			},
			expected: 0,
		},
		{
			name: "over-reviewed returns 0",
			state: ReviewState{
				TotalRecommendations: 2,
				ActedOn:              []int{0, 1, 2, 3},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.state.UnreviewedCount(); got != tt.expected {
				t.Errorf("UnreviewedCount() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestLoadSaveReviewState(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create a review state
	original := &ReviewState{
		ReviewedAt:           time.Now().Truncate(time.Second), // JSON loses nanoseconds
		ActedOn:              []int{0, 2},
		Dismissed:            []int{1},
		WorkspaceID:          "test-workspace",
		BeadsID:              "test-123",
		TotalRecommendations: 3,
	}

	// Save it
	err := SaveReviewState(tmpDir, original)
	if err != nil {
		t.Fatalf("SaveReviewState() error = %v", err)
	}

	// Verify file exists
	statePath := filepath.Join(tmpDir, ReviewStateFile)
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Fatalf("Review state file not created")
	}

	// Load it back
	loaded, err := LoadReviewState(tmpDir)
	if err != nil {
		t.Fatalf("LoadReviewState() error = %v", err)
	}

	// Verify fields
	if !loaded.ReviewedAt.Equal(original.ReviewedAt) {
		t.Errorf("ReviewedAt mismatch: got %v, want %v", loaded.ReviewedAt, original.ReviewedAt)
	}
	if len(loaded.ActedOn) != len(original.ActedOn) {
		t.Errorf("ActedOn length mismatch: got %d, want %d", len(loaded.ActedOn), len(original.ActedOn))
	}
	if len(loaded.Dismissed) != len(original.Dismissed) {
		t.Errorf("Dismissed length mismatch: got %d, want %d", len(loaded.Dismissed), len(original.Dismissed))
	}
	if loaded.WorkspaceID != original.WorkspaceID {
		t.Errorf("WorkspaceID mismatch: got %s, want %s", loaded.WorkspaceID, original.WorkspaceID)
	}
	if loaded.BeadsID != original.BeadsID {
		t.Errorf("BeadsID mismatch: got %s, want %s", loaded.BeadsID, original.BeadsID)
	}
	if loaded.TotalRecommendations != original.TotalRecommendations {
		t.Errorf("TotalRecommendations mismatch: got %d, want %d", loaded.TotalRecommendations, original.TotalRecommendations)
	}
}

func TestLoadReviewState_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()

	// Loading from non-existent file should return empty state, not error
	state, err := LoadReviewState(tmpDir)
	if err != nil {
		t.Fatalf("LoadReviewState() error = %v", err)
	}
	if state.IsReviewed() {
		t.Error("Expected empty state to not be reviewed")
	}
}

func TestReviewStateFromCompletion(t *testing.T) {
	state := ReviewStateFromCompletion(
		"my-workspace",
		"beads-abc",
		5,
		[]int{0, 1, 2},
		[]int{3, 4},
	)

	if state.WorkspaceID != "my-workspace" {
		t.Errorf("WorkspaceID = %s, want my-workspace", state.WorkspaceID)
	}
	if state.BeadsID != "beads-abc" {
		t.Errorf("BeadsID = %s, want beads-abc", state.BeadsID)
	}
	if state.TotalRecommendations != 5 {
		t.Errorf("TotalRecommendations = %d, want 5", state.TotalRecommendations)
	}
	if len(state.ActedOn) != 3 {
		t.Errorf("ActedOn length = %d, want 3", len(state.ActedOn))
	}
	if len(state.Dismissed) != 2 {
		t.Errorf("Dismissed length = %d, want 2", len(state.Dismissed))
	}
	if !state.IsReviewed() {
		t.Error("Expected state to be reviewed")
	}
}

func TestGetReviewStatePath(t *testing.T) {
	path := GetReviewStatePath("/some/workspace/path")
	expected := "/some/workspace/path/.review-state.json"
	if path != expected {
		t.Errorf("GetReviewStatePath() = %s, want %s", path, expected)
	}
}
