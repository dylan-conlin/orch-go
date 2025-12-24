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

var resumeCmd = &cobra.Command{
	Use:   "resume [beads-id]",
	Short: "Resume a paused agent with workspace-aware continuation",
	Long: `Resume a paused agent by sending a continuation prompt via the OpenCode API.

Looks up the agent by beads ID via workspace files, finds the associated session,
and sends a message to continue work with full workspace context.

Examples:
  orch-go resume proj-123           # Resume agent working on proj-123`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		beadsID := args[0]
		return runResume(beadsID)
	},
}

func init() {
	rootCmd.AddCommand(resumeCmd)
}

// GenerateResumePrompt creates a prompt for resuming an agent with workspace context.
func GenerateResumePrompt(workspaceName, projectDir, beadsID string) string {
	contextPath := filepath.Join(projectDir, ".orch", "workspace", workspaceName, "SPAWN_CONTEXT.md")
	return fmt.Sprintf(
		"You were paused mid-task. Re-read your spawn context from %s and continue your work. "+
			"Report progress via bd comment %s.",
		contextPath,
		beadsID,
	)
}

func runResume(beadsID string) error {
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
		client := opencode.NewClient(serverURL)
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
	client := opencode.NewClient(serverURL)
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
