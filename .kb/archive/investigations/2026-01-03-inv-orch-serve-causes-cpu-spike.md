## Summary (D.E.K.N.)

**Delta:** orch serve was spawning 20+ concurrent bd processes per /api/agents request because it fetched beads data for all 600+ workspaces without caching.

**Evidence:** CPU spiked during dashboard polling; response times exceeded 90 seconds; `ps aux | grep bd` showed 20+ concurrent processes.

**Knowledge:** TTL-based caching plus limiting beads fetches to active agents only reduces bd processes to 1-2 and response time to 0.13s (cached).

**Next:** Close issue - fix implemented and verified.

---

# Investigation: Orch Serve Causes CPU Spike

**Question:** Why does orch serve cause CPU spike and how can we fix it?

**Started:** 2026-01-03
**Updated:** 2026-01-04
**Owner:** Worker agent (og-debug-orch-serve-causes-03jan)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Every /api/agents request spawned bd processes for ALL workspaces

**Evidence:** With 618 workspaces, each request to /api/agents triggered:
- `bd comments` for each workspace with a beads issue
- `bd show` for each workspace with a beads issue
- Reading SPAWN_CONTEXT.md for each workspace

**Source:** `cmd/orch/serve_agents.go:handleAgents()` - iterates all workspaces from `collectWorkspaces()`

**Significance:** O(n) bd process spawning per request where n=600+ is fundamentally unscalable.

---

### Finding 2: No caching existed for beads or workspace data

**Evidence:** Each HTTP request to /api/agents resulted in fresh data fetching:
- `beadsCache` struct did not exist
- `globalWorkspaceCache` did not exist
- Dashboard polls every few seconds, compounding the problem

**Source:** `cmd/orch/serve_agents.go` - no cache structures before fix

**Significance:** Repeated requests caused repeated process spawning with no reuse.

---

### Finding 3: Beads data was fetched for idle/completed agents unnecessarily

**Evidence:** Even agents that hadn't been updated in days (completed workspaces) triggered bd process spawning on every request.

**Source:** `enrichAgentWithBeads()` had no check for agent activity status

**Significance:** Historical workspaces (500+ of 618) caused most of the load despite being irrelevant.

---

## Synthesis

**Key Insights:**

1. **Cache eliminates redundant bd spawning** - With TTL-based caching (10-30s depending on data type), subsequent requests within the TTL window return cached data with zero bd processes.

2. **Active-only filtering reduces scope** - Only fetching beads for agents updated in the last 10 minutes reduces the effective workspace count from 600+ to typically <10.

3. **Workspace metadata caching compounds benefits** - SPAWN_CONTEXT.md parsing is also expensive; caching this adds another layer of efficiency.

**Answer to Investigation Question:**

The CPU spike was caused by spawning 20+ concurrent bd processes for every /api/agents request, iterating all 618 workspaces without caching. The fix implements TTL-based caching for beads data and workspace metadata, plus limits beads fetching to active agents only. Response time dropped from 90+ seconds to 0.13s (cached).

---

## Structured Uncertainty

**What's tested:**

- ✅ Response time drops to 0.13s on cached requests (verified via curl timing)
- ✅ bd process count drops to 1-2 during polling (verified via `ps aux | grep bd`)
- ✅ Tests pass with cache initialization (verified via `go test ./cmd/orch/...`)

**What's untested:**

- ⚠️ Cache invalidation under edge cases (manual bd operations during cached window)
- ⚠️ Memory usage with large cached data sets over time
- ⚠️ Behavior when beads daemon is unavailable

**What would change this:**

- Finding would be wrong if bd spawning occurs from another code path not covered by caching
- Fix might be insufficient if workspace count grows significantly beyond 1000

---

## Implementation Recommendations

### Recommended Approach ⭐

**TTL-based caching with active-agent filtering** - Add in-memory TTL caches for beads data and workspace metadata, plus skip beads fetching for inactive agents.

**Why this approach:**
- Minimal code changes (local to serve_agents.go)
- No external dependencies (no Redis/memcached)
- Immediate impact on existing load

**Trade-offs accepted:**
- Stale data possible within TTL window (acceptable for dashboard)
- Memory usage for cache (negligible for expected data sizes)

**Implementation sequence:**
1. Add beadsCache with TTL for issues and comments
2. Add globalWorkspaceCache for SPAWN_CONTEXT.md data
3. Add isAgentActive() check to skip idle agents
4. Initialize caches in runServe() and tests

### Alternative Approaches Considered

**Option B: Batch bd queries**
- **Pros:** Single bd process for all data
- **Cons:** Requires bd CLI changes; still no caching
- **When to use instead:** If bd adds native batch support

**Option C: Beads daemon RPC**
- **Pros:** Eliminates CLI spawning entirely
- **Cons:** Requires beads daemon development; more complex
- **When to use instead:** If bd daemon becomes standard

**Rationale for recommendation:** Caching solves the immediate problem with minimal changes and no external dependencies.

---

## References

**Files Examined:**
- `cmd/orch/serve_agents.go` - Main handler for /api/agents
- `cmd/orch/serve.go` - Server initialization
- `pkg/beads/client.go` - Beads CLI spawning

**Commands Run:**
```bash
# Verify response time
time curl -s localhost:5188/api/agents | jq '.agents | length'

# Check bd process count
ps aux | grep bd | wc -l

# Run tests
go test ./cmd/orch/... -run TestHandleAgents
```

**Related Artifacts:**
- **Workspace:** `.orch/workspace/og-debug-orch-serve-causes-03jan/`

---

## Investigation History

**2026-01-03 23:56:** Investigation started
- Initial question: Why does orch serve cause CPU spike?
- Context: Dashboard polling caused system slowdown

**2026-01-04 07:57:** Root cause identified
- bd processes spawned for all 618 workspaces per request
- No caching mechanism existed

**2026-01-04 08:30:** Fix implemented
- TTL caches added for beads and workspace data
- Active-agent filtering added

**2026-01-04:** Investigation completed
- Status: Complete
- Key outcome: Response time reduced from 90+s to 0.13s with caching
