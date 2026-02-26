## Summary (D.E.K.N.)

**Delta:** Active agents in the dashboard constantly reorder because sorting uses `updated_at` which changes on every SSE event.

**Evidence:** Code analysis shows `sortAgents()` in +page.svelte sorts by `updated_at` for "recent-activity"; SSE events update agents ~every second, causing grid shuffling.

**Knowledge:** Active agents need stable positions; use `spawned_at` (immutable) for secondary sort instead of `updated_at` (volatile).

**Next:** Implement fix by modifying sortAgents to use spawned_at for active section stability.

**Confidence:** High (90%) - Root cause clear from code analysis; fix is straightforward.

---

# Investigation: Fix Active Agents Constantly Reordering

**Question:** Why do active agents constantly reorder in the dashboard grid, and how can we stabilize their positions?

**Started:** 2025-12-24
**Updated:** 2025-12-24
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Sort uses volatile `updated_at` field

**Evidence:** The `sortAgents()` function (web/src/routes/+page.svelte:264-313) sorts by `updated_at` for the "recent-activity" sort mode:
```typescript
case 'recent-activity':
  if (a.is_processing !== b.is_processing) {
    return a.is_processing ? -1 : 1;
  }
  const bUpdated = b.updated_at ? new Date(b.updated_at).getTime() : 0;
  const aUpdated = a.updated_at ? new Date(a.updated_at).getTime() : 0;
  return bUpdated - aUpdated;
```

**Source:** `web/src/routes/+page.svelte:267-273`

**Significance:** `updated_at` changes on every SSE event, causing constant re-sorting of the grid.

---

### Finding 2: SSE events update agents continuously

**Evidence:** The `handleSSEEvent()` function (web/src/lib/stores/agents.ts:287-358) processes multiple event types that update agent state:
- `message.part` events update `current_activity` 
- `session.status` events update `is_processing`
- Various events trigger `agents.fetch()` which updates all agents

**Source:** `web/src/lib/stores/agents.ts:294-358`

**Significance:** Active agents receive events every few seconds, causing their `updated_at` to change constantly, triggering reactive re-sorts.

---

### Finding 3: Active section has different stability needs than archive

**Evidence:** 
- Active agents: receiving constant updates, positions should be stable
- Recent/Archive agents: not receiving updates, `updated_at` sort is appropriate to show most recently finished first

**Source:** UI behavior observation + code analysis

**Significance:** The fix should only change sort behavior for the active section, not for recent/archive where `updated_at` sorting makes sense.

---

## Synthesis

**Key Insights:**

1. **Stable vs Volatile Sort** - Active agents need stable grid positions to prevent visual churn. Using `spawned_at` (immutable) instead of `updated_at` (volatile) provides this stability while maintaining processing-first ordering.

2. **Section-Specific Behavior** - The fix should only apply to active agents. Recent/archive sections benefit from `updated_at` sorting since they show "most recently finished" ordering.

3. **Dead Code Removal** - The `filteredAgents` computed property was dead code from an older flat-grid implementation. Removing it eliminates duplicate sorting logic and simplifies the codebase.

**Answer to Investigation Question:**

Active agents constantly reorder because the sort function uses `updated_at` timestamps which change on every SSE event (~every second for active agents). The fix introduces a `useStableSort` parameter to `sortAgents()` that uses `spawned_at` (immutable) instead of `updated_at` (volatile) for the active section, while preserving `updated_at` sorting for recent/archive sections where recency ordering is appropriate.

---

## Implementation (Completed)

**Changes Made:**

1. **Modified `sortAgents()` function** (`web/src/routes/+page.svelte:264-343`)
   - Added `useStableSort: boolean = false` parameter
   - When `useStableSort=true`, uses `spawned_at` instead of `updated_at` for secondary sort
   - Applied to 'recent-activity', 'project', and 'phase' sort modes

2. **Updated call sites** (`web/src/routes/+page.svelte:345-349`)
   - `sortedActiveAgents` now uses stable sort (`true`)
   - `sortedRecentAgents` and `sortedArchivedAgents` use volatile sort (`false`)

3. **Removed dead code** 
   - Deleted unused `filteredAgents` computed property (76 lines of duplicate sorting logic)

**Tests:**
- `npm run check` - Type check passes
- `npm run build` - Build succeeds
- Playwright filtering tests - All 8 pass
- [Why that's acceptable given findings]

**Implementation sequence:**
1. [First step - why it's foundational]
2. [Second step - why it comes next]
3. [Third step - builds on previous]

### Alternative Approaches Considered

---

## References

**Files Modified:**
- `web/src/routes/+page.svelte` - Main dashboard component with sorting logic
- `web/src/lib/stores/agents.ts` - Agent store with SSE event handling

**Files Examined:**
- `web/src/lib/components/agent-card/agent-card.svelte` - Agent card component (unchanged)

**Commands Run:**
```bash
npm run check   # Type check - passed
npm run build   # Build - passed
npx playwright test filtering.spec.ts  # 8/8 tests passed
```
