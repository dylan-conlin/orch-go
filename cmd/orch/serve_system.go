// serve_system.go provides HTTP handlers for system-level API endpoints.
// Handles /api/usage, /api/focus, /api/servers, /api/services, /api/screenshots, and /api/file.
//
// Related files (extracted from this file):
//   - serve_system_config.go: Configuration handlers (/api/config, /api/config/daemon, /api/config/drift)
//   - serve_system_daemon.go: Daemon status handler (/api/daemon)
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/account"
	"github.com/dylan-conlin/orch-go/pkg/focus"
	"github.com/dylan-conlin/orch-go/pkg/execution"
	"github.com/dylan-conlin/orch-go/pkg/port"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
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
		client := execution.NewOpenCodeAdapter(serverURL)
		sessions, _ := client.ListSessions(context.Background(), "")

		var activeWork []focus.ActiveWork
		for _, s := range sessions {
			if beadsID := extractBeadsIDFromTitle(s.Title); beadsID != "" {
				activeWork = append(activeWork, focus.ActiveWork{BeadsID: beadsID})
			}
		}

		drift := store.CheckDrift(activeWork)
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
