# Design: Work-Graph Issues View Section Hierarchy & Mutual Exclusivity

**Date:** 2026-02-16
**Phase:** Complete
**Status:** Complete
**Trigger:** Triple-rendering bug — issues appearing in Ready to Complete, In Progress, AND the main tree simultaneously
**Scope:** Work-graph Issues view only (not Artifacts tab, Completed tab, or other dashboard views)

---

## Design Question

What are the mutual exclusivity rules between sections in the work-graph Issues view? When an issue qualifies for a promoted section (Ready to Complete, In Progress), should it be removed from the main tree?

## Problem Framing

### Success Criteria

1. Every open issue appears in exactly ONE location within the Issues view
2. Dylan can instantly see what needs his attention (completions to review)
3. Dylan can see what's actively being worked on without scanning the tree
4. The tree shows the remaining work landscape (open, blocked, queued)
5. The existing `pinnedTreeIds` deduplication pattern is extended, not replaced

### Constraints

- **Principle: Progressive Disclosure** — Front-load the signal. Action-required items first.
- **Principle: Coherence Over Patches** — The fix must be a clean design, not another patch
- **Principle: Observation faithfulness** — Deduplicated by entity, not by event
- **Existing pattern:** WIP section in WorkGraphTree already removes pinned items from tree via `flattenVisibleTree()`
- **Architecture:** Ready to Complete section is currently in `+page.svelte` (outside WorkGraphTree)

### Scope

- **In scope:** Section definitions, mutual exclusivity rules, information hierarchy, implementation approach
- **Out of scope:** Artifacts tab, Completed tab, dashboard changes outside work-graph

---

## Exploration

### Fork 1: Where should promoted sections live?

**Options:**
- **A: All sections inside WorkGraphTree** — Move Ready to Complete into the component, alongside WIP. Both participate in `pinnedTreeIds`.
- **B: Keep Ready to Complete in +page.svelte, pass IDs down** — WorkGraphTree receives `excludeIds` prop, filters them from tree.
- **C: Status quo** — No deduplication (current broken state).

**Substrate says:**
- Principle: Coherence Over Patches — Option A is cleaner; the deduplication logic lives in one place
- Existing pattern: WIP already works inside WorkGraphTree with pinning
- Architecture: WorkGraphTree already accepts `wipItems` prop and renders them above the tree

**Recommendation:** Option B (pass IDs down). Reasoning: Ready to Complete is a page-level concern with different data sources (agents store + work-graph nodes). Moving it into WorkGraphTree would couple the tree component to the agents store. Keeping it in `+page.svelte` with a clean `excludeIds` prop maintains separation of concerns while achieving deduplication.

**Trade-off accepted:** Two deduplication mechanisms (pinnedTreeIds for WIP, excludeIds for Ready to Complete) instead of one. This is acceptable because they serve different architectural layers.

### Fork 2: What are the section definitions?

**The three-section model:**

| Section | Purpose | Criteria | Visual Treatment |
|---------|---------|----------|-----------------|
| **Ready to Complete** | Needs Dylan's attention NOW | `agent.phase === 'complete'` AND `node.status === 'in_progress'` AND `node.source === 'beads'` | Green border, top position |
| **WIP (In Progress)** | Actively being worked on | Running agent linked to beads issue (via `wipItems` from agents store) | Blue accents, below Ready to Complete |
| **Tree** | Everything else | All remaining open issues not in above sections | Standard tree rendering |

### Fork 3: What are the mutual exclusivity rules?

**Priority cascade (highest priority wins):**

1. **Ready to Complete** takes first claim. If an issue matches Ready to Complete criteria, it appears there and nowhere else.
2. **WIP** takes second claim. If an issue has a running agent (not yet phase:complete), it appears in WIP and nowhere else.
3. **Tree** gets everything remaining.

**Edge cases:**
- An issue with multiple agents where one is complete and another is still running: **Ready to Complete wins** (the completion is the actionable signal)
- An issue with `status: in_progress` but no running agent and no completed agent: **Tree** (it's in progress but no agent is active — possibly manual work or agent died)
- An issue with `status: open` and a running agent: **WIP** (agent is working on it even though beads status hasn't transitioned yet)

**Substrate says:**
- Principle: Progressive Disclosure — Action-required items (Ready to Complete) must come first
- Principle: Observation faithfulness — Each entity in exactly one place
- Existing pattern: WIP's `pinnedTreeIds` already implements this cascade for WIP items

### Fork 4: Should "In Progress" be a dedicated section or just visual treatment in the tree?

**Options:**
- **A: Dedicated section** — Explicit "In Progress" section between Ready to Complete and WIP
- **B: No dedicated section** — `in_progress` nodes stay in tree with visual styling (blue border, status icon)
- **C: Merge with WIP** — All in_progress nodes with active agents already appear in WIP section

**Substrate says:**
- The reverted commit tried Option A and created the triple-rendering problem
- WIP section already surfaces actively-worked issues with running agents
- Issues that are `in_progress` without an active agent are edge cases (agent died, manual status change)

**Recommendation:** Option C — rely on WIP for agent-active issues, keep agentless in_progress nodes in tree with visual styling. No separate "In Progress" section needed.

**Reasoning:** The WIP section IS the "In Progress" view. It shows running agents with their beads-linked issues, including phase, runtime, model, and health. Adding a separate "In Progress" section would either duplicate WIP or create a confusing distinction between "has agent" and "marked in_progress." The tree's existing blue border styling for `in_progress` nodes handles the remaining cases.

---

## Synthesis

### Recommended Design: Three-Layer Priority Cascade

```
┌─────────────────────────────────────┐
│  Ready to Complete (green)          │  ← Needs Dylan's attention NOW
│  Phase:Complete agents awaiting     │  ← Highest priority claim
│  orch complete review               │
├─────────────────────────────────────┤
│  WIP Section (WorkGraphTree)        │  ← Running agents, live status
│  Running agents linked to issues    │  ← Second priority claim
│  Queued issues awaiting spawn       │
├─────────────────────────────────────┤
│  Tree (WorkGraphTree)               │  ← Everything else
│  Open, blocked, in_progress (no     │  ← Gets remaining issues
│  agent), epics, questions, etc.     │
└─────────────────────────────────────┘
```

### Mutual Exclusivity Rules (Definitive)

1. **An issue appears in AT MOST ONE section.** No exceptions.
2. **Ready to Complete claims first.** Any issue matching Ready to Complete criteria is excluded from WIP and Tree.
3. **WIP claims second.** Any issue with a running/queued agent (not already in Ready to Complete) is excluded from Tree.
4. **Tree gets the rest.** All issues not claimed by the above sections.

### Implementation Approach

**Step 1:** Compute `readyToCompleteIds` Set from `readyToCompleteItems` in `+page.svelte`.

**Step 2:** Pass `readyToCompleteIds` as a prop to `WorkGraphTree`:
```svelte
<WorkGraphTree
  tree={filteredTree}
  excludeIds={readyToCompleteIds}
  ...
/>
```

**Step 3:** In WorkGraphTree's flattening logic, add `excludeIds` to the pinned set:
```typescript
// Merge page-level exclusions with WIP pinned IDs
const allExcluded = new Set([...pinnedIds, ...excludeIds]);
items.push(...flattenVisibleTree(tree, allExcluded));
```

**Step 4:** WIP items that are ALSO in `readyToCompleteIds` should be excluded from WIP rendering too (Ready to Complete takes priority). Filter `wipItems` before rendering:
```typescript
const visibleWipItems = wipItems.filter(item => {
  const id = item.type === 'running' ? item.agent.beads_id : item.issue.id;
  return !excludeIds.has(id);
});
```

### What Each Section Shows

| Section | Shows | Does NOT Show | Data Source |
|---------|-------|--------------|-------------|
| **Ready to Complete** | Issues with Phase:Complete agent awaiting review | Agents still running, closed issues | `$agents` + `$workGraph.nodes` join |
| **WIP** | Running agents with linked issues, queued issues | Items already in Ready to Complete | `wipItems` prop (from agents store) |
| **Tree** | All open issues not in above sections | Items in Ready to Complete or WIP | `$workGraph.nodes` via `buildTree()` |

### File Targets

| File | Change |
|------|--------|
| `web/src/routes/work-graph/+page.svelte` | Compute `readyToCompleteIds` Set, pass as `excludeIds` prop |
| `web/src/lib/components/work-graph-tree/work-graph-tree.svelte` | Accept `excludeIds` prop, merge with `pinnedIds`, filter WIP |
| `web/src/lib/components/work-graph-tree/work-graph-tree-helpers.ts` | No changes needed (already has `flattenVisibleTree`) |

### Acceptance Criteria

- [ ] Issue appearing in Ready to Complete does NOT appear in WIP or Tree
- [ ] Issue appearing in WIP does NOT appear in Tree
- [ ] Issue counts in header still reflect total (not filtered) open issues
- [ ] Keyboard navigation works across all sections
- [ ] Existing attention badges still render on items in their respective sections

---

## Recommendations

**RECOMMENDED:** Three-layer priority cascade with `excludeIds` prop

- **Why:** Extends the working `pinnedTreeIds` pattern. Minimal code change. Clean separation of concerns.
- **Trade-off:** Two deduplication paths (pinnedTreeIds + excludeIds) instead of one. Acceptable because they serve different architectural layers.
- **Expected outcome:** Each issue appears in exactly one section. Dylan's attention is directed to completions first, then active work, then the full landscape.

**Alternative: Move Ready to Complete into WorkGraphTree**
- **Pros:** Single deduplication mechanism
- **Cons:** Couples tree component to agents store, increases component complexity
- **When to choose:** If Ready to Complete grows more complex (e.g., inline completion actions)

**Alternative: No In Progress section (status quo minus double-rendering)**
- **Pros:** Simplest change (just fix deduplication)
- **Cons:** Same as recommended — we're NOT adding an In Progress section anyway

## Decision Gate Guidance (if promoting to decision)

**Add blocks: frontmatter when:**
- Future work adds more promoted sections to the Issues view
- WIP section behavior changes

**Suggested blocks keywords:**
- "work-graph sections"
- "issues view"
- "ready to complete"
- "mutual exclusivity"
- "triple rendering"
