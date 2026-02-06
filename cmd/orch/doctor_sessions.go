package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
)

type SessionsCrossReferenceReport struct {
	WorkspaceCount       int `json:"workspace_count"`
	SessionCount         int `json:"session_count"`
	RegistryCount        int `json:"registry_count"`
	OrphanedWorkspaces   int `json:"orphaned_workspaces"` // Workspaces with deleted sessions
	OrphanedSessions     int `json:"orphaned_sessions"`   // Sessions without workspaces
	ZombieSessions       int `json:"zombie_sessions"`     // Sessions active but stuck
	RegistryMismatches   int `json:"registry_mismatches"` // Registry entries without sessions
	OrphanedWorkspaceIDs []string
	OrphanedSessionIDs   []string
	ZombieSessionIDs     []string
	RegistryMismatchIDs  []string
}

// runSessionsCrossReference performs a cross-reference between workspaces, OpenCode sessions,
// and the orchestrator registry to detect orphaned workspaces, orphaned sessions, and zombies.
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

			// Read session ID
			if data, err := os.ReadFile(filepath.Join(wsPath, ".session_id")); err == nil {
				sessionID := strings.TrimSpace(string(data))
				if sessionID != "" {
					workspaceToSession[entry.Name()] = sessionID
					sessionToWorkspace[sessionID] = entry.Name()
				}
			}

			// Read beads ID
			if data, err := os.ReadFile(filepath.Join(wsPath, ".beads_id")); err == nil {
				beadsID := strings.TrimSpace(string(data))
				if beadsID != "" {
					workspaceBeadsID[entry.Name()] = beadsID
				}
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

	// Step 3: Load registry (orchestrator sessions)
	registry := loadSessionRegistry()
	report.RegistryCount = len(registry)

	// Step 4: Find orphaned workspaces (workspace has session ID that doesn't exist in OpenCode)
	for name, sessionID := range workspaceToSession {
		if !sessionIDSet[sessionID] {
			report.OrphanedWorkspaces++
			report.OrphanedWorkspaceIDs = append(report.OrphanedWorkspaceIDs, name)
		}
	}

	// Step 5: Find orphaned sessions (session exists but has no workspace)
	for _, s := range sessions {
		if _, hasWorkspace := sessionToWorkspace[s.ID]; !hasWorkspace {
			// Check if this is an orchestrator session (expected to not have workspace tracking)
			isOrchestratorSession := isSessionInRegistry(s.ID, registry)
			if !isOrchestratorSession {
				report.OrphanedSessions++
				report.OrphanedSessionIDs = append(report.OrphanedSessionIDs, s.ID)
			}
		}
	}

	// Step 6: Find zombie sessions (sessions that claim to be active but haven't been updated in >30 min)
	const zombieThreshold = 30 * time.Minute
	now := time.Now()
	for _, s := range sessions {
		updatedAt := time.Unix(s.Time.Updated/1000, 0)
		idleTime := now.Sub(updatedAt)

		// Session is potentially a zombie if:
		// 1. Has a workspace (was spawned by orch)
		// 2. Hasn't been updated in >30 min
		// 3. Is still registered in registry as "active"
		workspaceName := sessionToWorkspace[s.ID]
		if workspaceName != "" && idleTime > zombieThreshold {
			// Check if this session is still marked as active in registry
			for _, reg := range registry {
				if reg.SessionID == s.ID && reg.Status == "active" {
					report.ZombieSessions++
					report.ZombieSessionIDs = append(report.ZombieSessionIDs, s.ID)
					break
				}
			}
		}
	}

	// Step 7: Find registry mismatches (registry entries with session IDs that don't exist)
	for _, reg := range registry {
		if reg.SessionID != "" && !sessionIDSet[reg.SessionID] {
			report.RegistryMismatches++
			report.RegistryMismatchIDs = append(report.RegistryMismatchIDs, reg.WorkspaceName)
		}
	}

	// Print summary report
	printSessionsCrossReferenceReport(report, projectDir, sessionByID, workspaceBeadsID)

	return nil
}

// loadSessionRegistry loads the orchestrator session registry from ~/.orch/sessions.json
func loadSessionRegistry() []struct {
	WorkspaceName string
	SessionID     string
	Status        string
} {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	registryPath := filepath.Join(home, ".orch", "sessions.json")
	data, err := os.ReadFile(registryPath)
	if err != nil {
		return nil
	}

	var registry struct {
		Sessions []struct {
			WorkspaceName string `json:"workspace_name"`
			SessionID     string `json:"session_id"`
			Status        string `json:"status"`
		} `json:"sessions"`
	}

	if err := json.Unmarshal(data, &registry); err != nil {
		return nil
	}

	var result []struct {
		WorkspaceName string
		SessionID     string
		Status        string
	}
	for _, s := range registry.Sessions {
		result = append(result, struct {
			WorkspaceName string
			SessionID     string
			Status        string
		}{s.WorkspaceName, s.SessionID, s.Status})
	}
	return result
}

// isSessionInRegistry checks if a session ID is tracked in the orchestrator registry
func isSessionInRegistry(sessionID string, registry []struct {
	WorkspaceName string
	SessionID     string
	Status        string
}) bool {
	if sessionID == "" {
		return false
	}
	for _, reg := range registry {
		if reg.SessionID == sessionID {
			return true
		}
	}
	return false
}

// printSessionsCrossReferenceReport prints the cross-reference report in a clean format
func printSessionsCrossReferenceReport(report *SessionsCrossReferenceReport, projectDir string, sessionByID map[string]opencode.Session, workspaceBeadsID map[string]string) {
	fmt.Println("orch doctor --sessions")
	fmt.Printf("Workspaces: %d\n", report.WorkspaceCount)
	fmt.Printf("Sessions: %d active\n", report.SessionCount)
	fmt.Printf("Orphaned workspaces: %d (session deleted)\n", report.OrphanedWorkspaces)
	fmt.Printf("Orphaned sessions: %d (no workspace)\n", report.OrphanedSessions)
	fmt.Printf("Zombie sessions: %d\n", report.ZombieSessions)
	if report.RegistryMismatches > 0 {
		fmt.Printf("Registry mismatches: %d\n", report.RegistryMismatches)
	}

	// If everything is clean, show success
	totalIssues := report.OrphanedWorkspaces + report.OrphanedSessions + report.ZombieSessions + report.RegistryMismatches
	if totalIssues == 0 {
		fmt.Println()
		fmt.Println("✓ All workspaces, sessions, and registry entries are properly linked")
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
			fmt.Printf("  - %s: %s (%.0f days old)\n", sessionID[:12], title, age.Hours()/24)
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
			fmt.Printf("  - %s: %s (idle %.0f min)\n", sessionID[:12], title, idleTime.Minutes())
		}
		fmt.Println()
	}

	if report.RegistryMismatches > 0 && doctorVerbose {
		fmt.Println("Registry mismatches (session ID no longer exists):")
		for _, name := range report.RegistryMismatchIDs {
			fmt.Printf("  - %s\n", name)
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
	if report.RegistryMismatches > 0 {
		fmt.Println("  - Registry entries with missing sessions can be cleaned with 'orch clean'")
	}
}
