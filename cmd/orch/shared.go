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
	"github.com/dylan-conlin/orch-go/pkg/beadsutil"
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

// extractBeadsIDFromTitle delegates to beadsutil.ExtractIDFromTitle.
func extractBeadsIDFromTitle(title string) string { return beadsutil.ExtractIDFromTitle(title) }

// extractSkillFromTitle extracts skill from an OpenCode session title.
// Infers skill from common workspace name prefixes (og-feat-, og-inv-, og-debug-, etc.)
func extractSkillFromTitle(title string) string {
	titleLower := strings.ToLower(title)
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
	if strings.Contains(titleLower, "-work-") {
		return "work"
	}
	return ""
}

// extractBeadsIDFromWindowName delegates to beadsutil.ExtractIDFromWindowName.
func extractBeadsIDFromWindowName(name string) string {
	return beadsutil.ExtractIDFromWindowName(name)
}

// extractSkillFromWindowName extracts skill from a tmux window name.
// First tries to match skill emoji, then falls back to workspace name patterns.
func extractSkillFromWindowName(name string) string {
	for skill, emoji := range tmux.SKILL_EMOJIS {
		if strings.Contains(name, emoji) {
			return skill
		}
	}
	return extractSkillFromTitle(name)
}

// extractProjectFromBeadsID delegates to beadsutil.ExtractProjectFromID.
func extractProjectFromBeadsID(beadsID string) string {
	return beadsutil.ExtractProjectFromID(beadsID)
}

// findWorkspaceByBeadsID delegates to workspace.FindByBeadsID.
func findWorkspaceByBeadsID(projectDir, beadsID string) (workspacePath, agentName string) {
	return workspace.FindByBeadsID(projectDir, beadsID)
}

// workspaceSpawnTime delegates to workspace.SpawnTime.
func workspaceSpawnTime(wsPath string) int64 { return workspace.SpawnTime(wsPath) }

// isOpenCodeSessionID returns true if the string looks like a valid OpenCode session ID.
func isOpenCodeSessionID(id string) bool {
	return strings.HasPrefix(id, "ses_")
}

// resolveSessionID resolves an identifier to an OpenCode session ID.
// The identifier can be:
// 1. A full OpenCode session ID (ses_xxx) - verified against API, returned if valid
// 2. A beads ID (project-xxxx) - looked up via workspace SPAWN_CONTEXT.md or API
// 3. A workspace name - looked up via workspace file
func resolveSessionID(serverURL, identifier string) (string, error) {
	if strings.HasPrefix(identifier, "ses_") {
		suffix := strings.TrimPrefix(identifier, "ses_")
		if len(suffix) < 8 {
			return "", fmt.Errorf("invalid session ID format: %s (too short)", identifier)
		}
		client := opencode.NewClient(serverURL)
		_, err := client.GetSession(identifier)
		if err != nil {
			return "", fmt.Errorf("session not found in OpenCode: %s", identifier)
		}
		return identifier, nil
	}

	client := opencode.NewClient(serverURL)
	projectDir, _ := os.Getwd()

	// Strategy 1: Use workspace.FindByBeadsID which scans SPAWN_CONTEXT.md
	workspacePath, _ := workspace.FindByBeadsID(projectDir, identifier)
	if workspacePath != "" {
		sessionID := spawn.ReadSessionID(workspacePath)
		if sessionID != "" && isOpenCodeSessionID(sessionID) {
			return sessionID, nil
		}
	}

	// Strategy 2: Direct workspace name match
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
		if strings.Contains(s.Title, identifier) || beadsutil.ExtractIDFromTitle(s.Title) == identifier {
			return s.ID, nil
		}
	}

	// Strategy 4: tmux window lookup as last resort
	sessions, err := tmux.ListWorkersSessions()
	if err == nil {
		for _, session := range sessions {
			window, err := tmux.FindWindowByBeadsID(session, identifier)
			if err != nil || window == nil {
				continue
			}
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
func findTmuxWindowByIdentifier(identifier string) (*tmux.WindowInfo, error) {
	sessions, err := tmux.ListWorkersSessions()
	if err != nil {
		return nil, err
	}

	if tmux.SessionExists(tmux.OrchestratorSessionName) {
		sessions = append(sessions, tmux.OrchestratorSessionName)
	}
	if tmux.SessionExists(tmux.MetaOrchestratorSessionName) {
		sessions = append(sessions, tmux.MetaOrchestratorSessionName)
	}

	for _, session := range sessions {
		window, err := tmux.FindWindowByBeadsID(session, identifier)
		if err == nil && window != nil {
			return window, nil
		}

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

	return nil, nil
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

// resolveShortBeadsID delegates to beadsutil.ResolveShortID.
func resolveShortBeadsID(id string) (string, error) { return beadsutil.ResolveShortID(id) }

// Process-level capacity cache shared across spawn/rework commands.
var processCapacityCache *account.CapacityCache

// buildCapacityFetcher returns a function that checks account capacity
// using a process-level cache.
func buildCapacityFetcher() func(string) *account.CapacityInfo {
	cfg, err := account.LoadConfig()
	if err != nil || len(cfg.Accounts) < 2 {
		return nil
	}

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

	if processCapacityCache == nil {
		processCapacityCache = account.NewCapacityCache(5 * time.Minute)
	}

	return func(name string) *account.CapacityInfo {
		if cached := processCapacityCache.Get(name); cached != nil {
			return cached
		}
		capacity, err := account.GetAccountCapacity(name)
		if err != nil || capacity == nil {
			return nil
		}
		processCapacityCache.Set(name, capacity)
		return capacity
	}
}

// buildOpenQuestionChecker creates an OpenQuestionChecker backed by the beads client.
func buildOpenQuestionChecker() gates.OpenQuestionChecker {
	fetcher := func(issueID string) (*gates.IssueSummary, error) {
		socketPath, err := beads.FindSocketPath("")
		if err == nil {
			client := beads.NewClient(socketPath, beads.WithAutoReconnect(2))
			defer client.Close()
			issue, showErr := client.Show(issueID)
			if showErr == nil {
				return issueToSummary(issue), nil
			}
		}
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
