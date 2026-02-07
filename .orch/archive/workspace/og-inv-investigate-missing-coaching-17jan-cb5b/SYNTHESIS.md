# Session Synthesis

**Agent:** og-inv-investigate-missing-coaching-17jan-cb5b
**Issue:** orch-go-ls80s
**Duration:** 2026-01-17 18:27 → 2026-01-17 18:45
**Outcome:** success

---

## TLDR

Investigated missing worker metrics and frame collapse detection in coaching plugin. Root cause: detectWorkerSession caches `false` results, permanently misclassifying workers as orchestrators. Frame collapse is working correctly (orchestrator hasn't edited code files).

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-17-inv-investigate-missing-coaching-metrics-frame.md` - Complete investigation with root cause analysis and fix recommendations

### Files Modified
- None (investigation only)

### Commits
- (pending) Investigation file with findings

---

## Evidence (What Was Observed)

- Worker session `ses_432d48144ffe7crCEx2kx1tGBG` emits orchestrator metrics (action_ratio, analysis_paralysis, compensation_pattern)
- Zero worker-specific metrics (tool_failure_rate, context_usage, time_in_phase, commit_gap) exist in coaching-metrics.jsonl
- Orchestrator session has action_ratio=0 with reads=6+, meaning 0 code edits
- detectWorkerSession caches both true AND false results at line 1256
- Once cached as false, subsequent detection signals are ignored

### Tests Run
```bash
# Check for worker-specific metrics (none found)
cat ~/.orch/coaching-metrics.jsonl | grep -E "tool_failure_rate|context_usage|time_in_phase|commit_gap"
# Result: No matches - confirms worker metrics never emitted

# Check for frame_collapse metrics (none found, expected)
cat ~/.orch/coaching-metrics.jsonl | grep "frame_collapse"
# Result: No matches - expected because orchestrator hasn't edited code files
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-17-inv-investigate-missing-coaching-metrics-frame.md` - Root cause analysis of coaching plugin detection bug

### Decisions Made
- Frame collapse detection is working correctly; lack of metrics is expected behavior (no code edits)

### Constraints Discovered
- detectWorkerSession must not cache negative results - once cached as false, detection cannot be corrected
- Detection signals are order-dependent; first tool call determines cache state

### Externalized via `kb`
- `kb quick constrain "detectWorkerSession must only cache positive results" --reason "Caching false permanently misclassifies workers"` - (to run)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete {issue-id}`

### Implementation Needed (by follow-up worker)
Fix detectWorkerSession in plugins/coaching.ts:
1. Remove early return on cached `false` (line 1232-1234)
2. Only cache when `isWorker=true` (line 1256)
3. Optional: Add check limit (10 tool calls) to stop evaluating eventually

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- What is the actual tool call order for spawned workers? (would confirm hypothesis about first-call detection)
- Is client.session.prompt always available at injection time? (would verify injection success/failure)

**What remains unclear:**
- Whether action_ratio coaching injection actually fired or silently failed

*(Investigation found root cause; implementation is straightforward bug fix)*

---

## Session Metadata

**Skill:** investigation
**Model:** opus
**Workspace:** `.orch/workspace/og-inv-investigate-missing-coaching-17jan-cb5b/`
**Investigation:** `.kb/investigations/2026-01-17-inv-investigate-missing-coaching-metrics-frame.md`
**Beads:** `bd show orch-go-ls80s`
