<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** `orch account add` was using the wrong OAuth authorization endpoint - `console.anthropic.com` instead of `claude.ai` for Claude Max/Pro subscriptions.

**Evidence:** OpenCode's `opencode-anthropic-auth@0.0.5` plugin uses `claude.ai/oauth/authorize` for Claude Max, while orch-go was using `console.anthropic.com/oauth/authorize`. The error message mentioned missing inference scope requirement.

**Knowledge:** Anthropic has two OAuth endpoints: `console.anthropic.com` (for API key creation) and `claude.ai` (for Claude Max/Pro subscription tokens with inference scope). The scopes were correct (`user:inference`), but the wrong endpoint doesn't grant the inference scope.

**Next:** Fix deployed, tests updated, ready for smoke test with real auth flow.

**Confidence:** High (90%) - Fix matches OpenCode's reference implementation exactly.

---

# Investigation: orch account add uses wrong OAuth authorization endpoint

**Question:** Why does `orch account add` fail with 'OAuth token does not meet scope requirement any_of(user:inference, user:ccr_inference, org:service_key_inference)' even though we request `user:inference` scope?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Scopes are identical between orch-go and OpenCode

**Evidence:** 
- orch-go (pkg/account/oauth.go:97): `"scope": {"org:create_api_key user:profile user:inference"}`
- OpenCode plugin (opencode-anthropic-auth@0.0.5): `url.searchParams.set("scope", "org:create_api_key user:profile user:inference")`

**Source:** 
- pkg/account/oauth.go:97
- npm package opencode-anthropic-auth@0.0.5 (index.mjs)

**Significance:** The scopes are NOT the problem. Both implementations request the exact same scopes.

---

### Finding 2: orch-go uses wrong authorization endpoint

**Evidence:** 
- orch-go uses: `https://console.anthropic.com/oauth/authorize`
- OpenCode plugin uses for Claude Max: `https://claude.ai/oauth/authorize`
- OpenCode plugin uses for API keys: `https://console.anthropic.com/oauth/authorize`

**Source:** 
- pkg/account/oauth.go:24: `AuthorizationEndpoint = "https://console.anthropic.com/oauth/authorize"`
- opencode-anthropic-auth plugin lines 10-16: Two modes with different endpoints

**Significance:** This is the root cause. `console.anthropic.com` is for API key creation, not for Claude Max subscription OAuth tokens with inference scope.

---

### Finding 3: OpenCode plugin has two authentication modes

**Evidence:** 
The OpenCode anthropic auth plugin supports two modes:
1. "Claude Pro/Max" mode - uses `claude.ai/oauth/authorize` - for subscription-based inference tokens
2. "Create an API Key" mode - uses `console.anthropic.com/oauth/authorize` - for API key creation

**Source:** 
opencode-anthropic-auth@0.0.5 index.mjs authorize() function and methods array

**Significance:** orch-go was incorrectly using the API key endpoint when it should use the Claude Max endpoint for subscription-based tokens.

---

## Synthesis

**Key Insights:**

1. **Domain matters for OAuth scope fulfillment** - Even with the same scopes requested, Anthropic grants different token capabilities based on which domain the OAuth flow uses.

2. **OpenCode's plugin is the reference implementation** - The `opencode-anthropic-auth` npm package shows the correct pattern for Claude Max OAuth.

3. **Simple one-line fix** - Changing `console.anthropic.com` to `claude.ai` in the authorization endpoint fixes the issue.

**Answer to Investigation Question:**

The tokens failed with missing inference scope because orch-go was using `console.anthropic.com/oauth/authorize` (for API key creation) instead of `claude.ai/oauth/authorize` (for Claude Max subscription tokens). The scopes were correctly requested (`user:inference`) but the wrong endpoint doesn't grant inference capability.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

The fix directly matches OpenCode's reference implementation. The root cause analysis is complete and the code change is minimal and isolated.

**What's certain:**

- ✅ Scopes match between orch-go and OpenCode (verified by code comparison)
- ✅ Authorization endpoint differs (verified by code comparison)
- ✅ OpenCode plugin uses `claude.ai` for Claude Max mode (verified by package inspection)

**What's uncertain:**

- ⚠️ Cannot fully verify without completing actual OAuth flow (requires interactive browser auth)
- ⚠️ Token endpoint differences (both use console.anthropic.com for token exchange - may be fine)

**What would increase confidence to Very High:**

- Complete actual OAuth flow with `claude.ai` endpoint
- Verify resulting token works for inference

---

## Implementation Recommendations

### Recommended Approach ⭐

**Change AuthorizationEndpoint to claude.ai** - Single constant change

**Why this approach:**
- Matches OpenCode's reference implementation exactly
- Minimal change, low risk
- Directly addresses root cause

**Trade-offs accepted:**
- Only supports Claude Max (not API key creation via console.anthropic.com)
- Acceptable since orch is designed for Claude Max accounts

**Implementation sequence:**
1. Change AuthorizationEndpoint constant from `console.anthropic.com` to `claude.ai`
2. Update test that validates authorization URL host
3. Test with actual OAuth flow

---

## References

**Files Examined:**
- pkg/account/oauth.go:24-28 - Authorization constants
- pkg/account/oauth_test.go:62-64 - Host validation test

**Commands Run:**
```bash
# Downloaded and inspected OpenCode anthropic auth plugin
npm pack opencode-anthropic-auth
tar -xzf opencode-anthropic-auth-0.0.5.tgz
cat package/index.mjs
```

**External Documentation:**
- opencode-anthropic-auth@0.0.5 npm package - Reference implementation

**Related Artifacts:**
- None

---

## Investigation History

**2025-12-26 10:00:** Investigation started
- Initial question: Why do tokens from `orch account add` fail with missing inference scope?
- Context: User reported `opencode auth login` works but `orch account add` fails

**2025-12-26 10:05:** Found scopes are identical
- Both orch-go and OpenCode request the same OAuth scopes

**2025-12-26 10:08:** Found root cause - wrong authorization endpoint
- orch-go uses console.anthropic.com, OpenCode uses claude.ai for Claude Max

**2025-12-26 10:12:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Changed authorization endpoint from console.anthropic.com to claude.ai
