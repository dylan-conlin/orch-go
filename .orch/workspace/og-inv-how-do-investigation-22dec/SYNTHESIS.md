# Session Synthesis

**Agent:** og-inv-how-do-investigation-22dec
**Issue:** orch-go-6akg
**Duration:** 2025-12-22 → 2025-12-22
**Outcome:** success

---

## TLDR

Root cause analysis of stale investigation file `2025-12-22-inv-update-orch-status-use-islive.md`. Found that agents update Status: Complete but forget to update D.E.K.N. "Next:" field, creating misleading artifacts that claim work is needed when it's already done.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-22-inv-how-do-investigation-files-become-stale.md` - Root cause analysis of stale investigation problem

### Commits
- Investigation file documenting root cause and recommendations

---

## Evidence (What Was Observed)

- Observation 1: Investigation file D.E.K.N. says "Next: Implement..." at line 14
- Observation 2: Same file says "Status: Complete" at line 46
- Observation 3: Actual code at `cmd/orch/main.go:1602-1616` shows `state.GetLiveness()` IS implemented
- Observation 4: Beads issue orch-go-0cjl confirms "Phase: Complete - Updated orch status to use state.GetLiveness()..."
- Observation 5: Comparison with `2025-12-21-inv-orch-status-showing-stale-sessions.md` shows CORRECT pattern: "Next: Implementation complete. Fix committed."

### Tests Run
```bash
# Compared 4 sources to verify divergence
# 1. D.E.K.N. summary - says "Next: Implement..."
# 2. Status field - says "Complete"
# 3. Actual code - GetLiveness() IS implemented
# 4. Beads comments - confirms implementation complete

# Result: D.E.K.N. is the only stale source
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-22-inv-how-do-investigation-files-become-stale.md` - Root cause analysis

### Decisions Made
- Decision 1: Add D.E.K.N./Status consistency check to Self-Review because it addresses root cause with minimal cost

### Constraints Discovered
- D.E.K.N. 'Next:' field must be updated when marking Status: Complete - prevents stale investigations

### Externalized via `kn`
- `kn constrain "D.E.K.N. 'Next:' field must be updated when marking Status: Complete" --reason "Prevents stale investigations that mislead future agents"`

---

## Next (What Should Happen)

**Recommendation:** close + spawn-follow-up

### If Close
- [x] All deliverables complete (investigation file created)
- [x] Tests passing (manual verification of 4-source comparison)
- [x] Investigation file has `**Status:** Complete`
- [x] Ready for `orch complete orch-go-6akg`

### If Spawn Follow-up
**Issue:** Add D.E.K.N./Status consistency check to investigation skill Self-Review
**Skill:** feature-impl
**Context:**
```
Root cause analysis in .kb/investigations/2025-12-22-inv-how-do-investigation-files-become-stale.md
found agents update Status: Complete but forget D.E.K.N. "Next:" field.
Add Self-Review checklist item requiring D.E.K.N. "Next:" to match Status.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How common is this D.E.K.N./Status divergence pattern across all investigations? (Only examined 2 files)
- Would agents actually follow an additional checklist item, or would they skip it?

**Areas worth exploring further:**
- `kb lint` command to automatically check D.E.K.N./Status consistency
- Auto-updating D.E.K.N. when Status changes (but may be overengineered)

**What remains unclear:**
- Whether Self-Review enforcement is sufficient, or if tooling is needed

*(Straightforward root cause analysis - limited unexplored territory)*

---

## Session Metadata

**Skill:** investigation
**Model:** claude
**Workspace:** `.orch/workspace/og-inv-how-do-investigation-22dec/`
**Investigation:** `.kb/investigations/2025-12-22-inv-how-do-investigation-files-become-stale.md`
**Beads:** `bd show orch-go-6akg`
