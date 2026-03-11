package verify

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckPlanHydration_NotArchitect(t *testing.T) {
	result := CheckPlanHydration("feature-impl", "/tmp/nonexistent", "/tmp/nonexistent")
	if result != nil {
		t.Errorf("expected nil for non-architect skill, got %+v", result)
	}
}

func TestCheckPlanHydration_NoPlanFiles(t *testing.T) {
	projectDir := t.TempDir()
	// Create .kb/plans/ but no plan files
	os.MkdirAll(filepath.Join(projectDir, ".kb", "plans"), 0o755)

	result := CheckPlanHydration("architect", t.TempDir(), projectDir)
	if result != nil {
		t.Errorf("expected nil when no plan files, got %+v", result)
	}
}

func TestCheckPlanHydration_SinglePhasePlan(t *testing.T) {
	projectDir := t.TempDir()
	plansDir := filepath.Join(projectDir, ".kb", "plans")
	os.MkdirAll(plansDir, 0o755)

	// Plan with only one phase — no warning needed
	plan := `## Summary
Some plan.

## Phases

### Phase 1: Do the thing
Do it.
`
	os.WriteFile(filepath.Join(plansDir, "2026-03-11-test-plan.md"), []byte(plan), 0o644)

	result := CheckPlanHydration("architect", t.TempDir(), projectDir)
	if result != nil {
		t.Errorf("expected nil for single-phase plan, got %+v", result)
	}
}

func TestCheckPlanHydration_MultiPhasePlanWarns(t *testing.T) {
	projectDir := t.TempDir()
	plansDir := filepath.Join(projectDir, ".kb", "plans")
	os.MkdirAll(plansDir, 0o755)

	plan := `## Summary
Multi-phase design.

## Phases

### Phase 1: Census
Enumerate gates.

### Phase 2: Fix noise gates
Fix false positives.

### Phase 3: Audit
Sample and classify.
`
	os.WriteFile(filepath.Join(plansDir, "2026-03-11-gate-plan.md"), []byte(plan), 0o644)

	result := CheckPlanHydration("architect", t.TempDir(), projectDir)
	if result == nil {
		t.Fatal("expected warning for multi-phase plan, got nil")
	}
	if len(result.Warnings) == 0 {
		t.Error("expected at least one warning")
	}
	// Should mention the plan file
	found := false
	for _, w := range result.Warnings {
		if contains(w, "gate-plan") && contains(w, "3 phases") {
			found = true
		}
	}
	if !found {
		t.Errorf("warning should mention plan file and phase count, got: %v", result.Warnings)
	}
}

func TestCheckPlanHydration_ScopedToRecentPlans(t *testing.T) {
	projectDir := t.TempDir()
	plansDir := filepath.Join(projectDir, ".kb", "plans")
	os.MkdirAll(plansDir, 0o755)

	// Workspace with a git baseline — simulates scoping
	workspacePath := t.TempDir()

	plan := `## Phases

### Phase 1: A
### Phase 2: B
### Phase 3: C
`
	os.WriteFile(filepath.Join(plansDir, "2026-03-11-scoped.md"), []byte(plan), 0o644)

	result := CheckPlanHydration("architect", workspacePath, projectDir)
	if result == nil {
		t.Fatal("expected result for multi-phase plan")
	}
	if len(result.Warnings) == 0 {
		t.Error("expected warning")
	}
}

func TestCountPlanPhases(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    int
	}{
		{
			name:    "no phases section",
			content: "## Summary\nJust a doc.",
			want:    0,
		},
		{
			name:    "one phase",
			content: "## Phases\n\n### Phase 1: Do it\nStuff.",
			want:    1,
		},
		{
			name: "four phases",
			content: `## Phases

### Phase 1: Census
Enumerate.

### Phase 2: Fix
Fix it.

### Phase 3: Audit
Sample.

### Phase 4: Measure
Track.
`,
			want: 4,
		},
		{
			name: "phases without numbers",
			content: `## Phases

### Investigation
Look.

### Implementation
Build.

### Validation
Test.
`,
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CountPlanPhases(tt.content)
			if got != tt.want {
				t.Errorf("CountPlanPhases() = %d, want %d", got, tt.want)
			}
		})
	}
}

