package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/spf13/cobra"
)

var (
	doctorFix     bool // Attempt to fix issues by starting services
	doctorVerbose bool // Show verbose output
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

Examples:
  orch doctor              # Check service health
  orch doctor --fix        # Check and start missing services
  orch doctor --verbose    # Show detailed output`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDoctor()
	},
}

func init() {
	doctorCmd.Flags().BoolVarP(&doctorFix, "fix", "f", false, "Attempt to start services that are not running")
	doctorCmd.Flags().BoolVarP(&doctorVerbose, "verbose", "v", false, "Show verbose output")
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
	fmt.Println("orch doctor - Service Health Check")
	fmt.Println("===================================")
	fmt.Println()

	report := &DoctorReport{
		Healthy:  true,
		Services: make([]ServiceStatus, 0),
	}

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
		URL:       fmt.Sprintf("http://127.0.0.1:%d", DefaultServePort),
		CanFix:    true,
		FixAction: fmt.Sprintf("orch serve --port %d", DefaultServePort),
	}

	healthURL := fmt.Sprintf("http://127.0.0.1:%d/health", DefaultServePort)
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
	healthURL := fmt.Sprintf("http://127.0.0.1:%d/health", DefaultServePort)
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
