<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Auto-starting beads daemon in orch serve is unnecessary - caching already solves the cold-start penalty, and multiple bd daemons already run per-project automatically.

**Evidence:** Prior investigation shows caching reduces API response from 6.5s to 15ms. System already has 5+ bd daemons running. BEADS_NO_DAEMON=1 is set in orch daemon plist (intentional separation).

**Knowledge:** The 6.5s "cold start" is actually RPC reconnect overhead when daemon is down, not daemon absence. TTL caching (30s stats, 15s ready) is the correct fix, already implemented.

**Next:** Close - no implementation needed. Document Option 3 (health check warning) as potential enhancement if users report confusion.

**Promote to Decision:** recommend-no (tactical investigation, existing architecture is correct)

---

# Investigation: Consider Auto Starting Beads Daemon

**Question:** Should beads daemon auto-start when `orch serve` starts?

**Started:** 2026-01-07
**Updated:** 2026-01-07
**Owner:** Agent (og-feat-consider-auto-starting-07jan-eb8a)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Beads daemons already run per-project automatically

**Evidence:** Running `pgrep -fl "bd.*daemon"` shows 5 bd daemon processes:
```
5021 /Users/dylanconlin/.bun/bin/bd daemon --start --interval 5s
11669 /Users/dylanconlin/.bun/bin/bd daemon --start --interval 5s
67584 /Users/dylanconlin/.bun/bin/bd daemon --start --interval 5s
67705 /Users/dylanconlin/.bun/bin/bd daemon --start --interval 5s
94613 /Users/dylanconlin/.bun/bin/bd daemon --start --interval 5s
```

**Source:** `pgrep -fl "bd.*daemon"` command output

**Significance:** The beads daemon runs per-project (one per .beads/ directory), not as a single global daemon. Auto-starting one daemon in orch serve wouldn't help projects in other directories.

---

### Finding 2: The cold-start penalty is RPC reconnect overhead, not daemon absence

**Evidence:** From prior investigation `.kb/investigations/2026-01-07-inv-api-beads-endpoint-takes-5s.md`:
- First API request: 6.5s
- Second (cached) request: 15ms
- Direct CLI `bd stats --json`: ~1.5s

The 6.5s latency came from RPC client auto-reconnect logic (3 retries with exponential backoff) when daemon socket doesn't exist, NOT from lack of caching.

**Source:** `cmd/orch/serve_beads.go:55-74` shows socket existence check optimization, `.kb/investigations/2026-01-07-inv-api-beads-endpoint-takes-5s.md` Finding 2

**Significance:** The fix is caching (already implemented with 30s TTL for stats, 15s TTL for ready), not daemon management. The socket existence check short-circuits failed RPC attempts.

---

### Finding 3: BEADS_NO_DAEMON=1 is intentionally set in orch daemon's launchd config

**Evidence:** The orch daemon plist (`~/Library/LaunchAgents/com.orch.daemon.plist`) includes:
```xml
<key>BEADS_NO_DAEMON</key>
<string>1</string>
```

**Source:** `~/Library/LaunchAgents/com.orch.daemon.plist` environment variables section

**Significance:** This indicates an intentional architectural decision to keep orch daemon and beads daemon separate. The orch daemon uses direct bd CLI calls (`--no-daemon` mode) rather than RPC, avoiding daemon coordination complexity.

---

### Finding 4: orch serve has no launchd PATH context for daemon management

**Evidence:** The orch-go.serve launchd plist is minimal:
```xml
<key>ProgramArguments</key>
<array>
    <string>/Users/dylanconlin/Documents/personal/orch-go/build/orch</string>
    <string>serve</string>
</array>
```
No PATH environment is set, which is why `beads.ResolveBdPath()` was added at startup.

**Source:** `~/Library/LaunchAgents/com.orch-go.serve.plist`

**Significance:** Adding daemon management to orch serve would require PATH configuration in the plist, adding complexity for marginal benefit.

---

## Synthesis

**Key Insights:**

1. **Caching is the correct solution** - The implemented TTL-based cache (`beadsStatsCache` with 30s/15s TTLs) already eliminates cold-start penalty for dashboard users. First request may be slow, but subsequent requests hit cache.

2. **Beads daemons are per-project** - There's no single "beads daemon" to start. Each project with `.beads/` runs its own daemon. Auto-starting one from orch serve wouldn't cover all projects.

3. **Separation of concerns is intentional** - `BEADS_NO_DAEMON=1` in orch daemon shows architectural intent: orch components should use CLI fallback, not depend on bd RPC daemon availability.

**Answer to Investigation Question:**

**No, orch serve should NOT auto-start beads daemon.** The 6.5s cold-start penalty is already solved by the caching implementation (TTL-based cache reduces subsequent requests to ~15ms). Auto-starting a beads daemon would:
- Only help one project (the one orch serve started from)
- Add process management complexity
- Duplicate work if bd daemon already running for that project
- Fight against intentional architecture (BEADS_NO_DAEMON=1 pattern)

The existing architecture is correct: orch serve uses RPC when available, falls back to CLI, and caches aggressively.

---

## Structured Uncertainty

**What's tested:**

- ✅ Multiple bd daemons are running (verified via pgrep)
- ✅ Caching reduces response time from 6.5s to 15ms (verified in prior investigation)
- ✅ Socket existence check short-circuits slow RPC attempts (verified in code)

**What's untested:**

- ⚠️ Dashboard UX when user first loads page after server start (first request still slow)
- ⚠️ Whether a "prewarming" call at startup would help (could call cache methods on init)

**What would change this:**

- Finding would be wrong if caching was ineffective (but prior investigation confirms ~450x improvement)
- Decision would change if there was a single global beads daemon (there isn't - it's per-project)

---

## Implementation Recommendations

**Purpose:** This investigation concludes with "no implementation needed" - the existing architecture is correct.

### Recommended Approach: Do Nothing (Option 2 in SPAWN_CONTEXT)

**Document that bd daemon should be running for best performance** is implicitly already the case. No explicit documentation needed because:

1. Caching already handles the performance gap
2. bd daemons auto-start per-project when bd commands are run
3. Dashboard users see fast responses after first poll

**Why this approach:**
- Zero implementation effort
- No additional process management complexity
- Existing system works correctly

**Trade-offs accepted:**
- First dashboard load may be slow (6.5s) if cache is cold
- Users must run bd commands in a project to start daemon (happens naturally)

### Alternative Approaches Considered

**Option A: Auto-start beads daemon in orch serve**
- **Pros:** Guaranteed daemon availability for one project
- **Cons:** Per-project architecture means this only helps one project; adds process management; conflicts with BEADS_NO_DAEMON=1 pattern
- **When to use instead:** Only if there was a single global beads daemon (there isn't)

**Option C: Health check that warns if daemon not running**
- **Pros:** Visibility into daemon status; helps debug slow API responses
- **Cons:** Additional complexity; may be confusing (which project's daemon?)
- **When to use instead:** If users report confusion about slow first-load performance

**Rationale for recommendation:** The caching implementation already solves the user-facing problem (dashboard performance). Adding daemon management would add complexity without benefit.

---

### Potential Enhancement: Cache Prewarming

If first-load performance is still a concern, a simple enhancement could prewarm caches at startup:

```go
// In runServe(), after cache initialization:
go func() {
    // Prewarm stats cache
    globalBeadsStatsCache.getStats()
    // Prewarm ready cache
    globalBeadsStatsCache.getReadyIssues()
}()
```

This is low-effort and would eliminate even the first-request latency. Not implementing now because:
- Prior investigation didn't flag this as a problem
- Adds 1-2s to startup time in background
- May fail silently if bd not available

---

## References

**Files Examined:**
- `cmd/orch/serve.go:157-196` - Server initialization, beads client setup
- `cmd/orch/serve_beads.go:1-142` - Cache implementation, handlers
- `~/Library/LaunchAgents/com.orch.daemon.plist` - BEADS_NO_DAEMON=1 pattern
- `~/Library/LaunchAgents/com.orch-go.serve.plist` - Minimal launchd config

**Commands Run:**
```bash
# Check for running bd daemons
pgrep -fl "bd.*daemon"

# List launchd agents
ls -la ~/Library/LaunchAgents/ | grep -E "(orch|beads)"

# Check bd daemon help
bd daemon --help
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-07-inv-api-beads-endpoint-takes-5s.md` - Prior investigation that implemented caching fix

---

## Investigation History

**2026-01-07 12:00:** Investigation started
- Initial question: Should beads daemon auto-start with orch serve?
- Context: Prior investigation found 6.5s cold-start penalty for /api/beads endpoint

**2026-01-07 12:15:** Architecture analyzed
- Found multiple bd daemons running per-project
- Found BEADS_NO_DAEMON=1 intentional separation
- Found caching already implemented as the fix

**2026-01-07 12:25:** Investigation completed
- Status: Complete
- Key outcome: No implementation needed - caching solves the problem, daemon architecture is intentionally per-project
