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
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
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
	debugTiming := os.Getenv("ORCH_STATUS_DEBUG") != ""
	timerStart := time.Now()
	timer := func(label string) {
		if debugTiming {
			fmt.Fprintf(os.Stderr, "[timing] %s: %v\n", label, time.Since(timerStart))
		}
	}

	// Use a longer timeout for status - with many sessions, OpenCode API can be slow
	client := opencode.NewClientWithTimeout(serverURL, 30*time.Second)
	now := time.Now()
	projectDir, _ := os.Getwd()

	// === Start independent async operations early ===
	// These run concurrently with agent discovery to hide their latency.

	// Fetch account usage (2 HTTP calls to Anthropic API, ~400ms)
	accountsCh := make(chan []AccountUsage, 1)
	go func() {
		accountsCh <- getAccountUsage()
	}()

	// Check infrastructure health (TCP connect tests, ~50ms)
	infraCh := make(chan *InfrastructureHealth, 1)
	go func() {
		infraCh <- checkInfrastructureHealth()
	}()

	// === PHASE A: Try state DB for immutable fields (fast path) ===
	// If the state DB has agents, use cached immutable data and only query
	// live sources (OpenCode, tmux) for mutable fields.
	stateDBResult := fetchAgentsFromStateDB(statusAll)
	timer("stateDB fetch")

	// === Fetch OpenCode sessions ===
	// Needed both for the state DB path (to discover untracked sessions and enrich live data)
	// and for the fallback path (full multi-source discovery).
	var listOpts *opencode.ListSessionsOpts
	if !statusAll {
		listOpts = &opencode.ListSessionsOpts{
			Start: now.Add(-1 * time.Hour).UnixMilli(),
		}
	}
	sessions, err := client.ListSessionsWithOpts("", listOpts)
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}
	timer("ListSessions")

	var agents []AgentInfo

	if stateDBResult != nil && len(stateDBResult.agents) > 0 {
		// === STATE DB PATH: Fast path using cached immutable fields ===
		if debugTiming {
			fmt.Fprintf(os.Stderr, "[timing] stateDB: found %d agents, using fast path\n", len(stateDBResult.agents))
		}

		// Enrich state DB agents with live data (processing, tokens, phase, issues)
		enrichStateDBAgentsLive(stateDBResult, client, sessions, now, statusAll, projectDir, timer)

		// Discover any agents NOT in state DB (untracked sessions, legacy agents)
		fallbackAgents, _, fallbackProjectDirs, _ := fallbackDiscoverAgents(sessions, now, projectDir)
		mergeDiscoveredAgents(stateDBResult, fallbackAgents, fallbackProjectDirs)

		// For newly discovered agents (not from state DB), run the legacy enrichment.
		// These are agents found via fallback that weren't in the state DB.
		// The enrichStateDBAgentsLive already handled state DB agents.
		// Newly merged agents still need comments/issues/enrichment.
		// For now, they get displayed with whatever data the discovery found.
		// Future: enrich them too (but this is the fallback path, so less critical).

		agents = stateDBResult.agents
		timer("stateDB path complete")
	} else {
		// === FALLBACK PATH: Full multi-source discovery ===
		// State DB is empty or unavailable. Use the existing distributed JOIN path.
		if debugTiming {
			fmt.Fprintf(os.Stderr, "[timing] stateDB: empty/unavailable, using fallback path\n")
		}

		agents = runStatusFallbackPath(client, sessions, now, projectDir, timer)
		timer("fallback path complete")
	}

	// === Filter agents based on flags ===
	filteredAgents := filterAgentsForDisplay(agents, statusAll, statusProject)
	timer("agent filtering")

	// Phase 5: Build swarm status (counts before filtering)
	swarm := computeSwarmStatus(agents)

	// Fetch account usage (already started in parallel above)
	accounts := <-accountsCh
	timer("accounts received")

	// Fetch token usage for agents that weren't enriched in the parallel batch.
	// Most agents already have tokens from GetSessionEnrichment; this only covers
	// agents discovered via tmux (claude mode) that were matched to sessions later.
	for i := range filteredAgents {
		agent := &filteredAgents[i]
		// Skip if already have tokens from parallel enrichment
		if agent.Tokens != nil {
			continue
		}
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

	timer("token fetch + risk assessment")

	// Fetch orchestrator sessions from registry
	orchestratorSessions := getOrchestratorSessions(statusProject)
	// In compact mode, limit to recent sessions
	if !statusAll && len(orchestratorSessions) > compactOrchestratorSessionsLimit {
		orchestratorSessions = orchestratorSessions[:compactOrchestratorSessionsLimit]
	}

	// Get infrastructure health (already started in parallel above)
	infraHealth := <-infraCh
	timer("infra health + orch sessions")

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
