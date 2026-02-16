# Probe: Work-Graph Issues View Section Design (Triple-Rendering)

**Model:** dashboard-architecture
**Status:** Complete
**Date:** 2026-02-16

## Question

Does the current work-graph Issues view faithfully render each issue in exactly one location, or do items appear in multiple sections simultaneously? The dashboard-architecture model expects the dashboard to be the "single source of truth" with entities deduplicated by entity, not by event.

## What I Tested

1. Read `web/src/routes/work-graph/+page.svelte` — the main page component
2. Read `web/src/lib/components/work-graph-tree/work-graph-tree.svelte` — the tree component
3. Read `web/src/lib/stores/work-graph.ts` — the data store (buildTree, nodes, edges)
4. Read `web/src/lib/stores/attention.ts` — attention signals / completed issues
5. Read `web/src/lib/stores/wip.ts` — WIP store (stub)
6. Analyzed the reverted commit `4023c439` ("feat: add In Progress section") via `git show`
7. Analyzed the revert commit `8933cf13`

## What I Observed

### Current Rendering Paths (Post-Revert)

Two rendering paths exist for an issue with `status: in_progress` and `agent.phase: complete`:

1. **Ready to Complete section** (`+page.svelte:338-376`): Reactive block iterates `$agents`, finds agents where `phase === 'complete'` AND matched beads node has `status === 'in_progress'`. Renders as a green-bordered card above the tree.

2. **Main tree** (WorkGraphTree): `buildTree()` includes ALL nodes regardless of status. The tree renders every node from `$workGraph.nodes`. No filtering by status occurs.

**Result:** An `in_progress` issue with a completed agent appears in BOTH locations. This is a double-rendering bug.

### WIP Deduplication Pattern (Already Working)

The WorkGraphTree has a well-designed deduplication pattern:
- `pinnedTreeIds` Set tracks which beads IDs appear in the WIP section
- `flattenVisibleTree()` filters pinned IDs out of the tree
- This prevents WIP items from appearing in both WIP section and tree

**But:** The Ready to Complete section is rendered in `+page.svelte` OUTSIDE the WorkGraphTree component. It doesn't participate in the `pinnedTreeIds` mechanism at all.

### The Reverted "In Progress" Commit

The commit at `4023c439` added a third section but explicitly chose NOT to filter items from the tree: "Main tree retains all in_progress nodes (not filtered out)." This created the triple-rendering problem:
- Ready to Complete (phase:complete + status:in_progress)
- In Progress (status:in_progress)
- Tree (all nodes)

An issue matching both Ready to Complete AND In Progress criteria appeared in all three.

## Model Impact

**Extends** the dashboard-architecture model:

1. **Confirms** the principle that observation should be deduplicated by entity. The WIP section's `pinnedTreeIds` pattern demonstrates the correct approach. The Ready to Complete section's lack of participation in this pattern is a known gap.

2. **Extends** with a new architectural constraint: **Any promoted section (a section that pulls items out of the tree for special display) must participate in the deduplication mechanism.** Currently only WIP does this. Ready to Complete does not.

3. **Extends** with a design recommendation: The Issues view needs explicit mutual exclusivity rules between its sections. See investigation: `.kb/investigations/2026-02-16-design-work-graph-issues-view-sections.md`
