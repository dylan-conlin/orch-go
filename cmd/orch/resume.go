// Package main provides the CLI entry point for orch-go.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/registry"
	"github.com/spf13/cobra"
)

var resumeCmd = &cobra.Command{
	Use:   "resume [beads-id]",
	Short: "Resume a paused agent with workspace-aware continuation",
	Long: `Resume a paused agent by sending a continuation prompt via the OpenCode API.

Looks up the agent by beads ID in the registry, finds the associated session,
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

	// Look up agent in registry
	reg, err := registry.New("")
	if err != nil {
		return fmt.Errorf("failed to open registry: %w", err)
	}

	agent := reg.Find(beadsID)
	if agent == nil {
		return fmt.Errorf("no agent found for beads ID: %s", beadsID)
	}

	// Check if agent is still active
	if agent.Status != registry.StateActive {
		return fmt.Errorf("agent %s is not active (status: %s)", beadsID, agent.Status)
	}

	// Verify the agent has a session ID
	if agent.SessionID == "" {
		return fmt.Errorf("agent %s has no associated session ID - cannot resume via API", beadsID)
	}

	// Generate the resume prompt
	prompt := GenerateResumePrompt(agent.ID, projectDir, beadsID)

	// Send the resume message via OpenCode API
	client := opencode.NewClient(serverURL)
	if err := client.SendMessageAsync(agent.SessionID, prompt); err != nil {
		return fmt.Errorf("failed to send resume prompt: %w", err)
	}

	// Log the resume event
	logger := events.NewLogger(events.DefaultLogPath())
	event := events.Event{
		Type:      "agent.resumed",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"beads_id":   beadsID,
			"agent_id":   agent.ID,
			"session_id": agent.SessionID,
			"project":    projectName,
			"skill":      agent.Skill,
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// Print summary
	fmt.Printf("Resumed agent:\n")
	fmt.Printf("  Agent ID:   %s\n", agent.ID)
	fmt.Printf("  Beads ID:   %s\n", beadsID)
	fmt.Printf("  Session ID: %s\n", agent.SessionID)
	fmt.Printf("  Skill:      %s\n", agent.Skill)

	return nil
}
