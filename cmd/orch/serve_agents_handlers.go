package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/agent"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// handleAgents returns JSON list of in-progress agents (beads-first) and completed workspaces.
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

	sessions, err := listSessionsAcrossProjects(client, projectDir)
	if err != nil {
		log.Printf("Warning: failed to list sessions: %v", err)
	}

	sessionByID := make(map[string]*opencode.Session)
	beadsToSession := make(map[string]*opencode.Session)
	for i := range sessions {
		session := &sessions[i]
		sessionByID[session.ID] = session

		beadsID := beadsIDFromSession(session)
		if beadsID != "" {
			beadsToSession[beadsID] = session
		}
	}

	sessionStatusMap := make(map[string]opencode.SessionStatusInfo)
	if status, err := client.GetAllSessionStatus(); err != nil {
		log.Printf("Warning: failed to fetch session status: %v", err)
	} else {
		sessionStatusMap = status
	}

	projectDirs := uniqueProjectDirs(append([]string{projectDir}, getKBProjectsFn()...))
	wsCache := globalWorkspaceCacheInstance.getCachedWorkspace(projectDirs)

	agents := []AgentAPIResponse{} // Initialize as empty slice, not nil, to return [] instead of null

	// Collect beads IDs for batch fetching
	beadsIDsToFetch := make([]string, 0)
	seenBeadsIDs := make(map[string]bool)

	// Track project directories for cross-project agents
	// Key: beadsID, Value: projectDir from workspace SPAWN_CONTEXT.md or beads query
	beadsProjectDirs := make(map[string]string)

	// Beads-first discovery: start from in_progress issues
	inProgressIssues, issueProjectDirs := listInProgressIssues(projectDirs)
	for beadsID, projectPath := range issueProjectDirs {
		if projectPath != "" {
			beadsProjectDirs[beadsID] = projectPath
		}
	}

	now := time.Now()
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

	// beadsFetchThreshold limits which completed workspaces we fetch beads data for.
	// This remains important for performance on large workspaces archives.
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

	for beadsID, issue := range inProgressIssues {
		agent := AgentAPIResponse{
			BeadsID:    beadsID,
			BeadsTitle: issue.Title,
			Task:       truncate(issue.Title, 60),
			Status:     "dead",
			Project:    extractProjectFromBeadsID(beadsID),
			ProjectDir: issueProjectDirs[beadsID],
		}

		workspacePath := wsCache.lookupWorkspace(beadsID)
		if workspacePath == "" {
			if session, ok := beadsToSession[beadsID]; ok {
				if sessionWorkspace := workspacePathFromSession(session); sessionWorkspace != "" {
					workspacePath = sessionWorkspace
				}
			}
		}
		if workspacePath != "" {
			workspaceName := filepath.Base(workspacePath)
			agent.ID = workspaceName
			agent.Skill = extractSkillFromTitle(workspaceName)
			if sessionID := spawn.ReadSessionID(workspacePath); sessionID != "" {
				agent.SessionID = sessionID
			}
			if spawnTime := spawn.ReadSpawnTime(workspacePath); !spawnTime.IsZero() {
				agent.SpawnedAt = spawnTime.Format(time.RFC3339)
			}
			if agentProjectDir := wsCache.lookupProjectDir(beadsID); agentProjectDir != "" {
				agent.ProjectDir = agentProjectDir
				beadsProjectDirs[beadsID] = agentProjectDir
			}
		}

		if agent.SessionID == "" {
			if session, ok := beadsToSession[beadsID]; ok {
				agent.SessionID = session.ID
			}
		}

		if agent.Project == "" && agent.ProjectDir != "" {
			agent.Project = extractProjectName(agent.ProjectDir)
		}

		if agent.SessionID != "" {
			session := sessionByID[agent.SessionID]
			if session == nil {
				var err error
				session, err = client.GetSession(agent.SessionID)
				if err != nil {
					log.Printf("Warning: failed to fetch session %s: %v", agent.SessionID, err)
				}
			}

			if session != nil {
				if agent.BeadsID == "" {
					agent.BeadsID = beadsIDFromSession(session)
				}
				if agent.ID == "" {
					agent.ID = workspaceNameFromSession(session)
				}
				if agent.ProjectDir == "" {
					if projectDir := projectDirFromWorkspacePath(workspacePathFromSession(session)); projectDir != "" {
						agent.ProjectDir = projectDir
					} else if session.Directory != "" {
						agent.ProjectDir = session.Directory
					}
				}
				if agent.Tier == "" && session.Metadata != nil {
					if tier, ok := session.Metadata["tier"]; ok {
						agent.Tier = tier
					}
				}

				if agent.ProjectDir != "" {
					if _, exists := beadsProjectDirs[beadsID]; !exists {
						beadsProjectDirs[beadsID] = agent.ProjectDir
					}
				}

				createdAt := time.Unix(session.Time.Created/1000, 0)
				updatedAt := time.Unix(session.Time.Updated/1000, 0)
				runtime := now.Sub(createdAt)
				timeSinceUpdate := now.Sub(updatedAt)

				status := "active"
				if timeSinceUpdate > deadThreshold {
					status = "dead"
				} else if timeSinceUpdate > activeThreshold {
					status = "idle"
				}

				agent.Status = status
				agent.Runtime = formatDuration(runtime)
				if agent.SpawnedAt == "" {
					agent.SpawnedAt = createdAt.Format(time.RFC3339)
				}
				agent.UpdatedAt = updatedAt.Format(time.RFC3339)
			}

			if statusInfo, ok := sessionStatusMap[agent.SessionID]; ok {
				agent.IsProcessing = statusInfo.IsBusy() || statusInfo.IsRetrying()
			}
		}

		if agent.Status == "idle" && agent.UpdatedAt != "" {
			if updatedAt, err := time.Parse(time.RFC3339, agent.UpdatedAt); err == nil {
				if now.Sub(updatedAt) > displayThreshold {
					pendingFilterByBeadsID[beadsID] = true
				}
			}
		}

		if !seenBeadsIDs[beadsID] {
			beadsIDsToFetch = append(beadsIDsToFetch, beadsID)
			seenBeadsIDs[beadsID] = true
		}
		agents = append(agents, agent)
	}

	// Add completed workspaces (those with SYNTHESIS.md or light-tier completions)
	// Reuse cached workspace entries to avoid redundant directory reads
	// Multi-project support: entries may come from different project workspace directories
	if len(wsCache.workspaceEntries) > 0 {
		for _, entry := range wsCache.workspaceEntries {
			if !entry.IsDir() {
				continue
			}

			workspaceName := entry.Name()

			// Extract workspacePath and beadsID early for accurate dedup.
			// This catches tmux agents whose IDs include emoji prefix and [beads-id] suffix,
			// preventing the workspace scan from adding a duplicate "completed" entry.
			workspacePath := wsCache.lookupWorkspacePathByEntry(workspaceName)
			wsBeadsID := extractBeadsIDFromWorkspace(workspacePath)
			if wsBeadsID == "" {
				wsBeadsID = extractBeadsIDFromTitle(workspaceName)
			}

			// Check if already in active list (by name OR beads_id)
			alreadyIn := false
			for _, a := range agents {
				if a.ID == workspaceName || strings.HasPrefix(a.ID, workspaceName+" ") {
					alreadyIn = true
					break
				}
				if wsBeadsID != "" && a.BeadsID == wsBeadsID {
					alreadyIn = true
					break
				}
			}

			if alreadyIn {
				continue
			}
			synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
			hasSynthesis := false

			// Check if SYNTHESIS.md exists (indicates full-tier completion)
			if _, err := os.Stat(synthesisPath); err == nil {
				hasSynthesis = true
			}

			// Only add workspaces that have SYNTHESIS.md in this scan.
			// Workspaces without SYNTHESIS.md are discovered via beads-first discovery,
			// which correctly sets status based on session activity and beads issue status.
			// Light-tier completions (no SYNTHESIS.md) are handled via Phase: Complete in beads comments,
			// but that check happens in the beads-first loop enrichment, not here.
			if !hasSynthesis {
				continue // Skip workspaces without SYNTHESIS.md - they're handled by beads-first discovery
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

			agent.BeadsID = wsBeadsID
			agent.Skill = extractSkillFromTitle(entry.Name())

			// Include beads enrichment for recent workspace entries to populate phase data.
			// Performance guard: only enrich workspaces within beadsFetchThreshold to avoid
			// CPU spikes from fetching beads data for 600+ historical workspaces.
			if agent.BeadsID != "" && !seenBeadsIDs[agent.BeadsID] {
				workspaceTime := extractDateFromWorkspaceName(workspaceName)
				if !workspaceTime.IsZero() && now.Sub(workspaceTime) <= beadsFetchThreshold {
					beadsIDsToFetch = append(beadsIDsToFetch, agent.BeadsID)
					seenBeadsIDs[agent.BeadsID] = true

					// For cross-project agent visibility: use cached PROJECT_DIR
					if agentProjectDir := wsCache.lookupProjectDir(agent.BeadsID); agentProjectDir != "" {
						beadsProjectDirs[agent.BeadsID] = agentProjectDir
					}
				}
			}

			agents = append(agents, agent)
		}
	}

	// Batch fetch beads data (phase from comments, task from issues, close_reason for completed)
	// This is the same pattern used by orch status for efficiency.
	// Uses TTL cache to prevent CPU spikes from spawning 20+ bd processes per request.
	if len(beadsIDsToFetch) > 0 {
		// Batch fetch all issues (including closed) for close_reason (cached with TTL)
		// Uses bd show which works for any issue status
		allIssues, _ := globalBeadsCache.getAllIssues(beadsIDsToFetch, beadsProjectDirs)

		// Batch fetch comments for all beads IDs (cached with TTL)
		// Use project-aware batch fetch for cross-project agent visibility
		commentsMap := globalBeadsCache.getComments(beadsIDsToFetch, beadsProjectDirs)

		// Build investigation directory cache ONCE before the agent loop.
		// This prevents O(n^2) behavior: without this, discoverInvestigationPath would call
		// os.ReadDir() 2-3 times per agent, scanning 500+ files each time.
		// With 300+ agents, that's 300 x 500 x 2 = 300,000+ file comparisons.
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

		// Track PhaseReportedAt for completion backlog detection.
		// Populated during the enrichment loop below, used after to detect backlogged agents.
		phaseReportedAtMap := make(map[string]time.Time)

		// Populate phase, task, close_reason, and status for each agent using Priority Cascade model.
		// See .kb/investigations/2026-01-04-design-dashboard-agent-status-model.md for design.
		for i := range agents {
			if agents[i].BeadsID == "" {
				continue
			}

			// Get task from in-progress issue first (beads-first)
			if issue, ok := inProgressIssues[agents[i].BeadsID]; ok {
				agents[i].Task = truncate(issue.Title, 60)
				if agents[i].BeadsTitle == "" {
					agents[i].BeadsTitle = issue.Title
				}
			}

			// If not in in-progress issues, try all issues (for closed ones)
			if agents[i].Task == "" {
				if issue, ok := allIssues[agents[i].BeadsID]; ok {
					agents[i].Task = truncate(issue.Title, 60)
					if agents[i].BeadsTitle == "" {
						agents[i].BeadsTitle = issue.Title
					}
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

					// Track PhaseReportedAt for completion backlog detection and API response
					if phaseStatus.PhaseReportedAt != nil {
						phaseReportedAtMap[agents[i].BeadsID] = *phaseStatus.PhaseReportedAt
						agents[i].PhaseReportedAt = phaseStatus.PhaseReportedAt.Format(time.RFC3339)
					}

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
			// Uses invDirCache to avoid O(n^2) directory scanning (built once before this loop)
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

		// Completion backlog detection: find agents at Phase: Complete for >10 min
		// without orch complete being run. Write metric to coaching-metrics.jsonl.
		// Rate-limited to avoid writing on every dashboard poll (every 30s).
		emitCompletionBacklogMetrics(agents, phaseReportedAtMap)

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

	// Fetch token usage and last activity for agents with valid session IDs
	// Parallelized to avoid sequential HTTP calls causing ~20s delays with 200+ agents.
	// Uses goroutines with semaphore to limit concurrent requests.
	// Both tokens and activity are extracted from the same GetMessages call for efficiency.
	type sessionResult struct {
		index    int
		tokens   *opencode.TokenStats
		activity *opencode.LastActivity
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

			// Fetch messages once and extract both tokens and activity
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

			resultChan <- result
		}(i, agents[i].SessionID)
	}

	// Wait for all goroutines to complete, then close channel
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results and check for stall
	for result := range resultChan {
		if result.tokens != nil {
			agents[result.index].Tokens = result.tokens

			// Check for stall: agent running but no token progress for N minutes
			// Only check for active agents (skip idle/dead/completed)
			if agents[result.index].Status == "active" && agents[result.index].SessionID != "" {
				// Update stall tracker and check if agent is stalled
				isStalled := globalStallTracker.Update(agents[result.index].SessionID, result.tokens)

				// Set IsStalled if either phase-based OR token-based stall detected
				// Phase-based stall: same phase for 15+ minutes (already set earlier)
				// Token-based stall: no token progress for 3+ minutes (checked here)
				if isStalled {
					agents[result.index].IsStalled = true
				}
			}
		}
		if result.activity != nil {
			agents[result.index].CurrentActivity = result.activity.Text
			agents[result.index].LastActivityAt = time.Unix(result.activity.Timestamp/1000, 0).Format(time.RFC3339)
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

			// Project filter: check project name against all filters
			// Use Project field (derived from beads or workspace) instead of ProjectDir
			// because cross-project agents have ProjectDir=orchestrator-cwd, Project=target-project
			if len(projectFilterParam) > 0 {
				// Get project name to match
				projectName := agent.Project
				if projectName == "" && agent.ProjectDir != "" {
					// Fallback: extract project name from directory path
					projectName = extractProjectName(agent.ProjectDir)
				}

				// Check if project name matches ANY filter
				matched := false
				for _, filter := range projectFilterParam {
					filterName := extractProjectName(filter) // Handle both "orch-go" and "/path/to/orch-go"
					if projectName == filterName {
						matched = true
						break
					}
				}
				if !matched {
					continue
				}
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
