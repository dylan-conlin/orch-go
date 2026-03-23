## Summary (D.E.K.N.)

**Delta:** GPT-5.4 is already supported end-to-end by OpenCode's provider layer and orch-go's model routing — adding it requires only a 2-line alias in model.go and a 1-line addition to the Codex plugin's allowed models whitelist.

**Evidence:** Read OpenCode provider.ts (26 providers, OpenAI already supported via Vercel AI SDK), codex.ts (OAuth flow already rewrites to /codex/responses endpoint), models-snapshot.ts (gpt-5.4 present: $2.50/$15 per MTok, 1.05M context), orch-go model.go (OpenAI aliases through gpt-5.2 but missing 5.4), resolve.go (OpenAI models auto-route to OpenCode backend).

**Knowledge:** The Anthropic/OpenAI subscription lock-in asymmetry is the key strategic finding: Anthropic banned subscription OAuth in third-party tools (Feb 19, 2026), while OpenAI explicitly allows ChatGPT Pro OAuth in third-party tools via Codex. This means GPT-5.4 can run through orch-go at $0/token via ChatGPT Pro ($200/mo), while Claude requires Claude Code CLI (no programmatic API access via subscription).

**Next:** Three-phase implementation: (1) Add gpt-5.4 alias + Codex whitelist update, (2) Run 5-task stall rate test on feature-impl, (3) If stall rate < 30%, update skill_inference.go to route implementation skills to GPT-5.4.

**Authority:** strategic — This is a provider strategy decision affecting cost structure, model lock-in, and subscription allocation. Dylan decides.

---

# Investigation: Revisit OpenCode Model Routing for GPT 5.4

**Question:** Can orch-go route agent work to GPT-5.4 via OpenCode? What changes are needed, what does the cost picture look like, and can GPT-5.4 handle the worker-base protocol without stalling?

**Started:** 2026-03-23
**Updated:** 2026-03-23
**Owner:** orch-go-uzx15
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/models/agent-lifecycle-state-model/probes/2026-02-28-probe-stalled-agent-failure-pattern-audit.md | extends | yes | GPT-5.4 is a generation ahead of GPT-5.2-codex tested there |
| .kb/models/daemon-autonomous-operation/model.md (DAO-13) | extends | yes | Claim is for GPT-5.2-codex; GPT-5.4 untested |
| .kb/guides/model-selection.md | extends | yes | Guide needs update for GPT-5.4 + Codex OAuth path |

---

## Findings

### Finding 1: OpenCode already supports OpenAI with full provider abstraction

**Evidence:** OpenCode's provider.ts bundles 26 providers including `@ai-sdk/openai` (line 93). The OpenAI custom loader (line 152-159) routes all requests through the Responses API (`sdk.responses(modelID)`). The Codex plugin (codex.ts) implements full OAuth flow with PKCE, token refresh, and URL rewriting from `/v1/responses` → `https://chatgpt.com/backend-api/codex/responses` (line 486-488).

**Source:** `~/Documents/personal/opencode/packages/opencode/src/provider/provider.ts:86-159`, `~/Documents/personal/opencode/packages/opencode/src/plugin/codex.ts:1-494`

**Significance:** No architectural changes needed in OpenCode to support GPT-5.4. The infrastructure is already there.

---

### Finding 2: GPT-5.4 is in the models snapshot but NOT in the Codex allowed models whitelist

**Evidence:** `models-snapshot.ts` contains `gpt-5.4` (released 2026-03-05, $2.50/$15 per MTok, 1.05M context, reasoning + tool_call capable) and `gpt-5.4-pro` ($30/$180 per MTok). However, the Codex plugin's `allowedModels` set (codex.ts:360-367) only includes: `gpt-5.1-codex-max`, `gpt-5.1-codex-mini`, `gpt-5.2`, `gpt-5.2-codex`, `gpt-5.3-codex`, `gpt-5.1-codex`. GPT-5.4 is filtered out when OAuth is active.

**Source:** `~/Documents/personal/opencode/packages/opencode/src/plugin/codex.ts:360-367`, models-snapshot.ts gpt-5.4 entry

**Significance:** To use GPT-5.4 via ChatGPT Pro OAuth (free at flat rate), we need to add `"gpt-5.4"` to the `allowedModels` set. Without this, gpt-5.4 only works with API key auth (pay-per-token).

---

### Finding 3: orch-go model routing already handles OpenAI but lacks GPT-5.4 alias

**Evidence:** `pkg/model/model.go` has OpenAI aliases through gpt-5.2 (lines 46-60) and Codex variants (lines 62-68). The `inferProviderFromModelID` function correctly infers `openai` for any model containing "gpt" (line 147-149). `pkg/spawn/resolve.go:604-616` auto-routes OpenAI models to the OpenCode backend. The daemon's `InferModelFromSkill` (skill_inference.go:279-283) returns empty string for implementation skills, allowing config override.

**Source:** `pkg/model/model.go:46-68`, `pkg/spawn/resolve.go:604-616`, `pkg/daemon/skill_inference.go:270-283`

**Significance:** Adding GPT-5.4 to orch-go requires only adding aliases to model.go. The entire routing pipeline (model → backend → spawn) already handles it.

---

### Finding 4: The subscription economics strongly favor multi-model routing

**Evidence:**

| Model | Input $/MTok | Output $/MTok | Context | Subscription Path |
|-------|-------------|--------------|---------|-------------------|
| Claude Opus 4.5 | $5.00 | $25.00 | 200K | Claude Max ($200/mo, CLI only) |
| GPT-5.4 | $2.50 | $15.00 | 1.05M | ChatGPT Pro ($200/mo, OAuth OK) |
| GPT-5.4 Pro | $30.00 | $180.00 | 1.05M | ChatGPT Pro ($200/mo, OAuth OK) |

Critical asymmetry: Anthropic banned subscription OAuth in third-party tools (Feb 19, 2026). OpenAI explicitly allows it — ChatGPT Pro subscription covers Codex usage in third-party tools at flat rate. This means:
- **Claude:** Must use Claude Code CLI → tmux → orch-go (no headless API path via subscription)
- **GPT-5.4:** Can use OpenCode HTTP API via Codex OAuth → headless (full programmatic control, $0/token)

Per-token GPT-5.4 is also 50% cheaper than Opus on input and 40% cheaper on output.

**Source:** [OpenAI API Pricing](https://developers.openai.com/api/docs/pricing), [Codex Pricing](https://developers.openai.com/codex/pricing), models-snapshot.ts, CLAUDE.md gotcha on OAuth ban

**Significance:** If GPT-5.4 can follow the worker-base protocol reliably, it's strictly superior for cost on implementation skills. The subscription path ($200/mo flat) is identical cost to Claude Max but with programmatic API access.

---

### Finding 5: GPT-5.4 Codex OAuth has a known endpoint fix (already in OpenCode)

**Evidence:** OpenClaw issue #38706 documents that GPT-5.4 via Codex OAuth initially failed because requests went to `/v1/responses` (requires `api.responses.write` scope) instead of the Codex endpoint. The fix: rewrite URL to `/codex/responses`. OpenCode's codex.ts already does this (line 486-488: checks for `/v1/responses` and rewrites to `CODEX_API_ENDPOINT = "https://chatgpt.com/backend-api/codex/responses"`).

**Source:** `~/Documents/personal/opencode/packages/opencode/src/plugin/codex.ts:12,486-488`, [OpenClaw GPT-5.4 OAuth Bug](https://github.com/openclaw/openclaw/issues/38706)

**Significance:** The endpoint fix that tripped up other tools is already implemented in our OpenCode fork. No code changes needed for the OAuth flow itself.

---

### Finding 6: Prior stall rate data is from GPT-5.2 — GPT-5.4 is a generation ahead

**Evidence:** The 67-87% stall rate data (CLAUDE.md, agent-lifecycle-state-model) comes from a Feb 2026 audit of GPT-4o (87.5%) and GPT-5.2-codex (67.5%). GPT-5.4 was released March 5, 2026, with claims of 33% fewer false claims and significantly better instruction-following (native computer-use, agentic workflow support). The stall rate claim's falsification criteria (DAO-13): "Non-Anthropic models achieve >80% completion rate on protocol-heavy daemon spawns (N>30)."

**Source:** `.kb/models/agent-lifecycle-state-model/model.md:469-480`, `.kb/models/daemon-autonomous-operation/claims.yaml:231-244`, [OpenAI GPT-5.4 announcement](https://openai.com/index/introducing-gpt-5-4/)

**Significance:** GPT-5.4's improved instruction-following could materially change stall rates. A controlled test is needed before extrapolating GPT-5.2 data to GPT-5.4. The DAO-13 falsification criteria gives us a clear threshold.

---

## Synthesis

**Key Insights:**

1. **The plumbing is done** — OpenCode + orch-go already support multi-model routing to OpenAI. GPT-5.4 support requires ~5 lines of code changes total (alias additions + whitelist update). No architectural work needed.

2. **The subscription asymmetry is the strategic unlock** — Anthropic forces CLI-only access via subscription. OpenAI allows programmatic API access via subscription OAuth. This means GPT-5.4 via ChatGPT Pro gives orch-go headless agent spawning at $0/token — something currently impossible with Claude Max.

3. **The stall rate question is the only real blocker** — Everything else is trivially implementable. Whether GPT-5.4 can follow the worker-base skill protocol (phase reporting, SYNTHESIS, bd comment discipline) across a multi-step agent session is the only question that matters, and it requires empirical testing.

**Answer to Investigation Question:**

Yes, orch-go can route agent work to GPT-5.4 via OpenCode with minimal changes. The architecture is already built for this — OpenCode has full OpenAI provider support with Codex OAuth, and orch-go's model routing auto-selects the OpenCode backend for OpenAI models. Two changes needed: (1) add `"gpt-5.4"` alias to `pkg/model/model.go` and `"gpt-5.4"` to `allowedModels` in OpenCode's codex.ts, (2) empirically test stall rates before routing production work. The cost case is strong: same subscription price ($200/mo) but with programmatic API access and 50% lower per-token costs. The risk is stall rates — GPT-5.2 had 67.5% stalls, but GPT-5.4 is architecturally different (native computer-use, better instruction-following).

---

## Structured Uncertainty

**What's tested:**

- ✅ OpenCode provider layer supports OpenAI via `@ai-sdk/openai` with Responses API routing (verified: read provider.ts lines 86-159)
- ✅ Codex OAuth flow rewrites endpoint correctly for subscription access (verified: read codex.ts lines 486-488)
- ✅ GPT-5.4 exists in models-snapshot with pricing/capabilities metadata (verified: extracted from models-snapshot.ts)
- ✅ orch-go model routing auto-selects OpenCode backend for OpenAI models (verified: read resolve.go lines 604-616)
- ✅ GPT-5.4 Codex OAuth endpoint fix already present in OpenCode fork (verified: codex.ts line 12 uses /codex/responses)

**What's untested:**

- ⚠️ GPT-5.4 stall rate on worker-base protocol (not tested — this is the critical unknown)
- ⚠️ GPT-5.4 via Codex OAuth actually authenticates and completes a session (not tested end-to-end)
- ⚠️ Whether ChatGPT Pro subscription is needed or Plus ($20/mo) suffices for Codex GPT-5.4 access
- ⚠️ Rate limits on GPT-5.4 via Codex OAuth vs API key

**What would change this:**

- GPT-5.4 stall rate > 50% on feature-impl tasks → keeps Claude as only viable option
- OpenAI restricting Codex OAuth to exclude GPT-5.4 (like they did initially) → forces API key path ($2.50/$15 per MTok)
- Anthropic reversing the OAuth ban → makes Claude Max available via OpenCode headless again

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add GPT-5.4 alias to model.go | implementation | Additive alias, follows established pattern |
| Add GPT-5.4 to Codex allowedModels | implementation | 1-line change in OpenCode fork |
| Run stall rate test (5 spawns) | implementation | Standard investigation methodology |
| Route implementation skills to GPT-5.4 in daemon | strategic | Changes cost structure and model strategy |
| Get ChatGPT Pro subscription for testing | strategic | $200/mo commitment, Dylan decides |

### Recommended Approach: Staged Enablement

**Phase 1: Enable (implementation, ~15 min)**
1. Add `"gpt-5.4": {Provider: "openai", ModelID: "gpt-5.4"}` to model.go Aliases
2. Add `"gpt-5.4"` to `allowedModels` set in OpenCode codex.ts
3. Rebuild OpenCode: `cd ~/Documents/personal/opencode/packages/opencode && bun run build`

**Phase 2: Test (implementation, ~1 hour)**
1. Authenticate: `opencode` → sign in with ChatGPT Pro OAuth
2. Spawn 5 feature-impl tasks: `orch spawn --model gpt-5.4 feature-impl "task"`
3. Measure: completion rate, phase reporting compliance, SYNTHESIS creation, total token usage
4. Compare against Opus baseline (~96% completion, ~4% stall rate)

**Phase 3: Route (strategic, Dylan decides)**
- If GPT-5.4 stall rate < 30%: update `skillModelMapping` in skill_inference.go to route feature-impl → gpt-5.4
- If stall rate 30-50%: use GPT-5.4 only for rate-limit overflow
- If stall rate > 50%: keep as manual `--model gpt-5.4` escape hatch only

### Alternative Approaches Considered

**Option B: API Key only (no Codex OAuth)**
- **Pros:** Simpler setup, no OAuth dance, works immediately with OPENAI_API_KEY env var
- **Cons:** Pay-per-token ($2.50/$15 per MTok), no subscription flat-rate benefit
- **When to use instead:** If Dylan doesn't have ChatGPT Pro, or for one-off testing

**Option C: Full dual-subscription strategy (Claude Max + ChatGPT Pro)**
- **Pros:** Best of both worlds — Claude for reasoning skills, GPT-5.4 for implementation skills
- **Cons:** $400/mo total subscription cost, requires stall rate validation first
- **When to use instead:** After Phase 2 validates GPT-5.4 reliability

**Rationale for staged approach:** The code changes are trivial but the strategic decision (which model for which skills) requires empirical data. We should unlock the capability first, test it, then decide on routing — not commit to a routing strategy before testing.

---

### Implementation Details

**What to implement first:**
- GPT-5.4 alias in model.go (trivial, follows existing pattern)
- Codex allowedModels update in OpenCode fork (1-line change)

**Things to watch out for:**
- ⚠️ Codex OAuth might require ChatGPT Pro ($200/mo), not Plus ($20/mo) — verify subscription tier
- ⚠️ GPT-5.4 Codex rate limits may differ from API rate limits
- ⚠️ The worker-base protocol is ~3,500 tokens of instructions — GPT-5.4's 1.05M context should handle this easily, but instruction-following fidelity is the real test
- ⚠️ SPAWN_CONTEXT.md files (63-76KB) consumed 40-50K GPT tokens per DAO-13 — GPT-5.4's larger context window (1.05M vs 128K) should reduce stalls from context exhaustion

**Success criteria:**
- ✅ `orch spawn --model gpt-5.4 feature-impl "task"` completes end-to-end
- ✅ GPT-5.4 agent reports Phase: Complete with SYNTHESIS.md
- ✅ Stall rate < 30% on N=5 feature-impl tasks
- ✅ Cost per task via Codex OAuth = $0 (subscription)

---

## References

**Files Examined:**
- `~/Documents/personal/opencode/packages/opencode/src/provider/provider.ts` - Provider abstraction, OpenAI custom loader, bundled SDKs
- `~/Documents/personal/opencode/packages/opencode/src/plugin/codex.ts` - Codex OAuth flow, allowed models whitelist, URL rewriting
- `~/Documents/personal/opencode/packages/opencode/src/provider/models-snapshot.ts` - GPT-5.4 model spec (pricing, context, capabilities)
- `~/Documents/personal/opencode/packages/opencode/src/provider/auth.ts` - Provider auth abstraction (OAuth + API key)
- `pkg/model/model.go` - Model aliases and resolution (missing gpt-5.4)
- `pkg/spawn/resolve.go` - Backend routing (OpenAI → OpenCode backend)
- `pkg/daemon/skill_inference.go` - Skill-to-model mapping
- `.kb/guides/model-selection.md` - Current model selection guide
- `.kb/models/agent-lifecycle-state-model/model.md` - Stall rate data (GPT-5.2: 67.5%)
- `.kb/models/daemon-autonomous-operation/claims.yaml` - DAO-13 falsification criteria

**External Documentation:**
- [OpenAI GPT-5.4 Announcement](https://openai.com/index/introducing-gpt-5-4/) - Model capabilities and release details
- [OpenAI API Pricing](https://developers.openai.com/api/docs/pricing) - GPT-5.4 token costs
- [Codex Pricing](https://developers.openai.com/codex/pricing) - Subscription vs API pricing
- [Codex Models](https://developers.openai.com/codex/models) - Available models for Codex
- [Codex Auth](https://developers.openai.com/codex/auth) - OAuth authentication documentation
- [OpenClaw GPT-5.4 OAuth Bug](https://github.com/openclaw/openclaw/issues/38706) - Endpoint fix (already in our fork)
- [OpenCode Codex Auth Plugin](https://github.com/numman-ali/opencode-openai-codex-auth) - Community OAuth plugin reference

**Related Artifacts:**
- **Model:** `.kb/models/agent-lifecycle-state-model/model.md` - Stall rate data and non-Anthropic model constraints
- **Model:** `.kb/models/daemon-autonomous-operation/model.md` - DAO-13 claim on non-Anthropic stall rates
- **Guide:** `.kb/guides/model-selection.md` - Current model selection reference (needs GPT-5.4 update)
- **Decision:** `.kb/decisions/2026-02-19-anthropic-oauth-ban-mandatory-claude-backend.md` - Anthropic OAuth ban context

---

## Investigation History

**2026-03-23:** Investigation started
- Initial question: Can orch-go route to GPT-5.4 via OpenCode? What changes needed? Cost comparison?
- Context: GPT-5.4 released March 5 outperforming Opus 4.6; current architecture has total Anthropic lock-in

**2026-03-23:** Architecture analysis complete
- OpenCode already supports OpenAI with full provider abstraction (26 providers)
- orch-go model routing already handles OpenAI but lacks gpt-5.4 alias
- Codex OAuth flow already present and working with endpoint fix

**2026-03-23:** Cost and strategy analysis complete
- Subscription asymmetry discovered: OpenAI allows OAuth in 3rd party, Anthropic doesn't
- GPT-5.4 pricing is 50% cheaper than Opus per token
- Stall rate is only real unknown — requires empirical test

**2026-03-23:** Investigation completed
- Status: Complete
- Key outcome: GPT-5.4 support requires ~5 lines of code; the strategic question is whether to route skills to it (requires stall rate testing)
