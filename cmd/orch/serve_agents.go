package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/agent"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/port"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// AgentAPIResponse is the JSON structure returned by /api/agents.
type AgentAPIResponse struct {
	ID                   string               `json:"id"`
	SessionID            string               `json:"session_id,omitempty"`
	BeadsID              string               `json:"beads_id,omitempty"`
	BeadsTitle           string               `json:"beads_title,omitempty"`
	BeadsLabels          []string             `json:"beads_labels,omitempty"` // Labels from beads issue
	Skill                string               `json:"skill,omitempty"`
	Model                string               `json:"model,omitempty"`        // Model ID from session messages (e.g., "claude-opus-4-5-20251101")
	Status               string               `json:"status"`                 // "active", "idle", "dead", "completed", "awaiting-cleanup"
	DeathReason          string               `json:"death_reason,omitempty"` // Reason for death: "server_restart", "context_exhausted", "auth_failed", "error", "timeout", "unknown"
	Phase                string               `json:"phase,omitempty"`        // "Planning", "Implementing", "Complete", etc.
	Task                 string               `json:"task,omitempty"`         // Task description from beads issue
	Project              string               `json:"project,omitempty"`      // Project name (orch-go, skillc, etc.)
	Runtime              string               `json:"runtime,omitempty"`
	Window               string               `json:"window,omitempty"`
	IsProcessing         bool                 `json:"is_processing,omitempty"` // True if actively generating response
	IsStale              bool                 `json:"is_stale,omitempty"`      // True if agent is older than beadsFetchThreshold (beads data not fetched)
	IsStalled            bool                 `json:"is_stalled,omitempty"`    // True if active agent has same phase for 15+ minutes (advisory)
	SpawnedAt            string               `json:"spawned_at,omitempty"`    // ISO 8601 timestamp
	UpdatedAt            string               `json:"updated_at,omitempty"`    // ISO 8601 timestamp
	Synthesis            *SynthesisResponse   `json:"synthesis,omitempty"`
	CloseReason          string               `json:"close_reason,omitempty"`          // Beads close reason, fallback for completed agents without synthesis
	GapAnalysis          *GapAPIResponse      `json:"gap_analysis,omitempty"`          // Context gap analysis from spawn time
	Tokens               *opencode.TokenStats `json:"tokens,omitempty"`                // Token usage for the session
	InvestigationPath    string               `json:"investigation_path,omitempty"`    // Path to investigation file from beads comments
	ProjectDir           string               `json:"project_dir,omitempty"`           // Project directory for the agent
	SynthesisContent     string               `json:"synthesis_content,omitempty"`     // Raw SYNTHESIS.md content for inline rendering
	InvestigationContent string               `json:"investigation_content,omitempty"` // Raw investigation file content for inline rendering
	CurrentActivity      string               `json:"current_activity,omitempty"`      // Last activity text from session messages
	LastActivityAt       string               `json:"last_activity_at,omitempty"`      // ISO 8601 timestamp of last activity
}

// GapAPIResponse represents gap analysis data for the API.
type GapAPIResponse struct {
	HasGaps        bool `json:"has_gaps"`
	ContextQuality int  `json:"context_quality"`
	ShouldWarn     bool `json:"should_warn"`
	MatchCount     int  `json:"match_count,omitempty"`
	Constraints    int  `json:"constraints,omitempty"`
	Decisions      int  `json:"decisions,omitempty"`
	Investigations int  `json:"investigations,omitempty"`
}

// SynthesisResponse is a condensed version of verify.Synthesis for the API.
// Uses the D.E.K.N. structure: Delta, Evidence, Knowledge, Next.
type SynthesisResponse struct {
	// Header fields
	TLDR           string `json:"tldr,omitempty"`
	Outcome        string `json:"outcome,omitempty"`        // success, partial, blocked, failed
	Recommendation string `json:"recommendation,omitempty"` // close, continue, escalate

	// Condensed sections
	DeltaSummary string   `json:"delta_summary,omitempty"` // e.g., "3 files created, 2 modified, 5 commits"
	NextActions  []string `json:"next_actions,omitempty"`  // Follow-up items
}

// investigationDirCache holds pre-loaded directory listings for investigation discovery.
// This prevents O(n²) behavior when discovering investigation paths for many agents.
// Without this cache, each agent would call os.ReadDir() 2-3 times on directories
// with 500+ files, resulting in 300+ agents × 500+ files × 2 calls = massive slowdown.
type investigationDirCache struct {
	// entries maps directory path -> list of .md file names (not full DirEntry, just names for efficiency)
	entries map[string][]string
}

// determineDeathReason analyzes why an agent died and returns a specific reason code.
// Reasons: "server_restart", "context_exhausted", "auth_failed", "error", "timeout", "unknown"
func determineDeathReason(sessionID string, sessionCreatedAt time.Time, client *opencode.Client) string {
	// Check if session existed before server restart
	if !serverStartTime.IsZero() && sessionCreatedAt.Before(serverStartTime) {
		return "server_restart"
	}

	// Check last message for specific error patterns
	lastMsg, err := client.GetLastMessage(sessionID)
	if err != nil || lastMsg == nil {
		// No messages available, likely timeout
		return "timeout"
	}

	// Check for context exhaustion (token limit errors)
	// OpenCode reports token limits in error messages
	for _, part := range lastMsg.Parts {
		if part.Type == "text" && part.Text != "" {
			text := strings.ToLower(part.Text)
			if strings.Contains(text, "token limit") ||
				strings.Contains(text, "context length") ||
				strings.Contains(text, "maximum context") ||
				strings.Contains(text, "too many tokens") {
				return "context_exhausted"
			}
		}
	}

	// Check for auth failures (401/403 errors)
	// Look at message finish reason or error info
	if lastMsg.Info.Finish != "" {
		finish := strings.ToLower(lastMsg.Info.Finish)
		if strings.Contains(finish, "unauthorized") ||
			strings.Contains(finish, "forbidden") ||
			strings.Contains(finish, "authentication") ||
			strings.Contains(finish, "auth") {
			return "auth_failed"
		}
	}

	// Check for general errors in the last message
	// If the last message had an error, it's an error death
	if lastMsg.Info.Finish == "error" || lastMsg.Info.Finish == "tool_error" {
		return "error"
	}

	// Check message parts for error indicators
	for _, part := range lastMsg.Parts {
		if part.Type == "error" || (part.Type == "text" && strings.HasPrefix(strings.ToLower(part.Text), "error:")) {
			return "error"
		}
	}

	// If we got here, it's likely a timeout (no activity for threshold period)
	// This is the most common death reason for agents that just stop responding
	return "timeout"
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
// IMPORTANT: Pass a pre-built investigationDirCache to avoid O(n²) directory scanning.
// Without the cache, this function would call os.ReadDir() for each agent, causing
// massive slowdowns with 300+ agents and 500+ investigation files.
//
// Fallback chain:
// 1. Search .kb/investigations/ for files matching beads ID (most specific)
// 2. Check workspace directory for investigation .md files (excluding SPAWN_CONTEXT.md and SYNTHESIS.md)
//
// NOTE: Keyword-based matching was REMOVED (Jan 2026) because it caused wrong investigations
// to be shown (e.g., "epic-auto-close" matching "audit-opencode-plugin" on common words).
// Agents should report investigation_path via beads comments for explicit linking.
func discoverInvestigationPath(workspaceName, beadsID, projectDir string, cache *investigationDirCache) string {
	if projectDir == "" {
		return ""
	}

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
	// BeadsID match doesn't need timestamp filtering - if the investigation file was
	// named with this agent's beads ID, it was definitely created for this agent.
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

	// NOTE: Keyword-based fallback matching was REMOVED (Jan 2026) because it caused
	// wrong investigations to be shown. Example: an agent working on "epic-auto-close"
	// would match "audit-opencode-plugin" because both contain common words.
	// Now we only match by beads ID (explicit) or workspace directory files (implicit).
	// Agents should report investigation_path via beads comments for explicit linking.

	// 2. Check workspace directory for investigation .md files
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
				name == ".session_id" || name == ".spawn_time" ||
				name == ".tier" || name == ".beads_id" {
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

// handleAgents returns JSON list of active agents from OpenCode/tmux and completed workspaces.
// Query parameters:
//   - since: Time filter (12h, 24h, 48h, 7d, all). Default: 12h
//   - project: Project filter (full path or project name). Default: none (all projects)
func handleAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters for filtering
	sinceDuration := parseSinceParam(r)
	projectFilterParam := parseProjectFilter(r)

	// Use sourceDir (set at build time) since serve may run from any working directory
	projectDir := sourceDir
	if projectDir == "" || projectDir == "unknown" {
		projectDir, _ = os.Getwd()
	}

	client := opencode.NewClient(serverURL)

	// Get active sessions from OpenCode
	// Don't filter by directory - show all sessions across all projects
	// (serve process CWD may not match project directory)
	sessions, err := client.ListSessions("")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list sessions: %v", err), http.StatusInternalServerError)
		return
	}

	// EARLY FILTERING: Apply time filter immediately after fetching sessions.
	// This is critical for performance - filtering BEFORE expensive operations (workspace cache,
	// beads batch fetches) reduces the number of sessions we need to process.
	// Previously: filters were applied at the END, causing 20s+ cold cache times.
	// Now: time filter applied early, reducing expensive operations proportionally.
	//
	// NOTE: Project filter is NOT applied here because s.Directory may be the orchestrator's cwd
	// due to OpenCode --attach bug, not the actual target project directory from --workdir spawns.
	// The correct project_dir is populated later from workspace cache (line ~727) and filtered
	// at the end (line ~894) using agent.ProjectDir which has the correct value.
	now := time.Now()
	if sinceDuration > 0 {
		filtered := make([]opencode.Session, 0, len(sessions))
		for _, s := range sessions {
			// Time filter: check session updated_at or created_at
			updatedAt := time.Unix(s.Time.Updated/1000, 0)
			if now.Sub(updatedAt) > sinceDuration {
				// Session is too old, skip it
				continue
			}

			filtered = append(filtered, s)
		}
		sessions = filtered
	}

	// Build multi-project workspace cache for cross-project agent visibility.
	// This aggregates workspace metadata from all projects with active sessions,
	// enabling the dashboard to show correct status for agents spawned with --workdir.
	// Previously: Only scanned current project's .orch/workspace/
	// Now: Scans all unique project directories found in OpenCode sessions
	// CACHED: Workspace scanning is slow (600+ files), so cache with 30s TTL.
	projectDirs := extractUniqueProjectDirs(sessions, projectDir)
	wsCache := globalWorkspaceCacheInstance.getCachedWorkspace(projectDirs)

	agents := []AgentAPIResponse{} // Initialize as empty slice, not nil, to return [] instead of null

	// Collect beads IDs for batch fetching
	var beadsIDsToFetch []string
	seenBeadsIDs := make(map[string]bool)

	// Track project directories for cross-project agents
	// Key: beadsID, Value: projectDir from workspace SPAWN_CONTEXT.md
	beadsProjectDirs := make(map[string]string)

	// Add active sessions from OpenCode
	// Filter: only show sessions updated in the last 10 minutes as "active"
	// Two-threshold ghost filtering:
	// - Active threshold (10min): determines "running" vs "idle" status
	// - Display threshold (4h): filters ghosts from default view (unless Phase: Complete)
	activeThreshold := 10 * time.Minute
	displayThreshold := 4 * time.Hour

	// Dead threshold: if no activity for 3 minutes, session is dead.
	// Agents are constantly reading, editing, running commands - 3 min silence = dead.
	// This is critical for visibility: dead agents need attention (crashed/stuck/killed).
	deadThreshold := 3 * time.Minute

	// Stalled threshold: if same phase for 15+ minutes, agent may be stuck.
	// This is advisory only - surfaces in Needs Attention but doesn't auto-abandon.
	// Designed to catch agents that have heartbeat but aren't making progress.
	// See .kb/investigations/2026-01-08-inv-design-stalled-agent-detection-agents.md
	stalledThreshold := 15 * time.Minute

	// beadsFetchThreshold limits which sessions we fetch beads data for.
	// Sessions older than this are excluded from beads lookups entirely.
	// This is a MAJOR optimization: with 600+ sessions but only ~6 active,
	// fetching beads for all would require 400+ RPC calls = 3+ seconds.
	// By limiting to recent sessions, we reduce this to ~10-20 RPC calls.
	// Sessions older than this are simply excluded from the API response.
	beadsFetchThreshold := 2 * time.Hour
	if sinceDuration > beadsFetchThreshold {
		beadsFetchThreshold = sinceDuration
	} else if sinceDuration == 0 {
		// "all" requested
		beadsFetchThreshold = 365 * 24 * time.Hour
	}

	// Track which agents need post-filtering by beads ID (idle > displayThreshold)
	// These will be filtered out after Phase check unless Phase: Complete
	pendingFilterByBeadsID := make(map[string]bool)

	// Track agents by title to deduplicate (OpenCode can have multiple sessions with same title)
	// Keep the most recently updated session for each title
	seenTitles := make(map[string]int) // title -> index in agents slice

	for _, s := range sessions {
		createdAt := time.Unix(s.Time.Created/1000, 0)
		updatedAt := time.Unix(s.Time.Updated/1000, 0)
		runtime := now.Sub(createdAt)
		timeSinceUpdate := now.Sub(updatedAt)

		// Determine status based on recent activity
		// Priority: dead (3min silence) > active (recent) > idle (10min+)
		// Dead agents need attention - they're crashed/stuck/killed.
		status := "active"
		var deathReason string
		if timeSinceUpdate > deadThreshold {
			status = "dead" // No activity for 3+ minutes = dead (crashed/stuck/killed)
			// Determine specific death reason for diagnostics
			deathReason = determineDeathReason(s.ID, createdAt, client)
		} else if timeSinceUpdate > activeThreshold {
			status = "idle" // Session exists but hasn't had recent activity
		}

		// NOTE: IsProcessing is now populated client-side via SSE session.status events.
		// Previously we called client.IsSessionProcessing(s.ID) here, but that makes
		// an HTTP call per session which caused 125% CPU when dashboard polled frequently.
		// The frontend already receives busy/idle state from OpenCode SSE and updates
		// is_processing in real-time, so we don't need to fetch it here.

		agent := AgentAPIResponse{
			ID:           s.Title,
			SessionID:    s.ID,
			Status:       status,
			DeathReason:  deathReason,
			Runtime:      formatDuration(runtime),
			SpawnedAt:    createdAt.Format(time.RFC3339),
			UpdatedAt:    updatedAt.Format(time.RFC3339),
			IsProcessing: false,       // Populated client-side via SSE
			ProjectDir:   s.Directory, // Set from session directory for project filtering
		}

		// Derive beadsID and skill from session title
		if s.Title != "" {
			agent.BeadsID = extractBeadsIDFromTitle(s.Title)
			agent.Skill = extractSkillFromTitle(s.Title)
			agent.Project = extractProjectFromBeadsID(agent.BeadsID)

		}

		// Only include sessions that were spawned via orch spawn (have beads ID)
		// This filters out interactive/ad-hoc OpenCode sessions
		if agent.BeadsID == "" {
			continue
		}

		// OPTIMIZATION: Mark sessions older than beadsFetchThreshold as stale.
		// We still include them in the response (for dashboard visibility) but skip
		// beads data fetch for performance. With 600+ sessions but only ~6-10 active,
		// fetching beads for all would require 400+ RPC calls = 3+ seconds per request.
		// By marking old sessions as stale and skipping beads fetch, we reduce to ~20-50 RPC calls.
		isStale := timeSinceUpdate > beadsFetchThreshold
		if isStale {
			agent.IsStale = true
			agent.Status = "idle" // Stale agents are not actively running
		}

		// Track if this agent should be filtered after Phase check using two-threshold logic
		// Don't filter yet - we need to check beads Phase: Complete first
		// Don't filter stale agents - they're already marked with is_stale=true
		// Use aggressive threshold for marking (agents idle > 4h are candidates for filtering)
		if status == "idle" && timeSinceUpdate > displayThreshold && !isStale {
			pendingFilterByBeadsID[agent.BeadsID] = true
		}

		// For cross-project agent visibility: use cached PROJECT_DIR from workspace
		// This is O(1) lookup and should happen for ALL agents (including stale)
		// to ensure correct project filtering when using --workdir spawns.
		// The session's directory may be wrong (orchestrator's cwd vs target workdir).
		if agent.BeadsID != "" {
			if agentProjectDir := wsCache.lookupProjectDir(agent.BeadsID); agentProjectDir != "" {
				beadsProjectDirs[agent.BeadsID] = agentProjectDir
			}
		}

		// Collect beads ID for batch fetch - only for NON-STALE agents with beads ID.
		// Stale agents (older than beadsFetchThreshold) are included in response but
		// skip beads fetch for performance optimization.
		// See .kb/investigations/2026-01-04-design-dashboard-agent-status-model.md
		if agent.BeadsID != "" && !seenBeadsIDs[agent.BeadsID] && !isStale {
			beadsIDsToFetch = append(beadsIDsToFetch, agent.BeadsID)
			seenBeadsIDs[agent.BeadsID] = true
		}

		// Deduplicate by title - keep the most recently updated session
		// OpenCode can have multiple sessions with the same title (e.g., resumed agents)
		if existingIdx, exists := seenTitles[s.Title]; exists {
			// Compare updated_at to keep the more recent session
			existingUpdatedAt, _ := time.Parse(time.RFC3339, agents[existingIdx].UpdatedAt)
			if updatedAt.After(existingUpdatedAt) {
				// Replace the existing agent with this newer one
				agents[existingIdx] = agent
			}
			// Skip appending since we either replaced or kept the existing one
			continue
		}

		seenTitles[s.Title] = len(agents)
		agents = append(agents, agent)
	}

	// Add tmux-only agents
	workersSessions, _ := tmux.ListWorkersSessions()
	for _, sessionName := range workersSessions {
		windows, _ := tmux.ListWindows(sessionName)
		for _, win := range windows {
			if win.Name == "servers" || win.Name == "zsh" {
				continue
			}

			beadsID := extractBeadsIDFromWindowName(win.Name)
			skill := extractSkillFromWindowName(win.Name)
			project := extractProjectFromBeadsID(beadsID)

			// Check if already in agents list
			alreadyIn := false
			for _, a := range agents {
				if (beadsID != "" && a.BeadsID == beadsID) || (a.ID != "" && strings.Contains(win.Name, a.ID)) {
					alreadyIn = true
					break
				}
			}

			if !alreadyIn {
				agent := AgentAPIResponse{
					ID:      win.Name,
					BeadsID: beadsID,
					Skill:   skill,
					Project: project,
					Status:  "active",
					Window:  win.Target,
				}

				// Look up workspace path for spawn time and activity detection
				// Tmux-only agents (Claude CLI escape hatch) need this for visibility parity
				if beadsID != "" {
					if workspacePath := wsCache.lookupWorkspace(beadsID); workspacePath != "" {
						// Read spawn time for runtime calculation
						if spawnTime := spawn.ReadSpawnTime(workspacePath); !spawnTime.IsZero() {
							agent.SpawnedAt = spawnTime.Format(time.RFC3339)
							agent.Runtime = formatDuration(now.Sub(spawnTime))
						}

						// Look up project dir for agent
						if agentProjectDir := wsCache.lookupProjectDir(beadsID); agentProjectDir != "" {
							agent.ProjectDir = agentProjectDir
						}

						// Activity detection: check workspace file modification times
						// Tmux agents are "dead" if no workspace activity for 3+ minutes
						lastActivity := getWorkspaceLastActivity(workspacePath)
						if !lastActivity.IsZero() {
							agent.LastActivityAt = lastActivity.Format(time.RFC3339)
							timeSinceActivity := now.Sub(lastActivity)
							if timeSinceActivity > deadThreshold {
								agent.Status = "dead"
								// For tmux-only agents without session ID, default to timeout
								// (we can't inspect messages to determine more specific reason)
								agent.DeathReason = "timeout"
							}
						}
					}
				}

				agents = append(agents, agent)

				// Collect beads ID for batch fetch
				if beadsID != "" && !seenBeadsIDs[beadsID] {
					beadsIDsToFetch = append(beadsIDsToFetch, beadsID)
					seenBeadsIDs[beadsID] = true

					// For cross-project agent visibility: use cached PROJECT_DIR
					// This replaces expensive directory scanning with O(1) lookup
					if agentProjectDir := wsCache.lookupProjectDir(beadsID); agentProjectDir != "" {
						beadsProjectDirs[beadsID] = agentProjectDir
					}
				}
			}
		}
	}

	// Add completed workspaces (those with SYNTHESIS.md or light-tier completions)
	// Reuse cached workspace entries to avoid redundant directory reads
	// Multi-project support: entries may come from different project workspace directories
	if len(wsCache.workspaceEntries) > 0 {
		for _, entry := range wsCache.workspaceEntries {
			if !entry.IsDir() {
				continue
			}

			// Check if already in active list
			// Active session IDs have format "workspace [beads-id]", workspace names don't
			alreadyIn := false
			workspaceName := entry.Name()
			for _, a := range agents {
				if a.ID == workspaceName || strings.HasPrefix(a.ID, workspaceName+" ") {
					alreadyIn = true
					break
				}
			}

			if alreadyIn {
				continue
			}

			// Use the lookup method for multi-project support
			workspacePath := wsCache.lookupWorkspacePathByEntry(entry.Name())
			synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
			hasSynthesis := false

			// Check if SYNTHESIS.md exists (indicates full-tier completion)
			if _, err := os.Stat(synthesisPath); err == nil {
				hasSynthesis = true
			}

			// Only add workspaces that have SYNTHESIS.md for now
			// Light-tier completions will be detected via Phase: Complete in beads comments
			if !hasSynthesis {
				// For light-tier, check if there's a SPAWN_CONTEXT.md (indicates it's a valid spawn)
				spawnContextPath := filepath.Join(workspacePath, "SPAWN_CONTEXT.md")
				if _, err := os.Stat(spawnContextPath); err != nil {
					continue // Not a valid spawn workspace
				}
			}

			agent := AgentAPIResponse{
				ID:     entry.Name(),
				Status: "completed",
			}

			// Set updated_at from workspace name date suffix or file modification time
			// This ensures proper sorting in archive section
			if parsedDate := extractDateFromWorkspaceName(entry.Name()); !parsedDate.IsZero() {
				agent.UpdatedAt = parsedDate.Format(time.RFC3339)
			} else if hasSynthesis {
				// Fallback to file modification time of SYNTHESIS.md
				if info, err := os.Stat(synthesisPath); err == nil {
					agent.UpdatedAt = info.ModTime().Format(time.RFC3339)
				}
			} else {
				// For light-tier, use SPAWN_CONTEXT.md modification time
				spawnContextPath := filepath.Join(workspacePath, "SPAWN_CONTEXT.md")
				if info, err := os.Stat(spawnContextPath); err == nil {
					agent.UpdatedAt = info.ModTime().Format(time.RFC3339)
				}
			}

			// Read session ID from workspace
			if sessionID := spawn.ReadSessionID(workspacePath); sessionID != "" {
				agent.SessionID = sessionID
			}

			// Read spawn time from workspace
			if spawnTime := spawn.ReadSpawnTime(workspacePath); !spawnTime.IsZero() {
				agent.SpawnedAt = spawnTime.Format(time.RFC3339)
			}

			// Parse synthesis (only for full-tier)
			if hasSynthesis {
				if synthesis, err := verify.ParseSynthesis(workspacePath); err == nil {
					agent.Synthesis = &SynthesisResponse{
						TLDR:           synthesis.TLDR,
						Outcome:        synthesis.Outcome,
						Recommendation: synthesis.Recommendation,
						DeltaSummary:   summarizeDelta(synthesis.Delta),
						NextActions:    synthesis.NextActions,
					}
				}
				// Read raw SYNTHESIS.md content for inline rendering in dashboard
				synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
				if content, err := os.ReadFile(synthesisPath); err == nil {
					agent.SynthesisContent = string(content)
				}
			}

			// Extract beadsID from workspace SPAWN_CONTEXT.md (more reliable than parsing name)
			agent.BeadsID = extractBeadsIDFromWorkspace(workspacePath)
			// Fallback to extracting from workspace name if SPAWN_CONTEXT.md doesn't have it
			if agent.BeadsID == "" {
				agent.BeadsID = extractBeadsIDFromTitle(entry.Name())
			}
			agent.Skill = extractSkillFromTitle(entry.Name())
			// Extract Project from beadsID for proper filtering
			// Without this, completed workspaces have project:null and won't match project filters.
			// See .kb/investigations/2026-01-29-inv-dashboard-follow-mode-doesn-show.md
			agent.Project = extractProjectFromBeadsID(agent.BeadsID)

			// NOTE: We intentionally DON'T extract PROJECT_DIR or fetch beads data for completed workspaces.
			// Completed workspaces have already done their work - they don't need phase updates.
			// The close_reason is nice-to-have but not worth spawning bd processes.
			// This optimization prevents CPU spikes from fetching beads data for 600+ historical workspaces.

			agents = append(agents, agent)
		}
	}

	// Batch fetch beads data (phase from comments, task from issues, close_reason for completed)
	// This is the same pattern used by orch status for efficiency.
	// Uses TTL cache to prevent CPU spikes from spawning 20+ bd processes per request.
	if len(beadsIDsToFetch) > 0 {
		// Fetch all open issues in one call (cached with TTL)
		openIssues, _ := globalBeadsCache.getOpenIssues()

		// Batch fetch all issues (including closed) for close_reason (cached with TTL)
		// Uses bd show which works for any issue status
		allIssues, _ := globalBeadsCache.getAllIssues(beadsIDsToFetch)

		// Batch fetch comments for all beads IDs (cached with TTL)
		// Use project-aware batch fetch for cross-project agent visibility
		commentsMap := globalBeadsCache.getComments(beadsIDsToFetch, beadsProjectDirs)

		// Build investigation directory cache ONCE before the agent loop.
		// This prevents O(n²) behavior: without this, discoverInvestigationPath would call
		// os.ReadDir() 2-3 times per agent, scanning 500+ files each time.
		// With 300+ agents, that's 300 × 500 × 2 = 300,000+ file comparisons.
		// The cache reduces this to a single ReadDir() call per directory.
		uniqueProjectDirs := make([]string, 0, len(beadsProjectDirs))
		seenDirs := make(map[string]bool)
		for _, dir := range beadsProjectDirs {
			if dir != "" && !seenDirs[dir] {
				seenDirs[dir] = true
				uniqueProjectDirs = append(uniqueProjectDirs, dir)
			}
		}
		invDirCache := buildInvestigationDirCache(uniqueProjectDirs)

		// Populate phase, task, close_reason, and status for each agent using Priority Cascade model.
		// See .kb/investigations/2026-01-04-design-dashboard-agent-status-model.md for design.
		for i := range agents {
			if agents[i].BeadsID == "" {
				continue
			}

			// Get task from open issue title first
			if issue, ok := openIssues[agents[i].BeadsID]; ok {
				agents[i].Task = truncate(issue.Title, 60)
				agents[i].BeadsLabels = issue.Labels
			}

			// If not in open issues, try all issues (for closed ones)
			if agents[i].Task == "" {
				if issue, ok := allIssues[agents[i].BeadsID]; ok {
					agents[i].Task = truncate(issue.Title, 60)
					agents[i].BeadsLabels = issue.Labels
					// For completed agents without synthesis, use close_reason as fallback
					if agents[i].Synthesis == nil && issue.CloseReason != "" {
						agents[i].CloseReason = issue.CloseReason
					}
				}
			}

			// Gather completion signals for Priority Cascade model
			issueClosed := false
			phaseComplete := false

			// Check if beads issue is closed (Priority 1)
			if issue, ok := allIssues[agents[i].BeadsID]; ok {
				issueClosed = strings.EqualFold(issue.Status, "closed")
				// Capture close_reason if available
				if issueClosed && issue.CloseReason != "" && agents[i].CloseReason == "" {
					agents[i].CloseReason = issue.CloseReason
				}
			}

			// Get phase from comments (Priority 2)
			if comments, ok := commentsMap[agents[i].BeadsID]; ok {
				phaseStatus := verify.ParsePhaseFromComments(comments)
				if phaseStatus.Found {
					agents[i].Phase = phaseStatus.Phase
					phaseComplete = strings.EqualFold(phaseStatus.Phase, "Complete")

					// Stalled detection: if active agent has same phase for 15+ minutes
					// Advisory only - surfaces in Needs Attention but doesn't auto-abandon.
					// See .kb/investigations/2026-01-08-inv-design-stalled-agent-detection-agents.md
					if agents[i].Status == "active" && phaseStatus.PhaseReportedAt != nil {
						timeSincePhase := now.Sub(*phaseStatus.PhaseReportedAt)
						if timeSincePhase > stalledThreshold {
							agents[i].IsStalled = true
						}
					}
				}
				// Extract investigation_path from comments for investigation tab rendering
				if investigationPath := verify.ParseInvestigationPathFromComments(comments); investigationPath != "" {
					agents[i].InvestigationPath = investigationPath
				}
			}

			// Populate project_dir from beadsProjectDirs lookup (for workspace path construction)
			// Track if we got a reliable project dir from workspace cache (vs using session directory)
			hasReliableProjectDir := false
			if agentProjectDir, ok := beadsProjectDirs[agents[i].BeadsID]; ok {
				agents[i].ProjectDir = agentProjectDir
				hasReliableProjectDir = true
			}

			// Auto-discover investigation path if not provided via beads comment
			// This uses a fallback chain: .kb/investigations/ matching -> workspace .md files
			// Uses invDirCache to avoid O(n²) directory scanning (built once before this loop)
			// IMPORTANT: Only auto-discover if we have a reliable project dir from workspace cache.
			// The session directory (s.Directory) may be the orchestrator's cwd due to OpenCode --attach bug,
			// which would cause us to search the wrong project's .kb/investigations/ for cross-project agents.
			if agents[i].InvestigationPath == "" && hasReliableProjectDir {
				workspaceName := agents[i].ID
				if idx := strings.Index(workspaceName, " ["); idx != -1 {
					workspaceName = workspaceName[:idx]
				}
				discoveredPath := discoverInvestigationPath(workspaceName, agents[i].BeadsID, agents[i].ProjectDir, invDirCache)
				if discoveredPath != "" {
					agents[i].InvestigationPath = discoveredPath
				}
			}

			// Read investigation file content for inline rendering in dashboard
			if agents[i].InvestigationPath != "" {
				if content, err := os.ReadFile(agents[i].InvestigationPath); err == nil {
					agents[i].InvestigationContent = string(content)
				}
			}

			// Get workspace path for SYNTHESIS.md check (Priority 3)
			workspacePath := wsCache.lookupWorkspace(agents[i].BeadsID)
			// Fallback: For untracked agents, try looking up by workspace name from session title
			if workspacePath == "" && agents[i].ID != "" {
				workspaceName := agents[i].ID
				if idx := strings.Index(workspaceName, " ["); idx != -1 {
					workspaceName = workspaceName[:idx]
				}
				workspacePath = wsCache.lookupWorkspacePathByEntry(workspaceName)
			}

			// Use Priority Cascade to determine final status
			// Priority order: issueClosed > phaseComplete > SYNTHESIS.md > sessionStatus
			agents[i].Status = determineAgentStatus(issueClosed, phaseComplete, workspacePath, agents[i].Status)

			// For completed agents, also check close_reason if synthesis is null
			if agents[i].Status == "completed" && agents[i].Synthesis == nil && agents[i].CloseReason == "" {
				if issue, ok := allIssues[agents[i].BeadsID]; ok && issue.CloseReason != "" {
					agents[i].CloseReason = issue.CloseReason
				}
			}

			// Read synthesis content for active agents that have a workspace with SYNTHESIS.md
			// (Completed workspaces already have this populated above in the workspace scan)
			if workspacePath != "" && agents[i].SynthesisContent == "" {
				synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
				if content, err := os.ReadFile(synthesisPath); err == nil {
					agents[i].SynthesisContent = string(content)
					// Also parse synthesis if not already parsed
					if agents[i].Synthesis == nil {
						if synthesis, err := verify.ParseSynthesis(workspacePath); err == nil {
							agents[i].Synthesis = &SynthesisResponse{
								TLDR:           synthesis.TLDR,
								Outcome:        synthesis.Outcome,
								Recommendation: synthesis.Recommendation,
								DeltaSummary:   summarizeDelta(synthesis.Delta),
								NextActions:    synthesis.NextActions,
							}
						}
					}
				}
			}
		}

		// Fetch gap analysis from spawn events for each agent
		gapAnalysisMap := getGapAnalysisFromEvents(beadsIDsToFetch)
		for i := range agents {
			if agents[i].BeadsID == "" {
				continue
			}
			if gapData, ok := gapAnalysisMap[agents[i].BeadsID]; ok {
				agents[i].GapAnalysis = gapData
			}
		}

		// Post-Phase filtering: remove agents using two-threshold ghost filtering
		// Uses IsVisibleByDefault to determine if agent should be shown
		// This ensures Phase: Complete agents are always visible
		filtered := make([]AgentAPIResponse, 0, len(agents))
		for _, agentItem := range agents {
			// Determine status for filtering ("running" vs "idle")
			status := agentItem.Status
			if status != "active" && status != "idle" && status != "completed" {
				// Map other statuses to idle for filtering purposes
				status = "idle"
			} else if status == "active" {
				status = "running"
			}

			// Get last activity time
			lastActivity := time.Time{}
			if agentItem.LastActivityAt != "" {
				lastActivity, _ = time.Parse(time.RFC3339, agentItem.LastActivityAt)
			}

			// Apply IsVisibleByDefault filter only if agent was marked as pending
			if pendingFilterByBeadsID[agentItem.BeadsID] {
				if !agent.IsVisibleByDefault(status, lastActivity, agentItem.Phase) {
					continue // Ghost agent - filter out
				}
			}

			filtered = append(filtered, agentItem)
		}
		agents = filtered
	}

	// NOTE: The duplicate SYNTHESIS.md check that was here has been removed.
	// All agents are now included in beadsIDsToFetch, so the Priority Cascade
	// in determineAgentStatus() handles all status determination in one place.
	// See .kb/investigations/2026-01-04-design-dashboard-agent-status-model.md

	// Fetch token usage, last activity, and model for agents with valid session IDs
	// Parallelized to avoid sequential HTTP calls causing ~20s delays with 200+ agents.
	// Uses goroutines with semaphore to limit concurrent requests.
	// Tokens, activity, and model are extracted from the same GetMessages call for efficiency.
	type sessionResult struct {
		index    int
		tokens   *opencode.TokenStats
		activity *opencode.LastActivity
		model    string
	}
	resultChan := make(chan sessionResult, len(agents))

	// Limit concurrent HTTP requests to avoid overwhelming the OpenCode server
	const maxConcurrent = 20
	sem := make(chan struct{}, maxConcurrent)

	var wg sync.WaitGroup
	for i := range agents {
		// Skip agents without session ID, completed agents, or idle agents.
		// Token data is static for completed agents, and idle agents are unlikely to have changed.
		// This prevents CPU spikes from making HTTP calls to OpenCode for 300+ inactive sessions.
		if agents[i].SessionID == "" || agents[i].Status == "completed" || agents[i].Status == "idle" {
			continue
		}

		wg.Add(1)
		go func(idx int, sessionID string) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire semaphore
			defer func() { <-sem }() // Release semaphore

			// Fetch messages once and extract tokens, activity, and model
			messages, err := client.GetMessages(sessionID)
			if err != nil || len(messages) == 0 {
				return
			}

			result := sessionResult{index: idx}

			// Extract tokens
			tokenStats := opencode.AggregateTokens(messages)
			result.tokens = &tokenStats

			// Extract last activity from messages
			result.activity = extractLastActivityFromMessages(messages)

			// Extract model from most recent assistant message
			for j := len(messages) - 1; j >= 0; j-- {
				if messages[j].Info.Role == "assistant" && messages[j].Info.ModelID != "" {
					result.model = messages[j].Info.ModelID
					break
				}
			}

			resultChan <- result
		}(i, agents[i].SessionID)
	}

	// Wait for all goroutines to complete, then close channel
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	for result := range resultChan {
		if result.tokens != nil {
			agents[result.index].Tokens = result.tokens
		}
		if result.activity != nil {
			agents[result.index].CurrentActivity = result.activity.Text
			agents[result.index].LastActivityAt = time.Unix(result.activity.Timestamp/1000, 0).Format(time.RFC3339)
		}
		if result.model != "" {
			agents[result.index].Model = result.model
		}
	}

	// Apply time and project filters for non-session agents.
	// NOTE: OpenCode sessions are filtered EARLY (after ListSessions call) for performance.
	// This late filter catches:
	// 1. Tmux-only agents (not in OpenCode sessions)
	// 2. Completed workspaces (timestamps from workspace metadata, not session)
	if sinceDuration > 0 || len(projectFilterParam) > 0 {
		filtered := make([]AgentAPIResponse, 0, len(agents))
		for _, agent := range agents {
			// Time filter: check updated_at or spawned_at
			if sinceDuration > 0 {
				var agentTime time.Time
				if agent.UpdatedAt != "" {
					agentTime, _ = time.Parse(time.RFC3339, agent.UpdatedAt)
				} else if agent.SpawnedAt != "" {
					agentTime, _ = time.Parse(time.RFC3339, agent.SpawnedAt)
				}
				if !agentTime.IsZero() && !filterByTime(agentTime, sinceDuration) {
					continue
				}
			}

			// Project filter: check project identifiers against all filters.
			// We match against BOTH beads prefix AND directory name because they can differ:
			// - Beads prefix: "ok" (from beads ID "ok-765")
			// - Directory name: "orch-knowledge" (from ProjectDir "/Users/.../orch-knowledge")
			// See .kb/investigations/2026-01-27-inv-investigate-untracked-agents-no-track.md
			if len(projectFilterParam) > 0 && !matchAgentProject(agent.Project, agent.ProjectDir, projectFilterParam) {
				continue
			}

			filtered = append(filtered, agent)
		}
		agents = filtered
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(agents); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode agents: %v", err), http.StatusInternalServerError)
		return
	}
}

// getProjectAPIPort returns the allocated API port for the current project.
// Returns 0 if no allocation exists or on error.
func getProjectAPIPort() int {
	projectDir, err := os.Getwd()
	if err != nil {
		return 0
	}
	projectName := filepath.Base(projectDir)

	registry, err := port.New("")
	if err != nil {
		return 0
	}

	alloc := registry.Find(projectName, "api")
	if alloc == nil {
		return 0
	}

	return alloc.Port
}

// checkWorkspaceSynthesis checks if a workspace has a non-empty SYNTHESIS.md file.
// This is used to detect completion for untracked agents (--no-track) where
// there's no beads issue to check Phase: Complete.
func checkWorkspaceSynthesis(workspacePath string) bool {
	if workspacePath == "" {
		return false
	}
	synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
	info, err := os.Stat(synthesisPath)
	if err != nil {
		return false
	}
	// SYNTHESIS.md must exist and be non-empty
	return info.Size() > 0
}

// determineAgentStatus implements the Priority Cascade model for agent status.
// This is the single source of truth for determining agent status.
//
// Priority order (highest to lowest):
//  1. Beads issue closed → "completed" (orchestrator verified completion)
//  2. Phase: Complete reported AND session dead → "awaiting-cleanup" (agent done, needs orch complete)
//  3. Phase: Complete reported → "completed" (agent declared done, still has active session)
//  4. SYNTHESIS.md exists AND session dead → "awaiting-cleanup" (artifact proves completion, needs cleanup)
//  5. SYNTHESIS.md exists → "completed" (artifact proves completion)
//  6. Session activity → sessionStatus ("active", "idle", or "dead")
//
// The "awaiting-cleanup" status distinguishes completed-but-orphaned agents from crashed agents.
// This helps orchestrators prioritize: awaiting-cleanup needs orch complete, dead needs investigation.
// See .kb/investigations/2026-01-04-design-dashboard-agent-status-model.md for design rationale.
// See .kb/investigations/2026-01-08-inv-handle-multiple-agents-same-beads.md for awaiting-cleanup addition.
func determineAgentStatus(issueClosed bool, phaseComplete bool, workspacePath string, sessionStatus string) string {
	// Priority 1: Beads issue closed → completed (orchestrator verified completion)
	if issueClosed {
		return "completed"
	}

	hasSynthesis := checkWorkspaceSynthesis(workspacePath)
	isDead := sessionStatus == "dead"

	// Priority 2: Phase: Complete reported AND session dead → awaiting-cleanup
	// Agent finished work and reported completion, but orchestrator hasn't run orch complete.
	// This is NOT an error state - the agent did its job, just needs cleanup.
	if phaseComplete && isDead {
		return "awaiting-cleanup"
	}

	// Priority 3: Phase: Complete reported (session still active/idle) → completed
	if phaseComplete {
		return "completed"
	}

	// Priority 4: SYNTHESIS.md exists AND session dead → awaiting-cleanup
	// Agent wrote synthesis artifact (proof of completion) but session is dead.
	// Similar to Phase: Complete case - needs cleanup, not investigation.
	if hasSynthesis && isDead {
		return "awaiting-cleanup"
	}

	// Priority 5: SYNTHESIS.md exists (session still active/idle) → completed
	if hasSynthesis {
		return "completed"
	}

	// Priority 6: Session activity (fallback)
	// "dead" agents without completion signals truly need attention (crashed/stuck)
	return sessionStatus
}

// getGapAnalysisFromEvents reads spawn events and extracts gap analysis data for given beads IDs.
// Returns a map of beads ID -> GapAPIResponse.
func getGapAnalysisFromEvents(beadsIDs []string) map[string]*GapAPIResponse {
	result := make(map[string]*GapAPIResponse)
	if len(beadsIDs) == 0 {
		return result
	}

	// Build a set of beads IDs for fast lookup
	beadsIDSet := make(map[string]bool)
	for _, id := range beadsIDs {
		beadsIDSet[id] = true
	}

	// Read events file
	logPath := events.DefaultLogPath()
	file, err := os.Open(logPath)
	if err != nil {
		return result
	}
	defer file.Close()

	// Scan events for spawn events matching our beads IDs
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var event events.Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}

		// Only process spawn events
		if event.Type != "session.spawned" {
			continue
		}

		// Check if this event is for one of our beads IDs
		beadsID, ok := event.Data["beads_id"].(string)
		if !ok || !beadsIDSet[beadsID] {
			continue
		}

		// Already have gap analysis for this beads ID? Skip (we want the most recent)
		// Since we read chronologically, later entries overwrite earlier ones
		if _, exists := result[beadsID]; exists {
			// We could skip, but let's allow overwrites for resumptions
		}

		// Extract gap analysis data from event
		gapData := extractGapAnalysisFromEvent(event.Data)
		if gapData != nil {
			result[beadsID] = gapData
		}
	}

	return result
}

// extractGapAnalysisFromEvent extracts gap analysis data from a spawn event's data map.
func extractGapAnalysisFromEvent(data map[string]interface{}) *GapAPIResponse {
	// Check if gap data exists
	hasGaps, ok := data["gap_has_gaps"].(bool)
	if !ok {
		return nil
	}

	contextQuality := 0
	if cq, ok := data["gap_context_quality"].(float64); ok {
		contextQuality = int(cq)
	}

	shouldWarn := false
	if sw, ok := data["gap_should_warn"].(bool); ok {
		shouldWarn = sw
	}

	matchCount := 0
	if mc, ok := data["gap_match_total"].(float64); ok {
		matchCount = int(mc)
	}

	constraints := 0
	if c, ok := data["gap_match_constraints"].(float64); ok {
		constraints = int(c)
	}

	decisions := 0
	if d, ok := data["gap_match_decisions"].(float64); ok {
		decisions = int(d)
	}

	investigations := 0
	if i, ok := data["gap_match_investigations"].(float64); ok {
		investigations = int(i)
	}

	return &GapAPIResponse{
		HasGaps:        hasGaps,
		ContextQuality: contextQuality,
		ShouldWarn:     shouldWarn,
		MatchCount:     matchCount,
		Constraints:    constraints,
		Decisions:      decisions,
		Investigations: investigations,
	}
}

// extractLastActivityFromMessages extracts the last meaningful activity from messages.
// It looks for the most recent assistant message and extracts a summary of what
// the agent is doing (tool use, text generation, etc.).
// Returns nil if no activity can be extracted.
func extractLastActivityFromMessages(messages []opencode.Message) *opencode.LastActivity {
	if len(messages) == 0 {
		return nil
	}

	// Find the last assistant message (most relevant for activity)
	var lastAssistantMsg *opencode.Message
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Info.Role == "assistant" {
			lastAssistantMsg = &messages[i]
			break
		}
	}

	if lastAssistantMsg == nil {
		return nil
	}

	// Extract activity from message parts
	// Priority: tool invocation > text > reasoning
	var activityText string
	for _, part := range lastAssistantMsg.Parts {
		switch part.Type {
		case "tool-invocation", "tool":
			// Tool use is the most informative activity
			activityText = "Using tool"
			if part.Text != "" {
				// Truncate tool text for display
				toolText := part.Text
				if len(toolText) > 40 {
					toolText = toolText[:40] + "..."
				}
				activityText = "Using tool: " + toolText
			}
		case "text":
			if part.Text != "" && activityText == "" {
				// Truncate long text
				text := part.Text
				if len(text) > 80 {
					// Find last space before 80 chars
					cutoff := 77
					for i := cutoff; i > 0; i-- {
						if text[i] == ' ' {
							cutoff = i
							break
						}
					}
					text = text[:cutoff] + "..."
				}
				activityText = text
			}
		case "reasoning":
			if activityText == "" {
				activityText = "Thinking..."
			}
		}
	}

	if activityText == "" {
		return nil
	}

	// Use message completion time if available, otherwise created time
	timestamp := lastAssistantMsg.Info.Time.Completed
	if timestamp == 0 {
		timestamp = lastAssistantMsg.Info.Time.Created
	}

	return &opencode.LastActivity{
		Text:      activityText,
		Timestamp: timestamp,
	}
}

// handleCacheInvalidate clears all dashboard caches to force fresh data on next request.
// This is called by orch complete to ensure the dashboard shows updated agent status.
// Without this, the TTL cache holds stale "active" status after agents complete.
func handleCacheInvalidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Invalidate beads cache (open issues, all issues, comments)
	if globalBeadsCache != nil {
		globalBeadsCache.invalidate()
	}

	// Invalidate beads stats cache (stats, ready issues)
	// Empty string clears all project caches
	if globalBeadsStatsCache != nil {
		globalBeadsStatsCache.invalidate("")
	}

	// Invalidate kb health cache (knowledge hygiene signals)
	if globalKBHealthCache != nil {
		globalKBHealthCache.invalidate()
	}

	// Invalidate workspace cache (workspace metadata)
	globalWorkspaceCacheInstance.invalidate()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": "Cache invalidated",
	})
}

// MessagePartResponse is the JSON structure for message parts in the activity feed.
// This mirrors the SSEEvent structure used by the frontend for real-time events,
// enabling seamless merging of historical API data with live SSE data.
type MessagePartResponse struct {
	ID         string                `json:"id"`
	Type       string                `json:"type"` // Always "message.part" to match SSE event type
	Properties MessagePartProperties `json:"properties"`
	Timestamp  int64                 `json:"timestamp,omitempty"`
}

// MessagePartProperties contains the part data in SSE-compatible format.
type MessagePartProperties struct {
	SessionID string      `json:"sessionID"`
	MessageID string      `json:"messageID"`
	Part      PartDetails `json:"part"`
}

// PartDetails contains the actual part content.
type PartDetails struct {
	ID        string     `json:"id"`
	Type      string     `json:"type"` // "text", "tool", "reasoning", "step-start", "step-finish"
	Text      string     `json:"text,omitempty"`
	SessionID string     `json:"sessionID"`
	Tool      string     `json:"tool,omitempty"`
	State     *ToolState `json:"state,omitempty"`
}

// ToolState contains tool invocation state for tool parts.
type ToolState struct {
	Title  string                 `json:"title,omitempty"`
	Status string                 `json:"status,omitempty"`
	Input  map[string]interface{} `json:"input,omitempty"`
	Output string                 `json:"output,omitempty"`
}

// ActivityJSONFile is the structure of ACTIVITY.json exported on agent completion.
// This file serves as archival storage for session activity, loaded when
// the OpenCode session no longer exists (deleted/cleaned up).
type ActivityJSONFile struct {
	Version    int                   `json:"version"`
	SessionID  string                `json:"session_id"`
	ExportedAt string                `json:"exported_at"`
	Events     []MessagePartResponse `json:"events"`
}

// findWorkspaceBySessionID searches for a workspace directory with a matching .session_id file.
// This is used to find archived activity when the OpenCode session has been deleted.
// Returns the workspace path if found, or empty string if not found.
func findWorkspaceBySessionID(projectDir, sessionID string) string {
	workspaceDir := filepath.Join(projectDir, ".orch", "workspace")
	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		workspacePath := filepath.Join(workspaceDir, entry.Name())
		storedSessionID := spawn.ReadSessionID(workspacePath)
		if storedSessionID == sessionID {
			return workspacePath
		}
	}

	// Also check archived workspaces
	archivedDir := filepath.Join(workspaceDir, "archived")
	archivedEntries, err := os.ReadDir(archivedDir)
	if err != nil {
		return ""
	}

	for _, entry := range archivedEntries {
		if !entry.IsDir() {
			continue
		}
		workspacePath := filepath.Join(archivedDir, entry.Name())
		storedSessionID := spawn.ReadSessionID(workspacePath)
		if storedSessionID == sessionID {
			return workspacePath
		}
	}

	return ""
}

// loadActivityFromWorkspace loads activity events from ACTIVITY.json in a workspace.
// Returns the events if found and valid, or nil if not available.
func loadActivityFromWorkspace(workspacePath string) []MessagePartResponse {
	activityPath := filepath.Join(workspacePath, "ACTIVITY.json")
	data, err := os.ReadFile(activityPath)
	if err != nil {
		return nil
	}

	var activityFile ActivityJSONFile
	if err := json.Unmarshal(data, &activityFile); err != nil {
		return nil
	}

	return activityFile.Events
}

// handleSessionMessages proxies OpenCode's /session/:sessionID/message API.
// This endpoint enables the dashboard to fetch historical session messages
// for the activity feed, complementing real-time SSE updates.
//
// GET /api/session/:sessionID/messages
// Response: Array of MessagePartResponse in SSE-compatible format
func handleSessionMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract sessionID from URL path: /api/session/{sessionID}/messages
	path := r.URL.Path
	prefix := "/api/session/"
	suffix := "/messages"

	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		http.Error(w, "Invalid path format. Expected: /api/session/{sessionID}/messages", http.StatusBadRequest)
		return
	}

	sessionID := path[len(prefix) : len(path)-len(suffix)]
	if sessionID == "" {
		http.Error(w, "Session ID is required", http.StatusBadRequest)
		return
	}

	client := opencode.NewClient(serverURL)
	messages, err := client.GetMessages(sessionID)
	if err != nil {
		// OpenCode API failed (session may be deleted/cleaned up).
		// Fall back to ACTIVITY.json if available in the workspace.
		projectDir, _ := os.Getwd()
		workspacePath := findWorkspaceBySessionID(projectDir, sessionID)
		if workspacePath != "" {
			if events := loadActivityFromWorkspace(workspacePath); events != nil {
				// Successfully loaded from ACTIVITY.json
				w.Header().Set("Content-Type", "application/json")
				if encErr := json.NewEncoder(w).Encode(events); encErr != nil {
					http.Error(w, fmt.Sprintf("Failed to encode events: %v", encErr), http.StatusInternalServerError)
				}
				return
			}
		}
		// No fallback available, return original error
		http.Error(w, fmt.Sprintf("Failed to fetch messages: %v", err), http.StatusInternalServerError)
		return
	}

	// Transform OpenCode messages to SSE-compatible format for the activity feed.
	// This enables seamless merging with real-time SSE events in the frontend.
	var parts []MessagePartResponse
	for _, msg := range messages {
		for _, part := range msg.Parts {
			// Map OpenCode part types to activity feed types
			partType := part.Type
			switch part.Type {
			case "tool-invocation":
				partType = "tool"
			}

			// Only include types that the activity feed displays
			if partType != "text" && partType != "tool" && partType != "reasoning" &&
				partType != "step-start" && partType != "step-finish" {
				continue
			}

			// Transform tool state if present
			var state *ToolState
			if part.State != nil {
				state = &ToolState{
					Title:  part.State.Title,
					Status: part.State.Status,
					Input:  part.State.Input,
					Output: part.State.Output,
				}
			}

			response := MessagePartResponse{
				ID:   part.ID,
				Type: "message.part", // Match SSE event type
				Properties: MessagePartProperties{
					SessionID: sessionID,
					MessageID: msg.Info.ID,
					Part: PartDetails{
						ID:        part.ID,
						Type:      partType,
						Text:      part.Text,
						SessionID: sessionID,
						Tool:      part.Tool, // Add tool name for tool invocations
						State:     state,     // Add tool state (input/output)
					},
				},
				Timestamp: msg.Info.Time.Created,
			}
			parts = append(parts, response)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(parts); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode messages: %v", err), http.StatusInternalServerError)
		return
	}
}

// getWorkspaceLastActivity returns the most recent file modification time in a workspace.
// This is used for activity detection in tmux agents (Claude CLI escape hatch) where
// we don't have session timestamps from OpenCode API.
// Returns zero time if workspace doesn't exist or has no activity files.
func getWorkspaceLastActivity(workspacePath string) time.Time {
	if workspacePath == "" {
		return time.Time{}
	}

	// Files that indicate agent activity (ordered by relevance)
	// Investigation files and SYNTHESIS.md are the most relevant indicators
	activityFiles := []string{
		"SYNTHESIS.md",
		"SPAWN_CONTEXT.md",
		".session_id",
		".spawn_time",
	}

	var lastMod time.Time

	// Check known activity files first
	for _, filename := range activityFiles {
		filePath := filepath.Join(workspacePath, filename)
		info, err := os.Stat(filePath)
		if err != nil {
			continue
		}
		if info.ModTime().After(lastMod) {
			lastMod = info.ModTime()
		}
	}

	// Also check for any .md files in the workspace (investigation files)
	entries, err := os.ReadDir(workspacePath)
	if err != nil {
		return lastMod
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		// Check markdown files (investigation files, notes, etc.)
		if strings.HasSuffix(entry.Name(), ".md") {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			if info.ModTime().After(lastMod) {
				lastMod = info.ModTime()
			}
		}
	}

	return lastMod
}
