**TLDR:** Implemented OAuth token refresh for `orch account switch` command. The command now successfully exchanges refresh tokens via Anthropic's OAuth API and updates both ~/.orch/accounts.yaml and OpenCode's auth.json. High confidence (95%) - manually tested switching between two accounts.

---

# Investigation: Implement Account Switch with OAuth Token Refresh

**Question:** How to implement account switching that refreshes OAuth tokens and updates OpenCode auth?

**Started:** 2025-12-20
**Updated:** 2025-12-20
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (95%)

---

## Findings

### Finding 1: Python orch implementation uses Anthropic OAuth token endpoint

**Evidence:** Python orch-cli uses `https://console.anthropic.com/v1/oauth/token` with OpenCode's public client ID (`9d1c250a-e61b-44d9-88ed-5944d1962f5e`) to exchange refresh tokens for new access/refresh tokens.

**Source:** `/Users/dylanconlin/Documents/personal/orch-cli/src/orch/usage.py:24-25, 354-408`

**Significance:** Provides exact API contract to implement in Go.

---

### Finding 2: Token exchange flow requires POST with JSON body

**Evidence:** Request format:

```json
{
  "grant_type": "refresh_token",
  "refresh_token": "<saved_refresh_token>",
  "client_id": "<opencode_client_id>"
}
```

Response includes `access_token`, `refresh_token`, and `expires_in` (seconds).

**Source:** `/Users/dylanconlin/Documents/personal/orch-cli/src/orch/usage.py:376-408`

**Significance:** Direct translation to Go HTTP client implementation.

---

### Finding 3: Account switch updates both accounts.yaml and auth.json

**Evidence:** Python implementation:

1. Loads account from ~/.orch/accounts.yaml
2. Calls token refresh API
3. Updates accounts.yaml with new refresh token
4. Writes new tokens to ~/.local/share/opencode/auth.json

**Source:** `/Users/dylanconlin/Documents/personal/orch-cli/src/orch/accounts.py:602-668`

**Significance:** Required behavior for Go implementation.

---

## Synthesis

**Key Insights:**

1. **Direct API port** - OAuth flow is straightforward HTTP POST, easily ported to Go's net/http client.

2. **Two-file update** - Must update both the saved account config and OpenCode's auth file for complete account switch.

3. **Refresh token rotation** - API returns new refresh token each time, must persist for future switches.

**Answer to Investigation Question:**

Implemented by adding `RefreshOAuthToken()` and `SwitchAccount()` functions to pkg/account/account.go, then updating cmd/orch/main.go to use the new `SwitchAccount()` function instead of the stub.

---

## Confidence Assessment

**Current Confidence:** Very High (95%)

**Why this level?**

Manually tested switching between personal and work accounts. Both succeeded and updated both config files correctly.

**What's certain:**

- ✅ OAuth token exchange works with Anthropic's endpoint
- ✅ accounts.yaml is updated with new refresh token
- ✅ auth.json is updated with all token fields
- ✅ Error handling for missing accounts works

**What's uncertain:**

- ⚠️ Behavior when refresh token is expired/revoked (not tested)

**What would increase confidence to 100%:**

- Test with expired/revoked refresh tokens
- Long-running usage to verify token rotation continues working

---

## Implementation Summary

**Files modified:**

- `pkg/account/account.go` - Added OAuth constants, TokenInfo struct, RefreshOAuthToken(), SwitchAccount()
- `cmd/orch/main.go` - Updated runAccountSwitch() to use account.SwitchAccount()

**Commands Run:**

```bash
# Build verification
go build ./pkg/account/...
go build ./cmd/orch/...

# Manual testing
go run ./cmd/orch account list
go run ./cmd/orch account switch personal
go run ./cmd/orch account switch work
go run ./cmd/orch account switch nonexistent  # error case
```

---

## Investigation History

**2025-12-20 22:40:** Investigation started

- Initial question: How to implement account switch with OAuth token refresh

**2025-12-20 22:45:** Found Python implementation

- Located OAuth constants and token refresh function in orch-cli

**2025-12-20 22:50:** Implementation complete

- Added RefreshOAuthToken() and SwitchAccount() to Go package
- Updated command to use new functions
- Manual testing passed
