package gates

import (
	"fmt"
	"testing"
)

func TestCheckHotspot_NilChecker(t *testing.T) {
	result, err := CheckHotspot("/some/dir", "some task", "feature-impl", false, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("CheckHotspot with nil checker returned non-nil result")
	}
}

func TestCheckHotspot_EmptyDir(t *testing.T) {
	checker := func(dir, task string) (*HotspotResult, error) {
		t.Fatal("checker should not be called with empty dir")
		return nil, nil
	}
	result, err := CheckHotspot("", "some task", "feature-impl", false, checker)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("CheckHotspot with empty dir returned non-nil result")
	}
}

func TestCheckHotspot_NoHotspots(t *testing.T) {
	checker := func(dir, task string) (*HotspotResult, error) {
		return nil, nil
	}
	result, err := CheckHotspot("/some/dir", "some task", "feature-impl", false, checker)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("CheckHotspot with no hotspots returned non-nil result")
	}
}

func TestCheckHotspot_NonCritical_Advisory(t *testing.T) {
	checker := func(dir, task string) (*HotspotResult, error) {
		return &HotspotResult{
			HasHotspots:        true,
			HasCriticalHotspot: false,
			Warning:            "test warning",
		}, nil
	}
	result, err := CheckHotspot("/some/dir", "some task", "feature-impl", false, checker)
	if err != nil {
		t.Fatalf("unexpected error for non-critical hotspot: %v", err)
	}
	if result == nil {
		t.Fatal("CheckHotspot with hotspots returned nil")
	}
	if !result.HasHotspots {
		t.Error("expected HasHotspots to be true")
	}
}

func TestCheckHotspot_Critical_NeverBlocks(t *testing.T) {
	// Advisory gate: CRITICAL hotspot with blocking skill should NOT return error
	checker := func(dir, task string) (*HotspotResult, error) {
		return &HotspotResult{
			HasHotspots:        true,
			HasCriticalHotspot: true,
			Warning:            "CRITICAL hotspot warning",
			CriticalFiles:      []string{"cmd/orch/main.go"},
		}, nil
	}
	result, err := CheckHotspot("/some/dir", "fix cmd/orch/main.go", "feature-impl", false, checker)
	if err != nil {
		t.Fatalf("advisory gate should never block, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected result for advisory CRITICAL hotspot")
	}
	if !result.HasCriticalHotspot {
		t.Error("expected HasCriticalHotspot to be true")
	}
}

func TestCheckHotspot_Critical_NeverBlocks_SystematicDebugging(t *testing.T) {
	checker := func(dir, task string) (*HotspotResult, error) {
		return &HotspotResult{
			HasHotspots:        true,
			HasCriticalHotspot: true,
			Warning:            "CRITICAL hotspot warning",
			CriticalFiles:      []string{"pkg/big/file.go"},
		}, nil
	}
	result, err := CheckHotspot("/some/dir", "debug pkg/big/file.go", "systematic-debugging", false, checker)
	if err != nil {
		t.Fatalf("advisory gate should never block, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected result")
	}
}

func TestCheckHotspot_DaemonDrivenSilent(t *testing.T) {
	checker := func(dir, task string) (*HotspotResult, error) {
		return &HotspotResult{
			HasHotspots:        true,
			HasCriticalHotspot: true,
			Warning:            "CRITICAL hotspot warning",
			CriticalFiles:      []string{"cmd/orch/main.go"},
		}, nil
	}
	result, err := CheckHotspot("/some/dir", "fix cmd/orch/main.go", "feature-impl", true, checker)
	if err != nil {
		t.Fatalf("daemon-driven should never error, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected result for daemon-driven")
	}
}

func TestCheckHotspot_ExemptSkills(t *testing.T) {
	checker := func(dir, task string) (*HotspotResult, error) {
		return &HotspotResult{
			HasHotspots:        true,
			HasCriticalHotspot: true,
			Warning:            "CRITICAL hotspot warning",
			CriticalFiles:      []string{"cmd/orch/main.go"},
		}, nil
	}
	exemptSkills := []string{"architect", "investigation", "capture-knowledge", "codebase-audit"}
	for _, skill := range exemptSkills {
		t.Run(skill, func(t *testing.T) {
			result, err := CheckHotspot("/some/dir", "task", skill, false, checker)
			if err != nil {
				t.Fatalf("%s should not error, got: %v", skill, err)
			}
			if result == nil {
				t.Fatalf("expected result for %s", skill)
			}
		})
	}
}

func TestCheckHotspot_CheckerReturnsError(t *testing.T) {
	checker := func(dir, task string) (*HotspotResult, error) {
		return nil, fmt.Errorf("hotspot analysis failed")
	}
	result, err := CheckHotspot("/some/dir", "some task", "feature-impl", false, checker)
	if err != nil {
		t.Errorf("expected nil error on checker failure, got: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result on checker failure, got: %v", result)
	}
}

func TestCheckHotspot_ResultFieldsPreserved(t *testing.T) {
	checker := func(dir, task string) (*HotspotResult, error) {
		return &HotspotResult{
			HasHotspots:        true,
			HasCriticalHotspot: false,
			Warning:            "test warning",
			CriticalFiles:      []string{},
			MatchedFiles:       []string{"pkg/spawn/context.go", "pkg/spawn/config.go"},
		}, nil
	}
	result, err := CheckHotspot("/some/dir", "update spawn", "feature-impl", false, checker)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.MatchedFiles) != 2 {
		t.Errorf("expected 2 matched files, got %d", len(result.MatchedFiles))
	}
}

func TestIsBlockingSkill(t *testing.T) {
	tests := []struct {
		skill    string
		blocking bool
	}{
		{"feature-impl", true},
		{"systematic-debugging", true},
		{"architect", false},
		{"investigation", false},
		{"capture-knowledge", false},
		{"codebase-audit", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.skill, func(t *testing.T) {
			got := IsBlockingSkill(tt.skill)
			if got != tt.blocking {
				t.Errorf("IsBlockingSkill(%q) = %v, want %v", tt.skill, got, tt.blocking)
			}
		})
	}
}
