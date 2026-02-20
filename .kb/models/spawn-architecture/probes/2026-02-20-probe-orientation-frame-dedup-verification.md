# Probe: ORIENTATION_FRAME Dedup Verification

**Date:** 2026-02-20
**Status:** Complete
**Model:** spawn-architecture (confirmatory)

---

## Question

Is the ORIENTATION_FRAME duplication issue (orch-go-1135) already fixed by the prior work in orch-go-1130?

---

## What I Tested

### 1. Verified Fix Commit Exists

```bash
git log --format="%ci %s" 3fc539bb4 -1
```

**Output:**
```
2026-02-20 09:04:24 -0800 fix: remove ORIENTATION_FRAME from SPAWN_CONTEXT.md template (orch-go-1130)
```

### 2. Verified Template No Longer Has ORIENTATION_FRAME Section

```bash
grep -n "ORIENTATION_FRAME" pkg/spawn/context.go
```

**Output:** No matches found (section was removed)

### 3. Verified New Spawns Don't Have Duplication

Checked workspace created AFTER fix:
```bash
grep -c "ORIENTATION_FRAME" .orch/workspace/og-arch-opencode-mcp-hot-20feb-9333/SPAWN_CONTEXT.md
```

**Output:** 0 occurrences

### 4. Confirmed Audit Evidence Was from BEFORE Fix

The audit (orch-go-1132) that reported 100% duplication examined workspaces from Feb 19, 2026. Example workspace `pw-debug-fix-verification-run-19feb-fffe` was created on Feb 19 16:24 — before the fix commit on Feb 20 09:04.

### 5. Verified FRAME Beads Comment Still Works

Checked extraction.go:941-948 — FRAME comment is still recorded in beads for orchestrator completion review (this is correct behavior, not duplication).

---

## What I Observed

1. **Fix already implemented:** orch-go-1130 removed ORIENTATION_FRAME from the SPAWN_CONTEXT.md template on Feb 20, 2026 at 09:04
2. **Template is clean:** No ORIENTATION_FRAME section in SpawnContextTemplate
3. **New spawns verified:** Workspaces created after the fix have 0 ORIENTATION_FRAME references
4. **Audit timing mismatch:** orch-go-1135 was created based on audit evidence from BEFORE the fix was deployed
5. **FRAME comment preserved:** The beads FRAME: comment is still recorded at spawn time (lines 941-948) for orchestrator use during completion review — this is the correct location

---

## Model Impact

**Verdict:** confirms

**Finding:** The ORIENTATION_FRAME duplication issue reported in orch-go-1135 was already fixed by orch-go-1130. The audit (orch-go-1132) that identified the issue examined historical workspaces created before the fix. No code changes needed — issue should be closed as duplicate/already-fixed.

**Invariant verified:** ORIENTATION_FRAME belongs only in beads comments (for orchestrator completion review), not in SPAWN_CONTEXT.md (for worker context). This invariant is now correctly implemented.

---

## Recommendation

Close orch-go-1135 as **already fixed** by orch-go-1130. The audit evidence was from pre-fix workspaces.
