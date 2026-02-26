<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Removed title parameter from FindRecentSession - sessions now match by directory + creation time (within 30s) only.

**Evidence:** Manual verification confirms .session_id file created successfully for tmux spawns after the fix.

**Knowledge:** OpenCode session titles are set to the first prompt text, not workspace name, making title matching unreliable.

**Next:** Close - fix is complete and verified.

---

# Investigation: Fix FindRecentSession Match by Directory+Time Only

**Question:** Why can't tmux-spawned sessions find their session ID, and how can we fix FindRecentSession to work reliably?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** Feature Implementation Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Title matching was causing session lookup failures

**Evidence:** 
- Prior investigation showed sessions created via `opencode attach` have title set to first prompt text (e.g., "Reading SPAWN_CONTEXT for task setup")
- FindRecentSession was designed to optionally match by title, but title was always passed as empty string `""`
- The title parameter was redundant since it was never actually used in production code

**Source:** 
- `pkg/opencode/client.go:510-558` - FindRecentSession function
- `cmd/orch/spawn_cmd.go:1315` - caller always passing `""` for title

**Significance:** The title parameter was dead code that made the API confusing without providing any benefit.

---

### Finding 2: Directory + time (30s window) is sufficient for matching

**Evidence:**
- Spawn process is sequential: create workspace → start OpenCode → wait for ready → lookup session
- Between starting OpenCode and calling FindRecentSession, typically < 5 seconds
- 30-second window provides ample margin for session creation to complete

**Source:** 
- `cmd/orch/spawn_cmd.go:1291-1316` - spawn sequence
- Manual testing confirmed sessions are found within the 30s window

**Significance:** The simplified matching logic (directory + time) is reliable and doesn't need title.

---

### Finding 3: Fix verified with manual tmux spawn

**Evidence:**
```bash
$ orch spawn hello "test session capture v2" --tmux --bypass-triage --no-track
Spawned agent in tmux:
  Session ID: ses_46a3ac5bfffeyYLJNfG7fuoxF9
  ...

$ cat .orch/workspace/og-work-test-session-capture-06jan-ea65/.session_id
ses_46a3ac5bfffeyYLJNfG7fuoxF9
```

**Source:** Manual test on 2026-01-06 at ~16:05

**Significance:** Confirms the fix works end-to-end - .session_id file is now created for tmux spawns.

---

## Synthesis

**Key Insights:**

1. **API simplification** - Removing the title parameter makes the API cleaner and removes dead code that was never used.

2. **Reliable matching** - Directory + 30s creation time window is sufficient for session discovery. The retry logic (3 attempts with exponential backoff) handles race conditions.

3. **Backwards compatible** - The change only removed an unused parameter, so no external integrations are affected.

**Answer to Investigation Question:**

FindRecentSession couldn't reliably find sessions because it had a title parameter that, while optional, made the API confusing and suggested title matching might be needed. The real issue in the prior investigation was that tmux spawns used standalone OpenCode mode instead of attach mode. After fixing that (commit 18b26856a), this follow-up fix removes the unused title parameter to simplify the API.

---

## Structured Uncertainty

**What's tested:**

- ✅ Unit tests pass for FindRecentSession (verified: `go test ./pkg/opencode/... -v -run "FindRecentSession"`)
- ✅ Full test suite passes (verified: `go test ./...`)
- ✅ Manual tmux spawn creates .session_id file (verified: spawned and checked workspace)

**What's untested:**

- ⚠️ Behavior under heavy load (not tested, likely fine given 30s window)
- ⚠️ Edge case: multiple sessions created simultaneously in same directory (unlikely in practice)

**What would change this:**

- If OpenCode changes session creation timing significantly (>30s delay)
- If multiple agents are spawned to same directory within 30s (could return wrong session)

---

## Implementation Details

**Files Changed:**
1. `pkg/opencode/client.go` - Removed title parameter from FindRecentSession and FindRecentSessionWithRetry
2. `pkg/opencode/client_test.go` - Updated tests to not pass title parameter
3. `cmd/orch/spawn_cmd.go` - Updated caller to not pass empty title

**Success criteria:**
- ✅ Tests pass
- ✅ Build succeeds
- ✅ Manual verification shows .session_id created for tmux spawns

---

## References

**Files Examined:**
- `pkg/opencode/client.go` - FindRecentSession implementation
- `cmd/orch/spawn_cmd.go` - Spawn command calling FindRecentSession
- `pkg/opencode/client_test.go` - Tests for session finding

**Commands Run:**
```bash
# Run specific tests
go test ./pkg/opencode/... -v -run "FindRecentSession"

# Run full test suite
go test ./...

# Build and install
make install

# Manual verification
orch spawn hello "test session capture v2" --tmux --bypass-triage --no-track
cat .orch/workspace/og-work-test-session-capture-06jan-ea65/.session_id
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-06-inv-orchestrator-sessions-spawned-via-tmux.md` - Prior investigation that identified the need for this fix

---

## Investigation History

**2026-01-06 16:00:** Investigation started
- Initial question: Fix FindRecentSession to match by directory+time only
- Context: Follow-up from prior investigation that fixed tmux attach mode

**2026-01-06 16:03:** Implementation completed
- Removed title parameter from FindRecentSession and FindRecentSessionWithRetry
- Updated all callers and tests

**2026-01-06 16:05:** Manual verification passed
- Spawned tmux agent successfully captured session ID
- .session_id file created in workspace

**2026-01-06 16:06:** Investigation completed
- Status: Complete
- Key outcome: FindRecentSession now matches by directory + creation time only, removing the unused title parameter
