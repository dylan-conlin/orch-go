package gates

import (
	"fmt"
	"strings"
	"testing"
)

func TestCheckHotspot_NilChecker(t *testing.T) {
	result, err := CheckHotspot("/some/dir", "some task", "feature-impl", false, false, "", nil, nil)
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
	result, err := CheckHotspot("", "some task", "feature-impl", false, false, "", checker, nil)
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
	result, err := CheckHotspot("/some/dir", "some task", "feature-impl", false, false, "", checker, nil)
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
	result, err := CheckHotspot("/some/dir", "some task", "feature-impl", false, false, "", checker, nil)
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
	result, err := CheckHotspot("/some/dir", "some task", "feature-impl", true, false, "", checker, nil)
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
	_, err := CheckHotspot("/some/dir", "fix cmd/orch/main.go", "feature-impl", false, false, "", checker, nil)
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
	_, err := CheckHotspot("/some/dir", "debug pkg/big/file.go", "systematic-debugging", false, false, "", checker, nil)
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
	result, err := CheckHotspot("/some/dir", "review cmd/orch/main.go", "architect", false, false, "", checker, nil)
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
	_, err := CheckHotspot("/some/dir", "investigate cmd/orch/main.go", "investigation", false, false, "", checker, nil)
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
	_, err := CheckHotspot("/some/dir", "capture knowledge about main.go", "capture-knowledge", false, false, "", checker, nil)
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
	_, err := CheckHotspot("/some/dir", "audit codebase", "codebase-audit", false, false, "", checker, nil)
	if err != nil {
		t.Fatalf("codebase-audit should be exempt, got: %v", err)
	}
}

// --- Architect-ref verification tests ---

func TestCheckHotspot_ForceHotspotRequiresArchitectRef(t *testing.T) {
	checker := func(dir, task string) (*HotspotResult, error) {
		return &HotspotResult{
			HasHotspots:        true,
			HasCriticalHotspot: true,
			Warning:            "CRITICAL hotspot warning",
			CriticalFiles:      []string{"cmd/orch/main.go"},
		}, nil
	}
	// forceHotspot=true but no architect ref → should error
	_, err := CheckHotspot("/some/dir", "fix cmd/orch/main.go", "feature-impl", false, true, "", checker, nil)
	if err == nil {
		t.Fatal("expected error when --force-hotspot used without --architect-ref")
	}
	if !strContains(err.Error(), "--architect-ref") {
		t.Errorf("error should mention --architect-ref, got: %v", err)
	}
}

func TestCheckHotspot_ForceHotspotWithArchitectRefVerified(t *testing.T) {
	checker := func(dir, task string) (*HotspotResult, error) {
		return &HotspotResult{
			HasHotspots:        true,
			HasCriticalHotspot: true,
			Warning:            "CRITICAL hotspot warning",
			CriticalFiles:      []string{"cmd/orch/main.go"},
		}, nil
	}
	// Valid verifier that returns nil (issue is closed architect)
	verifier := func(issueID string) error {
		if issueID != "orch-go-1184" {
			t.Errorf("verifier called with unexpected ID: %s", issueID)
		}
		return nil // Valid closed architect issue
	}
	result, err := CheckHotspot("/some/dir", "fix cmd/orch/main.go", "feature-impl", false, true, "orch-go-1184", checker, verifier)
	if err != nil {
		t.Fatalf("--force-hotspot with valid --architect-ref should succeed, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected result when force-hotspot bypasses block with valid architect ref")
	}
}

func TestCheckHotspot_ForceHotspotWithArchitectRefNotFound(t *testing.T) {
	checker := func(dir, task string) (*HotspotResult, error) {
		return &HotspotResult{
			HasHotspots:        true,
			HasCriticalHotspot: true,
			Warning:            "CRITICAL hotspot warning",
			CriticalFiles:      []string{"cmd/orch/main.go"},
		}, nil
	}
	verifier := func(issueID string) error {
		return fmt.Errorf("--architect-ref %s: issue not found", issueID)
	}
	_, err := CheckHotspot("/some/dir", "fix cmd/orch/main.go", "feature-impl", false, true, "orch-go-9999", checker, verifier)
	if err == nil {
		t.Fatal("expected error when architect ref issue not found")
	}
	if !strContains(err.Error(), "issue not found") {
		t.Errorf("error should mention issue not found, got: %v", err)
	}
}

func TestCheckHotspot_ForceHotspotWithArchitectRefWrongType(t *testing.T) {
	checker := func(dir, task string) (*HotspotResult, error) {
		return &HotspotResult{
			HasHotspots:        true,
			HasCriticalHotspot: true,
			Warning:            "CRITICAL hotspot warning",
			CriticalFiles:      []string{"cmd/orch/main.go"},
		}, nil
	}
	verifier := func(issueID string) error {
		return fmt.Errorf("--architect-ref %s: not an architect issue (skill=feature-impl)", issueID)
	}
	_, err := CheckHotspot("/some/dir", "fix cmd/orch/main.go", "feature-impl", false, true, "orch-go-1182", checker, verifier)
	if err == nil {
		t.Fatal("expected error when architect ref is not an architect issue")
	}
	if !strContains(err.Error(), "not an architect issue") {
		t.Errorf("error should mention not an architect issue, got: %v", err)
	}
}

func TestCheckHotspot_ForceHotspotWithArchitectRefNotClosed(t *testing.T) {
	checker := func(dir, task string) (*HotspotResult, error) {
		return &HotspotResult{
			HasHotspots:        true,
			HasCriticalHotspot: true,
			Warning:            "CRITICAL hotspot warning",
			CriticalFiles:      []string{"cmd/orch/main.go"},
		}, nil
	}
	verifier := func(issueID string) error {
		return fmt.Errorf("--architect-ref %s: architect review not complete (status=in_progress)", issueID)
	}
	_, err := CheckHotspot("/some/dir", "fix cmd/orch/main.go", "feature-impl", false, true, "orch-go-1184", checker, verifier)
	if err == nil {
		t.Fatal("expected error when architect ref issue is not closed")
	}
	if !strContains(err.Error(), "not complete") {
		t.Errorf("error should mention not complete, got: %v", err)
	}
}

func TestCheckHotspot_ForceHotspotNilVerifier(t *testing.T) {
	checker := func(dir, task string) (*HotspotResult, error) {
		return &HotspotResult{
			HasHotspots:        true,
			HasCriticalHotspot: true,
			Warning:            "CRITICAL hotspot warning",
			CriticalFiles:      []string{"cmd/orch/main.go"},
		}, nil
	}
	// architectRef provided but verifier is nil (should still succeed — allows offline/test scenarios)
	result, err := CheckHotspot("/some/dir", "fix cmd/orch/main.go", "feature-impl", false, true, "orch-go-1184", checker, nil)
	if err != nil {
		t.Fatalf("--force-hotspot with ref but nil verifier should succeed, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected result when force-hotspot bypasses with ref but nil verifier")
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
