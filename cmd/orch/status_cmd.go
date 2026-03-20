// Package main provides the status command for showing swarm status and active agents.
// Extracted from main.go as part of the main.go refactoring (Phase 3).
// Terminal formatting functions extracted to status_format.go (orch-go-vp594).
// Infrastructure health checking extracted to status_infra.go (orch-go-vp594).
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/account"
	"github.com/dylan-conlin/orch-go/pkg/discovery"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/session"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/dylan-conlin/orch-go/pkg/workspace"
	"github.com/spf13/cobra"
)

var (
	// Status command flags
	statusJSON    bool
	statusAll     bool   // Include all agents (default: compact mode showing running/recent only)
	statusProject string // Filter by project
)

// Compact mode thresholds
const (
	// Only show Phase: Complete agents from the last N hours in compact mode
	compactCompletedAgentsMaxAge = 6 * time.Hour
	// Only fetch processing status for sessions updated within this window
	processingCheckMaxAge = 5 * time.Minute
	// Treat session metrics as stale if no activity in this window
	sessionMetricsStaleAfter = session.DefaultInactivityTimeout
	// Flag agents as unresponsive if no phase update in this window
	phaseTimeoutThreshold = 30 * time.Minute
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show swarm status and active agents",
	Long: `Show swarm status including active/queued/completed agent counts,
per-account usage percentages, and individual agent details.

By default, uses compact mode showing only running agents and recent sessions.
This provides faster execution (<2s) with essential information.

Use --all to include all agents (idle, phantom, completed) and full session list.

Examples:
  orch status              # Compact: running agents only, fast
  orch status --all        # Full: all agents and sessions
  orch status --project snap  # Filter by project
  orch status --json       # Output as JSON for scripting`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStatus(serverURL)
	},
}

func init() {
	statusCmd.Flags().BoolVar(&statusJSON, "json", false, "Output as JSON for scripting")
	statusCmd.Flags().BoolVar(&statusAll, "all", false, "Show all agents and sessions (default: compact mode)")
	statusCmd.Flags().StringVar(&statusProject, "project", "", "Filter by project")
}

// SwarmStatus represents aggregate swarm information.
type SwarmStatus struct {
	Active     int `json:"active"`
	Processing int `json:"processing,omitempty"` // Agents actively generating response
	Idle       int `json:"idle,omitempty"`       // Agents with session but not processing
	Phantom    int `json:"phantom,omitempty"`    // Agents with open beads issue but not running
	Queued     int `json:"queued"`
	Completed  int `json:"completed_today"`
}

// AccountUsage represents usage info for a single account.
type AccountUsage struct {
	Name        string  `json:"name"`
	Email       string  `json:"email,omitempty"`
	UsedPercent float64 `json:"used_percent"`
	ResetTime   string  `json:"reset_time,omitempty"`
	IsActive    bool    `json:"is_active"`
}

// AgentInfo represents information about an active agent.
type AgentInfo struct {
	SessionID       string                        `json:"session_id"`
	BeadsID         string                        `json:"beads_id,omitempty"`
	Mode            string                        `json:"mode,omitempty"`  // Agent mode: "claude" or "opencode"
	Model           string                        `json:"model,omitempty"` // Model spec (e.g., "gemini-3-flash-preview", "claude-opus-4-5-20251101")
	Skill           string                        `json:"skill,omitempty"`
	Account         string                        `json:"account,omitempty"`
	Runtime         string                        `json:"runtime"`
	Title           string                        `json:"title,omitempty"`
	Window          string                        `json:"window,omitempty"`
	Phase           string                        `json:"phase,omitempty"`             // Current phase from beads comments
	Task            string                        `json:"task,omitempty"`              // Task description (truncated)
	Project         string                        `json:"project,omitempty"`           // Project name derived from beads ID or workspace
	ProjectDir      string                        `json:"project_dir,omitempty"`       // Full path to project directory (for cross-project agents)
	Source          string                        `json:"source,omitempty"`            // Source where agent originated: T=tmux, O=opencode, B=beads, W=workspace
	IsPhantom       bool                          `json:"is_phantom,omitempty"`        // True if beads issue open but agent not running
	IsProcessing    bool                          `json:"is_processing,omitempty"`     // True if session is actively generating a response
	IsCompleted     bool                          `json:"is_completed,omitempty"`      // True if beads issue is closed
	IsStalled       bool                          `json:"is_stalled,omitempty"`        // True if no token progress for 3+ minutes
	IsUnresponsive  bool                          `json:"is_unresponsive,omitempty"`   // True if no phase update for 30+ minutes
	Tokens          *opencode.TokenStats          `json:"tokens,omitempty"`            // Token usage for the session
	ContextRisk     *verify.ContextExhaustionRisk `json:"context_risk,omitempty"`      // Context exhaustion risk assessment
	PhaseReportedAt *time.Time                    `json:"phase_reported_at,omitempty"` // Timestamp when latest phase was reported
	LastActivity    time.Time                     `json:"last_activity,omitempty"`     // Timestamp of last activity (for ghost filtering)
}

// SessionMetrics represents drift detection metrics for the current orchestrator session.
// These metrics surface to Dylan so he can detect when the orchestrator is drifting
// (e.g., spending too long reading files instead of spawning agents).
type SessionMetrics struct {
	// TimeInSession is how long the current session has been running
	TimeInSession string `json:"time_in_session"`

	// TimeSinceLastSpawn is how long since the last agent was spawned
	// Empty string if no spawns in session or no active session
	TimeSinceLastSpawn string `json:"time_since_last_spawn,omitempty"`

	// SpawnCount is the number of agents spawned in the current session
	SpawnCount int `json:"spawn_count"`

	// HasActiveSession indicates if an orchestrator session is currently active
	HasActiveSession bool `json:"has_active_session"`

	// Goal is the current session's focus goal
	Goal string `json:"goal,omitempty"`

	// FileReadsSinceLastSpawn is the count of file reads since last spawn
	// NOTE: Not yet implemented - requires event tracking infrastructure
	FileReadsSinceLastSpawn int `json:"file_reads_since_last_spawn,omitempty"`
}

// ReviewQueueStatus holds the count of completions awaiting human review.
type ReviewQueueStatus struct {
	Ready int `json:"ready"` // Non-stale completions ready for orch review/complete
}

// StatusOutput represents the full status output for JSON serialization.
type StatusOutput struct {
	Infrastructure         *InfrastructureHealth          `json:"infrastructure,omitempty"`
	SessionMetrics         *SessionMetrics                `json:"session_metrics,omitempty"`
	Swarm                  SwarmStatus                    `json:"swarm"`
	ReviewQueue            *ReviewQueueStatus             `json:"review_queue,omitempty"`
	Accounts               []AccountUsage                 `json:"accounts"`
	Agents                 []AgentInfo                    `json:"agents"`
	SynthesisOpportunities *verify.SynthesisOpportunities `json:"synthesis_opportunities,omitempty"`
}

// isSessionLikelyProcessing checks if a session might be processing based on its last update time.
// Only makes the expensive IsSessionProcessing HTTP call for recently updated sessions.
// For sessions not updated recently, assumes they are idle (saves ~100ms per call).
func isSessionLikelyProcessing(client *opencode.Client, sessionID string, lastUpdated time.Time, now time.Time) bool {
	// If the session hasn't been updated in the last 5 minutes, it's definitely not processing
	if now.Sub(lastUpdated) > processingCheckMaxAge {
		return false
	}
	// For recently active sessions, make the HTTP call to check processing status
	return client.IsSessionProcessing(sessionID)
}

func runStatus(serverURL string) error {
	client := opencode.NewClient(serverURL)
	now := time.Now()

	// Fast connectivity check: probe OpenCode server before making HTTP calls.
	// When unreachable, skip all OpenCode enrichment to avoid multiple 10s timeouts.
	opencodeReachable := client.IsReachable()

	// Get current project directory
	projectDir, _ := os.Getwd()

	// Build project dirs list for cross-project visibility
	projectDirs := uniqueProjectDirs(append([]string{projectDir}, getKBProjectsFn()...))

	// === Core discovery via queryTrackedAgents (single-pass query engine) ===
	// This replaces the previous 3-phase ad-hoc discovery (workspace scan, tmux, opencode sessions).
	// See query_tracked.go for the implementation.
	trackedAgents, err := queryTrackedAgents(projectDirs)
	if err != nil {
		return fmt.Errorf("failed to query tracked agents: %w", err)
	}

	// Convert AgentStatus -> AgentInfo for display
	agents := make([]AgentInfo, 0, len(trackedAgents))
	for _, tracked := range trackedAgents {
		info := agentStatusToAgentInfo(tracked, now)
		agents = append(agents, info)
	}

	// Enrich with session data for runtime/processing status.
	// Skip when OpenCode is unreachable — local discovery still provides agent list.
	sessionMap := make(map[string]*opencode.Session)
	sessionStatusMap := make(map[string]opencode.SessionStatusInfo)

	if opencodeReachable {
		sessions, _ := listSessionsAcrossProjects(client, projectDir)
		for i := range sessions {
			sessionMap[sessions[i].ID] = &sessions[i]
		}

		statusIDs := make([]string, 0, len(agents))
		for _, a := range agents {
			if a.SessionID != "" {
				statusIDs = append(statusIDs, a.SessionID)
			}
		}
		if len(statusIDs) > 0 {
			if status, err := client.GetSessionStatusByIDs(statusIDs); err == nil {
				sessionStatusMap = status
			}
		}
	}

	for i := range agents {
		agent := &agents[i]
		if agent.SessionID == "" {
			continue
		}
		if s, ok := sessionMap[agent.SessionID]; ok {
			createdAt := time.Unix(s.Time.Created/1000, 0)
			updatedAt := time.Unix(s.Time.Updated/1000, 0)
			agent.Runtime = formatDuration(now.Sub(createdAt))
			agent.LastActivity = updatedAt
			// Don't override processing status for completed agents — their session
			// may still be technically alive, but the agent has reported Phase: Complete.
			if !agent.IsCompleted {
				if statusInfo, ok := sessionStatusMap[s.ID]; ok {
					agent.IsProcessing = statusInfo.IsBusy() || statusInfo.IsRetrying()
				}
			}
		}
		if agent.Runtime == "" {
			agent.Runtime = "unknown"
		}
	}

	// Determine phantom and completed status from the query engine's reason codes
	for i := range agents {
		agent := &agents[i]
		agent.Source = determineAgentSource(*agent, projectDir)
	}

	// Filter agents based on flags
	filteredAgents := make([]AgentInfo, 0)
	for _, agentItem := range agents {
		// Filter by project if specified
		if statusProject != "" && agentItem.Project != statusProject {
			continue
		}

		// In compact mode, only show:
		// 1. Running (processing) agents
		// 2. RECENT agents with Phase: Complete (need review)
		// 3. Agents with Phase: BLOCKED or QUESTION (need attention)
		// 4. Unresponsive agents (no phase update in 30+ minutes)
		if !statusAll {
			isRunning := agentItem.IsProcessing

			isComplete := strings.HasPrefix(agentItem.Phase, "Complete")
			isRecent := true
			if isComplete && agentItem.PhaseReportedAt != nil {
				if time.Since(*agentItem.PhaseReportedAt) > compactCompletedAgentsMaxAge {
					isRecent = false
				}
			}

			needsAttention := (isComplete && isRecent) ||
				strings.EqualFold(phaseName(agentItem.Phase), "BLOCKED") ||
				strings.EqualFold(phaseName(agentItem.Phase), "QUESTION")

			// Check phase timeout for unresponsive detection
			// This is done before filtering so unresponsive agents always show in compact mode
			isUnresponsive := false
			if agentItem.PhaseReportedAt != nil && !isComplete &&
				time.Since(*agentItem.PhaseReportedAt) > phaseTimeoutThreshold {
				isUnresponsive = true
			}

			if !isRunning && !needsAttention && !isUnresponsive {
				continue
			}
		}

		if agentItem.IsCompleted && !statusAll {
			continue
		}
		if agentItem.IsPhantom && !statusAll {
			continue
		}

		filteredAgents = append(filteredAgents, agentItem)
	}

	// Build swarm status (counts before filtering)
	activeCount := 0
	processingCount := 0
	idleCount := 0
	phantomCount := 0
	completedCount := 0
	for _, agent := range agents {
		if agent.IsPhantom {
			phantomCount++
		} else if agent.IsCompleted {
			completedCount++
		} else {
			activeCount++
			if agent.IsProcessing {
				processingCount++
			} else {
				idleCount++
			}
		}
	}

	swarm := SwarmStatus{
		Active:     activeCount,
		Processing: processingCount,
		Idle:       idleCount,
		Phantom:    phantomCount,
		Queued:     0,
		Completed:  completedCount,
	}

	// Fetch account usage information
	accounts := getAccountUsage()

	// Fetch token usage - in compact mode, only for running agents (expensive operation)
	// Skip when OpenCode is unreachable to avoid per-agent timeout delays.
	if opencodeReachable {
		for i := range filteredAgents {
			agent := &filteredAgents[i]
			if agent.SessionID == "" || agent.SessionID == "tmux-stalled" {
				continue
			}
			if !statusAll && !agent.IsProcessing {
				continue
			}
			tokens, err := client.GetSessionTokens(agent.SessionID)
			if err == nil && tokens != nil {
				agent.Tokens = tokens
				if agent.IsProcessing && !agent.IsPhantom && !agent.IsCompleted {
					isStalled := globalStallTracker.Update(agent.SessionID, tokens)
					if isStalled {
						agent.IsStalled = true
					}
				}
			}
		}
	}

	// Assess context exhaustion risk - in compact mode, only for running agents
	for i := range filteredAgents {
		agent := &filteredAgents[i]
		if agent.IsPhantom || agent.IsCompleted {
			continue
		}
		if !statusAll && !agent.IsProcessing {
			continue
		}
		totalTokens := 0
		if agent.Tokens != nil {
			totalTokens = agent.Tokens.TotalTokens
			if totalTokens == 0 {
				totalTokens = agent.Tokens.InputTokens + agent.Tokens.OutputTokens
			}
		}
		risk := verify.AssessContextRisk(totalTokens, agent.ProjectDir, agent.IsProcessing)
		if risk.IsAtRisk() {
			agent.ContextRisk = &risk
		}
	}

	// Detect unresponsive agents (phase timeout)
	for i := range filteredAgents {
		agent := &filteredAgents[i]
		if agent.IsPhantom || agent.IsCompleted {
			continue
		}
		if strings.HasPrefix(agent.Phase, "Complete") {
			continue
		}
		if agent.PhaseReportedAt != nil && time.Since(*agent.PhaseReportedAt) > phaseTimeoutThreshold {
			agent.IsUnresponsive = true
		}
	}

	// Check infrastructure health
	infraHealth := checkInfrastructureHealth()

	// Detect synthesis opportunities - skip in compact mode (expensive filesystem scan)
	var synthesisOpps *verify.SynthesisOpportunities
	if statusAll {
		synthesisOpps, _ = verify.DetectSynthesisOpportunities(projectDir)
	}

	// Get session metrics for drift detection
	sessionMetrics := getSessionMetrics()

	// Get review queue count (completions awaiting human review)
	var reviewQueue *ReviewQueueStatus
	if completions, err := getCompletionsForReview(); err == nil {
		readyCount := 0
		for _, c := range completions {
			if !c.IsStale {
				readyCount++
			}
		}
		if readyCount > 0 {
			reviewQueue = &ReviewQueueStatus{Ready: readyCount}
		}
	}

	// Build output
	output := StatusOutput{
		Infrastructure:         infraHealth,
		SessionMetrics:         sessionMetrics,
		Swarm:                  swarm,
		ReviewQueue:            reviewQueue,
		Accounts:               accounts,
		Agents:                 filteredAgents,
		SynthesisOpportunities: synthesisOpps,
	}

	if statusJSON {
		data, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	printSwarmStatus(output, statusAll)
	return nil
}

// agentStatusToAgentInfo converts a queryTrackedAgents result to the CLI display type.
// Maps reason codes to status annotations for human-readable output.
func agentStatusToAgentInfo(tracked discovery.AgentStatus, now time.Time) AgentInfo {
	info := AgentInfo{
		SessionID:  tracked.SessionID,
		BeadsID:    tracked.BeadsID,
		Model:      tracked.Model,
		Skill:      tracked.Skill,
		ProjectDir: tracked.ProjectDir,
		Project:    extractProjectFromBeadsID(tracked.BeadsID),
		Mode:       tracked.SpawnMode,
		Task:       truncate(tracked.Title, 40),
	}

	// Map status from query engine to display flags
	switch tracked.Status {
	case "active":
		info.IsProcessing = true
	case "idle":
		// idle but not phantom (has session)
	case "retrying":
		info.IsProcessing = true // show as running with annotation
	case "completed":
		info.IsCompleted = true
	case "dead":
		// Agent with no liveness signal — distinct from "idle" (has session but not processing).
		// Set IsPhantom so getAgentStatus returns "phantom" instead of misleading "idle".
		info.IsPhantom = true
	}

	// Map phase - extract just the phase name for the Phase field
	if tracked.Phase != "" {
		info.Phase = tracked.Phase
	}
	if tracked.PhaseReportedAt != nil {
		info.PhaseReportedAt = tracked.PhaseReportedAt
	}

	// Surface reason codes
	if tracked.Reason != "" {
		// Embed reason in the status display logic
		switch tracked.Reason {
		case "missing_binding":
			info.IsPhantom = true
		case "session_idle":
			// session exists but idle
		case "opencode_unreachable":
			// degrade gracefully
		}
	}

	if info.Runtime == "" {
		info.Runtime = "unknown"
	}

	return info
}

// extractDateFromWorkspaceName delegates to workspace.ExtractDate.
func extractDateFromWorkspaceName(name string) time.Time {
	return workspace.ExtractDate(name)
}

// phaseName extracts the phase keyword from a full phase string.
// Phase strings look like "QUESTION - Should we use JWT?" or "Implementing - Adding auth".
// Returns just the keyword part (e.g., "QUESTION", "Implementing").
// Returns the full string if there's no " - " separator.
func phaseName(phase string) string {
	if idx := strings.Index(phase, " - "); idx >= 0 {
		return strings.TrimSpace(phase[:idx])
	}
	return strings.TrimSpace(phase)
}

// getPhaseAndTask retrieves the current phase and task description from beads.
func getPhaseAndTask(beadsID string) (phase, task string) {
	// Get issue for task description
	issue, err := verify.GetIssue(beadsID, "")
	if err == nil {
		task = truncate(issue.Title, 40)
	}

	// Get phase from comments
	status, err := verify.GetPhaseStatus(beadsID, "")
	if err == nil && status.Found {
		phase = status.Phase
	}

	return phase, task
}

// getAccountUsage fetches usage info for all configured accounts.
func getAccountUsage() []AccountUsage {
	var result []AccountUsage

	// Identify active account by checking OpenCode auth refresh token
	activeEmail := ""
	if auth, authErr := account.LoadOpenCodeAuth(); authErr == nil && auth.Anthropic.Access != "" {
		cfg, cfgErr := account.LoadConfig()
		if cfgErr == nil {
			for _, acc := range cfg.Accounts {
				if acc.RefreshToken == auth.Anthropic.Refresh {
					activeEmail = acc.Email
					break
				}
			}
		}
	}

	accounts, err := account.ListAccountsWithCapacity()
	if err != nil {
		return result
	}

	for _, awc := range accounts {
		au := AccountUsage{
			Name:     awc.Name,
			Email:    awc.Email,
			IsActive: awc.Email != "" && awc.Email == activeEmail,
		}
		if awc.Capacity != nil && awc.Capacity.Error == "" {
			au.UsedPercent = awc.Capacity.SevenDayUsed
			if awc.Capacity.SevenDayResets != nil {
				au.ResetTime = timeUntilReset(awc.Capacity.SevenDayResets)
			}
		}
		result = append(result, au)
	}

	return result
}

// getSessionMetrics computes drift detection metrics for the current orchestrator session.
// Returns nil if no session is active (which is itself a signal for Dylan).
func getSessionMetrics() *SessionMetrics {
	store, err := session.New("")
	if err != nil {
		return nil // Can't access session state
	}

	if !store.IsActive() {
		// No active session - return minimal metrics to surface this
		return &SessionMetrics{
			HasActiveSession: false,
			TimeInSession:    "-",
		}
	}

	sess := store.Get()
	if sess == nil {
		return nil
	}

	now := time.Now()
	lastActivity := sess.StartedAt
	if len(sess.Spawns) > 0 {
		lastActivity = sess.Spawns[len(sess.Spawns)-1].SpawnedAt
	}
	if now.Sub(lastActivity) > sessionMetricsStaleAfter {
		return &SessionMetrics{
			HasActiveSession: false,
			TimeInSession:    "-",
		}
	}
	metrics := &SessionMetrics{
		HasActiveSession: true,
		Goal:             sess.Goal,
		SpawnCount:       len(sess.Spawns),
		TimeInSession:    formatDuration(now.Sub(sess.StartedAt)),
	}

	// Calculate time since last spawn
	if len(sess.Spawns) > 0 {
		lastSpawn := sess.Spawns[len(sess.Spawns)-1]
		metrics.TimeSinceLastSpawn = formatDuration(now.Sub(lastSpawn.SpawnedAt))
	}

	// NOTE: FileReadsSinceLastSpawn is not yet implemented.
	// Would require tracking orchestrator tool usage via OpenCode events or plugins.
	// For now, this field remains 0 (omitted in JSON).

	return metrics
}
