## Summary (D.E.K.N.)

**Delta:** Dashboard beads stats and ready queue now follow the orchestrator's tmux context via project_dir parameter.

**Evidence:** API returns project-specific beads - `/api/beads?project_dir=orch-go` returns 1581 issues, `/api/beads?project_dir=orch-knowledge` returns 0. Tests pass.

**Knowledge:** Cache must be per-project (keyed by directory) to support concurrent views of different projects.

**Next:** Close - implementation complete, tests passing, visual verification done.

**Promote to Decision:** recommend-no (tactical fix, not architectural)

---

# Investigation: Dashboard Beads Follow Orchestrator Tmux

**Question:** How to make dashboard beads display follow the orchestrator's current project context?

**Started:** 2026-01-07
**Updated:** 2026-01-07
**Owner:** og-feat-dashboard-beads-follow-07jan-ce85
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Beads API needed project_dir parameter

**Evidence:** The existing /api/beads endpoint had no way to specify which project's beads to query - it always used the server's sourceDir (orch-go).

**Source:** `cmd/orch/serve_beads.go:165-188` (handleBeads function)

**Significance:** This was the core issue - the API lacked the ability to query beads from different projects.

---

### Finding 2: Cache needed to be project-aware

**Evidence:** The existing beadsStatsCache used global fields (stats, statsFetchedAt) which could only hold one project's data at a time.

**Source:** `cmd/orch/serve_beads.go:14-36` (original beadsStatsCache struct)

**Significance:** Without per-project caching, switching projects would either bypass cache (slow) or return stale data from wrong project (incorrect).

---

### Finding 3: Frontend already had orchestrator context

**Evidence:** The context store (`web/src/lib/stores/context.ts`) already tracked `project_dir` from the orchestrator's tmux session via /api/context endpoint.

**Source:** `web/src/lib/stores/context.ts:7-13` (OrchestratorContext interface)

**Significance:** The infrastructure for following orchestrator context existed - just needed to wire it to beads fetch calls.

---

## Synthesis

**Key Insights:**

1. **Project-aware caching scales better** - Using a map keyed by project_dir allows concurrent users/views to see different projects without cache collisions.

2. **CLIClient works for cross-project queries** - For non-default projects, creating a CLIClient with WorkDir set allows querying any project's .beads/ directory.

3. **Reactive refetch pattern** - Frontend uses Svelte's reactive blocks to automatically refetch beads when orchestrator context changes.

**Answer to Investigation Question:**

The dashboard beads follow orchestrator context by: (1) adding project_dir query param to /api/beads and /api/beads/ready, (2) making the cache per-project, and (3) having the frontend pass project_dir from orchestrator context to beads fetch calls.

---

## Structured Uncertainty

**What's tested:**

- ✅ API returns project_dir in response (verified: curl with jq)
- ✅ Different projects return different counts (verified: orch-go=1581, orch-knowledge=0)
- ✅ Unit tests pass for project-aware endpoints (verified: go test)
- ✅ Dashboard loads with "Following" toggle enabled (verified: screenshot)

**What's untested:**

- ⚠️ Performance impact of per-project cache map growth (not benchmarked)
- ⚠️ Behavior when orchestrator switches projects rapidly (not load tested)
- ⚠️ Cross-project bd CLI works in all environments (fails in launchd due to PATH)

**What would change this:**

- If beads daemon supported cross-project queries via RPC, could avoid CLI fallback
- If multiple orchestrators were viewing dashboard simultaneously with different project focus

---

## Implementation Recommendations

### Recommended Approach ⭐

**Per-project cache with CLI fallback** - Cache beads data per project directory, use CLIClient with WorkDir for non-default projects.

**Why this approach:**
- Supports multiple concurrent project views
- No changes needed to beads daemon
- Leverages existing CLIClient infrastructure

**Trade-offs accepted:**
- CLI fallback may be slow/fail in launchd environments (known issue orch-go-loev7)
- Cache map can grow unbounded (mitigated: entries expire after TTL)

**Implementation sequence:**
1. Refactor cache to use per-project entries (map)
2. Update handlers to extract and pass project_dir
3. Update frontend to pass project_dir from context

---

## References

**Files Examined:**
- `cmd/orch/serve_beads.go` - Beads API handlers and cache
- `cmd/orch/serve_context.go` - Orchestrator context API
- `web/src/lib/stores/beads.ts` - Frontend beads store
- `web/src/lib/stores/context.ts` - Frontend context store
- `web/src/routes/+page.svelte` - Dashboard page

**Commands Run:**
```bash
# Test API with project_dir
curl -s -k "https://localhost:3348/api/beads?project_dir=/Users/dylanconlin/Documents/personal/orch-go" | jq .

# Run unit tests
go test -v ./cmd/orch -run "TestHandleBeads|TestBeadsStatsCache" -count=1
```

---

## Investigation History

**2026-01-07 23:40:** Investigation started
- Initial question: How to make dashboard beads follow orchestrator context?
- Context: Part of orch-go-tatzw design

**2026-01-08 00:05:** Investigation completed
- Status: Complete
- Key outcome: Implemented project-aware beads API and frontend integration
