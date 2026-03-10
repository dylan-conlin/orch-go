package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/spf13/cobra"
)

// ============================================================================
// Session Label Command
// ============================================================================

var sessionLabelCmd = &cobra.Command{
	Use:   "label [name]",
	Short: "Set a human-readable label for the current OpenCode session",
	Long: `Set a human-readable label for the current OpenCode session.

This label will be used in the dashboard timeline view to identify
the session instead of showing the raw session ID (ses_xxxxx).

The label is stored in the .orch workspace for the current project
and is associated with the OpenCode session ID from the environment.

Examples:
  orch session label "verifiability design review"
  orch session label "bug fixes"
  orch session label "dashboard timeline feature"`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		label := strings.Join(args, " ")
		return runSessionLabel(label)
	},
}

func runSessionLabel(label string) error {
	// Get current OpenCode session ID from environment
	sessionID := os.Getenv("CLAUDE_SESSION_ID")
	if sessionID == "" {
		return fmt.Errorf("no OpenCode session detected (CLAUDE_SESSION_ID not set)")
	}

	// Get current working directory (project directory)
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get project directory: %w", err)
	}

	// Ensure .orch directory exists
	orchDir := filepath.Join(projectDir, ".orch")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		return fmt.Errorf("failed to create .orch directory: %w", err)
	}

	// Store session label in .orch/session_labels.json
	labelsFile := filepath.Join(orchDir, "session_labels.json")

	// Load existing labels
	labels := make(map[string]string)
	if data, err := os.ReadFile(labelsFile); err == nil {
		json.Unmarshal(data, &labels)
	}

	// Add/update label for this session
	labels[sessionID] = label

	// Write back to file
	data, err := json.MarshalIndent(labels, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode labels: %w", err)
	}

	if err := os.WriteFile(labelsFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write labels file: %w", err)
	}

	// Log the session label event
	logger := events.NewLogger(events.DefaultLogPath())
	event := events.Event{
		Type:      "session.labeled",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"session_id": sessionID,
			"label":      label,
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	fmt.Printf("✅ Session labeled: %s\n", label)
	fmt.Printf("   Session ID: %s\n", sessionID)

	return nil
}
