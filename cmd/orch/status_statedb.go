// Package main provides state DB integration for the status command.
// Pre-fetches immutable agent metadata from SQLite to avoid the distributed JOIN
// across multiple systems (OpenCode, beads, tmux, workspace, Anthropic API).
//
// This is the Phase A projection-first optimization:
//   - Reads immutable fields from state DB (session_id, beads_id, model, skill, etc.)
//   - Falls back to full multi-source path if state DB is empty/missing
//   - Only queries OpenCode API for live fields (is_processing, tokens)
//
// See: .kb/investigations/2026-02-06-inv-evaluate-single-source-agent-state.md
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/state"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// stateDBAgentResult holds agents discovered from the state DB along with metadata
// needed by the rest of the status pipeline.
type stateDBAgentResult struct {
	// Agents built from state DB data.
	agents []AgentInfo

	// Set of beads IDs already covered by state DB agents.
	seenBeadsIDs map[string]bool

	// Set of session IDs already covered.
	seenSessionIDs map[string]bool

	// Mapping of beadsID -> projectDir for cross-project comment lookups.
	beadsProjectDirs map[string]string

	// Beads IDs that need comment/issue fetching.
	beadsIDsToFetch []string
}

// fetchAgentsFromStateDB attempts to pre-populate agents from the state DB.
// Returns nil if the state DB is unavailable or empty (graceful degradation).
//
// This replaces the expensive multi-source discovery (tmux + OpenCode sessions
// + workspace manifest reads) for agents that were tracked at spawn time.
func fetchAgentsFromStateDB(showAll bool) *stateDBAgentResult {
	db, err := state.OpenDefault()
	if err != nil || db == nil {
		return nil // Graceful degradation: state DB unavailable
	}
	defer db.Close()

	var dbAgents []*state.Agent
	if showAll {
		dbAgents, err = db.ListAllAgents()
	} else {
		dbAgents, err = db.ListActiveAgents()
	}
	if err != nil || len(dbAgents) == 0 {
		return nil // No agents in state DB
	}

	now := time.Now()
	projectDir, _ := os.Getwd()

	result := &stateDBAgentResult{
		agents:           make([]AgentInfo, 0, len(dbAgents)),
		seenBeadsIDs:     make(map[string]bool, len(dbAgents)),
		seenSessionIDs:   make(map[string]bool, len(dbAgents)),
		beadsProjectDirs: make(map[string]string, len(dbAgents)),
		beadsIDsToFetch:  make([]string, 0, len(dbAgents)),
	}

	for _, dbAgent := range dbAgents {
		agent := agentInfoFromStateDB(dbAgent, now, projectDir)

		if dbAgent.BeadsID != "" {
			result.seenBeadsIDs[dbAgent.BeadsID] = true
			result.beadsIDsToFetch = append(result.beadsIDsToFetch, dbAgent.BeadsID)

			// Set project dir for cross-project comment lookups
			if dbAgent.ProjectDir != "" && dbAgent.ProjectDir != projectDir {
				result.beadsProjectDirs[dbAgent.BeadsID] = dbAgent.ProjectDir
			}
		}
		if dbAgent.SessionID != "" {
			result.seenSessionIDs[dbAgent.SessionID] = true
		}

		result.agents = append(result.agents, agent)
	}

	return result
}

// agentInfoFromStateDB converts a state.Agent (SQLite row) to an AgentInfo for display.
// Populates all immutable fields from the cached data.
func agentInfoFromStateDB(dbAgent *state.Agent, now time.Time, currentProjectDir string) AgentInfo {
	agent := AgentInfo{
		SessionID:  dbAgent.SessionID,
		BeadsID:    dbAgent.BeadsID,
		Mode:       dbAgent.Mode,
		Model:      dbAgent.Model,
		Skill:      dbAgent.Skill,
		Project:    dbAgent.ProjectName,
		ProjectDir: dbAgent.ProjectDir,
		Task:       truncate(dbAgent.IssueTitle, 40),
		Window:     dbAgent.TmuxWindow,
	}

	// Compute runtime from spawn time
	if dbAgent.SpawnTime > 0 {
		spawnTime := time.UnixMilli(dbAgent.SpawnTime)
		agent.Runtime = formatDuration(now.Sub(spawnTime))
	}

	// Set phase from cached state (may be stale, will be refreshed from beads comments)
	if dbAgent.Phase != "" {
		agent.Phase = dbAgent.Phase
		if dbAgent.PhaseReportedAt > 0 {
			t := time.UnixMilli(dbAgent.PhaseReportedAt)
			agent.PhaseReportedAt = &t
		}
	}

	// Set completion/abandonment status from cached state
	agent.IsCompleted = dbAgent.IsCompleted

	// Treat --no-track spawns as untracked
	if dbAgent.BeadsID != "" && isUntrackedBeadsID(dbAgent.BeadsID) {
		agent.IsUntracked = true
	}

	// Set project dir for cross-project display
	if dbAgent.ProjectDir != "" && dbAgent.ProjectDir != currentProjectDir {
		agent.ProjectDir = dbAgent.ProjectDir
	}

	return agent
}

// enrichStateDBAgentsLive queries live-only data sources for agents pre-fetched from state DB.
// This replaces the full multi-source enrichment with targeted queries for mutable fields only.
//
// Live fields queried:
//   - OpenCode: is_processing, tokens (for recently active sessions)
//   - Tmux: window existence (to detect phantom agents)
//   - Beads: phase (from comments), issue status (open/closed)
func enrichStateDBAgentsLive(
	result *stateDBAgentResult,
	client *opencode.Client,
	sessions []opencode.Session,
	now time.Time,
	showAll bool,
	projectDir string,
	timer func(string),
) {
	if result == nil || len(result.agents) == 0 {
		return
	}

	// === Match state DB agents to OpenCode sessions ===
	// State DB may have agents without session_id (if RecordSessionID wasn't called
	// or failed). Match them via beads ID in session title.
	const maxIdleTime = 30 * time.Minute
	for i := range result.agents {
		agent := &result.agents[i]
		if agent.SessionID != "" {
			continue // Already has session ID
		}
		if agent.BeadsID == "" {
			continue
		}
		// Search sessions by beads ID in title
		for j := range sessions {
			s := &sessions[j]
			updatedAt := time.Unix(s.Time.Updated/1000, 0)
			if now.Sub(updatedAt) > maxIdleTime {
				continue
			}
			if extractBeadsIDFromTitle(s.Title) == agent.BeadsID {
				agent.SessionID = s.ID
				createdAt := time.Unix(s.Time.Created/1000, 0)
				agent.Runtime = formatDuration(now.Sub(createdAt))
				agent.LastActivity = updatedAt
				result.seenSessionIDs[s.ID] = true
				break
			}
		}
	}

	// === Parallel batch: live enrichment + comments + issues ===
	var commentsMap map[string][]verify.Comment
	var allIssues map[string]*verify.Issue
	var dataWg sync.WaitGroup

	beadsIDsToFetch := result.beadsIDsToFetch

	// Batch fetch comments
	dataWg.Add(1)
	go func() {
		defer dataWg.Done()
		commentsMap = verify.GetCommentsBatchWithProjectDirs(beadsIDsToFetch, result.beadsProjectDirs)
	}()

	// Batch fetch issue details
	dataWg.Add(1)
	go func() {
		defer dataWg.Done()
		allIssues, _ = verify.GetIssuesBatch(beadsIDsToFetch, result.beadsProjectDirs)
	}()

	// Parallel enrichment: fetch processing + tokens for recently active sessions.
	// Uses a mutex-protected slice instead of channels for simpler goroutine coordination.
	type enrichItem struct {
		idx        int
		enrichment opencode.SessionEnrichment
	}
	var enrichResults []enrichItem
	var enrichMu sync.Mutex
	var enrichWg sync.WaitGroup

	for i := range result.agents {
		agent := &result.agents[i]
		if agent.SessionID == "" || agent.IsCompleted {
			continue
		}

		enrichWg.Add(1)
		go func(idx int, sessionID string) {
			defer enrichWg.Done()
			e := client.GetSessionEnrichment(sessionID)
			enrichMu.Lock()
			enrichResults = append(enrichResults, enrichItem{idx: idx, enrichment: e})
			enrichMu.Unlock()
		}(i, agent.SessionID)
	}

	// Wait for enrichment in parallel with comments/issues
	dataWg.Add(1)
	go func() {
		defer dataWg.Done()
		enrichWg.Wait()
	}()

	// Check tmux window liveness in batch (fast, <50ms total)
	existingWindows := tmux.ListAllWindowTargets()

	dataWg.Wait()
	timer("stateDB parallel fetch (enrichment + comments + issues)")

	// Apply enrichment results
	for _, er := range enrichResults {
		agent := &result.agents[er.idx]
		if er.enrichment.Model != "" {
			agent.Model = er.enrichment.Model
		}
		agent.IsProcessing = er.enrichment.IsProcessing
		agent.Tokens = er.enrichment.Tokens
	}

	// Apply beads comments, issues, and tmux liveness
	for i := range result.agents {
		agent := &result.agents[i]

		// Update phase from beads comments (authoritative, may override stale cached value)
		if comments, ok := commentsMap[agent.BeadsID]; ok {
			phaseStatus := verify.ParsePhaseFromComments(comments)
			if phaseStatus.Found {
				agent.Phase = phaseStatus.Phase
				agent.PhaseReportedAt = phaseStatus.PhaseReportedAt
			}
		}

		// Update task and completion status from beads issues (authoritative)
		issue, issueExists := allIssues[agent.BeadsID]
		if issueExists && issue != nil {
			if issue.Title != "" {
				agent.Task = truncate(issue.Title, 40)
			}
			agent.IsCompleted = strings.EqualFold(issue.Status, "closed")
		}

		// Check tmux window liveness
		if agent.Window != "" {
			if !existingWindows[agent.Window] {
				agent.Window = "" // Window no longer exists
			}
		}

		// For tmux-based agents, check pane activity
		if agent.Window != "" {
			paneRunning := tmux.IsPaneProcessRunning(agent.Window)
			if agent.SessionID == "" {
				agent.IsProcessing = paneRunning
			} else if !agent.IsProcessing && paneRunning {
				agent.IsProcessing = true
			}
		}

		// Treat --no-track spawns as untracked for swarm accounting
		if agent.BeadsID != "" && isUntrackedBeadsID(agent.BeadsID) {
			agent.IsUntracked = true
		}

		// Compute phantom status
		agent.IsPhantom = computeIsPhantom(*agent, issue, issueExists)

		// Determine source indicator
		agent.Source = determineAgentSource(*agent, projectDir)

		// Ensure runtime has a value
		if agent.Runtime == "" {
			agent.Runtime = "unknown"
		}
	}

	timer("stateDB live enrichment apply")
}

// mergeDiscoveredAgents adds agents found through traditional discovery that
// were NOT already in the state DB result. This handles:
//   - Agents spawned before state DB was implemented (no state DB row)
//   - Untracked sessions (OpenCode sessions without beads tracking)
func mergeDiscoveredAgents(
	result *stateDBAgentResult,
	discoveredAgents []AgentInfo,
	discoveredProjectDirs map[string]string,
) {
	for i, agent := range discoveredAgents {
		// Skip if already covered by state DB
		if agent.BeadsID != "" && result.seenBeadsIDs[agent.BeadsID] {
			continue
		}
		if agent.SessionID != "" && result.seenSessionIDs[agent.SessionID] {
			continue
		}

		result.agents = append(result.agents, discoveredAgents[i])
		if agent.BeadsID != "" {
			result.seenBeadsIDs[agent.BeadsID] = true
			result.beadsIDsToFetch = append(result.beadsIDsToFetch, agent.BeadsID)
		}
		if agent.SessionID != "" {
			result.seenSessionIDs[agent.SessionID] = true
		}
	}

	// Merge project dirs
	for beadsID, dir := range discoveredProjectDirs {
		if _, exists := result.beadsProjectDirs[beadsID]; !exists {
			result.beadsProjectDirs[beadsID] = dir
		}
	}
}

// fallbackDiscoverAgents performs the traditional multi-source agent discovery.
// Used as fallback when state DB is unavailable and for discovering agents not in state DB.
func fallbackDiscoverAgents(
	sessions []opencode.Session,
	now time.Time,
	projectDir string,
) (agents []AgentInfo, beadsIDsToFetch []string, beadsProjectDirs map[string]string, seenBeadsIDs map[string]bool) {
	agents = make([]AgentInfo, 0)
	seenBeadsIDs = make(map[string]bool)
	beadsProjectDirs = make(map[string]string)

	// Build session maps
	beadsToSession := make(map[string]*opencode.Session)
	const maxIdleTime = 30 * time.Minute
	for i := range sessions {
		s := &sessions[i]
		updatedAt := time.Unix(s.Time.Updated/1000, 0)
		if now.Sub(updatedAt) <= maxIdleTime {
			beadsID := extractBeadsIDFromTitle(s.Title)
			if beadsID != "" {
				beadsToSession[beadsID] = s
			}
		}
	}

	// Phase 1: tmux windows
	workersSessions, _ := tmux.ListWorkersSessions()
	for _, sessionName := range workersSessions {
		windows, _ := tmux.ListWindows(sessionName)
		for _, w := range windows {
			if w.Name == "servers" || w.Name == "zsh" {
				continue
			}
			beadsID := extractBeadsIDFromWindowName(w.Name)
			if beadsID == "" || seenBeadsIDs[beadsID] {
				continue
			}

			agentProject := extractProjectFromBeadsID(beadsID)
			info := AgentInfo{
				BeadsID: beadsID,
				Mode:    "claude",
				Skill:   extractSkillFromWindowName(w.Name),
				Project: agentProject,
				Window:  w.Target,
				Title:   w.Name,
			}

			agentProjectDir := projectDir
			if agentProject != "" && agentProject != filepath.Base(projectDir) {
				if derivedDir := findProjectDirByName(agentProject); derivedDir != "" {
					agentProjectDir = derivedDir
					info.ProjectDir = derivedDir
					beadsProjectDirs[beadsID] = derivedDir
				}
			}

			workspacePath, _ := findWorkspaceByBeadsID(agentProjectDir, beadsID)
			if workspacePath != "" {
				manifest := readAgentManifest(workspacePath)
				if manifest != nil {
					if manifest.Skill != "" {
						info.Skill = manifest.Skill
					}
					if manifest.ProjectDir != "" {
						info.ProjectDir = manifest.ProjectDir
						beadsProjectDirs[beadsID] = manifest.ProjectDir
					}
					if manifest.SpawnMode != "" {
						info.Mode = manifest.SpawnMode
					}
					if manifest.Model != "" {
						info.Model = manifest.Model
					}
				}
			}

			agents = append(agents, info)
			beadsIDsToFetch = append(beadsIDsToFetch, beadsID)
			seenBeadsIDs[beadsID] = true
		}
	}

	// Phase 2: OpenCode sessions with beads IDs
	for _, s := range sessions {
		updatedAt := time.Unix(s.Time.Updated/1000, 0)
		if now.Sub(updatedAt) > maxIdleTime {
			continue
		}
		beadsID := extractBeadsIDFromTitle(s.Title)
		if beadsID == "" || seenBeadsIDs[beadsID] {
			continue
		}

		createdAt := time.Unix(s.Time.Created/1000, 0)
		agents = append(agents, AgentInfo{
			SessionID:    s.ID,
			BeadsID:      beadsID,
			Mode:         "opencode",
			Title:        s.Title,
			Runtime:      formatDuration(now.Sub(createdAt)),
			LastActivity: updatedAt,
			Skill:        extractSkillFromTitle(s.Title),
			Project:      extractProjectFromBeadsID(beadsID),
		})
		beadsIDsToFetch = append(beadsIDsToFetch, beadsID)
		seenBeadsIDs[beadsID] = true
	}

	// Phase 3: Untracked OpenCode sessions
	seenSessionIDs := make(map[string]bool)
	for _, agent := range agents {
		if agent.SessionID != "" {
			seenSessionIDs[agent.SessionID] = true
		}
	}

	for _, s := range sessions {
		if seenSessionIDs[s.ID] {
			continue
		}
		if extractBeadsIDFromTitle(s.Title) != "" {
			continue
		}
		updatedAt := time.Unix(s.Time.Updated/1000, 0)
		untrackedMaxIdleTime := 2 * time.Hour
		if now.Sub(updatedAt) > untrackedMaxIdleTime {
			continue
		}
		createdAt := time.Unix(s.Time.Created/1000, 0)
		projectName := ""
		if s.Directory != "" && s.Directory != "/" {
			projectName = filepath.Base(s.Directory)
		}
		agents = append(agents, AgentInfo{
			SessionID:    s.ID,
			Mode:         "opencode",
			Title:        s.Title,
			Runtime:      formatDuration(now.Sub(createdAt)),
			LastActivity: updatedAt,
			Project:      projectName,
			ProjectDir:   s.Directory,
			IsUntracked:  true,
		})
		seenSessionIDs[s.ID] = true
	}

	// Resolve beads project dirs
	for beadsID, session := range beadsToSession {
		if session != nil && session.Directory != "" && session.Directory != "/" && session.Directory != projectDir {
			if _, has := beadsProjectDirs[beadsID]; !has {
				beadsProjectDirs[beadsID] = session.Directory
			}
		}
	}
	for _, beadsID := range beadsIDsToFetch {
		if _, has := beadsProjectDirs[beadsID]; has {
			continue
		}
		workspacePath, _ := findWorkspaceByBeadsID(projectDir, beadsID)
		if workspacePath != "" {
			agentProjectDir := extractProjectDirFromWorkspace(workspacePath)
			if agentProjectDir != "" {
				beadsProjectDirs[beadsID] = agentProjectDir
				continue
			}
		}
		projectName := extractProjectFromBeadsID(beadsID)
		if projectName != "" && projectName != "untracked" {
			if derivedDir := findProjectDirByName(projectName); derivedDir != "" {
				beadsProjectDirs[beadsID] = derivedDir
			}
		}
	}

	return agents, beadsIDsToFetch, beadsProjectDirs, seenBeadsIDs
}

// runStatusFallbackPath runs the original multi-source agent discovery and enrichment.
// This is used when the state DB is empty or unavailable.
func runStatusFallbackPath(
	client *opencode.Client,
	sessions []opencode.Session,
	now time.Time,
	projectDir string,
	timer func(string),
) []AgentInfo {
	debugTiming := os.Getenv("ORCH_STATUS_DEBUG") != ""

	// Discover agents from all sources
	agents, beadsIDsToFetch, beadsProjectDirs, _ := fallbackDiscoverAgents(sessions, now, projectDir)

	// Build session maps for enrichment
	sessionMap := make(map[string]*opencode.Session)
	beadsToSession := make(map[string]*opencode.Session)
	const maxIdleTime = 30 * time.Minute

	for i := range sessions {
		s := &sessions[i]
		sessionMap[s.ID] = s
		updatedAt := time.Unix(s.Time.Updated/1000, 0)
		if now.Sub(updatedAt) <= maxIdleTime {
			beadsID := extractBeadsIDFromTitle(s.Title)
			if beadsID != "" {
				beadsToSession[beadsID] = s
			}
		}
	}

	timer("fallback beadsProjectDirs resolution")

	// Compact mode: reduce beads queries
	if !statusAll {
		recentBeadsIDs := make([]string, 0, len(beadsIDsToFetch))
		for _, beadsID := range beadsIDsToFetch {
			if s, ok := beadsToSession[beadsID]; ok {
				updatedAt := time.Unix(s.Time.Updated/1000, 0)
				if now.Sub(updatedAt) > 2*time.Hour {
					continue
				}
			}
			recentBeadsIDs = append(recentBeadsIDs, beadsID)
		}
		if debugTiming {
			fmt.Fprintf(os.Stderr, "[timing] compact mode: reduced beads IDs from %d to %d\n",
				len(beadsIDsToFetch), len(recentBeadsIDs))
		}
		beadsIDsToFetch = recentBeadsIDs
	}

	// Track sessions needing enrichment
	type enrichmentTarget struct {
		agentIdx  int
		sessionID string
		updatedAt time.Time
	}
	var enrichTargets []enrichmentTarget

	for i, agent := range agents {
		if agent.SessionID == "" {
			continue
		}
		if s, ok := sessionMap[agent.SessionID]; ok {
			updatedAt := time.Unix(s.Time.Updated/1000, 0)
			if now.Sub(updatedAt) <= processingCheckMaxAge {
				enrichTargets = append(enrichTargets, enrichmentTarget{
					agentIdx:  i,
					sessionID: agent.SessionID,
					updatedAt: updatedAt,
				})
			}
		}
	}

	// === Parallel batch: enrichment + comments + issues ===
	var commentsMap map[string][]verify.Comment
	var allIssues map[string]*verify.Issue
	var dataWg sync.WaitGroup

	dataWg.Add(1)
	go func() {
		defer dataWg.Done()
		commentsMap = verify.GetCommentsBatchWithProjectDirs(beadsIDsToFetch, beadsProjectDirs)
	}()

	dataWg.Add(1)
	go func() {
		defer dataWg.Done()
		allIssues, _ = verify.GetIssuesBatch(beadsIDsToFetch, beadsProjectDirs)
	}()

	dataWg.Add(1)
	go func() {
		defer dataWg.Done()
		if len(enrichTargets) == 0 {
			return
		}
		var enrichWg sync.WaitGroup
		for _, target := range enrichTargets {
			enrichWg.Add(1)
			go func(t enrichmentTarget) {
				defer enrichWg.Done()
				enrichment := client.GetSessionEnrichment(t.sessionID)
				agents[t.agentIdx].Model = enrichment.Model
				agents[t.agentIdx].IsProcessing = enrichment.IsProcessing
				agents[t.agentIdx].Tokens = enrichment.Tokens
			}(target)
		}
		enrichWg.Wait()
	}()

	dataWg.Wait()
	timer("fallback parallel data fetch")

	// Enrich and finalize agent data
	for i := range agents {
		agent := &agents[i]

		if comments, ok := commentsMap[agent.BeadsID]; ok {
			phaseStatus := verify.ParsePhaseFromComments(comments)
			if phaseStatus.Found {
				agent.Phase = phaseStatus.Phase
				agent.PhaseReportedAt = phaseStatus.PhaseReportedAt
			}
		}

		issue, issueExists := allIssues[agent.BeadsID]
		if issueExists && issue != nil {
			agent.Task = truncate(issue.Title, 40)
			agent.IsCompleted = strings.EqualFold(issue.Status, "closed")
		}

		if agent.BeadsID != "" && isUntrackedBeadsID(agent.BeadsID) {
			agent.IsUntracked = true
		}

		agent.IsPhantom = computeIsPhantom(*agent, issue, issueExists)
		agent.Source = determineAgentSource(*agent, projectDir)

		if agent.Mode == "claude" && agent.SessionID != "" {
			if s, ok := sessionMap[agent.SessionID]; ok {
				createdAt := time.Unix(s.Time.Created/1000, 0)
				updatedAt := time.Unix(s.Time.Updated/1000, 0)
				agent.Runtime = formatDuration(now.Sub(createdAt))
				if agent.Title == "" {
					agent.Title = s.Title
				}
				agent.IsProcessing = isSessionLikelyProcessing(client, s.ID, updatedAt, now)
			}
		}

		if agent.Window != "" {
			paneRunning := tmux.IsPaneProcessRunning(agent.Window)
			if agent.SessionID == "" {
				agent.IsProcessing = paneRunning
			} else if !agent.IsProcessing && paneRunning {
				agent.IsProcessing = true
			}
		}

		if agent.Runtime == "" {
			agent.Runtime = "unknown"
		}
	}

	timer("fallback agent enrichment")
	return agents
}

// filterAgentsForDisplay applies display filtering based on flags.
// Compact mode (default): Only show running agents + recently completed + needs-attention.
// Full mode (--all): Show all agents.
func filterAgentsForDisplay(agents []AgentInfo, showAll bool, projectFilter string) []AgentInfo {
	filteredAgents := make([]AgentInfo, 0, len(agents))
	for _, agentItem := range agents {
		// Filter by project if specified
		if projectFilter != "" && agentItem.Project != projectFilter {
			continue
		}

		// In compact mode, only show:
		// 1. Running (processing) agents
		// 2. RECENT agents with Phase: Complete (need review)
		// 3. Agents with Phase: BLOCKED or QUESTION (need attention)
		// 4. Untracked sessions (always visible for resource monitoring)
		if !showAll && !agentItem.IsUntracked {
			isRunning := agentItem.IsProcessing

			isComplete := strings.EqualFold(agentItem.Phase, "Complete")
			isRecent := true
			if isComplete && agentItem.PhaseReportedAt != nil {
				if time.Since(*agentItem.PhaseReportedAt) > compactCompletedAgentsMaxAge {
					isRecent = false
				}
			}

			needsAttention := (isComplete && isRecent) ||
				strings.EqualFold(agentItem.Phase, "BLOCKED") ||
				strings.EqualFold(agentItem.Phase, "QUESTION")

			if !isRunning && !needsAttention {
				continue
			}
		}

		// Filter completed agents (beads issue closed) unless --all is set
		if agentItem.IsCompleted && !showAll {
			continue
		}

		// Filter phantom agents unless --all is set
		if agentItem.IsPhantom && !showAll {
			continue
		}

		filteredAgents = append(filteredAgents, agentItem)
	}
	return filteredAgents
}
