# Design: Dashboard View Consolidation

**Date:** 2026-02-16
**Phase:** Complete
**Status:** Active
**Type:** Architecture Assessment

---

## Design Question

Dylan has three dashboard views (`/`, `/work-graph`, `/knowledge-tree`). The work-graph doesn't render. A recent design session (orch-go-nbem) produced 6 tasks for work-graph improvements that accidentally landed on the broken page. What should we consolidate, fix, or deprecate?

## Problem Framing

**Success Criteria:**
- Clear recommendation for which views to keep/fix/deprecate
- Root cause of work-graph rendering failure identified and documented
- Clear guidance on where pending presentation work should land
- Actionable for feature-impl agents to execute

**Constraints:**
- Dylan uses Knowledge Tree Work tab daily — can't break it
- Dashboard works and is mature — not worth rewriting
- Backend work (graph API, attention store) is already page-agnostic
- Must be implementable incrementally (not big-bang migration)

**Scope:**
- IN: View assessment, rendering diagnosis, consolidation recommendation, work landing guidance
- OUT: Implementing the fix, migrating components, writing new features

---

## Exploration

### Fork 1: What Causes Work Graph Rendering Failure?

**Root cause:** `wip.fetchQueued()` method doesn't exist on the wip store.

The work-graph page's `onMount` calls `wip.fetchQueued(projectDir)` on line 253 of `+page.svelte`. The wip store (`web/src/lib/stores/wip.ts`) is a 68-line stub with only a generic `fetch()` method. Calling `undefined()` throws `TypeError` synchronously, which — in the async onMount — prevents `loading = false` from executing on line 256. The page shows "Loading work graph..." forever.

**4 missing methods across 2 stores:**

| Store | Method | Lines | Severity |
|-------|--------|-------|----------|
| wip | `fetchQueued(projectDir)` | 151, 253 | **P0** — blocks render |
| wip | `setRunningAgents(agents)` | 317 | P2 — throws on agent updates |
| focus | `clearFocus()` | 605 | P3 — button crash on click |
| focus | `setFocus(title, beadsId)` | 618 | P3 — button crash on click |

**Fix:** Add no-op stubs to unblock rendering, then implement properly:
- `wip.fetchQueued()` → no-op returning `[]` (unblocks render immediately)
- `wip.setRunningAgents()` → set store from agents list
- `focus.clearFocus()` → POST to existing `/api/focus` endpoint
- `focus.setFocus()` → POST to existing `/api/focus` endpoint

### Fork 2: Should We Fix Work Graph or Abandon It?

**Options:**
- A: Fix work-graph (add 4 stub methods, page renders)
- B: Abandon work-graph, move features to knowledge-tree
- C: Abandon work-graph, enhance dashboard's operational mode

**Substrate says:**
- Principle: "Always prefer the long-term solution" → work-graph is the more capable long-term view
- Model: Dashboard architecture's two-mode design is for agent monitoring, not work tracking
- Decision: "Beads + Focus high priority" for dashboard → work tracking belongs in a work-focused view

**Recommendation: Option A — Fix work-graph.**

The fix is trivial (4 stub methods). The page has 1043 lines of sophisticated work-tracking UI that the knowledge-tree Work tab doesn't have: dependency trees, attention badges, verification gates, keyboard navigation, label filtering, group-by modes, ready-to-complete queue, in-progress section, new issue highlights, focus auto-scoping. Rebuilding this in knowledge-tree would be orders of magnitude more work.

### Fork 3: How Many Views Should Exist Long-Term?

**Options:**
- A: Three views (Dashboard + Work Graph + Knowledge Tree)
- B: Two views (Dashboard + Knowledge Tree with enhanced Work tab)
- C: Two views (Dashboard + Work Graph, deprecate Knowledge Tree)
- D: Two views (Agent Dashboard + Work Dashboard with knowledge tab)

**Substrate says:**
- Principle: "Evolve by distinction" → the three views serve genuinely different intents
- Constraint: "Dashboard must be usable at 666px width" → can't stuff work tracking into agent monitoring
- Decision: "Dashboard gets lightweight actions; orchestrator keeps reasoning" → work graph is the reasoning layer

**Recommendation: Option D — Two dashboards, different concerns.**

| View | Purpose | Keep? | Rationale |
|------|---------|-------|-----------|
| Dashboard (`/`) | Agent monitoring, health, performance | **Keep as-is** | Mature, battle-tested, serves distinct purpose |
| Work Graph (`/work-graph`) | Work tracking, issue triage, verification | **Fix + enhance** | Most capable work view, just needs 4 stubs |
| Knowledge Tree (`/knowledge-tree`) | Knowledge browsing + Work tab + Timeline | **Keep Knowledge + Timeline tabs, deprecate Work tab** | Knowledge and Timeline have no equivalent elsewhere; Work tab is redundant once work-graph works |

Long-term, consider moving the Knowledge Tree's Knowledge and Timeline tabs INTO the work-graph as additional view modes (it already has Issues/Artifacts/Completed toggle). This would consolidate to 2 routes:
- `/` — Agent monitoring dashboard
- `/work-graph` — Work + Knowledge + Timeline dashboard

But this is a future migration, not immediate.

### Fork 4: Where Should Pending Presentation Work Land?

The orch-go-nbem epic has 6 tasks:
1. Status-first partitioning (in-progress above open/blocked)
2. Dependency-chain grouping (blocked-by rendered under blockers)
3. Verification badges per issue
4. Daemon bar with verification gate banner
5. Sort logic (status → priority → date)
6. Focus auto-scoping

**All 6 should land on `/work-graph`**, which already has infrastructure for most:
- Task 1: Already has `inProgressItems` and `readyToCompleteItems` sections (lines 349-417)
- Task 2: Already uses `buildTree()` with parent-child/blocks edges from `/api/beads/graph`
- Task 3: Already has `attentionBadge` field on TreeNode (line 67 of work-graph.ts)
- Task 4: Already has daemon status in header (line 793-810) and verification gate banner (lines 862-877)
- Task 5: Already has groupBy infrastructure; just needs status-first sort
- Task 6: Already has focus breadcrumb and auto-scoping (lines 838-858)

The work-graph page was DESIGNED for exactly this work. The knowledge-tree Work tab has NONE of this infrastructure.

---

## Synthesis

### Recommended Action Plan

**Phase 0: Unblock Work Graph (30 min)**
1. Add `fetchQueued()` stub to wip store → returns `[]`
2. Add `setRunningAgents()` to wip store → updates store from agents list
3. Add `clearFocus()` to focus store → POST `/api/focus` with empty payload
4. Add `setFocus()` to focus store → POST `/api/focus` with goal + beads_id
5. Verify page renders

**Phase 1: Land Presentation Work on Work Graph (from orch-go-nbem epic)**
- All 6 tasks target `/work-graph` page
- Backend APIs already exist and are page-agnostic
- Focus on frontend presentation changes

**Phase 2: Deprecate Knowledge Tree Work Tab (future)**
- Add banner to Knowledge Tree Work tab: "Work tracking moved to /work-graph"
- Redirect or remove Work tab after Dylan confirms comfort with work-graph
- Keep Knowledge and Timeline tabs — they serve unique purposes

**Phase 3: Consider Tab Consolidation (future, optional)**
- Move Knowledge and Timeline into work-graph as view modes
- This consolidates to 2 routes: agent dashboard + work dashboard

---

## Recommendations

⭐ **RECOMMENDED: Fix work-graph + land all pending work there**

- **Why:** Trivial fix (4 stub methods), page already has 90% of needed infrastructure, knowledge-tree Work tab has 0% of needed infrastructure. The work-graph was designed for this exact use case — it just had a rendering bug from day one that nobody diagnosed.
- **Trade-off:** Two "dashboard-like" pages exist simultaneously until Knowledge Tree Work tab is deprecated. Acceptable because they serve different purposes and there's no user confusion since work-graph hasn't been visible.
- **Expected outcome:** Dylan gets a working work-tracking dashboard within hours, not weeks.

**Alternative: Enhance Knowledge Tree Work Tab**
- **Pros:** Already works, Dylan uses it daily
- **Cons:** Requires rebuilding ~900 lines of work-graph functionality (dependency trees, attention badges, keyboard nav, grouping, filtering, ready-to-complete queue, etc.) from scratch. Knowledge Tree architecture (flat tree with SSE file watching) is fundamentally different from work-graph architecture (graph with edges + attention signals).
- **When to choose:** Only if work-graph has deeper issues beyond the 4 missing methods.

## Decision Gate Guidance (if promoting to decision)

**Add blocks: frontmatter when:**
- This decision resolves the recurring "where does work-graph work land?" confusion
- Future agents spawning for work-graph or knowledge-tree features need to know the canonical view

**Suggested blocks keywords:**
- "dashboard consolidation"
- "work-graph"
- "knowledge-tree work tab"
- "presentation work"
