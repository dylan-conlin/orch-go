package spawn

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindProjectProbes_NoModelsDir(t *testing.T) {
	tmpDir := t.TempDir()
	probes := FindProjectProbes(tmpDir)
	if probes != nil {
		t.Errorf("Expected nil for no models dir, got %d probes", len(probes))
	}
}

func TestFindProjectProbes_EmptyModelsDir(t *testing.T) {
	tmpDir := t.TempDir()
	modelsDir := filepath.Join(tmpDir, ".kb", "models")
	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		t.Fatal(err)
	}
	probes := FindProjectProbes(tmpDir)
	if probes != nil {
		t.Errorf("Expected nil for empty models dir, got %d probes", len(probes))
	}
}

func TestFindProjectProbes_WithProbes(t *testing.T) {
	tmpDir := t.TempDir()
	modelsDir := filepath.Join(tmpDir, ".kb", "models")

	// Create model file
	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		t.Fatal(err)
	}
	modelContent := "# Model: Test\n\n**Last Updated:** 2026-01-01\n\n## Summary\nTest model.\n"
	if err := os.WriteFile(filepath.Join(modelsDir, "test-model.md"), []byte(modelContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create probes directory with probes
	probesDir := filepath.Join(modelsDir, "test-model", "probes")
	if err := os.MkdirAll(probesDir, 0755); err != nil {
		t.Fatal(err)
	}

	probeContent := `# Probe: Check session lifecycle

**Model:** test-model
**Date:** 2026-02-08

---

## Question

Does the session lifecycle handle cleanup correctly?

---

## What I Tested

` + "```bash" + `
go test ./pkg/session/...
` + "```" + `

---

## What I Observed

All tests pass. Session cleanup is called on shutdown.

---

## Model Impact

**Verdict:** confirms — session cleanup invariant

**Details:**
Session cleanup is properly invoked during shutdown. The model's claim about cleanup ordering is accurate.

**Confidence:** High — direct test evidence
`
	if err := os.WriteFile(filepath.Join(probesDir, "2026-02-08-check-session-lifecycle.md"), []byte(probeContent), 0644); err != nil {
		t.Fatal(err)
	}
	// Add .gitkeep (should be ignored)
	if err := os.WriteFile(filepath.Join(probesDir, ".gitkeep"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	probes := FindProjectProbes(tmpDir)
	if len(probes) != 1 {
		t.Fatalf("Expected 1 probe, got %d", len(probes))
	}

	p := probes[0]
	if p.ModelName != "test-model" {
		t.Errorf("Expected ModelName 'test-model', got '%s'", p.ModelName)
	}
	if !strings.Contains(p.ModelPath, "test-model.md") {
		t.Errorf("Expected ModelPath to contain 'test-model.md', got '%s'", p.ModelPath)
	}
	if p.Probe.Name != "2026-02-08-check-session-lifecycle" {
		t.Errorf("Expected probe name '2026-02-08-check-session-lifecycle', got '%s'", p.Probe.Name)
	}
	if !strings.Contains(p.Impact, "confirms") {
		t.Errorf("Expected Impact to contain 'confirms', got '%s'", p.Impact)
	}
}

func TestFindProjectProbes_NoModelFile(t *testing.T) {
	tmpDir := t.TempDir()
	modelsDir := filepath.Join(tmpDir, ".kb", "models")

	// Create probes directory WITHOUT corresponding model file
	probesDir := filepath.Join(modelsDir, "orphan-model", "probes")
	if err := os.MkdirAll(probesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(probesDir, "2026-02-08-test.md"), []byte("# Probe"), 0644); err != nil {
		t.Fatal(err)
	}

	probes := FindProjectProbes(tmpDir)
	if len(probes) != 0 {
		t.Errorf("Expected 0 probes (no model file), got %d", len(probes))
	}
}

func TestReadProbeModelImpact_ValidProbe(t *testing.T) {
	tmpDir := t.TempDir()
	probeContent := `# Probe: Test

## Question
Does X work?

## What I Tested
Ran the test suite.

## What I Observed
All passed.

## Model Impact

**Verdict:** extends — new invariant discovered

**Details:**
Found that X also handles Y, which the model doesn't mention.

## Notes
Follow-up needed.
`
	probePath := filepath.Join(tmpDir, "test-probe.md")
	if err := os.WriteFile(probePath, []byte(probeContent), 0644); err != nil {
		t.Fatal(err)
	}

	impact := ReadProbeModelImpact(probePath)
	if !strings.Contains(impact, "extends") {
		t.Errorf("Expected impact to contain 'extends', got '%s'", impact)
	}
	if !strings.Contains(impact, "Found that X also handles Y") {
		t.Errorf("Expected impact to contain details, got '%s'", impact)
	}
	// Should NOT contain the Notes section
	if strings.Contains(impact, "Follow-up needed") {
		t.Error("Impact should not include content from Notes section")
	}
}

func TestReadProbeModelImpact_NoImpactSection(t *testing.T) {
	tmpDir := t.TempDir()
	probeContent := `# Probe: Test

## Question
Does X work?

## Notes
Some notes.
`
	probePath := filepath.Join(tmpDir, "test-probe.md")
	if err := os.WriteFile(probePath, []byte(probeContent), 0644); err != nil {
		t.Fatal(err)
	}

	impact := ReadProbeModelImpact(probePath)
	if impact != "" {
		t.Errorf("Expected empty impact, got '%s'", impact)
	}
}

func TestReadProbeModelImpact_NonexistentFile(t *testing.T) {
	impact := ReadProbeModelImpact("/nonexistent/probe.md")
	if impact != "" {
		t.Errorf("Expected empty impact for nonexistent file, got '%s'", impact)
	}
}

func TestFormatProbeMergeSummary_Empty(t *testing.T) {
	result := FormatProbeMergeSummary(nil)
	if result != "" {
		t.Errorf("Expected empty string for nil probes, got '%s'", result)
	}
}

func TestFormatProbeMergeSummary_WithProbes(t *testing.T) {
	probes := []ProjectProbe{
		{
			ModelName: "completion-verification",
			Probe: probeEntry{
				Name: "2026-02-08-check-gates",
			},
			Impact: "**Verdict:** confirms — gate ordering is correct",
		},
		{
			ModelName: "spawn-architecture",
			Probe: probeEntry{
				Name: "2026-02-08-test-workspace",
			},
			Impact: "",
		},
	}

	result := FormatProbeMergeSummary(probes)
	if !strings.Contains(result, "2 probe(s)") {
		t.Errorf("Expected '2 probe(s)' in summary, got '%s'", result)
	}
	if !strings.Contains(result, "completion-verification") {
		t.Errorf("Expected model name in summary, got '%s'", result)
	}
	if !strings.Contains(result, "Impact:") {
		t.Errorf("Expected 'Impact:' line for probe with impact, got '%s'", result)
	}
}

func TestMergeProbeIntoModel_CreatesSection(t *testing.T) {
	tmpDir := t.TempDir()
	modelContent := `# Model: Test Model

**Domain:** Testing
**Last Updated:** 2026-01-15
**Synthesized From:** 5 investigations

---

## Summary (30 seconds)

This is a test model.

---

## Core Mechanism

Some mechanism here.
`
	modelPath := filepath.Join(tmpDir, "test-model.md")
	if err := os.WriteFile(modelPath, []byte(modelContent), 0644); err != nil {
		t.Fatal(err)
	}

	probe := ProjectProbe{
		ModelName: "test-model",
		ModelPath: modelPath,
		Probe: probeEntry{
			Name: "2026-02-08-check-mechanism",
		},
		Impact: "**Verdict:** confirms — mechanism works as documented\n\n**Details:**\nTested the mechanism and it works correctly.",
	}

	if err := MergeProbeIntoModel(modelPath, probe); err != nil {
		t.Fatalf("MergeProbeIntoModel failed: %v", err)
	}

	// Read back
	data, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	// Check Last Updated was updated to today
	if strings.Contains(content, "2026-01-15") {
		t.Error("Last Updated should have been updated (still contains old date)")
	}

	// Check Merged Probes section exists
	if !strings.Contains(content, "## Merged Probes") {
		t.Error("Expected '## Merged Probes' section")
	}

	// Check probe entry exists
	if !strings.Contains(content, "### Probe: 2026-02-08-check-mechanism") {
		t.Error("Expected probe entry header")
	}

	// Check impact content
	if !strings.Contains(content, "confirms — mechanism works as documented") {
		t.Error("Expected impact content in merged section")
	}
}

func TestMergeProbeIntoModel_AppendsToExistingSection(t *testing.T) {
	tmpDir := t.TempDir()
	modelContent := `# Model: Test Model

**Last Updated:** 2026-02-01

## Summary (30 seconds)

Test model.

---

## Merged Probes

### Probe: 2026-02-01-first-probe (2026-02-01)

**Verdict:** confirms — first finding
`
	modelPath := filepath.Join(tmpDir, "test-model.md")
	if err := os.WriteFile(modelPath, []byte(modelContent), 0644); err != nil {
		t.Fatal(err)
	}

	probe := ProjectProbe{
		ModelName: "test-model",
		ModelPath: modelPath,
		Probe: probeEntry{
			Name: "2026-02-08-second-probe",
		},
		Impact: "**Verdict:** extends — new finding discovered",
	}

	if err := MergeProbeIntoModel(modelPath, probe); err != nil {
		t.Fatalf("MergeProbeIntoModel failed: %v", err)
	}

	data, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	// Both probes should be present
	if !strings.Contains(content, "first-probe") {
		t.Error("Expected first probe to still be present")
	}
	if !strings.Contains(content, "second-probe") {
		t.Error("Expected second probe to be appended")
	}

	// Only one "## Merged Probes" header
	count := strings.Count(content, "## Merged Probes")
	if count != 1 {
		t.Errorf("Expected exactly 1 '## Merged Probes' header, got %d", count)
	}
}

func TestMergeProbeIntoModel_NonexistentFile(t *testing.T) {
	probe := ProjectProbe{
		ModelName: "test",
		ModelPath: "/nonexistent/model.md",
		Probe:     probeEntry{Name: "test"},
		Impact:    "test",
	}
	err := MergeProbeIntoModel("/nonexistent/model.md", probe)
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}
