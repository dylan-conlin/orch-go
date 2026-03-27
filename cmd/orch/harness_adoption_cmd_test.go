package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestHarnessAdoptionCmd_Flags(t *testing.T) {
	cmd := harnessAdoptionCmd
	if cmd.Use != "adoption" {
		t.Errorf("expected Use='adoption', got %q", cmd.Use)
	}
	f := cmd.Flags().Lookup("json")
	if f == nil {
		t.Error("expected flag --json to exist")
	}
}

func TestHarnessAdoptionCmd_IsRegistered(t *testing.T) {
	found := false
	for _, cmd := range harnessCmd.Commands() {
		if cmd.Use == "adoption" {
			found = true
			break
		}
	}
	if !found {
		t.Error("harness adoption command not registered as subcommand of harness")
	}
}

func TestMakeSignal_OK(t *testing.T) {
	sig := makeSignal("test", "surface", 100, 90, 80)
	if sig.Status != "ok" {
		t.Errorf("expected status=ok, got %q", sig.Status)
	}
	if sig.RatePct != 90 {
		t.Errorf("expected rate=90, got %.1f", sig.RatePct)
	}
}

func TestMakeSignal_Drift(t *testing.T) {
	sig := makeSignal("test", "surface", 100, 60, 80)
	if sig.Status != "drift" {
		t.Errorf("expected status=drift, got %q", sig.Status)
	}
}

func TestMakeSignal_Critical(t *testing.T) {
	sig := makeSignal("test", "surface", 100, 20, 80)
	if sig.Status != "critical" {
		t.Errorf("expected status=critical for 20%% rate with 80%% target, got %q", sig.Status)
	}
}

func TestMakeSignal_ZeroTotal(t *testing.T) {
	sig := makeSignal("test", "surface", 0, 0, 80)
	if sig.Status != "ok" {
		t.Errorf("expected status=ok when total=0, got %q", sig.Status)
	}
	if sig.RatePct != 0 {
		t.Errorf("expected rate=0 when total=0, got %.1f", sig.RatePct)
	}
}

func TestMakeSignal_BoundaryExactTarget(t *testing.T) {
	sig := makeSignal("test", "surface", 100, 80, 80)
	if sig.Status != "ok" {
		t.Errorf("expected status=ok when rate=target, got %q", sig.Status)
	}
}

func TestMakeSignal_BoundaryJustBelowTarget(t *testing.T) {
	sig := makeSignal("test", "surface", 100, 79, 80)
	if sig.Status != "drift" {
		t.Errorf("expected status=drift when 1 below target, got %q", sig.Status)
	}
}

func TestMakeSignal_BoundaryHalfTarget(t *testing.T) {
	// At exactly half of target, should be drift (not critical)
	sig := makeSignal("test", "surface", 100, 40, 80)
	if sig.Status != "drift" {
		t.Errorf("expected status=drift at exactly half target, got %q", sig.Status)
	}
}

func TestMeasureAdoption_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	result := measureAdoption(dir)
	if len(result.Signals) != 7 {
		t.Errorf("expected 7 signals, got %d", len(result.Signals))
	}
	for _, sig := range result.Signals {
		if sig.Total != 0 {
			t.Errorf("signal %q: expected total=0 in empty dir, got %d", sig.Name, sig.Total)
		}
		if sig.Status != "ok" {
			t.Errorf("signal %q: expected status=ok with 0 total, got %q", sig.Name, sig.Status)
		}
	}
}

func TestMeasureAdoption_WithTestCorpus(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")

	// Set up investigations: 3 total, 1 with model link
	invDir := filepath.Join(kbDir, "investigations")
	os.MkdirAll(invDir, 0755)
	writeTestFile(t, filepath.Join(invDir, "inv-1.md"), "# Inv 1\n**Model:** test-model\n")
	writeTestFile(t, filepath.Join(invDir, "inv-2.md"), "# Inv 2\nNo model link\n")
	writeTestFile(t, filepath.Join(invDir, "inv-3.md"), "# Inv 3\nJust text\n")

	// Set up briefs: 2 total, both with tension
	briefDir := filepath.Join(kbDir, "briefs")
	os.MkdirAll(briefDir, 0755)
	writeTestFile(t, filepath.Join(briefDir, "brief-1.md"), "# Brief 1\n## Frame\n## Resolution\n## Tension\nSomething\n")
	writeTestFile(t, filepath.Join(briefDir, "brief-2.md"), "# Brief 2\n## Frame\n## Resolution\n## Tension\nOther\n")

	// Set up probes: 3 total, 2 with claim, 1 with verdict
	modelDir := filepath.Join(kbDir, "models", "test-model", "probes")
	os.MkdirAll(modelDir, 0755)
	writeTestFile(t, filepath.Join(modelDir, "probe-1.md"), "# Probe 1\n**claim:** TM-01\n**verdict:** confirms\n")
	writeTestFile(t, filepath.Join(modelDir, "probe-2.md"), "# Probe 2\n**claim:** TM-02\n")
	writeTestFile(t, filepath.Join(modelDir, "probe-3.md"), "# Probe 3\nNo frontmatter fields\n")

	// Set up threads: 2 total, 1 with resolved_to
	threadsDir := filepath.Join(kbDir, "threads")
	os.MkdirAll(threadsDir, 0755)
	writeTestFile(t, filepath.Join(threadsDir, "thread-1.md"), "---\nresolved_to: model.md\n---\n")
	writeTestFile(t, filepath.Join(threadsDir, "thread-2.md"), "---\nresolved_to:\n---\n")

	// Set up decisions: 2 total, 1 with Extends
	decisionsDir := filepath.Join(kbDir, "decisions")
	os.MkdirAll(decisionsDir, 0755)
	writeTestFile(t, filepath.Join(decisionsDir, "dec-1.md"), "# Dec\n**Extends:** other-decision\n")
	writeTestFile(t, filepath.Join(decisionsDir, "dec-2.md"), "# Dec\nNo extends\n")

	// Set up beads: 3 issues, 1 enriched
	beadsDir := filepath.Join(dir, ".beads")
	os.MkdirAll(beadsDir, 0755)
	issues := []string{
		`{"id":"test-1","labels":["area:cli","effort:small"]}`,
		`{"id":"test-2","labels":[]}`,
		`{"id":"test-3","labels":["status:parked"]}`,
	}
	writeTestFile(t, filepath.Join(beadsDir, "issues.jsonl"),
		issues[0]+"\n"+issues[1]+"\n"+issues[2]+"\n")

	result := measureAdoption(dir)

	expected := map[string]struct{ total, adopted int }{
		"Investigation model link": {3, 1},
		"Brief tension":            {2, 2},
		"Probe claim":              {3, 2},
		"Probe verdict":            {3, 1},
		"Thread resolved_to":       {2, 1},
		"Decision Extends":         {2, 1},
		"Beads enrichment":         {3, 1},
	}

	for _, sig := range result.Signals {
		exp, ok := expected[sig.Name]
		if !ok {
			t.Errorf("unexpected signal %q", sig.Name)
			continue
		}
		if sig.Total != exp.total {
			t.Errorf("%s: expected total=%d, got %d", sig.Name, exp.total, sig.Total)
		}
		if sig.Adopted != exp.adopted {
			t.Errorf("%s: expected adopted=%d, got %d", sig.Name, exp.adopted, sig.Adopted)
		}
	}
}

func TestMeasureAdoption_AlertGeneration(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")

	// Create 10 investigations, 1 with model link → 10% rate (critical at <40%)
	invDir := filepath.Join(kbDir, "investigations")
	os.MkdirAll(invDir, 0755)
	writeTestFile(t, filepath.Join(invDir, "linked.md"), "# Inv\n**Model:** test\n")
	for i := 0; i < 9; i++ {
		writeTestFile(t, filepath.Join(invDir, "unlinked-"+string(rune('a'+i))+".md"), "# Inv\n")
	}

	result := measureAdoption(dir)

	// Should have at least one critical alert for investigations
	foundCritical := false
	for _, a := range result.Alerts {
		if a.Signal == "Investigation model link" && a.Level == "critical" {
			foundCritical = true
		}
	}
	if !foundCritical {
		t.Error("expected critical alert for investigation model link at 10% rate")
	}
}

func TestFormatAdoptionText(t *testing.T) {
	result := &AdoptionResult{
		Signals: []AdoptionSignal{
			{Name: "Brief tension", Total: 10, Adopted: 10, RatePct: 100, TargetPct: 100, Status: "ok"},
			{Name: "Investigation model link", Total: 100, Adopted: 15, RatePct: 15, TargetPct: 80, Status: "critical"},
		},
		Alerts: []AdoptionAlert{
			{Signal: "Investigation model link", Level: "critical", Message: "15% adoption (target 80%)"},
		},
	}
	output := formatAdoptionText(result)
	for _, expected := range []string{"ADOPTION RATES", "Brief tension", "ok", "CRITICAL", "ALERTS"} {
		if !contains(output, expected) {
			t.Errorf("expected output to contain %q", expected)
		}
	}
}

func TestFormatAdoptionJSON(t *testing.T) {
	result := &AdoptionResult{
		GeneratedAt: "2026-03-27T12:00:00Z",
		Signals: []AdoptionSignal{
			{Name: "test", Total: 10, Adopted: 5, RatePct: 50, TargetPct: 80, Status: "drift"},
		},
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	var parsed AdoptionResult
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(parsed.Signals) != 1 {
		t.Errorf("expected 1 signal in JSON, got %d", len(parsed.Signals))
	}
}

func TestFileHasFrontmatterField(t *testing.T) {
	dir := t.TempDir()

	// Bold format: **claim:**
	f1 := filepath.Join(dir, "bold.md")
	writeTestFile(t, f1, "# Title\n**claim:** TM-01\n**verdict:** confirms\n")
	if !fileHasFrontmatterField(f1, "claim:") {
		t.Error("should find **claim:** in bold format")
	}

	// Plain format: claim:
	f2 := filepath.Join(dir, "plain.md")
	writeTestFile(t, f2, "# Title\nclaim: TM-01\nverdict: extends\n")
	if !fileHasFrontmatterField(f2, "claim:") {
		t.Error("should find claim: in plain format")
	}

	// No field
	f3 := filepath.Join(dir, "none.md")
	writeTestFile(t, f3, "# Title\nJust content\n")
	if fileHasFrontmatterField(f3, "claim:") {
		t.Error("should not find claim: when absent")
	}

	// Field only in body (line 20) — should not match
	f4 := filepath.Join(dir, "body.md")
	lines := "# Title\n"
	for i := 0; i < 20; i++ {
		lines += "line\n"
	}
	lines += "claim: body-only\n"
	writeTestFile(t, f4, lines)
	if fileHasFrontmatterField(f4, "claim:") {
		t.Error("should not match claim: after line 15")
	}
}

func TestFileHasNonEmptyField(t *testing.T) {
	dir := t.TempDir()

	f1 := filepath.Join(dir, "full.md")
	writeTestFile(t, f1, "resolved_to: model.md\n")
	if !fileHasNonEmptyField(f1, "resolved_to:") {
		t.Error("should find non-empty resolved_to")
	}

	f2 := filepath.Join(dir, "empty.md")
	writeTestFile(t, f2, "resolved_to:\n")
	if fileHasNonEmptyField(f2, "resolved_to:") {
		t.Error("should not match empty resolved_to")
	}

	f3 := filepath.Join(dir, "quotes.md")
	writeTestFile(t, f3, `resolved_to: ""`+"\n")
	if fileHasNonEmptyField(f3, "resolved_to:") {
		t.Error("should not match quoted-empty resolved_to")
	}
}

func TestListAllProbeFiles(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")

	// Two models, each with probes
	m1 := filepath.Join(kbDir, "models", "model-a", "probes")
	m2 := filepath.Join(kbDir, "models", "model-b", "probes")
	os.MkdirAll(m1, 0755)
	os.MkdirAll(m2, 0755)

	writeTestFile(t, filepath.Join(m1, "probe-1.md"), "content")
	writeTestFile(t, filepath.Join(m1, "probe-2.md"), "content")
	writeTestFile(t, filepath.Join(m2, "probe-3.md"), "content")

	// Non-md file should be excluded
	writeTestFile(t, filepath.Join(m1, "notes.txt"), "content")

	files := listAllProbeFiles(kbDir)
	if len(files) != 3 {
		t.Errorf("expected 3 probe files, got %d", len(files))
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}
