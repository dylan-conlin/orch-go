# Design: Work Graph Store Throttling

**Date:** 2026-02-03  
**Status:** Draft  
**Owner:** orch-go-21203

## Problem

Work Graph page exhibits high CPU usage (>10%) during idle state when polling is active. The root cause is reactive cascades: multiple stores (`workGraph`, `wip`, `agents`, `attention`, `orchestratorContext`, `daemon`) poll at 2-5 second intervals, and each update triggers expensive tree rebuilds even when data hasn't meaningfully changed.

**Current behavior:**
- `orchestratorContext` polls every 2 seconds
- `workGraph`, `wip`, `daemon` poll every 5 seconds  
- Reactive block (line 143-227) rebuilds tree on EVERY store update
- Reactive block (line 232-271) fires on EVERY `$orchestratorContext` change
- Tree rebuild involves: filtering nodes, building hierarchy, applying expansion state, attaching badges, tracking new issues, persisting to localStorage

**Target:** CPU <10% during idle state (no active agents).

## Investigation Findings

From `.kb/investigations/2026-02-03-inv-systemic-stability-audit-take-stock.md`:

> Multiple stores (workGraph, wip, agents, attention, daemon) updating at different intervals can trigger cascading reactive updates. The Work Graph page subscribes to most of them, creating potential for render storms.

The investigation recommends (Phase 3):
- Add `requestAnimationFrame` or `debounce` to reactive blocks
- Consider `svelte/transition` for tree rebuilds to avoid layout thrashing
- Profile before/after to validate improvement

## Solution

Implement three-layer throttling strategy to reduce reactive cascades:

### Layer 1: Store-Level Shallow Equality Checks

**Problem:** Stores update on every fetch, even when API returns identical data (object reference changes but content doesn't).

**Solution:** Add shallow equality check before calling `set()` in store fetch methods.

**Implementation:**
```typescript
// In work-graph.ts fetch() method
const data = await response.json();

// Only update if data actually changed
if (!shallowEqual(currentData, data)) {
  set(data);
}
```

**Trade-off:** Adds shallow comparison overhead (~1ms for typical graph) but prevents downstream reactive cascades (~50-100ms of tree rebuild + DOM updates).

### Layer 2: Reactive Block Debouncing

**Problem:** When multiple stores update near-simultaneously, reactive blocks fire multiple times in rapid succession.

**Solution:** Debounce tree rebuild reactive block to batch updates within 100ms window.

**Implementation:**
```svelte
let rebuildDebounceTimeout: ReturnType<typeof setTimeout> | null = null;

$: {
  if ($workGraph && !$workGraph.error && $wip) {
    // Cancel pending rebuild
    if (rebuildDebounceTimeout) {
      clearTimeout(rebuildDebounceTimeout);
    }
    
    // Debounce rebuild to batch rapid updates
    rebuildDebounceTimeout = setTimeout(() => {
      rebuildDebounceTimeout = null;
      // ... existing tree rebuild logic ...
    }, 100);
  }
}
```

**Trade-off:** 100ms delay feels imperceptible to users but batches rapid store updates (e.g., when agents store updates → wip store updates → tree rebuilds once, not twice).

### Layer 3: Project Change Optimization

**Problem:** Reactive block fires on EVERY `$orchestratorContext` change, not just `project_dir` changes.

**Solution:** Extract `project_dir` into derived store to isolate reactivity.

**Current pattern (problematic):**
```svelte
$: {
  if ($orchestratorContext.project_dir !== currentProjectDir) {
    // Fires on EVERY orchestratorContext change
  }
}
```

**Optimized pattern:**
```svelte
// Derived store that extracts only project_dir
const projectDir = derived(orchestratorContext, $ctx => $ctx.project_dir);

$: {
  if ($projectDir !== currentProjectDir) {
    // Fires ONLY when project_dir changes
  }
}
```

**Trade-off:** Requires one more store subscription but eliminates false-positive reactivity when other context fields change (e.g., `cwd`, `included_projects`).

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ Polling Loop (2-5s intervals)                               │
│  - orchestratorContext (2s)                                 │
│  - workGraph, wip, daemon (5s)                              │
└──────────────────┬──────────────────────────────────────────┘
                   │ Fetch API data
                   ▼
┌─────────────────────────────────────────────────────────────┐
│ Layer 1: Store-Level Throttling                             │
│  - Shallow equality check before set()                      │
│  - Only update if data meaningfully changed                 │
└──────────────────┬──────────────────────────────────────────┘
                   │ Store update (fewer updates)
                   ▼
┌─────────────────────────────────────────────────────────────┐
│ Layer 2: Reactive Block Debouncing                          │
│  - Tree rebuild debounced (100ms window)                    │
│  - Batches rapid updates                                    │
└──────────────────┬──────────────────────────────────────────┘
                   │ Tree rebuild (batched)
                   ▼
┌─────────────────────────────────────────────────────────────┐
│ Layer 3: Project Change Isolation                           │
│  - Derived store for project_dir                            │
│  - Eliminates false-positive reactivity                     │
└─────────────────────────────────────────────────────────────┘
```

## Testing Strategy

### Before/After Profiling

1. **Baseline measurement (before changes):**
   - Open Work Graph page with 50+ open issues
   - No active agents (idle state)
   - Record CPU usage over 60 seconds using Chrome DevTools Performance tab
   - Note: Main thread activity, JS heap size, frame rate

2. **After implementation:**
   - Same scenario, measure CPU usage over 60 seconds
   - Target: CPU usage <10% during idle
   - Secondary target: Frame rate stable at 60 FPS

### Unit Tests

Add tests to `work-graph-polling.spec.ts`:

```typescript
test('should not rebuild tree when API returns identical data', async ({ page }) => {
  let treeRebuildCount = 0;
  
  // Inject rebuild counter
  await page.addInitScript(() => {
    window.__treeRebuildCount = 0;
  });
  
  // Mock API to return identical data twice
  // Verify tree rebuild happens only once
});

test('should batch rapid store updates within 100ms', async ({ page }) => {
  // Update 3 stores within 50ms
  // Verify tree rebuilds only once after 100ms debounce
});
```

### Manual Validation

- Visual verification: Work Graph still updates within ~5 seconds of issue changes
- No perceived lag in keyboard navigation (j/k/l/h keys)
- Project switching still works correctly (300ms debounce preserved)

## Implementation Plan

1. **Phase 1: Add shallow equality utility** (30 min)
   - Create `web/src/lib/utils/shallow-equal.ts`
   - Add unit tests

2. **Phase 2: Store-level throttling** (45 min)
   - Update `work-graph.ts` fetch method
   - Update `wip.ts` fetch methods
   - Update `context.ts` fetch method

3. **Phase 3: Reactive block debouncing** (30 min)
   - Add debounce to tree rebuild reactive block
   - Update cleanup in `onDestroy`

4. **Phase 4: Project change optimization** (30 min)
   - Create derived store for `project_dir`
   - Update reactive block to use derived store

5. **Phase 5: Validation** (45 min)
   - Profile before/after
   - Run unit tests
   - Manual visual verification
   - Document results in beads comment

**Total estimated time:** 3 hours

## Alternative Approaches Considered

### Alternative A: Increase Polling Intervals

**Approach:** Change polling from 2-5s to 10-30s intervals.

**Pros:** 
- Simplest solution
- Reduces API load

**Cons:**
- Less responsive UI (users wait 10-30s to see changes)
- Doesn't address root cause (reactive cascades still happen, just less often)

**Decision:** Rejected. UX degradation not acceptable for marginal CPU improvement.

### Alternative B: Virtual Scrolling

**Approach:** Only render visible tree nodes using virtualization library.

**Pros:**
- Reduces DOM size
- Handles very large trees (1000+ nodes)

**Cons:**
- Major refactor (~8-12 hours)
- Adds library dependency
- Current trees rarely exceed 100 nodes (virtualization not needed yet)

**Decision:** Deferred. Re-evaluate if tree size regularly exceeds 200 nodes.

### Alternative C: Move Tree Building to Web Worker

**Approach:** Offload `buildTree()` computation to background thread.

**Pros:**
- Main thread stays responsive
- Handles complex computations well

**Cons:**
- Can't access DOM or stores from worker
- Requires message passing overhead
- Tree rebuild is fast (<10ms for typical graph)

**Decision:** Rejected. Overhead exceeds benefit for current tree sizes.

## Success Criteria

- [ ] CPU usage <10% during idle state (60s measurement)
- [ ] No perceived lag in keyboard navigation
- [ ] Tree updates within 5s of issue changes (visual verification)
- [ ] Project switching works correctly (no flip-flop regression)
- [ ] All existing tests pass
- [ ] Frame rate stable at 60 FPS during polling

## Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Debounce delays feel laggy | Low | Medium | Use 100ms (imperceptible). If needed, reduce to 50ms. |
| Shallow equality misses nested changes | Low | High | Current stores don't have deeply nested data. Document assumption. |
| Derived store subscription overhead | Low | Low | Single derived store has negligible cost. Profile to confirm. |
| Breaks keyboard navigation | Low | High | Comprehensive manual testing before completion. |

## Open Questions

None. Design ready for implementation.

## References

- **Investigation:** `.kb/investigations/2026-02-03-inv-systemic-stability-audit-take-stock.md`
- **Epic:** `orch-go-21202` (Final Stabilization Sprint)
- **Current implementation:** `web/src/routes/work-graph/+page.svelte:143-271`
