package verify

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseProbeVerdict(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		wantVerdict  string
		wantDetails  string
		wantTitle    string
		wantModel    string
		wantQuestion string
	}{
		{
			name: "structured verdict format",
			content: `# Probe: Are completion gates catching defects?

**Model:** completion-verification
**Date:** 2026-02-09
**Status:** Complete

---

## Question

Since targeted --skip-{gate} bypasses were introduced, what is the real usage mix?

---

## What I Tested

Some tests here.

---

## What I Observed

Some observations here.

---

## Model Impact

**Verdict:** extends — targeted bypasses reduced blanket force, but gate noise remains.

**Details:**
This confirms the model's claim that targeted bypasses replaced blunt forcing.

**Confidence:** High
`,
			wantVerdict:  "extends",
			wantDetails:  "targeted bypasses reduced blanket force, but gate noise remains.",
			wantTitle:    "Are completion gates catching defects?",
			wantModel:    "completion-verification",
			wantQuestion: "Since targeted --skip-{gate} bypasses were introduced, what is the real usage mix?",
		},
		{
			name: "checkbox format - confirms",
			content: `# Probe: Does bd sync work correctly?

**Model:** beads-integration-architecture
**Date:** 2026-02-08
**Status:** Complete

---

## Question

Does bd sync handle hash mismatches?

---

## What I Tested

Ran bd sync with corrupted state.

---

## What I Observed

It recovered gracefully.

---

## Model Impact

- [x] **Confirms** invariant: bd sync recovers from hash mismatches automatically
- [ ] **Contradicts** invariant: [which one] — [what's actually true]
- [ ] **Extends** model with: [new finding not covered by existing model]
`,
			wantVerdict:  "confirms",
			wantDetails:  "bd sync recovers from hash mismatches automatically",
			wantTitle:    "Does bd sync work correctly?",
			wantModel:    "beads-integration-architecture",
			wantQuestion: "Does bd sync handle hash mismatches?",
		},
		{
			name: "checkbox format - contradicts",
			content: `# Probe: Is the default model opus?

**Model:** model-selection
**Date:** 2026-02-10
**Status:** Complete

---

## Question

Is opus the default model?

---

## What I Tested

Checked model.go

---

## What I Observed

Default is flash.

---

## Model Impact

- [ ] **Confirms** invariant: [which one]
- [x] **Contradicts** invariant: opus is default — actually flash is default since Jan 2026
- [ ] **Extends** model with: [new finding not covered by existing model]
`,
			wantVerdict:  "contradicts",
			wantDetails:  "opus is default — actually flash is default since Jan 2026",
			wantTitle:    "Is the default model opus?",
			wantModel:    "model-selection",
			wantQuestion: "Is opus the default model?",
		},
		{
			name: "checkbox format - extends",
			content: `# Probe: Event tap behavior

**Model:** macos-click-freeze
**Date:** 2026-02-12
**Status:** Complete

---

## Question

Does skhd use passive event taps?

---

## What I Tested

Read skhd source code.

---

## What I Observed

Uses active taps.

---

## Model Impact

- [ ] **Confirms** invariant: [which one]
- [ ] **Contradicts** invariant: [which one] — [what's actually true]
- [x] **Extends** model with: skhd uses kCGEventTapOptionDefault which is active interception
`,
			wantVerdict:  "extends",
			wantDetails:  "skhd uses kCGEventTapOptionDefault which is active interception",
			wantTitle:    "Event tap behavior",
			wantModel:    "macos-click-freeze",
			wantQuestion: "Does skhd use passive event taps?",
		},
		{
			name:         "no model impact section",
			content:      "# Probe: Something\n\n## Question\n\nSome question\n",
			wantVerdict:  "",
			wantDetails:  "",
			wantTitle:    "Something",
			wantModel:    "",
			wantQuestion: "Some question",
		},
		{
			name: "model with path format",
			content: `# Probe: Test probe

**Model:** ` + "`.kb/models/completion-verification.md`" + `
**Date:** 2026-02-09

---

## Question

Test question

---

## Model Impact

**Verdict:** confirms — the gate works
`,
			wantVerdict:  "confirms",
			wantDetails:  "the gate works",
			wantTitle:    "Test probe",
			wantModel:    "completion-verification",
			wantQuestion: "Test question",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verdict := ParseProbeVerdict([]byte(tt.content))
			if verdict.Verdict != tt.wantVerdict {
				t.Errorf("Verdict = %q, want %q", verdict.Verdict, tt.wantVerdict)
			}
			if verdict.Details != tt.wantDetails {
				t.Errorf("Details = %q, want %q", verdict.Details, tt.wantDetails)
			}
			if verdict.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", verdict.Title, tt.wantTitle)
			}
			if verdict.ModelName != tt.wantModel {
				t.Errorf("ModelName = %q, want %q", verdict.ModelName, tt.wantModel)
			}
			if verdict.Question != tt.wantQuestion {
				t.Errorf("Question = %q, want %q", verdict.Question, tt.wantQuestion)
			}
		})
	}
}

func TestFindProbesForWorkspace(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .kb/models structure with probes
	modelDir := filepath.Join(tmpDir, ".kb", "models", "test-model", "probes")
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a recent probe (modified "now")
	recentProbe := filepath.Join(modelDir, "2026-02-13-test-probe.md")
	probeContent := `# Probe: Test probe

**Model:** test-model
**Date:** 2026-02-13
**Status:** Complete

---

## Question

Does X work?

---

## What I Tested

Tested X.

---

## What I Observed

X works.

---

## Model Impact

**Verdict:** confirms — X works as documented
`
	if err := os.WriteFile(recentProbe, []byte(probeContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create an old probe (modified well before spawn time)
	oldProbe := filepath.Join(modelDir, "2026-01-01-old-probe.md")
	if err := os.WriteFile(oldProbe, []byte("# Probe: Old\n\n## Question\n\nOld question\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Set old probe's mod time to the past
	oldTime := time.Now().Add(-30 * 24 * time.Hour)
	os.Chtimes(oldProbe, oldTime, oldTime)

	// Create workspace with spawn time
	workspacePath := filepath.Join(tmpDir, ".orch", "workspace", "og-feat-test-13feb-abc1")
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatal(err)
	}
	spawnTime := time.Now().Add(-1 * time.Hour)
	// Write spawn time as Unix nanoseconds (the format readLegacyDotfiles expects)
	if err := os.WriteFile(
		filepath.Join(workspacePath, ".spawn_time"),
		[]byte(fmt.Sprintf("%d", spawnTime.UnixNano())),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	// Find probes created after spawn time
	probes := FindProbesForWorkspace(workspacePath, tmpDir)

	if len(probes) != 1 {
		t.Fatalf("Expected 1 probe, got %d", len(probes))
	}

	if probes[0].ModelName != "test-model" {
		t.Errorf("ModelName = %q, want %q", probes[0].ModelName, "test-model")
	}
	if probes[0].Verdict != "confirms" {
		t.Errorf("Verdict = %q, want %q", probes[0].Verdict, "confirms")
	}
}

func TestFindProbesForWorkspaceNoProbes(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .kb/models structure without probes
	modelDir := filepath.Join(tmpDir, ".kb", "models")
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create workspace
	workspacePath := filepath.Join(tmpDir, ".orch", "workspace", "og-feat-test-13feb-abc1")
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatal(err)
	}
	spawnTime := time.Now().Add(-1 * time.Hour)
	if err := os.WriteFile(
		filepath.Join(workspacePath, ".spawn_time"),
		[]byte(spawnTime.Format(time.RFC3339)),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	probes := FindProbesForWorkspace(workspacePath, tmpDir)
	if len(probes) != 0 {
		t.Errorf("Expected 0 probes, got %d", len(probes))
	}
}

func TestFindProbesForWorkspaceNoSpawnTime(t *testing.T) {
	tmpDir := t.TempDir()

	// Create probe
	modelDir := filepath.Join(tmpDir, ".kb", "models", "test-model", "probes")
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		t.Fatal(err)
	}
	probeContent := `# Probe: Test

**Model:** test-model

---

## Question

Q?

---

## Model Impact

**Verdict:** confirms — yes
`
	if err := os.WriteFile(filepath.Join(modelDir, "2026-02-13-test.md"), []byte(probeContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create workspace WITHOUT spawn time
	workspacePath := filepath.Join(tmpDir, ".orch", "workspace", "og-feat-test-13feb-abc1")
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatal(err)
	}

	// Without spawn time, should return no probes (can't determine which are relevant)
	probes := FindProbesForWorkspace(workspacePath, tmpDir)
	if len(probes) != 0 {
		t.Errorf("Expected 0 probes without spawn time, got %d", len(probes))
	}
}

func TestFormatProbeVerdicts(t *testing.T) {
	verdicts := []ProbeVerdict{
		{
			ModelName: "completion-verification",
			Title:     "Are gates catching defects?",
			Verdict:   "extends",
			Details:   "gate noise remains concentrated",
			Question:  "What is the real usage mix?",
			ProbePath: "/path/to/probe.md",
		},
		{
			ModelName: "beads-integration",
			Title:     "Does sync recover?",
			Verdict:   "confirms",
			Details:   "bd sync recovers automatically",
			Question:  "Does bd sync handle hash mismatches?",
			ProbePath: "/path/to/probe2.md",
		},
	}

	output := FormatProbeVerdicts(verdicts)

	// Should contain model names
	if output == "" {
		t.Error("Expected non-empty output")
	}

	// Check for key content
	for _, v := range verdicts {
		if !containsString(output, v.ModelName) {
			t.Errorf("Output missing model name %q", v.ModelName)
		}
		if !containsString(output, v.Verdict) {
			t.Errorf("Output missing verdict %q", v.Verdict)
		}
	}
}

func TestFormatProbeVerdictsEmpty(t *testing.T) {
	output := FormatProbeVerdicts(nil)
	if output != "" {
		t.Errorf("Expected empty output for nil verdicts, got %q", output)
	}
}

func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
