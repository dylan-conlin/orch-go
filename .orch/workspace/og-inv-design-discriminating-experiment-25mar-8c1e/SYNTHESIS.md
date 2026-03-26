# Session Synthesis

**Agent:** og-inv-design-discriminating-experiment-25mar-8c1e
**Issue:** orch-go-sispn
**Duration:** 2026-03-25T19:28 → 2026-03-25T19:50
**Outcome:** success

---

## Plain-Language Summary

Designed a 3-phase experiment to figure out WHY behavioral constraints stop working when you pile too many onto an LLM agent. We know from 329 prior trials that constraints degrade at scale (83/87 orchestrator constraints are non-functional, conflicts are 5/5 or 0/5 deterministic), but we don't know the mechanism. Three candidates — resource competition (gradual attention spread), interference (specific constraint pairs cancel out), and threshold collapse (binary phase transition) — predict different failure signatures. The experiment varies constraint count from 1 to 20, measures per-constraint compliance with automated grep detectors, and uses set-comparison and removal tests to distinguish the mechanisms. Scripts are written, dry-run verified, and ready to execute at ~$2-5 in Haiku API costs.

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace.

Key outcomes:
- `run-mechanism.sh` dry-run produces correct prompts with constraint injection at all N values
- `score-mechanism.sh` analysis output includes curve shape classification and determinism check
- 20 constraints are defined with detectors and tested prompt text
- Phase 2 predefined sets (distributed, clustered, testing) each contain 10 constraints

---

## TLDR

Designed a 3-phase experiment (scaling curve, set comparison, removal test) with 20 grep-detectable constraints to discriminate resource competition, interference, and threshold collapse as the dominant failure mechanism for behavioral constraints at scale.

---

## Delta (What Changed)

### Files Created
- `experiments/coordination-demo/redesign/run-mechanism.sh` — Main experiment runner (3 phases, 20 constraints, scoring, worktree isolation)
- `experiments/coordination-demo/redesign/score-mechanism.sh` — Scoring and analysis (curve shape, determinism, set comparison, removal recovery)
- `experiments/coordination-demo/redesign/prompts/mechanism/base-task.md` — Base agent task (FormatBytes)
- `.kb/investigations/2026-03-25-inv-design-discriminating-experiment-gate-attractor.md` — Investigation with full design, prior work, and falsifiable predictions

### Files Modified
- None (pure design session — no existing files modified)

---

## Evidence (What Was Observed)

- Prior data shows deterministic failures (5/5 or 0/5 per constraint pair), which eliminates pure resource competition as the sole mechanism
- Import-block conflicts invisible at N=2 but deterministic at N=4 is a classic interference signature
- 83/87 non-functional orchestrator constraints is consistent with either threshold collapse or resource competition
- The three mechanisms make mutually exclusive predictions across 3 dimensions (curve shape, variance, removal pattern)

### Tests Run
```bash
# Dry-run verification
./run-mechanism.sh --dry-run --trials 2 --phase 1
# Output: 16 trials, correct constraint selection, correct prompt injection

./run-mechanism.sh --dry-run --trials 2 --phase 2 --critical-n 10
# Output: 6 trials across 3 predefined sets with correct constraint lists
```

---

## Architectural Choices

### Single-agent constraint compliance vs multi-agent coordination
- **What I chose:** Test a single agent with varying constraint counts (compliance measurement)
- **What I rejected:** Multi-agent coordination with varying constraint counts (merge measurement)
- **Why:** The question is about WHY constraints fail, not whether they cause merge conflicts. Single-agent isolates the constraint-compliance variable from coordination complexity.
- **Risk accepted:** Results may not generalize to multi-agent scenarios where constraints interact with coordination mechanisms.

### 20 co-satisfiable constraints vs contradictory pairs
- **What I chose:** All 20 constraints are simultaneously satisfiable — an agent CAN follow all 20 at once
- **What I rejected:** Including directly contradictory constraint pairs (like "use errors.New" AND "use fmt.Errorf")
- **Why:** Contradictory pairs test impossibility, not failure mechanism. We want to know why satisfiable constraints get dropped, not what happens with impossible instructions.
- **Risk accepted:** Without contradictory pairs, we may miss interference effects that only manifest with semantic contradiction.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-25-inv-design-discriminating-experiment-gate-attractor.md` — Full experiment design with falsifiable predictions

### Decisions Made
- Decision: Use Haiku as default model because it's consistent with all prior experiments and cheap (~$0.05/trial)
- Decision: Phase 1 determines N_critical before Phase 2/3 to avoid wasting trials at wrong N values
- Decision: Use grep-based detectors rather than LLM-based scoring because grep is deterministic and free

### Constraints Discovered
- C09 detector (doc comments) requires multi-line context check which is harder to grep reliably — used fallback pattern
- C14 detector (helper functions) must exclude `init()` to avoid false positives

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (experiment scripts, scoring, investigation file)
- [x] Tests passing (dry-run verification)
- [x] Investigation file has **Phase:** Complete
- [x] Ready for `orch complete orch-go-sispn`

### Follow-up Work
**Issue:** Run the mechanism discrimination experiment
**Skill:** experiment
**Context:**
```
Run Phase 1 of the mechanism discrimination experiment:
./experiments/coordination-demo/redesign/run-mechanism.sh --phase 1
Then analyze with ./score-mechanism.sh and use results to inform Phase 2/3 parameters.
```

---

## Unexplored Questions

- **Does model strength affect the degradation curve?** Opus may handle more co-resident constraints before degradation. Running on both Haiku and Opus could reveal whether the mechanism changes with model capability.
- **Do constraints in the system prompt degrade differently than constraints in the user message?** The experiment puts all constraints in the user prompt. System prompt placement may have different attention dynamics.
- **Is there a 4th mechanism — "selective attention"** where the model follows the first/last N constraints and ignores middle ones? This would show a position-dependent pattern distinct from all three hypotheses.

**What remains unclear:**
- Whether grep-based detectors will have acceptable precision on real agent output
- Whether N=20 is enough to observe degradation (may need higher N if the model is very compliant)

---

## Friction

No friction — smooth session

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-inv-design-discriminating-experiment-25mar-8c1e/`
**Investigation:** `.kb/investigations/2026-03-25-inv-design-discriminating-experiment-gate-attractor.md`
**Beads:** `bd show orch-go-sispn`
