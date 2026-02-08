## Summary (D.E.K.N.)

**Delta:** HTTP/2 is the recommended permanent fix for recurring HTTP/1.1 connection pool exhaustion. Single multiplexed SSE is a viable alternative with lower complexity.

**Evidence:** Go 1.24's `http.ListenAndServeTLS` auto-enables HTTP/2. The dashboard uses two SSE endpoints (/api/events, /api/agentlog) that consume 2 of 6 HTTP/1.1 connections. Current workaround (making agentlog opt-in) is a band-aid that doesn't address root cause.

**Knowledge:** HTTP/2 requires TLS for browsers but not for localhost. Go's ServeMux is HTTP/2 compatible. The recurring nature of this bug (2nd or 3rd time) indicates the current architecture won't scale.

**Next:** Decision for orchestrator - HTTP/2 is recommended for permanent fix, with single multiplexed SSE as alternative.

---

# Investigation: Design Permanent Fix for HTTP Connection Pool Exhaustion

**Question:** What is the permanent fix for recurring HTTP/1.1 connection pool exhaustion in the dashboard?

**Started:** 2026-01-05
**Updated:** 2026-01-05
**Owner:** architect agent
**Phase:** Complete
**Next Step:** Orchestrator decision on recommended approach
**Status:** Complete

---

## Findings

### Finding 1: HTTP/1.1 connection limit is a browser standard

**Evidence:** 
- All major browsers limit HTTP/1.1 connections to 6 per origin
- This is defined in RFC 7230 Section 6.4
- Chrome, Firefox, Safari all enforce this limit
- Each SSE connection is long-lived and occupies one slot

**Source:** Browser networking standards, observed behavior in dashboard network panel

**Significance:** This is a fundamental constraint that cannot be bypassed with HTTP/1.1. Any architecture using multiple long-lived connections will eventually hit this limit.

---

### Finding 2: Current workaround is a band-aid

**Evidence:** 
- Prior investigation (2026-01-05-inv-dashboard-connection-pool-exhaustion-sse.md) removed agentlog auto-connect
- This is the 2nd or 3rd time this issue has occurred
- Workaround reduces symptom but doesn't address root cause
- Future features requiring SSE will re-introduce the problem

**Source:** 
- `.kb/investigations/2026-01-05-inv-dashboard-connection-pool-exhaustion-sse.md`
- `.kb/investigations/2026-01-05-inv-fix-dashboard-connection-pool-exhaustion.md`

**Significance:** The pattern of recurring workarounds signals a missing coherent model. Per spawn context constraint: "High patch density in a single area signals missing coherent model."

---

### Finding 3: HTTP/2 eliminates the connection limit

**Evidence:**
- HTTP/2 uses multiplexing - single TCP connection handles multiple streams
- No 6-connection limit per origin
- Browsers support HTTP/2 natively (no client changes needed)
- Go 1.24 supports HTTP/2 out of the box with `http.ListenAndServeTLS`

**Source:** Go documentation, HTTP/2 spec (RFC 7540)

**Significance:** HTTP/2 is a protocol-level solution that removes the constraint entirely. No application logic changes needed on client or server.

---

### Finding 4: HTTP/2 requires TLS for browsers

**Evidence:**
- All major browsers only support HTTP/2 over TLS (h2)
- HTTP/2 cleartext (h2c) is not supported by browsers
- For localhost development, self-signed certificates work fine
- Production would use proper TLS anyway

**Source:** Browser HTTP/2 implementation notes, Go h2 documentation

**Significance:** This is a deployment consideration, not a blocker. Self-signed certs for localhost are trivial to set up.

---

### Finding 5: Single multiplexed SSE is viable but requires more work

**Evidence:**
- Current architecture has two SSE endpoints:
  - `/api/events` - proxies OpenCode SSE (session.status, message.part, etc.)
  - `/api/agentlog` - streams from events.jsonl (spawned, completed, error events)
- These could be combined into one endpoint with event type discrimination
- Client would filter by event type

**Source:**
- `cmd/orch/serve_agents_events.go:16-91` - handleEvents function
- `cmd/orch/serve_agents_events.go:93-225` - handleAgentlog function

**Significance:** This is a valid alternative that reduces connection usage from 2 to 1, but still doesn't scale if more SSE streams are needed.

---

### Finding 6: WebSocket is overkill for this use case

**Evidence:**
- SSE is server-to-client only (which is what we need)
- WebSocket is bidirectional (unnecessary overhead)
- SSE has better automatic reconnection handling
- SSE works over standard HTTP (simpler infrastructure)

**Source:** Protocol comparison, dashboard only needs server→client push

**Significance:** WebSocket would work but adds complexity without benefit. SSE is the right abstraction for this use case.

---

### Finding 7: Go HTTP/2 implementation is straightforward

**Evidence:**
- Current server: `http.ListenAndServe(addr, mux)` in `cmd/orch/serve.go:289`
- For HTTP/2: `http.ListenAndServeTLS(addr, certFile, keyFile, mux)`
- Go's ServeMux is fully HTTP/2 compatible
- SSE works over HTTP/2 without modification

**Source:** `cmd/orch/serve.go:289`, Go net/http documentation

**Significance:** The implementation is a one-line change plus TLS setup. No application logic changes needed.

---

## Synthesis

**Key Insights:**

1. **Recurring workarounds indicate architectural gap** - This is the 2nd or 3rd occurrence. The current architecture (HTTP/1.1 + multiple SSE) structurally cannot scale. Per spawn context: "High patch density signals missing coherent model."

2. **HTTP/2 is the protocol-level fix** - It removes the connection limit constraint entirely. This is not a workaround but an upgrade that eliminates the problem class.

3. **TLS is required but not blocking** - Self-signed certs for localhost are trivial. The dashboard is infrastructure tooling, not user-facing, so self-signed is acceptable.

**Answer to Investigation Question:**

HTTP/2 with TLS is the recommended permanent fix. It eliminates the HTTP/1.1 6-connection limit by multiplexing all requests over a single TCP connection. Implementation requires:
1. Generate self-signed TLS cert for localhost
2. Change `ListenAndServe` to `ListenAndServeTLS`
3. Update frontend to use `https://localhost:3348`

Single multiplexed SSE is a viable alternative with lower complexity but doesn't fully solve the problem if more SSE streams are needed in the future.

---

## Structured Uncertainty

**What's tested:**

- ✅ Go 1.24 supports HTTP/2 (verified: `go version` shows 1.24.11)
- ✅ Current server uses standard ServeMux (verified: `cmd/orch/serve.go`)
- ✅ SSE handlers use standard ResponseWriter (verified: `handleEvents`, `handleAgentlog`)

**What's untested:**

- ⚠️ HTTP/2 SSE streaming performance (not benchmarked)
- ⚠️ Self-signed cert generation on macOS (not tested)
- ⚠️ Frontend SSE reconnection behavior over HTTP/2 (not tested)

**What would change this:**

- If HTTP/2 SSE has issues with connection keep-alive or reconnection, would need investigation
- If Go's HTTP/2 implementation has bugs with SSE, would need workarounds
- If TLS handshake adds unacceptable latency for localhost, might prefer h2c with a proxy

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**HTTP/2 with TLS** - Upgrade the API server from HTTP/1.1 to HTTP/2

**Why this approach:**
- Eliminates the connection limit constraint at the protocol level
- No client-side JavaScript changes needed (browsers auto-negotiate HTTP/2)
- No SSE handler changes needed (Go HTTP/2 is transparent)
- Solves the problem class, not just the current symptom
- Directly addresses the "missing coherent model" signal from recurring patches

**Trade-offs accepted:**
- Requires TLS setup (self-signed cert generation)
- Browser dev tools show connections differently (single multiplex vs 6 connections)
- Slightly more complex server startup

**Implementation sequence:**
1. **TLS setup** - Generate self-signed cert for localhost (one-time, can be committed)
2. **Server change** - Replace `ListenAndServe` with `ListenAndServeTLS`
3. **Frontend update** - Change API_BASE from `http://` to `https://`
4. **Documentation** - Update README with cert setup for fresh clones

### Alternative Approaches Considered

**Option B: Single Multiplexed SSE**
- **Pros:** Simpler (no TLS), reduces connections from 2 to 1
- **Cons:** Still limited to 5 remaining connections, doesn't scale to more SSE types
- **When to use instead:** If TLS complexity is unacceptable for this project

**Option C: WebSocket instead of SSE**
- **Pros:** Single connection, bidirectional if needed
- **Cons:** Overkill for server→client only, more complex reconnection handling
- **When to use instead:** If bidirectional communication is ever needed

**Option D: Polling fallback**
- **Pros:** No persistent connections, simpler
- **Cons:** Higher latency, higher server load, defeats purpose of real-time updates
- **When to use instead:** As degraded mode when SSE fails

**Option E: Move SSE to different port/origin**
- **Pros:** Gets separate 6-connection pool
- **Cons:** CORS complexity, additional port management
- **When to use instead:** Not recommended - adds complexity without solving root cause

**Rationale for recommendation:** HTTP/2 is the only option that eliminates the constraint rather than working around it. The TLS requirement is minor friction compared to the benefit of solving the problem class permanently.

---

### Implementation Details

**What to implement first:**
1. TLS certificate generation script (can use `mkcert` or `openssl`)
2. Server code change (one-line swap)
3. Frontend API_BASE update

**Things to watch out for:**
- ⚠️ SSE reconnection behavior may differ slightly over HTTP/2 (test manually)
- ⚠️ Browser security warnings for self-signed certs (add exception once)
- ⚠️ CI/CD may need cert available for tests

**Areas needing further investigation:**
- Whether to use `mkcert` (easier) vs `openssl` (no extra dependency)
- Whether to generate cert at build time vs commit to repo
- Whether tests need HTTP/2 or can stay HTTP/1.1

**Success criteria:**
- ✅ Dashboard loads all API data without pending requests
- ✅ Both SSE streams can connect simultaneously without exhaustion
- ✅ Network panel shows HTTP/2 protocol indicator
- ✅ No more recurring "connection pool exhaustion" bugs

---

## References

**Files Examined:**
- `cmd/orch/serve.go:289` - Current HTTP server setup
- `cmd/orch/serve_agents_events.go:16-91` - handleEvents SSE handler
- `cmd/orch/serve_agents_events.go:93-225` - handleAgentlog SSE handler
- `web/src/lib/stores/agents.ts` - Frontend SSE connection
- `web/src/lib/stores/agentlog.ts` - Frontend agentlog SSE connection

**Commands Run:**
```bash
# Check Go version for HTTP/2 support
go version
# Output: go version go1.24.11 darwin/arm64

# Find HTTP server setup
grep -r "http.ListenAndServe" --include="*.go"
# Output: ./cmd/orch/serve.go:289
```

**External Documentation:**
- Go net/http HTTP/2 docs: https://pkg.go.dev/net/http#hdr-HTTP_2
- RFC 7540 (HTTP/2): https://tools.ietf.org/html/rfc7540
- RFC 7230 (HTTP/1.1 connection limit): Section 6.4

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-05-inv-dashboard-connection-pool-exhaustion-sse.md` - Prior workaround
- **Investigation:** `.kb/investigations/2026-01-05-inv-fix-dashboard-connection-pool-exhaustion.md` - Prior workaround

---

## Investigation History

**2026-01-05 13:00:** Investigation started
- Initial question: What is the permanent fix for recurring connection pool exhaustion?
- Context: This is the 2nd or 3rd time the issue has occurred

**2026-01-05 13:30:** Analysis complete
- Evaluated 5 options: HTTP/2, single SSE, WebSocket, polling, different port
- Identified HTTP/2 as the architectural fix vs workaround

**2026-01-05 13:45:** Investigation completed
- Status: Complete
- Key outcome: HTTP/2 with TLS recommended as permanent fix
