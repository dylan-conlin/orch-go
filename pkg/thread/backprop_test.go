package thread

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBackPropagateCompletion(t *testing.T) {
	dir := tempThreadsDir(t)
	today := time.Now().Format("2006-01-02")

	// Thread with matching beads ID in active_work
	thread1 := fmt.Sprintf(`---
title: "Thread with active work"
status: open
created: %s
updated: %s
resolved_to: ""
active_work:
  - "orch-go-abc12"
  - "orch-go-def34"
resolved_by:
  - ".kb/decisions/existing.md"
---

# Thread with active work

## %s

Working on this.
`, today, today, today)
	os.WriteFile(filepath.Join(dir, today+"-thread-active-work.md"), []byte(thread1), 0644)

	// Thread without matching beads ID
	thread2 := fmt.Sprintf(`---
title: "Unrelated thread"
status: open
created: %s
updated: %s
resolved_to: ""
active_work:
  - "orch-go-xyz99"
---

# Unrelated thread

## %s

Different work.
`, today, today, today)
	os.WriteFile(filepath.Join(dir, today+"-unrelated-thread.md"), []byte(thread2), 0644)

	// Thread with no active_work
	thread3 := fmt.Sprintf(`---
title: "No active work"
status: open
created: %s
updated: %s
resolved_to: ""
---

# No active work

## %s

Just thinking.
`, today, today, today)
	os.WriteFile(filepath.Join(dir, today+"-no-active-work.md"), []byte(thread3), 0644)

	// Run back-propagation for orch-go-abc12
	results, err := BackPropagateCompletion(dir, "orch-go-abc12")
	if err != nil {
		t.Fatalf("BackPropagateCompletion failed: %v", err)
	}

	// Should have updated exactly 1 thread
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Slug != "thread-active-work" {
		t.Errorf("expected slug 'thread-active-work', got %q", results[0].Slug)
	}

	// Re-read the updated thread and verify
	data, err := os.ReadFile(filepath.Join(dir, today+"-thread-active-work.md"))
	if err != nil {
		t.Fatal(err)
	}
	updated, err := ParseThread(string(data))
	if err != nil {
		t.Fatalf("ParseThread failed: %v", err)
	}

	// active_work should only have orch-go-def34 (abc12 removed)
	if len(updated.ActiveWork) != 1 {
		t.Fatalf("expected 1 active_work item, got %d: %v", len(updated.ActiveWork), updated.ActiveWork)
	}
	if updated.ActiveWork[0] != "orch-go-def34" {
		t.Errorf("active_work[0] = %q, want 'orch-go-def34'", updated.ActiveWork[0])
	}

	// resolved_by should have existing + new
	if len(updated.ResolvedBy) != 2 {
		t.Fatalf("expected 2 resolved_by items, got %d: %v", len(updated.ResolvedBy), updated.ResolvedBy)
	}
	if updated.ResolvedBy[0] != ".kb/decisions/existing.md" {
		t.Errorf("resolved_by[0] = %q", updated.ResolvedBy[0])
	}
	if updated.ResolvedBy[1] != "orch-go-abc12" {
		t.Errorf("resolved_by[1] = %q, want 'orch-go-abc12'", updated.ResolvedBy[1])
	}

	// Unrelated thread should be untouched
	data2, _ := os.ReadFile(filepath.Join(dir, today+"-unrelated-thread.md"))
	unrelated, _ := ParseThread(string(data2))
	if len(unrelated.ActiveWork) != 1 || unrelated.ActiveWork[0] != "orch-go-xyz99" {
		t.Errorf("unrelated thread active_work modified: %v", unrelated.ActiveWork)
	}
}

func TestBackPropagateCompletion_NoMatch(t *testing.T) {
	dir := tempThreadsDir(t)
	today := time.Now().Format("2006-01-02")

	thread := fmt.Sprintf(`---
title: "No match"
status: open
created: %s
updated: %s
resolved_to: ""
active_work:
  - "orch-go-other"
---

# No match

## %s

Entry.
`, today, today, today)
	os.WriteFile(filepath.Join(dir, today+"-no-match.md"), []byte(thread), 0644)

	results, err := BackPropagateCompletion(dir, "orch-go-nonexistent")
	if err != nil {
		t.Fatalf("BackPropagateCompletion failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestBackPropagateCompletion_EmptyDir(t *testing.T) {
	dir := tempThreadsDir(t)

	results, err := BackPropagateCompletion(dir, "orch-go-abc12")
	if err != nil {
		t.Fatalf("BackPropagateCompletion failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}
