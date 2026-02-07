<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** OpenCode server unhandledRejections logged in crash.log don't kill the server - they leave sessions in bad state. Agent death is caused by SSE stream breaking when server restarts, not by these errors.

**Evidence:** crash.log shows unhandledRejection events but comment says "doesn't exit by default, we just log it"; run.ts shows `for await (const event of events.stream)` loop that exits when stream breaks; prior investigation (Jan 23) confirmed sessions persist to disk.

**Knowledge:** Three distinct failure modes: (1) unhandledRejection leaves session bad but server alive, (2) server crash/restart kills SSE stream → client exits, (3) orch complete auto-rebuild restarts orch serve (port 3348) NOT OpenCode (port 4096).

**Next:** Implement auto-resume mechanism (designed Jan 17) to detect server restart and resume interrupted agents; fix the three unhandledRejection root causes in OpenCode fork.

**Promote to Decision:** recommend-no (tactical fixes, not architectural decision)

---

# Investigation: OpenCode Server Keeps Crashing/Dying - Agent Session Loss

**Question:** Why do spawned agent sessions die mid-work, and what causes OpenCode server crashes that kill agents?

**Started:** 2026-01-26
**Updated:** 2026-01-26
**Owner:** Worker agent (investigation)
**Phase:** Complete
**Next Step:** None - findings documented, recommendations ready
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** None
**Extracted-From:** Issue orch-go-20933
**Supersedes:** None
**Superseded-By:** None

---

## Findings

### Finding 1: Crash Logging is Implemented and Capturing Errors

**Evidence:** `~/.local/share/opencode/crash.log` exists with structured entries:
```
=== unhandledRejection at 2026-01-26T21:51:41.253Z ===
Error: ProviderModelNotFoundError
Stack:
ProviderModelNotFoundError: ProviderModelNotFoundError
    at getModel (src/provider/provider.ts:1082:17)
Memory:
  heapUsed: 108MB
  heapTotal: 57MB
  rss: 331MB
  external: 63MB
```

OpenCode server has crash handlers at `server.ts:119-127`:
```typescript
process.on("uncaughtException", (error, origin) => {
  writeCrashLog("uncaughtException", error)
})
process.on("unhandledRejection", (reason, promise) => {
  writeCrashLog("unhandledRejection", reason)
  // Note: unhandledRejection doesn't exit by default, we just log it
})
```

**Source:** `~/.local/share/opencode/crash.log`, `~/Documents/personal/opencode/packages/opencode/src/server/server.ts:119-127`

**Significance:** Crash logging from Jan 23 investigation was implemented. However, the key insight is that `unhandledRejection` events are logged but **don't crash the server** - they may leave sessions in a bad state instead.

---

### Finding 2: Three Specific Error Types in Crash Log

**Evidence:** Recent crash log shows three distinct error patterns:

1. **Session Summary Error** (multiple occurrences):
   ```
   TypeError: undefined is not an object (evaluating 'msgWithParts.info')
       at summarizeMessage (src/session/summary.ts:69:21)
   ```

2. **User Message Stream Error**:
   ```
   Error: No user message found in stream. This should never happen.
       at <anonymous> (src/session/prompt.ts:293:32)
   ```

3. **Provider Model Error** (most recent):
   ```
   ProviderModelNotFoundError
       at getModel (src/provider/provider.ts:1082:17)
   ```

**Source:** `~/.local/share/opencode/crash.log`

**Significance:** These are code bugs in OpenCode that need fixing:
- `summary.ts:69` - null check missing for `msgWithParts.info`
- `prompt.ts:293` - defensive handling needed for empty message streams
- `provider.ts:1082` - model validation error

---

### Finding 3: Agent Death Caused by SSE Stream Breaking, Not These Errors

**Evidence:** The `opencode run --attach` command subscribes to SSE events:
```typescript
// run.ts:154-158
const events = await sdk.event.subscribe()
const eventProcessor = (async () => {
  for await (const event of events.stream) {
    // ... process events
  }
})()
```

When the server crashes or restarts:
1. The SSE connection drops
2. The `for await` loop terminates
3. The client process exits
4. The spawned agent work is lost

**Source:** `~/Documents/personal/opencode/packages/opencode/src/cli/cmd/run.ts:154-158`

**Significance:** The unhandledRejection errors logged in crash.log leave sessions in bad state but don't kill the server. Agent death happens when the server actually crashes (from a different cause) and the SSE stream breaks. This is a distinction between "session corruption" and "session loss".

---

### Finding 4: orch serve Restart Does NOT Kill OpenCode Sessions

**Evidence:** `orch complete` has auto-rebuild logic at `complete_cmd.go:1478-1485`:
```go
// Restart orch serve if orch-go was rebuilt
if rebuiltOrchGo {
    if restarted, err := restartOrchServe(orchGoDir); err != nil {
        fmt.Fprintf(os.Stderr, "Warning: failed to restart orch serve: %v\n", err)
    } else if restarted {
        fmt.Println("Restarted orch serve")
    }
}
```

This restarts `orch serve` (port 3348) which is the **dashboard API server**, not OpenCode server (port 4096). They are separate processes.

**Source:** `cmd/orch/complete_cmd.go:1478-1485`, `cmd/orch/complete_cmd.go:1497-1554`

**Significance:** `orch complete` auto-rebuild does NOT cause OpenCode session loss. The concern about "orch complete auto-rebuild triggers restarts that kill sessions" is unfounded.

---

### Finding 5: Sessions Persist to Disk but In-Memory State Lost on Restart

**Evidence:** From prior investigation (Jan 23) and Jan 17 design:
- Sessions stored in `~/.local/share/opencode/storage/session/{projectID}/{sessionID}.json`
- Messages stored in `~/.local/share/opencode/storage/message/{sessionID}/`
- After server restart, disk sessions exist but in-memory sessions = 0
- The `x-opencode-directory` header returns disk sessions vs without returns in-memory only

**Source:** `~/.local/share/opencode/storage/`, `.kb/investigations/2026-01-17-inv-design-auto-resume-mechanism-stalled.md`

**Significance:** Sessions survive server restarts on disk. The problem is that spawned agents (`opencode run`) exit when SSE breaks, losing their work. The auto-resume mechanism (designed Jan 17) would detect orphaned sessions and resume them.

---

### Finding 6: Auto-Rebuild Lock Can Get Stuck

**Evidence:** Daemon log shows 30+ minutes of "rebuild already in progress" errors:
```
[11:50:40] Completion processing error: ... bd list failed: exit status 1: ⚠️  Auto-rebuild failed: rebuild already in progress
[12:18:15] Completion processing error: ... bd list failed: exit status 1: ⚠️  Auto-rebuild failed: rebuild already in progress
```

The lock file at `.autorebuild.lock` persists if a rebuild process dies.

**Source:** `~/.orch/daemon.log`, `cmd/orch/autorebuild.go:68-113`

**Significance:** When auto-rebuild crashes, the lock file remains, blocking all subsequent rebuilds for the entire daemon run. The lock cleanup logic in `isRebuildInProgress()` should handle this, but may have edge cases.

---

## Synthesis

**Key Insights:**

1. **unhandledRejection ≠ server crash** - The logged errors in crash.log indicate code bugs that corrupt sessions but don't crash the server. Agent death is caused by actual server crashes (from other causes) that break the SSE stream.

2. **SSE stream breaking is the kill mechanism** - When `opencode run --attach` loses its SSE connection, the `for await` loop terminates and the client exits. This is expected behavior but means agents die when the server restarts.

3. **orch serve and OpenCode are independent** - They run on different ports (3348 vs 4096), managed by different processes. `orch complete` auto-rebuild only restarts `orch serve`, not OpenCode.

4. **Auto-resume mechanism exists but not implemented** - The Jan 17 investigation designed a complete solution: detect server restart, find orphaned disk sessions, resume with recovery context. This would solve the session loss problem.

**Answer to Investigation Question:**

Agent sessions die mid-work because of **two distinct failure modes**:

1. **Session Corruption** - unhandledRejection errors in `summary.ts:69`, `prompt.ts:293`, `provider.ts:1082` leave sessions in bad state but don't crash the server. These need code fixes.

2. **Session Loss** - When the OpenCode server actually crashes (from causes not captured in these logs), the SSE stream breaks and spawned agents exit. The auto-resume mechanism would recover these.

`orch serve` restart does NOT kill OpenCode sessions - they're separate processes.

---

## Structured Uncertainty

**What's tested:**

- ✅ crash.log exists and contains structured errors (verified: read file)
- ✅ unhandledRejection handler says "doesn't exit by default" (verified: read server.ts)
- ✅ SSE subscription in run.ts uses for await loop (verified: read run.ts)
- ✅ orch complete restarts orch serve, not OpenCode (verified: read complete_cmd.go)
- ✅ Sessions persist to disk at ~/.local/share/opencode/storage/ (verified: ls -la)

**What's untested:**

- ⚠️ What actually crashes the OpenCode server (not captured in crash.log)
- ⚠️ Auto-resume mechanism works as designed (not yet implemented)
- ⚠️ Auto-rebuild lock cleanup handles all edge cases (saw stuck lock in logs)

**What would change this:**

- If crash.log showed uncaughtException instead of just unhandledRejection, would indicate actual crashes
- If OpenCode server had reconnection logic in SSE client, agents might survive restarts
- If auto-resume was implemented, session loss would become session pause

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Fix the three code bugs + Implement auto-resume** - Address root causes in OpenCode fork and implement the designed auto-resume mechanism.

**Why this approach:**
- Fixes observable errors in crash.log (immediate quality improvement)
- Auto-resume handles server restart gracefully (resilience)
- Both are low-risk, incremental changes

**Trade-offs accepted:**
- Requires modifying OpenCode fork
- Auto-resume adds complexity to daemon

**Implementation sequence:**
1. Fix `summary.ts:69` - Add null check for `msgWithParts.info`
2. Fix `prompt.ts:293` - Add defensive handling for empty message stream
3. Fix `provider.ts:1082` - Better error handling for model not found
4. Implement auto-resume from Jan 17 design:
   - Server restart detection
   - Disk session scanning
   - Staggered resume with recovery context

### Alternative Approaches Considered

**Option B: Claude Backend for Critical Work**
- **Pros:** Independent of OpenCode server, crash-resistant
- **Cons:** Loses orchestration visibility, manual escape hatch
- **When to use instead:** Already documented as escape hatch pattern in CLAUDE.md

**Option C: SSE Reconnection in Client**
- **Pros:** Agents survive server restarts automatically
- **Cons:** More complex, state management issues
- **When to use instead:** If auto-resume proves insufficient

**Rationale for recommendation:** Fixing the root causes is straightforward and reduces error noise. Auto-resume is already designed and provides the safety net for any future crashes.

---

### Implementation Details

**What to implement first:**
- Fix the three OpenCode bugs (quick wins)
- Rebuild OpenCode fork: `cd ~/Documents/personal/opencode/packages/opencode && bun run build`
- Then implement auto-resume from daemon (Jan 17 design)

**Things to watch out for:**
- ⚠️ OpenCode rebuild required after changes (see global CLAUDE.md)
- ⚠️ Auto-resume needs 30s stabilization delay after server start
- ⚠️ Rate limit applies per agent (1/hour) to prevent overwhelming server

**Areas needing further investigation:**
- What causes actual server crashes (not captured in current crash.log)
- Memory profiling under load
- SSE connection limits

**Success criteria:**
- ✅ crash.log no longer shows the three known error types
- ✅ After server restart, agents resume automatically within 2 minutes
- ✅ Resumed agents receive recovery context in prompt
- ✅ Dashboard shows "Recovered" status for resumed agents

---

## References

**Files Examined:**
- `~/.local/share/opencode/crash.log` - Crash telemetry
- `~/Documents/personal/opencode/packages/opencode/src/server/server.ts` - Crash handlers
- `~/Documents/personal/opencode/packages/opencode/src/cli/cmd/run.ts` - SSE subscription
- `cmd/orch/complete_cmd.go` - Auto-rebuild and restart logic
- `.kb/investigations/2026-01-23-inv-opencode-server-crashes-under-load.md` - Prior crash investigation
- `.kb/investigations/2026-01-17-inv-design-auto-resume-mechanism-stalled.md` - Auto-resume design

**Commands Run:**
```bash
# Check crash log
cat ~/.local/share/opencode/crash.log

# Check session storage
ls -la ~/.local/share/opencode/storage/

# Check daemon log for errors
grep -E "connection refused|error|crash" ~/.orch/daemon.log | tail -30

# Check current service status
overmind status
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-23-inv-opencode-server-crashes-under-load.md` - Prior crash investigation
- **Investigation:** `.kb/investigations/2026-01-17-inv-design-auto-resume-mechanism-stalled.md` - Auto-resume design
- **Model:** `.kb/models/opencode-session-lifecycle.md` - Session persistence model
- **Model:** `.kb/models/agent-lifecycle-state-model.md` - Four-layer state model

---

## Investigation History

**2026-01-26 16:30:** Investigation started
- Initial question: Why do spawned agent sessions die mid-work?
- Context: Issue orch-go-20933 reported OpenCode server crashing/dying

**2026-01-26 16:35:** Found crash.log with errors
- crash.log exists with structured entries
- Three distinct error types identified
- Key insight: unhandledRejection doesn't exit server

**2026-01-26 16:45:** Traced SSE stream breaking as kill mechanism
- run.ts uses `for await (const event of events.stream)`
- When stream breaks, client exits
- This is the actual agent death mechanism

**2026-01-26 16:55:** Verified orch serve vs OpenCode independence
- orch complete auto-rebuild restarts orch serve (3348)
- OpenCode server (4096) is separate process
- Confirmed: orch complete does NOT kill OpenCode sessions

**2026-01-26 17:00:** Investigation completed
- Status: Complete
- Key outcome: Three distinct failure modes identified; auto-resume would solve session loss
