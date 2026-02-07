package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/userconfig"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
)

var (
	doctorFix       bool // Attempt to fix issues by starting services
	doctorVerbose   bool // Show verbose output
	doctorStaleOnly bool // Check stale binary only, exit with code 1 if stale
	doctorSessions  bool // Cross-reference workspaces and OpenCode sessions
	doctorConfig    bool // Check for config drift (plist vs config.yaml)
	doctorDocs      bool // Check for undocumented CLI commands (doc debt)
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check health of orch services and optionally fix issues",
	Long: `Check the health status of orch-related services.

Services checked:
  - OpenCode server (default port 4096)
  - orch serve API server (default port 3348)
  - Beads daemon

Use --fix to automatically start services that are not running.
Use --stale-only to check if the orch binary is stale (exit 1 if stale).
Use --sessions to cross-reference workspaces and OpenCode sessions for zombies.
Use --config to detect drift between config.yaml and external config (plist).
Use --docs to check for undocumented CLI commands (doc debt).

Examples:
  orch doctor              # Check service health
  orch doctor --fix        # Check and start missing services
  orch doctor --verbose    # Show detailed output
  orch doctor --stale-only # Check binary staleness only (for scripts/hooks)
  orch doctor --sessions   # Cross-reference workspaces and sessions
  orch doctor --config     # Check for config drift
  orch doctor --docs       # Check for undocumented CLI commands`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDoctor()
	},
}

func init() {
	doctorCmd.Flags().BoolVarP(&doctorFix, "fix", "f", false, "Attempt to start services that are not running")
	doctorCmd.Flags().BoolVarP(&doctorVerbose, "verbose", "v", false, "Show verbose output")
	doctorCmd.Flags().BoolVar(&doctorStaleOnly, "stale-only", false, "Check binary staleness only (exit 1 if stale)")
	doctorCmd.Flags().BoolVar(&doctorSessions, "sessions", false, "Cross-reference workspaces and OpenCode sessions")
	doctorCmd.Flags().BoolVar(&doctorConfig, "config", false, "Check for config drift (plist vs config.yaml)")
	doctorCmd.Flags().BoolVar(&doctorDocs, "docs", false, "Check for undocumented CLI commands (doc debt)")
	rootCmd.AddCommand(doctorCmd)
}

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

func runDoctor() error {
	// Handle --stale-only flag for quick staleness check
	if doctorStaleOnly {
		status := checkStaleBinary()
		if status.Error != "" {
			fmt.Fprintf(os.Stderr, "⚠️  %s\n", status.Error)
			return nil // Not an error, just a warning
		}
		if status.Stale {
			fmt.Printf("⚠️  STALE: binary=%s HEAD=%s\n", status.BinaryHash[:12], status.CurrentHash[:12])
			fmt.Printf("   rebuild: cd %s && make install\n", status.SourceDir)
			os.Exit(1)
		}
		fmt.Println("✓ UP TO DATE")
		return nil
	}

	// Handle --sessions flag for workspace ↔ session cross-reference
	if doctorSessions {
		return runSessionsCrossReference()
	}

	// Handle --config flag for config drift detection
	if doctorConfig {
		return runConfigDriftCheck()
	}

	// Handle --docs flag for doc debt check
	if doctorDocs {
		return runDocDebtCheck()
	}

	fmt.Println("orch doctor - Service Health Check")
	fmt.Println("===================================")
	fmt.Println()

	report := &DoctorReport{
		Healthy:  true,
		Services: make([]ServiceStatus, 0),
	}

	// Check binary staleness first
	binaryStatus := checkStaleBinary()
	binaryServiceStatus := ServiceStatus{
		Name:   "orch binary",
		CanFix: false,
	}
	if binaryStatus.Error != "" {
		binaryServiceStatus.Running = true // Don't mark as failure for dev builds
		binaryServiceStatus.Details = binaryStatus.Error
	} else if binaryStatus.Stale {
		binaryServiceStatus.Running = false
		binaryServiceStatus.Details = fmt.Sprintf("STALE (binary=%s, HEAD=%s)", binaryStatus.BinaryHash[:12], binaryStatus.CurrentHash[:12])
		binaryServiceStatus.FixAction = fmt.Sprintf("cd %s && make install", binaryStatus.SourceDir)
		report.Healthy = false
	} else {
		binaryServiceStatus.Running = true
		binaryServiceStatus.Details = "UP TO DATE"
	}
	report.Services = append(report.Services, binaryServiceStatus)

	// Check OpenCode server
	openCodeStatus := checkOpenCode()
	report.Services = append(report.Services, openCodeStatus)
	if !openCodeStatus.Running {
		report.Healthy = false
	}

	// Check orch serve
	orchServeStatus := checkOrchServe()
	report.Services = append(report.Services, orchServeStatus)
	if !orchServeStatus.Running {
		report.Healthy = false
	}

	// Check beads daemon
	beadsDaemonStatus := checkBeadsDaemon()
	report.Services = append(report.Services, beadsDaemonStatus)
	// Beads daemon is optional, so we don't mark as unhealthy if not running

	// Check for stalled sessions (sessions with no beads comments after >1 min)
	stalledStatus := checkStalledSessions()
	report.Services = append(report.Services, stalledStatus)
	if !stalledStatus.Running {
		report.Healthy = false
	}

	// Print status
	printDoctorReport(report)

	// If --fix flag is set, attempt to start missing services
	if doctorFix && !report.Healthy {
		fmt.Println()
		fmt.Println("Attempting to fix issues...")
		fmt.Println()

		fixed := false

		for _, svc := range report.Services {
			if !svc.Running && svc.CanFix {
				fmt.Printf("Starting %s...\n", svc.Name)

				var err error
				switch svc.Name {
				case "OpenCode":
					err = startOpenCode()
				case "orch serve":
					err = startOrchServe()
				}

				if err != nil {
					fmt.Printf("  ❌ Failed to start %s: %v\n", svc.Name, err)
				} else {
					fmt.Printf("  ✓ Started %s\n", svc.Name)
					fixed = true
				}
			}
		}

		if fixed {
			fmt.Println()
			fmt.Println("Services started. Run 'orch doctor' again to verify.")
		}
	} else if !report.Healthy && !doctorFix {
		fmt.Println()
		fmt.Println("Some services are not running. Use 'orch doctor --fix' to start them.")
	}

	return nil
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

// checkOrchServe checks if the orch serve API is running.
// Uses a simple TCP connect check for reliability (server uses HTTPS).
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
		Timeout: 2 * time.Second,
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
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		status.Running = true

		// Try to parse health response
		var health struct {
			Status string `json:"status"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&health); err == nil && health.Status != "" {
			status.Details = fmt.Sprintf("Status: %s", health.Status)
		} else {
			status.Details = "Health endpoint responding"
		}
	} else {
		// TCP works, HTTPS works, but health endpoint returns non-200
		status.Running = true
		status.Details = fmt.Sprintf("Running (health status %d)", resp.StatusCode)
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

// BinaryStatus represents the staleness status of the orch binary.
type BinaryStatus struct {
	Stale       bool   `json:"stale"`
	BinaryHash  string `json:"binary_hash,omitempty"`
	CurrentHash string `json:"current_hash,omitempty"`
	SourceDir   string `json:"source_dir,omitempty"`
	Error       string `json:"error,omitempty"`
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

// SessionsCrossReferenceReport contains the results of workspace/session/registry cross-reference.
type SessionsCrossReferenceReport struct {
	WorkspaceCount       int `json:"workspace_count"`
	SessionCount         int `json:"session_count"`
	RegistryCount        int `json:"registry_count"`
	OrphanedWorkspaces   int `json:"orphaned_workspaces"` // Workspaces with deleted sessions
	OrphanedSessions     int `json:"orphaned_sessions"`   // Sessions without workspaces
	ZombieSessions       int `json:"zombie_sessions"`     // Sessions active but stuck
	RegistryMismatches   int `json:"registry_mismatches"` // Registry entries without sessions
	OrphanedWorkspaceIDs []string
	OrphanedSessionIDs   []string
	ZombieSessionIDs     []string
	RegistryMismatchIDs  []string
}

// runSessionsCrossReference performs a cross-reference between workspaces, OpenCode sessions,
// and the orchestrator registry to detect orphaned workspaces, orphaned sessions, and zombies.
func runSessionsCrossReference() error {
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	client := opencode.NewClient(serverURL)
	report := &SessionsCrossReferenceReport{}

	// Step 1: Build map of workspace → session IDs
	workspaceDir := filepath.Join(projectDir, ".orch", "workspace")
	workspaceToSession := make(map[string]string) // workspace name → session ID
	sessionToWorkspace := make(map[string]string) // session ID → workspace name
	workspaceBeadsID := make(map[string]string)   // workspace name → beads ID

	entries, err := os.ReadDir(workspaceDir)
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() || entry.Name() == "archived" {
				continue
			}
			wsPath := filepath.Join(workspaceDir, entry.Name())

			// Read session ID
			if data, err := os.ReadFile(filepath.Join(wsPath, ".session_id")); err == nil {
				sessionID := strings.TrimSpace(string(data))
				if sessionID != "" {
					workspaceToSession[entry.Name()] = sessionID
					sessionToWorkspace[sessionID] = entry.Name()
				}
			}

			// Read beads ID
			if data, err := os.ReadFile(filepath.Join(wsPath, ".beads_id")); err == nil {
				beadsID := strings.TrimSpace(string(data))
				if beadsID != "" {
					workspaceBeadsID[entry.Name()] = beadsID
				}
			}
		}
	}
	report.WorkspaceCount = len(workspaceToSession)

	// Step 2: Get all OpenCode sessions for this project
	sessions, err := client.ListDiskSessions(projectDir)
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}
	report.SessionCount = len(sessions)

	// Build session ID set and map for quick lookup
	sessionIDSet := make(map[string]bool)
	sessionByID := make(map[string]opencode.Session)
	for _, s := range sessions {
		sessionIDSet[s.ID] = true
		sessionByID[s.ID] = s
	}

	// Step 3: Load registry (orchestrator sessions)
	registry := loadSessionRegistry()
	report.RegistryCount = len(registry)

	// Step 4: Find orphaned workspaces (workspace has session ID that doesn't exist in OpenCode)
	for name, sessionID := range workspaceToSession {
		if !sessionIDSet[sessionID] {
			report.OrphanedWorkspaces++
			report.OrphanedWorkspaceIDs = append(report.OrphanedWorkspaceIDs, name)
		}
	}

	// Step 5: Find orphaned sessions (session exists but has no workspace)
	for _, s := range sessions {
		if _, hasWorkspace := sessionToWorkspace[s.ID]; !hasWorkspace {
			// Check if this is an orchestrator session (expected to not have workspace tracking)
			isOrchestratorSession := isSessionInRegistry(s.ID, registry)
			if !isOrchestratorSession {
				report.OrphanedSessions++
				report.OrphanedSessionIDs = append(report.OrphanedSessionIDs, s.ID)
			}
		}
	}

	// Step 6: Find zombie sessions (sessions that claim to be active but haven't been updated in >30 min)
	const zombieThreshold = 30 * time.Minute
	now := time.Now()
	for _, s := range sessions {
		updatedAt := time.Unix(s.Time.Updated/1000, 0)
		idleTime := now.Sub(updatedAt)

		// Session is potentially a zombie if:
		// 1. Has a workspace (was spawned by orch)
		// 2. Hasn't been updated in >30 min
		// 3. Is still registered in registry as "active"
		workspaceName := sessionToWorkspace[s.ID]
		if workspaceName != "" && idleTime > zombieThreshold {
			// Check if this session is still marked as active in registry
			for _, reg := range registry {
				if reg.SessionID == s.ID && reg.Status == "active" {
					report.ZombieSessions++
					report.ZombieSessionIDs = append(report.ZombieSessionIDs, s.ID)
					break
				}
			}
		}
	}

	// Step 7: Find registry mismatches (registry entries with session IDs that don't exist)
	for _, reg := range registry {
		if reg.SessionID != "" && !sessionIDSet[reg.SessionID] {
			report.RegistryMismatches++
			report.RegistryMismatchIDs = append(report.RegistryMismatchIDs, reg.WorkspaceName)
		}
	}

	// Print summary report
	printSessionsCrossReferenceReport(report, projectDir, sessionByID, workspaceBeadsID)

	return nil
}

// loadSessionRegistry loads the orchestrator session registry from ~/.orch/sessions.json
func loadSessionRegistry() []struct {
	WorkspaceName string
	SessionID     string
	Status        string
} {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	registryPath := filepath.Join(home, ".orch", "sessions.json")
	data, err := os.ReadFile(registryPath)
	if err != nil {
		return nil
	}

	var registry struct {
		Sessions []struct {
			WorkspaceName string `json:"workspace_name"`
			SessionID     string `json:"session_id"`
			Status        string `json:"status"`
		} `json:"sessions"`
	}

	if err := json.Unmarshal(data, &registry); err != nil {
		return nil
	}

	var result []struct {
		WorkspaceName string
		SessionID     string
		Status        string
	}
	for _, s := range registry.Sessions {
		result = append(result, struct {
			WorkspaceName string
			SessionID     string
			Status        string
		}{s.WorkspaceName, s.SessionID, s.Status})
	}
	return result
}

// isSessionInRegistry checks if a session ID is tracked in the orchestrator registry
func isSessionInRegistry(sessionID string, registry []struct {
	WorkspaceName string
	SessionID     string
	Status        string
}) bool {
	if sessionID == "" {
		return false
	}
	for _, reg := range registry {
		if reg.SessionID == sessionID {
			return true
		}
	}
	return false
}

// printSessionsCrossReferenceReport prints the cross-reference report in a clean format
func printSessionsCrossReferenceReport(report *SessionsCrossReferenceReport, projectDir string, sessionByID map[string]opencode.Session, workspaceBeadsID map[string]string) {
	fmt.Println("orch doctor --sessions")
	fmt.Printf("Workspaces: %d\n", report.WorkspaceCount)
	fmt.Printf("Sessions: %d active\n", report.SessionCount)
	fmt.Printf("Orphaned workspaces: %d (session deleted)\n", report.OrphanedWorkspaces)
	fmt.Printf("Orphaned sessions: %d (no workspace)\n", report.OrphanedSessions)
	fmt.Printf("Zombie sessions: %d\n", report.ZombieSessions)
	if report.RegistryMismatches > 0 {
		fmt.Printf("Registry mismatches: %d\n", report.RegistryMismatches)
	}

	// If everything is clean, show success
	totalIssues := report.OrphanedWorkspaces + report.OrphanedSessions + report.ZombieSessions + report.RegistryMismatches
	if totalIssues == 0 {
		fmt.Println()
		fmt.Println("✓ All workspaces, sessions, and registry entries are properly linked")
		return
	}

	// Show details for issues
	fmt.Println()

	if report.OrphanedWorkspaces > 0 && doctorVerbose {
		fmt.Println("Orphaned workspaces (session was garbage-collected):")
		for _, name := range report.OrphanedWorkspaceIDs {
			beadsID := workspaceBeadsID[name]
			if beadsID != "" {
				fmt.Printf("  - %s [%s]\n", name, beadsID)
			} else {
				fmt.Printf("  - %s\n", name)
			}
		}
		fmt.Println()
	}

	if report.OrphanedSessions > 0 && doctorVerbose {
		fmt.Println("Orphaned sessions (no corresponding workspace):")
		for _, sessionID := range report.OrphanedSessionIDs {
			s := sessionByID[sessionID]
			title := s.Title
			if title == "" {
				title = "(untitled)"
			}
			age := time.Since(time.Unix(s.Time.Created/1000, 0))
			fmt.Printf("  - %s: %s (%.0f days old)\n", sessionID[:12], title, age.Hours()/24)
		}
		fmt.Println()
	}

	if report.ZombieSessions > 0 {
		fmt.Println("⚠️  Zombie sessions (marked active but idle >30min):")
		for _, sessionID := range report.ZombieSessionIDs {
			s := sessionByID[sessionID]
			title := s.Title
			if title == "" {
				title = "(untitled)"
			}
			idleTime := time.Since(time.Unix(s.Time.Updated/1000, 0))
			fmt.Printf("  - %s: %s (idle %.0f min)\n", sessionID[:12], title, idleTime.Minutes())
		}
		fmt.Println()
	}

	if report.RegistryMismatches > 0 && doctorVerbose {
		fmt.Println("Registry mismatches (session ID no longer exists):")
		for _, name := range report.RegistryMismatchIDs {
			fmt.Printf("  - %s\n", name)
		}
		fmt.Println()
	}

	// Recommendations
	fmt.Println("Recommendations:")
	if report.OrphanedWorkspaces > 0 {
		fmt.Println("  - Use 'orch clean --stale' to archive old workspaces")
	}
	if report.OrphanedSessions > 0 {
		fmt.Println("  - Orphaned sessions are usually interactive/test sessions (safe to ignore)")
	}
	if report.ZombieSessions > 0 {
		fmt.Println("  - Use 'orch abandon <id>' to clean up zombie sessions")
	}
	if report.RegistryMismatches > 0 {
		fmt.Println("  - Registry entries with missing sessions can be cleaned with 'orch clean'")
	}
}

// ConfigDrift represents a single configuration drift between expected and actual values.
type ConfigDrift struct {
	Field    string `json:"field"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
}

// ConfigDriftReport contains the results of config drift detection.
type ConfigDriftReport struct {
	Healthy    bool          `json:"healthy"`
	PlistFound bool          `json:"plist_found"`
	Drifts     []ConfigDrift `json:"drifts"`
}

// runConfigDriftCheck compares the expected config (from config.yaml) with the actual plist.
func runConfigDriftCheck() error {
	fmt.Println("orch doctor --config")
	fmt.Println("Checking daemon plist drift against ~/.orch/config.yaml...")
	fmt.Println()

	report, err := checkPlistDrift()
	if err != nil {
		return fmt.Errorf("drift check error: %w", err)
	}

	if !report.PlistFound {
		fmt.Println("✗ Plist not found: ~/Library/LaunchAgents/com.orch.daemon.plist")
		fmt.Println()
		fmt.Println("To generate the plist from config:")
		fmt.Println("  orch config generate plist")
		return nil
	}

	if report.Healthy {
		fmt.Println("✓ No drift detected - plist matches config.yaml")
		return nil
	}

	fmt.Printf("✗ Found %d drift(s):\n", len(report.Drifts))
	fmt.Println()
	for _, drift := range report.Drifts {
		fmt.Printf("  %s:\n", drift.Field)
		fmt.Printf("    config:  %s\n", drift.Expected)
		fmt.Printf("    plist:   %s\n", drift.Actual)
		fmt.Println()
	}

	fmt.Println("To fix, regenerate the plist from config:")
	fmt.Println("  orch config generate plist")

	return nil
}

// checkPlistDrift compares expected plist values from config.yaml with actual plist file.
func checkPlistDrift() (*ConfigDriftReport, error) {
	report := &ConfigDriftReport{
		Healthy: true,
		Drifts:  make([]ConfigDrift, 0),
	}

	// Get expected values from config
	cfg, err := userconfig.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Read actual plist
	plistPath := getPlistPath()
	plistContent, err := os.ReadFile(plistPath)
	if err != nil {
		if os.IsNotExist(err) {
			report.PlistFound = false
			report.Healthy = false
			return report, nil
		}
		return nil, fmt.Errorf("failed to read plist: %w", err)
	}
	report.PlistFound = true

	// Parse plist to extract values
	actualValues, err := parsePlistValues(string(plistContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse plist: %w", err)
	}

	// Compare expected vs actual
	comparisons := []struct {
		Field    string
		Expected string
		Actual   string
	}{
		{
			Field:    "poll_interval",
			Expected: fmt.Sprintf("%d", cfg.DaemonPollInterval()),
			Actual:   actualValues["poll_interval"],
		},
		{
			Field:    "max_agents",
			Expected: fmt.Sprintf("%d", cfg.DaemonMaxAgents()),
			Actual:   actualValues["max_agents"],
		},
		{
			Field:    "label",
			Expected: cfg.DaemonLabel(),
			Actual:   actualValues["label"],
		},
		{
			Field:    "verbose",
			Expected: fmt.Sprintf("%v", cfg.DaemonVerbose()),
			Actual:   actualValues["verbose"],
		},
		{
			Field:    "reflect_issues",
			Expected: fmt.Sprintf("%v", cfg.DaemonReflectIssues()),
			Actual:   actualValues["reflect_issues"],
		},
		{
			Field:    "working_directory",
			Expected: cfg.DaemonWorkingDirectory(),
			Actual:   actualValues["working_directory"],
		},
	}

	for _, c := range comparisons {
		if c.Expected != c.Actual {
			report.Drifts = append(report.Drifts, ConfigDrift{
				Field:    c.Field,
				Expected: c.Expected,
				Actual:   c.Actual,
			})
			report.Healthy = false
		}
	}

	return report, nil
}

// parsePlistValues extracts key values from the daemon plist.
// Uses simple string parsing (not full XML parsing) since the plist has a known structure.
func parsePlistValues(content string) (map[string]string, error) {
	values := make(map[string]string)

	// Extract ProgramArguments to parse flags
	// Look for patterns like:
	// <string>--poll-interval</string>
	// <string>60</string>

	// Parse poll-interval
	if idx := strings.Index(content, "--poll-interval"); idx != -1 {
		// Find the next <string> after this
		remaining := content[idx:]
		if start := strings.Index(remaining, "</string>"); start != -1 {
			remaining = remaining[start+9:] // Skip past </string>
			if strings.HasPrefix(strings.TrimSpace(remaining), "<string>") {
				remaining = strings.TrimSpace(remaining)[8:] // Skip <string>
				if end := strings.Index(remaining, "</string>"); end != -1 {
					values["poll_interval"] = remaining[:end]
				}
			}
		}
	}

	// Parse max-agents
	if idx := strings.Index(content, "--max-agents"); idx != -1 {
		remaining := content[idx:]
		if start := strings.Index(remaining, "</string>"); start != -1 {
			remaining = remaining[start+9:]
			if strings.HasPrefix(strings.TrimSpace(remaining), "<string>") {
				remaining = strings.TrimSpace(remaining)[8:]
				if end := strings.Index(remaining, "</string>"); end != -1 {
					values["max_agents"] = remaining[:end]
				}
			}
		}
	}

	// Parse label (--label flag value)
	if idx := strings.Index(content, "--label"); idx != -1 {
		remaining := content[idx:]
		if start := strings.Index(remaining, "</string>"); start != -1 {
			remaining = remaining[start+9:]
			if strings.HasPrefix(strings.TrimSpace(remaining), "<string>") {
				remaining = strings.TrimSpace(remaining)[8:]
				if end := strings.Index(remaining, "</string>"); end != -1 {
					values["label"] = remaining[:end]
				}
			}
		}
	}

	// Parse verbose (presence of --verbose flag)
	values["verbose"] = "false"
	if strings.Contains(content, "<string>--verbose</string>") {
		values["verbose"] = "true"
	}

	// Parse reflect-issues (--reflect-issues=true/false)
	values["reflect_issues"] = "true" // Default
	if idx := strings.Index(content, "--reflect-issues="); idx != -1 {
		remaining := content[idx+17:] // Skip "--reflect-issues="
		if end := strings.Index(remaining, "</string>"); end != -1 {
			values["reflect_issues"] = remaining[:end]
		}
	}

	// Parse WorkingDirectory
	if idx := strings.Index(content, "<key>WorkingDirectory</key>"); idx != -1 {
		remaining := content[idx:]
		if start := strings.Index(remaining, "<string>"); start != -1 {
			remaining = remaining[start+8:]
			if end := strings.Index(remaining, "</string>"); end != -1 {
				values["working_directory"] = remaining[:end]
			}
		}
	}

	return values, nil
}

// DocDebtReport contains the results of doc debt detection.
type DocDebtReport struct {
	Healthy             bool                      `json:"healthy"`
	TotalCommands       int                       `json:"total_commands"`
	UndocumentedCount   int                       `json:"undocumented_count"`
	DocumentedCount     int                       `json:"documented_count"`
	UndocumentedEntries []userconfig.DocDebtEntry `json:"undocumented_entries"`
}

// runDocDebtCheck surfaces undocumented CLI commands from the doc debt tracker.
func runDocDebtCheck() error {
	fmt.Println("orch doctor --docs")
	fmt.Println("Checking for undocumented CLI commands...")
	fmt.Println()

	debt, err := userconfig.LoadDocDebt()
	if err != nil {
		return fmt.Errorf("failed to load doc debt: %w", err)
	}

	report := &DocDebtReport{
		TotalCommands:       len(debt.Commands),
		UndocumentedEntries: debt.UndocumentedCommands(),
	}
	report.UndocumentedCount = len(report.UndocumentedEntries)
	report.DocumentedCount = report.TotalCommands - report.UndocumentedCount
	report.Healthy = report.UndocumentedCount == 0

	if report.TotalCommands == 0 {
		fmt.Println("No CLI commands tracked yet.")
		fmt.Println("Doc debt tracking starts automatically when new commands are detected during 'orch complete'.")
		return nil
	}

	// Print summary
	fmt.Printf("Total tracked commands: %d\n", report.TotalCommands)
	fmt.Printf("Documented: %d\n", report.DocumentedCount)
	fmt.Printf("Undocumented: %d\n", report.UndocumentedCount)
	fmt.Println()

	if report.Healthy {
		fmt.Println("✓ All tracked CLI commands are documented")
		return nil
	}

	// Print undocumented commands
	fmt.Println("✗ Undocumented commands:")
	fmt.Println()
	for _, entry := range report.UndocumentedEntries {
		fmt.Printf("  • %s (added %s)\n", entry.CommandFile, entry.DateAdded)
		if doctorVerbose && len(entry.DocLocations) > 0 {
			for _, loc := range entry.DocLocations {
				fmt.Printf("      → %s\n", loc)
			}
		}
	}

	fmt.Println()
	fmt.Println("Documentation locations to update:")
	fmt.Println("  - ~/.claude/skills/meta/orchestrator/SKILL.md")
	fmt.Println("  - docs/orch-commands-reference.md")
	fmt.Println()
	fmt.Println("After documenting, mark as complete:")
	fmt.Println("  orch docs mark <command-file>")

	return nil
}
