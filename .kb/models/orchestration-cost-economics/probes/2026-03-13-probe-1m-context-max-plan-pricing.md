# Probe: 1M Context Window on Max Plan — Pricing and Availability

**Model:** orchestration-cost-economics
**Date:** 2026-03-13
**Status:** Complete

---

## Question

Does the "Opus 4.6 (1M context)" shown in Claude Code status bar cost extra money beyond the Max subscription flat rate? When did 1M context become available on Max, and is it a billable add-on?

The cost economics model documents Max subscription as "$200/mo flat" with credit-based metering, but doesn't address whether 1M context (previously an API-only premium feature) has pricing implications for Max subscribers.

---

## What I Tested

Fetched and analyzed four authoritative sources:

1. **Anthropic Models page** (`platform.claude.com/docs/en/docs/about-claude/models`)
   - Checked context window specs for Opus 4.6 vs legacy models

2. **Anthropic Pricing page** (`platform.claude.com/docs/en/about-claude/pricing`)
   - Checked "Long context pricing" section specifically

3. **Anthropic Context Windows docs** (`platform.claude.com/docs/en/build-with-claude/context-windows`)
   - Checked 1M availability and restrictions

4. **Claude Code changelog** (via claude-code-guide agent, `code.claude.com/docs/en/changelog.md`)
   - Checked when 1M was enabled for Max subscribers

5. **Claude Code version check:**
```bash
claude --version
# 2.1.75 (Claude Code)
```

---

## What I Observed

### Finding 1: 1M is the NATIVE context window for Opus 4.6 (not an add-on)

From the models page, the latest models comparison table:

| Model | Context Window |
|-------|---------------|
| **Claude Opus 4.6** | **1M tokens** |
| **Claude Sonnet 4.6** | **1M tokens** |
| Claude Haiku 4.5 | 200k tokens |
| Claude Opus 4.5 (legacy) | 200k tokens |
| Claude Opus 4.1 (legacy) | 200k tokens |
| Claude Sonnet 4.5 (legacy) | 200k (1M via beta header) |

**Key distinction:** For legacy models, 1M required a beta header (`context-1m-2025-08-07`) and incurred premium pricing. For Opus 4.6 and Sonnet 4.6, 1M IS the standard context window.

### Finding 2: NO extra cost for 1M on Opus 4.6/Sonnet 4.6

From the pricing page, "Long context pricing" section (verbatim):

> **Claude Opus 4.6 and Sonnet 4.6 include the full 1M token context window at standard pricing.** (A 900k-token request is billed at the same per-token rate as a 9k-token request.) Prompt caching and batch processing discounts apply at standard rates across the full context window.

The premium pricing (2x input, 1.5x output) applies ONLY to legacy Sonnet 4.5/4 when using the beta header.

### Finding 3: Max subscription covers 1M at no extra charge

Since 1M is standard pricing for Opus 4.6 (not premium), and Max subscription covers standard usage at flat rate, there is NO extra charge.

The credit formula from the model still applies:
```
credits_used = ceil(input_tokens × input_rate + output_tokens × output_rate)
```
A 900K input request costs 9x the credits of a 100K request, but it's all within the Max subscription's credit allocation — no per-token API billing.

### Finding 4: Claude Code v2.1.75 timeline

From the Claude Code changelog:
- **v2.1.45 (Feb 17, 2026):** Initial Opus 4.6 support with 1M capability
- **v2.1.50 (Feb 20, 2026):** Added `CLAUDE_CODE_DISABLE_1M_CONTEXT` env var
- **v2.1.75 (Mar 13, 2026):** 1M enabled by default for Max, Team, Enterprise plans

### Finding 5: How to control/disable

```bash
# Disable 1M context (revert to 200K behavior)
export CLAUDE_CODE_DISABLE_1M_CONTEXT=1
```

Or use model selection syntax:
```
/model opus        # Uses 1M by default on Max
/model opus[1m]    # Explicitly request 1M
```

---

## Model Impact

- [x] **Confirms** invariant: Max subscription ($200/mo flat) is the primary economic path — 1M context doesn't change this
- [x] **Confirms** invariant: Anthropic protects Claude Code revenue (1M on Max is a competitive advantage over API-only)
- [x] **Extends** model with: 1M context is now standard for Opus 4.6/Sonnet 4.6 at no premium pricing. The "Long context pricing" section from legacy models (premium rates for >200K) does NOT apply to 4.6 models. This significantly increases the value proposition of Max subscription — previous Opus models had 200K context, now 5x more context at the same credit rates.
- [x] **Extends** model with: Claude Code v2.1.75 explicitly enables 1M by default for Max subscribers. Controllable via `CLAUDE_CODE_DISABLE_1M_CONTEXT` env var.

### Model Update Needed

The pricing comparison table should be updated:
- Opus 4.6 context: 200K → 1M (at standard pricing)
- Add note about `CLAUDE_CODE_DISABLE_1M_CONTEXT` control
- Value multiplier analysis should account for 5x context increase

The credit formula still applies — longer contexts consume more credits per request, but the per-token rate is unchanged. The economic trade-off is: more context = fewer requests before hitting weekly quota, but each request can do more work (less compaction, fewer context-switching spawns).

---

## Notes

### Financial Impact Assessment

**Short answer for Dylan: You are NOT being charged extra.** The 1M context on Max is:
1. Standard pricing (not premium)
2. Covered by flat-rate subscription
3. Uses the same credit system documented in the model

**Potential indirect cost:** Larger context windows consume credits faster per request (more input tokens = more credits). A 900K token request uses 9x the credits of a 100K request. This could cause weekly quota exhaustion faster if sessions consistently use very large contexts. However, this is self-regulating — the account distribution system already handles quota management.

### What Changed Architecturally

The shift from 200K to 1M context for Opus 4.6 has operational implications for orch-go:
1. **Less context compaction needed** — long agent sessions can maintain more history
2. **Fewer context-window-driven failures** — agents hitting 200K limit was a known friction point
3. **Credit consumption pattern change** — sessions may use more credits per request but fewer total requests (net effect likely neutral or positive)
