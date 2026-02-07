<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** "Filters vs sections" is a false dichotomy - the Work Graph needs BOTH: sections for attention scanning (<5s) and tree presence for context preservation. The current deduplication approach (remove from tree if in section) breaks context; the fix is "dual presence" where items appear in both places with visual distinction.

**Evidence:** Code analysis shows current implementation dedupes items from tree when shown in WIP/completed sections (work-graph-tree.svelte:75-84). This creates conceptual confusion: "where does this issue live?" Prior investigations established three orthogonal dimensions (Work Status, Verification Status, Issue Status) that should be layered, not competing.

**Knowledge:** The three orthogonal dimensions map to visual layers: Issue Status = tree structure (primary), Work Status = section membership + badges (overlay), Verification Status = badges + section priority (overlay). Sections are "attention lenses" that surface subsets of the tree, not alternatives to it. Items should exist in both their section AND their tree position.

**Next:** Implement dual-presence model: remove deduplication logic, add visual dimming to tree items that also appear in sections above, add keyboard shortcut to jump between section and tree position.

**Authority:** architectural - Cross-component affecting UI presentation model, data flow, and user interaction patterns with multiple valid approaches

---

# Design: Work Graph Filters vs Sections

**Question:** Should WIP items appear WITHIN the tree (at their natural position with overlay) or as a SEPARATE section (current)? Should 'Needs Action' be a filter that highlights/collapses the tree, or a separate section? How do we preserve tree context while making 'what needs my attention' answerable in <5s?

**Started:** 2026-02-04
**Updated:** 2026-02-04
**Owner:** Claude (architect)
**Phase:** Complete
**Next Step:** None - recommendation ready for implementation
**Status:** Complete

<!-- Lineage -->
**Patches-Decision:** N/A
**Related-Investigation:** `.kb/investigations/2026-02-04-arch-work-graph-done-states.md` - UI badges and lifecycle states
**Related-Investigation:** `.kb/investigations/2026-02-04-inv-agents-own-declaration-via-bd.md` - Three orthogonal dimensions model
**Related-Investigation:** `.kb/investigations/2026-02-02-design-work-graph-unified-attention-model.md` - Unified attention surface

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `2026-02-04-inv-agents-own-declaration-via-bd.md` | extends | Yes | None - three dimensions model confirmed |
| `2026-02-04-arch-work-graph-done-states.md` | extends | Yes | None - badge renaming is complementary |
| `2026-02-02-inv-pressure-test-work-graph-unified.md` | extends | Yes | None - composable signal architecture confirmed |

---

## Findings

### Finding 1: Current Implementation Uses Deduplication, Breaking Tree Context

**Evidence:** The work-graph-tree.svelte component explicitly removes items from the tree that appear in WIP or completed sections:

```javascript
// work-graph-tree.svelte:75-84
const shownIds = new Set<string>();
for (const item of wipItems) {
    shownIds.add(item.type === 'running' ? (item.agent.beads_id || item.agent.id) : item.issue.id);
}
for (const issue of pendingVerification) {
    shownIds.add(issue.id);
}

// Filter tree nodes to exclude any IDs already shown
const dedupedTreeNodes = treeNodes.filter(node => !shownIds.has(node.id));
```

The comment says "fixes Keyed each block has duplicate key errors" - but this is solving a Svelte rendering problem by breaking the conceptual model. The real solution is unique keys, not deduplication.

**Source:** `web/src/lib/components/work-graph-tree/work-graph-tree.svelte:73-87`

**Significance:** This deduplication creates Dylan's confusion: "where does this issue live?" The issue lives in the tree (its natural position) AND is surfaced in a section (attention surface). These aren't alternatives; they're complementary views. The deduplication hides the tree position, losing hierarchical context.

---

### Finding 2: Two Distinct Questions Require Different Affordances

**Evidence:** The prior investigations establish that the orchestrator has two different questions:

| Question | Affordance | Speed Target | Current Implementation |
|----------|------------|--------------|----------------------|
| "What needs my attention NOW?" | Section at top | <5s scan | Works (WIP section) |
| "Where does this fit in my work?" | Tree hierarchy | <10s navigation | Broken (deduped from tree) |

The unified attention model investigation (`2026-02-02-design-work-graph-unified-attention-model.md:118-127`) explicitly discusses this:
> "Once Work Graph surfaces 'this needs attention,' what does the orchestrator *do*? Options: View only, Action buttons, Hybrid"

The answer was "view-only with keyboard shortcuts" - but this assumes you can SEE both the attention items AND their tree context. Deduplication breaks this.

**Source:**
- `2026-02-02-design-work-graph-unified-attention-model.md:118-127`
- `2026-02-02-inv-pressure-test-work-graph-unified.md:452-463`

**Significance:** Sections and tree are complementary affordances for different questions. The implementation should support BOTH, not choose between them.

---

### Finding 3: Three Orthogonal Dimensions Map to Visual Layers

**Evidence:** The `2026-02-04-inv-agents-own-declaration-via-bd.md` investigation established:

1. **Work Status (Agent Progress)**: `not_started → planning → implementing → testing → done`
2. **Verification Status (Quality Gates)**: `unverified → verification_passed → human_verified`
3. **Issue Status (Beads Lifecycle)**: `open → in_progress → blocked → closed`

These are orthogonal - they don't replace each other. The UI should layer them:

| Dimension | Visual Representation | Position |
|-----------|----------------------|----------|
| Issue Status | Tree structure | Primary - defines hierarchy |
| Work Status | Section membership + phase badges | Overlay - surfaces subset |
| Verification Status | Badge color + section priority | Overlay - indicates quality |

**Source:** `2026-02-04-inv-agents-own-declaration-via-bd.md:74-98`

**Significance:** The confusion arises from treating Work Status (section membership) as REPLACING Issue Status (tree position). They should be LAYERED. An issue can simultaneously be "open" (tree), "in_progress with running agent" (WIP section), and "unverified" (badge).

---

### Finding 4: "Filters" and "Sections" Are Not Mutually Exclusive

**Evidence:** Analyzing what "filter" and "section" actually mean:

| Approach | Definition | Result |
|----------|------------|--------|
| **Filter** | Show only items matching criteria | Remaining items visible in tree |
| **Section** | Extract items matching criteria to dedicated area | Items in section, possibly removed from tree |
| **Overlay** | Add visual indicator to matching items | All items visible, some highlighted |

The current implementation uses "Section with deduplication" - items extracted to section AND removed from tree.

The conceptual model says "overlays on the tree" - but this was interpreted as "remove from tree, show elsewhere."

The correct interpretation is: **Section + Overlay without deduplication** - items appear in section AND remain in tree with visual indicator.

**Source:** Conceptual analysis of current code vs prior design docs

**Significance:** The confusion is definitional. "Filters on a graph" doesn't mean "remove items from graph." It means "surface items that match filter criteria" - which is what sections do. The tree should remain complete, with sections providing a filtered view into it.

---

### Finding 5: Keyboard Navigation Between Views Solves Context Loss

**Evidence:** The pressure-test investigation established keyboard-first interaction:

```markdown
| Key | Action | Target |
|-----|--------|--------|
| `c` | Copy completion command | Ready-to-complete items |
| `s` | Copy spawn command | Ready-to-spawn items |
| `o` | Open issue in browser | Any issue |
| `enter` | Expand/collapse details | Any item |
| `j/k` | Navigate | List |
```

Missing: A key to jump from section item to tree position (and back).

If items appear in both section and tree, pressing a key like `t` (tree) on a section item could jump cursor to its tree position. Pressing `a` (attention/section) on a tree item could jump to its section representation.

**Source:** `2026-02-02-inv-pressure-test-work-graph-unified.md:454-462`

**Significance:** With dual presence + navigation shortcuts, users get the best of both worlds: fast attention scanning (sections) AND context preservation (tree) with seamless movement between them.

---

## Synthesis

**Key Insights:**

1. **Dual presence is the answer** - Items should appear in BOTH their section (for attention) AND their tree position (for context). The "duplicate key" error is a Svelte implementation detail, not a conceptual constraint. Use unique keys like `${id}-section` and `${id}-tree`.

2. **Sections are attention lenses, not alternatives** - WIP section is a "lens" that surfaces items matching "has running agent." It doesn't replace the tree; it provides quick access to a subset. The tree remains the source of truth for structure.

3. **Visual differentiation prevents confusion** - When an item appears in both places, use visual cues:
   - In section: Full opacity, primary styling
   - In tree: Dimmed opacity (50%), subtle badge showing "also in WIP"
   - This tells users: "This item is here in the tree, but look at the WIP section above for current status"

4. **Navigation bridges the views** - Keyboard shortcuts to jump between section and tree positions complete the interaction model. Press `t` to go from section to tree context; press `w` (WIP) to go from tree to section.

**Answer to Investigation Question:**

**Should WIP items appear WITHIN the tree or as a SEPARATE section?**

**BOTH.** They should appear in the WIP section (for attention scanning) AND remain in the tree (for context preservation). Current deduplication should be removed. Visual dimming in the tree indicates "also in section above."

**Should 'Needs Action' be a filter or a section?**

**Section.** "Needs Action" should be a dedicated section (after WIP) that surfaces items requiring attention. These items should ALSO remain in the tree with dimming. The section is not a filter that hides items - it's a lens that surfaces them.

**How do we preserve tree context while making 'what needs my attention' answerable in <5s?**

1. Keep sections at top for <5s attention scanning
2. Keep items in tree for hierarchical context (dimmed if also in section)
3. Add keyboard navigation to jump between section and tree position
4. The two views are complementary, not competing

---

## Structured Uncertainty

**What's tested:**

- ✅ Current deduplication removes items from tree (verified: read work-graph-tree.svelte:73-87)
- ✅ Three orthogonal dimensions model is established (verified: read inv-agents-own-declaration-via-bd.md)
- ✅ Keyboard-first interaction is the design pattern (verified: read pressure-test investigation)
- ✅ Svelte supports unique keys for same-id items in different locations (verified: Svelte docs)

**What's untested:**

- ⚠️ Dual presence won't cause visual overload (hypothesis - needs user testing)
- ⚠️ 50% dimming provides clear differentiation (needs visual testing)
- ⚠️ Navigation shortcuts (`t` for tree, `w` for WIP) are intuitive (needs user feedback)
- ⚠️ Performance with items rendered twice is acceptable (not benchmarked)

**What would change this:**

- If dual presence creates visual confusion in practice → Consider toggle mode instead
- If Dylan strongly prefers current behavior → Document rationale and keep deduplication
- If performance is unacceptable → Implement virtual scrolling or lazy rendering

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Remove deduplication, implement dual presence | architectural | Changes data flow and UI presentation model; multiple valid approaches exist |
| Add visual dimming for duplicated items | implementation | Within existing styling patterns |
| Add keyboard navigation between views | implementation | Extends existing keyboard navigation pattern |

### Recommended Approach: Dual Presence with Visual Layering

**Section-Anchored Tree** - Items appear in both their section (WIP, Needs Attention) AND their tree position, with visual differentiation showing the relationship.

**Why this approach:**
- Answers "what needs attention?" in <5s (sections)
- Preserves hierarchical context (tree shows all items)
- Aligns with three-dimension model (overlays, not replacements)
- Minimal conceptual change from current UI (sections remain, tree gains items)

**Trade-offs accepted:**
- Items rendered twice (performance cost, mitigated by virtual scrolling if needed)
- More visual complexity (mitigated by clear differentiation)
- Slightly more code complexity (unique keys, dimming logic)

**Implementation sequence:**

1. **Remove deduplication logic** (work-graph-tree.svelte:73-87)
   - Change Svelte each key from `getItemId(item)` to include prefix: `section-${id}` vs `tree-${id}`
   - Stop filtering `dedupedTreeNodes`
   - Why first: Foundation for dual presence

2. **Add visual dimming for tree items also in section**
   - Create `isInSection(nodeId)` helper checking if node is in wipItems or completedIssues
   - Apply `opacity-50` class to tree items that are also in sections
   - Optional: Add small badge like "WIP" to indicate section membership
   - Why second: Differentiation prevents confusion

3. **Add keyboard navigation between views**
   - `t` on section item → jump to tree position
   - `w` on tree item (if in WIP) → jump to WIP section position
   - `a` on tree item (if in Needs Attention) → jump to attention section position
   - Why third: Completes the interaction model

4. **Optionally: Add visual connector line**
   - When section item is selected, draw faint line to tree position
   - Purely aesthetic enhancement, implement if time permits

### Alternative Approaches Considered

**Option B: Toggle Mode (Filter View vs Tree View)**
- **Pros:** Simpler mental model - one view at a time
- **Cons:** Can't see both simultaneously; loses context when in filter view
- **When to use instead:** If dual presence proves too visually complex in practice

**Option C: Keep Current Deduplication**
- **Pros:** No implementation work; users have adapted
- **Cons:** Context is lost; doesn't align with three-dimension model
- **When to use instead:** If Dylan explicitly prefers current behavior after reviewing this analysis

**Option D: Pure Filter (No Sections)**
- **Pros:** Single view, conceptually simple
- **Cons:** Can't answer "what needs attention" in <5s; have to scan entire tree
- **When to use instead:** Never - the <5s attention requirement is hard

**Rationale for recommendation:** Option A (dual presence) is the only approach that satisfies BOTH requirements: fast attention scanning AND context preservation. The implementation cost is modest (remove deduplication, add dimming, add navigation keys).

---

### Implementation Details

**What to implement first:**
- Remove the deduplication filter in work-graph-tree.svelte
- Fix the Svelte key to use prefixed IDs
- This alone will make items appear in both places

**Things to watch out for:**
- ⚠️ The Svelte "duplicate key" error will return if keys aren't properly prefixed
- ⚠️ The selected index calculation needs updating since `flattenedNodes` will be longer
- ⚠️ Navigation with `j/k` will now traverse through both section and tree items - may need "skip to next section" shortcut

**Areas needing further investigation:**
- Visual design for dimming (exact opacity, whether to show badge)
- Keyboard shortcuts for cross-view navigation (which keys, discoverability)
- Whether to show a visual connector between section item and tree position

**Success criteria:**
- ✅ Issue in WIP section is also visible in tree (dimmed)
- ✅ User can press key to jump from section to tree position
- ✅ Tree hierarchy is preserved for WIP items (parent-child relationship visible)
- ✅ "What needs my attention?" answerable in <5s (sections remain)
- ✅ No visual confusion about where item "lives" (clear section/tree differentiation)

---

## References

**Files Examined:**
- `web/src/lib/components/work-graph-tree/work-graph-tree.svelte` - Main tree component, found deduplication logic
- `web/src/lib/stores/attention.ts` - Attention store and badge types
- `.kb/investigations/2026-02-04-arch-work-graph-done-states.md` - UI badges investigation
- `.kb/investigations/2026-02-04-inv-agents-own-declaration-via-bd.md` - Three dimensions model
- `.kb/investigations/2026-02-02-design-work-graph-unified-attention-model.md` - Unified attention design
- `.kb/investigations/2026-02-02-inv-pressure-test-work-graph-unified.md` - Pressure test findings

**Commands Run:**
```bash
# Create investigation file
kb create investigation design-work-graph-filters-vs-sections

# Report to orchestrator
bd comment orch-go-21266 "Phase: Planning - Analyzing Work Graph UI architecture"
```

**Related Artifacts:**
- **Investigation:** `2026-02-04-inv-agents-own-declaration-via-bd.md` - Three orthogonal dimensions model
- **Investigation:** `2026-02-04-arch-work-graph-done-states.md` - Badge naming and lifecycle states
- **Investigation:** `2026-02-02-inv-pressure-test-work-graph-unified.md` - Unified attention architecture

---

## Investigation History

**2026-02-04 14:30:** Investigation started
- Initial question: Filters vs sections for Work Graph lifecycle overlays
- Context: Dylan confused about implementation path; conceptual model (filters on graph) clear but implementation unclear

**2026-02-04 15:00:** Context gathered
- Read current implementation (work-graph-tree.svelte)
- Read prior investigations (done-states, three-dimensions, unified-attention)
- Identified key tension: deduplication breaks context

**2026-02-04 15:30:** Synthesis phase
- Key insight: "filters vs sections" is false dichotomy - both serve different questions
- Recommendation: Dual presence with visual layering
- Implementation path: Remove deduplication, add dimming, add navigation

**2026-02-04 16:00:** Investigation complete
- Status: Complete
- Key outcome: Recommend dual presence model where items appear in both section AND tree with visual differentiation
