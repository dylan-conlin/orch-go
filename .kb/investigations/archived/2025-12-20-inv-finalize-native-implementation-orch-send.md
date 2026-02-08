**TLDR:** Question: How to finalize native Q&A implementation for `orch send` using OpenCode HTTP API exclusively? Answer: Implemented `SendMessageWithStreaming` method that sends message via async API, then connects to SSE to stream text events until session becomes idle. High confidence (90%) - validated with unit tests that mock SSE streaming.

---

# Investigation: Finalize Native Implementation for orch send

**Question:** How should `orch send` use the OpenCode HTTP API exclusively (no tmux dependency) with streaming responses?

**Started:** 2025-12-20
**Updated:** 2025-12-20
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Current implementation uses fire-and-forget async API

**Evidence:** The `runSend` function in `cmd/orch/main.go:942-967` calls `client.SendMessageAsync()` which just sends an HTTP POST to `/session/{id}/prompt_async` and returns immediately. No streaming or waiting for response.

**Source:** `cmd/orch/main.go:946`, `pkg/opencode/client.go:149-170`

**Significance:** Need to add SSE streaming after sending the message to capture the agent's response in real-time.

---

### Finding 2: SSE infrastructure already exists in pkg/opencode

**Evidence:** 
- `SSEClient` in `sse.go` connects to `/event` endpoint
- `ParseSSEEvent()` extracts event type and data from SSE format
- `ParseSessionStatus()` handles `session.status` events for completion detection

**Source:** `pkg/opencode/sse.go:12-84`, `pkg/opencode/monitor.go`

**Significance:** Can reuse existing SSE parsing infrastructure for streaming Q&A responses.

---

### Finding 3: Text streaming events use message.part format

**Evidence:** SSE events for text streaming use format:
```json
{"type":"message.part","properties":{"sessionID":"ses_xxx","messageID":"msg_1","part":{"type":"text","text":"..."}}}
```

**Source:** OpenCode SSE event structure, test cases in `pkg/opencode/client_test.go`

**Significance:** Need to parse `message.part` events and extract text from the part object to stream to output.

---

## Synthesis

**Key Insights:**

1. **Fire-and-forget is insufficient** - The old implementation sent messages but didn't wait for or display responses. Users need to see the agent's reply.

2. **SSE provides real-time streaming** - By connecting to `/event` after sending the message, we can stream text as it's generated rather than waiting for completion.

3. **Session filtering is essential** - The SSE endpoint streams ALL sessions. Must filter events by sessionID to only show responses from the target session.

**Answer to Investigation Question:**

The solution is a new `SendMessageWithStreaming` method that:
1. Sends message via existing async API
2. Connects to SSE endpoint at `/event`
3. Filters events for the target sessionID
4. Streams `message.part` text events to stdout
5. Returns when session becomes idle (busy → idle transition)

This uses OpenCode HTTP API exclusively with no tmux dependency.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Unit tests validate the SSE parsing and streaming logic with mock servers. The implementation follows established patterns from the existing monitor.go.

**What's certain:**

- ✅ SSE event format is correct (validated with tests)
- ✅ Session filtering works (TestSendMessageWithStreamingIgnoresOtherSessions passes)
- ✅ Completion detection works (busy → idle transition)

**What's uncertain:**

- ⚠️ Real-world SSE behavior may differ slightly from mock
- ⚠️ Edge cases with long-running sessions not fully tested

**What would increase confidence to Very High:**

- Integration test with real OpenCode server
- Test with concurrent sessions

---

## Implementation Recommendations

### Recommended Approach ⭐

**SSE Streaming after Async Send** - Send message asynchronously, then immediately connect to SSE to stream the response.

**Why this approach:**
- Non-blocking initial send avoids timeout issues
- Real-time streaming provides immediate feedback
- Reuses existing SSE infrastructure

**Trade-offs accepted:**
- More complex than fire-and-forget
- Slightly higher latency due to SSE connection setup

**Implementation sequence:**
1. Add `SendMessageWithStreaming` to `pkg/opencode/client.go`
2. Update `runSend` in `cmd/orch/main.go` to use streaming
3. Add unit tests for the new functionality

---

## References

**Files Examined:**
- `cmd/orch/main.go` - Entry point for send command
- `pkg/opencode/client.go` - HTTP client methods
- `pkg/opencode/sse.go` - SSE event parsing
- `pkg/opencode/monitor.go` - Completion detection patterns

**Commands Run:**
```bash
# Build verification
go build ./cmd/orch/...

# Run all tests
go test ./...
```

---

## Investigation History

**2025-12-20:** Investigation started
- Initial question: How to finalize native Q&A for orch send?
- Context: Need streaming responses without tmux dependency

**2025-12-20:** Implementation complete
- Added `SendMessageWithStreaming` method
- Updated `runSend` to use streaming
- Added comprehensive tests
- All tests passing
