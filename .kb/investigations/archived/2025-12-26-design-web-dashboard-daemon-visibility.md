<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Dashboard needs daemon status visibility - currently shows only beads stats (46 ready), not daemon health (running/stopped, capacity 1/3, queue depth).

**Evidence:** Analyzed serve.go API endpoints, daemon.go worker pool, and +page.svelte stats bar. Daemon has PoolStatus() method ready for exposure; dashboard has established pattern for status indicators.

**Knowledge:** Minimal viable approach: add `/api/daemon` endpoint exposing pool status + launchd health check; display as single indicator in stats bar alongside existing beads/servers indicators.

**Next:** Implement `/api/daemon` endpoint and add stats bar indicator. Follow existing pattern from `/api/servers`.

**Confidence:** High (85%) - clear architecture pattern, one uncertainty is launchd health check reliability.

---

# Investigation: Web Dashboard Daemon Visibility

**Question:** Should the web dashboard display daemon status and queue visibility? What info is actionable, where should it go, and how to get the data?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** architect-agent
**Phase:** Complete
**Next Step:** None - ready for implementation
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: Daemon Already Has Observability Primitives

**Evidence:** The daemon package exposes `PoolStatus()` method (pool.go:181-210) that returns:
- `MaxWorkers` (capacity limit, e.g., 3)
- `Active` (currently running agents)
- `Available` (slots free)
- `ActiveSlots` (details of each running slot with BeadsID, Duration)

**Source:** 
- `pkg/daemon/pool.go:164-210` - PoolStatus struct and method
- `pkg/daemon/daemon.go:240-248` - PoolStatus() method on Daemon

**Significance:** The hard work is done. No new daemon logic needed - just expose existing primitives via API.

---

### Finding 2: Dashboard Has Established Stats Bar Pattern

**Evidence:** The stats bar already displays:
- Errors indicator (from agentlog events)
- Focus indicator (from `/api/focus`)
- Servers indicator (from `/api/servers`)
- Beads indicator (from `/api/beads`)

Pattern: Each indicator shows an emoji + key number + secondary context in tooltip.

**Source:**
- `web/src/routes/+page.svelte:294-376` - Stats bar implementation
- `cmd/orch/serve.go:1186-1280` - `/api/servers` as reference pattern

**Significance:** Adding daemon status follows established UI pattern. No new design paradigm needed.

---

### Finding 3: Daemon Runs via launchd with Logged Output

**Evidence:** Daemon runs as `com.orch.daemon` launchd service:
- PID visible via `launchctl list`
- Logs to `~/.orch/daemon.log`
- Config: `--poll-interval 60 --max-agents 3 --label triage:ready`
- Log shows capacity status: `[09:25:59] At capacity (3/3 agents active), waiting...`

**Source:**
- `~/Library/LaunchAgents/com.orch.daemon.plist`
- `~/.orch/daemon.log` (tail shows real-time status)

**Significance:** Can detect daemon running via process check. launchd provides restart-on-failure.

---

### Finding 4: Beads Stats Doesn't Show Actionable Processing State

**Evidence:** Current `/api/beads` returns:
```json
{
  "ready_issues": 49,
  "blocked_issues": 1,
  "in_progress_issues": 12
}
```
But this doesn't tell orchestrator:
- Is daemon running?
- Is it stalled (not processing despite ready work)?
- How full is the capacity?
- When was last spawn?

**Source:**
- `cmd/orch/serve.go:1121-1184` - handleBeads endpoint
- `web/src/lib/stores/beads.ts` - BeadsStats interface

**Significance:** Beads shows the queue size, but not the processor health. Both are needed for at-a-glance status.

---

## Synthesis

**Key Insights:**

1. **Separation of concerns** - Beads shows "what work exists" (queue depth). Daemon shows "is work being processed" (processor health). Both are needed for complete visibility.

2. **Existing primitives** - Daemon already has `PoolStatus()` ready to expose. No new daemon logic needed, just API plumbing.

3. **UI pattern established** - Stats bar indicators follow consistent pattern (emoji + number + tooltip). Daemon fits this pattern naturally.

**Answer to Investigation Question:**

Yes, the dashboard should display daemon status. The key info for orchestrator is:
- **Running/Stopped** - Is the daemon process alive?
- **Capacity** - Active/Max (e.g., "2/3 agents")
- **Stall detection** - Time since last spawn when queue has work

This should go in the stats bar as a single indicator following the established pattern. Data comes from new `/api/daemon` endpoint that:
1. Checks launchd process status
2. Reads pool status from a shared mechanism (file or API)

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**
Architecture is clear, patterns established, primitives exist. Main uncertainty is how to get pool status from running daemon to serve process.

**What's certain:**

- PoolStatus data structure exists and has all needed fields
- Stats bar pattern works for similar indicators
- Daemon runs via launchd and logs to known location

**What's uncertain:**

- How to get pool status from daemon process to serve process (file? shared memory? API on daemon?)
- Whether launchd process check is reliable enough
- Polling frequency for daemon status (60s like other stats? More frequent?)

**What would increase confidence to Very High:**

- Prototype `/api/daemon` endpoint to verify data flow
- Test launchd health check reliability
- User feedback on indicator design

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach: File-Based Status + launchd Check

**Add `/api/daemon` endpoint that combines:**
1. **launchd check** - Process alive detection via `launchctl list`
2. **Status file** - Daemon writes `~/.orch/daemon-status.json` on each cycle

**Why this approach:**
- Minimal daemon changes (just write status file)
- Works with existing serve.go patterns
- No shared memory or inter-process complexity
- Status file can include last-spawn timestamp for stall detection

**Trade-offs accepted:**
- Status is slightly stale (up to poll interval)
- Relies on file I/O (negligible overhead)
- Daemon must be modified to write status file

**Implementation sequence:**
1. Add status file writing to daemon (on each cycle)
2. Add `/api/daemon` endpoint reading status file + launchd check
3. Add stats bar indicator following servers pattern
4. Add store and polling in frontend

### Alternative Approaches Considered

**Option B: Daemon exposes its own HTTP API**
- **Pros:** Real-time data, no file I/O
- **Cons:** Another port, complexity, daemon must run HTTP server
- **When to use instead:** If status staleness becomes a problem

**Option C: Parse daemon.log for status**
- **Pros:** No daemon changes
- **Cons:** Fragile parsing, log rotation issues, heavier I/O
- **When to use instead:** If daemon can't be modified

**Rationale for recommendation:** File-based status with launchd check is simplest, follows existing patterns (events.jsonl, etc.), and provides all needed data with minimal complexity.

---

### Implementation Details

**What to implement first:**
1. Daemon: Write `~/.orch/daemon-status.json` with PoolStatus + metadata
2. Serve: Add `/api/daemon` endpoint
3. Frontend: Add daemon store and stats bar indicator

**Status file format:**
```json
{
  "running": true,
  "max_agents": 3,
  "active": 2,
  "available": 1,
  "last_spawn_at": "2025-12-26T09:22:04Z",
  "last_poll_at": "2025-12-26T09:26:04Z",
  "poll_interval_seconds": 60,
  "label": "triage:ready"
}
```

**API response format (similar to /api/servers):**
```json
{
  "running": true,
  "process_id": 99854,
  "capacity": { "active": 2, "max": 3, "available": 1 },
  "last_spawn": { "at": "...", "beads_id": "...", "skill": "..." },
  "last_poll_at": "...",
  "config": { "poll_interval": 60, "label": "triage:ready" },
  "stalled": false
}
```

**Stats bar indicator design:**
- Emoji: (daemon running) or (stopped) or (stalled)
- Primary: "2/3" (active/max)
- Tooltip: "Daemon: 2/3 agents, last spawn 3m ago"
- Color: green (healthy), yellow (at capacity), red (stopped/stalled)

**Things to watch out for:**
- Race condition on status file write/read (atomic write)
- Stall detection threshold (suggest 2x poll interval)
- launchd check may show PID but daemon could be stuck

**Areas needing further investigation:**
- Should stall detection consider blocked issues?
- What if daemon is rate-limited (not stalled, just waiting)?
- Should we show queue depth alongside capacity?

**Success criteria:**
- Orchestrator can see daemon health at a glance
- Distinguish between "no work" vs "daemon not running"
- Detect stalls (work available but no recent spawns)

---

## References

**Files Examined:**
- `pkg/daemon/daemon.go` - Daemon struct and methods
- `pkg/daemon/pool.go` - WorkerPool and PoolStatus
- `cmd/orch/daemon.go` - CLI daemon command
- `cmd/orch/serve.go` - API server implementation
- `web/src/routes/+page.svelte` - Stats bar UI
- `web/src/lib/stores/beads.ts` - Store pattern reference
- `~/Library/LaunchAgents/com.orch.daemon.plist` - launchd config
- `~/.orch/daemon.log` - Daemon output

**Commands Run:**
```bash
# Check launchd service status
launchctl list | grep orch

# View daemon logs
tail -30 ~/.orch/daemon.log

# Check API response
curl http://127.0.0.1:3348/api/beads

# Check running processes
ps aux | grep "orch daemon"
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2025-12-25-orchestrator-system-resource-visibility.md` - Related to system visibility decisions

---

## Investigation History

**2025-12-26 09:30:** Investigation started
- Initial question: Should dashboard display daemon status?
- Context: Dashboard shows "46 ready (1 blocked)" but not daemon health

**2025-12-26 09:45:** Code exploration complete
- Found PoolStatus() method ready for exposure
- Identified stats bar pattern for UI
- Discovered launchd service and log file

**2025-12-26 10:00:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Recommend file-based status + launchd check with stats bar indicator
