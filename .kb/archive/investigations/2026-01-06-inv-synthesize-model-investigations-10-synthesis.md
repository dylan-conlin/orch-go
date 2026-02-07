## Summary (D.E.K.N.)

**Delta:** Synthesized 10 model investigations (Dec 20, 2025 - Jan 4, 2026) into authoritative guide at `.kb/guides/model-selection.md`.

**Evidence:** Read all 10 investigations, identified 5 key themes: (1) architecture split, (2) default model evolution, (3) spawn mode consistency, (4) multi-provider patterns, (5) cost/arbitrage strategies.

**Knowledge:** Model selection is now mature: Opus default, consistent across spawn modes, multi-provider via aliases. Guide consolidates scattered knowledge into single reference.

**Next:** Close - guide created, investigations can be archived.

---

# Investigation: Synthesize Model Investigations Into Guide

**Question:** What patterns emerge from 10 model-related investigations and how should they be consolidated into an authoritative guide?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** feature-impl agent (og-feat-synthesize-model-investigations-06jan-73be)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Architecture Split Is Clear and Correct

**Evidence:** Across investigations, consistent pattern emerged:
- pkg/model: Alias resolution (user convenience)
- pkg/account: Claude Max OAuth management (Anthropic-specific complexity)
- OpenCode: Runtime auth via auth.json handoff

**Source:** 
- 2025-12-24-inv-model-provider-architecture-orch-vs.md (Finding 1-4)
- 2025-12-20-inv-investigate-model-flexibility-arbitrage-orch.md (Finding 3)

**Significance:** The current architecture is sound. Multi-provider expansion only requires adding aliases to pkg/model - no fundamental changes needed.

---

### Finding 2: Default Model Evolution (Gemini → Opus)

**Evidence:** Three investigations documented confusion caused by Gemini 3 Flash default:
- 2025-12-21: DefaultModel was google/gemini-3-flash-preview
- 2025-12-21: Users expected Opus (per orchestrator guidance)
- Fixed: DefaultModel now claude-opus-4-5-20251101

**Source:**
- 2025-12-21-inv-model-handling-conflicts-between-orch.md (Findings 1, 8, 9)
- 2025-12-23-inv-model-selection-issue-architect-agent.md (Finding 2)

**Significance:** Historical context explains why some users saw unexpected models. The fix aligns with orchestrator skill guidance that recommends Opus for complex work.

---

### Finding 3: Spawn Mode Consistency Required Multiple Fixes

**Evidence:** Model passing was inconsistent across spawn modes:
- Tmux: Always worked (BuildOpencodeAttachCommand passed --model)
- Inline: Missing (BuildSpawnCommand didn't pass --model) - Fixed Dec 21
- Headless: Missing (CreateSessionRequest lacked model field) - Fixed Dec 22

**Source:**
- 2025-12-21-inv-fix-buildspawncommand-pass-model-flag.md (all findings)
- 2025-12-22-inv-model-flexibility-phase-expand-model.md (Findings 1-3)
- 2025-12-23-inv-model-selection-issue-architect-agent.md (Findings 1-3)

**Significance:** All spawn modes now correctly pass --model flag. This was a significant bug that caused user confusion.

---

### Finding 4: Multi-Provider Pattern Is Additive

**Evidence:** Research investigations documented Gemini, DeepSeek, OpenRouter alternatives:
- API key providers don't need pkg/account complexity
- Just add aliases to pkg/model, OpenCode handles auth
- Three-tier arbitrage strategy (routing/execution/reasoning)

**Source:**
- 2025-12-20-inv-research-gemini-model-arbitrage-alternatives.md (all findings)
- 2025-12-24-inv-model-provider-architecture-orch-vs.md (Findings 3-4)

**Significance:** Future provider expansion is straightforward: add aliases, let OpenCode handle auth.

---

### Finding 5: Cost/Pricing Analysis Provides Decision Framework

**Evidence:** API vs Max comparison established break-even points:
- Max 5x: ~100 turns/day break-even
- Context cliff: Sonnet 2x price at >200K tokens
- Gemini advantage: 1M token context, lower cost for large context

**Source:**
- 2025-12-20-research-model-arbitrage-api-vs-max.md (all findings)

**Significance:** Provides decision framework for when to use which plan/model.

---

## Synthesis

**Key Insights:**

1. **Model selection is now mature** - After Dec 2025 fixes, all spawn modes correctly respect --model flag, default is Opus, and architecture is clean.

2. **Scattered knowledge problem** - 10 investigations each contained partial understanding. No single place to look up "how does model selection work?"

3. **Multi-provider is future-ready** - Architecture supports adding Gemini, DeepSeek, OpenRouter by just adding aliases.

4. **Cost optimization documented** - API vs Max, context limits, arbitrage tiers all captured but not consolidated.

5. **Bug fixes are complete** - BuildSpawnCommand, CreateSessionRequest, DefaultModel all fixed. No open issues.

**Answer to Investigation Question:**

The 10 investigations reveal a maturation arc: initial implementation (Dec 20) → bug discovery (Dec 21-23) → fixes and testing (Dec 22-24) → architecture clarity (Dec 24) → completion (Jan 4).

Key patterns consolidated into guide:
- Architecture: pkg/model + pkg/account + OpenCode auth.json
- Defaults: Opus for quality, Flash for cost/context
- Spawn modes: All three now consistent
- Multi-provider: Alias-based, additive
- Cost: Break-even analysis, context cliffs

Guide created at `.kb/guides/model-selection.md` as single authoritative reference.

---

## Structured Uncertainty

**What's tested:**

- ✅ Guide structure follows existing guides format (verified: compared to spawn.md)
- ✅ All 10 investigations read and patterns extracted
- ✅ Key findings from each investigation represented in guide

**What's untested:**

- ⚠️ Guide completeness for new users (no user testing)
- ⚠️ Whether guide gets discovered by agents (depends on kb context queries)
- ⚠️ Pricing data accuracy (Dec 2025 data, may be stale)

**What would change this:**

- New model selection bugs discovered → update guide
- Pricing changes → update cost section
- New providers added → add to multi-provider section

---

## Implementation Recommendations

### Recommended Approach ⭐

**Guide Created** - Single authoritative reference at `.kb/guides/model-selection.md`.

**Why this approach:**
- Follows kb decision that guides consolidate 10+ investigations
- Single reference instead of reading 10 separate investigations
- Standard location makes it discoverable

**Trade-offs accepted:**
- Some investigation detail lost in consolidation
- Original investigations preserved but now secondary sources

**Implementation sequence:**
1. ✅ Read all investigations
2. ✅ Identify patterns
3. ✅ Create guide
4. Consider archiving older investigations (orchestrator decision)

---

### Implementation Details

**What to implement first:**
- Guide is complete, ready for use

**Things to watch out for:**
- ⚠️ Keep guide updated as model selection evolves
- ⚠️ Pricing section will become stale (consider adding "last verified" date)
- ⚠️ Original investigations remain in .kb/investigations/ (not deleted)

**Areas needing further investigation:**
- Whether to archive the 10 source investigations
- Whether guide should be linked from orchestrator skill

**Success criteria:**
- ✅ Guide exists at .kb/guides/model-selection.md
- ✅ Covers all 5 identified themes
- ✅ Follows existing guide format
- ✅ `kb context "model"` returns this guide

---

## References

**Files Examined:**
- 10 investigation files listed in spawn context
- .kb/guides/spawn.md - guide format reference

**Commands Run:**
```bash
# Create investigation file
kb create investigation synthesize-model-investigations-10-synthesis

# Report progress
bd comments add orch-go-bhlbe "Phase: ..."
```

**Related Artifacts:**
- **Guide Created:** .kb/guides/model-selection.md - The deliverable
- **Source Investigations:** All 10 listed in spawn context

---

## Investigation History

**2026-01-06:** Investigation started
- Initial question: What patterns emerge from 10 model investigations?
- Context: kb reflect triggered synthesis (10+ investigations on "model" topic)

**2026-01-06:** Read all investigations
- Identified 5 themes across 10 investigations
- Found maturation arc from initial impl through bug fixes to stability

**2026-01-06:** Guide created
- Consolidated findings into .kb/guides/model-selection.md
- Followed existing guide format from spawn.md

**2026-01-06:** Investigation completed
- Status: Complete
- Key outcome: Authoritative model selection guide created from 10 scattered investigations
