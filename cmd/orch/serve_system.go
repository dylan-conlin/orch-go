package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/account"
	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/dylan-conlin/orch-go/pkg/focus"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/port"
	"github.com/dylan-conlin/orch-go/pkg/session"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/dylan-conlin/orch-go/pkg/usage"
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

	info := usage.FetchUsage()

	resp := UsageAPIResponse{}

	if info.Error != "" {
		resp.Error = info.Error
	} else {
		resp.Account = info.Email
		resp.AccountName = lookupAccountName(info.Email)
		if info.FiveHour != nil {
			resp.FiveHour = &info.FiveHour.Utilization
			resp.FiveHourReset = info.FiveHour.TimeUntilReset()
		}
		// else: FiveHour remains nil (JSON: null) indicating data unavailable
		if info.SevenDay != nil {
			resp.Weekly = &info.SevenDay.Utilization
			resp.WeeklyReset = info.SevenDay.TimeUntilReset()
		}
		// else: Weekly remains nil (JSON: null) indicating data unavailable
		if info.SevenDayOpus != nil {
			resp.WeeklyOpus = &info.SevenDayOpus.Utilization
			resp.WeeklyOpusReset = info.SevenDayOpus.TimeUntilReset()
		}
		// else: WeeklyOpus remains nil (JSON: null) indicating data unavailable
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

// OrchestratorSessionsAPIResponse is the JSON structure returned by /api/orchestrator-sessions.
type OrchestratorSessionsAPIResponse struct {
	Sessions []OrchestratorSessionAPIItem `json:"sessions"`
	Count    int                          `json:"count"`
}

// OrchestratorSessionAPIItem represents an orchestrator session in the API response.
type OrchestratorSessionAPIItem struct {
	WorkspaceName   string `json:"workspace_name"`
	SessionID       string `json:"session_id,omitempty"`
	Goal            string `json:"goal"`
	Duration        string `json:"duration"`
	DurationSeconds int64  `json:"duration_seconds"` // For sorting/calculations
	Project         string `json:"project"`
	ProjectDir      string `json:"project_dir"`
	Status          string `json:"status"`
	SpawnTime       string `json:"spawn_time"`        // ISO 8601
	ChildAgentCount int    `json:"child_agent_count"` // Number of active agents in same project
}

// handleOrchestratorSessions returns active orchestrator sessions from the registry.
// Query parameters:
//   - project: Filter by project name (optional)
func handleOrchestratorSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	project := r.URL.Query().Get("project")

	registry := session.NewRegistry("")
	sessions, err := registry.ListActive()
	if err != nil {
		// Return empty list if registry doesn't exist yet
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(OrchestratorSessionsAPIResponse{
			Sessions: []OrchestratorSessionAPIItem{},
			Count:    0,
		})
		return
	}

	// Get active agents to count children per project
	client := opencode.NewClient(serverURL)
	opencodeSessions, _ := client.ListSessions("")

	// Count active agents per project
	projectAgentCounts := make(map[string]int)
	maxIdleTime := 30 * time.Minute
	now := time.Now()
	for _, s := range opencodeSessions {
		updatedAt := time.Unix(s.Time.Updated/1000, 0)
		if now.Sub(updatedAt) <= maxIdleTime {
			// Extract project from title (e.g., "og-feat-xxx" -> "orch-go")
			if beadsID := extractBeadsIDFromTitle(s.Title); beadsID != "" {
				proj := extractProjectFromBeadsID(beadsID)
				if proj != "" {
					projectAgentCounts[proj]++
				}
			}
		}
	}

	var result []OrchestratorSessionAPIItem

	for _, s := range sessions {
		projectName := filepath.Base(s.ProjectDir)

		// Filter by project if specified
		if project != "" && projectName != project {
			continue
		}

		duration := now.Sub(s.SpawnTime)

		result = append(result, OrchestratorSessionAPIItem{
			WorkspaceName:   s.WorkspaceName,
			SessionID:       s.SessionID,
			Goal:            s.Goal,
			Duration:        formatDuration(duration),
			DurationSeconds: int64(duration.Seconds()),
			Project:         projectName,
			ProjectDir:      s.ProjectDir,
			Status:          s.Status,
			SpawnTime:       s.SpawnTime.Format(time.RFC3339),
			ChildAgentCount: projectAgentCounts[projectName],
		})
	}

	resp := OrchestratorSessionsAPIResponse{
		Sessions: result,
		Count:    len(result),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode orchestrator sessions: %v", err), http.StatusInternalServerError)
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

// FileAPIResponse is the JSON structure returned by /api/file.
type FileAPIResponse struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Size    int64  `json:"size"`
	Error   string `json:"error,omitempty"`
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
