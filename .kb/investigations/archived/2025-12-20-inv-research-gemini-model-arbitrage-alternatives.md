**TLDR:** Gemini 2.0/3.0 and DeepSeek V3/R1 provide the strongest alternatives to Claude for model arbitrage in late 2025. DeepSeek is the price-performance leader for reasoning ($0.28/1M), while Gemini excels in multimodality and large context (1M+ tokens). High confidence (85%) based on current pricing and benchmark data.

---

# Investigation: Gemini 2.0 and Model Arbitrage Alternatives (Late 2025)

**Question:** What are the best alternatives to Claude for model arbitrage in agentic workflows, specifically focusing on Gemini 2.0 and other frontier models?

**Started:** 2025-12-20
**Updated:** 2025-12-20
**Owner:** opencode
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: Gemini 2.0/3.0 Landscape
**Evidence:** Gemini 2.0 Flash and Pro (and their 2.5/3.0 successors) offer a unique combination of speed, multimodality, and massive context.
- **Gemini 2.0 Flash:** ~$0.10-$0.30/1M input, 1M token context, Multimodal Live API.
- **Gemini 2.0 Pro:** ~$1.25-$2.00/1M input, 1M token context, superior reasoning/coding.
- **Key Advantage:** Native audio/video reasoning and the largest stable context window in the market.

**Source:** `.kb/investigations/2025-12-20-inv-research-gemini-2-0-models.md`

**Significance:** Gemini is the primary choice for "context-heavy" arbitrage where Claude's 200K "context cliff" or pricing becomes prohibitive.

---

### Finding 2: DeepSeek V3/R1 - The Price-Performance Leader
**Evidence:** DeepSeek V3.2 and R1 (Reasoning) models significantly undercut Western frontier models while matching their performance.
- **Pricing:** $0.28/1M input, $0.42/1M output (Native API).
- **Performance:** Matches or exceeds Llama 3.1 405B and Claude 3.5 Sonnet on many benchmarks.
- **R1 Reasoning:** Provides o1-level chain-of-thought reasoning at a fraction of the cost.

**Source:** `.kb/investigations/2025-12-20-research-deepseek-llama-arbitrage-comparison.md`

**Significance:** DeepSeek is the "Tier 2/3" arbitrage winner for non-compliance-restricted workflows, offering frontier intelligence at commodity prices.

---

### Finding 3: Llama 4 and Groq - The Speed Kings
**Evidence:** Llama 4 (Scout/Maverick) and Llama 3.3 70B on Groq provide the lowest latency for agentic "fast-loops".
- **Speed:** 400-600+ tokens per second (TPS).
- **Pricing:** Llama 4 Scout at ~$0.11/1M input; Llama 3.3 70B at ~$0.59/1M input.
- **Key Advantage:** Instantaneous response for routing, triage, and simple tool-use.

**Source:** `.kb/investigations/2025-12-20-research-deepseek-llama-arbitrage-comparison.md`

**Significance:** Essential for the "Tier 1" (Routing) layer of an arbitrage system to minimize agent "thinking" latency.

---

## Synthesis

**Key Insights:**

1. **The Three-Tier Arbitrage Strategy** - Effective agent orchestration requires three distinct tiers:
   - **Tier 1 (Routing):** Llama 4 Scout (Groq) for <100ms intent detection.
   - **Tier 2 (Execution):** DeepSeek V3.2 or Gemini 2.0 Flash for general tasks and coding.
   - **Tier 3 (Reasoning):** DeepSeek R1 or Claude 4.5 Sonnet for complex planning and debugging.

2. **Context-Based Routing** - Models should be selected not just by "intelligence" but by "context volume". Gemini is the clear winner for tasks requiring >200K tokens of context, while Claude/DeepSeek are better for dense, high-logic tasks within smaller windows.

3. **Multimodal Arbitrage** - Gemini 2.0's Live API creates a new category of arbitrage for real-time interactive agents that Claude and others cannot currently match in latency or native capability.

**Answer to Investigation Question:**
The best alternatives to Claude for model arbitrage are **DeepSeek V3/R1** (for cost-effective reasoning and general tasks) and **Gemini 2.0/3.0** (for large-context and multimodal tasks). **Llama 4 on Groq** remains the superior choice for the routing/triage layer. A hybrid strategy using these models can reduce agentic spend by 70-90% compared to a "Sonnet-only" approach.

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**
Pricing and benchmark data are current as of late 2025. The arbitrage tiers are based on established industry patterns.

**What's certain:**
- ✅ DeepSeek's price leadership.
- ✅ Gemini's context window and multimodal advantages.
- ✅ Groq's speed advantage for Llama models.

**What's uncertain:**
- ⚠️ Long-term stability of DeepSeek's native API.
- ⚠️ Exact performance of Llama 4 Maverick vs. DeepSeek V3.2 in complex agentic loops.

**What would increase confidence to Very High (95%+):**
- Direct side-by-side testing of these models on the project's specific agentic tasks.

---

## Implementation Recommendations

### Recommended Approach ⭐

**Dynamic Model Router** - Implement a routing layer in `orch-go` that selects models based on task type, context size, and required reasoning depth.

**Why this approach:**
- **Cost Efficiency:** Routes 80% of tasks to models costing <$0.30/1M.
- **Performance:** Uses Groq for low-latency triage and Gemini for high-context analysis.
- **Resilience:** Allows falling back to Claude if cheaper models fail or are rate-limited.

**Implementation sequence:**
1. **Add Provider Support:** Ensure `orch-go` can talk to DeepSeek, Google (Gemini), and Groq/Together (Llama).
2. **Implement Triage Layer:** Use Llama 4 Scout to classify incoming tasks.
3. **Context-Aware Routing:** Automatically switch to Gemini for prompts >128K tokens.

---

## References

**Files Examined:**
- `.kb/investigations/2025-12-20-inv-research-gemini-2-0-models.md`
- `.kb/investigations/2025-12-20-research-deepseek-llama-arbitrage-comparison.md`
- `.kb/investigations/2025-12-20-research-model-arbitrage-api-vs-max.md`

**External Documentation:**
- [Google AI Pricing](https://ai.google.dev/pricing)
- [DeepSeek API Docs](https://api-docs.deepseek.com/)
- [Groq Pricing](https://groq.com/pricing/)
- [Together AI Pricing](https://www.together.ai/pricing)

---

## Investigation History

**2025-12-20 12:00:** Investigation started
- Researched Gemini 2.0 family (Flash, Pro, Experimental).
- Researched DeepSeek V3/R1 and Llama 3.x/4 alternatives.
- Synthesized findings into a 3-tier arbitrage strategy.

**2025-12-20 13:00:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Recommended a hybrid 3-tier arbitrage strategy using DeepSeek, Gemini, and Llama.
