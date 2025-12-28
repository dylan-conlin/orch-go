package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/port"
	"github.com/dylan-conlin/orch-go/pkg/servers"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/spf13/cobra"
)

var serversCmd = &cobra.Command{
	Use:   "servers",
	Short: "Manage development servers across projects",
	Long: `Centralized server management across projects.
	
Commands:
  up       Start servers via launchd/Docker (from servers.yaml)
  down     Stop servers via launchd/Docker
  list     Show all projects with port allocations and running status
  start    Start servers via tmuxinator (legacy)
  stop     Stop servers session (legacy)
  attach   Attach to servers window
  open     Open servers in browser
  status   Show server status

Examples:
  orch servers up myproject          # Start servers via launchd/Docker
  orch servers down myproject        # Stop servers
  orch servers list                  # Show all projects
  orch servers status                # Show summary
  orch servers status myproject      # Show per-server status`,
}

var serversUpCmd = &cobra.Command{
	Use:   "up <project>",
	Short: "Start servers via launchd/Docker",
	Long: `Start all servers for a project using launchd (for native processes)
or Docker (for containers).

Reads server definitions from .orch/servers.yaml and starts each server
based on its type:
  - command: Uses launchd with generated plist files
  - docker:  Starts Docker containers with restart policy
  - launchd: Uses existing launchd service

Examples:
  orch servers up myproject
  orch servers up myproject --project-dir /path/to/project`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectDir, _ := cmd.Flags().GetString("project-dir")
		return runServersUp(args[0], projectDir)
	},
}

var serversDownCmd = &cobra.Command{
	Use:   "down <project>",
	Short: "Stop servers via launchd/Docker",
	Long: `Stop all servers for a project.

Stops servers based on their type:
  - command: Uses launchctl bootout
  - docker:  Uses docker stop
  - launchd: Uses launchctl kill

Examples:
  orch servers down myproject
  orch servers down myproject --project-dir /path/to/project`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectDir, _ := cmd.Flags().GetString("project-dir")
		return runServersDown(args[0], projectDir)
	},
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
	Short: "Start servers via tmuxinator (legacy)",
	Long: `Start development servers for a project using tmuxinator.

This runs 'tmuxinator start workers-{project}' which creates a tmux
session with the servers window.

Note: Consider using 'orch servers up' for launchd-based server management.

Examples:
  orch servers start myproject`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServersStart(args[0])
	},
}

var serversStopCmd = &cobra.Command{
	Use:   "stop [project]",
	Short: "Stop servers for a project (legacy)",
	Long: `Stop the servers tmux session for a project.

This kills the workers-{project} tmux session.

Note: Consider using 'orch servers down' for launchd-based server management.

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
	Use:   "status [project]",
	Short: "Show servers status",
	Long: `Show server status.

Without a project argument, shows a summary of all projects.
With a project argument, shows per-server status from servers.yaml.

Examples:
  orch servers status               # Summary view
  orch servers status myproject     # Per-server status`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectDir, _ := cmd.Flags().GetString("project-dir")
		if len(args) == 0 {
			return runServersStatus("")
		}
		return runServersStatusProject(args[0], projectDir)
	},
}

var serversGenPlistCmd = &cobra.Command{
	Use:   "gen-plist <project>",
	Short: "Generate launchd plist files from servers.yaml",
	Long: `Generate launchd plist files for a project's servers.

Reads servers from .orch/servers.yaml and generates launchd plist files
at ~/Library/LaunchAgents/com.<project>.<server>.plist.

Only generates plists for servers with type: command.

Options:
  --path          Override PATH environment variable
  --keep-alive    Keep service running (restart on failure)
  --run-at-load   Start service at login
  --dry-run       Print plists without writing files
  --project-dir   Project directory (default: current directory)

Examples:
  orch servers gen-plist myproject
  orch servers gen-plist myproject --dry-run
  orch servers gen-plist myproject --keep-alive --run-at-load`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectDir, _ := cmd.Flags().GetString("project-dir")
		pathEnv, _ := cmd.Flags().GetString("path")
		keepAlive, _ := cmd.Flags().GetBool("keep-alive")
		runAtLoad, _ := cmd.Flags().GetBool("run-at-load")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		return runServersGenPlist(args[0], projectDir, pathEnv, keepAlive, runAtLoad, dryRun)
	},
}

func init() {
	rootCmd.AddCommand(serversCmd)
	serversCmd.AddCommand(serversUpCmd)
	serversCmd.AddCommand(serversDownCmd)
	serversCmd.AddCommand(serversListCmd)
	serversCmd.AddCommand(serversStartCmd)
	serversCmd.AddCommand(serversStopCmd)
	serversCmd.AddCommand(serversAttachCmd)
	serversCmd.AddCommand(serversOpenCmd)
	serversCmd.AddCommand(serversStatusCmd)
	serversCmd.AddCommand(serversGenPlistCmd)

	// up/down flags
	serversUpCmd.Flags().String("project-dir", "", "Project directory (default: current directory)")
	serversDownCmd.Flags().String("project-dir", "", "Project directory (default: current directory)")
	serversStatusCmd.Flags().String("project-dir", "", "Project directory (default: current directory)")

	// gen-plist flags
	serversGenPlistCmd.Flags().String("project-dir", "", "Project directory (default: current directory)")
	serversGenPlistCmd.Flags().String("path", "", "Override PATH environment variable")
	serversGenPlistCmd.Flags().Bool("keep-alive", true, "Keep service running (restart on failure)")
	serversGenPlistCmd.Flags().Bool("run-at-load", false, "Start service at login")
	serversGenPlistCmd.Flags().Bool("dry-run", false, "Print plists without writing files")
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

// runServersUp starts all servers for a project using launchd/Docker.
func runServersUp(project, projectDir string) error {
	// Default to current directory if not specified
	if projectDir == "" {
		var err error
		projectDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Convert to absolute path
	absProjectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return fmt.Errorf("failed to resolve project directory: %w", err)
	}

	// Ensure log directory exists
	if err := servers.EnsureLogDir(absProjectDir); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	fmt.Printf("Starting servers for %s...\n", project)
	fmt.Println()

	results, err := servers.Up(project, absProjectDir)
	if err != nil {
		return err
	}

	// Print results
	hasError := false
	for _, r := range results {
		if r.Success {
			fmt.Printf("  ✓ %s: %s\n", r.Server, r.Message)
		} else {
			fmt.Printf("  ✗ %s: %s\n", r.Server, r.Message)
			hasError = true
		}
	}

	fmt.Println()
	if hasError {
		fmt.Println("Some servers failed to start")
		return fmt.Errorf("not all servers started successfully")
	}

	fmt.Printf("All servers started for %s\n", project)
	fmt.Printf("Check status: orch servers status %s\n", project)
	return nil
}

// runServersDown stops all servers for a project.
func runServersDown(project, projectDir string) error {
	// Default to current directory if not specified
	if projectDir == "" {
		var err error
		projectDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Convert to absolute path
	absProjectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return fmt.Errorf("failed to resolve project directory: %w", err)
	}

	fmt.Printf("Stopping servers for %s...\n", project)
	fmt.Println()

	results, err := servers.Down(project, absProjectDir)
	if err != nil {
		return err
	}

	// Print results
	hasError := false
	for _, r := range results {
		if r.Success {
			fmt.Printf("  ✓ %s: %s\n", r.Server, r.Message)
		} else {
			fmt.Printf("  ✗ %s: %s\n", r.Server, r.Message)
			hasError = true
		}
	}

	fmt.Println()
	if hasError {
		fmt.Println("Some servers failed to stop")
		return fmt.Errorf("not all servers stopped successfully")
	}

	fmt.Printf("All servers stopped for %s\n", project)
	return nil
}

// runServersStatusProject shows per-server status for a project.
func runServersStatusProject(project, projectDir string) error {
	// Default to current directory if not specified
	if projectDir == "" {
		var err error
		projectDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Convert to absolute path
	absProjectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return fmt.Errorf("failed to resolve project directory: %w", err)
	}

	states, err := servers.Status(project, absProjectDir)
	if err != nil {
		return err
	}

	if len(states) == 0 {
		fmt.Printf("No servers defined in %s\n", servers.DefaultPath(absProjectDir))
		return nil
	}

	fmt.Printf("Servers Status: %s\n", project)
	fmt.Printf("%s\n", strings.Repeat("-", 60))
	fmt.Printf("%-15s %-10s %-8s %-10s %s\n", "NAME", "TYPE", "PORT", "STATUS", "INFO")
	fmt.Printf("%s\n", strings.Repeat("-", 60))

	runningCount := 0
	for _, s := range states {
		statusIcon := "○"
		if s.Status == servers.StatusRunning {
			statusIcon = "●"
			runningCount++
		} else if s.Status == servers.StatusError {
			statusIcon = "✗"
		}

		fmt.Printf("%-15s %-10s %-8d %s %-8s %s\n",
			s.Name,
			s.Type,
			s.Port,
			statusIcon,
			s.Status,
			s.Message,
		)
	}

	fmt.Println()
	fmt.Printf("Running: %d/%d\n", runningCount, len(states))

	return nil
}

// runServersGenPlist generates launchd plist files for a project's servers.
func runServersGenPlist(project, projectDir, pathEnv string, keepAlive, runAtLoad, dryRun bool) error {
	// Default to current directory if not specified
	if projectDir == "" {
		var err error
		projectDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Convert to absolute path
	absProjectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return fmt.Errorf("failed to resolve project directory: %w", err)
	}

	// Load servers.yaml
	cfg, err := servers.Load(absProjectDir)
	if err != nil {
		return fmt.Errorf("failed to load servers.yaml: %w", err)
	}

	if len(cfg.Servers) == 0 {
		return fmt.Errorf("no servers found in %s", servers.DefaultPath(absProjectDir))
	}

	// Build options
	opts := servers.DefaultPlistOptions()
	if pathEnv != "" {
		opts.Path = pathEnv
	}
	opts.KeepAlive = keepAlive
	opts.RunAtLoad = runAtLoad

	// Filter to command-type servers only
	var commandServers []servers.Server
	for _, s := range cfg.Servers {
		if s.Type == servers.TypeCommand {
			commandServers = append(commandServers, s)
		}
	}

	if len(commandServers) == 0 {
		return fmt.Errorf("no command-type servers found in servers.yaml")
	}

	// Generate plists
	for _, s := range commandServers {
		plistCfg := servers.ServerToPlistConfig(project, s, absProjectDir, opts)
		content := servers.GeneratePlist(plistCfg)

		if dryRun {
			plistPath, _ := servers.PlistPath(project, s.Name)
			fmt.Printf("=== %s ===\n", plistPath)
			fmt.Println(content)
		} else {
			if err := servers.WritePlist(project, s.Name, content); err != nil {
				return fmt.Errorf("failed to write plist for %s: %w", s.Name, err)
			}
			plistPath, _ := servers.PlistPath(project, s.Name)
			fmt.Printf("Generated: %s\n", plistPath)
		}
	}

	if !dryRun {
		fmt.Println()
		fmt.Println("To load services:")
		for _, s := range commandServers {
			plistPath, _ := servers.PlistPath(project, s.Name)
			fmt.Printf("  launchctl load %s\n", plistPath)
		}
		fmt.Println()
		fmt.Println("To unload services:")
		for _, s := range commandServers {
			plistPath, _ := servers.PlistPath(project, s.Name)
			fmt.Printf("  launchctl unload %s\n", plistPath)
		}
	}

	return nil
}
