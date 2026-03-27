package thread

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func tempThreadsDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	threadsDir := filepath.Join(dir, "threads")
	if err := os.MkdirAll(threadsDir, 0755); err != nil {
		t.Fatal(err)
	}
	return threadsDir
}

func TestCreateOrAppend_NewThread(t *testing.T) {
	dir := tempThreadsDir(t)
	today := time.Now().Format("2006-01-02")

	result, err := CreateOrAppend(dir, "How enforcement and comprehension relate", "First insight about this topic")
	if err != nil {
		t.Fatalf("CreateOrAppend failed: %v", err)
	}

	if result.Created {
		// good
	} else {
		t.Error("expected Created=true for new thread")
	}

	if result.EntryCount != 1 {
		t.Errorf("expected EntryCount=1, got %d", result.EntryCount)
	}

	// Verify file exists with expected slug
	expectedSlug := today + "-enforcement-comprehension-relate"
	expectedPath := filepath.Join(dir, expectedSlug+".md")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		// Try to find what was actually created
		entries, _ := os.ReadDir(dir)
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("expected file %s not found, files: %v", expectedSlug+".md", names)
	}

	// Verify content
	content, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatal(err)
	}

	s := string(content)
	if !strings.Contains(s, "title: \"How enforcement and comprehension relate\"") {
		t.Error("missing title in frontmatter")
	}
	if !strings.Contains(s, "status: forming") {
		t.Error("missing status in frontmatter")
	}
	if !strings.Contains(s, "created: "+today) {
		t.Error("missing created date in frontmatter")
	}
	if !strings.Contains(s, "## "+today) {
		t.Error("missing dated section heading")
	}
	if !strings.Contains(s, "First insight about this topic") {
		t.Error("missing entry text")
	}
}

func TestCreateOrAppend_AppendNewDate(t *testing.T) {
	dir := tempThreadsDir(t)

	// Create a thread with a past date
	pastContent := `---
title: "Test thread"
status: open
created: 2026-03-01
updated: 2026-03-01
resolved_to: ""
---

# Test thread

## 2026-03-01

Original insight here.
`
	if err := os.WriteFile(filepath.Join(dir, "2026-03-01-test-thread.md"), []byte(pastContent), 0644); err != nil {
		t.Fatal(err)
	}

	today := time.Now().Format("2006-01-02")
	result, err := CreateOrAppend(dir, "test-thread", "New insight today")
	if err != nil {
		t.Fatalf("CreateOrAppend append failed: %v", err)
	}

	if result.Created {
		t.Error("expected Created=false for append")
	}

	// Read back
	content, err := os.ReadFile(filepath.Join(dir, "2026-03-01-test-thread.md"))
	if err != nil {
		t.Fatal(err)
	}

	s := string(content)
	if !strings.Contains(s, "## "+today) {
		t.Error("missing new dated section")
	}
	if !strings.Contains(s, "New insight today") {
		t.Error("missing appended entry")
	}
	if !strings.Contains(s, "Original insight here") {
		t.Error("original entry was lost")
	}
	if !strings.Contains(s, "updated: "+today) {
		t.Error("updated date not refreshed in frontmatter")
	}
}

func TestCreateOrAppend_AppendSameDate(t *testing.T) {
	dir := tempThreadsDir(t)
	today := time.Now().Format("2006-01-02")

	// Create thread
	_, err := CreateOrAppend(dir, "Same day thread", "First entry")
	if err != nil {
		t.Fatal(err)
	}

	// Append same day
	result, err := CreateOrAppend(dir, "same-day-thread", "Second entry")
	if err != nil {
		t.Fatal(err)
	}

	if result.Created {
		t.Error("expected Created=false for same-day append")
	}

	// Read back — should have one dated heading, two entries
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("expected 1 file, got %d", len(entries))
	}

	content, _ := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	s := string(content)

	// Should only have ONE instance of today's heading
	count := strings.Count(s, "## "+today)
	if count != 1 {
		t.Errorf("expected 1 occurrence of today's heading, got %d", count)
	}

	if !strings.Contains(s, "First entry") {
		t.Error("first entry missing")
	}
	if !strings.Contains(s, "Second entry") {
		t.Error("second entry missing")
	}
}

func TestList(t *testing.T) {
	dir := tempThreadsDir(t)

	// Create two threads
	thread1 := `---
title: "Thread one"
status: open
created: 2026-03-01
updated: 2026-03-05
resolved_to: ""
---

# Thread one

## 2026-03-05

Latest entry for thread one.
`
	thread2 := `---
title: "Thread two"
status: resolved
created: 2026-03-02
updated: 2026-03-04
resolved_to: ".kb/models/test.md"
---

# Thread two

## 2026-03-04

Resolved entry.
`
	os.WriteFile(filepath.Join(dir, "2026-03-01-thread-one.md"), []byte(thread1), 0644)
	os.WriteFile(filepath.Join(dir, "2026-03-02-thread-two.md"), []byte(thread2), 0644)

	threads, err := List(dir)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(threads) != 2 {
		t.Fatalf("expected 2 threads, got %d", len(threads))
	}

	// Should be sorted by updated date descending (most recent first)
	if threads[0].Title != "Thread one" {
		t.Errorf("expected most recently updated first, got %q", threads[0].Title)
	}
	if threads[0].Status != StatusForming {
		t.Errorf("expected forming status, got %q", threads[0].Status)
	}
	if threads[1].Status != "resolved" {
		t.Errorf("expected resolved status, got %q", threads[1].Status)
	}
	if threads[1].ResolvedTo != ".kb/models/test.md" {
		t.Errorf("expected resolved_to path, got %q", threads[1].ResolvedTo)
	}
}

func TestShow(t *testing.T) {
	dir := tempThreadsDir(t)

	content := `---
title: "Show test"
status: open
created: 2026-03-01
updated: 2026-03-01
resolved_to: ""
---

# Show test

## 2026-03-01

Test content here.
`
	os.WriteFile(filepath.Join(dir, "2026-03-01-show-test.md"), []byte(content), 0644)

	thread, err := Show(dir, "show-test")
	if err != nil {
		t.Fatalf("Show failed: %v", err)
	}

	if thread.Title != "Show test" {
		t.Errorf("expected title 'Show test', got %q", thread.Title)
	}
	if thread.Content == "" {
		t.Error("expected non-empty content")
	}
}

func TestShow_NotFound(t *testing.T) {
	dir := tempThreadsDir(t)

	_, err := Show(dir, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent thread")
	}
}

func TestResolve(t *testing.T) {
	dir := tempThreadsDir(t)

	content := `---
title: "Resolve test"
status: open
created: 2026-03-01
updated: 2026-03-01
resolved_to: ""
---

# Resolve test

## 2026-03-01

Some insight.
`
	os.WriteFile(filepath.Join(dir, "2026-03-01-resolve-test.md"), []byte(content), 0644)

	err := Resolve(dir, "resolve-test", ".kb/models/enforcement.md")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// Read back and verify
	updated, _ := os.ReadFile(filepath.Join(dir, "2026-03-01-resolve-test.md"))
	s := string(updated)

	if !strings.Contains(s, "status: resolved") {
		t.Error("status not updated to resolved")
	}
	if !strings.Contains(s, "resolved_to: \".kb/models/enforcement.md\"") {
		t.Error("resolved_to not set")
	}
}

func TestUpdateStatus(t *testing.T) {
	dir := tempThreadsDir(t)

	content := `---
title: "Update status test"
status: active
created: 2026-03-01
updated: 2026-03-01
resolved_to: ""
---

# Update status test

## 2026-03-01

Some insight.
`
	os.WriteFile(filepath.Join(dir, "2026-03-01-update-status-test.md"), []byte(content), 0644)

	err := UpdateStatus(dir, "update-status-test", StatusResolved, "decision: shipped in dashboard")
	if err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	updated, _ := os.ReadFile(filepath.Join(dir, "2026-03-01-update-status-test.md"))
	s := string(updated)

	if !strings.Contains(s, "status: resolved") {
		t.Error("status not updated")
	}
	if !strings.Contains(s, "resolved_to: \"decision: shipped in dashboard\"") {
		t.Error("resolved_to not updated")
	}
}

func TestTodaysEntries(t *testing.T) {
	dir := tempThreadsDir(t)
	today := time.Now().Format("2006-01-02")

	// Thread with today's entry
	thread1 := `---
title: "Today thread"
status: open
created: ` + today + `
updated: ` + today + `
resolved_to: ""
---

# Today thread

## ` + today + `

Insight from today.
`
	// Thread with old entry only
	thread2 := `---
title: "Old thread"
status: open
created: 2026-03-01
updated: 2026-03-01
resolved_to: ""
---

# Old thread

## 2026-03-01

Old insight.
`
	// Resolved thread with today's entry (should be excluded)
	thread3 := `---
title: "Resolved thread"
status: resolved
created: ` + today + `
updated: ` + today + `
resolved_to: ".kb/decisions/test.md"
---

# Resolved thread

## ` + today + `

This is resolved.
`
	os.WriteFile(filepath.Join(dir, today+"-today-thread.md"), []byte(thread1), 0644)
	os.WriteFile(filepath.Join(dir, "2026-03-01-old-thread.md"), []byte(thread2), 0644)
	os.WriteFile(filepath.Join(dir, today+"-resolved-thread.md"), []byte(thread3), 0644)

	entries, err := TodaysEntries(dir, today)
	if err != nil {
		t.Fatalf("TodaysEntries failed: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry (only open thread with today's date), got %d", len(entries))
	}

	if entries[0].ThreadName != "today-thread" {
		t.Errorf("expected thread name 'today-thread', got %q", entries[0].ThreadName)
	}
	if !strings.Contains(entries[0].Text, "Insight from today") {
		t.Errorf("expected entry text, got %q", entries[0].Text)
	}
}

func TestActiveThreads(t *testing.T) {
	dir := tempThreadsDir(t)
	today := time.Now().Format("2006-01-02")

	// Recent open thread
	thread1 := `---
title: "Recent open"
status: open
created: ` + today + `
updated: ` + today + `
resolved_to: ""
---

# Recent open

## ` + today + `

Fresh.
`
	// Old open thread (stale)
	thread2 := `---
title: "Stale open"
status: open
created: 2026-01-01
updated: 2026-01-01
resolved_to: ""
---

# Stale open

## 2026-01-01

Ancient.
`
	// Resolved thread
	thread3 := `---
title: "Done"
status: resolved
created: 2026-03-01
updated: ` + today + `
resolved_to: ".kb/models/test.md"
---

# Done

## ` + today + `

Resolved.
`
	os.WriteFile(filepath.Join(dir, today+"-recent-open.md"), []byte(thread1), 0644)
	os.WriteFile(filepath.Join(dir, "2026-01-01-stale-open.md"), []byte(thread2), 0644)
	os.WriteFile(filepath.Join(dir, "2026-03-01-done.md"), []byte(thread3), 0644)

	// maxAge=7 days — only recent open should appear
	active, err := ActiveThreads(dir, 7)
	if err != nil {
		t.Fatalf("ActiveThreads failed: %v", err)
	}

	if len(active) != 1 {
		t.Fatalf("expected 1 active thread, got %d", len(active))
	}
	if active[0].Title != "Recent open" {
		t.Errorf("expected 'Recent open', got %q", active[0].Title)
	}
}

func TestAppend_NonexistentSlug(t *testing.T) {
	dir := tempThreadsDir(t)

	_, err := Append(dir, "nonexistent-thread", "some text")
	if err == nil {
		t.Fatal("expected error for nonexistent thread, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got %q", err.Error())
	}

	// Verify no file was created
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Errorf("expected no files created, got %d", len(entries))
	}
}

func TestAppend_ExistingThread(t *testing.T) {
	dir := tempThreadsDir(t)

	// Create a thread first
	threadContent := `---
title: "Append target"
status: open
created: 2026-03-01
updated: 2026-03-01
resolved_to: ""
---

# Append target

## 2026-03-01

Original entry.
`
	os.WriteFile(filepath.Join(dir, "2026-03-01-append-target.md"), []byte(threadContent), 0644)

	result, err := Append(dir, "append-target", "New appended content")
	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	if result.Created {
		t.Error("expected Created=false for append")
	}

	content, _ := os.ReadFile(filepath.Join(dir, "2026-03-01-append-target.md"))
	s := string(content)
	if !strings.Contains(s, "New appended content") {
		t.Error("appended content missing")
	}
	if !strings.Contains(s, "Original entry") {
		t.Error("original entry was lost")
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"How enforcement and comprehension relate", "enforcement-comprehension-relate"},
		{"Whether daemon capacity should be event-sourced", "daemon-capacity-event-sourced"},
		{"simple", "simple"},
		{"A B C", "b-c"},
		{"with---dashes", "dashes"},
		{"UPPERCASE Title", "uppercase-title"},
	}

	for _, tt := range tests {
		got := Slugify(tt.input)
		if got != tt.want {
			t.Errorf("Slugify(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseThread(t *testing.T) {
	content := `---
title: "Test parse"
status: open
created: 2026-03-01
updated: 2026-03-05
resolved_to: ""
---

# Test parse

## 2026-03-01

First entry.

## 2026-03-05

Second entry with multiple lines.
This is line two.
`
	thread, err := ParseThread(content)
	if err != nil {
		t.Fatalf("ParseThread failed: %v", err)
	}

	if thread.Title != "Test parse" {
		t.Errorf("title = %q, want 'Test parse'", thread.Title)
	}
	if thread.Status != StatusForming {
		t.Errorf("status = %q, want %q (normalized from 'open')", thread.Status, StatusForming)
	}
	if thread.Created != "2026-03-01" {
		t.Errorf("created = %q, want '2026-03-01'", thread.Created)
	}
	if thread.Updated != "2026-03-05" {
		t.Errorf("updated = %q, want '2026-03-05'", thread.Updated)
	}
	if len(thread.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(thread.Entries))
	}
	if thread.Entries[0].Date != "2026-03-01" {
		t.Errorf("first entry date = %q, want '2026-03-01'", thread.Entries[0].Date)
	}
	if !strings.Contains(thread.Entries[1].Text, "multiple lines") {
		t.Error("second entry missing content")
	}
}

func TestParseThread_RelationalFrontmatter(t *testing.T) {
	content := `---
title: "Relational test"
status: open
created: 2026-03-10
updated: 2026-03-22
resolved_to: ""
spawned_from: "coordination-protocol-primitives"
spawned:
  - "beads-atom-problem-work-molecules"
  - "constraint-first-design-orchestration-wrong"
active_work:
  - "orch-go-abc12"
  - "orch-go-def34"
resolved_by:
  - ".kb/decisions/2026-03-20-coord-primitives.md"
---

# Relational test

## 2026-03-10

First entry.
`
	thread, err := ParseThread(content)
	if err != nil {
		t.Fatalf("ParseThread failed: %v", err)
	}

	if thread.Title != "Relational test" {
		t.Errorf("title = %q, want 'Relational test'", thread.Title)
	}

	// spawned_from (scalar string)
	if thread.SpawnedFrom != "coordination-protocol-primitives" {
		t.Errorf("spawned_from = %q, want 'coordination-protocol-primitives'", thread.SpawnedFrom)
	}

	// spawned (list)
	if len(thread.Spawned) != 2 {
		t.Fatalf("spawned length = %d, want 2", len(thread.Spawned))
	}
	if thread.Spawned[0] != "beads-atom-problem-work-molecules" {
		t.Errorf("spawned[0] = %q", thread.Spawned[0])
	}
	if thread.Spawned[1] != "constraint-first-design-orchestration-wrong" {
		t.Errorf("spawned[1] = %q", thread.Spawned[1])
	}

	// active_work (list)
	if len(thread.ActiveWork) != 2 {
		t.Fatalf("active_work length = %d, want 2", len(thread.ActiveWork))
	}
	if thread.ActiveWork[0] != "orch-go-abc12" {
		t.Errorf("active_work[0] = %q", thread.ActiveWork[0])
	}
	if thread.ActiveWork[1] != "orch-go-def34" {
		t.Errorf("active_work[1] = %q", thread.ActiveWork[1])
	}

	// resolved_by (list)
	if len(thread.ResolvedBy) != 1 {
		t.Fatalf("resolved_by length = %d, want 1", len(thread.ResolvedBy))
	}
	if thread.ResolvedBy[0] != ".kb/decisions/2026-03-20-coord-primitives.md" {
		t.Errorf("resolved_by[0] = %q", thread.ResolvedBy[0])
	}

	// Entries still parse correctly
	if len(thread.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(thread.Entries))
	}
}

func TestParseThread_RelationalEmpty(t *testing.T) {
	// Relational fields absent — should default to empty
	content := `---
title: "No relations"
status: open
created: 2026-03-01
updated: 2026-03-01
resolved_to: ""
---

# No relations

## 2026-03-01

Entry.
`
	thread, err := ParseThread(content)
	if err != nil {
		t.Fatalf("ParseThread failed: %v", err)
	}

	if thread.SpawnedFrom != "" {
		t.Errorf("spawned_from should be empty, got %q", thread.SpawnedFrom)
	}
	if len(thread.Spawned) != 0 {
		t.Errorf("spawned should be empty, got %v", thread.Spawned)
	}
	if len(thread.ActiveWork) != 0 {
		t.Errorf("active_work should be empty, got %v", thread.ActiveWork)
	}
	if len(thread.ResolvedBy) != 0 {
		t.Errorf("resolved_by should be empty, got %v", thread.ResolvedBy)
	}
}

func TestParseThread_InlineYAMLList(t *testing.T) {
	// Test inline YAML list syntax: key: ["a", "b"]
	content := `---
title: "Inline list"
status: open
created: 2026-03-01
updated: 2026-03-01
resolved_to: ""
spawned: ["child-one", "child-two"]
active_work: []
---

# Inline list

## 2026-03-01

Entry.
`
	thread, err := ParseThread(content)
	if err != nil {
		t.Fatalf("ParseThread failed: %v", err)
	}

	if len(thread.Spawned) != 2 {
		t.Fatalf("spawned length = %d, want 2", len(thread.Spawned))
	}
	if thread.Spawned[0] != "child-one" {
		t.Errorf("spawned[0] = %q, want 'child-one'", thread.Spawned[0])
	}
	if thread.Spawned[1] != "child-two" {
		t.Errorf("spawned[1] = %q, want 'child-two'", thread.Spawned[1])
	}
	if len(thread.ActiveWork) != 0 {
		t.Errorf("active_work should be empty for [], got %v", thread.ActiveWork)
	}
}

// --- End-to-end behavioral verification: spawn/complete/orient loop ---

func TestSpawnCompleteOrientLoop(t *testing.T) {
	dir := tempThreadsDir(t)
	today := time.Now().Format("2006-01-02")

	// SPAWN: create thread and link work
	result, err := CreateOrAppend(dir, "Investigation into spawn behavior", "Spawning first agent to explore this")
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	if !result.Created {
		t.Fatal("expected new thread")
	}

	slug := Slugify("Investigation into spawn behavior")
	if err := LinkWork(dir, slug, "orch-go-abc12"); err != nil {
		t.Fatalf("LinkWork: %v", err)
	}

	// ORIENT: thread should appear in active threads
	active, err := ActiveThreads(dir, 7)
	if err != nil {
		t.Fatalf("ActiveThreads: %v", err)
	}
	if len(active) != 1 {
		t.Fatalf("orient: expected 1 active thread, got %d", len(active))
	}
	if active[0].Name != slug {
		t.Errorf("orient: expected slug %q, got %q", slug, active[0].Name)
	}

	// Verify work is linked
	shown, err := Show(dir, slug)
	if err != nil {
		t.Fatalf("Show: %v", err)
	}
	if len(shown.ActiveWork) != 1 || shown.ActiveWork[0] != "orch-go-abc12" {
		t.Errorf("expected active_work=[orch-go-abc12], got %v", shown.ActiveWork)
	}

	// COMPLETE: resolve the thread
	if err := Resolve(dir, slug, ".kb/decisions/spawn-behavior.md"); err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	// ORIENT again: thread should no longer appear
	active, err = ActiveThreads(dir, 7)
	if err != nil {
		t.Fatalf("ActiveThreads post-resolve: %v", err)
	}
	if len(active) != 0 {
		t.Errorf("orient post-resolve: expected 0 active threads, got %d", len(active))
	}

	// Verify resolved state
	shown, err = Show(dir, slug)
	if err != nil {
		t.Fatalf("Show post-resolve: %v", err)
	}
	if shown.Status != StatusResolved {
		t.Errorf("expected resolved, got %q", shown.Status)
	}
	if shown.ResolvedTo != ".kb/decisions/spawn-behavior.md" {
		t.Errorf("resolved_to = %q", shown.ResolvedTo)
	}

	// TodaysEntries should exclude resolved thread
	entries, err := TodaysEntries(dir, today)
	if err != nil {
		t.Fatalf("TodaysEntries: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("TodaysEntries should exclude resolved, got %d entries", len(entries))
	}
}

func TestParentChildSpawnLoop(t *testing.T) {
	dir := tempThreadsDir(t)
	today := time.Now().Format("2006-01-02")

	// Create parent thread
	_, err := CreateOrAppend(dir, "Coordination protocol design", "Starting exploration")
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}
	parentSlug := "coordination-protocol-design"

	// Orient: parent visible
	active, err := ActiveThreads(dir, 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 1 {
		t.Fatalf("expected 1 active, got %d", len(active))
	}

	// Spawn child from parent
	childResult, err := CreateWithParent(dir, "Sequencing primitive details", "Deep dive", parentSlug)
	if err != nil {
		t.Fatalf("CreateWithParent: %v", err)
	}
	if !childResult.Created {
		t.Fatal("expected child created")
	}
	childSlug := Slugify("Sequencing primitive details")

	// Link work to child
	if err := LinkWork(dir, childSlug, "orch-go-xyz99"); err != nil {
		t.Fatalf("LinkWork child: %v", err)
	}

	// Orient: both threads visible
	active, err = ActiveThreads(dir, 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 2 {
		t.Fatalf("expected 2 active threads, got %d", len(active))
	}

	// Verify parent-child relationship
	parent, _ := Show(dir, parentSlug)
	child, _ := Show(dir, childSlug)

	if child.SpawnedFrom != parentSlug {
		t.Errorf("child.SpawnedFrom = %q, want %q", child.SpawnedFrom, parentSlug)
	}
	foundChild := false
	for _, s := range parent.Spawned {
		if s == childSlug {
			foundChild = true
		}
	}
	if !foundChild {
		t.Errorf("parent.Spawned %v missing child %q", parent.Spawned, childSlug)
	}

	// Complete child
	if err := Resolve(dir, childSlug, ""); err != nil {
		t.Fatalf("resolve child: %v", err)
	}

	// Orient: only parent remains
	active, err = ActiveThreads(dir, 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 1 {
		t.Fatalf("expected 1 active after child resolve, got %d", len(active))
	}
	if active[0].Name != parentSlug {
		t.Errorf("expected parent %q still active, got %q", parentSlug, active[0].Name)
	}

	// TodaysEntries should only include parent
	entries, err := TodaysEntries(dir, today)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 today entry, got %d", len(entries))
	}
	if entries[0].ThreadName != parentSlug {
		t.Errorf("expected parent in today entries, got %q", entries[0].ThreadName)
	}
}

func TestMultiWorkSpawnTracking(t *testing.T) {
	dir := tempThreadsDir(t)

	_, err := CreateOrAppend(dir, "Multi agent investigation", "Starting parallel exploration")
	if err != nil {
		t.Fatal(err)
	}
	slug := "multi-agent-investigation"

	// Link multiple work items
	beads := []string{"orch-go-aaa11", "orch-go-bbb22", "orch-go-ccc33"}
	for _, b := range beads {
		if err := LinkWork(dir, slug, b); err != nil {
			t.Fatalf("LinkWork %s: %v", b, err)
		}
	}

	// Verify all work is tracked
	shown, err := Show(dir, slug)
	if err != nil {
		t.Fatal(err)
	}
	if len(shown.ActiveWork) != 3 {
		t.Fatalf("expected 3 active_work items, got %d: %v", len(shown.ActiveWork), shown.ActiveWork)
	}
	for i, b := range beads {
		if shown.ActiveWork[i] != b {
			t.Errorf("active_work[%d] = %q, want %q", i, shown.ActiveWork[i], b)
		}
	}

	// Orient still surfaces as active
	active, err := ActiveThreads(dir, 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 1 {
		t.Fatalf("expected 1 active thread, got %d", len(active))
	}

	// Resolve and verify orient drops it
	Resolve(dir, slug, "")
	active, _ = ActiveThreads(dir, 7)
	if len(active) != 0 {
		t.Errorf("expected 0 active after resolve, got %d", len(active))
	}
}

func TestBackPropCompleteOrientLoop(t *testing.T) {
	dir := tempThreadsDir(t)

	// Create thread and link work
	_, err := CreateOrAppend(dir, "Backprop verification thread", "Testing completion back-propagation")
	if err != nil {
		t.Fatal(err)
	}
	slug := "backprop-verification-thread"

	if err := LinkWork(dir, slug, "orch-go-bp001"); err != nil {
		t.Fatalf("LinkWork: %v", err)
	}

	// Orient: thread is active with linked work
	active, err := ActiveThreads(dir, 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 1 {
		t.Fatalf("expected 1 active, got %d", len(active))
	}

	// BackPropagateCompletion simulates orch complete removing work
	results, err := BackPropagateCompletion(dir, "orch-go-bp001")
	if err != nil {
		t.Fatalf("BackPropagateCompletion: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 backprop result, got %d", len(results))
	}
	if results[0].Slug != slug {
		t.Errorf("backprop slug = %q, want %q", results[0].Slug, slug)
	}

	// Verify thread state: active_work empty, resolved_by populated
	shown, err := Show(dir, slug)
	if err != nil {
		t.Fatal(err)
	}
	if len(shown.ActiveWork) != 0 {
		t.Errorf("expected empty active_work after backprop, got %v", shown.ActiveWork)
	}
	if len(shown.ResolvedBy) != 1 || shown.ResolvedBy[0] != "orch-go-bp001" {
		t.Errorf("expected resolved_by=[orch-go-bp001], got %v", shown.ResolvedBy)
	}

	// Thread is still active (not resolved) — only the work item completed
	if shown.Status != StatusForming {
		t.Errorf("thread status should still be forming, got %q", shown.Status)
	}

	// Orient still surfaces it (thread is active, just the beads work completed)
	active, err = ActiveThreads(dir, 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 1 {
		t.Fatalf("thread should remain active after backprop, got %d", len(active))
	}
}

func TestAppendThenOrientShowsUpdatedEntry(t *testing.T) {
	dir := tempThreadsDir(t)

	_, err := CreateOrAppend(dir, "Evolving understanding", "Initial thought")
	if err != nil {
		t.Fatal(err)
	}
	slug := "evolving-understanding"

	// Append deeper insight
	_, err = Append(dir, slug, "Deeper insight after more investigation")
	if err != nil {
		t.Fatalf("Append: %v", err)
	}

	// Orient should show updated latest entry
	active, err := ActiveThreads(dir, 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 1 {
		t.Fatalf("expected 1 active, got %d", len(active))
	}
	if !strings.Contains(active[0].LatestEntry, "Deeper insight") {
		t.Errorf("latest entry = %q, want to contain 'Deeper insight'", active[0].LatestEntry)
	}
}
