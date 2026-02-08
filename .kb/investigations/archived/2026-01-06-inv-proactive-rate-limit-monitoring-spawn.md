<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added proactive rate limit monitoring to spawn that warns at 80% usage and blocks at 95% usage unless auto-switch succeeds.

**Evidence:** Implementation tested via unit tests (TestDefaultUsageThresholds, TestUsageThresholdsFromEnv, TestAddUsageInfoToEventData) - all pass. Full test suite passes. Usage data now included in session.spawned telemetry events.

**Knowledge:** Proactive monitoring prevents spawn failures by catching rate limits BEFORE they cause mid-session crashes. The 80% warning + 95% block pattern allows graceful degradation with auto-switch as an escape hatch.

**Next:** Close this issue. Monitor telemetry for spawn.blocked.rate_limit and spawn.warning.rate_limit events to verify effectiveness.

---

# Investigation: Proactive Rate Limit Monitoring Spawn

**Question:** How to implement proactive rate limit monitoring that warns at 80% usage and blocks at 95% to prevent rate limit abandonments?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** Worker Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Existing auto-switch mechanism was reactive, not proactive

**Evidence:** The existing `checkAndAutoSwitchAccount()` function (spawn_cmd.go:484-547) only switched accounts when usage exceeded thresholds. It didn't warn users at lower thresholds or provide visibility into usage before spawn.

**Source:** `cmd/orch/spawn_cmd.go:484-547`

**Significance:** Users had no warning before hitting rate limits, leading to mid-session crashes and agent abandonments.

---

### Finding 2: Usage API provides 5-hour and weekly utilization

**Evidence:** The `account.CapacityInfo` struct provides `FiveHourUsed` and `SevenDayUsed` percentages. The tighter constraint (higher of the two) should determine blocking behavior.

**Source:** `pkg/account/account.go:428-447`

**Significance:** Both limits need to be checked - a session can be blocked by either the 5-hour or weekly limit.

---

### Finding 3: Telemetry needed usage data for pattern analysis

**Evidence:** The spawn telemetry events (session.spawned) didn't include usage information, making it impossible to correlate abandonments with rate limit conditions.

**Source:** `cmd/orch/spawn_cmd.go:1238-1261` (inline mode), `cmd/orch/spawn_cmd.go:1354-1379` (headless mode)

**Significance:** Adding usage data to telemetry enables analysis of rate limit abandonment patterns.

---

## Synthesis

**Key Insights:**

1. **Proactive vs Reactive** - The key shift is from "switch when hit limit" to "warn early, block before critical". This gives users time to prepare (add accounts, wait for reset) instead of failing mid-session.

2. **Two-tier thresholds** - 80% warning and 95% blocking provides a buffer. Users can continue working after warning but are blocked before they hit critical limits that cause crashes.

3. **Auto-switch as escape hatch** - At 95% block threshold, the system attempts auto-switch first. If that succeeds, spawn proceeds. Only blocks if no alternate account has sufficient headroom.

**Answer to Investigation Question:**

Implemented `checkUsageBeforeSpawn()` function that:
1. Fetches current account capacity via `account.GetCurrentCapacity()`
2. Calculates effective usage (max of 5-hour and weekly)
3. At 80%+: Shows warning but allows spawn
4. At 95%+: Attempts auto-switch, blocks only if switch fails
5. Logs telemetry events for pattern analysis

Also added `spawn.UsageInfo` to config and `addUsageInfoToEventData()` helper to include usage in all spawn telemetry.

---

## Structured Uncertainty

**What's tested:**

- ✅ Default threshold values (verified: TestDefaultUsageThresholds passes)
- ✅ Environment variable threshold overrides (verified: TestUsageThresholdsFromEnv passes)
- ✅ Telemetry helper adds correct fields (verified: TestAddUsageInfoToEventData passes)
- ✅ Full test suite passes (verified: `go test ./...`)

**What's untested:**

- ⚠️ Real rate-limit blocking behavior (needs live API testing)
- ⚠️ Auto-switch behavior at 95% threshold (needs multiple accounts configured)
- ⚠️ Effectiveness at reducing abandonments (needs production telemetry analysis)

**What would change this:**

- Finding would be wrong if Anthropic changes rate limit behavior or API
- Finding would be wrong if usage API becomes unreliable/slow (adds latency to spawn)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach (Implemented)

**Proactive monitoring with two-tier thresholds** - Warn at 80%, block at 95%

**Why this approach:**
- Provides early warning to users
- Auto-switch as escape hatch at critical threshold
- Configurable via environment variables for tuning
- Non-breaking - existing spawns work, just with added checks

**Trade-offs accepted:**
- Adds API call latency to every spawn (30 second timeout, usually <1s)
- May block legitimate spawns if single account is used heavily

**Implementation sequence:**
1. ✅ Added `UsageThresholds` struct and `DefaultUsageThresholds()`
2. ✅ Added `UsageCheckResult` to capture warning/blocking state
3. ✅ Implemented `checkUsageBeforeSpawn()` with warn/block logic
4. ✅ Added `tryAutoSwitchForSpawn()` for emergency account switching
5. ✅ Added `spawn.UsageInfo` and telemetry integration
6. ✅ Added unit tests for thresholds and telemetry

---

## References

**Files Modified:**
- `cmd/orch/spawn_cmd.go` - Added proactive usage monitoring functions
- `cmd/orch/main_test.go` - Added tests for usage thresholds and telemetry
- `pkg/spawn/config.go` - Added UsageInfo struct to Config

**Commands Run:**
```bash
# Build to verify compilation
go build ./...

# Run tests
go test ./...
go test ./cmd/orch/... -v -run "TestDefaultUsageThresholds|TestUsageThresholdsFromEnv|TestAddUsageInfoToEventData"
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-06-inv-rate-limit-account-switch-kills.md` - Prior work on account switch protection
- **Investigation:** `.kb/investigations/2026-01-06-inv-diagnose-overall-66-completion-rate.md` - Source of requirement (14-21% abandonments from rate limiting)

---

## Investigation History

**2026-01-06 17:51:** Investigation started
- Initial question: How to implement proactive rate limit monitoring?
- Context: Issue orch-go-jcc6k from completion rate analysis

**2026-01-06 18:00:** Code analysis complete
- Found existing checkAndAutoSwitchAccount() was reactive only
- Identified account.CapacityInfo as data source
- Identified telemetry gap in spawn events

**2026-01-06 18:30:** Implementation complete
- Added checkUsageBeforeSpawn() with warn/block logic
- Added UsageInfo to spawn config
- Added telemetry integration
- All tests pass

**2026-01-06 18:35:** Investigation completed
- Status: Complete
- Key outcome: Proactive rate limit monitoring now warns at 80% and blocks at 95%
