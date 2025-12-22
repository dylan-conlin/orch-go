<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** orch status showed 339+ stale sessions as active because OpenCode API returns all persisted sessions and liveness checks used existence rather than activity time.

**Evidence:** GET /session returns 339 sessions; only 5-7 were updated in last 30 min; SessionExists() returned true for all persisted sessions.

**Knowledge:** OpenCode persists ALL sessions to disk; liveness detection must check activity time (updated within 30 min), not just whether session exists in API.

**Next:** Issue closed - fix implemented and committed (9606f43).

**Confidence:** Very High (95%) - Fix tested, all tests passing, smoke test confirms 339→7 reduction.

---

# Investigation: orch status stale sessions fix

**Question:** Why does orch status show 339+ stale OpenCode sessions as active?

**Started:** 2025-12-22
**Updated:** 2025-12-22
**Owner:** Agent og-debug-orch-status-shows-22dec
**Phase:** Complete
**Next Step:** None (fix implemented)
**Status:** Complete
**Confidence:** Very High (95%)

---

## Findings

### Finding 1: OpenCode API returns ALL persisted sessions

**Evidence:** `curl http://127.0.0.1:4096/session | jq 'length'` returns 339, but only 5-7 were updated within 30 minutes.

**Source:** Direct API call, confirmed via `jq '[.[] | select(.time.updated > (now * 1000 - 1800000))] | length'` returning 5-7.

**Significance:** The status command was iterating over 339 sessions when only ~5 were actually active.

---

### Finding 2: SessionExists() returns true for any persisted session

**Evidence:** `pkg/opencode/client.go:229-240` - SessionExists() calls GET /session/{id} and returns true if status 200. This returns true for ANY session that exists on disk, not just actively running ones.

**Source:** Code review of client.go

**Significance:** Liveness checks using SessionExists() incorrectly reported stale sessions as "live".

---

### Finding 3: Prior investigation identified root cause correctly

**Evidence:** `.kb/investigations/2025-12-21-inv-investigate-orch-status-showing-stale.md` documented the four-layer architecture (OpenCode memory, disk, registry, tmux) and recommended activity-based liveness.

**Source:** Prior investigation file

**Significance:** Solution was straightforward implementation of prior recommendation.

---

## Synthesis

**Key Insights:**

1. **Activity time is the correct liveness signal** - OpenCode persists sessions forever; checking "exists" is meaningless. Must check "updated recently".

2. **30 minute threshold matches existing behavior** - The code already used 30 min for sessions without beads IDs. Made this universal.

3. **Early filtering prevents wasted work** - Adding filter before GetLiveness() calls prevents 339 API calls.

**Answer to Investigation Question:**

Sessions appeared active because SessionExists() returned true for all 339 persisted sessions. Fix: Add IsSessionActive() method that checks updated time, use 30 min max idle threshold, filter early in runStatus().

---

## Confidence Assessment

**Current Confidence:** Very High (95%)

**Why this level?**

Fix is implemented, all tests pass, smoke test confirms dramatic reduction (339→7).

**What's certain:**

- ✅ OpenCode returns all persisted sessions via GET /session (tested: 339 returned)
- ✅ Activity filter correctly reduces to ~5-7 active sessions (tested)
- ✅ All existing tests still pass (tested: go test ./...)

**What's uncertain:**

- ⚠️ Edge case: agent actively working but idle for >30 min (rare, acceptable false negative)

---

## Implementation (Completed)

**Changes made:**

1. `pkg/opencode/client.go` - Added `IsSessionActive(sessionID, maxIdleTime)` method
2. `pkg/state/reconcile.go` - Updated `checkOpenCodeSession()` to use activity time (30 min)
3. `cmd/orch/main.go` - Added early activity filter in `runStatus()`

**Commit:** `9606f43` - fix: filter stale OpenCode sessions by activity time in orch status

---

## References

**Files Modified:**
- `pkg/opencode/client.go:229-253` - Added IsSessionActive()
- `pkg/state/reconcile.go:102-147` - Activity-based liveness
- `cmd/orch/main.go:1713-1750` - Early filtering

**Related Artifacts:**
- **Prior Investigation:** `.kb/investigations/2025-12-21-inv-investigate-orch-status-showing-stale.md`

---

## Investigation History

**2025-12-22 19:30:** Investigation started
- Initial question: Why does orch status show 339+ stale sessions?
- Context: Spawned to implement fix from prior investigation

**2025-12-22 19:45:** Fix implemented
- Added activity-based liveness detection
- All tests passing

**2025-12-22 19:55:** Investigation completed
- Final confidence: Very High (95%)
- Status: Complete
- Key outcome: Reduced active sessions from 339+ to ~7 actual active agents
