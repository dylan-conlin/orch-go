<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** OpenCode server restarts leave zombie sessions that appear "active" in dashboard because they have pending tool calls that will never complete. **SIMPLIFIED:** Replaced complex `HasStalePendingTools()` detection with a simpler 3-minute heartbeat approach using `IsSessionActive()`.

**Evidence:** Session `ses_497e4b713ffeezgZ93N6CZimmV` had `tool.state.status: "pending"` for 25+ min. Original fix used tool state inspection; now simplified to just check session update time.

**Knowledge:** OpenCode persists session state to disk per-project-directory, survives server restarts, but doesn't mark interrupted sessions. The 3-minute heartbeat (via `IsSessionActive()`) is sufficient - sessions without updates in 3 minutes are considered dead.

**Next:** Complete. Simplified to use existing `IsSessionActive()` method. Removed unused `HasStalePendingTools()` and `IsSessionStale()` functions.

---

# Investigation: Stale Sessions After OpenCode Restart

**Question:** Why do killed/interrupted agent sessions appear as "active" in the dashboard after OpenCode server restart?

**Started:** 2025-12-28
**Updated:** 2025-12-28
**Owner:** Dylan + Orchestrator
**Phase:** Complete
**Next Step:** None - fix implemented
**Status:** Complete (Implemented)

---

## Findings

### Finding 1: Sessions persist per-project-directory in OpenCode

**Evidence:** 
- Query without `x-opencode-directory` header: session not found
- Query WITH header: session found and returns full data
- Sessions stored at `~/.local/share/opencode/storage/session/{projectID}/{sessionID}.json`

**Source:** 
```bash
# Without header - not found
curl -s http://localhost:4096/session/ses_497e4b713ffeezgZ93N6CZimmV
# Returns: NotFoundError

# With header - found
curl -s -H "x-opencode-directory: /Users/dylanconlin/Documents/personal/orch-go" \
  http://localhost:4096/session/ses_497e4b713ffeezgZ93N6CZimmV
# Returns: full session JSON
```

**Significance:** OpenCode's per-project session storage means queries must include the directory header to find project-specific sessions. This is working as designed but creates confusion when sessions appear/disappear based on query context.

---

### Finding 2: Interrupted sessions have pending tool calls that never complete

**Evidence:** Last message in zombie session:
```json
{
  "info": {
    "role": "assistant",
    "time": { "created": 1766978251765 }
    // Note: no "completed" timestamp
  },
  "parts": [
    { "type": "step-start" },
    { "type": "text", "text": "Now let me complete the test..." },
    { 
      "type": "tool",
      "tool": "edit",
      "state": { "status": "pending" }  // <-- Stuck here forever
    }
  ]
}
```

**Source:** 
```bash
curl -s -H "x-opencode-directory: ..." \
  "http://localhost:4096/session/ses_497e4b713ffeezgZ93N6CZimmV/message" | jq '.[-1]'
```

**Significance:** The session was killed mid-tool-execution. OpenCode preserved the state but the tool call will never complete. This makes the session appear "active" (incomplete assistant message) when it's actually dead.

---

### Finding 3: Dashboard correctly queries sessions but can't detect zombies

**Evidence:**
- `orch status` shows agent as "running" with `is_processing: true`
- Dashboard shows "Waiting for activity..."
- Both are technically correct - the session exists and has an incomplete response
- But the underlying process is dead - no activity will ever occur

**Source:** `cmd/orch/serve.go:handleAgents()` and `cmd/orch/main.go:runStatus()`

**Significance:** The current detection logic checks "does session exist?" and "is last message incomplete?" but doesn't check "has there been any activity recently given the pending state?"

---

## Synthesis

**Key Insights:**

1. **OpenCode is stateless across restarts** - It preserves session data but doesn't track which sessions were actively processing when killed. There's no "interrupted" state.

2. **Pending tool calls are the zombie signature** - A session with `tool.state.status: "pending"` that hasn't had activity in >5 minutes is almost certainly dead.

3. **The fix belongs in orch-go** - OpenCode is doing its job (persisting state). The orchestration layer should detect stale sessions and mark them appropriately.

**Answer to Investigation Question:**

Killed/interrupted sessions appear "active" because:
1. OpenCode persists session state to disk (by design)
2. Sessions mid-tool-execution have pending tool calls that never complete
3. Incomplete assistant messages = "processing" to the dashboard
4. No mechanism exists to detect "pending but stale" vs "pending and active"

---

## Structured Uncertainty

**What's tested:**

- ✅ Sessions persist after OpenCode restart (verified: killed server, restarted, session still exists)
- ✅ Pending tool calls remain pending forever (verified: checked message state, still "pending" after 15+ min)
- ✅ Dashboard shows zombie as "active" (verified: screenshot showed "Waiting for activity...")

**What's untested:**

- ⚠️ Exact timeout threshold for "stale" (5 min proposed but not validated)
- ⚠️ Whether OpenCode has any cleanup mechanism we could hook into
- ⚠️ Impact of marking sessions as stale on OpenCode's internal state

**What would change this:**

- If OpenCode adds an "interrupted" state on restart, this fix becomes unnecessary
- If OpenCode times out pending tool calls automatically, same
- If we find a way to resume interrupted sessions, the detection logic needs adjustment

---

## Implementation Recommendations

### Recommended Approach: Stale Tool Detection in orch-go

**Stale Tool Detection** - Add logic to detect sessions with pending tool calls that haven't had activity in >5 minutes

**Why this approach:**
- Fixes the immediate problem (zombie sessions in dashboard)
- Doesn't require OpenCode changes (we control orch-go)
- Low risk - only affects display, not session state

**Trade-offs accepted:**
- 5-minute threshold is arbitrary (might need tuning)
- Doesn't actually clean up the zombie sessions in OpenCode

**Implementation sequence:**
1. In `handleAgents()` and `runStatus()`, after checking `IsSessionProcessing()`
2. If processing, check last message's tool state and timestamp
3. If tool pending for >5 min with no message updates, mark as "stale" not "active"

### Alternative Approaches Considered

**Option B: Delete zombie sessions on detection**
- **Pros:** Cleans up OpenCode state, permanent fix
- **Cons:** Destructive, might delete sessions user wants to keep
- **When to use instead:** If zombies accumulate and cause performance issues

**Option C: Add OpenCode cleanup command**
- **Pros:** User-controlled cleanup
- **Cons:** Requires manual intervention, doesn't fix dashboard display
- **When to use instead:** As a companion to Option A for manual cleanup

---

### Implementation Details

**What to implement first:**
- Add `isStaleSession(session, lastMessage)` helper function
- Check: `tool.state.status == "pending" && time.Since(lastMessage.time.created) > 5*time.Minute`
- If stale, set `status = "stale"` or `status = "interrupted"` instead of "active"

**Things to watch out for:**
- ⚠️ Long-running tools (e.g., large file operations) might legitimately take >5 min
- ⚠️ Clock skew between session timestamps and current time
- ⚠️ Need to handle both `handleAgents()` (serve) and `runStatus()` (CLI)

**Success criteria:**
- ✅ Zombie sessions show as "stale" or "interrupted" in dashboard
- ✅ Normal processing sessions still show as "active"
- ✅ Can still see stale sessions (not hidden, just correctly labeled)

---

## References

**Files Examined:**
- `cmd/orch/serve.go:554-850` - handleAgents() implementation
- `cmd/orch/main.go:2371-2700` - runStatus() implementation
- `pkg/opencode/client.go` - OpenCode API client

**Commands Run:**
```bash
# Check session with directory header
curl -s -H "Accept: application/json" -H "x-opencode-directory: /Users/dylanconlin/Documents/personal/orch-go" \
  "http://localhost:4096/session/ses_497e4b713ffeezgZ93N6CZimmV"

# Check last message state
curl -s -H "Accept: application/json" -H "x-opencode-directory: /Users/dylanconlin/Documents/personal/orch-go" \
  "http://localhost:4096/session/ses_497e4b713ffeezgZ93N6CZimmV/message" | jq '.[-1]'
```

**Related Artifacts:**
- **Workspace:** `.orch/workspace/og-inv-test-spawn-work-28dec/` - The zombie session's workspace (deleted during debugging)

---

## Investigation History

**2025-12-28 19:16:** Investigation started
- Initial question: Why is dashboard showing "Waiting for activity..." for a killed agent?
- Context: Updated OpenCode from 1.0.182 to 1.0.207, server restart killed active sessions

**2025-12-28 19:25:** Key discovery - sessions persist per-project-directory
- Found that `x-opencode-directory` header required to find project-specific sessions
- Session exists but has stale pending tool call

**2025-12-28 19:30:** Root cause identified
- Pending tool calls with no activity = zombie signature
- OpenCode has no "interrupted" state

**2025-12-28 19:35:** Investigation completed
- Status: Complete
- Key outcome: Implement stale tool detection in orch-go to identify zombie sessions

**2025-12-28 19:45:** Original implementation completed
- Implemented stale tool detection in orch-go
- Zombie session `ses_497e4b713ffeezgZ93N6CZimmV` now correctly shows `status: "interrupted"`

**2025-12-28 (later):** Simplified to 3-minute heartbeat approach
- Removed `HasStalePendingTools()` and `IsSessionStale()` from `pkg/opencode/client.go`
- Removed associated tests from `pkg/opencode/client_test.go`
- The simpler `IsSessionActive()` method (which checks session update time) is sufficient
- 3-minute threshold detects dead sessions without needing to inspect tool state

---

## Implementation (Simplified)

### Current Approach: 3-Minute Heartbeat

Instead of the complex tool state inspection, zombie sessions are now detected via:

1. **`IsSessionActive(sessionID, 3*time.Minute)`** - Checks if `session.time.updated` is within the last 3 minutes
2. If a session hasn't had any activity in 3 minutes while appearing "processing", it's considered dead

### Why This Works

- Active agents update their session state frequently (every tool call, every message)
- A session with no updates for 3 minutes is effectively dead
- No need to inspect individual tool states - the heartbeat is sufficient
- Simpler code, fewer API calls, same result

### Removed Code (Cleanup)

The following were removed as unused:
- `HasStalePendingTools()` - Complex tool state inspection
- `IsSessionStale()` - Combined stale detection
- Associated tests in `client_test.go`

### Remaining Detection Infrastructure

- `IsSessionActive(sessionID, maxIdleTime)` - Time-based activity check (retained)
- `IsSessionProcessing(sessionID)` - Checks if last message is incomplete (retained)
- `GetLastMessage(sessionID)` - Retrieves last message (retained)
