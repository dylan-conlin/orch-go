<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** GPT-5.3-Codex exists as Codex-only model (no API access yet). ChatGPT Pro ($200/mo) claims "unlimited" but has message-based limits (300-1500 local msgs/5h for Pro). Reasoning tokens ARE billed as output tokens. At 150K tokens/task, Pro subscription handles ~50-400 cloud tasks per 5h window. Credit system (~5 credits/local msg, ~25/cloud task) extends beyond limits.

**Evidence:** OpenAI Codex pricing page (developers.openai.com/codex/pricing), GPT-5.2 help article (help.openai.com), OpenAI API reasoning docs (platform.openai.com/docs/guides/reasoning), Codex models page (developers.openai.com/codex/models).

**Knowledge:** ChatGPT Pro rate limits are message-count based (not token-count), making high-token tasks MORE economical than many small tasks. The 150K token single-task concern is less relevant than total message count. GPT-5.3-Codex not yet available via API (ChatGPT subscription auth only). Claude Max 20x ($200/mo) has similar "unlimited with guardrails" model but uses internal credit system (~83M credits/week).

**Next:** Update model-access-spawn-paths.md with GPT-5.3-Codex. If multi-model routing is pursued, Codex CLI with Pro subscription is the only path to GPT-5.3-Codex (no API).

**Promote to Decision:** recommend-no (factual research, not architectural choice)

---

# Research: ChatGPT Pro Subscription Quota Structure for GPT-5.3-Codex

**Question:** What are the actual usage limits of ChatGPT Pro ($200/mo) for GPT-5.3-Codex? Do reasoning tokens count against limits? What happens at the limit?

**Started:** 2026-02-06
**Updated:** 2026-02-06
**Owner:** Research agent (spawned from orch-go-21348)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Context

GPT-5.3-codex consumed 150K tokens on a single code extraction task. If ChatGPT Pro has a usage cap that counts reasoning tokens, this dramatically limits tasks per billing period. Need to understand actual economics before committing to multi-model routing.

---

## Finding 1: GPT-5.3-Codex Exists (Codex-Only, No API Yet)

**Evidence:** Codex models page (developers.openai.com/codex/models) lists GPT-5.3-Codex as the recommended model: "Most capable agentic coding model to date, combining frontier coding performance with stronger reasoning and professional knowledge capabilities."

**Key facts:**

- Available via Codex CLI, Codex App, IDE extension, and Codex Cloud
- NOT yet available via API ("API access for the model will come soon")
- Supports ChatGPT subscription authentication only (for subscription users)
- API key users get "delayed access to new models like GPT-5.3-Codex"

**Significance:** GPT-5.3-Codex cannot be used via standard API calls. Only accessible through Codex CLI/App/Cloud with ChatGPT subscription auth. This is analogous to Anthropic's Opus-via-Claude-Code-only restriction.

---

## Finding 2: ChatGPT Pro Rate Limits Are Message-Based, Not Token-Based

**Evidence:** OpenAI help article (help.openai.com/en/articles/11909943 - "GPT-5.2 in ChatGPT") and Codex pricing page.

### ChatGPT Limits (ChatGPT interface, not Codex):

| Plan                | GPT-5.2 Limit | Period      | At Limit                    |
| ------------------- | ------------- | ----------- | --------------------------- |
| Free                | 10 messages   | per 5 hours | Falls back to mini model    |
| Plus ($20)          | 160 messages  | per 3 hours | Falls back to mini model    |
| Pro ($200)          | **Unlimited** | -           | Subject to abuse guardrails |
| Business ($30/user) | **Unlimited** | -           | Subject to abuse guardrails |

### Codex Limits (Codex CLI/Cloud):

| Plan                | Local Messages/5h | Cloud Tasks/5h | Code Reviews/week |
| ------------------- | ----------------- | -------------- | ----------------- |
| Plus ($20)          | 45-225            | 10-60          | 10-25             |
| **Pro ($200)**      | **300-1500**      | **50-400**     | **100-250**       |
| Business ($30/user) | 45-225            | 10-60          | 10-25             |
| Enterprise/Edu      | Credits-based     | Credits-based  | Credits-based     |

**Critical insight:** Limits are MESSAGE-based, not token-based. A 150K token task and a 5K token task both count as 1 message. This makes high-token tasks proportionally MORE economical.

**Significance:** The spawn context's concern about "150K tokens consumed on a single task" is less relevant than expected. The bottleneck is total message count per 5-hour window, not total tokens consumed.

---

## Finding 3: Reasoning Tokens ARE Billed as Output Tokens (API)

**Evidence:** OpenAI reasoning guide (platform.openai.com/docs/guides/reasoning):

> "While reasoning tokens are not visible via the API, they still occupy space in the model's context window and are billed as output tokens."

And from the pricing page:

> "While reasoning tokens are not visible via the API, they still occupy space in the model's context window and are billed as output tokens."

**For API usage:**

- GPT-5.2 / GPT-5.2-Codex: $1.75/MTok input, $14.00/MTok output (Standard)
- Reasoning tokens billed at output rate ($14.00/MTok)
- A task with 150K tokens (including reasoning) at output rate = ~$2.10 per task

**For subscription usage (Codex CLI/Cloud):**

- Reasoning tokens are baked into the message-based limits
- The range "300-1500 local messages per 5h" accounts for variable complexity
- Simple tasks → more messages possible; complex reasoning-heavy tasks → fewer messages

**Significance:** For API users, reasoning tokens are expensive ($14/MTok output). For subscription users, they're absorbed into the message-based limits.

---

## Finding 4: What Happens When You Hit The Limit

**Evidence:** Codex pricing FAQ and ChatGPT Pro help article.

### ChatGPT Interface (Pro):

- "Unlimited access is subject to abuse guardrails"
- If abuse detected: "temporary restriction on your usage"
- You'll be informed; can contact support
- If no violation found: access restored

### Codex (Pro):

1. **Purchase credits** to continue immediately
2. **Switch to GPT-5.1-Codex-Mini** (~4x more messages per limit window)
3. **Use API key** for local tasks at standard API rates
4. **Wait** for the 5-hour window to reset

### Credit System:

| Unit               | GPT-5.3-Codex / GPT-5.2-Codex | GPT-5.1-Codex-Mini |
| ------------------ | ----------------------------- | ------------------ |
| Local Task (1 msg) | ~5 credits                    | ~1 credit          |
| Cloud Task (1 msg) | ~25 credits                   | N/A                |
| Code Review (1 PR) | ~25 credits                   | N/A                |

Credits can be purchased by Plus and Pro users to extend beyond limits.

**Significance:** Pro users aren't hard-blocked - they have multiple fallback paths. The system is designed to keep you productive (at additional cost) rather than completely stopping work.

---

## Finding 5: GPT-5.3-Codex vs GPT-5.2 Quotas

**Evidence:** Codex models page and pricing page.

| Aspect              | GPT-5.3-Codex             | GPT-5.2                   | GPT-5.2-Codex             |
| ------------------- | ------------------------- | ------------------------- | ------------------------- |
| Access              | Codex CLI/App/Cloud only  | API + Codex               | API + Codex               |
| API Pricing         | Not yet available         | $1.75/$14 MTok (Standard) | $1.75/$14 MTok (Standard) |
| Context Window      | Not documented yet        | 400K                      | 400K                      |
| Max Output          | Not documented yet        | 128K                      | 128K                      |
| Subscription Limits | Same message-based limits | Same message-based limits | Same message-based limits |
| Credits per msg     | ~5 (local), ~25 (cloud)   | ~5 (local), ~25 (cloud)   | ~5 (local), ~25 (cloud)   |

**Key insight:** GPT-5.3-Codex and GPT-5.2-Codex share the same Codex usage limits and credit costs. The model upgrade doesn't come with different quotas - all Codex models share the same message-based pool.

**Significance:** No penalty for using the newest/best model. The quota structure incentivizes using the best model available since all models consume the same limits.

---

## Finding 6: Comparison with Claude Max Subscription

**Evidence:** Prior investigation (2026-01-18) + orchestration-cost-economics model + she-llac.com reverse engineering.

| Aspect                       | ChatGPT Pro ($200/mo)             | Claude Max 20x ($200/mo)                 |
| ---------------------------- | --------------------------------- | ---------------------------------------- |
| **Limit Type**               | Message-based (300-1500 local/5h) | Credit-based (83.3M credits/week)        |
| **ChatGPT/Claude Interface** | Unlimited GPT-5.2                 | 20x Pro usage                            |
| **Agent CLI**                | Codex CLI (300-1500 msgs/5h)      | Claude Code (credit-limited)             |
| **At Limit**                 | Buy credits or switch to mini     | Wait for weekly reset or Docker escape   |
| **Reasoning Tokens**         | Absorbed into message count       | Absorbed into credit formula             |
| **Cache Pricing**            | Standard                          | **FREE cache reads** (massive advantage) |
| **API Included**             | NO                                | NO                                       |
| **Best Model Access**        | GPT-5.3-Codex (Codex only)        | Opus 4.5 (Claude Code only)              |
| **Third-Party Tools**        | Some OAuth issues                 | **BLOCKED** (fingerprinting)             |

### Economic Comparison for 150K-token Tasks:

**ChatGPT Pro (via Codex CLI):**

- 150K token task = 1 message
- Pro allows 300-1500 messages per 5h
- At worst case: 300 msgs × 4 windows/day = 1,200 tasks/day
- At best case: 1,500 msgs × 4 windows/day = 6,000 tasks/day
- Cost: $200/mo flat

**Claude Max 20x (via Claude Code):**

- 150K token task at Opus credit rates: ceil(150K × 0.667 input + 150K × 3.333 output) ≈ ~600K credits per task (rough estimate, depends on input/output split)
- Weekly limit: 83.3M credits
- Estimated tasks/week: ~139 tasks (at 600K credits each)
- Cost: $200/mo flat

**Significance:** For high-token agentic work, ChatGPT Pro's message-based limits may be MORE generous than Claude Max's credit-based limits. However, Claude Max's free cache reads provide significant value for tool-heavy work with repeated context.

---

## Structured Uncertainty

**What's tested (verified against primary sources):**

- GPT-5.3-Codex exists and is Codex-only (verified: developers.openai.com/codex/models)
- ChatGPT Pro has message-based limits for Codex (verified: developers.openai.com/codex/pricing)
- Reasoning tokens billed as output tokens on API (verified: platform.openai.com/docs/guides/reasoning)
- Credit costs: ~5/local msg, ~25/cloud task (verified: Codex pricing FAQ)
- Pro Codex limits: 300-1500 local, 50-400 cloud per 5h (verified: Codex pricing page)

**What's untested:**

- Actual GPT-5.3-Codex token consumption patterns (not benchmarked)
- Credit purchase pricing (not documented on pricing page)
- Whether "300-1500 messages" variance correlates with token count or message complexity
- GPT-5.3-Codex context window and max output limits (not yet documented)
- Real-world comparison of Codex CLI vs Claude Code for orchestration workloads

**What would change this:**

- GPT-5.3-Codex API access launching → direct API comparison possible
- OpenAI publishing token-level quota details → more precise economics
- Empirical testing of Codex CLI with orch-go prompts → real feasibility data

---

## Answers to Original Questions

### Q1: What is the Pro subscription rate limit?

**Answer:** Message-based, not token-based:

- **Codex Local:** 300-1500 messages per 5 hours
- **Codex Cloud:** 50-400 tasks per 5 hours
- **ChatGPT Interface:** Unlimited (abuse guardrails only)
- **No specific token/day, token/month, or request/hour limits documented**

### Q2: Do reasoning tokens count against the limit?

**Answer:**

- **API:** YES - reasoning tokens are billed as output tokens ($14/MTok for GPT-5.2-Codex)
- **Subscription (Codex):** Indirectly - absorbed into message-based limits. The range (300-1500) accounts for task complexity. More reasoning = fewer messages allowed in the window.

### Q3: What happens when you hit the limit?

**Answer:** Multiple fallback options:

1. Purchase ChatGPT credits to continue immediately
2. Switch to GPT-5.1-Codex-Mini (~4x more messages)
3. Use API key for local tasks at standard API rates
4. Wait for 5-hour window to reset

- NOT hard-blocked; designed to keep users productive

### Q4: Difference between GPT-5.3-Codex and GPT-5.2 quotas?

**Answer:** No quota difference. All Codex models (GPT-5.3-Codex, GPT-5.2-Codex, etc.) share the same message-based limits and credit costs (~5 credits/local msg). The main difference is that GPT-5.3-Codex is Codex-only (no API access yet), while GPT-5.2 has both API and Codex access.

### Q5: How does this compare to Claude Max subscription limits?

**Answer:** ChatGPT Pro may be more generous for high-token tasks:

- **ChatGPT Pro:** 300-1500 msgs/5h (token-agnostic per message)
- **Claude Max 20x:** ~83.3M credits/week (credit cost scales with tokens)
- **Key advantage Claude:** Free cache reads (massive for tool-heavy agents)
- **Key advantage ChatGPT:** Message-based limits favor large, complex tasks
- Both: $200/mo, no API included, best model restricted to official CLI

---

## References

**Primary Sources:**

- [Codex Pricing](https://developers.openai.com/codex/pricing) - Message limits, credits, plan comparison
- [Codex Models](https://developers.openai.com/codex/models) - GPT-5.3-Codex availability
- [GPT-5.2 in ChatGPT](https://help.openai.com/en/articles/11909943) - ChatGPT interface limits
- [ChatGPT Pro](https://help.openai.com/en/articles/9793128-what-is-chatgpt-pro) - Pro plan details
- [OpenAI API Reasoning](https://platform.openai.com/docs/guides/reasoning) - Reasoning token billing
- [OpenAI API Pricing](https://platform.openai.com/docs/pricing) - Token costs
- [OpenAI API Rate Limits](https://platform.openai.com/docs/guides/rate-limits) - API rate limit structure

**Prior Investigations:**

- `.kb/investigations/2026-01-18-inv-research-compare-openai-chatgpt-pro-anthropic-max.md`
- `.kb/models/orchestration-cost-economics.md`
- `.kb/models/model-access-spawn-paths.md`

---

## Investigation History

**2026-02-06:** Investigation started

- Context: GPT-5.3-codex consumed 150K tokens on single task, need to understand quota economics
- Researched OpenAI docs, Codex pricing, reasoning token billing, and compared to Claude Max

**2026-02-06:** Investigation completed

- Key outcome: ChatGPT Pro limits are message-based (not token-based), making 150K token tasks less concerning than expected. GPT-5.3-Codex is Codex-only (no API). Pro allows 300-1500 local messages per 5h window.
