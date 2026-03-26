# Session Synthesis

**Agent:** og-debug-debug-investigate-whether-26mar-e4c1
**Issue:** orch-go-o5uih
**Duration:** 2026-03-26 17:20:42 UTC -> 2026-03-26 17:23:12 UTC
**Outcome:** success

---

## Plain-Language Summary

The stall tracker does not currently distinguish between a slow agent and a truly stalled agent in the way the comments promise. Because it refreshes the stored timestamp every time tokens are unchanged, it measures only the gap since the previous poll, so a genuinely stuck agent that is checked every 30 seconds never reaches the 3 minute stall threshold.

## TLDR

I verified that `pkg/daemon/stall_tracker.go` is incorrect under normal polling: repeated unchanged token snapshots reset the timer instead of accumulating no-progress time. Existing tests pass because they only cover a single sleep longer than the threshold, so I recorded the defect, created follow-up issue `orch-go-mpj69`, and captured a constraint in the knowledge base.

---

## Delta (What Changed)

### Files Created
- `.orch/workspace/og-debug-debug-investigate-whether-26mar-e4c1/VERIFICATION_SPEC.yaml` - Command-level verification evidence for the investigation.
- `.orch/workspace/og-debug-debug-investigate-whether-26mar-e4c1/SYNTHESIS.md` - Session findings and follow-up recommendation.
- `.orch/workspace/og-debug-debug-investigate-whether-26mar-e4c1/BRIEF.md` - Human-oriented comprehension artifact.

### Files Modified
- None.

### Commits
- Pending.

---

## Evidence (What Was Observed)

- `pkg/daemon/stall_tracker.go:58` stores a fresh `Timestamp` before evaluating whether tokens changed, so unchanged polls overwrite the last-known-progress time.
- `pkg/daemon/stall_tracker.go:74` computes stalled duration as `now.Sub(prev.Timestamp)`, which becomes only the interval since the immediately previous poll.
- `cmd/orch/status_cmd.go:361` and `cmd/orch/serve_agents_handlers.go:444` call `globalStallTracker.Update(...)` during status/dashboard refreshes.
- `cmd/orch/serve_agents_cache.go:166` documents that the dashboard polls `/api/agents` every 30 seconds, which is far below the 3 minute stall threshold configured in `cmd/orch/serve_agents_status.go:225`.
- `pkg/daemon/stall_tracker_test.go:10` through `pkg/daemon/stall_tracker_test.go:149` only cover a single wait longer than the threshold; they do not simulate repeated unchanged polls, so they miss the bug.

### Tests Run
```bash
go test ./pkg/daemon -run 'TestStallTracker_'
# PASS

python3 - <<'PY'
threshold = 180
polls = [0, 30, 60, 90, 120, 150, 180, 210]
last_total = None
last_timestamp = None
for t in polls:
    total = 1500
    if last_total is None:
        stalled = False
    elif total > last_total:
        stalled = False
    else:
        stalled = (t - last_timestamp) >= threshold
    print(f't={t}s stalled={stalled}')
    last_total = total
    last_timestamp = t
PY
# Observed: stalled=False for every poll, even after 210s with no token growth.
```

---

## Architectural Choices

### Keep this session investigative instead of patching the tracker
- **What I chose:** Stop at evidence gathering, issue creation, and knowledge capture.
- **What I rejected:** Directly editing `pkg/daemon/stall_tracker.go` in this worker session.
- **Why:** The spawn context marks this as a hotspot, and worker guidance requires investigation findings that imply code changes to route through `architect` before implementation.
- **Risk accepted:** The bug remains until follow-up work lands, but the system now has explicit evidence and a tracked next step.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.orch/workspace/og-debug-debug-investigate-whether-26mar-e4c1/SYNTHESIS.md` - Investigation synthesis for orchestrator review.

### Decisions Made
- Treat the result as an architectural follow-up rather than an immediate fix because the hotspot warning and worker protocol both require architect routing first.

### Constraints Discovered
- Stall detection must preserve the timestamp of the last token increase; otherwise frequent polls hide genuine stalls.

### Externalized via `kb quick`
- `kb quick constrain "stall detection must track time since last token increase, not time since last poll" --reason "pkg/daemon/stall_tracker.go currently overwrites snapshot timestamps on unchanged polls, so 30s dashboard refreshes never accumulate toward the 3m threshold"`

---

## Verification Contract

- Verification artifact: `.orch/workspace/og-debug-debug-investigate-whether-26mar-e4c1/VERIFICATION_SPEC.yaml`
- Key outcomes: existing tests pass, but source inspection plus simulation show the implementation never accumulates no-progress duration across frequent polls.

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** `orch-go-mpj69`
**Skill:** architect
**Context:**
```text
The stall tracker resets snapshot timestamps on every unchanged poll, so agents polled every 30s never cross the 3m stall threshold. The follow-up should design a tracker that preserves last-progress time and define the right regression coverage for both truly stalled and slow-but-advancing agents.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should slow-token agents get a distinct advisory state separate from "stalled" so the dashboard can surface slow progress without implying a hang?
- Should the tracker key off message/activity timestamps in addition to token deltas for tools that produce long silent stretches?

**Areas worth exploring further:**
- Regression coverage for repeated polls under both `orch status` and dashboard code paths.
- Whether `GetStallDuration` and `IsStalled` should share a single last-progress abstraction instead of duplicating logic.

**What remains unclear:**
- The intended product behavior for agents that make progress more slowly than the 3 minute threshold but still emit occasional activity.

---

## Friction

**System friction experienced during this session:**
- Tooling: `python` was unavailable in the worker shell, so I switched to `python3` for the quick behavior simulation.

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** openai/gpt-5.4
**Workspace:** `.orch/workspace/og-debug-debug-investigate-whether-26mar-e4c1/`
**Investigation:** None - evidence recorded in workspace artifacts
**Beads:** `bd show orch-go-o5uih`
