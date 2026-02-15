# Probe: Knowledge Tree SSE Cycling Fix

**Date:** 2026-02-15
**Status:** Active
**Model:** Dashboard Architecture

## Question

Does the knowledge tree page exhibit SSE cycling behavior (repeated disconnections/reconnections causing full re-renders that reset expand/collapse state and scroll position), and if so, what is the root cause?

## What I Tested

1. Examined frontend code:
   - `/web/src/routes/knowledge-tree/+page.svelte` - Page component
   - `/web/src/lib/stores/knowledge-tree.ts` - Store with SSE connection
   - `/web/src/lib/services/sse-connection.ts` - SSE connection manager

2. Examined backend code:
   - `/cmd/orch/serve_tree.go` - Tree API and SSE handlers
   - Checked if `/api/events/tree` endpoint is registered

3. Will test SSE endpoint connectivity and event format

## What I Observed

**Frontend architecture:**
- Page component loads expansion state from localStorage (line 14-24)
- Page component connects SSE on mount (line 42)
- Page component tracks expanded nodes locally (line 37)
- SSE 'tree-update' events trigger full tree replacement via `set({ tree })` (line 88)
- Expand/collapse state tracked locally but not merged with incoming SSE updates

**Backend architecture:**
- SSE endpoint `/api/events/tree` exists and is registered (serve.go:393)
- handleTreeEvents polls filesystem every 2 seconds for changes (line 245)
- When changes detected, cache invalidated and fresh tree sent (line 273)
- Tree updates sent as full tree replacements, not diffs (line 313)

**Potential issues identified:**
1. ❌ **SSE updates replace entire tree** - When SSE sends 'tree-update', store does `set({ tree })` which replaces the tree object entirely
2. ❌ **Expanded state not preserved** - Local `expandedNodes` Set is separate from tree data, but `toggleNode` mutates the tree's `expanded` property
3. ❌ **State divergence** - localStorage tracks expansion, tree objects track expansion, but SSE updates wipe tree expansion state

**Root cause hypothesis:**
The cycling behavior is likely caused by:
1. SSE connection working correctly
2. Tree updates arriving (filesystem polling every 2 seconds)
3. Each update replaces the entire tree object
4. Tree replacement wipes all `expanded` properties on nodes
5. UI re-renders with all nodes collapsed
6. This creates visual "cycling" effect

## Model Impact

**Target section:** "Why This Fails" - Should add new failure mode

**Result:** TBD (need to verify with actual reproduction and fix)

**Expected model extension:**

### Failure Mode N: Knowledge Tree SSE Cycling

**Symptom:** /knowledge-tree page shows "disconnected" SSE indicator and tree visually cycles through different expand/collapse states, resetting scroll position on each update

**Root cause:** SSE events send full tree replacements that wipe client-side expand/collapse state stored in tree nodes

**Why it happens:**
- SSE 'tree-update' events send complete tree structure
- Store handler does `set({ tree })` replacing entire tree object
- Tree nodes have `expanded?: boolean` property set by client
- Full replacement loses all client-side expanded state
- UI re-renders with nodes in default (collapsed) state
- Visual cycling as tree resets on each SSE update

**Fix:** Preserve expand/collapse state across SSE updates by:
1. Track expanded node IDs separately (already in localStorage)
2. When SSE update arrives, merge expansion state back into new tree
3. OR: Use diff-based updates that preserve existing node properties
4. OR: Make expanded state purely local, not stored in tree nodes

---

## Next Steps

1. Verify SSE endpoint is actually connecting (test in browser DevTools)
2. Confirm tree updates are arriving via SSE
3. Implement fix to preserve expansion state across updates
4. Verify scroll position preservation
5. Update this probe with final results
