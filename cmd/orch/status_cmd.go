// Package main provides the status command for showing swarm status and active agents.
// Extracted from main.go as part of the main.go refactoring (Phase 3).
package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/account"
	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/session"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
	"golang.org/x/term"
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
	Tokens          *opencode.TokenStats          `json:"tokens,omitempty"`            // Token usage for the session
	ContextRisk     *verify.ContextExhaustionRisk `json:"context_risk,omitempty"`      // Context exhaustion risk assessment
	PhaseReportedAt *time.Time                    `json:"phase_reported_at,omitempty"` // Timestamp when latest phase was reported
	LastActivity    time.Time                     `json:"last_activity,omitempty"`     // Timestamp of last activity (for ghost filtering)
}

// InfraServiceStatus represents the health status of an infrastructure service.
type InfraServiceStatus struct {
	Name    string `json:"name"`
	Running bool   `json:"running"`
	Port    int    `json:"port,omitempty"`
	Details string `json:"details,omitempty"`
}

// DaemonStatus represents the status from daemon-status.json.
type DaemonStatus struct {
	PID            int    `json:"pid,omitempty"`
	Status         string `json:"status"`
	LastPoll       string `json:"last_poll,omitempty"`
	LastSpawn      string `json:"last_spawn,omitempty"`
	LastCompletion string `json:"last_completion,omitempty"`
	ReadyCount     int    `json:"ready_count,omitempty"`
	Capacity       struct {
		Max       int `json:"max"`
		Active    int `json:"active"`
		Available int `json:"available"`
	} `json:"capacity,omitempty"`
}

// InfrastructureHealth represents the overall infrastructure health status.
type InfrastructureHealth struct {
	AllHealthy bool                 `json:"all_healthy"`
	Services   []InfraServiceStatus `json:"services"`
	Daemon     *DaemonStatus        `json:"daemon,omitempty"`
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

// StatusOutput represents the full status output for JSON serialization.
type StatusOutput struct {
	Infrastructure         *InfrastructureHealth          `json:"infrastructure,omitempty"`
	SessionMetrics         *SessionMetrics                `json:"session_metrics,omitempty"`
	Swarm                  SwarmStatus                    `json:"swarm"`
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

	// Enrich with session data for runtime/processing status
	// queryTrackedAgents provides liveness (active/idle/retrying) but not timestamps or processing details.
	sessions, _ := listSessionsAcrossProjects(client, projectDir)
	sessionMap := make(map[string]*opencode.Session)
	for i := range sessions {
		sessionMap[sessions[i].ID] = &sessions[i]
	}

	sessionStatusMap := make(map[string]opencode.SessionStatusInfo)
	{
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
			if statusInfo, ok := sessionStatusMap[s.ID]; ok {
				agent.IsProcessing = statusInfo.IsBusy() || statusInfo.IsRetrying()
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
				strings.EqualFold(agentItem.Phase, "BLOCKED") ||
				strings.EqualFold(agentItem.Phase, "QUESTION")

			if !isRunning && !needsAttention {
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

	// Check infrastructure health
	infraHealth := checkInfrastructureHealth()

	// Detect synthesis opportunities - skip in compact mode (expensive filesystem scan)
	var synthesisOpps *verify.SynthesisOpportunities
	if statusAll {
		synthesisOpps, _ = verify.DetectSynthesisOpportunities(projectDir)
	}

	// Get session metrics for drift detection
	sessionMetrics := getSessionMetrics()

	// Build output
	output := StatusOutput{
		Infrastructure:         infraHealth,
		SessionMetrics:         sessionMetrics,
		Swarm:                  swarm,
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
func agentStatusToAgentInfo(tracked AgentStatus, now time.Time) AgentInfo {
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

	// Map status
	switch tracked.Status {
	case "active":
		info.IsProcessing = true
	case "idle":
		// idle but not phantom (has session)
	case "retrying":
		info.IsProcessing = true // show as running with annotation
	}

	// Map phase - extract just the phase name for the Phase field
	if tracked.Phase != "" {
		info.Phase = tracked.Phase
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

// extractDateFromWorkspaceName parses the date suffix from a workspace name.
// Workspace names follow format: prefix-description-DDmon (e.g., og-feat-add-feature-24dec)
// Returns zero time if no valid date found.
func extractDateFromWorkspaceName(name string) time.Time {
	// Month abbreviations (lowercase)
	months := map[string]time.Month{
		"jan": time.January,
		"feb": time.February,
		"mar": time.March,
		"apr": time.April,
		"may": time.May,
		"jun": time.June,
		"jul": time.July,
		"aug": time.August,
		"sep": time.September,
		"oct": time.October,
		"nov": time.November,
		"dec": time.December,
	}

	// Get the last segment after the final hyphen
	parts := strings.Split(name, "-")
	if len(parts) == 0 {
		return time.Time{}
	}
	lastPart := strings.ToLower(parts[len(parts)-1])

	// Pattern: 1-2 digits followed by 3-letter month abbreviation (e.g., "24dec", "5jan")
	if len(lastPart) < 4 || len(lastPart) > 5 {
		return time.Time{}
	}

	// Extract the month abbreviation (last 3 chars)
	monthStr := lastPart[len(lastPart)-3:]
	month, ok := months[monthStr]
	if !ok {
		return time.Time{}
	}

	// Extract the day (remaining digits)
	dayStr := lastPart[:len(lastPart)-3]
	day, err := strconv.Atoi(dayStr)
	if err != nil || day < 1 || day > 31 {
		return time.Time{}
	}

	// Use current year, adjusting for year boundary
	// (if the date is in the future within this calendar, it's probably from last year)
	now := time.Now()
	year := now.Year()
	parsedDate := time.Date(year, month, day, 12, 0, 0, 0, time.Local)

	// If the parsed date is more than a week in the future, assume it's from last year
	if parsedDate.After(now.AddDate(0, 0, 7)) {
		parsedDate = time.Date(year-1, month, day, 12, 0, 0, 0, time.Local)
	}

	return parsedDate
}

// getPhaseAndTask retrieves the current phase and task description from beads.
func getPhaseAndTask(beadsID string) (phase, task string) {
	// Get issue for task description
	issue, err := verify.GetIssue(beadsID)
	if err == nil {
		task = truncate(issue.Title, 40)
	}

	// Get phase from comments
	status, err := verify.GetPhaseStatus(beadsID)
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

// Terminal width thresholds for adaptive output
const (
	termWidthWide   = 120 // Full table with all columns
	termWidthNarrow = 100 // Drop TASK column, abbreviate SKILL
	termWidthMin    = 80  // Minimum supported width (vertical card format)
)

// getTerminalWidth returns the current terminal width, or a default if detection fails.
// Returns the width and whether we're outputting to a real terminal.
func getTerminalWidth() (int, bool) {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		// Not a terminal (piped output) - use wide format
		return termWidthWide + 1, false
	}
	return width, true
}

// printSwarmStatus prints the swarm status in human-readable format.
// Adapts output format based on terminal width.
func printSwarmStatus(output StatusOutput, showAll bool) {
	width, _ := getTerminalWidth()
	printSwarmStatusWithWidth(output, showAll, width)
}

// printSwarmStatusWithWidth prints swarm status with explicit width (for testing).
func printSwarmStatusWithWidth(output StatusOutput, showAll bool, termWidth int) {
	// Check for dev mode and warn
	if devModeInfo, err := readDevModeFile(".dev-mode"); err == nil {
		duration := time.Since(devModeInfo.Started).Round(time.Minute)
		fmt.Printf("WARNING: DEV MODE ACTIVE (%s): %s\n", duration, devModeInfo.Reason)
		fmt.Println("   Infrastructure is unprotected. Run 'orch mode ops' when done.")
		fmt.Println()
	}

	// Print infrastructure health section first
	printInfrastructureHealth(output.Infrastructure)

	// Print session metrics for drift detection (surfaces to Dylan)
	// Position: After infrastructure, before swarm - visible at top
	printSessionMetrics(output.SessionMetrics)

	// Print swarm summary header with processing breakdown
	fmt.Printf("SWARM STATUS: Active: %d", output.Swarm.Active)
	if output.Swarm.Active > 0 {
		fmt.Printf(" (running: %d, idle: %d)", output.Swarm.Processing, output.Swarm.Idle)
	}
	if output.Swarm.Completed > 0 {
		fmt.Printf(", Completed: %d", output.Swarm.Completed)
		if !showAll {
			fmt.Printf(" (use --all to show)")
		}
	}
	if output.Swarm.Phantom > 0 {
		fmt.Printf(", Phantom: %d", output.Swarm.Phantom)
		if !showAll {
			fmt.Printf(" (use --all to show)")
		}
	}
	fmt.Println()
	// In compact mode, add hint about hidden idle agents
	if !showAll && output.Swarm.Idle > 0 && output.Swarm.Idle > len(output.Agents) {
		hiddenIdle := output.Swarm.Idle - countIdleInList(output.Agents)
		if hiddenIdle > 0 {
			fmt.Printf("  (compact mode: %d idle agents hidden, use --all for full list)\n", hiddenIdle)
		}
	}
	fmt.Println()

	// Print account usage
	if len(output.Accounts) > 0 {
		fmt.Println("ACCOUNTS")
		for _, acc := range output.Accounts {
			activeMarker := ""
			if acc.IsActive {
				activeMarker = " *"
			}
			usageStr := "N/A"
			if acc.UsedPercent > 0 || acc.IsActive {
				usageStr = fmt.Sprintf("%.0f%% used", acc.UsedPercent)
				if acc.ResetTime != "" {
					usageStr += fmt.Sprintf(" (resets in %s)", acc.ResetTime)
				}
			}
			name := acc.Name
			if acc.Email != "" && acc.Name == "current" {
				name = acc.Email
			}
			fmt.Printf("  %-20s %s%s\n", name+":", usageStr, activeMarker)
		}
		fmt.Println()
	}

	// Print agents in format appropriate for terminal width
	if len(output.Agents) > 0 {
		fmt.Println("AGENTS")
		if termWidth < termWidthMin {
			printAgentsCardFormat(output.Agents)
		} else if termWidth < termWidthNarrow {
			printAgentsNarrowFormat(output.Agents)
		} else {
			printAgentsWideFormat(output.Agents)
		}
	} else {
		fmt.Println("No active agents")
	}

	// Print synthesis opportunities (if any)
	if output.SynthesisOpportunities != nil && output.SynthesisOpportunities.HasOpportunities() {
		fmt.Println()
		printSynthesisOpportunities(output.SynthesisOpportunities)
	}
}

// printAgentsWideFormat prints agents in full table format (>120 chars).
// Columns: SOURCE, BEADS ID, STATUS, PHASE, TASK, SKILL, RUNTIME, TOKENS, RISK
func printAgentsWideFormat(agents []AgentInfo) {
	// Check if any agent has risk to show RISK column
	hasRisk := false
	for _, agent := range agents {
		if agent.ContextRisk != nil && agent.ContextRisk.IsAtRisk() {
			hasRisk = true
			break
		}
	}

	if hasRisk {
		fmt.Printf("  %-3s %-18s %-8s %-8s %-12s %-20s %-25s %-12s %-7s %-16s %s\n", "SRC", "BEADS ID", "MODE", "MODEL", "STATUS", "PHASE", "TASK", "SKILL", "RUNTIME", "TOKENS", "RISK")
		fmt.Printf("  %s\n", strings.Repeat("-", 150))
	} else {
		fmt.Printf("  %-3s %-18s %-8s %-20s %-8s %-12s %-23s %-12s %-8s %s\n", "SRC", "BEADS ID", "MODE", "MODEL", "STATUS", "PHASE", "TASK", "SKILL", "RUNTIME", "TOKENS")
		fmt.Printf("  %s\n", strings.Repeat("-", 140))
	}

	for _, agent := range agents {
		source := agent.Source
		if source == "" {
			source = "-"
		}
		beadsID := formatBeadsIDForDisplay(agent.BeadsID)
		if beadsID == "" {
			beadsID = "-"
		}
		mode := agent.Mode
		if mode == "" {
			mode = "-"
		}
		modelDisplay := formatModelForDisplay(agent.Model)
		phase := agent.Phase
		if phase == "" {
			phase = "-"
		}
		task := agent.Task
		if task == "" {
			task = "-"
		}
		skill := agent.Skill
		if skill == "" {
			skill = "-"
		}
		status := getAgentStatus(agent)
		tokens := formatTokenStatsCompact(agent.Tokens)

		if hasRisk {
			risk := formatContextRisk(agent.ContextRisk)
			fmt.Printf("  %-3s %-18s %-8s %-20s %-8s %-12s %-25s %-12s %-7s %-16s %s\n",
				source,
				beadsID,
				mode,
				modelDisplay,
				status,
				truncate(phase, 10),
				truncate(task, 23),
				truncate(skill, 10),
				agent.Runtime,
				tokens,
				risk)
		} else {
			fmt.Printf("  %-3s %-18s %-8s %-20s %-8s %-12s %-23s %-12s %-8s %s\n",
				source,
				beadsID,
				mode,
				modelDisplay,
				status,
				truncate(phase, 10),
				truncate(task, 21),
				truncate(skill, 10),
				agent.Runtime,
				tokens)
		}
	}
}

// formatContextRisk returns a formatted string for context exhaustion risk.
func formatContextRisk(risk *verify.ContextExhaustionRisk) string {
	if risk == nil || !risk.IsAtRisk() {
		return ""
	}
	emoji := risk.FormatRiskEmoji()
	status := risk.FormatRiskStatus()
	if emoji != "" {
		return emoji + " " + status
	}
	return status
}

// printAgentsNarrowFormat prints agents in narrow format (80-100 chars).
// Drops TASK column, abbreviates SKILL and MODEL.
// Columns: SOURCE, BEADS ID, MODE, MODEL, STATUS, PHASE, SKILL, RUNTIME, TOKENS
func printAgentsNarrowFormat(agents []AgentInfo) {
	fmt.Printf("  %-3s %-18s %-8s %-8s %-8s %-10s %-8s %-8s %s\n", "SRC", "BEADS ID", "MODE", "MODEL", "STATUS", "PHASE", "SKILL", "RUNTIME", "TOKENS")
	fmt.Printf("  %s\n", strings.Repeat("-", 98))

	for _, agent := range agents {
		source := agent.Source
		if source == "" {
			source = "-"
		}
		beadsID := formatBeadsIDForDisplay(agent.BeadsID)
		if beadsID == "" {
			beadsID = "-"
		}
		mode := agent.Mode
		if mode == "" {
			mode = "-"
		}
		modelDisplay := formatModelForDisplay(agent.Model)
		phase := agent.Phase
		if phase == "" {
			phase = "-"
		}
		skill := abbreviateSkill(agent.Skill)
		if skill == "" {
			skill = "-"
		}
		status := getAgentStatus(agent)
		tokens := formatTokenStatsCompact(agent.Tokens)

		fmt.Printf("  %-3s %-18s %-8s %-8s %-8s %-10s %-8s %-8s %s\n",
			source,
			beadsID,
			truncate(mode, 7),
			truncate(modelDisplay, 7),
			status,
			truncate(phase, 9),
			truncate(skill, 7),
			agent.Runtime,
			tokens)
	}
}

// printAgentsCardFormat prints agents in vertical card format (<80 chars).
// Each agent is a multi-line block for readability on very narrow terminals.
func printAgentsCardFormat(agents []AgentInfo) {
	for i, agent := range agents {
		if i > 0 {
			fmt.Println()
		}
		source := agent.Source
		if source == "" {
			source = "-"
		}
		beadsID := formatBeadsIDForDisplay(agent.BeadsID)
		if beadsID == "" {
			beadsID = "-"
		}
		modelDisplay := formatModelForDisplay(agent.Model)
		phase := agent.Phase
		if phase == "" {
			phase = "-"
		}
		task := agent.Task
		if task == "" {
			task = "-"
		}
		skill := agent.Skill
		if skill == "" {
			skill = "-"
		}
		status := getAgentStatus(agent)
		riskStr := formatContextRisk(agent.ContextRisk)

		if riskStr != "" {
			fmt.Printf("  [%s] %s [%s] %s\n", source, beadsID, status, riskStr)
		} else {
			fmt.Printf("  [%s] %s [%s]\n", source, beadsID, status)
		}
		fmt.Printf("    Model: %s | Phase: %s | Skill: %s\n", modelDisplay, phase, skill)
		fmt.Printf("    Task: %s\n", truncate(task, 50))
		fmt.Printf("    Runtime: %s | Tokens: %s\n", agent.Runtime, formatTokenStats(agent.Tokens))
		if agent.ContextRisk != nil && agent.ContextRisk.Reason != "" {
			fmt.Printf("    Risk: %s\n", agent.ContextRisk.Reason)
		}
	}
}

// countIdleInList counts the number of idle agents in a list.
// Used to calculate how many idle agents are hidden in compact mode.
func countIdleInList(agents []AgentInfo) int {
	count := 0
	for _, agent := range agents {
		if !agent.IsProcessing && !agent.IsPhantom && !agent.IsCompleted {
			count++
		}
	}
	return count
}

// getAgentStatus returns a status string based on agent state.
func getAgentStatus(agent AgentInfo) string {
	if agent.IsCompleted {
		return "completed"
	}
	if agent.IsPhantom {
		return "phantom"
	}
	if agent.IsStalled {
		return "⚠️ STALLED"
	}
	if agent.IsProcessing {
		return "running"
	}
	return "idle"
}

// determineAgentSource returns the primary source indicator for an agent.
// Priority: T (tmux) > O (OpenCode) > B (beads phantom) > W (workspace).
// Returns: T=tmux, O=opencode, B=beads phantom, W=workspace, or empty string if unknown.
func determineAgentSource(agent AgentInfo, projectDir string) string {
	// Tmux has highest priority (visible TUI)
	if agent.Window != "" {
		return "T"
	}

	// OpenCode session (headless or API mode)
	if agent.SessionID != "" && agent.SessionID != "tmux-stalled" && agent.SessionID != "api-stalled" {
		return "O"
	}

	// Beads phantom (issue exists but no active runtime)
	if agent.BeadsID != "" && agent.IsPhantom {
		return "B"
	}

	// Workspace (has workspace directory)
	if agent.BeadsID != "" && projectDir != "" {
		workspacePath, _ := findWorkspaceByBeadsID(projectDir, agent.BeadsID)
		if workspacePath != "" {
			return "W"
		}
	}

	return ""
}

// printSynthesisOpportunities prints the synthesis opportunities section.
// Only shown when there are opportunities (3+ investigations on a topic without synthesis).
func printSynthesisOpportunities(opps *verify.SynthesisOpportunities) {
	fmt.Println("SYNTHESIS OPPORTUNITIES")
	for _, opp := range opps.Opportunities {
		fmt.Printf("  %d investigations on '%s' without synthesis\n", opp.InvestigationCount, opp.Topic)
	}
}

// printSessionMetrics prints the session metrics section for drift detection.
// Purpose: Surface orchestrator behavior to Dylan so he can detect drift
// (e.g., "2 hours, no spawns" → orchestrator doing task work instead of delegating).
func printSessionMetrics(metrics *SessionMetrics) {
	if metrics == nil {
		return
	}

	fmt.Println("SESSION METRICS")

	if !metrics.HasActiveSession {
		fmt.Println("  No active session (run 'orch session start \"goal\"')")
		fmt.Println()
		return
	}

	// Print goal first (context for the numbers)
	if metrics.Goal != "" {
		fmt.Printf("  Goal: %s\n", truncate(metrics.Goal, 50))
	}

	// Core metrics for drift detection
	fmt.Printf("  Time in session: %s\n", metrics.TimeInSession)

	if metrics.SpawnCount > 0 {
		fmt.Printf("  Last spawn: %s ago\n", metrics.TimeSinceLastSpawn)
	} else {
		fmt.Printf("  Last spawn: no spawns yet\n")
	}

	fmt.Printf("  Spawns: %d\n", metrics.SpawnCount)

	// File reads metric (placeholder for future implementation)
	// NOTE: Not surfacing FileReadsSinceLastSpawn until tracking is implemented

	fmt.Println()
}

// abbreviateSkill returns a shortened version of skill names for narrow displays.
func abbreviateSkill(skill string) string {
	abbreviations := map[string]string{
		"feature-impl":         "feat",
		"investigation":        "inv",
		"systematic-debugging": "debug",
		"architect":            "arch",
		"codebase-audit":       "audit",
		"reliability-testing":  "rel-test",
		"issue-creation":       "issue",
		"design-session":       "design",
		"research":             "research",
	}
	if abbr, ok := abbreviations[skill]; ok {
		return abbr
	}
	return skill
}

// formatModelForDisplay formats a model spec for compact display.
// Shortens common model names (e.g., "gemini-3-flash-preview" -> "flash3", "claude-opus-4-5-20251101" -> "opus-4.5")
func formatModelForDisplay(model string) string {
	if model == "" {
		return "-"
	}

	// Map full model IDs to short display names
	modelAbbreviations := map[string]string{
		"gemini-3-flash-preview":     "flash3",
		"gemini-2.5-flash":           "flash-2.5",
		"gemini-2.5-pro":             "pro-2.5",
		"claude-opus-4-5-20251101":   "opus-4.5",
		"claude-sonnet-4-5-20250929": "sonnet-4.5",
		"claude-haiku-4-5-20251001":  "haiku-4.5",
		"gpt-5":                      "gpt5",
		"gpt-5.2":                    "gpt5-latest",
		"gpt-5-mini":                 "gpt5-mini",
		"o3":                         "o3",
		"o3-mini":                    "o3-mini",
		"deepseek-chat":              "deepseek",
		"deepseek-reasoner":          "deepseek-r1",
	}

	if abbr, ok := modelAbbreviations[model]; ok {
		return abbr
	}

	// For unknown models, truncate to 18 chars
	return truncate(model, 18)
}

// formatTokenCount formats a token count with K/M suffixes for readability.
func formatTokenCount(count int) string {
	if count < 1000 {
		return fmt.Sprintf("%d", count)
	}
	if count < 1000000 {
		return fmt.Sprintf("%.1fK", float64(count)/1000)
	}
	return fmt.Sprintf("%.1fM", float64(count)/1000000)
}

// formatTokenStats returns a formatted string of token usage.
func formatTokenStats(tokens *opencode.TokenStats) string {
	if tokens == nil {
		return "-"
	}
	// Format: "in:X out:Y (cache:Z)"
	result := fmt.Sprintf("in:%s out:%s", formatTokenCount(tokens.InputTokens), formatTokenCount(tokens.OutputTokens))
	if tokens.CacheReadTokens > 0 {
		result += fmt.Sprintf(" (cache:%s)", formatTokenCount(tokens.CacheReadTokens))
	}
	return result
}

// formatTokenStatsCompact returns a compact formatted string of token usage for table display.
// Shows total tokens with input/output breakdown: "12.5K (8K/4K)"
func formatTokenStatsCompact(tokens *opencode.TokenStats) string {
	if tokens == nil {
		return "-"
	}
	total := tokens.TotalTokens
	if total == 0 {
		total = tokens.InputTokens + tokens.OutputTokens
	}
	if total == 0 {
		return "-"
	}
	// Format: "total (in/out)" for quick scanning
	return fmt.Sprintf("%s (%s/%s)",
		formatTokenCount(total),
		formatTokenCount(tokens.InputTokens),
		formatTokenCount(tokens.OutputTokens))
}

// getBeadsIssuePrefix reads the issue_prefix for a project using bd CLI.
// Returns empty string if the command fails or project doesn't have beads.
func getBeadsIssuePrefix(projectPath string) string {
	cmd := exec.Command("bd", "config", "get", "issue_prefix")
	cmd.Dir = projectPath
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	// Output is just the value (e.g., "pw\n")
	return strings.TrimSpace(string(output))
}

// getKBProjectsWithNames fetches registered projects from kb with name and path.
// Returns empty slice if kb is unavailable or fails (graceful degradation).
func getKBProjectsWithNames() []kbProject {
	cmd := exec.Command("kb", "projects", "list", "--json")
	output, err := cmd.Output()
	if err != nil {
		return []kbProject{}
	}

	var projects []kbProject
	if err := json.Unmarshal(output, &projects); err != nil {
		return []kbProject{}
	}

	return projects
}

// findProjectByBeadsPrefix searches for a project with the given beads issue prefix.
// First checks kb's project registry, then falls back to standard locations.
// Returns the project directory path, or empty string if not found.
func findProjectByBeadsPrefix(prefix string) string {
	// Try kb's project registry first
	for _, project := range getKBProjectsWithNames() {
		if projectPrefix := getBeadsIssuePrefix(project.Path); projectPrefix == prefix {
			return project.Path
		}
	}

	// Fall back to checking standard locations
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	candidatePaths := []string{
		filepath.Join(homeDir, "Documents", "personal", prefix),
		filepath.Join(homeDir, prefix),
		filepath.Join(homeDir, "projects", prefix),
		filepath.Join(homeDir, "src", prefix),
	}

	for _, path := range candidatePaths {
		if projectPrefix := getBeadsIssuePrefix(path); projectPrefix == prefix {
			return path
		}
	}

	return ""
}

// findProjectDirByName looks up a project directory by its name or beads prefix.
// First checks kb's project registry, then searches common project locations.
// Verifies the project has a .beads/ directory.
// Returns empty string if not found.
func findProjectDirByName(projectName string) string {
	// Try kb's project registry first (handles non-standard locations)
	for _, project := range getKBProjectsWithNames() {
		if project.Name == projectName {
			// Verify it has a .beads directory
			beadsPath := filepath.Join(project.Path, ".beads")
			if info, err := os.Stat(beadsPath); err == nil && info.IsDir() {
				return project.Path
			}
		}
	}

	// If projectName looks like a beads prefix (short, no hyphens except separators),
	// try finding by prefix instead
	if len(projectName) <= 10 && !strings.Contains(projectName, "/") {
		if path := findProjectByBeadsPrefix(projectName); path != "" {
			return path
		}
	}

	// Fall back to checking standard locations
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	// Common project locations in order of priority
	candidatePaths := []string{
		filepath.Join(homeDir, "Documents", "personal", projectName),
		filepath.Join(homeDir, projectName),
		filepath.Join(homeDir, "projects", projectName),
		filepath.Join(homeDir, "src", projectName),
	}

	for _, path := range candidatePaths {
		// Check if directory exists and has .beads/ (confirms it's a beads-tracked project)
		beadsPath := filepath.Join(path, ".beads")
		if info, err := os.Stat(beadsPath); err == nil && info.IsDir() {
			return path
		}
	}

	return ""
}

// checkInfrastructureHealth checks the health of infrastructure services.
// Performs TCP connect tests for dashboard (port 3348) and OpenCode (port 4096),
// and reads daemon status from ~/.orch/daemon-status.json.
func checkInfrastructureHealth() *InfrastructureHealth {
	health := &InfrastructureHealth{
		AllHealthy: true,
		Services:   make([]InfraServiceStatus, 0, 2),
	}

	// Check Dashboard server (orch serve) on port 3348
	dashboardStatus := checkTCPPort("Dashboard", DefaultServePort)
	health.Services = append(health.Services, dashboardStatus)
	if !dashboardStatus.Running {
		health.AllHealthy = false
	}

	// Check OpenCode server on port 4096
	opencodeStatus := checkTCPPort("OpenCode", 4096)
	health.Services = append(health.Services, opencodeStatus)
	if !opencodeStatus.Running {
		health.AllHealthy = false
	}

	// Check daemon status from file
	daemonStatus := readDaemonStatus()
	health.Daemon = daemonStatus
	if daemonStatus == nil || daemonStatus.Status != "running" {
		health.AllHealthy = false
	}

	return health
}

// checkTCPPort performs a TCP connect test to verify a service is listening.
func checkTCPPort(name string, port int) InfraServiceStatus {
	status := InfraServiceStatus{
		Name: name,
		Port: port,
	}

	addr := fmt.Sprintf("localhost:%d", port)
	conn, err := tcpDialTimeout(addr, 1*time.Second)
	if err != nil {
		status.Running = false
		status.Details = "not responding"
		return status
	}
	conn.Close()

	status.Running = true
	status.Details = "listening"
	return status
}

// tcpDialTimeout dials a TCP address with a timeout.
// This is a wrapper to allow for testing.
var tcpDialTimeout = tcpDialTimeoutImpl

// tcpDialTimeoutImpl is the actual implementation of TCP dial using net.DialTimeout.
func tcpDialTimeoutImpl(addr string, timeout time.Duration) (interface{ Close() error }, error) {
	return net.DialTimeout("tcp", addr, timeout)
}

// readDaemonStatus reads the daemon status from ~/.orch/daemon-status.json.
// Validates PID liveness to avoid reporting stale status from dead daemons.
func readDaemonStatus() *DaemonStatus {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	statusPath := filepath.Join(homeDir, ".orch", "daemon-status.json")
	data, err := os.ReadFile(statusPath)
	if err != nil {
		return nil
	}

	var status DaemonStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil
	}

	// Check PID liveness — stale files from crashed daemons should not report as running
	if status.PID > 0 && !daemon.IsProcessAlive(status.PID) {
		return nil
	}

	return &status
}

// printInfrastructureHealth prints the infrastructure health section.
func printInfrastructureHealth(health *InfrastructureHealth) {
	if health == nil {
		return
	}

	fmt.Println("SYSTEM HEALTH")
	for _, svc := range health.Services {
		emoji := "✅"
		if !svc.Running {
			emoji = "❌"
		}
		fmt.Printf("  %s %s (port %d) - %s\n", emoji, svc.Name, svc.Port, svc.Details)
	}

	// Print daemon status
	if health.Daemon != nil {
		emoji := "✅"
		if health.Daemon.Status != "running" {
			emoji = "❌"
		}
		daemonDetails := health.Daemon.Status
		if health.Daemon.Status == "running" && health.Daemon.ReadyCount > 0 {
			daemonDetails = fmt.Sprintf("%s (%d ready)", health.Daemon.Status, health.Daemon.ReadyCount)
		}
		fmt.Printf("  %s Daemon - %s\n", emoji, daemonDetails)
	} else {
		fmt.Println("  ❌ Daemon - not running")
	}
	fmt.Println()
}
