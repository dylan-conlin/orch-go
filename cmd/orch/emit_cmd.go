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
  - exploration.decomposed: Exploration orchestrator decomposed question into subproblems
  - exploration.judged: Exploration judge produced verdicts on sub-findings
  - exploration.synthesized: Exploration run produced final synthesis

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
	// Parse additional data
	var additionalData map[string]interface{}
	if dataStr != "" {
		if err := json.Unmarshal([]byte(dataStr), &additionalData); err != nil {
			return fmt.Errorf("invalid --data JSON: %w", err)
		}
	}

	logger := events.NewLogger(events.DefaultLogPath())

	// Validate and handle event type
	switch eventType {
	case "agent.completed":
		if beadsID == "" {
			return fmt.Errorf("--beads-id is required for agent.completed events")
		}
		return emitAgentCompleted(logger, beadsID, reason, additionalData, jsonOutput)

	case "exploration.decomposed", "exploration.judged", "exploration.synthesized", "exploration.iterated":
		return emitExplorationEvent(logger, eventType, beadsID, reason, additionalData, jsonOutput)

	default:
		return fmt.Errorf("unsupported event type: %s (supported: agent.completed, exploration.decomposed, exploration.judged, exploration.synthesized, exploration.iterated)", eventType)
	}
}

func emitAgentCompleted(logger *events.Logger, beadsID, reason string, additionalData map[string]interface{}, jsonOutput bool) error {
	completedData := events.AgentCompletedData{
		BeadsID: beadsID,
		Reason:  reason,
		Outcome: "success",
	}

	if additionalData != nil {
		if v, ok := additionalData["skill"].(string); ok {
			completedData.Skill = v
		}
		if v, ok := additionalData["outcome"].(string); ok {
			completedData.Outcome = v
		}
	}

	// Auto-enrich from workspace manifest if available
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
		if manifest.VerifyLevel != "" {
			completedData.VerificationLevel = manifest.VerifyLevel
		}
		if spawnTime := manifest.ParseSpawnTime(); !spawnTime.IsZero() {
			completedData.DurationSeconds = int(time.Since(spawnTime).Seconds())
		}
	}

	if err := logger.LogAgentCompleted(completedData); err != nil {
		return fmt.Errorf("failed to log event: %w", err)
	}

	return emitOutput("agent.completed", beadsID, reason, completedData.Skill, jsonOutput)
}

func emitExplorationEvent(logger *events.Logger, eventType, beadsID, reason string, additionalData map[string]interface{}, jsonOutput bool) error {
	getStr := func(key string) string {
		if additionalData == nil {
			return ""
		}
		v, _ := additionalData[key].(string)
		return v
	}
	getInt := func(key string) int {
		if additionalData == nil {
			return 0
		}
		v, _ := additionalData[key].(float64)
		return int(v)
	}
	getStrSlice := func(key string) []string {
		if additionalData == nil {
			return nil
		}
		raw, ok := additionalData[key].([]interface{})
		if !ok {
			return nil
		}
		var result []string
		for _, item := range raw {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}

	var err error
	switch eventType {
	case "exploration.decomposed":
		err = logger.LogExplorationDecomposed(events.ExplorationDecomposedData{
			BeadsID:     beadsID,
			ParentSkill: getStr("parent_skill"),
			Question:    getStr("question"),
			Subproblems: getStrSlice("subproblems"),
			Breadth:     getInt("breadth"),
		})
	case "exploration.judged":
		err = logger.LogExplorationJudged(events.ExplorationJudgedData{
			BeadsID:       beadsID,
			ParentSkill:   getStr("parent_skill"),
			TotalFindings: getInt("total_findings"),
			Accepted:      getInt("accepted"),
			Contested:     getInt("contested"),
			Rejected:      getInt("rejected"),
			CoverageGaps:  getInt("coverage_gaps"),
		})
	case "exploration.synthesized":
		err = logger.LogExplorationSynthesized(events.ExplorationSynthesizedData{
			BeadsID:         beadsID,
			ParentSkill:     getStr("parent_skill"),
			WorkerCount:     getInt("worker_count"),
			DurationSeconds: getInt("duration_seconds"),
			SynthesisPath:   getStr("synthesis_path"),
		})
	case "exploration.iterated":
		err = logger.LogExplorationIterated(events.ExplorationIteratedData{
			BeadsID:       beadsID,
			ParentSkill:   getStr("parent_skill"),
			Iteration:     getInt("iteration"),
			GapsAddressed: getInt("gaps_addressed"),
			NewWorkers:    getInt("new_workers"),
		})
	}

	if err != nil {
		return fmt.Errorf("failed to log event: %w", err)
	}

	return emitOutput(eventType, beadsID, reason, "", jsonOutput)
}

func emitOutput(eventType, beadsID, reason, skill string, jsonOutput bool) error {
	if jsonOutput {
		event := events.Event{
			Type:      eventType,
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"beads_id": beadsID,
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
		fmt.Printf("✓ Emitted %s event", eventType)
		if beadsID != "" {
			fmt.Printf(" for %s", beadsID)
		}
		fmt.Println()
		if reason != "" {
			fmt.Printf("  Reason: %s\n", reason)
		}
		if skill != "" {
			fmt.Printf("  Skill: %s\n", skill)
		}
	}

	invalidateServeCache()
	return nil
}

// emitAgentCompletedFromHook is a helper function that can be called from Go code
// to emit an agent.completed event from a hook context.
func emitAgentCompletedFromHook(beadsID, reason string) error {
	return runEmit("agent.completed", beadsID, reason, "", false)
}

// EmitResult is the structured output when --json is used.
type EmitResult struct {
	Type      string                 `json:"type"`
	Timestamp int64                  `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Success   bool                   `json:"success"`
}

// printEmitHelp prints additional usage information.
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
