// serve_system_config.go provides HTTP handlers for user and daemon configuration management.
// Extracted from serve_system.go — handles /api/config, /api/config/daemon, /api/config/drift,
// and /api/config/regenerate endpoints.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/dylan-conlin/orch-go/pkg/daemonconfig"
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
