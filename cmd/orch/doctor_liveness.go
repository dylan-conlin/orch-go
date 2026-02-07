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

func checkOpenCode() ServiceStatus {
	return checkOpenCodeWithClient(opencode.NewClient(serverURL))
}

func checkOpenCodeWithClient(client opencode.ClientInterface) ServiceStatus {
	status := ServiceStatus{
		Name:      "OpenCode",
		Port:      4096,
		URL:       serverURL,
		CanFix:    true,
		FixAction: "opencode serve --port 4096",
	}
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

type BinaryStatus struct {
	Name        string `json:"name"`
	Stale       bool   `json:"stale"`
	BinaryHash  string `json:"binary_hash,omitempty"`
	CurrentHash string `json:"current_hash,omitempty"`
	SourceDir   string `json:"source_dir,omitempty"`
	Error       string `json:"error,omitempty"`
}

// EcosystemBinariesStatus represents the staleness status of all ecosystem binaries.
type EcosystemBinariesStatus struct {
	Binaries []BinaryStatus `json:"binaries"`
	AllFresh bool           `json:"all_fresh"`
}

// checkStaleBinary checks if the orch binary is stale compared to git HEAD.
// This reuses the logic from runVersionSource() in main.go.
func checkStaleBinary() BinaryStatus {
	status := BinaryStatus{
		Name:      "orch",
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

// checkEcosystemBinary checks if a specific ecosystem binary is stale.
// It attempts to run `<binary> version --json` to get version info.
func checkEcosystemBinary(binaryName string) BinaryStatus {
	status := BinaryStatus{
		Name: binaryName,
	}

	// Check if binary exists in PATH
	binaryPath, err := exec.LookPath(binaryName)
	if err != nil {
		status.Error = fmt.Sprintf("%s not found in PATH", binaryName)
		return status
	}

	// Try to get version info via --json flag
	cmd := exec.Command(binaryPath, "version", "--json")
	output, err := cmd.Output()
	if err != nil {
		// Binary doesn't support --json flag yet
		status.Error = fmt.Sprintf("%s version --json not supported", binaryName)
		return status
	}

	// Parse JSON output
	var versionInfo struct {
		GitHash   string `json:"git_hash"`
		SourceDir string `json:"source_dir"`
	}
	if err := json.Unmarshal(output, &versionInfo); err != nil {
		// Binary returned output but it's not valid JSON - treat as unsupported
		status.Error = fmt.Sprintf("%s version --json not supported", binaryName)
		return status
	}

	status.BinaryHash = versionInfo.GitHash
	status.SourceDir = versionInfo.SourceDir

	// Check if source directory exists
	if versionInfo.SourceDir == "" || versionInfo.SourceDir == "unknown" {
		status.Error = "source directory not embedded"
		return status
	}

	if _, err := os.Stat(versionInfo.SourceDir); os.IsNotExist(err) {
		status.Error = fmt.Sprintf("source directory not found: %s", versionInfo.SourceDir)
		return status
	}

	// Get current git hash from source directory
	gitCmd := exec.Command("git", "rev-parse", "HEAD")
	gitCmd.Dir = versionInfo.SourceDir
	gitOutput, err := gitCmd.Output()
	if err != nil {
		status.Error = fmt.Sprintf("could not get current git hash: %v", err)
		return status
	}

	currentHash := strings.TrimSpace(string(gitOutput))
	status.CurrentHash = currentHash

	// Compare hashes
	if versionInfo.GitHash == "" || versionInfo.GitHash == "unknown" {
		status.Error = "git hash not embedded (dev build)"
		return status
	}

	if currentHash != versionInfo.GitHash {
		status.Stale = true
	}

	return status
}

// checkAllEcosystemBinaries checks all Dylan ecosystem binaries for staleness.
func checkAllEcosystemBinaries() EcosystemBinariesStatus {
	result := EcosystemBinariesStatus{
		AllFresh: true,
	}

	// List of ecosystem binaries to check
	binaries := []string{"orch", "kb", "glass", "skillc", "agentlog"}

	for _, binName := range binaries {
		var status BinaryStatus
		if binName == "orch" {
			// Use the existing checkStaleBinary for orch (has embedded version info)
			status = checkStaleBinary()
		} else {
			// Use checkEcosystemBinary for others
			status = checkEcosystemBinary(binName)
		}

		result.Binaries = append(result.Binaries, status)

		// If any binary is stale, mark as not all fresh
		if status.Stale {
			result.AllFresh = false
		}
	}

	return result
}

func checkStalledSessions() ServiceStatus {
	return checkStalledSessionsWithClient(opencode.NewClient(serverURL))
}

func checkStalledSessionsWithClient(client opencode.ClientInterface) ServiceStatus {
	status := ServiceStatus{
		Name:      "Session Health",
		CanFix:    false,
		FixAction: "Use 'orch status' to review stalled sessions, 'orch abandon' to clean up",
	}
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
