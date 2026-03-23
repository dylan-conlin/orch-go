# SYNTHESIS: Modification Task Coordination Experiment

## Plain-Language Summary

We ran the same 4-condition coordination experiment (no-coord, placement, context-share, messaging) but with modification tasks instead of additive tasks. Where additive tasks produce 100% conflict rate without placement, modification tasks produce **0% conflict rate even with zero coordination** (40/40 SUCCESS across all conditions). The coordination problem that the entire model describes is specific to additive tasks where agents converge on the same insertion point. When agents modify different existing functions, they naturally produce non-overlapping diffs — the task structure itself coordinates them. This narrows Claims 1 and 4, answers the model's own open question about modification tasks, and has a practical implication: orch-go can dispatch modification tasks to parallel agents without coordination constraints.

## What Was Built

- **Experiment harness:** `run-modification.sh`, `score-modification.sh`, `analyze-modification.sh` in `experiments/coordination-demo/redesign/`
- **Modification prompts:** `prompts/modify-a.md` (refactor FormatDuration) and `prompts/modify-b.md` (refactor Truncate/TruncateWithPadding)
- **Experiment results:** `experiments/coordination-demo/redesign/results/modify-20260323-093711/` (40 trials, 80 agent invocations)
- **Probe:** `.kb/models/coordination/probes/2026-03-23-probe-modification-task-experiment.md`
- **Model updates:** Claims 1 and 4 scoped to additive tasks, open question marked answered, experiment added to evidence table

## Key Numbers

| Condition | Additive (original) | Modification (this probe) |
|-----------|:-------------------:|:-------------------------:|
| no-coord | 20/20 CONFLICT | **10/10 SUCCESS** |
| placement | 20/20 SUCCESS | **10/10 SUCCESS** |
| context-share | 20/20 CONFLICT | **10/10 SUCCESS** |
| messaging | 20/20 CONFLICT | **10/10 SUCCESS** |

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace directory.

Key verification:
1. Results exist at `experiments/coordination-demo/redesign/results/modify-20260323-093711/`
2. `analysis.md` in results shows 40/40 SUCCESS
3. Model updated with task-type boundary notes on Claims 1 and 4
4. Open question about modification tasks marked as answered
