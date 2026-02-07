package main

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// agentCollectionContext holds shared state used across the data collection pipeline.
// This replaces the many local variables that were scattered through handleAgents.
type agentCollectionContext struct {
	// Agents accumulated from all sources
	agents []AgentAPIResponse

	// Tracking maps
	seenBeadsIDs       map[string]bool
	beadsIDsToFetch    []string
	beadsProjectDirs   map[string]string // beadsID -> projectDir from workspace
	pendingFilterByIDs map[string]bool   // beadsIDs pending ghost filtering
	seenTitles         map[string]int    // title -> index in agents slice

	// Thresholds
	activeThreshold     time.Duration
	displayThreshold    time.Duration
	deadThreshold       time.Duration
	stalledThreshold    time.Duration
	beadsFetchThreshold time.Duration

	// Dependencies
	wsCache *workspaceCache
	client  *opencode.Client
	now     time.Time
}

// newAgentCollectionContext creates a new context with default thresholds and dependencies.
func newAgentCollectionContext(client *opencode.Client, wsCache *workspaceCache, sinceDuration time.Duration) *agentCollectionContext {
	// Active threshold (10min): determines "running" vs "idle" status
	activeThreshold := 10 * time.Minute
	// Display threshold (4h): filters ghosts from default view (unless Phase: Complete)
	displayThreshold := 4 * time.Hour
	// Dead threshold: if no activity for 3 minutes, session is dead.
	// Agents are constantly reading, editing, running commands - 3 min silence = dead.
	deadThreshold := 3 * time.Minute
	// Stalled threshold: if same phase for 15+ minutes, agent may be stuck.
	// Advisory only - surfaces in Needs Attention but doesn't auto-abandon.
	stalledThreshold := 15 * time.Minute

	// beadsFetchThreshold limits which sessions we fetch beads data for.
	// Sessions older than this are excluded from beads lookups entirely.
	// MAJOR optimization: with 600+ sessions but only ~6 active,
	// fetching beads for all would require 400+ RPC calls = 3+ seconds.
	beadsFetchThreshold := 2 * time.Hour
	if sinceDuration > beadsFetchThreshold {
		beadsFetchThreshold = sinceDuration
	} else if sinceDuration == 0 {
		// "all" requested
		beadsFetchThreshold = 365 * 24 * time.Hour
	}

	return &agentCollectionContext{
		agents:              []AgentAPIResponse{}, // Initialize as empty slice to return [] instead of null
		seenBeadsIDs:        make(map[string]bool),
		beadsProjectDirs:    make(map[string]string),
		pendingFilterByIDs:  make(map[string]bool),
		seenTitles:          make(map[string]int),
		activeThreshold:     activeThreshold,
		displayThreshold:    displayThreshold,
		deadThreshold:       deadThreshold,
		stalledThreshold:    stalledThreshold,
		beadsFetchThreshold: beadsFetchThreshold,
		wsCache:             wsCache,
		client:              client,
		now:                 time.Now(),
	}
}

// collectBeadsID adds a beads ID for batch fetching if not already seen and not stale/untracked.
func (ctx *agentCollectionContext) collectBeadsID(beadsID string, isStale, isUntracked bool) {
	if beadsID == "" || ctx.seenBeadsIDs[beadsID] || isStale || isUntracked {
		return
	}
	ctx.beadsIDsToFetch = append(ctx.beadsIDsToFetch, beadsID)
	ctx.seenBeadsIDs[beadsID] = true
}

// collectProjectDir records a project directory for a beads ID from workspace cache.
func (ctx *agentCollectionContext) collectProjectDir(beadsID string) {
	if beadsID == "" {
		return
	}
	if agentProjectDir := ctx.wsCache.lookupProjectDir(beadsID); agentProjectDir != "" {
		ctx.beadsProjectDirs[beadsID] = agentProjectDir
	}
}

// collectOpenCodeSessions processes OpenCode sessions into agent entries.
// Determines status (active/idle/dead), extracts beads ID and metadata,
// deduplicates by title, and filters stale sessions.
func (ctx *agentCollectionContext) collectOpenCodeSessions(sessions []opencode.Session) {
	for _, s := range sessions {
		createdAt := time.Unix(s.Time.Created/1000, 0)
		updatedAt := time.Unix(s.Time.Updated/1000, 0)
		runtime := ctx.now.Sub(createdAt)
		timeSinceUpdate := ctx.now.Sub(updatedAt)

		// Determine status based on recent activity
		// Priority: dead (3min silence) > active (recent) > idle (10min+)
		status := "active"
		var deathReason string
		if timeSinceUpdate > ctx.deadThreshold {
			status = "dead"
			deathReason = determineDeathReason(s.ID, createdAt, ctx.client)
		} else if timeSinceUpdate > ctx.activeThreshold {
			status = "idle"
		}

		// NOTE: IsProcessing is now populated client-side via SSE session.status events.
		agent := AgentAPIResponse{
			ID:           s.Title,
			SessionID:    s.ID,
			Status:       status,
			DeathReason:  deathReason,
			Runtime:      formatDuration(runtime),
			SpawnedAt:    createdAt.Format(time.RFC3339),
			UpdatedAt:    updatedAt.Format(time.RFC3339),
			IsProcessing: false,
			ProjectDir:   s.Directory,
		}

		// Derive beadsID and skill from session title
		if s.Title != "" {
			agent.BeadsID = extractBeadsIDFromTitle(s.Title)
			agent.Skill = extractSkillFromTitle(s.Title)
			agent.Project = extractProjectFromBeadsID(agent.BeadsID)
		}

		// Only include sessions spawned via orch spawn (have beads ID)
		if agent.BeadsID == "" {
			continue
		}

		// Mark untracked agents (--no-track spawns with synthetic beads IDs)
		if isUntrackedBeadsID(agent.BeadsID) {
			agent.IsUntracked = true
		}

		// OPTIMIZATION: Mark sessions older than beadsFetchThreshold as stale.
		// Still included in response but skip beads data fetch for performance.
		isStale := timeSinceUpdate > ctx.beadsFetchThreshold
		if isStale {
			agent.IsStale = true
			agent.Status = "idle"
		}

		// Track if this agent should be filtered after Phase check (two-threshold logic)
		if status == "idle" && timeSinceUpdate > ctx.displayThreshold && !isStale {
			ctx.pendingFilterByIDs[agent.BeadsID] = true
		}

		// Use cached PROJECT_DIR from workspace for cross-project visibility (O(1) lookup)
		ctx.collectProjectDir(agent.BeadsID)

		// Collect beads ID for batch fetch (non-stale, tracked agents only)
		ctx.collectBeadsID(agent.BeadsID, isStale, agent.IsUntracked)

		// Deduplicate by title - keep the most recently updated session
		if existingIdx, exists := ctx.seenTitles[s.Title]; exists {
			existingUpdatedAt, _ := time.Parse(time.RFC3339, ctx.agents[existingIdx].UpdatedAt)
			if updatedAt.After(existingUpdatedAt) {
				ctx.agents[existingIdx] = agent
			}
			continue
		}

		ctx.seenTitles[s.Title] = len(ctx.agents)
		ctx.agents = append(ctx.agents, agent)
	}
}

// collectTmuxAgents adds tmux-only agents that don't have OpenCode sessions.
func (ctx *agentCollectionContext) collectTmuxAgents() {
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
			for _, a := range ctx.agents {
				if (beadsID != "" && a.BeadsID == beadsID) || (a.ID != "" && strings.Contains(win.Name, a.ID)) {
					alreadyIn = true
					break
				}
			}

			if alreadyIn {
				continue
			}

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
			if beadsID != "" {
				if workspacePath := ctx.wsCache.lookupWorkspace(beadsID); workspacePath != "" {
					if spawnTime := spawn.ReadSpawnTime(workspacePath); !spawnTime.IsZero() {
						agent.SpawnedAt = spawnTime.Format(time.RFC3339)
						agent.Runtime = formatDuration(ctx.now.Sub(spawnTime))
					}

					if agentProjectDir := ctx.wsCache.lookupProjectDir(beadsID); agentProjectDir != "" {
						agent.ProjectDir = agentProjectDir
					}

					// Activity detection: check workspace file modification times
					lastActivity := getWorkspaceLastActivity(workspacePath)
					if !lastActivity.IsZero() {
						agent.LastActivityAt = lastActivity.Format(time.RFC3339)
						timeSinceActivity := ctx.now.Sub(lastActivity)
						if timeSinceActivity > ctx.deadThreshold {
							agent.Status = "dead"
							agent.DeathReason = "timeout"
						}
					}
				}
			}

			ctx.agents = append(ctx.agents, agent)

			// Collect beads ID for batch fetch
			if beadsID != "" && !ctx.seenBeadsIDs[beadsID] {
				ctx.beadsIDsToFetch = append(ctx.beadsIDsToFetch, beadsID)
				ctx.seenBeadsIDs[beadsID] = true
				ctx.collectProjectDir(beadsID)
			}
		}
	}
}

// collectCompletedWorkspaces adds completed workspaces (those with SYNTHESIS.md or light-tier completions).
func (ctx *agentCollectionContext) collectCompletedWorkspaces() {
	if len(ctx.wsCache.workspaceEntries) == 0 {
		return
	}

	for _, entry := range ctx.wsCache.workspaceEntries {
		if !entry.IsDir() {
			continue
		}

		// Check if already in active list
		alreadyIn := false
		workspaceName := entry.Name()
		for _, a := range ctx.agents {
			if a.ID == workspaceName || strings.HasPrefix(a.ID, workspaceName+" ") {
				alreadyIn = true
				break
			}
		}

		if alreadyIn {
			continue
		}

		// Use the lookup method for multi-project support
		workspacePath := ctx.wsCache.lookupWorkspacePathByEntry(entry.Name())
		synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
		hasSynthesis := false

		if _, err := os.Stat(synthesisPath); err == nil {
			hasSynthesis = true
		}

		// Only add workspaces that have SYNTHESIS.md or valid SPAWN_CONTEXT.md
		if !hasSynthesis {
			spawnContextPath := filepath.Join(workspacePath, "SPAWN_CONTEXT.md")
			if _, err := os.Stat(spawnContextPath); err != nil {
				continue
			}
		}

		agent := AgentAPIResponse{
			ID:     entry.Name(),
			Status: "completed",
		}

		// Set updated_at from workspace name date suffix or file modification time
		if parsedDate := extractDateFromWorkspaceName(entry.Name()); !parsedDate.IsZero() {
			agent.UpdatedAt = parsedDate.Format(time.RFC3339)
		} else if hasSynthesis {
			if info, err := os.Stat(synthesisPath); err == nil {
				agent.UpdatedAt = info.ModTime().Format(time.RFC3339)
			}
		} else {
			spawnContextPath := filepath.Join(workspacePath, "SPAWN_CONTEXT.md")
			if info, err := os.Stat(spawnContextPath); err == nil {
				agent.UpdatedAt = info.ModTime().Format(time.RFC3339)
			}
		}

		// Read session ID and spawn time from workspace
		if sessionID := spawn.ReadSessionID(workspacePath); sessionID != "" {
			agent.SessionID = sessionID
		}
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
			if content, err := os.ReadFile(synthesisPath); err == nil {
				agent.SynthesisContent = string(content)
			}
		}

		// Extract beadsID from workspace
		agent.BeadsID = extractBeadsIDFromWorkspace(workspacePath)
		if agent.BeadsID == "" {
			agent.BeadsID = extractBeadsIDFromTitle(entry.Name())
		}
		agent.Skill = extractSkillFromTitle(entry.Name())
		agent.Project = extractProjectFromBeadsID(agent.BeadsID)

		ctx.agents = append(ctx.agents, agent)
	}
}
