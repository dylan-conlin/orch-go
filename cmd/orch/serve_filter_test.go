package main

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestParseSinceParam(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected time.Duration
	}{
		{
			name:     "default_no_param",
			query:    "",
			expected: 12 * time.Hour,
		},
		{
			name:     "all_returns_zero",
			query:    "?since=all",
			expected: 0,
		},
		{
			name:     "12h",
			query:    "?since=12h",
			expected: 12 * time.Hour,
		},
		{
			name:     "24h",
			query:    "?since=24h",
			expected: 24 * time.Hour,
		},
		{
			name:     "7d",
			query:    "?since=7d",
			expected: 7 * 24 * time.Hour,
		},
		{
			name:     "invalid_returns_default",
			query:    "?since=invalid",
			expected: 12 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/agents"+tt.query, nil)
			got := parseSinceParam(req)
			if got != tt.expected {
				t.Errorf("parseSinceParam() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseProjectFilter(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected []string
	}{
		{
			name:     "no_filter",
			query:    "",
			expected: nil,
		},
		{
			name:     "single_project_name",
			query:    "?project=orch-go",
			expected: []string{"orch-go"},
		},
		{
			name:     "single_full_path",
			query:    "?project=/Users/dylan/orch-go",
			expected: []string{"/Users/dylan/orch-go"},
		},
		{
			name:     "multiple_projects_comma_separated",
			query:    "?project=orch-go,orch-cli,beads",
			expected: []string{"orch-go", "orch-cli", "beads"},
		},
		{
			name:     "multiple_projects_with_whitespace",
			query:    "?project=orch-go,%20orch-cli%20,%20beads",
			expected: []string{"orch-go", "orch-cli", "beads"},
		},
		{
			name:     "empty_values_filtered_out",
			query:    "?project=orch-go,,beads",
			expected: []string{"orch-go", "beads"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/agents"+tt.query, nil)
			got := parseProjectFilter(req)
			if len(got) != len(tt.expected) {
				t.Errorf("parseProjectFilter() len = %d, want %d", len(got), len(tt.expected))
				return
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("parseProjectFilter()[%d] = %v, want %v", i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestFilterByProjectDir(t *testing.T) {
	tests := []struct {
		name       string
		projectDir string
		filters    []string
		expected   bool
	}{
		// Empty filter cases - no filtering
		{
			name:       "empty_filters_returns_true",
			projectDir: "/Users/dylan/orch-go",
			filters:    nil,
			expected:   true,
		},
		{
			name:       "empty_filters_slice_returns_true",
			projectDir: "/Users/dylan/orch-go",
			filters:    []string{},
			expected:   true,
		},
		// Empty projectDir cases - should not match
		{
			name:       "empty_projectDir_returns_false",
			projectDir: "",
			filters:    []string{"orch-go"},
			expected:   false,
		},
		// Single filter - full path matching
		{
			name:       "single_filter_full_path_match",
			projectDir: "/Users/dylan/orch-go",
			filters:    []string{"/Users/dylan/orch-go"},
			expected:   true,
		},
		{
			name:       "single_filter_full_path_no_match",
			projectDir: "/Users/dylan/orch-go",
			filters:    []string{"/Users/dylan/kb-cli"},
			expected:   false,
		},
		// Single filter - project name matching
		{
			name:       "single_filter_project_name_match",
			projectDir: "/Users/dylan/orch-go",
			filters:    []string{"orch-go"},
			expected:   true,
		},
		{
			name:       "single_filter_project_name_no_match",
			projectDir: "/Users/dylan/orch-go",
			filters:    []string{"kb-cli"},
			expected:   false,
		},
		// Multi-project filtering - matches ANY filter
		{
			name:       "multi_filter_first_match",
			projectDir: "/Users/dylan/orch-go",
			filters:    []string{"orch-go", "orch-cli", "beads"},
			expected:   true,
		},
		{
			name:       "multi_filter_middle_match",
			projectDir: "/Users/dylan/orch-cli",
			filters:    []string{"orch-go", "orch-cli", "beads"},
			expected:   true,
		},
		{
			name:       "multi_filter_last_match",
			projectDir: "/Users/dylan/beads",
			filters:    []string{"orch-go", "orch-cli", "beads"},
			expected:   true,
		},
		{
			name:       "multi_filter_no_match",
			projectDir: "/Users/dylan/kb-cli",
			filters:    []string{"orch-go", "orch-cli", "beads"},
			expected:   false,
		},
		// Cross-project scenario: filter by project name when projectDir is from workspace cache
		// This is the key scenario for --workdir spawns
		{
			name:       "cross_project_workdir_match",
			projectDir: "/Users/dylan/kb-cli", // Actual target project from workspace cache
			filters:    []string{"kb-cli"},    // Filter by project name
			expected:   true,
		},
		{
			name:       "cross_project_workdir_filter_different_project",
			projectDir: "/Users/dylan/kb-cli", // Actual target project from workspace cache
			filters:    []string{"orch-go"},   // Filter for different project
			expected:   false,
		},
		// Trailing slash handling - extractProjectName handles this
		{
			name:       "trailing_slash_full_path",
			projectDir: "/Users/dylan/orch-go/",
			filters:    []string{"/Users/dylan/orch-go"},
			expected:   true, // extractProjectName handles trailing slash, both resolve to "orch-go"
		},
		{
			name:       "trailing_slash_project_name",
			projectDir: "/Users/dylan/orch-go/",
			filters:    []string{"orch-go"},
			expected:   true, // extractProjectName handles trailing slash
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterByProject(tt.projectDir, tt.filters)
			if got != tt.expected {
				t.Errorf("filterByProject(%q, %v) = %v, want %v", tt.projectDir, tt.filters, got, tt.expected)
			}
		})
	}
}

func TestExtractProjectName(t *testing.T) {
	tests := []struct {
		name     string
		dir      string
		expected string
	}{
		{
			name:     "empty_string",
			dir:      "",
			expected: "",
		},
		{
			name:     "simple_path",
			dir:      "/Users/dylan/orch-go",
			expected: "orch-go",
		},
		{
			name:     "trailing_slash",
			dir:      "/Users/dylan/orch-go/",
			expected: "orch-go",
		},
		{
			name:     "just_name",
			dir:      "orch-go",
			expected: "orch-go",
		},
		{
			name:     "root_path",
			dir:      "/",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractProjectName(tt.dir)
			if got != tt.expected {
				t.Errorf("extractProjectName(%q) = %q, want %q", tt.dir, got, tt.expected)
			}
		})
	}
}

func TestMatchAgentProject(t *testing.T) {
	tests := []struct {
		name         string
		beadsProject string
		projectDir   string
		filters      []string
		expected     bool
	}{
		// Empty filter cases - no filtering
		{
			name:         "empty_filters_returns_true",
			beadsProject: "ok",
			projectDir:   "/Users/dylan/orch-knowledge",
			filters:      nil,
			expected:     true,
		},
		// Match on beads prefix
		{
			name:         "match_beads_prefix",
			beadsProject: "orch-go",
			projectDir:   "/Users/dylan/orch-go",
			filters:      []string{"orch-go"},
			expected:     true,
		},
		// Match on directory name
		{
			name:         "match_directory_name",
			beadsProject: "ok",
			projectDir:   "/Users/dylan/orch-knowledge",
			filters:      []string{"orch-knowledge"},
			expected:     true,
		},
		// BUG FIX TEST: Beads prefix differs from directory name
		// This is the specific scenario that was broken before the fix.
		// Agent from orch-knowledge has beads prefix "ok" (from beads ID like "ok-765").
		// Filter is "orch-knowledge" (directory name). Should match via projectDir.
		{
			name:         "beads_prefix_differs_from_dir_name",
			beadsProject: "ok",
			projectDir:   "/Users/dylan/orch-knowledge",
			filters:      []string{"orch-knowledge"},
			expected:     true,
		},
		// Match when either matches (beads prefix)
		{
			name:         "match_either_beads_prefix",
			beadsProject: "ok",
			projectDir:   "/Users/dylan/orch-knowledge",
			filters:      []string{"ok"},
			expected:     true,
		},
		// No match when neither matches
		{
			name:         "no_match_when_neither_matches",
			beadsProject: "ok",
			projectDir:   "/Users/dylan/orch-knowledge",
			filters:      []string{"orch-go"},
			expected:     false,
		},
		// Multi-filter matching
		{
			name:         "multi_filter_match_by_dir",
			beadsProject: "ok",
			projectDir:   "/Users/dylan/orch-knowledge",
			filters:      []string{"orch-go", "orch-knowledge", "beads"},
			expected:     true,
		},
		{
			name:         "multi_filter_match_by_beads",
			beadsProject: "ok",
			projectDir:   "/Users/dylan/orch-knowledge",
			filters:      []string{"orch-go", "ok", "beads"},
			expected:     true,
		},
		// Empty projectDir - should still match if beads prefix matches
		{
			name:         "empty_projectDir_match_beads",
			beadsProject: "orch-go",
			projectDir:   "",
			filters:      []string{"orch-go"},
			expected:     true,
		},
		// Empty both - should not match
		{
			name:         "empty_both_no_match",
			beadsProject: "",
			projectDir:   "",
			filters:      []string{"orch-go"},
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchAgentProject(tt.beadsProject, tt.projectDir, tt.filters)
			if got != tt.expected {
				t.Errorf("matchAgentProject(%q, %q, %v) = %v, want %v",
					tt.beadsProject, tt.projectDir, tt.filters, got, tt.expected)
			}
		})
	}
}

func TestFilterByTime(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name          string
		timestamp     time.Time
		sinceDuration time.Duration
		expected      bool
	}{
		{
			name:          "zero_duration_returns_true",
			timestamp:     now.Add(-24 * time.Hour),
			sinceDuration: 0,
			expected:      true,
		},
		{
			name:          "within_duration",
			timestamp:     now.Add(-1 * time.Hour),
			sinceDuration: 12 * time.Hour,
			expected:      true,
		},
		{
			name:          "outside_duration",
			timestamp:     now.Add(-24 * time.Hour),
			sinceDuration: 12 * time.Hour,
			expected:      false,
		},
		{
			name:          "just_inside_boundary",
			timestamp:     now.Add(-11*time.Hour - 59*time.Minute),
			sinceDuration: 12 * time.Hour,
			expected:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterByTime(tt.timestamp, tt.sinceDuration)
			if got != tt.expected {
				t.Errorf("filterByTime() = %v, want %v", got, tt.expected)
			}
		})
	}
}
