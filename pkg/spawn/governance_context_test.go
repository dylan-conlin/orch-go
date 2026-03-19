package spawn

import (
	"strings"
	"testing"
)

func TestGenerateGovernanceContext(t *testing.T) {
	ctx := GenerateGovernanceContext(false)

	// Should contain header
	if !strings.Contains(ctx, "## GOVERNANCE-PROTECTED PATHS") {
		t.Error("missing governance header")
	}

	// Should list all protected paths
	for _, p := range GovernanceProtectedPaths {
		if !strings.Contains(ctx, p.Pattern) {
			t.Errorf("missing protected path: %s", p.Pattern)
		}
		if !strings.Contains(ctx, p.Description) {
			t.Errorf("missing description for: %s", p.Pattern)
		}
	}

	// Should include escalation action with beads reference
	if !strings.Contains(ctx, "bd comments add") {
		t.Error("missing beads escalation action for tracked spawn")
	}
}

func TestGenerateGovernanceContext_NoTrack(t *testing.T) {
	ctx := GenerateGovernanceContext(true)

	// Should NOT contain bd comment references
	if strings.Contains(ctx, "`bd comment") || strings.Contains(ctx, "`bd comments") {
		t.Error("noTrack governance context should not reference bd commands")
	}

	// Should still list protected paths
	if !strings.Contains(ctx, "pkg/spawn/gates/*") {
		t.Error("missing protected paths in noTrack governance context")
	}

	// Should have alternative escalation action
	if !strings.Contains(ctx, "Document in your investigation file") {
		t.Error("missing noTrack escalation action")
	}
}

func TestGovernanceContextInSpawnTemplate(t *testing.T) {
	cfg := &Config{
		Task:       "test task",
		SkillName:  "feature-impl",
		ProjectDir: "/tmp/test-project",
	}

	content, err := GenerateContext(cfg)
	if err != nil {
		t.Fatalf("GenerateContext failed: %v", err)
	}

	// Governance section should appear in generated context
	if !strings.Contains(content, "GOVERNANCE-PROTECTED PATHS") {
		t.Error("governance context missing from generated SPAWN_CONTEXT")
	}

	// Should list key protected paths
	if !strings.Contains(content, "pkg/spawn/gates/*") {
		t.Error("missing pkg/spawn/gates/* in governance context")
	}
	if !strings.Contains(content, "pkg/verify/precommit.go") {
		t.Error("missing pkg/verify/precommit.go in governance context")
	}
}
