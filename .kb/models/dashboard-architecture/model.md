# Model: Dashboard Architecture

**Domain:** Dashboard / Web UI
**Last Updated:** 2026-03-06
**Synthesized From:** 62 investigations (Dec 21, 2025 - Jan 8, 2026) + 14 probes (Feb 2026)

---

## Summary (30 seconds)

The dashboard is a **3-page SPA** (Svelte 5 + SvelteKit, adapter-static) served by `orch serve` (Go backend on port 3348) with a Vite dev server on port 5188 proxying to the backend.

**Three views, three distinct purposes:**
- `/` (Dashboard) — Agent monitoring: real-time swarm status, health coaching, performance
- `/work-graph` — Work tracking: beads issue tree with dependencies, attention signals, verification gates
- `/knowledge-tree` — Knowledge browsing: .kb/ artifact tree, session timeline (3 tabs: Knowledge/Work/Timeline)

**Critical context (Option A+):** The Dashboard (`/`) is Dylan's primary agent monitoring layer. Dylan also uses the Knowledge Tree Work tab daily for issue tracking. The Work Graph is the intended primary work-tracking layer (once stable). Dashboard failure = Dylan is blind to agent health. This makes dashboard reliability tier-0 infrastructure.

The architecture uses a **two-mode design** (Operational/Historical) to separate daily coordination from deep analysis. SSE connections enable real-time updates but are constrained by HTTP/1.1's 6-connection limit. Progressive disclosure and stable sorting prevent information overload while maintaining scan-ability.

---

## Core Mechanism

### Overall Architecture

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

### Key Components

**Frontend (Svelte 5 + SvelteKit):**
- `web/src/routes/+page.svelte` - Main dashboard page (~1043 lines, predominantly Svelte 4 patterns)
- `web/src/routes/work-graph/+page.svelte` - Work graph page (~1043 lines)
- `web/src/routes/knowledge-tree/+page.svelte` - Knowledge tree page (~371 lines, Svelte 5 runes)
- `web/src/lib/stores/` - 25+ reactive stores (agents, beads, daemon, wip, attention, work-graph, etc.)
- `web/src/lib/components/` - 40+ components in 6 categories (UI primitives, agent display, section containers, work/knowledge, data panels, controls)

**CSS/Theming:**
- Tailwind CSS v3 + shadcn-svelte (slate base) + HSL CSS variables
- 28 JSON theme files (`web/src/lib/themes/`) — catppuccin, dracula, tokyonight, etc.
- Dark mode via `dark` class on `<html>`
- Agent-specific colors: `swarm.active` (green), `swarm.completed` (blue), `swarm.abandoned` (red), `swarm.idle` (yellow)

**Responsive breakpoints (Tailwind):**
- `sm:` 640px — 2-col grids, show/hide text labels (primary structural shift)
- `md:` 768px — 3-col grids
- `lg:` 1024px — 4-col grids, 2-col event panels, side panel sizing
- `xl:` 1280px — 5-col agent grids
- `2xl:` 1400px — container max-width only
- Primary agent grid pattern: `grid sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5`

**Backend (Go):**
- `cmd/orch/serve_agents.go` - Agent status calculation (~1400 lines)
- `cmd/orch/serve_beads.go` - Beads graph API including `buildActiveAgentMap()`
- `cmd/orch/serve.go` - HTTP server setup, routing
- `cmd/orch/serve_tree.go` - Knowledge tree API + `/api/events/tree` SSE endpoint (polls filesystem every 2s)
- `pkg/attention/` - 11 signal collectors feeding `/api/attention`

### State Transitions

**Dashboard mode lifecycle:**

```
User loads dashboard
    ↓
Mode = Operational (default, from localStorage)
    ↓
Shows: Active agents, Needs Attention, Recent Wins
    ↓
User clicks "Historical" toggle
    ↓
Mode = Historical
    ↓
Shows: Full Swarm Map, Archive, SSE panels, all filters
```

**SSE connection lifecycle:**

```
Dashboard loads
    ↓
Events SSE connects automatically (/api/events)
    ↓
Agentlog SSE: opt-in via Follow button (/api/agentlog)
    ↓
HTTP/1.1 connection pool: 6 slots total
    ↓
Long-lived SSE occupies 1-2 slots
    ↓
Remaining 4-5 slots for API fetches
```

### Critical Invariants

1. **Two-mode design is mutually exclusive** - Cannot show both Operational and Historical views simultaneously
2. **SSE Events auto-connect, Agentlog is opt-in** - Connection pool management
3. **beadsFetchThreshold controls remote queries** - 5+ ready issues triggers `bd ready` shell-out
4. **Progressive disclosure via collapsed panels** - Event panels start collapsed, expand on click
5. **Stable sort maintains scan-ability** - Agent order doesn't change unless status changes
6. **Early filtering reduces payload size** - Backend filters before sending to frontend
7. **buildActiveAgentMap() is local-project-scoped** - Cross-project graph requests get nodes but NOT active agent enrichment. Any cross-project in_progress issue shows 'unassigned' as a result.
8. **Promoted sections must participate in pinnedTreeIds deduplication** - Any section that pulls items out of the work-graph tree must register IDs in `pinnedTreeIds` to prevent double-rendering. Currently only the WIP section does this; Ready to Complete does not.
9. **State persistence: localStorage primary, URL hash for deep-linking** - UI state (expansion, tab selection, view mode) persists in localStorage; URL hash used additionally for bookmarkable views (knowledge-tree tabs).

---

## Why This Fails

### Failure Mode 1: Connection Pool Exhaustion

**Symptom:** API fetches hang or timeout when SSE panels open

**Root cause:** HTTP/1.1 allows only 6 connections per origin; SSE occupies slots

**Why it happens:**
- Events SSE (auto-connect): 1 slot
- Agentlog SSE (auto-connect before fix): 1 slot
- Remaining 4 slots for API fetches
- If 5+ API requests concurrent, some block

**Fix (Jan 5):** Made Agentlog SSE opt-in via Follow button, freeing 1 slot

### Failure Mode 2: Slow Dashboard Load with 100+ Agents

**Symptom:** Dashboard takes 5-10 seconds to load with many agents

**Root cause:** `/api/agents` endpoint performs expensive operations (OpenCode queries, beads parsing) synchronously

**Why it happens:**
- Each agent requires OpenCode session query
- Full beads issue parsing for each agent
- No caching, recomputed on every request

**Fix (Jan 6):** Response caching with 2-second TTL, reduced load time to <1 second

### Failure Mode 3: Information Overload in Operational Mode

**Symptom:** Users overwhelmed by full swarm map with 50+ agents

**Root cause:** Single view tried to serve both daily coordination and deep analysis

**Why it happens:**
- Operational needs: "What's ready? What's broken?"
- Historical needs: "Show me everything, all filters, full archive"
- One view can't optimize for both

**Fix (Jan 7):** Two-mode design - Operational (focused) vs Historical (comprehensive)

### Failure Mode 4: Plugin Cascade (Dashboard "Disconnected" Despite Services Running)

**Symptom:** Dashboard shows "disconnected", `overmind status` shows all 3 services running, but `orch status` returns HTTP 500

**Root cause:** OpenCode plugin error (e.g., v1→v2 API incompatibility) crashes OpenCode's internal request handling

**Why it happens:**
- OpenCode loads plugins at startup
- Bad plugin throws error on every request
- `/api/agents` calls OpenCode → gets 500
- Dashboard can't fetch agent data → shows "disconnected"
- overmind sees process running (not crashed) → reports "running"

**Cascade:**
```
Plugin error → OpenCode internal 500 → orch status fails → API can't get agents → Dashboard "disconnected"
```

**Fix (Jan 14):** Disable plugins, restart OpenCode, re-enable one-by-one. Root cause was session-resume.js using v1 API (object export) instead of v2 (function export).

**Key insight:** Dashboard can appear "down" while all processes are technically "running". Health checks must verify data flow, not just port availability.

### Failure Mode 5: Attention Badge False Positives (75% Noise Rate)

**Symptom:** Work-graph tree shows amber "Awaiting verification" badges on ~75% of open issues

**Root cause:** `mapSignalToBadge()` in `attention.ts` had `default: return 'verify'` — any unmapped signal produced a false badge

**Why it happens:**
- 11 backend collectors emit signals; `mapSignalToBadge()` only has explicit cases for 6
- `issue-ready` (BeadsCollector) is the dominant unmapped signal — hits default → `'verify'`
- With 34/45 open issues having `issue-ready` signals, 75% got false "Awaiting verification" badges

**Fix (Feb 16):** Three-layer defense — changed default to `return null`; added null filter in attention store before `signals.set()`; added null guard in `work-graph-tree-helpers.ts`.

**Additional gap:** 3 of 9 badge types (`decide`, `escalate`, `crashed`) have no backend collector — they exist only in the frontend type system and can never fire.

### Failure Mode 6: Stub Store Crashes Page Permanently

**Symptom:** Work-graph page shows "Loading work graph..." indefinitely, never renders

**Root cause:** `wip.fetchQueued()` method did not exist on the stub `wip` store. Calling `undefined()` in `onMount` throws a `TypeError` that propagates through the async function, preventing `loading = false` from executing.

**Why it happens:**
- The `wip` store was scaffolded as a 68-line stub with TODO comments
- Page code calls methods on the stub that were never implemented
- `TypeError: wip.fetchQueued is not a function` thrown synchronously before `.catch()` can intercept it
- `loading` stays `true` → page never exits loading state

**Fix (Feb 16):** Added 4 stub methods (`wip.fetchQueued`, `wip.setRunningAgents`, `focus.clearFocus`, `focus.setFocus`) with minimal no-op implementations that satisfy the call sites without crashing.

### Failure Mode 7: Unguarded Browser APIs in Lifecycle Hooks (SSR 500)

**Symptom:** 500 error during initial page load on the knowledge-tree route

**Root cause:** `window.removeEventListener()` called in `onDestroy` without an SSR guard

**Why it happens:**
- SvelteKit runs component code both server-side and client-side
- Lifecycle hooks (`onDestroy`) run during SSR hydration
- `window` doesn't exist in Node.js context → `ReferenceError` → 500

**Fix (Feb 16):** Wrap all browser API access in lifecycle hooks with `typeof window !== 'undefined'` guard. The knowledge-tree page already used this pattern for localStorage access — the `onDestroy` cleanup missed it.

### Failure Mode 8: Knowledge Tree SSE Cycling (Expansion State Reset)

**Symptom:** Knowledge-tree tree view visually resets all nodes to collapsed state on each SSE update (every ~2 seconds when files change)

**Root cause:** SSE `tree-update` events send full tree replacements via `set({ tree })`, wiping client-side `expanded` properties on all nodes

**Why it happens:**
- `/api/events/tree` backend polls filesystem every 2 seconds
- Each change sends complete tree structure (not a diff)
- Store handler does `set({ tree })` replacing entire tree object
- Tree node `expanded?: boolean` properties set by user are lost
- UI re-renders with all nodes in default (collapsed) state

**Fix:** Preserve expansion state across SSE updates by merging localStorage-tracked expanded IDs back into incoming tree, or making expansion state purely local (not stored on tree nodes).

### Failure Mode 9: Cross-Project In-Progress Issues Show 'unassigned'

**Symptom:** Work-graph in_progress issues from non-local projects (e.g., `toolshed-*`) always show 'unassigned' instead of agent info

**Root cause:** `buildActiveAgentMap()` (`serve_beads.go:1047`) is scoped to local project only

**Why it happens:**
1. `listTrackedIssues()` uses `beads.FindSocketPath("")` (local beads only); cross-project IDs never enter the map
2. `client.ListSessions("")` queries OpenCode for default project scope only
3. Frontend `getInProgressSubline()` correctly falls through to `'unassigned'` when `active_agent` is null

**Fix available but not applied:** `listSessionsAcrossProjects()` exists in `serve_agents_cache.go:396` and was created for exactly this purpose; it needs to be used in `buildActiveAgentMap()`. `listTrackedIssues()` also needs to accept `projectDirs` and query each project's beads.

### Failure Mode 10: Missing `phase_reported_at` in API Response

**Symptom:** "Ready to Complete" section in work-graph never populated despite agents reaching phase=complete

**Root cause:** `PhaseReportedAt` timestamp was tracked internally in a map in `serve_agents.go` but never added to `AgentAPIResponse` struct

**Why it happens:**
- Frontend line 352: `if (!completionAt) continue;` — skips items without completion timestamp
- `completionAt` derived from `agent.phase_reported_at`
- Field not in JSON response → always undefined → all complete agents filtered out

**Fix (Feb 16):** Added `PhaseReportedAt string json:"phase_reported_at,omitempty"` to `AgentAPIResponse` struct and populate it when parsing phase from beads comments.

---

## Constraints

### Why HTTP/1.1 Connection Limit?

**Constraint:** Browsers limit HTTP/1.1 to 6 connections per origin

**Implication:** Long-lived SSE connections reduce slots available for API fetches

**Workaround:** Make SSE connections opt-in, use HTTP/2 (future), or batch API requests

**This enables:** Simple SSE implementation without server complexity
**This constrains:** Cannot have unlimited real-time streams

### Why Two Modes Instead of Smart Filtering?

**Constraint:** A single view optimized for "daily work" is too dense for deep analysis, and vice versa

**Implication:** Two distinct UX patterns for different cognitive modes

**Workaround:** Mode toggle with localStorage persistence

**This enables:** Focused daily coordination AND deep historical analysis
**This constrains:** Cannot see both views simultaneously

### Why Cache with Short TTL?

**Constraint:** Agent status changes frequently (every few seconds), but queries are expensive

**Implication:** Cache must be short-lived (2 seconds) to avoid stale data

**Workaround:** Balance freshness vs performance

**This enables:** Fast dashboard loads without expensive recomputation
**This constrains:** Multiple requests within 2 seconds see same data (eventual consistency)

### Why beadsFetchThreshold at 5?

**Constraint:** Shelling out to `bd ready` is expensive (~500ms)

**Implication:** Only query beads when likely to have useful data

**Workaround:** Use frontend-visible ready count as heuristic

**This enables:** Avoid unnecessary beads queries
**This constrains:** Ready queue might be incomplete if <5 issues

---

## Evolution

**Dec 21-24, 2025: Initial Implementation**
- Basic agent listing
- SSE events for real-time updates
- No filtering, no modes

**Dec 26-30, 2025: Performance Issues**
- Slow loads with 100+ agents
- Connection pool exhaustion discovered
- Response caching added

**Jan 3-5, 2026: UX Refinement**
- Progressive disclosure via collapsed panels
- Stable sort to maintain scan-ability
- Agentlog SSE made opt-in

**Jan 7, 2026: Two-Mode Design + Follow-Orchestrator**
- Operational vs Historical modes
- Mode toggle with localStorage persistence
- Conditional rendering based on mode
- Dashboard beads follow orchestrator's tmux context via project_dir parameter
- Per-project caching for multi-project orchestration support
- Reactive frontend updates when orchestrator switches projects

**Jan 8, 2026: Synthesis and Cleanup**
- 62 investigations synthesized into guide
- Common problems documented
- Architecture stabilized

**Jan 14, 2026: Option A+ and Plugin Cascade**
- Established Option A+ model: dashboard is Dylan's ONLY observability layer
- Discovered Failure Mode 4: plugin cascade (services running but dashboard broken)
- Fixed session-resume.js v1→v2 API migration
- Documented ONE process manager rule (overmind exclusive)
- Created infrastructure complexity decision (keep architecture, fix gaps)

**Feb 2026: Three-View Architecture + Attention System + Work Graph**
- Evolved from single dashboard to 3-page SPA: `/`, `/work-graph`, `/knowledge-tree`
- Work-graph added: issue tree with dependencies, attention badges, ready-to-complete queue; broken on creation (stub store crash)
- Knowledge-tree added: .kb/ artifact browsing, session timeline, SSE-driven live updates
- Discovered and fixed 6 new failure modes (see "Why This Fails" sections 5-10)
- Attention system audit: 11 real collectors, only 2 firing; fixed false-badge cascade (default→null)
- `phase_reported_at` field added to agents API; ready-to-complete section now functional
- Knowledge-tree: SSR guard fix, tab persistence (URL hash + localStorage), duplicate deduplication (`cloneNodeRecursiveWithDedup`, first-parent-wins)
- `buildActiveAgentMap()` identified as local-only; cross-project active-agent gap documented (fix available: `listSessionsAcrossProjects`)

---

## References

**Key Investigations:**
- `2026-01-05-inv-dashboard-connection-pool-exhaustion-sse.md` - HTTP/1.1 connection limit discovery
- `2026-01-06-inv-dashboard-slow-load-caching.md` - Response caching implementation
- `2026-01-07-inv-dashboard-two-mode-design.md` - Operational/Historical separation
- `2025-12-24-inv-dashboard-progressive-disclosure.md` - Collapsed panels
- `2025-12-26-inv-dashboard-stable-sort.md` - Scan-ability via sort stability
- ...and 57 others

**Decisions Informed by This Model:**
- Two-mode design (cognitive separation)
- Agentlog SSE opt-in (connection pool management)
- 2-second cache TTL (freshness vs performance)
- beadsFetchThreshold=5 (avoid expensive queries)

**Related Models:**
- `.kb/models/dashboard-agent-status.md` - How agent status is calculated (Priority Cascade)
- `.kb/models/opencode-session-lifecycle/model.md` - How dashboard queries session state

**Related Guides:**
- `.kb/guides/dashboard.md` - How to use dashboard, troubleshoot issues (procedural)
- `.kb/guides/dev-environment-setup.md` - Service management, ONE process manager rule

**Related Decisions:**
- `.kb/decisions/2026-01-14-infrastructure-complexity-justified.md` - Why we keep 3-service architecture
- `.kb/decisions/2026-01-09-dashboard-reliability-architecture.md` - orch doctor, orch deploy design

**Primary Evidence (Verify These):**
- `cmd/orch/serve_agents.go` - Agent status calculation and API endpoint (~1400 lines)
- `cmd/orch/serve_beads.go` - Beads graph API, `buildActiveAgentMap()` (local-scoped)
- `cmd/orch/serve.go` - HTTP server setup (~600 lines)
- `web/src/routes/+page.svelte` - Main dashboard page (~1043 lines)
- `web/src/routes/work-graph/+page.svelte` - Work graph page (~1043 lines)
- `web/src/routes/knowledge-tree/+page.svelte` - Knowledge tree page (~371 lines)
- `web/src/lib/stores/attention.ts` - Attention store; `mapSignalToBadge()` default fixed to `null`
- `web/src/lib/stores/wip.ts` - WIP store (still mostly stub, 4 methods added Feb 16)
- `pkg/attention/` - 11 signal collectors
- `pkg/tree/tree.go` - Knowledge tree building; `cloneNodeRecursiveWithDedup()` for deduplication

---

### Merged Probes

All 14 probes merged into this model on 2026-03-06:

| Probe | Verdict | Summary |
|-------|---------|---------|
| `2026-02-15-knowledge-tree-sse-cycling-fix` | EXTENDS | SSE full-tree replacements wipe client expand/collapse state; fix: merge expansion state on update |
| `2026-02-16-agents-api-phase-field-missing` | EXTENDS | `phase_reported_at` missing from API struct; added to fix Ready-to-Complete section |
| `2026-02-16-attention-badge-verify-noise` | EXTENDS | `default: return 'verify'` in `mapSignalToBadge()` caused 75% false-positive badges |
| `2026-02-16-attention-badge-verify-noise-fix` | CONFIRMS | Three-layer fix implemented: null default + store filter + tree-helpers guard |
| `2026-02-16-attention-pipeline-full-audit` | EXTENDS | 11 real collectors, only 2 firing; 3 badge types (`decide`, `escalate`, `crashed`) have no collector |
| `2026-02-16-knowledge-tree-duplicate-items-across-phase-groups` | EXTENDS | Multi-parent Prior-Work references create duplicate tree nodes (root cause identified) |
| `2026-02-16-knowledge-tree-ssr-window-check` | EXTENDS | Unguarded `window` in `onDestroy` caused SSR 500; fixed with `typeof window` guard |
| `2026-02-16-knowledge-tree-tab-persistence` | EXTENDS | Tab state now persisted via URL hash (primary) + localStorage (fallback) |
| `2026-02-16-three-view-consolidation-assessment` | EXTENDS + CONTRADICTS | Architecture evolved to 3-page SPA; contradicts "dashboard is ONLY observability layer" |
| `2026-02-16-work-graph-issues-view-section-design` | EXTENDS | Ready-to-Complete section outside `pinnedTreeIds` mechanism causes double-rendering |
| `2026-02-16-work-graph-missing-store-methods` | CONFIRMS | Confirms stub-store crash diagnosis; 4 missing methods added to unblock rendering |
| `2026-02-17-knowledge-tree-duplicate-fix` | CONFIRMS | Deduplication already fixed via `cloneNodeRecursiveWithDedup()`; first-parent-wins strategy |
| `2026-02-25-probe-work-graph-unassigned-cross-project` | EXTENDS | `buildActiveAgentMap()` local-only; cross-project in_progress issues always show 'unassigned' |
| `2026-02-25-probe-dashboard-web-ui-framework-and-responsive-patterns` | EXTENDS | Full tech stack: shadcn-svelte + Tailwind v3 + 28 themes + 25 stores + 5-tier responsive grid |

## Auto-Linked Investigations

- .kb/investigations/archived/2026-01-05-design-review-dashboard-architecture-request-handling.md
- .kb/investigations/2026-02-13-inv-recover-dashboard-web-ui-entropy.md
- .kb/investigations/archived/2025-12-23-inv-audit-swarm-dashboard-web-ui.md
- .kb/investigations/2026-01-18-inv-update-dashboard-architecture-md-evolution.md
- .kb/investigations/archived/2025-12-26-design-web-dashboard-daemon-visibility.md
- .kb/investigations/archived/2026-01-10-inv-dashboard-supervision-circular-debugging.md
