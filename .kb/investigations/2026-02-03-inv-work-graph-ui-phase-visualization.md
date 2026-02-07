<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Layer calculation exists in beads CLI (`bd graph`) but is not exposed via API or Work Graph UI - phased plan visualization requires backend layer field + frontend phase grouping component.

**Evidence:** Tested `bd graph orch-go-21202` (shows layers), `curl /api/beads/graph` (no layer field), read computeLayout() algorithm in beads/cmd/bd/graph.go:322-419, read work-graph.ts buildTree() (parent-child only, no layer logic).

**Knowledge:** Backend calculation + frontend grouping is best approach (reuses proven algorithm, single source of truth, enables future clients). Port ~100 lines from beads graph.go to serve_beads.go, add `layer: int` to GraphNode, render as collapsible phase sections in UI.

**Next:** Architectural decision required - Orchestrator should decide whether to implement phase visualization, and if so, which UI pattern (sections vs swimlanes vs badges). Ready for implementation if approved.

**Authority:** architectural - Cross-component decision (API + UI changes), multiple valid UI patterns, requires synthesis of trade-offs between backend vs frontend calculation

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Authority: implementation - Tactical fix within existing patterns, no architectural impact

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Authority: Classify by who decides - implementation (worker within scope), architectural (orchestrator across boundaries), strategic (Dylan for irreversible/value choices)
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Work Graph Ui Phase Visualization

**Question:** What exists in the work-graph UI for phase visualization, and what's needed to show phased plan orchestration (dependency-based phase sequencing)?

**Started:** 2026-02-03
**Updated:** 2026-02-03
**Owner:** Investigation Worker
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Starting Investigation - Current API and UI Structure

**Evidence:** Read key files:
- web/src/routes/work-graph/+page.svelte (441 lines) - Main UI page
- web/src/lib/stores/work-graph.ts (248 lines) - Store with buildTree logic
- cmd/orch/serve_beads.go (1331 lines) - API endpoint handler
- `bd graph` command has layer calculation built-in (help text mentions "left-to-right" execution order)

**Source:** 
- /Users/dylanconlin/Documents/personal/orch-go/web/src/routes/work-graph/+page.svelte
- /Users/dylanconlin/Documents/personal/orch-go/web/src/lib/stores/work-graph.ts
- /Users/dylanconlin/Documents/personal/orch-go/cmd/orch/serve_beads.go
- `bd graph --help`

**Significance:** Need to understand what data the API currently returns and whether it includes phase/layer information

---

### Finding 2: Layer Calculation Exists in beads CLI

**Evidence:**
- `bd graph` command performs topological sort to calculate layers (beads/cmd/bd/graph.go:322-419)
- Layer 0 = nodes with no blocking dependencies (ready to start)
- Layer N = nodes whose dependencies are all in layers 0 through N-1
- CLI displays this as "Layer 0 (ready)", "Layer 1", etc.
- Example output: `bd graph orch-go-21197` shows "Layer 0 (ready)" with the issue

**Source:**
- /Users/dylanconlin/Documents/personal/beads/cmd/bd/graph.go:322-419 (computeLayout function)
- `bd graph orch-go-21197` (tested - shows layer visualization)
- `bd graph --help` (describes execution order: left-to-right, same column can run in parallel)

**Significance:** The logic for phase/layer calculation already exists and is battle-tested in beads CLI - we don't need to implement it from scratch

---

### Finding 3: API Does NOT Return Layer Information

**Evidence:**
- `/api/beads/graph` returns nodes with: id, title, type, status, priority, source
- Edges have: from, to, type
- No layer/phase field in node data
- `bd graph --all --json` also doesn't include layer data in export (tested)

**Source:**
- cmd/orch/serve_beads.go:600-627 (GraphNode and GraphEdge type definitions)
- `curl https://localhost:3348/api/beads/graph?scope=open | jq '.nodes[0]'` (tested - no layer field)
- `bd graph --all --json | jq '.nodes[0] | keys'` (tested - confirmed no layer field)

**Significance:** To add phase visualization, we need EITHER:
1. Add layer calculation to API response (backend approach)
2. Calculate layers client-side in the UI (frontend approach)

---

### Finding 4: UI Has No Phase/Layer Logic

**Evidence:**
- work-graph.ts buildTree() builds parent-child hierarchy only (lines 168-245)
- Tree structure based on `parseParentId()` from issue ID patterns (orch-go-X.Y) 
- Blocking dependencies (`blocks` edges) are tracked as `blocked_by`/`blocks` arrays but not used for visualization
- No layer/phase grouping or calculation in frontend

**Source:**
- web/src/lib/stores/work-graph.ts:168-245 (buildTree function)
- web/src/routes/work-graph/+page.svelte:1-441 (no phase grouping logic)

**Significance:** The UI would need NEW logic to:
1. Calculate layers from blocking dependencies OR receive layer data from API
2. Group/render nodes by layer (Phase 1, Phase 2, etc.)
3. Show phase progress and blocking relationships visually

---

### Finding 5: No Blocking Dependencies in Current Graph

**Evidence:**
- Queried `/api/beads/graph?scope=open` - 69 nodes, 5 edges
- All 5 edges have empty `type` field (meaning parent-child, not blocks)
- No `blocks` type edges in current active work

**Source:**
- `curl 'https://localhost:3348/api/beads/graph?scope=open' | jq '.edges'` (tested)

**Significance:** Cannot test phase visualization with real data currently - would need to create test issues with blocking dependencies or wait for a phased plan to be created

---

## Synthesis

**Key Insights:**

1. **Layer calculation is solved, just not exposed** - The beads CLI already has production-quality topological sort for layer calculation (Finding 2). We don't need to design or implement the algorithm - we need to either port it to the API/UI or expose the existing calculation.

2. **Current API is phase-blind** - The `/api/beads/graph` endpoint returns nodes and edges but no layer/phase information (Finding 3). The UI can see dependencies but doesn't know which phase each issue belongs to.

3. **Frontend has no blocking-aware visualization** - The UI's `buildTree()` only handles parent-child hierarchy from ID patterns, not blocking dependencies (Finding 4). Even if we added layer data to the API, the UI would need new rendering logic for phase grouping.

4. **User wants to SEE phased orchestration** - The context mentions validation that "phased plan orchestration works" with Phase 1 ready immediately and Phase 2 becoming ready when Phase 1 closes. The gap is visibility - orchestrator can see this via `bd graph`, but the web UI cannot.

**Answer to Investigation Question:**

**What exists:**
- `bd graph` CLI command with layer calculation (topological sort algorithm)
- `/api/beads/graph` endpoint returning nodes/edges without layer data
- Work-graph UI with parent-child tree visualization

**What's needed for phase visualization:**
1. **Backend:** Add layer calculation to `/api/beads/graph` response (port computeLayout logic from beads or call `bd graph --all --json` and augment with layers)
2. **Frontend:** Add phase grouping UI (similar to `bd graph` CLI output: "Phase 1 (ready)", "Phase 2 (blocked)", etc.)
3. **Design decisions:** How to show phases - accordion sections? Swimlanes? Tree with layer badges?

**What would be useful for phased work visibility** (from spawn context):
- **Staging view:** What's blocked waiting for what? (show blocked_by relationships)
- **Working view:** What's currently being executed? (filter by status=in_progress, show phase)  
- **Finished view:** What completed and unblocked what? (recently closed with dependents)
- **Progress indicators:** How far through a phase/plan? (count closed vs total per layer)

---

## Structured Uncertainty

**What's tested:**

- ✅ **bd graph calculates layers** - Ran `bd graph orch-go-21202`, confirmed CLI displays "Layer 0 (ready)" with topological sort
- ✅ **API returns nodes/edges without layers** - Ran `curl https://localhost:3348/api/beads/graph?scope=open | jq '.nodes[0]'`, confirmed no layer field
- ✅ **bd graph --all --json has no layer data** - Ran `bd graph --all --json | jq '.nodes[0] | has("layer")'`, returned false
- ✅ **No blocking dependencies in current graph** - Queried `/api/beads/graph`, found 5 edges all with empty type (parent-child), zero `blocks` edges
- ✅ **buildTree() doesn't use blocking deps** - Read work-graph.ts:168-245, confirmed only parent-child hierarchy, not layer calculation

**What's untested:**

- ⚠️ **computeLayout port will work in Go** - Algorithm is in beads (Go) already, but different package/context. Assume portable but untested in serve_beads.go
- ⚠️ **Phase grouping UI will be usable** - No mockup or user testing, just conceptual design
- ⚠️ **Performance with 1000+ nodes** - computeLayout is O(n*m) but haven't benchmarked with real scale
- ⚠️ **User actually wants this feature** - Inferred from spawn context ("user wants to SEE this") but no direct user interview

**What would change this:**

- **If bd graph shows inconsistent layers across runs** → Algorithm might not be deterministic, would affect API reliability
- **If API response time >1s with layer calculation** → Would need caching or optimization
- **If user feedback says "phase view is confusing"** → Would need to reconsider UI pattern (swimlanes vs sections vs badges)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add phase visualization to Work Graph UI | **architectural** | Cross-component decision (API + UI), multiple valid approaches (backend vs frontend calculation, different UI patterns), requires synthesis of trade-offs |

**Authority Levels:**
- **implementation**: Worker decides within scope (reversible, single-scope, clear criteria, no cross-boundary impact)
- **architectural**: Orchestrator decides across boundaries (cross-component, multiple valid approaches, requires synthesis)
- **strategic**: Dylan decides on direction (irreversible, resource commitment, value judgment, premise-level question)

### Recommended Approach ⭐

**Backend Layer Calculation with Frontend Phase Grouping** - Add layer field to `/api/beads/graph` response, then render phases as collapsible sections in the UI

**Why this approach:**
- **Reuses proven algorithm** - Port `computeLayout()` from beads/cmd/bd/graph.go (lines 322-419), already battle-tested in CLI
- **Single source of truth** - Backend calculates once, frontend displays consistently across all clients
- **Progressive enhancement** - API change is additive (doesn't break existing consumers), UI can fall back to tree view if layers missing
- **Enables future features** - Once backend has layer data, other UIs (CLI, mobile) can use it too

**Trade-offs accepted:**
- **Backend complexity** - Adds ~100 lines of Go code to serve_beads.go (acceptable - it's a port, not new logic)
- **Can't filter by layer client-side** - If user wants "show only Phase 2", frontend can't recalculate without all dependencies (acceptable - unlikely use case)

**Implementation sequence:**
1. **Backend: Add layer calculation** - Port computeLayout() to serve_beads.go, add `layer: int` field to GraphNode, calculate during buildFocusGraph() and buildFullGraph()
2. **Frontend: Add phase grouping component** - New `PhaseGroup` component that groups TreeNode[] by layer, renders as collapsible sections like "Phase 1 (ready)", "Phase 2 (blocked)"
3. **Frontend: Wire up phase view toggle** - Add "Tree View" vs "Phase View" toggle in work-graph page header

### Alternative Approaches Considered

**Option B: Frontend-only layer calculation**
- **Pros:** No backend changes, all logic in TypeScript, easier to iterate on UI
- **Cons:** 
  - Duplicates algorithm (now in beads Go + work-graph.ts TypeScript)
  - Potential for inconsistency (CLI shows Layer 2, UI shows Layer 3)
  - Frontend must receive full dependency graph to calculate (can't work with filtered/paginated data)
- **When to use instead:** If backend is frozen or API cannot change

**Option C: Call bd graph --all --json from backend**
- **Pros:** Zero algorithm duplication, guaranteed consistency with CLI
- **Cons:** 
  - Spawns subprocess on every API call (performance hit)
  - JSON parsing overhead
  - bd command might not be available in all deployment environments
  - Harder to debug (subprocess failures)
- **When to use instead:** If backend team doesn't want to maintain Go port of algorithm

**Option D: Add phase badges to existing tree view**
- **Pros:** No new UI component, minimal change
- **Cons:**
  - Doesn't show "what can run in parallel" (key insight from phased plans)
  - User has to mentally group items by phase number
  - Doesn't show blocking relationships clearly
- **When to use instead:** If phase grouping is deemed too complex for MVP

**Rationale for recommendation:** Option A (backend calculation + frontend grouping) provides the best balance of code reuse (port proven algorithm), single source of truth (backend calculates), and UX clarity (phase grouping makes parallelism visible). The 100 lines of Go code is acceptable complexity for the value delivered.

---

### Implementation Details

**What to implement first:**
1. **Backend layer calculation (MVP)** - Port computeLayout() to serve_beads.go:handleBeadsGraph(), add `layer` field to GraphNode struct, return in JSON response
2. **Verification** - Add test: create 3 issues with blocking deps (A blocks B, B blocks C), verify API returns layer:0 for A, layer:1 for B, layer:2 for C
3. **Frontend display (optional but recommended)** - If backend works, add simple phase grouping UI

**Things to watch out for:**
- ⚠️ **Cycles in dependency graph** - bd graph handles with fallback to layer 0 (line 389), API should do same
- ⚠️ **Parent-child vs blocks dependencies** - Only `blocks` type should affect layers, parent-child is hierarchy not sequencing (beads/cmd/bd/graph.go:334 filters for `types.DepBlocks`)
- ⚠️ **Empty type field means parent-child** - API edges with `type: ""` should be treated as parent-child, not blocks (Finding 5)
- ⚠️ **Performance with large graphs** - computeLayout is O(n*m) where n=nodes, m=max_dependencies_per_node. Should be fine for <1000 nodes but consider caching for very large graphs

**Areas needing further investigation:**
- **UI design for phase grouping** - Should phases be collapsible sections? Swimlanes? Kanban columns? Needs design mockup
- **Phase progress indicators** - How to show "Phase 1: 3/5 complete" - count per layer? Requires status awareness
- **Filter interactions** - If user filters to only show P0 issues, how do phases display? Partial layers? Skip empty layers?

**Success criteria:**
- ✅ API returns `layer` field for all nodes in `/api/beads/graph` response
- ✅ Layer 0 contains only nodes with no blocking dependencies (can start immediately)
- ✅ Layer N contains only nodes whose blockers are all in layers 0..N-1
- ✅ UI displays phases in a way that makes "what can run in parallel" obvious (if UI implemented)
- ✅ `bd graph` CLI and web UI show consistent layer numbers for the same issue

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/orch-go/web/src/routes/work-graph/+page.svelte` (441 lines) - Main Work Graph UI page
- `/Users/dylanconlin/Documents/personal/orch-go/web/src/lib/stores/work-graph.ts` (248 lines) - Store with buildTree() logic
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/serve_beads.go` (1331 lines) - API endpoint for /api/beads/graph
- `/Users/dylanconlin/Documents/personal/beads/cmd/bd/graph.go` (651 lines) - Layer calculation algorithm in computeLayout()

**Commands Run:**
```bash
# Check API structure
curl -s 'https://localhost:3348/api/beads/graph?scope=open' | jq '{node_count, edge_count, sample_node: .nodes[0], sample_edge: .edges[0]}'

# Test bd graph layer visualization
bd graph orch-go-21202

# Check if bd graph JSON includes layers
bd graph --all --json | jq '{node_count, edge_count, has_layer_info: (.nodes[0] | has("layer"))}'

# Find issues with dependencies
bd list --json --limit 20 | jq -r '.[] | select(.dependency_count > 0 or .dependent_count > 0) | "\(.id) - deps:\(.dependency_count) blocked_by:\(.dependent_count)"'

# View dependency structure
bd show orch-go-21202 --json | jq '{id: .[0].id, title: .[0].title, dependencies: .[0].dependencies, dependents: .[0].dependents}'

# Check for blocking edges
curl -s 'https://localhost:3348/api/beads/graph?scope=open' | jq '.edges[] | select(.type == "blocks")'
```

**External Documentation:**
- N/A

**Related Artifacts:**
- **Guide:** `.kb/guides/dashboard.md` - Dashboard architecture guide (work-graph is separate UI, not covered in detail)
- **Spawn Context:** Referenced "phased plan orchestration" validation and user wanting to SEE phases

---

## Investigation History

**[YYYY-MM-DD HH:MM]:** Investigation started
- Initial question: [Original question as posed]
- Context: [Why this investigation was initiated]

**[YYYY-MM-DD HH:MM]:** [Milestone or significant finding]
- [Description of what happened or was discovered]

**[YYYY-MM-DD HH:MM]:** Investigation completed
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]
