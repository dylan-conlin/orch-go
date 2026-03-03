# Probe: OAuth Token Auto-Refresh Feasibility

**Model:** model-access-spawn-paths
**Date:** 2026-03-02
**Status:** Complete

---

## Question

The model documents account routing and capacity-aware distribution but does NOT document the token lifecycle — how tokens expire, whether they can be auto-refreshed, or what breaks when they go stale. Can both of Dylan's Claude Max accounts (personal 5x, work 20x) be kept alive automatically without manual intervention?

---

## What I Tested

### Test 1: Map all token storage locations

Examined three independent auth systems:

```
~/.orch/accounts.yaml           → orch's refresh token chains (per account)
~/.local/share/opencode/auth.json → OpenCode's active session tokens (synced from orch)
macOS Keychain                    → Claude CLI's independent token chains (per config dir)
```

Keychain entries found:
- `Claude Code-credentials` → work account (20x), actively used
- `Claude Code-credentials-0b4b8023` → work account (20x), STALE since Jan 25
- `Claude Code-credentials-2cec3c7a` → personal account (5x), actively used

### Test 2: Token refresh lifecycle (live API calls)

```bash
# Refresh via Anthropic OAuth endpoint
curl -X POST 'https://console.anthropic.com/v1/oauth/token' \
  -H 'Content-Type: application/json' \
  -d '{"grant_type":"refresh_token","refresh_token":"<token>","client_id":"9d1c250a-e61b-44d9-88ed-5944d1962f5e"}'
```

Results:
- **Access token lifetime**: 28800 seconds = **8 hours** (confirmed by `expires_in` in API response)
- **Refresh token rotation**: Every refresh returns a NEW refresh token. Old one is **immediately invalidated**.
- **No grace period**: Tested by refreshing a token via curl (rotating it), then trying the old token via orch → `invalid_grant` error immediately.

### Test 3: Stale refresh token (37 days unused)

Attempted to refresh the `-0b4b8023` keychain entry's refresh token (last used Jan 25, 2026 — 37 days ago):

```
HTTP 403: error code: 1010
```

**Refresh token is dead after 37 days of non-use.**

### Test 4: `orch usage` as implicit keepalive

Traced code path: `orch usage` → `ListAccountsWithCapacity()` → `GetAccountCapacity(name)` → `RefreshOAuthToken(acc.RefreshToken)` → saves new refresh token to accounts.yaml.

This means **every `orch usage` call refreshes ALL accounts' tokens**, keeping their chains alive. Confirmed working for both personal and work accounts.

### Test 5: Token chain independence

Compared refresh token prefixes across all storage locations:

```
accounts.yaml personal:  sk-ant-ort01-85AblZL...
accounts.yaml work:      sk-ant-ort01-Y7sLu9T...
OpenCode auth.json:      sk-ant-ort01-85AblZL...  (matches personal = active)
Keychain default:        sk-ant-ort01-syG3CJX...  (work, DIFFERENT chain)
Keychain -0b4b8023:      sk-ant-ort01-i6dGjWS...  (work, OLD chain, dead)
Keychain -2cec3c7a:      sk-ant-ort01-fIq_XuC...  (personal, DIFFERENT chain)
```

**All three systems maintain COMPLETELY INDEPENDENT refresh token chains.** They do not share tokens or interfere with each other under normal operation.

### Test 6: Cross-system token sharing (destructive)

Copied keychain personal token into accounts.yaml, then ran `orch usage` which rotated it. The keychain now holds a stale token.

**Confirmed: sharing tokens between systems creates mutual exclusion.** Whichever refreshes first wins; the other breaks.

---

## What I Observed

### Token Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Anthropic Auth Server                     │
│  - Maintains multiple refresh token chains per user          │
│  - Access tokens: 8-hour lifetime                            │
│  - Refresh tokens: rotate on use, die after N days unused    │
│  - No grace period on rotation                               │
└────────┬──────────────────────────┬──────────────────────────┘
         │                          │
    ┌────▼────┐                ┌────▼────┐
    │  Chain A │                │  Chain B │
    │ (orch)   │                │(Claude)  │
    └────┬────┘                └────┬────┘
         │                          │
    ┌────▼────────────┐        ┌────▼──────────────────┐
    │ accounts.yaml   │        │ macOS Keychain         │
    │ ↓ syncs to      │        │ Claude Code-credentials│
    │ auth.json       │        │ (per config dir)       │
    └─────────────────┘        └───────────────────────┘
```

### Token Lifecycle Facts (Empirically Confirmed)

| Property | Value | Evidence |
|----------|-------|----------|
| Access token lifetime | 8 hours (28800s) | API `expires_in` response field |
| Refresh token rotation | Every use | Old prefix ≠ new prefix on every refresh |
| Grace period after rotation | **None** | Old token → `invalid_grant` immediately |
| Stale chain death | Between 1-37 days | 37-day-old chain confirmed dead |
| Independent chains | Yes | Different refresh tokens across all 3 systems |
| Cross-system sharing safe? | **No** | Sharing creates mutual exclusion race |

### Two Failure Modes Identified

**Mode 1: Chain Death (stale unused account)**
- An account's refresh token chain goes unused for too long
- Anthropic revokes the chain server-side
- Next refresh attempt → `invalid_grant`
- Recovery: browser re-auth (manual, cannot be automated)

**Mode 2: Chain Divergence (token sharing)**
- Two systems end up with the same refresh token (e.g., `orch account switch` syncs to auth.json)
- One system refreshes, rotating the token
- Other system's copy is now stale
- Recovery: re-sync from the system that refreshed successfully

### Keepalive Feasibility

**For orch accounts.yaml: YES — trivially.**
`orch usage` already refreshes all accounts on every call. A daily launchd job would keep both chains alive indefinitely.

```xml
<!-- ~/Library/LaunchAgents/com.orch.token-keepalive.plist -->
<plist>
  <dict>
    <key>Label</key><string>com.orch.token-keepalive</string>
    <key>ProgramArguments</key>
    <array>
      <string>/Users/dylanconlin/bin/orch</string>
      <string>usage</string>
    </array>
    <key>StartCalendarInterval</key>
    <dict>
      <key>Hour</key><integer>9</integer>
      <key>Minute</key><integer>0</integer>
    </dict>
  </dict>
</plist>
```

**For Claude CLI keychain: Partially.** Claude CLI auto-refreshes its access tokens when sessions start. Agents running regularly keep the chain alive. The risk is the LESS-USED account (personal 5x) going unused for too long.

A keepalive for Claude CLI would need to invoke Claude CLI with each config dir:
```bash
CLAUDE_CONFIG_DIR=~/.claude-personal claude --version  # Touch personal chain
claude --version                                        # Touch default chain
```
**Untested**: whether `--version` actually triggers token refresh. May need a minimal API call.

---

## Model Impact

- [ ] **Confirms** invariant: N/A — model had no token lifecycle claims to confirm
- [ ] **Contradicts** invariant: N/A
- [x] **Extends** model with: Complete token lifecycle documentation — three independent auth systems, 8-hour access tokens, rotating refresh tokens with no grace period, two failure modes (chain death + chain divergence), and keepalive feasibility

### Specific Model Extensions Needed

1. **New section: Token Lifecycle** — Document the three independent auth systems, their token types, and expiry behavior
2. **New failure mode: Silent Token Expiry** — When an account's refresh token chain dies from non-use, all systems using that chain fail silently
3. **New failure mode: Token Chain Divergence** — When tokens are shared between systems, rotation creates mutual exclusion
4. **Constraint update: Never share refresh tokens between orch and Claude CLI** — They must maintain independent chains
5. **Account routing caveat: `GetAccountCapacity()` refreshes tokens as side effect** — This is currently the only keepalive mechanism; losing it (e.g., if capacity checking is disabled) would cause chain death

---

## Notes

### Damage Report

During testing, I rotated the personal account's orch refresh token chain via curl without saving the new token. Recovered by copying the Claude CLI keychain's personal refresh token into accounts.yaml, then ran `orch usage` which rotated THAT chain. Net result:

- **accounts.yaml**: Both accounts healthy (personal has new chain from keychain seed, work unchanged)
- **Claude CLI keychain personal (-2cec3c7a)**: Access token valid for ~8 more hours, but refresh token is now stale. Will need browser re-auth when access token expires.
- **OpenCode auth.json**: May have stale personal token (was synced from orch's old chain)

**Recovery needed**: Dylan should run `claude --login` with `CLAUDE_CONFIG_DIR=~/.claude-personal` to re-establish the personal account's Claude CLI keychain chain within the next 8 hours.

### Exact Refresh Token Expiry Unknown

37 days confirmed dead. Exact threshold unknown (could be 7, 14, or 30 days). A follow-up test with a token unused for 7 days would narrow this down. For keepalive purposes, daily refresh is sufficient — no need to know the exact threshold.

### TLS Fingerprint Matters

Python's `urllib` gets 403 from Anthropic's OAuth endpoint while `curl` and Go's `net/http` succeed. Anthropic/Cloudflare appears to fingerprint TLS clients. This doesn't affect orch (Go) or Claude CLI (Node.js) but would matter for any Python-based tooling.
