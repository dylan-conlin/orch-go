## Summary (D.E.K.N.)

**Delta:** Focus-scoped comprehension doesn't require a new concept - it requires making Work Graph scope-aware and adding view modes. The three axes (epic, focus, phase) are conflated in user's mind because Work Graph doesn't expose them cleanly.

**Evidence:** Analyzed `orch focus` (goal string, no visibility), epic model (parent-child works, UI doesn't scope well), `bd graph` (phase calculation exists, not in UI), Work Graph (tree view only, no scope persistence).

**Knowledge:** Epic = structural containment (underused, not wrong). Focus = attention scoping (should be saved scope filter). Phase = view mode within any scope (orthogonal to both). The user's pain is "I can't see my plan" because Work Graph doesn't persist scope or offer phase view.

**Next:** Implement recommended approach - add scope persistence to Work Graph (saves to `orch focus`), add Phase View mode (reuses `bd graph` layer calculation), add Status View mode (grouped by status).

**Authority:** architectural - Cross-component (Work Graph UI + orch focus + beads API), multiple valid approaches, requires synthesis of three conceptual axes.

---

# Investigation: Focus-Scoped Work Comprehension Design

**Question:** How should 'focus' work for plan visibility? What should define a focus, how does it relate to epics and phases, and what's the UX direction?

**Started:** 2026-02-03
**Updated:** 2026-02-03
**Owner:** Architect Worker
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Current Tools Serve Different Purposes But Don't Integrate

**Evidence:** 
- `orch focus`: Sets a goal string with optional beads ID. Purpose is "strategic alignment" - helps orchestrator remember north star. Does NOT display work.
- Epics (beads parent-child): Structural grouping. `bd list --parent X` shows children. Works for hierarchy.
- `bd graph`: Shows execution phases based on blocking dependencies. Layer calculation is solid (topological sort). CLI-only.
- Work Graph UI: Tree view based on parent-child hierarchy. Can filter by status/priority. No "scope to epic and show phases" mode.

**Source:**
- `orch focus --help` - "helps orchestrators stay aligned with priorities and avoid drift"
- `bd graph orch-go-21202` - Shows "Layer 0 (ready)", phase-based display
- Prior phase visualization investigation confirmed layer calculation not in API

**Significance:** User's pain ("can't see my plan") exists because these tools don't connect.

---

### Finding 2: Three Organizational Axes Are Distinct Concepts

**Evidence:**
- **Epic (Hierarchy)**: Answers "what contains what?" Parent-child via beads ID patterns. Structural, not temporal.
- **Focus (Attention)**: Answers "what am I thinking about?" Currently just a string goal.
- **Phase (Sequencing)**: Answers "what runs when?" Based on blocking dependencies. Temporal, not structural.

Example: Epic `pw-123` has 5 children. 2 children in Phase 1 (no blockers), 3 in Phase 2 (blocked by Phase 1). User's focus is `pw-123`. Focus ≠ Phase: Focus is WHERE you're looking, Phase is WHEN things run.

**Significance:** The user conflates these because the tooling conflates them. Clarifying enables clean design.

---

### Finding 3: Epic Model is Underused, Not Wrong

**Evidence:**
- Current epic structure works: `bd list --parent X` returns children, `bd graph X` shows the plan
- Problem is visibility: You can query epic children, but Work Graph doesn't make this easy
- Missing: "scope to this epic" persistent mode, "epic progress" view, phase grouping within epic

**Significance:** We don't need a new "focus" concept. We need Work Graph to make epic-scoped views first-class.

---

### Finding 4: Phase Visualization Is Ready to Implement

**Evidence:**
- Layer calculation exists in `bd graph` (beads/cmd/bd/graph.go:322-419)
- Prior investigation recommended: "Add layer field to API, render phases as collapsible sections"
- Implementation estimate: ~100 lines of Go to port calculation to serve_beads.go

**Significance:** Phase view is a UI feature, not a conceptual problem.

---

### Finding 5: Focus Should Be "Saved Scope" Not New Concept

**Evidence:**
Work Graph already supports scoping: `?scope=open`, `?parent=X`. What's missing is persistence.

`orch focus` already stores: goal, beads_id, set_at. Natural evolution: When beads_id is set, Work Graph loads scoped to that issue.

**Significance:** Focus-scoped comprehension is largely wiring existing pieces together.

---

## Synthesis

**Key Insights:**

1. **The conflation is a UX problem, not conceptual.** Epic, Focus, Phase are distinct. Tooling conflates them. Fix is UI.

2. **Focus = Saved Scope.** Rather than inventing new entity, focus is persistent scope filter.

3. **Phase = View Mode.** Within any scope, view by hierarchy (tree), execution order (phase), or status.

4. **Epic is the primary scope mechanism.** For most use cases, user wants one epic's children grouped by phase/status.

**Answer to Investigation Questions:**

**What should define a 'focus'?**
A focus is a saved scope filter. Primary mechanism: an issue ID (usually an epic). `orch focus --issue` already supports this.

**Is the current epic model wrong, or underused?**
Underused. The gap is that Work Graph doesn't make epic-scoped, phase-grouped views easy to access.

**What's the right entry point/UX?**
Multiple entry points to same view:
1. CLI: `orch focus --issue pw-123` → Work Graph loads scoped
2. UI: Click "Set as Focus" → Saves to orch focus
3. Ad-hoc: Use Work Graph filters without saving

**How does this relate to phase visualization?**
Phase is a view mode within any scope. Complements Tree View and Status View. Should implement as recommended in prior investigation.

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add Phase View to Work Graph | architectural | Cross-component (API + UI) |
| Wire orch focus to Work Graph scope | architectural | Connects two components |
| Add Status View grouping | implementation | Single component (UI) |

### Recommended Approach: Focus as Saved Scope + View Modes

**Why this approach:**
- Minimal new concepts: Reuses epic, focus, phase (all exist)
- Incremental delivery: Each piece ships independently
- Clear mental model: Focus = WHERE, View = HOW

**Trade-offs accepted:**
- Focus limited to single issue scope (acceptable, epic is primary use case)
- Cross-epic focus is second-class (acceptable for MVP)

**Implementation sequence:**

1. **Phase 1: Add Phase View to Work Graph**
   - Add `layer` field to `/api/beads/graph` (port computeLayout)
   - Add "Phase View" toggle in UI
   - Render issues grouped by layer with progress summary

2. **Phase 2: Wire orch focus to Work Graph**
   - Work Graph reads `orch focus --json` on load
   - If focus has beads_id, auto-scope
   - Add "Set as Focus" button
   - Add scope breadcrumb

3. **Phase 3: Add Status View**
   - Group by status: Ready | In Progress | Verifying | Done
   - Show counts

4. **Phase 4: Comprehension features**
   - Progress summary per phase
   - Completeness check
   - "Related issues" section

### Alternative Approaches Considered

**Option B: Focus as First-Class Entity**
- New focus.json with issue list, label filter
- Pros: Most flexible
- Cons: New concept, complexity
- When: If cross-epic focus proves primary

**Option C: Epics-Only (No Focus)**
- Work Graph scoped via URL, user bookmarks
- Pros: Simplest
- Cons: Loses persistence
- When: If focus state unnecessary

---

## UX Sketch

```
┌─────────────────────────────────────────────────────────────────────┐
│  Work Graph                                                         │
│  ┌────────────────────────────────────────────────────────────┐    │
│  │ Focus: pw-123 "Ship Detection MVP"     [Clear Focus]       │    │
│  └────────────────────────────────────────────────────────────┘    │
│                                                                     │
│  View: [Tree] [Phase ●] [Status]                                   │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  Phase 1 (Ready) ─────────────────────────────────── 2/4 ✓        │
│  ├─ ✓ pw-123.1  Setup infrastructure                               │
│  ├─ ✓ pw-123.2  Basic scraping                                     │
│  ├─ ○ pw-123.3  Notification service              [in_progress]    │
│  └─ ○ pw-123.4  Dashboard widget                  [open]           │
│                                                                     │
│  Phase 2 (Blocked) ───────────────────────────────── 0/3          │
│  ├─ ○ pw-123.5  Alert integration                 [blocked]        │
│  ├─ ○ pw-123.6  Price charts                      [blocked]        │
│  └─ ○ pw-123.7  Performance                       [blocked]        │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│  Summary: 8 issues │ 2 done │ 1 in progress │ 5 blocked            │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Success Criteria

- User can set focus via `orch focus --issue X` and see Work Graph scoped
- Work Graph offers Phase View that groups by execution layer
- Phase View shows progress: "Phase 1: 3/5 ✓"
- User can validate completeness: "All 8 children shown"

---

## References

**Prior Investigations:**
- `.kb/investigations/2026-02-03-inv-work-graph-ui-phase-visualization.md` - Phase visualization design
- `.kb/investigations/2026-02-02-design-work-graph-unified-attention-model.md` - Work Graph attention surface

**Commands Run:**
- `orch focus --help` - Verified focus command structure
- `bd graph orch-go-21202` - Tested phase layer visualization
- `bd list --parent orch-go-21193 --json` - Verified epic children query
