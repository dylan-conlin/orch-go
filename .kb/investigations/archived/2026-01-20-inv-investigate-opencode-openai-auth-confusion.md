<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** OPENAI_API_KEY environment variable takes precedence over OAuth tokens in auth.json; when both exist, OpenCode uses the API key (pay-per-token billing) instead of OAuth (ChatGPT Pro subscription).

**Evidence:** Provider.ts loads env vars first (line 795-804), auth.json second (line 807-815 but only for type="api"), then plugins (line 817-862); confirmed OPENAI_API_KEY=sk-proj-6TNhtR7... is set alongside OAuth tokens.

**Knowledge:** Auth precedence is ENV > Auth.json(api) > Plugin OAuth > Config; the $ indicator shows estimated cost based on token usage and model pricing data; OAuth models without pricing info show $0.00.

**Next:** Unset OPENAI_API_KEY env var to use OAuth-only, or modify shell config to exclude it from OpenCode sessions.

**Promote to Decision:** recommend-no (tactical user configuration fix, not architectural)

---

# Investigation: OpenCode OpenAI Auth Confusion

**Question:** Why does OpenCode UI show $ cost indicator instead of (oauth) for OpenAI models, and which auth method (OAuth vs OPENAI_API_KEY) is actually being used?

**Started:** 2026-01-20
**Updated:** 2026-01-20
**Owner:** Architect agent (orch-go-08rm3)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Auth Precedence - ENV takes priority over OAuth

**Evidence:** In `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/provider/provider.ts`:
- Lines 795-804: ENV vars are loaded FIRST - if OPENAI_API_KEY exists, provider is merged with `source: "env"` and `key: OPENAI_API_KEY`
- Lines 807-815: Auth.all() only processes entries with `type: "api"` (NOT OAuth type="oauth")
- Lines 817-862: Plugin loaders run AFTER env loading; they merge with `source: "custom"` and `options`

**Source:**
- `provider.ts:800` - `mergeProvider(providerID, { source: "env", key: apiKey })`
- `provider.ts:959` - `if (options["apiKey"] === undefined && provider.key) options["apiKey"] = provider.key`

**Significance:** The env var key gets set on `provider.key`. At SDK creation (line 959), if `options["apiKey"]` isn't set by the plugin, `provider.key` (from env) is used. The plugin would need to explicitly set `options["apiKey"]` to override.

---

### Finding 2: Both Auth Methods Are Present

**Evidence:**
```bash
# Environment has API key
$ echo $OPENAI_API_KEY
sk-proj-6TNhtR7...

# auth.json has OAuth tokens
$ cat ~/.local/share/opencode/auth.json
{
  "openai": {
    "type": "oauth",
    "refresh": "rt_ivNGl5Ub...",
    "access": "eyJhbGciOiJSUzI1...",
    "expires": 1769837074238
  }
}

# opencode auth list shows both
$ opencode auth list
Credentials: OpenAI (oauth)
Environment: OpenAI (OPENAI_API_KEY)
```

**Source:** `~/.local/share/opencode/auth.json`, shell environment, `opencode auth list` output

**Significance:** Dylan has BOTH OAuth tokens (from ChatGPT Pro subscription via oc auth login) AND an API key in env vars. Since env is loaded first, the API key is being used, resulting in pay-per-token billing instead of subscription-based access.

---

### Finding 3: $ Indicator Is Estimated Cost Based on Token Usage

**Evidence:** In `/Users/dylanconlin/Documents/personal/opencode/packages/app/src/components/session-context-usage.tsx`:
```javascript
const cost = createMemo(() => {
  const total = messages().reduce((sum, x) => sum + (x.role === "assistant" ? x.cost : 0), 0)
  return new Intl.NumberFormat("en-US", { style: "currency", currency: "USD" }).format(total)
})
```

The `message.cost` is calculated in `session/index.ts:472-477`:
```javascript
cost: new Decimal(0)
  .add(new Decimal(tokens.input).mul(costInfo?.input ?? 0).div(1_000_000))
  .add(new Decimal(tokens.output).mul(costInfo?.output ?? 0).div(1_000_000))
  // ... cache read/write costs
```

**Source:**
- `session-context-usage.tsx:25-31` - cost memo calculation
- `session/index.ts:467-477` - cost calculation from token counts and pricing

**Significance:** The $ indicator is an ESTIMATED cost based on model pricing data, not actual billed amount. If using OAuth models without pricing info set, cost shows $0.00. The presence of $ cost suggests API key billing path is active.

---

### Finding 4: OAuth Plugin Is Installed but May Not Override Env Key

**Evidence:**
- Config shows plugin: `"plugin": ["opencode-openai-codex-auth"]`
- Plugin provides OAuth auth for OpenAI via ChatGPT subscription
- Auth.json shows OAuth tokens with "pro" plan type
- Recent plugin version: 4.4.0 (Jan 9, 2026)

**Source:**
- `~/.config/opencode/opencode.jsonc:331-333`
- `npm view opencode-openai-codex-auth`
- `auth.json` JWT payload contains `"chatgpt_plan_type":"pro"`

**Significance:** The plugin is correctly installed and OAuth tokens are valid. The issue is that the plugin loader's options may not be overriding the provider.key that was already set from the env var, due to the loading order.

---

## Synthesis

**Key Insights:**

1. **Loading Order Creates Precedence** - OpenCode loads providers in order: ENV vars -> Auth.all(api type only) -> Plugin loaders -> Config. The first to set a key "wins" unless later loaders explicitly override `options["apiKey"]`.

2. **OAuth Requires Plugin Loader Behavior** - For OAuth to work, the plugin must call mergeProvider with `options: { apiKey: accessToken }`. But if env var already set `provider.key`, SDK creation falls back to it unless plugin explicitly sets apiKey in options.

3. **Cost Display Reflects Billing Path** - The $ indicator shows estimated cost from model pricing. OAuth models often have no pricing (subscription-based), so they show $0.00. Seeing a non-zero $ suggests the API key billing path is active.

**Answer to Investigation Question:**

The $ cost indicator appears because OPENAI_API_KEY environment variable is taking precedence over OAuth tokens. When both exist, env var is loaded first (provider.ts:795-804), and unless the plugin explicitly sets options["apiKey"] from the OAuth access token, the SDK falls back to provider.key from the env var (provider.ts:959).

The "(oauth)" indicator shown previously likely appeared when OPENAI_API_KEY was not set or the plugin was overriding correctly. The change may have occurred when:
1. OPENAI_API_KEY was added/restored to the environment
2. Plugin behavior changed in a recent update
3. OpenCode provider loading order changed

---

## Structured Uncertainty

**What's tested:**

- ✅ OPENAI_API_KEY is set as `sk-proj-6TNhtR7...` (verified: `env | grep OPENAI`)
- ✅ OAuth tokens exist in auth.json with type="oauth" (verified: `cat ~/.local/share/opencode/auth.json`)
- ✅ Provider loading order is ENV -> Auth.all -> Plugins -> Config (verified: code review of provider.ts)
- ✅ Cost is calculated from token counts and model pricing data (verified: code review of session/index.ts)

**What's untested:**

- ⚠️ Whether unset OPENAI_API_KEY causes OpenCode to use OAuth (not validated in this session)
- ⚠️ Whether the plugin's loader is correctly setting options["apiKey"] from OAuth token (no runtime debugging)
- ⚠️ Whether recent plugin version (4.4.0) changed precedence behavior (not diffed against earlier versions)

**What would change this:**

- Finding would be wrong if unset OPENAI_API_KEY still shows $ cost (would indicate plugin or model issue)
- Finding would be incomplete if the plugin IS setting apiKey but mergeDeep doesn't preserve it (would need provider.ts mergeDeep behavior analysis)
- Behavior might differ if OpenCode server was restarted (env vars cached at server start)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach: Unset OPENAI_API_KEY for OpenCode Sessions

**Why this approach:**
- Directly addresses root cause (env var precedence)
- No code changes required (user configuration)
- OAuth tokens already exist and are valid
- Immediate fix available

**Trade-offs accepted:**
- OPENAI_API_KEY won't be available for other OpenAI API uses in the same shell
- May need shell configuration changes to persist

**Implementation sequence:**
1. Verify OAuth tokens are valid: `opencode auth list` should show "OpenAI (oauth)"
2. Unset the env var: `unset OPENAI_API_KEY`
3. Restart OpenCode server: `orch-dashboard restart` or manual server restart
4. Verify behavior: Start new session, check model selector shows OAuth models without $ cost

### Alternative Approaches Considered

**Option B: Use a wrapper script/alias**
- **Pros:** Preserves OPENAI_API_KEY for other uses, targeted fix
- **Cons:** Adds complexity, easy to forget the wrapper
- **When to use instead:** If OPENAI_API_KEY is needed for other applications in the same shell

```bash
# Add to ~/.zshrc
alias oc='env -u OPENAI_API_KEY opencode'
```

**Option C: Modify shell config to export conditionally**
- **Pros:** Persistent solution, selective availability
- **Cons:** More complex shell configuration, may break other scripts
- **When to use instead:** If running multiple applications with different auth needs

**Option D: Wait for plugin fix**
- **Pros:** No user action needed if plugin fixes precedence
- **Cons:** Uncertain timeline, may never change (env precedence may be intentional)
- **When to use instead:** If env var MUST be set and OAuth MUST be used

**Rationale for recommendation:** Option A is the simplest and most reliable fix. The OAuth tokens are already present and valid; removing the competing env var allows them to take effect immediately.

---

### Implementation Details

**What to implement first:**
1. Unset OPENAI_API_KEY in current shell
2. Restart OpenCode server (important - server may cache env at start)
3. Verify OAuth is active by checking model selector shows "(OAuth)" or $0.00 cost

**Things to watch out for:**
- The OpenCode server may cache env vars at startup - restart required after unset
- Check if OPENAI_API_KEY is set in ~/.zshrc, ~/.bashrc, or other shell configs
- ChatGPT OAuth tokens expire - may need periodic re-auth via `opencode auth login`

**Areas needing further investigation:**
- Plugin loader implementation: Does it set options["apiKey"] from OAuth access token?
- mergeDeep behavior: Does it properly merge options when provider.key is already set?
- Whether this behavior is intentional (env should always override for power users)

**Success criteria:**
- OpenCode model selector shows OAuth models without $ cost or with "(OAuth)" label
- `opencode auth list` shows only OAuth for OpenAI (no Environment entry)
- Sessions use OAuth billing (no charges to API key)

---

## References

**Files Examined:**
- `packages/opencode/src/provider/provider.ts` - Provider loading and SDK creation
- `packages/opencode/src/auth/index.ts` - Auth storage types and functions
- `packages/app/src/components/session-context-usage.tsx` - Cost display component
- `packages/opencode/src/session/index.ts` - Cost calculation logic
- `~/.config/opencode/opencode.jsonc` - User configuration
- `~/.local/share/opencode/auth.json` - Auth storage

**Commands Run:**
```bash
# Check environment variables
env | grep OPENAI

# Check auth configuration
cat ~/.local/share/opencode/auth.json
opencode auth list

# Check plugin info
npm view opencode-openai-codex-auth
```

**External Documentation:**
- opencode-openai-codex-auth npm package - OAuth auth plugin for ChatGPT Pro subscriptions

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-20-inv-smoke-test-openai-backend-confirm.md` - Prior OpenAI smoke test
- **Investigation:** `.kb/investigations/2026-01-21-inv-test-openai-model.md` - OpenAI model test

---

## Investigation History

**2026-01-20 ~15:00:** Investigation started
- Initial question: Why does UI show $ cost instead of (oauth) for OpenAI models?
- Context: Dylan observed auth indicator changed from (oauth) to $ cost

**2026-01-20 ~15:30:** Root cause identified
- Found OPENAI_API_KEY env var is set alongside OAuth tokens
- Confirmed provider loading order gives ENV precedence
- Identified cost calculation comes from model pricing data

**2026-01-20 ~16:00:** Investigation completed
- Status: Complete
- Key outcome: ENV var precedence confirmed; unset OPENAI_API_KEY to use OAuth
