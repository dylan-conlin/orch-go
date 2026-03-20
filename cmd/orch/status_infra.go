// Package main provides infrastructure health checking for the status command.
// Extracted from status_cmd.go as part of the status_cmd.go extraction (orch-go-vp594).
package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// InfraServiceStatus represents the health status of an infrastructure service.
type InfraServiceStatus struct {
	Name    string `json:"name"`
	Running bool   `json:"running"`
	Port    int    `json:"port,omitempty"`
	Details string `json:"details,omitempty"`
}

// DaemonStatus represents the status from daemon-status.json.
type DaemonStatus struct {
	PID            int    `json:"pid,omitempty"`
	Status         string `json:"status"`
	LastPoll       string `json:"last_poll,omitempty"`
	LastSpawn      string `json:"last_spawn,omitempty"`
	LastCompletion string `json:"last_completion,omitempty"`
	ReadyCount     int    `json:"ready_count,omitempty"`
	Capacity       struct {
		Max       int `json:"max"`
		Active    int `json:"active"`
		Available int `json:"available"`
	} `json:"capacity,omitempty"`
}

// InfrastructureHealth represents the overall infrastructure health status.
type InfrastructureHealth struct {
	AllHealthy bool                 `json:"all_healthy"`
	Services   []InfraServiceStatus `json:"services"`
	Daemon     *DaemonStatus        `json:"daemon,omitempty"`
}

// checkInfrastructureHealth checks the health of infrastructure services.
// Performs TCP connect tests for dashboard (port 3348) and OpenCode (port 4096),
// and reads daemon status from ~/.orch/daemon-status.json.
func checkInfrastructureHealth() *InfrastructureHealth {
	health := &InfrastructureHealth{
		AllHealthy: true,
		Services:   make([]InfraServiceStatus, 0, 2),
	}

	// Check Dashboard server (orch serve) on port 3348
	dashboardStatus := checkTCPPort("Dashboard", DefaultServePort)
	health.Services = append(health.Services, dashboardStatus)
	if !dashboardStatus.Running {
		health.AllHealthy = false
	}

	// Check OpenCode server on port 4096
	opencodeStatus := checkTCPPort("OpenCode", 4096)
	health.Services = append(health.Services, opencodeStatus)
	if !opencodeStatus.Running {
		health.AllHealthy = false
	}

	// Check daemon status from file
	daemonStatus := readDaemonStatus()
	health.Daemon = daemonStatus
	if daemonStatus == nil || daemonStatus.Status != "running" {
		health.AllHealthy = false
	}

	return health
}

// checkTCPPort performs a TCP connect test to verify a service is listening.
func checkTCPPort(name string, port int) InfraServiceStatus {
	status := InfraServiceStatus{
		Name: name,
		Port: port,
	}

	addr := fmt.Sprintf("localhost:%d", port)
	conn, err := tcpDialTimeout(addr, 1*time.Second)
	if err != nil {
		status.Running = false
		status.Details = "not responding"
		return status
	}
	conn.Close()

	status.Running = true
	status.Details = "listening"
	return status
}

// tcpDialTimeout dials a TCP address with a timeout.
// This is a wrapper to allow for testing.
var tcpDialTimeout = tcpDialTimeoutImpl

// tcpDialTimeoutImpl is the actual implementation of TCP dial using net.DialTimeout.
func tcpDialTimeoutImpl(addr string, timeout time.Duration) (interface{ Close() error }, error) {
	return net.DialTimeout("tcp", addr, timeout)
}

// readDaemonStatus reads the daemon status from ~/.orch/daemon-status.json.
// Validates PID liveness to avoid reporting stale status from dead daemons.
func readDaemonStatus() *DaemonStatus {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	statusPath := filepath.Join(homeDir, ".orch", "daemon-status.json")
	data, err := os.ReadFile(statusPath)
	if err != nil {
		return nil
	}

	var status DaemonStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil
	}

	// Check PID liveness -- stale files from crashed daemons should not report as running
	if status.PID > 0 && !daemon.IsProcessAlive(status.PID) {
		return nil
	}

	return &status
}

// printInfrastructureHealth prints the infrastructure health section.
func printInfrastructureHealth(health *InfrastructureHealth) {
	if health == nil {
		return
	}

	fmt.Println("SYSTEM HEALTH")
	for _, svc := range health.Services {
		emoji := "\u2705"
		if !svc.Running {
			emoji = "\u274c"
		}
		fmt.Printf("  %s %s (port %d) - %s\n", emoji, svc.Name, svc.Port, svc.Details)
	}

	// Print daemon status
	if health.Daemon != nil {
		emoji := "\u2705"
		if health.Daemon.Status != "running" {
			emoji = "\u274c"
		}
		daemonDetails := health.Daemon.Status
		if health.Daemon.Status == "running" && health.Daemon.ReadyCount > 0 {
			daemonDetails = fmt.Sprintf("%s (%d ready)", health.Daemon.Status, health.Daemon.ReadyCount)
		}
		fmt.Printf("  %s Daemon - %s\n", emoji, daemonDetails)
	} else {
		fmt.Println("  \u274c Daemon - not running")
	}
	fmt.Println()
}

// getBeadsIssuePrefix reads the issue_prefix for a project using bd CLI.
// Returns empty string if the command fails or project doesn't have beads.
func getBeadsIssuePrefix(projectPath string) string {
	cmd := exec.Command("bd", "config", "get", "issue_prefix")
	cmd.Dir = projectPath
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	// Output is just the value (e.g., "pw\n")
	return strings.TrimSpace(string(output))
}

// getKBProjectsWithNames fetches registered projects from kb with name and path.
// Returns empty slice if kb is unavailable or fails (graceful degradation).
func getKBProjectsWithNames() []kbProject {
	cmd := exec.Command("kb", "projects", "list", "--json")
	output, err := cmd.Output()
	if err != nil {
		return []kbProject{}
	}

	var projects []kbProject
	if err := json.Unmarshal(output, &projects); err != nil {
		return []kbProject{}
	}

	return projects
}

// findProjectByBeadsPrefix searches for a project with the given beads issue prefix.
// First checks kb's project registry, then falls back to standard locations.
// Returns the project directory path, or empty string if not found.
func findProjectByBeadsPrefix(prefix string) string {
	// Try kb's project registry first
	for _, project := range getKBProjectsWithNames() {
		if projectPrefix := getBeadsIssuePrefix(project.Path); projectPrefix == prefix {
			return project.Path
		}
	}

	// Fall back to checking standard locations
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	candidatePaths := []string{
		filepath.Join(homeDir, "Documents", "personal", prefix),
		filepath.Join(homeDir, prefix),
		filepath.Join(homeDir, "projects", prefix),
		filepath.Join(homeDir, "src", prefix),
	}

	for _, path := range candidatePaths {
		if projectPrefix := getBeadsIssuePrefix(path); projectPrefix == prefix {
			return path
		}
	}

	return ""
}

// findProjectDirByName looks up a project directory by its name or beads prefix.
// First checks kb's project registry, then searches common project locations.
// Verifies the project has a .beads/ directory.
// Returns empty string if not found.
func findProjectDirByName(projectName string) string {
	// Try kb's project registry first (handles non-standard locations)
	for _, project := range getKBProjectsWithNames() {
		if project.Name == projectName {
			// Verify it has a .beads directory
			beadsPath := filepath.Join(project.Path, ".beads")
			if info, err := os.Stat(beadsPath); err == nil && info.IsDir() {
				return project.Path
			}
		}
	}

	// If projectName looks like a beads prefix (short, no hyphens except separators),
	// try finding by prefix instead
	if len(projectName) <= 10 && !strings.Contains(projectName, "/") {
		if path := findProjectByBeadsPrefix(projectName); path != "" {
			return path
		}
	}

	// Fall back to checking standard locations
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	// Common project locations in order of priority
	candidatePaths := []string{
		filepath.Join(homeDir, "Documents", "personal", projectName),
		filepath.Join(homeDir, projectName),
		filepath.Join(homeDir, "projects", projectName),
		filepath.Join(homeDir, "src", projectName),
	}

	for _, path := range candidatePaths {
		// Check if directory exists and has .beads/ (confirms it's a beads-tracked project)
		beadsPath := filepath.Join(path, ".beads")
		if info, err := os.Stat(beadsPath); err == nil && info.IsDir() {
			return path
		}
	}

	return ""
}

// findIssueInAlternateProjects searches registered projects (other than excludeDir)
// that share the same beads issue prefix for a specific issue ID.
// This handles scenarios where multiple projects use the same issue-prefix
// (e.g., harness repo with issue-prefix: "orch-go").
// Returns the project directory containing the issue, or empty string if not found.
func findIssueInAlternateProjects(beadsID, prefix, excludeDir string) string {
	for _, project := range getKBProjectsWithNames() {
		if project.Path == excludeDir {
			continue
		}
		// Check if this project has a .beads directory
		beadsPath := filepath.Join(project.Path, ".beads")
		if info, err := os.Stat(beadsPath); err != nil || !info.IsDir() {
			continue
		}
		// Check if this project uses the same beads prefix
		if projectPrefix := getBeadsIssuePrefix(project.Path); projectPrefix == prefix {
			// Try to find the issue in this project's beads
			if _, err := verify.GetIssue(beadsID, project.Path); err == nil {
				return project.Path
			}
		}
	}
	return ""
}

// findIssueAcrossAllProjects searches ALL registered projects for a beads issue,
// regardless of prefix match. This handles cross-project issues where the beads ID
// prefix doesn't match the hosting project (e.g., orch-go-zrd created in
// scs-special-projects via cross-project spawn).
// Returns the project directory containing the issue, or empty string if not found.
func findIssueAcrossAllProjects(beadsID, excludeDir string) string {
	for _, project := range getKBProjectsWithNames() {
		if project.Path == excludeDir {
			continue
		}
		beadsPath := filepath.Join(project.Path, ".beads")
		if info, err := os.Stat(beadsPath); err != nil || !info.IsDir() {
			continue
		}
		if _, err := verify.GetIssue(beadsID, project.Path); err == nil {
			return project.Path
		}
	}
	return ""
}
