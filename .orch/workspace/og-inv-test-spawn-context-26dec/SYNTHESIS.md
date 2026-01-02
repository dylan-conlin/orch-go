# Session Synthesis

**Agent:** og-inv-test-spawn-context-26dec
**Issue:** orch-go-untracked-1766774790 (synthetic - spawn was --no-track)
**Duration:** 2025-12-26 10:46 → 2025-12-26 10:55
**Outcome:** success

---

## TLDR

Investigated spawn context template for correctness. Found a bug: beads tracking instructions (`bd comment`) are included unconditionally, causing failures for untracked spawns where the beads issue doesn't exist.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-26-inv-test-spawn-context.md` - Investigation documenting the bug

### Files Modified
- None

### Commits
- (pending) Investigation file commit

---

## Evidence (What Was Observed)

- `bd comment orch-go-untracked-1766774790 "..."` returns "issue not found" (tested)
- `bd show orch-go-untracked-1766774790` returns "no issue found" (tested)
- `pkg/spawn/context.go:16-196` - SpawnContextTemplate has no conditional for untracked BeadsIDs
- `cmd/orch/main.go:1672` - Untracked ID format: `{project}-untracked-{timestamp}`
- Spawn context contains beads instructions in multiple locations (lines 12, 23, 50, 94, etc.)

### Tests Run
```bash
# Test beads comment - FAILS
bd comment orch-go-untracked-1766774790 "Phase: Planning - test"
# Error adding comment: operation failed: failed to add comment: issue orch-go-untracked-1766774790 not found

# Test beads show - FAILS
bd show orch-go-untracked-1766774790
# Error resolving ID orch-go-untracked-1766774790: operation failed: failed to resolve ID: no issue found matching "orch-go-untracked-1766774790"
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-26-inv-test-spawn-context.md` - Documents bug and recommended fix

### Decisions Made
- None (bug found, fix requires implementation decision)

### Constraints Discovered
- Untracked BeadsIDs (containing "-untracked-") are intentionally not created in beads - they're internal tracking only
- Spawn context template must conditionally handle these IDs

### Externalized via `kn`
- None required - findings captured in investigation artifact

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** Spawn context includes invalid beads instructions for untracked spawns
**Skill:** feature-impl
**Context:**
```
Bug: SpawnContextTemplate in pkg/spawn/context.go includes `bd comment` instructions unconditionally.
For --no-track spawns, the BeadsID is synthetic (e.g., "orch-go-untracked-1766774790") and doesn't exist in beads.
Fix: Add IsUntracked field to contextData, set when BeadsID contains "-untracked-", wrap beads instructions in {{if not .IsUntracked}}.
See investigation: .kb/investigations/2025-12-26-inv-test-spawn-context.md
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- What instructions should untracked agents receive instead of beads tracking? (e.g., "workspace-only status tracking")
- Should untracked spawns be prevented from using investigation skill? (they can't report progress)

**Areas worth exploring further:**
- Whether --no-track should emit a warning that beads tracking is disabled
- Whether the tier system should interact differently with --no-track

**What remains unclear:**
- Design intent: is it acceptable that untracked spawns lose all phase monitoring capability?

---

## Session Metadata

**Skill:** investigation
**Model:** claude
**Workspace:** `.orch/workspace/og-inv-test-spawn-context-26dec/`
**Investigation:** `.kb/investigations/2025-12-26-inv-test-spawn-context.md`
**Beads:** (untracked - no beads issue)
