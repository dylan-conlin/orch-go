# Decision: Daemon Recomputes State Each Poll Cycle

**Status:** Accepted
**Date:** 2025-12-31
**Context:** Daemon state management for long-running overnight processing

---

## Summary

The daemon recomputes its capacity and queue state from authoritative sources (OpenCode API, beads) at the start of each poll cycle, rather than relying on internally tracked state. This "stateless per cycle" approach ensures self-healing behavior.

## Context

During December 2025, the daemon exhibited several state drift bugs:
1. **Stale capacity** - WorkerPool tracked slots internally but never reconciled with actual OpenCode sessions
2. **Zombie counting** - Completed agents still counted toward capacity because sessions persist after agent completion
3. **Closed issue drift** - Sessions with closed beads issues weren't excluded from capacity

All these bugs shared a root cause: **internal state diverging from external sources of truth**.

## Decision

**Daemon reconciles with external sources at each poll cycle rather than trusting internal state.**

Implementation:
1. **Capacity reconciliation**: `Pool.Reconcile(actualCount)` called at cycle start
   - Queries OpenCode API for actual session count
   - Filters by 30-minute recency, open beads issues, tracked status
   - Syncs pool's internal count to match reality

2. **Queue recomputation**: `bd list --status open` called each cycle
   - Fresh list of spawnable issues every 60 seconds
   - No caching of issue state between cycles

3. **Closed issue detection**: Batch query to beads for issue status
   - Sessions with closed beads issues don't count toward capacity
   - Enables immediate slot recovery after `orch complete`

## Rationale

1. **Self-healing behavior** - State drift is automatically corrected within one poll cycle (60s)

2. **Simplicity over optimization** - Polling is simpler than event-driven state sync; 60s latency is acceptable for overnight processing

3. **Authoritative sources over local state** - OpenCode and beads are the sources of truth; the daemon should defer to them

4. **Long-running resilience** - Daemons run for hours/days; any state drift would compound without reconciliation

## Consequences

**Positive:**
- Bugs in state tracking self-heal automatically
- No complex event-driven synchronization needed
- Works correctly even if daemon misses events (crash, network issues)

**Negative:**
- Increased API calls (OpenCode + beads each cycle)
- 60-second latency before capacity updates (acceptable for overnight work)
- Beads daemon required for fast batch queries

**Neutral:**
- State is "eventually consistent" rather than immediately consistent
- Monitoring/debugging uses external tools (`orch status`) not internal daemon state

## Key Insight

The daemon's internal state should be treated as a **cache for the current cycle only**, not as authoritative. This mental model prevents state drift bugs:

> "When in doubt, recompute. Don't trust yesterday's state."

## References

This decision consolidates findings from:
- `.kb/investigations/2025-12-26-inv-daemon-capacity-count-goes-stale.md` - Pool reconciliation
- `.kb/investigations/2025-12-26-debug-daemon-capacity-stale-after-complete.md` - Closed issue checking
- `.kb/investigations/2025-12-26-inv-daemon-capacity-count-stuck-while.md` - Recency filtering

Key implementation locations:
- `pkg/daemon/pool.go:Reconcile()` - Pool state sync
- `pkg/daemon/daemon.go:DefaultActiveCount()` - Session counting with filters
- `pkg/daemon/daemon.go:getClosedIssuesBatch()` - Beads status checking
