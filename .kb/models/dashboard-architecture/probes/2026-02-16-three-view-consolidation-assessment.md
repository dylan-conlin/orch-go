# Probe: Three Dashboard View Consolidation Assessment

**Model:** Dashboard Architecture
**Date:** 2026-02-16
**Status:** Complete

---

## Question

The dashboard architecture model describes a single "Swarm Dashboard" at `/`. But the codebase now has THREE dashboard views: Dashboard (`/`), Work Graph (`/work-graph`), and Knowledge Tree (`/knowledge-tree`). The model is silent on this multi-view evolution. Which views serve which purpose, and should they be consolidated?

---

## What I Tested

### View 1: Dashboard (/ route)
- **File:** `web/src/routes/+page.svelte` (~1043 lines in template)
- **Purpose:** Agent monitoring — real-time swarm status, health, coaching, performance
- **API endpoints:** `/api/agents`, `/api/beads`, `/api/usage`, `/api/focus`, `/api/daemon`, `/api/servers`, `/api/orchestrator-sessions`, `/api/services`, `/api/questions`, `/api/coaching`, `/api/hotspots`, `/api/events` (SSE), `/api/agentlog` (SSE)
- **Unique features:** Two-mode design (Operational/Historical), agent health coaching, context quality warnings, hotspot detection, orchestrator following, stable sort, progressive disclosure, approval workflow
- **State:** Working, mature, battle-tested through 62+ investigations

### View 2: Work Graph (/work-graph route)
- **File:** `web/src/routes/work-graph/+page.svelte` (1043 lines)
- **Purpose:** Issue/work tracking — beads issue tree with dependencies, attention signals, verification gates
- **API endpoints:** `/api/beads/graph`, `/api/agents`, `/api/attention`, `/api/daemon`, `/api/focus`, `/api/context`, `/api/events` (SSE), `/api/agentlog` (SSE)
- **Unique features:** Dependency-aware tree (nodes + edges from `/api/beads/graph`), attention badges, ready-to-complete section, in-progress section, focus auto-scoping, verification gate banner, label filtering, group-by (priority/area/effort), new issue highlights, keyboard navigation (j/k/h/l/enter), issue close/update actions
- **State:** BROKEN — never renders for Dylan

### View 3: Knowledge Tree (/knowledge-tree route)
- **File:** `web/src/routes/knowledge-tree/+page.svelte` (371 lines)
- **Purpose:** Knowledge organization (3 tabs: Knowledge, Work, Timeline)
- **API endpoints:** `/api/tree?view=knowledge`, `/api/tree?view=work`, `/api/timeline`, `/api/events/tree` (SSE), `/api/events/timeline` (SSE)
- **Unique features:** SSE-driven live updates when .kb/ or .beads/ files change, recursive tree rendering, search, node type filtering, animations (pulsing for in_progress, split-and-grow for completed)
- **State:** Working, used daily by Dylan (Work tab)

### Work Graph Root Cause Analysis

**Why doesn't /work-graph render?**

Traced the rendering failure to `onMount` in `+page.svelte`:

```
Line 246-254:
    await Promise.all([
        workGraph.fetch(projectDir, 'open', focusBeadsId),
        agents.fetch(),
        attention.fetch(projectDir)
    ]);

    wip.fetchQueued(projectDir).catch(console.error);  ← CRASH HERE
    daemon.fetch().catch(console.error);

    loading = false;  ← NEVER REACHED
```

`wip.fetchQueued()` does NOT exist on the wip store (only has `fetch()`). Calling `undefined()` throws `TypeError: wip.fetchQueued is not a function` synchronously, before `.catch()` can intercept it. In the async `onMount`, this prevents `loading = false` from executing.

**Result:** Page shows "Loading work graph..." permanently.

**Additional broken methods (4 total missing across 2 stores):**

| Store | Missing Method | Called At | Impact |
|-------|----------------|-----------|--------|
| `wip` | `fetchQueued(projectDir)` | Lines 151, 253 | **Blocks page render** (crashes onMount before loading=false) |
| `wip` | `setRunningAgents(agents)` | Line 317 | Reactive block throws on every agent update |
| `focus` | `clearFocus()` | Line 605 | "Clear Focus" button crashes on click |
| `focus` | `setFocus(title, beadsId)` | Line 618 | "Set Focus" button crashes on click |

The wip store (`web/src/lib/stores/wip.ts`) is a **complete stub** — 68 lines of TODO comments and hardcoded return values. It was never implemented.

**Fix difficulty:** Low. Add the 3 missing methods to 2 stores:
1. `wip.fetchQueued()` — can start as no-op returning empty array (like existing `wip.fetch()`)
2. `wip.setRunningAgents()` — set store value from agents list
3. `focus.clearFocus()` — POST to `/api/focus` with empty/clear payload
4. `focus.setFocus()` — POST to `/api/focus` with goal + beads_id

---

## What I Observed

### Feature Comparison Matrix

| Feature | Dashboard (/) | Work Graph | Knowledge Tree |
|---------|:---:|:---:|:---:|
| Agent monitoring | ✅ Rich | ❌ None | ❌ None |
| Issue tree with deps | ❌ None | ✅ Full graph | ⚠️ Flat list |
| Ready-to-complete queue | ⚠️ Needs Review section | ✅ Dedicated section | ❌ None |
| In-progress section | ⚠️ Active agents | ✅ Dedicated section | ❌ None |
| Attention badges | ❌ None | ✅ Full integration | ❌ None |
| Verification gate banner | ❌ None | ✅ Banner | ❌ None |
| Focus scoping | ⚠️ Stars aligned items | ✅ Auto-filter tree | ❌ None |
| Dependency grouping | ❌ None | ✅ Parent-child edges | ❌ None |
| Label filtering | ❌ None | ✅ Full | ⚠️ Node type only |
| Group by priority/area | ❌ None | ✅ Dropdown | ❌ None |
| Knowledge artifacts | ❌ None | ✅ Artifacts view | ✅ Primary purpose |
| Timeline | ❌ None | ❌ None | ✅ Session timeline |
| SSE live updates | ✅ Agent events | ✅ Agent events | ✅ File change events |
| Keyboard navigation | ❌ None | ✅ Full vim-style | ✅ Search shortcut |
| Issue close/update | ❌ None | ✅ Via side panel | ❌ None |
| Two-mode design | ✅ Operational/Historical | ❌ N/A | ❌ N/A |
| Health coaching | ✅ Full | ❌ None | ❌ None |
| Daemon status | ✅ Stats bar | ✅ Header bar | ❌ None |
| New issue detection | ❌ None | ✅ 30s highlight | ❌ None |

### Key Insight: Three Complementary Purposes

1. **Dashboard (/):** "How are my agents doing?" — Agent health, performance, swarm oversight
2. **Work Graph:** "What work needs attention?" — Issue triage, dependency tracking, verification
3. **Knowledge Tree:** "What do I know?" — Knowledge organization, artifact browsing, timeline

These are NOT redundant views competing for the same niche. They serve three distinct user intents.

### The knowledge-tree Work tab is a thin overlay on the Knowledge view

The Knowledge Tree's Work tab uses `/api/tree?view=work` which returns a flat list of beads issues with linked artifacts. It has none of the work-graph's sophistication:
- No dependency edges
- No attention signals
- No status-first sorting
- No in-progress/ready-to-complete sections
- No verification badges
- Sorts alphabetically by beads ID only (`stableSort` in +page.svelte)

It exists because the Knowledge Tree was built first and needed some work visibility. Now that work-graph exists (once fixed), the Work tab is redundant.

---

## Model Impact

### Extends: Model's "Single Dashboard" Assumption

The model assumes one dashboard at `/`. Reality is three views evolved independently:
- `/` (Dec 2025) — agent monitoring, heavily iterated
- `/knowledge-tree` (Feb 2026) — knowledge browsing, Working
- `/work-graph` (Feb 2026) — work tracking, broken since creation

The model should be updated to reflect the multi-view architecture and their distinct purposes.

### Extends: Missing Failure Mode

**New Failure Mode: Stub Store Crashes Page**

When a page calls methods on a stub store (e.g., `wip.fetchQueued()`), the TypeError thrown by `undefined()` propagates through the async onMount and prevents initialization from completing. The page appears stuck in loading state permanently.

This is distinct from the model's existing failure modes (connection pool, slow load, information overload, plugin cascade) because it's a pure frontend compilation/runtime issue, not a backend/infrastructure problem.

### Confirms: Model Invariant 6

"Early filtering reduces payload size" — The work-graph's backend (`/api/beads/graph`) applies early filtering with `scope` and `parent` parameters. The attention store collects and filters signals server-side. This confirms the model's invariant that filtering should happen early.

### Contradicts: Model's "Dashboard is Dylan's ONLY observability layer"

The model states the Dashboard is Dylan's **only** observability layer. In practice, Dylan uses the Knowledge Tree Work tab daily for issue tracking. The dashboard is his agent monitoring layer, but the Knowledge Tree is his work tracking layer. When work-graph is fixed, it should become the primary work tracking layer, displacing the Knowledge Tree's Work tab.
