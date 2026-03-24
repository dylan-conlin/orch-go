package thread

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLinkWork(t *testing.T) {
	dir := tempThreadsDir(t)

	threadContent := `---
title: "Link test"
status: open
created: 2026-03-01
updated: 2026-03-01
resolved_to: ""
---

# Link test

## 2026-03-01

Entry.
`
	os.WriteFile(filepath.Join(dir, "2026-03-01-link-test.md"), []byte(threadContent), 0644)

	if err := LinkWork(dir, "link-test", "orch-go-abc12"); err != nil {
		t.Fatalf("LinkWork failed: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "2026-03-01-link-test.md"))
	thread, _ := ParseThread(string(content))

	if len(thread.ActiveWork) != 1 || thread.ActiveWork[0] != "orch-go-abc12" {
		t.Errorf("active_work = %v, want [orch-go-abc12]", thread.ActiveWork)
	}
}

func TestLinkWork_Duplicate(t *testing.T) {
	dir := tempThreadsDir(t)

	threadContent := `---
title: "Dup test"
status: open
created: 2026-03-01
updated: 2026-03-01
resolved_to: ""
active_work:
  - "orch-go-abc12"
---

# Dup test

## 2026-03-01

Entry.
`
	os.WriteFile(filepath.Join(dir, "2026-03-01-dup-test.md"), []byte(threadContent), 0644)

	if err := LinkWork(dir, "dup-test", "orch-go-abc12"); err != nil {
		t.Fatalf("LinkWork failed: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "2026-03-01-dup-test.md"))
	thread, _ := ParseThread(string(content))
	if len(thread.ActiveWork) != 1 {
		t.Errorf("expected 1 active_work (deduped), got %d: %v", len(thread.ActiveWork), thread.ActiveWork)
	}
}

func TestLinkWork_NotFound(t *testing.T) {
	dir := tempThreadsDir(t)
	if err := LinkWork(dir, "nonexistent", "orch-go-abc12"); err == nil {
		t.Fatal("expected error for nonexistent thread")
	}
}

func TestAddSpawned(t *testing.T) {
	dir := tempThreadsDir(t)

	threadContent := `---
title: "Parent thread"
status: open
created: 2026-03-01
updated: 2026-03-01
resolved_to: ""
---

# Parent thread

## 2026-03-01

Entry.
`
	os.WriteFile(filepath.Join(dir, "2026-03-01-parent-thread.md"), []byte(threadContent), 0644)

	if err := AddSpawned(dir, "parent-thread", "child-thread"); err != nil {
		t.Fatalf("AddSpawned failed: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "2026-03-01-parent-thread.md"))
	thread, _ := ParseThread(string(content))

	if len(thread.Spawned) != 1 || thread.Spawned[0] != "child-thread" {
		t.Errorf("spawned = %v, want [child-thread]", thread.Spawned)
	}
}

func TestCreateWithParent(t *testing.T) {
	dir := tempThreadsDir(t)

	parentContent := `---
title: "Parent"
status: open
created: 2026-03-01
updated: 2026-03-01
resolved_to: ""
---

# Parent

## 2026-03-01

Entry.
`
	os.WriteFile(filepath.Join(dir, "2026-03-01-parent-slug.md"), []byte(parentContent), 0644)

	result, err := CreateWithParent(dir, "Child thread title", "First insight", "parent-slug")
	if err != nil {
		t.Fatalf("CreateWithParent failed: %v", err)
	}
	if !result.Created {
		t.Error("expected Created=true")
	}

	// Verify child has spawned_from
	content, _ := os.ReadFile(result.FilePath)
	child, _ := ParseThread(string(content))
	if child.SpawnedFrom != "parent-slug" {
		t.Errorf("spawned_from = %q, want 'parent-slug'", child.SpawnedFrom)
	}

	// Verify parent has child in spawned list
	parentData, _ := os.ReadFile(filepath.Join(dir, "2026-03-01-parent-slug.md"))
	parent, _ := ParseThread(string(parentData))
	childSlug := Slugify("Child thread title")
	found := false
	for _, s := range parent.Spawned {
		if s == childSlug {
			found = true
		}
	}
	if !found {
		t.Errorf("parent spawned %v does not contain child %q", parent.Spawned, childSlug)
	}
}

// tempRelationsDir is a helper for relations tests.
func tempRelationsDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	threadsDir := filepath.Join(dir, "threads")
	if err := os.MkdirAll(threadsDir, 0755); err != nil {
		t.Fatal(err)
	}
	return threadsDir
}

func TestLinkWork_MultipleItems(t *testing.T) {
	dir := tempRelationsDir(t)

	threadContent := `---
title: "Multi test"
status: open
created: 2026-03-01
updated: 2026-03-01
resolved_to: ""
---

# Multi test

## 2026-03-01

Entry.
`
	os.WriteFile(filepath.Join(dir, "2026-03-01-multi-test.md"), []byte(threadContent), 0644)

	LinkWork(dir, "multi-test", "orch-go-abc12")
	LinkWork(dir, "multi-test", "orch-go-def34")

	content, _ := os.ReadFile(filepath.Join(dir, "2026-03-01-multi-test.md"))
	thread, _ := ParseThread(string(content))

	if len(thread.ActiveWork) != 2 {
		t.Fatalf("expected 2 active_work, got %d: %v", len(thread.ActiveWork), thread.ActiveWork)
	}
	if !strings.Contains(thread.ActiveWork[0], "abc12") || !strings.Contains(thread.ActiveWork[1], "def34") {
		t.Errorf("unexpected active_work: %v", thread.ActiveWork)
	}
}
