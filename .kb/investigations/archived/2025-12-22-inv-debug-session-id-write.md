<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Session ID capture fails in tmux mode because `FindRecentSessionWithRetry` runs BEFORE the prompt is sent, but OpenCode only creates sessions after receiving the first message.

**Evidence:** Only 8 of 233 workspaces (3.4%) have `.session_id` files; all 4 recent tmux spawns in event log show empty `session_id`; session creation timestamps are after prompt is sent.

**Knowledge:** OpenCode attach mode (`opencode attach <url>`) starts TUI but does NOT create a session until a message is received - session creation is message-triggered, not TUI-triggered.

**Next:** Move `FindRecentSessionWithRetry` to run AFTER the prompt is sent and enters are pressed, with appropriate delay for session registration.

**Confidence:** High (90%) - Consistent pattern across all recent spawns, verified with API timing analysis.

---

# Investigation: Session ID Write Failure in tmux Mode

**Question:** Why do tmux-spawned agents not have `.session_id` files written to their workspaces?

**Started:** 2025-12-22
**Updated:** 2025-12-22
**Owner:** Investigation agent
**Phase:** Complete
**Next Step:** None (fix recommended)
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Pervasive Session ID Capture Failure

**Evidence:** 
- Only 8 out of 233 workspaces (3.4%) have `.session_id` files
- All 4 recent tmux-spawned events show `session_id: ""` in event log
- 96.6% failure rate for session ID capture

**Source:** 
- `find .orch/workspace -name ".session_id" | wc -l` → 8
- `ls -d .orch/workspace/*/ | wc -l` → 233
- `~/.orch/events.jsonl` last 5 entries all show empty session_id

**Significance:** This is not an intermittent issue - it's a systematic failure. Nearly all tmux-spawned agents have no session ID captured, which breaks session resolution for `orch send`, `orch question`, and other commands.

---

### Finding 2: Session Created AFTER Prompt Sent

**Evidence:**
- My spawn event: timestamp 1766445135000 (ms)
- Session created at: 1766445135855 (ms)  
- Delta: 855ms AFTER spawn event
- Spawn event is logged AFTER prompt is sent, so session was created even later in the flow

**Source:**
- `~/.orch/events.jsonl` spawn event
- `curl http://127.0.0.1:4096/session/{id}` for session creation time

**Significance:** OpenCode's attach mode does NOT create a session when the TUI loads - it only creates a session when the first message is received. The session lookup happens before the prompt is sent, so the session doesn't exist yet.

---

### Finding 3: Code Ordering is Root Cause

**Evidence:** In `runSpawnTmux` (cmd/orch/main.go:1134-1191), the order is:
1. Line 1167: `WaitForOpenCodeReady` - waits for TUI to render
2. Line 1174: `FindRecentSessionWithRetry` - tries to find session ← TOO EARLY
3. Line 1179: `PostReadyDelay` - 1 second wait
4. Line 1180: `SendKeysLiteral` - sends prompt
5. Line 1183: `SendEnter` - submits prompt ← SESSION CREATED HERE
6. Line 1188: `WriteSessionID` - writes if found (but it wasn't)

**Source:** cmd/orch/main.go lines 1165-1192

**Significance:** The session lookup (step 2) happens BEFORE the prompt is sent (steps 4-5). Since OpenCode only creates sessions after receiving a message, the lookup always fails for tmux mode.

---

### Finding 4: Wrong Session ID Can Be Written

**Evidence:** 
- Session `ses_4bb0a1dcdffeoLsSbI7A1n1Jyd` appears in 3 different workspaces:
  - og-feat-implement-orch-init-21dec
  - og-feat-implement-max-agents-21dec
  - og-feat-implement-session-handoff-21dec

**Source:** `.session_id` file contents across workspaces

**Significance:** Even when session IDs ARE captured, they can be WRONG. The 30-second window in `FindRecentSession` can match sessions from concurrent spawns. This causes commands like `orch send` to target the wrong agent.

---

## Synthesis

**Key Insights:**

1. **Session creation is message-triggered** - OpenCode in attach mode starts the TUI but only creates a session when the first message is received. This is different from `opencode run` (inline mode) which creates a session immediately.

2. **Timing is fundamentally wrong** - The current code tries to find the session before it exists. No amount of retry delay will help because the session is only created AFTER the prompt is sent.

3. **Race condition for concurrent spawns** - The 30-second window combined with no title matching can cause multiple spawns to get the same session ID if they happen close together.

**Answer to Investigation Question:**

Session ID write fails because `FindRecentSessionWithRetry` is called BEFORE the prompt is sent, but OpenCode only creates sessions after receiving the first message. The lookup always fails because the session doesn't exist yet.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Strong evidence from multiple sources:
- Event log shows consistent empty session_id for all tmux spawns
- Session creation timestamps are after spawn events
- Code analysis confirms ordering issue

**What's certain:**

- ✅ Only 8 of 233 workspaces have session IDs (verified with find command)
- ✅ Session creation happens after message is sent (verified with API timing)
- ✅ Code order puts lookup before prompt sending (verified by reading source)

**What's uncertain:**

- ⚠️ Exact timing of when OpenCode creates session (after connect? after first keystroke? after Enter?)
- ⚠️ Whether there are other factors causing the 8 successful cases
- ⚠️ Best delay to wait after prompt before looking up session

**What would increase confidence to Very High (95%+):**

- Test the fix directly and verify session ID is captured
- Trace OpenCode source to confirm session creation trigger
- Test with concurrent spawns to verify race condition fix

---

## Implementation Recommendations

**Purpose:** Fix session ID capture for tmux-spawned agents.

### Recommended Approach ⭐

**Move session lookup to after prompt is sent** - The fix is straightforward: relocate `FindRecentSessionWithRetry` to run after `SendEnter` with an appropriate delay.

**Why this approach:**
- Matches the actual session lifecycle (session exists after message sent)
- Minimal code change (just reordering and adding delay)
- Consistent with how inline mode works (processes output which contains session ID)

**Trade-offs accepted:**
- Adds latency to spawn completion (~2-4s)
- Still has race condition for concurrent spawns (would need title matching to fully fix)

**Implementation sequence:**
1. Move `FindRecentSessionWithRetry` call to after `SendEnter` (line 1183)
2. Add delay before lookup (1-2s to allow session registration)
3. Increase retry count or delay for reliability
4. Consider adding workspace name matching via title

### Alternative Approaches Considered

**Option B: Use SSE to detect session creation**
- **Pros:** Event-driven, no polling
- **Cons:** More complex, requires SSE parsing in spawn flow
- **When to use instead:** If polling proves unreliable

**Option C: Create session via API first**
- **Pros:** Guarantees session exists before TUI
- **Cons:** Changes spawn flow, may not work with attach mode
- **When to use instead:** If OpenCode adds session pre-creation API

**Rationale for recommendation:** Option A is the simplest fix that addresses the timing issue directly. The session will exist after the prompt is sent, so moving the lookup there should work reliably.

---

### Implementation Details

**What to implement first:**
- Move the `FindRecentSessionWithRetry` call to after line 1183 (after Enter is sent)
- Add a delay (2-3 seconds) before the lookup to allow session registration
- Consider passing workspace name as title hint for better matching

**Things to watch out for:**
- ⚠️ The delay should be enough for session to appear in API but not too long to slow spawns
- ⚠️ Concurrent spawns still risk getting wrong session ID without title matching
- ⚠️ The 30-second window may need adjustment based on actual timing

**Areas needing further investigation:**
- How quickly does OpenCode register sessions after receiving a message?
- Should we match on workspace name in session title?
- Should inline mode also be fixed (currently relies on stdout parsing)?

**Success criteria:**
- ✅ `.session_id` files created for tmux-spawned agents
- ✅ Session ID matches the actual agent session
- ✅ `orch send <beads-id>` works for tmux agents

---

## References

**Files Examined:**
- cmd/orch/main.go:1134-1192 - `runSpawnTmux` function showing timing issue
- pkg/opencode/client.go:330-403 - `FindRecentSession` and retry logic
- pkg/spawn/session.go - `WriteSessionID` function

**Commands Run:**
```bash
# Count session ID files
find .orch/workspace -name ".session_id" | wc -l  # → 8

# Count total workspaces
ls -d .orch/workspace/*/ | wc -l  # → 233

# Check event log for session IDs
tail -5 ~/.orch/events.jsonl | jq '.data.session_id'  # → all empty
```

**Related Artifacts:**
- **Investigation:** This is the investigation for the session ID write issue

---

## Self-Review

- [x] Real test performed (not code review) - Verified with API queries and file system checks
- [x] Conclusion from evidence (not speculation) - Based on timing analysis and code tracing
- [x] Question answered - Why session IDs aren't written in tmux mode
- [x] File complete - All sections filled

**Self-Review Status:** PASSED

---

## Investigation History

**2025-12-22 15:12:** Investigation started
- Initial question: Why don't tmux-spawned agents have session IDs?
- Context: Noticed my workspace had no `.session_id` file

**2025-12-22 15:20:** Found pervasive failure
- Only 8 of 233 workspaces have session IDs
- All recent tmux spawns show empty session_id in events

**2025-12-22 15:25:** Identified root cause
- Code order puts session lookup BEFORE prompt is sent
- OpenCode only creates sessions after receiving a message

**2025-12-22 15:30:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Move session lookup to after prompt is sent
