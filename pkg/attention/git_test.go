package attention

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

func TestExtractIssueIDs(t *testing.T) {
	tests := []struct {
		name     string
		commits  []commitInfo
		expected map[string]int // issue ID -> commit count
	}{
		{
			name: "single issue multiple commits",
			commits: []commitInfo{
				{Hash: "abc123", Message: "feat: implement orch-go-12345 feature", Timestamp: "2024-01-01"},
				{Hash: "def456", Message: "fix: resolve orch-go-12345 bug", Timestamp: "2024-01-02"},
			},
			expected: map[string]int{
				"orch-go-12345": 2,
			},
		},
		{
			name: "multiple issues single commit",
			commits: []commitInfo{
				{Hash: "abc123", Message: "feat: merge orch-go-12345 and og-67890", Timestamp: "2024-01-01"},
			},
			expected: map[string]int{
				"orch-go-12345": 1,
				"og-67890":      1,
			},
		},
		{
			name: "no issue IDs",
			commits: []commitInfo{
				{Hash: "abc123", Message: "feat: add new feature", Timestamp: "2024-01-01"},
			},
			expected: map[string]int{},
		},
		{
			name: "mixed formats",
			commits: []commitInfo{
				{Hash: "abc123", Message: "feat: bd-a3f8 specs-platform-10", Timestamp: "2024-01-01"},
			},
			expected: map[string]int{
				"bd-a3f8":           1,
				"specs-platform-10": 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractIssueIDs(tt.commits)

			// Check expected issues are present with correct count
			for issueID, expectedCount := range tt.expected {
				commits, ok := result[issueID]
				if !ok {
					t.Errorf("Expected issue ID %q not found in result", issueID)
					continue
				}
				if len(commits) != expectedCount {
					t.Errorf("Issue %q: expected %d commits, got %d", issueID, expectedCount, len(commits))
				}
			}

			// Check no unexpected issues
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d issues, got %d", len(tt.expected), len(result))
			}
		})
	}
}

func TestGetActiveWorkspaces(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace")

	// Create workspace with .beads_id
	ws1 := filepath.Join(workspaceDir, "og-feat-test-123")
	os.MkdirAll(ws1, 0755)
	os.WriteFile(filepath.Join(ws1, ".beads_id"), []byte("test-issue-1"), 0644)

	// Create workspace without .beads_id (should be ignored)
	ws2 := filepath.Join(workspaceDir, "og-feat-test-456")
	os.MkdirAll(ws2, 0755)

	// Create archived directory (should be ignored)
	archived := filepath.Join(workspaceDir, "archived")
	os.MkdirAll(archived, 0755)
	archivedWS := filepath.Join(archived, "og-feat-old")
	os.MkdirAll(archivedWS, 0755)
	os.WriteFile(filepath.Join(archivedWS, ".beads_id"), []byte("old-issue"), 0644)

	// Test
	result, err := getActiveWorkspaces(tmpDir)
	if err != nil {
		t.Fatalf("getActiveWorkspaces failed: %v", err)
	}

	// Verify results
	if len(result) != 1 {
		t.Errorf("Expected 1 active workspace, got %d", len(result))
	}

	if path, ok := result["test-issue-1"]; !ok {
		t.Errorf("Expected test-issue-1 in results")
	} else if path != ws1 {
		t.Errorf("Expected path %s, got %s", ws1, path)
	}

	if _, ok := result["old-issue"]; ok {
		t.Errorf("Archived workspace should not be in results")
	}
}

func TestLikelyDoneCache(t *testing.T) {
	cache := NewLikelyDoneCache()

	if cache == nil {
		t.Fatal("NewLikelyDoneCache returned nil")
	}

	// Verify TTL is set
	if cache.ttl == 0 {
		t.Error("Cache TTL should be non-zero")
	}
}

func TestCollectLikelyDoneSignalsEmptyRepo(t *testing.T) {
	// Create a temporary directory (not a git repo)
	tmpDir := t.TempDir()

	// Use mock client
	mockClient := beads.NewMockClient()

	// Should handle gracefully
	result, err := CollectLikelyDoneSignals(tmpDir, mockClient)
	if err != nil {
		t.Fatalf("CollectLikelyDoneSignals should handle non-git directories gracefully: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if result.Total != 0 {
		t.Errorf("Expected 0 signals for non-git directory, got %d", result.Total)
	}
}
