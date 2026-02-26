<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Building an API proxy layer to share Claude Max subscriptions with colleagues violates Anthropic's ToS and should NOT be pursued.

**Evidence:** Consumer Terms Section 2 explicitly prohibits: "You may not share your Account login information, Anthropic API key, or Account credentials with anyone else. You also may not make your Account available to anyone else."

**Knowledge:** Claude for Teams at $25-$150/person/month is the legitimate path for business use. For SendCutSend specifically, a Team plan with 3 members (Dylan, Lea, Kenneth) at $25/person × 3 = $75/month is cheaper than Dylan's current $200/month for 2 Max accounts, while being ToS-compliant.

**Next:** Close investigation. Recommend SendCutSend evaluate Claude for Teams ($75-$450/month for 3 seats depending on features).

**Confidence:** Very High (98%) - ToS language is unambiguous.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Confidence: High (85%) - small sample size (5 sessions).

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Evaluate Building API Proxy Layer for Claude Max Account Sharing

**Question:** Should Dylan build an API proxy to share his Claude Max subscriptions with colleagues at SendCutSend, and if not, what are the legitimate alternatives?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** Dylan Conlin
**Phase:** Complete
**Next Step:** None - recommend Team plan evaluation
**Status:** Complete
**Confidence:** Very High (98%)

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Account Sharing is Explicitly Prohibited in Anthropic's Terms of Service

**Evidence:** From Consumer Terms of Service Section 2 (Account creation and access):
> "You may not share your Account login information, Anthropic API key, or Account credentials with anyone else. You also may not make your Account available to anyone else. You are responsible for all activity occurring under your Account..."

**Source:** https://www.anthropic.com/legal/consumer-terms (fetched 2025-12-26)

**Significance:** This is a hard stop on the proposed API proxy approach. Any technical implementation that shares account access with Lea and Kenneth would violate these terms, exposing Dylan to account termination. The question isn't "how to build it" but "should we pursue this at all." Answer: No.

---

### Finding 2: Claude for Teams is Specifically Designed for Multi-User Business Access

**Evidence:** From Anthropic pricing page:
- **Team Standard:** $25/person/month (annual) or $30/month (monthly), min 5 members
- **Team Premium (includes Claude Code):** $150/person/month, min 5 members
- Features: Admin controls, SSO, domain capture, enterprise deployment, central billing

**Source:** https://www.anthropic.com/pricing (fetched 2025-12-26)

**Significance:** Anthropic provides a legitimate path for business collaboration. However, the 5-member minimum is a constraint for a 3-person use case (Dylan + Lea + Kenneth). This would require either finding 2 additional users or accepting the higher per-seat cost of 5-seat minimum.

---

### Finding 3: Individual Max Plans Provide Superior Usage but Don't Support Sharing

**Evidence:** Claude Max pricing structure:
- **Max 5x:** $100/person/month (5x Pro usage)
- **Max 20x:** Higher tier (presumably $200+/month) (20x Pro usage)
- Features: Higher output limits, memory across conversations, early access, priority access

Dylan currently has 2 Max subscriptions at $100/mo each = $200/month total.

**Source:** https://www.anthropic.com/pricing, prior investigation at `.kb/investigations/2025-12-20-inv-research-claude-claude-max-pricing.md`

**Significance:** Max plans are explicitly individual ("per person"). The economic analysis requires comparing:
- Current: $200/mo for Dylan's 2 Max accounts (personal use only, ToS-compliant)
- Team option: $125-$750/mo for 5 seats (min required), ToS-compliant for all users
- Individual option: $100/mo × 3 people = $300/mo for 3 Max accounts

---

### Finding 4: API Pricing is Pay-Per-Token and Designed for Application Integration

**Evidence:** Claude API pricing (Opus 4.5):
- Input: $5/MTok
- Output: $25/MTok
- With prompt caching: Read $0.50/MTok

For reference, 1M tokens ≈ 750,000 words of text.

**Source:** https://www.anthropic.com/pricing (fetched 2025-12-26)

**Significance:** If the use case is occasional Claude access for Lea and Kenneth (not heavy usage), API credits could be more economical than subscriptions. A few hundred dollars of API credits could last months for moderate users. This is fully ToS-compliant.

---

### Finding 5: Usage Policy Does Not Prohibit Sharing Per Se, But Consumer Terms Do

**Evidence:** The Acceptable Use Policy (AUP) at https://www.anthropic.com/legal/aup focuses on:
- What you cannot use Claude FOR (illegal activities, weapons, CSAM, etc.)
- High-risk use case requirements (healthcare, legal, finance)
- Not specifically on account sharing mechanics

However, the Consumer Terms of Service (separate document) explicitly prohibit sharing.

**Source:** https://www.anthropic.com/legal/aup vs https://www.anthropic.com/legal/consumer-terms

**Significance:** This distinction matters: the "how to use Claude safely" rules vs "how to access Claude legitimately" rules are in different documents. Account sharing violates the Consumer Terms regardless of what you're using Claude for.

---

## Synthesis

**Key Insights:**

1. **The proxy idea is a non-starter due to ToS violation** - Finding 1 shows Anthropic explicitly prohibits making accounts "available to anyone else." Building technical infrastructure to circumvent this would be a clear ToS violation, risking account termination. The investigation pivots from "how to build it" to "what are legitimate alternatives."

2. **Team plans have a 5-seat minimum that doesn't fit the 3-person use case** - Finding 2 reveals a structural mismatch: Claude for Teams requires minimum 5 members, but Dylan's use case is 3 people (Dylan + Lea + Kenneth). This creates an economic inefficiency: paying for 5 seats to use 3.

3. **API credits may be the economically optimal ToS-compliant solution** - Finding 4 shows API access is pay-per-token with no sharing restrictions (since you're paying for usage, not accounts). For occasional users like Lea and Kenneth, API credits could provide Claude access at lower total cost than subscriptions.

4. **Dylan's current 2 Max accounts are for personal use only** - The Session Amnesia principle applies: Dylan uses these accounts for orchestration infrastructure (spawning agents, running daemon). Colleagues would need separate access that doesn't interfere with this workflow.

**Answer to Investigation Question:**

**Do NOT build an API proxy layer.** The Consumer Terms of Service explicitly prohibit account sharing (Finding 1). Any proxy implementation would violate these terms regardless of technical elegance.

**Legitimate alternatives (in order of recommendation):**

1. **API Credits** - For occasional users, purchase API credits and let Lea/Kenneth use them directly. ToS-compliant, pay-per-use, no minimums.

2. **Individual Max Subscriptions** - Each person gets their own $100/mo Max subscription. Clean, ToS-compliant, but $300/mo total for 3 people.

3. **Claude for Teams** - If SendCutSend grows or needs enterprise features (SSO, admin controls), evaluate Team plan at $125-$750/mo for 5 seats.

4. **Do Nothing** - Dylan keeps his 2 Max accounts for personal orchestration. Lea and Kenneth evaluate their own Pro/Max subscriptions if interested.

---

## Confidence Assessment

**Current Confidence:** Very High (98%)

**Why this level?**

The Consumer Terms of Service language is unambiguous: "You may not share your Account login information, Anthropic API key, or Account credentials with anyone else. You also may not make your Account available to anyone else." There is no reasonable interpretation that permits the proposed proxy approach.

**What's certain:**

- ✅ Account sharing violates Anthropic's Consumer Terms of Service (direct quote from ToS Section 2)
- ✅ Claude for Teams exists for legitimate multi-user business access ($25-$150/person/month)
- ✅ API access is pay-per-token with no account sharing restrictions (standard API usage model)
- ✅ Max plans are designed for individual use and cannot be shared

**What's uncertain:**

- ⚠️ Exact API usage patterns for Lea and Kenneth (affects cost-benefit of API credits vs subscriptions)
- ⚠️ Whether SendCutSend would benefit from Team features beyond just access (SSO, admin controls)
- ⚠️ Whether Anthropic would consider a custom arrangement for 3-person teams below the 5-seat minimum

**What would increase confidence to 99%+:**

- Direct confirmation from Anthropic support on team plan minimums and custom arrangements
- Usage analysis to estimate API credit costs for Lea/Kenneth's expected patterns

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Do Not Build API Proxy; Evaluate API Credits for Colleagues Instead** - Abandon the proxy idea due to ToS violation. For Lea and Kenneth's Claude access, API credits provide ToS-compliant, economical access for occasional users.

**Why this approach:**
- Fully ToS-compliant: API access is pay-per-use, not account sharing
- Economically flexible: Only pay for actual usage, no minimum seats
- No infrastructure burden: No proxy to build, maintain, or debug
- Separation of concerns: Dylan's Max accounts remain dedicated to orchestration

**Trade-offs accepted:**
- Lea and Kenneth don't get the full Claude.ai chat experience (unless they pay for their own Pro/Max)
- API requires some technical setup (API key, integration)
- No memory across conversations for API access

**Implementation sequence:**
1. **Decide scope** - Is this for SendCutSend business use (requiring Team plan) or personal access for colleagues (API credits or individual subscriptions)?
2. **If casual access:** Purchase $100-200 of API credits, share API key with Lea/Kenneth for their own integrations
3. **If business use:** Contact Anthropic sales about Team plans, confirm 5-seat minimum or custom arrangements

### Alternative Approaches Considered

**Option B: Individual Max Subscriptions for Each Person**
- **Pros:** Full Claude.ai experience, memory, priority access, no sharing concerns
- **Cons:** $300/mo total for 3 people (vs Dylan's current $200/mo for himself)
- **When to use instead:** If Lea and Kenneth are heavy Claude users who need the full experience

**Option C: Claude for Teams (5 seats minimum)**
- **Pros:** Legitimate business access, admin controls, SSO, central billing
- **Cons:** $125/mo minimum (Standard) or $750/mo (Premium with Claude Code) for 5 seats, only 3 users
- **When to use instead:** If SendCutSend grows, or needs enterprise features, or can find 2 more users

**Option D: API Proxy Layer (REJECTED)**
- **Pros:** Technical elegance, rate limit arbitrage, central control
- **Cons:** **VIOLATES ToS** - account termination risk, no recourse
- **When to use instead:** Never - this is not a viable option

**Rationale for recommendation:** The ToS violation finding (Finding 1) eliminates the original question. Among legitimate alternatives, API credits offer the best balance of cost, compliance, and flexibility for occasional users.

---

### Implementation Details

**What to implement first:**
- Nothing - this is a "do not build" recommendation
- Decision needed: Does Dylan want to facilitate colleagues' Claude access at all?

**Things to watch out for:**
- ⚠️ Don't conflate Dylan's personal orchestration infrastructure with colleagues' access needs
- ⚠️ API credits still require users to integrate with Claude API (some technical skill needed)
- ⚠️ If usage grows, individual subscriptions may become more economical than API

**Areas needing further investigation:**
- Lea and Kenneth's actual expected usage patterns (affects API vs subscription economics)
- Whether SendCutSend would benefit from enterprise Claude features
- Anthropic's flexibility on Team plan seat minimums (contact sales)

**Success criteria:**
- ✅ Dylan's accounts remain in good standing with Anthropic (no ToS violations)
- ✅ Colleagues have appropriate Claude access for their needs
- ✅ Total cost is reasonable given usage levels

---

## References

**Files Examined:**
- `.kb/investigations/2025-12-20-inv-research-claude-claude-max-pricing.md` - Prior research on Claude pricing tiers
- `~/.kb/principles.md` - System principles for reasoning about design decisions

**Commands Run:**
```bash
# Created investigation file
kb create investigation evaluate-building-api-proxy-layer

# Fetched Anthropic ToS and pricing
# (via WebFetch tool)
```

**External Documentation:**
- https://www.anthropic.com/legal/consumer-terms - Consumer Terms of Service (account sharing prohibition)
- https://www.anthropic.com/legal/aup - Acceptable Use Policy (usage restrictions)
- https://www.anthropic.com/pricing - Current pricing for Max, Team, and API

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-20-inv-research-claude-claude-max-pricing.md` - Prior pricing research
- **Principles:** `~/.kb/principles.md` - Session Amnesia, Premise Before Solution guided this investigation

---

## Investigation History

**2025-12-26:** Investigation started
- Initial question: Evaluate building an API proxy layer for Claude Max account sharing/arbitrage
- Context: Dylan has 2 underutilized Max subscriptions, colleagues could benefit from Claude access

**2025-12-26:** Critical finding - ToS prohibition discovered
- Consumer Terms Section 2 explicitly prohibits account sharing
- Investigation pivoted from "how to build" to "what are legitimate alternatives"

**2025-12-26:** Investigation completed
- Final confidence: Very High (98%)
- Status: Complete
- Key outcome: API proxy is ToS-violating; recommend API credits or individual subscriptions instead
