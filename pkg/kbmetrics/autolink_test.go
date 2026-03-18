package kbmetrics

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractKeywords_Filename(t *testing.T) {
	// Should extract meaningful keywords from investigation filenames
	keywords := ExtractKeywords("2026-03-17-inv-hotspot-acceleration-cmd-orch-serve.md", "# Investigation\n\nSome content")
	if len(keywords) == 0 {
		t.Fatal("expected keywords from filename")
	}
	found := false
	for _, kw := range keywords {
		if kw == "hotspot" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'hotspot' in keywords, got %v", keywords)
	}
}

func TestExtractKeywords_Title(t *testing.T) {
	content := `# Investigation: Daemon Spawn Rate Analysis

## Question
Why are daemon spawns slow?

## Findings
The spawn rate is limited by account rotation.
`
	keywords := ExtractKeywords("inv-test.md", content)

	wantAny := []string{"daemon", "spawn", "rate"}
	for _, w := range wantAny {
		found := false
		for _, kw := range keywords {
			if kw == w {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected keyword %q in %v", w, keywords)
		}
	}
}

func TestExtractKeywords_StopWords(t *testing.T) {
	keywords := ExtractKeywords("inv-test.md", "# The Investigation of a Simple Problem\n")
	for _, kw := range keywords {
		if kw == "the" || kw == "of" || kw == "a" {
			t.Errorf("stop word %q should be filtered", kw)
		}
	}
}

func TestScoreMatch(t *testing.T) {
	invKeywords := []string{"daemon", "spawn", "rate", "analysis"}

	// Model with overlapping topic should score higher
	score1 := ScoreMatch(invKeywords, "daemon-autonomous-operation", []string{"daemon", "autonomous", "ooda", "spawn"})
	score2 := ScoreMatch(invKeywords, "harness-engineering", []string{"harness", "measurement", "gates"})

	if score1 <= score2 {
		t.Errorf("daemon model (score=%d) should score higher than harness model (score=%d)", score1, score2)
	}
}

func TestScoreMatch_ModelNameBonus(t *testing.T) {
	// Keywords matching the model name itself should get a bonus
	invKeywords := []string{"completion", "verification", "gates"}
	score := ScoreMatch(invKeywords, "completion-verification", []string{"gates", "pipeline"})
	if score == 0 {
		t.Error("expected non-zero score when keywords match model name parts")
	}
}

func TestScoreMatch_NoOverlap(t *testing.T) {
	score := ScoreMatch(
		[]string{"daemon", "spawn"},
		"beads-database-corruption",
		[]string{"sqlite", "corruption", "wal"},
	)
	if score != 0 {
		t.Errorf("expected 0 score for no overlap, got %d", score)
	}
}

func TestFindAutoLinks_Basic(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	invDir := filepath.Join(kbDir, "investigations")
	modelDir := filepath.Join(kbDir, "models", "daemon-autonomous-operation")
	os.MkdirAll(invDir, 0755)
	os.MkdirAll(modelDir, 0755)

	// Create an orphaned investigation about daemon spawning (needs 5+ content lines)
	invContent := `# Investigation: Daemon Spawn Failures

## Question
Why do daemon spawns occasionally fail?

## Findings
The daemon spawn logic has a race condition in slot allocation.
Account rotation during spawn causes intermittent failures.
When two OODA cycles fire simultaneously, both attempt to claim the same slot.
The daemon scheduler does not lock the slot table during allocation.
This leads to duplicate spawns that consume extra tokens and confuse completion tracking.
Root cause is missing mutex in the periodic spawn path.
`
	os.WriteFile(filepath.Join(invDir, "2026-03-10-inv-daemon-spawn-failures.md"), []byte(invContent), 0644)

	// Create a model about daemon
	modelContent := `# Model: Daemon Autonomous Operation

**Domain:** Daemon / OODA Cycle
**Last Updated:** 2026-03-10

## Summary
The daemon operates via OODA cycles, spawning agents autonomously.
`
	os.WriteFile(filepath.Join(modelDir, "model.md"), []byte(modelContent), 0644)

	links, err := FindAutoLinks(kbDir, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(links) == 0 {
		t.Fatal("expected at least one auto-link suggestion")
	}

	found := false
	for _, link := range links {
		if strings.Contains(link.InvestigationPath, "daemon-spawn-failures") &&
			strings.Contains(link.TargetPath, "daemon-autonomous-operation") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected link from daemon-spawn-failures to daemon-autonomous-operation, got %v", links)
	}
}

func TestFindAutoLinks_SkipsConnected(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	invDir := filepath.Join(kbDir, "investigations")
	modelDir := filepath.Join(kbDir, "models", "test-model")
	os.MkdirAll(invDir, 0755)
	os.MkdirAll(modelDir, 0755)

	os.WriteFile(filepath.Join(invDir, "inv-already-linked.md"), []byte("# Already Linked\nSome findings about test model.\n"), 0644)

	// Model already references this investigation
	modelContent := `# Model: Test
See .kb/investigations/inv-already-linked.md for details.
`
	os.WriteFile(filepath.Join(modelDir, "model.md"), []byte(modelContent), 0644)

	links, err := FindAutoLinks(kbDir, 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, link := range links {
		if strings.Contains(link.InvestigationPath, "already-linked") {
			t.Error("should not suggest links for already-connected investigations")
		}
	}
}

func TestFindAutoLinks_SkipsEmpty(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	invDir := filepath.Join(kbDir, "investigations")
	modelDir := filepath.Join(kbDir, "models", "test-model")
	os.MkdirAll(invDir, 0755)
	os.MkdirAll(modelDir, 0755)

	// Empty investigation (template only)
	os.WriteFile(filepath.Join(invDir, "inv-empty.md"), []byte("# Investigation\n---\n"), 0644)

	modelContent := `# Model: Test
**Domain:** Testing
`
	os.WriteFile(filepath.Join(modelDir, "model.md"), []byte(modelContent), 0644)

	links, err := FindAutoLinks(kbDir, 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, link := range links {
		if strings.Contains(link.InvestigationPath, "inv-empty") {
			t.Error("should not suggest links for empty investigations")
		}
	}
}

func TestApplyAutoLinks(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	modelDir := filepath.Join(kbDir, "models", "test-model")
	os.MkdirAll(modelDir, 0755)

	modelContent := `# Model: Test

## Summary
A test model.
`
	modelPath := filepath.Join(modelDir, "model.md")
	os.WriteFile(modelPath, []byte(modelContent), 0644)

	links := []AutoLink{
		{
			InvestigationPath: ".kb/investigations/inv-foo.md",
			TargetPath:        modelPath,
			TargetName:        "test-model",
			TargetType:        "model",
			Score:             3,
		},
	}

	applied, err := ApplyAutoLinks(links)
	if err != nil {
		t.Fatal(err)
	}
	if applied != 1 {
		t.Errorf("applied = %d, want 1", applied)
	}

	// Verify the model file now references the investigation
	data, _ := os.ReadFile(modelPath)
	content := string(data)
	if !strings.Contains(content, ".kb/investigations/inv-foo.md") {
		t.Errorf("model file should contain investigation reference, got:\n%s", content)
	}
}

func TestApplyAutoLinks_AppendsToExistingSection(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	modelDir := filepath.Join(kbDir, "models", "test-model")
	os.MkdirAll(modelDir, 0755)

	modelContent := `# Model: Test

## Summary
A test model.

## Auto-Linked Investigations

- .kb/investigations/inv-existing.md
`
	modelPath := filepath.Join(modelDir, "model.md")
	os.WriteFile(modelPath, []byte(modelContent), 0644)

	links := []AutoLink{
		{
			InvestigationPath: ".kb/investigations/inv-new.md",
			TargetPath:        modelPath,
			TargetName:        "test-model",
			TargetType:        "model",
			Score:             3,
		},
	}

	applied, err := ApplyAutoLinks(links)
	if err != nil {
		t.Fatal(err)
	}
	if applied != 1 {
		t.Errorf("applied = %d, want 1", applied)
	}

	data, _ := os.ReadFile(modelPath)
	content := string(data)
	if !strings.Contains(content, "inv-existing.md") {
		t.Error("should preserve existing references")
	}
	if !strings.Contains(content, "inv-new.md") {
		t.Error("should add new reference")
	}
	// Should not create duplicate section headers
	if strings.Count(content, "## Auto-Linked Investigations") != 1 {
		t.Error("should not duplicate section header")
	}
}

func TestFindAutoLinks_ThreadMatching(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	invDir := filepath.Join(kbDir, "investigations")
	threadDir := filepath.Join(kbDir, "threads")
	os.MkdirAll(invDir, 0755)
	os.MkdirAll(threadDir, 0755)

	invContent := `# Investigation: Harness Measurement Gaps

## Findings
The harness measurement system has blind spots in gate enforcement tracking.
Gate fire rates are not correlated with actual defect catches.
The enforcement layer applies rules but does not measure whether rules improve outcomes.
Measurement and enforcement are decoupled at the data layer, preventing feedback loops.
Precision tracking for individual gates is missing entirely from the dashboard.
This means the harness runs gates that may not be helping and we cannot tell which ones.
`
	os.WriteFile(filepath.Join(invDir, "inv-harness-measurement.md"), []byte(invContent), 0644)

	threadContent := `---
title: "Measurement Enforcement Pairing"
status: open
---

# Measurement Enforcement Pairing

## 2026-03-12
How measurement and enforcement relate in the harness system.
`
	os.WriteFile(filepath.Join(threadDir, "2026-03-12-measurement-enforcement-pairing.md"), []byte(threadContent), 0644)

	links, err := FindAutoLinks(kbDir, 1)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, link := range links {
		if strings.Contains(link.InvestigationPath, "harness-measurement") &&
			link.TargetType == "thread" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected link from harness investigation to measurement thread, got %v", links)
	}
}
