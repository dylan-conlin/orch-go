package main

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/orient"
)

func TestExploreCandidatesFromThreads(t *testing.T) {
	dir := t.TempDir()

	// Write a thread with many entries (should be a candidate)
	writeTestThread(t, dir, "big-thread.md", `---
title: Architectural displacement patterns
status: open
created: 2026-03-01
updated: 2026-03-19
---

## 2026-03-01
First entry about the pattern.

## 2026-03-05
Second entry with more observations.

## 2026-03-08
Third entry — seeing this more often.

## 2026-03-12
Fourth entry with concrete examples.

## 2026-03-15
Fifth entry — this really needs synthesis.

## 2026-03-19
Sixth entry — still accumulating.
`)

	// Write a thread with few entries (should NOT be a candidate)
	writeTestThread(t, dir, "small-thread.md", `---
title: Minor observation
status: open
created: 2026-03-18
updated: 2026-03-19
---

## 2026-03-18
Just one note.
`)

	candidates := exploreCandidatesFromThreads(dir)

	if len(candidates) == 0 {
		t.Fatal("expected at least 1 explore candidate from threads")
	}
	if len(candidates) > 2 {
		t.Errorf("expected at most 2 candidates, got %d", len(candidates))
	}

	c := candidates[0]
	if c.Signal != "thread-accumulation" {
		t.Errorf("expected signal 'thread-accumulation', got %q", c.Signal)
	}
	if c.Score <= 0 {
		t.Error("expected positive score")
	}
}

func TestExploreCandidatesFromUntestedClaims(t *testing.T) {
	dir := t.TempDir()
	writeTestClaimsYAML(t, dir, "test-model", `model: test-model
version: 1
claims:
  - id: TM-01
    text: "Tested claim"
    type: observation
    scope: local
    confidence: confirmed
    priority: core
    last_validated: "2026-03-15"
  - id: TM-02
    text: "Untested claim 1"
    type: mechanism
    scope: bounded
    confidence: unconfirmed
    priority: core
  - id: TM-03
    text: "Untested claim 2"
    type: generalization
    scope: universal
    confidence: unconfirmed
    priority: supporting
  - id: TM-04
    text: "Untested claim 3"
    type: observation
    scope: local
    confidence: unconfirmed
    priority: peripheral
`)

	now := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
	candidates := exploreCandidatesFromUntestedClaims(dir, now)

	if len(candidates) == 0 {
		t.Fatal("expected at least 1 explore candidate from untested claims")
	}

	c := candidates[0]
	if c.Signal != "untested-claims" {
		t.Errorf("expected signal 'untested-claims', got %q", c.Signal)
	}
	if c.Score <= 0 {
		t.Error("expected positive score")
	}
}

func TestExploreCandidatesFromUntestedClaims_BelowThreshold(t *testing.T) {
	dir := t.TempDir()
	writeTestClaimsYAML(t, dir, "almost-model", `model: almost-model
version: 1
claims:
  - id: AM-01
    text: "Confirmed"
    type: observation
    scope: local
    confidence: confirmed
    priority: core
    last_validated: "2026-03-15"
  - id: AM-02
    text: "Untested 1"
    type: observation
    scope: local
    confidence: unconfirmed
    priority: core
  - id: AM-03
    text: "Untested 2"
    type: observation
    scope: local
    confidence: unconfirmed
    priority: core
`)

	now := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
	candidates := exploreCandidatesFromUntestedClaims(dir, now)

	if len(candidates) != 0 {
		t.Errorf("expected 0 candidates (below threshold of 3), got %d", len(candidates))
	}
}

func TestCollectModelNames(t *testing.T) {
	dir := t.TempDir()

	for _, name := range []string{"spawn-architecture", "daemon-autonomous-operation", "harness-engineering"} {
		writeTestClaimsYAML(t, dir, name, "model: "+name+"\nversion: 1\nclaims: []\n")
	}

	names := collectModelNames(dir)
	if len(names) != 3 {
		t.Fatalf("expected 3 model names, got %d: %v", len(names), names)
	}
}

func TestCollectModelNames_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	names := collectModelNames(dir)
	if len(names) != 0 {
		t.Errorf("expected 0 model names for empty dir, got %d", len(names))
	}
}

func TestFormatExploreCandidates(t *testing.T) {
	data := &orient.OrientationData{
		ExploreCandidates: []orient.ExploreCandidate{
			{
				Question: "Resolve tension between models A and B",
				Signal:   "tension-cluster",
				Score:    8.0,
				Reason:   "3 models, domains: spawn, gates",
			},
			{
				Question: "Probe untested claims in model 'spawn-arch'",
				Signal:   "untested-claims",
				Score:    6.0,
				Reason:   "4 claims need validation",
			},
		},
	}

	// Explore candidates now render in FormatHealth, not FormatOrientation
	output := orient.FormatHealth(data)

	for _, want := range []string{
		"Explore candidates:",
		"[tension-cluster]",
		"[untested-claims]",
		"Resolve tension between",
		"3 models, domains: spawn, gates",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q", want)
		}
	}
}

func TestFormatExploreCandidates_Empty(t *testing.T) {
	data := &orient.OrientationData{}
	output := orient.FormatHealth(data)

	if strings.Contains(output, "Explore candidates:") {
		t.Error("output should not contain 'Explore candidates:' when empty")
	}
}

// --- test helpers ---

func writeTestThread(t *testing.T, dir, filename, content string) {
	t.Helper()
	if err := os.WriteFile(dir+"/"+filename, []byte(content), 0644); err != nil {
		t.Fatalf("write thread %s: %v", filename, err)
	}
}

func writeTestClaimsYAML(t *testing.T, modelsDir, modelName, content string) {
	t.Helper()
	modelDir := modelsDir + "/" + modelName
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", modelDir, err)
	}
	if err := os.WriteFile(modelDir+"/claims.yaml", []byte(content), 0644); err != nil {
		t.Fatalf("write claims %s: %v", modelDir, err)
	}
}
