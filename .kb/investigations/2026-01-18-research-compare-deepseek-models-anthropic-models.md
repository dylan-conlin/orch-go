## Summary (D.E.K.N.)

**Delta:** DeepSeek offers 10-30x lower pricing than Claude but lacks native function calling (critical for agent orchestration), making Claude models essential for complex agentic workflows.

**Evidence:** DeepSeek R1 costs $0.45/$2.15/MTok vs Claude Opus 4.5 at $5/$25/MTok. DeepSeek R1-0528 added function calling but it remains "unstable" per their docs. Claude Opus 4.5 scores 80.9% on SWE-bench vs DeepSeek R1's 49.2%, and 62.3% vs 43.8% on scaled tool use benchmarks.

**Knowledge:** For agent orchestration requiring reliable tool use, Claude models remain necessary. DeepSeek's value is in cost-sensitive bulk processing or as a planning/reasoning layer in multi-agent setups where another model handles execution.

**Next:** Add DeepSeek aliases to orch-go for cost-sensitive workloads, but default remains Claude Opus for orchestration work.

**Promote to Decision:** recommend-no (tactical knowledge, not architectural - the current model selection architecture already supports this)

---

# Research: DeepSeek vs Anthropic Models for Agent Orchestration

**Question:** How do DeepSeek models (V3, R1, upcoming V4) compare to Claude (Sonnet 4.5, Opus 4.5) on price, quality, rate limits, API access, and agent orchestration suitability?

**Started:** 2026-01-18
**Updated:** 2026-01-18
**Owner:** Research agent (spawned)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: DeepSeek Pricing is 10-30x Cheaper

**Evidence:**

| Model | Input ($/MTok) | Output ($/MTok) | Notes |
|-------|----------------|-----------------|-------|
| DeepSeek V3.2 | $0.25 | $0.38 | Current general chat (Dec 2025) |
| DeepSeek V3.1 | $0.15 | $0.75 | Older V3 variant |
| DeepSeek R1-0528 | $0.45 | $2.15 | Reasoning model (reduced from $2.19) |
| Claude Opus 4.5 | $5.00 | $25.00 | Frontier reasoning |
| Claude Sonnet 4.5 | $3.00 | $15.00 | Standard (doubles at >200K context) |
| Claude Haiku 4.5 | $1.00 | $5.00 | Fast, lightweight |

**Source:** [DeepSeek API Pricing](https://api-docs.deepseek.com/quick_start/pricing), [Claude Pricing](https://platform.claude.com/docs/en/about-claude/pricing), [CometAPI Guide](https://www.cometapi.com/the-guide-to-claude-opus-4--4-5-api-pricing-in-2026/)

**Significance:** At comparable output volumes, DeepSeek V3 is ~65x cheaper than Claude Opus and ~40x cheaper than Sonnet. DeepSeek R1 is ~12x cheaper than Opus for reasoning workloads. This makes DeepSeek attractive for high-volume, cost-sensitive use cases.

---

### Finding 2: Claude Dominates Quality Benchmarks, Especially Agentic Tasks

**Evidence:**

| Benchmark | Claude Opus 4.5 | DeepSeek R1 | Notes |
|-----------|-----------------|-------------|-------|
| SWE-bench Verified | 80.9% | 49.2% | Real-world coding (massive gap) |
| Scaled Tool Use | 62.3% | 43.8% | Multi-step agentic tasks |
| MATH-500 | ~96.2% (Sonnet 3.7 thinking) | 97.3% | Math reasoning (R1 slightly better) |
| Codeforces | N/A | 96.3 percentile | Competitive programming |
| Instruction Following | 93.2% (Claude 3.7) | 83.3% | Following directions |

**Source:** [DataCamp Opus 4.5 Guide](https://www.datacamp.com/blog/claude-opus-4-5), [Medium R1-0528 Analysis](https://medium.com/@leucopsis/deepseeks-new-r1-0528-performance-analysis-and-benchmark-comparisons-6440eac858d6), [Artificial Analysis Comparison](https://artificialanalysis.ai/models/comparisons/deepseek-r1-vs-claude-4-opus-thinking)

**Significance:** For agent orchestration (which requires SWE-bench-like real-world coding and scaled tool use), Claude Opus has a substantial lead. DeepSeek excels at pure reasoning/math benchmarks but falls behind on practical software engineering and agentic execution.

---

### Finding 3: DeepSeek Function Calling is Unstable

**Evidence:**

> "The current version of the deepseek-chat model's Function Calling capability is unstable, which may result in looped calls or empty responses." - [DeepSeek API Docs](https://api-docs.deepseek.com/)

> "DeepSeek-R1 doesn't support function calling, which makes developing agents with it quite challenging." - [Plain English AI Article](https://ai.plainenglish.io/function-calling-the-superpower-deepseek-r1-doesnt-have-yet-c80ef6f9057d)

The R1-0528 update (May 2025) added function calling but it remains in unstable/testing status. Workarounds include:
- Using DeepSeek R1 as a "planner" with another model handling execution
- Text-based command parsing (not native function calling)
- LlamaIndex ReActAgent wrapper

**Source:** [DeepSeek R1 GitHub Issue #9](https://github.com/deepseek-ai/DeepSeek-R1/issues/9), [LlamaIndex Integration Guide](https://www.dataleadsfuture.com/integrating-llamaindex-and-deepseek-r1-for-reasoning_content-and-function-call-features-2/)

**Significance:** This is a **blocker for direct use in orch-go agent orchestration**. Our spawn architecture relies on reliable tool use. Claude's function calling is battle-tested; DeepSeek's is experimental. For orchestration work requiring multi-step tool execution, Claude remains necessary.

---

### Finding 4: Rate Limits Favor DeepSeek for Bulk Work

**Evidence:**

**DeepSeek:**
- No explicit rate limits - "DeepSeek API does NOT constrain user's rate limit"
- Dynamic throttling under load (slower responses, not rejections)
- 30-minute timeout for long requests
- Soft limits based on overall platform traffic

**Claude (Max Subscription):**
- Pro: ~20 turns/day effective limit
- Max 5x: ~100 turns/day
- Max 20x: ~400 turns/day
- API: Hard rate limits per tier

**Claude (API):**
- Tier-based rate limits (tokens per minute)
- Batch API: 50% discount for non-urgent work
- Prompt caching: 90% discount on cached tokens

**Source:** [DeepSeek Rate Limit Docs](https://api-docs.deepseek.com/quick_start/rate_limit), [Simon Willison's Notes](https://simonwillison.net/2025/jan/18/deepseek-api-docs-rate-limit/)

**Significance:** DeepSeek's lack of hard rate limits makes it suitable for burst/batch workloads. Claude Max subscription has usage caps that can constrain high-volume orchestration. For sustained agent work, DeepSeek's approach is more permissive (at cost of potential slowdowns).

---

### Finding 5: DeepSeek Available Through Multiple Providers

**Evidence:**

**Direct Access:**
- platform.deepseek.com - Official API
- OpenAI SDK compatible (change base URL)
- 128K token context window (V3/R1)

**Third-Party Providers:**
- Amazon Bedrock: DeepSeek-V3.1 available globally (data stays in AWS)
- OpenRouter: DeepSeek V3 and R1 available with automated caching
- Together.ai: DeepSeek R1 (special rate limits due to demand)
- Self-hosted: MIT-licensed, can run on own infrastructure

**Source:** [AWS Bedrock Announcement](https://www.aboutamazon.com/news/aws/alibaba-qwen3-deepseek-v3-amazon-bedrock), [OpenRouter DeepSeek V3](https://openrouter.ai/deepseek/deepseek-chat), [OpenRouter R1](https://openrouter.ai/deepseek/deepseek-r1)

**Significance:** DeepSeek can be integrated through enterprise-grade channels (Bedrock) with data privacy guarantees. This removes some concerns about direct API usage with a Chinese company. For orch-go, OpenRouter provides a unified gateway.

---

### Finding 6: Upcoming DeepSeek V4 (Feb 2026) May Change Landscape

**Evidence:**

- Expected release: Mid-February 2026 (around Chinese New Year)
- Reported to outperform Claude 3.5 Sonnet and GPT-4o in coding tasks (internal testing)
- Optimized for "extremely long" contexts (exceeding 128K)
- Focused on multi-file architectures and complex coding

**Note:** DeepSeek R2 appears to have been merged into V4 rather than being released separately.

**Source:** [GIGAZINE V4 Report](https://gigazine.net/gsc_news/en/20260114-deepseek-next-flagship-ai-model-v4/), [Decrypt Insider Report](https://decrypt.co/354177/insiders-say-deepseek-v4-beat-claude-chatgpt-coding-launching-weeks)

**Significance:** The model landscape is shifting. V4 could close the gap on coding benchmarks. However, this doesn't address the function calling stability issue which is architectural. Worth monitoring for re-evaluation.

---

### Finding 7: Claude Opus 4.5 Has Advanced Tool Features

**Evidence:**

Claude Opus 4.5 includes three advanced tool-use features:

1. **Tool Search Tool**: Dynamically discover/load tools from large libraries (79.5% → 88.1% improvement)
2. **Programmatic Tool Calling**: Direct Python execution, ~37% token reduction
3. **Tool Use Examples**: Complex parameter accuracy 72% → 90%

Additional capabilities:
- "Effort" parameter to control token spending (high/medium/low)
- At "medium effort" beats Sonnet on SWE-bench while using 76% fewer tokens
- Self-refining agents reach peak performance in 4 iterations (others need 10+)

**Source:** [Anthropic Advanced Tool Use](https://www.anthropic.com/engineering/advanced-tool-use), [DataCamp Opus 4.5 Guide](https://www.datacamp.com/blog/claude-opus-4-5)

**Significance:** For agent orchestration, these features directly improve reliability and efficiency. Tool Search Tool in particular enables scaling to many tools without context bloat - exactly what orch-go needs for skill-based spawning.

---

## Synthesis

**Key Insights:**

1. **Price vs Capability Trade-off is Real** - DeepSeek offers dramatic cost savings (10-30x) but at the expense of agentic reliability. For pure reasoning tasks (math, planning), DeepSeek is competitive. For execution (tool use, coding implementation), Claude leads significantly.

2. **Function Calling is the Differentiator** - Agent orchestration requires reliable tool/function calling. DeepSeek's "unstable" function calling is disqualifying for primary orchestration use. However, DeepSeek can serve as a planning/reasoning layer with Claude handling execution.

3. **Access Parity Achieved** - Both providers are available through enterprise channels (Bedrock, OpenRouter). Data residency and compliance concerns can be addressed via AWS for DeepSeek.

4. **Rate Limit Philosophy Differs** - DeepSeek's "no hard limits" approach favors burst workloads. Claude's subscription tiers favor predictable usage. For high-volume orchestration, DeepSeek's model is more forgiving.

**Answer to Investigation Question:**

For orch-go agent orchestration, **Claude Opus 4.5 remains the correct default**:
- 80.9% SWE-bench vs 49.2% (massive real-world coding gap)
- 62.3% scaled tool use vs 43.8% (agentic execution)
- Stable, battle-tested function calling vs "unstable"
- Advanced tool features (Tool Search Tool, effort control)

**DeepSeek's role** in the architecture should be:
- **Cost-sensitive bulk work**: Large context analysis, document processing
- **Planning layer**: R1 for reasoning/planning, Claude for execution
- **Rate-limited escape hatch**: When Claude Max is exhausted, DeepSeek V3 for non-agentic work

Do NOT use DeepSeek R1 as the primary model for agent spawns that require tool execution.

---

## Structured Uncertainty

**What's tested:**

- ✅ Pricing verified from official documentation
- ✅ Benchmark scores from third-party analysis sites and official announcements
- ✅ Function calling stability documented in DeepSeek's own API docs
- ✅ Rate limit behavior documented officially

**What's untested:**

- ⚠️ DeepSeek V4 capabilities (not yet released)
- ⚠️ Real-world orchestration performance with DeepSeek (not tested in orch-go)
- ⚠️ Multi-agent DeepSeek+Claude hybrid performance (architectural hypothesis)
- ⚠️ OpenRouter/Bedrock latency vs direct API

**What would change this:**

- DeepSeek V4 with stable function calling → re-evaluate as primary option
- DeepSeek R1 function calling stabilizes → re-evaluate for worker agents
- Claude pricing increases significantly → DeepSeek becomes more attractive despite limitations
- orch-go adds planning-only agent tier → DeepSeek R1 viable for that tier

---

## Implementation Recommendations

**Purpose:** Integrate DeepSeek as cost-sensitive alternative without disrupting orchestration reliability.

### Recommended Approach ⭐

**Add DeepSeek aliases to pkg/model while keeping Claude as default**

**Why this approach:**
- Enables cost-sensitive workloads (flash-equivalent alternative)
- No changes to orchestration reliability (default stays Opus)
- Aligns with existing multi-provider architecture

**Trade-offs accepted:**
- DeepSeek models won't be suitable for most orchestration work
- Need documentation to guide model selection

**Implementation sequence:**
1. Add `deepseek` and `deepseek-r1` aliases to pkg/model/model.go
2. Update model selection guide with DeepSeek guidance
3. Document limitations (no agentic use for R1)

### Alternative Approaches Considered

**Option B: Use DeepSeek as default for cost savings**
- **Pros:** Massive cost reduction
- **Cons:** Function calling instability would break agent execution
- **When to use instead:** Never for orchestration; only for non-agentic batch work

**Option C: Hybrid planning/execution architecture**
- **Pros:** Uses R1's reasoning strength + Claude's execution reliability
- **Cons:** Adds complexity, latency, debugging difficulty
- **When to use instead:** If cost becomes critical constraint AND R1 proves stable enough

**Rationale for recommendation:** The current architecture works well. Adding DeepSeek as an option (not default) provides flexibility without risk.

---

### Implementation Details

**What to implement first:**
- Add aliases: `deepseek` → `deepseek/deepseek-chat` (V3), `deepseek-r1` → `deepseek/deepseek-reasoner`
- These map to OpenRouter endpoints which orch-go can use

**Things to watch out for:**
- ⚠️ Do not use DeepSeek for agents requiring tool execution
- ⚠️ OpenRouter may have different rate limits than direct API
- ⚠️ Context window differences (128K DeepSeek vs 200K+ Claude)

**Areas needing further investigation:**
- DeepSeek V4 launch (Feb 2026) - reassess
- Hybrid architecture feasibility if R1 function calling stabilizes
- Direct DeepSeek API integration vs OpenRouter trade-offs

**Success criteria:**
- ✅ DeepSeek aliases work for non-agentic spawns
- ✅ Default (Opus) unchanged for orchestration work
- ✅ Model selection guide documents when to use each

---

## References

**External Documentation:**
- [DeepSeek API Pricing](https://api-docs.deepseek.com/quick_start/pricing) - Official pricing
- [DeepSeek Rate Limits](https://api-docs.deepseek.com/quick_start/rate_limit) - Official rate limit policy
- [Claude Pricing](https://platform.claude.com/docs/en/about-claude/pricing) - Official Anthropic pricing
- [Anthropic Advanced Tool Use](https://www.anthropic.com/engineering/advanced-tool-use) - Tool features announcement
- [OpenRouter DeepSeek](https://openrouter.ai/deepseek/deepseek-chat) - Third-party access
- [AWS Bedrock DeepSeek](https://www.aboutamazon.com/news/aws/alibaba-qwen3-deepseek-v3-amazon-bedrock) - Enterprise access

**Benchmark Sources:**
- [DataCamp Opus 4.5 Guide](https://www.datacamp.com/blog/claude-opus-4-5) - Claude benchmarks
- [Artificial Analysis](https://artificialanalysis.ai/models/comparisons/deepseek-r1-vs-claude-4-opus-thinking) - Model comparisons
- [Medium R1-0528 Analysis](https://medium.com/@leucopsis/deepseeks-new-r1-0528-performance-analysis-and-benchmark-comparisons-6440eac858d6) - R1 benchmarks

**Related Artifacts:**
- **Guide:** `.kb/guides/model-selection.md` - Existing model guidance
- **Guide:** `.kb/models/model-access-spawn-paths.md` - Spawn architecture

---

## Investigation History

**2026-01-18 12:00:** Investigation started
- Initial question: Compare DeepSeek vs Anthropic for agent orchestration
- Context: Evaluating alternative models for orch-go cost optimization

**2026-01-18 12:30:** Research completed
- Status: Complete
- Key outcome: DeepSeek is 10-30x cheaper but lacks stable function calling; Claude remains essential for agent orchestration; DeepSeek viable for cost-sensitive non-agentic work
