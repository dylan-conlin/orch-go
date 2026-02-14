package gates

import (
	"fmt"
	"os"
)

// HotspotResult contains the result of a spawn hotspot check.
// This is a minimal interface used by the gate; the full analysis
// lives in cmd/orch/hotspot.go.
type HotspotResult struct {
	HasHotspots bool
	Warning     string
}

// HotspotChecker is a function that runs hotspot analysis for a given project directory and task.
// Returns nil if no hotspots were detected.
type HotspotChecker func(projectDir, task string) (*HotspotResult, error)

// CheckHotspot runs hotspot analysis and displays warnings if the task targets a high-churn area.
// The checker function performs the actual hotspot analysis (injected from cmd/orch).
// daemonDriven spawns suppress output (triage already happened).
// Returns the result for downstream use (e.g., event logging), or nil if no hotspots.
func CheckHotspot(projectDir, task, skillName string, daemonDriven bool, checker HotspotChecker) *HotspotResult {
	if projectDir == "" || checker == nil {
		return nil
	}

	result, err := checker(projectDir, task)
	if err != nil || result == nil {
		return nil
	}

	// Daemon-driven spawns stay silent (triage already happened)
	if daemonDriven {
		return result
	}

	// Show hotspot warning (includes recommendation to use architect)
	fmt.Fprint(os.Stderr, result.Warning)

	// Add context based on skill choice
	isStrategicSkill := skillName == "architect"
	if isStrategicSkill {
		fmt.Fprintln(os.Stderr, "✓ Strategic approach: architect skill selected for hotspot area")
	} else {
		fmt.Fprintln(os.Stderr, "⚠️  Proceeding with tactical approach in hotspot area")
	}
	fmt.Fprintln(os.Stderr, "")

	return result
}
