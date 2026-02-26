**TLDR:** Question: How to scaffold a modern beads-ui v2 for the Headless Swarm Dashboard? Answer: Created `web/` directory with SvelteKit 5, Tailwind CSS v3, and shadcn-svelte components (Card, Badge, Button). Dashboard displays agents from orch-go registry with status badges and SSE event stream placeholder. High confidence (90%) - builds and type-checks, but SSE integration not yet wired up.

---

# Investigation: Scaffold beads-ui v2 (Bun + SvelteKit 5 + shadcn-svelte)

**Question:** How should the new beads-ui v2 "Swarm Dashboard" be structured to monitor orch-go agents and SSE events?

**Started:** 2025-12-20
**Updated:** 2025-12-20
**Owner:** og-work-scaffold-beads-ui-20dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Existing beads-ui-svelte project already has the stack

**Evidence:** Found `~/Documents/personal/beads-ui-svelte/` with SvelteKit 5.43.8, Bun, Tailwind CSS, and shadcn-svelte already configured.

**Source:** 
- `~/Documents/personal/beads-ui-svelte/package.json`
- `~/Documents/personal/beads-ui-svelte/components.json`

**Significance:** Could have extended existing project, but decided to create embedded `web/` directory in orch-go for simpler deployment and separation of concerns (beads issues vs. orchestrator monitoring).

---

### Finding 2: orch-go SSE client connects to OpenCode at /event

**Evidence:** `pkg/opencode/sse.go` implements SSE client that parses `session.status` events for idle/busy transitions.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/pkg/opencode/sse.go:24-159`

**Significance:** The web dashboard can proxy to OpenCode's SSE endpoint or orch-go can expose its own aggregated stream. For now, Vite config proxies `/api/events` to `http://127.0.0.1:4096/event`.

---

### Finding 3: Agent registry provides the data model

**Evidence:** `pkg/registry/registry.go` defines `Agent` struct with states: active, completed, abandoned, deleted. Stores in `~/.orch/agent-registry.json`.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/pkg/registry/registry.go:37-60`

**Significance:** Dashboard mirrors this data model in `src/lib/stores/agents.ts` for type-safe agent rendering.

---

## Synthesis

**Key Insights:**

1. **Embedded UI is the right approach** - Putting the dashboard in `web/` keeps deployment simple (single binary + static files) and maintains clear ownership.

2. **SvelteKit 5 with Runes** - Using modern Svelte 5 syntax (`$derived`, `$props`, `{@render}`) for future-proofing, though some shadcn components still use legacy `$$props` pattern.

3. **Static adapter for production** - Using `@sveltejs/adapter-static` to build to `build/` directory. Can be served by orch-go binary or nginx.

**Answer to Investigation Question:**

Created a complete SvelteKit 5 scaffold in `web/` with:
- Tailwind CSS v3 with shadcn-svelte color tokens
- Card, Badge, Button components from shadcn
- Agent store with derived views (active, completed, abandoned)
- SSE event store (last 100 events)
- Connection status tracking
- Swarm Map layout showing agent cards with status badges
- Vite proxy configuration for OpenCode SSE

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Build passes, type-checks clean, structure follows established patterns from beads-ui-svelte.

**What's certain:**

- ✅ Project scaffolding is complete and builds
- ✅ UI components render correctly (verified via mock data)
- ✅ Data stores follow Svelte 5 patterns

**What's uncertain:**

- ⚠️ SSE connection not yet wired up (placeholder only)
- ⚠️ Agent data loading from registry not implemented
- ⚠️ May need API endpoint in orch-go to serve registry data

**What would increase confidence to Very High:**

- Wire up actual SSE EventSource connection
- Add API endpoint to orch-go for registry data
- Test with live agents running

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Extend orch-go with HTTP API** - Add `/api/agents` endpoint to serve registry data, then consume from dashboard.

**Why this approach:**
- Single source of truth (registry)
- No file system access from browser
- Can add WebSocket for real-time updates later

**Trade-offs accepted:**
- Need to add net/http dependency to orch-go
- Slightly more complex than file watching

**Implementation sequence:**
1. Add HTTP server to orch-go (port 4097?)
2. Expose `/api/agents` endpoint returning registry JSON
3. Wire up dashboard fetch in `+page.svelte`
4. Connect SSE EventSource to `/api/events` proxy

### Alternative Approaches Considered

**Option B: Polling registry file directly**
- **Pros:** Simpler, no server changes
- **Cons:** Browser can't access local files, would need Tauri/Electron
- **When to use instead:** If building as native app

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/beads-ui-svelte/package.json` - Stack reference
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/opencode/sse.go` - SSE parsing
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/registry/registry.go` - Agent model

**Commands Run:**
```bash
# Check for bun
which bun && bun --version
# /opt/homebrew/bin/bun
# 1.3.3

# Install dependencies
cd web && bun install
# 141 packages installed

# Type check
bun run check
# svelte-check found 0 errors

# Build
bun run build
# ✓ built in 5.57s
```

**Related Artifacts:**
- **Workspace:** `.orch/workspace/og-work-scaffold-beads-ui-20dec/`

---

## Investigation History

**2025-12-20 17:35:** Investigation started
- Initial question: How to scaffold beads-ui v2 for Swarm Dashboard
- Context: Spawned from beads issue orch-go-an0

**2025-12-20 17:45:** Context gathered
- Found existing beads-ui-svelte project
- Analyzed orch-go SSE and registry code

**2025-12-20 17:55:** Design decision made
- Chose embedded `web/` approach over extending beads-ui-svelte

**2025-12-20 18:10:** Implementation complete
- Created project structure, installed deps
- Build and type-check passing

**2025-12-20 18:15:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Scaffolded SvelteKit 5 + shadcn-svelte dashboard in web/
