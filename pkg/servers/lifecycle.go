// Package servers provides lifecycle management for per-project servers.
//
// This file contains functions for starting, stopping, and checking the status
// of servers defined in .orch/servers.yaml using launchd for native processes
// and Docker for containers.

package servers

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ServerStatus represents the running state of a server.
type ServerStatus string

const (
	StatusRunning ServerStatus = "running"
	StatusStopped ServerStatus = "stopped"
	StatusError   ServerStatus = "error"
	StatusUnknown ServerStatus = "unknown"
)

// ServerState holds the current state of a server.
type ServerState struct {
	Name    string
	Type    ServerType
	Port    int
	Status  ServerStatus
	Message string // Additional info (e.g., error message, PID)
}

// LifecycleResult holds the result of a lifecycle operation.
type LifecycleResult struct {
	Server  string
	Success bool
	Message string
}

// Up starts all servers for a project.
// Returns results for each server.
func Up(project, projectDir string) ([]LifecycleResult, error) {
	cfg, err := Load(projectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load servers.yaml: %w", err)
	}

	if len(cfg.Servers) == 0 {
		return nil, fmt.Errorf("no servers defined in %s", DefaultPath(projectDir))
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid servers config: %w", err)
	}

	var results []LifecycleResult

	// Start servers in dependency order (simple: just iterate for now)
	for _, server := range cfg.Servers {
		result := startServer(project, server, projectDir)
		results = append(results, result)
	}

	return results, nil
}

// Down stops all servers for a project.
// Returns results for each server.
func Down(project, projectDir string) ([]LifecycleResult, error) {
	cfg, err := Load(projectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load servers.yaml: %w", err)
	}

	if len(cfg.Servers) == 0 {
		return nil, fmt.Errorf("no servers defined in %s", DefaultPath(projectDir))
	}

	var results []LifecycleResult

	// Stop servers in reverse order (dependencies first should stop last)
	for i := len(cfg.Servers) - 1; i >= 0; i-- {
		server := cfg.Servers[i]
		result := stopServer(project, server, projectDir)
		results = append(results, result)
	}

	return results, nil
}

// Status returns the status of all servers for a project.
func Status(project, projectDir string) ([]ServerState, error) {
	cfg, err := Load(projectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load servers.yaml: %w", err)
	}

	var states []ServerState

	for _, server := range cfg.Servers {
		state := getServerStatus(project, server)
		states = append(states, state)
	}

	return states, nil
}

// startServer starts a single server based on its type.
func startServer(project string, server Server, projectDir string) LifecycleResult {
	switch server.Type {
	case TypeCommand:
		return startCommandServer(project, server, projectDir)
	case TypeDocker:
		return startDockerServer(project, server)
	case TypeLaunchd:
		return startLaunchdServer(server)
	default:
		return LifecycleResult{
			Server:  server.Name,
			Success: false,
			Message: fmt.Sprintf("unsupported server type: %s", server.Type),
		}
	}
}

// stopServer stops a single server based on its type.
func stopServer(project string, server Server, projectDir string) LifecycleResult {
	switch server.Type {
	case TypeCommand:
		return stopCommandServer(project, server)
	case TypeDocker:
		return stopDockerServer(project, server)
	case TypeLaunchd:
		return stopLaunchdServer(server)
	default:
		return LifecycleResult{
			Server:  server.Name,
			Success: false,
			Message: fmt.Sprintf("unsupported server type: %s", server.Type),
		}
	}
}

// getServerStatus checks the status of a single server.
func getServerStatus(project string, server Server) ServerState {
	switch server.Type {
	case TypeCommand:
		return getCommandServerStatus(project, server)
	case TypeDocker:
		return getDockerServerStatus(project, server)
	case TypeLaunchd:
		return getLaunchdServerStatus(server)
	default:
		return ServerState{
			Name:    server.Name,
			Type:    server.Type,
			Port:    server.Port,
			Status:  StatusUnknown,
			Message: fmt.Sprintf("unsupported server type: %s", server.Type),
		}
	}
}

// Command server lifecycle (uses launchd plist)

func startCommandServer(project string, server Server, projectDir string) LifecycleResult {
	label := getLaunchdLabel(project, server.Name)

	// Check if plist exists
	plistPath, err := PlistPath(project, server.Name)
	if err != nil {
		return LifecycleResult{
			Server:  server.Name,
			Success: false,
			Message: fmt.Sprintf("failed to get plist path: %v", err),
		}
	}

	// Generate plist if it doesn't exist
	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		opts := DefaultPlistOptions()
		plistConfig := ServerToPlistConfig(project, server, projectDir, opts)
		plistContent := GeneratePlist(plistConfig)

		if err := WritePlist(project, server.Name, plistContent); err != nil {
			return LifecycleResult{
				Server:  server.Name,
				Success: false,
				Message: fmt.Sprintf("failed to write plist: %v", err),
			}
		}
	}

	// Check if already running
	if isLaunchdServiceRunning(label) {
		return LifecycleResult{
			Server:  server.Name,
			Success: true,
			Message: "already running",
		}
	}

	// Bootstrap the service
	cmd := exec.Command("launchctl", "bootstrap", fmt.Sprintf("gui/%d", os.Getuid()), plistPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if it's already loaded
		if strings.Contains(string(output), "service already loaded") {
			// Try to kickstart it
			kickCmd := exec.Command("launchctl", "kickstart", fmt.Sprintf("gui/%d/%s", os.Getuid(), label))
			if kickErr := kickCmd.Run(); kickErr != nil {
				return LifecycleResult{
					Server:  server.Name,
					Success: false,
					Message: fmt.Sprintf("failed to kickstart: %v", kickErr),
				}
			}
			return LifecycleResult{
				Server:  server.Name,
				Success: true,
				Message: "started (kickstart)",
			}
		}
		return LifecycleResult{
			Server:  server.Name,
			Success: false,
			Message: fmt.Sprintf("failed to bootstrap: %v - %s", err, string(output)),
		}
	}

	return LifecycleResult{
		Server:  server.Name,
		Success: true,
		Message: "started",
	}
}

func stopCommandServer(project string, server Server) LifecycleResult {
	label := getLaunchdLabel(project, server.Name)

	// Check if running
	if !isLaunchdServiceRunning(label) {
		return LifecycleResult{
			Server:  server.Name,
			Success: true,
			Message: "already stopped",
		}
	}

	// Bootout the service
	plistPath, _ := PlistPath(project, server.Name)
	cmd := exec.Command("launchctl", "bootout", fmt.Sprintf("gui/%d/%s", os.Getuid(), label))
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Try alternate bootout with plist path
		altCmd := exec.Command("launchctl", "bootout", fmt.Sprintf("gui/%d", os.Getuid()), plistPath)
		if altErr := altCmd.Run(); altErr != nil {
			return LifecycleResult{
				Server:  server.Name,
				Success: false,
				Message: fmt.Sprintf("failed to stop: %v - %s", err, string(output)),
			}
		}
	}

	return LifecycleResult{
		Server:  server.Name,
		Success: true,
		Message: "stopped",
	}
}

func getCommandServerStatus(project string, server Server) ServerState {
	label := getLaunchdLabel(project, server.Name)

	if isLaunchdServiceRunning(label) {
		return ServerState{
			Name:    server.Name,
			Type:    server.Type,
			Port:    server.Port,
			Status:  StatusRunning,
			Message: label,
		}
	}

	return ServerState{
		Name:    server.Name,
		Type:    server.Type,
		Port:    server.Port,
		Status:  StatusStopped,
		Message: label,
	}
}

// Docker server lifecycle

func startDockerServer(project string, server Server) LifecycleResult {
	containerName := getDockerContainerName(project, server.Name)

	// Check if container exists
	checkCmd := exec.Command("docker", "ps", "-a", "--filter", fmt.Sprintf("name=%s", containerName), "--format", "{{.Names}}")
	output, _ := checkCmd.Output()

	if strings.TrimSpace(string(output)) == containerName {
		// Container exists, start it
		startCmd := exec.Command("docker", "start", containerName)
		if err := startCmd.Run(); err != nil {
			return LifecycleResult{
				Server:  server.Name,
				Success: false,
				Message: fmt.Sprintf("failed to start container: %v", err),
			}
		}
		return LifecycleResult{
			Server:  server.Name,
			Success: true,
			Message: "started (existing container)",
		}
	}

	// Create and start new container
	args := []string{"run", "-d", "--name", containerName, "-p", fmt.Sprintf("%d:%d", server.Port, server.Port)}

	// Add environment variables
	for k, v := range server.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	// Add restart policy
	args = append(args, "--restart", "unless-stopped")

	// Add image
	args = append(args, server.Image)

	runCmd := exec.Command("docker", args...)
	output, err := runCmd.CombinedOutput()
	if err != nil {
		return LifecycleResult{
			Server:  server.Name,
			Success: false,
			Message: fmt.Sprintf("failed to create container: %v - %s", err, string(output)),
		}
	}

	return LifecycleResult{
		Server:  server.Name,
		Success: true,
		Message: "started (new container)",
	}
}

func stopDockerServer(project string, server Server) LifecycleResult {
	containerName := getDockerContainerName(project, server.Name)

	// Stop container
	cmd := exec.Command("docker", "stop", containerName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "No such container") {
			return LifecycleResult{
				Server:  server.Name,
				Success: true,
				Message: "already stopped (no container)",
			}
		}
		return LifecycleResult{
			Server:  server.Name,
			Success: false,
			Message: fmt.Sprintf("failed to stop: %v - %s", err, string(output)),
		}
	}

	return LifecycleResult{
		Server:  server.Name,
		Success: true,
		Message: "stopped",
	}
}

func getDockerServerStatus(project string, server Server) ServerState {
	containerName := getDockerContainerName(project, server.Name)

	// Check if container is running
	cmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("name=%s", containerName), "--format", "{{.Status}}")
	output, err := cmd.Output()
	if err != nil {
		return ServerState{
			Name:    server.Name,
			Type:    server.Type,
			Port:    server.Port,
			Status:  StatusError,
			Message: fmt.Sprintf("docker error: %v", err),
		}
	}

	status := strings.TrimSpace(string(output))
	if status != "" && strings.HasPrefix(status, "Up") {
		return ServerState{
			Name:    server.Name,
			Type:    server.Type,
			Port:    server.Port,
			Status:  StatusRunning,
			Message: status,
		}
	}

	return ServerState{
		Name:    server.Name,
		Type:    server.Type,
		Port:    server.Port,
		Status:  StatusStopped,
		Message: containerName,
	}
}

// Launchd server lifecycle (for pre-existing launchd services)

func startLaunchdServer(server Server) LifecycleResult {
	label := server.LaunchdLabel

	if isLaunchdServiceRunning(label) {
		return LifecycleResult{
			Server:  server.Name,
			Success: true,
			Message: "already running",
		}
	}

	// Kickstart the service
	cmd := exec.Command("launchctl", "kickstart", fmt.Sprintf("gui/%d/%s", os.Getuid(), label))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return LifecycleResult{
			Server:  server.Name,
			Success: false,
			Message: fmt.Sprintf("failed to start: %v - %s", err, string(output)),
		}
	}

	return LifecycleResult{
		Server:  server.Name,
		Success: true,
		Message: "started",
	}
}

func stopLaunchdServer(server Server) LifecycleResult {
	label := server.LaunchdLabel

	if !isLaunchdServiceRunning(label) {
		return LifecycleResult{
			Server:  server.Name,
			Success: true,
			Message: "already stopped",
		}
	}

	// Kill the service
	cmd := exec.Command("launchctl", "kill", "SIGTERM", fmt.Sprintf("gui/%d/%s", os.Getuid(), label))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return LifecycleResult{
			Server:  server.Name,
			Success: false,
			Message: fmt.Sprintf("failed to stop: %v - %s", err, string(output)),
		}
	}

	return LifecycleResult{
		Server:  server.Name,
		Success: true,
		Message: "stopped",
	}
}

func getLaunchdServerStatus(server Server) ServerState {
	label := server.LaunchdLabel

	if isLaunchdServiceRunning(label) {
		return ServerState{
			Name:    server.Name,
			Type:    server.Type,
			Port:    server.Port,
			Status:  StatusRunning,
			Message: label,
		}
	}

	return ServerState{
		Name:    server.Name,
		Type:    server.Type,
		Port:    server.Port,
		Status:  StatusStopped,
		Message: label,
	}
}

// Helper functions

func getLaunchdLabel(project, serverName string) string {
	return fmt.Sprintf("com.%s.%s", project, serverName)
}

func getDockerContainerName(project, serverName string) string {
	return fmt.Sprintf("%s-%s", project, serverName)
}

func isLaunchdServiceRunning(label string) bool {
	cmd := exec.Command("launchctl", "print", fmt.Sprintf("gui/%d/%s", os.Getuid(), label))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	// Check if the service is actually running (not just loaded)
	return strings.Contains(string(output), "state = running")
}

// EnsureLogDir ensures the log directory exists for a project.
func EnsureLogDir(projectDir string) error {
	logDir := filepath.Join(projectDir, ".orch", "logs")
	return os.MkdirAll(logDir, 0755)
}
