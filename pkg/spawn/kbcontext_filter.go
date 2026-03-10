package spawn

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/group"
)

// OrchEcosystemRepos defines the allowlist of repos that are relevant for orchestration work.
// Used as fallback when groups.yaml doesn't exist (~/.kb/ or ~/.orch/).
// When groups.yaml exists, project group membership replaces this hardcode.
var OrchEcosystemRepos = map[string]bool{
	"orch-go":   true,
	"orch-cli":  true,
	"kb-cli":    true,
	"beads":     true,
	"kn":        true,
}

// filterToOrchEcosystem filters matches to only include those from orch ecosystem repos.
// Matches without a project prefix (local results) are always included.
// Deprecated: Use filterToProjectGroup for group-aware filtering. Kept as fallback.
func filterToOrchEcosystem(matches []KBContextMatch) []KBContextMatch {
	var filtered []KBContextMatch
	for _, m := range matches {
		project := extractProjectFromMatch(m)
		// Include if: no project prefix (local), OR project is in ecosystem allowlist
		if project == "" || OrchEcosystemRepos[project] {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

// filterToProjectGroup filters matches using group-aware allowlist.
// Matches without a project prefix (local results) are always included.
func filterToProjectGroup(matches []KBContextMatch, allowlist map[string]bool) []KBContextMatch {
	var filtered []KBContextMatch
	for _, m := range matches {
		project := extractProjectFromMatch(m)
		if project == "" || allowlist[project] {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

// resolveProjectAllowlist builds an allowlist using the current working directory.
// For cross-project spawns, use resolveProjectAllowlistForDir instead.
func resolveProjectAllowlist() map[string]bool {
	return resolveProjectAllowlistForDir("")
}

// resolveProjectAllowlistForDir builds an allowlist of project names for global search filtering.
// projectDir controls which project's groups are used. When empty, falls back to os.Getwd().
// Tries group-based resolution from groups.yaml first (~/.kb/ primary, ~/.orch/ fallback).
// Falls back to OrchEcosystemRepos if groups.yaml doesn't exist or the current project is ungrouped.
func resolveProjectAllowlistForDir(projectDir string) map[string]bool {
	cfg, err := group.Load()
	if err != nil {
		// groups.yaml doesn't exist or can't be parsed — use hardcoded fallback
		return OrchEcosystemRepos
	}

	// Detect project name from directory
	projectName := detectProjectNameFromDir(projectDir)
	if projectName == "" {
		return OrchEcosystemRepos
	}

	// Get kb projects list for parent inference
	kbProjects := loadKBProjectsMap()
	if kbProjects == nil {
		return OrchEcosystemRepos
	}

	// Resolve groups for current project
	groups := cfg.GroupsForProject(projectName, kbProjects)
	if len(groups) == 0 {
		// Project is ungrouped — no group-based filtering
		// Return nil to signal "don't filter" (include all global matches)
		return nil
	}

	// Build allowlist from all projects in matching groups
	allowlist := make(map[string]bool)
	for _, g := range groups {
		members := cfg.ResolveGroupMembers(g.Name, kbProjects)
		for _, m := range members {
			allowlist[m] = true
		}
	}

	return allowlist
}

// detectCurrentProjectName returns the project name from the current working directory.
// Deprecated: Use detectProjectNameFromDir for explicit directory control.
func detectCurrentProjectName() string {
	return detectProjectNameFromDir("")
}

// detectProjectNameFromDir returns the project name from the given directory.
// Uses .beads/config.yaml issue-prefix if available, otherwise falls back to directory basename.
// When dir is empty, falls back to os.Getwd().
func detectProjectNameFromDir(dir string) string {
	if dir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return ""
		}
		dir = cwd
	}

	// Try .beads/config.yaml for issue prefix (more reliable)
	configPath := filepath.Join(dir, ".beads", "config.yaml")
	if data, err := os.ReadFile(configPath); err == nil {
		// Simple YAML parsing — look for issue-prefix field
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "issue-prefix:") {
				prefix := strings.TrimSpace(strings.TrimPrefix(line, "issue-prefix:"))
				prefix = strings.Trim(prefix, `"'`)
				if prefix != "" {
					return prefix
				}
			}
		}
	}

	// Fall back to directory basename
	return filepath.Base(dir)
}

// kbProjectEntry matches the JSON format from `kb projects list --json`.
type kbProjectEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// loadKBProjectsMap runs `kb projects list --json` and returns name->path map.
func loadKBProjectsMap() map[string]string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "kb", "projects", "list", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var projects []kbProjectEntry
	if err := json.Unmarshal(output, &projects); err != nil {
		return nil
	}

	result := make(map[string]string, len(projects))
	for _, p := range projects {
		result[p.Name] = p.Path
	}
	return result
}

// extractProjectFromMatch extracts the project name from a match's title or path.
// Returns empty string if no project prefix found.
func extractProjectFromMatch(m KBContextMatch) string {
	// Check for [project] prefix in title (e.g., "[orch-go] Title here")
	if strings.HasPrefix(m.Title, "[") {
		end := strings.Index(m.Title, "]")
		if end > 1 {
			return m.Title[1:end]
		}
	}
	return ""
}

// applyPerCategoryLimits limits the number of matches per category type.
func applyPerCategoryLimits(matches []KBContextMatch, limit int) []KBContextMatch {
	categoryCounts := make(map[string]int)
	var filtered []KBContextMatch

	for _, m := range matches {
		if categoryCounts[m.Type] < limit {
			filtered = append(filtered, m)
			categoryCounts[m.Type]++
		}
	}
	return filtered
}

// mergeResults combines two KBContextResults, deduplicating matches.
func mergeResults(local, global *KBContextResult) *KBContextResult {
	if local == nil {
		return global
	}
	if global == nil {
		return local
	}

	// Create a set of existing titles to avoid duplicates
	seen := make(map[string]bool)
	var merged []KBContextMatch

	// Add local matches first (higher priority)
	for _, m := range local.Matches {
		key := m.Type + ":" + m.Title
		if !seen[key] {
			seen[key] = true
			merged = append(merged, m)
		}
	}

	// Add global matches that aren't duplicates
	for _, m := range global.Matches {
		key := m.Type + ":" + m.Title
		if !seen[key] {
			seen[key] = true
			merged = append(merged, m)
		}
	}

	return &KBContextResult{
		Query:      local.Query,
		HasMatches: len(merged) > 0,
		Matches:    merged,
		RawOutput:  formatMatchesForDisplay(merged, local.Query),
	}
}

// normalizeGlobalKBPaths resolves ~/.kb/ symlink paths to project-relative .kb/global/ paths.
// kb context returns paths like /Users/user/.kb/models/foo/model.md which is a symlink to
// {projectDir}/.kb/global/models/foo/model.md. Normalizing makes paths agent-friendly and
// prevents DetectCrossRepoModel from misclassifying global models as cross-repo.
func normalizeGlobalKBPaths(matches []KBContextMatch, projectDir string) []KBContextMatch {
	if projectDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return matches
		}
		projectDir = cwd
	}

	// Resolve ~/.kb to its real path (follows symlinks)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return matches
	}
	homeKB := filepath.Join(homeDir, ".kb")
	resolvedHomeKB, err := filepath.EvalSymlinks(homeKB)
	if err != nil {
		return matches // ~/.kb doesn't exist or can't resolve
	}

	// Check if ~/.kb resolves to {projectDir}/.kb/global/
	globalKB := filepath.Join(projectDir, ".kb", "global")
	// Use EvalSymlinks on globalKB too to get canonical path (handles macOS /var → /private/var)
	resolvedGlobalKB, err := filepath.EvalSymlinks(globalKB)
	if err != nil {
		// .kb/global/ might not exist — try Abs as fallback
		resolvedGlobalKB, err = filepath.Abs(globalKB)
		if err != nil {
			return matches
		}
	}

	if resolvedHomeKB != resolvedGlobalKB {
		return matches // ~/.kb doesn't point to this project's .kb/global/
	}

	// Replace ~/.kb/ prefix with .kb/global/ (using resolved project path)
	homeKBPrefix := homeKB + string(filepath.Separator)
	globalKBPrefix := resolvedGlobalKB + string(filepath.Separator)

	for i := range matches {
		if strings.HasPrefix(matches[i].Path, homeKBPrefix) {
			matches[i].Path = globalKBPrefix + matches[i].Path[len(homeKBPrefix):]
		}
	}

	return matches
}
