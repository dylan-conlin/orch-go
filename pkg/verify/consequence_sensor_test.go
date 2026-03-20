package verify

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckConsequenceSensors_NonArchitectSkill(t *testing.T) {
	skills := []string{"feature-impl", "systematic-debugging", "investigation", ""}
	for _, skill := range skills {
		result := CheckConsequenceSensors("/tmp/nonexistent", skill)
		if !result.Passed {
			t.Errorf("CheckConsequenceSensors(%q) should pass for non-architect skill", skill)
		}
	}
}

func TestCheckConsequenceSensors_NoInvestigation(t *testing.T) {
	dir := t.TempDir()
	result := CheckConsequenceSensors(dir, "architect")
	if !result.Passed {
		t.Error("should pass when no investigation files exist")
	}
}

func TestCheckConsequenceSensors_NoGatesOrHooks(t *testing.T) {
	dir := t.TempDir()
	// Investigation that doesn't recommend any gates or hooks
	content := `# Design: Improve Logging

## Recommendations

**RECOMMENDED:** Add structured logging
- **Why:** Current logging is unstructured
- **Expected outcome:** Better observability
`
	writeInvestigation(t, dir, content)

	result := CheckConsequenceSensors(dir, "architect")
	if !result.Passed {
		t.Errorf("should pass when no gates/hooks recommended, got errors: %v", result.Errors)
	}
}

func TestCheckConsequenceSensors_GateWithSensor(t *testing.T) {
	dir := t.TempDir()
	content := `# Design: Add Spawn Gate

## Recommendations

**RECOMMENDED:** Add duplication spawn gate
- **Why:** Duplicate spawns waste resources

### Enforcement Mechanisms

| Mechanism | Type | Consequence Sensor |
|-----------|------|--------------------|
| Duplication spawn gate | gate | events.jsonl spawn.gate_decision events — track fire rate and false positive rate via orch stats |
`
	writeInvestigation(t, dir, content)

	result := CheckConsequenceSensors(dir, "architect")
	if !result.Passed {
		t.Errorf("should pass with consequence sensor present, got errors: %v", result.Errors)
	}
	if len(result.OpenLoops) != 0 {
		t.Errorf("should have 0 open loops, got %d: %v", len(result.OpenLoops), result.OpenLoops)
	}
}

func TestCheckConsequenceSensors_GateWithoutSensor(t *testing.T) {
	dir := t.TempDir()
	content := `# Design: Add Enforcement

## Recommendations

### Enforcement Mechanisms

| Mechanism | Type | Consequence Sensor |
|-----------|------|--------------------|
| Accretion pre-commit hook | hook | none — open loop |
`
	writeInvestigation(t, dir, content)

	result := CheckConsequenceSensors(dir, "architect")
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
	dir := t.TempDir()
	// Recommends a gate/hook in prose but no Enforcement Mechanisms table
	content := `# Design: Add Gate

## Recommendations

**RECOMMENDED:** Add a pre-commit hook to block accretion
- **Why:** Files grow unbounded without gates

We should also add a spawn gate to prevent duplicate spawns.
`
	writeInvestigation(t, dir, content)

	result := CheckConsequenceSensors(dir, "architect")
	if result.Passed {
		t.Error("should fail when gate/hook mentioned without Enforcement Mechanisms table")
	}
	if len(result.GatesFailed) != 1 || result.GatesFailed[0] != GateConsequenceSensor {
		t.Errorf("expected gate %q failed, got %v", GateConsequenceSensor, result.GatesFailed)
	}
}

func TestCheckConsequenceSensors_MultipleEnforcements(t *testing.T) {
	dir := t.TempDir()
	content := `# Design: Governance

## Recommendations

### Enforcement Mechanisms

| Mechanism | Type | Consequence Sensor |
|-----------|------|--------------------|
| Duplication spawn gate | gate | events.jsonl spawn.gate_decision — fire rate via orch stats |
| Accretion pre-commit hook | hook | none — open loop |
| Build verification gate | gate | events.jsonl verification.failed — tracked in completion pipeline |
`
	writeInvestigation(t, dir, content)

	result := CheckConsequenceSensors(dir, "architect")
	if !result.Passed {
		t.Errorf("should pass (open loops are warnings), got errors: %v", result.Errors)
	}
	if len(result.OpenLoops) != 1 {
		t.Errorf("should have 1 open loop, got %d: %v", len(result.OpenLoops), result.OpenLoops)
	}
}

func TestCheckConsequenceSensors_MissingSensorColumn(t *testing.T) {
	dir := t.TempDir()
	// Table exists but without Consequence Sensor column
	content := `# Design: Add Gate

## Recommendations

### Enforcement Mechanisms

| Mechanism | Type |
|-----------|------|
| Spawn gate | gate |
`
	writeInvestigation(t, dir, content)

	result := CheckConsequenceSensors(dir, "architect")
	if result.Passed {
		t.Error("should fail when Enforcement Mechanisms table is missing Consequence Sensor column")
	}
}

// writeInvestigation creates a .kb/investigations/ file in the workspace
func writeInvestigation(t *testing.T, dir string, content string) {
	t.Helper()
	kbDir := filepath.Join(dir, ".kb", "investigations")
	if err := os.MkdirAll(kbDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(kbDir, "2026-03-20-design-test.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
