## Summary (D.E.K.N.)

**Delta:** Designed a 3-phase experiment that discriminates resource competition, interference, and threshold collapse as failure mechanisms for behavioral constraints at scale.

**Evidence:** Prior data (329 trials) shows deterministic constraint failure (5/5 or 0/5), constraint dilution at 10+, and 83/87 non-functional constraints — but cannot distinguish WHY. This experiment varies constraint count 1→20 on identical tasks and measures per-constraint compliance to expose the degradation curve shape.

**Knowledge:** The three mechanisms make mutually exclusive predictions: resource competition → gradual sigmoid + probabilistic dropout; interference → irregular curve + pair-specific failure; threshold collapse → step function + deterministic all-or-nothing. A single experiment with 3 phases can discriminate all three.

**Next:** Run Phase 1 (`./run-mechanism.sh --phase 1`), then Phase 2/3 based on results. Estimated cost: ~$2-5 at Haiku rates.

**Authority:** implementation - Experiment design within existing infrastructure, no architectural changes

---

# Investigation: Design Discriminating Experiment — Gate/Attractor Failure Mechanism

**Question:** Which failure mechanism dominates when behavioral constraints stop working at scale: resource competition (finite context), interference (constraint pairs contradict), or threshold collapse (binary phase transition)?

**Started:** 2026-03-25
**Updated:** 2026-03-25
**Owner:** orch-go-sispn
**Phase:** Complete
**Next Step:** Run experiment
**Status:** Complete
**Model:** coordination

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/models/coordination/probes/2026-03-23-probe-agent-scaling-limited-insertion-points.md` | extends | yes | - |
| `.kb/models/coordination/probes/2026-03-22-probe-attractor-decay-degradation-curve.md` | extends | yes | - |
| `.kb/models/coordination/probes/2026-03-23-probe-merge-educated-messaging-experiment.md` | extends | yes | - |
| `.kb/models/coordination/probes/2026-03-23-probe-modification-task-experiment.md` | extends | yes | - |
| `.kb/models/coordination/probes/2026-03-22-probe-automated-attractor-discovery.md` | extends | yes | - |

**Verified:** Read all 5 prior probes and the coordination model.md. Findings are consistent — deterministic conflicts (5/5 or 0/5), scaling degradation at ratio < 1.0, and 329 trials of quantitative data.

---

## Findings

### Finding 1: Three mechanisms make mutually exclusive predictions

**Evidence:** Analyzed the three candidate failure mechanisms against the existing quantitative evidence:

| Mechanism | Curve shape | Variance | Failure pattern | Removal effect |
|-----------|-----------|----------|-----------------|----------------|
| Resource competition | Gradual sigmoid | High (probabilistic: 3/5, 4/5) | Random dropout across all constraints | Uniform small improvement |
| Interference | Irregular with drops | Low (deterministic per pair) | Pair-specific: same N succeeds/fails by set | Selective: only removing bottleneck helps |
| Threshold collapse | Step function | Near-zero on each side | All-or-nothing at critical N | Universal: removing any one restores all |

**Source:** Coordination model claims (model.md lines 197-237), scaling probe data (N=4 70% pairwise, N=6 67%), determinism observation (5/5 or 0/5 per pair).

**Significance:** These predictions are falsifiable and mutually exclusive across 3 dimensions (curve shape, variance, removal pattern). A single experiment measuring all 3 dimensions can discriminate the mechanisms.

### Finding 2: Existing evidence already hints at interference + threshold hybrid

**Evidence:** Two observations from prior work point in different directions:
- **Deterministic failures (5/5 or 0/5)** eliminates pure resource competition (which predicts probabilistic 3/5, 4/5)
- **Import-block conflicts invisible at N=2 but deterministic at N=4** is a classic interference signature (specific pair that only manifests when co-resident with enough others)
- **83/87 non-functional orchestrator constraints** could be threshold collapse (crossed some critical mass) OR resource competition (each constraint diluted below detection)

The experiment must distinguish: do 83/87 constraints fail because they individually lost attention (resource competition), because specific pairs cancel each other out (interference), or because some critical count was crossed (threshold)?

**Source:** Scaling probe (`probes/2026-03-23-probe-agent-scaling-limited-insertion-points.md`), coordination model.md Part I claims.

**Significance:** The existing data narrows the hypothesis space but cannot determine the dominant mechanism. The experiment is genuinely needed — it's not answerable from prior data alone.

### Finding 3: Experiment infrastructure supports this design cleanly

**Evidence:** The existing coordination-demo infrastructure (run.sh, run-scaling.sh) provides:
- Agent runner with worktree isolation, timeout, and artifact capture
- Scoring with per-trial metric extraction
- Randomization with seeded Fisher-Yates shuffle
- Metadata capture for reproducibility

The new experiment (`run-mechanism.sh`) reuses these patterns but shifts from merge-conflict measurement to per-constraint compliance measurement. 20 co-satisfiable constraints with grep-detectable compliance markers enable automated scoring.

**Source:** `experiments/coordination-demo/redesign/run.sh` (450 lines), `run-scaling.sh` (500 lines)

**Significance:** Low implementation risk — builds on proven infrastructure with proven scoring patterns.

---

## Synthesis

**Key Insights:**

1. **The experiment measures shape, not magnitude** — We already know constraints degrade at scale. What matters is the degradation SHAPE: gradual (resource), jagged (interference), or step (threshold). Each shape implies different interventions.

2. **Three independent measurements are needed** — No single metric discriminates all three mechanisms. Curve shape, per-constraint determinism, and removal recovery must all be measured. Hence 3 phases.

3. **The experiment also tests the "competing attractors" question** — Phase 2 compares semantically clustered vs distributed constraint sets at fixed N. If clustered sets degrade more, interference is operating through semantic proximity — a novel finding about how attractors interact.

**Answer to Investigation Question:**

The experiment is designed. Three phases provide the discriminating data:

- **Phase 1 (Scaling curve)**: 40 trials across N={1,2,4,6,8,10,14,20}, 5 trials each with randomly selected constraints. Measures: compliance rate vs N (curve shape), per-constraint survival, failure determinism. Estimated cost: ~$2 at Haiku.

- **Phase 2 (Set specificity)**: 15 trials at N_critical with 3 predefined constraint sets (distributed, error-clustered, testing-clustered). Measures: whether set COMPOSITION matters at fixed N. If yes → interference. If no → resource competition or threshold.

- **Phase 3 (Removal test)**: ~55 trials removing one constraint at a time from a failing set. Measures: which removals restore compliance and in what pattern. Uniform → resource. Selective → interference. Universal → threshold.

---

## Structured Uncertainty

**What's tested:**

- ✅ Experiment scripts are syntactically correct and produce expected output (verified: dry-run with --dry-run flag)
- ✅ Constraint detectors use grep patterns that match expected code artifacts (verified: reviewed against Go syntax)
- ✅ Prior scaling data confirms degradation exists (verified: 67-70% pairwise at N=4-6 from probe data)

**What's untested:**

- ⚠️ Whether Haiku produces enough constraint-following variance at N=10-20 to discriminate mechanisms (if compliance drops to 0% at N=6, Phase 2/3 need lower critical N)
- ⚠️ Whether the 20 constraints are truly co-satisfiable in practice (some may be architecturally awkward together)
- ⚠️ Whether grep-based detectors have sufficient precision (false positives on existing code, false negatives on valid implementations)

**What would change this:**

- If compliance is 100% at N=20 → constraints don't degrade at this scale; need harder task or weaker model
- If compliance is 0% at N=4 → degradation starts earlier than expected; adjust N range downward
- If detectors show >20% false positive rate → need human scoring or more specific detectors

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Run the experiment | implementation | Uses existing infrastructure, no cross-boundary impact |
| Interpret results for model update | architectural | Results may change coordination model claims |
| Decision on whether to invest in constraint-aware prompt engineering | strategic | Resource allocation and research direction |

### Recommended Approach

**Run Phase 1 first, then adapt** — Execute the scaling curve (40 trials, ~$2), analyze the curve shape, then design Phase 2/3 parameters based on actual results.

**Why this approach:**
- Phase 1 results determine the critical N for Phase 2/3
- Avoids wasting trials at wrong N values
- Fastest path to first discriminating data point

**Implementation sequence:**
1. `./run-mechanism.sh --phase 1 --trials 5` — run scaling curve (~40 trials)
2. `./score-mechanism.sh results/mechanism-p1-TIMESTAMP` — analyze curve shape
3. Identify N_critical from degradation curve
4. `./run-mechanism.sh --phase 2 --critical-n <N>` — set comparison
5. `./run-mechanism.sh --phase 3 --critical-n <N>` — removal test
6. Update coordination model.md with mechanism findings

### Alternative Approaches Considered

**Option B: Single-phase with more trials per N**
- **Pros:** Simpler, more statistical power per data point
- **Cons:** Can't discriminate interference from threshold without Phase 2/3
- **When to use instead:** If curve shape alone is sufficient answer

**Option C: Use Opus instead of Haiku**
- **Pros:** Higher baseline compliance, may show degradation at higher N
- **Cons:** 50x cost increase (~$100 vs $2), longer runtime
- **When to use instead:** If Haiku shows 0% compliance before N=6

---

## References

**Files Examined:**
- `.kb/models/coordination/model.md` — Parent coordination model (329 trials, 14 probes)
- `.kb/models/coordination/probes/2026-03-23-probe-agent-scaling-limited-insertion-points.md` — Scaling data (N=4,6)
- `.kb/models/coordination/probes/2026-03-22-probe-attractor-decay-degradation-curve.md` — Attractor resilience (9/9)
- `experiments/coordination-demo/redesign/run.sh` — Base experiment infrastructure
- `experiments/coordination-demo/redesign/run-scaling.sh` — Scaling experiment pattern

**Artifacts Created:**
- `experiments/coordination-demo/redesign/run-mechanism.sh` — Main experiment runner (3 phases)
- `experiments/coordination-demo/redesign/score-mechanism.sh` — Scoring and analysis
- `experiments/coordination-demo/redesign/prompts/mechanism/base-task.md` — Agent task prompt

---

## Investigation History

**2026-03-25:** Investigation started
- Initial question: Which failure mechanism dominates for behavioral constraints at scale?
- Context: Coordination model has 329 trials and 0 disconfirmation attempts. Mechanism question is the largest open gap.

**2026-03-25:** Prior work analysis complete
- Read all 5 relevant probes and the full coordination model
- Identified 3 mutually exclusive predictions from the 3 candidate mechanisms

**2026-03-25:** Experiment design complete
- Created 3-phase experiment with 20 co-satisfiable constraints
- Verified dry-run produces correct prompts and metadata
- Estimated cost: $2-5 at Haiku rates for all 3 phases
