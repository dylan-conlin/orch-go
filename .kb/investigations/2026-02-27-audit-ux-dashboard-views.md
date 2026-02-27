# Investigation: UX Audit — Dashboard Views (/ and /work-graph)

**TLDR:** Neither view gives the full operational picture. The main dashboard is agent-centric but empty when no agents run. The work graph is issue-centric but has no live event feed or agent visibility. The critical gap: "10 unverified completions" pauses the daemon but neither view surfaces this clearly — the dashboard shows "0 unverified" (different metric) and the work graph shows "paused" without explaining why or which completions need attention.

**Status:** Complete
**Date:** 2026-02-27
**Beads:** orch-go-8p44
**Mode:** quick (with targeted deep dives on data flow)
**Target:** http://localhost:5188/ and http://localhost:5188/work-graph
**Viewports:** 1280, 666 (half MacBook Pro)

---

## Baseline Metrics

| Metric | Dashboard (/) | Work Graph (/work-graph) |
|--------|:---:|:---:|
| axe-core violations | 2 | 3 |
| axe-core critical | 1 (select-name) | 1 (select-name) |
| axe-core serious | 1 (color-contrast, 34 nodes) | 2 (color-contrast 30 nodes, nested-interactive 12 nodes) |
| axe-core passes | 18 | 25 |
| Console errors | 0 | 0 |
| Console warnings | 0 | 4 (wip store stubs) |
| JS errors on interaction | 0 | 0 |
| SSE connection time | ~2-3s | ~2-3s |
| Initial load time | ~1s | ~10s (with graph fetch) |

---

## Critical Finding: Verification Count Blind Spot

**Severity:** Major
**Impact:** Operator cannot tell why daemon is paused or which completions need review

### The Problem

The daemon pauses with `completions_since_verification: 10` (from `/api/daemon`), but the dashboard stats bar shows "0 unverified paused" (from `/api/verification`).

### Root Cause: Two Different Metrics

| Metric | API | What It Measures | Reset Trigger |
|--------|-----|-----------------|---------------|
| `unverified_count` | `/api/verification` | Open issues with incomplete verification gates (human sign-off) | When gate1/gate2 marked complete on checkpoint |
| `completions_since_verification` | `/api/daemon` | Daemon auto-completions since last heartbeat signal | When human sends verification heartbeat |

**The dashboard only shows `unverified_count` (0), hiding the fact that 10 completions accumulated since the last heartbeat.** The operator sees "0 unverified paused" which is confusing — "if 0 are unverified, why is it paused?"

### What the Operator Actually Needs

"10 completions since your last review — daemon paused. [Review them →]"

The `completions_since_verification` count from the daemon API is the actionable number. `unverified_count` is a secondary metric about formal gate completion.

---

## Findings by View

### Main Dashboard (/) — Agent-Centric View

#### What Works Well

1. **Stats bar** — Dense operational summary at top: errors, active, ready, review, unverified, slots. Good information density.
2. **Ops/History mode split** — Ops mode is clean and focused. History mode surfaces raw lifecycle events.
3. **Coaching health indicator** — "Behavioral warnings detected" banner with timestamp.
4. **Services section** — "3 running" with collapsible detail.
5. **Up Next section** — Priority-ordered queue with P1/P2 badges, age, and beads IDs.
6. **Following mode** — Auto-scopes to orchestrator's current project context.
7. **SSE auto-reconnect** — Connection status visible in header, auto-reconnects on disconnect.

#### Problems

##### 1. Empty When No Agents Run (Major)
**Severity:** Major
**Evidence:** Screenshot `dashboard-ops-mode-1280.png` — "Active Agents 0", "No active agents", "Spawn with orch spawn"
**Impact:** With 0 active agents, the dashboard's primary content area is empty. The operator sees the triage queue and upcoming work but has zero operational context about what JUST happened. The Ops view is purely forward-looking — it doesn't show recent completions, recent events, or what the daemon just did.
**Recommendation:** Add "Recent Activity" to Ops mode — last 5-10 lifecycle events (completions, spawns, errors) from Agent Lifecycle feed.

##### 2. Verification Count Shows Wrong Number (Major)
**Severity:** Major
**Evidence:** `/api/verification` returns `unverified_count: 0` while `/api/daemon` returns `completions_since_verification: 10`. Stats bar shows "0 unverified paused" — technically correct but operationally misleading.
**Impact:** Operator can't tell why daemon is paused. "0 unverified but paused" is confusing.
**Recommendation:** Show `completions_since_verification` from daemon API in the stats bar. Something like: "10 since review **paused**" with a link to review them.

##### 3. Lifecycle Events Only in History Mode (Minor)
**Severity:** Minor
**Evidence:** Ops mode has no event feed. Agent Lifecycle panel and SSE Stream panel only appear in History mode.
**Impact:** Ops mode — the default operational view — is blind to what's happening in real-time. The operator must switch to History mode to see events, losing the clean Ops layout.
**Recommendation:** Add a compact event ticker to Ops mode (last 3-5 events, auto-scrolling, no panel needed).

##### 4. No Cross-Project Visibility (Minor)
**Severity:** Minor
**Evidence:** Dashboard follows orchestrator context → shows toolshed data. Switching projects requires tmux context change.
**Impact:** Can't see which projects have pending completions. Daemon says "200 queued" but all you see is the 26 from toolshed.
**Recommendation:** Add cross-project summary row in stats bar or a project selector.

##### 5. Select Element Missing Accessible Name (Major — a11y)
**Severity:** Major (axe-core critical)
**WCAG:** 4.1.2 Name, Role, Value
**Evidence:** axe-core: `select-name` violation on time filter `<select>` element
**Recommendation:** Add `aria-label="Time filter"` to the select element.

##### 6. Color Contrast Failures (Major — a11y)
**Severity:** Major (axe-core serious)
**WCAG:** 1.4.3 Contrast (Minimum)
**Evidence:** axe-core: `color-contrast` on 34 elements, starting with nav links using `text-muted-foreground` class
**Recommendation:** Increase contrast of `text-muted-foreground` color token (known issue, toolshed-sck already tracks this).

---

### Work Graph (/work-graph) — Issue-Centric View

#### What Works Well

1. **Dependency tree with GATE markers** — Excellent visualization of work flow. "Anonymize PII → GATE → Fix AI Analysis panel" clearly shows what blocks what.
2. **Priority badges** — Color-coded P1 (red), P2 (blue), P3 (gray) with issue IDs.
3. **Keyboard navigation** — j/k navigate, h/l collapse/expand, enter details, v verify, x close. Power-user friendly.
4. **Daemon status in header** — "Daemon: paused · 0/3 slots · last poll 5 mins ago · 200 queued" — good at-a-glance.
5. **Issue count and edge count** — "28 issues · 9 edges · toolshed" — good context.
6. **Responsive at 666px** — Content flows, titles truncate with ellipsis. Fully usable at half-screen.
7. **SSE-triggered refresh** — Watches lifecycle events and auto-refreshes graph on spawn/complete/abandon.
8. **"Show all (16 more)" button** — Progressive disclosure for independent issues.

#### Problems

##### 1. No Live Event Feed (Major)
**Severity:** Major
**Evidence:** Work graph has no event panel. You can see the state of issues but not what JUST happened.
**Impact:** Operator can't tell if an agent just completed, just spawned, or just errored. The tree shows current state but not recent transitions. This is the #1 requirement failure per the audit criteria.
**Recommendation:** Add a compact event strip below the daemon status line — last 5 events with timestamps. Same data source as Agent Lifecycle on the dashboard.

##### 2. No Agent Status on Issues (Major)
**Severity:** Major
**Evidence:** Tree nodes show `○` (empty circle) for issue status but no indication of whether an agent is actively working on it.
**Impact:** Operator can't tell which issues have running agents. When agents ARE running, the work graph gives no visibility.
**Recommendation:** Add agent indicator to tree nodes: `●` (working), `✓` (Phase: Complete), spinner, etc. Data available via `/api/agents` cross-referenced with beads IDs.

##### 3. No Completions Needing Review Section (Major)
**Severity:** Major
**Evidence:** The "Ready to Complete" section appears only when agents are at Phase: Complete with open issues. But after `orch complete` or auto-completion, they disappear. The 10 completions the daemon is paused for are invisible.
**Impact:** This is the specific scenario Dylan described — "daemon pauses with 10 unverified completions and I can't tell which projects they're in."
**Recommendation:** Add a "Recent Completions" or "Needs Verification" section that pulls from `/api/agentlog` or a new endpoint that lists the last N completions with their beads ID, project, and verification status.

##### 4. Single-Project Scope (Minor)
**Severity:** Minor
**Evidence:** Shows "toolshed" in header. Switching requires tmux context change.
**Impact:** Can't see orch-go's 13 issues alongside toolshed's 28. Cross-project prioritization impossible.
**Recommendation:** Add an "All Projects" option or a project selector dropdown.

##### 5. Slow Initial Load (Minor)
**Severity:** Minor
**Evidence:** ~10 seconds showing "Loading work graph..." before content renders.
**Impact:** First impression is slow. Repeated navigations to this page feel sluggish.
**Recommendation:** Show cached/stale data immediately, refresh in background.

##### 6. Nested Interactive Controls (Major — a11y)
**Severity:** Major (axe-core serious)
**WCAG:** 4.1.2
**Evidence:** axe-core: `nested-interactive` on 12 issue row elements — `role="button"` wrapping inner `<button>` elements (beads ID links).
**Recommendation:** Restructure issue rows so the outer clickable area doesn't use `role="button"` or move the inner buttons outside the clickable parent.

##### 7. Select Element Missing Accessible Name (Major — a11y)
**Severity:** Major (axe-core critical)
**Evidence:** axe-core: Group-by dropdown `<select>` missing accessible name.
**Recommendation:** Add `aria-label="Group by"` to the select element.

##### 8. WIP Store Warnings (Cosmetic)
**Severity:** Cosmetic
**Evidence:** 4 console warnings per page load: "wip store: fetchQueued not implemented", "wip store: setRunningAgents not implemented"
**Impact:** No functional impact but creates noise in dev console.
**Recommendation:** Remove console.warn from stub methods or implement actual functionality.

---

## Gap Analysis: What an Operator Needs at a Glance

| Operational Need | Dashboard (/) | Work Graph | Gap? |
|-----------------|:---:|:---:|:---:|
| **Active agents & their phase** | ✅ (when agents exist) | ❌ No visibility | **CRITICAL** — work graph blind to agents |
| **Completions needing review** | ❌ Shows wrong metric (0 vs 10) | ❌ Not shown at all | **CRITICAL** — neither surfaces daemon pause reason |
| **Live lifecycle events** | ⚠️ History mode only | ❌ No event feed | **MAJOR** — work graph has no event feed |
| **Cross-project visibility** | ❌ Single project | ❌ Single project | **MAJOR** — can't see totals across projects |
| **Triage/ready queue** | ✅ Ready Queue section | ✅ Issue tree (filtered to open) | Both have this |
| **Dependencies** | ❌ | ✅ Tree with GATE markers | Dashboard has no dep visibility |
| **Daemon health** | ✅ Stats bar | ✅ Header bar | Both have this |
| **Services health** | ✅ Services section | ❌ | Work graph missing |
| **Questions/blockers** | ✅ Questions section | ❌ | Work graph missing |
| **Coaching health** | ✅ Banner | ❌ | Work graph missing |
| **Issue actions** | ❌ | ✅ Close/update/verify | Dashboard can't act on issues |

---

## Recommendation: Consolidate to Work Graph as Primary Operational View

**Rationale:** The work graph is the better foundation because it shows work (persistent) rather than agents (transient). When 0 agents are running, the dashboard is empty but the work graph still shows everything you need to prioritize and act.

### Consolidation Plan

#### Phase 1: Make Work Graph Operationally Complete (3-5 issues)

1. **Add compact event strip** — Below daemon status, show last 5 lifecycle events with timestamps. Use agentlog SSE data already connected.
2. **Add agent badges to tree nodes** — Show which issues have running agents. Cross-reference `/api/agents` beads_id with tree node IDs.
3. **Add "Needs Verification" section** — Show `completions_since_verification` from daemon API with links to the completed issues. This is the #1 pain point.
4. **Fix verification display on stats bar** — Show the daemon's completion count, not just the gate-based unverified count.

#### Phase 2: Absorb Dashboard Unique Features (2-3 issues)

5. **Add collapsed panels for services/questions/coaching** — These sections from the dashboard's Ops mode should be collapsible panels at the top of the work graph.
6. **Add cross-project totals** — "200 queued" is in daemon status; expand to show per-project breakdown.

#### Phase 3: Remove Main Dashboard (1 issue)

7. **Replace `/` route** — Redirect to `/work-graph` or make work graph the default route. Keep History mode as a tab/mode on the work graph for the full event archive.

### What Gets Cut

- **Agent card grid** — Replaced by agent badges on tree nodes + a compact "Active Agents" bar
- **Swarm Map** — Archive/historical view of past agents; keep as History mode
- **Two-mode Ops/History toggle** — Replace with collapsible panels
- **Stats bar** — Consolidate into work graph header (daemon status already there, just needs the other metrics)

### What Stays Separate

- **Knowledge Tree** (`/knowledge-tree`) — Different user intent ("what do I know?" vs "what work needs doing?"). No overlap.

---

## Screenshot Index

| Filename | Viewport | State | Description |
|----------|----------|-------|-------------|
| dashboard-ops-1280.png | 1280px | History mode | Full dashboard with stats bar, agent lifecycle events, SSE stream |
| dashboard-ops-mode-1280.png | 1280px | Ops mode | Clean operational view: Up Next, Questions, Active (0), Needs Attention, Ready Queue |
| dashboard-ops-666.png | 666px | Ops mode | Half-screen: nav wraps, stats bar flows, content readable |
| work-graph-1280.png | 1280px | Default | Full issue tree with dependency chains, GATE markers, priority badges |
| work-graph-666.png | 666px | Default | Half-screen: titles truncate, layout clean, keyboard hints visible |

---

## Reproducibility

**Auth:** None (local dev)
**Services:** orch-dashboard start (opencode:4096, orch:3348, web:5188)
**Commands:** Playwright MCP: browser_navigate, browser_resize, browser_snapshot, browser_evaluate(axe-core)
**Re-audit schedule:** After consolidation implementation
