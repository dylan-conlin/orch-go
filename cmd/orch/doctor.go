package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	doctorFix        bool // Attempt to fix issues by starting services
	doctorVerbose    bool // Show verbose output
	doctorStaleOnly  bool // Check stale binary only, exit with code 1 if stale
	doctorSessions   bool // Cross-reference workspaces and OpenCode sessions
	doctorConfig     bool // Check for config drift (plist vs config.yaml)
	doctorDocs       bool // Check for undocumented CLI commands (doc debt)
	doctorWatch      bool // Continuous monitoring with desktop notifications
	doctorDaemon     bool // Run as self-healing background daemon
	doctorDefectScan bool // Scan for Class 2 and Class 5 defect patterns
)

const (
	// DefaultWebPort is the port the web UI (vite dev server) runs on for orch-go.
	DefaultWebPort = 5188
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check health of orch services and optionally fix issues",
	Long: `Check the health status of orch-related services.

Services checked:
  - OpenCode server (default port 4096)
  - orch serve API server (default port 3348)
  - Web UI (vite dev server, port 5188)
  - Beads daemon
  - Overmind services (api, web, opencode)

Use --fix to automatically start services that are not running.
Use --stale-only to check if the orch binary is stale (exit 1 if stale).
Use --sessions to cross-reference workspaces and OpenCode sessions for zombies.
Use --config to detect drift between config.yaml and external config (plist).
Use --docs to check for undocumented CLI commands (doc debt).
Use --watch to continuously monitor services and send desktop notifications on failures.
Use --defect-scan to scan codebase for Class 2 (Multi-Backend Blindness) and Class 5 (Contradictory Authority Signals) patterns.

Examples:
  orch doctor              # Check service health
  orch doctor --fix        # Check and start missing services
  orch doctor --verbose    # Show detailed output
  orch doctor --stale-only # Check binary staleness only (for scripts/hooks)
  orch doctor --sessions   # Cross-reference workspaces and sessions
  orch doctor --config     # Check for config drift
  orch doctor --docs       # Check for undocumented CLI commands
  orch doctor --watch      # Continuous monitoring with notifications
  orch doctor --defect-scan # Scan for Class 2 and Class 5 defect patterns`,
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
	doctorCmd.Flags().BoolVarP(&doctorWatch, "watch", "w", false, "Continuous monitoring with desktop notifications")
	doctorCmd.Flags().BoolVar(&doctorDaemon, "daemon", false, "Run as self-healing background daemon")
	doctorCmd.Flags().BoolVar(&doctorDefectScan, "defect-scan", false, "Scan for Class 2 and Class 5 defect patterns")
	doctorCmd.AddCommand(doctorInstallCmd)
	doctorCmd.AddCommand(doctorUninstallCmd)
	rootCmd.AddCommand(doctorCmd)
}

var doctorInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the doctor daemon as a launchd service",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDoctorInstall()
	},
}

var doctorUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall the doctor daemon launchd service",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDoctorUninstall()
	},
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
			fmt.Printf("⚠️  STALE: binary=%s HEAD=%s\n", shortID(status.BinaryHash), shortID(status.CurrentHash))
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

	// Handle --watch flag for continuous monitoring
	if doctorWatch {
		return runDoctorWatch()
	}

	// Handle --daemon flag for self-healing background daemon
	if doctorDaemon {
		return runDoctorDaemon()
	}

	// Handle --defect-scan flag for static analysis
	if doctorDefectScan {
		return runDefectScan()
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
		binaryServiceStatus.Details = fmt.Sprintf("STALE (binary=%s, HEAD=%s)", shortID(binaryStatus.BinaryHash), shortID(binaryStatus.CurrentHash))
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

	// Check web UI
	webUIStatus := checkWebUI()
	report.Services = append(report.Services, webUIStatus)
	if !webUIStatus.Running {
		report.Healthy = false
	}

	// Check overmind services
	overmindStatus := checkOvermindServices()
	report.Services = append(report.Services, overmindStatus)
	if !overmindStatus.Running {
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
