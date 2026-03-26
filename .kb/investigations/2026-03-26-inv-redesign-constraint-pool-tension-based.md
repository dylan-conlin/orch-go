## Summary (D.E.K.N.)

**Delta:** Designed 12 tension-based constraint pairs (24 constraints) that create genuine competing gradients — where satisfying constraint A measurably reduces compliance with constraint B — replacing the orthogonal additive pool that produced Phase 1's null result.

**Evidence:** 48/48 detector tests pass across 2 synthetic Go files; scoring correctly identifies A-wins, B-wins, both-satisfied, and neither outcomes; HARD pairs (logically contradictory) force one-side sacrifice while EASY pairs (standard Go patterns) allow both-satisfied.

**Knowledge:** The independent variable for constraint degradation is not count but tension — orthogonal constraints scale indefinitely, but competing-gradient constraints should force tradeoffs that scale with N. Three-tier design (HARD/MEDIUM/EASY) enables mechanism discrimination within a single experiment.

**Next:** Run Phase 1 experiment: `./run-mechanism-v2.sh --trials 5 --model haiku` to produce tension scaling curve. Compare HARD vs EASY tier degradation to discriminate resource competition from interference.

**Authority:** implementation — New experiment scripts within existing infrastructure, no architectural changes.

---

# Investigation: Redesign Constraint Pool — Tension-Based Constraints for Mechanism Discrimination

**Question:** How should the constraint pool be redesigned so that Phase 2 of the mechanism discrimination experiment measures genuine constraint competition rather than boilerplate accretion?

**Started:** 2026-03-26
**Updated:** 2026-03-26
**Owner:** orch-go-bptuy
**Phase:** Complete
**Next Step:** Run experiment with new constraint pool
**Status:** Complete
**Model:** attractor-gate

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/models/attractor-gate/probes/2026-03-26-probe-constraint-scaling-null-result.md | extends | yes | None — null result confirmed orthogonal constraints don't degrade |
| .kb/investigations/2026-03-25-inv-design-discriminating-experiment-gate-attractor.md | extends | yes | Original experiment design didn't anticipate orthogonal-constraint null result |

---

## Findings

### Finding 1: Phase 1 null result was caused by orthogonal constraint design, not experimental methodology

**Evidence:** 40 trials across N=1→20 with 20 constraints showed ~97% compliance (flat curve). All 20 constraints were additive requests (add named returns, add const block, add example test) that don't interfere with each other. An agent can satisfy all 20 because each is a local addition.

**Source:** `.kb/models/attractor-gate/probes/2026-03-26-probe-constraint-scaling-null-result.md`, `experiments/coordination-demo/redesign/results/mechanism-p1-20260325-230114/`

**Significance:** The experimental infrastructure (runner, scorer, worktree management) is sound. Only the constraint pool needs redesign. The dependent variable needs to shift from compliance rate to sacrifice pattern.

---

### Finding 2: Three-tier tension design enables within-experiment mechanism discrimination

**Evidence:** Designed 12 constraint pairs in three tiers:
- **HARD** (P01-P04): Logically contradictory — exactly one side must win. Examples: error return vs simple return, switch vs loop, no comments vs heavy comments, no types vs rich types.
- **MEDIUM** (P05-P08): Both satisfiable but practically competing — creative solutions exist but are hard. Examples: 15-line brevity vs exhaustive edge cases, package-level lookup table vs self-contained function.
- **EASY** (P09-P12): Both satisfiable with standard Go patterns — serves as calibration control. Examples: standalone function + Formatter struct, table tests + benchmark.

**Source:** `experiments/coordination-demo/redesign/run-mechanism-v2.sh` (constraint pool lines 83-150)

**Significance:** If EASY pairs degrade at high N → resource competition (even non-conflicting constraints lose under load). If HARD/MEDIUM degrade while EASY holds → interference (tension-specific degradation). If sharp cutoff at same N across all tiers → threshold collapse. This discrimination was impossible with the orthogonal Phase 1 design.

---

### Finding 3: Per-pair scoring captures sacrifice patterns, not just compliance rates

**Evidence:** New scoring produces 4 outcomes per pair: A-wins (constraint A satisfied, B not), B-wins (B satisfied, A not), both (agent found creative workaround), neither (agent dropped both). Tested with synthetic files: HARD pairs correctly show one-side wins, EASY pairs correctly show both-satisfied. 48/48 detector tests pass.

**Source:** `experiments/coordination-demo/redesign/test-detectors-v2.sh`, `experiments/coordination-demo/redesign/test-scoring-v2.sh`

**Significance:** The dependent variable is now which constraint the agent sacrifices (sacrifice pattern) rather than what percentage it complied with (compliance rate). This directly measures constraint competition — the phenomenon the model claims causes degradation.

---

## Synthesis

**Key Insights:**

1. **Tension is the independent variable, not count** — The Phase 1 null result proved that constraint count alone doesn't cause degradation. The redesigned experiment holds count constant (2 per pair × N pairs) while varying tension level (HARD/MEDIUM/EASY), isolating tension as the causal variable.

2. **EASY-tier calibration is the critical comparison** — If EASY pairs (no genuine tension) maintain both-satisfied rate at high N while HARD pairs force sacrifice, that's direct evidence of interference as the mechanism. Without this control tier, any observed degradation could be attributed to simple load/context effects.

3. **Sacrifice patterns reveal agent decision-making** — Measuring WHICH side wins (A or B) across trials reveals whether agents have consistent preferences (deterministic) or random dropout (stochastic), directly discriminating the three candidate mechanisms.

**Answer to Investigation Question:**

The constraint pool should use tension pairs — sets of two constraints where satisfying A makes B harder — organized in three difficulty tiers. Scoring measures per-pair resolution (which side wins) rather than overall compliance rate. The three-tier design enables mechanism discrimination within a single experiment run.

---

## Structured Uncertainty

**What's tested:**

- ✅ All 24 constraint detectors produce correct results against synthetic Go files (48/48 pass)
- ✅ Scoring function correctly classifies A-wins, B-wins, both, neither outcomes
- ✅ Runner generates correct prompts with both sides of each pair presented to agent
- ✅ Analysis script produces tier-stratified degradation curves

**What's untested:**

- ⚠️ Whether HARD pairs actually force sacrifice in practice (agents may refuse or find unexpected workarounds)
- ⚠️ Whether MEDIUM pairs produce the expected ~20-40% both-satisfied rate
- ⚠️ Whether there's sufficient variance between tiers to discriminate mechanisms
- ⚠️ Whether Haiku (the default model) engages with contradictory constraints or just picks arbitrarily

**What would change this:**

- If HARD pairs show high both-satisfied rates, the "contradictions" aren't contradictory enough
- If EASY pairs show low both-satisfied rates, the detectors may be miscalibrated
- If all tiers show identical degradation patterns, the tier distinction doesn't matter

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Run Phase 1 with tension constraints | implementation | Uses existing infrastructure, no code changes needed |
| Analyze tier-stratified results | implementation | Analysis script already built |
| Update attractor-gate model based on results | implementation | Probe-to-model merge is standard workflow |

### Recommended Approach: Run Phase 1, Then Phase 2 If Signal Found

**Why this approach:**
- Phase 1 (scaling curve) reveals IF tension-based constraints produce degradation at all
- If Phase 1 shows degradation, Phase 2 (tier comparison) reveals mechanism type
- Phase 3 (pair removal) only runs if Phase 2 shows pair-specific effects

**Implementation sequence:**
1. `./run-mechanism-v2.sh --trials 5 --model haiku` — 40 trials, ~7 hours
2. `./score-mechanism-v2.sh results/mechanism-v2-p1-*` — analyze scaling curve
3. If degradation found at N_critical: `./run-mechanism-v2.sh --phase 2 --critical-n <N>`

**Things to watch out for:**
- ⚠️ P05 brevity detector uses awk line counting — may be off-by-one depending on formatting style
- ⚠️ P03 no-comments detector checks entire file, not just new code — existing comments in display.go would trigger false positive (but agents write to clean worktrees so this shouldn't matter)
- ⚠️ Agents may explicitly call out contradictions in their output rather than silently choosing

---

## References

**Files Examined:**
- `experiments/coordination-demo/redesign/run-mechanism.sh` — Phase 1 runner (v1), constraint pool structure
- `experiments/coordination-demo/redesign/score-mechanism.sh` — Phase 1 scorer, analysis patterns
- `experiments/coordination-demo/redesign/prompts/mechanism/base-task.md` — Base FormatBytes task
- `.kb/models/attractor-gate/model.md` — Parent model with Claim 3 qualification
- `.kb/models/attractor-gate/probes/2026-03-26-probe-constraint-scaling-null-result.md` — Phase 1 null result

**Commands Run:**
```bash
# Dry-run to verify prompt generation
./run-mechanism-v2.sh --dry-run --trials 1 --seed 42

# Detector validation (48 tests)
bash test-detectors-v2.sh

# Scoring validation with synthetic data
bash test-scoring-v2.sh

# Analysis script validation
./score-mechanism-v2.sh /tmp/mech2-scoring-test
```

**Related Artifacts:**
- **Workspace:** `.orch/workspace/og-inv-redesign-constraint-pool-26mar-944e/`
- **Scripts:** `experiments/coordination-demo/redesign/run-mechanism-v2.sh`, `score-mechanism-v2.sh`
- **Tests:** `experiments/coordination-demo/redesign/test-detectors-v2.sh`, `test-scoring-v2.sh`
