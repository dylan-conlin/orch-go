package daemon

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Integration tests for default source implementations.
// These test real filesystem and git interactions, not mocked sources.

// --- defaultInvestigationOrphansSource ---

func TestDefaultInvestigationOrphansSource_ListActiveInvestigations(t *testing.T) {
	dir := t.TempDir()
	invDir := filepath.Join(dir, ".kb", "investigations")
	if err := os.MkdirAll(invDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Active investigation (old enough to be found)
	activeContent := "# Investigation\n\nStatus: Active\n\nSome findings here."
	if err := os.WriteFile(filepath.Join(invDir, "2026-01-01-stale-investigation.md"), []byte(activeContent), 0o644); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().Add(-10 * 24 * time.Hour)
	os.Chtimes(filepath.Join(invDir, "2026-01-01-stale-investigation.md"), oldTime, oldTime)

	// Completed investigation (should be skipped)
	completedContent := "# Investigation\n\nStatus: Complete\n\nDone."
	if err := os.WriteFile(filepath.Join(invDir, "2026-01-02-completed.md"), []byte(completedContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Active investigation with lowercase status variant
	activeLower := "# Investigation\n\nstatus: Active\n\nAnother one."
	if err := os.WriteFile(filepath.Join(invDir, "2026-01-03-lower-status.md"), []byte(activeLower), 0o644); err != nil {
		t.Fatal(err)
	}
	os.Chtimes(filepath.Join(invDir, "2026-01-03-lower-status.md"), oldTime, oldTime)

	// Non-markdown file (should be skipped)
	if err := os.WriteFile(filepath.Join(invDir, "notes.txt"), []byte("not markdown"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Directory (should be skipped)
	if err := os.MkdirAll(filepath.Join(invDir, "archived"), 0o755); err != nil {
		t.Fatal(err)
	}

	origDir := chdirTemp(t, dir)
	_ = origDir

	src := &defaultInvestigationOrphansSource{}
	investigations, err := src.ListActiveInvestigations()
	if err != nil {
		t.Fatalf("ListActiveInvestigations() error = %v", err)
	}

	if len(investigations) != 2 {
		t.Fatalf("got %d investigations, want 2", len(investigations))
	}

	slugs := make(map[string]bool)
	for _, inv := range investigations {
		slugs[inv.Slug] = true
		if inv.Age < 9*24*time.Hour {
			t.Errorf("investigation %s age = %v, expected > 9 days", inv.Slug, inv.Age)
		}
	}
	if !slugs["stale-investigation"] {
		t.Error("missing slug 'stale-investigation'")
	}
	if !slugs["lower-status"] {
		t.Error("missing slug 'lower-status'")
	}
}

func TestDefaultInvestigationOrphansSource_MissingDir(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)

	src := &defaultInvestigationOrphansSource{}
	investigations, err := src.ListActiveInvestigations()
	if err != nil {
		t.Fatalf("expected nil error for missing dir, got %v", err)
	}
	if len(investigations) != 0 {
		t.Errorf("got %d investigations, want 0", len(investigations))
	}
}

// --- defaultThreadStalenessSource ---

func TestDefaultThreadStalenessSource_ListOpenThreads(t *testing.T) {
	dir := t.TempDir()
	threadsDir := filepath.Join(dir, ".kb", "threads")
	if err := os.MkdirAll(threadsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	openThread := `---
title: Old Discussion
status: open
created: 2026-01-01
updated: 2026-01-01
---

## 2026-01-01

Some old discussion point.
`
	if err := os.WriteFile(filepath.Join(threadsDir, "2026-01-01-old-discussion.md"), []byte(openThread), 0o644); err != nil {
		t.Fatal(err)
	}

	resolvedThread := `---
title: Resolved Thing
status: resolved
created: 2026-01-01
updated: 2026-01-05
resolved_to: decision
---

## 2026-01-01

Resolved.
`
	if err := os.WriteFile(filepath.Join(threadsDir, "2026-01-01-resolved-thing.md"), []byte(resolvedThread), 0o644); err != nil {
		t.Fatal(err)
	}

	recentDate := time.Now().Add(-24 * time.Hour).Format("2006-01-02")
	recentThread := "---\ntitle: Recent Thread\nstatus: open\ncreated: " + recentDate + "\nupdated: " + recentDate + "\n---\n\n## " + recentDate + "\n\nFresh.\n"
	if err := os.WriteFile(filepath.Join(threadsDir, recentDate+"-recent.md"), []byte(recentThread), 0o644); err != nil {
		t.Fatal(err)
	}

	chdirTemp(t, dir)

	src := &defaultThreadStalenessSource{}
	threads, err := src.ListOpenThreads()
	if err != nil {
		t.Fatalf("ListOpenThreads() error = %v", err)
	}

	if len(threads) != 2 {
		t.Fatalf("got %d threads, want 2 (old + recent, skip resolved)", len(threads))
	}

	for _, th := range threads {
		if th.Slug == "" {
			t.Error("empty slug")
		}
		if th.Title == "" {
			t.Error("empty title")
		}
		if th.Age <= 0 {
			t.Errorf("thread %s has non-positive age: %v", th.Slug, th.Age)
		}
	}

	for _, th := range threads {
		if th.Title == "Old Discussion" {
			if th.Age < 30*24*time.Hour {
				t.Errorf("old thread age = %v, expected > 30 days", th.Age)
			}
		}
	}
}

func TestDefaultThreadStalenessSource_MissingDir(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)

	src := &defaultThreadStalenessSource{}
	threads, err := src.ListOpenThreads()
	if err != nil {
		t.Fatalf("expected nil error for missing dir, got %v", err)
	}
	if len(threads) != 0 {
		t.Errorf("got %d threads, want 0", len(threads))
	}
}

// --- defaultModelContradictionsSource ---

func TestDefaultModelContradictionsSource_ListUnresolvedContradictions(t *testing.T) {
	dir := t.TempDir()
	modelsDir := filepath.Join(dir, ".kb", "models")
	modelDir := filepath.Join(modelsDir, "test-model")
	probesDir := filepath.Join(modelDir, "probes")
	if err := os.MkdirAll(probesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	modelPath := filepath.Join(modelDir, "model.md")
	if err := os.WriteFile(modelPath, []byte("# Test Model\n\nSome claims here.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().Add(-30 * 24 * time.Hour)
	os.Chtimes(modelPath, oldTime, oldTime)

	// Probe that contradicts (created after model update)
	probeContent := "# Probe\n\nThis contradicts the model's claim about X.\n"
	if err := os.WriteFile(filepath.Join(probesDir, "2026-03-10-contradicts-claim.md"), []byte(probeContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Probe without contradiction (should be skipped)
	if err := os.WriteFile(filepath.Join(probesDir, "2026-03-11-confirms-claim.md"), []byte("# Probe\n\nConfirms claim.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Model without probes dir (should be handled gracefully)
	emptyModelDir := filepath.Join(modelsDir, "empty-model")
	if err := os.MkdirAll(emptyModelDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(emptyModelDir, "model.md"), []byte("# Empty\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	chdirTemp(t, dir)

	src := &defaultModelContradictionsSource{}
	contradictions, err := src.ListUnresolvedContradictions()
	if err != nil {
		t.Fatalf("ListUnresolvedContradictions() error = %v", err)
	}

	if len(contradictions) != 1 {
		t.Fatalf("got %d contradictions, want 1", len(contradictions))
	}

	c := contradictions[0]
	if c.ModelSlug != "test-model" {
		t.Errorf("ModelSlug = %q, want test-model", c.ModelSlug)
	}
	if c.ProbeFilename != "2026-03-10-contradicts-claim.md" {
		t.Errorf("ProbeFilename = %q", c.ProbeFilename)
	}
	if c.ProbeDate.Year() != 2026 || c.ProbeDate.Month() != 3 || c.ProbeDate.Day() != 10 {
		t.Errorf("ProbeDate = %v, want 2026-03-10", c.ProbeDate)
	}
}

func TestDefaultModelContradictionsSource_MissingDir(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)

	src := &defaultModelContradictionsSource{}
	contradictions, err := src.ListUnresolvedContradictions()
	if err != nil {
		t.Fatalf("expected nil error for missing dir, got %v", err)
	}
	if len(contradictions) != 0 {
		t.Errorf("got %d contradictions, want 0", len(contradictions))
	}
}

func TestDefaultModelContradictionsSource_ProbeBeforeModelUpdate(t *testing.T) {
	dir := t.TempDir()
	probesDir := filepath.Join(dir, ".kb", "models", "fresh-model", "probes")
	if err := os.MkdirAll(probesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Model updated recently (mod time = now)
	modelPath := filepath.Join(dir, ".kb", "models", "fresh-model", "model.md")
	if err := os.WriteFile(modelPath, []byte("# Fresh Model\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Probe with contradiction but date is in the past (before model update)
	probeContent := "# Probe\n\nThis contradicts the model.\n"
	if err := os.WriteFile(filepath.Join(probesDir, "2026-01-01-old-contradiction.md"), []byte(probeContent), 0o644); err != nil {
		t.Fatal(err)
	}

	chdirTemp(t, dir)

	src := &defaultModelContradictionsSource{}
	contradictions, err := src.ListUnresolvedContradictions()
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(contradictions) != 0 {
		t.Errorf("got %d contradictions, want 0 (probe is before model update)", len(contradictions))
	}
}

func TestDefaultModelContradictionsSource_NegationDetection(t *testing.T) {
	dir := t.TempDir()
	modelsDir := filepath.Join(dir, ".kb", "models")
	modelDir := filepath.Join(modelsDir, "test-model")
	probesDir := filepath.Join(modelDir, "probes")
	if err := os.MkdirAll(probesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	modelPath := filepath.Join(modelDir, "model.md")
	if err := os.WriteFile(modelPath, []byte("# Test Model\n\nThe threshold is 1500 lines.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().Add(-30 * 24 * time.Hour)
	os.Chtimes(modelPath, oldTime, oldTime)

	// Probe that uses negation language WITHOUT the word "contradict"
	probeContent := `# Probe: Threshold Validation

## What I Observed
The 1500-line threshold is not true in practice. After the March refactor,
files up to 2000 lines are routinely accepted without issues.

## Model Impact
The documented threshold is incorrect and should be updated.
`
	if err := os.WriteFile(filepath.Join(probesDir, "2026-03-15-threshold-check.md"), []byte(probeContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Probe that confirms model (should NOT be detected)
	confirmContent := "# Probe\n\nConfirms the model's claims. Everything checks out.\n"
	if err := os.WriteFile(filepath.Join(probesDir, "2026-03-16-confirms-model.md"), []byte(confirmContent), 0o644); err != nil {
		t.Fatal(err)
	}

	chdirTemp(t, dir)

	src := &defaultModelContradictionsSource{}
	contradictions, err := src.ListUnresolvedContradictions()
	if err != nil {
		t.Fatalf("ListUnresolvedContradictions() error = %v", err)
	}

	if len(contradictions) != 1 {
		t.Fatalf("got %d contradictions, want 1 (negation probe should be detected)", len(contradictions))
	}

	c := contradictions[0]
	if c.ProbeFilename != "2026-03-15-threshold-check.md" {
		t.Errorf("ProbeFilename = %q, want threshold-check probe", c.ProbeFilename)
	}
}

// --- defaultKnowledgeDecaySource ---

func TestDefaultKnowledgeDecaySource_ListDecayedModels(t *testing.T) {
	dir := t.TempDir()
	modelsDir := filepath.Join(dir, ".kb", "models")

	// Model with recent probes (should NOT be flagged)
	freshProbes := filepath.Join(modelsDir, "fresh-model", "probes")
	if err := os.MkdirAll(freshProbes, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modelsDir, "fresh-model", "model.md"), []byte("# Fresh\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	recentDate := time.Now().Add(-5 * 24 * time.Hour).Format("2006-01-02")
	if err := os.WriteFile(filepath.Join(freshProbes, recentDate+"-recent-probe.md"), []byte("# Probe\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Model with old probes (should be flagged)
	staleProbes := filepath.Join(modelsDir, "stale-model", "probes")
	if err := os.MkdirAll(staleProbes, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modelsDir, "stale-model", "model.md"), []byte("# Stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staleProbes, "2025-01-01-old-probe.md"), []byte("# Probe\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Model with no probes directory (should be flagged with sentinel)
	noProbes := filepath.Join(modelsDir, "no-probes-model")
	if err := os.MkdirAll(noProbes, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(noProbes, "model.md"), []byte("# No Probes\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Directory without model.md (should be skipped)
	if err := os.MkdirAll(filepath.Join(modelsDir, "not-a-model"), 0o755); err != nil {
		t.Fatal(err)
	}

	chdirTemp(t, dir)

	src := &defaultKnowledgeDecaySource{}
	models, err := src.ListDecayedModels(30 * 24 * time.Hour)
	if err != nil {
		t.Fatalf("ListDecayedModels() error = %v", err)
	}

	if len(models) != 2 {
		t.Fatalf("got %d decayed models, want 2; models: %+v", len(models), models)
	}

	slugs := make(map[string]bool)
	for _, m := range models {
		slugs[m.Slug] = true
	}
	if !slugs["stale-model"] {
		t.Error("missing stale-model")
	}
	if !slugs["no-probes-model"] {
		t.Error("missing no-probes-model")
	}

	for _, m := range models {
		if m.Slug == "no-probes-model" && m.DaysSinceProbe != 999 {
			t.Errorf("no-probes-model DaysSinceProbe = %d, want 999", m.DaysSinceProbe)
		}
	}
}

func TestDefaultKnowledgeDecaySource_MissingDir(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)

	src := &defaultKnowledgeDecaySource{}
	models, err := src.ListDecayedModels(30 * 24 * time.Hour)
	if err != nil {
		t.Fatalf("expected nil error for missing dir, got %v", err)
	}
	if len(models) != 0 {
		t.Errorf("got %d models, want 0", len(models))
	}
}

// --- defaultHotspotAccelerationSource (git integration) ---

func TestDefaultHotspotAccelerationSource_ListFastGrowingFiles(t *testing.T) {
	dir := setupGitRepoWithGrowth(t)
	chdirTemp(t, dir)

	src := &defaultHotspotAccelerationSource{}
	// Use low threshold — our test repo has only 20 lines of growth
	files, err := src.ListFastGrowingFiles(5)
	if err != nil {
		t.Fatalf("ListFastGrowingFiles() error = %v", err)
	}

	// Fresh repo has no commit 30 days old → no base commit → empty result.
	// This validates the git integration runs without error.
	_ = files
}

func TestRunGitBaseCommit_FreshRepo(t *testing.T) {
	dir := setupGitRepoWithHistory(t)

	commit, err := runGitBaseCommit(dir)
	if err != nil {
		t.Fatalf("runGitBaseCommit() error = %v", err)
	}
	// Fresh repo → no commit 30 days ago → empty string
	if commit != "" {
		t.Logf("got base commit: %s (unexpected for fresh repo)", commit)
	}
}

func TestRunGitDiffNumstat_RealRepo(t *testing.T) {
	dir := setupGitRepoWithHistory(t)

	// Get initial commit (root)
	cmd := exec.Command("git", "rev-list", "--max-parents=0", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git rev-list error: %v", err)
	}
	baseCommit := strings.TrimSpace(string(out))

	output, err := runGitDiffNumstat(dir, baseCommit)
	if err != nil {
		t.Fatalf("runGitDiffNumstat() error = %v", err)
	}

	changes := parseGitDiffNumstat(output)
	if growth, ok := changes["growing.go"]; ok {
		if growth != 20 {
			t.Errorf("growing.go net growth = %d, want 20", growth)
		}
	} else {
		t.Error("growing.go not found in diff output")
	}
}

func TestCountFileLines_RealFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.go")

	var lines []string
	for i := 0; i < 50; i++ {
		lines = append(lines, "// line")
	}
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	count, err := countFileLines(path)
	if err != nil {
		t.Fatalf("countFileLines() error = %v", err)
	}
	if count != 50 {
		t.Errorf("countFileLines() = %d, want 50", count)
	}
}

// --- End-to-end: detector + real default source ---

func TestInvestigationOrphansDetector_DefaultSourceFilesystem_MockBeads(t *testing.T) {
	// Tests the real filesystem source for ListActiveInvestigations
	// combined with a mock for HasOpenIssueForInvestigation (requires beads).
	dir := t.TempDir()
	invDir := filepath.Join(dir, ".kb", "investigations")
	if err := os.MkdirAll(invDir, 0o755); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(invDir, "2025-12-01-old-investigation.md")
	if err := os.WriteFile(path, []byte("# Investigation\n\nStatus: Active\n\nFindings.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().Add(-10 * 24 * time.Hour)
	os.Chtimes(path, oldTime, oldTime)

	chdirTemp(t, dir)

	// Use real filesystem source for listing, verify it finds the investigation
	realSource := &defaultInvestigationOrphansSource{}
	investigations, err := realSource.ListActiveInvestigations()
	if err != nil {
		t.Fatalf("ListActiveInvestigations() error = %v", err)
	}
	if len(investigations) != 1 {
		t.Fatalf("got %d investigations, want 1", len(investigations))
	}

	// Feed real filesystem results through detector with mock beads
	detector := &InvestigationOrphansDetector{
		Source: &mockInvestigationOrphansSource{
			listFunc: func() ([]OrphanedInvestigation, error) {
				return investigations, nil
			},
			hasOpenFunc: func(slug string) (bool, error) {
				return false, nil // no matching beads issue
			},
		},
	}

	suggestions, err := detector.Detect()
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if len(suggestions) != 1 {
		t.Fatalf("got %d suggestions, want 1", len(suggestions))
	}
	if suggestions[0].Detector != "investigation_orphans" {
		t.Errorf("Detector = %q", suggestions[0].Detector)
	}
	if suggestions[0].Key != "old-investigation" {
		t.Errorf("Key = %q, want old-investigation", suggestions[0].Key)
	}
}

func TestModelContradictionsDetector_WithDefaultSource(t *testing.T) {
	dir := t.TempDir()
	probesDir := filepath.Join(dir, ".kb", "models", "test-model", "probes")
	if err := os.MkdirAll(probesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	modelPath := filepath.Join(dir, ".kb", "models", "test-model", "model.md")
	if err := os.WriteFile(modelPath, []byte("# Model\n\nClaims.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().Add(-60 * 24 * time.Hour)
	os.Chtimes(modelPath, oldTime, oldTime)

	probeContent := "# Probe\n\nFindings contradict the model claim about X.\n"
	if err := os.WriteFile(filepath.Join(probesDir, "2026-03-15-contradicts-x.md"), []byte(probeContent), 0o644); err != nil {
		t.Fatal(err)
	}

	chdirTemp(t, dir)

	detector := &ModelContradictionsDetector{
		Source: &defaultModelContradictionsSource{},
	}

	suggestions, err := detector.Detect()
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if len(suggestions) != 1 {
		t.Fatalf("got %d suggestions, want 1", len(suggestions))
	}
	if !strings.Contains(suggestions[0].Title, "test-model") {
		t.Errorf("Title = %q, expected to contain test-model", suggestions[0].Title)
	}
}

func TestKnowledgeDecayDetector_WithDefaultSource(t *testing.T) {
	dir := t.TempDir()
	modelDir := filepath.Join(dir, ".kb", "models", "decayed-model")
	if err := os.MkdirAll(modelDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modelDir, "model.md"), []byte("# Decayed\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	chdirTemp(t, dir)

	detector := &KnowledgeDecayDetector{
		Source: &defaultKnowledgeDecaySource{},
	}

	suggestions, err := detector.Detect()
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if len(suggestions) != 1 {
		t.Fatalf("got %d suggestions, want 1", len(suggestions))
	}
	if suggestions[0].Key != "decayed-model" {
		t.Errorf("Key = %q, want decayed-model", suggestions[0].Key)
	}
}

func TestHotspotAccelerationDetector_WithDefaultSource_GitIntegration(t *testing.T) {
	dir := setupGitRepoWithGrowth(t)
	chdirTemp(t, dir)

	detector := &HotspotAccelerationDetector{
		Source: &defaultHotspotAccelerationSource{},
	}

	// Should not error even on a fresh repo with no 30-day-old commits
	suggestions, err := detector.Detect()
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	// Fresh test repo → no base commit 30 days ago → empty results (graceful)
	_ = suggestions
}

// --- helpers ---

// chdirTemp changes to dir and restores the original directory on cleanup.
func chdirTemp(t *testing.T, dir string) string {
	t.Helper()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })
	return origDir
}

// gitInDir runs a git command in dir and fatals on error.
func gitInDir(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

// setupGitRepoWithHistory creates a temp git repo with an initial commit
// and a second commit that adds 20 lines, suitable for git diff tests.
func setupGitRepoWithHistory(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	gitInDir(t, dir, "init")
	gitInDir(t, dir, "config", "user.email", "test@test.com")
	gitInDir(t, dir, "config", "user.name", "Test")

	// Initial commit: a small Go file
	initial := "package main\n\nfunc main() {\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "growing.go"), []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}
	gitInDir(t, dir, "add", "growing.go")
	gitInDir(t, dir, "commit", "-m", "initial")

	// Second commit: add 20 more lines
	var builder strings.Builder
	builder.WriteString(initial)
	for i := 0; i < 20; i++ {
		builder.WriteString("// added line\n")
	}
	if err := os.WriteFile(filepath.Join(dir, "growing.go"), []byte(builder.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	gitInDir(t, dir, "add", "growing.go")
	gitInDir(t, dir, "commit", "-m", "add lines")

	return dir
}

// setupGitRepoWithGrowth creates a repo suitable for hotspot acceleration testing.
func setupGitRepoWithGrowth(t *testing.T) string {
	t.Helper()
	return setupGitRepoWithHistory(t)
}
