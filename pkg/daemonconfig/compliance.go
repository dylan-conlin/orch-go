package daemonconfig

import "strings"

// ComplianceLevel represents how strictly compliance mechanisms are enforced.
// Higher levels mean less enforcement overhead. Default is Strict (current behavior).
type ComplianceLevel int

const (
	// ComplianceStrict is the current default behavior. All compliance mechanisms active.
	ComplianceStrict ComplianceLevel = iota
	// ComplianceStandard relaxes some compliance for proven model/skill combos.
	ComplianceStandard
	// ComplianceRelaxed significantly reduces compliance overhead.
	ComplianceRelaxed
	// ComplianceAutonomous minimizes compliance to safety-only mechanisms.
	ComplianceAutonomous
)

// String returns the string representation of a ComplianceLevel.
func (l ComplianceLevel) String() string {
	switch l {
	case ComplianceStrict:
		return "strict"
	case ComplianceStandard:
		return "standard"
	case ComplianceRelaxed:
		return "relaxed"
	case ComplianceAutonomous:
		return "autonomous"
	default:
		return "strict"
	}
}

// ParseComplianceLevel parses a string into a ComplianceLevel.
// Returns the level and true if valid, or ComplianceStrict and false if invalid.
func ParseComplianceLevel(s string) (ComplianceLevel, bool) {
	switch strings.ToLower(s) {
	case "strict":
		return ComplianceStrict, true
	case "standard":
		return ComplianceStandard, true
	case "relaxed":
		return ComplianceRelaxed, true
	case "autonomous":
		return ComplianceAutonomous, true
	default:
		return ComplianceStrict, false
	}
}

// ComplianceConfig holds compliance level configuration with per-skill, per-model,
// and per-combo overrides. Resolution order: combo > skill > model > default.
type ComplianceConfig struct {
	// Default is the global compliance level when no override matches.
	// Zero value (ComplianceStrict) preserves current behavior.
	Default ComplianceLevel

	// Skills maps skill names to compliance levels.
	Skills map[string]ComplianceLevel

	// Models maps model names to compliance levels.
	Models map[string]ComplianceLevel

	// Combos maps "model+skill" keys to compliance levels (highest precedence).
	Combos map[string]ComplianceLevel
}

// Resolve determines the effective compliance level for a (skill, model) pair.
// Resolution order: combo(model+skill) > skill > model > default.
func (c *ComplianceConfig) Resolve(skill, model string) ComplianceLevel {
	// 1. Check combo (highest precedence)
	if c.Combos != nil {
		key := model + "+" + skill
		if level, ok := c.Combos[key]; ok {
			return level
		}
	}
	// 2. Check skill
	if c.Skills != nil {
		if level, ok := c.Skills[skill]; ok {
			return level
		}
	}
	// 3. Check model
	if c.Models != nil {
		if level, ok := c.Models[model]; ok {
			return level
		}
	}
	// 4. Global default
	return c.Default
}

// DeriveVerificationThreshold returns the verification pause threshold for a compliance level.
// This is the number of auto-completions before the daemon pauses for human review.
func DeriveVerificationThreshold(level ComplianceLevel) int {
	switch level {
	case ComplianceStrict:
		return 3
	case ComplianceStandard:
		return 8
	case ComplianceRelaxed:
		return 20
	case ComplianceAutonomous:
		return 0 // disabled
	default:
		return 3
	}
}

// DeriveInvariantThreshold returns the invariant violation threshold for a compliance level.
// This is the number of consecutive violation cycles before the daemon pauses.
func DeriveInvariantThreshold(level ComplianceLevel) int {
	switch level {
	case ComplianceStrict:
		return 3
	case ComplianceStandard:
		return 5
	case ComplianceRelaxed:
		return 10
	case ComplianceAutonomous:
		return 0 // disabled
	default:
		return 3
	}
}

// DeriveArchitectEscalationEnabled returns whether architect escalation is active
// at the given compliance level.
func DeriveArchitectEscalationEnabled(level ComplianceLevel) bool {
	return level <= ComplianceStandard
}

// DeriveSynthesisRequired returns whether SYNTHESIS.md is required at the given
// compliance level and spawn tier. Light-tier spawns never require synthesis
// regardless of compliance level.
func DeriveSynthesisRequired(level ComplianceLevel, tier string) bool {
	if tier == "light" {
		return false
	}
	return level <= ComplianceStandard
}

// DerivePhaseEnforcement returns the phase enforcement mode for a compliance level.
// "required" means missing phases block completion; "advisory" means they're logged but not enforced.
func DerivePhaseEnforcement(level ComplianceLevel) string {
	if level <= ComplianceStandard {
		return "required"
	}
	return "advisory"
}

// DeriveTriggerBudget returns the max open daemon:trigger issues for a compliance level.
// Used as a compliance ceiling on the config TriggerBudgetMax.
//   - Strict: conservative (10)
//   - Standard: moderate (10)
//   - Relaxed: generous (15)
//   - Autonomous: maximum (20)
func DeriveTriggerBudget(level ComplianceLevel) int {
	switch level {
	case ComplianceStrict:
		return 10
	case ComplianceStandard:
		return 10
	case ComplianceRelaxed:
		return 15
	case ComplianceAutonomous:
		return 20
	default:
		return 10
	}
}
