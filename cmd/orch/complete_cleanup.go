// Package main provides cleanup-stage operations for completion.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/process"
	"github.com/dylan-conlin/orch-go/pkg/session"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
)

// runCleanup handles session deletion, activity export, archival, docker, and tmux cleanup.
func runCleanup(target *CompletionTarget) *CleanupOutcome {
	outcome := &CleanupOutcome{}

	if target.WorkspacePath == "" {
		return outcome
	}

	// Export activity (before session deletion)
	if !target.IsOrchestratorSession {
		exportActivity(target)
	}

	// Delete OpenCode session and terminate process
	outcome.SessionDeleted, outcome.ProcessTerminated = deleteSessionAndProcess(target)

	// Remove from process ownership ledger
	removeLedgerEntry(target)

	// Export orchestrator transcript (needs tmux window alive, before tmux kill)
	if target.IsOrchestratorSession {
		if err := exportOrchestratorTranscript(target.WorkspacePath, target.BeadsProjectDir, target.AgentName); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to export orchestrator transcript: %v\n", err)
		} else {
			outcome.TranscriptExported = true
		}
	}

	// Clean up Docker container (before tmux kill to avoid orphaning container)
	containerName := spawn.ReadContainerID(target.WorkspacePath)
	if containerName != "" {
		if err := spawn.CleanupDockerContainer(containerName); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to clean up Docker container %s: %v\n", containerName, err)
		} else {
			outcome.DockerCleaned = true
			fmt.Printf("Cleaned up Docker container: %s\n", containerName)
		}
	}

	// Kill tmux window — sends SIGHUP to terminate the bun process.
	// This is the primary termination path for tmux-spawned agents since
	// OpenCode Session.remove() doesn't kill attached bun processes and
	// the process ledger is never populated for tmux spawns.
	// Must run before archival to ensure the bun process dies even if
	// later steps fail.
	outcome.TmuxWindowClosed = cleanupTmuxWindow(target)

	// Archive workspace (after all resource cleanup)
	if !completeNoArchive {
		archivedPath, err := archiveWorkspace(target.WorkspacePath, target.BeadsProjectDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to archive workspace: %v\n", err)
		} else {
			outcome.ArchivedPath = archivedPath
			fmt.Printf("Archived workspace: %s\n", filepath.Base(archivedPath))

			if target.IsOrchestratorSession && archivedPath != "" {
				registry := session.NewRegistry("")
				if err := registry.Update(target.AgentName, func(s *session.OrchestratorSession) {
					s.ArchivedPath = archivedPath
				}); err != nil {
					if err != session.ErrSessionNotFound {
						fmt.Fprintf(os.Stderr, "Warning: failed to update archived path in registry: %v\n", err)
					}
				}
			}
		}
	} else {
		fmt.Println("Skipped workspace archival (--no-archive)")
	}

	gitCleanup := cleanupManagedGitIsolation(target.WorkspacePath, target.sourceDir())
	outcome.GitWorktreeRemoved = gitCleanup.WorktreeRemoved
	outcome.GitBranchDeleted = gitCleanup.BranchDeleted

	return outcome
}

// deleteSessionAndProcess deletes the OpenCode session and terminates the process.
// Returns (sessionDeleted, processTerminated).
func deleteSessionAndProcess(target *CompletionTarget) (bool, bool) {
	return deleteSessionAndProcessWithClient(opencode.NewClient(serverURL), target)
}

func deleteSessionAndProcessWithClient(client opencode.ClientInterface, target *CompletionTarget) (bool, bool) {
	var sessionDeleted, processTerminated bool

	sessionFile := filepath.Join(target.WorkspacePath, ".session_id")
	data, err := os.ReadFile(sessionFile)
	if err == nil {
		sessionID := strings.TrimSpace(string(data))
		if sessionID != "" {
			if err := client.DeleteSession(sessionID); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to delete OpenCode session %s: %v\n", sessionID[:12], err)
			} else {
				sessionDeleted = true
				fmt.Printf("Deleted OpenCode session: %s\n", sessionID[:12])
			}
		}
	}

	// Try workspace .process_id first (written for non-headless spawns)
	pid := spawn.ReadProcessID(target.WorkspacePath)
	if pid > 0 {
		if process.Terminate(pid, "opencode") {
			processTerminated = true
		}
	}

	// For headless spawns, .process_id is never written. After deleting the session,
	// sweep for orphaned bun processes that are no longer associated with any active
	// session. This catches headless agent processes that would otherwise leak.
	if !processTerminated && sessionDeleted {
		killed := sweepOrphanedProcessesAfterSessionDelete(client)
		if killed > 0 {
			processTerminated = true
		}
	}

	return sessionDeleted, processTerminated
}

// sweepOrphanedProcessesAfterSessionDelete finds and kills bun agent processes
// that are no longer associated with any active OpenCode session. This is called
// after deleting a session to catch headless processes that have no .process_id file.
func sweepOrphanedProcessesAfterSessionDelete(client opencode.ClientInterface) int {
	sessions, err := client.ListSessions("")
	if err != nil {
		return 0
	}

	activeIDs := make(map[string]bool, len(sessions))
	activeTitles := make(map[string]bool)
	for _, s := range sessions {
		if s.ID != "" {
			activeIDs[s.ID] = true
		}
		if s.Title != "" {
			activeTitles[s.Title] = true
			if idx := strings.Index(s.Title, " ["); idx != -1 {
				activeTitles[strings.TrimSpace(s.Title[:idx])] = true
			}
		}
	}

	orphans, err := process.FindOrphanProcesses(activeTitles, activeIDs)
	if err != nil {
		return 0
	}

	killed := 0
	for _, orphan := range orphans {
		if process.Terminate(orphan.PID, "bun (orphan after session delete)") {
			killed++
			fmt.Printf("Terminated orphaned bun process: PID %d\n", orphan.PID)
		}
	}
	return killed
}

// cleanupTmuxWindow finds and kills the tmux window for the agent.
func cleanupTmuxWindow(target *CompletionTarget) bool {
	var window *tmux.WindowInfo
	var tmuxSessionName string
	var findErr error

	if target.IsOrchestratorSession {
		window, tmuxSessionName, findErr = tmux.FindWindowByWorkspaceNameAllSessions(target.AgentName)
	} else {
		windowSearchID := target.BeadsID
		if windowSearchID == "" {
			windowSearchID = target.Identifier
		}
		window, tmuxSessionName, findErr = tmux.FindWindowByBeadsIDAllSessions(windowSearchID)
	}

	if findErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to find tmux window for cleanup: %v\n", findErr)
		return false
	}
	if window == nil {
		return false
	}

	if err := tmux.KillWindow(window.Target); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to close tmux window %s: %v\n", window.Target, err)
		return false
	}

	fmt.Printf("Closed tmux window: %s:%s\n", tmuxSessionName, window.Name)
	return true
}

// removeLedgerEntry removes the process ledger entry for the completed agent.
// Uses workspace name as primary key, falls back to beads ID.
func removeLedgerEntry(target *CompletionTarget) {
	ledger := process.NewDefaultLedger()
	agentName := target.AgentName
	if agentName == "" {
		agentName = filepath.Base(target.WorkspacePath)
	}
	if agentName != "" {
		if err := ledger.RemoveByWorkspace(agentName); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove process ledger entry by workspace: %v\n", err)
		}
		return
	}
	if target.BeadsID != "" {
		if err := ledger.RemoveByBeadsID(target.BeadsID); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove process ledger entry by beads ID: %v\n", err)
		}
	}
}
