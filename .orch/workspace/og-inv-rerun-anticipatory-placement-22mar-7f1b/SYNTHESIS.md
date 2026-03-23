# Session Synthesis

**Agent:** og-inv-rerun-anticipatory-placement-22mar-7f1b
**Issue:** orch-go-dxhse
**Duration:** 2026-03-22 ~19:00 -> 2026-03-22 ~19:45
**Outcome:** success

---

## Plain-Language Summary

The original anticipatory placement experiment claimed 12/20 (60%) success, but that number came from manually reclassifying 12 build failures as false negatives caused by stale Go files in the repo. An epistemic audit flagged this as a confound that needed a clean rerun.

We fixed the harness (scoping `go build` to the package under test instead of the whole repo), preregistered our exclusion rules (no post-hoc reclassification), and reran 20 trials. The clean result is **9/20 (45%)** — lower than the original claimed 60%, but the directional finding holds: complex tasks succeed at 70% vs simple at 20%. The strongest new finding is that placement distance almost perfectly predicts outcome: distant placements succeed 100% (7/7), adjacent succeed 22% (2/9), same succeed 0% (0/3). Zero build failures in the clean run confirms the harness fix worked.

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` for experiment parameters, preregistered exclusion rules, and raw results.

---

## Delta (What Changed)

### Files Modified
- `experiments/coordination-demo/redesign/run-anticipatory.sh` — Fixed `go build ./...` to `go build ./pkg/display/` in two locations (run_agent and check_merge)

### Files Created
- `.kb/investigations/2026-03-22-inv-rerun-anticipatory-placement-experiment-clean.md` — Full investigation with preregistered protocol, raw results, and analysis
- `experiments/coordination-demo/redesign/results/20260322-190106/` — Clean experiment results (20 trials)
- `.orch/workspace/og-inv-rerun-anticipatory-placement-22mar-7f1b/VERIFICATION_SPEC.yaml`
- `.orch/workspace/og-inv-rerun-anticipatory-placement-22mar-7f1b/SYNTHESIS.md`

---

## Evidence (What Was Observed)

### Key Results

| Metric | Original (corrected) | Clean Rerun |
|--------|---------------------|-------------|
| Overall | 12/20 (60%) | **9/20 (45%)** |
| Simple | 2/10 (20%) | 2/10 (20%) |
| Complex | 10/10 (100%) | **7/10 (70%)** |
| Build failures | 12 (reclassified) | **0** |

### Placement Distance Analysis

| Distance | Trials | Success | Rate |
|----------|--------|---------|------|
| Distant | 7 | 7 | 100% |
| Adjacent | 9 | 2 | 22% |
| Same | 3 | 0 | 0% |

### Test Evidence
```bash
# Harness verification
go build ./pkg/display/   # PASS
go test ./pkg/display/    # ok (0.353s)

# Experiment run
bash experiments/coordination-demo/redesign/run-anticipatory.sh --trials 10
# 20 trials completed, 0 build failures, 9 successes
```

---

## Architectural Choices

No architectural choices — task was within existing patterns. The only change was scoping `go build` to the relevant package.

---

## Knowledge (What Was Learned)

### Constraints Discovered
- Post-hoc reclassification inflated the original result by 15 percentage points (60% -> 45%)
- The simple task success rate (20%) reproduced exactly, suggesting the placement model's gravitational bias toward adjacent functions is stable
- Complex task success dropped from 100% to 70%, with 3 failures attributable to same-placement generation (2) and adjacent placement (1)

---

## Next (What Should Happen)

**Recommendation:** close

- [x] All deliverables complete
- [x] Investigation file has Phase: Complete
- [x] VERIFICATION_SPEC.yaml created
- [x] Ready for `orch complete orch-go-dxhse`

---

## Unexplored Questions

- Would Opus as placement model produce more distant (and thus more successful) placements?
- Is the 2/9 adjacent success rate stable or noise? (Larger N would clarify)
- Does placement distance as a predictor generalize beyond same-file edits?

---

## Friction

No friction — smooth session. Experiment ran cleanly after harness fix.

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-inv-rerun-anticipatory-placement-22mar-7f1b/`
**Investigation:** `.kb/investigations/2026-03-22-inv-rerun-anticipatory-placement-experiment-clean.md`
**Beads:** `bd show orch-go-dxhse`
