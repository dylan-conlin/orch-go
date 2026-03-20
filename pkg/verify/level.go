package verify

import "github.com/dylan-conlin/orch-go/pkg/spawn"

// gatesByLevel defines which gates are introduced at each level.
// Each level is additive — gates from lower levels are always included.
var gatesByLevel = map[string][]string{
	spawn.VerifyV0: {
		GatePhaseComplete,
	},
	spawn.VerifyV1: {
		GateHandoffContent,
		GateDecisionPatchLimit,
		GateArchitecturalChoices,
		GateProbeModelMerge,
		GateArchitectHandoff,
		GateConsequenceSensor,
		GateArtifact,
	},
	spawn.VerifyV2: {
		GateSynthesis,
		GateTestEvidence,
		GateGitDiff,
		GateBuild,
		GateVet,
		GateAccretion,
	},
	spawn.VerifyV3: {
		GateVisualVerify,
		GateExplainBack,
	},
}

// levelOrder is the ordered list of levels for iteration.
var levelOrder = []string{
	spawn.VerifyV0,
	spawn.VerifyV1,
	spawn.VerifyV2,
	spawn.VerifyV3,
}

// GatesForLevel returns the list of gates that should fire for the given verification level.
// Each level includes all gates from lower levels (strict superset property).
// Unknown/empty levels default to V1 (conservative).
func GatesForLevel(level string) []string {
	if !spawn.IsValidVerifyLevel(level) {
		level = spawn.VerifyV1 // Conservative default
	}

	var gates []string
	for _, l := range levelOrder {
		if additions, ok := gatesByLevel[l]; ok {
			gates = append(gates, additions...)
		}
		if l == level {
			break
		}
	}
	return gates
}

// ShouldRunGate returns true if the given gate should be executed at the given verification level.
// This is the primary query function used by the verification pipeline.
func ShouldRunGate(level, gate string) bool {
	gates := GatesForLevel(level)
	for _, g := range gates {
		if g == gate {
			return true
		}
	}
	return false
}

// ReadVerifyLevelFromWorkspace reads the verification level from the workspace manifest.
// Falls back to inferring from skill name if not set in manifest.
// Returns VerifyV1 as the conservative default.
func ReadVerifyLevelFromWorkspace(workspacePath string) string {
	manifest := spawn.ReadAgentManifestWithFallback(workspacePath)
	if manifest.VerifyLevel != "" {
		return manifest.VerifyLevel
	}

	// Fallback: infer from skill name (for pre-V0-V3 workspaces)
	if manifest.Skill != "" {
		return spawn.DefaultVerifyLevel(manifest.Skill, "")
	}

	return spawn.VerifyV1 // Conservative default
}

// ReadReviewTierFromWorkspace reads the review tier from the workspace manifest.
// Falls back to inferring from skill name if not set in manifest.
// Returns ReviewReview as the conservative default.
func ReadReviewTierFromWorkspace(workspacePath string) string {
	manifest := spawn.ReadAgentManifestWithFallback(workspacePath)
	if manifest.ReviewTier != "" {
		return manifest.ReviewTier
	}

	// Fallback: infer from skill name (for pre-review-tier workspaces)
	if manifest.Skill != "" {
		return spawn.DefaultReviewTier(manifest.Skill, "")
	}

	return spawn.ReviewReview // Conservative default
}
