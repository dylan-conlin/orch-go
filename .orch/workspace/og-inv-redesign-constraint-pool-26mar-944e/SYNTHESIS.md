# Session Synthesis

**Agent:** og-inv-redesign-constraint-pool-26mar-944e
**Issue:** orch-go-bptuy
**Duration:** 2026-03-26 08:30 → 2026-03-26 09:15
**Outcome:** success

---

## Plain-Language Summary

Phase 1 of the mechanism discrimination experiment returned a null result because all 20 constraints were independent additions that don't interfere with each other — like asking someone to bring a hat, sunglasses, AND a jacket (easy to do all three). This session redesigned the constraint pool with 12 **tension pairs** — constraints where following one makes the other harder or impossible, like "keep it under 15 lines" AND "handle every edge case explicitly." The pool is organized in three tiers (hard contradictions, medium tension, easy calibration) so the experiment can distinguish between three candidate mechanisms for WHY constraints degrade: resource competition (everything degrades together), interference (only conflicting constraints degrade), or threshold collapse (sharp cutoff). All 24 detectors validated (48/48 tests), scoring correctly identifies which side the agent sacrifices, and the runner/analysis scripts are ready for a live experiment run.

---

## TLDR

Redesigned the mechanism experiment's constraint pool from 20 orthogonal additive constraints (which produced a flat null result) to 12 tension-based pairs organized in three difficulty tiers, enabling mechanism discrimination through per-pair sacrifice pattern analysis.

---

## Delta (What Changed)

### Files Created
- `experiments/coordination-demo/redesign/run-mechanism-v2.sh` — Runner with 12 tension pairs, 3 phases, tier metadata
- `experiments/coordination-demo/redesign/score-mechanism-v2.sh` — Analysis: per-pair resolution, tier-stratified degradation, mechanism discrimination
- `experiments/coordination-demo/redesign/test-detectors-v2.sh` — 48 detector tests against synthetic Go files
- `experiments/coordination-demo/redesign/test-scoring-v2.sh` — Scoring validation (A-wins, B-wins, both, neither)
- `.kb/investigations/2026-03-26-inv-redesign-constraint-pool-tension-based.md` — Investigation with D.E.K.N.

### Files Modified
- `.kb/models/attractor-gate/model.md` — Updated Claim 3 qualification and probe log (already done by prior session)

### Commits
- (pending — will commit before Phase: Complete)

---

## Evidence (What Was Observed)

- Phase 1 null result: ~97% compliance N=1→20 with orthogonal constraints (confirmed root cause: no tension)
- Synthetic file A (switch, error return, comments): correctly detected by all 12 pair-A detectors
- Synthetic file B (loop, simple return, types, strconv): correctly detected by all 12 pair-B detectors
- Scoring correctly classified: P01=b_wins, P02=b_wins, P09=both (EASY pair, standard Go pattern)
- Dry-run generated correct prompts with both conflicting constraints presented side-by-side

### Tests Run
```bash
# Detector validation
bash test-detectors-v2.sh
# Passed: 48 / 48 — ALL DETECTORS PASS

# Scoring validation
bash test-scoring-v2.sh
# P01=b_wins, P02=b_wins, P09=both, rate=1.000 — SCORING VALIDATION PASSED

# Dry-run
./run-mechanism-v2.sh --dry-run --trials 1 --seed 42
# 8 trials generated successfully, prompts correct
```

---

## Architectural Choices

### Tension pairs instead of individual constraint scaling
- **What I chose:** Constraint pairs (A vs B) as the atomic unit, with per-pair resolution scoring
- **What I rejected:** Larger constraint groups (3-way or n-way tensions), gradient scoring (0-1 per constraint)
- **Why:** Binary pair resolution (A-wins/B-wins/both/neither) is cleanly grep-detectable and directly maps to the mechanism discrimination predictions. Three-way tensions would create exponentially more states to score.
- **Risk accepted:** Some "MEDIUM" pairs might be closer to HARD (effectively contradictory) in practice

### Three tiers as within-experiment control
- **What I chose:** HARD/MEDIUM/EASY tiers that enable control comparison within a single experiment
- **What I rejected:** Separate experiments for each tension level
- **Why:** Same agent, same session, same N — the tier comparison is the mechanism discriminator. If EASY degrades like HARD → resource competition. If EASY holds → interference.
- **Risk accepted:** Small tier sizes (4 pairs each) may not provide enough statistical power

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` — 4 automated checks, 2 manual verifications, all passing.

---

## Knowledge (What Was Learned)

### Constraints Discovered
- Orthogonal constraints scale indefinitely (confirmed by Phase 1 null result) — the independent variable is tension, not count
- Grep-based detector design requires careful handling of existing file content (display.go already has code)
- `set -euo pipefail` interacts poorly with detector functions that intentionally return non-zero — fixed by restructuring P02 check

---

## Next (What Should Happen)

**Recommendation:** close (infrastructure ready; experiment run is a separate spawn)

### If Close
- [x] All deliverables complete (scripts, tests, investigation, synthesis)
- [x] Tests passing (48/48 detectors, scoring validation)
- [x] Investigation file has Phase: Complete
- [x] Ready for `orch complete orch-go-bptuy`

### Follow-up spawn needed:
**Issue:** Run Phase 1 tension experiment
**Skill:** experiment
**Context:**
```
Tension-based constraint pool is ready. Run: ./run-mechanism-v2.sh --trials 5 --model haiku
Then analyze: ./score-mechanism-v2.sh results/mechanism-v2-p1-*
```

---

## Unexplored Questions

- **Do agents call out contradictions explicitly?** When presented with "MUST return (string, error)" and "MUST return string only," does the agent explain the conflict in comments, or silently choose one? The stdout.log from real runs will reveal this.
- **Is there a sacrifice preference?** For HARD pairs, does the agent consistently pick side A or B? If P01 always resolves as "simple return wins," that reveals the agent's default priority ordering.
- **Does tension type matter more than tension count?** If MEDIUM pairs at N=8 degrade less than HARD pairs at N=4, that's strong evidence for interference over resource competition.

---

## Friction

No friction — smooth session. Existing experiment infrastructure was well-documented and easy to extend.

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-inv-redesign-constraint-pool-26mar-944e/`
**Investigation:** `.kb/investigations/2026-03-26-inv-redesign-constraint-pool-tension-based.md`
**Beads:** `bd show orch-go-bptuy`
