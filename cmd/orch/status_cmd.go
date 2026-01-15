// Package main provides the status command for showing swarm status and active agents.
// Extracted from main.go as part of the main.go refactoring (Phase 3).
package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/account"
	"github.com/dylan-conlin/orch-go/pkg/agent"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/registry"
	"github.com/dylan-conlin/orch-go/pkg/session"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/dylan-conlin/orch-go/pkg/usage"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	// Status command flags
	statusJSON    bool
	statusAll     bool   // Include phantom agents (default: hide)
	statusProject string // Filter by project
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show swarm status and active agents",
	Long: `Show swarm status including active/queued/completed agent counts,
per-account usage percentages, and individual agent details.

By default, phantom agents (beads issue open but no running agent) are hidden.
Use --all to include them.

Examples:
  orch-go status              # Show active agents only
  orch-go status --all        # Include phantom agents
  orch-go status --project snap  # Filter by project
  orch-go status --json       # Output as JSON for scripting`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStatus(serverURL)
	},
}

func init() {
	statusCmd.Flags().BoolVar(&statusJSON, "json", false, "Output as JSON for scripting")
	statusCmd.Flags().BoolVar(&statusAll, "all", false, "Include phantom agents")
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
	SessionID    string                        `json:"session_id"`
	BeadsID      string                        `json:"beads_id,omitempty"`
	Mode         string                        `json:"mode,omitempty"`  // Agent mode: "claude" or "opencode"
	Model        string                        `json:"model,omitempty"` // Model spec (e.g., "gemini-3-flash-preview", "claude-opus-4-5-20251101")
	Skill        string                        `json:"skill,omitempty"`
	Account      string                        `json:"account,omitempty"`
	Runtime      string                        `json:"runtime"`
	Title        string                        `json:"title,omitempty"`
	Window       string                        `json:"window,omitempty"`
	Phase        string                        `json:"phase,omitempty"`         // Current phase from beads comments
	Task         string                        `json:"task,omitempty"`          // Task description (truncated)
	Project      string                        `json:"project,omitempty"`       // Project name derived from beads ID or workspace
	ProjectDir   string                        `json:"project_dir,omitempty"`   // Full path to project directory (for cross-project agents)
	IsPhantom    bool                          `json:"is_phantom,omitempty"`    // True if beads issue open but agent not running
	IsProcessing bool                          `json:"is_processing,omitempty"` // True if session is actively generating a response
	IsCompleted  bool                          `json:"is_completed,omitempty"`  // True if beads issue is closed
	Tokens       *opencode.TokenStats          `json:"tokens,omitempty"`        // Token usage for the session
	ContextRisk  *verify.ContextExhaustionRisk `json:"context_risk,omitempty"`  // Context exhaustion risk assessment
	LastActivity time.Time                     `json:"last_activity,omitempty"` // Timestamp of last activity (for ghost filtering)
}

// OrchestratorSessionInfo represents an active orchestrator session for display.
type OrchestratorSessionInfo struct {
	WorkspaceName string `json:"workspace_name"`
	Goal          string `json:"goal"`
	Duration      string `json:"duration"`
	Project       string `json:"project,omitempty"`
	Status        string `json:"status"`
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
	OrchestratorSessions   []OrchestratorSessionInfo      `json:"orchestrator_sessions,omitempty"`
	Agents                 []AgentInfo                    `json:"agents"`
	SynthesisOpportunities *verify.SynthesisOpportunities `json:"synthesis_opportunities,omitempty"`
}

func runStatus(serverURL string) error {
	client := opencode.NewClient(serverURL)
	now := time.Now()

	// Initialize agent registry
	agentReg, err := registry.New("")
	if err != nil {
		// Log error but continue - registry might be missing or corrupt
		fmt.Fprintf(os.Stderr, "Warning: failed to load agent registry: %v\n", err)
	}

	agents := make([]AgentInfo, 0)
	seenBeadsIDs := make(map[string]bool)

	// === OPTIMIZED: Batch fetch all data upfront ===
	// 1. Fetch all OpenCode sessions in one call (already fast, ~15ms)
	sessions, err := client.ListSessions("")
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	// Build a map of session ID -> session for quick lookup
	sessionMap := make(map[string]*opencode.Session)
	// Also build a map of beadsID -> session for matching
	beadsToSession := make(map[string]*opencode.Session)
	const maxIdleTime = 30 * time.Minute

	for i := range sessions {
		s := &sessions[i]
		sessionMap[s.ID] = s

		// Only consider recently active sessions for beads matching
		updatedAt := time.Unix(s.Time.Updated/1000, 0)
		if now.Sub(updatedAt) <= maxIdleTime {
			beadsID := extractBeadsIDFromTitle(s.Title)
			if beadsID != "" {
				beadsToSession[beadsID] = s
			}
		}
	}

	// 2. Collect beads IDs first, then batch fetch issues later
	// (openIssues removed - we now use allIssues to check both open and closed status)

	// 3. Collect all beads IDs we need comments for
	var beadsIDsToFetch []string

	// Track project directories for cross-project agents (beadsID -> projectDir)
	beadsProjectDirs := make(map[string]string)

	// Get current project's workspace directory for workspace lookups
	projectDir, _ := os.Getwd()

	// Phase 1: Collect agents from registry (primary source of truth for mode)
	if agentReg != nil {
		registryAgents := agentReg.ListActive()
		if statusAll {
			registryAgents = append(registryAgents, agentReg.ListCompleted()...)
		}

		for _, a := range registryAgents {
			if a.BeadsID != "" {
				if !seenBeadsIDs[a.BeadsID] {
					beadsIDsToFetch = append(beadsIDsToFetch, a.BeadsID)
					seenBeadsIDs[a.BeadsID] = true
				}
				if a.ProjectDir != "" {
					beadsProjectDirs[a.BeadsID] = a.ProjectDir
				}
			}

			// Unify into AgentInfo
			info := AgentInfo{
				SessionID:  a.SessionID,
				BeadsID:    a.BeadsID,
				Mode:       a.Mode,
				Model:      a.Model,
				Skill:      a.Skill,
				ProjectDir: a.ProjectDir,
				Project:    extractProjectFromBeadsID(a.BeadsID),
			}

			// Mode-aware enrichment
			if a.Mode == "claude" || a.Mode == "tmux" {
				info.Window = a.TmuxWindow
				info.Title = a.TmuxWindow // Default to window name

				// Check if tmux window actually exists
				windowExists := false
				if a.TmuxWindow != "" {
					if strings.HasPrefix(a.TmuxWindow, "@") {
						windowExists = tmux.WindowExistsByID(a.TmuxWindow)
					} else {
						windowExists = tmux.WindowExists(a.TmuxWindow)
					}
				}

				if !windowExists {
					info.SessionID = "tmux-stalled"
				}

				// Even in claude mode, we might have an OpenCode session for tokens/processing
				if a.SessionID != "" {
					if s, ok := sessionMap[a.SessionID]; ok {
						createdAt := time.Unix(s.Time.Created/1000, 0)
						updatedAt := time.Unix(s.Time.Updated/1000, 0)
						info.Runtime = formatDuration(now.Sub(createdAt))
						info.LastActivity = updatedAt
						info.IsProcessing = client.IsSessionProcessing(s.ID)
						if info.Title == "" || info.Title == a.TmuxWindow {
							info.Title = s.Title
						}
					}
				}
			} else if a.Mode == "opencode" || a.Mode == "headless" {
				if s, ok := sessionMap[a.SessionID]; ok {
					createdAt := time.Unix(s.Time.Created/1000, 0)
					updatedAt := time.Unix(s.Time.Updated/1000, 0)
					info.Runtime = formatDuration(now.Sub(createdAt))
					info.LastActivity = updatedAt
					info.Title = s.Title
					info.IsProcessing = client.IsSessionProcessing(s.ID)
				} else {
					info.SessionID = "api-stalled"
				}
			}

			// Add to agents list
			agents = append(agents, info)
		}
	}

	// Phase 2: Discovery - Collect agents from tmux windows (for untracked or legacy agents)
	workersSessions, _ := tmux.ListWorkersSessions()
	for _, sessionName := range workersSessions {
		windows, _ := tmux.ListWindows(sessionName)
		for _, w := range windows {
			// Skip known non-agent windows
			if w.Name == "servers" || w.Name == "zsh" {
				continue
			}

			beadsID := extractBeadsIDFromWindowName(w.Name)
			if beadsID == "" {
				continue
			}

			// Skip if already tracked via registry
			if seenBeadsIDs[beadsID] {
				// Enrich existing AgentInfo with window details if missing
				for i := range agents {
					if agents[i].BeadsID == beadsID {
						if agents[i].Window == "" {
							agents[i].Window = w.Target
							agents[i].Title = w.Name
						}
						break
					}
				}
				continue
			}

			agents = append(agents, AgentInfo{
				BeadsID: beadsID,
				Mode:    "claude", // Legacy/untracked tmux agents are claude mode
				Skill:   extractSkillFromWindowName(w.Name),
				Project: extractProjectFromBeadsID(beadsID),
				Window:  w.Target,
				Title:   w.Name,
			})

			if beadsID != "" && !seenBeadsIDs[beadsID] {
				beadsIDsToFetch = append(beadsIDsToFetch, beadsID)
				seenBeadsIDs[beadsID] = true
			}
		}
	}

	// Phase 3: Discovery - Collect beads IDs from active OpenCode sessions (untracked)
	for _, s := range sessions {
		updatedAt := time.Unix(s.Time.Updated/1000, 0)
		if now.Sub(updatedAt) > maxIdleTime {
			continue
		}

		beadsID := extractBeadsIDFromTitle(s.Title)
		if beadsID == "" {
			continue
		}

		// Skip if already tracked via registry or tmux
		if seenBeadsIDs[beadsID] {
			continue
		}

		createdAt := time.Unix(s.Time.Created/1000, 0)
		// updatedAt already declared in loop, so just use existing value
		agents = append(agents, AgentInfo{
			SessionID:    s.ID,
			BeadsID:      beadsID,
			Mode:         "opencode", // Untracked OpenCode sessions are opencode mode
			Title:        s.Title,
			Runtime:      formatDuration(now.Sub(createdAt)),
			LastActivity: updatedAt,
			Skill:        extractSkillFromTitle(s.Title),
			Project:      extractProjectFromBeadsID(beadsID),
			IsProcessing: client.IsSessionProcessing(s.ID),
		})

		beadsIDsToFetch = append(beadsIDsToFetch, beadsID)
		seenBeadsIDs[beadsID] = true
	}

	// Build beadsProjectDirs map for cross-project agents.
	// Strategy 1: Use session.Directory from OpenCode sessions (if valid, not "/").
	// Strategy 2: Look up workspace from current project's .orch/workspace/.
	// Strategy 3: Derive project directory from beads ID prefix by checking known project locations.
	// This ensures we look up beads comments from the correct project's .beads/ directory.
	for beadsID, session := range beadsToSession {
		if session != nil && session.Directory != "" && session.Directory != "/" && session.Directory != projectDir {
			beadsProjectDirs[beadsID] = session.Directory
		}
	}

	// For beads IDs without a valid session directory, try additional strategies
	for _, beadsID := range beadsIDsToFetch {
		if _, hasProjectDir := beadsProjectDirs[beadsID]; hasProjectDir {
			continue // Already have project dir
		}

		// Strategy 2: Look up workspace from current project
		workspacePath, _ := findWorkspaceByBeadsID(projectDir, beadsID)
		if workspacePath != "" {
			agentProjectDir := extractProjectDirFromWorkspace(workspacePath)
			if agentProjectDir != "" {
				beadsProjectDirs[beadsID] = agentProjectDir
				continue
			}
		}

		// Strategy 3: Derive from beads ID prefix
		// Beads IDs have format: project-name-xxxx (e.g., orch-go-3anf, kb-cli-xrm)
		projectName := extractProjectFromBeadsID(beadsID)
		if projectName != "" && projectName != "untracked" {
			if derivedDir := findProjectDirByName(projectName); derivedDir != "" {
				beadsProjectDirs[beadsID] = derivedDir
			}
		}
	}

	// 4. Batch fetch all comments with project-aware lookup for cross-project agents
	commentsMap := verify.GetCommentsBatchWithProjectDirs(beadsIDsToFetch, beadsProjectDirs)

	// 5. Batch fetch issue details to check closed status
	// This also provides task info for closed issues (not returned by ListOpenIssues)
	allIssues, _ := verify.GetIssuesBatch(beadsIDsToFetch)

	// === Now enrich and filter agents ===

	for i := range agents {
		agent := &agents[i]

		// Get phase from pre-fetched comments
		if comments, ok := commentsMap[agent.BeadsID]; ok {
			phaseStatus := verify.ParsePhaseFromComments(comments)
			if phaseStatus.Found {
				agent.Phase = phaseStatus.Phase
			}
		}

		// Get task and check closed status from pre-fetched issues
		if issue, ok := allIssues[agent.BeadsID]; ok {
			agent.Task = truncate(issue.Title, 40)
			agent.IsCompleted = strings.EqualFold(issue.Status, "closed")
		}

		// Determine phantom status
		// For now, if we have a session ID or a tmux window, it's not a phantom
		if agent.SessionID != "" && agent.SessionID != "tmux-stalled" {
			agent.IsPhantom = false
		} else if agent.Window != "" {
			agent.IsPhantom = false
		} else {
			agent.IsPhantom = true
		}

		// If it's claude mode and we matched a session, get runtime
		if agent.Mode == "claude" && agent.SessionID != "" {
			if s, ok := sessionMap[agent.SessionID]; ok {
				createdAt := time.Unix(s.Time.Created/1000, 0)
				agent.Runtime = formatDuration(now.Sub(createdAt))
				if agent.Title == "" {
					agent.Title = s.Title
				}
				agent.IsProcessing = client.IsSessionProcessing(s.ID)
			}
		}

		// Ensure runtime has a value
		if agent.Runtime == "" {
			agent.Runtime = "unknown"
		}
	}

	// Phase 3: Filter agents based on flags
	filteredAgents := make([]AgentInfo, 0)
	for _, agentItem := range agents {
		// Filter by project if specified
		if statusProject != "" && agentItem.Project != statusProject {
			continue
		}

		// Determine status for filtering
		status := "idle"
		if agentItem.IsProcessing {
			status = "running"
		}

		// Apply two-threshold ghost filtering unless --all is set
		if !statusAll {
			// Use IsVisibleByDefault to determine if agent should be shown
			if !agent.IsVisibleByDefault(status, agentItem.LastActivity, agentItem.Phase) {
				continue // Ghost agent - hide by default
			}
		}

		// Filter completed agents (beads issue closed) unless --all is set
		// Note: Phase: Complete agents are handled by IsVisibleByDefault
		if agentItem.IsCompleted && !statusAll {
			continue
		}

		filteredAgents = append(filteredAgents, agentItem)
	}

	// Phase 4: Build swarm status (counts before filtering)
	activeCount := 0
	processingCount := 0
	idleCount := 0
	phantomCount := 0
	completedCount := 0
	for _, agent := range agents {
		if agent.IsPhantom {
			phantomCount++
		} else if agent.IsCompleted {
			// Completed agents (beads issue closed) don't count as active
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
		Queued:     0,              // TODO: implement queuing system
		Completed:  completedCount, // Agents with closed beads issues
	}

	// Fetch account usage information
	accounts := getAccountUsage()

	// Fetch token usage for each agent with a valid session ID
	for i := range filteredAgents {
		if filteredAgents[i].SessionID != "" && filteredAgents[i].SessionID != "tmux-stalled" {
			tokens, err := client.GetSessionTokens(filteredAgents[i].SessionID)
			if err == nil && tokens != nil {
				filteredAgents[i].Tokens = tokens
			}
		}
	}

	// Assess context exhaustion risk for each agent
	for i := range filteredAgents {
		agent := &filteredAgents[i]
		// Skip phantom or completed agents
		if agent.IsPhantom || agent.IsCompleted {
			continue
		}
		// Get total tokens for risk assessment
		totalTokens := 0
		if agent.Tokens != nil {
			totalTokens = agent.Tokens.TotalTokens
			if totalTokens == 0 {
				totalTokens = agent.Tokens.InputTokens + agent.Tokens.OutputTokens
			}
		}
		// Assess risk (uses ProjectDir for git status check)
		risk := verify.AssessContextRisk(totalTokens, agent.ProjectDir, agent.IsProcessing)
		if risk.IsAtRisk() {
			agent.ContextRisk = &risk
		}
	}

	// Fetch orchestrator sessions from registry
	orchestratorSessions := getOrchestratorSessions(statusProject)

	// Check infrastructure health
	infraHealth := checkInfrastructureHealth()

	// Detect synthesis opportunities
	synthesisOpps, _ := verify.DetectSynthesisOpportunities(projectDir)

	// Get session metrics for drift detection (surfaces to Dylan)
	sessionMetrics := getSessionMetrics()

	// Build output (use filtered agents for display)
	output := StatusOutput{
		Infrastructure:         infraHealth,
		SessionMetrics:         sessionMetrics,
		Swarm:                  swarm,
		Accounts:               accounts,
		OrchestratorSessions:   orchestratorSessions,
		Agents:                 filteredAgents,
		SynthesisOpportunities: synthesisOpps,
	}

	// Output as JSON if flag is set
	if statusJSON {
		data, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	// Print human-readable output
	printSwarmStatus(output, statusAll)
	return nil
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
	var accounts []AccountUsage

	// Get current account usage
	currentUsage := usage.FetchUsage()
	if currentUsage.Error == "" && currentUsage.SevenDay != nil {
		current := AccountUsage{
			Name:        "current",
			Email:       currentUsage.Email,
			UsedPercent: currentUsage.SevenDay.Utilization,
			IsActive:    true,
		}
		if currentUsage.SevenDay.ResetsAt != nil {
			current.ResetTime = currentUsage.SevenDay.TimeUntilReset()
		}
		accounts = append(accounts, current)
	}

	// Try to get saved accounts info (without switching)
	cfg, err := account.LoadConfig()
	if err == nil {
		for name, acc := range cfg.Accounts {
			if acc.Source == "saved" {
				// Check if this is the current account (by email match)
				isCurrentAccount := false
				for i := range accounts {
					if accounts[i].Email == acc.Email {
						accounts[i].Name = name // Update name to the saved account name
						isCurrentAccount = true
						break
					}
				}
				if !isCurrentAccount {
					// Add as a saved account (no live usage data without switching)
					accounts = append(accounts, AccountUsage{
						Name:     name,
						Email:    acc.Email,
						IsActive: false,
					})
				}
			}
		}
	}

	return accounts
}

// getOrchestratorSessions fetches active orchestrator sessions from the registry.
// If project is non-empty, filters to only sessions in that project.
func getOrchestratorSessions(project string) []OrchestratorSessionInfo {
	registry := session.NewRegistry("")
	sessions, err := registry.ListActive()
	if err != nil {
		return nil // Silently fail - registry may not exist yet
	}

	now := time.Now()
	var result []OrchestratorSessionInfo

	for _, s := range sessions {
		// Extract project name from directory path
		projectName := filepath.Base(s.ProjectDir)

		// Filter by project if specified
		if project != "" && projectName != project {
			continue
		}

		duration := formatDuration(now.Sub(s.SpawnTime))

		result = append(result, OrchestratorSessionInfo{
			WorkspaceName: s.WorkspaceName,
			Goal:          s.Goal,
			Duration:      duration,
			Project:       projectName,
			Status:        s.Status,
		})
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

	// Print orchestrator sessions
	if len(output.OrchestratorSessions) > 0 {
		printOrchestratorSessions(output.OrchestratorSessions, termWidth)
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

// printOrchestratorSessions prints orchestrator sessions in a table format.
func printOrchestratorSessions(sessions []OrchestratorSessionInfo, termWidth int) {
	fmt.Println("ORCHESTRATOR SESSIONS")

	if termWidth < termWidthMin {
		// Card format for very narrow terminals
		for i, s := range sessions {
			if i > 0 {
				fmt.Println()
			}
			fmt.Printf("  %s [%s]\n", s.WorkspaceName, s.Status)
			fmt.Printf("    Goal: %s\n", truncate(s.Goal, 50))
			fmt.Printf("    Duration: %s | Project: %s\n", s.Duration, s.Project)
		}
	} else if termWidth < termWidthNarrow {
		// Narrow format - drop goal column
		fmt.Printf("  %-40s %-10s %s\n", "WORKSPACE", "DURATION", "PROJECT")
		fmt.Printf("  %s\n", strings.Repeat("-", 65))
		for _, s := range sessions {
			project := s.Project
			if project == "" {
				project = "-"
			}
			fmt.Printf("  %-40s %-10s %s\n",
				truncate(s.WorkspaceName, 38),
				s.Duration,
				project)
		}
	} else {
		// Wide format - full table
		fmt.Printf("  %-40s %-30s %-10s %s\n", "WORKSPACE", "GOAL", "DURATION", "PROJECT")
		fmt.Printf("  %s\n", strings.Repeat("-", 95))
		for _, s := range sessions {
			project := s.Project
			if project == "" {
				project = "-"
			}
			fmt.Printf("  %-40s %-30s %-10s %s\n",
				truncate(s.WorkspaceName, 38),
				truncate(s.Goal, 28),
				s.Duration,
				project)
		}
	}
}

// printAgentsWideFormat prints agents in full table format (>120 chars).
// Columns: BEADS ID, STATUS, PHASE, TASK, SKILL, RUNTIME, TOKENS, RISK
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
		fmt.Printf("  %-18s %-8s %-8s %-12s %-20s %-25s %-12s %-7s %-16s %s\n", "BEADS ID", "MODE", "MODEL", "STATUS", "PHASE", "TASK", "SKILL", "RUNTIME", "TOKENS", "RISK")
		fmt.Printf("  %s\n", strings.Repeat("-", 145))
	} else {
		fmt.Printf("  %-18s %-8s %-20s %-8s %-12s %-23s %-12s %-8s %s\n", "BEADS ID", "MODE", "MODEL", "STATUS", "PHASE", "TASK", "SKILL", "RUNTIME", "TOKENS")
		fmt.Printf("  %s\n", strings.Repeat("-", 135))
	}

	for _, agent := range agents {
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
			fmt.Printf("  %-18s %-8s %-20s %-8s %-12s %-25s %-12s %-7s %-16s %s\n",
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
			fmt.Printf("  %-18s %-8s %-20s %-8s %-12s %-23s %-12s %-8s %s\n",
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
// Columns: BEADS ID, MODEL, STATUS, PHASE, SKILL, RUNTIME, TOKENS
func printAgentsNarrowFormat(agents []AgentInfo) {
	fmt.Printf("  %-18s %-10s %-8s %-10s %-8s %-8s %s\n", "BEADS ID", "MODEL", "STATUS", "PHASE", "SKILL", "RUNTIME", "TOKENS")
	fmt.Printf("  %s\n", strings.Repeat("-", 85))

	for _, agent := range agents {
		beadsID := formatBeadsIDForDisplay(agent.BeadsID)
		if beadsID == "" {
			beadsID = "-"
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

		fmt.Printf("  %-18s %-10s %-8s %-10s %-8s %-8s %s\n",
			beadsID,
			truncate(modelDisplay, 9),
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
		beadsID := agent.BeadsID
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
			fmt.Printf("  %s [%s] %s\n", beadsID, status, riskStr)
		} else {
			fmt.Printf("  %s [%s]\n", beadsID, status)
		}
		fmt.Printf("    Model: %s | Phase: %s | Skill: %s\n", modelDisplay, phase, skill)
		fmt.Printf("    Task: %s\n", truncate(task, 50))
		fmt.Printf("    Runtime: %s | Tokens: %s\n", agent.Runtime, formatTokenStats(agent.Tokens))
		if agent.ContextRisk != nil && agent.ContextRisk.Reason != "" {
			fmt.Printf("    Risk: %s\n", agent.ContextRisk.Reason)
		}
	}
}

// getAgentStatus returns a status string based on agent state.
func getAgentStatus(agent AgentInfo) string {
	if agent.IsCompleted {
		return "completed"
	}
	if agent.IsPhantom {
		return "phantom"
	}
	if agent.IsProcessing {
		return "running"
	}
	return "idle"
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

// findProjectDirByName looks up a project directory by its name.
// Searches common project locations and verifies the project has a .beads/ directory.
// Returns empty string if not found.
func findProjectDirByName(projectName string) string {
	// Get home directory
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
