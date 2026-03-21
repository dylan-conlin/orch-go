package decisions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestListActiveDecisions(t *testing.T) {
	dir := setupTestProject(t, map[string]string{
		"2026-03-01-foo.md": makeDecision("Foo", "Accepted", "gate"),
		"2026-03-02-bar.md": makeDecision("Bar", "Accepted", "context-only"),
	})

	decisions, err := ListActiveDecisions(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(decisions) != 2 {
		t.Fatalf("expected 2 decisions, got %d", len(decisions))
	}
	// Sorted by date descending
	if decisions[0].Name != "2026-03-02-bar" {
		t.Errorf("expected bar first (newest), got %s", decisions[0].Name)
	}
}

func TestListActiveDecisions_SkipsDirectories(t *testing.T) {
	dir := setupTestProject(t, map[string]string{
		"2026-03-01-foo.md": makeDecision("Foo", "Accepted", "gate"),
	})
	// Create archived/ subdirectory
	os.MkdirAll(filepath.Join(dir, ".kb", "decisions", "archived"), 0o755)
	os.WriteFile(filepath.Join(dir, ".kb", "decisions", "archived", "old.md"),
		[]byte(makeDecision("Old", "Superseded", "context-only")), 0o644)

	decisions, err := ListActiveDecisions(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(decisions) != 1 {
		t.Fatalf("expected 1 decision (skipping archived dir), got %d", len(decisions))
	}
}

func TestParseDecisionFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "2026-03-15-test-decision.md")
	os.WriteFile(path, []byte(makeDecision("Test Decision", "Accepted", "hook")), 0o644)

	d := parseDecisionFile(path)
	if d.Title != "Test Decision" {
		t.Errorf("expected title 'Test Decision', got %q", d.Title)
	}
	if d.Status != "Accepted" {
		t.Errorf("expected status 'Accepted', got %q", d.Status)
	}
	if d.Enforcement != EnforcementHook {
		t.Errorf("expected enforcement 'hook', got %q", d.Enforcement)
	}
	if d.Date.Format("2006-01-02") != "2026-03-15" {
		t.Errorf("expected date 2026-03-15, got %s", d.Date.Format("2006-01-02"))
	}
}

func TestParseDecisionFile_NoEnforcement(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "2026-03-15-test.md")
	os.WriteFile(path, []byte(`# Decision: Test

**Date:** 2026-03-15
**Status:** Accepted

## Context
No enforcement field.
`), 0o644)

	d := parseDecisionFile(path)
	if d.Enforcement != EnforcementUnknown {
		t.Errorf("expected unknown enforcement, got %q", d.Enforcement)
	}
}

func TestCountCitations(t *testing.T) {
	dir := setupTestProject(t, map[string]string{
		"2026-03-01-foo.md": makeDecision("Foo", "Accepted", "gate"),
	})

	// Create a file that cites the decision
	guideDir := filepath.Join(dir, ".kb", "guides")
	os.MkdirAll(guideDir, 0o755)
	os.WriteFile(filepath.Join(guideDir, "test-guide.md"),
		[]byte("See decision: 2026-03-01-foo for context."), 0o644)

	count := CountCitations("2026-03-01-foo", dir)
	if count != 1 {
		t.Errorf("expected 1 citation, got %d", count)
	}
}

func TestCountCitations_NoCitations(t *testing.T) {
	dir := setupTestProject(t, map[string]string{
		"2026-03-01-foo.md": makeDecision("Foo", "Accepted", "gate"),
	})

	count := CountCitations("2026-03-01-foo", dir)
	if count != 0 {
		t.Errorf("expected 0 citations, got %d", count)
	}
}

func TestFindStale(t *testing.T) {
	// Use a date >30 days ago
	oldDate := time.Now().AddDate(0, 0, -45).Format("2006-01-02")
	recentDate := time.Now().AddDate(0, 0, -5).Format("2006-01-02")

	dir := setupTestProject(t, map[string]string{
		oldDate + "-stale-context-only.md":    makeDecision("Stale", "Accepted", "context-only"),
		oldDate + "-old-gate.md":              makeDecision("Old Gate", "Accepted", "gate"),
		recentDate + "-recent-context-only.md": makeDecision("Recent", "Accepted", "context-only"),
	})

	result, err := FindStale(dir)
	if err != nil {
		t.Fatal(err)
	}

	if result.Active != 3 {
		t.Errorf("expected 3 active, got %d", result.Active)
	}

	// Only the old context-only with 0 citations should be stale
	if len(result.Stale) != 1 {
		t.Fatalf("expected 1 stale, got %d", len(result.Stale))
	}
	if result.Stale[0].Enforcement != EnforcementContextOnly {
		t.Errorf("expected stale decision to be context-only, got %q", result.Stale[0].Enforcement)
	}
}

func TestCheckBudget(t *testing.T) {
	dir := setupTestProject(t, map[string]string{
		"2026-03-01-a.md": makeDecision("A", "Accepted", "gate"),
		"2026-03-02-b.md": makeDecision("B", "Accepted", "hook"),
		"2026-03-03-c.md": makeDecision("C", "Accepted", "context-only"),
		"2026-03-04-d.md": makeDecision("D", "Accepted", ""),
	})

	status, err := CheckBudget(dir)
	if err != nil {
		t.Fatal(err)
	}

	if status.Active != 4 {
		t.Errorf("expected 4 active, got %d", status.Active)
	}
	if status.ByType[EnforcementGate] != 1 {
		t.Errorf("expected 1 gate, got %d", status.ByType[EnforcementGate])
	}
	if status.ByType[EnforcementHook] != 1 {
		t.Errorf("expected 1 hook, got %d", status.ByType[EnforcementHook])
	}
	if status.ByType[EnforcementContextOnly] != 1 {
		t.Errorf("expected 1 context-only, got %d", status.ByType[EnforcementContextOnly])
	}
	if status.Unclassified != 1 {
		t.Errorf("expected 1 unclassified, got %d", status.Unclassified)
	}
}

func TestCheckBudget_OverBudget(t *testing.T) {
	files := make(map[string]string)
	for i := 0; i < 35; i++ {
		name := filepath.Join(time.Now().AddDate(0, 0, -i).Format("2006-01-02") + "-decision-" + string(rune('a'+i%26)) + ".md")
		files[name] = makeDecision("Decision", "Accepted", "context-only")
	}

	dir := setupTestProject(t, files)
	status, err := CheckBudget(dir)
	if err != nil {
		t.Fatal(err)
	}

	if status.OverBy != 5 {
		t.Errorf("expected 5 over budget, got %d", status.OverBy)
	}
}

// --- helpers ---

func setupTestProject(t *testing.T, decisions map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	decDir := filepath.Join(dir, ".kb", "decisions")
	os.MkdirAll(decDir, 0o755)

	for name, content := range decisions {
		os.WriteFile(filepath.Join(decDir, name), []byte(content), 0o644)
	}
	return dir
}

func makeDecision(title, status, enforcement string) string {
	var sb []string
	sb = append(sb, "# Decision: "+title)
	sb = append(sb, "")
	sb = append(sb, "**Date:** 2026-03-15")
	sb = append(sb, "**Status:** "+status)
	if enforcement != "" {
		sb = append(sb, "**Enforcement:** "+enforcement)
	}
	sb = append(sb, "")
	sb = append(sb, "## Context")
	sb = append(sb, "Test context.")
	return strings.Join(sb, "\n")
}
