<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Agent grid jostling was caused by `is_processing` being used as primary sort key in `recent-activity` mode, and gold border flashing was caused by immediate state transitions between `busy` and `idle` SSE events.

**Evidence:** Traced SSE event flow → `session.status` events toggle `is_processing` → `sortAgents()` sorts by `is_processing` first → multiple agents toggle rapidly → grid positions swap constantly.

**Knowledge:** Stable sort (using `spawned_at`) only worked as tiebreaker after `is_processing` comparison; also rapid UI state changes need debouncing for visual stability.

**Next:** Close - fix implemented in `+page.svelte` (skip `is_processing` sort when `useStableSort=true`) and `agents.ts` (1s debounce on clearing `is_processing`).

**Confidence:** High (90%) - root cause identified through code analysis and fix directly addresses it.

---

# Investigation: Dashboard Agent Cards Rapidly Jostling

**Question:** Why are dashboard agent cards rapidly jostling for first position despite spawned_at stable sort implementation, and why is the gold border flashing on/off rapidly?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** orch-go
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: is_processing is primary sort key, not spawned_at

**Evidence:** In `sortAgents()` function, the `recent-activity` sort mode checks `is_processing` BEFORE checking `spawned_at`:

```javascript
case 'recent-activity':
    if (a.is_processing !== b.is_processing) {
        return a.is_processing ? -1 : 1;  // This comes FIRST
    }
    // spawned_at is only used as tiebreaker
    if (useStableSort) {
        return bSpawned - aSpawned;
    }
```

**Source:** `web/src/routes/+page.svelte:198-211`

**Significance:** This explains why setting `useStableSort=true` didn't prevent jostling - the `is_processing` comparison happens before the stable sort logic kicks in. When multiple agents toggle between processing and idle, they swap positions.

---

### Finding 2: SSE events trigger rapid is_processing toggles

**Evidence:** Every `session.status` SSE event immediately updates `is_processing` based on `busy`/`idle` state:

```typescript
if (data.type === 'session.status' && data.properties) {
    const isProcessing = statusType === 'busy';
    // Immediate update to store
    agents.update((agentList) => {
        return agentList.map((agent) => {
            if (agent.session_id === sessionID) {
                return { ...agent, is_processing: isProcessing };
            }
            return agent;
        });
    });
}
```

**Source:** `web/src/lib/stores/agents.ts:456-485`

**Significance:** Active agents cycle between `busy` and `idle` rapidly during normal operation (each tool use, each text generation). This causes the gold border to flash and triggers sort recalculations.

---

### Finding 3: Store updates trigger reactive re-sorts every time

**Evidence:** The reactive statement recalculates sorted agents on every store update:

```svelte
$: sortedActiveAgents = sortAgents(applyFilters($activeAgents), true);
```

**Source:** `web/src/routes/+page.svelte:285`

**Significance:** Even if sort order doesn't change, the array reference changes, potentially causing Svelte to re-render. Combined with Finding 1, this creates constant visual churn.

---

## Synthesis

**Key Insights:**

1. **Sort order priority was inverted** - The stable sort flag only controlled the secondary sort (spawned_at vs updated_at), not whether `is_processing` should be used for primary sorting. The intention was for Active section to maintain stable positions, but `is_processing` sorting defeated this.

2. **No debouncing on state clear** - Setting `is_processing=true` should be immediate (responsive), but clearing it can be debounced since brief idle periods shouldn't cause visual flapping.

3. **Visual stability requires ignoring transient state** - For grid stability, we need to distinguish between "permanent" sort criteria (spawned_at, project, phase) and "transient" criteria (is_processing, updated_at).

**Answer to Investigation Question:**

The jostling occurred because `is_processing` was the primary sort key in `recent-activity` mode, overriding the `useStableSort` flag. When multiple active agents toggle between `busy` and `idle` states via SSE events, they constantly swap grid positions. The gold border flashed because `is_processing` was set/cleared immediately without debouncing.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Code analysis clearly shows the sort order priority issue, and the SSE event flow is well-documented. The fix directly addresses both root causes.

**What's certain:**

- ✅ `is_processing` comparison precedes stable sort logic in `recent-activity` mode
- ✅ SSE `session.status` events trigger immediate `is_processing` updates
- ✅ Multiple agents processing simultaneously causes rapid state changes

**What's uncertain:**

- ⚠️ Exact frequency of SSE events during normal operation (not measured)
- ⚠️ Whether 1000ms debounce is optimal (may need tuning)

**What would increase confidence to Very High:**

- Visual verification in production with multiple active agents
- Performance profiling to confirm reduced re-renders

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Skip is_processing sort + debounced clear** - Modify sort logic to skip `is_processing` comparison when `useStableSort=true`, and add 1000ms debounce before clearing `is_processing` state.

**Why this approach:**
- Preserves visual feedback (gold border still shows processing state per-card)
- Prevents grid reordering chaos
- Simple changes with minimal risk

**Trade-offs accepted:**
- Processing agents no longer float to top of Active section
- 1s delay before gold border clears (acceptable for visual stability)

**Implementation sequence:**
1. Modify `sortAgents()` to skip `is_processing` when `useStableSort=true`
2. Add debounce timer for clearing `is_processing` in SSE handler
3. Clean up timers on disconnect to prevent memory leaks

### Alternative Approaches Considered

**Option B: Remove is_processing from sort entirely**
- **Pros:** Simpler, consistent behavior across all sections
- **Cons:** May want processing agents at top in Recent/Archive sections
- **When to use instead:** If behavior should be uniform across all sections

**Option C: Memoize sort results**
- **Pros:** Prevents re-renders when order doesn't change
- **Cons:** More complex, may have stale cache issues
- **When to use instead:** If performance is still an issue after this fix

---

## References

**Files Examined:**
- `web/src/routes/+page.svelte` - Sort logic and reactive statements
- `web/src/lib/stores/agents.ts` - SSE event handling and store updates
- `web/src/lib/components/agent-card/agent-card.svelte` - Gold border conditional logic
- `cmd/orch/serve.go` - API response structure to verify spawned_at is populated

**Commands Run:**
```bash
# Verified API returns spawned_at
curl -s http://127.0.0.1:3348/api/agents | jq '.[0:3] | .[] | {id, spawned_at, updated_at, is_processing}'

# Ran TypeScript check
cd web && bun check

# Ran Playwright tests
cd web && bunx playwright test
```

---

## Investigation History

**2025-12-26 09:30:** Investigation started
- Initial question: Why agent cards jostle and gold border flashes rapidly
- Context: User reported visual instability despite stable sort implementation

**2025-12-26 09:45:** Root cause identified
- `is_processing` is primary sort key in `recent-activity` mode
- SSE events trigger immediate state toggles without debouncing

**2025-12-26 10:00:** Fix implemented
- Modified `sortAgents()` to skip `is_processing` when `useStableSort=true`
- Added 1000ms debounce for clearing `is_processing`
- All tests pass

**2025-12-26 10:15:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Two-part fix prevents grid jostling and gold border flashing
