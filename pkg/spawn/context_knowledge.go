package spawn

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/config"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
)

// GenerateServerContext creates the server context section for SPAWN_CONTEXT.md.
// Returns empty string if no servers are configured for the project.
func GenerateServerContext(projectDir string) string {
	cfg, err := config.Load(projectDir)
	if err != nil {
		return "" // No config or can't load - skip silently
	}

	if len(cfg.Servers) == 0 {
		return "" // No servers configured
	}

	// Get project name from directory
	projectName := filepath.Base(projectDir)
	sessionName := tmux.GetWorkersSessionName(projectName)

	// Check if servers are running
	running := tmux.SessionExists(sessionName)
	status := "stopped"
	if running {
		status = "running"
	}

	// Build server list
	var serverLines []string
	for service, port := range cfg.Servers {
		serverLines = append(serverLines, fmt.Sprintf("- **%s:** http://localhost:%d", service, port))
	}

	// Format the context section
	var sb strings.Builder
	sb.WriteString("## LOCAL SERVERS\n\n")
	sb.WriteString(fmt.Sprintf("**Project:** %s\n", projectName))
	sb.WriteString(fmt.Sprintf("**Status:** %s\n\n", status))
	sb.WriteString("**Ports:**\n")
	for _, line := range serverLines {
		sb.WriteString(line + "\n")
	}

	// Special case: orch-go uses orch-dashboard (includes OpenCode server),
	// not orch servers (which only manages project dev servers via tmuxinator).
	// See .kb/guides/server-management.md for the boundary explanation.
	if projectName == "orch-go" {
		sb.WriteString("\n**Quick commands:**\n")
		sb.WriteString("- Start servers: `orch-dashboard start` (from macOS terminal, not agent)\n")
		sb.WriteString("- Stop servers: `orch-dashboard stop`\n")
		sb.WriteString("- Restart servers: `orch-dashboard restart`\n")
		sb.WriteString("\n⚠️ **Note:** orch-go uses `orch-dashboard`, not `orch servers`. ")
		sb.WriteString("The dashboard script manages OpenCode + API + Web UI via overmind with orphan cleanup.\n")
		sb.WriteString("Agents cannot start/stop services (runs on macOS host, agents run in Linux sandbox).\n\n")
	} else {
		sb.WriteString("\n**Quick commands:**\n")
		sb.WriteString(fmt.Sprintf("- Start servers: `orch servers start %s`\n", projectName))
		sb.WriteString(fmt.Sprintf("- Stop servers: `orch servers stop %s`\n", projectName))
		sb.WriteString(fmt.Sprintf("- Open in browser: `orch servers open %s`\n", projectName))
		sb.WriteString("\n")
	}

	return sb.String()
}

// RegisteredProject represents a project registered with kb.
type RegisteredProject struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// GenerateRegisteredProjectsContext creates the registered projects section for orchestrator contexts.
// Returns empty string if kb projects list fails or returns no projects.
func GenerateRegisteredProjectsContext() string {
	projects, err := GetRegisteredProjects()
	if err != nil || len(projects) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Registered Projects\n\n")
	sb.WriteString("These projects are registered with `kb` for cross-project orchestration:\n\n")
	sb.WriteString("| Project | Path |\n")
	sb.WriteString("|---------|------|\n")
	for _, p := range projects {
		sb.WriteString(fmt.Sprintf("| %s | `%s` |\n", p.Name, p.Path))
	}
	sb.WriteString("\n**Usage:** `orch spawn --workdir <path> SKILL \"task\"`\n\n")

	return sb.String()
}

// GetRegisteredProjects fetches the list of registered projects from kb.
func GetRegisteredProjects() ([]RegisteredProject, error) {
	cmd := exec.Command("kb", "projects", "list", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("kb projects list failed: %w", err)
	}

	var projects []RegisteredProject
	if err := json.Unmarshal(output, &projects); err != nil {
		return nil, fmt.Errorf("failed to parse kb projects output: %w", err)
	}

	return projects, nil
}
