// Package spawn provides spawn configuration and context generation.
package spawn

import (
	"os"
	"path/filepath"
	"strings"
)

// EcosystemFilePath is the path to the ecosystem registry file.
const EcosystemFilePath = "~/.orch/ECOSYSTEM.md"

// ExpandedOrchEcosystemRepos is the full list of repos in Dylan's orchestration ecosystem.
// This extends OrchEcosystemRepos (in kbcontext.go) with additional projects that were
// not originally included but are part of the ecosystem.
var ExpandedOrchEcosystemRepos = map[string]bool{
	// Core orchestration repos (from OrchEcosystemRepos)
	"orch-go":        true,
	"orch-cli":       true,
	"kb-cli":         true,
	"orch-knowledge": true,
	"beads":          true,
	"kn":             true,
	// Additional ecosystem repos
	"beads-ui-svelte": true,
	"skillc":          true,
	"agentlog":        true,
}

// IsEcosystemRepo checks if a project is part of Dylan's orchestration ecosystem.
// Uses the expanded list that includes all known ecosystem projects.
func IsEcosystemRepo(projectName string) bool {
	return ExpandedOrchEcosystemRepos[projectName]
}

// GenerateEcosystemContext reads the ecosystem registry from ~/.orch/ECOSYSTEM.md
// and extracts the Quick Reference table for inclusion in spawn context.
// This provides spawned agents with knowledge of Dylan's local project ecosystem
// so they don't try to search GitHub for projects like beads, kb-cli, etc.
// Returns empty string if file doesn't exist or can't be read.
func GenerateEcosystemContext() string {
	// Expand ~ to home directory
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	ecosystemPath := filepath.Join(home, ".orch", "ECOSYSTEM.md")

	// Read the file
	content, err := os.ReadFile(ecosystemPath)
	if err != nil {
		return "" // File doesn't exist or can't be read - skip silently
	}

	// Extract the Quick Reference section (table with repo info)
	// This is the most useful part for agents - concise project listing
	return ExtractQuickReference(string(content))
}

// ExtractQuickReference extracts the Quick Reference table from ECOSYSTEM.md.
// Returns the table content or the first ~50 lines if table not found.
// This is exported for use in tests and other packages.
func ExtractQuickReference(content string) string {
	lines := strings.Split(content, "\n")

	// Find "## Quick Reference" section
	var inQuickRef bool
	var result []string

	for _, line := range lines {
		// Start capturing at Quick Reference heading
		if strings.HasPrefix(line, "## Quick Reference") {
			inQuickRef = true
			result = append(result, line)
			continue
		}

		// Stop at next section (any ## heading)
		if inQuickRef && strings.HasPrefix(line, "## ") {
			break
		}

		// Capture content
		if inQuickRef {
			result = append(result, line)
		}
	}

	if len(result) > 0 {
		return strings.TrimSpace(strings.Join(result, "\n"))
	}

	// Fallback: return first 50 lines (minus frontmatter)
	var fallback []string
	skipFrontmatter := true
	for i, line := range lines {
		if skipFrontmatter && strings.HasPrefix(line, "---") {
			continue
		}
		if skipFrontmatter && strings.HasPrefix(line, ">") {
			continue
		}
		skipFrontmatter = false

		fallback = append(fallback, line)
		if len(fallback) >= 50 || i >= 60 {
			break
		}
	}

	return strings.TrimSpace(strings.Join(fallback, "\n"))
}
