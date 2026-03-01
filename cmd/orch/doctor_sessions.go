package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// SessionsCrossReferenceReport contains the results of workspace/session cross-reference.
type SessionsCrossReferenceReport struct {
	WorkspaceCount       int `json:"workspace_count"`
	SessionCount         int `json:"session_count"`
	OrphanedWorkspaces   int `json:"orphaned_workspaces"` // Workspaces with deleted sessions
	OrphanedSessions     int `json:"orphaned_sessions"`   // Sessions without workspaces
	ZombieSessions       int `json:"zombie_sessions"`     // Sessions active but stuck
	OrphanedWorkspaceIDs []string
	OrphanedSessionIDs   []string
	ZombieSessionIDs     []string
}

// runSessionsCrossReference performs a cross-reference between workspaces and OpenCode sessions
// to detect orphaned workspaces, orphaned sessions, and zombies.
func runSessionsCrossReference() error {
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	client := opencode.NewClient(serverURL)
	report := &SessionsCrossReferenceReport{}

	// Step 1: Build map of workspace → session IDs
	workspaceDir := filepath.Join(projectDir, ".orch", "workspace")
	workspaceToSession := make(map[string]string) // workspace name → session ID
	sessionToWorkspace := make(map[string]string) // session ID → workspace name
	workspaceBeadsID := make(map[string]string)   // workspace name → beads ID

	entries, err := os.ReadDir(workspaceDir)
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() || entry.Name() == "archived" {
				continue
			}
			wsPath := filepath.Join(workspaceDir, entry.Name())

			// Read session ID (.session_id stays separate - infrastructure handle)
			sessionID := spawn.ReadSessionID(wsPath)
			if sessionID != "" {
				workspaceToSession[entry.Name()] = sessionID
				sessionToWorkspace[sessionID] = entry.Name()
			}

			// Read beads ID from manifest (falls back to dotfiles)
			manifest := spawn.ReadAgentManifestWithFallback(wsPath)
			if manifest.BeadsID != "" {
				workspaceBeadsID[entry.Name()] = manifest.BeadsID
			}
		}
	}
	report.WorkspaceCount = len(workspaceToSession)

	// Step 2: Get all OpenCode sessions for this project
	sessions, err := client.ListDiskSessions(projectDir)
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}
	report.SessionCount = len(sessions)

	// Build session ID set and map for quick lookup
	sessionIDSet := make(map[string]bool)
	sessionByID := make(map[string]opencode.Session)
	for _, s := range sessions {
		sessionIDSet[s.ID] = true
		sessionByID[s.ID] = s
	}

	// Step 3: Find orphaned workspaces (workspace has session ID that doesn't exist in OpenCode)
	for name, sessionID := range workspaceToSession {
		if !sessionIDSet[sessionID] {
			report.OrphanedWorkspaces++
			report.OrphanedWorkspaceIDs = append(report.OrphanedWorkspaceIDs, name)
		}
	}

	// Step 4: Find orphaned sessions (session exists but has no workspace)
	for _, s := range sessions {
		if _, hasWorkspace := sessionToWorkspace[s.ID]; !hasWorkspace {
			// Only count worker sessions (identified by beads ID in title)
			if extractBeadsIDFromTitle(s.Title) != "" {
				report.OrphanedSessions++
				report.OrphanedSessionIDs = append(report.OrphanedSessionIDs, s.ID)
			}
		}
	}

	// Step 5: Find zombie sessions (sessions with workspaces idle >30 min)
	const zombieThreshold = 30 * time.Minute
	now := time.Now()
	for _, s := range sessions {
		updatedAt := time.Unix(s.Time.Updated/1000, 0)
		idleTime := now.Sub(updatedAt)

		// Session is potentially a zombie if:
		// 1. Has a workspace (was spawned by orch)
		// 2. Hasn't been updated in >30 min
		workspaceName := sessionToWorkspace[s.ID]
		if workspaceName != "" && idleTime > zombieThreshold {
			report.ZombieSessions++
			report.ZombieSessionIDs = append(report.ZombieSessionIDs, s.ID)
		}
	}

	// Print summary report
	printSessionsCrossReferenceReport(report, projectDir, sessionByID, workspaceBeadsID)

	return nil
}

// printSessionsCrossReferenceReport prints the cross-reference report in a clean format
func printSessionsCrossReferenceReport(report *SessionsCrossReferenceReport, projectDir string, sessionByID map[string]opencode.Session, workspaceBeadsID map[string]string) {
	fmt.Println("orch doctor --sessions")
	fmt.Printf("Workspaces: %d\n", report.WorkspaceCount)
	fmt.Printf("Sessions: %d active\n", report.SessionCount)
	fmt.Printf("Orphaned workspaces: %d (session deleted)\n", report.OrphanedWorkspaces)
	fmt.Printf("Orphaned sessions: %d (no workspace)\n", report.OrphanedSessions)
	fmt.Printf("Zombie sessions: %d\n", report.ZombieSessions)

	// If everything is clean, show success
	totalIssues := report.OrphanedWorkspaces + report.OrphanedSessions + report.ZombieSessions
	if totalIssues == 0 {
		fmt.Println()
		fmt.Println("✓ All workspaces and sessions are properly linked")
		return
	}

	// Show details for issues
	fmt.Println()

	if report.OrphanedWorkspaces > 0 && doctorVerbose {
		fmt.Println("Orphaned workspaces (session was garbage-collected):")
		for _, name := range report.OrphanedWorkspaceIDs {
			beadsID := workspaceBeadsID[name]
			if beadsID != "" {
				fmt.Printf("  - %s [%s]\n", name, beadsID)
			} else {
				fmt.Printf("  - %s\n", name)
			}
		}
		fmt.Println()
	}

	if report.OrphanedSessions > 0 && doctorVerbose {
		fmt.Println("Orphaned sessions (no corresponding workspace):")
		for _, sessionID := range report.OrphanedSessionIDs {
			s := sessionByID[sessionID]
			title := s.Title
			if title == "" {
				title = "(untitled)"
			}
			age := time.Since(time.Unix(s.Time.Created/1000, 0))
			fmt.Printf("  - %s: %s (%.0f days old)\n", shortID(sessionID), title, age.Hours()/24)
		}
		fmt.Println()
	}

	if report.ZombieSessions > 0 {
		fmt.Println("⚠️  Zombie sessions (marked active but idle >30min):")
		for _, sessionID := range report.ZombieSessionIDs {
			s := sessionByID[sessionID]
			title := s.Title
			if title == "" {
				title = "(untitled)"
			}
			idleTime := time.Since(time.Unix(s.Time.Updated/1000, 0))
			fmt.Printf("  - %s: %s (idle %.0f min)\n", shortID(sessionID), title, idleTime.Minutes())
		}
		fmt.Println()
	}

	// Recommendations
	fmt.Println("Recommendations:")
	if report.OrphanedWorkspaces > 0 {
		fmt.Println("  - Use 'orch clean --stale' to archive old workspaces")
	}
	if report.OrphanedSessions > 0 {
		fmt.Println("  - Orphaned sessions are usually interactive/test sessions (safe to ignore)")
	}
	if report.ZombieSessions > 0 {
		fmt.Println("  - Use 'orch abandon <id>' to clean up zombie sessions")
	}
}
