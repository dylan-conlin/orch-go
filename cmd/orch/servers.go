package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/port"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/spf13/cobra"
)

var serversCmd = &cobra.Command{
	Use:   "servers",
	Short: "Manage development servers across projects",
	Long: `Centralized server management across projects.
	
Commands:
  list     Show all projects with port allocations and running status
  start    Start servers via tmuxinator
  stop     Stop servers session
  attach   Attach to servers window
  open     Open servers in browser
  status   Show summary view

Examples:
  orch servers list                 # Show all projects
  orch servers start myproject      # Start servers via tmuxinator
  orch servers stop myproject       # Stop servers session
  orch servers attach myproject     # Attach to servers window
  orch servers open myproject       # Open in browser
  orch servers status               # Show summary`,
}

var serversListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all projects with port allocations and running status",
	Long: `List all projects with their port allocations and running status.

Shows which projects have port allocations, what ports are allocated,
and whether the servers are currently running in tmux.

Examples:
  orch servers list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServersList("")
	},
}

var serversStartCmd = &cobra.Command{
	Use:   "start [project]",
	Short: "Start servers via tmuxinator",
	Long: `Start development servers for a project using tmuxinator.

This runs 'tmuxinator start workers-{project}' which creates a tmux
session with the servers window.

Examples:
  orch servers start myproject`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServersStart(args[0])
	},
}

var serversStopCmd = &cobra.Command{
	Use:   "stop [project]",
	Short: "Stop servers for a project",
	Long: `Stop the servers tmux session for a project.

This kills the workers-{project} tmux session.

Examples:
  orch servers stop myproject`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServersStop(args[0])
	},
}

var serversAttachCmd = &cobra.Command{
	Use:   "attach [project]",
	Short: "Attach to servers window",
	Long: `Attach to the servers window for a project.

Switches to the servers window in the workers-{project} tmux session.

Examples:
  orch servers attach myproject`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServersAttach(args[0])
	},
}

var serversOpenCmd = &cobra.Command{
	Use:   "open [project]",
	Short: "Open servers in browser",
	Long: `Open the project's web server in the default browser.

Opens the web port (vite/dev server) in the browser.

Examples:
  orch servers open myproject`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServersOpen(args[0], "", false)
	},
}

var serversStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show servers status summary",
	Long: `Show a summary of server status across all projects.

Displays counts of running, allocated, and stopped servers.

Examples:
  orch servers status`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServersStatus("")
	},
}

func init() {
	rootCmd.AddCommand(serversCmd)
	serversCmd.AddCommand(serversListCmd)
	serversCmd.AddCommand(serversStartCmd)
	serversCmd.AddCommand(serversStopCmd)
	serversCmd.AddCommand(serversAttachCmd)
	serversCmd.AddCommand(serversOpenCmd)
	serversCmd.AddCommand(serversStatusCmd)
}

// ProjectServerInfo holds information about a project's servers.
type ProjectServerInfo struct {
	Project string
	Ports   []port.Allocation
	Running bool
	Session string // tmux session name
}

// runServersList lists all projects with their port allocations and running status.
func runServersList(registryPath string) error {
	// Load port registry
	if registryPath == "" {
		registryPath = port.DefaultPath()
	}

	reg, err := port.New(registryPath)
	if err != nil {
		return fmt.Errorf("failed to load port registry: %w", err)
	}

	allocs := reg.List()
	if len(allocs) == 0 {
		fmt.Println("No port allocations found")
		fmt.Println()
		fmt.Println("Use 'orch port allocate <project> <service> <purpose>' to allocate ports")
		return nil
	}

	// Group allocations by project
	projectMap := make(map[string][]port.Allocation)
	for _, alloc := range allocs {
		projectMap[alloc.Project] = append(projectMap[alloc.Project], alloc)
	}

	// Get list of running workers sessions
	runningSessions, err := tmux.ListWorkersSessions()
	if err != nil {
		// If tmux isn't available, show allocations without running status
		runningSessions = []string{}
	}

	// Build project info list
	var projects []ProjectServerInfo
	for projectName, projectPorts := range projectMap {
		sessionName := tmux.GetWorkersSessionName(projectName)
		running := false
		for _, sess := range runningSessions {
			if sess == sessionName {
				running = true
				break
			}
		}

		projects = append(projects, ProjectServerInfo{
			Project: projectName,
			Ports:   projectPorts,
			Running: running,
			Session: sessionName,
		})
	}

	// Sort projects by name
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Project < projects[j].Project
	})

	// Print header
	fmt.Printf("%-25s %-30s %-10s\n", "PROJECT", "PORTS", "STATUS")
	fmt.Printf("%s\n", strings.Repeat("-", 70))

	// Print each project
	for _, proj := range projects {
		// Format ports list
		var portStrs []string
		for _, p := range proj.Ports {
			portStrs = append(portStrs, fmt.Sprintf("%s:%d", p.Service, p.Port))
		}
		portsStr := strings.Join(portStrs, ", ")

		// Status
		status := "stopped"
		if proj.Running {
			status = "running"
		}

		fmt.Printf("%-25s %-30s %-10s\n", proj.Project, portsStr, status)
	}

	fmt.Println()
	fmt.Printf("Total: %d projects\n", len(projects))

	return nil
}

// runServersStart starts servers for a project via tmuxinator.
func runServersStart(project string) error {
	sessionName := tmux.GetWorkersSessionName(project)

	// Check if session already exists
	if tmux.SessionExists(sessionName) {
		return fmt.Errorf("session %s already exists (use 'orch servers attach %s' to connect)", sessionName, project)
	}

	// Check if tmuxinator config exists
	configPath := tmux.TmuxinatorConfigPath(project)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("tmuxinator config not found: %s\nUse 'orch port tmuxinator %s <project-dir>' to create", configPath, project)
	}

	// Start via tmuxinator
	fmt.Printf("Starting servers for %s...\n", project)
	cmd := exec.Command("tmuxinator", "start", sessionName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start tmuxinator: %w", err)
	}

	fmt.Printf("Servers started in session: %s\n", sessionName)
	fmt.Printf("Attach with: orch servers attach %s\n", project)

	return nil
}

// runServersStop stops servers for a project.
func runServersStop(project string) error {
	sessionName := tmux.GetWorkersSessionName(project)

	// Check if session exists
	if !tmux.SessionExists(sessionName) {
		return fmt.Errorf("session %s not found", sessionName)
	}

	// Kill the session
	fmt.Printf("Stopping servers for %s...\n", project)
	if err := tmux.KillSession(sessionName); err != nil {
		return fmt.Errorf("failed to stop session: %w", err)
	}

	fmt.Printf("Servers stopped: %s\n", sessionName)
	return nil
}

// runServersAttach attaches to the servers window for a project.
func runServersAttach(project string) error {
	sessionName := tmux.GetWorkersSessionName(project)

	// Check if session exists
	if !tmux.SessionExists(sessionName) {
		return fmt.Errorf("session %s not found\nStart servers with: orch servers start %s", sessionName, project)
	}

	// Attach to servers window (window 0 or window named "servers")
	// Try window name first, fall back to index 0
	windowTarget := fmt.Sprintf("%s:servers", sessionName)

	// Check if "servers" window exists, otherwise use index 0
	windows, err := tmux.ListWindows(sessionName)
	if err != nil {
		return fmt.Errorf("failed to list windows: %w", err)
	}

	found := false
	for _, w := range windows {
		if w.Name == "servers" {
			found = true
			break
		}
	}

	if !found && len(windows) > 0 {
		// Fall back to first window
		windowTarget = fmt.Sprintf("%s:0", sessionName)
	}

	// Attach
	return tmux.Attach(windowTarget)
}

// runServersOpen opens the project's web server in a browser.
func runServersOpen(project, registryPath string, dryRun bool) error {
	// Load port registry
	if registryPath == "" {
		registryPath = port.DefaultPath()
	}

	reg, err := port.New(registryPath)
	if err != nil {
		return fmt.Errorf("failed to load port registry: %w", err)
	}

	// Find web port for project
	allocs := reg.ListByProject(project)
	var webPort int
	for _, alloc := range allocs {
		if alloc.Purpose == port.PurposeVite {
			webPort = alloc.Port
			break
		}
	}

	if webPort == 0 {
		return fmt.Errorf("no web port found for project: %s", project)
	}

	url := fmt.Sprintf("http://localhost:%d", webPort)

	if dryRun {
		fmt.Printf("Would open: %s\n", url)
		return nil
	}

	// Open in browser
	fmt.Printf("Opening %s in browser...\n", url)

	var cmd *exec.Cmd
	switch {
	case exec.Command("which", "open").Run() == nil:
		// macOS
		cmd = exec.Command("open", url)
	case exec.Command("which", "xdg-open").Run() == nil:
		// Linux
		cmd = exec.Command("xdg-open", url)
	default:
		return fmt.Errorf("unable to detect browser opener (tried: open, xdg-open)")
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open browser: %w", err)
	}

	return nil
}

// runServersStatus shows a summary of server status.
func runServersStatus(registryPath string) error {
	// Load port registry
	if registryPath == "" {
		registryPath = port.DefaultPath()
	}

	reg, err := port.New(registryPath)
	if err != nil {
		return fmt.Errorf("failed to load port registry: %w", err)
	}

	allocs := reg.List()

	// Group by project
	projectMap := make(map[string][]port.Allocation)
	for _, alloc := range allocs {
		projectMap[alloc.Project] = append(projectMap[alloc.Project], alloc)
	}

	// Count running sessions
	runningSessions, err := tmux.ListWorkersSessions()
	if err != nil {
		runningSessions = []string{}
	}

	runningCount := 0
	for projectName := range projectMap {
		sessionName := tmux.GetWorkersSessionName(projectName)
		for _, sess := range runningSessions {
			if sess == sessionName {
				runningCount++
				break
			}
		}
	}

	totalProjects := len(projectMap)
	stoppedCount := totalProjects - runningCount

	// Print summary
	fmt.Println("Servers Status Summary")
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("Total projects:   %d\n", totalProjects)
	fmt.Printf("Running:          %d\n", runningCount)
	fmt.Printf("Stopped:          %d\n", stoppedCount)
	fmt.Println()
	fmt.Println("Use 'orch servers list' for detailed view")

	return nil
}
