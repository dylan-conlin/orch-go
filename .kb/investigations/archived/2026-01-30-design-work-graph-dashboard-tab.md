## Summary (D.E.K.N.)

**Delta:** The dashboard's 6 silos (Up Next, Frontier, Questions, Active Agents, Strategic Center, Ready Queue) fail because they're flat projections of a graph - what's needed is one unified explorable tree with two modes (Structure/Activity) sharing the same expansion mechanics.

**Evidence:** Reviewed current dashboard implementation (+page.svelte:890 lines, 6 separate sections). Each shows filtered flat lists. User feedback: "does the activity have to be separate from the beads tree? where is the chain?"

**Knowledge:** The mental model is unified - beads issues are nodes, agent sessions are execution attached to nodes, artifacts are outputs, blocking relationships are edges. Temporal questions ("where were we") need Activity mode; structural questions ("what's in this epic") need Structure mode.

**Next:** Implement Phase 1 - basic tree view with L0/L1 expansion and keyboard nav in new dashboard tab.

**Authority:** architectural - Cross-boundary change affecting dashboard architecture, data API, and user interaction patterns. Multiple valid approaches considered.

---

# Investigation: Work Graph Dashboard Tab

**Question:** How do we create a unified, explorable view of beads issues that shows hierarchy, blocking relationships, execution state, and artifacts - replacing the 6 silos with one integrated view?

**Started:** 2026-01-30
**Updated:** 2026-01-30
**Owner:** Dylan + Claude
**Phase:** Complete
**Next Step:** Create beads issue for implementation
**Status:** Complete

---

## Findings

### Finding 1: Current dashboard is 6 disconnected projections

**Evidence:** `web/src/routes/+page.svelte` contains 6 separate sections:
- UpNextSection (priority-sorted flat list)
- FrontierSection (ready/blocked/active/stuck grouping)
- QuestionsSection (isolated question entities)
- Active Agents (flat grid of running agents)
- DecisionCenter (strategic decisions)
- ReadyQueueSection (daemon queue visibility)

Each is a flat list filtered by some criteria. None show hierarchy, blocking relationships, or the chain of work.

**Source:** `web/src/routes/+page.svelte:466-556` (operational mode sections)

**Significance:** The problem isn't missing features - it's wrong information architecture. Users think in one unified graph, dashboard shows 6 slices.

---

### Finding 2: Graph API already exists

**Evidence:** `/api/beads/graph` endpoint returns:
```go
type BeadsGraphAPIResponse struct {
    Nodes      []GraphNode `json:"nodes"`      // id, title, type, status, priority, source
    Edges      []GraphEdge `json:"edges"`      // from, to, type (blocks, parent-child, relates_to)
    NodeCount  int         `json:"node_count"`
    EdgeCount  int         `json:"edge_count"`
}
```

Has `scope=focus` (in_progress + blockers + P0/P1) and `scope=open` modes.

**Source:** `cmd/orch/serve_beads.go:616-721`

**Significance:** Data layer exists. Need UI layer that consumes it properly.

---

### Finding 3: Hierarchy is encoded in IDs

**Evidence:** Beads uses ID convention for parent-child:
- `orch-go-gy1o4` (parent epic)
- `orch-go-gy1o4.1` (child sub-epic)
- `orch-go-gy1o4.2` (child sub-epic)

**Source:** `bd list --status open --json | jq` showing ID patterns

**Significance:** Tree structure can be inferred from IDs without explicit parent_id field. API may need enhancement to expose this as proper edges.

---

### Finding 4: User's core questions are temporal AND structural

**Evidence:** User stated first questions are:
- "where were we" (temporal - last activity)
- "what's next" (priority/readiness)
- "what did we do yesterday" (temporal - history)
- "what are we in the middle of" (state - in_progress)

Also wants: "if a question was answered by an architect and the architect created 3 follow up questions and 1 decision, I want to see the chain" (lineage/provenance).

**Source:** Conversation context

**Significance:** Need two modes - Structure (hierarchy) and Activity (time-sorted) - to answer both question types. Same expansion mechanics, different organization.

---

## Synthesis

**Key Insights:**

1. **Unified mental model** - User thinks of one work graph. Issues are nodes, agents are execution, artifacts are outputs, blocks are edges. Dashboard should reflect this.

2. **Two temporal frames** - "Show me the shape" (structure) vs "show me what happened" (activity). Both valid, both needed. Same expansion mechanics serve both.

3. **Lineage is the missing link** - Current sections don't show what spawned what. When an architect answers a question and creates follow-ups, that chain should be visible.

4. **Pinned attention section** - P0/P1, stuck agents, and pending decisions should always be visible regardless of mode. "Needs your attention" shouldn't be buried.

**Answer to Investigation Question:**

Create a new dashboard tab ("Work Graph") with:
- Two modes (Structure/Activity) toggled with Tab key
- Tree/list with 3-level expansion (L0: row, L1: details, L2: artifacts)
- Vim-style keyboard navigation (j/k/l/h/enter/esc)
- Pinned section at top for urgent items
- Lineage tracking ("Spawned from", "Created")

This replaces the need for 6 separate sections by providing one explorable view that answers both structural and temporal questions.

---

## Structured Uncertainty

**What's tested:**

- ✅ Graph API exists and returns nodes/edges (verified: read serve_beads.go)
- ✅ Hierarchy encoded in IDs (verified: bd list --json output)
- ✅ Current dashboard is 6 separate sections (verified: +page.svelte review)

**What's untested:**

- ⚠️ Keyboard nav in web will feel vim-like (needs implementation + testing)
- ⚠️ Performance with large issue sets (not benchmarked)
- ⚠️ Lineage data is available via existing APIs (may need new endpoints)

**What would change this:**

- If lineage data isn't accessible, L1 expansion would be limited
- If performance is poor, may need virtualized scrolling
- If keyboard handling is too fragile in web, may need different nav approach

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Create Work Graph tab with tree view | architectural | Cross-boundary: new tab, new data flows, affects overall dashboard UX |
| Extend API for lineage data | implementation | Stays within existing API patterns |
| Keyboard navigation implementation | implementation | UI-only, no architectural impact |

### Recommended Approach ⭐

**Tree View with Dual Modes** - Single explorable tree with Structure (hierarchy) and Activity (time-sorted) modes, sharing expansion mechanics and keyboard nav.

**Why this approach:**
- Matches user's mental model of unified work graph
- Answers both structural and temporal questions
- Progressive disclosure (overview → details → artifacts) reduces cognitive load
- Keyboard-first feels like TUI, matches orchestrator workflow

**Trade-offs accepted:**
- More complex than single flat list
- Requires API extension for lineage
- Web keyboard handling needs careful focus management

**Implementation sequence:**
1. Phase 1: Basic tree with L0/L1, keyboard nav, Structure mode only
2. Phase 2: Activity mode with time grouping
3. Phase 3: Pinned section, lineage tracking, artifact rendering (L2)
4. Phase 4: Polish - focus management, search, help overlay

### Alternative Approaches Considered

**Option B: Flat list with contextual panel**
- **Pros:** Simpler, no hierarchy assumptions
- **Cons:** Doesn't show chain naturally, still feels like "two things"
- **When to use instead:** If hierarchy proves too fragile or confusing

**Option C: Full graph visualization (network diagram)**
- **Pros:** Shows all relationships visually
- **Cons:** User rejected - "most work wasn't actually dependent"
- **When to use instead:** If blocking relationships become dominant

**Rationale for recommendation:** User explicitly wanted "tree/outline view that's RICH" with chain visibility through expansion. Dual modes address the temporal vs structural question split.

---

### Implementation Details

**What to implement first:**
- New route: `/work-graph` or tab within existing dashboard
- WorkGraphStore: fetch from `/api/beads/graph?scope=open`
- Tree component with keyboard event handling
- L0 row rendering with status indicators

**Things to watch out for:**
- ⚠️ Focus management - clicks, tab changes, text fields can steal focus
- ⚠️ ID hierarchy parsing - edge cases like `orch-go-20944.5` (is that a child?)
- ⚠️ SSE integration - active agent status should update in real-time

**Areas needing further investigation:**
- How to correlate beads issues with agent sessions (for "has active agent" indicator)
- How to get lineage data (spawned from, created by)
- Whether to virtualize the list for large issue sets

**Success criteria:**
- ✅ Can answer "where were we" by opening Activity mode
- ✅ Can answer "what's in this epic" by expanding in Structure mode
- ✅ Can see the chain (question → architect → outputs) through expansion
- ✅ Keyboard-only navigation feels natural (no mouse required)
- ✅ Urgent items visible without scrolling (pinned section)

---

## References

**Files Examined:**
- `web/src/routes/+page.svelte` - Current dashboard structure
- `web/src/lib/components/frontier-section/frontier-section.svelte` - Frontier implementation
- `web/src/lib/components/up-next-section/up-next-section.svelte` - Up Next implementation
- `web/src/lib/stores/beads.ts` - Beads store interface
- `cmd/orch/serve_beads.go:600-800` - Graph API implementation

**Commands Run:**
```bash
# Check beads issue structure
bd show orch-go-kz7zr --json

# Find issues with hierarchy
bd list --status open --json | jq '.[] | select(.id | startswith("orch-go-gy1o4"))'
```

**Related Artifacts:**
- **Model:** `.kb/models/decidability-graph.md` - Conceptual foundation for work graph
- **Decision:** `.kb/decisions/2026-01-30-recommendation-authority-classification.md` - Authority levels for questions

---

## Investigation History

**2026-01-30 18:00:** Investigation started
- Initial question: How to replace 6 dashboard silos with unified view
- Context: Dylan frustrated that each section is a flat projection, wants to see "the chain"

**2026-01-30 18:30:** Design exploration
- Explored 3 approaches: Tree with expansion, Flat list with panel, Graph visualization
- User rejected graph visualization, confirmed tree/outline is desired

**2026-01-30 18:45:** Design challenge and refinement
- Identified gap: temporal questions need Activity mode, not just Structure mode
- Added dual-mode design, pinned section, lineage tracking

**2026-01-30 19:00:** Investigation completed
- Status: Complete
- Key outcome: Unified tree view with Structure/Activity modes, vim-style keyboard nav, 3-level expansion
