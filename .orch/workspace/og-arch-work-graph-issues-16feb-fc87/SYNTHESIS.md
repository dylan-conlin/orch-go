# Session Synthesis

**Agent:** og-arch-work-graph-issues-16feb-fc87
**Issue:** orch-go-981
**Duration:** 2026-02-16T18:30 → 2026-02-16T19:00
**Outcome:** success

---

## Plain-Language Summary

The work-graph Issues view had a triple-rendering bug: issues could appear in Ready to Complete, In Progress, AND the main tree simultaneously. After analyzing the existing deduplication pattern (WorkGraphTree's `pinnedTreeIds` which already correctly removes WIP items from the tree), I designed a three-layer priority cascade where each issue appears in exactly one section. Ready to Complete claims first (needs Dylan's attention), WIP claims second (active agents), and the tree gets everything else. The fix extends the existing pattern by passing an `excludeIds` prop from the page to the tree component — no new patterns needed, just applying what already works. A separate "In Progress" section is NOT recommended since the WIP section already serves that purpose.

## Verification Contract

See `VERIFICATION_SPEC.yaml` for acceptance criteria.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-02-16-design-work-graph-issues-view-sections.md` — Design decision with section definitions, mutual exclusivity rules, and implementation approach
- `.kb/models/dashboard-architecture/probes/2026-02-16-work-graph-issues-view-section-design.md` — Probe confirming existing double-rendering bug and extending model with deduplication constraint

### Commits
- (pending) — architect: work-graph Issues view section hierarchy and mutual exclusivity design

---

## Evidence (What Was Observed)

- Ready to Complete section (`+page.svelte:338-376`) renders issues matching `agent.phase === 'complete'` AND `node.status === 'in_progress'` but does NOT filter them from the tree
- WorkGraphTree's `pinnedTreeIds` pattern (`work-graph-tree.svelte:209-256`) correctly deduplicates WIP items from tree via `flattenVisibleTree()`
- Ready to Complete section is in `+page.svelte` (outside WorkGraphTree) and doesn't participate in `pinnedTreeIds`
- Reverted commit `4023c439` explicitly chose NOT to filter In Progress items from tree, creating triple-rendering
- WIP store (`wip.ts`) is a stub — returns empty arrays, but the plumbing in WorkGraphTree is ready

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-02-16-design-work-graph-issues-view-sections.md` — Definitive section design for Issues view

### Decisions Made
- Three-layer priority cascade (Ready to Complete > WIP > Tree) with `excludeIds` prop
- No dedicated "In Progress" section — WIP section already serves this purpose
- Implementation approach: pass `readyToCompleteIds` as `excludeIds` to WorkGraphTree

### Constraints Discovered
- Any promoted section (that pulls items out of the tree for special display) MUST participate in the deduplication mechanism
- Ready to Complete takes priority over WIP when an issue has both a completed agent and a running agent

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation + probe)
- [x] Design decision documented with implementation approach
- [x] Ready for `orch complete orch-go-981`

### Follow-up Implementation
When ready to implement:
- Spawn `feature-impl` with context: "Implement three-layer priority cascade for work-graph Issues view per `.kb/investigations/2026-02-16-design-work-graph-issues-view-sections.md`"
- Estimated effort: Small (3 files, ~20 lines changed)

---

## Unexplored Questions

- Should the header issue count reflect total open issues or only tree-visible issues? (The investigation recommends total — the count is about the work landscape, not the view)
- When WIP store is fully implemented (currently stub), will the WIP section's agent-to-issue mapping create any new overlap scenarios?

---

## Session Metadata

**Skill:** architect
**Model:** opus
**Workspace:** `.orch/workspace/og-arch-work-graph-issues-16feb-fc87/`
**Investigation:** `.kb/investigations/2026-02-16-design-work-graph-issues-view-sections.md`
**Beads:** `bd show orch-go-981`
