// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"errors"
	"testing"
)

var errMockHotspotError = errors.New("mock hotspot error")

// =============================================================================
// Tests for HotspotWarning
// =============================================================================

func TestHotspotWarning_Fields(t *testing.T) {
	warning := HotspotWarning{
		Path:           "cmd/orch/status.go",
		Type:           "fix-density",
		Score:          7,
		Recommendation: "CRITICAL: Consider architect session",
	}

	if warning.Path != "cmd/orch/status.go" {
		t.Errorf("Path = %q, want 'cmd/orch/status.go'", warning.Path)
	}
	if warning.Type != "fix-density" {
		t.Errorf("Type = %q, want 'fix-density'", warning.Type)
	}
	if warning.Score != 7 {
		t.Errorf("Score = %d, want 7", warning.Score)
	}
}

func TestHotspotWarning_IsCritical(t *testing.T) {
	tests := []struct {
		name  string
		score int
		want  bool
	}{
		{"score 10 is critical", 10, true},
		{"score 11 is critical", 11, true},
		{"score 9 is not critical", 9, false},
		{"score 5 is not critical", 5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := HotspotWarning{Score: tt.score}
			if got := w.IsCritical(); got != tt.want {
				t.Errorf("IsCritical() = %v, want %v", got, tt.want)
			}
		})
	}
}

// =============================================================================
// Tests for PreviewResult Hotspot Integration
// =============================================================================

func TestPreviewResult_HotspotWarnings(t *testing.T) {
	result := PreviewResult{
		Issue: &Issue{ID: "proj-1", Title: "Test"},
		Skill: "feature-impl",
		HotspotWarnings: []HotspotWarning{
			{Path: "file1.go", Score: 7},
			{Path: "file2.go", Score: 10},
		},
	}

	if len(result.HotspotWarnings) != 2 {
		t.Errorf("HotspotWarnings length = %d, want 2", len(result.HotspotWarnings))
	}
}

func TestPreviewResult_HasHotspotWarnings(t *testing.T) {
	tests := []struct {
		name     string
		warnings []HotspotWarning
		want     bool
	}{
		{"no warnings", nil, false},
		{"empty warnings", []HotspotWarning{}, false},
		{"has warnings", []HotspotWarning{{Path: "file.go"}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PreviewResult{HotspotWarnings: tt.warnings}
			if got := result.HasHotspotWarnings(); got != tt.want {
				t.Errorf("HasHotspotWarnings() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPreviewResult_HasCriticalHotspots(t *testing.T) {
	tests := []struct {
		name     string
		warnings []HotspotWarning
		want     bool
	}{
		{"no warnings", nil, false},
		{"only moderate warnings", []HotspotWarning{{Score: 5}, {Score: 7}}, false},
		{"one critical warning", []HotspotWarning{{Score: 5}, {Score: 10}}, true},
		{"all critical warnings", []HotspotWarning{{Score: 10}, {Score: 12}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PreviewResult{HotspotWarnings: tt.warnings}
			if got := result.HasCriticalHotspots(); got != tt.want {
				t.Errorf("HasCriticalHotspots() = %v, want %v", got, tt.want)
			}
		})
	}
}

// =============================================================================
// Tests for CheckHotspotsForIssue
// =============================================================================

func TestCheckHotspotsForIssue_NoHotspots(t *testing.T) {
	// Create a mock hotspot checker that returns no hotspots
	checker := &MockHotspotChecker{
		Hotspots: []HotspotWarning{},
	}

	warnings := CheckHotspotsForIssue(&Issue{
		ID:    "proj-1",
		Title: "Add feature X",
	}, checker)

	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %d", len(warnings))
	}
}

func TestCheckHotspotsForIssue_WithHotspots(t *testing.T) {
	// Create a mock hotspot checker that returns hotspots
	checker := &MockHotspotChecker{
		Hotspots: []HotspotWarning{
			{Path: "cmd/orch/status.go", Type: "fix-density", Score: 8},
		},
	}

	warnings := CheckHotspotsForIssue(&Issue{
		ID:    "proj-1",
		Title: "Fix status display",
	}, checker)

	if len(warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(warnings))
	}
	if warnings[0].Path != "cmd/orch/status.go" {
		t.Errorf("warning path = %q, want 'cmd/orch/status.go'", warnings[0].Path)
	}
}

func TestCheckHotspotsForIssue_NilChecker(t *testing.T) {
	// When checker is nil, should return no warnings (graceful degradation)
	warnings := CheckHotspotsForIssue(&Issue{ID: "proj-1"}, nil)

	if len(warnings) != 0 {
		t.Errorf("expected no warnings with nil checker, got %d", len(warnings))
	}
}

func TestCheckHotspotsForIssue_NilIssue(t *testing.T) {
	checker := &MockHotspotChecker{
		Hotspots: []HotspotWarning{{Path: "file.go"}},
	}

	warnings := CheckHotspotsForIssue(nil, checker)

	if len(warnings) != 0 {
		t.Errorf("expected no warnings with nil issue, got %d", len(warnings))
	}
}

// =============================================================================
// Tests for FormatHotspotWarnings
// =============================================================================

func TestFormatHotspotWarnings_Empty(t *testing.T) {
	result := FormatHotspotWarnings(nil)
	if result != "" {
		t.Errorf("FormatHotspotWarnings(nil) = %q, want empty", result)
	}

	result = FormatHotspotWarnings([]HotspotWarning{})
	if result != "" {
		t.Errorf("FormatHotspotWarnings([]) = %q, want empty", result)
	}
}

func TestFormatHotspotWarnings_SingleWarning(t *testing.T) {
	warnings := []HotspotWarning{
		{Path: "cmd/status.go", Type: "fix-density", Score: 7, Recommendation: "Review before spawning"},
	}

	result := FormatHotspotWarnings(warnings)

	// Should contain the warning header
	if !contains(result, "HOTSPOT WARNING") {
		t.Error("FormatHotspotWarnings() missing HOTSPOT WARNING header")
	}
	// Should contain the file path
	if !contains(result, "cmd/status.go") {
		t.Error("FormatHotspotWarnings() missing file path")
	}
	// Should contain the recommendation
	if !contains(result, "Review before spawning") {
		t.Error("FormatHotspotWarnings() missing recommendation")
	}
}

func TestFormatHotspotWarnings_CriticalWarning(t *testing.T) {
	warnings := []HotspotWarning{
		{Path: "pkg/daemon/daemon.go", Type: "fix-density", Score: 10, Recommendation: "CRITICAL: Spawn architect"},
	}

	result := FormatHotspotWarnings(warnings)

	// Should indicate critical status
	if !contains(result, "CRITICAL") {
		t.Error("FormatHotspotWarnings() should highlight CRITICAL warnings")
	}
}

func TestFormatHotspotWarnings_MultipleWarnings(t *testing.T) {
	warnings := []HotspotWarning{
		{Path: "file1.go", Score: 5},
		{Path: "file2.go", Score: 8},
		{Path: "file3.go", Score: 10},
	}

	result := FormatHotspotWarnings(warnings)

	// Should contain all file paths
	if !contains(result, "file1.go") {
		t.Error("FormatHotspotWarnings() missing file1.go")
	}
	if !contains(result, "file2.go") {
		t.Error("FormatHotspotWarnings() missing file2.go")
	}
	if !contains(result, "file3.go") {
		t.Error("FormatHotspotWarnings() missing file3.go")
	}
}

func TestFormatHotspotWarnings_SeverityIcons(t *testing.T) {
	// The icons differ by severity: 🔸 (low), 🟡 (medium, score>=7), 🔴 (critical, score>=10)
	tests := []struct {
		name     string
		score    int
		wantIcon string
	}{
		{"low score gets orange diamond", 3, "🔸"},
		{"score 6 gets orange diamond", 6, "🔸"},
		{"score 7 gets yellow circle", 7, "🟡"},
		{"score 9 gets yellow circle", 9, "🟡"},
		{"score 10 gets red circle", 10, "🔴"},
		{"score 15 gets red circle", 15, "🔴"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := []HotspotWarning{
				{Path: "test.go", Type: "fix-density", Score: tt.score},
			}
			result := FormatHotspotWarnings(warnings)
			if !contains(result, tt.wantIcon) {
				t.Errorf("Score %d: expected icon %q in output, got:\n%s", tt.score, tt.wantIcon, result)
			}
		})
	}
}

func TestFormatHotspotWarnings_RecommendationContent(t *testing.T) {
	// Test that recommendation text is included when provided
	warnings := []HotspotWarning{
		{Path: "test.go", Type: "fix-density", Score: 5, Recommendation: "Consider refactoring"},
	}
	result := FormatHotspotWarnings(warnings)
	if !contains(result, "Consider refactoring") {
		t.Error("FormatHotspotWarnings() should include recommendation text")
	}
}

func TestFormatHotspotWarnings_NoRecommendation(t *testing.T) {
	// When recommendation is empty, should still format correctly
	warnings := []HotspotWarning{
		{Path: "test.go", Type: "fix-density", Score: 5, Recommendation: ""},
	}
	result := FormatHotspotWarnings(warnings)
	if !contains(result, "test.go") {
		t.Error("FormatHotspotWarnings() should still include file path without recommendation")
	}
	// Should NOT contain the recommendation arrow
	if contains(result, "└─") {
		t.Error("FormatHotspotWarnings() should not include recommendation prefix when empty")
	}
}

// =============================================================================
// Tests for GenerateHotspotRecommendation
// =============================================================================

func TestGenerateHotspotRecommendation(t *testing.T) {
	tests := []struct {
		name        string
		hasCritical bool
		wantContain string
	}{
		{"critical hotspots", true, "architect"},
		{"no critical hotspots", false, "review"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateHotspotRecommendation(tt.hasCritical)
			if !contains(result, tt.wantContain) {
				t.Errorf("GenerateHotspotRecommendation(%v) = %q, should contain %q",
					tt.hasCritical, result, tt.wantContain)
			}
		})
	}
}

// =============================================================================
// Mock Implementation for Testing
// =============================================================================

// MockHotspotChecker is a mock implementation of HotspotChecker for testing.
type MockHotspotChecker struct {
	Hotspots []HotspotWarning
	Error    error
}

func (m *MockHotspotChecker) CheckHotspots(projectDir string) ([]HotspotWarning, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	return m.Hotspots, nil
}

// =============================================================================
// Tests for GitHotspotChecker
// =============================================================================

func TestNewGitHotspotChecker(t *testing.T) {
	checker := NewGitHotspotChecker()

	if checker.FixThreshold != 5 {
		t.Errorf("FixThreshold = %d, want 5", checker.FixThreshold)
	}
	if checker.InvThreshold != 3 {
		t.Errorf("InvThreshold = %d, want 3", checker.InvThreshold)
	}
	if checker.DaysBack != 28 {
		t.Errorf("DaysBack = %d, want 28", checker.DaysBack)
	}
}

func TestGitHotspotChecker_CheckHotspots_NoOrchCommand(t *testing.T) {
	// When orch command is not available, should return nil gracefully
	checker := NewGitHotspotChecker()

	// Use a non-existent directory to ensure command fails
	warnings, err := checker.CheckHotspots("/nonexistent/path/12345")

	// Should not error - graceful degradation
	if err != nil {
		t.Errorf("CheckHotspots() error = %v, want nil", err)
	}

	// Should return nil or empty slice
	if warnings != nil && len(warnings) > 0 {
		t.Errorf("CheckHotspots() expected empty warnings on failure, got %d", len(warnings))
	}
}

// =============================================================================
// Tests for Daemon.Preview with Hotspots
// =============================================================================

func TestDaemon_Preview_WithHotspotChecker(t *testing.T) {
	checker := &MockHotspotChecker{
		Hotspots: []HotspotWarning{
			{Path: "cmd/orch/status.go", Type: "fix-density", Score: 8},
		},
	}

	d := &Daemon{
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Fix status", Priority: 0, IssueType: "bug", Status: "open"},
			}, nil
		}},
		HotspotChecker: checker,
	}

	result, err := d.Preview()
	if err != nil {
		t.Fatalf("Preview() unexpected error: %v", err)
	}

	if result.Issue == nil {
		t.Fatal("Preview() expected issue")
	}

	if len(result.HotspotWarnings) != 1 {
		t.Errorf("Preview() expected 1 hotspot warning, got %d", len(result.HotspotWarnings))
	}
}

func TestDaemon_Preview_NoHotspotChecker(t *testing.T) {
	// When no hotspot checker is configured, Preview should still work
	d := &Daemon{
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Fix bug", Priority: 0, IssueType: "bug", Status: "open"},
			}, nil
		}},
		// HotspotChecker is nil
	}

	result, err := d.Preview()
	if err != nil {
		t.Fatalf("Preview() unexpected error: %v", err)
	}

	if result.Issue == nil {
		t.Fatal("Preview() expected issue")
	}

	// Should have no warnings when checker is not configured
	if len(result.HotspotWarnings) != 0 {
		t.Errorf("Preview() expected 0 hotspot warnings, got %d", len(result.HotspotWarnings))
	}
}

func TestDaemon_Preview_HotspotCheckerError(t *testing.T) {
	// When hotspot checker returns an error, Preview should still work (graceful degradation)
	checker := &MockHotspotChecker{
		Error: errMockHotspotError,
	}

	d := &Daemon{
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Fix bug", Priority: 0, IssueType: "bug", Status: "open"},
			}, nil
		}},
		HotspotChecker: checker,
	}

	result, err := d.Preview()
	if err != nil {
		t.Fatalf("Preview() unexpected error: %v", err)
	}

	if result.Issue == nil {
		t.Fatal("Preview() expected issue (should not fail on hotspot error)")
	}

	// Should have no warnings on error
	if len(result.HotspotWarnings) != 0 {
		t.Errorf("Preview() expected 0 hotspot warnings on error, got %d", len(result.HotspotWarnings))
	}
}
