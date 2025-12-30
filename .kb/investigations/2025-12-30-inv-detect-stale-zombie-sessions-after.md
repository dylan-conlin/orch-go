<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Fixed `IsSessionProcessing()` to return false for stale sessions (no activity in 3 minutes), even if they have incomplete assistant messages with pending tool calls.

**Evidence:** Tests pass including new test case "idle - stale session even with incomplete assistant message". Dashboard correctly shows dead agents without `is_processing: true`.

**Knowledge:** Zombie sessions are sessions killed mid-execution that still have `tool.state.status: "pending"`. The fix adds a staleness check (3-minute threshold) before checking message completion status.

**Next:** Complete. Changes committed and tested.

---

# Investigation: Detect Stale/Zombie Sessions After OpenCode Restart

**Question:** Why do zombie sessions (killed mid-execution) still show `is_processing: true` after OpenCode restart?

**Started:** 2025-12-30
**Updated:** 2025-12-30
**Owner:** Agent og-debug-detect-stale-zombie-30dec
**Phase:** Complete
**Next Step:** None - fix implemented and verified
**Status:** Complete

---

## Findings

### Finding 1: IsSessionProcessing only checks message completion, not session activity

**Evidence:** The original `IsSessionProcessing()` function in `pkg/opencode/client.go` only checked:
1. If the last message is from assistant
2. If `lastMsg.Info.Finish == ""` and `lastMsg.Info.Time.Completed == 0`

It did not check when the session was last updated.

**Source:** `pkg/opencode/client.go:386-412` (before fix)

**Significance:** Zombie sessions (killed mid-tool-execution) would return `true` for `IsSessionProcessing()` because they have incomplete assistant messages, even though the session is dead and will never complete.

---

### Finding 2: Dashboard already marks sessions as "dead" based on activity time

**Evidence:** `serve.go` has a 3-minute `deadThreshold` and marks sessions as "dead" if no update in 3 minutes:
```go
deadThreshold := 3 * time.Minute
// ...
if timeSinceUpdate > deadThreshold {
    status = "dead"
}
```

**Source:** `cmd/orch/serve.go:783-805`

**Significance:** The dashboard was correctly setting status to "dead", but `is_processing` was still being set based on the flawed `IsSessionProcessing()` check. This could cause confusion in the UI.

---

### Finding 3: runStatus calls IsSessionProcessing directly

**Evidence:** The `runStatus` function in `main.go` calls `client.IsSessionProcessing()` for both tmux agents and OpenCode agents:
- Line 2801: `isProcessing = client.IsSessionProcessing(session.ID)`
- Line 2867: `isProcessing := client.IsSessionProcessing(oa.session.ID)`

**Source:** `cmd/orch/main.go:2801, 2867`

**Significance:** Fixing `IsSessionProcessing()` would fix the issue in both CLI status output and the dashboard API.

---

## Synthesis

**Key Insights:**

1. **Zombie sessions are detectable via activity time** - If a session hasn't been updated in 3 minutes, it's dead. Active agents constantly update session state (every tool call, every message part).

2. **The fix belongs in IsSessionProcessing()** - Rather than fixing in multiple places, fixing the core function ensures consistency across CLI and API.

3. **Using IsSessionActive before checking messages** - The fix first checks if the session is stale using `GetSession()` to get update time, then only checks message completion if the session is recently active.

**Answer to Investigation Question:**

Zombie sessions showed `is_processing: true` because `IsSessionProcessing()` only checked message completion status, not session activity. A session killed mid-execution has an incomplete assistant message (with pending tool call) that never completes, making it appear "processing" indefinitely. The fix adds a 3-minute staleness check: if a session hasn't been updated in 3 minutes, it cannot be processing.

---

## Structured Uncertainty

**What's tested:**

- ✅ `IsSessionProcessing()` returns false for stale sessions (verified: new test case "idle - stale session even with incomplete assistant message")
- ✅ All existing tests still pass (verified: `go test ./pkg/opencode/...`)
- ✅ Dashboard shows 24 dead agents with none having `is_processing=true` (verified: curl /api/agents)

**What's untested:**

- ⚠️ Performance impact of extra `GetSession()` call (should be minimal, single HTTP request)
- ⚠️ Edge case: session updated exactly at 3-minute boundary

**What would change this:**

- If OpenCode adds native "interrupted" state detection
- If OpenCode times out pending tool calls automatically

---

## Implementation

**Changes made:**

1. **pkg/opencode/client.go**:
   - Added `StaleSessionThreshold = 3 * time.Minute` constant
   - Modified `IsSessionProcessing()` to first check session activity time
   - If session not updated in 3 minutes, return false immediately

2. **pkg/opencode/client_test.go**:
   - Updated test to provide both session and message data
   - Added new test case for stale sessions with incomplete messages

3. **cmd/orch/serve.go**:
   - Changed `deadThreshold := 3 * time.Minute` to use `opencode.StaleSessionThreshold` for consistency

**Success criteria:**

- ✅ Zombie sessions no longer show `is_processing: true`
- ✅ Active sessions still show processing state correctly
- ✅ All tests pass

---

## References

**Files Examined:**
- `pkg/opencode/client.go` - Core client with IsSessionProcessing
- `pkg/opencode/client_test.go` - Tests for the client
- `cmd/orch/serve.go` - Dashboard API handler
- `cmd/orch/main.go` - CLI status command

**Commands Run:**
```bash
# Verify tests pass
go test ./pkg/opencode/... -v -run "TestIsSession"

# Check dead agents in dashboard
curl -s http://127.0.0.1:3348/api/agents | jq '[.[] | select(.status == "dead")]'
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-28-inv-stale-sessions-after-opencode-restart.md` - Original investigation of the symptom

---

## Investigation History

**2025-12-30 15:34:** Investigation started
- Initial question: Why do zombie sessions show `is_processing: true`?
- Context: Spawned to fix stale session detection

**2025-12-30 15:40:** Root cause identified
- `IsSessionProcessing()` doesn't check session activity time
- Only checks message completion status

**2025-12-30 15:45:** Fix implemented
- Added staleness check to `IsSessionProcessing()`
- Updated tests to verify fix

**2025-12-30 15:50:** Investigation completed
- Status: Complete
- Key outcome: `IsSessionProcessing()` now returns false for stale sessions
