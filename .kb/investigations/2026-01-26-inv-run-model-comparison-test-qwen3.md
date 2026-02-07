<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** DeepSeek V3 is 18-32x cheaper than Qwen3-Max/Claude with acceptable quality for non-agentic work, but fails tool use reliability test (only 1/2 parallel calls made vs 2/2 for Qwen and Claude).

**Evidence:** Ran 3 tasks (coding, reasoning, tool use) across all models with API calls - DeepSeek made 1 tool call instead of 2, added unnecessary text asking about units.

**Knowledge:** DeepSeek's "unstable" function calling is real and observed in this test; Qwen3-Max has reliable tool use but is 3x slower than Claude; Claude remains best for agentic work.

**Next:** Keep current model strategy (Claude for orchestration, DeepSeek for cost-sensitive non-agentic work); Qwen3-Max is not compelling for either use case.

**Promote to Decision:** recommend-no (confirms existing strategy, no change needed)

---

# Investigation: Run Model Comparison Test Qwen3

**Question:** How do Qwen3-Max, DeepSeek V3, and Claude compare on response quality, latency, cost per task, and tool use reliability for representative agent tasks?

**Started:** 2026-01-26
**Updated:** 2026-01-26
**Owner:** Worker (investigation skill)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Patches-Decision:** N/A
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: DeepSeek V3 is 18-32x Cheaper per Task

**Evidence:** Measured cost per task using actual token counts:

| Task | DeepSeek V3 | Qwen3-Max | Claude Sonnet | Qwen vs DS | Claude vs DS |
|------|-------------|-----------|---------------|------------|--------------|
| Coding | $0.000032 | $0.002008 | $0.001395 | 62.8x | 43.6x |
| Reasoning | $0.000168 | $0.003068 | $0.005553 | 18.3x | 33.1x |
| Tool Use | $0.000125 | $0.000749 | $0.003336 | 6.0x | 26.7x |
| **TOTAL** | $0.000325 | $0.005825 | $0.010284 | **17.9x** | **31.7x** |

Pricing used (per MTok):
- DeepSeek V3: $0.25 input / $0.38 output
- Qwen3-Max: $1.20 input / $6.00 output (0-32K tier)
- Claude Sonnet 4.5: $3.00 input / $15.00 output

**Source:** curl API calls with token usage extracted from response objects

**Significance:** DeepSeek V3's cost advantage is substantial (~20x cheaper on average). For high-volume, cost-sensitive workloads that don't require reliable tool use, DeepSeek is the clear choice.

---

### Finding 2: Qwen3-Max is Significantly Slower Than Claude

**Evidence:** Latency measurements from API calls:

| Task | Qwen3-Max | DeepSeek V3 | Claude Sonnet |
|------|-----------|-------------|---------------|
| Coding | 9,012ms | 2,641ms | 2,277ms |
| Reasoning | 12,372ms | 12,029ms | 6,625ms |
| Tool Use | 4,219ms | 3,934ms | 1,701ms |
| **Average** | **8,534ms** | **6,201ms** | **3,534ms** |

**Source:** curl timing using `$(date +%s%N)` before/after API calls

**Significance:** Qwen3-Max is 2.4x slower than Claude on average. For interactive or latency-sensitive work, this is a significant disadvantage. DeepSeek is in the middle (1.8x slower than Claude).

---

### Finding 3: DeepSeek V3 Fails Multi-Tool Reliability Test

**Evidence:** When asked "What is the weather like in Tokyo and Paris?" with a `get_weather` tool:

| Model | Tool Calls Made | Correct Behavior | Notes |
|-------|-----------------|------------------|-------|
| **Qwen3-Max** | 2 (Tokyo, Paris) | ✅ Yes | Clean parallel calls, proper JSON |
| **DeepSeek V3** | 1 (Tokyo only) | ❌ No | Added text: "I need to know which temperature unit you prefer" |
| **Claude Sonnet** | 2 (Tokyo, Paris) | ✅ Yes | Clean parallel calls with brief intro text |

DeepSeek response included both `tool_calls` AND unnecessary clarification text, suggesting it wasn't confident about making both calls.

**Source:** `/tmp/qwen_tooluse.json`, `/tmp/deepseek_tooluse.json`, `/tmp/claude_tooluse.json`

**Significance:** This confirms the "unstable" function calling documented in DeepSeek's own API docs. For agent orchestration requiring reliable multi-tool execution, DeepSeek V3 is not suitable. Qwen3-Max passed this test.

---

### Finding 4: All Three Models Produce Correct Coding Output

**Evidence:** Asked all models to write a Go function checking IPv4 validity using net package:

- **Qwen3-Max**: Comprehensive solution with edge case handling (leading zeros, colon check), 325 tokens
- **DeepSeek V3**: Minimal but correct solution using net.ParseIP().To4(), 59 tokens
- **Claude Sonnet**: Balanced solution with colon check for IPv6 disambiguation, 84 tokens

All three solutions are functionally correct for standard use cases.

**Source:** `/tmp/qwen_result.json`, `/tmp/deepseek_result.json`, `/tmp/claude_result.json`

**Significance:** For simple coding tasks, all three models are capable. DeepSeek produces the most token-efficient (and cheapest) output.

---

### Finding 5: Reasoning Quality is Comparable Across Models

**Evidence:** Logic puzzle test (Alice/Bob/Carol in line with constraints):

All three models correctly applied step-by-step constraint satisfaction:
- Identified valid permutations through elimination
- Applied constraints systematically
- Reached correct answer: Bob (1st), Carol (2nd), Alice (3rd)

Token usage varied: DeepSeek 406, Claude 359, Qwen 500 (hit max_tokens limit)

**Source:** `/tmp/*_reasoning.json` files

**Significance:** For pure reasoning tasks without tool use, all three models are capable. Cost becomes the primary differentiator, favoring DeepSeek.

---

## Synthesis

**Key Insights:**

1. **DeepSeek V3 is cost king but tool-unreliable** - At 18-32x cheaper than alternatives, DeepSeek is compelling for high-volume work. However, the observed tool use failure (1/2 calls made) confirms it's unsuitable for agent orchestration.

2. **Qwen3-Max occupies an awkward middle ground** - More expensive than DeepSeek (18x), slower than Claude (2.4x), but with reliable tool use. There's no clear use case where it beats both alternatives.

3. **Claude remains best for agentic work** - Fastest latency (3.5s avg), reliable tool use (2/2 calls), and highest quality. The cost premium (32x over DeepSeek) is justified for orchestration.

**Answer to Investigation Question:**

| Metric | Best Choice | Runner-Up | Avoid |
|--------|------------|-----------|-------|
| **Cost** | DeepSeek V3 | Qwen3-Max | Claude |
| **Latency** | Claude Sonnet | DeepSeek V3 | Qwen3-Max |
| **Tool Use** | Claude = Qwen | - | DeepSeek V3 |
| **Coding Quality** | All comparable | - | - |
| **Reasoning** | All comparable | - | - |

**Recommendation:** Keep current model strategy:
- **Claude (Opus/Sonnet)** for orchestration and agent work requiring tool use
- **DeepSeek V3** for cost-sensitive bulk work without tool use requirements
- **Qwen3-Max** is not recommended - no use case where it's optimal

---

## Structured Uncertainty

**What's tested:**

- ✅ API connectivity and response quality (verified: 3 tasks x 3 models = 9 API calls)
- ✅ Token-based cost comparison (verified: extracted usage from all responses)
- ✅ Latency comparison (verified: timed all API calls)
- ✅ Multi-tool function calling (verified: 2-location weather query test)

**What's untested:**

- ⚠️ Streaming performance (not tested - may affect perceived latency)
- ⚠️ Complex multi-step orchestration (single tool call per model tested)
- ⚠️ Rate limit behavior under load (single requests only)
- ⚠️ Qwen thinking mode (`enable_thinking: true`) not compared

**What would change this:**

- If DeepSeek V4 (Feb 2026) fixes tool use reliability → re-evaluate for worker agents
- If Qwen3-Max latency improves significantly → could become middle-tier option
- If Claude pricing changes → cost calculations would shift

---

## Implementation Recommendations

**Purpose:** Confirm current model strategy is correct based on empirical testing.

### Recommended Approach ⭐

**No Change to Model Strategy** - Current approach (Claude for orchestration, DeepSeek for cost-sensitive) is validated by this test.

**Why this approach:**
- Claude's tool use reliability (2/2) vs DeepSeek (1/2) confirms orchestration requirement
- DeepSeek's 18-32x cost advantage validates it for bulk work
- Qwen3-Max doesn't fill any gap in current stack

**Trade-offs accepted:**
- Higher cost for Claude-based orchestration
- Not using latest models (Qwen3-Max) despite availability

**Implementation sequence:**
1. No changes needed - current model.go aliases are correct
2. Update model-selection.md to reference this investigation
3. Monitor DeepSeek V4 release (Feb 2026) for tool use improvements

### Alternative Approaches Considered

**Option B: Add Qwen3-Max as fallback for DeepSeek**
- **Pros:** Another cost option if DeepSeek unavailable
- **Cons:** 18x more expensive than DeepSeek, 2.4x slower than Claude
- **When to use instead:** Only if DeepSeek API becomes unavailable

**Option C: Use Qwen3-Max for long-context work (256K)**
- **Pros:** Largest context window of the three
- **Cons:** Gemini Flash handles large context with better economics
- **When to use instead:** If both DeepSeek and Gemini become unavailable

**Rationale for recommendation:** Empirical testing confirms existing strategy. No compelling reason to change.

---

### Implementation Details

**What to implement first:**
- Nothing - this validates current approach
- Consider adding this investigation to model-selection.md references

**Things to watch out for:**
- ⚠️ DeepSeek tool use remains "unstable" - don't use for agentic work
- ⚠️ Qwen3-Max latency (8.5s avg) may feel sluggish for interactive use
- ⚠️ All costs measured at 0-32K tier - Qwen costs increase at higher context lengths

**Areas needing further investigation:**
- Qwen3-Max thinking mode comparison (may improve reasoning at cost of latency)
- DeepSeek V4 tool use reliability when released
- Multi-step orchestration stress test (this was single-call only)

**Success criteria:**
- ✅ Current model strategy confirmed
- ✅ Empirical data available for future comparisons
- ✅ Clear guidance on when to use each model

---

## References

**Files Examined:**
- `pkg/model/model.go` - Checked existing Qwen aliases
- `.kb/guides/model-selection.md` - Current model strategy
- `.kb/investigations/2026-01-26-inv-investigate-qwen3-max-thinking-model.md` - Prior Qwen investigation

**Commands Run:**
```bash
# Qwen3-Max API test
curl -X POST "https://dashscope-intl.aliyuncs.com/compatible-mode/v1/chat/completions" \
  -H "Authorization: Bearer ${QWEN_MAX}" \
  -d '{"model": "qwen3-max", "messages": [...], "tools": [...]}'

# DeepSeek V3 API test
curl -X POST "https://api.deepseek.com/chat/completions" \
  -H "Authorization: Bearer ${DEEPSEEK_API_KEY}" \
  -d '{"model": "deepseek-chat", "messages": [...], "tools": [...]}'

# Claude Sonnet API test
curl -X POST "https://api.anthropic.com/v1/messages" \
  -H "x-api-key: ${ANTHROPIC_API_KEY}" \
  -d '{"model": "claude-sonnet-4-5-20250929", "messages": [...], "tools": [...]}'
```

**External Documentation:**
- Alibaba Cloud Model Studio - Qwen3-Max pricing/API
- DeepSeek API Docs - Function calling limitations
- Anthropic API Docs - Claude tool use

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-26-inv-investigate-qwen3-max-thinking-model.md` - Prior Qwen API validation
- **Investigation:** `.kb/investigations/2026-01-18-research-compare-deepseek-models-anthropic-models.md` - DeepSeek vs Claude research
- **Guide:** `.kb/guides/model-selection.md` - Current model strategy

---

## Investigation History

**2026-01-26 20:45:** Investigation started
- Initial question: Compare Qwen3-Max, DeepSeek V3, Claude on quality/latency/cost/tool use
- Context: Spawned to evaluate Qwen3-Max as potential addition to model stack

**2026-01-26 20:55:** API tests completed
- Ran coding, reasoning, and tool use tests on all 3 models
- Captured latency, token usage, and response quality

**2026-01-26 21:05:** Analysis completed
- DeepSeek V3 failed tool use test (1/2 calls)
- Cost ratios calculated: DS 1x, Qwen 18x, Claude 32x
- Latency: Claude fastest (3.5s), Qwen slowest (8.5s)

**2026-01-26 21:10:** Investigation completed
- Status: Complete
- Key outcome: Current model strategy validated; Qwen3-Max not compelling for any use case
