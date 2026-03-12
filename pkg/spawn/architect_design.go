package spawn

import (
	"os"
	"path/filepath"
	"strings"
)

// FetchArchitectDesign retrieves the SYNTHESIS.md content from an architect's workspace
// identified by beads ID. Searches both archived and active workspaces.
// Returns the full SYNTHESIS.md content (minus the "# SYNTHESIS" heading) for injection
// into SPAWN_CONTEXT.md, or empty string if not found.
func FetchArchitectDesign(architectBeadsID, projectDir string) string {
	if architectBeadsID == "" || projectDir == "" {
		return ""
	}

	// Search archived workspaces first (architect issues are typically closed)
	archivedWorkspaces := FindArchivedWorkspacesByBeadsID(projectDir, architectBeadsID)
	for _, ws := range archivedWorkspaces {
		if content := readSynthesisContent(ws.Path); content != "" {
			return content
		}
	}

	// Fall back to active workspaces
	activeDir := filepath.Join(projectDir, ".orch", "workspace")
	entries, err := os.ReadDir(activeDir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "archived" {
			continue
		}
		wsPath := filepath.Join(activeDir, entry.Name())
		manifest := ReadAgentManifestWithFallback(wsPath)
		if manifest == nil || manifest.BeadsID != architectBeadsID {
			continue
		}
		if content := readSynthesisContent(wsPath); content != "" {
			return content
		}
	}

	return ""
}

// readSynthesisContent reads SYNTHESIS.md from a workspace and returns its content
// with the top-level "# SYNTHESIS" heading stripped (since it will be placed under
// a different heading in SPAWN_CONTEXT.md).
func readSynthesisContent(workspacePath string) string {
	synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
	data, err := os.ReadFile(synthesisPath)
	if err != nil {
		return ""
	}

	content := strings.TrimSpace(string(data))
	if content == "" {
		return ""
	}

	// Strip the top-level "# SYNTHESIS" heading if present
	lines := strings.Split(content, "\n")
	if len(lines) > 0 && strings.TrimSpace(strings.ToLower(lines[0])) == "# synthesis" {
		content = strings.TrimSpace(strings.Join(lines[1:], "\n"))
	}

	return content
}
