package gates

import (
	"fmt"
	"os"
)

const (
	// HealthScoreFloor is the minimum health score required to spawn feature-impl.
	// Below this, extraction work must happen first.
	// Decision ref: kb-ed3edc
	HealthScoreFloor = 65.0
)

// healthBlockedSkills are skills blocked when health score is below floor.
// Same set as hotspot blocking — tactical work that adds code.
var healthBlockedSkills = map[string]bool{
	"feature-impl":         true,
	"systematic-debugging": true,
}

// HealthScoreProvider computes the current health score.
// Injected to keep the gates package decoupled from snapshot collection.
type HealthScoreProvider func() (float64, string, error)

// CheckHealthScore blocks feature-impl spawns when the harness health score
// is below the floor (65/C). Extraction and architect skills are exempt.
// Returns nil if the check passes or is not applicable.
func CheckHealthScore(skillName string, daemonDriven, skipHealthGate bool, provider HealthScoreProvider) error {
	if provider == nil {
		return nil
	}

	// Only block tactical skills
	if !healthBlockedSkills[skillName] {
		return nil
	}

	// Daemon-driven spawns bypass (triage already approved)
	if daemonDriven {
		return nil
	}

	if skipHealthGate {
		fmt.Fprintln(os.Stderr, "⚠️  --skip-health-gate: Bypassing health score floor check")
		return nil
	}

	score, grade, err := provider()
	if err != nil {
		// If we can't compute the score, don't block
		fmt.Fprintf(os.Stderr, "⚠️  Could not compute health score: %v (proceeding)\n", err)
		return nil
	}

	if score < HealthScoreFloor {
		return fmt.Errorf("health score %.0f (%s) is below floor %.0f (C) — extract bloated files before adding features.\n"+
			"Run: orch health\n"+
			"Override: --skip-health-gate --reason \"...\"",
			score, grade, HealthScoreFloor)
	}

	fmt.Fprintf(os.Stderr, "✓ Health score: %.0f (%s)\n", score, grade)
	return nil
}
