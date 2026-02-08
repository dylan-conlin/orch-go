<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Orchestrator completions can't be correlated with spawns because they lack beads_id - they use workspace for identification instead.

**Evidence:** Orchestrator `agent.completed` events have `workspace` but no `beads_id`. The stats code only correlates via `beads_id`, so 0/41 orchestrator spawns show as completed.

**Knowledge:** Coordination skills (orchestrator/meta-orchestrator) are untracked by design, so they use workspace-based identification rather than beads tracking.

**Next:** Add workspace-based correlation to stats aggregation for orchestrator completions.

**Promote to Decision:** recommend-no (tactical bug fix, not architectural change)

---

# Investigation: Orch Stats Miscounts Orchestrator Meta

**Question:** Why does `orch stats` show 0% completion rate for orchestrator/meta-orchestrator skills when they do complete?

**Started:** 2026-01-07
**Updated:** 2026-01-07
**Owner:** Spawned agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Orchestrator completions use workspace, not beads_id

**Evidence:** Examining `agent.completed` events:

Worker completions:
```json
{"type":"agent.completed","data":{"beads_id":"orch-go-tnw","forced":true,"reason":"..."}}
```

Orchestrator completions:
```json
{"type":"agent.completed","data":{"orchestrator":true,"workspace":"og-orch-fix-critical-bugs-06jan-6189","reason":"..."}}
```

**Source:** `~/.orch/events.jsonl` - grep for `agent.completed` with `orchestrator:true`

**Significance:** Stats code correlates completions to spawns via `beads_id`, but orchestrator completions don't have one.

---

### Finding 2: Stats correlation only uses beads_id

**Evidence:** In `stats_cmd.go:325-367`, the `agent.completed` handler:
```go
case "agent.completed":
    var beadsID string
    if data := event.Data; data != nil {
        if b, ok := data["beads_id"].(string); ok && b != "" {
            beadsID = b
            // Find session with matching beads_id
            for sid, spawnBeadsID := range spawnBeadsIDs {
                if spawnBeadsID == beadsID {
                    sessionID = sid
                    break
                }
            }
        }
    }
```

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/stats_cmd.go:325-367`

**Significance:** When `beads_id` is empty (orchestrators), `sessionID` stays empty and no completion is recorded.

---

### Finding 3: Spawns DO track workspace

**Evidence:** Orchestrator spawn events include workspace:
```json
{"type":"session.spawned","data":{
  "skill":"orchestrator",
  "workspace":"og-orch-fix-critical-bugs-06jan-6189",
  "beads_id":"orch-go-untracked-1767745571"
}}
```

**Source:** `~/.orch/events.jsonl` - orchestrator spawn events

**Significance:** Workspace is present in both spawns and completions - can use it for correlation.

---

## Synthesis

**Key Insights:**

1. **Correlation gap** - The code has the data it needs (workspace in spawns and completions) but doesn't use it

2. **Untracked is intentional** - Orchestrators use `--no-track` by design (they're sessions, not discrete tasks), so beads_id is a placeholder

3. **0% is misleading** - Many orchestrators DO complete (17 in the events sample), but stats shows 0% because correlation fails

**Answer to Investigation Question:**

The 0% completion rate for orchestrator/meta-orchestrator is a correlation bug, not actual behavior. Orchestrator completions use `workspace` for identification, but stats only correlates via `beads_id`. Fix: Add workspace-based correlation for `agent.completed` events.

---

## Structured Uncertainty

**What's tested:**

- ✅ Worker completions have beads_id (verified: grep events.jsonl)
- ✅ Orchestrator completions have workspace but no beads_id (verified: grep events.jsonl)
- ✅ Spawns include workspace field (verified: grep events.jsonl)
- ✅ 17+ orchestrator completion events exist in the log (verified: count)

**What's untested:**

- ⚠️ Fix implementation (not yet coded)
- ⚠️ Impact on overall metrics calculation

**What would change this:**

- If orchestrators were changed to use beads tracking, this fix would be unnecessary
- If workspace wasn't in spawn events, we'd need a different correlation mechanism

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Add workspace-based correlation for orchestrator completions**

**Why this approach:**
- Uses data already present in events (no schema changes)
- Minimal code change (add tracking map + lookup)
- Preserves existing beads_id correlation for workers

**Trade-offs accepted:**
- Slightly more memory usage (additional map for workspace tracking)
- Dual correlation path (beads_id for workers, workspace for orchestrators)

**Implementation sequence:**
1. Add `spawnWorkspaces` map: `workspace -> session_id`
2. In `session.spawned` handler, extract and store workspace
3. In `agent.completed` handler, try workspace correlation when beads_id fails
4. Update tests to verify orchestrator completions are counted

### Alternative Approaches Considered

**Option B: Change orchestrator spawns to use tracking**
- **Pros:** Single correlation mechanism
- **Cons:** Orchestrators intentionally don't use beads tracking (sessions vs tasks)
- **When to use instead:** If decision changes on orchestrator tracking model

**Option C: Use session_id for orchestrators**
- **Pros:** More direct correlation
- **Cons:** Orchestrator spawns in tmux mode don't have session_id
- **When to use instead:** If headless-only orchestrators

**Rationale for recommendation:** Option A is the least invasive fix that works with the existing event structure.

---

### Implementation Details

**What to implement first:**
1. Add workspace tracking map in `aggregateStats`
2. Extract workspace from spawn events
3. Use workspace to correlate orchestrator completions

**Things to watch out for:**
- ⚠️ Orchestrator completions may be counted as "untracked" - need to handle `orchestrator: true` flag
- ⚠️ Workspace names may not be unique across all time - OK within 7 day window
- ⚠️ Need to update tests for the new correlation logic

**Areas needing further investigation:**
- Should orchestrator "completions" be called "session ends" instead?
- Consider adding explicit session events vs agent events distinction

**Success criteria:**
- ✅ `orch stats` shows non-zero completion rate for orchestrators
- ✅ Rate accurately reflects actual completions from events.jsonl
- ✅ Existing worker metrics unchanged
- ✅ All tests pass

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/stats_cmd.go` - Stats aggregation logic
- `~/.orch/events.jsonl` - Event data for analysis

**Commands Run:**
```bash
# Find orchestrator spawn events
cat ~/.orch/events.jsonl | grep '"session.spawned"' | grep '"orchestrator"' | tail -10

# Find orchestrator completion events
cat ~/.orch/events.jsonl | grep '"orchestrator":true' | head -20

# Run stats with and without untracked
go run ./cmd/orch stats --days 7
go run ./cmd/orch stats --days 7 --include-untracked
```

**Related Artifacts:**
- **Issue:** orch-go-zb3qn - orch stats miscounts orchestrator/meta-orchestrator as failures

---

## Investigation History

**2026-01-07 14:49:** Investigation started
- Initial question: Why does orch stats show 0% for orchestrators?
- Context: Daemon spawned this issue

**2026-01-07 14:55:** Root cause identified
- Correlation uses beads_id only
- Orchestrators use workspace instead
- Fix path: add workspace correlation

**2026-01-07 15:00:** Investigation complete
- Status: Complete
- Key outcome: Add workspace-based correlation for orchestrator completions
