package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// ServiceStatus represents the health status of a service.
type ServiceStatus struct {
	Name      string `json:"name"`
	Running   bool   `json:"running"`
	Port      int    `json:"port,omitempty"`
	URL       string `json:"url,omitempty"`
	Details   string `json:"details,omitempty"`
	CanFix    bool   `json:"can_fix"`
	FixAction string `json:"fix_action,omitempty"`
}

// DoctorReport is the overall health report.
type DoctorReport struct {
	Healthy  bool            `json:"healthy"`
	Services []ServiceStatus `json:"services"`
}

// BinaryStatus represents the staleness status of the orch binary.
type BinaryStatus struct {
	Stale       bool   `json:"stale"`
	BinaryHash  string `json:"binary_hash,omitempty"`
	CurrentHash string `json:"current_hash,omitempty"`
	SourceDir   string `json:"source_dir,omitempty"`
	Error       string `json:"error,omitempty"`
}

// checkOpenCode checks if the OpenCode server is running.
func checkOpenCode() ServiceStatus {
	status := ServiceStatus{
		Name:      "OpenCode",
		Port:      4096,
		URL:       serverURL,
		CanFix:    true,
		FixAction: "opencode serve --port 4096",
	}

	client := opencode.NewClient(serverURL)
	_, err := client.ListSessions("")
	if err == nil {
		status.Running = true
		status.Details = "API responding"
	} else {
		status.Running = false
		status.Details = "Not responding"
		if doctorVerbose {
			status.Details = fmt.Sprintf("Not responding: %v", err)
		}
	}

	return status
}

// checkOrchServe checks if the orch serve API is running and dashboard endpoints work.
// Uses a simple TCP connect check first, then verifies /health and /api/agents endpoints.
// The /api/agents check is critical because the dashboard depends on it for functionality.
func checkOrchServe() ServiceStatus {
	status := ServiceStatus{
		Name:      "orch serve",
		Port:      DefaultServePort,
		URL:       fmt.Sprintf("https://localhost:%d", DefaultServePort),
		CanFix:    true,
		FixAction: "Run: orch serve &",
	}

	// Simple TCP connect check - more reliable than HTTP since server uses HTTPS
	addr := fmt.Sprintf("localhost:%d", DefaultServePort)
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		status.Running = false
		status.Details = "Not listening"
		if doctorVerbose {
			status.Details = fmt.Sprintf("Not listening: %v", err)
		}
		return status
	}
	conn.Close()

	// TCP connect succeeded, try HTTPS health check for more details
	healthURL := fmt.Sprintf("https://localhost:%d/health", DefaultServePort)
	httpClient := &http.Client{
		Timeout: 5 * time.Second, // Increased timeout for /api/agents which may be slower
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, //nolint:gosec // Self-signed localhost cert
			},
		},
	}

	resp, err := httpClient.Get(healthURL)
	if err != nil {
		// TCP worked but HTTPS failed - server might still be starting
		status.Running = true
		status.Details = "Port listening (health check pending)"
		if doctorVerbose {
			status.Details = fmt.Sprintf("Port listening (health check failed: %v)", err)
		}
		return status
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// TCP works, HTTPS works, but health endpoint returns non-200
		status.Running = true
		status.Details = fmt.Sprintf("Running (health status %d)", resp.StatusCode)
		return status
	}

	// Health endpoint OK, now verify /api/agents endpoint (critical for dashboard)
	// The dashboard fetches agent data from this endpoint - if it fails, dashboard is non-functional
	agentsURL := fmt.Sprintf("https://localhost:%d/api/agents?since=1h", DefaultServePort)
	agentsResp, err := httpClient.Get(agentsURL)
	if err != nil {
		status.Running = true
		status.Details = "Health OK, /api/agents unreachable"
		if doctorVerbose {
			status.Details = fmt.Sprintf("Health OK, /api/agents failed: %v", err)
		}
		return status
	}
	defer agentsResp.Body.Close()

	if agentsResp.StatusCode != http.StatusOK {
		status.Running = true
		status.Details = fmt.Sprintf("Health OK, /api/agents status %d", agentsResp.StatusCode)
		return status
	}

	// Verify response is valid JSON array (agents endpoint returns [])
	var agents []interface{}
	if err := json.NewDecoder(agentsResp.Body).Decode(&agents); err != nil {
		status.Running = true
		status.Details = "Health OK, /api/agents invalid JSON"
		if doctorVerbose {
			status.Details = fmt.Sprintf("Health OK, /api/agents invalid JSON: %v", err)
		}
		return status
	}

	// All checks passed - server is fully functional for dashboard
	status.Running = true
	status.Details = fmt.Sprintf("Dashboard ready (%d agents)", len(agents))

	return status
}

// checkWebUI checks if the web UI (vite dev server) is running.
// Uses plain HTTP (not HTTPS) since vite serves over HTTP.
func checkWebUI() ServiceStatus {
	status := ServiceStatus{
		Name:      "Web UI",
		Port:      DefaultWebPort,
		URL:       fmt.Sprintf("http://localhost:%d", DefaultWebPort),
		CanFix:    false, // Web UI is started via overmind, not directly
		FixAction: "Run: overmind restart web",
	}

	// Simple TCP connect check first
	addr := fmt.Sprintf("localhost:%d", DefaultWebPort)
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		status.Running = false
		status.Details = "Not listening"
		if doctorVerbose {
			status.Details = fmt.Sprintf("Not listening: %v", err)
		}
		return status
	}
	conn.Close()

	// TCP connect succeeded, try HTTP GET for more details
	httpClient := &http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := httpClient.Get(status.URL)
	if err != nil {
		// TCP worked but HTTP failed - server might still be starting
		status.Running = true
		status.Details = "Port listening (HTTP check pending)"
		if doctorVerbose {
			status.Details = fmt.Sprintf("Port listening (HTTP failed: %v)", err)
		}
		return status
	}
	defer resp.Body.Close()

	// Any response from vite is good enough (could be 200 for app, or 404 for missing route)
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound {
		status.Running = true
		status.Details = "Responding"
	} else {
		status.Running = true
		status.Details = fmt.Sprintf("Running (status %d)", resp.StatusCode)
	}

	return status
}

// checkOvermindServices checks if overmind is running via launchd supervision.
// Since overmind runs in daemon mode, we check if the process is running.
func checkOvermindServices() ServiceStatus {
	status := ServiceStatus{
		Name:      "Overmind (launchd)",
		CanFix:    false,
		FixAction: "launchctl kickstart -k gui/$(id -u)/com.overmind.orch-go",
	}

	// Check for overmind process
	cmd := exec.Command("pgrep", "-f", "overmind start")
	output, err := cmd.Output()
	if err != nil || len(output) == 0 {
		status.Running = false
		status.Details = "overmind process not running"
		return status
	}

	// Extract PID from output
	pids := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(pids) > 0 {
		status.Running = true
		if len(pids) == 1 {
			status.Details = fmt.Sprintf("Running (PID %s)", pids[0])
		} else {
			status.Details = fmt.Sprintf("Running (%d instances)", len(pids))
		}
	}

	return status
}

// checkBeadsDaemon checks if the beads daemon is running.
func checkBeadsDaemon() ServiceStatus {
	status := ServiceStatus{
		Name:      "Beads Daemon",
		CanFix:    false,
		FixAction: "bd daemon start",
	}

	// Check for bd daemon process
	cmd := exec.Command("pgrep", "-f", "bd.*daemon")
	output, err := cmd.Output()
	if err != nil || len(output) == 0 {
		// Also check for beads serve
		cmd = exec.Command("pgrep", "-f", "beads.*serve")
		output, err = cmd.Output()
		if err != nil || len(output) == 0 {
			status.Running = false
			status.Details = "Not running (optional)"
			return status
		}
	}

	status.Running = true
	status.Details = "Process found"
	return status
}

// startOpenCode starts the OpenCode server in the background.
func startOpenCode() error {
	// Start OpenCode server in background, fully detached via shell
	// This ensures the process survives even if the parent is killed
	// Set ORCH_WORKER=1 so agents spawned by this server know they are orch-managed workers
	cmd := exec.Command("sh", "-c", "ORCH_WORKER=1 opencode serve --port 4096 </dev/null >/dev/null 2>&1 &")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start OpenCode: %w", err)
	}

	// Wait for it to be ready (poll for up to 10 seconds)
	client := opencode.NewClient(serverURL)
	for i := 0; i < 20; i++ {
		time.Sleep(500 * time.Millisecond)
		_, err := client.ListSessions("")
		if err == nil {
			return nil
		}
	}

	return fmt.Errorf("OpenCode started but not responding after 10s")
}

// startOrchServe starts the orch serve API server in the background.
func startOrchServe() error {
	// Find the orch binary path
	orchPath, err := exec.LookPath("orch")
	if err != nil {
		// Try with full path from home directory
		homeDir, _ := os.UserHomeDir()
		orchPath = homeDir + "/bin/orch"
		if _, err := os.Stat(orchPath); os.IsNotExist(err) {
			return fmt.Errorf("orch binary not found in PATH or ~/bin/orch")
		}
	}

	// Start orch serve in background
	cmd := exec.Command("sh", "-c", fmt.Sprintf("nohup %s serve --port %d </dev/null >/dev/null 2>&1 &", orchPath, DefaultServePort))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start orch serve: %w", err)
	}

	// Wait for it to be ready (poll for up to 5 seconds)
	// First check TCP, then HTTPS health endpoint
	addr := fmt.Sprintf("localhost:%d", DefaultServePort)

	for i := 0; i < 10; i++ {
		time.Sleep(500 * time.Millisecond)

		// Quick TCP check first
		conn, err := net.DialTimeout("tcp", addr, 1*time.Second)
		if err != nil {
			continue
		}
		conn.Close()

		// TCP succeeded, now verify HTTPS health
		healthURL := fmt.Sprintf("https://localhost:%d/health", DefaultServePort)
		httpClient := &http.Client{
			Timeout: 2 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, //nolint:gosec // Self-signed localhost cert
				},
			},
		}

		resp, err := httpClient.Get(healthURL)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
	}

	return fmt.Errorf("orch serve started but not responding after 5s")
}

// checkStaleBinary checks if the orch binary is stale compared to git HEAD.
// This reuses the logic from runVersionSource() in main.go.
func checkStaleBinary() BinaryStatus {
	status := BinaryStatus{
		SourceDir: sourceDir,
	}

	// Check if source directory is embedded
	if sourceDir == "unknown" {
		status.Error = "source directory not embedded (dev build)"
		return status
	}

	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		status.Error = fmt.Sprintf("source directory not found: %s", sourceDir)
		return status
	}

	// Check current git hash in source directory
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = sourceDir
	output, err := cmd.Output()
	if err != nil {
		status.Error = fmt.Sprintf("could not get current git hash: %v", err)
		return status
	}

	currentHash := strings.TrimSpace(string(output))
	status.CurrentHash = currentHash

	// Compare hashes
	if gitHash == "unknown" {
		status.Error = "git hash not embedded (dev build)"
		status.BinaryHash = gitHash
		return status
	}

	status.BinaryHash = gitHash
	if currentHash != gitHash {
		status.Stale = true
	}

	return status
}

// checkStalledSessions checks for sessions that spawned but never reported a Phase status.
// These are sessions with:
// - An active OpenCode session
// - A beads ID
// - No beads comments after >1 minute
// This indicates a potential failed-to-start situation.
func checkStalledSessions() ServiceStatus {
	status := ServiceStatus{
		Name:      "Session Health",
		CanFix:    false,
		FixAction: "Use 'orch status' to review stalled sessions, 'orch abandon' to clean up",
	}

	client := opencode.NewClient(serverURL)
	now := time.Now()

	// Get current project directory for session queries
	projectDir, _ := os.Getwd()

	// Fetch sessions
	var sessions []opencode.Session
	seenSessionIDs := make(map[string]bool)

	if projectDir != "" {
		dirSessions, err := client.ListSessions(projectDir)
		if err == nil {
			for _, s := range dirSessions {
				if !seenSessionIDs[s.ID] {
					seenSessionIDs[s.ID] = true
					sessions = append(sessions, s)
				}
			}
		}
	}

	globalSessions, err := client.ListSessions("")
	if err == nil {
		for _, s := range globalSessions {
			if !seenSessionIDs[s.ID] {
				seenSessionIDs[s.ID] = true
				sessions = append(sessions, s)
			}
		}
	}

	if len(sessions) == 0 {
		status.Running = true
		status.Details = "No active sessions"
		return status
	}

	// Check each recent session for stalled status
	const maxIdleTime = 30 * time.Minute
	var stalledSessions []string

	for _, s := range sessions {
		updatedAt := time.Unix(s.Time.Updated/1000, 0)
		createdAt := time.Unix(s.Time.Created/1000, 0)

		// Only check recently active sessions
		if now.Sub(updatedAt) > maxIdleTime {
			continue
		}

		// Skip sessions less than 1 minute old (still starting up)
		if now.Sub(createdAt) < time.Minute {
			continue
		}

		// Extract beads ID from session title
		beadsID := extractBeadsIDFromTitle(s.Title)
		if beadsID == "" {
			continue
		}

		// Check if this session has any beads comments
		hasComments, err := verify.HasBeadsComment(beadsID)
		if err != nil {
			// Skip on error (daemon might be down)
			continue
		}

		if !hasComments {
			// This session has no comments after >1 min - potential stalled session
			stalledSessions = append(stalledSessions, beadsID)
		}
	}

	if len(stalledSessions) == 0 {
		status.Running = true
		status.Details = fmt.Sprintf("%d active sessions, all reporting progress", len(sessions))
		return status
	}

	// Found stalled sessions
	status.Running = false
	if len(stalledSessions) == 1 {
		status.Details = fmt.Sprintf("⚠️ 1 stalled session (no Phase report after >1 min): %s", stalledSessions[0])
	} else {
		status.Details = fmt.Sprintf("⚠️ %d stalled sessions (no Phase report after >1 min): %s", len(stalledSessions), strings.Join(stalledSessions, ", "))
	}
	return status
}

// printDoctorReport prints the health report in a formatted way.
func printDoctorReport(report *DoctorReport) {
	for _, svc := range report.Services {
		var statusIcon string
		if svc.Running {
			statusIcon = "✓"
		} else {
			statusIcon = "✗"
		}

		fmt.Printf("%s %s", statusIcon, svc.Name)
		if svc.Port > 0 {
			fmt.Printf(" (port %d)", svc.Port)
		}
		fmt.Println()

		if svc.Details != "" {
			fmt.Printf("  %s\n", svc.Details)
		}

		if !svc.Running && svc.FixAction != "" && doctorVerbose {
			fmt.Printf("  Fix: %s\n", svc.FixAction)
		}
	}

	fmt.Println()
	if report.Healthy {
		fmt.Println("All required services are running.")
	} else {
		fmt.Println("Some services are not running.")
	}
}
