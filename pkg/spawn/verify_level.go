package spawn

// Verification level constants.
// Each level is a strict superset of the level below.
// Declared at spawn time, determines which gates fire at completion time.
const (
	VerifyV0 = "V0" // Acknowledge: Phase Complete only
	VerifyV1 = "V1" // Artifacts: V0 + deliverable/constraint checks
	VerifyV2 = "V2" // Evidence: V1 + test evidence, build, git diff
	VerifyV3 = "V3" // Behavioral: V2 + visual verification, explain-back
)

// verifyLevelOrder maps levels to their numeric order for comparison.
var verifyLevelOrder = map[string]int{
	VerifyV0: 0,
	VerifyV1: 1,
	VerifyV2: 2,
	VerifyV3: 3,
}

// SkillVerifyLevelDefaults maps skills to their default verification level.
var SkillVerifyLevelDefaults = map[string]string{
	// V0: Acknowledge — minimal verification
	"issue-creation":    VerifyV0,
	"capture-knowledge": VerifyV0,

	// V1: Artifacts — knowledge-producing skills
	"investigation":  VerifyV1,
	"architect":      VerifyV1,
	"research":       VerifyV1,
	"codebase-audit": VerifyV1,
	"design-session": VerifyV1,
	"probe":          VerifyV1,
	"ux-audit":                  VerifyV1,
	"exploration-orchestrator":  VerifyV1,

	// V2: Evidence — implementation-focused skills
	// Note: feature-impl defaults to V2 here, but light-tier spawns cap this
	// to V0 via VerifyLevelForTier (see TierMaxVerifyLevel). This means
	// light-tier feature-impl gets acknowledge-only verification, skipping
	// test evidence and synthesis gates that full-tier feature-impl requires.
	"feature-impl":         VerifyV2,
	"systematic-debugging": VerifyV2,
	"reliability-testing":  VerifyV2,

	// V3: Behavioral — visual/interactive verification required
	"debug-with-playwright": VerifyV3,
}

// IssueTypeMinVerifyLevel maps issue types to their minimum verification level.
// The actual level is max(skill_level, issue_type_minimum).
var IssueTypeMinVerifyLevel = map[string]string{
	"feature":       VerifyV2,
	"bug":           VerifyV2,
	"decision":      VerifyV2,
	"investigation": VerifyV1,
	"probe":         VerifyV1,
	// task, question: no minimum (empty string)
}

// DefaultVerifyLevel returns the default verification level for a skill and issue type.
// The level is max(skill_default, issue_type_minimum).
// Returns VerifyV1 for unknown skills (conservative default).
func DefaultVerifyLevel(skillName, issueType string) string {
	skillLevel, ok := SkillVerifyLevelDefaults[skillName]
	if !ok {
		skillLevel = VerifyV1 // Conservative default for unknown skills
	}

	issueMin, ok := IssueTypeMinVerifyLevel[issueType]
	if !ok {
		return skillLevel // No issue type minimum
	}

	return MaxVerifyLevel(skillLevel, issueMin)
}

// CompareVerifyLevels compares two verification levels.
// Returns -1 if a < b, 0 if equal, 1 if a > b.
// Unknown levels are treated as V1.
func CompareVerifyLevels(a, b string) int {
	orderA := levelToOrder(a)
	orderB := levelToOrder(b)
	if orderA < orderB {
		return -1
	}
	if orderA > orderB {
		return 1
	}
	return 0
}

// MaxVerifyLevel returns the higher of two verification levels.
func MaxVerifyLevel(a, b string) string {
	if CompareVerifyLevels(a, b) >= 0 {
		return a
	}
	return b
}

// TierMaxVerifyLevel maps spawn tiers to their maximum allowed verification level.
// Light tier caps at V0 (acknowledge only) since light spawns skip synthesis/knowledge gates.
var TierMaxVerifyLevel = map[string]string{
	TierLight: VerifyV0, // Light tier: minimal verification, no synthesis
	// TierFull: no cap (full verification applies)
}

// VerifyLevelForTier returns the effective verify level after applying tier-based capping.
// If the tier has a maximum level defined, the result is min(skillLevel, tierMax).
// Unknown or empty tiers do not cap (returns skillLevel unchanged).
func VerifyLevelForTier(tier, skillLevel string) string {
	tierMax, ok := TierMaxVerifyLevel[tier]
	if !ok {
		return skillLevel // No cap for this tier
	}
	if CompareVerifyLevels(skillLevel, tierMax) > 0 {
		return tierMax
	}
	return skillLevel
}

// IsValidVerifyLevel returns true if the level string is a valid verification level.
func IsValidVerifyLevel(level string) bool {
	_, ok := verifyLevelOrder[level]
	return ok
}

// levelToOrder returns the numeric order for a level.
// Unknown levels default to V1 order (conservative).
func levelToOrder(level string) int {
	if order, ok := verifyLevelOrder[level]; ok {
		return order
	}
	return verifyLevelOrder[VerifyV1] // Conservative default
}
