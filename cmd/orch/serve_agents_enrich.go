package main

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/agent"
	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// enrichAgentsWithBeadsData populates phase, task, close_reason, investigation paths,
// synthesis, and status for each agent using the Priority Cascade model.
// Also applies ghost filtering for idle agents.
func (ctx *agentCollectionContext) enrichAgentsWithBeadsData() {
	if len(ctx.beadsIDsToFetch) == 0 {
		return
	}

	// Fetch all open issues in one call (cached with TTL)
	openIssues, _ := ctx.beadsCache.getOpenIssues()

	// Batch fetch all issues (including closed) for close_reason (cached with TTL)
	allIssues, _ := ctx.beadsCache.getAllIssues(ctx.beadsIDsToFetch)

	// Batch fetch comments for all beads IDs (cached with TTL)
	commentsMap := ctx.beadsCache.getComments(ctx.beadsIDsToFetch, ctx.beadsProjectDirs)

	// Build investigation directory cache ONCE before the agent loop.
	// Prevents O(n²) behavior: without this, discoverInvestigationPath would call
	// os.ReadDir() 2-3 times per agent, scanning 500+ files each time.
	invDirCache := ctx.buildInvDirCache()

	// Enrich each agent with beads data
	for i := range ctx.agents {
		if ctx.agents[i].BeadsID == "" {
			continue
		}
		ctx.enrichSingleAgent(i, openIssues, allIssues, commentsMap, invDirCache)
	}

	// Fetch gap analysis from spawn events
	gapAnalysisMap := getGapAnalysisFromEvents(ctx.beadsIDsToFetch)
	for i := range ctx.agents {
		if ctx.agents[i].BeadsID == "" {
			continue
		}
		if gapData, ok := gapAnalysisMap[ctx.agents[i].BeadsID]; ok {
			ctx.agents[i].GapAnalysis = gapData
		}
	}

	// Post-Phase ghost filtering
	ctx.applyGhostFilter()
}

// buildInvDirCache creates an investigation directory cache from unique project directories.
func (ctx *agentCollectionContext) buildInvDirCache() *investigationDirCache {
	uniqueProjectDirs := make([]string, 0, len(ctx.beadsProjectDirs))
	seenDirs := make(map[string]bool)
	for _, dir := range ctx.beadsProjectDirs {
		if dir != "" && !seenDirs[dir] {
			seenDirs[dir] = true
			uniqueProjectDirs = append(uniqueProjectDirs, dir)
		}
	}
	return buildInvestigationDirCache(uniqueProjectDirs)
}

// enrichSingleAgent populates beads data for a single agent using Priority Cascade model.
func (ctx *agentCollectionContext) enrichSingleAgent(
	i int,
	openIssues map[string]*verify.Issue,
	allIssues map[string]*verify.Issue,
	commentsMap map[string][]beads.Comment,
	invDirCache *investigationDirCache,
) {
	beadsID := ctx.agents[i].BeadsID

	// Get task from open issue title first
	if issue, ok := openIssues[beadsID]; ok {
		ctx.agents[i].Task = truncate(issue.Title, 60)
		ctx.agents[i].BeadsLabels = issue.Labels
	}

	// If not in open issues, try all issues (for closed ones)
	if ctx.agents[i].Task == "" {
		if issue, ok := allIssues[beadsID]; ok {
			ctx.agents[i].Task = truncate(issue.Title, 60)
			ctx.agents[i].BeadsLabels = issue.Labels
			if ctx.agents[i].Synthesis == nil && issue.CloseReason != "" {
				ctx.agents[i].CloseReason = issue.CloseReason
			}
		}
	}

	// Gather completion signals for Priority Cascade model
	issueClosed := false
	phaseComplete := false

	// Check if beads issue is closed (Priority 1)
	if issue, ok := allIssues[beadsID]; ok {
		issueClosed = strings.EqualFold(issue.Status, "closed")
		if issueClosed && issue.CloseReason != "" && ctx.agents[i].CloseReason == "" {
			ctx.agents[i].CloseReason = issue.CloseReason
		}
	}

	// Get phase from comments (Priority 2)
	if comments, ok := commentsMap[beadsID]; ok {
		phaseStatus := verify.ParsePhaseFromComments(comments)
		if phaseStatus.Found {
			ctx.agents[i].Phase = phaseStatus.Phase
			phaseComplete = strings.EqualFold(phaseStatus.Phase, "Complete")

			// Stalled detection: advisory only
			if ctx.agents[i].Status == "active" && phaseStatus.PhaseReportedAt != nil {
				timeSincePhase := ctx.now.Sub(*phaseStatus.PhaseReportedAt)
				if timeSincePhase > ctx.stalledThreshold {
					ctx.agents[i].IsStalled = true
				}
			}
		}
		// Extract investigation_path from comments
		if investigationPath := verify.ParseInvestigationPathFromComments(comments); investigationPath != "" {
			ctx.agents[i].InvestigationPath = investigationPath
		}
	}

	// Populate project_dir from beadsProjectDirs lookup
	hasReliableProjectDir := false
	if agentProjectDir, ok := ctx.beadsProjectDirs[beadsID]; ok {
		ctx.agents[i].ProjectDir = agentProjectDir
		hasReliableProjectDir = true
	}

	// Auto-discover investigation path if not provided via beads comment
	if ctx.agents[i].InvestigationPath == "" && hasReliableProjectDir {
		workspaceName := ctx.agents[i].ID
		if idx := strings.Index(workspaceName, " ["); idx != -1 {
			workspaceName = workspaceName[:idx]
		}
		discoveredPath := discoverInvestigationPath(workspaceName, beadsID, ctx.agents[i].ProjectDir, invDirCache)
		if discoveredPath != "" {
			ctx.agents[i].InvestigationPath = discoveredPath
		}
	}

	// Read investigation file content for inline rendering
	if ctx.agents[i].InvestigationPath != "" {
		if content, err := os.ReadFile(ctx.agents[i].InvestigationPath); err == nil {
			ctx.agents[i].InvestigationContent = string(content)
		}
	}

	// Get workspace path for SYNTHESIS.md check (Priority 3)
	workspacePath := ctx.wsCache.lookupWorkspace(beadsID)
	if workspacePath == "" && ctx.agents[i].ID != "" {
		workspaceName := ctx.agents[i].ID
		if idx := strings.Index(workspaceName, " ["); idx != -1 {
			workspaceName = workspaceName[:idx]
		}
		workspacePath = ctx.wsCache.lookupWorkspacePathByEntry(workspaceName)
	}

	// Use Priority Cascade to determine final status
	ctx.agents[i].Status = determineAgentStatus(issueClosed, phaseComplete, workspacePath, ctx.agents[i].Status)

	// For completed agents, also check close_reason if synthesis is null
	if ctx.agents[i].Status == "completed" && ctx.agents[i].Synthesis == nil && ctx.agents[i].CloseReason == "" {
		if issue, ok := allIssues[beadsID]; ok && issue.CloseReason != "" {
			ctx.agents[i].CloseReason = issue.CloseReason
		}
	}

	// Read synthesis content for active agents with workspace SYNTHESIS.md
	if workspacePath != "" && ctx.agents[i].SynthesisContent == "" {
		synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
		if content, err := os.ReadFile(synthesisPath); err == nil {
			ctx.agents[i].SynthesisContent = string(content)
			if ctx.agents[i].Synthesis == nil {
				if synthesis, err := verify.ParseSynthesis(workspacePath); err == nil {
					ctx.agents[i].Synthesis = &SynthesisResponse{
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

// applyGhostFilter removes agents using two-threshold ghost filtering.
// Ensures Phase: Complete agents are always visible.
func (ctx *agentCollectionContext) applyGhostFilter() {
	filtered := make([]AgentAPIResponse, 0, len(ctx.agents))
	for _, agentItem := range ctx.agents {
		// Determine status for filtering
		status := agentItem.Status
		if status != "active" && status != "idle" && status != "completed" {
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
		if ctx.pendingFilterByIDs[agentItem.BeadsID] {
			if !agent.IsVisibleByDefault(status, lastActivity, agentItem.Phase) {
				continue
			}
		}

		filtered = append(filtered, agentItem)
	}
	ctx.agents = filtered
}

// fetchTokensAndActivity fetches token usage, last activity, and model for agents
// with valid session IDs. Parallelized to avoid sequential HTTP calls.
func (ctx *agentCollectionContext) fetchTokensAndActivity() {
	type sessionResult struct {
		index    int
		tokens   *opencode.TokenStats
		activity *opencode.LastActivity
		model    string
	}
	resultChan := make(chan sessionResult, len(ctx.agents))

	// Limit concurrent HTTP requests to avoid overwhelming the OpenCode server
	const maxConcurrent = 20
	sem := make(chan struct{}, maxConcurrent)

	var wg sync.WaitGroup
	for i := range ctx.agents {
		// Skip agents without session ID, completed agents, or idle agents.
		if ctx.agents[i].SessionID == "" || ctx.agents[i].Status == "completed" || ctx.agents[i].Status == "idle" {
			continue
		}

		wg.Add(1)
		go func(idx int, sessionID string) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire semaphore
			defer func() { <-sem }() // Release semaphore

			messages, err := ctx.client.GetMessages(sessionID)
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
		}(i, ctx.agents[i].SessionID)
	}

	// Wait for all goroutines then close channel
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	for result := range resultChan {
		if result.tokens != nil {
			ctx.agents[result.index].Tokens = result.tokens
		}
		if result.activity != nil {
			ctx.agents[result.index].CurrentActivity = result.activity.Text
			ctx.agents[result.index].LastActivityAt = time.Unix(result.activity.Timestamp/1000, 0).Format(time.RFC3339)
		}
		if result.model != "" {
			ctx.agents[result.index].Model = result.model
		}
	}
}

// applyLateFilters applies time and project filters for non-session agents.
// OpenCode sessions are filtered early; this catches tmux agents and completed workspaces.
func (ctx *agentCollectionContext) applyLateFilters(sinceDuration time.Duration, projectFilterParam []string) {
	if sinceDuration == 0 && len(projectFilterParam) == 0 {
		return
	}

	filtered := make([]AgentAPIResponse, 0, len(ctx.agents))
	for _, agentItem := range ctx.agents {
		// Time filter
		if sinceDuration > 0 {
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

		// Project filter
		if len(projectFilterParam) > 0 && !matchAgentProject(agentItem.Project, agentItem.ProjectDir, projectFilterParam) {
			continue
		}

		filtered = append(filtered, agentItem)
	}
	ctx.agents = filtered
}
