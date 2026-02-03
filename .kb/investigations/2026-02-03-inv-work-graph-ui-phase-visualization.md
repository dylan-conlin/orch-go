<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** [What was discovered/answered - the key finding in one sentence]

**Evidence:** [Primary evidence that supports the conclusion - test results, observations]

**Knowledge:** [What was learned - insights, constraints, or decisions made]

**Next:** [Recommended action - close, implement, investigate further, or escalate]

**Authority:** [implementation | architectural | strategic] - [Brief rationale for authority level - see Recommendation Authority section below]

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
**Phase:** Investigating
**Next Step:** Examine API data structure and bd graph layer calculation
**Status:** In Progress

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

1. **[Insight title]** - [Explanation of the insight, connecting multiple findings]

2. **[Insight title]** - [Explanation of the insight, connecting multiple findings]

3. **[Insight title]** - [Explanation of the insight, connecting multiple findings]

**Answer to Investigation Question:**

[Clear, direct answer to the question posed at the top of this investigation. Reference specific findings that support this answer. Acknowledge any limitations or gaps.]

---

## Structured Uncertainty

**What's tested:**

- ✅ [Claim with evidence of actual test performed - e.g., "API returns 200 (verified: ran curl command)"]
- ✅ [Claim with evidence of actual test performed]
- ✅ [Claim with evidence of actual test performed]

**What's untested:**

- ⚠️ [Hypothesis without validation - e.g., "Performance should improve (not benchmarked)"]
- ⚠️ [Hypothesis without validation]
- ⚠️ [Hypothesis without validation]

**What would change this:**

- [Falsifiability criteria - e.g., "Finding would be wrong if X produces different results"]
- [Falsifiability criteria]
- [Falsifiability criteria]

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

Classify each recommendation by authority level to route to the appropriate decision-maker:

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| [Primary recommendation from investigation] | implementation / architectural / strategic | [Why this authority level - stays inside scope? reaches across boundaries? involves irreversible choice?] |

**Authority Levels:**
- **implementation**: Worker decides within scope (reversible, single-scope, clear criteria, no cross-boundary impact)
- **architectural**: Orchestrator decides across boundaries (cross-component, multiple valid approaches, requires synthesis)
- **strategic**: Dylan decides on direction (irreversible, resource commitment, value judgment, premise-level question)

**Classification test:** "Does this decision stay inside my scoped context, or does it reach out?"
- Stays inside → implementation
- Reaches to other components/agents → architectural
- Reaches to values/direction/irreversibility → strategic

### Recommended Approach ⭐

**[Approach Name]** - [One sentence stating the recommended implementation]

**Why this approach:**
- [Key benefit 1 based on findings]
- [Key benefit 2 based on findings]
- [How this directly addresses investigation findings]

**Trade-offs accepted:**
- [What we're giving up or deferring]
- [Why that's acceptable given findings]

**Implementation sequence:**
1. [First step - why it's foundational]
2. [Second step - why it comes next]
3. [Third step - builds on previous]

### Alternative Approaches Considered

**Option B: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Option C: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Rationale for recommendation:** [Brief synthesis of why Option A beats alternatives given investigation findings]

---

### Implementation Details

**What to implement first:**
- [Highest priority change based on findings]
- [Quick wins or foundational work]
- [Dependencies that need to be addressed early]

**Things to watch out for:**
- ⚠️ [Edge cases or gotchas discovered during investigation]
- ⚠️ [Areas of uncertainty that need validation during implementation]
- ⚠️ [Performance, security, or compatibility concerns to address]

**Areas needing further investigation:**
- [Questions that arose but weren't in scope]
- [Uncertainty areas that might affect implementation]
- [Optional deep-dives that could improve the solution]

**Success criteria:**
- ✅ [How to know the implementation solved the investigated problem]
- ✅ [What to test or validate]
- ✅ [Metrics or observability to add]

---

## References

**Files Examined:**
- [File path] - [What you looked at and why]
- [File path] - [What you looked at and why]

**Commands Run:**
```bash
# [Command description]
[command]

# [Command description]
[command]
```

**External Documentation:**
- [Link or reference] - [What it is and relevance]

**Related Artifacts:**
- **Decision:** [Path to related decision document] - [How it relates]
- **Investigation:** [Path to related investigation] - [How it relates]
- **Workspace:** [Path to related workspace] - [How it relates]

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
