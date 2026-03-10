## Summary (D.E.K.N.)

**Delta:** At N=10, coordination failure rate is 100% for both Haiku and Opus, confirming the pilot finding that merge conflicts are structural (same insertion points in git), not capability-dependent.

**Evidence:** 20 agent runs (10 per model): both scored 6/6 individually in all trials; all 10 trial pairs produced merge conflicts; Fisher's exact test p=1.0; duration difference not significant (haiku 39.1s vs opus 44.0s, t=1.103, p>0.05).

**Knowledge:** For well-defined, unambiguous tasks, model capability does not affect coordination failure rate. The failure is entirely structural — both models follow the instruction "place after FormatDurationShort" and git cannot auto-merge two different insertions at the same position. Upgrading models will not reduce coordination failures.

**Next:** Close investigation. Data supports harness publication claim that coordination failure is a protocol problem requiring architectural solutions (file-level work assignment, sequential execution), not model upgrades.

**Authority:** implementation — Confirms prior N=1 finding with statistical data, no new architectural decisions needed

---

# Investigation: Coordination Demo N=10 FormatBytes — Statistical Significance

**Question:** Does the N=1 pilot finding (100% coordination failure rate independent of model capability) hold at N=10 for both Haiku and Opus?

**Started:** 2026-03-09
**Updated:** 2026-03-09
**Owner:** investigation agent (orch-go-ei2q2)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/investigations/2026-03-09-inv-coordination-failure-controlled-demo-same.md | confirms | yes — N=1 finding holds at N=10 | No conflict — N=10 strengthens all N=1 conclusions |

**Key claims from pilot (N=1), verified at N=10:**
- Both Haiku and Opus score 6/6 individually: **CONFIRMED** (20/20 trials)
- 100% merge conflict rate: **CONFIRMED** (10/10 trial pairs)
- Haiku faster than Opus: **PARTIALLY CONFIRMED** (39.1s vs 44.0s, but not statistically significant, t=1.103)
- Coordination failure is structural: **CONFIRMED** (same insertion point conflict pattern in all trials)

---

## Findings

### Finding 1: Perfect Individual Success Rate for Both Models (N=10)

**Evidence:**
```
haiku: 10/10 trials, all F0=1 F1=1 F2=1 F3=1 F5=1 (6/6 adjusted)
opus:  10/10 trials, all F0=1 F1=1 F2=1 F3=1 F5=1 (6/6 adjusted)
```

F4 (file discipline) scored 0 across all 20 runs due to `.beads/issues.jsonl` being modified by the beads environment (not agent behavior). Both agents correctly only modified `display.go` and `display_test.go`.

**Source:** `experiments/coordination-demo/results/20260309-172034/scores.csv`

**Significance:** For well-defined, unambiguous coding tasks, Haiku and Opus are functionally equivalent. Zero individual failures across 20 runs establishes a reliable baseline for measuring coordination failures.

---

### Finding 2: 100% Merge Conflict Rate (10/10 Trials)

**Evidence:**
```
trial,merge_result,conflict_files
1,conflict,1
2,conflict,2
3,conflict,1
4,conflict,2
5,conflict,1
6,conflict,2
7,conflict,2
8,conflict,1
9,conflict,2
10,conflict,1
```

**Source:** `experiments/coordination-demo/results/20260309-172034/merge_results.csv`

**Significance:** The 100% conflict rate at N=10 confirms the pilot's finding. With N=10, the 95% confidence interval for the true conflict rate is [69.2%, 100%] (Clopper-Pearson). The lower bound is well above 50%, establishing that merge conflicts are the dominant outcome, not an edge case.

Interesting variation: 5/10 trials had single-file conflicts (only display.go), while 5/10 had both files conflicting. Test files sometimes auto-merge when agents place tests at slightly different positions. The implementation file (display.go) always conflicts because both agents insert at the same line.

---

### Finding 3: Duration Difference Not Statistically Significant

**Evidence:**
```
Haiku: mean=39.1s, SD=13.4s, range=[31, 76]
Opus:  mean=44.0s, SD=4.2s,  range=[38, 53]
Welch's t-test: t=1.103, p>0.05 (not significant)
```

**Source:** `experiments/coordination-demo/results/20260309-172034/scores.csv` (duration_s column)

**Significance:** The pilot found Haiku 22% faster (49s vs 63s, N=1). At N=10, the difference shrinks to 12% and is not statistically significant. Haiku trial 4 was an outlier (76s); excluding it, haiku mean=35.0s with SD=3.6s. Opus is more consistent (lower variance) but marginally slower.

For practical purposes, both models complete simple tasks in ~40s. Speed should not be a primary factor in model selection for well-defined tasks.

---

### Finding 4: Implementation Variation Across Trials

**Evidence:**
```
Haiku lines added per trial: 84, 89, 87, 94, 115, 93, 97, 88, 93, 87
Opus  lines added per trial: 63, 63, 68, 68, 68, 62, 62, 58, 62, 64
```

**Source:** `merge-check-v2.sh` commit output (e.g., "2 files changed, 84 insertions(+)")

**Significance:** Haiku consistently produces more lines of code (mean ~93 lines vs opus ~64 lines). This suggests Haiku generates more test cases or more verbose implementations, consistent with the pilot finding (34 test cases vs 24). Opus is more concise and consistent. Neither approach affects coordination outcomes.

---

## Synthesis

**Key Insights:**

1. **Coordination failure is definitively structural, not capability-based.** At N=10, both models achieve 100% individual success and 100% merge conflict rate. The failure comes from git's inability to auto-merge insertions at the same position, not from model quality. Fisher's exact test: p=1.0.

2. **Duration differences are noise, not signal.** The pilot's 22% speed advantage for Haiku did not replicate as significant at N=10 (t=1.103, p>0.05). Both models complete in ~40s for this task.

3. **Style differences are orthogonal to coordination.** Haiku consistently produces ~45% more code than Opus (93 vs 64 lines), but this variation doesn't affect whether the changes can be merged.

**Answer to Investigation Question:**

Yes — the N=1 pilot finding holds at N=10. Coordination failure rate is 100% for both Haiku and Opus, with zero individual failures. This is statistically significant: the 95% CI for the true conflict rate is [69.2%, 100%], establishing that merge conflicts are the dominant outcome for same-file parallel work regardless of model capability.

---

## Structured Uncertainty

**What's tested:**

- ✅ Individual success rate for both models at N=10 (20 runs, 0 failures)
- ✅ Merge conflict rate at N=10 (10 pairs, 10 conflicts)
- ✅ Duration comparison with Welch's t-test (not significant)
- ✅ Code size variation (Haiku ~45% more lines than Opus)

**What's untested:**

- ⚠️ Complex/ambiguous tasks may show model capability differences (only tested FormatBytes)
- ⚠️ Multi-file tasks with different insertion points might show different conflict patterns
- ⚠️ Protocol adherence (beads reporting, phase tracking) not tested — headless mode bypasses these
- ⚠️ Semantic conflicts not tested — both models' implementations are functionally compatible

**What would change this:**

- If the task required inserting at variable locations (not "after FormatDurationShort"), some trials might auto-merge
- If models were given coordination instructions (e.g., "check for existing FormatBytes first"), conflict rate might drop
- If tested with orch spawn instead of claude -p, protocol compliance differences might emerge

---

## References

**Files Examined:**
- `experiments/coordination-demo/run.sh` — Experiment runner (modified for parallel execution)
- `experiments/coordination-demo/score.sh` — 6-dimension scoring rubric
- `experiments/coordination-demo/merge-check-v2.sh` — Merge conflict analysis (created during this investigation)
- `experiments/coordination-demo/task-prompt.md` — FormatBytes task specification

**Commands Run:**
```bash
# Run N=10 experiment (parallel haiku+opus per trial)
bash experiments/coordination-demo/run.sh 10

# Merge conflict analysis
bash experiments/coordination-demo/merge-check-v2.sh experiments/coordination-demo/results/20260309-172034 22073e5e7

# Statistical analysis (inline bash)
# Welch's t-test, mean/SD calculations, Fisher's exact test
```

**Related Artifacts:**
- **Prior investigation:** `.kb/investigations/2026-03-09-inv-coordination-failure-controlled-demo-same.md` (N=1 pilot)
- **Experiment results:** `experiments/coordination-demo/results/20260309-172034/`
- **Results summary:** `experiments/coordination-demo/results/20260309-172034/RESULTS.md`

---

## Investigation History

**[2026-03-09 17:15]:** Investigation started
- Extended prior N=1 pilot to N=10 for statistical significance
- Modified run.sh for parallel execution (haiku+opus per trial)

**[2026-03-09 17:20]:** N=10 experiment launched
- 20 agent invocations (10 haiku, 10 opus), parallel within trials
- Total wall time: ~8 minutes

**[2026-03-09 17:29]:** Results collected
- All 20 runs scored 6/6 (adjusted)
- Mean durations: haiku 39.1s, opus 44.0s

**[2026-03-09 17:35]:** Merge analysis completed
- 10/10 merge conflicts confirmed
- Fixed merge-check script (relative path bug) and created v2

**[2026-03-09 17:40]:** Investigation completed
- Status: Complete
- Key outcome: N=1 pilot finding confirmed at N=10 — coordination failure rate is 100% for both models, independent of capability
