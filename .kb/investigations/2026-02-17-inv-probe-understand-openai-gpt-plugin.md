## Summary (D.E.K.N.)

**Delta:** Orch's `gpt4o` alias resolves to `openai/gpt-4o` which will NOT work end-to-end because the Codex OAuth plugin explicitly filters to GPT-5.x models only; `gpt-4o` needs an API key (pay-per-token) path instead.

**Evidence:** Codex plugin at `codex.ts:360-372` whitelists only 6 Codex models and deletes all others from the OpenAI provider when OAuth is active; auth.json currently has no OpenAI credentials at all.

**Knowledge:** Three distinct auth paths exist (OAuth/API key/none), each with different model availability; OAuth = Codex models only (free with subscription), API key = all models (pay-per-token), none = provider not loaded.

**Next:** Update orch's GPT aliases to match Codex-available models; authenticate OpenAI via OpenCode TUI; update default_model recommendation to a Codex model.

**Authority:** architectural - Cross-component (orch aliases ↔ OpenCode provider ↔ Codex plugin), requires coordinated changes

---

# Investigation: OpenAI/GPT Plugin End-to-End Model String Compatibility

**Question:** Will `orch config set default_model gpt4o` (resolving to `openai/gpt-4o`) work end-to-end through the OpenCode API to the OpenAI provider?

**Defect-Class:** integration-mismatch

**Started:** 2026-02-17
**Updated:** 2026-02-17
**Owner:** orch-go-1018
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A - novel investigation | - | - | - |

---

## Findings

### Finding 1: Orch model resolution correctly produces `openai/gpt-4o`

**Evidence:** `pkg/model/model.go:47-58` defines GPT aliases including `gpt4o → {Provider: "openai", ModelID: "gpt-4o"}`. The `Format()` method at line 13 returns `"openai/gpt-4o"`. The `parseModelSpec()` in `pkg/opencode/client.go:274-288` splits this into `{providerID: "openai", modelID: "gpt-4o"}` for the OpenCode API.

**Source:** `pkg/model/model.go:47-58`, `pkg/opencode/client.go:274-288`

**Significance:** The orch → OpenCode format is correct. The gap is downstream in OpenCode's provider handling.

---

### Finding 2: Codex OAuth plugin whitelists only GPT-5.x models

**Evidence:** The Codex auth plugin at `opencode/src/plugin/codex.ts:360-372` defines an explicit whitelist:
```
allowedModels = ["gpt-5.1-codex-max", "gpt-5.1-codex-mini", "gpt-5.2", "gpt-5.2-codex", "gpt-5.3-codex", "gpt-5.1-codex"]
```
All models NOT in this set AND not containing "codex" are **deleted from the provider**. This means `gpt-4o`, `gpt-4o-mini`, `gpt-5`, `gpt-5-mini`, `o3`, `o3-mini` are all removed when OAuth is active.

**Source:** `~/Documents/personal/opencode/packages/opencode/src/plugin/codex.ts:360-372`

**Significance:** **This is the critical gap.** With ChatGPT Pro OAuth (free subscription usage), only 6 specific models work. Orch's current GPT aliases mostly point to models that would be filtered out.

---

### Finding 3: OpenAI auth not yet configured

**Evidence:** `~/.local/share/opencode/auth.json` contains only Anthropic OAuth credentials. No OpenAI entry exists. Without auth, the OpenAI provider is loaded (from config at `opencode.jsonc:39`) but has no `key`, so API calls would fail at the SDK level.

**Source:** `~/.local/share/opencode/auth.json` (inspected), `opencode/src/provider/provider.ts:840-923` (auth loading flow)

**Significance:** Even if model IDs were correct, nothing would work until the user authenticates. This requires running OpenCode TUI and going through the Codex OAuth flow (browser-based login).

---

### Finding 4: Three distinct auth paths with different model availability

**Evidence:** The provider initialization at `provider.ts:840-933` has three auth sources:

| Auth Path | How | Models Available | Cost |
|-----------|-----|-----------------|------|
| **OAuth (Codex)** | OpenCode TUI → browser login → tokens in auth.json | 6 Codex models (GPT-5.x) | Free with ChatGPT Pro |
| **API Key** | `OPENAI_API_KEY` env var | All 40+ OpenAI models | Pay-per-token |
| **None** | No auth configured | Provider loaded from config but all API calls fail | N/A |

With OAuth: plugin runs at line 863, the custom loader at line 918 sees `providers["openai"]` exists and registers `sdk.responses()` model loader.
With API key: env loading at line 840 adds provider, custom loader registers `sdk.responses()`.
With neither: config section at line 927 adds provider from database, but custom loader already ran and was skipped — no model loader registered, so GPT-5+ would use wrong API.

**Source:** `provider.ts:840-933` (init flow), `codex.ts:351-450` (plugin auth loader)

**Significance:** The auth path determines both which models are available AND whether the correct API endpoint (Responses vs Chat) is used.

---

### Finding 5: OpenCode user config already has GPT-5.x model overrides

**Evidence:** `~/.config/opencode/opencode.jsonc` (symlinked from dotfiles) at lines 39-268 defines 7 OpenAI models with custom variants (reasoning effort levels): `gpt-5.2`, `gpt-5.2-codex`, `gpt-5.1-codex-max`, `gpt-5.1-codex`, `gpt-5.1-codex-mini`, `gpt-5.1`.

**Source:** `~/Documents/dotfiles/.config/opencode/opencode.jsonc:39-268`

**Significance:** The config is already set up for GPT-5.x Codex usage. It just needs auth credentials.

---

### Finding 6: Model format flows correctly through spawn

**Evidence:** Traced the full flow:
1. `orch spawn --model gpt4o` → `model.Resolve("gpt4o")` returns `ModelSpec{Provider: "openai", ModelID: "gpt-4o"}`
2. `ModelSpec.Format()` → `"openai/gpt-4o"`
3. Headless: `client.CreateSession(...)` at `client.go:539` passes model to `parseModelSpec()` → `{providerID: "openai", modelID: "gpt-4o"}`
4. Tmux: `BuildOpencodeAttachCommand()` at `tmux.go:280` passes `--model "openai/gpt-4o"`
5. OpenCode session init at `session/index.ts:797` reconstructs `providerID + "/" + modelID`
6. `Provider.getModel("openai", "gpt-4o")` looks up in loaded providers

The `provider/model` format is consistent end-to-end. The issue is model availability, not format.

**Source:** `pkg/model/model.go`, `pkg/opencode/client.go:274-288`, `pkg/tmux/tmux.go:268-288`, `opencode/src/session/index.ts:786-800`

**Significance:** No format mismatch. The plumbing works. Only the model IDs and auth need updating.

---

### Finding 7: `gpt-4o` IS in the models-snapshot database

**Evidence:** Ran extraction script against `models-snapshot.ts`. The OpenAI provider includes `gpt-4o: GPT-4o` among 40+ models. So with API key auth (not OAuth), `gpt-4o` would be available.

**Source:** `opencode/packages/opencode/src/provider/models-snapshot.ts` (extracted via script)

**Significance:** `gpt-4o` exists in the canonical model database. The Codex plugin removes it when OAuth is active, but with API key auth it would work.

---

## Synthesis

**Key Insights:**

1. **Auth determines model universe** — OAuth (ChatGPT Pro subscription) gives free access to 6 Codex models; API key gives access to all models but costs money. This is not a bug but intentional design by the Codex plugin.

2. **Orch's GPT aliases are misaligned with Codex** — Most of orch's OpenAI aliases (`gpt4o`, `gpt-4o`, `gpt-4o-mini`, `gpt-5`, `gpt-5-mini`, `o3`, `o3-mini`) point to models that are NOT available via OAuth. Only `gpt5-latest` (→ `gpt-5.2`) overlaps with Codex's whitelist.

3. **One-time auth setup required** — The OpenCode TUI must be used interactively to complete the Codex OAuth flow. This is browser-based (opens auth.openai.com), stores tokens in auth.json, and then headless/API access works until tokens expire.

**Answer to Investigation Question:**

**No, `orch config set default_model gpt4o` will NOT work end-to-end.** Three gaps must be addressed:

1. **No auth** — auth.json lacks OpenAI credentials. Fix: Run OpenCode TUI, authenticate via Codex OAuth.
2. **Wrong model for OAuth** — `gpt-4o` is removed by Codex plugin when OAuth is active. Fix: Use a Codex-whitelisted model like `gpt-5.2` or `gpt-5.2-codex`.
3. **Missing aliases** — Orch has no aliases for the primary Codex models (`gpt-5.2-codex`, `gpt-5.1-codex`, `gpt-5.3-codex`). Fix: Add aliases.

With API key auth (`OPENAI_API_KEY` env var), `openai/gpt-4o` would work, but this is pay-per-token.

---

## Structured Uncertainty

**What's tested:**

- ✅ Model format `openai/gpt-4o` is correctly parsed end-to-end (read source: `model.go`, `client.go`, `session/index.ts`)
- ✅ auth.json currently has only Anthropic credentials (inspected file directly)
- ✅ Codex plugin whitelists exactly 6 models and deletes others (read source: `codex.ts:360-372`)
- ✅ `gpt-4o` exists in models-snapshot database (extracted via script)
- ✅ OpenCode config already has GPT-5.x model overrides configured (read `opencode.jsonc`)

**What's untested:**

- ⚠️ Actual OAuth flow completion (requires interactive browser login — cannot test in agent session)
- ⚠️ Whether `gpt-5.2-codex` works correctly after auth (requires running a real session)
- ⚠️ Rate limits and usage caps for Codex OAuth path (no documentation found in source)
- ⚠️ Whether API key path (`OPENAI_API_KEY`) correctly registers model loader (code reading suggests yes, but not tested)

**What would change this:**

- If OpenAI adds `gpt-4o` to the Codex whitelist, the current aliases would work
- If a non-OAuth auth path is preferred, `OPENAI_API_KEY` would enable all models
- If the Codex plugin changes its filtering logic, model availability would change

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add Codex-model aliases to orch | implementation | Within model.go scope, reversible |
| Authenticate OpenAI via OpenCode TUI | implementation | One-time operational setup |
| Update default_model guidance | architectural | Affects spawn behavior across all agents |
| Choose OAuth vs API key path | strategic | Cost model decision (free vs pay-per-token) |

### Recommended Approach ⭐

**OAuth-first with Codex aliases** — Authenticate via Codex OAuth, update orch aliases to match Codex models, default to `gpt-5.2-codex`.

**Why this approach:**
- Free with ChatGPT Pro subscription (no per-token cost)
- GPT-5.2 Codex is purpose-built for coding tasks
- Config already has the model definitions with reasoning effort variants

**Trade-offs accepted:**
- No access to GPT-4o via OAuth (use Anthropic models instead)
- Limited to 6 Codex models (but these are the best for coding)

**Implementation sequence:**
1. **Auth setup:** Run OpenCode TUI, go through Codex OAuth flow → tokens stored in auth.json
2. **Add aliases:** In `pkg/model/model.go`, add Codex-specific aliases:
   - `codex` → `openai/gpt-5.2-codex`
   - `codex-mini` → `openai/gpt-5.1-codex-mini`
   - `codex-max` → `openai/gpt-5.1-codex-max`
   - `gpt5-codex` → `openai/gpt-5.2-codex`
   - `gpt53-codex` → `openai/gpt-5.3-codex`
3. **Update default_model recommendation:** `gpt-5.2-codex` (not `gpt4o`)

### Alternative Approaches Considered

**Option B: API key path**
- **Pros:** All 40+ models available, simpler auth (just set env var)
- **Cons:** Costs money per-token, no subscription benefit
- **When to use instead:** When specific non-Codex models needed (e.g., GPT-4o for cheaper/faster tasks)

**Option C: Dual path (keep both)**
- **Pros:** Maximum flexibility
- **Cons:** Complexity — model availability depends on which auth is active, confusing UX
- **When to use instead:** If both free (OAuth) and paid (API key) usage is needed for different models

**Rationale for recommendation:** OAuth is free with existing ChatGPT Pro subscription. The Codex models are the best available for coding. No reason to pay per-token when subscription covers it.

---

### Implementation Details

**What to implement first:**
- Auth setup (one-time, manual, blocks everything else)
- Codex aliases in `model.go` (quick, unblocks `orch spawn --model codex`)
- Update `gpt` alias from `gpt-4o` to `gpt-5.2-codex` (or leave as-is for API key users)

**Things to watch out for:**
- ⚠️ OAuth tokens expire — Codex plugin auto-refreshes using refresh_token (`codex.ts:438-450`)
- ⚠️ Custom model loader timing — must be registered BEFORE config section. With OAuth, plugin auth runs first so this works correctly.
- ⚠️ `gpt-5.2` (without `-codex`) IS in the whitelist — it may use a different API endpoint than `gpt-5.2-codex`

**Areas needing further investigation:**
- Rate limits for Codex OAuth path (not documented in source code)
- Whether `gpt-5.2` vs `gpt-5.2-codex` have different capabilities/limits
- Whether the Codex Responses API (`chatgpt.com/backend-api/codex/responses`) has any restrictions for headless/API use

**Success criteria:**
- ✅ `orch spawn --model codex "test"` creates session with `openai/gpt-5.2-codex`
- ✅ OpenCode resolves the model and makes successful API calls
- ✅ Agent completes work using GPT-5.2 Codex model

---

## References

**Files Examined:**
- `pkg/model/model.go` - Orch model alias resolution
- `pkg/opencode/client.go:274-288` - parseModelSpec function
- `pkg/tmux/tmux.go:268-288` - BuildOpencodeAttachCommand
- `~/Documents/personal/opencode/packages/opencode/src/provider/provider.ts:60-134,700-933,988-1050,1097-1146` - Provider initialization, custom loaders, getModel/getSDK
- `~/Documents/personal/opencode/packages/opencode/src/plugin/codex.ts:1-60,120-143,351-450` - Codex auth plugin, OAuth flow, model filtering
- `~/Documents/personal/opencode/packages/opencode/src/session/index.ts:786-800` - Session creation with model
- `~/Documents/personal/opencode/packages/opencode/src/auth/index.ts` - Auth storage types
- `~/Documents/dotfiles/.config/opencode/opencode.jsonc` - User config with OpenAI models
- `~/.local/share/opencode/auth.json` - Current auth credentials
- `opencode/packages/opencode/src/provider/models-snapshot.ts` - Canonical model database

**Commands Run:**
```bash
# Check auth.json for OpenAI credentials
cat ~/.local/share/opencode/auth.json | python3 -m json.tool

# Extract OpenAI models from models-snapshot
bun -e '<script extracting openai models from snapshot>'
```

**Related Artifacts:**
- **Decision:** `orch-go commit bc974cd4` - "Add default_model config key and GPT-4o model aliases" (the commit that prompted this investigation)
