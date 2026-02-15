## Summary (D.E.K.N.)

**Delta:** Implemented auto-account-switching that checks usage before spawn and switches to alternate account if headroom is better.

**Evidence:** Build passes, all tests pass, binary installed and functional.

**Knowledge:** Auto-switch logic uses 5-hour >80% OR weekly >90% thresholds by default, configurable via env vars. MinHeadroomDelta (10%) prevents unnecessary switching.

**Next:** Deploy and monitor in production usage.

**Confidence:** High (90%) - Integration tested, but production verification pending.

---

# Investigation: Implement Auto Account Switching Before Spawn

**Question:** How to implement automatic account switching based on usage thresholds before spawning agents?

**Started:** 2025-12-24
**Updated:** 2025-12-24
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Existing Capacity Infrastructure

**Evidence:** The `pkg/account` package already has `GetCurrentCapacity()`, `GetAccountCapacity(name)`, `SwitchAccount(name)`, and `FindBestAccount()` functions. The capacity manager in `pkg/capacity` provides concurrent slot management.

**Source:** `pkg/account/account.go:481-738`

**Significance:** Can build auto-switch on top of existing infrastructure rather than rebuilding capacity tracking.

---

### Finding 2: Spawn Flow Integration Point

**Evidence:** The `runSpawnWithSkill()` function in `cmd/orch/main.go` already has a `checkConcurrencyLimit()` call before spawn. This is the ideal location to add auto-switch since it runs early in the spawn flow.

**Source:** `cmd/orch/main.go:1019-1030`

**Significance:** Adding auto-switch after concurrency check ensures we switch accounts before any heavy operations, and the error handling pattern is already established.

---

### Finding 3: Configuration via Environment Variables

**Evidence:** Other spawn-related configuration (ORCH_MAX_AGENTS, ORCH_WORKER) uses environment variables for threshold configuration. This pattern avoids modifying config files and allows per-invocation overrides.

**Source:** `cmd/orch/main.go:842-851` (ORCH_MAX_AGENTS handling)

**Significance:** Environment variable configuration is consistent with existing patterns and supports flexible override without file modification.

---

## Synthesis

**Key Insights:**

1. **Minimal Invasive Integration** - Added auto-switch as a single function call in spawn flow, leveraging existing capacity infrastructure.

2. **Conservative Defaults** - 5-hour >80% and weekly >90% thresholds with 10% minimum headroom delta prevent thrashing between accounts.

3. **Fail-Open Design** - Auto-switch failures are logged but don't block spawn, maintaining system availability.

**Answer to Investigation Question:**

Implemented auto-switch by:
1. Adding `ShouldAutoSwitch()` and `AutoSwitchIfNeeded()` to `pkg/account/account.go`
2. Adding `checkAndAutoSwitchAccount()` in `cmd/orch/main.go` that calls `AutoSwitchIfNeeded()`
3. Integrating into spawn flow after concurrency check

Configuration via environment variables:
- `ORCH_AUTO_SWITCH_5H_THRESHOLD` (default 80)
- `ORCH_AUTO_SWITCH_WEEKLY_THRESHOLD` (default 90)
- `ORCH_AUTO_SWITCH_MIN_DELTA` (default 10)
- `ORCH_AUTO_SWITCH_DISABLED=1` to disable

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Strong evidence from build passing, all tests passing, and code review. Main uncertainty is production behavior with real API calls under load.

**What's certain:**

- Code compiles and tests pass
- Integration point is correct (after concurrency limit check)
- Configuration mechanism works (env vars)
- Event logging is in place

**What's uncertain:**

- Real-world API latency impact
- Token refresh timing under high load
- Edge cases with expired/invalid tokens

**What would increase confidence to Very High (95%+):**

- Production usage with multiple accounts
- Verification of actual account switches logged in events
- Load testing with high spawn rate

---

## Implementation Recommendations

### Recommended Approach (Implemented)

**Environment Variable Configuration** - Thresholds configurable via env vars, defaults to 80% 5-hour, 90% weekly.

**Why this approach:**
- Consistent with existing ORCH_MAX_AGENTS pattern
- No config file modification required
- Easy to override per-invocation

**Trade-offs accepted:**
- Less discoverable than config file
- No persistence of custom thresholds (must set each session)

**Implementation sequence:**
1. Added types and functions to pkg/account/account.go
2. Added checkAndAutoSwitchAccount() to cmd/orch/main.go
3. Integrated into spawn flow after concurrency check
4. Added tests for new functionality

---

## References

**Files Examined:**
- `pkg/account/account.go` - Core capacity and switch functions
- `pkg/capacity/manager.go` - Capacity manager patterns
- `cmd/orch/main.go` - Spawn flow integration

**Commands Run:**
```bash
# Build verification
go build ./...

# Test verification
go test ./...

# Binary installation
make install
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-20-inv-add-usage-capacity-tracking-account.md` - Prior capacity work

---

## Investigation History

**2025-12-24 17:00:** Investigation started
- Initial question: How to implement auto-switch before spawn?
- Context: Task from SPAWN_CONTEXT.md

**2025-12-24 17:10:** Implementation complete
- Added AutoSwitchThresholds, ShouldAutoSwitch, AutoSwitchIfNeeded to account package
- Added checkAndAutoSwitchAccount to main.go
- Integrated into spawn flow

**2025-12-24 17:11:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Auto-switch implemented with configurable thresholds
