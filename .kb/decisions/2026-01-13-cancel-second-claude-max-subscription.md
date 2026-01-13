## Summary (D.E.K.N.)

**Delta:** Cancel second Claude Max subscription, operate on single account ($2,400/year savings).

**Evidence:** Currently not hitting weekly usage limits (previously maxed out when using Opus heavily). Opus gate forces Sonnet/Gemini workflow with lighter token consumption. Second account sitting idle at $200/month.

**Knowledge:** The need for dual subscriptions was caused by heavy Opus usage. Opus gate elimination of that usage pattern removed the capacity constraint. Re-subscribing takes 5 minutes if needed.

**Next:** Cancel second subscription. Track escape hatch usage via orch-go-vnqgd to detect usage pattern changes. Monitor for Opus gate announcements (Google Alert).

---

# Decision: Cancel Second Claude Max Subscription

**Date:** 2026-01-13
**Status:** Accepted

<!-- Lineage (fill only when applicable) -->
**Related-To:**
- `.kb/investigations/2026-01-08-inv-opus-auth-gate-fingerprinting.md` - Documents why Opus gate forces workflow change
- `.kb/models/escape-hatch-visibility-architecture.md` - Documents escape hatch spawning architecture
- Issue `orch-go-vnqgd` - Tracks escape hatch usage to inform future subscription decisions

---

## Context

**Background:** Originally subscribed to 2 Claude Max accounts ($400/month total) because heavy Opus usage was hitting weekly conversation limits on a single account. Needed capacity redundancy to maintain uninterrupted orchestration workflow.

**What changed:** Anthropic's Opus gate (auth fingerprinting) blocks Opus 4.5 access via OpenCode API. Only bypass requires Claude CLI via `--backend claude` flag, which still uses Max subscription but forces workflow shift.

**Current state:**
- Forced to Sonnet 4.5 / Gemini Flash for most work (lighter token consumption)
- No longer hitting weekly limits on primary account
- Second account sitting idle ($200/month waste)
- Escape hatch spawns use SAME account as orchestrator (no concurrency benefit from dual accounts)

**Decision trigger:** Analyzing escape hatch usage patterns to inform subscription optimization.

---

## Options Considered

### Option A: Keep Both Subscriptions (Status Quo)
- **Pros:**
  - Immediate capacity if Opus gate lifts
  - Insurance against unexpected usage spikes
  - No re-subscription friction
- **Cons:**
  - $2,400/year cost for idle capacity
  - No current benefit (not hitting limits)
  - Pays for hypothetical future need

### Option B: Cancel Second Subscription
- **Pros:**
  - $2,400/year savings
  - Can re-subscribe in 5 minutes if needed
  - Optimizes for current reality (not hitting limits)
  - Tracking issue (orch-go-vnqgd) will surface usage changes early
- **Cons:**
  - 1-2 week delay if Opus gate lifts suddenly and I miss announcement
  - Potential brief capacity constraint if usage spikes unexpectedly
  - Re-subscription requires manual action

### Option C: Hybrid (Keep Temporarily, Re-evaluate Quarterly)
- **Pros:**
  - More data before committing
  - Safe if uncertain about usage trends
- **Cons:**
  - Delays decision without adding information (already have clear data)
  - $200/month cost to defer decision

---

## Decision

**Chosen:** Option B (Cancel Second Subscription)

**Rationale:**

1. **Current capacity is sufficient:** Not hitting limits for months due to Opus gate forcing lighter consumption models
2. **Low re-activation risk:** Re-subscribing takes 5 minutes; worst case is 1-2 week delay if Opus gate lifts and I miss announcement
3. **Monitoring in place:** orch-go-vnqgd tracking issue will surface escape hatch usage trends and catch capacity creep early
4. **Opus gate unlikely to lift soon:** Anthropic protecting Claude Code subscription value; API access unlikely in 2026
5. **No concurrency benefit:** Escape hatch uses same account as orchestrator (both can't run simultaneously anyway)

**Trade-offs accepted:**
- Potential 1-2 week disruption if Opus gate lifts suddenly (mitigated by Google Alert monitoring)
- Manual re-subscription action if usage spikes (5-minute task, acceptable friction)
- Loss of "insurance" feeling (psychological, not practical)

---

## Structured Uncertainty

**What's tested:**
- ✅ Currently not hitting weekly limits on single account (observed for multiple months)
- ✅ Opus gate blocks OpenCode API access (investigation: 2026-01-08-inv-opus-auth-gate-fingerprinting.md)
- ✅ Second account unused in current workflow (account dashboard shows minimal activity)
- ✅ Re-subscription activation time is ~5 minutes (tested with prior account switches)
- ✅ Escape hatch uses same account as orchestrator (verified in spawn code: pkg/account/account.go)

**What's untested:**
- ⚠️ Assumption Opus gate will remain in place through 2026 (Anthropic could reverse policy)
- ⚠️ Assumption current Sonnet/Gemini usage won't increase significantly (workflow could change)
- ⚠️ Assumption re-subscription is always fast (could have delays during high-demand periods)

**What would change this:**
- Opus gate lifts and API access restored → would immediately hit limits again, need 2nd subscription
- Workflow changes to much heavier orchestration usage → capacity constraint emerges
- Anthropic introduces new premium model worth heavy usage → similar to original Opus pattern
- Weekly limit structure changes (reduced limits) → would hit constraints sooner

---

## Consequences

**Positive:**
- $2,400/year cost savings (12 months × $200/month)
- Optimized for current reality (not paying for unused capacity)
- Clear monitoring via orch-go-vnqgd (usage trends visible in `orch stats`)
- Forces intentional re-evaluation if circumstances change
- Simpler mental model (single account, no switching logic)

**Risks:**
- 1-2 week capacity gap if Opus gate lifts suddenly and announcement missed
- Brief workflow disruption if re-subscription has unexpected delays
- Potential for "I told you so" regret if need to re-subscribe within 3 months
- Must remember to set up Google Alert for "Anthropic Opus API access" (mitigation step)
