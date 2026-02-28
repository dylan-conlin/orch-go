// Package gates provides pre-spawn gate checks that must pass before an agent is spawned.
// These gates enforce triage workflow, concurrency limits, rate-limit awareness, and hotspot detection.
package gates

import (
	"fmt"
	"os"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
)

// CheckTriageBypass enforces the triage bypass requirement for manual spawns.
// Daemon-driven spawns (daemonDriven=true) skip this check since the issue is already triaged.
// Returns nil if spawn is allowed, or an error if --bypass-triage is required.
func CheckTriageBypass(daemonDriven, bypassTriage bool, skillName, task string) error {
	if daemonDriven || bypassTriage {
		return nil
	}
	return showTriageBypassRequired(skillName, task)
}

// LogTriageBypass logs a triage bypass event to events.jsonl for Phase 2 review.
// This tracks how often manual spawns occur vs daemon-driven spawns.
// Should be called when bypassTriage is true and daemonDriven is false.
func LogTriageBypass(skillName, task, reason string) {
	logger := events.NewLogger(events.DefaultLogPath())
	data := map[string]interface{}{
		"skill": skillName,
		"task":  task,
	}
	if reason != "" {
		data["reason"] = reason
	}
	event := events.Event{
		Type:      "spawn.triage_bypassed",
		Timestamp: time.Now().Unix(),
		Data:      data,
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log triage bypass: %v\n", err)
	}
}

// showTriageBypassRequired displays the triage bypass error message.
func showTriageBypassRequired(skillName, task string) error {
	truncatedTask := task
	if len(truncatedTask) > 30 {
		truncatedTask = truncatedTask[:30] + "..."
	}

	fmt.Fprintf(os.Stderr, `
┌─────────────────────────────────────────────────────────────────────────────┐
│  ⚠️  TRIAGE BYPASS REQUIRED                                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│  Manual spawn requires --bypass-triage flag.                                │
│                                                                             │
│  The preferred workflow is daemon-driven triage:                            │
│    1. Create issue: bd create "task" --type task -l triage:ready            │
│    2. Daemon auto-spawns: orch daemon run                                   │
│                                                                             │
│  Manual spawn is for exceptions only:                                       │
│    - Single urgent item requiring immediate attention                       │
│    - Complex/ambiguous task needing custom context                          │
│    - Skill selection requires orchestrator judgment                         │
│                                                                             │
│  To proceed with manual spawn, add --bypass-triage:                         │
│    orch spawn --bypass-triage %s "%s"                          │
└─────────────────────────────────────────────────────────────────────────────┘

`, skillName, truncatedTask)
	return fmt.Errorf("spawn blocked: --bypass-triage flag required for manual spawns")
}
