<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** OpenCode client was using http.DefaultClient which has no timeout, causing /api/agents to hang indefinitely when OpenCode enters a redirect loop or becomes unresponsive.

**Evidence:** Grep search showed 5 http.DefaultClient.Do calls and 5 http.Get/Post calls in pkg/opencode/client.go, all without timeouts. Tests pass after adding configurable timeout.

**Knowledge:** HTTP clients for API calls should always have timeouts; SSE streaming connections should have redirect limits but no timeout since they're meant to be long-running.

**Next:** Close this issue - fix is complete and tested.

---

# Investigation: API Endpoint /api/agents Hangs When OpenCode Has Redirect Loop

**Question:** Why does /api/agents hang when OpenCode has a redirect loop, and how can we fix it?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: http.DefaultClient has no timeout

**Evidence:** The OpenCode client was using `http.DefaultClient` for all HTTP calls. Go's default HTTP client has no timeout set, meaning requests can hang indefinitely if the server doesn't respond.

**Source:** pkg/opencode/client.go - Lines 247, 402, 456, 574, 600 (http.DefaultClient.Do calls)

**Significance:** This is the root cause of the hang. When OpenCode enters a redirect loop or becomes unresponsive, the HTTP request blocks forever, causing /api/agents to hang.

---

### Finding 2: Multiple unprotected HTTP calls

**Evidence:** The OpenCode client had 10+ HTTP calls without timeouts:
- 5 `http.DefaultClient.Do` calls
- 5 `http.Get` / `http.Post` calls

**Source:** pkg/opencode/client.go, pkg/opencode/sse.go

**Significance:** All API-facing HTTP calls needed timeout protection. The fix required updating all of them.

---

### Finding 3: SSE needs special handling

**Evidence:** SSE (Server-Sent Events) connections are meant to be long-running streams. Adding a timeout to these would break the streaming functionality.

**Source:** pkg/opencode/sse.go, pkg/opencode/client.go:SendMessageWithStreaming

**Significance:** SSE connections should have redirect limits (to prevent redirect loops) but no timeout. This is different from regular API calls.

---

## Synthesis

**Key Insights:**

1. **Timeout is essential for API calls** - All HTTP calls to external services should have timeouts to prevent indefinite hangs.

2. **Redirect limiting prevents loops** - Even when timeout isn't appropriate (SSE), limiting redirects to 10 prevents redirect loops from hanging.

3. **Client configuration consolidation** - Creating a shared httpClient in the Client struct ensures consistent timeout handling across all methods.

**Answer to Investigation Question:**

The /api/agents endpoint hangs because it calls `client.ListSessions("")` which uses the OpenCode HTTP client. That client was using `http.DefaultClient` with no timeout. When OpenCode has a redirect loop, the HTTP request blocks forever.

The fix adds a 10-second default timeout to the OpenCode client and limits redirects to 10. SSE streaming connections have redirect limits but no timeout since they're meant to be long-running.

---

## Structured Uncertainty

**What's tested:**

- ✅ All OpenCode package tests pass (verified: go test ./pkg/opencode/... -v)
- ✅ Code compiles (verified: go build ./...)
- ✅ Timeout constant is 10 seconds (verified in client.go)

**What's untested:**

- ⚠️ Actual redirect loop scenario (would need OpenCode in bad state)
- ⚠️ Connection refused behavior (timeout still applies)
- ⚠️ SSE streaming still works in production (tests use mocks)

**What would change this:**

- If OpenCode legitimately needs >10 seconds to respond, timeout may need adjustment
- If SSE connections fail to establish within reasonable time, may need connect timeout

---

## Implementation Recommendations

### Recommended Approach (Implemented)

**Add configurable HTTP client with timeout to OpenCode client**

**Why this approach:**
- Prevents indefinite hangs on API calls
- Centralizes timeout configuration in one place
- Allows custom timeout via NewClientWithTimeout for special cases

**Trade-offs accepted:**
- 10 second default may be too short for slow networks (can be configured)
- SSE connections have no timeout (correct for long-running streams)

**Implementation sequence:**
1. Added DefaultHTTPTimeout constant (10 seconds)
2. Added httpClient field to Client struct
3. Replaced all http.DefaultClient.Do calls with c.httpClient.Do
4. Replaced all http.Get/Post calls with proper request + c.httpClient.Do
5. Added redirect limiting (max 10) to prevent redirect loops
6. SSE clients get redirect limits but no timeout

---

## References

**Files Examined:**
- pkg/opencode/client.go - Main OpenCode HTTP client
- pkg/opencode/sse.go - SSE streaming client
- cmd/orch/serve.go - /api/agents handler using OpenCode client

**Commands Run:**
```bash
# Search for http.DefaultClient usage
grep -n "http\.DefaultClient\|http\.Get\|http\.Post" pkg/opencode/client.go

# Build to verify changes compile
go build ./...

# Run tests to verify changes work
go test ./pkg/opencode/... -v
```

---

## Investigation History

**2025-12-26:** Investigation started
- Initial question: Why does /api/agents hang when OpenCode has redirect loop?
- Context: Dashboard becomes unresponsive when OpenCode enters bad state

**2025-12-26:** Root cause identified
- Found http.DefaultClient usage without timeouts
- Identified 10+ unprotected HTTP calls

**2025-12-26:** Investigation completed
- Status: Complete
- Key outcome: Added 10-second timeout to OpenCode client with redirect limiting
