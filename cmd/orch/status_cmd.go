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
	"sync"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/state"
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
	// HTTP timeout for status command API requests.
	// Keep this short so monitoring stays responsive when OpenCode is degraded.
	statusAPIRequestTimeout = 5 * time.Second

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
	SessionID          string                        `json:"session_id"`
	BeadsID            string                        `json:"beads_id,omitempty"`
	Mode               string                        `json:"mode,omitempty"`  // Agent mode: "claude" or "opencode"
	Model              string                        `json:"model,omitempty"` // Model spec (e.g., "gemini-3-flash-preview", "claude-opus-4-5-20251101")
	Skill              string                        `json:"skill,omitempty"`
	Account            string                        `json:"account,omitempty"`
	Runtime            string                        `json:"runtime"`
	Title              string                        `json:"title,omitempty"`
	Window             string                        `json:"window,omitempty"`
	Phase              string                        `json:"phase,omitempty"`                // Current phase from beads comments
	Task               string                        `json:"task,omitempty"`                 // Task description (truncated)
	Project            string                        `json:"project,omitempty"`              // Project name derived from beads ID or workspace
	ProjectDir         string                        `json:"project_dir,omitempty"`          // Full path to project directory (for cross-project agents)
	Source             string                        `json:"source,omitempty"`               // Source where agent originated: T=tmux, O=opencode, B=beads, W=workspace
	IsPhantom          bool                          `json:"is_phantom,omitempty"`           // True if beads issue open but agent not running
	IsProcessing       bool                          `json:"is_processing,omitempty"`        // True if session is actively generating a response
	IsCompleted        bool                          `json:"is_completed,omitempty"`         // True if beads issue is closed
	IsUntracked        bool                          `json:"is_untracked,omitempty"`         // True if session has no beads tracking (OpenCode-only, or spawned with --no-track)
	HasGhostCompletion bool                          `json:"has_ghost_completion,omitempty"` // True if Phase: Complete but 0 commits (ghost completion early detection)
	Tokens             *opencode.TokenStats          `json:"tokens,omitempty"`               // Token usage for the session
	ContextRisk        *verify.ContextExhaustionRisk `json:"context_risk,omitempty"`         // Context exhaustion risk assessment
	PhaseReportedAt    *time.Time                    `json:"phase_reported_at,omitempty"`    // Timestamp when latest phase was reported
	LastActivity       time.Time                     `json:"last_activity,omitempty"`        // Timestamp of last activity (for ghost filtering)
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
	DriftMetrics           *state.DriftMetrics            `json:"drift_metrics,omitempty"`
}

func runStatus(serverURL string) error {
	return runStatusWithClient(opencode.NewClientWithTimeout(serverURL, statusAPIRequestTimeout), serverURL)
}

func runStatusWithClient(client opencode.ClientInterface, serverURL string) error {
	debugTiming := os.Getenv("ORCH_STATUS_DEBUG") != ""
	timerStart := time.Now()
	timer := func(label string) {
		if debugTiming {
			fmt.Fprintf(os.Stderr, "[timing] %s: %v\n", label, time.Since(timerStart))
		}
	}

	now := time.Now()
	projectDir, _ := currentProjectDir()

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
	sessions, opencodeSessionsAvailable := listSessionsForStatus(client, listOpts, debugTiming)
	timer("ListSessions")

	var agents []AgentInfo

	if stateDBResult != nil && len(stateDBResult.agents) > 0 {
		// === STATE DB PATH: Fast path using cached immutable fields ===
		if debugTiming {
			fmt.Fprintf(os.Stderr, "[timing] stateDB: found %d agents, using fast path\n", len(stateDBResult.agents))
		}

		// Enrich state DB agents with live data (processing, tokens, phase, issues)
		enrichStateDBAgentsLive(stateDBResult, client, sessions, now, statusAll, projectDir, opencodeSessionsAvailable, timer)

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

	// Parallel token fetch + risk assessment for agents not already enriched.
	// Most agents already have tokens from GetSessionEnrichment; this only covers
	// agents discovered via tmux (claude mode) that were matched to sessions later.
	var tokenWg sync.WaitGroup
	for i := range filteredAgents {
		agent := &filteredAgents[i]

		// Determine if this agent needs token fetch
		needsTokenFetch := opencodeSessionsAvailable &&
			agent.Tokens == nil &&
			agent.SessionID != "" && agent.SessionID != "tmux-stalled" &&
			(statusAll || agent.IsProcessing)

		// Determine if this agent needs risk assessment
		needsRisk := !agent.IsPhantom && !agent.IsCompleted &&
			(statusAll || agent.IsProcessing)

		if !needsTokenFetch && !needsRisk {
			continue
		}

		tokenWg.Add(1)
		go func(a *AgentInfo, fetchTokens, assessRisk bool) {
			defer tokenWg.Done()

			// Fetch tokens if needed
			if fetchTokens {
				tokens, err := client.GetSessionTokens(a.SessionID)
				if err == nil && tokens != nil {
					a.Tokens = tokens
				}
			}

			// Assess context exhaustion risk
			if assessRisk {
				totalTokens := 0
				if a.Tokens != nil {
					totalTokens = a.Tokens.TotalTokens
					if totalTokens == 0 {
						totalTokens = a.Tokens.InputTokens + a.Tokens.OutputTokens
					}
				}
				risk := verify.AssessContextRisk(totalTokens, a.ProjectDir, a.IsProcessing)
				if risk.IsAtRisk() {
					a.ContextRisk = &risk
				}
			}
		}(agent, needsTokenFetch, needsRisk)
	}
	tokenWg.Wait()
	timer("token fetch + risk assessment")

	// Fetch orchestrator sessions from session registry
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

	// Fetch drift metrics from state DB (fast, <1ms)
	var driftMetrics *state.DriftMetrics
	if driftDB, err := state.OpenDefault(); err == nil && driftDB != nil {
		driftMetrics, _ = driftDB.GetDriftMetrics()
		driftDB.Close()
	}

	// Build output (use filtered agents for display)
	output := StatusOutput{
		Infrastructure:         infraHealth,
		Swarm:                  swarm,
		Accounts:               accounts,
		OrchestratorSessions:   orchestratorSessions,
		Agents:                 filteredAgents,
		SynthesisOpportunities: synthesisOpps,
		DriftMetrics:           driftMetrics,
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
