<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** OpenAI is officially collaborating with OpenCode to enable third-party access via ChatGPT Plus/Pro subscriptions - the opposite of Anthropic's blocking stance. GPT-5.2 Codex and 22 model variants now accessible via `opencode-openai-codex-auth` plugin.

**Evidence:** OpenCode creator Dax Raad announced official partnership with OpenAI. Plugin exists on GitHub/npm. VentureBeat confirms OpenAI "embraced collaborative approach" while Anthropic explicitly blocked third-party tools.

**Knowledge:** OpenAI allows what Anthropic blocked. ChatGPT Pro ($200/mo) provides unlimited access to GPT-5.2 Codex via OpenCode OAuth plugin - viable escape hatch from Anthropic restrictions. Third-party tool access is a strategic differentiator between AI vendors.

**Next:** Consider adding OpenAI/Codex as backend option in orch-go. Test opencode-openai-codex-auth plugin reliability before production use.

**Promote to Decision:** recommend-yes (strategic choice: multi-vendor model access affects spawn architecture)

---

# Investigation: OpenAI Partnership with OpenCode for Third-Party Access

**Question:** What is OpenAI's stance on third-party tool access vs Anthropic's? Is there an official partnership with OpenCode? What models are available and how does this affect our spawn strategy?

**Started:** 2026-01-21
**Updated:** 2026-01-21
**Owner:** Architect agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** .kb/decisions/2026-01-09-abandon-claude-max-oauth-use-gemini-primary.md
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: OpenAI Is Officially Working With OpenCode (Opposite of Anthropic)

**Evidence:**
- OpenCode creator Dax Raad publicly announced: "we are working with openai to allow codex users to benefit from their subscription directly within OpenCode"
- VentureBeat confirms: "In contrast to Anthropic's restrictive measures, OpenAI has embraced a more collaborative approach by supporting third-party tools like OpenCode"
- Plugin released: `opencode-openai-codex-auth` - uses OpenAI's official OAuth authentication method
- GitHub partnership on Jan 16, 2026 unlocked "GitHub Copilot's model garden for terminal-native developers"

**Source:**
- https://jpcaparas.medium.com/jump-ship-in-minutes-codex-oauth-now-works-in-opencode-d2708c32f571
- https://numman-ali.github.io/opencode-openai-codex-auth/
- https://venturebeat.com/technology/anthropic-cracks-down-on-unauthorized-claude-usage-by-third-party-harnesses

**Significance:** Strategic vendor differentiation. OpenAI sees third-party ecosystem as feature; Anthropic sees it as threat. OpenCode (60K+ stars, 650K+ monthly users) now has official OpenAI path but blocked Anthropic path.

---

### Finding 2: 22 Model Variants Available via OpenAI/OpenCode Integration

**Evidence:**
From opencode-openai-codex-auth plugin documentation:
- **GPT 5.2** - flagship model (Auto, Instant, Thinking modes)
- **GPT 5.2 Codex** - optimized for agentic coding
- **GPT 5.1** - previous generation
- **Codex, Codex Max, Codex Mini** - coding-specific variants
- **o3, o3-pro** - most powerful reasoning models
- **o4-mini, o4-mini-high** - fast reasoning
- Different "reasoning effort" levels including "xhigh" for select models

**Source:**
- https://numman-ali.github.io/opencode-openai-codex-auth/
- https://openai.com/index/introducing-gpt-5-2-codex/
- https://platform.openai.com/docs/models/gpt-5-2-codex

**Significance:** More models than we had with Claude Max. GPT-5.2 Codex specifically optimized for "long-horizon work through context compaction, stronger performance on large code changes like refactors and migrations."

---

### Finding 3: Cost Structure Is Identical - $200/mo Unlimited Access

**Evidence:**
| Subscription | Price | Third-Party Access | Models |
|--------------|-------|-------------------|--------|
| ChatGPT Pro | $200/mo | ✅ ALLOWED via OpenCode | GPT-5.2, o3-pro, Codex |
| Claude Max | $200/mo | ❌ BLOCKED since Jan 9 | Opus 4.5, Sonnet 4 |
| Claude Max | $100/mo | ❌ BLOCKED since Jan 9 | Same (5x usage) |

Both subscriptions:
- Do NOT include API access (billed separately)
- Provide OAuth for CLI tools (Codex CLI / Claude Code)
- Unlimited usage within their respective ecosystems

**Source:**
- https://chatgpt.com/pricing/
- https://www.anthropic.com/pricing/
- .kb/investigations/2026-01-18-inv-research-compare-openai-chatgpt-pro-anthropic-max.md

**Significance:** Same price, but OpenAI allows third-party tools while Anthropic blocks them. For orchestration, this makes OpenAI subscription strategically valuable as escape hatch.

---

### Finding 4: Plugin Has Limitations - Personal Use Only

**Evidence:**
From opencode-openai-codex-auth documentation:
- "This plugin is for personal development use only"
- Commercial services, API resale, and multi-user applications prohibited
- Must comply with OpenAI's Terms of Use
- "GPT 5 models can be unstable; official configurations are mandatory"

Setup:
```bash
npx -y opencode-openai-codex-auth@latest
# Add to ~/.config/opencode/opencode.jsonc
opencode auth login
opencode run "write hello world" --model=openai/gpt-5.2
```

**Source:** https://numman-ali.github.io/opencode-openai-codex-auth/

**Significance:** Personal orchestration is allowed. Same restriction as Claude Max (personal use). Not a blocker for orch-go workflow.

---

### Finding 5: OpenAI API Access Coming Soon for GPT-5.2 Codex

**Evidence:**
- "GPT 5.2 Codex is available now for all paid ChatGPT users"
- "API access is expected to roll out in the coming weeks"
- OpenAI piloting "trusted access" program for vetted security researchers
- "Cyber Trusted Access" scheme for defensive research

**Source:**
- https://openai.com/index/introducing-gpt-5-2-codex/
- https://platform.openai.com/docs/models/gpt-5-2-codex

**Significance:** Eventually could add OpenAI API as pay-per-token option alongside DeepSeek V3. Right now subscription-only via OAuth.

---

## Synthesis

**Key Insights:**

1. **OpenAI Chose Third-Party Collaboration** - While Anthropic blocked third-party tools and sparked "Anthropic vs Community" drama (DHH, 119+ subscription cancellations), OpenAI took the opposite approach. Within hours of Anthropic's block, OpenCode shipped ChatGPT Plus/Pro support with OpenAI's cooperation.

2. **Same Price, Different Access** - Both vendors charge $200/mo for unlimited AI coding. But OpenAI allows third-party orchestration tools while Anthropic restricts to Claude Code only. For multi-model orchestration strategy, this makes OpenAI subscription more valuable.

3. **GPT-5.2 Codex Is Specifically Designed for Agents** - OpenAI's model is optimized for "long-horizon work through context compaction" and "large code changes like refactors and migrations" - exactly what orchestrated agents need.

**Answer to Investigation Question:**

**1. OpenAI's stance vs Anthropic's?**
Diametrically opposed. OpenAI: "working with OpenCode" to enable third-party access. Anthropic: technical fingerprinting to block all third-party tools. OpenAI sees ecosystem growth as beneficial; Anthropic sees it as revenue leakage.

**2. Official partnership or just compatibility?**
Official collaboration. Dax Raad (OpenCode creator) explicitly stated they're "working with openai." This is not a workaround - OpenAI sanctioned it.

**3. What models would be available?**
22 variants including: GPT-5.2, GPT-5.2 Codex, GPT-5.1, o3, o3-pro, o4-mini, Codex Max/Mini. Multiple reasoning effort levels. Context compaction for long-horizon work.

**4. How would this affect spawn strategy?**
YES - add OpenAI as backend option. It provides:
- Escape hatch from Anthropic blocks ($200/mo unlimited)
- Access to GPT-5.2 Codex optimized for agents
- Model diversity (reduces single-vendor risk)
- Same orchestration pattern (OAuth → OpenCode → spawn)

**5. Cost comparison:**

| Option | Cost | Pros | Cons |
|--------|------|------|------|
| OpenAI ChatGPT Pro | $200/mo flat | Unlimited, third-party allowed, GPT-5.2 Codex | OAuth plugin needed |
| Claude Max | $200/mo flat | Unlimited, best coding | Third-party BLOCKED |
| DeepSeek V3 | $0.25/$0.38/MTok | Cheapest, function calling works | Untested at scale |
| Gemini Flash | Free tier | No cost | 2000 req/min limit blocks agents |
| Docker workaround | $200/mo + complexity | Bypasses Anthropic fingerprinting | Complex setup, tmux-in-tmux |

---

## Structured Uncertainty

**What's tested:**

- ✅ OpenAI/OpenCode collaboration announced (verified: multiple sources, Dax Raad statement)
- ✅ Plugin exists and is documented (verified: GitHub repo, npm package)
- ✅ ChatGPT Pro costs $200/mo with unlimited usage (verified: official pricing)
- ✅ Anthropic blocks third-party tools (verified: Jan 9 investigation, VentureBeat)
- ✅ 22 model variants available via plugin (verified: plugin documentation)

**What's untested:**

- ⚠️ opencode-openai-codex-auth plugin stability in practice (not tested)
- ⚠️ GPT-5.2 Codex quality vs Claude Opus 4.5 for orchestration (not benchmarked)
- ⚠️ o3-pro reasoning capability vs extended thinking (not compared)
- ⚠️ Plugin OAuth reliability across sessions (reports exist but not verified)
- ⚠️ Model switching latency in OpenCode with OpenAI backend (not measured)

**What would change this:**

- OpenAI reversing third-party access policy → lose escape hatch
- Plugin OAuth becoming unreliable → not viable for production
- GPT-5.2 Codex significantly worse than Claude for coding → prefer Claude despite restrictions
- Anthropic reopening third-party access → re-evaluate strategy

---

## Implementation Recommendations

**Purpose:** Should we add OpenAI as a backend option in orch-go?

### Recommended Approach ⭐

**Add OpenAI/Codex as secondary backend option** - Implement OpenAI model aliases and OAuth support as escape hatch from Anthropic restrictions.

**Why this approach:**
- Same orchestration pattern (OAuth subscription → OpenCode → spawn)
- Reduces single-vendor risk (Anthropic can restrict further)
- Access to GPT-5.2 Codex optimized for agents
- Validates multi-provider architecture already in orch-go

**Trade-offs accepted:**
- Plugin stability unverified (test before production)
- Additional complexity (third auth mechanism after Claude + Gemini)
- Personal use restriction (same as Claude Max)

**Implementation sequence:**
1. **Test plugin locally** - Install opencode-openai-codex-auth, verify OAuth flow
2. **Add model aliases** - Update pkg/model/model.go with OpenAI variants (gpt-5.2, codex, o3)
3. **Document escape hatch** - Add to CLAUDE.md and spawn guide
4. **Production validation** - Run test spawns with GPT-5.2 Codex

### Alternative Approaches Considered

**Option B: Ignore OpenAI integration**
- **Pros:** Less complexity, current setup works via Docker workaround
- **Cons:** Single-vendor risk, Docker workaround is fragile, misses GPT-5.2 Codex capability
- **When to use instead:** If plugin proves unreliable in testing

**Option C: Full migration from Anthropic to OpenAI**
- **Pros:** Cleaner architecture, single vendor
- **Cons:** Lose Claude Opus 4.5 quality, rebuild agent prompts, lose institutional knowledge
- **When to use instead:** If Anthropic blocks Claude Code CLI itself

**Rationale for recommendation:** OpenAI/Codex provides backup path at same cost point ($200/mo) with explicit third-party support. Testing required before production reliance, but strategic optionality is valuable.

---

### Implementation Details

**What to implement first:**
- Install and test opencode-openai-codex-auth plugin locally
- Verify OAuth flow works with ChatGPT Plus/Pro subscription
- Document any friction or failures

**Things to watch out for:**
- ⚠️ Plugin is community-maintained (numman-ali), not OpenAI official
- ⚠️ "GPT 5 models can be unstable" per plugin docs
- ⚠️ May need manual URL paste for SSH/WSL environments
- ⚠️ Personal use only - same restriction as Claude Max

**Areas needing further investigation:**
- GPT-5.2 Codex vs Claude Opus 4.5 quality comparison
- Plugin reliability over 24+ hour sessions
- Model switching performance in OpenCode
- o3-pro reasoning vs Claude extended thinking

**Success criteria:**
- ✅ Can spawn agent with `orch spawn --model gpt-5.2-codex investigation "test"`
- ✅ OAuth persists across sessions without re-auth
- ✅ Agent completes standard orchestration task (investigation, feature-impl)
- ✅ Cost matches subscription ($0 incremental, already paying for ChatGPT Pro or not)

---

## References

**External Documentation:**
- [OpenCode Codex Auth Plugin](https://numman-ali.github.io/opencode-openai-codex-auth/) - Setup and model list
- [GitHub: opencode-openai-codex-auth](https://github.com/numman-ali/opencode-openai-codex-auth) - Source code
- [OpenAI GPT-5.2 Codex](https://openai.com/index/introducing-gpt-5-2-codex/) - Official announcement
- [OpenAI Platform Docs](https://platform.openai.com/docs/models/gpt-5-2-codex) - Model details
- [VentureBeat: Anthropic Crackdown](https://venturebeat.com/technology/anthropic-cracks-down-on-unauthorized-claude-usage-by-third-party-harnesses) - Context on blocking
- [JP Caparas: Codex OAuth in OpenCode](https://jpcaparas.medium.com/jump-ship-in-minutes-codex-oauth-now-works-in-opencode-d2708c32f571) - Integration guide
- [ChatGPT Pricing](https://chatgpt.com/pricing/) - Subscription tiers

**Related Artifacts:**
- **Decision:** .kb/decisions/2026-01-09-abandon-claude-max-oauth-use-gemini-primary.md - Context on Anthropic blocking
- **Investigation:** .kb/investigations/2026-01-18-inv-research-compare-openai-chatgpt-pro-anthropic-max.md - Prior comparison
- **Model:** .kb/models/orchestration-cost-economics.md - Cost analysis

---

## Investigation History

**2026-01-21 01:40:** Investigation started
- Question: What is OpenAI's stance on third-party tool access? Is there an official partnership with OpenCode?
- Context: Anthropic blocked third-party OAuth on Jan 9. News suggests OpenAI taking opposite approach.

**2026-01-21 01:50:** Web research completed
- Found OpenCode creator Dax Raad's announcement of OpenAI collaboration
- Discovered opencode-openai-codex-auth plugin with 22 model variants
- Confirmed strategic differentiation: OpenAI allows, Anthropic blocks

**2026-01-21 02:00:** Investigation completed
- Status: Complete
- Key outcome: OpenAI officially collaborating with OpenCode. GPT-5.2 Codex available via OAuth plugin. Same $200/mo cost as blocked Claude Max. Recommend adding as escape hatch backend.
