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

	"github.com/dylan-conlin/orch-go/pkg/discovery"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// handleAgents returns JSON list of agents using the single-pass query engine.
// Uses cached queryTrackedAgents for core discovery, then enriches with
// dashboard-specific data (tokens, activity, investigation, synthesis).
//
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

	projectDirs := uniqueProjectDirs(append([]string{projectDir}, getKBProjectsFn()...))

	// === Core discovery via cached queryTrackedAgents ===
	// Replaces the previous ad-hoc beads-first + workspace scan + session cross-referencing.
	// Cache TTL: 3s (balances freshness with performance for 30s dashboard poll interval).
	trackedAgents, err := globalTrackedAgentsCache.get(projectDirs)
	if err != nil {
		log.Printf("Warning: queryTrackedAgents failed: %v", err)
		trackedAgents = nil // Proceed with empty list - graceful degradation
	}

	now := time.Now()
	client := opencode.NewClient(serverURL)

	// Build session map for enrichment (runtime, processing status, tokens)
	sessions, _ := listSessionsAcrossProjects(client, projectDir)
	sessionByID := make(map[string]*opencode.Session)
	for i := range sessions {
		sessionByID[sessions[i].ID] = &sessions[i]
	}

	sessionStatusMap := make(map[string]opencode.SessionStatusInfo)
	if len(trackedAgents) > 0 {
		seenIDs := make(map[string]struct{}, len(trackedAgents))
		sessionIDs := make([]string, 0, len(trackedAgents))
		for _, tracked := range trackedAgents {
			if tracked.SessionID == "" {
				continue
			}
			if _, exists := seenIDs[tracked.SessionID]; exists {
				continue
			}
			seenIDs[tracked.SessionID] = struct{}{}
			sessionIDs = append(sessionIDs, tracked.SessionID)
		}
		if len(sessionIDs) > 0 {
			if status, err := client.GetSessionStatusByIDs(sessionIDs); err == nil {
				sessionStatusMap = status
			}
		}
	}

	// Workspace cache for enrichment (synthesis, investigation)
	wsCache := globalWorkspaceCacheInstance.getCachedWorkspace(projectDirs)

	// Stalled threshold for phase-based stall detection
	stalledThreshold := 15 * time.Minute
	// Unresponsive threshold: no phase update for 30+ minutes
	unresponsiveThreshold := 30 * time.Minute

	// Convert tracked agents to API response format
	agents := make([]AgentAPIResponse, 0, len(trackedAgents))
	beadsProjectDirs := make(map[string]string)
	beadsIDsToFetch := make([]string, 0, len(trackedAgents))

	for _, tracked := range trackedAgents {
		agent := agentStatusToAPIResponse(tracked)

		// Enrich from OpenCode session (runtime, processing, timestamps)
		if tracked.SessionID != "" {
			if s, ok := sessionByID[tracked.SessionID]; ok {
				createdAt := time.Unix(s.Time.Created/1000, 0)
				updatedAt := time.Unix(s.Time.Updated/1000, 0)
				agent.Runtime = formatDuration(now.Sub(createdAt))
				agent.UpdatedAt = updatedAt.Format(time.RFC3339)
				if agent.SpawnedAt == "" {
					agent.SpawnedAt = createdAt.Format(time.RFC3339)
				}
				if agent.ID == "" {
					agent.ID = workspaceNameFromSession(s)
				}

				if statusInfo, ok := sessionStatusMap[s.ID]; ok {
					agent.IsProcessing = statusInfo.IsBusy() || statusInfo.IsRetrying()
				}
			}
		}

		// Track project dir for beads enrichment
		if tracked.ProjectDir != "" {
			beadsProjectDirs[tracked.BeadsID] = tracked.ProjectDir
		}
		beadsIDsToFetch = append(beadsIDsToFetch, tracked.BeadsID)

		agents = append(agents, agent)
	}

	// Claude IsProcessing override: use live tmux pane signal instead of static phase-based status.
	// Mirrors the OpenCode session API override above, but for Claude-backend agents.
	for i := range agents {
		if agents[i].Status == "completed" || agents[i].Status == "dead" {
			continue
		}
		// Only override Claude agents (OpenCode agents already overridden above via session API)
		tracked := trackedAgents[i]
		if tracked.SpawnMode != "claude" || tracked.WorkspaceName == "" {
			continue
		}
		windowID, alive := discovery.CheckTmuxWindow(tracked.WorkspaceName, tracked.ProjectDir)
		if !alive {
			agents[i].IsProcessing = false
			continue
		}
		agents[i].IsProcessing = discovery.CheckPaneActive(windowID)
	}

	// === Dashboard-specific enrichment ===
	// This adds data that queryTrackedAgents doesn't provide:
	// phase timestamps, investigation paths, synthesis content, close_reason, gap analysis.

	if len(beadsIDsToFetch) > 0 {
		allIssues, _ := globalBeadsCache.getAllIssues(beadsIDsToFetch, beadsProjectDirs)
		commentsMap := globalBeadsCache.getComments(beadsIDsToFetch, beadsProjectDirs)

		// Build investigation directory cache once
		invProjectDirs := make([]string, 0, len(beadsProjectDirs))
		seenDirs := make(map[string]bool)
		for _, dir := range beadsProjectDirs {
			if dir != "" && !seenDirs[dir] {
				seenDirs[dir] = true
				invProjectDirs = append(invProjectDirs, dir)
			}
		}
		invDirCache := buildInvestigationDirCache(invProjectDirs)

		phaseReportedAtMap := make(map[string]time.Time)

		for i := range agents {
			if agents[i].BeadsID == "" {
				continue
			}

			// Enrich task from issues
			if issue, ok := allIssues[agents[i].BeadsID]; ok {
				if agents[i].Task == "" {
					agents[i].Task = truncate(issue.Title, 60)
				}
				if agents[i].BeadsTitle == "" {
					agents[i].BeadsTitle = issue.Title
				}
				if strings.EqualFold(issue.Status, "closed") && issue.CloseReason != "" {
					agents[i].CloseReason = issue.CloseReason
				}
			}

			// Enrich phase from comments (for PhaseReportedAt, stall detection, investigation paths)
			if comments, ok := commentsMap[agents[i].BeadsID]; ok {
				phaseStatus := verify.ParsePhaseFromComments(comments)
				if phaseStatus.Found {
					// queryTrackedAgents already set Phase, but we need PhaseReportedAt for the dashboard
					if phaseStatus.PhaseReportedAt != nil {
						phaseReportedAtMap[agents[i].BeadsID] = *phaseStatus.PhaseReportedAt
						agents[i].PhaseReportedAt = phaseStatus.PhaseReportedAt.Format(time.RFC3339)
					}

					// Stalled detection (15+ min without phase update)
					if agents[i].Status == "active" && phaseStatus.PhaseReportedAt != nil {
						elapsed := now.Sub(*phaseStatus.PhaseReportedAt)
						if elapsed > stalledThreshold {
							agents[i].IsStalled = true
						}
						// Unresponsive detection (30+ min without phase update).
						// Skip if agent is actively processing (confirmed by OpenCode session) —
						// an agent generating a response is not unresponsive.
						if elapsed > unresponsiveThreshold && !agents[i].IsProcessing {
							agents[i].IsUnresponsive = true
						}
					}
				}

				// Extract investigation_path from comments
				if investigationPath := verify.ParseInvestigationPathFromComments(comments); investigationPath != "" {
					agents[i].InvestigationPath = investigationPath
				}
			}

			// Never-started detection: agents with spawn_time but no phase
			if agents[i].Reason == "never_started" {
				agents[i].IsNeverStarted = true
				agents[i].IsStalled = true
				agents[i].IsUnresponsive = true
			} else if agents[i].Phase == "" && agents[i].SpawnedAt != "" {
				if spawnedAt, err := time.Parse(time.RFC3339, agents[i].SpawnedAt); err == nil {
					elapsed := now.Sub(spawnedAt)
					if elapsed > stalledThreshold {
						agents[i].IsStalled = true
					}
					if elapsed > unresponsiveThreshold && !agents[i].IsProcessing {
						agents[i].IsUnresponsive = true
					}
				}
			}

			// Auto-discover investigation path
			hasReliableProjectDir := agents[i].ProjectDir != ""
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

			// Read investigation file content
			if agents[i].InvestigationPath != "" {
				if content, err := os.ReadFile(agents[i].InvestigationPath); err == nil {
					agents[i].InvestigationContent = string(content)
				}
			}

			// Determine final status using Priority Cascade model
			issueClosed := false
			phaseComplete := false
			if issue, ok := allIssues[agents[i].BeadsID]; ok {
				issueClosed = strings.EqualFold(issue.Status, "closed")
			}
			phaseComplete = strings.HasPrefix(agents[i].Phase, "Complete")

			workspacePath := wsCache.lookupWorkspace(agents[i].BeadsID)
			if workspacePath == "" && agents[i].ID != "" {
				workspaceName := agents[i].ID
				if idx := strings.Index(workspaceName, " ["); idx != -1 {
					workspaceName = workspaceName[:idx]
				}
				workspacePath = wsCache.lookupWorkspacePathByEntry(workspaceName)
			}

			agents[i].Status = determineAgentStatus(issueClosed, phaseComplete, workspacePath, agents[i].Status)

			// Read synthesis content
			if workspacePath != "" && agents[i].SynthesisContent == "" {
				synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
				if content, err := os.ReadFile(synthesisPath); err == nil {
					agents[i].SynthesisContent = string(content)
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

			// Compute server-side escalation level for completed agents.
			// Uses DetermineEscalation with available data (avoids expensive
			// DetermineEscalationFromCompletion which shells out to git).
			if phaseComplete && agents[i].Synthesis != nil {
				input := verify.EscalationInput{
					VerificationPassed: true, // Phase: Complete implies verification ran
					SkillName:          agents[i].Skill,
					Outcome:            strings.ToLower(agents[i].Synthesis.Outcome),
					Recommendation:     strings.ToLower(agents[i].Synthesis.Recommendation),
					NextActions:        agents[i].Synthesis.NextActions,
				}
				agents[i].EscalationLevel = verify.DetermineEscalation(input).String()
			}
		}

		// Gap analysis from spawn events
		gapAnalysisMap := getGapAnalysisFromEvents(beadsIDsToFetch)
		for i := range agents {
			if agents[i].BeadsID == "" {
				continue
			}
			if gapData, ok := gapAnalysisMap[agents[i].BeadsID]; ok {
				agents[i].GapAnalysis = gapData
			}
		}

		// Completion backlog metrics (rate-limited)
		emitCompletionBacklogMetrics(agents, phaseReportedAtMap)
	}

	// Add completed workspaces (those with SYNTHESIS.md) not already in tracked agents
	seenBeadsIDs := make(map[string]bool)
	for _, a := range agents {
		if a.BeadsID != "" {
			seenBeadsIDs[a.BeadsID] = true
		}
	}

	if len(wsCache.workspaceEntries) > 0 {
		for _, entry := range wsCache.workspaceEntries {
			if !entry.IsDir() {
				continue
			}

			workspacePath := wsCache.lookupWorkspacePathByEntry(entry.Name())
			wsBeadsID := extractBeadsIDFromWorkspace(workspacePath)
			if wsBeadsID == "" {
				wsBeadsID = extractBeadsIDFromTitle(entry.Name())
			}

			// Skip if already tracked
			if wsBeadsID != "" && seenBeadsIDs[wsBeadsID] {
				continue
			}
			// Also skip by name
			alreadyIn := false
			for _, a := range agents {
				if a.ID == entry.Name() {
					alreadyIn = true
					break
				}
			}
			if alreadyIn {
				continue
			}

			synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
			if _, err := os.Stat(synthesisPath); err != nil {
				continue // Only add completed workspaces with SYNTHESIS.md
			}

			agent := AgentAPIResponse{
				ID:      entry.Name(),
				BeadsID: wsBeadsID,
				Status:  "completed",
				Skill:   extractSkillFromTitle(entry.Name()),
			}

			if parsedDate := extractDateFromWorkspaceName(entry.Name()); !parsedDate.IsZero() {
				agent.UpdatedAt = parsedDate.Format(time.RFC3339)
			} else if info, err := os.Stat(synthesisPath); err == nil {
				agent.UpdatedAt = info.ModTime().Format(time.RFC3339)
			}

			if sessionID := spawn.ReadSessionID(workspacePath); sessionID != "" {
				agent.SessionID = sessionID
			}
			if spawnTime := spawn.ReadSpawnTime(workspacePath); !spawnTime.IsZero() {
				agent.SpawnedAt = spawnTime.Format(time.RFC3339)
			}

			if synthesis, err := verify.ParseSynthesis(workspacePath); err == nil {
				agent.Synthesis = &SynthesisResponse{
					TLDR:           synthesis.TLDR,
					Outcome:        synthesis.Outcome,
					Recommendation: synthesis.Recommendation,
					DeltaSummary:   summarizeDelta(synthesis.Delta),
					NextActions:    synthesis.NextActions,
				}
				// Compute escalation for workspace-only completed agents
				input := verify.EscalationInput{
					VerificationPassed: true,
					SkillName:          agent.Skill,
					Outcome:            strings.ToLower(synthesis.Outcome),
					Recommendation:     strings.ToLower(synthesis.Recommendation),
					NextActions:        synthesis.NextActions,
				}
				agent.EscalationLevel = verify.DetermineEscalation(input).String()
			}
			if content, err := os.ReadFile(synthesisPath); err == nil {
				agent.SynthesisContent = string(content)
			}

			agents = append(agents, agent)
		}
	}

	// Fetch token usage and last activity (parallelized)
	type sessionResult struct {
		index    int
		tokens   *opencode.TokenStats
		activity *opencode.LastActivity
	}
	resultChan := make(chan sessionResult, len(agents))
	const maxConcurrent = 20
	sem := make(chan struct{}, maxConcurrent)

	var wg sync.WaitGroup
	for i := range agents {
		if agents[i].SessionID == "" || agents[i].Status == "completed" || agents[i].Status == "idle" {
			continue
		}
		wg.Add(1)
		go func(idx int, sessionID string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			messages, err := client.GetMessages(sessionID)
			if err != nil || len(messages) == 0 {
				return
			}
			result := sessionResult{index: idx}
			tokenStats := opencode.AggregateTokens(messages)
			result.tokens = &tokenStats
			result.activity = extractLastActivityFromMessages(messages)
			resultChan <- result
		}(i, agents[i].SessionID)
	}
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for result := range resultChan {
		if result.tokens != nil {
			agents[result.index].Tokens = result.tokens
			if agents[result.index].Status == "active" && agents[result.index].SessionID != "" {
				isStalled := globalStallTracker.Update(agents[result.index].SessionID, result.tokens)
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

	// Assess context exhaustion risk for agents with token data.
	// This matches what orch status --all shows (AT-RISK, CRITICAL flags).
	for i := range agents {
		if agents[i].Tokens == nil || agents[i].Status == "completed" {
			continue
		}
		totalTokens := agents[i].Tokens.TotalTokens
		if totalTokens == 0 {
			totalTokens = agents[i].Tokens.InputTokens + agents[i].Tokens.OutputTokens
		}
		if totalTokens == 0 {
			continue
		}
		risk := verify.AssessContextRisk(totalTokens, agents[i].ProjectDir, agents[i].IsProcessing)
		if risk.IsAtRisk() {
			agents[i].ContextRisk = &risk
		}
	}

	// Apply time and project filters.
	// Agents needing attention (dead, awaiting-cleanup, at-risk) bypass the time filter
	// so they are always visible in the dashboard regardless of the ?since= parameter.
	// This fixes the bug where idle/at-risk agents were hidden by the default 12h filter.
	if sinceDuration > 0 || len(projectFilterParam) > 0 {
		filtered := make([]AgentAPIResponse, 0, len(agents))
		for _, agentItem := range agents {
			// Agents that need attention bypass the time filter (but not project filter).
			// These are the agents most likely to need the orchestrator's intervention.
			needsAttention := agentItem.Status == "dead" ||
				agentItem.Status == "awaiting-cleanup" ||
				agentItem.IsNeverStarted ||
				agentItem.ContextRisk != nil

			if sinceDuration > 0 && !needsAttention {
				var agentTime time.Time
				if agentItem.UpdatedAt != "" {
					agentTime, _ = time.Parse(time.RFC3339, agentItem.UpdatedAt)
				} else if agentItem.SpawnedAt != "" {
					agentTime, _ = time.Parse(time.RFC3339, agentItem.SpawnedAt)
				}
				if !agentTime.IsZero() && !filterByTime(agentTime, sinceDuration) {
					continue
				}
			}

			if len(projectFilterParam) > 0 {
				projectName := agentItem.Project
				if projectName == "" && agentItem.ProjectDir != "" {
					projectName = extractProjectName(agentItem.ProjectDir)
				}
				matched := false
				for _, filter := range projectFilterParam {
					filterName := extractProjectName(filter)
					if projectName == filterName {
						matched = true
						break
					}
				}
				if !matched {
					continue
				}
			}

			filtered = append(filtered, agentItem)
		}
		agents = filtered
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(agents); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode agents: %v", err), http.StatusInternalServerError)
		return
	}
}

// agentStatusToAPIResponse converts a queryTrackedAgents result to the dashboard API type.
// Preserves the raw status from the query engine — status mapping is handled by
// determineAgentStatus which is the single canonical status derivation point.
//
// This separation prevents Class 5 (Contradictory Authority Signals) defects by
// ensuring status is only mapped once, by determineAgentStatus, with full context
// (beads issue closed, phase complete, synthesis artifact, session status).
func agentStatusToAPIResponse(tracked discovery.AgentStatus) AgentAPIResponse {
	resp := AgentAPIResponse{
		BeadsID:    tracked.BeadsID,
		BeadsTitle: tracked.Title,
		Task:       truncate(tracked.Title, 60),
		SessionID:  tracked.SessionID,
		ProjectDir: tracked.ProjectDir,
		Skill:      tracked.Skill,
		Tier:       tracked.Tier,
		Project:    extractProjectFromBeadsID(tracked.BeadsID),
		Phase:      tracked.Phase,
	}

	if tracked.WorkspaceName != "" {
		resp.ID = tracked.WorkspaceName
	}

	// Pass raw status through — determineAgentStatus handles all mapping.
	// Exception: when the discovery layer has flagged SessionDead, override
	// to "dead" so determineAgentStatus sees the correct session state.
	// This is not status mapping — it's resolving a contradiction between
	// Status (which may be stale) and SessionDead (the ground truth signal).
	resp.Status = tracked.Status
	if tracked.SessionDead && tracked.Status != "dead" {
		resp.Status = "dead"
	}
	switch tracked.Status {
	case "active":
		resp.IsProcessing = true
	case "retrying":
		resp.IsProcessing = true
	}

	// Surface reason code for partial metadata
	if tracked.Reason != "" {
		resp.Reason = tracked.Reason
	}

	return resp
}
