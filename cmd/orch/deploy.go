package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	deploySkipBuild   bool // Skip the build step (useful if already built)
	deploySkipOrphans bool // Skip orphan process cleanup
	deployVerbose     bool // Show verbose output
	deployTimeout     int  // Health check timeout in seconds
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Atomic deployment: rebuild binary, restart services, verify health",
	Long: `Deploy changes atomically with a single command.

Steps performed:
  1. Build binary (make build)
  2. Kill orphaned processes (stale vite, bd processes)
  3. Restart overmind services (api, web, opencode)
  4. Wait for health checks to pass
  5. Display deployment status

This ensures that rebuilding the binary and restarting services happens
atomically, avoiding the common "running old binary after rebuild" problem.

Examples:
  orch deploy              # Full deployment
  orch deploy --skip-build # Skip build step (already built)
  orch deploy -v           # Verbose output
  orch deploy --timeout 60 # Wait up to 60s for health checks`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDeploy()
	},
}

func init() {
	deployCmd.Flags().BoolVar(&deploySkipBuild, "skip-build", false, "Skip the build step (use existing binary)")
	deployCmd.Flags().BoolVar(&deploySkipOrphans, "skip-orphans", false, "Skip orphan process cleanup")
	deployCmd.Flags().BoolVarP(&deployVerbose, "verbose", "v", false, "Show verbose output")
	deployCmd.Flags().IntVar(&deployTimeout, "timeout", 30, "Health check timeout in seconds")
	rootCmd.AddCommand(deployCmd)
}

// DeployStep represents a step in the deployment process.
type DeployStep struct {
	Name    string
	Status  string // pending, running, success, failed, skipped
	Message string
}

func runDeploy() error {
	fmt.Println("orch deploy - Atomic Deployment")
	fmt.Println("================================")
	fmt.Println()

	steps := []DeployStep{
		{Name: "Building orch binary", Status: "pending"},
		{Name: "Killing orphaned processes", Status: "pending"},
		{Name: "Restarting services", Status: "pending"},
		{Name: "Health checks", Status: "pending"},
	}

	// Step 1: Build binary
	if deploySkipBuild {
		steps[0].Status = "skipped"
		steps[0].Message = "Skipped (--skip-build)"
		printStep(steps[0])
	} else {
		steps[0].Status = "running"
		printStep(steps[0])

		if err := runBuildStep(); err != nil {
			steps[0].Status = "failed"
			steps[0].Message = err.Error()
			printStep(steps[0])
			return fmt.Errorf("build failed: %w", err)
		}

		steps[0].Status = "success"
		steps[0].Message = "Built successfully"
		printStep(steps[0])
	}

	// Step 2: Kill orphaned processes
	if deploySkipOrphans {
		steps[1].Status = "skipped"
		steps[1].Message = "Skipped (--skip-orphans)"
		printStep(steps[1])
	} else {
		steps[1].Status = "running"
		printStep(steps[1])

		killed := killOrphanedProcesses()
		steps[1].Status = "success"
		if killed > 0 {
			steps[1].Message = fmt.Sprintf("Killed %d orphaned process(es)", killed)
		} else {
			steps[1].Message = "No orphaned processes found"
		}
		printStep(steps[1])
	}

	// Step 3: Restart overmind services
	steps[2].Status = "running"
	printStep(steps[2])

	if err := restartOvermind(); err != nil {
		steps[2].Status = "failed"
		steps[2].Message = err.Error()
		printStep(steps[2])
		return fmt.Errorf("restart failed: %w", err)
	}

	steps[2].Status = "success"
	steps[2].Message = "Overmind restarted"
	printStep(steps[2])

	// Step 4: Health checks
	steps[3].Status = "running"
	printStep(steps[3])

	if err := waitForHealthChecks(time.Duration(deployTimeout) * time.Second); err != nil {
		steps[3].Status = "failed"
		steps[3].Message = err.Error()
		printStep(steps[3])
		return fmt.Errorf("health checks failed: %w", err)
	}

	steps[3].Status = "success"
	steps[3].Message = "All services healthy"
	printStep(steps[3])

	// Final status
	fmt.Println()
	fmt.Println("Deployment complete!")
	fmt.Printf("Dashboard available at http://localhost:%d\n", DefaultWebPort)

	return nil
}

// printStep prints a step with appropriate formatting.
func printStep(step DeployStep) {
	var icon string
	switch step.Status {
	case "pending":
		icon = "○"
	case "running":
		icon = "●"
	case "success":
		icon = "✓"
	case "failed":
		icon = "✗"
	case "skipped":
		icon = "○"
	}

	// Clear line and print step
	fmt.Printf("\r%s %s", icon, step.Name)

	if step.Message != "" {
		fmt.Printf("... %s", step.Message)
	} else if step.Status == "running" {
		fmt.Printf("...")
	}

	fmt.Println()
}

// runBuildStep runs make build in the source directory.
func runBuildStep() error {
	// Use embedded source directory if available, otherwise try to detect
	buildDir := sourceDir
	if buildDir == "unknown" {
		// Try current directory
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("cannot determine build directory: %w", err)
		}

		// Check if Makefile exists
		if _, err := os.Stat(filepath.Join(cwd, "Makefile")); os.IsNotExist(err) {
			return fmt.Errorf("Makefile not found in %s (is this the orch-go directory?)", cwd)
		}
		buildDir = cwd
	}

	if deployVerbose {
		fmt.Printf("  Building in %s...\n", buildDir)
	}

	cmd := exec.Command("make", "install")
	cmd.Dir = buildDir
	cmd.Env = os.Environ()

	if deployVerbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("make install failed: %w", err)
	}

	return nil
}

// killOrphanedProcesses kills orphaned vite and bd processes.
// Returns the number of processes killed.
func killOrphanedProcesses() int {
	killed := 0

	// Kill orphaned vite processes (those with PPID=1, indicating parent died)
	// Note: We use pkill carefully to avoid killing the main vite process
	viteKilled := killOrphanedVite()
	killed += viteKilled

	if deployVerbose && viteKilled > 0 {
		fmt.Printf("  Killed %d orphaned vite process(es)\n", viteKilled)
	}

	// Kill long-running bd processes (stuck bd commands)
	bdKilled := killStuckBdProcesses()
	killed += bdKilled

	if deployVerbose && bdKilled > 0 {
		fmt.Printf("  Killed %d stuck bd process(es)\n", bdKilled)
	}

	return killed
}

// killOrphanedVite kills orphaned vite processes.
// An orphaned vite process has PPID=1 (parent died).
func killOrphanedVite() int {
	// Find vite processes with PPID=1
	// ps -eo pid,ppid,comm | grep vite | awk '$2==1 {print $1}'
	cmd := exec.Command("bash", "-c", `ps -eo pid,ppid,comm | grep vite | awk '$2==1 {print $1}'`)
	output, err := cmd.Output()
	if err != nil || len(output) == 0 {
		return 0
	}

	pids := strings.Fields(string(output))
	killed := 0
	for _, pid := range pids {
		if pid == "" {
			continue
		}
		killCmd := exec.Command("kill", "-9", pid)
		if err := killCmd.Run(); err == nil {
			killed++
		}
	}

	return killed
}

// killStuckBdProcesses kills bd processes running for more than 5 minutes.
// These are likely stuck health checks or failed commands.
func killStuckBdProcesses() int {
	// Find bd processes older than 5 minutes
	// This uses a more conservative approach - only kill processes that are clearly stuck
	cmd := exec.Command("bash", "-c", `ps -eo pid,etimes,comm | grep -E '^\s*[0-9]+\s+[3-9][0-9][0-9].*bd$' | awk '{print $1}'`)
	output, err := cmd.Output()
	if err != nil || len(output) == 0 {
		return 0
	}

	pids := strings.Fields(string(output))
	killed := 0
	for _, pid := range pids {
		if pid == "" {
			continue
		}
		killCmd := exec.Command("kill", "-9", pid)
		if err := killCmd.Run(); err == nil {
			killed++
		}
	}

	return killed
}

// restartOvermind restarts all overmind services atomically.
func restartOvermind() error {
	// Check if overmind is running
	statusCmd := exec.Command("overmind", "status")
	if err := statusCmd.Run(); err != nil {
		// Overmind not running, need to start it
		if deployVerbose {
			fmt.Println("  Overmind not running, starting...")
		}

		// Find project directory (where Procfile is)
		projectDir := findOrchProjectDir()
		if projectDir == "" {
			return fmt.Errorf("cannot find Procfile")
		}

		// Start overmind in daemon mode
		startCmd := exec.Command("overmind", "start", "-D")
		startCmd.Dir = projectDir
		startCmd.Env = os.Environ()

		if err := startCmd.Run(); err != nil {
			return fmt.Errorf("failed to start overmind: %w", err)
		}

		// Give it a moment to start
		time.Sleep(2 * time.Second)
		return nil
	}

	// Overmind is running, restart all services
	if deployVerbose {
		fmt.Println("  Restarting all overmind services...")
	}

	// Find project directory for restart
	projectDir := findOrchProjectDir()
	if projectDir == "" {
		return fmt.Errorf("cannot find Procfile")
	}

	restartCmd := exec.Command("overmind", "restart")
	restartCmd.Dir = projectDir
	restartCmd.Env = os.Environ()

	if deployVerbose {
		restartCmd.Stdout = os.Stdout
		restartCmd.Stderr = os.Stderr
	}

	if err := restartCmd.Run(); err != nil {
		return fmt.Errorf("failed to restart overmind: %w", err)
	}

	// Give services time to come up
	time.Sleep(2 * time.Second)

	return nil
}

// findOrchProjectDir finds the orch-go project directory (where Procfile is).
func findOrchProjectDir() string {
	// First try embedded source directory
	if sourceDir != "unknown" {
		if _, err := os.Stat(filepath.Join(sourceDir, "Procfile")); err == nil {
			return sourceDir
		}
	}

	// Try current directory
	cwd, err := os.Getwd()
	if err == nil {
		if _, err := os.Stat(filepath.Join(cwd, "Procfile")); err == nil {
			return cwd
		}
	}

	return ""
}

// waitForHealthChecks waits for all services to become healthy.
func waitForHealthChecks(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	pollInterval := 2 * time.Second

	// Services to check with their ports
	services := []struct {
		name string
		port int
	}{
		{"OpenCode", 4096},
		{"orch serve", DefaultServePort},
		{"Web UI", DefaultWebPort},
	}

	for time.Now().Before(deadline) {
		allHealthy := true

		for _, svc := range services {
			if !isPortResponding(svc.port) {
				allHealthy = false
				if deployVerbose {
					fmt.Printf("  Waiting for %s (port %d)...\n", svc.name, svc.port)
				}
				break
			}
		}

		if allHealthy {
			return nil
		}

		time.Sleep(pollInterval)
	}

	// Timeout - report which services are still down
	var failedServices []string
	for _, svc := range services {
		if !isPortResponding(svc.port) {
			failedServices = append(failedServices, fmt.Sprintf("%s (port %d)", svc.name, svc.port))
		}
	}

	return fmt.Errorf("timeout waiting for: %s", strings.Join(failedServices, ", "))
}

// isPortResponding checks if a port is accepting connections.
func isPortResponding(port int) bool {
	addr := fmt.Sprintf("localhost:%d", port)
	conn, err := net.DialTimeout("tcp", addr, 1*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
