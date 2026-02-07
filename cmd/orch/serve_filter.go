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
	since := r.URL.Query().Get("since")
	if since == "" || since == "all" {
		// "all" means no time filtering - return 0
		if since == "all" {
			return 0
		}
		// Default to 12h if not specified
		return 12 * time.Hour
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

	// Default fallback
	return 12 * time.Hour
}

// parseProjectFilter parses the ?project= query parameter.
// Returns empty string if not specified (meaning no project filtering).
func parseProjectFilter(r *http.Request) string {
	return r.URL.Query().Get("project")
}

// filterByTime returns true if the timestamp is within the since duration.
// If sinceDuration is 0, returns true (no filtering).
func filterByTime(timestamp time.Time, sinceDuration time.Duration) bool {
	if sinceDuration == 0 {
		return true // "all" mode - no time filtering
	}
	return time.Since(timestamp) <= sinceDuration
}

// filterByProject returns true if the projectDir matches the filter.
// If filter is empty, returns true (no filtering).
// Matches on:
//   - Full path match
//   - Project name (last path segment) match
func filterByProject(projectDir, filter string) bool {
	if filter == "" {
		return true // No filtering
	}
	if projectDir == "" {
		return false // No project_dir to match against
	}

	// Full path match
	if projectDir == filter {
		return true
	}

	// Extract project name from path and match
	// e.g., "/Users/dylan/orch-go" -> "orch-go"
	projectName := extractProjectName(projectDir)
	filterName := extractProjectName(filter)
	return projectName == filterName
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
