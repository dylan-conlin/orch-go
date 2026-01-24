<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Session ID capture fails in tmux spawn because OpenCode registers sessions asynchronously after TUI startup, causing race condition.

**Evidence:** Code shows `WaitForOpenCodeReady` + 2s sleep followed by `FindRecentSession` call; API check fails when session not yet registered.

**Knowledge:** The 30-second recency window in `FindRecentSession` is correct; the issue is retry timing not static delay.

**Next:** Implementation complete - added `FindRecentSessionWithRetry` with 3 attempts at 500ms/1s/2s backoff.

**Confidence:** High (90%) - solution matches pattern used in similar async scenarios, tests pass.

---

# Investigation: Fix Session ID Capture Timing in Tmux Spawn

**Question:** Why does tmux spawn warn "could not capture session_id" and what's the best fix?

**Started:** 2025-12-21
**Updated:** 2025-12-21
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Race condition between TUI startup and API registration

**Evidence:** In `runSpawnTmux` (cmd/orch/main.go:1008-1023):
1. `WaitForOpenCodeReady` detects TUI is visible in tmux
2. `time.Sleep(2 * time.Second)` provides static delay
3. `FindRecentSession` queries API for sessions created in last 30s
4. Warning printed if session not found

The 2 second sleep is arbitrary and may not be sufficient for OpenCode to register the session.

**Source:** cmd/orch/main.go:1008-1023

**Significance:** Static delays are fragile. The session registration timing varies based on system load, network conditions, etc.

---

### Finding 2: FindRecentSession has 30-second recency window

**Evidence:** `FindRecentSession` (pkg/opencode/client.go:347-350):
```go
// Only match sessions created in the last 30 seconds
if now-s.Time.Created > 30*1000 {
    continue
}
```

**Source:** pkg/opencode/client.go:347-350

**Significance:** The 30-second window is generous enough. The issue is not the window size but the timing of when we check.

---

### Finding 3: Window ID is sufficient for tmux monitoring

**Evidence:** After session ID capture, the code:
- Registers agent with both `SessionID` and `WindowID`
- Logs both in the event
- Prints both in summary (but only SessionID if non-empty)

The `WindowID` is always captured successfully and provides sufficient tracking for tmux-based monitoring.

**Source:** cmd/orch/main.go:1035-1054 (registry and event logging)

**Significance:** This means we can safely suppress the warning since tmux monitoring doesn't depend on session ID.

---

## Synthesis

**Key Insights:**

1. **Retry with backoff is better than static delay** - Instead of guessing a fixed wait time, exponential backoff handles variable registration timing gracefully.

2. **Silent failure is acceptable here** - Since `window_id` provides sufficient monitoring capability for tmux spawns, failing to capture `session_id` is not critical.

3. **Existing test was broken** - The `TestFindRecentSession` test used timestamps from 1970, which always failed the 30-second recency check. Fixed as part of this work.

**Answer to Investigation Question:**

The warning occurs because OpenCode's session registration with its API happens asynchronously after the TUI becomes visible. A static 2-second delay doesn't reliably bridge this gap. The fix is to use a retry loop with exponential backoff (3 attempts: 500ms, 1s, 2s) and silently accept failure since `window_id` provides sufficient tracking.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

The solution is straightforward and follows well-established patterns for handling async registration. Tests pass and the logic is simple.

**What's certain:**

- ✅ Race condition exists between TUI startup and API registration
- ✅ Retry with backoff is more robust than static delay
- ✅ Window ID is sufficient for tmux monitoring

**What's uncertain:**

- ⚠️ Exact timing characteristics of OpenCode's session registration
- ⚠️ Whether 3 attempts is optimal (could need adjustment in practice)

**What would increase confidence to Very High (95%+):**

- Real-world testing across different machines and load conditions
- Logging to measure actual session registration times

---

## Implementation Recommendations

### Recommended Approach ⭐

**Retry with exponential backoff, suppress warning** - Add `FindRecentSessionWithRetry` and use it in tmux spawn.

**Why this approach:**
- Handles variable registration timing gracefully
- Exponential backoff prevents aggressive polling
- Silent failure is safe since window_id works

**Trade-offs accepted:**
- Total wait time could be up to 3.5s (500ms + 1s + 2s) vs previous 2s
- Lost diagnostics since warning is suppressed (but warning was noise anyway)

**Implementation sequence:**
1. Add `FindRecentSessionWithRetry` to pkg/opencode/client.go
2. Update runSpawnTmux to use retry function
3. Remove static sleep and warning

### Alternative Approaches Considered

**Option B: Increase static delay**
- **Pros:** Simpler change
- **Cons:** Still fragile, slows down every spawn unnecessarily
- **When to use instead:** Never - retry is strictly better

**Option C: Keep warning but reduce delay**
- **Pros:** Faster spawn time
- **Cons:** More warning noise
- **When to use instead:** If session ID was critical for monitoring

---

## References

**Files Examined:**
- cmd/orch/main.go:1008-1054 - tmux spawn logic
- pkg/opencode/client.go:313-361 - FindRecentSession implementation

**Commands Run:**
```bash
# Search for warning message
grep -n "could not capture session_id" cmd/orch/main.go

# Run tests
go test ./pkg/opencode/... -v -run TestFindRecentSession
```

---

## Implementation Completed

**Changes made:**

1. **pkg/opencode/client.go** - Added `FindRecentSessionWithRetry` function with:
   - Configurable max attempts and initial delay
   - Exponential backoff (delay doubles each attempt)
   - Returns last error on failure

2. **cmd/orch/main.go** - Updated `runSpawnTmux` to:
   - Remove static 2-second sleep
   - Use `FindRecentSessionWithRetry(projectDir, "", 3, 500*time.Millisecond)`
   - Silently ignore errors (window_id is sufficient)

3. **pkg/opencode/client_test.go** - Added tests:
   - `TestFindRecentSessionWithRetry` with 3 sub-tests
   - Fixed `TestFindRecentSession` timestamps (was using 1970 timestamps)
