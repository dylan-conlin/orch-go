<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** ChatGPT Pro ($200/mo) does NOT include API access, but includes Codex CLI which authenticates via OAuth and works similarly to Claude Code with Max subscription for agent orchestration.

**Evidence:** OpenAI docs confirm API is billed separately. Codex CLI supports ChatGPT subscription OAuth authentication. Both platforms have $200/mo unlimited tiers with CLI-based coding agents.

**Knowledge:** For agent orchestration, both OpenAI (Codex CLI) and Anthropic (Claude Code) offer subscription-authenticated terminal agents. API access requires separate billing on both platforms. Anthropic recently blocked third-party tools from Max subscription (Jan 2026).

**Next:** If diversifying AI backends, Codex CLI with ChatGPT Pro is viable alternative to Claude Code with Max. Consider testing Codex CLI for parallel orchestration capability.

**Promote to Decision:** recommend-no (factual comparison, not architectural choice)

---

# Research: OpenAI ChatGPT Pro vs Anthropic Max Subscription

**Question:** Does ChatGPT Pro ($200/mo) provide API access? What models are available? Can it be used for agent orchestration like Claude Code?

**Started:** 2026-01-18
**Updated:** 2026-01-18
**Owner:** Orchestrator research agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Options Evaluated

### Option 1: OpenAI ChatGPT Pro ($200/month)

**Overview:** Premium tier providing unlimited access to OpenAI's most advanced models through ChatGPT interface and Codex CLI agent.

**Included Features:**
- Unlimited GPT-5.2 messages (Pro mode, all variants: Auto, Instant, Thinking)
- o3-pro access (most powerful reasoning model, replaces o1-pro)
- o3, o4-mini, o4-mini-high models
- Codex CLI (terminal-based coding agent)
- Codex Cloud (parallel task execution)
- Advanced Voice, Sora 2 Pro
- Priority access during peak times
- Never hit usage limits

**Models Available:**
| Model | Description | Pro Access |
|-------|-------------|------------|
| GPT-5.2 | Flagship model with 3 modes (Auto, Instant, Thinking) | Unlimited |
| o3 | Most powerful reasoning model (coding, math, science) | Unlimited |
| o3-pro | Extended reasoning for hardest problems | Exclusive |
| o4-mini | Fast, cost-efficient reasoning (best on AIME 2024/2025) | High limits |
| GPT-4.5 | Released Feb 2025, ~$75/$150 per M tokens via API | Included |
| Legacy | GPT-4o, GPT-4.1, o1, o1-mini | Included |

**API Access: NO**
- ChatGPT and API are completely separate platforms
- API billed per-token independently
- Pro subscription does NOT include API credits
- Must pay both if using both (duplicative spend)

**Pros:**
- Unlimited access to top models
- Codex CLI included (terminal agent)
- o3-pro exclusive (highest reasoning capability)
- GPT-5.2 Codex optimized for software engineering
- Cloud-based parallel task execution
- 200-400% productivity gains reported

**Cons:**
- No API access included
- Codex CLI OAuth has some bugs (may generate API key unexpectedly)
- Windows support experimental (WSL recommended)
- Some MCP OAuth issues reported

**Evidence:**
- Official: https://openai.com/index/introducing-chatgpt-pro/
- Codex CLI: https://developers.openai.com/codex/cli/
- Pricing: https://chatgpt.com/pricing/
- Help: https://help.openai.com/en/articles/9793128-what-is-chatgpt-pro

---

### Option 2: Anthropic Max ($100/$200/month)

**Overview:** Premium tier providing 5x-20x more usage than Pro for Claude and Claude Code.

**Included Features:**
- $100/mo tier: 5x Pro usage
- $200/mo tier: 20x Pro usage
- Claude Code CLI (terminal-based coding agent)
- Priority access during peak times
- Early access to new models
- Usage shared between Claude web and Claude Code

**Models Available:**
| Model | Description | Max Access |
|-------|-------------|------------|
| Claude Opus 4.5 | Most capable model | Included |
| Claude Sonnet 4 | Balanced performance/cost | Included |
| Claude Haiku | Fast, efficient | Included |

**API Access: NO**
- API billed separately per-token
- ANTHROPIC_API_KEY overrides subscription auth
- 1M token context via API vs 200K via subscription

**Pros:**
- Claude Code deeply integrated with terminal/IDE
- 200K token context window (subscription)
- Strong multi-file refactoring
- Agentic search understands full codebase
- OAuth authentication works

**Cons:**
- Third-party tool access blocked (Jan 9, 2026)
- OpenCode, Cursor, other IDEs blocked from using Max subscription
- Must use official Claude Code CLI
- Usage shared across Claude web + Claude Code

**Evidence:**
- Official: https://support.claude.com/en/articles/11145838-using-claude-code-with-your-pro-or-max-plan
- Pricing: https://www.anthropic.com/pricing/
- Third-party block: https://news.ycombinator.com/item?id=46549823

---

## Comparison Table

| Aspect | ChatGPT Pro | Anthropic Max |
|--------|-------------|---------------|
| **Price** | $200/mo | $100 or $200/mo |
| **API Access** | NO (separate billing) | NO (separate billing) |
| **Terminal Agent** | Codex CLI | Claude Code |
| **Authentication** | ChatGPT OAuth | Anthropic OAuth |
| **Models** | GPT-5.2, o3, o3-pro, o4-mini | Opus 4.5, Sonnet 4, Haiku |
| **Context Window** | ~200K (Codex) | 200K (Code) |
| **Reasoning Mode** | o3-pro (extended compute) | Extended thinking |
| **Third-party Tools** | Some OAuth issues | BLOCKED (Jan 2026) |
| **Parallel Tasks** | Codex Cloud | Not built-in |
| **Best For** | Prototyping, fast iteration | Multi-file refactoring |

---

## Recommendation

**For agent orchestration like orch-go:**

Claude Code with Max subscription remains the better choice for Dylan's current workflow because:

1. **OAuth works reliably** - Claude Code's subscription auth is stable
2. **Existing integration** - orch-go already built around Claude/OpenCode
3. **Context awareness** - Claude Code's agentic search understands codebases well

**However, Codex CLI is a viable alternative if:**
- Need to diversify AI backends (reduce single-vendor risk)
- Want o3-pro reasoning for hard problems
- Need parallel cloud task execution
- Anthropic further restricts Max subscription

**Key insight for orchestration:** Both platforms now have subscription-authenticated CLI agents (Codex CLI, Claude Code). Neither provides API access with subscription. The $200/mo tier buys unlimited agent usage, not API access.

**Trade-offs accepted:**
- Staying with Anthropic despite third-party tool restrictions
- Not exploring OpenAI's Codex Cloud parallel execution yet

**When this recommendation might change:**
- If Anthropic blocks more tools from Max subscription
- If OpenAI fixes Codex CLI OAuth bugs
- If needing o3-pro reasoning capability specifically

---

## Structured Uncertainty

**What's tested:**
- ✅ ChatGPT Pro costs $200/mo (verified: official pricing page)
- ✅ API is billed separately on both platforms (verified: official docs)
- ✅ Codex CLI supports ChatGPT subscription auth (verified: developer docs)
- ✅ Anthropic blocked third-party tools Jan 2026 (verified: HN, GitHub issues)

**What's untested:**
- ⚠️ Codex CLI OAuth reliability in practice (reports of bugs, not tested)
- ⚠️ Codex CLI performance vs Claude Code for orchestration (not benchmarked)
- ⚠️ o3-pro actual improvement over Claude Opus 4.5 (not compared)
- ⚠️ Codex Cloud parallel task execution for agent spawning (not tested)

**What would change this:**
- Codex CLI OAuth working reliably → viable backup orchestration
- Anthropic blocking Claude Code itself → forced migration to Codex
- OpenAI releasing agents API with subscription → direct orchestration

---

## Implementation Recommendations

### Recommended Approach ⭐

**Monitor but don't migrate** - Continue with Claude Code for orchestration, keep Codex CLI as backup option.

**Why this approach:**
- Existing orch-go infrastructure works
- Migration cost not justified by current needs
- Both platforms have similar subscription-agent model

**Trade-offs accepted:**
- Single-vendor dependency on Anthropic
- Not leveraging o3-pro reasoning

**Implementation sequence:**
1. Document Codex CLI as escape hatch option (this investigation)
2. Monitor OpenAI OAuth fixes in Codex CLI releases
3. Consider Codex CLI if Anthropic restricts Claude Code further

### Alternative Approaches Considered

**Option B: Migrate to Codex CLI**
- **Pros:** Access to o3-pro, parallel cloud execution
- **Cons:** OAuth bugs, rebuild orch-go integration, experimental
- **When to use:** If Anthropic blocks Claude Code CLI itself

**Option C: Dual-vendor orchestration**
- **Pros:** Resilience, access to both model families
- **Cons:** Double maintenance, double learning curve
- **When to use:** For high-value production systems needing redundancy

---

## References

**External Documentation:**
- [OpenAI ChatGPT Pro](https://openai.com/index/introducing-chatgpt-pro/) - Official announcement
- [Codex CLI](https://developers.openai.com/codex/cli/) - Developer documentation
- [Codex Authentication](https://developers.openai.com/codex/auth/) - OAuth and API key options
- [ChatGPT Pricing](https://chatgpt.com/pricing/) - Plan comparison
- [OpenAI API Pricing](https://platform.openai.com/docs/pricing) - Per-token costs
- [Claude Code with Max](https://support.claude.com/en/articles/11145838-using-claude-code-with-your-pro-or-max-plan) - Anthropic official
- [Anthropic Pricing](https://www.anthropic.com/pricing/) - Max subscription tiers
- [Codex vs Claude Code](https://www.builder.io/blog/codex-vs-claude-code) - Feature comparison

**GitHub Issues (OAuth problems):**
- [Codex CLI API key issues](https://github.com/openai/codex/issues/2000)
- [MCP OAuth issues](https://github.com/openai/codex/issues/6154)
- [Switching auth methods](https://github.com/openai/codex/issues/3286)

---

## Investigation History

**2026-01-18 13:00:** Investigation started
- Question: Does ChatGPT Pro include API access? Models? Agent orchestration capability?
- Context: Spawned by orchestrator to compare AI subscription options

**2026-01-18 13:30:** Research completed
- Evaluated ChatGPT Pro and Anthropic Max
- Found both have subscription-authenticated CLI agents
- Neither includes API access with subscription
- Documented OAuth issues with Codex CLI
- Recommendation: Monitor Codex CLI as backup option

**2026-01-18 13:30:** Investigation completed
- Status: Complete
- Key outcome: ChatGPT Pro viable for orchestration via Codex CLI, but OAuth has bugs; Claude Code more reliable currently
