# Session Synthesis

**Agent:** og-work-audit-orch-go-27feb-950e
**Issue:** orch-go-8p44
**Duration:** 2026-02-27 ~10:00 → ~10:45
**Outcome:** success

---

## Plain-Language Summary

Audited both dashboard views (main dashboard at `/` and work graph at `/work-graph`) to answer: which gives the operator the full picture, and should they be consolidated?

**The answer: neither gives the full picture today, but the work graph is the right foundation.** The main dashboard is agent-centric — when 0 agents are running (the current state), it's essentially empty. The work graph shows issues and dependencies regardless of agent activity, making it useful even when idle. However, the work graph lacks three critical things: a live event feed (what just happened?), agent status overlaid on issues (who's working on what?), and a "completions needing review" section (why is the daemon paused?).

The biggest discovery: the daemon is paused because 10 completions happened since Dylan's last verification heartbeat, but the dashboard shows "0 unverified" because it reads a different metric (formal gate completion count vs daemon heartbeat counter). This is the exact pain point Dylan described — and it's a data source mismatch, not a missing feature.

**Recommendation:** Consolidate to work graph as the primary view. Add event feed, agent badges, and verification queue to it. Phase out the main dashboard.

---

## TLDR

Audited both dashboard views. Neither shows the complete operational picture. The work graph is the better foundation (issue-centric, works even with 0 agents) but needs three additions: live event feed, agent badges on issues, and a "completions needing review" section. The "0 unverified paused" confusion is a data source mismatch — the stats bar reads `/api/verification` (0 issues with incomplete gates) instead of `/api/daemon` (10 completions since last heartbeat).

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-02-27-audit-ux-dashboard-views.md` — Full UX audit investigation with findings, gap analysis, and consolidation recommendation
- `.kb/investigations/screenshots/2026-02-27-audit-dashboard-views/` — 5 screenshots at 1280px and 666px across both views
- `.orch/workspace/og-work-audit-orch-go-27feb-950e/SYNTHESIS.md` — This file
- `.orch/workspace/og-work-audit-orch-go-27feb-950e/VERIFICATION_SPEC.yaml` — Verification contract

### Files Modified
- None (audit-only session)

### Commits
- (pending)

---

## Evidence (What Was Observed)

### Verification Count Mismatch (Critical Finding)
- `/api/daemon` returns `completions_since_verification: 10` and `is_paused: true`
- `/api/verification` returns `unverified_count: 0` and `daemon_paused: true`
- Stats bar template in `stats-bar.svelte` renders `$verification.unverified_count` → shows "0 unverified paused"
- These are different metrics: `unverified_count` = open issues with incomplete human gates; `completions_since_verification` = daemon auto-completions since last heartbeat signal
- Operator sees "0 unverified but paused" — confusing and non-actionable

### Main Dashboard State (0 agents)
- Ops mode: Services (3 running), Up Next (5 items, 2 urgent), Questions (4), Active Agents (0), Needs Attention (1 — blocked issues), Ready Queue (26)
- History mode: adds Agent Lifecycle events panel (showing daemon.complete, session.auto_completed events) and SSE Stream panel
- Stats bar: 0 errors, 0 active, 26 ready (2 blocked), 0 review, 0 unverified paused, 0/3 slots

### Work Graph State
- 28 issues, 9 dependency edges, toolshed project
- Daemon status: "paused · 0/3 slots · last poll X mins ago · 200 queued"
- 2 dependency chains with GATE markers visible
- 24 independent issues in collapsible group
- No event feed, no agent indicators, no completions section

### Accessibility (axe-core WCAG AA)
- Dashboard: 2 violations (1 critical: unlabeled select, 1 serious: 34 contrast failures)
- Work Graph: 3 violations (1 critical: unlabeled select, 1 serious: 30 contrast failures, 1 serious: 12 nested interactive controls)

### Responsive (666px — half MacBook Pro)
- Dashboard: Nav links wrap but readable, stats bar flows, content sections stack well
- Work Graph: Issue titles truncate with ellipsis, keyboard hints visible, layout clean

---

## Architectural Choices

### Consolidate to Work Graph as Primary View
- **What I chose:** Recommend work graph as the primary operational view
- **What I rejected:** Keeping both views; merging into dashboard; building a third view
- **Why:** Work graph shows work (persistent) not agents (transient). With 0 agents, the dashboard is empty. The work graph still shows every issue, dependency, and priority. The agent-centric view is useful but secondary to the issue-centric view.
- **Risk accepted:** Agent monitoring becomes less prominent; operators used to the dashboard layout will need to adapt

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-02-27-audit-ux-dashboard-views.md` — Full audit with 13 findings, gap analysis, consolidation plan

### Constraints Discovered
- Dashboard `/api/verification` and `/api/daemon` measure different things with the same word "verification" — creates confusion
- Work graph initial load takes ~10s (graph fetch + SSE connection)
- axe-core `nested-interactive` on work graph issue rows — structural a11y issue from `role="button"` wrapping `<button>` children

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up (multiple issues)

### Recommended Issues (by priority)

1. **IMMEDIATE: Fix verification display** — Show `completions_since_verification` from daemon API on stats bar and work graph header. This is the #1 pain point.
   - Skill: feature-impl
   - Files: `web/src/lib/components/stats-bar/stats-bar.svelte`, `web/src/routes/work-graph/+page.svelte`, `web/src/lib/stores/daemon.ts`

2. **Add live event strip to work graph** — Compact 5-event ticker below daemon status line using agentlog SSE data already connected.
   - Skill: feature-impl
   - Files: `web/src/routes/work-graph/+page.svelte`

3. **Add agent badges to work graph tree nodes** — Show which issues have running agents by cross-referencing `/api/agents` beads_id.
   - Skill: feature-impl
   - Files: `web/src/lib/components/work-graph-tree/work-graph-tree.svelte`, `web/src/lib/stores/work-graph.ts`

4. **Fix a11y violations** — Unlabeled selects (add `aria-label`), nested interactive controls (restructure issue rows).
   - Skill: feature-impl
   - Files: `web/src/lib/components/stats-bar/stats-bar.svelte`, `web/src/routes/work-graph/+page.svelte`, `web/src/lib/components/work-graph-tree/work-graph-tree.svelte`

---

## Unexplored Questions

- **What does `orch review` show?** — The daemon's 10 completions should be reviewable via CLI. Is the dashboard just duplicating what CLI already does, or should it be the primary review interface?
- **Should the daemon heartbeat be sendable from the dashboard?** — Currently requires CLI (`orch verify`?). A "I've reviewed these" button on the dashboard could reset the counter.
- **Knowledge tree relationship** — The audit scope excluded `/knowledge-tree` but it has a Work tab that overlaps with work graph. Should it be pruned?

---

## Session Metadata

**Skill:** ux-audit
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-work-audit-orch-go-27feb-950e/`
**Investigation:** `.kb/investigations/2026-02-27-audit-ux-dashboard-views.md`
**Beads:** `bd show orch-go-8p44`
