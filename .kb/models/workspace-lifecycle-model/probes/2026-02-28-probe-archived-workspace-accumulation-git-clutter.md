# Probe: Archived Workspace Accumulation and Git Status Clutter

**Model:** Workspace Lifecycle & Hierarchy
**Date:** 2026-02-28
**Status:** Complete

---

## Question

The model states the lifecycle flow is Spawn → Execute → Complete → Archive, and identifies "The Archival Gap" as a failure mode. Automated archival (Jan 17, 2026) closed that gap. But has archival itself created a new accumulation problem? Specifically: do archived workspaces accumulate unboundedly, and does this cause practical friction?

## What I Tested

### Test 1: Scale of archived workspace accumulation

```bash
ls -d .orch/workspace/archived/*/ | wc -l
# Result: 1,708 directories

du -sh .orch/workspace/archived/
# Result: 149MB
```

### Test 2: Git status impact

```bash
git status --porcelain .orch/workspace/archived/ | wc -l
# Result: All 1,708 dirs appear as untracked (??) in git status
```

### Test 3: Whether .gitignore covers workspaces

```bash
grep '\.orch' .gitignore
# Result: No .orch entries in .gitignore at all
```

### Test 4: Already-tracked workspace files in git

```bash
git ls-files .orch/workspace/ | wc -l
# Result: 10,448 files already tracked from historical commits

git ls-files .orch/workspace/archived/ | wc -l
# Result: 9,677 tracked files in archived

git ls-files .orch/workspace/ | grep -v archived | wc -l
# Result: 771 tracked files in active workspace dirs
```

### Test 5: Consumers of archived workspaces

Code review identified two real consumers:

1. **`FindArchivedWorkspaceByBeadsID()`** in `pkg/spawn/rework.go:19-66`
   - Used by `orch rework` to find prior work for rework agent spawns
   - Extracts SYNTHESIS.md TLDR + Delta sections for rework context
   - This is a genuine, valuable use case

2. **`review_orphans.go:240-350`**
   - Fallback search: checks archived for SYNTHESIS.md when active workspace not found
   - Secondary use case for orphan review reporting

### Test 6: Producers of archived workspaces

1. **`archiveWorkspace()`** in `complete_actions.go:17-52`
   - Called via `LifecycleManager.Archive()` during `orch complete`
   - Moves active → archived on completion

2. **`archiveStaleWorkspaces()`** in `clean_cmd.go:816-976`
   - Called by `orch clean --workspaces` for workspaces >7 days old

## What I Observed

1. **The model's "Archival Gap" was fixed, but created a new gap:** Archived workspaces now accumulate unboundedly. The model doesn't address archived workspace lifecycle at all — it treats archival as the terminal state.

2. **149MB and 1,708 dirs is material.** While not a disk space emergency, the git status noise is the real pain: every `git status` shows hundreds of untracked entries, making it hard to see actual code changes.

3. **10,448 already-tracked workspace files in git** is a historical artifact from when workspaces were committed. This is a separate (larger) cleanup concern.

4. **Rework is the binding constraint.** `FindArchivedWorkspaceByBeadsID()` needs archived workspaces to exist. Deleting instead of archiving would break rework. But rework typically happens within days, not months.

5. **No TTL or cleanup exists for archived workspaces.** Once archived, they persist forever. `orch clean --workspaces` archives stale active workspaces but has no mechanism to clean up stale archived workspaces.

## Model Impact

### Extends: Lifecycle flow needs a fifth stage

The model's lifecycle is: Spawn → Execute → Complete → Archive

**Should be:** Spawn → Execute → Complete → Archive → Expire (TTL-based deletion)

The model identifies archival as the terminal state, but in practice archived workspaces need their own lifecycle. The "two-tier cleanup" decision (2026-01-14) already establishes the pattern: event-based cleanup (archival on complete) + periodic background cleanup (TTL-based expiry of archived workspaces).

### Confirms: "Workspaces are visible to git status (unless ignored)"

The model's constraint section notes: "Worker workspaces are inside the project they are modifying → Workspaces are visible to `git status` (unless ignored)." The probe confirms this is exactly the problem — they were never ignored.

### Extends: Git tracking of workspaces is a dual problem

The model doesn't address that workspaces should never be committed to git. There are 10,448 tracked workspace files from historical commits, plus 1,708+ untracked archived dirs. Both need addressing:
- Immediate: `.gitignore` entry prevents new clutter
- Follow-up: `git rm --cached` to untrack historical files
