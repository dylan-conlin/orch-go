## Summary (D.E.K.N.)

**Delta:** Stance does NOT generalize to all attention types. Of 3 new scenarios testing absence, relationship-tracing, and information-freshness, NONE showed three-way separation (bare < without-stance < with-stance). Scenario 12 (downstream consumer) showed strong knowledge lift (bare 17% → skill 83%) but no stance advantage. Scenario 11 (auth gap) hit ceiling on all variants at 3/8. Scenario 13 (stale deprecation) showed knowledge lift but with-stance scored WORSE than without-stance.

**Evidence:** 144 total trials. Initial round: 54 (N=3 x 3 scenarios x 3 variants x 2 rounds). Higher-N: 36 (N=6 x 2 scenarios x 3 variants for 09+10). Generalization: 54 (N=6 x 3 scenarios x 3 variants for 11-13). Scenario 09 stance gap confirmed: 83% vs 17% vs 0%. Scenarios 11-13: 0 of 3 show stance advantage.

**Knowledge:** Stance as attention priming is scenario-09-specific, not a general mechanism. It works for implicit contradiction (two agents with incompatible assumptions) but not for: (1) absence detection — Sonnet already notices missing auth without stance, (2) relationship tracing — knowledge provides the lift, not stance orientation, (3) information freshness — temporal reasoning is hard and stance doesn't help. The distinction may be that scenario 09 requires *holding two things in mind simultaneously*, while the other attention types require different cognitive operations.

**Next:** Revise skill-content-transfer model to narrow stance claims. Scenario 09 remains the primary stance discriminator. Investigate what makes implicit contradiction uniquely responsive to stance. Consider whether scenarios 11/13 have indicator design issues vs genuine stance-irrelevance.

**Authority:** architectural — Establishes measurement methodology for skill authoring across the system

---

# Investigation: Comprehension Calibration — Contrastive Scenarios

**Question:** Can contrastive scenario design make comprehension measurable with the current `contains X|Y|Z` detection grammar, and do stance items produce measurably different scores than knowledge-only skill content?

**Started:** 2026-03-05
**Updated:** 2026-03-06 (generalization round: scenarios 11-13, 54 trials)
**Phase:** Complete
**Next Step:** Narrow stance claims in skill-content-transfer model; investigate scenario 11/13 indicator design; human calibration on 08+09
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

### Finding 5: N=6 confirms stance gap on scenario 09, refutes scenario 10 redesign

**Evidence:** Higher-N replication (6 trials per variant, 36 total):

Scenario 09 (contradiction detection):

| Variant | Scores | Median | Pass Rate |
|---------|--------|--------|-----------|
| bare | [4, 0, 1, 0, 0, 0] | 0/8 | 0/6 (0%) |
| without-stance | [1, 4, 1, 4, 1, 7] | 2.5/8 | 1/6 (17%) |
| with-stance | [7, 7, 7, 7, 1, 7] | 7/8 | 5/6 (83%) |

Scenario 10 (distributed-symptom-pattern, redesigned):

| Variant | Scores | Median | Pass Rate |
|---------|--------|--------|-----------|
| bare | [1, 4, 4, 1, 4, 4] | 4/8 | 0/6 (0%) |
| without-stance | [1, 4, 4, 5, 7, 4] | 4/8 | 2/6 (33%) |
| with-stance | [1, 4, 1, 1, 1, 4] | 1/8 | 0/6 (0%) |

**Source:** `evidence/2026-03-05-higher-n-09-10/bare.json`, `without-stance.json`, `with-stance.json`

**Significance:**

Scenario 09: The stance gap is *stronger* at N=6 than N=3. With-stance pass rate went from 67% (2/3) to 83% (5/6). Without-stance is not identical to bare at higher N (1/6 vs 0/6), but the gap between without-stance and with-stance is massive (17% → 83%). This decisively confirms stance as a functional category.

Scenario 10: The redesigned distributed-symptom-pattern scenario shows NO stance advantage — with-stance actually scores worse (median 1 vs 4). This scenario tests cross-completion pattern recognition (3 timing fixes masking a daemon perf issue). The signal is distributed across 3 agents rather than implicit between 2 — this is harder and neither knowledge nor stance helps. The scenario may need simpler indicators or a less demanding comprehension target.

Combined N=3+N=6 for scenario 09: bare 0/9 (0%), without-stance 1/9 (11%), with-stance 7/9 (78%).

---

### Finding 6: Stance does NOT generalize to absence, relationship-tracing, or freshness attention types

**Evidence:** N=6 contrastive trials across 3 new scenarios (54 total runs):

Scenario 11 (absence-as-evidence / auth gap):

| Variant | Scores | Median | Pass Rate |
|---------|--------|--------|-----------|
| bare | [3, 3, 4, 3, 3, 3] | 3/8 | 0/6 (0%) |
| without-stance | [3, 4, 1, 3, 3, 3] | 3/8 | 0/6 (0%) |
| with-stance | [3, 3, 3, 4, 3, 3] | 3/8 | 0/6 (0%) |

Per-indicator: `notices-auth-gap` 6/6 on bare (already caught), `identifies-mechanism` 0/6 on ALL variants. The auth gap is too obvious for the "notice" indicator but too implicit for the "mechanism" indicator. Sonnet always says "there's a security issue" but never pinpoints "registered on wrong group." Neither knowledge nor stance helps with this specific gap.

Scenario 12 (downstream-consumer-contract):

| Variant | Scores | Median | Pass Rate |
|---------|--------|--------|-----------|
| bare | [0, 3, 0, 3, 6, 0] | 1/8 | 1/6 (17%) |
| without-stance | [6, 3, 7, 7, 7, 6] | 6/8 | 5/6 (83%) |
| with-stance | [6, 6, 7, 6, 3, 7] | 6/8 | 5/6 (83%) |

Per-indicator: Knowledge drives the entire lift. `notices-consumer-impact` goes from variable (bare) to 6/6 (both skill variants). `connects-volume-change` goes from ~50% (bare) to ~83% (both). With-stance adds zero incremental value over without-stance.

Scenario 13 (stale-deprecation-claim):

| Variant | Scores | Median | Pass Rate |
|---------|--------|--------|-----------|
| bare | [4, 1, 0, 0, 1, 4] | 1/8 | 0/6 (0%) |
| without-stance | [4, 3, 7, 4, 4, 7] | 4/8 | 2/6 (33%) |
| with-stance | [4, 4, 1, 4, 4, 4] | 4/8 | 0/6 (0%) |

Per-indicator: `notices-stale-claim` 0/6 for bare AND with-stance, 2/6 for without-stance. The stance ("Information decays... verify against current evidence") paradoxically performs WORSE than knowledge-only. `connects-git-evidence` and `recommends-verification` are strong across skill variants but the crucial stale-claim detection doesn't benefit from stance.

**Source:** `evidence/2026-03-05-contrastive-11-13-N6/scenario-{11,12,13}/{bare,without-stance,with-stance}.json`

**Significance:**

The three-way separation (bare < without-stance < with-stance) does NOT replicate outside scenario 09. Key findings:

1. **Scenario 11:** All variants hit the same ceiling (3/8). The auth gap is simultaneously too obvious (Sonnet catches it bare) and too specific (detection patterns don't match how Sonnet describes the mechanism). This is likely an indicator design issue rather than a genuine stance test.

2. **Scenario 12:** Knowledge provides massive lift (17% → 83%), but stance adds nothing. The information about AgentGrid's layout constraints and volume characteristics is sufficient — the agent just needs to KNOW about the consumer, not be oriented to look for consumers. This contradicts the hypothesis that relationship-tracing requires stance.

3. **Scenario 13:** With-stance scores WORSE than without-stance (0/6 vs 2/6 pass). The `notices-stale-claim` indicator (which detects "stale|outdated|old comment|months ago") fires for without-stance but not with-stance. Possible explanation: the stance framing ("information decays") may prime the agent to recommend verification as a general practice without explicitly calling the deprecation "stale" — the agent verifies but uses different vocabulary than the detector expects.

**Implication for skill-content-transfer model:** Stance's measured effect is specific to implicit contradiction detection (scenario 09), not a general attention-priming mechanism. The model's claim that "stance is the highest-leverage content type" needs narrowing: stance is high-leverage for *holding two competing models simultaneously*, not for all attention types.

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
- ✅ N=6 confirms scenario 09 stance gap: 0% → 17% → 83% pass rate (bare → without-stance → with-stance)
- ✅ Scenario 10 redesigned as distributed-symptom-pattern: too hard for all variants, no stance advantage
- ✅ Scenarios 11-13 generalization: stance does NOT generalize beyond implicit contradiction
- ⚠️ Whether scenario 11/13 indicator design is masking real stance effects (agents may use different vocabulary)
- ⚠️ Cross-model generalization (only tested Sonnet; Opus may not need stance)
- ⚠️ Multi-turn comprehension decay

**What would change this:**

- ~~If N=10 shows without-stance occasionally catching implicit contradictions (>0/10), the stance effect may be noise~~ → N=6 shows without-stance at 1/6 (17%) vs with-stance 5/6 (83%). Gap is real, not noise.
- ~~If stance generalizes to absence/relationship/freshness, it's a universal attention mechanism~~ → N=6 on scenarios 11-13 shows NO generalization. Stance is specific to implicit contradiction.
- If human ratings don't correlate with scenario 09 scores, keyword detection is measuring keyword presence not comprehension
- If Opus catches implicit contradictions without stance, this is a Sonnet-specific finding
- If scenario 11/13 with revised indicators show stance effects, the current indicators are masking real effects

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
- `evidence/2026-03-05-comprehension-calibration/bare.json` — V1 bare baseline (N=3)
- `evidence/2026-03-05-comprehension-calibration/without-stance.json` — V1 without-stance (N=3)
- `evidence/2026-03-05-comprehension-calibration/with-stance.json` — V1 with-stance (N=3)
- `evidence/2026-03-05-comprehension-calibration/bare-v2.json` — V2 bare baseline (N=3)
- `evidence/2026-03-05-comprehension-calibration/without-stance-v2.json` — V2 without-stance (N=3)
- `evidence/2026-03-05-comprehension-calibration/with-stance-v2.json` — V2 with-stance (N=3)
- `evidence/2026-03-05-higher-n-09-10/bare.json` — Higher-N bare (N=6, scenarios 09+10)
- `evidence/2026-03-05-higher-n-09-10/without-stance.json` — Higher-N without-stance (N=6, scenarios 09+10)
- `evidence/2026-03-05-higher-n-09-10/with-stance.json` — Higher-N with-stance (N=6, scenarios 09+10)
- `evidence/2026-03-05-contrastive-11-13-N6/scenario-11/{bare,without-stance,with-stance}.json` — Scenario 11 generalization (N=6 x 3 variants)
- `evidence/2026-03-05-contrastive-11-13-N6/scenario-12/{bare,without-stance,with-stance}.json` — Scenario 12 generalization (N=6 x 3 variants)
- `evidence/2026-03-05-contrastive-11-13-N6/scenario-13/{bare,without-stance,with-stance}.json` — Scenario 13 generalization (N=6 x 3 variants)

**Artifacts:**
- **Variants:** `skills/src/meta/orchestrator/.skillc/tests/variants/with-stance.md`, `without-stance.md`
- **Scenarios:** `skills/src/meta/orchestrator/.skillc/tests/scenarios/08-*.yaml`, `09-*.yaml`, `10-*.yaml`
- **Plan:** `.kb/plans/2026-03-05-comprehension-measurement-program.md`
- **Related models:** `.kb/global/models/behavioral-grammars/model.md`, `.kb/models/orchestrator-skill/model.md`
- **Thread:** `.kb/threads/2026-03-05-throughput-completions-vs-comprehension-completions.md`
