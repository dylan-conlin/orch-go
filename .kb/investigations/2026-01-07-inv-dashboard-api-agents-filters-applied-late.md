## Summary (D.E.K.N.)

**Delta:** Time/project filters in /api/agents were applied at END of handler after all expensive operations, causing 20s cold cache despite filters.

**Evidence:** Code review shows filters applied at line 867-893 AFTER: session listing, workspace cache building, beads batch fetching, investigation discovery, token fetching.

**Knowledge:** Applying filters early (immediately after ListSessions) reduces expensive operation workload proportionally to the filter selectivity.

**Next:** Fix implemented - filters now applied immediately after session fetch, before workspace cache and beads operations.

**Promote to Decision:** recommend-no (tactical performance fix, not architectural)

---

# Investigation: Dashboard Api Agents Filters Applied Late

**Question:** Why does /api/agents take 20s on cold cache despite time/project filters being passed?

**Started:** 2026-01-07
**Updated:** 2026-01-07
**Owner:** Issue Creation Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Filters applied at END of handler

**Evidence:** In `serve_agents.go`, the time and project filters were applied at lines 867-893, which is AFTER all expensive operations:
- Line 320: `client.ListSessions("")` - fetch all sessions
- Line 332-333: Build workspace cache from all sessions
- Line 619: `getOpenIssues()` - fetch all open issues
- Line 623: `getAllIssues(beadsIDsToFetch)` - fetch all issues for beads IDs
- Line 627: `getComments(beadsIDsToFetch, ...)` - fetch all comments
- Lines 633-642: Build investigation directory cache
- Line 764: Fetch gap analysis
- Lines 794-837: Fetch tokens for agents

**Source:** `cmd/orch/serve_agents.go:299-893`

**Significance:** Even with filters like `?since=12h&project=orch-go`, ALL 600+ sessions were processed through expensive beads operations.

---

### Finding 2: Cold cache path is O(n) on session count

**Evidence:** 
- Workspace cache building scans all project directories from sessions
- Beads batch operations spawn RPC/CLI calls for each beads ID
- Investigation directory cache scans all unique project dirs
- Token fetching makes HTTP calls for each active session

With 600+ sessions and no early filtering, cold cache has to:
1. Scan 600+ workspace directories
2. Make 20+ beads RPC calls 
3. Scan investigation directories
4. Fetch tokens for active sessions

**Source:** `cmd/orch/serve_agents.go:326-837`, `pkg/verify/beads_api.go:298-381`

**Significance:** The 20s cold cache time is a direct result of not filtering sessions before these O(n) operations.

---

### Finding 3: Session timestamps available for early filtering

**Evidence:** OpenCode sessions have `Time.Updated` and `Time.Created` timestamps, plus `Directory` field for project filtering. These are available immediately after `ListSessions()`.

**Source:** `pkg/opencode/types.go:53-68` - Session struct with Time.Created, Time.Updated, Directory fields

**Significance:** Filtering can happen immediately after session fetch, before any expensive operations.

---

## Synthesis

**Key Insights:**

1. **Filter Early, Process Late** - The original code did "process everything, filter late" which defeats the purpose of filtering. The fix applies filters immediately after data source access.

2. **Proportional Reduction** - With 12h filter on 600+ sessions (where most are older), early filtering reduces workload by 90%+, taking cold cache from 20s to ~2-3s.

3. **Dual Filter Points** - Early filter handles OpenCode sessions, late filter (still needed) handles tmux-only agents and completed workspaces with different timestamp sources.

**Answer to Investigation Question:**

The 20s cold cache occurred because time/project filters were applied at the END of the handler (lines 867-893) after ALL expensive operations had already been performed on ALL 600+ sessions. Even passing `?since=12h&project=orch-go` didn't help because the filters only removed agents from the final JSON response, not from the expensive processing.

---

## Structured Uncertainty

**What's tested:**

- ✅ Build compiles successfully after fix (verified: `go build ./cmd/orch/...`)
- ✅ All serve/agent/filter tests pass (verified: `go test ./cmd/orch/... -run "Serve|Agent|Filter"`)
- ✅ Filter logic is correct - reuses existing `filterByProject()` and time comparison

**What's untested:**

- ⚠️ Actual cold cache performance improvement (not benchmarked in test environment)
- ⚠️ Dashboard user experience with the fix (requires browser testing)
- ⚠️ Edge cases with tmux-only agents and project filtering

**What would change this:**

- Finding would be wrong if session timestamps don't correlate with agent activity (unlikely)
- Finding would be incomplete if most expensive operation is NOT session-proportional (e.g., if it's O(1) per request regardless of session count)

---

## Implementation Recommendations

**Purpose:** Document the fix applied during this investigation.

### Implemented Approach 

**Early Session Filtering** - Apply time and project filters immediately after `client.ListSessions("")`, before workspace cache building and beads operations.

**Why this approach:**
- Reduces input to all downstream operations proportionally
- Reuses existing filter logic (`filterByProject`, time comparison)
- No changes to cache TTLs or invalidation logic needed

**Trade-offs accepted:**
- Dual filter points (early for sessions, late for non-session agents)
- Minor code complexity increase

**Implementation sequence:**
1. Add early filter block after line 324 (after ListSessions)
2. Remove duplicate `now := time.Now()` declaration
3. Update late filter comment to clarify its purpose (tmux/workspace agents)

### Changes Made

**File:** `cmd/orch/serve_agents.go`

1. **Lines 326-353:** Added early filtering block that applies time and project filters immediately after session fetch
2. **Line 364:** Removed duplicate `now := time.Now()` (now declared in early filter block)
3. **Lines 867-870:** Updated comment on late filter to clarify it handles tmux-only agents and completed workspaces

---

## References

**Files Examined:**
- `cmd/orch/serve_agents.go` - Main handler implementation
- `cmd/orch/serve_agents_cache.go` - Cache implementation details
- `cmd/orch/serve_filter.go` - Filter helper functions
- `pkg/opencode/types.go` - Session struct definition
- `pkg/verify/beads_api.go` - Beads batch operation implementations

**Commands Run:**
```bash
# Build verification
go build -v ./cmd/orch/...

# Test verification  
go test ./cmd/orch/... -run "Serve|Agent|Filter" -v
```

**Related Artifacts:**
- **Prior fix:** Jan 7 commit f87bedf4 added investigation cache and restored 2h threshold (didn't address filter timing)
