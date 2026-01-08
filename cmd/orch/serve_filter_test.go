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
		expected string
	}{
		{
			name:     "no_filter",
			query:    "",
			expected: "",
		},
		{
			name:     "project_name",
			query:    "?project=orch-go",
			expected: "orch-go",
		},
		{
			name:     "full_path",
			query:    "?project=/Users/dylan/orch-go",
			expected: "/Users/dylan/orch-go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/agents"+tt.query, nil)
			got := parseProjectFilter(req)
			if got != tt.expected {
				t.Errorf("parseProjectFilter() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFilterByProjectDir(t *testing.T) {
	tests := []struct {
		name       string
		projectDir string
		filter     string
		expected   bool
	}{
		// Empty filter cases - no filtering
		{
			name:       "empty_filter_returns_true",
			projectDir: "/Users/dylan/orch-go",
			filter:     "",
			expected:   true,
		},
		// Empty projectDir cases - should not match
		{
			name:       "empty_projectDir_returns_false",
			projectDir: "",
			filter:     "orch-go",
			expected:   false,
		},
		// Full path matching
		{
			name:       "full_path_match",
			projectDir: "/Users/dylan/orch-go",
			filter:     "/Users/dylan/orch-go",
			expected:   true,
		},
		{
			name:       "full_path_no_match",
			projectDir: "/Users/dylan/orch-go",
			filter:     "/Users/dylan/kb-cli",
			expected:   false,
		},
		// Project name matching
		{
			name:       "project_name_match",
			projectDir: "/Users/dylan/orch-go",
			filter:     "orch-go",
			expected:   true,
		},
		{
			name:       "project_name_no_match",
			projectDir: "/Users/dylan/orch-go",
			filter:     "kb-cli",
			expected:   false,
		},
		// Cross-project scenario: filter by project name when projectDir is from workspace cache
		// This is the key scenario for --workdir spawns
		{
			name:       "cross_project_workdir_match",
			projectDir: "/Users/dylan/kb-cli", // Actual target project from workspace cache
			filter:     "kb-cli",              // Filter by project name
			expected:   true,
		},
		{
			name:       "cross_project_workdir_filter_different_project",
			projectDir: "/Users/dylan/kb-cli", // Actual target project from workspace cache
			filter:     "orch-go",             // Filter for different project
			expected:   false,
		},
		// Trailing slash handling - extractProjectName handles this
		{
			name:       "trailing_slash_full_path",
			projectDir: "/Users/dylan/orch-go/",
			filter:     "/Users/dylan/orch-go",
			expected:   true, // extractProjectName handles trailing slash, both resolve to "orch-go"
		},
		{
			name:       "trailing_slash_project_name",
			projectDir: "/Users/dylan/orch-go/",
			filter:     "orch-go",
			expected:   true, // extractProjectName handles trailing slash
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterByProject(tt.projectDir, tt.filter)
			if got != tt.expected {
				t.Errorf("filterByProject(%q, %q) = %v, want %v", tt.projectDir, tt.filter, got, tt.expected)
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
