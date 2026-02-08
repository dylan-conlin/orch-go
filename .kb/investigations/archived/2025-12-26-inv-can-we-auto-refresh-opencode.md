<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** OpenCode already implements automatic OAuth token refresh in the anthropic-auth plugin - no changes needed in orch.

**Evidence:** Extracted opencode-anthropic-auth@0.0.5 plugin code showing auto-refresh logic at fetch time: when `auth.expires < Date.now()`, it calls `https://console.anthropic.com/v1/oauth/token` with `grant_type: refresh_token` and updates stored tokens.

**Knowledge:** The "redirected too many times" error is NOT caused by token expiry - it's caused by hitting incorrect API endpoints that get proxied to desktop.opencode.ai (a web app). Token refresh is handled transparently by OpenCode.

**Next:** No action needed - OpenCode handles auto-refresh. Document this finding for future reference when auth issues arise.

**Confidence:** High (90%) - Code inspection confirms implementation; prior investigation (orch-go-16vb) already resolved the redirect error as unrelated to auth.

---

# Investigation: Can We Auto-Refresh OpenCode OAuth Tokens?

**Question:** Can we auto-refresh OpenCode OAuth tokens programmatically to prevent daemon stalls when tokens expire or when Anthropic returns redirect errors?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** Investigation agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: OpenCode already implements auto-refresh in the anthropic-auth plugin

**Evidence:** The opencode-anthropic-auth plugin (version 0.0.5) contains token refresh logic in its custom fetch wrapper:

```javascript
// From opencode-anthropic-auth/index.mjs
async fetch(input, init) {
  const auth = await getAuth();
  if (auth.type !== "oauth") return fetch(input, init);
  if (!auth.access || auth.expires < Date.now()) {
    const response = await fetch(
      "https://console.anthropic.com/v1/oauth/token",
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          grant_type: "refresh_token",
          refresh_token: auth.refresh,
          client_id: CLIENT_ID,  // 9d1c250a-e61b-44d9-88ed-5944d1962f5e
        }),
      },
    );
    if (!response.ok) {
      throw new Error(`Token refresh failed: ${response.status}`);
    }
    const json = await response.json();
    await client.auth.set({
      path: { id: "anthropic" },
      body: {
        type: "oauth",
        refresh: json.refresh_token,
        access: json.access_token,
        expires: Date.now() + json.expires_in * 1000,
      },
    });
    auth.access = json.access_token;
  }
  // ... continues with request using auth.access
}
```

**Source:** 
- `npm pack opencode-anthropic-auth@0.0.5` -> extracted index.mjs
- Plugin loaded in `packages/opencode/src/plugin/index.ts:32` as default plugin

**Significance:** This means token refresh is handled transparently at the SDK level, before any API request is made. Orch doesn't need to implement refresh logic - it's already handled by OpenCode.

---

### Finding 2: The "redirected too many times" error is NOT caused by token expiry

**Evidence:** Prior investigation (orch-go-16vb, closed 2025-12-25) determined:
- The redirect error occurs when hitting incorrect API paths (e.g., `/health`, root `/prompt_async`)
- OpenCode's local server proxies unknown routes to `desktop.opencode.ai` (a web app, not API)
- The web app returns HTML and triggers auth redirects, causing the loop
- Correct paths like `/session/{id}/prompt_async` work correctly

**Source:** `.kb/investigations/2025-12-23-debug-opencode-api-redirect-loop.md`

**Significance:** The redirect error symptom was misattributed to auth issues. It's actually a routing issue - using wrong API endpoints. Token expiry would result in a different error (401 Unauthorized or token refresh failure).

---

### Finding 3: Current token is valid with ~7 hours remaining

**Evidence:** 
```bash
$ cat ~/.local/share/opencode/auth.json
{
  "anthropic": {
    "type": "oauth",
    "refresh": "sk-ant-ort01-...",
    "access": "sk-ant-oat01-...",
    "expires": 1766803342685  # ~7 hours from investigation time
  }
}
```

Token structure matches what the plugin expects:
- `type: "oauth"` - correct auth type
- `refresh` - refresh token present for auto-refresh
- `access` - current access token
- `expires` - milliseconds since epoch, checked against `Date.now()`

**Source:** Direct inspection of `~/.local/share/opencode/auth.json`

**Significance:** Tokens are stored correctly and the refresh mechanism has all required data. When `expires < Date.now()`, the plugin will automatically refresh before making API calls.

---

### Finding 4: Anthropic's OAuth refresh endpoint is functional

**Evidence:** Tested the endpoint directly:
```bash
$ curl -s -X POST "https://console.anthropic.com/v1/oauth/token" \
  -H "Content-Type: application/json" \
  -d '{"grant_type": "refresh_token", "refresh_token": "invalid", "client_id": "9d1c250a-e61b-44d9-88ed-5944d1962f5e"}'
{"error": "invalid_grant", "error_description": "Refresh token not found or invalid"}
```

The endpoint:
- Exists and responds correctly
- Returns proper error for invalid tokens
- Uses standard OAuth error format

**Source:** Direct API testing

**Significance:** The refresh infrastructure works. With a valid refresh token, OpenCode's plugin would successfully obtain new access tokens.

---

## Synthesis

**Key Insights:**

1. **Auto-refresh is already implemented** - OpenCode's anthropic-auth plugin handles token refresh transparently at fetch time. No work needed in orch.

2. **The redirect error was misdiagnosed** - Prior investigation confirmed the "redirected too many times" error is caused by hitting incorrect API endpoints (that get proxied to a web app), not by token expiry.

3. **Token storage is correct** - The auth.json file contains all required fields (`refresh`, `access`, `expires`) for the auto-refresh mechanism to work.

**Answer to Investigation Question:**

**No, we should NOT implement auto-refresh in orch** - OpenCode already handles this automatically. The opencode-anthropic-auth plugin checks token expiry before each API request and refreshes if needed. 

The "redirected too many times" error that prompted this investigation is NOT related to token expiry - it's caused by using incorrect API endpoints that get proxied to desktop.opencode.ai (a web app). This was already resolved in investigation orch-go-16vb.

If auth errors do occur, they would manifest as:
1. `Token refresh failed: {status}` - if refresh endpoint fails
2. 401 responses - if access token is invalid and refresh fails
3. NOT as redirect loops - that's a different issue entirely

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Code inspection definitively shows the auto-refresh implementation. Prior investigation provides strong evidence that redirect errors are unrelated to auth. However, we haven't observed actual token expiry and refresh in production.

**What's certain:**

- ✅ OpenCode plugin contains auto-refresh logic (code inspection)
- ✅ Refresh endpoint exists and responds correctly (API test)
- ✅ Token storage format matches plugin expectations (file inspection)
- ✅ Redirect errors are NOT caused by token expiry (prior investigation orch-go-16vb)

**What's uncertain:**

- ⚠️ Haven't observed actual token refresh in production
- ⚠️ Unknown failure modes if refresh token itself expires or is revoked
- ⚠️ Unknown if refresh token has limited lifetime

**What would increase confidence to Very High:**

- Observe actual token refresh happening (wait for current token to expire)
- Find documentation on refresh token lifetime
- Confirm error handling when refresh fails

---

## Implementation Recommendations

### Recommended Approach ⭐

**No changes to orch** - OpenCode handles token refresh automatically.

**Why this approach:**
- Token refresh is already implemented correctly in OpenCode
- Adding duplicate logic would create maintenance burden
- The problem (redirect errors) had a different root cause

**Trade-offs accepted:**
- Orch cannot preemptively refresh tokens
- Orch cannot distinguish auth failures from other errors easily

### Alternative Approaches Considered

**Option B: Implement token refresh in orch as backup**
- **Pros:** Defense in depth, could refresh proactively
- **Cons:** Duplicates OpenCode logic, could cause race conditions
- **When to use instead:** If OpenCode's refresh mechanism proves unreliable

**Option C: Add auth health checking to orch daemon**
- **Pros:** Could detect auth issues early
- **Cons:** No reliable /health endpoint in OpenCode (per prior investigation)
- **When to use instead:** If OpenCode adds proper health/auth status endpoint

---

## References

**Files Examined:**
- `packages/opencode/src/auth/index.ts` - Auth storage schema
- `packages/opencode/src/provider/auth.ts` - Provider auth wiring
- `packages/opencode/src/plugin/index.ts` - Plugin loading
- `opencode-anthropic-auth@0.0.5/index.mjs` - Auto-refresh implementation

**Commands Run:**
```bash
# Extract and read the anthropic auth plugin
npm pack opencode-anthropic-auth@0.0.5 --pack-destination /tmp
tar -xzf /tmp/opencode-anthropic-auth-0.0.5.tgz -C /tmp
cat /tmp/package/index.mjs

# Check current token status
cat ~/.local/share/opencode/auth.json

# Test refresh endpoint
curl -s -X POST "https://console.anthropic.com/v1/oauth/token" \
  -H "Content-Type: application/json" \
  -d '{"grant_type": "refresh_token", "refresh_token": "invalid", "client_id": "9d1c250a-e61b-44d9-88ed-5944d1962f5e"}'
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-23-debug-opencode-api-redirect-loop.md` - Explains redirect error root cause (not auth)
- **Beads Issue:** orch-go-16vb - Original redirect error investigation

---

## Self-Review

- [x] Real test performed (API endpoint tested, code inspected)
- [x] Conclusion from evidence (not speculation)
- [x] Question answered (no, don't implement - OpenCode handles it)
- [x] File complete (all sections filled)
- [x] D.E.K.N. filled

**Self-Review Status:** PASSED

---

## Leave it Better

```bash
kn decide "OpenCode handles OAuth auto-refresh via anthropic-auth plugin" --reason "Code inspection shows token check at fetch time - no orch implementation needed"
```

---

## Investigation History

**2025-12-26 18:34:** Investigation started
- Initial question: Can we auto-refresh OAuth tokens to prevent daemon stalls?
- Context: Spawned from beads issue orch-go-zz42

**2025-12-26 18:50:** Key finding: OpenCode already implements auto-refresh
- Extracted opencode-anthropic-auth plugin code
- Found transparent refresh in custom fetch wrapper

**2025-12-26 18:55:** Connected to prior investigation
- Redirect error was NOT auth-related (orch-go-16vb)
- Confirmed token storage format is correct

**2025-12-26 19:10:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: No action needed - OpenCode already handles token refresh automatically
