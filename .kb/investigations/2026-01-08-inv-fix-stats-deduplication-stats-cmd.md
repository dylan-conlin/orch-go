## Summary (D.E.K.N.)

**Delta:** Stats command now deduplicates completions by beads_id, counting unique completions instead of completion events.

**Evidence:** Added tests proving deduplication works (TestAggregateStatsDeduplicationByBeadsID, TestAggregateStatsDeduplicationMixedEventTypes), all pass.

**Knowledge:** Multiple `agent.completed` events can exist for the same beads_id (from retries, orch complete, etc). Before this fix, each event was counted separately, inflating completion counts.

**Next:** Close issue - implementation complete and tested.

**Promote to Decision:** recommend-no (tactical bug fix, not architectural)

---

# Investigation: Fix Stats Deduplication Stats Cmd

**Question:** How to deduplicate completion stats by beads_id to show unique completions instead of completion events?

**Started:** 2026-01-08
**Updated:** 2026-01-08
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Multiple completion events per beads_id exist

**Evidence:** Analysis of events.jsonl shows many beads_ids with multiple completion events:
- `orch-go-578d` has 7 completion events
- `orch-go-yhag`, `orch-go-kive`, `orch-go-i914`, `orch-go-fwmw` each have 6 events
- In 7-day window: 301 completion events but only 275 unique beads_ids

**Source:** `grep "agent.completed" ~/.orch/events.jsonl | jq -r '.data.beads_id // empty' | sort | uniq -c | sort -rn`

**Significance:** The completion count was inflated by ~10% due to duplicate events. This causes reported completion rate to be lower than actual.

---

### Finding 2: Both session.completed and agent.completed events exist

**Evidence:** Two event types contribute to completion counting:
- `session.completed` - ~3 events in 7-day window
- `agent.completed` - ~301 events in 7-day window
- Some beads_ids have BOTH event types (e.g., `orch-go-gaf8`)

**Source:** `stats_cmd.go` lines 328-435

**Significance:** Both event types needed to share a deduplication set to prevent double-counting across event types.

---

### Finding 3: Orchestrator completions use workspace for correlation

**Evidence:** Orchestrator completions don't have beads_id (they're untracked by design). They use the workspace field for correlation with their spawn events.

**Source:** `stats_cmd.go` lines 263-264, 379-387

**Significance:** Deduplication key needed to handle both beads_id and workspace-based identification.

---

## Synthesis

**Key Insights:**

1. **Single deduplication set across event types** - The fix uses a single `completedBeadsIDs` map shared between `session.completed` and `agent.completed` handlers, ensuring a beads_id is only counted once regardless of which event type reports it first.

2. **Fallback key for orchestrators** - When beads_id is empty, the code uses `"ws:" + workspace` as the deduplication key, properly handling orchestrator completions that identify by workspace.

3. **Duration uses first event** - Since we're deduplicating, the duration calculation uses the first completion event's timestamp. The "latest" approach was considered but adds complexity without significant benefit.

**Answer to Investigation Question:**

The fix adds a `completedBeadsIDs` map to track which beads_ids have been counted. Before incrementing completion counters, the code checks if the deduplication key (beads_id or workspace) has already been processed. This ensures each unique completion is counted exactly once, regardless of how many completion events exist for it.

---

## Structured Uncertainty

**What's tested:**

- ✅ Deduplication works for multiple agent.completed events with same beads_id (TestAggregateStatsDeduplicationByBeadsID)
- ✅ Deduplication works across session.completed and agent.completed for same beads_id (TestAggregateStatsDeduplicationMixedEventTypes)
- ✅ All existing stats tests still pass (no regression)

**What's untested:**

- ⚠️ Performance impact with very large events.jsonl files (not benchmarked)
- ⚠️ Duration calculation accuracy with latest vs first event

**What would change this:**

- If events.jsonl format changes to include new completion event types
- If beads_id semantics change (e.g., reusing beads_ids)

---

## Implementation Recommendations

### Recommended Approach ⭐

**Deduplication via shared map** - Track processed beads_ids in a single map shared across all completion event handlers.

**Why this approach:**
- Simple and efficient (single map lookup per event)
- Handles all edge cases (missing beads_id, mixed event types)
- Minimal code changes to existing logic

**Trade-offs accepted:**
- Duration uses first event timestamp (not latest)
- Map grows with unique completions (memory)

**Implementation sequence:**
1. Add `completedBeadsIDs` map
2. Check map before counting completion
3. Mark beads_id as counted after processing

---

## References

**Files Examined:**
- `cmd/orch/stats_cmd.go` - Main implementation
- `cmd/orch/stats_test.go` - Added deduplication tests

**Commands Run:**
```bash
# Count unique vs total completion events
grep "agent.completed" ~/.orch/events.jsonl | jq -r '.data.beads_id // empty' | sort | uniq -c

# Verify expected unique count
cutoff=$(date -v-7d +%s) && grep "agent.completed" ~/.orch/events.jsonl | jq -r "select(.timestamp >= $cutoff) | .data.beads_id" | sort -u | wc -l
```

---

## Investigation History

**2026-01-08 14:10:** Investigation started
- Initial question: Fix stats double-counting completion events
- Context: Constraint noted that stats double-count completions (26 duplicates in 7-day sample)

**2026-01-08 14:25:** Implementation completed
- Added deduplication via `completedBeadsIDs` map
- All tests pass including new deduplication-specific tests
- Status: Complete
