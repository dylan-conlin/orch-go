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
	WorkspaceName string // Extracted workspace name from --title arg (old run --attach format)
	BeadsID       string // Extracted beads ID from [beads-id] in title (old run --attach format)
	SessionID     string // Extracted session ID from --session arg (new attach format)
}

// FindAgentProcesses discovers all bun processes that are OpenCode agent processes.
// These are identified by having "src/index.ts" in their command line (the OpenCode
// entrypoint), while excluding the OpenCode server process (which has "serve --port").
// Covers both old format (opencode run --attach) and new format (opencode attach).
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

		// Look for bun processes running OpenCode (src/index.ts entrypoint).
		// Covers both old format (run --attach) and new format (opencode attach).
		// Old: bun run --conditions=browser ./src/index.ts run --attach http://... --title <workspace> [beads-id]
		// New: bun run --conditions=browser ./src/index.ts attach http://... --dir /path --session <id>
		if !strings.Contains(line, "bun") || !strings.Contains(line, "src/index.ts") {
			continue
		}
		// Exclude the OpenCode server process — it also runs via bun + src/index.ts
		// but has "serve --port" in its args.
		if strings.Contains(line, "serve --port") {
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

		// Extract workspace name from --title argument (old run --attach format)
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

		// Extract session ID from --session argument (new attach format)
		if sessionIdx := strings.Index(fullCmd, "--session "); sessionIdx != -1 {
			rest := fullCmd[sessionIdx+len("--session "):]
			parts := strings.Fields(rest)
			if len(parts) > 0 {
				agent.SessionID = parts[0]
			}
		}

		agents = append(agents, agent)
	}

	return agents, nil
}

// FindOrphanProcesses discovers bun agent processes that are not associated with
// any active OpenCode session. It takes active session titles and IDs as input
// and returns processes that don't match any active session by title, beads ID,
// or session ID.
func FindOrphanProcesses(activeSessionTitles map[string]bool, activeSessionIDs map[string]bool) ([]OrphanProcess, error) {
	allAgents, err := FindAgentProcesses()
	if err != nil {
		return nil, err
	}

	var orphans []OrphanProcess
	for _, agent := range allAgents {
		// Check if this agent's workspace is in active sessions (old format)
		if agent.WorkspaceName != "" && activeSessionTitles[agent.WorkspaceName] {
			continue // Still active
		}
		// Check by beads ID (old format)
		if agent.BeadsID != "" && activeSessionTitles[agent.BeadsID] {
			continue // Still active
		}
		// Check by session ID (new attach format)
		if agent.SessionID != "" && activeSessionIDs[agent.SessionID] {
			continue // Still active
		}
		orphans = append(orphans, agent)
	}

	return orphans, nil
}
