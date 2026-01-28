<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** she-llac.com article reveals Claude subscription internals via reverse engineering: credit-based formula, 5-hour session limits, and FREE cache reads for subscribers vs 10% for API - orthogonal to stealth mode (about economics, not access).

**Evidence:** WebFetch extracted credit formulas and value multipliers; compared against 3 kb files (decision, investigation, model) - article adds new implementation detail but doesn't contradict our access patterns.

**Knowledge:** Max subscriptions deliver 13.5× API value; cache reads being free is massive for tool-heavy agentic work like ours; stealth mode is about ACCESS while this article is about USAGE - complementary, not overlapping concerns.

**Next:** Update orchestration-cost-economics model with credit formula details and free cache read insight; no action needed on stealth mode decision.

**Promote to Decision:** recommend-no - This is additional context for existing decisions, not a new architectural choice.

---

# Investigation: Analyze she-llac.com Claude Limits Article

**Question:** What does she-llac.com/claude-limits reveal about Claude usage limits, and how does it compare to/impact our stealth mode decision and existing kb knowledge?

**Started:** 2026-01-28
**Updated:** 2026-01-28
**Owner:** Worker agent (spawned from orch-go-20968)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Patches-Decision:** `.kb/decisions/2026-01-26-claude-max-oauth-stealth-mode-viable.md` - extends with economic validation
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Credit-Based Usage System (Internal Formula Exposed)

**Evidence:** The article reveals Claude.ai uses an internal credit system:

```
credits_used = ceil(input_tokens × input_rate + output_tokens × output_rate)
```

Per-model credit rates:
| Model | Input Rate | Output Rate |
|-------|------------|-------------|
| Haiku | 2/15 (0.133) | 10/15 (0.667) |
| Sonnet | 6/15 (0.4) | 30/15 (2.0) |
| Opus | 10/15 (0.667) | 50/15 (3.333) |

**Source:** https://she-llac.com/claude-limits (WebFetch extraction)

**Significance:** This is the internal pricing mechanism Anthropic uses to meter subscription usage. We've been thinking in tokens; they're thinking in credits. This doesn't change our usage patterns but explains the "usage %" we display in our dashboard.

---

### Finding 2: Actual Limit Values Differ from Marketing Claims

**Evidence:** The article provides specific credit limits:

| Plan | Marketing Claim | 5-Hour Session | Weekly Limit | Actual Ratio |
|------|-----------------|----------------|--------------|--------------|
| Pro ($20) | 1× | 550,000 | 5,000,000 | baseline |
| Max 5× ($100) | 5× | 3,300,000 | 41,666,700 | **6× session, 8.33× weekly** |
| Max 20× ($200) | 20× | 11,000,000 | 83,333,300 | **20× session, 16.67× weekly** |

The Max 5× plan actually **overdelivers** on session limits (6× instead of 5×) while Max 20× **underdelivers** on weekly limits (16.67× instead of 20×).

**Source:** https://she-llac.com/claude-limits (WebFetch extraction)

**Significance:** The Max 5× plan offers better value per dollar than Max 20× for weekly usage. This aligns with our current $200/mo Max usage but suggests the $100/mo tier might be sufficient for some workloads.

---

### Finding 3: Cache Reads are FREE for Subscribers (Major Agentic Advantage)

**Evidence:** Critical pricing difference for cache operations:

| Operation | API Cost | Subscription Cost |
|-----------|----------|-------------------|
| Cache read | 10% of input rate | **FREE** |
| Cache write (5-min) | 1.25× input rate | Regular input price |
| Cache write (1-hour) | 2× input rate | Regular input price |

Article quote: "In an agentic loop (e.g. Claude Code), the model makes dozens of tool calls per turn" - cache reads accumulate rapidly.

**Source:** https://she-llac.com/claude-limits (WebFetch extraction)

**Significance:** This is the most actionable new insight. Our tool-heavy orchestration (dozens of tool calls per agent turn) benefits massively from free cache reads. A warm-cache scenario shows **36× value over API**. This strongly validates our Max subscription choice.

---

### Finding 4: Value Multipliers Validate Max Subscription Economics

**Evidence:** Calculated API-equivalent values:

| Plan | Monthly Cost | API Equivalent Value | Multiplier |
|------|-------------|---------------------|------------|
| Pro | $20 | $163 | 8.1× |
| Max 5× | $100 | $1,354 | 13.5× |
| Max 20× | $200 | $2,708 | 13.5× |

**Source:** https://she-llac.com/claude-limits (WebFetch extraction)

**Significance:** Our Jan 18 discovery showed $70-80/day API burn rate (~$2,100-2,400/mo). At 13.5× value multiplier, our $200/mo Max subscription is delivering ~$2,708 equivalent API value - right at our burn rate. This validates our cost decision with external data.

---

### Finding 5: Data Obtained via Reverse Engineering (Not Official)

**Evidence:** The author describes methodology:
- SSE responses from generation endpoint returned unrounded doubles: `0.16327272727272726`
- Used Stern-Brocot tree algorithm to recover underlying fractions from IEEE-754 float precision artifacts
- "A whole bunch of manual data collection, then automated data collection after I modded my extension"

Author caveat: "I expect if this post gets any attention, that might not last very long" (regarding float precision leak)

**Source:** https://she-llac.com/claude-limits methodology section

**Significance:** This data could become stale if Anthropic patches the float precision leak. The credit formulas and ratios are implementation details that may change. However, the economic principles (subscription advantage for cached workloads) are likely stable.

---

### Finding 6: Article is Orthogonal to Stealth Mode Decision

**Evidence:** Comparing article scope vs our kb scope:

| Concern | Article Addresses | Our kb Addresses |
|---------|------------------|------------------|
| Usage limits (credits/tokens) | ✅ Yes | ✅ Yes (less detail) |
| OAuth access restrictions | ❌ No | ✅ Yes |
| Fingerprinting/detection | ❌ No | ✅ Yes |
| Stealth mode headers | ❌ No | ✅ Yes |
| Cache pricing economics | ✅ Yes (detailed) | ⚠️ Partial |
| Credit formula internals | ✅ Yes | ❌ No |

**Source:** Comparison of https://she-llac.com/claude-limits vs `.kb/decisions/2026-01-26-claude-max-oauth-stealth-mode-viable.md`, `.kb/investigations/2026-01-26-inv-analyze-pi-ai-anthropic-oauth.md`, `.kb/models/model-access-spawn-paths.md`

**Significance:** The article and our stealth mode decision address **different problems**: the article explains usage economics (how much you can use), while stealth mode solves access (whether you can use it at all). No contradiction exists.

---

## Synthesis

**Key Insights:**

1. **Different Problem Domains** - The she-llac article is about understanding Claude subscription VALUE (how much usage you get for your money), while our stealth mode work is about ACCESS (being able to use Max subscription from third-party tools at all). These are complementary, not competing.

2. **Free Cache Reads = Agentic Advantage** - The most actionable new insight is that cache reads are FREE for subscribers while costing 10% on API. Our tool-heavy orchestration generates massive cache hits. This makes subscription plans even more valuable than our previous analysis suggested.

3. **Value Multipliers Validate Our Decision** - The 13.5× value multiplier for Max plans aligns with our empirical Jan 18 discovery of $70-80/day API burn. External validation of our cost economics.

4. **5× Plan May Be Sufficient** - The Max 5× plan actually delivers 6× session limits and 8.33× weekly limits while costing half of Max 20×. Worth considering for cost optimization.

**Answer to Investigation Question:**

The she-llac.com article reveals detailed credit-based usage mechanics for Claude subscriptions, obtained via reverse engineering. Key findings:
- Credit formula and per-model rates exposed
- Actual limits differ from marketing (5× overdelivers, 20× underdelivers weekly)
- Cache reads are FREE for subscribers (major advantage for agentic work)
- Value multipliers (13.5×) validate our Max subscription choice

**Regarding stealth mode decision:** No impact. The article addresses usage economics, not OAuth access restrictions. Our stealth mode decision remains sound - it solves the ACCESS problem (using Max subscription from third-party tools), while this article explains the VALUE problem (how much that access is worth).

---

## Structured Uncertainty

**What's tested:**

- ✅ Article extracted via WebFetch - verified content includes credit formulas and limit values
- ✅ Comparison with kb files completed - 3 relevant documents reviewed
- ✅ No contradiction found between article and stealth mode decision

**What's untested:**

- ⚠️ Credit formula accuracy not independently verified (trust author's reverse engineering)
- ⚠️ Current validity unknown - Anthropic may have changed internal implementation since article
- ⚠️ Free cache read claim not tested with our OpenCode stealth mode implementation

**What would change this:**

- If Anthropic changed credit formula since article publication
- If subscription cache reads actually cost credits (would reduce value multiplier)
- If stealth mode access didn't receive same cache benefits as native Claude Code

---

## Implementation Recommendations

**Purpose:** Bridge investigation findings to actionable updates.

### Recommended Approach ⭐

**Update orchestration-cost-economics model** - Add credit formula details and free cache read insight to existing model.

**Why this approach:**
- Enriches existing documentation without creating new artifacts
- Credit formulas provide concrete numbers for future cost calculations
- Free cache read insight explains why Max is so valuable for our use case

**Trade-offs accepted:**
- Data may become stale if Anthropic patches float precision leak
- Credit formulas are implementation details, not official documentation

**Implementation sequence:**
1. Add "Internal Credit System" section to `.kb/models/orchestration-cost-economics.md`
2. Update cache pricing comparison with free-read insight
3. Note Max 5× overdelivery for cost optimization discussions

### Alternative Approaches Considered

**Option B: Create new decision record**
- **Pros:** Formally documents validation of Max subscription choice
- **Cons:** No new decision being made - this validates existing decision
- **When to use instead:** If we were changing subscription tier based on this

**Option C: No action**
- **Pros:** Investigation provides context, no changes needed
- **Cons:** Loses opportunity to improve cost economics documentation
- **When to use instead:** If documentation already sufficient

**Rationale for recommendation:** The findings enrich our understanding without changing decisions. Model update captures the knowledge for future reference.

---

### Implementation Details

**What to implement first:**
- Add credit formula to orchestration-cost-economics.md
- Note free cache read advantage

**Things to watch out for:**
- ⚠️ Data may be stale - article methodology relies on float precision leak
- ⚠️ Don't treat credit values as official - they're reverse-engineered

**Areas needing further investigation:**
- Whether stealth mode gets same cache benefits as native Claude Code
- Whether Max 5× ($100/mo) is sufficient for our usage patterns

**Success criteria:**
- ✅ Model updated with new insights
- ✅ No changes to stealth mode decision (validated, not changed)

---

## References

**Files Examined:**
- `.kb/decisions/2026-01-26-claude-max-oauth-stealth-mode-viable.md` - Stealth mode decision context
- `.kb/investigations/2026-01-26-inv-analyze-pi-ai-anthropic-oauth.md` - pi-ai implementation analysis
- `.kb/models/model-access-spawn-paths.md` - Spawn path economics
- `.kb/models/orchestration-cost-economics.md` - Cost economics model
- `.kb/investigations/2025-12-20-research-model-arbitrage-api-vs-max.md` - Prior API vs Max analysis

**Commands Run:**
```bash
# Investigation setup
kb create investigation download-analyze-https-she-llac

# Report phase
bd comment orch-go-20968 "Phase: Planning - Analyzing task, reading kb context files"

# Search for existing knowledge
Grep for credit, cache, weekly quota patterns in .kb/
```

**External Documentation:**
- https://she-llac.com/claude-limits - Primary source (analyzed via WebFetch)

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-26-claude-max-oauth-stealth-mode-viable.md` - Stealth mode decision (validated, not changed)
- **Model:** `.kb/models/orchestration-cost-economics.md` - Should be updated with findings
- **Model:** `.kb/models/model-access-spawn-paths.md` - Access patterns (orthogonal to this)

---

## Investigation History

**2026-01-28 ~10:00:** Investigation started
- Initial question: What does she-llac.com reveal about Claude limits and how does it impact stealth mode decision?
- Context: Spawned from orch-go-20968 to analyze external article

**2026-01-28 ~10:15:** Article fetched and analyzed
- WebFetch extracted credit formulas, limit values, cache pricing
- Identified article is about usage economics, not OAuth access

**2026-01-28 ~10:30:** KB comparison completed
- Read 5 related kb files for context
- Found no contradiction with stealth mode decision
- Identified new insight: free cache reads for subscribers

**2026-01-28 ~10:45:** Investigation completed
- Status: Complete
- Key outcome: Article validates Max subscription economics (13.5× value), adds credit formula details, is orthogonal to stealth mode (different problem domain)
