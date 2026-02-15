<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The `orch account add` OAuth flow was using a local callback server which Anthropic doesn't allow - they only permit their official callback URL.

**Evidence:** Compared current implementation with opencode-anthropic-auth plugin reference showing redirect_uri must be `https://console.anthropic.com/oauth/code/callback` and code is displayed for user to paste.

**Knowledge:** Anthropic OAuth requires using their official callback URL; the code is displayed on their page and must be manually pasted. Code may include state appended after `#` separator.

**Next:** Smoke test the `orch account add` command with a real Anthropic account to verify the full flow works.

**Confidence:** High (90%) - All tests pass, matches working reference implementation, but real OAuth flow not tested.

---

# Investigation: Fix orch account add OAuth flow

**Question:** Why does `orch account add` fail, and how can we fix it to use Anthropic's allowed OAuth callback URL?

**Started:** 2025-12-22
**Updated:** 2025-12-22
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Local callback server not permitted by Anthropic

**Evidence:** The original implementation used `http://127.0.0.1:19283/callback` as redirect_uri, but Anthropic only whitelists their own callback URL `https://console.anthropic.com/oauth/code/callback`.

**Source:** 
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/account/oauth.go` (original)
- `/tmp/auth-check/package/index.mjs:19-20` (reference implementation)

**Significance:** This was the root cause of the OAuth flow failing - Anthropic rejects any redirect_uri that isn't their official callback URL.

---

### Finding 2: OpenCode plugin uses code-paste flow

**Evidence:** The opencode-anthropic-auth plugin reference shows the correct pattern:
1. Set `redirect_uri` to `https://console.anthropic.com/oauth/code/callback`
2. Include `code=true` parameter in auth URL
3. Prompt user to paste the authorization code displayed by Anthropic
4. Exchange code for tokens via JSON POST to token endpoint
5. Code may include state appended after `#` (format: `code#state`)

**Source:** `/tmp/auth-check/package/index.mjs:11-65` - authorize() and exchange() functions

**Significance:** This provided the working reference pattern to follow.

---

### Finding 3: Token exchange requires JSON body, not form-urlencoded

**Evidence:** The reference implementation uses JSON Content-Type and body for the token exchange, while the original implementation used form-urlencoded.

**Source:** `/tmp/auth-check/package/index.mjs:41-53` - exchange() function uses JSON.stringify for body

**Significance:** Ensures compatibility with Anthropic's token endpoint.

---

## Synthesis

**Key Insights:**

1. **Anthropic controls the callback flow** - Unlike typical OAuth where apps run local servers, Anthropic displays the auth code on their own page and expects users to manually copy it.

2. **State parameter serves as verifier** - The reference uses the PKCE code_verifier as the state parameter, simplifying the flow.

3. **Code format includes state** - The authorization code from Anthropic may include the state after a `#` separator, requiring parsing.

**Answer to Investigation Question:**

The `orch account add` OAuth flow failed because Anthropic only allows their official callback URL (`https://console.anthropic.com/oauth/code/callback`). The fix involved:
1. Removing the local callback server
2. Using Anthropic's callback URL as redirect_uri
3. Adding `code=true` parameter to signal code display mode
4. Prompting user to paste the authorization code
5. Updating token exchange to use JSON body format

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

The implementation follows the exact pattern from the working opencode-anthropic-auth plugin reference. All unit tests pass and the code compiles. The only gap is lack of end-to-end testing with a real Anthropic account.

**What's certain:**

- ✅ The redirect_uri now uses Anthropic's official callback URL
- ✅ The authorization URL includes `code=true` parameter
- ✅ Code parsing handles the `code#state` format correctly
- ✅ All unit tests pass

**What's uncertain:**

- ⚠️ Real end-to-end OAuth flow not tested (requires browser interaction and Anthropic credentials)

**What would increase confidence to Very High (95%+):**

- Manual end-to-end test of `orch account add` with a real Anthropic account

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Code-paste OAuth flow** - Replace local callback server with manual code paste flow

**Why this approach:**
- Matches Anthropic's official OAuth requirements
- Proven pattern from working opencode-anthropic-auth plugin
- Simpler than local server (no port conflicts, no networking)

**Trade-offs accepted:**
- Requires manual copy/paste step from user
- Less seamless UX than automatic callback

**Implementation sequence:**
1. Remove local callback server code - eliminates unused complexity
2. Use Anthropic's callback URL as redirect_uri
3. Prompt user to paste authorization code from browser
4. Parse code#state format if present
5. Exchange code for tokens using JSON body

### Alternative Approaches Considered

**Option B: Keep local server, try different ports**
- **Pros:** More seamless UX
- **Cons:** Won't work - Anthropic doesn't allow custom redirect URIs
- **When to use instead:** Never - this is blocked by Anthropic's OAuth policy

---

### Implementation Details

**What was implemented:**
- Changed `AnthropicCallbackURL` constant to `https://console.anthropic.com/oauth/code/callback`
- Removed local callback server and related code
- Added `code=true` parameter to authorization URL
- Changed buildAuthorizationURL signature to use codeVerifier as state
- Updated token exchange to use JSON body format
- Added code#state parsing in exchangeCodeForTokens
- Updated prompts to guide user through paste flow

**Success criteria:**
- ✅ All unit tests pass
- ✅ Code compiles without errors
- ⏳ End-to-end test with real account (requires manual verification)

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/account/oauth.go` - Original OAuth implementation
- `/tmp/auth-check/package/index.mjs` - Working reference implementation
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/account/account.go` - Account constants

**Commands Run:**
```bash
# Run account package tests
go test ./pkg/account/... -v

# Build binary to verify compilation
go build -o /tmp/orch-test ./cmd/orch/...
```

**External Documentation:**
- opencode-anthropic-auth plugin - Reference implementation for OAuth flow

---

## Investigation History

**2025-12-22:** Investigation started
- Initial question: Why does `orch account add` fail with local callback server?
- Context: Anthropic only allows their own callback URL

**2025-12-22:** Root cause identified
- Found reference implementation in opencode-anthropic-auth plugin
- Confirmed Anthropic requires their official callback URL

**2025-12-22:** Implementation completed
- Rewrote OAuth flow to use code-paste pattern
- All tests pass, code compiles
- Final confidence: High (90%)
- Status: Complete
- Key outcome: OAuth flow now uses Anthropic's official callback URL with manual code paste
