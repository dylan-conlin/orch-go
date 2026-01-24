<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added 4 worker-specific metrics (tool_failure_rate, context_usage, time_in_phase, commit_gap) to coaching.ts, transforming workers from "skip all metrics" to "track worker health metrics."

**Evidence:** Code review shows metrics emit to coaching-metrics.jsonl using existing writeMetric() pattern (lines 1029-1118).

**Knowledge:** Workers need different signals than orchestrators - context budget warnings, tool failure patterns, progress tracking. Existing detection via .orch/workspace/ path works for routing.

**Next:** Complete - implementation finished, ready for orch complete.

**Promote to Decision:** recommend-no - Tactical implementation of existing architectural design, no new pattern.

---

# Investigation: Add Worker Specific Metrics Plugins

**Question:** How should the 4 new worker-specific metrics be implemented in coaching.ts using the existing action_ratio pattern?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** Agent og-feat-add-worker-specific-17jan-c994
**Phase:** Complete
**Next Step:** None - implementation ready
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Existing action_ratio Pattern Uses writeMetric()

**Evidence:** The `action_ratio` metric at lines 586-602 of coaching.ts uses `writeMetric()` to record to `~/.orch/coaching-metrics.jsonl`:

```typescript
writeMetric({
  timestamp: now,
  session_id: state.sessionId,
  metric_type: "action_ratio",
  value: parseFloat(actionRatio.toFixed(2)),
  details: {
    actions: state.actions,
    reads: state.reads,
  },
})
```

**Source:** `plugins/coaching.ts:586-602`, `plugins/coaching.ts:391-399` (writeMetric function)

**Significance:** This is the pattern to follow for new metrics - all use the same JSONL format with timestamp, session_id, metric_type, value, and details fields.

---

### Finding 2: Workers Currently Skip All Metrics

**Evidence:** Lines 1362-1366 (pre-change) returned early for worker sessions:
```typescript
if (detectWorkerSession(sessionId, tool, input.args)) {
  // Skip all metrics tracking for worker sessions
  return
}
```

Worker detection uses three signals:
1. bash tool with workdir in `.orch/workspace/`
2. read tool accessing SPAWN_CONTEXT.md
3. any tool with filePath in `.orch/workspace/`

**Source:** `plugins/coaching.ts:1145-1189` (detectWorkerSession function)

**Significance:** Workers were excluded because orchestrator metrics (frame_collapse, action_ratio) don't apply to workers. But workers still need health metrics - different signals, not no signals.

---

### Finding 3: Token Estimation Must Be Approximate

**Evidence:** OpenCode doesn't expose actual token counts via plugin API. The architect investigation noted this constraint and accepted approximate estimation.

Token estimation formula implemented:
```typescript
const TOKENS_PER_TOOL_CALL = 500   // Average tool call overhead
const CHARS_PER_TOKEN = 4          // Average chars per token

const toolCallTokens = state.totalToolCalls * TOKENS_PER_TOOL_CALL
const readTokens = Math.round(state.totalReadBytes / CHARS_PER_TOKEN)
```

**Source:** `plugins/coaching.ts:993-1001` (estimateWorkerTokenUsage function), architect investigation design notes

**Significance:** This approximation will systematically underestimate (misses assistant output, system prompts) but serves as a useful early warning signal.

---

## Synthesis

**Key Insights:**

1. **Workers Need Health Metrics, Not Orchestrator Metrics** - The existing worker detection is correct, but the action should be "track different metrics" not "skip all metrics." Workers benefit from: context budget awareness, tool failure patterns, time tracking, commit gap reminders.

2. **Periodic Emission Prevents Metric Spam** - Metrics are emitted at intervals (every 30-50 tool calls) or at thresholds to avoid flooding the metrics file. This balances visibility with noise reduction.

3. **Existing Infrastructure Is Sufficient** - No new file formats, no new APIs needed. The existing `writeMetric()` function and worker detection logic provide all required infrastructure.

**Answer to Investigation Question:**

The 4 metrics were implemented using the exact same pattern as action_ratio:
- `tool_failure_rate`: Emits when consecutive failures >= 3, resets on success
- `context_usage`: Emits every 50 tool calls with estimated token percentage
- `time_in_phase`: Emits every 30 tool calls with minutes since session start
- `commit_gap`: Emits every 30 tool calls with minutes since last git commit

Each metric uses `writeMetric()` with the standard format and includes relevant details for debugging/analysis.

---

## Structured Uncertainty

**What's tested:**

- TypeScript syntax is valid (verified: diff review shows proper syntax)
- Metrics use same format as existing action_ratio (verified: code structure matches)
- Worker detection continues to work (verified: detectWorkerSession unchanged)

**What's untested:**

- Token estimation accuracy (not benchmarked against actual token counts)
- Threshold values (3 failures, 80% context, 15 min phase, 30 min commit) need production tuning
- Integration with downstream consumers (dashboard, daemon recovery) not validated

**What would change this:**

- If OpenCode exposes actual token counts via API, context_usage could be more accurate
- If metrics cause performance issues, emission frequency may need adjustment
- If thresholds are wrong, agents may get too many or too few warnings

---

## Implementation Recommendations

**Purpose:** N/A - this IS the implementation, following the architect's design.

### Approach Taken

**Extend coaching.ts with worker-specific tracking** - Added WorkerHealthState interface, trackWorkerHealth() function, and modified tool.execute.after to call it for worker sessions.

**Why this approach:**
- Follows existing patterns in the file
- Uses existing writeMetric() infrastructure
- Maintains separation between worker and orchestrator metrics

**Trade-offs accepted:**
- Token estimation is approximate (no API for accurate counts)
- Metrics emitted periodically, not on every threshold cross

---

## References

**Files Examined:**
- `plugins/coaching.ts` - Full implementation context
- `.kb/investigations/2026-01-17-inv-design-agent-self-health-context.md` - Architect design
- `.orch/workspace/og-arch-design-agent-self-17jan-daa6/SYNTHESIS.md` - Architect session summary

**Commands Run:**
```bash
# Check TypeScript syntax
npx tsc --noEmit plugins/coaching.ts

# Review diff
git diff plugins/coaching.ts
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-17-inv-design-agent-self-health-context.md` - Design source
- **Beads:** `bd show orch-go-uf7zf` - This issue

---

## Investigation History

**2026-01-17 01:35:** Investigation started
- Initial question: How to add 4 worker metrics using action_ratio pattern
- Context: Implementing Phase 1 of architect's Agent Self-Health Context Injection design

**2026-01-17 01:40:** Found existing patterns
- Read coaching.ts, identified writeMetric pattern and worker detection
- Determined implementation approach

**2026-01-17 01:45:** Implementation completed
- Added WorkerHealthState interface (lines 450-463)
- Added estimateWorkerTokenUsage() and trackWorkerHealth() functions (lines 987-1120)
- Modified tool.execute.after to track worker health instead of skipping (lines 1362-1391)

**2026-01-17 01:50:** Investigation completed
- Status: Complete
- Key outcome: 4 worker-specific metrics now emitting to coaching-metrics.jsonl for worker sessions
