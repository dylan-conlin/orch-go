# Session Synthesis

**Agent:** og-inv-experiment-git-merge-23mar-0141
**Issue:** orch-go-y4o3s
**Duration:** ~90 minutes
**Outcome:** success

---

## Plain-Language Summary

We tested whether teaching AI agents how git merge actually works would help them avoid merge conflicts when coding in parallel. The baseline: when two agents share plans without merge education, they conflict 100% of the time because they falsely believe different function names at the same insertion point will merge cleanly. With merge education (a 200-word explanation of textual merge mechanics plus a concrete example), conflicts dropped to 70% — a statistically significant improvement (p=0.02). This means the false merge model was suppressing about 30% of coordination capacity. But 70% still conflict despite understanding merge mechanics, because agents have five other failure modes: they converge on the same "safe" alternative spot, misread each other's plans, contradict themselves, miscalculate line gaps, or write plans that cross in flight. The communication ceiling is ~30%, well below structural placement (100%). Merge education is necessary but not sufficient.

## Verification Contract

See `VERIFICATION_SPEC.yaml` for reproducibility details. Key outcomes:
- 20 trials executed (10 simple, 10 complex)
- 6/20 SUCCESS (4 simple, 2 complex) vs 0/20 baseline
- Fisher's exact p=0.02 (two-sided)
- All 40 agents scored 6/6 individually

---

## Delta (What Changed)

### Files Created
- `experiments/coordination-demo/redesign/run-merge-educated.sh` — Experiment harness for merge-educated messaging condition
- `experiments/coordination-demo/redesign/results/merge-educated-20260323-093342/` — Full results (20 trials, plans, diffs, scores)
- `.kb/models/coordination/probes/2026-03-23-probe-merge-educated-messaging-experiment.md` — Probe documenting findings

### Files Modified
- `.kb/models/coordination/model.md` — Updated Claim 1, core insight, Implication 1, observed patterns, evidence summary, open questions (answered), and added Key Experiment section

---

## Evidence (What Was Observed)

- **Merge-educated messaging: 6/20 SUCCESS (30%)** vs messaging baseline 0/20 (0%)
- **Fisher's exact test: p=0.010 one-sided, p=0.020 two-sided** — statistically significant
- **95% Wilson CI for success rate: [14.5%, 51.9%]**
- **Simple tasks: 4/10 SUCCESS (40%)** — better coordination than complex
- **Complex tasks: 2/10 SUCCESS (20%)** — larger insertions reduce viable separation gaps
- **All 40 agents: 6/6 individual scores** — Claim 3 continues to hold
- **5 distinct failure patterns identified** from plan file analysis of 14 conflict trials

### Tests Run
```bash
# Full experiment run
./run-merge-educated.sh --trials 10
# 20 trials completed, scoring and analysis automatic

# Statistical test
python3 -c "from math import comb; print(comb(6,6)*comb(34,14)/comb(40,20))"
# p = 0.010098 (one-sided)
```

---

## Architectural Choices

No architectural choices — this was an experiment execution and analysis task.

---

## Knowledge (What Was Learned)

### Key Findings

1. **The false merge model IS real and quantifiable.** It suppresses ~30% of coordination capacity (the gap between 0% and 30% success). Previous probes identified it qualitatively; this experiment quantifies its impact.

2. **Communication has a structural ceiling.** Even with correct merge knowledge, communication achieves only ~30% success. The file geometry (2-3 viable insertion points, ~50 lines of usable space, agents producing 30-80 lines each) constrains the coordination ceiling.

3. **Neither pure hypothesis was correct.** The "false merge model" hypothesis predicted high success (wrong — 30% not 80%+). The "semantic gravity" hypothesis predicted no change (wrong — 30% not 0%). The real mechanism is a combination: false merge model + structural constraints + communication quality failures.

4. **Five failure patterns persist despite education:** mutual convergence, plan misreading, self-contradiction, gap overestimation, messaging lag. These represent the irreducible communication failure surface for single-round messaging.

---

## Next (What Should Happen)

**Recommendation:** close

- [x] All deliverables complete
- [x] Experiment run (20 trials)
- [x] Probe file created and complete
- [x] Model updated with findings
- [x] SYNTHESIS.md created
- [x] Ready for `orch complete orch-go-y4o3s`

---

## Unexplored Questions

- **Multi-round negotiation:** Would allowing agents to revise plans iteratively (Pattern 5 fix) plus force re-reading each other's plans (Pattern 2 fix) close the gap further? Estimated ceiling: 50-60%.
- **Tool-augmented messaging:** A `check-merge-conflict` tool that simulates the merge before commit would bypass all 5 failure patterns. But this converts messaging to a different coordination mechanism.
- **Model strength:** Would a stronger model (Opus vs Haiku) reduce communication quality failures (patterns 2-4)?

---

## Friction

- `go build ./...` false positives from stale `.go` files in experiment results directories. Fixed by targeting `go build ./pkg/display/` instead. Same issue affected the anticipatory placement experiment.
- Edit tool unicode em-dash matching failure in model.md — required sed workaround for one edit.

---

## Session Metadata

**Skill:** investigation (probe mode)
**Model:** claude-opus-4-5 (session), claude-haiku-4-5 (experiment agents)
**Workspace:** `.orch/workspace/og-inv-experiment-git-merge-23mar-0141/`
**Probe:** `.kb/models/coordination/probes/2026-03-23-probe-merge-educated-messaging-experiment.md`
**Beads:** `bd show orch-go-y4o3s`
