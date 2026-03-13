package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
)

const cleanupMaxIdleTime = 30 * time.Minute

func defaultCleanup(config Config) (int, string, error) {
	closed, err := cleanStaleTmuxWindows(config.CleanupServerURL, config.CleanupPreserveOrchestrator)
	if err != nil {
		return 0, fmt.Sprintf("Cleanup failed: %v", err), err
	}

	// Also expire old archived workspaces
	expired := 0
	if config.CleanupArchivedTTLDays > 0 {
		projectDir, _ := os.Getwd()
		if projectDir != "" {
			expired, _ = expireArchivedWorkspaces(projectDir, config.CleanupArchivedTTLDays)
		}
	}

	total := closed + expired
	if total == 0 {
		return 0, "No stale resources found", nil
	}

	var parts []string
	if closed > 0 {
		parts = append(parts, fmt.Sprintf("closed %d stale tmux windows", closed))
	}
	if expired > 0 {
		parts = append(parts, fmt.Sprintf("deleted %d expired archived workspaces", expired))
	}
	return total, fmt.Sprintf("Cleanup: %s", strings.Join(parts, ", ")), nil
}

// expireArchivedWorkspaces deletes archived workspaces older than ttlDays.
// Uses spawn time from AGENT_MANIFEST.json or .spawn_time, falling back to dir modtime.
func expireArchivedWorkspaces(projectDir string, ttlDays int) (int, error) {
	archivedDir := filepath.Join(projectDir, ".orch", "workspace", "archived")

	if _, err := os.Stat(archivedDir); os.IsNotExist(err) {
		return 0, nil
	}

	cutoff := time.Now().AddDate(0, 0, -ttlDays)

	entries, err := os.ReadDir(archivedDir)
	if err != nil {
		return 0, fmt.Errorf("failed to read archived directory: %w", err)
	}

	deleted := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirPath := filepath.Join(archivedDir, entry.Name())

		// Determine workspace age
		var wsTime time.Time
		manifest := spawn.ReadAgentManifestWithFallback(dirPath)
		wsTime = manifest.ParseSpawnTime()

		if wsTime.IsZero() {
			wsTime = fallbackArchivedSpawnTime(dirPath)
		}

		if wsTime.IsZero() {
			continue
		}

		if wsTime.After(cutoff) {
			continue
		}

		if err := os.RemoveAll(dirPath); err != nil {
			continue
		}
		deleted++
	}

	return deleted, nil
}

// fallbackArchivedSpawnTime tries .spawn_time file mtime, SPAWN_CONTEXT.md mtime,
// AGENT_MANIFEST.json mtime, then directory mtime.
func fallbackArchivedSpawnTime(dirPath string) time.Time {
	candidates := []string{".spawn_time", "SPAWN_CONTEXT.md", spawn.AgentManifestFilename}
	for _, name := range candidates {
		if info, err := os.Stat(filepath.Join(dirPath, name)); err == nil {
			return info.ModTime()
		}
	}
	if info, err := os.Stat(dirPath); err == nil {
		return info.ModTime()
	}
	return time.Time{}
}

// beadsStatusFunc is a function that returns the beads issue status for a given ID.
// Extracted for testability.
type beadsStatusFunc func(beadsID string) (string, error)

// isWindowStale determines whether a tmux window should be cleaned up.
// A window is stale if:
//  1. Its beads ID is not active in any OpenCode session, AND
//  2. Its beads issue status is NOT in_progress or open (i.e., it's closed/completed)
//
// If beads status cannot be determined, the window is kept alive (fail-safe).
// This protects Claude CLI workers which run in tmux without OpenCode sessions.
func isWindowStale(beadsID string, activeBeadsIDs map[string]bool, getStatus beadsStatusFunc) bool {
	// Skip if active in OpenCode (headless/tmux backend)
	if activeBeadsIDs[beadsID] {
		return false
	}

	// Check beads issue status (protects Claude CLI workers).
	// Claude CLI workers run in tmux without OpenCode sessions, so they
	// would always appear "stale" if we only checked OpenCode. Querying
	// beads status ensures we don't kill workers that are still in_progress.
	// Fail-safe: if we can't determine status, keep the window alive.
	status, err := getStatus(beadsID)
	if err != nil {
		// Can't determine status — keep window alive (fail-safe)
		return false
	}
	if status == "in_progress" || status == "open" {
		// Issue is still active — agent is working
		return false
	}

	return true
}

// isWindowStaleBatch determines whether a tmux window should be cleaned up
// using pre-fetched sets of active IDs. This avoids per-window beads queries.
// A window is stale if its beads ID is not in activeOpenCodeIDs AND not in openBeadsIDs.
// If openBeadsIDs is nil (batch fetch failed), falls back to fail-safe (not stale).
func isWindowStaleBatch(beadsID string, activeOpenCodeIDs map[string]bool, openBeadsIDs map[string]bool) bool {
	if activeOpenCodeIDs[beadsID] {
		return false
	}
	if openBeadsIDs == nil {
		// Batch fetch failed — fail-safe: keep window alive
		return false
	}
	if openBeadsIDs[beadsID] {
		return false
	}
	return true
}

// listOpenBeadsIDs batch-fetches all open/in_progress beads issue IDs.
// Returns a set of IDs that should NOT be cleaned up.
// Uses beads RPC if available, falls back to CLI.
func listOpenBeadsIDs() (map[string]bool, error) {
	result := make(map[string]bool)

	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		if err := client.Connect(); err == nil {
			defer client.Close()
			for _, status := range []string{"open", "in_progress"} {
				issues, err := client.List(&beads.ListArgs{Status: status, Limit: beads.IntPtr(0)})
				if err == nil {
					for _, issue := range issues {
						result[issue.ID] = true
					}
				}
			}
			return result, nil
		}
	}

	// Fallback to CLI: bd list --json --limit 0 --status open
	for _, status := range []string{"open", "in_progress"} {
		output, err := runBdCommand("list", "--json", "--limit", "0", "--status", status)
		if err != nil {
			continue
		}
		var issues []struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(output, &issues); err != nil {
			continue
		}
		for _, issue := range issues {
			result[issue.ID] = true
		}
	}

	return result, nil
}

func cleanStaleTmuxWindows(serverURL string, preserveOrchestrator bool) (int, error) {
	client := opencode.NewClient(serverURL)
	now := time.Now()

	// Source 1: OpenCode sessions (headless/tmux backend)
	activeOpenCodeIDs := make(map[string]bool)
	sessions, err := client.ListSessions("")
	if err == nil {
		for _, s := range sessions {
			updatedAt := time.Unix(s.Time.Updated/1000, 0)
			if now.Sub(updatedAt) <= cleanupMaxIdleTime {
				beadsID := extractBeadsIDFromTitle(s.Title)
				if beadsID != "" {
					activeOpenCodeIDs[beadsID] = true
				}
			}
		}
	}

	// Source 2: Batch-fetch all open/in_progress beads IDs (one call instead of per-window)
	openBeadsIDs, err := listOpenBeadsIDs()
	if err != nil {
		// Can't determine open issues — set to nil so isWindowStaleBatch fails safe
		openBeadsIDs = nil
	}

	workersSessions, _ := tmux.ListWorkersSessions()
	var staleWindows []tmux.WindowInfo
	for _, sessionName := range workersSessions {
		if preserveOrchestrator && (sessionName == tmux.OrchestratorSessionName || sessionName == tmux.MetaOrchestratorSessionName) {
			continue
		}

		windows, err := tmux.ListWindows(sessionName)
		if err != nil {
			continue
		}

		for _, w := range windows {
			if w.Name == "servers" || w.Name == "zsh" {
				continue
			}

			beadsID := extractBeadsIDFromWindowName(w.Name)
			if beadsID == "" {
				continue
			}

			if isWindowStaleBatch(beadsID, activeOpenCodeIDs, openBeadsIDs) {
				staleWindows = append(staleWindows, w)
			}
		}
	}

	closed := 0
	for _, w := range staleWindows {
		if err := tmux.KillWindow(w.Target); err != nil {
			continue
		}
		closed++
	}

	return closed, nil
}

func extractBeadsIDFromTitle(title string) string {
	start := strings.LastIndex(title, "[")
	end := strings.LastIndex(title, "]")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return strings.TrimSpace(title[start+1 : end])
}

func extractBeadsIDFromWindowName(name string) string {
	start := strings.LastIndex(name, "[")
	end := strings.LastIndex(name, "]")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return strings.TrimSpace(name[start+1 : end])
}
