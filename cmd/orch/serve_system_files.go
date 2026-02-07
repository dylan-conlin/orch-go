package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/port"
	"github.com/dylan-conlin/orch-go/pkg/session"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
)

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
func (s *Server) handleServers(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
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

	if err := jsonOK(w, resp); err != nil {
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
func (s *Server) handleServices(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	s.ServiceMonitorMu.RLock()
	monitor := s.ServiceMonitor
	s.ServiceMonitorMu.RUnlock()

	if monitor == nil {
		// Service monitor not initialized (shouldn't happen but handle gracefully)
		_ = jsonOK(w, ServicesAPIResponse{
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
	projectName := filepath.Base(s.SourceDir)

	resp := ServicesAPIResponse{
		Project:      projectName,
		Services:     services,
		TotalCount:   len(services),
		RunningCount: runningCount,
		StoppedCount: stoppedCount,
	}

	if err := jsonOK(w, resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode services: %v", err), http.StatusInternalServerError)
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
func (s *Server) handleOrchestratorSessions(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	project := r.URL.Query().Get("project")

	registry := session.NewRegistry("")
	sessions, err := registry.ListActive()
	if err != nil {
		// Return empty list if registry doesn't exist yet
		_ = jsonOK(w, OrchestratorSessionsAPIResponse{
			Sessions: []OrchestratorSessionAPIItem{},
			Count:    0,
		})
		return
	}

	// Get active agents to count children per project
	client := opencode.NewClient(s.ServerURL)
	opencodeSessions, _ := client.ListSessions("")

	// Count active agents per project
	projectAgentCounts := make(map[string]int)
	maxIdleTime := 30 * time.Minute
	now := time.Now()
	for _, sess := range opencodeSessions {
		updatedAt := time.Unix(sess.Time.Updated/1000, 0)
		if now.Sub(updatedAt) <= maxIdleTime {
			// Extract project from title (e.g., "og-feat-xxx" -> "orch-go")
			if beadsID := extractBeadsIDFromTitle(sess.Title); beadsID != "" {
				proj := extractProjectFromBeadsID(beadsID)
				if proj != "" {
					projectAgentCounts[proj]++
				}
			}
		}
	}

	var result []OrchestratorSessionAPIItem

	for _, session := range sessions {
		projectName := filepath.Base(session.ProjectDir)

		// Filter by project if specified
		if project != "" && projectName != project {
			continue
		}

		duration := now.Sub(session.SpawnTime)

		result = append(result, OrchestratorSessionAPIItem{
			WorkspaceName:   session.WorkspaceName,
			SessionID:       session.SessionID,
			Goal:            session.Goal,
			Duration:        formatDuration(duration),
			DurationSeconds: int64(duration.Seconds()),
			Project:         projectName,
			ProjectDir:      session.ProjectDir,
			Status:          session.Status,
			SpawnTime:       session.SpawnTime.Format(time.RFC3339),
			ChildAgentCount: projectAgentCounts[projectName],
		})
	}

	resp := OrchestratorSessionsAPIResponse{
		Sessions: result,
		Count:    len(result),
	}

	if err := jsonOK(w, resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode orchestrator sessions: %v", err), http.StatusInternalServerError)
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
func (s *Server) handleScreenshots(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	agentID := r.URL.Query().Get("agent_id")
	projectDir := r.URL.Query().Get("project_dir")

	if agentID == "" {
		_ = jsonOK(w, ScreenshotsAPIResponse{
			Error: "agent_id query parameter is required",
		})
		return
	}

	if projectDir == "" {
		_ = jsonOK(w, ScreenshotsAPIResponse{
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
		_ = jsonOK(w, ScreenshotsAPIResponse{
			AgentID:     agentID,
			Screenshots: []string{},
		})
		return
	}

	// Read directory contents
	entries, err := os.ReadDir(screenshotsDir)
	if err != nil {
		_ = jsonOK(w, ScreenshotsAPIResponse{
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
	_ = jsonOK(w, ScreenshotsAPIResponse{
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
func (s *Server) handleFile(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		_ = jsonOK(w, FileAPIResponse{
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
		_ = jsonOK(w, FileAPIResponse{
			Path:  filePath,
			Error: "access denied: path must be in .kb/ or .orch/workspace/",
		})
		return
	}

	// Clean the path to prevent traversal attacks
	cleanPath := filepath.Clean(filePath)
	if cleanPath != filePath {
		_ = jsonOK(w, FileAPIResponse{
			Path:  filePath,
			Error: "invalid path",
		})
		return
	}

	// Check if file exists
	info, err := os.Stat(cleanPath)
	if err != nil {
		_ = jsonOK(w, FileAPIResponse{
			Path:  cleanPath,
			Error: fmt.Sprintf("file not found: %v", err),
		})
		return
	}

	// Don't allow reading directories
	if info.IsDir() {
		_ = jsonOK(w, FileAPIResponse{
			Path:  cleanPath,
			Error: "cannot read directory",
		})
		return
	}

	// Read file content
	content, err := os.ReadFile(cleanPath)
	if err != nil {
		_ = jsonOK(w, FileAPIResponse{
			Path:  cleanPath,
			Error: fmt.Sprintf("failed to read file: %v", err),
		})
		return
	}

	// Return success response
	_ = jsonOK(w, FileAPIResponse{
		Path:    cleanPath,
		Content: string(content),
		Size:    info.Size(),
	})
}
