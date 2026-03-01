package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/verify"
)

var listOpenIssues = verify.ListOpenIssues
var listOpenIssuesWithDir = verify.ListOpenIssuesWithDir

// investigationDirCache holds pre-loaded directory listings for investigation discovery.
// This prevents O(n^2) behavior when discovering investigation paths for many agents.
// Without this cache, each agent would call os.ReadDir() 2-3 times on directories
// with 500+ files, resulting in 300+ agents x 500+ files x 2 calls = massive slowdown.
type investigationDirCache struct {
	// entries maps directory path -> list of .md file names (not full DirEntry, just names for efficiency)
	entries map[string][]string
}

// buildInvestigationDirCache pre-loads directory listings for investigation discovery.
// Call this once before processing agents, then pass to discoverInvestigationPath.
func buildInvestigationDirCache(projectDirs []string) *investigationDirCache {
	cache := &investigationDirCache{
		entries: make(map[string][]string),
	}

	for _, projectDir := range projectDirs {
		if projectDir == "" {
			continue
		}

		// Cache .kb/investigations/
		investigationsDir := filepath.Join(projectDir, ".kb", "investigations")
		if entries, err := os.ReadDir(investigationsDir); err == nil {
			var mdFiles []string
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
					mdFiles = append(mdFiles, entry.Name())
				}
			}
			cache.entries[investigationsDir] = mdFiles
		}

		// Cache .kb/investigations/simple/
		simpleDir := filepath.Join(investigationsDir, "simple")
		if entries, err := os.ReadDir(simpleDir); err == nil {
			var mdFiles []string
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
					mdFiles = append(mdFiles, entry.Name())
				}
			}
			cache.entries[simpleDir] = mdFiles
		}
	}

	return cache
}

// getEntries returns cached directory entries, or empty slice if not cached.
func (c *investigationDirCache) getEntries(dirPath string) []string {
	if c == nil || c.entries == nil {
		return nil
	}
	return c.entries[dirPath]
}

// discoverInvestigationPath attempts to find an investigation file for an agent
// using a fallback chain when the agent hasn't reported an investigation_path via beads comment.
//
// IMPORTANT: Pass a pre-built investigationDirCache to avoid O(n^2) directory scanning.
// Without the cache, this function would call os.ReadDir() for each agent, causing
// massive slowdowns with 300+ agents and 500+ investigation files.
//
// Fallback chain:
// 1. Search .kb/investigations/ for files matching workspace name pattern
// 2. Search .kb/investigations/ for files matching beads ID
// 3. Check workspace directory for investigation .md files (excluding SPAWN_CONTEXT.md and SYNTHESIS.md)
func discoverInvestigationPath(workspaceName, beadsID, projectDir string, cache *investigationDirCache) string {
	if projectDir == "" {
		return ""
	}

	// Extract keywords from workspace name for matching (e.g., "og-inv-skillc-deploy-06jan-ed96" -> "skillc-deploy")
	// Workspace names follow pattern: {project}-{skill}-{topic}-{date}-{hash}
	workspaceKeywords := extractWorkspaceKeywords(workspaceName)

	investigationsDir := filepath.Join(projectDir, ".kb", "investigations")

	// Use cached entries if available (O(1) lookup vs O(n) ReadDir)
	entries := cache.getEntries(investigationsDir)
	if entries == nil {
		// Fallback to direct read if not cached (shouldn't happen in normal use)
		if dirEntries, err := os.ReadDir(investigationsDir); err == nil {
			for _, entry := range dirEntries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
					entries = append(entries, entry.Name())
				}
			}
		}
	}

	// 1. Search for files matching beads ID (e.g., "orch-go-51jz" in filename)
	// This is the most specific match and should be checked first.
	if beadsID != "" {
		// Extract short ID from beads ID (last segment after -)
		shortID := beadsID
		if idx := strings.LastIndex(beadsID, "-"); idx != -1 && idx < len(beadsID)-1 {
			shortID = beadsID[idx+1:]
		}

		for _, name := range entries {
			// Check if filename contains beads ID or short ID
			if strings.Contains(name, beadsID) || strings.Contains(name, shortID) {
				return filepath.Join(investigationsDir, name)
			}
		}
	}

	// 2. Search .kb/investigations/ for files matching workspace name pattern
	// Workspace names are specific to the agent's task.
	// We reverse the entries list to find the most recent files first (since they are date-prefixed).
	reversedEntries := make([]string, len(entries))
	for i, name := range entries {
		reversedEntries[len(entries)-1-i] = name
	}

	// First pass: look for exact topic match (highest confidence)
	// We now require at least one keyword match, but we prefer files that match MORE keywords.
	var bestMatch string
	maxMatches := 0

	for _, name := range reversedEntries {
		matches := 0
		for _, keyword := range workspaceKeywords {
			if keyword != "" && strings.Contains(strings.ToLower(name), strings.ToLower(keyword)) {
				matches++
			}
		}

		if matches > maxMatches {
			maxMatches = matches
			bestMatch = filepath.Join(investigationsDir, name)
			// If we match all keywords, return immediately (highest confidence)
			if matches == len(workspaceKeywords) && len(workspaceKeywords) > 0 {
				return bestMatch
			}
		}
	}

	if bestMatch != "" {
		return bestMatch
	}

	// 3. Search for simpler investigations or workspace-specific ones
	if beadsID != "" {
		// Also check .kb/investigations/simple/ (for simpler investigations)
		simpleDir := filepath.Join(investigationsDir, "simple")
		simpleEntries := cache.getEntries(simpleDir)
		if simpleEntries == nil {
			// Fallback to direct read if not cached
			if dirEntries, err := os.ReadDir(simpleDir); err == nil {
				for _, entry := range dirEntries {
					if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
						simpleEntries = append(simpleEntries, entry.Name())
					}
				}
			}
		}

		for _, name := range simpleEntries {
			for _, keyword := range workspaceKeywords {
				if keyword != "" && strings.Contains(strings.ToLower(name), strings.ToLower(keyword)) {
					return filepath.Join(simpleDir, name)
				}
			}
		}
	}

	// 4. Check workspace directory for investigation .md files
	// This is per-workspace so not cached (each workspace is different)
	workspaceDir := filepath.Join(projectDir, ".orch", "workspace", workspaceName)
	if wsEntries, err := os.ReadDir(workspaceDir); err == nil {
		for _, entry := range wsEntries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			// Skip standard workspace files
			if name == "SPAWN_CONTEXT.md" || name == "SYNTHESIS.md" || name == "ORCHESTRATOR_CONTEXT.md" ||
				name == "SESSION_HANDOFF.md" || name == "AGENT_MANIFEST.json" || name == "VERIFICATION_SPEC.yaml" ||
				name == ".session_id" || name == ".spawn_time" ||
				name == ".tier" || name == ".beads_id" || name == ".spawn_mode" {
				continue
			}
			// Check for .md files that might be investigation files
			if strings.HasSuffix(name, ".md") && strings.Contains(strings.ToLower(name), "inv") {
				return filepath.Join(workspaceDir, name)
			}
		}
	}

	return ""
}

// extractWorkspaceKeywords extracts meaningful keywords from a workspace name for investigation matching.
// Workspace names follow pattern: {project}-{skill}-{topic}-{date}-{hash}
// Example: "og-inv-skillc-deploy-06jan-ed96" -> ["skillc", "deploy"]
func extractWorkspaceKeywords(workspaceName string) []string {
	parts := strings.Split(workspaceName, "-")
	if len(parts) < 3 {
		return nil
	}

	var keywords []string

	// Skip prefix parts that are likely project or skill markers
	skipPrefixes := []string{"og", "inv", "feat", "fix", "debug", "audit", "impl", "arch", "research"}
	prefixSet := make(map[string]bool)
	for _, p := range skipPrefixes {
		prefixSet[p] = true
	}

	for _, part := range parts {
		// Skip short parts (likely hash or date)
		if len(part) <= 2 {
			continue
		}
		// Skip parts that look like dates (e.g., "06jan", "2026")
		if len(part) == 5 && strings.Contains(part, "jan") || strings.Contains(part, "feb") ||
			strings.Contains(part, "mar") || strings.Contains(part, "apr") ||
			strings.Contains(part, "may") || strings.Contains(part, "jun") ||
			strings.Contains(part, "jul") || strings.Contains(part, "aug") ||
			strings.Contains(part, "sep") || strings.Contains(part, "oct") ||
			strings.Contains(part, "nov") || strings.Contains(part, "dec") {
			continue
		}
		// Skip common prefixes
		if prefixSet[strings.ToLower(part)] {
			continue
		}
		// Skip parts that look like short hashes (4 hex chars at end)
		if len(part) == 4 && isHexLike(part) {
			continue
		}
		keywords = append(keywords, part)
	}

	return keywords
}

// isHexLike returns true if the string looks like a short hex hash (all lowercase letters/digits).
func isHexLike(s string) bool {
	for _, c := range s {
		if !((c >= 'a' && c <= 'f') || (c >= '0' && c <= '9')) {
			return false
		}
	}
	return true
}

// uniqueProjectDirs deduplicates project directories while preserving order.
func uniqueProjectDirs(dirs []string) []string {
	seen := make(map[string]bool, len(dirs))
	result := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		cleaned := filepath.Clean(dir)
		if seen[cleaned] {
			continue
		}
		seen[cleaned] = true
		result = append(result, cleaned)
	}
	return result
}

// listActiveIssues retrieves active issues (open, in_progress, or blocked) across project directories.
// Returns a map of beadsID -> Issue, plus beadsID -> projectDir for cross-project lookups.
//
// This includes "open", "in_progress", and "blocked" statuses because:
// - Auto-created issues (spawn without --issue) may not get updated to in_progress
// - The UpdateIssueStatus call can fail silently during spawn
// - Filtering only in_progress causes newly spawned agents to be invisible in the dashboard
//   while orch status (which uses workspace/session discovery) can see them
// - Blocked issues represent agents waiting on dependencies and must remain visible
//   in the dashboard, otherwise they silently vanish
func listActiveIssues(projectDirs []string) (map[string]*verify.Issue, map[string]string) {
	issues := make(map[string]*verify.Issue)
	projectDirsByID := make(map[string]string)

	if len(projectDirs) == 0 {
		openIssues, err := listOpenIssues()
		if err != nil {
			log.Printf("Warning: failed to list open issues: %v", err)
			return issues, projectDirsByID
		}

		for id, issue := range openIssues {
			status := strings.ToLower(issue.Status)
			if status == "open" || status == "in_progress" || status == "blocked" {
				issues[id] = issue
			}
		}

		return issues, projectDirsByID
	}

	for _, projectDir := range projectDirs {
		openIssues, err := listOpenIssuesWithDir(projectDir)
		if err != nil {
			log.Printf("Warning: failed to list open issues for %s: %v", projectDir, err)
			continue
		}

		for id, issue := range openIssues {
			status := strings.ToLower(issue.Status)
			if status == "open" || status == "in_progress" || status == "blocked" {
				if _, exists := issues[id]; !exists {
					issues[id] = issue
					projectDirsByID[id] = projectDir
				}
			}
		}
	}

	return issues, projectDirsByID
}
