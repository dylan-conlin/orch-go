# Session Synthesis

**Agent:** og-inv-investigate-orchestration-lifecycle-25dec
**Issue:** orch-go-h2x8
**Duration:** 2025-12-25 11:10 → 2025-12-25 11:45
**Outcome:** success

---

## TLDR

Investigated why agents complete work (report Phase: Complete) but beads issues remain unclosed. Found 5 distinct breakpoints in the completion loop, with the primary gap being that daemon only spawns but doesn't process completions. Recommended adding daemon completion polling to close the loop automatically.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-25-inv-investigate-orchestration-lifecycle-end-end.md` - Full investigation with 6 findings, D.E.K.N. summary

### Files Modified
- None (investigation-only task)

### Commits
- (Investigation file commit pending)

---

## Evidence (What Was Observed)

- `orch review` shows 216 pending completions with SYNTHESIS.md in orch-go project
- 418 workspace directories exist in `.orch/workspace/`
- Beads shows 200 open issues, 16 in_progress, 1008 closed (clear backlog)
- Agent orch-go-k08g has Phase: Complete, beads closed, but still shows "active" in `orch status`
- SSE auto-complete code explicitly disabled in `pkg/opencode/service.go:100-105` with documented reason
- Daemon code (`pkg/daemon/daemon.go`) only has spawn loop, no completion processing

### Tests Run
```bash
# Count pending completions
orch review | head -60
# Result: 216 completions in orch-go project

# Check workspace count
ls .orch/workspace/ | wc -l
# Result: 419 directories

# Verify agent with closed beads still shows active
bd show orch-go-k08g  # Status: closed
orch status | grep k08g  # Shows as idle/active
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-25-inv-investigate-orchestration-lifecycle-end-end.md` - Complete lifecycle analysis

### Decisions Made
- Daemon completion polling is the recommended approach because:
  - Uses reliable Phase: Complete signal (not flaky SSE busy→idle)
  - Leverages existing daemon infrastructure
  - Doesn't require new monitoring architecture

### Constraints Discovered
- SSE busy→idle detection triggers during normal operation (loading, thinking, tools) - fundamentally unreliable
- Phase: Complete is the only reliable completion signal
- Pre-spawn duplicate check already handles Phase: Complete case correctly

### Externalized via `kn`
- `kn decide "Daemon completion polling preferred over SSE detection" --reason "SSE busy->idle triggers false positives during normal agent operation; Phase: Complete is only reliable signal"`
- `kn constraint "SSE busy->idle cannot detect true completion" --reason "Agents go idle during loading, thinking, waiting for tools - not just when done"`

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** Add daemon completion polling to close agent lifecycle loop
**Skill:** feature-impl
**Context:**
```
Daemon currently only spawns new work. Add completion processing that polls for 
Phase: Complete agents with unclosed beads issues, then runs `orch complete` on them.
This closes the completion loop automatically without human intervention.
See: .kb/investigations/2025-12-25-inv-investigate-orchestration-lifecycle-end-end.md
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should `orch review done` actually run `orch complete` on all listed items? (Currently only prints confirmation)
- Should completion also kill orphaned OpenCode sessions? (Currently only closes beads + tmux)

**Areas worth exploring further:**
- Batch completion performance (running 216 completions sequentially vs parallel)
- Session cleanup strategies (aggressive vs conservative)

**What remains unclear:**
- Optimal polling interval for completion processing (60s seems reasonable but untested)
- Whether Phase: Complete parsing handles all edge cases (dash variants, encoding)

---

## Session Metadata

**Skill:** investigation
**Model:** anthropic/claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-inv-investigate-orchestration-lifecycle-25dec/`
**Investigation:** `.kb/investigations/2025-12-25-inv-investigate-orchestration-lifecycle-end-end.md`
**Beads:** `bd show orch-go-h2x8`
