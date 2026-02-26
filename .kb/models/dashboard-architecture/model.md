# Model: Dashboard Architecture

**Domain:** Dashboard / Web UI
**Last Updated:** 2026-01-14
**Synthesized From:** 62 investigations (Dec 21, 2025 - Jan 8, 2026) into dashboard performance, UX, and architectural issues

---

## Summary (30 seconds)

The Swarm Dashboard is a Svelte 5 web UI served by `orch serve` (Go backend) that provides real-time monitoring of agent status, daemon health, and operational metrics.

**Critical context (Option A+):** The dashboard is Dylan's (meta-orchestrator's) ONLY observability layer. He does not use CLI tools directly. Dashboard failure = Dylan is blind. This makes dashboard reliability tier-0 infrastructure. See orchestrator skill "Observability Architecture (Option A+)" section.

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

**Frontend (Svelte 5):**
- `web/src/routes/+page.svelte` - Main dashboard page with mode-conditional rendering
- `web/src/lib/stores/` - Reactive state management (agents, beads, daemon, etc.)
- `web/src/lib/components/` - UI components (SwarmMap, StatsBar, EventPanels)

**Backend (Go):**
- `cmd/orch/serve_agents.go` - Agent status calculation (~1400 lines)
- `cmd/orch/serve.go` - HTTP server setup, routing
- `pkg/dashboard/` - Data aggregation, caching

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
- `cmd/orch/serve.go` - HTTP server setup (~600 lines)
- `web/src/routes/+page.svelte` - Main dashboard page (~800 lines)
- `web/src/lib/stores/dashboard-mode.ts` - Mode state management
- `web/src/lib/stores/agents.ts` - Agent data store with SSE updates
