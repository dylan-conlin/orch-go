<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** OpenCode local server proxies unknown routes to desktop.opencode.ai, which returns HTML web app causing redirect loops - this is expected OpenCode behavior, not a bug.

**Evidence:** Testing confirmed /session and /session/{id}/prompt_async work (200/204), while /health, /message/{id}, and root /prompt_async fail (500) because they're proxied upstream.

**Knowledge:** The correct API path for sending prompts is `/session/{id}/prompt_async`, not `/prompt_async` at root - orch-go already uses the correct path.

**Next:** Close as non-issue - orch-go uses correct endpoints; /health endpoint failure is OpenCode upstream behavior, not actionable.

**Confidence:** High (90%) - Reproduced consistently, tested multiple endpoints, verified orch-go client code uses correct paths.

---

# Investigation: OpenCode API Redirect Loop

**Question:** Why does OpenCode API return 'The response redirected too many times' on /health and prompt_async endpoints while session listing works fine?

**Started:** 2025-12-23
**Updated:** 2025-12-23
**Owner:** Debug agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: OpenCode proxies unknown routes to desktop.opencode.ai

**Evidence:** 
- `/health` returns 500 with error: "The response redirected too many times"
- `/prompt_async` (root) returns 500 with: "UnexpectedRedirect fetching https://desktop.opencode.ai/prompt_async"
- `curl https://desktop.opencode.ai/health` returns HTML web app, not API response

**Source:** 
```bash
curl -v http://127.0.0.1:4096/health  # Returns 500 with redirect error
curl -sL https://desktop.opencode.ai/health  # Returns HTML web app
```

**Significance:** The local OpenCode server acts as a proxy - routes it doesn't handle locally are forwarded to desktop.opencode.ai, which is a web app that returns HTML and causes auth redirects.

---

### Finding 2: Session and prompt endpoints work when using correct paths

**Evidence:**
| Endpoint | Status | Notes |
|----------|--------|-------|
| `GET /session` | 200 | Works - lists sessions |
| `GET /session/{id}` | 200 | Works - gets session details |
| `POST /session/{id}/prompt_async` | 204 | Works - sends prompt correctly |
| `GET /health` | 500 | Fails - proxied upstream |
| `GET /message/{id}` | 500 | Fails - proxied upstream |
| `POST /prompt_async` (root) | 500 | Fails - wrong path |

**Source:**
```bash
curl -s -o /dev/null -w "%{http_code}" "http://127.0.0.1:4096/session"  # 200
curl -s -o /dev/null -w "%{http_code}" -X POST "http://127.0.0.1:4096/session/{id}/prompt_async" ...  # 204
```

**Significance:** The working endpoints are all under `/session/` prefix. Routes outside this pattern get proxied upstream.

---

### Finding 3: orch-go client already uses correct API paths

**Evidence:** Reviewed `pkg/opencode/client.go`:
- `SendMessageAsync()` at line 177 uses `/session/{id}/prompt_async` - correct path
- `ListSessions()` uses `/session` - works
- `GetSession()` uses `/session/{id}` - works
- No code uses `/health` or root `/prompt_async`

**Source:** `pkg/opencode/client.go:177`

**Significance:** The original bug report mentioned `/health` and `prompt_async` failing, but orch-go doesn't use these endpoints. The issue was likely confusion about which paths to use.

---

## Synthesis

**Key Insights:**

1. **Proxy architecture** - OpenCode local server handles `/session/*` routes locally and proxies everything else to desktop.opencode.ai

2. **Path matters** - `/session/{id}/prompt_async` works, but `/prompt_async` at root fails because it's not a local route

3. **No orch-go bug** - The orch-go client uses correct paths and is unaffected by this behavior

**Answer to Investigation Question:**

The redirect loop occurs because OpenCode's local server proxies unhandled routes (like `/health` and root `/prompt_async`) to desktop.opencode.ai, which is a web app that triggers auth redirects. Session listing works because `/session` is handled locally. This is expected OpenCode behavior, not a bug. The orch-go client already uses the correct paths (`/session/{id}/prompt_async`) and functions correctly.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Clear reproduction of the issue with consistent behavior across multiple test runs. The pattern is unambiguous: `/session/*` routes work, others fail.

**What's certain:**

- ✅ `/session` and `/session/{id}/*` endpoints work correctly
- ✅ `/health` and root `/prompt_async` fail due to upstream proxy
- ✅ orch-go client code uses correct paths

**What's uncertain:**

- ⚠️ Whether OpenCode intends to add local /health endpoint in future
- ⚠️ Whether there are other undocumented local endpoints

**What would increase confidence to Very High:**

- Confirmation from OpenCode documentation about routing behavior
- Testing after OpenCode version update

---

## Implementation Recommendations

### Recommended Approach ⭐

**No changes needed** - orch-go already uses correct API paths.

**Why this approach:**
- orch-go client code verified to use `/session/{id}/prompt_async`
- All session-related operations work correctly
- /health is not used by orch-go

**Trade-offs accepted:**
- Cannot use /health for server availability checks
- Must use session listing as proxy for server availability

### Alternative Approaches Considered

**Option B: Add /health check via session listing**
- **Pros:** Provides server availability check
- **Cons:** Not needed currently, adds complexity
- **When to use instead:** If health checks become required

---

## References

**Files Examined:**
- `pkg/opencode/client.go` - Verified API paths used by orch-go client

**Commands Run:**
```bash
# Test session endpoints
curl -s -o /dev/null -w "%{http_code}" "http://127.0.0.1:4096/session"  # 200

# Test prompt_async with correct path
curl -s -o /dev/null -w "%{http_code}" -X POST "http://127.0.0.1:4096/session/{id}/prompt_async" ...  # 204

# Test failing endpoints
curl -v http://127.0.0.1:4096/health  # 500 redirect error

# Verify upstream returns HTML
curl -sL https://desktop.opencode.ai/health  # Returns HTML web app
```

---

## Investigation History

**2025-12-23 16:30:** Investigation started
- Initial question: Why do /health and prompt_async return redirect errors?
- Context: Spawned from beads issue orch-go-16vb

**2025-12-23 16:32:** Root cause identified
- OpenCode proxies unknown routes to desktop.opencode.ai
- Upstream is web app, not API, causing redirect loops

**2025-12-23 16:35:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Non-issue for orch-go - correct paths already used
