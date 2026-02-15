# Session Synthesis

**Agent:** og-debug-fix-dashboard-knowledge-15feb-1381
**Issue:** orch-go-p7b9
**Duration:** 2026-02-15T11:00 → 2026-02-15T11:15
**Outcome:** success

---

## TLDR

Fixed the knowledge tree dashboard SSE cycling bug. Tree was re-rendering every 2 seconds (matching SSE poll interval), resetting expand/collapse and scroll state. Root cause: expansion state was stored on tree data objects that got replaced on every SSE update. Fix: decoupled expansion state into component-owned Set, added fingerprint deduplication (order-insensitive), and stable client-side sorting.

---

## Delta (What Changed)

### Files Modified
- `web/src/lib/stores/knowledge-tree.ts` - Removed expansion state from tree data (`expanded` property removed from interface), removed `applyExpansionState`/`toggleNode` methods, added order-insensitive `treeFingerprint` to skip duplicate SSE updates, removed `onDisconnect` error handler that was hiding tree on SSE drops, added `getSSEStatus()` for connection indicator
- `web/src/lib/components/knowledge-tree/knowledge-tree.svelte` - Component now accepts `expandedNodes: Set<string>` prop instead of reading `node.expanded`, expansion determined by `expandedNodes.has(node.ID)` which is immune to tree data replacement
- `web/src/routes/knowledge-tree/+page.svelte` - Expansion state fully owned by page component (localStorage-backed Set), SSE connection status indicator (green/yellow/red dot), stable alphabetical sorting of tree children, error only shows when tree is null (not on SSE disconnect)

---

## Evidence (What Was Observed)

- SSE endpoint `/api/events/tree` polls filesystem every 2s and sends full tree replacement whenever `.kb/` or `.beads/` directories change (serve_tree.go:245)
- In active development, these directories change constantly (agents writing investigations, beads being updated), causing tree updates every 2 seconds
- Original `onDisconnect` handler set `error` in store state, causing page to alternate between tree view and error view on each SSE drop/reconnect
- Original `node.expanded` property was set on tree data objects that were completely replaced on each SSE update
- Backend sends clusters in different order between polls (sorted by filesystem modification time), causing DOM reordering and scroll position shifts
- After fix: Network tab shows SSE connection at 46 B (just `connected` event), no tree-update events reaching store when content unchanged
- After fix: 28 seconds between Knowledge and Work fetches with no intermediate fetches or reconnections

### Tests Run
```bash
# TypeScript check - no errors in modified files
npx svelte-check --threshold error 2>&1 | grep "knowledge-tree"
# No errors in knowledge-tree files

# Visual verification - 3 screenshots over 15+ seconds showing stable tree
# Tree renders once, expansion state persists, no cycling
```

---

## Knowledge (What Was Learned)

### Decisions Made
- Expansion state decoupled from tree data: Component-owned Set<string> prop instead of `node.expanded` property on tree data. This makes UI state immune to data layer replacements.
- Order-insensitive fingerprinting: Children sorted before fingerprinting so backend reordering doesn't trigger false store updates
- Stable client-side sort: Children sorted alphabetically by ID regardless of backend order, preventing DOM reordering and scroll shifts
- SSE disconnect doesn't hide tree: Removed `onDisconnect` error handler. Error state only shown when tree is null (initial fetch failure). SSE status shown via small indicator dot.

### Constraints Discovered
- Svelte keyed `{#each}` reuses components for same keys but still triggers prop updates and reactive recalculations on data replacement
- `getLastModTime` in serve_tree.go walks entire `.kb/` and `.beads/` directory trees every 2 seconds - sensitive to any file modification, not just structural changes

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Visual verification passing (3 screenshots showing stable tree)
- [x] TypeScript clean (no errors in modified files)
- [x] Ready for `orch complete orch-go-p7b9`

---

## Unexplored Questions

- The backend `getLastModTime` walk of entire `.kb/` directory is O(n) every 2 seconds. For large knowledge bases, this could become expensive. Consider using filesystem watchers (fsnotify) instead of polling.
- The 521 KB initial tree payload is large. Consider lazy-loading children on expand or pagination for large trees.

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** opus
**Workspace:** `.orch/workspace/og-debug-fix-dashboard-knowledge-15feb-1381/`
**Beads:** `bd show orch-go-p7b9`
