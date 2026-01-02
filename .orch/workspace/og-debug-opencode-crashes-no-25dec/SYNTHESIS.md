# Session Synthesis

**Agent:** og-debug-opencode-crashes-no-25dec
**Issue:** orch-go-2s2d
**Duration:** 2025-12-25
**Outcome:** success

---

## TLDR

Investigated OpenCode crash with "No user message found" - root cause is in OpenCode (missing `await` in `/prompt_async` endpoint), not orch-go. Added defensive error event handling to orch-go for better error visibility.

---

## Delta (What Changed)

### Files Modified
- `pkg/opencode/client.go` - Added session.error event handling in SendMessageWithStreaming
- `pkg/opencode/sse.go` - Added ParseSessionError helper function
- `pkg/opencode/sse_test.go` - Added tests for error parsing
- `pkg/opencode/client_test.go` - Added test for session error detection

### Commits
- (Not yet committed - ready for commit)

---

## Evidence (What Was Observed)

- Error message "No user message found in stream. This should never happen." originates at `opencode/packages/opencode/src/session/prompt.ts:273`
- The `/prompt_async` endpoint at `server.ts:1240` does not await the prompt call: `SessionPrompt.prompt({ ...body, sessionID })` (missing `await`)
- orch-go uses safe paths that should always create user messages before `loop()` is called
- The OpenCode bug causes async errors to become unhandled promise rejections

### Tests Run
```bash
go test ./pkg/opencode/... -v
# PASS: All tests passing including new error handling tests

go test ./...
# PASS: Full test suite passing
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-25-inv-opencode-crashes-no-user-message.md` - Full investigation with root cause analysis

### Decisions Made
- Decision: Add defensive error handling rather than attempting to prevent the error (cannot prevent OpenCode internal errors)

### Constraints Discovered
- OpenCode's `/prompt_async` endpoint doesn't await the prompt, so errors are fire-and-forget
- orch-go can only detect errors via SSE events, cannot prevent them

### Externalized via `kn`
- N/A - This is an OpenCode bug that should be filed as an issue

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (defensive error handling added)
- [x] Tests passing (all pkg/opencode and full suite)
- [x] Investigation file has `**Status:** Complete`
- [x] Ready for `orch complete orch-go-2s2d`

**Additional action:** File bug in OpenCode repo for missing `await` at server.ts:1240

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- What is the specific user scenario that triggers this error? (Need reproduction steps)
- Are there other paths to `loop()` that could cause similar issues?

**Areas worth exploring further:**
- OpenCode's error handling patterns in other async endpoints
- Whether adding request tracing would help diagnose these issues

**What remains unclear:**
- Exact user scenario that triggers the error
- Whether OpenCode storage has race conditions affecting message visibility

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** claude-opus-4-20250514
**Workspace:** `.orch/workspace/og-debug-opencode-crashes-no-25dec/`
**Investigation:** `.kb/investigations/2025-12-25-inv-opencode-crashes-no-user-message.md`
**Beads:** `bd show orch-go-2s2d`
