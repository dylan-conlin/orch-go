# Session Synthesis

**Agent:** og-debug-clean-up-stale-26mar-5312
**Issue:** orch-go-sc0av
**Duration:** 2026-03-26T17:22 → 2026-03-26T17:35
**Outcome:** success

---

## Plain-Language Summary

Closed worker issues were leaving behind stale tmux windows in `workers-orch-go` because the daemon's periodic cleanup task was never executing during verification pause. The daemon's main loop checked for verification pause *before* running periodic tasks — when paused, it would `continue` the loop immediately, skipping the cleanup entirely. The fix moves the periodic tasks call above the verification pause check, so maintenance operations (including stale window cleanup) run regardless of pause state. The pause is about preventing new spawns, not stopping housekeeping.

---

## TLDR

Stale tmux windows accumulated during daemon verification pause because the loop ordering skipped `runPeriodicTasks()`. Moved the call before `checkVerificationPause()` so cleanup runs unconditionally.

---

## Delta (What Changed)

### Files Modified
- `cmd/orch/daemon.go` — Reordered daemon loop: `runPeriodicTasks()` now runs before `checkVerificationPause()` instead of after it

### Files Created
- `.orch/workspace/og-debug-clean-up-stale-26mar-5312/VERIFICATION_SPEC.yaml`
- `.orch/workspace/og-debug-clean-up-stale-26mar-5312/SYNTHESIS.md`
- `.orch/workspace/og-debug-clean-up-stale-26mar-5312/BRIEF.md`

---

## Evidence (What Was Observed)

- `daemon.go:59-68` (before fix): `checkVerificationPause()` returned `true` → `continue` → skipped `runPeriodicTasks()` on line 63
- `daemon.go:59-68` (after fix): `runPeriodicTasks()` runs on line 64, then `checkVerificationPause()` on line 66
- `pkg/daemon/cleanup.go:211-274`: `cleanStaleTmuxWindows()` correctly identifies and kills windows for closed issues — the logic is sound, it just wasn't being called during pause
- `pkg/daemon/periodic.go:28-53`: `RunPeriodicCleanup()` delegates to scheduler timing — safe to call in any state
- `checkVerificationPause()` at `daemon_loop.go:557`: writes its own status file during pause, sleeps for poll interval, returns `true`

### Tests Run
```bash
go test ./pkg/daemon/ -run 'TestIsWindowStale|TestRunPeriodicCleanup' -v
# PASS: 14/14 tests

go test ./cmd/orch/ -run TestRunPeriodicTasks -v
# PASS: 4/4 tests

go build ./cmd/orch/
# Clean compilation
```

---

## Architectural Choices

### Move all periodic tasks before pause check (not just cleanup)
- **What I chose:** Move the entire `runPeriodicTasks()` call before `checkVerificationPause()`
- **What I rejected:** Extracting only cleanup into a separate call before the pause check
- **Why:** All periodic tasks are maintenance operations independent of spawn decisions. Cleanup, orphan detection, phase timeout monitoring, health checks — none of these should be blocked by verification pause. The pause exists to prevent new agent spawns, not to stop housekeeping.
- **Risk accepted:** Periodic tasks will now run during every pause cycle (every poll interval), but the scheduler's `IsDue()` check already enforces their individual intervals, so this is a no-op for non-due tasks.

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` — key outcomes:
- All existing tests pass (14 daemon tests + 4 periodic task tests)
- Clean build
- Code review confirms `runPeriodicTasks()` precedes `checkVerificationPause()` in loop

---

## Knowledge (What Was Learned)

### Constraints Discovered
- The daemon OODA loop's ordering is load-bearing: maintenance tasks must come before any `continue` path to avoid being skipped

### Decisions Made
- All periodic tasks run before verification pause (not just cleanup) — they're all maintenance, none spawn work

---

## Next (What Should Happen)

**Recommendation:** close

- [x] All deliverables complete
- [x] Tests passing
- [x] Fix is minimal (4-line reorder with comment)
- [x] Ready for `orch complete orch-go-sc0av`

---

## Unexplored Questions

- The `periodicResult` computed during pause is discarded (the pause path writes its own simplified status file). Could include periodic snapshots in the pause status for richer monitoring during pause. Not needed for this fix.

---

## Friction

No friction — smooth session. Root cause was identified quickly from the daemon loop structure.

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-debug-clean-up-stale-26mar-5312/`
**Beads:** `bd show orch-go-sc0av`
