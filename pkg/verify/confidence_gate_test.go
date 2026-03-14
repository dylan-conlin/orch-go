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

// TestGate_ProbeContradiction_WarnsOnUnmergedModel verifies that the
// probe-to-model merge gate catches the dilution curve scenario:
// a probe with "contradicts" verdict where the model was NOT updated.
//
// This is the per-session enforcement layer of confidence propagation.
func TestGate_ProbeContradiction_WarnsOnUnmergedModel(t *testing.T) {
	tmpDir := t.TempDir()

	if err := initGitRepo(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Create model and commit (pre-spawn baseline)
	modelDir := filepath.Join(tmpDir, ".kb", "models", "orchestrator-skill")
	os.MkdirAll(filepath.Join(modelDir, "probes"), 0755)
	os.WriteFile(filepath.Join(modelDir, "model.md"), []byte(`# Model: Orchestrator Skill

## Core Claims

### Claim 4: Behavioral dilution budget is ~4 constraints

**Evidence quality:** Single-source measured (N=3).
`), 0644)

	gitRun(t, tmpDir, "add", filepath.Join(modelDir, "model.md"))
	gitRun(t, tmpDir, "commit", "-m", "initial model")
	baseline := gitOutput(t, tmpDir, "rev-parse", "HEAD")

	// Create workspace with baseline
	workspacePath := filepath.Join(tmpDir, ".orch", "workspace", "og-feat-dilution-test-13mar-abc1")
	os.MkdirAll(workspacePath, 0755)
	manifestContent := fmt.Sprintf(`{"workspace_name":"og-feat-dilution-test-13mar-abc1","git_baseline":"%s","spawn_time":"%s"}`,
		baseline, time.Now().Add(-1*time.Hour).Format(time.RFC3339))
	os.WriteFile(filepath.Join(workspacePath, "AGENT_MANIFEST.json"), []byte(manifestContent), 0644)

	// Agent creates a probe that contradicts Claim 4 (replication failed)
	probeContent := `# Probe: Dilution Curve Replication

**Model:** orchestrator-skill

---

## Question

Can the dilution budget threshold be replicated?

---

## Model Impact

**Verdict:** contradicts — Replication failed, N=3 too small, thresholds are unreplicated hypotheses
`
	probePath := filepath.Join(modelDir, "probes", "2026-03-04-probe-dilution-replication.md")
	os.WriteFile(probePath, []byte(probeContent), 0644)

	// Agent commits probe but does NOT update model.md
	gitRun(t, tmpDir, "add", probePath)
	gitRun(t, tmpDir, "commit", "-m", "add probe without model update")

	result := CheckProbeModelMerge(workspacePath, tmpDir)
	if result == nil {
		t.Fatal("expected non-nil result for contradicts verdict without model update")
	}
	if result.Passed {
		t.Error("expected gate to FAIL — model not updated after contradiction")
	}
	if len(result.UnmergedProbes) != 1 {
		t.Errorf("expected 1 unmerged probe, got %d", len(result.UnmergedProbes))
	}
	if len(result.UnmergedProbes) > 0 {
		p := result.UnmergedProbes[0]
		if p.Verdict != "contradicts" {
			t.Errorf("expected verdict 'contradicts', got %q", p.Verdict)
		}
		if p.ModelName != "orchestrator-skill" {
			t.Errorf("expected model 'orchestrator-skill', got %q", p.ModelName)
		}
	}

	// Verify the formatted failure message is actionable
	formatted := FormatProbeModelMergeFailure(result)
	if !strings.Contains(formatted, "orchestrator-skill") {
		t.Error("failure message should contain model name")
	}
	if !strings.Contains(formatted, "contradicts") {
		t.Error("failure message should contain verdict")
	}
	if !strings.Contains(formatted, "Merge probe findings") {
		t.Error("failure message should include fix guidance")
	}
}

// TestGate_ProbeContradiction_PassesWhenModelUpdated verifies the gate
// passes when an agent properly merges a contradicting probe into the model.
func TestGate_ProbeContradiction_PassesWhenModelUpdated(t *testing.T) {
	tmpDir := t.TempDir()

	if err := initGitRepo(tmpDir); err != nil {
		t.Fatal(err)
	}

	modelDir := filepath.Join(tmpDir, ".kb", "models", "orchestrator-skill")
	os.MkdirAll(filepath.Join(modelDir, "probes"), 0755)
	os.WriteFile(filepath.Join(modelDir, "model.md"), []byte(`# Model: Orchestrator Skill

### Claim 4: Behavioral dilution budget is ~4 constraints

**Evidence quality:** Single-source measured (N=3).
`), 0644)

	gitRun(t, tmpDir, "add", filepath.Join(modelDir, "model.md"))
	gitRun(t, tmpDir, "commit", "-m", "initial model")
	baseline := gitOutput(t, tmpDir, "rev-parse", "HEAD")

	workspacePath := filepath.Join(tmpDir, ".orch", "workspace", "og-feat-dilution-test-13mar-abc1")
	os.MkdirAll(workspacePath, 0755)
	manifestContent := fmt.Sprintf(`{"workspace_name":"og-feat-dilution-test-13mar-abc1","git_baseline":"%s","spawn_time":"%s"}`,
		baseline, time.Now().Add(-1*time.Hour).Format(time.RFC3339))
	os.WriteFile(filepath.Join(workspacePath, "AGENT_MANIFEST.json"), []byte(manifestContent), 0644)

	// Agent creates probe AND updates model (proper merge)
	probeContent := `# Probe: Dilution Curve Replication

**Model:** orchestrator-skill

---

## Question

Can the dilution budget threshold be replicated?

---

## Model Impact

**Verdict:** contradicts — Replication failed
`
	probePath := filepath.Join(modelDir, "probes", "2026-03-04-probe-dilution-replication.md")
	os.WriteFile(probePath, []byte(probeContent), 0644)

	// Agent also updates model.md with downgraded evidence quality
	updatedModel := `# Model: Orchestrator Skill

### Claim 4: Behavioral dilution budget is ~4 constraints

**Evidence quality:** Single-source measured (N=3, replication failed Mar 4). HYPOTHESIZED — do not cite as established.
`
	os.WriteFile(filepath.Join(modelDir, "model.md"), []byte(updatedModel), 0644)

	gitRun(t, tmpDir, "add", probePath)
	gitRun(t, tmpDir, "add", filepath.Join(modelDir, "model.md"))
	gitRun(t, tmpDir, "commit", "-m", "probe + model merge with evidence quality update")

	result := CheckProbeModelMerge(workspacePath, tmpDir)
	if result != nil && !result.Passed {
		t.Errorf("expected gate to pass when model was updated, but got errors: %v", result.Errors)
	}
}

// initGitRepoForConfidence initializes a git repo with author config.
func initGitRepoForConfidence(dir string) error {
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return err
	}
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return err
	}
	cmd = exec.Command("git", "config", "user.name", "Test")
	cmd.Dir = dir
	return cmd.Run()
}
