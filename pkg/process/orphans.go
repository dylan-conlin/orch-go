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
	PPID          int    // Parent process ID
	Command       string // Full command line
	WorkspaceName string // Extracted workspace name from --title arg (old run --attach format)
	BeadsID       string // Extracted beads ID from [beads-id] in title (old run --attach format)
	SessionID     string // Extracted session ID from --session arg (new attach format)
}

// isOpenCodeProcess checks if a ps output line represents any OpenCode process
// (agent, server, or TUI). Matches on --conditions=browser + src/index.ts.
func isOpenCodeProcess(line string) bool {
	return strings.Contains(line, "--conditions=browser") && strings.Contains(line, "src/index.ts")
}

// isOpenCodeServer checks if a ps output line is the OpenCode server process.
func isOpenCodeServer(line string) bool {
	return isOpenCodeProcess(line) && strings.Contains(line, "serve --port")
}

// isReapableAgent determines if an OpenCode process is a reapable agent
// (as opposed to the TUI or server). Uses three signals:
//
//  1. Has "attach" in cmdline → explicitly an agent process (attach mode)
//  2. PPID == serverPID → headless agent spawned by the server
//  3. PPID == 1 → orphan whose parent (server) already died
//
// The TUI has none of these: no "attach", PPID is the user's shell, not the server.
func isReapableAgent(line string, ppid, serverPID int) bool {
	// Attach-mode agents always have "attach" in their command line
	if strings.Contains(line, "attach") {
		return true
	}
	// Headless agents are direct children of the OpenCode server
	if serverPID > 0 && ppid == serverPID {
		return true
	}
	// Orphans whose parent died get reparented to PID 1 (init/launchd)
	if ppid == 1 {
		return true
	}
	return false
}

// FindAgentProcesses discovers reapable OpenCode agent processes.
//
// Uses a two-pass approach:
//  1. Find the OpenCode server PID (if running)
//  2. Identify agent processes using PPID-based classification
//
// A process is a reapable agent if it matches the OpenCode pattern AND:
//   - Has "attach" in cmdline (attach-mode agent), OR
//   - Is a child of the OpenCode server (headless agent), OR
//   - Has PPID 1 (orphan — server already died, reparented to init/launchd)
//
// This correctly excludes the TUI, which is a child of the user's shell.
func FindAgentProcesses() ([]OrphanProcess, error) {
	// Include PPID in output for parent-based classification
	cmd := exec.Command("ps", "-eo", "pid,ppid,args")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run ps: %w", err)
	}

	lines := strings.Split(string(output), "\n")

	// Pass 1: Find the OpenCode server PID
	serverPID := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if isOpenCodeServer(line) {
			fields := strings.Fields(line)
			if len(fields) >= 1 {
				if pid, err := strconv.Atoi(fields[0]); err == nil {
					serverPID = pid
				}
			}
			break
		}
	}

	// Pass 2: Find reapable agent processes
	var agents []OrphanProcess
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if !isOpenCodeProcess(line) || isOpenCodeServer(line) {
			continue
		}

		// Parse PID and PPID (first two fields)
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		ppid, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}

		fullCmd := strings.Join(fields[2:], " ")

		if !isReapableAgent(fullCmd, ppid, serverPID) {
			continue
		}

		agent := OrphanProcess{
			PID:     pid,
			PPID:    ppid,
			Command: fullCmd,
		}

		// Extract workspace name from --title argument (old run --attach format)
		if titleIdx := strings.Index(fullCmd, "--title "); titleIdx != -1 {
			rest := fullCmd[titleIdx+len("--title "):]
			parts := strings.Fields(rest)
			if len(parts) > 0 {
				agent.WorkspaceName = parts[0]
			}
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
