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

// CheckHealthScore warns when the harness health score is below the floor (65/C).
// Advisory only — does not block spawns. Pre-commit accretion gate and hotspot
// blocking are the real enforcement.
// Decision ref: kb-3e651d (downgraded from blocking after Phase 4 probe found
// score improvement was 89% calibration artifact).
func CheckHealthScore(skillName string, daemonDriven, skipHealthGate bool, provider HealthScoreProvider) error {
	if provider == nil {
		return nil
	}

	// Only warn for tactical skills
	if !healthBlockedSkills[skillName] {
		return nil
	}

	// Daemon-driven spawns skip the advisory
	if daemonDriven {
		return nil
	}

	score, grade, err := provider()
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Could not compute health score: %v (proceeding)\n", err)
		return nil
	}

	if score < HealthScoreFloor {
		fmt.Fprintf(os.Stderr, "⚠️  Health score %.0f (%s) — below %.0f threshold\n", score, grade, HealthScoreFloor)
		return nil
	}

	fmt.Fprintf(os.Stderr, "✓ Health score: %.0f (%s)\n", score, grade)
	return nil
}
