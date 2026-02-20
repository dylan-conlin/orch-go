package gates

import (
	"strings"
	"testing"
)

func TestCheckHotspot_NilChecker(t *testing.T) {
	result, err := CheckHotspot("/some/dir", "some task", "feature-impl", false, false, nil)
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
	result, err := CheckHotspot("", "some task", "feature-impl", false, false, checker)
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
	result, err := CheckHotspot("/some/dir", "some task", "feature-impl", false, false, checker)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("CheckHotspot with no hotspots returned non-nil result")
	}
}

func TestCheckHotspot_WithHotspots_NonCritical(t *testing.T) {
	checker := func(dir, task string) (*HotspotResult, error) {
		return &HotspotResult{
			HasHotspots:        true,
			HasCriticalHotspot: false,
			Warning:            "test warning",
		}, nil
	}
	result, err := CheckHotspot("/some/dir", "some task", "feature-impl", false, false, checker)
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

func TestCheckHotspot_DaemonDrivenSilent(t *testing.T) {
	checker := func(dir, task string) (*HotspotResult, error) {
		return &HotspotResult{
			HasHotspots: true,
			Warning:     "test warning",
		}, nil
	}
	result, err := CheckHotspot("/some/dir", "some task", "feature-impl", true, false, checker)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("CheckHotspot daemon-driven returned nil")
	}
	if !result.HasHotspots {
		t.Error("expected HasHotspots to be true")
	}
}

func TestCheckHotspot_CriticalBlocksFeatureImpl(t *testing.T) {
	checker := func(dir, task string) (*HotspotResult, error) {
		return &HotspotResult{
			HasHotspots:        true,
			HasCriticalHotspot: true,
			Warning:            "CRITICAL hotspot warning",
			CriticalFiles:      []string{"cmd/orch/main.go"},
		}, nil
	}
	_, err := CheckHotspot("/some/dir", "fix cmd/orch/main.go", "feature-impl", false, false, checker)
	if err == nil {
		t.Fatal("expected error for CRITICAL hotspot with feature-impl skill")
	}
	if !strContains(err.Error(), "CRITICAL hotspot") {
		t.Errorf("error should mention CRITICAL hotspot, got: %v", err)
	}
	if !strContains(err.Error(), "--force-hotspot") {
		t.Errorf("error should mention --force-hotspot flag, got: %v", err)
	}
}

func TestCheckHotspot_CriticalBlocksSystematicDebugging(t *testing.T) {
	checker := func(dir, task string) (*HotspotResult, error) {
		return &HotspotResult{
			HasHotspots:        true,
			HasCriticalHotspot: true,
			Warning:            "CRITICAL hotspot warning",
			CriticalFiles:      []string{"pkg/big/file.go"},
		}, nil
	}
	_, err := CheckHotspot("/some/dir", "debug pkg/big/file.go", "systematic-debugging", false, false, checker)
	if err == nil {
		t.Fatal("expected error for CRITICAL hotspot with systematic-debugging skill")
	}
}

func TestCheckHotspot_CriticalExemptsArchitect(t *testing.T) {
	checker := func(dir, task string) (*HotspotResult, error) {
		return &HotspotResult{
			HasHotspots:        true,
			HasCriticalHotspot: true,
			Warning:            "CRITICAL hotspot warning",
			CriticalFiles:      []string{"cmd/orch/main.go"},
		}, nil
	}
	result, err := CheckHotspot("/some/dir", "review cmd/orch/main.go", "architect", false, false, checker)
	if err != nil {
		t.Fatalf("architect should be exempt from CRITICAL hotspot blocking, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected result for architect with CRITICAL hotspot")
	}
}

func TestCheckHotspot_CriticalExemptsInvestigation(t *testing.T) {
	checker := func(dir, task string) (*HotspotResult, error) {
		return &HotspotResult{
			HasHotspots:        true,
			HasCriticalHotspot: true,
			Warning:            "CRITICAL hotspot warning",
			CriticalFiles:      []string{"cmd/orch/main.go"},
		}, nil
	}
	_, err := CheckHotspot("/some/dir", "investigate cmd/orch/main.go", "investigation", false, false, checker)
	if err != nil {
		t.Fatalf("investigation should be exempt from CRITICAL hotspot blocking, got: %v", err)
	}
}

func TestCheckHotspot_CriticalExemptsCaptureKnowledge(t *testing.T) {
	checker := func(dir, task string) (*HotspotResult, error) {
		return &HotspotResult{
			HasHotspots:        true,
			HasCriticalHotspot: true,
			Warning:            "CRITICAL hotspot warning",
			CriticalFiles:      []string{"cmd/orch/main.go"},
		}, nil
	}
	_, err := CheckHotspot("/some/dir", "capture knowledge about main.go", "capture-knowledge", false, false, checker)
	if err != nil {
		t.Fatalf("capture-knowledge should be exempt, got: %v", err)
	}
}

func TestCheckHotspot_CriticalExemptsCodebaseAudit(t *testing.T) {
	checker := func(dir, task string) (*HotspotResult, error) {
		return &HotspotResult{
			HasHotspots:        true,
			HasCriticalHotspot: true,
			Warning:            "CRITICAL hotspot warning",
			CriticalFiles:      []string{"cmd/orch/main.go"},
		}, nil
	}
	_, err := CheckHotspot("/some/dir", "audit codebase", "codebase-audit", false, false, checker)
	if err != nil {
		t.Fatalf("codebase-audit should be exempt, got: %v", err)
	}
}

func TestCheckHotspot_ForceHotspotBypassesBlock(t *testing.T) {
	checker := func(dir, task string) (*HotspotResult, error) {
		return &HotspotResult{
			HasHotspots:        true,
			HasCriticalHotspot: true,
			Warning:            "CRITICAL hotspot warning",
			CriticalFiles:      []string{"cmd/orch/main.go"},
		}, nil
	}
	result, err := CheckHotspot("/some/dir", "fix cmd/orch/main.go", "feature-impl", false, true, checker)
	if err != nil {
		t.Fatalf("--force-hotspot should bypass CRITICAL block, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected result when force-hotspot bypasses block")
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
		{"code-review", false},
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

func strContains(s, substr string) bool {
	return strings.Contains(s, substr)
}
