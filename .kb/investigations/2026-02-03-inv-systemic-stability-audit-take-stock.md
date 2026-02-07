<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Five interconnected stability issues stem from architectural decisions being violated or incomplete: daemon can reprocess completed work due to dedup gaps, registry shows 636 phantom "active" agents because it's designed as spawn-cache only, Work Graph flip-flop was fixed but may recur, attention API works correctly but polling patterns cause CPU concerns.

**Evidence:** Registry has 636 active / 1 completed agents; decision 2026-01-12 documents registry-as-cache design; daemon has 3-layer dedup (SpawnedIssueTracker, SessionDedupChecker, HasPhaseComplete) but gaps exist; Work Graph fix in commit 585c69e6 addressed flip-flop.

**Knowledge:** The "registry is spawn cache" decision creates expected staleness but causes dashboard confusion at scale; daemon dedup is fragmented across layers without unified ownership; render loops possible when multiple stores update at 2-5s intervals.

**Next:** Three-phase stabilization: (1) Add registry cleanup command for manual hygiene, (2) Unify daemon dedup under single ProcessedIssueCache with persistent state, (3) Throttle reactive dependencies in Work Graph page.

**Authority:** architectural - Cross-component changes affecting daemon, registry, and frontend coordination

---

# Investigation: Systemic Stability Audit Take Stock

**Question:** What cascading failures exist across daemon, registry, Work Graph, and attention API, and what stabilization path addresses them?

**Started:** 2026-02-03
**Updated:** 2026-02-03
**Owner:** og-arch-systemic-stability-audit-03feb-2f9e
**Phase:** Complete
**Next Step:** None - recommendations ready for implementation
**Status:** Complete

**Patches-Decision:** `.kb/decisions/2026-01-12-registry-is-spawn-cache.md` - validates decision is being followed but surfaces consequences at scale

---

## Findings

### Finding 1: Registry Contains 636 Phantom "Active" Agents

**Evidence:**
- `cat ~/.orch/agent-registry.json | jq '[.agents[] | .status] | group_by(.) | map({status: .[0], count: length})'`
- Result: `[{"status": "active", "count": 636}, {"status": "completed", "count": 1}]`

**Source:**
- `~/.orch/agent-registry.json` - 7645 lines, 637 total agents
- `pkg/registry/registry.go:75-110` - DESIGN CONTRACT documentation
- `.kb/decisions/2026-01-12-registry-is-spawn-cache.md`

**Significance:**
This is **expected behavior** per the 2026-01-12 decision, NOT a bug. The decision explicitly states:
- "Registry is spawn-time metadata cache, NOT lifecycle tracker"
- State transition methods (`Abandon()`, `Complete()`, `Remove()`) are deprecated
- Actual state derived from OpenCode API + beads, not registry

However, 636 phantom entries causes:
- Dashboard confusion (status shows many "active" agents that completed long ago)
- Performance drag (every status lookup iterates 637 entries)
- Cognitive overhead (hard to see actual active agents)

---

### Finding 2: Daemon Has Fragmented Dedup Across Three Layers

**Evidence:**
Three separate dedup mechanisms without unified coordination:

| Mechanism | Location | Scope | TTL | Gap |
|-----------|----------|-------|-----|-----|
| SpawnedIssueTracker | `pkg/daemon/spawn_tracker.go` | In-memory | 6h | Lost on daemon restart |
| SessionDedupChecker | `pkg/daemon/session_dedup.go` | OpenCode sessions | 6h | Only works for opencode mode, not claude CLI |
| HasPhaseComplete | `pkg/daemon/issue_adapter.go:220-287` | Beads comments | None | Comment parsing can fail |

**Source:**
- `pkg/daemon/spawn_tracker.go:35-43` - "TTL was increased from 5 minutes to 6 hours to provide backup protection for long-running agents when session-level dedup fails"
- `pkg/daemon/session_dedup.go:67-96` - `HasExistingSession()` only checks OpenCode sessions
- `pkg/daemon/issue_adapter.go:276-287` - `checkCommentsForPhaseComplete()` case-insensitive but fragile

**Significance:**
Dedup can fail when:
1. Daemon restarts (SpawnedIssueTracker cleared)
2. Agent uses claude CLI backend (SessionDedupChecker doesn't see it)
3. "Phase: Complete" comment missing or malformed (HasPhaseComplete returns false)

The combination creates a gap where completed work can be respawned after ~6 hours if issue status is still open/triage:ready.

---

### Finding 3: Work Graph Flip-Flop Was Fixed But Pattern May Recur

**Evidence:**
Commit `585c69e6` (2026-02-03) fixed the flip-flopping issue with three mechanisms:
1. AbortController - cancels in-flight requests when project changes
2. Sequence guard - ignores responses from stale requests
3. Debounce (300ms) - waits for stable project value before fetching

**Source:**
- `web/src/lib/stores/work-graph.ts:60-144` - Store with AbortController + sequence guard
- `web/src/routes/work-graph/+page.svelte:231-271` - Debounce on project change

**Significance:**
The fix is solid for project switching. However, the reactive block pattern is fragile:
```svelte
$: if ($workGraph && !$workGraph.error && $wip) {
  // Rebuilds tree when ANY of these change
}
```

Multiple stores updating at 2-5s intervals creates potential for:
- Cascading reactive updates
- Tree rebuild storms when stores update near-simultaneously
- CPU spikes when many agents are running (each agent update triggers WIP store, triggers tree rebuild)

---

### Finding 4: Attention API Is Correct But Polling Creates Load

**Evidence:**
The `/api/attention` endpoint correctly aggregates signals from multiple collectors:
- BeadsCollector (ready issues)
- GitCollector (likely-done signals)
- RecentlyClosedCollector (recently closed issues)

**Source:**
- `cmd/orch/serve_attention.go:103-228` - handleAttention implementation
- `web/src/lib/stores/attention.ts` - Frontend store (fetches once on mount, no polling)
- `web/src/routes/work-graph/+page.svelte:76` - Single fetch on mount

**Significance:**
The attention store itself is well-behaved (single fetch, no polling). However, the Work Graph page has multiple polling loops:
- `orchestratorContext.startPolling(2000)` - 2s interval
- `refreshInterval = setInterval(...)` at 5s for workGraph, wip, daemon

This means 3+ API calls every 5 seconds (6+ counting context at 2s). When backend is slow, requests can pile up.

---

### Finding 5: Registry Decision Creates Maintenance Burden

**Evidence:**
The 2026-01-12 decision explicitly accepted these consequences:
- "Registry shows stale 'active' status for completed agents (acceptable: not used for decisions)"
- "State transition methods remain as dead code until cleanup"

But the decision didn't anticipate:
- 636+ phantom entries after 3 weeks of operation
- The registry file is now 7645 lines / ~300KB
- No cleanup mechanism was implemented

**Source:**
- `.kb/decisions/2026-01-12-registry-is-spawn-cache.md:93-95`
- Follow-up item: "Optional future cleanup: Remove deprecated methods after confirming no external usage" - never done

**Significance:**
The decision was correct (registry as cache works well for lookups) but incomplete (no hygiene mechanism for long-term operation).

---

## Synthesis

**Key Insights:**

1. **Decision Consequences at Scale** - The "registry is spawn-cache" decision is architecturally sound but creates unbounded growth. 636 phantom entries is noise; at 1000+ the performance and clarity impact will be significant.

2. **Dedup Layer Fragmentation** - Three dedup mechanisms (SpawnedIssueTracker, SessionDedupChecker, HasPhaseComplete) address different failure modes but lack unified ownership. Each has gaps the others don't cover, creating a swiss cheese defense.

3. **Reactive Cascade Risk** - Multiple stores (workGraph, wip, agents, attention, daemon) updating at different intervals can trigger cascading reactive updates. The Work Graph page subscribes to most of them, creating potential for render storms.

**Answer to Investigation Question:**

The cascading failures are not arbitrary bugs but consequences of architectural decisions being violated or incomplete:

| Issue | Root Cause | Status |
|-------|-----------|--------|
| Daemon reprocessing completed work | Dedup layers have gaps, no persistent state | **Active** - needs fix |
| Registry phantom entries (636) | Decision expected, but no cleanup implemented | **Expected but degrading** |
| Work Graph flip-flop | Race condition on project switch | **Fixed** (commit 585c69e6) |
| CPU maxing from render loops | Multiple polling stores triggering reactive cascades | **Potential** - needs monitoring |

---

## Structured Uncertainty

**What's tested:**

- ✅ Registry has 636 active / 1 completed agents (verified: `jq` query)
- ✅ Daemon dedup has 3 separate mechanisms with different scopes (verified: code review)
- ✅ Work Graph flip-flop fix uses AbortController + sequence guard (verified: commit 585c69e6)
- ✅ Attention API fetches once on mount, not polling (verified: code review)

**What's untested:**

- ⚠️ CPU load correlation with multiple store updates (not profiled)
- ⚠️ Daemon respawn scenario after 6h TTL expiration (not reproduced)
- ⚠️ Registry performance impact at 1000+ entries (not benchmarked)

**What would change this:**

- If CPU spikes don't correlate with store updates, render loop theory is wrong
- If daemon respawn never happens in practice, dedup gaps are theoretical not practical
- If registry lookup remains fast at 1000+ entries, phantom entries are cosmetic only

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Registry cleanup command | implementation | Tactical feature within existing patterns |
| Unified daemon dedup | architectural | Cross-component change affecting daemon behavior |
| Store throttling | implementation | Frontend optimization within existing patterns |

### Recommended Approach ⭐

**Three-Phase Stabilization** - Address issues in order of impact and dependency

**Why this approach:**
- Phase 1 (cleanup command) is immediate relief for registry bloat
- Phase 2 (unified dedup) addresses the most dangerous failure mode (respawning completed work)
- Phase 3 (throttling) is optimization, not correctness

**Trade-offs accepted:**
- Registry cleanup is manual (not automatic) to avoid data loss
- Unified dedup requires daemon restart to take effect
- Throttling may make UI feel slightly less responsive

**Implementation sequence:**

1. **Phase 1: Registry Cleanup Command** (1-2h)
   - Add `orch registry clean` command to remove completed/abandoned entries
   - Filter by age (e.g., `--older-than 7d`)
   - Matches existing `orch clean` pattern for workspaces
   - Respects registry-as-cache decision (cleanup is manual hygiene, not lifecycle tracking)

2. **Phase 2: Unified Daemon Dedup** (4-6h)
   - Create `ProcessedIssueCache` that wraps all three dedup mechanisms
   - Add persistent state (SQLite or JSONL) that survives daemon restart
   - Single entry point: `cache.ShouldProcess(beadsID) bool`
   - Consolidates: SpawnedIssueTracker + SessionDedupChecker + HasPhaseComplete

3. **Phase 3: Store Update Throttling** (2-3h)
   - Add `requestAnimationFrame` or `debounce` to reactive blocks in Work Graph
   - Consider `svelte/transition` for tree rebuilds to avoid layout thrashing
   - Profile before/after to validate improvement

### Alternative Approaches Considered

**Option B: Automatic Registry Pruning**
- **Pros:** Zero maintenance burden
- **Cons:** Risk of data loss if pruning logic is wrong; harder to debug issues
- **When to use instead:** After manual cleanup proves safe for 2+ weeks

**Option C: Replace Registry with OpenCode Session State**
- **Pros:** Single source of truth, always current
- **Cons:** Major refactor; doesn't work for claude CLI mode; loses historical data
- **When to use instead:** If registry maintenance becomes untenable despite cleanup

**Rationale for recommendation:** Phased approach addresses immediate pain (636 entries) while building toward systemic fix (unified dedup). Each phase is independently valuable and can be paused if priorities shift.

---

### Implementation Details

**What to implement first:**
- Registry cleanup command (highest immediate impact, lowest risk)
- Manual invocation initially: `orch registry clean --older-than 7d`

**Things to watch out for:**
- ⚠️ Don't auto-clean entries that might be needed for debugging
- ⚠️ Unified dedup must handle daemon restart gracefully (don't duplicate spawns)
- ⚠️ Throttling must not break keyboard navigation in Work Graph

**Areas needing further investigation:**
- Actual CPU usage during heavy polling (profile with Chrome DevTools)
- Whether 6h TTL is correct (should it match max agent runtime?)
- Whether "Phase: Complete" parsing failures happen in practice

**Success criteria:**
- ✅ Registry entries < 100 after cleanup
- ✅ No daemon respawns of Phase: Complete issues
- ✅ Work Graph CPU usage < 10% during idle (no active agents)
- ✅ No flip-flopping on project switch (regression test)

---

## References

**Files Examined:**
- `pkg/registry/registry.go` - Registry design contract and state methods
- `pkg/daemon/spawn_tracker.go` - SpawnedIssueTracker with 6h TTL
- `pkg/daemon/session_dedup.go` - OpenCode session dedup checker
- `pkg/daemon/issue_adapter.go` - HasPhaseComplete and SpawnWork
- `pkg/daemon/completion_processing.go` - Completion loop and verification
- `web/src/lib/stores/work-graph.ts` - AbortController + sequence guard
- `web/src/lib/stores/attention.ts` - Attention signal store
- `web/src/routes/work-graph/+page.svelte` - Work Graph page with polling
- `.kb/decisions/2026-01-12-registry-is-spawn-cache.md` - Registry design decision

**Commands Run:**
```bash
# Count agents by status
cat ~/.orch/agent-registry.json | jq '[.agents[] | .status] | group_by(.) | map({status: .[0], count: length})'

# Registry file size
wc -l ~/.orch/agent-registry.json

# Recent commits
git log --oneline -20

# Check flip-flop fix
git show 585c69e6 --stat
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-12-registry-is-spawn-cache.md` - Registry design rationale
- **Commit:** `585c69e6` - Work Graph flip-flop fix

---

## Investigation History

**2026-02-03 11:10:** Investigation started
- Initial question: What cascading failures exist across daemon, registry, Work Graph, and attention API?
- Context: Orchestrator flagged systemic instability across multiple subsystems

**2026-02-03 11:45:** Key findings completed
- Registry phantom entries: 636 (expected per decision but degrading)
- Daemon dedup: 3 fragmented layers with gaps
- Work Graph flip-flop: Already fixed
- Attention API: Correct but polling creates load

**2026-02-03 12:00:** Investigation completed
- Status: Complete
- Key outcome: Three-phase stabilization path recommended: cleanup command, unified dedup, store throttling
