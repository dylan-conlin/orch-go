// Package main provides the status command for showing swarm status and active agents.
// Extracted from main.go as part of the main.go refactoring (Phase 3).
//
// The status command is split across multiple files:
//   - status_cmd.go: Command definition, types, flags, and runStatus orchestration
//   - status_agents.go: Agent discovery, enrichment, filtering, and project lookup
//   - status_health.go: Infrastructure health checks (TCP, daemon)
//   - status_display.go: All print/format functions for terminal output
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/registry"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/dylan-conlin/orch-go/pkg/verify"
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
	// Only show orchestrator sessions from the last N hours in compact mode
	compactOrchestratorSessionsMaxAge = 6 * time.Hour
	// Maximum orchestrator sessions to show in compact mode
	compactOrchestratorSessionsLimit = 5
	// Only show Phase: Complete agents from the last N hours in compact mode
	compactCompletedAgentsMaxAge = 6 * time.Hour
	// Only fetch processing status for sessions updated within this window
	processingCheckMaxAge = 5 * time.Minute
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
	Untracked  int `json:"untracked,omitempty"`  // Sessions not tracked in beads (no beads ID, or spawned with --no-track)
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
	IsUntracked     bool                          `json:"is_untracked,omitempty"`      // True if session has no beads tracking (OpenCode-only, or spawned with --no-track)
	Tokens          *opencode.TokenStats          `json:"tokens,omitempty"`            // Token usage for the session
	ContextRisk     *verify.ContextExhaustionRisk `json:"context_risk,omitempty"`      // Context exhaustion risk assessment
	PhaseReportedAt *time.Time                    `json:"phase_reported_at,omitempty"` // Timestamp when latest phase was reported
	LastActivity    time.Time                     `json:"last_activity,omitempty"`     // Timestamp of last activity (for ghost filtering)
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

// StatusOutput represents the full status output for JSON serialization.
type StatusOutput struct {
	Infrastructure         *InfrastructureHealth          `json:"infrastructure,omitempty"`
	Swarm                  SwarmStatus                    `json:"swarm"`
	Accounts               []AccountUsage                 `json:"accounts"`
	OrchestratorSessions   []OrchestratorSessionInfo      `json:"orchestrator_sessions,omitempty"`
	Agents                 []AgentInfo                    `json:"agents"`
	SynthesisOpportunities *verify.SynthesisOpportunities `json:"synthesis_opportunities,omitempty"`
}

func runStatus(serverURL string) error {
	client := opencode.NewClient(serverURL)
	now := time.Now()

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

	// Phase 1: Discovery - Collect agents from tmux windows (claude mode agents)
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

			// Skip if already seen (duplicate window)
			if seenBeadsIDs[beadsID] {
				continue
			}

			// Build agent info from tmux window
			agentProject := extractProjectFromBeadsID(beadsID)
			info := AgentInfo{
				BeadsID: beadsID,
				Mode:    "claude", // tmux agents are claude mode
				Skill:   extractSkillFromWindowName(w.Name),
				Project: agentProject,
				Window:  w.Target,
				Title:   w.Name,
			}

			// Determine the correct project directory for workspace lookup
			// For cross-project agents, we need to look in the agent's project, not cwd
			agentProjectDir := projectDir
			if agentProject != "" && agentProject != filepath.Base(projectDir) {
				// Cross-project agent: look up the actual project directory
				if derivedDir := findProjectDirByName(agentProject); derivedDir != "" {
					agentProjectDir = derivedDir
					info.ProjectDir = derivedDir
					// Always set beadsProjectDirs for cross-project agents
					// This ensures beads comments are fetched from the correct project
					beadsProjectDirs[beadsID] = derivedDir
				}
			}

			// Try to enrich with workspace metadata (skill, projectDir, model)
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

	// Phase 1.5: Discovery - Collect agents from registry (claude-mode agents not visible via tmux)
	// This catches:
	// - Claude inline mode agents (no tmux window)
	// - Claude agents whose tmux window closed but are still running
	// - Docker mode agents without active tmux windows
	agentReg, _ := registry.New("")
	if agentReg != nil {
		// Batch fetch all existing tmux window targets once (O(1) vs O(n) subprocess calls)
		existingWindows := tmux.ListAllWindowTargets()

		for _, regAgent := range agentReg.ListActive() {
			// Only process claude and docker mode agents (opencode mode is handled in Phase 2)
			if regAgent.Mode != registry.ModeTmux && regAgent.Mode != registry.ModeDocker {
				continue
			}

			beadsID := regAgent.BeadsID
			if beadsID == "" {
				continue
			}

			// Skip if already discovered via tmux windows
			if seenBeadsIDs[beadsID] {
				continue
			}

			// Build agent info from registry
			info := AgentInfo{
				BeadsID:    beadsID,
				Mode:       regAgent.Mode,
				Skill:      regAgent.Skill,
				Project:    extractProjectFromBeadsID(beadsID),
				ProjectDir: regAgent.ProjectDir,
				Title:      regAgent.ID, // Workspace name as title
				Model:      regAgent.Model,
			}

			// If registry has tmux window info, verify it still exists using batch lookup
			if regAgent.TmuxWindow != "" {
				if existingWindows[regAgent.TmuxWindow] {
					info.Window = regAgent.TmuxWindow
				}
				// Note: If window doesn't exist, agent is a phantom (will be handled later)
			}

			// Set project directory for cross-project lookups
			if info.ProjectDir != "" {
				beadsProjectDirs[beadsID] = info.ProjectDir
			}

			agents = append(agents, info)
			beadsIDsToFetch = append(beadsIDsToFetch, beadsID)
			seenBeadsIDs[beadsID] = true
		}
	}

	// Phase 2: Discovery - Collect agents from active OpenCode sessions (opencode/headless mode)
	for _, s := range sessions {
		updatedAt := time.Unix(s.Time.Updated/1000, 0)
		if now.Sub(updatedAt) > maxIdleTime {
			continue
		}

		beadsID := extractBeadsIDFromTitle(s.Title)
		if beadsID == "" {
			continue
		}

		// Skip if already tracked via tmux
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
			IsProcessing: isSessionLikelyProcessing(client, s.ID, updatedAt, now),
			Model:        client.GetSessionModel(s.ID),
		})

		beadsIDsToFetch = append(beadsIDsToFetch, beadsID)
		seenBeadsIDs[beadsID] = true
	}

	// Phase 3: Discovery - Collect UNTRACKED OpenCode sessions (no beads ID in title)
	// These are sessions started directly through OpenCode, not via orch spawn
	// Track seen session IDs to avoid duplicates
	seenSessionIDs := make(map[string]bool)
	for _, agent := range agents {
		if agent.SessionID != "" {
			seenSessionIDs[agent.SessionID] = true
		}
	}

	for _, s := range sessions {
		// Skip if already tracked
		if seenSessionIDs[s.ID] {
			continue
		}

		// Skip sessions with beads ID (already handled in Phase 2)
		if extractBeadsIDFromTitle(s.Title) != "" {
			continue
		}

		updatedAt := time.Unix(s.Time.Updated/1000, 0)
		// For untracked sessions, use a longer idle threshold since they may be
		// orchestrator conversations or long-running interactive sessions
		untrackedMaxIdleTime := 2 * time.Hour
		if now.Sub(updatedAt) > untrackedMaxIdleTime {
			continue
		}

		createdAt := time.Unix(s.Time.Created/1000, 0)

		// Extract project name from directory (e.g., /Users/.../orch-go -> orch-go)
		projectName := ""
		if s.Directory != "" && s.Directory != "/" {
			projectName = filepath.Base(s.Directory)
		}

		agents = append(agents, AgentInfo{
			SessionID:    s.ID,
			BeadsID:      "", // No beads ID - this is an untracked session
			Mode:         "opencode",
			Title:        s.Title,
			Runtime:      formatDuration(now.Sub(createdAt)),
			LastActivity: updatedAt,
			Project:      projectName,
			ProjectDir:   s.Directory,
			IsUntracked:  true,
			IsProcessing: isSessionLikelyProcessing(client, s.ID, updatedAt, now),
			Model:        client.GetSessionModel(s.ID),
		})

		seenSessionIDs[s.ID] = true
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
	allIssues, _ := verify.GetIssuesBatch(beadsIDsToFetch, beadsProjectDirs)

	// === Now enrich and filter agents ===

	for i := range agents {
		agent := &agents[i]

		// Get phase from pre-fetched comments
		if comments, ok := commentsMap[agent.BeadsID]; ok {
			phaseStatus := verify.ParsePhaseFromComments(comments)
			if phaseStatus.Found {
				agent.Phase = phaseStatus.Phase
				agent.PhaseReportedAt = phaseStatus.PhaseReportedAt
			}
		}

		// Get task and check closed status from pre-fetched issues
		issue, issueExists := allIssues[agent.BeadsID]
		if issueExists && issue != nil {
			agent.Task = truncate(issue.Title, 40)
			agent.IsCompleted = strings.EqualFold(issue.Status, "closed")
		}

		// Treat --no-track spawns (project-untracked-*) as untracked for swarm accounting.
		// These IDs intentionally do not exist in beads.
		if agent.BeadsID != "" && isUntrackedBeadsID(agent.BeadsID) {
			agent.IsUntracked = true
		}

		// Determine phantom status.
		// Phantom means: beads issue exists AND is open AND there is no active runtime.
		// Note: --no-track IDs are never phantom because they have no beads issue by design.
		agent.IsPhantom = computeIsPhantom(*agent, issue, issueExists)

		// Determine source indicator
		agent.Source = determineAgentSource(*agent, projectDir)

		// If it's claude mode and we matched a session, get runtime
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

		// For tmux-based agents, check tmux pane activity as primary or fallback signal
		// This handles:
		// 1. Pure claude CLI mode agents that don't have an OpenCode session
		// 2. OpenCode session-based agents where session reports idle but tmux shows activity
		//    (e.g., agent is reading files or thinking - no new messages but process is running)
		if agent.Window != "" {
			paneRunning := tmux.IsPaneProcessRunning(agent.Window)
			if agent.SessionID == "" {
				// No OpenCode session - tmux pane activity is the only signal
				agent.IsProcessing = paneRunning
			} else if !agent.IsProcessing && paneRunning {
				// OpenCode session said idle, but tmux shows active process
				// Trust tmux as the more direct signal of agent activity
				agent.IsProcessing = true
			}
		}

		// Ensure runtime has a value
		if agent.Runtime == "" {
			agent.Runtime = "unknown"
		}
	}

	// Phase 3: Filter agents based on flags
	// Compact mode (default): Only show running agents + recently completed (Phase: Complete)
	// Full mode (--all): Show all agents including idle, phantom, completed
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
		// 4. Untracked sessions (always visible for resource monitoring)
		if !statusAll && !agentItem.IsUntracked {
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
				continue // Skip idle or stale complete agents in compact mode
			}
		}

		// Filter completed agents (beads issue closed) unless --all is set
		if agentItem.IsCompleted && !statusAll {
			continue
		}

		// Filter phantom agents unless --all is set
		if agentItem.IsPhantom && !statusAll {
			continue
		}

		// Untracked sessions are shown by default for operational visibility.
		// These are OpenCode sessions without beads ID tracking - typically interactive
		// human-to-AI conversations or sessions started outside orch spawn.
		// They represent real OpenCode resource usage and should be visible for monitoring.

		filteredAgents = append(filteredAgents, agentItem)
	}

	// Phase 5: Build swarm status (counts before filtering)
	swarm := computeSwarmStatus(agents)

	// Fetch account usage information
	accounts := getAccountUsage()

	// Fetch token usage - in compact mode, only for running agents (expensive operation)
	for i := range filteredAgents {
		agent := &filteredAgents[i]
		// Skip if no valid session ID
		if agent.SessionID == "" || agent.SessionID == "tmux-stalled" {
			continue
		}
		// In compact mode, only fetch tokens for running agents (saves ~100ms per idle agent)
		if !statusAll && !agent.IsProcessing {
			continue
		}
		tokens, err := client.GetSessionTokens(agent.SessionID)
		if err == nil && tokens != nil {
			agent.Tokens = tokens
		}
	}

	// Assess context exhaustion risk - in compact mode, only for running agents
	for i := range filteredAgents {
		agent := &filteredAgents[i]
		// Skip phantom or completed agents
		if agent.IsPhantom || agent.IsCompleted {
			continue
		}
		// In compact mode, only assess risk for running agents
		if !statusAll && !agent.IsProcessing {
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
	// In compact mode, limit to recent sessions
	if !statusAll && len(orchestratorSessions) > compactOrchestratorSessionsLimit {
		orchestratorSessions = orchestratorSessions[:compactOrchestratorSessionsLimit]
	}

	// Check infrastructure health
	infraHealth := checkInfrastructureHealth()

	// Detect synthesis opportunities - skip in compact mode (expensive filesystem scan)
	var synthesisOpps *verify.SynthesisOpportunities
	if statusAll {
		synthesisOpps, _ = verify.DetectSynthesisOpportunities(projectDir)
	}

	// Build output (use filtered agents for display)
	output := StatusOutput{
		Infrastructure:         infraHealth,
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
