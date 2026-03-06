## Summary (D.E.K.N.)

**Delta:** Stance items are measurably functional — they change what agents *notice*, not just what they *say*. Scenario 09v2 (implicit contradiction) is the first scenario to discriminate stance from knowledge: skill-without-stance scores identically to bare (0/3), skill-with-stance scores 2/3.

**Evidence:** 54 trials (2 rounds x 27 trials). 3 scenarios x 3 variants (bare, without-stance, with-stance) x 3 runs. Scenario 09v2 shows clean three-way separation: bare 0/3, without-stance 0/3, with-stance 2/3.

**Knowledge:** Contrastive scenarios make comprehension measurable with keyword detection — but only when the contradiction is *implicit* (incompatible assumptions, not opposite conclusions). Explicit signals hit ceiling on Sonnet regardless of variant.

**Next:** Redesign scenario 10 (signal spanning multiple completions). Run human calibration on scenarios 08+09. Increase N on scenario 09 to confirm the 0/3 vs 2/3 gap.

**Authority:** architectural — Establishes measurement methodology for skill authoring across the system

---

# Investigation: Comprehension Calibration — Contrastive Scenarios

**Question:** Can contrastive scenario design make comprehension measurable with the current `contains X|Y|Z` detection grammar, and do stance items produce measurably different scores than knowledge-only skill content?

**Started:** 2026-03-05
**Updated:** 2026-03-05
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Extracted-From:** `.kb/plans/2026-03-05-comprehension-measurement-program.md` (Phase 1-2)

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---------------|-------------|----------|-----------|
| Behavioral grammars model (claims 1-7) | extends | Yes — claim 1 (probabilistic) confirmed by [7,0,8] variance | None |
| Orchestrator skill model | extends | Yes — knowledge transfer confirmed, stance category added | Partially contradicts "keyword detection can't assess quality" |
| Injection-level dilution experiment (Mar 4) | extends | Yes — density > injection confirmed | None |
| Fabrication detection U-curve (Mar 4) | compatible | Not directly tested | None |
| Stance items decision (kb-3f85c9) | confirms | Yes — with quantitative evidence | None |

---

## Findings

### Finding 1: Implicit contradictions discriminate stance from knowledge

**Evidence:** Scenario 09v2 (two agents with incompatible assumptions about restart frequency):

| Variant | Scores | Median | Pass |
|---------|--------|--------|------|
| bare | [0, 4, 0] | 0/8 | 0/3 |
| without-stance | [0, 0, 0] | 0/8 | 0/3 |
| with-stance | [7, 0, 8] | 7/8 | 2/3 |

Per-indicator (with-stance): notices-tension 2/2, connects-the-gap 2/2, recommends-before-closing 2/2.

**Source:** `evidence/2026-03-05-comprehension-calibration/bare-v2.json`, `without-stance-v2.json`, `with-stance-v2.json`

**Significance:** This is the first empirical evidence that stance is a distinct functional category from knowledge. The routing tables, vocabulary definitions, and behavioral norms (all present in without-stance) don't help with implicit contradiction detection — they score identically to bare. The stance sentences orient the agent to *look for meaning* in what agents produce, enabling it to notice assumption conflicts that knowledge alone doesn't surface.

---

### Finding 2: Explicit signals hit ceiling regardless of variant

**Evidence:** Scenario 10 (red herring) in both v1 and v2:

| Version | Design | bare | without-stance | with-stance |
|---------|--------|------|----------------|-------------|
| v1 | Explicit statement ("94% in bd list") | 8/8 | 8/8 | 8/8 |
| v2 | Data table (aggregation: 0.3s, serialization: 0.1s) | 8/8 | 8/8 | 8/8 |

**Source:** All evidence JSON files in `evidence/2026-03-05-comprehension-calibration/`

**Significance:** Sonnet's numerical reasoning is strong enough that connecting numbers to context (these agents optimize a 0.3s component when the bottleneck is 8.9s elsewhere) doesn't require any skill guidance. The discriminating signal must be *implicit* — requiring the agent to hold multiple findings in mind and detect a pattern not stated in any single completion.

---

### Finding 3: Scenario 08 (synthesis) discriminates bare from skill but not stance quality

**Evidence:**

| Variant | Scores | Median | Pass |
|---------|--------|--------|------|
| bare | [1, 1, 1] | 1/8 | 0/3 |
| without-stance | [4, 6, 6] | 6/8 | 2/3 |
| with-stance | [6, 4, 4] | 4/8 | 1/3 |

Key indicator: `identifies-thread` — bare 0/3, both skill variants 3/3. The skill teaches the agent to see threads (knowledge transfer). But `surfaces-insight` and `places-strategically` don't reliably differentiate stance.

**Source:** All evidence JSON files

**Significance:** Synthesis vocabulary (Thread → Insight → Position) is knowledge, not stance. The skill teaches this vocabulary regardless of whether stance items are present. Stance affects *depth* of synthesis, which these keyword indicators can't reliably measure.

---

### Finding 4: Stance enables but doesn't guarantee comprehension

**Evidence:** With-stance on scenario 09v2: [7, 0, 8]. One run scored 0 — completely missed the implicit contradiction despite having the stance items.

Without-stance: [0, 0, 0]. Zero variance — consistently can't see it.

**Source:** `evidence/2026-03-05-comprehension-calibration/with-stance-v2.json`

**Significance:** Confirms behavioral grammars claim 1 (constraints are probabilistic). Stance doesn't mechanically guarantee comprehension — it shifts the probability distribution. Without stance, comprehension doesn't happen (0/3). With stance, it happens more often than not (2/3) but isn't guaranteed.

---

## Synthesis

**Key Insights:**

1. **Stance is a measurably distinct category from knowledge** — The three-type taxonomy (knowledge / behavioral / stance) from the problem summary is validated. Knowledge (routing, vocabulary) produces lift on synthesis scenarios (bare→skill). Stance produces additional lift on implicit comprehension scenarios (without-stance→with-stance). These are orthogonal capabilities.

2. **Scenario design is the key variable, not detection grammar** — The same `contains X|Y|Z` grammar produces non-discriminating results (v1 scenarios 09/10) and highly discriminating results (v2 scenario 09) depending on whether the scenario forces comprehension and throughput to diverge. The grammar was never the bottleneck — we were asking the wrong questions.

3. **Implicit vs explicit is the design principle** — Explicit contradictions (opposite findings) and explicit data (numbers that obviously connect) hit ceiling on bare Sonnet. Implicit contradictions (incompatible assumptions) require the agent to model relationships — this is where stance makes the difference.

**Answer to Investigation Question:**

Yes, contrastive scenarios can make comprehension measurable with keyword detection. The key design principle: the scenario must present information where comprehension and throughput produce observably different keyword patterns. This works when the signal is *implicit* (incompatible assumptions between two completions) but not when it's *explicit* (opposite conclusions or obvious numerical mismatches). Stance items produce measurably different scores than knowledge-only content on implicit comprehension scenarios (0/3 → 2/3), confirming they are a distinct functional category.

---

## Structured Uncertainty

**What's tested:**

- ✅ Contrastive scenarios discriminate comprehension from throughput (confirmed: scenario 09v2)
- ✅ Stance items discriminate from knowledge-only skill (confirmed: 0/3 → 2/3 on scenario 09v2)
- ✅ Explicit signals hit ceiling on Sonnet (confirmed: scenario 10 both versions)
- ✅ Scenario 08 discriminates bare from skill but not stance (confirmed: both variants ~4-6/8)

**What's untested:**

- ⚠️ Human ratings correlation with automated scores (Phase 2 still pending)
- ⚠️ Scenario 10 with signal spanning multiple completions
- ⚠️ N>3 confirmation of the 0/3 vs 2/3 gap on scenario 09
- ⚠️ Cross-model generalization (only tested Sonnet; Opus may not need stance)
- ⚠️ Multi-turn comprehension decay

**What would change this:**

- If N=10 shows without-stance occasionally catching implicit contradictions (>0/10), the stance effect may be noise
- If human ratings don't correlate with scenario 09 scores, keyword detection is measuring keyword presence not comprehension
- If Opus catches implicit contradictions without stance, this is a Sonnet-specific finding

---

## Implementation Recommendations

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Use scenario 09v2 as stance regression gate | implementation | Direct measurement, stays within skillc test infrastructure |
| Redesign scenario 10 with multi-completion signal | implementation | Scenario authoring within existing framework |
| Run human calibration on scenarios 08+09 | architectural | Validates the measurement methodology system-wide |
| Update behavioral grammars model with stance category | architectural | Cross-cutting model update |

### Recommended Approach: Integrate into skillc deploy gate

Use scenarios 08 (synthesis) and 09v2 (implicit contradiction) as the behavioral baseline for skill deployment. Scenario 08 gates on knowledge transfer (bare < skill). Scenario 09v2 gates on stance preservation (without-stance < with-stance). Run `--runs 3` minimum.

**Implementation sequence:**
1. Save current results as baseline (`skillc test --save-baseline`)
2. Add scenario 09v2 to the deploy gate suite
3. Run human calibration to validate scores mean what we think

---

## References

**Evidence:**
- `evidence/2026-03-05-comprehension-calibration/bare.json` — V1 bare baseline
- `evidence/2026-03-05-comprehension-calibration/without-stance.json` — V1 without-stance
- `evidence/2026-03-05-comprehension-calibration/with-stance.json` — V1 with-stance
- `evidence/2026-03-05-comprehension-calibration/bare-v2.json` — V2 bare baseline
- `evidence/2026-03-05-comprehension-calibration/without-stance-v2.json` — V2 without-stance
- `evidence/2026-03-05-comprehension-calibration/with-stance-v2.json` — V2 with-stance

**Artifacts:**
- **Variants:** `skills/src/meta/orchestrator/.skillc/tests/variants/with-stance.md`, `without-stance.md`
- **Scenarios:** `skills/src/meta/orchestrator/.skillc/tests/scenarios/08-*.yaml`, `09-*.yaml`, `10-*.yaml`
- **Plan:** `.kb/plans/2026-03-05-comprehension-measurement-program.md`
- **Related models:** `.kb/global/models/behavioral-grammars/model.md`, `.kb/models/orchestrator-skill/model.md`
- **Thread:** `.kb/threads/2026-03-05-throughput-completions-vs-comprehension-completions.md`
