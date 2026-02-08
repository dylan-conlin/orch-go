<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** OpenCode Zen is a potentially viable cooperative buying pool with breakeven economics (pending transparency), while OpenCode Black is a temporary emergency response to Anthropic's Jan 9, 2026 crackdown - not a sustainable product.

**Evidence:** Zen operates at stated breakeven with volume pooling model (verified from official docs); Black launched within hours of Anthropic's OAuth block on Jan 9 (verified from multiple news sources); economic arbitrage was 5-10x (Claude Max $200/month vs $1,000+ API equivalent, verified from community analysis); no cease & desist letter exists (extensive web search found zero legal evidence).

**Knowledge:** The massive pricing arbitrage (5-10x) made Anthropic's crackdown inevitable - they were subsidizing competitors. Community loyalty was to the arbitrage pricing, not to Claude specifically (mass cancellations when arbitrage disappeared). OpenCode Zen's sustainability depends entirely on whether "breakeven" is genuine cooperative economics or venture-subsidized loss-leading - no financial transparency exists to assess. Technical enforcement can be circumvented (hence OpenCode Black), but it's a cat-and-mouse game with no long-term viability.

**Next:** Maintain status quo (direct Anthropic API + local OpenCode server) for orch-go. Monitor OpenCode Zen for financial transparency signals. Track when OpenCode Black gets blocked to validate "temporary" assessment. Do not adopt either service until sustainability is proven.

**Promote to Decision:** recommend-no - This is validation research for existing kb quick decision (kb-b764f4), not new architectural choice requiring decision record.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: OpenCode Zen/Black Architecture, Economics, and Sustainability

**Question:** What are OpenCode Zen and OpenCode Black, how do they work architecturally, what is their economic model, and are they sustainable business models or temporary market responses to Anthropic's pricing arbitrage?

**Started:** 2026-01-13
**Updated:** 2026-01-13
**Owner:** Agent og-research-investigate-opencode-zen-13jan-fd44
**Phase:** Complete
**Next Step:** None - investigation complete, SYNTHESIS.md creation next
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: OpenCode Zen is a Cooperative Buying Pool, Not a Traditional Vendor

**Evidence:**
- OpenCode Zen is explicitly "not a for-profit thing" operating at breakeven
- Uses volume pooling model: "Every new person that starts to use Zen, they're pooling all of our volume together and we're going to providers and negotiating discounted rates"
- Cost savings flow directly to users: "When we do that, the cost savings flow right back down to everyone"
- Zero markup pricing with only processing fees (4.4% + $0.30 per transaction)
- Claims to offer "the cheapest possible rates that exist out there right now"

**Source:**
- [OpenCode Zen official page](https://opencode.ai/zen)
- [OpenCode documentation](https://opencode.ai/docs/zen/)
- Web searches on pricing and economic model

**Significance:** This is a fundamentally different business model from traditional AI gateway providers like OpenRouter. Instead of marking up API access for profit, OpenCode Zen operates as a collective purchasing cooperative. This raises sustainability questions - can a breakeven model survive long-term, or does it depend on venture funding subsidies?

---

### Finding 2: OpenCode Black is an Emergency Response to Anthropic's Crackdown, Not a Planned Product

**Evidence:**
- Timeline: Jan 6 2026 - OpenCode operating normally with Claude Max subscriptions
- Jan 9 2026, 02:20 UTC - Anthropic blocked third-party OAuth tokens without warning
- Within hours - OpenCode shipped "OpenCode Black" as $200/month tier
- OpenCode Black routes through "enterprise API gateway" to bypass consumer OAuth restrictions
- Community characterized it as rapid countermeasure, not strategic product launch

**Source:**
- [VentureBeat article on Anthropic crackdown](https://venturebeat.com/technology/anthropic-cracks-down-on-unauthorized-claude-usage-by-third-party-harnesses)
- [Paddo.dev timeline analysis](https://paddo.dev/blog/anthropic-walled-garden-crackdown/)
- [Hacker News discussion](https://news.ycombinator.com/item?id=46549823)
- Local kb quick entry: kb-b764f4 noting "Timeline suspicious"

**Significance:** OpenCode Black appears to be a reactive patch, not a sustainable product. The rapid launch suggests desperation to maintain Claude access for users rather than a well-designed business offering. The lack of transparency on refund policies and limits (noted in kb quick entry) reinforces this as "industry drama" rather than strategic product evolution.

---

### Finding 3: The Economic Arbitrage Driving This Was Massive - 5x to 10x Price Difference

**Evidence:**
- Claude Max subscription: $200/month for unlimited tokens through Claude Code
- Equivalent API usage for power users: $1,000+ per month
- Third-party tools enabled "autonomous agent loops" - extended overnight processes without human intervention
- This arbitrage was only economically viable on flat-rate plans, not pay-per-token
- Anthropic's Section D.4 commercial use restrictions explicitly prohibit "using the API to access the Services to build a competing product or service"

**Source:**
- [Hacker News community analysis](https://news.ycombinator.com/item?id=46549823)
- [Paddo.dev economics analysis](https://paddo.dev/blog/anthropic-walled-garden-crackdown/)
- [VentureBeat reporting](https://venturebeat.com/technology/anthropic-cracks-down-on-unauthorized-claude-usage-by-third-party-harnesses)

**Significance:** This wasn't a minor price optimization - third-party tools were accessing 5-10x cheaper compute by routing through consumer subscriptions. Anthropic's crackdown makes perfect business sense: they were subsidizing competitor products (OpenCode, Cursor) that directly cannibalized Claude Code adoption. The sustainability question is: was OpenCode ever viable without this arbitrage?

---

### Finding 4: OpenCode Zen Architecture is Standard AI Gateway + Curated Model List

**Evidence:**
- 26+ models across OpenAI, Anthropic, Google, Asian providers (GLM, Kimi, Qwen, Grok, Big Pickle, MiniMax)
- Standard provider integration pattern: authenticate → get API key → route requests via dedicated endpoints
- Configuration format: `opencode/<model-id>`
- US-only hosting with zero-retention policies (except OpenAI/Anthropic 30-day retention per their policies)
- Beta phase with free workspace management
- Model metadata endpoint: `https://opencode.ai/zen/v1/models`

**Source:**
- [OpenCode Zen documentation](https://opencode.ai/docs/zen/)
- [OpenCode Zen landing page](https://opencode.ai/zen)
- WebFetch analysis of official docs

**Significance:** Architecturally, OpenCode Zen is unremarkable - it's a standard AI gateway aggregator. The differentiation is purely economic (breakeven pricing) and curation (tested models). This suggests the business model is the moat, not technical innovation. If the breakeven model fails, there's little to prevent users from switching to OpenRouter or direct API access.

---

### Finding 5: Community Response Was Immediate Mass Cancellation, Not Loyalty

**Evidence:**
- GitHub issue accumulated 147+ reactions within hours of Jan 9 block
- Hacker News discussion hit 245+ points same day
- DHH (Ruby on Rails creator) called it "very customer hostile"
- Multiple users reported immediate cancellation: "I immediately downgraded my $200/month Max subscription, then canceled entirely"
- Developer community split on blame - some sided with Anthropic (protecting business), others with OpenCode (vendor lock-in concerns)

**Source:**
- [Multiple](https://news.ycombinator.com/item?id=46549823) [community](https://paddo.dev/blog/anthropic-walled-garden-crackdown/) [sources](https://byteiota.com/anthropic-blocks-claude-max-in-opencode-devs-cancel-200-month-plans/)
- Web search results on developer reactions

**Significance:** The rapid mass cancellation reveals that user loyalty to Anthropic was contingent on the arbitrage pricing being accessible through preferred tools. When forced to choose between Claude via official client vs. cheaper/different models via OpenCode, many chose the latter. This validates concerns about vendor lock-in and suggests Claude's moat is shallower than Anthropic hoped.

---

### Finding 6: No Evidence of Cease & Desist Letter - This Was Pure Technical Enforcement

**Evidence:**
- Extensive web search found zero mentions of actual C&D letter from Anthropic to OpenCode
- Anthropic staff (Thariq Shihipar) publicly stated they "tightened our safeguards against spoofing the Claude Code harness"
- Enforcement method: OAuth token fingerprinting and client identity verification
- No official statement from Anthropic Legal, only engineering team technical comments
- Local kb quick entry (kb-b764f4) mentions "Anthropic C&D" but web research finds no evidence

**Source:**
- Web search: "OpenCode Black Anthropic cease and desist January 2026" returned zero C&D evidence
- [VentureBeat technical analysis](https://venturebeat.com/technology/anthropic-cracks-down-on-unauthorized-claude-usage-by-third-party-harnesses)
- Multiple community discussions reference "the block" or "crackdown" but never "cease and desist"

**Significance:** The kb quick entry claiming "Jan 7 Anthropic C&D" appears to be inaccurate community shorthand, not legal fact. This was a technical enforcement action (API restrictions), not legal threat. This distinction matters for assessing risk: technical blocks can be circumvented (hence OpenCode Black), whereas legal C&D would create ongoing liability concerns.

---

## Synthesis

**Key Insights:**

1. **OpenCode Zen and Black Represent Two Different Business Strategies** - OpenCode Zen (the cooperative buying pool at breakeven) is a legitimate attempt at sustainable infrastructure, while OpenCode Black (the $200/month emergency patch) is a temporary market response to Anthropic's crackdown. These should not be conflated - Zen has strategic merit, Black is tactical desperation.

2. **The Arbitrage Was Unsustainable and Anthropic's Response Was Inevitable** - A 5-10x price difference between consumer subscriptions and equivalent API usage created perverse incentives. Third-party tools were effectively laundering cheap consumer pricing for commercial/competitive use. Anthropic's technical enforcement (not legal C&D) was a rational business decision to stop subsidizing competitors.

3. **The Breakeven Model's Sustainability Depends on What You're Buying** - If OpenCode Zen is truly negotiating bulk discounts from providers and passing savings through, it's sustainable as a cooperative. But if it's venture-subsidized to build market share, it's not sustainable. The lack of transparency on funding model makes this impossible to assess.

4. **Community Loyalty Was to Pricing Arbitrage, Not to Anthropic** - Mass cancellations within hours of the Jan 9 block reveal that users weren't loyal to Claude specifically - they were loyal to accessing Claude at 5-10x discount through preferred tools. When the arbitrage disappeared, so did the subscriptions. This has implications for vendor lock-in strategies.

5. **OpenCode Black Has No Long-Term Viability** - Routing through "enterprise API gateway" to bypass consumer OAuth restrictions is a cat-and-mouse game. If Anthropic blocks this too (which they can via usage pattern detection), OpenCode Black collapses. The lack of refund policy transparency suggests OpenCode knows this is temporary.

**Answer to Investigation Question:**

**OpenCode Zen** is a legitimate AI gateway operating as a cooperative buying pool with a breakeven economic model. Architecturally unremarkable (standard gateway + curated models), its differentiation is purely economic. Sustainability is uncertain - depends on whether volume pooling genuinely achieves discounts or if it's venture-subsidized. If the former, sustainable; if the latter, not.

**OpenCode Black** is an emergency market response to Anthropic's Jan 9, 2026 OAuth crackdown, not a sustainable product. It's a $200/month patch routing through enterprise gateways to bypass consumer restrictions. This is a temporary technical workaround in a cat-and-mouse game with Anthropic, not a durable business model.

**Recommendation:** Track OpenCode Zen as a potentially viable alternative to OpenRouter for cost-sensitive users, but with skepticism until funding model is transparent. Treat OpenCode Black as industry drama and temporary arbitrage play - not a strategic option for production systems. The kb quick decision (kb-b764f4: "Track OpenCode Black as industry drama, not strategic option") is validated by this research.

---

## Structured Uncertainty

**What's tested:**

- ✅ OpenCode Zen pricing model is pay-as-you-go with 4.4% + $0.30 processing fees (verified from official docs)
- ✅ OpenCode Black was launched immediately after Jan 9, 2026 Anthropic block (verified from multiple news sources and community timelines)
- ✅ Anthropic's enforcement was technical (OAuth fingerprinting), not legal C&D (verified: zero evidence of C&D letter in extensive web search)
- ✅ Economic arbitrage was 5-10x (Claude Max $200/month vs $1,000+ API equivalent) (verified from Hacker News analysis and VentureBeat reporting)
- ✅ Community response included mass cancellations (verified from GitHub issues, Hacker News, and developer blog posts)
- ✅ OpenCode Zen hosts 26+ models across multiple providers (verified from official docs)

**What's untested:**

- ⚠️ Whether OpenCode Zen's "breakeven" claim is accurate or venture-subsidized (no financial transparency)
- ⚠️ Whether volume pooling actually achieves discounted provider rates vs. loss-leading for market share
- ⚠️ OpenCode Black's actual architecture - "enterprise API gateway" is vague, could be multiple things
- ⚠️ Sustainability of OpenCode Black if Anthropic implements usage pattern detection (not tested, just assumed)
- ⚠️ Refund policy for OpenCode Black users if service shuts down (no public policy found)
- ⚠️ How long OpenCode Zen has been operating at breakeven (launch date unclear, sustainability timeline unknown)

**What would change this:**

- Finding OpenCode Zen financial statements or funding disclosures would clarify breakeven vs. subsidized model
- Evidence of OpenCode Black being blocked again would validate the "cat-and-mouse" prediction
- Published refund policy from OpenCode would change risk assessment for users
- Transparent provider contracts showing actual bulk discount terms would validate the cooperative model claim
- Evidence of venture funding rounds would indicate subsidy model, contradicting "breakeven" positioning

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable decisions for orch-go project's model provider strategy.

### Recommended Approach ⭐

**Status Quo: Continue using direct Anthropic API + local OpenCode server** - Maintain current orch-go architecture without switching to OpenCode Zen or Black.

**Why this approach:**
- OpenCode Zen's breakeven model is unproven without financial transparency (Finding 1)
- OpenCode Black is temporary arbitrage play, not durable (Finding 2, 5)
- We already have direct API access and OpenCode server running locally - no migration cost
- Anthropic's crackdown shows they're willing to kill third-party integrations without warning (Finding 2)

**Trade-offs accepted:**
- Potentially paying slightly more than OpenCode Zen's pooled rates (if the cooperative model is genuine)
- Missing out on curated model testing that OpenCode Zen provides
- Not benefiting from multi-provider aggregation in single gateway

**Implementation sequence:**
1. **Monitor OpenCode Zen for transparency signals** - If they publish financials showing genuine cooperative economics, reassess
2. **Track OpenCode Black lifespan** - When Anthropic blocks it (predicted), validate the "temporary" assessment
3. **Keep existing architecture** - Local OpenCode server + direct API keys gives us flexibility without vendor lock-in

### Alternative Approaches Considered

**Option B: Switch to OpenCode Zen for cost savings**
- **Pros:** Potentially cheaper rates via volume pooling; curated model testing; multi-provider access
- **Cons:** Unproven sustainability (Finding 1); adds another intermediary; risk of service shutdown; no transparency on funding model
- **When to use instead:** If OpenCode publishes financials proving genuine cooperative economics, or if our API costs become prohibitive

**Option C: Adopt OpenCode Black to access Claude cheaply**
- **Pros:** Maintains Claude access at flat $200/month rate similar to pre-crackdown
- **Cons:** Temporary workaround in cat-and-mouse game (Finding 5); no refund policy transparency; ethical gray area (circumventing Anthropic's intended restrictions)
- **When to use instead:** Never - this is industry drama, not strategic infrastructure (validated by kb quick decision kb-b764f4)

**Option D: Use OpenRouter as multi-provider gateway**
- **Pros:** Established business, transparent pricing, proven track record, not currently in conflict with providers
- **Cons:** Markup on API pricing (unlike OpenCode Zen's claimed breakeven model); less aggressive on cost optimization
- **When to use instead:** If we need multi-provider aggregation and prefer established vendor over unproven cooperative model

**Rationale for recommendation:** The uncertainty around OpenCode Zen's funding model (Finding 1) combined with OpenCode Black's obvious unsustainability (Finding 2, 5) makes sticking with our current direct API + local OpenCode server the safest path. We avoid vendor lock-in risk while maintaining flexibility to adopt OpenCode Zen later if they prove genuine cooperative economics.

---

### Implementation Details

**What to implement first:**
- Nothing - maintain status quo architecture
- Add monitoring for OpenCode Zen financial transparency signals (periodic web search or RSS feed)
- Document this decision in orch-go's CLAUDE.md as rationale for not adopting Zen/Black

**Things to watch out for:**
- ⚠️ If OpenCode Black gets blocked by Anthropic (validates our "temporary" assessment), use as teaching moment
- ⚠️ If OpenCode Zen publishes financials showing genuine cooperative model, reassess recommendation
- ⚠️ If our direct API costs spike unexpectedly, consider OpenCode Zen or OpenRouter as mitigation

**Areas needing further investigation:**
- OpenCode Zen's actual launch date and operational history (how long has "breakeven" been sustained?)
- Comparative pricing analysis: direct API vs. OpenCode Zen vs. OpenRouter for our actual usage patterns
- Whether other AI gateway providers (OpenRouter, etc.) offer similar cooperative/breakeven models

**Success criteria:**
- ✅ Current orch-go architecture continues functioning without dependency on OpenCode Zen/Black
- ✅ If OpenCode Black disappears (predicted), we're unaffected
- ✅ If OpenCode Zen proves sustainable and transparent, we can adopt later without having locked into wrong choice now

---

## References

**Files Examined:**
- `.kb/quick/entries.jsonl` (line 554: kb-b764f4) - Found initial decision to track OpenCode Black as industry drama
- Local grep search results - Confirmed no other references to "zen" or "black" in orch-go codebase

**Commands Run:**
```bash
# Search codebase for existing references to Zen/Black
grep -ri "zen\|black" /Users/dylanconlin/Documents/personal/orch-go

# Create investigation file
kb create investigation "research/opencode-zen-black-architecture-economics"
```

**External Documentation:**

**OpenCode Official:**
- [OpenCode Zen landing page](https://opencode.ai/zen) - Product overview and pricing
- [OpenCode Zen documentation](https://opencode.ai/docs/zen/) - Technical architecture and integration details

**News & Analysis:**
- [VentureBeat: Anthropic cracks down on unauthorized Claude usage](https://venturebeat.com/technology/anthropic-cracks-down-on-unauthorized-claude-usage-by-third-party-harnesses) - Primary reporting on Jan 9 crackdown
- [Paddo.dev: Anthropic's Walled Garden](https://paddo.dev/blog/anthropic-walled-garden-crackdown/) - Detailed timeline and technical analysis
- [Hacker News discussion (46549823)](https://news.ycombinator.com/item?id=46549823) - Community reactions and economic analysis

**Community Sources:**
- [ByteIota: Anthropic Blocks Claude Max in OpenCode](https://byteiota.com/anthropic-blocks-claude-max-in-opencode-devs-cancel-200-month-plans/) - Developer cancellation reporting
- Multiple Medium articles and developer blogs on the controversy

**Related Artifacts:**
- **Decision:** `.kb/quick/entries.jsonl` (kb-b764f4) - "Track OpenCode Black as industry drama, not strategic option"
- **Workspace:** `.orch/workspace/og-research-investigate-opencode-zen-13jan-fd44/` - This research workspace

---

## Investigation History

**2026-01-13 16:00:** Investigation started
- Initial question: Investigate OpenCode Zen/Black architecture, economics, and sustainability
- Context: kb quick entry (kb-b764f4) flagged OpenCode Black as "industry drama" - needed deeper research to validate

**2026-01-13 16:15:** Found OpenCode Zen is cooperative buying pool, not traditional vendor
- Discovery: Breakeven model with volume pooling contradicts typical AI gateway markup strategy
- Significance: Changes sustainability assessment - depends on funding transparency

**2026-01-13 16:30:** Confirmed OpenCode Black is emergency response, not planned product
- Timeline validated: Jan 6 normal → Jan 9 Anthropic block → within hours OpenCode Black launched
- Significance: Reinforces kb-b764f4 decision that this is temporary arbitrage, not strategic

**2026-01-13 16:45:** Corrected kb quick entry inaccuracy - no C&D letter exists
- Finding: kb-b764f4 mentioned "Anthropic C&D" but extensive web search found zero evidence
- Reality: Technical enforcement (OAuth fingerprinting), not legal action
- Significance: Changes risk profile - technical blocks can be circumvented, legal threats cannot

**2026-01-13 17:00:** Investigation completed
- Status: Complete
- Key outcome: OpenCode Zen is potentially viable cooperative model (pending transparency), OpenCode Black is confirmed temporary drama (not strategic option)
