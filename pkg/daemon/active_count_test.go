// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"testing"
)

func TestExtractProjectFromBeadsID(t *testing.T) {
	tests := []struct {
		name     string
		beadsID  string
		expected string
	}{
		{
			name:     "standard beads ID",
			beadsID:  "orch-go-abc1",
			expected: "orch-go",
		},
		{
			name:     "hyphenated project name",
			beadsID:  "kb-cli-xyz9",
			expected: "kb-cli",
		},
		{
			name:     "longer hash",
			beadsID:  "my-project-12345",
			expected: "my-project",
		},
		{
			name:     "single char hash",
			beadsID:  "test-a",
			expected: "test",
		},
		{
			name:     "no dash",
			beadsID:  "nodash",
			expected: "",
		},
		{
			name:     "empty string",
			beadsID:  "",
			expected: "",
		},
		{
			name:     "just a dash",
			beadsID:  "-",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractProjectFromBeadsID(tt.beadsID)
			if result != tt.expected {
				t.Errorf("extractProjectFromBeadsID(%q) = %q, want %q",
					tt.beadsID, result, tt.expected)
			}
		})
	}
}

func TestGroupBeadsIDsByProject(t *testing.T) {
	projectPaths := map[string]string{
		"orch-go": "/Users/test/orch-go",
		"kb-cli":  "/Users/test/kb-cli",
	}

	beadsIDs := []string{
		"orch-go-abc1",
		"kb-cli-xyz2",
		"orch-go-def3",
		"unknown-proj-ghi4", // Not in projectPaths
	}

	grouped := groupBeadsIDsByProject(beadsIDs, projectPaths)

	// Check orch-go group
	orchGoIDs := grouped["/Users/test/orch-go"]
	if len(orchGoIDs) != 2 {
		t.Errorf("expected 2 orch-go IDs, got %d", len(orchGoIDs))
	}

	// Check kb-cli group
	kbCliIDs := grouped["/Users/test/kb-cli"]
	if len(kbCliIDs) != 1 {
		t.Errorf("expected 1 kb-cli ID, got %d", len(kbCliIDs))
	}

	// Check unknown project (should be grouped under "" for current dir)
	unknownIDs := grouped[""]
	if len(unknownIDs) != 1 {
		t.Errorf("expected 1 unknown project ID, got %d", len(unknownIDs))
	}
}

func TestBuildProjectPathMap(t *testing.T) {
	// This test verifies the function doesn't panic when kb is unavailable
	// In a real environment, it would return projects from kb projects list
	pathMap := buildProjectPathMap()
	if pathMap == nil {
		t.Error("buildProjectPathMap() returned nil, expected empty map")
	}
	// Note: Can't test actual content without mocking ListProjects()
}

func TestExtractBeadsIDFromSessionTitle(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		expected string
	}{
		{
			name:     "standard format",
			title:    "og-feat-add-feature-24dec [orch-go-3anf]",
			expected: "orch-go-3anf",
		},
		{
			name:     "spaces in beads ID",
			title:    "workspace [proj-abc ]",
			expected: "proj-abc",
		},
		{
			name:     "no brackets",
			title:    "og-feat-add-feature-24dec",
			expected: "",
		},
		{
			name:     "empty brackets",
			title:    "workspace []",
			expected: "",
		},
		{
			name:     "nested brackets",
			title:    "workspace [outer [proj-123]]",
			expected: "proj-123]", // Takes from last [ to last ]
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractBeadsIDFromSessionTitle(tt.title)
			if result != tt.expected {
				t.Errorf("extractBeadsIDFromSessionTitle(%q) = %q, want %q",
					tt.title, result, tt.expected)
			}
		})
	}
}

func TestIsUntrackedBeadsID(t *testing.T) {
	tests := []struct {
		name     string
		beadsID  string
		expected bool
	}{
		{
			name:     "tracked ID",
			beadsID:  "orch-go-abc1",
			expected: false,
		},
		{
			name:     "untracked ID",
			beadsID:  "orch-go-untracked-1234567890",
			expected: true,
		},
		{
			name:     "empty string",
			beadsID:  "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isUntrackedBeadsID(tt.beadsID)
			if result != tt.expected {
				t.Errorf("isUntrackedBeadsID(%q) = %v, want %v",
					tt.beadsID, result, tt.expected)
			}
		})
	}
}

func TestGetClosedIssuesBatchWithProjectDirs_EmptyInput(t *testing.T) {
	// Test with nil beads IDs
	result := GetClosedIssuesBatchWithProjectDirs(nil, nil)
	if len(result) != 0 {
		t.Errorf("expected empty map for nil input, got %v", result)
	}

	// Test with empty slice
	result = GetClosedIssuesBatchWithProjectDirs([]string{}, nil)
	if len(result) != 0 {
		t.Errorf("expected empty map for empty input, got %v", result)
	}

	// Test with nil projectDirs (should use kb projects fallback)
	result = GetClosedIssuesBatchWithProjectDirs([]string{"orch-go-abc1"}, nil)
	// Note: This won't actually find closed issues without a beads daemon,
	// but it shouldn't panic
	if result == nil {
		t.Error("expected non-nil map, got nil")
	}
}

func TestGetClosedIssuesBatchWithProjectDirs_ProjectDirsUsed(t *testing.T) {
	// This test verifies that explicit projectDirs are preferred over kb projects.
	// We can't test the actual beads lookup without a daemon, but we can verify
	// the function doesn't panic and handles the input correctly.

	beadsIDs := []string{
		"proj-a-abc1",
		"proj-b-xyz2",
	}

	projectDirs := map[string]string{
		"proj-a-abc1": "/path/to/proj-a",
		// proj-b-xyz2 is intentionally missing - should fall back to kb projects
	}

	// This won't find actual issues (no daemon), but it tests the code path
	result := GetClosedIssuesBatchWithProjectDirs(beadsIDs, projectDirs)

	// Should return a valid (possibly empty) map
	if result == nil {
		t.Error("expected non-nil result map")
	}
}

func TestGetClosedIssuesBatchWithProjectDirs_LookupFailuresTreatedAsClosed(t *testing.T) {
	// This test verifies the critical behavior fix: when beads lookups fail,
	// the issues should be treated as "closed" to prevent capacity leaks.
	// Without this behavior, lookup failures would cause issues to be incorrectly
	// counted as "active", preventing reconciliation from freeing slots.

	beadsIDs := []string{
		"nonexistent-proj-abc1",
		"another-missing-xyz2",
	}

	// Use invalid paths that will definitely fail lookup
	projectDirs := map[string]string{
		"nonexistent-proj-abc1": "/definitely/not/a/real/path",
		"another-missing-xyz2":  "/also/not/real",
	}

	result := GetClosedIssuesBatchWithProjectDirs(beadsIDs, projectDirs)

	// With the fix: lookup failures should add to closed map
	// Both issues should be marked as closed (lookup failed)
	if len(result) != 2 {
		t.Errorf("expected 2 issues marked as closed (due to lookup failure), got %d", len(result))
	}

	for _, id := range beadsIDs {
		if !result[id] {
			t.Errorf("expected %s to be marked as closed (lookup failure), but it wasn't", id)
		}
	}
}
