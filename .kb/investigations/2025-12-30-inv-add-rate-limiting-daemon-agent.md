<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added MaxSpawnsPerHour rate limiting to daemon to prevent runaway spawning when many issues are batch-labeled as triage:ready.

**Evidence:** Implementation complete with 25+ tests passing. RateLimiter tracks spawn history and enforces hourly limits.

**Knowledge:** Daemon already had MaxAgents for concurrent limits; new MaxSpawnsPerHour (default 20) adds throughput control. Both are configurable via ~/.orch/config.yaml.

**Next:** Close - implementation complete, ready for production use.

---

# Investigation: Add Rate Limiting Daemon Agent

**Question:** How to prevent daemon from spawning too many agents when batch-labeling issues as triage:ready?

**Started:** 2025-12-30
**Updated:** 2025-12-30
**Owner:** Agent (og-feat-add-rate-limiting-30dec)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Daemon already has MaxAgents for concurrent limit

**Evidence:** `pkg/daemon/daemon.go` has `MaxAgents` in Config (default 3) and uses WorkerPool for slot management.

**Source:** `pkg/daemon/daemon.go:27-28` and `pkg/daemon/pool.go`

**Significance:** Existing concurrency control works well. New rate limiting needs to complement it, not replace it.

---

### Finding 2: RateLimiter with sliding window is the right approach

**Evidence:** Created `RateLimiter` struct that tracks spawn timestamps and prunes entries older than 1 hour. This allows gradual recovery rather than abrupt resets.

**Source:** `pkg/daemon/daemon.go:91-177` - RateLimiter implementation

**Significance:** Sliding window prevents burst recovery scenarios where all capacity becomes available at once.

---

### Finding 3: User config supports daemon section

**Evidence:** Added `DaemonConfig` to `pkg/userconfig/userconfig.go` with `max_agents` and `max_spawns_per_hour` fields.

**Source:** `pkg/userconfig/userconfig.go:26-35`

**Significance:** Users can customize both limits via `~/.orch/config.yaml` without code changes.

---

## Synthesis

**Key Insights:**

1. **Two-layer rate limiting** - MaxAgents controls concurrency (how many running at once), MaxSpawnsPerHour controls throughput (how many started per hour). Both needed.

2. **Sliding window prunes memory** - SpawnHistory auto-prunes entries older than 1 hour, preventing unbounded growth during long daemon runs.

3. **Preview shows rate status** - `orch daemon preview` now shows rate limit status (e.g., "5/20 spawns in last hour") for visibility.

**Answer to Investigation Question:**

Rate limiting implemented via `MaxSpawnsPerHour` config option (default 20). Daemon checks limit before spawning, records successful spawns, and shows clear messages when limited.

---

## Structured Uncertainty

**What's tested:**

- ✅ RateLimiter.CanSpawn() returns false when at limit (unit test)
- ✅ RateLimiter.RecordSpawn() tracks spawns (unit test)
- ✅ RateLimiter.prune() removes old entries (unit test with mocked time)
- ✅ Daemon.Once() blocks when rate limited (unit test)
- ✅ Daemon.Preview() shows rate status (unit test)
- ✅ User config parsing works (unit test)

**What's untested:**

- ⚠️ Production behavior with real overnight runs
- ⚠️ Interaction with account switching when hitting Claude rate limits
- ⚠️ Edge case: daemon restart mid-hour loses spawn history

**What would change this:**

- Finding would be wrong if 20/hour proves too restrictive for legitimate batch work
- Implementation would need change if spawn history persistence across restarts is required

---

## Implementation Recommendations

Not applicable - implementation is complete.

---

## References

**Files Modified:**
- `pkg/daemon/daemon.go` - Added RateLimiter struct, updated Config, Once(), Preview()
- `pkg/daemon/daemon_test.go` - Added 25+ tests for rate limiting
- `pkg/userconfig/userconfig.go` - Added DaemonConfig with max_agents and max_spawns_per_hour
- `pkg/userconfig/userconfig_test.go` - Added tests for daemon config

**Commands Run:**
```bash
# Verify build
go build ./...

# Run tests
go test ./pkg/daemon/... ./pkg/userconfig/... -count=1
# EXIT_CODE=0
```

---

## Investigation History

**2025-12-30:** Investigation started
- Initial question: How to add rate limiting to daemon spawning?
- Context: 32+ concurrent agents and 193 commits in 2 days from aggressive batch-labeling

**2025-12-30:** Implementation complete
- Status: Complete
- Key outcome: Added MaxSpawnsPerHour with default 20, configurable via ~/.orch/config.yaml
