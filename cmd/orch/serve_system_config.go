package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/dylan-conlin/orch-go/pkg/userconfig"
)

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
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	methodRouter(w, r, map[string]http.HandlerFunc{
		http.MethodGet: s.handleConfigGet,
		http.MethodPut: s.handleConfigPut,
	})
}

// handleConfigGet returns the current user configuration.
func (s *Server) handleConfigGet(w http.ResponseWriter, r *http.Request) {
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

	if err := jsonOK(w, resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode config: %v", err), http.StatusInternalServerError)
		return
	}
}

// handleConfigPut updates the user configuration with the provided values.
func (s *Server) handleConfigPut(w http.ResponseWriter, r *http.Request) {
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

	if err := jsonOK(w, resp); err != nil {
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
	WorkingDirectory *string `json:"working_directory,omitempty"`
}

// handleDaemonConfig handles GET and PUT requests for daemon configuration.
// GET returns current daemon config from ~/.orch/config.yaml
// PUT updates daemon config, writes to config.yaml, regenerates plist, and kicks daemon
func (s *Server) handleDaemonConfig(w http.ResponseWriter, r *http.Request) {
	methodRouter(w, r, map[string]http.HandlerFunc{
		http.MethodGet: s.handleDaemonConfigGet,
		http.MethodPut: s.handleDaemonConfigPut,
	})
}

// handleDaemonConfigGet returns the current daemon configuration.
func (s *Server) handleDaemonConfigGet(w http.ResponseWriter, r *http.Request) {
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
		WorkingDirectory: cfg.DaemonWorkingDirectory(),
		Path:             cfg.DaemonPath(),
	}

	if err := jsonOK(w, resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode daemon config: %v", err), http.StatusInternalServerError)
		return
	}
}

// handleDaemonConfigPut updates the daemon configuration.
// After saving config, it regenerates the plist and kicks the daemon.
func (s *Server) handleDaemonConfigPut(w http.ResponseWriter, r *http.Request) {
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
			WorkingDirectory: cfg.DaemonWorkingDirectory(),
			Path:             cfg.DaemonPath(),
		},
		PlistRegenerated: regenerateErr == nil,
		DaemonKicked:     regenerateErr == nil,
	}

	if regenerateErr != nil {
		resp.RegenerateError = regenerateErr.Error()
	}

	if err := jsonOK(w, resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode daemon config: %v", err), http.StatusInternalServerError)
		return
	}
}
