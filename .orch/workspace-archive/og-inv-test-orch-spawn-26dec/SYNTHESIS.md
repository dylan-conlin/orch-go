# Session Synthesis

**Agent:** og-inv-test-orch-spawn-26dec
**Issue:** orch-go-untracked-1766774810 (synthetic/fake ID - see findings)
**Duration:** ~30 minutes
**Outcome:** success

---

## TLDR

Investigated orch spawn context; discovered bug where untracked spawns (`--no-track`) include beads instructions (`bd comment`) that fail because the BeadsID is synthetic and doesn't exist in beads. Fix: conditionally skip beads instructions for untracked spawns.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-26-inv-test-orch-spawn-context.md` - Full investigation with D.E.K.N. summary, findings, and recommendations

### Files Modified
- None

### Commits
- (Pending) Investigation file commit

---

## Evidence (What Was Observed)

- Running `bd comment orch-go-untracked-1766774810 "..."` fails with "issue not found" error
- Spawn context template at `pkg/spawn/context.go:32-39` unconditionally includes beads instructions
- Detection function `isUntrackedBeadsID` exists at `cmd/orch/review.go:271-274` but isn't used during spawn context generation
- Untracked pattern format: `{project}-untracked-{timestamp}` (e.g., `orch-go-untracked-1766774810`)

### Tests Run
```bash
# Verify untracked issue doesn't exist
bd show orch-go-untracked-1766774810
# Error: no issue found matching "orch-go-untracked-1766774810"

# Confirm comment fails
bd comment orch-go-untracked-1766774810 "Phase: Planning - test"
# Error: issue not found
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-26-inv-test-orch-spawn-context.md` - Documents the bug and proposed fix

### Decisions Made
- Recommended fix: Conditional template rendering (add `IsUntracked` field to `contextData`, skip beads instructions when true)

### Constraints Discovered
- Untracked spawns cannot use beads for progress tracking
- Spawn context template assumes all BeadsIDs are valid

### Externalized via `kn`
- N/A - findings are in investigation artifact

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** Spawn context includes beads instructions for untracked spawns (causes failure)
**Skill:** feature-impl
**Context:**
```
Bug: Untracked spawns (--no-track or beads failure) have synthetic BeadsIDs that cause bd comment to fail.
Fix: Add IsUntracked bool to contextData in pkg/spawn/context.go, detect using strings.Contains(BeadsID, "-untracked-"),
conditionally skip beads instructions in template. See .kb/investigations/2025-12-26-inv-test-orch-spawn-context.md for details.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How often does beads issue creation fail silently? (Could be masking bugs)
- What should untracked agents do for progress reporting instead of bd comment?
- Should `orch complete` handle untracked agents differently?

**Areas worth exploring further:**
- Whether skill self-review checklists need updates for untracked spawns
- Whether investigation file creation should be conditional for untracked

**What remains unclear:**
- Was this specific spawn intentionally untracked or did beads creation fail?

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-inv-test-orch-spawn-26dec/`
**Investigation:** `.kb/investigations/2025-12-26-inv-test-orch-spawn-context.md`
**Beads:** `bd show orch-go-untracked-1766774810` (synthetic ID - doesn't exist)
