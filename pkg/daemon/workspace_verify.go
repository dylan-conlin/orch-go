package daemon

import (
	"os"
	"path/filepath"
	"strings"
)

// workspaceExistsForIssue checks if a workspace directory exists for the given
// beads ID in the project's .orch/workspace/ directory. It scans workspace
// directories and checks SPAWN_CONTEXT.md files for the beads ID reference,
// since workspace names use random suffixes rather than the beads ID.
func workspaceExistsForIssue(beadsID, projectDir string) bool {
	if beadsID == "" || projectDir == "" {
		return false
	}

	workspaceBase := filepath.Join(projectDir, ".orch", "workspace")
	entries, err := os.ReadDir(workspaceBase)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "archived" {
			continue
		}

		// Check SPAWN_CONTEXT.md for beads ID reference.
		// This is authoritative — workspace names don't contain the beads ID.
		spawnCtxPath := filepath.Join(workspaceBase, entry.Name(), "SPAWN_CONTEXT.md")
		content, err := os.ReadFile(spawnCtxPath)
		if err != nil {
			continue
		}
		if strings.Contains(string(content), beadsID) {
			return true
		}
	}

	return false
}
