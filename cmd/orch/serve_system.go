package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/account"
	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/dylan-conlin/orch-go/pkg/daemonconfig"
	"github.com/dylan-conlin/orch-go/pkg/focus"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/port"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/dylan-conlin/orch-go/pkg/userconfig"
)

// UsageAPIResponse is the JSON structure returned by /api/usage.
// Note: Percentage fields use *float64 to distinguish between 0% (valid) and unavailable (null).
// When Anthropic API returns null for usage data, these fields will be null in JSON response.
type UsageAPIResponse struct {
	Account         string   `json:"account"`                       // Account email
	AccountName     string   `json:"account_name,omitempty"`        // Account name from accounts.yaml (e.g., "personal", "work")
	FiveHour        *float64 `json:"five_hour_percent"`             // 5-hour session usage % (null if unavailable)
	FiveHourReset   string   `json:"five_hour_reset,omitempty"`     // Human-readable time until 5-hour reset
	Weekly          *float64 `json:"weekly_percent"`                // 7-day weekly usage % (null if unavailable)
	WeeklyReset     string   `json:"weekly_reset,omitempty"`        // Human-readable time until weekly reset
	WeeklyOpus      *float64 `json:"weekly_opus_percent,omitempty"` // 7-day Opus-specific usage % (null if unavailable)
	WeeklyOpusReset string   `json:"weekly_opus_reset,omitempty"`   // Human-readable time until Opus weekly reset
	Error           string   `json:"error,omitempty"`               // Error message if any
}

// handleUsage returns Claude Max usage stats.
func handleUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	capacity, _ := account.GetCurrentCapacity()

	resp := UsageAPIResponse{}

	if capacity == nil || capacity.Error != "" {
		errMsg := "failed to fetch capacity"
		if capacity != nil {
			errMsg = capacity.Error
		}
		resp.Error = errMsg
	} else {
		resp.Account = capacity.Email
		resp.AccountName = lookupAccountName(capacity.Email)
		resp.FiveHour = &capacity.FiveHourUsed
		fiveHourReset := timeUntilReset(capacity.FiveHourResets)
		if fiveHourReset != "" {
			resp.FiveHourReset = fiveHourReset
		}
		resp.Weekly = &capacity.SevenDayUsed
		weeklyReset := timeUntilReset(capacity.SevenDayResets)
		if weeklyReset != "" {
			resp.WeeklyReset = weeklyReset
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

// ServiceInfoAPIItem represents a single service in the API response.
type ServiceInfoAPIItem struct {
	Name         string `json:"name"`
	PID          int    `json:"pid"`
	Status       string `json:"status"`        // "running", "stopped", etc.
	RestartCount int    `json:"restart_count"` // Number of restarts since monitor started
	Uptime       string `json:"uptime"`        // Human-readable uptime (e.g., "2h 15m")
}

// ServicesAPIResponse is the JSON structure returned by /api/services.
type ServicesAPIResponse struct {
	Project      string               `json:"project"`       // Project name
	Services     []ServiceInfoAPIItem `json:"services"`      // List of services
	TotalCount   int                  `json:"total_count"`   // Total number of services
	RunningCount int                  `json:"running_count"` // Number of running services
	StoppedCount int                  `json:"stopped_count"` // Number of stopped services
}

// handleServices returns service health from the overmind monitor.
func handleServices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	serviceMonitorMu.RLock()
	monitor := serviceMonitor
	serviceMonitorMu.RUnlock()

	if monitor == nil {
		// Service monitor not initialized (shouldn't happen but handle gracefully)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ServicesAPIResponse{
			Project:      "unknown",
			Services:     []ServiceInfoAPIItem{},
			TotalCount:   0,
			RunningCount: 0,
			StoppedCount: 0,
		})
		return
	}

	// Get current service state from monitor
	states := monitor.GetState()

	// Convert to API format
	var services []ServiceInfoAPIItem
	runningCount := 0
	stoppedCount := 0

	for _, state := range states {
		// Calculate uptime (time since last seen)
		uptime := formatDuration(time.Since(state.LastSeen))

		services = append(services, ServiceInfoAPIItem{
			Name:         state.Name,
			PID:          state.PID,
			Status:       state.Status,
			RestartCount: state.RestartCount,
			Uptime:       uptime,
		})

		if state.Status == "running" && state.PID != 0 {
			runningCount++
		} else {
			stoppedCount++
		}
	}

	// Extract project name from source directory
	projectName := filepath.Base(sourceDir)

	resp := ServicesAPIResponse{
		Project:      projectName,
		Services:     services,
		TotalCount:   len(services),
		RunningCount: runningCount,
		StoppedCount: stoppedCount,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode services: %v", err), http.StatusInternalServerError)
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

	// Utilization metrics - tracks daemon vs manual spawn ratio to surface triage discipline
	Utilization *DaemonUtilizationMetrics `json:"utilization,omitempty"`

	// SpawnFailures holds spawn failure tracking information for health card alerting
	SpawnFailures *DaemonSpawnFailures `json:"spawn_failures,omitempty"`

	// Verification holds verification tracking information for gate visibility
	Verification *DaemonVerificationStatus `json:"verification,omitempty"`
}

// DaemonUtilizationMetrics tracks the ratio of daemon-spawned vs manual-spawned agents.
// This surfaces where triage discipline is slipping (manual spawns bypassing daemon workflow).
type DaemonUtilizationMetrics struct {
	TotalSpawns     int     `json:"total_spawns"`      // Total spawns in analysis window
	DaemonSpawns    int     `json:"daemon_spawns"`     // Spawns triggered by daemon
	ManualSpawns    int     `json:"manual_spawns"`     // Spawns triggered manually (not by daemon)
	DaemonSpawnRate float64 `json:"daemon_spawn_rate"` // % of spawns from daemon (higher is better)
	TriageBypassed  int     `json:"triage_bypassed"`   // Count of spawns that bypassed triage
	TriageSlipRate  float64 `json:"triage_slip_rate"`  // % of spawns that bypassed triage (lower is better)
	AutoCompletions int     `json:"auto_completions"`  // Count of daemon auto-completions
	AnalysisPeriod  string  `json:"analysis_period"`   // Time window description (e.g., "Last 7 days")
	DaysAnalyzed    int     `json:"days_analyzed"`     // Number of days in analysis window
}

// DaemonSpawnFailures tracks spawn failures to surface them in health metrics.
// This prevents silent failure when UpdateBeadsStatus or spawn persistently fails.
type DaemonSpawnFailures struct {
	ConsecutiveFailures int    `json:"consecutive_failures"`          // Failures since last successful spawn
	TotalFailures       int    `json:"total_failures"`                // Total failures (lifetime)
	LastFailure         string `json:"last_failure,omitempty"`        // ISO 8601 timestamp
	LastFailureAgo      string `json:"last_failure_ago,omitempty"`    // Human-readable time since last failure
	LastFailureReason   string `json:"last_failure_reason,omitempty"` // Error message from last failure
}

// DaemonVerificationStatus tracks verification gate state for dashboard visibility.
type DaemonVerificationStatus struct {
	IsPaused                     bool   `json:"is_paused"`                       // Whether daemon is paused due to verification threshold
	CompletionsSinceVerification int    `json:"completions_since_verification"`  // Count of auto-completions since last verification
	Threshold                    int    `json:"threshold"`                       // Maximum auto-completions before pausing
	RemainingBeforePause         int    `json:"remaining_before_pause"`          // Completions allowed before pause
	LastVerification             string `json:"last_verification,omitempty"`     // ISO 8601 timestamp of last verification
	LastVerificationAgo          string `json:"last_verification_ago,omitempty"` // Human-readable time since last verification
}

// handleDaemon returns the daemon status from ~/.orch/daemon-status.json.
// If the daemon is not running (file doesn't exist), returns running: false.
// Also includes utilization metrics computed from events.jsonl.
func handleDaemon(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse optional days parameter (default: 7)
	days := 7
	if daysStr := r.URL.Query().Get("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= 90 {
			days = d
		}
	}

	resp := DaemonAPIResponse{
		Running: false,
	}

	// Try to read daemon status file (validates PID liveness to detect stale files)
	status, err := daemon.ReadValidatedStatusFile()
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

		// Populate spawn failures for health card visibility
		if status.SpawnFailures != nil {
			resp.SpawnFailures = &DaemonSpawnFailures{
				ConsecutiveFailures: status.SpawnFailures.ConsecutiveFailures,
				TotalFailures:       status.SpawnFailures.TotalFailures,
				LastFailureReason:   status.SpawnFailures.LastFailureReason,
			}
			if !status.SpawnFailures.LastFailure.IsZero() {
				resp.SpawnFailures.LastFailure = status.SpawnFailures.LastFailure.Format(time.RFC3339)
				resp.SpawnFailures.LastFailureAgo = formatDurationAgo(time.Since(status.SpawnFailures.LastFailure))
			}
		}

		// Populate verification status for gate visibility
		if status.Verification != nil {
			resp.Verification = &DaemonVerificationStatus{
				IsPaused:                     status.Verification.IsPaused,
				CompletionsSinceVerification: status.Verification.CompletionsSinceVerification,
				Threshold:                    status.Verification.Threshold,
				RemainingBeforePause:         status.Verification.RemainingBeforePause,
			}
			if !status.Verification.LastVerification.IsZero() {
				resp.Verification.LastVerification = status.Verification.LastVerification.Format(time.RFC3339)
				resp.Verification.LastVerificationAgo = formatDurationAgo(time.Since(status.Verification.LastVerification))
			}
		}
	}

	// Get utilization metrics from events.jsonl
	utilization, err := daemon.GetUtilizationMetrics(days)
	if err == nil && utilization != nil {
		resp.Utilization = &DaemonUtilizationMetrics{
			TotalSpawns:     utilization.TotalSpawns,
			DaemonSpawns:    utilization.DaemonSpawns,
			ManualSpawns:    utilization.ManualSpawns,
			DaemonSpawnRate: utilization.DaemonSpawnRate,
			TriageBypassed:  utilization.TriageBypassed,
			TriageSlipRate:  utilization.TriageSlipRate,
			AutoCompletions: utilization.AutoCompletions,
			AnalysisPeriod:  utilization.AnalysisPeriod,
			DaysAnalyzed:    utilization.DaysAnalyzed,
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

// DaemonConfigAPIResponse is the JSON structure returned by GET /api/config/daemon.
type DaemonConfigAPIResponse struct {
	PollInterval     int      `json:"poll_interval"`     // Seconds between daemon polls
	MaxAgents        int      `json:"max_agents"`        // Maximum concurrent agents
	Label            string   `json:"label"`             // Beads label filter (e.g., "triage:ready")
	Verbose          bool     `json:"verbose"`           // Verbose logging enabled
	ReflectIssues    bool     `json:"reflect_issues"`    // Create issues from kb reflect
	ReflectOpen      bool     `json:"reflect_open"`      // Create issues for open investigation actions
	WorkingDirectory string   `json:"working_directory"` // Daemon's working directory
	Path             []string `json:"path"`              // PATH directories for daemon environment
}

// DaemonConfigUpdateRequest is the JSON structure for PUT /api/config/daemon.
type DaemonConfigUpdateRequest struct {
	PollInterval     *int    `json:"poll_interval,omitempty"`
	MaxAgents        *int    `json:"max_agents,omitempty"`
	Label            *string `json:"label,omitempty"`
	Verbose          *bool   `json:"verbose,omitempty"`
	ReflectIssues    *bool   `json:"reflect_issues,omitempty"`
	ReflectOpen      *bool   `json:"reflect_open,omitempty"`
	WorkingDirectory *string `json:"working_directory,omitempty"`
}

// handleDaemonConfig handles GET and PUT requests for daemon configuration.
// GET returns current daemon config from ~/.orch/config.yaml
// PUT updates daemon config, writes to config.yaml, regenerates plist, and kicks daemon
func handleDaemonConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleDaemonConfigGet(w, r)
	case http.MethodPut:
		handleDaemonConfigPut(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleDaemonConfigGet returns the current daemon configuration.
func handleDaemonConfigGet(w http.ResponseWriter, r *http.Request) {
	cfg, err := userconfig.Load()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	resp := DaemonConfigAPIResponse{
		PollInterval:     cfg.DaemonPollInterval(),
		MaxAgents:        cfg.DaemonMaxAgents(),
		Label:            cfg.DaemonLabel(),
		Verbose:          cfg.DaemonVerbose(),
		ReflectIssues:    cfg.DaemonReflectIssues(),
		ReflectOpen:      cfg.DaemonReflectOpen(),
		WorkingDirectory: cfg.DaemonWorkingDirectory(),
		Path:             cfg.DaemonPath(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode daemon config: %v", err), http.StatusInternalServerError)
		return
	}
}

// handleDaemonConfigPut updates the daemon configuration.
// After saving config, it regenerates the plist and kicks the daemon.
func handleDaemonConfigPut(w http.ResponseWriter, r *http.Request) {
	var req DaemonConfigUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate inputs
	if req.PollInterval != nil && *req.PollInterval < 10 {
		http.Error(w, "poll_interval must be at least 10 seconds", http.StatusBadRequest)
		return
	}
	if req.MaxAgents != nil && (*req.MaxAgents < 1 || *req.MaxAgents > 10) {
		http.Error(w, "max_agents must be between 1 and 10", http.StatusBadRequest)
		return
	}
	if req.Label != nil && *req.Label == "" {
		http.Error(w, "label cannot be empty", http.StatusBadRequest)
		return
	}

	// Load existing config
	cfg, err := userconfig.Load()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	// Update only the fields that were provided
	if req.PollInterval != nil {
		cfg.Daemon.PollInterval = req.PollInterval
	}
	if req.MaxAgents != nil {
		cfg.Daemon.MaxAgents = req.MaxAgents
	}
	if req.Label != nil {
		cfg.Daemon.Label = *req.Label
	}
	if req.Verbose != nil {
		cfg.Daemon.Verbose = req.Verbose
	}
	if req.ReflectIssues != nil {
		cfg.Daemon.ReflectIssues = req.ReflectIssues
	}
	if req.ReflectOpen != nil {
		cfg.Daemon.ReflectOpen = req.ReflectOpen
	}
	if req.WorkingDirectory != nil {
		cfg.Daemon.WorkingDirectory = *req.WorkingDirectory
	}

	// Save the updated config
	if err := userconfig.Save(cfg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save config: %v", err), http.StatusInternalServerError)
		return
	}

	// Regenerate plist and kick daemon
	regenerateErr := regeneratePlistAndKickDaemon()

	// Return the updated config along with regeneration status
	resp := struct {
		DaemonConfigAPIResponse
		PlistRegenerated bool   `json:"plist_regenerated"`
		DaemonKicked     bool   `json:"daemon_kicked"`
		RegenerateError  string `json:"regenerate_error,omitempty"`
	}{
		DaemonConfigAPIResponse: DaemonConfigAPIResponse{
			PollInterval:     cfg.DaemonPollInterval(),
			MaxAgents:        cfg.DaemonMaxAgents(),
			Label:            cfg.DaemonLabel(),
			Verbose:          cfg.DaemonVerbose(),
			ReflectIssues:    cfg.DaemonReflectIssues(),
			ReflectOpen:      cfg.DaemonReflectOpen(),
			WorkingDirectory: cfg.DaemonWorkingDirectory(),
			Path:             cfg.DaemonPath(),
		},
		PlistRegenerated: regenerateErr == nil,
		DaemonKicked:     regenerateErr == nil,
	}

	if regenerateErr != nil {
		resp.RegenerateError = regenerateErr.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode daemon config: %v", err), http.StatusInternalServerError)
		return
	}
}

// DriftStatusAPIResponse is the JSON structure returned by GET /api/config/drift.
type DriftStatusAPIResponse struct {
	InSync       bool   `json:"in_sync"`                 // Whether plist matches config
	PlistPath    string `json:"plist_path"`              // Path to the plist file
	PlistExists  bool   `json:"plist_exists"`            // Whether plist file exists
	ConfigPath   string `json:"config_path"`             // Path to config.yaml
	DriftDetails string `json:"drift_details,omitempty"` // Human-readable drift description
}

// handleConfigDrift checks if the plist file matches the current config.
func handleConfigDrift(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	plistPath := daemonconfig.GetPlistPath()
	configPath := userconfig.ConfigPath()

	resp := DriftStatusAPIResponse{
		PlistPath:  plistPath,
		ConfigPath: configPath,
	}

	// Check if plist exists
	existingContent, err := os.ReadFile(plistPath)
	if err != nil {
		if os.IsNotExist(err) {
			resp.InSync = false
			resp.PlistExists = false
			resp.DriftDetails = "plist file does not exist"
		} else {
			http.Error(w, fmt.Sprintf("Failed to read plist: %v", err), http.StatusInternalServerError)
			return
		}
	} else {
		resp.PlistExists = true

		// Generate expected plist content from config
		expectedContent, err := generatePlistContent()
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to generate expected plist: %v", err), http.StatusInternalServerError)
			return
		}

		// Compare contents
		if bytes.Equal(existingContent, expectedContent) {
			resp.InSync = true
		} else {
			resp.InSync = false
			resp.DriftDetails = "plist content differs from config"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode drift status: %v", err), http.StatusInternalServerError)
		return
	}
}

// RegenerateAPIResponse is the JSON structure returned by POST /api/config/regenerate.
type RegenerateAPIResponse struct {
	Success      bool   `json:"success"`
	PlistPath    string `json:"plist_path"`
	DaemonKicked bool   `json:"daemon_kicked"`
	Error        string `json:"error,omitempty"`
}

// handleConfigRegenerate regenerates the plist from config and kicks the daemon.
func handleConfigRegenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	plistPath := daemonconfig.GetPlistPath()

	err := regeneratePlistAndKickDaemon()
	resp := RegenerateAPIResponse{
		Success:      err == nil,
		PlistPath:    plistPath,
		DaemonKicked: err == nil,
	}

	if err != nil {
		resp.Error = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode regenerate response: %v", err), http.StatusInternalServerError)
		return
	}
}

// regeneratePlistAndKickDaemon generates plist from config and restarts the daemon.
// This is shared by PUT /api/config/daemon and POST /api/config/regenerate.
func regeneratePlistAndKickDaemon() error {
	plistContent, err := generatePlistContent()
	if err != nil {
		return fmt.Errorf("failed to generate plist: %w", err)
	}

	plistPath := daemonconfig.GetPlistPath()

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(plistPath), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Check if file exists and compare
	existingContent, readErr := os.ReadFile(plistPath)
	if readErr == nil && bytes.Equal(existingContent, plistContent) {
		// Plist is already up to date, still kick daemon in case it's stale
	} else {
		// Write the new plist
		if err := os.WriteFile(plistPath, plistContent, 0644); err != nil {
			return fmt.Errorf("failed to write plist: %w", err)
		}
	}

	// Kick the daemon to reload the plist
	cmd := exec.Command("launchctl", "kickstart", "-k", fmt.Sprintf("gui/%d/com.orch.daemon", os.Getuid()))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to kick daemon: %w", err)
	}

	return nil
}

// generatePlistContent generates plist content from config using the consolidated daemonconfig package.
func generatePlistContent() ([]byte, error) {
	cfg, err := userconfig.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return daemonconfig.GeneratePlist(cfg)
}

// FileAPIResponse is the JSON structure returned by /api/file.
type FileAPIResponse struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Size    int64  `json:"size"`
	Error   string `json:"error,omitempty"`
}

// ScreenshotsAPIResponse is the JSON structure returned by /api/screenshots.
type ScreenshotsAPIResponse struct {
	AgentID     string   `json:"agent_id"`
	Screenshots []string `json:"screenshots"` // Filenames only (not full paths)
	Error       string   `json:"error,omitempty"`
}

// handleScreenshots returns a list of screenshot filenames for a given agent.
// Query parameters:
//   - agent_id: Agent ID to fetch screenshots for (required)
//   - project_dir: Project directory (required)
//
// Security: Only lists files in {project_dir}/.orch/workspace/{agent_id}/screenshots/
// Returns filenames only (not full paths) to prevent path traversal.
func handleScreenshots(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	agentID := r.URL.Query().Get("agent_id")
	projectDir := r.URL.Query().Get("project_dir")

	if agentID == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ScreenshotsAPIResponse{
			Error: "agent_id query parameter is required",
		})
		return
	}

	if projectDir == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ScreenshotsAPIResponse{
			AgentID: agentID,
			Error:   "project_dir query parameter is required",
		})
		return
	}

	// Construct workspace path: {project_dir}/.orch/workspace/{agent_id}
	workspacePath := filepath.Join(projectDir, ".orch", "workspace", agentID)

	// Build screenshots directory path
	screenshotsDir := filepath.Join(workspacePath, "screenshots")

	// Check if screenshots directory exists
	info, err := os.Stat(screenshotsDir)
	if err != nil || !info.IsDir() {
		// Directory doesn't exist - return empty list (not an error)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ScreenshotsAPIResponse{
			AgentID:     agentID,
			Screenshots: []string{},
		})
		return
	}

	// Read directory contents
	entries, err := os.ReadDir(screenshotsDir)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ScreenshotsAPIResponse{
			AgentID: agentID,
			Error:   fmt.Sprintf("failed to read screenshots directory: %v", err),
		})
		return
	}

	// Filter to image files only and collect filenames
	screenshots := make([]string, 0)
	imageExtensions := map[string]bool{
		".png":  true,
		".jpg":  true,
		".jpeg": true,
		".gif":  true,
		".webp": true,
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if imageExtensions[ext] {
			screenshots = append(screenshots, entry.Name())
		}
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ScreenshotsAPIResponse{
		AgentID:     agentID,
		Screenshots: screenshots,
	})
}

// handleFile returns the contents of a file at the specified path.
// Query parameters:
//   - path: Absolute path to the file (required)
//
// Security: Only allows reading files in allowed directories (.kb/, .orch/workspace/).
// This prevents arbitrary file reads while enabling investigation and workspace file viewing.
func handleFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(FileAPIResponse{
			Error: "path query parameter is required",
		})
		return
	}

	// Security check: only allow files in allowed directories
	// This prevents arbitrary file reads while enabling investigation file viewing
	allowedPatterns := []string{
		"/.kb/",             // Knowledge base (investigations, decisions)
		"/.orch/workspace/", // Agent workspaces (SYNTHESIS.md, SPAWN_CONTEXT.md)
		"/.orch/templates/", // Templates
	}

	allowed := false
	for _, pattern := range allowedPatterns {
		if strings.Contains(filePath, pattern) {
			allowed = true
			break
		}
	}

	if !allowed {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(FileAPIResponse{
			Path:  filePath,
			Error: "access denied: path must be in .kb/ or .orch/workspace/",
		})
		return
	}

	// Clean the path to prevent traversal attacks
	cleanPath := filepath.Clean(filePath)
	if cleanPath != filePath {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(FileAPIResponse{
			Path:  filePath,
			Error: "invalid path",
		})
		return
	}

	// Check if file exists
	info, err := os.Stat(cleanPath)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(FileAPIResponse{
			Path:  cleanPath,
			Error: fmt.Sprintf("file not found: %v", err),
		})
		return
	}

	// Don't allow reading directories
	if info.IsDir() {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(FileAPIResponse{
			Path:  cleanPath,
			Error: "cannot read directory",
		})
		return
	}

	// Read file content
	content, err := os.ReadFile(cleanPath)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(FileAPIResponse{
			Path:  cleanPath,
			Error: fmt.Sprintf("failed to read file: %v", err),
		})
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(FileAPIResponse{
		Path:    cleanPath,
		Content: string(content),
		Size:    info.Size(),
	})
}
