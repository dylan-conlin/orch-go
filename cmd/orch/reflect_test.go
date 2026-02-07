package main

import (
	"encoding/json"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

func TestFormatPaths(t *testing.T) {
	tests := []struct {
		name     string
		paths    []string
		expected string
	}{
		{
			name:     "single path",
			paths:    []string{"/foo/bar/baz.md"},
			expected: "baz.md",
		},
		{
			name:     "multiple paths",
			paths:    []string{"/a/b/one.md", "/c/d/two.md"},
			expected: "one.md, two.md",
		},
		{
			name:     "empty",
			paths:    []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatPaths(tt.paths)
			if result != tt.expected {
				t.Errorf("formatPaths(%v) = %q, want %q", tt.paths, result, tt.expected)
			}
		})
	}
}

func TestPrintJSON_ReflectSuggestions(t *testing.T) {
	s := &daemon.ReflectSuggestions{
		OrphanInvestigations: []daemon.OrphanInvestigationSuggestion{
			{
				Path:                  "/test/inv.md",
				Topic:                 "daemon",
				SimilarInvestigations: []string{"/test/inv2.md"},
				Suggestion:            "Potential lineage gap",
			},
		},
	}

	// Verify it marshals without error
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Verify the JSON contains expected fields
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	orphans, ok := parsed["orphan_investigations"]
	if !ok {
		t.Fatal("expected orphan_investigations in JSON output")
	}

	arr, ok := orphans.([]any)
	if !ok || len(arr) != 1 {
		t.Fatalf("expected 1 orphan investigation, got %v", orphans)
	}
}

func TestPrintJSON_OrphanInvestigations(t *testing.T) {
	o := &verify.OrphanInvestigations{
		TotalScanned: 10,
		Orphans: []verify.OrphanInvestigation{
			{
				Path:                  "/test/inv.md",
				Topic:                 "spawn",
				SimilarInvestigations: []string{"/test/inv2.md", "/test/inv3.md"},
				Suggestion:            "2 investigations on 'spawn' exist but not cited",
			},
		},
	}

	data, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	scanned, ok := parsed["total_scanned"]
	if !ok {
		t.Fatal("expected total_scanned in JSON output")
	}
	if scanned.(float64) != 10 {
		t.Errorf("expected total_scanned=10, got %v", scanned)
	}
}

func TestReflectOrphansOnly_NoKBDir(t *testing.T) {
	tempDir := t.TempDir()
	// Should not error on missing .kb directory
	err := runReflectOrphansOnlyForTest(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// runReflectOrphansOnlyForTest is a test helper that runs orphan detection
// without printing to stdout (avoids polluting test output).
func runReflectOrphansOnlyForTest(projectDir string) error {
	_, err := verify.DetectOrphanInvestigations(projectDir)
	return err
}
