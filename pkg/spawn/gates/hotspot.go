package gates

import (
	"fmt"
	"os"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
)

// HotspotResult contains the result of a spawn hotspot check.
// This is a minimal interface used by the gate; the full analysis
// lives in cmd/orch/hotspot.go.
type HotspotResult struct {
	HasHotspots        bool
	HasCriticalHotspot bool     // True when any matched hotspot is CRITICAL (>1500 lines)
	Warning            string
	CriticalFiles      []string // File paths of CRITICAL hotspots
	MatchedFiles       []string // All matched hotspot file/topic paths (for context injection)
}

// HotspotChecker is a function that runs hotspot analysis for a given project directory and task.
// Returns nil if no hotspots were detected.
type HotspotChecker func(projectDir, task string) (*HotspotResult, error)

// blockingSkills are skills that modify code and should be blocked on CRITICAL hotspots.
// Read-only/strategic skills are exempt because they need to READ hotspot files.
var blockingSkills = map[string]bool{
	"feature-impl":         true,
	"systematic-debugging": true,
}

// IsBlockingSkill returns true if the skill should be blocked on CRITICAL hotspots.
func IsBlockingSkill(skillName string) bool {
	return blockingSkills[skillName]
}

// CheckHotspot runs hotspot analysis and displays warnings if the task targets a high-churn area.
// Advisory only — emits warnings and events but never blocks.
// Daemon extraction cascades (triggered by events) handle structural health.
func CheckHotspot(projectDir, task, skillName string, daemonDriven bool, checker HotspotChecker) (*HotspotResult, error) {
	if projectDir == "" || checker == nil {
		return nil, nil
	}

	result, err := checker(projectDir, task)
	if err != nil || result == nil {
		return nil, nil
	}

	// Daemon-driven spawns stay silent (triage already happened)
	if daemonDriven {
		return result, nil
	}

	// Show hotspot warning (includes recommendation to use architect)
	fmt.Fprint(os.Stderr, result.Warning)

	// Emit advisory event for CRITICAL hotspots (daemon responds with extraction cascades)
	if result.HasCriticalHotspot && IsBlockingSkill(skillName) {
		LogHotspotAdvisory(skillName, task, result.CriticalFiles)
		fmt.Fprintln(os.Stderr, "⚠️  Advisory: CRITICAL hotspot area — daemon will schedule extraction")
	} else if !IsBlockingSkill(skillName) {
		fmt.Fprintln(os.Stderr, "✓ Strategic/read-only skill: hotspot advisory noted")
	}
	fmt.Fprintln(os.Stderr, "")

	return result, nil
}

// LogHotspotAdvisory logs a spawn.hotspot_advisory event to events.jsonl.
// This tracks when CRITICAL hotspot areas are targeted — daemon uses these
// events to trigger extraction cascades.
func LogHotspotAdvisory(skillName, task string, criticalFiles []string) {
	logger := events.NewLogger(events.DefaultLogPath())
	data := map[string]interface{}{
		"skill": skillName,
		"task":  task,
	}
	if len(criticalFiles) > 0 {
		data["critical_files"] = criticalFiles
	}
	event := events.Event{
		Type:      events.EventTypeHotspotBypassed, // keep event type for backward compat
		Timestamp: time.Now().Unix(),
		Data:      data,
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log hotspot advisory: %v\n", err)
	}
}
