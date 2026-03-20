package verify

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCheckConsequenceSensors_NonArchitectSkill(t *testing.T) {
	skills := []string{"feature-impl", "systematic-debugging", "investigation", ""}
	for _, skill := range skills {
		result := CheckConsequenceSensors("/tmp/nonexistent", "/tmp/nonexistent", skill)
		if !result.Passed {
			t.Errorf("CheckConsequenceSensors(%q) should pass for non-architect skill", skill)
		}
	}
}

func TestCheckConsequenceSensors_NoInvestigation(t *testing.T) {
	dir := t.TempDir()
	result := CheckConsequenceSensors(dir, dir, "architect")
	if !result.Passed {
		t.Error("should pass when no investigation files exist")
	}
}

func TestCheckConsequenceSensors_GateWithSensor(t *testing.T) {
	projectDir := initCSGitRepo(t)
	workspace := csSetupWorkspaceWithBaseline(t, projectDir)

	content := `# Design: Add Spawn Gate

## Recommendations

**RECOMMENDED:** Add duplication spawn gate
- **Why:** Duplicate spawns waste resources

### Enforcement Mechanisms

| Mechanism | Type | Consequence Sensor |
|-----------|------|--------------------|
| Duplication spawn gate | gate | events.jsonl spawn.gate_decision events — track fire rate and false positive rate via orch stats |
`
	csCommitInvestigation(t, projectDir, "2026-03-20-design-test.md", content)

	result := CheckConsequenceSensors(workspace, projectDir, "architect")
	if !result.Passed {
		t.Errorf("should pass with consequence sensor present, got errors: %v", result.Errors)
	}
	if len(result.OpenLoops) != 0 {
		t.Errorf("should have 0 open loops, got %d: %v", len(result.OpenLoops), result.OpenLoops)
	}
}

func TestCheckConsequenceSensors_GateWithoutSensor(t *testing.T) {
	projectDir := initCSGitRepo(t)
	workspace := csSetupWorkspaceWithBaseline(t, projectDir)

	content := `# Design: Add Enforcement

## Recommendations

### Enforcement Mechanisms

| Mechanism | Type | Consequence Sensor |
|-----------|------|--------------------|
| Accretion pre-commit hook | hook | none — open loop |
`
	csCommitInvestigation(t, projectDir, "2026-03-20-design-test.md", content)

	result := CheckConsequenceSensors(workspace, projectDir, "architect")
	// Should still pass (warning, not blocking) but surface open loops
	if !result.Passed {
		t.Errorf("open loops should warn, not block, got errors: %v", result.Errors)
	}
	if len(result.OpenLoops) != 1 {
		t.Errorf("should have 1 open loop, got %d", len(result.OpenLoops))
	}
	if len(result.Warnings) != 1 {
		t.Errorf("should have 1 warning, got %d: %v", len(result.Warnings), result.Warnings)
	}
}

func TestCheckConsequenceSensors_MissingTable(t *testing.T) {
	projectDir := initCSGitRepo(t)
	workspace := csSetupWorkspaceWithBaseline(t, projectDir)

	// Recommends a gate/hook in prose but no Enforcement Mechanisms table
	content := `# Design: Add Gate

## Recommendations

**RECOMMENDED:** Add a pre-commit hook to block accretion
- **Why:** Files grow unbounded without gates

We should also add a spawn gate to prevent duplicate spawns.
`
	csCommitInvestigation(t, projectDir, "2026-03-20-design-test.md", content)

	result := CheckConsequenceSensors(workspace, projectDir, "architect")
	if result.Passed {
		t.Error("should fail when gate/hook mentioned without Enforcement Mechanisms table")
	}
	if len(result.GatesFailed) != 1 || result.GatesFailed[0] != GateConsequenceSensor {
		t.Errorf("expected gate %q failed, got %v", GateConsequenceSensor, result.GatesFailed)
	}
}

func TestCheckConsequenceSensors_MultipleEnforcements(t *testing.T) {
	projectDir := initCSGitRepo(t)
	workspace := csSetupWorkspaceWithBaseline(t, projectDir)

	content := `# Design: Governance

## Recommendations

### Enforcement Mechanisms

| Mechanism | Type | Consequence Sensor |
|-----------|------|--------------------|
| Duplication spawn gate | gate | events.jsonl spawn.gate_decision — fire rate via orch stats |
| Accretion pre-commit hook | hook | none — open loop |
| Build verification gate | gate | events.jsonl verification.failed — tracked in completion pipeline |
`
	csCommitInvestigation(t, projectDir, "2026-03-20-design-test.md", content)

	result := CheckConsequenceSensors(workspace, projectDir, "architect")
	if !result.Passed {
		t.Errorf("should pass (open loops are warnings), got errors: %v", result.Errors)
	}
	if len(result.OpenLoops) != 1 {
		t.Errorf("should have 1 open loop, got %d: %v", len(result.OpenLoops), result.OpenLoops)
	}
}

func TestCheckConsequenceSensors_MissingSensorColumn(t *testing.T) {
	projectDir := initCSGitRepo(t)
	workspace := csSetupWorkspaceWithBaseline(t, projectDir)

	// Table exists but without Consequence Sensor column
	content := `# Design: Add Gate

## Recommendations

### Enforcement Mechanisms

| Mechanism | Type |
|-----------|------|
| Spawn gate | gate |
`
	csCommitInvestigation(t, projectDir, "2026-03-20-design-test.md", content)

	result := CheckConsequenceSensors(workspace, projectDir, "architect")
	if result.Passed {
		t.Error("should fail when Enforcement Mechanisms table is missing Consequence Sensor column")
	}
}

// TestCheckConsequenceSensors_ScopedToAgentCommits verifies that pre-existing
// investigations in .kb/investigations/ are NOT scanned by the gate — only
// files modified after the agent's baseline commit.
func TestCheckConsequenceSensors_ScopedToAgentCommits(t *testing.T) {
	projectDir := initCSGitRepo(t)

	// Pre-existing investigation (committed BEFORE agent starts) — missing table
	preExisting := `# Design: Old Gate

## Recommendations

**RECOMMENDED:** Add a pre-commit hook to block accretion
- **Why:** Files grow unbounded without gates
`
	csCommitInvestigation(t, projectDir, "2026-01-01-design-old-gate.md", preExisting)

	// Record baseline (this is where agent "starts")
	workspace := csSetupWorkspaceWithBaseline(t, projectDir)

	// Agent's investigation (committed AFTER baseline) — has proper table
	agentInv := `# Design: New Gate

## Recommendations

**RECOMMENDED:** Add spawn gate
- **Why:** Duplicate spawns

### Enforcement Mechanisms

| Mechanism | Type | Consequence Sensor |
|-----------|------|--------------------|
| Spawn gate | gate | events.jsonl — fire rate |
`
	csCommitInvestigation(t, projectDir, "2026-03-20-design-new-gate.md", agentInv)

	result := CheckConsequenceSensors(workspace, projectDir, "architect")
	// Should pass — only the agent's investigation (with proper table) is scanned.
	// The pre-existing investigation (missing table) is NOT scanned.
	if !result.Passed {
		t.Errorf("should pass — pre-existing investigations should not be scanned, got errors: %v", result.Errors)
	}
}

// TestCheckConsequenceSensors_NoGatesOrHooks verifies that investigations
// without gate/hook mentions pass even when committed by the agent.
func TestCheckConsequenceSensors_NoGatesOrHooks(t *testing.T) {
	projectDir := initCSGitRepo(t)
	workspace := csSetupWorkspaceWithBaseline(t, projectDir)

	content := `# Design: Improve Logging

## Recommendations

**RECOMMENDED:** Add structured logging
- **Why:** Current logging is unstructured
- **Expected outcome:** Better observability
`
	csCommitInvestigation(t, projectDir, "2026-03-20-design-logging.md", content)

	result := CheckConsequenceSensors(workspace, projectDir, "architect")
	if !result.Passed {
		t.Errorf("should pass when no gates/hooks recommended, got errors: %v", result.Errors)
	}
}

// --- test helpers (prefixed cs to avoid collisions with other test files) ---

func initCSGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	csGitRun(t, dir, "init")
	csGitRun(t, dir, "config", "user.email", "test@test.com")
	csGitRun(t, dir, "config", "user.name", "Test")

	readme := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readme, []byte("# test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	csGitRun(t, dir, "add", ".")
	csGitRun(t, dir, "commit", "-m", "initial commit")

	return dir
}

func csSetupWorkspaceWithBaseline(t *testing.T, projectDir string) string {
	t.Helper()
	workspace := filepath.Join(projectDir, ".orch", "workspace", "test-agent")
	if err := os.MkdirAll(workspace, 0755); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = projectDir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git rev-parse HEAD: %v", err)
	}
	baseline := string(out[:len(out)-1])

	manifest := `{"git_baseline":"` + baseline + `","skill":"architect"}`
	if err := os.WriteFile(filepath.Join(workspace, "AGENT_MANIFEST.json"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	return workspace
}

func csCommitInvestigation(t *testing.T, projectDir, filename, content string) {
	t.Helper()
	kbDir := filepath.Join(projectDir, ".kb", "investigations")
	if err := os.MkdirAll(kbDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(kbDir, filename), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	csGitRun(t, projectDir, "add", filepath.Join(".kb", "investigations", filename))
	csGitRun(t, projectDir, "commit", "-m", "add investigation "+filename)
}

func csGitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
