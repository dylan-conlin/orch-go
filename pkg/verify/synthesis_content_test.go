package verify

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEvidenceClaimsTestPass(t *testing.T) {
	tests := []struct {
		name     string
		evidence string
		want     bool
	}{
		{
			name:     "empty evidence",
			evidence: "",
			want:     false,
		},
		{
			name:     "claims tests passed",
			evidence: "All tests passed. Build succeeded.",
			want:     true,
		},
		{
			name:     "claims tests passing",
			evidence: "15 tests passing, no failures.",
			want:     true,
		},
		{
			name:     "claims test suite passed",
			evidence: "Test suite passed with full coverage.",
			want:     true,
		},
		{
			name:     "claims verification passed",
			evidence: "Verification passed, all checks green.",
			want:     true,
		},
		{
			name:     "claims build succeeded",
			evidence: "Build succeeded and deployed.",
			want:     true,
		},
		{
			name:     "no test claims",
			evidence: "Investigated the issue and found root cause.",
			want:     false,
		},
		{
			name:     "mentions tests but no pass claim",
			evidence: "Tests were updated to cover the new functionality.",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evidenceClaimsTestPass(tt.evidence)
			if got != tt.want {
				t.Errorf("evidenceClaimsTestPass(%q) = %v, want %v", tt.evidence, got, tt.want)
			}
		})
	}
}

func TestParseDurationClaim(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  time.Duration
	}{
		{
			name:  "standard format minutes",
			input: "45m",
			want:  45 * time.Minute,
		},
		{
			name:  "standard format hours",
			input: "2h",
			want:  2 * time.Hour,
		},
		{
			name:  "hours with word",
			input: "2 hours",
			want:  2 * time.Hour,
		},
		{
			name:  "minutes with word",
			input: "30 minutes",
			want:  30 * time.Minute,
		},
		{
			name:  "decimal hours",
			input: "1.5h",
			want:  90 * time.Minute,
		},
		{
			name:  "with tilde prefix",
			input: "~1h",
			want:  1 * time.Hour,
		},
		{
			name:  "with about prefix",
			input: "about 45m",
			want:  45 * time.Minute,
		},
		{
			name:  "unparseable",
			input: "some time",
			want:  0,
		},
		{
			name:  "empty",
			input: "",
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDurationClaim(tt.input)
			if got != tt.want {
				t.Errorf("parseDurationClaim(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsDurationReasonable(t *testing.T) {
	tests := []struct {
		name    string
		claimed string
		actual  time.Duration
		want    bool
	}{
		{
			name:    "exact match",
			claimed: "1h",
			actual:  1 * time.Hour,
			want:    true,
		},
		{
			name:    "within 50% variance - slightly under",
			claimed: "1h",
			actual:  45 * time.Minute, // 75% of claimed
			want:    true,
		},
		{
			name:    "within 50% variance - slightly over",
			claimed: "1h",
			actual:  80 * time.Minute, // 133% of claimed
			want:    true,
		},
		{
			name:    "way under claimed",
			claimed: "2h",
			actual:  30 * time.Minute, // 25% of claimed
			want:    false,
		},
		{
			name:    "way over claimed",
			claimed: "30m",
			actual:  2 * time.Hour, // 400% of claimed
			want:    false,
		},
		{
			name:    "unparseable claimed - gives benefit of doubt",
			claimed: "some time",
			actual:  1 * time.Hour,
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isDurationReasonable(tt.claimed, tt.actual)
			if got != tt.want {
				t.Errorf("isDurationReasonable(%q, %v) = %v, want %v", tt.claimed, tt.actual, got, tt.want)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{
			name:     "seconds",
			duration: 45 * time.Second,
			want:     "45s",
		},
		{
			name:     "minutes",
			duration: 30 * time.Minute,
			want:     "30m",
		},
		{
			name:     "hours",
			duration: 2 * time.Hour,
			want:     "2.0h",
		},
		{
			name:     "hours with fraction",
			duration: 90 * time.Minute,
			want:     "1.5h",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDuration(tt.duration)
			if got != tt.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, got, tt.want)
			}
		})
	}
}

func TestVerifySynthesisContent_NoWorkspace(t *testing.T) {
	result := VerifySynthesisContent("test-id", "", "/project")

	if !result.Passed {
		t.Error("Expected Passed=true for empty workspace")
	}
	if len(result.Warnings) == 0 {
		t.Error("Expected warning about no workspace path")
	}
}

func TestVerifySynthesisContent_NoSynthesis(t *testing.T) {
	// Create temp workspace without SYNTHESIS.md
	tmpDir, err := os.MkdirTemp("", "verify-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	result := VerifySynthesisContent("test-id", tmpDir, "/project")

	if !result.Passed {
		t.Error("Expected Passed=true when no SYNTHESIS.md")
	}
	if result.HasSynthesis {
		t.Error("Expected HasSynthesis=false")
	}
}

func TestVerifySynthesisContent_WithTestClaims(t *testing.T) {
	// Create temp workspace with SYNTHESIS.md that claims tests passed
	tmpDir, err := os.MkdirTemp("", "verify-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	synthesisContent := `# Session Synthesis

**Agent:** test-agent
**Issue:** test-123
**Duration:** 1h
**Outcome:** success

## TLDR

Fixed the bug and all tests pass.

## Evidence (What Was Observed)

All 15 tests passed. Build succeeded with no errors.

## Delta (What Changed)

- ` + "`pkg/verify/check.go`" + ` - Added new validation

## Next

**Recommendation:** close
`

	synthesisPath := filepath.Join(tmpDir, "SYNTHESIS.md")
	if err := os.WriteFile(synthesisPath, []byte(synthesisContent), 0644); err != nil {
		t.Fatal(err)
	}

	result := VerifySynthesisContent("test-id", tmpDir, "/project")

	if !result.HasSynthesis {
		t.Error("Expected HasSynthesis=true")
	}
	if !result.HasEvidenceSection {
		t.Error("Expected HasEvidenceSection=true")
	}
	if !result.EvidenceClaimsTestPass {
		t.Error("Expected EvidenceClaimsTestPass=true")
	}
	if result.ClaimedDuration != "1h" {
		t.Errorf("Expected ClaimedDuration='1h', got %q", result.ClaimedDuration)
	}
	// Note: We can't fully test beads comment validation without mocking
	// The warning about missing beads evidence would be added if beads check succeeded
}

func TestVerifySynthesisContentForCompletion_NilCases(t *testing.T) {
	// Empty workspace - should return nil
	result := VerifySynthesisContentForCompletion("test-id", "", "/project")
	if result != nil {
		t.Error("Expected nil for empty workspace")
	}

	// Workspace without SYNTHESIS.md - should return nil
	tmpDir, err := os.MkdirTemp("", "verify-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	result = VerifySynthesisContentForCompletion("test-id", tmpDir, "/project")
	if result != nil {
		t.Error("Expected nil when no SYNTHESIS.md")
	}
}
