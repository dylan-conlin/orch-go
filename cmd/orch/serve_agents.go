package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/agent"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
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
	IsUntracked          bool                 `json:"is_untracked,omitempty"`  // True if agent was spawned with --no-track (synthetic beads ID)
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

// Extracted to sub-domain files:
// - serve_agents_status.go: determineDeathReason, determineAgentStatus, checkWorkspaceSynthesis,
//   getWorkspaceLastActivity, getProjectAPIPort
// - serve_agents_activity.go: handleSessionMessages, extractLastActivityFromMessages,
//   findWorkspaceBySessionID, loadActivityFromWorkspace, MessagePartResponse types
// - serve_agents_gap.go: getGapAnalysisFromEvents, extractGapAnalysisFromEvent
// - serve_agents_investigation.go: investigationDirCache, buildInvestigationDirCache, discoverInvestigationPath

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

		// Mark untracked agents (--no-track spawns with synthetic beads IDs)
		// These have no real beads issue, so phase/task/completion will always be empty.
		if isUntrackedBeadsID(agent.BeadsID) {
			agent.IsUntracked = true
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

		// Collect beads ID for batch fetch - only for NON-STALE, TRACKED agents with beads ID.
		// Stale agents (older than beadsFetchThreshold) are included in response but
		// skip beads fetch for performance optimization.
		// Untracked agents have synthetic beads IDs not in the database - skip them too.
		// See .kb/investigations/2026-01-04-design-dashboard-agent-status-model.md
		if agent.BeadsID != "" && !seenBeadsIDs[agent.BeadsID] && !isStale && !agent.IsUntracked {
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
					ID:          win.Name,
					BeadsID:     beadsID,
					Skill:       skill,
					Project:     project,
					Status:      "active",
					Window:      win.Target,
					IsUntracked: isUntrackedBeadsID(beadsID),
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
