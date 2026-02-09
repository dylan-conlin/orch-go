# Dashboard

**Purpose:** Single authoritative reference for the Swarm Dashboard web UI. Read this before debugging dashboard issues or implementing new features.

**Last verified:** 2026-01-29

---

## Overview

The Swarm Dashboard is a web-based monitoring UI for the orchestration system, served via `orch serve`. It provides real-time visibility into agent status, daemon health, and operational metrics. The dashboard evolved significantly from Dec 21, 2025 to Jan 29, 2026, addressing performance, UX, and architectural issues through 80+ investigations.

**Major evolution:** From Dec 2025 (status-oriented) → Jan 2026 (action-oriented Strategic Center design with Decision Center, screenshot artifacts, and multi-source agent visibility including tmux escape hatch agents).

**Access:** http://localhost:5188 (default) or http://localhost:3348/api/... for API endpoints

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Browser (Dashboard)                       │
│                                                                  │
│   ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐ │
│   │ Stats Bar   │  │ Swarm Map   │  │ Event Panels (collapsed)│ │
│   │ (metrics)   │  │ (agents)    │  │ (SSE + Agentlog)        │ │
│   └──────┬──────┘  └──────┬──────┘  └───────────┬─────────────┘ │
│          │                │                      │               │
│          ▼                ▼                      ▼               │
│   ┌──────────────────────────────────────────────────────────┐  │
│   │                    Svelte Stores                          │  │
│   │  agents.ts | beads.ts | usage.ts | daemon.ts | focus.ts  │  │
│   └──────────────────────────┬───────────────────────────────┘  │
└──────────────────────────────┼───────────────────────────────────┘
                               │ HTTP + SSE
                               ▼
┌──────────────────────────────────────────────────────────────────┐
│                      orch serve (Go backend)                      │
│                                                                   │
│   ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│   │ /api/agents │  │ /api/beads  │  │ /api/events (SSE proxy) │  │
│   │ /api/usage  │  │ /api/daemon │  │ /api/agentlog (SSE)     │  │
│   │ /api/focus  │  │ /api/servers│  │                         │  │
│   └──────┬──────┘  └──────┬──────┘  └───────────┬─────────────┘  │
│          │                │                      │                │
│          ▼                ▼                      ▼                │
│   ┌──────────────────────────────────────────────────────────┐   │
│   │                    Data Sources                           │   │
│   │  OpenCode API (sessions) | bd CLI | ~/.orch/* files      │   │
│   └──────────────────────────────────────────────────────────┘   │
└───────────────────────────────────────────────────────────────────┘
```

---

## How It Works

### Agent Status Pipeline

**What:** Determines agent status (active/idle/completed) from multiple sources

**Key insight:** Status is determined by a **Priority Cascade Model** - beads/Phase signals always override session activity signals.

| Priority | Source | Status Result |
|----------|--------|---------------|
| 1 (highest) | Beads issue closed | completed |
| 2 | Phase: Complete in beads comments | completed |
| 3 | SYNTHESIS.md exists in workspace | completed |
| 4 (lowest) | Session activity (10min threshold) | active/idle |

**Key files:**
- `cmd/orch/serve_agents.go` - Main status logic (~1400 lines)
- `pkg/verify/check.go` - Phase parsing from comments

### Two-Mode Dashboard

**What:** Operational mode (default) vs Historical mode for different use cases

**Key insight:** A single view cannot serve both daily coordination AND historical debugging. Mode toggle provides separation of concerns.

| Mode | Shows | Purpose |
|------|-------|---------|
| Operational (default) | Active agents, Needs Attention, Recent Wins | Daily coordination |
| Historical | Full Swarm Map, Archive, SSE panels, all filters | Deep analysis |

**Key files:**
- `web/src/lib/stores/dashboard-mode.ts` - Mode state with localStorage persistence
- `web/src/routes/+page.svelte` - Conditional rendering by mode

### SSE Connections

**What:** Real-time updates via Server-Sent Events

**Key insight:** HTTP/1.1 allows only 6 connections per origin. Long-lived SSE connections occupy these slots, potentially blocking API fetches.

| Connection | Purpose | Auto-connect? |
|------------|---------|---------------|
| Events SSE | Agent activity updates | Yes |
| Agentlog SSE | Lifecycle events (spawn/complete) | **No** (opt-in via Follow button) |

**Constraint:** Agentlog SSE was changed to opt-in to prevent connection pool exhaustion (see 2026-01-05-inv-dashboard-connection-pool-exhaustion-sse.md)

### Strategic Center / Decision Center (Jan 2026 Redesign)

**What:** Meta-orchestrator decision hub replacing operational NeedsAttention component

**Key insight:** Dashboard should frame work as "what decision do I need to make?" rather than "what's wrong?"

| Category | Purpose | Examples |
|----------|---------|----------|
| Absorb Knowledge | Knowledge-producing skill completions needing synthesis | investigation, architect, research completions |
| Give Approvals | Items requiring visual verification | web/ changes with screenshot evidence |
| Answer Questions | Strategic questions blocking work | Questions from questions store |
| Handle Failures | Failed verifications, escalated agents | Dead agents, failed visual verification |
| Tend Knowledge (Future) | Knowledge hygiene signals | Synthesis opportunities, pending promotions, stale decisions |

**Status:** Designed (Jan 27-28, 2026). Implementation in progress as feat-052.

**Reference:** `.kb/investigations/2026-01-27-design-redesign-dashboard-ops-view-meta.md`, `.kb/investigations/2026-01-28-inv-design-unified-strategic-center-dashboard.md`

### Screenshot Artifacts

**What:** Dashboard tab displaying screenshots from `.orch/workspace/{agent_id}/screenshots/`

**Key features:**
- Responsive thumbnail grid (2-3 columns based on width)
- Click-to-expand modal with Escape key handling
- Lazy loading for performance
- Empty/loading/error states

**Status:** Fully implemented (Jan 2026). Available in agent detail panel.

**Integration:** `/api/screenshots` endpoint scans workspace screenshots directory, filters for image extensions (.png, .jpg, .jpeg, .gif, .webp).

**Reference:** `.kb/investigations/2026-01-17-inv-dashboard-surface-screenshot-artifacts-verification.md`

### Tmux Session Visibility (Escape Hatch Agents)

**What:** Dashboard integration for Claude CLI agents spawned with `--backend claude --tmux`

**Why it matters:** Escape hatch agents work on critical infrastructure when primary path fails - need same visibility as OpenCode agents.

| Data Point | Claude CLI (tmux) | OpenCode Agents |
|------------|-------------------|-----------------|
| Status | From beads Phase + activity detection | From session + Phase |
| Phase | From beads comments | From beads comments |
| Tokens | Not available (architectural constraint) | From OpenCode API |
| Runtime | From .spawn_time file | From session created_at |
| Activity | Transcript file mtime or pane content | Session last_updated |

**Status:** Designed (Jan 18, 2026). Partial implementation exists (beads lookup), activity detection pending.

**Reference:** `.kb/investigations/2026-01-18-design-dashboard-add-tmux-session-visibility.md`

### Follow Orchestrator (Multi-Project Filtering)

**What:** Dashboard tracks orchestrator's project context for cross-project coordination

**How:** Orchestrator context includes `included_projects` array (e.g., [orch-go, orch-cli, beads, kb-cli, orch-knowledge, opencode]). Dashboard filters agents to show only work in these projects.

**Fix (Jan 14):** Frontend serializes `included_projects` as comma-separated URL param; backend splits and matches against ANY project in the array.

**Key insight:** Cross-project agents have `ProjectDir=spawner-cwd` and `Project=target-project`, so filtering must use `Project` field for correct behavior.

**Reference:** `.kb/investigations/2026-01-14-inv-dashboard-follow-orchestrator-broken-implemented.md`

---

## Key Concepts

| Concept | Definition | Why It Matters |
|---------|------------|----------------|
| Progressive Disclosure | Collapsible Active/Recent/Archive sections | Reduces visual clutter while preserving history access |
| Stable Sort | Sort by spawned_at, not activity | Prevents grid jostling when multiple agents are processing |
| beadsFetchThreshold | Only fetch beads data for sessions < 2 hours old | Prevents O(n) performance degradation with 600+ sessions |
| session_id vs id | session_id = unique OpenCode session, id = workspace name | Use session_id as key to avoid duplicate key errors |
| is_stale | Agents older than beadsFetchThreshold (displayed with 📦) | Shows old agents without expensive beads fetch |
| project_dir | Target project from workspace cache, not session directory | Correct for --workdir spawns; session directory is orchestrator's cwd |
| Early Filtering | Apply filters before expensive operations | "Filter Early, Process Late" prevents 20s cold cache |

---

## Common Problems

### "Dashboard API takes 5-7 seconds to load"

**Cause:** Unbounded session accumulation - fetching beads data for 600+ historical sessions

**Fix:** The `beadsFetchThreshold` filter was added (2 hours). If slowness recurs:
1. Check session count: `curl -s 'http://localhost:4096/session' | jq 'length'`
2. If >500, the threshold may need tightening
3. Check cache TTLs in `serve_agents_cache.go` (should be 15-60s)

**NOT the fix:** Increasing concurrency - O(n) problem needs filtering, not parallelism

### "Dashboard shows 0 agents but API returns data"

**Cause:** Svelte 5 runes mixed with Svelte 4 reactive syntax

**Fix:** The dashboard uses **pure Svelte 4 syntax**. Do NOT use:
- `$state`
- `$derived`
- `$effect`

Using any rune triggers "runes mode" which silently breaks `$:` reactive statements and `$` store auto-subscription.

**Reference:** 2025-12-22-debug-dashboard-shows-0-agents-despite-api-returning-209.md

### "Agent cards jostling/jumping positions"

**Cause:** `is_processing` was primary sort key, causing position swaps on SSE state changes

**Fix:** When `useStableSort=true`, skip `is_processing` in sort comparison. Use `spawned_at` as primary sort key.

**Key insight:** Also add debounce (1s) for clearing `is_processing` to prevent gold border flashing.

### "each_key_duplicate Svelte error"

**Cause:** Using `agent.id` (workspace name) as key - can duplicate on respawn

**Fix:** Use `(agent.session_id ?? agent.id)` as key in all `{#each}` blocks

### "API requests showing as 'pending' in network tab"

**Cause:** HTTP/1.1 connection pool exhaustion from SSE connections

**Fix:** Agentlog SSE is now opt-in only. Primary SSE + API requests fit within 6-connection limit.

**Long-term:** Consider HTTP/2 on API server for multiplexing

### "Completed agent still shows as 'idle'"

**Cause:** Line 609 optimization in serve_agents.go skips beads fetch for idle sessions

**Fix:** The Priority Cascade Model was implemented. Ensure all agents with beadsID are added to `beadsIDsToFetch` regardless of session activity status.

### "Dashboard API still slow despite filters" (Jan 7)

**Cause:** Filters were applied at the END of the handler after all expensive operations

**Fix:** Apply filters immediately after `client.ListSessions("")`, before workspace cache building and beads operations. The pattern is **Filter Early, Process Late**.

**Reference:** 2026-01-07-inv-dashboard-api-agents-filters-applied-late.md

### "Cross-project agents not showing with project filter" (Jan 7)

**Cause:** Early project filter used `s.Directory` (session directory) which for `--workdir` spawns is the orchestrator's cwd, not the target project

**Fix:** Remove early project filter. Use late filter with `agent.ProjectDir` from workspace cache, which correctly extracts PROJECT_DIR from SPAWN_CONTEXT.md.

**Key insight:** Cross-project filtering must happen AFTER workspace cache lookup populates `project_dir`.

**Reference:** 2026-01-07-inv-dashboard-agents-filter-session-directory.md

### "Usage shows 0% when data unavailable" (Jan 7)

**Cause:** Anthropic API returns null for inactive billing periods; Go's `float64` defaults to 0, losing the null distinction

**Fix:** Use `*float64` pointers in Go struct (`UsageAPIResponse`), `number | null` in TypeScript, and display "N/A" when null.

**Key insight:** Null preservation requires explicit handling at each layer: API → Go → JSON → TypeScript → UI.

**Reference:** 2026-01-07-inv-dashboard-shows-usage-anthropic-api.md

### "Old agents completely hidden" (Jan 7)

**Cause:** Agents older than 2h `beadsFetchThreshold` were excluded via `continue`

**Fix:** Added `is_stale` boolean field. Stale agents are included in response (skip beads fetch for performance) and displayed with 📦 indicator in Archive section.

**Reference:** 2026-01-07-inv-fix-dashboard-show-older-agents.md

### "Dashboard not loading / Services not running" (Jan 21)

**Cause:** All three dashboard services stopped (OpenCode on 4096, orch API on 3348, web UI on 5188)

**Why it happens:** Claude Code runs in a Linux sandbox while service binaries are compiled for macOS ARM - agents cannot start/stop host services.

**Fix:** User must run `~/bin/orch-dashboard start` from macOS terminal (not from agent)

**Check:** `lsof -i :4096 -i :3348 -i :5188` - should show 3 processes

**Key insight:** Dashboard services are host infrastructure - agents provide observability but can't manage lifecycle.

**Reference:** 2026-01-21-inv-dashboard-not-loading-opencode-server.md

### "Price-watch agents showing in orch-go dashboard" (Jan 14)

**Cause:** Multi-project filtering incomplete - frontend stored `included_projects` but didn't serialize to URL params; backend only accepted single filter string

**Fix:** Frontend serializes `included_projects` as comma-separated param; backend splits and matches against ANY project in array

**Key insight:** Cross-project agents have `ProjectDir=spawner-cwd`, `Project=target-project` - must use `Project` field for filtering

**Reference:** 2026-01-14-inv-dashboard-follow-orchestrator-broken-implemented.md

---

## Key Decisions (from kn)

These are settled. Don't re-investigate:

- **Svelte 4 syntax only** - Don't use Svelte 5 runes until full migration (see Dec 22 investigation)
- **Dashboard gets lightweight actions; orchestrator keeps reasoning** - Control separation principle
- **Phase: Complete from beads is authoritative** - Not session activity time
- **Beads + Focus high priority; KB/KN low priority** - Dashboard is for operational awareness, not knowledge discovery
- **Two-mode approach** - Operational default, Historical opt-in
- **Slide-out panel for agent detail** - Not inline expansion

---

## What Lives Where

| Component | Location | Purpose |
|-----------|----------|---------|
| Main page | `web/src/routes/+page.svelte` | Dashboard layout and logic (~700 lines) |
| Agent store | `web/src/lib/stores/agents.ts` | Agent data, SSE handling, derived stores |
| Agent card | `web/src/lib/components/agent-card/` | Individual agent display |
| Agent detail panel | `web/src/lib/components/agent-detail/` | Slide-out panel for selected agent |
| Collapsible sections | `web/src/lib/components/collapsible-section/` | Active/Recent/Archive grouping |
| API server | `cmd/orch/serve.go` | Main serve command + handlers |
| Agent API | `cmd/orch/serve_agents.go` | /api/agents logic (~1400 lines) |
| Caching | `cmd/orch/serve_agents_cache.go` | TTL caches for beads, workspaces |
| Playwright tests | `web/tests/` | E2E tests (filtering, stats bar, race conditions) |

---

## Debugging Checklist

Before spawning an investigation about dashboard issues:

1. **Check kb:** `kb context "dashboard"`
2. **Check this guide:** You're reading it
3. **Check API response:** `curl -s http://localhost:3348/api/agents | jq '. | length'`
4. **Check OpenCode sessions:** `curl -s http://localhost:4096/session | jq 'length'`
5. **Check console errors:** Browser dev tools for Svelte/JS errors
6. **Check network tab:** Look for pending/failed requests, connection count
7. **Run tests:** `cd web && bunx playwright test`

If those don't answer your question, then investigate. But update this doc with what you learn.

---

## Performance Considerations

| Metric | Threshold | Fix if Exceeded |
|--------|-----------|-----------------|
| /api/agents response | <500ms | Check session count, verify beadsFetchThreshold filter |
| Session count | <500 | Restart OpenCode server to clear accumulated sessions |
| Cache hit rate | >80% | Check TTL values in serve_agents_cache.go |
| Active SSE connections | ≤2 | Agentlog should be opt-in only |

### Performance Patterns (Lessons from 4+ Slowness Incidents)

The dashboard has had recurring performance issues. They follow predictable patterns:

| Date | Sessions | Root Cause | Fix |
|------|----------|------------|-----|
| Dec 22 | 209 | Svelte 5 runes broke reactivity | Remove runes |
| Dec 27 | 564 | O(N) sequential RPC calls | Parallelization |
| Jan 6 | 623 | Session accumulation, cache TTLs | 2h threshold |
| Jan 7 | 226 | O(n²) investigation discovery + filter timing | Cache + early filter |

**Key principles:**
1. **Always profile before fixing** - Timing logs revealed actual bottlenecks
2. **Check previous fixes** - Threshold regressions (24h → 2h) are common
3. **O(n²) hides in function calls** - Investigation discovery looked innocent but scaled terribly
4. **Thresholds need justification** - "2 hours" matches operational reality, "24 hours" was arbitrary

### Filter Timing (Jan 7 Discovery)

**The pattern:** "Process everything, filter late" defeats the purpose of filtering.

**The fix:** Apply filters immediately after `client.ListSessions("")`, before:
- Workspace cache building
- Beads batch operations  
- Investigation directory scanning
- Token fetching

This reduces workload proportionally to filter selectivity.

### Caching Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    serve_agents_cache.go                     │
├─────────────────────────────────────────────────────────────┤
│ openIssuesCache      │ bd list --open    │ TTL: 30s        │
│ allIssuesCache       │ bd list --all     │ TTL: 60s        │
│ commentsCache        │ bd comments <id>  │ TTL: 15s        │
│ workspacesByBeadsID  │ Workspace lookup  │ Populated once  │
│ investigationDirCache│ .kb/investigations│ Per-request     │
│ beadsStatsCache      │ bd stats per proj │ Per-project     │
└─────────────────────────────────────────────────────────────┘
```

**Investigation directory cache (Jan 7):** Build once before agent loop. Changes O(n×m) to O(n+m) for investigation discovery.

**Per-project beads cache (Jan 7):** Keyed by project directory to support cross-project views and "Follow Orchestrator" mode.

---

## Integration Points

### Stats Bar Indicators

| Indicator | Source | Data |
|-----------|--------|------|
| Active agents | agents store | Count of status="active" |
| Recent | agents store | Completed within 24h |
| Archive | agents store | Completed >24h ago |
| Errors | agentlog store | Error events count |
| Usage | /api/usage | Weekly Claude Max % |
| Focus | /api/focus | Goal + drift status |
| Beads | /api/beads | Ready/blocked/in-progress counts |
| Daemon | /api/daemon | Capacity + running status |
| Servers | /api/servers | Running server count |

### Activity Feed Persistence (Jan 7 Design)

**Problem:** Activity events stored in global 1000-event buffer, diluted across agents, lost on refresh.

**Solution (Designed, not yet implemented):** Hybrid SSE + API architecture

| Source | Purpose | When Used |
|--------|---------|-----------|
| SSE | Real-time updates | Always connected |
| OpenCode API | Historical data | On activity tab open |

OpenCode persists all session data to `~/.local/share/opencode/storage/` and exposes it via `GET /session/:sessionID/message`. Dashboard should treat SSE as real-time updates and API as source of truth for history.

**Reference:** 2026-01-07-design-dashboard-activity-feed-persistence.md

### Future Integrations (Deferred)

- **KB/KN panels** - Not recommended; CLI tools serve this better
- **Orchestrator sessions** - Partial implementation exists but paused
- **Activity feed persistence** - Designed, awaiting implementation

---

## References

**Related decisions:**
- `.kb/decisions/2025-12-21-minimal-artifact-taxonomy.md`
- `.kb/decisions/2026-01-17-event-sourced-monitoring-architecture.md`
- `.kb/decisions/2026-01-24-readable-frontier-over-graph-visualization.md`
- `.kb/decisions/2026-01-30-strategic-center-dashboard-architecture.md`
- `.kb/decisions/2026-01-30-sse-reconnection-resilience-patterns.md`

**Key investigations (by theme):**

*Performance:*
- `2026-01-07-inv-dashboard-api-agents-performance-synthesis.md` - O(n²) investigation discovery fix (51x improvement)
- `2026-01-07-inv-dashboard-api-agents-filters-applied-late.md` - Filter timing fix
- `2026-01-06-inv-dashboard-api-slow-again-623.md` - Session accumulation fix
- `2026-01-05-inv-dashboard-connection-pool-exhaustion-sse.md` - HTTP/1.1 limits

*Cross-Project Visibility:*
- `2026-01-07-inv-dashboard-agents-filter-session-directory.md` - project_dir vs session directory
- `2026-01-07-inv-dashboard-beads-follow-orchestrator-tmux.md` - Per-project beads cache
- `2026-01-14-inv-dashboard-follow-orchestrator-broken-implemented.md` - Multi-project filtering fix (comma-separated params)
- `2026-01-16-inv-dashboard-follow-mode-project-mismatch.md` - Follow mode project matching

*Data Pipeline Integrity:*
- `2026-01-07-inv-dashboard-shows-usage-anthropic-api.md` - Null handling with pointer types
- `2026-01-07-inv-fix-dashboard-show-older-agents.md` - is_stale field for old agents

*UX/Stability:*
- `2025-12-26-inv-dashboard-agent-cards-rapidly-jostling.md` - Stable sort fix
- `2025-12-25-inv-fix-dashboard-each-key-duplicate.md` - Key uniqueness
- `2025-12-24-inv-implement-progressive-disclosure-swarm-dashboard.md` - Collapsible sections

*Architecture:*
- `2026-01-07-design-dashboard-activity-feed-persistence.md` - Hybrid SSE + API architecture
- `2025-12-27-inv-dashboard-two-modes-operational-default.md` - Two-mode design
- `2026-01-04-design-dashboard-agent-status-model.md` - Priority cascade model
- `2025-12-26-design-web-dashboard-daemon-visibility.md` - Daemon integration
- `2026-01-18-design-dashboard-add-tmux-session-visibility.md` - Tmux escape hatch agent visibility

*Strategic Center / UX Redesign:*
- `2026-01-27-design-redesign-dashboard-ops-view-meta.md` - Decision Center (action-oriented UX)
- `2026-01-28-inv-design-unified-strategic-center-dashboard.md` - 5-category Strategic Center
- `2026-01-09-inv-dashboard-add-approval-action-design.md` - Approval workflow for visual verification

*Screenshot Artifacts:*
- `2026-01-17-inv-dashboard-surface-screenshot-artifacts-verification.md` - Screenshots tab implementation
- `2026-01-16-inv-dashboard-add-image-paste-upload.md` - Image upload design

*Service Reliability:*
- `2026-01-21-inv-dashboard-not-loading-opencode-server.md` - Services lifecycle vs agent sandbox constraint
- `2026-01-16-inv-orch-dashboard-handle-already-running.md` - Dashboard startup handling

*Bug fixes:*
- `2025-12-22-debug-dashboard-shows-0-agents-despite-api-returning-209.md` - Svelte 5 runes

**Source code:**
- `web/` - SvelteKit 5 frontend
- `cmd/orch/serve*.go` - Go backend

---

## History

- **2025-12-21:** Initial dashboard created (orch-go port from Python)
- **2025-12-22:** Major bug fix - Svelte 5 runes incompatibility
- **2025-12-23:** Audit revealed 6 test failures, 1 code bug
- **2025-12-24:** Progressive disclosure implemented, account name display
- **2025-12-25:** Agent details pane redesign, each_key_duplicate fix
- **2025-12-26:** Daemon visibility design, agent jostling fix, theme system
- **2025-12-27:** Two-mode dashboard (Operational/Historical)
- **2026-01-04:** Agent status model redesign (priority cascade)
- **2026-01-05:** Connection pool exhaustion fix
- **2026-01-06:** Guide created synthesizing 44 investigations
- **2026-01-07:** Major performance work - O(n²) investigation discovery fix (51x improvement), early filter application, cross-project visibility fixes, null/stale handling, activity feed persistence design. Guide updated with 14 new investigations (58 total)
- **2026-01-09:** Approval action design for visual verification workflow
- **2026-01-14:** Follow-orchestrator multi-project filtering fix (comma-separated projects param)
- **2026-01-17:** Screenshot artifacts feature verified complete (thumbnails, click-to-expand)
- **2026-01-18:** Tmux session visibility design for Claude CLI escape hatch agents
- **2026-01-21:** Service lifecycle investigation (dashboard services vs agent sandbox constraint)
- **2026-01-27:** Strategic Center UX redesign - from status-oriented to action-oriented (Decision Center)
- **2026-01-28:** Strategic Center expanded to 5 categories (added "Tend Knowledge" for knowledge hygiene)
- **2026-01-29:** Guide updated synthesizing 24 new investigations (Jan 7-29), total 82+ investigations
