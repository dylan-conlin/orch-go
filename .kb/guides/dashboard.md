# Dashboard

**Purpose:** Single authoritative reference for the Swarm Dashboard web UI. Read this before debugging dashboard issues or implementing new features.

**Last verified:** 2026-01-06

---

## Overview

The Swarm Dashboard is a web-based monitoring UI for the orchestration system, served via `orch serve`. It provides real-time visibility into agent status, daemon health, and operational metrics. The dashboard evolved significantly from Dec 21, 2025 to Jan 6, 2026, addressing performance, UX, and architectural issues through 44+ investigations.

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

---

## Key Concepts

| Concept | Definition | Why It Matters |
|---------|------------|----------------|
| Progressive Disclosure | Collapsible Active/Recent/Archive sections | Reduces visual clutter while preserving history access |
| Stable Sort | Sort by spawned_at, not activity | Prevents grid jostling when multiple agents are processing |
| beadsFetchThreshold | Only fetch beads data for sessions < 2 hours old | Prevents O(n) performance degradation with 600+ sessions |
| session_id vs id | session_id = unique OpenCode session, id = workspace name | Use session_id as key to avoid duplicate key errors |

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

### Caching Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    serve_agents_cache.go                     │
├─────────────────────────────────────────────────────────────┤
│ openIssuesCache      │ bd list --open    │ TTL: 30s        │
│ allIssuesCache       │ bd list --all     │ TTL: 60s        │
│ commentsCache        │ bd comments <id>  │ TTL: 15s        │
│ workspacesByBeadsID  │ Workspace lookup  │ Populated once  │
└─────────────────────────────────────────────────────────────┘
```

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

### Future Integrations (Deferred)

- **KB/KN panels** - Not recommended; CLI tools serve this better
- **Orchestrator sessions** - Partial implementation exists but paused

---

## References

**Related decisions:**
- `.kb/decisions/2025-12-21-minimal-artifact-taxonomy.md`

**Key investigations (by theme):**

*Performance:*
- `2026-01-06-inv-dashboard-api-slow-again-623.md` - Session accumulation fix
- `2026-01-05-inv-dashboard-connection-pool-exhaustion-sse.md` - HTTP/1.1 limits

*UX/Stability:*
- `2025-12-26-inv-dashboard-agent-cards-rapidly-jostling.md` - Stable sort fix
- `2025-12-25-inv-fix-dashboard-each-key-duplicate.md` - Key uniqueness
- `2025-12-24-inv-implement-progressive-disclosure-swarm-dashboard.md` - Collapsible sections

*Architecture:*
- `2025-12-27-inv-dashboard-two-modes-operational-default.md` - Two-mode design
- `2026-01-04-design-dashboard-agent-status-model.md` - Priority cascade model
- `2025-12-26-design-web-dashboard-daemon-visibility.md` - Daemon integration

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
