package thread

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPromote_ConvergedThread(t *testing.T) {
	dir := tempThreadsDir(t)

	content := `---
title: "Named incompleteness"
status: converged
created: 2026-03-20
updated: 2026-03-27
resolved_to: ""
---

# Named incompleteness

## 2026-03-27

Generative systems are organized around named incompleteness.
`
	os.WriteFile(filepath.Join(dir, "2026-03-20-named-incompleteness.md"), []byte(content), 0644)

	targetPath := ".kb/models/named-incompleteness/model.md"
	err := Promote(dir, "named-incompleteness", "model", targetPath)
	if err != nil {
		t.Fatalf("Promote failed: %v", err)
	}

	// Verify via Show
	shown, err := Show(dir, "named-incompleteness")
	if err != nil {
		t.Fatal(err)
	}
	if shown.Status != StatusPromoted {
		t.Errorf("status = %q, want promoted", shown.Status)
	}
	if shown.PromotedTo != targetPath {
		t.Errorf("promoted_to = %q, want %q", shown.PromotedTo, targetPath)
	}

	// Promoted thread should not appear in active threads
	active, err := ActiveThreads(dir, 30)
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 0 {
		t.Errorf("promoted thread should not appear in active threads, got %d", len(active))
	}
}

func TestPromote_PropagatesAncestors(t *testing.T) {
	dir := tempThreadsDir(t)

	ancestor := `---
title: "Absorbed insight"
status: subsumed
created: 2026-03-15
updated: 2026-03-20
resolved_to: "named-incompleteness"
---

# Absorbed insight

## 2026-03-15

This was absorbed.
`
	converged := `---
title: "Named incompleteness"
status: converged
created: 2026-03-20
updated: 2026-03-27
resolved_to: ""
---

# Named incompleteness

## 2026-03-27

Converged thinking.
`
	os.WriteFile(filepath.Join(dir, "2026-03-15-absorbed-insight.md"), []byte(ancestor), 0644)
	os.WriteFile(filepath.Join(dir, "2026-03-20-named-incompleteness.md"), []byte(converged), 0644)

	targetPath := ".kb/models/named-incompleteness/model.md"
	if err := Promote(dir, "named-incompleteness", "model", targetPath); err != nil {
		t.Fatal(err)
	}

	ancestorThread, err := Show(dir, "absorbed-insight")
	if err != nil {
		t.Fatal(err)
	}
	if ancestorThread.ResolvedTo != targetPath {
		t.Errorf("ancestor resolved_to = %q, want %q", ancestorThread.ResolvedTo, targetPath)
	}
}

func TestPromote_NotFound(t *testing.T) {
	dir := tempThreadsDir(t)

	err := Promote(dir, "nonexistent", "model", ".kb/models/test/model.md")
	if err == nil {
		t.Error("expected error for nonexistent thread")
	}
}

func TestPromotionReady(t *testing.T) {
	dir := tempThreadsDir(t)

	converged := `---
title: "Ready to promote"
status: converged
created: 2026-03-20
updated: 2026-03-27
resolved_to: ""
---

# Ready to promote

## 2026-03-27

Done thinking.
`
	promoted := `---
title: "Already promoted"
status: promoted
created: 2026-03-15
updated: 2026-03-25
resolved_to: ""
promoted_to: ".kb/models/test/model.md"
---

# Already promoted

## 2026-03-25

Was promoted.
`
	active := `---
title: "Still active"
status: active
created: 2026-03-10
updated: 2026-03-27
resolved_to: ""
---

# Still active

## 2026-03-27

Working.
`
	os.WriteFile(filepath.Join(dir, "2026-03-20-ready-promote.md"), []byte(converged), 0644)
	os.WriteFile(filepath.Join(dir, "2026-03-15-already-promoted.md"), []byte(promoted), 0644)
	os.WriteFile(filepath.Join(dir, "2026-03-10-still-active.md"), []byte(active), 0644)

	candidates, err := PromotionReady(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	if candidates[0].Slug != "ready-promote" {
		t.Errorf("name = %q, want ready-promote", candidates[0].Slug)
	}
	if candidates[0].Title != "Ready to promote" {
		t.Errorf("title = %q", candidates[0].Title)
	}
}

func TestIsResolved_IncludesPromoted(t *testing.T) {
	if !IsResolved(StatusPromoted) {
		t.Error("IsResolved should return true for promoted status")
	}
	if IsActive(StatusPromoted) {
		t.Error("IsActive should return false for promoted status")
	}
}

func TestParseThread_PromotedTo(t *testing.T) {
	content := `---
title: "Promoted thread"
status: promoted
created: 2026-03-01
updated: 2026-03-27
resolved_to: ""
promoted_to: ".kb/models/named-incompleteness/model.md"
---

# Promoted thread

## 2026-03-01

Some insight.
`
	thread, err := ParseThread(content)
	if err != nil {
		t.Fatalf("ParseThread failed: %v", err)
	}
	if thread.Status != StatusPromoted {
		t.Errorf("status = %q, want %q", thread.Status, StatusPromoted)
	}
	if thread.PromotedTo != ".kb/models/named-incompleteness/model.md" {
		t.Errorf("promoted_to = %q, want '.kb/models/named-incompleteness/model.md'", thread.PromotedTo)
	}
}
