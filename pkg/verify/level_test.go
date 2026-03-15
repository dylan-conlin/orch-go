package verify

import (
	"os"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

func TestGatesForLevel_V0(t *testing.T) {
	gates := GatesForLevel(spawn.VerifyV0)
	// V0 (Acknowledge): only Phase Complete
	expected := map[string]bool{
		GatePhaseComplete: true,
	}
	assertGateSet(t, "V0", gates, expected)
}

func TestGatesForLevel_V1(t *testing.T) {
	gates := GatesForLevel(spawn.VerifyV1)
	// V1 (Artifacts): V0 + Handoff Content, Skill Output, Phase Gates, Constraint, Decision Patch Limit, Architectural Choices, Probe Model Merge, Architect Handoff
	expected := map[string]bool{
		GatePhaseComplete:        true,
		GateHandoffContent:       true,
		GateSkillOutput:          true,
		GatePhaseGate:            true,
		GateConstraint:           true,
		GateDecisionPatchLimit:   true,
		GateArchitecturalChoices: true,
		GateProbeModelMerge:      true,
		GateArchitectHandoff:     true,
		GateArtifact:             true,
	}
	assertGateSet(t, "V1", gates, expected)
}

func TestGatesForLevel_V2(t *testing.T) {
	gates := GatesForLevel(spawn.VerifyV2)
	// V2 (Evidence): V1 + Test Evidence, Git Diff, Build, Vet, Accretion
	expected := map[string]bool{
		GatePhaseComplete:        true,
		GateSynthesis:            true,
		GateHandoffContent:       true,
		GateSkillOutput:          true,
		GatePhaseGate:            true,
		GateConstraint:           true,
		GateDecisionPatchLimit:   true,
		GateArchitecturalChoices: true,
		GateProbeModelMerge:      true,
		GateArchitectHandoff:     true,
		GateArtifact:             true,
		GateTestEvidence:         true,
		GateGitDiff:              true,
		GateBuild:                true,
		GateVet:                  true,
		GateAccretion:            true,
	}
	assertGateSet(t, "V2", gates, expected)
}

func TestGatesForLevel_V3(t *testing.T) {
	gates := GatesForLevel(spawn.VerifyV3)
	// V3 (Behavioral): V2 + Visual Verification, Explain-Back
	expected := map[string]bool{
		GatePhaseComplete:        true,
		GateSynthesis:            true,
		GateHandoffContent:       true,
		GateSkillOutput:          true,
		GatePhaseGate:            true,
		GateConstraint:           true,
		GateDecisionPatchLimit:   true,
		GateArchitecturalChoices: true,
		GateProbeModelMerge:      true,
		GateArchitectHandoff:     true,
		GateArtifact:             true,
		GateTestEvidence:         true,
		GateGitDiff:              true,
		GateBuild:                true,
		GateVet:                  true,
		GateAccretion:            true,
		GateVisualVerify:         true,
		GateExplainBack:          true,
	}
	assertGateSet(t, "V3", gates, expected)
}

func TestGatesForLevel_Superset(t *testing.T) {
	// Each level must be a strict superset of the level below
	v0 := GatesForLevel(spawn.VerifyV0)
	v1 := GatesForLevel(spawn.VerifyV1)
	v2 := GatesForLevel(spawn.VerifyV2)
	v3 := GatesForLevel(spawn.VerifyV3)

	v0Set := toSet(v0)
	v1Set := toSet(v1)
	v2Set := toSet(v2)
	v3Set := toSet(v3)

	// V1 must contain all V0 gates
	for gate := range v0Set {
		if !v1Set[gate] {
			t.Errorf("V1 missing V0 gate %q", gate)
		}
	}
	if len(v1Set) <= len(v0Set) {
		t.Error("V1 must be a strict superset of V0")
	}

	// V2 must contain all V1 gates
	for gate := range v1Set {
		if !v2Set[gate] {
			t.Errorf("V2 missing V1 gate %q", gate)
		}
	}
	if len(v2Set) <= len(v1Set) {
		t.Error("V2 must be a strict superset of V1")
	}

	// V3 must contain all V2 gates
	for gate := range v2Set {
		if !v3Set[gate] {
			t.Errorf("V3 missing V2 gate %q", gate)
		}
	}
	if len(v3Set) <= len(v2Set) {
		t.Error("V3 must be a strict superset of V2")
	}
}

func TestGatesForLevel_UnknownDefaultsV1(t *testing.T) {
	// Unknown/empty level defaults to V1 (conservative)
	gates := GatesForLevel("")
	v1Gates := GatesForLevel(spawn.VerifyV1)
	if len(gates) != len(v1Gates) {
		t.Errorf("empty level gates count = %d, want %d (V1 default)", len(gates), len(v1Gates))
	}
}

func TestShouldRunGate(t *testing.T) {
	tests := []struct {
		level string
		gate  string
		want  bool
	}{
		// V0 only runs phase_complete
		{spawn.VerifyV0, GatePhaseComplete, true},
		{spawn.VerifyV0, GateSynthesis, false},
		{spawn.VerifyV0, GateTestEvidence, false},
		{spawn.VerifyV0, GateVisualVerify, false},

		// V1 runs artifact gates but not synthesis or evidence gates
		{spawn.VerifyV1, GatePhaseComplete, true},
		{spawn.VerifyV1, GateSynthesis, false},
		{spawn.VerifyV1, GateConstraint, true},
		{spawn.VerifyV1, GateTestEvidence, false},
		{spawn.VerifyV1, GateBuild, false},
		{spawn.VerifyV1, GateVisualVerify, false},

		// V2 runs evidence gates but not behavioral
		{spawn.VerifyV2, GateTestEvidence, true},
		{spawn.VerifyV2, GateBuild, true},
		{spawn.VerifyV2, GateGitDiff, true},
		{spawn.VerifyV2, GateVisualVerify, false},
		{spawn.VerifyV2, GateExplainBack, false},

		// V3 runs all gates
		{spawn.VerifyV3, GateVisualVerify, true},
		{spawn.VerifyV3, GateExplainBack, true},
		{spawn.VerifyV3, GateTestEvidence, true},
		{spawn.VerifyV3, GatePhaseComplete, true},
	}

	for _, tt := range tests {
		t.Run(tt.level+"_"+tt.gate, func(t *testing.T) {
			got := ShouldRunGate(tt.level, tt.gate)
			if got != tt.want {
				t.Errorf("ShouldRunGate(%q, %q) = %v, want %v", tt.level, tt.gate, got, tt.want)
			}
		})
	}
}

func TestReadReviewTierFromWorkspace_ManifestHasReviewTier(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "review-tier-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manifest := spawn.AgentManifest{
		WorkspaceName: "og-feat-test",
		Skill:         "investigation",
		ReviewTier:    spawn.ReviewDeep,
	}
	if err := spawn.WriteAgentManifest(tmpDir, manifest); err != nil {
		t.Fatalf("WriteAgentManifest failed: %v", err)
	}

	got := ReadReviewTierFromWorkspace(tmpDir)
	if got != spawn.ReviewDeep {
		t.Errorf("ReadReviewTierFromWorkspace() = %q, want %q", got, spawn.ReviewDeep)
	}
}

func TestReadReviewTierFromWorkspace_FallbackToSkill(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "review-tier-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Manifest without ReviewTier — should infer from skill
	manifest := spawn.AgentManifest{
		WorkspaceName: "og-feat-test",
		Skill:         "feature-impl",
	}
	if err := spawn.WriteAgentManifest(tmpDir, manifest); err != nil {
		t.Fatalf("WriteAgentManifest failed: %v", err)
	}

	got := ReadReviewTierFromWorkspace(tmpDir)
	if got != spawn.ReviewReview {
		t.Errorf("ReadReviewTierFromWorkspace() = %q, want %q (inferred from feature-impl)", got, spawn.ReviewReview)
	}
}

func TestReadReviewTierFromWorkspace_NoManifest(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "review-tier-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// No manifest at all — conservative default
	got := ReadReviewTierFromWorkspace(tmpDir)
	if got != spawn.ReviewReview {
		t.Errorf("ReadReviewTierFromWorkspace() = %q, want %q (conservative default)", got, spawn.ReviewReview)
	}
}

// helper: convert slice to set
func toSet(gates []string) map[string]bool {
	s := make(map[string]bool)
	for _, g := range gates {
		s[g] = true
	}
	return s
}

// helper: assert gate set matches expected
func assertGateSet(t *testing.T, level string, gates []string, expected map[string]bool) {
	t.Helper()
	actual := toSet(gates)

	for gate := range expected {
		if !actual[gate] {
			t.Errorf("%s: missing expected gate %q", level, gate)
		}
	}
	for gate := range actual {
		if !expected[gate] {
			t.Errorf("%s: unexpected gate %q", level, gate)
		}
	}
}
