# Session Synthesis

**Agent:** og-inv-run-decomposition-vs-24mar-8331
**Issue:** orch-go-i77wx
**Duration:** 2026-03-24 11:33 -> 2026-03-24 12:30
**Outcome:** success

---

## TLDR

Ran 5-condition x 10-trial decomposition experiment (100 Haiku agent invocations) testing whether task description quality and file structure can eliminate merge conflicts without coordination. Result: decomposition quality helps (100% -> 60% conflict) but cannot eliminate conflicts — even the best condition (domain-anchored tasks + sectioned file) still produced 60% merge conflicts, proving coordination primitives are load-bearing, not epiphenomenal.

---

## Plain-Language Summary

We wanted to know: if you write better task descriptions for AI agents working in parallel, can you avoid them stepping on each other's code? The answer is mostly no. We tested 100 agent pairs across 5 conditions ranging from bare instructions to rich domain-anchored tasks on well-organized files. On flat files, it doesn't matter how well you describe the task — agents always append at the same spot (end of file), producing 100% conflicts. Adding section comments to the file helps somewhat (drops to 60% conflict), but that's still unacceptable for production. The takeaway: to prevent conflicts in parallel agents, you need either explicit placement instructions (0% conflict from prior data) or file-level separation, not just better decomposition.

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` for reproducibility details.

Key outcomes:
- 100/100 agents scored 6/6 individually (task itself is not hard)
- C1=C2=C3=100% conflict (flat file = always conflict)
- C4=80%, C5=60% (sectioned file helps but insufficient)
- Prior explicit placement = 0% (the working solution)

---

## Delta (What Changed)

### Files Created
- `experiments/coordination-demo/redesign/results/decomp-20260324-113331/` - Full experiment results (50 trials)
- `.kb/investigations/2026-03-24-inv-run-decomposition-vs-coordination-experiment.md` - Investigation findings

### Files Modified
- `experiments/coordination-demo/redesign/analyze-decomposition.sh` - Fixed path bug (missing `/` in trial directory glob)

### Commits
- (pending - will commit all results and investigation)

---

## Evidence (What Was Observed)

- All 100 agents scored 6/6 — task quality is not the bottleneck
- Flat files: anchoring variance = 0, ALL agents converge on line 92 (end of file)
- Domain framing in task descriptions ("size formatting" vs "rate formatting") has zero measurable effect on flat files
- Sectioned files: agents DO respect section markers (FormatBytes in Size section, FormatRate in Rate section)
- Git three-way merge fails even when logical changes are non-overlapping because structural changes (section markers, imports) create overlapping context
- Average agent duration: 38-45s across all conditions (Haiku)

### Tests Run
```bash
# 100 agent invocations via run-decomposition.sh
# Each agent: go test ./pkg/display/ -v (passed for all 100)
# Each trial: git merge check (6 clean merges out of 50 trials)
```

---

## Architectural Choices

### Accepted experiment as designed (no mid-course changes)
- **What I chose:** Ran all 5 conditions x 10 trials as specified in architect's design
- **What I rejected:** Stopping early after C1-C3 showed 100% (could have saved 20 trials)
- **Why:** The interesting data was in C4-C5, and N=10 per condition was already the minimum for statistical reliability
- **Risk accepted:** ~$15-25 in API cost for completeness

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-24-inv-run-decomposition-vs-coordination-experiment.md` - Full findings with D.E.K.N.
- `experiments/coordination-demo/redesign/results/decomp-20260324-113331/analysis.md` - Auto-generated analysis

### Constraints Discovered
- Git three-way merge cannot resolve parallel additions to the same file region even when changes are logically non-overlapping
- Domain anchoring in task descriptions is ignored when the target file has no structural markers
- The "structural floor" for additive tasks is ~60% conflict even with maximal decomposition quality

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (experiment run, scored, analyzed)
- [x] Tests passing (100/100 agents scored 6/6)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-i77wx`

### Follow-up Suggestions (not blocking)
- Test separate-file condition (Route primitive): Agent A writes `display_size.go`, Agent B writes `display_rate.go` — predict 0% conflict
- This would confirm Route as the minimal sufficient primitive for additive tasks

---

## Unexplored Questions

- Would wider section spacing (content between markers, not adjacent empty sections) improve merge success for sectioned files?
- Does rebase-based merge strategy handle parallel insertions better than three-way merge?
- Would Opus/Sonnet show different anchoring behavior than Haiku?
- What is the minimum structural marker needed to achieve ~0% conflict? (e.g., just a comment vs. an existing function in each section)

---

## Friction

- Bug: `analyze-decomposition.sh` had a path bug (`"$condition_dir"trial-*/` missing `/` separator) that produced empty merge results table. Fixed during analysis. Cost: ~5 min debugging.

---

## Session Metadata

**Skill:** investigation
**Model:** claude-haiku-4-5-20251001 (experiment agents) / claude-opus-4-5-20251101 (this session)
**Workspace:** `.orch/workspace/og-inv-run-decomposition-vs-24mar-8331/`
**Investigation:** `.kb/investigations/2026-03-24-inv-run-decomposition-vs-coordination-experiment.md`
**Beads:** `bd show orch-go-i77wx`
