// Package main provides the CLI entry point for orch-go.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/spf13/cobra"
)

// Resume command flags
var (
	resumeWorkspace string
	resumeSession   string
)

var resumeCmd = &cobra.Command{
	Use:   "resume [beads-id]",
	Short: "Resume a paused agent with workspace-aware continuation",
	Long: `Resume a paused agent by sending a continuation prompt via the OpenCode API.

Looks up the agent by beads ID, workspace name, or session ID, finds the associated
session, and sends a message to continue work with full workspace context.

For workers (with beads IDs):
  orch resume proj-123                    # Resume by beads ID

For orchestrators or sessions without beads:
  orch resume --workspace meta-orch-xyz   # Resume by workspace name
  orch resume --session ses_abc123        # Resume by session ID directly

The --workspace flag finds the workspace directory, reads the .session_id file,
and generates an appropriate resume prompt based on the context type
(SPAWN_CONTEXT.md for workers, ORCHESTRATOR_CONTEXT.md or META_ORCHESTRATOR_CONTEXT.md
for orchestrators).

The --session flag sends a resume prompt directly to the session without
requiring a workspace.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validate flag combinations
		hasBeadsID := len(args) > 0
		hasWorkspace := resumeWorkspace != ""
		hasSession := resumeSession != ""

		// Count how many identifiers provided
		count := 0
		if hasBeadsID {
			count++
		}
		if hasWorkspace {
			count++
		}
		if hasSession {
			count++
		}

		if count == 0 {
			return fmt.Errorf("must provide one of: beads-id argument, --workspace, or --session")
		}
		if count > 1 {
			return fmt.Errorf("provide only one of: beads-id argument, --workspace, or --session")
		}

		client := opencode.NewClient(serverURL)
		switch {
		case hasBeadsID:
			return runResumeByBeadsID(client, args[0])
		case hasWorkspace:
			return runResumeByWorkspace(client, resumeWorkspace)
		case hasSession:
			return runResumeBySession(client, resumeSession)
		default:
			return fmt.Errorf("no identifier provided")
		}
	},
}

func init() {
	resumeCmd.Flags().StringVar(&resumeWorkspace, "workspace", "", "Resume by workspace name (for orchestrators)")
	resumeCmd.Flags().StringVar(&resumeSession, "session", "", "Resume by session ID directly")
	rootCmd.AddCommand(resumeCmd)
}

// GenerateResumePrompt creates a prompt for resuming an agent with workspace context.
// For workers with beads tracking.
func GenerateResumePrompt(workspaceName, projectDir, beadsID string) string {
	contextPath := filepath.Join(projectDir, ".orch", "workspace", workspaceName, "SPAWN_CONTEXT.md")
	return fmt.Sprintf(
		"You were paused mid-task. Re-read your spawn context from %s and continue your work. "+
			"Report progress via bd comment %s.",
		contextPath,
		beadsID,
	)
}

// GenerateOrchestratorResumePrompt creates a prompt for resuming an orchestrator session.
// It detects the context file type and generates an appropriate prompt.
func GenerateOrchestratorResumePrompt(workspaceName, projectDir string) string {
	workspacePath := filepath.Join(projectDir, ".orch", "workspace", workspaceName)

	// Check for different context file types in order of specificity
	contextFiles := []string{
		"META_ORCHESTRATOR_CONTEXT.md",
		"ORCHESTRATOR_CONTEXT.md",
		"SPAWN_CONTEXT.md",
	}

	var contextPath string
	for _, filename := range contextFiles {
		path := filepath.Join(workspacePath, filename)
		if _, err := os.Stat(path); err == nil {
			contextPath = path
			break
		}
	}

	if contextPath == "" {
		// Fallback to SPAWN_CONTEXT.md path even if it doesn't exist
		contextPath = filepath.Join(workspacePath, "SPAWN_CONTEXT.md")
	}

	return fmt.Sprintf(
		"You were paused mid-session. Re-read your context from %s and continue your work.",
		contextPath,
	)
}

// GenerateSessionResumePrompt creates a minimal prompt for direct session resume.
func GenerateSessionResumePrompt() string {
	return "You were paused mid-task. Continue your work from where you left off."
}

// runResumeByBeadsID resumes an agent by beads ID (original behavior).
func runResumeByBeadsID(client opencode.ClientInterface, beadsID string) error {
	// Get current directory to determine project
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	projectName := filepath.Base(projectDir)

	// Find workspace by beadsID and read session_id
	var sessionID, agentID, workspacePath string
	workspaceBase := filepath.Join(projectDir, ".orch", "workspace")
	if entries, err := os.ReadDir(workspaceBase); err == nil {
		for _, entry := range entries {
			if entry.IsDir() && strings.Contains(entry.Name(), beadsID) {
				workspacePath = filepath.Join(workspaceBase, entry.Name())
				sessionID = spawn.ReadSessionID(workspacePath)
				agentID = entry.Name()
				break
			}
		}
	}

	// If workspace file doesn't have session_id, try to find via OpenCode API
	if sessionID == "" {
		allSessions, err := client.ListSessions(projectDir)
		if err == nil {
			for _, s := range allSessions {
				if strings.Contains(s.Title, beadsID) || extractBeadsIDFromTitle(s.Title) == beadsID {
					sessionID = s.ID
					break
				}
			}
		}
	}

	if sessionID == "" {
		return fmt.Errorf("no agent found for beads ID: %s (no workspace file or active session)", beadsID)
	}

	// If we didn't find workspace, use beadsID as agentID
	if agentID == "" {
		agentID = beadsID
	}

	// Generate the resume prompt
	prompt := GenerateResumePrompt(agentID, projectDir, beadsID)

	// Send the resume message via OpenCode API (no model for resume)
	if err := client.SendMessageAsync(sessionID, prompt, ""); err != nil {
		return fmt.Errorf("failed to send resume prompt: %w", err)
	}

	// Log the resume event
	logger := events.NewLogger(events.DefaultLogPath())
	event := events.Event{
		Type:      "agent.resumed",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"beads_id":   beadsID,
			"agent_id":   agentID,
			"session_id": sessionID,
			"project":    projectName,
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// Print summary
	fmt.Printf("Resumed agent:\n")
	fmt.Printf("  Agent ID:   %s\n", agentID)
	fmt.Printf("  Beads ID:   %s\n", beadsID)
	fmt.Printf("  Session ID: %s\n", sessionID)

	return nil
}

// runResumeByWorkspace resumes an agent by workspace name.
// This is particularly useful for orchestrators which don't have beads IDs.
func runResumeByWorkspace(client opencode.ClientInterface, workspaceName string) error {
	// Get current directory to determine project
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	projectName := filepath.Base(projectDir)

	// Build workspace path
	workspaceBase := filepath.Join(projectDir, ".orch", "workspace")
	workspacePath := filepath.Join(workspaceBase, workspaceName)

	// Check if workspace exists
	if _, err := os.Stat(workspacePath); os.IsNotExist(err) {
		// Try to find a workspace that starts with or contains the given name
		if entries, err := os.ReadDir(workspaceBase); err == nil {
			for _, entry := range entries {
				if entry.IsDir() && (strings.HasPrefix(entry.Name(), workspaceName) || strings.Contains(entry.Name(), workspaceName)) {
					workspacePath = filepath.Join(workspaceBase, entry.Name())
					workspaceName = entry.Name()
					break
				}
			}
		}
	}

	// Re-check if workspace exists after potential match
	if _, err := os.Stat(workspacePath); os.IsNotExist(err) {
		return fmt.Errorf("workspace not found: %s", workspaceName)
	}

	// Read session ID from workspace
	sessionID := spawn.ReadSessionID(workspacePath)
	if sessionID == "" {
		// Try to find via OpenCode API by workspace name in title
		allSessions, err := client.ListSessions(projectDir)
		if err == nil {
			for _, s := range allSessions {
				if strings.Contains(s.Title, workspaceName) {
					sessionID = s.ID
					break
				}
			}
		}
	}

	if sessionID == "" {
		return fmt.Errorf("no session found for workspace: %s (no .session_id file or matching session)", workspaceName)
	}

	// Check for beads ID (optional for orchestrators)
	beadsIDPath := filepath.Join(workspacePath, ".beads_id")
	beadsIDBytes, _ := os.ReadFile(beadsIDPath)
	beadsID := strings.TrimSpace(string(beadsIDBytes))

	// Generate appropriate resume prompt
	var prompt string
	if beadsID != "" {
		prompt = GenerateResumePrompt(workspaceName, projectDir, beadsID)
	} else {
		prompt = GenerateOrchestratorResumePrompt(workspaceName, projectDir)
	}

	// Send the resume message via OpenCode API
	if err := client.SendMessageAsync(sessionID, prompt, ""); err != nil {
		return fmt.Errorf("failed to send resume prompt: %w", err)
	}

	// Log the resume event
	logger := events.NewLogger(events.DefaultLogPath())
	eventData := map[string]interface{}{
		"workspace":  workspaceName,
		"session_id": sessionID,
		"project":    projectName,
	}
	if beadsID != "" {
		eventData["beads_id"] = beadsID
	}
	event := events.Event{
		Type:      "agent.resumed",
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// Print summary
	fmt.Printf("Resumed agent:\n")
	fmt.Printf("  Workspace:  %s\n", workspaceName)
	fmt.Printf("  Session ID: %s\n", sessionID)
	if beadsID != "" {
		fmt.Printf("  Beads ID:   %s\n", beadsID)
	}

	return nil
}

// runResumeBySession resumes an agent directly by session ID.
// This is useful when you have the session ID but not the workspace.
func runResumeBySession(client opencode.ClientInterface, sessionID string) error {
	// Get current directory to determine project
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	projectName := filepath.Base(projectDir)

	// Verify session exists
	if !client.SessionExists(sessionID) {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Generate minimal resume prompt
	prompt := GenerateSessionResumePrompt()

	// Send the resume message via OpenCode API
	if err := client.SendMessageAsync(sessionID, prompt, ""); err != nil {
		return fmt.Errorf("failed to send resume prompt: %w", err)
	}

	// Log the resume event
	logger := events.NewLogger(events.DefaultLogPath())
	event := events.Event{
		Type:      "agent.resumed",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"session_id": sessionID,
			"project":    projectName,
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// Print summary
	fmt.Printf("Resumed session:\n")
	fmt.Printf("  Session ID: %s\n", sessionID)

	return nil
}
