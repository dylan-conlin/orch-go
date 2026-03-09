package verify

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCheckProbeModelMerge_NoProbes(t *testing.T) {
	tmpDir := t.TempDir()
	workspacePath := filepath.Join(tmpDir, ".orch", "workspace", "og-feat-test-09mar-abc1")
	os.MkdirAll(workspacePath, 0755)

	// Write spawn time
	spawnTime := time.Now().Add(-1 * time.Hour)
	os.WriteFile(filepath.Join(workspacePath, ".spawn_time"),
		[]byte(fmt.Sprintf("%d", spawnTime.UnixNano())), 0644)

	// No probes exist
	os.MkdirAll(filepath.Join(tmpDir, ".kb", "models"), 0755)

	result := CheckProbeModelMerge(workspacePath, tmpDir)
	if result != nil {
		t.Errorf("Expected nil result when no probes exist, got %+v", result)
	}
}

func TestCheckProbeModelMerge_ConfirmsOnly(t *testing.T) {
	tmpDir := t.TempDir()
	workspacePath := filepath.Join(tmpDir, ".orch", "workspace", "og-feat-test-09mar-abc1")
	os.MkdirAll(workspacePath, 0755)

	// Write spawn time
	spawnTime := time.Now().Add(-1 * time.Hour)
	os.WriteFile(filepath.Join(workspacePath, ".spawn_time"),
		[]byte(fmt.Sprintf("%d", spawnTime.UnixNano())), 0644)

	// Create a probe with "confirms" verdict only
	probeDir := filepath.Join(tmpDir, ".kb", "models", "test-model", "probes")
	os.MkdirAll(probeDir, 0755)
	probeContent := `# Probe: Test probe

**Model:** test-model

---

## Question

Does X work?

---

## Model Impact

**Verdict:** confirms — X works as documented
`
	os.WriteFile(filepath.Join(probeDir, "2026-03-09-test-probe.md"), []byte(probeContent), 0644)

	result := CheckProbeModelMerge(workspacePath, tmpDir)
	if result != nil {
		t.Errorf("Expected nil result when only confirms verdicts exist, got %+v", result)
	}
}

func TestCheckProbeModelMerge_ExtendsWithoutModelUpdate(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize a git repo so git diff works
	if err := initGitRepo(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Create model.md and commit it (simulates pre-spawn state)
	modelDir := filepath.Join(tmpDir, ".kb", "models", "test-model")
	os.MkdirAll(filepath.Join(modelDir, "probes"), 0755)
	os.WriteFile(filepath.Join(modelDir, "model.md"),
		[]byte("# Test Model\n\nOriginal content.\n"), 0644)
	gitRun(t, tmpDir, "add", filepath.Join(modelDir, "model.md"))
	gitRun(t, tmpDir, "commit", "-m", "initial model")

	// Capture baseline commit SHA (simulates spawn time baseline)
	baseline := gitOutput(t, tmpDir, "rev-parse", "HEAD")

	workspacePath := filepath.Join(tmpDir, ".orch", "workspace", "og-feat-test-09mar-abc1")
	os.MkdirAll(workspacePath, 0755)

	// Write agent manifest with git baseline (more reliable than time-based)
	manifestContent := fmt.Sprintf(`{"workspace_name":"og-feat-test-09mar-abc1","git_baseline":"%s","spawn_time":"%s"}`,
		baseline, time.Now().Add(-1*time.Hour).Format(time.RFC3339))
	os.WriteFile(filepath.Join(workspacePath, "AGENT_MANIFEST.json"), []byte(manifestContent), 0644)

	// Create a probe with "extends" verdict (after baseline)
	probeContent := `# Probe: New finding

**Model:** test-model

---

## Question

Does X have property Y?

---

## Model Impact

**Verdict:** extends — X also has property Y, not documented
`
	os.WriteFile(filepath.Join(modelDir, "probes", "2026-03-09-new-finding.md"), []byte(probeContent), 0644)

	// Commit ONLY the probe (model.md NOT updated)
	gitRun(t, tmpDir, "add", filepath.Join(modelDir, "probes", "2026-03-09-new-finding.md"))
	gitRun(t, tmpDir, "commit", "-m", "add probe without model update")

	result := CheckProbeModelMerge(workspacePath, tmpDir)
	if result == nil {
		t.Fatal("Expected non-nil result for extends verdict without model update")
	}
	if result.Passed {
		t.Error("Expected gate to fail for extends verdict without model update")
	}
	if len(result.UnmergedProbes) != 1 {
		t.Errorf("Expected 1 unmerged probe, got %d", len(result.UnmergedProbes))
	}
	if len(result.UnmergedProbes) > 0 && result.UnmergedProbes[0].Verdict != "extends" {
		t.Errorf("Expected verdict 'extends', got %q", result.UnmergedProbes[0].Verdict)
	}
}

func TestCheckProbeModelMerge_ContradictsWithModelUpdate(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize git repo
	if err := initGitRepo(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Create model dir and initial model.md
	modelDir := filepath.Join(tmpDir, ".kb", "models", "test-model")
	os.MkdirAll(filepath.Join(modelDir, "probes"), 0755)
	os.WriteFile(filepath.Join(modelDir, "model.md"),
		[]byte("# Test Model\n\nOriginal content.\n"), 0644)

	// Initial commit (pre-spawn baseline)
	gitRun(t, tmpDir, "add", filepath.Join(modelDir, "model.md"))
	gitRun(t, tmpDir, "commit", "-m", "initial state")
	baseline := gitOutput(t, tmpDir, "rev-parse", "HEAD")

	workspacePath := filepath.Join(tmpDir, ".orch", "workspace", "og-feat-test-09mar-abc1")
	os.MkdirAll(workspacePath, 0755)

	// Write agent manifest with baseline
	manifestContent := fmt.Sprintf(`{"workspace_name":"og-feat-test-09mar-abc1","git_baseline":"%s","spawn_time":"%s"}`,
		baseline, time.Now().Add(-1*time.Hour).Format(time.RFC3339))
	os.WriteFile(filepath.Join(workspacePath, "AGENT_MANIFEST.json"), []byte(manifestContent), 0644)

	// Now create the probe and update model.md (simulating agent work)
	probeContent := `# Probe: Contradiction found

**Model:** test-model

---

## Question

Is X true?

---

## Model Impact

**Verdict:** contradicts — X is false, model claims X is true
`
	os.WriteFile(filepath.Join(modelDir, "probes", "2026-03-09-contradiction.md"), []byte(probeContent), 0644)

	// Update model.md (the merge)
	os.WriteFile(filepath.Join(modelDir, "model.md"),
		[]byte("# Test Model\n\nUpdated: X is false (corrected by probe 2026-03-09).\n"), 0644)

	// Commit both probe and model update
	gitRun(t, tmpDir, "add", filepath.Join(modelDir, "probes", "2026-03-09-contradiction.md"))
	gitRun(t, tmpDir, "add", filepath.Join(modelDir, "model.md"))
	gitRun(t, tmpDir, "commit", "-m", "probe and model update")

	result := CheckProbeModelMerge(workspacePath, tmpDir)
	if result == nil {
		// nil means no actionable probes or gate passed — either is fine
		return
	}
	if !result.Passed {
		t.Errorf("Expected gate to pass when model.md was updated, but it failed: %v", result.Errors)
	}
}

func TestCheckProbeModelMerge_MultipleProbesMixed(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize git repo
	if err := initGitRepo(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Create two models and commit (pre-spawn baseline)
	model1Dir := filepath.Join(tmpDir, ".kb", "models", "model-a")
	model2Dir := filepath.Join(tmpDir, ".kb", "models", "model-b")
	os.MkdirAll(filepath.Join(model1Dir, "probes"), 0755)
	os.MkdirAll(filepath.Join(model2Dir, "probes"), 0755)

	os.WriteFile(filepath.Join(model1Dir, "model.md"), []byte("# Model A\n"), 0644)
	os.WriteFile(filepath.Join(model2Dir, "model.md"), []byte("# Model B\n"), 0644)

	gitRun(t, tmpDir, "add", filepath.Join(model1Dir, "model.md"))
	gitRun(t, tmpDir, "add", filepath.Join(model2Dir, "model.md"))
	gitRun(t, tmpDir, "commit", "-m", "initial models")
	baseline := gitOutput(t, tmpDir, "rev-parse", "HEAD")

	workspacePath := filepath.Join(tmpDir, ".orch", "workspace", "og-feat-test-09mar-abc1")
	os.MkdirAll(workspacePath, 0755)

	// Write agent manifest with baseline
	manifestContent := fmt.Sprintf(`{"workspace_name":"og-feat-test-09mar-abc1","git_baseline":"%s","spawn_time":"%s"}`,
		baseline, time.Now().Add(-1*time.Hour).Format(time.RFC3339))
	os.WriteFile(filepath.Join(workspacePath, "AGENT_MANIFEST.json"), []byte(manifestContent), 0644)

	// Probe 1: extends model-a (model-a will be updated)
	probe1 := `# Probe: Extends A

**Model:** model-a

---

## Question

Does A have property P?

---

## Model Impact

**Verdict:** extends — A has property P
`
	os.WriteFile(filepath.Join(model1Dir, "probes", "2026-03-09-extends-a.md"), []byte(probe1), 0644)

	// Probe 2: contradicts model-b (model-b will NOT be updated)
	probe2 := `# Probe: Contradicts B

**Model:** model-b

---

## Question

Is B correct about Q?

---

## Model Impact

**Verdict:** contradicts — B is wrong about Q
`
	os.WriteFile(filepath.Join(model2Dir, "probes", "2026-03-09-contradicts-b.md"), []byte(probe2), 0644)

	// Update only model-a's model.md, NOT model-b's
	os.WriteFile(filepath.Join(model1Dir, "model.md"), []byte("# Model A\n\nHas property P.\n"), 0644)

	// Commit probes + model-a update (model-b NOT updated)
	gitRun(t, tmpDir, "add", filepath.Join(model1Dir, "probes", "2026-03-09-extends-a.md"))
	gitRun(t, tmpDir, "add", filepath.Join(model2Dir, "probes", "2026-03-09-contradicts-b.md"))
	gitRun(t, tmpDir, "add", filepath.Join(model1Dir, "model.md"))
	gitRun(t, tmpDir, "commit", "-m", "probe work")

	result := CheckProbeModelMerge(workspacePath, tmpDir)
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.Passed {
		t.Error("Expected gate to fail — model-b not updated")
	}
	if len(result.UnmergedProbes) != 1 {
		t.Errorf("Expected 1 unmerged probe (model-b), got %d", len(result.UnmergedProbes))
	}
	if len(result.UnmergedProbes) > 0 && result.UnmergedProbes[0].ModelName != "model-b" {
		t.Errorf("Expected unmerged probe for model-b, got %q", result.UnmergedProbes[0].ModelName)
	}
}

func TestCheckProbeModelMerge_EmptyInputs(t *testing.T) {
	result := CheckProbeModelMerge("", "")
	if result != nil {
		t.Error("Expected nil for empty inputs")
	}

	result = CheckProbeModelMerge("/some/path", "")
	if result != nil {
		t.Error("Expected nil for empty projectDir")
	}
}

func TestIsModelFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{".kb/models/test-model/model.md", true},
		{".kb/models/completion-verification/model.md", true},
		{".kb/models/test-model/probes/2026-03-09-probe.md", false},
		{".kb/models/test-model/README.md", false},
		{"pkg/verify/check.go", false},
		{"model.md", false}, // Not in .kb/models/
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := isModelFile(tt.path)
			if got != tt.want {
				t.Errorf("isModelFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestExtractModelNameFromPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{".kb/models/test-model/model.md", "test-model"},
		{".kb/models/completion-verification/model.md", "completion-verification"},
		{".kb/models/spawn-architecture/model.md", "spawn-architecture"},
		{".kb/models/test-model/probes/probe.md", ""},       // Not model.md
		{"pkg/verify/model.md", ""},                          // Not in .kb/models/
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := extractModelNameFromPath(tt.path)
			if got != tt.want {
				t.Errorf("extractModelNameFromPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestFormatProbeModelMergeFailure_Nil(t *testing.T) {
	output := FormatProbeModelMergeFailure(nil)
	if output != "" {
		t.Error("Expected empty output for nil result")
	}
}

func TestFormatProbeModelMergeFailure_Passed(t *testing.T) {
	result := &ProbeModelMergeResult{Passed: true}
	output := FormatProbeModelMergeFailure(result)
	if output != "" {
		t.Error("Expected empty output for passed result")
	}
}

func TestFormatProbeModelMergeFailure_Failed(t *testing.T) {
	result := &ProbeModelMergeResult{
		Passed: false,
		UnmergedProbes: []ProbeVerdict{
			{
				ModelName: "test-model",
				ProbePath: "/path/to/2026-03-09-probe.md",
				Verdict:   "extends",
				Details:   "new property found",
			},
		},
	}
	output := FormatProbeModelMergeFailure(result)
	if output == "" {
		t.Error("Expected non-empty output for failed result")
	}
	if !strings.Contains(output, "test-model") {
		t.Error("Output should contain model name")
	}
	if !strings.Contains(output, "extends") {
		t.Error("Output should contain verdict")
	}
}

// gitOutput runs a git command and returns its trimmed stdout.
func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %s failed: %v", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(output))
}

// gitCommitAll stages all files and commits them.
func gitCommitAll(t *testing.T, dir, message string) {
	t.Helper()
	gitRun(t, dir, "add", "-A")
	gitRun(t, dir, "commit", "-m", message, "--allow-empty")
}

// gitRun runs a git command in the given directory.
func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, output)
	}
}
