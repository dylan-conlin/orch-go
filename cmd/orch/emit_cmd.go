// Package main provides the emit command for emitting events to events.jsonl.
// This enables external tools (like beads hooks) to log agent lifecycle events
// that bypass the normal orch complete flow.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/spf13/cobra"
)

var (
	// Emit command flags
	emitBeadsID string
	emitReason  string
	emitJSON    bool
	emitData    string // JSON string for additional data
)

var emitCmd = &cobra.Command{
	Use:   "emit [event-type]",
	Short: "Emit an event to events.jsonl",
	Long: `Emit an event to the orchestration event log.

This command is primarily used by beads hooks to emit agent.completed events
when issues are closed directly via 'bd close', bypassing 'orch complete'.

Supported event types:
  - agent.completed: Agent finished work (requires --beads-id)

The event is appended to ~/.orch/events.jsonl in JSONL format.

Examples:
  # Emit agent.completed when bd close runs (used by .beads/hooks/on_close)
  orch emit agent.completed --beads-id proj-123 --reason "Closed via bd close"

  # Emit with JSON output
  orch emit agent.completed --beads-id proj-123 --json

  # Emit with additional data (for custom integrations)
  orch emit agent.completed --beads-id proj-123 --data '{"source":"hook"}'`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		eventType := args[0]
		return runEmit(eventType, emitBeadsID, emitReason, emitData, emitJSON)
	},
}

func init() {
	emitCmd.Flags().StringVar(&emitBeadsID, "beads-id", "", "Beads issue ID associated with this event (required for agent.completed)")
	emitCmd.Flags().StringVar(&emitReason, "reason", "", "Reason for the event (optional)")
	emitCmd.Flags().StringVar(&emitData, "data", "", "Additional data as JSON string (optional)")
	emitCmd.Flags().BoolVar(&emitJSON, "json", false, "Output the emitted event as JSON")
	rootCmd.AddCommand(emitCmd)
}

func runEmit(eventType, beadsID, reason, dataStr string, jsonOutput bool) error {
	// Validate event type
	switch eventType {
	case "agent.completed":
		if beadsID == "" {
			return fmt.Errorf("--beads-id is required for agent.completed events")
		}
	default:
		return fmt.Errorf("unsupported event type: %s (supported: agent.completed)", eventType)
	}

	// Build enriched completion event
	completedData := events.AgentCompletedData{
		BeadsID: beadsID,
		Reason:  reason,
		Outcome: "success", // Default for hook-closed issues
	}

	// Parse additional data for overrides
	if dataStr != "" {
		var additionalData map[string]interface{}
		if err := json.Unmarshal([]byte(dataStr), &additionalData); err != nil {
			return fmt.Errorf("invalid --data JSON: %w", err)
		}
		// Apply overrides from --data
		if v, ok := additionalData["skill"].(string); ok {
			completedData.Skill = v
		}
		if v, ok := additionalData["outcome"].(string); ok {
			completedData.Outcome = v
		}
	}

	// Auto-enrich from workspace manifest if available
	// Try current directory first, then cross-project search
	projectDir, _ := os.Getwd()
	wsPath, wsName := findWorkspaceByBeadsID(projectDir, beadsID)
	if wsPath == "" {
		wsPath, wsName = findWorkspaceByBeadsIDAcrossProjects(beadsID)
	}
	if wsPath != "" {
		completedData.Workspace = wsName
		manifest := spawn.ReadAgentManifestWithFallback(wsPath)
		if completedData.Skill == "" && manifest.Skill != "" {
			completedData.Skill = manifest.Skill
		}
		if spawnTime := manifest.ParseSpawnTime(); !spawnTime.IsZero() {
			completedData.DurationSeconds = int(time.Since(spawnTime).Seconds())
		}
	}

	// Log to events.jsonl
	logger := events.NewLogger(events.DefaultLogPath())
	if err := logger.LogAgentCompleted(completedData); err != nil {
		return fmt.Errorf("failed to log event: %w", err)
	}

	// Output
	if jsonOutput {
		// Build a raw event for JSON output compatibility
		event := events.Event{
			Type:      eventType,
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"beads_id": beadsID,
				"source":   "bd_close_hook",
			},
		}
		if reason != "" {
			event.Data["reason"] = reason
		}
		output, err := json.MarshalIndent(event, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal event: %w", err)
		}
		fmt.Println(string(output))
	} else {
		fmt.Printf("✓ Emitted %s event for %s\n", eventType, beadsID)
		if reason != "" {
			fmt.Printf("  Reason: %s\n", reason)
		}
		if completedData.Skill != "" {
			fmt.Printf("  Skill: %s\n", completedData.Skill)
		}
	}

	// Invalidate serve cache to ensure dashboard shows updated status immediately.
	// This is non-critical so we just call it and ignore errors.
	invalidateServeCache()

	return nil
}

// emitAgentCompletedFromHook is a helper function that can be called from Go code
// to emit an agent.completed event from a hook context.
// This is exported for potential use by other packages.
func emitAgentCompletedFromHook(beadsID, reason string) error {
	return runEmit("agent.completed", beadsID, reason, "", false)
}

// EmitResult is the structured output when --json is used.
// Not currently used but defined for future API stability.
type EmitResult struct {
	Type      string                 `json:"type"`
	Timestamp int64                  `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Success   bool                   `json:"success"`
}

// printEmitHelp prints additional usage information.
// Called when the user runs `orch emit --help`.
func printEmitHelp(cmd *cobra.Command) {
	fmt.Fprintf(os.Stderr, `
Beads Hook Integration:

To automatically emit agent.completed events when 'bd close' is called,
create an executable script at .beads/hooks/on_close:

    #!/bin/bash
    # .beads/hooks/on_close
    # Emit agent.completed event when issues are closed via bd close
    
    # BD_ISSUE_ID is set by beads when the hook runs
    if [ -n "$BD_ISSUE_ID" ]; then
        orch emit agent.completed --beads-id "$BD_ISSUE_ID" --reason "Closed via bd close"
    fi

Make it executable:
    chmod +x .beads/hooks/on_close

This closes the tracking gap where work completes but bypasses orch complete.
`)
}
