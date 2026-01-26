<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Qwen3-Max is a capable OpenAI-compatible model with working function calling and thinking mode, priced between DeepSeek V3 and Claude Sonnet.

**Evidence:** Tested API directly - function calling returned valid tool_calls, thinking mode exposed reasoning_content field with 382 reasoning tokens.

**Knowledge:** Qwen3-Max offers a middle-tier cost option ($1.2/$6/MTok) with 256K context and thinking mode at no extra cost; viable for cost-sensitive workers when DeepSeek is unavailable.

**Next:** Do not integrate now - wait for operational need; add aliases to model.go only if/when Qwen becomes primary cost-savings path.

**Promote to Decision:** recommend-no (market research, not architectural change)

---

# Investigation: Investigate Qwen3 Max Thinking Model

**Question:** Should we integrate Qwen3-Max Thinking into orch-go's model system? What are the pricing, capabilities, tool use support, and API compatibility?

**Started:** 2026-01-26
**Updated:** 2026-01-26
**Owner:** Worker (architect skill)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Qwen3-Max API is OpenAI-Compatible with Working Function Calling

**Evidence:** Direct API testing with provided key (QWEN_MAX=sk-351ff6...) confirmed:
- Models endpoint returns 110+ models including `qwen3-max`, `qwen3-max-2026-01-23` (thinking mode snapshot)
- Function calling works correctly - returned properly structured tool_calls:
```json
{
  "tool_calls": [{
    "function": {"name": "get_weather", "arguments": "{\"location\": \"San Francisco\"}"},
    "type": "function"
  }],
  "finish_reason": "tool_calls"
}
```
- API endpoint: `https://dashscope-intl.aliyuncs.com/compatible-mode/v1`

**Source:** curl tests against live API with provided key

**Significance:** Integration is technically straightforward - would use `@ai-sdk/openai-compatible` in OpenCode and simple alias mapping in orch-go's pkg/model/model.go

---

### Finding 2: Thinking Mode Exposes Reasoning Content Similar to DeepSeek R1

**Evidence:** Testing `qwen3-max-2026-01-23` with `enable_thinking: true` produced:
- `reasoning_content` field containing chain-of-thought (382 tokens in test)
- Response includes `completion_tokens_details.reasoning_tokens` count
- Jan 2026 snapshot "effectively integrates thinking and non-thinking mode" with "interleaved thinking with three built-in tools: web search, webpage extraction, and code interpreter"

**Source:** curl test with enable_thinking parameter, Alibaba Cloud documentation

**Significance:** Thinking mode is comparable to DeepSeek R1's reasoning approach. However, thinking mode pricing is identical to non-thinking mode - no premium for reasoning.

---

### Finding 3: Pricing is Mid-Tier - Between DeepSeek and Claude

**Evidence:** Pricing comparison (per MTok):

| Model | Input | Output | Context | Notes |
|-------|-------|--------|---------|-------|
| **DeepSeek V3** | $0.25 | $0.38 | 64K | Cheapest option |
| **Qwen3-Max** | $1.2-$3 | $6-$15 | 256K | Tiered by context (0-32K/32K-128K/128K-252K) |
| **Claude Sonnet 4.5** | $3 | $15 | 200K | API pricing |
| **Claude Opus 4.5** | $200/mo flat | - | 200K | Max subscription |

Cache pricing: $0.24/MTok (80% discount from input cost)

**Source:** Alibaba Cloud Model Studio pricing, pricepertoken.com, internal model-selection.md

**Significance:** Qwen3-Max is ~5x more expensive than DeepSeek V3 but comparable to Claude Sonnet. Not a cost-savings opportunity compared to current stack (DeepSeek for cost, Opus/Max for quality).

---

### Finding 4: Benchmark Performance is Strong but Not Leading

**Evidence:** Benchmark comparisons:
- **Math (AIME'25):** Qwen3-Max 81.6 vs Gemini 2.5 Pro (higher) vs Opus 4 (unspecified)
- **Coding (LiveCodeBench v6):** 74.8
- **SWE-Bench Verified:** 72.5 (Claude Sonnet 4.5: 64.8%)
- **MATH Level 5:** 97.1% (6th place, GPT-5 leads at 97.9%)

Qwen3-235B-A22B (thinking mode) outperforms DeepSeek-R1 on 17/23 benchmarks per Qwen technical report.

**Source:** lmcouncil.ai/benchmarks, apidog.com/blog/qwen3-max/, datacamp.com/blog/qwen3

**Significance:** Competitive quality for workers but doesn't justify switching from current Opus (orchestration) + DeepSeek (cost-sensitive workers) strategy.

---

### Finding 5: Multiple Provider Routes Available

**Evidence:** Qwen3-Max accessible via:
1. **Direct Alibaba Cloud API** - `https://dashscope-intl.aliyuncs.com/compatible-mode/v1` (tested, working)
2. **OpenRouter** - `qwen3-max` model ID, OpenAI-compatible routing
3. **vLLM** - Self-hosted with `--tool-call-parser hermes` for function calling

**Source:** OpenRouter docs, vLLM documentation, Alibaba Cloud docs

**Significance:** If we need Qwen, OpenRouter is easiest (already integrated in OpenCode). Direct API requires separate auth setup.

---

## Synthesis

**Key Insights:**

1. **Technically Sound, Not Economically Compelling** - Qwen3-Max works well (tested function calling, thinking mode) but occupies an awkward middle ground: more expensive than DeepSeek ($1.2/$6 vs $0.25/$0.38) without the quality justification for switching from Opus for critical work.

2. **Thinking Mode at No Premium** - Unlike some reasoning models, Qwen3-Max's thinking mode has identical pricing to non-thinking mode. This makes it potentially attractive for reasoning-heavy tasks if cost optimization from Opus is needed.

3. **256K Context is Largest** - The 256K context window exceeds Claude (200K), making it potentially useful for large-context work. However, Gemini Flash handles this with better economics (free tier).

**Answer to Investigation Question:**

**Should we integrate Qwen3-Max?** Not now. The model works well technically but doesn't fit a gap in our current stack:

- **For orchestration:** Opus via Claude Max ($200/mo flat) is superior quality and predictable cost
- **For cost-sensitive workers:** DeepSeek V3 ($0.25/$0.38/MTok) is 5x cheaper than Qwen3-Max
- **For large context:** Gemini Flash handles this (though with TPM limits)

**When to reconsider:**
- If DeepSeek becomes unavailable (rate limits, policy changes)
- If we need 256K+ context regularly
- If benchmark improvements make it clearly superior for specific task types

---

## Structured Uncertainty

**What's tested:**

- ✅ API connectivity and authentication (verified: curl to models endpoint returned 110 models)
- ✅ Function calling works correctly (verified: tool_calls returned with proper structure)
- ✅ Thinking mode works (verified: reasoning_content field returned with 382 tokens)
- ✅ OpenAI-compatible format (verified: standard chat/completions endpoint)

**What's untested:**

- ⚠️ Streaming performance (not benchmarked - may be slower than Claude/DeepSeek)
- ⚠️ Complex multi-tool orchestration (only single tool tested)
- ⚠️ Agent behavior in production (no real spawn tests)
- ⚠️ Rate limits and quotas (not tested)

**What would change this:**

- If DeepSeek pricing increases or availability decreases, Qwen becomes viable fallback
- If Qwen3-Max-2026-01-23's "built-in tools" (web search, code interpreter) provide value Claude lacks
- If Opus becomes unavailable and we need alternative reasoning model

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Wait-and-See with Documented Path** - Do not integrate now; document the integration path for future use.

**Why this approach:**
- Current stack (Opus for quality, DeepSeek for cost) covers all use cases
- Integration has engineering cost without clear benefit
- Documented path enables quick integration if needs change

**Trade-offs accepted:**
- May miss potential quality/cost improvements in edge cases
- Won't have production experience if we suddenly need it

**Implementation sequence if needed later:**
1. Add aliases to `pkg/model/model.go`: `qwen`, `qwen-max`, `qwen-thinking`
2. Configure OpenCode via opencode.json with Alibaba Cloud endpoint
3. Test spawn with `--model qwen` against real tasks

### Alternative Approaches Considered

**Option B: Integrate via OpenRouter**
- **Pros:** Already configured, no new auth needed, unified billing
- **Cons:** Additional routing latency, OpenRouter markup
- **When to use instead:** If quick access needed without setup

**Option C: Full Direct Integration Now**
- **Pros:** Ready for use immediately, production-tested
- **Cons:** Engineering cost for unclear benefit, another model to maintain
- **When to use instead:** If we have a specific task Qwen excels at

**Rationale for recommendation:** No compelling use case exists today. Engineering effort is better spent elsewhere.

---

### Implementation Details (If Needed)

**What to implement first:**
- Add to `pkg/model/model.go` Aliases map:
```go
"qwen":      {Provider: "alibaba", ModelID: "qwen3-max"},
"qwen-max":  {Provider: "alibaba", ModelID: "qwen3-max"},
"qwen-thinking": {Provider: "alibaba", ModelID: "qwen3-max-2026-01-23"},
```

**Things to watch out for:**
- ⚠️ `enable_thinking` parameter only valid for thinking model snapshots
- ⚠️ Tiered pricing (costs jump at 32K and 128K tokens) - may need tracking
- ⚠️ Different endpoint from other providers (`dashscope-intl.aliyuncs.com`)

**Areas needing further investigation:**
- Rate limits and quotas for provided API key
- Whether thinking mode's built-in tools (web search, code interpreter) are useful
- Streaming performance comparison

**Success criteria:**
- ✅ `orch spawn --model qwen investigation "test"` creates working session
- ✅ Function calling works in real agent task
- ✅ Cost tracking accurately captures tiered pricing

---

## References

**Files Examined:**
- `pkg/model/model.go` - Current model alias system, would need Qwen aliases
- `.kb/guides/model-selection.md` - Current model strategy documentation
- `.kb/models/current-model-stack.md` - Current operational stack

**Commands Run:**
```bash
# List available models (verified 110+ models including qwen3-max variants)
curl https://dashscope-intl.aliyuncs.com/compatible-mode/v1/models -H "Authorization: Bearer sk-..."

# Test function calling (verified tool_calls returned correctly)
curl https://dashscope-intl.aliyuncs.com/compatible-mode/v1/chat/completions -d @test.json

# Test thinking mode (verified reasoning_content field)
curl https://dashscope-intl.aliyuncs.com/compatible-mode/v1/chat/completions -d '{"enable_thinking":true}'
```

**External Documentation:**
- [Alibaba Cloud Model Studio - Models](https://www.alibabacloud.com/help/en/model-studio/models)
- [Alibaba Cloud Model Studio - Pricing](https://www.alibabacloud.com/help/en/model-studio/billing-for-model-studio)
- [Qwen Function Calling Guide](https://www.alibabacloud.com/help/en/model-studio/qwen-function-calling)
- [Qwen3 Technical Report](https://arxiv.org/pdf/2505.09388)
- [DeepWiki - Qwen3 Function Calling](https://deepwiki.com/QwenLM/Qwen3/4.3-function-calling-and-tool-use)
- [LM Council Benchmarks Jan 2026](https://lmcouncil.ai/benchmarks)

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-18-max-subscription-primary-spawn-path.md` - Why Claude Max is primary
- **Decision:** `.kb/decisions/2026-01-21-gpt-unsuitable-for-orchestration.md` - Model selection criteria
- **Guide:** `.kb/guides/model-selection.md` - How to choose models

---

## Investigation History

**2026-01-26 19:50:** Investigation started
- Initial question: Should we integrate Qwen3-Max Thinking into orch-go?
- Context: Blog post at qwen.ai/blog?id=qwen3-max-thinking, API key provided

**2026-01-26 20:15:** API testing completed
- Function calling works correctly
- Thinking mode exposes reasoning_content field
- Pricing researched across multiple sources

**2026-01-26 20:30:** Investigation completed
- Status: Complete
- Key outcome: Qwen3-Max works well but doesn't fill a gap in current stack; recommend wait-and-see
