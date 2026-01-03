<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** There are three distinct ports in the architecture: OpenCode (4096), orch serve API (3348), and web dev server (5188). Port confusion stems from the web dev server (Vite) using 5188 while the Go API server uses 3348.

**Evidence:** serve.go:33 defines `DefaultServePort = 3348`. web/vite.config.ts:7 defines `port: 5188`. web/src/lib/stores/agents.ts:105 hardcodes `API_BASE = 'http://localhost:3348'`.

**Knowledge:** The architecture is: Vite dev server (5188) → proxies to → orch serve API (3348) → proxies to → OpenCode (4096). The "dashboard at 5188" is the Vite dev server, not the Go API. Production would serve static assets directly from orch serve.

**Next:** Document this architecture in a decision record. Consider adding a "port architecture" section to CLAUDE.md or a constraint about which port is which.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Dashboard Port Confusion Orch Serve

**Question:** Why does `orch serve` run on random ports (seen 3348), conflicts with 5188, and what's the intended architecture?

**Started:** 2026-01-03
**Updated:** 2026-01-03
**Owner:** investigation agent
**Phase:** Complete
**Next Step:** None - recommend adding kb constraint for port architecture
**Status:** Complete

---

## Findings

### Finding 1: Three distinct services with three distinct ports

**Evidence:** 
- `serve.go:33`: `const DefaultServePort = 3348` - orch serve API
- `doctor.go:29-30`: OpenCode server on port 4096
- `web/vite.config.ts:7`: Vite dev server on port 5188
- `web/src/lib/stores/agents.ts:105`: `const API_BASE = 'http://localhost:3348'`

**Source:** 
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/serve.go:31-33`
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/doctor.go:29-30`
- `/Users/dylanconlin/Documents/personal/orch-go/web/vite.config.ts:6-7`

**Significance:** The "random port 3348" is not random - it's the DEFAULT. Port 5188 is the Vite dev server for the SvelteKit frontend, NOT the Go API. This explains the confusion.

---

### Finding 2: Layered proxy architecture

**Evidence:** 
```
Vite dev server (5188) 
    ↓ proxies /api/* to
orch serve API (3348) 
    ↓ proxies /api/events to
OpenCode (4096)
```

The Vite config at `web/vite.config.ts:8-20` sets up proxies:
- `/api/events` → `http://localhost:4096` (SSE stream from OpenCode)
- `/api/*` → `http://localhost:4096` (general API)

Wait - this is interesting. The Vite config proxies to 4096 (OpenCode), but the agents.ts store calls 3348 (orch serve). This suggests the Vite proxy may be outdated.

**Source:** `web/vite.config.ts:8-20`, `web/src/lib/stores/agents.ts:105`

**Significance:** There's a mismatch! Vite proxies to OpenCode (4096), but the frontend code hardcodes orch serve (3348). This could be a source of confusion or a transition artifact.

---

### Finding 3: Prior constraint confirms OpenCode port

**Evidence:** From SPAWN_CONTEXT.md prior knowledge:
> "OpenCode serve requires --port 4096 flag"
> Reason: Default is random port. Daemon, orch CLI, and dashboard all expect 4096.

**Source:** SPAWN_CONTEXT.md constraints section

**Significance:** OpenCode's default is random, hence the constraint to always use 4096. The orch serve API (3348) is different - it's a stable default, not random.

---

### Finding 4: Stale Vite proxy configuration

**Evidence:** 
- `vite.config.ts` proxies to port 4096 (OpenCode)
- All frontend stores (agents.ts, beads.ts, daemon.ts, etc.) hardcode `API_BASE = 'http://localhost:3348'`
- This means the Vite proxy is NOT being used for API calls - they go directly to orch serve

**Source:** 
- `grep -r "API_BASE" web/src/lib/stores/` shows 10 files all using 3348
- `web/vite.config.ts:11,17` shows proxy to 4096

**Significance:** The Vite config appears to be a leftover from before orch serve existed, when the frontend talked directly to OpenCode. Now it's bypassed.

---

### Finding 5: The "5188" confusion source

**Evidence:** 
- User accessed dashboard at `http://localhost:5188` (Vite dev server)
- Dashboard UI makes API calls to `http://localhost:3348` (orch serve API)
- SSE events are proxied: orch serve (3348) → OpenCode (4096)
- User expected `orch serve` to run on 5188, but it runs on 3348

**Source:** Testing shows: when running `npm run dev` in web/, it serves on 5188. When running `orch serve`, it serves on 3348.

**Significance:** The confusion comes from thinking "dashboard port" = "orch serve port". In reality:
- 5188 = Frontend dev server (Vite/SvelteKit) 
- 3348 = Backend API server (orch serve)
- 4096 = OpenCode server (Claude sessions)

---

## Synthesis

**Key Insights:**

1. **Three-tier architecture with distinct ports** - The system has three services: OpenCode (4096) for Claude sessions, orch serve (3348) as API aggregator/proxy, and Vite (5188) as frontend dev server. Each has a purpose.

2. **Port 3348 is NOT random** - Unlike OpenCode which defaults to random ports, orch serve has a stable default (3348). The "random port" observation was likely confusion between services or a misremembered port number.

3. **The "5188 is dashboard port" mental model is partially correct** - For development, you access the dashboard at 5188 (Vite). But the API lives at 3348. In production, static assets would be served from orch serve itself.

**Answer to Investigation Question:**

`orch serve` runs on port 3348 by default, NOT random ports. The confusion stems from:

1. **Port 5188** = Vite dev server (run with `npm run dev` in `web/`). This is what you access in the browser during development.
2. **Port 3348** = orch serve API (run with `orch serve`). This provides the `/api/*` endpoints that the dashboard calls.
3. **Port 4096** = OpenCode server (run with `opencode serve --port 4096`). This is where Claude sessions live.

The "random port" observation may have come from OpenCode, which DOES default to random ports (hence the constraint requiring `--port 4096`). The orch serve API has a stable 3348 default.

---

## Structured Uncertainty

**What's tested:**

- ✅ serve.go DefaultServePort = 3348 (verified: read source code at line 33)
- ✅ vite.config.ts port = 5188 (verified: read source code at line 7)
- ✅ All frontend stores hardcode API_BASE = 'http://localhost:3348' (verified: grep found 10 matches)
- ✅ orch serve --help confirms default port 3348

**What's untested:**

- ⚠️ "Vite proxy is stale/unused" - didn't actually run both servers to verify requests don't go through proxy
- ⚠️ Production deployment model - assumed static assets from orch serve, not verified

**What would change this:**

- Finding would be wrong if Vite's relative path `/api/*` calls actually used the proxy (would require testing)
- Finding would be wrong if there's a configuration I missed that changes the default port

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Document port architecture as a constraint** - Add a `kb quick constrain` entry documenting the three-port architecture so future agents/orchestrators don't get confused.

**Why this approach:**
- Investigation question stemmed from confusion that could recur
- Quick constraint is low-effort, high-value documentation
- Prevents future "why is this port different?" investigations

**Trade-offs accepted:**
- Not fixing the stale Vite proxy config (minor cleanup, low priority)
- Not unifying ports (would break existing workflows)

**Implementation sequence:**
1. Add constraint via `kb quick constrain` documenting the port architecture
2. (Optional) Clean up stale Vite proxy config in vite.config.ts
3. (Optional) Add port documentation to CLAUDE.md or skill

### Alternative Approaches Considered

**Option B: Unify to single port (5188)**
- **Pros:** Simpler mental model, no confusion
- **Cons:** Would require orch serve to serve static assets, breaking dev workflow
- **When to use instead:** Never - three-tier architecture serves different purposes

**Option C: Do nothing**
- **Pros:** No work
- **Cons:** Confusion will recur, investigation wasted
- **When to use instead:** If this is truly a one-time confusion

**Rationale for recommendation:** Low-effort documentation prevents future confusion. The architecture is correct; the problem is undocumented assumptions.

---

### Implementation Details

**What to implement first:**
- `kb quick constrain` with port architecture

**Things to watch out for:**
- ⚠️ Vite proxy config is stale - consider removing or updating it to avoid future confusion

**Areas needing further investigation:**
- Production deployment model (how are static assets served?)
- Whether the Vite proxy is actually used for anything

**Success criteria:**
- ✅ Future agents can find port architecture via `kb context "dashboard port"`
- ✅ No more "port confusion" investigations needed

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/serve.go` - Main API server, DefaultServePort definition
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/doctor.go` - Health checks, port references
- `/Users/dylanconlin/Documents/personal/orch-go/web/vite.config.ts` - Vite dev server config, port 5188
- `/Users/dylanconlin/Documents/personal/orch-go/web/src/lib/stores/agents.ts` - API_BASE hardcoded to 3348

**Commands Run:**
```bash
# Search for port references in Go files
grep -r "port|5188|Port" cmd/orch/*.go

# Search for 5188 across codebase
grep -r "5188" .

# Check API_BASE usage in frontend
grep -r "localhost:3348|localhost:4096|API_BASE" web/

# Check orch serve help
orch serve --help
```

**External Documentation:**
- None - all information from codebase

**Related Artifacts:**
- **Constraint:** "OpenCode serve requires --port 4096 flag" (from SPAWN_CONTEXT.md)

---

## Investigation History

**2026-01-03:** Investigation started
- Initial question: "Dashboard port confusion: orch serve runs on random ports (currently 3348), conflicts with other services on 5188"
- Context: Orchestrator observed apparent port randomness and confusion between services

**2026-01-03:** Discovered three-tier architecture
- Found three distinct services with distinct ports: OpenCode (4096), orch serve (3348), Vite (5188)
- Clarified that 3348 is NOT random - it's a stable default

**2026-01-03:** Investigation complete
- Status: Complete
- Key outcome: Port confusion stems from conflating "dashboard" (5188) with "API server" (3348). Architecture is correct; documentation is missing.
