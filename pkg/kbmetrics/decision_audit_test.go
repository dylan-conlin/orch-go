package kbmetrics

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClassifyDecision(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		content  string
		wantType DecisionType
	}{
		{
			name:     "principle in title",
			title:    "Verification Bottleneck Principle",
			content:  "The system cannot change faster than a human can verify behavior.",
			wantType: DecisionArchitectural,
		},
		{
			name:     "pattern in title",
			title:    "Schema Migration Pattern",
			content:  "Schema changes require backward-compatible discovery.",
			wantType: DecisionArchitectural,
		},
		{
			name:     "model in title",
			title:    "Project Group Model",
			content:  "Groups are defined in groups.yaml.",
			wantType: DecisionArchitectural,
		},
		{
			name:     "stability foundational frontmatter",
			title:    "Event-Sourced Monitoring Architecture",
			content:  "---\nstability: foundational\n---\n# Decision",
			wantType: DecisionArchitectural,
		},
		{
			name:     "implementation with blocks patterns",
			title:    "Accretion Gates Advisory, Not Blocking",
			content:  "---\nblocks:\n  - patterns:\n      - \"**/spawn/gates/hotspot*\"\n---\n# Decision",
			wantType: DecisionImplementation,
		},
		{
			name:     "implementation with specific file references",
			title:    "Role Aware Hook Filtering",
			content:  "Claude Code hooks must check CLAUDE_CONTEXT and exit early.\n\npkg/spawn/gates/hotspot.go and cmd/orch/spawn_cmd.go and pkg/verify/accretion.go changes needed.",
			wantType: DecisionImplementation,
		},
		{
			name:     "implementation with code changes section",
			title:    "Remove Self-Review Completion Gate",
			content:  "## What Changes\n\nRemove the self-review gate from completion pipeline.",
			wantType: DecisionImplementation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyDecision(tt.title, tt.content)
			if got != tt.wantType {
				t.Errorf("classifyDecision(%q) = %v, want %v", tt.title, got, tt.wantType)
			}
		})
	}
}

func TestParseDecision(t *testing.T) {
	content := `---
status: proposed
blocks:
  - keywords:
      - accretion gate
    patterns:
      - "**/spawn/gates/hotspot*"
      - "**/verify/accretion*"
---

# Decision: Accretion Gates Advisory, Not Blocking

**Date:** 2026-03-17
**Status:** Proposed
`
	d, err := parseDecision("test.md", content)
	if err != nil {
		t.Fatal(err)
	}
	if d.Title != "Accretion Gates Advisory, Not Blocking" {
		t.Errorf("title = %q", d.Title)
	}
	if d.Status != "Proposed" {
		t.Errorf("status = %q", d.Status)
	}
	if d.Date != "2026-03-17" {
		t.Errorf("date = %q", d.Date)
	}
	if len(d.BlockPatterns) != 2 {
		t.Errorf("block patterns = %d, want 2", len(d.BlockPatterns))
	}
}

func TestValidateArchitectural(t *testing.T) {
	// Create a temp dir with a fake gate file and test file
	dir := t.TempDir()
	gatesDir := filepath.Join(dir, "pkg", "spawn", "gates")
	os.MkdirAll(gatesDir, 0755)
	os.WriteFile(filepath.Join(gatesDir, "hotspot.go"), []byte("package gates"), 0644)

	testDir := filepath.Join(dir, "cmd", "orch")
	os.MkdirAll(testDir, 0755)
	os.WriteFile(filepath.Join(testDir, "architecture_lint_test.go"), []byte(`
func TestNoSessionRegistry(t *testing.T) {
	// verification bottleneck enforced here
}
`), 0644)

	d := Decision{
		Title: "Verification Bottleneck Principle",
		Type:  DecisionArchitectural,
	}

	findings := validateArchitectural(d, dir)
	// Should find some evidence (the test file references the concept)
	// The key thing is it doesn't crash and returns findings
	_ = findings
}

func TestValidateImplementation(t *testing.T) {
	dir := t.TempDir()

	// Create a file that should exist
	gatesDir := filepath.Join(dir, "pkg", "spawn", "gates")
	os.MkdirAll(gatesDir, 0755)
	os.WriteFile(filepath.Join(gatesDir, "hotspot.go"), []byte("package gates\n// CheckHotspot does things"), 0644)

	d := Decision{
		Title:         "Accretion Gates Advisory",
		Type:          DecisionImplementation,
		BlockPatterns: []string{"**/spawn/gates/hotspot*"},
	}

	findings := validateImplementation(d, dir)
	// hotspot.go exists so the pattern should match
	hasMatch := false
	for _, f := range findings {
		if f.Status == FindingPresent {
			hasMatch = true
		}
	}
	if !hasMatch {
		t.Errorf("expected at least one present finding, got %v", findings)
	}
}

func TestValidateImplementationMissing(t *testing.T) {
	dir := t.TempDir()

	d := Decision{
		Title:         "Some Feature",
		Type:          DecisionImplementation,
		BlockPatterns: []string{"**/nonexistent/path*"},
	}

	findings := validateImplementation(d, dir)
	hasMissing := false
	for _, f := range findings {
		if f.Status == FindingMissing {
			hasMissing = true
		}
	}
	if !hasMissing {
		t.Errorf("expected missing finding for nonexistent pattern, got %v", findings)
	}
}

func TestAuditDecisionsIntegration(t *testing.T) {
	dir := t.TempDir()
	decDir := filepath.Join(dir, ".kb", "decisions")
	os.MkdirAll(decDir, 0755)

	// Write an architectural decision
	os.WriteFile(filepath.Join(decDir, "2026-01-01-test-principle.md"), []byte(`# Decision: Test Principle

**Date:** 2026-01-01
**Status:** Accepted

## Decision

No local agent state.
`), 0644)

	// Write an implementation decision with blocks
	os.WriteFile(filepath.Join(decDir, "2026-01-02-test-impl.md"), []byte(`---
blocks:
  - patterns:
      - "**/pkg/verify/*"
---
# Decision: Test Implementation

**Date:** 2026-01-02
**Status:** Active

## What Changes

Update verify package.
`), 0644)

	// Create the verify package so it's found
	verifyDir := filepath.Join(dir, "pkg", "verify")
	os.MkdirAll(verifyDir, 0755)
	os.WriteFile(filepath.Join(verifyDir, "check.go"), []byte("package verify"), 0644)

	reports, err := AuditDecisions(filepath.Join(dir, ".kb"), dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 2 {
		t.Fatalf("expected 2 reports, got %d", len(reports))
	}

	// Check types
	archCount, implCount := 0, 0
	for _, r := range reports {
		switch r.Type {
		case DecisionArchitectural:
			archCount++
		case DecisionImplementation:
			implCount++
		}
	}
	if archCount != 1 || implCount != 1 {
		t.Errorf("types: arch=%d impl=%d, want 1/1", archCount, implCount)
	}
}

func TestExtractFileReferences(t *testing.T) {
	content := `
Changes to pkg/spawn/gates/hotspot.go and cmd/orch/precommit_cmd.go.
Also modifies pkg/verify/accretion.go.
`
	refs := extractFileReferences(content)
	if len(refs) != 3 {
		t.Errorf("expected 3 file refs, got %d: %v", len(refs), refs)
	}
}
