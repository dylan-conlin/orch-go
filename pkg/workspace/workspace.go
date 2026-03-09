// Package workspace provides shared workspace scanning and manifest operations.
// Workspaces live in .orch/workspace/ and contain agent session artifacts
// (SPAWN_CONTEXT.md, AGENT_MANIFEST.json, .session_id, .spawn_time, etc.).
package workspace

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// FindByBeadsID searches for a workspace directory spawned from the beads ID.
// Looks in .orch/workspace/ for directories that match the beads ID in their name
// or contain a SPAWN_CONTEXT.md with "spawned from beads issue: **beadsID**".
// When multiple workspaces match (duplicate spawns), prefers the one with SYNTHESIS.md,
// then the most recently spawned (by .spawn_time file).
// Returns the workspace path and agent name (directory name) if found.
func FindByBeadsID(projectDir, beadsID string) (workspacePath, agentName string) {
	workspaceDir := filepath.Join(projectDir, ".orch", "workspace")
	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		return "", ""
	}

	type candidate struct {
		path string
		name string
	}
	var candidates []candidate

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip archived directory - only scan active workspaces
		if entry.Name() == "archived" {
			continue
		}

		dirName := entry.Name()
		dirPath := filepath.Join(workspaceDir, dirName)

		matched := false

		// Check if the beads ID is in the directory name
		if strings.Contains(dirName, beadsID) {
			matched = true
		}

		// Check AGENT_MANIFEST.json for beads_id (primary, falls back to .beads_id dotfile)
		if !matched {
			manifest := spawn.ReadAgentManifestWithFallback(dirPath)
			if manifest.BeadsID == beadsID {
				matched = true
			}
		}

		// Check SPAWN_CONTEXT.md for authoritative "spawned from beads issue" line
		if !matched {
			spawnContextPath := filepath.Join(dirPath, "SPAWN_CONTEXT.md")
			if content, err := os.ReadFile(spawnContextPath); err == nil {
				contentStr := string(content)
				for _, line := range strings.Split(contentStr, "\n") {
					lineLower := strings.ToLower(line)
					if strings.Contains(lineLower, "spawned from beads issue:") {
						if strings.Contains(line, beadsID) {
							matched = true
						}
						break
					}
				}
			}
		}

		if matched {
			candidates = append(candidates, candidate{path: dirPath, name: dirName})
		}
	}

	if len(candidates) == 0 {
		return "", ""
	}
	if len(candidates) == 1 {
		return candidates[0].path, candidates[0].name
	}

	// Multiple candidates: prefer workspace with SYNTHESIS.md, then most recent spawn time
	bestIdx := 0
	bestHasSynthesis := false
	bestSpawnTime := SpawnTime(candidates[0].path)
	if _, err := os.Stat(filepath.Join(candidates[0].path, "SYNTHESIS.md")); err == nil {
		bestHasSynthesis = true
	}

	for i := 1; i < len(candidates); i++ {
		c := candidates[i]
		hasSynthesis := false
		if _, err := os.Stat(filepath.Join(c.path, "SYNTHESIS.md")); err == nil {
			hasSynthesis = true
		}

		// Prefer SYNTHESIS.md
		if hasSynthesis && !bestHasSynthesis {
			bestIdx = i
			bestHasSynthesis = hasSynthesis
			bestSpawnTime = SpawnTime(c.path)
			continue
		}
		if !hasSynthesis && bestHasSynthesis {
			continue
		}

		// Tiebreak: most recent spawn time
		spawnTime := SpawnTime(c.path)
		if spawnTime > bestSpawnTime {
			bestIdx = i
			bestHasSynthesis = hasSynthesis
			bestSpawnTime = spawnTime
		}
	}

	return candidates[bestIdx].path, candidates[bestIdx].name
}

// SpawnTime reads the .spawn_time file from a workspace directory.
// Returns the Unix nanosecond timestamp, or 0 if not found.
func SpawnTime(wsPath string) int64 {
	data, err := os.ReadFile(filepath.Join(wsPath, ".spawn_time"))
	if err != nil {
		return 0
	}
	t, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0
	}
	return t
}

// FindByName searches for a workspace directory by its name.
// Returns the workspace path if found, or empty string if not found.
func FindByName(projectDir, workspaceName string) string {
	workspaceDir := filepath.Join(projectDir, ".orch", "workspace")
	dirPath := filepath.Join(workspaceDir, workspaceName)

	if stat, err := os.Stat(dirPath); err == nil && stat.IsDir() {
		return dirPath
	}

	return ""
}

// IsOrchestrator checks if a workspace is for an orchestrator session.
// Returns true if .orchestrator or .meta-orchestrator marker file exists.
func IsOrchestrator(workspacePath string) bool {
	if _, err := os.Stat(filepath.Join(workspacePath, ".orchestrator")); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(workspacePath, ".meta-orchestrator")); err == nil {
		return true
	}
	return false
}

// HasSessionHandoff checks if SESSION_HANDOFF.md exists in the workspace.
func HasSessionHandoff(workspacePath string) bool {
	_, err := os.Stat(filepath.Join(workspacePath, "SESSION_HANDOFF.md"))
	return err == nil
}
