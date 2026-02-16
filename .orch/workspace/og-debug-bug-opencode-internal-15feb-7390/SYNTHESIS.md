# SYNTHESIS: OpenCode 'Internal server error' on large bash tool output

**Issue:** `orch-go-1nh7`
**Skill:** systematic-debugging
**Status:** Fix committed locally

## Root Cause

The "Internal server error" originates from **Anthropic's API** (HTTP 500), not from within OpenCode itself. When a tool result contains large output, the Anthropic API occasionally returns a 500 error during streaming (`AI_APICallError: Internal server error` from `@ai-sdk/anthropic` at `doStream`).

The actual bug is in OpenCode's **SessionProcessor** (`packages/opencode/src/session/processor.ts`): the retry logic had **no maximum retry count**. When the API returns a retryable 500:

1. `SessionRetry.retryable(error)` returns truthy (500 is retryable per AI SDK)
2. `attempt++` increments without bound
3. The `while(true)` loop continues forever with exponential backoff capped at 30s
4. Session status stays in "retry" permanently — appearing "halted" to the user

## Fix

Added `MAX_RETRIES = 5` to `SessionProcessor`. After 5 failed retry attempts, the error falls through to the non-retryable error path:
- Sets error on the assistant message
- Publishes `Session.Event.Error`
- Sets session status to "idle"
- Logs "max retries exceeded" for observability

**File:** `packages/opencode/src/session/processor.ts`
**Commit:** In opencode repo (local, not pushed per task scope)

## Data Flow (for context)

```
BashTool.execute() → full output
    ↓
Tool.define() wrapper → Truncate.output() (50KB / 2000 lines)
    ↓
AI SDK streamText() → sends to Anthropic API
    ↓
Anthropic API → returns 500 on some large payloads
    ↓
SessionProcessor.process() catch block → [BUG: infinite retry]
```

Truncation IS working correctly (50KB limit applied before sending). The Anthropic 500 is likely triggered by payloads that are large but still within the 50KB truncation limit.

## What Was NOT the Problem

- SQLite storage limits (no size constraints on part data)
- Zod schema validation (schemas accept arbitrary string length)
- OpenCode's own HTTP server (global error handler not involved)
- Bus event publishing (fire-and-forget, no failure propagation)
- Tool truncation logic (working correctly at 50KB/2000 lines)

## Verification Contract

**Reproduction:** Send a prompt that triggers large bash output (e.g., `find / -type f 2>&1 | head -5000`). With the old code, the session hangs in "retry" status forever. With the fix, after 5 retries (~2.5 min with backoff), the session surfaces the error and returns to "idle".

**Smoke test:** The fix is structural — the `attempt < MAX_RETRIES` guard is deterministic. If `attempt >= 5`, the `continue` is skipped and error handling proceeds normally.

## Upstream Note

The root cause (Anthropic API returning 500 on certain payloads) is outside OpenCode's control. This fix handles the symptom gracefully. If the upstream issue is resolved, the retry logic still works correctly — it just succeeds within 5 attempts instead of failing after 5.
