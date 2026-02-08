<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The "No user message found" crash originates in OpenCode's `loop()` function at prompt.ts:273, not in orch-go; the root cause is OpenCode's `/prompt_async` endpoint not awaiting the prompt call.

**Evidence:** Traced error to `packages/opencode/src/session/prompt.ts:273`; server.ts:1240 shows `SessionPrompt.prompt({...})` without `await`, causing unhandled promise rejections.

**Knowledge:** orch-go can add defensive error event detection in SSE streams; the true fix requires OpenCode to await the async prompt call.

**Next:** Defensive fix applied to orch-go (session.error event handling); OpenCode bug should be filed for the missing await.

**Confidence:** High (85%) - Cannot test OpenCode fix directly from orch-go, but code analysis is clear.

---

# Investigation: OpenCode Crashes with "No user message found in conversation"

**Question:** When does OpenCode crash with "No user message found" and can orch-go prevent or handle this?

**Started:** 2025-12-25
**Updated:** 2025-12-25
**Owner:** orch-go debugging agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: Error Origin in OpenCode loop() Function

**Evidence:** The error message "No user message found in stream. This should never happen." is thrown at `packages/opencode/src/session/prompt.ts:273`:

```typescript
if (!lastUser) throw new Error("No user message found in stream. This should never happen.")
```

This occurs in the `loop()` function when iterating through session messages and finding no user messages.

**Source:** `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/prompt.ts:273`

**Significance:** The error is in OpenCode, not orch-go. The error text indicates this is meant to be an impossible state ("should never happen").

---

### Finding 2: Missing Await in /prompt_async Endpoint

**Evidence:** The `/prompt_async` endpoint at server.ts:1240 does not await the prompt call:

```typescript
.post(
  "/session/:sessionID/prompt_async",
  // ... validators ...
  async (c) => {
    c.status(204)
    c.header("Content-Type", "application/json")
    return stream(c, async () => {
      const sessionID = c.req.valid("param").sessionID
      const body = c.req.valid("json")
      SessionPrompt.prompt({ ...body, sessionID })  // <-- NO await!
    })
  },
)
```

**Source:** `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/server/server.ts:1240`

**Significance:** Without `await`, any error thrown by `SessionPrompt.prompt()` becomes an unhandled promise rejection. This can cause OpenCode to crash or behave unexpectedly, and the error is not returned to the client.

---

### Finding 3: orch-go Uses Safe Paths

**Evidence:** orch-go interacts with OpenCode through two main paths:
1. **Headless spawn**: Uses `opencode run --attach` CLI which creates session AND sends prompt atomically
2. **orch send**: Uses `SendMessageAsync` which calls `/prompt_async`, which calls `prompt()` that creates user message before `loop()`

Both paths should create user messages before `loop()` is called. However, the missing `await` in `/prompt_async` means errors are not propagated.

**Source:** 
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/main.go:1454-1475` (headless spawn)
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/opencode/client.go:182-214` (SendMessageAsync)

**Significance:** orch-go is not directly causing the error, but can add defensive handling for session errors via SSE events.

---

## Synthesis

**Key Insights:**

1. **Root cause is in OpenCode** - The missing `await` on line 1240 means async errors from prompt processing are not handled, leading to unhandled rejections.

2. **orch-go cannot prevent the crash** - The error occurs in OpenCode's internal processing. orch-go can only detect errors via SSE `session.error` events.

3. **Defensive improvement possible** - Adding `session.error` event detection to orch-go's SSE handling provides better error visibility to callers.

**Answer to Investigation Question:**

OpenCode crashes with "No user message found" when the `loop()` function is called on a session that has no user messages in storage. The most likely trigger is the `/prompt_async` endpoint not awaiting the prompt call, so errors become unhandled rejections. orch-go cannot prevent this but can detect and surface session errors via SSE event handling. The true fix requires modifying OpenCode to add `await` to the prompt call.

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

Strong code evidence traces the error origin and the likely cause. Cannot directly test the OpenCode fix without modifying OpenCode.

**What's certain:**

- ✅ Error originates at prompt.ts:273 in OpenCode's `loop()` function
- ✅ The `/prompt_async` endpoint is missing `await` on the prompt call
- ✅ orch-go's spawn and send paths should create user messages before `loop()`

**What's uncertain:**

- ⚠️ Exact user scenario that triggers this error in practice
- ⚠️ Whether there are other paths to `loop()` that could cause this
- ⚠️ Whether OpenCode's storage has race conditions

**What would increase confidence to Very High:**

- Direct reproduction of the error
- Fix applied to OpenCode and tested
- User confirmation that the error no longer occurs

---

## Implementation Recommendations

### Recommended Approach ⭐

**Defensive Error Handling in orch-go** - Add `session.error` event detection to SSE handling for better error visibility.

**Why this approach:**
- orch-go cannot fix the root cause (in OpenCode)
- Error events provide immediate feedback to callers
- Minimal invasive change to existing code

**Trade-offs accepted:**
- Does not prevent the error, only surfaces it better
- Requires OpenCode to emit `session.error` events (which it does)

**Implementation sequence:**
1. Add `ParseSessionError()` helper to parse error events ✅
2. Add session.error handling in `SendMessageWithStreaming()` ✅
3. Add tests for error event handling ✅

### Alternative Approaches Considered

**Option B: Pre-flight validation in orch-go**
- **Pros:** Could detect empty sessions before sending
- **Cons:** Extra API call; doesn't prevent all scenarios; race conditions still possible
- **When to use instead:** If error events are not emitted by OpenCode

**Option C: Report bug to OpenCode team**
- **Pros:** Fixes root cause
- **Cons:** Requires OpenCode team to review and release
- **When to use instead:** Always - should be done alongside defensive fix

---

### Implementation Details

**What to implement first:**
- ✅ Added `ParseSessionError()` function in sse.go
- ✅ Added session.error handling in `SendMessageWithStreaming()` in client.go
- ✅ Added tests for error parsing and error handling

**Things to watch out for:**
- ⚠️ Session error events may have different formats depending on OpenCode version
- ⚠️ Error events are not guaranteed to be emitted for all error types

**Areas needing further investigation:**
- When exactly the user sees this error (specific trigger scenario)
- Whether OpenCode should be fixed to add the missing `await`

**Success criteria:**
- ✅ Error events are detected and returned as Go errors
- ✅ Tests pass for error event handling
- ✅ All existing tests still pass

---

## References

**Files Examined:**
- `opencode/packages/opencode/src/session/prompt.ts:273` - Error origin
- `opencode/packages/opencode/src/server/server.ts:1240` - Missing await
- `orch-go/pkg/opencode/client.go` - SendMessageAsync and streaming
- `orch-go/pkg/opencode/sse.go` - SSE event parsing

**Commands Run:**
```bash
# Search for error message in OpenCode
rg "No user message" ~/Documents/personal/opencode

# Run tests
go test ./pkg/opencode/... -v
```

**Related Artifacts:**
- **Decision:** OpenCode model selection is per-message (from spawn context)
- **Investigation:** SSE Event Monitoring Client (2025-12-19)

---

## Investigation History

**2025-12-25:** Investigation started
- Initial question: When does OpenCode crash with "No user message found" and can orch-go fix it?
- Context: User reported OpenCode crashes with this error

**2025-12-25:** Traced error to OpenCode prompt.ts:273
- Found error is thrown in loop() when no user messages exist
- Identified missing await in /prompt_async endpoint

**2025-12-25:** Implemented defensive fix
- Added session.error event handling to orch-go
- Added tests for error parsing and handling
- All tests passing

**2025-12-25:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Root cause is in OpenCode (missing await); defensive error handling added to orch-go
