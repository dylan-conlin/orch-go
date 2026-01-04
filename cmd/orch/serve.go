package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	_ "net/http/pprof" // Enable pprof for CPU profiling
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/account"
	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/focus"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/port"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/dylan-conlin/orch-go/pkg/usage"
	"github.com/dylan-conlin/orch-go/pkg/userconfig"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
)

// DefaultServePort is the port orch serve listens on.
// This is infrastructure, not a project dev server.
const DefaultServePort = 3348

var (
	servePort int

	// beadsClient is a persistent RPC client for beads operations.
	// Initialized at startup with auto-reconnect enabled.
	beadsClient *beads.Client
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP API server for the beads-ui dashboard",
	Long: `Start an HTTP API server that provides endpoints for the beads-ui dashboard.

This is orchestration infrastructure (persistent monitoring), NOT a project
dev server. Use 'orch serve status' to check if the API is running.

Endpoints:
  GET /api/agents    - Returns JSON list of active agents from OpenCode/tmux
  GET /api/events    - Proxies the OpenCode SSE stream for real-time updates
  GET /api/agentlog  - Agent lifecycle events
  GET /api/usage     - Claude Max usage stats
  GET /api/focus     - Current focus and drift status
  GET /api/beads     - Beads stats (ready, blocked, open)
  GET /api/beads/ready - List of ready issues for queue visibility
  GET /api/servers   - Servers status across projects
  GET /api/daemon    - Daemon status (running, capacity, last poll)
  GET /api/gaps      - Gap tracker stats (total, recurring, by-skill)
  GET /api/reflect   - Reflect suggestions (synthesis, promote, stale)
  GET /api/errors    - Error pattern analysis (recent errors, recurring patterns)
  GET/PUT /api/config - User configuration settings (~/.orch/config.yaml)
  GET /api/changelog - Aggregated changelog (?days=7&project=all)
  GET /health        - Health check

Examples:
  orch-go serve              # Start server on port 3348
  orch-go serve --port 8080  # Override with explicit port
  orch-go serve status       # Check if server is running`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServe(servePort)
	},
}

var serveStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check if the orch serve API is running",
	Long: `Check if the orch serve API server is running and accessible.

This command checks if the API server is listening on the expected port
and returns its status. The API server is orchestration infrastructure,
separate from project dev servers managed by 'orch servers'.

Examples:
  orch-go serve status           # Check status on default port
  orch-go serve status -p 8080   # Check status on custom port`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServeStatus(servePort)
	},
}

func init() {
	serveCmd.Flags().IntVarP(&servePort, "port", "p", DefaultServePort, "Port to check/listen on")
	serveStatusCmd.Flags().IntVarP(&servePort, "port", "p", DefaultServePort, "Port to check")

	serveCmd.AddCommand(serveStatusCmd)
	rootCmd.AddCommand(serveCmd)
}

// runServeStatus checks if the orch serve API is running on the given port.
func runServeStatus(portNum int) error {
	addr := fmt.Sprintf("http://localhost:%d/health", portNum)

	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := client.Get(addr)
	if err != nil {
		fmt.Printf("❌ API server is NOT running on port %d\n", portNum)
		fmt.Println()
		fmt.Println("Start it with: orch serve")
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("⚠️  API server on port %d returned status %d\n", portNum, resp.StatusCode)
		return nil
	}

	// Parse the health response
	var health struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		fmt.Printf("✅ API server is running on port %d (health check responded)\n", portNum)
		return nil
	}

	fmt.Printf("✅ API server is running on port %d\n", portNum)
	fmt.Printf("   Status: %s\n", health.Status)
	fmt.Printf("   URL:    http://localhost:%d\n", portNum)
	fmt.Println()
	fmt.Println("Endpoints:")
	fmt.Println("  GET /api/agents    - Active agents")
	fmt.Println("  GET /api/events    - SSE event stream")
	fmt.Println("  GET /api/agentlog  - Agent lifecycle events")
	fmt.Println("  GET /api/usage     - Claude Max usage")
	fmt.Println("  GET /api/focus     - Focus and drift status")
	fmt.Println("  GET /api/beads     - Beads stats")
	fmt.Println("  GET /api/beads/ready - Ready issues list")
	fmt.Println("  GET /api/servers   - Project servers status")
	fmt.Println("  POST /api/issues   - Create new beads issue (for follow-ups)")
	fmt.Println("  GET /api/daemon    - Daemon status (running, capacity, last poll)")
	fmt.Println("  GET /api/gaps      - Gap tracker stats")
	fmt.Println("  GET /api/reflect   - Reflect suggestions")
	fmt.Println("  GET /api/errors    - Error pattern analysis")
	fmt.Println("  GET /api/changelog - Aggregated changelog")
	fmt.Println("  GET /health        - Health check")

	return nil
}

func runServe(portNum int) error {
	// Set default directory for beads socket discovery
	// This is needed because serve may run from any working directory
	if sourceDir != "" && sourceDir != "unknown" {
		beads.DefaultDir = sourceDir
	}

	// Initialize persistent beads client with auto-reconnect.
	// This avoids per-request connection overhead and handles daemon restarts.
	socketPath, err := beads.FindSocketPath(sourceDir)
	if err == nil {
		beadsClient = beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		if connErr := beadsClient.Connect(); connErr != nil {
			// Non-fatal: handlers will fallback to CLI if client is nil
			fmt.Printf("Warning: beads daemon not available, using CLI fallback: %v\n", connErr)
			beadsClient = nil
		}
	}

	mux := http.NewServeMux()

	// CORS middleware wrapper
	corsHandler := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Allow requests from SvelteKit dev server and any localhost
			origin := r.Header.Get("Origin")
			if origin == "" || strings.HasPrefix(origin, "http://localhost") || strings.HasPrefix(origin, "http://127.0.0.1") {
				if origin != "" {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				} else {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				}
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept")

			// Handle preflight
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			h(w, r)
		}
	}

	// GET /api/agents - returns JSON list of agents from OpenCode/tmux
	mux.HandleFunc("/api/agents", corsHandler(handleAgents))

	// GET /api/events - proxies OpenCode SSE stream
	mux.HandleFunc("/api/events", corsHandler(handleEvents))

	// GET /api/agentlog - returns agent lifecycle events from events.jsonl
	mux.HandleFunc("/api/agentlog", corsHandler(handleAgentlog))

	// GET /api/usage - returns Claude Max usage stats
	mux.HandleFunc("/api/usage", corsHandler(handleUsage))

	// GET /api/focus - returns current focus and drift status
	mux.HandleFunc("/api/focus", corsHandler(handleFocus))

	// GET /api/beads - returns beads stats (ready, blocked, open issues)
	mux.HandleFunc("/api/beads", corsHandler(handleBeads))

	// GET /api/beads/ready - returns list of ready issues for dashboard queue visibility
	mux.HandleFunc("/api/beads/ready", corsHandler(handleBeadsReady))

	// GET /api/servers - returns servers status across projects
	mux.HandleFunc("/api/servers", corsHandler(handleServers))

	// POST /api/issues - creates a new beads issue (for follow-up from synthesis)
	mux.HandleFunc("/api/issues", corsHandler(handleIssues))

	// GET /api/daemon - returns daemon status (running, capacity, last poll)
	mux.HandleFunc("/api/daemon", corsHandler(handleDaemon))

	// GET /api/gaps - returns gap tracker statistics
	mux.HandleFunc("/api/gaps", corsHandler(handleGaps))

	// GET /api/reflect - returns reflect suggestions for kb reflect UI
	mux.HandleFunc("/api/reflect", corsHandler(handleReflect))

	// GET /api/errors - returns error pattern analysis
	mux.HandleFunc("/api/errors", corsHandler(handleErrors))

	// GET /api/pending-reviews - returns agents with unreviewed synthesis recommendations
	mux.HandleFunc("/api/pending-reviews", corsHandler(handlePendingReviews))

	// POST /api/dismiss-review - dismiss a specific recommendation
	mux.HandleFunc("/api/dismiss-review", corsHandler(handleDismissReview))

	// GET/PUT /api/config - user configuration settings
	mux.HandleFunc("/api/config", corsHandler(handleConfig))

	// GET /api/changelog - aggregated changelog across ecosystem repos
	mux.HandleFunc("/api/changelog", corsHandler(handleChangelog))

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// pprof handlers for CPU profiling (useful for debugging CPU runaway)
	// Access at: http://localhost:3348/debug/pprof/
	mux.HandleFunc("/debug/pprof/", http.DefaultServeMux.ServeHTTP)

	addr := fmt.Sprintf(":%d", portNum)
	fmt.Printf("Starting orch-go API server on http://localhost%s\n", addr)
	fmt.Println("Endpoints:")
	fmt.Println("  GET /api/agents    - List of active agents from OpenCode/tmux")
	fmt.Println("  GET /api/events    - SSE proxy for OpenCode events")
	fmt.Println("  GET /api/agentlog  - Agent lifecycle events (supports ?follow=true for SSE)")
	fmt.Println("  GET /api/usage     - Claude Max usage stats")
	fmt.Println("  GET /api/focus     - Current focus and drift status")
	fmt.Println("  GET /api/beads     - Beads stats (ready, blocked, open)")
	fmt.Println("  GET /api/beads/ready - List of ready issues for queue visibility")
	fmt.Println("  GET /api/servers   - Servers status across projects")
	fmt.Println("  POST /api/issues   - Create new beads issue (for follow-ups)")
	fmt.Println("  GET /api/gaps      - Gap tracker stats (total, recurring, by-skill)")
	fmt.Println("  GET /api/reflect   - Reflect suggestions (synthesis, promote, stale)")
	fmt.Println("  GET /api/errors    - Error pattern analysis (recent errors, recurring patterns)")
	fmt.Println("  GET /api/pending-reviews - Agents with unreviewed synthesis recommendations")
	fmt.Println("  POST /api/dismiss-review - Dismiss a specific recommendation")
	fmt.Println("  GET/PUT /api/config - User configuration settings")
	fmt.Println("  GET /api/changelog - Aggregated changelog (?days=7&project=all)")
	fmt.Println("  GET /health        - Health check")
	fmt.Println("\nPress Ctrl+C to stop")

	return http.ListenAndServe(addr, mux)
}

// UsageAPIResponse is the JSON structure returned by /api/usage.
type UsageAPIResponse struct {
	Account         string  `json:"account"`                       // Account email
	AccountName     string  `json:"account_name,omitempty"`        // Account name from accounts.yaml (e.g., "personal", "work")
	FiveHour        float64 `json:"five_hour_percent"`             // 5-hour session usage %
	FiveHourReset   string  `json:"five_hour_reset,omitempty"`     // Human-readable time until 5-hour reset
	Weekly          float64 `json:"weekly_percent"`                // 7-day weekly usage %
	WeeklyReset     string  `json:"weekly_reset,omitempty"`        // Human-readable time until weekly reset
	WeeklyOpus      float64 `json:"weekly_opus_percent,omitempty"` // 7-day Opus-specific usage %
	WeeklyOpusReset string  `json:"weekly_opus_reset,omitempty"`   // Human-readable time until Opus weekly reset
	Error           string  `json:"error,omitempty"`               // Error message if any
}

// handleUsage returns Claude Max usage stats.
func handleUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	info := usage.FetchUsage()

	resp := UsageAPIResponse{}

	if info.Error != "" {
		resp.Error = info.Error
	} else {
		resp.Account = info.Email
		resp.AccountName = lookupAccountName(info.Email)
		if info.FiveHour != nil {
			resp.FiveHour = info.FiveHour.Utilization
			resp.FiveHourReset = info.FiveHour.TimeUntilReset()
		}
		if info.SevenDay != nil {
			resp.Weekly = info.SevenDay.Utilization
			resp.WeeklyReset = info.SevenDay.TimeUntilReset()
		}
		if info.SevenDayOpus != nil {
			resp.WeeklyOpus = info.SevenDayOpus.Utilization
			resp.WeeklyOpusReset = info.SevenDayOpus.TimeUntilReset()
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode usage: %v", err), http.StatusInternalServerError)
		return
	}
}

// lookupAccountName finds the account name from ~/.orch/accounts.yaml by matching email.
// Returns the account name (e.g., "personal", "work") if found, empty string otherwise.
func lookupAccountName(email string) string {
	if email == "" {
		return ""
	}

	cfg, err := account.LoadConfig()
	if err != nil {
		return ""
	}

	// Find account by matching email
	for name, acc := range cfg.Accounts {
		if acc.Email == email {
			return name
		}
	}

	return ""
}

// FocusAPIResponse is the JSON structure returned by /api/focus.
type FocusAPIResponse struct {
	Goal       string `json:"goal,omitempty"`
	BeadsID    string `json:"beads_id,omitempty"`
	SetAt      string `json:"set_at,omitempty"`
	IsDrifting bool   `json:"is_drifting"`
	HasFocus   bool   `json:"has_focus"`
}

// handleFocus returns current focus and drift status.
func handleFocus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	store, err := focus.New("")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load focus: %v", err), http.StatusInternalServerError)
		return
	}

	resp := FocusAPIResponse{}

	f := store.Get()
	if f != nil {
		resp.HasFocus = true
		resp.Goal = f.Goal
		resp.BeadsID = f.BeadsID
		resp.SetAt = f.SetAt

		// Check drift by getting active agents from current sessions
		client := opencode.NewClient(serverURL)
		sessions, _ := client.ListSessions("")

		var activeIssues []string
		for _, s := range sessions {
			if beadsID := extractBeadsIDFromTitle(s.Title); beadsID != "" {
				activeIssues = append(activeIssues, beadsID)
			}
		}

		drift := store.CheckDrift(activeIssues)
		resp.IsDrifting = drift.IsDrifting
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode focus: %v", err), http.StatusInternalServerError)
		return
	}
}

// BeadsAPIResponse is the JSON structure returned by /api/beads.
type BeadsAPIResponse struct {
	TotalIssues    int     `json:"total_issues"`
	OpenIssues     int     `json:"open_issues"`
	InProgress     int     `json:"in_progress_issues"`
	BlockedIssues  int     `json:"blocked_issues"`
	ReadyIssues    int     `json:"ready_issues"`
	ClosedIssues   int     `json:"closed_issues"`
	AvgLeadTimeHrs float64 `json:"avg_lead_time_hours,omitempty"`
	Error          string  `json:"error,omitempty"`
}

// handleBeads returns beads stats by shelling out to `bd stats --json`.
func handleBeads(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Use persistent RPC client (with auto-reconnect), fallback to CLI if unavailable
	var stats *beads.Stats
	var err error

	if beadsClient != nil {
		stats, err = beadsClient.Stats()
		if err != nil {
			// Fall through to CLI fallback on RPC error
			stats, err = beads.FallbackStats()
		}
	} else {
		stats, err = beads.FallbackStats()
	}

	if err != nil {
		resp := BeadsAPIResponse{Error: fmt.Sprintf("Failed to get bd stats: %v", err)}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp := BeadsAPIResponse{
		TotalIssues:    stats.Summary.TotalIssues,
		OpenIssues:     stats.Summary.OpenIssues,
		InProgress:     stats.Summary.InProgressIssues,
		BlockedIssues:  stats.Summary.BlockedIssues,
		ReadyIssues:    stats.Summary.ReadyIssues,
		ClosedIssues:   stats.Summary.ClosedIssues,
		AvgLeadTimeHrs: stats.Summary.AvgLeadTimeHours,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode beads: %v", err), http.StatusInternalServerError)
		return
	}
}

// ReadyIssueResponse represents a ready issue for the dashboard queue.
type ReadyIssueResponse struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Priority  int      `json:"priority"`
	IssueType string   `json:"issue_type"`
	Labels    []string `json:"labels,omitempty"`
	CreatedAt string   `json:"created_at,omitempty"`
}

// BeadsReadyAPIResponse is the JSON structure returned by /api/beads/ready.
type BeadsReadyAPIResponse struct {
	Issues []ReadyIssueResponse `json:"issues"`
	Count  int                  `json:"count"`
	Error  string               `json:"error,omitempty"`
}

// handleBeadsReady returns list of ready issues for dashboard queue visibility.
func handleBeadsReady(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Use persistent RPC client (with auto-reconnect), fallback to CLI if unavailable
	var issues []beads.Issue
	var err error

	if beadsClient != nil {
		issues, err = beadsClient.Ready(nil)
		if err != nil {
			// Fall through to CLI fallback on RPC error
			issues, err = beads.FallbackReady()
		}
	} else {
		issues, err = beads.FallbackReady()
	}

	if err != nil {
		resp := BeadsReadyAPIResponse{
			Issues: []ReadyIssueResponse{},
			Count:  0,
			Error:  fmt.Sprintf("Failed to get ready issues: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Convert beads.Issue to ReadyIssueResponse
	readyIssues := make([]ReadyIssueResponse, 0, len(issues))
	for _, issue := range issues {
		readyIssues = append(readyIssues, ReadyIssueResponse{
			ID:        issue.ID,
			Title:     issue.Title,
			Priority:  issue.Priority,
			IssueType: issue.IssueType,
			Labels:    issue.Labels,
			CreatedAt: issue.CreatedAt,
		})
	}

	resp := BeadsReadyAPIResponse{
		Issues: readyIssues,
		Count:  len(readyIssues),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode beads ready: %v", err), http.StatusInternalServerError)
		return
	}
}

// ServerPortInfo represents a port allocation for a server.
type ServerPortInfo struct {
	Service string `json:"service"`
	Port    int    `json:"port"`
	Purpose string `json:"purpose,omitempty"`
}

// ServerProjectInfo represents a project's server status.
type ServerProjectInfo struct {
	Project string           `json:"project"`
	Ports   []ServerPortInfo `json:"ports"`
	Running bool             `json:"running"`
	Session string           `json:"session,omitempty"` // tmux session name
}

// ServersAPIResponse is the JSON structure returned by /api/servers.
type ServersAPIResponse struct {
	Projects     []ServerProjectInfo `json:"projects"`
	TotalCount   int                 `json:"total_count"`
	RunningCount int                 `json:"running_count"`
	StoppedCount int                 `json:"stopped_count"`
}

// handleServers returns servers status across projects.
func handleServers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Load port registry
	reg, err := port.New("")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load port registry: %v", err), http.StatusInternalServerError)
		return
	}

	allocs := reg.List()

	// Group allocations by project
	projectMap := make(map[string][]port.Allocation)
	for _, alloc := range allocs {
		projectMap[alloc.Project] = append(projectMap[alloc.Project], alloc)
	}

	// Get list of running workers sessions
	runningSessions, _ := tmux.ListWorkersSessions()
	runningSessionSet := make(map[string]bool)
	for _, sess := range runningSessions {
		runningSessionSet[sess] = true
	}

	// Build project info list
	var projects []ServerProjectInfo
	runningCount := 0

	for projectName, projectPorts := range projectMap {
		sessionName := tmux.GetWorkersSessionName(projectName)
		running := runningSessionSet[sessionName]

		// Convert port allocations
		var ports []ServerPortInfo
		for _, p := range projectPorts {
			ports = append(ports, ServerPortInfo{
				Service: p.Service,
				Port:    p.Port,
				Purpose: p.Purpose,
			})
		}

		projects = append(projects, ServerProjectInfo{
			Project: projectName,
			Ports:   ports,
			Running: running,
			Session: sessionName,
		})

		if running {
			runningCount++
		}
	}

	resp := ServersAPIResponse{
		Projects:     projects,
		TotalCount:   len(projects),
		RunningCount: runningCount,
		StoppedCount: len(projects) - runningCount,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode servers: %v", err), http.StatusInternalServerError)
		return
	}
}

// DaemonAPIResponse is the JSON structure returned by /api/daemon.
type DaemonAPIResponse struct {
	Running       bool   `json:"running"`                   // Whether the daemon is currently running
	Status        string `json:"status,omitempty"`          // "running", "stalled", or empty if not running
	LastPoll      string `json:"last_poll,omitempty"`       // ISO 8601 timestamp of last poll
	LastPollAgo   string `json:"last_poll_ago,omitempty"`   // Human-readable time since last poll
	LastSpawn     string `json:"last_spawn,omitempty"`      // ISO 8601 timestamp of last spawn
	LastSpawnAgo  string `json:"last_spawn_ago,omitempty"`  // Human-readable time since last spawn
	ReadyCount    int    `json:"ready_count"`               // Number of issues ready to process
	CapacityMax   int    `json:"capacity_max"`              // Maximum concurrent agents
	CapacityUsed  int    `json:"capacity_used"`             // Currently active agents
	CapacityFree  int    `json:"capacity_free"`             // Available slots for spawning
	IssuesPerHour int    `json:"issues_per_hour,omitempty"` // Approximate processing rate (future)
}

// handleDaemon returns the daemon status from ~/.orch/daemon-status.json.
// If the daemon is not running (file doesn't exist), returns running: false.
func handleDaemon(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp := DaemonAPIResponse{
		Running: false,
	}

	// Try to read daemon status file
	status, err := daemon.ReadStatusFile()
	if err == nil && status != nil {
		resp.Running = true
		resp.Status = status.Status
		resp.ReadyCount = status.ReadyCount
		resp.CapacityMax = status.Capacity.Max
		resp.CapacityUsed = status.Capacity.Active
		resp.CapacityFree = status.Capacity.Available

		// Format timestamps
		if !status.LastPoll.IsZero() {
			resp.LastPoll = status.LastPoll.Format(time.RFC3339)
			resp.LastPollAgo = formatDurationAgo(time.Since(status.LastPoll))
		}
		if !status.LastSpawn.IsZero() {
			resp.LastSpawn = status.LastSpawn.Format(time.RFC3339)
			resp.LastSpawnAgo = formatDurationAgo(time.Since(status.LastSpawn))
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode daemon status: %v", err), http.StatusInternalServerError)
		return
	}
}

// formatDurationAgo formats a duration into a human-readable "X ago" string.
func formatDurationAgo(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d mins ago", mins)
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	}
	days := int(d.Hours() / 24)
	if days == 1 {
		return "1 day ago"
	}
	return fmt.Sprintf("%d days ago", days)
}

// CreateIssueRequest is the JSON request body for POST /api/issues.
type CreateIssueRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	IssueType   string   `json:"issue_type,omitempty"` // task, bug, etc.
	Priority    int      `json:"priority,omitempty"`
	Labels      []string `json:"labels,omitempty"`
	ParentID    string   `json:"parent_id,omitempty"` // Optional parent issue for follow-ups
}

// CreateIssueResponse is the JSON response for POST /api/issues.
type CreateIssueResponse struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// handleIssues handles POST /api/issues - creates a new beads issue.
// This is used by the dashboard to create follow-up issues from synthesis recommendations.
func handleIssues(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req CreateIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp := CreateIssueResponse{Success: false, Error: fmt.Sprintf("Invalid request body: %v", err)}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Validate title
	if req.Title == "" {
		resp := CreateIssueResponse{Success: false, Error: "Title is required"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Use persistent RPC client (with auto-reconnect), fallback to CLI if unavailable
	var issue *beads.Issue
	var err error

	if beadsClient != nil {
		issue, err = beadsClient.Create(&beads.CreateArgs{
			Title:       req.Title,
			Description: req.Description,
			IssueType:   req.IssueType,
			Priority:    req.Priority,
			Labels:      req.Labels,
		})
		if err != nil {
			// Fall through to CLI fallback on RPC error
			issue, err = beads.FallbackCreate(req.Title, req.Description, req.IssueType, req.Priority, req.Labels)
		}
	} else {
		issue, err = beads.FallbackCreate(req.Title, req.Description, req.IssueType, req.Priority, req.Labels)
	}

	if err != nil {
		resp := CreateIssueResponse{Success: false, Error: fmt.Sprintf("Failed to create issue: %v", err)}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp := CreateIssueResponse{
		ID:      issue.ID,
		Title:   issue.Title,
		Success: true,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// GapsAPIResponse is the JSON structure returned by /api/gaps.
type GapsAPIResponse struct {
	TotalGaps         int                    `json:"total_gaps"`
	RecurringPatterns int                    `json:"recurring_patterns"`
	BySkill           map[string]int         `json:"by_skill"`
	RecentGaps        int                    `json:"recent_gaps,omitempty"`       // Gaps in last 7 days
	Suggestions       []GapSuggestionSummary `json:"suggestions,omitempty"`       // Top recurring gap suggestions
	Error             string                 `json:"error,omitempty"`
}

// ReflectAPIResponse is the JSON structure returned by /api/reflect.
// It exposes the reflect-suggestions.json data with synthesis/promote/stale info.
type ReflectAPIResponse struct {
	Timestamp string                   `json:"timestamp"`
	Synthesis []ReflectSynthesisSummary `json:"synthesis"`
	Refine    []ReflectRefineSummary   `json:"refine,omitempty"`
	Error     string                   `json:"error,omitempty"`
}

// ReflectRefineSummary represents a kn entry that refines an existing principle.
type ReflectRefineSummary struct {
	ID         string   `json:"id"`
	Content    string   `json:"content"`
	Principle  string   `json:"principle"`
	MatchTerms []string `json:"match_terms"`
	Suggestion string   `json:"suggestion"`
}

// ReflectSynthesisSummary represents a topic with accumulated investigations.
type ReflectSynthesisSummary struct {
	Topic          string   `json:"topic"`
	Count          int      `json:"count"`
	Investigations []string `json:"investigations"`
	Suggestion     string   `json:"suggestion"`
}

// GapSuggestionSummary is a condensed version of LearningSuggestion for the API.
type GapSuggestionSummary struct {
	Query      string `json:"query"`
	Count      int    `json:"count"`
	Priority   string `json:"priority"`
	Suggestion string `json:"suggestion"`
}

// handleGaps returns gap tracker statistics from ~/.orch/gap-tracker.json.
func handleGaps(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tracker, err := spawn.LoadTracker()
	if err != nil {
		resp := GapsAPIResponse{Error: fmt.Sprintf("Failed to load gap tracker: %v", err)}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Calculate by_skill breakdown
	bySkill := tracker.GetSkillGapRates()

	// Find recurring patterns (gaps that occurred 3+ times)
	suggestions := tracker.FindRecurringGaps()

	// Count recent gaps (last 7 days)
	recentGaps := 0
	weekAgo := time.Now().Add(-7 * 24 * time.Hour)
	for _, event := range tracker.Events {
		if event.Timestamp.After(weekAgo) {
			recentGaps++
		}
	}

	// Convert suggestions to API format (top 5)
	var apiSuggestions []GapSuggestionSummary
	for i, s := range suggestions {
		if i >= 5 {
			break
		}
		apiSuggestions = append(apiSuggestions, GapSuggestionSummary{
			Query:      s.Query,
			Count:      s.Count,
			Priority:   s.Priority,
			Suggestion: s.Suggestion,
		})
	}

	resp := GapsAPIResponse{
		TotalGaps:         len(tracker.Events),
		RecurringPatterns: len(suggestions),
		BySkill:           bySkill,
		RecentGaps:        recentGaps,
		Suggestions:       apiSuggestions,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode gaps: %v", err), http.StatusInternalServerError)
		return
	}
}

// handleReflect returns reflect suggestions from ~/.orch/reflect-suggestions.json.
// This exposes synthesis/promote/stale data for kb reflect UI integration.
func handleReflect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read reflect-suggestions.json from ~/.orch/
	homeDir, err := os.UserHomeDir()
	if err != nil {
		resp := ReflectAPIResponse{Error: fmt.Sprintf("Failed to get home directory: %v", err)}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	reflectPath := filepath.Join(homeDir, ".orch", "reflect-suggestions.json")
	data, err := os.ReadFile(reflectPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty response if file doesn't exist yet
			resp := ReflectAPIResponse{
				Timestamp: "",
				Synthesis: []ReflectSynthesisSummary{},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		resp := ReflectAPIResponse{Error: fmt.Sprintf("Failed to read reflect-suggestions.json: %v", err)}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Parse the raw JSON structure
	var rawReflect struct {
		Timestamp string `json:"timestamp"`
		Synthesis []struct {
			Topic          string   `json:"topic"`
			Count          int      `json:"count"`
			Investigations []string `json:"investigations"`
			Suggestion     string   `json:"suggestion"`
		} `json:"synthesis"`
		Refine []struct {
			ID         string   `json:"id"`
			Content    string   `json:"content"`
			Principle  string   `json:"principle"`
			MatchTerms []string `json:"match_terms"`
			Suggestion string   `json:"suggestion"`
		} `json:"refine"`
	}

	if err := json.Unmarshal(data, &rawReflect); err != nil {
		resp := ReflectAPIResponse{Error: fmt.Sprintf("Failed to parse reflect-suggestions.json: %v", err)}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Convert to API response format
	var synthesis []ReflectSynthesisSummary
	for _, s := range rawReflect.Synthesis {
		synthesis = append(synthesis, ReflectSynthesisSummary{
			Topic:          s.Topic,
			Count:          s.Count,
			Investigations: s.Investigations,
			Suggestion:     s.Suggestion,
		})
	}

	// Convert refine data
	var refine []ReflectRefineSummary
	for _, r := range rawReflect.Refine {
		refine = append(refine, ReflectRefineSummary{
			ID:         r.ID,
			Content:    r.Content,
			Principle:  r.Principle,
			MatchTerms: r.MatchTerms,
			Suggestion: r.Suggestion,
		})
	}

	resp := ReflectAPIResponse{
		Timestamp: rawReflect.Timestamp,
		Synthesis: synthesis,
		Refine:    refine,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode reflect: %v", err), http.StatusInternalServerError)
		return
	}
}

// ErrorEvent represents an error event for the API.
type ErrorEvent struct {
	Type       string `json:"type"`                  // "session.error" or "agent.abandoned"
	SessionID  string `json:"session_id,omitempty"`  // Session ID if available
	BeadsID    string `json:"beads_id,omitempty"`    // Beads issue ID if available
	Timestamp  string `json:"timestamp"`             // ISO 8601 timestamp
	Message    string `json:"message,omitempty"`     // Error message or abandon reason
	Workspace  string `json:"workspace,omitempty"`   // Workspace path if available
	Skill      string `json:"skill,omitempty"`       // Skill type if known
	RecurCount int    `json:"recur_count,omitempty"` // How many times this error pattern has occurred
}

// ErrorPattern represents a recurring error pattern.
type ErrorPattern struct {
	Pattern    string   `json:"pattern"`     // Error message pattern (may be truncated/normalized)
	Count      int      `json:"count"`       // Number of occurrences
	LastSeen   string   `json:"last_seen"`   // ISO 8601 timestamp of most recent occurrence
	BeadsIDs   []string `json:"beads_ids"`   // Affected beads issues
	Suggestion string   `json:"suggestion"`  // Remediation suggestion
}

// ErrorsAPIResponse is the JSON structure returned by /api/errors.
type ErrorsAPIResponse struct {
	TotalErrors     int            `json:"total_errors"`               // Total error events
	ErrorsLast24h   int            `json:"errors_last_24h"`            // Errors in last 24 hours
	ErrorsLast7d    int            `json:"errors_last_7d"`             // Errors in last 7 days
	AbandonedCount  int            `json:"abandoned_count"`            // Total agent.abandoned events
	SessionErrors   int            `json:"session_errors"`             // Total session.error events
	RecentErrors    []ErrorEvent   `json:"recent_errors,omitempty"`    // Last 20 error events
	Patterns        []ErrorPattern `json:"patterns,omitempty"`         // Recurring error patterns
	ByType          map[string]int `json:"by_type"`                    // Breakdown by error type
	Error           string         `json:"error,omitempty"`            // Error message if any
}

// handleErrors returns error pattern analysis from ~/.orch/events.jsonl.
func handleErrors(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	logPath := events.DefaultLogPath()

	file, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty response if file doesn't exist
			resp := ErrorsAPIResponse{
				ByType: make(map[string]int),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		resp := ErrorsAPIResponse{Error: fmt.Sprintf("Failed to open events file: %v", err)}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}
	defer file.Close()

	now := time.Now()
	day24h := now.Add(-24 * time.Hour)
	days7 := now.Add(-7 * 24 * time.Hour)

	var allErrors []ErrorEvent
	byType := make(map[string]int)
	patternCounts := make(map[string]*ErrorPattern)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var event events.Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}

		// Only process error-related events
		if event.Type != events.EventTypeSessionError && event.Type != "agent.abandoned" {
			continue
		}

		ts := time.Unix(event.Timestamp, 0)
		errorEvent := ErrorEvent{
			Type:      event.Type,
			SessionID: event.SessionID,
			Timestamp: ts.Format(time.RFC3339),
		}

		// Extract details based on event type
		if event.Type == events.EventTypeSessionError {
			if msg, ok := event.Data["error"].(string); ok {
				errorEvent.Message = msg
			}
		} else if event.Type == "agent.abandoned" {
			if reason, ok := event.Data["reason"].(string); ok {
				errorEvent.Message = reason
			}
			if beadsID, ok := event.Data["beads_id"].(string); ok {
				errorEvent.BeadsID = beadsID
			}
			if workspace, ok := event.Data["workspace_path"].(string); ok {
				errorEvent.Workspace = filepath.Base(workspace)
			}
			if agentID, ok := event.Data["agent_id"].(string); ok {
				// Extract skill from agent_id pattern like "og-feat-xxx" or "og-debug-xxx"
				errorEvent.Skill = extractSkillFromAgentID(agentID)
			}
		}

		allErrors = append(allErrors, errorEvent)
		byType[event.Type]++

		// Track patterns for recurring error detection
		patternKey := normalizeErrorMessage(errorEvent.Message)
		if patternKey != "" {
			if p, exists := patternCounts[patternKey]; exists {
				p.Count++
				p.LastSeen = errorEvent.Timestamp
				if errorEvent.BeadsID != "" && !containsString(p.BeadsIDs, errorEvent.BeadsID) {
					p.BeadsIDs = append(p.BeadsIDs, errorEvent.BeadsID)
				}
			} else {
				beadsIDs := []string{}
				if errorEvent.BeadsID != "" {
					beadsIDs = append(beadsIDs, errorEvent.BeadsID)
				}
				patternCounts[patternKey] = &ErrorPattern{
					Pattern:  patternKey,
					Count:    1,
					LastSeen: errorEvent.Timestamp,
					BeadsIDs: beadsIDs,
				}
			}
		}
	}

	// Count errors by time window
	var errorsLast24h, errorsLast7d int
	for _, e := range allErrors {
		ts, _ := time.Parse(time.RFC3339, e.Timestamp)
		if ts.After(day24h) {
			errorsLast24h++
		}
		if ts.After(days7) {
			errorsLast7d++
		}
	}

	// Get recent errors (last 20, most recent first)
	recentErrors := allErrors
	if len(recentErrors) > 20 {
		recentErrors = recentErrors[len(recentErrors)-20:]
	}
	// Reverse to show most recent first
	for i, j := 0, len(recentErrors)-1; i < j; i, j = i+1, j-1 {
		recentErrors[i], recentErrors[j] = recentErrors[j], recentErrors[i]
	}

	// Convert patterns map to slice and sort by count
	var patterns []ErrorPattern
	for _, p := range patternCounts {
		if p.Count >= 2 { // Only include patterns that occurred 2+ times
			p.Suggestion = suggestRemediation(p.Pattern)
			patterns = append(patterns, *p)
		}
	}
	// Sort by count descending
	for i := 0; i < len(patterns); i++ {
		for j := i + 1; j < len(patterns); j++ {
			if patterns[j].Count > patterns[i].Count {
				patterns[i], patterns[j] = patterns[j], patterns[i]
			}
		}
	}
	// Limit to top 10 patterns
	if len(patterns) > 10 {
		patterns = patterns[:10]
	}

	resp := ErrorsAPIResponse{
		TotalErrors:    len(allErrors),
		ErrorsLast24h:  errorsLast24h,
		ErrorsLast7d:   errorsLast7d,
		AbandonedCount: byType["agent.abandoned"],
		SessionErrors:  byType[events.EventTypeSessionError],
		RecentErrors:   recentErrors,
		Patterns:       patterns,
		ByType:         byType,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode errors: %v", err), http.StatusInternalServerError)
		return
	}
}

// extractSkillFromAgentID extracts the skill type from an agent ID.
// Agent IDs have patterns like "og-feat-xxx", "og-debug-xxx", "og-inv-xxx".
func extractSkillFromAgentID(agentID string) string {
	parts := strings.Split(agentID, "-")
	if len(parts) < 2 {
		return ""
	}
	// Map short prefixes to skill names
	switch parts[1] {
	case "feat":
		return "feature-impl"
	case "debug":
		return "systematic-debugging"
	case "inv":
		return "investigation"
	case "arch":
		return "architect"
	case "work":
		return "design-session"
	default:
		return parts[1]
	}
}

// normalizeErrorMessage normalizes an error message for pattern matching.
// Removes specific identifiers to group similar errors together.
func normalizeErrorMessage(msg string) string {
	if msg == "" {
		return ""
	}
	// Truncate long messages
	if len(msg) > 100 {
		msg = msg[:100]
	}
	// Simple normalization - could be enhanced with regex for IDs, paths, etc.
	return strings.TrimSpace(msg)
}

// containsString checks if a string is in a slice.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// suggestRemediation provides a remediation suggestion based on error pattern.
func suggestRemediation(pattern string) string {
	lower := strings.ToLower(pattern)
	switch {
	case strings.Contains(lower, "stall"):
		return "Check agent for long-running operations or API timeouts"
	case strings.Contains(lower, "timeout"):
		return "Review API response times or increase timeout limits"
	case strings.Contains(lower, "capacity"):
		return "Increase daemon capacity or check for stuck agents"
	case strings.Contains(lower, "daemon"):
		return "Check daemon logs at ~/.orch/daemon.log"
	case strings.Contains(lower, "context"):
		return "Review spawn context for missing or incorrect information"
	case strings.Contains(lower, "connection"):
		return "Check network connectivity or API endpoint availability"
	default:
		return "Review agent workspace for more details"
	}
}

// PendingReviewItem represents a single synthesis recommendation pending review.
type PendingReviewItem struct {
	WorkspaceID string `json:"workspace_id"`
	BeadsID     string `json:"beads_id"`
	Index       int    `json:"index"`       // Index of the recommendation (0-based)
	Text        string `json:"text"`        // The recommendation text
	Reviewed    bool   `json:"reviewed"`    // Whether this item has been reviewed
	ActedOn     bool   `json:"acted_on"`    // Whether an issue was created
	Dismissed   bool   `json:"dismissed"`   // Whether this was dismissed
}

// PendingReviewAgent represents an agent with pending synthesis reviews.
type PendingReviewAgent struct {
	WorkspaceID          string              `json:"workspace_id"`
	WorkspacePath        string              `json:"workspace_path"`
	BeadsID              string              `json:"beads_id"`
	TLDR                 string              `json:"tldr,omitempty"`
	TotalRecommendations int                 `json:"total_recommendations"`
	UnreviewedCount      int                 `json:"unreviewed_count"`
	Items                []PendingReviewItem `json:"items"`
	IsLightTier          bool                `json:"is_light_tier,omitempty"` // True if this was a light tier spawn (no synthesis by design)
}

// PendingReviewsAPIResponse is the JSON structure returned by /api/pending-reviews.
type PendingReviewsAPIResponse struct {
	Agents           []PendingReviewAgent `json:"agents"`
	TotalAgents      int                  `json:"total_agents"`
	TotalUnreviewed  int                  `json:"total_unreviewed"`
}

// handlePendingReviews returns pending synthesis reviews.
// This includes both full-tier agents with SYNTHESIS.md and light-tier agents
// that have completed (Phase: Complete) but have no synthesis by design.
func handlePendingReviews(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Scan workspaces for SYNTHESIS.md and review state
	workspaceDir := filepath.Join(sourceDir, ".orch", "workspace")
	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		// No workspace directory is fine - just return empty response
		resp := PendingReviewsAPIResponse{
			Agents:          []PendingReviewAgent{},
			TotalAgents:     0,
			TotalUnreviewed: 0,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	var agents []PendingReviewAgent
	totalUnreviewed := 0

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirName := entry.Name()
		dirPath := filepath.Join(workspaceDir, dirName)

		// Check for SYNTHESIS.md (full-tier agents)
		synthesisPath := filepath.Join(dirPath, "SYNTHESIS.md")
		hasSynthesis := false
		if _, err := os.Stat(synthesisPath); err == nil {
			hasSynthesis = true
		}

		// Check for light-tier completion (no synthesis by design)
		isLightComplete, lightBeadsID := isLightTierComplete(dirPath)

		// Skip workspaces that are neither full-tier with synthesis nor light-tier complete
		if !hasSynthesis && !isLightComplete {
			continue
		}

		// Handle full-tier agents with synthesis
		if hasSynthesis {
			// Parse synthesis
			synthesis, err := verify.ParseSynthesis(dirPath)
			if err != nil || synthesis == nil {
				continue
			}

			// Skip if no recommendations
			if len(synthesis.NextActions) == 0 {
				continue
			}

			// Load review state
			reviewState, err := verify.LoadReviewState(dirPath)
			if err != nil {
				reviewState = &verify.ReviewState{}
			}

			// Extract beads ID from SPAWN_CONTEXT
			beadsID := extractBeadsIDFromWorkspace(dirPath)

			// Build item list
			var items []PendingReviewItem
			unreviewedCount := 0

			for i, action := range synthesis.NextActions {
				actedOn := contains(reviewState.ActedOn, i)
				dismissed := contains(reviewState.Dismissed, i)
				reviewed := actedOn || dismissed

				if !reviewed {
					unreviewedCount++
				}

				items = append(items, PendingReviewItem{
					WorkspaceID: dirName,
					BeadsID:     beadsID,
					Index:       i,
					Text:        action,
					Reviewed:    reviewed,
					ActedOn:     actedOn,
					Dismissed:   dismissed,
				})
			}

			// Only include agents with unreviewed recommendations
			if unreviewedCount > 0 {
				agents = append(agents, PendingReviewAgent{
					WorkspaceID:          dirName,
					WorkspacePath:        dirPath,
					BeadsID:              beadsID,
					TLDR:                 synthesis.TLDR,
					TotalRecommendations: len(synthesis.NextActions),
					UnreviewedCount:      unreviewedCount,
					Items:                items,
					IsLightTier:          false,
				})
				totalUnreviewed += unreviewedCount
			}
		} else if isLightComplete {
			// Handle light-tier agents (no synthesis by design)
			// Light-tier completions appear with a special indicator and no items
			// They still need orchestrator acknowledgment via beads close

			// Load review state to check if already acknowledged
			reviewState, err := verify.LoadReviewState(dirPath)
			if err != nil {
				reviewState = &verify.ReviewState{}
			}

			// Skip if already reviewed (acknowledged)
			if reviewState.LightTierAcknowledged {
				continue
			}

			// Create a single "pseudo-item" indicating the light tier completion needs review
			items := []PendingReviewItem{
				{
					WorkspaceID: dirName,
					BeadsID:     lightBeadsID,
					Index:       0,
					Text:        "Light tier agent completed - no synthesis produced (by design). Review and close via orch complete.",
					Reviewed:    false,
					ActedOn:     false,
					Dismissed:   false,
				},
			}

			agents = append(agents, PendingReviewAgent{
				WorkspaceID:          dirName,
				WorkspacePath:        dirPath,
				BeadsID:              lightBeadsID,
				TLDR:                 "Light tier completion - review agent output directly",
				TotalRecommendations: 1,
				UnreviewedCount:      1,
				Items:                items,
				IsLightTier:          true,
			})
			totalUnreviewed++
		}
	}

	resp := PendingReviewsAPIResponse{
		Agents:          agents,
		TotalAgents:     len(agents),
		TotalUnreviewed: totalUnreviewed,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode pending reviews: %v", err), http.StatusInternalServerError)
		return
	}
}

// contains checks if a slice contains a value.
func contains(slice []int, val int) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

// isLightTierWorkspace checks if a workspace is a light tier spawn.
// Light tier workspaces have a .tier file containing "light".
func isLightTierWorkspace(workspacePath string) bool {
	tierPath := filepath.Join(workspacePath, ".tier")
	data, err := os.ReadFile(tierPath)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(data)) == "light"
}

// isLightTierComplete checks if a light tier workspace has Phase: Complete in beads comments.
// Returns true if the workspace is light tier AND has Phase: Complete.
func isLightTierComplete(workspacePath string) (isComplete bool, beadsID string) {
	if !isLightTierWorkspace(workspacePath) {
		return false, ""
	}

	// Extract beads ID from SPAWN_CONTEXT.md
	beadsID = extractBeadsIDFromWorkspace(workspacePath)
	if beadsID == "" {
		return false, ""
	}

	// Get comments for this beads ID
	comments, err := verify.GetComments(beadsID)
	if err != nil {
		return false, beadsID
	}

	// Check for Phase: Complete
	phaseStatus := verify.ParsePhaseFromComments(comments)
	return phaseStatus.Found && strings.EqualFold(phaseStatus.Phase, "Complete"), beadsID
}

// DismissReviewRequest is the request body for POST /api/dismiss-review.
type DismissReviewRequest struct {
	WorkspaceID string `json:"workspace_id"`
	Index       int    `json:"index"` // Index of the recommendation to dismiss
}

// DismissReviewResponse is the response for POST /api/dismiss-review.
type DismissReviewResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message,omitempty"`
	Error    string `json:"error,omitempty"`
}

// handleDismissReview dismisses a synthesis recommendation.
func handleDismissReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DismissReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DismissReviewResponse{
			Success: false,
			Error:   fmt.Sprintf("Invalid request body: %v", err),
		})
		return
	}

	if req.WorkspaceID == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DismissReviewResponse{
			Success: false,
			Error:   "workspace_id is required",
		})
		return
	}

	// Build workspace path
	workspacePath := filepath.Join(sourceDir, ".orch", "workspace", req.WorkspaceID)

	// Check for light-tier workspace - these don't have SYNTHESIS.md by design.
	// Light-tier dismissals set LightTierAcknowledged instead of tracking individual recommendations.
	if isLightTierWorkspace(workspacePath) {
		reviewState, err := verify.LoadReviewState(workspacePath)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(DismissReviewResponse{
				Success: false,
				Error:   fmt.Sprintf("Failed to load review state: %v", err),
			})
			return
		}

		// Check if already acknowledged
		if reviewState.LightTierAcknowledged {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(DismissReviewResponse{
				Success: true,
				Message: "Already acknowledged",
			})
			return
		}

		// Mark as acknowledged
		reviewState.LightTierAcknowledged = true
		reviewState.WorkspaceID = req.WorkspaceID
		reviewState.BeadsID = extractBeadsIDFromWorkspace(workspacePath)
		if reviewState.ReviewedAt.IsZero() {
			reviewState.ReviewedAt = time.Now()
		}

		if err := verify.SaveReviewState(workspacePath, reviewState); err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(DismissReviewResponse{
				Success: false,
				Error:   fmt.Sprintf("Failed to save review state: %v", err),
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DismissReviewResponse{
			Success: true,
			Message: "Light tier completion acknowledged",
		})
		return
	}

	// Full-tier path: Load existing review state
	reviewState, err := verify.LoadReviewState(workspacePath)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DismissReviewResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to load review state: %v", err),
		})
		return
	}

	// Parse synthesis to get total recommendations
	synthesis, err := verify.ParseSynthesis(workspacePath)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DismissReviewResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to parse synthesis: %v", err),
		})
		return
	}

	// Validate index
	if req.Index < 0 || req.Index >= len(synthesis.NextActions) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DismissReviewResponse{
			Success: false,
			Error:   fmt.Sprintf("Invalid index %d (total recommendations: %d)", req.Index, len(synthesis.NextActions)),
		})
		return
	}

	// Check if already reviewed
	if contains(reviewState.ActedOn, req.Index) || contains(reviewState.Dismissed, req.Index) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DismissReviewResponse{
			Success: true,
			Message: "Already reviewed",
		})
		return
	}

	// Add to dismissed
	reviewState.Dismissed = append(reviewState.Dismissed, req.Index)
	reviewState.TotalRecommendations = len(synthesis.NextActions)
	reviewState.WorkspaceID = req.WorkspaceID
	if reviewState.ReviewedAt.IsZero() {
		reviewState.ReviewedAt = time.Now()
	}

	// Extract beads ID if not set
	if reviewState.BeadsID == "" {
		reviewState.BeadsID = extractBeadsIDFromWorkspace(workspacePath)
	}

	// Save updated state
	if err := verify.SaveReviewState(workspacePath, reviewState); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DismissReviewResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to save review state: %v", err),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(DismissReviewResponse{
		Success: true,
		Message: fmt.Sprintf("Dismissed recommendation %d", req.Index),
	})
}

// ConfigAPIResponse is the JSON structure returned by GET /api/config.
type ConfigAPIResponse struct {
	Backend              string `json:"backend"`
	AutoExportTranscript bool   `json:"auto_export_transcript"`
	NotificationsEnabled bool   `json:"notifications_enabled"`
	ConfigPath           string `json:"config_path,omitempty"` // Path to config file for reference
}

// ConfigUpdateRequest is the JSON structure for PUT /api/config.
type ConfigUpdateRequest struct {
	Backend              *string `json:"backend,omitempty"`
	AutoExportTranscript *bool   `json:"auto_export_transcript,omitempty"`
	NotificationsEnabled *bool   `json:"notifications_enabled,omitempty"`
}

// handleConfig handles GET and PUT requests for user configuration.
// GET returns current config from ~/.orch/config.yaml
// PUT updates specified fields in the config
func handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleConfigGet(w, r)
	case http.MethodPut:
		handleConfigPut(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleConfigGet returns the current user configuration.
func handleConfigGet(w http.ResponseWriter, r *http.Request) {
	cfg, err := userconfig.Load()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	resp := ConfigAPIResponse{
		Backend:              cfg.Backend,
		AutoExportTranscript: cfg.AutoExportTranscript,
		NotificationsEnabled: cfg.NotificationsEnabled(),
		ConfigPath:           userconfig.ConfigPath(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode config: %v", err), http.StatusInternalServerError)
		return
	}
}

// handleConfigPut updates the user configuration with the provided values.
func handleConfigPut(w http.ResponseWriter, r *http.Request) {
	var req ConfigUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Load existing config
	cfg, err := userconfig.Load()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	// Update only the fields that were provided
	if req.Backend != nil {
		cfg.Backend = *req.Backend
	}
	if req.AutoExportTranscript != nil {
		cfg.AutoExportTranscript = *req.AutoExportTranscript
	}
	if req.NotificationsEnabled != nil {
		cfg.Notifications.Enabled = req.NotificationsEnabled
	}

	// Save the updated config
	if err := userconfig.Save(cfg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save config: %v", err), http.StatusInternalServerError)
		return
	}

	// Return the updated config
	resp := ConfigAPIResponse{
		Backend:              cfg.Backend,
		AutoExportTranscript: cfg.AutoExportTranscript,
		NotificationsEnabled: cfg.NotificationsEnabled(),
		ConfigPath:           userconfig.ConfigPath(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode config: %v", err), http.StatusInternalServerError)
		return
	}
}

// handleChangelog returns aggregated changelog data across ecosystem repos.
// Query parameters:
//   - days: Number of days to include (default: 7)
//   - project: Project to filter (default: "all" for all repos)
func handleChangelog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	daysStr := r.URL.Query().Get("days")
	project := r.URL.Query().Get("project")

	// Default values
	days := 7
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			days = d
		}
	}
	if project == "" {
		project = "all"
	}

	// Get changelog data using the reusable function from changelog.go
	result, err := GetChangelog(days, project)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get changelog: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode changelog: %v", err), http.StatusInternalServerError)
		return
	}
}
