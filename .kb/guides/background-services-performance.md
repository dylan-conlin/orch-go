# Background Services Performance Guide

**Purpose:** Single authoritative reference for performance patterns in long-running background services (orch serve, orch daemon). This guide synthesizes learnings from 9 serve investigations conducted between Dec 2025 - Jan 2026.

**Last verified:** Jan 17, 2026

---

## Executive Summary

Background services face unique performance challenges:
1. Sustained load over hours/days compounds inefficiencies
2. Polling + SSE can create feedback loops
3. File I/O and process spawning don't scale linearly
4. Cache staleness creates UX issues

**Key insight:** O(1) operations that seem fast become O(n*m) disasters at scale. Always analyze: "What happens when there are 500+ agents/workspaces?"

---

## CPU Performance Anti-Patterns

### Anti-Pattern 1: SSE + Polling Feedback Loop

**Problem:** SSE events trigger API refetches that fetch the same state SSE provides.

**Evidence (Dec 25, 2025 investigation):**
```
125% CPU with 3 agents:
1. OpenCode sends session.status SSE events frequently
2. Each event triggers frontend to refetch /api/agents (debounced 100ms)
3. Each refetch causes backend to call IsSessionProcessing per session (HTTP call)
4. With 3 agents * high event frequency = hundreds of HTTP calls per minute
```

**Solution:**
- **Let SSE update local state for high-frequency fields** - Don't refetch what SSE provides
- **Increase debounce to 500ms** - 100ms is too short for rapid events
- **Remove redundant backend calls** - If frontend manages state via SSE, backend doesn't need to fetch it

**Code pattern:**
```javascript
// BAD: Refetch on every SSE event
sse.on('session.status', () => refetch('/api/agents'))

// GOOD: Update local state from SSE, refetch only for structural changes
sse.on('session.status', (data) => updateLocalState(data))
sse.on('session.created', () => refetch('/api/agents'))
sse.on('session.deleted', () => refetch('/api/agents'))
```

---

### Anti-Pattern 2: O(n*m) File Operations

**Problem:** Per-agent operations that scan all workspaces create multiplicative complexity.

**Evidence (Dec 25, 2025 investigation):**
```
124% CPU after 15 minutes:
- handleAgents() called findWorkspaceByBeadsID() for each agent with beads ID
- findWorkspaceByBeadsID() scanned all ~466 workspace directories
- For each directory, it read SPAWN_CONTEXT.md looking for matching beads ID
- With 10 agents: 10 * 466 = 4,660 file operations per request
- With 500ms SSE debounce: ~2 requests/second = 9,320 file operations/second
```

**Solution:** Per-request caching
- Build workspace cache once at the start of request
- Use O(1) map lookup for each agent
- Cache is discarded after request (no invalidation needed)

**Code pattern:**
```go
// BAD: O(n*m) - scan all workspaces for each agent
func handleAgents() {
    for _, agent := range agents {
        ws := findWorkspaceByBeadsID(agent.BeadsID) // Scans 500+ directories
    }
}

// GOOD: O(m) + O(n) - build cache once, lookup is O(1)
func handleAgents() {
    cache := buildWorkspaceCache() // O(m) - scan once
    for _, agent := range agents {
        ws := cache[agent.BeadsID]  // O(1) lookup
    }
}
```

---

### Anti-Pattern 3: Unbounded Process Spawning

**Problem:** Spawning external processes (bd, git, etc.) for each item creates exponential load.

**Evidence (Jan 3, 2026 investigation):**
```
90+ second response times:
- orch serve spawned bd processes for ALL workspaces
- With 618 workspaces, each request triggered bd commands for beads data
- No caching meant 20+ concurrent bd processes per request
- Dashboard polls every few seconds, compounding the problem
```

**Solution:** TTL-based caching with active filtering
- Cache process results with appropriate TTL (10-30s depending on freshness needs)
- Only fetch data for active items (e.g., agents updated in last 10 minutes)
- Batch operations when possible

**Code pattern:**
```go
// BAD: Spawn bd for every workspace on every request
func handleAgents() {
    for _, ws := range allWorkspaces { // 600+ workspaces
        comments := exec.Command("bd", "comments", ws.BeadsID).Output()
    }
}

// GOOD: TTL cache + active filtering
func handleAgents() {
    for _, ws := range activeWorkspaces { // Only ~10 recent ones
        comments, hit := cache.Get(ws.BeadsID)
        if !hit {
            comments = exec.Command("bd", "comments", ws.BeadsID).Output()
            cache.Set(ws.BeadsID, comments, 30*time.Second)
        }
    }
}
```

---

## Caching Patterns

### When to Use TTL Caching

| Data Type | TTL | Reason |
|-----------|-----|--------|
| Stats/counts | 30s | Coarse-grained, slight staleness OK |
| Issue list | 15s | Changes moderately, needs freshness |
| Agent status | 10s | Changes frequently during activity |
| Workspace metadata | 60s | Rarely changes once created |

### Cache Invalidation

**Problem (Jan 4, 2026 investigation):** TTL alone is insufficient for event-driven updates.

```
Dashboard showed "active" status 30 seconds after orch complete:
- orch complete ran as separate CLI process
- Server's in-memory cache held stale data
- TTL hadn't expired yet
```

**Solution:** Explicit invalidation API
- Add `POST /api/cache/invalidate` endpoint
- Call from CLI after state changes
- Silent failure is OK (TTL handles eventual consistency)

**Pattern:**
```go
// Server: Add invalidation endpoint
http.HandleFunc("/api/cache/invalidate", func(w http.ResponseWriter, r *http.Request) {
    beadsCache.Invalidate()
    workspaceCache.Invalidate()
    w.WriteHeader(http.StatusOK)
})

// CLI: Call invalidation after state change
func runComplete() {
    closeBeadsIssue()
    http.Post("http://localhost:3348/api/cache/invalidate", ...)
    // Silent failure OK - TTL will refresh
}
```

---

## Service Architecture

### Three-Tier Port Architecture

| Service | Port | Purpose | Lifecycle |
|---------|------|---------|-----------|
| OpenCode | 4096 | Claude sessions, SSE events | Persistent via launchd |
| orch serve | 3348 | API aggregator, dashboard backend | Persistent via overmind |
| Vite dev | 5188 | Frontend dev server | Ephemeral during development |

**Common confusion:** "Dashboard at 5188" ≠ "orch serve". The dashboard UI (5188) makes API calls to orch serve (3348), which proxies to OpenCode (4096).

**Data flow:**
```
Browser → Vite (5188) → static assets
Browser → orch serve (3348) → aggregated API
orch serve (3348) → OpenCode (4096) → SSE events
```

---

### launchd PATH Resolution

**Problem (Jan 7, 2026 investigation):** launchd provides minimal PATH (`/usr/bin:/bin:/usr/sbin:/sbin`). User shell paths aren't inherited.

**Symptoms:**
```
"exec: "bd": executable file not found in $PATH"
```

**Solution Options:**

| Option | Pros | Cons | Recommendation |
|--------|------|------|----------------|
| Configure PATH in plist | Works for specific service | External to code, platform-specific | Use for one-off |
| Resolve paths at startup | Self-contained, portable | Slight startup cost | **Recommended** |
| Use full paths everywhere | Simple | Hardcoded, inflexible | Avoid |

**Startup resolution pattern:**
```go
var bdPath string

func init() {
    // Try PATH first
    if path, err := exec.LookPath("bd"); err == nil {
        bdPath = path
        return
    }
    // Search common locations
    locations := []string{
        os.ExpandEnv("$HOME/bin/bd"),
        os.ExpandEnv("$HOME/go/bin/bd"),
        os.ExpandEnv("$HOME/.bun/bin/bd"),
    }
    for _, loc := range locations {
        if _, err := os.Stat(loc); err == nil {
            bdPath = loc
            return
        }
    }
    // Fall back to "bd" (will fail if not in PATH)
    bdPath = "bd"
}
```

---

### Separating Infrastructure from Project Services

**Insight (Dec 25, 2025 investigation):** `orch serve` is infrastructure (persistent monitoring), not a project dev server. Mixing these creates confusion.

| Aspect | orch serve | orch servers (per-project) |
|--------|-----------|---------------------------|
| **Purpose** | Dashboard API, agent monitoring | Project dev servers (Rails, Node, etc.) |
| **Lifecycle** | Persistent (launchd/overmind) | Ephemeral (tmux/tmuxinator) |
| **Port** | 3348 (stable) | Varies by project |
| **Scope** | Cross-project | Single project |
| **Status check** | `orch serve status` | `orch servers status` |

---

## Agent Status Determination

### Priority Cascade Model (Jan 4, 2026 investigation)

Multiple signals indicate agent completion. Check in order of authority:

| Priority | Signal | Source | Definitiveness |
|----------|--------|--------|----------------|
| 1 | Beads issue closed | `bd show <id>` status | **Definitive** - orchestrator verified |
| 2 | Phase: Complete comment | Beads comments | Agent self-report |
| 3 | SYNTHESIS.md exists | Workspace filesystem | Artifact presence |
| 4 | Session idle timeout | OpenCode session state | Inferential |

**Code pattern:**
```go
func determineAgentStatus(agent Agent, issue Issue) string {
    // Beads is source of truth
    if issue.Status == "closed" {
        return "completed"
    }
    // Then check Phase: Complete comment
    if hasPhaseCompleteComment(issue.ID) {
        return "completed"
    }
    // Then check SYNTHESIS.md
    if fileExists(agent.Workspace + "/SYNTHESIS.md") {
        return "completed"
    }
    // Fall back to session activity
    return sessionActivityStatus(agent.SessionID)
}
```

---

## Code Organization for Large Handler Files

### When to Extract (Jan 4, 2026 investigation)

Extract when a single file exceeds 1000 lines and contains distinct domains:

| Domain | Target File | Characteristics |
|--------|-------------|-----------------|
| Caching | `serve_agents_cache.go` | Structs with TTL, cache methods, builders |
| Events/SSE | `serve_agents_events.go` | SSE handlers, event streaming |
| Core handlers | `serve_agents.go` | HTTP handlers, business logic |

### Extraction Checklist

1. **Identify boundaries:** Group by single responsibility
2. **Check imports:** No circular dependencies between new files
3. **Keep handlers with handlers:** HTTP handlers stay together
4. **Keep primitives with primitives:** Cache structs and methods stay together
5. **Test after each extraction:** `go build && go test`

---

## Debugging Checklist

Before investigating performance issues:

1. **Check CPU:** `ps aux | grep orch`
2. **Profile endpoints:** Add `/debug/pprof/` for Go profiling
3. **Measure request latency:** `time curl http://localhost:3348/api/agents`
4. **Check process spawning:** `ps aux | grep bd | wc -l`
5. **Review cache hit rates:** Add cache metrics if not present
6. **Check event frequency:** SSE events per second

---

## Quick Reference: Performance Multipliers

| Factor | Impact | Mitigation |
|--------|--------|------------|
| # of workspaces | O(m) per scan | Per-request cache |
| # of agents | O(n) per operation | Active-only filtering |
| Event frequency | Multiplies API calls | Debounce (500ms+) |
| Polling interval | Multiplies everything | SSE where possible |
| Process spawning | 10-100ms per spawn | TTL caching |

---

## Key Decisions (Historical)

| Decision | Reason | Date |
|----------|--------|------|
| Per-request workspace caching | O(n*m) → O(m) + O(n) | Dec 2025 |
| 500ms debounce (not 100ms) | Rapid SSE events | Dec 2025 |
| TTL + explicit invalidation | Event-driven freshness | Jan 2026 |
| Startup path resolution | launchd minimal PATH | Jan 2026 |
| Beads as completion source of truth | Multiple signals, one authority | Jan 2026 |
| Active-agent filtering | 600 workspaces → ~10 active | Jan 2026 |
| Three-tier port separation | Avoid confusion | Jan 2026 |

---

## Synthesized From

This guide consolidates learnings from 9 investigations:
- 2025-12-25: orch serve 125% CPU (SSE + polling feedback loop)
- 2025-12-25: orch serve CPU runaway recurring (O(n*m) file operations)
- 2025-12-25: Separate orch serve status from orch servers
- 2026-01-03: orch serve causes CPU spike (bd process spawning)
- 2026-01-03: Dashboard port confusion (three-tier architecture)
- 2026-01-04: orch serve cache not invalidated (event-driven invalidation)
- 2026-01-04: orch serve shows closed agents (status determination)
- 2026-01-04: Analyze serve_agents.go (code extraction patterns)
- 2026-01-07: orch serve PATH issue (launchd executable resolution)

---

## Related Resources

- **daemon.md:** Autonomous agent spawning (different concerns)
- **resilient-infrastructure-patterns.md:** Crash recovery, escape hatches
- **opencode.md:** OpenCode server integration
- **dashboard.md:** Dashboard-specific patterns
