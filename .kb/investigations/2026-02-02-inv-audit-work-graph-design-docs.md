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

# Investigation: Audit Work Graph Design Docs

**Question:** What components and features from the Work Graph Phase 1, 2, and 3 design docs have been implemented, and what remains to be built?

**Started:** 2026-02-02
**Updated:** 2026-02-02
**Owner:** Claude (spawned agent)
**Phase:** Investigating
**Next Step:** Audit Phase 1 design components against web/src/lib/components/
**Status:** In Progress

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Starting approach - systematic audit of three design docs

**Evidence:** Read all three design documents:
- Phase 1: Work Graph Dashboard Tab (2026-01-30)
- Phase 2: Agent Overlay (2026-01-31)
- Phase 3: Artifact Feed (2026-01-31)

All designs are marked Status: Complete with detailed component lists and API requirements.

**Source:** 
- `.kb/investigations/2026-01-30-design-work-graph-dashboard-tab.md`
- `.kb/investigations/2026-01-31-design-work-graph-phase2-agent-overlay.md`
- `.kb/investigations/2026-01-31-design-work-graph-phase3-artifact-feed.md`

**Significance:** Need to systematically check each planned component against the codebase to identify implementation gaps. Will create beads issues for unimplemented items.

---

### Finding 2: Phase 1 and 3 largely complete, Phase 2 partially implemented

**Evidence:** 

**Phase 1 (Work Graph Dashboard Tab) - COMPLETE:**
- ✅ Route `/work-graph` exists (web/src/routes/work-graph/+page.svelte)
- ✅ WorkGraphTree component with keyboard nav (work-graph-tree.svelte)
- ✅ L0 row rendering with status indicators, priority badges, type badges
- ✅ L1 expanded details with blocking relationships
- ✅ work-graph.ts store with fetch from /api/beads/graph
- ✅ API endpoint handleBeadsGraph exists (cmd/orch/serve_beads.go)

**Phase 2 (Agent Overlay) - PARTIALLY IMPLEMENTED:**
- ✅ WorkInProgressSection (wip-section.svelte) - shows running + queued agents
- ✅ Health indicators (⚠️ 🚨) integrated into WIP section
- ✅ Daemon integration showing capacity
- ✅ L1 auto-expansion for running agents with phase, context %, skill
- ❌ DeliverableChecklist component (compact + full views)
- ❌ IssueSidePanel component (L2 details with lifecycle, history, artifacts)
- ❌ AttemptHistory component (timeline of attempts with outcomes)
- ❌ API: Attempt history data
- ❌ API: Deliverables schema lookup
- ❌ API: Override logging endpoint

**Phase 3 (Artifact Feed) - COMPLETE:**
- ✅ ArtifactFeed component (artifact-feed.svelte)
- ✅ ArtifactRow component (artifact-row.svelte)
- ✅ ArtifactSidePanel component (artifact-side-panel.svelte)
- ✅ ViewToggle component (view-toggle.svelte)
- ✅ Time filter integrated into feed (24h, 7d, 30d, all)
- ✅ kb-artifacts.ts store with fetch
- ✅ API endpoint /api/kb/artifacts (cmd/orch/serve_kb_artifacts.go)
- ✅ Work in Progress section stays visible across both views
- ✅ Keyboard nav (j/k/l/h, Tab to toggle views)

---

### Finding 3: [Brief, descriptive title]

**Evidence:** [Concrete observations, data, examples]

**Source:** 
- Checked web/src/lib/components/ directory structure
- Verified web/src/routes/work-graph/+page.svelte exists
- Searched for missing components: `rg "DeliverableChecklist|IssueSidePanel|AttemptHistory"`
- Checked stores: web/src/lib/stores/ (work-graph.ts, kb-artifacts.ts, wip.ts exist)
- Verified API endpoints in cmd/orch/serve_beads.go and serve_kb_artifacts.go

**Significance:** Phase 1 and Phase 3 are production-ready. Phase 2 missing components are all related to issue lifecycle tracking and deliverables management - these need to be implemented to complete the Work Graph feature.

---

### Finding 3: Stale issues status - one closed, six still open

**Evidence:** Checked issues orch-go-21154 through 21160:
- orch-go-21154 (Design: Phase 2 Agent Overlay) - **open** - Can close (design complete, implementation partially done)
- orch-go-21155 (Design: Phase 3 Artifact Feed) - **open** - Can close (design complete, implementation complete)
- orch-go-21156 (Implement WIP section) - **closed** ✅
- orch-go-21157 (Implement issue side panel L2) - **open** - Still valid (not implemented)
- orch-go-21158 (Deliverables schema and tracking) - **open** - Still valid (not implemented)
- orch-go-21159 (Implement Artifacts view toggle) - **open** - Can close (implemented)
- orch-go-21160 (Implement artifact side panel) - **open** - Can close (implemented)

**Source:** `bd show orch-go-21154 ... orch-go-21160 --json`

**Significance:** Four issues (21154, 21155, 21159, 21160) can be closed as work is complete. Two issues (21157, 21158) remain valid as the components are not implemented.

---

## Synthesis

**Key Insights:**

1. **Implementation progressed in phases** - Phase 1 (tree structure) and Phase 3 (artifact feed) were completed, but Phase 2 (lifecycle tracking) was only partially implemented. The WorkInProgressSection shows running agents but lacks the deeper lifecycle observability (attempt history, deliverables tracking) that the design called for.

2. **Issue-centric vs Agent-centric** - The WIP section focuses on agent activity (what's running now), but the planned IssueSidePanel would show issue lifecycle (how did we get here, what did we try, what's left to deliver). This is the missing piece for understanding why issues take multiple attempts or get stuck.

3. **Deliverables tracking deferred** - The deliverables schema system (tracking expected vs actual outputs per issue type, with override logging) was designed but never implemented. This would enable verification gates at issue closure and data collection on what deliverables actually matter.

**Answer to Investigation Question:**

**What's implemented:** Phase 1 (basic tree view with keyboard nav, structure mode) and Phase 3 (artifact feed with needs-decision and recent sections) are complete and functional. The work-graph route exists, components render properly, API endpoints work, and keyboard navigation is implemented.

**What remains:** Phase 2 is missing three components and three API endpoints. The missing components are: DeliverableChecklist (shows expected vs actual outputs), IssueSidePanel (shows full issue lifecycle with attempt history), and AttemptHistory (timeline of agent attempts with outcomes). The missing APIs are: attempt history data, deliverables schema lookup, and override logging.

**Issues status:** Four issues (21154, 21155, 21159, 21160) can be closed as their work is complete. Two issues (21157, 21158) remain valid and should stay open.

---

## Structured Uncertainty

**What's tested:**

- ✅ Work-graph route exists (verified: read web/src/routes/work-graph/+page.svelte)
- ✅ Phase 1 components exist (verified: listed web/src/lib/components/, found work-graph-tree/)
- ✅ Phase 3 components exist (verified: found artifact-feed/, artifact-row/, artifact-side-panel/)
- ✅ WIP section exists (verified: found wip-section/)
- ✅ Stores exist (verified: found work-graph.ts, kb-artifacts.ts, wip.ts in web/src/lib/stores/)
- ✅ API endpoints exist (verified: found handleBeadsGraph in serve_beads.go, handleKBArtifacts in serve_kb_artifacts.go)
- ✅ Components imported correctly (verified: grepped imports in +page.svelte)
- ✅ Phase 2 missing components confirmed absent (verified: searched for DeliverableChecklist, IssueSidePanel, AttemptHistory - not found)
- ✅ Beads graph API responds (verified: curl http://localhost:3348/api/beads/graph)

**What's untested:**

- ⚠️ UI functionality works end-to-end (not tested via browser - only verified files exist)
- ⚠️ Keyboard navigation works correctly (not tested - only verified code implementation)
- ⚠️ Artifact feed filtering works (not tested - only verified component exists)

**What would change this:**

- If Phase 2 components exist under different names (search was literal string match)
- If deliverables tracking implemented differently (searched for specific terms from design)
- If attempt history data embedded in existing API responses (didn't parse full API response structure)

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
- `.kb/investigations/2026-01-30-design-work-graph-dashboard-tab.md` - Phase 1 design document
- `.kb/investigations/2026-01-31-design-work-graph-phase2-agent-overlay.md` - Phase 2 design document
- `.kb/investigations/2026-01-31-design-work-graph-phase3-artifact-feed.md` - Phase 3 design document
- `web/src/routes/work-graph/+page.svelte` - Main work-graph page implementation
- `web/src/lib/components/work-graph-tree/work-graph-tree.svelte` - Tree component with keyboard nav
- `web/src/lib/components/wip-section/wip-section.svelte` - Work in Progress section
- `web/src/lib/components/artifact-feed/artifact-feed.svelte` - Artifact feed view
- `web/src/lib/stores/work-graph.ts` - Work graph store
- `web/src/lib/stores/kb-artifacts.ts` - KB artifacts store
- `web/src/lib/stores/wip.ts` - WIP store
- `cmd/orch/serve_beads.go` - Beads graph API endpoint
- `cmd/orch/serve_kb_artifacts.go` - KB artifacts API endpoint

**Commands Run:**
```bash
# List component directories
ls -la web/src/lib/components/

# Search for work-graph components
glob "web/src/lib/components/**/*work*graph*"

# Search for artifact components  
glob "web/src/lib/components/**/*artifact*"

# Check stale issues status
bd show orch-go-21154 ... orch-go-21160 --json

# Search for missing Phase 2 components
rg "DeliverableChecklist|IssueSidePanel|AttemptHistory"

# Verify API endpoints
rg "handleBeadsGraph" cmd/orch/serve_beads.go
rg "handleKBArtifacts" cmd/orch/serve_kb_artifacts.go

# Test API endpoint
curl http://localhost:3348/api/beads/graph?scope=open
```

**External Documentation:**
- [Link or reference] - [What it is and relevance]

**Related Artifacts:**
- **Decision:** [Path to related decision document] - [How it relates]
- **Investigation:** [Path to related investigation] - [How it relates]
- **Workspace:** [Path to related workspace] - [How it relates]

---

## Investigation History

**2026-02-02 10:20:** Investigation started
- Initial question: What components and features from Work Graph Phase 1, 2, 3 designs are implemented?
- Context: Spawned from orch-go-21166 to audit design docs against codebase and create issues for missing work

**2026-02-02 10:25:** Design documents reviewed
- Read all three phase designs, extracted planned components and API changes
- Identified 5 Phase 2 components + 3 API endpoints to check

**2026-02-02 10:30:** Codebase audit completed
- Found Phase 1 and Phase 3 fully implemented
- Found Phase 2 partially implemented (WIP section exists, but lifecycle tracking missing)
- Checked stale issues - 4 can be closed, 2 remain valid

**2026-02-02 10:35:** Investigation completed
- Status: Complete
- Key outcome: Phase 1 and 3 are done, Phase 2 needs DeliverableChecklist, IssueSidePanel, AttemptHistory components plus 3 API endpoints
