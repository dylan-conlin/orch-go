// Package process provides utilities for managing OS processes.
package process

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// OrphanProcess represents a bun process that may be orphaned.
type OrphanProcess struct {
	PID           int
	Command       string // Full command line
	WorkspaceName string // Extracted workspace name from --title arg
	BeadsID       string // Extracted beads ID from [beads-id] in title
}

// FindAgentProcesses discovers all bun processes that are spawned agents.
// These are identified by having "run --attach" in their command line (spawned via opencode run).
// Returns all agent processes regardless of whether they are orphaned or not.
// The caller is responsible for determining which are orphans by cross-referencing
// with active OpenCode sessions.
func FindAgentProcesses() ([]OrphanProcess, error) {
	// Use ps to list all bun processes with their full command line
	// -e: all processes, -o: custom output format
	cmd := exec.Command("ps", "-eo", "pid,args")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run ps: %w", err)
	}

	var agents []OrphanProcess

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Look for bun processes that are spawned agents (run --attach pattern)
		// These look like:
		//   PID bun run --conditions=browser ./src/index.ts run --attach http://... --title <workspace> [beads-id] ...
		if !strings.Contains(line, "bun") || !strings.Contains(line, "run --attach") {
			continue
		}

		// Parse PID (first field)
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}

		fullCmd := strings.Join(fields[1:], " ")

		agent := OrphanProcess{
			PID:     pid,
			Command: fullCmd,
		}

		// Extract workspace name from --title argument
		if titleIdx := strings.Index(fullCmd, "--title "); titleIdx != -1 {
			rest := fullCmd[titleIdx+len("--title "):]
			// Title is followed by either [beads-id] or another flag
			parts := strings.Fields(rest)
			if len(parts) > 0 {
				agent.WorkspaceName = parts[0]
			}
			// Extract beads ID from [beads-id] bracket notation
			if bracketIdx := strings.Index(rest, "["); bracketIdx != -1 {
				if endIdx := strings.Index(rest[bracketIdx:], "]"); endIdx != -1 {
					agent.BeadsID = rest[bracketIdx+1 : bracketIdx+endIdx]
				}
			}
		}

		agents = append(agents, agent)
	}

	return agents, nil
}

// FindOrphanProcesses discovers bun agent processes that are not associated with
// any active OpenCode session. It takes a set of active session titles as input
// and returns processes whose workspace names don't match any active session.
func FindOrphanProcesses(activeSessionTitles map[string]bool) ([]OrphanProcess, error) {
	allAgents, err := FindAgentProcesses()
	if err != nil {
		return nil, err
	}

	var orphans []OrphanProcess
	for _, agent := range allAgents {
		// Check if this agent's workspace is in active sessions
		if agent.WorkspaceName != "" && activeSessionTitles[agent.WorkspaceName] {
			continue // Still active
		}
		// Also check by beads ID
		if agent.BeadsID != "" && activeSessionTitles[agent.BeadsID] {
			continue // Still active
		}
		orphans = append(orphans, agent)
	}

	return orphans, nil
}
