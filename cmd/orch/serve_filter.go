package main

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// parseSinceParam parses the ?since= query parameter into a time.Duration.
// Supported formats:
//   - 12h, 24h, 48h (hours)
//   - 7d (days)
//   - all (returns 0, meaning no filtering)
//
// Default: 12h if not specified or invalid.
func parseSinceParam(r *http.Request) time.Duration {
	return parseSinceParamWithDefault(r, 12*time.Hour)
}

// parseSinceParamWithDefault parses the ?since= query parameter with a custom default.
// A defaultDuration of 0 means no time filtering when the param is not specified.
func parseSinceParamWithDefault(r *http.Request, defaultDuration time.Duration) time.Duration {
	since := r.URL.Query().Get("since")
	if since == "" {
		return defaultDuration
	}
	if since == "all" {
		return 0
	}

	// Try parsing as hours (e.g., "12h", "24h")
	if strings.HasSuffix(since, "h") {
		hours, err := strconv.Atoi(strings.TrimSuffix(since, "h"))
		if err == nil && hours > 0 {
			return time.Duration(hours) * time.Hour
		}
	}

	// Try parsing as days (e.g., "7d")
	if strings.HasSuffix(since, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(since, "d"))
		if err == nil && days > 0 {
			return time.Duration(days) * 24 * time.Hour
		}
	}

	// Default fallback for invalid values
	return defaultDuration
}

// parseProjectFilter parses the ?project= query parameter.
// Supports comma-separated values for multi-project filtering (e.g., "?project=orch-go,orch-cli,beads").
// Returns empty slice if not specified (meaning no project filtering).
func parseProjectFilter(r *http.Request) []string {
	param := r.URL.Query().Get("project")
	if param == "" {
		return nil
	}
	// Split on comma and trim whitespace from each project name
	projects := strings.Split(param, ",")
	result := make([]string, 0, len(projects))
	for _, p := range projects {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// filterByTime returns true if the timestamp is within the since duration.
// If sinceDuration is 0, returns true (no filtering).
func filterByTime(timestamp time.Time, sinceDuration time.Duration) bool {
	if sinceDuration == 0 {
		return true // "all" mode - no time filtering
	}
	return time.Since(timestamp) <= sinceDuration
}

// filterByProject returns true if the projectDir matches ANY of the filters.
// If filters is empty, returns true (no filtering).
// Matches on:
//   - Full path match
//   - Project name (last path segment) match
func filterByProject(projectDir string, filters []string) bool {
	if len(filters) == 0 {
		return true // No filtering
	}
	if projectDir == "" {
		return false // No project_dir to match against
	}

	// Check if projectDir matches ANY of the filters
	for _, filter := range filters {
		// Full path match
		if projectDir == filter {
			return true
		}

		// Extract project name from path and match
		// e.g., "/Users/dylan/orch-go" -> "orch-go"
		projectName := extractProjectName(projectDir)
		filterName := extractProjectName(filter)
		if projectName == filterName {
			return true
		}
	}

	return false
}

// extractProjectName extracts the last path segment from a directory path.
func extractProjectName(dir string) string {
	if dir == "" {
		return ""
	}
	// Handle trailing slash
	dir = strings.TrimSuffix(dir, "/")
	// Get last segment
	if idx := strings.LastIndex(dir, "/"); idx != -1 {
		return dir[idx+1:]
	}
	return dir
}
