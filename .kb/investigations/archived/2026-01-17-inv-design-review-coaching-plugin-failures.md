<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Worker health metrics are not appearing because detectWorkerSession() caches `false` prematurely on the first non-matching tool call, and detection signal 1 (bash workdir) never fires because bash tools don't have a `workdir` argument.

**Evidence:** Verified: (1) `grep "tool_failure_rate|context_usage" ~/.orch/coaching-metrics.jsonl` returns zero hits, (2) bash tool args only include `command`/`timeout` - no `workdir`, (3) commit b82715c1 removed detection signal 3 (filePath containing .orch/workspace/), (4) caching at line 1256 fires on EVERY tool call, not just successful detections.

**Knowledge:** The "fix" in commit b82715c1 made detection worse, not better. The pattern of caching on first miss creates a race condition: if ANY tool call happens before reading SPAWN_CONTEXT.md, the session is permanently marked as non-worker.

**Next:** Fix detectWorkerSession() to (1) only cache `true` results, not `false`, (2) restore filePath-based detection for any .orch/workspace/ path, and (3) remove the broken bash workdir check.

**Promote to Decision:** Superseded - coaching plugin disabled (2026-01-28-coaching-plugin-disabled.md)

---

# Investigation: Design Review Coaching Plugin Failures

**Question:** Why are worker health metrics (tool_failure_rate, context_usage, time_in_phase, commit_gap) not appearing in ~/.orch/coaching-metrics.jsonl despite implementation in coaching.ts?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** Agent og-arch-review-design-failure-17jan-792c
**Phase:** Complete
**Next Step:** None - findings documented with fix recommendations
**Status:** Complete

<!-- Lineage -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Worker Detection Signal 1 (bash workdir) Never Fires

**Evidence:** The detectWorkerSession function checks for `args?.workdir`:

```typescript
// Detection signal 1: bash tool with workdir in .orch/workspace/
if (tool === "bash" && args?.workdir) {
  if (args.workdir.includes(".orch/workspace/")) {
    isWorker = true
  }
}
```

However, the bash tool in OpenCode/Claude does NOT have a `workdir` argument. The bash tool args are:
- `command` (required)
- `timeout` (optional)
- `dangerouslyDisableSandbox` (optional)
- `run_in_background` (optional)

There is no `workdir` argument, so this detection signal **will never fire**.

**Source:** `plugins/coaching.ts:1238-1244`, OpenCode tool schema

**Significance:** This is 50% of the remaining detection logic, and it's completely broken. Detection now relies entirely on Signal 2 (SPAWN_CONTEXT.md read).

---

### Finding 2: Caching Prematurely Marks Sessions as Non-Worker

**Evidence:** The caching logic at line 1256 runs on EVERY tool call, not just successful detections:

```typescript
// Cache the result
workerSessions.set(sessionId, isWorker)  // Always caches, even when false
```

**Failure sequence for a typical worker session:**
1. First tool call: `bash` with command `pwd` - no `args.workdir` exists → `isWorker = false` → **cached as false**
2. Second tool call: `read` SPAWN_CONTEXT.md - **already cached as false, returns immediately without checking**
3. All subsequent calls: Return cached `false`

The function should only cache `true` (confirmed worker), not `false` (unconfirmed).

**Source:** `plugins/coaching.ts:1255-1263`

**Significance:** This is the root cause. Even if Signal 2 would correctly detect a worker, the premature caching prevents it from ever being evaluated if any other tool call happens first.

---

### Finding 3: Commit b82715c1 Made Detection Worse

**Evidence:** The commit titled "fix: enable plugin loading and refine worker detection" (b82715c1) actually REMOVED detection signal 3:

```diff
-    // Detection signal 3: any tool with filePath in .orch/workspace/
-    if (args?.filePath && typeof args.filePath === "string") {
-      if (args.filePath.includes(".orch/workspace/")) {
-        log(`Worker detected (filePath in workspace): session ${sessionId}, file: ${args.filePath}`)
-        isWorker = true
-      }
-    }
```

Before this commit, workers could be detected by ANY read/write to a file in `.orch/workspace/`. After this commit, only the specific `SPAWN_CONTEXT.md` file triggers detection.

**Source:** `git show b82715c1`

**Significance:** The "fix" actually removed the most reliable detection signal. The rationale in the commit message doesn't explain why this was removed.

---

### Finding 4: Orchestrator Metrics Are Working; Worker Metrics Are Not

**Evidence:** The metrics file (`~/.orch/coaching-metrics.jsonl`, 78KB) contains recent entries for:
- `action_ratio` - orchestrator metric, working
- `analysis_paralysis` - orchestrator metric, working
- `compensation_pattern` - orchestrator metric, working
- `frame_collapse` - orchestrator metric (would fire if orchestrators edit code)

But ZERO entries for worker-specific metrics:
- `tool_failure_rate` - missing
- `context_usage` - missing
- `time_in_phase` - missing
- `commit_gap` - missing

This confirms the plugin is loaded and working for orchestrators, but worker detection is failing.

**Source:** `tail -20 ~/.orch/coaching-metrics.jsonl`, `grep -E "tool_failure_rate|context_usage" ~/.orch/coaching-metrics.jsonl`

**Significance:** The worker health tracking code exists but is never reached because all worker sessions are misclassified as orchestrators.

---

### Finding 5: Prior Architectural Recommendation Not Implemented

**Evidence:** The Jan 11 investigation (`2026-01-11-inv-review-design-coaching-plugin-injection.md`) identified a fundamental architectural problem: injection is coupled to observation, creating restart brittleness. The recommendation was to separate injection into an independent daemon.

This recommendation was never implemented. The current approach continues to have:
- In-memory session state (lost on restart)
- Injection triggered from tool.execute.after (can't run independently)
- Coupling between passive observation and active intervention

**Source:** `.kb/investigations/2026-01-11-inv-review-design-coaching-plugin-injection.md`

**Significance:** The worker detection bugs are symptoms of a larger architectural issue. Even if detection is fixed, the system will still have restart brittleness and coupling problems identified in the prior investigation.

---

## Synthesis

**Key Insights:**

1. **Broken Detection Creates Broken Metrics** - Worker health tracking was implemented correctly, but it's never reached because detectWorkerSession() returns false for all sessions due to (a) broken bash workdir check and (b) premature caching of false results.

2. **"Fix" Commits Without Testing** - Commit b82715c1 claims to "fix" and "refine" worker detection, but actually made it worse by removing the most reliable detection signal. This suggests changes are being made without testing whether workers are actually detected.

3. **Symptom Chasing Without Root Cause Analysis** - The coaching plugin area has 8+ bugs and 2 abandoned investigations because fixes target symptoms (detection signals, timing, content checks) rather than the fundamental issues (premature caching, wrong assumptions about tool args, coupled architecture).

**Answer to Investigation Question:**

Worker health metrics are not appearing because:

1. **Detection Signal 1 never fires** (bash has no workdir arg) - Finding 1
2. **Detection Signal 3 was removed** (commit b82715c1) - Finding 3
3. **Detection Signal 2 is race-losered by premature caching** (first tool call caches false) - Finding 2

The fix requires:
1. Only cache `true` results in detectWorkerSession(), never cache `false`
2. Restore filePath-based detection for any `.orch/workspace/` path
3. Remove the broken bash workdir check (it provides false confidence)
4. Consider implementing the daemon-based architecture from the Jan 11 investigation

---

## Structured Uncertainty

**What's tested:**

- ✅ Metrics file has zero worker health metrics (verified: grep for tool_failure_rate/context_usage returns empty)
- ✅ Orchestrator metrics ARE appearing (verified: tail shows action_ratio, analysis_paralysis)
- ✅ bash tool has no workdir argument (verified: OpenCode tool schema)
- ✅ Commit b82715c1 removed detection signal 3 (verified: git show b82715c1)
- ✅ Caching runs on every call (verified: line 1256 is outside any conditional)

**What's untested:**

- ⚠️ Whether fixing caching will actually produce worker metrics (hypothesis, not tested)
- ⚠️ Whether restoring detection signal 3 might over-detect (orchestrators reading workspace files)
- ⚠️ Performance impact of not caching (each tool call rechecks detection)

**What would change this:**

- Finding would be wrong if bash tool actually has a workdir arg we didn't find
- Finding would be wrong if workers ARE being detected but metrics aren't being written (different bug)
- Finding would be wrong if there's a separate code path that prevents metric writing after detection

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Fix Detection Logic** - Modify detectWorkerSession() to only cache positive results and restore reliable detection signals.

**Why this approach:**
- Directly addresses the root cause (Finding 2: premature caching)
- Minimal change (few lines of code)
- Immediately enables worker health tracking

**Trade-offs accepted:**
- Detection checks happen on each tool call (tiny performance cost)
- Orchestrators reading workspace files might be misclassified (rare edge case)

**Implementation sequence:**

1. **Fix caching** - Only cache when `isWorker = true`:
```typescript
if (isWorker) {
  workerSessions.set(sessionId, true)
  log(`Session ${sessionId} marked as worker`)
}
// Don't cache false - keep checking
return isWorker
```

2. **Restore filePath detection** - Re-add signal 3:
```typescript
// Detection signal: any tool with filePath in .orch/workspace/
if (args?.filePath && typeof args.filePath === "string") {
  if (args.filePath.includes(".orch/workspace/")) {
    isWorker = true
  }
}
```

3. **Remove broken bash check** - Delete lines 1238-1244 (bash workdir check that never fires)

4. **Add debug logging** - Temporarily add `log()` calls to verify detection is working

### Alternative Approaches Considered

**Option B: Explicit Worker Flag at Spawn Time**
- **Pros:** Definitive detection, no heuristics
- **Cons:** Requires orch spawn changes, doesn't help existing sessions
- **When to use instead:** If heuristic detection proves unreliable long-term

**Option C: Implement Daemon Architecture**
- **Pros:** Fixes deeper architectural issues (coupling, restart brittleness)
- **Cons:** Higher implementation cost, scope expansion
- **When to use instead:** If detection fixes reveal more coupling bugs

**Rationale for recommendation:** Option A is the minimal fix that addresses the immediate problem. Options B and C are valid but represent larger scope changes that should be separate work items.

---

### Implementation Details

**What to implement first:**
- Fix caching logic (highest impact, least risk)
- Restore filePath detection (second highest impact)
- Remove bash workdir check (cleanup)

**Things to watch out for:**
- ⚠️ Test with actual worker sessions, not just code review
- ⚠️ Orchestrators might read workspace files (e.g., monitoring) - consider adding exclusion for orchestrator session IDs if this becomes a problem
- ⚠️ Plugin needs server restart to pick up changes

**Areas needing further investigation:**
- Whether the bash workdir arg was ever real (or always broken)
- Whether orchestrator frame collapse detection is also affected by similar bugs
- Whether the daemon architecture should be prioritized

**Success criteria:**
- ✅ Worker sessions produce tool_failure_rate metrics when tools fail
- ✅ Worker sessions produce context_usage metrics every 50 tool calls
- ✅ Dashboard shows worker health for active agents
- ✅ Zero regression in orchestrator metrics

---

## References

**Files Examined:**
- `plugins/coaching.ts` - Main coaching plugin implementation
- `.kb/investigations/2026-01-11-inv-review-design-coaching-plugin-injection.md` - Prior architectural analysis
- `.kb/investigations/2026-01-17-inv-add-worker-specific-metrics-plugins.md` - Worker metrics implementation
- `.kb/investigations/2026-01-17-inv-design-agent-self-health-context.md` - Agent health design
- `~/.orch/coaching-metrics.jsonl` - Metrics output file

**Commands Run:**
```bash
# Check for worker metrics
grep -E "tool_failure_rate|context_usage|time_in_phase|commit_gap" ~/.orch/coaching-metrics.jsonl

# Check recent metrics
tail -20 ~/.orch/coaching-metrics.jsonl

# Check commit that "fixed" detection
git show b82715c1

# Verify plugin symlink
ls -la /Users/dylanconlin/Documents/personal/orch-go/.opencode/plugins
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-11-inv-review-design-coaching-plugin-injection.md` - Identified architectural coupling problem
- **Investigation:** `.kb/investigations/2026-01-17-inv-design-agent-self-health-context.md` - Designed the worker health metrics system
- **Beads:** `orch-go-k3xr1` - This investigation task

---

## Investigation History

**2026-01-17 10:31:** Investigation started
- Initial question: Why are worker health metrics not appearing despite implementation?
- Context: Coaching plugin has been iteratively debugged but worker metrics remain silent

**2026-01-17 10:35:** Found metrics file status
- Confirmed 78KB metrics file exists with recent entries
- Confirmed zero worker-specific metric types present
- Confirmed orchestrator metrics ARE working (action_ratio, analysis_paralysis)

**2026-01-17 10:40:** Identified root cause
- Found premature caching of false results in detectWorkerSession()
- Found bash workdir check is invalid (bash has no workdir arg)
- Found commit b82715c1 removed the most reliable detection signal

**2026-01-17 10:50:** Investigation completed
- Status: Complete
- Key outcome: Detection is broken by premature caching and invalid bash arg check; fix is minimal but requires testing
