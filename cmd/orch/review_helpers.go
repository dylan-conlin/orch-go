package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// extractBeadsIDFromWorkspace extracts the beads ID from workspace files.
// Checks sources in order of reliability:
// 1. .beads_id file (written directly by spawn code)
// 2. AGENT_MANIFEST.json (has beads_id field)
// 3. SPAWN_CONTEXT.md "beads issue:" pattern (legacy fallback)
func extractBeadsIDFromWorkspace(workspacePath string) string {
	// Source 1: .beads_id file (most reliable - written directly by spawn code)
	beadsIDPath := filepath.Join(workspacePath, ".beads_id")
	if data, err := os.ReadFile(beadsIDPath); err == nil {
		id := strings.TrimSpace(string(data))
		if id != "" {
			return id
		}
	}

	// Source 2: AGENT_MANIFEST.json
	manifestPath := filepath.Join(workspacePath, "AGENT_MANIFEST.json")
	if data, err := os.ReadFile(manifestPath); err == nil {
		// Simple extraction - avoid importing encoding/json just for one field
		// Look for "beads_id": "value" in the JSON
		content := string(data)
		if idx := strings.Index(content, `"beads_id"`); idx != -1 {
			// Find the value after the colon
			rest := content[idx+len(`"beads_id"`):]
			// Skip whitespace and colon
			rest = strings.TrimLeft(rest, " \t\n:")
			// Extract quoted value
			if len(rest) > 0 && rest[0] == '"' {
				end := strings.Index(rest[1:], `"`)
				if end > 0 {
					id := rest[1 : end+1]
					if id != "" {
						return id
					}
				}
			}
		}
	}

	// Source 3: SPAWN_CONTEXT.md (legacy fallback for older workspaces)
	spawnContextPath := filepath.Join(workspacePath, "SPAWN_CONTEXT.md")
	content, err := os.ReadFile(spawnContextPath)
	if err != nil {
		return ""
	}

	// Look for "beads issue: **xxx**" pattern or "orch-go-pe5d.2" format
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		lineLower := strings.ToLower(line)
		if strings.Contains(lineLower, "beads issue:") || strings.Contains(lineLower, "spawned from beads issue:") {
			// Extract beads ID from the line
			// Patterns: "beads issue: **orch-go-pe5d.2**" or "orch-go-pe5d.2"
			for _, part := range strings.Fields(line) {
				part = strings.Trim(part, "*`[]")
				// Look for pattern like "project-xxxx" or "project-xxxx.n"
				if strings.Count(part, "-") >= 1 && len(part) > 5 {
					// Skip common non-ID words
					if strings.HasPrefix(part, "beads") || strings.HasPrefix(part, "BEADS") ||
						strings.HasPrefix(part, "issue") || strings.HasPrefix(part, "ISSUE") ||
						strings.HasPrefix(part, "bd") || strings.HasPrefix(part, "comment") {
						continue
					}
					return part
				}
			}
		}
	}
	return ""
}

// extractProjectDirFromWorkspace extracts the PROJECT_DIR from SPAWN_CONTEXT.md
// This is used to determine which project's beads database to query for cross-project agents.
func extractProjectDirFromWorkspace(workspacePath string) string {
	spawnContextPath := filepath.Join(workspacePath, "SPAWN_CONTEXT.md")
	content, err := os.ReadFile(spawnContextPath)
	if err != nil {
		return ""
	}

	// Look for "PROJECT_DIR: /path/to/project" pattern
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "PROJECT_DIR:") {
			// Extract path after "PROJECT_DIR:"
			path := strings.TrimPrefix(line, "PROJECT_DIR:")
			path = strings.TrimSpace(path)
			return path
		}
	}
	return ""
}

// extractProject gets project name from project directory.
func extractProject(projectDir string) string {
	if projectDir == "" {
		return "unknown"
	}
	return filepath.Base(projectDir)
}

// isStaleAgent returns true if the agent is in a non-Complete phase and
// the workspace hasn't been modified in over 24 hours.
func isStaleAgent(phase string, modTime time.Time) bool {
	if phase == "Complete" {
		return false
	}
	return time.Since(modTime) > StaleThreshold
}

// groupByProject groups completions by project.
func groupByProject(completions []CompletionInfo) map[string][]CompletionInfo {
	grouped := make(map[string][]CompletionInfo)
	for _, c := range completions {
		grouped[c.Project] = append(grouped[c.Project], c)
	}
	return grouped
}

// filterClosedIssues removes completions whose beads issues are closed/deferred/tombstone.
// Uses ListOpenIssues for efficiency - a single call to get all open issues.
// If beads is unavailable, returns all candidates (better to show potential false positives than hide real issues).
func filterClosedIssues(candidates []CompletionInfo) []CompletionInfo {
	if len(candidates) == 0 {
		return candidates
	}

	// Use ListOpenIssues to get all open issues in a single call
	// This is much faster than individual Show() calls for each beads ID
	openIssueMap, err := verify.ListOpenIssues("")
	if err != nil {
		// If beads is unavailable, return all candidates
		return candidates
	}

	// Filter out closed issues (keep only those that exist in openIssueMap)
	var results []CompletionInfo
	for _, c := range candidates {
		// Keep agents without beads ID (no issue to check)
		if c.BeadsID == "" {
			results = append(results, c)
			continue
		}

		// Check if issue is open (exists in openIssueMap)
		if _, isOpen := openIssueMap[c.BeadsID]; isOpen {
			results = append(results, c)
		}
		// If not in openIssueMap, it's closed - skip it
	}

	return results
}

// findBdCommand locates the bd binary.
func findBdCommand() (string, error) {
	// Try common locations
	paths := []string{
		filepath.Join(os.Getenv("HOME"), "bin", "bd"),
		filepath.Join(os.Getenv("HOME"), "go", "bin", "bd"),
		filepath.Join(os.Getenv("HOME"), ".local", "bin", "bd"),
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	// Try PATH
	if path, err := exec.LookPath("bd"); err == nil {
		return path, nil
	}

	return "", fmt.Errorf("bd not found in common locations or PATH")
}
