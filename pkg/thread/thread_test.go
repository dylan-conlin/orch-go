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
	if !strings.Contains(s, "status: open") {
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
	if threads[0].Status != "open" {
		t.Errorf("expected open status, got %q", threads[0].Status)
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
	if thread.Status != "open" {
		t.Errorf("status = %q, want 'open'", thread.Status)
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
