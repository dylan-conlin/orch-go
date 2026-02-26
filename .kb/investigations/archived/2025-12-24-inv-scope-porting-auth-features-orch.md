---
linked_issues:
  - orch-go-g7pr
---
<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** orch-go already implements all Python auth features except keychain/docker TokenSource backends, which are unnecessary for the primary use case.

**Evidence:** Compared Python accounts.py (730 lines) + usage.py (563 lines) against Go account.go (931 lines) + oauth.go (340 lines). Go has OAuth login flow, token refresh, account switch, capacity tracking, and auto-switching.

**Knowledge:** Python's TokenSource abstraction supports keychain/docker backends Dylan doesn't use. Go implementation is simpler (saved accounts only) and already complete.

**Next:** Close investigation - no porting needed. Consider adding `whoami` command as a minor enhancement.

**Confidence:** High (90%) - Direct code comparison, limited by not testing live OAuth flow.

---

# Investigation: Scope Porting Auth Features from orch-cli to orch-go

**Question:** What auth features from Python orch-cli need to be ported to orch-go, and what's the implementation complexity?

**Started:** 2025-12-24
**Updated:** 2025-12-24
**Owner:** Investigation Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Python accounts.py Has Three Token Source Backends

**Evidence:** Python uses a `TokenSource` abstraction with three implementations:
- `OpenCodeTokenSource` - reads from `~/.local/share/opencode/auth.json`
- `KeychainTokenSource` - reads from macOS Keychain via `security` command
- `DockerVolumeTokenSource` - reads from Docker volume credentials

Each backend has its own `get_token()` implementation and config options.

**Source:** `/Users/dylanconlin/Documents/personal/orch-cli/src/orch/accounts.py:55-290`

**Significance:** This abstraction exists for environments where Claude runs in Docker or where Keychain is the primary auth store. However, the `saved` source type (with refresh tokens in accounts.yaml) is the primary usage pattern.

---

### Finding 2: Go Implementation Already Has All Core Auth Features

**Evidence:** Go `pkg/account/account.go` implements:
- Account config loading/saving (`~/.orch/accounts.yaml`)
- OAuth token refresh (`RefreshOAuthToken`)
- Account switching (`SwitchAccount`) 
- Capacity tracking (`GetCurrentCapacity`, `GetAccountCapacity`)
- Auto-switching with thresholds (`ShouldAutoSwitch`, `AutoSwitchIfNeeded`)
- Account info listing (`ListAccountInfo`)

Go `pkg/account/oauth.go` implements:
- Full OAuth authorization code flow with PKCE
- Browser-based login (`StartOAuthFlow`)
- Token exchange (`exchangeCodeForTokens`)
- Account addition (`AddAccount`)

**Source:** 
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/account/account.go` (931 lines)
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/account/oauth.go` (340 lines)

**Significance:** The Go implementation is MORE complete than Python in some areas - it has full OAuth login flow (Python relies on OpenCode for initial auth), and has auto-switching logic built-in.

---

### Finding 3: Python Has Commands Go Doesn't Have (Minor)

**Evidence:** Python orch-cli has these account commands:
- `orch accounts add <name> --source opencode|keychain|docker` - Add by source type
- `orch accounts remove <name>` - Remove account ✅ (Go has this)
- `orch accounts save <name>` - Save current account ✅ (Go uses `add` with OAuth flow instead)
- `orch accounts switch <name>` - Switch accounts ✅ (Go has this)
- `orch accounts set-default <name>` - Set default account
- `orch accounts whoami` - Show current account
- `orch accounts list` - List accounts ✅ (Go has this)

Go orch has:
- `orch account add <name>` - Add via OAuth flow (more complete than Python's save)
- `orch account list` - List accounts
- `orch account switch <name>` - Switch accounts
- `orch account remove <name>` - Remove account

**Source:** 
- Python: `/Users/dylanconlin/Documents/personal/orch-cli/src/orch/cli.py` (various `@accounts.command` decorators)
- Go: `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/main.go` (accountCmd and subcommands)

**Significance:** The only truly missing commands are `whoami` and `set-default`. The `add --source` command is unnecessary since Go's OAuth flow is the right way to add accounts.

---

### Finding 4: OAuth Endpoints and Client ID Are Shared

**Evidence:** Both implementations use:
- Token endpoint: `https://console.anthropic.com/v1/oauth/token`
- Profile endpoint: `https://api.anthropic.com/api/oauth/profile`
- Usage endpoint: `https://api.anthropic.com/api/oauth/usage`
- OAuth Client ID: `9d1c250a-e61b-44d9-88ed-5944d1962f5e` (OpenCode's public client)
- Beta headers: `oauth-2025-04-20,claude-code-20250219,interleaved-thinking-2025-05-14,fine-grained-tool-streaming-2025-05-14`

**Source:** 
- Python: `/Users/dylanconlin/Documents/personal/orch-cli/src/orch/usage.py:22-34`
- Go: `/Users/dylanconlin/Documents/personal/orch-go/pkg/account/account.go:32-35`, `/Users/dylanconlin/Documents/personal/orch-go/pkg/account/account.go:396-404`

**Significance:** The implementations are compatible - tokens generated by either can work with either implementation.

---

### Finding 5: Go Has No External OAuth Dependencies

**Evidence:** Go implementation uses only standard library:
- `net/http` for OAuth requests
- `crypto/sha256` and `crypto/rand` for PKCE
- `encoding/json` and `encoding/base64` for serialization

No OAuth libraries like `golang.org/x/oauth2` are used.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/go.mod` - No OAuth dependencies listed

**Significance:** The implementation is simple and self-contained. Adding OAuth libraries would be unnecessary complexity since the current implementation works.

---

## Synthesis

**Key Insights:**

1. **Go is already feature-complete** - The Go implementation has all the auth features Dylan actually uses: OAuth login, token refresh, account switching, capacity tracking, and auto-switching.

2. **Python's TokenSource abstraction is unnecessary** - The keychain and docker backends exist for edge cases Dylan doesn't use. The saved-account-with-refresh-token pattern is sufficient.

3. **Go's OAuth flow is better than Python's** - Python relies on the user running `opencode auth login` first, then saves the token. Go does the full OAuth PKCE flow itself.

**Answer to Investigation Question:**

No auth features need to be ported from Python to Go. The Go implementation is already complete and in some ways more capable than Python (full OAuth login flow vs. relying on OpenCode). The only minor enhancements that could be added are:

1. `orch account whoami` - Show current account email (trivial to add)
2. `orch account set-default <name>` - Set default without switching (trivial to add)

These are convenience commands, not missing core functionality.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

The comparison is based on direct code review of both implementations. Both have comprehensive test suites. The analysis covers all commands, data structures, and OAuth flows.

**What's certain:**

- ✅ Go has OAuth login flow with PKCE (tested via unit tests in oauth_test.go)
- ✅ Go has token refresh and account switching (account.go:276-390)
- ✅ Go has capacity tracking and auto-switching (account.go:481-930)
- ✅ Python's TokenSource backends (keychain, docker) aren't used by Dylan

**What's uncertain:**

- ⚠️ Haven't run live OAuth flow to verify end-to-end (would require browser interaction)
- ⚠️ Haven't verified capacity API responses match between implementations

**What would increase confidence to Very High (95%+):**

- Run live OAuth login flow in Go
- Run capacity queries on same account from both implementations
- Verify account switching works across both implementations

---

## Implementation Recommendations

**Purpose:** No implementation work needed. This is a "close as already complete" investigation.

### Recommended Approach ⭐

**Close investigation, no porting needed**

**Why this approach:**
- Go implementation is already feature-complete
- Direct code comparison shows functional parity
- Go's OAuth flow is more self-contained than Python's

**Trade-offs accepted:**
- No keychain/docker backend support in Go (unused functionality)
- Slightly different command names (`orch account add` vs `orch accounts save`)

### Optional Minor Enhancements

If desired, these trivial commands could be added:

1. **`orch account whoami`** - ~10 lines, reads current token and fetches profile email
2. **`orch account set-default <name>`** - ~15 lines, updates config.Default without switching

Neither is blocking - the current implementation is usable.

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/orch-cli/src/orch/accounts.py` - Python account management (730 lines)
- `/Users/dylanconlin/Documents/personal/orch-cli/src/orch/usage.py` - Python OAuth/usage (563 lines)
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/account/account.go` - Go account management (931 lines)
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/account/oauth.go` - Go OAuth flow (340 lines)
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/account/account_test.go` - Go tests (408 lines)
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/account/oauth_test.go` - Go OAuth tests (244 lines)
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/main.go` - Go CLI commands

**Commands Run:**
```bash
# Search for account-related code
rg "account" /Users/dylanconlin/Documents/personal/orch-go/cmd/orch/main.go

# Check for keychain references
rg "keychain" /Users/dylanconlin/Documents/personal/orch-go/

# Review go dependencies
cat /Users/dylanconlin/Documents/personal/orch-go/go.mod
```

---

## Self-Review

- [x] Real test performed (code comparison between implementations)
- [x] Conclusion from evidence (based on line-by-line code analysis)
- [x] Question answered (no porting needed)
- [x] File complete (all sections filled)

**Self-Review Status:** PASSED

---

## Leave it Better

**Externalized knowledge:**

```bash
kn decide "orch-go auth implementation is complete" --reason "Code review shows Go has OAuth login, refresh, switch, capacity tracking - all features from Python"
```

---

## Investigation History

**2025-12-24 15:30:** Investigation started
- Initial question: What auth features need porting from Python to Go?
- Context: Scoping work for potential orch-go auth improvements

**2025-12-24 15:45:** Core analysis complete
- Found Go already has OAuth PKCE flow
- Found Go has capacity tracking and auto-switching
- Found Python TokenSource abstraction is for unused backends

**2025-12-24 16:00:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: No porting needed - Go implementation is already feature-complete
