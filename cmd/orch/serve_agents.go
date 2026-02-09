package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/config"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
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
// - serve_agents_collect.go: agentCollectionContext, collectOpenCodeSessions, collectTmuxAgents, collectCompletedWorkspaces
// - serve_agents_enrich.go: enrichAgentsWithBeadsData, enrichSingleAgent, applyGhostFilter, fetchTokensAndActivity, applyLateFilters
// - serve_agents_status.go: determineDeathReason, determineAgentStatus, checkWorkspaceSynthesis,
//   getWorkspaceLastActivity
// - serve_agents_activity.go: handleSessionMessages, extractLastActivityFromMessages,
//   findWorkspaceBySessionID, loadActivityFromWorkspace, MessagePartResponse types
// - serve_agents_gap.go: getGapAnalysisFromEvents, extractGapAnalysisFromEvent
// - serve_agents_investigation.go: investigationDirCache, buildInvestigationDirCache, discoverInvestigationPath

// handleAgents returns JSON list of active agents from OpenCode/tmux and completed workspaces.
// Query parameters:
//   - since: Time filter (12h, 24h, 48h, 7d, all). Default: 12h
//   - project: Project filter (full path or project name). Default: none (all projects)
//
// Pipeline: collect (sessions, tmux, workspaces) -> enrich (beads, investigation, synthesis) -> filter -> format
func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters for filtering
	sinceDuration := parseSinceParam(r)
	projectFilterParam := parseProjectFilter(r)

	// Use sourceDir (set at build time) since serve may run from any working directory
	projectDir, _ := s.currentProjectDir()
	projCfg, _ := config.Load(projectDir)

	client := opencode.NewClient(s.ServerURL)

	// Get active sessions from OpenCode (all projects, not filtered by directory).
	// Gracefully degrade if OpenCode is unavailable — dashboard still shows tmux/workspace data.
	sessions, err := client.ListSessions("")
	if err != nil {
		sessions = nil // Continue without OpenCode sessions
	}

	// EARLY FILTERING: Apply time filter before expensive operations (workspace cache, beads).
	// NOTE: Project filter is NOT applied here because s.Directory may be the orchestrator's cwd
	// due to OpenCode --attach bug. Correct project_dir is populated from workspace cache later.
	sessions = filterSessionsByTime(sessions, sinceDuration)

	// Build multi-project workspace cache for cross-project agent visibility (30s TTL).
	projectDirs := extractUniqueProjectDirs(sessions, projectDir)
	wsCache := s.WorkspaceCache.getCachedWorkspace(projectDirs)

	// Create collection context with thresholds and dependencies
	ctx := newAgentCollectionContext(client, wsCache, s.BeadsCache, sinceDuration, s.ServerStartTime, projCfg)

	// Phase 1: Collect agents from all sources
	ctx.collectOpenCodeSessions(sessions)
	ctx.collectTmuxAgents()
	ctx.collectCompletedWorkspaces()

	// Phase 2: Enrich with beads data (phase, task, investigation, synthesis, gap analysis)
	// Also applies ghost filtering for idle agents
	ctx.enrichAgentsWithBeadsData()

	// Phase 3: Fetch token usage, activity, and model (parallelized)
	ctx.fetchTokensAndActivity()

	// Phase 4: Apply late time/project filters (for tmux agents and completed workspaces)
	ctx.applyLateFilters(sinceDuration, projectFilterParam)

	// Encode response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ctx.agents); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode agents: %v", err), http.StatusInternalServerError)
		return
	}
}

// filterSessionsByTime applies early time filtering to sessions before expensive operations.
func filterSessionsByTime(sessions []opencode.Session, sinceDuration time.Duration) []opencode.Session {
	if sinceDuration <= 0 {
		return sessions
	}
	now := time.Now()
	filtered := make([]opencode.Session, 0, len(sessions))
	for _, s := range sessions {
		updatedAt := time.Unix(s.Time.Updated/1000, 0)
		if now.Sub(updatedAt) > sinceDuration {
			continue
		}
		filtered = append(filtered, s)
	}
	return filtered
}

// handleCacheInvalidate clears all dashboard caches to force fresh data on next request.
// This is called by orch complete to ensure the dashboard shows updated agent status.
// Without this, the TTL cache holds stale "active" status after agents complete.
func (s *Server) handleCacheInvalidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Invalidate beads cache (open issues, all issues, comments)
	if s.BeadsCache != nil {
		s.BeadsCache.invalidate()
	}

	// Invalidate beads stats cache (stats, ready issues)
	// Empty string clears all project caches
	if s.BeadsStatsCache != nil {
		s.BeadsStatsCache.invalidate("")
	}

	// Invalidate kb health cache (knowledge hygiene signals)
	if s.KBHealthCache != nil {
		s.KBHealthCache.invalidate()
	}

	// Invalidate workspace cache (workspace metadata)
	s.WorkspaceCache.invalidate()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": "Cache invalidated",
	})
}
