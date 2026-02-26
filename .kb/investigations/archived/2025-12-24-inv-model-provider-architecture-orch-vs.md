<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Clarified the model provider architecture - orch handles account management and model resolution, OpenCode handles API auth and runtime inference.

**Evidence:** Code analysis of pkg/model (aliases), pkg/account (Claude Max OAuth), pkg/opencode (session/prompt APIs), and spawn flows in cmd/orch/main.go confirms current separation.

**Knowledge:** Current architecture is sound for Anthropic (Claude Max). Multi-provider support (Gemini, OpenRouter, DeepSeek) requires provider-specific auth patterns that OpenCode must handle, not orch.

**Next:** Create follow-up decision record if multi-provider auth complexity warrants formal architecture decision.

**Confidence:** High (85%) - Current architecture understood, multi-provider paths need more exploration.

---

# Investigation: Model Provider Architecture - orch vs OpenCode Auth Responsibility

**Question:** How should model provider authentication be divided between orch and OpenCode, particularly for multi-provider support (Gemini, OpenRouter, DeepSeek)?

**Started:** 2025-12-24
**Updated:** 2025-12-24
**Owner:** og-work-model-provider-architecture-24dec agent
**Phase:** Complete
**Next Step:** None (investigation complete)
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: Current Architecture Cleanly Separates Concerns

**Evidence:** The current orch-go architecture has clear layers:
1. **orch (pkg/model)**: Model alias resolution (`opus` → `anthropic/claude-opus-4-5-20251101`)
2. **orch (pkg/account)**: Claude Max account management (OAuth token refresh, account switching)
3. **orch (cmd/orch/main.go)**: Orchestration (spawn, complete, monitor) with model flag handling
4. **OpenCode**: Session management, prompt execution, API auth at runtime

**Source:** 
- `pkg/model/model.go:46-83` - Resolve() function handles aliases and provider/model format
- `pkg/account/account.go:354-411` - SwitchAccount() handles OAuth token refresh
- `pkg/opencode/client.go:154-168` - BuildSpawnCommand() passes --model to CLI
- `cmd/orch/main.go:1103` - resolvedModel := model.Resolve(spawnModel)

**Significance:** The separation is clean - orch handles orchestration-layer concerns (which account, which model alias), OpenCode handles runtime concerns (API auth, inference).

---

### Finding 2: Anthropic Auth Flow Works Through OpenCode's auth.json

**Evidence:** When orch switches Claude Max accounts:
1. orch calls Anthropic OAuth endpoint to exchange refresh token for access token
2. orch writes tokens to `~/.local/share/opencode/auth.json`
3. OpenCode reads auth.json for API calls
4. orch also saves updated refresh token to `~/.orch/accounts.yaml`

**Source:**
- `pkg/account/account.go:396-405` - SaveOpenCodeAuth() writes to OpenCode's auth file
- `pkg/account/account.go:294-352` - RefreshOAuthToken() calls Anthropic OAuth endpoint
- OpenCode auth file location: `~/.local/share/opencode/auth.json`

**Significance:** orch "owns" account management but delegates runtime auth to OpenCode by writing to its auth file. This is an implicit contract.

---

### Finding 3: Gemini Uses Different Auth Pattern (API Key)

**Evidence:** From prior investigations and model resolution code:
- Gemini uses API keys, not OAuth
- Model resolution maps `flash` → `google/gemini-2.5-flash`
- OpenCode handles Gemini API keys through its own config

**Source:**
- `pkg/model/model.go:39-43` - Gemini aliases (flash, pro)
- Prior investigation: `.kb/investigations/2025-12-20-inv-research-gemini-model-arbitrage-alternatives.md`

**Significance:** Multi-provider support requires understanding that different providers have different auth patterns. OpenCode already handles Gemini auth separately.

---

### Finding 4: OpenRouter and DeepSeek Would Be Third-Party Providers

**Evidence:** From research investigations:
- OpenRouter: API key + routing header, accesses multiple models
- DeepSeek: Native API with its own API key at `api-docs.deepseek.com`
- These are not in current orch model aliases

**Source:**
- `.kb/investigations/2025-12-20-research-deepseek-llama-arbitrage-comparison.md`
- Model aliases in `pkg/model/model.go` only cover Anthropic and Google

**Significance:** Adding OpenRouter/DeepSeek would require:
1. Adding model aliases to `pkg/model`
2. OpenCode supporting the provider's auth pattern
3. No orch account management needed (these are API key based, not Claude Max OAuth)

---

## Synthesis

**Key Insights:**

1. **orch manages Claude Max accounts because of OAuth complexity** - Token refresh, account switching, capacity tracking all require coordinated state management. This is orch's domain.

2. **OpenCode is the auth runtime** - It reads auth files and makes API calls. orch writes to OpenCode's auth.json as the handoff mechanism.

3. **Multi-provider support is provider-specific** - API key providers (Gemini, OpenRouter, DeepSeek) don't need orch account management. They just need:
   - Model aliases in `pkg/model`
   - OpenCode to support the provider (already does for Gemini)

**Answer to Investigation Question:**

The current division of responsibility is appropriate:

| Responsibility | Owner | Why |
|---------------|-------|-----|
| Model alias resolution | orch (pkg/model) | User convenience, single place for mapping |
| Claude Max accounts | orch (pkg/account) | OAuth complexity, multi-account capacity tracking |
| Claude API auth | OpenCode (via auth.json) | Runtime concern, orch writes file |
| Gemini API auth | OpenCode | API key, no orch management needed |
| OpenRouter/DeepSeek | OpenCode (future) | API key, just need model aliases in orch |

**For multi-provider expansion:**
- orch only needs to add model aliases (`deepseek` → provider/model)
- OpenCode needs to support provider auth (API keys in its config)
- orch account management is Claude Max specific, not needed for API key providers

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

Clear code analysis confirms current architecture. The multi-provider paths are reasonable extrapolations but haven't been implemented.

**What's certain:**

- ✅ Current Anthropic flow works (orch manages accounts, writes to OpenCode auth.json)
- ✅ Model resolution is cleanly separated (pkg/model owns aliases)
- ✅ Gemini works without orch account management (API key based)

**What's uncertain:**

- ⚠️ OpenCode's support for OpenRouter/DeepSeek (not yet explored)
- ⚠️ Whether API key providers should have orch-level management for rate limiting
- ⚠️ How model routing decisions should be made (orch or OpenCode)

**What would increase confidence to 95%+:**

- Test spawning with Gemini model to verify end-to-end
- Review OpenCode's provider configuration for API keys
- Prototype DeepSeek integration to validate assumptions

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Keep current separation, extend model aliases incrementally** - orch owns model resolution and Claude Max accounts; OpenCode owns provider auth at runtime.

**Why this approach:**
- Current architecture is clean and working
- No need to move auth complexity into orch for API key providers
- Incremental model alias additions are low risk

**Trade-offs accepted:**
- orch account management is Anthropic-specific (acceptable - it's the only OAuth provider)
- Rate limiting for API key providers is not orch-managed (defer until needed)

**Implementation sequence:**
1. Add DeepSeek aliases to pkg/model if needed (`deepseek` → `deepseek/deepseek-v3`)
2. Ensure OpenCode supports DeepSeek API key configuration
3. Test spawn with `--model deepseek` to verify

### Alternative Approaches Considered

**Option B: Unified orch auth for all providers**
- **Pros:** Single place for all API key management
- **Cons:** Duplicates OpenCode's config, adds complexity for simple API key providers
- **When to use instead:** If rate limiting across all providers becomes critical

**Option C: Push all auth to OpenCode, orch just passes model string**
- **Pros:** Simpler orch, no auth code
- **Cons:** Loses Claude Max multi-account switching value
- **When to use instead:** If moving away from Claude Max subscriptions

**Rationale for recommendation:** Current architecture works well for the primary use case (Claude Max with account arbitrage). API key providers don't need the complexity of orch account management.

---

### Implementation Details

**What to implement first:**
- If DeepSeek support needed: Add aliases to `pkg/model/model.go`
- No orch auth changes needed for API key providers

**Things to watch out for:**
- ⚠️ OpenCode may need `--provider-key` flag or config file updates for new providers
- ⚠️ Model resolution should gracefully handle unknown providers (pass through)
- ⚠️ Don't add account switching for API key providers (overengineering)

**Areas needing further investigation:**
- How OpenCode handles API keys for non-Anthropic providers
- Whether DeepSeek's native API has rate limits that need orch-level management
- OpenRouter's model routing and how it maps to orch aliases

**Success criteria:**
- ✅ `orch spawn --model flash investigation "task"` continues to work
- ✅ New aliases resolve correctly without breaking existing flows
- ✅ No changes needed to orch account management for API key providers

---

## References

**Files Examined:**
- `pkg/model/model.go` - Model alias resolution
- `pkg/account/account.go` - Claude Max OAuth and account switching
- `pkg/opencode/client.go` - OpenCode API client
- `cmd/orch/main.go:1031-1344` - Spawn flows (inline, headless, tmux)
- `pkg/tmux/tmux.go` - Tmux mode command building

**Commands Run:**
```bash
# Knowledge context
kb context "model"
kb context "authentication"
kb context "deepseek"

# File exploration via grep
grep -l "runSpawnWithSkill\|spawnModel\|model\." cmd/orch/
```

**External Documentation:**
- DeepSeek API docs: https://api-docs.deepseek.com
- Anthropic OAuth: Used by pkg/account for token refresh

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-20-inv-investigate-model-flexibility-arbitrage-orch.md` - Prior work on model flexibility
- **Investigation:** `.kb/investigations/2025-12-20-research-deepseek-llama-arbitrage-comparison.md` - DeepSeek/Llama comparison

---

## Investigation History

**[2025-12-24 18:00]:** Investigation started
- Initial question: How should model provider auth be divided between orch and OpenCode?
- Context: Design session for multi-provider support (Gemini, OpenRouter, DeepSeek)

**[2025-12-24 18:30]:** Context gathering complete
- Analyzed pkg/model, pkg/account, pkg/opencode, cmd/orch/main.go
- Found clear separation of concerns already exists

**[2025-12-24 18:45]:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Current architecture is sound, multi-provider expansion is incremental via model aliases
