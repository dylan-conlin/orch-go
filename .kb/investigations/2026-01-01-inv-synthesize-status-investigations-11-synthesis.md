## Summary (D.E.K.N.)

**Delta:** `orch status` evolved through 11 investigations to address a core architectural challenge: detecting "live" agents across a four-layer state system (OpenCode memory, OpenCode disk, registry, tmux).

**Evidence:** Investigations document the progression from 339+ stale sessions to accurate counts, 12.2s to ~1s performance, and discovery of multiple filtering/reconciliation mechanisms needed.

**Knowledge:** Liveness detection requires activity-based heuristics (30-min idle), not existence checks; session titles must embed beads IDs for matching; OpenCode's `x-opencode-directory` header drastically changes API behavior; batch/parallel beads calls are essential for performance.

**Next:** Archive 9 of 11 investigations as superseded (this synthesis captures their findings); keep 2 (performance, architecture) as still-relevant references.

---

# Investigation: Synthesis of 11 Status-Related Investigations

**Question:** What patterns, lessons, and remaining gaps emerge from consolidating 11 investigations about `orch status` functionality?

**Started:** 2026-01-01
**Updated:** 2026-01-01
**Owner:** Synthesis agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage -->
**Supersedes:** The following 9 investigations are now subsumed:
- `.kb/investigations/2025-12-21-inv-investigate-orch-status-showing-stale.md`
- `.kb/investigations/2025-12-21-inv-orch-status-showing-stale-sessions.md`
- `.kb/investigations/2025-12-22-debug-orch-status-stale-sessions.md`
- `.kb/investigations/2025-12-22-inv-update-orch-status-use-islive.md`
- `.kb/investigations/2025-12-23-inv-orch-status-can-detect-active.md`
- `.kb/investigations/2025-12-23-inv-orch-status-shows-active-agents.md`
- `.kb/investigations/2025-12-24-inv-fix-status-filter-test-expects.md`
- `.kb/investigations/2025-12-28-inv-orch-status-detect-dead-orphaned.md` (incomplete template)
- `.kb/investigations/2025-12-30-inv-orch-status-shows-stale-architect.md`

**Still Relevant (not superseded):**
- `.kb/investigations/2025-12-20-inv-enhance-status-command-swarm-progress.md` - Original feature design
- `.kb/investigations/2025-12-23-inv-orch-status-takes-11-seconds.md` - Performance optimization reference

---

## Executive Summary

Over 10 days (Dec 20-30, 2025), the `orch status` command was the subject of 11 investigations addressing:

| Problem Domain | Investigations | Key Fix |
|----------------|----------------|---------|
| Stale session display | 5 | Activity-based filtering (30-min idle), not existence checks |
| Session-to-beads matching | 2 | Embed beads ID in session title format |
| Performance | 1 | Batch beads calls, parallel goroutines (12s → 1s) |
| Filtering inconsistencies | 2 | Closed-issue filtering, test alignment |
| Feature enhancement | 1 | Added swarm metrics, account usage, JSON output |

---

## Findings

### Finding 1: Four-Layer Architecture Creates State Reconciliation Complexity

**Evidence:** The 2025-12-21 investigation identified four independent state layers:
1. **OpenCode in-memory** (ACPSessionManager.sessions) - Transient, cleared on server restart
2. **OpenCode disk** (~/.local/share/opencode/storage/session/) - Persistent, hundreds of sessions accumulate
3. **orch registry** (~/.orch/agent-registry.json) - Tracks agent metadata (beads ID, skill, status)
4. **tmux windows** - Ground truth for non-headless agents

**Source:** `.kb/investigations/2025-12-21-inv-investigate-orch-status-showing-stale.md`

**Significance:** Each layer has different persistence semantics. `orch status` must aggregate and reconcile across all four. Ghost sessions appear when any layer has stale data without cleanup.

---

### Finding 2: OpenCode API Header Behavior is Counterintuitive

**Evidence:** 
- Without `x-opencode-directory` header: Returns only in-memory sessions (2-4 typically)
- With `x-opencode-directory` header: Returns ALL historical disk sessions for that directory (238-339+)

This caused `orch status` to display hundreds of stale sessions as "active."

**Source:** `.kb/investigations/2025-12-21-inv-orch-status-showing-stale-sessions.md`

**Significance:** Documented as constraint via `kn constrain`. Fix: Call `ListSessions("")` (no header) to get only in-memory sessions.

---

### Finding 3: Liveness Must Be Activity-Based, Not Existence-Based

**Evidence:** Three investigations converged on this insight:
- `SessionExists()` returns true for ANY persisted session (339+ on disk)
- OpenCode keeps sessions in memory 6+ hours after agents exit
- SSE busy/idle detection has false positives (agents go idle during normal operation)

**Solution:** Use 30-minute idle threshold based on `time.updated`. The messages endpoint (`/session/{id}/message`) provides authoritative state: `finish: ""` + `completed: 0` = actively processing.

**Source:** 
- `.kb/investigations/2025-12-22-debug-orch-status-stale-sessions.md`
- `.kb/investigations/2025-12-23-inv-orch-status-can-detect-active.md`

**Significance:** This is now embedded in `IsSessionActive()` and `IsSessionProcessing()` methods.

---

### Finding 4: Session Title Format Must Match Extraction Expectations

**Evidence:** Session titles were just workspace names (e.g., "og-debug-orch-status-23dec") but `extractBeadsIDFromTitle()` expected `[beads-id]` pattern. Result: 0 agents matched, all shown as "phantom."

**Fix:** Added `formatSessionTitle(workspaceName, beadsID)` helper to ensure titles use format: `"workspace-name [beads-id]"`

**Source:** `.kb/investigations/2025-12-23-inv-orch-status-shows-active-agents.md`

**Significance:** Tmux windows already used this format. The fix aligned OpenCode session titles with the established pattern.

---

### Finding 5: Sequential Subprocess Calls are Performance Bottleneck

**Evidence:**
- OpenCode API: 15-18ms for 54 sessions
- Each `bd show` call: ~140ms
- Each `bd comments` call: ~95ms
- 3 calls × 37 agents = ~11 seconds total

**Fix:** Batch fetching (`bd list --status open --json`) and parallel goroutines for comments reduced time from 12.2s to 1.05s (11× improvement).

**Source:** `.kb/investigations/2025-12-23-inv-orch-status-takes-11-seconds.md`

**Significance:** Performance optimization reference that should inform future beads-heavy operations.

---

### Finding 6: Lightweight Function Variants Must Preserve Filtering

**Evidence:** `getCompletionsForSurfacing()` was a "lightweight" version of `getCompletionsForReview()` but omitted `filterClosedIssues()` call. Result: `orch status` showed architect recommendations for closed issues.

**Fix:** One-line addition of `filterClosedIssues()` to the return.

**Source:** `.kb/investigations/2025-12-30-inv-orch-status-shows-stale-architect.md`

**Significance:** Pattern lesson: When creating performance-optimized variants, filtering logic must be preserved alongside verification.

---

## Synthesis

### Key Patterns Across Investigations

1. **Root Cause Convergence** - Multiple investigations (Dec 21, 22, 23) circled the same core issue: existence checks don't mean liveness. The system needed activity-based heuristics.

2. **Incremental Fix Layering** - Each investigation fixed one aspect:
   - Dec 21: Identified four-layer architecture
   - Dec 21: Fixed API header usage
   - Dec 22: Added activity-based liveness
   - Dec 23: Fixed session title format, added processing detection, optimized performance
   - Dec 24: Test alignment (already fixed)
   - Dec 30: Closed-issue filtering

3. **Documentation Gaps** - Investigation `2025-12-28-inv-orch-status-detect-dead-orphaned.md` was incomplete (just template). The question was likely addressed by prior fixes.

4. **Investigation Velocity** - 11 investigations in 10 days indicates either: (a) complex problem domain requiring iteration, or (b) insufficient upfront design. Given the four-layer architecture complexity, (a) seems more likely.

### Current State of `orch status`

Based on all investigations, `orch status` now:
- Uses in-memory sessions only (no `x-opencode-directory` header)
- Filters by 30-minute activity threshold
- Checks `IsSessionProcessing()` for active generation detection
- Uses consistent session title format with beads ID
- Batches beads calls for ~1s performance
- Filters closed issues from architect recommendations

### Remaining Gaps/Uncertainties

1. **Edge case: Long-paused agents** - 30-minute idle threshold may false-negative agents that are legitimately paused (rare but possible)
2. **Headless agent visibility** - Headless agents rely solely on OpenCode session + registry, not tmux. May need additional monitoring.
3. **OpenCode disk cleanup** - Hundreds of sessions accumulate on disk. No automatic cleanup mechanism exists.

---

## Implementation Recommendations

### Recommended Action: Archive Superseded Investigations

**Why:** 9 of 11 investigations contain knowledge now encoded in code or superseded by subsequent fixes. Keeping them active creates noise.

**Process:**
1. Move to `.kb/investigations/archived/status-stale-sessions-dec2025/`
2. Add `Superseded-By: 2026-01-01-inv-synthesize-status-investigations-11-synthesis.md` to each
3. Keep this synthesis as the canonical reference

### Investigations to Keep Active

1. **2025-12-20-inv-enhance-status-command-swarm-progress.md** - Feature design document, still valid
2. **2025-12-23-inv-orch-status-takes-11-seconds.md** - Performance pattern, reusable knowledge

---

## Constraints Extracted (for kn)

These constraints should be externalized if not already:

| Constraint | Reason |
|------------|--------|
| OpenCode `x-opencode-directory` header returns ALL disk sessions | API behavior is counterintuitive |
| Use activity time (30-min idle) for liveness, not existence | Sessions persist indefinitely |
| Session titles must include `[beads-id]` pattern | Matching logic expects this format |
| Batch beads CLI calls where possible | O(N) subprocess calls are slow |
| Lightweight function variants must preserve filtering | Filtering != verification |

---

## References

**Investigations Analyzed:**
1. `2025-12-20-inv-enhance-status-command-swarm-progress.md` - Feature enhancement
2. `2025-12-21-inv-investigate-orch-status-showing-stale.md` - Four-layer architecture
3. `2025-12-21-inv-orch-status-showing-stale-sessions.md` - API header fix
4. `2025-12-22-debug-orch-status-stale-sessions.md` - Activity-based liveness
5. `2025-12-22-inv-update-orch-status-use-islive.md` - IsLive pattern (partial)
6. `2025-12-23-inv-orch-status-can-detect-active.md` - Messages endpoint detection
7. `2025-12-23-inv-orch-status-shows-active-agents.md` - Session title format
8. `2025-12-23-inv-orch-status-takes-11-seconds.md` - Performance optimization
9. `2025-12-24-inv-fix-status-filter-test-expects.md` - Already fixed
10. `2025-12-28-inv-orch-status-detect-dead-orphaned.md` - Incomplete template
11. `2025-12-30-inv-orch-status-shows-stale-architect.md` - Closed-issue filtering

---

## Investigation History

**2026-01-01:** Investigation started
- Initial question: Synthesize 11 status-related investigations
- Context: Topic accumulated investigations that may benefit from consolidation

**2026-01-01:** Synthesis completed
- Status: Complete
- Key outcome: Identified core patterns (four-layer architecture, activity-based liveness), extracted constraints, recommended archiving 9 of 11 investigations
