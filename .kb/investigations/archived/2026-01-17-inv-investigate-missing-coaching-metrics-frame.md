<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Worker sessions fail to emit worker-specific metrics (tool_failure_rate, context_usage, etc.) because detectWorkerSession caches `false` results, permanently misclassifying workers as orchestrators.

**Evidence:** Session ses_432d48144ffe7crCEx2kx1tGBG emits orchestrator metrics (action_ratio, analysis_paralysis) instead of worker metrics. Zero worker-specific metrics exist in coaching-metrics.jsonl.

**Knowledge:** The caching logic at line 1256 caches both true AND false results. Once a session's first tool call doesn't trigger detection (e.g., glob before read of SPAWN_CONTEXT.md), subsequent detection signals are ignored.

**Next:** Fix detectWorkerSession to only cache when isWorker=true, or re-evaluate until worker is detected. See Implementation Recommendations.

**Promote to Decision:** recommend-no (bug fix, not architectural choice)

---

# Investigation: Missing Coaching Metrics and Frame Collapse Detection

**Question:** Why are worker metrics not appearing for ses_432d48144ffe7crCEx2kx1tGBG, and why is frame_collapse not triggering for orchestrator ses_432cd86bfffeTzC5UoALT8xNs7?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** Investigation worker (spawned by orchestrator)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Worker Detection Caching Bug Prevents Worker Metrics

**Evidence:**
- Session `ses_432d48144ffe7crCEx2kx1tGBG` (worker) emits orchestrator metrics: `action_ratio`, `analysis_paralysis`, `compensation_pattern`
- Zero worker-specific metrics (`tool_failure_rate`, `context_usage`, `time_in_phase`, `commit_gap`) exist in coaching-metrics.jsonl
- Worker health tracking (lines 1427-1452) never starts because `detectWorkerSession` returns false

**Source:**
- `plugins/coaching.ts:1229-1262` - detectWorkerSession function
- `~/.orch/coaching-metrics.jsonl` - actual metrics data
- Command: `grep -E "tool_failure_rate|context_usage|time_in_phase|commit_gap" ~/.orch/coaching-metrics.jsonl` returns zero results

**Significance:** The critical bug is at line 1231-1234 and line 1256:
```typescript
const cached = workerSessions.get(sessionId)
if (cached !== undefined) {
  return cached  // Returns false even when subsequent signals would indicate worker
}
// ... detection logic ...
workerSessions.set(sessionId, isWorker)  // Caches false results too
```

When a session's first tool call doesn't trigger detection (e.g., `glob` before `read` of SPAWN_CONTEXT.md), the session is cached as NOT a worker, and all subsequent detection signals are ignored.

---

### Finding 2: Frame Collapse Detection is Working Correctly (Expected No Triggers)

**Evidence:**
- Session `ses_432cd86bfffeTzC5UoALT8xNs7` (orchestrator) has `action_ratio=0` with `actions=0, reads=6` (then reads=9)
- Zero `edit` or `write` tool calls recorded for this session
- Frame collapse only triggers on code file edits (lines 1504-1548)

**Source:**
- `~/.orch/coaching-metrics.jsonl` - shows action_ratio metrics with actions=0
- `plugins/coaching.ts:1504-1507` - frame collapse condition: `if (tool === "edit" || tool === "write")`

**Significance:** This is expected behavior, not a bug. The orchestrator hasn't edited any code files, so frame_collapse shouldn't trigger. The detection logic is correct - it waits for actual code edits before warning.

---

### Finding 3: action_ratio Coaching Injection Conditions Met but Not Verified

**Evidence:**
- Orchestrator at 18:23:47 had `action_ratio=0` with `reads=6, actions=0`
- Injection condition (line 618): `actionRatio < 0.5 && state.reads >= 6` was TRUE
- Injection calls `client.session.prompt` with `noReply: true` (line 717)
- No direct evidence of injection success/failure in metrics file

**Source:**
- `plugins/coaching.ts:618-620` - injection condition
- `plugins/coaching.ts:715-725` - injection implementation
- Metrics show condition was met but injection result not logged to file

**Significance:** The injection SHOULD happen but we can't verify success without DEBUG logs. The code at line 723-724 logs success/failure but only when `DEBUG=true`. A production issue might exist where `client.session.prompt` is undefined.

---

## Synthesis

**Key Insights:**

1. **Caching negative results breaks detection** - The detectWorkerSession function's caching strategy is fundamentally flawed. It treats "not yet detected" the same as "definitely not a worker", permanently locking sessions into the wrong category.

2. **Detection signals are order-dependent** - The two detection signals (bash workdir, SPAWN_CONTEXT.md read) happen at different times in a session. If a non-matching tool call comes first, detection fails.

3. **Frame collapse is working as designed** - The lack of frame_collapse metrics for the orchestrator is expected. The orchestrator hasn't edited code files, so there's nothing to warn about. This finding validates the frame collapse logic.

**Answer to Investigation Question:**

**Worker metrics missing:** The root cause is the caching bug in detectWorkerSession (lines 1229-1262). The worker session's first tool call was likely `glob` or `bash` (without workdir), which cached `false`. Subsequent detection signals were ignored.

**Frame collapse not triggering:** This is expected behavior. The orchestrator session has action_ratio=0 (0 actions, 6+ reads), meaning no code edits have occurred. Frame collapse only triggers on actual code file edits.

**action_ratio injection:** The condition was met but injection success cannot be verified without DEBUG logs. The injection may have succeeded silently, or `client.session.prompt` may be undefined.

---

## Structured Uncertainty

**What's tested:**

- ✅ Worker session emits orchestrator metrics (verified: grep coaching-metrics.jsonl for session ID)
- ✅ Zero worker-specific metrics exist in the file (verified: grep for tool_failure_rate etc. returns nothing)
- ✅ Orchestrator session has action_ratio=0 with reads>=6 (verified: metrics file shows exact values)
- ✅ Bash tool does have workdir parameter (verified: read opencode/packages/opencode/src/tool/bash.ts)
- ✅ OpenCode uses filePath (camelCase) in Read tool args (verified: grep opencode prompt.ts)

**What's untested:**

- ⚠️ Actual order of first tool calls for worker session (would need OpenCode session history)
- ⚠️ Whether client.session.prompt is available at injection time (would need DEBUG logs)
- ⚠️ Whether injection actually fired or silently failed (no observable evidence)

**What would change this:**

- Finding would be wrong if worker session's first tool WAS read of SPAWN_CONTEXT.md (need to verify tool call order)
- Finding would be wrong if there's a different caching mechanism we haven't found
- Injection hypothesis would be wrong if DEBUG logs show injection fired successfully

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Only cache positive detection results** - Modify detectWorkerSession to only cache when `isWorker=true`, allowing re-evaluation until worker is detected.

**Why this approach:**
- Directly fixes the root cause without changing detection signals
- Maintains performance (still caches after positive detection)
- Preserves existing detection logic that's correct when it fires

**Trade-offs accepted:**
- Slightly more detection checks for orchestrator sessions (negligible overhead)
- Doesn't address the fundamental signal-ordering issue

**Implementation sequence:**
1. Remove the early return on cached `false` - Allow re-evaluation each call
2. Only call `workerSessions.set(sessionId, true)` when isWorker is true
3. Add a check limit (e.g., 10 tool calls) to eventually stop checking

### Alternative Approaches Considered

**Option B: Add more detection signals**
- **Pros:** Makes detection more robust against ordering issues
- **Cons:** Adds complexity; current signals ARE correct when they fire
- **When to use instead:** If caching fix alone doesn't solve all cases

**Option C: Check workdir path contains .orch/workspace/ in directory context**
- **Pros:** Uses session directory (Instance.directory) which is always available
- **Cons:** Requires access to Instance.directory in plugin context; may not be available
- **When to use instead:** If tool args-based detection proves unreliable

**Rationale for recommendation:** Option A is the simplest fix that directly addresses the root cause. The detection signals are correct - they just need to be evaluated until one succeeds.

---

### Implementation Details

**What to implement first:**
- Fix the caching logic in detectWorkerSession (highest impact, 5 lines of code)
- Add injection result logging to metrics file (for verification)

**Things to watch out for:**
- ⚠️ Don't break orchestrator detection - need to verify orchestrators are still NOT cached as workers
- ⚠️ Check limit needs tuning - too low might miss workers, too high wastes cycles
- ⚠️ Injection logging shouldn't reveal sensitive session content

**Areas needing further investigation:**
- What's the actual tool call order for spawned workers? (not in scope but would confirm hypothesis)
- Is client.session.prompt always available? (need DEBUG logs to verify)
- Should we add a third detection signal (e.g., workspace directory pattern)?

**Success criteria:**
- ✅ Worker sessions emit worker-specific metrics (tool_failure_rate, context_usage, etc.)
- ✅ Orchestrator sessions still emit orchestrator metrics (action_ratio, frame_collapse)
- ✅ Injection results logged to metrics file for observability

---

## References

**Files Examined:**
- `plugins/coaching.ts:1229-1262` - detectWorkerSession function with caching bug
- `plugins/coaching.ts:618-620` - action_ratio injection condition
- `plugins/coaching.ts:715-725` - injectCoachingMessage implementation
- `plugins/coaching.ts:1504-1548` - Frame collapse detection logic
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/tool/bash.ts` - Verified bash tool has workdir parameter
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/prompt.ts` - Verified filePath casing

**Commands Run:**
```bash
# Check coaching metrics for both sessions
cat ~/.orch/coaching-metrics.jsonl | grep -E "(ses_432d48144ffe7crCEx2kx1tGBG|ses_432cd86bfffeTzC5UoALT8xNs7)"

# Check for worker-specific metrics (none found)
cat ~/.orch/coaching-metrics.jsonl | grep -E "tool_failure_rate|context_usage|time_in_phase|commit_gap"

# Check for frame_collapse metrics (none found)
cat ~/.orch/coaching-metrics.jsonl | grep "frame_collapse"

# Verify filePath casing in OpenCode
grep -r "filePath|file_path" /Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/prompt.ts
```

**External Documentation:**
- N/A

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-11-inv-coaching-plugin-injection-debug.md` - Prior debugging of injection issues
- **Investigation:** `.kb/investigations/2026-01-10-inv-debug-worker-filtering-coaching-ts.md` - Prior worker filtering work
- **Design:** `docs/designs/2026-01-10-orchestrator-coaching-plugin.md` - Original coaching plugin design

---

## Investigation History

**2026-01-17 18:27:** Investigation started
- Initial question: Why are worker metrics missing and frame_collapse not triggering?
- Context: Spawned by orchestrator to investigate coaching plugin issues for two specific sessions

**2026-01-17 18:30:** Found critical caching bug
- detectWorkerSession caches false results, permanently misclassifying workers
- Worker session emits orchestrator metrics instead of worker-specific metrics

**2026-01-17 18:35:** Verified frame_collapse is working correctly
- Orchestrator has 0 actions, so no code edits to trigger frame_collapse
- This is expected behavior, not a bug

**2026-01-17 18:40:** Investigation completed
- Status: Complete
- Key outcome: Root cause identified - caching bug in detectWorkerSession; fix is to only cache positive detection results
