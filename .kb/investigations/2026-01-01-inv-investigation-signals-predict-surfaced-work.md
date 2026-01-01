<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The system already has rich signal infrastructure but lacks action logging - we can identify 5 candidate signals (blocking count, cost, trend, resolution status, focus alignment) from existing data without new instrumentation.

**Evidence:** Analyzed next.go, learning.go, attempts.go, patterns/analyzer.go - found GapTracker, FixAttemptStats, RecommendationType with priority/focus signals already computed but no action outcome logging.

**Knowledge:** The missing piece is action logging (what orchestrator did after seeing surfaced work), not signal infrastructure. Existing signals (priority, focus alignment, retry count, trend) are already computed but never correlated with outcomes.

**Next:** Create a lightweight action logger to track what work gets acted on vs ignored, then measure which existing signals predict action. Defer evaluation layer until data proves which signals matter.

---

# Investigation: What Signals Predict Which Surfaced Work Gets Acted On?

**Question:** What signals predict which surfaced work the orchestrator acts on vs ignores? Should we build an evaluation layer, and if so, what signals to use?

**Started:** 2026-01-01
**Updated:** 2026-01-01
**Owner:** Investigation agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Extracted-From:** Research question from closed epic orch-go-idmr

---

## Findings

### Finding 1: Five Signal Hypotheses Are Testable with Existing Infrastructure

**Evidence:** The SPAWN_CONTEXT posed these hypotheses: blocking count, cost (tokens/time), trend (accelerating vs stable), resolution status. Reviewing the codebase shows:

| Signal | Already Captured? | Where |
|--------|-------------------|-------|
| Blocking count | Yes | beads.Issue.BlockedBy, Issue.BlockerFor |
| Cost (tokens/time) | Partial | usage.go tracks usage, but not per-issue |
| Trend (accelerating vs stable) | Yes | learning.go:determineTrend() computes this |
| Resolution status | Yes | GapEvent.Resolution in gap-tracker.json |
| Focus alignment | Yes (bonus) | next.go:matchesFocusGoal() already checks this |

**Source:** pkg/spawn/learning.go:591-617 (determineTrend), cmd/orch/next.go:253-265 (focus alignment), pkg/verify/attempts.go:29-54 (retry patterns)

**Significance:** 4 of 5 hypothesized signals are already computed. This means we don't need new signal infrastructure - we need action outcome logging to correlate signals with what gets acted on.

---

### Finding 2: The orch next Command Already Ranks Work by Multiple Signals

**Evidence:** The next.go command synthesizes work into 4 priority tiers:
1. **BLOCKER** - Issues with persistent failures (from FixAttemptStats)
2. **FOCUS** - Focus-aligned ready work (from focus.Store.CheckDrift)
3. **MAINTENANCE** - Recurring gaps (from GapTracker.FindRecurringGaps)
4. **BACKLOG** - Standard ready work (from beads.FallbackReady)

Each recommendation includes:
- Priority (1-5 numeric)
- FocusMatch (boolean)
- Reason (string explanation)
- Command (suggested action)

**Source:** cmd/orch/next.go:23-60 (RecommendationType, Recommendation struct), lines 111-161 (runNextSynth function)

**Significance:** The ranking algorithm exists. What is missing is tracking whether the orchestrator actually follows these recommendations or ignores them.

---

### Finding 3: Action Outcome Logging is the Missing Link

**Evidence:** Searched for action outcome tracking:
- events.jsonl tracks: session.spawned, agent.completed, agent.abandoned, daemon.spawn
- gap-tracker.json tracks: Gap occurrences with query, skill, resolution
- action-log.jsonl tracks: Tool invocations (Read, Bash, etc.) and outcomes (success/empty/error)

What is NOT tracked:
- Whether orch next recommendations were acted on
- Whether surfaced work (blockers, gaps, focus-aligned) was spawned or ignored
- Time between surfacing and action

**Source:** pkg/events/logger.go, pkg/spawn/learning.go, pkg/patterns/analyzer.go

**Significance:** We have rich "what was surfaced" data and "what happened eventually" data, but no correlation between them. The daemon auto-spawns triage:ready issues, bypassing human decision logging. Manual spawns don't log why something was chosen.

---

### Finding 4: Resolution Status Already Distinguishes Acted vs Not-Acted

**Evidence:** The GapEvent struct has a Resolution field:
- Values: "proceeded", "added_knowledge", "created_issue", "aborted"

And FindRecurringGaps() filters unresolved gaps - gaps with Resolution="" were ignored; those with Resolution="added_knowledge" or "created_issue" were acted on.

**Source:** pkg/spawn/learning.go:49-54 (Resolution field), lines 320-327 (filter logic)

**Significance:** For gaps specifically, resolution status IS our action outcome signal.

---

### Finding 5: Retry Patterns Provide Natural Experiment for Signal Effectiveness

**Evidence:** The FixAttemptStats system tracks:
- SpawnCount - How many times an issue was attempted
- AbandonedCount - How many failures
- CompletedCount - Successes
- IsPersistentFailure() - 2+ spawns, 0 completions, 2+ abandons

Issues with high retry counts that eventually succeed vs fail provide data on whether surfacing warnings helped.

**Source:** pkg/verify/attempts.go:29-54

**Significance:** This is a natural A/B test: did issues with retry warnings get handled differently (e.g., sent to systematic-debugging or reliability-testing) vs issues without warnings?

---

## Synthesis

**Key Insights:**

1. **Signal infrastructure is mature** - Trend detection, focus alignment, priority, and retry patterns are already computed. Building an "evaluation layer" for signals would be redundant.

2. **Action logging is the gap** - We know what was surfaced and what happened eventually, but not whether the surfaced recommendations led to the actions. The daemon bypasses decision logging entirely.

3. **Gap resolution is a proxy for action** - The Resolution field on GapEvents provides a natural experiment: unresolved gaps = ignored, resolved = acted on. We can correlate gap attributes with resolution rates.

**Answer to Investigation Question:**

The signals that predict action are likely already being computed (focus alignment, priority, trend, retry count). The reason we cannot identify them is lack of action outcome logging, not lack of signal computation.

**Recommendation:** Do not build an "evaluation layer" for signals. Instead:
1. Add action logging to manual spawns (which issue was spawned and why)
2. Mine existing gap resolution data to find which gap attributes predict resolution
3. Compare issues with/without retry warnings to measure warning effectiveness

---

## Structured Uncertainty

**What is tested:**

- Signal infrastructure exists (reviewed code: next.go, learning.go, attempts.go)
- Action outcome logging is missing (grep for "acted on", "ignored", "chose" - no results)
- Gap resolution provides proxy for action (GapEvent.Resolution field)
- Retry patterns create natural experiment (FixAttemptStats tracks outcomes)

**What is untested:**

- Which signals actually predict action (no data correlation done)
- Whether orchestrators read orch next output (no usage telemetry)
- Whether daemon-spawned vs manual-spawned issues have different success rates

**What would change this:**

- Finding would be wrong if there IS action outcome logging somewhere I did not find
- Finding would be wrong if signals do not matter because orchestrator ignores them
- Finding would be wrong if daemon handles all spawning (then manual decision data does not exist)

---

## Implementation Recommendations

### Recommended Approach

**Log-First Before Evaluation** - Add lightweight action logging before building evaluation layer.

**Why this approach:**
- Avoids building evaluation layer for signals that may not predict action
- Uses existing GapEvent.Resolution as starting point
- Minimal code change: just add event logging to spawn decisions

**Trade-offs accepted:**
- Delays evaluation layer until data is collected (acceptable - need data first)
- Will not improve predictions immediately (acceptable - premature optimization)

**Implementation sequence:**
1. Add decision.spawn event to events.jsonl when manual spawn occurs (captures beads_id, source: {next, review, manual}, signals seen)
2. Add decision.skip event when orchestrator sees recommendation but does not act (harder - may need UI instrumentation)
3. After 2 weeks, analyze which signals correlate with action vs skip

### Alternative Approaches Considered

**Option B: Build evaluation layer now**
- **Pros:** Immediate signal ranking, can test hypotheses
- **Cons:** No action outcome data to validate; would be ranking signals blindly
- **When to use instead:** If we have external evidence (papers, other systems) for signal weights

**Option C: Mine gap resolution data only**
- **Pros:** No new code needed; data exists in gap-tracker.json
- **Cons:** Only covers gaps, not all surfaced work; limited sample size
- **When to use instead:** Quick win before building action logging

**Rationale for recommendation:** Log-First is the scientific approach: collect observations before building models. Gap resolution mining is a good parallel track.

---

### Implementation Details

**What to implement first:**
- Add decision.spawn event to manual spawn path in main.go (near line 1515 where gaps are recorded)
- Include: beads_id, source (manual/next/review), signals_seen (focus_aligned, priority, has_retry_warning, trend)

**Things to watch out for:**
- Do not add logging overhead to daemon (it does not make decisions, just processes queue)
- Action logging must be opt-in or minimal to avoid event log bloat
- "Decision to skip" is harder to capture than "decision to act"

**Areas needing further investigation:**
- How to capture "saw but did not act" decisions (may need UI/TUI instrumentation)
- Whether focus alignment is computed before or after decision (timing matters)
- How beads label changes (triage:review to triage:ready) correlate with action

**Success criteria:**
- Can query events.jsonl for decision.spawn with source and signals_seen
- After 2 weeks, can correlate signals_seen with success/failure outcomes
- Can identify top 2-3 signals that predict action (measured, not hypothesized)

---

## References

**Files Examined:**
- cmd/orch/next.go - Work recommendation synthesis with signal ranking
- cmd/orch/daemon.go - Autonomous overnight processing without decision logging
- cmd/orch/review.go - Agent completion review workflow
- pkg/spawn/learning.go - Gap tracking with resolution status and trend detection
- pkg/verify/attempts.go - Retry pattern detection with outcome tracking
- pkg/patterns/analyzer.go - Behavioral pattern detection from action log

**Related Artifacts:**
- **Epic:** orch-go-idmr (closed) - Original source of this research question

---

## Investigation History

**2026-01-01:** Investigation started
- Initial question: What signals predict which surfaced work gets acted on?
- Context: Research question from closed epic orch-go-idmr

**2026-01-01:** Found signal infrastructure is mature
- Discovered next.go already computes focus alignment, priority, trend, retry warnings
- Identified GapEvent.Resolution as proxy for action outcome

**2026-01-01:** Investigation completed
- Status: Complete
- Key outcome: Do not build evaluation layer; add action logging first, then measure
