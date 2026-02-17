# Probe: bd sync pre-commit hook deadlock (second deadlock path)

**Model:** beads-integration-architecture
**Date:** 2026-02-17
**Status:** Active

---

## Question

The beads-integration-architecture model describes `bd sync` integration with git.
A deadlock fix was shipped in beads commit c2af5a82 (importFromJSONL subprocess acquiring same flock).
Yet `bd sync` still hangs repeatedly (2 occurrences in 24 hours). Does the deadlock fix fully resolve all flock contention paths, or are there additional deadlock paths?

---

## What I Tested

### 1. Verified patched binary

```bash
$ bd version
bd version 0.41.0 (dev: master@f9c1bc9f0af3)
# f9c1bc9f is one commit AFTER c2af5a82 deadlock fix
# Binary date: Feb 16 15:18:55 2026 (post-fix)
```

### 2. Checked for zombie processes

```bash
$ ps -eo pid,ppid,pgid,stat,lstart,command | grep "bd " | grep -v grep
# Found 28+ hanging bd processes including:
# PID 34059: bd sync (orch-go) since Mon Feb 16 22:02:48 — 12+ hours hung
# PID 35984: bd sync (price-watch) since Tue Feb 17 10:01:18
# Multiple bd sync --flush-only, bd comments, bd list, bd ready all hanging
```

### 3. Inspected process state with lsof

```bash
$ lsof -p 34059
# fd 5: holds .beads/jsonl.lock (exclusive flock)
# fd 3,4: Go runtime self-pipe
# fd 8: pipe to subprocess (no child process exists)
# PPID=1 (orphaned — parent process died)
```

### 4. Verified git is NOT the cause

```bash
$ timeout 5 git pull --rebase --dry-run
error: cannot pull with rebase: You have unstaged changes.
error: Please commit or stash them.
exit: 128
# Git fails FAST (not a hang). Exit code 128.
```

### 5. Reproduced the hang

```bash
$ timeout 10 bd sync
→ Exporting pending changes to JSONL...
→ Committing changes to git...
# HANGS HERE — never returns
# New processes spawned:
# PID 35918: bd sync (parent, holds jsonl.lock)
# PID 36326: bd sync --flush-only (child, blocked on jsonl.lock)
```

### 6. Confirmed deadlock chain via pre-commit hook

```bash
$ cat .git/hooks/pre-commit
# Line 24: exec bd hooks run pre-commit "$@"
```

Beads source (hooks.go:428):
```go
// runPreCommitHook() calls:
cmd := exec.Command("bd", "sync", "--flush-only")
cmd.Run()  // This subprocess acquires jsonl.lock → DEADLOCK
```

### 7. Checked for stale git lock files

```bash
$ ls -la .git/next-index-*.lock
# next-index-14337.lock from Jan 22 (stale)
# next-index-34125.lock from Feb 16 (stale, from zombie bd sync)
# These are harmless temp files, NOT the cause of hangs
```

---

## What I Observed

**The deadlock chain (100% reproducible):**

```
bd sync (PID A)
  → initializeNoDbMode() → acquires exclusive flock on .beads/jsonl.lock
  → exportToJSONL() → success
  → git add .beads/ → success
  → git commit → triggers pre-commit hook
    → pre-commit hook → exec bd hooks run pre-commit
      → runPreCommitHook() → exec.Command("bd", "sync", "--flush-only").Run()
        → bd sync --flush-only (PID B)
          → initializeNoDbMode() → FlockExclusiveBlocking(.beads/jsonl.lock) → BLOCKS
    → pre-commit hook blocks waiting for flush-only
  → git commit blocks waiting for pre-commit hook
→ bd sync (PID A) blocks waiting for git commit
→ DEADLOCK: A holds lock, waits for B. B needs lock held by A.
```

**Secondary effects observed:**
- Zombie `bd sync` processes hold jsonl.lock indefinitely after parent orphans them
- All subsequent `bd` commands in the same project may block on the same lock
- `orch status` times out because it calls `bd list/ready/stats` which block on the lock
- Stale `next-index-*.lock` files accumulate from interrupted git operations

**Scope of the bug:**
- Affects ALL projects using noDb (JSONL-only) mode with beads git hooks installed
- Reproducible on orch-go and price-watch (confirmed on both)
- Every `bd sync` call that has uncommitted changes in .beads/ will deadlock
- The pre-commit hook's flush is REDUNDANT when called from within `bd sync` (sync already did the export)

---

## Model Impact

- [ ] **Confirms** invariant: The beads-integration model correctly identifies flock contention as a critical concern
- [x] **Contradicts** invariant: The c2af5a82 deadlock fix does NOT fully resolve all flock deadlock paths — a second deadlock via pre-commit hook exists
- [x] **Extends** model with: Second deadlock path: `bd sync` → `git commit` → pre-commit hook → `bd sync --flush-only` → flock contention. The fix for this should be one of:
  1. `bd sync` should use `git commit --no-verify` for its internal commits (flush already done)
  2. `bd sync` should set `BD_SYNC_IN_PROGRESS=1` env var, and `bd hooks run pre-commit` should skip flush when set
  3. `bd sync --flush-only` should use non-blocking flock and skip if already held
  4. `PersistentPreRun` should detect reentrant lock acquisition and skip

---

## Notes

- The deadlock fix c2af5a82 only addressed the `importFromJSONL()` → subprocess path
- This bug is in beads (not orch-go) but blocks ALL orch-go operations that touch beads
- Immediate workaround: kill zombie `bd sync` processes and use `git commit --no-verify` manually
- Long-term fix should be in beads repo
- The pre-commit hook's `cmd.Run()` ignores errors (`Don't block the commit`), but it DOES block because the subprocess hangs rather than erroring
- Process accumulation: each hang creates 2+ zombie processes (bd sync + bd sync --flush-only), compounding over time
