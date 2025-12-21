## Summary (D.E.K.N.)

**Delta:** Refactored `orch tail` to use OpenCode API for headless agents instead of requiring tmux.

**Evidence:** API endpoint `/session/{id}/message` tested and confirmed working; all tests pass; headless agent output successfully extracted.

**Knowledge:** OpenCode messages have structured parts (text, reasoning, step-start, etc.); filter for "text" type to get readable output; session_id tracking needed in registry for headless agents.

**Next:** Close issue - implementation complete and tested.

**Confidence:** High (90%) - tested with real API, all unit tests pass.

---

# Investigation: Refactor orch tail to use OpenCode API

**Question:** How can we refactor `orch tail` to use OpenCode API instead of tmux for headless agents?

**Started:** 2025-12-20
**Updated:** 2025-12-20
**Owner:** Worker Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: OpenCode API has message endpoint

**Evidence:** `curl http://127.0.0.1:4096/session/{id}/message` returns array of messages with parts.

**Source:** Direct API testing during investigation

**Significance:** This endpoint provides all message history for a session, enabling tail functionality without tmux.

---

### Finding 2: Messages have structured parts

**Evidence:** Each message has an `info` object (id, role, time) and `parts` array with types: "text", "reasoning", "step-start", "step-finish", "tool-invocation".

**Source:** API response structure from `/session/{id}/message`

**Significance:** Need to filter for "text" type parts to get readable output; reasoning and step markers can be skipped.

---

### Finding 3: Session ID tracking needed for headless agents

**Evidence:** Registry Agent struct only had window_id; headless agents have session_id from CreateSession response.

**Source:** pkg/registry/registry.go, cmd/orch/main.go

**Significance:** Added session_id field to Agent struct; stored at spawn time for headless agents.

---

## Synthesis

**Key Insights:**

1. **API-based tail is feasible** - OpenCode `/session/{id}/message` endpoint provides full message history that can be filtered for recent text output.

2. **Dual-mode operation required** - Need to support both headless (API) and tmux (capture-pane) modes for backward compatibility.

3. **Registry enhancement minimal** - Single field addition (session_id) enables tracking headless sessions.

**Answer to Investigation Question:**

Refactored `orch tail` to detect agent mode via registry lookup. Headless agents (window_id='headless') use API via GetMessages() + ExtractRecentText(). Tmux agents use existing capture-pane approach. Implementation complete with tests.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Tested with real OpenCode API, all unit tests pass, implementation follows existing patterns.

**What's certain:**

- ✅ API endpoint returns message data in documented format
- ✅ Text extraction works for multi-message sessions
- ✅ Session ID tracking works for headless spawns

**What's uncertain:**

- ⚠️ Long-running sessions with many messages (performance not tested at scale)
- ⚠️ Edge cases with non-text message types

---

## Implementation Recommendations

**Purpose:** Implementation complete.

### Recommended Approach ⭐

**Implemented:** Dual-mode tail with automatic detection.

**Implementation sequence:**
1. ✅ Added GetMessages() to pkg/opencode/client.go
2. ✅ Added ExtractRecentText() helper function
3. ✅ Added session_id to registry Agent struct
4. ✅ Refactored runTail() to detect mode and dispatch

---

## References

**Files Examined:**
- `cmd/orch/main.go` - Existing runTail implementation
- `pkg/opencode/client.go` - OpenCode API methods
- `pkg/registry/registry.go` - Agent tracking

**Commands Run:**
```bash
# Test API endpoint
curl http://127.0.0.1:4096/session/{id}/message

# Run tests
go test ./...
```

---

## Investigation History

**2025-12-20:** Investigation started
- Initial question: How to use OpenCode API for tail instead of tmux
- Context: Default spawn mode is now headless, tail needs to work without tmux

**2025-12-20:** Implementation complete
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Dual-mode tail working for both headless and tmux agents
