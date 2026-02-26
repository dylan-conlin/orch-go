package gates

import (
	"fmt"
	"strings"
	"testing"
)

func TestCheckHotspot_NilChecker(t *testing.T) {
	result, err := CheckHotspot("/some/dir", "some task", "feature-impl", false, false, "", nil, nil, nil)
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
	result, err := CheckHotspot("", "some task", "feature-impl", false, false, "", checker, nil, nil)
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
	result, err := CheckHotspot("/some/dir", "some task", "feature-impl", false, false, "", checker, nil, nil)
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
	result, err := CheckHotspot("/some/dir", "some task", "feature-impl", false, false, "", checker, nil, nil)
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
	result, err := CheckHotspot("/some/dir", "some task", "feature-impl", true, false, "", checker, nil, nil)
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
	_, err := CheckHotspot("/some/dir", "fix cmd/orch/main.go", "feature-impl", false, false, "", checker, nil, nil)
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
	_, err := CheckHotspot("/some/dir", "debug pkg/big/file.go", "systematic-debugging", false, false, "", checker, nil, nil)
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
	result, err := CheckHotspot("/some/dir", "review cmd/orch/main.go", "architect", false, false, "", checker, nil, nil)
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
	_, err := CheckHotspot("/some/dir", "investigate cmd/orch/main.go", "investigation", false, false, "", checker, nil, nil)
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
	_, err := CheckHotspot("/some/dir", "capture knowledge about main.go", "capture-knowledge", false, false, "", checker, nil, nil)
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
	_, err := CheckHotspot("/some/dir", "audit codebase", "codebase-audit", false, false, "", checker, nil, nil)
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
	_, err := CheckHotspot("/some/dir", "fix cmd/orch/main.go", "feature-impl", false, true, "", checker, nil, nil)
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
	result, err := CheckHotspot("/some/dir", "fix cmd/orch/main.go", "feature-impl", false, true, "orch-go-1184", checker, verifier, nil)
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
	_, err := CheckHotspot("/some/dir", "fix cmd/orch/main.go", "feature-impl", false, true, "orch-go-9999", checker, verifier, nil)
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
	_, err := CheckHotspot("/some/dir", "fix cmd/orch/main.go", "feature-impl", false, true, "orch-go-1182", checker, verifier, nil)
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
	_, err := CheckHotspot("/some/dir", "fix cmd/orch/main.go", "feature-impl", false, true, "orch-go-1184", checker, verifier, nil)
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
	result, err := CheckHotspot("/some/dir", "fix cmd/orch/main.go", "feature-impl", false, true, "orch-go-1184", checker, nil, nil)
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

// --- Additional coverage tests ---

func TestCheckHotspot_CheckerReturnsError(t *testing.T) {
	// When checker returns an error, CheckHotspot should gracefully return nil
	checker := func(dir, task string) (*HotspotResult, error) {
		return nil, fmt.Errorf("hotspot analysis failed")
	}
	result, err := CheckHotspot("/some/dir", "some task", "feature-impl", false, false, "", checker, nil, nil)
	if err != nil {
		t.Errorf("expected nil error on checker failure, got: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result on checker failure, got: %v", result)
	}
}

func TestCheckHotspot_DaemonDrivenBypassesCriticalBlock(t *testing.T) {
	// Daemon-driven spawns return result even for CRITICAL hotspots without blocking.
	// The daemon handles hotspot routing at its own layer (architect escalation).
	checker := func(dir, task string) (*HotspotResult, error) {
		return &HotspotResult{
			HasHotspots:        true,
			HasCriticalHotspot: true,
			Warning:            "CRITICAL hotspot warning",
			CriticalFiles:      []string{"cmd/orch/main.go"},
		}, nil
	}
	result, err := CheckHotspot("/some/dir", "fix cmd/orch/main.go", "feature-impl", true, false, "", checker, nil, nil)
	if err != nil {
		t.Fatalf("daemon-driven should bypass CRITICAL block, got error: %v", err)
	}
	if result == nil {
		t.Fatal("expected result for daemon-driven CRITICAL hotspot")
	}
	if !result.HasCriticalHotspot {
		t.Error("expected HasCriticalHotspot to be true")
	}
}

func TestCheckHotspot_MultipleCriticalFilesInError(t *testing.T) {
	// When multiple files are CRITICAL, error message should list them all.
	checker := func(dir, task string) (*HotspotResult, error) {
		return &HotspotResult{
			HasHotspots:        true,
			HasCriticalHotspot: true,
			Warning:            "CRITICAL hotspot warning",
			CriticalFiles:      []string{"cmd/orch/main.go", "pkg/daemon/daemon.go"},
		}, nil
	}
	_, err := CheckHotspot("/some/dir", "refactor multiple files", "feature-impl", false, false, "", checker, nil, nil)
	if err == nil {
		t.Fatal("expected error for CRITICAL hotspot with multiple files")
	}
	if !strContains(err.Error(), "cmd/orch/main.go") {
		t.Errorf("error should mention cmd/orch/main.go, got: %v", err)
	}
	if !strContains(err.Error(), "pkg/daemon/daemon.go") {
		t.Errorf("error should mention pkg/daemon/daemon.go, got: %v", err)
	}
}

func TestCheckHotspot_ForceHotspotNoEffectWhenNotCritical(t *testing.T) {
	// --force-hotspot with a non-critical hotspot should proceed normally
	// (force flag only matters when there's a CRITICAL block to bypass).
	checker := func(dir, task string) (*HotspotResult, error) {
		return &HotspotResult{
			HasHotspots:        true,
			HasCriticalHotspot: false,
			Warning:            "non-critical warning",
		}, nil
	}
	result, err := CheckHotspot("/some/dir", "some task", "feature-impl", false, true, "orch-go-1184", checker, nil, nil)
	if err != nil {
		t.Fatalf("non-critical hotspot with force flag should not error, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected result for non-critical hotspot")
	}
}

func TestCheckHotspot_ForceHotspotNonBlockingSkill(t *testing.T) {
	// --force-hotspot with exempt skill should proceed normally even with CRITICAL hotspot.
	checker := func(dir, task string) (*HotspotResult, error) {
		return &HotspotResult{
			HasHotspots:        true,
			HasCriticalHotspot: true,
			Warning:            "CRITICAL hotspot warning",
			CriticalFiles:      []string{"cmd/orch/main.go"},
		}, nil
	}
	result, err := CheckHotspot("/some/dir", "review architecture", "architect", false, true, "", checker, nil, nil)
	if err != nil {
		t.Fatalf("exempt skill should not be blocked even with CRITICAL hotspot, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected result for exempt skill with CRITICAL hotspot")
	}
}

func TestCheckHotspot_ResultFieldsPreserved(t *testing.T) {
	// Verify that the result returned preserves all fields from the checker.
	checker := func(dir, task string) (*HotspotResult, error) {
		return &HotspotResult{
			HasHotspots:        true,
			HasCriticalHotspot: false,
			Warning:            "test warning",
			CriticalFiles:      []string{},
			MatchedFiles:       []string{"pkg/spawn/context.go", "pkg/spawn/config.go"},
		}, nil
	}
	result, err := CheckHotspot("/some/dir", "update spawn", "feature-impl", false, false, "", checker, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.MatchedFiles) != 2 {
		t.Errorf("expected 2 matched files, got %d", len(result.MatchedFiles))
	}
	if result.MatchedFiles[0] != "pkg/spawn/context.go" {
		t.Errorf("expected first matched file to be 'pkg/spawn/context.go', got %q", result.MatchedFiles[0])
	}
}

// --- Auto-detection tests ---

func TestCheckHotspot_AutoDetectPriorArchitect(t *testing.T) {
	// When no --force-hotspot but architectFinder finds a prior closed architect review,
	// the gate should auto-bypass.
	checker := func(dir, task string) (*HotspotResult, error) {
		return &HotspotResult{
			HasHotspots:        true,
			HasCriticalHotspot: true,
			Warning:            "CRITICAL hotspot warning",
			CriticalFiles:      []string{"cmd/orch/main.go"},
		}, nil
	}
	finder := func(criticalFiles []string) (string, error) {
		if len(criticalFiles) == 1 && criticalFiles[0] == "cmd/orch/main.go" {
			return "orch-go-1184", nil
		}
		return "", nil
	}
	verifier := func(issueID string) error {
		if issueID == "orch-go-1184" {
			return nil // Valid closed architect issue
		}
		return fmt.Errorf("unexpected issue: %s", issueID)
	}
	result, err := CheckHotspot("/some/dir", "fix cmd/orch/main.go", "feature-impl", false, false, "", checker, verifier, finder)
	if err != nil {
		t.Fatalf("auto-detected architect review should bypass block, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected result when auto-detected architect bypasses block")
	}
}

func TestCheckHotspot_AutoDetectNoMatchStillBlocks(t *testing.T) {
	// When architectFinder returns empty string (no prior architect), gate should still block.
	checker := func(dir, task string) (*HotspotResult, error) {
		return &HotspotResult{
			HasHotspots:        true,
			HasCriticalHotspot: true,
			Warning:            "CRITICAL hotspot warning",
			CriticalFiles:      []string{"cmd/orch/main.go"},
		}, nil
	}
	finder := func(criticalFiles []string) (string, error) {
		return "", nil // No prior architect found
	}
	_, err := CheckHotspot("/some/dir", "fix cmd/orch/main.go", "feature-impl", false, false, "", checker, nil, finder)
	if err == nil {
		t.Fatal("expected error when no prior architect found")
	}
	if !strContains(err.Error(), "CRITICAL hotspot") {
		t.Errorf("error should mention CRITICAL hotspot, got: %v", err)
	}
}

func TestCheckHotspot_AutoDetectFinderErrorStillBlocks(t *testing.T) {
	// When architectFinder returns an error, gate should still block (graceful degradation).
	checker := func(dir, task string) (*HotspotResult, error) {
		return &HotspotResult{
			HasHotspots:        true,
			HasCriticalHotspot: true,
			Warning:            "CRITICAL hotspot warning",
			CriticalFiles:      []string{"cmd/orch/main.go"},
		}, nil
	}
	finder := func(criticalFiles []string) (string, error) {
		return "", fmt.Errorf("beads query failed")
	}
	_, err := CheckHotspot("/some/dir", "fix cmd/orch/main.go", "feature-impl", false, false, "", checker, nil, finder)
	if err == nil {
		t.Fatal("expected error when finder fails")
	}
	if !strContains(err.Error(), "CRITICAL hotspot") {
		t.Errorf("error should mention CRITICAL hotspot, got: %v", err)
	}
}

func TestCheckHotspot_AutoDetectVerificationFailsStillBlocks(t *testing.T) {
	// When architectFinder finds a candidate but verifier rejects it, gate should block.
	checker := func(dir, task string) (*HotspotResult, error) {
		return &HotspotResult{
			HasHotspots:        true,
			HasCriticalHotspot: true,
			Warning:            "CRITICAL hotspot warning",
			CriticalFiles:      []string{"cmd/orch/main.go"},
		}, nil
	}
	finder := func(criticalFiles []string) (string, error) {
		return "orch-go-1184", nil
	}
	verifier := func(issueID string) error {
		return fmt.Errorf("architect review not complete (status=in_progress)")
	}
	_, err := CheckHotspot("/some/dir", "fix cmd/orch/main.go", "feature-impl", false, false, "", checker, verifier, finder)
	if err == nil {
		t.Fatal("expected error when auto-detected architect fails verification")
	}
	if !strContains(err.Error(), "CRITICAL hotspot") {
		t.Errorf("error should mention CRITICAL hotspot, got: %v", err)
	}
}

func TestCheckHotspot_AutoDetectNilVerifierTrustsFinder(t *testing.T) {
	// When architectFinder finds a candidate and verifier is nil, trust the finder.
	checker := func(dir, task string) (*HotspotResult, error) {
		return &HotspotResult{
			HasHotspots:        true,
			HasCriticalHotspot: true,
			Warning:            "CRITICAL hotspot warning",
			CriticalFiles:      []string{"cmd/orch/main.go"},
		}, nil
	}
	finder := func(criticalFiles []string) (string, error) {
		return "orch-go-1184", nil
	}
	result, err := CheckHotspot("/some/dir", "fix cmd/orch/main.go", "feature-impl", false, false, "", checker, nil, finder)
	if err != nil {
		t.Fatalf("auto-detected architect with nil verifier should succeed, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected result when auto-detected architect bypasses with nil verifier")
	}
}

func TestCheckHotspot_ForceHotspotTakesPrecedenceOverAutoDetect(t *testing.T) {
	// When --force-hotspot is used, auto-detection should not be attempted.
	finderCalled := false
	checker := func(dir, task string) (*HotspotResult, error) {
		return &HotspotResult{
			HasHotspots:        true,
			HasCriticalHotspot: true,
			Warning:            "CRITICAL hotspot warning",
			CriticalFiles:      []string{"cmd/orch/main.go"},
		}, nil
	}
	finder := func(criticalFiles []string) (string, error) {
		finderCalled = true
		return "orch-go-1184", nil
	}
	verifier := func(issueID string) error {
		return nil
	}
	result, err := CheckHotspot("/some/dir", "fix cmd/orch/main.go", "feature-impl", false, true, "orch-go-1184", checker, verifier, finder)
	if err != nil {
		t.Fatalf("explicit force-hotspot should succeed, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected result with explicit force-hotspot")
	}
	if finderCalled {
		t.Error("finder should not be called when --force-hotspot is explicit")
	}
}

func strContains(s, substr string) bool {
	return strings.Contains(s, substr)
}
