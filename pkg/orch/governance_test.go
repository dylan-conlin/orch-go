package orch

import (
	"testing"
)

func TestGovernanceProtectedPaths(t *testing.T) {
	paths := GovernanceProtectedPaths()
	if len(paths) == 0 {
		t.Fatal("expected non-empty governance protected paths")
	}

	found := map[string]bool{}
	for _, p := range paths {
		found[p.Pattern] = true
	}
	for _, expected := range []string{"pkg/spawn/gates/", "_precommit.go", "pkg/verify/accretion.go"} {
		if !found[expected] {
			t.Errorf("expected protected path %q not found", expected)
		}
	}
}

func TestCheckGovernance_NoMatch(t *testing.T) {
	result := CheckGovernance("implement a new feature for the dashboard", "feature-impl", false)
	if result != nil {
		t.Errorf("expected nil result for task with no governance paths, got %+v", result)
	}
}

func TestCheckGovernance_MatchesGatesPath(t *testing.T) {
	result := CheckGovernance("edit pkg/spawn/gates/hotspot.go to fix the blocking logic", "feature-impl", false)
	if result == nil {
		t.Fatal("expected governance warning for task mentioning gates path")
	}
	if len(result.MatchedPaths) == 0 {
		t.Error("expected at least one matched path")
	}
	if result.MatchedPaths[0].Pattern != "pkg/spawn/gates/" {
		t.Errorf("expected pkg/spawn/gates/ match, got %s", result.MatchedPaths[0].Pattern)
	}
}

func TestCheckGovernance_MatchesVerifyPrecommit(t *testing.T) {
	result := CheckGovernance("modify pkg/verify/accretion_precommit.go", "feature-impl", false)
	if result == nil {
		t.Fatal("expected governance warning for task mentioning precommit file")
	}
	if len(result.MatchedPaths) == 0 {
		t.Error("expected at least one matched path")
	}
}

func TestCheckGovernance_MatchesVerifyAccretion(t *testing.T) {
	result := CheckGovernance("modify pkg/verify/accretion.go to change gate", "feature-impl", false)
	if result == nil {
		t.Fatal("expected governance warning for task mentioning accretion.go")
	}
}

func TestCheckGovernance_NoMatchUnprotectedVerify(t *testing.T) {
	result := CheckGovernance("modify pkg/verify/check.go to update verification", "feature-impl", false)
	if result != nil {
		t.Errorf("expected no governance warning for unprotected pkg/verify/check.go, got %+v", result)
	}
}

func TestCheckGovernance_MatchesMultiplePaths(t *testing.T) {
	result := CheckGovernance("update pkg/spawn/gates/triage.go and pkg/verify/accretion.go", "feature-impl", false)
	if result == nil {
		t.Fatal("expected governance warning")
	}
	if len(result.MatchedPaths) < 2 {
		t.Errorf("expected at least 2 matched paths, got %d", len(result.MatchedPaths))
	}
}

func TestCheckGovernance_DaemonDrivenSilent(t *testing.T) {
	result := CheckGovernance("edit pkg/spawn/gates/hotspot.go", "feature-impl", true)
	if result == nil {
		t.Fatal("expected governance result even for daemon-driven")
	}
	if len(result.MatchedPaths) == 0 {
		t.Error("expected matches for daemon-driven")
	}
}

func TestCheckGovernance_SkillsDir(t *testing.T) {
	result := CheckGovernance("modify skills/src/shared/worker-base to add new protocol", "feature-impl", false)
	if result == nil {
		t.Fatal("expected governance warning for skills path")
	}
}

func TestCheckGovernance_CaseInsensitive(t *testing.T) {
	result := CheckGovernance("edit PKG/SPAWN/GATES/hotspot.go", "feature-impl", false)
	if result == nil {
		t.Fatal("expected governance warning for case-insensitive match")
	}
}

func TestCheckGovernance_WarningMessage(t *testing.T) {
	result := CheckGovernance("edit pkg/spawn/gates/hotspot.go", "feature-impl", false)
	if result == nil {
		t.Fatal("expected governance result")
	}
	if result.Warning == "" {
		t.Error("expected non-empty warning message")
	}
}

func TestCheckGovernance_HooksPath(t *testing.T) {
	result := CheckGovernance("update .orch/hooks/gate-governance-file-protection.py", "feature-impl", false)
	if result == nil {
		t.Fatal("expected governance warning for hooks path")
	}
}

func TestCheckGovernance_PrecommitPath(t *testing.T) {
	result := CheckGovernance("fix scripts/pre-commit-growth-gate.sh", "feature-impl", false)
	if result == nil {
		t.Fatal("expected governance warning for pre-commit scripts")
	}
}
