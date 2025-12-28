<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The "redirected too many times" error is NOT a bug or auth issue - it's OpenCode's intentional design. Unknown routes are proxied to `desktop.opencode.ai` (a web app), which triggers auth redirects causing the loop. Valid endpoints work correctly.

**Evidence:** OpenCode `server.ts:2475-2482` has a catch-all `.all("/*")` route that proxies to `https://desktop.opencode.ai`. Tests confirm: `/session` returns 200, `/health` and `/sessions` (plural) return 500 with redirect error.

**Knowledge:** OpenCode does NOT have a `/health` endpoint. Use `/session` for health checks. The error is expected behavior, not something to fix in orch-go or OpenCode.

**Next:** Close - this is documented behavior. Prior investigations already captured this. Update orch doctor if needed to use `/session` instead of `/health`.

---

# Investigation: OpenCode Redirect Loop Health Sessions

**Question:** Why does OpenCode return "redirected too many times" for /health and /sessions endpoints, and is this affecting orch-go functionality?

**Started:** 2025-12-28
**Updated:** 2025-12-28
**Owner:** og-inv-opencode-redirect-loop-28dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage -->
**Extracted-From:** N/A
**Supersedes:** N/A (consolidates findings from 3+ prior investigations on this topic)
**Superseded-By:** N/A

---

## Findings

### Finding 1: OpenCode Catch-All Proxy Route

**Evidence:** In OpenCode source code (`packages/opencode/src/server/server.ts:2475-2482`):

```typescript
.all("/*", async (c) => {
  return proxy(`https://desktop.opencode.ai${c.req.path}`, {
    ...c.req,
    headers: {
      host: "desktop.opencode.ai",
    },
  })
})
```

This catch-all route is the LAST route in the chain, meaning any path NOT explicitly defined gets proxied to `desktop.opencode.ai`.

**Source:** `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/server/server.ts:2475-2482`

**Significance:** This is the root cause of the redirect loop. `desktop.opencode.ai` is a web application that requires OAuth authentication. When an invalid/unknown API route is hit, it gets proxied to this web app, which returns HTML and triggers auth redirects. Since orch-go isn't a browser with session cookies, the redirect loop never completes.

---

### Finding 2: Valid vs Invalid OpenCode Endpoints

**Evidence:** Testing against OpenCode (port 4096):

| Endpoint | Status | Response |
|----------|--------|----------|
| `GET /session` | 200 OK | JSON array of sessions |
| `GET /session/{id}` | 200 OK | Session details |
| `POST /session/{id}/prompt_async` | 204 | Prompt accepted |
| `GET /event` | 200 | SSE stream |
| `GET /health` | 500 | "redirected too many times" |
| `GET /sessions` (plural) | 500 | "redirected too many times" |
| `GET /message/{id}` | 500 | "redirected too many times" |

**Source:** `curl -s -w "\nStatus: %{http_code}\n" http://127.0.0.1:4096/session` (and similar tests)

**Significance:** OpenCode only handles routes under specific paths (`/session/*`, `/event`, `/config`, `/provider/*`, etc.). Any path not in this list gets proxied to the web app.

---

### Finding 3: orch-go Uses Correct Endpoints

**Evidence:** In `pkg/opencode/client.go`:
- Line 295: `ListSessions` uses `GET /session` ✓
- Line 324: `GetSession` uses `GET /session/{id}` ✓
- Line 254: `SendMessageAsync` uses `POST /session/{id}/prompt_async` ✓
- Line 497: `GetMessages` uses `GET /session/{id}/message` ✓

No code in orch-go uses `/health` or `/sessions` (plural).

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/pkg/opencode/client.go`

**Significance:** orch-go is not affected by this issue. All API calls use valid, local endpoints.

---

### Finding 4: This Has Been Investigated Multiple Times

**Evidence:** Prior investigations on this topic:
1. `2025-12-23-debug-opencode-api-redirect-loop.md` - Identified proxy architecture
2. `2025-12-26-inv-opencode-health-endpoint-returns-redirected.md` - Confirmed `/sessions` vs `/session` 
3. `2025-12-26-inv-can-we-auto-refresh-opencode.md` - Confirmed NOT auth-related

All three reached the same conclusion: this is expected OpenCode behavior, not a bug.

**Source:** `.kb/investigations/` directory search

**Significance:** This is the 4th investigation on the same topic. The issue keeps resurfacing because:
1. Dylan periodically tests with wrong endpoints
2. The error message is misleading ("redirected too many times" sounds like a bug)
3. No `/health` endpoint exists, which is unexpected for an HTTP server

---

### Finding 5: OpenCode Client Has Timeout and Redirect Protections

**Evidence:** In `pkg/opencode/client.go:17-41`:

```go
const DefaultHTTPTimeout = 10 * time.Second

func NewClient(serverURL string) *Client {
    return &Client{
        ServerURL: serverURL,
        httpClient: &http.Client{
            Timeout: DefaultHTTPTimeout,
            CheckRedirect: func(req *http.Request, via []*http.Request) error {
                if len(via) >= 10 {
                    return fmt.Errorf("too many redirects (max 10)")
                }
                return nil
            },
        },
    }
}
```

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/pkg/opencode/client.go:17-41`

**Significance:** Even if orch-go accidentally hit an invalid endpoint, it would fail gracefully with a timeout (10s) or redirect limit (max 10), not hang indefinitely.

---

## Synthesis

**Key Insights:**

1. **Intentional Design, Not Bug** - OpenCode proxies unknown routes to `desktop.opencode.ai` by design. This allows the local server to seamlessly integrate with the web app for features it doesn't handle locally.

2. **No /health Endpoint** - Unlike typical HTTP servers, OpenCode does NOT have a dedicated health check endpoint. Use `GET /session` to verify the server is running.

3. **Error Message is Misleading** - "The response redirected too many times" sounds like an auth or loop bug, but it's actually the expected response for hitting an invalid route.

4. **orch-go Is Protected** - The OpenCode client in orch-go has timeouts and redirect limits, so even if something went wrong, it wouldn't cause the system to hang.

**Answer to Investigation Question:**

The "redirected too many times" error occurs because OpenCode's local server proxies unknown routes (like `/health`, `/sessions`) to `desktop.opencode.ai`, which is a web app that triggers OAuth redirects. This is INTENTIONAL OpenCode behavior, not a bug.

This does NOT affect orch-go because:
1. orch-go uses only valid endpoints (`/session`, `/session/{id}/*`, `/event`)
2. The orch-go client has a 10-second timeout
3. The orch-go client limits redirects to 10

The "recurring for weeks" perception is likely because:
1. Dylan periodically tests with wrong endpoints (e.g., `/health` or `/sessions` plural)
2. The error message is misleading
3. Prior investigations exist but aren't surfaced in context

---

## Structured Uncertainty

**What's tested:**

- ✅ `/session` returns 200 OK (verified: `curl http://127.0.0.1:4096/session`)
- ✅ `/health` returns 500 with redirect error (verified: `curl http://127.0.0.1:4096/health`)
- ✅ OpenCode source shows catch-all proxy (verified: read server.ts:2475-2482)
- ✅ orch-go uses only valid endpoints (verified: grep in client.go)

**What's untested:**

- ⚠️ Whether desktop.opencode.ai could add local /health handling (would require OpenCode PR)
- ⚠️ Whether there are edge cases where valid endpoints trigger the proxy

**What would change this:**

- Finding that orch-go actually uses an invalid endpoint somewhere
- OpenCode adding a local /health endpoint
- A bug in OpenCode's route matching

---

## Implementation Recommendations

### Recommended Approach ⭐

**No changes needed** - orch-go is working correctly. This is expected OpenCode behavior.

**Why this approach:**
- orch-go uses only valid endpoints
- The error only occurs on invalid routes
- Adding a workaround would mask the real issue

**Trade-offs accepted:**
- No `/health` endpoint for simple health checks
- Must use `/session` as a proxy for health

### Alternative Approaches Considered

**Option B: Add /health proxy in orch serve**
- **Pros:** Provides expected health check semantics
- **Cons:** Adds complexity, masks OpenCode behavior
- **When to use instead:** If other tools require a `/health` endpoint

**Option C: Submit PR to OpenCode for local /health**
- **Pros:** Fixes the root cause for everyone
- **Cons:** Requires upstream changes, may not be accepted
- **When to use instead:** If this becomes a blocker for multiple tools

---

## Implementation Details

**What to implement first:**
- Nothing - this is not a bug to fix

**Things to watch out for:**
- ⚠️ Don't use `/health` - it will never work
- ⚠️ Don't use `/sessions` (plural) - it's `/session` (singular)
- ⚠️ The error message is misleading - it's not an auth issue

**Success criteria:**
- ✅ orch-go continues to work (already does)
- ✅ Dylan understands this is expected behavior

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/server/server.ts` - OpenCode HTTP server implementation
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/opencode/client.go` - orch-go OpenCode client

**Commands Run:**
```bash
# Test valid endpoint
curl -s -w "\nStatus: %{http_code}\n" http://127.0.0.1:4096/session

# Test invalid endpoint
curl -s -w "\nStatus: %{http_code}\n" http://127.0.0.1:4096/health
# Returns 500 with "redirected too many times"

# Check OpenCode processes
pgrep -fl opencode
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-23-debug-opencode-api-redirect-loop.md` - First investigation on this topic
- **Investigation:** `.kb/investigations/2025-12-26-inv-opencode-health-endpoint-returns-redirected.md` - Second investigation
- **Investigation:** `.kb/investigations/2025-12-26-inv-can-we-auto-refresh-opencode.md` - Confirmed not auth-related
- **kn decision:** `kn-2a4e34` - "OpenCode has no /health endpoint - use /session to check server status"

---

## Self-Review

- [x] Real test performed (curl tests against OpenCode, source code review)
- [x] Conclusion from evidence (based on test results and source code)
- [x] Question answered (redirect loop is expected behavior, not a bug)
- [x] File complete (all sections filled)
- [x] D.E.K.N. filled
- [x] NOT DONE claims verified (searched for prior investigations, found 3)

**Self-Review Status:** PASSED

---

## Investigation History

**2025-12-28 15:30:** Investigation started
- Initial question: Why does OpenCode return redirect loop errors?
- Context: Issue has been "recurring for weeks"

**2025-12-28 15:35:** Found prior investigations
- 3+ prior investigations on this exact topic
- All reached same conclusion: expected behavior

**2025-12-28 15:40:** Root cause confirmed
- server.ts:2475-2482 has catch-all proxy to desktop.opencode.ai
- This is intentional design, not a bug

**2025-12-28 15:50:** Investigation completed
- Status: Complete
- Key outcome: This is the 4th investigation on the same "issue" - needs better knowledge surfacing
