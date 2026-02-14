# Model: Orchestration Cost Economics

**Domain:** Agent Orchestration / Model Selection / Cost Management
**Last Updated:** 2026-01-28
**Synthesized From:** 15 investigations, 25+ kb quick entries, and decisions spanning Nov 2025 - Jan 2026

---

## Summary (30 seconds)

Agent orchestration cost is driven by three factors: **model pricing** (10-100x variance), **access restrictions** (fingerprinting, OAuth blocking), and **visibility** (lack of tracking caused $402 surprise spend). The Jan 2026 cost crisis revealed that headless spawning without cost visibility leads to runaway spend. DeepSeek V3 at $0.25/$0.38/MTok is now a **viable primary option** after testing confirmed stable function calling (contradicting earlier "unstable" documentation).

---

## The Cost Problem Timeline

| Date | Event | Impact |
|------|-------|--------|
| Dec 2025 | Gemini Flash (free) as default | $0/mo, hit 2,000 req/min TPM limit |
| Jan 9, 2026 | Switch to Sonnet API (pay-per-token) | Unknown cost, no tracking |
| Jan 9, 2026 | Anthropic blocks OAuth for third-party tools | Can't use Max subscription via OpenCode |
| Jan 12, 2026 | Investigation identifies cost tracking need | Never implemented |
| Jan 13, 2026 | Cancel second Max subscription | $2,400/year savings |
| Jan 18, 2026 | Discover $402 spent in ~2 weeks | $70-80/day burn rate |
| Jan 18, 2026 | Switch back to Max subscription default | $200/mo flat |
| Jan 19, 2026 | Test confirms DeepSeek V3 function calling works | New viable option |

---

## Model Pricing Comparison

| Model | Input $/MTok | Output $/MTok | Access Method | Notes |
|-------|--------------|---------------|---------------|-------|
| **DeepSeek V3** | $0.25 | $0.38 | API | **10-65x cheaper, function calling works** |
| DeepSeek R1 | $0.45 | $2.15 | API | Reasoning model, function calling experimental |
| Gemini Flash | ~$0.10-0.30 | Variable | API/Free tier | 2,000 req/min limit blocks tool-heavy agents |
| Claude Haiku | $1.00 | $5.00 | API | Fast, lightweight |
| Claude Sonnet | $3.00 | $15.00 | API | Doubles at >200K context |
| Claude Opus | $5.00 | $25.00 | API or Max+CLI | Highest quality |
| **Claude Max** | $200/mo flat | - | Claude CLI only | Unlimited Sonnet + Opus |

### Cost Equivalence Points

At **$200/mo Max subscription breakeven:**
- DeepSeek V3: ~317M input tokens OR ~526M output tokens
- Sonnet: ~67M input tokens OR ~13M output tokens
- Opus: ~40M input tokens OR ~8M output tokens

**Key insight:** At Jan 18 burn rate ($70-80/day with Sonnet), Max subscription is 10x cheaper. But DeepSeek V3 could be even cheaper if function calling holds up.

---

## Internal Credit System (Reverse Engineered Jan 2026)

**Source:** she-llac.com/claude-limits - obtained via reverse engineering SSE responses and Stern-Brocot tree algorithm on float precision artifacts. Data may become stale if Anthropic patches the leak.

### Credit Formula

Anthropic uses an internal credit system to meter subscription usage:

```
credits_used = ceil(input_tokens × input_rate + output_tokens × output_rate)
```

**Per-model credit rates:**

| Model | Input Rate | Output Rate |
|-------|------------|-------------|
| Haiku | 2/15 (0.133) | 10/15 (0.667) |
| Sonnet | 6/15 (0.4) | 30/15 (2.0) |
| Opus | 10/15 (0.667) | 50/15 (3.333) |

### Actual Limits vs Marketing Claims

| Plan | Marketing | 5-Hour Session | Weekly Limit | Actual Ratio |
|------|-----------|----------------|--------------|--------------|
| Pro ($20) | 1× | 550,000 | 5,000,000 | baseline |
| Max 5× ($100) | 5× | 3,300,000 | 41,666,700 | **6× session, 8.33× weekly** |
| Max 20× ($200) | 20× | 11,000,000 | 83,333,300 | **20× session, 16.67× weekly** |

**Key insight:** Max 5× actually **overdelivers** (6× session, 8.33× weekly) while Max 20× **underdelivers** on weekly (16.67× instead of 20×). The 5× plan may offer better value per dollar.

### Cache Pricing: Subscription Advantage

| Operation | API Cost | Subscription Cost |
|-----------|----------|-------------------|
| Cache read | 10% of input rate | **FREE** |
| Cache write (5-min) | 1.25× input rate | Regular input price |
| Cache write (1-hour) | 2× input rate | Regular input price |

**This is massive for agentic work:** Our tool-heavy orchestration (dozens of tool calls per agent turn) generates heavy cache hits. Free cache reads mean warm-cache scenarios deliver up to **36× value over API**.

### Value Multipliers (API Equivalent)

| Plan | Monthly Cost | API Equivalent Value | Multiplier |
|------|-------------|---------------------|------------|
| Pro | $20 | $163 | 8.1× |
| Max 5× | $100 | $1,354 | 13.5× |
| Max 20× | $200 | $2,708 | 13.5× |

**Validation:** Our Jan 18 discovery showed $70-80/day API burn rate (~$2,100-2,400/mo). At 13.5× value multiplier, our $200/mo Max subscription delivers ~$2,708 equivalent API value - right at our burn rate. External validation of cost decision.

---

## Access Restrictions

### Anthropic Fingerprinting (Jan 2026)

**What's blocked:**
- Third-party tools (OpenCode, Cursor) using Max subscription OAuth
- Opus 4.5 via any non-Claude-Code path

**How it works:**
- TLS fingerprinting (JA3)
- HTTP/2 frame characteristics
- Tool name patterns (lowercase vs PascalCase+_tool)
- OAuth scope requirements

**Evidence:** All community workarounds failed within 6 hours. Anthropic iterates faster than bypasses can stabilize.

**Source:** `.kb/investigations/archived/2026-01-08-inv-opus-auth-gate-fingerprinting.md`, `.kb/investigations/archived/2026-01-09-inv-anthropic-oauth-community-workarounds.md`

#### Failed Bypass Timeline (Jan 8-9, 2026)

| Time | Attempt | Result |
|------|---------|--------|
| Jan 8 PM | Opus 4.5 fingerprint spoofing | Failed - "authorized for use with Claude Code only" |
| Jan 8 PM | Direct header injection | Failed - sophisticated detection, broke other models |
| Jan 9 AM | opencode-anthropic-auth@0.0.7 plugin | Worked briefly, then blocked |
| Jan 9 PM | Community coordination attempts | All failed within hours |

**Conclusion:** Cat-and-mouse is not a viable strategy. Anthropic's detection is actively maintained.

### Rate Limits vs Usage Quota (Critical Distinction)

**⚠️ There are TWO distinct limit types, and they behave differently:**

| Limit Type | Scope | Docker Bypass? | Resets |
|------------|-------|----------------|--------|
| **Request-rate limits** | Device (Statsig fingerprint) | ✅ YES | Immediately with new fingerprint |
| **Weekly usage quota** | Account | ❌ NO | Weekly (Sunday) |

**Request-rate limits:** Per-device throttling based on Statsig fingerprint. Fresh Docker container = fresh fingerprint = no throttling.

**Weekly usage quota:** Account-level weekly allocation (e.g., "97% used"). Tied to account, not device. Docker escape hatch does NOT bypass this.

**Test evidence (Jan 20-21):**
- Wiped `~/.claude-docker/`, logged in as gmail account
- Usage charged to gmail (2%→3%) while sendcutsend stayed at 94-95%
- Copying Statsig fingerprint did NOT bypass 97% quota

**When to use Docker escape hatch:**
- ✅ Request-rate limited (getting throttled, need to spawn immediately)
- ❌ Weekly quota exhausted (must wait for reset or switch accounts)

**Source:** `kb-e3e0a8`, `kb-c3dbe7`, `.kb/investigations/2026-01-21-inv-synthesize-anthropic-blocking-kb-quick.md`

### Cross-Account Rate Limit Bug

**Problem:** Device fingerprinting causes one account's rate limit to affect other accounts on same device.

**Status:** Known bug since March 2025 (GitHub #630), unfixed.

**Workaround:** Docker container (confirmed working for rate limits, not for quota).

**Source:** `~/.kb/investigations/2025-11-30-claude-code-cross-account-rate-limit-bug.md`

---

## DeepSeek V3: Viable Primary Option

### Previous Understanding (Jan 18)

> "The current version of the deepseek-chat model's Function Calling capability is unstable, which may result in looped calls or empty responses." - DeepSeek API docs

**Recommendation at the time:** Do not use for agentic work.

### Current Understanding (Jan 19)

**Test performed:** Spawned DeepSeek V3 agent via OpenCode API with investigation skill.

**Tool calls executed successfully:**
- Read (3 files)
- Grep (17 matches found)
- Bash (ran go test)
- Write (created investigation file)
- kb quick constrain (externalized knowledge)

**Result:** 3 minutes, 62K tokens, successful completion with SYNTHESIS.md.

**Conclusion:** DeepSeek V3 function calling is stable enough for standard orchestration tasks. The "unstable" warning may be outdated or apply only to edge cases.

### Revised Recommendation

| Workload | Recommended Model | Cost |
|----------|-------------------|------|
| Standard investigation/feature work | DeepSeek V3 | $0.25/$0.38/MTok |
| Complex reasoning, architecture | Claude Opus (via Max) | $200/mo flat |
| Tool-heavy bursts (>2000 req/min) | Sonnet API or Max | Variable |
| Cost-insensitive, highest quality | Claude Opus | $5/$25/MTok |

**Source:** `.orch/workspace/og-inv-test-deepseek-v3-19jan-25d3/SYNTHESIS.md`

---

## Cost Visibility Gap

### The Problem

Switched from free Gemini to paid Sonnet on Jan 9 with **no cost tracking**:
- Dashboard shows Max subscription usage (OAuth)
- Dashboard does NOT show API token usage
- No per-spawn cost visibility
- No budget alerts

**Result:** $402 spent in ~2 weeks without awareness, ramping to $70-80/day.

### The Solution (Not Yet Implemented)

**Hybrid approach recommended:**
1. **Local token counting** - Calculate from OpenCode session metadata, per-spawn granularity
2. **Anthropic Usage API** - `/v1/billing/cost` for ground truth (returns "Not Found" currently)
3. **Budget alerts** - Warn at 80%, critical at 95%, auto-switch at 100%

**Pricing for local calculation (Sonnet 4.5):**
- Input: $3.00/MTok
- Output: $15.00/MTok
- Cache write: $3.75/MTok
- Cache read: $0.30/MTok

**Source:** `.kb/investigations/archived/2026-01-12-inv-sonnet-cost-tracking-requirements.md`, `.kb/investigations/2026-01-18-inv-add-api-cost-tracking-widget.md`

---

## Spawn Path Economics

### Current Architecture (Dual Spawn)

| Path | Backend | Models | Cost | Use When |
|------|---------|--------|------|----------|
| **Primary** | OpenCode API | Sonnet, DeepSeek, Gemini | Pay-per-token | Normal work |
| **Escape Hatch** | Claude CLI | Opus, Sonnet (Max) | $200/mo flat | Infrastructure, high-quality |

### Economic Decision Tree

```
Is this infrastructure work (orch, opencode, spawn system)?
  → YES: Use escape hatch (Claude CLI + Max) - independence matters
  → NO: Continue...

Is cost the primary constraint?
  → YES: Use DeepSeek V3 ($0.25/$0.38/MTok)
  → NO: Continue...

Is highest quality needed?
  → YES: Use Opus via Max subscription
  → NO: Use Sonnet or DeepSeek V3
```

**Source:** `.kb/models/model-access-spawn-paths.md`, `.kb/decisions/2026-01-18-max-subscription-primary-spawn-path.md`

---

## Alternatives Evaluated

### OpenAI ChatGPT Pro ($200/mo)

- Codex CLI available (terminal agent like Claude Code)
- OAuth has bugs, less mature than Claude Code
- No API access included (same as Anthropic Max)
- Viable backup if Anthropic restricts further

**Source:** `.kb/investigations/archived/2026-01-18-inv-research-compare-openai-chatgpt-pro-anthropic-max.md`

### OpenCode Zen (Cooperative Buying Pool)

- Claims breakeven pricing via volume pooling
- Sustainability uncertain (no financial transparency)
- Not recommended until funding model proven

### OpenCode Black ($200/mo)

- Emergency response to Jan 9 Anthropic block
- Routes through "enterprise gateway" to bypass restrictions
- Temporary cat-and-mouse, not sustainable
- Treat as industry drama, not strategic option

**Source:** `.kb/investigations/archived/2026-01-13-research-opencode-zen-black-architecture-economics.md`

---

## Strategic Questions Answered

### Q: Is Sonnet API cheaper than Max subscription?

**A:** At normal usage, no. Jan 18 showed $70-80/day burn rate with Sonnet API → $2,100-2,400/mo projected. Max at $200/mo is 10x cheaper.

### Q: Should we use DeepSeek for orchestration?

**A:** Yes, now that function calling is confirmed working. DeepSeek V3 at $0.25/$0.38/MTok is viable for most orchestration work. Reserve Opus for complex reasoning.

### Q: When to use escape hatch (Max subscription)?

**A:** Infrastructure work (fixing orch/opencode itself), complex architecture decisions, or when API is rate-limited.

### Q: Is second Max subscription needed?

**A:** No. Cancelled Jan 13. Opus gate forced lighter consumption patterns. Can re-subscribe in 5 minutes if needed.

---

## Constraints

### C1: Anthropic Protects Claude Code Revenue

Fingerprinting blocks third-party tools from Max subscription. This is intentional product strategy, not a bug. Expect continued enforcement.

### C2: Cost Visibility Required Before API Usage

The $402 surprise proves: never use pay-per-token without cost tracking. Implement tracking before returning to Sonnet API as default.

### C3: Model Quality vs Task Complexity

**Benchmark caveat:** SWE-bench scores (R1: 49.2% vs Opus: 80.9%) measure complex multi-file coding, not general orchestration. Most orchestration tasks (investigation, search, file reading) don't require SWE-bench-level capability.

**Practical guidance:**
- Standard investigation/search → DeepSeek V3 sufficient (tested Jan 19)
- Complex multi-file refactoring → Prefer Claude
- Architecture decisions → Prefer Opus (reasoning quality matters)

### C4: Gemini Flash TPM Limits Block Tool-Heavy Work

2,000 req/min limit hit by single investigation agent (35+ tool calls/sec). Not viable for primary orchestration.

---

## References

### Decisions
- `.kb/decisions/2026-01-09-abandon-claude-max-oauth-use-gemini-primary.md`
- `.kb/decisions/2026-01-13-cancel-second-claude-max-subscription.md`
- `.kb/decisions/2026-01-18-max-subscription-primary-spawn-path.md`

### Investigations
- `.kb/investigations/archived/2026-01-28-inv-download-analyze-https-she-llac.md` - Credit formula, free cache reads, value multipliers
- `.kb/investigations/archived/2026-01-08-inv-opus-auth-gate-fingerprinting.md`
- `.kb/investigations/archived/2026-01-09-inv-anthropic-oauth-community-workarounds.md`
- `.kb/investigations/archived/2026-01-12-inv-sonnet-cost-tracking-requirements.md`
- `.kb/investigations/2026-01-18-inv-add-api-cost-tracking-widget.md`
- `.kb/investigations/archived/2026-01-18-research-compare-deepseek-models-anthropic-models.md`
- `.kb/investigations/2026-01-21-inv-synthesize-anthropic-blocking-kb-quick.md` - kb quick entries synthesis
- `~/.kb/investigations/2025-11-30-claude-code-cross-account-rate-limit-bug.md`

### Models & Guides
- `.kb/models/model-access-spawn-paths.md`
- `.kb/guides/model-selection.md`

### Test Evidence
- `.orch/workspace/og-inv-test-deepseek-v3-19jan-25d3/SYNTHESIS.md` - DeepSeek V3 function calling test

**Primary Evidence (Verify These):**
- Anthropic billing dashboard - Actual spend history showing $402 in ~2 weeks
- `~/.anthropic/` - OAuth token storage showing Max subscription authentication
- DeepSeek API documentation - Current pricing ($0.25/$0.38/MTok) and function calling status
- `.kb/decisions/2026-01-18-max-subscription-primary-spawn-path.md` - Economic decision
- `pkg/spawn/backend.go` - Dual spawn path implementation (OpenCode API vs Claude CLI)
- she-llac.com credit formula reverse engineering - Internal credit system documentation
