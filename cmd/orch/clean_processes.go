// Package main provides orphan process cleanup for the clean command.
// Extracted from clean_cmd.go for per-concern file organization.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/process"
)

// cleanOrphanProcesses finds and kills bun agent processes that are not associated
// with any active OpenCode session. Returns the number of processes killed.
func cleanOrphanProcesses(serverURL, projectDir string, dryRun bool) (int, error) {
	return cleanOrphanProcessesWithClient(opencode.NewClient(serverURL), projectDir, dryRun)
}

func cleanOrphanProcessesWithClient(client opencode.ClientInterface, projectDir string, dryRun bool) (int, error) {
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

	dashboardWebZombies, trackedWebPID, err := findDashboardWebBunZombies(projectDir)
	if err != nil {
		return 0, fmt.Errorf("failed to find dashboard web bun zombies: %w", err)
	}

	if len(orphans) == 0 && len(dashboardWebZombies) == 0 {
		fmt.Println("  No orphan bun processes found")
		return 0, nil
	}

	totalOrphans := len(orphans) + len(dashboardWebZombies)
	fmt.Printf("  Found %d orphan bun processes:\n", totalOrphans)

	killed := 0

	if len(orphans) > 0 {
		fmt.Printf("    Agent processes: %d\n", len(orphans))
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
	}

	if len(dashboardWebZombies) > 0 {
		if trackedWebPID > 0 {
			fmt.Printf("    Dashboard web bun zombies: %d (tracked PID %d kept)\n", len(dashboardWebZombies), trackedWebPID)
		} else {
			fmt.Printf("    Dashboard web bun zombies: %d\n", len(dashboardWebZombies))
		}
		for _, zombie := range dashboardWebZombies {
			if dryRun {
				fmt.Printf("    [DRY-RUN] Would kill: PID %d (dashboard web bun)\n", zombie.PID)
				killed++
				continue
			}

			if process.Terminate(zombie.PID, "bun (dashboard-web orphan)") {
				fmt.Printf("    Killed: PID %d (dashboard web bun)\n", zombie.PID)
				killed++
			} else {
				fmt.Printf("    Already dead: PID %d (dashboard web bun)\n", zombie.PID)
			}
		}
	}

	return killed, nil
}

func findDashboardWebBunZombies(projectDir string) ([]process.BunProcess, int, error) {
	if projectDir == "" {
		return nil, 0, nil
	}

	webDir := filepath.Join(projectDir, "web")
	if info, err := os.Stat(webDir); err != nil || !info.IsDir() {
		return nil, 0, nil
	}

	trackedPID, err := readTrackedDashboardWebPID(projectDir)
	if err != nil {
		return nil, 0, err
	}

	allWebBunProcesses, err := process.FindBunProcessesInDirectory(webDir)
	if err != nil {
		return nil, 0, err
	}

	zombies := make([]process.BunProcess, 0, len(allWebBunProcesses))
	for _, webProcess := range allWebBunProcesses {
		if trackedPID > 0 && webProcess.PID == trackedPID {
			continue
		}
		zombies = append(zombies, webProcess)
	}

	return zombies, trackedPID, nil
}

func readTrackedDashboardWebPID(projectDir string) (int, error) {
	pidFile := filepath.Join(projectDir, ".orch", "run", "dashboard-web.pid")
	content, err := os.ReadFile(pidFile)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to read %s: %w", pidFile, err)
	}

	pidText := strings.TrimSpace(string(content))
	if pidText == "" {
		return 0, nil
	}

	pid, err := strconv.Atoi(pidText)
	if err != nil {
		return 0, nil
	}

	return pid, nil
}
