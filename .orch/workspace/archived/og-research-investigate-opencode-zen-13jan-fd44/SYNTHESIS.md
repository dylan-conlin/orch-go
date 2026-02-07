# Session Synthesis

**Agent:** og-research-investigate-opencode-zen-13jan-fd44
**Issue:** orch-go-3wgmo
**Duration:** 2026-01-13 16:00 → 2026-01-13 17:15
**Outcome:** success

---

## TLDR

Researched OpenCode Zen and OpenCode Black to validate kb quick decision (kb-b764f4). Found: OpenCode Zen is a potentially viable cooperative buying pool with breakeven economics (pending financial transparency), while OpenCode Black is a confirmed temporary emergency response to Anthropic's Jan 9, 2026 OAuth crackdown - not a sustainable product. Recommendation: maintain status quo for orch-go (direct API + local OpenCode server).

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-13-research-opencode-zen-black-architecture-economics.md` - Complete research investigation with 6 findings, synthesis, recommendations

### Files Modified
- None (pure research, no code changes)

### Commits
- `7d69481b` - research: OpenCode Zen/Black architecture, economics, sustainability analysis

---

## Evidence (What Was Observed)

### OpenCode Zen Findings (from official docs and web research):
- Operates explicitly as "not a for-profit thing" at breakeven
- Volume pooling model: users pool demand → negotiate bulk discounts → savings passed through
- Zero markup pricing (only 4.4% + $0.30 processing fees)
- 26+ models across OpenAI, Anthropic, Google, Asian providers
- Standard AI gateway architecture (unremarkable technically)
- Claims to offer "the cheapest possible rates that exist out there right now"

### OpenCode Black Findings (from news sources and community):
- Timeline: Jan 6, 2026 (normal) → Jan 9, 2026 02:20 UTC (Anthropic block) → within hours (Black launched)
- $200/month tier routing through "enterprise API gateway" to bypass consumer OAuth restrictions
- No transparency on refund policies or service guarantees
- Community characterized it as rapid countermeasure, not strategic product

### Anthropic Crackdown Details:
- Economic arbitrage was 5-10x: Claude Max $200/month vs $1,000+ API equivalent for power users
- Third-party tools enabled "autonomous agent loops" (overnight processes) only viable on flat-rate plans
- Anthropic's enforcement was technical (OAuth fingerprinting), NOT legal cease & desist (zero evidence found)
- Community response: mass cancellations within hours, 147+ GitHub reactions, 245+ Hacker News points

### Key Correction:
- kb quick entry kb-b764f4 mentioned "Anthropic C&D" but extensive web research found zero evidence of cease & desist letter
- Reality: Technical enforcement only (API restrictions, client identity verification)
- This matters: technical blocks can be circumvented (hence OpenCode Black), legal threats cannot

### Tests Run
```bash
# Web searches for OpenCode Zen/Black information
WebSearch: "OpenCode Zen Black architecture 2026"
WebSearch: "OpenCode Black Anthropic cease and desist January 2026"
WebSearch: "OpenCode Zen pricing economics sustainability"
WebSearch: "OpenCode Zen breakeven cost model sustainability nonprofit volume pooling"
WebSearch: "OpenCode Black refund policy transparency pricing $200 tier January 2026"
WebSearch: ""OpenCode Black" controversy community response developer reactions January 2026"

# Web fetches from official docs
WebFetch: https://opencode.ai/docs/zen/
WebFetch: https://opencode.ai/zen
WebFetch: https://paddo.dev/blog/anthropic-walled-garden-crackdown/
WebFetch: https://news.ycombinator.com/item?id=46549823

# Local codebase search
grep -ri "zen\|black" /Users/dylanconlin/Documents/personal/orch-go
# Result: Only found kb-b764f4 entry, no other references
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-13-research-opencode-zen-black-architecture-economics.md` - Full research investigation with 6 findings, synthesis, uncertainty assessment, and recommendations

### Decisions Made
- **Decision 1:** Maintain status quo (direct Anthropic API + local OpenCode server) - because OpenCode Zen sustainability unproven and Black is temporary
- **Decision 2:** Track OpenCode Zen for financial transparency signals - if they publish financials showing genuine cooperative economics, reassess
- **Decision 3:** Monitor for OpenCode Black shutdown - validates "temporary" assessment when it happens

### Constraints Discovered
- **OpenCode Zen sustainability depends on funding transparency** - Breakeven claim cannot be validated without financial disclosure. Could be genuine cooperative or venture-subsidized loss-leading.
- **Economic arbitrage made Anthropic crackdown inevitable** - 5-10x price difference (consumer subscriptions vs API pricing) created unsustainable subsidy of competitor products
- **Community loyalty was to pricing arbitrage, not Claude** - Mass cancellations when arbitrage disappeared reveals shallow moat for vendor lock-in
- **Technical enforcement can be circumvented but legal cannot** - kb quick entry incorrectly mentioned C&D letter; reality is technical blocks (OAuth) which can be worked around (hence OpenCode Black)

### Key Insights
1. **OpenCode Zen and Black are fundamentally different strategies** - Zen (cooperative buying pool) has potential strategic merit; Black (emergency patch) is tactical desperation
2. **The breakeven model's viability is unknowable** - Without financial transparency, can't distinguish genuine cooperative economics from venture-subsidized market share grab
3. **OpenCode Black has no long-term viability** - Cat-and-mouse game with Anthropic; when next block happens, service collapses
4. **Architecturally unremarkable, economically differentiated** - Zen is standard AI gateway + curated models; only moat is pricing model (if genuine)

### Externalized via `kb quick`
- None directly - this research validates existing kb-b764f4 decision, doesn't create new constraints/decisions

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file created and committed)
- [x] Investigation file has `**Phase:** Complete`
- [x] SYNTHESIS.md created in workspace
- [x] Ready for `orch complete orch-go-3wgmo`

**For orchestrator:**
- kb quick entry kb-b764f4 is validated by this research
- Consider correcting kb-b764f4 to note "no actual C&D letter - technical enforcement only"
- Monitor for OpenCode Black shutdown (predicted) as validation of research findings
- Monitor for OpenCode Zen financial transparency (if published, reassess recommendation)

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- What is OpenCode Zen's actual launch date and operational history? (How long has "breakeven" been sustained?)
- Does OpenCode have venture funding, and if so, how much? (Would contradict "breakeven" positioning)
- What exactly is OpenCode Black's "enterprise API gateway" architecture? (Vague description, could be multiple things)
- Do other AI gateway providers (OpenRouter, etc.) offer similar cooperative/breakeven models? (Comparative analysis)
- What happens to OpenCode Black users if service shuts down? (No public refund policy found)

**Areas worth exploring further:**
- Comparative pricing analysis: direct API vs. OpenCode Zen vs. OpenRouter for actual orch-go usage patterns
- OpenCode Zen's provider contracts - are the bulk discounts real or marketing claims?
- When will Anthropic block OpenCode Black? (Predicted cat-and-mouse next move)

**What remains unclear:**
- OpenCode Zen's funding model (cooperative vs. VC-subsidized) - critical for sustainability assessment
- OpenCode Black's refund policy and user protections
- Timeline for OpenCode Black shutdown (when, not if)

---

## Session Metadata

**Skill:** research
**Model:** google/gemini-2.5-flash-preview
**Workspace:** `.orch/workspace/og-research-investigate-opencode-zen-13jan-fd44/`
**Investigation:** `.kb/investigations/2026-01-13-research-opencode-zen-black-architecture-economics.md`
**Beads:** `bd show orch-go-3wgmo`
