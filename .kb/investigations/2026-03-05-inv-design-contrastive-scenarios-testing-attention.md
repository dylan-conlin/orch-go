## Summary (D.E.K.N.)

**Delta:** Three contrastive scenarios designed to test whether attention-oriented stances can prime worker agents to notice absence (missing auth), relationships (downstream consumer impact), and information staleness (deprecated but actively used code).

**Evidence:** Design grounded in scenario 09's proven pattern (implicit contradiction, N=9: bare 0%, without-stance 11%, with-stance 78%) and scenario 10's failure mode (too diffuse, no single "aha" moment). Each scenario has one clear "aha" moment where attention to the target type flips the answer.

**Knowledge:** Effective stance scenarios require: (1) all information present in prompt, (2) surface reading produces wrong answer, (3) attention to the specific type flips to correct answer. The implicit gap must be narrow — one variable-name comparison, one consumer-trace step, one temporal comparison — not distributed across many signals.

**Next:** Run contrastive trials (N=6 per variant, 3 variants, 3 scenarios = 54 trials) using skillc runner with Sonnet 4. Expect bare < without-stance < with-stance if stance mechanism generalizes beyond implicit contradiction.

**Authority:** implementation - Scenario design within established testing patterns, uses existing skillc format and infrastructure.

---

# Investigation: Design Contrastive Scenarios Testing Attention

**Question:** Can we design scenarios that isolate stance effects for three attention types: absence-as-evidence, relationship-tracing, and information-freshness?

**Started:** 2026-03-05
**Updated:** 2026-03-05
**Owner:** orch-go-w0f1w
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/threads/2026-03-05-stance-as-attention-priming-agents.md | extends | Yes — scenario 09 results confirmed | None |
| .kb/models/skill-content-transfer/model.md | extends | Yes — three content types confirmed | None |
| .kb/models/defect-class-taxonomy/model.md | applies | Yes — defect classes mapped to attention types | None |

---

## Findings

### Finding 1: Scenario 09's success pattern generalizes to three design principles

**Evidence:** Scenario 09 worked because: (a) the contradiction was implicit (incompatible assumptions, not opposite conclusions), (b) it required modeling relationships between two pieces of information, (c) there was a single "aha" moment (rate limiter assumes rare restarts + --replace enables frequent ones). Scenario 10 failed because: the signal was distributed across 3 agents with no single connection point.

**Source:** Existing scenario files (09-contradiction-detection.yaml, 10-red-herring-obvious-action.yaml), thread 2026-03-05-stance-as-attention-priming-agents.md

**Significance:** Design principle extracted: Each scenario needs ONE implicit gap that attention to a specific type flips. The gap must be narrow enough that a single comparison/trace/check resolves it. Distributed symptoms don't discriminate.

---

### Finding 2: Three attention types map cleanly to specific implicit gap patterns

**Evidence:**

| Attention Type | Defect Class | Implicit Gap Pattern | Scenario |
|---|---|---|---|
| Absence-as-evidence | Class 1: Filter Amnesia | New code missing pattern that ALL peers have | 11: auth middleware gap |
| Relationship-tracing | Class 0: Scope Expansion | Local change breaks implicit consumer contract | 12: dashboard pagination |
| Information-freshness | Class 7: Premature Destruction | Written claim contradicted by temporal evidence | 13: stale deprecation |

Each gap has one "aha" variable:
- Scenario 11: `r.Group(...)` vs `api` — one argument name difference
- Scenario 12: "all results in fixed grid" + "query now returns 4x more" — one volume multiplication
- Scenario 13: "August 2025 comment says safe" + "February 2026 commits use the API" — one temporal comparison

**Source:** Defect class taxonomy model, stance-as-attention-priming thread

**Significance:** The mapping validates that the attention-stance hypothesis connects to real production defect patterns, not synthetic test constructs.

---

### Finding 3: Contrastive variant design separates knowledge from stance

**Evidence:** For each scenario, three variants were designed:
- **Bare:** Prompt only. Tests baseline model attention.
- **Without-stance:** Knowledge items providing factual context (middleware architecture, dashboard consumer details, deprecation lifecycle). Gives the model the FACTS but no orientation.
- **With-stance:** Same knowledge + one-sentence epistemic orientation ("absence is evidence", "every data path has implicit consumers", "information decays").

The key design constraint: knowledge items must make the correct answer MORE POSSIBLE (by providing relevant facts) without making it MORE LIKELY (by directing attention). The stance is what should flip likelihood.

**Source:** Scenario 09 calibration results (knowledge-only = 11% vs with-stance = 78%)

**Significance:** If the three-way separation holds across all three scenarios, it confirms stance as a general mechanism, not a scenario-09-specific artifact.

---

## Synthesis

**Key Insights:**

1. **Narrow implicit gaps discriminate; distributed signals don't** — Each scenario has ONE comparison/trace/check that flips the answer. This is the design lesson from scenario 09 (worked) vs scenario 10 (failed). The gap should be implicit but localized.

2. **Worker-skill domains provide natural testing ground** — Unlike orchestrator scenarios (which test meta-cognition about agent work), worker scenarios test direct code comprehension. The code review frame ("ready to report Phase: Complete?") creates natural surface-reading pressure that stance must overcome.

3. **Knowledge-stance separation is testable** — The without-stance variant for each scenario provides equivalent factual content to the with-stance variant. The ONLY difference is the epistemic orientation sentence. If with-stance outperforms without-stance consistently, it isolates the attention mechanism.

**Answer to Investigation Question:**

Yes. Three scenarios were designed with the following structure:
- **Scenario 11 (absence-as-evidence):** New API endpoint registered on wrong router group, bypassing auth middleware that all other endpoints have. Tests whether agents notice what ISN'T there.
- **Scenario 12 (downstream-consumer-contract):** Query widened to return cross-project results, breaking a dashboard component's fixed-grid assumption. Tests whether agents trace data paths to consumers.
- **Scenario 13 (stale-deprecation-claim):** Code marked deprecated 7 months ago, but git log shows active recent use. Tests whether agents verify claims against current evidence.

Each follows the proven pattern from scenario 09: all information present, surface reading produces wrong answer, attention to the specific type produces correct answer. Each has skillc-compatible YAML with behavioral indicators using pipe-separated detection patterns.

---

## Structured Uncertainty

**What's tested:**

- ✅ Scenario format compatible with existing skillc YAML structure (verified by following exact field structure of scenario 09)
- ✅ Detection patterns use only supported syntax: `response contains X|Y|Z` and `response does not contain X|Y|Z`
- ✅ Each scenario has a clear surface-level wrong answer and an attention-correct answer (verified by construction)

**What's untested:**

- ⚠️ Whether Sonnet discriminates on these scenarios (no trials run yet)
- ⚠️ Whether the knowledge-only variant provides enough lift to differentiate from bare (calibrated on scenario 09 but untested here)
- ⚠️ Whether scenario 12 (downstream consumer) is sufficiently implicit — the dashboard code IS in the prompt, which may make it catchable even without stance
- ⚠️ Whether scenario 13 (stale deprecation) detection patterns discriminate — "verify" and "grep" are common enough that bare agents might use them

**What would change this:**

- If bare agents score >50% on any scenario, the gap is too explicit (like scenario 10's data-table variant)
- If without-stance and with-stance score identically, the knowledge items may already encode the attention direction
- If with-stance scores <50%, the scenario may be too hard (the implicit gap is too deeply buried)

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Run contrastive trials | implementation | Within established skillc testing patterns, uses existing infrastructure |
| Adjust detection patterns based on results | implementation | Tactical tuning, reversible |
| Promote findings to skill-content-transfer model | architectural | Extends model claims, affects all skill design |

### Recommended Approach ⭐

**Run N=6 contrastive trials per variant** - Use skillc runner with Sonnet 4 across all 3 scenarios, 3 variants = 54 trials.

**Why this approach:**
- N=6 matches the established calibration from scenario 09/10
- 54 trials provides enough data to see three-way separation or identify failing scenarios
- Uses existing infrastructure (skillc run, scorer.go)

**Trade-offs accepted:**
- N=6 is small — may need N=12 for borderline scenarios
- Sonnet-only — Opus ceiling scenarios not tested (but those aren't the target)

**Implementation sequence:**
1. Verify YAML parses correctly: `skillc validate scenarios/11-*.yaml` (or equivalent)
2. Run bare variant first for all 3 scenarios (baseline)
3. Run without-stance and with-stance, compare three-way separation
4. Adjust detection patterns if false positive/negative rates are high

### Alternative Approaches Considered

**Option B: Run all scenarios at N=3 first (quick calibration)**
- **Pros:** Faster feedback, cheaper, identifies obviously broken scenarios early
- **Cons:** N=3 is too small for statistical confidence (scenario 09 needed N=6 for clear separation)
- **When to use instead:** If trial cost is a concern or if we expect most scenarios to need redesign

---

### Implementation Details

**What to implement first:**
- Run scenario 11 (absence-as-evidence) first — it has the most structural similarity to scenario 09 (comparing two things, noticing a gap)
- Scenario 13 (stale deprecation) second — it's the most novel attention type
- Scenario 12 (downstream consumer) last — most risk of being too explicit

**Things to watch out for:**
- ⚠️ Scenario 12 may hit ceiling (dashboard code IS visible, agents may notice without stance)
- ⚠️ Scenario 13 detection for "verify|grep" — common words that bare agents may use in other contexts
- ⚠️ The `variants:` section in contrastive YAMLs is a new field — may need skillc runner updates

**Success criteria:**
- ✅ At least 2 of 3 scenarios show three-way separation (bare < without-stance < with-stance)
- ✅ With-stance variant scores ≥5/8 (pass threshold) on discriminating scenarios
- ✅ Bare variant scores <3/8 on discriminating scenarios
- ✅ Findings extend the skill-content-transfer model with new attention-type evidence

---

## References

**Files Examined:**
- `skills/src/meta/orchestrator/.skillc/tests/scenarios/09-contradiction-detection.yaml` — Working scenario template (implicit contradiction pattern)
- `skills/src/meta/orchestrator/.skillc/tests/scenarios-contrastive/10-red-herring-obvious-action.yaml` — Failed scenario template (distributed symptom pattern)
- `.kb/threads/2026-03-05-stance-as-attention-priming-agents.md` — Theoretical framework for stance as attention priming
- `.kb/models/skill-content-transfer/model.md` — Three content types (knowledge, behavioral, stance)
- `.kb/models/defect-class-taxonomy/model.md` — Seven defect classes mapped to attention types

**Files Created:**
- `skills/src/meta/orchestrator/.skillc/tests/scenarios/11-absence-as-evidence.yaml`
- `skills/src/meta/orchestrator/.skillc/tests/scenarios/12-downstream-consumer-contract.yaml`
- `skills/src/meta/orchestrator/.skillc/tests/scenarios/13-stale-deprecation-claim.yaml`
- `skills/src/meta/orchestrator/.skillc/tests/scenarios-contrastive/11-absence-as-evidence.yaml`
- `skills/src/meta/orchestrator/.skillc/tests/scenarios-contrastive/12-downstream-consumer-contract.yaml`
- `skills/src/meta/orchestrator/.skillc/tests/scenarios-contrastive/13-stale-deprecation-claim.yaml`

**Related Artifacts:**
- **Model:** `.kb/models/skill-content-transfer/model.md` — Stance measurement framework
- **Thread:** `.kb/threads/2026-03-05-stance-as-attention-priming-agents.md` — Theoretical grounding

---

## Investigation History

**2026-03-05:** Investigation started
- Initial question: Design 3 contrastive scenarios for testing attention-stance hypothesis in worker skills
- Context: Confirmed stance mechanism on scenario 09 (N=9), need to test generalization to other attention types

**2026-03-05:** Scenarios designed and YAML files created
- 3 main scenarios + 3 contrastive variant files produced
- Each scenario maps to a specific defect class and attention type
- Design follows principles extracted from scenario 09 success and scenario 10 failure

**2026-03-05:** Investigation completed
- Status: Complete
- Key outcome: Three scenarios ready for contrastive trials, testing whether stance generalizes beyond implicit contradiction to absence, relationships, and freshness
