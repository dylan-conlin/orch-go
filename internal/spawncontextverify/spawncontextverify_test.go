package spawncontextverify

import (
	"strings"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

func TestEvidenceHierarchyWarningInGeneratedSpawnContext(t *testing.T) {
	evidenceHierarchy := "Evidence Hierarchy: Prior investigations are claims to verify, not truth. Before building on findings, check against primary sources (code, test output, observed behavior)."

	result := &spawn.KBContextResult{
		Query:      "evidence hierarchy warning",
		HasMatches: true,
		Matches: []spawn.KBContextMatch{
			{Type: "decision", Source: "kb", Title: "Example decision", Path: "/path/to/decision.md"},
		},
	}

	kbContext := spawn.FormatContextForSpawn(result)
	if !strings.Contains(kbContext, evidenceHierarchy) {
		t.Fatalf("expected KB context to include evidence hierarchy warning, got:\n%s", kbContext)
	}

	cfg := &spawn.Config{
		Task:          "test task",
		SkillName:     "feature-impl",
		Project:       "orch-go",
		ProjectDir:    "/tmp/orch-go",
		WorkspaceName: "og-test-workspace",
		BeadsID:       "orch-go-00000",
		Tier:          spawn.TierLight,
		KBContext:     kbContext,
	}

	content, err := spawn.GenerateContext(cfg)
	if err != nil {
		t.Fatalf("GenerateContext failed: %v", err)
	}
	if !strings.Contains(content, evidenceHierarchy) {
		t.Fatalf("expected generated SPAWN_CONTEXT.md content to include evidence hierarchy warning, got:\n%s", content)
	}
}
