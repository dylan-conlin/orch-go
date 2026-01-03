package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
)

var (
	doctorFix       bool // Attempt to fix issues by starting services
	doctorVerbose   bool // Show verbose output
	doctorStaleOnly bool // Check stale binary only, exit with code 1 if stale
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

Examples:
  orch doctor              # Check service health
  orch doctor --fix        # Check and start missing services
  orch doctor --verbose    # Show detailed output
  orch doctor --stale-only # Check binary staleness only (for scripts/hooks)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDoctor()
	},
}

func init() {
	doctorCmd.Flags().BoolVarP(&doctorFix, "fix", "f", false, "Attempt to start services that are not running")
	doctorCmd.Flags().BoolVarP(&doctorVerbose, "verbose", "v", false, "Show verbose output")
	doctorCmd.Flags().BoolVar(&doctorStaleOnly, "stale-only", false, "Check binary staleness only (exit 1 if stale)")
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
func checkOrchServe() ServiceStatus {
	status := ServiceStatus{
		Name:      "orch serve",
		Port:      DefaultServePort,
		URL:       fmt.Sprintf("http://localhost:%d", DefaultServePort),
		CanFix:    true,
		FixAction: fmt.Sprintf("orch serve --port %d", DefaultServePort),
	}

	healthURL := fmt.Sprintf("http://localhost:%d/health", DefaultServePort)
	httpClient := &http.Client{Timeout: 2 * time.Second}
	
	resp, err := httpClient.Get(healthURL)
	if err != nil {
		status.Running = false
		status.Details = "Not responding"
		if doctorVerbose {
			status.Details = fmt.Sprintf("Not responding: %v", err)
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
		status.Running = false
		status.Details = fmt.Sprintf("Unhealthy (status %d)", resp.StatusCode)
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
	healthURL := fmt.Sprintf("http://localhost:%d/health", DefaultServePort)
	httpClient := &http.Client{Timeout: 2 * time.Second}
	
	for i := 0; i < 10; i++ {
		time.Sleep(500 * time.Millisecond)
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
