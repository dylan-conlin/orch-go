package main

import (
	"fmt"
	"os"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/stability"
	"github.com/spf13/cobra"
)

var (
	doctorFix       bool // Attempt to fix issues by starting services
	doctorVerbose   bool // Show verbose output
	doctorStaleOnly bool // Check stale binary only, exit with code 1 if stale
	doctorSessions  bool // Cross-reference workspaces and OpenCode sessions
	doctorConfig    bool // Check for config drift (plist vs config.yaml)
	doctorDocs      bool // Check for undocumented CLI commands (doc debt)
	doctorWatch     bool // Continuous monitoring with desktop notifications
	doctorDaemon    bool // Run as self-healing background daemon
)

const (
	// DefaultWebPort is the port used for dashboard UI checks.
	// The dashboard UI is statically served by orch serve on DefaultServePort.
	DefaultWebPort = DefaultServePort
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check health of orch services and optionally fix issues",
	Long: `Check the health status of orch-related services.

Liveness checks (is it running?):
  - OpenCode server (default port 4096)
  - orch serve API server (default port 3348)
  - Dashboard UI (static, served by orch serve on port 3348)
  - Beads daemon
  - Overmind services (api, daemon, doctor, opencode)

Correctness checks (is it working correctly?):
  - Beads DB integrity (PRAGMA integrity_check)
  - Registry reconciliation (compare against tmux windows)
  - Docker backend (trivial container spawn test)
  - OpenCode API (session list, not just port check)

Use --fix to automatically start services that are not running.
Use --stale-only to check if the orch binary is stale (exit 1 if stale).
Use --sessions to cross-reference workspaces and OpenCode sessions for zombies.
Use --config to detect drift between config.yaml and external config (plist).
Use --docs to check for undocumented CLI commands (doc debt).
Use --watch to continuously monitor services and send desktop notifications on failures.

Examples:
  orch doctor              # Check service health
  orch doctor --fix        # Check and start missing services
  orch doctor --verbose    # Show detailed output
  orch doctor --stale-only # Check binary staleness only (for scripts/hooks)
  orch doctor --sessions   # Cross-reference workspaces and sessions
  orch doctor --config     # Check for config drift
  orch doctor --docs       # Check for undocumented CLI commands
  orch doctor --watch      # Continuous monitoring with notifications`,
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
	doctorCmd.AddCommand(doctorInstallCmd)
	doctorCmd.AddCommand(doctorUninstallCmd)
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
		result := checkAllEcosystemBinaries()

		// Print status for each binary
		hasStale := false
		for _, status := range result.Binaries {
			if status.Error != "" {
				fmt.Printf("⚠️  %s: %s\n", status.Name, status.Error)
			} else if status.Stale {
				fmt.Printf("⚠️  %s STALE: binary=%s HEAD=%s\n", status.Name, status.BinaryHash[:12], status.CurrentHash[:12])
				fmt.Printf("   rebuild: cd %s && make install\n", status.SourceDir)
				hasStale = true
			} else {
				fmt.Printf("✓ %s UP TO DATE\n", status.Name)
			}
		}

		if hasStale {
			os.Exit(1)
		}
		return nil
	}

	// Handle --sessions flag for workspace ↔ session cross-reference
	if doctorSessions {
		return runSessionsCrossReference(opencode.NewClient(serverURL))
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

	// =============================================================================
	// Correctness checks (verify things are working correctly, not just running)
	// =============================================================================

	// Check beads database integrity
	beadsIntegrityStatus := checkBeadsIntegrity()
	report.Services = append(report.Services, beadsIntegrityStatus)
	if !beadsIntegrityStatus.Running {
		report.Healthy = false
	}

	// Check Docker backend (optional - only if docker is installed)
	dockerStatus := checkDockerBackend()
	report.Services = append(report.Services, dockerStatus)
	// Docker check only fails health if Docker IS installed but broken
	// (the check returns Running=true if Docker is not installed)
	if !dockerStatus.Running {
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
			// Record stability intervention: manual fix via orch doctor --fix
			recorder := stability.NewRecorder(stability.DefaultPath())
			var fixedServices []string
			for _, svc := range report.Services {
				if !svc.Running && svc.CanFix {
					fixedServices = append(fixedServices, svc.Name)
				}
			}
			recorder.RecordIntervention(stability.SourceDoctorFix, "orch doctor --fix", fixedServices, "")

			fmt.Println()
			fmt.Println("Services started. Run 'orch doctor' again to verify.")
		}
	} else if !report.Healthy && !doctorFix {
		fmt.Println()
		fmt.Println("Some services are not running. Use 'orch doctor --fix' to start them.")
	}

	return nil
}

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
