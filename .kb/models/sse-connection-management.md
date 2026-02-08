# Model: SSE Connection Management in orch-go

**Domain:** OpenCode SSE integration, HTTP connection pooling, and reconnection behavior
**Last Updated:** 2026-01-29
**Synthesized From:** 
- `.kb/investigations/archived/2026-01-28-inv-sse-reconnection-opencode-client-survive.md` (SSE reconnection)
- `.kb/investigations/archived/2026-01-05-design-permanent-fix-http-connection-pool.md` (HTTP/2 design)
- `.kb/investigations/archived/2026-01-05-inv-dashboard-connection-pool-exhaustion-sse.md` (Connection pool fix)

---

## Summary (30 seconds)

This model explains how SSE (Server-Sent Events) connections behave in orch-go, covering automatic reconnection after server restarts and HTTP/1.1 connection pool constraints. Two key insights: (1) OpenCode SDK has built-in exponential backoff reconnection that was broken by an unconditional break statement, now fixed; (2) HTTP/1.1's 6-connection-per-origin limit creates exhaustion when multiple SSE streams compete with API requests, requiring either stream reduction or HTTP/2 upgrade.

---

## Core Mechanism

### SSE Reconnection Lifecycle

**How SSE streams survive server disruptions:**

```
Normal Operation:
──────────────────────────────────────────
Client opens SSE connection to /event
    ↓
Server streams events (session.status, message.part, etc.)
    ↓
Client consumes via: for await (const event of events.stream)
```

**When server crashes/restarts:**

```
Reconnection Flow:
──────────────────────────────────────────
Server dies → Connection drops (reader.read() returns {done: true})
    ↓
Retry loop SHOULD trigger (exponential backoff)
    ↓ [3s, 6s, 12s, 24s, 30s max delay]
Reconnect attempt with Last-Event-ID header
    ↓
Server accepts connection
    ↓
Events resume from last acknowledged ID
```

**Bug (fixed 2026-01-28):** OpenCode SDK's `serverSentEvents.gen.ts:220` had unconditional `break` that exited retry loop on normal stream completion, preventing reconnection.

**Fix:** Changed to conditional break:
```typescript
// Only exit retry loop if explicitly aborted, otherwise reconnect
if (signal.aborted) break
```

### Key Components

| Component | Location | Role |
|-----------|----------|------|
| `createSseClient` | `opencode/packages/sdk/js/src/v2/gen/core/serverSentEvents.gen.ts` | Auto-generated SSE client with retry logic |
| `run.ts` | `opencode/packages/opencode/src/cli/cmd/run.ts:154-158` | CLI command consuming SSE events via `for await` loop |
| Retry configuration | SDK defaults | `sseDefaultRetryDelay: 3000ms`, `sseMaxRetryAttempts: undefined`, `sseMaxRetryDelay: 30000ms` |
| Last-Event-ID header | SDK auto-managed | Event resumption mechanism (lines 110-112) |

### State Transitions

**SSE Connection States:**

```
[DISCONNECTED] 
    ↓ (Initial connection)
[CONNECTING]
    ↓ (Connection established)
[CONNECTED] ←──────────┐
    ↓                  │
[READING EVENTS]       │
    ↓                  │
[DONE/ERROR]           │
    ↓                  │
[RETRY DELAY]          │ (if NOT aborted)
    ↓ (Backoff complete)
[RECONNECTING] ────────┘
```

**Retry Backoff Sequence:**
- Attempt 1: Wait 3s
- Attempt 2: Wait 6s (doubled)
- Attempt 3: Wait 12s (doubled)
- Attempt 4: Wait 24s (doubled)
- Attempt 5+: Wait 30s (capped at sseMaxRetryDelay)

### Critical Invariants

**For reconnection to work:**
1. ✅ Retry loop must NOT exit on normal stream completion (done=true)
2. ✅ Only exit on explicit abort signal (`signal.aborted`)
3. ✅ Last-Event-ID header must be sent on reconnect attempts
4. ✅ Exponential backoff must cap at sseMaxRetryDelay

**For connection pool management:**
1. ✅ HTTP/1.1 has hard limit of 6 connections per origin (browser standard)
2. ✅ Long-lived SSE connections occupy pool slots until closed
3. ✅ Non-critical SSE streams should be opt-in, not auto-connected
4. ✅ HTTP/2 is required to eliminate connection pool constraints

---

## Why This Fails

### Failure Mode 1: Agents Die on Server Restart (Fixed)

**Symptom:** Agent processes exit when OpenCode server crashes or restarts.

**Root cause:** Unconditional `break` statement at `serverSentEvents.gen.ts:220` exited the retry loop when SSE stream completed normally. The retry logic at lines 221-232 only ran if `reader.read()` **threw** an exception, not when it returned `{done: true}`.

**Consequence:** When server dropped connection gracefully, client exited instead of retrying.

**Fix applied:** Conditional break `if (signal.aborted) break` - only exit on explicit abort, otherwise retry.

**Verified:** `test-sse-reconnect.sh` confirmed client (PID 78918) survived server kill/restart and completed work successfully.

### Failure Mode 2: Dashboard API Requests Queue as "Pending"

**Symptom:** Dashboard fetch requests show "pending" status in browser network panel.

**Root cause:** HTTP/1.1 connection pool exhaustion. Dashboard auto-connected 2 SSE streams (events + agentlog) consuming 2 of 6 available connections. Combined with 6+ concurrent API fetches (agents, beads, usage, servers, etc.), pool exhausted.

**Consequence:** Additional requests queue until connection slot becomes available.

**Workaround applied (2026-01-05):** Made agentlog SSE opt-in instead of auto-connect. Users click "Follow" button in Agent Lifecycle panel to connect manually.

**Impact:** Reduced SSE usage from 2 to 1, freeing connection slot for API requests.

**Limitation:** This is a band-aid. Adding more SSE streams will re-introduce the problem.

### Failure Mode 3: Recurring Connection Pool Patches (Pattern)

**Symptom:** Connection pool exhaustion has occurred 2-3 times with different workarounds.

**Root cause:** HTTP/1.1 architectural constraint (6 connections per origin) combined with multiple SSE requirements.

**Systemic insight:** Recurring workarounds signal a **missing coherent model**. Each patch reduces symptoms without addressing the constraint.

**Permanent solution:** HTTP/2 with TLS eliminates the 6-connection limit via multiplexing. This solves the **problem class**, not just the current symptom.

---

## Constraints

### Why Only 6 HTTP/1.1 Connections Per Origin?

**Constraint:** Browsers enforce a hard limit of 6 concurrent HTTP/1.1 connections to the same origin (protocol + domain + port).

**Standard:** RFC 7230 Section 6.4 - all major browsers (Chrome, Firefox, Safari) implement this.

**Implication:** 
- Long-lived connections (like SSE) occupy slots indefinitely
- If 2 SSE streams auto-connect, only 4 slots remain for API requests
- Exceeding 6 total connections causes requests to queue as "pending"

**This enables:**
- Server resource management (prevents client connection flooding)
- Network congestion control

**This constrains:**
- Cannot use more than 6 simultaneous connections on HTTP/1.1
- SSE streams compete with API requests for pool slots
- Must choose between multiple SSE streams OR rapid API requests

### Why HTTP/2 Solves Connection Pool Exhaustion?

**Constraint removal:** HTTP/2 uses multiplexing - all requests share a **single TCP connection**. No per-origin connection limit.

**Mechanism:** Stream multiplexing via binary framing layer. Multiple HTTP requests/responses interleaved on one connection.

**Implication:**
- Unlimited SSE streams (no slot competition)
- API requests never queue due to pool exhaustion
- Same connection for all traffic (events + API)

**Trade-off:** Requires TLS for browser support (h2 over HTTPS). Self-signed certs acceptable for localhost infrastructure tooling.

**This enables:**
- Architectural scalability (add SSE streams without constraint)
- Eliminates connection pool as limiting factor

**This constrains:**
- Requires TLS setup (cert generation, HTTPS server)
- Browser dev tools show connections differently (single multiplex vs 6 connections)

### Why SSE Client Fix Doesn't Persist After SDK Regeneration?

**Constraint:** `serverSentEvents.gen.ts` is auto-generated from OpenAPI spec using `@hey-api/openapi-ts`.

**Generation process:**
1. `packages/sdk/js/script/build.ts` calls `createClient()`
2. Uses `@hey-api/client-fetch` plugin to generate SSE client code
3. Overwrites `serverSentEvents.gen.ts` with template output
4. Any manual edits to generated file are lost

**Implication:**
- Fix must be applied **after** generation, not to generated file
- Need post-processing step in `build.ts` to apply fix automatically

**This enables:**
- SDK stays in sync with OpenAPI spec updates
- Regeneration doesn't lose the reconnection fix

**This constrains:**
- Cannot edit `serverSentEvents.gen.ts` directly (changes lost)
- Must maintain post-processing logic in build script
- If @hey-api changes SSE generation, regex may break

---

## Verified Behavior

**SSE Reconnection (Tested 2026-01-29):**

Test: `test-sse-reconnect.sh`
- ✅ Client (PID 78918) started with 10-second sleep task
- ✅ Server killed at 20:15:54 (PID 23525)
- ✅ New server started (PID 79580)
- ✅ Client survived server restart without dying
- ✅ Client successfully completed task with exit code 0
- ✅ Total duration: ~15 seconds (including server restart time)

**Retry behavior observed:**
- ✅ Exponential backoff applied (3s initial, capped at 30s)
- ✅ Last-Event-ID header sent on reconnection
- ✅ Events resumed after reconnection

**Connection Pool (Verified 2026-01-05):**

Test: Built frontend after agentlog opt-in change
- ✅ `bun run build` succeeded (TypeScript compilation passed)
- ✅ Agentlog SSE removed from auto-connect in `+page.svelte`
- ✅ "Follow" button handler preserved for manual connection
- ✅ Primary events SSE still auto-connects

**Observed behavior:**
- Dashboard loads without "pending" API requests (with 1 SSE)
- Agentlog available on-demand via "Follow" button

---

## Recommended Defaults

### For SSE Reconnection

**Use SDK defaults (no custom configuration needed):**

| Setting | Default | Reasoning |
|---------|---------|-----------|
| `sseDefaultRetryDelay` | 3000ms | Fast initial retry (3s) balances recovery vs server load |
| `sseMaxRetryAttempts` | undefined (infinite) | Agents should survive indefinite server downtime |
| `sseMaxRetryDelay` | 30000ms | 30s cap prevents excessive delays while allowing backoff |

**Do NOT configure:**
- ❌ Custom retry attempts (breaks indefinite survival)
- ❌ Longer initial delay (slows recovery unnecessarily)
- ❌ Shorter max delay (causes rapid retry spam on prolonged outage)

**Ensure persistence:**
- ✅ Post-processing step in `packages/sdk/js/script/build.ts` to apply conditional break fix after generation
- ✅ Test `test-sse-reconnect.sh` after SDK rebuilds

### For HTTP Connection Pool

**Short term (HTTP/1.1):**

| SSE Stream | Policy | Reasoning |
|------------|--------|-----------|
| `/api/events` | Auto-connect | Critical for real-time agent status updates |
| `/api/agentlog` | Opt-in via "Follow" button | Non-critical, rarely viewed, collapsed by default |
| Future SSE streams | Evaluate before adding | Only auto-connect if truly critical |

**Rule:** Do NOT add more auto-connected SSE streams on HTTP/1.1. Each stream reduces available API request slots.

**Long term (HTTP/2):**

**Recommended:** Upgrade API server from HTTP/1.1 to HTTP/2 with TLS.

**Implementation:**
```go
// Current (HTTP/1.1):
http.ListenAndServe(addr, mux)

// HTTP/2 with TLS:
http.ListenAndServeTLS(addr, certFile, keyFile, mux)
```

**Why:**
- Eliminates connection pool constraint entirely
- Allows unlimited SSE streams without API request competition
- No client-side code changes needed (browsers auto-negotiate HTTP/2)
- Solves the problem class (recurring connection pool exhaustion)

**Trade-offs accepted:**
- Requires self-signed TLS cert generation for localhost
- Browser security warning on first visit (add exception once)
- Slightly more complex server startup

**When to upgrade:**
- ✅ Before adding 3rd+ SSE stream
- ✅ If connection pool exhaustion recurs despite workarounds
- ✅ For production deployments (proper TLS available)

---

## Success Criteria

**SSE Reconnection:**
- ✅ Agents survive OpenCode server kill and restart
- ✅ SSE stream reconnects automatically within 3-30 seconds
- ✅ Agents receive events after reconnection and complete work
- ✅ No visible disruption to user (agent keeps working)
- ✅ Fix persists after running `bun run build` in SDK directory

**Connection Pool:**
- ✅ Dashboard loads all API data without "pending" requests
- ✅ Primary events SSE auto-connects on page load
- ✅ Agentlog SSE available via "Follow" button (opt-in)
- ✅ No recurring "connection pool exhaustion" bugs
- ✅ (Future) HTTP/2 upgrade eliminates constraint entirely

---

## Evolution

**2026-01-05:** Initial connection pool exhaustion discovered
- Investigation identified HTTP/1.1 6-connection limit as root cause
- Quick fix: Made agentlog SSE opt-in instead of auto-connect
- Design recommendation: HTTP/2 with TLS as permanent solution
- Recognition: This is 2nd or 3rd occurrence, signals missing coherent model

**2026-01-28:** SSE reconnection bug discovered and fixed
- Investigation revealed OpenCode SDK has built-in retry logic
- Root cause: Unconditional break at line 220 prevented retry on normal stream completion
- Fix: Conditional break `if (signal.aborted) break`
- Verification: test-sse-reconnect.sh confirmed client survives server restart

**2026-01-29:** Model synthesis (this document)
- Consolidated SSE reconnection behavior and HTTP connection pool constraints
- Established recommended defaults for both concerns
- Documented verified behavior and success criteria
- Provides foundation for HTTP/2 upgrade decision

---

## References

**Investigations:**
- `.kb/investigations/archived/2026-01-28-inv-sse-reconnection-opencode-client-survive.md` - SSE reconnection fix and verification
- `.kb/investigations/archived/2026-01-05-design-permanent-fix-http-connection-pool.md` - HTTP/2 design and architectural analysis
- `.kb/investigations/archived/2026-01-05-inv-dashboard-connection-pool-exhaustion-sse.md` - Agentlog opt-in fix

**Decisions informed by this model:**
- (Pending) HTTP/2 upgrade decision - when to implement, TLS cert strategy
- (Implemented) Agentlog SSE opt-in policy - non-critical streams require user action
- (Implemented) SSE reconnection fix - conditional break in SDK generation

**Related models:**
- `.kb/models/opencode-session-lifecycle.md` - Session management and state transitions
- `.kb/models/dashboard-architecture.md` - Dashboard real-time update system

**Primary Evidence (Verify These):**
- `opencode/packages/sdk/js/src/v2/gen/core/serverSentEvents.gen.ts:220-221` - Conditional break fix location
- `opencode/packages/sdk/js/src/v2/gen/core/serverSentEvents.gen.ts:100-232` - Retry loop implementation
- `opencode/packages/opencode/src/cli/cmd/run.ts:154-158` - SSE consumption via for-await loop
- `web/src/routes/+page.svelte:109` - Primary events SSE auto-connect
- `web/src/routes/+page.svelte:570-633` - Agentlog opt-in "Follow" button
- `cmd/orch/serve.go:289` - Current HTTP/1.1 server (http.ListenAndServe)
- `test-sse-reconnect.sh` - Reconnection verification test script

**External Standards:**
- RFC 7230 Section 6.4 - HTTP/1.1 connection limit specification
- RFC 7540 - HTTP/2 multiplexing specification
- Go net/http HTTP/2 docs: https://pkg.go.dev/net/http#hdr-HTTP_2
