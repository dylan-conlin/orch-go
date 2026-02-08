<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** GLM 4.7 is a strong coding model at exceptional value ($3-60/mo), but OpenCode integration has known issues that make it unsuitable for orch-go's current architecture.

**Evidence:** Reddit reports OpenCode issues (tool calls in thinking tags, timeouts); official docs confirm Claude Code works better; concurrency limits (1-5 on GLM 4.7 vs 12+ on 4.6) conflict with multi-agent spawning.

**Knowledge:** z.ai GLM works via ANTHROPIC_BASE_URL override with Claude Code (direct connection) but suffers from gateway issues with OpenCode; current Gemini Flash stack is more reliable.

**Next:** WATCH status - revisit when OpenCode-GLM issues stabilize or consider for Claude CLI escape hatch only.

**Promote to Decision:** recommend-no (tactical observation, not architectural change)

---

# Investigation: GLM 4.7 from z.ai for Orchestration Use

**Question:** Should orch-go adopt z.ai's GLM 4.7 as a model option for agent spawning?

**Started:** 2026-01-22
**Updated:** 2026-01-22
**Owner:** Worker agent (spawned by orchestrator)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Patches-Decision:** .kb/decisions/2026-01-18-max-subscription-primary-spawn-path.md (extends model considerations)
**References-Superseded:** .kb/decisions/2026-01-09-abandon-claude-max-oauth-use-gemini-primary.md (historical context, superseded Jan 18)

---

## Findings

### Finding 1: GLM 4.7 Model Capabilities Are Competitive

**Evidence:**
- LiveCodeBench V6: 84.9% (vs Claude Sonnet 4.5 at 64.0%)
- SWE-bench Verified: 73.8% (+5.8% over GLM 4.6)
- SWE-bench Multilingual: 66.7% (+12.9% over predecessor)
- Terminal Bench 2.0: 41% (+16.5%)
- Context window: 200K tokens
- Max output: 128K tokens
- Architecture: 358B parameters, Mixture-of-Experts (MoE)

**Key features:**
- Interleaved Thinking: Reasons before every response/tool call
- Preserved Thinking: Retains reasoning across multi-turn conversations
- Turn-level Thinking: Toggle reasoning per session

**Source:**
- [Z.AI GLM-4.7 Blog](https://z.ai/blog/glm-4.7)
- [GLM-4.7: Pricing, Benchmarks Analysis](https://llm-stats.com/blog/research/glm-4.7-launch)
- [Vertu Comparison](https://vertu.com/lifestyle/glm-4-7-vs-claude-opus-4-5-the-thinking-open-source-challenger/)

**Significance:** GLM 4.7 is a legitimate frontier-tier coding model. Real-world testing shows it's "best at iterative implementation (fewer context losses)" while Opus is "best at first-pass architecture." This positions GLM 4.7 as potentially useful for worker agents doing iterative implementation.

---

### Finding 2: Pricing Is Extremely Competitive (But Possibly Unsustainable)

**Evidence:**

**Coding Plans (subscription):**
| Plan | Price | Prompts/5hr | MCP Quota |
|------|-------|-------------|-----------|
| Lite | $3/mo | 120 | 100 web searches/mo |
| Pro | $15/mo | 600 | 1,000 web searches/mo |
| Max | $60/mo | 2,400 | 4,000 web searches/mo |

**API Pricing (per 1M tokens):**
| Model | Input | Cached | Output |
|-------|-------|--------|--------|
| GLM-4.7 | $0.60 | $0.11 | $2.20 |
| GLM-4.7-FlashX | $0.07 | $0.01 | $0.40 |
| GLM-4.7-Flash | Free | Free | Free |

**Comparison to Claude:**
- Opus 4.5 is ~12.5x more expensive for input, ~16.7x for output
- Z.ai claims "200M tokens for $3" user experience

**Sustainability concern:** One Reddit user claimed z.ai "loses ~$7 per $1 revenue" - enjoy it while it lasts.

**Source:**
- [Z.AI Pricing Docs](https://docs.z.ai/guides/overview/pricing)
- [Z.AI FAQ](https://docs.z.ai/devpack/faq)
- Reddit thread: `.orch/glm-4-7-reddit.txt`

**Significance:** The value proposition is exceptional IF sustainable. The pricing is 1/7th Claude Max with 3x usage quota. However, burn rate concerns suggest this could change.

---

### Finding 3: Claude Code Works Well, OpenCode Has Issues

**Evidence:**

**Claude Code (works well):**
- Direct connection via `ANTHROPIC_BASE_URL=https://api.z.ai/api/anthropic`
- Users report 200M tokens/month for $3
- Model mapping: `ANTHROPIC_DEFAULT_OPUS_MODEL=glm-4.7`
- No gateway bottleneck

**OpenCode (has issues):**
- [Issue #6708](https://github.com/anomalyco/opencode/issues/6708): Tool calls placed inside thinking tags (forces new session)
- [Issue #8428](https://github.com/anomalyco/opencode/issues/8428): GLM-4.7 often gets stuck in thinking
- [Issue #8515](https://github.com/anomalyco/opencode/issues/8515): Z.ai API fails in OpenCode but works elsewhere
- [Issue #3139](https://github.com/sst/opencode/issues/3139): GLM model timing out

**Root cause hypothesis:** OpenCode uses more aggressive multi-threading, hitting gateway/concurrency limits. Claude Code connects direct.

**User quotes:**
- "OpenCode is hitting concurrency issues with GLM plan"
- "I never get this using Claude Code, so it's possible OpenCode is more aggressive with multi threading requests"
- "OpenCode is low quality cli compared to Claude code"

**Source:**
- GitHub issues linked above
- Reddit thread: `.orch/glm-4-7-reddit.txt`
- [Z.AI OpenCode Docs](https://docs.z.ai/scenario-example/develop-tools/opencode)

**Significance:** **Critical for orch-go.** Orch-go uses OpenCode as its primary interface. The known issues with OpenCode + GLM would directly impact reliability.

---

### Finding 4: Concurrency Limits Are Strict on GLM 4.7

**Evidence:**
- GLM 4.7: 1-5 concurrent connections (reports vary, sometimes as low as 1)
- GLM 4.6: 12+ concurrent connections with no issues
- Concurrency limits cause "connection throttling slowing down even before the LLM kicks in"

**User reports:**
- "On their coding plan there is currently a rate limit cap of 1 concurrent connection on 4.7 and 4.6"
- "I run 12 concurrently with GLM-4.6 no problem... So it's limits imposed specifically to GLM-4.7"
- "It had 3 concurrency limits until sometime yesterday"

**Implications:**
- Orch-go spawns multiple agents in parallel
- Current concurrency limits on GLM 4.7 would bottleneck multi-agent workflows
- GLM 4.6 has better concurrency but inferior coding capabilities

**Source:**
- Reddit thread: `.orch/glm-4-7-reddit.txt`

**Significance:** Multi-agent orchestration requires concurrent sessions. GLM 4.7's strict limits (1-5 vs 12+ on 4.6) make it unsuitable for orch-go's daemon mode which can spawn 5+ agents simultaneously.

---

### Finding 5: GLM 4.7 Flash Is Free and Local-Capable

**Evidence:**
- Released January 19, 2026
- 30B parameters, 3B active per token (MoE)
- Runs on RTX 3090 / Apple Silicon
- Speed: 82 tokens/sec on M4 Max, 4000+ tps on server hardware
- SWE-bench Verified: 59.2% (beats Qwen3-Coder 480B at 55.4%)
- **API pricing: FREE**

**Source:**
- [TechLoy Article](https://www.techloy.com/zhipu-ai-launches-glm-4-7-flash-a-local-ai-coding-model-for-consumer-hardware/)
- [Hugging Face](https://huggingface.co/zai-org/GLM-4.7-Flash)
- [Z.AI Pricing](https://docs.z.ai/guides/overview/pricing)

**Significance:** GLM 4.7 Flash could be a "haiku-tier" alternative with free API access. However, Reddit reports the "Air" model (their haiku equivalent) "hallucinates responses like mad." Need to distinguish between Flash and Air.

---

### Finding 6: Current Model Stack Comparison

**Current orch-go stack (from decision 2026-01-18):**
- Primary: Claude Max via CLI ($200/mo flat, unlimited Sonnet/Opus)
- Fallback: OpenCode API + Sonnet (pay-per-token, opt-in only)

> **Note:** The Jan 9 decision (Gemini Flash primary) was superseded on Jan 18 due to API costs spiraling to $70-80/day without visibility. See `.kb/models/current-model-stack.md` for authoritative current state.

**GLM 4.7 comparison:**
| Aspect | Claude Max (CLI) | GLM 4.7 | OpenCode API |
|--------|------------------|---------|--------------|
| Price | $200/mo flat | $3-60/mo | Pay-per-token |
| Reliability | High | Medium (OpenCode issues) | High |
| Coding quality | Excellent (Opus) | Very good | Good (Sonnet) |
| Concurrency | Limited (tmux) | Poor (1-5) | High (5+) |
| Dashboard | No visibility | Problematic | Full visibility |

**Source:**
- `.kb/models/current-model-stack.md` (authoritative current state)
- `.kb/decisions/2026-01-18-max-subscription-primary-spawn-path.md` (current policy)
- `.kb/decisions/2026-01-09-abandon-claude-max-oauth-use-gemini-primary.md` (superseded)

**Significance:** Current Claude Max stack is working and cost-predictable. GLM 4.7's OpenCode issues and concurrency limits conflict with orch-go architecture regardless of which backend is primary.

---

## Synthesis

**Key Insights:**

1. **GLM 4.7 is legitimately good** - Benchmark scores competitive with Opus on coding tasks, particularly for iterative implementation. The "Preserved Thinking" feature specifically helps with multi-turn context retention, which is valuable for agents.

2. **Integration path matters** - Claude Code (direct connection) works well; OpenCode (orch-go's interface) has documented issues. This is a blocker for adoption.

3. **Concurrency is a deal-breaker for orchestration** - GLM 4.7's 1-5 concurrent connection limit vs GLM 4.6's 12+ means it's unsuited for multi-agent spawning. Orch-go's daemon mode regularly spawns 5+ parallel agents.

4. **Value proposition is exceptional but risky** - At $3-60/month for frontier-tier coding, the pricing is unsustainable long-term (claimed $7 loss per $1 revenue). Any strategy depending on this pricing should have contingency.

**Answer to Investigation Question:**

**No, orch-go should NOT adopt z.ai GLM 4.7 currently.** The OpenCode integration issues and strict concurrency limits directly conflict with orch-go's architecture (OpenCode-based, multi-agent).

However, GLM 4.7 could be considered for:
1. **Claude CLI escape hatch** (`--mode claude`) - Direct connection avoids OpenCode issues
2. **Future re-evaluation** - If OpenCode-GLM issues are resolved
3. **GLM 4.7 Flash** - Free tier for haiku-tier tasks, pending quality validation

---

## Structured Uncertainty

**What's tested:**

- ✅ GLM 4.7 benchmarks are publicly documented (LiveCodeBench, SWE-bench scores verified via multiple sources)
- ✅ Pricing structure confirmed via official z.ai docs
- ✅ OpenCode issues documented via GitHub issues and Reddit reports
- ✅ Claude Code integration works (multiple user confirmations)

**What's untested:**

- ⚠️ Actual GLM 4.7 quality in orch-go context (not directly tested, relying on reports)
- ⚠️ GLM 4.7 Flash vs "Air" distinction for haiku-tier quality (not validated)
- ⚠️ OpenCode issue resolution timeline (unknown)
- ⚠️ z.ai sustainability (burn rate claim is secondhand)

**What would change this:**

- If OpenCode ships stable GLM integration (monitor GitHub issues)
- If z.ai increases concurrency limits on GLM 4.7
- If Gemini Flash quality degrades significantly
- If orch-go switches primary interface from OpenCode to Claude CLI

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**WATCH status** - Do not adopt GLM 4.7 for orch-go currently. Revisit in 30-60 days.

**Why this approach:**
- Current Gemini Flash stack is stable and working (per Jan 9 decision)
- OpenCode-GLM issues are unresolved
- Concurrency limits conflict with multi-agent architecture
- No urgent need to change what's working

**Trade-offs accepted:**
- Missing potential cost savings (GLM is cheaper than Gemini API)
- Not leveraging GLM 4.7's strong coding capabilities
- Acceptable given reliability concerns outweigh cost savings

**Implementation sequence:**
1. Document this investigation for future reference
2. Monitor OpenCode-GLM GitHub issues monthly
3. Re-evaluate if issues are resolved or if Gemini Flash degrades

### Alternative Approaches Considered

**Option B: Add GLM 4.7 for Claude CLI escape hatch only**
- **Pros:** Direct connection avoids OpenCode issues; useful for `--mode claude` spawns
- **Cons:** Adds complexity; need to manage z.ai account; limited use case
- **When to use instead:** If Claude Max OAuth remains blocked AND need frontier-tier for Claude CLI mode

**Option C: Adopt GLM 4.6 instead (better concurrency)**
- **Pros:** 12+ concurrent connections; same API/integration
- **Cons:** Inferior coding capabilities to 4.7; still has OpenCode issues
- **When to use instead:** If concurrency is paramount and coding quality can be sacrificed

**Option D: Adopt GLM 4.7 Flash as haiku-tier**
- **Pros:** Free; fast; potentially useful for quick tasks
- **Cons:** Quality untested; "Air" model (similar tier) reportedly hallucinates
- **When to use instead:** After validating quality on non-critical tasks

**Rationale for recommendation:** OpenCode is orch-go's primary interface. Until OpenCode-GLM issues are resolved, any GLM adoption creates reliability risk that outweighs the cost/capability benefits. Current stack works.

---

### Implementation Details

**What to implement first:**
- Nothing for now - maintain current Gemini Flash + Sonnet API stack

**Things to watch out for:**
- ⚠️ z.ai pricing changes (unsustainable burn rate reported)
- ⚠️ OpenCode GLM issue resolution (GitHub #6708, #8428, #8515)
- ⚠️ GLM 4.8 or future versions may improve concurrency

**Areas needing further investigation:**
- GLM 4.7 Flash quality validation (is it actually haiku-tier?)
- z.ai reliability during US peak hours (reported slowdowns)
- Self-hosted GLM 4.7 as alternative to z.ai API (MIT licensed)

**Success criteria:**
- ✅ This investigation documents current state clearly
- ✅ Decision to wait is justified with evidence
- ✅ Re-evaluation triggers are defined (30-60 days, or OpenCode issues resolved)

---

## References

**Files Examined:**
- `.orch/glm-4-7-reddit.txt` - Reddit discussion with user experiences
- `.kb/models/current-model-stack.md` - Authoritative current model stack reference
- `.kb/decisions/2026-01-18-max-subscription-primary-spawn-path.md` - Current model stack policy
- `.kb/decisions/2026-01-09-abandon-claude-max-oauth-use-gemini-primary.md` - Historical context (superseded)

**External Documentation:**
- [Z.AI Pricing](https://docs.z.ai/guides/overview/pricing) - Official pricing structure
- [Z.AI GLM-4.7 Docs](https://docs.z.ai/guides/llm/glm-4.7) - Model capabilities
- [Z.AI FAQ](https://docs.z.ai/devpack/faq) - Plan details and quotas
- [Z.AI Claude Code Setup](https://docs.z.ai/scenario-example/develop-tools/claude) - Integration guide
- [OpenCode Issue #6708](https://github.com/anomalyco/opencode/issues/6708) - Tool calls in thinking tags
- [OpenCode Issue #8428](https://github.com/anomalyco/opencode/issues/8428) - Stuck in thinking
- [GLM-4.7 Blog](https://z.ai/blog/glm-4.7) - Official release post
- [Benchmarks Comparison](https://llm-stats.com/models/compare/claude-opus-4-5-20251101-vs-glm-4.7) - Opus vs GLM comparison
- [Hugging Face GLM-4.7](https://huggingface.co/zai-org/GLM-4.7) - Open source weights
- [TechLoy Flash Article](https://www.techloy.com/zhipu-ai-launches-glm-4-7-flash-a-local-ai-coding-model-for-consumer-hardware/) - Flash variant details

**Related Artifacts:**
- **Model:** `.kb/models/current-model-stack.md` - Authoritative current model stack reference
- **Decision:** `.kb/decisions/2026-01-18-max-subscription-primary-spawn-path.md` - Current model stack policy
- **Decision (superseded):** `.kb/decisions/2026-01-09-abandon-claude-max-oauth-use-gemini-primary.md` - Historical context (Gemini Flash primary, superseded)

---

## Investigation History

**2026-01-22 ~09:00:** Investigation started
- Initial question: Should orch-go adopt z.ai's GLM 4.7?
- Context: Reddit discussion surfaced claims about GLM 4.7 value/capabilities

**2026-01-22 ~09:30:** Research completed
- Gathered pricing, capabilities, integration details from official docs
- Identified OpenCode-specific issues via GitHub and Reddit
- Compared to current model stack per Jan 9 decision

**2026-01-22 ~10:00:** Investigation completed
- Status: Complete
- Key outcome: WATCH recommendation - OpenCode issues and concurrency limits make GLM 4.7 unsuitable for orch-go currently; revisit in 30-60 days
