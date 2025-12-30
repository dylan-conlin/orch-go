<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Headless spawns can silently fail because SendPrompt is called immediately after CreateSession with no delay or verification that the session is ready to receive messages.

**Evidence:** Code review shows tmux mode waits 1s after TUI ready before sending prompt (PostReadyDelay), but headless API mode calls SendPrompt immediately after CreateSession with no delay. The observed failure (session with 0 messages) matches this race condition.

**Knowledge:** OpenCode's HTTP API may not be immediately ready to process prompts after session creation; the session needs initialization time similar to TUI startup.

**Next:** Implement verification that session is ready + add delay before SendPrompt, OR wait for first message confirmation after SendPrompt (polling GetMessages until count > 0).

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Headless Spawn Silent Failures Agents

**Question:** Why do headless spawns sometimes create sessions that never execute (0 messages)?

**Started:** 2025-12-30
**Updated:** 2025-12-30
**Owner:** Agent (spawned by orchestrator)
**Phase:** Complete
**Next Step:** None - implement recommended fix
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Headless API mode has no delay between CreateSession and SendPrompt

**Evidence:** In `startHeadlessSessionAPI` (cmd/orch/main.go:2030-2041), `SendPrompt` is called immediately after `CreateSession` returns. There is no delay or readiness check.

```go
// Create session via HTTP API with correct directory
session, err := client.CreateSessionWithOptions(sessionTitle, cfg.ProjectDir, cfg.Model, opts)
if err != nil {
    return nil, spawn.WrapSpawnError(err, "Failed to create session via API")
}

// Send the initial prompt (IMMEDIATELY - no delay)
if err := client.SendPrompt(session.ID, minimalPrompt, cfg.Model); err != nil {
    return nil, spawn.WrapSpawnError(err, "Failed to send initial prompt")
}
```

**Source:** `cmd/orch/main.go:2030-2043`

**Significance:** This creates a race condition where SendPrompt may be called before OpenCode has fully initialized the session.

---

### Finding 2: Tmux mode has a 1-second delay before sending prompt

**Evidence:** In `runSpawnTmux` (cmd/orch/main.go:2113-2121), there's an explicit `PostReadyDelay` of 1 second after TUI is ready before sending the prompt.

```go
// Wait for OpenCode TUI to be ready
waitCfg := tmux.DefaultWaitConfig()
if err := tmux.WaitForOpenCodeReady(windowTarget, waitCfg); err != nil {
    return fmt.Errorf("failed to start opencode: %w", err)
}

// Send prompt
sendCfg := tmux.DefaultSendPromptConfig()
time.Sleep(sendCfg.PostReadyDelay)  // 1 second delay
if err := tmux.SendKeysLiteral(windowTarget, minimalPrompt); err != nil {
```

The `DefaultSendPromptConfig` (pkg/tmux/tmux.go:142-147) sets `PostReadyDelay: 1 * time.Second` with comment "TUI needs time for input focus".

**Source:** `cmd/orch/main.go:2101-2121`, `pkg/tmux/tmux.go:142-147`

**Significance:** The tmux mode explicitly waits for readiness, but the headless API mode assumes the session is immediately ready after CreateSession returns.

---

### Finding 3: SendPrompt returns success (HTTP 200) but doesn't verify message delivery

**Evidence:** `SendMessageAsync` (pkg/opencode/client.go:236-271) only checks for HTTP status code success, not whether the message was actually enqueued or processed.

```go
if resp.StatusCode < 200 || resp.StatusCode >= 300 {
    respBody, _ := io.ReadAll(resp.Body)
    return fmt.Errorf("unexpected status code: %d: %s", resp.StatusCode, string(respBody))
}
return nil  // Returns success without verifying message was received
```

**Source:** `pkg/opencode/client.go:266-270`

**Significance:** The spawn logic trusts that HTTP 200 means the message was delivered, but OpenCode might return 200 while the session is still initializing.

---

### Finding 4: No message count verification after spawn

**Evidence:** After `SendPrompt` returns, there's no check that the message was actually received. The `GetMessages` API exists and could be used to verify message count > 0.

```go
// GetMessages fetches all messages for a session from the OpenCode API.
func (c *Client) GetMessages(sessionID string) ([]Message, error) {
```

**Source:** `pkg/opencode/client.go:515-536`

**Significance:** The infrastructure for verification exists but isn't used in the headless spawn flow.

---

### Finding 5: monitorFirstComment only warns after 60 seconds

**Evidence:** After headless spawn, `monitorFirstComment` is started in a goroutine to detect failed starts:

```go
// Start background monitoring for first comment (failed-to-start detection)
if beadsID != "" && !cfg.NoTrack {
    go monitorFirstComment(beadsID, sessionID, cfg.WorkspaceName)
}
```

This uses `verify.WaitForFirstComment` with a 60-second timeout. This is detection, not prevention.

**Source:** `cmd/orch/main.go:1851-1854`, `pkg/verify/check.go:962-1010`

**Significance:** Current approach detects failures after the fact (60s later) but doesn't prevent them or retry.

---

## Synthesis

**Key Insights:**

1. **Race condition between session creation and prompt delivery** - The headless API mode calls SendPrompt immediately after CreateSession with no delay (Finding 1), while tmux mode has explicit delays and readiness checks (Finding 2). This explains why rapid sequential spawns (4 in 45 seconds) can fail - later spawns hit OpenCode while it's still processing earlier sessions.

2. **Fire-and-forget pattern without verification** - SendPrompt returns HTTP 200 but doesn't verify the message was actually processed (Finding 3). No verification step checks that the session has messages after spawn (Finding 4). This means silent failures go undetected until 60s later (Finding 5).

3. **Detection vs Prevention gap** - Current architecture detects failures after 60 seconds via `monitorFirstComment`, but doesn't prevent them or retry. The failure mode is "spawn succeeded but agent never started" which is hard to distinguish from "agent started but is slow".

**Answer to Investigation Question:**

Headless spawns can create sessions with 0 messages because `SendPrompt` is called immediately after `CreateSession` with no delay or verification. When spawning multiple agents in quick succession, OpenCode may not be ready to receive prompts immediately after creating sessions - the session needs initialization time similar to what tmux mode provides with its 1-second `PostReadyDelay`.

The observed failure (4th agent of 4 spawned in 45 seconds, session ses_48fd9f00dffe with 0 messages) is consistent with this race condition: session creation succeeded, SendPrompt returned 200, but the message was dropped because the session wasn't fully initialized.

**Limitations:** Without access to OpenCode source code, I cannot confirm exactly why messages are dropped. The analysis is based on behavioral evidence (tmux needing delays) and the observed failure pattern.

---

## Structured Uncertainty

**What's tested:**

- ✅ Code review confirmed no delay in headless API mode (verified: read cmd/orch/main.go:2030-2043)
- ✅ Tmux mode has 1-second delay (verified: read cmd/orch/main.go:2113-2121, pkg/tmux/tmux.go:142-147)
- ✅ SendMessageAsync only checks HTTP status (verified: read pkg/opencode/client.go:266-270)
- ✅ GetMessages API exists for verification (verified: read pkg/opencode/client.go:515-536)

**What's untested:**

- ⚠️ Exact timing required for session initialization (not benchmarked - need to test with various delays)
- ⚠️ Whether adding delay vs message verification is better (not compared - need A/B testing)
- ⚠️ OpenCode internal behavior when receiving prompt too early (no access to OpenCode source)
- ⚠️ Whether rate limiting is a factor (not tested - need to spawn rapidly and observe)

**What would change this:**

- Finding would be wrong if OpenCode always initializes sessions instantly (would need confirmation from OpenCode maintainers)
- Finding would be wrong if the failure was caused by something else (network issues, OpenCode bugs) and delay doesn't help
- Recommendation would change if message verification has significant latency cost (would need benchmarking)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Message Verification with Retry** - After SendPrompt, poll GetMessages until count > 0 or timeout (5s), with optional retry if message not delivered.

**Why this approach:**
- Provides positive confirmation that message was received (addresses Finding 3 & 4)
- Self-adjusting: no fixed delay, works whether OpenCode needs 100ms or 2s
- Enables automatic retry on failure instead of just detecting after 60s (addresses Finding 5)
- Aligns with pattern in `FindRecentSessionWithRetry` already in codebase

**Trade-offs accepted:**
- Adds ~1-5 seconds to spawn time (acceptable for reliability)
- More HTTP calls to OpenCode (acceptable given spawn frequency)

**Implementation sequence:**
1. Add `WaitForMessage(sessionID, timeout, interval)` function to `pkg/opencode/client.go` - polls GetMessages until count > 0
2. Modify `startHeadlessSessionAPI` to call WaitForMessage after SendPrompt
3. Add retry logic: if WaitForMessage times out, retry SendPrompt once
4. Update `monitorFirstComment` timeout from 60s to 30s since we now have early detection

### Alternative Approaches Considered

**Option B: Fixed delay before SendPrompt**
- **Pros:** Simple to implement, matches tmux pattern
- **Cons:** Wastes time if session is ready earlier; doesn't catch cases where message is still dropped
- **When to use instead:** If message verification adds too much latency or complexity

**Option C: SSE-based verification**
- **Pros:** Real-time notification when session starts processing
- **Cons:** More complex to implement; requires SSE subscription per spawn; may have race conditions
- **When to use instead:** If polling GetMessages proves too slow or rate-limited

**Option D: Add session readiness endpoint to OpenCode**
- **Pros:** Clean solution at the right layer
- **Cons:** Requires changes to OpenCode itself; not in our control
- **When to use instead:** If we can contribute to OpenCode project

**Rationale for recommendation:** Option A (message verification with retry) provides the strongest guarantee that spawns succeed, adapts to variable initialization times, and enables recovery from transient failures. The polling pattern is already established in the codebase (`FindRecentSessionWithRetry`) so the implementation fits existing conventions.

---

### Implementation Details

**What to implement first:**
- Add `WaitForMessage(sessionID, timeout, interval)` to `pkg/opencode/client.go`
- Integrate into `startHeadlessSessionAPI` after `SendPrompt`
- Add retry logic for failed message delivery

**Things to watch out for:**
- ⚠️ GetMessages might not immediately reflect new messages (eventual consistency)
- ⚠️ Polling interval should be short (200-500ms) but not too aggressive
- ⚠️ Total timeout should be generous (5s) to handle slow OpenCode startup
- ⚠️ Retry should only happen once to avoid infinite loops

**Areas needing further investigation:**
- What is the typical latency of GetMessages returning the first message?
- Is there a session.status SSE event when session becomes ready for prompts?
- Should we expose the verification timeout as a configuration option?

**Success criteria:**
- ✅ Spawning 4+ agents in 45 seconds should have 0% failure rate (currently has race condition)
- ✅ All spawned sessions should have message count > 0 after spawn returns
- ✅ Failed message deliveries should be logged with clear error messages
- ✅ Add metric: spawn_message_wait_ms (P50, P99) to track typical wait times

---

## References

**Files Examined:**
- `cmd/orch/main.go:1765-2055` - Headless spawn implementation (startHeadlessSessionAPI, runSpawnHeadless)
- `cmd/orch/main.go:2057-2200` - Tmux spawn implementation for comparison (runSpawnTmux)
- `pkg/opencode/client.go:236-271` - SendMessageAsync implementation
- `pkg/opencode/client.go:447-505` - CreateSession/CreateSessionWithOptions implementation
- `pkg/opencode/client.go:515-536` - GetMessages implementation
- `pkg/tmux/tmux.go:142-147` - DefaultSendPromptConfig with PostReadyDelay
- `pkg/verify/check.go:962-1010` - WaitForFirstComment implementation
- `pkg/opencode/monitor.go` - SSE monitoring for session status

**Commands Run:**
```bash
# Search for headless spawn patterns
grep -r "startHeadlessSessionAPI\|CreateSessionWithOptions\|SendPrompt" --include="*.go"

# Search for delay patterns in tmux mode
grep -rn "PostReadyDelay\|DefaultSendPromptConfig" --include="*.go"

# Search for session status handling
grep -r "session.status\|busy\|idle" --include="*.go"
```

**External Documentation:**
- OpenCode API behavior (inferred from code patterns, no direct documentation accessed)

**Related Artifacts:**
- **Prior Decisions:** kb context "headless" returned multiple related decisions about headless spawns
- **Investigation:** This investigation addresses observed failure pattern described in spawn task

---

## Investigation History

**2025-12-30 10:00:** Investigation started
- Initial question: Why do headless spawns sometimes create sessions with 0 messages?
- Context: Observed failure when spawning 4 agents in 45 seconds - 4th agent had 0 messages

**2025-12-30 10:15:** Analyzed headless spawn flow
- Found no delay between CreateSession and SendPrompt in startHeadlessSessionAPI

**2025-12-30 10:25:** Compared with tmux spawn flow
- Found 1-second PostReadyDelay in tmux mode before sending prompt
- Key insight: tmux knows to wait, headless doesn't

**2025-12-30 10:35:** Analyzed SendPrompt/SendMessageAsync
- Found fire-and-forget pattern with no message delivery verification
- GetMessages API exists but isn't used to confirm delivery

**2025-12-30 10:45:** Investigation completed
- Status: Complete
- Key outcome: Race condition between session creation and prompt delivery causes silent failures; recommended message verification with retry pattern
