package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/dylan-conlin/orch-go/pkg/userconfig"
)

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

// handleDaemon returns the daemon status from ~/.orch/daemon-status.json.
// If the daemon is not running (file doesn't exist), returns running: false.
// Also includes utilization metrics computed from events.jsonl.
func (s *Server) handleDaemon(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
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

	if err := jsonOK(w, resp); err != nil {
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

// DriftStatusAPIResponse is the JSON structure returned by GET /api/config/drift.
type DriftStatusAPIResponse struct {
	InSync       bool   `json:"in_sync"`                 // Whether plist matches config
	PlistPath    string `json:"plist_path"`              // Path to the plist file
	PlistExists  bool   `json:"plist_exists"`            // Whether plist file exists
	ConfigPath   string `json:"config_path"`             // Path to config.yaml
	DriftDetails string `json:"drift_details,omitempty"` // Human-readable drift description
}

// handleConfigDrift checks if the plist file matches the current config.
func (s *Server) handleConfigDrift(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	plistPath := getPlistPath()
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

	if err := jsonOK(w, resp); err != nil {
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
func (s *Server) handleConfigRegenerate(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	plistPath := getPlistPath()

	err := regeneratePlistAndKickDaemon()
	resp := RegenerateAPIResponse{
		Success:      err == nil,
		PlistPath:    plistPath,
		DaemonKicked: err == nil,
	}

	if err != nil {
		resp.Error = err.Error()
	}

	if err := jsonOK(w, resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode regenerate response: %v", err), http.StatusInternalServerError)
		return
	}
}

// regeneratePlistAndKickDaemon generates plist from config and restarts the daemon.
// This is shared by PUT /api/config/daemon and POST /api/config/regenerate.
func regeneratePlistAndKickDaemon() error {
	cfg, err := userconfig.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Generate plist content
	plistContent, err := generatePlistContent()
	if err != nil {
		return fmt.Errorf("failed to generate plist: %w", err)
	}

	plistPath := getPlistPath()

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
	// Uses launchctl kickstart -k to force restart
	_ = cfg // Suppress unused warning
	cmd := exec.Command("launchctl", "kickstart", "-k", fmt.Sprintf("gui/%d/com.orch.daemon", os.Getuid()))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to kick daemon: %w", err)
	}

	return nil
}

// generatePlistContent generates plist content from config.
// This is a helper that mirrors the logic from config_cmd.go but returns bytes.
func generatePlistContent() ([]byte, error) {
	cfg, err := userconfig.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	data, err := buildPlistDataForAPI(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build plist data: %w", err)
	}

	tmpl, err := template.New("plist").Parse(plistTemplateAPI)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.Bytes(), nil
}

// PlistDataAPI is a copy of PlistData for use in daemon API handlers
// to avoid import cycles with main package.
type PlistDataAPI struct {
	Label            string
	OrchPath         string
	PollInterval     int
	MaxAgents        int
	IssueLabel       string
	Verbose          bool
	ReflectIssues    bool
	LogPath          string
	WorkingDirectory string
	PATH             string
	Home             string
}

// buildPlistDataForAPI builds plist template data from config.
// This mirrors buildPlistData from config_cmd.go.
func buildPlistDataForAPI(cfg *userconfig.Config) (*PlistDataAPI, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Find orch binary path
	orchPath := findOrchPathForAPI(home)

	// Build PATH from config
	pathDirs := cfg.DaemonPath()
	// Add system paths
	systemPaths := []string{"/usr/local/bin", "/usr/bin", "/bin"}
	allPaths := append(pathDirs, systemPaths...)
	pathStr := strings.Join(allPaths, ":")

	return &PlistDataAPI{
		Label:            "com.orch.daemon",
		OrchPath:         orchPath,
		PollInterval:     cfg.DaemonPollInterval(),
		MaxAgents:        cfg.DaemonMaxAgents(),
		IssueLabel:       cfg.DaemonLabel(),
		Verbose:          cfg.DaemonVerbose(),
		ReflectIssues:    cfg.DaemonReflectIssues(),
		LogPath:          filepath.Join(home, ".orch", "daemon.log"),
		WorkingDirectory: cfg.DaemonWorkingDirectory(),
		PATH:             pathStr,
		Home:             home,
	}, nil
}

// findOrchPathForAPI finds the orch binary path.
// This mirrors findOrchPath from config_cmd.go.
func findOrchPathForAPI(home string) string {
	candidates := []string{
		filepath.Join(home, "bin", "orch"),
		filepath.Join(home, "go", "bin", "orch"),
		filepath.Join(home, ".bun", "bin", "orch"),
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	// Fall back to which
	if path, err := exec.LookPath("orch"); err == nil {
		return path
	}

	// Default to ~/bin/orch
	return filepath.Join(home, "bin", "orch")
}

// plistTemplateAPI is the launchd plist template.
// This is a copy of plistTemplate from config_cmd.go to avoid import issues.
const plistTemplateAPI = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>{{.Label}}</string>

    <key>ProgramArguments</key>
    <array>
        <string>{{.OrchPath}}</string>
        <string>daemon</string>
        <string>run</string>
        <string>--poll-interval</string>
        <string>{{.PollInterval}}</string>
        <string>--max-agents</string>
        <string>{{.MaxAgents}}</string>
        <string>--label</string>
        <string>{{.IssueLabel}}</string>{{if .Verbose}}
        <string>--verbose</string>{{end}}
        <string>--reflect-issues={{.ReflectIssues}}</string>
    </array>

    <key>RunAtLoad</key>
    <true/>

    <key>KeepAlive</key>
    <true/>

    <key>StandardOutPath</key>
    <string>{{.LogPath}}</string>

    <key>StandardErrorPath</key>
    <string>{{.LogPath}}</string>

    <key>WorkingDirectory</key>
    <string>{{.WorkingDirectory}}</string>

    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>{{.PATH}}</string>
        <key>BEADS_NO_DAEMON</key>
        <string>1</string>
    </dict>
</dict>
</plist>
`
