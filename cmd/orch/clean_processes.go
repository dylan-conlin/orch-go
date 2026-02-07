// Package main provides orphan process cleanup for the clean command.
// Extracted from clean_cmd.go for per-concern file organization.
package main

import (
	"fmt"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/process"
)

// cleanOrphanProcesses finds and kills bun agent processes that are not associated
// with any active OpenCode session. Returns the number of processes killed.
func cleanOrphanProcesses(serverURL string, dryRun bool) (int, error) {
	return cleanOrphanProcessesWithClient(opencode.NewClient(serverURL), dryRun)
}

func cleanOrphanProcessesWithClient(client opencode.ClientInterface, dryRun bool) (int, error) {
	fmt.Println("\nScanning for orphan bun processes...")

	sessions, err := client.ListSessions("")
	if err != nil {
		return 0, fmt.Errorf("failed to list sessions: %w", err)
	}

	activeTitles := make(map[string]bool)
	for _, s := range sessions {
		title := s.Title
		if title == "" {
			continue
		}
		activeTitles[title] = true
		if idx := strings.Index(title, " ["); idx != -1 {
			activeTitles[strings.TrimSpace(title[:idx])] = true
		}
	}

	fmt.Printf("  Found %d active OpenCode sessions\n", len(sessions))

	orphans, err := process.FindOrphanProcesses(activeTitles)
	if err != nil {
		return 0, fmt.Errorf("failed to find orphan processes: %w", err)
	}

	if len(orphans) == 0 {
		fmt.Println("  No orphan bun processes found")
		return 0, nil
	}

	fmt.Printf("  Found %d orphan bun processes:\n", len(orphans))

	killed := 0
	for _, orphan := range orphans {
		name := orphan.WorkspaceName
		if name == "" {
			name = "(unknown)"
		}
		beadsInfo := ""
		if orphan.BeadsID != "" {
			beadsInfo = fmt.Sprintf(" [%s]", orphan.BeadsID)
		}

		if dryRun {
			fmt.Printf("    [DRY-RUN] Would kill: PID %d (%s%s)\n", orphan.PID, name, beadsInfo)
			killed++
			continue
		}

		if process.Terminate(orphan.PID, "bun (orphan)") {
			fmt.Printf("    Killed: PID %d (%s%s)\n", orphan.PID, name, beadsInfo)
			killed++
		} else {
			fmt.Printf("    Already dead: PID %d (%s%s)\n", orphan.PID, name, beadsInfo)
		}
	}

	return killed, nil
}
