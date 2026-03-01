# Design: Fix Archived Workspace Git Clutter and Unbounded Growth

**Date:** 2026-02-28
**Phase:** Complete
**Trigger:** 1,708 archived workspace dirs (149MB) cluttering git status as untracked files
**Beads:** orch-go-u5jo

---

## Design Question

Archived agent workspaces in `.orch/workspace/archived/` accumulate unboundedly as untracked files, cluttering `git status` (currently 1,708 dirs). What is the right fix?

## Problem Framing

**Success Criteria:**
- `git status` no longer shows workspace directories as untracked
- Archived workspaces don't grow unboundedly
- `orch rework` continues to work (requires access to recent archived workspaces)
- Minimal code changes; follows established patterns

**Constraints:**
- `FindArchivedWorkspaceByBeadsID()` is a real consumer — cannot delete instead of archiving
- 10,448 workspace files already tracked in git from historical commits (separate concern)
- Must follow "two-tier cleanup" pattern (decision: 2026-01-14)
- Must follow "No Local Agent State" constraint (workspaces are raw artifacts, not derived state — this is fine)

**Scope:**
- IN: gitignore for workspaces, TTL cleanup design for archived
- OUT: Removing 10,448 historically-tracked workspace files from git (follow-up task)
- OUT: Changes to workspace lifecycle model (separate model update)

---

## Exploration

### Fork 1: How to fix git status noise?

**Options:**
- A: Gitignore `.orch/workspace/archived/` only
- B: Gitignore `.orch/workspace/` entirely
- C: Move archived outside the repo (`~/.orch/archived/`)

**Substrate says:**
- Model: "Worker workspaces are inside the project they are modifying → Workspaces are visible to `git status` (unless ignored)"
- Model: "Keeps evidence close to the code" is the rationale for project-local workspaces
- Decision (ghost-visibility): "Filter at query time, not cleanup" — but this is about display, not about git tracking

**Recommendation:** Option B — gitignore `.orch/workspace/` entirely.

Both active and archived workspaces show up as untracked in git status. Neither should ever be committed. Active workspaces are local development state (spawned agents' working directories). Archived workspaces are completed agent records. The `.orch/workspace/` directory is already fully covered by the "evidence close to code" rationale — it doesn't need to be in git to be local.

Note: `.gitignore` only affects untracked files. The 10,448 already-tracked files will continue to be tracked and won't be affected by this change.

**Trade-off:** Option A is more targeted but leaves active workspaces cluttering status. Option C adds complexity for marginal benefit (code changes to rework.go, review_orphans.go; moves evidence away from code).

### Fork 2: Should we delete instead of archiving?

**Options:**
- A: Keep archiving (status quo behavior, just fix visibility)
- B: Delete on completion instead of archiving
- C: Archive with TTL, then delete expired

**Substrate says:**
- `FindArchivedWorkspaceByBeadsID()` is used by `orch rework` — real dependency
- Rework extracts SYNTHESIS.md TLDR + Delta sections from prior workspace
- Two-tier cleanup decision: event-based + periodic background cleanup
- Model: Archive is the terminal lifecycle state (needs extending)

**Recommendation:** Option C — archive with TTL-based expiry.

Deleting immediately (Option B) breaks rework. Keeping forever (Option A) is the current problem. TTL-based expiry balances both: archived workspaces persist long enough for rework (which happens within days), then expire to bound growth.

### Fork 3: What should the TTL be?

**Options:**
- A: 7 days (matches stale workspace threshold)
- B: 30 days (generous buffer for rework)
- C: 90 days (very conservative)

**Substrate says:**
- Rework typically happens within hours to days of completion
- After 30 days, SYNTHESIS.md content should be promoted to `.kb/` artifacts
- Disk usage is 149MB for 1,708 dirs — growth rate is ~50 dirs/day at current pace

**Recommendation:** Option B — 30-day TTL.

7 days is too aggressive (some rework happens after a week). 90 days provides little benefit over 30. At 50 dirs/day, a 30-day TTL caps the archive at ~1,500 dirs (similar to current size but self-regulating).

### Fork 4: Where should TTL cleanup live?

**Options:**
- A: New `orch clean --archived-ttl` flag
- B: Extend existing `orch clean --workspaces` to also clean expired archives
- C: Daemon periodic task

**Substrate says:**
- Two-tier cleanup decision: Tier 1 (event-based) + Tier 2 (periodic background)
- `orch clean --workspaces` already archives stale active workspaces
- Daemon already runs periodic tasks

**Recommendation:** Option B + C — extend `orch clean --workspaces` to also expire old archived workspaces (Tier 1 manual), and add daemon periodic cleanup (Tier 2 automatic).

This follows the established two-tier pattern exactly. `orch clean --workspaces` becomes the full workspace lifecycle manager: archive stale active workspaces AND delete expired archived workspaces.

### Fork 5: What about the 10,448 already-tracked workspace files?

**Options:**
- A: Remove from git tracking now (`git rm --cached -r .orch/workspace/`)
- B: Separate follow-up task
- C: Leave them (tracked files don't clutter status)

**Substrate says:**
- This is a large change (10,448 files in one commit)
- Tracked files don't show in `git status` unless modified
- Risk of disrupting other agents' work if done concurrently

**Recommendation:** Option B — create follow-up task. The gitignore fix addresses the immediate pain (untracked files). The tracked file cleanup is a separate, larger operation that should have its own issue.

---

## Synthesis

### Recommended Approach (Two Changes)

**Change 1 (Immediate): Gitignore workspaces**
- Add `.orch/workspace/` to `.gitignore`
- Immediately fixes git status clutter for all untracked workspace dirs
- Zero code changes, zero risk to consumers

**Change 2 (Follow-up implementation): TTL-based archived workspace expiry**
- Extend `orch clean --workspaces` to delete archived workspaces older than 30 days
- Add `--archived-ttl` flag (default 30 days) for configurability
- Add daemon periodic task to run this cleanup automatically
- Pattern: scan `.orch/workspace/archived/*/`, read `.spawn_time`, delete if > TTL

### Implementation-Ready Specification

**Change 1 — Gitignore (.gitignore modification)**

File target: `.gitignore`

Add:
```
# Agent workspaces (local development state, never committed)
.orch/workspace/
```

Acceptance criteria:
- `git status` no longer shows workspace directories as untracked
- `FindArchivedWorkspaceByBeadsID()` still works (reads filesystem, not git)
- `review_orphans.go` archived search still works

**Change 2 — TTL Cleanup (code changes)**

File targets:
- `cmd/orch/clean_cmd.go` — add `cleanExpiredArchives()` function
- `pkg/daemon/daemon.go` — add periodic cleanup task (or extend existing cleanup)

Logic:
```
func cleanExpiredArchives(projectDir string, ttlDays int, dryRun bool) (int, error) {
    archivedDir := filepath.Join(projectDir, ".orch", "workspace", "archived")
    entries, _ := os.ReadDir(archivedDir)
    cutoff := time.Now().AddDate(0, 0, -ttlDays)

    for each entry:
        read .spawn_time (or fall back to dir modtime)
        if older than cutoff:
            if dryRun: log
            else: os.RemoveAll(entry path)

    return deleted count, nil
}
```

Acceptance criteria:
- `orch clean --workspaces` archives stale active AND deletes expired archived
- `--archived-ttl N` flag allows configuring TTL (default 30)
- `--dry-run` shows what would be deleted
- Daemon runs this cleanup periodically (e.g., every 6 hours, matching session cleanup cadence)

**Follow-up Task — Untrack historical workspace files**
- `git rm --cached -r .orch/workspace/` removes 10,448 files from git index
- Combined with gitignore entry, these files stop being tracked
- Large commit — should be its own task with clear commit message

### Out of Scope
- Changing where workspaces are stored (they stay project-local)
- Changing the archival mechanism itself (it works correctly)
- Modifying rework behavior
- UI/dashboard changes

---

## Decision Gate Guidance (if promoting to decision)

**Add blocks: frontmatter when promoting:**
- This resolves recurring workspace accumulation issues (3+ prior investigations)
- Future spawns working on workspace cleanup should see this

**Suggested blocks keywords:**
- archived workspaces
- workspace cleanup
- git status clutter
- workspace gitignore

---

## Recommendations

**RECOMMENDED:** Gitignore + TTL expiry (two-part approach)
- **Why:** Addresses both symptoms (git clutter) and root cause (unbounded growth) with minimal risk
- **Trade-off:** 30-day TTL means very old rework attempts may not find archived workspace — acceptable because rework should happen promptly
- **Expected outcome:** Git status clean immediately; archived dir self-regulates to ~1,500 dirs max

**Alternative: Gitignore only (no TTL)**
- **Pros:** Simplest possible fix, zero code changes
- **Cons:** Disk growth continues unchecked (149MB now, ~50MB/month at current pace)
- **When to choose:** If disk space is never a concern and you prefer minimal changes

**Alternative: Delete instead of archive**
- **Pros:** No accumulation at all
- **Cons:** Breaks `orch rework` — a real, valuable feature
- **When to choose:** Never, unless rework is deprecated

**Alternative: Move archived outside repo**
- **Pros:** Completely isolates from git
- **Cons:** Requires code changes to rework.go, review_orphans.go; moves evidence away from code
- **When to choose:** If multi-project archive aggregation becomes needed
