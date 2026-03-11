package main

import (
	"strings"
	"testing"
)

func TestFormatHotspotAdvisory_NoMatches(t *testing.T) {
	result := formatHotspotAdvisory(nil)
	if result != "" {
		t.Errorf("expected empty string for nil hotspots, got %q", result)
	}

	result = formatHotspotAdvisory([]HotspotAdvisoryMatch{})
	if result != "" {
		t.Errorf("expected empty string for empty hotspots, got %q", result)
	}
}

func TestFormatHotspotAdvisory_SingleMatch(t *testing.T) {
	matches := []HotspotAdvisoryMatch{
		{
			FilePath: "cmd/orch/complete_cmd.go",
			Hotspot: Hotspot{
				Path:           "cmd/orch/complete_cmd.go",
				Type:           "bloat-size",
				Score:          2180,
				Recommendation: "CRITICAL: complete_cmd.go (2180 lines)",
			},
		},
	}

	result := formatHotspotAdvisory(matches)

	if !strings.Contains(result, "HOTSPOT ADVISORY") {
		t.Error("expected advisory header")
	}
	if !strings.Contains(result, "complete_cmd.go") {
		t.Error("expected file path in output")
	}
	if !strings.Contains(result, "bloat-size") {
		t.Error("expected hotspot type in output")
	}
}

func TestFormatHotspotAdvisory_MultipleMatches(t *testing.T) {
	matches := []HotspotAdvisoryMatch{
		{
			FilePath: "cmd/orch/daemon.go",
			Hotspot: Hotspot{
				Path:  "cmd/orch/daemon.go",
				Type:  "fix-density",
				Score: 7,
			},
		},
		{
			FilePath: "cmd/orch/daemon.go",
			Hotspot: Hotspot{
				Path:  "cmd/orch/daemon.go",
				Type:  "bloat-size",
				Score: 1200,
			},
		},
	}

	result := formatHotspotAdvisory(matches)

	if !strings.Contains(result, "fix-density") {
		t.Error("expected fix-density in output")
	}
	if !strings.Contains(result, "bloat-size") {
		t.Error("expected bloat-size in output")
	}
}

func TestMatchModifiedFilesToHotspots(t *testing.T) {
	hotspots := []Hotspot{
		{Path: "cmd/orch/complete_cmd.go", Type: "bloat-size", Score: 2180},
		{Path: "cmd/orch/daemon.go", Type: "fix-density", Score: 7},
		{Path: "spawn", Type: "investigation-cluster", Score: 5},
		{Path: "pkg/spawn/config.go", Type: "coupling-cluster", Score: 3, RelatedFiles: []string{"pkg/spawn/context.go"}},
	}

	tests := []struct {
		name          string
		modifiedFiles []string
		wantCount     int
		wantFiles     []string // files expected to match
	}{
		{
			name:          "exact match",
			modifiedFiles: []string{"cmd/orch/complete_cmd.go"},
			wantCount:     1,
			wantFiles:     []string{"cmd/orch/complete_cmd.go"},
		},
		{
			name:          "no match",
			modifiedFiles: []string{"README.md"},
			wantCount:     0,
		},
		{
			name:          "investigation cluster topic match",
			modifiedFiles: []string{"pkg/spawn/config.go"},
			wantCount:     2, // coupling-cluster (related file) + investigation-cluster (topic "spawn" in path)
		},
		{
			name:          "coupling cluster related file",
			modifiedFiles: []string{"pkg/spawn/context.go"},
			wantCount:     2, // coupling-cluster (related file match) + investigation-cluster (topic "spawn" in path)
		},
		{
			name:          "multiple files multiple matches",
			modifiedFiles: []string{"cmd/orch/complete_cmd.go", "cmd/orch/daemon.go"},
			wantCount:     2, // bloat-size for complete_cmd.go, fix-density for daemon.go
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := matchModifiedFilesToHotspots(tt.modifiedFiles, hotspots)
			if len(matches) != tt.wantCount {
				t.Errorf("got %d matches, want %d", len(matches), tt.wantCount)
				for _, m := range matches {
					t.Logf("  match: %s -> %s (%s)", m.FilePath, m.Hotspot.Path, m.Hotspot.Type)
				}
			}
		})
	}
}

func TestMatchModifiedFilesToHotspots_Dedup(t *testing.T) {
	// A file should only match each hotspot once, not multiple times
	hotspots := []Hotspot{
		{Path: "cmd/orch/daemon.go", Type: "bloat-size", Score: 1200},
	}

	modifiedFiles := []string{"cmd/orch/daemon.go"}
	matches := matchModifiedFilesToHotspots(modifiedFiles, hotspots)

	if len(matches) != 1 {
		t.Errorf("expected exactly 1 match (deduped), got %d", len(matches))
	}
}

func TestIsCodeFile(t *testing.T) {
	tests := []struct {
		path     string
		wantCode bool
	}{
		// Code files
		{"cmd/orch/main.go", true},
		{"pkg/verify/check.go", true},
		{"web/src/App.svelte", true},
		{"internal/handler.ts", true},
		{"main.py", true},
		{"Makefile", true},

		// Non-code files: .kb/ directory
		{".kb/models/foo/model.md", false},
		{".kb/investigations/2026-01-01-inv-something.md", false},
		{".kb/guides/spawn.md", false},
		{".kb/global/models/bar/probes/probe.md", false},

		// Non-code files: .orch/ directory
		{".orch/workspace/og-foo/SYNTHESIS.md", false},
		{".orch/templates/SYNTHESIS.md", false},
		{".orch/config.yaml", false},

		// Non-code files: .beads/ directory
		{".beads/issues.jsonl", false},
		{".beads/hooks/on_close", false},

		// Non-code files: .github/ directory
		{".github/workflows/ci.yml", false},

		// Non-code files: by extension
		{"README.md", false},
		{"CHANGELOG.md", false},
		{"go.sum", false},
		{"config.yaml", false},
		{"package.json", false},
		{"schema.json", false},
		{"notes.txt", false},
		{"Cargo.toml", false},
		{"go.mod.sum", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := isCodeFile(tt.path)
			if got != tt.wantCode {
				t.Errorf("isCodeFile(%q) = %v, want %v", tt.path, got, tt.wantCode)
			}
		})
	}
}

func TestFilterCodeFiles(t *testing.T) {
	input := []string{
		"cmd/orch/main.go",
		".kb/models/foo/model.md",
		"pkg/verify/check.go",
		".orch/workspace/foo/SYNTHESIS.md",
		"README.md",
		".beads/issues.jsonl",
		"web/src/App.svelte",
		"go.sum",
	}

	got := filterCodeFiles(input)

	want := []string{"cmd/orch/main.go", "pkg/verify/check.go", "web/src/App.svelte"}
	if len(got) != len(want) {
		t.Fatalf("filterCodeFiles: got %d files, want %d: %v", len(got), len(want), got)
	}
	for i, f := range got {
		if f != want[i] {
			t.Errorf("filterCodeFiles[%d] = %q, want %q", i, f, want[i])
		}
	}
}

func TestFilterCodeFiles_AllNonCode(t *testing.T) {
	input := []string{
		".kb/models/foo/probes/probe.md",
		".orch/workspace/bar/SYNTHESIS.md",
		"README.md",
		"CHANGELOG.md",
		"go.sum",
	}

	got := filterCodeFiles(input)
	if len(got) != 0 {
		t.Errorf("expected empty slice for all non-code files, got %v", got)
	}
}

func TestFilterCodeFiles_Empty(t *testing.T) {
	got := filterCodeFiles(nil)
	if len(got) != 0 {
		t.Errorf("expected empty slice for nil input, got %v", got)
	}
}
