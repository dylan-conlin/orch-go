// Package main provides shared utility functions used across multiple commands.
// This file contains extraction and lookup utilities that are used by spawn, status,
// complete, send, tail, question, abandon, and other commands.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/account"
	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/display"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/spawn/gates"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/dylan-conlin/orch-go/pkg/workspace"
)

// truncate delegates to display.Truncate.
func truncate(s string, maxLen int) string { return display.Truncate(s, maxLen) }

// shortID delegates to display.ShortID.
func shortID(s string) string { return display.ShortID(s) }

// formatDuration delegates to display.FormatDuration.
func formatDuration(d time.Duration) string { return display.FormatDuration(d) }

// extractBeadsIDFromTitle extracts beads ID from an OpenCode session title.
// Looks for patterns like "[beads-id]" at the end of the title.
func extractBeadsIDFromTitle(title string) string {
	// Look for "[beads-id]" pattern
	if start := strings.LastIndex(title, "["); start != -1 {
		if end := strings.LastIndex(title, "]"); end != -1 && end > start {
			return strings.TrimSpace(title[start+1 : end])
		}
	}
	return ""
}

// extractSkillFromTitle extracts skill from an OpenCode session title.
// Infers skill from common workspace name prefixes (og-feat-, og-inv-, og-debug-, etc.)
func extractSkillFromTitle(title string) string {
	titleLower := strings.ToLower(title)
	// Check for workspace name patterns
	if strings.Contains(titleLower, "-feat-") {
		return "feature-impl"
	}
	if strings.Contains(titleLower, "-inv-") {
		return "investigation"
	}
	if strings.Contains(titleLower, "-debug-") {
		return "systematic-debugging"
	}
	if strings.Contains(titleLower, "-arch-") {
		return "architect"
	}
	if strings.Contains(titleLower, "-audit-") {
		return "codebase-audit"
	}
	if strings.Contains(titleLower, "-research-") {
		return "research"
	}
	return ""
}

// extractBeadsIDFromWindowName extracts beads ID from a tmux window name.
// Window names follow format: "emoji workspace-name [beads-id]"
func extractBeadsIDFromWindowName(name string) string {
	// Look for "[beads-id]" pattern
	if start := strings.LastIndex(name, "["); start != -1 {
		if end := strings.LastIndex(name, "]"); end != -1 && end > start {
			return strings.TrimSpace(name[start+1 : end])
		}
	}
	return ""
}

// extractSkillFromWindowName extracts skill from a tmux window name.
// First tries to match skill emoji, then falls back to workspace name patterns.
func extractSkillFromWindowName(name string) string {
	// Try emoji matching first (most reliable)
	for skill, emoji := range tmux.SKILL_EMOJIS {
		if strings.Contains(name, emoji) {
			return skill
		}
	}
	// Fall back to workspace name patterns
	return extractSkillFromTitle(name)
}

// extractProjectFromBeadsID extracts the project name from a beads ID.
// Beads IDs follow the format: project-xxxx (e.g., orch-go-3anf)
func extractProjectFromBeadsID(beadsID string) string {
	if beadsID == "" {
		return ""
	}
	// Find the last hyphen followed by 4 alphanumeric characters (the hash)
	// The project is everything before that
	parts := strings.Split(beadsID, "-")
	if len(parts) < 2 {
		return beadsID
	}
	// The last part should be the 4-char hash, join everything else
	return strings.Join(parts[:len(parts)-1], "-")
}

// findWorkspaceByBeadsID delegates to workspace.FindByBeadsID.
func findWorkspaceByBeadsID(projectDir, beadsID string) (workspacePath, agentName string) {
	return workspace.FindByBeadsID(projectDir, beadsID)
}

// workspaceSpawnTime delegates to workspace.SpawnTime.
func workspaceSpawnTime(wsPath string) int64 {
	return workspace.SpawnTime(wsPath)
}

// resolveSessionID resolves an identifier to an OpenCode session ID.
// isOpenCodeSessionID returns true if the string looks like a valid OpenCode session ID.
// OpenCode session IDs start with "ses_". Tmux window IDs start with "@" and should
// not be treated as OpenCode sessions. Claude-backend agents store window IDs
// in workspace session_id files, not OpenCode session IDs.
func isOpenCodeSessionID(id string) bool {
	return strings.HasPrefix(id, "ses_")
}

// The identifier can be:
// 1. A full OpenCode session ID (ses_xxx) - verified against API, returned if valid
// 2. A beads ID (project-xxxx) - looked up via workspace SPAWN_CONTEXT.md or API
// 3. A workspace name - looked up via workspace file
//
// Returns the resolved session ID or an error if resolution fails.
func resolveSessionID(serverURL, identifier string) (string, error) {
	// If it looks like a full session ID, verify it exists
	if strings.HasPrefix(identifier, "ses_") {
		// Validate the session ID has content after the prefix
		suffix := strings.TrimPrefix(identifier, "ses_")
		if len(suffix) < 8 { // Session IDs have substantial content after ses_
			return "", fmt.Errorf("invalid session ID format: %s (too short)", identifier)
		}
		// Verify the session exists in OpenCode
		client := opencode.NewClient(serverURL)
		_, err := client.GetSession(identifier)
		if err != nil {
			return "", fmt.Errorf("session not found in OpenCode: %s", identifier)
		}
		return identifier, nil
	}

	client := opencode.NewClient(serverURL)
	projectDir, _ := os.Getwd()

	// Strategy 1: Use findWorkspaceByBeadsID which scans SPAWN_CONTEXT.md
	// This is the authoritative way to find workspace by beads ID
	workspacePath, _ := findWorkspaceByBeadsID(projectDir, identifier)
	if workspacePath != "" {
		sessionID := spawn.ReadSessionID(workspacePath)
		if sessionID != "" && isOpenCodeSessionID(sessionID) {
			return sessionID, nil
		}
		// If session_id is a tmux window ID (@xxx) or other non-OpenCode format,
		// skip it and fall through to tmux lookup. Claude-backend agents store
		// window IDs in session_id, not OpenCode session IDs.
	}

	// Strategy 2: Direct workspace name match (for workspace name identifiers)
	workspaceBase := filepath.Join(projectDir, ".orch", "workspace")
	if entries, err := os.ReadDir(workspaceBase); err == nil {
		for _, entry := range entries {
			if entry.IsDir() && strings.Contains(entry.Name(), identifier) {
				wp := filepath.Join(workspaceBase, entry.Name())
				sessionID := spawn.ReadSessionID(wp)
				if sessionID != "" && isOpenCodeSessionID(sessionID) {
					return sessionID, nil
				}
			}
		}
	}

	// Strategy 3: API lookup - search sessions by title containing identifier
	allSessions, err := client.ListSessions(projectDir)
	if err != nil {
		return "", fmt.Errorf("failed to list sessions: %w", err)
	}

	for _, s := range allSessions {
		// Match session by title containing identifier (beads ID or workspace name)
		if strings.Contains(s.Title, identifier) || extractBeadsIDFromTitle(s.Title) == identifier {
			return s.ID, nil
		}
	}

	// Strategy 4: tmux window lookup as last resort - find window, then try to get session
	sessions, err := tmux.ListWorkersSessions()
	if err == nil {
		for _, session := range sessions {
			window, err := tmux.FindWindowByBeadsID(session, identifier)
			if err != nil || window == nil {
				continue
			}

			// Found tmux window - try to find matching OpenCode session by window name
			// Window names have workspace names in them
			for _, s := range allSessions {
				if strings.Contains(window.Name, s.Title) || strings.Contains(s.Title, identifier) {
					return s.ID, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no session found for identifier: %s (checked workspace files, API sessions, and tmux windows)", identifier)
}

// findTmuxWindowByIdentifier searches for a tmux window matching the identifier.
// The identifier can be a beads ID, workspace name, or partial match.
// Searches workers sessions, orchestrator session, and meta-orchestrator session.
func findTmuxWindowByIdentifier(identifier string) (*tmux.WindowInfo, error) {
	sessions, err := tmux.ListWorkersSessions()
	if err != nil {
		return nil, err
	}

	// Also search orchestrator and meta-orchestrator sessions
	if tmux.SessionExists(tmux.OrchestratorSessionName) {
		sessions = append(sessions, tmux.OrchestratorSessionName)
	}
	if tmux.SessionExists(tmux.MetaOrchestratorSessionName) {
		sessions = append(sessions, tmux.MetaOrchestratorSessionName)
	}

	for _, session := range sessions {
		// First try exact beads ID match (format: "[beads-id]" in window name)
		window, err := tmux.FindWindowByBeadsID(session, identifier)
		if err == nil && window != nil {
			return window, nil
		}

		// Also try partial match on window name (for workspace name matches)
		windows, err := tmux.ListWindows(session)
		if err != nil {
			continue
		}
		for i := range windows {
			if strings.Contains(windows[i].Name, identifier) {
				return &windows[i], nil
			}
		}
	}

	return nil, nil // Not found (no error, just not found)
}

// findWorkspaceByName delegates to workspace.FindByName.
func findWorkspaceByName(projectDir, workspaceName string) string {
	return workspace.FindByName(projectDir, workspaceName)
}

// isOrchestratorWorkspace delegates to workspace.IsOrchestrator.
func isOrchestratorWorkspace(workspacePath string) bool {
	return workspace.IsOrchestrator(workspacePath)
}

// hasSessionHandoff delegates to workspace.HasSessionHandoff.
func hasSessionHandoff(workspacePath string) bool {
	return workspace.HasSessionHandoff(workspacePath)
}

// resolveShortBeadsID resolves a potentially short beads ID to a full ID.
// Short IDs like "57dn" are resolved to full IDs like "orch-go-57dn".
// This ensures commands receive full IDs that bd commands can use.
// Returns an error if the issue doesn't exist - this prevents spawning
// agents with invalid beads IDs that can never be closed.
func resolveShortBeadsID(id string) (string, error) {
	// Try RPC client first for ID resolution
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath)
		if err := client.Connect(); err == nil {
			defer client.Close()

			resolvedID, err := client.ResolveID(id)
			if err == nil && resolvedID != "" {
				return resolvedID, nil
			}
			// Fall through to CLI fallback on RPC error
		}
	}

	// Fallback: Use bd show to resolve the ID
	// bd show handles short ID resolution and returns the full ID
	issue, err := beads.FallbackShow(id, "")
	if err != nil {
		// Issue doesn't exist - return error with helpful hint for cross-project issues
		// Extract project prefix from ID if present (e.g., "kb-cli" from "kb-cli-xyz123")
		hint := ""
		if parts := strings.Split(id, "-"); len(parts) >= 2 {
			// Check if ID looks like it has a project prefix (e.g., "kb-cli-xyz123")
			// Project prefixes are typically not just single short segments
			possibleProject := parts[0]
			if len(parts) >= 3 {
				possibleProject = parts[0] + "-" + parts[1]
			}
			hint = fmt.Sprintf("\n\nHint: Issue '%s' may belong to a different project.\n"+
				"If the issue is in '%s', try:\n"+
				"  cd ~/Documents/personal/%s && orch complete %s\n"+
				"Or use --workdir:\n"+
				"  orch complete %s --workdir ~/Documents/personal/%s",
				id, possibleProject, possibleProject, id, id, possibleProject)
		}
		return "", fmt.Errorf("beads issue '%s' not found%s", id, hint)
	}

	return issue.ID, nil
}

// Process-level capacity cache shared across spawn/rework commands.
// Since each orch command is a short-lived process, the cache ensures
// multiple resolve calls within the same process don't re-fetch capacity.
var processCapacityCache *account.CapacityCache

// buildCapacityFetcher returns a function that checks account capacity
// using a process-level cache. The fetcher calls GetAccountCapacity on
// cache miss and caches the result for 5 minutes.
//
// Returns nil if no saved accounts are configured (no routing needed).
func buildCapacityFetcher() func(string) *account.CapacityInfo {
	cfg, err := account.LoadConfig()
	if err != nil || len(cfg.Accounts) < 2 {
		// No routing needed with 0 or 1 accounts
		return nil
	}

	// Check if any accounts have routing roles configured
	hasRoles := false
	for _, acc := range cfg.Accounts {
		if acc.Role == "primary" || acc.Role == "spillover" {
			hasRoles = true
			break
		}
	}
	if !hasRoles {
		return nil
	}

	// Lazy-init the process-level cache
	if processCapacityCache == nil {
		processCapacityCache = account.NewCapacityCache(5 * time.Minute)
	}

	return func(name string) *account.CapacityInfo {
		// Check cache first
		if cached := processCapacityCache.Get(name); cached != nil {
			return cached
		}

		// Cache miss — fetch from API
		capacity, err := account.GetAccountCapacity(name)
		if err != nil || capacity == nil {
			return nil
		}

		processCapacityCache.Set(name, capacity)
		return capacity
	}
}

// buildOpenQuestionChecker creates an OpenQuestionChecker backed by the beads client.
// Returns nil if beads is not available (graceful degradation).
func buildOpenQuestionChecker() gates.OpenQuestionChecker {
	fetcher := func(issueID string) (*gates.IssueSummary, error) {
		// Try RPC client first
		socketPath, err := beads.FindSocketPath("")
		if err == nil {
			client := beads.NewClient(socketPath, beads.WithAutoReconnect(2))
			defer client.Close()
			issue, showErr := client.Show(issueID)
			if showErr == nil {
				return issueToSummary(issue), nil
			}
		}

		// Fallback to CLI
		issue, err := beads.FallbackShow(issueID, "")
		if err != nil {
			return nil, err
		}
		return issueToSummary(issue), nil
	}
	return gates.BuildOpenQuestionChecker(fetcher)
}

// issueToSummary converts a beads Issue to a gates.IssueSummary.
func issueToSummary(issue *beads.Issue) *gates.IssueSummary {
	summary := &gates.IssueSummary{
		ID:        issue.ID,
		IssueType: issue.IssueType,
		Status:    issue.Status,
		Title:     issue.Title,
	}
	deps := issue.ParseDependencies()
	for _, dep := range deps {
		summary.Deps = append(summary.Deps, gates.DepSummary{
			ID:     dep.EffectiveID(),
			Type:   dep.EffectiveType(),
			Status: dep.Status,
		})
	}
	return summary
}
